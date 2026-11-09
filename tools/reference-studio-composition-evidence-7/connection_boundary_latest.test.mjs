// #7 independent, strictly pinned latest PoC B -> single Workbench tests.
// Stages the exact #1833 / #1849 / Command source files without merging Draft PRs.
// Do not turn mocked Host replies into Host authorization claims.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { composeUI } from '../reference-studio-static-composition/compose-ui.mjs';
import {
  connectWorkbenchSource, inspectGeneratedAssets, inspectPinnedDependencies,
  pinnedWorkbenchRevision, inspectWorkbenchImports, WORKBENCH_PINS,
} from '../reference-studio-static-composition/connect-workbench.mjs';

const root = resolve(fileURLToPath(new URL('../../', import.meta.url)));
const read = path => readFile(resolve(root, path));
const blob = bytes => createHash('sha1').update(Buffer.from('blob ' + bytes.length + '\0')).update(bytes).digest('hex');
const sha256 = bytes => createHash('sha256').update(bytes).digest('hex');
const paths = Object.freeze({
  'tools/reference-studio-static-composition/fixture.json': '48f1167a957dfe07ba36c51d2c7b15e54d38c13a',
  'tools/reference-studio-static-composition/compose-ui.mjs': '012137687eb188369d13fa00377cec3e500c72c8',
  'tools/reference-studio-static-composition/connect-workbench.mjs': '63b9cefd3f4edfe52605ab3c0ed81a60eedef8d5',
  'reference/tei-studio-workbench/main.ts': 'cc2644e5a9167dbc3a0b5368ff270276a2ba1b67',
  'reference/tei-studio-workbench/responses.ts': '9fa305e227bee96b66002362121db3878e92327d',
  'reference/tei-studio-command/command.ts': '6c6ee81288c9857ca2f1c282639ba49f059f5535',
  'cmd/tei-studio/main.go': '8d7698e371aabcde0bdc2e9d248a94c5f19fe791',
  'internal/studiohost/readonly.go': 'f1bd33f0e48a63bf9997f74285a23f8870fa32ae',
});
const mutate = (raw, change) => {
  const value = JSON.parse(raw.toString('utf8'));
  change(value);
  return Buffer.from(JSON.stringify(value));
};

test('actual #1833 fe8e37 / #1849 c55ad2 / Command 01f502 sources are pinned', async () => {
  for (const [path, expected] of Object.entries(paths)) {
    assert.equal(blob(await read(path)), expected, path);
  }
  assert.deepEqual(WORKBENCH_PINS.map(x => x.head), [
    '9a290dc61f585dfbf292e9032e5c5bbbad04351a',
    'c55ad2285268533f5f1c90fbc9fe766bae57f009',
  ]);
  assert.equal(WORKBENCH_PINS[1].commandHead, '01f5022240fd86c6ac38c1107310e299a1615b48');
  assert.equal(WORKBENCH_PINS[1].commandBlob, paths['reference/tei-studio-command/command.ts']);
  assert.equal(WORKBENCH_PINS[1].responsesBlob, paths['reference/tei-studio-workbench/responses.ts']);
});

test('the real current Workbench + Command + response scope must all be accepted together', async () => {
  const source = (await read('reference/tei-studio-workbench/main.ts')).toString('utf8');
  const command = (await read('reference/tei-studio-command/command.ts')).toString('utf8');
  const responses = (await read('reference/tei-studio-workbench/responses.ts')).toString('utf8');
  const pin = pinnedWorkbenchRevision(source);
  assert.strictEqual(pin, WORKBENCH_PINS[1]);
  assert.equal(inspectPinnedDependencies(pin, command, responses), true);
  assert.throws(() => inspectPinnedDependencies(pin, command + '\n', responses),
    /command_revision_mismatch/);
  assert.throws(() => inspectPinnedDependencies(pin, command, responses + '\n'),
    /responses_revision_mismatch/);
  assert.throws(() => inspectPinnedDependencies({ ...pin }, command, responses),
    /workbench_revision_mismatch/);
  assert.throws(() => pinnedWorkbenchRevision(source + '\n'),
    /workbench_revision_mismatch/);
});

test('no pin=false bypass, no duplicate or missing imports, and no repeated connection', async () => {
  const source = (await read('reference/tei-studio-workbench/main.ts')).toString('utf8');
  const graph = 'import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";\n';
  const command = 'import { mountCommandPanel } from "../tei-studio-command/command.ts";\n';
  assert.equal(inspectWorkbenchImports(source), true);
  assert.throws(() => connectWorkbenchSource(source, { pin: false }),
    /workbench_pin_override_forbidden/);
  assert.throws(() => connectWorkbenchSource(source, {}),
    /workbench_pin_override_forbidden/);
  assert.throws(() => inspectWorkbenchImports(source.replace(graph, '')),
    /workbench_import_missing_or_duplicate/);
  assert.throws(() => inspectWorkbenchImports(source.replace(command, command + command)),
    /workbench_import_missing_or_duplicate/);
  const connected = connectWorkbenchSource(source);
  assert.throws(() => inspectWorkbenchImports(connected), /workbench_already_connected/);
  assert.throws(() => connectWorkbenchSource(connected), /workbench_revision_mismatch/);
});

test('connection changes exactly two imports and retains response generation + four Host operations', async () => {
  const source = (await read('reference/tei-studio-workbench/main.ts')).toString('utf8');
  const originalGraph = 'import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";\n';
  const originalCommand = 'import { mountCommandPanel } from "../tei-studio-command/command.ts";\n';
  const generated = 'import { mountReadonlyGraph, mountCommandPanel } from ' +
    '"../../tools/reference-studio-static-composition/generated/ui.generated.mjs";\n';
  assert.equal(source.split(originalGraph).length, 2);
  assert.equal(source.split(originalCommand).length, 2);
  const actual = connectWorkbenchSource(source);
  assert.equal(actual, source.replace(originalGraph, generated).replace(originalCommand, ''));
  for (const preserved of [
    'createReadOnlyClient', 'listWorkspaces(', 'catalogWorkspace(',
    'validateWorkspace(', 'inspectSubject(', 'workspaceEpoch', 'inspectionEpoch',
    'createResponseScope', 'responses.run(', 'responses.reset()', 'responses.dispose()',
    'panel.beginContext()', 'setDiagnosticsForContext', 'runtime.Call.ByName',
  ]) {
    assert.ok(actual.includes(preserved), 'missing after composition: ' + preserved);
  }
  assert.ok(!actual.includes('staticservices.Services('));
  assert.equal((actual.match(/ui\.generated\.mjs/g) ?? []).length, 1);
});

test('generated source/declarations/trace hashes are deterministic and match the fixture', async () => {
  const raw = await read('tools/reference-studio-static-composition/fixture.json');
  const first = composeUI(raw);
  assert.deepEqual(composeUI(raw), first);
  assert.equal(inspectGeneratedAssets(first.source, first.declarations, first.provenance), true);
  assert.equal(first.provenance.fixtureSha256, sha256(raw));
  assert.equal(first.provenance.generatedSha256, sha256(first.source));
  assert.equal(first.provenance.declarationsSha256, sha256(first.declarations));
  assert.deepEqual(first.provenance.uiContributions.map(x => x.slot), ['commands', 'graph']);
  for (const symbol of ['mountReadonlyGraph', 'mountCommandPanel']) {
    assert.ok(first.source.includes('export const ' + symbol + ' = '));
    assert.ok(first.declarations.includes('export { ' + symbol + ' } from '));
  }
});

test('reordered Go/UI selections retain generated bytes, not input fingerprint', async () => {
  const raw = await read('tools/reference-studio-static-composition/fixture.json');
  const original = composeUI(raw);
  const reordered = composeUI(mutate(raw, p => {
    p.services.reverse(); p.uiKit.slots.reverse(); p.uiKit.changes.reverse();
  }));
  assert.equal(reordered.source, original.source);
  assert.equal(reordered.declarations, original.declarations);
  assert.equal(reordered.provenance.generatedSha256, original.provenance.generatedSha256);
  assert.equal(reordered.provenance.declarationsSha256, original.provenance.declarationsSha256);
  assert.notEqual(reordered.provenance.fixtureSha256, original.provenance.fixtureSha256);
});

test('unknown/duplicate/incompatible services and missing selections reject in UI as in Go', async () => {
  const raw = await read('tools/reference-studio-static-composition/fixture.json');
  const inputs = [
    [p => p.services.pop(), /invalid_poc_fixture/],
    [p => p.services[0].id = 'unknown.service', /invalid_poc_fixture_services/],
    [p => p.services[1].id = p.services[0].id, /invalid_poc_fixture_services/],
    [p => p.services[1].version = '9.0.0', /invalid_poc_fixture_services/],
    [p => p.scope = 'other.scope', /invalid_poc_fixture/],
  ];
  for (const [change, expected] of inputs) {
    assert.throws(() => composeUI(mutate(raw, change)), expected);
  }
});

test('unknown UI, duplicate slots/components, kit versions and dependency cycles still reject', async () => {
  const raw = await read('tools/reference-studio-static-composition/fixture.json');
  const inputs = [
    [p => p.uiKit.changes[0].implementation = 'untrusted', /implementation_missing/],
    [p => p.uiKit.changes[1].slot = 'graph', /invalid_change/],
    [p => p.uiKit.changes[1].slot = 'unknown-slot', /unknown_extension_point/],
    [p => p.uiKit.changes[1].implementation = 'studio.graph', /duplicate_ui_component/],
    [p => p.uiKit.version = 'latest', /invalid_request/],
    [p => p.uiKit.version = '0.2.0', /unsupported_ui_kit_identity_or_version/],
    [p => p.uiKit.framework.major = 3, /framework_mismatch/],
    [p => {
      p.uiKit.extends = { id: 'missing.parent', version: '0.1.0' };
      delete p.uiKit.slots;
    }, /missing_exact_parent/],
    [p => {
      p.uiKit.extends = { id: p.uiKit.id, version: p.uiKit.version };
      delete p.uiKit.slots;
    }, /inheritance_cycle/],
  ];
  for (const [change, expected] of inputs) {
    assert.throws(() => composeUI(mutate(raw, change)), expected);
  }
});

test('tampered generated bytes, declarations and trace fail closed even if a digest is substituted', async () => {
  const raw = await read('tools/reference-studio-static-composition/fixture.json');
  const { source, declarations, provenance } = composeUI(raw);
  assert.throws(() => inspectGeneratedAssets(source + '\n', declarations, provenance),
    /generated_asset_digest_mismatch/);
  assert.throws(() => inspectGeneratedAssets(source, declarations + '\n', provenance),
    /generated_asset_digest_mismatch/);
  const altered = source.replace('export const mountReadonlyGraph', 'export const other');
  assert.throws(() => inspectGeneratedAssets(altered, declarations,
    { ...provenance, generatedSha256: sha256(altered) }),
  /generated_mount_export_missing/);
  assert.throws(() => inspectGeneratedAssets(source, declarations,
    { ...provenance, uiContributions: [{ slot: 'graph' }] }),
  /invalid_generated_contributions/);
});

test('fixed Wails surface stays allowlist-gated and exposes only four read-only RPCs', async () => {
  const bridge = (await read('cmd/tei-studio/main.go')).toString('utf8');
  const host = (await read('internal/studiohost/readonly.go')).toString('utf8');
  assert.ok(bridge.includes('studiohost.NewReadOnly(paths)'));
  assert.ok(bridge.includes('application.NewService(&StudioBridge{host: host})'));
  assert.ok(!bridge.includes('staticservices.Services('));
  const methods = [...bridge.matchAll(/func \(b \*StudioBridge\) ([A-Za-z]+)\(/g)]
    .map(x => x[1]).sort();
  assert.deepEqual(methods, [
    'CatalogWorkspace', 'InspectSubject', 'ListWorkspaces', 'ValidateWorkspace',
  ]);
  assert.ok(host.includes('WORKSPACE_NOT_ALLOWED'));
  for (const method of methods) {
    assert.ok(bridge.includes('return b.host.' + method + '('), method);
  }
});
