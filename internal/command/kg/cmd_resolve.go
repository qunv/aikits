package kg

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"aikits/internal/config"
	pkgkg "aikits/pkg/kg"
)

func newKgResolveCmd(_ *config.Config, log *zap.Logger) *cobra.Command {
	var (
		langFlag          string
		budget            int
		mavenDownloadDeps bool
	)
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve callsites using gopls (semantic resolution)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := pkgkg.FindRepoRoot("")
			if err != nil {
				fmt.Fprintln(os.Stderr, "❌ "+err.Error())
				os.Exit(1)
			}

			kg, err := pkgkg.Open(repoRoot, log)
			if err != nil {
				handleOpenErr(err)
				return err
			}
			defer kg.Close()

			err = kg.Resolve(context.Background(), pkgkg.ResolveOptions{
				Lang:              pkgkg.Lang(langFlag),
				Budget:            budget,
				MavenDownloadDeps: mavenDownloadDeps,
			})
			if err != nil {
				var toolErr *pkgkg.ErrToolNotFound
				if errors.As(err, &toolErr) {
					fmt.Fprintf(os.Stderr, "⚠️  %s\n", toolErr.Error())
					os.Exit(2)
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&langFlag, "lang", "go", "language: go|java")
	cmd.Flags().IntVar(&budget, "budget", 1000, "maximum callsites to resolve")
	cmd.Flags().BoolVar(&mavenDownloadDeps, "maven-download-deps", false, "run 'mvn dependency:resolve' before jdtls startup (Java only)")
	return cmd
}
