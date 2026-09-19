# Gate C Evidence Contract

## 目的

この文書は、第一Web Targetが「表示できた」ことではなく、Gate Bを通過したApplication meaningをTargetへProjectionし、その意味・操作・視覚品質を再確認可能なEvidenceとして残せたことをもってGate Cを判定する契約です。

Gate Cでは、Driverと第一Web Targetが同じCanonical Application上の意味を共有し、そのうえで何がTarget-neutralで、何がWeb固有だったかを説明できる必要があります。

この文書はWeb framework、browser automation、GUI agent、VLM、visual test製品を標準化しません。どの実装技術を選んでも保持すべきEvidenceの意味を定めます。

- Product Acceptance Scenario: **PAS-04 primary**
- inherited baseline context: PAS-02 / PAS-03

関連:

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
- GitHub Issue #70 / #259 / #270 / #291

## Gate CはHandover Baselineから開始する

第一Web Targetは、曖昧な「最新Driver」から開始しません。

Gate B Readyで固定したHandover Baselineを`B`、そのrepository revisionを`R`とします。

```text
Gate B Ready Evidence
        ↓
Handover Baseline B
  └─ baselineRepositoryRevision = R
        ↓
First Web Target implementation
  └─ targetBuildRevision = T
        ↓
Gate C Evidence
```

通常、Web Targetの実装commitを含むため`T`は`R`より後のrevisionになります。したがって、Gate C Evidenceで`repositoryRevision = R`をTarget buildへ強制しません。

代わりに、次を分けて追跡します。

### Handover Baseline identity

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- Application / Pattern / Data / Relation / Query / Capability等のCanonical references
- Driver Scenario identity
- Gate B Evidence references
- unresolved

### Target build identity

- `targetBuildRevision = T`
- Target Adapter / Host identity
- build / run identity
- Targetで実際に利用したCanonical subject

### RとTの関係

`T`にWeb Adapter、Host、Projection等を追加しても、Handover Baselineが表すApplication meaningを黙って変更しません。

Gate C進行中に`order_entry`等のbaseline subjectの意味変更が必要になった場合は:

1. Web固有fieldやruleで不足を補わない
2. authoritative sourceへ戻って修正する
3. 必要なDriver / Gate B Evidenceを再確認する
4. 新しいHandover Baseline `B'` / baseline revision `R'`を発行する
5. 以後のGate C Evidenceは`B'`へ追跡する

旧`B`のEvidenceを新baselineの成功へ黙って読み替えません。

## Gate Cで分ける三種類のEvidence

第一Web Targetの評価を一つのscoreや一つの「成功」に畳みません。

```text
Handover Baseline B / meaning at R
                │
                ├─ Semantic Conformance
                │
Target build T  ├─ Interaction / Functional Evidence
                │
                └─ Visual Fidelity Evidence
```

### 1. Semantic Conformance

Web TargetがHandover BaselineのCanonical meaningを保持しているかを確認します。

少なくとも次を対象にします。

- Application Pattern identity
- Data / Relation / Query / Datasource identity
- read / read-write境界
- Validation rule
- Authentication / Authorization
- Capability / Binding
- Business Result / named Outcome
- Execution Error
- Command / Observation
- Source-to-Canonical Traceability

Semantic ConformanceはGate Cの必須条件です。

DOM、route、HTTP status、component名等の一致ではなく、Handover Baselineに含まれるApplication meaningとの対応を確認します。

### 2. Interaction / Functional Evidence

利用者がTarget build `T`上でReference Scenarioを実行できるかを確認します。

browser automation、GUI agent、人による再現手順等は実行手段です。どの手段でも、何のHandover Baseline / Application intentを実行したかへ戻れる必要があります。

#### Replayable Behavior

Functional Evidenceは、一回のscreen snapshotや最終DOMだけでなく、必要に応じて再実行可能なbehaviorとして残せます。

```text
Handover Scenario / Semantic Operation
        ↓
behavior precondition
        ↓
Target-specific interaction realization
        ↓
Business Result / Execution Error / Observation
        ↓
Functional conformance result
```

behaviorをTarget固有のselector、route、coordinate、component identity等で実現しても、それらをCanonical interaction identityへ昇格しません。

単一behaviorの成功で、複数featureにまたがるcumulative workflow全体の成立を代替しません。Reference Scenarioが複数段の操作を要求する場合は、その深さを保ったreplayもEvidenceとして確認します。

異なるbrowser automation / GUI agent / Targetから得たeventを比較するため、derivedなnormalized event / behavior recordを作ることはできます。ただしraw Target Evidenceへ戻れるprovenanceを保持し、normalizationで失われるTarget固有差を成功へ畳みません。Universal Event ModelをGate C前提にはしません。

### 3. Visual Fidelity Evidence

layout、grouping、readability、visual hierarchy、feedbackの識別性等、Targetとしての視覚品質を記録します。

Visual Fidelityは重要ですが、Semantic / Functional Evidenceの代替にはしません。

VLM、image similarity、screenshot diff等を使っても、そのscoreだけでSemantic ConformanceやScenario成立を合格にしません。

## Scenario Matrix

少なくとも次をHandover BaselineのDriver Scenarioと対応付けます。

| Scenario | Semantic確認 | Functional確認 |
| --- | --- | --- |
| browse | 同じbrowse Pattern / Query / Datasource | 一覧を取得・閲覧できる |
| record open | 同じrecord Pattern / Data | 一件を開ける |
| reference lookup | 同じRelation / read-only Datasource | 参照候補を検索・選択できる |
| header-detail read | 同じ1:N Relation | HeaderとDetailを閲覧できる |
| header-detail edit | baselineにwrite意味がある場合、同じRelation / Validation | 許可された範囲だけ編集できる |
| validation satisfied | 同じCanonical Validation | 正常入力が次へ進める |
| validation not satisfied | 同じCanonical Validation | 不成立理由を返しCommandへ進まない |
| authorization allowed | 同じAuthorization Capability | 許可された操作を実行できる |
| authorization denied | 同じAuthorization意味 / Outcome | Execution Errorへ畳まず扱える |
| command | 同じcommand Pattern / Capability Binding | 明示Capabilityをinvokeできる |
| business result | 同じoutput / named Outcome | Target表現へProjectionできる |
| execution error | 同じInvocation境界 | 業務Outcomeと区別して扱える |
| observation | 同じObservation意図 / read境界 | acknowledgementとObserved stateを区別できる |

Handover Baselineに存在しないScenario意味をWeb側だけで追加してcoverage不足を隠しません。

## Negative / Architecture Conformance

正常操作だけでGate Cを判定しません。

### Canonical不変

Web Adapterを外してもCanonical Worldが変わらないことを確認します。

Web固有component、route、HTTP status、DOM selector、CSS class等をCanonical Semantic Layerへ追加しません。

### 二重実装を作らない

次をWeb Target内へ独立正本として複製しません。

- native SQL
- Capability implementation
- Validation rule
- Authorization rule
- Business Outcome定義
- Observation対象の意味

### Result境界を保つ

少なくとも次を分けます。

```text
Business Result
Execution Error
Observation
Developer Diagnostic
```

HTTP status、toast、dialog、inline message等は表現方式です。transport / UI表現からCanonical意味を逆定義しません。

### Command acknowledgementとObservationを分ける

Commandが正常なBusiness Resultを返しただけで、外部状態が成立済みとみなしません。

Observationが必要なScenarioではCommand EvidenceとObservation Evidenceを別々に残します。

### Datasource境界を破らない

read-only DatasourceをTarget都合でwrite可能にしません。browser storageやform stateもCanonical Datasourceのwrite authorityにはしません。

### Standard ProjectionとCustom UI

少なくとも一箇所でCustom UIの接続を確認します。

Custom UIでもApplication Pattern、Validation、Authorization、Capability、Result、Observation等の上位意味へ接続し、独自の業務正本を作りません。

## Evidence Recordの最小項目

最初から最終JSON schemaを固定しません。ただしEvidence artifactから少なくとも次を復元できる必要があります。

### Baseline provenance

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- Driver Scenario identity
- Gate B Evidence reference
- baseline unresolved

### Target provenance

- `targetBuildRevision = T`
- Target Adapter / Host identity
- build / run identity
- Reference Workspace / Canonical subject identity

### Canonical references

- Application ID
- Pattern ID
- Data / Relation / Query / Capability等、Scenarioで参照したCanonical ID
- 必要なSource Reference

### Runtime / Target observations

- observed Target Capability
- observed Runtime Binding
- execution artifact / ABI / Host identity / observations（適用可能な範囲）
- observed Execution / Trust条件
- interaction steps
- replayable behavior identity / precondition / expected semantic operation（利用する場合）
- raw Target Evidence reference
- normalized event / behavior Evidence referenceとraw provenance（利用する場合）
- Business Result / Execution Error / Observationの分類

### Conformance result

- Semantic Conformance result
- Functional result
- Visual Evidence reference
- performance measurement reference
- Diagnostic / unresolved

### Evaluator information

自動評価を使った場合は少なくとも次を追跡します。

- evaluator / automation tool identity
- version
- 入力artifact
- evaluatorが判定したEvidence種別

prompt、trajectory、score等を保存してもCanonical meaningへ含めません。secret、credential、不要なprivate dataを通常Evidenceへ保存しません。

## Evidence artifactはCanonicalではない

Gate C Evidenceはderivedな検証成果物です。

```text
Authoritative Source / Handover Baseline
          ↓
Driver / Target execution
          ↓
Evidence artifact
```

Evidence artifactからCanonicalへ暗黙に逆書きしません。

Screenshot、DOM snapshot、browser trace、VLM judge結果、GUI agent trajectory、performance trace等は再確認や比較に利用できますが、source of truthにはしません。

## Target / Runtime Binding差の引き継ぎ

第一Web Targetだけで完全なtaxonomyを作りません。

Gate Cでは観測した差を次へ整理し、`ISSUE-000026` / GitHub Issue #259へ渡します。

1. Application上の意味
2. Target-neutral Presentation候補
3. Target Capability候補
4. Runtime Binding差
5. execution artifact / ABI / Host差（観測可能な範囲）
6. Execution / Trust差
7. Web固有実装
8. Host control plane固有差
9. Canonicalへ持ち上げてはいけない差
10. unresolved

分類できないものは無理に既存bucketへ押し込まず`unresolved`として残します。

Gate Cでは、複数execution artifactを実装・比較すること自体は要求しません。第一Web Targetで実際に使ったartifact / ABI / Host identityと、観測できた実現差をEvidenceへ残せればよく、native Go対Wasm等の比較はGate C後のportable artifact ReferenceでPressureします。

## Visual Fidelity

初期Referenceでは一律の数値閾値を置きません。

少なくとも次を比較可能にします。

- field groupingの可読性
- Relation / referenceの理解しやすさ
- header-detailの局所性
- validation feedbackの識別性
- authorization denialとsystem errorの識別性
- Business Result / Observationの識別性
- standard ProjectionとCustom UIの差

Screenshotまたは同等の再確認可能なartifactを優先して残します。VLM等のjudge結果だけを保存して入力artifactを捨てません。

## Performance Evidence

RQ-024の最終performance contractは第二Target以降の比較から決めますが、第一Web Targetから測定点を用意します。

- initial projection / load
- browse result render
- record open
- reference lookup
- validation
- authorization
- command response
- Business Result projection
- Execution Error projection
- observed completion

根拠のない一律閾値は第一Targetだけから固定しません。

## Gate C判定

Gate CはIssue closeやWeb画面の存在だけでは通過しません。

少なくとも次をEvidenceから説明できることを要求します。

- すべてのGate C runが明示Handover Baseline `B` / revision `R`へ追跡できる
- Target build revision `T`をbaseline `R`と混同していない
- `T`でbaseline subjectの意味を変更した場合、新Handover Baselineへ切り替えている
- DriverとWebが同じApplication / Pattern meaningを共有している
- Semantic Conformanceが成立している
- Reference ScenarioのFunctional Evidenceが揃っている
- replay可能なbehaviorを利用する場合、behaviorがHandover Scenario / semantic operation / raw Target Evidenceへ追跡できる
- Visual Fidelityが別Evidenceとして残っている
- Result / Execution Error / Observationの境界が保たれている
- Web固有実装をCanonicalへ逆流させていない
- Web Adapterを除去してもCanonical Worldが変わらない
- standard ProjectionとCustom UIが同じ上位意味へ接続できる
- Target Capability / Runtime Binding / execution artifact・ABI・Host / Execution・Trust差を第二Targetへ渡せる
- unresolvedを成功へ読み替えず明示できる

Gate C通過後、`ISSUE-000008`のEditor / AI支援保留解除と、`ISSUE-000026`の第二Target圧力試験を判断します。

判定記録: [`GATE_C_READINESS_AUDIT.md`](GATE_C_READINESS_AUDIT.md)（第一Web Target、2026-09-19、Ready）

## 非目標

- 第一TargetだけでTarget-neutral Presentation modelを完成させること
- Web frameworkをTEI標準として固定すること
- 完全なTarget Capability taxonomy
- Gate Cのためだけに複数execution artifactやWasmを実装・比較すること
- execution artifact / ABI / Hostのproduction taxonomy / schema
- pixel-perfect design system
- VLM / GUI agentの標準化
- visual scoreによる単一合否
- 第二Targetの実装
- Evidence artifactのCanonical化
- Target build revisionをHandover Baseline revisionと同一に固定すること
