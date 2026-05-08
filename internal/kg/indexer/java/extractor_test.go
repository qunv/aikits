package java_test

import (
	"testing"

	"aikits/internal/kg/db"
	"aikits/internal/kg/indexer/java"
)

func TestExtractJavaBasic(t *testing.T) {
	src := []byte(`package com.example;

public class Greeter {
    private String name;

    public Greeter(String name) {
        this.name = name;
    }

    public String greet() {
        return "Hello, " + name;
    }
}
`)
	syms, calls := java.ExtractJavaSymbols(src, 1, 1)

	byName := make(map[string]string) // name → kind
	byFQN := make(map[string]string)  // fqn → kind
	for _, s := range syms {
		byFQN[s.FQN] = s.Kind
		if _, exists := byName[s.Name]; !exists {
			byName[s.Name] = s.Kind
		}
	}

	if _, ok := byFQN["com.example.Greeter"]; !ok {
		t.Errorf("class Greeter FQN not found; fqns: %v", fqnList(syms))
	}
	if byFQN["com.example.Greeter"] != "class" {
		t.Errorf("com.example.Greeter: want kind=class, got %q", byFQN["com.example.Greeter"])
	}
	if byName["name"] != "field" {
		t.Errorf("name field: want kind=field, got %q", byName["name"])
	}
	foundCtor := false
	for fqn, kind := range byFQN {
		if kind == "constructor" && fqn != "" {
			foundCtor = true
		}
	}
	if !foundCtor {
		t.Errorf("expected constructor symbol; fqns: %v", fqnList(syms))
	}
	if _, ok := byFQN["com.example.Greeter#greet():String"]; !ok {
		t.Errorf("greet method FQN not found; fqns: %v", fqnList(syms))
	}
	_ = calls
}

func TestExtractJavaBraceOnNextLine(t *testing.T) {
	src := []byte(`package com.example;

public class Formatter
{
    public String format(int value)
    {
        return String.valueOf(value);
    }
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byName := symNameMap(syms)
	if byName["Formatter"] != "class" {
		t.Errorf("Formatter with brace on next line: want class, got %q (all: %v)", byName["Formatter"], symNames(syms))
	}
	if byName["format"] != "method" {
		t.Errorf("format with brace on next line: want method, got %q", byName["format"])
	}
}

func TestExtractJavaNestedClass(t *testing.T) {
	src := []byte(`package com.example;

public class Outer {
    public class Inner {
        public void run() {}
    }
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	fqns := fqnList(syms)

	found := false
	for _, fqn := range fqns {
		if fqn == "com.example.Outer.Inner" {
			found = true
		}
	}
	if !found {
		t.Errorf("nested class FQN com.example.Outer.Inner not found; got: %v", fqns)
	}
}

func TestExtractJavaBlockComments(t *testing.T) {
	src := []byte(`package com.example;

public class Documented {
    /**
     * This method has braces in the comment: { and }.
     * class FakeClass { void fakeMethod() {} }
     */
    public void real() {
        doWork();
    }
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byName := symNameMap(syms)

	if _, ok := byName["FakeClass"]; ok {
		t.Error("FakeClass from block comment should not be extracted as a symbol")
	}
	if byName["real"] != "method" {
		t.Errorf("real method: want kind=method, got %q (all: %v)", byName["real"], symNames(syms))
	}
}

func TestExtractJavaInterface(t *testing.T) {
	src := []byte(`package com.example;

public interface Processor {
    void process(String input);
    String transform(String input);
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byName := symNameMap(syms)

	if byName["Processor"] != "interface" {
		t.Errorf("Processor: want interface, got %q", byName["Processor"])
	}
	if byName["process"] != "method" {
		t.Errorf("process: want method, got %q (all: %v)", byName["process"], symNames(syms))
	}
}

func TestExtractJavaCallsites(t *testing.T) {
	src := []byte(`package com.example;

public class Caller {
    public void run() {
        doFirst();
        helper.doSecond();
    }
}
`)
	_, calls := java.ExtractJavaSymbols(src, 1, 1)
	names := make(map[string]bool)
	for _, c := range calls {
		names[c.CalleeText] = true
	}
	if !names["doFirst"] {
		t.Error("expected callsite for doFirst")
	}
	if !names["doSecond"] {
		t.Error("expected callsite for doSecond")
	}
}

func TestExtractJavaVisibility(t *testing.T) {
	src := []byte(`package com.example;

public class Vis {
    public void pub() {}
    protected void prot() {}
    private void priv() {}
    void pkg() {}
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	visByName := make(map[string]string)
	for _, s := range syms {
		visByName[s.Name] = s.Visibility
	}

	cases := map[string]string{
		"pub":  "public",
		"prot": "protected",
		"priv": "private",
		"pkg":  "package",
	}
	for name, want := range cases {
		if visByName[name] != want {
			t.Errorf("method %s: want visibility=%s, got %s", name, want, visByName[name])
		}
	}
}

func TestExtractJavaImports(t *testing.T) {
	src := []byte(`package com.example;

import com.example.other.Util;
import com.example.base.*;
import static com.example.Constants.VALUE;
import java.util.List;

public class Foo {}
`)
	paths := java.ExtractJavaImports(src)
	want := map[string]bool{
		"com.example.other": true,
		"com.example.base":  true,
		"java.util":         true,
	}
	if len(paths) != len(want) {
		t.Errorf("expected %d import paths, got %d: %v", len(want), len(paths), paths)
	}
	for _, p := range paths {
		if !want[p] {
			t.Errorf("unexpected import path: %q", p)
		}
	}
}

func TestExtractJavaPackageSymbol(t *testing.T) {
	src := []byte(`package com.example.svc;

public class MyService {
	public void serve() {}
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	if len(syms) == 0 {
		t.Fatal("expected at least one symbol")
	}
	if syms[0].Kind != "package" {
		t.Errorf("expected first symbol to be package, got %q", syms[0].Kind)
	}
	if syms[0].FQN != "com.example.svc" {
		t.Errorf("expected FQN com.example.svc, got %q", syms[0].FQN)
	}
}

func TestExtractJavaExtendsRefs(t *testing.T) {
	src := []byte(`package com.example;

public class Child extends Parent {
    public void method() {}
}

public interface SubInterface extends BaseInterface {
    void doSomething();
}

public class MultiLine
    extends AbstractBase
    implements SomeInterface {
    public void method() {}
}
`)
	refs := java.ExtractJavaExtendsRefs(src)
	want := map[string]string{
		"com.example.Child":        "Parent",
		"com.example.SubInterface": "BaseInterface",
		"com.example.MultiLine":    "AbstractBase",
	}
	if len(refs) != len(want) {
		t.Errorf("expected %d extends refs, got %d: %v", len(want), len(refs), refs)
	}
	for _, r := range refs {
		if wantSuper, ok := want[r.ClassFQN]; !ok {
			t.Errorf("unexpected extends ref: %q extends %q", r.ClassFQN, r.SuperName)
		} else if r.SuperName != wantSuper {
			t.Errorf("%q: expected extends %q, got %q", r.ClassFQN, wantSuper, r.SuperName)
		}
	}
}

func TestExtractJavaImplementsRefs(t *testing.T) {
	src := []byte(`package com.example;

// Single interface on same line
public class SingleImpl implements Runnable {
    public void run() {}
}

// Multiple interfaces, same line
public class MultiImpl implements Runnable, Serializable, Closeable {
    public void run() {}
    public void close() {}
}

// Multi-line declaration
public class MultiLine
    extends AbstractBase
    implements EventHandler, Comparable<MultiLine> {
    public void handle() {}
    public int compareTo(MultiLine o) { return 0; }
}

// Qualified name in implements clause
public class QualifiedImpl implements java.io.Serializable {
    // body
}

// Interface should NOT produce implements refs
public interface AnInterface extends BaseInterface {
    void doWork();
}
`)
	refs := java.ExtractJavaImplementsRefs(src)

	type wantRef struct {
		classFQN      string
		interfaceName string
	}
	want := []wantRef{
		{"com.example.SingleImpl", "Runnable"},
		{"com.example.MultiImpl", "Runnable"},
		{"com.example.MultiImpl", "Serializable"},
		{"com.example.MultiImpl", "Closeable"},
		{"com.example.MultiLine", "EventHandler"},
		{"com.example.MultiLine", "Comparable"}, // generics stripped
		{"com.example.QualifiedImpl", "Serializable"}, // last component of qualified name
	}

	// Build lookup map
	got := make(map[string]map[string]bool)
	for _, r := range refs {
		if got[r.ClassFQN] == nil {
			got[r.ClassFQN] = make(map[string]bool)
		}
		got[r.ClassFQN][r.InterfaceName] = true
	}

	for _, w := range want {
		if !got[w.classFQN][w.interfaceName] {
			t.Errorf("missing implements ref: %q implements %q; got for class: %v",
				w.classFQN, w.interfaceName, got[w.classFQN])
		}
	}

	// Interface should NOT appear as a class producing implements refs
	for _, r := range refs {
		if r.ClassFQN == "com.example.AnInterface" {
			t.Errorf("interface should not produce implements refs, got: %+v", r)
		}
	}
}

// TestExtractJavaImplementsRefsNoDuplicateInterfaceCollision is the core regression test:
// two interfaces share the same method name; only the explicitly declared interface
// should produce an implements ref.
func TestExtractJavaImplementsRefsNoDuplicateInterfaceCollision(t *testing.T) {
	// InterfaceA and InterfaceB both declare handle(). ClassC only implements InterfaceA.
	// The old structural heuristic would link ClassC to both — this test verifies the
	// extractor only emits the explicitly declared interface.
	src := []byte(`package com.example;

public class ClassC implements InterfaceA {
    public void handle() {}
}
`)
	refs := java.ExtractJavaImplementsRefs(src)
	if len(refs) != 1 {
		t.Fatalf("expected exactly 1 implements ref, got %d: %v", len(refs), refs)
	}
	if refs[0].ClassFQN != "com.example.ClassC" {
		t.Errorf("ClassFQN: want com.example.ClassC, got %q", refs[0].ClassFQN)
	}
	if refs[0].InterfaceName != "InterfaceA" {
		t.Errorf("InterfaceName: want InterfaceA, got %q", refs[0].InterfaceName)
	}
}

func TestExtractJavaTypeRefs(t *testing.T) {
	src := []byte(`package com.example;

import com.example.models.UserService;
import com.example.models.OrderRepository;

public class OrderController {
    private UserService userService;
    private OrderRepository repo;

    public void process(UserService svc, String id) {
        // body
    }

    public UserService getService() {
        return userService;
    }
}
`)
	refs := java.ExtractJavaTypeRefs(src)
	refMap := make(map[string]map[string]bool)
	for _, r := range refs {
		if refMap[r.SrcFQN] == nil {
			refMap[r.SrcFQN] = make(map[string]bool)
		}
		refMap[r.SrcFQN][r.TypeName] = true
	}

	classFQN := "com.example.OrderController"

	tests := []struct {
		typeName string
	}{
		{"com.example.models.UserService"},
		{"com.example.models.OrderRepository"},
	}
	for _, tt := range tests {
		if !refMap[classFQN][tt.typeName] {
			t.Errorf("expected ref %q -> %q; got: %v", classFQN, tt.typeName, refMap[classFQN])
		}
	}

	// Builtins should NOT appear
	for _, r := range refs {
		if r.TypeName == "com.example.String" {
			t.Errorf("builtin String should not appear as a type ref: %q -> %q", r.SrcFQN, r.TypeName)
		}
	}
}

// TestExtractJavaFullResult verifies the single-pass ExtractJava entry point
// returns all extraction results correctly.
func TestExtractJavaFullResult(t *testing.T) {
	src := []byte(`package com.example;

import java.util.List;
import com.example.models.UserService;

public class Service extends BaseService implements Runnable, Comparable<Service> {
    private List<String> items;
    private UserService svc;

    public Service(List<String> items, UserService svc) {
        this.items = items;
        this.svc = svc;
    }

    public void run() {
        process(items);
        svc.execute();
    }

    public UserService getService() {
        return svc;
    }
}
`)
	r := java.ExtractJava(src, 1, 1)

	if r.SrcPkgFQN != "com.example" {
		t.Errorf("SrcPkgFQN: want com.example, got %q", r.SrcPkgFQN)
	}

	byFQN := make(map[string]string)
	for _, s := range r.Symbols {
		byFQN[s.FQN] = s.Kind
	}
	if byFQN["com.example.Service"] != "class" {
		t.Errorf("class Service not found; fqns: %v", fqnList(r.Symbols))
	}

	if len(r.ExtendsRefs) == 0 {
		t.Fatal("expected at least one ExtendsRef")
	}
	foundExtends := false
	for _, ref := range r.ExtendsRefs {
		if ref.ClassFQN == "com.example.Service" && ref.SuperName == "BaseService" {
			foundExtends = true
		}
	}
	if !foundExtends {
		t.Errorf("ExtendsRef com.example.Service->BaseService not found; got: %v", r.ExtendsRefs)
	}

	if len(r.ImplRefs) == 0 {
		t.Fatal("expected at least one ImplRef")
	}
	implNames := make(map[string]bool)
	for _, ref := range r.ImplRefs {
		if ref.ClassFQN == "com.example.Service" {
			implNames[ref.InterfaceName] = true
		}
	}
	if !implNames["Runnable"] {
		t.Errorf("ImplRef Runnable not found; got: %v", implNames)
	}
	if !implNames["Comparable"] {
		t.Errorf("ImplRef Comparable not found (generics should be stripped); got: %v", implNames)
	}

	importSet := make(map[string]bool)
	for _, p := range r.ImportPaths {
		importSet[p] = true
	}
	if !importSet["java.util"] {
		t.Errorf("import java.util not found; got: %v", r.ImportPaths)
	}
	if !importSet["com.example.models"] {
		t.Errorf("import com.example.models not found; got: %v", r.ImportPaths)
	}
}

// TestExtractJavaRecord verifies that record declarations are indexed correctly.
func TestExtractJavaRecord(t *testing.T) {
	src := []byte(`package com.example;

public record Point(int x, int y) implements Comparable<Point> {
    public double distance() {
        return Math.sqrt(x * x + y * y);
    }
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byFQN := make(map[string]string)
	for _, s := range syms {
		byFQN[s.FQN] = s.Kind
	}
	if byFQN["com.example.Point"] != "record" {
		t.Errorf("Point: want kind=record, got %q (all: %v)", byFQN["com.example.Point"], fqnList(syms))
	}
	refs := java.ExtractJavaImplementsRefs(src)
	found := false
	for _, r := range refs {
		if r.ClassFQN == "com.example.Point" && r.InterfaceName == "Comparable" {
			found = true
		}
	}
	if !found {
		t.Errorf("record implements Comparable ref not found; refs: %v", refs)
	}
}

// TestExtractJavaAnnotationType verifies that annotation type declarations are indexed.
func TestExtractJavaAnnotationType(t *testing.T) {
	src := []byte(`package com.example;

public @interface MyAnnotation {
    String value() default "";
    int count() default 0;
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byFQN := make(map[string]string)
	for _, s := range syms {
		byFQN[s.FQN] = s.Kind
	}
	if byFQN["com.example.MyAnnotation"] != "annotation" {
		t.Errorf("MyAnnotation: want kind=annotation, got %q (all: %v)", byFQN["com.example.MyAnnotation"], fqnList(syms))
	}
}

// TestExtractJavaEnumWithMethods verifies enum declarations and their method members.
func TestExtractJavaEnumWithMethods(t *testing.T) {
	src := []byte(`package com.example;

public enum Status {
    ACTIVE, INACTIVE;

    public boolean isActive() {
        return this == ACTIVE;
    }
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byName := symNameMap(syms)
	if byName["Status"] != "enum" {
		t.Errorf("Status: want kind=enum, got %q", byName["Status"])
	}
	if byName["isActive"] != "method" {
		t.Errorf("isActive: want kind=method, got %q (all: %v)", byName["isActive"], symNames(syms))
	}
}

// TestExtractJavaNestedInterface verifies nested interface declarations within a class.
func TestExtractJavaNestedInterface(t *testing.T) {
	src := []byte(`package com.example;

public class Handler {
    public interface Listener {
        void onEvent(String event);
    }
    public void register(Listener l) {}
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byFQN := make(map[string]string)
	for _, s := range syms {
		byFQN[s.FQN] = s.Kind
	}
	if byFQN["com.example.Handler.Listener"] != "interface" {
		t.Errorf("Handler.Listener: want interface, got %q (all: %v)", byFQN["com.example.Handler.Listener"], fqnList(syms))
	}
}

// TestExtractJavaInterfaceExtendsMultiple verifies interface extends with multiple parents.
func TestExtractJavaInterfaceExtendsMultiple(t *testing.T) {
	src := []byte(`package com.example;

public interface Combined extends Runnable, Comparable<Combined>, AutoCloseable {
    void execute();
}
`)
	refs := java.ExtractJavaExtendsRefs(src)
	got := make(map[string]bool)
	for _, r := range refs {
		if r.ClassFQN == "com.example.Combined" {
			got[r.SuperName] = true
		}
	}
	for _, want := range []string{"Runnable", "Comparable", "AutoCloseable"} {
		if !got[want] {
			t.Errorf("interface extends %q not found; got: %v", want, got)
		}
	}
}

// TestExtractJavaGenericsInImplements verifies that generic implements clauses are
// stripped to simple names (regression for IMPLEMENTS false-positive fix).
func TestExtractJavaGenericsInImplements(t *testing.T) {
	src := []byte(`package com.example;

public class SortedList<T> implements Comparable<SortedList<T>>, java.io.Serializable {
    public int compareTo(SortedList<T> other) { return 0; }
}
`)
	refs := java.ExtractJavaImplementsRefs(src)
	got := make(map[string]bool)
	for _, r := range refs {
		got[r.InterfaceName] = true
	}
	if !got["Comparable"] {
		t.Errorf("Comparable (generics stripped) not found; got: %v", got)
	}
	if !got["Serializable"] {
		t.Errorf("Serializable (qualified name) not found; got: %v", got)
	}
}

// TestExtractJavaObjectCreation verifies that constructor calls (new Foo(...))
// are extracted as callsites — covers visitObjectCreation (was 0%).
func TestExtractJavaObjectCreation(t *testing.T) {
	src := []byte(`package com.example;

public class Factory {
    public Object build() {
        Widget w = new Widget("hello");
        Builder2 b = new Builder2();
        return w;
    }
}
`)
	_, calls := java.ExtractJavaSymbols(src, 1, 1)
	calleeSet := make(map[string]bool)
	for _, c := range calls {
		calleeSet[c.CalleeText] = true
	}
	if !calleeSet["Widget"] {
		t.Errorf("callsite Widget not found; got: %v", calleeSet)
	}
	if !calleeSet["Builder2"] {
		t.Errorf("callsite Builder2 not found; got: %v", calleeSet)
	}
}

// TestExtractJavaNoPackage verifies FQN construction when there is no package
// declaration — covers the buildFQN bare-name path.
func TestExtractJavaNoPackage(t *testing.T) {
	src := []byte(`public class Standalone {
    public void run() {}
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byFQN := make(map[string]string)
	for _, s := range syms {
		byFQN[s.FQN] = s.Kind
	}
	if byFQN["Standalone"] != "class" {
		t.Errorf("Standalone (no pkg): want kind=class, got %q (all: %v)", byFQN["Standalone"], fqnList(syms))
	}
	if byFQN["Standalone#run():void"] != "method" {
		t.Errorf("Standalone#run():void: want kind=method, got %q (all: %v)", byFQN["Standalone#run():void"], fqnList(syms))
	}
}

// TestExtractJavaIdentifierWithDigit verifies that identifiers containing digits
// are accepted — covers the isDigit branch (reached via addTypeRef with a type
// name like Http2Client that has a digit in a non-first position).
func TestExtractJavaIdentifierWithDigit(t *testing.T) {
	src := []byte(`package com.example;

public class Http2Client {
    private int maxRetries2;
    public void sendRequest2() {}
}

public class ConnectionPool {
    private Http2Client client2;
    public void connect(Http2Client conn) {}
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byName := symNameMap(syms)
	if byName["Http2Client"] != "class" {
		t.Errorf("Http2Client: want kind=class, got %q", byName["Http2Client"])
	}
	if byName["maxRetries2"] != "field" {
		t.Errorf("maxRetries2: want kind=field, got %q", byName["maxRetries2"])
	}
	if byName["sendRequest2"] != "method" {
		t.Errorf("sendRequest2: want kind=method, got %q", byName["sendRequest2"])
	}
	// client2 field has type Http2Client — exercises isDigit in isJavaIdent
	if byName["client2"] != "field" {
		t.Errorf("client2: want kind=field, got %q", byName["client2"])
	}
}

// TestExtractJavaRecordCompactConstructor verifies compact constructors in records
// — covers compact_constructor_declaration in walkBodyChildren.
func TestExtractJavaRecordCompactConstructor(t *testing.T) {
	src := []byte(`package com.example;

public record Range(int min, int max) {
    public Range {
        if (min > max) throw new IllegalArgumentException("invalid");
    }
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	// Look for the record by FQN (symNameMap would be overwritten by same-named constructor)
	byFQN := make(map[string]string)
	for _, s := range syms {
		byFQN[s.FQN] = s.Kind
	}
	if byFQN["com.example.Range"] != "record" {
		t.Errorf("com.example.Range: want kind=record, got %q (all: %v)", byFQN["com.example.Range"], fqnList(syms))
	}
	_, calls := java.ExtractJavaSymbols(src, 1, 1)
	calleeSet := make(map[string]bool)
	for _, c := range calls {
		calleeSet[c.CalleeText] = true
	}
	if !calleeSet["IllegalArgumentException"] {
		t.Errorf("constructor callsite IllegalArgumentException not found; got: %v", calleeSet)
	}
}

func TestExtractJavaGenericArrayReturnType(t *testing.T) {
	src := []byte(`package com.example;
import java.util.List;

public class Matrix {
    public String[] getRows() { return null; }
    public String[][] getMatrix() { return null; }
    public List<String>[] getGenericRows() { return null; }
}
`)
	syms, _ := java.ExtractJavaSymbols(src, 1, 1)
	byFQN := make(map[string]string)
	for _, s := range syms {
		byFQN[s.FQN] = s.Kind
	}
	if byFQN["com.example.Matrix#getRows():String[]"] != "method" {
		t.Errorf("getRows FQN: want String[] return, got keys: %v", fqnList(syms))
	}
	if byFQN["com.example.Matrix#getMatrix():String[][]"] != "method" {
		t.Errorf("getMatrix FQN: want String[][] return, got keys: %v", fqnList(syms))
	}
	if byFQN["com.example.Matrix#getGenericRows():List[]"] != "method" {
		t.Errorf("getGenericRows FQN: want List[] return (generic stripped, brackets kept), got keys: %v", fqnList(syms))
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func symNameMap(syms []db.SymbolRow) map[string]string {
	m := make(map[string]string)
	for _, s := range syms {
		m[s.Name] = s.Kind
	}
	return m
}

func symNames(syms []db.SymbolRow) []string {
	var names []string
	for _, s := range syms {
		names = append(names, s.Name+"/"+s.Kind)
	}
	return names
}

func fqnList(syms []db.SymbolRow) []string {
	var fqns []string
	for _, s := range syms {
		fqns = append(fqns, s.FQN)
	}
	return fqns
}
