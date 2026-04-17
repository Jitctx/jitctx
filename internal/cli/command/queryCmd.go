package command

import (
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/queryuc"
	"github.com/jitctx/jitctx/internal/domain/vo"
	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

type queryOpts struct {
	workDir string
	module  string
	tags    []string
	types   []string
	file    string
	budget  int
	output  string
}

// parseArtifactTypes converts raw --type values into domain ArtifactTypes.
// Unknown values are logged via slog.Warn and dropped (EP01RF-008).
func parseArtifactTypes(raw []string, logger *slog.Logger) []vo.ArtifactType {
	out := make([]vo.ArtifactType, 0, len(raw))
	for _, t := range raw {
		trimmed := strings.TrimSpace(t)
		at := vo.ArtifactType(trimmed)
		if err := at.Validate(); err != nil {
			logger.Warn("ignoring unknown --type value",
				"value", trimmed,
				"accepted", "guidelines|requirements|scenarios|contracts",
			)
			continue
		}
		out = append(out, at)
	}
	return out
}

func NewQueryCmd(uc queryuc.UseCase, logger *slog.Logger) *cobra.Command {
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
			types := parseArtifactTypes(opts.types, logger)
			out, err := uc.Execute(cmd.Context(), queryvo.QueryContextInput{
				WorkDir:  opts.workDir,
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
	cmd.Flags().StringVar(&opts.workDir, "dir", ".", "project root to query")
	cmd.Flags().StringVar(&opts.workDir, "path", ".", "project root to query (alias for --dir)")
	cmd.Flags().StringVarP(&opts.module, "module", "m", "", "module id (kebab-case)")
	cmd.Flags().StringSliceVar(&opts.tags, "tags", nil, "filter by tags (comma-separated; OR within the flag)")
	cmd.Flags().StringSliceVar(&opts.types, "type", nil, "artifact types: guidelines|requirements|scenarios|contracts (comma-separated; unknowns warn and are ignored)")
	cmd.Flags().StringVar(&opts.file, "file", "", "infer module from this source file")
	cmd.Flags().IntVar(&opts.budget, "budget", 0, "token budget (0 = unlimited)")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "markdown", "output format: markdown|json|raw|yaml")
	cmd.Flags().StringVar(&opts.output, "format", "markdown", "output format: markdown|json|raw|yaml (alias of --output)")
	_ = cmd.MarkFlagRequired("module")
	return cmd
}
