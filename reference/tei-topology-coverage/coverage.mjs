/** Read-only bounded-view experiment, not a TEI Schema, Source proof or Authority. */
const exact = s => typeof s === 'string' && s.length > 0 && s.trim() === s && !/[\r\n\t]/.test(s);
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

// Consistency of the *presented* Graph/Table only. Does not inspect actual Source.
function projectionIntegrity(view) {
  if (!view?.graph || !view?.table ||
      !Array.isArray(view.graph.nodes) || !Array.isArray(view.graph.edges) ||
      !Array.isArray(view.table.subjects) || !Array.isArray(view.table.relations))
    return 'invalid_projection';
  const subjects = view.table.subjects, relations = view.table.relations;
  const keys = new Set();
  for (const s of subjects) {
    try {
      if (tuple(s) !== s.key || s.viewId !== `row:${s.key}`)
        return 'subject_identity_mismatch';
    } catch { return 'subject_identity_mismatch'; }
    if (keys.has(s.key)) return 'subject_coverage_loss';
    keys.add(s.key);
  }
  const subjectNodes = view.graph.nodes.filter(n => n.key !== undefined);
  if (subjectNodes.length !== keys.size || subjectNodes.some(n =>
      !keys.has(n.key) || n.viewId !== `s:${n.key}` || n.relationId !== undefined))
    return 'subject_coverage_loss';
  const relationNodes = view.graph.nodes.filter(n => n.relationId !== undefined);
  if (view.graph.nodes.length !== subjectNodes.length + relationNodes.length)
    return 'view_coverage_loss';
  if (relationNodes.length !== relations.length) return 'relation_coverage_loss';
  const ids = new Set(), expected = new Map();
  for (const r of relations) {
    if (!exact(r?.id) || !exact(r?.type) || ids.has(r.id) ||
        !Array.isArray(r.roles) || r.roles.length < 2)
      return 'relation_identity_mismatch';
    ids.add(r.id);
    if (!relationNodes.some(n => n.relationId === r.id &&
        n.viewId === `r:${r.id}` && n.key === undefined))
      return 'relation_coverage_loss';
    for (const [index,x] of r.roles.entries()) {
      if (!exact(x?.role) || x.index !== index || !keys.has(x.key))
        return 'role_coverage_loss';
      const edgeId = JSON.stringify([r.id,index,x.role]);
      expected.set(edgeId,{
        from:`r:${r.id}`,to:`s:${x.key}`,role:x.role,relationId:r.id
      });
    }
  }
  if (view.graph.edges.length !== expected.size)
    return 'relation_coverage_loss';
  for (const e of view.graph.edges) {
    const x = expected.get(e.viewId);
    if (!x || e.from !== x.from || e.to !== x.to ||
        e.role !== x.role || e.relationId !== x.relationId)
      return 'role_coverage_loss';
    expected.delete(e.viewId);
  }
  return expected.size === 0 ? null : 'role_coverage_loss';
}

// View occurrences are local UI identifiers, never qualified Source or Action authority.
export function selectView(view, viewId, observedSnapshot) {
  if (!view?.graph || !view?.table) return {status:'unresolved'};
  if (view.revision !== revision(observedSnapshot) ||
      view.graph.revision !== view.revision || view.table.revision !== view.revision)
    return {status:'stale'};
  if (projectionIntegrity(view)) return {status:'unresolved'};
  const occurrences = [...view.graph.nodes,...view.table.subjects].filter(x=>x.viewId === viewId && x.key);
  if (occurrences.length !== 1) return {status:'unresolved'};
  return {status:'candidate',qualifiedKey:JSON.parse(occurrences[0].key)};
}

// Verifies *fixture-declared* closure, never the authenticity of its witness.
export function boundedAbsence(view, query, witness) {
  const unknown = reason => ({status:'unknown',reason});
  if (!view || !query || !witness) return unknown('missing_input');
  let rev; try { rev=revision(witness.snapshot); } catch { return unknown('invalid_witness'); }
  if (!view.graph || !view.table) return unknown('invalid_projection');
  if (view.revision!==rev || view.graph.revision!==rev || view.table.revision!==rev) return unknown('mixed_revision');
  if (!exact(query.scope) || !['subject','relation'].includes(query.kind) || !exact(query.type) ||
      (query.id !== undefined && !exact(query.id))) return unknown('invalid_query');
  // A known row is not necessarily disclosable to this Consumer.
  if (witness.disclosureQualified !== true) return unknown('disclosure_unqualified');
  const mismatch = projectionIntegrity(view);
  if (mismatch) return unknown(mismatch);
  const subjects=view.table.subjects, relations=view.table.relations;
  if (query.kind==='subject' && subjects.some(s=>s.scope===query.scope && s.kind===query.type && (query.id===undefined || s.id===query.id))) return {status:'present-in-snapshot'};
  if (query.kind==='relation' && relations.some(r=>r.type===query.type &&
      (query.id===undefined || r.id===query.id) &&
      r.roles.some(x=>JSON.parse(x.key)[0]===query.scope))) return {status:'present-in-snapshot'};
  if (!Array.isArray(witness.scopes) || !witness.scopes.includes(query.scope)) return unknown('scope_unbounded');
  if (!witness.subjectsComplete || !witness.relationsComplete || !witness.rolesComplete || !witness.compositionComplete || !witness.readerPinned || !witness.disclosureQualified || witness.omitted!==0 || witness.opaque!==0 || !exact(witness.evidenceId)) return unknown('closure_unqualified');
  return {status:'absent-in-declared-snapshot',evidenceId:witness.evidenceId};
}
