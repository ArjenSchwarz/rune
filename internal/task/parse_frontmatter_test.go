package task

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRequirements(t *testing.T) {
	tests := map[string]struct {
		input    string
		wantIDs  []string
		wantFile string
	}{
		"single_requirement": {
			input:    "[1.1](requirements.md#1.1)",
			wantIDs:  []string{"1.1"},
			wantFile: "requirements.md",
		},
		"multiple_requirements": {
			input:    "[1.1](requirements.md#1.1), [1.2](requirements.md#1.2)",
			wantIDs:  []string{"1.1", "1.2"},
			wantFile: "requirements.md",
		},
		"malformed_link_no_markdown": {
			input:    "1.1, 1.2",
			wantIDs:  []string{},
			wantFile: "",
		},
		"whitespace_handling": {
			input:    "  [1.1](requirements.md#1.1)  ,  [2.3](requirements.md#2.3)  ",
			wantIDs:  []string{"1.1", "2.3"},
			wantFile: "requirements.md",
		},
		"custom_requirements_file": {
			input:    "[1.1](specs/requirements.md#1.1), [1.2](specs/requirements.md#1.2)",
			wantIDs:  []string{"1.1", "1.2"},
			wantFile: "specs/requirements.md",
		},
		"mixed_valid_invalid": {
			input:    "[1.1](requirements.md#1.1), invalid, [2.3](requirements.md#2.3)",
			wantIDs:  []string{"1.1", "2.3"},
			wantFile: "requirements.md",
		},
		"invalid_requirement_id_format": {
			input:    "[abc](requirements.md#abc)",
			wantIDs:  []string{},
			wantFile: "",
		},
		"hierarchical_requirement_ids": {
			input:    "[1.2.3](requirements.md#1.2.3), [2.1.4.5](requirements.md#2.1.4.5)",
			wantIDs:  []string{"1.2.3", "2.1.4.5"},
			wantFile: "requirements.md",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotIDs, gotFile := parseRequirements(tc.input)

			if len(gotIDs) != len(tc.wantIDs) {
				t.Errorf("parseRequirements() returned %d IDs, want %d", len(gotIDs), len(tc.wantIDs))
			}

			for i, wantID := range tc.wantIDs {
				if i >= len(gotIDs) {
					break
				}
				if gotIDs[i] != wantID {
					t.Errorf("parseRequirements() ID[%d] = %q, want %q", i, gotIDs[i], wantID)
				}
			}

			if gotFile != tc.wantFile {
				t.Errorf("parseRequirements() file = %q, want %q", gotFile, tc.wantFile)
			}
		})
	}
}

func TestParseMarkdownWithRequirements(t *testing.T) {
	tests := map[string]struct {
		content          string
		taskID           string
		wantRequirements []string
		wantReqFile      string
		wantDetails      []string
	}{
		"task_with_single_requirement": {
			content: `# Tasks
- [ ] 1. Implement feature
  - Requirements: [1.1](requirements.md#1.1)`,
			taskID:           "1",
			wantRequirements: []string{"1.1"},
			wantReqFile:      "requirements.md",
		},
		"task_with_multiple_requirements": {
			content: `# Tasks
- [ ] 1. Implement authentication
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [2.3](requirements.md#2.3)`,
			taskID:           "1",
			wantRequirements: []string{"1.1", "1.2", "2.3"},
			wantReqFile:      "requirements.md",
		},
		"task_with_custom_requirements_file": {
			content: `# Tasks
- [ ] 1. Implement feature
  - Requirements: [1.1](specs/requirements.md#1.1), [1.2](specs/requirements.md#1.2)`,
			taskID:           "1",
			wantRequirements: []string{"1.1", "1.2"},
			wantReqFile:      "specs/requirements.md",
		},
		"task_with_requirements_and_details": {
			content: `# Tasks
- [ ] 1. Implement login
  - Use JWT tokens
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2)
  - Add proper validation`,
			taskID:           "1",
			wantRequirements: []string{"1.1", "1.2"},
			wantReqFile:      "requirements.md",
			wantDetails:      []string{"Use JWT tokens", "Add proper validation"},
		},
		"task_with_requirements_and_references": {
			content: `# Tasks
- [ ] 1. Implement feature
  - Requirements: [1.1](requirements.md#1.1)
  - References: design.md, spec.md`,
			taskID:           "1",
			wantRequirements: []string{"1.1"},
			wantReqFile:      "requirements.md",
		},
		"malformed_requirements_treated_as_detail": {
			content: `# Tasks
- [ ] 1. Implement feature
  - Requirements: 1.1, 1.2`,
			taskID:      "1",
			wantDetails: []string{"Requirements: 1.1, 1.2"},
		},
		"subtask_with_requirements": {
			content: `# Tasks
- [ ] 1. Parent task
  - [ ] 1.1. Child task
    - Requirements: [2.1](requirements.md#2.1)`,
			taskID:           "1.1",
			wantRequirements: []string{"2.1"},
			wantReqFile:      "requirements.md",
		},
		"multiple_tasks_different_requirements": {
			content: `# Tasks
- [ ] 1. First task
  - Requirements: [1.1](requirements.md#1.1)
- [ ] 2. Second task
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2)`,
			taskID:           "2",
			wantRequirements: []string{"2.1", "2.2"},
			wantReqFile:      "requirements.md",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("ParseMarkdown() error: %v", err)
			}

			task := tl.FindTask(tc.taskID)
			if task == nil {
				t.Fatalf("Task %s not found", tc.taskID)
			}

			// Check requirements
			if tc.wantRequirements != nil {
				if len(task.Requirements) != len(tc.wantRequirements) {
					t.Errorf("Task requirements count = %d, want %d", len(task.Requirements), len(tc.wantRequirements))
				}
				for i, req := range tc.wantRequirements {
					if i >= len(task.Requirements) {
						break
					}
					if task.Requirements[i] != req {
						t.Errorf("Task requirement[%d] = %q, want %q", i, task.Requirements[i], req)
					}
				}
			}

			// Check requirements file
			if tc.wantReqFile != "" {
				if tl.RequirementsFile != tc.wantReqFile {
					t.Errorf("TaskList RequirementsFile = %q, want %q", tl.RequirementsFile, tc.wantReqFile)
				}
			}

			// Check details
			if tc.wantDetails != nil {
				if len(task.Details) != len(tc.wantDetails) {
					t.Errorf("Task details count = %d, want %d", len(task.Details), len(tc.wantDetails))
				}
				for i, detail := range tc.wantDetails {
					if i >= len(task.Details) {
						break
					}
					if task.Details[i] != detail {
						t.Errorf("Task detail[%d] = %q, want %q", i, task.Details[i], detail)
					}
				}
			}
		})
	}
}

func TestParseRequirementsRoundTrip(t *testing.T) {
	tests := map[string]struct {
		content string
	}{
		"requirements_preserved_in_roundtrip": {
			content: `# Tasks

- [ ] 1. Implement authentication
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2)
  - Use JWT tokens
  - References: auth-spec.md

- [ ] 2. Add validation
  - Requirements: [2.1](requirements.md#2.1)
  - Validate all inputs
`,
		},
		"requirements_with_custom_file": {
			content: `# Tasks

- [ ] 1. Implement feature
  - Requirements: [1.1](specs/requirements.md#1.1), [1.2](specs/requirements.md#1.2)
  - Add proper tests
`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Parse original content
			tl1, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("First ParseMarkdown() error: %v", err)
			}

			// Render to markdown (we'll need to implement this in the next phase)
			// For now, just verify the data is preserved in the parsed structure

			// Verify requirements are preserved
			for _, task := range tl1.Tasks {
				if len(task.Requirements) > 0 {
					// Requirements should be preserved
					for _, reqID := range task.Requirements {
						if reqID == "" {
							t.Errorf("Empty requirement ID found in task %s", task.ID)
						}
					}
				}
			}

			// Verify requirements file is preserved
			if tl1.RequirementsFile != "" {
				t.Logf("RequirementsFile preserved: %s", tl1.RequirementsFile)
			}
		})
	}
}

func TestParseRoundTrip(t *testing.T) {
	// This test will verify that parse → render → parse produces identical structure
	// Will be implemented after renderer is complete
	t.Skip("Round-trip test will be implemented after renderer")
}

func TestParseMarkdownWithFrontMatter(t *testing.T) {
	tests := map[string]struct {
		content        string
		wantTitle      string
		wantTasks      int
		wantReferences []string
		wantMetadata   map[string]string
		wantErr        bool
		errContains    string
	}{
		"with_front_matter_and_references": {
			content: `---
references:
  - ./docs/architecture.md
  - ./specs/api-specification.yaml
metadata:
  project: backend-api
  created: "2024-01-30"
---
# Project Tasks

- [ ] 1. Setup development environment
  - [x] 1.1. Install dependencies
  - [ ] 1.2. Configure database
- [x] 2. Implement authentication`,
			wantTitle: "Project Tasks",
			wantTasks: 2,
			wantReferences: []string{
				"./docs/architecture.md",
				"./specs/api-specification.yaml",
			},
			wantMetadata: map[string]string{
				"project": "backend-api",
				"created": "2024-01-30",
			},
		},
		"with_empty_front_matter": {
			content: `---
---
# Tasks

- [ ] 1. First task
- [ ] 2. Second task`,
			wantTitle:      "Tasks",
			wantTasks:      2,
			wantReferences: nil,
			wantMetadata:   nil,
		},
		"without_front_matter": {
			content: `# Regular Tasks

- [ ] 1. Task one
- [x] 2. Task two
- [-] 3. Task three`,
			wantTitle:      "Regular Tasks",
			wantTasks:      3,
			wantReferences: nil,
			wantMetadata:   nil,
		},
		"front_matter_only_references": {
			content: `---
references:
  - ../shared/database-schema.sql
  - ./docs/setup.md
---
# Setup Tasks

- [ ] 1. Initialize project`,
			wantTitle: "Setup Tasks",
			wantTasks: 1,
			wantReferences: []string{
				"../shared/database-schema.sql",
				"./docs/setup.md",
			},
			wantMetadata: nil,
		},
		"front_matter_only_metadata": {
			content: `---
metadata:
  version: 1.0.0
  author: John Doe
---
# Version Tasks

- [ ] 1. Update version`,
			wantTitle:      "Version Tasks",
			wantTasks:      1,
			wantReferences: nil,
			wantMetadata: map[string]string{
				"version": "1.0.0",
				"author":  "John Doe",
			},
		},
		"unclosed_front_matter": {
			content: `---
references:
  - ./docs/test.md
# This should fail

- [ ] 1. Task`,
			wantErr:     true,
			errContains: "unclosed front matter block",
		},
		"invalid_yaml_in_front_matter": {
			content: `---
references: [
  - item1
  - item2
---
# Tasks

- [ ] 1. Task`,
			wantErr:     true,
			errContains: "parsing front matter",
		},
		"tasks_immediately_after_front_matter": {
			content: `---
references:
  - ./README.md
---
- [ ] 1. First task without title
- [ ] 2. Second task`,
			wantTitle: "",
			wantTasks: 2,
			wantReferences: []string{
				"./README.md",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			taskList, err := ParseMarkdown([]byte(tc.content))

			if tc.wantErr {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("Error = %v, want error containing %q", err, tc.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if taskList.Title != tc.wantTitle {
				t.Errorf("Title = %q, want %q", taskList.Title, tc.wantTitle)
			}

			if len(taskList.Tasks) != tc.wantTasks {
				t.Errorf("Tasks count = %d, want %d", len(taskList.Tasks), tc.wantTasks)
			}

			// Check references
			if tc.wantReferences == nil {
				if taskList.FrontMatter != nil && len(taskList.FrontMatter.References) > 0 {
					t.Errorf("Expected no references, but got %v", taskList.FrontMatter.References)
				}
			} else {
				if taskList.FrontMatter == nil {
					t.Fatal("Expected front matter but got nil")
				}
				if len(taskList.FrontMatter.References) != len(tc.wantReferences) {
					t.Errorf("References count = %d, want %d", len(taskList.FrontMatter.References), len(tc.wantReferences))
				} else {
					for i, ref := range taskList.FrontMatter.References {
						if ref != tc.wantReferences[i] {
							t.Errorf("Reference[%d] = %q, want %q", i, ref, tc.wantReferences[i])
						}
					}
				}
			}

			// Check metadata
			if tc.wantMetadata == nil {
				if taskList.FrontMatter != nil && len(taskList.FrontMatter.Metadata) > 0 {
					t.Errorf("Expected no metadata, but got %v", taskList.FrontMatter.Metadata)
				}
			} else {
				if taskList.FrontMatter == nil {
					t.Fatal("Expected front matter but got nil")
				}
				if len(taskList.FrontMatter.Metadata) != len(tc.wantMetadata) {
					t.Errorf("Metadata count = %d, want %d", len(taskList.FrontMatter.Metadata), len(tc.wantMetadata))
				} else {
					for key, wantVal := range tc.wantMetadata {
						if gotVal, ok := taskList.FrontMatter.Metadata[key]; !ok {
							t.Errorf("Metadata missing key %q", key)
						} else if gotVal != wantVal {
							t.Errorf("Metadata[%q] = %v, want %v", key, gotVal, wantVal)
						}
					}
				}
			}
		})
	}
}

func TestParseFileWithFrontMatter(t *testing.T) {
	// Create a temporary file with front matter
	content := `---
references:
  - ./docs/test.md
metadata:
  test: true
---
# Test File

- [ ] 1. Test task`

	tmpFile := filepath.Join(t.TempDir(), "test_with_frontmatter.md")
	if err := writeTestFile(tmpFile, content); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	taskList, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Verify FilePath is set
	if taskList.FilePath != tmpFile {
		t.Errorf("FilePath = %q, want %q", taskList.FilePath, tmpFile)
	}

	// Verify front matter is parsed
	if taskList.FrontMatter == nil {
		t.Fatal("Expected front matter but got nil")
	}

	if len(taskList.FrontMatter.References) != 1 || taskList.FrontMatter.References[0] != "./docs/test.md" {
		t.Errorf("References = %v, want [./docs/test.md]", taskList.FrontMatter.References)
	}

	// Verify task content is still parsed correctly
	if taskList.Title != "Test File" {
		t.Errorf("Title = %q, want %q", taskList.Title, "Test File")
	}

	if len(taskList.Tasks) != 1 {
		t.Errorf("Tasks count = %d, want 1", len(taskList.Tasks))
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that files without front matter continue to work
	tests := map[string]struct {
		content   string
		wantTitle string
		wantTasks int
	}{
		"simple_markdown": {
			content: `# My Tasks

- [ ] 1. First task
- [x] 2. Completed task`,
			wantTitle: "My Tasks",
			wantTasks: 2,
		},
		"no_title": {
			content: `- [ ] 1. Task one
- [ ] 2. Task two`,
			wantTitle: "",
			wantTasks: 2,
		},
		"with_subtasks": {
			content: `# Project

- [ ] 1. Main
  - [ ] 1.1. Sub one
  - [x] 1.2. Sub two`,
			wantTitle: "Project",
			wantTasks: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			taskList, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if taskList.Title != tc.wantTitle {
				t.Errorf("Title = %q, want %q", taskList.Title, tc.wantTitle)
			}

			if len(taskList.Tasks) != tc.wantTasks {
				t.Errorf("Tasks count = %d, want %d", len(taskList.Tasks), tc.wantTasks)
			}

			// Ensure FrontMatter is empty but not nil for backward compatibility
			if taskList.FrontMatter == nil {
				t.Error("FrontMatter should not be nil for backward compatibility")
			}
		})
	}
}
