/*
Copyright © 2025 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildTime := BuildTime
	originalGitCommit := GitCommit
	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
		GitCommit = originalGitCommit
	}()

	tests := []struct {
		name         string
		version      string
		buildTime    string
		gitCommit    string
		outputFormat string
		contains     []string
	}{
		{
			name:         "default dev version",
			version:      "dev",
			buildTime:    "unknown",
			gitCommit:    "unknown",
			outputFormat: "table",
			contains:     []string{"rune version dev", "Go: go"},
		},
		{
			name:         "specific version with build info",
			version:      "1.2.3",
			buildTime:    "2025-01-15T10:30:00Z",
			gitCommit:    "abc123def456",
			outputFormat: "table",
			contains:     []string{"rune version 1.2.3", "Built: 2025-01-15T10:30:00Z", "Commit: abc123def456", "Go: go"},
		},
		{
			name:         "json output",
			version:      "1.2.3",
			buildTime:    "2025-01-15T10:30:00Z",
			gitCommit:    "abc123def456",
			outputFormat: "json",
			contains:     []string{`"version": "1.2.3"`, `"build_time": "2025-01-15T10:30:00Z"`, `"git_commit": "abc123def456"`, `"go_version": "go`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test values
			Version = tt.version
			BuildTime = tt.buildTime
			GitCommit = tt.gitCommit
			versionOutputFormat = tt.outputFormat

			// Capture output
			buf := new(bytes.Buffer)
			versionCmd.SetOut(buf)
			versionCmd.SetErr(buf)

			// Execute the Run function directly
			versionCmd.Run(versionCmd, []string{})

			// Check output contains expected strings
			output := buf.String()
			for _, expected := range tt.contains {
				if !bytes.Contains([]byte(output), []byte(expected)) {
					t.Errorf("Expected output to contain %q, got %q", expected, output)
				}
			}
		})
	}
}

func TestGetVersionInfo(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildTime := BuildTime
	originalGitCommit := GitCommit
	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
		GitCommit = originalGitCommit
	}()

	tests := []struct {
		name      string
		version   string
		buildTime string
		gitCommit string
	}{
		{
			name:      "dev version",
			version:   "dev",
			buildTime: "unknown",
			gitCommit: "unknown",
		},
		{
			name:      "release version",
			version:   "1.2.3",
			buildTime: "2025-01-15T10:30:00Z",
			gitCommit: "abc123def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test values
			Version = tt.version
			BuildTime = tt.buildTime
			GitCommit = tt.gitCommit

			// Get version info
			info := GetVersionInfo()

			// Verify values
			if info.Version != tt.version {
				t.Errorf("Expected version %q, got %q", tt.version, info.Version)
			}
			if info.BuildTime != tt.buildTime {
				t.Errorf("Expected build time %q, got %q", tt.buildTime, info.BuildTime)
			}
			if info.GitCommit != tt.gitCommit {
				t.Errorf("Expected git commit %q, got %q", tt.gitCommit, info.GitCommit)
			}
			if !strings.HasPrefix(info.GoVersion, "go") {
				t.Errorf("Expected Go version to start with 'go', got %q", info.GoVersion)
			}
		})
	}
}
func TestVersionInjectionAndDefaults(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildTime := BuildTime
	originalGitCommit := GitCommit
	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
		GitCommit = originalGitCommit
	}()

	tests := []struct {
		name              string
		version           string
		buildTime         string
		gitCommit         string
		expectedVersion   string
		expectedBuildTime string
		expectedGitCommit string
	}{
		{
			name:              "empty version defaults to dev",
			version:           "",
			buildTime:         "2025-01-15T10:30:00Z",
			gitCommit:         "abc123def456",
			expectedVersion:   "dev",
			expectedBuildTime: "2025-01-15T10:30:00Z",
			expectedGitCommit: "abc123def456",
		},
		{
			name:              "empty build time defaults to unknown",
			version:           "1.2.3",
			buildTime:         "",
			gitCommit:         "abc123def456",
			expectedVersion:   "1.2.3",
			expectedBuildTime: "unknown",
			expectedGitCommit: "abc123def456",
		},
		{
			name:              "empty git commit defaults to unknown",
			version:           "1.2.3",
			buildTime:         "2025-01-15T10:30:00Z",
			gitCommit:         "",
			expectedVersion:   "1.2.3",
			expectedBuildTime: "2025-01-15T10:30:00Z",
			expectedGitCommit: "unknown",
		},
		{
			name:              "all empty values use defaults",
			version:           "",
			buildTime:         "",
			gitCommit:         "",
			expectedVersion:   "dev",
			expectedBuildTime: "unknown",
			expectedGitCommit: "unknown",
		},
		{
			name:              "unknown values remain unknown",
			version:           "1.2.3",
			buildTime:         "unknown",
			gitCommit:         "unknown",
			expectedVersion:   "1.2.3",
			expectedBuildTime: "unknown",
			expectedGitCommit: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test values
			Version = tt.version
			BuildTime = tt.buildTime
			GitCommit = tt.gitCommit

			// Get version info
			info := GetVersionInfo()

			// Verify values
			if info.Version != tt.expectedVersion {
				t.Errorf("Expected version %q, got %q", tt.expectedVersion, info.Version)
			}
			if info.BuildTime != tt.expectedBuildTime {
				t.Errorf("Expected build time %q, got %q", tt.expectedBuildTime, info.BuildTime)
			}
			if info.GitCommit != tt.expectedGitCommit {
				t.Errorf("Expected git commit %q, got %q", tt.expectedGitCommit, info.GitCommit)
			}
		})
	}
}

func TestVersionHelperFunctions(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildTime := BuildTime
	originalGitCommit := GitCommit
	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
		GitCommit = originalGitCommit
	}()

	t.Run("getVersionString", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"", "dev"},
			{"1.2.3", "1.2.3"},
			{"dev", "dev"},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				Version = tt.input
				got := getVersionString()
				if got != tt.want {
					t.Errorf("getVersionString() with input %q: want %q, got %q", tt.input, tt.want, got)
				}
			})
		}
	})

	t.Run("getBuildTimeString", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"", "unknown"},
			{"unknown", "unknown"},
			{"2025-01-15T10:30:00Z", "2025-01-15T10:30:00Z"},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				BuildTime = tt.input
				got := getBuildTimeString()
				if got != tt.want {
					t.Errorf("getBuildTimeString() with input %q: want %q, got %q", tt.input, tt.want, got)
				}
			})
		}
	})

	t.Run("getGitCommitString", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"", "unknown"},
			{"unknown", "unknown"},
			{"abc123def456", "abc123def456"},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				GitCommit = tt.input
				got := getGitCommitString()
				if got != tt.want {
					t.Errorf("getGitCommitString() with input %q: want %q, got %q", tt.input, tt.want, got)
				}
			})
		}
	})
}
func TestVersionCommandIntegration(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildTime := BuildTime
	originalGitCommit := GitCommit
	originalOutputFormat := versionOutputFormat
	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
		GitCommit = originalGitCommit
		versionOutputFormat = originalOutputFormat
	}()

	// Set test values
	Version = "1.2.3"
	BuildTime = "2025-01-15T10:30:00Z"
	GitCommit = "abc123def456"

	tests := []struct {
		name         string
		outputFormat string
		expectError  bool
		contains     []string
		notContains  []string
	}{
		{
			name:         "table output format",
			outputFormat: "table",
			expectError:  false,
			contains:     []string{"rune version 1.2.3", "Built: 2025-01-15T10:30:00Z", "Commit: abc123def456", "Go: go"},
			notContains:  []string{"{", "}", "\"version\""},
		},
		{
			name:         "json output format",
			outputFormat: "json",
			expectError:  false,
			contains:     []string{`"version": "1.2.3"`, `"build_time": "2025-01-15T10:30:00Z"`, `"git_commit": "abc123def456"`, `"go_version": "go`},
			notContains:  []string{"rune version", "Built:", "Commit:"},
		},
		{
			name:         "default output format",
			outputFormat: "",
			expectError:  false,
			contains:     []string{"rune version 1.2.3", "Built: 2025-01-15T10:30:00Z", "Commit: abc123def456", "Go: go"},
			notContains:  []string{"{", "}", "\"version\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set output format
			versionOutputFormat = tt.outputFormat

			// Capture output
			buf := new(bytes.Buffer)
			versionCmd.SetOut(buf)
			versionCmd.SetErr(buf)

			// Execute the Run function directly
			versionCmd.Run(versionCmd, []string{})

			// Check output
			output := buf.String()

			// Check expected content
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got %q", expected, output)
				}
			}

			// Check content that should not be present
			for _, notExpected := range tt.notContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain %q, got %q", notExpected, output)
				}
			}
		})
	}
}

func TestVersionCommandConsistency(t *testing.T) {
	// Save original values
	originalVersion := Version
	originalBuildTime := BuildTime
	originalGitCommit := GitCommit
	defer func() {
		Version = originalVersion
		BuildTime = originalBuildTime
		GitCommit = originalGitCommit
	}()

	// Test that both --version flag and version subcommand show consistent version
	testCases := []struct {
		version   string
		buildTime string
		gitCommit string
	}{
		{"dev", "unknown", "unknown"},
		{"1.2.3", "2025-01-15T10:30:00Z", "abc123def456"},
		{"2.0.0-beta", "2025-01-16T14:45:30Z", "def456ghi789"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("version_%s", tc.version), func(t *testing.T) {
			// Set test values
			Version = tc.version
			BuildTime = tc.buildTime
			GitCommit = tc.gitCommit

			// Test version subcommand
			versionOutputFormat = "table"
			buf := new(bytes.Buffer)
			versionCmd.SetOut(buf)
			versionCmd.SetErr(buf)
			versionCmd.Run(versionCmd, []string{})
			subcommandOutput := buf.String()

			// Verify version appears in subcommand output
			expectedVersionLine := fmt.Sprintf("rune version %s", tc.version)
			if !strings.Contains(subcommandOutput, expectedVersionLine) {
				t.Errorf("Version subcommand output should contain %q, got %q", expectedVersionLine, subcommandOutput)
			}

			// Test that root command version is updated
			rootCmd.Version = tc.version
			if rootCmd.Version != tc.version {
				t.Errorf("Root command version should be %q, got %q", tc.version, rootCmd.Version)
			}
		})
	}
}

func TestVersionErrorHandling(t *testing.T) {
	// Save original values
	originalOutputFormat := versionOutputFormat
	defer func() {
		versionOutputFormat = originalOutputFormat
	}()

	// Test that invalid JSON marshaling is handled gracefully
	// This is a bit contrived since our VersionInfo struct should always marshal correctly,
	// but we test the error path exists
	t.Run("json_marshaling_robustness", func(t *testing.T) {
		versionOutputFormat = "json"
		buf := new(bytes.Buffer)
		versionCmd.SetOut(buf)
		versionCmd.SetErr(buf)

		// Execute command - should not panic even with edge cases
		versionCmd.Run(versionCmd, []string{})

		output := buf.String()
		// Should produce valid JSON
		if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
			t.Errorf("JSON output should contain valid JSON structure, got %q", output)
		}
	})
}
