---
phase: requirements
title: Requirements & Problem Understanding
description: Clarify the problem space, gather requirements, and define success criteria
---

# Requirements & Problem Understanding

## Problem Statement

The `aikits kg` knowledge graph currently supports Go, Java, and JavaScript. HTML is
ubiquitous in web projects — every frontend repo contains `.html` files that wire together
scripts, styles, and custom elements. Developers working in those repos cannot use
`aikits kg` to navigate or analyse their HTML files.

**Who is affected?** Any team using aikits on repos that contain `.html` files (static
sites, full-stack apps, template-based projects).

**Current workaround:** None — `.html` files are silently skipped by the walker.

## Goals & Objectives

### Primary goals
- Parse HTML source files with `tree-sitter-html` (`github.com/tree-sitter/tree-sitter-html`)
  and extract:
  - **Script-src references** (`<script src="...">`) as import edges.
  - **Link/stylesheet references** (`<link href="...">`) as import edges.
  - **Element IDs** (`id="..."`) as symbols (kind: `id`, visibility: `public`).
  - **Custom element usage** (tags containing `-`, e.g. `<my-button>`) as call-site edges.
  - **Inline `<script>` blocks** — delegate to the JS extractor for symbol/call-site extraction.
- Register `html` as a first-class `Lang` constant alongside `go`, `java`, and `javascript`.
- Honour `--lang html` in all `aikits kg` sub-commands (`index`, `query`, `export`).
- Include `.html` and `.htm` in the automatic file walker when no `--lang` filter is specified.

### Secondary goals
- No-op resolver: `aikits kg resolve --lang html` should succeed without crashing
  (HTML has no LSP-based semantic upgrade in this release).

### Non-goals
- CSS extraction from `<style>` blocks — out of scope; separate feature.
- Template engines (Jinja2, Go templates, ERB) — out of scope.
- Full DOM relationship graph — out of scope.
- TypeScript inside `<script type="ts">` — out of scope.

## User Stories & Use Cases

- **As a developer**, I want to run `aikits kg index --lang html` on my static site so
  that I can navigate element IDs and script references with `aikits kg query`.
- **As a developer**, I want `aikits kg index` (no `--lang`) in a full-stack repo to index
  Go, Java, JavaScript, and HTML files in a single pass.
- **As a developer**, I want `aikits kg query symbol myId` to return results for element
  IDs defined in `.html` files.
- **As a developer**, I want `aikits kg query callers my-button` to see all HTML files
  that use the `<my-button>` custom element.

## Success Criteria

- `aikits kg index --lang html` completes without errors on an HTML-heavy repo and
  populates symbols with `lang = "html"`.
- `aikits kg query symbol <id>` surfaces HTML element-ID symbols.
- `aikits kg resolve --lang html` exits 0 (no-op, with informational log).
- All existing Go, Java, and JavaScript tests continue to pass.
- New unit tests cover: element-ID extraction, script-src import edges, link-href import
  edges, custom-element call-sites, inline-script delegation, `langForFile` routing for
  `.html`/`.htm`, and walker inclusion.

## Constraints & Assumptions

- **Library**: `github.com/tree-sitter/tree-sitter-html/bindings/go` — must be added as
  a direct `go.mod` dependency.
- **Pattern**: mirror the JavaScript indexer structure (`internal/kg/indexer/html/`,
  `internal/kg/lang/lang_html.go`).
- Confidence and provenance for heuristic extraction: `0.5` / `"heuristic"` (same as JS).
- FQN scheme for element IDs: `<reldir>/<basename_no_ext>#<id>` (e.g. `pages/index#hero`).
- FQN scheme for custom-element call-sites: `<reldir>/<basename_no_ext>.<tagName>`.
- Inline `<script>` blocks re-use the JS extractor; their symbols/call-sites inherit the
  HTML file's `fileID` but use JS FQN conventions.

## Questions & Open Items

- None — scope and approach are clear.
