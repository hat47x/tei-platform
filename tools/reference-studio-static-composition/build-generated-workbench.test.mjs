import test from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const builder = resolve(root, "tools/reference-studio-static-composition/build-generated-workbench.mjs");
const artifact = resolve(root, "cmd/tei-studio/assets/workbench/main.js");

function run(args) {
  const result = spawnSync(process.execPath, [builder, ...args], {
    cwd: root, encoding: "utf8", timeout: 15000,
  });
  assert.ifError(result.error);
  assert.notEqual(result.status, 0, "an invalid builder invocation must fail");
  return result.stderr;
}

test("generated Workbench builder refuses unbound arguments without changing the asset", async () => {
  const before = await readFile(artifact);
  assert.match(run(["--bypass-pin"]), /generated_workbench_builder_does_not_accept_arguments/);
  assert.deepEqual(await readFile(artifact), before);
});

test("generated Workbench builder refuses an existing artifact before esbuild", async () => {
  const before = await readFile(artifact);
  assert.match(run([]), /generated_workbench_artifact_already_exists/);
  assert.deepEqual(await readFile(artifact), before);
});
