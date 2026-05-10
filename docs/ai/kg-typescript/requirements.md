---
phase: requirements
title: Requirements & Problem Understanding
description: Clarify the problem space, gather requirements, and define success criteria
---

# Requirements & Problem Understanding

## Problem Statement

The `aikits kg` knowledge graph currently supports Go, Java, JavaScript, HTML, and CSS. TypeScript
is one of the most widely used languages today, appearing in virtually all modern frontend projects
(React, Angular, Vue, Next.js) and many Node.js/backend codebases. Developers working in those
repos cannot use `aikits kg` to navigate or analyse their TypeScript code.

**Who is affected?** Any team using aikits on repos that contain `.ts` or `.tsx` files.

**Current workaround:** None — TypeScript files are silently skipped by the walker.

## Goals & Objectives

### Primary goals
- Parse TypeScript source files (`.ts`, `.tsx`) with tree-sitter using
  `github.com/tree-sitter/tree-sitter-typescript` and extract symbols (functions, classes,
  interfaces, type aliases, enums, variables, arrow-functions, methods) and call-site edges
  into the kg database.
- Register `typescript` as a first-class `Lang` constant.
- Honour `--lang typescript` (and the short alias `ts`) in all `aikits kg` sub-commands
  (`index`, `resolve`, `export`, `query`).
- Include `.ts` and `.tsx` in the automatic file walker when no `--lang` filter is specified.

### Secondary goals
- No-op resolver: TypeScript does not have an LSP-based semantic upgrade pass in this release;
  `aikits kg resolve --lang typescript` should succeed without crashing.
- TypeScript-specific symbols: extract `interface`, `type_alias`, `enum`, `enum_member` in
  addition to the JS base-set, since these are first-class TypeScript constructs.

### Non-goals
- Full TypeScript type-checking or type inference.
- Module-resolution across npm packages (same limitation as the JS indexer).
- Test files (`*.test.ts`, `*.spec.ts`) are indexed but not given special treatment.
- `.d.ts` declaration files — out of scope for this release.

## User Stories & Use Cases

- **As a developer**, I want to run `aikits kg index --lang typescript` on my TypeScript project
  so that I can navigate symbols and call graphs with `aikits kg query`.
- **As a developer**, I want `aikits kg index` (no `--lang`) in a full-stack repo to index Go,
  Java, JavaScript, and TypeScript files in a single pass.
- **As a developer**, I want `aikits kg query symbol MyInterface` to return TypeScript interface
  symbols extracted from `.ts` files.
- **As a developer**, I want `.tsx` React components to be indexed so I can trace component
  hierarchies.

## Success Criteria

- `aikits kg index --lang typescript` completes without errors on a TS-heavy repo and
  populates symbols with `lang = "typescript"`.
- `aikits kg query symbol <name>` surfaces TS symbols including interfaces and enums.
- `aikits kg resolve --lang typescript` exits 0 (no-op, with informational log).
- All existing Go, Java, JavaScript, HTML, and CSS tests continue to pass.
- New unit tests cover: symbol extraction (functions, classes, interfaces, type aliases, enums,
  methods, arrow-functions), call-site extraction, import extraction, FQN derivation, and
  `langForFile` routing for `.ts`/`.tsx` extensions.

## Constraints & Assumptions

- **Library**: use `github.com/tree-sitter/tree-sitter-typescript/bindings/go` from
  `github.com/tree-sitter/tree-sitter-typescript` (not yet in `go.mod`; must be added).
- **Pattern**: follow the exact same structure as the JavaScript indexer
  (`internal/kg/indexer/javascript/`, `internal/kg/lang/lang_javascript.go`).
- `tree-sitter-typescript` provides two grammars: `typescript` (for `.ts`) and `tsx` (for `.tsx`).
  Both are in the same module. The extractor must select the correct grammar based on file extension.
- Confidence and provenance for heuristic extraction: `0.5` / `"heuristic"` (same as JS/Java).
- FQN scheme: `<reldir>/<basename_no_ext>.<symbolName>` (same as JS).
- Short alias `ts` must be normalised to `typescript` in the CLI layer (same pattern as `js` → `javascript`).

## Questions & Open Items

- None — scope and approach are clear based on the established JS indexer pattern.
