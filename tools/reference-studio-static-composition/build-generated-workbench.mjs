// Thin adapter for the existing generated Workbench esbuild command.
// Invoked as a pinned child by the product tei self-build CLI. It does not
// select a Builder, authorize execution, or qualify a Stage2 Studio.
import { createHash } from "node:crypto";
import { access, readFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

if (process.argv.length !== 2) {
  throw new Error("generated_workbench_builder_does_not_accept_arguments");
}

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const workbench = resolve(root, "reference/tei-studio-workbench");
const artifact = resolve(root, "cmd/tei-studio/assets/workbench/main.js");

try {
  await access(artifact);
  throw new Error("generated_workbench_artifact_already_exists");
} catch (error) {
  if (error?.code !== "ENOENT") throw error;
}

const { build } = createRequire(resolve(workbench, "package.json"))("esbuild");
await build({
  absWorkingDir: workbench,
  entryPoints: ["main.generated.ts"],
  outfile: artifact,
  bundle: true,
  format: "esm",
  platform: "browser",
  target: "es2022",
  nodePaths: [resolve(workbench, "node_modules")],
});

const bytes = await readFile(artifact);
const digest = createHash("sha256").update(bytes).digest("hex");
process.stdout.write(
  JSON.stringify({
    schemaVersion: "tei.builder.studio-workbench-esbuild/v0",
    artifactDigest: `sha256:${digest}`,
  }) + "\n",
);
