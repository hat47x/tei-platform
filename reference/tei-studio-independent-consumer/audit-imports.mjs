// Narrow, fail-closed import-graph audit for this independent JavaScript consumer.
// Not a full parser or an authority/sandbox check; repeat on actual shared modules.
import { readFileSync, realpathSync, existsSync } from 'node:fs';
import { extname, dirname, resolve, relative, isAbsolute } from 'node:path';
import { fileURLToPath } from 'node:url';

const forbidden = [
  /(^|[/@])tei(?:-platform)?(?:\/|$)/i,
  /(?:^|\/)internal\/(?:operations|desktophost|servicehost)(?:\/|$)/i,
  /(?:^|\/)(?:workspace|acceptance|semantic-subject)(?:\/|$)/i,
];
const importPattern = /\b(?:import|export)\s+(?:[^'"`\n]*?\s+from\s*)?['"]([^'"\n]+)['"]|\bimport\s*\(\s*['"]([^'"\n]+)['"]\s*\)/g;
const extensions = ['', '.mjs', '.js', '.ts', '.mts', '/index.mjs', '/index.js'];

export function auditImportGraph(entryFiles, root) {
  const rootPath = realpathSync(root);
  const visited = new Set();
  const errors = [];
  const visit = path => {
    let physical;
    try { physical = realpathSync(path); }
    catch { errors.push(`unresolved import: ${path}`); return; }
    if (relative(rootPath, physical).startsWith('..') || isAbsolute(relative(rootPath, physical))) {
      errors.push(`module outside repository: ${physical}`);
      return;
    }
    if (visited.has(physical)) return;
    visited.add(physical);
    const source = readFileSync(physical, 'utf8');
    if (/\bimport\s*\(\s*(?!['"])/.test(source)) errors.push(`computed dynamic import: ${physical}`);
    for (const match of source.matchAll(importPattern)) {
      const specifier = match[1] ?? match[2];
      if (forbidden.some(re => re.test(specifier))) {
        errors.push(`TEI-specific dependency: ${physical} imports ${specifier}`);
      }
      if (!specifier.startsWith('.')) continue;
      const base = resolve(dirname(physical), specifier);
      const dest = extensions.map(ext => base + ext).find(candidate => existsSync(candidate));
      if (!dest) errors.push(`unresolved relative import: ${physical} imports ${specifier}`);
      else visit(dest);
    }
  };
  for (const file of entryFiles) visit(resolve(rootPath, file));
  return { inspected: [...visited].map(p => relative(rootPath, p)).sort(), errors };
}

if (process.argv[1] && realpathSync(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = resolve(fileURLToPath(new URL('.', import.meta.url)), '../../');
  const files = process.argv.slice(2);
  if (!files.length) throw new Error('Provide at least one entry module');
  const result = auditImportGraph(files.map(file => `reference/tei-studio-independent-consumer/${file}`), root);
  for (const file of result.inspected) console.log(`INSPECTED ${file}`);
  for (const error of result.errors) console.error(`FAIL ${error}`);
  if (result.errors.length) process.exitCode = 1;
  else console.log('PASS static import graph: no TEI-specific dependency found in inspected files (not a transitive bundler proof)');
}
