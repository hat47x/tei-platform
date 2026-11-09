// Fixed-reference, metadata-only evidence summary. No Source, IDs or free-form diagnostics in output.
// Consistency is validated here rather than asserted by an arbitrary caller.
export const REFERENCE_SUBJECTS = Object.freeze([
  {kind:'application',id:'order_entry'},
  {kind:'application-pattern',id:'order_entry.order_lines'},
  {kind:'semantic-relation',id:'sales.order.lines'},
  {kind:'semantic-data',id:'sales.order'},
  {kind:'application-pattern',id:'order_entry.customer_reference'}
].map(Object.freeze));

export class EvidenceSummaryError extends Error {
  constructor(code='INVALID_INSPECTION_EVIDENCE') {super(code);this.code=code;}
}
const valid=x=>typeof x==='string'&&x.length>0;
const object=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const COMPLETENESS=new Set(['structurally_projected','partial']);
const SOURCE_STATUSES=new Set(['indexed','not-indexed']);
const LINK_STATUSES={
  applicationPattern:new Set(['declared_membership','unresolved_membership','unknown','stale_or_different_snapshot']),
  patternRelation:new Set(['declared_reference_match','unresolved_reference','no_declared_relation','unknown','stale_or_different_snapshot'])
};
const fail=()=>{throw new EvidenceSummaryError();};

export function summarizeInspectionEvidence(batch) {
  if(!object(batch)||batch.schemaVersion!=='tei.research.inspection-batch/v0'||
     batch.effect!=='read-only'||batch.authority!=='derived_observation_only'||
     batch.acceptance!=='not_asserted'||!valid(batch.worldFingerprint)||
     !valid(batch.sourceIndexFingerprint)||!Array.isArray(batch.observations)||
     batch.observations.length!==REFERENCE_SUBJECTS.length) fail();

  const subjects=batch.observations.map((o,index)=>{
    const selector=REFERENCE_SUBJECTS[index];
    if(!object(o)||!object(o.subject)||o.subject.kind!==selector.kind||
       o.subject.id!==selector.id||o.schemaVersion!=='tei.research.inspection-features/v0'||
       o.effect!=='read-only'||o.authority!=='derived_observation_only'||
       o.acceptance!=='not_asserted'||o.contractConformance!=='not_verified'||
       !COMPLETENESS.has(o.completeness)||!Array.isArray(o.diagnostics)||
       o.diagnostics.some(x=>typeof x!=='string')||!object(o.provenance)||
       o.provenance.worldFingerprint!==batch.worldFingerprint||
       o.provenance.sourceIndexFingerprint!==batch.sourceIndexFingerprint||
       !SOURCE_STATUSES.has(o.provenance.sourceStatus)) fail();
    return {index,kind:selector.kind,completeness:o.completeness,
      diagnosticCount:o.diagnostics.length,sourceStatus:o.provenance.sourceStatus};
  });
  const links={};
  for(const key of ['applicationPattern','patternRelation']) {
    const link=batch.links?.[key];
    if(!object(link)||link.accepted!==false||!LINK_STATUSES[key].has(link.status)) fail();
    // Whitelisted statuses only. Other link fields are never forwarded.
    links[key]={status:link.status,accepted:false};
  }
  return {schemaVersion:'tei.research.inspection-evidence/v0',effect:'read-only',
    acceptance:'not_asserted',subjectCount:subjects.length,subjects,
    consistentSnapshot:true,links};
}
