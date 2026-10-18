// Actual browser bundle. Unlike Node's stripTypeScriptTypes, esbuild resolves
// CodeMirror bare imports and their dependencies from THIS consumer's node_modules.
// Never copies / rewrites the #4/#5/#9 source modules.
import { build } from 'esbuild';
import { readFileSync, writeFileSync, mkdirSync, copyFileSync, statSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';

const here = dirname(fileURLToPath(import.meta.url));
const providers = {
  editor: '../tei-studio-editor/editor.ts',
  codemirror: '../tei-studio-editor/codemirror.ts',
  graph: '../tei-studio-graph/graph.ts',
  command: '../tei-studio-command/command.ts',
};
const lockBytes = readFileSync(resolve(here, 'package-lock.json'));
const npmLock = JSON.parse(lockBytes.toString('utf8'));
const exactPackages = {
  '@codemirror/commands': '6.11.1',
  '@codemirror/state': '6.7.6',
  '@codemirror/view': '6.43.14',
  '@codemirror/lint': '6.9.7',
  esbuild: '0.25.12',
};
for (const [name, version] of Object.entries(exactPackages)) {
  if (npmLock.packages?.['node_modules/' + name]?.version !== version)
    throw new Error(`LOCK_PACKAGE_MISMATCH: ${name}@${version}`);
}
const lockSha256 = createHash('sha256').update(lockBytes).digest('hex');
const hashes = {};
for (const [name, path] of Object.entries(providers)) {
  const file = resolve(here, path);
  try { if (!statSync(file).isFile()) throw new Error('not a file'); }
  catch { console.error(`BLOCKED: actual ${name} provider is missing: ${path}`); process.exit(2); }
  hashes[name] = createHash('sha256').update(readFileSync(file)).digest('hex');
}
const dist = resolve(here, 'dist');
const entries = Object.fromEntries(
  ['main', 'graph-entry', 'command-entry', 'editor-entry'].map(name =>
    [name, resolve(here, `${name}.mjs`)]),
);
const result = await build({
  absWorkingDir: here,
  entryPoints: entries,
  outdir: dist,
  entryNames: '[name]',
  outExtension: { '.js': '.mjs' },
  format: 'esm',
  bundle: true,
  splitting: false,
  platform: 'browser',
  target: ['es2022'],
  nodePaths: [resolve(here, 'node_modules')],
  metafile: true,
  minify: false,
  sourcemap: false,
  logLevel: 'warning',
});
const inputs = Object.keys(result.metafile.inputs).map(x => x.replaceAll('\\', '/'));
for (const path of Object.values(providers)) {
  const suffix = path.replace(/^\.\.\//, '');
  if (!inputs.some(x => x.endsWith(suffix))) {
    throw new Error(`REQUIRED_PROVIDER_NOT_BUNDLED: ${path}`);
  }
}
if (!inputs.some(x => x.includes('node_modules/@codemirror/commands/')) ||
    !inputs.some(x => x.includes('node_modules/@codemirror/state/')) ||
    !inputs.some(x => x.includes('node_modules/@codemirror/view/')) ||
    !inputs.some(x => x.includes('node_modules/@codemirror/lint/'))) {
  throw new Error('CODEMIRROR_BARE_IMPORTS_NOT_BUNDLED');
}
if (inputs.some(x => /(?:^|\/)(?:internal\/(?:operations|desktophost|servicehost)|workspace|acceptance|semantic-subject)(?:\/|$)/i.test(x))) {
  throw new Error('TEI_SPECIFIC_IMPLEMENTATION_IMPORTED');
}
mkdirSync(dist, { recursive: true });
copyFileSync(resolve(here, 'index.html'), resolve(dist, 'index.html'));
const provenance = { providerSourceSha256: hashes, entryPoints: Object.keys(entries),
  inputModules: inputs.sort(), npmLockSha256: lockSha256,
  bundler: 'esbuild@0.25.12', browser: 'esm' };
writeFileSync(resolve(dist, 'build-provenance.json'), JSON.stringify(provenance, null, 2) + '\n');
console.log(`PASS bundle: ${Object.keys(entries).join(', ')}; actual Editor/Graph/Command and CodeMirror linked`);
console.log('OUTPUT dist/index.html (serve dist/ on loopback; no raw TypeScript imports)');
