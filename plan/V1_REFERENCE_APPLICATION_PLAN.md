# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、既存の`order_entry`を中心に一つの流れとして整理するものです。

Application Pattern、既存認証Compatibility、Driver、第一Web Target、Editor、AI支援を個別に実装するだけでは、一つの業務ApplicationがTEIの各層をどのように通り、どの意味が引き継がれるのかが見えにくくなります。

そこで、同じReference Applicationを段階的に育て、前段で確認した意味を後段がそのまま利用する進め方を採ります。

期限は置きません。次へ進む条件は日付ではなく、Reference / Conformanceから確認できるEvidenceで判断します。

## v1 Reference Application

v1では新しい理想化されたサンプルを別に作らず、既存の`reference/workspace/applications/order-entry`を育てます。

`order_entry`には、すでに次の資産があります。

- `sales`と`customer`のSemantic Namespace
- `sales_core`へのread-write接続
- `master_data`へのread接続
- Database Contract
- native SQL Query
- `sales.order.submit` Capability
- Source-to-Canonical Traceability
- Reference Workspaceと再生成可能なCanonical World

これらを捨てずに利用することで、M0.xxで積み上げたSource / Meaning / Database / Runtime側のEvidenceを、Application側のv1実証へ接続します。

## v1で確認する業務シナリオ

最初の業務シナリオは、次の一続きの操作です。

```text
受注一覧を探す
    ↓
受注を開く
    ↓
顧客を参照する
    ↓
受注明細を編集する
    ↓
入力内容を検証する
    ↓
利用権限を確認する
    ↓
受注を確定する
    ↓
結果を観測する
```

このシナリオをApplication Pattern、Driver、Web Targetごとに別々に定義し直しません。

同じApplication上の意味を共有し、それぞれの層では必要なBindingやProjectionだけを追加します。

## v1で検証するApplication Pattern

最初の候補は次の5種類です。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

これらは画面テンプレートの一覧ではありません。Targetが変わっても残る可能性のあるApplication上の意図として、一つずつ実Referenceから検証します。

5分類をTEI全体の永久的な標準語彙として先に固定しません。

詳細な設計・実装基線は[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)を正とします。

## 現在地

Stage 1は進行中です。

Application Pattern側では、次の3種類について実Referenceまで完了しています。

### `browse`

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

Canonical model、Parser、Loader、Normalization、Source Index、Workspace Validation、Diagnostic projection、Golden World、実Referenceでのfile rename invarianceまで確認しました。

QueryがDatasourceを正本として持つ場合、Patternへ同じDatasourceを重複記述しない判断もReferenceから確認しています。

### `record`

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

`record`はApplicationがどのSemantic Dataを一件として扱うかという意味だけで成立しました。

QueryやDatasourceを追加せず、特定`browse`との遷移関係もCanonicalな意味へ持ち込んでいません。

### `reference`

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

Semantic Relationが持つsource / target / cardinalityをPatternへ複製せず、Application固有のlogical datasourceだけを追加する形で成立しました。

`master`のaccessはApplication定義を正本とし、Pattern側ではread可能であることを確認します。read-only値そのものを重複記述しません。

実Referenceでは、Datasourceが要求するDatabase ContractがRelation targetをread対象として公開していることまでConformanceで追跡しています。

### 次の対象 — `command`

Repository内の作業定義は[`ISSUE-000010`](issue/000010-command-application-pattern.md)、GitHub Issueは#110です。

第一候補は次です。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

Capabilityがすでに持つinput / output、Datasource、implementationをPatternへ複製しない方針から検証します。

### `header-detail`

`command`の後に、`OrderLine`と対応Semantic Relationを追加して検証します。

```text
Order 1 ── N OrderLine
```

Semantic Relation、Application上のheader-detail、Database Foreign Key、Target固有UIは別の意味として扱います。

## Capability結果の共通境界

Application Pattern、既存認証、Driver、Web Targetでは、Capabilityの結果について同じ境界を利用します。

[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に従い、次を区別します。

```text
Business Result
  ├─ 既存output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Capability呼び出しそのものを完了できなかった状態
```

既存の単一`output`を持つCapabilityは、そのままBusiness Resultとして利用できます。

`command` Patternを追加するという理由だけで、`sales.order.submit`の`output sales.order`を名前付きOutcomeへ書き換えません。Commandを実行して初めて分かる複数の業務上の結末が本当に必要だと確認できた場合だけ、GitHub Issue #77の圧力試験として扱います。

CommandのBusiness Resultと、その後に外部世界から得たObservationも区別します。

## 既存認証Compatibility

Application Patternと並行して、`ISSUE-000003` / GitHub Issue #68で既存認証Compatibilityを進めます。

既存システム固有の認証・認可仕様を、最初にTEI標準User schemaへ作り替えることは前提にしません。

Application側はAuthentication / AuthorizationをSemantic Capabilityとして利用し、既存schema、既存関数、stored procedure、外部API、custom implementation等はBinding / implementation側へ残します。

Authentication / Authorizationの責務境界はADR-0019、Business Result / Execution ErrorはADR-0020を利用します。

認証・認可の実装EvidenceはApplication Patternとは別に揃え、Gate Aで合流させます。

## 全体の作業順

```text
Stage 0  Product Baseline / Gap Audit
               ↓
Stage 1  Application Pattern ─────────┐
               │                      │
               │              既存認証Compatibility
               │                      │
               ↓                      │
Gate A ───────────────────────────────┘
               ↓
Stage 2  Application Driver
               ↓
Gate B
               ↓
Stage 3  第一Web Target
               ↓
Gate C
               ↓
Stage 4  Editor / AI支援
               ↓
Gate D
               ↓
Stage 5  第二Target
```

Stage 1と既存認証Compatibilityは並行して進められます。

Database / Provider側の作業も止めません。ただし、v1 Referenceに必要な境界や安全上の重要課題を優先し、特定Providerの機能数を増やすこと自体を目的にはしません。

## Stage 0 — Product Baseline / Gap Audit

### 状態

上流基線として整備済みです。

### Evidence

- Product Value
- Product Vision
- Product Principles
- Product Requirements RQ-001〜RQ-024
- Master Plan
- 期限を置かないMilestone
- Product Requirement Gap Audit

### この段階で確認したこと

Source / Meaning、Database / Provider、Runtime安全境界に比べ、Application Pattern、既存認証、Driver、Target、Editor、AI支援の実証が薄いことを確認しました。

この偏りを受け、v1の主線をApplication vertical sliceへ移しています。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67  
詳細: [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)

### 状態

進行中です。

`browse`、`record`、`reference`は実Referenceまで完了しています。

次は`command`を`ISSUE-000010` / GitHub Issue #110で実証し、その後`OrderLine`と`header-detail`へ進みます。

### 現在の作業順

```text
browse                完了
  ↓
record                完了
  ↓
reference             完了
  ↓
command               ISSUE-000010 / #110
  ↓
OrderLine + Relation
  ↓
header-detail
  ↓
Stage 1 Evidence監査
```

Pattern基盤の変更、新しい業務Semanticの追加、Driver実装を一つの変更へ混ぜません。

### Gate Aへ渡す情報

Application Pattern側から少なくとも次をTarget非依存の情報として説明できる必要があります。

- browse対象とQuery identity
- record対象
- reference対象となるRelationとDatasource
- header-detailの関係
- commandとして呼ぶCapability identity
- Application Validation
- read / read-write Datasource境界
- PatternのCanonical identityとauthoring source

既存認証Compatibility側からは次を受け取ります。

- Authentication Capability
- Authorization Capability
- 対象action
- 既存仕様を保持したBinding境界

Capability結果境界として、ADR-0020 / #77からBusiness Result / Execution Errorを受け取ります。

### Gate Aの条件

次をRepository内のReference / Conformanceから再現できたとき、Gate Aを評価します。

- v1業務シナリオに必要なApplication Patternを同じ`order_entry`から識別できる
- Pattern identityがfile pathに依存しない
- Semantic / Query / Relation / Capabilityの正本をPatternへ複製していない
- native SQLを別DSLへ複製していない
- read-only Datasourceの境界を維持している
- `command`がCapability identityへ接続されている
- Semantic Relation、Application Pattern、Persistence Mappingを混同していない
- Authentication / Authorizationを既存schemaの作り直しなしにApplicationへ接続できる
- Source Index / Diagnosticからauthoring sourceへ戻れる
- 後続Driverが利用できるTarget非依存情報が揃っている

現時点では`command`、`header-detail`、既存認証側の実装Evidenceが残っているため、Gate Aは未達です。

## Stage 2 — Application Driver

関連: `ISSUE-000006` / GitHub Issue #69

### 目的

UIを作る前に、Stage 1で得たApplication上の意味を設計検証用Driverから実行できることを確認します。

最初のDriverはproduction Runtimeの縮小版ではありません。設計途中のApplicationを動かし、意味上の矛盾やBinding不足を検出する環境です。

### 最小Scenario

```text
browse
  ↓
record
  ↓
reference
  ↓
header-detail edit
  ↓
validation
  ↓
authorization
  ↓
command
  ↓
Business Result / Execution Error
  ↓
必要に応じてObservation
```

Driverはlogical datasource alias、Query identity、Capability identity、read-only境界を維持します。

Capability implementationを選ぶためのBindingはDriverで解決しますが、その結果をApplication Patternの正本へ逆流させません。

### Gate Bの条件

上記ScenarioをRepositoryだけから再現でき、Business Result / Execution Error / Observationの意味を区別したまま第一Web Targetへ渡せることを確認します。

production-grade Workflow Runtime、distributed execution、HA、full transaction coordinatorはGate Bの条件ではありません。

## Stage 3 — 第一Web Target

関連: `ISSUE-000007` / GitHub Issue #70

### 目的

最初から巨大なTarget-neutral Presentation IRを作らず、Driverで確認済みの`order_entry`を第一TargetであるWebへProjectionします。

具体的なWeb実装から、複数Targetで共有できそうなPresentation上の意味と、Web固有の差分を抽出します。

### 守る境界

- Application上の意味をWeb側で再定義しない
- native SQLやCapability implementationをWebへ複製しない
- Validation / AuthorizationをWeb固有コードへ重複実装しない
- Business ResultとExecution Errorを一種類のWeb errorへ畳まない
- ObservationとCommand responseを必要な箇所で区別する
- HTTP statusやDOMをSemanticな意味へ逆流させない
- standard ProjectionとCustom UIを併存できる
- Web Adapterを外してもCanonical Worldが変わらない

### Gate Cの条件

DriverとWebが同じApplication / Patternと結果境界を共有し、共通Presentation候補とWeb固有差分を明示できることを確認します。

第一Web TargetだけでTarget-neutral設計が完成したとはみなしません。

## Stage 4 — Editor / AI支援

関連: `ISSUE-000008`

### 着手条件

Gate CまでのEvidenceから、次が明確になっていることを前提にします。

- Application Patternの意味境界
- Driverが必要とする情報
- 第一Targetで共通化できた情報
- Web固有として残った情報

これらが不明なままEditorを先に作ると、Editor都合のmetadataが新しい正本になりやすいためです。

### 最初のEditorで扱う範囲

`order_entry`を高密度に確認できる開発者向け環境を目指します。

少なくとも次を一画面または近接した文脈で追跡できることを重視します。

- Application / Datasource
- Semantic Data / Relation
- Application Pattern
- Validation / Command
- Source / Diagnostic
- Driver結果

### AI支援

最初のAI支援では、大規模な自動生成よりも次を優先します。

- 既存Pattern / Capability / Semantic知識の探索
- 典型的な不足条件の控えめな確認
- 一度回答した質問を不必要に繰り返さない
- suggestionとaccepted decisionを区別する
- 過去の選択を追跡・変更できる
- 提案を無視・抑制・OFFにできる

AIの提案を暗黙の正本にはしません。

### Gate Dの条件

Editor / AI支援がCanonical sourceを第二の正本へ置き換えず、日常的な設計作業を軽くできるEvidenceが得られることを確認します。

## Stage 5 — 第二Target

第一Web Targetで抽出した共通候補を、別のTargetで圧力試験します。

第二Targetの種類は、第一Web Targetから得られた残差を見て選びます。API、MCP、Desktop、Mobile、VUI等を先に固定しません。

ここで重要なのは機能数ではなく、Web固有の前提を誤ってTarget-neutralな意味へ持ち上げていないかを確認することです。

第二Targetで境界修正が必要になった場合も、それ自体を失敗とはみなしません。Target-neutralな意味をEvidenceから修正できることを重視します。

## v1で完了とみなさないこと

v1 Reference Applicationが成立しても、次を同時に完成したとはみなしません。

- production-grade Workflow Runtime
- 全Providerの完全対応
- 全Target対応
- 完成した汎用Editor
- AIによる完全自動設計
- enterprise-wide IAM標準化
- distributed transaction coordinator
- すべてのApplicationに通用するPattern Catalog

v1の中心は、業務上の意味を失わず、同じApplicationをCanonical source → Driver → 第一Targetへ通せることをReference / Conformanceで証明することです。

## Evidenceの扱い

各Stageでは、実装が存在するだけで完了とはみなしません。

少なくとも次を区別します。

- 設計上決定したこと
- 実装したこと
- Referenceで実際に使ったこと
- Conformanceで再現可能にしたこと
- 次のTargetやProviderでまだ圧力試験していないこと

不明なものは不明なまま残し、将来の実装都合から過去のEvidenceを強く言い換えません。

## 文書更新の原則

実装が進んだ際は、古い時制の文章へ進捗を継ぎ足すだけにしません。

現在地、成立した境界、残っている問い、次へ進む条件が自然な日本語として一続きに読めるよう、必要に応じて文書全体を再構成します。

意味や完了条件を変える場合は、単なる文面整理として扱わず、ADR / Issue / Requirementとの関係を明示します。
