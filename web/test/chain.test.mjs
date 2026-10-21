import assert from 'node:assert/strict'
import test from 'node:test'
import { dependencyChain } from '../src/chain.ts'

const edge = (id, source, target) => ({
  id,
  source,
  target,
  kind: 'value_flow',
  reference: '',
  attributePath: '',
  range: { start: { line: 1, column: 1, byte: 0 }, end: { line: 1, column: 1, byte: 0 } },
})

test('dependencyChain follows all upstream and downstream branches', () => {
  const chain = dependencyChain([
    edge('e1', 'root-a', 'local'),
    edge('e2', 'root-b', 'local'),
    edge('e3', 'local', 'selected'),
    edge('e4', 'selected', 'consumer-a'),
    edge('e5', 'selected', 'consumer-b'),
    edge('unrelated', 'x', 'y'),
  ], 'selected')

  assert.equal(chain.nodeRoles.get('selected'), 'selected')
  assert.equal(chain.nodeRoles.get('root-a'), 'upstream')
  assert.equal(chain.nodeRoles.get('root-b'), 'upstream')
  assert.equal(chain.nodeRoles.get('local'), 'upstream')
  assert.equal(chain.nodeRoles.get('consumer-a'), 'downstream')
  assert.equal(chain.nodeRoles.get('consumer-b'), 'downstream')
  assert.equal(chain.nodeRoles.has('x'), false)
  assert.equal(chain.edgeRoles.has('unrelated'), false)
})

test('dependencyChain terminates cycles and marks shared paths', () => {
  const chain = dependencyChain([
    edge('e1', 'a', 'b'),
    edge('e2', 'b', 'c'),
    edge('e3', 'c', 'a'),
  ], 'b')

  assert.equal(chain.nodeRoles.get('a'), 'both')
  assert.equal(chain.nodeRoles.get('b'), 'selected')
  assert.equal(chain.nodeRoles.get('c'), 'both')
  assert.equal(chain.edgeRoles.get('e1'), 'both')
  assert.equal(chain.edgeRoles.get('e2'), 'both')
  assert.equal(chain.edgeRoles.get('e3'), 'both')
})
