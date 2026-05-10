---
phase: planning
title: Project Planning & Task Breakdown
description: Break down work into actionable tasks and estimate timeline
---

# Project Planning & Task Breakdown

## Milestones

- [x] M1: Go module dependency wired; tree-sitter JS grammar available
- [x] M2: JS extractor package complete with unit tests
- [x] M3: Lang layer wired; discovery updated; CLI alias added
- [x] M4: All existing tests green; feature tests passing; PR ready

## Task Breakdown

### Phase 1: Dependency & scaffolding
- [x] T1.1: Add `github.com/tree-sitter/tree-sitter-javascript` to `go.mod` (`go get`)
- [x] T1.2: Create `internal/kg/indexer/javascript/` directory

### Phase 2: Core extractor
- [x] T2.1: `parser.go` — singleton language + `ExtractJS()` entry point (mirror `java/parser.go`)
- [x] T2.2: `extractor.go` — AST walker for JS symbols, call-sites, imports
- [x] T2.3: `extractor_test.go` — unit tests for all symbol kinds, call-sites, imports

### Phase 3: Lang & discovery wiring
- [x] T3.1: `lang_javascript.go` — `JavaScriptIndexer` implementing `lang.Indexer`
- [x] T3.2: `resolve_javascript.go` — no-op `JavaScriptResolver` implementing `lang.Resolver`
- [x] T3.3: `discovery.go` — add JS extensions to `langForFile`; add `"javascript"` to walker defaults
- [x] T3.4: `pkg/kg/types.go` — add `LangJavaScript`
- [x] T3.5: `pkg/kg/resolve.go` — accept `"javascript"` without error
- [x] T3.6: `internal/command/kg/cmd_index.go` — map `"js"` alias → `"javascript"` in `parseLangFlag`

### Phase 4: Verification
- [x] T4.1: `go build ./internal/... ./pkg/... ./cmd/...` passes
- [x] T4.2: `go test ./internal/... ./pkg/...` passes (all existing + new tests)

## Dependencies

- T2.1 → T1.1 (needs grammar in module graph)
- T2.2 → T2.1
- T2.3 → T2.2
- T3.1 → T2.1
- T3.2 → T3.1
- T3.3 → T3.1 (needs lang constant)
- T3.4 → (independent)
- T3.5 → T3.4
- T3.6 → T3.4
- T4.x → all T3

## Risks & Mitigation

| Risk | Mitigation |
|---|---|
| `tree-sitter-javascript` Go binding import path differs from go.sum entry | Verify with `go get` and check `bindings/go` sub-path |
| JS AST node kinds differ between tree-sitter-javascript versions | Pin to version in go.sum; write tests against real AST |
| `.jsx` AST uses JSX-specific node kinds that break walker | Wrap JSX nodes in a fallback / skip in `walkNode` |
