# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、既存の`order_entry`を中心に一つの流れとして整理するものです。

個別のIssueには、それぞれの設計課題や完了条件があります。しかし、Application Pattern、既存認証Compatibility、Driver、第一Web Target、Editor、AI支援を別々に進めるだけでは、「一つの業務ApplicationがTEIの各層をどのように通るのか」が見えにくくなります。

そこで、同じReference Applicationを段階的に育て、前の段階で確認した意味を次の段階がそのまま利用する進め方を採ります。

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
- Source-to-Canonical Traceability
- Reference Workspaceと再生成可能なCanonical World

これらを捨てずに使うことで、M0.xxで積み上げたEvidenceをApplication側のv1実証へ接続します。

Application Patternについても実装を開始しており、`browse`を受け止める最小Canonical model、Parser、`patterns/*.tei` Loader、Normalization、Source IndexまではRepositoryへ入っています。ただし、実際の`order_entry`にはまだPatternのReference sourceがなく、Stage 1やGate Aを完了した段階ではありません。

## v1で確認する業務シナリオ

最初の業務シナリオは、「受注を探し、開き、顧客を参照し、明細を編集し、内容と利用権限を確認して確定する」という一続きの操作とします。

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

このシナリオをApplication Pattern、Driver、Web Targetで別々に作り直しません。同じApplication上の意味を共有し、それぞれの層では必要なProjectionやBindingだけを追加します。

## 実証するApplication Pattern

最初のReferenceでは、次の5種類を候補として扱います。

- `browse`
- `record`
- `reference`
- `header-detail`
- `command`

これらは画面テンプレートの一覧ではなく、Targetが変わっても残る可能性があるApplication上の意図として検証します。現時点で5分類をTEI全体の永久的な標準語彙とはみなしません。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に定めます。現在の実装Evidence、次の最小縦切り、保留事項は[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)を正とします。

## Referenceへ追加する意味

### `OrderLine`

受注明細に相当する`OrderLine`は必要ですが、Pattern基盤の最初の検証と同時には追加しません。

`browse`を受け止めるmodel、Parser、Loader、Normalization、Source Indexはすでに実装済みです。次は既存の`order_entry.recent_orders`とnative SQLを実Reference Patternから参照し、Workspace参照整合性、Diagnostic、Reference World、実Referenceでのfile rename invarianceまでを一続きに閉じます。

その後、既存のSemantic Data、Relation、Capabilityを使って`record`、`reference`、`command`を追加し、最後に`header-detail`を検証する段階で`OrderLine`を導入します。

概念上は次の関係です。

```text
Order 1 ── N OrderLine
```

最初から販売管理全体を再現する必要はありません。header-detail、複数fieldにまたがるValidation、金額計算を確かめられる最小構成を選びます。

Semantic Relation、Application上のheader-detail、Database Foreign Keyは別の意味として扱います。1:N関係が存在するという理由だけで、FKや画面構造を暗黙に生成しません。

### Validation

少なくとも次の二種類を扱います。

1. 一つのfieldだけで判定できるValidation
2. HeaderとDetail、または複数fieldをまたぐValidation

これにより、単純なHTML入力制約やDatabase constraintだけでは表せないApplication上のruleを確認します。

### Command

受注確定には既存の`sales.order.submit` Capabilityを利用します。

`command` Patternを単なるRecord更新へ還元せず、業務上の操作としてCapabilityへ接続します。

Capabilityの結果境界は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に従います。既存の単一`output`と名前付きOutcomeはいずれもBusiness Resultとして扱い、Capability呼び出しそのものを完了できなかった状態はExecution Errorとして分離します。

`command` Patternを追加するという理由だけで、現行`sales.order.submit`の`output`を名前付きOutcomeへ書き換えません。Commandを正常に実行して初めて分かる業務上の分岐がReferenceで本当に必要な場合に、Capability Outcomeの圧力試験として扱います。

### Authorization

既存認証CompatibilityはApplication Patternと並行して進めます。

認証・認可の都合で`order_entry`の業務モデルをTEI標準User schemaへ作り直しません。Application利用者のAuthentication / AuthorizationはSemantic Capabilityとして扱い、既存schemaや認証製品はBinding / implementation側へ残します。

基本境界は[ADR-0019](adr/0019-authentication-capability-boundary.md)、実装順とConformanceは[`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)を参照します。

Authenticationの結果についてもADR-0020を利用します。たとえば`authenticated`と`denied`は、認証処理を正常に完了して得られる名前付きOutcomeとして表せます。Database接続失敗やadapter異常など、認証判断そのものを完了できなかった状態はExecution Errorとして外側へ残します。

この境界はAuthentication専用ではなく、[`ISSUE-000009`](issue/000009-capability-outcome-and-execution-error.md) / GitHub Issue #77でCapability共通の実装・検証課題として扱います。

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

Database / Provider側の作業も止めません。ただし、v1 Referenceに必要な境界や安全上の重要課題を優先し、PostgreSQL固有機能を増やすこと自体を目的にはしません。

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

この偏りを受け、以後の主線をApplication vertical sliceへ移します。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67  
詳細: [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)

### 状態

進行中です。

PR #87〜#89で、`browse`を受け止める次の基盤までは実装済みです。

- Applicationが所有するPatternの最小Canonical model
- `browse` Parser
- `applications/*/patterns/*.tei` Loader
- Canonical ID順のNormalization
- `application-pattern:<Canonical ID>`のSource Index
- Pattern ID tokenへのsource range

一方、実際の`order_entry`にはまだ`patterns/`がありません。したがって、実Reference、Workspace参照整合性、Reference World、後続Patternは未完了です。

### 目的

`order_entry`の業務シナリオを、Target固有UIへ依存しないApplication上の意図として表現できることを確かめます。

### 現在の作業順

新しい業務SemanticとPattern基盤を一度に変更しません。実装済みの基盤を踏まえ、次の順で進めます。

1. Patternが参照するData / Query / DatasourceのWorkspace参照整合性ValidationとDiagnosticを整える
2. 既存の`order_entry.recent_orders`とnative SQLを使い、`browse`を実Reference Patternとして追加する
3. Patternを含むReference Worldの再生成一致と、実Referenceでのfile rename invarianceを確認する
4. 既存のSemantic Data、Relation、Capabilityを使って`record`、`reference`、`command`を追加する
5. `OrderLine`と対応RelationをSemantic側へ追加し、`header-detail`を検証する
6. 5 Patternを同じApplication上の意味として説明できるか監査し、Gate Aへ渡すEvidenceを揃える

この順序により、Pattern基盤の不具合と、新しい業務Semanticの不足を切り分けやすくします。

PR #89ではSource IndexのSubject identityとArtifact pathについてfile renameの挙動を確認していますが、実Referenceを含むCanonical World全体のrename invarianceまで完了したとはみなしません。

### Gate Aへ進む条件

次を満たしたら、Application Pattern側はGate Aへ合流できる状態とみなします。

- 5 Patternを同じApplication定義から識別できる
- Patternのidentityがfile pathへ依存しない
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

### 現在の設計基線

Authentication / Authorizationの責務分離はADR-0019、CapabilityのBusiness Result / Execution Error境界はADR-0020で決定済みです。

ただし、既存認証fixture、複数の実現方式、`order_entry`へのAuthorization接続など、Reference / Conformanceによる実装Evidenceはこれから揃えます。

### 作業順

1. Reference Workspace内の`security`意味空間へAuthentication / Authorizationの最小Referenceを置く
2. ADR-0020に従い、Authenticationの`authenticated` / `denied`とExecution Errorを区別できるCapability結果契約を実装する
3. user ID、credential reference、disabled、有効期間、失敗回数、lock、organization、role、system-specific ruleを含むlegacy認証fixtureを用意する
4. 既存schemaや既存functionをCapabilityへ接続する
5. 同じCapability identityを保ったまま、異なる実現方式を一つ追加する
6. secretがCanonical World、Source Index、Durable artifact、通常logへ混入しないことをConformance化する
7. `order_entry`のbrowse / edit / submit Authorizationへ接続する
8. Source / Binding / Mappingの来歴を確認し、Stage 1と合わせてGate Aを監査する

### Gate Aへ合流する条件

- Authentication / Authorizationを別のCapability identityとして扱える
- legacy fieldやfunctionをCapabilityへMappingできる
- secret値をCanonicalな正本へ保存しない
- Application利用者の認可とPlugin / Runtime Permissionを混同しない
- Applicationが認証実装の物理構造へ直接依存しない
- 同じCapability identityのまま実現方式を交換できる
- Authenticationの名前付きOutcomeとExecution Errorを区別できる
- `order_entry`のbrowse / edit / submit Authorizationへ接続できる

Gate Aでは、認証基盤全体がproduction-readyであることは求めません。既存仕様を保持したままApplicationへ接続できる意味境界が成立していることを確認します。

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
- Capabilityが返したBusiness Resultと、呼び出しを完了できなかったExecution Errorをどう区別するのか

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

Business Resultは、既存の単一`output` valueまたは名前付きOutcomeとして扱います。期待した`denied`や`rejected`等のOutcomeをDriver障害へ変換せず、Execution Errorとも混同しません。

### Driverがしてはいけないこと

- Application ruleをDriver専用Goコードとして重複実装する
- native SQLをDriver側へコピーする
- logical datasourceを一つのin-memory DBへ意味上統合する
- Target固有Presentationを必要条件にする
- Driver独自のOutcome体系を作る
- HTTP statusやGUI messageをDriver Coreの意味として固定する
- Driver実行ログをCanonicalの正本へ昇格させる

### Gate Bへ進む条件

- ScenarioをRepositoryだけから再現できる
- Application / Pattern / Query / Capability / Datasourceの解決結果を追跡できる
- `master_data`へwriteしないことを検証できる
- Validationが成立しない場合にcommandへ進まない
- Authorizationが成立しない場合にcommandへ進まない
- 既存`output`と名前付きOutcomeを同じBusiness Result境界から扱える
- Execution Errorを名前付きOutcomeや業務result codeへ押し込まない
- command acknowledgementとobserved resultを必要に応じて区別できる
- Driver固有の状態がCanonical Worldへ逆流しない
- 同じApplication定義を第一Targetへ渡せる

## Gate B — UIなしで業務シナリオを説明できる

Gate Bでは、画面がなくてもApplicationの主要な振る舞いを実行・検証できることを確認します。

ここを通る前にWeb実装へ進むと、Application上の不足をWeb固有コードで埋める危険があります。したがって、第一Web Targetの実装はGate Bを原則的な前提とします。

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
- expectedな名前付きOutcomeとExecution Errorを一種類のWeb errorへ畳む
- HTTP statusやUI messageからSemantic Outcomeを逆定義する
- 既存`output`をWeb都合の暗黙`success` Outcomeへ変換する
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
- Reference Scenarioの主要な測定点を取得できる

## Gate C — Editor / AIを具体化してよい

Gate Cを通過すると、Application上の意味、Driver実行結果、第一Targetとの差が実資産として揃います。

この時点で初めて、EditorやAI支援が何を見せ、何を探索し、何を確認すべきかを具体化できます。

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

- 既存Patternの探索
- 既存Capabilityの探索
- 既回答事項の再質問抑止
- 1:N、履歴、状態遷移、認証などの典型論点の確認提案
- AI提案と採用済みDecisionの分離
- Decision変更履歴
- 提案の無視・抑制・無効化

## Gate D — 第二Targetで検証できる

Editor / AIが完成していること自体をGate Dの条件にはしません。

第二Targetへ進むために必要なのは、第一Web Targetで「共通と思われた部分」と「Web固有と判断した部分」が追跡可能なことです。

第二Targetは、その分類が本当に妥当だったかを検証するために利用します。

## Stage 5 — 第二Target

第二Targetは、単なる対応Target数の増加ではなく、Target-neutral境界の圧力試験です。

選定時には、Webとの差が十分に表れるTargetを優先します。Desktop、API、MCP、VUIなどの候補から、その時点で最も設計上の差を引き出せるものを選びます。

第二TargetのためにCore変更が必要になった場合は、それ自体を失敗とはみなしません。変更理由を分析し、第一Web Targetの都合を誤って共通意味へ持ち上げていなかったかを確認します。

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

実験結果が予想と異なる場合、その差を隠して既存計画へ合わせません。Product Valueに照らして意味のある差であれば、PatternやRequirement、Milestone側を更新します。

## 判断を保留する事項

次は現時点で固定しません。

- Application Patternの網羅的な標準一覧
- 後続PatternのDSL最終構文
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

EditorとAI支援の完成、第二Target、production HAはv1の前提条件にはしません。それらをv1へ無理に詰め込まず、v1で成立したApplication vertical sliceを基盤としてv2以降で拡張します。

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
