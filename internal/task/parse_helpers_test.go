package task

import "os"

// writeTestFile is a helper to write content to a file for testing
func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
