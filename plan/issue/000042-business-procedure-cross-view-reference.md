# 000042 Business Procedure cross-view Reference

Status: In Progress

## 目的

`order_entry`を起点にData / Function-Interaction / Businessの三視点cross-checkをReferenceとしてPressureし、Business Procedureの最小production Canonical contractをEvidenceから決める。

先にWorkflow Engine、full Procedure schema、global Step identity、full Business Flow schema、generic semantic ref bagを固定しない。

## 正式基線

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — RQ-025 / RQ-026
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `docs/business-system-design.md`
- `plan/LANE_B_RQ002_BUSINESS_PROCEDURE_CANONICAL_TRACE_AUDIT_2026-09.md`
- `plan/LANE_B_RQ025_RQ026_TRACE_AUDIT_2026-09.md`
- `plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`
- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`
- `plan/BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_EXPLICIT_REF_AUTHORITY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_D4_REF_KIND_PRESSURE_2026-09.md`

## Lane境界

本IssueはLane CのBusiness Procedure Reference / Evidenceを所有する。

RQ-025 / RQ-026はProduct Requirementとして採用済み。PAS-07 Acceptance trace正本はLane B側へ残す。

Gate B / Gate C scope、Business Lifecycle 000038、共通change graph 000039を本Issueから拡張しない。

## 固定する境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Handoff != full Business Flow / DFD
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Business Procedure != single Application child
Runtime Observation operation != Business / Semantic identity
Data / Function-Interaction / Business = 同じaccepted meaningを相互牽制する視点
```

Business ProcedureにはApplication / Semantic Worldから機械的に導出できないresidual meaningがある。

```text
purpose
trigger / event + epistemic status
actor / job context + epistemic status
manual / external / unresolved Business Action
judgment / continuation
handoff
```

Pattern-owned Data / Relation / Query / Capability / Datasource / Application ownership / datasource policyはnormalized `ir.World`からderiveし、Business sourceへコピーしない。

## Reference

### Primary Procedure

`reference/business-system-design/order-entry-business-procedure.json`

- BP-00〜BP-09
- BP-04 / BP-09はPatternなしのmanual / external Business Action
- BP-05だけがaccepted deterministic Validation authorityをdirect refとして保持
- BP-06は既存Semantic authority `security.authorize`をdirect refとして保持
- BP-08はObservation intentionをBusiness wordingとして保持し、runtime Observation identityをfake Semantic refへ昇格しない

### Follow-up Procedure

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

## Gate 15 Reference / Evidence分類

Gate 15a / 15b / 15cはAcceptance IDではなく、上記Pressureを評価するReference/Evidence分類として扱う。

### 15a — multi-Application structural pressure

temp Workspaceで:

```text
order_entry interaction
  -> synthetic fulfillment_console interaction
  -> manual warehouse coordination
  -> external delivery continuation
```

を同一Procedureへ共存させる。

Business Procedureを単一Application画面操作列へ縮めない構造Pressureであり、production業務仕様ではない。

### 15b — social-system realism pressure

current production `order_entry`だけで:

```text
sales_core      read-write / managed-here
master_data     read-only / external-observed
customer ref    datasource + Contract Query
BP-04           human judgment
BP-06           Authorization
BP-07           Application Command
BP-08           separate Observation intention
BP-09           manual / external continuation
```

を同じBusiness Procedure文脈に保持する。

### 15c — conditional broader realistic pressure

15a + 15bでminimum Canonical contract sufficiencyを説明できない具体的不足が出た場合だけ、realistic second Application / broader corpus pressureを追加する。

Application数を成熟度KPIにしない。

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
- explicit refsはEvidence-backedなBusiness dependencyだけを保持
- 名称類似で補完しない
- Pattern ID文字列からownerを推測しない
- Pattern + owner Applicationをnormalized Worldからexact解決
- Projectionだけでaccepted authorityを拡張しない
- `realization` labelからref kindを推測しない
- runtime Observation APIをSemantic identityへ昇格しない

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
| A37 | multi-App + manual + external coexistence | Gate 15a構造Pressureとして実装済み |
| A38 | Application associationはWorld解決でderiveし、名前からguessしない | 実装済み |
| A39 | access policyはPattern owner Applicationを使う | checker修正 + 回帰Pressure実装済み |
| A40 | fixture Application context != Procedure owner | 実装済み |
| A41 | normalized handoff採用時はStep address解決をSource Index依存にしない | S4-A優先Pressure |
| A42 | Step local key renameをBodyMeaningChangedと分離 | test-only実装済み |
| A43 | global Canonical Step IDをdefault導入しない | 方針固定 |
| A44 | primary Application contextをsemantic prerequisiteからsubtract | C1をcheckerへ実装済み、actual runner待ち |
| A45 | H0ではhandoff condition / endpoint changeをnormalized diffから失う | 反証Pressure実装済み |
| A46 | H1でaccepted handoff relation changeを保持 | primary candidate / test-only |
| A47 | Data `semantic.Relation`流用 / generic Core Edge新設を避ける | R1 reject / R2 defer |
| A48 | Business-side thin relation R3 | primary pre-ADR candidate |
| A49 | R3 + S4-AをSource Index非依存validate | test-only実装済み |
| A50 | X-TEA fallbackでBusiness Flowが独立Dataflow pressureを持つことを確認 | E-014追加済み |
| A51 | R3をhandoff-only relationへ限定 | negative Pressure実装済み |
| A52 | handoff validation PASSをBusiness Flow / DFD Conformanceと呼ばない | 境界固定 |
| A53 | full Business Flowは別pressureが出た場合だけreopen | 方針固定 |
| A54 | `realization` label変更でcross-view diagnostics不変 | test-only実装済み |
| A55 | BP-06 Authorizationは`security.authorize`へexplicit trace | 実装済み |
| A56 | BP-08 Observationはruntime read identityをSemantic refへ昇格しない | negative Pressure実装済み |
| A57 | explicit refs are evidence-driven, not symmetry-driven | 方針固定 |
| A58 | BP-03 line reviewとBP-05 Validation authorityを分離 | Reference修正 + test-only実装済み |
| A59 | Pattern-owned CapabilityをBusiness sourceへ重複コピーしない | negative Pressure実装済み |
| A60 | current Referenceでdirect Data / Relation ref minimum必要性は未証明 | Evidence固定 |
| A61 | D4 bodyをPattern / Validation / Capability refsへ縮小比較 | test-only実装済み |
| A62 | future direct Data / Relation refをD4がfail closed / reopen | test-only実装済み |
| A63 | ref authority kindをkind-less bagへ畳まない | test-only実装済み |
| A64 | D4 bodyとS4-A addressabilityを分離 | test-only構造固定 |
| A65 | D4 + S4-A + H1/R3を合成してnormalized validation | test-only Pressure追加 |

## Candidate比較

```text
A  identity-only
D1 identity + thin Business residual meaning + handoff
D2 D1 - realization review label
D3 D2 + Step body identity/addressability separation
D4 D3 body - current未証明のdirect Data/Relation ref slots + fail-closed reopen
```

D3はbroader Step body comparison candidate、D4はsmaller evidence-backed Step body candidate。

D4自身はStep local addressを持たない。S4-Aを別layerとして合成する。

production `ir.World`へはいずれもまだ統合しない。

## Current composition pressure

```text
D4 Step body
  businessAction
  applicationPatternRefs[]
  validationRefs[]
  capabilityRefs[]

S4-A
  Procedure-scoped Step local address

H1 / R3
  Procedure/localStep
    -> Procedure/localStep
    + condition
```

`d4_normalized_composition_pressure_test.go`で、bodyを膨らませずこの3関心を同時に成立させるPressureを置く。

## Business Flow boundary — X-TEA fallback

X-TEA / XeadModelerへ戻ったEvidenceから、historical Business FlowはProcess(Task) node、participant / storage-like node、DataflowLine等を持つ別pressureである。

```text
Business Handoff
  Procedure/Step -> Procedure/Step + condition

Business Flow / DFD
  activity / participant / information-document-data flow等の別pressure
```

Current Handoff validator PASSをBusiness Flow / DFD Conformanceとはみなさない。

## Application context

```text
single Application owner
  -> production Procedureへ入れない

Pattern owner Application
  -> cross-view policy traceのauthority

fixture Application
  -> optional Reference reporting context
  -> semantic ownerではない
```

## Authorization / Observation boundary

```text
BP-06 Authorization
  existing Semantic authorityあり
  -> security.authorize explicit ref

BP-08 Observation
  Business intentionあり
  independent Semantic authorityは未証明
  -> Business wordingを保持
  -> runtime ExecuteObservationRead / ReadIdentityをrefにしない
```

## 非対象

- production Canonical Procedure / Handoff schema採用
- `ir.World.Business`採用
- full Business Flow / DFD Canonical schema採用
- Workflow Engine / retry / timer / queue / saga / scheduler
- generic Core Edge / graph engine
- generic `semanticRefs[]` / kind-less Canonical ID bag
- global / durable Canonical Step ID
- `realization` production enum
- generic Uncertainty framework
- Observation専用Canonical identityの先行導入
- Business Lifecycle 000038実装
- 000039共通change graph実装
- Gate B / Gate C完了条件変更

## Verification status

Repository上にはchecker / fixtures / Go pressure testsを実装済み。

GitHub Actionsは過去にrunner assignment前で終了し、一時workflowはmerge対象から除去した。Remote Desktop経由のlocal executionもオンライン端末不在で実行できなかった。

したがってactual `gofmt / go vet / go test` greenとはまだ主張しない。

PR #482はDraftを維持する。

## 完了条件

1. actual Repository environmentで`gofmt / go vet / go test`がgreen
2. baseline checkerがclean
3. intentional access conflictをexpected diagnosticとして再現
4. Business-only wording / uncertainty changeをmeaning差として再現
5. file move / logical rename / Step local address renameを分類
6. manual / external workをfake Functionなしで保持
7. Pattern-owned meaningを重複しない
8. one Procedure -> multiple Applicationsをsingle ownerなしで保持
9. Pattern owner Applicationからpolicyをexact trace
10. `realization` subtraction / diagnostic independenceをactual testsで確認
11. Step body meaningとaddressabilityを分離
12. global Step IDなしで必要addressabilityを説明
13. H0 semantic-loss / H1 relation preservationをactual testsで確認
14. R3 + S4-A normalized modelをactual testsで確認
15. Gate 15a / 15bをactual Evidenceで確認し、15cは不足時だけ追加
16. Authorization / Observation非対称境界をactual testsで確認
17. BP-03 / BP-05 authority分離をactual testsで確認
18. D3 / D4の最小十分性をcurrent + next realistic Referenceで説明
19. D4 future Data / Relation ref fail-closedをactual testsで確認
20. authority kindをgeneric ref bagへ失わない
21. D4 + S4-A + H1/R3 compositionをactual testsで確認
22. Lane B再監査と矛盾しない
23. H1/R3をhandoff-onlyとして保ちBusiness Flow surfaceを混入させない

この完了後に初めてproduction Canonical contract ADR候補を作る。
