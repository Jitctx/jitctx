package command

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/planuc"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

type planOpts struct {
	module string
	output string
}

func NewPlanCmd(uc planuc.UseCase, _ *slog.Logger) *cobra.Command {
	var opts planOpts
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the parallel execution plan for a module",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := uc.Execute(cmd.Context(), planvo.PlanModuleInput{Module: opts.module})
			if err != nil {
				return format.TranslateError(err)
			}
			return format.WritePlan(cmd.OutOrStdout(), opts.output, out)
		},
	}
	cmd.Flags().StringVarP(&opts.module, "module", "m", "", "module id (kebab-case)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "markdown", "output format: markdown|json|raw")
	_ = cmd.MarkFlagRequired("module")
	return cmd
}
