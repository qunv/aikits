---
phase: planning
title: Project Planning & Task Breakdown
description: Break down work into actionable tasks and estimate timeline
---

# Project Planning & Task Breakdown

## Milestones

- [ ] M1: Go module dependency wired; tree-sitter JS grammar available
- [ ] M2: JS extractor package complete with unit tests
- [ ] M3: Lang layer wired; discovery updated; CLI alias added
- [ ] M4: All existing tests green; feature tests passing; PR ready

## Task Breakdown

### Phase 1: Dependency & scaffolding
- [ ] T1.1: Add `github.com/tree-sitter/tree-sitter-javascript` to `go.mod` (`go get`)
- [ ] T1.2: Create `internal/kg/indexer/javascript/` directory

### Phase 2: Core extractor
- [ ] T2.1: `parser.go` — singleton language + `ExtractJS()` entry point (mirror `java/parser.go`)
- [ ] T2.2: `extractor.go` — AST walker for JS symbols, call-sites, imports
- [ ] T2.3: `extractor_test.go` — unit tests for all symbol kinds, call-sites, imports

### Phase 3: Lang & discovery wiring
- [ ] T3.1: `lang_javascript.go` — `JavaScriptIndexer` implementing `lang.Indexer`
- [ ] T3.2: `resolve_javascript.go` — no-op `JavaScriptResolver` implementing `lang.Resolver`
- [ ] T3.3: `discovery.go` — add JS extensions to `langForFile`; add `"javascript"` to walker defaults
- [ ] T3.4: `pkg/kg/types.go` — add `LangJavaScript`
- [ ] T3.5: `pkg/kg/resolve.go` — accept `"javascript"` without error
- [ ] T3.6: `internal/command/kg.go` — map `"js"` alias → `"javascript"` in `parseLangFlag`

### Phase 4: Verification
- [ ] T4.1: `go build ./...` passes
- [ ] T4.2: `go test ./...` passes (all existing + new tests)

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
