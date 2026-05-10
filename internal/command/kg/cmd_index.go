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

func newKgIndexCmd(_ *config.Config, log *zap.Logger) *cobra.Command {
	var (
		full     bool
		langFlag string
		jobs     int
	)
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index source files into the knowledge graph",
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

			langs := parseLangFlag(langFlag)
			result, err := kg.Index(context.Background(), pkgkg.IndexOptions{
				Full: full,
				Lang: langs,
				Jobs: jobs,
			})
			if err != nil {
				return err
			}

			fmt.Printf("indexed %d files, %d symbols, %d callsites (%d unchanged)\n",
				result.Indexed, result.Symbols, result.Callsites, result.Unchanged)
			if result.Errors > 0 {
				fmt.Fprintf(os.Stderr, "⚠️  %d file(s) failed to index\n", result.Errors)
				os.Exit(2)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&full, "full", false, "ignore cache and reindex all files")
	cmd.Flags().StringVar(&langFlag, "lang", "", "language filter: go,java,javascript (default: all)")
	cmd.Flags().IntVar(&jobs, "jobs", 0, "number of parallel workers (default: NumCPU)")
	return cmd
}

// parseLangFlag converts a comma-separated lang string (e.g. "go,java,javascript") to []pkgkg.Lang.
// "js" is accepted as an alias for "javascript".
func parseLangFlag(flag string) []pkgkg.Lang {
	if flag == "" {
		return nil
	}
	parts := splitComma(flag)
	langs := make([]pkgkg.Lang, 0, len(parts))
	for _, p := range parts {
		switch p {
		case "go":
			langs = append(langs, pkgkg.LangGo)
		case "java":
			langs = append(langs, pkgkg.LangJava)
		case "javascript", "js":
			langs = append(langs, pkgkg.LangJavaScript)
		}
	}
	return langs
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}
