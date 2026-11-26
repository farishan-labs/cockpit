package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestT3StateCRUD(t *testing.T) {
	dir, _ := os.MkdirTemp("", "cockpit")
	defer os.RemoveAll(dir)
	st, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.UpsertT3State(T3State{MachineID: "m1", Port: 3773, ServiceStatus: "ok"}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.GetT3State("m1")
	if err != nil || !ok {
		t.Fatalf("get: %v %v", err, ok)
	}
	if got.Port != 3773 || got.ServiceStatus != "ok" {
		t.Fatalf("bad: %+v", got)
	}
	if err := st.AddMachineCheck("m1", `{"ok":true}`); err != nil {
		t.Fatal(err)
	}
	ch, err := st.ListMachineChecks("m1", 10)
	if err != nil || len(ch) != 1 {
		t.Fatalf("checks: %v %d", err, len(ch))
	}
}
