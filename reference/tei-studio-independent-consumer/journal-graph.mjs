import { notes } from './notes.mjs';

// Consumer-owned positioning and relationships. Uses #9's documented GraphInput.
// Neither this data nor the graph selection grants permissions or performs I/O.
export function journalGraph(collection = notes) {
  const positions = [[75, 80], [240, 80], [405, 80]];
  if (collection.length > positions.length) throw new Error('unsupported_sample_size');
  return {
    width: 480,
    height: 170,
    nodes: collection.map((note, index) => ({
      id: note.id, label: note.title, x: positions[index][0], y: positions[index][1],
    })),
    edges: collection.flatMap(note => note.related.map(id => ({
      id: `${note.id}--${id}`, source: note.id, target: id,
    }))),
  };
}
