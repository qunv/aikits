/**
 * Stub the Wails `window.go` bridge for development when running outside
 * the Wails desktop shell (e.g. `npm run dev` in a plain browser).
 *
 * All methods return empty/no-op responses so the UI renders without crashing.
 * This file is imported only when `window.go` is not already set by the Wails runtime.
 */

const win = window as unknown as Record<string, unknown>;

if (typeof window !== 'undefined' && !win['go']) {
  const noop = () => Promise.resolve(null);

  const App = {
    AgentList: () => Promise.resolve([]),
    DocsFeaturePhases: (_feature: string) => Promise.resolve([]),
    DocsListFeatures: () => Promise.resolve([]),
    DocsReadPhase: (_feature: string, _phase: string) => Promise.resolve(''),
    KGGetGraph: (_repo: string, _filter: string) =>
      Promise.resolve({ nodes: [], edges: [] }),
    KGIndex: noop,
    KGInit: noop,
    KGQuerySymbol: (_repo: string, _query: string) => Promise.resolve([]),
    KGStatus: (_repo: string) => Promise.resolve(null),
    MemorySearch: noop,
    MemoryStore: noop,
    MemoryUpdate: noop,
    ScaffoldInit: noop,
    ScaffoldSkillAdd: noop,
    ScaffoldSkillList: () => Promise.resolve([]),
    SelectRepository: () => Promise.resolve(''),
  };

  win['go'] = { main: { App } };

  if (!win['runtime']) {
    win['runtime'] = new Proxy({}, { get: () => () => undefined });
  }
}
