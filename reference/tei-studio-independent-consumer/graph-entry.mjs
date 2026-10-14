import { notes } from './notes.mjs';
import { journalGraph } from './journal-graph.mjs';
import { mountReadonlyGraph } from '../tei-studio-graph/graph.ts';

// Same exact #9 Graph module used by Studio; esbuild resolves and compiles
// the original TypeScript. No copied renderer or raw .ts browser request.
const slot = document.getElementById('graph-slot');
const picker = document.getElementById('note-picker');
slot.replaceChildren();
const graph = mountReadonlyGraph(slot, journalGraph(), {
  onSelect(noteId) {
    if (!notes.some(note => note.id === noteId)) return;
    picker.value = noteId;
    picker.dispatchEvent(new Event('change'));
  },
});
slot.dataset.lifecycle = 'mounted';
graph.setSelectedNode(picker.value);
function syncSelection() { graph.setSelectedNode(picker.value); }
picker.addEventListener('change', syncSelection);
let released = false;
function release() {
  if (released) return;
  released = true;
  picker.removeEventListener('change', syncSelection);
  graph.dispose();
  slot.dataset.lifecycle = 'disposed';
}
window.addEventListener('pagehide', release, { once: true });
