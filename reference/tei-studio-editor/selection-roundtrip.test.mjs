import test from "node:test";
import assert from "node:assert/strict";
import { createReadOnlyTextEditor } from "./editor.ts";
import { CodeMirrorCoordinates } from "./codemirror-coordinates.ts";

const span = (line, start, end, unit = "utf16-unit") => ({
  start: { line, column: start }, end: { line, column: end }, unit
});

// Models CodeMirror's LF-normalized offset boundary, NOT CodeMirror's real DOM.
// This proves the port's public callback contract jointly with the adapter
// coordinate table; a browser keyboard/selection event remains a separate E2E.
function normalizedViewFactory() {
  const state = { map: null, text: "", visibleDiagnostics: [],
    visualSelection: { from: 0, to: 0 }, emitUserSelection: null, destroyed: false };
  const factory = {
    mount(_host, snapshot, onSelection) {
      function render(next) {
        state.map = new CodeMirrorCoordinates(next.text);
        state.text = state.map.text;
        state.visualSelection = state.map.rangeToView(next.selection);
        state.visibleDiagnostics = next.diagnostics.map(d => ({
          id: d.id, ...state.map.rangeToView(d)
        }));
      }
      render(snapshot);
      state.emitUserSelection = range => {
        const sourceSelection = state.map.rangeToSource(range);
        onSelection(sourceSelection);
      };
      return {
        render,
        select(range) {
          state.visualSelection = state.map.rangeToView(range);
          // Some renderers emit a selection event during programmatic select:
          // the public editor must suppress this synchronous notification.
          onSelection(state.map.rangeToSource(state.visualSelection));
        },
        focus() {},
        dispose() { state.destroyed = true; }
      };
    }
  };
  return { state, factory };
}

test("Host range -> normalized selection -> user selection callback; CRLF and astral UTF-16", () => {
  const text = "a😀x\r\nsecond🧪!";
  const { state, factory } = normalizedViewFactory();
  const received = [];
  const editor = createReadOnlyTextEditor({}, factory, {
    id: "host:sample", version: "v1", text, diagnostics: [{
      id: "d1", message: "second", severity: "warning",
      range: span(2, 1, 7)
    }]
  }, event => received.push(event));

  assert.equal(editor.snapshot().text, text); // Original Source is not normalized.
  assert.equal(state.text, "a😀x\nsecond🧪!");
  assert.deepEqual(editor.snapshot().diagnostics.map(({ from, to }) => ({ from, to })),
    [{ from: 6, to: 12 }]);
  assert.deepEqual(state.visibleDiagnostics, [{ id: "d1", from: 5, to: 11 }]);

  // Host-originated Unicode columns -> original Source offset -> CodeMirror.
  editor.select(span(2, 1, 10));
  assert.deepEqual(editor.snapshot().selection, { from: 6, to: 15 });
  assert.deepEqual(state.visualSelection, { from: 5, to: 14 });
  assert.equal(received.length, 0, "programmatic select must not emit");

  // Browser-originated normalized offsets -> original Source UTF-16 callback.
  state.emitUserSelection({ from: 5, to: 14 });
  assert.deepEqual(received, [{
    documentId: "host:sample", version: "v1",
    selection: { from: 6, to: 15 }
  }]);
  assert.deepEqual(editor.snapshot().selection, { from: 6, to: 15 });
  assert.equal(text.slice(received[0].selection.from, received[0].selection.to), "second🧪!");
  editor.destroy();
});

test("UTF-8 byte columns around astral scalar survive Host-view-Host roundtrip", () => {
  const { state, factory } = normalizedViewFactory();
  const editor = createReadOnlyTextEditor({}, factory,
    { id: "unicode", version: "r1", text: "a😀x\r\nsecond🧪!" }, event => {
      state.received = event;
    });
  // After six ASCII bytes, 🧪 occupies 4 UTF-8 bytes (2 UTF-16 units).
  editor.select(span(2, 7, 11, "utf8-byte"));
  assert.deepEqual(editor.snapshot().selection, { from: 12, to: 14 });
  assert.deepEqual(state.visualSelection, { from: 11, to: 13 });
  state.emitUserSelection({ from: 11, to: 13 });
  assert.deepEqual(state.received.selection, { from: 12, to: 14 });
  assert.throws(() => editor.select(span(2, 8, 11, "utf8-byte")),
    /SPLITS_CODEPOINT/);
  assert.throws(() => editor.select(span(2, 8, 9)), /SPLITS_CODEPOINT/);
  editor.destroy();
});

test("re-render with identical view text but changed CRLF offsets uses new Source map", () => {
  const { state, factory } = normalizedViewFactory();
  const notifications = [];
  const editor = createReadOnlyTextEditor({}, factory,
    { id: "doc", version: "v1", text: "a😀x\r\nsecond" },
    event => notifications.push(event));
  state.emitUserSelection({ from: 5, to: 11 });
  assert.deepEqual(notifications.at(-1).selection, { from: 6, to: 12 });

  editor.setDocument({ id: "doc", version: "v2", text: "a😀x\nsecond" });
  assert.equal(state.text, "a😀x\nsecond");
  state.emitUserSelection({ from: 5, to: 11 });
  assert.deepEqual(notifications.at(-1), {
    documentId: "doc", version: "v2", selection: { from: 5, to: 11 }
  });
  assert.throws(() => editor.setDiagnostics("doc", "v1", []),
    /STALE_EDITOR_DIAGNOSTICS/);
  editor.destroy();
  state.emitUserSelection({ from: 5, to: 11 });
  assert.equal(notifications.length, 2, "no callbacks after destruction");
  assert.equal(state.destroyed, true);
});
