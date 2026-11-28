package storage

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database.
type Store struct {
	db *sql.DB
}

// Open opens (and migrates) the store at path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			agent TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'running',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS agents (
			name TEXT PRIMARY KEY,
			path TEXT NOT NULL,
			detected_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS tools (
			name TEXT PRIMARY KEY,
			binary TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT,
			type TEXT NOT NULL,
			payload TEXT,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS extensions (
			name TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			version TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			available INTEGER NOT NULL DEFAULT 0,
			last_sync TEXT,
			last_seen TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS t3_state (
			machine_id TEXT PRIMARY KEY,
			port INTEGER NOT NULL DEFAULT 3773,
			pairing_token TEXT NOT NULL DEFAULT '',
			service_status TEXT NOT NULL DEFAULT '',
			last_check TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS machine_checks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			machine_id TEXT NOT NULL,
			result TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }

// Session record.
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Agent     string    `json:"agent"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// CreateSession inserts a session.
func (s *Store) CreateSession(id, name, agent string) error {
	t := now()
	_, err := s.db.Exec(`INSERT INTO sessions (id,name,agent,status,created_at,updated_at) VALUES (?,?,?,?,?,?)`,
		id, name, agent, "running", t, t)
	return err
}

// UpdateSessionStatus updates status by id.
func (s *Store) UpdateSessionStatus(id, status string) error {
	_, err := s.db.Exec(`UPDATE sessions SET status=?, updated_at=? WHERE id=?`, status, now(), id)
	return err
}

// UpdateSessionStatusByName updates status by session name.
func (s *Store) UpdateSessionStatusByName(name, status string) error {
	_, err := s.db.Exec(`UPDATE sessions SET status=?, updated_at=? WHERE name=?`, status, now(), name)
	return err
}

// ListSessions returns all sessions.
func (s *Store) ListSessions() ([]Session, error) {
	rows, err := s.db.Query(`SELECT id,name,agent,status,created_at,updated_at FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var se Session
		var c, u string
		if err := rows.Scan(&se.ID, &se.Name, &se.Agent, &se.Status, &c, &u); err != nil {
			return nil, err
		}
		se.CreatedAt, _ = time.Parse(time.RFC3339, c)
		se.UpdatedAt, _ = time.Parse(time.RFC3339, u)
		out = append(out, se)
	}
	return out, rows.Err()
}

// UpsertAgent records a detected agent.
func (s *Store) UpsertAgent(name, path string) error {
	_, err := s.db.Exec(`INSERT INTO agents (name,path,detected_at) VALUES (?,?,?) ON CONFLICT(name) DO UPDATE SET path=excluded.path, detected_at=excluded.detected_at`,
		name, path, now())
	return err
}

// AgentRecord is a stored agent.
type AgentRecord struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListAgents returns recorded agents.
func (s *Store) ListAgents() ([]AgentRecord, error) {
	rows, err := s.db.Query(`SELECT name,path FROM agents ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AgentRecord
	for rows.Next() {
		var a AgentRecord
		if err := rows.Scan(&a.Name, &a.Path); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// LogEvent appends an event row to SQLite and a JSONL file.
func (s *Store) LogEvent(sessionID, typ string, payload any) error {
	b, _ := json.Marshal(payload)
	_, err := s.db.Exec(`INSERT INTO events (session_id,type,payload,created_at) VALUES (?,?,?,?)`,
		sessionID, typ, string(b), now())
	return err
}

// AppendJSONL appends a JSON line to path.
func AppendJSONL(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = f.Write(b)
	return err
}

// Extension record.
type Extension struct {
	Name      string     `json:"name"`
	Kind      string     `json:"kind"`
	Version   string     `json:"version"`
	Enabled   bool       `json:"enabled"`
	Available bool       `json:"available"`
	LastSync  *time.Time `json:"last_sync,omitempty"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
}

// UpsertExtension inserts or updates an extension row.
func (s *Store) UpsertExtension(name, kind, version string, enabled bool) error {
	_, err := s.db.Exec(`INSERT INTO extensions (name,kind,version,enabled,available) VALUES (?,?,?,?,0)
		ON CONFLICT(name) DO UPDATE SET kind=excluded.kind, version=excluded.version, enabled=excluded.enabled`,
		name, kind, version, boolToInt(enabled))
	return err
}

// SetExtensionEnabled toggles enabled.
func (s *Store) SetExtensionEnabled(name string, enabled bool) error {
	_, err := s.db.Exec(`UPDATE extensions SET enabled=? WHERE name=?`, boolToInt(enabled), name)
	return err
}

// SetExtensionAvailable updates availability + last_seen.
func (s *Store) SetExtensionAvailable(name string, available bool) error {
	t := now()
	_, err := s.db.Exec(`UPDATE extensions SET available=?, last_seen=? WHERE name=?`, boolToInt(available), t, name)
	return err
}

// SetExtensionLastSync updates last_sync.
func (s *Store) SetExtensionLastSync(name string) error {
	t := now()
	_, err := s.db.Exec(`UPDATE extensions SET last_sync=? WHERE name=?`, t, name)
	return err
}

// ListExtensions returns all extension rows.
func (s *Store) ListExtensions() ([]Extension, error) {
	rows, err := s.db.Query(`SELECT name,kind,version,enabled,available,last_sync,last_seen FROM extensions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Extension
	for rows.Next() {
		var e Extension
		var en, av int
		var ls, lseen sql.NullString
		if err := rows.Scan(&e.Name, &e.Kind, &e.Version, &en, &av, &ls, &lseen); err != nil {
			return nil, err
		}
		e.Enabled = en != 0
		e.Available = av != 0
		if ls.Valid {
			t, _ := time.Parse(time.RFC3339, ls.String)
			e.LastSync = &t
		}
		if lseen.Valid {
			t, _ := time.Parse(time.RFC3339, lseen.String)
			e.LastSeen = &t
		}
		out = append(out, e)
	}
	return out, nil
}

// GetExtension returns a single extension row.
func (s *Store) GetExtension(name string) (Extension, bool, error) {
	rows, err := s.db.Query(`SELECT name,kind,version,enabled,available,last_sync,last_seen FROM extensions WHERE name=?`, name)
	if err != nil {
		return Extension{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Extension{}, false, nil
	}
	var e Extension
	var en, av int
	var ls, lseen sql.NullString
	if err := rows.Scan(&e.Name, &e.Kind, &e.Version, &en, &av, &ls, &lseen); err != nil {
		return Extension{}, false, err
	}
	e.Enabled = en != 0
	e.Available = av != 0
	if ls.Valid {
		t, _ := time.Parse(time.RFC3339, ls.String)
		e.LastSync = &t
	}
	if lseen.Valid {
		t, _ := time.Parse(time.RFC3339, lseen.String)
		e.LastSeen = &t
	}
	return e, true, nil
}

// RemoveExtension deletes an extension row.
func (s *Store) RemoveExtension(name string) error {
	_, err := s.db.Exec(`DELETE FROM extensions WHERE name=?`, name)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// T3State records T3 Code state for a machine (keyed by machines ID).
type T3State struct {
	MachineID     string    `json:"machine_id"`
	Port          int       `json:"port"`
	PairingToken  string    `json:"pairing_token,omitempty"`
	ServiceStatus string    `json:"service_status"`
	LastCheck     string    `json:"last_check,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UpsertT3State inserts or updates T3 state for a machine.
func (s *Store) UpsertT3State(st T3State) error {
	if st.Port == 0 {
		st.Port = 3773
	}
	t := now()
	_, err := s.db.Exec(`INSERT INTO t3_state (machine_id,port,pairing_token,service_status,last_check,updated_at) VALUES (?,?,?,?,?,?)
		ON CONFLICT(machine_id) DO UPDATE SET port=excluded.port, pairing_token=excluded.pairing_token, service_status=excluded.service_status, last_check=excluded.last_check, updated_at=excluded.updated_at`,
		st.MachineID, st.Port, st.PairingToken, st.ServiceStatus, st.LastCheck, t)
	return err
}

// GetT3State returns T3 state for a machine.
func (s *Store) GetT3State(machineID string) (T3State, bool, error) {
	row := s.db.QueryRow(`SELECT machine_id,port,pairing_token,service_status,last_check,updated_at FROM t3_state WHERE machine_id=?`, machineID)
	var st T3State
	var u string
	if err := row.Scan(&st.MachineID, &st.Port, &st.PairingToken, &st.ServiceStatus, &st.LastCheck, &u); err != nil {
		if err == sql.ErrNoRows {
			return T3State{}, false, nil
		}
		return T3State{}, false, err
	}
	st.UpdatedAt, _ = time.Parse(time.RFC3339, u)
	return st, true, nil
}

// ListT3States returns all T3 states.
func (s *Store) ListT3States() ([]T3State, error) {
	rows, err := s.db.Query(`SELECT machine_id,port,pairing_token,service_status,last_check,updated_at FROM t3_state ORDER BY machine_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T3State
	for rows.Next() {
		var st T3State
		var u string
		if err := rows.Scan(&st.MachineID, &st.Port, &st.PairingToken, &st.ServiceStatus, &st.LastCheck, &u); err != nil {
			return nil, err
		}
		st.UpdatedAt, _ = time.Parse(time.RFC3339, u)
		out = append(out, st)
	}
	return out, rows.Err()
}

// AddMachineCheck appends a check result (JSON) for a machine.
func (s *Store) AddMachineCheck(machineID, resultJSON string) error {
	_, err := s.db.Exec(`INSERT INTO machine_checks (machine_id,result,created_at) VALUES (?,?,?)`, machineID, resultJSON, now())
	return err
}

// ListMachineChecks returns recent checks for a machine.
func (s *Store) ListMachineChecks(machineID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`SELECT result FROM machine_checks WHERE machine_id=? ORDER BY id DESC LIMIT ?`, machineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
