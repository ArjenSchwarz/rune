package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// T-2063 review: commands that scan raw file bytes for phase headers must
// ignore a leading UTF-8 BOM, or a "## First" header on line 1 is missed and
// BOM-prefixed front matter is not skipped.
const (
	bomPhasedFile      = "\xef\xbb\xbf## First\n\n- [ ] 1. A\n\n## Second\n\n- [ ] 2. B\n"
	bomFrontMatterFile = "\xef\xbb\xbf---\n## yaml comment\nmetadata:\n  x: 1\n---\n## Real\n\n- [ ] 1. A\n"
	// The only "## " line is a YAML comment, so the file has no phases.
	bomFrontMatterNoPhases = "\xef\xbb\xbf---\n## yaml comment\nmetadata:\n  x: 1\n---\n- [ ] 1. A\n"
)

func writeBOMFile(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNextPhase_UTF8BOM(t *testing.T) {
	tests := map[string]struct {
		content   string
		stream    int
		wantPhase string
	}{
		"bom before first phase":             {content: bomPhasedFile, wantPhase: "First"},
		"bom before first phase with stream": {content: bomPhasedFile, stream: 1, wantPhase: "First"},
		"bom before front matter":            {content: bomFrontMatterNoPhases, wantPhase: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			writeBOMFile(t, tc.content)
			oldFormat := format
			t.Cleanup(func() {
				format = oldFormat
				phaseFlag, streamFlag = false, 0
			})
			format = formatJSON
			phaseFlag, streamFlag, claimFlag, oneFlag = true, tc.stream, "", false

			out := captureStdout(t, func() {
				if err := runNext(nextCmd, []string{"tasks.md"}); err != nil {
					t.Errorf("next failed: %v", err)
				}
			})

			var got struct {
				PhaseName string `json:"phase_name"`
			}
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("invalid JSON %q: %v", out, err)
			}
			if got.PhaseName != tc.wantPhase {
				t.Errorf("phase_name = %q, want %q", got.PhaseName, tc.wantPhase)
			}
		})
	}
}

func TestHasPhases_UTF8BOM(t *testing.T) {
	tests := map[string]struct {
		content    string
		wantPhases []string
	}{
		"bom before first phase":  {content: bomPhasedFile, wantPhases: []string{"First", "Second"}},
		"bom before front matter": {content: bomFrontMatterFile, wantPhases: []string{"Real"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			writeBOMFile(t, tc.content)
			t.Cleanup(func() { hasPhasesVerbose = false })
			hasPhasesVerbose = true

			out := captureStdout(t, func() {
				if err := runHasPhases(hasPhasesCmd, []string{"tasks.md"}); err != nil {
					t.Errorf("has-phases failed: %v", err)
				}
			})

			var got HasPhasesOutput
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("invalid JSON %q: %v", out, err)
			}
			if !slices.Equal(got.Phases, tc.wantPhases) {
				t.Errorf("phases = %q, want %q", got.Phases, tc.wantPhases)
			}
		})
	}
}
