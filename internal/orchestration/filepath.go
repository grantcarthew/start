package orchestration

import (
	"os"
	"path/filepath"
	"strings"
)

// IsFilePath returns true if the string looks like a file path.
// Detection rule per DR-038: strings starting with ./, /, or ~ are file paths.
func IsFilePath(s string) bool {
	if s == "" {
		return false
	}
	return strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "~")
}

// ExpandFilePath expands tilde and converts to absolute path.
// Returns the expanded path and any error.
func ExpandFilePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand tilde
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	// Convert to absolute path
	return filepath.Abs(path)
}

// ReadFilePath reads the content of a file path, expanding tilde if present.
// Returns the content and any error.
func ReadFilePath(path string) (string, error) {
	expanded, err := ExpandFilePath(path)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(expanded)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
