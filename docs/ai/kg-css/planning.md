---
phase: planning
title: Project Planning & Task Breakdown
description: Break down work into actionable tasks and estimate timeline
---

# Project Planning & Task Breakdown

## Milestones

- [x] M1: tree-sitter-css dependency added and compiles
- [x] M2: CSS extractor implemented with full symbol/import extraction
- [x] M3: Walker and discovery wired up; `aikits kg index --lang css` works end-to-end
- [x] M4: Unit tests pass; all existing tests green

## Task Breakdown

### Phase 1: Dependency
- [x] T1.1: Add `github.com/tree-sitter/tree-sitter-css` to `go.mod` / `go.sum` via `go get`
  - Also upgraded `github.com/tree-sitter/go-tree-sitter` to v0.25.0 (required for ABI 15 support)

### Phase 2: Core Extractor
- [x] T2.1: Create `internal/kg/indexer/css/parser.go` — language singleton + `ExtractCSS`
- [x] T2.2: Create `internal/kg/indexer/css/extractor.go` — `cssWalker` with:
  - class-selector extraction from `rule_set`
  - ID-selector extraction from `rule_set`
  - `@keyframes` name extraction
  - `@import` path extraction
  - CSS custom-property (`--var`) extraction

### Phase 3: Discovery Wiring
- [x] T3.1: Update `internal/kg/indexer/discovery.go` — add `.css` to `langForFile` and default lang set
- [x] T3.2: Create `internal/kg/lang/lang_css.go` and `resolve_css.go`; wire into `pkg/kg/index.go`, `resolve.go`, `types.go`

### Phase 4: Tests
- [x] T4.1: Create `internal/kg/indexer/css/extractor_test.go` — 19 table-driven tests for all extraction cases
- [x] T4.2: Update `internal/kg/indexer/discovery_test.go` — add CSS file routing cases

## Dependencies

- T2.1 → T1.1 (need the Go bindings to compile)
- T2.2 → T2.1
- T3.1 → T2.2 (need `ExtractCSS` signature stable)
- T3.2 → T3.1
- T4.x → T2.2, T3.1

## Risks & Mitigation

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| tree-sitter-css Go bindings missing or broken | Low | Verify import path from GitHub before T1.1 |
| CSS node kinds differ from expected names | Medium | Inspect grammar with a small debug parse before writing extractor |
| Custom-property scope logic complex | Low | Limit to top-level declarations only; expand later |

## Resources Needed

- `github.com/tree-sitter/tree-sitter-css` — Go bindings at `bindings/go`
- tree-sitter-css grammar reference: https://github.com/tree-sitter/tree-sitter-css
