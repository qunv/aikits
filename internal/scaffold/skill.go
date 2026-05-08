package scaffold

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

// SkillNames returns the available built-in skill names.
func SkillNames() ([]string, error) {
	entries, err := fs.ReadDir(templates, skillsTemplateRoot)
	if err != nil {
		return nil, fmt.Errorf("read embedded skills: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// AddSkill copies the named skill into destRoot/<skillName>/.
// Existing files are skipped (not overwritten).
func AddSkill(name, destRoot string) (*SkillResult, error) {
	names, err := SkillNames()
	if err != nil {
		return nil, err
	}
	if !contains(names, name) {
		return nil, fmt.Errorf("unknown skill %q — available: %v", name, names)
	}

	sub, err := fs.Sub(templates, skillsTemplateRoot+"/"+name)
	if err != nil {
		return nil, fmt.Errorf("access embedded skill %q: %w", name, err)
	}

	destDir := filepath.Join(destRoot, name)
	result := &SkillResult{SkillName: name, DestDir: destDir}

	err = fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(path))
		if fileExists(destPath) {
			result.Skipped = append(result.Skipped, path)
			return nil
		}

		result.Created = append(result.Created, path)
		return copyEmbeddedFile(sub, path, destPath)
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(result.Created)
	sort.Strings(result.Skipped)
	return result, nil
}
