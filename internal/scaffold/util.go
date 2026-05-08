package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func copyEmbeddedFile(fsys fs.FS, srcPath, destPath string) error {
	data, err := fs.ReadFile(fsys, srcPath)
	if err != nil {
		return fmt.Errorf("read embedded file %q: %w", srcPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", destPath, err)
	}

	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", destPath, err)
	}

	return nil
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
