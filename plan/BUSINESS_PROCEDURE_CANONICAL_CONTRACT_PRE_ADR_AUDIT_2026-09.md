# Business Procedure Canonical Contract Pre-ADR Audit — 2026-09

Status: **Lane C / sole Contract ADR readiness authority / Not Ready / no production Core adoption**

## 1. この文書の責務

この文書はBusiness Procedure / Handoffの**Contract ADRを起こしてよいかだけを判定する正本**です。

Production adoptionの完了条件やPAS-01 / Studio / AI end-to-endの完了判定は所有しません。

詳細Evidenceは次へ委譲します。

- `plan/issue/000042-business-procedure-cross-view-reference.md` — A-series Pressure inventory
- `plan/BUSINESS_MODEL_CURRENT_MINIMUM_CANDIDATE_2026-09.md` — current minimum比較基線
- `plan/BUSINESS_AUTHORING_PRODUCTION_ADOPTION_EVIDENCE_CONTRACT_2026-09.md` — Contract ADR / production adoption Evidence contract
- `plan/BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md` — readiness段階とGate C名称衝突の整理
- `plan/BUSINESS_MODEL_SHARED_ACCEPTANCE_BUNDLE_PRESSURE_2026-09.md` — multi-source Acceptance / freshness / atomicity pressure
- `plan/BUSINESS_MODEL_PRODUCTION_ACCEPTANCE_BOUNDARY_AUDIT_2026-09.md` — source / semantic / Conformance base境界
- `plan/BUSINESS_AUTHORING_AUTHORITY_CUTOVER_PRESSURE_2026-09.md` — legacy -> candidate authority cutover pressure
- `plan/BUSINESS_AUTHORING_PRODUCTION_LOADER_VERSIONING_OPTION_AUDIT_2026-09.md` — migration adapter / candidate-only loader option audit
- `plan/BUSINESS_AUTHORING_WORKSPACE_DISCOVERY_BOUNDARY_PRESSURE_2026-09.md` — workspace / Source Index discovery pressure
- `plan/BUSINESS_AUTHORING_EXISTING_ACCEPTANCE_IMPLEMENTATION_REUSE_AUDIT_2026-09.md` — existing primitive / reuse / non-reuse audit
- `plan/BUSINESS_MODEL_PHYSICAL_BASE_REVISION_OPTION_AUDIT_2026-09.md` — complete stale-envelope option audit
- `plan/BUSINESS_HANDOFF_WATANABE_SECOND_REALISTIC_EVIDENCE_PRESSURE_2026-09.md` — 渡辺幸三氏のexternal corroboration
- `plan/BUSINESS_HANDOFF_TEI_ORIGINAL_SECOND_CASE_SEARCH_2026-09.md` — second realistic Handoff候補監査
- 各`BUSINESS_PROCEDURE_*` / `BUSINESS_ACTION_*` Pressure文書
- X-TEA Evidence Register / E-015 / E-016

### 1.1 Gate Cの名称を混同しない

本書のPre-ADR Gate A〜Gは、このContract ADRだけのlocal audit gateです。

特に:

```text
Lane C Pre-ADR Gate C
  = Procedure-scoped Step address / Handoff contract audit

Product Gate C
  = GATE_C_EVIDENCE_CONTRACT.md が定義する第一Web Target gate
```

であり別物です。

`ISSUE-000027` / `ISSUE-000046`が`Gate C Ready`待ちとしているのは**Product Gate C**です。本書のPre-ADR Gate Cではありません。

したがってBusiness Procedure Contract ADRが、Product Gate C後に実装するStudio / AI Shared Acceptance end-to-endの完成を前提にする循環依存は作りません。

## 2. Current minimum candidate

```text
BusinessModel
  Procedures[]
    Procedure
      id
      purpose
      trigger { epistemicStatus, meaning }
      actor   { epistemicStatus, businessLabel }
      Steps[]
        localKey
        Body
          businessAction
          applicationPatterns[]
          validations[]
          capabilities[]
        protectedApplicationPattern?

  Handoffs[]
    from { procedure, step }
    to   { procedure, step }
    condition
```

Current interpretation:

```text
D4 + singular narrow O1 + S4-A + H1/R3
```

- D3はhistorical/subtraction Evidenceでありpreferred minimumではない。
- D4単独はBP-06のexact protected targetを失うためlossy。
- direct Data / Relation refsはcurrent minimum必要性未証明。既存refsからderived viewを作る。
- HandoffはProcedure bodyから復元できないset-level Business meaning。
- authoring / discovery / freshness / cutover Pressureはこのsemantic candidateを拡張せず、実装境界だけを狭める。

## 3. Current authoring candidates

### 3.1 Procedure / O1

Current strongest candidate:

```text
same Business Procedure source
  Step.protectedApplicationPattern?  // zero-or-one
```

Test-only Pressureでは:

- strict decode
- round-trip
- zero-or-one
- multi-target fail closed
- generic role field fail closed
- prior sidecarとsame normalized D4+O1 meaning

まで進んでいます。

Production sidecarは一つのStep-local relationのためにcomposite source machineryを要求するため第一候補から下げます。

Migration direction:

```text
legacy O1 sidecar
  -> migration input / Reference Evidence

inline singular O1
  -> accepted candidate authority after cutover
```

Migration previewでlegacy sidecarを読むことは許可しますが、inline O1とlegacy sidecarを同時authorityとして有効化しません。

### 3.2 Handoff

Current strongest candidate:

```text
standalone Handoff source
  Handoffs[]
    from { procedure, step }
    to   { procedure, step }
    condition
```

Test-only Pressureでは:

- current Reference Setとsame normalized Handoff meaning
- Procedure artifact locator非依存
- stale endpoint fail closed
- Workflow field reject

まで進んでいます。

Migration direction:

```text
Reference Set Handoff
  -> fixture assembly / migration input / Reference Evidence

standalone Handoff source
  -> accepted candidate authority after cutover
```

両者をpermanent dual-read authorityにしません。

## 4. Authority boundaries

### O1

```text
BP-06
  supporting authority -> security.authorize
  protected target     -> order_entry.submit_order
  direct realization  -> none
```

Authorization policyはcorroboration-onlyでありBusiness targetをauthor / backfillしません。
Generic `{Kind, Role, Target}` taxonomyは未採用です。

### Handoff

```text
explicit Handoff source -> normalized relation
Step adjacency           -X-> Handoff
Observation              -X-> Handoff
Function sequence        -X-> Handoff
```

H1/R3はhandoff-onlyです。Business Flow / DFD / Workflow runtimeへ拡張しません。

```text
Procedure.actor
  != Handoff endpoint identity

Procedure.trigger
  != Handoff.condition
```

も維持します。Current Evidenceから`fromActor` / `toActor` / `eventId` / document / payload等を追加しません。

### Migration authority

```text
legacy sourceをmigration previewで読む
  -> allowed

candidate objectが存在する
  -> non-authoritative previewならallowed

legacy + candidateを同時にaccepted authorityとして読む
  -> reject
```

Same meaningでも二重正本とはみなしません。競合時にauto-merge / last-writer-winsしません。

### Step address

Global Canonical Step IDではなくProcedure-scoped `localKey`を第一候補とします。

## 5. Placement

Current strongest placement candidate:

```text
Business authoring sources
  -> normalized Business model
  -> Business semantic revision

TEI workspace sources
  -> ir.World

normalized Business model
  <-> exact cross-view validation <-> ir.World
```

Current source-level Pressureでは:

- Business wording change -> Business semantic revision changes / `ir.World` unchanged
- protected-target-only change -> Business semantic revision changes / `ir.World` unchanged
- Handoff condition-only change -> Business semantic revision changes / `ir.World` unchanged
- stale Application target -> separate Business planeでもfail closed

が成立します。

したがって:

```text
separate normalized Business plane -> strongest current candidate
ir.World.Business                  -> current Evidenceでは不要
production internal/business       -> deferred
```

です。

**`ir.World.Business`が必要だというEvidenceがないことはADR blockerではありません。** これはseparate planeを維持し、embeddingをdeferするnegative Evidenceです。Concrete insufficiencyが出た場合だけplacementをreopenします。

## 6. Multi-source / freshness / implementation boundary

Current test-only `BusinessAuthoringBundle`は:

```text
inline Procedure source A
inline Procedure source B
...
standalone Handoff source
        ↓
normalize
        ↓
one BusinessModel
```

をPressureします。

Source-level Evidenceでは:

- current normalized Business meaningと同値
- Procedure wording / O1 target / Handoff conditionのいずれでもsame Business semantic revisionが変化
- Procedure source orderingだけではsemantic no-op
- semantic bundleへpath / artifact / runtime metadataを持ち込まない

まで進んでいます。

### Freshness

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

を固定します。

Test-only PressureではBusiness source changeとWorld / Conformance base changeを独立にstale rejectします。

One opaque `baseModelRevision`でも、Business source setとWorld / Conformance baseをfalse-negativeなくcoverするcomplete stale envelopeなら許容します。

```text
false-negative stale -> reject
false-positive stale -> conservative tradeoffとして許容
```

Git HEAD equalityだけはdirty / untracked authoritative sourceを見逃し得るため無条件なfreshness proofにしません。Source add/remove・authoritative source-root membership changeもfreshness envelope対象です。

Production hash / ETag / token shapeはLane C Contract ADRで固定しません。

### Discovery / reload

Current strongest direction:

```text
migration adapter
  -> legacy authorityをpreview inputとしてread-onlyに読む

candidate Procedure + standalone Handoff
  -> non-authoritative staging

accepted authority cutover

steady-state discovery / loader
  -> candidate-only
```

```text
file exists
  != authoritative source reachable

schemaVersion
  != source authority priority
```

です。

Current test-only Pressureはcandidate sourceをdiskへserializeし、legacy inputを読まずsame processでstrict reload / normalize / Conformanceします。

ただし:

```text
disk reopen in same Go test process
  != fresh-process feasibility Evidence
  != production steady-state loader
```

です。

### Existing implementation reuse

再利用候補:

```text
internal/workspace
  -> source-root discovery / ManagedFiles / reload

Source Index
  -> physical source fingerprint / stale validation

Workbench / Consumer Impact / Release Graph
  -> deterministic Preview / impact projection

Deployment expected-revision + bound fingerprints
  -> stale-reject architectural precedent
```

ただし:

```text
source discovery primitive
  != candidate authority

fingerprint primitive
  != complete Shared Acceptance revision

review projection
  != authoritative apply

Deployment Run.Revision
  != Business semantic revision

Deployment durable state machine
  != Business Workflow / source transaction
```

です。

Current audited surfacesではgeneric source-authoring Shared Acceptance / atomic writer / authority-discovery cutover / production Business loaderは確立されていません。

これは**production implementation gap**であり、Business schema拡張Evidenceではありません。

## 7. Derived views / mutation guardrail

Business sourceへData / Relation / CRUD / ownership / Namespaceを重複authorしません。

### Data

```text
no trace != no dependency
trace exists != complete dependency set
```

### CRUD

Current exact operation Evidenceで安全な主要cell:

```text
BP-00 -> R sales.order
BP-02 -> R customer.customer
BP-03 -> R sales.order_line
```

C/U/Dはcurrent TEI-original Referenceでqualifying caseなし。

禁止:

- wordingからCRUDを推測
- `record` / `command` kindからCRUDを推測
- Database Contract WritesをAction Uへ自動帰属
- Capability payload / Custom Go / Lifecycle transitionをmutation contractへ読み替える

X-TEA E-016はFunction/Table I/Oのexplicit `OpC/R/U/D`からmatrixをderiveしていたことを示します。

**qualifying C/U/D caseがないこと自体はBusiness Procedure Contract ADR blockerではありません。** Current ADRで必要なのは、authority不足をunknownのまま保持して誤推測しないことです。

C/U/D caseは将来generic mutation/effect contractをreopenするtriggerとして残します。

## 8. Coreへ今入れないもの

- production `BusinessProcedure` / `BusinessHandoff` kind
- `ir.World.Business`
- production `internal/business`
- generic SemanticRef / role enum / Core Edge
- full Business Flow / DFD
- Workflow state / retry / timer / queue / saga
- global Canonical Step ID
- Observation専用Canonical identity
- Business Procedure CRUD field
- Application Pattern generic CRUD field
- Semantic Capability generic Effect
- Database Contract C/U/D enum
- mutation binding / MutationIntent / Function-Data-I/O Canonical kind
- synthetic second-corpus Data ownership
- Business専用generic transaction / revision subsystem
- old/new Business authoring sourceのpermanent dual-read / merge-priority contract
- Git SHAのplatform-wide `baseModelRevision`固定
- schemaVersionによるauthority priority
- mandatory `businessAuthoringMode`
- Workbench / impact projectionのauthoritative source化
- Deployment durable state machineのBusiness authoring / Workflowへの流用

## 9. Contract ADR readiness gates

以下のGate A〜Gは**Lane C Pre-ADR local gates**です。Product Gate A/B/Cとは別です。

### Pre-ADR Gate A — actual execution

Actual Repository environmentで同一revisionに対し:

```text
read-only gofmt check
go vet ./...
go test ./...
focused Lane C tests
reference checker baseline
intentional defect expected diagnostic
```

がgreenであること。

`BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md`を使用し、source-level inspectionをactual greenへ読み替えません。

### Pre-ADR Gate B — minimum meaning

Actual testsでmanual/external Action、uncertainty、multi-Application、D4 ref kind separation、future Data/Relation fail-closed等を確認する。

### Pre-ADR Gate C — address / Handoff

- Procedure-scoped localKey
- stale endpoint fail closed
- explicit Handoff authority only
- standalone Handoff candidate = current normalized meaning
- Handoff-only boundary維持
- Procedure actorをHandoff endpoint identityへ複製しない
- Procedure triggerとHandoff conditionを別authorityとして扱う
- migration previewでReference Set Handoffをinput / Evidenceとして読める
- legacy + candidate simultaneous authorityをrejectする
- candidate-only reload / normalizeのfeasibilityをactual確認
- second realistic **TEI-original** Handoffで`from/to/condition` sufficiency確認

Current Repository監査ではBP adjacency、FU adjacency、intentional defect、Lifecycle、Deploymentはいずれもsecond realistic Handoffへ昇格できません。外部Watanabe patternもTEI-original authorityではないためsecond caseは未充足です。

### Pre-ADR Gate D — O1

- D4-only lossをactual確認
- singular narrow O1 lossless
- inline candidate strict decode / round-trip / sidecar-equivalent normalization
- Authorization policy corroboration-only
- migration previewでlegacy sidecarをinput / Evidenceとして読める
- legacy sidecar + inline O1 simultaneous authorityをrejectする
- same-step conflicting targetをauto-mergeしない
- candidate-only reload / normalizeのfeasibilityをactual確認
- multi-targetを先取りしない

### Pre-ADR Gate E — placement / freshness / cutover feasibility

Contract ADR前に必要なのはproduction Shared Acceptance完成ではなく、contract invariantsのactual / executable feasibility Evidenceです。

必要:

1. multi-source bundle / source freshness / Conformance-base freshness / coordinated rename / authority conflict / workspace discovery / candidate disk reloadをactual greenにする。
2. Business source base + World / Conformance baseをfalse-negativeなく固定できるcomplete stale-envelope **requirement**が成立することを示す。Token shapeは固定しない。
3. candidate-only sourceをlegacy migration inputなしに**fresh process / separate executable boundary**でloadし、same expected Business semantic revision + current World Conformanceを再現できるfeasibility Evidenceを得る。
4. separate Business planeでactual cross-view Conformanceをlossless / fail-closedにできることを確認する。
5. legacy + candidate simultaneous authorityをrejectできることをactual確認する。

Contract ADR前に不要:

- production generic/shared Acceptance implementation完成
- production multi-file writer完成
- final repository transaction API
- final hash / ETag / revision-token schema
- Studio / AI Shared Acceptance end-to-end

これらが未実装であることだけではContract ADRを止めません。

ただしfeasibility Evidenceで、candidate-only reloadやstale detection、authority cutover semanticsが成立しないことが判明した場合はcandidateをreopenします。

### Pre-ADR Gate F — derived projections / mutation guardrail

- Data / CRUD projection actual green
- no-traceをno-dependencyへ丸めない
- authority不足のC/U/Dをunknownのまま保持する
- Pattern kind / Database write / LifecycleからCRUDを推測しない
- future operation modelでdirect/effective distinctionを失わない

**qualifying C/U/D caseそのものはGate F Required Evidenceではありません。**

### Pre-ADR Gate G — adjacent domain

- multi-Application / social-system realism actual green
- physical Command/Observation actual green
- exact identitiesなしにsecond-corpus ownershipをsynthetic化しない
- Subsystem / SubjectAreaはderived lens不足のconcrete caseまでreopenしない

## 10. Production adoption readiness — Contract ADR後の別段階

Contract ADRがReadyになってもproduction authorityとして即採用しません。

Production adoption前には追加で:

```text
Business-specific production authoring adapter
  -> workspace / Source Index reuse
  -> strict production decode / normalize

Shared Acceptance / repository boundary
  -> complete production stale envelope
  -> Validation / Conformance
  -> atomic candidate source apply
  -> atomic authority-discovery cutover

fresh production process
  -> candidate-only steady-state load
  -> legacy migration adapter unreachable
  -> same accepted Business semantic revision
  -> current World Conformance
```

をactualに実証します。

さらに失敗時にpartial source write / partial authority switchを残さないことを要求します。

この段階は`BUSINESS_AUTHORING_PRODUCTION_ADOPTION_EVIDENCE_CONTRACT_2026-09.md`が詳細を所有します。

`ISSUE-000027` / `ISSUE-000046`のStudio / AI shared Acceptance end-to-endはProduct Gate C後のworkです。Lane C Contract ADRはintegration obligationだけを保持し、その完成を先取りしません。

## 11. Current readiness

```text
minimum semantic candidate
  = D4 + singular narrow O1 + S4-A + H1/R3

Procedure authoring candidate
  = inline singular protectedApplicationPattern?

Handoff authoring candidate
  = standalone set-level Handoff source

placement candidate
  = separate normalized Business semantic plane

migration architecture
  = legacy-read migration adapter
    -> one accepted authority cutover
    -> candidate-only steady-state loader

workspace/source discovery
  = existing source-root + Source Index reuse strongest

shared Acceptance implementation
  = integration principleあり
  = production generic source-authoring substrateはcurrent auditで未確立

qualifying C/U/D case
  = absent, but not Contract ADR blocker

ir.World.Business necessity Evidence
  = absent, supports current separate-plane direction
```

### Contract ADR blockers

1. **Actual Repository runner green未取得。** HAT47X offline / current PR headにCI Evidenceなし。
2. **Candidate-only fresh-process feasibility actual未取得。** Same-process disk reload Pressureはあるがfresh process Evidenceではない。
3. **Second realistic TEI-original Handoff未充足。** `from/to/condition` sufficiencyを独立realistic caseでまだ確認できない。

したがって:

```text
Contract ADR
  = Not Ready
```

を維持します。

### Production adoption blockers

Contract ADR blockersに加えて:

- production Business-specific adapter未実装
- production complete stale envelope未実装
- atomic source apply / authority-discovery cutover未実証
- production fresh-process candidate-only reload未実証
- partial failure safety未実証

が残っています。

したがって:

```text
Production adoption
  = Not Ready
```

です。

### Blockerではないもの

- qualifying C/U/D caseがないこと
- `ir.World.Business`必要性Evidenceがないこと
- final revision-token shapeが未決定であること
- Product Gate C後のStudio / AI Shared Acceptance end-to-endが未実装であること

これらをContract ADR Not Ready理由へ混ぜません。
