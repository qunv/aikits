---
phase: testing
title: Testing Strategy
description: Define testing approach, test cases, and quality assurance
---

# Testing Strategy

## Approach

Tests are written TDD-first (failing → passing). All extraction logic is tested through the public `ExtractHTML` entry point. Integration with the `lang.Indexer` pipeline is covered by `discovery_test.go` walker tests.

## Test Files

| File | Package | Tests |
|------|---------|-------|
| `internal/kg/indexer/html/extractor_test.go` | `html` | 16 unit tests |
| `internal/kg/indexer/discovery_test.go` | `indexer` | 2 walker integration tests |

## Coverage

```
aikits/internal/kg/indexer/html    97.1% of statements
  extractor.go:newHTMLWalker       100.0%
  extractor.go:text                100.0%
  extractor.go:addImport           100.0%
  extractor.go:walkNode             83.3%  (*)
  extractor.go:walkChildren        100.0%
  extractor.go:visitScriptElement   95.0%  (*)
  extractor.go:visitElement         87.5%  (*)
  extractor.go:visitStartTag       100.0%
  extractor.go:attrValue           100.0%
  extractor.go:childByKind         100.0%
  extractor.go:ExtractHTML         100.0%

(*) Remaining 2.9%: defensive nil-guards for tree-sitter-internal nil child nodes
    and IsError/IsMissing node states. These are never produced by valid or
    reasonably malformed HTML input and cannot be exercised without mocking
    tree-sitter internals.
```

## Test Cases

### Element-ID symbol extraction
| Test | Description |
|------|-------------|
| `TestElementIDSymbol` | `<div id="hero">` → symbol `hero` of kind `id` |
| `TestElementIDFQN` | FQN follows `reldir/basename#idValue` scheme |
| `TestMultipleElementIDs` | Multiple `id` attributes in one file |

### Import edges
| Test | Description |
|------|-------------|
| `TestScriptSrcImport` | `<script src="app.js">` → import `app.js` |
| `TestLinkHrefImport` | `<link href="style.css">` → import `style.css` |
| `TestMultipleImports` | Multiple script/link imports in one file |
| `TestDeduplicateImports` | Duplicate imports collapsed to one entry |

### Custom-element callsites
| Test | Description |
|------|-------------|
| `TestCustomElementCallsite` | `<my-button>` → callsite with `CalleeText="my-button"` |
| `TestMultipleCustomElements` | Multiple custom elements in one file |
| `TestStandardTagNotCallsite` | `<div>`, `<span>`, `<p>` are NOT callsites |

### Inline script delegation
| Test | Description |
|------|-------------|
| `TestInlineScriptDelegation` | JS function in `<script>` body → symbol in result |
| `TestInlineScriptCallsiteDelegation` | JS call in `<script>` body → callsite in result |
| `TestInlineScriptImportDelegation` | JS `import` in `<script>` body → import path in result |

### Attribute parsing
| Test | Description |
|------|-------------|
| `TestUnquotedAttributeValue` | `<div id=hero>` (unquoted) → symbol `hero` |

### SrcPkgFQN
| Test | Description |
|------|-------------|
| `TestSrcPkgFQN` | `pages/index.html` → `SrcPkgFQN = "pages/index"` |
| `TestSrcPkgFQNRootFile` | Root `index.html` → `SrcPkgFQN = "index"` |

### Walker integration (discovery_test.go)
| Test | Description |
|------|-------------|
| `TestWalkFindsHTMLFiles` | `.html` and `.htm` files discovered by default lang set |
| `TestWalkHTMLLangFilter` | `--lang go` filter excludes `.html` files |

## Gaps and Rationale

The only uncovered statements are nil-guard branches that depend on tree-sitter-internal node states (`IsError`, `IsMissing`, nil children). These guards exist for defensive correctness but are not reachable through the public `ExtractHTML` API without mocking tree-sitter internals. They are accepted as uncoverable at 97.1% overall.
