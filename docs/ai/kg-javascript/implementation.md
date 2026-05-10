---
phase: implementation
title: Implementation Guide
description: Technical implementation notes, patterns, and code guidelines
---

# Implementation Guide

## Development Setup

- Worktree: `.worktrees/feature-kg-javascript` on branch `feature-kg-javascript`
- Build: `go build ./...` from worktree root
- Test: `go test ./...`
- Grammar: `github.com/tree-sitter/tree-sitter-javascript/bindings/go`

## Code Structure

```
internal/kg/indexer/javascript/
    parser.go          ← singleton ts.Language + ExtractJS()
    extractor.go       ← jsWalker AST traversal
    extractor_test.go  ← unit tests

internal/kg/lang/
    lang_javascript.go      ← JavaScriptIndexer (implements Indexer)
    resolve_javascript.go   ← JavaScriptResolver (no-op, implements Resolver)
```

Modified files:
- `internal/kg/indexer/discovery.go` — langForFile + walker defaults
- `pkg/kg/types.go` — LangJavaScript const
- `pkg/kg/resolve.go` — accept javascript
- `internal/command/kg.go` — "js" alias

## Implementation Notes

### Core Features
- **parser.go**: same pattern as `java/parser.go` — `sync.Once` singleton, `ts.NewLanguage(tsjs.Language())`.
- **extractor.go**: `jsWalker` with `walkNode` dispatch on tree-sitter node kinds:
  - `function_declaration` → function symbol + recurse body for calls
  - `class_declaration` / `class` → class symbol + recurse body
  - `method_definition` → method symbol (prefixed with class FQN)
  - `lexical_declaration` / `variable_declaration` → inspect declarator RHS:
    - `function` or `arrow_function` → function/arrow_function symbol
    - otherwise → variable symbol
  - `call_expression` → callsite row
  - `import_statement` → import path
  - `expression_statement` with `call_expression` using `require` → import path
- **FQN**: `<fileModule>.<name>` where `fileModule = reldir/basename` (no extension).
- **Visibility**: always `"public"` (JS has no access modifiers at file scope).

### Patterns & Best Practices
- Each `ExtractJS` call creates its own `ts.Parser` and `ts.Tree`; both are `defer`-closed.
- Confidence: `0.5`, Provenance: `"heuristic"` for all rows.
- Do not traverse into nested function bodies for symbol extraction (top-level + class methods only).

## Error Handling

Parse errors are non-fatal — return `(FileExtract{}, error)` and let the caller log + continue.
