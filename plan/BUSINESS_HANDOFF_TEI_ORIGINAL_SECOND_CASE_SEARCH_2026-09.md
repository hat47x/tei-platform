# Business Handoff TEI-original Second Case Search — 2026-09

Status: **Lane C / negative Evidence / existing-corpus search closed / no second realistic Handoff fixture yet / no production schema adoption**

## 1. 目的

Pre-ADR Gate Cには:

```text
second realistic Handoffで
from / to / condition sufficiencyを確認
```

が残っている。

Watanabe external evidenceからcaseをコピーせず、まずcurrent TEI Repositoryに**CRUD / Handoff pressureとは独立した理由で既に存在するBusiness scenario**がないかを再探索する。

見つからない場合、matrixを埋めるためだけにsecond fixtureを発明しない。

## 2. Qualifying条件

Second Handoff caseとして採用するには少なくとも:

1. Handoff検証以外の独立したBusiness requirementがある。
2. source responsibilityとtarget responsibilityをBusinessとして説明できる。
3. 単なるStep adjacency / Function sequence / state transitionではない。
4. actor / trigger / conditionを必要に応じてreviewできる。
5. exact TEI Procedure / Step addressを持てる、またはその不足自体がEvidenceになる。
6. intentional defect / negative probeをaccepted Business meaningへ格上げしない。
7. Deployment / Database schema lifecycle / Runtime takeoverをBusiness Handoffへ読み替えない。
8. synthetic scenario自身が「production業務仕様ではない」と宣言している場合、そのscenarioからconcrete Handoff authorityを生成しない。

## 3. Current Reference Application inventory

Current `reference/workspace/applications/`のApplication treeは:

```text
order-entry
```

だけである。

Current `reference/business-system-design/`も:

```text
order-entry-business-procedure.json
order-entry-follow-up-procedure.json
order-entry-procedure-reference-set.json
order-entry-customer-edit-conflict.json
order-entry-protected-target-overlay.json
```

を中心とし、独立した別Business domain / responsibility chainをauthorするfixtureはない。

同directoryのREADME自身も、今後のpressureとして:

```text
複数Subject Area / Subsystemを跨ぐhandoff
外部contractや長期reuse
```

を挙げている。これはcurrent accepted second caseの存在ではなく、future pressureの明示である。

## 4. Candidate audit

### Candidate A — BP-08 -> BP-09

Current primary Procedure:

```text
BP-08
  verify relevant external state when continuation depends on actual state

BP-09
  continue the surrounding business work after the application interaction
```

はBusiness上の重要な境界を含む。

しかし:

```text
Step adjacency
  != accepted responsibility Handoff
```

である。

BP-08の次にBP-09が記載されていることだけからHandoffを作ると、Lane C自身が固定したnon-inference boundaryを破る。

Current: **reject as second Handoff authority**。

### Candidate B — FU-01 -> FU-02

Follow-up Procedure内にも:

```text
FU-01
  inspect current order information

FU-02
  decide and continue surrounding follow-up work
```

がある。

これもProcedure内Step sequenceであり、separate responsibility relationのaccepted sourceはない。

Current: **reject as second Handoff authority**。

### Candidate C — customer edit conflict -> master-data responsibility

`order-entry-customer-edit-conflict.json`は:

```text
order_entry.customer_reference
requested read-write
```

というintentional architecture defect probeである。

一見:

```text
sales operator detects customer change need
  -> master-data steward handles it
```

のようなHandoffを想像できる。

しかしcurrent probeがauthoritativeに示すのは:

```text
order_entryからread-writeを要求するとaccepted datasource contractと矛盾する
```

ことだけである。

Master-data steward、change request、external ownership workflow等はaccepted sourceにない。

Defectを正常業務へ反転させない。

Current: **reject / independent master-data Business requirementが生じた場合のみreopen**。

### Candidate D — ISSUE-000038 Business Lifecycle

`ISSUE-000038`はcurrent `order_entry`の:

```text
pending -> submitted
```

Lifecycleを候補とする。

これはBusiness state / transition semanticsの独立pressureであり重要だが、Issue自身も汎用Workflow / BPMNを非目標とする。

```text
state transition
  != responsibility Handoff
```

であり、同じorder_entry vertical sliceをsecond Handoff caseへ数えない。

Current: **not qualifying**。

### Candidate E — Deployment / observed runtime scenarios

Referenceにはdeployment state / durable run等もある。

しかし:

```text
Deployment / Runtime ownership / takeover
  != Business Procedure responsibility
```

である。

PAS-05 / Runtime lifecycleをPAS-07 Business Handoffへ流用しない。

Current: **reject**。

### Candidate F — Watanabe Business Flow patterns

External evidenceには:

```text
要求の認識と報告
作業指示の立案と発行
作業の実施と報告
実績の集計と分析
```

というrealistic pattern pressureがある。

しかしexact TEI Procedure / actor / event / condition authorityではない。

Current: **pressure seed only / not TEI-original fixture**。

### Candidate G — RQ-025 / RQ-026 requirement-level handoff allowance

`requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`では、RQ-025の解釈としてBusiness Procedureが必要に応じて:

```text
purpose
trigger / event
actor
business action
manual / external work
Application Function利用
handoff
completion / continuation
```

を説明することを明示している。

これはHandoffがBusiness Procedure meaningとして正当であることを支える独立Requirementであり、H1/R3を単なるfixture都合にしない。

しかしRequirementがauthorしているのは**表現可能性・分離原則**であって、第二の具体的Business responsibility chainではない。

Requirementだけから:

```text
source actor / responsibility
source Procedure / Step
target actor / responsibility
target Procedure / Step
continuation condition
```

を生成することはできない。

RQ-026もData / Function-Interaction / Business Procedureを相互検証するviewとして要求するが、特定の責任移譲eventをauthorしない。

したがって:

```text
Requirement says Handoff may be needed
  != Requirement authors a concrete Handoff instance
```

である。

Current: **requirement-level corroboration only / not a second concrete TEI-original Handoff case**。

### Candidate H — realistic multi-Application / fulfillment scenario

`BUSINESS_PROCEDURE_REALISTIC_MULTI_APPLICATION_SCENARIO_PRESSURE_2026-09.md`にはtest-onlyで:

```text
accepted order interaction
  ↓
review whether the accepted order can proceed through fulfillment
  -> fulfillment_console.order_record
  ↓
coordinate physical preparation of the order with warehouse staff
  -> manual / external
  ↓
continue shipment-related work with the external delivery party when required
  -> manual / external
```

という、responsibility transferを想起させるscenarioがある。

しかし同文書自身が:

```text
このscenarioはproduction業務仕様ではない
第二Applicationはsynthetic
```

と明記している。

その目的はBusiness Procedureがmulti-Application / manual / physical / external workを保持できるかの構造Pressureであり、warehouse responsibilityやdelivery-party responsibilityの具体的accepted Handoffをauthorすることではない。

したがって、このscenarioから:

```text
fulfillment review -> warehouse preparation
warehouse preparation -> external delivery
```

等のHandoffを新たに切り出すと、test apparatusが自分自身のsecond Handoff Evidenceを生成する循環になる。

Current: **reject as concrete second Handoff authority / structural scenario only**。

## 5. Search result

Current Repositoryでqualifyingなsecond Handoffは:

```text
TEI-original independent second Handoff case
  = none

existing accepted corpusからのsafe promotion candidate
  = none
```

ここまでに確認した範囲は:

```text
current Business-system-design fixtures
current explicit Handoff / adjacent Steps
intentional cross-view defect
Business Lifecycle
Deployment / runtime scenarios
RQ-025 / RQ-026
Watanabe external patterns
realistic multi-Application synthetic pressure
```

である。

この探索結果により、**existing-corpus searchは一旦closed**とする。

これはH1/R3が十分だという証明ではない。

意味は:

> current accepted sourceを捻じ曲げずにsecond realistic Handoffへ使える独立caseはRepository内にまだない。

である。

### 5.1 PR #482 merge後のdelta scan

Existing-corpus searchをclosedした後もRepositoryは進むため、全corpusを毎回再探索するのではなく、**accepted mainへ新しく入った差分だけ**をreopen trigger候補として確認する。

PR #482 merge commit:

```text
29674c63583065086258ab53d7e8fc2a46717a17
```

から2026-09-09のpost-merge Lane C follow-upを含むmainまでのdeltaを確認した。

この期間のmeaningful non-Lane-C changesは主に:

```text
Gate B baseline source qualification
  -> workflow_dispatch追加

ADR-0026
  -> PostgreSQL partition childをprovider-owned realizationとして
     generic Schema Driftから分離

ADR / status maintenance
```

である。

#### Gate B workflow delta

Gate B workflowはsource qualificationの実行経路を追加するが、Business requirement / responsibility boundaryをauthorしない。

```text
execution capability
  != Business responsibility Handoff
```

Current: **not qualifying**。

#### ADR-0026 partition realization delta

ADR-0026は:

```text
provider-owned partition realization
  != generic Physical Schema convergence subject
```

というPhysical Schema / provider boundaryをauthorする。

そこにはparent / child relation、provider ownership、destructive DDL safety等があるが、それを:

```text
source Business responsibility
  -> target Business responsibility
```

へ読み替える根拠はない。

```text
provider relation edge
  != Business Procedure Handoff

schema lifecycle / realization responsibility
  != Business responsibility transfer
```

Current: **reject as second Business Handoff authority**。

#### Lane C follow-up delta

Standalone Handoff source uniqueness、inventory/readiness/runbook更新は、既存H1/R3 contractをPressure / documentするLane C自身の成果物である。

これらから第二のHandoff instanceを作ると:

```text
Handoff contract work
  -> second Handoff Evidence
```

という循環になるため、当然ながらindependent Business requirementには数えない。

Current delta result:

```text
new independent Business requirement
  = none

new accepted Business responsibility chain
  = none

new qualifying second Handoff candidate
  = none
```

したがってBlocker Bは、既存corpusの検索からは未充足のままである（2026-09-21に、既存corpusではなくTEI-originalの二つ目のReferenceをauthorしてEvidenceを取得した。[Pre-ADR Audit](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)§17）。

### 5.2 Existing-corpus search reopen trigger

今後mainが進むたびに全Repositoryを無条件再探索しない。

次のいずれかがaccepted mainへ入った場合だけ、second-case searchをdelta reopenする。

```text
new / changed Business requirement
new Business Procedure / Reference seed
new accepted manual / external responsibility boundary
new actor/job responsibility separation
new external contract / continuation responsibility
new domain scenario that is explicitly production/reference meaning rather than synthetic Pressure
```

一方、次だけではreopenしない。

```text
docs formatting / status maintenance
CI / workflow execution changes
Physical Schema / Deployment / Runtime lifecycle changes
Lane C Handoff tests / Pressure自身の追加
synthetic scenarioの拡張
```

これは探索を省略するためではなく、**Evidence acquisition blockerと一般Repository churnを分離するため**の運用境界である。

## 6. Negative Evidenceとしての価値

Second case不足を埋めるためだけに次を行わない。

- BP-08 -> BP-09 adjacencyをHandoff化
- FU-01 -> FU-02 adjacencyをHandoff化
- customer-edit defectからmaster-data workflowを発明
- Lifecycle transitionをHandoff化
- Deployment ownershipをBusiness Handoff化
- Watanabe patternをTEI fixtureへコピー
- RQ-025の一般的handoff許容から具体的actor / responsibility / conditionを生成
- synthetic fulfillment / warehouse / delivery scenarioからaccepted Handoffを逆生成
- provider-owned Physical Schema relation / lifecycleをBusiness Handoffへ読み替える

この抑制により:

```text
explicit Handoff authority only
```

というcurrent contractを、テスト件数のために自ら破壊しない。

## 7. Actual green後の次のcase authoring protocol

Gate A actual green後は「既存fixtureからsecond caseを探す」ことを繰り返さない。

次の順序で、Handoff検証とは独立した**small TEI-original Business requirement / Reference seed**を先に成立させる。

```text
independent Business need / responsibility boundary
  ↓
Business requirement / Reference seedを先に記述
  ↓
source responsibility / target responsibility / continuation conditionをreview
  ↓
minimal Procedure sources
  ↓
explicit Handoff candidate
  ↓
from / to / conditionだけでlosslessかPressure
```

候補domainはあらかじめ決定しない。たとえば:

- external/master-data maintenance ownership
- approval / exception handling
- fulfillment / physical operation responsibility
- reporting / work-instruction responsibility

は**探索方向**であってaccepted meaningではない。

特に:

```text
Handoff fieldを検証したい
  ↓
それらしい業務を作る
```

の順序は禁止する。

Requirement / Reference seedを作った時点で、HandoffなしでもそのBusiness needを説明する独立価値がなければsecond caseとして採用しない。

## 8. Second-case review questions

新しいindependent Business seedが成立した場合だけ、次を確認する。

1. `from {procedure, step}` はsource responsibilityをlosslessに指せるか。
2. `to {procedure, step}` はtarget responsibilityをlosslessに指せるか。
3. natural-language `condition`だけでcontinuation conditionを区別できるか。
4. Procedure `trigger`とHandoff `condition`は別authorityのままでよいか。
5. actor/job labelが変わってもendpoint identityを保てるか。
6. event/document identityを追加しないと具体的lossが起きるか。
7. thin Handoffでは不足しfull Business Flow / DFDが必要になる具体的lossがあるか。

Lossが出た項目だけをschema reopen候補にする。

## 9. Current Gate C interpretation

Current Handoff Evidenceは:

```text
one explicit current Handoff
  + exact endpoint / condition pressure
  + actor / trigger independence structural pressure
  + requirement-level Handoff legitimacy
  + Watanabe external corroboration
  + existing-corpus negative search
  + post-merge Business-delta reopen rule
```

まで進んだ。

残るGate C blockerは:

```text
actual runner green
+ independent Business requirementから生じたsecond realistic TEI-original Handoff
```

であり、second caseを探す過程から新しいHandoff field不足はまだ発見されていない。

## 10. Current decision

```text
second realistic Handoff
  = unresolved

existing-corpus search
  = closed / no qualifying case

post-merge delta scan
  = no qualifying Business delta

reopen trigger
  = accepted Business requirement / Reference / responsibility delta only

next action before actual green
  = do not manufacture fixture
  = inspect only qualifying Business deltas when they appear

next action after actual green
  = author independent small Business requirement / Reference seed first

H1/R3 expansion because second case is absent
  = reject

Pre-ADR Gate C
  = Not Ready
```
