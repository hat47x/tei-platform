import assert from 'node:assert/strict';
import test from 'node:test';
import { ActionConfirmationError, createConfirmedActionCommit } from './confirmed-action-commit.mjs';

const base = { protocolVersion: '1', applicationID: 'sui', resourceID: 'doc/1', actionID: 'sui.move', expectedRevision: 'r1', payload: { cardId: 'a', x: 20, y: 40 } };
const persisted = { applicationID: 'sui', resourceID: 'doc/1', revision: 'r2', value: { cards: [{ id: 'a', x: 20, y: 40 }], islands: [{ id: 'i', cardIds: ['a'] }], reviewAttribution: { a: 'human' } } };
const failure = (code, revision = null) => (err) => err instanceof ActionConfirmationError && err.code === code && err.committedRevision === revision;

function harness(overrides = {}) {
  let writes = 0;
  let reads = 0;
  let applications = 0;
  let current = true;
  let doc = { cards: [{ id: 'a', x: 1, y: 2 }], other: 'source' };
  const record = [];
  const controller = createConfirmedActionCommit({
    dispatchCommit: async (intent) => { writes++; record.push('commit'); return 'r2'; },
    readResource: async () => { reads++; record.push('read'); return structuredClone(persisted); },
    applySnapshot: async (snap) => { applications++; record.push('apply'); doc = snap.value; },
    isCurrent: () => current,
    ...overrides,
  });
  return { controller, get state() { return { writes, reads, applications, current, doc, record }; }, setCurrent(v) { current = v; } };
}

test('commits once, re-reads matching revision and preserves application-owned fields', async () => {
  const h = harness();
  assert.equal(await h.controller(base), 'r2');
  assert.deepEqual(h.state.record, ['commit', 'read', 'apply']);
  assert.deepEqual(h.state.doc, persisted.value);
});

test('local stale origin before dispatch blocks all side effects', async () => {
  const h = harness(); h.setCurrent(false);
  await assert.rejects(h.controller(base), failure('local_state_changed'));
  assert.deepEqual(h.state.record, []);
});

test('server denial or conflict leaves local state untouched and does not retry', async () => {
  for (const code of ['action_denied', 'revision_conflict']) {
    let attempted = 0;
    const h = harness({ dispatchCommit: async () => { attempted++; throw Object.assign(new Error(code), { code }); } });
    await assert.rejects(h.controller(base), { code });
    assert.equal(attempted, 1);
    assert.deepEqual(h.state.record, []);
    assert.equal(h.state.applications, 0);
  }
});

test('in-flight same resource is blocked, but independent resources may commit', async () => {
  let resolveFirst;
  let started = 0;
  const h = harness({ dispatchCommit: async () => { started++; if (started === 1) return await new Promise((resolve) => resolveFirst = resolve); return 'r2'; } });
  const first = h.controller(base);
  await assert.rejects(h.controller(base), failure('action_in_flight'));
  const independent = h.controller({ ...base, resourceID: 'doc/2' });
  // Use the same read fixture for the second resource to demonstrate strict resource identity checks.
  await assert.rejects(independent, failure('invalid_resource_snapshot', 'r2'));
  resolveFirst('r2');
  await first;
  assert.equal(started, 2);
});

test('server response acknowledged but readback fails: no local success, no retry', async () => {
  let writes = 0;
  const h = harness({ dispatchCommit: async () => { writes++; return 'r2'; }, readResource: async () => { throw new Error('network'); } });
  await assert.rejects(h.controller(base), failure('resource_read_failed', 'r2'));
  assert.equal(writes, 1);
  assert.equal(h.state.applications, 0);
});

test('wrong resource, stale revision, or missing document cannot be applied', async () => {
  const cases = [
    [ { ...persisted, resourceID: 'other' }, 'invalid_resource_snapshot' ],
    [ { ...persisted, revision: 'r3' }, 'revision_changed_after_commit' ],
    [ { ...persisted, value: undefined, revision: '' }, 'invalid_resource_snapshot' ],
  ];
  for (const [resource, code] of cases) {
    const h = harness({ readResource: async () => resource });
    await assert.rejects(h.controller(base), failure(code, 'r2'));
    assert.equal(h.state.applications, 0);
  }
});

test('local edits during network round-trip are not overwritten', async () => {
  const h = harness({ readResource: async () => { h.setCurrent(false); return persisted; } });
  await assert.rejects(h.controller(base), failure('local_state_changed', 'r2'));
  assert.equal(h.state.applications, 0);
});

test('failed application of acknowledged snapshot is not reported as a server rollback', async () => {
  const h = harness({ applySnapshot: async () => { throw new Error('stale UI'); } });
  await assert.rejects(h.controller(base), failure('snapshot_apply_failed', 'r2'));
});

test('inconsistent commit revision never reads resource', async () => {
  const h = harness({ dispatchCommit: async () => null });
  await assert.rejects(h.controller(base), failure('invalid_commit_result'));
  assert.deepEqual(h.state.record, []);
});

test('configuration or identity missing fails before network', async () => {
  assert.throws(() => createConfirmedActionCommit({}), TypeError);
  const h = harness();
  await assert.rejects(h.controller({ ...base, resourceID: '' }), TypeError);
  assert.deepEqual(h.state.record, []);
});

test('native preview and cancel do not mutate; drop only applies server-confirmed SUI state', async () => {
  const { createNativeActionBridge } = await import('./native-action-bridge.mjs');
  let seen = null;
  let applied = 0;
  const controller = createConfirmedActionCommit({
    dispatchCommit: async (intent) => { seen = intent; return 'r2'; },
    readResource: async () => persisted,
    applySnapshot: async (readback) => { applied++; assert.deepEqual(readback.value.islands, persisted.value.islands); },
    isCurrent: () => true,
  });
  const bridge = createNativeActionBridge(controller);
  bridge.begin(base);
  bridge.preview(base.payload);
  assert.equal(applied, 0);
  assert.equal(seen, null);
  assert.equal(bridge.cancel(), true);
  assert.equal(applied, 0);
  bridge.begin(base);
  bridge.preview(base.payload);
  assert.equal(await bridge.drop(), 'r2');
  assert.deepEqual(seen, base);
  assert.equal(applied, 1);
  assert.equal(bridge.phase, 'idle');
});


test('guard must return synchronous true, never a truthy Promise or object', async () => {
  for (const returnValue of [Promise.resolve(true), { current: true }, 1, undefined]) {
    const h = harness({ isCurrent: () => returnValue });
    await assert.rejects(h.controller(base), failure('local_state_changed'));
    assert.equal(h.state.writes, 0);
  }
});

test('throwing local guard after remote acknowledgment retains committed revision', async () => {
  let calls = 0;
  const h = harness({ isCurrent: () => { if (++calls === 2) throw new Error('UI unmounted'); return true; } });
  await assert.rejects(h.controller(base), failure('local_state_changed', 'r2'));
  assert.equal(h.state.applications, 0);
});

test('application may reject an atomic apply with false without reporting confirmed UI state', async () => {
  const h = harness({ applySnapshot: async () => false });
  await assert.rejects(h.controller(base), failure('snapshot_apply_failed', 'r2'));
  assert.equal(h.state.applications, 0);
});
