package lang

import (
	"database/sql"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
)

// HTMLResolver implements Resolver for HTML.
// This is a no-op: HTML does not have an LSP-based semantic upgrade pass.
type HTMLResolver struct{}

func (h *HTMLResolver) Resolve(db *sql.DB, repo *kgdb.RepoRow, repoRoot string, budget int, log *zap.Logger) error {
	log.Info("html resolver: no-op (LSP resolution not supported for HTML)",
		zap.String("repo", repo.Name))
	return nil
}
