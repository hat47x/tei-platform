import { linkedNotes } from './notes.mjs';

// All domain decisions stay with the Reading Journal consumer, NOT the
// presentation component. In particular, 'enabled' is not an authority grant.
export function journalCommandOptions(currentNoteId) {
  if (typeof currentNoteId !== 'function') throw new TypeError('note selector is required');
  return {
    commands: [
      { id: 'count-links', label: 'Count related entries', enabled: true },
      { id: 'delete-entry', label: 'Delete entry (denied)', enabled: true },
    ],
    diagnostics: [{ id: 'readonly', severity: 'info', message: 'This journal is read-only.' }],
    async invoke(id) {
      if (id === 'count-links') {
        return { kind: 'success', message: `${linkedNotes(currentNoteId()).length} related entries` };
      }
      return { kind: 'denied', message: 'The reading journal does not permit deletion.' };
    },
  };
}
