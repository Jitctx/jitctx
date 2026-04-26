package command

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/profileinituc"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

type profileInitOpts struct {
	workDir     string
	profilesDir string
}

// NewProfileInitCmd constructs the "profile init <name>" subcommand.
// Positional arg: the bundled profile name (required).
func NewProfileInitCmd(uc profileinituc.UseCase, defaultProfilesDir string, _ *slog.Logger) *cobra.Command {
	var opts profileInitOpts

	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Extract a bundled profile into the project's profiles directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// ProfilesDir defaults to cfg.ProfilesDir (passed in via
			// defaultProfilesDir). The --profiles-dir flag overrides.
			profilesDir := opts.profilesDir
			if profilesDir == "" {
				profilesDir = defaultProfilesDir
			}

			out, err := uc.Execute(cmd.Context(), profilevo.ProfileInitInput{
				Name:        name,
				WorkDir:     opts.workDir,
				ProfilesDir: profilesDir,
			})
			if err != nil {
				return format.TranslateError(err)
			}
			// stdout is the only success surface — short, machine-friendly line.
			_, perr := fmt.Fprintf(cmd.OutOrStdout(),
				"Initialised profile %q at %s (%d files)\n",
				out.Name, out.TargetDir, out.FilesWritten)
			return perr
		},
	}

	cmd.Flags().StringVar(&opts.workDir, "dir", ".", "project root (default \".\")")
	cmd.Flags().StringVar(&opts.profilesDir, "profiles-dir", "",
		"profiles directory relative to --dir (default from config: .jitctx/profiles)")
	return cmd
}
