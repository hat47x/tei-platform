// Independent consumer fixture: a private reading journal, not a TEI Workspace.
export const notes = Object.freeze([
  Object.freeze({ id: 'map', title: 'Neighborhood walk', body: 'A short walk around the park and river.', related: Object.freeze(['tea']) }),
  Object.freeze({ id: 'tea', title: 'Tea tasting', body: 'Compare the aroma and brewing time of two teas.', related: Object.freeze(['map', 'sketch']) }),
  Object.freeze({ id: 'sketch', title: 'Window sketch', body: 'Study light, shadow, and simple shapes.', related: Object.freeze([]) }),
]);

export function selectNote(noteId, collection = notes) {
  if (typeof noteId !== 'string') return null;
  return collection.find(note => note.id === noteId) ?? null;
}

export function linkedNotes(noteId, collection = notes) {
  const note = selectNote(noteId, collection);
  if (!note) return [];
  return note.related.map(id => selectNote(id, collection)).filter(Boolean);
}
