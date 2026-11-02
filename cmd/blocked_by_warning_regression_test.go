package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T-1998: a malformed Blocked-by reference must produce a warning on stderr
// for read commands, in JSON and human formats alike, without breaking the
// JSON written to stdout.
func TestReadCommands_WarnOnMalformedBlockedBy(t *testing.T) {
	content := "- [x] 1. Finished prerequisite <!-- id:aaaaaaa -->\n" +
		"- [ ] 2. Must wait <!-- id:bbbbbbb -->\n" +
		"  - Blocked-by: abc123 (missing prerequisite)\n"

	tests := map[string]struct {
		format string
		run    func(file string) error
	}{
		"list json":  {format: formatJSON, run: func(f string) error { return runList(listCmd, []string{f}) }},
		"list table": {format: formatTable, run: func(f string) error { return runList(listCmd, []string{f}) }},
		"next json":  {format: formatJSON, run: func(f string) error { return runNext(nextCmd, []string{f}) }},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			// Command flags are package globals that other tests may leave set.
			oldFormat := format
			format = tc.format
			listFilter, showAll, listStreamFilter, listOwnerFilter = "", false, 0, ""
			phaseFlag, streamFlag, claimFlag, oneFlag = false, 0, "", false
			t.Cleanup(func() { format = oldFormat })

			var stdout string
			stderr := captureStderr(t, func() {
				stdout = captureStdout(t, func() {
					if err := tc.run("tasks.md"); err != nil {
						t.Errorf("command failed: %v", err)
					}
				})
			})

			if !strings.Contains(stderr, "invalid blocked-by reference") || !strings.Contains(stderr, "abc123") {
				t.Errorf("stderr = %q, want a warning naming abc123", stderr)
			}
			if tc.format == formatJSON && !json.Valid([]byte(stdout)) {
				t.Errorf("stdout is not valid JSON: %q", stdout)
			}
		})
	}
}
