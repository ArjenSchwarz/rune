package task

import (
	"slices"
	"testing"
)

// T-1999: a detail that looks like a task line was written verbatim and read
// back as a child task. Expected: such details are rejected before writing.
func TestTaskShapedDetailsRejected(t *testing.T) {
	tests := map[string]string{
		"pending task":       "[ ] 99. Injected child",
		"completed task":     "[x] 1. Done",
		"in-progress task":   "[-] 2.1. Busy",
		"missing number":     "[ ] no number",
		"invalid checkbox":   "[y] 1. Odd",
		"malformed checkbox": "[] 1. Odd",
		// The parser treats any "- [..]" line as task syntax, so these would
		// make the written file unparsable.
		"leading link":  "[link](x.md) text",
		"bare checkbox": "[ ]",
	}
	for name, detail := range tests {
		t.Run(name, func(t *testing.T) {
			tl := NewTaskList("T")
			if _, err := tl.AddTask("", "Parent", ""); err != nil {
				t.Fatal(err)
			}
			if err := tl.UpdateTask("1", "", []string{detail}, nil, nil); err == nil {
				t.Errorf("detail %q accepted; rendered:\n%s", detail, RenderMarkdown(tl))
			}
		})
	}

	// Details that merely contain brackets still round-trip.
	tl := NewTaskList("T")
	if _, err := tl.AddTask("", "Parent", ""); err != nil {
		t.Fatal(err)
	}
	ok := []string{"see [link](x.md)", "see [ ] boxes"}
	if err := tl.UpdateTask("1", "", ok, nil, nil); err != nil {
		t.Fatalf("plain details rejected: %v", err)
	}
	got := roundTrip(t, tl)
	if !slices.Equal(got.Tasks[0].Details, ok) || len(got.Tasks[0].Children) != 0 {
		t.Errorf("details = %q children = %d", got.Tasks[0].Details, len(got.Tasks[0].Children))
	}
}
