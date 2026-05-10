package css

import (
	"path/filepath"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	kgdb "aikits/internal/kg/db"
)

// cssWalker accumulates all extraction results in a single recursive AST pass.
type cssWalker struct {
	src        []byte
	repoID     int64
	fileID     int64
	fileModule string // "<reldir>/<basename_no_ext>"

	importSeen map[string]bool
	symbols    []kgdb.SymbolRow
	imports    []string
}

func newCSSWalker(src []byte, relPath string, repoID, fileID int64) *cssWalker {
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
	return &cssWalker{
		src:        src,
		repoID:     repoID,
		fileID:     fileID,
		fileModule: module,
		importSeen: make(map[string]bool),
	}
}

func (w *cssWalker) text(n *ts.Node) string {
	return n.Utf8Text(w.src)
}

func (w *cssWalker) sym(name, kind, fqn string, n *ts.Node) kgdb.SymbolRow {
	sp := n.StartPosition()
	ep := n.EndPosition()
	return kgdb.SymbolRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		Lang:       "css",
		Kind:       kind,
		Name:       name,
		FQN:        fqn,
		Signature:  kind + " " + name,
		Visibility: "public",
		StartLine:  int(sp.Row) + 1,
		StartCol:   int(sp.Column) + 1,
		EndLine:    int(ep.Row) + 1,
		StartByte:  int(n.StartByte()),
		EndByte:    int(n.EndByte()),
	}
}

func (w *cssWalker) addImport(path string) {
	if path == "" || w.importSeen[path] {
		return
	}
	w.importSeen[path] = true
	w.imports = append(w.imports, path)
}

// ─── Top-level dispatch ───────────────────────────────────────────────────────

func (w *cssWalker) walkNode(node *ts.Node) {
	if node == nil || node.IsError() || node.IsMissing() {
		return
	}
	switch node.Kind() {
	case "rule_set":
		w.visitRuleSet(node)
	case "keyframes_statement":
		w.visitKeyframesStatement(node)
	case "import_statement":
		w.visitImportStatement(node)
	default:
		w.walkChildren(node)
	}
}

func (w *cssWalker) walkChildren(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkNode(node.Child(i))
	}
}

// ─── Rule sets ────────────────────────────────────────────────────────────────

// visitRuleSet walks all selectors in a rule set and also checks for custom
// properties in the rule block.
func (w *cssWalker) visitRuleSet(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "selectors":
			w.visitSelectors(child)
		case "block":
			w.visitBlock(child, node)
		}
	}
}

func (w *cssWalker) visitSelectors(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "class_selector":
			w.visitClassSelector(child)
		case "id_selector":
			w.visitIDSelector(child)
		}
	}
}

func (w *cssWalker) visitClassSelector(node *ts.Node) {
	nameNode := w.childByKind(node, "class_name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	fqn := w.fileModule + "." + name
	w.symbols = append(w.symbols, w.sym(name, "class", fqn, node))
}

func (w *cssWalker) visitIDSelector(node *ts.Node) {
	nameNode := w.childByKind(node, "id_name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	fqn := w.fileModule + "#" + name
	w.symbols = append(w.symbols, w.sym(name, "id", fqn, node))
}

// visitBlock looks for CSS custom property declarations (--var-name) inside a block.
// Only declarations whose property name starts with "--" are extracted.
func (w *cssWalker) visitBlock(block, ruleSet *ts.Node) {
	// Only extract custom props from :root or top-level (no selector specificity filter needed
	// at this stage — we index all --custom-prop declarations in any rule block).
	n := block.ChildCount()
	for i := uint(0); i < n; i++ {
		child := block.Child(i)
		if child == nil || child.Kind() != "declaration" {
			continue
		}
		propNode := child.ChildByFieldName("property")
		if propNode == nil {
			// Fallback: find the first child that looks like a property name.
			propNode = w.childByKind(child, "property_name")
		}
		if propNode == nil {
			continue
		}
		propName := w.text(propNode)
		if !strings.HasPrefix(propName, "--") {
			continue
		}
		fqn := w.fileModule + propName
		w.symbols = append(w.symbols, w.sym(propName, "variable", fqn, child))
	}
}

// ─── @keyframes ───────────────────────────────────────────────────────────────

func (w *cssWalker) visitKeyframesStatement(node *ts.Node) {
	nameNode := w.childByKind(node, "keyframes_name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	fqn := w.fileModule + "@" + name
	w.symbols = append(w.symbols, w.sym(name, "keyframes", fqn, node))
}

// ─── @import ──────────────────────────────────────────────────────────────────

func (w *cssWalker) visitImportStatement(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "string_value":
			path := strings.Trim(w.text(child), `"'`)
			w.addImport(path)
			return
		case "call_expression":
			// url("path") form
			if args := child.ChildByFieldName("arguments"); args != nil {
				m := args.ChildCount()
				for j := uint(0); j < m; j++ {
					arg := args.Child(j)
					if arg != nil && arg.Kind() == "string_value" {
						path := strings.Trim(w.text(arg), `"'`)
						w.addImport(path)
						return
					}
				}
			}
		}
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (w *cssWalker) childByKind(node *ts.Node, kind string) *ts.Node {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == kind {
			return child
		}
	}
	return nil
}
