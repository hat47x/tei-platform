"use strict";
/*
 * Independent regression tests for collect-session.mjs (TEI #2509 / TAI #168).
 * Synthetic Session logs only: no real Harness session, Provider call, or
 * Product answer is used or produced here.
 */
import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import zlib from "node:zlib";
import { decodeAllFrames, collectSession, tokenizeCommand, resolvePowerShellVars, unwrapCmd, classifyTeiArgv, sha256hex } from "./collect-session.mjs";

function ev(obj) {
  return JSON.stringify(obj) + "\n";
}

function buildLog(frames) {
  return Buffer.concat(frames.map((t) => zlib.zstdCompressSync(Buffer.from(t, "utf8"))));
}

const SESSION_HEADER = ev({
  type: "session", version: 4, id: "session-synthetic-0001", createdAt: 1794000000000,
  cwd: "D:\\exp\\slot", isSeeded: false, delegationDepth: 0,
});

function assistantStep(step, toolCalls) {
  return ev({
    type: "assistant/message", seq: 100 + step, time: 1794000000000 + step,
    data: {
      turn: 1, step,
      message: {
        role: "assistant",
        content: [
          { type: "text", text: `step ${step} text` },
          ...toolCalls.map((tc) => ({ type: "tool-call", id: tc.callId, name: tc.tool, arguments: tc.arguments })),
        ],
        source: {
          kind: "model", provider: "openrouter", model: "~deepseek/deepseek-flash-latest",
          replayState: { response: { kind: "pi-ai", provider: "openrouter", model: "~deepseek/deepseek-flash-latest", responseModel: "deepseek/synthetic-flash-1017", responseId: "gen-synthetic", stopReason: "toolUse" } },
        },
      },
    },
  });
}

function toolCallEvent(seq, turn, step, callId, name, argumentsJson) {
  return ev({ type: "tool/call", seq, time: 1794000000000 + seq, data: { turn, step, callId, name, arguments: argumentsJson } });
}

function toolResultEvent(seq, turn, step, callId, text, isError = false) {
  return ev({
    type: "tool/result", seq, time: 1794000000000 + seq,
    data: {
      turn, step,
      message: {
        role: "tool", source: { kind: "tool", callId }, toolCallId: callId,
        content: [{ type: "text", text }], isError, id: `result-${callId}`,
      },
      meta: { cwd: "D:\\exp\\slot" },
    },
  });
}

test("decodeAllFrames expands every concatenated zstd frame", async () => {
  const buf = buildLog([ev({ type: "note", t: "a" }), ev({ type: "note", t: "b" }), ev({ type: "note", t: "c" })]);
  const single = zlib.zstdDecompressSync(buf).toString("utf8");
  assert.equal(single.split("\n").filter(Boolean).length, 1, "single-frame decode drops the tail frames");
  const { text, frames } = await decodeAllFrames(buf);
  assert.equal(frames, 3);
  assert.equal(text.split("\n").filter(Boolean).length, 3);
});

test("paired tool calls and results are correlated by callId", () => {
  const log = SESSION_HEADER
    + assistantStep(1, [{ callId: "call_a", tool: "pwsh", arguments: JSON.stringify({ command: "tei version -json" }) }])
    + toolCallEvent(10, 1, 1, "call_a", "pwsh", JSON.stringify({ command: "tei version -json" }))
    + toolResultEvent(11, 1, 1, "call_a", '{"schemaVersion":"tei.cli.version/v0"}')
    + ev({ type: "turn/end", seq: 50, data: { turn: 1, reason: { kind: "completed" } } });
  const out = collectSession(log);
  assert.equal(out.counts.toolCalls, 1);
  assert.equal(out.counts.toolResults, 1);
  assert.equal(out.pairing.callResultCountMatch, true);
  assert.equal(out.pairing.unpairedCallIds.length, 0);
  assert.equal(out.final.turnEndReason.kind, "completed");
  assert.equal(out.calls[0].resultReceived, true);
});

test("a call without a result is an unpaired observation, not a success", () => {
  const log = SESSION_HEADER
    + toolCallEvent(10, 1, 1, "call_x", "pwsh", JSON.stringify({ command: "tei commands" }))
    + toolResultEvent(11, 1, 1, "call_y", "orphan result")
    + ev({ type: "turn/end", seq: 50, data: { turn: 1, reason: { kind: "completed" } } });
  const out = collectSession(log);
  assert.equal(out.counts.toolCalls, 1);
  assert.equal(out.counts.toolResults, 1);
  assert.equal(out.pairing.callResultCountMatch, false, "counts match but correlation does not");
  assert.deepEqual(out.pairing.unpairedCallIds, ["call_x"]);
  assert.deepEqual(out.pairing.orphanResultCallIds, ["call_y"]);
});

test("count mismatch like the historical 56 calls / 55 results is recorded", () => {
  let log = SESSION_HEADER;
  for (let i = 0; i < 3; i++) {
    log += toolCallEvent(10 + i, 1, 1, `call_${i}`, "pwsh", JSON.stringify({ command: "tei commands" }));
    if (i < 2) log += toolResultEvent(20 + i, 1, 1, `call_${i}`, "ok");
  }
  const out = collectSession(log);
  assert.equal(out.counts.toolCalls, 3);
  assert.equal(out.counts.toolResults, 2);
  assert.equal(out.pairing.callResultCountMatch, false);
  assert.deepEqual(out.pairing.unpairedCallIds, ["call_2"]);
});

test("result at the pruner threshold with unparseable JSON is flagged as suspected truncation", () => {
  const prefix = '{"schemaVersion":"tei.cli.model-catalog/v0","entries":[';
  const truncated = prefix + "x".repeat(8192 - prefix.length);
  assert.equal(Buffer.byteLength(truncated, "utf8"), 8192);
  const log = SESSION_HEADER
    + toolCallEvent(10, 1, 1, "call_c", "pwsh", JSON.stringify({ command: "tei model catalog -workspace reference/workspace" }))
    + toolResultEvent(11, 1, 1, "call_c", truncated);
  const out = collectSession(log);
  assert.equal(out.truncation.length, 1);
  assert.equal(out.truncation[0].callId, "call_c");
  assert.equal(out.truncation[0].atPrunerThreshold, true);
  assert.equal(out.truncation[0].jsonPayloadParse, false);
});

test("a complete short JSON result is not flagged", () => {
  const log = SESSION_HEADER
    + toolCallEvent(10, 1, 1, "call_ok", "pwsh", JSON.stringify({ command: "tei version -json" }))
    + toolResultEvent(11, 1, 1, "call_ok", '{"schemaVersion":"tei.cli.version/v0"}');
  const out = collectSession(log);
  assert.equal(out.truncation.length, 0);
});

test("stdout and the PowerShell [stderr] section are separated before judging truncation", () => {
  const reply = '{"schemaVersion":"tei.cli.version/v0","ok":true}\n[stderr]\nCannotCreateTypeConstrainedLanguage noise';
  const log = SESSION_HEADER
    + toolCallEvent(10, 1, 1, "call_mix", "pwsh", JSON.stringify({ command: "tei version -json" }))
    + toolResultEvent(11, 1, 1, "call_mix", reply);
  const out = collectSession(log);
  assert.equal(out.truncation.length, 0, "a mixed stdout/stderr reply is not a truncated result");
  const row = out.calls.find((c) => c.callId === "call_mix");
  assert.equal(row.stderrPresent, true);
  assert.equal(row.suspectedTruncation, false);
  assert.equal(row.stdoutSha256, sha256hex(Buffer.from('{"schemaVersion":"tei.cli.version/v0","ok":true}', "utf8")));
});

test("a pruner-threshold stdout inside a mixed reply is still flagged", () => {
  const prefix = '{"schemaVersion":"tei.cli.model-catalog/v0","entries":[';
  const reply = prefix + "x".repeat(8192 - prefix.length) + "\n[stderr]\nnoise";
  const log = SESSION_HEADER
    + toolCallEvent(10, 1, 1, "call_cut", "pwsh", JSON.stringify({ command: "tei model catalog -workspace reference/workspace" }))
    + toolResultEvent(11, 1, 1, "call_cut", reply);
  const out = collectSession(log);
  assert.equal(out.truncation.length, 1);
  assert.equal(out.truncation[0].stdoutBytes, 8192);
  assert.equal(out.truncation[0].atPrunerThreshold, true);
});

test("an exit-suffixed JSON stdout is not mistaken for a truncated result", () => {
  const reply = '{"schemaVersion":"tei.cli.validate/v0","ok":true}\nEXIT=0\n[stderr]\nnoise';
  const log = SESSION_HEADER
    + toolCallEvent(10, 1, 1, "call_exit", "pwsh", JSON.stringify({ command: "tei validate -json" }))
    + toolResultEvent(11, 1, 1, "call_exit", reply);
  const out = collectSession(log);
  assert.equal(out.truncation.length, 0);
  const row = out.calls.find((c) => c.callId === "call_exit");
  assert.equal(row.exitSuffix, true);
  assert.equal(row.suspectedTruncation, false);
});

test("a stderr-only reply is not mistaken for a truncated result", () => {
  const reply = "[stderr]\nCannotCreateTypeConstrainedLanguage noise";
  const log = SESSION_HEADER
    + toolCallEvent(10, 1, 1, "call_err", "pwsh", JSON.stringify({ command: "Get-ChildItem" }))
    + toolResultEvent(11, 1, 1, "call_err", reply);
  const out = collectSession(log);
  assert.equal(out.truncation.length, 0);
  const row = out.calls.find((c) => c.callId === "call_err");
  assert.equal(row.stderrPresent, true);
  assert.equal(row.suspectedTruncation, false);
});

test("TEI argv family classification follows the initial read-only selection", () => {
  assert.equal(classifyTeiArgv([".\\bin\\tei.exe", "version", "-json"]).family, "allowed-family");
  assert.equal(classifyTeiArgv(["C:\\bin\\tei.exe", "model", "catalog", "-workspace", "reference/workspace"]).family, "allowed-family");
  assert.equal(classifyTeiArgv(["tei", "inspect", "-workspace=reference/workspace", "-subject-id", "sales.order"]).workspace, "reference/workspace");
  assert.equal(classifyTeiArgv(["tei", "validate", "--workspace", "reference/workspace", "-json"]).family, "allowed-family");
  assert.equal(classifyTeiArgv(["tei", "source-index", "-subject-kind", "semantic-data"]).family, "outside-initial-readonly-profile");
  assert.equal(classifyTeiArgv(["tei", "dbcheck"]).family, "outside-initial-readonly-profile");
  assert.equal(classifyTeiArgv(["git", "status"]).tei, false);
  assert.equal(classifyTeiArgv(["tei"]).family, "no-command");
});

test("out-of-family TEI invocation and mixed workspace targets are surfaced", () => {
  const log = SESSION_HEADER
    + toolCallEvent(10, 1, 1, "call_1", "pwsh", JSON.stringify({ command: ".\\bin\\tei.exe source-index -workspace reference/workspace" }))
    + toolResultEvent(11, 1, 1, "call_1", "{}")
    + toolCallEvent(12, 1, 2, "call_2", "pwsh", JSON.stringify({ command: ".\\bin\\tei.exe inspect -workspace other/workspace" }))
    + toolResultEvent(13, 1, 2, "call_2", "{}");
  const out = collectSession(log);
  assert.equal(out.counts.teiInvocations, 2);
  assert.equal(out.outsideInitialFamily.length, 1);
  assert.equal(out.outsideInitialFamily[0].argv[1], "source-index");
  assert.equal(out.workspaceTargets.length, 2);
  assert.equal(out.mixedWorkspaceTargets, true);
});

test("Skill tool invocation records received bytes without claiming correctness", () => {
  const skillText = "synthetic skill body";
  const log = SESSION_HEADER
    + assistantStep(1, [{ callId: "call_s", tool: "skill", arguments: JSON.stringify({ name: "yi-tei-modeling" }) }])
    + toolCallEvent(10, 1, 1, "call_s", "skill", JSON.stringify({ name: "yi-tei-modeling" }))
    + toolResultEvent(11, 1, 1, "call_s", skillText);
  const out = collectSession(log);
  assert.equal(out.counts.skillInvocations, 1);
  assert.equal(out.skillCalls[0].name, "yi-tei-modeling");
  assert.equal(out.skillCalls[0].resultBytes, Buffer.byteLength(skillText, "utf8"));
  assert.equal(out.skillCalls[0].resultSha256, sha256hex(Buffer.from(skillText, "utf8")));
});

test("participant task digest matching separates the fixed task from other prompts", () => {
  const task = "公開TEI参照Workspace（reference/workspace）にあるモデル要素 semantic-data:sales.order を調査してください。";
  const digest = sha256hex(Buffer.from(task, "utf8"));
  const log = SESSION_HEADER
    + ev({ type: "user/message", seq: 11, data: { content: [{ type: "text", text: task }], source: { kind: "user" }, role: "user", id: "u1" } })
    + ev({ type: "user/message", seq: 12, data: { content: [{ type: "text", text: "別の短い確認" }], source: { kind: "user" }, role: "user", id: "u2" } });
  const out = collectSession(log, { taskDigest: digest });
  assert.equal(out.participantTaskMatched.sha256, digest);
  assert.equal(out.userMessages.length, 2);
  const outNoDigest = collectSession(log);
  assert.equal(outNoDigest.participantTaskMatched, null);
});

test("tokenizeCommand handles quoted paths and equals-form flags", () => {
  assert.deepEqual(tokenizeCommand('tei.exe model catalog -workspace="C:\\a b\\ws" -json'),
    ["tei.exe", "model", "catalog", "-workspace=C:\\a b\\ws", "-json"]);
  assert.deepEqual(tokenizeCommand("  tei.exe   validate   -json  "),
    ["tei.exe", "validate", "-json"]);
});

test("PowerShell variable invocations resolve to a classifiable argv", () => {
  const command = '$exe = "D:\\slot\\bin\\tei.exe"; $ws = "D:\\slot\\reference\\workspace"; & $exe model catalog -workspace $ws 2>&1 | Out-String';
  const resolved = resolvePowerShellVars(command);
  const argv = tokenizeCommand(resolved);
  const cls = classifyTeiArgv(argv);
  assert.equal(cls.tei, true);
  assert.equal(cls.family, "allowed-family");
  assert.equal(cls.command, "model");
  assert.equal(cls.workspace, "D:\\slot\\reference\\workspace");
});

test("a variable-driven out-of-family command is still surfaced", () => {
  const command = '$exe = ".\\bin\\tei.exe"; $ws = ".\\reference\\workspace"; & $exe source-index -workspace $ws -subject-kind semantic-data';
  const cls = classifyTeiArgv(tokenizeCommand(resolvePowerShellVars(command)));
  assert.equal(cls.family, "outside-initial-readonly-profile");
  assert.equal(cls.command, "source-index");
});

test("commands without variables are unchanged by the resolver", () => {
  const command = ".\\bin\\tei.exe version -json 2>&1 | Out-String";
  assert.equal(resolvePowerShellVars(command), command);
});

test("a cmd /c wrapped invocation is unwrapped before classification", () => {
  const command = "cmd /c 'bin\\tei.exe version -json' 2>&1 | Out-String";
  const cls = classifyTeiArgv(unwrapCmd(tokenizeCommand(command)));
  assert.equal(cls.tei, true);
  assert.equal(cls.family, "allowed-family");
  assert.equal(cls.command, "version");
});

test("a cmd /c wrapped out-of-family command is still surfaced", () => {
  const command = 'cmd /c ".\\bin\\tei.exe workspace-files -workspace reference/workspace"';
  const cls = classifyTeiArgv(unwrapCmd(tokenizeCommand(command)));
  assert.equal(cls.family, "outside-initial-readonly-profile");
  assert.equal(cls.workspace, "reference/workspace");
});

test("full pipeline over a 3-frame synthetic log on disk", async () => {
  const frame1 = SESSION_HEADER + ev({ type: "user/message", seq: 11, data: { content: [{ type: "text", text: "task" }], source: { kind: "user" }, role: "user", id: "u1" } });
  const frame2 = assistantStep(1, [{ callId: "call_a", tool: "pwsh", arguments: JSON.stringify({ command: "tei version -json" }) }])
    + toolCallEvent(10, 1, 1, "call_a", "pwsh", JSON.stringify({ command: "tei version -json" }));
  const frame3 = toolResultEvent(11, 1, 1, "call_a", '{"ok":true}')
    + ev({ type: "assistant/message", seq: 200, data: { turn: 1, step: 2, message: { role: "assistant", content: [{ type: "text", text: "final answer" }], source: { kind: "model", provider: "openrouter", model: "~deepseek/deepseek-flash-latest", replayState: { response: { responseModel: "deepseek/synthetic-flash-1017" } } } } } })
    + ev({ type: "turn/end", seq: 300, data: { turn: 1, reason: { kind: "completed" } } });
  const buf = buildLog([frame1, frame2, frame3]);
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "dsh-evidence-test-"));
  const file = path.join(dir, "session.v4.jsonl.zstd");
  fs.writeFileSync(file, buf);
  try {
    const decoded = await decodeAllFrames(fs.readFileSync(file));
    const out = collectSession(decoded.text);
    assert.equal(decoded.frames, 3);
    assert.equal(out.sessionId, "session-synthetic-0001");
    assert.equal(out.cwd, "D:\\exp\\slot");
    assert.equal(out.counts.toolCalls, 1);
    assert.equal(out.counts.toolResults, 1);
    assert.equal(out.final.turnEndReason.kind, "completed");
    assert.equal(out.final.finalAnswerSha256, sha256hex(Buffer.from("final answer", "utf8")));
    assert.equal(out.steps[0].responseModel, "deepseek/synthetic-flash-1017");
    assert.equal(out.truncation.length, 0);
    assert.equal(out.workspaceTargets.length, 0);
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
});
