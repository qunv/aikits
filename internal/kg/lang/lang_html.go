package lang

import (
	"database/sql"
	"fmt"

	kgdb "aikits/internal/kg/db"
	htmlindexer "aikits/internal/kg/indexer/html"
)

// HTMLIndexer implements Indexer for HTML source files.
type HTMLIndexer struct{}

func (h *HTMLIndexer) Extract(src []byte, _, relPath string, repoID int64) (FileExtract, error) {
	r := htmlindexer.ExtractHTML(src, relPath, repoID, 0)
	return FileExtract{
		Symbols:     r.Symbols,
		Callsites:   r.Callsites,
		ImportPaths: r.ImportPaths,
		SrcPkgFQN:   r.SrcPkgFQN,
	}, nil
}

func (h *HTMLIndexer) StoreRefs(sqlDB *sql.DB, repoID, fileID int64, ext FileExtract) error {
	if ext.SrcPkgFQN != "" && len(ext.ImportPaths) > 0 {
		if err := kgdb.StoreImportRefs(sqlDB, repoID, fileID, ext.SrcPkgFQN, ext.ImportPaths); err != nil {
			return fmt.Errorf("store import refs: %w", err)
		}
	}
	return nil
}
