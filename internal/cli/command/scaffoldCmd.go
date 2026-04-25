package command

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/scaffolduc"
	scaffoldvo "github.com/jitctx/jitctx/internal/domain/vo/scaffold"
)

type scaffoldOpts struct {
	feature string
	file    string
	format  string
}

func NewScaffoldCmd(uc scaffolduc.UseCase, workDir, plansDir string, logger *slog.Logger) *cobra.Command {
	var opts scaffoldOpts
	cmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Generate production Java source files from a feature spec",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			modes := 0
			if opts.feature != "" {
				modes++
			}
			if opts.file != "" {
				modes++
			}
			if modes == 0 {
				return errors.New("one of --feature or --file is required")
			}
			if modes > 1 {
				return errors.New("--feature and --file are mutually exclusive")
			}
			switch opts.format {
			case "", "text", "json":
			default:
				return fmt.Errorf("--format must be text or json, got %q", opts.format)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := uc.Execute(cmd.Context(), scaffoldvo.ScaffoldInput{
				Feature:  opts.feature,
				FilePath: opts.file,
				BaseDir:  workDir,
				PlansDir: plansDir,
			})
			if err != nil {
				return format.TranslateError(err)
			}
			logger.Info("scaffold complete",
				slog.String("feature", out.Feature),
				slog.Int("count", len(out.WrittenPaths)))
			if opts.format == "json" {
				return format.WriteScaffoldJSON(cmd.OutOrStdout(), out)
			}
			return format.WriteScaffoldText(cmd.OutOrStdout(), out)
		},
	}
	cmd.Flags().StringVar(&opts.feature, "feature", "", "feature name (mutually exclusive with --file)")
	cmd.Flags().StringVar(&opts.file, "file", "", "explicit spec path (mutually exclusive with --feature)")
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text|json")
	return cmd
}
