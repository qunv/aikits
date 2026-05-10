package command

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"aikits/internal/command/agent"
	pkgscaffold "aikits/pkg/scaffold"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newInitCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "init [target]",
		Short: "Initialize AI workflow docs and prompts",
		Long: `Create the standard AI workflow scaffold in a target directory.

This command writes:
  - <agent-dir>/prompts/*
  - <agent-dir>/instructions/*  (agent-specific, e.g. copilot → .github/)
  - docs/ai/README.md

Existing files are preserved and reported as skipped.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}

			agents, err := resolveInitAgents(agentFlag, term.IsTerminal(int(os.Stdin.Fd())))
			if err != nil {
				return err
			}

			agentNames := make([]string, len(agents))
			for i, a := range agents {
				agentNames[i] = a.Name()
			}

			result, err := pkgscaffold.Init(cmd.Context(), pkgscaffold.InitOptions{
				Target: target,
				Agents: agentNames,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Initialized AI workflow scaffold in %s\n", result.TargetDir)
			if len(result.Created) > 0 {
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintln(cmd.OutOrStdout(), "Created:")
				for _, path := range result.Created {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", path)
				}
			}
			if len(result.Skipped) > 0 {
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintln(cmd.OutOrStdout(), "Skipped existing:")
				for _, path := range result.Skipped {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", path)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", fmt.Sprintf("comma-separated agents to initialize for (e.g. copilot,claude); skips interactive prompt (supported: %s)", agent.Names()))
	return cmd
}

// resolveInitAgents returns the validated list of agents to initialize for.
func resolveInitAgents(agentFlag string, isTTY bool) ([]agent.Agent, error) {
	if agentFlag != "" {
		return parseInitAgentFlag(agentFlag)
	}
	if !isTTY {
		return nil, fmt.Errorf("interactive prompt requires a terminal; use --agent flag (supported: %s)", agent.Names())
	}
	return promptInitAgents()
}

// parseInitAgentFlag parses and validates a comma-separated agent string.
func parseInitAgentFlag(flag string) ([]agent.Agent, error) {
	seen := map[string]bool{}
	var result []agent.Agent
	for _, raw := range strings.Split(flag, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		a, err := agent.Get(name)
		if err != nil {
			return nil, err
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, a)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid agents specified — supported: %s", agent.Names())
	}
	sort.Slice(result, func(i, j int) bool {
		return agent.Index(result[i].Name()) < agent.Index(result[j].Name())
	})
	return result, nil
}

// promptInitAgents shows an interactive multi-select and returns selected agents.
func promptInitAgents() ([]agent.Agent, error) {
	all := agent.All()
	options := make([]huh.Option[string], len(all))
	for i, a := range all {
		options[i] = huh.NewOption(fmt.Sprintf("%s  (%s)", a.Name(), a.DestDir()), a.Name())
	}

	var selectedNames []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select agents to initialize for").
				Description("Space to toggle · Enter to confirm").
				Options(options...).
				Value(&selectedNames).
				Validate(func(v []string) error {
					if len(v) == 0 {
						return fmt.Errorf("select at least one agent")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("agent selection: %w", err)
	}

	sort.Slice(selectedNames, func(i, j int) bool {
		return agent.Index(selectedNames[i]) < agent.Index(selectedNames[j])
	})

	result := make([]agent.Agent, 0, len(selectedNames))
	for _, name := range selectedNames {
		a, err := agent.Get(name)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, nil
}

// initAgentIndex returns the canonical ordering index for the named agent.
func initAgentIndex(name string) int {
	return agent.Index(name)
}

// --- thin wrappers retained for test compatibility ---

type initResult = pkgscaffold.InitResult

func initWorkspace(target string) (*initResult, error) {
	return pkgscaffold.InitWorkspace(context.Background(), target)
}

func installAgentInstructions(a agent.Agent, target string) (*initResult, error) {
	return pkgscaffold.InstallAgent(context.Background(), target, a.Name())
}

