# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、既存の`order_entry`を中心に一つの流れとして整理するものです。

Application Pattern、既存認証Compatibility、Driver、第一Web Target、Editor、AI支援を個別に実装するだけでは、「一つの業務ApplicationがTEIの各層をどのように通るのか」を確かめにくくなります。

そのため、同じReference Applicationを段階的に育て、前の段階で確認した意味を次の段階がそのまま利用する進め方を採ります。

期限は置きません。次へ進む条件は日付ではなく、ReferenceとConformanceから確認できるEvidenceで判断します。

## v1 Reference Application

v1では、新しい理想化されたサンプルを別に作らず、既存の`reference/workspace/applications/order-entry`を育てます。

現在の`order_entry`には、すでに次の資産があります。

- `sales`と`customer`のSemantic Namespace
- `sales_core`へのread-write接続
- `master_data`へのread-only接続
- Database Contract
- native SQL Query
- `sales.order.submit` Capability
- Application Pattern `browse` / `record` / `reference`
- Source-to-Canonical Traceability
- Reference Workspaceと再生成可能なCanonical World

M0.xxで積み上げた資産を捨てず、そのままApplication側のv1実証へ接続します。

## 現在地

Stage 1「Application Pattern」は進行中です。

`browse`、`record`、`reference`の3種類については、Canonical modelから実Reference、共通Validation、Source Index、Golden Worldまで成立しています。

現在の実Referenceは次です。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}

pattern order_entry.order_record record {
    data sales.order
}

pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

次は`command`として既存Capability `sales.order.submit`への接続を実証します。

`header-detail`は、新しい`sales.OrderLine`と対応RelationをSemantic側へ追加した後に扱います。

Stage 1、Gate A、Milestone 1の完了はまだ宣言しません。

## v1で確認する業務シナリオ

最初の業務シナリオは、「受注を探し、開き、顧客を参照し、明細を編集し、内容と利用権限を確認して確定する」という一続きの操作です。

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

このシナリオをApplication Pattern、Driver、Web Targetで別々に作り直しません。同じApplication上の意味を共有し、各層では必要なProjectionやBindingだけを追加します。

## Application Patternの位置づけ

v1では次の5種類を実証候補として扱います。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

これらは画面テンプレートの一覧ではありません。Targetが変わっても残る可能性があるApplication上の意図として、実Referenceから一つずつ検証します。

現時点で実証済みなのは`browse`、`record`、`reference`です。5分類をTEI全体の永久的な標準語彙とはみなしません。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

現在の詳細なEvidenceと次の実装順は[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)を正とします。

## ここまでのPattern実証から得た判断

### `browse`

既存Queryをidentityで参照し、native SQLをPatternへ複製せず成立しました。

Query自身がDatasourceを保持しているため、実ReferenceではPattern側へDatasourceを重複記述していません。

### `record`

`sales.order`を「Application上の一件」として扱う意味だけで成立しました。

一件取得用のQueryやDatasourceをPatternへ持ち込む必要はありませんでした。

### `reference`

Semantic Relation `sales.order.customer`とlogical datasource `master`の組み合わせで成立しました。

Relationが参照元と参照先をすでに持つため、DataやtargetをPatternへ重複記述していません。

`master`の`access read`はApplication定義を正本とし、Patternへread-only属性を複製していません。

Reference Conformanceでは、`master_data.customer ^2`がRelation target `customer.customer`を実際に`read`対象として公開していることまで確認しています。

## Referenceへ追加する意味

### `command`

次の実証では、既存`sales.order.submit` Capabilityへ接続します。

第一候補はCapability identityだけを保持する最小形です。

```text
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

ただし、これは実装前の候補です。対象Data、Datasource、Validation、Authorizationとの関係をPatternへ直接持たせる必要があるかは、実Referenceから判断します。

`command`を単なるRecord更新へ還元せず、業務上の操作としてCapabilityへ接続します。

Capabilityの結果境界は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に従います。

既存の単一`output`とnamed OutcomeはいずれもBusiness Resultとして扱い、Capability呼び出しそのものを完了できなかった状態はExecution Errorとして分離します。

`command`を追加するという理由だけで、現行`sales.order.submit`の`output`をnamed Outcomeへ書き換えません。

### `OrderLine`と`header-detail`

受注明細に相当する`OrderLine`は必要ですが、Pattern基盤と同時には追加しません。

`command`まで既存資産で実証した後に、`OrderLine`と対応RelationをSemantic側へ追加し、`header-detail`へ進みます。

概念上は次の関係です。

```text
Order 1 ── N OrderLine
```

Semantic Relation、Application上の`header-detail`、Database Foreign Keyは別の意味です。

1:N関係が存在するという理由だけで、FKや画面構造を暗黙生成しません。

### Validation

少なくとも次の二種類を扱います。

1. 一つのfieldだけで判定できるValidation
2. HeaderとDetail、または複数fieldをまたぐValidation

単純なHTML入力制約やDatabase constraintだけでは表せないApplication上のruleを確認します。

ValidationをPatternそのものが直接参照するべきか、Driver Scenario側で構成するべきかは、必要な実例が揃うまで固定しません。

### Authorization

既存認証CompatibilityはApplication Patternと並行して進めます。

認証・認可の都合で`order_entry`の業務モデルをTEI標準User schemaへ作り直しません。Application利用者のAuthentication / AuthorizationはSemantic Capabilityとして扱い、既存schemaや認証製品はBinding / implementation側へ残します。

基本境界は[ADR-0019](adr/0019-authentication-capability-boundary.md)、実装順とConformanceは[`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)を参照します。

AuthenticationのBusiness ResultとExecution ErrorにもADR-0020を適用します。

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

Database / Provider側の作業も止めません。ただし、v1 Referenceに必要な境界や安全上の重要課題を優先し、Provider固有機能を増やすこと自体を目的にはしません。

## Stage 0 — Product BaselineとGap Audit

### 状態

上流基線として整備済みです。

### Evidence

- Product Value
- Product Vision
- Product Principles
- Product Requirements
- Master Plan
- 期限を置かないMilestone
- Product Requirement Gap Audit

Database / Provider、Source / Meaning、Runtime安全境界に比べ、Application Pattern、既存認証、Driver、Target、Editor、AI支援の実証が薄いことが確認されました。

この偏りを受け、主線をApplication vertical sliceへ移しています。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67  
詳細: [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)

### 状態

進行中です。

### 完了済み

- PatternのCanonical model
- `patterns/*.tei` Loader / Normalization
- Source Indexとfile pathから独立したPattern identity
- 共通Validation / Diagnostic projection
- `browse`実Reference
- `record`実Reference
- `reference`実Reference
- Repository自身の`tei world`によるGolden再生成

### 次の作業順

1. `command`として`sales.order.submit` Capabilityへの接続を実証する
2. `OrderLine`と対応RelationをSemantic側へ追加する
3. `header-detail`を実証する
4. 5 Patternを同じApplication上の意味として説明できるか監査する
5. 既存認証CompatibilityのEvidenceと合流し、Gate Aを評価する

### Gate Aへ進む条件

- 5 Patternを同じApplication定義から識別できる
- Pattern identityがfile pathへ依存しない
- Patternの正本とEditor内部状態を暗黙の二重正本にしていない
- Semantic Relation、Application Pattern、Persistence Mappingの境界を説明できる
- native SQLを別のDSLへ複製していない
- read-only Datasourceの制約を維持している
- `command`がCapability identityへ接続されている
- Driverが利用できるTarget非依存情報が揃っている
- Source Index / DiagnosticからPattern定義へ戻れる
- Reference / Conformanceから上記を再現できる

現時点では、Stage 1、Gate A、Milestone 1の完了は宣言しません。

## 並行Stage — 既存認証Compatibility

関連: `ISSUE-000003` / GitHub Issue #68  
詳細: [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)

### 目的

既存システム固有の認証・認可仕様を、最初にTEI標準へ作り替えなくてもApplicationから利用できるCapabilityへ接続できることを確かめます。

### 設計基線

Authentication / Authorizationの責務分離はADR-0019、CapabilityのBusiness Result / Execution Error境界はADR-0020に従います。

### Gate Aへ合流する条件

- Authentication / Authorizationを別のCapability identityとして扱える
- legacy fieldやfunctionをCapabilityへMappingできる
- secret値をCanonicalな正本へ保存しない
- Application利用者の認可とPlugin / Runtime Permissionを混同しない
- Applicationが認証実装の物理構造へ直接依存しない
- 同じCapability identityのまま実現方式を交換できる
- Authenticationのnamed OutcomeとExecution Errorを区別できる
- `order_entry`のbrowse / edit / submit Authorizationへ接続できる

Gate Aでは、認証基盤全体がproduction-readyであることは求めません。

## Gate A — Application上の意味をDriverへ渡せる

Gate Aは、Stage 1と既存認証Compatibilityの成果をDriverへ渡す接続点です。

次の質問にRepository上のEvidenceから答えられることを条件とします。

- このApplicationは何をbrowseするのか
- 一件のrecordとは何か
- どのRelationを、どのDatasourceからreferenceするのか
- HeaderとDetailの関係は何か
- どのValidationを実行するのか
- どのCapabilityをcommandとして呼ぶのか
- 誰がその操作を行えるのか
- どのDatabaseがread-onlyで、どこへwriteできるのか
- CapabilityのBusiness ResultとExecution Errorをどう区別するのか

これらに答えるためにWeb画面やEditor内部状態を読む必要がある場合、Gate Aは未達とします。

## Stage 2 — Application Driver

関連: `ISSUE-000006` / GitHub Issue #69

### 目的

UIを作る前に、Gate Aで揃えたApplication上の意味を実行し、設計の整合を確かめられるようにします。

### 最小Scenario

1. browseで受注一覧を取得する
2. recordで一件を開く
3. referenceで顧客を解決する
4. header-detailで明細を編集する
5. Validationを実行する
6. Authorizationを確認する
7. commandで`sales.order.submit`を呼び出す
8. CapabilityのBusiness ResultとExecution Errorを区別する
9. 必要な場合は、外部世界で得られた状態をObservationから確認する

### Driverがしてはいけないこと

- Application ruleをDriver専用Goコードとして重複実装する
- native SQLをDriver側へコピーする
- logical datasourceを意味上ひとつのDatabaseへ統合する
- Target固有Presentationを必要条件にする
- Driver独自のOutcome体系を作る
- HTTP statusやGUI messageをDriver Coreの意味として固定する
- Driver実行ログをCanonicalの正本へ昇格させる

### Gate Bへ進む条件

- ScenarioをRepositoryだけから再現できる
- Application / Pattern / Query / Capability / Datasourceの解決結果を追跡できる
- `master_data`へwriteしないことを検証できる
- ValidationまたはAuthorizationが成立しない場合にcommandへ進まない
- 既存`output`とnamed Outcomeを同じBusiness Result境界から扱える
- Execution Errorをnamed Outcomeや業務result codeへ押し込まない
- command acknowledgementとobserved resultを必要に応じて区別できる
- Driver固有の状態がCanonical Worldへ逆流しない
- 同じApplication定義を第一Targetへ渡せる

## Gate B — UIなしで業務シナリオを説明できる

Gate Bでは、画面がなくてもApplicationの主要な振る舞いを実行・検証できることを確認します。

ここを通る前にWeb実装へ進むと、Application上の不足をWeb固有コードで埋める危険があります。第一Web Targetの実装はGate Bを原則的な前提とします。

## Stage 3 — 第一Web Target

関連: `ISSUE-000007` / GitHub Issue #70

### 目的

Driverで確認済みのApplication上の意味をWebへProjectionし、複数Targetで共有できそうなPresentation上の意味と、Web固有実装の境界を実例から抽出します。

### 対象

- browse
- record
- reference
- header-detail
- command
- Validation feedback
- Authorizationに応じた利用可否
- CapabilityのBusiness Result
- Execution Error
- Observation

### Web Targetがしてはいけないこと

- Web専用のApplication modelやresult modelを作る
- native SQLやCapability implementationを複製する
- Application ValidationやAuthorization ruleをWeb専用ruleとして書き直す
- HTMLやframeworkの都合をSemantic Kernelへ持ち上げる
- expectedなnamed OutcomeとExecution Errorを一種類のWeb errorへ畳む
- HTTP statusやUI messageからSemantic Outcomeを逆定義する
- read-only Datasource制約をUI都合で無視する

### Custom UI

少なくとも一箇所でCustom UIを接続し、標準Projectionだけに閉じないことを確認します。

Custom UIでもApplication上の意味、Validation、Capability、Authorizationとの接続を維持します。

### Gate Cへ進む条件

- DriverとWebが同じApplication / Patternを利用している
- Web固有情報をCanonical Semantic Modelへ追加していない
- Business Result / Execution Error / Observationの意味境界を維持している
- Web Adapterを外してもCanonical Worldが変わらない
- 標準ProjectionとCustom UIを併存できる
- Source / Application / Targetの来歴を追跡できる
- 共通候補とWeb固有差分を文書化できる

## Gate C — Editor / AIを具体化してよい

Gate Cを通過すると、Application上の意味、Driver実行結果、第一Targetとの差が実資産として揃います。

この時点でEditorやAI支援が何を見せ、何を探索し、何を確認すべきかを具体化します。

`ISSUE-000008`の保留を解除する判断は、このGateを基準に行います。

## Stage 4 — Editor / AI支援

### Editor

最初から完成IDEを作りません。

v1 `order_entry`について、少なくとも次を同じCanonical上の意味から高密度に確認できるProjectionを優先します。

- Application / Datasource
- Data / Relation
- Application Pattern
- Validation / Command
- Authorization
- Source / Diagnostic
- Driver result
- 第一TargetとのProjection差

Editor内部状態を第二の正本にはしません。

### AI支援

最初のAI支援では、コード生成量ではなく、既存知識を利用しながら設計上の見落としを減らせるかを確かめます。

対象は次です。

- 既存Pattern / Capabilityの探索
- 既回答事項の再質問抑止
- 1:N、履歴、状態遷移、認証などの典型論点の確認提案
- AI提案と採用済みDecisionの分離
- Decision変更履歴
- 提案の無視・抑制・無効化

## Gate D — 第二Targetで検証できる

Editor / AIが完成していること自体をGate Dの条件にはしません。

第一Web Targetで「共通と思われた部分」と「Web固有と判断した部分」が追跡可能であることを条件とします。

## Stage 5 — 第二Target

第二Targetは、単なる対応Target数の増加ではなく、Target-neutral境界の圧力試験です。

Webとの差が十分に表れるTargetを優先し、Desktop、API、MCP、VUIなどから、その時点で最も設計上の差を引き出せるものを選びます。

第二TargetのためにCore変更が必要になった場合は、それ自体を失敗とはみなしません。第一Web Targetの都合を誤って共通意味へ持ち上げていなかったかを確認します。

## Evidenceの残し方

各Stageでは「実装した」という記録だけで完了にしません。

少なくとも次のどれかを残します。

- Reference source
- deterministic fixture
- Conformance test
- golden artifact
- Driver Scenario
- Target comparison
- ADR
- 未解決事項を明示した監査文書

実験結果が予想と異なる場合、その差を隠して既存計画へ合わせません。Product Valueに照らして意味のある差であれば、Pattern、Requirement、Milestoneを更新します。

## 判断を保留する事項

次は現時点で固定しません。

- Application Patternの網羅的な標準一覧
- `command` / `header-detail`のDSL最終構文
- PatternとValidation / Authorizationの最終的な直接関係
- 第一Web Targetのframework
- Editorの最終UI framework
- 第二Targetの種類
- B2C向けの一律性能目標
- Authentication ReferenceをDatabase、LDAP、IdP、外部APIなどのどれへ固定するか
- `std.auth.*`の標準契約
- Application単位の分散transactionを一般化する方法

これらは、先に決めるよりReferenceから得られるEvidenceを待つ方が安全です。

## v1へ進んだと判断する条件

v1を単なるVersion番号として扱わず、少なくとも次をRepositoryのEvidenceから説明できることを目安とします。

- 一つの現実的なApplicationをSemantic上の意味から第一Targetまで一続きに追跡できる
- 複数Databaseを自然に利用できる
- native SQLとCustom implementationを正本のまま利用できる
- 既存認証仕様を保持したままCapabilityへ接続できる
- Application PatternをDriverとTargetで共有できる
- UIがなくてもDriverで主要シナリオを検証できる
- 第一Targetで標準ProjectionとCustom UIを併用できる
- CapabilityのBusiness ResultとExecution Errorを必要な箇所で分離できる
- CommandとObservationを必要な箇所で分離できる
- Reference / Conformanceから同じ結果を再現できる
- 何が未保証かを`docs/status.md`から明確に判断できる

EditorとAI支援の完成、第二Target、production HAはv1の前提条件にはしません。v1で成立したApplication vertical sliceを基盤として、その後に拡張します。

## 関連文書

- `product/value/TEI_PRODUCT_VALUE.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/MASTER_PLAN.md`
- `plan/MILESTONES.md`
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
