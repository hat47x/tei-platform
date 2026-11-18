// Check a real tei.authoring/v1 Data change through the existing Go Generator
// and esbuild, not by editing the JSON Reader contract or Generator Source.
import test from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { copyFile, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { build } from "esbuild";

const here = dirname(fileURLToPath(import.meta.url));
const repo = resolve(here, "../..");
const generatorSource = join(repo, "reference/tei-studio-workbench/builder-sources/inspection-reader-v1.transform.json");
const baselineContract = join(here, "inspection-reader.contract.json");
const labelSource = join(here, "inspection-limit-labels.ts");
const limits = [
  "inspection-is-a-derived-read-only-projection",
  "source-location-is-not-semantic-acceptance",
  "source-location-is-not-write-authority",
  "local-read-is-not-disclosure-permission",
];
const subject = { kind: "semantic-data", id: "sales.order" };
const catalog = { entries: [], modelRevision: "r", worldFingerprint: "w1", sourceIndexFingerprint: "s1" };

function sha256(bytes) {
  return "sha256:" + createHash("sha256").update(bytes).digest("hex");
}

function authoringDoc(required) {
  return `# Studio response

<!-- tei: data InspectionResponse in studio -->
| attribute | type | required | display |
| --- | --- | --- | --- |
| schemaVersion | std.text | yes | Schema |
| snapshotFingerprint | std.text | ${required ? "yes" : "no"} | Snapshot |
`;
}

test("Data required flag changes the actual esbuild Reader behavior; Generator Source is unchanged", async () => {
  const scratch = await mkdtemp(join(tmpdir(), "tei-studio-reader-e2e-"));
  try {
    const workspace = join(scratch, "workspace");
    const source = join(workspace, "semantic/studio/inspection-response.tei.md");
    await mkdir(dirname(source), { recursive: true });
    await writeFile(join(workspace, "tei.workspace.yaml"),
      "workspace: studio_reader\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n");
    const child = join(scratch, process.platform === "win32" ? "generator.exe" : "generator");
    execFileSync("go", ["build", "-mod=readonly", "-o", child, "./tools/reference-studio-part-generator"], {
      cwd: repo, env: { ...process.env, GOTOOLCHAIN: "local", GOWORK: "off" }, stdio: "pipe",
      timeout: 240_000,
    });
    const initialGeneratorBytes = await readFile(generatorSource);

    async function generation(index, required) {
      await writeFile(source, authoringDoc(required));
      const folder = join(scratch, "output-" + index);
      await mkdir(folder);
      const part = join(folder, "inspection-reader.generated.ts");
      const output = execFileSync(child, [
        "-transform", generatorSource, "-contract", baselineContract,
        "-authoring-root", workspace, "-authoring-data", "studio.inspection_response",
        "-artifact-out", part,
      ], { cwd: repo, encoding: "utf8", timeout: 30_000 });
      const report = JSON.parse(output);
      await copyFile(labelSource, join(folder, "inspection-limit-labels.ts"));
      const bundle = join(folder, "reader.mjs");
      await build({
        entryPoints: [part], outfile: bundle, absWorkingDir: folder,
        bundle: true, format: "esm", platform: "browser", target: "es2022",
        logLevel: "silent",
      });
      // Build the real product Workbench entrypoint with a test-only Source
      // overlay. No committed generated Reader or embedded asset is changed.
      // The production main.ts -> presentation.ts import closure is preserved.
      let overlayLoads = 0;
      const built = await build({
        absWorkingDir: here, entryPoints: [resolve(here, "main.ts")],
        outfile: join(folder, "workbench.mjs"), bundle: true, write: false,
        format: "esm", platform: "browser", target: "es2022",
        nodePaths: [resolve(here, "node_modules")], logLevel: "silent",
        plugins: [{
          name: "test-only-data-derived-reader",
          setup(api) {
            api.onLoad({ filter: /inspection-reader\.generated\.ts$/ }, async () => {
              overlayLoads++;
              return { contents: await readFile(part, "utf8"), loader: "ts", resolveDir: here };
            });
          },
        }],
      });
      assert.equal(overlayLoads, 1, "production Workbench must use exactly one model-derived Reader");
      assert.equal(built.outputFiles.length, 1, "expected one assembled Workbench JS");
      return { report, part: await readFile(part), bundle: await readFile(bundle),
        workbench: Buffer.from(built.outputFiles[0].contents),
        reader: await import(pathToFileURL(bundle).href) };
    }

    const before = await generation("before", false);
    const after = await generation("after", true);

    assert.equal(sha256(initialGeneratorBytes), before.report.transformBytesDigest);
    assert.equal(before.report.transformBytesDigest, after.report.transformBytesDigest);
    assert.equal(sha256(await readFile(generatorSource)), after.report.transformBytesDigest);
    assert.notEqual(before.report.authoring.sourceFingerprint, after.report.authoring.sourceFingerprint);
    assert.notEqual(before.report.authoring.modelRevision, after.report.authoring.modelRevision);
    assert.notEqual(before.report.contractCanonicalDigest, after.report.contractCanonicalDigest);
    assert.notEqual(before.reader.inspectionReaderProvenance.contractDigest,
      after.reader.inspectionReaderProvenance.contractDigest);
    assert.equal(before.report.artifactDigest, sha256(before.part));
    assert.equal(after.report.artifactDigest, sha256(after.part));
    assert.notEqual(sha256(before.bundle), sha256(after.bundle));
    assert.notEqual(sha256(before.workbench), sha256(after.workbench));
    for (const generated of [before, after]) {
      const expected = generated.reader.inspectionReaderProvenance.contractDigest;
      assert.ok(generated.workbench.includes(Buffer.from('contractDigest: "' + expected + '"')),
        "real Workbench bundle did not carry the selected Data-derived Reader");
      assert.equal((generated.workbench.toString("utf8").match(/function readInspection\d*\(/g) ?? []).length, 1,
        "Workbench must contain one generated Inspector Reader");
    }
    assert.ok(!after.workbench.includes(Buffer.from(
      'contractDigest: "' + before.reader.inspectionReaderProvenance.contractDigest + '"')),
      "new Workbench still contains the old Reader contract");

    const valid = {
      schemaVersion: "tei.cli.inspect/v0", snapshotFingerprint: "w1:s1",
      effect: "read-only", subject, canonical: { id: "sales.order" },
      worldFingerprint: "w1", sourceIndexFingerprint: "s1",
      sourceStatus: "indexed", authorityLimits: limits,
    };
    const { snapshotFingerprint: omitted, ...withoutNewField } = valid;
    assert.equal(omitted, "w1:s1");
    // Old Reader still accepts the old response. New Reader must not.
    assert.doesNotThrow(() => before.reader.readInspection(withoutNewField, subject, catalog));
    assert.throws(() => after.reader.readInspection(withoutNewField, subject, catalog),
      /INVALID_INSPECTION_RESPONSE/);
    assert.throws(() => after.reader.readInspection({ ...valid, snapshotFingerprint: null }, subject, catalog),
      /INVALID_INSPECTION_RESPONSE/);
    assert.throws(() => after.reader.readInspection({ ...valid, snapshotFingerprint: 7 }, subject, catalog),
      /INVALID_INSPECTION_RESPONSE/);
    assert.doesNotThrow(() => after.reader.readInspection(valid, subject, catalog));
    assert.equal(after.reader.inspectionReaderProvenance.operation, "main.StudioBridge.InspectSubject");
  } finally {
    await rm(scratch, { recursive: true, force: true });
  }
});
