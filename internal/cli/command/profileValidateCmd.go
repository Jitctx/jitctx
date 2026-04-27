package command

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/profilevalidateuc"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// NewProfileValidateCmd constructs the "profile validate <path>" cobra
// subcommand. Positional arg: path to a profile directory (required).
func NewProfileValidateCmd(uc profilevalidateuc.UseCase, _ *slog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "validate <path>",
		Short: "Validate a profile directory for structural and logical errors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			out, err := uc.Execute(cmd.Context(), profilevo.ValidateProfileInput{Path: path})

			// Print warnings to stderr REGARDLESS of error state, so
			// scenario 4 ("warning, may still exit 0") is satisfied.
			for _, w := range out.Warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), w.Message)
			}
			if err != nil {
				return format.TranslateError(err)
			}
			_, perr := fmt.Fprintln(cmd.OutOrStdout(), "Profile valid")
			return perr
		},
	}
}
