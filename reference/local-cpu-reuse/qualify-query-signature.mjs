// Research-only qualification of a declared Query signature against Go's normalized M0 World.
// Caller must authorize the entire World before supplying it. Never log the World or raw CLI.
import {isDeepStrictEqual} from 'node:util';
import {proposeReferenceQueryAdapter, TWO_APPLICATION_REFERENCE_SUBJECTS} from './compare-two-application-references.mjs';

export class QuerySignatureEvidenceError extends Error {
  constructor(code) {super(code);this.code=code;}
}
const fail=code=>{throw new QuerySignatureEvidenceError(code);};
const object=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const nonempty=x=>typeof x==='string'&&x.trim().length>0;
const shape=(x,keys)=>object(x)&&Object.keys(x).every(k=>keys.includes(k));
const exactlyOne=(items,pred,code)=>{
  if(!Array.isArray(items))fail(code);
  const found=items.filter(pred);
  if(found.length!==1)fail(code);
  return found[0];
};
const inputModel=(x)=>shape(x,['name','type'])&&nonempty(x.name)&&nonempty(x.type);
const cardinality=new Set(['one','zero-or-one','many']);
const parseCanonical=report=>{
  let x=report?.result?.canonical;
  if(typeof x==='string'){try{x=JSON.parse(x);}catch{fail('UNREADABLE_CANONICAL');}}
  if(!object(x))fail('UNREADABLE_CANONICAL');
  return x;
};
function matchWorldApplications(world,reports) {
  // The CLI 'world' and four 'inspect' invocations reload the Workspace separately.
  // Guard this important overlap instead of treating two independent reads as atomic.
  for(const [appIndex,patternIndex] of [[0,1],[2,3]]) {
    const appId=TWO_APPLICATION_REFERENCE_SUBJECTS[appIndex].id;
    const pId=TWO_APPLICATION_REFERENCE_SUBJECTS[patternIndex].id;
    const app=exactlyOne(world.applications,x=>x?.id===appId,'WORLD_APPLICATION_NOT_UNIQUE');
    if(!isDeepStrictEqual(app,parseCanonical(reports[appIndex])))
      fail('WORLD_INSPECT_APPLICATION_DRIFT');
    const p=exactlyOne(app.patterns,x=>x?.id===pId,'WORLD_PATTERN_NOT_UNIQUE');
    if(!isDeepStrictEqual(p,parseCanonical(reports[patternIndex])))
      fail('WORLD_INSPECT_PATTERN_DRIFT');
  }
}
function findAttributeType(world,attributeId) {
  if(!Array.isArray(world.semantics))fail('SEMANTIC_TYPE_UNRESOLVED');
  const matches=[];
  for(const ns of world.semantics) {
    if(!Array.isArray(ns?.data))continue;
    for(const data of ns.data)for(const attr of data.attributes||[]) {
      if(attr?.id===attributeId)matches.push(attr);
    }
  }
  if(matches.length!==1||!shape(matches[0],['id','symbol','type'])||!nonempty(matches[0].type))
    fail('SEMANTIC_TYPE_UNRESOLVED');
  return matches[0].type;
}
function checkResultData(world,id) {
  if(!Array.isArray(world.semantics))fail('QUERY_RESULT_DATA_NOT_FOUND');
  let count=0;
  for(const ns of world.semantics)for(const data of ns.data||[])
    if(data?.id===id)count++;
  if(count!==1)fail('QUERY_RESULT_DATA_NOT_FOUND');
}
/** One narrowly-scoped checked-in Reference Workspace only.
 * 'world' must come from the existing Go 'tei world -workspace' normalized M0 command.
 * Snapshot source authenticity or concurrent atomicity are NOT certified.
 */
export function qualifyReferenceQuerySignature(world,reports) {
  if(!object(world)||!Array.isArray(world.applications)||!Array.isArray(world.databases)||
     !Array.isArray(reports)||reports.length!==4)
    fail('INVALID_WORLD_OR_REPORTS');
  const proposal=proposeReferenceQueryAdapter(reports); // exact subjects, read route, Source, fingerprint cross-inspection
  matchWorldApplications(world,reports);
  if(proposal.commonInvocation===null)
    return {schemaVersion:'tei.research.query-signature-evidence/v0',
      status:'not_qualified_different_invocations',querySignatureDeclared:false,
      executionPermitted:false,accepted:false,authority:'derived_observation_only'};
  const shared=proposal.commonInvocation;
  const db=exactlyOne(world.databases,x=>x?.id===shared.databaseTarget,'DATABASE_NOT_UNIQUE');
  if(!shape(db,['id','semanticUses','kind','ownership','contracts'])||
     !Array.isArray(db.contracts))fail('UNMODELED_DATABASE_CONSTRAINT');
  const contract=exactlyOne(db.contracts,x=>x?.id===shared.contract,'CONTRACT_NOT_UNIQUE');
  if(!shape(contract,['id','version','reads','writes','queries'])||
     !shape(contract.version,['major','minor'])||
     !Number.isSafeInteger(contract.version.major)||contract.version.major<1||
     !Number.isSafeInteger(contract.version.minor)||contract.version.minor<0||
     !Array.isArray(contract.queries))fail('UNMODELED_CONTRACT');
  if(contract.version.major!==shared.compatibleMajor)fail('INCOMPATIBLE_CONTRACT_MAJOR');
  const query=exactlyOne(contract.queries,x=>x?.name===shared.query,'QUERY_NOT_UNIQUE');
  if(!shape(query,['name','inputs','result','resultCardinality'])||
     !Array.isArray(query.inputs)||!nonempty(query.result)||
     !cardinality.has(query.resultCardinality)||query.inputs.some(x=>!inputModel(x)))
    fail('INCOMPLETE_QUERY_SIGNATURE');
  const expectedInputs=shared.bindings;
  if(query.inputs.length!==expectedInputs.length||
     new Set(query.inputs.map(x=>x.name)).size!==query.inputs.length)
    fail('QUERY_INPUT_BINDINGS_INCOMPLETE');
  const typedBindings=expectedInputs.map(binding=>{
    const match=exactlyOne(query.inputs,x=>x.name===binding.input,'QUERY_INPUT_BINDINGS_INCOMPLETE');
    const type=findAttributeType(world,binding.targetAttribute);
    if(type!==match.type)fail('QUERY_INPUT_TYPE_MISMATCH');
    return {input:binding.input,type:match.type,targetAttribute:binding.targetAttribute};
  });
  checkResultData(world,query.result);
  return {
    schemaVersion:'tei.research.query-signature-evidence/v0',
    status:'declared_signature_matched',
    querySignatureDeclared:true,
    // This is only a declaration-level type check, not provider output conformance.
    contract:{id:contract.id,version:{...contract.version},query:query.name},
    inputTypes:typedBindings,
    output:{semanticData:query.result,cardinality:query.resultCardinality},
    applicationContext:proposal.applicationContext.map(x=>({...x})),
    executionPermitted:false,implementationSelected:false,
    accepted:false,acceptance:'not_asserted',
    verified:{typedDeclaration:true,providerImplementation:false,
      resultRuntimeShape:false,authorization:false,sourceAuthenticity:false,
      atomicWorkspaceSnapshot:false},
    authority:'derived_observation_only'
  };
}
