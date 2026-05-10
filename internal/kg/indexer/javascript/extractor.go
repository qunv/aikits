package javascript

import (
	"path/filepath"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	kgdb "aikits/internal/kg/db"
)

// jsWalker accumulates all extraction results in a single recursive AST pass.
type jsWalker struct {
	src    []byte
	repoID int64
	fileID int64

	// fileModule is the FQN prefix for symbols in this file:
	// "<reldir>/<basename_no_ext>" (forward-slash separators, no leading slash).
	fileModule string

	// classStack tracks the enclosing class name during body traversal.
	classStack []string

	importSeen map[string]bool

	symbols   []kgdb.SymbolRow
	callsites []kgdb.CallsiteRow
	imports   []string

	// relPath is set once at construction; used to derive fileModule.
	relPath string
}

func newJSWalker(src []byte, relPath string, repoID, fileID int64) *jsWalker {
	dir := filepath.ToSlash(filepath.Dir(relPath))
	if dir == "." {
		dir = ""
	}
	base := filepath.Base(relPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	module := base
	if dir != "" {
		module = dir + "/" + base
	}
	return &jsWalker{
		src:        src,
		repoID:     repoID,
		fileID:     fileID,
		relPath:    relPath,
		fileModule: module,
		importSeen: make(map[string]bool),
	}
}

func (w *jsWalker) text(n *ts.Node) string {
	return n.Utf8Text(w.src)
}

func (w *jsWalker) sym(name, kind, fqn, sig string, n *ts.Node) kgdb.SymbolRow {
	sp := n.StartPosition()
	ep := n.EndPosition()
	return kgdb.SymbolRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		Lang:       "javascript",
		Kind:       kind,
		Name:       name,
		FQN:        fqn,
		Signature:  sig,
		Visibility: "public",
		StartLine:  int(sp.Row) + 1,
		StartCol:   int(sp.Column) + 1,
		EndLine:    int(ep.Row) + 1,
		StartByte:  int(n.StartByte()),
		EndByte:    int(n.EndByte()),
	}
}

func (w *jsWalker) fqn(name string) string {
	if len(w.classStack) > 0 {
		return w.fileModule + "." + w.classStack[len(w.classStack)-1] + "." + name
	}
	return w.fileModule + "." + name
}

// ─── Top-level dispatch ───────────────────────────────────────────────────────

func (w *jsWalker) walkNode(node *ts.Node) {
	if node == nil || node.IsError() || node.IsMissing() {
		return
	}
	switch node.Kind() {
	case "function_declaration":
		w.visitFunctionDecl(node)
	case "class_declaration":
		w.visitClassDecl(node)
	case "lexical_declaration", "variable_declaration":
		w.visitVarDecl(node)
	case "export_statement":
		w.visitExportStatement(node)
	case "import_statement":
		w.visitImportStatement(node)
	case "expression_statement":
		w.visitExprStatement(node)
	case "call_expression":
		w.visitCallExpr(node, nil)
	default:
		w.walkChildren(node)
	}
}

func (w *jsWalker) walkChildren(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkNode(node.Child(i))
	}
}

// ─── Declarations ─────────────────────────────────────────────────────────────

func (w *jsWalker) visitFunctionDecl(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	sig := w.buildFuncSig(name, node)
	w.symbols = append(w.symbols, w.sym(name, "function", w.fqn(name), sig, node))
	if body := node.ChildByFieldName("body"); body != nil {
		w.walkForCallsites(body)
	}
}

func (w *jsWalker) visitClassDecl(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	w.symbols = append(w.symbols, w.sym(name, "class", w.fqn(name), "class "+name, node))
	w.classStack = append(w.classStack, name)
	if body := node.ChildByFieldName("body"); body != nil {
		w.walkClassBody(body)
	}
	w.classStack = w.classStack[:len(w.classStack)-1]
}

func (w *jsWalker) walkClassBody(body *ts.Node) {
	n := body.ChildCount()
	for i := uint(0); i < n; i++ {
		child := body.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "method_definition" {
			w.visitMethodDef(child)
		}
	}
}

func (w *jsWalker) visitMethodDef(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	sig := w.buildFuncSig(name, node)
	w.symbols = append(w.symbols, w.sym(name, "method", w.fqn(name), sig, node))
	if body := node.ChildByFieldName("body"); body != nil {
		w.walkForCallsites(body)
	}
}

func (w *jsWalker) visitVarDecl(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil || child.Kind() != "variable_declarator" {
			continue
		}
		w.visitVarDeclarator(child)
	}
}

func (w *jsWalker) visitVarDeclarator(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	// Only handle simple identifier names (not destructuring patterns).
	if nameNode.Kind() != "identifier" {
		return
	}
	name := w.text(nameNode)
	valNode := node.ChildByFieldName("value")
	if valNode == nil {
		w.symbols = append(w.symbols, w.sym(name, "variable", w.fqn(name), name, node))
		return
	}
	switch valNode.Kind() {
	case "function_expression":
		sig := w.buildFuncSig(name, valNode)
		w.symbols = append(w.symbols, w.sym(name, "function", w.fqn(name), sig, node))
		if body := valNode.ChildByFieldName("body"); body != nil {
			w.walkForCallsites(body)
		}
	case "arrow_function":
		sig := w.buildFuncSig(name, valNode)
		w.symbols = append(w.symbols, w.sym(name, "arrow_function", w.fqn(name), sig, node))
		if body := valNode.ChildByFieldName("body"); body != nil {
			w.walkForCallsites(body)
		}
	case "class":
		w.symbols = append(w.symbols, w.sym(name, "class", w.fqn(name), "class "+name, node))
		w.classStack = append(w.classStack, name)
		if body := valNode.ChildByFieldName("body"); body != nil {
			w.walkClassBody(body)
		}
		w.classStack = w.classStack[:len(w.classStack)-1]
	default:
		w.symbols = append(w.symbols, w.sym(name, "variable", w.fqn(name), name, node))
		// Walk the value for nested calls (e.g. require("...")).
		w.walkForCallsites(valNode)
	}
}

func (w *jsWalker) visitExportStatement(node *ts.Node) {
	// Walk children — exported declarations are regular decl nodes inside.
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "function_declaration":
			w.visitFunctionDecl(child)
		case "class_declaration":
			w.visitClassDecl(child)
		case "lexical_declaration", "variable_declaration":
			w.visitVarDecl(child)
		}
	}
}

// ─── Imports ──────────────────────────────────────────────────────────────────

func (w *jsWalker) visitImportStatement(node *ts.Node) {
	// Find the string source node.
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "string" {
			raw := w.text(child)
			path := strings.Trim(raw, `"'` + "`")
			w.addImport(path)
			return
		}
	}
}

func (w *jsWalker) visitExprStatement(node *ts.Node) {
	if node.ChildCount() == 0 {
		return
	}
	expr := node.Child(0)
	if expr == nil {
		return
	}
	if expr.Kind() == "call_expression" {
		w.visitCallExpr(expr, nil)
	}
}

func (w *jsWalker) addImport(path string) {
	if path == "" || w.importSeen[path] {
		return
	}
	w.importSeen[path] = true
	w.imports = append(w.imports, path)
}

// ─── Callsites ────────────────────────────────────────────────────────────────

// walkForCallsites recursively finds call_expression nodes inside a body.
func (w *jsWalker) walkForCallsites(node *ts.Node) {
	if node == nil || node.IsError() || node.IsMissing() {
		return
	}
	if node.Kind() == "call_expression" {
		w.visitCallExpr(node, nil)
		return
	}
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkForCallsites(node.Child(i))
	}
}

func (w *jsWalker) visitCallExpr(node *ts.Node, _ *ts.Node) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}
	calleeText := w.text(funcNode)

	// Detect require("path") import.
	if calleeText == "require" {
		if args := node.ChildByFieldName("arguments"); args != nil {
			n := args.ChildCount()
			for i := uint(0); i < n; i++ {
				arg := args.Child(i)
				if arg != nil && arg.Kind() == "string" {
					raw := w.text(arg)
					path := strings.Trim(raw, `"'`+"`")
					w.addImport(path)
					return
				}
			}
		}
		return
	}

	sp := node.StartPosition()
	ep := node.EndPosition()
	row := kgdb.CallsiteRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		CalleeText: calleeText,
		StartLine:  int(sp.Row) + 1,
		StartCol:   int(sp.Column) + 1,
		EndLine:    int(ep.Row) + 1,
		StartByte:  int(node.StartByte()),
		EndByte:    int(node.EndByte()),
		Confidence: 0.5,
		Provenance: "heuristic",
	}
	w.callsites = append(w.callsites, row)

	// Recurse into arguments for nested calls.
	if args := node.ChildByFieldName("arguments"); args != nil {
		w.walkForCallsites(args)
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (w *jsWalker) buildFuncSig(name string, node *ts.Node) string {
	params := ""
	if pn := node.ChildByFieldName("parameters"); pn != nil {
		params = w.text(pn)
	} else if pn := node.ChildByFieldName("parameter"); pn != nil {
		// Arrow functions with a single parameter have "parameter" not "parameters".
		params = "(" + w.text(pn) + ")"
	}
	return name + params
}
