import {assessIndexedSubjects} from './e4-index-scope.mjs';
// E8 provisional declaration-only probe; NOT semantic acceptance or full closure.
export function inspectCapabilityInputTypes(world, index, applicationId) {
  const issues = [], dependencies = [];
  // Inspect parsed declarations only. Structural validity of a Source Index
  // remains owned by E4; never silently accept malformed evidence.
  if (!world || !Array.isArray(world.applications) ||
      !Array.isArray(world.semantics) ||
      !index || !Array.isArray(index.artifacts) ||
      !Array.isArray(index.references) ||
      world.applications.some(a => !a || typeof a !== 'object' ||
        (a.patterns != null && !Array.isArray(a.patterns)) ||
        (Array.isArray(a.patterns) && a.patterns.some(p => !p || typeof p !== 'object'))) ||
      world.semantics.some(ns => !ns || typeof ns !== 'object' ||
        (ns.capabilities != null && !Array.isArray(ns.capabilities)) ||
        (ns.types != null && !Array.isArray(ns.types)) ||
        (Array.isArray(ns.capabilities) && ns.capabilities.some(c =>
          !c || typeof c !== 'object' || (c.inputs != null && !Array.isArray(c.inputs)) ||
          (Array.isArray(c.inputs) && c.inputs.some(i => !i || typeof i !== 'object')))) ||
        (Array.isArray(ns.types) && ns.types.some(t => !t || typeof t !== 'object'))) ||
      index.artifacts.some(a => !a || typeof a !== 'object') ||
      index.references.some(ref => !ref || typeof ref !== 'object')) {
    return {status:"unresolved",qualification:"declaration-only-not-semantic-acceptance",
      dependencies,issues:["invalid_snapshot_shape"]};
  }
  const apps = (world.applications ?? []).filter(a => a.id === applicationId);
  if (apps.length !== 1) return {status:"unresolved", qualification:"declaration-only", dependencies, issues:["application_missing_or_ambiguous"]};
  const definitions = (kind,id) => (index.references ?? []).filter(r => r.subject?.kind === kind && r.subject?.id === id && r.role === "definition");
  const evidence = (kind,id) => {
    const refs = definitions(kind,id);
    if (refs.length !== 1) {issues.push("source_definition_missing_or_ambiguous:"+kind+":"+id); return;}
    const artifacts = (index.artifacts ?? []).filter(a => a.path === refs[0].artifactPath && a.fingerprint);
    if (artifacts.length !== 1) {issues.push("source_artifact_missing_or_ambiguous:"+kind+":"+id); return;}
    dependencies.push({subject:{kind,id},path:artifacts[0].path,fingerprint:artifacts[0].fingerprint});
  };
  const capabilities = (world.semantics ?? []).flatMap(ns => ns.capabilities ?? []);
  const types = (world.semantics ?? []).flatMap(ns => ns.types ?? []);
  const seen = new Set();
  let inspected = 0;
  for (const pattern of apps[0].patterns ?? []) {
    if (!pattern.capability) continue;
    const matches = capabilities.filter(c => c.id === pattern.capability);
    if (matches.length !== 1) {issues.push("capability_missing_or_ambiguous:"+pattern.capability);continue;}
    inspected++;
    for (const input of matches[0].inputs ?? []) {
      const id = input.type;
      if (!id) {issues.push("input_type_missing:"+matches[0].id);continue;}
      if (types.filter(t => t.id === id).length !== 1) {issues.push("type_missing_or_ambiguous:"+id);continue;}
      if (!seen.has(id)) {seen.add(id); evidence("semantic-type",id);}
    }
  }
  // E4 checks all indexed Artifact paths, SHA-256s, roles and references.
  // A same-index self-compare only checks structural evidence. It does NOT
  // verify original bytes, provenance, cross-snapshot drift, or authority.
  if (dependencies.length) {
    try {
      const result = assessIndexedSubjects({
        baselineIndex:index,currentIndex:index,
        consumers:[{id:'capability-input-types',
          subjects:dependencies.map(d=>d.subject)}],
      })[0];
      if (result.status !== 'unchanged') issues.push('unresolved_source_index_subject');
    } catch {
      issues.push('invalid_source_index');
    }
  }
  dependencies.sort((a,b)=>a.subject.id.localeCompare(b.subject.id));
  return {status:issues.length?"unresolved":dependencies.length?"observed-limited":inspected?"no-input-types":"no-capability-references",qualification:"declaration-only-not-semantic-acceptance",dependencies,issues};
}
