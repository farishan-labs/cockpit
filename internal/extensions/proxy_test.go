package extensions

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/farishan-labs/cockpit/internal/storage"
)

func TestProxyRoundTrip(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from upstream"))
	}))
	defer upstream.Close()

	tmp := t.TempDir()
	extDir := filepath.Join(tmp, "foo")
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"foo","kind":"remote","web":"` + upstream.URL + `"}`
	if err := os.WriteFile(filepath.Join(extDir, "cockpit-extension.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COCKPIT_EXTENSIONS", tmp)

	store, err := storage.Open(filepath.Join(tmp, "cockpit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	mgr := NewManager(store)
	if err := mgr.Discover(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ext/foo/api", nil)
	rr := httptest.NewRecorder()
	mgr.ProxyHandler("foo").ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if body := rr.Body.String(); body != "hello from upstream" {
		t.Fatalf("body: got %q", body)
	}
}
