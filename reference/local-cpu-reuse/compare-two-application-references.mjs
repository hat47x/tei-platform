// Narrow R0 two-Application observation: same query invocation != reusable UI or business logic.
// Inputs are already-authorized, real TEI cli.inspect/v0 reports. No canonical writes or training.
import {projectCliInspectReport} from './inspect-cli-bridge.mjs';
import {linkApplicationPattern} from './project-inspection-v0.mjs';

export class CrossApplicationObservationError extends Error {
  constructor(code) {super(code);this.code=code;}
}
const valid=x=>typeof x==='string'&&x.trim().length>0;
const obj=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const keys=(v,allowed)=>obj(v)&&Object.keys(v).every(k=>allowed.includes(k));
const same=(a,b)=>JSON.stringify(a)===JSON.stringify(b);
const SUBJECTS=Object.freeze([
  {kind:'application',id:'order_entry'},
  {kind:'application-pattern',id:'order_entry.customer_reference'},
  {kind:'application',id:'repair_intake'},
  {kind:'application-pattern',id:'repair_intake.customer_reference'}
].map(Object.freeze));
export {SUBJECTS as TWO_APPLICATION_REFERENCE_SUBJECTS};
const fail=code=>{throw new CrossApplicationObservationError(code);};

function canonical(report) {
  let result=report?.result?.canonical;
  if(typeof result==='string') {
    try{result=JSON.parse(result);}catch{fail('INVALID_CANONICAL');}
  }
  return result;
}
function applicationSource(source) {
  if(!keys(source,['id','semanticUses','validations','databases','patterns']) ||
     !valid(source.id)||!Array.isArray(source.databases)||!Array.isArray(source.patterns))
    fail('UNVERIFIED_APPLICATION_MODEL');
  // These are the current Go model fields. Unknown fields may alter interpretation.
  if(source.databases.some(d=>!keys(d,['alias','target','access','requires']) ||
      !valid(d.alias)||!valid(d.target)||!valid(d.access)||!Array.isArray(d.requires)||
      d.requires.some(c=>!keys(c,['contract','compatibleMajor'])||!valid(c.contract)||
        !Number.isSafeInteger(c.compatibleMajor)||c.compatibleMajor<1))) fail('UNVERIFIED_DATABASE_ROUTE');
  return source;
}
function patternSource(source) {
  if(!keys(source,['id','kind','data','relation','query','capability','datasource','contractQuery'])||
    !valid(source.id)||source.kind!=='reference'||!valid(source.relation)||
    !valid(source.datasource)||!keys(source.contractQuery,['contract','query','inputs'])||
    !valid(source.contractQuery.contract)||!valid(source.contractQuery.query)||
    !Array.isArray(source.contractQuery.inputs)) fail('UNVERIFIED_REFERENCE_PATTERN');
  if(source.contractQuery.inputs.some(i=>!keys(i,['targetAttribute','input'])||
      !valid(i.targetAttribute)||!valid(i.input))) fail('UNVERIFIED_QUERY_BINDING');
  if(source.data!==undefined||source.query!==undefined||source.capability!==undefined)
    fail('UNMODELED_REFERENCE_CONSTRAINT');
  return source;
}
function invoke(reportApp,reportPattern) {
  const app=applicationSource(canonical(reportApp)),pattern=patternSource(canonical(reportPattern));
  const routes=app.databases.filter(d=>d.alias===pattern.datasource);
  if(routes.length!==1) fail('UNRESOLVED_DATASOURCE_ALIAS');
  const route=routes[0];
  if(route.access!=='read') fail('UNVERIFIED_READ_ONLY_ROUTE');
  const requirements=route.requires.filter(r=>r.contract===pattern.contractQuery.contract);
  if(requirements.length!==1) fail('UNRESOLVED_CONTRACT_REQUIREMENT');
  const bindings=pattern.contractQuery.inputs.map(i=>({targetAttribute:i.targetAttribute,input:i.input}))
    .sort((a,b)=>a.input<b.input?-1:a.input>b.input?1:a.targetAttribute<b.targetAttribute?-1:1);
  if(new Set(bindings.map(i=>i.input)).size!==bindings.length)fail('DUPLICATE_QUERY_BINDING');
  return {application:app.id,pattern:pattern.id,relation:pattern.relation,
    datasourceAlias:pattern.datasource,databaseTarget:route.target,
    access:route.access,contract:pattern.contractQuery.contract,
    compatibleMajor:requirements[0].compatibleMajor,query:pattern.contractQuery.query,
    bindings};
}

// Returns a reviewable declaration observation, *never* a shareable implementation contract.
// Exactly 4 fixed pre-authorized reports, not a general-purpose cross-tenant search.
export function compareTwoReferenceApplications(reports) {
  if(!Array.isArray(reports)||reports.length!==SUBJECTS.length)fail('INVALID_SUBJECT_BATCH');
  const observations=reports.map((report,i)=>projectCliInspectReport(report,SUBJECTS[i]));
  const provenance=observations[0].provenance;
  if(observations.some(o=>o.completeness!=='structurally_projected'||
     o.provenance.sourceStatus!=='indexed'))fail('INCOMPLETE_SOURCE_OBSERVATION');
  if(observations.some(o=>o.provenance.worldFingerprint!==provenance.worldFingerprint||
     o.provenance.sourceIndexFingerprint!==provenance.sourceIndexFingerprint))
    fail('SNAPSHOT_DRIFT');
  for(const [app,pattern] of [[0,1],[2,3]]) {
    if(linkApplicationPattern(observations[app],observations[pattern]).status!=='declared_membership')
      fail('UNVERIFIED_APPLICATION_MEMBERSHIP');
  }
  const left=invoke(reports[0],reports[1]),right=invoke(reports[2],reports[3]);
  if(left.application!==SUBJECTS[0].id||right.application!==SUBJECTS[2].id||
     left.pattern!==SUBJECTS[1].id||right.pattern!==SUBJECTS[3].id)
    fail('WRONG_CANONICAL_ID');
  const sharedInvocation=left.databaseTarget===right.databaseTarget&&
    left.access===right.access&&left.contract===right.contract&&
    left.compatibleMajor===right.compatibleMajor&&left.query===right.query&&
    same(left.bindings,right.bindings);
  const differingContext=['relation','datasourceAlias'].filter(k=>left[k]!==right[k]);
  return {
    schemaVersion:'tei.research.cross-application-reference/v0',
    effect:'read-only',authority:'derived_observation_only',acceptance:'not_asserted',
    contractConformance:'not_verified',
    sharedInvocationDeclared:sharedInvocation,
    contextDifferences:differingContext,
    observation:sharedInvocation?'shared_query_invocation_declared':'query_invocations_differ',
    reuseDecision:'needs_semantic_review',
    // Public checked-in Reference IDs only. Not a safe general-purpose stdout format.
    applications:[left.application,right.application],
    patterns:[left.pattern,right.pattern],
    declaredQuery:sharedInvocation?{databaseTarget:left.databaseTarget,contract:left.contract,
      compatibleMajor:left.compatibleMajor,query:left.query,bindings:left.bindings}:null,
    sourceSnapshot:{worldFingerprint:provenance.worldFingerprint,
      sourceIndexFingerprint:provenance.sourceIndexFingerprint}
  };
}

// A proposal about separation of declarations, NOT a callable Query adapter.
// Preserve the existing comparator's exact-subject, Source and Contract-route abstention
// checks before separating the shared invocation from local Application wiring.
export function proposeReferenceQueryAdapter(reports) {
  const comparison=compareTwoReferenceApplications(reports);
  // invoke() is safe here: compareTwoReferenceApplications validated both exact
  // Canonical rows, both routes, membership and shared World/Source snapshot.
  const left=invoke(reports[0],reports[1]);
  const right=invoke(reports[2],reports[3]);
  const appContext=[left,right].map(item=>({
    application:item.application,pattern:item.pattern,relation:item.relation,
    datasourceAlias:item.datasourceAlias
  }));
  return {
    schemaVersion:'tei.research.query-invocation-adapter-proposal/v0',
    proposalType:'declarative_read_only_query_invocation',
    status:comparison.sharedInvocationDeclared?
      'candidate_requires_semantic_review':'not_shareable_from_declarations',
    // The common layer is only the already-validated, observed Query declaration;
    // no runtime target/provider, credential, endpoint, cache or result shape.
    commonInvocation:comparison.sharedInvocationDeclared?
      {databaseTarget:comparison.declaredQuery.databaseTarget,
        contract:comparison.declaredQuery.contract,
        compatibleMajor:comparison.declaredQuery.compatibleMajor,
        query:comparison.declaredQuery.query,
        bindings:comparison.declaredQuery.bindings.map(x=>({...x}))}:null,
    // App-specific relation and alias deliberately never move into shared core.
    applicationContext:appContext,
    contextDifferences:[...comparison.contextDifferences],
    verified:{declarationMatch:comparison.sharedInvocationDeclared,
      contractSemantics:false,queryResultShape:false,readAuthorization:false,
      sourceAuthenticity:false,runtimeInteroperability:false},
    executionPermitted:false,implementationSelected:false,accepted:false,
    acceptance:'not_asserted',authority:'derived_observation_only',
    unresolvedGates:['contract_semantics','query_result_signature',
      'caller_authorization','runtime_binding','acceptance'],
    // In-memory Reference observation only; do not export snapshots for tenants.
    observationSnapshot:{...comparison.sourceSnapshot}
  };
}
