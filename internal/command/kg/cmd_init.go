package kg

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"aikits/internal/config"
	pkgkg "aikits/pkg/kg"
)

func newKgInitCmd(_ *config.Config, log *zap.Logger) *cobra.Command {
	var reinit bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the knowledge graph database",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := pkgkg.FindRepoRoot("")
			if err != nil {
				fmt.Fprintln(os.Stderr, "❌ "+err.Error())
				os.Exit(1)
			}

			kg, err := pkgkg.Init(context.Background(), repoRoot, reinit, log)
			if err != nil {
				return err
			}
			defer kg.Close()

			fmt.Printf("✅ initialized .kg/kg.sqlite (repo: %s)\n", kg.RepoRoot())
			return nil
		},
	}
	cmd.Flags().BoolVar(&reinit, "reinit", false, "delete and recreate the database")
	return cmd
}
