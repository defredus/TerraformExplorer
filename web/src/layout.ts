import { MarkerType, type Edge, type Node } from '@xyflow/react'
import type { ELK as ELKInstance, ElkNode } from 'elkjs/lib/elk-api'
import { dependencyChain, type ChainRole } from './chain'
import type { GraphEdge, GraphNode } from './types'

const nodeWidth = 224
const nodeHeight = 78
let elkPromise: Promise<ELKInstance> | undefined

async function getELK() {
  if (!elkPromise) {
    elkPromise = import('elkjs/lib/elk.bundled.js').then(({ default: ELK }) => new ELK())
  }
  return elkPromise
}

export interface ExplorerNodeData extends Record<string, unknown> {
  terraform: GraphNode
  dimmed: boolean
  chainRole: ChainRole
}

export type ExplorerFlowNode = Node<ExplorerNodeData, 'terraform'>

export async function layoutGraph(
  graphNodes: GraphNode[],
  graphEdges: GraphEdge[],
  activeKinds: Set<string>,
): Promise<{ nodes: ExplorerFlowNode[]; edges: Edge[] }> {
  const elk = await getELK()
  const visible = graphNodes.filter((node) => activeKinds.has(node.kind))
  const visibleIDs = new Set(visible.map((node) => node.id))
  const edges = graphEdges.filter((edge) => visibleIDs.has(edge.source) && visibleIDs.has(edge.target))

  const layout = await elk.layout({
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': 'RIGHT',
      'elk.spacing.nodeNode': '52',
      'elk.layered.spacing.nodeNodeBetweenLayers': '92',
      'elk.padding': '[top=42,left=42,bottom=42,right=42]',
      'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
    },
    children: visible.map((node) => ({ id: node.id, width: nodeWidth, height: nodeHeight })),
    edges: edges.map((edge) => ({ id: edge.id, sources: [edge.source], targets: [edge.target] })),
  })

  const positions = new Map<string, ElkNode>(layout.children?.map((child) => [child.id, child]) ?? [])
  const flowNodes: ExplorerFlowNode[] = visible.map((node) => {
    const position = positions.get(node.id)
    return {
      id: node.id,
      type: 'terraform',
      position: { x: position?.x ?? 0, y: position?.y ?? 0 },
      width: nodeWidth,
      height: nodeHeight,
      data: {
        terraform: node,
        dimmed: false,
        chainRole: 'none',
      },
    }
  })

  const flowEdges: Edge[] = edges.map((edge) => ({
    id: edge.id,
    source: edge.source,
    target: edge.target,
    type: 'smoothstep',
    markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
    className: `dependency-edge dependency-edge--${edge.kind}`,
    data: { terraform: edge },
  }))

  return { nodes: flowNodes, edges: flowEdges }
}

export function focusGraph(
  nodes: ExplorerFlowNode[],
  edges: Edge[],
  graphEdges: GraphEdge[],
  focusNodeID: string | undefined,
  search: string,
): { nodes: ExplorerFlowNode[]; edges: Edge[] } {
  const chain = dependencyChain(graphEdges, focusNodeID)
  const normalizedSearch = search.trim().toLowerCase()

  const focusedNodes = nodes.map((node) => {
    const terraform = node.data.terraform
    const role = chain.nodeRoles.get(node.id) ?? 'none'
    const haystack = `${terraform.address} ${terraform.type ?? ''} ${terraform.file}`.toLowerCase()
    const dimmed = focusNodeID ? role === 'none' : normalizedSearch !== '' && !haystack.includes(normalizedSearch)
    return {
      ...node,
      data: { ...node.data, dimmed, chainRole: role },
    }
  })

  const focusedEdges = edges.map((edge) => {
    const terraform = (edge.data?.terraform ?? {}) as Partial<GraphEdge>
    const role = chain.edgeRoles.get(edge.id) ?? 'none'
    const highlighted = role !== 'none'
    const color = chainColor(role)
    return {
      ...edge,
      animated: highlighted,
      zIndex: highlighted ? 3 : 0,
      className: [
        'dependency-edge',
        `dependency-edge--${terraform.kind ?? 'value_flow'}`,
        focusNodeID && !highlighted ? 'is-focus-dimmed' : '',
        highlighted ? 'is-chain' : '',
        highlighted ? `is-chain-${role}` : '',
      ].filter(Boolean).join(' '),
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: highlighted ? 19 : 16,
        height: highlighted ? 19 : 16,
        color,
      },
      data: { ...edge.data, chainRole: role },
    }
  })

  return { nodes: focusedNodes, edges: focusedEdges }
}

function chainColor(role: ChainRole) {
  switch (role) {
  case 'upstream': return '#66b9e8'
  case 'downstream': return '#64dfa9'
  case 'both': return '#e7b768'
  case 'selected': return '#f0f4f8'
  default: return '#667687'
  }
}
