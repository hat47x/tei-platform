// E38: bridge a fixed real PR delta to the existing E28 workflow record.
// Only already-modified paths have old/new blob identities. New files and
// all human/AI actions remain unmeasured, not zero.
import {assessE28} from './e28_workflow_measurement.mjs';
import {observeE27} from './e27_residual_selection_audit.mjs';
import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';

const sha=x=>typeof x==='string'&&/^[a-f0-9]{40}$/.test(x);
const seenFile=x=>x&&x.status==='modified'&&
  x.path==='tools/reference-project-advice-lit/main.go'||x?.path==='tools/reference-project-advice-lit/README.md';
export function submitE38SourceOnly(audit,snapshot){
  if(snapshot?.kind!=='tei-e34-project-bridge-fixed-revision-delta-v0'||
      snapshot.repository!=='hat47x/tei-platform'||snapshot.pullRequest!==1567||
      !sha(snapshot.previousHead)||!sha(snapshot.currentHead)||
      !Array.isArray(snapshot.files)||snapshot.files.length!==4)
    throw new TypeError('invalid_e38_source');
  const old=new Set(),fresh=new Set(),events=[];
  for(const f of snapshot.files){
    if(!f||typeof f.path!=='string'||old.has(f.path)||!sha(f.afterBlob)||
       !Number.isSafeInteger(f.additions)||f.additions<0||
       !Number.isSafeInteger(f.deletions)||f.deletions<0)
      throw new TypeError('invalid_e38_diff');
    old.add(f.path);
    if(f.status==='added'){
      if(f.beforeBlob!==null||f.deletions!==0)throw new TypeError('invalid_e38_addition');
      fresh.add(f.path);
    }else if(f.status==='modified'&&seenFile(f)){
      if(!sha(f.beforeBlob)||f.beforeBlob===f.afterBlob)throw new TypeError('invalid_e38_prior_blob');
      events.push({
        id:'e38-'+String(events.length+1),choiceId:'web.ui-kit-target-adoption',
        operation:'changed-source',evidenceRef:'https://github.com/hat47x/tei-platform/pull/1567',
        repository:snapshot.repository,path:f.path,beforeBlob:f.beforeBlob,afterBlob:f.afterBlob,
      });
    }else throw new TypeError('invalid_e38_diff_path');
  }
  const required=['tools/reference-project-advice-lit/main.go',
    'tools/reference-project-advice-lit/README.md',
    'tools/reference-project-advice-lit/contract_bridge.go',
    'tools/reference-project-advice-lit/contract_bridge_test.go'];
  if(required.some(x=>!old.has(x))||fresh.size!==2||events.length!==2)
    throw new TypeError('incomplete_e38_diff');
  const submission={
    kind:'tei-e28-workflow-submission-v0',complete:false,
    scenario:{id:'order-entry-project-advice',consumer:'order_entry',
      arm:'existing-source',baseRevision:snapshot.previousHead,target:'web'},
    events,coverage:{humanDecisions:'untracked',aiQuestions:'untracked',
      manualConnections:'untracked',changedFiles:'untracked',
      validationRuns:'untracked',requalificationRuns:'untracked'},
  };
  const e28=assessE28(audit,submission);
  return {kind:'tei-e38-real-source-workflow-gap-v0',
    pinned:{before:snapshot.previousHead,after:snapshot.currentHead},
    modifiedSourceRecords:events.length,
    newlyAddedSourcePaths:fresh.size,
    sourceEventsSubmittedToE28:e28.submittedCounts.changedFiles,
    e28Scope:e28.scope,claimedCoveredCounts:e28.claimedCoveredCounts,
    verifiedCounts:e28.verifiedCounts,actualHumanDecisions:null,
    actualAIQuestions:null,actualManualConnections:null,
    improvementVerified:null,runtimeAuthorityVerified:false};
}
if(process.argv[1]&&import.meta.url===pathToFileURL(resolve(process.argv[1])).href){
  const audit=await observeE27(process.argv[2]||'.');
  const data=JSON.parse(await readFile(new URL('./e34_project_bridge_pr1567_delta.json',import.meta.url),'utf8'));
  process.stdout.write(JSON.stringify(submitE38SourceOnly(audit,data),null,2)+'\n');
}
