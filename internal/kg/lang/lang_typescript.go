package lang

import (
	"database/sql"
	"fmt"

	kgdb "aikits/internal/kg/db"
	tsindexer "aikits/internal/kg/indexer/typescript"
)

// TypeScriptIndexer implements Indexer for TypeScript source files.
type TypeScriptIndexer struct{}

func (t *TypeScriptIndexer) Extract(src []byte, _, relPath string, repoID int64) (FileExtract, error) {
	r := tsindexer.ExtractTS(src, relPath, repoID, 0)
	return FileExtract{
		Symbols:     r.Symbols,
		Callsites:   r.Callsites,
		ImportPaths: r.ImportPaths,
		SrcPkgFQN:   r.SrcPkgFQN,
		TypeRefs:    r.TypeRefs,
	}, nil
}

func (t *TypeScriptIndexer) StoreRefs(sqlDB *sql.DB, repoID, fileID int64, ext FileExtract) error {
	if ext.SrcPkgFQN != "" && len(ext.ImportPaths) > 0 {
		if err := kgdb.StoreImportRefs(sqlDB, repoID, fileID, ext.SrcPkgFQN, ext.ImportPaths); err != nil {
			return fmt.Errorf("store import refs: %w", err)
		}
	}
	if len(ext.TypeRefs) > 0 {
		if err := kgdb.StoreTypeRefs(sqlDB, repoID, fileID, ext.TypeRefs); err != nil {
			return fmt.Errorf("store type refs: %w", err)
		}
	}
	return nil
}
