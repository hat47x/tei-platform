import { notes, selectNote } from './notes.mjs';
import { createReadOnlyTextEditor } from '../tei-studio-editor/editor.ts';
import { codeMirrorRenderer } from '../tei-studio-editor/codemirror.ts';

// This application owns its read-only journal data. The shared editor never
// receives a TEI Workspace, Accepted Source or a file-system capability.
const picker = document.getElementById('note-picker');
const slot = document.getElementById('editor-slot');
const disposeButton = document.getElementById('dispose-editor');

function noteDocument(noteId) {
  const note = selectNote(noteId);
  if (!note) throw new Error('Unknown reading journal entry');
  return {
    id: `journal:${note.id}`,
    version: 'fixture-v1',
    text: `${note.title}\n\n${note.body}`,
    diagnostics: [],
  };
}

slot.replaceChildren();
const editor = createReadOnlyTextEditor(
  slot, codeMirrorRenderer, noteDocument(picker.value),
);
slot.dataset.lifecycle = 'mounted';
slot.dataset.documentId = editor.snapshot().documentId;

function changeNote() {
  if (released) return;
  editor.setDocument(noteDocument(picker.value));
  slot.dataset.documentId = editor.snapshot().documentId;
}
let released = false;
picker.addEventListener('change', changeNote);

function release() {
  if (released) return;
  released = true;
  picker.removeEventListener('change', changeNote);
  editor.destroy();
  slot.dataset.lifecycle = 'disposed';
  disposeButton.disabled = true;
}
disposeButton.addEventListener('click', release);
window.addEventListener('pagehide', release, { once: true });
