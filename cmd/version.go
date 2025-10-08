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
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	unknownValue = "unknown"
)

// VersionInfo holds version information for display
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time,omitempty"`
	GitCommit string `json:"git_commit,omitempty"`
	GoVersion string `json:"go_version"`
}

// GetVersionInfo returns version information
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   getVersionString(),
		BuildTime: getBuildTimeString(),
		GitCommit: getGitCommitString(),
		GoVersion: runtime.Version(),
	}
}

// getVersionString returns the version string, handling missing information gracefully
func getVersionString() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

// getBuildTimeString returns the build time string, omitting if unknown
func getBuildTimeString() string {
	if BuildTime == "" || BuildTime == unknownValue {
		return unknownValue
	}
	return BuildTime
}

// getGitCommitString returns the git commit string, omitting if unknown
func getGitCommitString() string {
	if GitCommit == "" || GitCommit == unknownValue {
		return unknownValue
	}
	return GitCommit
}

var versionOutputFormat string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display version information for Rune.

This command shows the current version of Rune, along with build information
when available. Use this to verify which version you're running or for
troubleshooting purposes.`,
	Run: func(cmd *cobra.Command, args []string) {
		versionInfo := GetVersionInfo()

		switch versionOutputFormat {
		case formatJSON:
			jsonData, err := json.MarshalIndent(versionInfo, "", "  ")
			if err != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStderr(), "Error marshaling version info to JSON: %v\n", err)
				return
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
		default:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "rune version %s\n", versionInfo.Version)
			if versionInfo.BuildTime != unknownValue {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Built: %s\n", versionInfo.BuildTime)
			}
			if versionInfo.GitCommit != unknownValue {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", versionInfo.GitCommit)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Go: %s\n", versionInfo.GoVersion)
		}
	},
}

func init() {
	versionCmd.Flags().StringVarP(&versionOutputFormat, "output", "o", "table", "Output format (table, json)")
	rootCmd.AddCommand(versionCmd)
}
