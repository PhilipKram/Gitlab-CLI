package main

import (
	"fmt"
	"os"

	"github.com/PhilipKram/gitlab-cli/cmd"
)

// version is set at build time via ldflags
var version = "dev"

func main() {
	rootCmd := cmd.NewRootCmd(version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
