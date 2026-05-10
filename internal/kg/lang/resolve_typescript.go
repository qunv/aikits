package lang

import (
	"database/sql"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
)

// TypeScriptResolver implements Resolver for TypeScript.
// This is a no-op: TypeScript does not yet have an LSP-based semantic upgrade pass.
type TypeScriptResolver struct{}

func (t *TypeScriptResolver) Resolve(db *sql.DB, repo *kgdb.RepoRow, repoRoot string, budget int, log *zap.Logger) error {
	log.Info("typescript resolver: no-op (LSP resolution not yet supported)",
		zap.String("repo", repo.Name))
	return nil
}
