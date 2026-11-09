import test from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { auditWorkbenchReuse } from './audit-workbench-reuse.mjs';

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, '../../');
const sourceRoot = process.env.STUDIO_SOURCE_ROOT || repoRoot;

test('Studio and independent consumer resolve the same exact Editor, CodeMirror, Graph and Command source files', () => {
  const result = auditWorkbenchReuse({
    sourceRoot,
    expectedMainSha: process.env.STUDIO_MAIN_SOURCE_SHA || '',
  });
  assert.deepEqual(result.errors, []);
  assert.deepEqual(result.sources.map(source => source.name), ['editor', 'codemirror', 'graph', 'command']);
  assert.ok(result.sources.every(source => /^[0-9a-f]{40}$/.test(source.blob)));
  assert.ok(result.inspected.includes('reference/tei-studio-independent-consumer/editor-entry.mjs'));
});

test('exact source qualification rejects a different repository revision', () => {
  const result = auditWorkbenchReuse({ sourceRoot, expectedMainSha: '0'.repeat(40) });
  assert.ok(result.errors.some(error => error.includes('source revision mismatch')));
});
