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

したがってfresh-process candidate-only feasibilityは**source-level implementation済み / actual runner未実行**であり、独立した設計未解決ではない（2026-09-20更新: Gate A actual executionは実行済みでPASS。R = `e2856e6e568b6be966c6b8689ebea479ec184a4a`、判定の正本は[Pre-ADR Audit](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)§13）。

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

したがって既存accepted corpusについては:

```text
TEI-original independent second Handoff
  = none

existing accepted corpusからのsafe promotion candidate
  = none

existing-corpus search
  = closed
```

とする。

その後、ISSUE-000050では**受注生産を背景とする在庫・製造連携の小型TEI-original scenario**をsecond-corpus research targetとして選定し、provenance / reuse boundary / target scope / fact-candidate-unresolved分類まで整理した。

ただし2026-09-21より前の状態として（この節の状態は§17で更新した）:

```text
accepted independent Business need / Reference seed
  = none

accepted second Handoff from / to / condition
  = none

Blocker B positive Evidence
  = unresolved
```

である。Corpus selectionはH1/R3 sufficiency証明ではない。Test件数のためにsource authorityを発明しないためのnegative / preparatory Evidenceとして扱う。

PR #596 / `DESIGN_SYNTHESIS_MANUFACTURING_ACTUALITY_HANDOFF_PROOF_REFERENCE_2026-09.md`では、このsecond-corpus pressureへP5 / P6のnegative proofを適用した。ここで確認できたのは、**Handoff positive Evidenceではなく、authorityを越えた導出規則を除外できること**である。

```text
Command accepted -> physical completion
  = contradiction as derivation rule

Observation execution success -> physical completion
  = contradiction as derivation rule

explicit external Observation Result=true
  -> observed physical fact only
  = supported at that limited scope

Step adjacency / actor change / Function sequence / positive Observation
  -> Handoff
  = contradiction as derivation rule

second-corpus Handoff existence / from / to / condition
  = unresolved
```

したがってR2 negative proofは、Gate A green前に許されるnon-inference / candidate-contraction Evidenceを強めるが、accepted independent Business seed、second Handoff authority、Blocker B positive Evidenceのいずれにも昇格しない。Actual green後のauthoring protocolも変更しない。

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

ISSUE-000050のcorpus / provenance researchはGate A green前でも継続できるが、accepted Business seed / Procedure / Handoff authoringへは進めない。

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

**実行結果（2026-09-20）**: `tools/run-business-procedure-gate-a.sh`により、clean full checkoutのexact revision `R = e2856e6e568b6be966c6b8689ebea479ec184a4a`について、Step A1〜A7がすべて成功し（`gateAPassed = true`）、Gate Aは**PASS**した。A7ではcritical test 26件のexact nameがすべて存在し、focused runで26件すべてがPASSした（source-uniqueness、source-root membership、candidate-only disk / fresh-process reload、second-corpus non-inference Pressureを含む）。Evidence: GitHub Release [`business-procedure-gate-a-evidence-R-e2856e6`](https://github.com/hat47x/tei-platform/releases/tag/business-procedure-gate-a-evidence-R-e2856e6)（tagは`R`を指す）のasset `tei-lane-c-gate-a-evidence-R-e2856e6.tar.gz`、SHA-256 `d0f0f5a937340a59e04b00cc8bcd8d8046f0327c2384e951dc0de2fd4a788971`。Gate Aはmechanicalなexecution Evidenceであり、Contract ADR readinessやPAS-07 acceptedを意味しない。

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

Fresh-process / source-uniqueness testsはsource-level実装済みで、Gate A actual execution Evidenceも取得済みである（上記、R = `e2856e6`）。

### Gate C — address / Handoff sufficiency

- Procedure-scoped localKey
- stale endpoint fail closed
- explicit Handoff authority only
- standalone Handoff candidate = current normalized meaning
- exact duplicate H1/R3 across managed standalone sources fail closed
- path/input orderでHandoff source authorityを裁定しない
- same endpoint pairだけをimplicit Handoff identityにしない
- same endpoints + different conditionはdistinct H1/R3 meaningとして併存できる
- 同一H1/R3 meaning集合のnormalized resultはReference / input列挙順に依存しない
- global Handoff IDを先取りしない
- Handoff-only boundary維持
- Procedure actorをendpoint identityへ複製しない
- Procedure triggerとHandoff conditionを別authorityとして扱う
- migration previewではlegacy Handoffをinput / Evidenceとして読める
- accepted cutover後はlegacy + standalone Handoff simultaneous authorityを許さない
- candidate-only reload / normalize feasibility
- second realistic **TEI-original independent** Handoffで`from/to/condition` sufficiency確認

Existing-corpus searchはclosed。ISSUE-000050でsecond-corpus research targetは選定済みだが、accepted Business seed / Handoff authorityは未成立。Gate A actual green後、独立Business requirement / Reference seedを先にauthorしてsecond caseを成立させる。（2026-09-21: seedとHandoffをauthorし、Evidenceを取得した。§17）

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
  = actual execution PASS（R = e2856e6、2026-09-20）

fresh-process feasibility
  = source-level subprocess Pressure implemented
  = actual execution PASS（R = e2856e6、2026-09-20）

second realistic Handoff
  = second-corpus research target selected
  = provenance / reuse / fact-candidate-unresolved boundary established
  = P5/P6 negative proof confirms non-inference / actuality boundary
  = independent Business seed authored（2026-09-21、作成者self-review。独立review未実施）
  = from / to / condition source/test-level losslessness + actual execution PASS（R = 82e4301、§17）
  = Contract ADR readiness re-judgment pending

Contract ADR
  = Not Ready

Production adoption
  = Not Ready
```

### Remaining Contract ADR blockers

```text
Blocker A — 充足（2026-09-20、R = e2856e6）
  actual Repository runner green
  including source-uniqueness and fresh-process candidate-only Pressure

Blocker B — Evidence取得済み（2026-09-21、R = 82e4301、§17）。作成者self-reviewのみ、再判定は未実施
  independent Business requirementから生じた
  second realistic TEI-original Handoffで
  from / to / condition sufficiency確認
```

**実質的なContract ADR blockerは、Blocker Bだけまで縮約した（Blocker Aは2026-09-20に充足）。Blocker Bは2026-09-21にEvidenceを取得したが、再判定が済むまで充足とは扱わない。**

次はblockerではない:

- production shared Acceptance / atomic apply未実装
- final revision token shape未決定
- qualifying C/U/D case absence
- `ir.World.Business` necessity Evidence absence
- Product Gate C後のStudio / AI integration未実装
- PR #482がmainへmerge済みであること自体

## 16. Verification status

Actual Repository Gate A execution Evidenceは2026-09-20に取得した（R = `e2856e6e568b6be966c6b8689ebea479ec184a4a`、結果PASS、上記§13）。以後のexecution target revisionは一時状態なので本正本へ固定せず、`BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md`に従って、実行ごとにexact revision `R`、runner kind、environmentを記録する。判定対象のtreeが変わった場合は、新しいrevisionでGate Aを取り直す。

Lane C Reference / Pressure PR #482はmainへmerge済みで、standalone Handoff source uniqueness PressureもPR #520でmainへmerge済みである。ただしRepositoryへのReference / Pressure mergeは、Contract ADR adoption・production Core adoption・Gate A actual greenのいずれも意味しない。

したがってcurrent source-level tests / checkerについて:

```text
R = e2856e6について:
gofmt PASS  -> 主張（Gate A A1）
go vet PASS -> 主張（A2）
go test PASS -> 主張（A3 / A4）
checker PASS -> 主張（A5 / A6）
fresh-process PASS -> 主張（A7）
source-uniqueness PASS -> 主張（A7）
```

ただしこれはそのrevisionのmechanical Evidenceに限る。他のrevisionへ引き継がない。

Contract ADRは**Not Ready**を維持する。Blocker Aは充足し、Blocker BのEvidenceは2026-09-21に取得した（§17）。次の状態遷移は、Blocker Bの再判定である。


## 17. Blocker B — second realistic Reference Evidence（2026-09-21）

Gate A green後のauthoring protocol（§11）に従い、独立したBusiness needのseedを先にauthorし、そこから自然に生じた責任のHandoffを一件置いた。作業と対応づけの詳細は[ISSUE-000050](issue/000050-second-realistic-handoff-corpus.md)の「実施記録」を正とし、ここには判定に必要な事実だけを置く。

```text
Evidence nature
  = source/test-level cross-view pressure + actual Repository execution
  = author self-reviewed（独立review未実施）

revision
  R = 82e430109f8f02944519738045f3d6c0bef4f295

source
  reference/business-system-design/second-corpus/
  make-to-order.production-run / PR-06
    -> finished-goods.stocking / ST-00

actual execution
  tools/run-business-procedure-gate-a.sh
  A1-A8 exit 0、gateAPassed = true
  A7: critical 39/39 present、39/39 PASS（+13 TestSecondRealisticReference*）
  A8: 二つのfixtureがbaseline checkerでdiagnosticなし
```

保管先: GitHub Release [`business-procedure-gate-a-evidence-R-82e4301`](https://github.com/hat47x/tei-platform/releases/tag/business-procedure-gate-a-evidence-R-82e4301)（tagは`R`を指す）のasset `tei-lane-c-gate-a-evidence-R-82e4301.tar.gz`、SHA-256 `3a433835241b82b564e47cb5eb48e63c24f7a5afd3797ae1090afa9add058d7a`。展開して`MANIFEST.sha256`で各fileを再照合できる。§13のR = `e2856e6`の記録は書き換えない。本revisionでも同じrunnerがA1〜A7をPASSしたが、Blocker Aの判定を置き換えるものではない。

§11が「新caseで確認する」とした五点の結果:

| 点 | 結果 |
| --- | --- |
| natural-language condition sufficiency | conditionはcanonical、正規化model、standalone sourceを無損失に往復した。このfixtureのconditionの語彙とtarget triggerとの一致をtestで固定したが、意味上の言い換えを検出する仕組みではない |
| trigger vs condition separation | 二つのProcedureはtriggerとconditionを別々に持ち、conditionはtarget triggerと文字列として一致しない |
| actor/job change independence | actorが異なる場合に、explicit sourceが無ければHandoffが導出されないことを確認した。**actorが同じ場合にtransferを否定しないこと**は、両Procedureのactorが異なるため確認していない |
| event/document identity necessity | conditionはevent / documentのidentityなしで保持できた。conditionを支えるevent / documentはauthorしていない（未解決U-5）ため、識別の要否には圧力を掛けていない |
| thin Handoff vs full Business Flow necessity | 一件の物理的な引き渡しと責任の移転は、from / to / conditionだけで失われなかった。情報 / document / partyのflowを含む業務は扱っていない |

具体的なlossは出なかったため、Contract candidateのreopen triggerは成立していない。ただし上のとおり圧力を掛けていない点があり、それらが不要であることの証明ではない。RRB-P2の項目4（Data-state / ownership境界のcross-check）も、対象のData authorityが無いため未確認である。

Blocker Bについて次を区別する。

```text
Evidence取得
  = 完了（R = 82e4301）

独立review
  = 未実施（作成者とreviewerが同一のAI）

Auditによる再判定
  = 未実施

Contract ADR
  = Not Ready（維持）
```

利用者の方針（2026-09-21）により、再判定は別の生成AIによる独立reviewの後に行う。self-reviewだけを根拠にBlocker Bを充足とはしない。二例目も同じ作成者が同じprotocolでauthorしたため、二例あることをHandoff一般の妥当性の証明として扱わない。
