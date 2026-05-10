package typescript

import (
	"path/filepath"
	"strings"
	"sync"

	ts "github.com/tree-sitter/go-tree-sitter"
	tsts "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	kgdb "aikits/internal/kg/db"
)

// TSExtractResult holds all extraction output for a single TypeScript source file.
type TSExtractResult struct {
	Symbols     []kgdb.SymbolRow
	Callsites   []kgdb.CallsiteRow
	ImportPaths []string
	SrcPkgFQN   string
	TypeRefs    []kgdb.TypeRef
}

var (
	tsTSLang     *ts.Language
	tsTSLangOnce sync.Once

	tsTSXLang     *ts.Language
	tsTSXLangOnce sync.Once
)

func getTypescriptLanguage() *ts.Language {
	tsTSLangOnce.Do(func() {
		tsTSLang = ts.NewLanguage(tsts.LanguageTypescript())
	})
	return tsTSLang
}

func getTSXLanguage() *ts.Language {
	tsTSXLangOnce.Do(func() {
		tsTSXLang = ts.NewLanguage(tsts.LanguageTSX())
	})
	return tsTSXLang
}

// ExtractTS parses src as TypeScript (or TSX) source using tree-sitter and returns all
// indexable information in a single AST pass. The grammar is selected based on the
// file extension of relPath (.tsx → TSX grammar, otherwise TypeScript grammar).
// It is safe to call concurrently; each call creates its own parser and tree.
func ExtractTS(src []byte, relPath string, repoID, fileID int64) TSExtractResult {
	ext := strings.ToLower(filepath.Ext(relPath))
	var lang *ts.Language
	if ext == ".tsx" {
		lang = getTSXLanguage()
	} else {
		lang = getTypescriptLanguage()
	}

	p := ts.NewParser()
	defer p.Close()
	p.SetLanguage(lang)
	tree := p.Parse(src, nil)
	defer tree.Close()

	w := newTSWalker(src, relPath, repoID, fileID)
	w.walkNode(tree.RootNode())

	return TSExtractResult{
		Symbols:     w.symbols,
		Callsites:   w.callsites,
		ImportPaths: w.imports,
		SrcPkgFQN:   w.fileModule,
		TypeRefs:    w.typeRefs,
	}
}
