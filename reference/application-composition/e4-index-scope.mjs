// #1016 E4: consumer-selected subjects joined to existing Source Index metadata.
// Read-only diagnostic. It neither derives semantic dependencies nor grants authority.
const isObject = x => x !== null && typeof x === 'object' && !Array.isArray(x);
const safePath = p => typeof p === 'string' && p.length > 0 && !p.startsWith('/') &&
  !p.includes('\\') && !p.split('/').some(segment => !segment || segment === '.' || segment === '..' || segment.includes(':'));
const fingerprint = x => typeof x === 'string' && /^sha256:[a-f0-9]{64}$/.test(x);
const validSubject = x => isObject(x) && typeof x.kind === 'string' && x.kind.trim() === x.kind && !!x.kind &&
  typeof x.id === 'string' && x.id.trim() === x.id && !!x.id;
const validRole = role => ['definition','authoring','implementation','derived'].includes(role);
const subjectKey = x => JSON.stringify([x.kind,x.id]);

// Inputs must be snapshots produced by the trusted Source Index boundary.
// Shape validation here is NOT a provenance check or semantic parser.
function indexedSources(index) {
  if (!isObject(index) || !Array.isArray(index.artifacts) || !Array.isArray(index.references)) {
    throw new TypeError('invalid_source_index');
  }
  const artifacts = new Map();
  for (const artifact of index.artifacts) {
    if (!isObject(artifact) || !safePath(artifact.path) || !fingerprint(artifact.fingerprint) ||
        artifacts.has(artifact.path)) throw new TypeError('invalid_index_artifact');
    artifacts.set(artifact.path, artifact.fingerprint);
  }
  const subjects = new Map();
  for (const reference of index.references) {
    if (!isObject(reference) || !validSubject(reference.subject) ||
        !safePath(reference.artifactPath) || !artifacts.has(reference.artifactPath) ||
        !validRole(reference.role)) throw new TypeError('invalid_index_reference');
    const key = subjectKey(reference.subject);
    if (!subjects.has(key)) subjects.set(key, new Map());
    const refKey = JSON.stringify([reference.role,reference.artifactPath]);
    subjects.get(key).set(refKey, artifacts.get(reference.artifactPath));
  }
  return subjects;
}

// Caller-supplied subject selections are NOT verified dependency edges.
// An absent semantic edge can yield 'unchanged' for an incomplete consumer.
export function assessIndexedSubjects({baselineIndex,currentIndex,consumers} = {}) {
  const before = indexedSources(baselineIndex);
  const after = indexedSources(currentIndex);
  if (!Array.isArray(consumers)) throw new TypeError('invalid_consumers');
  const seen = new Set();
  return consumers.map(consumer => {
    if (!isObject(consumer) || typeof consumer.id !== 'string' || !/^[A-Za-z][\w.-]*$/.test(consumer.id) ||
        seen.has(consumer.id) || !Array.isArray(consumer.subjects) || !consumer.subjects.length ||
        consumer.subjects.some(s => !validSubject(s)) ||
        new Set(consumer.subjects.map(subjectKey)).size !== consumer.subjects.length) {
      throw new TypeError('invalid_consumer_subjects');
    }
    seen.add(consumer.id);
    const affected = [];
    for (const subject of consumer.subjects) {
      const key = subjectKey(subject), oldRefs = before.get(key), newRefs = after.get(key);
      if (!oldRefs || !newRefs) {
        affected.push({subject,reason:'missing_indexed_subject'});
        continue;
      }
      if (oldRefs.size !== newRefs.size || [...oldRefs.keys()].some(k => !newRefs.has(k))) {
        affected.push({subject,reason:'reference_set_changed'});
        continue;
      }
      if ([...oldRefs].some(([k,value]) => newRefs.get(k) !== value)) {
        affected.push({subject,reason:'source_fingerprint_changed'});
      }
    }
    return {id:consumer.id,status:affected.length ? 'unresolved' : 'unchanged',
      affected,qualification:'indexed-sources-only',dependencyProvenance:'caller-supplied-subject-selection'};
  });
}
