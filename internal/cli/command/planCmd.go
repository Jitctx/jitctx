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
	newFeature string
	module     string
	feature    string
	file       string
	format     string
}

func NewPlanCmd(
	layers planuc.UseCase,
	newTpl plannewuc.UseCase,
	workDir string,
	plansDir string,
	logger *slog.Logger,
) *cobra.Command {
	var opts planOpts
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show the parallel execution plan for a feature spec",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			modes := 0
			if opts.newFeature != "" {
				modes++
			}
			if opts.feature != "" {
				modes++
			}
			if opts.file != "" {
				modes++
			}
			if modes == 0 {
				return errors.New("one of --new, --feature, --file is required")
			}
			if modes > 1 {
				return errors.New("--new, --feature, --file are mutually exclusive")
			}
			if opts.newFeature != "" && opts.module == "" {
				return errors.New("--module is required with --new")
			}
			switch opts.format {
			case "", "text", "json":
			default:
				return fmt.Errorf("--format must be text or json, got %q", opts.format)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// --new mode (UNCHANGED from US-002)
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

			// --feature / --file mode (NEW for US-003)
			out, err := layers.Execute(cmd.Context(), planvo.LayersInput{
				Feature:  opts.feature,
				FilePath: opts.file,
				BaseDir:  workDir,
				PlansDir: plansDir,
			})
			if err != nil {
				return format.TranslateError(err)
			}
			if opts.format == "json" {
				return format.WriteLayersJSON(cmd.OutOrStdout(), out)
			}
			return format.WriteLayersText(cmd.OutOrStdout(), out)
		},
	}
	cmd.Flags().StringVar(&opts.newFeature, "new", "", "feature name; toggles new-template mode")
	cmd.Flags().StringVarP(&opts.module, "module", "m", "", "module id (required when --new is set)")
	cmd.Flags().StringVar(&opts.feature, "feature", "", "feature name for layers mode (mutually exclusive with --new and --file)")
	cmd.Flags().StringVar(&opts.file, "file", "", "explicit spec path for layers mode (mutually exclusive with --new and --feature)")
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format for layers mode: text|json")
	_ = logger
	return cmd
}
