import { useMemo, type KeyboardEvent, type MouseEvent } from 'react'
import { useReactFlow, useStore, useViewport, type Edge } from '@xyflow/react'
import type { ChainRole } from './chain'
import { createGraphMiniMapModel, graphMapHeight, graphMapWidth } from './graphMinimapModel'
import type { ExplorerFlowNode } from './layout'
import type { DeclarationKind } from './types'

const fallbackNodeWidth = 224
const fallbackNodeHeight = 78

interface GraphMiniMapProps {
  nodes: ExplorerFlowNode[]
  edges: Edge[]
}

export function GraphMiniMap({ nodes, edges }: GraphMiniMapProps) {
  const viewport = useViewport()
  const flowWidth = useStore((state) => state.width)
  const flowHeight = useStore((state) => state.height)
  const reactFlow = useReactFlow<ExplorerFlowNode>()
  const model = useMemo(() => createGraphMiniMapModel(
    nodes.map((node) => ({
      id: node.id,
      address: node.data.terraform.address,
      kind: node.data.terraform.kind,
      role: node.data.chainRole,
      dimmed: node.data.dimmed,
      x: node.position.x,
      y: node.position.y,
      width: node.measured?.width ?? node.width ?? fallbackNodeWidth,
      height: node.measured?.height ?? node.height ?? fallbackNodeHeight,
    })),
    edges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      role: chainRole(edge.data?.chainRole),
    })),
  ), [nodes, edges])

  function centerAt(event: MouseEvent<SVGSVGElement>) {
    if (!model) return
    const bounds = event.currentTarget.getBoundingClientRect()
    const mapX = ((event.clientX - bounds.left) / bounds.width) * graphMapWidth
    const mapY = ((event.clientY - bounds.top) / bounds.height) * graphMapHeight
    void reactFlow.setCenter(
      (mapX - model.offsetX) / model.scale,
      (mapY - model.offsetY) / model.scale,
      { zoom: viewport.zoom, duration: 220 },
    )
  }

  function handleKey(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== 'Enter' && event.key !== ' ') return
    event.preventDefault()
    void reactFlow.fitView({ padding: 0.18, duration: 250 })
  }

  const viewportRect = model ? {
    x: model.projectX(-viewport.x / viewport.zoom),
    y: model.projectY(-viewport.y / viewport.zoom),
    width: (flowWidth / viewport.zoom) * model.scale,
    height: (flowHeight / viewport.zoom) * model.scale,
  } : null

  return (
    <div
      className="graph-minimap"
      role="button"
      tabIndex={0}
      aria-label="Graph overview. Click to navigate, press Enter to fit the graph."
      title="Graph overview · click to navigate · double-click to fit"
      onKeyDown={handleKey}
    >
      <svg
        viewBox={`0 0 ${graphMapWidth} ${graphMapHeight}`}
        role="img"
        aria-label="Dependency topology overview"
        onClick={centerAt}
        onDoubleClick={() => void reactFlow.fitView({ padding: 0.18, duration: 250 })}
      >
        <defs>
          <clipPath id="graph-minimap-clip">
            <rect x="1" y="1" width={graphMapWidth - 2} height={graphMapHeight - 2} rx="6" />
          </clipPath>
        </defs>
        <g clipPath="url(#graph-minimap-clip)">
          <rect width={graphMapWidth} height={graphMapHeight} className="graph-minimap__background" />
          {model?.connections.map((connection) => (
            <line
              key={connection.id}
              className={`graph-minimap__edge ${connection.role !== 'none' ? `is-${connection.role}` : ''}`}
              x1={connection.x1}
              y1={connection.y1}
              x2={connection.x2}
              y2={connection.y2}
            />
          ))}
          {model?.points.map((point) => (
            <circle
              key={point.id}
              className={`graph-minimap__node ${point.dimmed ? 'is-dimmed' : ''} ${point.role !== 'none' ? `is-${point.role}` : ''}`}
              cx={point.x}
              cy={point.y}
              r={point.role === 'selected' ? 4.5 : 3.1}
              fill={nodeColor(point.kind)}
            >
              <title>{point.address}</title>
            </circle>
          ))}
          {viewportRect && (
            <rect
              className="graph-minimap__viewport"
              x={viewportRect.x}
              y={viewportRect.y}
              width={viewportRect.width}
              height={viewportRect.height}
              rx="2"
            />
          )}
        </g>
      </svg>
    </div>
  )
}

function chainRole(value: unknown): ChainRole {
  return value === 'selected' || value === 'upstream' || value === 'downstream' || value === 'both' ? value : 'none'
}

function nodeColor(kind: DeclarationKind) {
  return {
    resource: '#67ddb1',
    data: '#68b9e4',
    module: '#e0b568',
    variable: '#b19af3',
    local: '#ee9878',
    output: '#d9d66f',
  }[kind]
}
