// E8 provisional declaration-only probe; NOT semantic acceptance or full closure.
export function inspectCapabilityInputTypes(world, index, applicationId) {
  const issues = [], dependencies = [];
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
  for (const pattern of apps[0].patterns ?? []) {
    if (!pattern.capability) continue;
    const matches = capabilities.filter(c => c.id === pattern.capability);
    if (matches.length !== 1) {issues.push("capability_missing_or_ambiguous:"+pattern.capability);continue;}
    for (const input of matches[0].inputs ?? []) {
      const id = input.type;
      if (types.filter(t => t.id === id).length !== 1) {issues.push("type_missing_or_ambiguous:"+id);continue;}
      if (!seen.has(id)) {seen.add(id); evidence("semantic-type",id);}
    }
  }
  dependencies.sort((a,b)=>a.subject.id.localeCompare(b.subject.id));
  return {status:issues.length?"unresolved":"observed-limited",qualification:"declaration-only-not-semantic-acceptance",dependencies,issues};
}
