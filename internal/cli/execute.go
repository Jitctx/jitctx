package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jitctx/jitctx/internal/config"
)

func Execute(args []string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		return 2
	}

	logger := config.NewLogger(cfg)
	deps := Wire(cfg, logger)

	root := NewRootCmd(deps)
	root.SetArgs(args)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	if err := root.ExecuteContext(context.Background()); err != nil {
		if errors.Is(err, context.Canceled) {
			return 130
		}
		return 1
	}
	return 0
}
