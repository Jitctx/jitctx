package command

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/scanuc"
	scanvo "github.com/jitctx/jitctx/internal/domain/vo/scan"
)

// ScanUseCaseFactory creates a scan use case configured for the given manifest path.
// The manifest path is resolved at command parse time (in PreRunE).
type ScanUseCaseFactory func(manifestPath string) scanuc.UseCase

type scanOpts struct {
	workDir      string
	profile      string
	manifestPath string
	output       string
}

// NewScanCmd constructs the scan subcommand.
// factory is called in PreRunE with the resolved manifest path to get a properly configured use case.
func NewScanCmd(factory ScanUseCaseFactory, _ *slog.Logger) *cobra.Command {
	var opts scanOpts
	var uc scanuc.UseCase

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan the project and emit project-state.yaml",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			// Resolve manifest path against workDir when not explicitly provided.
			if !cmd.Flags().Changed("manifest") {
				opts.manifestPath = filepath.Join(opts.workDir, "project-state.yaml")
			} else if !filepath.IsAbs(opts.manifestPath) {
				// SEC-001: ensure relative --manifest path stays under --dir.
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
			// Create use case with the resolved manifest path.
			uc = factory(opts.manifestPath)
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := uc.Execute(cmd.Context(), scanvo.ScanProjectInput{
				WorkDir:      opts.workDir,
				ProfileName:  opts.profile,
				ManifestPath: opts.manifestPath,
			})
			if err != nil {
				return format.TranslateError(err)
			}
			return format.WriteScanReport(cmd.OutOrStdout(), opts.output, out)
		},
	}

	// --dir and --path are aliases for the same backing field.
	cmd.Flags().StringVar(&opts.workDir, "dir", ".", "project root to scan")
	cmd.Flags().StringVar(&opts.workDir, "path", ".", "project root to scan (alias for --dir)")
	cmd.Flags().StringVar(&opts.profile, "profile", "", "framework profile name (auto-detected if empty)")
	cmd.Flags().StringVar(&opts.manifestPath, "manifest", "project-state.yaml", "manifest output path")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "markdown", "output format: markdown|json")
	return cmd
}
