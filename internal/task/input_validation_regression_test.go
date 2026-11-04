package task

import (
	"errors"
	"strings"
	"testing"
)

// T-1782: an owner that is blank or has surrounding whitespace cannot survive
// a write/parse round trip, so every path that sets an owner rejects it.
func TestOwnerRejectsBlankAndSurroundingWhitespace(t *testing.T) {
	owners := map[string]string{
		"whitespace only":     "   ",
		"leading whitespace":  " agent",
		"trailing whitespace": "agent ",
	}
	paths := map[string]func(tl *TaskList, owner string) error{
		"add": func(tl *TaskList, owner string) error {
			_, err := tl.AddTaskWithOptions("", "Task", AddOptions{Owner: owner})
			return err
		},
		"update": func(tl *TaskList, owner string) error {
			return tl.UpdateTaskWithOptions("1", UpdateOptions{Owner: &owner})
		},
		"batch add": func(tl *TaskList, owner string) error {
			return batchError(tl, Operation{Type: addOperation, Title: "Task", Owner: &owner})
		},
		"batch update": func(tl *TaskList, owner string) error {
			return batchError(tl, Operation{Type: updateOperation, ID: "1", Owner: &owner})
		},
		"phased batch add": func(tl *TaskList, owner string) error {
			resp, err := tl.ExecuteBatchWithPhases([]Operation{{Type: addOperation, Title: "Task", Phase: "Work", Owner: &owner}}, true, nil, "tasks.md")
			if err != nil {
				return err
			}
			if !resp.Success {
				return errors.New(strings.Join(resp.Errors, "; "))
			}
			return nil
		},
		"claim": func(_ *TaskList, owner string) error {
			return ValidateOwner(owner)
		},
	}

	for ownerName, owner := range owners {
		for pathName, set := range paths {
			t.Run(pathName+"/"+ownerName, func(t *testing.T) {
				tl := NewTaskList("Tasks")
				if _, err := tl.AddTask("", "Existing", ""); err != nil {
					t.Fatal(err)
				}
				err := set(tl, owner)
				if err == nil {
					t.Fatalf("owner %q was accepted", owner)
				}
				if !strings.Contains(err.Error(), ErrInvalidOwner.Error()) {
					t.Errorf("error = %v, want it to report %v", err, ErrInvalidOwner)
				}
			})
		}
	}
}

func batchError(tl *TaskList, op Operation) error {
	resp, err := tl.ExecuteBatch([]Operation{op}, true)
	if err != nil {
		return err
	}
	if !resp.Success {
		return errors.New(strings.Join(resp.Errors, "; "))
	}
	return nil
}

func TestOwnerWithInnerSpacesRoundTrips(t *testing.T) {
	tl := NewTaskList("Tasks")
	if _, err := tl.AddTaskWithOptions("", "Task", AddOptions{Owner: "agent one"}); err != nil {
		t.Fatalf("AddTaskWithOptions: %v", err)
	}
	parsed, err := ParseMarkdown(RenderMarkdown(tl))
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if got := parsed.Tasks[0]; got.Owner != "agent one" || len(got.Details) != 0 {
		t.Errorf("owner = %q, details = %q; want owner %q and no details", got.Owner, got.Details, "agent one")
	}
}

// T-1793: limits documented in characters count characters, not UTF-8 bytes.
func TestLengthLimitsCountCharacters(t *testing.T) {
	multiByte := func(n int) string { return strings.Repeat("é", n) }

	tests := map[string]struct {
		check func(s string) error
		limit int
	}{
		"task title": {
			check: func(s string) error {
				_, err := NewTaskList("Tasks").AddTask("", s, "")
				return err
			},
			limit: MaxTitleLength,
		},
		"task list title": {check: ValidateTaskListTitle, limit: MaxTitleLength},
		"task validate": {
			check: func(s string) error { return (&Task{ID: "1", Title: s}).Validate() },
			limit: MaxTitleLength,
		},
		"batch add title": {
			check: func(s string) error {
				return batchError(NewTaskList("Tasks"), Operation{Type: addOperation, Title: s})
			},
			limit: MaxTitleLength,
		},
		"batch update title": {
			check: func(s string) error {
				tl := NewTaskList("Tasks")
				tl.AddTask("", "Existing", "")
				return batchError(tl, Operation{Type: updateOperation, ID: "1", Title: s})
			},
			limit: MaxTitleLength,
		},
		"detail": {
			check: func(s string) error { return validateDetails([]string{s}) },
			limit: MaxDetailLength,
		},
		"reference": {
			check: func(s string) error { return validateReferences([]string{s}) },
			limit: 500,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if err := tc.check(multiByte(tc.limit)); err != nil {
				t.Errorf("%d multi-byte characters rejected: %v", tc.limit, err)
			}
			if err := tc.check(multiByte(tc.limit + 1)); err == nil {
				t.Errorf("%d multi-byte characters accepted", tc.limit+1)
			}
		})
	}
}
