// E32: account for a pinned GitHub PR diff without confusing source-change
// metadata with implementation effort, Provider selection, or Runtime grants.
// A caller supplies the PR/files snapshot: no GitHub calls happen at runtime.
import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';

const hash = value => typeof value === 'string' && /^[0-9a-f]{40}$/.test(value);
const safePath = value => typeof value === 'string' && value.length > 0 &&
  !value.startsWith('/') && !value.includes('\\') &&
  value.split('/').every(part => part !== '' && part !== '.' && part !== '..');

function expectedRole(path) {
  if (path.startsWith('.github/workflows/')) return 'verification';
  if (path.endsWith('/README.md')) return 'documentation';
  if (path.startsWith('plugins/reference-project-customer-advice/')) return 'project-logic';
  if (/\/(main|main_test|order_entry_context|order_entry_context_test)\.go$/.test(path))
    return 'diagnostic-host';
  if (/\/(real_browser_replay\.mjs|smoke_order_entry\.sh)$/.test(path))
    return 'verification';
  if (path.endsWith('/lit-core.min.js') || path.endsWith('/LIT-LICENSE'))
    return 'third-party';
  if (path.startsWith('tools/reference-project-advice-lit/web/')) return 'web-ui';
  return null;
}

const roleNames = ['documentation','project-logic','diagnostic-host',
  'web-ui','third-party','verification'];

export function summarizeE32PR(snapshot) {
  if (snapshot?.kind !== 'tei-e32-github-pr-files-evidence-v0' ||
      snapshot?.source !== 'github-pr-files-fixed-head' ||
      snapshot.repository !== 'hat47x/tei-platform' ||
      !Number.isSafeInteger(snapshot.pullRequest) || snapshot.pullRequest <= 0 ||
      !hash(snapshot.baseSha) || !hash(snapshot.headSha) ||
      snapshot.baseSha === snapshot.headSha ||
      typeof snapshot.draftAtObservation !== 'boolean' ||
      typeof snapshot.mergedAtObservation !== 'boolean' ||
      !Array.isArray(snapshot.files) || snapshot.files.length === 0)
    throw new TypeError('invalid_e32_snapshot');

  const seen = new Set();
  const byRole = Object.fromEntries(roleNames.map(name =>
    [name,{files:0,addedLines:0}]));
  let additions = 0, deletions = 0, existingModified = 0;
  for (const file of snapshot.files) {
    if (!file || !safePath(file.path) || seen.has(file.path) ||
        file.role !== expectedRole(file.path) || !hash(file.gitBlobSha) ||
        !['added','modified','removed','renamed'].includes(file.status) ||
        !Number.isSafeInteger(file.additions) || file.additions < 0 ||
        !Number.isSafeInteger(file.deletions) || file.deletions < 0 ||
        (file.status === 'added' && file.deletions !== 0))
      throw new TypeError('invalid_e32_file_evidence');
    seen.add(file.path);
    additions += file.additions;
    deletions += file.deletions;
    if (file.status !== 'added') existingModified++;
    byRole[file.role].files++;
    byRole[file.role].addedLines += file.additions;
  }

  return {
    kind:'tei-e32-pr-footprint-review-v0',
    snapshot:{repository:snapshot.repository,pr:snapshot.pullRequest,
      baseSha:snapshot.baseSha,headSha:snapshot.headSha,
      draftAtObservation:snapshot.draftAtObservation,
      mergedAtObservation:snapshot.mergedAtObservation},
    observed:{changedFiles:seen.size,filesAdded:snapshot.files.length-existingModified,
      filesNotAdded:existingModified,additions,deletions,byRole},
    verification:'caller-supplied-pinned-pr-metadata-not-live-refetched',
    actualHumanDecisions:null,
    actualAIQuestions:null,
    actualManualConnections:null,
    additionalProfileBenefit:null,
    formalProviderSelectionProven:null,
    acceptedBusinessMeaningProven:null,
    runtimeAuthorityProven:null,
    productionAdoptionProven:null,
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const url = new URL('./e32_project_plugin_pr1567_snapshot.json',import.meta.url);
  const data = JSON.parse(await readFile(url,'utf8'));
  process.stdout.write(JSON.stringify(summarizeE32PR(data),null,2)+'\n');
}
