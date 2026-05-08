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

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage aikits skills",
		Long:  `Install and list bundled AI workflow skills.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newSkillAddCmd())
	cmd.AddCommand(newSkillListCmd())
	return cmd
}

func newSkillAddCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "add <skill>",
		Short: "Install a skill to one or more agent directories",
		Long: `Copy a bundled skill into the target agent skills directory.

Each agent maps to a directory under the git root:
  copilot  →  <git-root>/.github/skills/<skill>/
  claude   →  <git-root>/.claude/skills/<skill>/

Use --agent to skip the interactive prompt (comma-separated, e.g. --agent copilot,claude).

Available skills: dev-lifecycle, tdd, verify

Existing files are preserved and reported as skipped.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			agents, err := resolveSkillAgents(agentFlag, term.IsTerminal(int(os.Stdin.Fd())))
			if err != nil {
				return err
			}

			agentNames := make([]string, len(agents))
			for i, a := range agents {
				agentNames[i] = a.Name()
			}

			results, err := pkgscaffold.SkillAdd(cmd.Context(), pkgscaffold.SkillAddOptions{
				SkillName: name,
				Agents:    agentNames,
			})
			for _, r := range results {
				if r.Err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "error installing to %s: %v\n", r.Agent, r.Err)
					continue
				}
				printSkillAddResult(r)
			}
			return err
		},
	}

	cmd.Flags().StringVar(&agentFlag, "agent", "", fmt.Sprintf("comma-separated agents to install for (e.g. copilot,claude); skips interactive prompt (supported: %s)", agent.Names()))
	return cmd
}

func newSkillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available built-in skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := pkgscaffold.SkillList(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Println("Available skills:")
			for _, n := range names {
				fmt.Printf("  - %s\n", n)
			}
			return nil
		},
	}
}

func printSkillAddResult(r pkgscaffold.SkillAddResult) {
	fmt.Printf("Installed skill %q for %s → %s\n", r.SkillName, r.Agent, r.DestDir)
	if len(r.Created) > 0 {
		fmt.Println()
		fmt.Println("  Created:")
		for _, p := range r.Created {
			fmt.Printf("    - %s\n", p)
		}
	}
	if len(r.Skipped) > 0 {
		fmt.Println()
		fmt.Println("  Skipped existing:")
		for _, p := range r.Skipped {
			fmt.Printf("    - %s\n", p)
		}
	}
}

// resolveSkillAgents returns the validated list of agents to install for.
func resolveSkillAgents(agentFlag string, isTTY bool) ([]agent.Agent, error) {
	if agentFlag != "" {
		return parseAgentFlag(agentFlag)
	}
	if !isTTY {
		return nil, fmt.Errorf("interactive prompt requires a terminal; use --agent flag (supported: %s)", agent.Names())
	}
	return promptSkillAgents()
}

// parseAgentFlag parses and validates a comma-separated agent string.
func parseAgentFlag(flag string) ([]agent.Agent, error) {
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

// promptSkillAgents shows an interactive multi-select and returns selected agents.
func promptSkillAgents() ([]agent.Agent, error) {
	all := agent.All()
	options := make([]huh.Option[string], len(all))
	for i, a := range all {
		options[i] = huh.NewOption(fmt.Sprintf("%s  (%s)", a.Name(), skillDestDir(a)), a.Name())
	}

	var selectedNames []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select agents to install for").
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

func skillDestDir(a agent.Agent) string {
	return a.DestDir() + "/skills"
}

// findGitRoot walks up from start until it finds a directory containing .git.
func findGitRoot(start string) (string, error) {
	return pkgscaffold.FindGitRoot(start)
}

// --- thin wrappers retained for test compatibility ---

type skillAddResult = pkgscaffold.SkillAddResult

func addSkill(name, destRoot string) (*skillAddResult, error) {
	r, err := pkgscaffold.AddSkillTo(context.Background(), name, destRoot)
	return r, err
}

func builtinSkillNames() ([]string, error) {
	return pkgscaffold.SkillList(context.Background())
}

