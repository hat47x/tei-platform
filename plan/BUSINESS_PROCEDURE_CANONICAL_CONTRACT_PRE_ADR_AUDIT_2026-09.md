# Business Procedure Canonical Contract Pre-ADR Audit — 2026-09

Status: **Lane C / sole Contract ADR readiness authority / Not Ready / no production Core adoption**

## 1. この文書の責務

この文書はBusiness Procedure / Handoff production contractについて、**Contract ADRを起こしてよいかだけを判定する正本**です。

Production adoption readiness、Product Gate C後のStudio / AI shared Acceptance実装は別段階です。

Current minimum candidate:

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

```text
D4 + singular narrow O1 + S4-A + H1/R3
```

## 2. Readiness層を分ける

```text
Contract ADR readiness
  -> semantic / authority / freshness contractを決めてよいか

Production adoption readiness
  -> production source authorityを安全に切替・再読込できるか

PAS-01 / Studio / AI integration
  -> Product Gate C後のshared Acceptance end-to-end
```

`ISSUE-000027` / `ISSUE-000046`はProduct Gate C Ready後のIssueである。これらのproduction実装が未完成であることをLane C Contract ADR blockerへ循環させない。

詳細は`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`を正とする。

## 3. Current semantic decision

### Procedure body

Current minimumはD4。

```text
Body
  businessAction
  applicationPatterns[]
  validations[]
  capabilities[]
```

D3はhistorical/subtraction Evidenceとして残す。

### O1 protected Application target

BP-06では:

```text
supporting semantic authority -> security.authorize
protected Application target  -> order_entry.submit_order
direct Application realization -> none
```

である。

D4だけではprotected targetが失われるためlossy。Current production authoring candidateはsame Procedure source内のsingular narrow:

```text
protectedApplicationPattern?
```

である。

`applicationPatterns[]`へ混ぜてdirect realization扱いしない。Generic `{Kind, Role, Target}` taxonomy、multi-target、sidecar production authorityはEvidenceが出るまで採用しない。

### Step address

S4-A:

```text
Procedure-scoped localKey
```

をcurrent candidateとする。

Handoff endpointをexactに指すためのaddressabilityは必要だが、global Canonical Step identityやruntime identityは不要。

### Handoff

H1/R3:

```text
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

をcurrent candidateとする。

Authorityはexplicit Handoff sourceだけ。

```text
Step adjacency           -X-> Handoff
Application Function seq -X-> Handoff
physical Observation     -X-> Handoff
Lifecycle transition     -X-> Handoff
Deployment state         -X-> Handoff
```

Handoffはhandoff-only relationであり、full Business Flow / DFD / Workflow runtimeへ膨張させない。

Current H1/R3にはglobal Handoff IDを追加しない。Exact duplicate relationのsource-authority conflictは既存`from/to/condition` meaningでdetectでき、same endpoints + different conditionをendpoint-only identityで潰す根拠もない。

## 4. Placement

Current strongest candidateはseparate normalized Business semantic plane。

```text
Business authoring sources
  -> normalized Business model
  -> Business semantic revision

TEI workspace sources
  -> ir.World

normalized Business model
  <-> exact cross-view Conformance <-> ir.World
```

Current Evidenceでは`ir.World.Business` embeddingは不要。

Business-only wording / protected target / Handoff condition changeは`ir.World`を偽変更せずBusiness semantic changeとして扱える。一方World側exact targetが消えればseparate planeでもfail closedする。

`ir.World.Business`必要性Evidenceがないことはblockerではない。

## 5. Authoring / migration direction

```text
legacy authority
  -> migration adapterでread
  -> candidate preview

candidate
  -> non-authoritative
  -> normalize / compare / Conformance

Shared Acceptance
  -> one accepted authority cutover

steady-state
  -> candidate-only loader
```

Production candidate:

```text
Procedure source
  -> inline singular O1

standalone Handoff source
  -> H1/R3
```

禁止:

- permanent legacy + candidate dual-read
- merge priority / last writer wins
- schemaVersionによるauthority priority
- source pathをBusiness identityにすること
- `file exists == authority`とみなすこと
- duplicate `Procedure.id`を複数managed sourceから受理すること
- exact duplicate H1/R3 meaningを複数managed standalone Handoff sourceから受理すること
- path order / input order / first-writer / last-writerでsource authorityを裁定すること
- Handoff endpoint pairだけを暗黙Handoff identityとして発明すること

```text
physical source path
  != Business logical identity
  != accepted source authority
```

Procedure sourceでは`Procedure.id`をcurrent logical identity candidateとして使うが、durable Canonical Procedure kind採用を意味しない。

Standalone Handoff sourceではexact H1/R3 tupleをduplicate meaning detectionに使えるが、global Handoff ID採用を意味しない。

Migration adapterとsteady-state candidate loaderを分離する。

## 6. Freshness / Shared Acceptance contract

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

Current source-level Pressureでは:

- Business source changed after Preview -> stale
- Business source unchanged but World changed -> stale
- Procedure Step rename only -> Handoff endpoint stale / fail closed
- Handoff endpointだけrename -> fail closed
- coordinated Procedure + Handoff rename -> one new candidate revision
- legacy + candidate simultaneous authority -> reject
- source add/remove -> freshness対象
- authoritative source-root membership change -> freshness対象
- duplicate Procedure logical source -> fail closed
- duplicate exact standalone Handoff meaning -> fail closed

まで確認対象を作っている。

Complete stale envelopeは少なくとも:

```text
relevant source bytes
+ source add/remove
+ authoritative source-root membership
+ logical source uniqueness
+ World / Conformance base
```

をfalse-negativeなくcoverする。

物理token shapeは固定しない。

```text
one token
  -> complete stale envelopeをfalse-negativeなくcoverすれば可

independent source planes
  -> independent/composite checkへreopen可
```

Git HEAD equalityだけをunconditional freshness proofとしない。Unrelated changeによるfalse-positive staleより、relevant changeのfalse-negativeを避ける。

## 7. Existing implementation reuse audit

既存TEIから再利用候補:

```text
workspace / ManagedFiles
  -> source-root discovery / source set

Source Index
  -> Artifact fingerprint / source trace / stale validation primitive

Workbench / Consumer Impact / Release Graph
  -> deterministic Preview / impact projection precedent

Deployment expected-revision / bound fingerprints
  -> stale-reject architectural precedent
```

ただし:

```text
Deployment Run.Revision
  != Business semantic revision

Deployment durable state machine
  != Business Workflow / Business source transaction
```

である。

Current audited implementation surfacesでは、generic source-authoring Shared Acceptance / multi-file authoritative apply / authority cutover / production Business loaderは確立されていない。

これはProduction adoption implementation gapであって、Business schema field追加Evidenceではない。

## 8. Fresh-process feasibility

`business_authoring_candidate_fresh_process_pressure_test.go`はtest-onlyで、parent processがcandidate Procedure / Handoffをdiskへ書き、child processへcandidate paths + World workspaceだけを渡す。

Legacy-looking decoyをdiskへ置いてもchildへlegacy locatorを渡さず:

```text
candidate Procedure + Handoff
  -> fresh process
  -> strict decode
  -> normalize
  -> current World Conformance
  -> same Business semantic fingerprint
```

をPressureする。

したがってfresh-process candidate-only feasibilityは**source-level implementation済み / actual runner未実行**であり、独立した設計未解決ではない。

## 9. Derived Data / CRUD / ownership

Business sourceへData / Relation / CRUD / ownership / Namespaceを重複authorしない。

```text
no Data trace != no Data dependency
trace exists  != complete dependency set
```

Current exact CRUD operation Evidence:

```text
BP-00 -> R sales.order
BP-02 -> R customer.customer
BP-03 -> R sales.order_line
```

禁止:

- wordingからCRUDを推測
- `record` / `command` kindからCRUDを推測
- Database Contract WritesをBusiness Action Uへ自動帰属
- Lifecycle transitionをCRUD Uへ短絡
- Capability payload / Custom Go realizationからgeneric effectを発明

Qualifying C/U/D case absenceはBusiness Procedure Contract ADR blockerではない。Future mutation/effect contract reopen triggerとして残す。

```text
Namespace != SubjectArea
Database ownership/access lens != Subsystem
```

も維持する。

## 10. Command / Observation

- Command accepted != physical completion
- transport/read success != observed fact
- runtime Observation identityをBusiness Canonical identityへ昇格しない
- positive Observationからresponsibility Handoffを生成しない

Manufacturing / inventory synthetic outlineはこの非対称性を別domainでもPressureするが、accepted Business Handoff authorityにはしない。

## 11. Second realistic Handoff audit

Current explicit primary Handoff:

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
condition = follow-up required after earlier order interaction
```

Second-case探索では次をrejectした。

- BP-08 -> BP-09 adjacency
- FU-01 -> FU-02 adjacency
- customer-edit intentional defectからmaster-data workflowを発明
- Business Lifecycle state transition
- Deployment / runtime transition
- Watanabe external patternの直接fixture化
- RQ-025 / RQ-026の一般的Handoff allowanceから具体的instanceを生成
- synthetic `fulfillment_console` / warehouse / external delivery pressureからaccepted Handoffを逆生成

`reference/business-system-design` inventoryもorder-entry / follow-up中心で、別のaccepted Business responsibility chainはない。同directory README自身が複数Subject Area / Subsystem跨ぎやexternal contractをfuture pressureとして挙げる。

したがって:

```text
TEI-original independent second Handoff
  = none

existing accepted corpusからのsafe promotion candidate
  = none

existing-corpus search
  = closed
```

とする。

これはH1/R3 sufficiency証明ではない。Test件数のためにsource authorityを発明しないためのnegative Evidenceである。

### Actual green後のauthoring protocol

Gate A actual green後に第二例を作る場合:

```text
independent Business need / responsibility boundary
  ↓
Business requirement / Reference seedを先に成立
  ↓
source responsibility / target responsibility / continuation conditionをreview
  ↓
minimal Procedure sources
  ↓
explicit Handoff candidate
  ↓
from / to / condition losslessness Pressure
```

とする。

`Handoff fieldを検証したい -> それらしい業務を作る`の順序は禁止する。

新caseでは必要に応じて:

- natural-language condition sufficiency
- trigger vs condition separation
- actor/job change independence
- event/document identity necessity
- thin Handoff vs full Business Flow necessity

を確認し、具体的lossが出た項目だけcontract reopenする。

## 12. Coreへ今入れないもの

- production `BusinessProcedure` / `BusinessHandoff` kind
- `ir.World.Business`
- production `internal/business`
- generic SemanticRef / role enum / Core Edge
- full Business Flow / DFD
- Workflow state / retry / timer / queue / saga
- global Canonical Step ID
- global Canonical Handoff ID
- Observation専用Canonical identity
- Business Procedure CRUD field
- Application Pattern generic CRUD field
- Semantic Capability generic Effect
- Database Contract C/U/D enum
- mutation binding / MutationIntent / Function-Data-I/O Canonical kind
- synthetic second-corpus Data ownership
- Business専用generic transaction / revision subsystem
- old/new Business source permanent dual-read / merge-priority contract
- Git SHAのplatform-wide `baseModelRevision`固定
- schemaVersionによるauthority priority
- mandatory `businessAuthoringMode`
- Workbench / impact projectionのauthoritative source化
- Deployment durable state machineのBusiness authoring / Workflowへの流用

## 13. Contract ADR readiness gates

### Gate A — actual execution

Actual Repository environmentで同一revisionについて:

```text
read-only gofmt check
go vet ./...
go test ./...
go test -count=1 ./tools/reference-business-procedure-check
baseline reference checker
intentional defect checker
```

がgreenであること。

`BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md`を使う。

Fresh-process candidate-only subprocess Pressureもこのactual test runに含める。Source inspectionをactual greenへ読み替えない。

### Gate B — minimum meaning / authoring feasibility

Actual testsで:

- manual / external Action
- uncertainty
- multi-Application
- D4 ref kind separation
- D4-only loss / D4+O1 losslessness
- inline O1 strict decode / round-trip
- standalone Handoff strict decode / round-trip
- candidate-only disk reload
- fresh-process candidate-only reload
- duplicate Procedure logical source fail closed
- duplicate exact standalone Handoff source authority fail closed
- endpoint-only Handoff identityを発明しない

を確認する。

Fresh-process / source-uniqueness testsはsource-level実装済みなので、残るのはGate A actual execution Evidenceである。

### Gate C — address / Handoff sufficiency

- Procedure-scoped localKey
- stale endpoint fail closed
- explicit Handoff authority only
- standalone Handoff candidate = current normalized meaning
- exact duplicate H1/R3 across managed standalone sources fail closed
- path/input orderでHandoff source authorityを裁定しない
- same endpoint pairだけをimplicit Handoff identityにしない
- global Handoff IDを先取りしない
- Handoff-only boundary維持
- Procedure actorをendpoint identityへ複製しない
- Procedure triggerとHandoff conditionを別authorityとして扱う
- migration previewではlegacy Handoffをinput / Evidenceとして読める
- accepted cutover後はlegacy + standalone Handoff simultaneous authorityを許さない
- candidate-only reload / normalize feasibility
- second realistic **TEI-original independent** Handoffで`from/to/condition` sufficiency確認

Existing-corpus searchはclosed。Gate A actual green後、独立Business requirement / Reference seedを先にauthorしてsecond caseを成立させる。

### Gate D — O1

- D4-only loss actual確認
- singular narrow O1 lossless
- inline strict decode / round-trip / sidecar-equivalent normalization
- Authorization policy corroboration-only
- migration preview legacy sidecar read可
- accepted cutover後legacy sidecar + inline O1 dual authority reject
- same-step conflicting targetを自動mergeしない
- candidate-only reload / normalize
- multi-targetを先取りしない

### Gate E — placement / freshness feasibility

Contract ADRで必要なのはproduction Shared Acceptance engine完成ではなく、次のcontract / feasibility Evidence。

- source freshnessとConformance base freshnessを区別
- complete stale envelope原則
- source add/remove / membership changeをcover
- logical source uniquenessをcover
- duplicate Procedure logical authorityをfail closed
- duplicate exact standalone Handoff authorityをfail closed
- coordinated multi-source changeをone candidate bundleとして扱える
- legacy + candidate dual authority reject
- candidate-only fresh-process reload feasibility
- separate Business planeでlossless Conformance可能

Production adapter / atomic write implementationはProduction adoption readinessへ送る。

### Gate F — derived projections

- Data / CRUD projection actual green
- no-traceをno-dependencyへ丸めない
- C/U/D gapをBusiness Procedureへ埋めない
- realistic mutation caseまでgeneric mutation/effect contractを作らない

### Gate G — adjacent domain

- multi-Application / social-system realism actual green
- Command/Observation boundary actual green
- synthetic second-corpus ownershipを作らない
- Subsystem / SubjectAreaはconcrete lens failureまでreopenしない

## 14. Production adoption readiness — Contract ADR後の追加Evidence

Contract ADR Readyになってもproduction authorityへ直ちに昇格しない。

Production adoptionでは別途:

1. Business-specific production authoring adapter
2. workspace / Source Indexを利用したproduction candidate discovery
3. complete stale envelope実装
4. shared Acceptance integration
5. atomic multi-file source apply
6. atomic authority-discovery cutover
7. fresh-process candidate-only production reload
8. partial failure / crash safety
9. post-reload cross-view Conformance

を実証する。

`ISSUE-000027` / `ISSUE-000046`のProduct Gate C後shared Acceptance loopと整合させるが、Lane C専用generic transaction subsystemは作らない。

## 15. Current readiness

```text
minimum semantic candidate
  = D4 + singular narrow O1 + S4-A + H1/R3

Procedure authoring candidate
  = inline singular protectedApplicationPattern?

Handoff authoring candidate
  = standalone set-level Handoff source

placement
  = separate normalized Business semantic plane

migration
  = migration adapter + candidate-only steady-state loader
  = legacy-read preview -> one cutover -> candidate-only authority

source-authority uniqueness
  = Procedure duplicate logical source Pressure implemented
  = exact duplicate standalone Handoff Pressure implemented
  = actual execution pending

fresh-process feasibility
  = source-level subprocess Pressure implemented
  = actual execution pending

second realistic Handoff
  = existing-corpus search closed / none qualifying
  = after actual green, independent Business requirement first

Contract ADR
  = Not Ready

Production adoption
  = Not Ready
```

### Remaining Contract ADR blockers

```text
Blocker A
  actual Repository runner green
  including source-uniqueness and fresh-process candidate-only Pressure

Blocker B
  independent Business requirementから生じた
  second realistic TEI-original Handoffで
  from / to / condition sufficiency確認
```

**実質的なContract ADR blockerはこの二つまで縮約した。**

次はblockerではない:

- production shared Acceptance / atomic apply未実装
- final revision token shape未決定
- qualifying C/U/D case absence
- `ir.World.Business` necessity Evidence absence
- Product Gate C後のStudio / AI integration未実装
- PR #482がmainへmerge済みであること自体

## 16. Verification status

2026-09-09時点でHAT47Xはoffline。

Lane C Reference / Pressure PR #482はmainへmerge済みで、standalone Handoff source uniqueness PressureもPR #520でmainへmerge済みである。ただしRepositoryへのReference / Pressure mergeは、Contract ADR adoption・production Core adoption・Gate A actual greenのいずれも意味しない。

したがってcurrent source-level tests / checkerについて:

```text
gofmt PASS  -> 未主張
go vet PASS -> 未主張
go test PASS -> 未主張
checker PASS -> 未主張
fresh-process PASS -> 未主張
source-uniqueness PASS -> 未主張
```

Contract ADRは**Not Ready**を維持する。次の状態遷移は、同一revisionでのactual Gate A greenとindependent second realistic TEI-original Handoffの両方が揃った場合だけ再判定する。
