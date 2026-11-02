package task

import (
	"slices"
	"strings"
	"testing"
)

func roundTrip(t *testing.T, tl *TaskList) *TaskList {
	t.Helper()
	got, err := ParseMarkdown(RenderMarkdown(tl))
	if err != nil {
		t.Fatalf("re-parse failed: %v\nrendered:\n%s", err, RenderMarkdown(tl))
	}
	return got
}

// T-1848: a requirements file path containing a comma lost its requirement
// semantics because the parser split the whole line on every comma.
func TestRequirementsPathWithCommaRoundTrip(t *testing.T) {
	tl := NewTaskList("T")
	if _, err := tl.AddTask("", "Linked", ""); err != nil {
		t.Fatal(err)
	}
	tl.Tasks[0].Requirements = []string{"1.1", "2.3"}
	tl.RequirementsFile = "specs/requirements,v2.md"

	got := roundTrip(t, tl)
	if want := []string{"1.1", "2.3"}; !slices.Equal(got.Tasks[0].Requirements, want) {
		t.Errorf("requirements = %q, want %q (details: %q)", got.Tasks[0].Requirements, want, got.Tasks[0].Details)
	}
	if got.RequirementsFile != "specs/requirements,v2.md" {
		t.Errorf("requirements file = %q, want %q", got.RequirementsFile, "specs/requirements,v2.md")
	}
	if len(got.Tasks[0].Details) != 0 {
		t.Errorf("unexpected details: %q", got.Tasks[0].Details)
	}
}

// T-1907: requirement lines with any text besides comma-separated links were
// accepted as requirements and the extra text was dropped on the next write.
// Requirement 4.4: malformed requirement lines stay plain details, verbatim.
func TestMalformedRequirementsLinesStayDetails(t *testing.T) {
	tests := map[string]string{
		"prefix and suffix":             "Requirements: prefix [1.1](requirements.md#1.1) suffix",
		"suffix only":                   "Requirements: [1.1](requirements.md#1.1) see notes",
		"valid then malformed fragment": "Requirements: [1.1](requirements.md#1.1), not a link",
		"malformed then valid fragment": "Requirements: junk, [1.2](requirements.md#1.2)",
		"invalid requirement id":        "Requirements: [1.1](requirements.md#1.1), [abc](requirements.md#abc)",
		"trailing comma":                "Requirements: [1.1](requirements.md#1.1),",
	}

	for name, line := range tests {
		t.Run(name, func(t *testing.T) {
			content := "# Demo\n\n- [ ] 1. Task\n  - " + line + "\n"
			tl, err := ParseMarkdown([]byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if len(tl.Tasks[0].Requirements) != 0 {
				t.Errorf("requirements = %q, want none", tl.Tasks[0].Requirements)
			}
			if !slices.Equal(tl.Tasks[0].Details, []string{line}) {
				t.Errorf("details = %q, want %q", tl.Tasks[0].Details, []string{line})
			}

			if err := tl.UpdateStatus("1", InProgress); err != nil {
				t.Fatal(err)
			}
			if rendered := string(RenderMarkdown(tl)); !strings.Contains(rendered, "  - "+line+"\n") {
				t.Errorf("line not preserved verbatim after write:\n%s", rendered)
			}
		})
	}
}

// Valid requirement lines must keep working (requirement 4.3).
func TestValidRequirementsLinesStillParse(t *testing.T) {
	tests := map[string]struct {
		line string
		want []string
	}{
		"single":          {line: "Requirements: [1.1](requirements.md#1.1)", want: []string{"1.1"}},
		"multiple":        {line: "Requirements: [1.1](requirements.md#1.1), [2.3](requirements.md#2.3)", want: []string{"1.1", "2.3"}},
		"no space":        {line: "Requirements: [1.1](requirements.md#1.1),[2.3](requirements.md#2.3)", want: []string{"1.1", "2.3"}},
		"trailing spaces": {line: "Requirements: [1.1](requirements.md#1.1)  ", want: []string{"1.1"}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte("- [ ] 1. Task\n  - " + tc.line + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(tl.Tasks[0].Requirements, tc.want) {
				t.Errorf("requirements = %q, want %q (details %q)", tl.Tasks[0].Requirements, tc.want, tl.Tasks[0].Details)
			}
		})
	}
}
