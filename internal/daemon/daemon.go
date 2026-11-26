package daemon

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/farishan-labs/cockpit/internal/adapters"
	"github.com/farishan-labs/cockpit/internal/agents"
	"github.com/farishan-labs/cockpit/internal/machines"
	"github.com/farishan-labs/cockpit/internal/config"
	"github.com/farishan-labs/cockpit/internal/extensions"
	"github.com/farishan-labs/cockpit/internal/storage"
	"github.com/farishan-labs/cockpit/internal/terminal"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

//go:embed web
var webFS embed.FS

// Daemon is the cockpit core.
type Daemon struct {
	Cfg         config.Config
	Store       *storage.Store
	Backend     terminal.Backend
	adapters    []adapters.Adapter
	agents      []agents.Agent
	extensions  *extensions.Manager
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]bool
	broadcast   chan []byte
}

// New creates a daemon.
func New(cfg config.Config, store *storage.Store, backend terminal.Backend) *Daemon {
	d := &Daemon{
		Cfg:         cfg,
		Store:       store,
		Backend:     backend,
		adapters:    adapters.All(),
		agents:      agents.Detect(),
		extensions:  extensions.NewManager(store),
		upgrader:    websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		clients:     make(map[*websocket.Conn]bool),
		broadcast:   make(chan []byte, 64),
	}
	// record agents
	for _, a := range d.agents {
		_ = store.UpsertAgent(a.Name, a.Path)
	}
	// discover extensions
	if err := d.extensions.Discover(); err != nil {
		fmt.Fprintf(os.Stderr, "extensions: discover: %v\n", err)
	}
	return d
}

// Run starts the daemon and blocks until signal.
func (d *Daemon) Run() error {
	// PID file
	pidPath := filepath.Join(config.Dir(), "cockpit.pid")
	_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644)
	defer os.Remove(pidPath)

	mux := http.NewServeMux()
	d.routes(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", d.Cfg.Port),
		Handler: mux,
	}

	// websocket broadcaster
	go d.runBroadcaster()

	// watch terminal sessions for changes and push updates
	go d.watchSessions()

	// open browser
	if d.Cfg.OpenBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(fmt.Sprintf("http://127.0.0.1:%d", d.Cfg.Port))
		}()
	}

	fmt.Printf("cockpit serve: listening on http://127.0.0.1:%d\n", d.Cfg.Port)

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (d *Daemon) runBroadcaster() {
	for msg := range d.broadcast {
		for c := range d.clients {
			_ = c.WriteMessage(websocket.TextMessage, msg)
		}
	}
}

func (d *Daemon) notify() {
	b, _ := json.Marshal(map[string]any{"type": "update", "ts": time.Now()})
	d.broadcast <- b
}

func (d *Daemon) watchSessions() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var last string
	for range ticker.C {
		ts, _ := d.Backend.ListSessions()
		b, _ := json.Marshal(ts)
		cur := string(b)
		if cur != last {
			last = cur
			d.notify()
		}
	}
}

func (d *Daemon) routes(mux *http.ServeMux) {
	// web UI
	webSub, _ := fs.Sub(webFS, "web")
	mux.Handle("/", http.FileServer(http.FS(webSub)))
	// Actually we want index template; file server is fine for static.

	mux.HandleFunc("/api/sessions", d.handleSessions)
	mux.HandleFunc("/api/agents", d.handleAgents)
	mux.HandleFunc("/api/launch", d.handleLaunch)
	mux.HandleFunc("/api/kill", d.handleKill)
	mux.HandleFunc("/api/doctor", d.handleDoctor)
	mux.HandleFunc("/api/adapters", d.handleAdapters)
	mux.HandleFunc("/api/fleet", d.handleFleet)
	mux.HandleFunc("/api/machines", d.handleMachines)
	mux.HandleFunc("/api/machines/check", d.handleMachinesCheck)
	mux.HandleFunc("/api/machines/pair", d.handleMachinesPair)
	mux.HandleFunc("/api/extensions", d.handleExtensions)
	mux.HandleFunc("/operator", d.handleOperator)
	mux.HandleFunc("/api/operator", d.handleOperator)
	mux.HandleFunc("/ws", d.handleWS)

	// native extensions
	d.extensions.MountNative(mux)

	// remote extension proxy: /ext/<name>/*
	mux.HandleFunc("/ext/", func(w http.ResponseWriter, r *http.Request) {
		// path: /ext/<name>/...
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/ext/"), "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		name := parts[0]
		// native extensions already mounted their own handlers; fall through to proxy for remote
		if _, ok := extensions.Native(name); ok {
			http.NotFound(w, r)
			return
		}
		d.extensions.ProxyHandler(name).ServeHTTP(w, r)
	})

	d.extensions.StartHealthLoop()
}

func (d *Daemon) handleSessions(w http.ResponseWriter, r *http.Request) {
	// merge tmux sessions with stored
	ts, _ := d.Backend.ListSessions()
	stored, _ := d.Store.ListSessions()
	// map stored by name
	byName := map[string]storage.Session{}
	for _, s := range stored {
		byName[s.Name] = s
	}
	type row struct {
		Name     string `json:"name"`
		Agent    string `json:"agent"`
		Status   string `json:"status"`
		Windows  int    `json:"windows"`
		Attached bool   `json:"attached"`
		Live     bool   `json:"live"`
	}
	var rows []row = []row{}
	liveNames := map[string]bool{}
	for _, t := range ts {
		liveNames[t.Name] = true
		agent := ""
		status := "running"
		if st, ok := byName[t.Name]; ok {
			agent = st.Agent
			status = st.Status
		}
		rows = append(rows, row{Name: t.Name, Agent: agent, Status: status, Windows: t.Windows, Attached: t.Attached, Live: true})
	}
	// stored but not live
	for _, s := range stored {
		if !liveNames[s.Name] {
			rows = append(rows, row{Name: s.Name, Agent: s.Agent, Status: "exited", Live: false})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

func (d *Daemon) handleAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d.agents)
}

type launchReq struct {
	Agent string `json:"agent"`
	Name  string `json:"name"`
}

func (d *Daemon) handleLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req launchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// try form
		req.Agent = r.FormValue("agent")
		req.Name = r.FormValue("name")
	}
	if req.Agent == "" {
		req.Agent = d.Cfg.DefaultAgent
	}
	if req.Name == "" {
		req.Name = req.Agent + "-" + uuid.New().String()[:8]
	}
	// ensure USER.md symlink in session cwd
	cwd, _ := os.Getwd()
	if cwd != "" {
		link := filepath.Join(cwd, "USER.md")
		if _, err := os.Lstat(link); err != nil {
			_ = os.Symlink(d.Cfg.UserMDPath, link)
			_ = storage.AppendJSONL(d.Cfg.EventLogPath, map[string]any{
				"type": "user_md_injected", "session": req.Name, "path": d.Cfg.UserMDPath, "ts": time.Now().UTC(),
			})
		}
	}
	// launch tmux session running agent with USER_MD_PATH exported
	cmd := fmt.Sprintf("sh -c 'cd %s 2>/dev/null; export USER_MD_PATH=%s; exec %s'",
		shellQuote(cwd), shellQuote(d.Cfg.UserMDPath), req.Agent)
	if err := d.Backend.CreateSession(req.Name, cmd); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	id := uuid.New().String()
	_ = d.Store.CreateSession(id, req.Name, req.Agent)
	_ = d.Store.LogEvent(id, "launch", map[string]any{"agent": req.Agent, "session": req.Name, "user_md_path": d.Cfg.UserMDPath})
	_ = storage.AppendJSONL(d.Cfg.EventLogPath, map[string]any{
		"type": "launch", "session": req.Name, "agent": req.Agent, "ts": time.Now().UTC(),
	})
	d.notify()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id, "name": req.Name, "agent": req.Agent})
}

func (d *Daemon) handleKill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" {
		req.Name = r.FormValue("name")
	}
	if err := d.Backend.KillSession(req.Name); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = d.Store.UpdateSessionStatusByName(req.Name, "killed")
	d.notify()
	w.WriteHeader(204)
}

func (d *Daemon) handleDoctor(w http.ResponseWriter, r *http.Request) {
	res := map[string]any{
		"tmux":   d.Backend.Available(),
		"agents": d.agents,
		"port":   d.Cfg.Port,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (d *Daemon) handleOperator(w http.ResponseWriter, r *http.Request) {
	profile, _ := os.ReadFile(d.Cfg.UserMDPath)
	resp := map[string]any{
		"path":    d.Cfg.UserMDPath,
		"profile": string(profile),
	}
	if fi, err := os.Stat(d.Cfg.UserMDPath); err == nil {
		resp["last_audit"] = fi.ModTime().UTC().Format(time.RFC3339)
	}
	// validation via usermd-check --json
	if bin, err := exec.LookPath("usermd-check"); err == nil {
		cmd := exec.Command(bin, "--json", d.Cfg.UserMDPath)
		out, _ := cmd.Output()
		var v map[string]any
		if json.Unmarshal(out, &v) == nil {
			resp["validation"] = v
		} else {
			resp["validation"] = nil
		}
	} else {
		resp["validation"] = nil
		resp["validation_skipped"] = true
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}


func (d *Daemon) handleMachines(w http.ResponseWriter, r *http.Request) {
	ms, _, err := machines.ListMachines()
	if err != nil {
		// return empty list
		ms = []machines.Machine{}
	}
	// merge t3 state
	type row struct {
		machines.Machine
		T3 *storage.T3State `json:"t3_state,omitempty"`
	}
	rows := make([]row, 0, len(ms))
	for _, m := range ms {
		rw := row{Machine: m}
		if st, ok, _ := d.Store.GetT3State(m.ID); ok {
			s := st
			rw.T3 = &s
		}
		rows = append(rows, rw)
	}
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(500)
	}
	json.NewEncoder(w).Encode(map[string]any{"machines": rows, "error": errString(err)})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (d *Daemon) handleMachinesCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		name = r.FormValue("name")
	}
	ms, _, err := machines.ListMachines()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var results []machines.CheckResult
	for _, m := range ms {
		if name != "" && m.ID != name && m.Name != name {
			continue
		}
		// determine ssh user/port from t3 state (port) - default
		port := 3773
		sshUser := ""
		if st, ok, _ := d.Store.GetT3State(m.ID); ok && st.Port > 0 {
			port = st.Port
		}
		// ssh user not in machines product; leave empty (ssh uses default)
		res := machines.CheckMachine(d.Store, m, sshUser, port)
		results = append(results, res)
		// jsonl event
		b, _ := json.Marshal(res)
		_ = storage.AppendJSONL(d.Cfg.EventLogPath, map[string]any{"type": "machine_check", "machine": m.ID, "result": string(b), "ts": time.Now().UTC()})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (d *Daemon) handleMachinesPair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	machine := r.FormValue("machine")
	ttl := r.FormValue("ttl")
	tailscaleFlag := r.FormValue("tailscale") == "true" || r.FormValue("tailscale") == "1"
	var req struct {
		Machine   string `json:"machine"`
		TTL       string `json:"ttl"`
		Tailscale bool   `json:"tailscale"`
	}
	if machine == "" {
		json.NewDecoder(r.Body).Decode(&req)
		machine = req.Machine
		if ttl == "" { ttl = req.TTL }
		tailscaleFlag = tailscaleFlag || req.Tailscale
	}
	if ttl == "" { ttl = "24h" }
	ms, _, err := machines.ListMachines()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var target *machines.Machine
	for i := range ms {
		if ms[i].ID == machine || ms[i].Name == machine {
			target = &ms[i]
			break
		}
	}
	if target == nil {
		http.Error(w, "machine not found", 404)
		return
	}
	port := 3773
	if st, ok, _ := d.Store.GetT3State(target.ID); ok && st.Port > 0 {
		port = st.Port
	}
	out, err := machines.PairMachine("", target.Host, tailscaleFlag, ttl)
	if err != nil {
		http.Error(w, err.Error()+"\n"+out, 500)
		return
	}
	st := storage.T3State{MachineID: target.ID, Port: port, PairingToken: out, ServiceStatus: "paired"}
	_ = d.Store.UpsertT3State(st)
	_ = storage.AppendJSONL(d.Cfg.EventLogPath, map[string]any{"type": "machine_pair", "machine": target.ID, "ts": time.Now().UTC()})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"machine": target.ID, "result": out})
}



func (d *Daemon) handleFleet(w http.ResponseWriter, r *http.Request) {
	type proc struct {
		Name      string `json:"name"`
		Command   string `json:"command"`
		Type      string `json:"type"`
		HealthURL string `json:"health_url"`
	}
	path := os.Getenv("PROCESSOR_REGISTRY")
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".config", "fhl", "processor.json")
	}
	data, _ := os.ReadFile(path)
	var reg struct {
		Processes []proc `json:"processes"`
	}
	_ = json.Unmarshal(data, &reg)

	type row struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Command   string `json:"command"`
		Health    string `json:"health"`
		HealthURL string `json:"health_url,omitempty"`
		LatencyMs int    `json:"latency_ms,omitempty"`
	}
	rows := []row{}
	client := &http.Client{Timeout: 3 * time.Second}
	for _, p := range reg.Processes {
		t := p.Type
		if t == "" {
			t = "daemon"
		}
		health := "n/a"
		latency := 0
		if p.HealthURL != "" {
			start := time.Now()
			resp, err := client.Get(p.HealthURL)
			if err != nil {
				health = "down"
			} else {
				resp.Body.Close()
				latency = int(time.Since(start).Milliseconds())
				if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					health = "ok"
				} else {
					health = fmt.Sprintf("http %d", resp.StatusCode)
				}
			}
		}
		rows = append(rows, row{Name: p.Name, Type: t, Command: p.Command, Health: health, HealthURL: p.HealthURL, LatencyMs: latency})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

func (d *Daemon) handleAdapters(w http.ResponseWriter, r *http.Request) {
	type row struct {
		Name   string         `json:"name"`
		Avail  bool           `json:"available"`
		Status map[string]any `json:"status"`
		Error  string         `json:"error,omitempty"`
	}
	var rows []row
	for _, a := range d.adapters {
		st, err := a.Status()
		avail := a.Available()
		rr := row{Name: a.Name(), Avail: avail, Status: st}
		if err != nil {
			rr.Error = err.Error()
		}
		rows = append(rows, rr)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

func (d *Daemon) handleExtensions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d.extensions.All())
}


func (d *Daemon) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := d.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	d.clients[conn] = true
	// send initial
	b, _ := json.Marshal(map[string]any{"type": "hello"})
	conn.WriteMessage(websocket.TextMessage, b)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			delete(d.clients, conn)
			conn.Close()
			break
		}
	}
}

// need UpdateSessionStatusByName in storage; add helper below via extension? We'll add to store.
func openBrowser(url string) {
	// best-effort
	for _, bin := range []string{"xdg-open", "open"} {
		if p, err := exec.LookPath(bin); err == nil {
			cmd := exec.Command(p, url)
			_ = cmd.Start()
			return
		}
	}
}


func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
