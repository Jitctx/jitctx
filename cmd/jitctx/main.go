package main

import (
	"os"

	"github.com/jitctx/jitctx/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:]))
}
