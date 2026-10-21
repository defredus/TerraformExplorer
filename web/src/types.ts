export type DeclarationKind = 'resource' | 'data' | 'module' | 'variable' | 'local' | 'output'

export interface Position {
  line: number
  column: number
  byte: number
}

export interface SourceRange {
  file?: string
  start: Position
  end: Position
}

export interface Diagnostic {
  severity: 'error' | 'warning'
  summary: string
  detail?: string
  subject?: SourceRange
}

export interface SourceFile {
  path: string
  bytes: number
  moduleAddress: string
}

export interface ModuleInstance {
  address: string
  path: string
  source?: string
  parent?: string
}

export interface GraphNode {
  id: string
  address: string
  moduleAddress: string
  label: string
  kind: DeclarationKind
  type?: string
  file: string
  range: SourceRange
  selectionRange: SourceRange
}

export interface GraphEdge {
  id: string
  source: string
  target: string
  kind: 'value_flow' | 'explicit_dependency' | 'cardinality'
  reference: string
  attributePath: string
  range: SourceRange
}

export interface UnresolvedReference {
  consumerId: string
  address?: string
  reason: string
  reference: {
    text: string
    attributePath: string
    range: SourceRange
  }
}

export interface Snapshot {
  id: string
  createdAt: string
  analysis: {
    schemaVersion: string
    module: ModuleInstance
    modules: ModuleInstance[]
    files: SourceFile[]
    declarations: unknown[]
    diagnostics: Diagnostic[]
  }
  graph: {
    nodes: GraphNode[]
    edges: GraphEdge[]
    unresolved: UnresolvedReference[]
  }
}

export interface SourceResponse {
  file: string
  content: string
}
