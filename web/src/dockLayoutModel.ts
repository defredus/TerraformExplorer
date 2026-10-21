export type DockPanelID = 'explorer' | 'inspector'
export type DockSide = 'left' | 'right'

export interface DockLayoutState {
  leftPanel: DockPanelID
  leftWidth: number
  rightWidth: number
}

export const defaultDockLayout: DockLayoutState = {
  leftPanel: 'explorer',
  leftWidth: 250,
  rightWidth: 390,
}

const minPanelWidth = 190
const maxPanelWidth = 620
const minCenterWidth = 420
const resizerSpace = 10

export function normalizeDockLayout(value: unknown): DockLayoutState {
  if (!value || typeof value !== 'object') return { ...defaultDockLayout }
  const candidate = value as Partial<DockLayoutState>
  return {
    leftPanel: candidate.leftPanel === 'inspector' ? 'inspector' : 'explorer',
    leftWidth: clampWidth(candidate.leftWidth, defaultDockLayout.leftWidth),
    rightWidth: clampWidth(candidate.rightWidth, defaultDockLayout.rightWidth),
  }
}

export function dockPanel(layout: DockLayoutState, panel: DockPanelID, side: DockSide): DockLayoutState {
  const leftPanel = side === 'left' ? panel : oppositePanel(panel)
  return leftPanel === layout.leftPanel ? layout : { ...layout, leftPanel }
}

export function resizedPanelWidth(
  startWidth: number,
  pointerDelta: number,
  side: DockSide,
  workspaceWidth: number,
  oppositeWidth: number,
) {
  const desired = side === 'left' ? startWidth + pointerDelta : startWidth - pointerDelta
  const available = Math.max(minPanelWidth, workspaceWidth - oppositeWidth - minCenterWidth - resizerSpace)
  return Math.min(Math.max(desired, minPanelWidth), Math.min(maxPanelWidth, available))
}

export function oppositePanel(panel: DockPanelID): DockPanelID {
  return panel === 'explorer' ? 'inspector' : 'explorer'
}

function clampWidth(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value)
    ? Math.min(Math.max(value, minPanelWidth), maxPanelWidth)
    : fallback
}
