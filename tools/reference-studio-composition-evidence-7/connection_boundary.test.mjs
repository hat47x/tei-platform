// Independent #7 suite for #1833 exact 05926052 and #1849 exact 9a290dc.
// Runs in a fixed non-merge snapshot with the actual #1/#Work source files.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { resolve } from 'node:path';
import { composeUI } from '../reference-studio-static-composition/compose-ui.mjs';
import {
  connectWorkbenchSource, inspectGeneratedAssets, gitBlobSHA,
  WORKBENCH_HEAD, WORKBENCH_MAIN_BLOB,
} from '../reference-studio-static-composition/connect-workbench.mjs';

const root = resolve(fileURLToPath(new URL('../..', import.meta.url)));
const src = name => resolve(root, name);
const b = async name => readFile(src(name));
const fixed = Object.freeze({
  'tools/reference-studio-static-composition/compose-ui.mjs': '6e20b723ff7f0a3abf2ddce9267c879feeff3599',
  'tools/reference-studio-static-composition/connect-workbench.mjs': '29249c44d02af687103eff5124db95e5bea88c00',
  'tools/reference-studio-static-composition/fixture.json': '48f1167a957dfe07ba36c51d2c7b15e54d38c13a',
  'reference/tei-studio-workbench/main.ts': '2a65a9f70245068bf2455248d98d280b382b01e4',
  'cmd/tei-studio/main.go': '8d7698e371aabcde0bdc2e9d248a94c5f19fe791',
  'internal/studiohost/readonly.go': 'f1bd33f0e48a63bf9997f74285a23f8870fa32ae',
  'cmd/tei-studio/assets/client.mjs': '1afec5762a96bf34eead768a9fc7de4b7eced564',
});
const sha256 = input => createHash('sha256').update(input).digest('hex');
const mutate = (raw, callback) => {
  const copy = JSON.parse(raw);
  callback(copy);
  return Buffer.from(JSON.stringify(copy));
};

test('only the requested exact #1/#Work/#2 source revisions are used', async () => {
  for (const [name, blob] of Object.entries(fixed)) {
    assert.equal(gitBlobSHA((await b(name)).toString('utf8')), blob, name);
  }
  assert.equal(WORKBENCH_HEAD, '9a290dc61f585dfbf292e9032e5c5bbbad04351a');
  assert.equal(WORKBENCH_MAIN_BLOB, fixed['reference/tei-studio-workbench/main.ts']);
});

test('real fixed workbench preserves entire Host code except two UI import lines', async () => {
  const original = (await b('reference/tei-studio-workbench/main.ts')).toString('utf8');
  const graph = 'import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";\n';
  const commands = 'import { mountCommandPanel } from "../tei-studio-command/command.ts";\n';
  const generated = 'import { mountReadonlyGraph, mountCommandPanel } from ' +
    '"../../tools/reference-studio-static-composition/generated/ui.generated.mjs";\n';
  assert.equal(original.split(graph).length, 2);
  assert.equal(original.split(commands).length, 2);
  const expected = original.replace(graph, generated).replace(commands, '');
  const connected = connectWorkbenchSource(original);
  assert.equal(connected, expected, 'all code outside the two import lines must be identical');
  assert.equal(connected.split('ui.generated.mjs').length, 2);
  for (const marker of [
    'createReadOnlyClient', 'client.listWorkspaces(',
    'client.catalogWorkspace(', 'client.validateWorkspace(',
    'client.inspectSubject(', 'workspaceEpoch', 'inspectionEpoch',
    'window.addEventListener(', 'panel.dispose()', 'graph.dispose()', 'editor.destroy()',
  ]) {
    assert.ok(connected.includes(marker), marker);
  }
  assert.ok(!connected.includes('staticservices.Services('));
  assert.throws(() => connectWorkbenchSource(connected), /workbench_revision_mismatch/);
  assert.throws(() => connectWorkbenchSource(original + '\n'), /workbench_revision_mismatch/);
});

test('actual Kit resolver yields stable two slots, alias exports and pinned digests', async () => {
  const raw = await b('tools/reference-studio-static-composition/fixture.json');
  const first = composeUI(raw), second = composeUI(raw);
  assert.deepEqual(second, first);
  assert.equal(inspectGeneratedAssets(first.source, first.declarations, first.provenance), true);
  assert.equal(first.provenance.fixtureSha256, sha256(raw));
  assert.equal(first.provenance.generatedSha256, sha256(first.source));
  assert.equal(first.provenance.declarationsSha256, sha256(first.declarations));
  assert.deepEqual(first.provenance.uiContributions.map(v => v.slot), ['commands', 'graph']);
  assert.ok(first.source.includes('export const mountCommandPanel = ui0;'));
  assert.ok(first.source.includes('export const mountReadonlyGraph = ui1;'));
});

test('reordered declarations retain generated source order, new input hash is explicit', async () => {
  const raw = await b('tools/reference-studio-static-composition/fixture.json');
  const original = composeUI(raw);
  const changed = composeUI(mutate(raw, p => {
    p.services.reverse();
    p.uiKit.changes.reverse();
    p.uiKit.slots.reverse();
  }));
  assert.equal(original.source, changed.source);
  assert.equal(original.declarations, changed.declarations);
  assert.equal(original.provenance.generatedSha256, changed.provenance.generatedSha256);
  assert.equal(original.provenance.declarationsSha256, changed.provenance.declarationsSha256);
  assert.notEqual(original.provenance.fixtureSha256, changed.provenance.fixtureSha256);
});

test('unknown, duplicate, missing, version, scope and inheritance failures remain denied', async () => {
  const raw = await b('tools/reference-studio-static-composition/fixture.json');
  const negatives = [
    ['missing-service', p => p.services.pop(), /invalid_poc_fixture/],
    ['unknown-implementation', p => p.uiKit.changes[0].implementation = 'unknown', /implementation_missing/],
    ['duplicate-slot', p => p.uiKit.changes[1].slot = 'graph', /invalid_change/],
    ['duplicate-implementation', p => p.uiKit.changes[1].implementation = 'studio.graph', /duplicate_ui_component/],
    ['unknown-slot', p => p.uiKit.changes[1].slot = 'secret', /unknown_extension_point/],
    ['framework', p => p.uiKit.framework.major = 2, /framework_mismatch/],
    ['invalid-version', p => p.uiKit.version = 'latest', /invalid_request/],
    ['valid-but-unaudited-version', p => p.uiKit.version = '0.2.0', /unsupported_ui_kit_identity_or_version/],
    ['different-kit', p => p.uiKit.id = 'other.kit', /unsupported_ui_kit_identity_or_version/],
    ['wrong-scope', p => p.scope = 'other.tenant', /invalid_poc_fixture/],
    ['missing-parent', p => {
      p.uiKit.extends = {id: 'absent.kit', version: '0.1.0'};
      delete p.uiKit.slots;
    }, /missing_exact_parent/],
    ['inheritance-cycle', p => {
      p.uiKit.extends = {id: 'studio.readonly', version: '0.1.0'};
      delete p.uiKit.slots;
    }, /inheritance_cycle/],
  ];
  for (const [label, edit, expected] of negatives) {
    assert.throws(() => composeUI(mutate(raw, edit)), expected, label);
  }
});

test('tampered generated declaration/source/trace is not accepted', async () => {
  const raw = await b('tools/reference-studio-static-composition/fixture.json');
  const {source, declarations, provenance} = composeUI(raw);
  assert.throws(() => inspectGeneratedAssets(source + '\n', declarations, provenance),
    /generated_asset_digest_mismatch/);
  assert.throws(() => inspectGeneratedAssets(source, declarations + '\n', provenance),
    /generated_asset_digest_mismatch/);
  const broken = source.replace('export const mountReadonlyGraph = ', 'export const other = ');
  assert.throws(() => inspectGeneratedAssets(broken, declarations,
    {...provenance, generatedSha256: sha256(broken)}), /generated_mount_export_missing/);
  assert.throws(() => inspectGeneratedAssets(source, declarations,
    {...provenance, uiContributions: [{slot: 'graph'}]}), /invalid_generated_contributions/);
});

test('Wails publishes only allowlist-enforcing StudioBridge, never raw generated services', async () => {
  const host = (await b('internal/studiohost/readonly.go')).toString('utf8');
  const bridge = (await b('cmd/tei-studio/main.go')).toString('utf8');
  assert.match(bridge, /studiohost\.NewReadOnly\(paths\)/);
  assert.match(bridge, /application\.NewService\(&StudioBridge\{host: host\}\)/);
  assert.doesNotMatch(bridge, /staticservices\.Services\(|NewService\(&CatalogService|NewService\(&ValidateService/);
  const methods = [...bridge.matchAll(/func \(b \*StudioBridge\) ([A-Za-z]+)\(/g)]
    .map(v => v[1]).sort();
  assert.deepEqual(methods, [
    'CatalogWorkspace', 'InspectSubject', 'ListWorkspaces', 'ValidateWorkspace',
  ]);
  for (const method of methods) assert.ok(bridge.includes('return b.host.' + method + '('));
  assert.match(host, /h\.roots\[id\]/);
  for (const method of ['CatalogWorkspace', 'ValidateWorkspace', 'InspectSubject']) {
    assert.match(host, new RegExp('func \\(h \\*ReadOnly\\) ' + method + '\\([^]*?h.workspace\\(id\\)'));
  }
  assert.match(host, /WORKSPACE_NOT_ALLOWED/);
});
