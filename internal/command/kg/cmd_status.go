package kg

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"aikits/internal/config"
	pkgkg "aikits/pkg/kg"
)

func newKgStatusCmd(_ *config.Config, _ *zap.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show knowledge graph status",
		RunE: func(cmd *cobra.Command, args []string) error {
			kg, err := openKG(nil)
			if err != nil {
				return err
			}
			defer kg.Close()

			st, err := kg.Status(context.Background())
			if err != nil {
				return err
			}

			pct := 0.0
			if st.Callsites > 0 {
				pct = float64(st.Resolved) / float64(st.Callsites) * 100
			}
			lastIndexed := "never"
			if st.LastIndexed != nil {
				lastIndexed = st.LastIndexed.Format(pkgkg.TimeFormat)
			}
			fmt.Printf("Repository:      %s (%s)\n", st.RepoName, st.RootPath)
			fmt.Printf("Files indexed:   %d\n", st.Files)
			fmt.Printf("Symbols:         %d\n", st.Symbols)
			fmt.Printf("Callsites:       %d total, %d resolved (%.0f%%)\n", st.Callsites, st.Resolved, pct)
			fmt.Printf("Last indexed:    %s\n", lastIndexed)
			return nil
		},
	}
}
