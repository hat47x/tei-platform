#!/usr/bin/env node
// Read-only Reference CLI: accepted World -> scoped projections / proposals.
// Use the existing reference-shared-acceptance for preview, review and accept.
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { projectWorld, candidateFromEdit } from "./bridge.mjs";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const usage = [
  "usage:",
  "  node reference/studio-authoring/cli.mjs inspect -workspace DIR -focus relation:ID",
  "  node reference/studio-authoring/cli.mjs propose -workspace DIR -edit-file FILE",
  "  alternatively: -world-file FILE -revision-file FILE (read-only artifact inputs)",
  "  output is JSON on stdout; no command in this CLI writes authoritative source"
].join("\n");

function flags(argv) {
  const result = {};
  for (let i = 0; i < argv.length; i += 2) {
    if (!argv[i].startsWith("-") || !argv[i + 1] || ![
      "-workspace", "-world-file", "-revision-file", "-focus", "-edit-file"
    ].includes(argv[i]) || Object.hasOwn(result, argv[i])) {
      throw new Error("INVALID_FLAG: " + (argv[i] || ""));
    }
    result[argv[i]] = argv[i + 1];
  }
  return result;
}
function jsonFile(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}
function fromWorkspace(workspace) {
  // Resolve relative workspaces before switching the child process to repo root.
  workspace = resolve(workspace);
  // The two reads are not atomic. The Shared Acceptance Preview and Accept
  // must re-check the base revision and fail closed on concurrent edits.
  const run = (...args) => execFileSync("go", ["run", ...args], {
    cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024
  });
  return {
    world: JSON.parse(run("./cmd/tei", "world", "-workspace", workspace)),
    revision: JSON.parse(run("./tools/reference-shared-acceptance",
      "revision", "-workspace", workspace)).modelRevision
  };
}
function main(argv) {
  const [command, ...rest] = argv;
  if (!["inspect", "propose"].includes(command)) throw new Error("UNKNOWN_COMMAND");
  const opt = flags(rest);
  const fromFiles = !!opt["-world-file"] && !!opt["-revision-file"];
  if (!!opt["-workspace"] === fromFiles ||
      (!!opt["-workspace"] && (opt["-world-file"] || opt["-revision-file"])) ||
      (command === "inspect" && (!opt["-focus"] || opt["-edit-file"])) ||
      (command === "propose" && (!opt["-edit-file"] || opt["-focus"])))
    throw new Error("INVALID_INPUT");
  const { world, revision } = fromFiles
    ? { world: jsonFile(opt["-world-file"]),
        revision: jsonFile(opt["-revision-file"]).modelRevision }
    : fromWorkspace(opt["-workspace"]);
  if (command === "inspect") return projectWorld(world, revision, opt["-focus"]);
  return candidateFromEdit(world, revision, jsonFile(opt["-edit-file"]));
}
try {
  const result = main(process.argv.slice(2));
  process.stdout.write(JSON.stringify(result, null, 2) + "\n");
} catch (err) {
  process.stderr.write(String(err.message) + "\n" + usage + "\n");
  process.exitCode = 1;
}
