package command

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/queryuc"
)

func NewListCmd(_ queryuc.UseCase, _ *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [modules|tags]",
		Short: "List modules or tags discovered in the manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "modules", "tags":
				return format.TranslateError(fmt.Errorf("list %s: not implemented", args[0]))
			default:
				return fmt.Errorf("unknown list target: %q (expected modules|tags)", args[0])
			}
		},
	}
	return cmd
}
