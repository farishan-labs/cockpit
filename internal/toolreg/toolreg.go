package toolreg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Tool struct {
	Name        string `toml:"name"`
	Binary      string `toml:"binary"`
	Enabled     bool   `toml:"enabled"`
	Description string `toml:"description"`
}

type Manifest struct {
	Tools []Tool `toml:"tool"`
}

func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cockpit", "tools.toml")
}

func Load() (Manifest, error) {
	var m Manifest
	p := Path()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return m, nil
	}
	if _, err := toml.DecodeFile(p, &m); err != nil {
		return m, err
	}
	return m, nil
}

func Save(m Manifest) error {
	os.MkdirAll(filepath.Dir(Path()), 0o755)
	f, err := os.Create(Path())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(m)
}

func Find(m Manifest, name string) (Tool, int) {
	for i, t := range m.Tools {
		if t.Name == name {
			return t, i
		}
	}
	return Tool{}, -1
}

func Add(name, binary, desc string) error {
	m, err := Load()
	if err != nil {
		return err
	}
	if _, idx := Find(m, name); idx >= 0 {
		return fmt.Errorf("tool %s already exists", name)
	}
	m.Tools = append(m.Tools, Tool{Name: name, Binary: binary, Enabled: true, Description: desc})
	return Save(m)
}

func Remove(name string) error {
	m, err := Load()
	if err != nil {
		return err
	}
	_, idx := Find(m, name)
	if idx < 0 {
		return fmt.Errorf("tool %s not found", name)
	}
	m.Tools = append(m.Tools[:idx], m.Tools[idx+1:]...)
	return Save(m)
}

func SetEnabled(name string, enabled bool) error {
	m, err := Load()
	if err != nil {
		return err
	}
	_, idx := Find(m, name)
	if idx < 0 {
		return fmt.Errorf("tool %s not found", name)
	}
	m.Tools[idx].Enabled = enabled
	return Save(m)
}

func Run(name string, args []string) error {
	m, err := Load()
	if err != nil {
		return err
	}
	t, idx := Find(m, name)
	if idx < 0 {
		return fmt.Errorf("tool %s not found", name)
	}
	if !t.Enabled {
		return fmt.Errorf("tool %s is disabled", name)
	}
	bin := t.Binary
	if bin == "" {
		bin = t.Name
	}
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("binary not found: %s", bin)
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
