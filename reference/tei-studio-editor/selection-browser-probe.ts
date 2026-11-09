/** TEST-ONLY real CodeMirror browser probe. Never import into a product Host. */
import { createReadOnlyTextEditor } from "./editor.ts";
import { codeMirrorRenderer } from "./codemirror.ts";
import type { SelectionEvent } from "./editor.ts";

const text = "a😀x\r\nsecond🧪!";
const host = document.getElementById("editor-selection-probe");
if (!host) throw new Error("EDITOR_SELECTION_PROBE_HOST_MISSING");
const events: SelectionEvent[] = [];
const keys: string[] = [];
const inputs: string[] = [];
const editor = createReadOnlyTextEditor(host, codeMirrorRenderer, {
  id: "browser-probe", version: "v1", text,
  diagnostics: [{
    id: "diag-second", message: "second", severity: "warning",
    range: {
      start: { line: 2, column: 1 }, end: { line: 2, column: 7 },
      unit: "utf16-unit"
    }
  }]
}, event => events.push(event));
const content = host.querySelector(".cm-content");
if (!content) throw new Error("EDITOR_SELECTION_PROBE_CONTENT_MISSING");
content.addEventListener("keydown", event => keys.push((event as KeyboardEvent).key));
content.addEventListener("beforeinput", event => inputs.push((event as InputEvent).inputType));
content.addEventListener("input", event => inputs.push("input:" + (event as InputEvent).inputType));

/** Exposed only on this test page for Playwright/evaluation. */
Object.assign(window, { __editorSelectionProbe: {
  editor, text, events, keys, inputs,
  content, host,
  placeCaret() {
    editor.select({
      start: { line: 2, column: 1 }, end: { line: 2, column: 1 },
      unit: "utf16-unit"
    });
    editor.focus();
  },
  clearEvents() { events.length = 0; keys.length = 0; inputs.length = 0; },
  destroy() { editor.destroy(); }
} });
