import { notes } from './notes.mjs';
import { journalGraph } from './journal-graph.mjs';

// This consumer uses the exact export documented by #9, not an imitation.
// .generated/graph.mjs is built locally from #9's graph.ts, never vendored.
const slot = document.getElementById('graph-slot');
const picker = document.getElementById('note-picker');
try {
  const { mountReadonlyGraph } = await import('./.generated/graph.mjs');
  if (typeof mountReadonlyGraph !== 'function') throw new Error('missing_export');
  slot.replaceChildren();
  const graph = mountReadonlyGraph(slot, journalGraph(), {
    onSelect(noteId) {
      if (!notes.some(note => note.id === noteId)) return;
      picker.value = noteId;
      picker.dispatchEvent(new Event('change'));
    },
  });
  graph.setSelectedNode(picker.value);
  picker.addEventListener('change', () => graph.setSelectedNode(picker.value));
  window.addEventListener('pagehide', () => graph.dispose(), { once: true });
} catch (error) {
  // An unavailable / unbuildable provider must remain BLOCKED, not a fake renderer.
  slot.textContent = 'BLOCKED: shared Graph source/build not available; see README.';
  console.warn('Shared Graph not connected:', error);
}
