---
phase: testing
title: Testing Strategy
description: Define testing approach, test cases, and quality assurance
---

# Testing Strategy

## Test Coverage Goals

- 100% of new code in `internal/kg/indexer/css/`
- Discovery routing for `.css` covered in `discovery_test.go`
- All existing tests must remain green

## Unit Tests

### `css/extractor_test.go`

- [ ] Class selector `.foo` is extracted as symbol (kind=`class`, FQN=`<module>.foo`)
- [ ] Multiple class selectors in one rule set are all extracted
- [ ] ID selector `#hero` is extracted as symbol (kind=`id`, FQN=`<module>#hero`)
- [ ] `@keyframes fade-in` is extracted as symbol (kind=`keyframes`, FQN=`<module>@fade-in`)
- [ ] `@import "tokens.css"` produces an import path `"tokens.css"`
- [ ] `@import url("base.css")` produces an import path `"base.css"`
- [ ] CSS custom property `--primary-color` at `:root` extracted (kind=`variable`)
- [ ] Tag selectors (`div`, `h1`) are NOT extracted
- [ ] Empty file produces zero symbols and zero imports
- [ ] Malformed/invalid CSS does not panic (error nodes skipped)

### `discovery_test.go` additions

- [ ] `langForFile("styles.css")` returns `"css"`
- [ ] Walker default set includes `"css"`

## Test Data

- Inline CSS strings embedded in table-driven test cases (no fixture files needed).
- Example CSS snippets cover all extraction cases listed above.

## Test Reporting & Coverage

Run with: `go test ./internal/kg/indexer/css/... -v -count=1`
Full suite: `go test ./internal/...`
