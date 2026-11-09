import { notes, selectNote, linkedNotes } from './notes.mjs';

// Only this independent consumer's browse/select UI is active. Do not replace
// shared Editor/Command/Graph with look-alike implementations for a fake PASS.
const picker = document.getElementById('note-picker');
const title = document.getElementById('note-title');
const body = document.getElementById('note-body');
const links = document.getElementById('note-links');

for (const note of notes) {
  const option = document.createElement('option');
  option.value = note.id;
  option.textContent = note.title;
  picker.append(option);
}

function render(noteId) {
  const note = selectNote(noteId);
  if (!note) throw new Error('Unknown reading journal entry');
  title.textContent = note.title;
  body.textContent = note.body;
  links.textContent = linkedNotes(noteId).map(other => other.title).join(', ') || '(none)';
}

picker.addEventListener('change', () => render(picker.value));
render(notes[0].id);
