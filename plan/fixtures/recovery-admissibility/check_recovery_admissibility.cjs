"use strict";
// TEI x EKI recovery feasibility AND effect safety REFERENCE ONLY.
// No actual TEI/EKI/Provider/Host/Permit authority, pricing or placement.
// Run: node plan/fixtures/recovery-admissibility/check_recovery_admissibility.cjs
// Independent synthetic scenarios; statuses are local TEST labels, not
// adopted EKI state, TEI Job ABI, recovery permit or Host grant.
function qualify(s) {
  const out = (decision, reason) => ({decision, reason});
  if (s.effect === "business" && s.effectState === "unknown")
    return out("reconcile-only", "commit-uncertain-even-if-output-verified");
  if (s.output === "body-verified")
    return out("no-new-attempt", "already-verified; current-disclosure-separate");
  if (s.effect === "business" && s.effectState === "committed")
    return out("retrieve-only", "committed-effect-result-unverified");
  if (s.taskPhase === "completed" && s.output !== "body-verified")
    return out("retrieve-only", "task-completed-is-not-verified-body");
  if (s.activeLease)
    return out("observe-only", "active-attempt-not-replaceable");
  if (!s.sourceQualified || !s.targetBound || !s.grantCurrent)
    return out("blocked", "source-target-or-current-grant-unqualified");
  if (s.readComplete !== true)
    return out("needs-evidence", "input-or-scope-partial-unknown");
  if (s.effect === "business") {
    if (s.backendRetry !== "disabled" && !s.businessIdempotencyProven)
      return out("blocked", "opaque-backend-retry-and-non-idempotent-effect");
    if (!s.sameOperationID || !s.businessRetryAllowed)
      return out("blocked", "business-retry-or-identity-not-qualified");
  } else if (s.backendRetry === "opaque")
    return out("needs-evidence", "hidden-retries-not-budget-accounted");
  if (!s.permitCurrent || !s.leaseFencingCurrent)
    return out("blocked", "current-recovery-permit-or-fence-missing");
  if (s.resourceEligibility === "denied")
    return out("blocked", "resource-not-eligible");
  if (s.resourceEligibility !== "eligible" || s.reservation !== "confirmed")
    return out("needs-evidence", "availability-or-reservation-not-confirmed");
  if (s.attemptsRemaining < 1 || s.physicalNeed > s.physicalRemaining)
    return out("not-feasible", "physical-attempt-budget-exhausted");
  if (s.financeProof === "denied" || s.externalCostNeed > s.externalCostRemaining)
    return out("not-feasible", "external-spend-not-authorized");
  if (s.financeProof !== "approved")
    return out("needs-evidence", "price-or-external-spend-unknown");
  if (s.recoveryIndependence !== "established" && s.requiresIndependentRecovery)
    return out("needs-evidence", "correlated-failure-independence-unknown");
  const durations = [s.queue, s.inputTransfer, s.compute, s.outputTransfer,
    s.verification, s.recoveryAllowance];
  if (durations.some(v => !Number.isFinite(v) || v < 0) ||
      !Number.isFinite(s.now) || !Number.isFinite(s.deadline))
    return out("needs-evidence", "time-bound-not-measured");
  const needed = durations.reduce((a,b) => a+b,0);
  if (s.now + needed > s.deadline)
    return out("not-feasible", "verified-result-plus-recovery-exceeds-deadline");
  return out("candidate-only", "synthetic-evidence-not-actual-grant");
}
const baseline = Object.freeze({
  effect:"none", effectState:"not-started", output:"none", taskPhase:"queued",
  activeLease:false, sourceQualified:true, targetBound:true, grantCurrent:true,
  readComplete:true, backendRetry:"disabled", businessIdempotencyProven:false,
  businessRetryAllowed:false, sameOperationID:false,
  permitCurrent:true, leaseFencingCurrent:true,
  resourceEligibility:"eligible", reservation:"confirmed",
  attemptsRemaining:1, physicalNeed:20, physicalRemaining:50,
  financeProof:"approved", externalCostNeed:10, externalCostRemaining:30,
  recoveryIndependence:"established", requiresIndependentRecovery:false,
  now:0, deadline:90, queue:10, inputTransfer:10, compute:20,
  outputTransfer:5, verification:5, recoveryAllowance:10,
});
const cases = [
  ["RA-01","pure job: verified result incl recovery",{},"candidate-only"],
  ["RA-02","verified historical output: no re-dispatch",
    {output:"body-verified",grantCurrent:false},"no-new-attempt"],
  ["RA-03","Commit unknown blocks resubmit despite deadline pressure",
    {effect:"business",effectState:"unknown",now:89,deadline:90},"reconcile-only"],
  ["RA-04","committed effect but missing delivered body: retrieve not rerun",
    {effect:"business",effectState:"committed",output:"ack-only"},"retrieve-only"],
  ["RA-05","EKI completed without result body is retrieval-only",
    {taskPhase:"completed",output:"ack-only"},"retrieve-only"],
  ["RA-06","O1 selected / O2 payload target is not bound",
    {targetBound:false},"blocked"],
  ["RA-07","stale current Host grant not revived by placement",
    {grantCurrent:false},"blocked"],
  ["RA-08","partial input cannot prove complete task",
    {readComplete:false},"needs-evidence"],
  ["RA-09","opaque backend retry on non-idempotent business effect",
    {effect:"business",businessRetryAllowed:true,sameOperationID:true,
      backendRetry:"opaque"},"blocked"],
  ["RA-10","effectful candidate with explicit retry and disabled backend retry",
    {effect:"business",businessRetryAllowed:true,sameOperationID:true},"candidate-only"],
  ["RA-11","pure job with opaque extra executions has unbounded cost",
    {backendRetry:"opaque"},"needs-evidence"],
  ["RA-12","active lease may still be running",
    {activeLease:true},"observe-only"],
  ["RA-13","old Permit or missing lease fence cannot authorize a new Attempt",
    {permitCurrent:false},"blocked"],
  ["RA-14","unconfirmed group reservation is not physical capacity",
    {reservation:"forecast-only"},"needs-evidence"],
  ["RA-15","physical budget short despite sufficient monetary allowance",
    {physicalRemaining:10},"not-feasible"],
  ["RA-16","financial spend lacks current approval despite physical budget",
    {financeProof:"unknown"},"needs-evidence"],
  ["RA-17","possible spending exceeds externally approved financial cap",
    {externalCostRemaining:5},"not-feasible"],
  ["RA-18","all costs + verify+recovery exceed hard deadline",
    {now:35},"not-feasible"],
  ["RA-19","compute finishes before deadline but verification too late",
    {now:50,verification:20,recoveryAllowance:0},"not-feasible"],
  ["RA-20","groups share failure domain; independence unverified",
    {requiresIndependentRecovery:true,recoveryIndependence:"unknown"},"needs-evidence"],
  ["RA-21","revoked input Source qualification blocks reissue",
    {sourceQualified:false},"blocked"],
  ["RA-22","no remaining Attempt cannot be revived by a cheap estimate",
    {attemptsRemaining:0},"not-feasible"],
  ["RA-23","pure task but resource explicitly denied",
    {resourceEligibility:"denied"},"blocked"],
  ["RA-24","unknown verification time is not presumed zero",
    {verification:null},"needs-evidence"],
  ["RA-25","verified output cannot settle uncertain business commit",
    {effect:"business",effectState:"unknown",output:"body-verified"},"reconcile-only"],
  ["RA-26","business effect has no accepted retry semantics",
    {effect:"business",sameOperationID:true,businessRetryAllowed:false},"blocked"],
];
function runCases() {
  const rows=[];
  for(const [id,description,change,expected] of cases) {
    const actual=qualify({...baseline,...change});
    if(actual.decision!==expected) throw new Error(
      id+" "+description+": expected "+expected+", got "+actual.decision+", reason "+actual.reason);
    rows.push({id,expected,reason:actual.reason});
  }
  return {status:"PASS",count:rows.length,scope:"synthetic-reference-only",
    categories:{candidate:rows.filter(x=>x.expected==="candidate-only").length,
      blocked:rows.filter(x=>x.expected==="blocked").length,
      other:rows.filter(x=>!["candidate-only","blocked"].includes(x.expected)).length},
    cases:rows};
}
if (typeof module !== "undefined") {
  module.exports={qualify,runCases};
  if(typeof require!=="undefined" && require.main===module)
    process.stdout.write(JSON.stringify(runCases(),null,2)+"\n");
}
