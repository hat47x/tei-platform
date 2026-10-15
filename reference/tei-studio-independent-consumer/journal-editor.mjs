import { selectNote } from './notes.mjs';

// This consumer owns observed document identity/version, not Source authority.
// A new selected note is an independent read-only document projection.
export function journalEditorDocument(noteId) {
  const note = selectNote(noteId);
  if (!note) throw new Error('Unknown reading journal entry');
  return {
    id: `journal:${note.id}`,
    version: 'fixture-v1',
    text: `${note.title}\n\n${note.body}`,
    diagnostics: [],
  };
}
