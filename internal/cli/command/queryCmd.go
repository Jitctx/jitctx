package command

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/queryuc"
	"github.com/jitctx/jitctx/internal/domain/vo"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

type queryOpts struct {
	module string
	tags   []string
	types  []string
	file   string
	budget int
	output string
}

func NewQueryCmd(uc queryuc.UseCase, _ *slog.Logger) *cobra.Command {
	var opts queryOpts
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Emit filtered context fragments to stdout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			budget, err := vo.NewTokenBudget(opts.budget)
			if err != nil {
				return format.TranslateError(err)
			}
			types := make([]vo.ArtifactType, 0, len(opts.types))
			for _, t := range opts.types {
				types = append(types, vo.ArtifactType(t))
			}
			out, err := uc.Execute(cmd.Context(), queryvo.QueryContextInput{
				Module:   opts.module,
				Tags:     opts.tags,
				Types:    types,
				FilePath: opts.file,
				Budget:   budget,
			})
			if err != nil {
				return format.TranslateError(err)
			}
			return format.WriteQueryResult(cmd.OutOrStdout(), opts.output, out)
		},
	}
	cmd.Flags().StringVarP(&opts.module, "module", "m", "", "module id (kebab-case)")
	cmd.Flags().StringSliceVar(&opts.tags, "tag", nil, "filter by tags")
	cmd.Flags().StringSliceVar(&opts.types, "type", nil, "artifact types: guidelines|requirements|scenarios|contracts")
	cmd.Flags().StringVar(&opts.file, "file", "", "infer module from this source file")
	cmd.Flags().IntVar(&opts.budget, "budget", 0, "token budget (0 = unlimited)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "markdown", "output format: markdown|json|raw")
	return cmd
}
