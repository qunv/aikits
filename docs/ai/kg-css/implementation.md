---
phase: implementation
title: Implementation Guide
description: Technical implementation notes, patterns, and code guidelines
---

# Implementation Guide

## Development Setup

- Go module: `aikits` — build with `go build ./internal/... ./pkg/... ./cmd/...`
- Worktree: `.worktrees/feature-kg-css` on branch `feature-kg-css`
- New dependency: `github.com/tree-sitter/tree-sitter-css/bindings/go`

## Code Structure

```
internal/kg/indexer/
  css/
    parser.go          ← language singleton + ExtractCSS entry point
    extractor.go       ← cssWalker AST traversal
    extractor_test.go  ← table-driven unit tests
  discovery.go         ← add ".css" extension + default lang set
```

## Implementation Notes

### Core Features

- **parser.go**: Identical structure to `html/parser.go`; replace `tshtml` with `tscss` import, rename types/vars with `CSS` prefix.
- **extractor.go**: `walkNode` dispatches on `stylesheet`, `rule_set`, `at_rule`, `declaration` node kinds.
- **discovery.go**: Two-line change — add `.css` case to `langForFile` switch and `langMap["css"] = true` in the `NewWalker` default block.

### Patterns & Best Practices

- Use `sync.Once` for the language singleton (thread-safe, zero-allocation after first call).
- Return `CSSExtractResult` (not a pointer) — consistent with JS/HTML.
- All symbol visibility: `"public"`.
- Confidence/provenance: `0.5` / `"heuristic"`.

## Error Handling

- Skip `nil`, error, and missing nodes in `walkNode` (same guard as JS/HTML walkers).
- `@import` with no string child → silently skip (malformed CSS).

## Performance Considerations

- Single AST pass; no second traversal.
- Language singleton avoids repeated grammar initialisation across concurrent indexing goroutines.
