package cli

import (
	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/command"
)

func NewRootCmd(d Deps) *cobra.Command {
	root := &cobra.Command{
		Use:           "jitctx",
		Short:         "Just-in-time context for AI coding agents",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(
		command.NewScanCmd(d.ScanFactory, d.Logger),
		command.NewQueryCmd(d.Query, d.Logger),
		command.NewPlanCmd(d.Plan, d.Logger),
		command.NewContractsCmd(d.Contracts, d.Logger),
		command.NewListCmd(d.Query, d.Logger),
	)
	return root
}
