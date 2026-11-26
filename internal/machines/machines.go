package machines

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/farishan-labs/cockpit/internal/storage"
)

// Machine is a machine from the machines product (inventory source of truth).
type Machine struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	OS   string   `json:"os"`
	Role string   `json:"role"`
	Host string   `json:"host"`
	Tags []string `json:"tags"`
}

// ListMachines consumes `machines list --json`.
func ListMachines() ([]Machine, string, error) {
	if _, err := exec.LookPath("machines"); err != nil {
		return nil, "", fmt.Errorf("machines not found on PATH")
	}
	out, err := exec.Command("machines", "list", "--json").Output()
	if err != nil {
		return nil, "", fmt.Errorf("machines list --json: %w", err)
	}
	var data struct {
		Machines         []Machine `json:"machines"`
		DefaultMachineID string    `json:"defaultMachineId"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, "", fmt.Errorf("parse machines: %w", err)
	}
	return data.Machines, data.DefaultMachineID, nil
}

// SSHRun executes a command on a remote machine via ssh.
func SSHRun(user, host, cmd string) (string, error) {
	target := host
	if user != "" {
		target = user + "@" + host
	}
	out, err := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", target, cmd).CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("ssh %s: %w: %s", target, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// CheckStep is one step in a remote check.
type CheckStep struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// CheckResult is the aggregated health of a machine.
type CheckResult struct {
	MachineID string      `json:"machine_id"`
	Host      string      `json:"host"`
	Steps     []CheckStep `json:"steps"`
	OK        bool        `json:"ok"`
	WebURL    string      `json:"web_url,omitempty"`
	At        time.Time   `json:"at"`
}

// CheckMachine runs the remote check flow for a machine.
func CheckMachine(store *storage.Store, m Machine, sshUser string, port int) CheckResult {
	if port == 0 {
		port = 3773
	}
	res := CheckResult{MachineID: m.ID, Host: m.Host, At: time.Now().UTC()}
	okAll := true
	add := func(name string, ok bool, detail string) {
		res.Steps = append(res.Steps, CheckStep{Name: name, OK: ok, Detail: detail})
		if !ok {
			okAll = false
		}
	}

	// 1. reachability (ping)
	pingOK := false
	pingOut, err := exec.Command("ping", "-c", "1", "-W", "2", m.Host).CombinedOutput()
	if err == nil {
		pingOK = true
	}
	add("reachability", pingOK, strings.TrimSpace(string(pingOut)))

	// 2. SSH auth probe
	sshOK := false
	sshDetail := ""
	if _, err := SSHRun(sshUser, m.Host, "true"); err == nil {
		sshOK = true
	} else {
		sshDetail = err.Error()
	}
	add("ssh_auth", sshOK, sshDetail)

	// 3. T3 status
	t3OK := false
	t3Detail := ""
	if sshOK {
		out, err := SSHRun(sshUser, m.Host, "t3 --version 2>/dev/null; t3 status --json 2>/dev/null || t3 status")
		if err == nil {
			t3OK = true
			t3Detail = strings.TrimSpace(out)
		} else {
			t3Detail = err.Error()
		}
	}
	add("t3_status", t3OK, t3Detail)

	// 4. HTTP GET
	httpOK := false
	url := fmt.Sprintf("http://%s:%d/", m.Host, port)
	resp, err := http.Get(url)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			httpOK = true
		} else {
			url = ""
		}
	}
	add("http", httpOK, fmt.Sprintf("GET %s", url))

	res.OK = okAll && t3OK && httpOK
	if httpOK {
		res.WebURL = fmt.Sprintf("http://%s:%d/", m.Host, port)
	} else if sshOK {
		res.WebURL = fmt.Sprintf("http://%s:%d/", m.Host, port)
	}

	// persist state + check
	b, _ := json.Marshal(res)
	_ = store.AddMachineCheck(m.ID, string(b))
	st := storage.T3State{MachineID: m.ID, Port: port, ServiceStatus: boolStatus(res.OK), LastCheck: string(b)}
	if old, ok, _ := store.GetT3State(m.ID); ok {
		st.PairingToken = old.PairingToken
	}
	_ = store.UpsertT3State(st)

	return res
}

func boolStatus(ok bool) string {
	if ok {
		return "ok"
	}
	return "degraded"
}

// PairMachine mints a pairing token over SSH.
func PairMachine(sshUser, host string, tailscale bool, ttl string) (string, error) {
	cmd := "t3 pair"
	if tailscale {
		cmd += " --tailscale"
	}
	if ttl != "" {
		cmd += " --ttl " + ttl
	}
	out, err := SSHRun(sshUser, host, cmd)
	if err != nil {
		// fallback without flags
		out2, err2 := SSHRun(sshUser, host, "t3 pair")
		if err2 != nil {
			return out + "\n" + out2, err2
		}
		return strings.TrimSpace(out2), nil
	}
	return strings.TrimSpace(out), nil
}
