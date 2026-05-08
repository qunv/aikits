package java

import (
	"sync"

	ts "github.com/tree-sitter/go-tree-sitter"
	tsjava "github.com/tree-sitter/tree-sitter-java/bindings/go"

	kgdb "aikits/internal/kg/db"
)

// JavaExtractResult holds all extraction output for a single Java source file.
type JavaExtractResult struct {
	Symbols     []kgdb.SymbolRow
	Callsites   []kgdb.CallsiteRow
	ImportPaths []string
	SrcPkgFQN   string
	ExtendsRefs []kgdb.ExtendsRef
	ImplRefs    []kgdb.ImplementsRef
	TypeRefs    []kgdb.TypeRef
}

var (
	tsJavaLang     *ts.Language
	tsJavaLangOnce sync.Once
)

func getJavaLanguage() *ts.Language {
	tsJavaLangOnce.Do(func() {
		tsJavaLang = ts.NewLanguage(tsjava.Language())
	})
	return tsJavaLang
}

// ExtractJava parses src as Java source using tree-sitter and returns all
// indexable information in a single AST pass. It is safe to call concurrently;
// each call creates its own parser and tree, which are freed before returning.
func ExtractJava(src []byte, repoID, fileID int64) JavaExtractResult {
	lang := getJavaLanguage()
	p := ts.NewParser()
	defer p.Close()
	p.SetLanguage(lang)
	tree := p.Parse(src, nil)
	defer tree.Close()

	w := &walker{
		src:            src,
		repoID:         repoID,
		fileID:         fileID,
		classImportMap: make(map[string]string),
		importSeen:     make(map[string]bool),
	}
	w.walkNode(tree.RootNode())

	return JavaExtractResult{
		Symbols:     w.symbols,
		Callsites:   w.callsites,
		ImportPaths: w.imports,
		SrcPkgFQN:   w.pkg,
		ExtendsRefs: w.extendsRefs,
		ImplRefs:    w.implRefs,
		TypeRefs:    w.typeRefs,
	}
}
