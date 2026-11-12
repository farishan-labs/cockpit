package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.Port == 0 {
		t.Fatal("default port zero")
	}
	if c.TerminalBackend != "tmux" {
		t.Fatalf("unexpected backend: %s", c.TerminalBackend)
	}
}

func TestUserMDPathDefault(t *testing.T) {
	c := Default()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "cockpit", "USER.md")
	if c.UserMDPath != want {
		t.Fatalf("default user_md_path: got %s want %s", c.UserMDPath, want)
	}
}
