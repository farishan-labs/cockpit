package agents

import "os/exec"

// Agent is a detected agent CLI.
type Agent struct {
	Name    string `json:"name"`
	Binary  string `json:"binary"`
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

// KnownAgents is the list of agent CLIs cockpit can launch.
var KnownAgents = []string{"claude", "codex", "pi", "opencode", "aider", "goose", "cline"}

// Detect scans PATH for known agents.
func Detect() []Agent {
	var out []Agent
	for _, name := range KnownAgents {
		if p, err := exec.LookPath(name); err == nil {
			out = append(out, Agent{Name: name, Binary: name, Path: p})
		}
	}
	return out
}
