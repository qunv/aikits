package lang

import (
	"database/sql"
	"fmt"

	kgdb "aikits/internal/kg/db"
	jsindexer "aikits/internal/kg/indexer/javascript"
)

// JavaScriptIndexer implements Indexer for JavaScript source files.
type JavaScriptIndexer struct{}

func (j *JavaScriptIndexer) Extract(src []byte, _, relPath string, repoID int64) (FileExtract, error) {
	r := jsindexer.ExtractJS(src, relPath, repoID, 0)
	return FileExtract{
		Symbols:     r.Symbols,
		Callsites:   r.Callsites,
		ImportPaths: r.ImportPaths,
		SrcPkgFQN:   r.SrcPkgFQN,
	}, nil
}

func (j *JavaScriptIndexer) StoreRefs(sqlDB *sql.DB, repoID, fileID int64, ext FileExtract) error {
	if ext.SrcPkgFQN != "" && len(ext.ImportPaths) > 0 {
		if err := kgdb.StoreImportRefs(sqlDB, repoID, fileID, ext.SrcPkgFQN, ext.ImportPaths); err != nil {
			return fmt.Errorf("store import refs: %w", err)
		}
	}
	return nil
}
