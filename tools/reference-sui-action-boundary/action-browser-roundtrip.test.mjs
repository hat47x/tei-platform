import assert from 'node:assert/strict';
import test from 'node:test';
import { createNativeActionBridge } from './native-action-bridge.mjs';
import { ActionCommitError, createJSONActionDispatcher } from './json-action-dispatch.mjs';
import { createConfirmedActionCommit } from './confirmed-action-commit.mjs';

// This is a synthetic browser/HTTP contract fixture, NOT an integrated Go Host
// or the SUI application's authoritative persistence implementation.
function scenario() {
  const snapshots = new Map([
    [JSON.stringify(['sui', 'canvas/1']), { revision: 'r1', value: {
      cards: [{ id: 'card-a', x: 1, y: 2, meta: { source: 'original' } }],
      edges: [{ id: 'opaque', type: 'future-kind' }],
      islands: [{ id: 'island-a', cardIds: ['card-a'] }],
      reviewAttribution: { 'card-a': 'reviewed' },
    }}],
    [JSON.stringify(['inventory', 'warehouse/sku-7']), { revision: 'r1', value: { sku: 'sku-7', quantity: 9 } }],
  ]);
  let writes = 0;
  let reads = 0;
  let allowed = true;
  let loseReply = false;
  let localState = null;
  const keyFor = (app, id) => JSON.stringify([app, id]);
  const serverFetch = async (path, init) => {
    assert.equal(path, '/actions/commit');
    assert.equal(init.method, 'POST');
    assert.equal(init.mode, 'same-origin');
    const intent = JSON.parse(init.body);
    const current = snapshots.get(keyFor(intent.applicationID, intent.resourceID));
    const reply = (status, body) => ({ status, ok: status === 200, json: async () => ({ protocolVersion: '1', ...body }) });
    if (!current || !allowed) return reply(403, { error: 'action_denied' });
    if (current.revision !== intent.expectedRevision) return reply(409, { error: 'revision_conflict' });
    const updated = structuredClone(current.value);
    if (intent.applicationID === 'sui' && intent.actionID === 'sui.move') {
      const card = updated.cards.find((c) => c.id === intent.payload.cardId);
      if (!card) return reply(400, { error: 'invalid_payload' });
      card.x = intent.payload.x;
      card.y = intent.payload.y;
    } else if (intent.applicationID === 'inventory' && intent.actionID === 'inventory.adjust') {
      if (updated.sku !== intent.payload.sku || !Number.isInteger(intent.payload.delta)) return reply(400, { error: 'invalid_payload' });
      updated.quantity += intent.payload.delta;
    } else {
      return reply(400, { error: 'invalid_action' });
    }
    current.value = updated;
    current.revision = 'r2';
    writes++;
    if (loseReply) throw new Error('connection interrupted after store commit');
    return reply(200, { revision: current.revision });
  };
  const confirmed = createConfirmedActionCommit({
    dispatchCommit: createJSONActionDispatcher('/actions/commit', serverFetch),
    readResource: async (intent) => {
      reads++;
      const snap = snapshots.get(keyFor(intent.applicationID, intent.resourceID));
      return snap && { applicationID: intent.applicationID, resourceID: intent.resourceID, ...structuredClone(snap) };
    },
    applySnapshot: async (resource) => { localState = structuredClone(resource.value); return true; },
    isCurrent: () => true,
  });
  return {
    bridge: createNativeActionBridge(confirmed),
    setAllowed(v) { allowed = v; },
    loseReply(v) { loseReply = v; },
    externalChange(app, id) { snapshots.get(keyFor(app, id)).revision = 'r3'; },
    get state() { return { writes, reads, localState, snapshots }; },
  };
}

test('SUI preview/cancel no writes; drop writes once and reloads preserving opaque fields', async () => {
  const h = scenario();
  const i = { applicationID: 'sui', resourceID: 'canvas/1', actionID: 'sui.move', expectedRevision: 'r1' };
  h.bridge.begin(i);
  h.bridge.preview({ cardId: 'card-a', x: 9, y: 10 });
  assert.equal(h.state.writes, 0);
  assert.equal(h.state.reads, 0);
  h.bridge.cancel();
  assert.equal(h.state.localState, null);
  h.bridge.begin(i);
  h.bridge.preview({ cardId: 'card-a', x: 9, y: 10 });
  assert.equal(await h.bridge.drop(), 'r2');
  assert.equal(h.state.writes, 1);
  assert.equal(h.state.reads, 1);
  assert.equal(h.state.localState.cards[0].x, 9);
  assert.equal(h.state.localState.cards[0].meta.source, 'original');
  assert.equal(h.state.localState.edges[0].type, 'future-kind');
  assert.deepEqual(h.state.localState.islands[0].cardIds, ['card-a']);
  assert.equal(h.state.localState.reviewAttribution['card-a'], 'reviewed');
});

test('independent Inventory application uses the same transport/confirmation pipeline', async () => {
  const h = scenario();
  h.bridge.begin({ applicationID: 'inventory', resourceID: 'warehouse/sku-7', actionID: 'inventory.adjust', expectedRevision: 'r1' });
  h.bridge.preview({ sku: 'sku-7', delta: 4 });
  assert.equal(await h.bridge.drop(), 'r2');
  assert.deepEqual(h.state.localState, { sku: 'sku-7', quantity: 13 });
});

test('server denial and externally changed revision never apply local data', async () => {
  const i = { applicationID: 'sui', resourceID: 'canvas/1', actionID: 'sui.move', expectedRevision: 'r1' };
  const denied = scenario();
  denied.setAllowed(false);
  denied.bridge.begin(i);
  denied.bridge.preview({ cardId: 'card-a', x: 9, y: 10 });
  await assert.rejects(denied.bridge.drop(), (e) => e instanceof ActionCommitError && e.code === 'action_denied' && e.status === 403);
  assert.equal(denied.state.writes, 0);
  assert.equal(denied.state.reads, 0);
  assert.equal(denied.state.localState, null);

  const stale = scenario();
  stale.externalChange('sui', 'canvas/1');
  stale.bridge.begin(i);
  stale.bridge.preview({ cardId: 'card-a', x: 9, y: 10 });
  await assert.rejects(stale.bridge.drop(), (e) => e instanceof ActionCommitError && e.code === 'revision_conflict' && e.status === 409);
  assert.equal(stale.state.writes, 0);
  assert.equal(stale.state.localState, null);
});

test('lost HTTP acknowledgement after server commit never marks UI committed', async () => {
  const h = scenario();
  h.loseReply(true);
  h.bridge.begin({ applicationID: 'sui', resourceID: 'canvas/1', actionID: 'sui.move', expectedRevision: 'r1' });
  h.bridge.preview({ cardId: 'card-a', x: 9, y: 10 });
  await assert.rejects(h.bridge.drop(), /connection interrupted/);
  assert.equal(h.state.writes, 1);
  assert.equal(h.state.reads, 0);
  assert.equal(h.state.localState, null);
});
