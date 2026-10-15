import { notes } from './notes.mjs';
import { journalGraph } from './journal-graph.mjs';
import { mountReadonlyGraph } from '../tei-studio-graph/graph.ts';

// GraphSelection is a consumer-owned note ID, never a Source/authority ID.
const slot = document.getElementById('graph-slot');
const picker = document.getElementById('note-picker');
const unmount = document.getElementById('dispose-graph');
const remount = document.getElementById('remount-graph');
let graph = null;
let pageDisposed = false;

function syncSelection() {
  graph?.setSelectedNode(picker.value);
}
function onSelect(noteId) {
  if (!notes.some(note => note.id === noteId)) return;
  picker.value = noteId;
  picker.dispatchEvent(new Event('change'));
}
function mount() {
  if (pageDisposed || graph) return;
  slot.replaceChildren();
  graph = mountReadonlyGraph(slot, journalGraph(), { onSelect });
  graph.setSelectedNode(picker.value);
  slot.dataset.lifecycle = 'mounted';
  picker.addEventListener('change', syncSelection);
  unmount.disabled = false;
  remount.disabled = true;
}
function dispose() {
  if (!graph) return;
  picker.removeEventListener('change', syncSelection);
  const old = graph;
  graph = null;
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
