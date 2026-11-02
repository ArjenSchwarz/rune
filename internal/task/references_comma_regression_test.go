package task

import (
	"slices"
	"testing"
)

func TestReferencesWithCommasRoundTrip(t *testing.T) {
	tests := map[string]struct {
		refs []string
	}{
		"url with comma":            {refs: []string{"https://example.test/search?q=a,b"}},
		"comma ref among plain":     {refs: []string{"a.md", "dir/x,y.md", "b.md"}},
		"comma and space":           {refs: []string{"one, two"}},
		"backslash before comma":    {refs: []string{`dir\`, "b.md"}},
		"escaped-looking backslash": {refs: []string{`a\,b`, `c\\d`}},
		"windows path":              {refs: []string{`docs\specs\a.md`}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := NewTaskList("T")
			if _, err := tl.AddTask("", "Task", ""); err != nil {
				t.Fatal(err)
			}
			tl.Tasks[0].References = tc.refs

			got := roundTrip(t, tl)
			if !slices.Equal(got.Tasks[0].References, tc.refs) {
				t.Errorf("references = %q, want %q\nrendered:\n%s", got.Tasks[0].References, tc.refs, RenderMarkdown(tl))
			}
		})
	}
}

func TestReferencesLegacyFormat(t *testing.T) {
	tests := map[string]struct {
		line string
		want []string
	}{
		"comma separated": {line: "References: a.md, b.md", want: []string{"a.md", "b.md"}},
		"windows path":    {line: `References: docs\a.md, c.md`, want: []string{`docs\a.md`, "c.md"}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte("- [ ] 1. Task\n  - " + tc.line + "\n"))
			if err != nil {
				t.Fatal(err)
			}
			if got := tl.Tasks[0].References; !slices.Equal(got, tc.want) {
				t.Errorf("references = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBatchReferencesWithCommaRoundTrip(t *testing.T) {
	tl := NewTaskList("T")
	resp, err := tl.ExecuteBatch([]Operation{{
		Type:       "add",
		Title:      "With URL",
		References: []string{"https://example.test/search?q=a,b"},
	}}, false)
	if err != nil || !resp.Success {
		t.Fatalf("batch failed: %v %+v", err, resp)
	}
	got := roundTrip(t, tl)
	want := []string{"https://example.test/search?q=a,b"}
	if !slices.Equal(got.Tasks[0].References, want) {
		t.Errorf("references = %q, want %q", got.Tasks[0].References, want)
	}
}
