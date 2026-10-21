import assert from 'node:assert/strict'
import test from 'node:test'
import { createGraphMiniMapModel } from '../src/graphMinimapModel.ts'

const node = (id, x) => ({
  id,
  address: `var.${id}`,
  kind: 'variable',
  role: 'none',
  dimmed: false,
  x,
  y: 20,
  width: 100,
  height: 60,
})

test('minimap model renders topology nodes and connecting edges', () => {
  const model = createGraphMiniMapModel(
    [node('source', 0), node('target', 200)],
    [{ id: 'edge-1', source: 'source', target: 'target', role: 'upstream' }],
  )

  assert.ok(model)
  assert.equal(model.points.length, 2)
  assert.equal(model.connections.length, 1)
  assert.equal(model.connections[0].role, 'upstream')
  assert.equal(model.connections[0].x1, model.points[0].x)
  assert.equal(model.connections[0].x2, model.points[1].x)
  assert.ok(model.points.every((point) => point.x > 0 && point.x < 220))
  assert.ok(model.points.every((point) => point.y > 0 && point.y < 150))
})

test('minimap model ignores edges whose endpoint is filtered out', () => {
  const model = createGraphMiniMapModel(
    [node('source', 0)],
    [{ id: 'edge-1', source: 'source', target: 'hidden', role: 'none' }],
  )

  assert.ok(model)
  assert.equal(model.connections.length, 0)
  assert.equal(createGraphMiniMapModel([], []), null)
})
