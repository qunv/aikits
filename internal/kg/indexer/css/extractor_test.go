package css

import (
	"testing"
)

func extract(t *testing.T, src string) CSSExtractResult {
	t.Helper()
	return ExtractCSS([]byte(src), "styles/main.css", 1, 2)
}

func findSym(r CSSExtractResult, name, kind string) bool {
	for _, s := range r.Symbols {
		if s.Name == name && s.Kind == kind {
			return true
		}
	}
	return false
}

func findImport(r CSSExtractResult, path string) bool {
	for _, p := range r.ImportPaths {
		if p == path {
			return true
		}
	}
	return false
}

// ─── Class selector extraction ────────────────────────────────────────────────

func TestClassSelector(t *testing.T) {
	r := extract(t, `.button { color: red; }`)
	if !findSym(r, "button", "class") {
		t.Errorf("expected class symbol 'button', got %+v", r.Symbols)
	}
}

func TestClassSelectorFQN(t *testing.T) {
	r := extract(t, `.hero { font-size: 2rem; }`)
	found := false
	for _, s := range r.Symbols {
		if s.Name == "hero" && s.FQN == "styles/main.hero" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected FQN 'styles/main.hero', got %+v", r.Symbols)
	}
}

func TestMultipleClassSelectors(t *testing.T) {
	r := extract(t, `.foo { color: red; } .bar { color: blue; }`)
	if !findSym(r, "foo", "class") {
		t.Errorf("expected class symbol 'foo'")
	}
	if !findSym(r, "bar", "class") {
		t.Errorf("expected class symbol 'bar'")
	}
}

func TestMultipleSelectorsInOneRule(t *testing.T) {
	r := extract(t, `.foo, .bar { color: red; }`)
	if !findSym(r, "foo", "class") {
		t.Errorf("expected class symbol 'foo' from compound selector")
	}
	if !findSym(r, "bar", "class") {
		t.Errorf("expected class symbol 'bar' from compound selector")
	}
}

// ─── ID selector extraction ────────────────────────────────────────────────────

func TestIDSelector(t *testing.T) {
	r := extract(t, `#hero { background: #fff; }`)
	if !findSym(r, "hero", "id") {
		t.Errorf("expected id symbol 'hero', got %+v", r.Symbols)
	}
}

func TestIDSelectorFQN(t *testing.T) {
	r := extract(t, `#header { display: flex; }`)
	found := false
	for _, s := range r.Symbols {
		if s.Name == "header" && s.FQN == "styles/main#header" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected FQN 'styles/main#header', got %+v", r.Symbols)
	}
}

// ─── Tag selector NOT extracted ────────────────────────────────────────────────

func TestTagSelectorNotExtracted(t *testing.T) {
	r := extract(t, `div { margin: 0; } h1 { font-size: 2rem; }`)
	for _, s := range r.Symbols {
		if s.Name == "div" || s.Name == "h1" {
			t.Errorf("tag selector %q should not be extracted as symbol", s.Name)
		}
	}
}

// ─── @keyframes extraction ────────────────────────────────────────────────────

func TestKeyframesExtraction(t *testing.T) {
	r := extract(t, `@keyframes fade-in { from { opacity: 0; } to { opacity: 1; } }`)
	if !findSym(r, "fade-in", "keyframes") {
		t.Errorf("expected keyframes symbol 'fade-in', got %+v", r.Symbols)
	}
}

func TestKeyframesFQN(t *testing.T) {
	r := extract(t, `@keyframes slide-up { from { transform: translateY(10px); } }`)
	found := false
	for _, s := range r.Symbols {
		if s.Name == "slide-up" && s.FQN == "styles/main@slide-up" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected FQN 'styles/main@slide-up', got %+v", r.Symbols)
	}
}

// ─── @import extraction ────────────────────────────────────────────────────────

func TestImportStatement(t *testing.T) {
	r := extract(t, `@import "tokens.css";`)
	if !findImport(r, "tokens.css") {
		t.Errorf("expected import 'tokens.css', got %+v", r.ImportPaths)
	}
}

func TestImportStatementSingleQuote(t *testing.T) {
	r := extract(t, `@import 'base.css';`)
	if !findImport(r, "base.css") {
		t.Errorf("expected import 'base.css', got %+v", r.ImportPaths)
	}
}

func TestImportDeduplication(t *testing.T) {
	r := extract(t, `@import "tokens.css"; @import "tokens.css";`)
	count := 0
	for _, p := range r.ImportPaths {
		if p == "tokens.css" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'tokens.css' once, got %d times", count)
	}
}

// ─── CSS custom property extraction ───────────────────────────────────────────

func TestCustomPropertyAtRoot(t *testing.T) {
	r := extract(t, `:root { --primary-color: #333; }`)
	if !findSym(r, "--primary-color", "variable") {
		t.Errorf("expected variable symbol '--primary-color', got %+v", r.Symbols)
	}
}

func TestCustomPropertyFQN(t *testing.T) {
	r := extract(t, `:root { --spacing-md: 8px; }`)
	found := false
	for _, s := range r.Symbols {
		if s.Name == "--spacing-md" && s.FQN == "styles/main--spacing-md" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected FQN 'styles/main--spacing-md', got %+v", r.Symbols)
	}
}

func TestNonCustomPropertyNotExtracted(t *testing.T) {
	r := extract(t, `:root { color: red; font-size: 16px; }`)
	for _, s := range r.Symbols {
		if s.Kind == "variable" {
			t.Errorf("non-custom property should not be extracted as variable, got %+v", s)
		}
	}
}

// ─── Edge cases ────────────────────────────────────────────────────────────────

func TestEmptyFileProducesNoResults(t *testing.T) {
	r := extract(t, ``)
	if len(r.Symbols) != 0 {
		t.Errorf("expected 0 symbols, got %d", len(r.Symbols))
	}
	if len(r.ImportPaths) != 0 {
		t.Errorf("expected 0 imports, got %d", len(r.ImportPaths))
	}
}

func TestCallsitesAlwaysEmpty(t *testing.T) {
	r := extract(t, `.foo { color: red; } @keyframes bar {} @import "x.css";`)
	if len(r.Callsites) != 0 {
		t.Errorf("CSS should produce no callsites, got %d", len(r.Callsites))
	}
}

func TestSrcPkgFQN(t *testing.T) {
	r := extract(t, `.foo {}`)
	if r.SrcPkgFQN != "styles/main" {
		t.Errorf("expected SrcPkgFQN 'styles/main', got %q", r.SrcPkgFQN)
	}
}

func TestSrcPkgFQNRootFile(t *testing.T) {
	r := ExtractCSS([]byte(`.foo {}`), "global.css", 1, 2)
	if r.SrcPkgFQN != "global" {
		t.Errorf("expected SrcPkgFQN 'global', got %q", r.SrcPkgFQN)
	}
}
