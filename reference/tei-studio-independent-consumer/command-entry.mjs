import { linkedNotes } from './notes.mjs';
import { mountCommandPanel } from '../tei-studio-command/command.ts';

// The journal's invoke adapter owns success/denial; the panel grants no rights.
const slot = document.getElementById('command-slot');
const picker = document.getElementById('note-picker');
const unmount = document.getElementById('dispose-command');
const remount = document.getElementById('remount-command');
let panel = null;
let pageDisposed = false;
const options = {
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
};
function mount() {
  if (pageDisposed || panel) return;
  slot.replaceChildren();
  panel = mountCommandPanel(slot, options);
  slot.dataset.lifecycle = 'mounted';
  unmount.disabled = false;
  remount.disabled = true;
}
function dispose() {
  if (!panel) return;
  const old = panel;
  panel = null;
  old.dispose();
  slot.dataset.lifecycle = 'disposed';
  unmount.disabled = true;
  remount.disabled = pageDisposed;
}
unmount.addEventListener('click', dispose);
remount.addEventListener('click', mount);
window.addEventListener('pagehide', () => {
  pageDisposed = true;
  dispose();
}, { once: true });
mount();
