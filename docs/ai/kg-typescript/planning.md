---
phase: planning
title: Project Planning & Task Breakdown
description: Break down work into actionable tasks and estimate timeline
---

# Project Planning & Task Breakdown

## Milestones

- [x] M1: Dependency added and build passes
- [x] M2: TypeScript extractor parses .ts and .tsx files correctly
- [x] M3: TypeScript wired into kg index/resolve pipeline
- [x] M4: All tests pass (new + existing)

## Task Breakdown

### Foundation
- [x] T1: Add `github.com/tree-sitter/tree-sitter-typescript` to go.mod (worktree)
- [x] T2: Write failing extractor tests (`internal/kg/indexer/typescript/extractor_test.go`)

### Core Implementation
- [x] T3: Create `internal/kg/indexer/typescript/parser.go` (ExtractTS, dual grammar)
- [x] T4: Create `internal/kg/indexer/typescript/extractor.go` (tsWalker + all node kinds)
- [x] T5: Create `internal/kg/lang/lang_typescript.go` (TypeScriptIndexer)
- [x] T6: Create `internal/kg/lang/resolve_typescript.go` (TypeScriptResolver no-op)

### Integration
- [x] T7: Update `internal/kg/indexer/discovery.go` (langForFile + default walker)
- [x] T8: Update `pkg/kg/types.go` (LangTypeScript constant)
- [x] T9: Update `pkg/kg/index.go` (register TypeScriptIndexer)
- [x] T10: Update `pkg/kg/resolve.go` (register TypeScriptResolver)

### Validation
- [x] T11: Update `internal/kg/indexer/discovery_test.go` (.ts/.tsx routing)
- [x] T12: Build and run all kg tests — all 15 packages pass

## Summary

All tasks complete. 24 new extractor tests + 3 new discovery tests added. All 15 kg packages
pass. No regressions. Implementation follows the JS indexer pattern exactly, with additions for
`interface`, `type_alias`, `enum`, and `enum_member` node kinds plus TSX grammar selection.

## Risks & Mitigation

- **tree-sitter-typescript Go binding API**: resolved — `LanguageTypescript()` and `LanguageTSX()`
  confirmed from `v0.23.2` binding source.

