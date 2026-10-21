import assert from 'node:assert/strict'
import test from 'node:test'
import {
  defaultDockLayout,
  dockPanel,
  normalizeDockLayout,
  resizedPanelWidth,
} from '../src/dockLayoutModel.ts'

test('dock layout swaps explorer and inspector between sides', () => {
  const swapped = dockPanel(defaultDockLayout, 'inspector', 'left')
  assert.equal(swapped.leftPanel, 'inspector')
  assert.equal(dockPanel(swapped, 'inspector', 'right').leftPanel, 'explorer')
})

test('dock widths are normalized and preserve a usable center viewport', () => {
  assert.deepEqual(normalizeDockLayout({ leftPanel: 'inspector', leftWidth: 10, rightWidth: 9999 }), {
    leftPanel: 'inspector',
    leftWidth: 190,
    rightWidth: 620,
  })
  assert.equal(resizedPanelWidth(250, 400, 'left', 1200, 390), 380)
  assert.equal(resizedPanelWidth(390, 500, 'right', 1200, 250), 190)
})
