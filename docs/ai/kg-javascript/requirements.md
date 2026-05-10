---
phase: requirements
title: Requirements & Problem Understanding
description: Clarify the problem space, gather requirements, and define success criteria
---

# Requirements & Problem Understanding

## Problem Statement

The `aikits kg` knowledge graph currently supports only Go and Java. JavaScript is the
world's most widely-used language and appears in many mixed-language repos (frontend
apps, Node.js services, scripts). Developers working in those repos cannot use `aikits kg`
to navigate or analyse their JS code.

**Who is affected?** Any team using aikits on repos that contain `.js`, `.mjs`, `.cjs`,
or `.jsx` files alongside (or instead of) Go/Java.

**Current workaround:** None — JS files are silently skipped by the walker.

## Goals & Objectives

### Primary goals
- Parse JavaScript source files with tree-sitter and extract symbols (functions, classes,
  variables, arrow-function properties) and call-site edges into the kg database.
- Register `javascript` as a first-class `Lang` constant alongside `go` and `java`.
- Honour `--lang javascript` (and the short alias `js`) in all `aikits kg` sub-commands
  (`index`, `resolve`, `export`, `query`).
- Include `.js`, `.mjs`, `.cjs`, and `.jsx` in the automatic file walker when no `--lang`
  filter is specified.

### Secondary goals
- No-op resolver: JavaScript does not have an LSP-based semantic upgrade pass in this
  release; `aikits kg resolve --lang javascript` should succeed without crashing.

### Non-goals
- TypeScript support (`.ts`, `.tsx`) — out of scope; separate feature.
- Module-resolution / cross-file import graph — the existing heuristic approach is
  sufficient; full npm-module resolution is out of scope.
- Test files (`*.test.js`, `*.spec.js`) are indexed but not given special treatment.

## User Stories & Use Cases

- **As a developer**, I want to run `aikits kg index --lang javascript` on my Node.js
  project so that I can navigate symbols and call graphs with `aikits kg query`.
- **As a developer**, I want `aikits kg index` (no `--lang`) in a full-stack repo to index
  Go, Java, and JavaScript files in a single pass.
- **As a developer**, I want `aikits kg query symbol myFunc` to return results for
  JavaScript functions extracted from `.js` files.

## Success Criteria

- `aikits kg index --lang javascript` completes without errors on a JS-heavy repo and
  populates symbols with `lang = "javascript"`.
- `aikits kg query symbol <name>` surfaces JS symbols.
- `aikits kg resolve --lang javascript` exits 0 (no-op, with informational log).
- All existing Go and Java tests continue to pass.
- New unit tests cover: symbol extraction, call-site extraction, import extraction,
  module FQN derivation, and `langForFile` routing for JS extensions.

## Constraints & Assumptions

- **Library**: use `github.com/tree-sitter/tree-sitter-javascript/bindings/go`
  (already present in `go.sum`; must be added as a direct `go.mod` dependency).
- **Pattern**: follow the exact same structure as the Java indexer
  (`internal/kg/indexer/java/`, `internal/kg/lang/lang_java.go`).
- Confidence and provenance for heuristic extraction: `0.5` / `"heuristic"` (same as Java).
- FQN scheme: `<reldir>/<basename_no_ext>.<symbolName>` (e.g. `src/utils/helpers.formatDate`).

## Questions & Open Items

- None — scope and approach are clear.
