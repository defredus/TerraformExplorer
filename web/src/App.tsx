import { lazy, Suspense, useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from 'react'
import {
  AlertTriangle,
  ArrowDownToLine,
  ArrowUpFromLine,
  Braces,
  ChevronRight,
  CircleDot,
  ExternalLink,
  FileCode2,
  FolderOpen,
  Info,
  Maximize2,
  Network,
  RefreshCw,
  Search,
  ShieldCheck,
  TerminalSquare,
  X,
} from 'lucide-react'
import {
  Background,
  BackgroundVariant,
  Controls,
  ReactFlow,
  type Edge,
  type ReactFlowInstance,
} from '@xyflow/react'
import { createScan, readSource } from './api'
import { GraphMiniMap } from './GraphMiniMap'
import { focusGraph, layoutGraph, type ExplorerFlowNode } from './layout'
import { TerraformNode } from './TerraformNode'
import type { DeclarationKind, Diagnostic, GraphEdge, GraphNode, Snapshot, SourceRange } from './types'

const nodeTypes = { terraform: TerraformNode }
const CodeViewer = lazy(() => import('./CodeViewer').then((module) => ({ default: module.CodeViewer })))
const allKinds: DeclarationKind[] = ['resource', 'data', 'module', 'variable', 'local', 'output']
const kindLabels: Record<DeclarationKind, string> = {
  resource: 'Resources',
  data: 'Data',
  module: 'Modules',
  variable: 'Variables',
  local: 'Locals',
  output: 'Outputs',
}

interface Selection {
  nodeId?: string
  file?: string
  range?: SourceRange
}

export function App() {
  const [path, setPath] = useState(() => localStorage.getItem('terraform-explorer:last-path') ?? '')
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null)
  const [selection, setSelection] = useState<Selection>({})
  const [source, setSource] = useState('')
  const [sourceLoading, setSourceLoading] = useState(false)
  const [scanLoading, setScanLoading] = useState(false)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [graphOnly, setGraphOnly] = useState(false)
  const [sourceOpen, setSourceOpen] = useState(false)
  const [activeKinds, setActiveKinds] = useState(() => new Set<DeclarationKind>(allKinds))
  const [flowNodes, setFlowNodes] = useState<ExplorerFlowNode[]>([])
  const [flowEdges, setFlowEdges] = useState<Edge[]>([])
  const [reactFlow, setReactFlow] = useState<ReactFlowInstance<ExplorerFlowNode> | null>(null)
  const sourceCache = useRef(new Map<string, string>())

  const selectedNode = useMemo(
    () => snapshot?.graph.nodes.find((node) => node.id === selection.nodeId),
    [snapshot, selection.nodeId],
  )
  const focusedGraph = useMemo(
    () => focusGraph(flowNodes, flowEdges, snapshot?.graph.edges ?? [], selection.nodeId, search),
    [flowNodes, flowEdges, snapshot, selection.nodeId, search],
  )

  useEffect(() => {
    if (!snapshot) {
      setFlowNodes([])
      setFlowEdges([])
      return
    }
    let current = true
    layoutGraph(snapshot.graph.nodes, snapshot.graph.edges, activeKinds).then((layout) => {
      if (!current) return
      setFlowNodes(layout.nodes)
      setFlowEdges(layout.edges)
      window.setTimeout(() => reactFlow?.fitView({ padding: 0.18, duration: 350 }), 30)
    })
    return () => {
      current = false
    }
  }, [snapshot, activeKinds, reactFlow])

  useEffect(() => {
    const fitTimer = window.setTimeout(() => reactFlow?.fitView({ padding: graphOnly ? 0.08 : 0.18, duration: 300 }), 40)
    if (!graphOnly) return () => window.clearTimeout(fitTimer)

    function exitGraphView(event: KeyboardEvent) {
      if (event.key === 'Escape') setGraphOnly(false)
    }
    window.addEventListener('keydown', exitGraphView)
    return () => {
      window.clearTimeout(fitTimer)
      window.removeEventListener('keydown', exitGraphView)
    }
  }, [graphOnly, reactFlow])

  useEffect(() => {
    if (!sourceOpen) return
    function closeSourceWindow(event: KeyboardEvent) {
      if (event.key === 'Escape') setSourceOpen(false)
    }
    window.addEventListener('keydown', closeSourceWindow)
    return () => window.removeEventListener('keydown', closeSourceWindow)
  }, [sourceOpen])

  useEffect(() => {
    const file = selection.file
    if (!snapshot || !file) {
      setSource('')
      return
    }

    const cacheKey = `${snapshot.id}:${file}`
    const cached = sourceCache.current.get(cacheKey)
    if (cached !== undefined) {
      setSource(cached)
      return
    }

    let current = true
    setSourceLoading(true)
    readSource(snapshot.id, file)
      .then((response) => {
        if (!current) return
        sourceCache.current.set(cacheKey, response.content)
        setSource(response.content)
      })
      .catch((reason: unknown) => current && setError(messageFrom(reason)))
      .finally(() => current && setSourceLoading(false))
    return () => {
      current = false
    }
  }, [snapshot, selection.file])

  async function scan(explicitPath?: string) {
    const target = (explicitPath ?? path).trim()
    if (!target) {
      setError('Enter a local directory containing Terraform .tf files.')
      return
    }
    setScanLoading(true)
    setError('')
    setSelection({})
    setSource('')
    setSourceOpen(false)
    try {
      const next = await createScan(target)
      setSnapshot(next)
      setPath(target)
      localStorage.setItem('terraform-explorer:last-path', target)
      const first = next.graph.nodes[0]
      if (first) setSelection({ nodeId: first.id, file: first.file, range: first.range })
    } catch (reason) {
      setError(messageFrom(reason))
    } finally {
      setScanLoading(false)
    }
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    void scan()
  }

  function selectNode(node: GraphNode) {
    setSelection({ nodeId: node.id, file: node.file, range: node.range })
  }

  function selectFile(file: string, range?: SourceRange) {
    setSelection({ file, range })
    setSourceOpen(true)
  }

  function toggleKind(kind: DeclarationKind) {
    setActiveKinds((current) => {
      const next = new Set(current)
      if (next.has(kind)) next.delete(kind)
      else next.add(kind)
      return next
    })
  }

  const moduleName = snapshot ? basename(snapshot.analysis.module.path) : 'No module open'
  const dependencies = selectedNode ? dependencyDetails(snapshot!, selectedNode) : null

  return (
    <div className={`app-shell ${graphOnly ? 'app-shell--graph-only' : ''}`}>
      <header className="app-header">
        <div className="brand">
          <span className="brand__mark"><Network size={19} /></span>
          <span>Terraform Explorer</span>
          <span className="brand__badge">LOCAL</span>
        </div>

        <form className="path-form" onSubmit={submit}>
          <FolderOpen size={16} />
          <input
            aria-label="Terraform module directory"
            value={path}
            onChange={(event) => setPath(event.target.value)}
            placeholder="/path/to/terraform/module"
            spellCheck={false}
          />
          {path && <button type="button" className="icon-button" onClick={() => setPath('')} aria-label="Clear path"><X size={14} /></button>}
          <button className="primary-button" disabled={scanLoading} type="submit">
            {scanLoading ? <><span className="spinner" /> Scanning</> : <><TerminalSquare size={15} /> Analyze</>}
          </button>
        </form>

        <div className="safety-pill" title="The application only reads Terraform source files">
          <ShieldCheck size={15} /> Read-only
        </div>
      </header>

      {error && (
        <div className="error-banner" role="alert">
          <AlertTriangle size={16} />
          <span>{error}</span>
          <button className="icon-button" onClick={() => setError('')} aria-label="Dismiss error"><X size={14} /></button>
        </div>
      )}

      <main className="workspace">
        <aside className="sidebar">
          <section className="module-card">
            <div className="module-card__eyebrow">PROJECT ROOT</div>
            <div className="module-card__name"><CircleDot size={15} /> {moduleName}</div>
            <div className="module-card__path" title={snapshot?.analysis.module.path}>{snapshot?.analysis.module.path ?? 'Choose a local directory to begin'}</div>
            {snapshot && (
              <div className="module-card__stats">
                <span><b>{snapshot.graph.nodes.length}</b> objects</span>
                <span><b>{snapshot.graph.edges.length}</b> links</span>
                <span><b>{snapshot.analysis.modules.length}</b> modules</span>
              </div>
            )}
          </section>

          <div className="sidebar-search">
            <Search size={14} />
            <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search objects" />
          </div>

          <section className="sidebar-section sidebar-section--grow">
            <div className="section-heading"><span>FILES</span><span>{snapshot?.analysis.files.length ?? 0}</span></div>
            <div className="file-list">
              {snapshot?.analysis.files.map((file) => {
                const objectCount = snapshot.graph.nodes.filter((node) => node.file === file.path).length
                return (
                  <button
                    key={file.path}
                    className={`file-row ${selection.file === file.path && !selection.nodeId ? 'is-active' : ''}`}
                    onClick={() => selectFile(file.path)}
                  >
                    <FileCode2 size={15} />
                    <span>{file.path}</span>
                    <small>{objectCount}</small>
                  </button>
                )
              })}
              {!snapshot && <div className="empty-list">No files scanned</div>}
            </div>
          </section>

          <section className="sidebar-section diagnostics-section">
            <div className="section-heading">
              <span>DIAGNOSTICS</span>
              <span className={snapshot?.analysis.diagnostics.length ? 'count-warn' : ''}>{snapshot?.analysis.diagnostics.length ?? 0}</span>
            </div>
            <div className="diagnostic-list">
              {snapshot?.analysis.diagnostics.slice(0, 4).map((diagnostic, index) => (
                <DiagnosticRow key={`${diagnostic.summary}-${index}`} diagnostic={diagnostic} onSelect={selectFile} />
              ))}
              {snapshot && snapshot.analysis.diagnostics.length === 0 && (
                <div className="diagnostics-clear"><ShieldCheck size={14} /> No parser issues</div>
              )}
            </div>
          </section>
        </aside>

        <section className="graph-panel">
          <div className="graph-toolbar">
            <div className="filter-group">
              {allKinds.map((kind) => {
                const count = snapshot?.graph.nodes.filter((node) => node.kind === kind).length ?? 0
                return (
                  <button key={kind} className={`filter-chip filter-chip--${kind} ${activeKinds.has(kind) ? 'is-active' : ''}`} onClick={() => toggleKind(kind)}>
                    <span className="filter-chip__dot" /> {kindLabels[kind]} <b>{count}</b>
                  </button>
                )
              })}
            </div>
            <div className="graph-toolbar__actions">
              {selectedNode && (
                <button className="focus-pill" onClick={() => setSelection((current) => ({ file: current.file }))} title="Clear dependency-chain focus">
                  <Network size={13} /> Chain: {selectedNode.label} <X size={12} />
                </button>
              )}
              <button className="toolbar-button" onClick={() => reactFlow?.fitView({ padding: 0.18, duration: 300 })} disabled={!snapshot}>
                <RefreshCw size={14} /> Fit graph
              </button>
              <button className="toolbar-button" onClick={() => { setSourceOpen(false); setGraphOnly(true) }} disabled={!snapshot} title="Show only the graph; press Escape to exit">
                <Maximize2 size={14} /> Graph view
              </button>
            </div>
          </div>

          <div className="graph-canvas">
            {snapshot ? (
              <ReactFlow<ExplorerFlowNode>
                nodes={focusedGraph.nodes}
                edges={focusedGraph.edges}
                nodeTypes={nodeTypes}
                onInit={setReactFlow}
                onNodeClick={(_, node) => selectNode(node.data.terraform)}
                onPaneClick={() => setSelection((current) => ({ file: current.file }))}
                minZoom={0.2}
                maxZoom={2}
                nodesDraggable={false}
                panOnDrag
                nodesConnectable={false}
                elementsSelectable
                proOptions={{ hideAttribution: true }}
              >
                <Background variant={BackgroundVariant.Dots} gap={22} size={1} color="#252c35" />
                {!graphOnly && <Controls showInteractive={false} position="bottom-left" />}
                {!graphOnly && <GraphMiniMap nodes={focusedGraph.nodes} edges={focusedGraph.edges} />}
              </ReactFlow>
            ) : (
              <Welcome onDemo={() => void scan('./testdata/simple-module')} />
            )}
            {snapshot && snapshot.graph.nodes.length === 0 && (
              <div className="canvas-empty"><FileCode2 size={28} /><b>No supported declarations found</b><span>The directory contains no resource, data, module, variable, local, or output blocks.</span></div>
            )}
            {!graphOnly && <div className={`graph-legend ${selectedNode ? 'graph-legend--focused' : ''}`}>
              {selectedNode ? <><Network size={13} /> Blue = upstream · green = downstream · click canvas to clear</> : <><Info size={13} /> Static configuration graph · not runtime infrastructure</>}
            </div>}
            {sourceOpen && selection.file && !graphOnly && (
              <div className="source-window" role="dialog" aria-modal="false" aria-labelledby="source-window-title">
                <div className="source-window__header">
                  <div className="source-window__identity">
                    <span className="source-window__icon"><FileCode2 size={16} /></span>
                    <div>
                      <span id="source-window-title">SOURCE FILE</span>
                      <code title={selection.file}>{selection.file}</code>
                    </div>
                  </div>
                  {selection.range && <span className="source-window__range">Lines {selection.range.start.line}–{selection.range.end.line}</span>}
                  <button className="icon-button source-window__close" onClick={() => setSourceOpen(false)} aria-label="Close source window" title="Close source window (Esc)"><X size={16} /></button>
                </div>
                <div className="source-window__editor">
                  <Suspense fallback={<div className="code-state"><span className="spinner" /> Loading local editor…</div>}>
                    <CodeViewer content={source} file={selection.file} range={selection.range} loading={sourceLoading} />
                  </Suspense>
                </div>
              </div>
            )}
          </div>
        </section>

        <aside className="inspector">
          <div className="inspector__header">
            <div>
              <span className="inspector__eyebrow">INSPECTOR</span>
              <h2>{selectedNode?.label ?? selection.file ?? 'Nothing selected'}</h2>
            </div>
            {selectedNode && <span className={`kind-badge kind-badge--${selectedNode.kind}`}>{selectedNode.kind}</span>}
          </div>

          {selectedNode ? (
            <>
              <div className="inspector__meta">
                <div className="address-block"><code>{selectedNode.address}</code></div>
                <div className="meta-grid">
                  <span>Defined in</span><button onClick={() => setSourceOpen(true)}>{selectedNode.file}:{selectedNode.range.start.line}</button>
                  <span>Module</span><b>{selectedNode.moduleAddress}</b>
                  <span>Block type</span><b>{selectedNode.type || selectedNode.kind}</b>
                </div>
              </div>
              <div className="dependency-summary">
                <DependencyColumn icon={<ArrowDownToLine size={14} />} title="Depends on" edges={dependencies?.incoming ?? []} nodes={snapshot!.graph.nodes} onSelect={selectNode} direction="source" />
                <DependencyColumn icon={<ArrowUpFromLine size={14} />} title="Used by" edges={dependencies?.outgoing ?? []} nodes={snapshot!.graph.nodes} onSelect={selectNode} direction="target" />
              </div>
              {(dependencies?.unresolved.length ?? 0) > 0 && (
                <div className="unresolved-box">
                  <AlertTriangle size={14} />
                  <div><b>{dependencies!.unresolved.length} unresolved reference{dependencies!.unresolved.length === 1 ? '' : 's'}</b><span>{dependencies!.unresolved.map((item) => item.reference.text).join(', ')}</span></div>
                </div>
              )}
            </>
          ) : (
            <div className="inspector-placeholder">
              <Braces size={20} />
              <span>{selection.file ? 'File selected. Open it in the source workspace.' : 'Select a graph object to inspect its dependencies and source.'}</span>
            </div>
          )}

          <div className="source-launch">
            <div className="source-launch__heading"><FileCode2 size={14} /> SOURCE</div>
            <button disabled={!selection.file} onClick={() => setSourceOpen(true)}>
              <ExternalLink size={16} />
              <span>
                <b>{selection.file ? 'Open source workspace' : 'No source selected'}</b>
                <small>{selection.file ?? 'Select a node or file first'}</small>
              </span>
            </button>
          </div>
        </aside>
      </main>
    </div>
  )
}

function Welcome({ onDemo }: { onDemo: () => void }) {
  return (
    <div className="welcome">
      <div className="welcome__mark"><Network size={31} /></div>
      <span className="welcome__eyebrow">LOCAL CONFIGURATION EXPLORER</span>
      <h1>Understand Terraform<br />before changing it.</h1>
      <p>Open a module directory to map declarations, static value flow, diagnostics, and source locations. Nothing is executed or uploaded.</p>
      <div className="welcome__actions">
        <button className="primary-button primary-button--large" onClick={() => document.querySelector<HTMLInputElement>('.path-form input')?.focus()}><FolderOpen size={17} /> Enter a local path</button>
        <button className="secondary-button" onClick={onDemo}>Open bundled example <ChevronRight size={15} /></button>
      </div>
      <div className="welcome__features">
        <span><ShieldCheck size={14} /> Read-only</span>
        <span><Network size={14} /> Static dependencies</span>
        <span><FileCode2 size={14} /> Source navigation</span>
      </div>
    </div>
  )
}

function DiagnosticRow({ diagnostic, onSelect }: { diagnostic: Diagnostic; onSelect: (file: string, range?: SourceRange) => void }) {
  return (
    <button className={`diagnostic-row diagnostic-row--${diagnostic.severity}`} onClick={() => diagnostic.subject?.file && onSelect(diagnostic.subject.file, diagnostic.subject)}>
      <AlertTriangle size={14} />
      <span><b>{diagnostic.summary}</b><small>{diagnostic.subject?.file ?? diagnostic.detail}</small></span>
    </button>
  )
}

function DependencyColumn({ icon, title, edges, nodes, onSelect, direction }: {
  icon: ReactNode
  title: string
  edges: GraphEdge[]
  nodes: GraphNode[]
  onSelect: (node: GraphNode) => void
  direction: 'source' | 'target'
}) {
  return (
    <div className="dependency-column">
      <div className="dependency-column__title">{icon} {title} <b>{edges.length}</b></div>
      {edges.slice(0, 3).map((edge) => {
        const node = nodes.find((candidate) => candidate.id === edge[direction])
        return node ? <button key={edge.id} onClick={() => onSelect(node)}><span>{node.label}</span><small>{edge.attributePath}</small></button> : null
      })}
      {edges.length === 0 && <span className="dependency-empty">No static links</span>}
    </div>
  )
}

function dependencyDetails(snapshot: Snapshot, node: GraphNode) {
  return {
    incoming: snapshot.graph.edges.filter((edge) => edge.target === node.id),
    outgoing: snapshot.graph.edges.filter((edge) => edge.source === node.id),
    unresolved: snapshot.graph.unresolved.filter((item) => item.consumerId === node.id),
  }
}

function basename(path: string) {
  return path.replace(/[\\/]+$/, '').split(/[\\/]/).pop() || path
}

function messageFrom(reason: unknown) {
  return reason instanceof Error ? reason.message : 'Unexpected application error'
}
