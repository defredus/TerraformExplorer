import { memo } from 'react'
import { Blocks, Box, Braces, Database, Share2, Variable } from 'lucide-react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import type { ExplorerFlowNode } from './layout'
import type { DeclarationKind } from './types'

const icons: Record<DeclarationKind, typeof Box> = {
  resource: Box,
  data: Database,
  module: Blocks,
  variable: Variable,
  local: Braces,
  output: Share2,
}

const labels: Record<DeclarationKind, string> = {
  resource: 'Resource',
  data: 'Data source',
  module: 'Module',
  variable: 'Input variable',
  local: 'Local value',
  output: 'Output',
}

function TerraformNodeComponent({ data, selected }: NodeProps<ExplorerFlowNode>) {
  const node = data.terraform
  const Icon = icons[node.kind]

  return (
    <div className={`terraform-node terraform-node--${node.kind} ${selected ? 'is-selected' : ''} ${data.dimmed ? 'is-dimmed' : ''}`}>
      <Handle type="target" position={Position.Left} className="node-handle" />
      <div className="terraform-node__icon"><Icon size={17} strokeWidth={1.8} /></div>
      <div className="terraform-node__content">
        <span className="terraform-node__kind">{labels[node.kind]}</span>
        <strong title={node.address}>{node.label}</strong>
        <span className="terraform-node__type">{node.type || node.address}</span>
      </div>
      <Handle type="source" position={Position.Right} className="node-handle" />
    </div>
  )
}

export const TerraformNode = memo(TerraformNodeComponent)
