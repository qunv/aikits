---
phase: planning
title: Project Planning & Task Breakdown
description: Break down work into actionable tasks and estimate timeline
---

# Project Planning & Task Breakdown

## Milestones

- [x] M1: tree-sitter-html dependency added and HTML parser skeleton compiles
- [x] M2: HTML extractor produces correct symbols, imports, and call-sites
- [x] M3: `html` registered as a Lang; walker and indexer route HTML files
- [x] M4: No-op resolver in place; all existing tests pass
- [x] M5: Full test coverage; feature complete

## Task Breakdown

### Foundation
- [x] T1: Add `github.com/tree-sitter/tree-sitter-html/bindings/go` to `go.mod` / `go.sum`
- [x] T2: Create `internal/kg/indexer/html/parser.go` — singleton language + `ExtractHTML` stub
- [x] T3: Create `internal/kg/lang/lang_html.go` — `HTML` constant and descriptor

### Core Extraction
- [x] T4: Implement `htmlWalker` in `internal/kg/indexer/html/extractor.go`
  - Walk `script_element` with `src` attribute → import edge
  - Walk `element` with `href` attribute on `<link>` → import edge
  - Walk any element with `id` attribute → symbol (kind `id`)
  - Walk custom-element start tags (tag-name contains `-`) → call-site
  - Walk `script_element` without `src` → extract text, delegate to `javascript.ExtractJS`

### Integration
- [x] T5: Extend `langForFile` in `discovery.go` to return `"html"` for `.html` / `.htm`
- [x] T6: Extend `NewWalker` default lang set to include `"html"`
- [x] T7: Route `"html"` case in `indexer.go` to `html.ExtractHTML`
- [x] T8: Create `internal/kg/lang/resolve_html.go` — no-op resolver
- [x] T9: Register `html` in `lang.go` lang registry / switch statements

### Tests
- [x] T10: `extractor_test.go` — element-ID symbol extraction
- [x] T11: `extractor_test.go` — `<script src>` import edge
- [x] T12: `extractor_test.go` — `<link href>` import edge
- [x] T13: `extractor_test.go` — custom-element call-site
- [x] T14: `extractor_test.go` — inline `<script>` JS delegation
- [x] T15: `discovery_test.go` — walker finds `.html` / `.htm` files
- [x] T16: `discovery_test.go` — `--lang html` filter

## Dependencies

- T2, T3 depend on T1
- T4 depends on T2
- T5, T6 depend on T3
- T7 depends on T4, T5, T6
- T8, T9 depend on T3
- Tests (T10–T16) depend on T4–T9

## Risks & Mitigation

| Risk | Mitigation |
|------|-----------|
| `tree-sitter-html` Go binding API differs from JS binding | Check binding source before writing parser.go |
| Inline `<script>` text extraction is non-trivial in AST | Use `Named()` child traversal to locate text nodes within `raw_text` |
| Custom-element heuristic (contains `-`) produces false positives | Acceptable for v1; can be tightened later with an allowlist |
