// #1016 E3: read-only, caller-supplied source identities and consumers.
// This is a selective requalification experiment, not a second lock or a runtime resolver.
const isObject = value => value !== null && typeof value === 'object' && !Array.isArray(value);
const safePath = path => typeof path === 'string' && path.length > 0 && !path.startsWith('/') &&
  !path.includes('\\') && !path.split('/').some(p => !p || p === '.' || p === '..');
const digest = value => typeof value === 'string' && /^[a-f0-9]{40}$/.test(value);
const token = value => typeof value === 'string' && /^[A-Za-z][\w.-]*$/.test(value);

function indexSource(entries) {
  if (!Array.isArray(entries)) throw new TypeError('invalid_sources');
  const index = new Map();
  for (const entry of entries) {
    if (!isObject(entry) || !safePath(entry.path) || !digest(entry.gitBlobSha) ||
        index.has(entry.path)) throw new TypeError('invalid_or_duplicate_source');
    index.set(entry.path, entry.gitBlobSha);
  }
  return index;
}

// Both baseline and current identities must originate from trusted, explicit reads.
// "unchanged" only means observed dependencies retained identical blob contents.
// "unresolved" requires independent qualification; it never implies a successful upgrade.
export function assessScopedChanges({baseline, current, consumers} = {}) {
  const old = indexSource(baseline);
  const now = indexSource(current);
  if (!Array.isArray(consumers)) throw new TypeError('invalid_consumers');
  const seen = new Set();
  return consumers.map(consumer => {
    if (!isObject(consumer) || !token(consumer.id) || seen.has(consumer.id) ||
        !Array.isArray(consumer.dependencies) || consumer.dependencies.length === 0 ||
        new Set(consumer.dependencies).size !== consumer.dependencies.length ||
        consumer.dependencies.some(path => !safePath(path))) throw new TypeError('invalid_consumer');
    seen.add(consumer.id);
    const missing = consumer.dependencies.filter(path => !old.has(path) || !now.has(path));
    const changed = consumer.dependencies.filter(path => old.has(path) && now.has(path) && old.get(path) !== now.get(path));
    if (missing.length) return {id:consumer.id, status:'unresolved', reason:'missing_source_identity', missing, changed};
    if (changed.length) return {id:consumer.id, status:'unresolved', reason:'dependency_changed', changed};
    return {id:consumer.id, status:'unchanged', changed:[]};
  });
}

// A scoped change can be *locally applicable* only when all declared dependencies
// are unchanged AND the existing E2 candidate was a bounded, proposed edit.
// The return value does not attest to acceptance, provider conformance or permissions.
export function assessCandidateApplicability({candidate, scope} = {}) {
  if (!isObject(candidate) || !isObject(scope)) throw new TypeError('invalid_candidate_scope');
  if (scope.status === 'unresolved') return {status:'unresolved', reason:scope.reason};
  if (scope.status !== 'unchanged') return {status:'unresolved', reason:'unknown_scope_state'};
  if (candidate.status === 'proposed') return {status:'applicable', qualification:'local-edit-only'};
  if (candidate.status === 'unchanged') return {status:'unchanged'};
  return {status:'unresolved', reason:candidate.reason || 'additional_qualification_required'};
}
