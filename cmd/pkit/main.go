package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/DotNaos/project-toolkit/internal/cli"
	"github.com/DotNaos/project-toolkit/internal/exitcode"
	"github.com/DotNaos/project-toolkit/internal/nodebridge"
)

func main() {
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		var exitErr *nodebridge.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}

		var codedErr *exitcode.Error
		if errors.As(err, &codedErr) {
			os.Exit(codedErr.Code)
		}

		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
