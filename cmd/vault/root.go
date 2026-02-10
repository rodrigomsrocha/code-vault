package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "vault",
	Short: "Code vault - A pastebin for serious developers",
	Long: `Code Vault is a CLI tool for storing and managing code snippets,
configurations, and any text-based files with version control,
tags, and expiration support.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {}
