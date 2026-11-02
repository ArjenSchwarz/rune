package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
	"github.com/spf13/cobra"
)

// newUpdateTestCmd returns a command carrying the update flags bound to the
// package globals, with the given flags explicitly set (so Changed is true).
func newUpdateTestCmd(t *testing.T, flags map[string]string) *cobra.Command {
	t.Helper()
	resetUpdateFlags()
	t.Cleanup(resetUpdateFlags)

	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&updateTitle, "title", "", "")
	cmd.Flags().StringVar(&updateRequirements, "requirements", "", "")
	cmd.Flags().IntVar(&updateStream, "stream", 0, "")
	cmd.Flags().StringVar(&updateBlockedBy, "blocked-by", "", "")
	cmd.Flags().StringVar(&updateOwner, "owner", "", "")
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s: %v", name, err)
		}
	}
	return cmd
}

// T-1381: update --dry-run must preview stream, blocked-by and owner changes.
func TestUpdateDryRunPreviewsExtendedFields(t *testing.T) {
	file := writeTemp(t, "# Demo\n\n- [ ] 1. Blocker\n- [ ] 2. Target\n")
	before := readFile(t, file)

	cmd := newUpdateTestCmd(t, map[string]string{"stream": "3", "blocked-by": "1", "owner": "agent-a"})
	dryRun = true

	var runErr error
	out := captureStdout(t, func() { runErr = runUpdate(cmd, []string{file, "2"}) })
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	for _, want := range []string{"New stream: 3", "New blocked-by: 1", "New owner: agent-a"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, out)
		}
	}
	if !bytes.Equal(before, readFile(t, file)) {
		t.Error("dry run modified the file")
	}
}

// T-1750: update --dry-run must run the same validation as a real update.
func TestUpdateDryRunValidatesDependencies(t *testing.T) {
	tests := map[string]map[string]string{
		"unknown blocker": {"blocked-by": "999"},
		"negative stream": {"stream": "-1"},
	}
	for name, flags := range tests {
		t.Run(name, func(t *testing.T) {
			file := writeTemp(t, "# Repro\n\n- [ ] 1. Root\n")
			cmd := newUpdateTestCmd(t, flags)
			dryRun = true

			var runErr error
			captureStdout(t, func() { runErr = runUpdate(cmd, []string{file, "1"}) })
			if runErr == nil {
				t.Fatal("expected dry run to fail validation, got success")
			}
		})
	}
}

// T-1750 (add side, fixed by #104): add --dry-run validates --blocked-by.
func TestAddDryRunValidatesDependencies(t *testing.T) {
	file := writeTemp(t, "# Repro\n\n- [ ] 1. Root\n")
	resetAddFlags(t)
	addTitle = "Child"
	addBlockedBy = "999"
	dryRun = true

	var runErr error
	captureStdout(t, func() { runErr = runAdd(&cobra.Command{}, []string{file}) })
	if runErr == nil {
		t.Fatal("expected dry run to fail validation, got success")
	}
}

// T-1493: an explicit empty --blocked-by clears dependencies.
func TestUpdateEmptyBlockedByClears(t *testing.T) {
	tests := map[string]map[string]string{
		"empty blocked-by only":       {"blocked-by": ""},
		"empty blocked-by with title": {"blocked-by": "", "title": "Renamed"},
	}
	for name, flags := range tests {
		t.Run(name, func(t *testing.T) {
			file := writeTemp(t, "# Demo\n\n- [ ] 1. Blocker <!-- id:abc1234 -->\n- [ ] 2. Dependent <!-- id:def5678 -->\n  - Blocked-by: abc1234 (Blocker)\n")
			cmd := newUpdateTestCmd(t, flags)
			format = formatJSON

			var runErr error
			out := captureStdout(t, func() { runErr = runUpdate(cmd, []string{file, "2"}) })
			if runErr != nil {
				t.Fatalf("unexpected error: %v", runErr)
			}
			if !strings.Contains(out, `"blocked-by"`) {
				t.Errorf("fields_updated does not report blocked-by:\n%s", out)
			}
			tl, err := task.ParseFile(file)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := tl.FindTask("2").BlockedBy; len(got) != 0 {
				t.Errorf("blocked-by not cleared: %v", got)
			}
		})
	}
}

// T-2023: comma/whitespace-only --requirements is rejected, file untouched.
func TestCommaOnlyRequirementsRejected(t *testing.T) {
	values := map[string]string{
		"single comma":    ",",
		"spaced comma":    " , ",
		"repeated commas": ",,,",
	}
	for name, value := range values {
		t.Run("update "+name, func(t *testing.T) {
			file := writeTemp(t, "# Requirements Repro\n\n- [ ] 1. Linked task\n  - Requirements: [1.1](requirements.md#1.1)\n")
			before := readFile(t, file)
			cmd := newUpdateTestCmd(t, map[string]string{"requirements": value})

			if err := runUpdate(cmd, []string{file, "1"}); err == nil {
				t.Fatal("expected error for comma-only --requirements")
			}
			if !bytes.Equal(before, readFile(t, file)) {
				t.Error("rejected update modified the file")
			}
		})
		t.Run("add "+name, func(t *testing.T) {
			file := writeTemp(t, "# Requirements Repro\n\n- [ ] 1. Existing\n")
			before := readFile(t, file)
			resetAddFlags(t)
			addTitle = "New"
			addRequirements = value

			if err := runAdd(&cobra.Command{}, []string{file}); err == nil {
				t.Fatal("expected error for comma-only --requirements")
			}
			if !bytes.Equal(before, readFile(t, file)) {
				t.Error("rejected add modified the file")
			}
		})
	}
}
