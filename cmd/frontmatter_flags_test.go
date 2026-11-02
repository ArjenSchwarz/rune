package cmd

import (
	"os"
	"reflect"
	"testing"

	"github.com/arjenschwarz/rune/internal/task"
)

// TestFrontMatterFlagsPreserveCommas is a regression test for T-1804. The
// repeatable --reference and --meta flags on `create` and `add-frontmatter`
// were registered as StringSlice flags, which Cobra splits on commas: a single
// reference "docs/a,b.md" became two references, and "tags:alpha,beta" failed
// because "beta" was treated as a second key:value pair. Each flag occurrence
// must be stored as one literal string.
//
// The test drives rootCmd so the values go through real flag parsing, which
// is where the splitting happened.
func TestFrontMatterFlagsPreserveCommas(t *testing.T) {
	tests := map[string]struct {
		existing bool // run add-frontmatter on an existing file instead of create
		args     []string
		wantRefs []string
		wantMeta map[string]string
	}{
		"create keeps a comma in a reference": {
			args:     []string{"--reference", "docs/a,b.md", "--reference", "c.md"},
			wantRefs: []string{"docs/a,b.md", "c.md"},
		},
		"create keeps a comma in a metadata value": {
			args:     []string{"--meta", "tags:alpha,beta"},
			wantMeta: map[string]string{"tags": "alpha,beta"},
		},
		"add-frontmatter keeps a comma in a reference": {
			existing: true,
			args:     []string{"--reference", "docs/a,b.md"},
			wantRefs: []string{"docs/a,b.md"},
		},
		"add-frontmatter keeps a comma in a metadata value": {
			existing: true,
			args:     []string{"--meta", "tags:alpha,beta"},
			wantMeta: map[string]string{"tags": "alpha,beta"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			resetFrontMatterFlags(t)

			const file = "tasks.md"
			var args []string
			if tc.existing {
				if err := os.WriteFile(file, []byte("# Tasks\n\n- [ ] 1. Task\n"), 0644); err != nil {
					t.Fatalf("failed to write fixture: %v", err)
				}
				args = append([]string{"add-frontmatter", file}, tc.args...)
			} else {
				args = append([]string{"create", file, "--title", "Tasks"}, tc.args...)
			}

			rootCmd.SetArgs(args)
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("rune %v failed: %v", args, err)
			}

			tl, err := task.ParseFile(file)
			if err != nil {
				t.Fatalf("failed to parse result: %v", err)
			}
			var gotRefs []string
			var gotMeta map[string]string
			if tl.FrontMatter != nil {
				gotRefs = tl.FrontMatter.References
				gotMeta = tl.FrontMatter.Metadata
			}
			if len(gotRefs) > 0 || len(tc.wantRefs) > 0 {
				if !reflect.DeepEqual(gotRefs, tc.wantRefs) {
					t.Errorf("references = %q, want %q", gotRefs, tc.wantRefs)
				}
			}
			if len(gotMeta) > 0 || len(tc.wantMeta) > 0 {
				if !reflect.DeepEqual(gotMeta, tc.wantMeta) {
					t.Errorf("metadata = %v, want %v", gotMeta, tc.wantMeta)
				}
			}
		})
	}
}

// resetFrontMatterFlags clears the package-level flag variables used by create
// and add-frontmatter before the test and restores them afterwards, so values
// parsed by rootCmd do not leak between tests.
func resetFrontMatterFlags(t *testing.T) {
	t.Helper()

	oldTitle, oldRefs, oldMeta := createTitle, createReferences, createMetadata
	oldFMRefs, oldFMMeta := addFMReferences, addFMMetadata
	oldDryRun, oldFormat := dryRun, format
	t.Cleanup(func() {
		createTitle, createReferences, createMetadata = oldTitle, oldRefs, oldMeta
		addFMReferences, addFMMetadata = oldFMRefs, oldFMMeta
		dryRun, format = oldDryRun, oldFormat
	})

	createTitle, createReferences, createMetadata = "", nil, nil
	addFMReferences, addFMMetadata = nil, nil
}
