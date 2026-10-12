// This is a reference-only adapter between a selected DFD element and a
// constrained review edit. Shared Acceptance and Source Index remain separate.
import {proposeDFDReviewEdit, EDIT_SCHEMA} from './dfd-edit-proposal.mjs';

export const BINDING_SCHEMA='tei.reference.business-dfd-selected-edit/v0';
const SHA=/^sha256:[0-9a-f]{64}$/;
const KINDS=new Set(['step','trigger','party','store']);
function exact(id,name){
  if(typeof id!=='string'||!id||id!==id.trim())throw Error('INVALID_'+name);
  return id;
}
function rows(value,name){if(!Array.isArray(value))throw Error('INVALID_ARRAY_'+name);return value;}
function sameRows(actual,expected,name,project){
  const keys=v=>rows(v,name).map(project).map(x=>JSON.stringify(x)).sort();
  const a=keys(actual),b=keys(expected);
  if(a.length!==b.length||a.some((x,i)=>x!==b[i]))
    throw Error('DFD_REVIEW_SNAPSHOT_MISMATCH_'+name);
}
function sameSteps(dfd,procedure){
  const project=rows(procedure.steps,'PROCEDURE_STEPS');
  const a=rows(dfd.steps,'DFD_STEPS');
  sameRows(a,project,'STEPS',s=>{
    if(!s||typeof s!=='object')throw Error('INVALID_STEP');
    return {id:exact(s.id,'STEP_ID'),
      // The derived label and explicit Function references must match the
      // reference Procedure; a local Step is not Canonical World identity.
      label:String(s.label??s.businessAction??''),
      realization:s.realization,
      functions:(s.functionRefs||s.applicationPatterns||[]).map(r=>typeof r==='string'?r:r.id).sort()};
  });
}
// Producer annotations do not create accepted, runtime or authorization
// authority. A forged/stale projection must not override this boundary even
// when all authored labels and IDs happen to match.
function checkProducerStatus(dfd,procedure){
  if(dfd.roleAuthorizationBinding!=='not-inferred')
    throw Error('DFD_PRODUCER_AUTHORITY_BOUNDARY_MISMATCH');
  const sourceSteps=new Map(procedure.steps.map(s=>[s.id,s]));
  for(const step of dfd.steps){
    const source=sourceSteps.get(step.id);
    const expectedManual=(source?.applicationPatterns||[]).length===0;
    if(!source||step.noApplicationFunction!==expectedManual||
       !Array.isArray(step.functionRefs)||
       step.functionRefs.some(ref=>ref?.status!=='resolved'))
      throw Error('DFD_PRODUCER_STEP_RESOLUTION_MISMATCH');
  }
  const proposed=[
    ...dfd.flows,...dfd.information.roles,
    ...dfd.information.externalParties,...dfd.information.stores,
    ...dfd.information.responsibilities,...dfd.information.informationFlows
  ];
  if(proposed.some(item=>item?.authority!=='proposal')||
     dfd.triggers.some(t=>t?.authority!=='proposal'||t.runtimeBinding!=='unbound'))
    throw Error('DFD_PRODUCER_AUTHORITY_BOUNDARY_MISMATCH');
  for(const store of dfd.information.stores){
    const status=store.semanticData?'resolved':'not-declared';
    if(store.dataStatus!==status)throw Error('DFD_PRODUCER_STORE_RESOLUTION_MISMATCH');
  }
}
function align(dfd,procedure,review){
  if(!dfd||dfd.schemaVersion!=='business-dfd-projection-v0'||
      dfd.authority!=='derived-read-only'||dfd.canonicalProcedureUsed!==false||
      !Array.isArray(dfd.diagnostics)||dfd.diagnostics.some(d=>d?.severity==='error'))
    throw Error('INVALID_DFD_PRODUCER_AUTHORITY_OR_DIAGNOSTIC');
  if(!procedure||procedure.schemaVersion!=='business-procedure-reference-v0'||
      procedure.canonical!==false||procedure.artifactRole!=='reference-authoring-fixture'||
      !review||review.schemaVersion!=='business-dfd-review-v0'||review.authority!=='proposal')
    throw Error('NON_REFERENCE_SOURCE_NOT_SUPPORTED');
  const procedureId=exact(procedure.procedure?.id,'PROCEDURE_ID');
  if(dfd.procedureId!==procedureId||review.procedure!==procedureId)
    throw Error('DFD_REVIEW_PROCEDURE_IDENTITY_MISMATCH');
  sameSteps(dfd,procedure);
  sameRows(dfd.triggers,review.triggers,'TRIGGERS',
    x=>({id:x.id,kind:x.kind,targetStep:x.targetStep,label:x.label}));
  sameRows(dfd.flows,review.flows,'CONTROL',
    x=>({from:x.from,to:x.to,kind:x.kind,condition:x.condition||''}));
  const info=dfd.information;
  if(!info||typeof info!=='object')throw Error('MISSING_DFD_INFORMATION');
  sameRows(info.roles,review.businessRoles,'ROLES',x=>({id:x.id,label:x.label}));
  sameRows(info.externalParties,review.externalParties,'PARTIES',x=>({id:x.id,label:x.label}));
  sameRows(info.stores,review.stores,'STORES',
    x=>({id:x.id,label:x.label,semanticData:x.semanticData||''}));
  sameRows(info.responsibilities,review.responsibilities,'RESPONSIBILITIES',
    x=>({step:x.step,role:x.role}));
  // JSON property ordering is not an identity rule: Go and JS serializers
  // may emit the same endpoint fields in different orders.
  sameRows(info.informationFlows,review.informationFlows,'INFORMATION_FLOWS',
    x=>({id:x.id,
      from:{kind:x.from?.kind,id:x.from?.id},
      to:{kind:x.to?.kind,id:x.to?.id},
      payload:x.payload}));
  checkProducerStatus(dfd,procedure);
  return procedureId;
}
function selectedNode(dfd,kind,id) {
  if(!KINDS.has(kind))throw Error('UNSUPPORTED_DFD_SELECTION_KIND');
  const entries=kind==='step'?dfd.steps:kind==='trigger'?dfd.triggers:
    kind==='party'?dfd.information.externalParties:dfd.information.stores;
  const match=entries.filter(x=>x.id===id);
  if(match.length!==1)throw Error('UNRESOLVED_OR_AMBIGUOUS_DFD_SELECTION');
  return match[0];
}
function binds(selection,operation) {
  const kind=operation?.kind,sel=selection.kind,id=selection.id;
  if(kind==='assign-business-role')return sel==='step'&&id===operation.step;
  if(kind==='replace-trigger-label')return sel==='trigger'&&id===operation.trigger;
  if(kind==='add-control-transition')return sel==='step'&&id===operation.from;
  if(kind==='add-information-flow')return sel===operation.from?.kind&&id===operation.from?.id;
  return false;
}
// Input fingerprints must be obtained from actual source bytes by the caller;
// this pure adapter cannot authenticate a workspace or accepted model revision.
export function previewSelectedDFDEdit(dfd,procedure,review,binding){
  if(!binding||binding.schema!==BINDING_SCHEMA||
    !SHA.test(binding.reviewSHA256)||!SHA.test(binding.procedureSHA256)||
    !SHA.test(binding.projectionSHA256)||
    !['studio','ai','file','cli'].includes(binding.origin)||
    typeof binding.intent!=='string'||!binding.intent.trim()||
    typeof binding.id!=='string'||!binding.id.trim()||
    !binding.selection||typeof binding.selection!=='object')
    throw Error('INVALID_SELECTED_EDIT_BINDING');
  const allowed=['schema','reviewSHA256','procedureSHA256','projectionSHA256',
    'origin','intent','id','selection','operation'];
  if(Object.keys(binding).some(k=>!allowed.includes(k))||
    Object.keys(binding.selection).some(k=>!['kind','id'].includes(k)))
    throw Error('UNSUPPORTED_BINDING_ATTRIBUTE');
  const procId=align(dfd,procedure,review);
  const selection={kind:exact(binding.selection.kind,'SELECTION_KIND'),
    id:exact(binding.selection.id,'SELECTION_ID')};
  const node=selectedNode(dfd,selection.kind,selection.id);
  if(!binds(selection,binding.operation))
    throw Error('DFD_SELECTION_EDIT_TARGET_MISMATCH');
  const edit={schema:EDIT_SCHEMA,id:binding.id,origin:binding.origin,
    intent:binding.intent,baseReviewSHA256:binding.reviewSHA256,
    procedureId:procId,operation:binding.operation};
  const preview=proposeDFDReviewEdit(procedure,review,binding.reviewSHA256,edit);
  return {
    schema:'tei.reference.business-dfd-selected-review/v0',
    authority:'unaccepted-proposal-only',
    procedureId:procId,
    selected:{...selection,scope:'reference-procedure:'+procId,
      source:node.source||null,acceptedIdentity:'unverified'},
    inputFingerprints:{procedure:binding.procedureSHA256,
      review:binding.reviewSHA256,projection:binding.projectionSHA256,
      verification:'caller-supplied-unverified'},
    change:preview.proposedChange,
    proposedReview:preview.proposedReview,
    diagnostics:[],
    readyForAcceptance:false,
    editableCanonicalSource:false,
    sharedAcceptanceCandidate:null,
    sourceWrites:[],
    freshness:'unverified-no-model-revision-binding',
    nextAction:'human-review-and-explicit-shared-acceptance-handoff-required'
  };
}
