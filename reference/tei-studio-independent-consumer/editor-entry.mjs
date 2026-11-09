import { journalEditorDocument } from './journal-editor.mjs';
import { createReadOnlyTextEditor } from '../tei-studio-editor/editor.ts';
import { codeMirrorRenderer } from '../tei-studio-editor/codemirror.ts';

// Consumer-owned note document; no TEI Workspace, Source or acceptance access.
const picker = document.getElementById('note-picker');
const slot = document.getElementById('editor-slot');
const unmount = document.getElementById('dispose-editor');
const remount = document.getElementById('remount-editor');
let editor = null;
let pageDisposed = false;

function changeNote() {
  if (!editor) return;
  editor.setDocument(journalEditorDocument(picker.value));
  slot.dataset.documentId = editor.snapshot().documentId;
}
function mount() {
  if (pageDisposed || editor) return;
  slot.replaceChildren();
  editor = createReadOnlyTextEditor(
    slot, codeMirrorRenderer, journalEditorDocument(picker.value),
  );
  slot.dataset.lifecycle = 'mounted';
  slot.dataset.documentId = editor.snapshot().documentId;
  picker.addEventListener('change', changeNote);
  unmount.disabled = false;
  remount.disabled = true;
}
function destroy() {
  if (!editor) return;
  picker.removeEventListener('change', changeNote);
  const previous = editor;
  editor = null;
  previous.destroy();
  slot.dataset.lifecycle = 'disposed';
  delete slot.dataset.documentId;
  unmount.disabled = true;
  remount.disabled = pageDisposed;
}
unmount.addEventListener('click', destroy);
remount.addEventListener('click', mount);
window.addEventListener('pagehide', () => {
  pageDisposed = true;
  destroy();
}, { once: true });
mount();
