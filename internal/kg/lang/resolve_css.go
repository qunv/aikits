package lang

import (
	"database/sql"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
)

// CSSResolver implements Resolver for CSS.
// This is a no-op: CSS does not have an LSP-based semantic upgrade pass.
type CSSResolver struct{}

func (c *CSSResolver) Resolve(db *sql.DB, repo *kgdb.RepoRow, repoRoot string, budget int, log *zap.Logger) error {
	log.Info("css resolver: no-op (LSP resolution not supported for CSS)",
		zap.String("repo", repo.Name))
	return nil
}
