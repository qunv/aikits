package command

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	kgcmd "aikits/internal/command/kg"
	"aikits/internal/config"
)

// NewRootCmd builds the root cobra command and registers all sub-commands.
func NewRootCmd(cfg *config.Config, log *zap.Logger) *cobra.Command {
	root := &cobra.Command{
		Use:   "aikits",
		Short: "aikits – your AI-powered toolkit",
		Long:  `aikits is a CLI toolkit that provides AI-assisted utilities.`,
		// Run this when no sub-command is given.
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newLintCmd())
	root.AddCommand(newSkillCmd())
	root.AddCommand(newMemoryCmd(cfg, log))
	root.AddCommand(kgcmd.NewCmd(cfg, log))

	return root
}
