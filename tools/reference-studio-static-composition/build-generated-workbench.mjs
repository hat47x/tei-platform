// Thin adapter for the existing generated Workbench esbuild command.
// Invoked as a pinned child by the product tei self-build CLI. It does not
// select a Builder, authorize execution, or qualify a Stage2 Studio.
// No arguments preserves the fixed historical static-composition build.
// Explicit --inspection-reader accepts a part made by the existing Go Generator
// and rebuilds the actual Workbench entrypoint without changing tracked Sources.
import { createHash } from "node:crypto";
import { access, readFile, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const workbench = resolve(root, "reference/tei-studio-workbench");
const fixedArtifact = resolve(root, "cmd/tei-studio/assets/workbench/main.js");
const args = process.argv.slice(2);
const fromModel = args.length === 4 &&
  args[0] === "--inspection-reader" && args[2] === "--artifact-out";
if (args.length !== 0 && !fromModel)
  throw new Error("generated_workbench_builder_unsupported_arguments");
const artifact = fromModel ? resolve(args[3]) : fixedArtifact;
const part = fromModel ? resolve(args[1]) : null;
if (fromModel && (part === artifact || part === resolve(workbench, "inspection-reader.generated.ts")))
  throw new Error("generated_workbench_builder_requires_distinct_derived_input");

try {
  await access(artifact);
  throw new Error("generated_workbench_artifact_already_exists");
} catch (error) {
  if (error?.code !== "ENOENT") throw error;
}

const { build } = createRequire(resolve(workbench, "package.json"))("esbuild");
const buildOptions = {
  absWorkingDir: workbench,
  entryPoints: [fromModel ? "main.ts" : "main.generated.ts"],
  outfile: artifact,
  bundle: true,
  format: "esm",
  platform: "browser",
  target: "es2022",
  nodePaths: [resolve(workbench, "node_modules")],
};

let partDigest;
let contractDigest;
if (fromModel) {
  const partBytes = await readFile(part);
  const ids = [...partBytes.toString("utf8").matchAll(/contractDigest: "(sha256:[0-9a-f]{64})"/g)];
  if (ids.length !== 1)
    throw new Error("generated_workbench_reader_contract_ambiguous");
  contractDigest = ids[0][1];
  partDigest = "sha256:" + createHash("sha256").update(partBytes).digest("hex");
  let loads = 0;
  const built = await build({
    ...buildOptions,
    write: false,
    plugins: [{
      name: "generated-inspection-reader-input",
      setup(plugin) {
        plugin.onLoad({ filter: /inspection-reader\.generated\.ts$/ }, async (request) => {
          if (resolve(request.path) !== resolve(workbench, "inspection-reader.generated.ts"))
            throw new Error("generated_workbench_unexpected_reader_path");
          loads++;
          return { contents: partBytes.toString("utf8"), loader: "ts", resolveDir: workbench };
        });
      },
    }],
  });
  if (loads !== 1 || built.outputFiles.length !== 1)
    throw new Error("generated_workbench_reader_not_unique");
  const bundled = Buffer.from(built.outputFiles[0].contents);
  const content = bundled.toString("utf8");
  if (!content.includes('contractDigest: "' + contractDigest + '"') ||
      (content.match(/function readInspection\d*\(/g) ?? []).length !== 1 ||
      !content.includes("teiInspectionReaderCalls"))
    throw new Error("generated_workbench_reader_not_in_bundle");
  // Do not leave an artifact when the Reader input or the Workbench closure
  // cannot be verified. Exclusive creation prevents stale-content overwrite.
  await writeFile(artifact, bundled, { flag: "wx" });
} else {
  await build(buildOptions);
}

const bytes = await readFile(artifact);
const digest = createHash("sha256").update(bytes).digest("hex");
process.stdout.write(
  JSON.stringify({
    schemaVersion: "tei.builder.studio-workbench-esbuild/v0",
    artifactDigest: `sha256:${digest}`,
    ...(fromModel ? {
      readerPartDigest: partDigest,
      readerContractDigest: contractDigest,
    } : {}),
  }) + "\n",
);
