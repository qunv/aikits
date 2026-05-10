package html

import (
	"testing"
)

func extract(t *testing.T, src string) HTMLExtractResult {
	t.Helper()
	return ExtractHTML([]byte(src), "pages/index.html", 1, 2)
}

func findSym(r HTMLExtractResult, name, kind string) bool {
	for _, s := range r.Symbols {
		if s.Name == name && s.Kind == kind {
			return true
		}
	}
	return false
}

func findImport(r HTMLExtractResult, path string) bool {
	for _, p := range r.ImportPaths {
		if p == path {
			return true
		}
	}
	return false
}

func findCallsite(r HTMLExtractResult, callee string) bool {
	for _, c := range r.Callsites {
		if c.CalleeText == callee {
			return true
		}
	}
	return false
}

// ─── Element-ID symbol extraction ─────────────────────────────────────────────

func TestElementIDSymbol(t *testing.T) {
	r := extract(t, `<html><body><div id="hero">Hello</div></body></html>`)
	if !findSym(r, "hero", "id") {
		t.Errorf("expected id symbol 'hero', got %+v", r.Symbols)
	}
}

func TestElementIDFQN(t *testing.T) {
	r := extract(t, `<html><body><section id="about"></section></body></html>`)
	found := false
	for _, s := range r.Symbols {
		if s.Name == "about" && s.FQN == "pages/index#about" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected FQN 'pages/index#about', got %+v", r.Symbols)
	}
}

func TestMultipleElementIDs(t *testing.T) {
	r := extract(t, `<html><body><div id="hero"></div><div id="footer"></div></body></html>`)
	if !findSym(r, "hero", "id") {
		t.Errorf("expected symbol 'hero'")
	}
	if !findSym(r, "footer", "id") {
		t.Errorf("expected symbol 'footer'")
	}
}

// ─── Import edges ──────────────────────────────────────────────────────────────

func TestScriptSrcImport(t *testing.T) {
	r := extract(t, `<html><head><script src="app.js"></script></head></html>`)
	if !findImport(r, "app.js") {
		t.Errorf("expected import 'app.js', got %+v", r.ImportPaths)
	}
}

func TestLinkHrefImport(t *testing.T) {
	r := extract(t, `<html><head><link rel="stylesheet" href="style.css"></head></html>`)
	if !findImport(r, "style.css") {
		t.Errorf("expected import 'style.css', got %+v", r.ImportPaths)
	}
}

func TestMultipleImports(t *testing.T) {
	r := extract(t, `<html><head>
		<link href="main.css">
		<script src="vendor.js"></script>
		<script src="app.js"></script>
	</head></html>`)
	if !findImport(r, "main.css") {
		t.Errorf("expected import 'main.css'")
	}
	if !findImport(r, "vendor.js") {
		t.Errorf("expected import 'vendor.js'")
	}
	if !findImport(r, "app.js") {
		t.Errorf("expected import 'app.js'")
	}
}

func TestDeduplicateImports(t *testing.T) {
	r := extract(t, `<html><head>
		<script src="app.js"></script>
		<script src="app.js"></script>
	</head></html>`)
	count := 0
	for _, p := range r.ImportPaths {
		if p == "app.js" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'app.js' to appear once, got %d times", count)
	}
}

// ─── Custom-element call-sites ─────────────────────────────────────────────────

func TestCustomElementCallsite(t *testing.T) {
	r := extract(t, `<html><body><my-button>Click</my-button></body></html>`)
	if !findCallsite(r, "my-button") {
		t.Errorf("expected callsite 'my-button', got %+v", r.Callsites)
	}
}

func TestMultipleCustomElements(t *testing.T) {
	r := extract(t, `<html><body><app-header></app-header><app-footer></app-footer></body></html>`)
	if !findCallsite(r, "app-header") {
		t.Errorf("expected callsite 'app-header'")
	}
	if !findCallsite(r, "app-footer") {
		t.Errorf("expected callsite 'app-footer'")
	}
}

func TestStandardTagNotCallsite(t *testing.T) {
	r := extract(t, `<html><body><div><span><p>text</p></span></div></body></html>`)
	for _, c := range r.Callsites {
		if c.CalleeText == "div" || c.CalleeText == "span" || c.CalleeText == "p" {
			t.Errorf("standard tag %q should not be a callsite", c.CalleeText)
		}
	}
}

// ─── Inline script delegation ──────────────────────────────────────────────────

func TestInlineScriptDelegation(t *testing.T) {
	r := extract(t, `<html><body><script>function greet() { console.log("hi"); }</script></body></html>`)
	if !findSym(r, "greet", "function") {
		t.Errorf("expected JS function 'greet' from inline script, got %+v", r.Symbols)
	}
}

func TestInlineScriptCallsiteDelegation(t *testing.T) {
	r := extract(t, `<html><body><script>greet();</script></body></html>`)
	if !findCallsite(r, "greet") {
		t.Errorf("expected callsite 'greet' from inline script, got %+v", r.Callsites)
	}
}

// ─── SrcPkgFQN ────────────────────────────────────────────────────────────────

func TestSrcPkgFQN(t *testing.T) {
	r := extract(t, `<html></html>`)
	if r.SrcPkgFQN != "pages/index" {
		t.Errorf("expected SrcPkgFQN 'pages/index', got %q", r.SrcPkgFQN)
	}
}

func TestSrcPkgFQNRootFile(t *testing.T) {
	r := ExtractHTML([]byte(`<html></html>`), "index.html", 1, 2)
	if r.SrcPkgFQN != "index" {
		t.Errorf("expected SrcPkgFQN 'index', got %q", r.SrcPkgFQN)
	}
}
