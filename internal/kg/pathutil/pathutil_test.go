package pathutil_test

import (
	"path/filepath"
	"testing"

	"aikits/internal/kg/pathutil"
)

func TestToSlash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a/b/c", "a/b/c"},
		{"a", "a"},
		{"", ""},
	}
	for _, tt := range tests {
		got := pathutil.ToSlash(tt.input)
		if got != tt.want {
			t.Errorf("ToSlash(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRepoRelative(t *testing.T) {
	root := "/home/user/project"
	abs := "/home/user/project/internal/foo/bar.go"

	rel, err := pathutil.RepoRelative(root, abs)
	if err != nil {
		t.Fatalf("RepoRelative: %v", err)
	}
	want := "internal/foo/bar.go"
	if rel != want {
		t.Errorf("RepoRelative = %q, want %q", rel, want)
	}
}

func TestRepoRelativeRootFile(t *testing.T) {
	root := filepath.FromSlash("/home/user/project")
	abs := filepath.FromSlash("/home/user/project/main.go")

	rel, err := pathutil.RepoRelative(root, abs)
	if err != nil {
		t.Fatalf("RepoRelative: %v", err)
	}
	if rel != "main.go" {
		t.Errorf("RepoRelative = %q, want %q", rel, "main.go")
	}
}
