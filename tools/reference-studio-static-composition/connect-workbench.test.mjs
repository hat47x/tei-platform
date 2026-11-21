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
  WORKBENCH_MAIN_CONSUMER_HEAD, WORKBENCH_MAIN_CONSUMER_BLOB,
  WORKBENCH_PR1_HEAD, WORKBENCH_PR1_BLOB,
  WORKBENCH_NATIVE_HEAD, WORKBENCH_NATIVE_MAIN_BLOB,
  WORKBENCH_GENERATED_READER_HEAD, WORKBENCH_GENERATED_READER_MAIN_BLOB,
  WORKBENCH_REQUIRED_FIELDS_READER_HEAD, WORKBENCH_PIN_INSPECTION_TESTS_HEAD,
  WORKBENCH_REQUIRED_PREVIEW_HEAD, WORKBENCH_REQUIRED_PREVIEW_MAIN_BLOB,
  WORKBENCH_REQUIRED_EVIDENCE_HEAD, WORKBENCH_REQUIRED_EVIDENCE_MAIN_BLOB,
  WORKBENCH_REQUIRED_EVIDENCE_RELOCATION_HEAD,
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

test('historical Workbench pins and exact PR1 candidate reject unknown revisions', () => {
  assert.deepEqual(WORKBENCH_PINS.map(pin => pin.head), [
    '9a290dc61f585dfbf292e9032e5c5bbbad04351a',
    'c55ad2285268533f5f1c90fbc9fe766bae57f009',
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_PR1_HEAD,
    WORKBENCH_NATIVE_HEAD,
    WORKBENCH_GENERATED_READER_HEAD,
    WORKBENCH_REQUIRED_FIELDS_READER_HEAD,
    WORKBENCH_PIN_INSPECTION_TESTS_HEAD,
    WORKBENCH_REQUIRED_PREVIEW_HEAD,
    WORKBENCH_REQUIRED_EVIDENCE_HEAD,
    WORKBENCH_REQUIRED_EVIDENCE_RELOCATION_HEAD,
  ]);
  assert.deepEqual(WORKBENCH_PINS.map(pin => pin.mainBlob), [
    '2a65a9f70245068bf2455248d98d280b382b01e4',
    'cc2644e5a9167dbc3a0b5368ff270276a2ba1b67',
    WORKBENCH_MAIN_CONSUMER_BLOB,
    WORKBENCH_PR1_BLOB,
    WORKBENCH_NATIVE_MAIN_BLOB,
    WORKBENCH_GENERATED_READER_MAIN_BLOB,
    WORKBENCH_GENERATED_READER_MAIN_BLOB,
    WORKBENCH_GENERATED_READER_MAIN_BLOB,
    WORKBENCH_REQUIRED_PREVIEW_MAIN_BLOB,
    WORKBENCH_REQUIRED_EVIDENCE_MAIN_BLOB,
    WORKBENCH_REQUIRED_EVIDENCE_MAIN_BLOB,
  ]);
  assert.deepEqual(WORKBENCH_PINS.map(pin => pin.commandHead), [
    '26f47772ed49f5c4a9e1c50026c335783bc5e153',
    '01f5022240fd86c6ac38c1107310e299a1615b48',
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_MAIN_CONSUMER_HEAD,
    WORKBENCH_MAIN_CONSUMER_HEAD,
  ]);
  assert.deepEqual(WORKBENCH_PINS.map(pin => pin.commandBlob), [
    '4e0d5484f82944d5b7d3b0b8431f3a9a1eaa7081',
    '6c6ee81288c9857ca2f1c282639ba49f059f5535',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
  ]);
  assert.equal(WORKBENCH_PINS[1].responsesBlob,
    '9fa305e227bee96b66002362121db3878e92327d');
  assert.equal(WORKBENCH_HEAD, WORKBENCH_PINS[0].head);
  assert.equal(WORKBENCH_MAIN_BLOB, WORKBENCH_PINS[0].mainBlob);
  assert.equal(WORKBENCH_CURRENT_HEAD, WORKBENCH_PINS[1].head);
  assert.equal(WORKBENCH_CURRENT_MAIN_BLOB, WORKBENCH_PINS[1].mainBlob);
  assert.equal(WORKBENCH_MAIN_CONSUMER_BLOB, WORKBENCH_PINS[2].mainBlob);
  assert.equal(WORKBENCH_PR1_BLOB, WORKBENCH_PINS[3].mainBlob);
  assert.equal(WORKBENCH_NATIVE_MAIN_BLOB, WORKBENCH_PINS[4].mainBlob);
  assert.equal(WORKBENCH_GENERATED_READER_MAIN_BLOB, WORKBENCH_PINS[5].mainBlob);
  assert.notEqual(WORKBENCH_PINS[4].mainBlob, WORKBENCH_PINS[5].mainBlob);
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

test('new generated Inspector closure is qualified separately from historical native closure', async () => {
  const historical = WORKBENCH_PINS.find(value => value.head === WORKBENCH_NATIVE_HEAD);
  // The current source closure is qualified by the newest pin for this main.ts.
  const pin = WORKBENCH_PINS.find(value =>
    value.head === WORKBENCH_REQUIRED_EVIDENCE_RELOCATION_HEAD);
  assert.ok(historical);
  assert.ok(pin);
  assert.equal(historical.mainBlob, WORKBENCH_NATIVE_MAIN_BLOB);
  assert.equal(pin.mainBlob, WORKBENCH_REQUIRED_EVIDENCE_MAIN_BLOB);
  assert.notEqual(pin.mainBlob, WORKBENCH_GENERATED_READER_MAIN_BLOB);
  assert.notEqual(pin.mainBlob, WORKBENCH_PR1_BLOB);
  const root = new URL('../../', import.meta.url);
  const source = await readFile(new URL('reference/tei-studio-workbench/main.ts', root), 'utf8');
  const commandSource = await readFile(new URL('reference/tei-studio-command/command.ts', root), 'utf8');
  const responsesSource = await readFile(new URL('reference/tei-studio-workbench/responses.ts', root), 'utf8');
  const graphSource = await readFile(new URL('reference/tei-studio-graph/graph.ts', root), 'utf8');
  const closureSources = Object.fromEntries(await Promise.all(
    Object.keys(pin.closureBlobs).map(async path =>
      [path, await readFile(new URL(path, root), 'utf8')])));
  assert.equal(pinnedWorkbenchRevision(source), pin);
  assert.throws(() => pinnedWorkbenchRevision(source + '\n// unqualified change\n'),
    /workbench_revision_mismatch/);
  // The old native pin must still reject the newly modified presentation.
  const oldClosure = Object.fromEntries(Object.keys(historical.closureBlobs).map(path => [path, closureSources[path]]));
  // Isolate the changed presentation source: the earlier test command is
  // restored in this local fixture only, never by rewriting a historical pin.
  const packagePath = 'reference/tei-studio-workbench/package.json';
  oldClosure[packagePath] = oldClosure[packagePath].replace(
    ' inspector-selection.test.mjs pin-inspection.test.mjs"',
    ' inspector-selection.test.mjs"');
  assert.equal(gitBlobSHA(oldClosure[packagePath]), historical.closureBlobs[packagePath]);
  assert.throws(() => inspectPinnedDependencies(historical, commandSource, responsesSource, graphSource, oldClosure),
    /workbench_closure_revision_mismatch:reference\/tei-studio-workbench\/presentation.ts/);
  // The required-fields pin must reject the new test script source revision.
  const requiredFieldsPin = WORKBENCH_PINS.find(value =>
    value.head === WORKBENCH_REQUIRED_FIELDS_READER_HEAD);
  const requiredFieldsClosure = Object.fromEntries(
    Object.keys(requiredFieldsPin.closureBlobs).map(path => [path, closureSources[path]]));
  assert.throws(() => inspectPinnedDependencies(
    requiredFieldsPin, commandSource, responsesSource, graphSource, requiredFieldsClosure),
    /workbench_closure_revision_mismatch:reference\/tei-studio-workbench\/package.json/);
  // Evidence display was qualified before this extra regression. Keep its
  // historical pin immutable: it must reject the new test Source.
  const previous = WORKBENCH_PINS.find(value =>
    value.head === WORKBENCH_REQUIRED_EVIDENCE_HEAD);
  assert.ok(previous);
  const oldSources = Object.fromEntries(
    Object.keys(previous.closureBlobs).map(path => [path, closureSources[path]]));
  assert.throws(() => inspectPinnedDependencies(
    previous, commandSource, responsesSource, graphSource, oldSources),
    /workbench_closure_revision_mismatch:reference\/tei-studio-workbench\/source-locations\.test\.mjs/);
  assert.equal(inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource, closureSources), true);
  // The prior qualified closure cannot silently accept the new editable-draft surface.
  const prior = WORKBENCH_PINS.find(value => value.head === WORKBENCH_PIN_INSPECTION_TESTS_HEAD);
  assert.notEqual(prior.mainBlob, pin.mainBlob);
  // The earlier generated-Reader pin keeps its exact closure: the part
  // regenerated from the changed contract is not qualified by it.
  const earlier = WORKBENCH_PINS.find(value => value.head === WORKBENCH_GENERATED_READER_HEAD);
  assert.equal(earlier.mainBlob, WORKBENCH_GENERATED_READER_MAIN_BLOB);
  assert.notEqual(earlier.mainBlob, pin.mainBlob);
  const earlierClosure = Object.fromEntries(
    Object.keys(earlier.closureBlobs).map(path => [path, closureSources[path]]));
  assert.throws(() => inspectPinnedDependencies(earlier, commandSource, responsesSource, graphSource, earlierClosure),
    /workbench_closure_revision_mismatch:reference\/tei-studio-workbench\/inspection-reader.generated.ts/);
  assert.throws(() => inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource),
    /workbench_closure_missing_or_extra/);
  const missing = {...closureSources};
  delete missing['reference/tei-studio-workbench/inspector-selection.ts'];
  assert.throws(() => inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource, missing),
    /workbench_closure_missing_or_extra/);
  const extra = {...closureSources, 'unqualified.ts': 'export {}'};
  assert.throws(() => inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource, extra),
    /workbench_closure_missing_or_extra/);
  for (const path of Object.keys(pin.closureBlobs)) {
    const changed = {...closureSources, [path]: closureSources[path] + '\\n// changed\\n'};
    assert.throws(() => inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource, changed),
      /workbench_closure_revision_mismatch/, path);
  }
  assert.throws(() => inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource + '\n', closureSources),
    /graph_revision_mismatch/);
  const connected = connectWorkbenchSource(source);
  assert.equal((connected.match(/ui.generated.mjs/g) ?? []).length, 1);
  assert.ok(connected.includes('mountCommandPanel('));
  assert.ok(connected.includes('mountReadonlyGraph('));
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
