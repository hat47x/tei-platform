# 000042 Business Procedure cross-view Reference

Status: In Progress

## 目的

`order_entry`を使い、Business ProcedureをCanonical schemaへ先に昇格させずに、Data / Function-Interaction / Businessの三視点cross-checkを実行可能なReferenceとして再現する。

正式基線:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — RQ-025 / RQ-026
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/business-system-design.md`
- `plan/ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md`
- `plan/BUSINESS_PROCEDURE_IDENTITY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`
- `plan/UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`
- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`

## Lane境界

本IssueはレーンCのReference / Evidenceを所有する。

RQ-025 / RQ-026はmainのProduct Requirementとしてすでに採用済みである。一方、PAS-07のAcceptance trace統合はレーンB側で扱う。

本IssueではRequirement / PAS正本を追加・再定義せず、Referenceで要求の設計圧力を実証する。

## 固定する境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Data / Function-Interaction / Business = 同じ業務システムを相互牽制する視点
```

Business ProcedureにはApplication / Semantic Worldから導出できない意味がある。

```text
purpose
trigger / event
actor / job context
manual / external / unresolved Business Action
judgment / continuation
handoff
```

したがってBusiness Procedure fixture全体をderived artifactとは扱わない。

## Authoring / Derived boundary

Reference Business Procedure fixtureは次を明示する。

```text
artifactRole = reference-authoring-fixture
canonical = false
procedure.identityStatus = reference-local-stable
```

`procedure.id`はReference corpus内でreview target、handoff endpoint、cross-referenceを安定して指すauthoring keyである。

top-level Canonical `Procedure`を採用したことを意味しない。

一方、Application Patternが所有する次の意味はnormalized Worldからderivedする。

```text
Data
Relation
Query
Capability
Datasource
```

Business Action × Pattern Matrix、Pattern-owned meaning rows、cross-view diagnosticはderived Projectionとし、authoring fixtureへ逆書きしない。

## Reference構成

### Primary Business Procedure

`reference/business-system-design/order-entry-business-procedure.json`

BP-00〜BP-09を持つ。

少なくともBP-04 / BP-09はApplication Patternを持たないmanual / external Business Actionとして保持する。

### Follow-up Business Procedure

`reference/business-system-design/order-entry-follow-up-procedure.json`

別Business文脈から同じ`order_entry.order_record`を利用する。

multiple Procedure -> same Patternを一つのBusiness Actionへ畳まない。

### Reference set / handoff

`reference/business-system-design/order-entry-procedure-reference-set.json`

Procedure keyとartifact locatorを分離し、次のhandoffを保持する。

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
```

handoffはfile pathではなくProcedure key + Step keyを参照する。

pure file moveではlocator mappingだけを更新し、handoff semantic endpointを変えない。

logical Procedure rename後にlinkを更新しなければfail closedとする。

### Intentional cross-view defect

`reference/business-system-design/order-entry-customer-edit-conflict.json`

```text
BP-02
  Pattern = order_entry.customer_reference
  Pattern-derived datasource = master
  accepted Application access = read

experiment probe
  requested access = read-write
```

期待diagnostic:

```text
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
severity = architecture-policy
```

このdiagnosticはCustomer更新を永久に禁止するBusiness ruleではない。

現在acceptedされているApplication contractとcandidate Interactionが矛盾するため、Projectionだけでwrite authorityを拡張してはいけないことを示す。

## Cross-view checker

`tools/reference-business-procedure-check`は独自に`.tei`をparseせず、既存`workspace.Load()`が返すnormalized `ir.World`を利用する。

```text
Business Procedure Reference authoring fixture
                  +
          reference/workspace
                  ↓
           workspace.Load()
                  ↓
          normalized ir.World
                  ↓
       experiment-only checker
                  ↓
 derived Matrix / diagnostics
```

explicit referenceはexact identityでのみ解決する。

名称類似からRelation / Capability / Procedure linkを補完しない。

## Acceptance

### A1 exact reference resolution

Application Pattern / direct semantic referenceをnormalized Worldでexactに解決する。

### A2 no fake Function

manual / external Business ActionはApplication Pattern / Capabilityなしでも正常状態とする。

### A3 Pattern-owned meaning is derived

少なくとも次をfixtureへ二重記述せずPatternから導出する。

```text
BP-01 order_entry.order_record
  -> Data sales.order

BP-02 order_entry.customer_reference
  -> Relation sales.order.customer
  -> Datasource master

BP-03 order_entry.order_lines
  -> Relation sales.order.lines

BP-07 order_entry.submit_order
  -> Capability sales.order.submit
```

### A4 Procedure-side explicit semantic reference

Patternから導けないProcedure-side Validation等はexplicit referenceとしてexactに解決する。

### A5 read-only conflict

intentional probeが`BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT`を再現する。

### A6 baseline clean

同じaccepted `master=read`を利用するbaselineではaccess conflictを発生させない。

### A7 fail closed decode

fixture / probeの未知field、未対応schemaVersion、別artifact roleを黙って解釈しない。

### A8 Business authoring residual

BP-04 / BP-09、purpose、trigger、actor候補等をPattern-derived Projectionから再生成できるものとして扱わない。

### A9 Business-only change separation

Business Action文言だけを変更した場合、Business-side Projectionには差分を出すがPattern-derived Application meaningを偽って変更しない。

### A10 multiple Procedure -> same Pattern

同じPatternを利用する複数Procedureを区別してtraceできる。

### A11 Reference-local stable key

review / handoff用に`procedure.id`をReference-local stable authoring keyとして使える。

このkeyはCanonical World identityではない。

### A12 handoff survives source move

Procedure keyを保ったpure file moveではhandoff semantic endpointを保つ。

### A13 logical rename fails closed

Procedure keyをlogical renameしてlinkを更新しなかった場合、stale handoffを類似名称から自動修復しない。

### A14 review target continuity

同じProcedure keyを保ったcontent changeではreview target continuityを保つ。

### A15 approval remains snapshot-bound

approval Evidenceは少なくとも次へ束縛する。

```text
Procedure key
Repository revision
Artifact fingerprint
Source locator
Decision
```

同じProcedureでも別revision / fingerprintへ旧approvalを暗黙carry-forwardしない。

pure file moveでfingerprintが同じでも別revisionへ自動継承しない。

### A16 no Canonical expansion by assumption

本Issueの成功だけを根拠にCanonical `Procedure` / `Step`を追加しない。

## Identity pressureの現在地

最初のReferenceでは、基本cross-viewだけならdurable Procedure identityは不要だった。

review continuity / handoffを加えた結果、安定した**authoring-level logical key**には具体的価値があることが確認できた。

現時点の判断:

```text
Reference-local stable Procedure key
  required/useful for review + handoff

Canonical Procedure identity
  not yet required by Evidence
```

identityの責務を混同しない。

```text
A. Canonical business semantic identity
B. Reference-local authoring identity
C. Source / Evidence locator
D. Repository revision provenance
```

## 000039 / Shared Acceptanceとの境界

`ISSUE-000039`は将来のrevision-aware semantic diff / dependency propagationを所有する。

本Issueはchange graphを先取り実装しない。

今回固定するのは、000039が利用できるidentity区別である。

```text
same Procedure key + changed revision/fingerprint
  = same review target, changed source snapshot

changed Procedure key
  = logical authoring identity change
```

Shared Acceptanceの実装も本Issueでは完成させない。

次段ではBusiness Procedureをfile-native accepted authoring sourceへ進めた場合に、Reference-local keyでreload / Source Index / Acceptanceが成立するかを圧力試験する。

## 非対象

- Canonical `Procedure` / `Step`
- Workflow Engine / retry / timer / queue / saga / scheduler
- Business actorからAuthorization Roleへの自動mapping
- Business Lifecycle 000038のstate transition実装
- 000039の共通change graph実装
- PAS-07 trace正本の統合
- Gate B / Gate C完了条件の変更

## 現在のVerification status

Repository上には次を実装済み。

- primary / follow-up Business Procedure authoring fixture
- Procedure Reference set / handoff
- intentional access-conflict probe
- cross-view checker
- Pattern-owned meaning derivation
- strict decoder
- baseline / conflict / no-fake-function / non-duplication / no-name-guess tests
- Source Index pressure tests
- authoring residual / Business-only change tests
- review continuity / handoff identity pressure tests
- PR pathに連動したGitHub Actions workflow

GitHub-hosted Actionsは現時点までrunnerが割り当てられず、`runner_id=0` / `steps=[]`でcheckout前に終了している。

これはコードのPASS / FAIL Evidenceには数えない。

したがって本Issueは`In Progress`、PRはDraftのまま維持する。

## 完了時のEvidence

- Reference authoring fixtures
- Procedure Reference set / handoff
- intentional conflict probe
- checker source
- cross-view / identity / review / handoff tests
- Reference README / pressure documents
- actual Repository runnerによる`gofmt / go vet / go test / baseline / conflict` green verification

actual runner green後も、Canonical Procedure採用は別判断とする。
