import test from "node:test";
import assert from "node:assert/strict";
import { createReadOnlyTextEditor, pointToOffset, rangeToOffsets } from "./editor.ts";

function fixture(text = "a😀x\r\nsecond") {
  return { id: "unrelated:document", version: "observed-v1", text };
}
function mockFactory() {
  const record = { renders: [], selections: [], focused: 0, disposed: 0, notify: undefined };
  return { record, factory: { mount(_host, snapshot, notify) {
    record.renders.push(snapshot); record.notify = notify;
    return {
      render(next) { record.renders.push(next); },
      select(range) { record.selections.push(range); notify(range); },
      focus() { record.focused++; }, dispose() { record.disposed++; }
    };
  } } };
}
const span = (line, start, end, unit = "utf8-byte") => ({
  start: { line, column: start }, end: { line, column: end }, unit
});

test("UTF-8 bytes, UTF-16 units and CRLF map exactly, never inside an emoji", () => {
  const text = fixture().text;
  assert.equal(pointToOffset(text, { line: 1, column: 2 }, "utf8-byte"), 1);
  assert.equal(pointToOffset(text, { line: 1, column: 6 }, "utf8-byte"), 3);
  assert.equal(pointToOffset(text, { line: 2, column: 1 }, "utf8-byte"), 6);
  assert.deepEqual(rangeToOffsets(text, span(1, 2, 6)), { from: 1, to: 3 });
  assert.deepEqual(rangeToOffsets(text, span(1, 2, 4, "utf16-unit")), { from: 1, to: 3 });
  assert.throws(() => pointToOffset(text, { line: 1, column: 3 }, "utf8-byte"), /SPLITS_CODEPOINT/);
  assert.throws(() => pointToOffset(text, { line: 1, column: 3 }, "utf16-unit"), /SPLITS_CODEPOINT/);
  assert.throws(() => pointToOffset(text, { line: 1, column: 6 }, "utf16-unit"), /OUT_OF_RANGE/);
  assert.throws(() => rangeToOffsets(text, { start: {line:2,column:1}, end:{line:1,column:1}, unit:"utf8-byte" }), /REVERSED/);
});

test("read-only editor projects diagnostics and selection into a non-TEI consumer", () => {
  const { factory, record } = mockFactory();
  const notifications = [];
  const editor = createReadOnlyTextEditor({}, factory, fixture(), e => notifications.push(e));
  editor.setDiagnostics("unrelated:document", "observed-v1", [
    { id: "d1", message: "見つからない参照", severity: "error", range: span(1, 2, 6) }
  ]);
  assert.deepEqual(editor.snapshot().diagnostics[0], {
    id: "d1", message: "見つからない参照", severity: "error", from: 1, to: 3
  });
  editor.select(span(2, 1, 4));
  assert.deepEqual(record.selections.pop(), { from: 6, to: 9 });
  assert.equal(notifications.length, 0); // internal select must not emit stale navigation
  record.notify({ from: 3, to: 4 });
  assert.deepEqual(notifications[0], {
    documentId: "unrelated:document", version: "observed-v1", selection: { from: 3, to: 4 }
  });
  record.notify({ from: 2, to: 2 }); // forged surrogate split: ignore
  assert.equal(notifications.length, 1);
  const external = editor.snapshot();
  external.diagnostics[0].message = "mutated";
  assert.equal(editor.snapshot().diagnostics[0].message, "見つからない参照");
  editor.focus(); assert.equal(record.focused, 1);
});

test("fail closed on stale diagnostics, invalid ranges, duplicate IDs, revision reuse", () => {
  const { factory, record } = mockFactory();
  const editor = createReadOnlyTextEditor({}, factory, fixture());
  const count = record.renders.length;
  assert.throws(() => editor.setDiagnostics("other", "observed-v1", []), /STALE/);
  assert.throws(() => editor.setDiagnostics("unrelated:document", "older", []), /STALE/);
  assert.throws(() => editor.setDiagnostics("unrelated:document", "observed-v1", [
    { id: "same", message: "1", severity: "error", range: span(1, 1, 2) },
    { id: "same", message: "2", severity: "hint", range: span(1, 1, 2) }
  ]), /INVALID_DIAGNOSTIC/);
  assert.throws(() => editor.setDocument({ ...fixture(), text: "different" }), /VERSION_REUSED/);
  assert.equal(record.renders.length, count);
  editor.setDocument({ id: "unrelated:document", version: "observed-v2", text: "new" });
  assert.equal(editor.snapshot().diagnostics.length, 0);
  assert.deepEqual(editor.snapshot().selection, { from: 0, to: 0 });
});

test("destroy once, remove selection callbacks and reject later API calls", () => {
  const { factory, record } = mockFactory();
  const changes = [];
  const editor = createReadOnlyTextEditor({}, factory, fixture(), e => changes.push(e));
  editor.destroy(); editor.destroy(); record.notify({ from: 0, to: 1 });
  assert.equal(record.disposed, 1); assert.equal(changes.length, 0);
  assert.throws(() => editor.focus(), /EDITOR_DISPOSED/);
  assert.throws(() => editor.snapshot(), /EDITOR_DISPOSED/);
  assert.throws(() => editor.select(span(1, 1, 2)), /EDITOR_DISPOSED/);
});
