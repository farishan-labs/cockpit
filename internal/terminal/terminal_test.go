package terminal

import "testing"

func TestTmuxBackendName(t *testing.T) {
	b := NewTmux()
	if b.Name() != "tmux" {
		t.Fatalf("unexpected name: %s", b.Name())
	}
}
