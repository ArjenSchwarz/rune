package task

import (
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"strings"
)

var (
	// Valid YAML key pattern: starts with letter or underscore, followed by letters, numbers, or underscores
	yamlKeyPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// ParseMetadataFlags converts "key:value" strings to map[string]any
// Multiple values for the same key create arrays
// Only flat key:value pairs are supported (no nested keys)
func ParseMetadataFlags(flags []string) (map[string]any, error) {
	result := make(map[string]any)

	for _, flag := range flags {
		if flag == "" {
			return nil, fmt.Errorf("empty metadata flag")
		}

		// Split on first colon only to allow colons in values
		colonIndex := strings.Index(flag, ":")
		if colonIndex == -1 {
			return nil, fmt.Errorf("invalid metadata format: %s (expected key:value)", flag)
		}

		key := flag[:colonIndex]
		value := flag[colonIndex+1:]

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
			result[key] = appendToValue(existing, value)
		} else {
			result[key] = value
		}
	}

	return result, nil
}

// appendToValue appends a new value to an existing value, creating an array if needed
func appendToValue(existing any, newValue string) any {
	switch v := existing.(type) {
	case string:
		// Convert to array with both values
		return []string{v, newValue}
	case []string:
		// Append to existing array
		return append(v, newValue)
	default:
		// Shouldn't happen with our usage, but handle gracefully
		return []any{existing, newValue}
	}
}

// MergeFrontMatter merges two FrontMatter structures
// References are appended without deduplication
// Metadata is merged with type-aware logic
func MergeFrontMatter(existing, new *FrontMatter) (*FrontMatter, error) {
	result := &FrontMatter{
		References: []string{},
		Metadata:   make(map[string]any),
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

		// Merge metadata
		for key, newValue := range new.Metadata {
			if existingValue, exists := result.Metadata[key]; exists {
				merged, err := mergeValues(existingValue, newValue)
				if err != nil {
					return nil, fmt.Errorf("merge conflict for key %s: %w", key, err)
				}
				result.Metadata[key] = merged
			} else {
				result.Metadata[key] = newValue
			}
		}
	}

	return result, nil
}

// mergeValues merges two values with type-aware logic
// Only supports flat values - no nested maps
func mergeValues(existing, new any) (any, error) {
	// Check if both are the same type
	switch existingVal := existing.(type) {
	case string:
		if _, ok := new.(string); ok {
			// Scalar replacement
			return new, nil
		}
		return nil, fmt.Errorf("type conflict: cannot merge string with %T", new)

	case []string:
		if newSlice, ok := new.([]string); ok {
			// Append arrays
			return append(existingVal, newSlice...), nil
		}
		// Try to convert new value to []string if it's []any
		if newAnySlice, ok := new.([]any); ok {
			result := existingVal
			for _, v := range newAnySlice {
				if str, ok := v.(string); ok {
					result = append(result, str)
				} else {
					return nil, fmt.Errorf("type conflict: array contains non-string value")
				}
			}
			return result, nil
		}
		return nil, fmt.Errorf("type conflict: cannot merge []string with %T", new)

	case []any:
		if newSlice, ok := new.([]any); ok {
			// Append arrays
			return append(existingVal, newSlice...), nil
		}
		// Try to append other slice types
		if newStrSlice, ok := new.([]string); ok {
			result := existingVal
			for _, v := range newStrSlice {
				result = append(result, v)
			}
			return result, nil
		}
		return nil, fmt.Errorf("type conflict: cannot merge []any with %T", new)

	case map[string]any:
		// Nested maps are not supported
		return nil, fmt.Errorf("nested maps are not supported")

	default:
		// For other types (int, float, bool, etc.), replace with new value if same type
		if reflect.TypeOf(existing) == reflect.TypeOf(new) {
			return new, nil
		}
		return nil, fmt.Errorf("type conflict: cannot merge %T with %T", existing, new)
	}
}
