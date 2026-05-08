package main

import (
	"context"

	"go.uber.org/zap"

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
