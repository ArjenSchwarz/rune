package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

func TestHasPhasesDetection(t *testing.T) {
	tests := map[string]struct {
		content       string
		wantHasPhases bool
		wantCount     int
		wantPhases    []string
	}{
		"file_with_phases": {
			content: `# Tasks

## Planning
- [ ] 1. First task

## Implementation
- [ ] 2. Second task`,
			wantHasPhases: true,
			wantCount:     2,
			wantPhases:    []string{"Planning", "Implementation"},
		},
		"file_without_phases": {
			content: `# Tasks

- [ ] 1. First task
- [ ] 2. Second task`,
			wantHasPhases: false,
			wantCount:     0,
			wantPhases:    []string{},
		},
		"empty_file": {
			content:       "",
			wantHasPhases: false,
			wantCount:     0,
			wantPhases:    []string{},
		},
		"file_with_single_phase": {
			content: `## Setup
- [ ] 1. Task one`,
			wantHasPhases: true,
			wantCount:     1,
			wantPhases:    []string{"Setup"},
		},
		"file_with_mixed_content": {
			content: `# Tasks

- [ ] 1. Task before phases

## Phase One
- [ ] 2. Task in phase

- [ ] 3. Task after phase`,
			wantHasPhases: true,
			wantCount:     1,
			wantPhases:    []string{"Phase One"},
		},
		"file_with_duplicate_phase_names": {
			content: `## Testing
- [ ] 1. First task

## Testing
- [ ] 2. Second task`,
			wantHasPhases: true,
			wantCount:     2,
			wantPhases:    []string{"Testing", "Testing"},
		},
		"file_with_empty_phases": {
			content: `## Phase One

## Phase Two
- [ ] 1. Task`,
			wantHasPhases: true,
			wantCount:     2,
			wantPhases:    []string{"Phase One", "Phase Two"},
		},
		"file_with_h1_and_h3_only": {
			content: `# Main Title

### Not a phase
- [ ] 1. Task`,
			wantHasPhases: false,
			wantCount:     0,
			wantPhases:    []string{},
		},
		"file_with_special_characters_in_phase": {
			content: `## Phase-1: Setup & Config
- [ ] 1. Task`,
			wantHasPhases: true,
			wantCount:     1,
			wantPhases:    []string{"Phase-1: Setup & Config"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test using ExtractPhaseMarkers directly
			lines := strings.Split(tc.content, "\n")
			markers := task.ExtractPhaseMarkers(lines)

			hasPhases := len(markers) > 0
			if hasPhases != tc.wantHasPhases {
				t.Errorf("HasPhases = %v, want %v", hasPhases, tc.wantHasPhases)
			}

			if len(markers) != tc.wantCount {
				t.Errorf("Count = %d, want %d", len(markers), tc.wantCount)
			}

			// Verify phase names
			if len(markers) == len(tc.wantPhases) {
				for i, marker := range markers {
					if marker.Name != tc.wantPhases[i] {
						t.Errorf("Phase[%d] = %q, want %q", i, marker.Name, tc.wantPhases[i])
					}
				}
			}
		})
	}
}

func TestHasPhasesCommandOutput(t *testing.T) {
	// Create temp directory within current working directory
	tempDir := filepath.Join(".", "test-tmp-has-phases")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := map[string]struct {
		content       string
		verbose       bool
		wantHasPhases bool
		wantCount     int
		wantPhases    []string
	}{
		"basic_with_phases": {
			content: `## Planning
- [ ] 1. Task`,
			verbose:       false,
			wantHasPhases: true,
			wantCount:     1,
			wantPhases:    []string{}, // Not verbose, so empty
		},
		"verbose_with_phases": {
			content: `## Planning
- [ ] 1. Task

## Implementation
- [ ] 2. Task`,
			verbose:       true,
			wantHasPhases: true,
			wantCount:     2,
			wantPhases:    []string{"Planning", "Implementation"},
		},
		"without_phases": {
			content:       `- [ ] 1. Task`,
			verbose:       false,
			wantHasPhases: false,
			wantCount:     0,
			wantPhases:    []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpFile := filepath.Join(tempDir, name+".md")
			if err := os.WriteFile(tmpFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			// Test the output structure
			lines := strings.Split(tc.content, "\n")
			markers := task.ExtractPhaseMarkers(lines)

			result := HasPhasesOutput{
				HasPhases: len(markers) > 0,
				Count:     len(markers),
			}

			if tc.verbose {
				result.Phases = make([]string, len(markers))
				for i, marker := range markers {
					result.Phases[i] = marker.Name
				}
			} else {
				result.Phases = []string{}
			}

			// Verify JSON marshaling works
			jsonOutput, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Failed to marshal JSON: %v", err)
			}

			// Verify output can be unmarshaled
			var unmarshaled HasPhasesOutput
			if err := json.Unmarshal(jsonOutput, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if unmarshaled.HasPhases != tc.wantHasPhases {
				t.Errorf("HasPhases = %v, want %v", unmarshaled.HasPhases, tc.wantHasPhases)
			}

			if unmarshaled.Count != tc.wantCount {
				t.Errorf("Count = %d, want %d", unmarshaled.Count, tc.wantCount)
			}

			if len(unmarshaled.Phases) != len(tc.wantPhases) {
				t.Errorf("Number of phases = %d, want %d", len(unmarshaled.Phases), len(tc.wantPhases))
			}
		})
	}
}

func TestHasPhasesWithMalformedFile(t *testing.T) {
	tests := map[string]struct {
		content       string
		wantHasPhases bool
		wantCount     int
	}{
		"file_with_incomplete_task_markers": {
			content: `## Planning
- [ ] 1. Good task
- [x Bad marker`,
			wantHasPhases: true,
			wantCount:     1,
		},
		"file_with_only_phase_headers": {
			content: `## Phase 1
## Phase 2
## Phase 3`,
			wantHasPhases: true,
			wantCount:     3,
		},
		"file_with_malformed_headers": {
			content: `##No space phase
## Good Phase
###Three hashes`,
			wantHasPhases: true,
			wantCount:     1, // Only "Good Phase" should count
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			lines := strings.Split(tc.content, "\n")
			markers := task.ExtractPhaseMarkers(lines)

			hasPhases := len(markers) > 0
			if hasPhases != tc.wantHasPhases {
				t.Errorf("HasPhases = %v, want %v", hasPhases, tc.wantHasPhases)
			}

			if len(markers) != tc.wantCount {
				t.Errorf("Count = %d, want %d", len(markers), tc.wantCount)
			}
		})
	}
}
