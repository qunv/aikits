package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"aikits/internal/config"
	"aikits/internal/memory"
	"aikits/internal/memory/db"
	"aikits/internal/memory/handlers"
	"aikits/internal/memory/types"
)

func newMemoryCmd(cfg *config.Config, log *zap.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "AI memory store (MCP server)",
		Long:  "Manage persistent AI knowledge via MCP tools (storeKnowledge, updateKnowledge, searchKnowledge).",
	}

	cmd.AddCommand(newMemoryStoreCmd(cfg, log))
	cmd.AddCommand(newMemoryUpdateCmd(cfg, log))
	cmd.AddCommand(newMemorySearchCmd(cfg, log))
	return cmd
}

func newMemoryStoreCmd(_ *config.Config, log *zap.Logger) *cobra.Command {
	var (
		dbPath  string
		title   string
		content string
		tags    []string
		scope   string
	)

	cmd := &cobra.Command{
		Use:   "store",
		Short: "Start the MCP server exposing the storeKnowledge tool",
		Long: `Starts the aikits memory MCP server exposing the memory.storeKnowledge tool.

When --title and --content are provided, stores a knowledge item directly from
the CLI instead of starting the MCP server. Useful for scripting or quick writes.

Add to your MCP client config (e.g. Claude Code, Cursor):

  {
    "mcpServers": {
      "memory": {
        "command": "aikits",
        "args": ["memory", "store"]
      }
    }
  }
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dbPath != "" {
				if _, err := db.Get(dbPath); err != nil {
					return fmt.Errorf("open db: %w", err)
				}
			}

			if title != "" || content != "" {
				return runDirectStore(title, content, tags, scope)
			}

			log.Info("starting aikits memory MCP server", zap.String("tool", "memory.storeKnowledge"))

			s := memory.NewServerWithTool("memory.storeKnowledge")
			if err := memory.Serve(s); err != nil {
				fmt.Fprintf(os.Stderr, "memory server error: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite db file (default: ~/.aikits/memory.db)")
	cmd.Flags().StringVar(&title, "title", "", "knowledge title (10-100 chars); triggers direct CLI store")
	cmd.Flags().StringVar(&content, "content", "", "knowledge content in markdown (50-5000 chars)")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "domain tags (comma-separated, max 10)")
	cmd.Flags().StringVar(&scope, "scope", "", `scope: "global", "project:<name>", or "repo:<name>" (default: "global")`)

	return cmd
}

func newMemoryUpdateCmd(_ *config.Config, log *zap.Logger) *cobra.Command {
	var (
		dbPath  string
		id      string
		title   string
		content string
		tags    []string
		scope   string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Start the MCP server exposing the updateKnowledge tool",
		Long: `Starts the aikits memory MCP server exposing the memory.updateKnowledge tool.

When --id is provided, updates a knowledge item directly from the CLI instead
of starting the MCP server. At least one of --title, --content, --tags, or
--scope must also be supplied.

Add to your MCP client config (e.g. Claude Code, Cursor):

  {
    "mcpServers": {
      "memory": {
        "command": "aikits",
        "args": ["memory", "update"]
      }
    }
  }
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dbPath != "" {
				if _, err := db.Get(dbPath); err != nil {
					return fmt.Errorf("open db: %w", err)
				}
			}

			if id != "" {
				return runDirectUpdate(cmd, id, title, content, tags, scope)
			}

			log.Info("starting aikits memory MCP server", zap.String("tool", "memory.updateKnowledge"))

			s := memory.NewServerWithTool("memory.updateKnowledge")
			if err := memory.Serve(s); err != nil {
				fmt.Fprintf(os.Stderr, "memory server error: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite db file (default: ~/.aikits/memory.db)")
	cmd.Flags().StringVar(&id, "id", "", "UUID of the knowledge item to update; triggers direct CLI update")
	cmd.Flags().StringVar(&title, "title", "", "new title (10-100 chars)")
	cmd.Flags().StringVar(&content, "content", "", "new content in markdown (50-5000 chars)")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "new tags, replaces existing (comma-separated, max 10)")
	cmd.Flags().StringVar(&scope, "scope", "", `new scope: "global", "project:<name>", or "repo:<name>"`)

	return cmd
}

func newMemorySearchCmd(_ *config.Config, log *zap.Logger) *cobra.Command {
	var (
		dbPath string
		query  string
		tags   []string
		scope  string
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Start the MCP server exposing the searchKnowledge tool",
		Long: `Starts the aikits memory MCP server exposing the memory.searchKnowledge tool.

When --query is provided, performs a direct CLI search and prints results
instead of starting the MCP server. Useful for scripting or quick lookups.

Add to your MCP client config (e.g. Claude Code, Cursor):

  {
    "mcpServers": {
      "memory": {
        "command": "aikits",
        "args": ["memory", "search"]
      }
    }
  }
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dbPath != "" {
				if _, err := db.Get(dbPath); err != nil {
					return fmt.Errorf("open db: %w", err)
				}
			}

			if query != "" {
				return runDirectSearch(query, tags, scope, limit)
			}

			log.Info("starting aikits memory MCP server", zap.String("tool", "memory.searchKnowledge"))

			s := memory.NewServerWithTool("memory.searchKnowledge")
			if err := memory.Serve(s); err != nil {
				fmt.Fprintf(os.Stderr, "memory server error: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "path to SQLite db file (default: ~/.aikits/memory.db)")
	cmd.Flags().StringVar(&query, "query", "", "search query (runs a direct CLI search instead of starting the MCP server)")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "context tags to boost matching results (comma-separated)")
	cmd.Flags().StringVar(&scope, "scope", "", `scope filter: "global", "project:<name>", or "repo:<name>"`)
	cmd.Flags().IntVar(&limit, "limit", 5, "maximum number of results (1-20)")

	return cmd
}

func runDirectStore(title, content string, tags []string, scope string) error {
	input := &types.StoreInput{
		Title:   title,
		Content: content,
		Tags:    tags,
		Scope:   scope,
	}

	result, err := handlers.Store(input)
	if err != nil {
		return fmt.Errorf("store failed: %w", err)
	}

	fmt.Printf("✅ %s\n", result.Message)
	fmt.Printf("   id: %s\n", result.ID)
	return nil
}

func runDirectUpdate(cmd *cobra.Command, id, title, content string, tags []string, scope string) error {
	input := &types.UpdateInput{ID: id}

	if cmd.Flags().Changed("title") {
		input.Title = &title
	}
	if cmd.Flags().Changed("content") {
		input.Content = &content
	}
	if cmd.Flags().Changed("tags") {
		input.Tags = tags
		input.TagsProvided = true
	}
	if cmd.Flags().Changed("scope") {
		input.Scope = &scope
	}

	result, err := handlers.Update(input)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("✅ %s\n", result.Message)
	fmt.Printf("   id: %s\n", result.ID)
	return nil
}

func runDirectSearch(query string, tags []string, scope string, limit int) error {
	input := &types.SearchInput{
		Query:       query,
		ContextTags: tags,
		Scope:       scope,
		Limit:       limit,
	}

	result, err := handlers.Search(input)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(result.Results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	for i, item := range result.Results {
		tagsStr := strings.Join(item.Tags, ", ")
		fmt.Printf("[%d] %s (score: %.3f)\n", i+1, item.Title, item.Score)
		if tagsStr != "" {
			fmt.Printf("    tags: %s\n", tagsStr)
		}
		fmt.Printf("    scope: %s  |  id: %s\n", item.Scope, item.ID)
		fmt.Printf("    %s\n\n", strings.ReplaceAll(strings.TrimSpace(item.Content), "\n", "\n    "))
	}

	return nil
}
