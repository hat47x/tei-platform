import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { composeUI } from './compose-ui.mjs';
import {
  connectWorkbenchSource, gitBlobSHA, inspectGeneratedAssets,
  WORKBENCH_HEAD, WORKBENCH_MAIN_BLOB,
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

test('only the two workbench imports change; Host and mount usage remain byte-for-byte', () => {
  const next = connectWorkbenchSource(workbench, {pin: false});
  const generated = 'import { mountReadonlyGraph, mountCommandPanel } from ' +
    '"../../tools/reference-studio-static-composition/generated/ui.generated.mjs";\n';
  assert.equal(next, workbench.replace(graph, generated).replace(command, ''));
  assert.equal((next.match(/ui.generated.mjs/g) ?? []).length, 1);
  for (const symbol of ['createReadOnlyClient', 'listWorkspaces', 'catalogWorkspace',
    'validateWorkspace', 'inspectSubject', 'mountReadonlyGraph(root',
    'mountCommandPanel(root']) assert.ok(next.includes(symbol), symbol);
});

test('missing/duplicate imports and unexpected workbench revision fail closed', () => {
  assert.throws(() => connectWorkbenchSource(workbench.replace(graph, ''), {pin: false}),
    /workbench_import_missing_or_duplicate/);
  assert.throws(() => connectWorkbenchSource(workbench.replace(command, command + command),
    {pin: false}), /workbench_import_missing_or_duplicate/);
  assert.throws(() => connectWorkbenchSource(workbench.replace(graph, ''),
    {pin: true}), /workbench_revision_mismatch/);
  const connected = connectWorkbenchSource(workbench, {pin: false});
  assert.throws(() => connectWorkbenchSource(connected, {pin: false}),
    /workbench_import_missing_or_duplicate|workbench_already_connected/);
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

// #1849 is a separate Draft. When its exact source is present in a non-merge
// snapshot, assert actual blob identity; do not silently test a different head.
test('fixed #1849 real source is accepted when staged', async t => {
  const pathname = new URL('../../reference/tei-studio-workbench/main.ts', import.meta.url);
  let source;
  try { source = await readFile(pathname, 'utf8'); }
  catch (error) {
    if (error.code !== 'ENOENT') throw error;
    t.skip('exact #1849 source absent on isolated #1833 branch');
    return;
  }
  assert.equal(WORKBENCH_HEAD, '9a290dc61f585dfbf292e9032e5c5bbbad04351a');
  assert.equal(gitBlobSHA(source), WORKBENCH_MAIN_BLOB);
  const output = connectWorkbenchSource(source);
  assert.ok(output.includes('ui.generated.mjs'));
  assert.ok(output.includes('createReadOnlyClient'));
  assert.ok(output.includes('workspaceEpoch'));
  assert.ok(output.includes('inspectionEpoch'));
});
