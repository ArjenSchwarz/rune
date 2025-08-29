package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool
	format  string
	dryRun  bool

	// Version information
	version = "0.1.0-dev"

	rootCmd = &cobra.Command{
		Use:   "go-tasks",
		Short: "A CLI tool for managing hierarchical markdown task lists",
		Long: `Go-Tasks is a command-line tool designed specifically for AI agents
to create and manage hierarchical markdown task lists with consistent formatting.

This tool provides:
- CRUD operations on hierarchical task structures
- Standardized markdown file format
- JSON API for batch operations  
- Query and search capabilities
- Multiple output formats`,
		Version: version,
		// Uncomment below to have it run the completion command
		// CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table", "output format (table, markdown, json)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying them")

	// Set custom version template
	rootCmd.SetVersionTemplate("go-tasks version {{.Version}}\n")
}
