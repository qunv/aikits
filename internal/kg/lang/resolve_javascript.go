package lang

import (
	"database/sql"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
)

// JavaScriptResolver implements Resolver for JavaScript.
// This is a no-op: JavaScript does not yet have an LSP-based semantic upgrade pass.
type JavaScriptResolver struct{}

func (j *JavaScriptResolver) Resolve(db *sql.DB, repo *kgdb.RepoRow, repoRoot string, budget int, log *zap.Logger) error {
	log.Info("javascript resolver: no-op (LSP resolution not yet supported)",
		zap.String("repo", repo.Name))
	return nil
}
