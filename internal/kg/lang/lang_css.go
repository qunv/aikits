package lang

import (
	"database/sql"
	"fmt"

	kgdb "aikits/internal/kg/db"
	cssindexer "aikits/internal/kg/indexer/css"
)

// CSSIndexer implements Indexer for CSS source files.
type CSSIndexer struct{}

func (c *CSSIndexer) Extract(src []byte, _, relPath string, repoID int64) (FileExtract, error) {
	r := cssindexer.ExtractCSS(src, relPath, repoID, 0)
	return FileExtract{
		Symbols:     r.Symbols,
		Callsites:   r.Callsites,
		ImportPaths: r.ImportPaths,
		SrcPkgFQN:   r.SrcPkgFQN,
	}, nil
}

func (c *CSSIndexer) StoreRefs(sqlDB *sql.DB, repoID, fileID int64, ext FileExtract) error {
	if ext.SrcPkgFQN != "" && len(ext.ImportPaths) > 0 {
		if err := kgdb.StoreImportRefs(sqlDB, repoID, fileID, ext.SrcPkgFQN, ext.ImportPaths); err != nil {
			return fmt.Errorf("store import refs: %w", err)
		}
	}
	return nil
}
