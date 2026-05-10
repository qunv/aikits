import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import ForceGraph2D from 'react-force-graph-2d';
import { KGGetGraph } from '@wailsjs/go/main/App';
import { useRepoStore } from '@stores/StoreContext';
import { observer } from 'mobx-react-lite';
import { Alert, Button, Spin } from 'antd';
import { ThunderboltOutlined, ThunderboltFilled, ReloadOutlined, RadarChartOutlined } from '@ant-design/icons';


interface KGNode {
  id: number;
  lang: string;
  kind: string;
  name: string;
  fqn: string;
  signature: string;
  visibility: string;
  startLine: number;
}

interface KGEdge {
  id: number;
  kind: string;
  src: number;
  dst: number;
  confidence: number;
  provenance: string;
}

// Color palette per node kind
const KIND_COLORS: Record<string, string> = {
  function:  '#4096ff',
  method:    '#1677ff',
  class:     '#52c41a',
  interface: '#13c2c2',
  struct:    '#722ed1',
  variable:  '#fa8c16',
  constant:  '#eb2f96',
  type:      '#faad14',
};

function nodeColor(node: KGNode): string {
  return KIND_COLORS[node.kind] ?? '#8c8c8c';
}

const EDGE_COLORS: Record<string, string> = {
  CALLS:      '#ff4d4f',
  REFERENCES: '#1677ff',
  IMPLEMENTS: '#52c41a',
  OVERRIDES:  '#722ed1',
};

function edgeColor(kind: string): string {
  return EDGE_COLORS[kind] ?? '#bfbfbf';
}

type Layout = 'force' | 'concentric';

/** Place nodes in concentric rings ordered by degree (hubs at center).
 *  Each ring radius is computed from how many nodes it holds so nodes
 *  never overlap, regardless of graph size.
 */
function computeConcentricPositions(
  nodes: KGNode[],
  edges: KGEdge[],
): Map<number, { fx: number; fy: number }> {
  const degree = new Map<number, number>(nodes.map((n) => [n.id, 0]));
  edges.forEach((e) => {
    degree.set(e.src, (degree.get(e.src) ?? 0) + 1);
    degree.set(e.dst, (degree.get(e.dst) ?? 0) + 1);
  });

  const sorted = [...nodes].sort((a, b) => (degree.get(b.id) ?? 0) - (degree.get(a.id) ?? 0));

  // Adapt node spacing to graph size so large graphs stay navigable
  const total = nodes.length;
  const minArc  = total < 100 ? 28 : total < 500 ? 18 : 12; // px between node centres
  const minGap  = 70;  // minimum radial gap between adjacent rings
  const innerR  = 60;  // minimum radius for the innermost ring

  // Cumulative fraction of nodes per ring (inner → outer)
  const ringFracs = [0.05, 0.15, 0.35, 1.0];

  const positions = new Map<number, { fx: number; fy: number }>();
  let start      = 0;
  let prevRadius = 0;

  ringFracs.forEach((frac, i) => {
    const end  = Math.min(Math.ceil(frac * sorted.length), sorted.length);
    const ring = sorted.slice(start, end);

    // Radius must be large enough so the arc between adjacent nodes ≥ minArc
    const minBySpacing = ring.length <= 1
      ? innerR
      : (ring.length * minArc) / (2 * Math.PI);

    const r = Math.max(
      i === 0 ? innerR : prevRadius + minGap,
      minBySpacing,
    );

    ring.forEach((node, j) => {
      const angle = (2 * Math.PI * j) / Math.max(ring.length, 1);
      positions.set(node.id, { fx: r * Math.cos(angle), fy: r * Math.sin(angle) });
    });

    prevRadius = r;
    start      = end;
  });

  return positions;
}

const KnowledgeGraphPage = observer(() => {
  const repoStore = useRepoStore();
  const containerRef = useRef<HTMLDivElement>(null);
  const fgRef = useRef<any>(null);
  const [graphData, setGraphData] = useState<{ nodes: KGNode[]; edges: KGEdge[] } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<{ type: 'node'; data: KGNode } | { type: 'edge'; data: KGEdge } | null>(null);
  const selectedRef = useRef(selected);
  useEffect(() => { selectedRef.current = selected; }, [selected]);
  const [animate, setAnimate] = useState(false);
  const [layout, setLayout] = useState<Layout>('force');
  const [dimensions, setDimensions] = useState({ width: 800, height: 600 });
  const [hlNodes, setHlNodes] = useState<Set<number>>(new Set());
  const [hlLinks, setHlLinks] = useState<Set<number>>(new Set());

  const clearHighlight = useCallback(() => {
    setSelected(null);
    setHlNodes(new Set());
    setHlLinks(new Set());
  }, []);

  const handleNodeClick = useCallback((node: any) => {
    const n = node as KGNode;
    const nodes = new Set<number>([n.id]);
    const links = new Set<number>();
    graphData?.edges.forEach((e) => {
      if (e.src === n.id || e.dst === n.id) {
        links.add(e.id);
        nodes.add(e.src);
        nodes.add(e.dst);
      }
    });
    setHlNodes(nodes);
    setHlLinks(links);
    setSelected({ type: 'node', data: n });
  }, [graphData]);

  const handleLinkClick = useCallback((link: any) => {
    const e = link as KGEdge;
    setHlNodes(new Set([e.src, e.dst]));
    setHlLinks(new Set([e.id]));
    setSelected({ type: 'edge', data: e });
  }, []);

  const load = useCallback(async () => {
    if (!repoStore.repoPath) return;
    setLoading(true);
    setError(null);
    try {
      const data = await KGGetGraph(repoStore.repoPath, '');
      setGraphData({ nodes: data.nodes ?? [], edges: data.edges ?? [] });
    } catch (e: any) {
      setError(e?.message ?? String(e));
    } finally {
      setLoading(false);
    }
  }, [repoStore.repoPath]);

  useEffect(() => { load(); }, [load]);

  // Track container size
  useEffect(() => {
    if (!containerRef.current) return;
    const obs = new ResizeObserver((entries) => {
      const rect = entries[0].contentRect;
      setDimensions({ width: rect.width, height: rect.height });
    });
    obs.observe(containerRef.current);
    return () => obs.disconnect();
  }, []);

  const fgData = useMemo(() => {
    if (!graphData) return { nodes: [], links: [] };
    const positions = layout === 'concentric'
      ? computeConcentricPositions(graphData.nodes, graphData.edges)
      : null;
    return {
      nodes: graphData.nodes.map((n) => {
        const pos = positions?.get(n.id);
        return { ...n, id: n.id, ...(pos ?? { fx: undefined, fy: undefined }) };
      }),
      links: graphData.edges.map((e) => ({ ...e, source: e.src, target: e.dst })),
    };
  }, [graphData, layout]);

  // Reheat force simulation when switching back from concentric
  useEffect(() => {
    if (layout === 'force' && fgRef.current) {
      fgRef.current.d3ReheatSimulation();
    }
  }, [layout]);

  return (
    <div className="flex flex-col -m-4" style={{ height: 'calc(100% + 2rem)' }}>
      {(!repoStore.repoPath || error) && (
        <div className="px-4 pt-4 flex flex-col gap-2">
          {!repoStore.repoPath && (
            <Alert type="warning" message="Please open a repository first." showIcon />
          )}
          {error && (
            <Alert type="error" message={error} showIcon closable onClose={() => setError(null)} />
          )}
        </div>
      )}

      {/* Graph canvas */}
      <div ref={containerRef} className="flex-1 overflow-hidden bg-white relative">
        {/* Initial load: full canvas skeleton */}
        {loading && !graphData && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-4 z-10 bg-white">
            <Spin size="large" />
            <span className="text-sm text-gray-400 animate-pulse">Querying knowledge graph…</span>
          </div>
        )}

        {/* Reload: dim overlay on top of existing graph */}
        {loading && graphData && (
          <div className="absolute inset-0 flex items-center justify-center z-10 bg-white/60 backdrop-blur-sm">
            <Spin size="large" />
          </div>
        )}

        {!loading && graphData && graphData.nodes.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center text-gray-400">
            No data — run <code className="mx-1">kg index</code> first.
          </div>
        )}
        <ForceGraph2D
          ref={fgRef}
          width={dimensions.width}
          height={dimensions.height}
          graphData={fgData}
          nodeId="id"
          linkSource="source"
          linkTarget="target"
          nodeLabel={(node: any) => `${node.kind}: ${node.fqn || node.name}`}
          nodeColor={(node: any) =>
            hlNodes.size === 0 || hlNodes.has(node.id)
              ? nodeColor(node as KGNode)
              : '#e8e8e8'
          }
          nodeRelSize={5}
          nodeVal={(node: any) => hlNodes.has(node.id) ? 2 : 1}
          nodeCanvasObjectMode={(node: any) =>
            selectedRef.current?.type === 'node' && node.id === selectedRef.current.data.id
              ? 'after'
              : undefined
          }
          nodeCanvasObject={(node: any, ctx: CanvasRenderingContext2D, globalScale: number) => {
            const r = Math.sqrt(2) * 5; // nodeVal=2 * nodeRelSize=5
            const color = nodeColor(node as KGNode);
            ctx.beginPath();
            ctx.arc(node.x, node.y, r + 3 / globalScale, 0, 2 * Math.PI);
            ctx.strokeStyle = color;
            ctx.lineWidth = 2.5 / globalScale;
            ctx.stroke();
            // soft glow
            ctx.beginPath();
            ctx.arc(node.x, node.y, r + 6 / globalScale, 0, 2 * Math.PI);
            ctx.strokeStyle = color + '55';
            ctx.lineWidth = 4 / globalScale;
            ctx.stroke();
          }}
          linkColor={(link: any) => {
            if (hlLinks.size === 0) return edgeColor(link.kind) + '66'; // default: ~40% opacity
            if (hlLinks.has(link.id)) return edgeColor(link.kind) + 'dd'; // selected: ~87% opacity
            return '#e8e8e833'; // dimmed: nearly invisible
          }}
          linkWidth={(link: any) => hlLinks.has(link.id) ? 2 : 1}
          linkDirectionalArrowLength={2}
          linkDirectionalArrowRelPos={1}
          linkLabel={(link: any) => link.kind}
          onNodeClick={handleNodeClick}
          onLinkClick={handleLinkClick}
          onBackgroundClick={clearHighlight}
          backgroundColor="#ffffff"
          enableNodeDrag
          enableZoomInteraction
          warmupTicks={layout === 'concentric' ? 0 : (animate ? 0 : 200)}
          cooldownTicks={layout === 'concentric' ? 0 : (animate ? Infinity : 0)}
        />

        {/* Bottom-left overlay: stats table + reload */}
        <div className="absolute bottom-3 left-3 flex items-end gap-2">
          {graphData && (
            <table className="text-xs bg-white/90 backdrop-blur rounded-lg shadow border border-gray-100 overflow-hidden">
              <tbody>
                <tr className="border-b border-gray-100">
                  <td className="px-3 py-1 text-gray-500 font-medium">Nodes</td>
                  <td className="px-3 py-1 text-right font-semibold text-blue-600">{graphData.nodes.length.toLocaleString()}</td>
                </tr>
                <tr>
                  <td className="px-3 py-1 text-gray-500 font-medium">Edges</td>
                  <td className="px-3 py-1 text-right font-semibold text-red-500">{graphData.edges.length.toLocaleString()}</td>
                </tr>
              </tbody>
            </table>
          )}
          <Button size="small" icon={<ReloadOutlined />} onClick={load} loading={loading} />
          <Button
            size="small"
            icon={animate ? <ThunderboltFilled /> : <ThunderboltOutlined />}
            onClick={() => setAnimate((v) => !v)}
            title={animate ? 'Disable animation' : 'Enable animation'}
            type={animate ? 'primary' : 'default'}
          />
          <Button
            size="small"
            icon={<RadarChartOutlined />}
            onClick={() => setLayout((v) => v === 'concentric' ? 'force' : 'concentric')}
            title={layout === 'concentric' ? 'Switch to force layout' : 'Switch to concentric layout'}
            type={layout === 'concentric' ? 'primary' : 'default'}
          />
        </div>

        {/* Top-right overlay: selected node/edge info */}
        {selected && (
          <div className="absolute top-3 right-3 w-72 bg-white/95 backdrop-blur rounded-xl shadow-lg border border-gray-100 z-20 text-xs overflow-hidden">
            <div className="flex items-center justify-between px-3 py-2 border-b border-gray-100">
              <span className="font-semibold text-gray-700">
                {selected.type === 'node' ? 'Node' : 'Edge'} details
              </span>
              <button
                onClick={clearHighlight}
                className="text-gray-400 hover:text-gray-600 leading-none text-base cursor-pointer"
              >
                ×
              </button>
            </div>
            <table className="w-full">
              <tbody>
                {selected.type === 'node' ? (
                  <>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400 w-24">Name</td>
                      <td className="px-3 py-1.5 font-medium break-all">{selected.data.name}</td>
                    </tr>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400">Kind</td>
                      <td className="px-3 py-1.5">
                        <span className="px-1.5 py-0.5 rounded text-white text-[10px]" style={{ background: KIND_COLORS[selected.data.kind] ?? '#8c8c8c' }}>
                          {selected.data.kind}
                        </span>
                      </td>
                    </tr>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400">Language</td>
                      <td className="px-3 py-1.5">{selected.data.lang}</td>
                    </tr>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400">Visibility</td>
                      <td className="px-3 py-1.5">{selected.data.visibility || '—'}</td>
                    </tr>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400">FQN</td>
                      <td className="px-3 py-1.5 font-mono break-all">{selected.data.fqn || '—'}</td>
                    </tr>
                    {selected.data.signature && (
                      <tr className="border-b border-gray-50">
                        <td className="px-3 py-1.5 text-gray-400">Signature</td>
                        <td className="px-3 py-1.5 font-mono break-all">{selected.data.signature}</td>
                      </tr>
                    )}
                    <tr>
                      <td className="px-3 py-1.5 text-gray-400">Start line</td>
                      <td className="px-3 py-1.5">{selected.data.startLine}</td>
                    </tr>
                  </>
                ) : (
                  <>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400 w-24">Kind</td>
                      <td className="px-3 py-1.5">
                        <span className="px-1.5 py-0.5 rounded text-white text-[10px]" style={{ background: EDGE_COLORS[selected.data.kind] ?? '#8c8c8c' }}>
                          {selected.data.kind}
                        </span>
                      </td>
                    </tr>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400">Source ID</td>
                      <td className="px-3 py-1.5">{selected.data.src}</td>
                    </tr>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400">Target ID</td>
                      <td className="px-3 py-1.5">{selected.data.dst}</td>
                    </tr>
                    <tr className="border-b border-gray-50">
                      <td className="px-3 py-1.5 text-gray-400">Confidence</td>
                      <td className="px-3 py-1.5">{selected.data.confidence.toFixed(2)}</td>
                    </tr>
                    <tr>
                      <td className="px-3 py-1.5 text-gray-400">Provenance</td>
                      <td className="px-3 py-1.5 font-mono break-all">{selected.data.provenance || '—'}</td>
                    </tr>
                  </>
                )}
              </tbody>
            </table>
          </div>
        )}

      </div>
    </div>
  );
});

export default KnowledgeGraphPage;
