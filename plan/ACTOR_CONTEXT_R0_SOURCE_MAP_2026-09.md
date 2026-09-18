# Actor Context R0 Source Map — current TEI assetsからの静的導出

- Status: ISSUE-000053 / R0 implementation map / test-only first
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md`
- Production schema: None
- Product Gate impact: None

## 1. 目的

R0では、同一のshared accepted / reference baselineから次の三つを静的に導出する。

1. Builder Re-entry Context
2. Human Review Projection
3. read-only Operational Projection

目的は新しいAI層を作ることではない。

> **current TEIがすでに持つauthority / dependency / evidenceを、actorごとにどこまで再構成できるかを測る。**

## 2. 共通の入力基線

初回は`order_entry`を使う。

### TEI World

- `reference/workspace`
- `workspace.Load`で得るnormalized `ir.World`
- `internal/index.Build`で得られるderived dependency / binding index

### Business Reference

- `reference/business-system-design/order-entry-business-procedure.json`
- current statusは`reference-authoring-fixture` / `canonical=false`
- Business Procedure production authorityではない

### Source / freshness

- `internal/sourceindex`
- artifact path
- content fingerprint
- subject reference
- Business Procedure Reference fingerprint
- Projectionごとのactual selected input fingerprint集合

R0では一つのBusiness Procedure fingerprintを三Projectionで共有するが、それだけをProjection全体のfreshness authorityにはしない。Builder / Operationalは選択したworkspace subjectのsource artifactにも依存し、Datasource / accepted accessをProjectionする場合はその値を所有するApplication sourceにも依存する。Human Reviewはさらにconflict Evidence inputへ依存する。

```text
same Business Reference baseline
  + purpose-specific selected inputs
  -> projection freshness dependencies
```

無関係なworkspace source変更を理由に全Projectionをstale化せず、逆にactual dependency変更をBusiness Procedure fingerprintだけで見逃さない。

### 既存test-only cross-view Evidence

- Business Action → Pattern / Data / Relation
- Validation → Data / Relation
- Capability payload → Data
- Authorization target corroboration
- Observation boundary
- access conflict
- cross-view proof
- Source Index path / fingerprint separation

R0はこれらの既存authority directionを壊さない。

## 3. Builder Re-entry Contextのsource map

| Needed context | Current source | R0扱い |
|---|---|---|
| relevant Canonical subject | `ir.World` / `internal/index.Derived.SemanticElements` | reuse |
| dependency / impact | SemanticDependencies / DeclaredDependencies / Consumer Impact | reuse |
| authoritative source locator | Source Index | reuse |
| Application Pattern meaning | `ir.World.Applications[].Patterns` | reuse |
| Capability contract | Semantic Capability | reuse |
| Binding / native implementation | CustomGoBindings / Plugin / Environment resolution | reuse |
| DB / provider constraint | ApplicationDatabases / EnvironmentBindings / Database Contract | reuse |
| unresolved cross-view concern | Design Synthesis proof / diagnostics | reuse as derived |
| accepted Decision / Variation | current accepted source / ADR等へのtrace | current機構を利用、万能Decision DBは作らない |
| native asset touchpoint | Source Index implementation role / Binding | partial |
| generic semantic diff | current surfacesを再利用してPressure。production universal diff schemaは未確定 | gap / derive |

### 既存DesignContextPacketの扱い

Design Synthesisではすでに、

- accepted snapshot
- exact facts
- candidate-local proof
- contradiction
- unresolved
- authority limits

をbounded contextへ渡す設計がある。

これをBuilder Re-entryの**design synthesis case**として再利用する。

Migration / provider change / defect correction等へ同じpacket schemaを強制しない。

## 4. Human Review Projectionのsource map

| Review concern | Current source | R0扱い |
|---|---|---|
| changed meaning | Proposal / candidate diff / source delta | derive |
| impacted subjects | dependency / Consumer Impact | reuse |
| unresolved | diagnostics / proof result | reuse |
| permission / resource change | Environment / Permission / Resource index | reuse |
| native code touchpoint | Binding / Source Index implementation refs | partial |
| Validation / Conformance | existing validators / Reference outputs | reuse |
| provenance | Source Index / Evidence refs | reuse |
| drill-down | Canonical subject → Source Index refs | reuse |
| one high-density review packet | production shapeなし | R0 test-only |

Human Review Projectionで重要なのは、AI生成文章の読みやすさより、

```text
review statement
  -> exact subject
  -> exact source / evidence
```

へ戻れることである。

## 5. Operational Projectionのsource map

### 5.1 Business Action

current source:

- Business Procedure Reference fixture
- `businessAction`
- `realization`
- Application Pattern / Validation / Capability refs

現時点ではReference authorityでありproduction Canonicalではない。

### 5.2 Action → Data / Relation

既存test:

`tools/reference-business-procedure-check/business_action_data_projection_pressure_test.go`

すでに、

- exact Pattern ref
- Pattern Data / Relation / Capability
- Validation owner Data
- Capability payload Data

からBusiness Actionに関係するData / Relationをderivedに投影できる。

重要なguard:

```text
no trace
  != no data
```

BP-04 / BP-08 / BP-09は`NoDataTraceFromCurrentRefs`としてunresolvedを保持する。

これはAgent contextにもそのまま必要である。

### 5.3 Authorization target

既存test:

`authorization_target_authority_direction_pressure_test.go`

current Evidenceでは、

- Business protected target candidate
- legacy Authorization policy corroboration

を分離している。

```text
authorization policy
  may corroborate target
  but may not author / backfill target
```

したがってAgent Operational Projectionも、

- protected target meaning
- authorization policy result

を別authorityとして表示する。

### 5.4 Binding / Permission / Resource

`internal/index.Derived`には少なくとも、

- CustomGoBindings
- EnvironmentPluginResolution
- PermissionGrants
- ResourceBindings
- LeasePolicies
- EnvironmentBindings

が存在する。

これはOperational resolutionに利用できる。

ただしderived index entryの存在をactual Provider availabilityやsecret取得成功へ読み替えない。

### 5.5 Observation

既存test:

`observation_cross_view_pressure_test.go`

current BP-08はObservationのBusiness meaningを持つが、runtime read operationをfake Semantic Capabilityへ昇格させない。

したがってOperational Projectionでは、

```text
what must be observed
  != exact runtime read locator
```

を保つ。

runtime locatorはEvidence取得のrealizationであり、Business identityではない。

## 6. R0で見えているgap

### G1 — Business Procedure production authority

current `order_entry` ProcedureはReference fixtureであり`canonical=false`。

したがってR0 Operational Projectionは**Reference-only**である。

Agent production executionの根拠にはしない。

### G2 — protected targetのauthority

BP-06のprotected targetはcurrent overlay / pressure Evidenceであり、Business source上のproduction accepted fieldではない。

Agentにsubmit targetが確定済みと表示しない。

### G3 — Observation realization

Business上「実状態を確認する」は存在するが、どのSemantic Capabilityで観測するかは確定していない。

runtime read identityをSemantic identityへ昇格しない。

### G4 — generic Action eligibility

current TEIには、

```text
defined
resolved
bound
authorized
resource-ready
invokable
observed
business-complete
```

を一つのproduction evaluatorとして束ねたものはない。

R0では作らない。

各層の有無 / unresolvedを並べてPressureする。

### G5 — current actor / principal context

Permission GrantやAuthorization policyが存在しても、AI Business Actorというactor labelからprincipal / role / grantを自動推測しない。

actor identityとAuthorization Principalの接続は別Evidenceが必要。

### G6 — Human Reviewのgeneric semantic diff

個別diff / impact surfaceは存在するが、全TEI subjectを対象にした万能semantic diff contractはまだ固定しない。

R0では対象challengeに必要な差分だけをderiveする。

### G7 — 一つのsnapshot fingerprintではactual dependency freshnessにならない

R0 sourceの初版はBusiness Procedure Reference fingerprintを三Projectionへ共通保持したが、各Projectionはそれ以外にも依存する。

- Builder / Operational: selected Pattern / Data / Relation / Validation / Capabilityのworkspace source
- Builder / Operational: Datasource / accepted accessを読む場合、そのowner Application source
- Human Review: 上記に加えてcustomer-edit conflict Evidence

したがって、

```text
same reference fingerprint
  != same dependency set
  != projection freshness proven
```

である。

test-only pressureでは、shared Business Reference fingerprintを残しつつ、Source Indexからselected subjectのartifact fingerprintを集める。PatternからDatasource / accepted accessを読む経路ではPattern owner Applicationのsource fingerprintも必須とし、Human Reviewではconflict fixture fingerprintも追加する。production global revisionや万能Snapshot IDは導入しない。

## 7. R0 test-only shape

production schemaではなく、test-only構造として最小限次をPressureする。

### Builder

```text
BuilderContextProbe
  Purpose
  ReferenceFingerprint
  RequiredInputFingerprints[]
  Subjects[]
  SourceRefs[]
  Dependencies[]
  Constraints[]
  NativeTouchpoints[]
  Unresolved[]
  AuthorityLimits[]
```

### Human

```text
HumanReviewProbe
  Purpose
  ReferenceFingerprint
  RequiredInputFingerprints[]
  MeaningDelta[]
  ImpactedSubjects[]
  PermissionResourceDelta[]
  Unresolved[]
  EvidenceRefs[]
  DrillDownRefs[]
  AuthorityLimits[]
```

### Operational

```text
OperationalContextProbe
  ReferenceFingerprint
  RequiredInputFingerprints[]
  ProcedureRef
  StepRef
  BusinessAction
  Realization
  SemanticRefs[]
  PatternRefs[]
  ValidationRefs[]
  CapabilityRefs[]
  DataProjection[]
  AuthorityRequirements[]
  BindingResolution[]
  ResourceResolution[]
  ObservationObligations[]
  HandoffOrUnresolved[]
  EvidenceRefs[]
  AuthorityLimits[]
```

これらのfield名をproductionへ採用する判断ではない。

## 8. forbidden shortcuts

R0 test-only typeへ、次のような総合判定fieldを置かない。

- `CanExecute`
- `Executable`
- `Authorized`をBusiness meaningと同じ層に置くこと
- `Complete`
- `Truth`
- `Confidence`
- `AgentTrust`
- `Winner`
- `Preferred`
- `AutoApprove`

個別authorityの結果としてauthorization allowを観測すること自体は禁止しない。

禁止するのは、それを他の境界まで含む総合authorityへ短絡することである。

## 9. R0 comparison

同じ`order_entry` shared reference baselineから三Projectionを作り、purpose-specificなrequired input fingerprintsと情報内容を比較する。

| Concern | Builder | Human | Operational |
|---|---|---|---|
| source refs | high | drill-down中心 | authority/evidence中心 |
| dependency | high | impact中心 | current Action周辺 |
| code/native asset | high | touched point中心 | 通常は低 |
| semantic diff | change task中心 | primary | 通常不要 |
| Business Procedure | impact時のみ | impact時のみ | primary |
| Authorization | change impact | authority delta | execution precondition |
| Resource | implementation impact | risk delta | execution precondition |
| Observation | design impact | verification view | primary |
| unresolved | design frontier | review focus | handoff / stop condition |
| generated code | relevant only | summary / drill-down | 原則不要 |

「同じsourceだから同じcontextを渡す」設計にしない。

## 10. Test-only implementation source

最初のsourceは次へ隔離する。

`tools/reference-business-procedure-check/actor_context_r0_pressure_test.go`

このtest sourceはproduction package / schemaを追加せず、同package内の既存Reference helperを再利用する。

現在のPressure:

1. BP-02からread-only Operational Projectionを導出
2. 同じBusiness Procedure Reference fingerprintを保持しつつ、三Projectionのactual selected input fingerprint集合を分ける
3. Builder / Operationalではselected workspace source definition fingerprintへtraceする
4. Builderではselected dependency集合そのものもderived build inputとしてfingerprintする
5. Human側では既存customer-edit conflictとそのEvidence fingerprintを保持
6. Builder側ではexact dependencyをderive
7. BP-04 / BP-08をunresolvedのまま保持
8. authority shortcut fieldをreflectionで拒否

sourceの存在はtest PASSを意味しない。actual `go test`結果は別Evidenceとして取得する。

## 10. R0完了条件

- [ ] 三Projectionを同じshared reference baselineから再構築できる
- [ ] Projectionごとのactual selected input fingerprintへ逆追跡できる
- [ ] Source / Evidenceへ逆追跡できる
- [ ] Business Referenceがcanonical=falseであることを失わない
- [ ] BP-04 / BP-08 / BP-09のunresolvedを埋めない
- [ ] Authorization policyからBusiness protected targetをbackfillしない
- [ ] runtime Observation locatorをSemantic identityへ昇格しない
- [ ] Permission / Resource indexをactual runtime successへ昇格しない
- [ ] 三Projectionを削除してもaccepted meaningが不変
- [ ] DesignContextPacketを万能context schemaへ昇格しない
- [ ] side-effect invocationを実装しない

## 11. R0後の判断

R0で情報損失がなければ、次へ進む。

1. Builder re-entryの同一change challenge比較
2. Human first-pass review
3. read-only AI Business Actor

R0で不足が出た場合は、

```text
existing projection
  -> reusable derived projection helper
  -> local Reference helper
  -> cross-application EvidenceがあるProduct gap
```

の順で扱う。

一つのAgent use caseだけを理由にTEI CoreへAgent専用kindを追加しない。
