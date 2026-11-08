package main

import (
	"fmt"

	"github.com/farishan-labs/cockpit/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "cockpit",
		Short: "The Faris Han Labs control plane: a human+AI cockpit.",
		Long: `cockpit aggregates and extends existing tools (herdr, pi, opencode, nakama, ...)
into one minimal, extensible control plane. It reuses wheels; it does not reinvent them.`,
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
		Short: "Run the cockpit core daemon (TUI + embedded web UI).",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("cockpit serve: not implemented yet (M1).")
			return nil
		},
	}
	root.AddCommand(serve)

	doctor := &cobra.Command{
		Use:   "doctor",
		Short: "Check that required tools (tmux, agents) are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("cockpit doctor: not implemented yet (M1).")
			return nil
		},
	}
	root.AddCommand(doctor)

	if err := root.Execute(); err != nil {
		fmt.Println(err)
	}
}
