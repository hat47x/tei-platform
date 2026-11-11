"use strict";
/*
 * Independent regression tests for the DSH session runner helpers
 * (TEI #2509 / TAI #168). These cover only the pure helpers; no Session is
 * started and no Provider is contacted.
 */
import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { resolveDshBin, shellCommandFor } from "./run-dsh-session.mjs";

test("an explicit launcher path is used as given", () => {
  assert.equal(resolveDshBin("C:\\tools\\dsh.cmd"), "C:\\tools\\dsh.cmd");
});

test("the installed DSH runtime path is discovered when PATH lacks it", () => {
  const savedPath = process.env.PATH;
  process.env.PATH = "C:\\windows\\system32";
  try {
    const resolved = resolveDshBin(undefined);
    assert.ok(fs.existsSync(resolved), `resolved launcher must exist: ${resolved}`);
    assert.ok(/dsh\.cmd$/i.test(resolved));
  } finally {
    process.env.PATH = savedPath;
  }
});

test("a launcher path with spaces is quoted for cmd.exe", () => {
  assert.equal(shellCommandFor("C:\\Program Files\\DeepSeek Harness\\dsh.cmd"),
    '"C:\\Program Files\\DeepSeek Harness\\dsh.cmd"');
  assert.equal(shellCommandFor("dsh"), "dsh");
  assert.equal(shellCommandFor("C:\\tools\\dsh.cmd"), "C:\\tools\\dsh.cmd");
});

test("PATH discovery wins over the installed fallback", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "dsh-bin-test-"));
  fs.writeFileSync(path.join(dir, "dsh.cmd"), "@echo off\r\n");
  const savedPath = process.env.PATH;
  process.env.PATH = `${dir};${savedPath}`;
  try {
    assert.equal(resolveDshBin(undefined), path.join(dir, "dsh.cmd"));
  } finally {
    process.env.PATH = savedPath;
    fs.rmSync(dir, { recursive: true, force: true });
  }
});
