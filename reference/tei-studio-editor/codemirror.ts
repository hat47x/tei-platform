/** CodeMirror 6-only rendering adapter. No TEI, filesystem or Host authority. */
import { EditorState } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { lintGutter, setDiagnostics } from "@codemirror/lint";
import type { RendererFactory, EditorRenderer, EditorSnapshot, OffsetRange } from "./editor.ts";

/** Injectable renderer used in a browser or Desktop WebView with the same UI assets. */
export const codeMirrorRenderer: RendererFactory<HTMLElement> = {
  mount(parent, initial, onSelection): EditorRenderer {
    if (!parent) throw new Error("EDITOR_PARENT_REQUIRED");
    const view = new EditorView({
      parent,
      state: EditorState.create({
        doc: initial.text,
        extensions: [
          EditorState.readOnly.of(true),
          EditorView.editable.of(false),
          EditorView.lineWrapping,
          EditorView.contentAttributes.of({ "aria-label": "Read-only source text" }),
          lintGutter(),
          EditorView.updateListener.of(update => {
            if (!update.selectionSet || update.docChanged) return;
            const main = update.state.selection.main;
            onSelection({ from: main.from, to: main.to });
          })
        ]
      })
    });
    let disposed = false;
    function check(): void { if (disposed) throw new Error("EDITOR_RENDERER_DISPOSED"); }
    function render(snapshot: EditorSnapshot): void {
      check();
      const before = view.state.doc.toString();
      if (before !== snapshot.text) {
        view.dispatch({ changes: { from: 0, to: before.length, insert: snapshot.text },
          selection: { anchor: snapshot.selection.from, head: snapshot.selection.to } });
      } else {
        const main = view.state.selection.main;
        if (main.from !== snapshot.selection.from || main.to !== snapshot.selection.to) {
          view.dispatch({ selection: {
            anchor: snapshot.selection.from, head: snapshot.selection.to
          } });
        }
      }
      view.dispatch(setDiagnostics(view.state,
        snapshot.diagnostics.map(({ from, to, message, severity }) => ({
          from, to, message, severity
        }))));
    }
    render(initial);
    return {
      render,
      select(range: OffsetRange) {
        check();
        view.dispatch({ selection: { anchor: range.from, head: range.to }, scrollIntoView: true });
      },
      focus() { check(); view.focus(); },
      dispose() { if (disposed) return; disposed = true; view.destroy(); }
    };
  }
};
