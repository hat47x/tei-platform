import { linkedNotes } from './notes.mjs';
import { mountCommandPanel } from '../tei-studio-command/command.ts';

// The journal owns outcomes; the shared command panel never grants authority.
// The same #5 TypeScript module is bundled for both consumer and Studio.
const slot = document.getElementById('command-slot');
const picker = document.getElementById('note-picker');
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
slot.dataset.lifecycle = 'mounted';
let released = false;
function release() {
  if (released) return;
  released = true;
  panel.dispose();
  slot.dataset.lifecycle = 'disposed';
}
window.addEventListener('pagehide', release, { once: true });
