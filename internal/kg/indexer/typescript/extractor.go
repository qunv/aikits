package typescript

import (
	"path/filepath"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	kgdb "aikits/internal/kg/db"
)

// tsWalker accumulates all extraction results in a single recursive AST pass.
type tsWalker struct {
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

	relPath string
}

func newTSWalker(src []byte, relPath string, repoID, fileID int64) *tsWalker {
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
	return &tsWalker{
		src:        src,
		repoID:     repoID,
		fileID:     fileID,
		relPath:    relPath,
		fileModule: module,
		importSeen: make(map[string]bool),
	}
}

func (w *tsWalker) text(n *ts.Node) string {
	return n.Utf8Text(w.src)
}

func (w *tsWalker) sym(name, kind, fqn, sig string, n *ts.Node) kgdb.SymbolRow {
	sp := n.StartPosition()
	ep := n.EndPosition()
	return kgdb.SymbolRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		Lang:       "typescript",
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

func (w *tsWalker) fqn(name string) string {
	if len(w.classStack) > 0 {
		return w.fileModule + "." + w.classStack[len(w.classStack)-1] + "." + name
	}
	return w.fileModule + "." + name
}

// ─── Top-level dispatch ───────────────────────────────────────────────────────

func (w *tsWalker) walkNode(node *ts.Node) {
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
		w.visitCallExpr(node)
	// TypeScript-specific declarations
	case "interface_declaration":
		w.visitInterfaceDecl(node)
	case "type_alias_declaration":
		w.visitTypeAliasDecl(node)
	case "enum_declaration":
		w.visitEnumDecl(node)
	case "ambient_declaration":
		w.visitAmbientDecl(node)
	default:
		w.walkChildren(node)
	}
}

func (w *tsWalker) walkChildren(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkNode(node.Child(i))
	}
}

// ─── JS-compatible declarations ───────────────────────────────────────────────

func (w *tsWalker) visitFunctionDecl(node *ts.Node) {
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

func (w *tsWalker) visitClassDecl(node *ts.Node) {
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

func (w *tsWalker) walkClassBody(body *ts.Node) {
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

func (w *tsWalker) visitMethodDef(node *ts.Node) {
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

func (w *tsWalker) visitVarDecl(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil || child.Kind() != "variable_declarator" {
			continue
		}
		w.visitVarDeclarator(child)
	}
}

func (w *tsWalker) visitVarDeclarator(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
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
		w.walkForCallsites(valNode)
	}
}

func (w *tsWalker) visitExportStatement(node *ts.Node) {
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
		case "interface_declaration":
			w.visitInterfaceDecl(child)
		case "type_alias_declaration":
			w.visitTypeAliasDecl(child)
		case "enum_declaration":
			w.visitEnumDecl(child)
		}
	}
}

// ─── TypeScript-specific declarations ────────────────────────────────────────

func (w *tsWalker) visitInterfaceDecl(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	w.symbols = append(w.symbols, w.sym(name, "interface", w.fqn(name), "interface "+name, node))
}

func (w *tsWalker) visitTypeAliasDecl(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	w.symbols = append(w.symbols, w.sym(name, "type_alias", w.fqn(name), "type "+name, node))
}

func (w *tsWalker) visitEnumDecl(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	w.symbols = append(w.symbols, w.sym(name, "enum", w.fqn(name), "enum "+name, node))

	// Walk enum body for members.
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	n := body.ChildCount()
	for i := uint(0); i < n; i++ {
		child := body.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "property_identifier", "identifier":
			memberName := w.text(child)
			w.symbols = append(w.symbols, w.sym(memberName, "enum_member",
				w.fqn(name+"."+memberName), memberName, child))
		case "enum_assignment":
			// enum_assignment: first named child is the identifier
			memberNameNode := child.ChildByFieldName("name")
			if memberNameNode == nil && child.NamedChildCount() > 0 {
				memberNameNode = child.NamedChild(0)
			}
			if memberNameNode != nil {
				memberName := w.text(memberNameNode)
				w.symbols = append(w.symbols, w.sym(memberName, "enum_member",
					w.fqn(name+"."+memberName), memberName, child))
			}
		}
	}
}

// visitAmbientDecl handles `declare ...` statements by delegating to the inner node.
func (w *tsWalker) visitAmbientDecl(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "function_declaration", "function_signature":
			w.visitFunctionDecl(child)
		case "class_declaration":
			w.visitClassDecl(child)
		case "interface_declaration":
			w.visitInterfaceDecl(child)
		case "type_alias_declaration":
			w.visitTypeAliasDecl(child)
		case "enum_declaration":
			w.visitEnumDecl(child)
		}
	}
}

// ─── Imports ──────────────────────────────────────────────────────────────────

func (w *tsWalker) visitImportStatement(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "string" {
			raw := w.text(child)
			path := strings.Trim(raw, `"'`+"`")
			w.addImport(path)
			return
		}
	}
}

func (w *tsWalker) visitExprStatement(node *ts.Node) {
	if node.ChildCount() == 0 {
		return
	}
	expr := node.Child(0)
	if expr == nil {
		return
	}
	if expr.Kind() == "call_expression" {
		w.visitCallExpr(expr)
	}
}

func (w *tsWalker) addImport(path string) {
	if path == "" || w.importSeen[path] {
		return
	}
	w.importSeen[path] = true
	w.imports = append(w.imports, path)
}

// ─── Callsites ────────────────────────────────────────────────────────────────

func (w *tsWalker) walkForCallsites(node *ts.Node) {
	if node == nil || node.IsError() || node.IsMissing() {
		return
	}
	if node.Kind() == "call_expression" {
		w.visitCallExpr(node)
		return
	}
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkForCallsites(node.Child(i))
	}
}

func (w *tsWalker) visitCallExpr(node *ts.Node) {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return
	}
	calleeText := w.text(funcNode)

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

	if args := node.ChildByFieldName("arguments"); args != nil {
		w.walkForCallsites(args)
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (w *tsWalker) buildFuncSig(name string, node *ts.Node) string {
	params := ""
	if pn := node.ChildByFieldName("parameters"); pn != nil {
		params = w.text(pn)
	} else if pn := node.ChildByFieldName("parameter"); pn != nil {
		params = "(" + w.text(pn) + ")"
	}
	return name + params
}
