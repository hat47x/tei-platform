#!/usr/bin/env node
"use strict";
/*
 * Participant root builder for the DSH H2/H3 comparison (TEI #2509 / TAI #168).
 *
 * Assembles the ONLY content a participant Session may see:
 *   bin/tei.exe                 the fixed, digest-pinned Product executable
 *   reference/workspace/...     the public Workspace byte copy (the logical
 *                               path the fixed participant Task names)
 *   task.txt                    the fixed participant Task raw bytes
 *   .agents/skills/yi-tei-modeling/...   H3 arm only: the pinned Product Skill
 *
 * The builder wipes and rebuilds the target directory so every Session starts
 * from a verified state, and writes a manifest with per-file digests plus the
 * aggregate digests the SK-02 trial record binds. It refuses to build when a
 * pinned digest does not match, instead of silently recording a different
 * binary/workspace/task.
 *
 * Usage:
 *   node build-participant-root.mjs --out <dir> --binary <tei.exe> \
 *     [--binary-digest sha256:...] --workspace-src <dir> --task <task.txt> \
 *     [--task-digest sha256:...] [--skill-src <dir> --skill-digest sha256:...]
 */
import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { pathToFileURL } from "node:url";

function sha256hex(buf) {
  return crypto.createHash("sha256").update(buf).digest("hex");
}

function normalizeDigest(value) {
  if (typeof value !== "string") return null;
  const v = value.startsWith("sha256:") ? value.slice(7) : value;
  return /^[a-f0-9]{64}$/.test(v) ? v.toLowerCase() : null;
}

function walk(dir, base = "") {
  const out = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    const rel = base ? `${base}/${entry.name}` : entry.name;
    const abs = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...walk(abs, rel));
    else if (entry.isFile()) out.push({ rel, abs });
  }
  return out;
}

function treeDigest(files) {
  // Deterministic: sorted relative paths + per-file byte length + sha256.
  const h = crypto.createHash("sha256");
  for (const f of files) {
    const buf = fs.readFileSync(f.abs);
    h.update(`${f.rel}\n${buf.length}\n${sha256hex(buf)}\n`, "utf8");
  }
  return h.digest("hex");
}

function parseArgs(argv) {
  const args = {};
  for (let i = 0; i < argv.length; i++) {
    const key = argv[i];
    if (!key.startsWith("--")) throw new Error(`unexpected argument: ${key}`);
    const name = key.slice(2);
    if (args[name] !== undefined) throw new Error(`duplicate argument: ${key}`);
    args[name] = argv[++i];
  }
  return args;
}

/** Map kebab-case CLI flags to the camelCase options buildRoot takes.
 *  Kept separate so a flag rename cannot silently drop a pinned digest. */
export function optionsFromArgs(raw) {
  return {
    out: raw.out,
    binary: raw.binary,
    binaryDigest: raw["binary-digest"],
    workspaceSrc: raw["workspace-src"],
    task: raw.task,
    taskDigest: raw["task-digest"],
    skillSrc: raw["skill-src"] ?? null,
    skillDigest: raw["skill-digest"] ?? null,
  };
}

export function buildRoot(options) {
  const {
    out, binary, binaryDigest,
    workspaceSrc, task, taskDigest,
    skillSrc = null, skillDigest = null,
  } = options;
  if (!out || !binary || !workspaceSrc || !task) {
    throw new Error("required: --out --binary --workspace-src --task");
  }
  const problems = [];

  const binaryBuf = fs.readFileSync(binary);
  const actualBinaryDigest = sha256hex(binaryBuf);
  const expectedBinary = normalizeDigest(binaryDigest);
  if (expectedBinary && expectedBinary !== actualBinaryDigest) {
    problems.push(`binary digest mismatch: expected sha256:${expectedBinary}, built sha256:${actualBinaryDigest}`);
  }

  const taskBuf = fs.readFileSync(task);
  const actualTaskDigest = sha256hex(taskBuf);
  const expectedTask = normalizeDigest(taskDigest);
  if (expectedTask && expectedTask !== actualTaskDigest) {
    problems.push(`task digest mismatch: expected sha256:${expectedTask}, file sha256:${actualTaskDigest}`);
  }

  let skillFiles = [];
  if (skillSrc) {
    skillFiles = walk(skillSrc);
    const skillMain = skillFiles.find((f) => f.rel === "SKILL.md");
    if (!skillMain) problems.push("skill source has no SKILL.md");
    if (skillDigest) {
      const expectedSkill = normalizeDigest(skillDigest);
      const actualSkill = skillMain ? sha256hex(fs.readFileSync(skillMain.abs)) : null;
      if (expectedSkill && actualSkill && expectedSkill !== actualSkill) {
        problems.push(`skill SKILL.md digest mismatch: expected sha256:${expectedSkill}, source sha256:${actualSkill}`);
      }
    }
  }

  if (problems.length > 0) {
    const err = new Error("participant root build refused:\n" + problems.map((p) => `  - ${p}`).join("\n"));
    err.code = "ROOT_BUILD_REFUSED";
    throw err;
  }

  fs.rmSync(out, { recursive: true, force: true });
  fs.mkdirSync(path.join(out, "bin"), { recursive: true });
  fs.mkdirSync(path.join(out, "reference"), { recursive: true });
  fs.copyFileSync(binary, path.join(out, "bin", "tei.exe"));
  fs.cpSync(workspaceSrc, path.join(out, "reference", "workspace"), { recursive: true });
  fs.writeFileSync(path.join(out, "task.txt"), taskBuf);

  if (skillSrc) {
    fs.mkdirSync(path.join(out, ".agents", "skills"), { recursive: true });
    fs.cpSync(skillSrc, path.join(out, ".agents", "skills", "yi-tei-modeling"), { recursive: true });
  }

  const built = walk(out);
  const manifest = {
    schema: "tei.tools.dsh-participant-root/v1",
    out: path.resolve(out),
    arm: skillSrc ? "H3" : "H2",
    binary: { name: "bin/tei.exe", bytes: binaryBuf.length, sha256: actualBinaryDigest },
    task: { name: "task.txt", bytes: taskBuf.length, sha256: actualTaskDigest },
    skill: skillSrc
      ? {
          root: ".agents/skills/yi-tei-modeling",
          files: skillFiles.map((f) => ({ rel: f.rel, sha256: sha256hex(fs.readFileSync(f.abs)) })),
        }
      : null,
    workspaceTreeDigest: treeDigest(walk(path.join(out, "reference", "workspace"))),
    rootTreeDigest: treeDigest(built),
    fileCount: built.length,
    files: built.map((f) => ({ rel: f.rel, bytes: fs.statSync(f.abs).size, sha256: sha256hex(fs.readFileSync(f.abs)) })),
  };
  fs.writeFileSync(path.join(path.dirname(out), `${path.basename(out)}.manifest.json`), JSON.stringify(manifest, null, 2));
  return manifest;
}

function main() {
  const argv = process.argv.slice(2);
  if (argv.length === 0 || argv[0] === "-h" || argv[0] === "--help") {
    console.error("usage: node build-participant-root.mjs --out <dir> --binary <tei.exe> [--binary-digest sha256:...] --workspace-src <dir> --task <task.txt> [--task-digest sha256:...] [--skill-src <dir> --skill-digest sha256:...]");
    process.exit(2);
  }
  const options = optionsFromArgs(parseArgs(argv));
  try {
    const manifest = buildRoot(options);
    console.log(JSON.stringify(manifest, null, 2));
  } catch (e) {
    console.error(String(e.message ?? e));
    process.exit(1);
  }
}

const invokedDirectly = process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href;
if (invokedDirectly) main();
