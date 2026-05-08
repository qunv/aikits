package kg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"aikits/internal/config"
	pkgkg "aikits/pkg/kg"
)

// NewCmd returns the root "kg" command with all subcommands attached.
func NewCmd(cfg *config.Config, log *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kg",
		Short: "Knowledge Graph indexer and query engine",
		Long:  "Index your codebase into a knowledge graph and query symbols, call graphs, and impact analysis.",
	}
	cmd.AddCommand(newKgInitCmd(cfg, log))
	cmd.AddCommand(newKgIndexCmd(cfg, log))
	cmd.AddCommand(newKgStatusCmd(cfg, log))
	cmd.AddCommand(newKgResolveCmd(cfg, log))
	cmd.AddCommand(newKgQueryCmd(cfg, log))
	cmd.AddCommand(newKgExportCmd(cfg, log))
	return cmd
}

// openKG resolves the repo root from cwd, opens the KG, and handles typed
// errors with appropriate exit codes. Returns nil on error (exit already called).
func openKG(log *zap.Logger) (*pkgkg.KG, error) {
	repoRoot, err := pkgkg.FindRepoRoot("")
	if err != nil {
		fmt.Fprintln(os.Stderr, "❌ "+err.Error())
		os.Exit(1)
	}
	kg, err := pkgkg.Open(repoRoot, log)
	if err != nil {
		if errors.Is(err, pkgkg.ErrNotInitialized) {
			fmt.Fprintln(os.Stderr, "❌ "+err.Error())
			os.Exit(1)
		}
		var sve *pkgkg.ErrSchemaMismatch
		if errors.As(err, &sve) {
			fmt.Fprintln(os.Stderr, "❌ "+sve.Error())
			os.Exit(3)
		}
		return nil, fmt.Errorf("open kg: %w", err)
	}
	return kg, nil
}

// handleOpenErr inspects a pkgkg.Open error and calls os.Exit with the
// appropriate code. It does nothing if err is nil.
func handleOpenErr(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, pkgkg.ErrNotInitialized) {
		fmt.Fprintln(os.Stderr, "❌ "+err.Error())
		os.Exit(1)
	}
	var sve *pkgkg.ErrSchemaMismatch
	if errors.As(err, &sve) {
		fmt.Fprintln(os.Stderr, "❌ "+sve.Error())
		os.Exit(3)
	}
}

// kgDBPath is kept for backward-compatible use in the export command's default
// path resolution. It returns the path to kg.sqlite without opening the DB.
func kgDBPath(repoRoot string) (string, error) {
	kgDir := filepath.Join(repoRoot, ".kg")
	if _, err := os.Stat(kgDir); os.IsNotExist(err) {
		return "", pkgkg.ErrNotInitialized
	}
	return filepath.Join(kgDir, "kg.sqlite"), nil
}
