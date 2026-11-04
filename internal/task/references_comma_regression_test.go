package task

import (
	"slices"
	"strings"
	"testing"
)

// T-1808: task references containing commas were split into several
// references on the next read because the renderer joined them with ", " and
// the parser split on every comma. Expected: each reference round-trips as
// one element.
func TestReferencesWithCommasRoundTrip(t *testing.T) {
	tests := map[string]struct {
		refs []string
	}{
		"url with comma":            {refs: []string{"https://example.test/search?q=a,b"}},
		"comma ref among plain":     {refs: []string{"a.md", "dir/x,y.md", "b.md"}},
		"comma and space":           {refs: []string{"one, two"}},
		"escaped-looking backslash": {refs: []string{`a\,b`, `c\\d`, `e\\,f`}},
		"windows path":              {refs: []string{`docs\specs\a.md`}},
		"unc path":                  {refs: []string{`\\server\share\file.md`, "b.md"}},
		"unc path with comma":       {refs: []string{`\\server\share\a,b.md`}},
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

// Existing files written before references were escaped must keep their
// meaning, and a rewrite must not change the line (PR #120 review: a UNC
// path lost a leading backslash when "\\" was also an escape).
func TestReferencesLegacyFormat(t *testing.T) {
	tests := map[string]struct {
		line string
		want []string
	}{
		"comma separated": {line: "References: a.md, b.md", want: []string{"a.md", "b.md"}},
		"windows path":    {line: `References: docs\a.md, c.md`, want: []string{`docs\a.md`, "c.md"}},
		"unc path":        {line: `References: \\server\share\file.md`, want: []string{`\\server\share\file.md`}},
		"unc among plain": {line: `References: a.md, \\server\share\file.md, b.md`, want: []string{"a.md", `\\server\share\file.md`, "b.md"}},
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

			// A mutation that does not touch references rewrites the line unchanged.
			if err := tl.UpdateStatus("1", InProgress); err != nil {
				t.Fatal(err)
			}
			if rendered := string(RenderMarkdown(tl)); !strings.Contains(rendered, "  - "+tc.line+"\n") {
				t.Errorf("line changed on rewrite, want %q in:\n%s", tc.line, rendered)
			}
		})
	}
}

// A reference ending in a backslash that is followed by another reference
// would be read back as an escaped comma, so it is rejected before it is
// written. As the last reference it is unambiguous and round-trips.
func TestReferencesTrailingBackslash(t *testing.T) {
	tests := map[string]struct {
		refs    []string
		wantErr bool
	}{
		"first of two":    {refs: []string{`dir\`, "b.md"}, wantErr: true},
		"middle of three": {refs: []string{"a.md", `dir\`, "b.md"}, wantErr: true},
		"only reference":  {refs: []string{`docs\`}},
		"last reference":  {refs: []string{"a.md", `docs\`}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl := NewTaskList("T")
			if _, err := tl.AddTask("", "Task", ""); err != nil {
				t.Fatal(err)
			}
			err := tl.UpdateTask("1", "", nil, tc.refs, nil)
			if tc.wantErr {
				if err == nil {
					t.Errorf("references %q accepted; rendered:\n%s", tc.refs, RenderMarkdown(tl))
				}
				return
			}
			if err != nil {
				t.Fatalf("references %q rejected: %v", tc.refs, err)
			}
			if got := roundTrip(t, tl).Tasks[0].References; !slices.Equal(got, tc.refs) {
				t.Errorf("references = %q, want %q", got, tc.refs)
			}
		})
	}
}

// T-1808 via the batch API: a one-element references array must survive a
// write and a re-read as one element.
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
