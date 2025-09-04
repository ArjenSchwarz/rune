package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

var addFrontMatterCmd = &cobra.Command{
	Use:   "add-frontmatter [file] [flags]",
	Short: "Add front matter content to a task file",
	Long: `Add or merge front matter content (references and metadata) to an existing task file.

This command allows you to add references and metadata to a markdown task file.
If the file already has front matter, new content will be merged with existing content.

Examples:
  # Add references to a file
  rune add-frontmatter tasks.md --reference doc.md --reference spec.md
  
  # Add metadata to a file
  rune add-frontmatter tasks.md --meta "author:John" --meta "version:1.0"
  
  # Add both references and metadata
  rune add-frontmatter tasks.md --reference readme.md --meta "status:draft"
  
  # Preview changes without applying them
  rune add-frontmatter tasks.md --reference doc.md --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runAddFrontMatter,
}

var (
	addFMReferences []string
	addFMMetadata   []string
)

func init() {
	rootCmd.AddCommand(addFrontMatterCmd)
	addFrontMatterCmd.Flags().StringSliceVar(&addFMReferences, "reference", []string{}, "add reference file (can be used multiple times)")
	addFrontMatterCmd.Flags().StringSliceVar(&addFMMetadata, "meta", []string{}, "add metadata as key:value (can be used multiple times)")
}

func runAddFrontMatter(cmd *cobra.Command, args []string) error {
	filename := args[0]

	// Validate that at least one flag is provided
	if len(addFMReferences) == 0 && len(addFMMetadata) == 0 {
		return fmt.Errorf("at least one --reference or --meta flag must be provided")
	}

	// Validate file extension
	if !strings.HasSuffix(filename, ".md") {
		return fmt.Errorf("only .md files are supported")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", filename)
	}

	// Load existing TaskList from file
	tl, err := task.ParseFile(filename)
	if err != nil {
		return fmt.Errorf("failed to load file: %w", err)
	}

	// Parse metadata flags if provided
	var parsedMeta map[string]string
	if len(addFMMetadata) > 0 {
		parsedMeta, err = task.ParseMetadataFlags(addFMMetadata)
		if err != nil {
			return fmt.Errorf("invalid metadata format: %w", err)
		}
	}

	// Store original counts for feedback
	originalRefCount := 0
	if tl.FrontMatter != nil {
		originalRefCount = len(tl.FrontMatter.References)
	}

	// Add front matter content using the TaskList method
	err = tl.AddFrontMatterContent(addFMReferences, parsedMeta)
	if err != nil {
		return fmt.Errorf("failed to add front matter: %w", err)
	}

	// Calculate changes for feedback
	newRefCount := 0
	if tl.FrontMatter != nil {
		newRefCount = len(tl.FrontMatter.References) - originalRefCount
	}

	// Dry run mode - show what would be added
	if dryRun {
		fmt.Printf("Would update file: %s\n", filename)
		if newRefCount > 0 {
			fmt.Printf("Would add %d reference(s)\n", newRefCount)
			for _, ref := range addFMReferences {
				fmt.Printf("  - %s\n", ref)
			}
		}
		if len(parsedMeta) > 0 {
			fmt.Printf("Would merge %d metadata field(s)\n", len(parsedMeta))
			for key, value := range parsedMeta {
				fmt.Printf("  - %s: %s\n", key, value)
			}
		}
		fmt.Printf("\nResulting front matter:\n")
		// Show the resulting front matter
		if tl.FrontMatter != nil {
			// Use SerializeWithFrontMatter with empty content to get just the front matter
			fullContent := task.SerializeWithFrontMatter(tl.FrontMatter, "")
			// The resulting string will have the front matter with delimiters
			fmt.Print(fullContent)
		}
		return nil
	}

	// Write the file atomically (WriteFile already uses atomic write)
	if err := tl.WriteFile(filename); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Provide success feedback
	if verbose {
		fmt.Printf("Successfully updated: %s\n", filename)
		if newRefCount > 0 {
			fmt.Printf("Added %d reference(s):\n", newRefCount)
			for _, ref := range addFMReferences {
				fmt.Printf("  - %s\n", ref)
			}
		}
		if len(parsedMeta) > 0 {
			fmt.Printf("Merged %d metadata field(s):\n", len(parsedMeta))
			for key, value := range parsedMeta {
				fmt.Printf("  - %s: %s\n", key, value)
			}
		}
		// Show file stats
		if info, err := os.Stat(filename); err == nil {
			fmt.Printf("File size: %d bytes\n", info.Size())
		}
	} else {
		fmt.Printf("Updated: %s\n", filename)
		if newRefCount > 0 {
			fmt.Printf("  Added %d reference(s)\n", newRefCount)
		}
		if len(parsedMeta) > 0 {
			fmt.Printf("  Merged %d metadata field(s)\n", len(parsedMeta))
		}
	}

	return nil
}
