package config

import "testing"

func TestDefault(t *testing.T) {
	c := Default()
	if c.Port == 0 {
		t.Fatal("default port zero")
	}
	if c.TerminalBackend != "tmux" {
		t.Fatalf("unexpected backend: %s", c.TerminalBackend)
	}
}
