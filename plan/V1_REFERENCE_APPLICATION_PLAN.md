# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、既存の`order_entry`を中心に一つの流れとして整理するものです。

Application Pattern、既存認証Compatibility、Capability結果境界、Driver、第一Web Target、Editor、AI支援を個別に実装するだけでは、「一つの業務ApplicationがTEIの各層をどのように通るのか」が見えにくくなります。

そこで、同じReference Applicationを段階的に育て、前の段階で確認した意味を次の段階がそのまま利用する進め方を採ります。

期限は置きません。次へ進む条件は日付ではなく、Repository内のReference、Conformance、Golden、Driver Scenarioなどから確認できるEvidenceで判断します。

## v1 Reference Application

v1では、新しい理想化されたサンプルを別に作らず、既存の`reference/workspace/applications/order-entry`を育てます。

`order_entry`には、すでに次の資産があります。

- `sales`と`customer`のSemantic Namespace
- `sales_core`へのread-write接続
- `master_data`へのread接続
- Database Contract
- native SQL Query
- Semantic Relation `sales.order.customer`
- Semantic Capability `sales.order.submit`
- Application Pattern
- Source-to-Canonical Traceability
- Reference Workspaceと再生成可能なCanonical World

M0.xxで積み上げたDatabase / Provider / Source TraceabilityのEvidenceを捨てず、Application側のv1実証へ接続します。

## 現在地

Application Patternは、候補5種類のうち3種類が実Referenceまで成立しています。

| Pattern | 状態 | 実Reference |
| --- | --- | --- |
| `browse` | 完了 | `order_entry.recent_orders` |
| `record` | 完了 | `order_entry.order_record` |
| `reference` | 完了 | `order_entry.customer_reference` |
| `command` | 次の優先単位 | `sales.order.submit`へ接続予定 |
| `header-detail` | 未着手 | `OrderLine`追加後に検証 |

現在のReference sourceは次です。

```text
applications/order-entry/
├─ application.tei
├─ patterns/
│  ├─ recent-orders.tei
│  ├─ order-record.tei
│  └─ customer-reference.tei
├─ queries/
└─ realizations/
```

Patternの詳細な設計基線は[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)を正とします。

既存認証Compatibility #68とCapability Outcome #77は、Gate Aへ向けて引き続き並行作業です。Stage 1全体、Gate A、Milestone 1の完了はまだ宣言しません。

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

このシナリオをApplication Pattern、Driver、Web Targetで別々に作り直しません。

同じApplication上の意味を共有し、それぞれの層では必要なProjection、Binding、実行状態だけを追加します。

## Application Patternで現在確認できている意味

### `browse`

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

一覧取得の対象とQueryをApplication上の意図として保持します。native SQLやQueryのDatasourceは複製しません。

### `record`

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

Applicationが`sales.order`を「一件」として扱う意味だけを保持します。一件の取得方法や画面遷移はPatternへ固定しません。

### `reference`

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

Semantic Relationが持つsource / target / cardinalityを再利用し、Application固有のlogical datasourceだけを組み合わせます。

`master`のaccess modeはApplication定義が正本であり、Patternへ`read-only`を重複記述しません。`reference`ではwrite専用Datasourceを拒否し、read / read-writeを読み取り可能として扱います。

Reference Conformanceでは、Relation target `customer.customer`と、`master`が要求する`master_data.customer ^2`の`read customer.customer`まで一続きに追跡できます。

## 今後Referenceへ追加する意味

### `command`

次は既存の`sales.order.submit` Capabilityへ接続します。

```text
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

`command`を単なるRecord更新へ還元せず、業務上の操作としてCapability identityへ接続します。

最小候補は次です。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

ただし、Capability以外のfieldを本当に必要とするかは実装前に既存Binding / Driver境界を確認して決めます。入力DataやDatasourceをCapability定義・Bindingから導けるなら、Patternへ重複記述しません。

### `OrderLine`と`header-detail`

受注明細に相当する`OrderLine`は必要ですが、`command`実証と同時には追加しません。

`header-detail`を検証する段階で、Semantic側へ最小の`OrderLine`と対応Relationを追加します。

概念上は次の関係です。

```text
Order 1 ── N OrderLine
```

最初から販売管理全体を再現する必要はありません。header-detail、複数fieldにまたがるValidation、金額計算を確認できる最小構成を選びます。

Semantic Relation、Application上の`header-detail`、Database Foreign Keyは別の意味です。1:N関係が存在するという理由だけでFKや画面構造を暗黙生成しません。

### Validation

少なくとも次の二種類を扱います。

1. 一つのfieldだけで判定できるValidation
2. HeaderとDetail、または複数fieldをまたぐValidation

単純なHTML入力制約やDatabase constraintだけでは表せないApplication上のruleを確認します。

ValidationをPatternが直接参照する構造にするかは、実Referenceで必要性を確認してから決めます。

### Authorization

既存認証Compatibility #68はApplication Patternと並行して進めます。

認証・認可の都合で`order_entry`の業務モデルをTEI標準User schemaへ作り直しません。Application利用者のAuthentication / AuthorizationはSemantic Capabilityとして扱い、既存schemaや認証製品はBinding / implementation側へ残します。

基本境界は[ADR-0019](adr/0019-authentication-capability-boundary.md)、実装順とConformanceは[`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)を参照します。

Application利用者のAuthorizationと、Plugin / RuntimeのPermission Grant / Resource Leaseは別の責務です。

## CapabilityのBusiness ResultとExecution Error

Capability結果の共通境界は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)、GitHub Issue #77で扱います。

基本境界は次です。

```text
Business Result
  ├─ 既存output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Capability呼び出しそのものを完了できなかった状態
```

Authenticationの`denied`のような期待される業務上の結末をExecution Errorへ変換しません。一方、Database接続失敗やadapter異常を業務Outcomeへ押し込みません。

`command` Patternを追加するという理由だけで、現行`sales.order.submit`の`output`を名前付きOutcomeへ書き換えません。Commandを正常に実行して初めて分かる複数の業務上の結末が本当に必要な場合に、#77の圧力試験として扱います。

## 全体の作業順

```text
Stage 0  Product Baseline / Gap Audit
               ↓
Stage 1  Application Pattern ─────────┐
               │                      │
               │              既存認証Compatibility
               │                      │
               │              Capability Outcome
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

Stage 1、既存認証Compatibility、Capability Outcomeは、互いの境界を壊さない範囲で並行して進められます。

Database / Provider側の作業も止めません。ただし、v1 Referenceに必要な境界や安全上の重要課題を優先し、PostgreSQL固有機能を増やすこと自体を目的にはしません。

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

### この段階で確認したこと

Database / Provider、Source / Meaning、Runtime安全境界に比べ、Application Pattern、既存認証、Driver、Target、Editor、AI支援の実証が薄いことが明確になりました。

この偏りを受け、主線をApplication vertical sliceへ移しています。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67  
詳細: [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)

### 状態

進行中です。

`browse`、`record`、`reference`は実Referenceまで完了しました。

次は`command`、その後`header-detail`です。

### ここまでのEvidence

- ApplicationがPatternを所有する最小Canonical model
- `applications/*/patterns/*.tei` Loader
- Canonical ID順Normalization
- `application-pattern:<Canonical ID>` Source Index
- Pattern ID tokenへのsource range
- Pattern固有のWorkspace Validation
- Pattern Diagnosticの共通`validation.Validate`統合
- Repository自身の`tei world`によるGolden再生成
- `browse`実Referenceでのfile rename invariance
- `browse` / `record` / `reference`の同一Application内での共存
- `reference`からread可能DatasourceとDatabase Contractの接続を追跡するConformance

### 現在の作業順

1. `sales.order.submit`を使って`command`の最小契約を実証する
2. CapabilityのBusiness Result / Execution Error境界と無理なく接続できることを確認する
3. `OrderLine`と対応RelationをSemantic側へ追加する
4. `header-detail`を実証する
5. 必要なApplication ValidationのReferenceを追加する
6. 5 Patternを同じApplication上の意味として説明できるか監査する
7. #68 / #77の成果と合わせてGate Aへ渡すEvidenceを揃える

### Gate Aへ進む条件

Application Pattern側では、少なくとも次を満たします。

- 5 Patternを同じApplication定義から識別できる
- Pattern identityがfile pathへ依存しない
- Patternの正本とEditor内部状態を暗黙の二重正本にしていない
- Semantic Data / Relation / CapabilityとApplication Patternの責務を説明できる
- Semantic RelationとPersistence FKを混同していない
- native SQLを別のDSLへ複製していない
- logical datasourceのaccess制約を維持している
- `command`がCapability identityへ接続されている
- Driverが利用できるTarget非依存情報が揃っている
- Source Index / DiagnosticからPattern定義へ戻れる
- Reference / Conformanceから上記を再現できる

現時点ではStage 1、Gate A、Milestone 1の完了は宣言しません。

## 並行Stage — 既存認証Compatibility

関連: `ISSUE-000003` / GitHub Issue #68  
詳細: [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)

### 状態

進行中です。

ADR-0019でAuthentication / Authorizationの意味境界を定め、ADR-0020 / #77で業務OutcomeとExecution Errorの共通境界を整理しています。

既存認証fixture、複数の実現方式、`order_entry`のbrowse / edit / submit Authorizationへ接続するReference Evidenceは引き続き整備します。

### 目的

既存システム固有の認証・認可仕様を、最初にTEI標準へ作り替えなくてもApplicationから利用できるCapabilityへ接続できることを確かめます。

### Gate Aへ渡すもの

- authenticated subjectを表すidentity
- Application上で認可対象となるaction / resource
- Authentication / Authorization Capability identity
- Capability BindingとMappingの来歴
- Business Result / Execution Errorの結果契約
- 既存仕様のうち認証・認可判断に使う条件
- secret materialを解決する境界

Gate Aでは認証基盤全体がproduction-readyであることは求めません。既存仕様を保持したままApplicationへ接続できる意味境界を確認します。

## Gate A — Application上の意味をDriverへ渡せる

Gate Aは、Stage 1、既存認証Compatibility、Capability Outcomeの成果をDriverへ渡す接続点です。

次の質問にRepository上のEvidenceから答えられることを条件とします。

- このApplicationは何をbrowseするのか
- 一件のrecordとは何か
- どのRelationを、どのDatasourceからreferenceするのか
- HeaderとDetailの関係は何か
- どのValidationを実行するのか
- どのCapabilityをcommandとして呼ぶのか
- 誰がその操作を行えるのか
- どのDatasourceがread / read-write / writeなのか
- CapabilityのBusiness ResultとExecution Errorをどう区別するのか

これらに答えるためにWeb画面やEditor内部状態を読む必要がある場合、Gate Aは未達とします。

## Stage 2 — Application Driver

関連: `ISSUE-000006` / GitHub Issue #69

### 目的

UIを作る前に、Gate Aで揃えたApplication上の意味を実行し、設計の整合を確かめられるようにします。

最初のDriverはproduction Runtimeの簡易版ではなく、設計途中のApplicationを動かし、意味上の矛盾を確認するための実証環境です。

### 最小Scenario

1. `browse`で受注一覧を取得する
2. `record`で一件を開く
3. `reference`で顧客を解決する
4. `header-detail`で明細を編集する
5. Validationを実行する
6. Authorizationを確認する
7. `command`で`sales.order.submit`を呼び出す
8. CapabilityのBusiness ResultとExecution Errorを区別する
9. 必要な場合は、外部世界で得られた状態をObservationから確認する

### Driverがしてはいけないこと

- Application ruleをDriver専用Goコードとして重複実装する
- native SQLをDriver側へコピーする
- logical datasourceを一つのin-memory DBへ意味上統合する
- Target固有Presentationを必要条件にする
- Driver独自のOutcome体系を作る
- HTTP statusやGUI messageをDriver Coreの意味として固定する
- read-only / read-write境界を無視する
- Driver実行ログをCanonicalの正本へ昇格させる

### Gate Bへ進む条件

- ScenarioをRepositoryだけから再現できる
- Application / Pattern / Query / Relation / Capability / Datasourceの解決結果を追跡できる
- `master_data`へwriteしないことを検証できる
- Validationが成立しない場合にcommandへ進まない
- Authorizationが成立しない場合にcommandへ進まない
- 既存`output`と名前付きOutcomeを同じBusiness Result境界から扱える
- Execution Errorを名前付きOutcomeや業務result codeへ押し込まない
- command acknowledgementとobserved resultを必要に応じて区別できる
- Driver固有状態がCanonical Worldへ逆流しない
- 同じApplication定義を第一Targetへ渡せる

## Gate B — UIなしで業務シナリオを説明できる

Gate Bでは、画面がなくてもApplicationの主要な振る舞いを実行・検証できることを確認します。

ここを通る前にWeb実装へ進むと、Application上の不足をWeb固有コードで埋める危険があります。第一Web Targetの実装はGate Bを原則的な前提とします。

## Stage 3 — 第一Web Target

関連: `ISSUE-000007` / GitHub Issue #70

### 目的

Driverで確認済みのApplication上の意味をWebへProjectionし、複数Targetで共有できそうなPresentation上の意味とWeb固有実装の境界を実例から抽出します。

### 対象

- `browse`
- `record`
- `reference`
- `header-detail`
- `command`
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
- Datasource access制約をUI都合で無視する

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

この時点で、EditorやAI支援が何を見せ、何を探索し、何を確認すべきかを具体化します。

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

実験結果が予想と異なる場合、その差を隠して既存計画へ合わせません。Product Valueに照らして意味のある差であれば、Pattern、Requirement、Milestone側を更新します。

## 判断を保留する事項

次は現時点で固定しません。

- Application Patternの網羅的な標準一覧
- `command` / `header-detail`の最終DSL
- reference Relation targetとRequired Contract Readsの横断ValidationをCoreへ昇格するか
- Validation / AuthorizationをPatternが直接参照するか
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

EditorとAI支援の完成、第二Target、production HAはv1の前提条件にはしません。それらをv1へ無理に詰め込まず、v1で成立したApplication vertical sliceを基盤として後続で拡張します。

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

## 関連GitHub Issue

- #67 — Application Pattern実証
- #68 — 既存認証Compatibility
- #69 — Application Driver
- #70 — 第一Web Target
- #77 — Capability Outcome / Execution Error
- #104 — `record`実証、完了
- #107 — `reference`実証、完了
