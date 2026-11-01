package task

import (
	"reflect"
	"strings"
	"testing"
)

// phasedWithDotlessSubtask is a valid file whose subtask is hand-written with
// a dotless raw ID. The parser nests it by indentation and numbers it 1.1;
// every phase scanner must agree that it is not a top-level task.
const phasedWithDotlessSubtask = `# T

## A

- [x] 1. a1
  - [x] 1. sub

## B

- [ ] 2. b1

- [ ] 3. b2
`

func TestPhaseScannersIgnoreIndentedTaskLines(t *testing.T) {
	lines := splitLines(phasedWithDotlessSubtask)

	wantMarkers := []PhaseMarker{{Name: "A", AfterTaskID: ""}, {Name: "B", AfterTaskID: "1"}}
	if got := ExtractPhaseMarkers(lines); !reflect.DeepEqual(got, wantMarkers) {
		t.Errorf("ExtractPhaseMarkers() = %v, want %v", got, wantMarkers)
	}

	wantMap := map[string]string{"1": "A", "2": "B", "3": "B"}
	if got := buildTaskPhaseMap(lines); !reflect.DeepEqual(got, wantMap) {
		t.Errorf("buildTaskPhaseMap() = %v, want %v", got, wantMap)
	}

	if got := getTaskPhase("2", []byte(phasedWithDotlessSubtask)); got != "B" {
		t.Errorf("getTaskPhase(2) = %q, want B", got)
	}

	tl, err := ParseMarkdown([]byte(phasedWithDotlessSubtask))
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	phases := extractPhasesWithTaskRanges(lines, tl.Tasks)
	var got []string
	for _, p := range phases {
		titles := make([]string, 0, len(p.Tasks))
		for _, task := range p.Tasks {
			titles = append(titles, task.Title)
		}
		got = append(got, p.Name+":"+strings.Join(titles, ","))
	}
	want := []string{"A:a1", "B:b1,b2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("extractPhasesWithTaskRanges() = %v, want %v", got, want)
	}
}

func TestPhaseScannersCountUnindentedDottedIDAsTopLevel(t *testing.T) {
	// The parser makes any unindented task line top-level, whatever its raw ID.
	content := "# T\n\n## A\n\n- [ ] 1.1. first\n\n## B\n\n- [ ] 2. second\n"
	want := []PhaseMarker{{Name: "A", AfterTaskID: ""}, {Name: "B", AfterTaskID: "1"}}
	if got := ExtractPhaseMarkers(splitLines(content)); !reflect.DeepEqual(got, want) {
		t.Errorf("ExtractPhaseMarkers() = %v, want %v", got, want)
	}
}

func TestPhaseMarkersFromContentSkipsFrontMatter(t *testing.T) {
	content := "---\nmetadata:\n  a: b\n## a yaml comment\n---\n# T\n\n## A\n\n- [ ] 1. a\n"
	want := []PhaseMarker{{Name: "A", AfterTaskID: ""}}
	if got := phaseMarkersFromContent([]byte(content)); !reflect.DeepEqual(got, want) {
		t.Errorf("phaseMarkersFromContent() = %v, want %v", got, want)
	}
}

// threePhaseList returns A:[a1,a2] B:[b1] with its markers.
func threePhaseList(t *testing.T) (*TaskList, []PhaseMarker) {
	t.Helper()
	content := "# T\n\n## A\n\n- [ ] 1. a1\n\n- [ ] 2. a2\n\n## B\n\n- [ ] 3. b1\n"
	tl, err := ParseMarkdown([]byte(content))
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	return tl, ExtractPhaseMarkers(splitLines(content))
}

// phaseLayout renders tasks grouped by phase, e.g. "A:a1,a2 B:b1".
func phaseLayout(tl *TaskList, markers []PhaseMarker) string {
	tasksByPhase := map[string][]string{}
	for _, task := range tl.Tasks {
		phase := GetTaskPhase(tl, markers, task.ID)
		tasksByPhase[phase] = append(tasksByPhase[phase], task.Title)
	}
	parts := make([]string, 0, len(markers))
	for _, m := range markers {
		parts = append(parts, m.Name+":"+strings.Join(tasksByPhase[m.Name], ","))
	}
	return strings.Join(parts, " ")
}

func TestAddTaskWithPhasesPositionKeepsPhaseBoundaries(t *testing.T) {
	tests := map[string]struct {
		position string
		wantID   string
		want     string
	}{
		"start of first phase":  {position: "1", wantID: "1", want: "A:new,a1,a2 B:b1"},
		"middle of first phase": {position: "2", wantID: "2", want: "A:a1,new,a2 B:b1"},
		"start of second phase": {position: "3", wantID: "3", want: "A:a1,a2 B:new,b1"},
		"past the end":          {position: "9", wantID: "4", want: "A:a1,a2 B:b1,new"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tl, markers := threePhaseList(t)
			id, err := tl.AddTaskWithPhases(Operation{Type: addOperation, Title: "new", Position: tc.position}, &markers)
			if err != nil {
				t.Fatalf("AddTaskWithPhases: %v", err)
			}
			if id != tc.wantID {
				t.Errorf("id = %q, want %q", id, tc.wantID)
			}
			if got := phaseLayout(tl, markers); got != tc.want {
				t.Errorf("layout = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAddTaskWithPhasesHonoursAllFieldsInPhase(t *testing.T) {
	tl, markers := threePhaseList(t)
	stream := 2
	owner := "bob"
	id, err := tl.AddTaskWithPhases(Operation{
		Type:         addOperation,
		Title:        "new",
		Phase:        "A",
		Stream:       &stream,
		Owner:        &owner,
		BlockedBy:    []string{"1"},
		Details:      []string{"a detail"},
		References:   []string{"ref.md"},
		Requirements: []string{"1.1"},
	}, &markers)
	if err != nil {
		t.Fatalf("AddTaskWithPhases: %v", err)
	}
	if id != "3" {
		t.Fatalf("id = %q, want 3", id)
	}
	if got, want := phaseLayout(tl, markers), "A:a1,a2,new B:b1"; got != want {
		t.Errorf("layout = %q, want %q", got, want)
	}
	task := tl.FindTask(id)
	blocker := tl.FindTask("1")
	if task.Stream != 2 || task.Owner != "bob" || task.StableID == "" ||
		!reflect.DeepEqual(task.BlockedBy, []string{blocker.StableID}) ||
		!reflect.DeepEqual(task.Details, []string{"a detail"}) ||
		!reflect.DeepEqual(task.References, []string{"ref.md"}) ||
		!reflect.DeepEqual(task.Requirements, []string{"1.1"}) {
		t.Errorf("fields not applied: %+v", *task)
	}
}

func TestAddTaskWithPhasesSubtaskNeverCreatesPhase(t *testing.T) {
	tl, markers := threePhaseList(t)
	id, err := tl.AddTaskWithPhases(Operation{Type: addOperation, Title: "child", Parent: "1", Phase: "Nope"}, &markers)
	if err != nil {
		t.Fatalf("AddTaskWithPhases: %v", err)
	}
	if id != "1.1" {
		t.Errorf("id = %q, want 1.1", id)
	}
	if len(markers) != 2 {
		t.Errorf("subtask add created a phase: %v", markers)
	}
}

func TestValidateDetailsRejectsMetadataLookalikes(t *testing.T) {
	tests := map[string]struct {
		detail  string
		wantErr bool
	}{
		"stream":                 {detail: "Stream: 3", wantErr: true},
		"stream lowercase":       {detail: "stream: 3", wantErr: true},
		"owner":                  {detail: "Owner: mallory", wantErr: true},
		"blocked-by":             {detail: "Blocked-by: abc1234", wantErr: true},
		"references":             {detail: "References: foo", wantErr: true},
		"requirements link":      {detail: "Requirements: [1.1](requirements.md#1.1)", wantErr: true},
		"stream not a number":    {detail: "Stream: soon"},
		"blocked-by no ids":      {detail: "Blocked-by: the vendor"},
		"requirements free text": {detail: "Requirements: see the spec"},
		"plain":                  {detail: "Owner of the API is unclear"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateDetails([]string{tc.detail})
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateDetails(%q) error = %v, wantErr %v", tc.detail, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			// Accepted details must round-trip as details
			tl := NewTaskList("T")
			if _, err := tl.AddTask("", "x", ""); err != nil {
				t.Fatal(err)
			}
			if err := tl.UpdateTask("1", "", []string{tc.detail}, nil, nil); err != nil {
				t.Fatal(err)
			}
			parsed, err := ParseMarkdown(RenderMarkdown(tl))
			if err != nil {
				t.Fatal(err)
			}
			if got := parsed.Tasks[0].Details; !reflect.DeepEqual(got, []string{tc.detail}) {
				t.Errorf("round-tripped details = %v, want [%q]", got, tc.detail)
			}
		})
	}
}

func TestExecuteBatchWithPhasesPositionalAddKeepsPhaseBoundaries(t *testing.T) {
	tl, markers := threePhaseList(t)
	resp, err := tl.ExecuteBatchWithPhases([]Operation{{Type: addOperation, Title: "new", Position: "1"}}, true, markers, "")
	if err != nil || !resp.Success {
		t.Fatalf("ExecuteBatchWithPhases: %v %v", err, resp.Errors)
	}
	parsed, err := ParseMarkdown([]byte(resp.Preview))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := phaseLayout(parsed, ExtractPhaseMarkers(splitLines(resp.Preview))), "A:new,a1,a2 B:b1"; got != want {
		t.Errorf("layout = %q, want %q", got, want)
	}
}

func TestExecuteBatchAppliesOnceAndIsAtomic(t *testing.T) {
	tl := NewTaskList("T")
	for _, title := range []string{"a", "b"} {
		if _, err := tl.AddTask("", title, ""); err != nil {
			t.Fatal(err)
		}
	}
	before := string(RenderMarkdown(tl))

	// A failing batch leaves the original untouched
	resp, err := tl.ExecuteBatch([]Operation{
		{Type: updateOperation, ID: "1", Title: "changed"},
		{Type: removeOperation, ID: "9"},
	}, false)
	if err != nil || resp.Success {
		t.Fatalf("expected failed response, got %+v, %v", resp, err)
	}
	if got := string(RenderMarkdown(tl)); got != before {
		t.Errorf("failed batch modified the list:\n%s", got)
	}

	// A successful batch is adopted as a whole, including a detail that a
	// render/parse based copy used to rewrite
	resp, err = tl.ExecuteBatch([]Operation{
		{Type: updateOperation, ID: "1", Title: "changed", Details: []string{"keep me"}},
		{Type: addOperation, Title: "c", Parent: "2"},
	}, false)
	if err != nil || !resp.Success || resp.Applied != 2 {
		t.Fatalf("ExecuteBatch: %+v, %v", resp, err)
	}
	if tl.Tasks[0].Title != "changed" || !reflect.DeepEqual(tl.Tasks[0].Details, []string{"keep me"}) {
		t.Errorf("update not applied: %+v", tl.Tasks[0])
	}
	if len(tl.Tasks[1].Children) != 1 {
		t.Errorf("add not applied exactly once: %+v", tl.Tasks[1].Children)
	}
}

func TestDeepCopySharesNoState(t *testing.T) {
	tl := NewTaskList("T", &FrontMatter{References: []string{"r"}, Metadata: map[string]string{"k": "v"}})
	if _, err := tl.AddTask("", "a", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := tl.AddTask("1", "child", ""); err != nil {
		t.Fatal(err)
	}
	tl.Tasks[0].Details = []string{"d"}
	tl.Tasks[0].BlockedBy = []string{"abc1234"}

	c := tl.deepCopy()
	c.Tasks[0].Details[0] = "changed"
	c.Tasks[0].BlockedBy[0] = "zzzzzzz"
	c.Tasks[0].Children[0].Title = "changed"
	c.FrontMatter.References[0] = "changed"
	c.FrontMatter.Metadata["k"] = "changed"

	if tl.Tasks[0].Details[0] != "d" || tl.Tasks[0].BlockedBy[0] != "abc1234" ||
		tl.Tasks[0].Children[0].Title != "child" || tl.FrontMatter.References[0] != "r" ||
		tl.FrontMatter.Metadata["k"] != "v" {
		t.Errorf("deepCopy shares state with the original")
	}
}
