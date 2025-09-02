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
If the file already exists, this command will fail to prevent accidental overwrites.

Optional front matter can be added using --reference and --meta flags:
  --reference: Add reference files (can be used multiple times)
  --meta: Add metadata in key:value format (can be used multiple times)`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

var (
	createTitle      string
	createReferences []string
	createMetadata   []string
)

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&createTitle, "title", "t", "", "title for the task list")
	createCmd.MarkFlagRequired("title")
	createCmd.Flags().StringSliceVar(&createReferences, "reference", []string{}, "add reference file (can be used multiple times)")
	createCmd.Flags().StringSliceVar(&createMetadata, "meta", []string{}, "add metadata as key:value (can be used multiple times)")
}

func runCreate(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file %s already exists", filename)
	}

	// Build FrontMatter if references or metadata are provided
	var fm *task.FrontMatter
	if len(createReferences) > 0 || len(createMetadata) > 0 {
		fm = &task.FrontMatter{
			References: createReferences,
		}

		// Parse and add metadata if provided
		if len(createMetadata) > 0 {
			parsedMeta, err := task.ParseMetadataFlags(createMetadata)
			if err != nil {
				return fmt.Errorf("invalid metadata format: %w", err)
			}
			fm.Metadata = parsedMeta
		}
	}

	// Create new task list with optional front matter
	tl := task.NewTaskList(createTitle, fm)

	// Dry run mode - just show what would be created
	if dryRun {
		fmt.Printf("Would create file: %s\n", filename)
		fmt.Printf("Title: %s\n", createTitle)
		if fm != nil {
			if len(fm.References) > 0 {
				fmt.Printf("References: %d\n", len(fm.References))
			}
			if len(fm.Metadata) > 0 {
				fmt.Printf("Metadata fields: %d\n", len(fm.Metadata))
			}
		}
		fmt.Printf("\nContent preview:\n")
		// Generate the full content including front matter
		var content []byte
		if fm != nil && (len(fm.References) > 0 || len(fm.Metadata) > 0) {
			markdownContent := task.RenderMarkdown(tl)
			fullContent := task.SerializeWithFrontMatter(fm, string(markdownContent))
			content = []byte(fullContent)
		} else {
			content = task.RenderMarkdown(tl)
		}
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
		if fm != nil {
			if len(fm.References) > 0 {
				fmt.Printf("Added %d reference(s)\n", len(fm.References))
			}
			if len(fm.Metadata) > 0 {
				fmt.Printf("Added %d metadata field(s)\n", len(fm.Metadata))
			}
		}

		// Show file stats
		if info, err := os.Stat(filename); err == nil {
			fmt.Printf("File size: %d bytes\n", info.Size())
		}
	} else {
		fmt.Printf("Created: %s\n", filename)
		if fm != nil {
			if len(fm.References) > 0 {
				fmt.Printf("Added %d reference(s)\n", len(fm.References))
			}
			if len(fm.Metadata) > 0 {
				fmt.Printf("Added %d metadata field(s)\n", len(fm.Metadata))
			}
		}
	}

	return nil
}
