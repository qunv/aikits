package golang_test

import (
	"testing"

	"aikits/internal/kg/db"
	"aikits/internal/kg/indexer/golang"
)

func TestParseGoFile(t *testing.T) {
	src := []byte(`package main

import "fmt"

func Hello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

func main() {
	fmt.Println(Hello("world"))
}
`)
	pool := golang.NewParserPool()
	file, fset, err := pool.ParseGoFile("main.go", src)
	if err != nil {
		t.Fatalf("ParseGoFile: %v", err)
	}
	if file == nil {
		t.Fatal("expected ast.File, got nil")
	}
	if fset == nil {
		t.Fatal("expected FileSet, got nil")
	}
	if file.Name.Name != "main" {
		t.Errorf("package name: want main, got %s", file.Name.Name)
	}
}

func TestExtractGoSymbols(t *testing.T) {
	src := []byte(`package mypackage

// MyFunc does something.
func MyFunc(x int) int {
	return x + 1
}

// MyStruct is a struct.
type MyStruct struct {
	Field1 string
	Field2 int
}

type MyInterface interface {
	Method1() string
}

const MyConst = 42
var MyVar = "hello"
`)
	pool := golang.NewParserPool()
	file, fset, err := pool.ParseGoFile("mypackage.go", src)
	if err != nil {
		t.Fatalf("ParseGoFile: %v", err)
	}

	syms, calls := golang.ExtractGoSymbols(file, fset, 1, 2, "github.com/example", "internal/mypackage")

	if len(syms) < 5 {
		t.Errorf("expected at least 5 symbols, got %d", len(syms))
		for _, s := range syms {
			t.Logf("  sym: kind=%s name=%s fqn=%s", s.Kind, s.Name, s.FQN)
		}
	}

	kindMap := make(map[string]string)
	for _, s := range syms {
		kindMap[s.Name] = s.Kind
	}

	if kindMap["MyFunc"] != "function" {
		t.Errorf("MyFunc kind: want function, got %s", kindMap["MyFunc"])
	}
	if kindMap["MyStruct"] != "type" {
		t.Errorf("MyStruct kind: want type, got %s", kindMap["MyStruct"])
	}
	if kindMap["MyInterface"] != "interface" {
		t.Errorf("MyInterface kind: want interface, got %s", kindMap["MyInterface"])
	}
	if kindMap["MyConst"] != "const" {
		t.Errorf("MyConst kind: want const, got %s", kindMap["MyConst"])
	}
	if kindMap["MyVar"] != "var" {
		t.Errorf("MyVar kind: want var, got %s", kindMap["MyVar"])
	}

	_ = calls
}

func TestExtractGoMethodsAndCallsites(t *testing.T) {
	src := []byte(`package svc

import "fmt"

type Service struct{}

func (s *Service) DoWork() error {
	fmt.Println("working")
	return nil
}

func NewService() *Service {
	return &Service{}
}
`)
	pool := golang.NewParserPool()
	file, fset, err := pool.ParseGoFile("svc.go", src)
	if err != nil {
		t.Fatalf("ParseGoFile: %v", err)
	}

	syms, calls := golang.ExtractGoSymbols(file, fset, 1, 1, "github.com/example", "svc")

	var methodKinds []string
	for _, s := range syms {
		if s.Kind == "method" {
			methodKinds = append(methodKinds, s.Name)
		}
	}
	if len(methodKinds) == 0 {
		t.Error("expected at least one method symbol")
	}

	if len(calls) == 0 {
		t.Error("expected at least one callsite")
	}
}

func TestExtractGoImports(t *testing.T) {
	src := []byte(`package mypkg

import (
	"fmt"
	"os"
	net "net/http"
	_ "github.com/example/side"
)

func main() {}
`)
	pool := golang.NewParserPool()
	file, _, err := pool.ParseGoFile("mypkg.go", src)
	if err != nil {
		t.Fatalf("ParseGoFile: %v", err)
	}
	paths := golang.ExtractGoImports(file)
	want := map[string]bool{
		"fmt":                     true,
		"os":                      true,
		"net/http":                true,
		"github.com/example/side": true,
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

func TestExtractGoTypeRefs(t *testing.T) {
	src := []byte(`package mypackage

import (
	"github.com/example/models"
	myhttp "net/http"
)

type Server struct {
	client  *myhttp.Client
	handler models.Handler
}

func (s *Server) Process(req models.Request) (*models.Response, error) {
	return nil, nil
}
`)
	pool := golang.NewParserPool()
	file, _, err := pool.ParseGoFile("server.go", src)
	if err != nil {
		t.Fatalf("ParseGoFile: %v", err)
	}

	refs := golang.ExtractGoTypeRefs(file, "github.com/example/app", "internal/server")
	refMap := make(map[string]map[string]bool) // srcFQN → set of typeNames
	for _, r := range refs {
		if refMap[r.SrcFQN] == nil {
			refMap[r.SrcFQN] = make(map[string]bool)
		}
		refMap[r.SrcFQN][r.TypeName] = true
	}

	structFQN := "github.com/example/app/internal/server.Server"
	methodFQN := "github.com/example/app/internal/server.(Server).Process"

	tests := []struct {
		srcFQN   string
		typeName string
	}{
		{structFQN, "net/http.Client"},
		{structFQN, "github.com/example/models.Handler"},
		{methodFQN, "github.com/example/models.Request"},
		{methodFQN, "github.com/example/models.Response"},
	}
	for _, tt := range tests {
		if !refMap[tt.srcFQN][tt.typeName] {
			t.Errorf("expected ref %q -> %q; got refs for %q: %v", tt.srcFQN, tt.typeName, tt.srcFQN, keys(refMap[tt.srcFQN]))
		}
	}

	// Builtins and error should NOT appear as refs
	for _, r := range refs {
		if r.TypeName == "github.com/example/app/internal/server.error" ||
			r.TypeName == "github.com/example/app/internal/server.string" {
			t.Errorf("unexpected builtin ref: %q -> %q", r.SrcFQN, r.TypeName)
		}
	}
}

func keys(m map[string]bool) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func goSymNames(syms []db.SymbolRow) []string {
	var names []string
	for _, s := range syms {
		names = append(names, s.Name+"/"+s.Kind)
	}
	return names
}
