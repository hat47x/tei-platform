// E28 is a read-only recorder for caller-attested composition work.
// It never infers labor, AI questions, acceptance or permissions from a
// Source listing, a pull request or an E27 selection proposal.
const ID = /^[A-Za-z][A-Za-z0-9._-]{0,119}$/;
const SHA = /^[a-f0-9]{40}$/;
const UTC = /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,3})?Z$/;
const REPO = /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;
const PATH = /^[^/\\][^\\]*$/;
const kinds = new Set(['manual-action','ai-question','source-change','verification']);
const coverageKeys = ['manualActions','aiQuestions','sourceChanges','verifications'];
const approaches = new Set(['existing-sources','profile-candidate']);
const targets = new Set(['web','native']);
const consumers = new Set(['order_entry','sui_stage0']);
const modes = new Set(['illustrative','caller-recorded']);

const object = x => x!==null && typeof x==='object' && !Array.isArray(x);
const validID = x => typeof x==='string' && ID.test(x);
const validPath = x => typeof x==='string' && PATH.test(x) &&
  !x.split('/').some(y=>!y || y==='.' || y==='..') &&
  !x.includes('\u0000') && !x.startsWith('/');
const validTime = x => typeof x==='string' && UTC.test(x) &&
  !Number.isNaN(Date.parse(x)) && new Date(x).toISOString().startsWith(x.slice(0,19));
const fail = code => {throw new TypeError(code);};

function validateSession(session) {
  if (!object(session) || !validID(session.id) || !validID(session.requirementId) ||
      !consumers.has(session.consumer) || !approaches.has(session.approach) ||
      !targets.has(session.target) || !modes.has(session.mode) ||
      !['open','closed'].includes(session.state) || !object(session.baseline) ||
      !SHA.test(session.baseline.tei) ||
      (session.baseline.sui!==null && session.baseline.sui!==undefined &&
        !SHA.test(session.baseline.sui)) ||
      !object(session.coverage) ||
      Object.keys(session.coverage).sort().join('|')!==coverageKeys.slice().sort().join('|') ||
      coverageKeys.some(k=>!['complete','untracked'].includes(session.coverage[k]))) fail('invalid_session');
  if (session.consumer==='sui_stage0' && !session.baseline.sui) fail('missing_sui_baseline');
}

function validateEntry(entry, session, known) {
  if (!object(entry) || !validID(entry.id) || !kinds.has(entry.kind) ||
      !validTime(entry.observedAt) || !known.has(entry.choiceId) ||
      known.get(entry.choiceId).consumer!==session.consumer ||
      !object(entry.provenance) ||
      !['human-attested','tool-captured'].includes(entry.provenance.kind) ||
      typeof entry.provenance.ref!=='string' ||
      !/^[A-Za-z0-9_.:/@#-]{1,250}$/.test(entry.provenance.ref)) fail('invalid_work_event');
  if (entry.kind==='source-change') {
    if (!object(entry.file) || !REPO.test(entry.file.repository) ||
        !validPath(entry.file.path) ||
        ![null,undefined].includes(entry.file.before) && !SHA.test(entry.file.before) ||
        ![null,undefined].includes(entry.file.after) && !SHA.test(entry.file.after) ||
        entry.file.before===entry.file.after ||
        !entry.file.before && !entry.file.after) fail('invalid_file_change');
  } else if (entry.file!==undefined) fail('unexpected_file_change');
  if (entry.kind==='verification') {
    if (!['pass','fail','unknown'].includes(entry.outcome)) fail('invalid_verification_outcome');
  } else if (entry.outcome!==undefined) fail('unexpected_verification_outcome');
  // Only structured metadata. No copied emails, prompts, secrets or business data.
  const allowed = new Set(['id','kind','observedAt','choiceId','provenance',
    ...(entry.kind==='source-change'?['file']:[]),
    ...(entry.kind==='verification'?['outcome']:[])]);
  if (Object.keys(entry).some(k=>!allowed.has(k))) fail('unexpected_work_event_field');
}

export function measureE28Work({session,events=[],audit}={}) {
  validateSession(session);
  if (!object(audit) || audit.kind!=='tei-composition-e27-reference-selection-audit-v0' ||
      !Array.isArray(audit.choices) || !Array.isArray(events)) fail('invalid_audit_or_events');
  const known = new Map();
  for (const choice of audit.choices) {
    if (!object(choice) || !validID(choice.id) || !consumers.has(choice.consumer) ||
        known.has(choice.id)) fail('invalid_audit_choice');
    known.set(choice.id,choice);
  }
  const seen = new Set();
  const counts = Object.fromEntries([...kinds].map(k=>[k,0]));
  const files = new Set();
  const outcomes = {pass:0,fail:0,unknown:0};
  for (const e of events) {
    validateEntry(e,session,known);
    if (seen.has(e.id)) fail('duplicate_work_event');
    seen.add(e.id);
    counts[e.kind]++;
    if (e.kind==='source-change') files.add(e.file.repository+'/'+e.file.path);
    if (e.kind==='verification') outcomes[e.outcome]++;
  }
  const reported = session.mode==='caller-recorded' && session.state==='closed';
  const measure = (field,value)=>reported && session.coverage[field]==='complete'?value:null;
  return {
    kind:'tei-composition-e28-attested-work-measure-v0',
    source:'caller-supplied-events-not-independently-verified',
    sessionId:session.id,consumer:session.consumer,requirementId:session.requirementId,
    approach:session.approach,target:session.target,baseline:{...session.baseline},
    mode:session.mode,state:session.state,
    eventCounts:{...counts},
    measurements:{
      manualActions:measure('manualActions',counts['manual-action']),
      aiQuestions:measure('aiQuestions',counts['ai-question']),
      changedFiles:measure('sourceChanges',files.size),
      verificationRuns:measure('verifications',counts.verification),
    },
    verificationOutcomes:session.mode==='caller-recorded' &&
      session.state==='closed' && session.coverage.verifications==='complete'?outcomes:null,
    // Even a complete self-attestation does not establish accepted semantics,
    // deployed Target, valid source provenance or current runtime permission.
    acceptedBusinessMeaning:null,realTargetQualified:null,runtimeAuthority:null,
  };
}

export function compareE28Work(left,right) {
  if (!object(left) || !object(right) ||
      left.kind!=='tei-composition-e28-attested-work-measure-v0' ||
      right.kind!==left.kind) fail('invalid_measurement');
  const same = left.consumer===right.consumer &&
    left.requirementId===right.requirementId &&
    left.target===right.target &&
    left.baseline?.tei===right.baseline?.tei &&
    left.baseline?.sui===right.baseline?.sui &&
    left.approach==='existing-sources' && right.approach==='profile-candidate';
  if (!same) return {status:'not-comparable',reason:'different-task-baseline-or-approach'};
  const keys = ['manualActions','aiQuestions','changedFiles','verificationRuns'];
  if (!keys.every(k=>Number.isInteger(left.measurements?.[k]) &&
     Number.isInteger(right.measurements?.[k]))) {
    return {status:'not-comparable',reason:'incomplete-attested-coverage'};
  }
  return {
    status:'caller-reported-comparison-only',
    differences:Object.fromEntries(keys.map(k=>[k,right.measurements[k]-left.measurements[k]])),
    productAdoptionQualified:false,
    reason:'real-target-and-authority-require-independent-evidence',
  };
}
