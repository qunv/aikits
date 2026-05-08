package lang

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
	"aikits/internal/kg/resolver"
)

// JavaResolver implements Resolver for Java using jdtls.
type JavaResolver struct {
	MavenDownloadDeps bool
}

func (j *JavaResolver) Resolve(db *sql.DB, repo *kgdb.RepoRow, repoRoot string, budget int, log *zap.Logger) error {
	dataDir := filepath.Join(repoRoot, ".kg", "jdtls-data")
	if mkErr := os.MkdirAll(dataDir, 0o755); mkErr != nil {
		return fmt.Errorf("create jdtls data dir: %w", mkErr)
	}

	// Maven diagnostics — only when a pom.xml is present.
	if _, statErr := os.Stat(filepath.Join(repoRoot, "pom.xml")); statErr == nil {
		mvnPath, mvnLookErr := exec.LookPath("mvn")
		if mvnLookErr != nil {
			if j.MavenDownloadDeps {
				return fmt.Errorf("--maven-download-deps requires 'mvn' in PATH: %w", ErrToolNotFound("mvn"))
			}
			fmt.Fprintln(os.Stderr, "⚠️  mvn not found in PATH; Maven classpath may be incomplete")
		} else {
			m2Repo := filepath.Join(os.Getenv("HOME"), ".m2", "repository")
			if _, m2Err := os.Stat(m2Repo); os.IsNotExist(m2Err) {
				fmt.Fprintln(os.Stderr, "⚠️  ~/.m2/repository not found; consider running 'mvn dependency:resolve'")
			}
			if j.MavenDownloadDeps {
				fmt.Fprintln(os.Stderr, "⚙️  Downloading Maven dependencies...")
				mvnCmd := exec.Command(mvnPath, "-q", "-DskipTests", "dependency:resolve")
				mvnCmd.Dir = repoRoot
				mvnCmd.Stdout = os.Stderr
				mvnCmd.Stderr = os.Stderr
				if runErr := mvnCmd.Run(); runErr != nil {
					fmt.Fprintf(os.Stderr, "⚠️  mvn dependency:resolve failed: %v\n", runErr)
				}
			}
		}
	}

	logDir := filepath.Join(repoRoot, ".kg", "logs")
	client, startErr := resolver.StartJdtls(repoRoot, dataDir, nil, logDir)
	if startErr != nil {
		return fmt.Errorf("could not start jdtls (set JDTLS_LAUNCHER_JAR, JDTLS_HOME, or install the jdtls wrapper script): %w", ErrToolNotFound("jdtls"))
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
				Provenance:  "jdtls",
			})
		}
		resolved++
	}
	if len(semanticEdges) > 0 {
		tx, txErr := db.Begin()
		if txErr == nil {
			if insErr := kgdb.UpsertSemanticEdges(tx, semanticEdges); insErr != nil {
				_ = tx.Rollback()
				log.Warn("upsert jdtls edges", zap.Error(insErr))
			} else {
				_ = tx.Commit()
			}
		}
	}
	fmt.Printf("resolved %d/%d callsites\n", resolved, len(callsites))

	// Build semantic OVERRIDES edges from implementation queries.
	implEdges, implErr := resolver.FindImplementationEdges(client, db, repo.ID, repoRoot, budget)
	if implErr != nil {
		log.Warn("find implementation edges", zap.Error(implErr))
	} else if len(implEdges) > 0 {
		tx, txErr := db.Begin()
		if txErr == nil {
			if insErr := kgdb.UpsertSemanticEdges(tx, implEdges); insErr != nil {
				_ = tx.Rollback()
				log.Warn("upsert impl edges", zap.Error(insErr))
			} else {
				_ = tx.Commit()
			}
		}
		fmt.Printf("inserted %d OVERRIDES edges\n", len(implEdges))
	}

	// Build semantic EXTENDS/IMPLEMENTS edges from type hierarchy.
	hierEdges, hierErr := resolver.FindTypeHierarchyEdges(client, db, repo.ID, repoRoot, budget)
	if hierErr != nil {
		log.Warn("find type hierarchy edges", zap.Error(hierErr))
	} else if len(hierEdges) > 0 {
		tx, txErr := db.Begin()
		if txErr == nil {
			if insErr := kgdb.UpsertSemanticEdges(tx, hierEdges); insErr != nil {
				_ = tx.Rollback()
				log.Warn("upsert hierarchy edges", zap.Error(insErr))
			} else {
				_ = tx.Commit()
			}
		}
		fmt.Printf("inserted %d type hierarchy edges\n", len(hierEdges))
	}

	return nil
}
