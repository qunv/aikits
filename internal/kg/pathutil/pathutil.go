package pathutil

import (
	"path/filepath"
)

// ToSlash converts a file system path to a forward-slash normalized path for DB storage.
func ToSlash(path string) string {
	return filepath.ToSlash(path)
}

// RepoRelative returns the path relative to repoRoot, normalized to forward slashes.
func RepoRelative(repoRoot, absPath string) (string, error) {
	rel, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
