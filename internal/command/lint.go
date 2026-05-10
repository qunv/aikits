package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newLintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint [target]",
		Short: "Verify the docs/ai/ directory exists",
		Long: `Checks that the base docs/ai/ scaffold is present.

Validates:
  - docs/ai/ directory exists

Exit code 0 when all checks pass, 1 otherwise.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) == 1 {
				target = args[0]
			}

			result, err := lintDocsAI(target)
			if err != nil {
				return err
			}

			printLintResult(result)

			if len(result.Errors) > 0 {
				return fmt.Errorf("lint failed with %d error(s)", len(result.Errors))
			}
			return nil
		},
	}
}

type lintCheck struct {
	Name   string
	Passed bool
	Detail string
}

type lintResult struct {
	TargetDir string
	Checks    []lintCheck
	Errors    []string
}

func lintDocsAI(target string) (*lintResult, error) {
	targetDir, err := filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("resolve target directory: %w", err)
	}

	result := &lintResult{TargetDir: targetDir}
	docsAIDir := filepath.Join(targetDir, "docs", "ai")

	// Check docs/ai/ directory exists
	if !dirExists(docsAIDir) {
		result.addFail("docs/ai/ directory", "directory does not exist")
		return result, nil
	}
	result.addPass("docs/ai/ directory", "exists")

	return result, nil
}

func (r *lintResult) addPass(name, detail string) {
	r.Checks = append(r.Checks, lintCheck{Name: name, Passed: true, Detail: detail})
}

func (r *lintResult) addFail(name, detail string) {
	r.Checks = append(r.Checks, lintCheck{Name: name, Passed: false, Detail: detail})
	r.Errors = append(r.Errors, fmt.Sprintf("%s: %s", name, detail))
}

func printLintResult(result *lintResult) {
	fmt.Printf("Linting docs/ai/ in %s\n\n", result.TargetDir)
	for _, c := range result.Checks {
		icon := "✅"
		if !c.Passed {
			icon = "❌"
		}
		fmt.Printf("  %s %s — %s\n", icon, c.Name, c.Detail)
	}

	passed := len(result.Checks) - len(result.Errors)
	fmt.Printf("\n%d/%d checks passed\n", passed, len(result.Checks))
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
