// #1016 E7a: minimize paired Source Index disclosure without adding an authority or resolver.
// The caller must already have a trusted same-invocation E5 observation.
const object = x => x !== null && typeof x === 'object' && !Array.isArray(x);
const subjectKey = x => JSON.stringify([x.kind, x.id]);
const validSubject = x => object(x) && typeof x.kind === 'string' && x.kind.trim() !== '' &&
  typeof x.id === 'string' && x.id.trim() !== '';
const fingerprint = x => typeof x === 'string' && /^sha256:[a-f0-9]{64}$/.test(x);
const path = x => typeof x === 'string' && x.length > 0 && !x.startsWith('/') &&
  !x.includes('\\') && !x.split('/').some(p => !p || p === '.' || p === '..' || p.includes(':'));
const roles = new Set(['definition', 'authoring', 'implementation', 'derived']);
const format = 'tei.e5-paired-source-observation.v1';
const qualification = 'read-only-paired-input-not-an-authority';

// This projection is an *optional local export*, not a replacement for the
// unredacted trusted Source Index. Unknown selected role contributions are
// retained; unrelated workspace paths are not emitted.
export function minimizePairedSourceEvidence(envelope) {
  if (!object(envelope) || envelope.format !== format ||
      envelope.qualification !== qualification || !object(envelope.report) ||
      envelope.report.status !== 'observed' ||
      envelope.report.qualification !== 'declaration-and-source-index-only' ||
      typeof envelope.report.application !== 'string' || !envelope.report.application.trim() ||
      !Array.isArray(envelope.report.issues) || envelope.report.issues.length !== 0 ||
      !Array.isArray(envelope.report.dependencies) || !envelope.report.dependencies.length ||
      !object(envelope.index) || !Array.isArray(envelope.index.artifacts) ||
      !Array.isArray(envelope.index.references)) {
    return {status:'unresolved', reason:'incomplete_or_unqualified_input'};
  }
  const selected = new Map();
  for (const dependency of envelope.report.dependencies) {
    if (!object(dependency) || typeof dependency.relation !== 'string' ||
        !dependency.relation.trim() || !validSubject(dependency.subject) ||
        !object(dependency.source) || !path(dependency.source.path) ||
        !fingerprint(dependency.source.fingerprint)) {
      return {status:'unresolved', reason:'invalid_dependency'};
    }
    const key = subjectKey(dependency.subject);
    if (selected.has(key)) return {status:'unresolved', reason:'duplicate_dependency'};
    selected.set(key, dependency);
  }
  const selectedRefs = [];
  const paths = new Set();
  const definitions = new Map();
  for (const reference of envelope.index.references) {
    if (!object(reference) || !validSubject(reference.subject)) {
      return {status:'unresolved', reason:'malformed_index_reference'};
    }
    const key = subjectKey(reference.subject);
    if (!selected.has(key)) continue;
    if (!roles.has(reference.role) || !path(reference.artifactPath)) {
      return {status:'unresolved', reason:'invalid_selected_reference'};
    }
    selectedRefs.push(reference);
    paths.add(reference.artifactPath);
    if (reference.role === 'definition') {
      const prior = definitions.get(key) || [];
      prior.push(reference.artifactPath);
      definitions.set(key, prior);
    }
  }
  for (const [key, dependency] of selected) {
    const found = definitions.get(key) || [];
    if (found.length !== 1 || found[0] !== dependency.source.path) {
      return {status:'unresolved', reason:'definition_mismatch'};
    }
  }
  const artifacts = new Map();
  for (const artifact of envelope.index.artifacts) {
    if (!object(artifact) || !path(artifact.path) || !fingerprint(artifact.fingerprint)) {
      return {status:'unresolved', reason:'malformed_index_artifact'};
    }
    if (!paths.has(artifact.path)) continue;
    if (artifacts.has(artifact.path)) return {status:'unresolved', reason:'ambiguous_artifact'};
    artifacts.set(artifact.path, artifact);
  }
  if (artifacts.size !== paths.size) {
    return {status:'unresolved', reason:'missing_selected_artifact'};
  }
  for (const dependency of selected.values()) {
    if (artifacts.get(dependency.source.path)?.fingerprint !== dependency.source.fingerprint) {
      return {status:'unresolved', reason:'fingerprint_mismatch'};
    }
  }
  // Reconstruct an allowlisted shape. Unknown JSON fields must not become
  // accidental data disclosure when an external tool consumes the projection.
  const projectedRefs = selectedRefs.map(ref => ({
    subject:{kind:ref.subject.kind,id:ref.subject.id},
    artifactPath:ref.artifactPath, role:ref.role,
  }));
  const projectedDependencies = envelope.report.dependencies.map(dep => ({
    relation:dep.relation, subject:{kind:dep.subject.kind,id:dep.subject.id},
    source:{path:dep.source.path,fingerprint:dep.source.fingerprint},
  }));
  const snapshot = {
    format, qualification, report:{
      application:envelope.report.application, status:'observed',
      qualification:envelope.report.qualification, issues:[],
      dependencies:projectedDependencies,
    },
    index:{artifacts:[...artifacts.values()].map(a=>({
      path:a.path, fingerprint:a.fingerprint,
    })), references:projectedRefs},
  };
  return {status:'projected', qualification:'selected-source-disclosure-only',
    fullClosure:false, authority:'none', evidence:snapshot,
    accounting:{selectedSubjects:selected.size, sourceArtifacts:artifacts.size,
      sourceReferences:selectedRefs.length}};
}
