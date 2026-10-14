import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { auditImportGraph } from './audit-imports.mjs';
import { availability } from './check-components.mjs';

const repoRoot = resolve(new URL('.', import.meta.url).pathname, '../../');

test('sample imports contain no TEI-specific modules', () => {
  const result = auditImportGraph(['reference/tei-studio-independent-consumer/main.mjs'], repoRoot);
  assert.deepEqual(result.errors, []);
  assert.equal(result.inspected.length, 2);
});

test('scanner detects transitive TEI dependency, missing import, and computed dynamic import', () => {
  const root = mkdtempSync(join(tmpdir(), 'tei-independent-audit-'));
  try {
    mkdirSync(join(root, 'demo'));
    writeFileSync(join(root, 'demo', 'entry.mjs'), "import './shared.mjs';\n");
    writeFileSync(join(root, 'demo', 'shared.mjs'), "import '@tei/internal/operations';\n");
    const rejected = auditImportGraph(['demo/entry.mjs'], root);
    assert.ok(rejected.errors.some(value => value.includes('TEI-specific dependency')));
    writeFileSync(join(root, 'demo', 'shared.mjs'), "import './missing.mjs';\nimport(variable);\n");
    const unresolved = auditImportGraph(['demo/entry.mjs'], root);
    assert.ok(unresolved.errors.some(value => value.includes('unresolved relative import')));
    assert.ok(unresolved.errors.some(value => value.includes('computed dynamic import')));
  } finally { rmSync(root, { recursive: true, force: true }); }
});

test('missing shared components do not count as integrated', () => {
  const status = availability([]);
  assert.equal(status.found.length, 0);
  assert.equal(status.missing.length, 3);
  assert.equal(status.reusable, false);
});
