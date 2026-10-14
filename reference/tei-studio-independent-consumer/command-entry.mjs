import { linkedNotes } from './notes.mjs';

// This local journal owns actions and their outcomes; the shared #5 panel is
// presentation/dispatch only and grants no authority to write or run shell.
const slot = document.getElementById('command-slot');
const picker = document.getElementById('note-picker');
try {
  const { mountCommandPanel } = await import('./.generated/command.mjs');
  if (typeof mountCommandPanel !== 'function') throw new Error('missing_export');
  slot.replaceChildren();
  const panel = mountCommandPanel(slot, {
    commands: [
      { id: 'count-links', label: 'Count related entries', enabled: true },
      { id: 'delete-entry', label: 'Delete entry (denied)', enabled: true },
    ],
    diagnostics: [{ id: 'readonly', severity: 'info', message: 'This journal is read-only.' }],
    async invoke(id) {
      if (id === 'count-links') {
        return { kind: 'success', message: `${linkedNotes(picker.value).length} related entries` };
      }
      return { kind: 'denied', message: 'The reading journal does not permit deletion.' };
    },
  });
  window.addEventListener('pagehide', () => panel.dispose(), { once: true });
} catch (error) {
  slot.textContent = 'BLOCKED: shared Command source/build not available; see README.';
  console.warn('Shared Command not connected:', error);
}
