# Second Application Reference — S5 Independent Generative-AI Review Handoff

- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Review type: independent pre-Acceptance review by a **different generative AI**
- Proposal under review: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Acceptance status: **not accepted**
- Review status: **not yet performed**
- Rule: the author / prior evaluator's self-review does not satisfy this review requirement

## 1. Reviewer task

Review the exact Second Application Proposal and its Acceptance consequences. Do not execute Acceptance and do not reinterpret Preview / tests as a human decision.

Return one of:

```text
review-clear
review-clear-with-conditions
review-blocked
```

and include:

- reviewer / model identity and review date
- findings with severity and evidence
- whether each finding must be fixed before Acceptance or may remain unresolved with rationale
- explicit assessment of the golden / Gate B-C questions in §8
- no Acceptance decision on behalf of the human

## 2. Exact Proposal identity

Payload order:

```text
<Semantic source>
\n---\n
<Application source>
\n---\n
<Pattern source>
```

Identity:

```text
Semantic SHA-256:
  1005c6deb670c277c8b80fb0b353f1eecea690594f2ef4b80df28cc638736e63

Application SHA-256:
  f96fdb46f9385c205d360f56d0f91a6e841ff31b348ebf1db2cf03381751772f

Pattern SHA-256:
  41c8f81ebab384486d4df6a7cdaade07df8d07c89d22c25c2291d8b71a583ab1

Bundle:
  741 bytes
  c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d
```

The superseded two-source fingerprint `b347b41b...` was never accepted and must not be used.

## 3. Meaning added / reused

Destination-owned additions:

```text
namespace:repair_intake
semantic-data:repair_intake.intake
semantic-attribute:repair_intake.intake.customer_id
semantic-relation:repair_intake.intake.customer
application:repair_intake
application-pattern:repair_intake.customer_reference
datasource alias: repair_intake/customer_master
```

Shared authority reused by exact identity:

```text
semantic-data:customer.customer
semantic-type:customer.customer_id
database-contract:master_data.customer
database-contract-query:master_data.customer/find_by_id
```

Provenance only:

```text
application-pattern:order_entry.customer_reference
```

The Proposal does **not** add Persistence Mapping, Environment binding, Provider configuration, Permission / Resource grant, Business Procedure, or Target implementation.

## 4. Source files an Acceptance would create

```text
reference/workspace/semantic/repair-intake/intake.tei
reference/workspace/applications/repair-intake/application.tei
reference/workspace/applications/repair-intake/patterns/customer-reference.tei
```

No workspace manifest change is required by the current loader.

## 5. Executable pressure already obtained

Integrated at:

```text
41b21d1068bdfabd122b8543afe980efc4b5fdb5
```

Observed:

```text
focused S3-S5 tests                 = 22 PASS
full workspace positive Validation = PASS
unknown Relation negative control  = PASS
cardinality mismatch control       = PASS
reference evidence runner          = PASS
repository health check            = PASS
```

The negative controls require:

- `UNKNOWN_APPLICATION_PATTERN_RELATION`
- `REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH`

through the normal shared Validation path.

## 6. Shared Acceptance Preview already obtained

Multi-layer operations were implemented at `0f069838aecd3cc6627a1bc2c11cac308493021f`.

Latest re-Preview after Relation-impact expansion:

```text
baseModelRevision:
  sha256:f70da9b64a5b48c7682b24c64f455b0b0c2a7278be4faa297b6ff99c2d3707bf

Acceptance candidate fingerprint:
  sha256:8a936cf96ddeb4faa220285dcef5055d418030373f9425ed8d0d1c2b3e7c9b11

modelRevisionAfter:
  sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762

stages:
  11 / 11 PASS

changed subjects:
  6

changed collections:
  [applications, semantics]

impact edges:
  15

unresolved:
  ENVIRONMENT_BINDING_NOT_AUTHORED
  DRIVER_SCENARIO_NOT_RUN
  TARGET_PROJECTION_NOT_EVALUATED
```

Preview leaves `reference/workspace` byte-identical and is not Acceptance.

## 7. Retained post-Acceptance unresolved

The review should not require these to be guessed merely to accept the semantic composition:

- exact authorization principal
- exact Resource / Permission identity
- development binding target
- production binding target
- Provider verification
- Driver / Target Evidence
- change propagation Evidence

Assess whether any of these unexpectedly becomes a pre-Acceptance blocker; if so, identify the exact architecture or safety reason.

## 8. Required review questions

### 8.1 Reference golden

Current repository tests require the loaded current `reference/workspace` World to equal `reference/world.json`.

`tools/reference-golden` deterministically derives and checks:

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`

and uses temp-file + sync + rename for safe replacement.

The Shared Acceptance tool can also emit the accepted World through `-derived-out`; its JSON encoding is the same indented encoder used by `reference-golden`.

Pre-review technical observation, **not a review verdict**:

- source writes use atomic temp-file replacement and rollback on failure;
- `-derived-out` currently uses direct `os.WriteFile`;
- if that direct write fails after truncation / partial write, source rollback occurs but the derived file is not restored by the same rollback path.

Reviewer must assess:

1. whether Acceptance should be blocked until the derived-out path is made atomic, **or**
2. whether actual Acceptance should omit `-derived-out` and regenerate goldens separately through `go run ./tools/reference-golden -write` before repository health / commit, **or**
3. another safer transaction boundary.

Do not treat a stale `reference/world.json` as an acceptable committed state.

### 8.2 Gate B / Gate C historical baseline

Gate C Ready is historical Evidence tied to:

```text
R = e5231839a5dcc62cce1181a2731fcb7c70d16736
T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe
```

Its `canonical-world-invariant-R-to-T` check established that `reference/workspace/**` and `reference/world.json` were unchanged between those exact revisions.

The Second Application is intentionally post-Gate-C work. Reviewer must assess and state explicitly:

- whether accepting new post-Gate-C source preserves the historical Gate C claim as a claim about B/R/T;
- whether any document or automated check incorrectly assumes current main must remain equal to historical R/T forever;
- whether a new baseline / evidence record is required merely because current Reference evolves, or only when a later gate / target claim needs one;
- whether updating `reference/world.json` with the accepted source could be misread as rewriting historical Gate C Evidence, and what documentation wording prevents that.

### 8.3 Acceptance boundary

Verify that:

- the exact S4 source identity remains traceable through typed operations;
- only the three declared files are authored;
- changed collections remain exactly `[applications, semantics]`;
- shared Customer / Contract source is referenced, not copied;
- origin `order_entry` Pattern remains provenance only;
- Environment / Permission / Provider authority is not silently created;
- the 3 unresolved Preview items are accurately scoped and not mislabeled as success.

## 9. Files to inspect

Primary:

- `plan/SECOND_APPLICATION_S4_FULL_VALIDATION_PREFLIGHT_2026-09.md`
- `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_REVIEW_PACKET_2026-09.md`
- `plan/SECOND_APPLICATION_S5_EXECUTABLE_PRESSURE_IMPLEMENTATION_PACKET_2026-09.md`
- `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_IMPLEMENTATION_PACKET_2026-09.md`
- `plan/issue/000040-semantic-component-composition.md`

Implementation:

- `tools/reference-business-procedure-check/second_application_s3_destination_decision_pressure_test.go`
- `tools/reference-business-procedure-check/second_application_s4_composition_candidate_pressure_test.go`
- `tools/reference-business-procedure-check/second_application_s4_full_validation_pressure_test.go`
- `tools/reference-business-procedure-check/second_application_s5_shared_acceptance_review_pressure_test.go`
- `tools/reference-shared-acceptance/`
- `tools/reference-golden/`
- `internal/workspace/loader_test.go`

Historical gate scope:

- `plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/GATE_C_READINESS_AUDIT.md`

## 10. Review output record

After a different generative AI completes the review, record in ISSUE-000040:

```text
reviewer/model:
review date:
review result:
proposal fingerprint reviewed:
findings:
required pre-Acceptance fixes:
retained unresolved with rationale:
golden update judgment:
Gate B/C historical-scope judgment:
```

Only after that review and any required fixes does the human Acceptance decision become eligible to be recorded. The review result itself is not the human Acceptance witness.
