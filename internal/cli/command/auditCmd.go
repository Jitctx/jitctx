package command

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/audituc"
	auditvo "github.com/jitctx/jitctx/internal/domain/vo/audit"
)

type auditOpts struct {
	workDir      string
	profile      string
	manifestPath string
}

// NewAuditCmd constructs the audit subcommand.
// uc is the audit use case; it is injected by the composition root via Deps.Audit.
func NewAuditCmd(uc audituc.UseCase, _ *slog.Logger) *cobra.Command {
	var opts auditOpts

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit existing source code against the active profile rules",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			// Mirror scanCmd's manifest-path resolution and SEC-001 escape check.
			if !cmd.Flags().Changed("manifest") {
				opts.manifestPath = filepath.Join(opts.workDir, "project-state.yaml")
			} else if !filepath.IsAbs(opts.manifestPath) {
				dirAbs, err := filepath.Abs(opts.workDir)
				if err != nil {
					return fmt.Errorf("resolve --dir: %w", err)
				}
				resolved := filepath.Clean(filepath.Join(dirAbs, opts.manifestPath))
				if !strings.HasPrefix(resolved, dirAbs+string(filepath.Separator)) {
					return fmt.Errorf("--manifest %q escapes project directory", opts.manifestPath)
				}
				opts.manifestPath = resolved
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := uc.Execute(cmd.Context(), auditvo.AuditProjectInput{
				WorkDir:      opts.workDir,
				ManifestPath: opts.manifestPath,
				ProfileName:  opts.profile,
			})
			if err != nil {
				return format.TranslateError(err)
			}
			return format.WriteAuditReport(cmd.OutOrStdout(), out)
		},
	}

	// --dir and --path are aliases for the same backing field.
	cmd.Flags().StringVar(&opts.workDir, "dir", ".", "project root to audit")
	cmd.Flags().StringVar(&opts.workDir, "path", ".", "project root to audit (alias for --dir)")
	cmd.Flags().StringVar(&opts.profile, "profile", "", "framework profile name (auto-detected if empty)")
	cmd.Flags().StringVar(&opts.manifestPath, "manifest", "project-state.yaml", "manifest input path")
	// NOTE: no -o / --output flag — markdown is the only output format per EP03RF-007.
	return cmd
}
