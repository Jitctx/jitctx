package command

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/contractsuc"
	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
)

type contractsOpts struct {
	forPath string
	output  string
}

func NewContractsCmd(uc contractsuc.UseCase, _ *slog.Logger) *cobra.Command {
	var opts contractsOpts
	cmd := &cobra.Command{
		Use:   "contracts",
		Short: "Extract interface/contract skeletons for a target file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := uc.Execute(cmd.Context(), contractsvo.ExtractContractsInput{
				TargetFile: opts.forPath,
			})
			if err != nil {
				return format.TranslateError(err)
			}
			return format.WriteContracts(cmd.OutOrStdout(), opts.output, out)
		},
	}
	cmd.Flags().StringVar(&opts.forPath, "for", "", "target file path for which contracts are required")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "markdown", "output format: markdown|json|raw")
	_ = cmd.MarkFlagRequired("for")
	return cmd
}
