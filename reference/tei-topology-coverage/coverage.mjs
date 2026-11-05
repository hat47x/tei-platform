/** Read-only bounded-view experiment, not a TEI Schema, Source proof or Authority. */
const exact = s => typeof s === 'string' && s.length > 0 && s.trim() === s;
const need = (s, name) => { if (!exact(s)) throw Error(`invalid_${name}`); return s; };
const tuple = o => JSON.stringify([need(o?.scope,'scope'),need(o?.kind,'kind'),need(o?.id,'id')]);
const revision = s => JSON.stringify([need(s?.workspace,'workspace'),need(s?.modelRevision,'modelRevision'),need(s?.sourceIndexFingerprint,'sourceIndexFingerprint'),need(s?.projectionRevision,'projectionRevision')]);

// Always generate both views from the same immutable observation; no implied write path.
export function project(input) {
  if (!Array.isArray(input?.subjects) || !Array.isArray(input?.relations)) throw Error('invalid_projection');
  const rev = revision(input.snapshot), seen = new Set(), relationIDs = new Set();
  const subjects = input.subjects.map(s => {
    const key = tuple(s);
    if (seen.has(key)) throw Error('duplicate_subject');
    seen.add(key);
    return {key, scope:s.scope, kind:s.kind, id:s.id};
  });
  const relations = input.relations.map(r => {
    need(r?.id,'relation_id'); need(r?.type,'relation_type');
    if (relationIDs.has(r.id)) throw Error('duplicate_relation');
    relationIDs.add(r.id);
    if (!Array.isArray(r.roles) || r.roles.length < 2) throw Error('invalid_roles');
    const roles = r.roles.map((x,i) => {
      need(x?.role,'role'); const key = tuple(x.subject);
      if (!seen.has(key)) throw Error('missing_endpoint');
      return {role:x.role, index:i, key};
    });
    return {id:r.id,type:r.type,roles};
  });
  return {
    revision:rev,
    graph:{revision:rev,
      nodes:[...subjects.map(s=>({viewId:`s:${s.key}`,key:s.key})),...relations.map(r=>({viewId:`r:${r.id}`,relationId:r.id}))],
      edges:relations.flatMap(r=>r.roles.map(x=>({viewId:JSON.stringify([r.id,x.index,x.role]),from:`r:${r.id}`,to:`s:${x.key}`,relationId:r.id,role:x.role})))},
    table:{revision:rev,subjects:subjects.map(s=>({...s,viewId:`row:${s.key}`})),relations:relations.map(r=>({...r,roles:r.roles.map(x=>({...x}))}))},
  };
}

// View occurrences are local UI identifiers, never qualified Source or Action authority.
export function selectView(view, viewId, observedSnapshot) {
  if (view.revision !== revision(observedSnapshot)) return {status:'stale'};
  const occurrences = [...view.graph.nodes,...view.table.subjects].filter(x=>x.viewId === viewId && x.key);
  if (occurrences.length !== 1) return {status:'unresolved'};
  return {status:'candidate',qualifiedKey:JSON.parse(occurrences[0].key)};
}

// Verifies *fixture-declared* closure, never the authenticity of its witness.
export function boundedAbsence(view, query, witness) {
  const unknown = reason => ({status:'unknown',reason});
  if (!view || !query || !witness) return unknown('missing_input');
  let rev; try { rev=revision(witness.snapshot); } catch { return unknown('invalid_witness'); }
  if (view.revision!==rev || view.graph.revision!==rev || view.table.revision!==rev) return unknown('mixed_revision');
  if (!exact(query.scope) || !['subject','relation'].includes(query.kind) || !exact(query.type) ||
      (query.id !== undefined && !exact(query.id))) return unknown('invalid_query');
  // A known row is not necessarily disclosable to this Consumer.
  if (witness.disclosureQualified !== true) return unknown('disclosure_unqualified');
  const subjects=view.table.subjects, relations=view.table.relations;
  if (query.kind==='subject' && subjects.some(s=>s.scope===query.scope && s.kind===query.type && (query.id===undefined || s.id===query.id))) return {status:'present-in-snapshot'};
  if (query.kind==='relation' && relations.some(r=>r.type===query.type && r.roles.some(x=>JSON.parse(x.key)[0]===query.scope))) return {status:'present-in-snapshot'};
  if (!Array.isArray(witness.scopes) || !witness.scopes.includes(query.scope)) return unknown('scope_unbounded');
  if (!witness.subjectsComplete || !witness.relationsComplete || !witness.rolesComplete || !witness.compositionComplete || !witness.readerPinned || !witness.disclosureQualified || witness.omitted!==0 || witness.opaque!==0 || !exact(witness.evidenceId)) return unknown('closure_unqualified');
  const subjectKeys=new Set(subjects.map(s=>s.key));
  const graphKeys=view.graph.nodes.filter(n=>n.key).map(n=>n.key);
  if (subjects.length!==subjectKeys.size || graphKeys.length!==subjectKeys.size || graphKeys.some(k=>!subjectKeys.has(k))) return unknown('subject_coverage_loss');
  const graphRelations=view.graph.nodes.filter(n=>n.relationId).map(n=>n.relationId);
  if (graphRelations.length!==relations.length || relations.some(r=>!graphRelations.includes(r.id))) return unknown('relation_coverage_loss');
  const expected=[];
  for(const r of relations) for(const role of r.roles){
    if(!subjectKeys.has(role.key)) return unknown('missing_role_endpoint');
    const edgeId=JSON.stringify([r.id,role.index,role.role]);
    expected.push(edgeId);
    if(!view.graph.edges.some(e=>e.viewId===edgeId && e.from===`r:${r.id}` && e.to===`s:${role.key}` && e.role===role.role)) return unknown('role_coverage_loss');
  }
  if(view.graph.edges.length!==expected.length || view.graph.edges.some(e=>!expected.includes(e.viewId))) return unknown('relation_coverage_loss');
  return {status:'absent-in-declared-snapshot',evidenceId:witness.evidenceId};
}
