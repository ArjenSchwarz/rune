package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

const inputValidationFixture = "# Tasks\n\n- [ ] 1. First\n"

// writeInputValidationFixture writes content to tasks.md in a fresh working
// directory and returns the file name.
func writeInputValidationFixture(t *testing.T, content string) string {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile("tasks.md", []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return "tasks.md"
}

// T-1243: every command rejects an unsupported --format before doing anything.
func TestUnsupportedFormatIsRejected(t *testing.T) {
	tests := map[string][]string{
		"list":  {"list", "tasks.md"},
		"add":   {"add", "tasks.md", "--title", "Second"},
		"next":  {"next", "tasks.md"},
		"batch": {"batch", "tasks.md", "--input", `{"operations":[{"type":"add","title":"Second"}]}`},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			resetOutputContextFlags(t)
			t.Cleanup(resetBatchFlags)
			file := writeInputValidationFixture(t, inputValidationFixture)

			rootCmd.SetArgs(append(args, "--format", "yaml"))
			var runErr error
			out := captureStdout(t, func() { runErr = rootCmd.Execute() })
			if runErr == nil || !strings.Contains(runErr.Error(), "unsupported output format: yaml") {
				t.Fatalf("error = %v, want unsupported output format", runErr)
			}
			if out != "" {
				t.Errorf("stdout = %q, want nothing", out)
			}
			if got, _ := os.ReadFile(file); string(got) != inputValidationFixture {
				t.Errorf("file changed:\n%s", got)
			}
		})
	}
}

func TestFormatIsCaseInsensitive(t *testing.T) {
	resetOutputContextFlags(t)
	writeInputValidationFixture(t, inputValidationFixture)

	rootCmd.SetArgs([]string{"list", "tasks.md", "--format", "JSON"})
	var runErr error
	out := captureStdout(t, func() { runErr = rootCmd.Execute() })
	if runErr != nil {
		t.Fatalf("list --format JSON: %v", runErr)
	}
	decodeJSON(t, out)
}

// T-1754: a subtask stays under its parent, so the response must not claim
// it went into the requested --phase.
func TestAddSubtaskDoesNotReportRequestedPhase(t *testing.T) {
	const fixture = "# Repro\n\n## Planning\n\n- [ ] 1. Parent in planning\n\n## Build\n\n- [ ] 2. Build task\n"
	for _, dry := range []bool{false, true} {
		t.Run(map[bool]string{false: "live", true: "dry-run"}[dry], func(t *testing.T) {
			resetAddFlags(t)
			resetOutputContextFlags(t)
			writeInputValidationFixture(t, fixture)
			addTitle, addParent, addPhase = "Child request build", "1", "Build"
			format, dryRun = formatJSON, dry

			var runErr error
			out := captureStdout(t, func() { runErr = runAdd(&cobra.Command{}, []string{"tasks.md"}) })
			if runErr != nil {
				t.Fatalf("runAdd: %v", runErr)
			}
			var resp AddResponse
			if err := json.Unmarshal([]byte(out), &resp); err != nil {
				t.Fatalf("invalid JSON %q: %v", out, err)
			}
			if resp.TaskID != "1.1" || resp.Phase != "" {
				t.Errorf("task_id = %q, phase = %q; want 1.1 with no phase", resp.TaskID, resp.Phase)
			}
		})
	}
}

func TestAddTopLevelStillReportsPhase(t *testing.T) {
	resetAddFlags(t)
	resetOutputContextFlags(t)
	writeInputValidationFixture(t, "# T\n\n## Build\n\n- [ ] 1. Build task\n")
	addTitle, addPhase = "New", "Build"
	format = formatJSON

	var runErr error
	out := captureStdout(t, func() { runErr = runAdd(&cobra.Command{}, []string{"tasks.md"}) })
	if runErr != nil {
		t.Fatalf("runAdd: %v", runErr)
	}
	var resp AddResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}
	if resp.Phase != "Build" {
		t.Errorf("phase = %q, want Build", resp.Phase)
	}
}

// T-2145 and T-2325: add flags whose value would be silently dropped are
// rejected without writing the file.
func TestAddRejectsSilentlyDroppedFlags(t *testing.T) {
	tests := map[string]struct {
		set     func()
		wantErr string
	}{
		"requirements-file without requirements": {
			set:     func() { addRequirementsFile = "custom.md" },
			wantErr: "--requirements-file requires --requirements",
		},
		"comma-only blocked-by": {
			set:     func() { addBlockedBy = "," },
			wantErr: `invalid --blocked-by value ",": no task IDs found`,
		},
		"whitespace and comma blocked-by": {
			set:     func() { addBlockedBy = " , " },
			wantErr: "no task IDs found",
		},
	}
	for name, tc := range tests {
		for _, dry := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "", true: "/dry-run"}[dry], func(t *testing.T) {
				resetAddFlags(t)
				resetOutputContextFlags(t)
				file := writeInputValidationFixture(t, inputValidationFixture)
				addTitle = "Second"
				dryRun = dry
				tc.set()

				var runErr error
				captureStdout(t, func() { runErr = runAdd(&cobra.Command{}, []string{file}) })
				if runErr == nil || !strings.Contains(runErr.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", runErr, tc.wantErr)
				}
				if got, _ := os.ReadFile(file); string(got) != inputValidationFixture {
					t.Errorf("file changed:\n%s", got)
				}
			})
		}
	}
}

func TestAddRequirementsFileWithRequirementsIsKept(t *testing.T) {
	resetAddFlags(t)
	resetOutputContextFlags(t)
	file := writeInputValidationFixture(t, inputValidationFixture)
	addTitle, addRequirements, addRequirementsFile = "Second", "1", "custom.md"

	captureStdout(t, func() {
		if err := runAdd(&cobra.Command{}, []string{file}); err != nil {
			t.Fatalf("runAdd: %v", err)
		}
	})
	got, _ := os.ReadFile(file)
	if !strings.Contains(string(got), "[1](custom.md#1)") {
		t.Errorf("requirements link not written with custom.md:\n%s", got)
	}
}
