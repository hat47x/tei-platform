// Qualify that the independent consumer imports the exact shared UI source
// modules imported by the TEI Studio workbench at a pinned repository revision.
import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync, realpathSync } from 'node:fs';
import { dirname, isAbsolute, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { auditImportGraph } from './audit-imports.mjs';

const here = dirname(fileURLToPath(import.meta.url));
const defaultRoot = resolve(here, '../../');
const args = process.argv.slice(2);
const positional = args.filter(value => !value.startsWith('--'));
const requireBundle = args.includes('--require-bundle');

const componentImports = [
  { name: 'editor', symbol: 'createReadOnlyTextEditor', workbench: 'reference/tei-studio-workbench/main.ts', consumer: 'reference/tei-studio-independent-consumer/editor-entry.mjs' },
  { name: 'codemirror', symbol: 'codeMirrorRenderer', workbench: 'reference/tei-studio-workbench/main.ts', consumer: 'reference/tei-studio-independent-consumer/editor-entry.mjs' },
  { name: 'graph', symbol: 'mountReadonlyGraph', workbench: 'reference/tei-studio-workbench/main.ts', consumer: 'reference/tei-studio-independent-consumer/graph-entry.mjs' },
  { name: 'command', symbol: 'mountCommandPanel', workbench: 'reference/tei-studio-workbench/main.ts', consumer: 'reference/tei-studio-independent-consumer/command-entry.mjs' },
];

function parseNamedImports(source) {
  const imports = [];
  const pattern = /import\s*\{([\s\S]*?)\}\s*from\s*['"]([^'"]+)['"]/g;
  for (const match of source.matchAll(pattern)) {
    const names = match[1].split(',').map(value => value.trim()).filter(Boolean)
      .map(value => value.split(/\s+as\s+/)[0].trim());
    imports.push({ names, specifier: match[2] });
  }
  return imports;
}

function importedPath(root, file, symbol) {
  const sourcePath = resolve(root, file);
  if (!existsSync(sourcePath)) throw new Error(`missing source file: ${file}`);
  const source = readFileSync(sourcePath, 'utf8');
  const matches = parseNamedImports(source).filter(item => item.names.includes(symbol));
  if (matches.length !== 1) throw new Error(`${file} must import ${symbol} exactly once`);
  if (!matches[0].specifier.startsWith('.')) throw new Error(`${file} imports ${symbol} from a package, not shared source`);
  const target = resolve(dirname(sourcePath), matches[0].specifier);
  if (!existsSync(target)) throw new Error(`${file} import target does not exist: ${matches[0].specifier}`);
  const physical = realpathSync(target);
  const fromRoot = relative(realpathSync(root), physical);
  if (fromRoot.startsWith('..') || isAbsolute(fromRoot)) throw new Error(`${symbol} resolves outside repository: ${physical}`);
  return { physical, relativePath: fromRoot.replaceAll('\\', '/') };
}

function sha256(path) {
  return createHash('sha256').update(readFileSync(path)).digest('hex');
}

function gitHead(root) {
  return execFileSync('git', ['-C', root, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
}

function gitBlobAt(root, revision, path) {
  return execFileSync('git', ['-C', root, 'rev-parse', `${revision}:${path.replaceAll('\\', '/')}`], { encoding: 'utf8' }).trim();
}

function verifyCommitSnapshot(root, revision, paths, errors) {
  const checked = [];
  for (const path of [...new Set(paths.map(value => value.replaceAll('\\', '/')))]) {
    try {
      const committed = gitBlobAt(root, revision, path);
      const working = execFileSync('git', ['-C', root, 'hash-object', resolve(root, path)], { encoding: 'utf8' }).trim();
      if (committed !== working) errors.push(`source differs from pinned commit ${revision}: ${path}`);
      checked.push(path);
    } catch (error) { errors.push(`cannot verify pinned source ${path}: ${error.message}`); }
  }
  return checked;
}

function verifyImportGraph(root, errors) {
  const entries = [
    'main.mjs', 'notes.mjs', 'journal-editor.mjs', 'journal-graph.mjs',
    'journal-command.mjs', 'editor-entry.mjs', 'graph-entry.mjs', 'command-entry.mjs',
  ].map(file => `reference/tei-studio-independent-consumer/${file}`);
  const result = auditImportGraph(entries, root);
  errors.push(...result.errors);
  return result.inspected.map(path => path.replaceAll('\\', '/'));
}

function verifyBundle(root, componentSources, errors) {
  const consumerDir = resolve(root, 'reference/tei-studio-independent-consumer');
  const provenancePath = resolve(consumerDir, 'dist/build-provenance.json');
  if (!existsSync(provenancePath)) {
    errors.push('missing built bundle provenance: reference/tei-studio-independent-consumer/dist/build-provenance.json');
    return null;
  }
  let provenance;
  try { provenance = JSON.parse(readFileSync(provenancePath, 'utf8')); }
  catch (error) { errors.push(`invalid bundle provenance: ${error.message}`); return null; }

  const lockPath = resolve(consumerDir, 'package-lock.json');
  const lockSha256 = sha256(lockPath);
  if (provenance.npmLockSha256 !== lockSha256) errors.push('bundle provenance npm lock hash does not match current lock');
  if (provenance.bundler !== 'esbuild@0.25.12') errors.push(`unexpected bundle tool: ${provenance.bundler}`);
  for (const { name, physical } of componentSources) {
    if (provenance.providerSourceSha256?.[name] !== sha256(physical)) {
      errors.push(`bundle provenance source hash mismatch: ${name}`);
    }
    const suffix = physical.replaceAll('\\', '/').split('/reference/').at(-1);
    if (!provenance.inputModules?.some(path => path.replaceAll('\\', '/').endsWith(suffix))) {
      errors.push(`built bundle does not include exact shared source: ${name} (${suffix})`);
    }
  }
  for (const dependency of ['@codemirror/state', '@codemirror/view', '@codemirror/lint']) {
    if (!provenance.inputModules?.some(path => path.replaceAll('\\', '/').includes(`node_modules/${dependency}/`))) {
      errors.push(`built bundle does not include ${dependency}`);
    }
  }
  if (provenance.inputModules?.some(path => /(?:^|\/)(?:internal\/(?:operations|desktophost|servicehost)|workspace|acceptance|semantic-subject)(?:\/|$)/i.test(path))) {
    errors.push('built bundle includes a forbidden TEI-specific module');
  }
  return { lockSha256, inputs: provenance.inputModules?.length ?? 0, bundler: provenance.bundler };
}

export function auditWorkbenchReuse({ sourceRoot = defaultRoot, expectedMainSha = '', requireBuiltBundle = false } = {}) {
  const root = realpathSync(resolve(sourceRoot));
  const errors = [];
  let actualSha = '';
  try {
    actualSha = gitHead(root);
    if (expectedMainSha && actualSha !== expectedMainSha) {
      errors.push(`source revision mismatch: expected ${expectedMainSha}, found ${actualSha}`);
    }
  } catch (error) { errors.push(`cannot read source revision: ${error.message}`); }

  const sources = [];
  for (const item of componentImports) {
    try {
      const workbench = importedPath(root, item.workbench, item.symbol);
      const consumer = importedPath(root, item.consumer, item.symbol);
      if (workbench.physical !== consumer.physical) {
        errors.push(`${item.name} source differs: Studio=${workbench.relativePath}, consumer=${consumer.relativePath}`);
      }
      sources.push({ name: item.name, physical: workbench.physical, path: workbench.relativePath, blob: gitBlobAt(root, actualSha || 'HEAD', workbench.relativePath), sha256: sha256(workbench.physical) });
    } catch (error) { errors.push(`${item.name}: ${error.message}`); }
  }

  const inspected = verifyImportGraph(root, errors);
  const snapshotFiles = verifyCommitSnapshot(root, actualSha || 'HEAD', [
    'reference/tei-studio-workbench/main.ts', ...inspected, ...sources.map(source => source.path),
  ], errors);
  const bundle = requireBuiltBundle ? verifyBundle(root, sources, errors) : null;
  return { actualSha, sources, inspected, snapshotFiles, bundle, errors };
}

if (process.argv[1] && realpathSync(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const sourceRoot = positional[0] ?? process.env.STUDIO_SOURCE_ROOT ?? defaultRoot;
  const expectedMainSha = positional[1] ?? process.env.STUDIO_MAIN_SOURCE_SHA ?? '';
  const result = auditWorkbenchReuse({ sourceRoot, expectedMainSha, requireBuiltBundle: requireBundle });
  console.log(`SOURCE_REVISION ${result.actualSha || 'unavailable'}`);
  for (const source of result.sources) console.log(`SHARED_SOURCE ${source.name} ${source.path} blob=${source.blob} sha256=${source.sha256}`);
  console.log(`INSPECTED_CONSUMER_MODULES ${result.inspected.length}`);
  console.log(`PINNED_COMMIT_FILES ${result.snapshotFiles.length}`);
  if (result.bundle) console.log(`BUILT_BUNDLE esbuild=${result.bundle.bundler} inputs=${result.bundle.inputs} lock_sha256=${result.bundle.lockSha256}`);
  for (const error of result.errors) console.error(`FAIL ${error}`);
  if (result.errors.length) process.exitCode = 1;
  else console.log(`PASS exact shared-source reuse${requireBundle ? ' and built-bundle audit' : ''}`);
}
