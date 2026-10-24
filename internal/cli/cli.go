package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/kOlapsis/speckit-board/internal/spec"
	"github.com/kOlapsis/speckit-board/internal/status"
	"github.com/kOlapsis/speckit-board/internal/tui"
)

type Info struct {
	Version string
	Commit  string
	Date    string
}

func New(info Info) *cobra.Command {
	var target string
	root := &cobra.Command{
		Use:           "speckit-board [project...]",
		Short:         "Kanban board for SpecKit tasks",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			if !term.IsTerminal(int(os.Stdout.Fd())) {
				return errors.New("the board needs a terminal; use the status command or --html instead")
			}
			if len(args) == 0 {
				args = []string{"."}
			}
			return tui.Run(tui.Options{Roots: args, Target: target})
		},
	}
	root.Flags().StringVar(&target, "target", "", "tmux pane that receives the actions, for example %3")
	root.AddCommand(statusCmd(), versionCmd(info))
	return root
}

func statusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status [path]",
		Short: "Print a one line summary of the current feature",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			info := status.FromProject(spec.Load(path))
			if asJSON {
				b, err := info.JSON()
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), info.Line())
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON instead of a line")
	return cmd
}

func versionCmd(info Info) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit and build date",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "speckit-board %s (commit %s, built %s)\n", info.Version, info.Commit, info.Date)
			return err
		},
	}
}
