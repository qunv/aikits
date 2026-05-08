package indexer_test

import (
	"os"
	"path/filepath"
	"testing"

	"aikits/internal/kg/indexer"
)

func TestFileChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	if err := os.WriteFile(path, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, _ := os.Stat(path)
	sha, err := indexer.ComputeSHA256(path)
	if err != nil {
		t.Fatalf("ComputeSHA256: %v", err)
	}
	if sha == "" {
		t.Fatal("expected non-empty sha256")
	}

	// No db record => changed
	changed := indexer.FileChanged(nil, path, info, sha)
	if !changed {
		t.Error("nil db record should report changed")
	}
}

func TestComputeSHA256Consistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.go")
	content := []byte("package main\nfunc main() {}\n")
	_ = os.WriteFile(path, content, 0o644)

	sha1, err1 := indexer.ComputeSHA256(path)
	sha2, err2 := indexer.ComputeSHA256(path)
	if err1 != nil || err2 != nil {
		t.Fatalf("errors: %v, %v", err1, err2)
	}
	if sha1 != sha2 {
		t.Errorf("inconsistent SHA256: %s vs %s", sha1, sha2)
	}
	if len(sha1) != 64 {
		t.Errorf("expected 64-char hex SHA256, got %d chars", len(sha1))
	}
}

