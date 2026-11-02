package task

import (
	"slices"
	"testing"
)

// T-1998: malformed Blocked-by references were silently turned into a plain
// detail. Requirement 2.11: warn and ignore the invalid reference.
func TestMalformedBlockedByWarns(t *testing.T) {
	tests := map[string]struct {
		line        string
		wantBlocked []string
	}{
		"only invalid id":      {line: "Blocked-by: abc123 (missing prerequisite)", wantBlocked: nil},
		"valid and invalid id": {line: "Blocked-by: aaaaaaa (Finished prerequisite), abc123", wantBlocked: []string{"aaaaaaa"}},
		"no id":                {line: "Blocked-by: whenever", wantBlocked: nil},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			content := "- [x] 1. Finished prerequisite <!-- id:aaaaaaa -->\n" +
				"- [ ] 2. Must wait <!-- id:bbbbbbb -->\n" +
				"  - " + tc.line + "\n"
			tl, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(tl.Tasks[1].BlockedBy, tc.wantBlocked) {
				t.Errorf("BlockedBy = %q, want %q", tl.Tasks[1].BlockedBy, tc.wantBlocked)
			}
			if len(tl.Warnings) != 1 {
				t.Fatalf("warnings = %+v, want exactly one", tl.Warnings)
			}
			w := tl.Warnings[0]
			if w.Code != WarnInvalidBlockedBy || w.TaskID != "2" {
				t.Errorf("warning = %+v, want code %q on task 2", w, WarnInvalidBlockedBy)
			}
		})
	}
}

func TestValidBlockedByHasNoWarnings(t *testing.T) {
	content := "- [x] 1. A <!-- id:aaaaaaa -->\n- [ ] 2. B <!-- id:bbbbbbb -->\n  - Blocked-by: aaaaaaa (A, with comma)\n"
	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(tl.Warnings) != 0 {
		t.Errorf("unexpected warnings: %+v", tl.Warnings)
	}
	if !slices.Equal(tl.Tasks[1].BlockedBy, []string{"aaaaaaa"}) {
		t.Errorf("BlockedBy = %q", tl.Tasks[1].BlockedBy)
	}
}

// Warnings found while parsing lines must survive any check that runs on the
// parsed tasks afterwards, such as duplicate stable ID detection (PR #113).
// A file with both problems must still report the invalid Blocked-by
// reference.
func TestInvalidBlockedByWarningSurvivesDuplicateStableIDs(t *testing.T) {
	content := "- [x] 1. A <!-- id:aaaaaaa -->\n" +
		"- [ ] 2. B <!-- id:aaaaaaa -->\n" +
		"- [ ] 3. C <!-- id:ccccccc -->\n" +
		"  - Blocked-by: abc123\n"
	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range tl.Warnings {
		if w.Code == WarnInvalidBlockedBy && w.TaskID == "3" {
			found = true
		}
	}
	if !found {
		t.Errorf("invalid Blocked-by warning for task 3 missing from %+v", tl.Warnings)
	}
}
