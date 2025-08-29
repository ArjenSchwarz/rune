package cmd

import (
	"fmt"
	"os"

	"github.com/ArjenSchwarz/go-tasks/internal/task"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [file] --title [title]",
	Short: "Create a new task file",
	Long: `Create a new task markdown file with the specified title.

The file will be initialized with proper markdown structure and formatting.
If the file already exists, this command will fail to prevent accidental overwrites.`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

var createTitle string

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&createTitle, "title", "t", "", "title for the task list")
	createCmd.MarkFlagRequired("title")
}

func runCreate(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file %s already exists", filename)
	}

	// Create new task list
	tl := task.NewTaskList(createTitle)

	// Dry run mode - just show what would be created
	if dryRun {
		fmt.Printf("Would create file: %s\n", filename)
		fmt.Printf("Title: %s\n", createTitle)
		fmt.Printf("\nContent preview:\n")
		content := task.RenderMarkdown(tl)
		fmt.Print(string(content))
		return nil
	}

	// Write the file
	if err := tl.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	if verbose {
		fmt.Printf("Successfully created task file: %s\n", filename)
		fmt.Printf("Title: %s\n", createTitle)

		// Show file stats
		if info, err := os.Stat(filename); err == nil {
			fmt.Printf("File size: %d bytes\n", info.Size())
		}
	} else {
		fmt.Printf("Created: %s\n", filename)
	}

	return nil
}
