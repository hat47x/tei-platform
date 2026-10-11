import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {fileURLToPath} from "node:url";
import {dirname, join} from "node:path";
import {proposeDFDReviewEdit, EDIT_SCHEMA, PREVIEW_SCHEMA} from "./dfd-edit-proposal.mjs";

const root = dirname(fileURLToPath(import.meta.url));
const read = path => JSON.parse(readFileSync(join(root,path),"utf8"));
const procedure = read("dfd-order-approval/order-approval-procedure.json");
const review = read("dfd-order-approval/review-proposal.json");
const sha = "sha256:"+"a".repeat(64);
const edit = (id,operation,overrides={}) => ({
  schema:EDIT_SCHEMA,id,origin:"studio",intent:"test candidate",
  baseReviewSHA256:sha,procedureId:procedure.procedure.id,operation,...overrides
});

test("Role assignment is a new unaccepted review proposal, never a Source edit",()=>{
  const before=JSON.stringify(review);
  const result=proposeDFDReviewEdit(procedure,review,sha,
    edit("assign",{kind:"assign-business-role",step:"APP-40",role:"approver"}));
  assert.equal(result.schema,PREVIEW_SCHEMA);
  assert.equal(result.authority,"unaccepted-proposal-only");
  assert.equal(result.acceptanceStatus,"not-accepted");
  assert.equal(result.readyForAcceptance,false);
  assert.deepEqual(result.sourceWrites,[]);
  assert.equal(result.modelRevisionStatus,"unverified");
  assert.equal(result.reviewSHA256Verification,"caller-supplied-not-cryptographically-verified");
  assert.equal(result.proposedReview.responsibilities.length,review.responsibilities.length+1);
  assert.equal(JSON.stringify(review),before);
});

test("Trigger label can change but trigger ID and kind cannot be rewritten",()=>{
  const result=proposeDFDReviewEdit(procedure,review,sha,
    edit("label",{kind:"replace-trigger-label",trigger:"TR-ORDER",label:"new wording"}));
  assert.equal(result.proposedReview.triggers[0].id,"TR-ORDER");
  assert.equal(result.proposedReview.triggers[0].kind,"event");
  assert.equal(result.proposedChange.action,"proposal-update");
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("label-invalid",{kind:"replace-trigger-label",trigger:"TR-ORDER",label:"new",
      runtimeBinding:"cron"})),/UNKNOWN_OR_INVALID_FIELD/);
});

test("Conditional control-flow is permitted only with an explicit condition",()=>{
  const result=proposeDFDReviewEdit(procedure,review,sha,
    edit("flow",{kind:"add-control-transition",from:"APP-60",
      to:"APP-00",transitionKind:"exception",condition:"review not complete"}));
  assert.equal(result.proposedChange.kind,"control-transition");
  assert.equal(result.proposedReview.flows.at(-1).kind,"exception");
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("flow-unbound",{kind:"add-control-transition",from:"APP-60",
      to:"APP-00",transitionKind:"exception"})),/INVALID_EXACT_ID_OR_TEXT/);
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("flow-dup",{kind:"add-control-transition",from:"APP-00",
      to:"APP-10",transitionKind:"sequence"})),/DUPLICATE_CONTROL_FLOW/);
});

test("Information payload respects exact endpoint and Step process boundary",()=>{
  const result=proposeDFDReviewEdit(procedure,review,sha,
    edit("info",{kind:"add-information-flow",id:"INFO-NEW",
      from:{kind:"party",id:"supplier"},to:{kind:"step",id:"APP-20"},
      payload:"independent availability information"}));
  assert.equal(result.proposedReview.informationFlows.at(-1).id,"INFO-NEW");
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("info-no-step",{kind:"add-information-flow",id:"INFO-NEW",
      from:{kind:"party",id:"supplier"},to:{kind:"store",id:"orders"},
      payload:"direct store write"})),/INFORMATION_PROCESS_BOUNDARY_REQUIRED/);
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("info-dup",{kind:"add-information-flow",id:"INFO-ORDER",
      from:{kind:"store",id:"orders"},to:{kind:"step",id:"APP-00"},payload:"duplicate"})),
    /DUPLICATE_INFORMATION_FLOW/);
});

test("Stale digest, arbitrary patch and fake role/Step are rejected",()=>{
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("stale",{kind:"assign-business-role",step:"APP-40",role:"approver"},
      {baseReviewSHA256:"sha256:"+"b".repeat(64)})),/INVALID_EDIT_OR_STALE_REVIEW/);
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("wrong-id",{kind:"assign-business-role",step:"fake",role:"approver"})),
    /UNRESOLVED_ROLE_ASSIGNMENT/);
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("wrong-role",{kind:"assign-business-role",step:"APP-40",role:"auth.admin"})),
    /UNRESOLVED_ROLE_ASSIGNMENT/);
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    edit("patch",{kind:"arbitrary-patch",path:"/steps/0",value:"write"})),
    /UNSUPPORTED_EDIT_OPERATION/);
  assert.throws(()=>proposeDFDReviewEdit(procedure,review,sha,
    {...edit("source",{kind:"replace-trigger-label",trigger:"TR-ORDER",label:"text"}),
       target:"source"}),/UNKNOWN_OR_INVALID_FIELD/);
});

test("Unaccepted Procedure identity is not substituted for Canonical authoring",()=>{
  assert.throws(()=>proposeDFDReviewEdit({...procedure,canonical:true},review,sha,
    edit("wrong",{kind:"replace-trigger-label",trigger:"TR-ORDER",label:"text"})),
    /PROCEDURE_IDENTITY_OR_AUTHORITY_MISMATCH/);
  assert.throws(()=>proposeDFDReviewEdit(procedure,{...review,authority:"accepted"},sha,
    edit("wrong",{kind:"replace-trigger-label",trigger:"TR-ORDER",label:"text"})),
    /REVIEW_NOT_UNACCEPTED_PROPOSAL/);
});
