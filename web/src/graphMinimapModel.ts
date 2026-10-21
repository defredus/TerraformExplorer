import type { ChainRole } from './chain'
import type { DeclarationKind } from './types'

export const graphMapWidth = 220
export const graphMapHeight = 150
const mapPadding = 12

export interface MiniMapNodeInput {
  id: string
  address: string
  kind: DeclarationKind
  role: ChainRole
  dimmed: boolean
  x: number
  y: number
  width: number
  height: number
}

export interface MiniMapEdgeInput {
  id: string
  source: string
  target: string
  role: ChainRole
}

export function createGraphMiniMapModel(nodes: MiniMapNodeInput[], edges: MiniMapEdgeInput[]) {
  if (nodes.length === 0) return null

  const minX = Math.min(...nodes.map((node) => node.x))
  const minY = Math.min(...nodes.map((node) => node.y))
  const maxX = Math.max(...nodes.map((node) => node.x + node.width))
  const maxY = Math.max(...nodes.map((node) => node.y + node.height))
  const contentWidth = Math.max(maxX - minX, 1)
  const contentHeight = Math.max(maxY - minY, 1)
  const scale = Math.min(
    (graphMapWidth - mapPadding * 2) / contentWidth,
    (graphMapHeight - mapPadding * 2) / contentHeight,
  )
  const offsetX = (graphMapWidth - contentWidth * scale) / 2 - minX * scale
  const offsetY = (graphMapHeight - contentHeight * scale) / 2 - minY * scale
  const projectX = (value: number) => value * scale + offsetX
  const projectY = (value: number) => value * scale + offsetY
  const byID = new Map(nodes.map((node) => [node.id, node]))

  const points = nodes.map((node) => ({
    id: node.id,
    address: node.address,
    kind: node.kind,
    role: node.role,
    dimmed: node.dimmed,
    x: projectX(node.x + node.width / 2),
    y: projectY(node.y + node.height / 2),
  }))
  const connections = edges.flatMap((edge) => {
    const source = byID.get(edge.source)
    const target = byID.get(edge.target)
    if (!source || !target) return []
    return [{
      id: edge.id,
      role: edge.role,
      x1: projectX(source.x + source.width / 2),
      y1: projectY(source.y + source.height / 2),
      x2: projectX(target.x + target.width / 2),
      y2: projectY(target.y + target.height / 2),
    }]
  })

  return { points, connections, scale, offsetX, offsetY, projectX, projectY }
}
