package lang

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kgdb "aikits/internal/kg/db"
	goindexer "aikits/internal/kg/indexer/golang"
)

// GoIndexer implements Indexer for Go source files.
type GoIndexer struct {
	parser  *goindexer.ParserPool
	modPath string
}

// NewGoIndexer creates a GoIndexer configured for the given repo root.
func NewGoIndexer(repoRoot string) *GoIndexer {
	return &GoIndexer{
		parser:  goindexer.NewParserPool(),
		modPath: DetectGoModulePath(repoRoot),
	}
}

func (g *GoIndexer) Extract(src []byte, absPath, relPath string, repoID int64) (FileExtract, error) {
	astFile, fset, err := g.parser.ParseGoFile(absPath, src)
	if err != nil {
		return FileExtract{}, fmt.Errorf("parse %s: %w", relPath, err)
	}
	pkgPath := filepath.ToSlash(filepath.Dir(relPath))
	if pkgPath == "." {
		pkgPath = ""
	}
	syms, calls := goindexer.ExtractGoSymbols(astFile, fset, repoID, 0, g.modPath, pkgPath)
	ext := FileExtract{
		Symbols:     syms,
		Callsites:   calls,
		ImportPaths: goindexer.ExtractGoImports(astFile),
		TypeRefs:    goindexer.ExtractGoTypeRefs(astFile, g.modPath, pkgPath),
	}
	if len(syms) > 0 {
		ext.SrcPkgFQN = syms[0].FQN
	}
	return ext, nil
}

func (g *GoIndexer) StoreRefs(sqlDB *sql.DB, repoID, fileID int64, ext FileExtract) error {
	if ext.SrcPkgFQN != "" && len(ext.ImportPaths) > 0 {
		if err := kgdb.StoreImportRefs(sqlDB, repoID, fileID, ext.SrcPkgFQN, ext.ImportPaths); err != nil {
			return fmt.Errorf("store import refs: %w", err)
		}
	}
	if len(ext.TypeRefs) > 0 {
		if err := kgdb.StoreTypeRefs(sqlDB, repoID, fileID, ext.TypeRefs); err != nil {
			return fmt.Errorf("store type refs: %w", err)
		}
	}
	return nil
}

// DetectGoModulePath reads go.mod in repoRoot to extract the module path.
func DetectGoModulePath(repoRoot string) string {
	data, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return "unknown"
}
