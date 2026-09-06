# 000042 Business Procedure cross-view Reference

Status: In Progress

## 目的

`order_entry`を起点にData / Function-Interaction / Businessの三視点cross-checkをReferenceとしてPressureし、Business Procedureの最小production Canonical contractをEvidenceから決める。

先にWorkflow Engine、full Procedure schema、global Step identity、full Business Flow schemaを固定しない。

## 正式基線

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — RQ-025 / RQ-026
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `docs/business-system-design.md`
- `plan/LANE_B_RQ002_BUSINESS_PROCEDURE_CANONICAL_TRACE_AUDIT_2026-09.md`
- `plan/LANE_B_RQ025_RQ026_TRACE_AUDIT_2026-09.md`
- `plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`
- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`
- `plan/BUSINESS_PROCEDURE_IDENTITY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_FILE_NATIVE_SOURCE_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_ACCEPTANCE_TARGET_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_IDENTITY_OPTION_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_CANONICAL_MEANING_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_HANDOFF_CANONICAL_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_HANDOFF_CANONICAL_NECESSITY_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_HANDOFF_REPRESENTATION_OPTION_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_HANDOFF_VS_BUSINESS_FLOW_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_MULTI_APPLICATION_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_REALISTIC_MULTI_APPLICATION_SCENARIO_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_UNCERTAINTY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_REALIZATION_SUBTRACTION_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_STEP_IDENTITY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_STEP_ADDRESS_OPTION_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_STEP_ADDRESS_LAYER_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_APPLICATION_CONTEXT_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_NORMALIZED_MODEL_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`

## Lane境界

本IssueはレーンCのBusiness Procedure Reference / Evidenceを所有する。

RQ-025 / RQ-026はmainのProduct Requirementとして採用済み。PAS-07 Acceptance trace正本はレーンB側へ残す。

Gate B / Gate C scope、Business Lifecycle 000038、共通change graph 000039を本Issueから拡張しない。

## 固定する境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Handoff != full Business Flow / DFD
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Business Procedure != single Application child
Data / Function-Interaction / Business = 同じaccepted meaningを相互牽制する視点
```

Business ProcedureにはApplication / Semantic Worldから機械的に導出できないresidual meaningがある。

```text
purpose
trigger / event
actor / job context
manual / external / unresolved Business Action
judgment / continuation
handoff
```

Pattern-owned Data / Relation / Query / Capability / Datasource / Application ownership / datasource policyはnormalized `ir.World`からderiveし、Business sourceへコピーしない。

## Reference

### Primary

`reference/business-system-design/order-entry-business-procedure.json`

- BP-00〜BP-09
- BP-04 / BP-09はPatternなしのmanual / external Business Action

### Follow-up

`reference/business-system-design/order-entry-follow-up-procedure.json`

別Business文脈から同じ`order_entry.order_record`を利用する。

### Handoff set

`reference/business-system-design/order-entry-procedure-reference-set.json`

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
```

Procedure keyとartifact locatorを分離する。

### Intentional defect

`reference/business-system-design/order-entry-customer-edit-conflict.json`

期待diagnostic:

```text
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
severity = architecture-policy
```

### Intermediate multi-Application scenario

temp Workspaceで次を同一Procedureへ共存させる。

```text
order_entry interaction
  -> fulfillment_console interaction
  -> manual warehouse coordination
  -> external delivery continuation
```

synthetic second Applicationのためproduction業務仕様とはみなさず、中間Pressureとする。

## Cross-view checker原則

`tools/reference-business-procedure-check`は既存`workspace.Load()`のnormalized `ir.World`を利用する。

```text
Business authoring source
        +
normalized ir.World
        ↓
experiment checker
        ↓
Matrix / diagnostics / semantic pressure
```

- explicit referenceはexact identityで解決
- 名称類似で補完しない
- Pattern ID文字列からownerを推測しない
- Pattern + owner Applicationをnormalized Worldからexact解決
- Projectionだけでaccepted authorityを拡張しない

## Acceptance / Evidence Pressure matrix

| ID | Pressure | 現在の扱い |
| --- | --- | --- |
| A1 | exact semantic / Pattern reference resolution | test-only実装済み |
| A2 | manual / external actionにfake Functionを要求しない | 実装済み |
| A3 | Pattern-owned meaningをBusiness sourceへ複製しない | 実装済み |
| A4 | Patternから導けないexplicit refsだけBusiness側に保持 | 実装済み |
| A5 | intentional datasource access conflict | 実装済み |
| A6 | baseline clean | actual runner待ち |
| A7 | unknown field / schemaVersion等fail closed | 実装済み |
| A8 | purpose / trigger / actor / manual actionはBusiness residual | 実装済み |
| A9 | Business-only changeとPattern meaning changeを分離 | 実装済み |
| A10 | multiple Procedure -> same Pattern | 実装済み |
| A11 | Procedure keyをfile pathから分離 | 実装済み |
| A12 | pure file move = semantic no-op候補 | 実装済み |
| A13 | logical Procedure renameはstale link fail closed | 実装済み |
| A14 | review target continuity | 実装済み |
| A15 | approvalはrevision / fingerprint bound | 実装済み |
| A16 | file-native Procedure source pressure | 実装済み |
| A17 | Shared Acceptance target / stale pressure | test-only実装済み |
| A18 | identity-only shell | 比較用、最小十分とはみなさない |
| A19 | thin Business meaning | test-only実装済み |
| A20 | Canonical candidateでもPattern meaning非重複 | 実装済み |
| A21 | manual actionをthin candidateへ保持 | 実装済み |
| A22 | handoffをthin Business relationとして表現 | test-only実装済み |
| A23 | handoff condition change = semantic change | 実装済み |
| A24 | runtime Workflow detailを侵入させない | 実装済み |
| A25 | one Procedure -> multiple Applications | 実装済み |
| A26 | multi-Application PressureをGate B realizationへ結合しない | 実装済み |
| A27 | trigger / actor epistemic statusをlossless保持 | 実装済み |
| A28 | uncertainty change = Business semantic change | 実装済み |
| A29 | actor uncertainty != Authorization change | 実装済み |
| A30 | uncertainty vocabularyを早期固定しない | 方針固定 |
| A31 | realization subtraction | test-only実装済み |
| A32 | realization-only label changeをCore changeへ自動昇格しない | 実装済み |
| A33 | manual action survives realization subtraction | 実装済み |
| A34 | Step body meaning can be ID-free | test-only実装済み |
| A35 | handoffはexact Step addressabilityを要求 | 実装済み |
| A36 | Step addressability != runtime identity | 方針固定 |
| A37 | realistic multi-App + manual + external coexistence | 中間Pressure実装済み |
| A38 | Application associationはWorld解決でderiveし、名前からguessしない | 実装済み |
| A39 | access policyはPattern owner Applicationを使う | checker修正 + 回帰Pressure実装済み |
| A40 | fixture Application context != Procedure owner | 実装済み |
| A41 | Canonical handoff採用時はStep address解決をSource Index依存にしない | S4-A優先Pressure |
| A42 | Step local key renameをBodyMeaningChangedと分離 | test-only実装済み |
| A43 | global Canonical Step IDをdefault導入しない | 方針固定 |
| A44 | primary Application contextをsemantic prerequisiteからsubtract | C1をcheckerへ実装済み、actual runner待ち |
| A45 | authoring-only H0ではhandoff condition / endpoint changeをnormalized diffから失う | 反証Pressure実装済み |
| A46 | thin normalized handoff H1でaccepted relation changeを保持 | primary candidate / test-only |
| A47 | Data `semantic.Relation`流用 / generic Core Edge新設を避ける | R1 reject / R2 defer |
| A48 | Business-side thin relation R3 | primary pre-ADR candidate |
| A49 | R3 + S4-Aをnormalized Business modelとしてSource Index非依存validate | test-only実装済み |
| A50 | X-TEA fallbackでBusiness Flowが独立Dataflow model pressureを持つことを確認 | E-014追加済み |
| A51 | R3をhandoff-only relationへ限定し、full Business Flowを所有しない | negative Pressure実装済み |
| A52 | normalized handoff validation PASSをBusiness Flow / DFD Conformanceと呼ばない | test-only境界固定 |
| A53 | full Business Flowはrealistic Referenceで別pressureが出た場合だけreopen | 方針固定 |

## 現在のCanonical候補比較

```text
A  = identity-only
D1 = identity + thin Business residual meaning + handoff
D2 = D1 - realization review label
D3 = D2 + Step body identity / Step addressability separation
```

D3が現在の主比較対象。

production `ir.World`へはいずれもまだ統合しない。

## D3の候補形

```text
BusinessProcedure
  procedure identity / address
  purpose
  trigger
    epistemic status
    meaning
  actor
    epistemic status
    business label
  ordered steps
    address metadata?  // Procedure-scoped
    body
      businessAction
      explicit semantic refs
```

`realization`はReference authoring / review labelとして残せるが、production Core enumにはしない。

## Handoff necessity / representation

比較:

```text
H0
  handoff authoring-only

H1
  accepted handoffのthin normalized meaningを保持
```

current ReferenceではH0だとcondition / endpoint changeをProcedure body snapshotから観測できない。

したがってH1をprimary candidateとする。ただし専用production `BusinessHandoff` kind採用はまだ決めない。

representation比較:

```text
R1  semantic.Data.Relation再利用   reject
R2  generic Core Edge新設           defer / reject for now
R3  Business semantic boundaryのthin handoff relation  primary
R4  full Business Flow / DFD        separate unresolved pressure
```

既存`semantic.Relation`はData配下でData target / cardinality / semantic attribute keysを持つData Relation専用contractであり、Business handoffを混ぜない。

R3をgeneric Business graphへ拡張しない。

## Step address

```text
S4-A
  normalized Procedure
    ordered Step
      Procedure-scoped local address metadata
      body meaning

S4-B
  Canonical bodyはID-free
  Step local keyはSource Indexだけ
```

H1 + R3を採用候補とする現在、Semantic validationをSource Index依存にしないためS4-Aが有力。

local keyはglobal identityでもruntime identityでもない。

## Normalized Business model pressure

R3 + S4-Aを次のtest-only modelへ統合した。

```text
NormalizedBusinessModel
  Procedures[]
    thin Business meaning
    ordered Step { localKey, body }
  Handoffs[]
    from Procedure/localKey
    to Procedure/localKey
    condition
```

normalize後のreferential validationはこのmodelだけを入力とし、Source Index / file path / fingerprint / Workspaceへ戻らない。

Pressure:

- artifact locator moveでmodel不変
- stale local keyでhandoff fail closed
- local key + endpoint協調renameは成立
- missing endpointをnormalized modelだけでreject
- natural-language condition変更をBusiness semantic relation差として保持
- source/runtime container fieldsを持たない

production `ir.World.Business`等はまだ導入しない。

## Business Flow boundary — X-TEA fallback

H1/R3をfull Business Flowへ統合すべきか迷ったため、設計fallback規則に従い渡辺幸三氏の言説、X-TEA公開説明、XeadModeler sourceへ戻った。

`plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md` E-014で次を記録した。

```text
X-TEA Business Flow
  Process node -> Task
  other participant / storage-like nodes
  DataflowLine
```

historical Business Flowは、単なるProcedure handoff shorthandではない。

TEIでは次を固定する。

```text
Business Handoff
  Procedure/Step -> Procedure/Step + condition

Business Flow / DFD
  activity / participant / information-document-data flow等の別pressure
```

`business_flow_scope_pressure_test.go`はcurrent normalized model / handoff probeがActor / Role / ExternalParty / Process / Data / Information / Document / Store / Payload / FlowType等を所有しないことをnegative Pressureとして固定する。

current handoff validatorがPASSしてもBusiness Flow / DFD Conformanceとはみなさない。

full Business Flow production contractは、realistic ReferenceでProcedure handoffだけではaccepted meaningを保てないEvidenceが出た場合に別途reopenする。

## Application context

C1を実験checkerへ採用した。

```text
single Application owner
  -> production Business Procedureへ入れない

Pattern owner Application
  -> realization / policy traceのauthority

fixture Application
  -> optional authoring / reporting context
  -> absentでもcross-view evaluation可能
  -> unknownでもsemantic ownership authorityにしない
```

C2（fixture field完全除去）はcurrent Evidenceから必要性がないため保留する。

## Lane B独立監査との整合

latest mainのLane B RQ-002監査も独立に次を確認した。

```text
identity-only shell
  insufficient as complete Business meaning

thin normalized Business residual meaning
  strongest current production candidate pressure
  not adopted

Step body meaning
  != Step addressability

thin Business handoff relation
  useful independent semantic pressure
  not adopted
```

Lane Cはこの監査をproduction adoption理由として使わず、Reference Evidenceのcross-lane整合確認として扱う。

## 000039 / Shared Acceptanceとの境界

`ISSUE-000039`は共通revision-aware semantic diff / dependency propagationを将来所有する。

本Issueは共通change graphを先取りせず、Business changeを見失わないmeaning粒度とaddress境界だけをReferenceで明らかにする。

## 非対象

- production Canonical Procedure / Handoff schema採用
- `ir.World.Business`採用
- full Business Flow / DFD Canonical schema採用
- Workflow Engine / retry / timer / queue / saga / scheduler
- generic Core Edge / graph engine
- global / durable Canonical Step ID
- actor -> Authorization Role自動mapping
- `realization` production enum
- generic Uncertainty framework / certainty lattice / approval state machine
- Business Lifecycle 000038実装
- 000039共通change graph実装
- PAS-07 trace正本統合
- Gate B / Gate C完了条件変更

## Verification status

Repository上にはchecker / fixtures / Go pressure testsを実装済み。

一時GitHub Actions workflowは、runが繰り返し`runner_id=0` / `steps=[]`でcheckout前に終了し続けたためmerge対象から除去した。

Remote Desktop経由のlocal executionもオンライン端末不在で実行できなかった。

したがってactual `go test` greenとはまだ主張しない。

PR #482はDraftを維持する。

## 完了条件

1. actual Repository environmentで`gofmt / go vet / go test`がgreen
2. baseline checkerがclean
3. intentional access conflictだけをexpected diagnosticとして再現
4. Business-only wording / uncertainty changeをthin meaning差として再現
5. file move / logical rename / local Step address renameを分類して再現
6. manual / external workをfake Functionなしで保持
7. Pattern-owned meaningを重複しない
8. one Procedure -> multiple Applicationsを単一owner fieldなしで保持
9. Pattern owner Applicationからpolicyをexact trace
10. C1 optional Application contextをactual testsで確認
11. `realization` subtractionをactual testsで確認
12. Step body meaningとhandoff / review addressabilityを分離
13. global Step IDなしで必要addressabilityを説明
14. H0 semantic-loss / H1 relation preservationをactual testsで確認
15. R3 + S4-A normalized modelをactual testsで確認
16. dedicated production kindなしでどこまでBusiness boundaryを表現できるか説明
17. current `order_entry`より一段現実的なmulti-App / manual / external Pressureをactual Evidenceで確認
18. A / D1 / D2 / D3の最小十分性を説明
19. Lane B再監査と矛盾しないproduction ADR候補を作れる状態にする
20. H1/R3をhandoff-onlyとしてactual testsで確認し、Business Flow surfaceを混入させない
21. current handoff EvidenceをBusiness Flow / DFD Conformanceへ過大主張しないことを説明できる

この完了後に初めてproduction Canonical contract ADR候補を作る。
