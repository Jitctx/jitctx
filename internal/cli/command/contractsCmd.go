package command

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/contractsuc"
	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
)

type contractsOpts struct {
	forPath string
	feature string
	file    string
	format  string
}

func NewContractsCmd(uc contractsuc.UseCase, workDir, plansDir string, _ *slog.Logger) *cobra.Command {
	var opts contractsOpts
	cmd := &cobra.Command{
		Use:   "contracts",
		Short: "Emit the contract slice required to implement a target file",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if opts.forPath == "" {
				return errors.New("--for is required")
			}
			if opts.feature != "" && opts.file != "" {
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
			out, err := uc.Execute(cmd.Context(), contractsvo.ExtractContractsInput{
				TargetFile: opts.forPath,
				Feature:    opts.feature,
				FilePath:   opts.file,
				BaseDir:    workDir,
				PlansDir:   plansDir,
			})
			if err != nil {
				return format.TranslateError(err)
			}
			if opts.format == "json" {
				return format.WriteContractsJSON(cmd.OutOrStdout(), out)
			}
			return format.WriteContractsText(cmd.OutOrStdout(), out)
		},
	}
	cmd.Flags().StringVar(&opts.forPath, "for", "", "target file path (required)")
	cmd.Flags().StringVar(&opts.feature, "feature", "", "feature name (mutually exclusive with --file)")
	cmd.Flags().StringVar(&opts.file, "file", "", "explicit spec path (mutually exclusive with --feature)")
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text|json")
	return cmd
}
