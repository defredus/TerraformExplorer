import type { GraphEdge } from './types'

export type ChainRole = 'none' | 'selected' | 'upstream' | 'downstream' | 'both'

export function dependencyChain(edges: GraphEdge[], focusNodeID?: string) {
  const nodeRoles = new Map<string, ChainRole>()
  const edgeRoles = new Map<string, ChainRole>()
  if (!focusNodeID) return { nodeRoles, edgeRoles }

  const incoming = new Map<string, GraphEdge[]>()
  const outgoing = new Map<string, GraphEdge[]>()
  for (const edge of edges) {
    incoming.set(edge.target, [...(incoming.get(edge.target) ?? []), edge])
    outgoing.set(edge.source, [...(outgoing.get(edge.source) ?? []), edge])
  }

  const upstreamNodes = walkChain(focusNodeID, incoming, (edge) => edge.source, edgeRoles, 'upstream')
  const downstreamNodes = walkChain(focusNodeID, outgoing, (edge) => edge.target, edgeRoles, 'downstream')

  nodeRoles.set(focusNodeID, 'selected')
  for (const id of upstreamNodes) {
    if (id !== focusNodeID) nodeRoles.set(id, downstreamNodes.has(id) ? 'both' : 'upstream')
  }
  for (const id of downstreamNodes) {
    if (id !== focusNodeID && !nodeRoles.has(id)) nodeRoles.set(id, 'downstream')
  }

  return { nodeRoles, edgeRoles }
}

function walkChain(
  start: string,
  adjacency: Map<string, GraphEdge[]>,
  nextNode: (edge: GraphEdge) => string,
  edgeRoles: Map<string, ChainRole>,
  role: 'upstream' | 'downstream',
) {
  const visited = new Set<string>([start])
  const queue = [start]
  while (queue.length > 0) {
    const current = queue.shift()!
    for (const edge of adjacency.get(current) ?? []) {
      const previousRole = edgeRoles.get(edge.id)
      edgeRoles.set(edge.id, previousRole && previousRole !== role ? 'both' : role)
      const next = nextNode(edge)
      if (!visited.has(next)) {
        visited.add(next)
        queue.push(next)
      }
    }
  }
  return visited
}
