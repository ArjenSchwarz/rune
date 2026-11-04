package cmd

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
)

// The formatted previews must retain the validation and no-write guarantees
// of extended updates when the output and dry-run fixes are combined.
func TestUpdateFormattedDryRunValidatesBeforeOutput(t *testing.T) {
	for _, outputFormat := range []string{formatJSON, formatMarkdown} {
		for name, flags := range map[string]map[string]string{
			"unknown blocker": {"blocked-by": "999"},
			"negative stream": {"stream": "-1"},
		} {
			t.Run(outputFormat+"/"+name, func(t *testing.T) {
				file := writeTemp(t, "# Tasks\n\n- [ ] 1. Target\n")
				before := readFile(t, file)
				cmd := newUpdateTestCmd(t, flags)
				format, dryRun = outputFormat, true
				var runErr error
				out := captureStdout(t, func() { runErr = runUpdate(cmd, []string{file, "1"}) })
				if runErr == nil || out != "" {
					t.Fatalf("invalid dry run produced output %q, error %v", out, runErr)
				}
				if !bytes.Equal(before, readFile(t, file)) {
					t.Fatal("invalid dry run modified the file")
				}
			})
		}
	}
}

func TestUpdateJSONDryRunReportsExtendedChangesWithoutWriting(t *testing.T) {
	file := writeTemp(t, "# Tasks\n\n- [ ] 1. Blocker <!-- id:abc1234 -->\n- [ ] 2. Target <!-- id:def5678 -->\n  - Blocked-by: abc1234 (Blocker)\n")
	before := readFile(t, file)
	cmd := newUpdateTestCmd(t, map[string]string{"title": "Renamed", "blocked-by": "", "stream": "3", "owner": "agent-a"})
	format, dryRun = formatJSON, true
	var runErr error
	out := captureStdout(t, func() { runErr = runUpdate(cmd, []string{file, "2"}) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	var response UpdateResponse
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		t.Fatalf("invalid JSON: %v: %s", err, out)
	}
	if !response.Success || !response.DryRun || response.Title != "Renamed" || response.TaskID != "2" {
		t.Fatalf("incorrect preview: %+v", response)
	}
	for _, field := range []string{"title", "blocked-by", "stream", "owner"} {
		if !slices.Contains(response.FieldsUpdated, field) {
			t.Errorf("preview omitted %q: %+v", field, response)
		}
	}
	if !bytes.Equal(before, readFile(t, file)) {
		t.Fatal("JSON preview modified the file")
	}
}
