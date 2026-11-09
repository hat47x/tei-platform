import assert from "node:assert/strict";
import { test } from "node:test";

import { parseStrictDiagnosticJSON } from "./strict-diagnostic-json.mjs";

test("strict diagnostic JSON retains legitimate nested structures", () => {
  const value = '{"applicationId":"order_entry","nested":[{"x":1},{"x":2}],"literal":"a\\\\b"}';
  assert.deepEqual(parseStrictDiagnosticJSON(value), JSON.parse(value));
});

test("duplicate top-level and nested keys are rejected before last-wins parsing", () => {
  for (const raw of [
    '{"apiSHA256":"old","apiSHA256":"new"}',
    '{"nested":{"digest":"old","digest":"new"}}',
    '[{"revision":"a","revision":"b"}]',
    '{"x":1,"nested":{"field":1,"field":2},"ok":true}',
  ]) {
    assert.throws(() => parseStrictDiagnosticJSON(raw), /duplicate diagnostic JSON key/);
  }
});

test("JSON string escape equivalence cannot bypass duplicate-key detection", () => {
  const first = '{"applicationId":1,"\\u0061pplicationId":2}';
  const second = '{"digest":1,"di\\u0067est":2}';
  assert.throws(() => parseStrictDiagnosticJSON(first), /duplicate diagnostic JSON key: applicationId/);
  assert.throws(() => parseStrictDiagnosticJSON(second), /duplicate diagnostic JSON key: digest/);
});

test("strict parser rejects non-JSON and oversized nesting without issuing acceptance", () => {
  for (const raw of ['{} {}', '{"x":}', '{"x":1,}', '{"a":"\\q"}', '{"a":1,', '']) {
    assert.throws(() => parseStrictDiagnosticJSON(raw));
  }
  const nested = "[".repeat(140) + "0" + "]".repeat(140);
  assert.throws(() => parseStrictDiagnosticJSON(nested), /nesting exceeds limit/);
});
