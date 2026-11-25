package extensions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// nativeRegistry holds in-process extensions registered at startup.
var (
	nativeMu       sync.Mutex
	nativeRegistry = map[string]Extension{}
)

// RegisterNative adds a native extension.
func RegisterNative(e Extension) {
	nativeMu.Lock()
	defer nativeMu.Unlock()
	nativeRegistry[e.Name()] = e
}

// Native returns a registered native extension by name, if any.
func Native(name string) (Extension, bool) {
	nativeMu.Lock()
	defer nativeMu.Unlock()
	e, ok := nativeRegistry[name]
	return e, ok
}

// AllNative returns all registered native extensions.
func AllNative() []Extension {
	nativeMu.Lock()
	defer nativeMu.Unlock()
	out := make([]Extension, 0, len(nativeRegistry))
	for _, e := range nativeRegistry {
		out = append(out, e)
	}
	return out
}

// HealthCheck pings a remote extension's /cockpit/health endpoint.
func HealthCheck(base string) error {
	if strings.TrimSpace(base) == "" {
		return fmt.Errorf("no base url")
	}
	resp, err := http.Get(strings.TrimRight(base, "/") + "/cockpit/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health status %d", resp.StatusCode)
	}
	return nil
}

// FetchManifest fetches a remote extension's manifest.
func FetchManifest(base string) (Manifest, error) {
	var m Manifest
	resp, err := http.Get(strings.TrimRight(base, "/") + "/cockpit/manifest")
	if err != nil {
		return m, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return m, fmt.Errorf("manifest status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return m, err
	}
	if strings.TrimSpace(m.Name) == "" {
		return m, fmt.Errorf("manifest missing name")
	}
	return m, nil
}
