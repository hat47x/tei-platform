// Snapshot-only adapter for #1849's exact single workbench entry.
// Does not edit the #Work branch or create a second UI registry/Host client.
import { createHash } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

// Exact source allowlist. Both entries remain usable in separate, non-merge
// snapshots. The new Command source is supplied by the snapshot, never loaded
// by this adapter or selected from a client-controlled input.
export const WORKBENCH_HEAD = '9a290dc61f585dfbf292e9032e5c5bbbad04351a';
export const WORKBENCH_MAIN_BLOB = '2a65a9f70245068bf2455248d98d280b382b01e4';
export const WORKBENCH_CURRENT_HEAD = 'c55ad2285268533f5f1c90fbc9fe766bae57f009';
export const WORKBENCH_CURRENT_MAIN_BLOB = 'cc2644e5a9167dbc3a0b5368ff270276a2ba1b67';
// Current-main consumer pin for the generated Workbench browser gate.
export const WORKBENCH_MAIN_CONSUMER_HEAD = 'a97046d513eabdada86542e95553eaf450173c3e';
export const WORKBENCH_MAIN_CONSUMER_BLOB = '11815ab583dbcf6bb28baab83d7854389c1043be';
export const WORKBENCH_PINS = Object.freeze([
  Object.freeze({head: WORKBENCH_HEAD, mainBlob: WORKBENCH_MAIN_BLOB,
    commandHead: '26f47772ed49f5c4a9e1c50026c335783bc5e153',
    commandBlob: '4e0d5484f82944d5b7d3b0b8431f3a9a1eaa7081'}),
  Object.freeze({head: WORKBENCH_CURRENT_HEAD, mainBlob: WORKBENCH_CURRENT_MAIN_BLOB,
    commandHead: '01f5022240fd86c6ac38c1107310e299a1615b48',
    commandBlob: '6c6ee81288c9857ca2f1c282639ba49f059f5535',
    responsesBlob: '9fa305e227bee96b66002362121db3878e92327d'}),
  Object.freeze({head: WORKBENCH_MAIN_CONSUMER_HEAD, mainBlob: WORKBENCH_MAIN_CONSUMER_BLOB,
    commandHead: 'a97046d513eabdada86542e95553eaf450173c3e',
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6'}),
]);
const graphImport = 'import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";\n';
const commandImport = 'import { mountCommandPanel } from "../tei-studio-command/command.ts";\n';
const generatedImport = 'import { mountReadonlyGraph, mountCommandPanel } from "../../tools/reference-studio-static-composition/generated/ui.generated.mjs";\n';
const hash = data => createHash('sha256').update(data).digest('hex');

export function gitBlobSHA(text) {
  // Git stores normalized LF text even when a Windows checkout materializes CRLF.
  const normalized = text.replaceAll('\r\n', '\n');
  return createHash('sha1').update('blob ' + Buffer.byteLength(normalized) + '\0').update(normalized).digest('hex');
}
export function inspectWorkbenchImports(source) {
  if (source.includes('ui.generated.mjs')) throw new Error('workbench_already_connected');
  for (const entry of [graphImport, commandImport]) {
    if (source.split(entry).length !== 2) throw new Error('workbench_import_missing_or_duplicate');
  }
  return true;
}
export function pinnedWorkbenchRevision(source) {
  const blob = gitBlobSHA(source);
  const matched = WORKBENCH_PINS.find(pin => pin.mainBlob === blob);
  if (!matched) throw new Error('workbench_revision_mismatch');
  return matched;
}
// Verify actual source assets in the snapshot, not only their documented
// commit labels. The new Workbench requires Command's context-aware API.
export function inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource) {
  if (!WORKBENCH_PINS.includes(pin)) throw new Error('workbench_revision_mismatch');
  if (gitBlobSHA(commandSource) !== pin.commandBlob)
    throw new Error('command_revision_mismatch');
  if (pin.responsesBlob && gitBlobSHA(responsesSource ?? '') !== pin.responsesBlob)
    throw new Error('responses_revision_mismatch');
  if (pin.graphBlob && gitBlobSHA(graphSource ?? '') !== pin.graphBlob)
    throw new Error('graph_revision_mismatch');
  return true;
}
export function connectWorkbenchSource(source) {
  // Do not accept pin:false or any other request to skip source validation.
  if (arguments.length !== 1) throw new Error('workbench_pin_override_forbidden');
  source = source.replaceAll('\r\n', '\n');
  pinnedWorkbenchRevision(source);
  inspectWorkbenchImports(source);
  // Only these two import lines differ. The read-only Host envelope,
  // workspace allowlist and response-generation guards stay untouched.
  return source.replace(graphImport, generatedImport).replace(commandImport, '');
}
export function inspectGeneratedAssets(moduleSource, declarations, provenance) {
  if (provenance.generatedSha256 !== hash(moduleSource) ||
      provenance.declarationsSha256 !== hash(declarations))
    throw new Error('generated_asset_digest_mismatch');
  if (provenance.reference !== 'tei-studio-poc-b-reference-v0' ||
      provenance.uiContributions?.map(v => v.slot).join(',') !== 'commands,graph')
    throw new Error('invalid_generated_contributions');
  for (const symbol of ['mountCommandPanel', 'mountReadonlyGraph']) {
    if (!moduleSource.includes('export const ' + symbol + ' = ') ||
        !declarations.includes('export { ' + symbol + ' } from '))
      throw new Error('generated_mount_export_missing');
  }
  return true;
}

async function main() {
  const mainPath = process.argv[2];
  if (!mainPath || process.argv.length !== 3)
    throw new Error('usage: node connect-workbench.mjs /SNAPSHOT/reference/tei-studio-workbench/main.ts');
  const sourcePath = resolve(mainPath);
  if (sourcePath.split(/[\\/]/).at(-1) !== 'main.ts')
    throw new Error('expected_original_workbench_main');
  const outputPath = join(dirname(sourcePath), 'main.generated.ts');
  const generatedDir = resolve(dirname(sourcePath), '../../tools/reference-studio-static-composition/generated');
  const [original, moduleSource, declarations, rawMeta] = await Promise.all([
    readFile(sourcePath, 'utf8'),
    readFile(join(generatedDir, 'ui.generated.mjs'), 'utf8'),
    readFile(join(generatedDir, 'ui.generated.d.mts'), 'utf8'),
    readFile(join(generatedDir, 'ui.provenance.json'), 'utf8'),
  ]);
  const pin = pinnedWorkbenchRevision(original);
  const commandSource = await readFile(
    resolve(dirname(sourcePath), '../tei-studio-command/command.ts'), 'utf8');
  const responsesSource = pin.responsesBlob
    ? await readFile(join(dirname(sourcePath), 'responses.ts'), 'utf8')
    : undefined;
  const graphSource = pin.graphBlob
    ? await readFile(resolve(dirname(sourcePath), '../tei-studio-graph/graph.ts'), 'utf8')
    : undefined;
  inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource);
  inspectGeneratedAssets(moduleSource, declarations, JSON.parse(rawMeta));
  const connected = connectWorkbenchSource(original);
  let existing;
  try { existing = await readFile(outputPath, 'utf8'); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  if (existing !== undefined && existing !== connected)
    throw new Error('derived_workbench_output_conflict');
  if (existing === undefined) await writeFile(outputPath, connected, { flag: 'wx' });
  console.log('workbench_adapter_pass', pin.head, hash(connected));
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main().catch(error => { console.error(error); process.exitCode = 1; });
}
