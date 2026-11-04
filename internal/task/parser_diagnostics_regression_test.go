package task

import (
	"strings"
	"testing"
)

// T-1879: parse errors reported line numbers relative to the content left
// after front matter and the title line were removed, not the source file.
func TestParseErrorReportsSourceLineNumber(t *testing.T) {
	tests := map[string]struct {
		content  string
		wantLine string
	}{
		"front matter and title": {
			content:  "---\nmetadata:\n  owner: demo\n---\n# Error Lines\n\n- [ ] 1. Valid\n- [q] 2. Invalid checkbox\n",
			wantLine: "line 8:",
		},
		"title only": {
			content:  "# Error Lines\n\n- [ ] 1. Valid\n- [q] 2. Invalid checkbox\n",
			wantLine: "line 4:",
		},
		"title after blank lines": {
			content:  "\n\n# Error Lines\n- [q] 1. Invalid checkbox\n",
			wantLine: "line 4:",
		},
		"no title": {
			content:  "- [ ] 1. Valid\n- [q] 2. Invalid checkbox\n",
			wantLine: "line 2:",
		},
		"front matter with CRLF": {
			content:  "---\r\nmetadata:\r\n  owner: demo\r\n---\r\n# Error Lines\r\n- [ ] 1. Valid\r\n  bad detail\r\n",
			wantLine: "line 7:",
		},
		"error in nested details": {
			content:  "---\nmetadata:\n  owner: demo\n---\n# T\n- [ ] 1. Parent\n  - detail\n  - [ ] 1.1. Child\n\tbad\n",
			wantLine: "line 9:",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseMarkdown([]byte(tc.content))
			if err == nil {
				t.Fatal("expected a parse error")
			}
			if !strings.HasPrefix(err.Error(), tc.wantLine) {
				t.Errorf("error = %q, want prefix %q", err, tc.wantLine)
			}
		})
	}
}

// T-2197: a malformed Stream value was silently kept as a plain detail with
// no warning. It must stay a detail (treated as unset) and emit a warning.
func TestMalformedStreamWarns(t *testing.T) {
	tests := map[string]string{
		"zero":         "Stream: 0",
		"negative":     "Stream: -1",
		"non-integer":  "Stream: two",
		"decimal":      "Stream: 1.5",
		"overflow":     "Stream: 99999999999999999999",
		"lowercase":    "stream: 0",
		"empty value":  "Stream:",
		"extra tokens": "Stream: 2 later",
	}

	for name, line := range tests {
		t.Run(name, func(t *testing.T) {
			content := "# Tasks\n\n- [ ] 1. Assigned-looking task\n  - " + line + "\n"
			tl, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatal(err)
			}
			got := tl.Tasks[0]
			if got.Stream != 0 {
				t.Errorf("Stream = %d, want 0 (unset)", got.Stream)
			}
			if len(got.Details) != 1 || got.Details[0] != line {
				t.Errorf("Details = %q, want [%q]", got.Details, line)
			}
			if len(tl.Warnings) != 1 {
				t.Fatalf("warnings = %+v, want exactly one", tl.Warnings)
			}
			if w := tl.Warnings[0]; w.Code != WarnInvalidStreamValue || w.TaskID != "1" {
				t.Errorf("warning = %+v, want code %q on task 1", w, WarnInvalidStreamValue)
			}
		})
	}
}

func TestValidStreamHasNoWarnings(t *testing.T) {
	tests := map[string]string{
		"single digit": "Stream: 2",
		"lowercase":    "stream: 3",
		"no space":     "Stream:4",
	}
	for name, line := range tests {
		t.Run(name, func(t *testing.T) {
			content := "- [ ] 1. Task\n  - " + line + "\n  - Streams are explained elsewhere\n"
			tl, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if len(tl.Warnings) != 0 {
				t.Errorf("unexpected warnings: %+v", tl.Warnings)
			}
			if tl.Tasks[0].Stream == 0 {
				t.Errorf("Stream not parsed from %q", line)
			}
		})
	}
}
