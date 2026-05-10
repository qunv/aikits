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
	Lang    string // "go", "java", "javascript", "typescript", "html", or "css"
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
		langMap["javascript"] = true
		langMap["typescript"] = true
		langMap["html"] = true
		langMap["css"] = true
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
	case ".js", ".mjs", ".cjs", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
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
	// Process all patterns in order; later patterns (including negations) override earlier ones.
	result := false
	for _, pattern := range ig.patterns {
		neg := strings.HasPrefix(pattern, "!")
		p := pattern
		if neg {
			p = p[1:]
		}
		if matchPattern(p, relPath) {
			result = !neg
		}
	}
	return result
}

// matchPattern matches a gitignore-style pattern against a repo-relative path.
// path is repo-relative with forward slashes; directories are passed with a trailing "/".
func matchPattern(pattern, path string) bool {
	// Determine if the pattern is anchored to the repo root (starts with "/").
	rootAnchored := strings.HasPrefix(pattern, "/")
	p := strings.TrimPrefix(pattern, "/")

	// Strip trailing slash from both pattern and path for uniform matching.
	// The caller passes dirs with a trailing "/" so a dir-only pattern like "bin/"
	// will only match directories (the trailing "/" in path distinguishes them).
	pClean := strings.TrimSuffix(p, "/")
	pathClean := strings.TrimSuffix(path, "/")

	if rootAnchored || strings.Contains(pClean, "/") {
		// Anchored pattern: match from repo root only.
		if ok, _ := filepath.Match(pClean, pathClean); ok {
			return true
		}
		// Also match paths inside an ignored directory (e.g. "build" matches "build/main.go").
		if strings.HasPrefix(pathClean, pClean+"/") {
			return true
		}
		return false
	}

	// Unanchored pattern: match the last path component or any component.
	base := filepath.Base(pathClean)
	if ok, _ := filepath.Match(pClean, base); ok {
		return true
	}
	for _, part := range strings.Split(pathClean, "/") {
		if ok, _ := filepath.Match(pClean, part); ok {
			return true
		}
	}
	return false
}
