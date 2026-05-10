package html

import (
	"sync"

	ts "github.com/tree-sitter/go-tree-sitter"
	tshtml "github.com/tree-sitter/tree-sitter-html/bindings/go"

	kgdb "aikits/internal/kg/db"
)

// HTMLExtractResult holds all extraction output for a single HTML source file.
type HTMLExtractResult struct {
	Symbols     []kgdb.SymbolRow
	Callsites   []kgdb.CallsiteRow
	ImportPaths []string
	SrcPkgFQN   string
}

var (
	tsHTMLLang     *ts.Language
	tsHTMLLangOnce sync.Once
)

func getHTMLLanguage() *ts.Language {
	tsHTMLLangOnce.Do(func() {
		tsHTMLLang = ts.NewLanguage(tshtml.Language())
	})
	return tsHTMLLang
}

// ExtractHTML parses src as HTML using tree-sitter and returns all indexable
// information in a single AST pass. It is safe to call concurrently; each call
// creates its own parser and tree, which are freed before returning.
func ExtractHTML(src []byte, relPath string, repoID, fileID int64) HTMLExtractResult {
	lang := getHTMLLanguage()
	p := ts.NewParser()
	defer p.Close()
	p.SetLanguage(lang)
	tree := p.Parse(src, nil)
	defer tree.Close()

	w := newHTMLWalker(src, relPath, repoID, fileID)
	w.walkNode(tree.RootNode())

	return HTMLExtractResult{
		Symbols:     w.symbols,
		Callsites:   w.callsites,
		ImportPaths: w.imports,
		SrcPkgFQN:   w.fileModule,
	}
}
