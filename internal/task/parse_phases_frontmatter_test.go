package task

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestParseFileWithPhases_FrontMatterStripping verifies that
// ParseFileWithPhases correctly strips front matter and extracts phase
// markers. The bug (T-458) was that the front-matter stripping loop
// treated ANY "---" line as a front-matter delimiter, not just the
// initial pair. This caused lines after a later "---" to be dropped
// from phase-marker extraction.
func TestParseFileWithPhases_FrontMatterStripping(t *testing.T) {
	tests := map[string]struct {
		content     string
		wantPhases  []PhaseMarker
		wantTasks   int
		description string
	}{
		"front_matter_with_phases": {
			content: `---
references:
  - ./docs/spec.md
---
# Tasks

## Phase 1

- [ ] 1. First task

## Phase 2

- [ ] 2. Second task`,
			wantPhases: []PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "1"},
			},
			wantTasks:   2,
			description: "Front matter with phases should extract all phase markers",
		},
		"empty_front_matter_with_phases": {
			content: `---
---
# Tasks

## Phase 1

- [ ] 1. First task

## Phase 2

- [ ] 2. Second task`,
			wantPhases: []PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "1"},
			},
			wantTasks:   2,
			description: "Empty front matter should still extract all phase markers",
		},
		"no_front_matter_with_phases": {
			content: `# Tasks

## Phase 1

- [ ] 1. First task

## Phase 2

- [ ] 2. Second task`,
			wantPhases: []PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "1"},
			},
			wantTasks:   2,
			description: "No front matter should extract all phase markers",
		},
		"front_matter_three_phases": {
			content: `---
metadata:
  project: test
---
# Tasks

## Planning

- [ ] 1. Plan

## Implementation

- [ ] 2. Build

## Testing

- [ ] 3. Test`,
			wantPhases: []PhaseMarker{
				{Name: "Planning", AfterTaskID: ""},
				{Name: "Implementation", AfterTaskID: "1"},
				{Name: "Testing", AfterTaskID: "2"},
			},
			wantTasks:   3,
			description: "Front matter with three phases should extract all markers",
		},
		"front_matter_with_many_references": {
			content: `---
references:
  - ./docs/spec.md
  - ./docs/design.md
  - ./docs/requirements.md
metadata:
  project: complex
  version: "2.0"
---

## Phase 1

- [ ] 1. First task
  - [ ] 1.1. Subtask

## Phase 2

- [ ] 2. Second task`,
			wantPhases: []PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "1"},
			},
			wantTasks:   2,
			description: "Complex front matter should not interfere with phase extraction",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test.md")
			if err := writeTestFile(tmpFile, tc.content); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			taskList, phaseMarkers, err := ParseFileWithPhases(tmpFile)
			if err != nil {
				t.Fatalf("%s\nParseFileWithPhases() error: %v", tc.description, err)
			}

			if len(taskList.Tasks) != tc.wantTasks {
				t.Errorf("%s\nTask count = %d, want %d", tc.description, len(taskList.Tasks), tc.wantTasks)
			}

			if !reflect.DeepEqual(phaseMarkers, tc.wantPhases) {
				t.Errorf("%s\nPhase markers mismatch:", tc.description)
				t.Errorf("  got  %d markers, want %d", len(phaseMarkers), len(tc.wantPhases))
				for i, m := range phaseMarkers {
					if i < len(tc.wantPhases) {
						if m != tc.wantPhases[i] {
							t.Errorf("  marker[%d]: got {Name: %q, AfterTaskID: %q}, want {Name: %q, AfterTaskID: %q}",
								i, m.Name, m.AfterTaskID, tc.wantPhases[i].Name, tc.wantPhases[i].AfterTaskID)
						}
					} else {
						t.Errorf("  extra marker[%d]: {Name: %q, AfterTaskID: %q}", i, m.Name, m.AfterTaskID)
					}
				}
				for i := len(phaseMarkers); i < len(tc.wantPhases); i++ {
					t.Errorf("  missing marker[%d]: {Name: %q, AfterTaskID: %q}",
						i, tc.wantPhases[i].Name, tc.wantPhases[i].AfterTaskID)
				}
			}
		})
	}
}

// TestFrontMatterStrippingForPhaseExtraction directly tests the front-matter
// stripping logic that ParseFileWithPhases uses before calling
// ExtractPhaseMarkers. This tests the stripping in isolation from
// ParseMarkdown, so we can verify it handles "---" lines correctly even
// when they appear after front matter (the T-458 bug scenario).
func TestFrontMatterStrippingForPhaseExtraction(t *testing.T) {
	tests := map[string]struct {
		// rawLines is the full file content split into lines, as
		// ParseFileWithPhases would see before stripping front matter
		rawContent string
		// startsWithFrontMatter controls whether the front-matter stripping
		// code path is entered
		startsWithFrontMatter bool
		wantPhases            []PhaseMarker
		description           string
	}{
		"front_matter_then_hr_then_phase": {
			// This is the exact T-458 scenario: front matter at top,
			// then a horizontal rule, then a phase header after it.
			// The old code would treat the 3rd --- as re-entering front
			// matter, dropping Phase 2 from extraction.
			rawContent:            "---\nreferences:\n  - ./spec.md\n---\n\n## Phase 1\n\n- [ ] 1. First\n\n---\n\n## Phase 2\n\n- [ ] 2. Second",
			startsWithFrontMatter: true,
			wantPhases: []PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "1"},
			},
			description: "Horizontal rule after front matter must not suppress later phase markers",
		},
		"front_matter_then_multiple_hrs": {
			rawContent:            "---\nmetadata:\n  v: 1\n---\n\n## P1\n\n- [ ] 1. T1\n\n---\n\n## P2\n\n- [ ] 2. T2\n\n---\n\n## P3\n\n- [ ] 3. T3",
			startsWithFrontMatter: true,
			wantPhases: []PhaseMarker{
				{Name: "P1", AfterTaskID: ""},
				{Name: "P2", AfterTaskID: "1"},
				{Name: "P3", AfterTaskID: "2"},
			},
			description: "Multiple horizontal rules after front matter must not suppress phase markers",
		},
		"front_matter_only_no_hr": {
			rawContent:            "---\nreferences:\n  - ./spec.md\n---\n\n## Phase 1\n\n- [ ] 1. First\n\n## Phase 2\n\n- [ ] 2. Second",
			startsWithFrontMatter: true,
			wantPhases: []PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "1"},
			},
			description: "Front matter without horizontal rules should work correctly (baseline)",
		},
		"no_front_matter_with_hr": {
			rawContent:            "# Tasks\n\n## Phase 1\n\n- [ ] 1. First\n\n---\n\n## Phase 2\n\n- [ ] 2. Second",
			startsWithFrontMatter: false,
			wantPhases: []PhaseMarker{
				{Name: "Phase 1", AfterTaskID: ""},
				{Name: "Phase 2", AfterTaskID: "1"},
			},
			description: "Without front matter, horizontal rules should not affect phase extraction",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reproduce the front-matter stripping logic from ParseFileWithPhases
			lines := strings.Split(tc.rawContent, "\n")

			if tc.startsWithFrontMatter {
				lines = stripFrontMatterLines(lines, tc.rawContent)
			}

			markers := ExtractPhaseMarkers(lines)

			if !reflect.DeepEqual(markers, tc.wantPhases) {
				t.Errorf("%s\nPhase markers mismatch:", tc.description)
				t.Errorf("  got  %d markers, want %d", len(markers), len(tc.wantPhases))
				for i, m := range markers {
					if i < len(tc.wantPhases) {
						if m != tc.wantPhases[i] {
							t.Errorf("  marker[%d]: got {Name: %q, AfterTaskID: %q}, want {Name: %q, AfterTaskID: %q}",
								i, m.Name, m.AfterTaskID, tc.wantPhases[i].Name, tc.wantPhases[i].AfterTaskID)
						}
					} else {
						t.Errorf("  extra marker[%d]: {Name: %q, AfterTaskID: %q}", i, m.Name, m.AfterTaskID)
					}
				}
				for i := len(markers); i < len(tc.wantPhases); i++ {
					t.Errorf("  missing marker[%d]: {Name: %q, AfterTaskID: %q}",
						i, tc.wantPhases[i].Name, tc.wantPhases[i].AfterTaskID)
				}
			}
		})
	}
}
