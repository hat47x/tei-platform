# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、既存の`order_entry`を中心に一つの流れとして整理します。

Application Pattern、既存認証Compatibility、Driver、第一Web Target、Editor、AI支援を別々の機能として作るだけでは、「一つの業務ApplicationがTEIの各層をどのように通るのか」が見えにくくなります。

そこで、同じReference Applicationを段階的に育て、前の段階で確認した意味を次の段階がそのまま利用する進め方を採ります。

期限は置きません。次へ進む条件は日付ではなく、ReferenceとConformanceから確認できるEvidenceで判断します。

## v1 Reference Application

v1では、新しい理想化されたサンプルを別に作らず、既存の`reference/workspace/applications/order-entry`を育てます。

`order_entry`には、すでに次の資産があります。

- `sales`と`customer`のSemantic Namespace
- `sales_core`へのread-write接続
- `master_data`へのread-only接続
- Database Contract
- native SQL Query `order_entry.recent_orders`
- `sales.order.submit` Capability
- Source-to-Canonical Traceability
- Reference Workspaceと再生成可能なCanonical World

M0.xxで積み上げたこれらのEvidenceを捨てず、Application Pattern、Driver、Targetへ接続します。

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
必要な結果を観測する
```

このシナリオをApplication Pattern、Driver、Web Targetで別々に作り直しません。同じApplication上の意味を共有し、それぞれの層では必要なProjectionやBindingだけを追加します。

## Application Patternの位置づけ

v1では、次の5種類を候補として検証します。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

これらは画面テンプレートの一覧ではなく、Targetが変わっても残る可能性があるApplication上の意図です。

現時点で5分類をTEI全体の永久的な標準語彙とはみなしません。Applicationで実証した後、Driver、第一Target、第二Targetから得られるEvidenceも使い、共有・標準化できる範囲を判断します。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。現在の実装Evidenceと次の最小縦切りは[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)を正とします。

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

## Stage 0 — Product BaselineとGap Audit

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

Database / Provider、Source / Meaning、Runtime安全境界に比べ、Application Pattern、既存認証、Driver、Target、Editor、AI支援の実証が薄いことが明確になりました。

この偏りを受け、v1の主線をApplication vertical sliceへ移しています。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67  
詳細: [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)

### 状態

進行中です。

`browse`を受け止める基盤は、PR #87〜#99で次のところまで実装されています。

- Applicationが所有するPatternの最小Canonical model
- `browse` Parser
- `applications/*/patterns/*.tei` Loader
- Canonical ID順Normalization
- `application-pattern:<Canonical ID>`のSource Index
- Pattern ID tokenへのsource range
- Data / Query / Datasourceの参照整合性Validation
- Pattern Canonical ID重複の検出
- Pattern固有DiagnosticのSubject保持
- Diagnostic SubjectからPattern ID tokenへのSource projection

一方、`order_entry`の実Reference Patternと、Pattern固有Validationの共通`validation.Validate`への統合はまだ未完了です。

### 次に閉じる二つのEvidence

#### 共通Validation統合

[`ISSUE-000010`](issue/000010-application-pattern-validation-integration.md) / GitHub Issue #98で、Pattern固有Validationを通常の`validation.Validate`へ統合します。

目的は、CLI、Editor、Driver等の利用側がApplication Patternだけを特別扱いして追加のValidation入口を呼ばなくても済む状態にすることです。

#### `browse`実Reference

[`ISSUE-000011`](issue/000011-order-entry-browse-reference.md) / GitHub Issue #100で、`order_entry.recent_orders`を実Reference Patternとして固定します。

既存の`sales.order`とnative SQL Query `order_entry.recent_orders`をidentityで参照し、Queryがすでに持つDatasource `sales`を理由なくPatternへ重複記述しません。

Pattern追加後は、次までを一続きに確認します。

```text
Pattern source
    ↓
Parser / Loader
    ↓
Application.Patterns
    ↓
Normalization / Validation
    ↓
reference/world.json
    ↓
Source Index / Diagnostic
```

実Referenceのファイル名を変更してもCanonical WorldとPattern Subject identityが変わらず、Source Artifact pathだけが追随することも確認します。

`ISSUE-000010`と`ISSUE-000011`の実装順は固定しません。両者が揃った時点で、`browse`を通常のWorkspace validationから実Referenceまで通して扱える状態になります。

### `browse`後の作業順

`browse`を閉じた後、既存資産だけで扱えるものを先に追加します。

1. `record`: `sales.order`をApplication上の一件として扱う
2. `reference`: `sales.order.customer`から`customer.customer`を参照し、`master` Datasourceのread-only境界を維持する
3. `command`: `sales.order.submit` Capabilityへ接続する
4. `header-detail`: `OrderLine`と対応Relationを追加した後に扱う
5. 5種類を同じApplication上の意味として説明できるか監査し、Gate Aへ渡すEvidenceを揃える

### `OrderLine`

受注明細に相当する`OrderLine`は必要ですが、Pattern基盤の検証と同時には追加しません。

```text
Order 1 ── N OrderLine
```

Semantic Relation、Application上の`header-detail`、Database Foreign Keyは別の意味として扱います。1:N関係があるという理由だけで、FKや画面構造を暗黙生成しません。

### Validation

最終的には少なくとも次の二種類を扱います。

1. 一つのfieldだけで判定できるValidation
2. HeaderとDetail、または複数fieldをまたぐValidation

単純なHTML入力制約やDatabase constraintだけでは表せないApplication上のruleを確認します。

### Command

受注確定には既存の`sales.order.submit` Capabilityを利用します。

`command` Patternを単なるRecord更新へ還元せず、業務上の操作としてCapabilityへ接続します。

Capability結果は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)に従い、既存`output`とnamed OutcomeをBusiness Resultとして扱い、Execution Errorと分離します。

### Gate Aへ進む条件

Application Pattern側では、少なくとも次をRepository上のEvidenceから説明できる状態にします。

- 何を`browse`するか
- 一件の`record`とは何か
- 何を別Datasourceから`reference`するか
- HeaderとDetailの関係は何か
- どのValidationを実行するか
- どのCapabilityを`command`として呼ぶか
- どのDatasourceがread-onlyか
- Patternの正本とCanonical identityをどのように追跡するか
- Driverが利用できるTarget非依存情報が揃っているか

現時点ではStage 1、Gate A、Milestone 1の完了は宣言しません。

## 並行Stage — 既存認証Compatibility

関連: `ISSUE-000003` / GitHub Issue #68  
詳細: [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)

### 目的

既存システム固有の認証・認可仕様を、最初にTEI標準へ作り替えなくてもApplicationから利用できるCapabilityへ接続できることを確かめます。

### 設計基線

Authentication / Authorizationの責務分離はADR-0019、CapabilityのBusiness Result / Execution Error境界はADR-0020で決定済みです。

一方、既存認証fixture、複数の実現方式、`order_entry`へのAuthorization接続など、Reference / ConformanceによるEvidenceは引き続き必要です。

### 主な作業

1. `security`意味空間へAuthentication / Authorizationの最小Referenceを置く
2. AuthenticationのBusiness ResultとExecution Errorを区別する
3. legacy認証fixtureを用意する
4. 既存schemaや既存functionをCapabilityへ接続する
5. 同じCapability identityのまま異なる実現方式を追加する
6. secretがCanonical Worldや通常logへ混入しないことを確認する
7. `order_entry`のbrowse / edit / submit Authorizationへ接続する
8. Stage 1と合わせてGate Aを監査する

### Gate Aへ合流する条件

- Authentication / Authorizationを別のCapability identityとして扱える
- legacy fieldやfunctionをCapabilityへMappingできる
- secret値をCanonicalな正本へ保存しない
- Application利用者の認可とPlugin / Runtime Permissionを混同しない
- Applicationが認証実装の物理構造へ直接依存しない
- 同じCapability identityのまま実現方式を交換できる
- named OutcomeとExecution Errorを区別できる
- `order_entry`のbrowse / edit / submit Authorizationへ接続できる

Gate Aでは、認証基盤全体がproduction-readyであることは求めません。

## Gate A — Application上の意味をDriverへ渡せる

Gate Aは、Stage 1と既存認証Compatibilityの成果をDriverへ渡す接続点です。

次の質問にRepository上のEvidenceから答えられることを条件とします。

- このApplicationは何をbrowseするのか
- 一件のRecordとは何か
- 何を別Datasourceからreferenceするのか
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

UIを作る前に、Gate Aで揃えたApplication上の意味を実行し、設計の整合を確かめます。

### 最小Scenario

1. browseで受注一覧を取得する
2. recordで一件を開く
3. referenceで顧客を解決する
4. header-detailで明細を編集する
5. Validationを実行する
6. Authorizationを評価する
7. commandで`sales.order.submit`を呼び出す
8. Business ResultとExecution Errorを区別する
9. 必要な場合はObservationから外部世界の状態を確認する

### Driverがしてはいけないこと

- Application ruleをDriver専用Goコードとして重複実装する
- native SQLをDriver側へコピーする
- logical datasourceを一つのin-memory DBへ意味上統合する
- Target固有Presentationを必要条件にする
- Driver独自のOutcome体系を作る
- Driver実行ログをCanonicalの正本へ昇格させる

### Gate Bへ進む条件

- ScenarioをRepositoryだけから再現できる
- Application / Pattern / Query / Capability / Datasourceの解決結果を追跡できる
- `master_data`へwriteしないことを検証できる
- ValidationやAuthorizationが成立しない場合にcommandへ進まない
- 既存`output`とnamed Outcomeを同じBusiness Result境界から扱える
- Execution Errorを業務resultへ押し込まない
- command acknowledgementとobserved resultを必要に応じて区別できる
- Driver固有状態がCanonical Worldへ逆流しない
- 同じApplication定義を第一Targetへ渡せる

## Gate B — UIなしで業務シナリオを説明できる

Gate Bでは、画面がなくてもApplicationの主要な振る舞いを実行・検証できることを確認します。

ここを通る前にWeb実装へ進むと、Application上の不足をWeb固有コードで埋める危険があります。第一Web TargetはGate Bを原則的な前提とします。

## Stage 3 — 第一Web Target

関連: `ISSUE-000007` / GitHub Issue #70

### 目的

Driverで確認済みのApplication上の意味をWebへProjectionし、複数Targetで共有できそうな意味とWeb固有差分を実例から抽出します。

### 対象

- browse / record / reference / header-detail / command
- Validation feedback
- Authorizationに応じた利用可否
- CapabilityのBusiness Result
- Execution Error
- Observation

### Web Targetがしてはいけないこと

- Web専用のApplication modelやresult modelを作る
- native SQLやCapability implementationを複製する
- ValidationやAuthorization ruleをWeb専用ruleとして書き直す
- HTMLやframeworkの都合をSemantic Kernelへ持ち上げる
- expectedなOutcomeとExecution Errorを一種類のWeb errorへ畳む
- HTTP statusやUI messageからSemantic Outcomeを逆定義する
- read-only Datasource制約をUI都合で無視する

### Custom UI

少なくとも一箇所でCustom UIを接続し、標準Projectionだけに閉じないことを確認します。

### Gate Cへ進む条件

- DriverとWebが同じApplication / Patternを利用している
- Web固有情報をCanonical Semantic Modelへ追加していない
- Business Result / Execution Error / Observationの意味境界を維持している
- Web Adapterを外してもCanonical Worldが変わらない
- 標準ProjectionとCustom UIを併存できる
- Source / Application / Targetの来歴を追跡できる
- 共通候補とWeb固有差分を文書化できる
- Reference Scenarioの主要な測定点を取得できる

## Gate C — Editor / AIを具体化してよい

Gate Cを通過すると、Application上の意味、Driver実行結果、第一Targetとの差が実資産として揃います。

この時点で、EditorやAI支援が何を見せ、何を探索し、何を確認すべきかを具体化します。

`ISSUE-000008`の保留を解除する判断は、このGateを基準に行います。

## Stage 4 — Editor / AI支援

### Editor

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

- 既存Pattern / Capabilityの探索
- 既回答事項の再質問抑止
- 1:N、履歴、状態遷移、認証等の典型論点の確認提案
- AI提案と採用済みDecisionの分離
- Decision変更履歴
- 提案の無視・抑制・無効化

## Gate D — 第二Targetで検証できる

第二Targetへ進むために必要なのは、第一Web Targetで「共通と思われた部分」と「Web固有と判断した部分」が追跡可能なことです。

Editor / AIが完成していること自体をGate Dの条件にはしません。

## Stage 5 — 第二Target

第二Targetは対応Target数を増やすためではなく、Target-neutral境界を圧力試験するために利用します。

Webとの差が十分に表れるTargetを、その時点のEvidenceから選びます。Desktop、API、MCP、VUI等が候補です。

第二TargetのためにCore変更が必要になった場合も、それ自体を失敗とはみなしません。第一Web Targetの都合を誤って共通意味へ持ち上げていなかったかを検証します。

## Evidenceの残し方

各Stageでは「実装した」という記録だけで完了にしません。

少なくとも次のいずれかを残します。

- Reference source
- deterministic fixture
- Conformance test
- golden artifact
- Driver Scenario
- Target comparison
- ADR
- 未解決事項を明示した監査文書

実験結果が予想と異なる場合、その差を隠して既存計画へ合わせません。Product Valueに照らして意味のある差であれば、Pattern、Requirement、Milestone側を更新します。

## 判断を保留する事項

次は現時点で固定しません。

- Application Patternの網羅的な標準一覧
- 後続PatternのDSL最終構文
- 第一Web Targetのframework
- Editorの最終UI framework
- 第二Targetの種類
- B2C向けの一律性能目標
- Authentication ReferenceをDatabase、LDAP、IdP、外部API等のどれへ固定するか
- `std.auth.*`の標準契約
- Application単位の分散transactionを一般化する方法

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

EditorとAI支援の完成、第二Target、production HAはv1の前提条件にはしません。v1で成立したApplication vertical sliceを基盤として後続段階で拡張します。

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
- `plan/issue/000010-application-pattern-validation-integration.md`
- `plan/issue/000011-order-entry-browse-reference.md`
