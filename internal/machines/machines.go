package machines

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"os/exec"
	"strings"
	"time"

	"github.com/farishan-labs/cockpit/internal/storage"
)

// Tunnel describes how to reach a machine (e.g. tailscale).
type Tunnel struct {
	Backend string `json:"backend,omitempty"`
	Target  string `json:"target,omitempty"`
	Command string `json:"command,omitempty"`
}

// Machine is a machine from the machines product (inventory source of truth).
type Machine struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	OS     string   `json:"os"`
	Role   string   `json:"role"`
	Host   string   `json:"host"`
	Tunnel *Tunnel  `json:"tunnel,omitempty"`
	Tags   []string `json:"tags"`
}

// ConnectHost returns the host to use for reachability/SSH/HTTP,
// preferring the tunnel target (e.g. tailscale) when available.
func (m Machine) ConnectHost() string {
	if m.Tunnel != nil && m.Tunnel.Target != "" {
		return m.Tunnel.Target
	}
	return m.Host
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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "ssh",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "ServerAliveInterval=5",
		"-o", "ServerAliveCountMax=1",
		target, cmd)
	out, err := c.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("ssh %s: timed out waiting for connection/auth", target)
	}
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
	host := m.ConnectHost()
	res := CheckResult{MachineID: m.ID, Host: host, At: time.Now().UTC()}
	okAll := true
	add := func(name string, ok bool, detail string) {
		res.Steps = append(res.Steps, CheckStep{Name: name, OK: ok, Detail: detail})
		if !ok {
			okAll = false
		}
	}

	// 1. reachability (ping)
	pingOK := false
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	pingOut, err := exec.CommandContext(pingCtx, "ping", "-c", "1", "-W", "2000", host).CombinedOutput()
	if err == nil {
		pingOK = true
	}
	add("reachability", pingOK, strings.TrimSpace(string(pingOut)))

	// 2. SSH auth probe
	sshOK := false
	sshDetail := ""
	if _, err := SSHRun(sshUser, host, "true"); err == nil {
		sshOK = true
	} else {
		sshDetail = err.Error()
	}
	add("ssh_auth", sshOK, sshDetail)

	// 3. T3 status
	t3OK := false
	t3Detail := ""
	if sshOK {
		out, err := SSHRun(sshUser, host, "export PATH=\"$HOME/.local/bin:$PATH\"; t3 --version 2>/dev/null; t3 status --json 2>/dev/null || t3 status")
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
	url := fmt.Sprintf("http://%s:%d/", host, port)
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
		res.WebURL = fmt.Sprintf("http://%s:%d/", host, port)
	} else if sshOK {
		res.WebURL = fmt.Sprintf("http://%s:%d/", host, port)
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
	cmd := "export PATH=\"$HOME/.local/bin:$PATH\"; t3 pair"
	if tailscale {
		cmd += " --tailscale"
	}
	if ttl != "" {
		cmd += " --ttl " + ttl
	}
	out, err := SSHRun(sshUser, host, cmd)
	if err != nil {
		if u := tailscaleAuthURL(out); u != "" {
			acct := tailscaleAccount(); msg := "tailscale SSH approval required"; if acct != "" { msg += " (tailnet account: " + acct + ")" }; return "", fmt.Errorf("%s: visit %s", msg, u)
		}
		// fallback without flags (only for flag-related failures)
		out2, err2 := SSHRun(sshUser, host, "export PATH=\"$HOME/.local/bin:$PATH\"; t3 pair")
		if err2 != nil {
			if u := tailscaleAuthURL(out2); u != "" {
				acct := tailscaleAccount(); msg := "tailscale SSH approval required"; if acct != "" { msg += " (tailnet account: " + acct + ")" }; return "", fmt.Errorf("%s: visit %s", msg, u)
			}
			detail := strings.TrimSpace(out2)
			if strings.Contains(detail, "command not found") || strings.Contains(detail, "not found") {
				return "", fmt.Errorf("t3 CLI not found on %s; install T3 Code on the remote machine", host)
			}
			return "", fmt.Errorf("ssh to %s failed: %s", host, detail)
		}
		return strings.TrimSpace(out2), nil
	}
	return strings.TrimSpace(out), nil
}

func tailscaleAuthURL(s string) string {
	re := regexp.MustCompile(`https://login\.tailscale\.com/a/[A-Za-z0-9]+`)
	return re.FindString(s)
}

func tailscaleAccount() string {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return ""
	}
	var d struct {
		Self struct {
			CapMap map[string][]interface{} `json:"CapMap"`
		} `json:"Self"`
	}
	if err := json.Unmarshal(out, &d); err != nil {
		return ""
	}
	if v := d.Self.CapMap["tailnet-display-name"]; len(v) > 0 {
		if s, ok := v[0].(string); ok {
			return s
		}
	}
	return ""
}
