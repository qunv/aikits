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

	// callerStack tracks the FQN of the enclosing named symbol for callsite attribution.
	callerStack []string

	importSeen map[string]bool

	symbols   []kgdb.SymbolRow
	callsites []kgdb.CallsiteRow
	imports   []string
	typeRefs  []kgdb.TypeRef

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
	fqn := w.fqn(name)
	sig := w.buildFuncSig(name, node)
	w.symbols = append(w.symbols, w.sym(name, "function", fqn, sig, node))
	if body := node.ChildByFieldName("body"); body != nil {
		w.callerStack = append(w.callerStack, fqn)
		w.walkForCallsites(body)
		w.callerStack = w.callerStack[:len(w.callerStack)-1]
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
	fqn := w.fqn(name)
	sig := w.buildFuncSig(name, node)
	w.symbols = append(w.symbols, w.sym(name, "method", fqn, sig, node))
	if body := node.ChildByFieldName("body"); body != nil {
		w.callerStack = append(w.callerStack, fqn)
		w.walkForCallsites(body)
		w.callerStack = w.callerStack[:len(w.callerStack)-1]
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
	fqn := w.fqn(name)
	valNode := node.ChildByFieldName("value")
	if valNode == nil {
		w.symbols = append(w.symbols, w.sym(name, "variable", fqn, name, node))
		return
	}
	switch valNode.Kind() {
	case "function_expression":
		sig := w.buildFuncSig(name, valNode)
		w.symbols = append(w.symbols, w.sym(name, "function", fqn, sig, node))
		if body := valNode.ChildByFieldName("body"); body != nil {
			w.callerStack = append(w.callerStack, fqn)
			w.walkForCallsites(body)
			w.callerStack = w.callerStack[:len(w.callerStack)-1]
		}
	case "arrow_function":
		sig := w.buildFuncSig(name, valNode)
		w.symbols = append(w.symbols, w.sym(name, "arrow_function", fqn, sig, node))
		if body := valNode.ChildByFieldName("body"); body != nil {
			w.callerStack = append(w.callerStack, fqn)
			w.walkForCallsites(body)
			w.callerStack = w.callerStack[:len(w.callerStack)-1]
		}
	case "call_expression":
		// HOC pattern: const X = observer(() => { ... }) or connect(...)(() => { ... })
		// Register X as a named arrow_function if the first argument is an arrow_function.
		if firstFuncArg := w.firstArrowArg(valNode); firstFuncArg != nil {
			sig := w.buildFuncSig(name, firstFuncArg)
			w.symbols = append(w.symbols, w.sym(name, "arrow_function", fqn, sig, node))
			if body := firstFuncArg.ChildByFieldName("body"); body != nil {
				w.callerStack = append(w.callerStack, fqn)
				w.walkForCallsites(body)
				w.callerStack = w.callerStack[:len(w.callerStack)-1]
			}
		} else {
			w.symbols = append(w.symbols, w.sym(name, "variable", fqn, name, node))
			w.walkForCallsites(valNode)
		}
	case "class":
		w.symbols = append(w.symbols, w.sym(name, "class", fqn, "class "+name, node))
		w.classStack = append(w.classStack, name)
		if body := valNode.ChildByFieldName("body"); body != nil {
			w.walkClassBody(body)
		}
		w.classStack = w.classStack[:len(w.classStack)-1]
	default:
		w.symbols = append(w.symbols, w.sym(name, "variable", fqn, name, node))
		// Walk the value for nested calls (e.g. require("...")).
		w.walkForCallsites(valNode)
	}
}

// firstArrowArg returns the first arrow_function argument of a call_expression, if any.
func (w *jsWalker) firstArrowArg(callNode *ts.Node) *ts.Node {
	args := callNode.ChildByFieldName("arguments")
	if args == nil {
		return nil
	}
	n := args.ChildCount()
	for i := uint(0); i < n; i++ {
		child := args.Child(i)
		if child != nil && child.Kind() == "arrow_function" {
			return child
		}
	}
	return nil
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
	switch node.Kind() {
	case "call_expression":
		w.visitCallExpr(node, nil)
		return
	case "binary_expression":
		// instanceof: `x instanceof MyClass` → REFERENCES edge to MyClass
		if node.ChildCount() == 3 {
			op := node.Child(1)
			rhs := node.Child(2)
			if op != nil && op.Kind() == "instanceof" && rhs != nil && rhs.Kind() == "identifier" {
				w.collectTypeRef(w.text(rhs))
			}
		}
	case "new_expression":
		// `new MyClass(...)` → REFERENCES edge to MyClass (constructor call)
		if ctorNode := node.ChildByFieldName("constructor"); ctorNode != nil && ctorNode.Kind() == "identifier" {
			w.collectTypeRef(w.text(ctorNode))
		}
	}
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkForCallsites(node.Child(i))
	}
}

// collectTypeRef records a type reference from the current enclosing symbol.
func (w *jsWalker) collectTypeRef(typeName string) {
	if typeName == "" || isJSPrimitive(typeName) {
		return
	}
	caller := ""
	if len(w.callerStack) > 0 {
		caller = w.callerStack[len(w.callerStack)-1]
	}
	w.typeRefs = append(w.typeRefs, kgdb.TypeRef{
		SrcFQN:   caller,
		TypeName: typeName,
	})
}

// isJSPrimitive reports whether name is a JS built-in that should not produce REFERENCES edges.
func isJSPrimitive(name string) bool {
	switch name {
	case "Array", "Object", "Function", "Promise", "Map", "Set", "WeakMap", "WeakSet",
		"Date", "Error", "RegExp", "Event", "Element", "Node", "Math", "JSON",
		"Boolean", "Number", "String", "Symbol", "BigInt", "null", "undefined",
		"HTMLElement", "EventTarget", "XMLHttpRequest", "URL", "Blob", "File",
		"Window", "Document", "console":
		return true
	}
	return false
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
