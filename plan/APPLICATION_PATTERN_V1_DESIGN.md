# Application Pattern v1 設計・実装基線

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で進めるApplication Patternについて、v1で守る設計境界、現在までに得られた実装Evidence、次に閉じるべきGapを一つの基線として整理するものです。

TEIでは、Application Patternを画面テンプレートの一覧として先に固定しません。既存の`order_entry`を使い、Applicationが業務上どのような意図を持つかをTarget固有UIから切り離して表現し、その意味をDriverや将来の複数Targetから再利用できるかを一つずつ実証します。

Patternの初期所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

この文書で扱う`browse`、`record`、`reference`、`command`、`header-detail`はv1 Referenceで検証する候補です。5分類をTEI全体の永久的な標準語彙として先に確定するものではありません。

## 現在地

Stage 1のApplication Pattern実証は進行中です。

現時点では、次の3種類について実Referenceまで閉じています。

1. `browse`
2. `record`
3. `reference`

次の主対象は`command`です。`command`が成立した後、`OrderLine`と対応するSemantic Relationを追加し、最後に`header-detail`を検証します。

Stage 1、Gate A、Milestone 1はまだ完了扱いにしません。

## ここまでに成立した共通基盤

PR #87〜#103により、Application Patternを扱うための共通基盤を整えました。

### Canonical model

PatternはApplicationが所有します。

```text
Application
├─ ID
├─ SemanticUses
├─ Databases
└─ Patterns
   ├─ Pattern
   └─ ...
```

PatternにはApplication上の意図だけを保持し、Semantic定義、Query本文、物理path、Target固有Presentationを複製しません。

現行modelは概ね次の情報を持ちます。

```text
Pattern
- ID
- Kind
- Data
- Relation
- Query
- Datasource
```

`command`実証では、ここへ必要最小限の`Capability`参照を追加する候補を検証します。

### Authoring source

Patternの正本はApplication配下の`patterns/*.tei`です。

```text
applications/order-entry/
├─ application.tei
├─ patterns/
├─ queries/
└─ realizations/
```

v1では一ファイル一Patternから始めます。

Patternのidentityはファイル名やdirectory名から導出しません。authoring sourceに明示されたCanonical IDを正とします。

### Loader / Normalization

`applications/*/patterns/*.tei`をWorkspace Loaderが読み込み、所属Applicationの`Patterns`へ統合します。

PatternはCanonical ID順に正規化されるため、ファイル列挙順やファイル名をCanonicalな意味へ持ち込みません。

### Source Index / Diagnostic

Patternには`application-pattern:<Canonical ID>`というSubjectを用います。

Source IndexはSubjectからPattern宣言中のCanonical ID tokenへ戻れるため、Validationで見つけた不整合を元のauthoring sourceへ投影できます。

`browse`の実Referenceでは、ファイル名を変更してもCanonical WorldとSubject identityが変化せず、Source Artifact pathだけが追随することまで確認しています。

### 共通Validation

Pattern固有Validationは通常の`validation.Validate`へ統合済みです。

Parserでは構文だけで確定できる条件を扱い、Workspace全体を見なければ分からない参照整合性はValidation層で扱います。

同じ原因について既存ValidationとPattern Validationが重複Diagnosticを大量に出さないことも重視します。

## `browse` — 完了した最初の縦切り

実Referenceは次です。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

### 現在の契約

- `data`: 必須
- `query`: 必須
- `datasource`: 任意
- `relation`: 受理しない

Query `order_entry.recent_orders`が`datasource sales`を正本として保持しているため、実ReferenceではPattern側へDatasourceを重複記述していません。

### Evidence

PR #87〜#103で次を一続きに確認しました。

- Canonical model / Parser
- `patterns/*.tei` Loader / Normalization
- Source IndexとPattern ID token range
- Data / Query / Datasourceの参照整合性
- Pattern固有Diagnosticのauthoring sourceへのprojection
- 共通`validation.Validate`への統合
- `order_entry.recent_orders`実Reference
- Repository自身の`tei world`によるGolden World再生成
- 実Referenceでのfile rename invariance

### 得られた判断

`browse`は新しいQuery定義を抱える層ではありません。既存QueryとSemantic DataをCanonical IDで結び、「Applicationが何を一覧として扱うか」という意図を表す層として成立しました。

Datasourceについても、Query側に正本がある場合はPatternへ同じ値を重複記述しない方が自然です。

## `record` — 完了

GitHub Issue #104、PR #105 / #106で実証しました。

実Referenceは次です。

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

### 現在の契約

- `data`: 必須
- `query`: 受理しない
- `datasource`: 受理しない
- `relation`: 受理しない

### 得られた判断

`record`は「ApplicationがどのSemantic Dataを一件として扱うか」という意味だけで成立しました。

既存Database Contractに一件取得相当の操作が存在することや、将来Driverが取得処理を必要とすることは、Pattern自身へQueryを追加する理由にはなりませんでした。

同様に、`sales.order`とlogical datasource `sales`の対応をPatternへ再記述する必要もありませんでした。

`record`をWebの詳細画面、Form、特定`browse`からの遷移先とは定義しません。画面遷移や入口の構造は、必要性が実Referenceから確認できるまでCanonicalな意味へ持ち込みません。

## `reference` — 完了

GitHub Issue #107、PR #108 / #109で実証しました。

実Referenceは次です。

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

### 現在の契約

- `relation`: 必須
- `datasource`: 必須
- `data`: 受理しない
- `query`: 受理しない

### Relationの再利用

Semantic Relation `sales.order.customer`は、参照元` sales.order `、参照先`customer.customer`、cardinalityをすでに保持しています。

そのため、Patternへsource Dataやtarget Dataを重複記述していません。

Semantic Relation、Database Foreign Key、UI上のlookup widgetは別の意味です。Relationを参照しただけでFKやUI部品を暗黙生成する規則は置きません。

### Datasource境界

`reference`は読み取り意図なので、write-only Datasourceは受理しません。

一方、read-write Datasourceも読み取り可能であるため受理します。`reference`だからDatasourceがread-onlyでなければならない、という一般則にはしていません。

access modeはApplicationのDatabaseRequirementを正本とし、Patternへ`read-only`等を重複記述しません。

### Database Contractとの関係

実Referenceでは、Datasource `master`が要求する`master_data.customer ^2` ContractがRelation target `customer.customer`をread対象として公開していることまで確認しています。

ただし、この一例だけを理由にPattern専用のContract体系は追加していません。

Semantic Relation、Database Contract、Application Datasourceの各境界は既存Validationで個別に確認できます。特定の`reference`が選んだDatasourceからRelation targetを読めるという横断条件は現時点ではReference Conformanceで観測し、Driverへ進む際に共通検査へ昇格させる必要性が増えた場合だけ切り出します。

## 次の実証 — `command`

Repository内の作業定義は[`ISSUE-000010`](issue/000010-command-application-pattern.md)、GitHub上の進行管理はIssue #110です。

既存Capabilityは次です。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

最小構文の第一候補は次です。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

### 初期仮説

`command`は「Applicationがどの業務Capabilityを呼ぶか」という意図だけを保持します。

そのため、最初は次の境界から検証します。

- `capability`: 必須
- `data`: 受理しない
- `relation`: 受理しない
- `query`: 受理しない
- `datasource`: 受理しない

Capabilityがすでに持つinput / outputをPatternへ複製しません。

### Capability Bindingとの分離

PatternはCapability identityを参照します。

どの実装を選ぶかはBinding / Realization / Driver側の責務です。Custom Go、process plugin、既存API等の実装方式をPatternへ持ち込みません。

同じCapabilityの実装方式を差し替えてもApplication上のcommand identityが変化しない構造を維持します。

### Business Result / Execution Error

Capability結果の共通境界は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に従います。

現行`sales.order.submit`は`output sales.order`を持っています。この既存outputを、`command` Patternを追加するという理由だけで名前付きOutcomeへ書き換えません。

Commandを正常に実行して初めて判明する複数の業務上の結末が本当に必要だと分かった場合だけ、GitHub Issue #77の圧力試験としてOutcomeを検討します。

Database接続失敗やadapter異常等はExecution Errorであり、業務Outcomeへ押し込みません。

### CommandとObservation

CapabilityのBusiness Resultと、その後に外部世界から得られるActual / Observationは別の意味です。

Application Patternではcommand intentまでを扱い、ObservationをPatternのfieldへ持ち込みません。両者の接続は後続Driverで扱います。

## `header-detail`と`OrderLine`

`header-detail`は最後に検証します。

既存Semanticにはまだ受注明細を表す`OrderLine`がありません。そのため、Pattern基盤だけを先に一般化せず、次の順で進めます。

1. `OrderLine`の最小Semantic Dataを追加する
2. `Order 1 ── N OrderLine`を表すSemantic Relationを追加する
3. そのRelationをApplication上の`header-detail`として利用する意味を検証する
4. Persistence FKやTarget固有UIとの境界をConformanceで確認する

Semantic Relation、Application上のheader-detail、Database Foreign Keyは別の意味単位です。

1:N関係があるという理由だけでPersistence FKや画面構造を自動決定しません。

## Pattern Kindごとの現在の意味

| Kind | 現在の最小意味 | 主な参照 | 状態 |
| --- | --- | --- | --- |
| `browse` | Applicationが何を一覧として扱うか | Data + Query | 実Reference完了 |
| `record` | Applicationが何を一件として扱うか | Data | 実Reference完了 |
| `reference` | 既存Relationをどのlogical datasourceから参照するか | Relation + Datasource | 実Reference完了 |
| `command` | どの業務Capabilityを呼ぶか | Capability | 次の実証対象 |
| `header-detail` | HeaderとDetailをApplication上でどの関係として扱うか | 未確定。Semantic Relationを中心に検証予定 | 未着手 |

この表は現在のEvidenceを要約したものであり、将来の全Applicationに一律適用する固定schemaではありません。

## Patternが保持しないもの

Application Patternへ次を安易に入れません。

- SQL本文
- Semantic DataやRelationの定義内容
- Database Contractの複製
- Capability input / outputの複製
- Custom Go等の実装方式
- HTML / DOM / Widget
- HTTP status
- 画面サイズやレスポンシブ指定
- file pathを由来とするidentity
- Driver固有状態
- Observation結果

必要性がReferenceから確認された場合は、その情報がApplication上の意味なのか、Projectionなのか、Bindingなのかを先に判定します。

## Validationの責務

### Parserで扱うこと

- Pattern IDが有効なCanonical IDである
- Kindが現時点で有効化されている
- fieldの綴りと型が正しい
- 同じfieldが重複していない
- Kindに必須のfieldが欠けていない
- Kindで禁止しているfieldが含まれていない

### Workspace Validationで扱うこと

- Data / Relation / Query / Capabilityが存在する
- Datasource aliasが対象Applicationに存在する
- Queryの所有Applicationとの整合
- Datasource accessなど、Workspace全体を見なければ判断できない条件
- Pattern Canonical IDの重複

既存Semantic / Database / Query Validationですでに保証されている条件を、Pattern側へ無条件に複製しません。

## Shared Pattern / Catalogをまだ作らない理由

複数Applicationで同じPatternを共有できそうに見えても、現段階ではApplication自身がPatternを所有します。

`order_entry`一例から一般化すると、業務固有の意味とTarget固有の都合を誤って標準化する危険があります。

少なくとも第一Web Targetと第二Targetまで圧力を掛け、何がApplication固有で、何が再利用可能なPatternやSemantic Patternとして共有できるのかをEvidenceから判断します。

## Driverへ渡す境界

Gate Aでは、Application Pattern側から少なくとも次をTarget非依存の情報として渡せる必要があります。

- browse対象とQuery identity
- record対象
- reference対象となるRelationとDatasource
- header-detailの関係
- commandとして呼ぶCapability identity
- Application Validation
- read / read-write Datasource境界
- PatternのCanonical identityとauthoring source

Authentication / Authorizationは並行する既存認証Compatibility側から合流します。

CapabilityのBusiness Result / Execution Error境界もADR-0020 / Issue #77から受け取ります。

## Gate Aへ進む条件

Application Pattern側は、次をReference / Conformanceから説明できる状態になったときGate Aへ合流できます。

- v1業務シナリオに必要なPattern候補を同じ`order_entry`から識別できる
- Patternのidentityがfile pathに依存しない
- Semantic / Query / Relation / Capabilityの正本をPatternへ複製していない
- native SQLを別DSLへ書き直していない
- read-only Datasourceの境界を破っていない
- `command`がCapability identityへ接続されている
- Semantic Relation、Application Pattern、Persistence Mappingを混同していない
- Source Index / Diagnosticからauthoring sourceへ戻れる
- 後続Driverが利用できるTarget非依存情報を説明できる
- Reference WorldをRepositoryだけから再生成できる

`browse`、`record`、`reference`の完了だけでは、この条件を満たしたとはみなしません。

## 実証順

現在の主線は次です。

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
  ↓
既存認証CompatibilityとGate Aで合流
```

この順序は期限ではありません。前段の意味をReference / Conformanceで確認できたことを、次へ進む条件とします。

## 非目標

Application Patternの実証では、次を目的にしません。

- 完成した画面設計
- Target-neutral Presentation IRの先行完成
- Workflow Runtimeの実装
- production transaction coordinator
- 全業務システムに共通するPattern Catalogの先行定義
- 5分類を不変の標準語彙として固定すること
- Editor専用metadataを第二の正本にすること

## 文書更新の原則

実装Evidenceが増えたら、この文書を進捗ログとして継ぎ足すだけにはしません。

現在地、成立した設計判断、残っている問い、次の実証順が自然に読めるよう、必要に応じて文書全体を再構成します。

その際も、実装済み・設計済み・未検証を明確に分け、Evidenceより強い主張へ書き換えません。
