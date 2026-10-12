// Research-only adapter for the existing operations.InspectSubject JSON contract.
// Input authenticity, workspace access, and acceptance are NOT established here.
// No TEI DSL parser, Canonical authority, or new contract resolver is introduced.
export class InspectionProjectionError extends Error {
  constructor(code) { super(code); this.code=code; }
}
const nonEmpty = s => typeof s==='string' && s.length>0;
const object = x => x!==null && typeof x==='object' && !Array.isArray(x);
const knownPatternKinds=new Set(['browse','record','reference','command','header-detail']);
const keep=(o,fields)=>Object.fromEntries(fields.filter(k=>Object.hasOwn(o,k)).map(k=>[k,o[k]]));
const unknown=(o,allowed)=>Object.keys(o).filter(k=>!allowed.includes(k));
function requireFields(o,keys) {
  if (!object(o) || keys.some(k=>!nonEmpty(o[k]))) throw new InspectionProjectionError('malformed_canonical');
}
function copyArray(a,fields,required,diags,label) {
  if (a===undefined) return [];
  if (!Array.isArray(a)) throw new InspectionProjectionError('malformed_canonical');
  return a.map((v,i)=>{
    if (!object(v)) throw new InspectionProjectionError('malformed_canonical');
    for(const k of required) if(k==='attributes'||k==='determinants'||k==='dependents'||k==='keys') {
      if(!Array.isArray(v[k]))throw new InspectionProjectionError('malformed_canonical');
    } else if(!nonEmpty(v[k])) throw new InspectionProjectionError('malformed_canonical');
    for(const k of unknown(v,fields)) diags.push(label+'['+i+'].'+k);
    return keep(v,fields);
  });
}

function projectCanonical(kind,c,diagnostics) {
  let allowed, model;
  switch(kind) {
    case 'application': {
      requireFields(c,['id']);
      // Application membership is observed, not a declaration of Pattern semantics.
      allowed=['id','semanticUses','validations','databases','patterns'];
      const patterns=copyArray(c.patterns,
        ['id','kind','data','relation','query','capability','datasource','contractQuery'],
        ['id','kind'],diagnostics,'patterns');
      if(new Set(patterns.map(p=>p.id)).size!==patterns.length) diagnostics.push('duplicate_pattern_identity');
      for(let i=0;i<patterns.length;i++) if(!knownPatternKinds.has(patterns[i].kind))
        diagnostics.push('patterns['+i+'].unknown_kind');
      if(c.databases!==undefined && !Array.isArray(c.databases))
        throw new InspectionProjectionError('malformed_canonical');
      if(c.semanticUses!==undefined && (!Array.isArray(c.semanticUses)||c.semanticUses.some(x=>!nonEmpty(x))))
        throw new InspectionProjectionError('malformed_canonical');
      if(c.validations!==undefined && (!Array.isArray(c.validations)||c.validations.some(x=>!nonEmpty(x))))
        throw new InspectionProjectionError('malformed_canonical');
      model={
        declaredPatternIds:patterns.map(p=>p.id),
        semanticUses:c.semanticUses||[],
        validations:c.validations||[],
        databaseAliases:(c.databases||[]).map((d,i)=>{
          if(!object(d)||!nonEmpty(d.alias))throw new InspectionProjectionError('malformed_canonical');
          for(const k of unknown(d,['alias','target','access','requires']))diagnostics.push('databases['+i+'].'+k);
          return d.alias;
        }),
        coverage:'membership_only'
      };
      break;
    }
    case 'application-pattern': {
      requireFields(c,['id','kind']);
      allowed=['id','kind','data','relation','query','capability','datasource','contractQuery'];
      if(!knownPatternKinds.has(c.kind)) diagnostics.push('unknown_pattern_kind');
      model=keep(c,['kind','data','relation','query','capability','datasource']);
      if(Object.hasOwn(c,'contractQuery')) {
        const q=c.contractQuery;
        if(q!==null) {
          requireFields(q,['contract','query']);
          for(const k of unknown(q,['contract','query','inputs']))diagnostics.push('contractQuery.'+k);
          model.contractQuery={
            contract:q.contract,query:q.query,
            inputs:copyArray(q.inputs,['targetAttribute','input'],['targetAttribute','input'],diagnostics,'contractQuery.inputs')
          };
        } else model.contractQuery=null;
      }
      break;
    }
    case 'semantic-relation': {
      requireFields(c,['id','target','cardinality']);
      allowed=['id','symbol','target','cardinality','keys'];
      if(!['one','many'].includes(c.cardinality)) diagnostics.push('unknown_cardinality');
      model={target:c.target,cardinality:c.cardinality,
        keys:copyArray(c.keys,['sourceAttribute','targetAttribute'],['sourceAttribute','targetAttribute'],diagnostics,'keys')};
      break;
    }
    case 'semantic-data': {
      requireFields(c,['id']);
      allowed=['id','symbol','attributes','relations','candidateKeys','functionalDependencies'];
      model={
        attributes:copyArray(c.attributes,['id','symbol','type'],['id','type'],diagnostics,'attributes'),
        relations:copyArray(c.relations,['id','symbol','target','cardinality','keys'],['id','target','cardinality'],diagnostics,'relations'),
        candidateKeys:copyArray(c.candidateKeys,['id','symbol','attributes'],['id','attributes'],diagnostics,'candidateKeys'),
        functionalDependencies:copyArray(c.functionalDependencies,['id','symbol','determinants','dependents'],['id','determinants','dependents'],diagnostics,'functionalDependencies')
      };
      for(let i=0;i<model.relations.length;i++){
        const r=model.relations[i];
        if(!['one','many'].includes(r.cardinality)) diagnostics.push('relations['+i+'].unknown_cardinality');
        r.keys=copyArray(r.keys,['sourceAttribute','targetAttribute'],['sourceAttribute','targetAttribute'],diagnostics,'relations['+i+'].keys');
      }
      break;
    }
    default: throw new InspectionProjectionError('unsupported_subject_kind');
  }
  for(const k of unknown(c,allowed))diagnostics.push('canonical.'+k);
  return model;
}

/** Only a read-only *structural observation* from the already-normalized World.
 * The caller must authorize/disclose data before invoking InspectSubject.
 */
export function projectInspectionV0(input) {
  if (!object(input) || input.schemaVersion!=='tei.operations.inspect/v0' || input.effect!=='read-only')
    throw new InspectionProjectionError('unsupported_inspection_envelope');
  const s=input.subject;
  if (!object(s)||!nonEmpty(s.kind)||!nonEmpty(s.id)||!nonEmpty(input.worldFingerprint)||!nonEmpty(input.sourceIndexFingerprint))
    throw new InspectionProjectionError('incomplete_inspection_envelope');
  if (!['indexed','not-indexed'].includes(input.sourceStatus))throw new InspectionProjectionError('invalid_source_status');
  let c=input.canonical;
  if(typeof c==='string') {
    try{c=JSON.parse(c);}catch{throw new InspectionProjectionError('malformed_canonical');}
  }
  if (!object(c)||c.id!==s.id) throw new InspectionProjectionError('canonical_subject_mismatch');
  const src=input.sources;
  if (!object(src) || !object(src.subject) || src.subject.id!==s.id || src.subject.kind!==s.kind ||
      !Array.isArray(src.references) || !Array.isArray(src.artifacts))
    throw new InspectionProjectionError('invalid_source_trace');
  if(src.references.some(r=>!object(r) || !object(r.subject) || r.subject.id!==s.id || r.subject.kind!==s.kind))
    throw new InspectionProjectionError('mixed_subject_trace');
  if(src.artifacts.some(a=>!object(a)||!nonEmpty(a.path)||!nonEmpty(a.fingerprint)))
    throw new InspectionProjectionError('invalid_artifact_trace');
  if (input.sourceStatus==='indexed' && (src.references.length===0 || src.artifacts.length===0))
    throw new InspectionProjectionError('inconsistent_source_status');
  const diagnostics=[];
  if(input.sourceStatus==='not-indexed')diagnostics.push('source_not_indexed');
  const features=projectCanonical(s.kind,c,diagnostics);
  return {
    schemaVersion:'tei.research.inspection-features/v0',effect:'read-only',
    subject:{kind:s.kind,id:s.id},features,
    provenance:{
      worldFingerprint:input.worldFingerprint,
      sourceIndexFingerprint:input.sourceIndexFingerprint,
      sourceStatus:input.sourceStatus,
      sourceArtifacts:src.artifacts.map(a=>({path:a.path,fingerprint:a.fingerprint}))
    },
    completeness:diagnostics.length===0?'structurally_projected':'partial',
    diagnostics:diagnostics.sort(),
    acceptance:'not_asserted',contractConformance:'not_verified',
    authority:'derived_observation_only'
  };
}

/** A declared link across two compatible observation snapshots, NOT a proven business relation. */
export function linkPatternRelation(pattern,relation) {
  if(pattern?.subject?.kind!=='application-pattern' || relation?.subject?.kind!=='semantic-relation')
    throw new InspectionProjectionError('invalid_link_kinds');
  if(pattern.provenance?.worldFingerprint!==relation.provenance?.worldFingerprint ||
     pattern.provenance?.sourceIndexFingerprint!==relation.provenance?.sourceIndexFingerprint)
    return {status:'stale_or_different_snapshot',accepted:false};
  if(pattern.completeness!=='structurally_projected' || relation.completeness!=='structurally_projected')
    return {status:'unknown',accepted:false};
  if(!nonEmpty(pattern.features?.relation))
    return {status:'no_declared_relation',accepted:false};
  return {status:pattern.features.relation===relation.subject.id?'declared_reference_match':'unresolved_reference',accepted:false};
}

/** Check only explicit Application.Pattern membership in one observed snapshot.
 * This does NOT grant cross-application read rights or prove business meaning.
 */
export function linkApplicationPattern(application,pattern) {
  if(application?.subject?.kind!=='application'||pattern?.subject?.kind!=='application-pattern')
    throw new InspectionProjectionError('invalid_membership_kinds');
  if(application.provenance?.worldFingerprint!==pattern.provenance?.worldFingerprint ||
     application.provenance?.sourceIndexFingerprint!==pattern.provenance?.sourceIndexFingerprint)
    return {status:'stale_or_different_snapshot',accepted:false};
  if(application.completeness!=='structurally_projected'||pattern.completeness!=='structurally_projected')
    return {status:'unknown',accepted:false};
  const ids=application.features?.declaredPatternIds;
  if(!Array.isArray(ids))return {status:'unknown',accepted:false};
  return {status:ids.includes(pattern.subject.id)?'declared_membership':'unresolved_membership',accepted:false};
}
