import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { composeUI } from './compose-ui.mjs';

const raw = await readFile(new URL('./fixture.json', import.meta.url));
const fixture = () => JSON.parse(raw.toString());
const modified = edit => {
  const p = fixture();
  edit(p);
  return Buffer.from(JSON.stringify(p));
};

test('two UI imports are deterministic and preserve existing UI Kit trace', () => {
  const a = composeUI(raw);
  const b = composeUI(raw);
  assert.deepEqual(a, b);
  assert.deepEqual(a.provenance.uiContributions.map(c => c.slot), ['commands', 'graph']);
  assert.match(a.source, /mountCommandPanel/);
  assert.match(a.source, /mountReadonlyGraph/);
  assert.match(a.source, /studioUIContributions/);
  assert.ok(!a.source.includes('eval('));
  assert.match(a.provenance.fixtureSha256, /^[a-f0-9]{64}$/);
  assert.match(a.provenance.generatedSha256, /^[a-f0-9]{64}$/);
});

test('unrecognized UI implementation is rejected by existing Kit resolver', () => {
  const changed = modified(p => { p.uiKit.changes[0].implementation = 'untrusted.module'; });
  assert.throws(() => composeUI(changed), error => error.code === 'implementation_missing');
});

test('duplicate or undeclared visual slot is rejected by existing Kit resolver', () => {
  const duplicate = modified(p => { p.uiKit.changes[1].slot = 'graph'; });
  assert.throws(() => composeUI(duplicate), error => error.code === 'invalid_change');
  const unknown = modified(p => { p.uiKit.changes[1].slot = 'secret'; });
  assert.throws(() => composeUI(unknown), error => error.code === 'unknown_extension_point');
});

test('two slots cannot silently resolve to the same component', () => {
  const duplicate = modified(p => { p.uiKit.changes[1].implementation = 'studio.graph'; });
  assert.throws(() => composeUI(duplicate), /duplicate_ui_component/);
});

test('framework and version mismatch are not silently recovered', () => {
  const wrongFramework = modified(p => { p.uiKit.framework.major = 2; });
  assert.throws(() => composeUI(wrongFramework), error => error.code === 'framework_mismatch');
  const wrongVersion = modified(p => { p.uiKit.version = 'latest'; });
  // composeUI also uses kit.version as selected.version: the existing resolver
  // rejects that request before inspecting the catalog manifest.
  assert.throws(() => composeUI(wrongVersion), error => error.code === 'invalid_request');
});

test('unsupported UI slots and empty fixture are rejected', () => {
  const invalid = modified(p => { p.uiKit.slots.push('extra'); });
  assert.throws(() => composeUI(invalid), /unsupported_ui_slots/);
  assert.throws(() => composeUI('{}'), /invalid_poc_fixture/);
});
