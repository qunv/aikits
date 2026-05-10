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
- [x] Function declaration: `function foo() {}` → symbol kind=function, name=foo
- [x] Arrow function assigned to const: `const bar = () => {}` → kind=arrow_function
- [x] Function expression assigned to const: `const baz = function() {}` → kind=function
- [x] Class declaration: `class MyClass {}` → kind=class
- [x] Method definition inside class: `class A { doThing() {} }` → kind=method, fqn=`src/utils/helpers.A.doThing`
- [x] Top-level variable (non-function): `const x = 42` → kind=variable
- [x] Exported function declaration: `export function greet() {}` → kind=function
- [x] Exported class with method: `export class Widget { render() {} }` → class + method symbols
- [x] Call expression: `foo()` → callsite row with callee=foo
- [x] Member call: `obj.method()` → callsite row
- [x] Import statement: `import foo from './foo'` → import path recorded
- [x] Require call: `const x = require('./bar')` → import path recorded
- [x] FQN for nested file: `src/utils/helpers.js` → FQN prefix `src/utils/helpers`
- [x] FQN for root-level file: `index.js` → FQN prefix `index`
- [x] Symbol lang and visibility fields: lang=javascript, visibility=public
- [x] Symbol line numbers: StartLine > 0
- [ ] Syntax error in src → non-nil error returned, empty FileExtract *(not implemented: `ExtractJS` is best-effort and never returns an error; tree-sitter handles malformed input gracefully)*

### `internal/kg/indexer/discovery_test.go` (extended)
- [x] Walker (nil langs) discovers `.js` files → `TestWalkFindsJavaScriptFiles`
- [x] Walker (nil langs) discovers `.mjs`, `.cjs`, `.jsx` files → `TestWalkFindsJavaScriptFiles`
- [x] Walker (nil langs) does NOT discover `.ts` files → `TestWalkFindsJavaScriptFiles`
- [x] Walker with `--lang go` does NOT discover `.js` → `TestWalkJavaScriptLangFilter`
- [x] Walker with `--lang javascript` discovers only `.js` → `TestWalkJavaScriptOnlyFilter`
- [ ] `langForFile("foo.js")` → "javascript" *(private function; covered indirectly via walker tests above)*
- [ ] `langForFile("foo.mjs")` → "javascript" *(covered indirectly)*
- [ ] `langForFile("foo.cjs")` → "javascript" *(covered indirectly)*
- [ ] `langForFile("foo.jsx")` → "javascript" *(covered indirectly)*
- [ ] `langForFile("foo.ts")` → "" *(covered indirectly)*

## Integration Tests

- [x] `NewWalker(root, nil)` discovers `.js` files when root contains them
- [x] `NewWalker(root, []string{"go"})` does NOT discover `.js` files

## Test Data

- Inline source strings passed directly to `ExtractJS` — no fixture files needed.

## Test Reporting & Coverage

Run: `go test ./internal/kg/indexer/javascript/... -v -cover`
