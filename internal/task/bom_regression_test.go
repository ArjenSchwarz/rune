package task

import (
	"slices"
	"testing"
)

// T-2063: a leading UTF-8 byte-order mark prevented title and front matter
// detection, so the file failed to parse at line 1.
func TestParseMarkdownStripsUTF8BOM(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	tests := map[string]struct {
		content   string
		wantTitle string
		wantRefs  []string
	}{
		"h1 title":          {content: bom + "# BOM Project\n\n- [ ] 1. A\n", wantTitle: "BOM Project"},
		"crlf h1 title":     {content: bom + "# BOM Project\r\n\r\n- [ ] 1. A\r\n", wantTitle: "BOM Project"},
		"front matter":      {content: bom + "---\nreferences:\n  - a.md\n---\n# FM\n\n- [ ] 1. A\n", wantTitle: "FM", wantRefs: []string{"a.md"}},
		"crlf front matter": {content: bom + "---\r\nreferences:\r\n  - a.md\r\n---\r\n# FM\r\n\r\n- [ ] 1. A\r\n", wantTitle: "FM", wantRefs: []string{"a.md"}},
		"no title":          {content: bom + "- [ ] 1. A\n"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, err := ParseMarkdown([]byte(tc.content))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if tl.Title != tc.wantTitle {
				t.Errorf("title = %q, want %q", tl.Title, tc.wantTitle)
			}
			if len(tl.Tasks) != 1 || tl.Tasks[0].Title != "A" {
				t.Errorf("tasks = %+v", tl.Tasks)
			}
			var refs []string
			if tl.FrontMatter != nil {
				refs = tl.FrontMatter.References
			}
			if !slices.Equal(refs, tc.wantRefs) {
				t.Errorf("front matter references = %q, want %q", refs, tc.wantRefs)
			}
		})
	}

	// Only one BOM is stripped; a second one is content and is rejected.
	if _, err := ParseMarkdown([]byte(bom + bom + "# T\n")); err == nil {
		t.Error("expected error for a doubled BOM")
	}
}

// Phase markers are read from raw file bytes; a BOM must not hide the front
// matter delimiter there either.
func TestPhaseMarkersIgnoreUTF8BOM(t *testing.T) {
	content := "\xef\xbb\xbf---\n## not a phase\nmetadata:\n  x: 1\n---\n# T\n\n## Real\n\n- [ ] 1. A\n"
	markers := phaseMarkersFromContent([]byte(content))
	if len(markers) != 1 || markers[0].Name != "Real" {
		t.Errorf("markers = %+v, want only Real", markers)
	}
}
