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

	"github.com/farishan-labs/cockpit/internal/agents"
	"github.com/farishan-labs/cockpit/internal/config"
	"github.com/farishan-labs/cockpit/internal/storage"
	"github.com/farishan-labs/cockpit/internal/terminal"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

//go:embed web
var webFS embed.FS

// Daemon is the cockpit core.
type Daemon struct {
	Cfg       config.Config
	Store     *storage.Store
	Backend   terminal.Backend
	agents    []agents.Agent
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]bool
	broadcast chan []byte
}

// New creates a daemon.
func New(cfg config.Config, store *storage.Store, backend terminal.Backend) *Daemon {
	d := &Daemon{
		Cfg:       cfg,
		Store:     store,
		Backend:   backend,
		agents:    agents.Detect(),
		upgrader:  websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan []byte, 64),
	}
	// record agents
	for _, a := range d.agents {
		_ = store.UpsertAgent(a.Name, a.Path)
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
	mux.HandleFunc("/operator", d.handleOperator)
	mux.HandleFunc("/api/operator", d.handleOperator)
	mux.HandleFunc("/ws", d.handleWS)
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
