package scaffold

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Agent describes an AI coding agent (name + destination directory).
type Agent interface {
	Name() string
	DestDir() string
}

// Instructions copies all agent-specific template files into the agent's
// destination directory under targetDir.
func Instructions(a Agent, target string) (*InitResult, error) {
	destDir := a.DestDir()

	targetDir, err := filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory: %w", err)
	}

	result := &InitResult{TargetDir: targetDir}

	err = fs.WalkDir(templates, agentTemplateRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		// Skills are installed on-demand via SkillAdd, not during init.
		if strings.HasPrefix(path, skillsTemplateRoot) {
			return nil
		}

		relPath, err := filepath.Rel(agentTemplateRoot, path)
		if err != nil {
			return fmt.Errorf("resolve agent template path %q: %w", path, err)
		}

		destPath := filepath.Join(targetDir, filepath.FromSlash(destDir), filepath.FromSlash(relPath))
		displayPath := filepath.ToSlash(filepath.Join(destDir, relPath))

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
