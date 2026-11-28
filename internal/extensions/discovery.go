package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Discovered bundles a manifest with its source location.
type Discovered struct {
	Manifest     Manifest
	Dir          string
	ManifestPath string
}

// Discover loads extensions from the default extensions dir and COCKPIT_EXTENSIONS env.
func Discover() ([]Discovered, error) {
	var out []Discovered
	seen := map[string]bool{}

	addDir := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return
		}
		// If dir is a manifest file, handle single.
		if strings.HasSuffix(dir, "cockpit-extension.json") {
			if d, ok := loadManifest(dir); ok {
				if !seen[d.Manifest.Name] {
					seen[d.Manifest.Name] = true
					out = append(out, d)
				}
			}
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			mp := filepath.Join(dir, e.Name(), "cockpit-extension.json")
			if _, err := os.Stat(mp); err != nil {
				continue
			}
			if d, ok := loadManifest(mp); ok {
				if !seen[d.Manifest.Name] {
					seen[d.Manifest.Name] = true
					out = append(out, d)
				}
			}
		}
	}

	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".config", "cockpit", "extensions")
	addDir(base)

	if v := os.Getenv("COCKPIT_EXTENSIONS"); v != "" {
		for _, p := range strings.Split(v, ",") {
			addDir(strings.TrimSpace(p))
		}
	}
	return out, nil
}

func loadManifest(path string) (Discovered, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extensions: read %s: %v\n", path, err)
		return Discovered{}, false
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		fmt.Fprintf(os.Stderr, "extensions: parse %s: %v\n", path, err)
		return Discovered{}, false
	}
	if strings.TrimSpace(m.Name) == "" {
		fmt.Fprintf(os.Stderr, "extensions: %s missing name\n", path)
		return Discovered{}, false
	}
	return Discovered{Manifest: m, Dir: filepath.Dir(path), ManifestPath: path}, true
}
