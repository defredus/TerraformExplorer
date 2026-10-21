import { useEffect, useRef, useState, type DragEvent, type PointerEvent, type ReactNode } from 'react'
import { GripVertical, RotateCcw } from 'lucide-react'
import {
  defaultDockLayout,
  dockPanel,
  normalizeDockLayout,
  oppositePanel,
  resizedPanelWidth,
  type DockPanelID,
  type DockSide,
} from './dockLayoutModel'

const storageKey = 'terraform-explorer:dock-layout'

interface DockWorkspaceProps {
  explorer: ReactNode
  center: ReactNode
  inspector: ReactNode
}

interface ResizeSession {
  side: DockSide
  startX: number
  startWidth: number
}

export function DockWorkspace({ explorer, center, inspector }: DockWorkspaceProps) {
  const [layout, setLayout] = useState(readStoredLayout)
  const [draggedPanel, setDraggedPanel] = useState<DockPanelID | null>(null)
  const [dropSide, setDropSide] = useState<DockSide | null>(null)
  const workspaceRef = useRef<HTMLElement>(null)
  const resizeSession = useRef<ResizeSession | null>(null)
  const rightPanel = oppositePanel(layout.leftPanel)
  const panels = { explorer, inspector }

  useEffect(() => {
    try {
      localStorage.setItem(storageKey, JSON.stringify(layout))
    } catch {
      // The layout remains usable when storage is disabled by the browser.
    }
  }, [layout])

  useEffect(() => () => document.body.classList.remove('is-resizing-dock'), [])

  function startPanelDrag(panel: DockPanelID, event: DragEvent<HTMLElement>) {
    setDraggedPanel(panel)
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('application/x-terraform-explorer-panel', panel)
  }

  function finishPanelDrag() {
    setDraggedPanel(null)
    setDropSide(null)
  }

  function acceptPanel(side: DockSide, event: DragEvent<HTMLElement>) {
    event.preventDefault()
    const value = event.dataTransfer.getData('application/x-terraform-explorer-panel')
    if (value === 'explorer' || value === 'inspector') {
      setLayout((current) => dockPanel(current, value, side))
    }
    finishPanelDrag()
  }

  function startResize(side: DockSide, event: PointerEvent<HTMLDivElement>) {
    event.preventDefault()
    event.currentTarget.setPointerCapture(event.pointerId)
    resizeSession.current = {
      side,
      startX: event.clientX,
      startWidth: side === 'left' ? layout.leftWidth : layout.rightWidth,
    }
    document.body.classList.add('is-resizing-dock')
  }

  function resize(event: PointerEvent<HTMLDivElement>) {
    const session = resizeSession.current
    const workspaceWidth = workspaceRef.current?.getBoundingClientRect().width
    if (!session || !workspaceWidth) return
    const oppositeWidth = session.side === 'left' ? layout.rightWidth : layout.leftWidth
    const width = resizedPanelWidth(session.startWidth, event.clientX - session.startX, session.side, workspaceWidth, oppositeWidth)
    setLayout((current) => session.side === 'left' ? { ...current, leftWidth: width } : { ...current, rightWidth: width })
  }

  function stopResize(event: PointerEvent<HTMLDivElement>) {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
    resizeSession.current = null
    document.body.classList.remove('is-resizing-dock')
  }

  function resetLayout() {
    setLayout({ ...defaultDockLayout })
  }

  return (
    <main
      ref={workspaceRef}
      className={`workspace dock-workspace ${draggedPanel ? 'is-dragging-panel' : ''}`}
      style={{ gridTemplateColumns: `${layout.leftWidth}px 5px minmax(420px, 1fr) 5px ${layout.rightWidth}px` }}
    >
      <DockSlot
        side="left"
        panel={layout.leftPanel}
        dropActive={dropSide === 'left'}
        onDragStart={startPanelDrag}
        onDragEnd={finishPanelDrag}
        onDragOver={(event) => { event.preventDefault(); setDropSide('left') }}
        onDragLeave={() => setDropSide((current) => current === 'left' ? null : current)}
        onDrop={acceptPanel}
        onReset={resetLayout}
      >
        {panels[layout.leftPanel]}
      </DockSlot>
      <DockResizer side="left" onPointerDown={startResize} onPointerMove={resize} onPointerUp={stopResize} onDoubleClick={resetLayout} />
      {center}
      <DockResizer side="right" onPointerDown={startResize} onPointerMove={resize} onPointerUp={stopResize} onDoubleClick={resetLayout} />
      <DockSlot
        side="right"
        panel={rightPanel}
        dropActive={dropSide === 'right'}
        onDragStart={startPanelDrag}
        onDragEnd={finishPanelDrag}
        onDragOver={(event) => { event.preventDefault(); setDropSide('right') }}
        onDragLeave={() => setDropSide((current) => current === 'right' ? null : current)}
        onDrop={acceptPanel}
        onReset={resetLayout}
      >
        {panels[rightPanel]}
      </DockSlot>
    </main>
  )
}

function DockSlot({ side, panel, dropActive, children, onDragStart, onDragEnd, onDragOver, onDragLeave, onDrop, onReset }: {
  side: DockSide
  panel: DockPanelID
  dropActive: boolean
  children: ReactNode
  onDragStart: (panel: DockPanelID, event: DragEvent<HTMLElement>) => void
  onDragEnd: () => void
  onDragOver: (event: DragEvent<HTMLElement>) => void
  onDragLeave: () => void
  onDrop: (side: DockSide, event: DragEvent<HTMLElement>) => void
  onReset: () => void
}) {
  return (
    <section
      className={`dock-slot dock-slot--${side} ${dropActive ? 'is-drop-target' : ''}`}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={(event) => onDrop(side, event)}
    >
      <div className="dock-tab" draggable onDragStart={(event) => onDragStart(panel, event)} onDragEnd={onDragEnd}>
        <GripVertical size={13} />
        <span>{panel === 'explorer' ? 'Explorer' : 'Inspector'}</span>
        <button draggable={false} onClick={onReset} aria-label="Reset panel layout" title="Reset panel layout"><RotateCcw size={12} /></button>
      </div>
      <div className="dock-slot__content">{children}</div>
      {dropActive && <div className="dock-drop-preview">Dock {side}</div>}
    </section>
  )
}

function DockResizer({ side, onPointerDown, onPointerMove, onPointerUp, onDoubleClick }: {
  side: DockSide
  onPointerDown: (side: DockSide, event: PointerEvent<HTMLDivElement>) => void
  onPointerMove: (event: PointerEvent<HTMLDivElement>) => void
  onPointerUp: (event: PointerEvent<HTMLDivElement>) => void
  onDoubleClick: () => void
}) {
  return (
    <div
      className="dock-resizer"
      role="separator"
      aria-label={`Resize ${side} panel`}
      aria-orientation="vertical"
      onPointerDown={(event) => onPointerDown(side, event)}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerUp}
      onDoubleClick={onDoubleClick}
    />
  )
}

function readStoredLayout() {
  try {
    const stored = localStorage.getItem(storageKey)
    return normalizeDockLayout(stored ? JSON.parse(stored) : undefined)
  } catch {
    return { ...defaultDockLayout }
  }
}
