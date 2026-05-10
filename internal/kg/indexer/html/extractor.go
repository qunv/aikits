package html

import (
	"path/filepath"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	kgdb "aikits/internal/kg/db"
	jsindexer "aikits/internal/kg/indexer/javascript"
)

type htmlWalker struct {
	src        []byte
	repoID     int64
	fileID     int64
	fileModule string // "<reldir>/<basename_no_ext>"

	importSeen map[string]bool
	symbols    []kgdb.SymbolRow
	callsites  []kgdb.CallsiteRow
	imports    []string
}

func newHTMLWalker(src []byte, relPath string, repoID, fileID int64) *htmlWalker {
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
	return &htmlWalker{
		src:        src,
		repoID:     repoID,
		fileID:     fileID,
		fileModule: module,
		importSeen: make(map[string]bool),
	}
}

func (w *htmlWalker) text(n *ts.Node) string {
	return n.Utf8Text(w.src)
}

func (w *htmlWalker) addImport(path string) {
	if path == "" || w.importSeen[path] {
		return
	}
	w.importSeen[path] = true
	w.imports = append(w.imports, path)
}

// walkNode dispatches on the AST node kind.
func (w *htmlWalker) walkNode(node *ts.Node) {
	if node == nil || node.IsError() || node.IsMissing() {
		return
	}
	switch node.Kind() {
	case "script_element":
		w.visitScriptElement(node)
	case "element":
		w.visitElement(node)
	default:
		w.walkChildren(node)
	}
}

func (w *htmlWalker) walkChildren(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkNode(node.Child(i))
	}
}

// visitScriptElement handles <script> elements: extracts the src attribute as an
// import, or delegates the inline body to the JS extractor.
func (w *htmlWalker) visitScriptElement(node *ts.Node) {
	var srcVal string
	var rawText *ts.Node

	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "start_tag":
			srcVal = w.attrValue(child, "src")
		case "raw_text":
			rawText = child
		}
	}

	if srcVal != "" {
		w.addImport(srcVal)
		return
	}
	// Inline <script>: delegate to JS extractor.
	if rawText != nil {
		jsText := w.text(rawText)
		jsResult := jsindexer.ExtractJS([]byte(jsText), w.fileModule+".js", w.repoID, w.fileID)
		w.symbols = append(w.symbols, jsResult.Symbols...)
		w.callsites = append(w.callsites, jsResult.Callsites...)
		for _, imp := range jsResult.ImportPaths {
			w.addImport(imp)
		}
	}
}

// visitElement handles regular HTML elements: extracts id attributes as symbols,
// link href as imports, and custom-element start tags as call-sites.
func (w *htmlWalker) visitElement(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "start_tag", "self_closing_tag":
			w.visitStartTag(child)
		default:
			// Recurse into child elements.
			w.walkNode(child)
		}
	}
}

// visitStartTag processes attributes on a start tag and records the tag name
// if it is a custom element (contains "-").
func (w *htmlWalker) visitStartTag(node *ts.Node) {
	tagName := ""
	if tn := w.childByKind(node, "tag_name"); tn != nil {
		tagName = w.text(tn)
	}

	// Gather all attributes.
	idVal := w.attrValue(node, "id")
	hrefVal := w.attrValue(node, "href")

	if idVal != "" {
		fqn := w.fileModule + "#" + idVal
		sp := node.StartPosition()
		ep := node.EndPosition()
		w.symbols = append(w.symbols, kgdb.SymbolRow{
			RepoID:     w.repoID,
			FileID:     w.fileID,
			Lang:       "html",
			Kind:       "id",
			Name:       idVal,
			FQN:        fqn,
			Signature:  tagName + "#" + idVal,
			Visibility: "public",
			StartLine:  int(sp.Row) + 1,
			StartCol:   int(sp.Column) + 1,
			EndLine:    int(ep.Row) + 1,
			StartByte:  int(node.StartByte()),
			EndByte:    int(node.EndByte()),
		})
	}

	if hrefVal != "" {
		w.addImport(hrefVal)
	}

	// Custom elements: tag names containing "-" are Web Components.
	if strings.Contains(tagName, "-") {
		sp := node.StartPosition()
		ep := node.EndPosition()
		w.callsites = append(w.callsites, kgdb.CallsiteRow{
			RepoID:     w.repoID,
			FileID:     w.fileID,
			CalleeText: tagName,
			StartLine:  int(sp.Row) + 1,
			StartCol:   int(sp.Column) + 1,
			EndLine:    int(ep.Row) + 1,
			StartByte:  int(node.StartByte()),
			EndByte:    int(node.EndByte()),
			Confidence: 0.5,
			Provenance: "heuristic",
		})
	}
}

// attrValue returns the value of the named attribute within a start_tag node,
// or "" if the attribute is absent or has no value.
func (w *htmlWalker) attrValue(startTag *ts.Node, attrName string) string {
	n := startTag.ChildCount()
	for i := uint(0); i < n; i++ {
		attr := startTag.Child(i)
		if attr == nil || attr.Kind() != "attribute" {
			continue
		}
		nameNode := w.childByKind(attr, "attribute_name")
		if nameNode == nil || w.text(nameNode) != attrName {
			continue
		}
		// Value is either attribute_value or quoted_attribute_value.
		valNode := w.childByKind(attr, "quoted_attribute_value")
		if valNode != nil {
			// quoted_attribute_value contains an attribute_value child.
			inner := w.childByKind(valNode, "attribute_value")
			if inner != nil {
				return w.text(inner)
			}
		}
		valNode = w.childByKind(attr, "attribute_value")
		if valNode != nil {
			return w.text(valNode)
		}
	}
	return ""
}

// childByKind returns the first direct child of node with the given kind, or nil.
func (w *htmlWalker) childByKind(node *ts.Node, kind string) *ts.Node {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == kind {
			return child
		}
	}
	return nil
}
