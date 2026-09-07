# Business Procedure A91–A99 Pressure Delta — 2026-09

Status: **Lane C / ISSUE-000042 inventory delta / not a separate readiness authority**

## 1. 目的

`plan/issue/000042-business-procedure-cross-view-reference.md`のA-series matrixはA90まで整理されている。

その後に追加したShared Acceptance / migration / Handoff realistic-pressureを、次回matrix統合時に取りこぼさないようA91–A99候補として整理する。

ADR readinessの正本は引き続き:

`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`

であり、本書からReady判定を行わない。

## 2. Delta inventory

| ID | Pressure | Current handling |
| --- | --- | --- |
| A91 | inline Procedure複数source + standalone Handoff sourceをone normalized Business semantic revisionへ束ねる | test-only implementation |
| A92 | Business source freshnessをsemantic revisionから分離し、logical source key + content stateでstaleを検出する | test-only implementation / production token未採用 |
| A93 | Business sourceがfreshでもcross-view Conformance baseが変わればPreviewをstaleとする | test-only implementation |
| A94 | Procedure Step rename + Handoff endpoint renameは片側だけならfail closed、coordinated change-setならone new semantic revisionになる | test-only implementation |
| A95 | migration previewでlegacy sourceを読めるが、legacy + candidate simultaneous authorityをrejectし、one cutover後candidate-only reloadへ進む | test-only implementation / production cutover未採用 |
| A96 | target Procedure actor context変更をHandoff endpoint identityへ複製せず、Procedure + Step local addressで同じrelationをresolveする | test-only structural pressure |
| A97 | target Procedure triggerとHandoff conditionを別authorityとして扱い、片側変更を他方へ暗黙同期しない | test-only structural pressure |
| A98 | Watanabe external evidenceはthree-view mutual constraint / event-driven Business Flowをcorroborateするが、actor/event/document fieldをHandoffへ追加するauthorityにはしない | external corroboration / no schema expansion |
| A99 | current TEI-original Repositoryにindependent second realistic Handoffはなく、adjacency / defect / Lifecycle / Deploymentをsecond caseへ昇格しない | negative Evidence / Gate C remains unresolved |

## 3. A91 — one Business semantic revision

Relevant:

- `business_model_shared_acceptance_bundle_pressure_test.go`
- `BUSINESS_MODEL_SHARED_ACCEPTANCE_BUNDLE_PRESSURE_2026-09.md`

Meaning:

```text
Procedure A
Procedure B
standalone Handoff
  ↓
normalize
  ↓
one BusinessModel revision
```

Raw file countをsemantic authority countへしない。

## 4. A92 / A93 — freshnessを二層で見る

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

A92はBusiness source自体のstaleを扱う。

A93はBusiness sourceが変化しなくても、Preview時にvalidateしたWorld側meaningが変化すれば旧ConformanceをAccept根拠にできないことを扱う。

Relevant:

- `business_model_source_freshness_pressure_test.go`
- `business_model_conformance_base_freshness_pressure_test.go`
- `BUSINESS_MODEL_PRODUCTION_ACCEPTANCE_BOUNDARY_AUDIT_2026-09.md`

## 5. A94 / A95 — atomicityとauthority cutover

A94:

```text
Procedure rename only
  -> stale Handoff -> fail

Handoff rename only
  -> missing Step -> fail

both
  -> normalize -> new revision
```

A95:

```text
legacy authority
  ↓ read for migration preview
candidate
  ↓ non-authoritative compare
Shared Acceptance
  ↓ one cutover
candidate-only authority
```

Relevant:

- `business_authoring_authority_cutover_pressure_test.go`
- `BUSINESS_AUTHORING_AUTHORITY_CUTOVER_PRESSURE_2026-09.md`

Same meaningのold/new sourceもpermanent dual authorityにしない。

## 6. A96 / A97 — Handoff field expansionへのsubtraction pressure

A96:

```text
Procedure.actor
  != Handoff endpoint identity
```

Target actor labelが変わってもexact Procedure/Step addressでsame Handoff relationをresolveできる。

A97:

```text
Procedure.trigger
  != Handoff.condition
```

双方は関連し得るが、一方を他方のauthoritative copyにしない。

Relevant:

- `handoff_actor_trigger_independence_pressure_test.go`
- `BUSINESS_HANDOFF_ACTOR_TRIGGER_AUTHORITY_PRESSURE_2026-09.md`

Current Evidenceから:

- `fromActor`
- `toActor`
- `eventId`
- trigger/condition auto-sync

を追加しない。

## 7. A98 — external corroborationのauthority limit

Watanabe evidenceは:

```text
Data / Function / Business
  independent + mutually constraining

Business work
  social-system / procedural meaning

Business Flow
  broader event-driven overview
```

をcorroborateする。

しかしexternal conceptual modelをTEI Canonical schemaへコピーしない。

Relevant:

- `BUSINESS_HANDOFF_WATANABE_SECOND_REALISTIC_EVIDENCE_PRESSURE_2026-09.md`

## 8. A99 — second realistic case search negative Evidence

Current Repositoryでsafe promotionできる候補を監査した。

Rejected:

```text
BP-08 -> BP-09
  because adjacency only

FU-01 -> FU-02
  because adjacency only

customer edit conflict -> master-data workflow
  because intentional defect does not author such workflow

Business Lifecycle transition
  because state transition != responsibility Handoff

Deployment/runtime ownership
  because runtime lifecycle != Business Procedure

Watanabe pattern
  because external pressure != TEI-original authority
```

Relevant:

- `BUSINESS_HANDOFF_TEI_ORIGINAL_SECOND_CASE_SEARCH_2026-09.md`

Current result:

```text
second realistic Handoff
  = unresolved

new Handoff field insufficiency
  = not discovered
```

## 9. PAS-07 / E7 implication

A91–A97はPAS-07のE7-10 / PAS-01 Shared Acceptance cross-checkへ関連するが、Lane CからLane B Evidence contractを第二正本として更新しない。

Lane B再監査時に少なくとも:

```text
source freshness
Conformance-base freshness
multi-source atomicity
authority cutover
candidate-only reload
```

をE7-10 supporting Evidenceとして再評価できるようにする。

A96–A99はP25-4 / P26-4 / Canonical expansion suppressionのsupporting pressureになる。

## 10. Actual runnerとの関係

A91–A97のGo testsはRepositoryに存在するが、HAT47X actual runner / GitHub Actions Evidenceがまだない。

したがって:

```text
implementation exists
  != actual green Evidence
```

を維持する。

A98 / A99は設計・negative Evidenceであり、runner PASSへ読み替えない。

## 11. Current decision

```text
A91-A99 delta
  = documented

ISSUE-000042 A-series integration
  = next inventory-maintenance step

Pre-ADR status
  = Not Ready

new Core schema from these pressures
  = none
```
