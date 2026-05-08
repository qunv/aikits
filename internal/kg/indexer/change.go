package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"aikits/internal/kg/db"
)

// ComputeSHA256 returns the hex SHA256 of the file at path.
func ComputeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileChanged returns true if the file's sha256 or mtime differs from the DB record.
func FileChanged(dbFile *db.FileRow, path string, info os.FileInfo, sha256sum string) bool {
	if dbFile == nil {
		return true
	}
	if info.ModTime().Unix() != dbFile.Mtime {
		return true
	}
	if sha256sum != dbFile.SHA256 {
		return true
	}
	return false
}
