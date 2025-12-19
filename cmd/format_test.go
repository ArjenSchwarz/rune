package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout captures stdout during function execution
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy stdout: %v", err)
	}
	return buf.String()
}

// captureStderr captures stderr during function execution
func captureStderr(t *testing.T, f func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy stderr: %v", err)
	}
	return buf.String()
}

func TestOutputJSON(t *testing.T) {
	tests := map[string]struct {
		input    any
		wantErr  bool
		contains []string
	}{
		"simple_struct": {
			input: struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}{Success: true, Message: "test"},
			wantErr:  false,
			contains: []string{`"success": true`, `"message": "test"`},
		},
		"map": {
			input:    map[string]any{"key": "value", "count": 42},
			wantErr:  false,
			contains: []string{`"key": "value"`, `"count": 42`},
		},
		"empty_slice": {
			input:    []any{},
			wantErr:  false,
			contains: []string{"[]"},
		},
		"nil_value": {
			input:    nil,
			wantErr:  false,
			contains: []string{"null"},
		},
		"nested_struct": {
			input: struct {
				Success bool   `json:"success"`
				Data    []int  `json:"data"`
				Message string `json:"message,omitempty"`
			}{Success: true, Data: []int{1, 2, 3}},
			wantErr:  false,
			contains: []string{`"success": true`, `"data": [`, "1", "2", "3"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			output := captureStdout(t, func() {
				err := outputJSON(tc.input)
				if (err != nil) != tc.wantErr {
					t.Errorf("outputJSON() error = %v, wantErr %v", err, tc.wantErr)
				}
			})

			for _, want := range tc.contains {
				if !strings.Contains(output, want) {
					t.Errorf("outputJSON() output = %q, should contain %q", output, want)
				}
			}
		})
	}
}

func TestOutputMarkdownMessage(t *testing.T) {
	tests := map[string]struct {
		message string
		want    string
	}{
		"simple_message": {
			message: "All tasks are complete!",
			want:    "> All tasks are complete!\n",
		},
		"empty_message": {
			message: "",
			want:    "> \n",
		},
		"message_with_special_chars": {
			message: "No matching tasks found for 'test'",
			want:    "> No matching tasks found for 'test'\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			output := captureStdout(t, func() {
				outputMarkdownMessage(tc.message)
			})

			if output != tc.want {
				t.Errorf("outputMarkdownMessage(%q) = %q, want %q", tc.message, output, tc.want)
			}
		})
	}
}

func TestOutputMessage(t *testing.T) {
	tests := map[string]struct {
		message string
		want    string
	}{
		"simple_message": {
			message: "All tasks are complete!",
			want:    "All tasks are complete!\n",
		},
		"empty_message": {
			message: "",
			want:    "\n",
		},
		"message_with_special_chars": {
			message: "No matching tasks found for 'test'",
			want:    "No matching tasks found for 'test'\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			output := captureStdout(t, func() {
				outputMessage(tc.message)
			})

			if output != tc.want {
				t.Errorf("outputMessage(%q) = %q, want %q", tc.message, output, tc.want)
			}
		})
	}
}

func TestVerboseStderr(t *testing.T) {
	tests := map[string]struct {
		verboseFlag bool
		format      string
		args        []any
		want        string
	}{
		"verbose_enabled_simple": {
			verboseFlag: true,
			format:      "Using task file: %s",
			args:        []any{"tasks.md"},
			want:        "Using task file: tasks.md\n",
		},
		"verbose_disabled": {
			verboseFlag: false,
			format:      "Using task file: %s",
			args:        []any{"tasks.md"},
			want:        "",
		},
		"verbose_enabled_no_args": {
			verboseFlag: true,
			format:      "Processing complete",
			args:        []any{},
			want:        "Processing complete\n",
		},
		"verbose_enabled_multiple_args": {
			verboseFlag: true,
			format:      "Task %s in %s: %d items",
			args:        []any{"1.1", "tasks.md", 5},
			want:        "Task 1.1 in tasks.md: 5 items\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Save and restore verbose flag
			oldVerbose := verbose
			verbose = tc.verboseFlag
			defer func() { verbose = oldVerbose }()

			output := captureStderr(t, func() {
				verboseStderr(tc.format, tc.args...)
			})

			if output != tc.want {
				t.Errorf("verboseStderr() = %q, want %q", output, tc.want)
			}
		})
	}
}
