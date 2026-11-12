package main

import (
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
	"github.com/farishan-labs/cockpit/internal/herdr"
	"github.com/farishan-labs/cockpit/internal/storage"
	"github.com/farishan-labs/cockpit/internal/terminal"
	"github.com/farishan-labs/cockpit/internal/tui"
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
			fmt.Printf("tmux: %v\n", b.Available())
			fmt.Printf("agents:\n")
			for _, a := range agents.Detect() {
				fmt.Printf("  - %s (%s)\n", a.Name, a.Path)
			}
			if store, err := storage.Open(cfg.DBPath); err == nil {
				store.Close()
				fmt.Printf("storage: ok (%s)\n", cfg.DBPath)
			} else {
				fmt.Printf("storage: %v\n", err)
			}
			if pid, ok := readPID(); ok && isRunning(pid) {
				fmt.Printf("daemon: running (pid %d)\n", pid)
			} else {
				fmt.Println("daemon: stopped")
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

	trayCmd := &cobra.Command{
		Use:   "tray",
		Short: "Run the Ubuntu tray indicator (requires cockpit-tray).",
		RunE: func(cmd *cobra.Command, args []string) error {
			// try installed script
			home, _ := os.UserHomeDir()
			candidates := []string{
				filepath.Join(home, ".local", "bin", "cockpit-tray"),
				filepath.Join(home, "dev", "labs", "cockpit", "scripts", "cockpit-tray.py"),
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					cmd := exec.Command(c)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					return cmd.Run()
				}
			}
			return fmt.Errorf("cockpit-tray not found; install it first")
		},
	}
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

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
