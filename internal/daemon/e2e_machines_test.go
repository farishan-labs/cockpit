package daemon

import (
	"net/http"
	"net/http/httptest"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/farishan-labs/cockpit/internal/storage"
	"github.com/farishan-labs/cockpit/internal/config"
	"github.com/farishan-labs/cockpit/internal/terminal"
)

type fakeBackend struct{}

func (fakeBackend) Name() string                       { return "fake" }
func (fakeBackend) Available() bool                    { return true }
func (fakeBackend) ListSessions() ([]terminal.Session, error) {
	return nil, nil
}
func (fakeBackend) CreateSession(name, command string) error { return nil }
func (fakeBackend) SendKeys(session, keys string) error      { return nil }
func (fakeBackend) CapturePane(session string) (string, error) { return "", nil }
func (fakeBackend) KillSession(name string) error            { return nil }

func TestMachinesPanelE2E(t *testing.T) {
	dir, err := os.MkdirTemp("", "cockpit-e2e")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	store, err := storage.Open(filepath.Join(dir, "cockpit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	cfg := config.Config{
		Port:            0,
		DBPath:          filepath.Join(dir, "cockpit.db"),
		EventLogPath:    filepath.Join(dir, "events.jsonl"),
		OpenBrowser:     false,
		UserMDPath:      filepath.Join(dir, "USER.md"),
	}

	d := New(cfg, store, fakeBackend{})
	mux := http.NewServeMux()
	d.routes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// UI loads and contains machines panel loading/error affordances.
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	body := string(b)
	for _, want := range []string{"Machines", "Re-check", "Pair", "Checking…", "Pairing…", "machine-status", "Failed to load machines"} {
		if !contains(body, want) {
			t.Fatalf("missing %q in UI", want)
		}
	}

	// machines API responds (possibly with error due to missing inventory) as JSON.
	r2, err := http.Post(srv.URL+"/api/machines/check?name=nonexistent", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Body.Close()
	if r2.StatusCode != http.StatusOK && r2.StatusCode != http.StatusInternalServerError {
		t.Fatalf("unexpected status %d", r2.StatusCode)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
