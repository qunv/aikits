---
phase: testing
title: Testing Strategy
description: Define testing approach, test cases, and quality assurance
---

# Testing Strategy

## Test Coverage Goals

- 100% of new production code in `internal/kg/indexer/javascript/` covered by unit tests.
- Existing tests for Go and Java indexers must continue to pass.
- Discovery routing for JS extensions covered in `indexer_test.go`.

## Unit Tests

### `internal/kg/indexer/javascript/extractor_test.go`
- [ ] Function declaration: `function foo() {}` → symbol kind=function, name=foo
- [ ] Arrow function assigned to const: `const bar = () => {}` → kind=arrow_function
- [ ] Class declaration: `class MyClass {}` → kind=class
- [ ] Method definition inside class: `class A { doThing() {} }` → kind=method, fqn contains A.doThing
- [ ] Top-level variable (non-function): `const x = 42` → kind=variable
- [ ] Call expression: `foo()` → callsite row with callee=foo
- [ ] Member call: `obj.method()` → callsite row
- [ ] Import statement: `import foo from './foo'` → import path recorded
- [ ] Require call: `const x = require('./bar')` → import path recorded
- [ ] Syntax error in src → non-nil error returned, empty FileExtract

### `internal/kg/indexer/discovery_test.go` (extend existing)
- [ ] `langForFile("foo.js")` → "javascript"
- [ ] `langForFile("foo.mjs")` → "javascript"
- [ ] `langForFile("foo.cjs")` → "javascript"
- [ ] `langForFile("foo.jsx")` → "javascript"
- [ ] `langForFile("foo.ts")` → "" (not yet supported)

## Integration Tests

- [ ] `NewWalker(root, nil)` discovers `.js` files when root contains them
- [ ] `NewWalker(root, []string{"go"})` does NOT discover `.js` files

## Test Data

- Inline source strings passed directly to `ExtractJS` — no fixture files needed.

## Test Reporting & Coverage

Run: `go test ./internal/kg/indexer/javascript/... -v -cover`
