#!/usr/bin/env node
"use strict";
/*
 * Verified slot sync for the DSH H2/H3 comparison (TEI #2509 / TAI #168).
 *
 * Each participant Session runs in ONE fixed slot path (so the recorded
 * cli_cwd is identical across the four trials), but the slot content is
 * re-assembled from a pre-built, digest-pinned staging root before every
 * Session. This tool performs the copy and then verifies every file byte
 * digest and the file count against the staging manifest, refusing to leave
 * a slot that does not match its arm's pinned content (e.g. leftover Skill
 * files from the previous H3 run would contaminate the next H2 run).
 *
 * Usage:
 *   node sync-participant-slot.mjs --stage <stageRoot> --slot <slotDir>
 */
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { sha256hex } from "./collect-session.mjs";

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

/** Copy a staging root into the slot and verify byte identity. */
export function syncVerifiedSlot(stage, slot) {
  const stageDir = path.resolve(stage);
  const slotDir = path.resolve(slot);
  const manifestPath = path.join(path.dirname(stageDir), `${path.basename(stageDir)}.manifest.json`);
  if (!fs.existsSync(manifestPath)) {
    throw new Error(`staging manifest not found: ${manifestPath}`);
  }
  const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));

  // wipe the slot (removes any previous arm's residue) and copy
  fs.rmSync(slotDir, { recursive: true, force: true });
  fs.cpSync(stageDir, slotDir, { recursive: true });

  const stageFiles = walk(stageDir);
  const slotFiles = walk(slotDir);
  const problems = [];
  if (stageFiles.length !== slotFiles.length) {
    problems.push(`file count differs: stage ${stageFiles.length} vs slot ${slotFiles.length}`);
  }
  const stageByRel = new Map(stageFiles.map((f) => [f.rel, f]));
  for (const f of slotFiles) {
    const src = stageByRel.get(f.rel);
    if (!src) { problems.push(`unexpected file in slot: ${f.rel}`); continue; }
    const a = sha256hex(fs.readFileSync(src.abs));
    const b = sha256hex(fs.readFileSync(f.abs));
    if (a !== b) problems.push(`digest mismatch: ${f.rel}`);
  }
  // the staging manifest must describe the same content that was copied
  if (manifest.arm && manifest.fileCount && manifest.fileCount !== slotFiles.length) {
    problems.push(`slot file count ${slotFiles.length} != manifest ${manifest.fileCount}`);
  }
  if (problems.length > 0) {
    const err = new Error("slot sync refused:\n" + problems.map((p) => `  - ${p}`).join("\n"));
    err.code = "SLOT_SYNC_REFUSED";
    throw err;
  }
  return {
    schema: "tei.tools.dsh-participant-slot-sync/v1",
    arm: manifest.arm ?? null,
    stage: stageDir,
    slot: slotDir,
    fileCount: slotFiles.length,
    workspaceTreeDigest: manifest.workspaceTreeDigest ?? null,
    rootTreeDigest: manifest.rootTreeDigest ?? null,
    binarySha256: manifest.binary?.sha256 ?? null,
    taskSha256: manifest.task?.sha256 ?? null,
    skillFileCount: manifest.skill ? manifest.skill.files.length : 0,
  };
}

function main() {
  const argv = process.argv.slice(2);
  const args = {};
  for (let i = 0; i < argv.length; i++) {
    const key = argv[i];
    if (!key.startsWith("--")) { console.error(`unexpected argument: ${key}`); process.exit(2); }
    args[key.slice(2)] = argv[++i];
  }
  if (!args.stage || !args.slot) {
    console.error("usage: node sync-participant-slot.mjs --stage <stageRoot> --slot <slotDir>");
    process.exit(2);
  }
  try {
    console.log(JSON.stringify(syncVerifiedSlot(args.stage, args.slot), null, 2));
  } catch (e) {
    console.error(String(e.message ?? e));
    process.exit(1);
  }
}

const invokedDirectly = process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href;
if (invokedDirectly) main();
