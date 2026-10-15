import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { composeUI } from './compose-ui.mjs';
import {
  connectWorkbenchSource, gitBlobSHA, inspectGeneratedAssets,
  inspectWorkbenchImports, inspectPinnedDependencies, pinnedWorkbenchRevision, WORKBENCH_PINS,
  WORKBENCH_HEAD, WORKBENCH_MAIN_BLOB,
  WORKBENCH_CURRENT_HEAD, WORKBENCH_CURRENT_MAIN_BLOB,
} from './connect-workbench.mjs';

const graph = 'import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";\n';
const command = 'import { mountCommandPanel } from "../tei-studio-command/command.ts";\n';
const workbench = 'import { createReadOnlyTextEditor } from "../tei-studio-editor/editor.ts";\n' +
  graph + command +
  'const client = createReadOnlyClient(Call.ByName);\n' +
  'client.listWorkspaces();\nclient.catalogWorkspace(id, "");\n' +
  'client.validateWorkspace(id);\nclient.inspectSubject(id, kind, subject);\n' +
  'const graphPort = mountReadonlyGraph(root, input, options);\n' +
  'const panel = mountCommandPanel(root, props);\n';

test('both exact #1849 workbench pins are retained without any bypass', () => {
  assert.deepEqual(WORKBENCH_PINS.map(pin => pin.head), [
    '9a290dc61f585dfbf292e9032e5c5bbbad04351a',
    'c55ad2285268533f5f1c90fbc9fe766bae57f009',
  ]);
  assert.deepEqual(WORKBENCH_PINS.map(pin => pin.mainBlob), [
    '2a65a9f70245068bf2455248d98d280b382b01e4',
    'cc2644e5a9167dbc3a0b5368ff270276a2ba1b67',
  ]);
  assert.deepEqual(WORKBENCH_PINS.map(pin => pin.commandHead), [
    '26f47772ed49f5c4a9e1c50026c335783bc5e153',
    '01f5022240fd86c6ac38c1107310e299a1615b48',
  ]);
  assert.deepEqual(WORKBENCH_PINS.map(pin => pin.commandBlob), [
    '4e0d5484f82944d5b7d3b0b8431f3a9a1eaa7081',
    '6c6ee81288c9857ca2f1c282639ba49f059f5535',
  ]);
  assert.equal(WORKBENCH_PINS[1].responsesBlob,
    '9fa305e227bee96b66002362121db3878e92327d');
  assert.equal(WORKBENCH_HEAD, WORKBENCH_PINS[0].head);
  assert.equal(WORKBENCH_MAIN_BLOB, WORKBENCH_PINS[0].mainBlob);
  assert.equal(WORKBENCH_CURRENT_HEAD, WORKBENCH_PINS[1].head);
  assert.equal(WORKBENCH_CURRENT_MAIN_BLOB, WORKBENCH_PINS[1].mainBlob);
  assert.ok(Object.isFrozen(WORKBENCH_PINS));
  assert.ok(WORKBENCH_PINS.every(Object.isFrozen));
  assert.notEqual(WORKBENCH_PINS[0].mainBlob, WORKBENCH_PINS[1].mainBlob);
  assert.throws(() => connectWorkbenchSource(workbench), /workbench_revision_mismatch/);
  assert.throws(() => pinnedWorkbenchRevision(workbench), /workbench_revision_mismatch/);
  assert.throws(() => connectWorkbenchSource(workbench, {pin: false}),
    /workbench_pin_override_forbidden/);
  assert.throws(() => connectWorkbenchSource(workbench, {pin: true}),
    /workbench_pin_override_forbidden/);
});

test('Command revision and current response-scope are mandatory source pins', () => {
  assert.throws(() => inspectPinnedDependencies({...WORKBENCH_PINS[0]}, 'fake'),
    /workbench_revision_mismatch/);
  assert.throws(() => inspectPinnedDependencies(WORKBENCH_PINS[0], 'fake'),
    /command_revision_mismatch/);
  assert.throws(() => inspectPinnedDependencies(WORKBENCH_PINS[1], 'fake'),
    /command_revision_mismatch/);
});

test('untrusted duplicate, missing or already-generated imports fail shape validation', () => {
  assert.equal(inspectWorkbenchImports(workbench), true);
  assert.throws(() => inspectWorkbenchImports(workbench.replace(graph, '')),
    /workbench_import_missing_or_duplicate/);
  assert.throws(() => inspectWorkbenchImports(workbench.replace(command, command + command)),
    /workbench_import_missing_or_duplicate/);
  const generated = 'import { mountReadonlyGraph, mountCommandPanel } from ' +
    '"../../tools/reference-studio-static-composition/generated/ui.generated.mjs";\n';
  assert.throws(() => inspectWorkbenchImports(workbench.replace(graph, generated).replace(command, '')),
    /workbench_already_connected/);
});

test('generated module, declaration and trace digests must agree', async () => {
  const raw = await readFile(new URL('./fixture.json', import.meta.url));
  const {source, declarations, provenance} = composeUI(raw);
  assert.equal(inspectGeneratedAssets(source, declarations, provenance), true);
  assert.throws(() => inspectGeneratedAssets(source + '\n', declarations, provenance),
    /generated_asset_digest_mismatch/);
  assert.throws(() => inspectGeneratedAssets(source, declarations.slice(1), provenance),
    /generated_asset_digest_mismatch/);
  const wrong = {...provenance, uiContributions: [{slot: 'graph'}]};
  assert.throws(() => inspectGeneratedAssets(source, declarations, wrong),
    /invalid_generated_contributions/);
  const tampered = source.replace('export const mountReadonlyGraph', 'export const missing');
  assert.throws(() => inspectGeneratedAssets(tampered, declarations,
    {...provenance, generatedSha256: createHash('sha256').update(tampered).digest('hex')}),
    /generated_mount_export_missing/);
});

// The checkout may contain a different Workbench revision. Use only this
// explicitly pinned fixture instead of inferring a snapshot from the caller.
test('fixed #1849 Workbench fixture and its dependencies match exact blobs', async () => {
  const snapshot = new URL('./testdata/poc-b-workbench-snapshot/', import.meta.url);
  const source = await readFile(
    new URL('reference/tei-studio-workbench/main.ts', snapshot), 'utf8');
  const matched = pinnedWorkbenchRevision(source);
  assert.equal(matched.head, WORKBENCH_CURRENT_HEAD);
  assert.equal(gitBlobSHA(source), matched.mainBlob);
  const commandSource = await readFile(
    new URL('reference/tei-studio-command/command.ts', snapshot), 'utf8');
  const responsesSource = matched.responsesBlob
    ? await readFile(new URL('reference/tei-studio-workbench/responses.ts', snapshot), 'utf8')
    : undefined;
  assert.equal(inspectPinnedDependencies(matched, commandSource, responsesSource), true);
  assert.throws(() => inspectPinnedDependencies(matched, commandSource + '\\n', responsesSource),
    /command_revision_mismatch/);
  if (matched.responsesBlob) {
    assert.throws(() => inspectPinnedDependencies(matched, commandSource, 'stale response adapter'),
      /responses_revision_mismatch/);
  }
  const output = connectWorkbenchSource(source);
  const generated = 'import { mountReadonlyGraph, mountCommandPanel } from ' +
    '"../../tools/reference-studio-static-composition/generated/ui.generated.mjs";\n';
  assert.equal(output, source.replace(graph, generated).replace(command, ''));
  assert.equal((output.match(/ui.generated.mjs/g) ?? []).length, 1);
  for (const symbol of ['createReadOnlyClient', 'listWorkspaces', 'catalogWorkspace',
    'validateWorkspace', 'inspectSubject', 'workspaceEpoch', 'inspectionEpoch',
    'mountReadonlyGraph(', 'mountCommandPanel(']) {
    assert.ok(output.includes(symbol), symbol);
  }
  if (matched.head === WORKBENCH_CURRENT_HEAD) {
    for (const guard of ['createResponseScope', 'responses.reset()', 'responses.dispose()',
      'panel.beginContext()', 'setDiagnosticsForContext']) {
      assert.ok(output.includes(guard), guard);
    }
  }
  assert.throws(() => connectWorkbenchSource(output),
    /workbench_revision_mismatch/);
});
