package task

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontMatter represents the YAML front matter in task files
type FrontMatter struct {
	References []string          `yaml:"references,omitempty"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
	// Extra holds top-level keys rune does not model, as alternating key and
	// value nodes in source order. Keeping the YAML nodes (rather than decoded
	// Go values) preserves tags, anchors and aliases, so these keys survive a
	// parse/write round trip unchanged in meaning.
	Extra []*yaml.Node `yaml:"-" json:"-"`
}

// knownFrontMatter is the part of FrontMatter that rune models.
type knownFrontMatter struct {
	References []string          `yaml:"references,omitempty"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
}

// UnmarshalYAML decodes the modelled keys and keeps every other top-level
// key/value pair as YAML nodes.
func (fm *FrontMatter) UnmarshalYAML(value *yaml.Node) error {
	var known knownFrontMatter
	if err := value.Decode(&known); err != nil {
		return err
	}
	fm.References = known.References
	fm.Metadata = known.Metadata
	fm.Extra = nil
	for i := 0; i+1 < len(value.Content); i += 2 {
		switch value.Content[i].Value {
		case "references", "metadata":
			continue
		}
		fm.Extra = append(fm.Extra, value.Content[i], value.Content[i+1])
	}
	return nil
}

// MarshalYAML writes the modelled keys followed by the unknown keys in their
// original order, so an alias is never written before its anchor.
func (fm FrontMatter) MarshalYAML() (any, error) {
	node := &yaml.Node{}
	if err := node.Encode(knownFrontMatter{References: fm.References, Metadata: fm.Metadata}); err != nil {
		return nil, err
	}
	// An empty mapping encodes in flow style ({}); use block style so the
	// appended keys are written one per line.
	node.Style = 0
	node.Content = append(node.Content, fm.Extra...)
	return node, nil
}

// IsEmpty reports whether there is no front matter content to write.
func (fm *FrontMatter) IsEmpty() bool {
	return fm == nil || (len(fm.References) == 0 && len(fm.Metadata) == 0 && len(fm.Extra) == 0)
}

// ParseFrontMatter extracts YAML front matter from markdown content
// Returns the parsed front matter, remaining content, and any error
func ParseFrontMatter(content string) (*FrontMatter, string, error) {
	// Normalize CRLF to LF so delimiter matching works with Windows-style line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

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

	rest := content[searchStart:]
	switch {
	case strings.HasPrefix(rest, "---\n"):
		// Closing delimiter right after opening delimiter (empty front matter)
		actualEndIndex = searchStart
		remainingContent = content[searchStart+4:]
	case rest == "---":
		// Closing delimiter at EOF without trailing newline (empty front matter)
		actualEndIndex = searchStart
		remainingContent = ""
	default:
		// Look for "\n---\n" pattern, or "\n---" at end of content
		endPattern := "\n---\n"
		endIndex := strings.Index(rest, endPattern)
		if endIndex == -1 {
			// Check if content ends with "\n---" (no trailing newline)
			if strings.HasSuffix(rest, "\n---") {
				endIndex = len(rest) - 4 // position of the "\n"
				actualEndIndex = searchStart + endIndex
				remainingContent = ""
			} else {
				return nil, content, fmt.Errorf("unclosed front matter block")
			}
		} else {
			actualEndIndex = searchStart + endIndex
			remainingContent = content[actualEndIndex+len(endPattern):]
		}
	}

	// Extract front matter YAML (between opening and closing delimiters)
	frontMatterYAML := content[4:actualEndIndex]

	// Parse YAML
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(frontMatterYAML), &fm); err != nil {
		return nil, content, fmt.Errorf("parsing front matter: %w", err)
	}

	// Reject unknown keys rune could not write back as a parsable document
	// (for example an alias of an anchor on a modelled key, which is
	// re-encoded without its anchor), before any command rewrites the file.
	if len(fm.Extra) > 0 {
		out, err := yaml.Marshal(&fm)
		if err == nil {
			err = yaml.Unmarshal(out, &FrontMatter{})
		}
		if err != nil {
			return nil, content, fmt.Errorf("front matter cannot be preserved on write: %w", err)
		}
	}

	// No validation of reference paths per design decision - all paths allowed

	return &fm, remainingContent, nil
}

// SerializeWithFrontMatter combines front matter and content
func SerializeWithFrontMatter(fm *FrontMatter, content string) string {
	// If no front matter data, return content as-is
	if fm.IsEmpty() {
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
