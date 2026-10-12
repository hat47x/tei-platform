// Research-only deterministic pairwise reuse triage; NOT semantic authority.
const KINDS = new Set(['ui', 'logic']);
const STATES = { reuse: 'reuse_candidate', pattern: 'pattern_candidate', review: 'needs_semantic_review', incompatible: 'incompatible' };

export class CandidateInputError extends Error {
  constructor(code) { super(code); this.code = code; }
}
const valid = (s) => typeof s === 'string' && s.trim().length > 0;
const stable = (a) => JSON.stringify(a);
const normalizedSet = (a) => [...new Set(a)].sort();

function checkRecord(r) {
  if (!r || !valid(r.id) || !valid(r.application) || !KINDS.has(r.kind)) throw new CandidateInputError('invalid_record');
  if (!r.source || !valid(r.source.locator) || !valid(r.source.digest) || !valid(r.source.revision)) throw new CandidateInputError('missing_source_provenance');
  if (!r.signature || !Array.isArray(r.signature.inputs) || !Array.isArray(r.signature.outputs) || !Array.isArray(r.signature.effects) || !Array.isArray(r.signature.failures)) throw new CandidateInputError('invalid_signature');
  for (const f of [...r.signature.inputs, ...r.signature.outputs]) {
    if (!f || !valid(f.name) || !valid(f.type) || typeof f.nullable !== 'boolean') throw new CandidateInputError('invalid_field');
  }
  for (const fields of [r.signature.inputs, r.signature.outputs]) {
    if (new Set(fields.map(f => f.name)).size !== fields.length) throw new CandidateInputError('duplicate_field_name');
  }
  for (const key of ['effects', 'failures']) if (r.signature[key].some(x => !valid(x))) throw new CandidateInputError('invalid_signature_item');
  if (r.kind === 'ui' && (!r.platform || !valid(r.platform.target) || !valid(r.platform.framework))) throw new CandidateInputError('missing_ui_platform');
  if (r.contract && (typeof r.contract !== 'object' || (r.contract.id !== undefined && !valid(r.contract.id)) || (r.contract.fingerprint !== undefined && !valid(r.contract.fingerprint)))) throw new CandidateInputError('invalid_contract');
  if (r.labels !== undefined && (!Array.isArray(r.labels) || r.labels.some(x => !valid(x)))) throw new CandidateInputError('invalid_labels');
}

function compareFieldList(a, b) {
  const normalize = values => values.map(v => ({name:v.name,type:v.type,nullable:v.nullable}))
    .sort((x,y) => x.name.localeCompare(y.name));
  return stable(normalize(a)) === stable(normalize(b));
}
function compareSignatures(a, b) {
  return compareFieldList(a.inputs, b.inputs) && compareFieldList(a.outputs, b.outputs) &&
    stable(normalizedSet(a.effects)) === stable(normalizedSet(b.effects)) &&
    stable(normalizedSet(a.failures)) === stable(normalizedSet(b.failures));
}
function lexicalScore(a, b) {
  const tokenize = r => new Set([r.id.split('.').slice(1).join(' '), ...(r.labels || [])].flatMap(s => s.toLowerCase().split(/[^a-z0-9\p{L}]+/u)).filter(Boolean));
  const x=tokenize(a), y=tokenize(b);
  const common=[...x].filter(t=>y.has(t)).length, total=new Set([...x,...y]).size;
  return total ? Math.round(100 * common / total) / 100 : 0;
}

// A high ranking is a retrieval clue, never calibrated confidence or Accept.
export function comparePair(a, b) {
  checkRecord(a); checkRecord(b);
  if (a.application === b.application) throw new CandidateInputError('same_application');
  let state, reason;
  if (a.kind !== b.kind) { state = STATES.incompatible; reason = 'different_function_kind'; }
  else if (!compareSignatures(a.signature, b.signature)) { state = STATES.incompatible; reason = 'signature_or_effects_mismatch'; }
  else if (!a.contract?.id || !a.contract?.fingerprint || !b.contract?.id || !b.contract?.fingerprint) {
    state = STATES.review; reason = 'missing_authoritative_contract';
  } else if (a.contract.id !== b.contract.id) { state = STATES.review; reason = 'unverified_semantic_equivalence'; }
  else if (a.contract.fingerprint !== b.contract.fingerprint) { state = STATES.incompatible; reason = 'contract_fingerprint_drift'; }
  else if (a.kind === 'ui' && (a.platform.target !== b.platform.target || a.platform.framework !== b.platform.framework)) {
    state = STATES.pattern; reason = 'target_or_framework_differs';
  } else { state = STATES.reuse; reason = 'same_pinned_contract_and_signature'; }
  return {
    left: a.id, right: b.id, state, reason, lexicalHint: lexicalScore(a,b),
    sources: [a,b].map(x=>({id:x.id,locator:x.source.locator,digest:x.source.digest,revision:x.source.revision})),
    acceptance: 'not_accepted', authority: 'read_only_candidate'
  };
}

// Caller supplies a filtering allowlist; upstream MUST perform actual access control before supplying records.
export function findReuseCandidates(records, {allowedApplications} = {}) {
  if (!Array.isArray(records) || !Array.isArray(allowedApplications) || allowedApplications.length === 0 || allowedApplications.some(x=>!valid(x))) throw new CandidateInputError('authorization_scope_required');
  const allowed = new Set(allowedApplications);
  const visible = records.filter(r => allowed.has(r?.application));
  for (const r of visible) checkRecord(r);
  if (new Set(visible.map(r=>r.id)).size !== visible.length) throw new CandidateInputError('duplicate_id');
  const results=[];
  const ordered=[...visible].sort((a,b)=>a.id.localeCompare(b.id));
  for (let i=0;i<ordered.length;i++) for(let j=i+1;j<ordered.length;j++) {
    if (ordered[i].application!==ordered[j].application) results.push(comparePair(ordered[i], ordered[j]));
  }
  const priority={reuse_candidate:0,pattern_candidate:1,needs_semantic_review:2,incompatible:3};
  return results.sort((a,b)=>priority[a.state]-priority[b.state] || b.lexicalHint-a.lexicalHint || a.left.localeCompare(b.left) || a.right.localeCompare(b.right));
}
