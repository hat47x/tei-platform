// #1016: read-only checks against known Source examples, not a TEI parser or authority resolver.
import {readFile, realpath} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {resolve, relative, isAbsolute, sep} from 'node:path';
import {pathToFileURL} from 'node:url';

export const probes = Object.freeze([
  {id:'order.application', group:'order_entry', path:'reference/workspace/applications/order-entry/application.tei', contains:'application order_entry {'},
  {id:'order.semantic-reference', group:'order_entry', path:'reference/workspace/applications/order-entry/application.tei', contains:'uses semantic sales'},
  {id:'order.database-contract', group:'order_entry', path:'reference/workspace/applications/order-entry/application.tei', contains:'contract sales_core.order ^4'},
  {id:'order.command-pattern', group:'order_entry', path:'reference/workspace/applications/order-entry/patterns/submit-order.tei', contains:'pattern order_entry.submit_order command'},
  {id:'order.dev-binding', group:'order_entry', path:'reference/workspace/environments/development/environment.tei', contains:'bind order_entry.sales -> sales_core.dev'},
  {id:'order.provider-bundle', group:'order_entry', path:'reference/workspace/environments/development/environment.tei', contains:'bundle tei.reference.providers ^1'},
  {id:'order.plugin-lock-entry', group:'order_entry', path:'reference/workspace/.tei/plugin-lock.json', contains:'"id": "tei.runtime.reference"'},
  {id:'order.web-endpoint', group:'order_entry', path:'tools/reference-first-web-target/server.go', contains:'/scenario/submit'},
  {id:'order.ui-adapter', group:'order_entry', path:'reference/ui-customization/web-react/order-entry-example.mjs', contains:'createReferenceSubmitAction'},
  {id:'order.project-ui-extension', group:'order_entry', path:'reference/ui-customization/web-react/project-extension.mjs', contains:'export function projectSidePanel'},
  {id:'sui.workspace-scope', group:'sui_stage0', path:'reference/sui-adoption/workspace/tei.workspace.yaml', contains:'  - semantic'},
  {id:'sui.semantic-definition', group:'sui_stage0', path:'reference/sui-adoption/workspace/semantic/sui/island.tei', contains:'namespace sui'},
  {id:'sui.semantic-relation', group:'sui_stage0', path:'reference/sui-adoption/workspace/semantic/sui/island.tei', contains:'relation parent to sui.island one'},
  {id:'sui.fixture-root', group:'sui_stage0', path:'reference/sui-adoption/M1-root.valid.json', contains:'"parentIslandId": null'},
]);

function withinRoot(base, target) {
  const rel = relative(base, target);
  return rel === '' || (rel !== '..' && !rel.startsWith('..' + sep) && !isAbsolute(rel));
}

// Only checks a specific textual observation. Found does not mean a valid TEI model,
// a resolved Provider, an Accepted change, or a current Runtime permission.
export async function observeSource(root, checks = probes) {
  const base = await realpath(root);
  const observations = [];
  for (const p of checks) {
    if (!p || !['id','group','path','contains'].every(k => typeof p[k] === 'string' && p[k].trim()) ||
        isAbsolute(p.path) || !withinRoot(base, resolve(base, p.path))) {
      throw new TypeError('invalid_probe');
    }
    const path = resolve(base, p.path);
    let target;
    try { target = await realpath(path); }
    catch (e) {
      if (e.code === 'ENOENT') {
        observations.push({id:p.id, group:p.group, path:p.path, status:'unresolved', reason:'missing-source'});
        continue;
      }
      throw e;
    }
    if (!withinRoot(base, target)) {
      observations.push({id:p.id, group:p.group, path:p.path, status:'unresolved', reason:'outside-root'});
      continue;
    }
    const data = await readFile(target);
    const matched = data.toString('utf8').includes(p.contains);
    observations.push({id:p.id, group:p.group, path:p.path,
      status:matched ? 'observed' : 'unresolved', reason:matched ? 'literal-present' : 'marker-missing',
      sha256:createHash('sha256').update(data).digest('hex')});
  }
  return {kind:'tei-application-composition-e1-source-observation-v0', observations};
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const report = await observeSource(process.argv[2] || '.');
  process.stdout.write(JSON.stringify(report, null, 2) + '\n');
  if (report.observations.some(o => o.status !== 'observed')) process.exitCode = 1;
}
