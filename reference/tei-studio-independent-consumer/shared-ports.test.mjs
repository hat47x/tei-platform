import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { stripTypeScriptTypes } from 'node:module';
import { journalGraph } from './journal-graph.mjs';
import { journalCommandOptions } from './journal-command.mjs';

const here = fileURLToPath(new URL('.', import.meta.url));
const graphSource = resolve(here, '../tei-studio-graph/graph.ts');
const commandSource = resolve(here, '../tei-studio-command/command.ts');
async function loadActualPort(path) {
  // No clone, imitation, or rewritten provider. Read the exact path extracted
  // from its own Draft PR in the fixed, no-merge qualification snapshot.
  const source = stripTypeScriptTypes(readFileSync(path, 'utf8'), { mode: 'strip' });
  return import('data:text/javascript;charset=utf-8,' + encodeURIComponent(source));
}

// Only the Graph DOM adapter needs this tiny test DOM. Browser E2E is separate.
class FakeElement {
  constructor(tag, doc) {
    this.tagName = tag;
    this.ownerDocument = doc;
    this.parentNode = null;
    this.children = [];
    this.attributes = new Map();
    this.listeners = new Map();
    this.style = {};
    this.textContent = '';
  }
  focus() {
    this.ownerDocument.activeElement = this;
    this.dispatch('focus');
  }
  setAttribute(key, value) { this.attributes.set(key, String(value)); }
  getAttribute(key) { return this.attributes.get(key) ?? null; }
  addEventListener(key, listener) {
    const existing = this.listeners.get(key) ?? [];
    existing.push(listener);
    this.listeners.set(key, existing);
  }
  appendChild(child) {
    child.remove();
    child.parentNode = this;
    this.children.push(child);
    return child;
  }
  replaceChildren(...children) {
    for (const child of this.children) child.parentNode = null;
    this.children = [];
    for (const child of children) this.appendChild(child);
  }
  remove() {
    if (!this.parentNode) return;
    this.parentNode.children = this.parentNode.children.filter(child => child !== this);
    this.parentNode = null;
  }
  dispatch(type, key) {
    let prevented = false;
    const event = { key, preventDefault() { prevented = true; } };
    for (let current = this; current; current = current.parentNode) {
      for (const fn of current.listeners.get(type) ?? []) fn(event);
    }
    return prevented;
  }
}
function fakeRoot() {
  const doc = {
    activeElement: null,
    createElement(tag) { return new FakeElement(tag, doc); },
    createElementNS(_ns, tag) { return new FakeElement(tag, doc); },
  };
  return doc.createElement('div');
}
function descendants(root, tag) {
  return [root, ...root.children.flatMap(x => descendants(x, tag))]
    .filter(x => x.tagName === tag);
}

test('actual Graph accepts a journal-only minimum input and rejects missing endpoints', {
  skip: !existsSync(graphSource) && 'extract exact #1828 graph.ts into the fixed snapshot',
}, async () => {
  const { normalizeGraph, GraphViewError } = await loadActualPort(graphSource);
  const input = journalGraph();
  const normalized = normalizeGraph(input);
  assert.deepEqual(normalized.nodes.map(x => x.id), ['map', 'tea', 'sketch']);
  assert.equal(normalized.edges.length, 3);
  assert.throws(() => normalizeGraph({
    ...input, edges: [{ id: 'invalid', source: 'tea', target: 'not-a-journal-note' }],
  }), e => e instanceof GraphViewError && e.code === 'missing_endpoint');
});

test('actual Graph emits note IDs via pointer/keyboard and remains fresh after remount', {
  skip: !existsSync(graphSource) && 'extract exact #1828 graph.ts into the fixed snapshot',
}, async () => {
  const { mountReadonlyGraph } = await loadActualPort(graphSource);
  const root = fakeRoot(), received = [];
  const first = mountReadonlyGraph(root, journalGraph(), { onSelect: id => received.push(id) });
  const firstGroups = descendants(root, 'g');
  assert.deepEqual(firstGroups.map(x => x.getAttribute('aria-label')), [
    'Neighborhood walk', 'Tea tasting', 'Window sketch',
  ]);
  const center = descendants(firstGroups[1], 'circle')[0];
  assert.equal(center.getAttribute('fill'), 'transparent');
  center.dispatch('click');
  assert.deepEqual(received, ['tea']);
  assert.equal(first.getSelectedNode(), 'tea');
  assert.equal(firstGroups[2].dispatch('keydown', 'Enter'), true);
  assert.deepEqual(received, ['tea', 'sketch']);
  first.dispose();
  first.dispose();
  assert.equal(root.children.length, 0);
  center.dispatch('click');
  assert.deepEqual(received, ['tea', 'sketch'], 'disposed graph must not emit');
  const second = mountReadonlyGraph(root, journalGraph(), { onSelect: id => received.push(id) });
  second.setSelectedNode('sketch');
  assert.equal(second.getSelectedNode(), 'sketch');
  descendants(root, 'g')[0].dispatch('click');
  assert.deepEqual(received, ['tea', 'sketch', 'map']);
  second.dispose();
  assert.equal(root.children.length, 0);
});

test('journal selection remains keyboard-operable when the actual Graph is synchronously re-rendered', {
  skip: !existsSync(graphSource) && 'extract exact #1828 graph.ts into the fixed snapshot',
}, async () => {
  const { mountReadonlyGraph } = await loadActualPort(graphSource);
  const root = fakeRoot(), events = [];
  let graph;
  // Exercise the reusable Graph callback contract with *journal IDs*, not TEI
  // subjects. The callback intentionally re-renders as the Studio consumer does.
  graph = mountReadonlyGraph(root, journalGraph(), {
    onSelect(id) {
      events.push(id);
      graph.update(journalGraph());
      graph.setSelectedNode(id);
    },
  });
  const first = descendants(root, 'g').find(x => x.getAttribute('aria-label') === 'Tea tasting');
  first.focus();
  assert.equal(root.ownerDocument.activeElement, first);
  assert.equal(first.dispatch('keydown', 'Enter'), true);
  const successor = descendants(root, 'g').find(x => x.getAttribute('aria-label') === 'Tea tasting');
  assert.notEqual(successor, first, 'update must replace the SVG node');
  assert.equal(root.ownerDocument.activeElement, successor,
    'the next keyboard action must remain on the selected journal note');
  assert.equal(graph.getSelectedNode(), 'tea');
  assert.equal(successor.dispatch('keydown', ' '), true);
  assert.deepEqual(events, ['tea', 'tea']);
  const afterSpace = descendants(root, 'g').find(x => x.getAttribute('aria-label') === 'Tea tasting');
  assert.equal(root.ownerDocument.activeElement, afterSpace);
  first.dispatch('click');
  assert.deepEqual(events, ['tea', 'tea'], 'a stale SVG must not publish a new note selection');
  const outside = root.ownerDocument.createElement('button');
  outside.focus();
  graph.update(journalGraph());
  assert.equal(root.ownerDocument.activeElement, outside,
    'unrelated controls must not lose focus to a background graph refresh');
  graph.dispose();
});

test('actual Command port keeps journal authorization outside shared panel after remount', {
  skip: !existsSync(commandSource) && 'extract exact #1829 command.ts into the fixed snapshot',
}, async () => {
  const { createCommandPort } = await loadActualPort(commandSource);
  let note = 'tea';
  const options = journalCommandOptions(() => note);
  const first = createCommandPort(options);
  assert.equal(await first.execute('count-links'), true);
  assert.deepEqual(first.snapshot().status, {
    phase: 'success', commandId: 'count-links', message: '2 related entries',
  });
  assert.equal(await first.execute('delete-entry'), true);
  assert.equal(first.snapshot().status.phase, 'denied');
  first.dispose();
  assert.equal(await first.execute('count-links'), false);
  note = 'sketch';
  const second = createCommandPort(options);
  assert.equal(await second.execute('count-links'), true);
  assert.deepEqual(second.snapshot().status, {
    phase: 'success', commandId: 'count-links', message: '0 related entries',
  });
  assert.equal(await second.execute('unknown-id'), false);
  second.dispose();
});
