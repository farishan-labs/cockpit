package herdr

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type terminal struct {
	App string `json:"app"`
}
type tab struct {
	Terminals []terminal `json:"terminals"`
}
type space struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Folder  string `json:"folder"`
	Active  bool   `json:"active"`
	Tabs    []tab  `json:"tabs"`
}

type lsOut struct {
	Spaces []space `json:"spaces"`
}

func hasOpencode(s space) bool {
	for _, t := range s.Tabs {
		for _, term := range t.Terminals {
			if term.App == "opencode" {
				return true
			}
		}
	}
	return false
}

// Open reproduces the legacy cockpit launcher: ensure herdr-app is running,
// reuse or create a space at preferredDir with opencode, then raise window.
func Open(preferredDir, spaceName string) error {
	if preferredDir == "" {
		home, _ := os.UserHomeDir()
		preferredDir = filepath.Join(home, "dev", "labs")
	}
	if spaceName == "" {
		spaceName = filepath.Base(preferredDir)
	}
	for _, bin := range []string{"herdr-app", "opencode"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("%s is required but not on PATH", bin)
		}
	}
	if err := os.MkdirAll(preferredDir, 0o755); err != nil {
		return err
	}

	// ensure herdr-app running
	if err := exec.Command("herdr-app", "status").Run(); err != nil {
		cmd := exec.Command("herdr-app")
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start herdr-app: %w", err)
		}
		for i := 0; i < 50; i++ {
			if exec.Command("herdr-app", "status").Run() == nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if err := exec.Command("herdr-app", "status").Run(); err != nil {
			return fmt.Errorf("herdr-app did not start")
		}
	}

	// list spaces
	out, err := exec.Command("herdr-app", "ls").Output()
	if err != nil {
		return fmt.Errorf("herdr-app ls: %w", err)
	}
	var data lsOut
	if err := json.Unmarshal(out, &data); err != nil {
		return fmt.Errorf("parse herdr-app ls: %w", err)
	}

	// candidates
	var candidates []space
	for _, s := range data.Spaces {
		if s.Folder != preferredDir {
			continue
		}
		if !hasOpencode(s) {
			continue
		}
		candidates = append(candidates, s)
	}

	if len(candidates) > 0 {
		var pick space
		active := []space{}
		for _, c := range candidates {
			if c.Active {
				active = append(active, c)
			}
		}
		if len(active) > 0 {
			pick = active[0]
		} else {
			pick = candidates[0]
		}
		_ = exec.Command("herdr-app", "do", "space.select", "--id", pick.ID).Run()
		fmt.Printf("cockpit: opened existing space at %s (opencode ready).\n", preferredDir)
	} else {
		argv := `["opencode"]`
		cmd := exec.Command("herdr-app", "do", "space.new", "--name", spaceName, "--folder", preferredDir, "--argv", argv)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("space.new: %w", err)
		}
		fmt.Printf("cockpit: created space '%s' at %s with opencode.\n", spaceName, preferredDir)
	}

	// raise window
	_ = exec.Command("herdr-app").Run()
	return nil
}

func PreferredDir() string {
	if v := os.Getenv("COCKPIT_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "dev", "labs")
}

func SpaceName(preferredDir string) string {
	if v := os.Getenv("COCKPIT_SPACE_NAME"); v != "" {
		return v
	}
	return filepath.Base(preferredDir)
}
