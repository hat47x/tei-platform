import { journalCommandOptions } from './journal-command.mjs';
import { mountCommandPanel } from '../tei-studio-command/command.ts';

// The journal's invoke adapter owns success/denial; the panel grants no rights.
const slot = document.getElementById('command-slot');
const picker = document.getElementById('note-picker');
const unmount = document.getElementById('dispose-command');
const remount = document.getElementById('remount-command');
let panel = null;
let pageDisposed = false;
const options = journalCommandOptions(() => picker.value);
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
