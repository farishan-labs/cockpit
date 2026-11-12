package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/bubbletea"
	"github.com/farishan-labs/cockpit/internal/agents"
	"github.com/farishan-labs/cockpit/internal/config"
	"github.com/farishan-labs/cockpit/internal/daemon"
	"github.com/farishan-labs/cockpit/internal/herdr"
	"github.com/farishan-labs/cockpit/internal/storage"
	"github.com/farishan-labs/cockpit/internal/terminal"
	"github.com/farishan-labs/cockpit/internal/tui"
	"github.com/farishan-labs/cockpit/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "cockpit",
		Short: "The Faris Han Labs control plane: a human+AI cockpit.",
		Long: `cockpit aggregates and extends existing tools (herdr, pi, opencode, nakama, ...)
into one minimal, extensible control plane. It reuses wheels; it does not reinvent them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// default: run the daemon
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
			return d.Run()
		},
	}

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("cockpit %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
		},
	})

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Run the cockpit core daemon (web UI on localhost).",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			store, err := storage.Open(cfg.DBPath)
			if err != nil {
				return err
			}
			defer store.Close()
			var backend terminal.Backend
			switch cfg.TerminalBackend {
			case "tmux":
				backend = terminal.NewTmux()
			default:
				backend = terminal.NewTmux()
			}
			d := daemon.New(cfg, store, backend)
			_ = agents.Detect()
			return d.Run()
		},
	}
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
			return nil
		},
	}
	root.AddCommand(doctor)

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
	}
}
