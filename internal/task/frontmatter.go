package task

import (
	"fmt"
	"maps"
	"regexp"
	"strings"
)

var (
	// Valid YAML key pattern: starts with letter or underscore, followed by letters, numbers, or underscores
	yamlKeyPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// ParseMetadataFlags converts "key:value" strings to map[string]string
// Multiple values for the same key are concatenated with commas
// Only flat key:value pairs are supported (no nested keys)
func ParseMetadataFlags(flags []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, flag := range flags {
		if flag == "" {
			return nil, fmt.Errorf("empty metadata flag")
		}

		// Split on first colon only to allow colons in values
		before, after, ok := strings.Cut(flag, ":")
		if !ok {
			return nil, fmt.Errorf("invalid metadata format: %s (expected key:value)", flag)
		}

		key := before
		value := after

		if key == "" {
			return nil, fmt.Errorf("empty metadata key in: %s", flag)
		}

		// Validate the key - no dots allowed
		if strings.Contains(key, ".") {
			return nil, fmt.Errorf("nested keys not supported: %s", key)
		}

		// Check for reserved YAML keys
		if key == "<<" || key == "&" || key == "*" {
			return nil, fmt.Errorf("reserved YAML key: %s", key)
		}

		// Validate key format
		if !yamlKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid key %q: must start with letter or underscore, followed by letters, numbers, or underscores", key)
		}

		// Simple key - check if it already exists
		if existing, exists := result[key]; exists {
			// Concatenate multiple values with comma separator
			result[key] = existing + "," + value
		} else {
			result[key] = value
		}
	}

	return result, nil
}

// MergeFrontMatter merges two FrontMatter structures
// References are appended without deduplication
// Metadata uses simple 'last wins' replacement strategy
func MergeFrontMatter(existing, new *FrontMatter) (*FrontMatter, error) {
	result := &FrontMatter{
		References: []string{},
		Metadata:   make(map[string]string),
	}

	// Handle nil inputs
	if existing == nil && new == nil {
		return result, nil
	}

	if existing != nil {
		// Copy existing references
		result.References = append(result.References, existing.References...)

		// Deep copy existing metadata
		maps.Copy(result.Metadata, existing.Metadata)
	}

	if new != nil {
		// Append new references (no deduplication per requirements)
		result.References = append(result.References, new.References...)

		// Simple replacement for metadata - last wins
		maps.Copy(result.Metadata, new.Metadata)
	}

	return result, nil
}
