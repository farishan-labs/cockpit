package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestParseMinimal(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "foo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cockpit-extension.json"), []byte(`{"name":"foo"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	d, ok := loadManifest(filepath.Join(dir, "cockpit-extension.json"))
	if !ok {
		t.Fatal("expected manifest load")
	}
	if d.Manifest.Name != "foo" {
		t.Fatalf("name: got %s", d.Manifest.Name)
	}
	if d.Manifest.EffectiveKind() != KindRemote {
		t.Fatalf("default kind should be remote, got %s", d.Manifest.EffectiveKind())
	}
	if !d.Manifest.IsEnabled() {
		t.Fatal("default enabled should be true")
	}
}

func TestManifestInvalidSkipped(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "bad")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "cockpit-extension.json"), []byte(`{"version":"1"}`), 0o644)
	if _, ok := loadManifest(filepath.Join(dir, "cockpit-extension.json")); ok {
		t.Fatal("expected invalid manifest to be skipped")
	}
}

func TestDiscoverDedup(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"a", "b"} {
		d := filepath.Join(tmp, name)
		os.MkdirAll(d, 0o755)
		os.WriteFile(filepath.Join(d, "cockpit-extension.json"), []byte(`{"name":"same"}`), 0o644)
	}
	t.Setenv("COCKPIT_EXTENSIONS", tmp)
	list, err := Discover()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, d := range list {
		if d.Manifest.Name == "same" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected dedup to 1, got %d", count)
	}
}
