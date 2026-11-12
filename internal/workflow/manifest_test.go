package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Use the installed manifest if present; otherwise skip.
	p := Path()
	if _, err := os.Stat(p); err != nil {
		t.Skipf("manifest not present: %v", err)
	}
	m, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Meta.Version == "" {
		t.Fatal("empty version")
	}
	if m.Meta.Status != "draft" && m.Meta.Status != "stable" {
		t.Fatalf("bad status: %s", m.Meta.Status)
	}
	if len(m.OpenSpec.InstalledIn) == 0 {
		t.Fatal("expected installed_in entries")
	}
}

func TestExpand(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := Expand("~/foo/bar")
	want := filepath.Join(home, "foo", "bar")
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	if Expand("/abs/path") != "/abs/path" {
		t.Fatal("absolute path changed")
	}
}
