// External contract compatibility pressure against a PINNED EKI OpenAPI fixture.
//
// This is a test-only classifier: it never submits a Task, invokes a Worker,
// grants authority, creates a Business Effect, or proves runtime availability.
// Use "node --test tools/reference-tei-eki-contract/probe.test.cjs".
"use strict";
const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const witness = JSON.parse(fs.readFileSync(path.join(__dirname, "eki-openapi-v1-snapshot.json"), "utf8"));
const schemas = witness.components.schemas;
const task = schemas.TaskRequest;
const effectMode = schemas.ReliabilityRequest.properties.effect_mode.enum;
const paramDescription = task.properties.parameters.description || "";
const versionDescription = schemas.ExecutorRequirement.properties.version_constraint.description || "";
const artifactGETCodes = witness.paths["/v1/artifacts/{artifact_id}"].responses.get || [];

function classify(job, contract = witness) {
  const c = contract.components.schemas;
  const p = c.TaskRequest.properties;
  const effectModes = c.ReliabilityRequest.properties.effect_mode.enum;
  const issues = [];
  const unresolved = [];
  if (job.effectMode && !effectModes.includes(job.effectMode)) issues.push("effect_mode_not_in_v1");
  if (job.effectMode && job.effectMode !== "none") unresolved.push("business_effect_requires_host_provider");
  if (Object.keys(job.parameters || {}).length) {
    const description = p.parameters.description || "";
    if (description.includes("not passed to workers")) issues.push("worker_parameters_not_delivered");
    else unresolved.push("worker_parameter_delivery_not_verified");
  }
  if (job.executorDigest) {
    if (!Object.hasOwn(c.ExecutorRequirement.properties,"digest")) issues.push("executor_digest_binding_not_representable");
    else unresolved.push("executor_digest_runtime_enforcement_not_verified");
  }
  if (job.versionConstraint) {
    if ((c.ExecutorRequirement.properties.version_constraint?.description || "").includes("does not evaluate")) {
      unresolved.push("executor_version_constraint_not_enforced_by_admission");
    } else unresolved.push("executor_version_constraint_runtime_unverified");
  }
  if (job.notBefore) {
    if (!Object.hasOwn(p,"not_before")) issues.push("not_before_not_representable");
    else unresolved.push("not_before_semantics_not_verified");
  }
  if (job.preferredCompletionAt) {
    if (!Object.hasOwn(p,"preferred_completion_at")) issues.push("preferred_completion_not_representable");
    else unresolved.push("preferred_completion_semantics_not_verified");
  }
  if (job.latestUsefulAt) {
    if (!Object.hasOwn(p,"deadline")) issues.push("deadline_field_unavailable");
    unresolved.push("deadline_scope_includes_consumer_verification_unproven");
  }
  if (job.resourceGroup) {
    if (!Object.hasOwn(p,"resource_group") && !Object.hasOwn(c.ResourceRequest.properties,"resource_group")) {
      issues.push("resource_group_selection_not_representable");
    } else unresolved.push("resource_group_grant_unverified");
  }
  if (job.dataResidency) {
    if (!Object.hasOwn(p,"data_residency") && !Object.hasOwn(c.ResourceRequest.properties,"data_residency")) {
      issues.push("data_residency_not_representable");
    } else unresolved.push("data_residency_enforcement_unverified");
  }
  if (job.inputs?.length) unresolved.push("input_artifact_ingest_availability_unverified");
  if (job.needsOutputBytes) unresolved.push("result_artifact_body_actual_read_unverified");
  if (job.requiresGuaranteedDeadline) unresolved.push("deadline_sla_not_proven");
  if (job.requiresBusinessCommit) unresolved.push("business_commit_receipt_not_proven");
  if (job.exactlyOnce) issues.push("exactly_once_effect_not_guaranteed");
  return Object.freeze({
    representability: issues.length ? "not-representable" : "wire-shape-only",
    runtimeQualification: "NOT_RUN",
    issues: Object.freeze([...new Set(issues)].sort()),
    unresolved: Object.freeze([...new Set(unresolved)].sort()),
  });
}

test("WP-01 actual EKI v1 source blob and contract field provenance are explicit", () => {
  assert.equal(witness.source.repository, "hat47x/eki-fabric");
  assert.equal(witness.source.path, "spec/external/openapi.json");
  assert.equal(witness.source.source_git_blob_sha, "388f2e3ad32ba663b4777fcfb4304b0cf6f48f3f");
  assert.deepEqual(task.required, ["schema","executor","inputs","parameters","resources","reliability"]);
  assert.equal(task.properties.schema.const, "eki.task-request/v1");
});

test("WP-02 EKI v1 effect mode allows only pure computation", () => {
  assert.deepEqual(effectMode, ["none"]);
  assert.deepEqual(schemas.ReliabilityRequest.required, ["retryable","preemptible","max_attempts","effect_mode"]);
});

test("WP-03 parameters are admitted in shape but not delivered to workers", () => {
  assert.ok(Object.hasOwn(task.properties,"parameters"));
  assert.match(paramDescription,/not passed to workers/);
});

test("WP-04 executor version constraint is recorded, not enforced by admission", () => {
  assert.ok(Object.hasOwn(schemas.ExecutorRequirement.properties,"version_constraint"));
  assert.match(versionDescription,/does not evaluate/);
  assert.equal(Object.hasOwn(schemas.ExecutorRequirement.properties,"digest"),false);
});

test("WP-05 one deadline is not TEI's start, target, and verified-useful deadlines", () => {
  assert.ok(Object.hasOwn(task.properties,"deadline"));
  assert.equal(Object.hasOwn(task.properties,"not_before"),false);
  assert.equal(Object.hasOwn(task.properties,"preferred_completion_at"),false);
});

test("WP-06 artifact retrieval route's declared 501 is not evidence of body delivery", () => {
  assert.ok(artifactGETCodes.includes("501"));
  assert.ok(witness.paths["/v1/artifacts/{artifact_id}"].verbs.includes("get"));
  assert.ok(Object.hasOwn(schemas.TaskResult.properties,"artifacts"));
  assert.ok(Object.hasOwn(schemas.ResultArtifact.properties,"available"));
});

test("WP-07 public completed Task status is not a business commit receipt", () => {
  assert.ok(schemas.TaskPublicStatus.properties.status.enum.includes("completed"));
  assert.equal(Object.hasOwn(schemas.TaskResult.properties,"business_commit_receipt"),false);
  assert.equal(Object.hasOwn(schemas.TaskAccepted.properties,"business_commit_receipt"),false);
});

const rows = [
 {id:"WP-08",name:"plain pure job produces only request shape",job:{effectMode:"none"},issues:[],unresolved:[]},
 {id:"WP-09",name:"nonempty worker parameters cannot silently disappear",job:{effectMode:"none",parameters:{domain:"orders"}},issues:["worker_parameters_not_delivered"],unresolved:[]},
 {id:"WP-10",name:"business DB effect cannot be projected as pure task",job:{effectMode:"write",requiresBusinessCommit:true},issues:["effect_mode_not_in_v1"],unresolved:["business_commit_receipt_not_proven","business_effect_requires_host_provider"]},
 {id:"WP-11",name:"exact executor digest cannot be silently reduced to version constraint",job:{effectMode:"none",executorDigest:"sha256:example",versionConstraint:"=1.2"},issues:["executor_digest_binding_not_representable"],unresolved:["executor_version_constraint_not_enforced_by_admission"]},
 {id:"WP-12",name:"earliest/start and preferred deadline are not single EKI deadline",job:{effectMode:"none",notBefore:"2026-11-04T01:00:00Z",preferredCompletionAt:"2026-11-04T02:00:00Z",latestUsefulAt:"2026-11-04T03:00:00Z"},issues:["not_before_not_representable","preferred_completion_not_representable"],unresolved:["deadline_scope_includes_consumer_verification_unproven"]},
 {id:"WP-13",name:"latest useful only still needs semantic deadline-scope qualification",job:{effectMode:"none",latestUsefulAt:"2026-11-04T03:00:00Z",requiresGuaranteedDeadline:true},issues:[],unresolved:["deadline_scope_includes_consumer_verification_unproven","deadline_sla_not_proven"]},
 {id:"WP-14",name:"group and residency selection require separate authorized profile",job:{effectMode:"none",resourceGroup:"HPC-A",dataResidency:"onprem"},issues:["data_residency_not_representable","resource_group_selection_not_representable"],unresolved:[]},
 {id:"WP-15",name:"artifact input/output availability requires real body acquisition",job:{effectMode:"none",inputs:[{name:"source",artifact_id:"A1"}],needsOutputBytes:true},issues:[],unresolved:["input_artifact_ingest_availability_unverified","result_artifact_body_actual_read_unverified"]},
 {id:"WP-16",name:"exactly once effect is not a Task-level v1 guarantee",job:{effectMode:"none",exactlyOnce:true},issues:["exactly_once_effect_not_guaranteed"],unresolved:[]},
 {id:"WP-17",name:"task completed never proves business verified",job:{effectMode:"none",requiresBusinessCommit:true,needsOutputBytes:true},issues:[],unresolved:["business_commit_receipt_not_proven","result_artifact_body_actual_read_unverified"]},
];
for (const row of rows) {
  test(row.id + " " + row.name, () => {
    const a = classify(row.job);
    assert.equal(a.representability,row.issues.length ? "not-representable" : "wire-shape-only");
    assert.equal(a.runtimeQualification,"NOT_RUN");
    assert.deepEqual(a.issues,[...row.issues].sort());
    assert.deepEqual(a.unresolved,[...row.unresolved].sort());
  });
}

test("WP-18 a hypothetical future effect enum does not imply safe host authority", () => {
  const clone = structuredClone(witness);
  clone.components.schemas.ReliabilityRequest.properties.effect_mode.enum.push("write");
  const a=classify({effectMode:"write",requiresBusinessCommit:true},clone);
  assert.deepEqual(a.issues,[]);
  assert.deepEqual(a.unresolved,["business_commit_receipt_not_proven","business_effect_requires_host_provider"]);
  assert.equal(a.runtimeQualification,"NOT_RUN");
});

test("WP-19 unknown contract proof is never turned into live EKI admission", () => {
  const a=classify({effectMode:"none",versionConstraint:"^1",inputs:[{name:"x",artifact_id:"art_01"}]});
  assert.equal(a.representability,"wire-shape-only");
  assert.equal(a.runtimeQualification,"NOT_RUN");
  assert.ok(a.unresolved.includes("executor_version_constraint_not_enforced_by_admission"));
  assert.ok(a.unresolved.includes("input_artifact_ingest_availability_unverified"));
});
