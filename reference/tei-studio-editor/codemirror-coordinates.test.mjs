import test from "node:test";
import assert from "node:assert/strict";
import { CodeMirrorCoordinates } from "./codemirror-coordinates.ts";
import { pointToOffset, rangeToOffsets } from "./editor.ts";

const fixture = "a😀x\r\nsecond";

test("real browser regression: CRLF Source diagnostics and selection cover 'second'", () => {
  const map = new CodeMirrorCoordinates(fixture);
  assert.equal(map.text, "a😀x\nsecond");
  assert.equal(map.viewLength, 11);
  const sourceRange = rangeToOffsets(fixture, {
    start: { line: 2, column: 1 }, end: { line: 2, column: 7 }, unit: "utf16-unit"
  });
  assert.deepEqual(sourceRange, { from: 6, to: 12 });
  assert.deepEqual(map.rangeToView(sourceRange), { from: 5, to: 11 });
  assert.equal(map.text.slice(5, 11), "second");
  assert.deepEqual(map.rangeToSource({ from: 5, to: 11 }), sourceRange);
  assert.equal(map.toView(pointToOffset(fixture, { line: 2, column: 1 }, "utf8-byte")), 5);
  assert.equal(map.toView(pointToOffset(fixture, { line: 2, column: 7 }, "utf8-byte")), 11);
});

test("multi CRLF + LF preserve original Source callback offsets, empty trailing line", () => {
  const source = "one\r\ntwo\n三😀\r\n";
  const map = new CodeMirrorCoordinates(source);
  assert.equal(map.text, "one\ntwo\n三😀\n");
  const points = [0, 1, 3, 5, 7, 8, 9, 10, 11, 13, 14, source.length];
  for (const from of points) {
    if (source[from - 1] === "\r") continue;
    const to = map.toView(from);
    assert.equal(map.toSource(to), from);
  }
  assert.equal(map.toView(source.length), map.viewLength);
  assert.equal(map.toSource(map.viewLength), source.length);
});

test("LF-only and empty input map identity, including end offset", () => {
  for (const source of ["", "a\nb", "😀x\n", "a\n\nb"]) {
    const map = new CodeMirrorCoordinates(source);
    assert.equal(map.text, source);
    for (let i = 0; i <= source.length; i++) {
      assert.equal(map.toSource(map.toView(i)), i);
    }
  }
});

test("reject CRLF interior and invalid/noninteger/out-of-bound coordinates", () => {
  const map = new CodeMirrorCoordinates(fixture);
  assert.throws(() => map.toView(5), /SOURCE_OFFSET_INSIDE_CRLF/);
  assert.throws(() => map.toView(13), /INVALID_SOURCE_OFFSET/);
  assert.throws(() => map.toView(2.5), /INVALID_SOURCE_OFFSET/);
  assert.throws(() => map.toView(-1), /INVALID_SOURCE_OFFSET/);
  assert.throws(() => map.toSource(12), /INVALID_EDITOR_VIEW_OFFSET/);
  assert.throws(() => map.rangeToView({ from: 6, to: 1 }), /REVERSED/);
  assert.throws(() => new CodeMirrorCoordinates("a\rb"), /UNSUPPORTED_EDITOR_CR_ONLY_LINE_BREAK/);
});

test("re-map every render when Source changes from CRLF to LF without changing view text", () => {
  const old = new CodeMirrorCoordinates("x\r\ny");
  const next = new CodeMirrorCoordinates("x\ny");
  assert.equal(old.text, next.text);
  assert.equal(old.toSource(2), 3);
  assert.equal(next.toSource(2), 2);
  assert.equal(old.toView(3), next.toView(2));
  assert.equal(old.toView(4), next.toView(3));
});
