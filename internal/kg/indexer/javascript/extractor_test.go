package javascript

import (
	"testing"
)

func extract(t *testing.T, src string) JSExtractResult {
	t.Helper()
	return ExtractJS([]byte(src), "src/utils/helpers.js", 1, 2)
}

func findSym(t *testing.T, r JSExtractResult, name, kind string) bool {
	t.Helper()
	for _, s := range r.Symbols {
		if s.Name == name && s.Kind == kind {
			return true
		}
	}
	return false
}

func findCallsite(t *testing.T, r JSExtractResult, callee string) bool {
	t.Helper()
	for _, c := range r.Callsites {
		if c.CalleeText == callee {
			return true
		}
	}
	return false
}

// ─── Symbol extraction ────────────────────────────────────────────────────────

func TestFunctionDeclaration(t *testing.T) {
	r := extract(t, `function foo(a, b) { return a + b; }`)
	if !findSym(t, r, "foo", "function") {
		t.Errorf("expected function symbol 'foo', got %+v", r.Symbols)
	}
}

func TestArrowFunctionConst(t *testing.T) {
	r := extract(t, `const bar = (x) => x * 2;`)
	if !findSym(t, r, "bar", "arrow_function") {
		t.Errorf("expected arrow_function symbol 'bar', got %+v", r.Symbols)
	}
}

func TestFunctionExpressionConst(t *testing.T) {
	r := extract(t, `const baz = function(x) { return x; };`)
	if !findSym(t, r, "baz", "function") {
		t.Errorf("expected function symbol 'baz', got %+v", r.Symbols)
	}
}

func TestClassDeclaration(t *testing.T) {
	r := extract(t, `class MyClass { }`)
	if !findSym(t, r, "MyClass", "class") {
		t.Errorf("expected class symbol 'MyClass', got %+v", r.Symbols)
	}
}

func TestMethodDefinitionInsideClass(t *testing.T) {
	r := extract(t, `class A { doThing() { } }`)
	if !findSym(t, r, "A", "class") {
		t.Errorf("expected class symbol 'A'")
	}
	if !findSym(t, r, "doThing", "method") {
		t.Errorf("expected method symbol 'doThing', got %+v", r.Symbols)
	}
	// FQN must contain both class name and method name.
	for _, s := range r.Symbols {
		if s.Kind == "method" && s.Name == "doThing" {
			if s.FQN == "" {
				t.Errorf("method FQN is empty")
			}
			// FQN should contain the file module, class name, and method name.
			want := "src/utils/helpers.A.doThing"
			if s.FQN != want {
				t.Errorf("method FQN = %q, want %q", s.FQN, want)
			}
		}
	}
}

func TestTopLevelVariable(t *testing.T) {
	r := extract(t, `const x = 42;`)
	if !findSym(t, r, "x", "variable") {
		t.Errorf("expected variable symbol 'x', got %+v", r.Symbols)
	}
}

func TestExportedFunctionDeclaration(t *testing.T) {
	r := extract(t, `export function greet(name) { return "hi " + name; }`)
	if !findSym(t, r, "greet", "function") {
		t.Errorf("expected exported function symbol 'greet', got %+v", r.Symbols)
	}
}

func TestExportedClass(t *testing.T) {
	r := extract(t, `export class Widget { render() {} }`)
	if !findSym(t, r, "Widget", "class") {
		t.Errorf("expected exported class 'Widget', got %+v", r.Symbols)
	}
	if !findSym(t, r, "render", "method") {
		t.Errorf("expected method 'render' inside Widget, got %+v", r.Symbols)
	}
}

// ─── Callsite extraction ──────────────────────────────────────────────────────

func TestCallExpression(t *testing.T) {
	r := extract(t, `function main() { foo(); }`)
	if !findCallsite(t, r, "foo") {
		t.Errorf("expected callsite 'foo', got %+v", r.Callsites)
	}
}

func TestMemberCallExpression(t *testing.T) {
	r := extract(t, `function main() { obj.method(); }`)
	if !findCallsite(t, r, "obj.method") {
		t.Errorf("expected callsite 'obj.method', got %+v", r.Callsites)
	}
}

// ─── Import extraction ────────────────────────────────────────────────────────

func TestImportStatement(t *testing.T) {
	r := extract(t, `import foo from './foo';`)
	found := false
	for _, p := range r.ImportPaths {
		if p == "./foo" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected import path './foo', got %+v", r.ImportPaths)
	}
}

func TestRequireCall(t *testing.T) {
	r := extract(t, `const x = require('./bar');`)
	found := false
	for _, p := range r.ImportPaths {
		if p == "./bar" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected require path './bar', got %+v", r.ImportPaths)
	}
}

// ─── File module ──────────────────────────────────────────────────────────────

func TestFileModuleFQN(t *testing.T) {
	r := ExtractJS([]byte(`function myFunc() {}`), "src/utils/helpers.js", 1, 2)
	for _, s := range r.Symbols {
		if s.Name == "myFunc" {
			want := "src/utils/helpers.myFunc"
			if s.FQN != want {
				t.Errorf("FQN = %q, want %q", s.FQN, want)
			}
			return
		}
	}
	t.Errorf("symbol 'myFunc' not found")
}

func TestFileModuleFQNRootFile(t *testing.T) {
	r := ExtractJS([]byte(`function myFunc() {}`), "index.js", 1, 2)
	for _, s := range r.Symbols {
		if s.Name == "myFunc" {
			want := "index.myFunc"
			if s.FQN != want {
				t.Errorf("FQN = %q, want %q", s.FQN, want)
			}
			return
		}
	}
	t.Errorf("symbol 'myFunc' not found")
}

// ─── Symbol metadata ──────────────────────────────────────────────────────────

func TestSymbolLangAndVisibility(t *testing.T) {
	r := extract(t, `function pub() {}`)
	for _, s := range r.Symbols {
		if s.Lang != "javascript" {
			t.Errorf("symbol lang = %q, want 'javascript'", s.Lang)
		}
		if s.Visibility != "public" {
			t.Errorf("symbol visibility = %q, want 'public'", s.Visibility)
		}
	}
}

func TestSymbolLineNumbers(t *testing.T) {
	src := "function foo() {}\nfunction bar() {}"
	r := ExtractJS([]byte(src), "a.js", 1, 2)
	for _, s := range r.Symbols {
		if s.StartLine <= 0 {
			t.Errorf("symbol %q has StartLine %d, want > 0", s.Name, s.StartLine)
		}
	}
}

// ─── Type refs (instanceof / new / HOC) ──────────────────────────────────────

func findTypeRef(r JSExtractResult, typeName string) bool {
for _, ref := range r.TypeRefs {
if ref.TypeName == typeName {
return true
}
}
return false
}

func TestInstanceofTypeRef(t *testing.T) {
r := extract(t, `function check(x) { return x instanceof MyClass; }`)
if !findTypeRef(r, "MyClass") {
t.Errorf("expected type ref 'MyClass' from instanceof, got %+v", r.TypeRefs)
}
}

func TestNewExpressionTypeRef(t *testing.T) {
r := extract(t, `function create() { return new MyService(); }`)
if !findTypeRef(r, "MyService") {
t.Errorf("expected type ref 'MyService' from new expression, got %+v", r.TypeRefs)
}
}

func TestInstanceofPrimitiveSkipped(t *testing.T) {
r := extract(t, `function isArr(x) { return x instanceof Array; }`)
if findTypeRef(r, "Array") {
t.Errorf("Array is a primitive, should not produce type ref")
}
}

func TestHOCArrowFunctionJS(t *testing.T) {
r := extract(t, `const MyComp = observer(() => { return null; });`)
if !findSym(t, r, "MyComp", "arrow_function") {
t.Errorf("expected HOC-wrapped arrow_function 'MyComp', got %+v", r.Symbols)
}
}

func TestInstanceofCallerFQN(t *testing.T) {
r := extract(t, `function check(x) { return x instanceof MyClass; }`)
for _, ref := range r.TypeRefs {
if ref.TypeName == "MyClass" && ref.SrcFQN != "src/utils/helpers.check" {
t.Errorf("instanceof ref SrcFQN = %q, want 'src/utils/helpers.check'", ref.SrcFQN)
}
}
}
