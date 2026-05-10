package indexer_test

import (
	"os"
	"path/filepath"
	"testing"

	"aikits/internal/kg/indexer"
)

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWalkFindsGoAndJavaFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "main.go"))
	writeTestFile(t, filepath.Join(dir, "Hello.java"))
	writeTestFile(t, filepath.Join(dir, "README.md"))

	w := indexer.NewWalker(dir, nil)
	files, err := w.Walk()
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
		for _, f := range files {
			t.Logf("  found: %s (lang=%s)", f.RelPath, f.Lang)
		}
	}

	langs := make(map[string]bool)
	for _, f := range files {
		langs[f.Lang] = true
	}
	if !langs["go"] {
		t.Error("expected go file in results")
	}
	if !langs["java"] {
		t.Error("expected java file in results")
	}
}

func TestWalkRespectsGitignore(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.gen.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "gen.gen.go"))
	writeTestFile(t, filepath.Join(dir, "real.go"))

	w := indexer.NewWalker(dir, nil)
	files, err := w.Walk()
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (real.go), got %d", len(files))
		for _, f := range files {
			t.Logf("  found: %s", f.RelPath)
		}
		return
	}
	if files[0].RelPath != "real.go" {
		t.Errorf("expected real.go, got %s", files[0].RelPath)
	}
}

func TestWalkLangFilter(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "main.go"))
	writeTestFile(t, filepath.Join(dir, "Main.java"))

	w := indexer.NewWalker(dir, []string{"go"})
	files, err := w.Walk()
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 go file, got %d", len(files))
		for _, f := range files {
			t.Logf("  found: %s (lang=%s)", f.RelPath, f.Lang)
		}
		return
	}
	if files[0].Lang != "go" {
		t.Errorf("expected lang=go, got %s", files[0].Lang)
	}
}

func TestWalkSkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".hidden", "foo.go"))
	writeTestFile(t, filepath.Join(dir, "visible", "bar.go"))

	w := indexer.NewWalker(dir, nil)
	files, err := w.Walk()
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (visible/bar.go), got %d", len(files))
		for _, f := range files {
			t.Logf("  found: %s", f.RelPath)
		}
		return
	}
	if files[0].RelPath != "visible/bar.go" {
		t.Errorf("expected visible/bar.go, got %s", files[0].RelPath)
	}
}

func TestWalkSkipsVendor(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "vendor", "lib.go"))
	writeTestFile(t, filepath.Join(dir, "pkg", "main.go"))

	w := indexer.NewWalker(dir, nil)
	files, err := w.Walk()
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (pkg/main.go), got %d", len(files))
		for _, f := range files {
			t.Logf("  found: %s", f.RelPath)
		}
		return
	}
	if files[0].RelPath != "pkg/main.go" {
		t.Errorf("expected pkg/main.go, got %s", files[0].RelPath)
	}
}

func TestWalkFindsJavaScriptFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "app.js"))
	writeTestFile(t, filepath.Join(dir, "module.mjs"))
	writeTestFile(t, filepath.Join(dir, "common.cjs"))
	writeTestFile(t, filepath.Join(dir, "component.jsx"))
	writeTestFile(t, filepath.Join(dir, "skip.ts")) // TypeScript not yet supported

	w := indexer.NewWalker(dir, nil)
	files, err := w.Walk()
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	langs := make(map[string]int)
	for _, f := range files {
		langs[f.Lang]++
	}
	if langs["javascript"] != 4 {
		t.Errorf("expected 4 javascript files, got %d; all: %+v", langs["javascript"], langs)
	}
	if langs["javascript"]+langs["go"]+langs["java"] != len(files) {
		t.Errorf("unexpected langs in results: %+v", langs)
	}
}

func TestWalkJavaScriptLangFilter(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "main.go"))
	writeTestFile(t, filepath.Join(dir, "app.js"))

	w := indexer.NewWalker(dir, []string{"go"})
	files, err := w.Walk()
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	for _, f := range files {
		if f.Lang == "javascript" {
			t.Errorf("javascript file should not be discovered with --lang go, found: %s", f.RelPath)
		}
	}
	if len(files) != 1 || files[0].Lang != "go" {
		t.Errorf("expected 1 go file, got %+v", files)
	}
}

func TestWalkJavaScriptOnlyFilter(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "main.go"))
	writeTestFile(t, filepath.Join(dir, "app.js"))
	writeTestFile(t, filepath.Join(dir, "Foo.java"))

	w := indexer.NewWalker(dir, []string{"javascript"})
	files, err := w.Walk()
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(files) != 1 || files[0].Lang != "javascript" {
		t.Errorf("expected 1 javascript file, got %+v", files)
	}
}

func TestWalkFindsHTMLFiles(t *testing.T) {
dir := t.TempDir()
writeTestFile(t, filepath.Join(dir, "index.html"))
writeTestFile(t, filepath.Join(dir, "about.htm"))
writeTestFile(t, filepath.Join(dir, "main.go"))

w := indexer.NewWalker(dir, nil)
files, err := w.Walk()
if err != nil {
t.Fatalf("Walk: %v", err)
}

langs := make(map[string]int)
for _, f := range files {
langs[f.Lang]++
}
if langs["html"] != 2 {
t.Errorf("expected 2 html files, got %d; all: %+v", langs["html"], langs)
}
}

func TestWalkHTMLLangFilter(t *testing.T) {
dir := t.TempDir()
writeTestFile(t, filepath.Join(dir, "index.html"))
writeTestFile(t, filepath.Join(dir, "main.go"))
writeTestFile(t, filepath.Join(dir, "App.java"))

w := indexer.NewWalker(dir, []string{"html"})
files, err := w.Walk()
if err != nil {
t.Fatalf("Walk: %v", err)
}
if len(files) != 1 || files[0].Lang != "html" {
t.Errorf("expected 1 html file, got %+v", files)
}
}
