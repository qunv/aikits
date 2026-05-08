package lang

import (
	"database/sql"
	"fmt"

	kgdb "aikits/internal/kg/db"
	javaindexer "aikits/internal/kg/indexer/java"
)

// JavaIndexer implements Indexer for Java source files.
type JavaIndexer struct{}

func (j *JavaIndexer) Extract(src []byte, _, relPath string, repoID int64) (FileExtract, error) {
	r := javaindexer.ExtractJava(src, repoID, 0)
	return FileExtract{
		Symbols:        r.Symbols,
		Callsites:      r.Callsites,
		ImportPaths:    r.ImportPaths,
		SrcPkgFQN:      r.SrcPkgFQN,
		ExtendsRefs:    r.ExtendsRefs,
		ImplementsRefs: r.ImplRefs,
		TypeRefs:       r.TypeRefs,
	}, nil
}

func (j *JavaIndexer) StoreRefs(sqlDB *sql.DB, repoID, fileID int64, ext FileExtract) error {
	if ext.SrcPkgFQN != "" && len(ext.ImportPaths) > 0 {
		if err := kgdb.StoreImportRefs(sqlDB, repoID, fileID, ext.SrcPkgFQN, ext.ImportPaths); err != nil {
			return fmt.Errorf("store import refs: %w", err)
		}
	}
	if len(ext.ExtendsRefs) > 0 {
		if err := kgdb.StoreExtendsRefs(sqlDB, repoID, fileID, ext.ExtendsRefs); err != nil {
			return fmt.Errorf("store extends refs: %w", err)
		}
	}
	if len(ext.ImplementsRefs) > 0 {
		if err := kgdb.StoreImplementsRefs(sqlDB, repoID, fileID, ext.ImplementsRefs); err != nil {
			return fmt.Errorf("store implements refs: %w", err)
		}
	}
	if len(ext.TypeRefs) > 0 {
		if err := kgdb.StoreTypeRefs(sqlDB, repoID, fileID, ext.TypeRefs); err != nil {
			return fmt.Errorf("store type refs: %w", err)
		}
	}
	return nil
}
