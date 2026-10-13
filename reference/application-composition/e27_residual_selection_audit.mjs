// E27 is a read-only selection audit of E1 Source observations. It does not
// create a composition profile, persist review choices or grant permissions.
import {observeSource, probes} from './e1-observe.mjs';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';

const choiceSpecs = Object.freeze([
  {id:'web.application-action', consumer:'order_entry', class:'existing-source',
    evidence:['order.application','order.command-pattern'], owner:'TEI Application'},
  {id:'web.database-binding', consumer:'order_entry', class:'existing-source',
    evidence:['order.database-contract','order.dev-binding'], owner:'TEI Application and Environment'},
  {id:'web.provider-reference', consumer:'order_entry', class:'existing-source',
    evidence:['order.provider-bundle','order.plugin-lock-entry'], owner:'TEI Environment and Plugin Lock'},
  {id:'web.project-ui-extension', consumer:'order_entry', class:'existing-source',
    evidence:['order.ui-adapter','order.project-ui-extension'], owner:'Project UI Source'},
  {id:'web.ui-kit-target-adoption', consumer:'order_entry', class:'selection-required',
    evidence:['order.ui-adapter','order.web-endpoint'], owner:'Project adoption review'},
  {id:'web.current-runtime-authority', consumer:'order_entry', class:'runtime-proof-required',
    evidence:['order.web-endpoint'], owner:'Runtime permission and deployed endpoint'},
  {id:'sui.stage0-meaning', consumer:'sui_stage0', class:'stage0-only',
    evidence:['sui.workspace-scope','sui.semantic-definition','sui.semantic-relation'],
    owner:'TEI SUI Stage 0 Reference'},
  {id:'sui.native-action-contract', consumer:'sui_stage0', class:'selection-required',
    evidence:['sui.workspace-scope'], owner:'SUI-native Action contract / TEI adoption review'},
  {id:'sui.native-kit-target', consumer:'sui_stage0', class:'selection-required',
    evidence:['sui.workspace-scope'], owner:'SUI Native framework and Target review'},
  {id:'sui.authorized-backend-commit', consumer:'sui_stage0', class:'runtime-proof-required',
    evidence:['sui.workspace-scope'], owner:'SUI #993 authenticated session and CAS'},
]);

const validClasses = new Set(['existing-source','selection-required','runtime-proof-required','stage0-only']);
const knownProbes = new Map(probes.map(p=>[p.id,p]));

// The entries are a *reference-only checklist*, not an authoritative list of
// everything every application needs. "observed" only means E1 matched a
// literal in the current TEI checkout. External SUI source is not read here.
export function auditE27(observations) {
  if (!Array.isArray(observations)) throw new TypeError('invalid_observations');
  const seen = new Map();
  for (const item of observations) {
    const known = knownProbes.get(item?.id);
    if (!known || seen.has(item.id) || item.group !== known.group ||
        item.path !== known.path || !['observed','unresolved'].includes(item.status) ||
        (item.status === 'observed' &&
         (item.reason !== 'literal-present' || !/^[a-f0-9]{64}$/.test(item.sha256)))) {
      throw new TypeError('invalid_e1_observation');
    }
    seen.set(item.id, item);
  }
  const result = choiceSpecs.map(spec => {
    if (!validClasses.has(spec.class)) throw new TypeError('invalid_choice_class');
    const missing = spec.evidence.filter(id=>seen.get(id)?.status !== 'observed');
    const status = missing.length ? 'source-unresolved' :
      spec.class === 'existing-source' ? 'source-observed' :
      spec.class === 'stage0-only' ? 'stage0-observed-only' :
      spec.class === 'selection-required' ? 'selection-unresolved' :
      'runtime-proof-unresolved';
    return {id:spec.id, consumer:spec.consumer, class:spec.class,
      owner:spec.owner, status, evidence:spec.evidence, missing};
  });
  const summary = {
    scope:'bounded-e1-tei-source-only',
    sourceObserved:result.filter(r=>r.status==='source-observed').length,
    stage0ObservedOnly:result.filter(r=>r.status==='stage0-observed-only').length,
    selectionUnresolved:result.filter(r=>r.status==='selection-unresolved').length,
    runtimeProofUnresolved:result.filter(r=>r.status==='runtime-proof-unresolved').length,
    sourceUnresolved:result.filter(r=>r.status==='source-unresolved').length,
    actualManualActions:null, actualAIQuestions:null, acceptedProfiles:null,
  };
  return {kind:'tei-composition-e27-reference-selection-audit-v0', summary, choices:result};
}

export async function observeE27(root) {
  const observed = await observeSource(root);
  return auditE27(observed.observations);
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const report = await observeE27(process.argv[2] || '.');
  process.stdout.write(JSON.stringify(report,null,2)+'\n');
  if (report.summary.sourceUnresolved) process.exitCode = 1;
}
