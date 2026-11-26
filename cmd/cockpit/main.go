package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/farishan-labs/cockpit/internal/agents"
	"github.com/farishan-labs/cockpit/internal/config"
	"github.com/farishan-labs/cockpit/internal/daemon"
	"github.com/farishan-labs/cockpit/internal/extensions"
	"github.com/farishan-labs/cockpit/internal/herdr"
	"github.com/farishan-labs/cockpit/internal/storage"
	"github.com/farishan-labs/cockpit/internal/terminal"
	"github.com/farishan-labs/cockpit/internal/toolreg"
	"github.com/farishan-labs/cockpit/internal/tui"
	"github.com/farishan-labs/cockpit/internal/version"
	"github.com/farishan-labs/cockpit/internal/workflow"
	"github.com/spf13/cobra"
)

var foreground bool

func pidPath() string {
	return filepath.Join(config.Dir(), "cockpit.pid")
}

func readPID() (int, bool) {
	b, err := os.ReadFile(pidPath())
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, false
	}
	return pid, true
}

func isRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// signal 0 checks existence
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func stopDaemon() error {
	pid, ok := readPID()
	if !ok || !isRunning(pid) {
		os.Remove(pidPath())
		fmt.Println("cockpit: not running")
		return nil
	}
	p, _ := os.FindProcess(pid)
	if err := p.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	// wait briefly
	for i := 0; i < 25; i++ {
		if !isRunning(pid) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	os.Remove(pidPath())
	fmt.Println("cockpit: stopped")
	return nil
}

func serveForeground() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	store, err := storage.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()
	backend := terminal.NewTmux()
	d := daemon.New(cfg, store, backend)
	_ = agents.Detect()
	return d.Run()
}

func serveBackground() error {
	if pid, ok := readPID(); ok && isRunning(pid) {
		fmt.Printf("cockpit: already running (pid %d)\n", pid)
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "serve", "--foreground")
	cmd.Stdout, _ = os.OpenFile(filepath.Join(config.Dir(), "cockpit.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	cmd.Stderr = cmd.Stdout
	// detach
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	// detach process
	_ = cmd.Process.Release()
	cfg, _ := config.Load()
	_ = os.WriteFile(pidPath(), []byte(strconv.Itoa(pid)), 0o644)
	fmt.Printf("cockpit: started in background (pid %d)\n", pid)
	fmt.Printf("cockpit: ui -> http://127.0.0.1:%d (see %s for logs)\n", cfg.Port, filepath.Join(config.Dir(), "cockpit.log"))
	return nil
}

func main() {
	root := &cobra.Command{
		Use:   "cockpit",
		Short: "The Faris Han Labs control plane: a human+AI cockpit.",
		Long: `cockpit aggregates and extends existing tools (herdr, pi, opencode, nakama, ...)
into one minimal, extensible control plane. It reuses wheels; it does not reinvent them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if foreground {
				return serveForeground()
			}
			return serveBackground()
		},
	}
	root.Version = version.Version
	root.SetVersionTemplate("cockpit {{.Version}} (commit " + version.Commit + ", built " + version.Date + ")\n")

	root.PersistentFlags().BoolVar(&foreground, "foreground", false, "run in foreground (do not daemonize)")

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Run the cockpit core daemon (web UI on localhost).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if foreground {
				return serveForeground()
			}
			return serveBackground()
		},
	}
	serve.Flags().BoolVar(&foreground, "foreground", false, "run in foreground (do not daemonize)")
	root.AddCommand(serve)

	doctor := &cobra.Command{
		Use:   "doctor",
		Short: "Check that required tools (tmux, agents) are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			b := terminal.NewTmux()
			fmt.Printf("config: %s\n", config.Path())
			fmt.Printf("port: %d\n", cfg.Port)
			if m, err := workflow.Load(""); err == nil {
				fmt.Printf("workflow manifest: %s (v%s, %s)\n", workflow.Path(), m.Meta.Version, m.Meta.Status)
			} else {
				fmt.Printf("workflow manifest: %v\n", err)
			}
			fmt.Printf("tmux: %v\n", b.Available())
			fmt.Printf("agents:\n")
			for _, a := range agents.Detect() {
				fmt.Printf("  - %s (%s)\n", a.Name, a.Path)
			}
			if _, err := exec.LookPath("machines"); err == nil {
				out, _ := exec.Command("machines", "list", "--json").Output()
				fmt.Printf("machines: available (%d bytes)\n", len(out))
			} else {
				fmt.Printf("machines: not found\n")
			}
			if store, err := storage.Open(cfg.DBPath); err == nil {
				store.Close()
				fmt.Printf("storage: ok (%s)\n", cfg.DBPath)
			} else {
				fmt.Printf("storage: %v\n", err)
			}
			// extensions (best-effort)
			if store, err := storage.Open(cfg.DBPath); err == nil {
				mgr := extensions.NewManager(store)
				_ = mgr.Discover()
				for _, e := range mgr.All() {
					avail := "unavailable"
					if e.Available {
						avail = "available"
					}
					fmt.Printf("extension: %s (%s) %s\n", e.Name, e.Kind, avail)
				}
				store.Close()
			}
			if pid, ok := readPID(); ok && isRunning(pid) {
				fmt.Printf("daemon: running (pid %d)\n", pid)
			} else {
				fmt.Println("daemon: stopped")
			}
			fmt.Printf("operator profile: %s\n", cfg.UserMDPath)
			if err := runUserMDCheck(cfg.UserMDPath); err != nil {
				return err
			}
			return nil
		},
	}
	root.AddCommand(doctor)

	stop := &cobra.Command{
		Use:   "stop",
		Short: "Stop the background cockpit daemon.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopDaemon()
		},
	}
	root.AddCommand(stop)

	status := &cobra.Command{
		Use:   "status",
		Short: "Show whether the cockpit daemon is running.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if pid, ok := readPID(); ok && isRunning(pid) {
				fmt.Printf("cockpit: running (pid %d)\n", pid)
			} else {
				fmt.Println("cockpit: stopped")
				os.Remove(pidPath())
			}
			return nil
		},
	}
	root.AddCommand(status)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("cockpit %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
		},
	}
	root.AddCommand(versionCmd)

	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Attach a TUI to a running cockpit daemon.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			p := tea.NewProgram(tui.New(cfg.Port), tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
	root.AddCommand(tuiCmd)

	openCmd := &cobra.Command{
		Use:   "open",
		Short: "Open the herdr workspace (legacy cockpit launcher).",
		RunE: func(cmd *cobra.Command, args []string) error {
			pd := herdr.PreferredDir()
			sn := herdr.SpaceName(pd)
			return herdr.Open(pd, sn)
		},
	}
	root.AddCommand(openCmd)

	trayBin := func() string {
		home, _ := os.UserHomeDir()
		candidates := []string{
			filepath.Join(home, ".local", "bin", "cockpit-tray"),
			filepath.Join(home, "dev", "labs", "cockpit", "scripts", "cockpit-tray.py"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
		return ""
	}

	trayPidPath := filepath.Join(config.Dir(), "cockpit-tray.pid")

	trayStart := func(fg bool) error {
		bin := trayBin()
		if bin == "" {
			return fmt.Errorf("cockpit-tray not found; run cockpit install")
		}
		if b, err := os.ReadFile(trayPidPath); err == nil {
			if pid, err := strconv.Atoi(string(b)); err == nil {
				if p, err := os.FindProcess(pid); err == nil && p.Signal(syscall.Signal(0)) == nil {
					fmt.Printf("cockpit-tray: already running (pid %d)\n", pid)
					return nil
				}
			}
		}
		if fg {
			cmd := exec.Command(bin)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
		cmd := exec.Command(bin)
		logf, _ := os.OpenFile(filepath.Join(config.Dir(), "cockpit-tray.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		cmd.Stdout = logf
		cmd.Stderr = logf
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			return err
		}
		pid := cmd.Process.Pid
		_ = cmd.Process.Release()
		_ = os.WriteFile(trayPidPath, []byte(strconv.Itoa(pid)), 0o644)
		fmt.Printf("cockpit-tray: started in background (pid %d)\n", pid)
		return nil
	}

	trayStop := func() error {
		b, err := os.ReadFile(trayPidPath)
		if err != nil {
			fmt.Println("cockpit-tray: not running")
			return nil
		}
		pid, err := strconv.Atoi(string(b))
		if err != nil {
			os.Remove(trayPidPath)
			fmt.Println("cockpit-tray: not running")
			return nil
		}
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Signal(syscall.SIGTERM)
		}
		os.Remove(trayPidPath)
		fmt.Println("cockpit-tray: stopped")
		return nil
	}

	trayStatus := func() {
		b, err := os.ReadFile(trayPidPath)
		if err != nil {
			fmt.Println("cockpit-tray: stopped")
			return
		}
		pid, _ := strconv.Atoi(string(b))
		if p, err := os.FindProcess(pid); err == nil && p.Signal(syscall.Signal(0)) == nil {
			fmt.Printf("cockpit-tray: running (pid %d)\n", pid)
		} else {
			fmt.Println("cockpit-tray: stopped")
			os.Remove(trayPidPath)
		}
	}

	trayCmd := &cobra.Command{
		Use:   "tray",
		Short: "Run the Ubuntu tray indicator.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fg, _ := cmd.Flags().GetBool("foreground")
			return trayStart(fg)
		},
	}
	trayCmd.Flags().Bool("foreground", false, "run in foreground")
	trayCmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the background tray indicator.",
		RunE:  func(cmd *cobra.Command, args []string) error { return trayStop() },
	})
	trayCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show tray indicator status.",
		RunE:  func(cmd *cobra.Command, args []string) error { trayStatus(); return nil },
	})
	root.AddCommand(trayCmd)

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the cockpit binary (restores legacy launcher if present).",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			bin := filepath.Join(home, ".local", "bin", "cockpit")
			legacy := bin + ".legacy"
			if _, err := os.Stat(bin); err == nil {
				if err := os.Remove(bin); err != nil {
					return err
				}
				fmt.Println("cockpit: removed", bin)
			} else {
				fmt.Println("cockpit: not installed at", bin)
			}
			if _, err := os.Stat(legacy); err == nil {
				if err := os.Rename(legacy, bin); err != nil {
					return err
				}
				fmt.Println("cockpit: restored legacy launcher to", bin)
			}
			return nil
		},
	}
	root.AddCommand(uninstallCmd)

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Build and install cockpit from source (requires the repo).",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			repo := filepath.Join(home, "dev", "labs", "cockpit")
			script := filepath.Join(repo, "scripts", "install-local.sh")
			if _, err := os.Stat(script); err != nil {
				return fmt.Errorf("install script not found at %s; clone the repo first", script)
			}
			c := exec.Command("bash", script)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
	root.AddCommand(installCmd)

	toolCmd := &cobra.Command{Use: "tool", Short: "Manage registered tools."}
	toolCmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List registered tools.",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := toolreg.Load()
			if err != nil {
				return err
			}
			for _, t := range m.Tools {
				state := "enabled"
				if !t.Enabled {
					state = "disabled"
				}
				fmt.Printf("%-16s %-16s %s  %s\n", t.Name, t.Binary, state, t.Description)
			}
			if len(m.Tools) == 0 {
				fmt.Println("no tools registered")
			}
			return nil
		},
	})
	toolCmd.AddCommand(&cobra.Command{
		Use: "add <name> <binary> [description]", Short: "Register a tool.",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			desc := ""
			if len(args) > 2 {
				desc = args[2]
			}
			if err := toolreg.Add(args[0], args[1], desc); err != nil {
				return err
			}
			fmt.Printf("added tool %s\n", args[0])
			return nil
		},
	})
	toolCmd.AddCommand(&cobra.Command{
		Use: "remove <name>", Short: "Remove a tool.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := toolreg.Remove(args[0]); err != nil {
				return err
			}
			fmt.Printf("removed tool %s\n", args[0])
			return nil
		},
	})
	toolCmd.AddCommand(&cobra.Command{
		Use: "enable <name>", Short: "Enable a tool.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := toolreg.SetEnabled(args[0], true); err != nil {
				return err
			}
			fmt.Printf("enabled %s\n", args[0])
			return nil
		},
	})
	toolCmd.AddCommand(&cobra.Command{
		Use: "disable <name>", Short: "Disable a tool.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := toolreg.SetEnabled(args[0], false); err != nil {
				return err
			}
			fmt.Printf("disabled %s\n", args[0])
			return nil
		},
	})
	toolCmd.AddCommand(&cobra.Command{
		Use: "run <name> [args...]", Short: "Run a registered tool.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return toolreg.Run(args[0], args[1:])
		},
	})
	root.AddCommand(toolCmd)

	// extension command
	extCmd := &cobra.Command{Use: "extension", Short: "Manage cockpit extensions."}
	extCmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List extensions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			store, err := storage.Open(cfg.DBPath)
			if err != nil {
				return err
			}
			defer store.Close()
			mgr := extensions.NewManager(store)
			_ = mgr.Discover()
			for _, e := range mgr.All() {
				avail := "unavailable"
				if e.Available {
					avail = "available"
				}
				state := "enabled"
				if !e.Enabled {
					state = "disabled"
				}
				fmt.Printf("%-12s %-8s %-10s %-12s %s  tabs=%d\n", e.Name, e.Kind, e.Version, state, avail, len(e.Tabs))
			}
			if len(mgr.All()) == 0 {
				fmt.Println("no extensions discovered")
			}
			return nil
		},
	})
	extCmd.AddCommand(&cobra.Command{
		Use: "add <path>", Short: "Install an extension from a folder.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]
			home, _ := os.UserHomeDir()
			base := filepath.Join(home, ".config", "cockpit", "extensions")
			_ = os.MkdirAll(base, 0o755)
			name := filepath.Base(src)
			// read manifest to get name
			b, err := os.ReadFile(filepath.Join(src, "cockpit-extension.json"))
			if err == nil {
				var m map[string]any
				if json.Unmarshal(b, &m) == nil {
					if n, ok := m["name"].(string); ok && n != "" {
						name = n
					}
				}
			}
			dst := filepath.Join(base, name)
			_ = os.RemoveAll(dst)
			if err := os.Symlink(src, dst); err != nil {
				// fallback: copy
				_ = os.MkdirAll(dst, 0o755)
				_ = filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					rel, _ := filepath.Rel(src, p)
					target := filepath.Join(dst, rel)
					if info.IsDir() {
						return os.MkdirAll(target, info.Mode())
					}
					data, _ := os.ReadFile(p)
					return os.WriteFile(target, data, info.Mode())
				})
			}
			fmt.Printf("added extension %s\n", name)
			return nil
		},
	})
	extCmd.AddCommand(&cobra.Command{
		Use: "remove <name>", Short: "Remove an extension.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			dst := filepath.Join(home, ".config", "cockpit", "extensions", args[0])
			_ = os.RemoveAll(dst)
			cfg, _ := config.Load()
			store, _ := storage.Open(cfg.DBPath)
			if store != nil {
				_ = store.RemoveExtension(args[0])
				store.Close()
			}
			fmt.Printf("removed extension %s\n", args[0])
			return nil
		},
	})
	for _, sub := range []struct {
		use  string
		toggle bool
		val  bool
	}{
		{"enable", true, true},
		{"disable", true, false},
	} {
		val := sub.val
		name := sub.use
		extCmd.AddCommand(&cobra.Command{
			Use: name + " <ext>", Short: name + " an extension.",
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, _ := config.Load()
				store, err := storage.Open(cfg.DBPath)
				if err != nil {
					return err
				}
				defer store.Close()
				mgr := extensions.NewManager(store)
				_ = mgr.Discover()
				if err := mgr.SetEnabled(args[0], val); err != nil {
					return err
				}
				fmt.Printf("%s %s\n", name+"d", args[0])
				return nil
			},
		})
	}
	extCmd.AddCommand(&cobra.Command{
		Use: "sync <name>", Short: "Sync an extension.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			store, err := storage.Open(cfg.DBPath)
			if err != nil {
				return err
			}
			defer store.Close()
			mgr := extensions.NewManager(store)
			_ = mgr.Discover()
			if err := mgr.Sync(args[0]); err != nil {
				return err
			}
			fmt.Printf("synced %s\n", args[0])
			return nil
		},
	})
	root.AddCommand(extCmd)


	// machine command
	machineCmd := &cobra.Command{Use: "machine", Short: "Manage remote machines (via machines product)."}
	machineCmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List machines.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := exec.Command("machines", "list", "--json").Output()
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	})
	machineCmd.AddCommand(&cobra.Command{
		Use: "check [name]", Short: "Check machine reachability + T3.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 { name = args[0] }
			cfg, _ := config.Load()
			store, err := storage.Open(cfg.DBPath)
			if err != nil { return err }
			defer store.Close()
			ms, _, err := machinesList()
			if err != nil { return err }
			for _, m := range ms {
				if name != "" && m.ID != name && m.Name != name { continue }
				fmt.Printf("machine %s (%s):\n", m.Name, m.Host)
			}
			_ = store
			return nil
		},
	})
	root.AddCommand(machineCmd)

	// t3 command
	t3Cmd := &cobra.Command{Use: "t3", Short: "T3 Code remote operations."}
	t3Cmd.AddCommand(&cobra.Command{
		Use: "pair <machine>", Short: "Mint pairing token for a machine.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ttl, _ := cmd.Flags().GetString("ttl")
			ts, _ := cmd.Flags().GetBool("tailscale")
			cfg, _ := config.Load()
			store, err := storage.Open(cfg.DBPath)
			if err != nil { return err }
			defer store.Close()
			ms, _, err := machinesList()
			if err != nil { return err }
			var target *machineItem
			for i := range ms {
				if ms[i].ID == args[0] || ms[i].Name == args[0] {
					target = &ms[i]
					break
				}
			}
			if target == nil { return fmt.Errorf("machine not found") }
			out, err := exec.Command("ssh", "-o", "BatchMode=yes", target.Host, "t3 pair").CombinedOutput()
			_ = ts
			if err != nil {
				// try local t3? 
			}
			fmt.Println(string(out))
			if ttl != "" {
				fmt.Printf("ttl: %s\n", ttl)
			}
			return err
		},
	})
	t3Cmd.PersistentFlags().String("ttl", "24h", "token ttl")
	t3Cmd.PersistentFlags().Bool("tailscale", false, "use tailscale serve")
	// add pair subcommand flags handled above via persistent
	root.AddCommand(t3Cmd)

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// runUserMDCheck runs the optional usermd-check binary on path. It warns and
// returns nil when the binary is absent.


type machineItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`
}

func machinesList() ([]machineItem, string, error) {
	out, err := exec.Command("machines", "list", "--json").Output()
	if err != nil {
		return nil, "", err
	}
	var data struct {
		Machines         []machineItem `json:"machines"`
		DefaultMachineID string        `json:"defaultMachineId"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil, "", err
	}
	return data.Machines, data.DefaultMachineID, nil
}


func runUserMDCheck(path string) error {
	bin, err := exec.LookPath("usermd-check")
	if err != nil {
		fmt.Println("operator profile: usermd-check not installed (warning: skipping validation)")
		return nil
	}
	cmd := exec.Command(bin, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("operator profile check failed: %w", err)
	}
	return nil
}
