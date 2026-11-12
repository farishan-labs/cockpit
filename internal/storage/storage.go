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
