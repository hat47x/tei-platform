# Business Procedure Canonical Contract Pre-ADR Audit — 2026-09

Status: **Lane C / sole Pre-ADR readiness authority / Not Ready / no production Core adoption**

## 1. この文書の責務

この文書はBusiness Procedure / Handoff production contractについて、**ADRを起こしてよいかだけを判定する正本**です。

詳細Evidenceは次へ委譲します。

- `plan/issue/000042-business-procedure-cross-view-reference.md` — A-series Pressure inventory
- `plan/BUSINESS_MODEL_CURRENT_MINIMUM_CANDIDATE_2026-09.md` — current minimum比較基線
- `plan/BUSINESS_MODEL_SHARED_ACCEPTANCE_BUNDLE_PRESSURE_2026-09.md` — multi-source Acceptance / freshness / atomicity pressure
- `plan/BUSINESS_MODEL_PRODUCTION_ACCEPTANCE_BOUNDARY_AUDIT_2026-09.md` — production Shared Acceptanceで分離すべきsource / semantic / Conformance base境界
- 各`BUSINESS_PROCEDURE_*` / `BUSINESS_ACTION_*` Pressure文書
- X-TEA Evidence Register / E-015 / E-016

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

## 3. Current authoring candidates

### Procedure / O1

Current strongest candidate:

```text
same Business Procedure source
  Step.protectedApplicationPattern?  // zero-or-one
```

Test-only inline candidateで:

- strict decode
- round-trip
- zero-or-one
- multi-target fail closed
- generic role field fail closed
- prior sidecar pressureと同じnormalized D4+O1 meaning

をPressure済みです。

Production sidecarは一つのStep-local relationのためにcomposite source machineryを要求するため第一候補から下げます。

### Handoff

Current strongest candidate:

```text
standalone Handoff source
  Handoffs[]
    from { procedure, step }
    to   { procedure, step }
    condition
```

Test-only candidateで:

- current Reference Setと同じnormalized Handoff meaning
- Procedure artifact locatorを保持しない
- stale endpoint fail closed
- Workflow field拒否

をPressure済みです。

Reference Set manifest内Handoffはfixture assemblyには便利ですが、locator metadataとBusiness meaningを混在させるためproduction source候補としては弱いです。

## 4. Authority boundaries

### O1

```text
BP-06
  supporting authority -> security.authorize
  protected target     -> order_entry.submit_order
  direct realization  -> none
```

Authorization policyはcorroboration-onlyでありBusiness targetをauthor/backfillしません。
Generic `{Kind, Role, Target}` taxonomyは未採用です。

### Handoff

```text
explicit Handoff source -> normalized relation
Step adjacency           -X-> Handoff
Observation              -X-> Handoff
Function sequence        -X-> Handoff
```

H1/R3はhandoff-onlyです。Business Flow / DFD / Workflow runtimeへ拡張しません。

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

したがってfirst-class Business meaningと`ir.World`埋め込みは同義ではありません。

```text
separate normalized Business plane -> strongest current candidate
ir.World.Business                  -> current Evidenceでは不要
production internal/business       -> deferred
```

## 6. Multi-source Shared Acceptance

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

Test-only source snapshot:

```text
logical source key
+ content fingerprint
```

でPreview対象sourceのfreshnessを分離してPressureします。

- Procedure source change -> stale
- inline O1 source change -> stale
- Handoff source change -> stale
- source enumeration orderだけ -> same snapshot

Semantic revisionとsource freshnessを同一視しません。

さらにproduction Acceptanceでは、Business source freshnessだけでは不足します。Preview後に`ir.World`側のApplication Pattern等が変化した場合、Business sourcesとBusiness semantic meaningが同一でもcross-view Conformanceの根拠はstaleになります。

したがって概念上は少なくとも:

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

を分離します。

`business_model_conformance_base_freshness_pressure_test.go`では、Business sourceを変えずに`order_entry.submit_order`をWorldから除去した場合にPreview baseをstaleとしてrejectするtest-only Pressureを追加しました。

Production hash / ETag / revision algorithmはまだ採用しません。既存の`ISSUE-000027` / `ISSUE-000046`が持つ`baseModelRevision` / shared Acceptance原則と接続し、Lane C独自のgeneric transaction engineは作りません。read-only Semantic diff / impactは`ISSUE-000039`へ委譲します。

### Atomicity

Current coordinated case:

```text
Procedure BP-09 -> BP-09-renamed
Handoff from.step BP-09 -> BP-09-renamed
```

Source-level Pressureでは:

```text
Procedure sideだけrename -> fail closed
Handoff sideだけrename   -> fail closed
both in same bundle       -> normalize succeeds / new Business revision
```

まで進んでいます。

したがってGate Eの概念的問いはかなり閉じ、残るのはproduction implementation contractです。

## 7. Derived views

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

X-TEA E-016はFunction/Table I/Oのexplicit `OpC/R/U/D`からmatrixをderiveしていたことを示します。将来C/U/D gapが実例で反復してもBusiness Procedure CRUD fieldを第一候補にしません。

## 8. Subsystem / SubjectArea

Current Referenceではderived ownership lens / Namespace lensがかなり有用です。

```text
Namespace != SubjectArea
ownership/access lens != Subsystem
```

Canonical Subsystem / SubjectAreaは、direct/effective CUD、explicit Function call、published cross-boundary Action encapsulation等でcurrent lensが具体的に破綻した場合だけreopenします。

## 9. Command / Observation

- Command accepted != physical completion
- transport/read success != observed fact
- runtime Observation identityをBusiness Canonical identityへ昇格しない
- positive ObservationからHandoffを生成しない

Manufacturing / inventory second-corpus outlineはこの非対称性を別domainでも補強しましたが、synthetic Data ownership等を追加する根拠にはしていません。

## 10. Coreへ今入れないもの

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

## 11. ADR readiness gates

### Gate A — actual execution

Actual Repository environmentで:

```text
gofmt
go vet
go test
reference checker
```

がgreenであること。

### Gate B — minimum meaning

Actual testsでmanual/external Action、uncertainty、multi-Application、D4 ref kind separation、future Data/Relation fail-closed等を確認する。

### Gate C — address / Handoff

- Procedure-scoped localKey
- stale endpoint fail closed
- explicit Handoff authority only
- standalone Handoff candidate = current normalized meaning
- Handoff-only boundary維持
- second realistic Handoffで`from/to/condition` sufficiency確認

### Gate D — O1

- D4-only lossをactual確認
- singular narrow O1 lossless
- inline candidate strict decode / round-trip / sidecar-equivalent normalization
- Authorization policy corroboration-only
- production migration/versioning説明
- multi-targetを先取りしない

### Gate E — placement / Shared Acceptance

Conceptual multi-source normalization / Business source freshness / referential atomicityはsource-level test-only Pressureで前進済みです。さらにWorld / Conformance base freshnessをBusiness source freshnessから分離するPressureを追加しました。

Production ADR前に残るのは:

1. existing shared Acceptance boundaryがBusiness source baseとcross-view Conformance baseをどのproduction contractで固定・再確認するか
2. candidate change-setをどのrepository transaction boundaryでatomicにapplyするか
3. Accept後に全Business sourceをreloadしてsame normalized Business revisionを再現するproduction flow
4. cross-view Conformance失敗時にpartial writeを残さないcommit boundary
5. semantic revision identity / source freshness / conformance baseを物理的に一つのrevision tokenへ束ねるか分離するか
6. actual Repository testsでmulti-source bundle / source freshness / conformance-base freshness / coordinated renameをgreenにする
7. separate Business planeでactual Conformanceがlosslessであること
8. `ir.World` embeddingが必要になる具体的Evidenceがない限り配置Decisionを分離すること

Lane Cはplatform-wide `baseModelRevision`、hash / ETag algorithm、Studio / AI専用Acceptance APIを決めません。shared Acceptance原則は`ISSUE-000027` / `ISSUE-000046`へ接続し、Semantic diff / impact解析は`ISSUE-000039`へ委譲します。

### Gate F — derived projections / mutation gap

- Data / CRUD projection actual green
- no-traceをno-dependencyへ丸めない
- C/U/D gapをBusiness Procedureへ埋めない
- realistic C/U/D caseまでgeneric mutation/effect contractを作らない
- future operation modelでdirect/effective distinctionを失わない

### Gate G — adjacent domain

- multi-Application / social-system realism actual green
- physical Command/Observation actual green
- exact identitiesなしにsecond-corpus ownershipをsynthetic化しない
- Subsystem / SubjectAreaはderived lens不足のconcrete caseまでreopenしない

## 12. Current readiness

```text
minimum semantic candidate
  = D4 + singular narrow O1 + S4-A + H1/R3

Procedure authoring candidate
  = inline singular protectedApplicationPattern?

Handoff authoring candidate
  = standalone set-level Handoff source

placement candidate
  = separate normalized Business semantic plane

multi-source semantic bundling / source freshness / atomicity
  = source-level test-only Pressureで有力

cross-view Conformance base freshness
  = test-only Pressure追加

shared Acceptance implementation
  = existing cross-lane acceptance workへ接続

ir.World.Business
  = current Evidenceでは不要

Production ADR
  = Not Ready
```

主要blocker:

1. actual Repository runner green未取得。
2. inline O1 / standalone Handoff production adoption・migration未確定。
3. production shared AcceptanceでBusiness source base / Conformance base / atomic apply / reload Evidenceをどう実装するか未確定。
4. second realistic Handoff不足。
5. qualifying TEI-original C/U/D case不足。
6. separate Business plane不足を示すEvidenceはなく、`ir.World` integrationはまだ不要。

したがって**Pre-ADR Status = Not Ready**を維持します。
