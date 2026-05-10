package typescript

import (
	"testing"
)

func extract(t *testing.T, src string) TSExtractResult {
	t.Helper()
	return ExtractTS([]byte(src), "src/utils/helpers.ts", 1, 2)
}

func extractTSX(t *testing.T, src string) TSExtractResult {
	t.Helper()
	return ExtractTS([]byte(src), "src/components/App.tsx", 1, 2)
}

func findSym(t *testing.T, r TSExtractResult, name, kind string) bool {
	t.Helper()
	for _, s := range r.Symbols {
		if s.Name == name && s.Kind == kind {
			return true
		}
	}
	return false
}

func findCallsite(t *testing.T, r TSExtractResult, callee string) bool {
	t.Helper()
	for _, c := range r.Callsites {
		if c.CalleeText == callee {
			return true
		}
	}
	return false
}

// ─── Symbol extraction: JS-compatible ────────────────────────────────────────

func TestFunctionDeclaration(t *testing.T) {
	r := extract(t, `function foo(a: number, b: number): number { return a + b; }`)
	if !findSym(t, r, "foo", "function") {
		t.Errorf("expected function symbol 'foo', got %+v", r.Symbols)
	}
}

func TestArrowFunctionConst(t *testing.T) {
	r := extract(t, `const bar = (x: string) => x.toUpperCase();`)
	if !findSym(t, r, "bar", "arrow_function") {
		t.Errorf("expected arrow_function symbol 'bar', got %+v", r.Symbols)
	}
}

func TestClassDeclaration(t *testing.T) {
	r := extract(t, `class MyClass { }`)
	if !findSym(t, r, "MyClass", "class") {
		t.Errorf("expected class symbol 'MyClass', got %+v", r.Symbols)
	}
}

func TestMethodInsideClass(t *testing.T) {
	r := extract(t, `class A { doThing(): void { } }`)
	if !findSym(t, r, "A", "class") {
		t.Errorf("expected class symbol 'A'")
	}
	if !findSym(t, r, "doThing", "method") {
		t.Errorf("expected method symbol 'doThing', got %+v", r.Symbols)
	}
	for _, s := range r.Symbols {
		if s.Kind == "method" && s.Name == "doThing" {
			want := "src/utils/helpers.A.doThing"
			if s.FQN != want {
				t.Errorf("method FQN = %q, want %q", s.FQN, want)
			}
		}
	}
}

func TestTopLevelVariable(t *testing.T) {
	r := extract(t, `const x: number = 42;`)
	if !findSym(t, r, "x", "variable") {
		t.Errorf("expected variable symbol 'x', got %+v", r.Symbols)
	}
}

func TestExportedFunction(t *testing.T) {
	r := extract(t, `export function greet(name: string): string { return "hi " + name; }`)
	if !findSym(t, r, "greet", "function") {
		t.Errorf("expected exported function symbol 'greet', got %+v", r.Symbols)
	}
}

func TestExportedClass(t *testing.T) {
	r := extract(t, `export class Widget { render(): void {} }`)
	if !findSym(t, r, "Widget", "class") {
		t.Errorf("expected exported class 'Widget'")
	}
	if !findSym(t, r, "render", "method") {
		t.Errorf("expected method 'render' inside Widget, got %+v", r.Symbols)
	}
}

// ─── TypeScript-specific symbols ──────────────────────────────────────────────

func TestInterfaceDeclaration(t *testing.T) {
	r := extract(t, `interface Shape { area(): number; }`)
	if !findSym(t, r, "Shape", "interface") {
		t.Errorf("expected interface symbol 'Shape', got %+v", r.Symbols)
	}
}

func TestExportedInterface(t *testing.T) {
	r := extract(t, `export interface Repo { name: string; }`)
	if !findSym(t, r, "Repo", "interface") {
		t.Errorf("expected exported interface symbol 'Repo', got %+v", r.Symbols)
	}
}

func TestTypeAliasDeclaration(t *testing.T) {
	r := extract(t, `type ID = string;`)
	if !findSym(t, r, "ID", "type_alias") {
		t.Errorf("expected type_alias symbol 'ID', got %+v", r.Symbols)
	}
}

func TestExportedTypeAlias(t *testing.T) {
	r := extract(t, `export type Status = "active" | "inactive";`)
	if !findSym(t, r, "Status", "type_alias") {
		t.Errorf("expected exported type_alias symbol 'Status', got %+v", r.Symbols)
	}
}

func TestEnumDeclaration(t *testing.T) {
	r := extract(t, `enum Direction { Up, Down, Left, Right }`)
	if !findSym(t, r, "Direction", "enum") {
		t.Errorf("expected enum symbol 'Direction', got %+v", r.Symbols)
	}
}

func TestEnumMembers(t *testing.T) {
	r := extract(t, `enum Color { Red = 0, Green = 1, Blue = 2 }`)
	if !findSym(t, r, "Color", "enum") {
		t.Errorf("expected enum symbol 'Color'")
	}
	if !findSym(t, r, "Red", "enum_member") {
		t.Errorf("expected enum_member 'Red', got %+v", r.Symbols)
	}
	if !findSym(t, r, "Green", "enum_member") {
		t.Errorf("expected enum_member 'Green'")
	}
	if !findSym(t, r, "Blue", "enum_member") {
		t.Errorf("expected enum_member 'Blue'")
	}
}

func TestExportedEnum(t *testing.T) {
	r := extract(t, `export enum Status { Active, Inactive }`)
	if !findSym(t, r, "Status", "enum") {
		t.Errorf("expected exported enum symbol 'Status', got %+v", r.Symbols)
	}
}

// ─── Callsite extraction ──────────────────────────────────────────────────────

func TestCallExpression(t *testing.T) {
	r := extract(t, `function main(): void { foo(); }`)
	if !findCallsite(t, r, "foo") {
		t.Errorf("expected callsite 'foo', got %+v", r.Callsites)
	}
}

func TestMemberCallExpression(t *testing.T) {
	r := extract(t, `function main(): void { obj.method(); }`)
	if !findCallsite(t, r, "obj.method") {
		t.Errorf("expected callsite 'obj.method', got %+v", r.Callsites)
	}
}

// ─── Import extraction ────────────────────────────────────────────────────────

func TestImportStatement(t *testing.T) {
	r := extract(t, `import { foo } from './foo';`)
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

func TestImportTypeStatement(t *testing.T) {
	r := extract(t, `import type { MyType } from './types';`)
	found := false
	for _, p := range r.ImportPaths {
		if p == "./types" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected import path './types', got %+v", r.ImportPaths)
	}
}

// ─── FQN derivation ──────────────────────────────────────────────────────────

func TestFileModuleFQN(t *testing.T) {
	r := ExtractTS([]byte(`function myFunc(): void {}`), "src/utils/helpers.ts", 1, 2)
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
	r := ExtractTS([]byte(`function myFunc(): void {}`), "index.ts", 1, 2)
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

// ─── TSX grammar ──────────────────────────────────────────────────────────────

func TestTSXFunctionComponent(t *testing.T) {
	r := extractTSX(t, `export function App(): JSX.Element { return <div/>; }`)
	if !findSym(t, r, "App", "function") {
		t.Errorf("expected function symbol 'App' in TSX file, got %+v", r.Symbols)
	}
}

func TestTSXArrowComponent(t *testing.T) {
	r := extractTSX(t, `const Button = (): JSX.Element => <button/>;`)
	if !findSym(t, r, "Button", "arrow_function") {
		t.Errorf("expected arrow_function symbol 'Button' in TSX file, got %+v", r.Symbols)
	}
}

// ─── Symbol metadata ──────────────────────────────────────────────────────────

func TestSymbolLangAndVisibility(t *testing.T) {
	r := extract(t, `function pub(): void {}`)
	for _, s := range r.Symbols {
		if s.Lang != "typescript" {
			t.Errorf("symbol lang = %q, want 'typescript'", s.Lang)
		}
		if s.Visibility != "public" {
			t.Errorf("symbol visibility = %q, want 'public'", s.Visibility)
		}
	}
}

func TestSymbolLineNumbers(t *testing.T) {
	src := "function foo(): void {}\nfunction bar(): void {}"
	r := ExtractTS([]byte(src), "a.ts", 1, 2)
	for _, s := range r.Symbols {
		if s.StartLine <= 0 {
			t.Errorf("symbol %q has StartLine %d, want > 0", s.Name, s.StartLine)
		}
	}
}

// ─── Additional coverage ──────────────────────────────────────────────────────

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

func TestArrowFunctionSingleParam(t *testing.T) {
	r := extract(t, `const double = x => x * 2;`)
	if !findSym(t, r, "double", "arrow_function") {
		t.Errorf("expected arrow_function 'double', got %+v", r.Symbols)
	}
}

func TestVariableWithFunctionExpression(t *testing.T) {
	r := extract(t, `const fn = function(x: number) { return x; };`)
	if !findSym(t, r, "fn", "function") {
		t.Errorf("expected function symbol 'fn' from function expression, got %+v", r.Symbols)
	}
}

func TestVariableWithClassExpression(t *testing.T) {
	r := extract(t, `const MyClass = class { method() {} };`)
	if !findSym(t, r, "MyClass", "class") {
		t.Errorf("expected class symbol 'MyClass' from class expression, got %+v", r.Symbols)
	}
}

func TestAmbientFunctionDeclaration(t *testing.T) {
	r := extract(t, `declare function external(x: string): void;`)
	if !findSym(t, r, "external", "function") {
		t.Errorf("expected function symbol 'external' from ambient declaration, got %+v", r.Symbols)
	}
}

func TestEnumMembersNoValue(t *testing.T) {
	r := extract(t, `enum Dir { Up, Down }`)
	if !findSym(t, r, "Up", "enum_member") {
		t.Errorf("expected enum_member 'Up', got %+v", r.Symbols)
	}
	if !findSym(t, r, "Down", "enum_member") {
		t.Errorf("expected enum_member 'Down'")
	}
}

func TestNestedCallInBody(t *testing.T) {
	r := extract(t, `function outer() { inner(helper()); }`)
	if !findCallsite(t, r, "inner") {
		t.Errorf("expected callsite 'inner'")
	}
	if !findCallsite(t, r, "helper") {
		t.Errorf("expected callsite 'helper'")
	}
}

func TestDuplicateImportDeduplication(t *testing.T) {
	r := extract(t, `import { a } from './mod'; import { b } from './mod';`)
	count := 0
	for _, p := range r.ImportPaths {
		if p == "./mod" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 deduplicated import for './mod', got %d", count)
	}
}

func TestTopLevelCallExpression(t *testing.T) {
	// Top-level call expressions hit visitExprStatement -> visitCallExpr
	r := extract(t, `init();`)
	if !findCallsite(t, r, "init") {
		t.Errorf("expected callsite 'init' from top-level expression, got %+v", r.Callsites)
	}
}

func TestAmbientClassDeclaration(t *testing.T) {
	r := extract(t, `declare class External { method(): void; }`)
	if !findSym(t, r, "External", "class") {
		t.Errorf("expected class symbol 'External' from ambient declaration, got %+v", r.Symbols)
	}
}

func TestAmbientInterface(t *testing.T) {
	r := extract(t, `declare interface Config { key: string; }`)
	if !findSym(t, r, "Config", "interface") {
		t.Errorf("expected interface 'Config' from ambient declaration, got %+v", r.Symbols)
	}
}

func TestAmbientEnum(t *testing.T) {
	r := extract(t, `declare enum Flags { A = 1, B = 2 }`)
	if !findSym(t, r, "Flags", "enum") {
		t.Errorf("expected enum 'Flags' from ambient declaration, got %+v", r.Symbols)
	}
}

func TestUppercaseTSXExtension(t *testing.T) {
	// .TSX (uppercase) must use the TSX grammar so JSX doesn't cause parse errors.
	r := ExtractTS([]byte(`function App() { return <div/>; }`), "src/App.TSX", 1, 2)
	if !findSym(t, r, "App", "function") {
		t.Errorf("expected function 'App' from uppercase .TSX file, got %+v", r.Symbols)
	}
}

// ─── HOC-wrapped components ───────────────────────────────────────────────────

func TestHOCWrappedArrowFunction(t *testing.T) {
	r := extractTSX(t, `
import { observer } from 'mobx-react-lite';
const MyComp = observer(() => { return <div/>; });
`)
	if !findSym(t, r, "MyComp", "arrow_function") {
		t.Errorf("expected arrow_function symbol 'MyComp' from observer HOC, got %+v", r.Symbols)
	}
}

func TestHOCCallsiteExtracted(t *testing.T) {
	r := extractTSX(t, `
import { observer } from 'mobx-react-lite';
const MyComp = observer(() => {
  doSomething();
  return <div/>;
});
`)
	if !findCallsite(t, r, "doSomething") {
		t.Errorf("expected callsite 'doSomething' inside HOC body, got %+v", r.Callsites)
	}
}

func TestHOCObserverCallsiteRecorded(t *testing.T) {
	r := extractTSX(t, `const X = observer(() => <div/>);`)
	if !findCallsite(t, r, "observer") {
		t.Errorf("expected callsite 'observer' from HOC call, got %+v", r.Callsites)
	}
}

// ─── JSX callsites ────────────────────────────────────────────────────────────

func TestJSXSelfClosingComponentCallsite(t *testing.T) {
	r := extractTSX(t, `function App() { return <MyButton label="click"/>; }`)
	if !findCallsite(t, r, "MyButton") {
		t.Errorf("expected callsite 'MyButton' from JSX self-closing element, got %+v", r.Callsites)
	}
}

func TestJSXOpeningComponentCallsite(t *testing.T) {
	r := extractTSX(t, `function App() { return <MyWrapper><span/></MyWrapper>; }`)
	if !findCallsite(t, r, "MyWrapper") {
		t.Errorf("expected callsite 'MyWrapper' from JSX opening element, got %+v", r.Callsites)
	}
}

func TestJSXHTMLTagNotCallsite(t *testing.T) {
	r := extractTSX(t, `function App() { return <div><span/></div>; }`)
	if findCallsite(t, r, "div") {
		t.Errorf("expected no callsite for HTML tag 'div', got %+v", r.Callsites)
	}
	if findCallsite(t, r, "span") {
		t.Errorf("expected no callsite for HTML tag 'span'")
	}
}

// ─── Type references (REFERENCES edges) ──────────────────────────────────────

func findTypeRef(r TSExtractResult, srcFQNSuffix, typeName string) bool {
	for _, ref := range r.TypeRefs {
		if ref.TypeName == typeName &&
			(ref.SrcFQN == srcFQNSuffix || len(ref.SrcFQN) >= len(srcFQNSuffix) &&
				ref.SrcFQN[len(ref.SrcFQN)-len(srcFQNSuffix):] == srcFQNSuffix) {
			return true
		}
	}
	return false
}

func TestAsExpressionTypeRef(t *testing.T) {
	// `link as KGEdge` inside handleLinkClick should produce a type ref.
	r := extract(t, `
interface KGEdge { id: number; }
const handleLinkClick = (link: any) => {
    const e = link as KGEdge;
};
`)
	if !findTypeRef(r, "handleLinkClick", "KGEdge") {
		t.Errorf("expected type ref KGEdge from handleLinkClick (as_expression), got %+v", r.TypeRefs)
	}
}

func TestTypeAnnotationRef(t *testing.T) {
	// `const e: KGEdge` variable annotation should produce a type ref.
	r := extract(t, `
interface KGEdge { id: number; }
function process(link: any) {
    const e: KGEdge = link;
}
`)
	if !findTypeRef(r, "process", "KGEdge") {
		t.Errorf("expected type ref KGEdge from process (type_annotation), got %+v", r.TypeRefs)
	}
}

func TestHOCTypeRef(t *testing.T) {
	// Type ref inside HOC-wrapped arrow function body.
	r := extractTSX(t, `
interface KGEdge { id: number; }
const handleLinkClick = useCallback((link: any) => {
    const e = link as KGEdge;
}, []);
`)
	if !findTypeRef(r, "handleLinkClick", "KGEdge") {
		t.Errorf("expected type ref KGEdge from handleLinkClick (HOC+as_expression), got %+v", r.TypeRefs)
	}
}

func TestPrimitiveTypeNotRef(t *testing.T) {
	// Built-in types like string, number, any must not be recorded as type refs.
	r := extract(t, `
function foo(x: string, y: number): boolean {
    const z = x as any;
    return true;
}
`)
	for _, ref := range r.TypeRefs {
		switch ref.TypeName {
		case "string", "number", "boolean", "any":
			t.Errorf("primitive type %q should not produce a type ref", ref.TypeName)
		}
	}
}

func TestNoTypeRefOutsideCallable(t *testing.T) {
	// Top-level type annotations (not inside a function body) should not generate refs
	// because we have no caller to attach them to.
	r := extract(t, `
interface KGEdge { id: number; }
const e: KGEdge = { id: 1 };
`)
	for _, ref := range r.TypeRefs {
		if ref.TypeName == "KGEdge" {
			t.Errorf("should not produce type ref for top-level variable annotation outside a function, got %+v", r.TypeRefs)
		}
	}
}
