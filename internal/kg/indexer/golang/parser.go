package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sync"
)

// ParserPool holds a pool of *token.FileSet for concurrent Go parsing.
type ParserPool struct {
	pool sync.Pool
}

// NewParserPool creates a new ParserPool.
func NewParserPool() *ParserPool {
	return &ParserPool{
		pool: sync.Pool{
			New: func() any { return token.NewFileSet() },
		},
	}
}

// ParseGoFile parses a Go source file and returns the AST and a fresh FileSet.
// The caller should not return the FileSet to the pool as it holds positional data.
func (p *ParserPool) ParseGoFile(path string, src []byte) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	mode := parser.ParseComments | parser.AllErrors
	f, err := parser.ParseFile(fset, path, src, mode)
	if err != nil {
		return nil, nil, err
	}
	return f, fset, nil
}
