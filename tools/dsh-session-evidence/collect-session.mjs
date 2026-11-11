#!/usr/bin/env node
"use strict";
/*
 * DSH Session evidence collector (TEI #2509 / TAI #168 verification base).
 *
 * Reads a persisted DeepSeek Harness Session log (`session.v4.jsonl.zstd`),
 * decodes EVERY concatenated zstd frame (a single-frame decode is a known
 * false-negative: DSH writes the log as a chain of frames), and correlates
 *   Session id / cwd / preset / sandbox / approval
 *   user task bytes
 *   tool calls  <->  tool results      (by callId, with pairing report)
 *   assistant steps (provider / model alias / provider-resolved model)
 *   TEI CLI invocations observed in tool arguments (argv family + workspace)
 *   Product Skill tool invocations (name + received bytes)
 *   final turn reason and final assistant text
 *
 * It deliberately does NOT decide correctness. It records observations and
 * flags the independent failure conditions required by the SK-02 runbook:
 *   - tool call/result count mismatch (e.g. the historical 56 calls / 55 results)
 *   - suspected Tool Result truncation (result at the pruner threshold, or a
 *     JSON payload that no longer parses)
 *   - TEI argv outside the initially selected read-only command family
 *   - mixed -workspace targets inside one Session
 *
 * Usage:
 *   node collect-session.mjs <session.v4.jsonl.zstd | sessionDir> [-o out.json]
 *                        [--task-file <participant task bytes>]
 */
import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import zlib from "node:zlib";
import { Readable } from "node:stream";
import { pathToFileURL } from "node:url";

const ZSTD_MAGIC = Buffer.from([0x28, 0xb5, 0x2f, 0xfd]);

/** Decode every concatenated zstd frame. Single-frame decoders silently drop
 *  the remaining frames, which previously produced a false "log has no
 *  canary" negative (TEI #2515 H6-E6). */
export async function decodeAllFrames(buf) {
  const parts = [];
  let offset = 0;
  let frames = 0;
  const gaps = [];
  while (offset < buf.length) {
    if (!buf.subarray(offset, offset + 4).equals(ZSTD_MAGIC)) {
      const next = buf.indexOf(ZSTD_MAGIC, offset + 1);
      if (next < 0) {
        if (offset < buf.length) gaps.push({ at: offset, bytes: buf.length - offset });
        break;
      }
      gaps.push({ at: offset, bytes: next - offset });
      offset = next;
    }
    const stream = zlib.createZstdDecompress();
    let out = Buffer.alloc(0);
    stream.on("data", (c) => { out = Buffer.concat([out, c]); });
    await new Promise((resolve, reject) => {
      Readable.from([buf.subarray(offset)]).pipe(stream);
      stream.on("end", resolve);
      stream.on("error", reject);
    });
    parts.push(out);
    frames += 1;
    const consumed = stream.bytesWritten;
    if (!consumed) break;
    offset += consumed;
  }
  return { text: Buffer.concat(parts).toString("utf8"), frames, gaps };
}

export function sha256hex(buf) {
  return crypto.createHash("sha256").update(buf).digest("hex");
}

/** Resolve simple PowerShell variable assignments inside one command string
 *  (`$exe = "C:\\...\\tei.exe"; $ws = "..."; & $exe model catalog -workspace $ws`).
 *  The Harness model in this experiment drives the CLI through such
 *  variables, so an argv classifier that cannot resolve them would report
 *  zero TEI invocations. Only straight-line string assignments are handled;
 *  anything else is left untouched. */
export function resolvePowerShellVars(command) {
  const vars = new Map();
  const assignRe = /\$([A-Za-z_][\w]*)\s*=\s*(?:"([^"]*)"|'([^']*)')/g;
  let m;
  while ((m = assignRe.exec(command)) !== null) {
    vars.set(m[1], m[2] ?? m[3] ?? "");
  }
  if (vars.size === 0) return command;
  let out = command;
  for (const [name, value] of vars) {
    // replace usages only, never the assignment's own left-hand side
    out = out.split(new RegExp(`\\$${name}(?!\\s*=)`, "g")).join(value);
  }
  return out;
}

/** Tokenize a shell command into argv. Handles double quotes the way the
 *  PowerShell tool calls in this experiment write them. */
export function tokenizeCommand(command) {
  const tokens = [];
  let current = "";
  let quote = null;
  let has = false;
  for (let i = 0; i < command.length; i++) {
    const c = command[i];
    if (quote) {
      if (c === quote) { quote = null; continue; }
      current += c;
      continue;
    }
    if (c === '"') { quote = c; has = true; continue; }
    if (c === " " || c === "\t" || c === "\r" || c === "\n" || c === ";") {
      if (has || current) { tokens.push(current); current = ""; has = false; }
      continue;
    }
    current += c;
    has = true;
  }
  if (has || current) tokens.push(current);
  return tokens;
}

const ALLOWED_FAMILY = new Set(["version", "commands", "inspect", "trace", "validate"]);
// TEI flags that take a value; their value token is not the sub-command.
const VALUE_FLAGS = new Set(["-workspace", "--workspace", "-subject-kind", "--subject-kind",
  "-subject-id", "--subject-id", "-scope", "--scope", "-namespace", "--namespace",
  "-format", "--format", "-limit", "--limit", "-offset", "--offset"]);
const SEPARATORS = new Set(["&", ";", "|", "&&", "||", "2>&1", "2>&1|", "|"]);

function isTeiToken(token) {
  const exe = String(token).replace(/\\/g, "/").split("/").pop().toLowerCase();
  return exe === "tei" || exe === "tei.exe";
}

/** Unwrap a `cmd /c <command>` invocation (one Session in this experiment
 *  drives the CLI through cmd to avoid PowerShell's ConstrainedLanguage
 *  stderr noise). Returns the argv starting at the real command. A single
 *  quoted payload is re-tokenized, because cmd re-parses that string. */
export function unwrapCmd(argv) {
  if (Array.isArray(argv) && argv.length >= 3 &&
      /^cmd(\.exe)?$/i.test(String(argv[0])) &&
      /^\/[cC]$/.test(String(argv[1]))) {
    const rest = argv.slice(2);
    if (rest.length === 1 && /\s/.test(rest[0])) return tokenizeCommand(rest[0]);
    return rest;
  }
  return argv;
}

/** Classify one argv against the initially selected read-only family.
 *  Returns {family, tei, workspace, command}. The tei executable may sit at
 *  argv[0] or follow a call operator / separator (the model in this
 *  experiment drives the CLI as `$exe = ...; & $exe <cmd> ...`). A token that
 *  merely mentions tei as an argument is not treated as an invocation. */
export function classifyTeiArgv(argv) {
  if (!Array.isArray(argv) || argv.length === 0) return { family: "unobserved", tei: false, workspace: null };
  let start = -1;
  for (let i = 0; i < argv.length; i++) {
    const t = argv[i];
    if (typeof t !== "string" || t.length === 0) return { family: "unobserved", tei: false, workspace: null };
    if (isTeiToken(t) && (i === 0 || ["&", ";", "|", "&&", "||"].includes(argv[i - 1]))) { start = i; break; }
  }
  if (start < 0) return { family: "not-tei", tei: false, workspace: null, argvStart: -1 };
  const rest = argv.slice(start + 1);
  let cmd = null;
  let workspace = null;
  const args = [];
  for (let i = 0; i < rest.length; i++) {
    const t = rest[i];
    if (SEPARATORS.has(t)) break;
    if (VALUE_FLAGS.has(t)) {
      const value = rest[i + 1] ?? null;
      if (t === "-workspace" || t === "--workspace") workspace = value;
      args.push(t, value);
      i++;
      continue;
    }
    if (t.startsWith("-workspace=") || t.startsWith("--workspace=")) {
      workspace = t.slice(t.indexOf("=") + 1);
      args.push(t);
      continue;
    }
    if (t.startsWith("-")) { args.push(t); continue; }
    if (cmd === null) { cmd = t; continue; }
    args.push(t);
  }
  let family;
  if (cmd === "model" && args[0] === "catalog") family = "allowed-family";
  else if (cmd && ALLOWED_FAMILY.has(cmd)) family = "allowed-family";
  else if (cmd === null) family = "no-command";
  else family = "outside-initial-readonly-profile";
  return { family, tei: true, workspace, command: cmd, argvStart: start };
}

function safeJsonParse(s) {
  try { return { ok: true, value: JSON.parse(s) }; } catch { return { ok: false, value: null }; }
}

export function collectSession(logText, options = {}) {
  const taskDigest = options.taskDigest ? String(options.taskDigest) : null;
  const lines = logText.split("\n").filter((l) => l.length > 0);
  const events = [];
  let unparseableLines = 0;
  for (const line of lines) {
    const parsed = safeJsonParse(line);
    if (!parsed.ok) { unparseableLines += 1; continue; }
    events.push(parsed.value);
  }

  // The session header carries id/cwd/createdAt at the top level, unlike the
  // data-wrapped domain events.
  const sessionHeader = events.find((e) => e.type === "session") ?? null;
  const session = sessionHeader
    ? { id: sessionHeader.id, cwd: sessionHeader.cwd, createdAt: sessionHeader.createdAt }
    : null;
  const presets = events.filter((e) => e.type === "permission/preset").map((e) => ({ seq: e.seq, ...e.data }));
  const sandboxModes = events.filter((e) => e.type === "sandbox/mode").map((e) => ({ seq: e.seq, ...e.data }));
  const approvalPolicies = events.filter((e) => e.type === "approval/policy").map((e) => ({ seq: e.seq, ...e.data }));

  const userMessages = events.filter((e) => e.type === "user/message").map((e) => {
    const text = (e.data?.content ?? []).filter((c) => c.type === "text").map((c) => c.text).join("\n");
    const bytes = Buffer.byteLength(text, "utf8");
    return { seq: e.seq, bytes, sha256: sha256hex(Buffer.from(text, "utf8")), text };
  });

  const toolCalls = [];
  for (const e of events) {
    if (e.type !== "tool/call") continue;
    const args = safeJsonParse(e.data?.arguments ?? "");
    toolCalls.push({
      seq: e.seq,
      callId: e.data?.callId ?? null,
      turn: e.data?.turn ?? null,
      step: e.data?.step ?? null,
      tool: e.data?.name ?? null,
      argumentsRaw: e.data?.arguments ?? "",
      arguments: args.ok ? args.value : null,
      argumentsParseError: args.ok ? null : "MALFORMED_TOOL_ARGUMENTS",
    });
  }

  const toolResults = [];
  for (const e of events) {
    if (e.type !== "tool/result") continue;
    const msg = e.data?.message ?? {};
    const text = (msg.content ?? []).filter((c) => c.type === "text").map((c) => c.text).join("\n");
    const buf = Buffer.from(text, "utf8");
    toolResults.push({
      seq: e.seq,
      callId: msg.toolCallId ?? msg.source?.callId ?? null,
      resultId: msg.id ?? null,
      isError: msg.isError === true,
      resultBytes: buf.length,
      resultSha256: sha256hex(buf),
      cwd: e.data?.meta?.cwd ?? null,
      text,
    });
  }

  // --- pairing ------------------------------------------------------------
  const callIds = toolCalls.map((c) => c.callId);
  const resultByCall = new Map();
  for (const r of toolResults) {
    if (r.callId == null) continue;
    if (!resultByCall.has(r.callId)) resultByCall.set(r.callId, r);
  }
  const unpairedCalls = toolCalls.filter((c) => !resultByCall.has(c.callId)).map((c) => c.callId);
  const orphanResults = toolResults.filter((r) => r.callId != null && !callIds.includes(r.callId)).map((r) => r.callId);
  const duplicateResultCallIds = [...new Set(toolResults.map((r) => r.callId).filter((id, i, a) => id != null && a.indexOf(id) !== i))];

  // --- per-call joined view ----------------------------------------------
  const calls = toolCalls.map((c) => {
    const r = resultByCall.get(c.callId) ?? null;
    const row = {
      seq: c.seq, callId: c.callId, tool: c.tool, turn: c.turn, step: c.step,
      resultSeq: r ? r.seq : null, resultReceived: r !== null,
      resultBytes: r ? r.resultBytes : null, resultSha256: r ? r.resultSha256 : null,
      resultIsError: r ? r.isError : null,
      argumentsParseError: c.argumentsParseError,
    };
    if (c.tool === "pwsh" && c.arguments && typeof c.arguments.command === "string") {
      const argv = unwrapCmd(tokenizeCommand(resolvePowerShellVars(c.arguments.command)));
      const cls = classifyTeiArgv(argv);
      if (cls.tei) {
        // record the invocation from the executable onward, so the stored argv
        // is the command line that actually ran (assignments/operators dropped)
        const invocationArgv = cls.argvStart >= 0 ? argv.slice(cls.argvStart) : argv;
        row.teiInvocation = { argv: invocationArgv, family: cls.family, command: cls.command ?? null, workspace: cls.workspace };
      }
    }
    if (c.tool === "skill" && c.arguments && typeof c.arguments.name === "string") {
      row.skillInvocation = { name: c.arguments.name };
    }
    return row;
  });

  // --- truncation ---------------------------------------------------------
  // DSH's tool-result pruner (default thresholdChars 8192, head 4096, tail 1024)
  // can replace long results before the model sees them. A result whose stdout
  // part sits exactly at the threshold, or whose JSON payload no longer
  // parses, is recorded as a SUSPECTED truncation, not as a complete
  // observation. PowerShell tool replies interleave a "[stderr]" section
  // (observed: ConstrainedLanguage noise) and the operator wrapper appends
  // "EXIT=<code>"; stdout and stderr are separated first so neither a mixed
  // reply nor an exit-suffixed JSON is mistaken for a truncated one, and so
  // the JSON payload is parsed from the stdout part alone.
  const STDERR_MARKERS = ["\n[stderr]\n", "[stderr]\n"];
  const EXIT_SUFFIX = /(^|\n)EXIT=-?\d+\s*$/;
  const truncation = [];
  const callsOut = [];
  for (const c of calls) {
    if (!c.resultReceived) { callsOut.push(c); continue; }
    const r = resultByCall.get(c.callId);
    let markerAt = -1;
    let markerLen = 0;
    for (const m of STDERR_MARKERS) {
      const at = r.text.indexOf(m);
      if (at >= 0 && (markerAt < 0 || at < markerAt)) { markerAt = at; markerLen = m.length; }
    }
    const stdoutText = markerAt >= 0 ? r.text.slice(0, markerAt) : r.text;
    const stderrText = markerAt >= 0 ? r.text.slice(markerAt + markerLen) : null;
    const stdoutBytes = Buffer.byteLength(stdoutText, "utf8");
    const looksJson = /^\s*[[{]/.test(stdoutText);
    const jsonOk = looksJson ? safeJsonParse(stdoutText).ok : null;
    const exitSuffix = EXIT_SUFFIX.test(stdoutText);
    const atThreshold = stdoutBytes === 8192;
    const suspectedTruncation = atThreshold || (looksJson && jsonOk === false && !exitSuffix && stderrText === null);
    const row = {
      ...c,
      stdoutBytes,
      stdoutSha256: sha256hex(Buffer.from(stdoutText, "utf8")),
      stderrBytes: stderrText === null ? null : Buffer.byteLength(stderrText, "utf8"),
      stderrPresent: stderrText !== null,
      exitSuffix,
      suspectedTruncation,
    };
    callsOut.push(row);
    if (suspectedTruncation) {
      truncation.push({
        callId: c.callId, tool: c.tool, resultBytes: r.resultBytes,
        stdoutBytes, atPrunerThreshold: atThreshold,
        jsonPayloadParse: jsonOk,
        stdoutTail: stdoutText.slice(-80),
      });
    }
  }
  calls.length = 0;
  calls.push(...callsOut);

  // --- TEI invocation summary --------------------------------------------
  const teiInvocations = calls.filter((c) => c.teiInvocation).map((c) => ({
    callId: c.callId, argv: c.teiInvocation.argv, family: c.teiInvocation.family,
    command: c.teiInvocation.command, workspace: c.teiInvocation.workspace,
    resultReceived: c.resultReceived, resultBytes: c.resultBytes,
  }));
  const outsideFamily = teiInvocations.filter((t) => t.family !== "allowed-family");
  const workspaces = [...new Set(teiInvocations.map((t) => t.workspace).filter((w) => typeof w === "string" && w.length > 0))];

  // --- skill --------------------------------------------------------------
  const skillCalls = calls.filter((c) => c.skillInvocation).map((c) => ({
    callId: c.callId, name: c.skillInvocation.name, resultReceived: c.resultReceived,
    resultBytes: c.resultBytes, resultSha256: c.resultSha256,
  }));

  // --- assistant steps / model identity -----------------------------------
  const steps = [];
  for (const e of events) {
    if (e.type !== "assistant/message") continue;
    const src = e.data?.message?.source ?? {};
    const replay = src.replayState?.response ?? {};
    const textBlocks = (e.data?.message?.content ?? []).filter((c) => c.type === "text").map((c) => c.text);
    steps.push({
      seq: e.seq, turn: e.data?.turn ?? null, step: e.data?.step ?? null,
      provider: src.provider ?? null, modelAlias: src.model ?? null,
      responseModel: replay.responseModel ?? null, responseId: replay.responseId ?? null,
      stopReason: replay.stopReason ?? null,
      textBytes: Buffer.byteLength(textBlocks.join("\n"), "utf8"),
    });
  }

  const finalAssistant = [...events].reverse().find((e) => e.type === "assistant/message");
  const finalText = finalAssistant
    ? (finalAssistant.data?.message?.content ?? []).filter((c) => c.type === "text").map((c) => c.text).join("\n")
    : null;
  const finalBuf = finalText === null ? null : Buffer.from(finalText, "utf8");
  const turnEnd = events.find((e) => e.type === "turn/end");

  const participantTask = userMessages.find((m) => taskDigest && m.sha256 === taskDigest) ?? null;

  return {
    schema: "tei.tools.dsh-session-evidence/v1",
    sessionId: session?.id ?? null,
    cwd: session?.cwd ?? null,
    createdAt: session?.createdAt ?? null,
    presets, sandboxModes, approvalPolicies,
    userMessages: userMessages.map(({ text, ...rest }) => rest),
    participantTaskMatched: participantTask ? { seq: participantTask.seq, sha256: participantTask.sha256, bytes: participantTask.bytes } : null,
    counts: {
      events: events.length,
      unparseableLines,
      toolCalls: toolCalls.length,
      toolResults: toolResults.length,
      teiInvocations: teiInvocations.length,
      skillInvocations: skillCalls.length,
      steps: steps.length,
    },
    pairing: {
      unpairedCallIds: unpairedCalls,
      orphanResultCallIds: orphanResults,
      duplicateResultCallIds,
      callResultCountMatch: toolCalls.length === toolResults.length && unpairedCalls.length === 0,
    },
    truncation,
    teiInvocations,
    outsideInitialFamily: outsideFamily.map((t) => ({ callId: t.callId, argv: t.argv, family: t.family })),
    workspaceTargets: workspaces,
    mixedWorkspaceTargets: workspaces.length > 1,
    skillCalls,
    steps,
    final: {
      turnEndReason: turnEnd?.data?.reason ?? null,
      finalAnswerBytes: finalBuf ? finalBuf.length : null,
      finalAnswerSha256: finalBuf ? sha256hex(finalBuf) : null,
    },
    calls: calls.map((c) => ({ ...c, argumentsRaw: undefined })),
    limits: [
      "Observations come from the persisted Session log only; stdout/stderr streams and process-level facts need the run capture.",
      "Truncation is suspected, not proven: the pruner threshold and JSON parse failure are heuristics.",
      "This collector does not evaluate answer correctness or Skill benefit.",
    ],
  };
}

async function main() {
  const argv = process.argv.slice(2);
  if (argv.length === 0 || argv[0] === "-h" || argv[0] === "--help") {
    console.error("usage: node collect-session.mjs <session.v4.jsonl.zstd|dir> [-o out.json] [--task-file <task>]");
    process.exit(2);
  }
  let input = argv[0];
  let out = null;
  let taskFile = null;
  for (let i = 1; i < argv.length; i++) {
    if (argv[i] === "-o") { out = argv[++i]; continue; }
    if (argv[i] === "--task-file") { taskFile = argv[++i]; continue; }
  }
  if (fs.existsSync(input) && fs.statSync(input).isDirectory()) {
    input = path.join(input, "session.v4.jsonl.zstd");
  }
  const raw = fs.readFileSync(input);
  const decoded = await decodeAllFrames(raw);
  const taskDigest = taskFile ? sha256hex(fs.readFileSync(taskFile)) : null;
  const collected = collectSession(decoded.text, { taskDigest });
  const result = {
    sourceFile: path.resolve(input),
    sourceBytes: raw.length,
    zstdFrames: decoded.frames,
    interFrameGaps: decoded.gaps,
    ...collected,
  };
  const text = JSON.stringify(result, null, 2);
  if (out) fs.writeFileSync(out, text);
  else console.log(text);
  if (out) console.error(`wrote ${out} (${text.length} bytes)`);
}

const invokedDirectly = process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href;
if (invokedDirectly) {
  main().catch((e) => { console.error(e); process.exit(1); });
}
