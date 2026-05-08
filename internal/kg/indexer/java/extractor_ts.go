package java

import (
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	kgdb "aikits/internal/kg/db"
)

// ─── Walker state ─────────────────────────────────────────────────────────────

// classFrame tracks an enclosing type declaration during the AST walk.
type classFrame struct {
	name string
	fqn  string
	kind string
}

// walker accumulates all extraction results in a single recursive AST pass.
type walker struct {
	src    []byte
	repoID int64
	fileID int64

	pkg            string
	classStack     []classFrame
	classImportMap map[string]string // simple name → FQN (from explicit class imports)
	importSeen     map[string]bool   // dedup import paths

	symbols     []kgdb.SymbolRow
	callsites   []kgdb.CallsiteRow
	imports     []string
	extendsRefs []kgdb.ExtendsRef
	implRefs    []kgdb.ImplementsRef
	typeRefs    []kgdb.TypeRef
}

func (w *walker) text(n *ts.Node) string {
	return n.Utf8Text(w.src)
}

// ─── Top-level dispatch ───────────────────────────────────────────────────────

func (w *walker) walkNode(node *ts.Node) {
	if node == nil || node.IsError() || node.IsMissing() {
		return
	}
	switch node.Kind() {
	case "package_declaration":
		w.visitPackage(node)
	case "import_declaration":
		w.visitImport(node)
	case "class_declaration":
		w.visitTypeDecl(node, "class")
	case "interface_declaration":
		w.visitInterfaceDecl(node)
	case "enum_declaration":
		w.visitTypeDecl(node, "enum")
	case "record_declaration":
		w.visitTypeDecl(node, "record")
	case "annotation_type_declaration":
		w.visitTypeDecl(node, "annotation")
	default:
		w.walkChildren(node)
	}
}

func (w *walker) walkChildren(node *ts.Node) {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkNode(node.Child(i))
	}
}

// ─── Package & imports ────────────────────────────────────────────────────────

func (w *walker) visitPackage(node *ts.Node) {
	if node.NamedChildCount() == 0 {
		return
	}
	pkgName := w.text(node.NamedChild(0))
	w.pkg = pkgName

	startPos := node.StartPosition()
	endPos := node.EndPosition()
	w.symbols = append(w.symbols, kgdb.SymbolRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		Lang:       "java",
		Kind:       "package",
		Name:       pkgName,
		FQN:        pkgName,
		Visibility: "public",
		StartLine:  int(startPos.Row) + 1,
		StartCol:   int(startPos.Column) + 1,
		EndLine:    int(endPos.Row) + 1,
		StartByte:  int(node.StartByte()),
		EndByte:    int(node.EndByte()),
	})
}

func (w *walker) visitImport(node *ts.Node) {
	// Static imports don't affect the symbol graph.
	if strings.Contains(w.text(node), "import static ") {
		return
	}
	nc := node.NamedChildCount()
	if nc == 0 {
		return
	}
	importPath := w.text(node.NamedChild(0))
	// Wildcard: import com.example.base.*; — NamedChild(1) is the asterisk node.
	if nc >= 2 && node.NamedChild(1).Kind() == "asterisk" {
		w.addImportPath(importPath)
		return
	}
	// Specific class import: add package path and build classImportMap.
	if idx := strings.LastIndex(importPath, "."); idx >= 0 {
		simpleName := importPath[idx+1:]
		if len(simpleName) > 0 && simpleName[0] >= 'A' && simpleName[0] <= 'Z' {
			w.classImportMap[simpleName] = importPath
		}
		w.addImportPath(importPath[:idx])
	}
}

func (w *walker) addImportPath(path string) {
	if !w.importSeen[path] {
		w.importSeen[path] = true
		w.imports = append(w.imports, path)
	}
}

// ─── Type declarations ────────────────────────────────────────────────────────

// visitTypeDecl handles class, enum, record, and annotation type declarations.
func (w *walker) visitTypeDecl(node *ts.Node, kind string) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	fqn := w.buildFQN(name)
	vis := w.extractVisibility(node)
	startPos := node.StartPosition()
	endPos := node.EndPosition()

	w.symbols = append(w.symbols, kgdb.SymbolRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		Lang:       "java",
		Kind:       kind,
		Name:       name,
		FQN:        fqn,
		Visibility: vis,
		StartLine:  int(startPos.Row) + 1,
		StartCol:   int(startPos.Column) + 1,
		EndLine:    int(endPos.Row) + 1,
		StartByte:  int(node.StartByte()),
		EndByte:    int(node.EndByte()),
	})

	// Extends (superclass: only class/record; not enum, not annotation).
	if kind == "class" || kind == "record" {
		if superNode := node.ChildByFieldName("superclass"); superNode != nil {
			superText := strings.TrimPrefix(w.text(superNode), "extends ")
			if superName := javaNormalizeType(strings.TrimSpace(superText)); superName != "" {
				w.extendsRefs = append(w.extendsRefs, kgdb.ExtendsRef{
					ClassFQN:  fqn,
					SuperName: superName,
				})
			}
		}
	}

	// Implements (super_interfaces field: class, enum, record).
	if interfacesNode := node.ChildByFieldName("interfaces"); interfacesNode != nil {
		w.extractImplementsRefs(fqn, interfacesNode)
	}

	w.classStack = append(w.classStack, classFrame{name: name, fqn: fqn, kind: kind})
	if bodyNode := node.ChildByFieldName("body"); bodyNode != nil {
		w.walkBodyChildren(bodyNode)
	}
	w.classStack = w.classStack[:len(w.classStack)-1]
}

// visitInterfaceDecl handles interface declarations (uses extends_interfaces, not implements).
func (w *walker) visitInterfaceDecl(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := w.text(nameNode)
	fqn := w.buildFQN(name)
	vis := w.extractVisibility(node)
	startPos := node.StartPosition()
	endPos := node.EndPosition()

	w.symbols = append(w.symbols, kgdb.SymbolRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		Lang:       "java",
		Kind:       "interface",
		Name:       name,
		FQN:        fqn,
		Visibility: vis,
		StartLine:  int(startPos.Row) + 1,
		StartCol:   int(startPos.Column) + 1,
		EndLine:    int(endPos.Row) + 1,
		StartByte:  int(node.StartByte()),
		EndByte:    int(node.EndByte()),
	})

	// extends_interfaces is not a grammar field; find by child kind.
	for i := uint(0); i < node.ChildCount(); i++ {
		if child := node.Child(i); child != nil && child.Kind() == "extends_interfaces" {
			w.extractExtendsInterfaceRefs(fqn, child)
			break
		}
	}

	w.classStack = append(w.classStack, classFrame{name: name, fqn: fqn, kind: "interface"})
	if bodyNode := node.ChildByFieldName("body"); bodyNode != nil {
		w.walkBodyChildren(bodyNode)
	}
	w.classStack = w.classStack[:len(w.classStack)-1]
}

// walkBodyChildren dispatches member declarations within any type body
// (class_body, interface_body, enum_body, enum_body_declarations).
func (w *walker) walkBodyChildren(bodyNode *ts.Node) {
	n := bodyNode.ChildCount()
	for i := uint(0); i < n; i++ {
		child := bodyNode.Child(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "class_declaration":
			w.visitTypeDecl(child, "class")
		case "interface_declaration":
			w.visitInterfaceDecl(child)
		case "enum_declaration":
			w.visitTypeDecl(child, "enum")
		case "record_declaration":
			w.visitTypeDecl(child, "record")
		case "annotation_type_declaration":
			w.visitTypeDecl(child, "annotation")
		case "method_declaration":
			w.visitMethod(child)
		case "constructor_declaration", "compact_constructor_declaration":
			w.visitConstructor(child)
		case "field_declaration":
			w.visitField(child)
		case "enum_body_declarations":
			// The declarations block inside an enum body (after the constants).
			w.walkBodyChildren(child)
		}
	}
}

// ─── Member declarations ──────────────────────────────────────────────────────

func (w *walker) visitMethod(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	methodName := w.text(nameNode)

	returnType := ""
	if typeNode := node.ChildByFieldName("type"); typeNode != nil {
		returnType = w.text(typeNode)
	}

	paramTypes := ""
	rawParams := ""
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		paramTypes = strings.Join(w.extractParamTypes(paramsNode), ",")
		rawParams = strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(w.text(paramsNode)), ")"), "(")
		w.addParamTypeRefs(w.currentClassFQN(), paramsNode)
	}

	classFQN := w.currentClassFQN()
	fqn := classFQN + "#" + methodName + "(" + paramTypes + "):" + javaNormalizeType(returnType)

	startPos := node.StartPosition()
	endPos := node.EndPosition()
	w.symbols = append(w.symbols, kgdb.SymbolRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		Lang:       "java",
		Kind:       "method",
		Name:       methodName,
		FQN:        fqn,
		Signature:  returnType + " " + methodName + "(" + rawParams + ")",
		Visibility: w.extractVisibility(node),
		StartLine:  int(startPos.Row) + 1,
		StartCol:   int(startPos.Column) + 1,
		EndLine:    int(endPos.Row) + 1,
		StartByte:  int(node.StartByte()),
		EndByte:    int(node.EndByte()),
	})

	if bodyNode := node.ChildByFieldName("body"); bodyNode != nil {
		w.walkForCallsites(bodyNode)
	}
	if returnType != "" {
		w.addTypeRef(classFQN, returnType)
	}
}

func (w *walker) visitConstructor(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	ctorName := w.text(nameNode)

	paramTypes := ""
	rawParams := ""
	classFQN := w.currentClassFQN()
	if paramsNode := node.ChildByFieldName("parameters"); paramsNode != nil {
		paramTypes = strings.Join(w.extractParamTypes(paramsNode), ",")
		rawParams = strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(w.text(paramsNode)), ")"), "(")
		w.addParamTypeRefs(classFQN, paramsNode)
	}

	fqn := classFQN + "#" + ctorName + "(" + paramTypes + ")"
	startPos := node.StartPosition()
	endPos := node.EndPosition()
	w.symbols = append(w.symbols, kgdb.SymbolRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		Lang:       "java",
		Kind:       "constructor",
		Name:       ctorName,
		FQN:        fqn,
		Signature:  ctorName + "(" + rawParams + ")",
		Visibility: w.extractVisibility(node),
		StartLine:  int(startPos.Row) + 1,
		StartCol:   int(startPos.Column) + 1,
		EndLine:    int(endPos.Row) + 1,
		StartByte:  int(node.StartByte()),
		EndByte:    int(node.EndByte()),
	})

	if bodyNode := node.ChildByFieldName("body"); bodyNode != nil {
		w.walkForCallsites(bodyNode)
	}
}

func (w *walker) visitField(node *ts.Node) {
	classFQN := w.currentClassFQN()
	if classFQN == "" {
		return
	}
	typeName := ""
	if typeNode := node.ChildByFieldName("type"); typeNode != nil {
		typeName = w.text(typeNode)
	}
	vis := w.extractVisibility(node)
	startPos := node.StartPosition()
	endPos := node.EndPosition()

	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child == nil || child.Kind() != "variable_declarator" {
			continue
		}
		nameNode := child.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		fieldName := w.text(nameNode)
		w.symbols = append(w.symbols, kgdb.SymbolRow{
			RepoID:     w.repoID,
			FileID:     w.fileID,
			Lang:       "java",
			Kind:       "field",
			Name:       fieldName,
			FQN:        classFQN + "." + fieldName,
			Visibility: vis,
			StartLine:  int(startPos.Row) + 1,
			StartCol:   int(startPos.Column) + 1,
			EndLine:    int(endPos.Row) + 1,
			StartByte:  int(node.StartByte()),
			EndByte:    int(node.EndByte()),
		})
	}

	if typeName != "" {
		w.addTypeRef(classFQN, typeName)
	}
}

// ─── Callsite extraction ──────────────────────────────────────────────────────

// walkForCallsites recursively finds method invocations and object creations,
// but does not descend into nested type declarations.
func (w *walker) walkForCallsites(node *ts.Node) {
	if node == nil || node.IsError() || node.IsMissing() {
		return
	}
	switch node.Kind() {
	case "class_declaration", "interface_declaration", "enum_declaration",
		"record_declaration", "annotation_type_declaration":
		return
	case "method_invocation":
		w.visitCallsite(node)
	case "object_creation_expression":
		w.visitObjectCreation(node)
	}
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		w.walkForCallsites(node.Child(i))
	}
}

func (w *walker) visitCallsite(node *ts.Node) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	calleeName := w.text(nameNode)
	if javaKeywords[calleeName] || javaSkipTypes[calleeName] {
		return
	}
	startPos := nameNode.StartPosition()
	endPos := nameNode.EndPosition()
	w.callsites = append(w.callsites, kgdb.CallsiteRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		CalleeText: calleeName,
		StartLine:  int(startPos.Row) + 1,
		StartCol:   int(startPos.Column) + 1,
		EndLine:    int(endPos.Row) + 1,
		EndCol:     int(endPos.Column) + 1,
		StartByte:  int(nameNode.StartByte()),
		EndByte:    int(nameNode.EndByte()),
		Confidence: 0.5,
		Provenance: "heuristic",
	})
}

func (w *walker) visitObjectCreation(node *ts.Node) {
	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return
	}
	typeName := javaNormalizeType(w.text(typeNode))
	if typeName == "" || javaSkipTypes[typeName] {
		return
	}
	startPos := typeNode.StartPosition()
	endPos := typeNode.EndPosition()
	w.callsites = append(w.callsites, kgdb.CallsiteRow{
		RepoID:     w.repoID,
		FileID:     w.fileID,
		CalleeText: typeName,
		StartLine:  int(startPos.Row) + 1,
		StartCol:   int(startPos.Column) + 1,
		EndLine:    int(endPos.Row) + 1,
		EndCol:     int(endPos.Column) + 1,
		StartByte:  int(typeNode.StartByte()),
		EndByte:    int(typeNode.EndByte()),
		Confidence: 0.5,
		Provenance: "heuristic",
	})
}

// ─── Ref extraction ───────────────────────────────────────────────────────────

func (w *walker) extractImplementsRefs(classFQN string, node *ts.Node) {
	// super_interfaces text: "implements Foo, Bar<Baz, Qux>, java.io.Serializable"
	text := strings.TrimPrefix(w.text(node), "implements ")
	for _, part := range javaTopLevelCommaSplit(text) {
		if name := javaImplementsSimpleName(part); name != "" && isJavaIdent(name) {
			w.implRefs = append(w.implRefs, kgdb.ImplementsRef{
				ClassFQN:      classFQN,
				InterfaceName: name,
			})
		}
	}
}

func (w *walker) extractExtendsInterfaceRefs(ifaceFQN string, node *ts.Node) {
	// extends_interfaces text: "extends TypeA, TypeB<T>"
	text := strings.TrimPrefix(w.text(node), "extends ")
	for _, part := range javaTopLevelCommaSplit(text) {
		name := javaNormalizeType(strings.TrimSpace(part))
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		if name != "" && isJavaIdent(name) {
			w.extendsRefs = append(w.extendsRefs, kgdb.ExtendsRef{
				ClassFQN:  ifaceFQN,
				SuperName: name,
			})
		}
	}
}

func (w *walker) extractParamTypes(paramsNode *ts.Node) []string {
	var types []string
	n := paramsNode.ChildCount()
	for i := uint(0); i < n; i++ {
		child := paramsNode.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "formal_parameter" || child.Kind() == "spread_parameter" {
			if typeNode := child.ChildByFieldName("type"); typeNode != nil {
				types = append(types, javaNormalizeType(w.text(typeNode)))
			}
		}
	}
	return types
}

func (w *walker) addParamTypeRefs(classFQN string, paramsNode *ts.Node) {
	n := paramsNode.ChildCount()
	for i := uint(0); i < n; i++ {
		child := paramsNode.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "formal_parameter" || child.Kind() == "spread_parameter" {
			if typeNode := child.ChildByFieldName("type"); typeNode != nil {
				w.addTypeRef(classFQN, w.text(typeNode))
			}
		}
	}
}

func (w *walker) addTypeRef(srcFQN, typeName string) {
	typeName = javaNormalizeType(typeName)
	if typeName == "" || javaSkipTypeNames[typeName] || !isJavaIdent(typeName) {
		return
	}
	var resolved string
	if fqn, ok := w.classImportMap[typeName]; ok {
		resolved = fqn
	} else if w.pkg != "" {
		resolved = w.pkg + "." + typeName
	} else {
		resolved = typeName
	}
	if resolved == srcFQN {
		return
	}
	w.typeRefs = append(w.typeRefs, kgdb.TypeRef{
		SrcFQN:   srcFQN,
		TypeName: resolved,
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (w *walker) extractVisibility(node *ts.Node) string {
	n := node.ChildCount()
	for i := uint(0); i < n; i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == "modifiers" {
			text := w.text(child)
			switch {
			case strings.Contains(text, "public"):
				return "public"
			case strings.Contains(text, "protected"):
				return "protected"
			case strings.Contains(text, "private"):
				return "private"
			}
			return "package"
		}
	}
	return "package"
}

func (w *walker) buildFQN(name string) string {
	if len(w.classStack) > 0 {
		return w.classStack[len(w.classStack)-1].fqn + "." + name
	}
	if w.pkg != "" {
		return w.pkg + "." + name
	}
	return name
}

func (w *walker) currentClassFQN() string {
	if len(w.classStack) > 0 {
		return w.classStack[len(w.classStack)-1].fqn
	}
	return ""
}

// ─── Pure helper functions (shared with wrappers) ─────────────────────────────

// javaNormalizeType strips generic parameters for FQN stability,
// preserving any trailing array brackets (e.g. "List<String>[]" → "List[]").
func javaNormalizeType(t string) string {
	t = strings.TrimSpace(t)
	// Collect trailing array dimensions before stripping generics.
	arraySuffix := ""
	for strings.HasSuffix(t, "[]") {
		arraySuffix = "[]" + arraySuffix
		t = strings.TrimSpace(t[:len(t)-2])
	}
	if idx := strings.Index(t, "<"); idx >= 0 {
		t = strings.TrimSpace(t[:idx])
	}
	return t + arraySuffix
}

// javaTopLevelCommaSplit splits s on commas that are not inside angle-bracket generics.
// e.g. "Comparable<Foo>, Map.Entry<K, V>, Serializable" → ["Comparable<Foo>", "Map.Entry<K, V>", "Serializable"]
func javaTopLevelCommaSplit(s string) []string {
	var parts []string
	depth := 0
	var cur strings.Builder
	for _, ch := range s {
		switch ch {
		case '<':
			depth++
			cur.WriteRune(ch)
		case '>':
			depth--
			cur.WriteRune(ch)
		case ',':
			if depth == 0 {
				if p := strings.TrimSpace(cur.String()); p != "" {
					parts = append(parts, p)
				}
				cur.Reset()
			} else {
				cur.WriteRune(ch)
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if p := strings.TrimSpace(cur.String()); p != "" {
		parts = append(parts, p)
	}
	return parts
}

// javaImplementsSimpleName normalizes a single interface token from an implements clause:
// strips generic parameters and returns the last dot-component (simple name).
func javaImplementsSimpleName(s string) string {
	s = javaNormalizeType(s)
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.TrimSpace(s)
}

// isJavaIdent returns true when s is a syntactically valid Java identifier.
func isJavaIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, ch := range s {
		if i == 0 {
			if ch != '_' && ch != '$' && !isLetter(ch) {
				return false
			}
		} else {
			if ch != '_' && ch != '$' && !isLetter(ch) && !isDigit(ch) {
				return false
			}
		}
	}
	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// javaKeywords contains Java control-flow and reserved words that cannot be method names.
var javaKeywords = map[string]bool{
	"if": true, "else": true, "for": true, "while": true, "do": true,
	"switch": true, "case": true, "break": true, "continue": true,
	"return": true, "throw": true, "throws": true, "try": true,
	"catch": true, "finally": true, "new": true, "this": true,
	"super": true, "instanceof": true, "assert": true,
	"synchronized": true,
}

// javaSkipTypes is the set of well-known Java type names whose constructor calls
// do not need to be recorded as interesting callsites.
var javaSkipTypes = map[string]bool{
	"String": true, "Integer": true, "Long": true, "Double": true,
	"Float": true, "Boolean": true, "Byte": true, "Short": true,
	"Character": true, "Object": true, "Class": true, "Number": true,
	"StringBuilder": true, "StringBuffer": true,
}

// javaSkipTypeNames is the set of Java built-in and standard-library type names
// that should not generate REFERENCES edges.
var javaSkipTypeNames = map[string]bool{
	"void": true, "int": true, "long": true, "boolean": true, "double": true,
	"float": true, "char": true, "byte": true, "short": true,
	"String": true, "Integer": true, "Long": true, "Boolean": true,
	"Double": true, "Float": true, "Character": true, "Byte": true, "Short": true,
	"Object": true, "Class": true, "Number": true, "Enum": true,
	"StringBuilder": true, "StringBuffer": true, "CharSequence": true,
	"Throwable": true, "Exception": true, "RuntimeException": true, "Error": true,
	"Comparable": true, "Cloneable": true, "Serializable": true, "Iterable": true,
	"Override": true, "SuppressWarnings": true, "Deprecated": true,
	// Common single-letter type parameters
	"T": true, "E": true, "K": true, "V": true, "R": true, "N": true,
}
