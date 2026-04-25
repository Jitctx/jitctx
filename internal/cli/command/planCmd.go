package command

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/plannewuc"
	"github.com/jitctx/jitctx/internal/domain/usecase/planuc"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

type planOpts struct {
	module     string
	output     string
	newFeature string
}

func NewPlanCmd(
	legacy planuc.UseCase,
	newTpl plannewuc.UseCase,
	workDir string,
	plansDir string,
	logger *slog.Logger,
) *cobra.Command {
	var opts planOpts
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the parallel execution plan for a module",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if opts.module == "" {
				if opts.newFeature != "" {
					return errors.New("--module is required with --new")
				}
				return errors.New("--module is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.newFeature != "" {
				base := plansDir
				if base == "" {
					base = filepath.Join(workDir, "jitctx-plans")
				} else if !filepath.IsAbs(base) {
					base = filepath.Join(workDir, base)
				}
				out, err := newTpl.Execute(cmd.Context(), planvo.NewTemplateInput{
					Feature: opts.newFeature,
					Module:  opts.module,
					BaseDir: base,
				})
				if err != nil {
					return format.TranslateError(err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", out.Path)
				return nil
			}

			// legacy --module path — unchanged
			out, err := legacy.Execute(cmd.Context(), planvo.PlanModuleInput{Module: opts.module})
			if err != nil {
				return format.TranslateError(err)
			}
			return format.WritePlan(cmd.OutOrStdout(), opts.output, out)
		},
	}
	cmd.Flags().StringVarP(&opts.module, "module", "m", "", "module id (kebab-case)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "markdown", "output format: markdown|json|raw")
	cmd.Flags().StringVar(&opts.newFeature, "new", "", "feature name (kebab-case); generates a new spec template")
	_ = logger
	return cmd
}
