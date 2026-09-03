# Gate C Evidence Contract

## 目的

この文書は、第一Web Targetが「表示できた」ことではなく、TEIのTarget-neutral Presentation / Interaction境界を検証できるEvidenceを残したことをもってGate Cを判定するための契約を定めます。

Gate Cでは、Driverと第一Web Targetが同じCanonical Application上の意味を共有し、そのうえでTargetへProjectionした結果として何が共通で、何がWeb固有だったかを説明できる必要があります。

この文書はWeb framework、GUI test tool、VLM、browser automation製品を標準化するものではありません。どの実装技術を選んでも残すべきEvidenceの意味を定めます。

関連:

- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- GitHub Issue #70
- GitHub Issue #270
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`

## Gate Cで分ける三種類のEvidence

第一Web Targetの評価を一つのscoreや一つの「成功」に畳みません。

```text
Canonical Application / Driver baseline
                │
                ├─ Semantic Conformance
                │
                ├─ Interaction / Functional Evidence
                │
                └─ Visual Fidelity Evidence
```

### Semantic Conformance

Web Targetが、Driverで成立したCanonical Application上の意味を保持しているかを確認します。

少なくとも次を対象にします。

- Application Pattern identity
- Data / Relation / Query / Capability identity
- Datasourceとread / read-write境界
- Validation rule
- Authentication / Authorization
- Capability Binding
- Business Result / named Outcome
- Execution Error
- Command / Observation
- Source-to-Canonical Traceability

Semantic ConformanceはGate Cの必須条件です。

### Interaction / Functional Evidence

利用者がWeb Target上で必要なReference Scenarioを実行できるかを確認します。

browser automation、GUI agent、人による再現手順等は実行手段です。どの手段を使っても、何のApplication上の意図を実行したかへ戻れる必要があります。

### Visual Fidelity Evidence

layout、grouping、readability、visual hierarchy、feedbackの見分けやすさ等、Targetとしての視覚品質を記録します。

Visual FidelityはTarget品質として重要ですが、Semantic / Functional Evidenceの代替にはしません。

VLM、image similarity、screenshot diff等を利用しても、そのscoreだけでSemantic ConformanceやScenario成立を合格にしません。

## Driverをbaselineとする

Web Target固有の期待値をゼロから作りません。

Gate Bを通過したDriver Scenarioを、Web Targetが保持すべき意味のbaselineとして利用します。

```text
Canonical Application
      │
      ├─ Driver Scenario ───── baseline meaning / result
      │
      └─ Web Projection ───── target interaction / representation
```

同じ意味を比較するのであって、Driverのconsole表現とWebのDOM表現を一致させることが目的ではありません。

## 最小Scenario Matrix

第一Web Targetでは、少なくとも次をDriver baselineと対応付けます。

| Scenario | Semantic確認 | Functional確認 |
| --- | --- | --- |
| browse | 同じbrowse Pattern / Query / Datasource | 一覧を取得・閲覧できる |
| record open | 同じrecord Pattern / Data | 一件を開ける |
| reference lookup | 同じRelation / read-only Datasource | 参照候補を検索・選択できる |
| header-detail read | 同じ1:N Relation | HeaderとDetailを閲覧できる |
| header-detail edit | 同じRelation / Validation | Detailを追加・変更・削除できる範囲を確認できる |
| validation satisfied | 同じCanonical Validation | 正常入力が次へ進める |
| validation not satisfied | 同じCanonical Validation | 不成立理由を利用者へ返しCommandへ進まない |
| authorization allowed | 同じAuthorization Capability | 許可された操作を実行できる |
| authorization denied | 同じAuthorization Capability / Outcome | 拒否をExecution Errorへ畳まず表示できる |
| command | 同じcommand Pattern / Capability Binding | 明示Capabilityをinvokeできる |
| business result | 同じoutput / named Outcome | Target表現へProjectionできる |
| execution error | 同じInvocation境界 | 業務Outcomeと区別して扱える |
| observation | 同じObservation意図 /境界 | acknowledgementとObserved stateを区別できる |

Gate BでScenarioが追加・修正された場合、このMatrixもbaselineを追従させます。Web側だけで独自Scenario意味を追加してbaseline不足を隠しません。

## Negative / Architecture Conformance

正常操作だけでGate Cを判定しません。少なくとも次をnegative caseとして確認します。

### Canonical不変

Web Adapterを外してもCanonical Worldが変わらないことを確認します。

Web固有のcomponent、route、HTTP status、DOM selector、CSS class等をCanonical Semantic Layerへ追加しません。

### 二重実装を作らない

次をWeb Target内へ複製しません。

- native SQL
- Capability implementation
- Validation rule
- Authorization rule
- Business Outcome定義
- Observation対象の意味

Targetはこれらを利用・Projectionしますが、Target固有コードを新しい正本にはしません。

### Result境界を保つ

少なくとも次を分けます。

```text
Business Result
Execution Error
Observation
Developer Diagnostic
```

HTTP status、toast、dialog、inline message等は表現方式です。

`HTTP 401だからdenied`、`HTTP 500だからExecution Error`のようにtransport / UI表現からCanonical意味を逆定義しません。

### Command acknowledgementとObservationを分ける

Commandが正常なBusiness Resultを返しただけで、外部状態が成立済みとみなしません。

Observationが必要なScenarioでは、Command EvidenceとObservation Evidenceを別々に残します。

### Datasource境界を破らない

read-only DatasourceをTarget都合でwrite可能にしません。

Target内のcache、form state、browser storage等も、Canonical Datasourceのwrite authorityとして扱いません。

### Standard ProjectionとCustom UI

少なくとも一箇所でCustom UIの接続を確認します。

Custom UIでもApplication Pattern、Validation、Authorization、Capability、Result、Observation等の上位意味へ接続し、独自の業務正本を作りません。

## Evidence Recordの最小項目

具体的なJSON schemaやfile formatは第一実装で決めますが、Evidence artifactから少なくとも次を復元できる必要があります。

### Provenance

- scenario ID
- repository revision / build identity
- Reference Workspace identity
- Driver baseline identity
- Target Adapter / Host identity
- 実行日時またはrun identity

### Canonical references

- Application ID
- Pattern ID
- Data / Relation / Query / Capability等、Scenarioで参照したCanonical ID
- 必要なSource Reference

### Runtime / Target observations

- observed Target Capability
- observed Runtime Binding
- observed Execution / Trust条件
- interaction steps
- Business Result / Execution Error / Observationの分類

### Conformance result

- Semantic Conformance result
- Functional result
- Visual Evidence reference
- performance measurement reference
- Diagnostic / unresolved事項

### Evaluator information

自動評価を使った場合は少なくとも次を追跡します。

- evaluator / automation tool名
- versionまたはidentity
- 入力artifact
- evaluatorが判定したEvidence種別

model prompt、trajectory、score等を保存する場合もCanonical meaningへ含めません。secret、credential、不要なprivate dataを通常Evidenceへ保存しません。

## Evidence artifactはCanonicalではない

Gate C Evidenceは検証成果物です。

```text
Canonical / Authoritative Source
          ↓
Driver / Target execution
          ↓
Evidence artifact
```

Evidence artifactからCanonicalへ暗黙に逆書きしません。

Screenshot、DOM snapshot、browser trace、VLM judge結果、GUI agent trajectory、performance trace等は、再確認や比較には利用できますが、Canonical source of truthにはしません。

## Target Capability / Execution / Trustの引き継ぎ

第一Web Targetだけで完全なtaxonomyを作りません。

Gate Cでは、観測した差分を次のbucketへ整理し、`ISSUE-000026` / GitHub Issue #259へ渡します。

1. Application上の意味
2. Target-neutral Presentation候補
3. Target Capability候補
4. Runtime Binding差
5. Execution / Trust差
6. Web固有実装
7. Host control plane固有差
8. Canonicalへ持ち上げてはいけない差

分類できないものは無理に既存bucketへ押し込まず、`unresolved`としてEvidenceを残します。

## Visual Fidelityの扱い

初期Referenceでは一律の数値閾値を置きません。

少なくとも次を比較可能にします。

- field groupingの可読性
- Relation / referenceの理解しやすさ
- header-detailの局所性
- validation feedbackの識別性
- authorization denialとsystem errorの識別性
- Business Result / Observation状態の識別性
- standard ProjectionとCustom UIの差

Screenshotまたは同等の再確認可能なartifactを残せるようにします。

VLM evaluator等を利用する場合、同じartifactを別evaluatorで再評価できるよう、judge結果より入力artifactを優先して残します。

## Performance Evidence

RQ-024の最終性能contractは第二Target以降で決めますが、第一Web Targetから測定点を用意します。

少なくとも次を測定可能にします。

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

現時点で根拠のない一律の数値閾値は固定しません。

## Gate C判定

Gate CはIssueがcloseされたことやWeb画面が存在することだけでは通過しません。

少なくとも次をEvidenceから説明できることを要求します。

- DriverとWebが同じApplication / Patternを共有している
- Semantic Conformanceが成立している
- Reference ScenarioのFunctional Evidenceが揃っている
- Visual Fidelityが別Evidenceとして残っている
- Result / Execution Error / Observationの意味境界が保たれている
- Web固有実装をCanonicalへ逆流させていない
- Web Adapterを除去してもCanonical Worldが変わらない
- standard ProjectionとCustom UIが同じ上位意味へ接続できる
- Target Capability / Execution / Trust差を第二Targetへ渡せる
- unresolvedな境界を未解決のまま明示できる

Gate C通過後、`ISSUE-000008`のEditor / AI支援保留解除と、`ISSUE-000026`の第二Target圧力試験を判断します。

## 非目標

- 第一TargetだけでTarget-neutral Presentation modelを完成させること
- Web frameworkをTEI標準として固定すること
- 完全なTarget Capability taxonomy
- pixel-perfect design system
- VLM / GUI agentの標準化
- visual scoreによる単一合否
- 第二Targetの実装
- Evidence artifactのCanonical化
