package extensions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/farishan-labs/cockpit/internal/storage"
)

// TabInfo is a tab as exposed to the UI, with its proxied URL.
type TabInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	Route string `json:"route"`
	URL   string `json:"url"`
}

// Info is the extension summary exposed via /api/extensions.
type Info struct {
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Version   string    `json:"version"`
	Enabled   bool      `json:"enabled"`
	Available bool      `json:"available"`
	Tabs      []TabInfo `json:"tabs"`
	Commands  []Command `json:"commands"`
	Provides  []string  `json:"provides,omitempty"`
	APIBase   string    `json:"api_base,omitempty"`
	Web       string    `json:"web,omitempty"`
	Dir       string    `json:"dir,omitempty"`
}

// Manager discovers and tracks extensions.
type Manager struct {
	store    *storage.Store
	discovered []Discovered
	native   map[string]Extension
}

// NewManager creates a manager.
func NewManager(store *storage.Store) *Manager {
	m := &Manager{store: store, native: map[string]Extension{}}
	for _, e := range AllNative() {
		m.native[e.Name()] = e
	}
	return m
}

// Discover loads manifests and syncs to storage.
func (m *Manager) Discover() error {
	list, err := Discover()
	if err != nil {
		return err
	}
	m.discovered = list
	for _, d := range list {
		kind := string(d.Manifest.EffectiveKind())
		if _, ok := m.native[d.Manifest.Name]; ok {
			kind = string(KindNative)
		}
		enabled := d.Manifest.IsEnabled()
		if row, ok, _ := m.store.GetExtension(d.Manifest.Name); ok {
			enabled = row.Enabled
		}
		if err := m.store.UpsertExtension(d.Manifest.Name, kind, d.Manifest.Version, enabled); err != nil {
			fmt.Fprintf(os.Stderr, "extensions: upsert %s: %v\n", d.Manifest.Name, err)
		}
	}
	return nil
}

// All returns current Info list, refreshing availability for remote extensions.
func (m *Manager) All() []Info {
	rows, _ := m.store.ListExtensions()
	rowByName := map[string]storage.Extension{}
	for _, r := range rows {
		rowByName[r.Name] = r
	}

	// discovered by name
	discByName := map[string]Discovered{}
	for _, d := range m.discovered {
		discByName[d.Manifest.Name] = d
	}

	var out []Info
	for name, d := range discByName {
		inf := m.buildInfo(d, rowByName[name])
		out = append(out, inf)
	}
	// native-only extensions (registered but not discovered via manifest) — include them
	for name, e := range m.native {
		if _, ok := discByName[name]; ok {
			continue
		}
		inf := Info{
			Name:      e.Name(),
			Kind:      string(KindNative),
			Version:   e.Version(),
			Enabled:   true,
			Available: true,
			Tabs:      toTabInfo(e.Tabs(), name),
			Commands:  e.Commands(),
		}
		if row, ok := rowByName[name]; ok {
			inf.Enabled = row.Enabled
			inf.Available = row.Available
		}
		out = append(out, inf)
	}
	return out
}

func (m *Manager) buildInfo(d Discovered, row storage.Extension) Info {
	inf := Info{
		Name:     d.Manifest.Name,
		Kind:     string(d.Manifest.EffectiveKind()),
		Version:  d.Manifest.Version,
		Enabled:  d.Manifest.IsEnabled(),
		Provides: d.Manifest.Provides,
		APIBase:  d.Manifest.APIBase,
		Web:      d.Manifest.Web,
		Dir:      d.Dir,
		Tabs:     toTabInfo(d.Manifest.Tabs, d.Manifest.Name),
		Commands: d.Manifest.Commands,
	}
	if ne, ok := m.native[d.Manifest.Name]; ok {
		inf.Kind = string(KindNative)
		inf.Tabs = toTabInfo(ne.Tabs(), d.Manifest.Name)
		inf.Commands = ne.Commands()
		inf.Available = true
	}
	if row.Name != "" {
		inf.Enabled = row.Enabled
		inf.Available = row.Available
	}
	// remote availability check
	if inf.Kind == string(KindRemote) && inf.Enabled {
		if err := HealthCheck(firstNonEmpty(d.Manifest.APIBase, d.Manifest.Web)); err != nil {
			inf.Available = false
		} else {
			inf.Available = true
			_ = m.store.SetExtensionAvailable(d.Manifest.Name, true)
		}
	}
	// sync storage enabled flag
	_ = m.store.UpsertExtension(d.Manifest.Name, inf.Kind, inf.Version, inf.Enabled)
	return inf
}

func toTabInfo(tabs []Tab, name string) []TabInfo {
	out := make([]TabInfo, 0, len(tabs))
	for _, t := range tabs {
		route := t.Route
		if !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		out = append(out, TabInfo{
			ID:    t.ID,
			Title: t.Title,
			Icon:  t.Icon,
			Route: route,
			URL:   "/ext/" + name + route,
		})
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// Enabled returns whether an extension is enabled (storage overrides manifest).
func (m *Manager) Enabled(name string) bool {
	if row, ok, _ := m.store.GetExtension(name); ok {
		return row.Enabled
	}
	for _, d := range m.discovered {
		if d.Manifest.Name == name {
			return d.Manifest.IsEnabled()
		}
	}
	return false
}

// SetEnabled toggles and persists.
func (m *Manager) SetEnabled(name string, enabled bool) error {
	if err := m.store.SetExtensionEnabled(name, enabled); err != nil {
		return err
	}
	// if no row, upsert
	if _, ok, _ := m.store.GetExtension(name); !ok {
		_ = m.store.UpsertExtension(name, string(KindRemote), "", enabled)
	}
	return nil
}

// Remove removes an extension from storage (does not delete files).
func (m *Manager) Remove(name string) error {
	return m.store.RemoveExtension(name)
}

// ProxyHandler returns an http.Handler reverse-proxying to a remote extension.
// Paths under /ext/<name>/api/* go to api_base; everything else to web base.
func (m *Manager) ProxyHandler(name string) http.Handler {
	var apiBase, webBase string
	for _, d := range m.discovered {
		if d.Manifest.Name == name {
			apiBase = d.Manifest.APIBase
			webBase = d.Manifest.Web
		}
	}
	if apiBase == "" {
		apiBase = webBase
	}
	webTarget, _ := url.Parse(webBase)
	apiTarget, _ := url.Parse(apiBase)
	if webTarget == nil || webTarget.Scheme == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "extension not available", http.StatusBadGateway)
		})
	}
	webProxy := httputil.NewSingleHostReverseProxy(webTarget)
	apiProxy := httputil.NewSingleHostReverseProxy(apiTarget)
	// director fixups
	fix := func(proxy *httputil.ReverseProxy, target *url.URL, strip string) {
		orig := proxy.Director
		proxy.Director = func(req *http.Request) {
			orig(req)
			req.URL.Path = strings.TrimPrefix(req.URL.Path, strip)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req.URL.RawPath = ""
			req.Host = target.Host
		}
	}
	fix(webProxy, webTarget, "/ext/"+name)
	fix(apiProxy, apiTarget, "/ext/"+name+"/api")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/ext/"+name+"/api") {
			apiProxy.ServeHTTP(w, r)
			return
		}
		webProxy.ServeHTTP(w, r)
	})
}

// Sync calls POST /cockpit/sync on a remote extension (best-effort).
func (m *Manager) Sync(name string) error {
	var base string
	for _, d := range m.discovered {
		if d.Manifest.Name == name {
			base = firstNonEmpty(d.Manifest.APIBase, d.Manifest.Web)
		}
	}
	if base == "" {
		// native: call in-process if registered
		if ne, ok := m.native[name]; ok {
			_ = ne
			_ = m.store.SetExtensionLastSync(name)
			return nil
		}
		return fmt.Errorf("extension %s not found", name)
	}
	req, err := http.NewRequest("POST", strings.TrimRight(base, "/")+"/cockpit/sync", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sync status %d", resp.StatusCode)
	}
	return m.store.SetExtensionLastSync(name)
}

// MountNative mounts native extensions onto mux.
func (m *Manager) MountNative(mux *http.ServeMux) {
	for name, e := range m.native {
		prefix := "/ext/" + name
		if err := e.Mount(mux, prefix); err != nil {
			fmt.Fprintf(os.Stderr, "extensions: mount %s: %v\n", name, err)
		}
	}
}

// periodic health refresh
func (m *Manager) StartHealthLoop() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for _, d := range m.discovered {
				if d.Manifest.EffectiveKind() != KindRemote {
					continue
				}
				base := firstNonEmpty(d.Manifest.APIBase, d.Manifest.Web)
				avail := HealthCheck(base) == nil
				_ = m.store.SetExtensionAvailable(d.Manifest.Name, avail)
			}
		}
	}()
}

// json helpers
func jsonEncode(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// unused guard
var _ = filepath.Join
var _ = os.Args
var _ = time.Now