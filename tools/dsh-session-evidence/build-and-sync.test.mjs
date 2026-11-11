"use strict";
/*
 * Independent regression tests for build-participant-root.mjs and
 * sync-participant-slot.mjs. Synthetic fixtures only; no real binary,
 * Workspace, Task, or Skill bytes.
 */
import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import crypto from "node:crypto";
import { buildRoot, optionsFromArgs } from "./build-participant-root.mjs";
import { syncVerifiedSlot } from "./sync-participant-slot.mjs";

function sha256hex(buf) {
  return crypto.createHash("sha256").update(buf).digest("hex");
}

function tmpdir(name) {
  return fs.mkdtempSync(path.join(os.tmpdir(), `dsh-root-test-${name}-`));
}

function fixture() {
  const dir = tmpdir("fixture");
  fs.writeFileSync(path.join(dir, "tei.exe"), Buffer.from("MZ-synthetic-binary"));
  const ws = path.join(dir, "workspace-src");
  fs.mkdirSync(path.join(ws, "semantic", "sales"), { recursive: true });
  fs.writeFileSync(path.join(ws, "manifest.yaml"), "name: synthetic\n");
  fs.writeFileSync(path.join(ws, "semantic", "sales", "order.tei"), "subject sales.order\n");
  const task = path.join(dir, "task.txt");
  fs.writeFileSync(task, "合成タスク本文\n");
  const skill = path.join(dir, "skill-src");
  fs.mkdirSync(path.join(skill, "references"), { recursive: true });
  fs.writeFileSync(path.join(skill, "SKILL.md"), "synthetic skill\n");
  fs.writeFileSync(path.join(skill, "references", "read-only-model-investigation.md"), "synthetic reference\n");
  return { dir, binary: path.join(dir, "tei.exe"), ws, task, skill };
}

function makeStage(dir, arm) {
  fs.mkdirSync(path.join(dir, "bin"), { recursive: true });
  fs.mkdirSync(path.join(dir, "reference", "workspace", "semantic"), { recursive: true });
  fs.writeFileSync(path.join(dir, "bin", "tei.exe"), "synthetic-binary");
  fs.writeFileSync(path.join(dir, "reference", "workspace", "semantic", "order.tei"), "synthetic source\n");
  fs.writeFileSync(path.join(dir, "task.txt"), "synthetic task\n");
  if (arm === "H3") {
    fs.mkdirSync(path.join(dir, ".agents", "skills", "yi-tei-modeling"), { recursive: true });
    fs.writeFileSync(path.join(dir, ".agents", "skills", "yi-tei-modeling", "SKILL.md"), "synthetic skill\n");
  }
  const manifest = {
    schema: "tei.tools.dsh-participant-root/v1",
    arm,
    fileCount: arm === "H3" ? 4 : 3,
    workspaceTreeDigest: "sha256:synthetic-ws",
    rootTreeDigest: "sha256:synthetic-root",
    binary: { sha256: "sha256:synthetic-bin" },
    task: { sha256: "sha256:synthetic-task" },
    skill: arm === "H3" ? { files: [{ rel: "SKILL.md", sha256: "sha256:synthetic-skill" }] } : null,
  };
  fs.writeFileSync(path.join(path.dirname(dir), `${path.basename(dir)}.manifest.json`), JSON.stringify(manifest));
  return dir;
}

test("H2 root contains only binary, workspace and task", () => {
  const f = fixture();
  const out = path.join(f.dir, "slot");
  const m = buildRoot({
    out, binary: f.binary, workspaceSrc: f.ws, task: f.task,
    binaryDigest: "sha256:" + sha256hex(fs.readFileSync(f.binary)),
    taskDigest: "sha256:" + sha256hex(fs.readFileSync(f.task)),
  });
  assert.equal(m.arm, "H2");
  assert.equal(m.skill, null);
  const rels = m.files.map((x) => x.rel).sort();
  assert.deepEqual(rels, [
    "bin/tei.exe",
    "reference/workspace/manifest.yaml",
    "reference/workspace/semantic/sales/order.tei",
    "task.txt",
  ]);
  assert.ok(fs.existsSync(path.join(out, "reference", "workspace", "semantic", "sales", "order.tei")));
  assert.ok(!fs.existsSync(path.join(out, ".agents")));
});

test("H3 root adds the pinned skill package under .agents/skills", () => {
  const f = fixture();
  const out = path.join(f.dir, "slot");
  const m = buildRoot({
    out, binary: f.binary, workspaceSrc: f.ws, task: f.task, skillSrc: f.skill,
    skillDigest: "sha256:" + sha256hex(fs.readFileSync(path.join(f.skill, "SKILL.md"))),
  });
  assert.equal(m.arm, "H3");
  assert.equal(m.skill.files.length, 2);
  assert.ok(fs.existsSync(path.join(out, ".agents", "skills", "yi-tei-modeling", "SKILL.md")));
  assert.ok(fs.existsSync(path.join(out, ".agents", "skills", "yi-tei-modeling", "references", "read-only-model-investigation.md")));
});

test("a digest mismatch refuses the build instead of recording a different input", () => {
  const f = fixture();
  const out = path.join(f.dir, "slot");
  assert.throws(() => buildRoot({
    out, binary: f.binary, workspaceSrc: f.ws, task: f.task,
    binaryDigest: "sha256:" + "0".repeat(64),
  }), (e) => e.code === "ROOT_BUILD_REFUSED" && /binary digest mismatch/.test(e.message));
  assert.ok(!fs.existsSync(path.join(out, "task.txt")), "refused build must not leave a participant root");
});

test("rebuild wipes the previous root so no arm residue survives", () => {
  const f = fixture();
  const out = path.join(f.dir, "slot");
  buildRoot({ out, binary: f.binary, workspaceSrc: f.ws, task: f.task, skillSrc: f.skill });
  assert.ok(fs.existsSync(path.join(out, ".agents", "skills", "yi-tei-modeling", "SKILL.md")));
  const m2 = buildRoot({ out, binary: f.binary, workspaceSrc: f.ws, task: f.task });
  assert.equal(m2.arm, "H2");
  assert.ok(!fs.existsSync(path.join(out, ".agents")), "H2 rebuild must remove the H3 skill residue");
});

test("tree digests are deterministic across rebuilds", () => {
  const f = fixture();
  const outA = path.join(f.dir, "slotA");
  const outB = path.join(f.dir, "slotB");
  const a = buildRoot({ out: outA, binary: f.binary, workspaceSrc: f.ws, task: f.task });
  const b = buildRoot({ out: outB, binary: f.binary, workspaceSrc: f.ws, task: f.task });
  assert.equal(a.workspaceTreeDigest, b.workspaceTreeDigest);
  assert.equal(a.rootTreeDigest, b.rootTreeDigest);
});

test("kebab-case CLI flags map onto the build options without dropping a pin", () => {
  const opts = optionsFromArgs({
    out: "/x/slot", binary: "/x/tei.exe", "binary-digest": "sha256:" + "a".repeat(64),
    "workspace-src": "/x/ws", task: "/x/task.txt", "task-digest": "sha256:" + "b".repeat(64),
    "skill-src": "/x/skill", "skill-digest": "sha256:" + "c".repeat(64),
  });
  assert.equal(opts.workspaceSrc, "/x/ws");
  assert.equal(opts.binaryDigest, "sha256:" + "a".repeat(64));
  assert.equal(opts.taskDigest, "sha256:" + "b".repeat(64));
  assert.equal(opts.skillSrc, "/x/skill");
  assert.equal(opts.skillDigest, "sha256:" + "c".repeat(64));
});

test("verified sync reproduces the staged arm content in the slot", () => {
  const base = fs.mkdtempSync(path.join(os.tmpdir(), "dsh-sync-test-"));
  const stage = makeStage(path.join(base, "stage-h2-a"), "H2");
  const slot = path.join(base, "slot");
  const r = syncVerifiedSlot(stage, slot);
  assert.equal(r.arm, "H2");
  assert.equal(r.fileCount, 3);
  assert.ok(fs.existsSync(path.join(slot, "bin", "tei.exe")));
  assert.ok(!fs.existsSync(path.join(slot, ".agents")));
});

test("syncing an H3 stage then an H2 stage leaves no Skill residue", () => {
  const base = fs.mkdtempSync(path.join(os.tmpdir(), "dsh-sync-test-"));
  const h3 = makeStage(path.join(base, "stage-h3-a"), "H3");
  const h2 = makeStage(path.join(base, "stage-h2-b"), "H2");
  const slot = path.join(base, "slot");
  syncVerifiedSlot(h3, slot);
  assert.ok(fs.existsSync(path.join(slot, ".agents", "skills", "yi-tei-modeling", "SKILL.md")));
  const r = syncVerifiedSlot(h2, slot);
  assert.equal(r.arm, "H2");
  assert.ok(!fs.existsSync(path.join(slot, ".agents")), "H2 sync must remove the H3 Skill residue");
});

test("a slot whose bytes were tampered with after sync is detectable", () => {
  const base = fs.mkdtempSync(path.join(os.tmpdir(), "dsh-sync-test-"));
  const stage = makeStage(path.join(base, "stage-h2-a"), "H2");
  const slot = path.join(base, "slot");
  syncVerifiedSlot(stage, slot);
  fs.writeFileSync(path.join(slot, "rogue.txt"), "written during the session");
  fs.writeFileSync(path.join(slot, "task.txt"), "modified task");
  const walk = (dir, base2 = "") => {
    const out = [];
    for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
      const rel = base2 ? `${base2}/${e.name}` : e.name;
      if (e.isDirectory()) out.push(...walk(path.join(dir, e.name), rel));
      else out.push(rel);
    }
    return out;
  };
  assert.notEqual(walk(slot).length, walk(stage).length, "the tampering is detectable by file count");
});
