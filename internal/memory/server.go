package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	merrors "aikits/internal/memory/errors"
	"aikits/internal/memory/handlers"
	"aikits/internal/memory/types"
)

const (
	serverName    = "aikits-memory"
	serverVersion = "1.0.0"
)

// NewServer creates and wires the MCP memory server with all tools.
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(serverName, serverVersion,
		server.WithToolCapabilities(false),
	)

	s.AddTool(storeTool(), storeHandler)
	s.AddTool(updateTool(), updateHandler)
	s.AddTool(searchTool(), searchHandler)

	return s
}

// NewServerWithTool creates an MCP memory server exposing only the named tool.
// Valid tool names: "memory.storeKnowledge", "memory.updateKnowledge", "memory.searchKnowledge".
func NewServerWithTool(tool string) *server.MCPServer {
	s := server.NewMCPServer(serverName, serverVersion,
		server.WithToolCapabilities(false),
	)

	switch tool {
	case "memory.storeKnowledge":
		s.AddTool(storeTool(), storeHandler)
	case "memory.updateKnowledge":
		s.AddTool(updateTool(), updateHandler)
	case "memory.searchKnowledge":
		s.AddTool(searchTool(), searchHandler)
	}

	return s
}

// Serve runs the MCP server over stdin/stdout (blocking).
func Serve(s *server.MCPServer) error {
	return server.ServeStdio(s)
}

// ── Tool definitions ────────────────────────────────────────────────────────

func storeTool() mcp.Tool {
	return mcp.NewTool("memory.storeKnowledge",
		mcp.WithDescription("Store a new knowledge item. Use this to save actionable guidelines, rules, or patterns for future reference."),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Short, explicit description of the rule (10-100 chars)"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Detailed explanation in markdown format. Supports code blocks and examples. (50-5000 chars)"),
		),
		mcp.WithArray("tags",
			mcp.Description(`Optional domain keywords (e.g., ["api", "backend"]). Max 10 tags.`),
		),
		mcp.WithString("scope",
			mcp.Description(`Optional scope: "global", "project:<name>", or "repo:<name>". Default: "global"`),
		),
	)
}

func updateTool() mcp.Tool {
	return mcp.NewTool("memory.updateKnowledge",
		mcp.WithDescription("Update an existing knowledge item by ID. Use this to correct outdated or inaccurate knowledge."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("UUID of the knowledge item to update"),
		),
		mcp.WithString("title",
			mcp.Description("New title (10-100 chars). Only provide if changing."),
		),
		mcp.WithString("content",
			mcp.Description("New content in markdown format (50-5000 chars). Only provide if changing."),
		),
		mcp.WithArray("tags",
			mcp.Description("New tags (replaces existing). Only provide if changing. Max 10 tags."),
		),
		mcp.WithString("scope",
			mcp.Description(`New scope: "global", "project:<name>", or "repo:<name>". Only provide if changing.`),
		),
	)
}

func searchTool() mcp.Tool {
	return mcp.NewTool("memory.searchKnowledge",
		mcp.WithDescription("Search for relevant knowledge based on a task description. Returns ranked results."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Natural language task description to search for relevant knowledge (3-500 chars)"),
		),
		mcp.WithArray("contextTags",
			mcp.Description(`Optional tags to boost matching results (e.g., ["api", "backend"])`),
		),
		mcp.WithString("scope",
			mcp.Description("Optional project/repo scope filter. Results from this scope are prioritised."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (1-20, default: 5)"),
		),
	)
}

// ── Handlers ────────────────────────────────────────────────────────────────

func storeHandler(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := &types.StoreInput{
		Title:   req.GetString("title", ""),
		Content: req.GetString("content", ""),
		Tags:    stringSliceArg(req.GetArguments(), "tags"),
		Scope:   req.GetString("scope", ""),
	}

	result, err := handlers.Store(input)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(result), nil
}

func updateHandler(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	input := &types.UpdateInput{
		ID: req.GetString("id", ""),
	}
	if v, ok := args["title"].(string); ok {
		input.Title = &v
	}
	if v, ok := args["content"].(string); ok {
		input.Content = &v
	}
	if _, ok := args["tags"]; ok {
		input.Tags = stringSliceArg(args, "tags")
		input.TagsProvided = true
	}
	if v, ok := args["scope"].(string); ok {
		input.Scope = &v
	}

	result, err := handlers.Update(input)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(result), nil
}

func searchHandler(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := &types.SearchInput{
		Query:       req.GetString("query", ""),
		ContextTags: stringSliceArg(req.GetArguments(), "contextTags"),
		Scope:       req.GetString("scope", ""),
		Limit:       int(req.GetFloat("limit", 0)),
	}

	result, err := handlers.Search(input)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(result), nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func stringSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func jsonResult(v any) *mcp.CallToolResult {
	b, _ := json.MarshalIndent(v, "", "  ")
	return mcp.NewToolResultText(string(b))
}

func errorResult(err error) *mcp.CallToolResult {
	type errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}

	var code, msg string
	switch e := err.(type) {
	case *merrors.DuplicateError:
		code, msg = "DUPLICATE_KNOWLEDGE", e.Error()
	case *merrors.NotFoundError:
		code, msg = "NOT_FOUND", e.Error()
	case *merrors.ValidationError:
		code, msg = "VALIDATION_ERROR", e.Error()
	case *merrors.StorageError:
		code, msg = "STORAGE_ERROR", e.Error()
	default:
		code, msg = "INTERNAL_ERROR", fmt.Sprintf("%v", err)
	}

	b, _ := json.MarshalIndent(errResp{Error: code, Message: msg}, "", "  ")
	result := mcp.NewToolResultText(string(b))
	result.IsError = true
	return result
}
