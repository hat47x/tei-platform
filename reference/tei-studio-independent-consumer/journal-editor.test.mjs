import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { stripTypeScriptTypes } from 'node:module';
import { journalEditorDocument } from './journal-editor.mjs';

test('journal selections yield independent readonly document projections', () => {
  const first = journalEditorDocument('map');
  const second = journalEditorDocument('sketch');
  assert.equal(first.id, 'journal:map');
  assert.equal(first.version, 'fixture-v1');
  assert.match(second.text, /Window sketch/);
  assert.notEqual(first.id, second.id);
  assert.deepEqual(second.diagnostics, []);
  assert.throws(() => journalEditorDocument('unknown'), /Unknown reading journal/);
});

const provider = resolve(fileURLToPath(new URL('.', import.meta.url)), '../tei-studio-editor/editor.ts');
test('real #4 editor port accepts setDocument and invokes destroy', {
  skip: !existsSync(provider) && 'fixed snapshot has not supplied #1830 editor.ts',
}, async () => {
  // Load the exact TypeScript port without bundling its CodeMirror DOM adapter.
  // The real browser bundle exercises the adapter; this is a lifecycle unit test.
  const code = stripTypeScriptTypes(readFileSync(provider, 'utf8'), { mode: 'strip' });
  const { createReadOnlyTextEditor } = await import(
    'data:text/javascript;charset=utf-8,' + encodeURIComponent(code)
  );
  const renders = [];
  let disposed = 0;
  const factory = {
    mount(host, initial) {
      renders.push(initial);
      return {
        render(next) { renders.push(next); },
        select() {},
        focus() {},
        dispose() { disposed++; },
      };
    },
  };
  const port = createReadOnlyTextEditor({}, factory, journalEditorDocument('map'));
  port.setDocument(journalEditorDocument('sketch'));
  assert.equal(port.snapshot().documentId, 'journal:sketch');
  assert.match(renders.at(-1).text, /Window sketch/);
  assert.throws(() => port.setDocument({ ...journalEditorDocument('sketch'),
    text: 'different contents' }), /EDITOR_VERSION_REUSED_FOR_DIFFERENT_TEXT/);
  port.destroy();
  port.destroy();
  assert.equal(disposed, 1);
  assert.throws(() => port.snapshot(), /EDITOR_DISPOSED/);
});
