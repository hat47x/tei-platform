// Snapshot-only adapter for #1849's exact single workbench entry.
// Does not edit the #Work branch or create a second UI registry/Host client.
import { createHash } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export const WORKBENCH_HEAD = '9a290dc61f585dfbf292e9032e5c5bbbad04351a';
export const WORKBENCH_MAIN_BLOB = '2a65a9f70245068bf2455248d98d280b382b01e4';
const graphImport = 'import { mountReadonlyGraph } from "../tei-studio-graph/graph.ts";\n';
const commandImport = 'import { mountCommandPanel } from "../tei-studio-command/command.ts";\n';
const generatedImport = 'import { mountReadonlyGraph, mountCommandPanel } from "../../tools/reference-studio-static-composition/generated/ui.generated.mjs";\n';
const hash = data => createHash('sha256').update(data).digest('hex');

export function gitBlobSHA(text) {
  return createHash('sha1').update('blob ' + Buffer.byteLength(text) + '\0').update(text).digest('hex');
}
export function connectWorkbenchSource(source, { pin = true } = {}) {
  if (pin && gitBlobSHA(source) !== WORKBENCH_MAIN_BLOB)
    throw new Error('workbench_revision_mismatch');
  for (const entry of [graphImport, commandImport]) {
    if (source.split(entry).length !== 2) throw new Error('workbench_import_missing_or_duplicate');
  }
  if (source.includes('ui.generated.mjs')) throw new Error('workbench_already_connected');
  // Only these two import lines differ. All four Host operations, the
  // allowlisted workspace selector, reply envelope and lifetimes are untouched.
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
  inspectGeneratedAssets(moduleSource, declarations, JSON.parse(rawMeta));
  const connected = connectWorkbenchSource(original);
  let existing;
  try { existing = await readFile(outputPath, 'utf8'); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  if (existing !== undefined && existing !== connected)
    throw new Error('derived_workbench_output_conflict');
  if (existing === undefined) await writeFile(outputPath, connected, { flag: 'wx' });
  console.log('workbench_adapter_pass', WORKBENCH_HEAD, hash(connected));
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main().catch(error => { console.error(error); process.exitCode = 1; });
}
