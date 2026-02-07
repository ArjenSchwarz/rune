package task

import (
	"fmt"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

// TestProperty_ParseRenderRoundTrip tests that parse(render(tasks)) equals original tasks
// This property ensures that rendering and parsing are inverse operations
func TestProperty_ParseRenderRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random task list
		numTasks := rapid.IntRange(1, 10).Draw(t, "numTasks")
		tasks := generateRandomValidTasks(t, numTasks, "")

		// Use a simple valid title (must start with letter, can't be just whitespace)
		title := rapid.StringMatching(`[A-Za-z][A-Za-z0-9 ]*`).Draw(t, "title")
		if title == "" {
			title = "Test List"
		}

		original := &TaskList{
			Title: title,
			Tasks: tasks,
		}

		// Render to markdown
		rendered := RenderMarkdown(original)

		// Parse the rendered markdown
		parsed, err := ParseMarkdown(rendered)
		if err != nil {
			t.Fatalf("ParseMarkdown() error = %v, rendered:\n%s", err, string(rendered))
		}

		// Property: parsed tasks should match original tasks
		if !tasksEqual(original.Tasks, parsed.Tasks) {
			t.Fatalf("Round-trip failed.\nOriginal: %+v\nParsed: %+v\nRendered:\n%s",
				original.Tasks, parsed.Tasks, string(rendered))
		}
	})
}

// TestProperty_RenderStableIDPreserved tests that stable IDs survive round-trip
func TestProperty_RenderStableIDPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate tasks with stable IDs
		numTasks := rapid.IntRange(1, 10).Draw(t, "numTasks")
		tasks := make([]Task, 0, numTasks)

		gen := NewStableIDGenerator([]string{})
		for i := range numTasks {
			stableID, err := gen.Generate()
			if err != nil {
				t.Fatalf("Failed to generate stable ID: %v", err)
			}

			// Generate a simple title (avoid characters that might break parsing)
			title := rapid.StringMatching(`[A-Za-z0-9 ]+`).Draw(t, "title")
			if title == "" {
				title = "Task"
			}

			task := Task{
				ID:       idFromIndex(i + 1),
				Title:    title,
				Status:   Status(rapid.IntRange(0, 2).Draw(t, "status")),
				StableID: stableID,
			}
			tasks = append(tasks, task)
		}

		original := &TaskList{
			Title: "Test Tasks",
			Tasks: tasks,
		}

		// Render and parse
		rendered := RenderMarkdown(original)
		parsed, err := ParseMarkdown(rendered)
		if err != nil {
			t.Fatalf("ParseMarkdown() error = %v", err)
		}

		// Property: all stable IDs should be preserved
		for i, origTask := range original.Tasks {
			if i >= len(parsed.Tasks) {
				t.Fatalf("Task count mismatch: original=%d, parsed=%d", len(original.Tasks), len(parsed.Tasks))
			}
			parsedTask := parsed.Tasks[i]
			if origTask.StableID != parsedTask.StableID {
				t.Fatalf("Stable ID not preserved for task %d: original=%q, parsed=%q",
					i, origTask.StableID, parsedTask.StableID)
			}
		}
	})
}

// TestProperty_RenderBlockedByPreserved tests that BlockedBy references survive round-trip
func TestProperty_RenderBlockedByPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate tasks with dependencies
		numTasks := rapid.IntRange(2, 5).Draw(t, "numTasks")
		tasks := make([]Task, 0, numTasks)

		gen := NewStableIDGenerator([]string{})
		stableIDs := make([]string, numTasks)

		// First, generate all tasks with stable IDs
		for i := range numTasks {
			stableID, _ := gen.Generate()
			stableIDs[i] = stableID

			task := Task{
				ID:       idFromIndex(i + 1),
				Title:    "Task " + idFromIndex(i+1),
				Status:   Status(rapid.IntRange(0, 2).Draw(t, "status")),
				StableID: stableID,
			}
			tasks = append(tasks, task)
		}

		// Add some dependencies (later tasks depend on earlier tasks)
		for i := 1; i < numTasks; i++ {
			// Randomly decide if this task has dependencies
			hasDeps := rapid.Bool().Draw(t, "hasDeps")
			if hasDeps {
				// Pick 1-2 earlier tasks to depend on
				numDeps := rapid.IntRange(1, min(2, i)).Draw(t, "numDeps")
				deps := make([]string, 0, numDeps)
				for range numDeps {
					depIdx := rapid.IntRange(0, i-1).Draw(t, "depIdx")
					deps = append(deps, stableIDs[depIdx])
				}
				tasks[i].BlockedBy = deps
			}
		}

		original := &TaskList{
			Title: "Dependency Test",
			Tasks: tasks,
		}

		// Render and parse
		rendered := RenderMarkdown(original)
		parsed, err := ParseMarkdown(rendered)
		if err != nil {
			t.Fatalf("ParseMarkdown() error = %v", err)
		}

		// Property: all BlockedBy references should be preserved
		for i, origTask := range original.Tasks {
			if i >= len(parsed.Tasks) {
				t.Fatalf("Task count mismatch")
			}
			parsedTask := parsed.Tasks[i]
			if !slicesEqual(origTask.BlockedBy, parsedTask.BlockedBy) {
				t.Fatalf("BlockedBy not preserved for task %d: original=%v, parsed=%v\nRendered:\n%s",
					i, origTask.BlockedBy, parsedTask.BlockedBy, string(rendered))
			}
		}
	})
}

// TestProperty_RenderStreamPreserved tests that Stream values survive round-trip
func TestProperty_RenderStreamPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numTasks := rapid.IntRange(1, 10).Draw(t, "numTasks")
		tasks := make([]Task, 0, numTasks)

		gen := NewStableIDGenerator([]string{})

		for i := range numTasks {
			stableID, _ := gen.Generate()

			// Stream can be 0 (not set), or positive values
			stream := rapid.IntRange(0, 5).Draw(t, "stream")

			task := Task{
				ID:       idFromIndex(i + 1),
				Title:    "Task " + idFromIndex(i+1),
				Status:   Status(rapid.IntRange(0, 2).Draw(t, "status")),
				StableID: stableID,
				Stream:   stream,
			}
			tasks = append(tasks, task)
		}

		original := &TaskList{
			Title: "Stream Test",
			Tasks: tasks,
		}

		// Render and parse
		rendered := RenderMarkdown(original)
		parsed, err := ParseMarkdown(rendered)
		if err != nil {
			t.Fatalf("ParseMarkdown() error = %v", err)
		}

		// Property: Stream values > 0 should be preserved, 0 remains 0 (not rendered)
		for i, origTask := range original.Tasks {
			if i >= len(parsed.Tasks) {
				t.Fatalf("Task count mismatch")
			}
			parsedTask := parsed.Tasks[i]
			if origTask.Stream != parsedTask.Stream {
				t.Fatalf("Stream not preserved for task %d: original=%d, parsed=%d",
					i, origTask.Stream, parsedTask.Stream)
			}
		}
	})
}

// TestProperty_RenderOwnerPreserved tests that Owner values survive round-trip
func TestProperty_RenderOwnerPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numTasks := rapid.IntRange(1, 10).Draw(t, "numTasks")
		tasks := make([]Task, 0, numTasks)

		gen := NewStableIDGenerator([]string{})

		for i := range numTasks {
			stableID, _ := gen.Generate()

			// Owner is either empty or a simple alphanumeric string
			hasOwner := rapid.Bool().Draw(t, "hasOwner")
			var owner string
			if hasOwner {
				owner = rapid.StringMatching(`[a-z0-9-]+`).Draw(t, "owner")
			}

			task := Task{
				ID:       idFromIndex(i + 1),
				Title:    "Task " + idFromIndex(i+1),
				Status:   Status(rapid.IntRange(0, 2).Draw(t, "status")),
				StableID: stableID,
				Owner:    owner,
			}
			tasks = append(tasks, task)
		}

		original := &TaskList{
			Title: "Owner Test",
			Tasks: tasks,
		}

		// Render and parse
		rendered := RenderMarkdown(original)
		parsed, err := ParseMarkdown(rendered)
		if err != nil {
			t.Fatalf("ParseMarkdown() error = %v", err)
		}

		// Property: Owner values should be preserved
		for i, origTask := range original.Tasks {
			if i >= len(parsed.Tasks) {
				t.Fatalf("Task count mismatch")
			}
			parsedTask := parsed.Tasks[i]
			if origTask.Owner != parsedTask.Owner {
				t.Fatalf("Owner not preserved for task %d: original=%q, parsed=%q",
					i, origTask.Owner, parsedTask.Owner)
			}
		}
	})
}

// TestProperty_RenderAllMetadataPreserved tests that all metadata survives round-trip together
func TestProperty_RenderAllMetadataPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numTasks := rapid.IntRange(2, 5).Draw(t, "numTasks")
		tasks := make([]Task, 0, numTasks)

		gen := NewStableIDGenerator([]string{})
		stableIDs := make([]string, numTasks)

		// First pass: create tasks with stable IDs
		for i := range numTasks {
			stableID, _ := gen.Generate()
			stableIDs[i] = stableID
		}

		// Second pass: create full tasks with all metadata
		for i := range numTasks {
			// Generate details
			numDetails := rapid.IntRange(0, 3).Draw(t, "numDetails")
			details := make([]string, 0, numDetails)
			for range numDetails {
				detail := rapid.StringMatching(`[A-Za-z0-9 ]+`).Draw(t, "detail")
				if detail != "" {
					details = append(details, detail)
				}
			}

			// Generate BlockedBy (only for tasks after first)
			var blockedBy []string
			if i > 0 && rapid.Bool().Draw(t, "hasBlockedBy") {
				depIdx := rapid.IntRange(0, i-1).Draw(t, "depIdx")
				blockedBy = []string{stableIDs[depIdx]}
			}

			// Stream: 0-3 (0 means not set)
			stream := rapid.IntRange(0, 3).Draw(t, "stream")

			// Owner
			var owner string
			if rapid.Bool().Draw(t, "hasOwner") {
				owner = rapid.StringMatching(`[a-z0-9-]+`).Draw(t, "owner")
			}

			task := Task{
				ID:        idFromIndex(i + 1),
				Title:     "Task " + idFromIndex(i+1),
				Status:    Status(rapid.IntRange(0, 2).Draw(t, "status")),
				StableID:  stableIDs[i],
				Details:   details,
				BlockedBy: blockedBy,
				Stream:    stream,
				Owner:     owner,
			}
			tasks = append(tasks, task)
		}

		original := &TaskList{
			Title: "Full Metadata Test",
			Tasks: tasks,
		}

		// Render and parse
		rendered := RenderMarkdown(original)
		parsed, err := ParseMarkdown(rendered)
		if err != nil {
			t.Fatalf("ParseMarkdown() error = %v", err)
		}

		// Property: all fields should match
		for i, origTask := range original.Tasks {
			if i >= len(parsed.Tasks) {
				t.Fatalf("Task count mismatch: original=%d, parsed=%d", len(original.Tasks), len(parsed.Tasks))
			}
			parsedTask := parsed.Tasks[i]

			if origTask.ID != parsedTask.ID {
				t.Fatalf("ID mismatch for task %d", i)
			}
			if origTask.Title != parsedTask.Title {
				t.Fatalf("Title mismatch for task %d: orig=%q, parsed=%q", i, origTask.Title, parsedTask.Title)
			}
			if origTask.Status != parsedTask.Status {
				t.Fatalf("Status mismatch for task %d", i)
			}
			if origTask.StableID != parsedTask.StableID {
				t.Fatalf("StableID mismatch for task %d", i)
			}
			if !slicesEqual(origTask.Details, parsedTask.Details) {
				t.Fatalf("Details mismatch for task %d: orig=%v, parsed=%v", i, origTask.Details, parsedTask.Details)
			}
			if !slicesEqual(origTask.BlockedBy, parsedTask.BlockedBy) {
				t.Fatalf("BlockedBy mismatch for task %d: orig=%v, parsed=%v\nRendered:\n%s", i, origTask.BlockedBy, parsedTask.BlockedBy, string(rendered))
			}
			if origTask.Stream != parsedTask.Stream {
				t.Fatalf("Stream mismatch for task %d: orig=%d, parsed=%d", i, origTask.Stream, parsedTask.Stream)
			}
			if origTask.Owner != parsedTask.Owner {
				t.Fatalf("Owner mismatch for task %d: orig=%q, parsed=%q", i, origTask.Owner, parsedTask.Owner)
			}
		}
	})
}

// Helper functions

func generateRandomTasks(t *rapid.T, numTasks int, parentID string) []Task {
	tasks := make([]Task, 0, numTasks)
	gen := NewStableIDGenerator([]string{})

	for i := range numTasks {
		stableID, _ := gen.Generate()

		// Generate a simple title (avoid characters that might break parsing)
		title := rapid.StringMatching(`[A-Za-z0-9 ]+`).Draw(t, "title")
		if title == "" {
			title = "Task"
		}

		id := idFromIndex(i + 1)
		if parentID != "" {
			id = parentID + "." + idFromIndex(i+1)
		}

		task := Task{
			ID:       id,
			Title:    title,
			Status:   Status(rapid.IntRange(0, 2).Draw(t, "status")),
			StableID: stableID,
			ParentID: parentID,
		}
		tasks = append(tasks, task)
	}

	return tasks
}

// generateRandomValidTasks creates tasks with valid titles (non-empty, no special chars)
func generateRandomValidTasks(t *rapid.T, numTasks int, parentID string) []Task {
	tasks := make([]Task, 0, numTasks)
	gen := NewStableIDGenerator([]string{})

	for i := range numTasks {
		stableID, _ := gen.Generate()

		// Generate a valid title that will round-trip properly
		// Must be non-empty and contain at least one non-whitespace character
		// Use TrimRight to match parser behavior (parser trims trailing whitespace)
		title := rapid.StringMatching(`[A-Za-z][A-Za-z0-9]*`).Draw(t, "title")
		if title == "" {
			title = "Task"
		}

		id := idFromIndex(i + 1)
		if parentID != "" {
			id = parentID + "." + idFromIndex(i+1)
		}

		task := Task{
			ID:       id,
			Title:    title,
			Status:   Status(rapid.IntRange(0, 2).Draw(t, "status")),
			StableID: stableID,
			ParentID: parentID,
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func idFromIndex(i int) string {
	return fmt.Sprintf("%d", i)
}

func tasksEqual(a, b []Task) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !taskEqual(&a[i], &b[i]) {
			return false
		}
	}
	return true
}

func taskEqual(a, b *Task) bool {
	if a.ID != b.ID || a.Title != b.Title || a.Status != b.Status {
		return false
	}
	if a.StableID != b.StableID {
		return false
	}
	if !slicesEqual(a.Details, b.Details) {
		return false
	}
	if !slicesEqual(a.BlockedBy, b.BlockedBy) {
		return false
	}
	if a.Stream != b.Stream {
		return false
	}
	if a.Owner != b.Owner {
		return false
	}
	if !slicesEqual(a.References, b.References) {
		return false
	}
	if !slicesEqual(a.Requirements, b.Requirements) {
		return false
	}
	if !tasksEqual(a.Children, b.Children) {
		return false
	}
	return true
}

func slicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}
