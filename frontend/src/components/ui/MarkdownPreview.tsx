import '@uiw/react-markdown-preview/markdown.css';
import { useEffect, useRef } from 'react';
import MarkdownPreviewLib from '@uiw/react-markdown-preview';
import mermaid from 'mermaid';

mermaid.initialize({ startOnLoad: false, theme: 'default', securityLevel: 'loose' });

// Injected once into <head> to restore styles that Tailwind v4 preflight strips.
const STYLE_ID = 'wmde-tailwind-fix';
const OVERRIDE_CSS = `
  .wmde-markdown ul { list-style-type: disc !important; }
  .wmde-markdown ul ul { list-style-type: circle !important; }
  .wmde-markdown ul ul ul { list-style-type: square !important; }
  .wmde-markdown ol { list-style-type: decimal !important; }
  .wmde-markdown h1 { font-size: 2em !important; font-weight: 600 !important; border-bottom: 1px solid #d0d7de; padding-bottom: 0.3em; }
  .wmde-markdown h2 { font-size: 1.5em !important; font-weight: 600 !important; border-bottom: 1px solid #d0d7de; padding-bottom: 0.3em; }
  .wmde-markdown h3 { font-size: 1.25em !important; font-weight: 600 !important; }
  .wmde-markdown h4 { font-size: 1em !important; font-weight: 600 !important; }
  .wmde-markdown h5 { font-size: 0.875em !important; font-weight: 600 !important; }
  .wmde-markdown h6 { font-size: 0.85em !important; font-weight: 600 !important; color: #656d76; }
`;

function injectStyles() {
  if (document.getElementById(STYLE_ID)) return;
  const el = document.createElement('style');
  el.id = STYLE_ID;
  el.textContent = OVERRIDE_CSS;
  document.head.appendChild(el);
}

// ── Frontmatter ───────────────────────────────────────────────────────────────

function parseFrontmatter(raw: string): string {
  const match = raw.match(/^---\r?\n[\s\S]*?\r?\n---\r?\n?/);
  return match ? raw.slice(match[0].length) : raw;
}

// ── Mermaid ───────────────────────────────────────────────────────────────────

// Split markdown into alternating markdown/mermaid segments so mermaid code
// never passes through rehype-prism-plus (which mangles it before render).
type Segment = { type: 'md'; text: string } | { type: 'mermaid'; code: string };

function splitMermaid(src: string): Segment[] {
  const segments: Segment[] = [];
  const re = /^```mermaid\r?\n([\s\S]*?)^```/gm;
  let last = 0;
  let m: RegExpExecArray | null;

  while ((m = re.exec(src)) !== null) {
    if (m.index > last) segments.push({ type: 'md', text: src.slice(last, m.index) });
    segments.push({ type: 'mermaid', code: m[1].trim() });
    last = m.index + m[0].length;
  }

  if (last < src.length) segments.push({ type: 'md', text: src.slice(last) });
  return segments.length ? segments : [{ type: 'md', text: src }];
}

function MermaidBlock({ code }: { code: string }) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.textContent = code;
    el.removeAttribute('data-processed');
    mermaid.run({ nodes: [el], suppressErrors: false }).catch((err: Error) => {
      el.innerHTML = `<pre style="color:red;white-space:pre-wrap">${err.message}</pre>`;
    });
  }, [code]);

  return <div className="mermaid my-4 flex justify-center" ref={ref} />;
}

// ── Public component ──────────────────────────────────────────────────────────

interface MarkdownPreviewProps {
  content: string;
}

export function MarkdownPreview({ content }: MarkdownPreviewProps) {
  useEffect(() => { injectStyles(); }, []);

  const body = parseFrontmatter(content);
  const segments = splitMermaid(body);

  return (
    <div>
      {segments.map((seg, i) =>
        seg.type === 'mermaid' ? (
          <MermaidBlock key={i} code={seg.code} />
        ) : (
          <MarkdownPreviewLib
            key={i}
            source={seg.text}
            style={{ background: 'transparent', fontSize: 14 }}
            wrapperElement={{ 'data-color-mode': 'light' }}
          />
        ),
      )}
    </div>
  );
}
