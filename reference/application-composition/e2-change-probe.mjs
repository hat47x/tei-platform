// #1016: Two narrowly scoped, non-mutating change proposals for existing TEI sources.
// This is neither a DSL parser nor Plugin/Runtime authority. It does not write files.
import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';

export const uiPath = 'reference/ui-customization/web-react/project-extension.mjs';
export const environmentPath = 'reference/workspace/environments/development/environment.tei';
const previousLabel = "React.createElement('h2',null,'Details')";
const oldBinding = 'bind order_entry.sales -> sales_core.dev';
const token = /^[A-Za-z][A-Za-z0-9._-]*$/;

function once(source, needle) {
  if (typeof source !== 'string' || source.split(needle).length !== 2) {
    return null; // Missing or ambiguous anchor: do not invent a patch.
  }
  return source;
}

// The proposed edit is deliberately limited to one visible UI heading.
export function proposeUIHeading(source, newLabel = '案件の詳細') {
  if (typeof newLabel !== 'string' || !newLabel.trim() || newLabel.includes("'") ||
      newLabel.includes('\\') || /[\r\n]/.test(newLabel)) {
    return {status:'unresolved', reason:'unsafe-ui-label'};
  }
  if (once(source, previousLabel) === null) return {status:'unresolved', reason:'ui-anchor-drift'};
  return {
    status:'proposed', kind:'ui-heading', path:uiPath, owner:'project-ui',
    after:source.replace(previousLabel, `React.createElement('h2',null,'${newLabel}')`),
    checks:['ui-extension-test','react-dom-and-accessibility-not-yet-verified'],
    doesNotChange:['accepted-business-meaning','application-action-contract','environment-binding'],
  };
}

// Only the already-declared Environment-owned binding is eligible for change.
// The candidate target must be an exact instance declared in the same Environment.
export function proposeDevelopmentDatabaseBinding(source, newInstance) {
  if (typeof newInstance !== 'string' || !token.test(newInstance)) {
    return {status:'unresolved', reason:'invalid-instance-id'};
  }
  if (once(source, oldBinding) === null) return {status:'unresolved', reason:'binding-anchor-drift'};
  if (newInstance === 'sales_core.dev') return {status:'unchanged'};
  const declarations = [...source.matchAll(/database-instance\s+([A-Za-z][A-Za-z0-9._-]*)\s*\{([^{}]*)\}/g)];
  const instances = declarations.filter(m => m[1] === newInstance);
  if (instances.length !== 1) return {status:'unresolved', reason:'instance-not-unique-or-missing'};
  if (!new RegExp('(?:^|\\n)\\s*definition\\s+sales_core\\s*(?:\\n|$)').test(instances[0][2])) {
    return {status:'unresolved', reason:'target-definition-mismatch'};
  }
  // Provider conformance, network, credentials, migrations, etc. are not validated by this probe.
  // Do not label the candidate as fully compatible or executable.
  return {
    status:'proposed-unqualified', kind:'environment-binding', path:environmentPath,
    owner:'environment', after:source.replace(oldBinding, `bind order_entry.sales -> ${newInstance}`),
    checks:['application-database-contract','provider-capability-and-migration',
      'secret-network-resource-permission','connection-and-isolation','actual-runtime-evidence'],
    doesNotChange:['accepted-business-meaning','ui-composition-kit'],
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const root = resolve(process.argv[2] || '.');
  const [ui, environment] = await Promise.all([
    readFile(resolve(root,uiPath),'utf8'), readFile(resolve(root,environmentPath),'utf8'),
  ]);
  const proposal = {
    ui:proposeUIHeading(ui),
    database:proposeDevelopmentDatabaseBinding(environment,'sales_core.dev_v2'),
  };
  // Never print source or proposed full content: the output is a diagnostic, not a patch to apply.
  for (const change of Object.values(proposal)) delete change.after;
  process.stdout.write(JSON.stringify(proposal, null, 2) + '\n');
}
