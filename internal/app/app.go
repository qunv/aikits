package app

import (
	"fmt"
	"os"

	"aikits/internal/command"
	"aikits/internal/config"
	"aikits/pkg/logger"
)

// Run initializes the application and executes the root command.
func Run() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	root := command.NewRootCmd(cfg, log)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
