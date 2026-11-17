import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { createInspectorCandidate } from "./inspector-candidate.mjs";
import { WORKBENCH_NATIVE_MAIN_BLOB, gitBlobSHA } from "./connect-workbench.mjs";

const root = new URL("../../", import.meta.url);
const sourceFiles = [
  "reference/tei-studio-workbench/main.ts",
  "reference/tei-studio-workbench/presentation.ts",
  "reference/tei-studio-workbench/inspector-presentation.ts",
];

async function currentSources() {
  return Promise.all(sourceFiles.map(path => readFile(new URL(path, root), "utf8")));
}

test("WEB-P1a-2 candidate connects only validated read-only Inspector display", async () => {
  const [main, reader, presenter] = await currentSources();
  const a = createInspectorCandidate(main, reader, presenter);
  const b = createInspectorCandidate(main, reader, presenter);
  assert.equal(gitBlobSHA(main), WORKBENCH_NATIVE_MAIN_BLOB);
  assert.deepEqual(a, b, "same source closure must yield exactly the same candidate");
  assert.notEqual(a.candidateBlob, WORKBENCH_NATIVE_MAIN_BLOB,
    "the new source must not inherit a historical Consumer acceptance");
  assert.equal(a.candidateBlob, gitBlobSHA(a.content));
  assert.match(a.content, /import \{ readInspectorPresentation \} from "\.\/inspector-presentation\.ts";/);
  assert.match(a.content, /const presentation = readInspectorPresentation\(unwrap\(reply\), \{/);
  assert.match(a.content, /editor\.setDocument\(presentation\.document\);/);
  assert.match(a.content, /showInspectionLimits\(presentation\.authorityLimits\);/);
  assert.match(a.content, /inspectEligibility\(catalogSnapshot, subject\)/);
  assert.match(a.content, /responses\.run\("inspection"/);
  assert.match(a.content, /client!\.inspectSubject\(id, subject\.kind, subject\.id\)/);
  assert.match(a.content, /catalog: catalogSnapshot!/);
  assert.equal(a.content.includes("const presentation = readInspectorPresentation"), true);
  assert.equal(main.includes("const presentation = readInspectorPresentation"), false);
  assert.equal(a.content.split("const presentation = readInspectorPresentation").length, 2);
});

test("WEB-P1a-2 candidate rejects any of the three altered source blobs", async () => {
  const [main, reader, presenter] = await currentSources();
  assert.throws(() => createInspectorCandidate(main + "\n", reader, presenter),
    /INSPECTOR_CANDIDATE_BASE_REVISION_MISMATCH/);
  assert.throws(() => createInspectorCandidate(main, reader + "\n", presenter),
    /INSPECTOR_CANDIDATE_READER_REVISION_MISMATCH/);
  assert.throws(() => createInspectorCandidate(main, reader, presenter + "\n"),
    /INSPECTOR_CANDIDATE_PRESENTER_REVISION_MISMATCH/);
  assert.throws(() => createInspectorCandidate(main.replace("readInspection", "readOther"),
    reader, presenter), /INSPECTOR_CANDIDATE_BASE_REVISION_MISMATCH/);
});

test("WEB-P1a-2 candidate does not replace Source, Pin or Host execution", async () => {
  const [main, reader, presenter] = await currentSources();
  const result = createInspectorCandidate(main, reader, presenter);
  const region = str => str.slice(
    str.indexOf("async function inspect(subject: Subject)"),
    str.indexOf("async function validate("),
  );
  const original = region(main);
  const candidate = region(result.content);
  assert.match(original, /showInspectionLimits\(view\.authorityLimits\)/);
  assert.match(candidate, /showInspectionLimits\(presentation\.authorityLimits\)/);
  assert.equal(candidate.includes('responses.invalidate("inspection")'),
    original.includes('responses.invalidate("inspection")'));
  assert.equal(candidate.includes('notice.dataset.state = "error"'),
    original.includes('notice.dataset.state = "error"'));
  assert.equal(candidate.includes("client!.inspectSubject("),
    original.includes("client!.inspectSubject("));
  assert.equal(result.content.includes("source-location"), main.includes("source-location"));
  assert.equal(result.sourceBlobs.reader, gitBlobSHA(reader));
  assert.equal(result.sourceBlobs.presenter, gitBlobSHA(presenter));
});
