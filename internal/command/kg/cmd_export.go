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

func newKgExportCmd(_ *config.Config, _ *zap.Logger) *cobra.Command {
	var (
		format   string
		output   string
		langFlag string
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the knowledge graph to JSON or GraphML",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := pkgkg.FindRepoRoot("")
			if err != nil {
				fmt.Fprintln(os.Stderr, "❌ "+err.Error())
				os.Exit(1)
			}

			kg, err := pkgkg.Open(repoRoot, nil)
			if err != nil {
				if errors.Is(err, pkgkg.ErrNotInitialized) {
					fmt.Fprintln(os.Stderr, "❌ "+err.Error())
					os.Exit(1)
				}
				return err
			}
			defer kg.Close()

			opts := pkgkg.ExportOptions{
				Format: pkgkg.ExportFormat(format),
				Output: output,
				Lang:   pkgkg.Lang(langFlag),
			}
			fmt.Fprintf(os.Stderr, "📄 exporting to %s\n", opts.Output)
			return kg.Export(context.Background(), opts)
		},
	}
	cmd.Flags().StringVar(&format, "format", "json", "output format: json|graphml")
	cmd.Flags().StringVar(&output, "output", "", "write to file instead of stdout")
	cmd.Flags().StringVar(&langFlag, "lang", "", "filter by language: go|java (default: all)")
	return cmd
}
