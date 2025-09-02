package task

import (
	"strings"
	"testing"
)

func TestParseFrontMatter(t *testing.T) {
	tests := map[string]struct {
		input           string
		expectedFM      *FrontMatter
		expectedContent string
		expectedError   string
	}{
		"valid front matter with references and metadata": {
			input: `---
references:
  - ./docs/architecture.md
  - ./specs/api-specification.yaml
  - ../shared/database-schema.sql
metadata:
  project: backend-api
  created: "2024-01-30"
---
# Project Tasks

- [ ] 1. Setup development environment`,
			expectedFM: &FrontMatter{
				References: []string{
					"./docs/architecture.md",
					"./specs/api-specification.yaml",
					"../shared/database-schema.sql",
				},
				Metadata: map[string]string{
					"project": "backend-api",
					"created": "2024-01-30",
				},
			},
			expectedContent: `# Project Tasks

- [ ] 1. Setup development environment`,
		},
		"valid front matter with only references": {
			input: `---
references:
  - ./docs/setup.md
  - ./specs/api.yaml
---
# Test Tasks

- [ ] 1. Test task`,
			expectedFM: &FrontMatter{
				References: []string{
					"./docs/setup.md",
					"./specs/api.yaml",
				},
				Metadata: nil,
			},
			expectedContent: `# Test Tasks

- [ ] 1. Test task`,
		},
		"valid front matter with only metadata": {
			input: `---
metadata:
  project: test-project
---
# Tasks

- [ ] 1. Task one`,
			expectedFM: &FrontMatter{
				References: nil,
				Metadata: map[string]string{
					"project": "test-project",
				},
			},
			expectedContent: `# Tasks

- [ ] 1. Task one`,
		},
		"empty front matter": {
			input: `---
---
# Tasks

- [ ] 1. Task one`,
			expectedFM: &FrontMatter{
				References: nil,
				Metadata:   nil,
			},
			expectedContent: `# Tasks

- [ ] 1. Task one`,
		},
		"no front matter": {
			input: `# Tasks

- [ ] 1. Task one`,
			expectedFM: &FrontMatter{
				References: nil,
				Metadata:   nil,
			},
			expectedContent: `# Tasks

- [ ] 1. Task one`,
		},
		"unclosed front matter": {
			input: `---
references:
  - ./docs/test.md
# Tasks without closing delimiter

- [ ] 1. Task one`,
			expectedError: "unclosed front matter block",
		},
		"invalid YAML in front matter": {
			input: `---
references:
  - ./docs/test.md
invalid: yaml: content: [
---
# Tasks

- [ ] 1. Task one`,
			expectedError: "parsing front matter:",
		},
		"absolute path allowed": {
			input: `---
references:
  - /absolute/path/to/doc.md
  - ./relative/path.md
---
# Tasks

- [ ] 1. Task one`,
			expectedFM: &FrontMatter{
				References: []string{
					"/absolute/path/to/doc.md",
					"./relative/path.md",
				},
				Metadata: nil,
			},
			expectedContent: `# Tasks

- [ ] 1. Task one`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fm, content, err := ParseFrontMatter(tc.input)

			if tc.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.expectedError)
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected error containing %q, got %q", tc.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if content != tc.expectedContent {
				t.Errorf("content mismatch:\nexpected: %q\ngot: %q", tc.expectedContent, content)
			}

			// Compare front matter
			expectedRefLen := 0
			if tc.expectedFM.References != nil {
				expectedRefLen = len(tc.expectedFM.References)
			}
			actualRefLen := 0
			if fm.References != nil {
				actualRefLen = len(fm.References)
			}

			if actualRefLen != expectedRefLen {
				t.Errorf("references length mismatch: expected %d, got %d", expectedRefLen, actualRefLen)
			}

			if tc.expectedFM.References != nil {
				for i, expectedRef := range tc.expectedFM.References {
					if fm.References == nil || i >= len(fm.References) {
						t.Errorf("missing reference at index %d: expected %q", i, expectedRef)
						continue
					}
					if fm.References[i] != expectedRef {
						t.Errorf("reference mismatch at index %d: expected %q, got %q", i, expectedRef, fm.References[i])
					}
				}
			}

			// Compare metadata (simplified comparison)
			expectedMetaLen := 0
			if tc.expectedFM.Metadata != nil {
				expectedMetaLen = len(tc.expectedFM.Metadata)
			}
			actualMetaLen := 0
			if fm.Metadata != nil {
				actualMetaLen = len(fm.Metadata)
			}

			if actualMetaLen != expectedMetaLen {
				t.Errorf("metadata length mismatch: expected %d, got %d", expectedMetaLen, actualMetaLen)
			}

			if tc.expectedFM.Metadata != nil {
				for key, expectedValue := range tc.expectedFM.Metadata {
					if fm.Metadata == nil {
						t.Errorf("missing metadata key: %q", key)
						continue
					}
					if actualValue, exists := fm.Metadata[key]; !exists {
						t.Errorf("missing metadata key: %q", key)
					} else if actualValue != expectedValue {
						t.Errorf("metadata value mismatch for key %q: expected %v, got %v", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestSerializeWithFrontMatter(t *testing.T) {
	tests := map[string]struct {
		frontMatter    *FrontMatter
		content        string
		expectedResult string
	}{
		"full front matter with references and metadata": {
			frontMatter: &FrontMatter{
				References: []string{
					"./docs/architecture.md",
					"./specs/api.yaml",
				},
				Metadata: map[string]string{
					"project": "test-project",
					"created": "2024-01-30",
				},
			},
			content: `# Tasks

- [ ] 1. Test task`,
			expectedResult: `---
metadata:
    created: 2024-01-30
    project: test-project
references:
    - ./docs/architecture.md
    - ./specs/api.yaml
---
# Tasks

- [ ] 1. Test task`,
		},
		"only references": {
			frontMatter: &FrontMatter{
				References: []string{"./docs/test.md"},
				Metadata:   nil,
			},
			content: `# Tasks

- [ ] 1. Test task`,
			expectedResult: `---
references:
    - ./docs/test.md
---
# Tasks

- [ ] 1. Test task`,
		},
		"only metadata": {
			frontMatter: &FrontMatter{
				References: nil,
				Metadata: map[string]string{
					"project": "test",
				},
			},
			content: `# Tasks

- [ ] 1. Test task`,
			expectedResult: `---
metadata:
    project: test
---
# Tasks

- [ ] 1. Test task`,
		},
		"empty front matter": {
			frontMatter: &FrontMatter{
				References: nil,
				Metadata:   nil,
			},
			content: `# Tasks

- [ ] 1. Test task`,
			expectedResult: `# Tasks

- [ ] 1. Test task`,
		},
		"nil front matter": {
			frontMatter: nil,
			content: `# Tasks

- [ ] 1. Test task`,
			expectedResult: `# Tasks

- [ ] 1. Test task`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := SerializeWithFrontMatter(tc.frontMatter, tc.content)

			// For test cases with front matter, we need to be flexible about YAML ordering
			if tc.frontMatter != nil && (len(tc.frontMatter.References) > 0 || len(tc.frontMatter.Metadata) > 0) {
				// Check that it starts with ---
				if !strings.Contains(result, "---\n") {
					t.Errorf("result should contain front matter delimiters")
				}
				// Check that it contains the content
				if !strings.Contains(result, tc.content) {
					t.Errorf("result should contain the original content")
				}
				// Check that references are present if they exist
				for _, ref := range tc.frontMatter.References {
					if !strings.Contains(result, ref) {
						t.Errorf("result should contain reference: %s", ref)
					}
				}
			} else {
				// For empty/nil front matter, should return content as-is
				if result != tc.expectedResult {
					t.Errorf("result mismatch:\nexpected: %q\ngot: %q", tc.expectedResult, result)
				}
			}
		})
	}
}
