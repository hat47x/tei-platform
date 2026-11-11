#!/usr/bin/env node
"use strict";
/*
 * DSH headless session runner for the H2/H3 comparison (TEI #2509 / TAI #168).
 *
 * Runs ONE independent DSH Session against a pre-built participant root and
 * captures the complete evidence chain the SK-02 runbook requires:
 *   - the exact CLI, launch arguments, working directory and exit status
 *   - the stdout run-event stream (--json) and stderr, with byte digests
 *   - the Session id, per-step usage and final answer text from the stream
 *   - the persisted Session log (all zstd frames) plus the collector output
 *
 * The runner never edits the participant root and never passes a Skill path,
 * Expected answer, or Reviewer material to the Session.
 *
 * Usage:
 *   node run-dsh-session.mjs --root <slot> --overlay <overlay.yml> \
 *     --task <task.txt> --label <h2-a> --out <evidenceDir> [--arm H2|H3] \
 *     [--dsh-bin <path to dsh.cmd>] [--timeout-ms 1800000]
 */
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { pathToFileURL } from "node:url";
import { decodeAllFrames, collectSession, sha256hex } from "./collect-session.mjs";

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

/** Resolve the dsh launcher: explicit path, then PATH, then the installed
 *  DeepSeek Harness runtime location. The resolved binary is recorded so the
 *  launch evidence names the actual executable. */
export function resolveDshBin(explicit) {
  if (explicit) return explicit;
  const names = ["dsh.cmd", "dsh.exe", "dsh"];
  for (const dir of (process.env.PATH ?? "").split(path.delimiter)) {
    if (!dir) continue;
    for (const name of names) {
      const candidate = path.join(dir, name);
      if (fs.existsSync(candidate)) return candidate;
    }
  }
  const installed = "C:\\Users\\yhata\\AppData\\Local\\Programs\\DeepSeek Harness\\resources\\runtime\\cli\\bin\\dsh.cmd";
  if (fs.existsSync(installed)) return installed;
  return "dsh";
}

/** Windows `shell: true` concatenates arguments without quoting, so a
 *  launcher path containing spaces (the installed DSH runtime path has one)
 *  must be wrapped in double quotes for cmd.exe. */
export function shellCommandFor(dshBin) {
  return /[\s"]/.test(dshBin) ? `"${dshBin}"` : dshBin;
}

function findSessionLog(storeRoot, sessionId) {
  const sessionsRoot = path.join(storeRoot, "sessions");
  if (!fs.existsSync(sessionsRoot)) return null;
  for (const entry of fs.readdirSync(sessionsRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const candidate = path.join(sessionsRoot, entry.name, sessionId, "session.v4.jsonl.zstd");
    if (fs.existsSync(candidate)) return candidate;
  }
  return null;
}

/** Locate the persisted Session log inside the (possibly redirected) store.
 *  DSH may flush the final log bytes slightly after the headless process
 *  exits, so this retries briefly instead of recording a false "no log".
 *  The persisted directory is named exactly by the Session id. */
async function findSessionLogWithRetry(storeRoot, sessionId, attempts = 30, delayMs = 1000) {
  for (let i = 0; i < attempts; i++) {
    const found = findSessionLog(storeRoot, sessionId);
    if (found) return found;
    await new Promise((r) => setTimeout(r, delayMs));
  }
  return null;
}

async function main() {
  const argv = process.argv.slice(2);
  const collectOnly = argv.includes("--collect-only");
  const parsedArgv = argv.filter((a) => a !== "--collect-only");
  const args = parseArgs(parsedArgv);
  const root = args.root ? path.resolve(args.root) : null;
  const overlay = args.overlay ? path.resolve(args.overlay) : null;
  const taskFile = path.resolve(args.task);
  const outDir = path.resolve(args.out);
  const label = args.label;
  const arm = args.arm ?? null;
  const timeoutMs = Number(args["timeout-ms"] ?? 1800000);
  const dshBin = resolveDshBin(args["dsh-bin"]);
  if (!taskFile || !outDir || !label) {
    console.error("missing required argument");
    process.exit(2);
  }
  if (!collectOnly && (!root || !overlay)) {
    console.error("missing required argument (--root/--overlay, or --collect-only)");
    process.exit(2);
  }
  fs.mkdirSync(outDir, { recursive: true });

  const taskBytes = fs.readFileSync(taskFile);
  const env = { ...process.env };
  let cliArgs = [
    "--profile", "headless",
    "--patch", overlay,
    "--permission-mode", "read-only",
    "--approval-policy", "never",
    "--json",
  ];
  // Windows shell:true concatenates without quoting, so a launcher path with
  // spaces (the installed DSH runtime path has one) must be quoted here.
  let shellCommand = shellCommandFor(dshBin);
  let startedAt = new Date().toISOString();
  let finishedAt = startedAt;
  let elapsedMs = 0;
  let stdout;
  let stderr = Buffer.alloc(0);
  let exitCode = null;
  let childSignal = null;
  let childError = null;

  if (collectOnly) {
    // Re-build the run record from saved artifacts (no new Session is started).
    const streamPath = path.resolve(args["from-stream"]);
    stdout = fs.readFileSync(streamPath);
    startedAt = new Date(fs.statSync(streamPath).mtime).toISOString();
  } else {
    const t0 = Date.now();
    const child = spawnSync(shellCommand, cliArgs, {
      cwd: root,
      input: taskBytes,
      shell: true,
      env,
      timeout: timeoutMs,
      maxBuffer: 512 * 1024 * 1024,
      encoding: "buffer",
    });
    elapsedMs = Date.now() - t0;
    finishedAt = new Date().toISOString();
    stdout = child.stdout ?? Buffer.alloc(0);
    stderr = child.stderr ?? Buffer.alloc(0);
    exitCode = child.status;
    childSignal = child.signal ?? null;
    childError = child.error ? String(child.error.message ?? child.error) : null;
    fs.writeFileSync(path.join(outDir, `${label}.stdout.jsonl`), stdout);
    fs.writeFileSync(path.join(outDir, `${label}.stderr.txt`), stderr);
  }

  // --- parse the run-event stream -----------------------------------------
  const streamLines = stdout.toString("utf8").split("\n").filter((l) => l.length > 0);
  const events = [];
  let streamUnparseable = 0;
  for (const line of streamLines) {
    try { events.push(JSON.parse(line)); } catch { streamUnparseable += 1; }
  }
  const typeCounts = {};
  for (const e of events) typeCounts[e.type] = (typeCounts[e.type] ?? 0) + 1;
  const sessionEvent = events.find((e) => e.type === "session");
  const finalEvent = [...events].reverse().find((e) => e.type === "final");
  const usageSteps = events.filter((e) => e.type === "status" && e.phase === "step_end" && e.usage).map((e) => ({ step: e.step, turn: e.turn, ...e.usage }));
  const usageSum = usageSteps.reduce((acc, u) => ({
    inputTokens: acc.inputTokens + (u.inputTokens ?? 0),
    outputTokens: acc.outputTokens + (u.outputTokens ?? 0),
    totalTokens: acc.totalTokens + (u.totalTokens ?? 0),
  }), { inputTokens: 0, outputTokens: 0, totalTokens: 0 });

  const sessionId = sessionEvent?.sessionId ?? finalEvent?.sessionId ?? null;

  // --- collect the persisted session log ----------------------------------
  let collected = null;
  let sessionLog = null;
  if (sessionId) {
    const storeRoot = args.store ? path.resolve(args.store) : (env.DSH_HOME ?? path.join(env.USERPROFILE ?? "", ".dsh"));
    sessionLog = await findSessionLogWithRetry(storeRoot, sessionId);
    if (sessionLog) {
      const raw = fs.readFileSync(sessionLog);
      fs.writeFileSync(path.join(outDir, `${label}.session.jsonl.zstd`), raw);
      const decoded = await decodeAllFrames(raw);
      collected = collectSession(decoded.text, { taskDigest: sha256hex(taskBytes) });
      collected.zstdFrames = decoded.frames;
      collected.interFrameGaps = decoded.gaps;
      collected.sourceBytes = raw.length;
    }
  }

  const record = {
    schema: "tei.tools.dsh-session-run/v1",
    label,
    arm,
    startedAt, finishedAt, elapsedMs,
    cli: {
      command: dshBin,
      shellCommand,
      args: cliArgs,
      cwd: root,
      stdin: { file: taskFile, bytes: taskBytes.length, sha256: sha256hex(taskBytes) },
      exitCode,
      signal: childSignal,
      error: childError,
      collectOnly,
    },
    env: {
      DSH_HOME: env.DSH_HOME ?? null,
      note: "Credential material is never written to this record; the Provider key resolves through the DSH credentials service.",
    },
    stream: {
      stdoutBytes: stdout.length,
      stdoutSha256: sha256hex(stdout),
      stderrBytes: stderr.length,
      stderrSha256: sha256hex(stderr),
      events: events.length,
      unparseableLines: streamUnparseable,
      typeCounts,
    },
    sessionId,
    finalFromStream: finalEvent
      ? { reason: finalEvent.reason ?? null, textBytes: Buffer.byteLength(finalEvent.text ?? "", "utf8"), textSha256: sha256hex(Buffer.from(finalEvent.text ?? "", "utf8")) }
      : null,
    usageFromStream: { steps: usageSteps, sum: usageSum },
    persistedLog: sessionLog ? { file: sessionLog, collected: true } : { file: null, collected: false },
    sessionStore: args.store ? path.resolve(args.store) : (env.DSH_HOME ?? null),
    collected,
  };
  fs.writeFileSync(path.join(outDir, `${label}.run.json`), JSON.stringify(record, null, 2));
  if (finalEvent?.text != null) {
    fs.writeFileSync(path.join(outDir, `${label}.final.txt`), finalEvent.text, "utf8");
  }
  console.log(JSON.stringify({
    label, sessionId, exitCode, elapsedMs,
    streamEvents: events.length,
    toolCalls: collected?.counts?.toolCalls ?? null,
    toolResults: collected?.counts?.toolResults ?? null,
    pairingOk: collected?.pairing?.callResultCountMatch ?? null,
    truncation: collected?.truncation?.length ?? null,
    outsideFamily: collected?.outsideInitialFamily?.length ?? null,
    skillCalls: collected?.counts?.skillInvocations ?? null,
    finalBytes: record.finalFromStream?.textBytes ?? null,
  }, null, 2));
}

const invokedDirectly = process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href;
if (invokedDirectly) {
  main().catch((e) => { console.error(e); process.exit(1); });
}
