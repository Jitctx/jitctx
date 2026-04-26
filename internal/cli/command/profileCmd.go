package command

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// NewProfileCmd constructs the parent "profile" cobra group. It has no
// RunE — running "jitctx profile" with no subcommand prints help and
// exits 0 (cobra default). Children (init, validate-future) attach
// via cmd.AddCommand inside cli.NewRootCmd.
func NewProfileCmd(_ *slog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "profile",
		Short: "Manage framework profiles (init, validate)",
		Long: `Manage framework profiles. Subcommands:

  init <name>      Extract a bundled profile into .jitctx/profiles/<name>/`,
		Args: cobra.NoArgs,
	}
}
