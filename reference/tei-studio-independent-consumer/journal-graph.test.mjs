import test from 'node:test';
import assert from 'node:assert/strict';
import { journalGraph } from './journal-graph.mjs';

test('journal GraphInput maps consumer-owned notes and relationships', () => {
  const graph = journalGraph();
  assert.deepEqual(graph.nodes.map(node => node.id), ['map','tea','sketch']);
  assert.deepEqual(graph.edges.map(edge => [edge.source, edge.target]), [
    ['map','tea'], ['tea','map'], ['tea','sketch'],
  ]);
  assert.equal(new Set(graph.edges.map(edge => edge.id)).size, graph.edges.length);
  for (const node of graph.nodes) {
    assert.ok(node.x >= 0 && node.x <= graph.width);
    assert.ok(node.y >= 0 && node.y <= graph.height);
  }
});

test('sample refuses unsupported sizes rather than generating invalid positions', () => {
  assert.throws(() => journalGraph(Array.from({length:4},(_,i)=>({id:String(i),title:'x',related:[]}))), /unsupported_sample_size/);
});
