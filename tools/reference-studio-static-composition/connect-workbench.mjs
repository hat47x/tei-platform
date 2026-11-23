// Snapshot-only adapter for #1849's exact single workbench entry.
// Does not edit the #Work branch or create a second UI registry/Host client.
import { createHash } from 'node:crypto';
import { readFile, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

// Exact source allowlist. Historical snapshots remain pinned unchanged;
// the PR1 candidate is an additional, explicitly qualified snapshot.
// Command/Graph dependencies must match the selected snapshot exactly.
export const WORKBENCH_HEAD = '9a290dc61f585dfbf292e9032e5c5bbbad04351a';
export const WORKBENCH_MAIN_BLOB = '2a65a9f70245068bf2455248d98d280b382b01e4';
export const WORKBENCH_CURRENT_HEAD = 'c55ad2285268533f5f1c90fbc9fe766bae57f009';
export const WORKBENCH_CURRENT_MAIN_BLOB = 'cc2644e5a9167dbc3a0b5368ff270276a2ba1b67';
// Current-main consumer pin for the generated Workbench browser gate.
export const WORKBENCH_MAIN_CONSUMER_HEAD = 'a97046d513eabdada86542e95553eaf450173c3e';
export const WORKBENCH_MAIN_CONSUMER_BLOB = '11815ab583dbcf6bb28baab83d7854389c1043be';
// #1124 PR1 candidate: retain historical evidence and reject unknown revisions.
export const WORKBENCH_PR1_HEAD = 'dc7b44750279d8f9e4c4e4b797c054b865021c1a';
export const WORKBENCH_PR1_BLOB = '9582c85db4c27c1f89d8594f5b212984d755c9c0';
// Exact native Workbench revision verified by the ordinary WebView2 consumer
// (#5025, run 43706898173). Generated PoC B is independently requalified;
// the source closure is not inferred from whichever main branch is checked out.
export const WORKBENCH_NATIVE_HEAD = '646dc64f746377c73c28d5112dc5c0793af4018c';
export const WORKBENCH_NATIVE_MAIN_BLOB = 'bdcceb73e6113c15162637b3ef5909252459a9fe';
const WORKBENCH_NATIVE_CLOSURE = Object.freeze({
  'reference/tei-studio-editor/editor.ts': '146c4fb631a55913a3f0e0aeb82043096f5b118e',
  'reference/tei-studio-editor/codemirror.ts': '61dddbc2df6708830e86d344782dec7207463bec',
  'reference/tei-studio-editor/codemirror-coordinates.ts': '686a54a685e0078975623e2ed130b2c33f33bc89',
  'cmd/tei-studio/assets/client.mjs': '2153f2ad67387032487763d68dec23aa016776fe',
  'reference/tei-studio-workbench/presentation.ts': '034e983aa9ba85009cfacbd47d67239b12807c46',
  'reference/tei-studio-workbench/inspector-selection.ts': 'e6b2d53d6bcf69c04bb4976ea33a483407f856d8',
  'reference/tei-studio-workbench/pin-reader.ts': 'bf8e1af9cd583fe115d6089ab6fe6b6c119c8c2b',
  'reference/tei-studio-workbench/pin-state.ts': 'cec97c75f92b406435eac0ac60cc679ec5f36aa3',
  'reference/tei-studio-workbench/keyboard.ts': 'fdf5cc9d619fcf3a187a19342af6edba925ddb61',
  'reference/tei-studio-workbench/package.json': '55c30d71161b0318f33313389a6455348916a771',
  'reference/tei-studio-workbench/package-lock.json': '0dc26032126a8a3ab5ed874d3d47d9cd550aa475',
});
// New source revision: generated Inspector is imported by the production entry.
// This is a candidate closure until the exact-head Windows Consumer qualifies it;
// historical native and earlier pins remain byte-for-byte unchanged.
export const WORKBENCH_GENERATED_READER_HEAD = '648cfa983639b6e716b9d47dbf4d154c8f38e5fb';
export const WORKBENCH_GENERATED_READER_MAIN_BLOB = 'b9927b2d4b2b18b13cd905fc0bdcc03f656d0df6';
const WORKBENCH_GENERATED_READER_CLOSURE = Object.freeze({
  ...WORKBENCH_NATIVE_CLOSURE,
  'reference/tei-studio-workbench/presentation.ts': 'a31dc45024725a529a6956d896ffd67ebdd2dd84',
  'reference/tei-studio-workbench/inspection-reader.generated.ts': '5ae1d69a41cc71300d322ee95909640c70d2cb9e',
  'reference/tei-studio-workbench/inspection-limit-labels.ts': '0584b591ba6a48d72e355d2cf67b92abc5b040fe',
});
// Same main.ts, new generated Reader: the contract alone added a required
// response field (#4583). The earlier pin keeps its exact closure and now
// rejects this revision; this pin qualifies only the new generated part.
export const WORKBENCH_REQUIRED_FIELDS_READER_HEAD = '1d442ce88c4b51240c53d92331064e8c8a039d1f';
const WORKBENCH_REQUIRED_FIELDS_READER_CLOSURE = Object.freeze({
  ...WORKBENCH_GENERATED_READER_CLOSURE,
  'reference/tei-studio-workbench/inspection-reader.generated.ts': '2ea1ab908931101db1c439302235d57a9f18654e',
});
// Enabling Pin Inspector regression tests changes package.json (the test
// command), which is part of the pinned build/source closure. Preserve every
// historical pin and qualify this new source revision explicitly. The head
// identifies the exact pre-requalification commit whose bytes are pinned.
export const WORKBENCH_PIN_INSPECTION_TESTS_HEAD = '093342e158795514c52e293cfc9abaef99108206';
const WORKBENCH_PIN_INSPECTION_TESTS_CLOSURE = Object.freeze({
  ...WORKBENCH_REQUIRED_FIELDS_READER_CLOSURE,
  'reference/tei-studio-workbench/package.json': 'd48621814bdc690e6da2cfb8a6a26a68dc90653a',
});
// Qualified, explicit new Workbench revision: preserve historical pins and
// bind the new Source-backed draft preview to its exact source, UI assets
// and test entry. This is a source closure identity, not a Writer Grant.
export const WORKBENCH_REQUIRED_PREVIEW_HEAD = '58251bcf61938b585287450ce7949b8ea855b480';
export const WORKBENCH_REQUIRED_PREVIEW_MAIN_BLOB = '557bb1f46755a0233324406b23db942af36e9e26';
const WORKBENCH_REQUIRED_PREVIEW_CLOSURE = Object.freeze({
  ...WORKBENCH_PIN_INSPECTION_TESTS_CLOSURE,
  'reference/tei-studio-workbench/source-locations.ts': '431c11efb09c1a2aa37c5a086501cf611fcc0426',
  'reference/tei-studio-workbench/source-locations.test.mjs': '5c776d20b557cf345c5ec17eddf302acdaa4c907',
  'reference/tei-studio-workbench/index.html': 'a9baa5018f83161439c99eacf4f41e2b5004d71a',
  'reference/tei-studio-workbench/style.css': 'b990387707078f8b76e24cab0719c107c176a50d',
});
// Source-evidence display uses a separately pinned Workbench revision; keep
// all earlier browser source and test closures immutable.
export const WORKBENCH_REQUIRED_EVIDENCE_HEAD = '4714f206951c9cad06593d20b312add18fdab232';
export const WORKBENCH_REQUIRED_EVIDENCE_MAIN_BLOB = 'e62ad5ad7e68392f23024ebfdf9c99eb2b05c506';
const WORKBENCH_REQUIRED_EVIDENCE_CLOSURE = Object.freeze({
  ...WORKBENCH_REQUIRED_PREVIEW_CLOSURE,
  'reference/tei-studio-workbench/source-locations.test.mjs': '031c1bad1b0f8a45eec3166541adabf5a630f836',
});

// Additive Source relocation test pin; the prior evidence-display candidate
// remains immutable. This is not a Source Writer or an accepted Edit Grant.
export const WORKBENCH_REQUIRED_EVIDENCE_SOURCE_RELOCATION_HEAD = 'e7509fcce8be6c63506056ddbd6d5eb7c415bd90';
const WORKBENCH_REQUIRED_EVIDENCE_SOURCE_RELOCATION_CLOSURE = Object.freeze({
  ...WORKBENCH_REQUIRED_EVIDENCE_CLOSURE,
  'reference/tei-studio-workbench/source-locations.test.mjs': '116de8ea7a02f80d2ba9fa6733ecaf98723d4be3',
});


// New candidate: foreground opening of an inspected pinned Subject is
// explicit user navigation. Preserve prior qualified Source+closure pins.
// This candidate identifies the exact current-main source snapshot, not a
// Writer Grant, runtime deployment, or Windows WebView2 qualification.
export const WORKBENCH_FOREGROUND_OPEN_HEAD = 'd20f75636d3bc7dc947f1b2d73b4f96399162678';
export const WORKBENCH_FOREGROUND_OPEN_MAIN_BLOB = 'b5cfcd0c6cd714c226cfe58515460890452a6af8';
const WORKBENCH_FOREGROUND_OPEN_CLOSURE = Object.freeze({
  ...WORKBENCH_REQUIRED_EVIDENCE_SOURCE_RELOCATION_CLOSURE,
  'reference/tei-studio-workbench/index.html': 'a5bd1d2aac58e5c607aaa805a975ddcf0891813f',
});

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
  Object.freeze({head: WORKBENCH_PR1_HEAD, mainBlob: WORKBENCH_PR1_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6'}),
  Object.freeze({head: WORKBENCH_NATIVE_HEAD, mainBlob: WORKBENCH_NATIVE_MAIN_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6',
    closureBlobs: WORKBENCH_NATIVE_CLOSURE}),
  Object.freeze({head: WORKBENCH_GENERATED_READER_HEAD,
    mainBlob: WORKBENCH_GENERATED_READER_MAIN_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6',
    closureBlobs: WORKBENCH_GENERATED_READER_CLOSURE}),
  Object.freeze({head: WORKBENCH_REQUIRED_FIELDS_READER_HEAD,
    mainBlob: WORKBENCH_GENERATED_READER_MAIN_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6',
    closureBlobs: WORKBENCH_REQUIRED_FIELDS_READER_CLOSURE}),
  Object.freeze({head: WORKBENCH_PIN_INSPECTION_TESTS_HEAD,
    mainBlob: WORKBENCH_GENERATED_READER_MAIN_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6',
    closureBlobs: WORKBENCH_PIN_INSPECTION_TESTS_CLOSURE}),
  Object.freeze({head: WORKBENCH_REQUIRED_PREVIEW_HEAD,
    mainBlob: WORKBENCH_REQUIRED_PREVIEW_MAIN_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6',
    closureBlobs: WORKBENCH_REQUIRED_PREVIEW_CLOSURE}),
  Object.freeze({head: WORKBENCH_REQUIRED_EVIDENCE_HEAD,
    mainBlob: WORKBENCH_REQUIRED_EVIDENCE_MAIN_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6',
    closureBlobs: WORKBENCH_REQUIRED_EVIDENCE_CLOSURE}),
  Object.freeze({head: WORKBENCH_REQUIRED_EVIDENCE_SOURCE_RELOCATION_HEAD,
    mainBlob: WORKBENCH_REQUIRED_EVIDENCE_MAIN_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6',
    closureBlobs: WORKBENCH_REQUIRED_EVIDENCE_SOURCE_RELOCATION_CLOSURE}),
  Object.freeze({head: WORKBENCH_FOREGROUND_OPEN_HEAD,
    mainBlob: WORKBENCH_FOREGROUND_OPEN_MAIN_BLOB,
    commandHead: WORKBENCH_MAIN_CONSUMER_HEAD,
    commandBlob: 'bb56f3dba6387f7d9bf9604de9b519fb68feb50d',
    responsesBlob: 'e1fa0d6b8e07af1158416d6c6d2469a408f64495',
    graphBlob: '436c7d07c56802dbbc6d1dfeb1251470fca75cf6',
    closureBlobs: WORKBENCH_FOREGROUND_OPEN_CLOSURE}),
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
// This historical Host client is stored with BOM and mixed CRLF/LF bytes in
// Git. Normalizing it would no longer validate the actual committed blob.
function exactGitBlobSHA(text) {
  return createHash('sha1').update('blob ' + Buffer.byteLength(text) + '\0').update(text).digest('hex');
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
  // Several pins may share main.ts when only its closure changed; the newest
  // qualified one is the candidate, and its closure is then checked exactly.
  const matched = WORKBENCH_PINS.findLast(pin => pin.mainBlob === blob);
  if (!matched) throw new Error('workbench_revision_mismatch');
  return matched;
}
// Verify actual source assets in the snapshot, not only their documented
// commit labels. The new Workbench requires Command's context-aware API.
export function inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource, closureSources) {
  if (!WORKBENCH_PINS.includes(pin)) throw new Error('workbench_revision_mismatch');
  if (gitBlobSHA(commandSource) !== pin.commandBlob)
    throw new Error('command_revision_mismatch');
  if (pin.responsesBlob && gitBlobSHA(responsesSource ?? '') !== pin.responsesBlob)
    throw new Error('responses_revision_mismatch');
  if (pin.graphBlob && gitBlobSHA(graphSource ?? '') !== pin.graphBlob)
    throw new Error('graph_revision_mismatch');
  if (pin.closureBlobs) {
    // Missing or unexpected files are not an acceptable replacement for the
    // exact browser source + build-lock closure qualified with this candidate.
    const expected = Object.keys(pin.closureBlobs).sort();
    if (!closureSources ||
        Object.keys(closureSources).sort().join('\\0') !== expected.join('\\0'))
      throw new Error('workbench_closure_missing_or_extra');
    for (const path of expected) {
      // The committed client.mjs has mixed line endings; compare its exact
      // source bytes, not the line-ending-normalized Workbench TS digest.
      const actualBlob = path === 'cmd/tei-studio/assets/client.mjs'
        ? exactGitBlobSHA(closureSources[path]) : gitBlobSHA(closureSources[path]);
      if (typeof closureSources[path] !== 'string' ||
          actualBlob !== pin.closureBlobs[path])
        throw new Error('workbench_closure_revision_mismatch:' + path);
    }
  }
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
  const closureSources = pin.closureBlobs
    ? Object.fromEntries(await Promise.all(Object.keys(pin.closureBlobs).map(async path =>
        [path, await readFile(resolve(dirname(sourcePath), '../..', path), 'utf8')])))
    : undefined;
  inspectPinnedDependencies(pin, commandSource, responsesSource, graphSource, closureSources);
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
