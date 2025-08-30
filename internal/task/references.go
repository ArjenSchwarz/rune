package task

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontMatter represents the YAML front matter in task files
type FrontMatter struct {
	References []string       `yaml:"references,omitempty"`
	Metadata   map[string]any `yaml:"metadata,omitempty"`
}

// ParseFrontMatter extracts YAML front matter from markdown content
// Returns the parsed front matter, remaining content, and any error
func ParseFrontMatter(content string) (*FrontMatter, string, error) {
	// Check if content starts with front matter delimiter
	if !strings.HasPrefix(content, "---\n") {
		// No front matter present - return empty front matter and original content
		return &FrontMatter{}, content, nil
	}

	// Find the end of front matter - look for closing "---" on its own line
	searchStart := 4 // Skip the opening "---\n"

	// Look for "\n---\n" or "---\n" at the beginning of the remaining content
	var actualEndIndex int
	var remainingContent string

	if strings.HasPrefix(content[searchStart:], "---\n") {
		// Closing delimiter right after opening delimiter (empty front matter)
		actualEndIndex = searchStart
		remainingContent = content[searchStart+4:]
	} else {
		// Look for "\n---\n" pattern
		endPattern := "\n---\n"
		endIndex := strings.Index(content[searchStart:], endPattern)
		if endIndex == -1 {
			return nil, content, fmt.Errorf("unclosed front matter block")
		}
		actualEndIndex = searchStart + endIndex
		remainingContent = content[actualEndIndex+len(endPattern):]
	}

	// Extract front matter YAML (between opening and closing delimiters)
	frontMatterYAML := content[4:actualEndIndex]

	// Parse YAML
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(frontMatterYAML), &fm); err != nil {
		return nil, content, fmt.Errorf("parsing front matter: %w", err)
	}

	// No validation of reference paths per design decision - all paths allowed

	return &fm, remainingContent, nil
}

// SerializeWithFrontMatter combines front matter and content
func SerializeWithFrontMatter(fm *FrontMatter, content string) string {
	// If no front matter data, return content as-is
	if fm == nil || (len(fm.References) == 0 && len(fm.Metadata) == 0) {
		return content
	}

	var builder strings.Builder

	// Write opening front matter delimiter
	builder.WriteString("---\n")

	// Marshal front matter to YAML
	yamlData, err := yaml.Marshal(fm)
	if err != nil {
		// If marshaling fails, return content without front matter
		return content
	}
	builder.Write(yamlData)

	// Write closing front matter delimiter
	builder.WriteString("---\n")
	builder.WriteString(content)

	return builder.String()
}
