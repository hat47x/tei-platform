# Second Application Reference — S8 Successor Handover B2 / R2 Preflight

- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Parent contract: `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- Historical baseline: `B / R=e5231839a5dcc62cce1181a2731fcb7c70d16736`
- Current S8a branchless candidate: `decdb5c16d403e72e8ce12dcd62698d29b3ae40f`
- Status: **preflight only; B2 / R2 not yet issued**
- R2 assignment condition: S8a Driver source/evidence integrated, repository health PASS, clean stable revision

## 1. Why a successor Handover is required

Historical `B / R` predates `repair_intake`.

Therefore a Target run whose manifest says:

```text
Application = repair_intake
baselineRepositoryRevision = e5231839...
```

would be false provenance.

The existing Handover contract already defines the response to changed / expanded Application meaning:

```text
new authoritative meaning / Driver scenario
  -> new clean revision R2
  -> re-run required Gate B Evidence at R2
  -> audit exact same-run artifacts
  -> issue successor Handover B2 / R2
  -> keep Target build identity separate
```

This packet instantiates that rule for the accepted Second Application.

## 2. Do not assign R2 yet

Current S8a Driver code is still a branchless candidate and has no repository-health Evidence.

Therefore:

```text
B2 = not issued
R2 = not assigned
```

Do not use a moving `main` SHA as R2 merely because it contains S5/S7 source.

R2 is the clean stable revision after:

- S8a Driver pressure is integrated
- focused S8 Driver tests PASS
- knowledge-portability evidence runner PASS
- repository health check PASS
- no competing source drift remains

If source fixes are required during Evidence acquisition, commit them and restart the cycle at the new clean revision.

## 3. R2 content requirements

R2 must contain, at minimum:

### accepted meaning

```text
application repair_intake
semantic repair_intake.intake
relation repair_intake.intake.customer -> customer.customer
pattern repair_intake.customer_reference
datasource customer_master -> master_data / read
contract master_data.customer ^2
query find_by_id
```

### accepted Environment binding

```text
development:
  repair_intake.customer_master -> master_data.dev

production:
  repair_intake.customer_master -> master_data.prod
```

### S8a deterministic Driver pressure

development must demonstrate through the ordinary Driver path:

```text
repair_intake.intake.customer_id
  -> Relation key
  -> master_data.customer/find_by_id input id
  -> development / master_data.dev
  -> explicit Contract Query executor binding
  -> exact instance-aware runtime registry
  -> synthetic Customer Semantic result
```

and fail closed for:

- wrong Semantic source key
- missing exact runtime registry
- wrong runtime provider
- production executor-binding absence
- zero-or-one cardinality violation

## 4. Required pre-baseline qualification

At the eventual candidate R2:

```bash
go test -count=1 -v ./tools/reference-business-procedure-check \
  -run '^TestSecondApplicationS8'

tools/run-knowledge-portability-reference-evidence.sh

tools/repository-health-check.sh
```

All must PASS before R2 is eligible.

This is separate from the Gate B three-lane requalification below.

## 5. Re-run Gate B Required Evidence at exact R2

The historical B record used three Required Evidence lanes:

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility
3. #308 real PostgreSQL Provider + ephemeral fixture

For B2, do not carry their old success forward to the new revision.

Use the existing final convergence runner at the exact clean R2.

Because R2 introduces the Second Application Driver scenario, use the broad deterministic mode so repository-wide tests participate in #310:

```bash
export GOENV=off
export GOWORK=off
export GOFLAGS=

TEI_TEST_POSTGRES_DSN_FILE='/secure/path/to/dsn' \
go run ./tools/reference-gate-b-final-evidence \
  -broad \
  -artifact-dir /tmp/tei-gate-b-R2/children \
  > /tmp/tei-gate-b-R2/final.json
```

Required final properties include:

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R2
reproducibleRevision = true
allLanesSameRevision = true
childArtifactsRetained = true
postRunRepository.revisionStable = true
```

The three child artifacts must all report the exact same R2.

## 6. Independent same-run artifact audit

Using the exact child directory retained by the final convergence run:

```bash
go run ./tools/reference-gate-b-readiness-artifact-audit \
  -final /tmp/tei-gate-b-R2/final.json \
  -artifact-dir /tmp/tei-gate-b-R2/children \
  > /tmp/tei-gate-b-R2/readiness-artifact-audit.json
```

Require:

```text
evidenceNature = gate-b-readiness-artifact-audit
artifactAuditPassed = true
repositoryRevision = R2
```

A later same-R2 rerun is not a substitute for the child artifacts the final convergence actually read.

## 7. Readiness re-audit

After §§5–6 succeed, re-apply the Gate B Readiness Audit to R2.

At minimum verify:

- #310 / #255 / #308 exact same-run Evidence
- final convergence = R2
- artifact audit = R2
- source tree remained clean and revision-stable
- Second Application accepted meaning is present
- S7 binding is present
- S8a deterministic Driver pressure passes
- actual Oracle remains explicitly unresolved

Do not issue B2 from source presence or unit-test success alone.

## 8. What #308 means for B2

#308 remains:

```text
real PostgreSQL Provider + tool-owned ephemeral fixture Evidence
```

It demonstrates that the provider/runtime architecture works against a real engine under the established safety boundary.

It does **not** demonstrate:

- Oracle compatibility
- repair_intake actual external read
- Oracle principal / grants
- master_data.prod connectivity
- production runtime authority

Therefore B2 keeps actual Oracle items in `unresolved`.

## 9. Additional Second Application Driver Evidence

B2 should reference, in addition to the three Required Gate B lanes:

- exact R2
- S8a focused test result
- `tools/run-knowledge-portability-reference-evidence.sh` result
- `tools/reference-business-procedure-check/second_application_s8_driver_pressure_test.go`
- accepted S5 Decision record
- accepted S7 Decision record
- current `repair_intake` source/index identities

These are supplemental Handover references for the new Application/Driver scenario.

They do not replace #310 / #255 / #308.

## 10. B2 record contents

When all required Evidence is green, create a new record rather than editing historical B in place.

Expected identity:

```text
Handover Baseline identity = B2
baselineRepositoryRevision = R2
```

The record should enumerate at least:

### order_entry

Historical first-Application Canonical / Driver identities still present at R2.

### repair_intake

```text
Application = repair_intake
Semantic Data = repair_intake.intake
Relation = repair_intake.intake.customer
Pattern = repair_intake.customer_reference
Datasource = customer_master
Database = master_data
Contract Query = master_data.customer/find_by_id
development instance = master_data.dev
production instance = master_data.prod
```

The Handover record is a manifest/reference bundle, not a second authority source.

## 11. Unresolved carried by B2

Unless separately closed by accepted Evidence:

```text
RUNTIME_AUTHORITY_NOT_GRANTED
actual Oracle principal / grants unresolved
credential / network authority unresolved
production Contract Query executor binding absent
actual Oracle read not acquired
First Web Target repair_intake interaction not yet evaluated
```

Do not weaken these to make B2 issueable.

## 12. Relationship to historical B / R / T

Historical:

```text
B / R=e5231839...
First Web Target T=0ba87a4e...
```

remain immutable historical Evidence for the original Gate C run.

B2 does not retroactively alter them.

After B2 is issued:

```text
B2 / R2
  -> Second Application First Web Target work
       target build identity = T2
```

T2 must be tracked separately from R2.

Do not label the old T as Evidence for `repair_intake`.

## 13. Failure / restart rules

Restart with a new candidate R2 if:

- source changes during qualification
- generated artifact repair is needed
- any Required Evidence lane fails
- final convergence or artifact audit fails
- repository becomes dirty
- revision moves during the run
- S8a Driver pressure fails

Do not patch the B2 document around failed Evidence.

## 14. Exit

This preflight exits only when:

- S8a is integrated and health-clean
- exact R2 is fixed
- #310 / #255 / #308 same-run Evidence PASS at R2
- artifact audit PASS at R2
- Gate B Readiness re-audit supports R2
- supplemental Second Application Driver Evidence is recorded
- unresolved actual Oracle authority remains explicit

Only then issue `B2 / R2` and begin S8b First Web Target Evidence against that baseline.
