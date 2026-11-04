package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

// T-1808: a one-element references array passed through the batch command
// must be read back as one element, even when it contains a comma.
func TestBatchCommand_ReferenceWithCommaRoundTrip(t *testing.T) {
	t.Cleanup(resetBatchFlags)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("# T\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input := `{"operations":[{"type":"add","title":"With URL","references":["https://example.test/search?q=a,b"]}]}`
	rootCmd.SetArgs([]string{"batch", "tasks.md", "--input", input, "--format", "json"})
	captureStdout(t, func() {
		if err := rootCmd.Execute(); err != nil {
			t.Errorf("batch failed: %v", err)
		}
	})

	tl, err := task.ParseFile("tasks.md")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://example.test/search?q=a,b"}
	if got := tl.Tasks[0].References; !slices.Equal(got, want) {
		t.Errorf("references = %q, want %q", got, want)
	}
}
