---
phase: requirements
title: Requirements & Problem Understanding
description: Clarify the problem space, gather requirements, and define success criteria
---

# Requirements & Problem Understanding

## Problem Statement

The `aikits kg` knowledge graph currently supports Go, Java, JavaScript, and HTML.
CSS is the styling layer present in virtually every web project — `.css` files define
reusable class names, IDs, keyframe animations, custom properties, and cross-file
`@import` chains. Developers working on those repos cannot use `aikits kg` to navigate
or analyse their CSS files.

**Who is affected?** Any team using aikits on repos that contain `.css` files (design
systems, component libraries, static sites, full-stack apps).

**Current workaround:** None — `.css` files are silently skipped by the walker.

## Goals & Objectives

### Primary goals
- Parse CSS source files with `tree-sitter-css` (`github.com/tree-sitter/tree-sitter-css`)
  and extract:
  - **Class selectors** (`.foo`) as symbols (kind: `class`, FQN: `<module>.foo`).
  - **ID selectors** (`#bar`) as symbols (kind: `id`, FQN: `<module>#bar`).
  - **@keyframes** names as symbols (kind: `keyframes`, FQN: `<module>@<name>`).
  - **@import** paths as import edges.
  - **CSS custom properties** (`--var-name`) defined at `:root` or top-level as symbols
    (kind: `variable`, FQN: `<module>--<name>`).
- Register `css` as a first-class `Lang` constant alongside existing languages.
- Honour `--lang css` in all `aikits kg` sub-commands (`index`, `query`, `export`).
- Include `.css` in the automatic file walker when no `--lang` filter is specified.

### Secondary goals
- No-op resolver: `aikits kg resolve --lang css` should succeed without crashing
  (CSS has no LSP-based semantic upgrade in this release).

### Non-goals
- SCSS/LESS/Sass — out of scope; separate feature.
- CSS-in-JS (styled-components, emotion) — out of scope.
- Cross-file specificity analysis — out of scope.
- Tag name selectors (e.g. `div`, `h1`) as symbols — too noisy, out of scope.
- Call-site extraction — CSS does not call functions in the graph sense; out of scope.

## User Stories & Use Cases

- **As a developer**, I want to run `aikits kg index --lang css` on my design-system repo
  so that I can navigate class names and custom properties with `aikits kg query`.
- **As a developer**, I want `aikits kg index` (no `--lang`) in a full-stack repo to
  index Go, Java, JavaScript, HTML, and CSS files in a single pass.
- **As a developer**, I want `aikits kg query symbol button` to return results for
  `.button` class selectors defined in `.css` files.
- **As a developer**, I want `aikits kg query symbol --fade-in` to surface `@keyframes`
  named `fade-in` in my animation library.

## Success Criteria

- `aikits kg index --lang css` completes without errors on a CSS-heavy repo and
  populates symbols with `lang = "css"`.
- `aikits kg query symbol <class>` surfaces CSS class-selector symbols.
- `aikits kg resolve --lang css` exits 0 (no-op, with informational log).
- All existing Go, Java, JavaScript, and HTML tests continue to pass.
- New unit tests cover: class-selector extraction, ID-selector extraction,
  `@keyframes` name extraction, `@import` path extraction, CSS custom-property
  extraction, `langForFile` routing for `.css`, and walker inclusion.

## Constraints & Assumptions

- **Library**: `github.com/tree-sitter/tree-sitter-css/bindings/go` — must be added
  as a direct `go.mod` dependency.
- **Pattern**: mirror the HTML indexer structure (`internal/kg/indexer/css/`,
  with `parser.go` and `extractor.go`).
- Confidence and provenance for heuristic extraction: `0.5` / `"heuristic"` (same as JS/HTML).
- FQN scheme: `<reldir>/<basename_no_ext>.<classname>` for classes,
  `<reldir>/<basename_no_ext>#<id>` for IDs,
  `<reldir>/<basename_no_ext>@<keyframes-name>` for keyframes,
  `<reldir>/<basename_no_ext>--<prop>` for custom properties.
- Visibility: all `"public"` (CSS has no access modifiers).

## Questions & Open Items

- None — scope and approach are clear.
