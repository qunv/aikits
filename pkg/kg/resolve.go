package kg

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kgdb "aikits/internal/kg/db"
	kglang "aikits/internal/kg/lang"
)

// Resolve performs semantic callsite resolution for the given language using
// the configured language server (gopls for Go, jdtls for Java).
// Returns *ErrToolNotFound if the required tool is not available.
func (kg *KG) Resolve(_ context.Context, opts ResolveOptions) error {
	lang := strings.ToLower(strings.TrimSpace(string(opts.Lang)))
	if lang == "" {
		lang = "go"
	}

	budget := opts.Budget
	if budget <= 0 {
		budget = 1000
	}

	resolvers := map[string]kglang.Resolver{
		"go":         &kglang.GoResolver{},
		"java":       &kglang.JavaResolver{MavenDownloadDeps: opts.MavenDownloadDeps},
		"javascript": &kglang.JavaScriptResolver{},
		"typescript": &kglang.TypeScriptResolver{},
		"html":       &kglang.HTMLResolver{},
		"css":        &kglang.CSSResolver{},
	}
	r, ok := resolvers[lang]
	if !ok {
		return fmt.Errorf("unsupported language %q; use %q, %q, %q, %q, %q, or %q", lang, LangGo, LangJava, LangJavaScript, LangTypeScript, LangHTML, LangCSS)
	}

	err := r.Resolve(kg.db, kg.repo, kg.root, budget, kg.log)
	if err != nil {
		// Translate internal ErrToolNotFound to the public type.
		var toolErr kglang.ErrToolNotFound
		if errors.As(err, &toolErr) {
			return &ErrToolNotFound{Tool: string(toolErr)}
		}
		return err
	}
	return nil
}

// DeleteFile removes a file and all its symbols/callsites from the index.
func (kg *KG) DeleteFile(fileID int64) error {
	return kgdb.DeleteFile(kg.db, fileID)
}
