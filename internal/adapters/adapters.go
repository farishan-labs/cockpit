package adapters

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Adapter aggregates an external tool into the cockpit dashboard.
type Adapter interface {
	Name() string
	Available() bool
	Status() (map[string]any, error)
}

// Herdr aggregates herdr-core (optional richer status + remote machines).
type Herdr struct{}

func (Herdr) Name() string { return "herdr" }

func (Herdr) Available() bool {
	_, err := exec.LookPath("herdr-app")
	return err == nil
}

func (Herdr) Status() (map[string]any, error) {
	if _, err := exec.LookPath("herdr-app"); err != nil {
		return map[string]any{"available": false}, nil
	}
	out, err := exec.Command("herdr-app", "ls").Output()
	if err != nil {
		return nil, fmt.Errorf("herdr-app ls: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(out, &data); err != nil {
		// fall back to raw
		return map[string]any{"available": true, "raw": string(out)}, nil
	}
	spaces, _ := data["spaces"].([]any)
	return map[string]any{
		"available": true,
		"spaces":    len(spaces),
	}, nil
}

// Opencode aggregates opencode session/agent info (minimal: presence + version).
type Opencode struct{}

func (Opencode) Name() string { return "opencode" }

func (Opencode) Available() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

func (Opencode) Status() (map[string]any, error) {
	if _, err := exec.LookPath("opencode"); err != nil {
		return map[string]any{"available": false}, nil
	}
	out, _ := exec.Command("opencode", "--version").Output()
	return map[string]any{
		"available": true,
		"version":   string(out),
	}, nil
}

// Pi aggregates pi (extension/skill discovery).
type Pi struct{}

func (Pi) Name() string { return "pi" }

func (Pi) Available() bool {
	_, err := exec.LookPath("pi")
	return err == nil
}

func (Pi) Status() (map[string]any, error) {
	if _, err := exec.LookPath("pi"); err != nil {
		return map[string]any{"available": false}, nil
	}
	out, _ := exec.Command("pi", "--version").Output()
	return map[string]any{
		"available": true,
		"version":   string(out),
	}, nil
}

// All returns the default adapter set for M3.
func All() []Adapter {
	return []Adapter{Herdr{}, Opencode{}, Pi{}}
}
