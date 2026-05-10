package css

import (
	"sync"

	ts "github.com/tree-sitter/go-tree-sitter"
	tscss "github.com/tree-sitter/tree-sitter-css/bindings/go"

	kgdb "aikits/internal/kg/db"
)

// CSSExtractResult holds all extraction output for a single CSS source file.
type CSSExtractResult struct {
	Symbols     []kgdb.SymbolRow
	Callsites   []kgdb.CallsiteRow
	ImportPaths []string
	SrcPkgFQN   string
}

var (
	tsCSSLang     *ts.Language
	tsCSSLangOnce sync.Once
)

func getCSSLanguage() *ts.Language {
	tsCSSLangOnce.Do(func() {
		tsCSSLang = ts.NewLanguage(tscss.Language())
	})
	return tsCSSLang
}

// ExtractCSS parses src as CSS using tree-sitter and returns all indexable
// information in a single AST pass. It is safe to call concurrently; each
// call creates its own parser and tree, which are freed before returning.
func ExtractCSS(src []byte, relPath string, repoID, fileID int64) CSSExtractResult {
	lang := getCSSLanguage()
	p := ts.NewParser()
	defer p.Close()
	p.SetLanguage(lang)
	tree := p.Parse(src, nil)
	defer tree.Close()

	w := newCSSWalker(src, relPath, repoID, fileID)
	w.walkNode(tree.RootNode())

	return CSSExtractResult{
		Symbols:     w.symbols,
		Callsites:   nil,
		ImportPaths: w.imports,
		SrcPkgFQN:   w.fileModule,
	}
}
