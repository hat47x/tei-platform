import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readCatalog, readInspection, relatedGraph } from './presentation.ts';
import { createResponseScope } from './responses.ts';

// P0 contract fixture. Runs the exact current Presenter projection functions and
// response gate. This is an in-memory Presenter-state probe, not a browser E2E.
const selected = { kind: 'semantic-data', id: 'sales.order' };
function deferred() {
  let resolve;
  const promise = new Promise((yes) => { resolve = yes; });
  return { promise, resolve };
}
function catalog(fingerprint) {
  return {
    effect: 'read-only',
    modelRevision: `rev-${fingerprint}`,
    worldFingerprint: `world-${fingerprint}`,
    sourceIndexFingerprint: `source-${fingerprint}`,
    entries: [{ subject: selected, sourceStatus: 'indexed' }],
  };
}
function inspection(fingerprint) {
  return {
    effect: 'read-only', subject: selected,
    canonical: { version: fingerprint, id: selected.id },
    worldFingerprint: `world-${fingerprint}`,
    sourceIndexFingerprint: `source-${fingerprint}`,
    sourceStatus: 'indexed',
  };
}

// Mirrors main.ts: catalog is projected to entries/revision, then Graph is
// prepared from catalog while the deferred InspectSubject response is pending.
// The publish callback uses readInspection() as main.ts does: no comparison
// against the catalog's raw fingerprints is currently available.
async function simulatePresenter(catalogFingerprint, inspectFingerprint) {
  const scope = createResponseScope();
  const held = deferred();
  const state = {
    graph: null, modelRevision: null, catalogProjection: null,
    editorText: null, inspectionVersion: null,
    validationInvoked: false,
    diagnosticSummary: '未検証',
  };
  const catalogPublished = await scope.run('catalog', async () => catalog(catalogFingerprint), (raw) => {
    const projection = readCatalog(raw);
    state.catalogProjection = projection;
    state.graph = relatedGraph(projection.entries, selected).input;
    state.modelRevision = projection.modelRevision;
  }, (error) => { throw error; });
  assert.equal(catalogPublished, true);

  // loadWorkspace() calls setFindings([]) *before* its first ValidateWorkspace.
  // setFindings([]) uses the zero counters even though no validation ran yet.
  state.diagnosticSummary = 'エラー 0・警告 0';

  const pendingInspection = scope.run('inspection', () => held.promise, (raw) => {
    const projection = readInspection(raw, selected);
    state.editorText = projection.text;
    state.inspectionVersion = projection.version;
  }, (error) => { throw error; });
  assert.equal(state.editorText, null, 'Inspect has not completed before deferred delivery');
  held.resolve(inspection(inspectFingerprint));
  assert.equal(await pendingInspection, true);
  scope.dispose();
  return state;
}

test('same-version A Catalog + A deferred Inspect is displayed', async () => {
  const state = await simulatePresenter('A', 'A');
  assert.equal(state.modelRevision, 'rev-A');
  assert.equal(state.graph.nodes.length, 1);
  assert.deepEqual(JSON.parse(state.editorText), { version: 'A', id: 'sales.order' });
  assert.equal(state.inspectionVersion, 'world-A:source-A');
});

test('current-head counterexample: A graph and deferred B inspector are both published', async () => {
  const state = await simulatePresenter('A', 'B');
  assert.equal(state.modelRevision, 'rev-A');
  assert.equal(state.graph.nodes.length, 1);
  assert.deepEqual(JSON.parse(state.editorText), { version: 'B', id: 'sales.order' });
  assert.equal(state.inspectionVersion, 'world-B:source-B');
  assert.equal(state.catalogProjection.worldFingerprint, undefined);
  assert.equal(state.catalogProjection.sourceIndexFingerprint, undefined);
});

test('P0 RED: A Catalog + B deferred Inspect must reject mixed-version display', async () => {
  const state = await simulatePresenter('A', 'B');
  assert.equal(state.modelRevision, 'rev-A');
  assert.equal(state.graph.nodes.length, 1);
  assert.equal(state.editorText, null, 'stale B Inspect must not be shown alongside A Catalog graph');
});

test('P0 RED: unrun Validate must be presented as unverified, not 0 findings', async () => {
  const state = await simulatePresenter('A', 'A');
  assert.equal(state.validationInvoked, false);
  assert.equal(state.diagnosticSummary, '未検証',
    'a zero-finding summary requires a completed successful ValidateWorkspace');
});
