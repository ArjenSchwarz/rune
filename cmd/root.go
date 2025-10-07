package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool
	format  string
	dryRun  bool

	// Version information
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"

	rootCmd = &cobra.Command{
		Use:   "rune",
		Short: "A CLI tool for managing hierarchical markdown task lists",
		Long: `Rune is a command-line tool designed specifically for AI agents
to create and manage hierarchical markdown task lists with consistent formatting.

This tool provides:
- CRUD operations on hierarchical task structures
- Standardized markdown file format
- JSON API for batch operations
- Query and search capabilities
- Multiple output formats`,
		// Uncomment below to have it run the completion command
		// CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Only print error if not silenced by the command
		// Cobra already handles error printing for commands with SilenceErrors=false
		os.Exit(1)
	}
}

func init() {
	// Set version function to use current values (allows ldflags to override)
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("rune version {{.Version}}\n")

	// Global persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table", "output format (table, markdown, json)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying them")
}
