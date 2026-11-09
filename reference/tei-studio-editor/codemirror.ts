/** CodeMirror 6-only rendering adapter. No TEI, filesystem or Host authority. */
import { EditorState } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { lintGutter, setDiagnostics } from "@codemirror/lint";
import { standardKeymap } from "@codemirror/commands";
import { CodeMirrorCoordinates } from "./codemirror-coordinates.ts";
import type { RendererFactory, EditorRenderer, EditorSnapshot, OffsetRange } from "./editor.ts";

/** Injectable renderer used in a browser or Desktop WebView with the same UI assets. */
export const codeMirrorRenderer: RendererFactory<HTMLElement> = {
  mount(parent, initial, onSelection): EditorRenderer {
    if (!parent) throw new Error("EDITOR_PARENT_REQUIRED");
    // All public snapshots/selection notifications use the ORIGINAL Source's
    // UTF-16 offsets. Only the renderer sees CodeMirror's LF-normalized text.
    let coordinates = new CodeMirrorCoordinates(initial.text);
    let disposed = false;
    let programmaticDispatch = false;
    const view = new EditorView({
      parent,
      state: EditorState.create({
        doc: coordinates.text,
        extensions: [
          EditorState.readOnly.of(true),
          EditorView.editable.of(false),
          EditorView.lineWrapping,
          keymap.of(standardKeymap),
          // A non-editable .cm-content is otherwise not keyboard-focusable.
          EditorView.contentAttributes.of({ "aria-label": "Read-only source text", tabindex: "0" }),
          lintGutter(),
          EditorView.updateListener.of(update => {
            if (disposed || programmaticDispatch || !update.selectionSet || update.docChanged) return;
            const main = update.state.selection.main;
            onSelection(coordinates.rangeToSource({ from: main.from, to: main.to }));
          })
        ]
      })
    });
    function check(): void { if (disposed) throw new Error("EDITOR_RENDERER_DISPOSED"); }
    function render(snapshot: EditorSnapshot): void {
      check();
      // Validate all positions against the next Source BEFORE dispatching any
      // UI mutation. Rebuild even when a CRLF->LF change leaves view text equal.
      const next = new CodeMirrorCoordinates(snapshot.text);
      const selected = next.rangeToView(snapshot.selection);
      const diagnostics = snapshot.diagnostics.map(({ from, to, message, severity }) => ({
        ...next.rangeToView({ from, to }), message, severity
      }));
      const before = view.state.doc.toString();
      programmaticDispatch = true;
      try {
        if (before !== next.text) {
          view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: next.text },
            selection: { anchor: selected.from, head: selected.to } });
        } else {
          const main = view.state.selection.main;
          if (main.from !== selected.from || main.to !== selected.to) {
            view.dispatch({ selection: { anchor: selected.from, head: selected.to } });
          }
        }
        coordinates = next;
        view.dispatch(setDiagnostics(view.state, diagnostics));
      } finally {
        programmaticDispatch = false;
      }
    }
    render(initial);
    return {
      render,
      select(range: OffsetRange) {
        check();
        const selection = coordinates.rangeToView(range);
        programmaticDispatch = true;
        try {
          view.dispatch({ selection: { anchor: selection.from, head: selection.to },
            scrollIntoView: true });
        } finally {
          programmaticDispatch = false;
        }
      },
      focus() { check(); view.focus(); },
      dispose() { if (disposed) return; disposed = true; view.destroy(); }
    };
  }
};
