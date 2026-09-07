# RQ-025 / RQ-026 PAS-07 Evidence Contract

Status: **supporting Lane B Evidence contract / PAS baseline is `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` / Lane C Evidence detail is ISSUE-000042**

## 1. 正本関係

本書は採用済みRQ-025 / RQ-026をPAS-07 Evidenceへ接続するsupporting contractである。

```text
PAS-07 user value / Requirement mapping / Gate relation
  -> plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md

Lane C detailed Reference Evidence inventory
  -> plan/issue/000042-business-procedure-cross-view-reference.md A-set

本書
  -> A-set -> E7 / S7 mapping
  -> A-set detailを逐語複製しない
```

Reference fixture / checker / Canonical pressureはLane Cが所有する。Gate B / Gate C Required Evidenceは変更しない。

## 2. PAS-07の壊してはいけない境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Handoff != full Business Flow / DFD
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Business Procedure != mandatory single Application child / owner
reference target != Application realization
ref kind separation != ref role separation
Validation PASS != whole-business approval
Application Business Result != external-world Observation
Command accepted != physical-world fact observed
physical-world fact observed != accepted responsibility Handoff
Projection != second source of truth
```

利用者価値・代表Scenario・Requirement mappingの正本は主Traceとする。

## 3. Current representation pressure

PAS-07はCanonical `BusinessProcedure` kindを前提にしない。

```text
V0  derived Projection
V1  Business固有meaningを持つauthoring source
V2  durable Canonical Business meaning / identity
```

Lane C current candidate stack:

```text
D3    broader thin Business Step body
D4    smaller evidence-backed Step body
S4-A  Procedure-scoped Step local address
H1/R3 thin normalized Handoff / handoff-only relation
C1    optional reporting Application context
O1    Application Pattern protected-target overlay
```

D4 current minimum:

```text
businessAction
applicationPatternRefs[]   // direct realization only
validationRefs[]
capabilityRefs[]
```

Direct Data / Relation refはminimum必要性未証明。将来accepted sourceに不可欠ならfail closedしてreopenする。いずれもproduction adoptionではない。

## 4. RQ-025 acceptance pressure

- Functionを持たないmanual / external / human judgment / physical Actionを正常なBusiness meaningとして保持する。
- Business actor / jobからPrincipal / Authorization Role / Permissionを生成しない。
- `direct realization != protected target / concern`を保持する。
- Business HandoffをRuntime Workflowへ変換しない。
- purpose / trigger / actor context / manual action / judgment / continuationをPattern-derived扱いしない。
- Procedureをsingle Application containerへ閉じ込めない。

## 5. RQ-026 acceptance pressure

- accepted identityをexactに参照し、名称類似やprefixで補完しない。
- accepted datasource / policy contradictionをdiagnosticとして露出する。
- Business-only / Handoff / target / actuality changeをApplication-owned meaningの偽変更へ拡大しない。
- logical address / Handoff endpointをfile pathから分離する。
- review / approvalをrevision / fingerprintへ束縛する。
- EvidenceなしにProcedure / Handoff / generic ref-role / graph / global Step ID / Workflow / Business Flowをproductionへ固定しない。
- Pattern owner / policyを正しいauthorityから解決し、Legacy policyからBusiness targetをbackfillしない。
- normalized Handoff integrityをSource Index / locatorへ戻さない。
- physical ObservationだけからData ownership / Application Pattern / responsibility Handoffを生成しない。

## 6. PAS-07 Evidence mapping

Lane Cの詳細statusはISSUE-000042 A-setを正本とする。Current A-setはA1〜A79。

| Evidence | Current supporting A-set | Claim |
| --- | --- | --- |
| E7-01 | A2 / A8 / A21 / A25 / A33 / A37 / A67 / A71 / A72 | manual / external / physical meaningをFunctionへ還元せず、targetもrealizationへ誤変換しない |
| E7-02 | A1 / A3 / A4 / A20 / A38 / A39 / A55 / A57〜A59 / A61 / A63 / A66 / A68〜A71 / A75 | exact authority trace、非重複、kind / role / authority direction、synthetic ownership禁止 |
| E7-03 | A5 / A6 / A39 | accepted datasource policy contradictionをbaselineと区別して検出 |
| E7-04 | A9 / A19 / A23 / A28 / A31 / A32 / A45 / A46 / A48 / A49 / A51 / A54 / A65〜A67 / A69 / A72〜A74 / A76 / A78 | Business / Handoff / target / actuality / responsibility meaningを分離 |
| E7-05 | A10 / A11 / A25 / A26 / A37 / A38 / A40 / A44 | reuse、multi-Application、single-owner非依存、optional reporting context |
| E7-06 | A12 / A13 / A22 / A23 / A34〜A36 / A41〜A43 / A45 / A46 / A49 / A64 / A65 / A76〜A79 | source move、Step address、explicit Handoff authority、referential integrity |
| E7-07 | A14 / A15 / A17 / A35 | review continuity、snapshot-bound approval、Acceptance target |
| E7-08 | A7 / A27〜A30 / A35 / A44 / A62 / A68 / A70 / A74〜A79 | fail closed、unknown保持、policy / Observation / adjacencyから未証明meaningをbackfillしない |
| E7-09 | A18 / A20 / A24 / A26 / A30〜A34 / A36 / A40 / A41 / A43〜A64 / A69〜A79 | identity / runtime / enum / ownership / graph / Business Flow / generic ref-roleをEvidenceなしでproductionへ固定しない |
| E7-10 | A16 / A17 + PAS-01 | authoritative file-native sourceをstale baseから守りfresh Accept後にreload / re-deriveする |

同じAが複数境界を支える。mappingはcoverage scoreではない。

## 7. A54〜A79の現在地

```text
A54〜A59
  realization / Authorization / Observation / explicit-ref authority

A60〜A65
  D4 minimum refs + S4-A + H1/R3 composition

A66〜A69
  ref kind != ref role / O1 / D4+O1

A70
  Authorization policy may corroborate O1 target
  but must not author / backfill it

A71
  X-TEA supports Action / Function separation
  but not generic protected-target role taxonomy

A72
  manufacturing / inventory outline reinforces physical/manual + Observation boundary
  without proving candidate-stack insufficiency

A73〜A74
  Command acceptance / read success / positive Observationを分離

A75
  exact TEI identityなしにsecond-corpus ownershipをsynthetic化しない

A76〜A79
  physical fact / Step adjacency / Function sequenceからHandoffを推測せず
  explicit accepted Handoff sourceだけをH1/R3 authorityとする
```

新E7は追加しない。

## 8. O1 / B-F27

```text
role separation is representable
  != protected target is authoritative Business source meaning
  != reload / RQ-002 trace / Shared Acceptance preserves it
```

B-F27はintentionally / conditionally unresolved。

A72ではsecond domainでO1 roleの反復は未証明。generic Role taxonomyへ一般化しない。

## 9. Secondary semantic boundaries

### S7-01 — Validation / business judgment
manual business judgmentとdeterministic Validationを同一Actionへ畳まない。actual execution / approval semanticsはPAS-02側とcross-checkする。

### S7-02 — actor / Authorization

```text
Business actor / job != Principal / Role / Permission
BP-06 supporting authority -> security.authorize
BP-06 protected target     -> O1 pressure
Application Function       -> none
```

Legacy Authorization policyはtargetをcorroborateできるが、author / backfillしない。actual enforcementはPAS-02が所有する。

### S7-03 — Business Result / Observation

```text
Command accepted
  != Observation read succeeded
  != physical fact observed
  != responsibility Handoff accepted
```

runtime `ExecuteObservationRead` / `ReadIdentity`をBusiness/Semantic identityへ昇格しない。A73/A74はmechanical pressureだがPAS-02全体のRuntime Evidenceを代替しない。

## 10. Gate 15 / RRB

```text
15a synthetic multi-Application structural pressure
15b current order_entry social-system realism
15c broader realistic pressure
    = concrete minimum-contract insufficiencyが出た場合だけopen
```

A72〜A79では具体的contract insufficiencyを観測していないため15cは未起動。

```text
Gate 15c not activated
  != realistic Reference maturity established
```

B-F19b status:

```text
RRB-P1 provenance-safe TEI-native second Business source exists
RRB-P2 P1 sourceをsource/test cross-viewへ統合
RRB-P3 actual execution-qualified breadth Evidence
```

Current outlineおよびA73〜A79のtest-only pressureはP1ではない。P1がないためP2でもない。

## 11. Current findings / status

```text
Lane C A-set
  A1〜A79

B-F21 / B-F23 / B-F25 / B-F26 / B-F29 / B-F30
  source/document level Resolved

B-F27 authoritative protected target
  intentionally / conditionally unresolved

B-F28 candidate composition
  source-level Confirmed / actual execution pending

B-F32 outline != RRB-P1
  Confirmed

B-F33 O1 generalization evidence insufficient
  Confirmed negative Evidence

B-F34 Data authority relevance != exact authority existence
  Confirmed deferral boundary

B-F35 H1/R3 authority direction = explicit source only
  source/test-level strengthened

RRB-P1 / P2 / P3
  not achieved

actual gofmt / go vet / go test / checker green
  unresolved

production Canonical contract
  not adopted

Pre-ADR
  Not Ready
```

## 12. Existing PASとの関係

- PAS-01: correction / review / Shared Acceptance。E7-10のend-to-end成立にactual Evidenceが必要。
- PAS-02: Validation / Authorization / Business Result / ObservationのRuntime意味を所有する。
- PAS-04: Targetがaccepted Business / Function / Data meaningを壊さないことへ接続するがGate C Required EvidenceへPAS-07を追加しない。
- PAS-06: Reference / Pattern Knowledge探索と接続可能だが、それだけでBusiness Procedure成立とはしない。

## 13. Requirement -> PAS

| Requirement | Primary PAS | Secondary / cross-cutting |
| --- | --- | --- |
| RQ-025 | PAS-07 | PAS-02 / PAS-04 |
| RQ-026 | PAS-07 | PAS-01 / PAS-02 / PAS-04 |

RQ-021 / RQ-022もcross-cuttingする。正本は主Trace。

## 14. Gate / Stage

PAS-07を理由に現行Gateを変更しない。

```text
Gate A / B / C
  existing definitions unchanged

PAS-07 Reference Evidence
  Gate B Required Evidenceへ追加しない
  Gate C First Web Target Required Evidenceへ追加しない
```

将来「業務システム設計として成熟した」というclaimを置く場合、actual PAS-07 execution EvidenceとRRB-P3相当のrealistic breadthを別途要求し得るが、新Gateは現時点で作らない。

## 15. 更新規則

1. Lane C A-setをdetail正本とし、本書はrange/mappingだけを更新する。
2. 新pressureは新E7を増やす前に既存E7へ吸収できるか確認する。
3. B-F27はactual Evidenceからauthoritative persistenceが必要になった場合だけreopenする。
4. S7-01〜03はPAS-02とcross-checkしRepresentation / Runtimeを混同しない。
5. Pre-ADR readinessはLane C正本からderiveし、Lane Bへ独立Gate inventoryを作らない。
6. RRB-P1 / P2 / P3とGate 15cを同じstatusへ潰さない。
7. Observation / adjacency / Function sequenceからData authorityやHandoffを推論しない。
8. PAS-07 user value / Requirement mapping / Gate影響が変わる場合は主Traceを先に更新する。
