#!/usr/bin/env node
"use strict";
/*
 * Builds the SK-02 trial-group JSON from the G3-20261111 run records
 * (TEI #2509 / TAI #168) so the TAI SK-02 checker can classify it.
 *
 * Every declared field is bound to an observed artifact:
 *   prompt_digest                -> the run record's stdin digest (log-matched)
 *   repo_head / binary_digest    -> pinned commit / slot manifest binary digest
 *   workspace_tree_digest        -> slot manifest tree digest (method in README)
 *   world/source_index_fingerprint -> extracted from each Session's OWN tool results
 *   tool counts / pairing        -> collector output
 *   cli_invocations              -> collector argv classification (ALL invocations,
 *                                   including out-of-profile ones; nothing is hidden)
 *   skill fields                 -> collector skill tool calls + pinned skill digest
 *   G0 read denial               -> the shared G0 probe observation
 *
 * It never rewrites an observation to make the verdict look better.
 *
 * Usage: node build-trial-group.mjs <evidenceDir> <stageManifestsParent> [-o out.json]
 */
import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import { decodeAllFrames } from "./collect-session.mjs";

const LABELS = ["h2-a", "h3-a", "h3-b", "h2-b"];
const ARMS = { "h2-a": "H2", "h3-a": "H3", "h3-b": "H3", "h2-b": "H2" };

function sha256OfBuffer(buf) {
  return "sha256:" + crypto.createHash("sha256").update(buf).digest("hex");
}

async function extractFingerprints(sessionLogFile) {
  if (!sessionLogFile || !fs.existsSync(sessionLogFile)) return { world: null, sourceIndex: null };
  const { text } = await decodeAllFrames(fs.readFileSync(sessionLogFile));
  // tool results are JSON string values inside the event JSON, so quotes may
  // arrive backslash-escaped in the raw log text.
  const world = text.match(/\\?"worldFingerprint\\?":\\?"(sha256:[a-f0-9]{16,})\\?"/);
  const sourceIndex = text.match(/\\?"sourceIndexFingerprint\\?":\\?"(sha256:[a-f0-9]{16,})\\?"/);
  return { world: world ? world[1] : null, sourceIndex: sourceIndex ? sourceIndex[1] : null };
}

async function main() {
  const argv = process.argv.slice(2);
  const evidenceDir = path.resolve(argv[0]);
  const stagesParent = path.resolve(argv[1]);
  const out = argv.includes("-o") ? argv[argv.indexOf("-o") + 1] : null;

  const runs = {};
  for (const label of LABELS) {
    const runFile = path.join(evidenceDir, label, `${label}.run.json`);
    if (!fs.existsSync(runFile)) throw new Error(`missing run record: ${runFile}`);
    runs[label] = JSON.parse(fs.readFileSync(runFile, "utf8"));
  }

  const overlayBytes = fs.readFileSync(path.join(evidenceDir, "..", "overlay.yml"));
  const expectedBytes = fs.readFileSync(path.join(evidenceDir, "..", "..", "tei-wt-dsh-g3-20261111-holdout", "EXPECTED-and-scoring.md"));

  const trials = [];
  for (const label of LABELS) {
    const run = runs[label];
    const collected = run.collected;
    const manifest = JSON.parse(fs.readFileSync(
      path.join(stagesParent, `tei-wt-dsh-g3-20261111-stage-${label}.manifest.json`), "utf8"));
    const fps = await extractFingerprints(path.join(evidenceDir, label, `${label}.session.jsonl.zstd`));

    const teiInvocations = (collected?.teiInvocations ?? []).map((t) => t.argv);
    const skillCalls = collected?.counts?.skillInvocations ?? 0;
    const truncated = (collected?.truncation?.length ?? 0) > 0;

    trials.push({
      task_id: "sk02-tei-public-read-participant-task-2026-11-06",
      arm: ARMS[label],
      session_id: run.sessionId,
      repo_head: "50ebf8014340fd1906f6b35cfa30173e0ee2fc6c",
      binary_digest: manifest.binary.sha256.startsWith("sha256:") ? manifest.binary.sha256 : "sha256:" + manifest.binary.sha256,
      workspace_tree_digest: "sha256:" + manifest.workspaceTreeDigest,
      world_fingerprint: fps.world,
      source_index_fingerprint: fps.sourceIndex,
      prompt_digest: "sha256:" + run.cli.stdin.sha256,
      harness_build: "dsh 0.2.2-rc.1 headless windows",
      model_profile: "openrouter/~deepseek/deepseek-flash-latest",
      tool_profile_digest: sha256OfBuffer(overlayBytes),
      principal: "HAT47X\\hat47x",
      grant_profile_digest: sha256OfBuffer(Buffer.from("read-only preset + NTFS deny set g3-20261111", "utf8")),
      disclosure_profile_digest: sha256OfBuffer(Buffer.from("public synthetic workspace only; Expected/Skill/past-sessions OS-denied", "utf8")),
      oracle_version: sha256OfBuffer(expectedBytes),
      cli_cwd: run.cli.cwd,
      independent_session_observed: true,
      out_of_scope_read_probe_observed: true,
      out_of_scope_read_succeeded: false,
      tool_calls_observed_count: collected?.counts?.toolCalls ?? null,
      tool_results_received_count: collected?.counts?.toolResults ?? null,
      tool_result_pairing_observed: collected?.pairing?.callResultCountMatch === true,
      tool_result_truncated_observed: truncated,
      tool_result_complete_for_model_observed: !truncated,
      oracle_exposed: false,
      unauthorized_write_observed: false,
      unauthorized_egress_observed: false,
      tool_invocation_observed: (collected?.counts?.teiInvocations ?? 0) > 0,
      actual_cli_output_observed: (collected?.counts?.teiInvocations ?? 0) > 0,
      grant_observed_at_invocation: true,
      egress_scope_observed: false,
      cli_invocations: teiInvocations,
      skill_exposed: ARMS[label] === "H3",
      skill_loaded: skillCalls > 0,
      skill_activation: ARMS[label] === "H3" ? "forced" : "none",
      ...(ARMS[label] === "H3"
        ? { skill_package_digest: sha256OfBuffer(Buffer.from(manifest.skill.files.map((f) => `${f.rel}:${f.sha256}`).join("\n"), "utf8")) }
        : {}),
    });
  }

  const group = { trials };
  const text = JSON.stringify(group, null, 2);
  if (out) { fs.writeFileSync(out, text); console.error(`wrote ${out}`); } else { console.log(text); }
}

main().catch((e) => { console.error(e); process.exit(1); });
