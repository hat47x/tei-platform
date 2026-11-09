import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import { auditImportGraph } from './audit-imports.mjs';
import { availability } from './check-components.mjs';

const repoRoot = resolve(fileURLToPath(new URL('.', import.meta.url)), '../../');

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

test('Windows drive-letter file URLs convert to native paths (not URL pathnames)', () => {
  const location = new URL('file:///C:/snapshot/reference/tei-studio-independent-consumer/');
  assert.equal(
    fileURLToPath(location, { windows: true }),
    'C:\\snapshot\\reference\\tei-studio-independent-consumer\\',
  );
  assert.notEqual(fileURLToPath(location, { windows: true }), location.pathname);
});

test('component availability CLI explicitly reports BLOCKED with exit code 2', () => {
  const script = fileURLToPath(new URL('./check-components.mjs', import.meta.url));
  const result = spawnSync(process.execPath, [script], { encoding: 'utf8' });
  assert.equal(result.error, undefined);
  assert.equal(result.status, 2, result.stderr || result.stdout);
  assert.match(result.stdout, /BLOCKED editor: path not specified/);
  assert.match(result.stdout, /BLOCKED graph: path not specified/);
});

test('import audit CLI resolves its checkout root and exits successfully', () => {
  const script = fileURLToPath(new URL('./audit-imports.mjs', import.meta.url));
  const result = spawnSync(process.execPath, [script, 'main.mjs', 'notes.mjs'], { encoding: 'utf8' });
  assert.equal(result.error, undefined);
  assert.equal(result.status, 0, result.stderr || result.stdout);
  assert.match(result.stdout, /PASS static import graph/);
});
