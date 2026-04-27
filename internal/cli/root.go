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
	profileCmd := command.NewProfileCmd(d.Logger)
	profileCmd.AddCommand(command.NewProfileInitCmd(d.InitProfile, d.ProfilesDir, d.Logger))
	profileCmd.AddCommand(command.NewProfileValidateCmd(d.ValidateProfile, d.Logger))

	root.AddCommand(
		command.NewScanCmd(d.ScanFactory, d.Refactor, d.Logger),
		command.NewQueryCmd(d.Query, d.Logger),
		command.NewPlanCmd(d.Plan, d.PlanNew, d.Diff, d.WorkDir, d.PlansDir, d.Logger),
		command.NewContractsCmd(d.Contracts, d.WorkDir, d.PlansDir, d.Logger),
		command.NewScaffoldCmd(d.Scaffold, d.WorkDir, d.PlansDir, d.Logger),
		command.NewListCmd(d.Query, d.Logger),
		command.NewAuditCmd(d.Audit, d.Logger),
		profileCmd,
	)
	return root
}
