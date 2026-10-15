import test from 'node:test';
import assert from 'node:assert/strict';
import { notes, selectNote, linkedNotes } from './notes.mjs';

test('reading journal is a consumer fixture with no workspace model', () => {
  assert.deepEqual(notes.map(item => item.id), ['map', 'tea', 'sketch']);
  assert.equal(selectNote('tea').title, 'Tea tasting');
  assert.equal(selectNote('../unknown'), null);
  assert.deepEqual(linkedNotes('tea').map(item => item.id), ['map', 'sketch']);
  assert.deepEqual(linkedNotes('missing'), []);
});

test('consumer fixture does not contain aliasing mutable relationship lists', () => {
  assert.equal(Object.isFrozen(notes), true);
  for (const note of notes) assert.equal(Object.isFrozen(note.related), true);
});
