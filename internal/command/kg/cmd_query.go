package kg

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"aikits/internal/config"
	pkgkg "aikits/pkg/kg"
)

func newKgQueryCmd(_ *config.Config, _ *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query the knowledge graph",
	}
	cmd.AddCommand(newKgQuerySymbolCmd())
	cmd.AddCommand(newKgQueryCallersCmd())
	cmd.AddCommand(newKgQueryCalleesCmd())
	cmd.AddCommand(newKgQueryImpactCmd())
	cmd.AddCommand(newKgQueryImplsCmd())
	cmd.AddCommand(newKgQueryOverridesCmd())
	return cmd
}

func printNodes(nodes []pkgkg.SymbolNode) {
	if len(nodes) == 0 {
		fmt.Println("No results.")
		return
	}
	for _, n := range nodes {
		indent := strings.Repeat("  ", n.Depth)
		fmt.Printf("%s[%s] %s\n", indent, n.Symbol.Kind, n.Symbol.FQN)
	}
}

func newKgQuerySymbolCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "symbol <name|fqn>",
		Short: "Look up symbols by name or FQN",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kg, err := openKG(nil)
			if err != nil {
				return err
			}
			defer kg.Close()

			syms, err := kg.QuerySymbol(context.Background(), args[0])
			if err != nil {
				return err
			}
			if len(syms) == 0 {
				fmt.Println("No symbols found.")
				return nil
			}
			for _, s := range syms {
				fmt.Printf("[%s] %s  (%s, line %d)\n", s.Kind, s.FQN, s.Visibility, s.StartLine)
				if s.Signature != "" {
					fmt.Printf("    sig: %s\n", s.Signature)
				}
			}
			return nil
		},
	}
}

func newKgQueryCallersCmd() *cobra.Command {
	var depth, maxNodes int
	cmd := &cobra.Command{
		Use:   "callers <fqn>",
		Short: "Show callers of the given FQN",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kg, err := openKG(nil)
			if err != nil {
				return err
			}
			defer kg.Close()
			nodes, err := kg.Callers(context.Background(), args[0], depth, maxNodes)
			if err != nil {
				return err
			}
			printNodes(nodes)
			return nil
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 3, "traversal depth")
	cmd.Flags().IntVar(&maxNodes, "max-nodes", 100, "maximum nodes to return")
	return cmd
}

func newKgQueryCalleesCmd() *cobra.Command {
	var depth, maxNodes int
	cmd := &cobra.Command{
		Use:   "callees <fqn>",
		Short: "Show callees of the given FQN",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kg, err := openKG(nil)
			if err != nil {
				return err
			}
			defer kg.Close()
			nodes, err := kg.Callees(context.Background(), args[0], depth, maxNodes)
			if err != nil {
				return err
			}
			printNodes(nodes)
			return nil
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 3, "traversal depth")
	cmd.Flags().IntVar(&maxNodes, "max-nodes", 100, "maximum nodes to return")
	return cmd
}

func newKgQueryImpactCmd() *cobra.Command {
	var depth, maxNodes int
	cmd := &cobra.Command{
		Use:   "impact <fqn>",
		Short: "Show impact (dependents) of the given FQN",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kg, err := openKG(nil)
			if err != nil {
				return err
			}
			defer kg.Close()
			nodes, err := kg.Impact(context.Background(), args[0], depth, maxNodes)
			if err != nil {
				return err
			}
			printNodes(nodes)
			return nil
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 3, "traversal depth")
	cmd.Flags().IntVar(&maxNodes, "max-nodes", 100, "maximum nodes to return")
	return cmd
}

func newKgQueryImplsCmd() *cobra.Command {
	var depth, maxNodes int
	cmd := &cobra.Command{
		Use:   "impls <interface-fqn>",
		Short: "Show implementations of the given interface or type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kg, err := openKG(nil)
			if err != nil {
				return err
			}
			defer kg.Close()
			nodes, err := kg.Impls(context.Background(), args[0], depth, maxNodes)
			if err != nil {
				return err
			}
			printNodes(nodes)
			return nil
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 3, "traversal depth")
	cmd.Flags().IntVar(&maxNodes, "max-nodes", 100, "maximum nodes to return")
	return cmd
}

func newKgQueryOverridesCmd() *cobra.Command {
	var depth, maxNodes int
	cmd := &cobra.Command{
		Use:   "overrides <method-fqn>",
		Short: "Show methods that override the given method",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kg, err := openKG(nil)
			if err != nil {
				return err
			}
			defer kg.Close()
			nodes, err := kg.Overrides(context.Background(), args[0], depth, maxNodes)
			if err != nil {
				return err
			}
			printNodes(nodes)
			return nil
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 3, "traversal depth")
	cmd.Flags().IntVar(&maxNodes, "max-nodes", 100, "maximum nodes to return")
	return cmd
}
