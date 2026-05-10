package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"aikits/internal/memory/handlers"
	"aikits/internal/memory/types"
	pkgkg "aikits/pkg/kg"
	pkgscaffold "aikits/pkg/scaffold"
)


// App struct
type App struct {
	ctx context.Context
	log *zap.Logger
}

// NewApp creates a new App application struct
func NewApp() *App {
	log, _ := zap.NewProduction()
	return &App{log: log}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ── Repository ───────────────────────────────────────────────────────────────

// SelectRepository opens a native folder picker and returns the chosen path.
func (a *App) SelectRepository() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Repository",
	})
}


// ── Memory ───────────────────────────────────────────────────────────────────

// MemoryStore stores a new knowledge item and returns the result.
func (a *App) MemoryStore(input types.StoreInput) (*types.StoreResult, error) {
	return handlers.Store(&input)
}

// MemoryUpdate updates an existing knowledge item by ID.
func (a *App) MemoryUpdate(input types.UpdateInput) (*types.UpdateResult, error) {
	return handlers.Update(&input)
}

// MemorySearch searches the knowledge store and returns ranked results.
func (a *App) MemorySearch(input types.SearchInput) (*types.SearchResult, error) {
	return handlers.Search(&input)
}

// ── Knowledge Graph ──────────────────────────────────────────────────────────

// KGInit initialises the knowledge graph for the given repo root directory.
func (a *App) KGInit(repoRoot string, reinit bool) error {
	kg, err := pkgkg.Init(a.ctx, repoRoot, reinit, a.log)
	if err != nil {
		return err
	}
	return kg.Close()
}

// KGIndex indexes a repo directory into the knowledge graph.
func (a *App) KGIndex(repoRoot string, opts pkgkg.IndexOptions) (*pkgkg.IndexResult, error) {
	kg, err := pkgkg.Open(repoRoot, a.log)
	if err != nil {
		return nil, err
	}
	defer kg.Close() //nolint:errcheck
	return kg.Index(a.ctx, opts)
}

// KGStatus returns the current status of the knowledge graph for a repo directory.
func (a *App) KGStatus(repoRoot string) (*pkgkg.StatusResult, error) {
	kg, err := pkgkg.Open(repoRoot, a.log)
	if err != nil {
		return nil, err
	}
	defer kg.Close() //nolint:errcheck
	return kg.Status(a.ctx)
}

// KGGetGraph returns all nodes and edges in the knowledge graph for repoRoot.
// lang filters by language ("go", "java"); empty string returns all.
func (a *App) KGGetGraph(repoRoot, lang string) (*pkgkg.GraphData, error) {
	kg, err := pkgkg.Open(repoRoot, a.log)
	if err != nil {
		return nil, err
	}
	defer kg.Close() //nolint:errcheck
	return kg.GetGraph(a.ctx, lang)
}

// KGQuerySymbol looks up a symbol by name or fully-qualified name.
func (a *App) KGQuerySymbol(repoRoot, nameOrFQN string) ([]pkgkg.Symbol, error) {
	kg, err := pkgkg.Open(repoRoot, a.log)
	if err != nil {
		return nil, err
	}
	defer kg.Close() //nolint:errcheck
	return kg.QuerySymbol(a.ctx, nameOrFQN)
}

// ── Scaffold ─────────────────────────────────────────────────────────────────

// ScaffoldInit initialises the AI workflow scaffold in the target directory.
func (a *App) ScaffoldInit(opts pkgscaffold.InitOptions) (*pkgscaffold.InitResult, error) {
	return pkgscaffold.Init(a.ctx, opts)
}

// ScaffoldSkillList returns the available built-in skill names.
func (a *App) ScaffoldSkillList() ([]string, error) {
	return pkgscaffold.SkillList(a.ctx)
}

// ScaffoldSkillAdd installs a skill for the given agents in the current git repo.
func (a *App) ScaffoldSkillAdd(opts pkgscaffold.SkillAddOptions) ([]pkgscaffold.SkillAddResult, error) {
	return pkgscaffold.SkillAdd(a.ctx, opts)
}

// AgentList returns all supported AI agents.
func (a *App) AgentList() []pkgscaffold.AgentInfo {
	return pkgscaffold.AgentList()
}

// ── Docs ─────────────────────────────────────────────────────────────────────

// docsAIDir returns the absolute path to the docs/ai directory, resolved
// relative to the running executable so it works both in dev (wails dev) and
// in a compiled binary shipped alongside the repo.
func docsAIDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// During `wails dev` the executable is a temp binary; CWD is the repo root.
	// For a compiled binary the exe lives next to the repo root as well.
	// Try exe-relative first, then fall back to CWD.
	candidates := []string{
		filepath.Join(filepath.Dir(exe), "docs", "ai"),
		filepath.Join("docs", "ai"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	// Return the CWD-relative path; callers will surface the OS error.
	return filepath.Join("docs", "ai"), nil
}

// DocsListFeatures returns the list of feature names found under docs/ai/.
func (a *App) DocsListFeatures() ([]string, error) {
	base, err := docsAIDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	var features []string
	for _, e := range entries {
		if e.IsDir() {
			features = append(features, e.Name())
		}
	}
	return features, nil
}

// DocsFeaturePhases returns the list of phase names (without .md) for a feature.
func (a *App) DocsFeaturePhases(feature string) ([]string, error) {
	base, err := docsAIDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(base, feature))
	if err != nil {
		return nil, err
	}
	var phases []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			phases = append(phases, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return phases, nil
}

// DocsReadPhase returns the markdown content of docs/ai/{feature}/{phase}.md.
func (a *App) DocsReadPhase(feature, phase string) (string, error) {
	base, err := docsAIDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(base, feature, phase+".md"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
