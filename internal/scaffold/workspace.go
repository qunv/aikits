package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Workspace copies the non-agent scaffold templates into targetDir.
// Existing files are skipped (not overwritten).
func Workspace(target string) (*InitResult, error) {
	targetDir, err := filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("create target directory: %w", err)
	}

	result := &InitResult{TargetDir: targetDir}

	err = fs.WalkDir(templates, templateRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		// Agent-specific files are installed separately via Instructions.
		if strings.HasPrefix(path, agentTemplateRoot) {
			return nil
		}

		relPath, err := filepath.Rel(templateRoot, path)
		if err != nil {
			return fmt.Errorf("resolve template path %q: %w", path, err)
		}

		destPath := filepath.Join(targetDir, filepath.FromSlash(relPath))
		displayPath := filepath.ToSlash(relPath)
		if fileExists(destPath) {
			result.Skipped = append(result.Skipped, displayPath)
			return nil
		}

		result.Created = append(result.Created, displayPath)
		return copyEmbeddedFile(templates, path, destPath)
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(result.Created)
	sort.Strings(result.Skipped)
	return result, nil
}
