// WEB-P1a-2: qualified, ephemeral Workbench source candidate.
// This does not modify a historical Workbench Pin or its committed bundle.
import { readFile, writeFile } from "node:fs/promises";
import { dirname, resolve, join } from "node:path";
import { fileURLToPath } from "node:url";
import {
  WORKBENCH_NATIVE_HEAD, WORKBENCH_NATIVE_MAIN_BLOB,
  pinnedWorkbenchRevision, inspectPinnedDependencies, gitBlobSHA,
} from "./connect-workbench.mjs";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const workbench = resolve(root, "reference/tei-studio-workbench");
const READER_BLOB = "034e983aa9ba85009cfacbd47d67239b12807c46";
const PRESENTER_BLOB = "14324071db2321fc8d557613b5a1779f45d6de9d";
const IMPORT_ANCHOR = "import type { Entry, Subject, Finding, Catalog } from \"./presentation.ts\";";
const IMPORT_NEW = "import { readInspectorPresentation } from \"./inspector-presentation.ts\";";
const OLD_BLOCK = "    const view = readInspection(unwrap(reply), subject, catalogSnapshot!);\n    editor.setDocument({\n      id: `${id}/${subjectKey(subject)}`,\n      version: view.version,\n      text: view.text,\n    });\n    element(\"source-status\").textContent = `参照情報：${view.sourceStatus}`;\n    showInspectionLimits(view.authorityLimits);\n    notice.textContent = \"選択したモデルの投影を表示しています。\";\n    notice.dataset.state = \"ready\";";
const NEW_BLOCK = "    const presentation = readInspectorPresentation(unwrap(reply), {\n      workspaceId: id,\n      selected: subject,\n      catalog: catalogSnapshot!,\n    });\n    editor.setDocument(presentation.document);\n    element(\"source-status\").textContent = presentation.sourceStatusText;\n    showInspectionLimits(presentation.authorityLimits);\n    notice.textContent = presentation.noticeText;\n    notice.dataset.state = \"ready\";";

function exactlyOnce(source, needle) {
  return source.split(needle).length === 2;
}

/** No DOM, Host, IO, or authority is involved in this deterministic transformation. */
export function createInspectorCandidate(main, reader, presenter) {
  if (typeof main !== "string" || gitBlobSHA(main) !== WORKBENCH_NATIVE_MAIN_BLOB)
    throw Error("INSPECTOR_CANDIDATE_BASE_REVISION_MISMATCH");
  if (typeof reader !== "string" || gitBlobSHA(reader) !== READER_BLOB)
    throw Error("INSPECTOR_CANDIDATE_READER_REVISION_MISMATCH");
  if (typeof presenter !== "string" || gitBlobSHA(presenter) !== PRESENTER_BLOB)
    throw Error("INSPECTOR_CANDIDATE_PRESENTER_REVISION_MISMATCH");
  if (!exactlyOnce(main, IMPORT_ANCHOR) || !exactlyOnce(main, OLD_BLOCK))
    throw Error("INSPECTOR_CANDIDATE_EDIT_ANCHOR_MISMATCH");
  const candidate = main.replace(IMPORT_ANCHOR, IMPORT_ANCHOR + "\n" + IMPORT_NEW)
    .replace(OLD_BLOCK, NEW_BLOCK);
  if (candidate === main ||
      !exactlyOnce(candidate, IMPORT_NEW) ||
      !exactlyOnce(candidate, NEW_BLOCK) ||
      candidate.includes(OLD_BLOCK))
    throw Error("INSPECTOR_CANDIDATE_TRANSFORM_INCOMPLETE");
  return {
    content: candidate,
    sourceBlobs: Object.freeze({
      main: gitBlobSHA(main),
      reader: gitBlobSHA(reader),
      presenter: gitBlobSHA(presenter),
    }),
    candidateBlob: gitBlobSHA(candidate),
  };
}

async function main() {
  if (process.argv.length !== 2)
    throw Error("usage: node tools/reference-studio-static-composition/inspector-candidate.mjs");
  const mainPath = join(workbench, "main.ts");
  const readerPath = join(workbench, "presentation.ts");
  const presenterPath = join(workbench, "inspector-presentation.ts");
  const [original, reader, presenter] = await Promise.all(
    [mainPath, readerPath, presenterPath].map(path => readFile(path, "utf8")));
  // Existing historical fixture must still be valid. Never adjust its pin to match.
  const pin = pinnedWorkbenchRevision(original);
  if (pin.head !== WORKBENCH_NATIVE_HEAD || !pin.closureBlobs)
    throw Error("INSPECTOR_CANDIDATE_NOT_NATIVE_PIN");
  const closureSources = Object.fromEntries(await Promise.all(
    Object.keys(pin.closureBlobs).map(async path =>
      [path, await readFile(resolve(root, path), "utf8")])));
  const [command, responses, graph] = await Promise.all([
    readFile(resolve(root, "reference/tei-studio-command/command.ts"), "utf8"),
    readFile(join(workbench, "responses.ts"), "utf8"),
    readFile(resolve(root, "reference/tei-studio-graph/graph.ts"), "utf8"),
  ]);
  inspectPinnedDependencies(pin, command, responses, graph, closureSources);
  const candidate = createInspectorCandidate(original, reader, presenter);
  const target = join(workbench, "main.inspector-candidate.ts");
  // Fail closed: never replace an existing file, including a prior generated output.
  await writeFile(target, candidate.content, { flag: "wx" });
  process.stdout.write(JSON.stringify({
    schemaVersion: "tei.studio.inspector-candidate/v0",
    historicPin: pin.head,
    sourceBlobs: candidate.sourceBlobs,
    candidateBlob: candidate.candidateBlob,
    status: "candidate-only-not-embedded-or-consumer-accepted",
  }) + "\n");
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main().catch(error => { console.error(error); process.exitCode = 1; });
}
