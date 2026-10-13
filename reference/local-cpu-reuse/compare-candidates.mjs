// Research-only deterministic pairwise reuse triage; NOT semantic authority.
const KINDS = new Set(['ui', 'logic']);
const STATES = { reuse: 'reuse_candidate', pattern: 'pattern_candidate', review: 'needs_semantic_review', incompatible: 'incompatible' };
const MAX_RECORDS = 4096, MAX_PAIRS = 65536;

export class CandidateInputError extends Error {
  constructor(code) { super(code); this.code = code; }
}
const valid = (s) => typeof s === 'string' && s.trim().length > 0;
const stable = (a) => JSON.stringify(a);
const ordinal = (a,b) => a < b ? -1 : a > b ? 1 : 0;
const plain = (x) => x !== null && typeof x === 'object' && !Array.isArray(x) &&
  (Object.getPrototypeOf(x) === Object.prototype || Object.getPrototypeOf(x) === null);
const shape = (x, keys) => plain(x) && Object.keys(x).every(k => keys.includes(k));
const normalizedSet = (a) => [...a].sort(ordinal);

function checkRecord(r) {
  if (!plain(r) || !valid(r.id) || !valid(r.application) || !KINDS.has(r.kind)) throw new CandidateInputError('invalid_record');
  // Unknown fields can carry unexamined semantic constraints: never discard them and infer reuse.
  if (!shape(r, ['id','application','kind','source','contract','signature','platform','labels'])) throw new CandidateInputError('unsupported_record_field');
  if (!shape(r.source, ['locator','digest','revision']) || !valid(r.source.locator) || !valid(r.source.digest) || !valid(r.source.revision)) throw new CandidateInputError('missing_source_provenance');
  if (!shape(r.signature, ['inputs','outputs','effects','failures']) || !Array.isArray(r.signature.inputs) || !Array.isArray(r.signature.outputs) || !Array.isArray(r.signature.effects) || !Array.isArray(r.signature.failures)) throw new CandidateInputError('invalid_signature');
  for (const f of [...r.signature.inputs, ...r.signature.outputs]) {
    if (!shape(f, ['name','type','nullable']) || !valid(f.name) || !valid(f.type) || typeof f.nullable !== 'boolean') throw new CandidateInputError('invalid_field');
  }
  for (const fields of [r.signature.inputs, r.signature.outputs]) {
    if (new Set(fields.map(f => f.name)).size !== fields.length) throw new CandidateInputError('duplicate_field_name');
  }
  for (const key of ['effects', 'failures']) {
    const values = r.signature[key];
    if (values.some(x => !valid(x))) throw new CandidateInputError('invalid_signature_item');
    if (new Set(values).size !== values.length) throw new CandidateInputError('duplicate_signature_item');
  }
  if (r.kind === 'ui' && (!shape(r.platform, ['target','framework']) || !valid(r.platform.target) || !valid(r.platform.framework))) throw new CandidateInputError('missing_ui_platform');
  if (r.kind === 'logic' && r.platform !== undefined) throw new CandidateInputError('unsupported_logic_platform');
  if (r.contract !== undefined && r.contract !== null &&
      (!shape(r.contract, ['id','fingerprint']) || (r.contract.id !== undefined && !valid(r.contract.id)) || (r.contract.fingerprint !== undefined && !valid(r.contract.fingerprint)))) throw new CandidateInputError('invalid_contract');
  if (r.labels !== undefined && (!Array.isArray(r.labels) || r.labels.some(x => !valid(x)))) throw new CandidateInputError('invalid_labels');
}

function compareFieldList(a, b) {
  const normalize = values => values.map(v => ({name:v.name,type:v.type,nullable:v.nullable}))
    .sort((x,y) => ordinal(x.name,y.name));
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
  if (a.id === b.id) throw new CandidateInputError('duplicate_id');
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
// Bounds apply to the already-authorized visible set; this is not a security boundary for upstream collection.
export function findReuseCandidates(records, options = {}) {
  if (!plain(options)) throw new CandidateInputError('invalid_options');
  const {allowedApplications, maxRecords=256, maxPairs=8192} = options;
  if (!Array.isArray(records) || !Array.isArray(allowedApplications) || allowedApplications.length === 0 || allowedApplications.some(x=>!valid(x))) throw new CandidateInputError('authorization_scope_required');
  if (!Number.isSafeInteger(maxRecords) || maxRecords < 1 || maxRecords > MAX_RECORDS ||
      !Number.isSafeInteger(maxPairs) || maxPairs < 1 || maxPairs > MAX_PAIRS) throw new CandidateInputError('invalid_limits');
  const allowed = new Set(allowedApplications);
  const visible = records.filter(r => allowed.has(r?.application));
  if (visible.length > maxRecords) throw new CandidateInputError('record_limit_exceeded');
  for (const r of visible) checkRecord(r);
  if (new Set(visible.map(r=>r.id)).size !== visible.length) throw new CandidateInputError('duplicate_id');
  const perApp = new Map();
  for (const r of visible) perApp.set(r.application,(perApp.get(r.application)||0)+1);
  const pairCount=(visible.length**2 - [...perApp.values()].reduce((sum,n)=>sum+n**2,0))/2;
  if (pairCount > maxPairs) throw new CandidateInputError('pair_limit_exceeded');
  const results=[];
  const ordered=[...visible].sort((a,b)=>ordinal(a.id,b.id));
  for (let i=0;i<ordered.length;i++) for(let j=i+1;j<ordered.length;j++) {
    if (ordered[i].application!==ordered[j].application) results.push(comparePair(ordered[i], ordered[j]));
  }
  const priority={reuse_candidate:0,pattern_candidate:1,needs_semantic_review:2,incompatible:3};
  return results.sort((a,b)=>priority[a.state]-priority[b.state] || b.lexicalHint-a.lexicalHint || ordinal(a.left,b.left) || ordinal(a.right,b.right));
}
