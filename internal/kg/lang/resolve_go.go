package lang

import (
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
	"aikits/internal/kg/resolver"
)

// GoResolver implements Resolver for Go using gopls.
type GoResolver struct{}

func (g *GoResolver) Resolve(db *sql.DB, repo *kgdb.RepoRow, repoRoot string, budget int, log *zap.Logger) error {
	goplsPath, lookErr := exec.LookPath("gopls")
	if lookErr != nil {
		return fmt.Errorf("gopls not found in PATH; install with: go install golang.org/x/tools/gopls@latest: %w", ErrToolNotFound("gopls"))
	}

	rootURI := "file://" + filepath.ToSlash(repoRoot)
	logDir := filepath.Join(repoRoot, ".kg", "logs")
	client, startErr := resolver.Start(goplsPath, []string{}, rootURI, logDir)
	if startErr != nil {
		return fmt.Errorf("start gopls: %w", startErr)
	}
	defer client.Shutdown()

	callsites, csErr := kgdb.GetUnresolvedCallsites(db, repo.ID, budget)
	if csErr != nil {
		return fmt.Errorf("get unresolved callsites: %w", csErr)
	}

	resolved := 0
	var semanticEdges []kgdb.EdgeRow
	for i := range callsites {
		cs := &callsites[i]
		symbolID, resErr := resolver.ResolveCallsite(client, db, repo.ID, repoRoot, cs)
		if resErr != nil {
			log.Debug("resolve error", zap.Error(resErr))
			continue
		}
		if symbolID == 0 {
			continue
		}
		if updateErr := kgdb.UpdateCallsiteResolved(db, cs.ID, symbolID); updateErr != nil {
			log.Warn("update callsite", zap.Error(updateErr))
			continue
		}
		if cs.CallerSymbolID != nil {
			semanticEdges = append(semanticEdges, kgdb.EdgeRow{
				RepoID:      repo.ID,
				Kind:        "CALLS",
				SrcSymbolID: *cs.CallerSymbolID,
				DstSymbolID: symbolID,
				Confidence:  1.0,
				Provenance:  "gopls",
			})
		}
		resolved++
	}
	if len(semanticEdges) > 0 {
		tx, txErr := db.Begin()
		if txErr == nil {
			if insErr := kgdb.UpsertSemanticEdges(tx, semanticEdges); insErr != nil {
				_ = tx.Rollback()
				log.Warn("upsert gopls edges", zap.Error(insErr))
			} else {
				_ = tx.Commit()
			}
		}
	}
	fmt.Printf("resolved %d/%d callsites\n", resolved, len(callsites))
	return nil
}
