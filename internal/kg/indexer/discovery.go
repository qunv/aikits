package indexer

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DiscoveredFile holds information about a discovered source file.
type DiscoveredFile struct {
	AbsPath string
	RelPath string // repo-relative, forward-slash
	Lang    string // "go" or "java"
}

// Walker discovers source files under repoRoot, respecting .gitignore patterns.
type Walker struct {
	repoRoot string
	langs    map[string]bool
	ignorer  *gitignorer
}

// NewWalker creates a walker for the given repo root and language filter.
// langs may be nil or empty to include all supported languages.
func NewWalker(repoRoot string, langs []string) *Walker {
	langMap := make(map[string]bool)
	if len(langs) == 0 {
		langMap["go"] = true
		langMap["java"] = true
	} else {
		for _, l := range langs {
			langMap[strings.ToLower(l)] = true
		}
	}
	ig := loadGitignore(repoRoot)
	return &Walker{repoRoot: repoRoot, langs: langMap, ignorer: ig}
}

// Walk traverses the repository and sends discovered files to the returned channel.
func (w *Walker) Walk() ([]DiscoveredFile, error) {
	var results []DiscoveredFile

	err := filepath.WalkDir(w.repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		rel, relErr := filepath.Rel(w.repoRoot, path)
		if relErr != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)

		// Skip hidden directories (except .git which we never descend anyway)
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "vendor" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if w.ignorer.ignored(relSlash + "/") {
				return filepath.SkipDir
			}
			return nil
		}

		if w.ignorer.ignored(relSlash) {
			return nil
		}

		lang := langForFile(path)
		if lang == "" || !w.langs[lang] {
			return nil
		}

		results = append(results, DiscoveredFile{
			AbsPath: path,
			RelPath: relSlash,
			Lang:    lang,
		})
		return nil
	})
	return results, err
}

func langForFile(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".java":
		return "java"
	default:
		return ""
	}
}

// gitignorer provides simple gitignore-style pattern matching.
type gitignorer struct {
	patterns []string
}

func loadGitignore(repoRoot string) *gitignorer {
	ig := &gitignorer{}
	f, err := os.Open(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		return ig
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ig.patterns = append(ig.patterns, line)
	}
	return ig
}

func (ig *gitignorer) ignored(relPath string) bool {
	for _, pattern := range ig.patterns {
		neg := strings.HasPrefix(pattern, "!")
		p := pattern
		if neg {
			p = p[1:]
		}
		matched := matchPattern(p, relPath)
		if matched {
			return !neg
		}
	}
	return false
}

// matchPattern matches a gitignore-style pattern against a repo-relative path.
func matchPattern(pattern, path string) bool {
	// If pattern contains a slash, match from root; otherwise match any component.
	if strings.Contains(strings.TrimSuffix(pattern, "/"), "/") {
		ok, _ := filepath.Match(pattern, path)
		return ok
	}
	// Match against the last component or any component for directory patterns.
	base := filepath.Base(path)
	if ok, _ := filepath.Match(pattern, base); ok {
		return true
	}
	// Also try matching the full path segments.
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if ok, _ := filepath.Match(pattern, part); ok {
			return true
		}
	}
	return false
}
