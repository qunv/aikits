package javascript

import (
	"sync"

	ts "github.com/tree-sitter/go-tree-sitter"
	tsjs "github.com/tree-sitter/tree-sitter-javascript/bindings/go"

	kgdb "aikits/internal/kg/db"
)

// JSExtractResult holds all extraction output for a single JavaScript source file.
type JSExtractResult struct {
	Symbols     []kgdb.SymbolRow
	Callsites   []kgdb.CallsiteRow
	ImportPaths []string
	SrcPkgFQN   string
	TypeRefs    []kgdb.TypeRef
}

var (
	tsJSLang     *ts.Language
	tsJSLangOnce sync.Once
)

func getJSLanguage() *ts.Language {
	tsJSLangOnce.Do(func() {
		tsJSLang = ts.NewLanguage(tsjs.Language())
	})
	return tsJSLang
}

// ExtractJS parses src as JavaScript source using tree-sitter and returns all
// indexable information in a single AST pass. It is safe to call concurrently;
// each call creates its own parser and tree, which are freed before returning.
func ExtractJS(src []byte, relPath string, repoID, fileID int64) JSExtractResult {
	lang := getJSLanguage()
	p := ts.NewParser()
	defer p.Close()
	p.SetLanguage(lang)
	tree := p.Parse(src, nil)
	defer tree.Close()

	w := newJSWalker(src, relPath, repoID, fileID)
	w.walkNode(tree.RootNode())

	return JSExtractResult{
		Symbols:     w.symbols,
		Callsites:   w.callsites,
		ImportPaths: w.imports,
		SrcPkgFQN:   w.fileModule,
		TypeRefs:    w.typeRefs,
	}
}
