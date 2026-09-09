// Requirements HTML is a derived view. Edit the Markdown authority, then run this.
import { readFile, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { fileURLToPath, pathToFileURL } from 'node:url';
import path from 'node:path';
const root = fileURLToPath(new URL('../../', import.meta.url));
const moduleRoot = process.env.CODEX_PRIMARY_RUNTIME_NODE_MODULES;
const { marked } = moduleRoot
  ? await import(pathToFileURL(path.join(moduleRoot, 'marked/lib/marked.esm.js')))
  : await import('marked');
const sourcePath = path.join(root, 'requirements/TEI_PRODUCT_REQUIREMENTS.md');
const outputPath = path.join(root, 'docs/specification/requirements.html');
const markdown = await readFile(sourcePath, 'utf8');
const hash = createHash('sha256').update(markdown).digest('hex');
const ids = [...markdown.matchAll(/^## (RQ-\d{3}) /gm)].map(m => m[1]);
if (ids.length !== 26 || new Set(ids).size !== ids.length) {
  throw new Error('Requirement list changed: review HTML navigation and count explicitly.');
}
let body = marked.parse(markdown, {gfm: true});
body = body.replace(/<h1>(.*?)<\/h1>/g, '<h2>$1</h2>')
  .replace(/<h2>(RQ-\d{3}) /g, '<h2 id="$1">$1 ')
  .replace(/<table>/g, '<div class="table-wrap"><table>')
  .replace(/<\/table>/g, '</table></div>');
const nav = '<nav class="toc" aria-label="要求一覧">' + ids.map(id => `<a href="#${id}">${id}</a>`).join('') + '</nav>';
const fragment = `<!-- REQUIREMENTS:START -->\n<p class="meta">Generated from requirements/TEI_PRODUCT_REQUIREMENTS.md · SHA-256 <code>${hash}</code></p>\n${nav}\n${body}\n<!-- REQUIREMENTS:END -->`;
const old = await readFile(outputPath, 'utf8');
const matcher = /<!-- REQUIREMENTS:START -->[\s\S]*?<!-- REQUIREMENTS:END -->/;
if (!matcher.test(old)) throw new Error('Generated-region markers are missing');
const result = old.replace(matcher, () => fragment);
if (process.argv.includes('--check')) {
  if (result !== old) throw new Error('requirements.html is stale; run npm run render');
  console.log('Requirements HTML: exact regeneration verified (26 requirements)');
} else {
  await writeFile(outputPath, result);
  console.log('Requirements HTML rendered (26 requirements)');
}
