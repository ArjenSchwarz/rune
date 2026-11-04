package task

import (
	"fmt"
	"maps"
	"slices"
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

	// source is the parsed top-level mapping and parsed holds the values it
	// decoded to. A write reuses the source key/value nodes, and with them
	// their comments, for every key whose value has not changed. Neither is
	// ever modified, so copies of a FrontMatter may share them.
	source *yaml.Node
	parsed knownFrontMatter
	// comments holds the raw text of front matter that contains only
	// comments, which YAML parsing yields no node for.
	comments string
}

// clone returns a copy that shares no mutable state with fm.
func (fm *FrontMatter) clone() *FrontMatter {
	c := *fm
	c.References = slices.Clone(fm.References)
	c.Metadata = maps.Clone(fm.Metadata)
	c.Extra = slices.Clone(fm.Extra)
	return &c
}

// Top-level front matter keys that rune models.
const (
	referencesKey = "references"
	metadataKey   = "metadata"
)

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
	if value.Kind == yaml.MappingNode {
		fm.source = value
	}
	fm.parsed = knownFrontMatter{References: slices.Clone(known.References), Metadata: maps.Clone(known.Metadata)}
	fm.Extra = nil
	for i := 0; i+1 < len(value.Content); i += 2 {
		switch value.Content[i].Value {
		case referencesKey, metadataKey:
			continue
		}
		fm.Extra = append(fm.Extra, value.Content[i], value.Content[i+1])
	}
	return nil
}

// MarshalYAML writes the modelled keys followed by the unknown keys in their
// original order, so an alias is never written before its anchor. Parsed
// front matter is written from its source nodes instead, keeping comments.
func (fm FrontMatter) MarshalYAML() (any, error) {
	node := &yaml.Node{}
	if err := node.Encode(knownFrontMatter{References: fm.References, Metadata: fm.Metadata}); err != nil {
		return nil, err
	}
	if fm.source != nil {
		return fm.fromSource(node), nil
	}
	// An empty mapping encodes in flow style ({}); use block style so the
	// appended keys are written one per line.
	node.Style = 0
	node.Content = append(node.Content, fm.Extra...)
	return node, nil
}

// fromSource returns the source mapping with the references and metadata
// values replaced by their encoding in fresh only where they changed since
// parsing. Unchanged keys and the unknown keys keep their source nodes,
// comments included. A modelled key that is new is appended at the end.
func (fm FrontMatter) fromSource(fresh *yaml.Node) *yaml.Node {
	changed := map[string]bool{
		referencesKey: !slices.Equal(fm.References, fm.parsed.References),
		metadataKey:   !maps.Equal(fm.Metadata, fm.parsed.Metadata),
	}
	freshValue := func(key string) *yaml.Node {
		for i := 0; i+1 < len(fresh.Content); i += 2 {
			if fresh.Content[i].Value == key {
				return fresh.Content[i+1]
			}
		}
		return nil // empty values are omitted from fresh
	}

	out := *fm.source
	out.Content = nil
	seen := map[string]bool{}
	for i := 0; i+1 < len(fm.source.Content); i += 2 {
		key, value := fm.source.Content[i], fm.source.Content[i+1]
		if isChanged, modelled := changed[key.Value]; modelled {
			seen[key.Value] = true
			if isChanged {
				value = freshValue(key.Value)
				if value == nil {
					continue
				}
			}
		}
		out.Content = append(out.Content, key, value)
	}
	for _, key := range []string{referencesKey, metadataKey} {
		if value := freshValue(key); value != nil && !seen[key] {
			out.Content = append(out.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
		}
	}
	return &out
}

func joinComments(a, b string) string {
	if a == "" || b == "" {
		return a + b
	}
	return a + "\n\n" + b
}

// IsEmpty reports whether there is no front matter content to write.
func (fm *FrontMatter) IsEmpty() bool {
	return fm == nil || (!fm.hasYAML() && fm.comments == "")
}

// hasYAML reports whether there are YAML keys to write, including parsed
// keys whose values are empty but which may carry comments.
func (fm *FrontMatter) hasYAML() bool {
	return len(fm.References) > 0 || len(fm.Metadata) > 0 || len(fm.Extra) > 0 || fm.source != nil
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
	var doc yaml.Node
	err := yaml.Unmarshal([]byte(frontMatterYAML), &doc)
	if err == nil && len(doc.Content) > 0 {
		err = doc.Content[0].Decode(&fm)
	}
	if err != nil {
		return nil, content, fmt.Errorf("parsing front matter: %w", err)
	}
	if len(doc.Content) == 0 && strings.TrimSpace(frontMatterYAML) != "" {
		fm.comments = frontMatterYAML + "\n"
	}
	// A comment separated from the first or last key by a blank line belongs
	// to the document rather than the mapping; move it onto the mapping so
	// it is written back too.
	if fm.source != nil {
		fm.source.HeadComment = joinComments(doc.HeadComment, fm.source.HeadComment)
		fm.source.FootComment = joinComments(fm.source.FootComment, doc.FootComment)
	}

	// Reject unknown keys rune could not write back as a parsable document
	// (for example an alias of an anchor on a modelled key, which is
	// re-encoded without its anchor), before any command rewrites the file.
	if len(fm.Extra) > 0 {
		fresh := fm
		fresh.source = nil // check the encoding used once a modelled key changes
		out, err := yaml.Marshal(&fresh)
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

	// Comment-only front matter is written back verbatim, ahead of any keys
	// added since it was parsed.
	builder.WriteString(fm.comments)

	if fm.hasYAML() {
		// Marshal front matter to YAML
		yamlData, err := yaml.Marshal(fm)
		if err != nil {
			// If marshaling fails, return content without front matter
			return content
		}
		builder.Write(yamlData)
	}

	// Write closing front matter delimiter
	builder.WriteString("---\n")
	builder.WriteString(content)

	return builder.String()
}
