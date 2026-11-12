package terminal

import (
	"fmt"
	"os/exec"
	"strings"
)

// Session represents a terminal session.
type Session struct {
	Name     string `json:"name"`
	Windows  int    `json:"windows"`
	Attached bool   `json:"attached"`
}

// Backend is the terminal backend interface.
type Backend interface {
	Name() string
	Available() bool
	ListSessions() ([]Session, error)
	CreateSession(name, command string) error
	SendKeys(session, keys string) error
	CapturePane(session string) (string, error)
	KillSession(name string) error
}

// TmuxBackend implements Backend using tmux.
type TmuxBackend struct{}

func NewTmux() *TmuxBackend { return &TmuxBackend{} }

func (t *TmuxBackend) Name() string { return "tmux" }

func (t *TmuxBackend) Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func run(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func (t *TmuxBackend) ListSessions() ([]Session, error) {
	if !t.Available() {
		return nil, fmt.Errorf("tmux not found on PATH")
	}
	out, err := run("list-sessions", "-F", "#{session_name}\t#{session_windows}\t#{session_attached}")
	if err != nil {
		// no sessions -> tmux returns error "no server running"
		if strings.Contains(err.Error(), "no server running") || strings.Contains(err.Error(), "No such file") {
			return []Session{}, nil
		}
		return nil, err
	}
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		s := Session{Name: parts[0]}
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &s.Windows)
		}
		if len(parts) > 2 {
			s.Attached = parts[2] != "0"
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (t *TmuxBackend) CreateSession(name, command string) error {
	if !t.Available() {
		return fmt.Errorf("tmux not found on PATH")
	}
	args := []string{"new-session", "-d", "-s", name}
	if command != "" {
		args = append(args, command)
	}
	_, err := run(args...)
	return err
}

func (t *TmuxBackend) SendKeys(session, keys string) error {
	_, err := run("send-keys", "-t", session, keys, "Enter")
	return err
}

func (t *TmuxBackend) CapturePane(session string) (string, error) {
	out, err := run("capture-pane", "-t", session, "-p")
	return out, err
}

func (t *TmuxBackend) KillSession(name string) error {
	_, err := run("kill-session", "-t", name)
	return err
}
