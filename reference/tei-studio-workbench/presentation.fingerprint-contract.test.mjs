import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import {
  readCatalog, readInspection, relatedGraph, summarizeFindings,
} from './presentation.ts';
import { createResponseScope } from './responses.ts';

// Deterministic, host-neutral Presenter regression against the actual modules.
// It does not assert browser or Wails runtime integration.
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

async function simulatePresenter(catalogFingerprint, rawInspection) {
  const scope = createResponseScope();
  const held = deferred();
  const state = {
    graph: null, modelRevision: null, catalogProjection: null,
    editorText: null, inspectionVersion: null, inspectionPublished: false,
    failure: null, validationInvoked: false, diagnosticSummary: null,
  };
  await scope.run('catalog', async () => catalog(catalogFingerprint), (raw) => {
    const projection = readCatalog(raw);
    state.catalogProjection = projection;
    state.graph = relatedGraph(projection.entries, selected).input;
    state.modelRevision = projection.modelRevision;
  }, (error) => { throw error; });

  // Current main.ts resets the Command diagnostics and presentation summary
  // without claiming ValidateWorkspace has completed.
  state.diagnosticSummary = summarizeFindings(null).text;

  const pending = scope.run('inspection', () => held.promise, (raw) => {
    const view = readInspection(raw, selected, state.catalogProjection);
    state.editorText = view.text;
    state.inspectionVersion = view.version;
    state.inspectionPublished = true;
  }, (error) => {
    state.failure = error instanceof Error ? error.message : 'OPERATION_FAILED';
    if (state.failure === 'STALE_MODEL_PROJECTION') state.graph = null;
  });

  assert.equal(state.editorText, null, 'Inspect is unresolved before delivery');
  held.resolve(rawInspection);
  assert.equal(await pending, true, 'current request was handled, even when rejected');
  scope.dispose();
  return state;
}

test('matching A Catalog and deferred A Inspect are accepted', async () => {
  const state = await simulatePresenter('A', inspection('A'));
  assert.equal(state.modelRevision, 'rev-A');
  assert.equal(state.graph.nodes.length, 1);
  assert.equal(state.inspectionPublished, true);
  assert.equal(state.failure, null);
  assert.deepEqual(JSON.parse(state.editorText), { version: 'A', id: 'sales.order' });
  assert.equal(state.inspectionVersion, 'world-A:source-A');
  assert.equal(state.catalogProjection.worldFingerprint, 'world-A');
  assert.equal(state.catalogProjection.sourceIndexFingerprint, 'source-A');
});

test('A Catalog + deferred B Inspect rejects mixed-version model and graph', async () => {
  const state = await simulatePresenter('A', inspection('B'));
  assert.equal(state.modelRevision, 'rev-A');
  assert.equal(state.inspectionPublished, false);
  assert.equal(state.editorText, null);
  assert.equal(state.graph, null);
  assert.equal(state.failure, 'STALE_MODEL_PROJECTION');
});

test('World fingerprint mismatch alone is rejected', async () => {
  const raw = inspection('A');
  raw.worldFingerprint = 'world-B';
  const state = await simulatePresenter('A', raw);
  assert.equal(state.failure, 'STALE_MODEL_PROJECTION');
  assert.equal(state.editorText, null);
});

test('Source Index fingerprint mismatch alone is rejected', async () => {
  const raw = inspection('A');
  raw.sourceIndexFingerprint = 'source-B';
  const state = await simulatePresenter('A', raw);
  assert.equal(state.failure, 'STALE_MODEL_PROJECTION');
  assert.equal(state.editorText, null);
});

test('unrun Validate has unverified summary, not zero findings', async () => {
  const state = await simulatePresenter('A', inspection('A'));
  assert.equal(state.validationInvoked, false);
  assert.equal(state.diagnosticSummary, '未検証');
  assert.notEqual(state.diagnosticSummary, 'エラー 0・警告 0');
});

test('only completed Validate results display zero or warning counts', () => {
  assert.equal(summarizeFindings([]).text, 'エラー 0・警告 0');
  assert.equal(summarizeFindings([{id:'w',message:'warn',severity:'warning'}]).text,
    'エラー 0・警告 1');
  assert.equal(summarizeFindings(null).verified, false);
  assert.equal(summarizeFindings([]).verified, true);
});

test('Workbench wires snapshot matching and unverified state before painting', () => {
  const main = readFileSync(new URL('./main.ts', import.meta.url), 'utf8');
  assert.match(main, /readInspection\(unwrap\(reply\), subject, catalogSnapshot!\)/);
  assert.match(main, /catalogSnapshot = catalog;/);
  assert.match(main, /setFindings\(null\);/);
  assert.match(main, /context === workspaceEpoch\) await validate\(\);/);
});
