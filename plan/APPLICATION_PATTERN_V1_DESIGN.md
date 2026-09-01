# Application Pattern v1 設計・実装基線

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で実証したApplication Patternについて、v1で確認できた設計基線と、今後も未確定として扱う境界を整理するものです。

Application Patternは、画面テンプレートを分類するための仕組みではありません。ApplicationがSemantic上の意味や既存の業務資産を、どのような意図で利用するかを保持する層です。Persistence上の実現方法やWebなどのTarget固有表現とは分離します。

v1では`order_entry`をReference Applicationとして、次の5種類を実証候補として一つずつ検証しました。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

5種類すべてが実Referenceまで成立しました。ただし、これは5分類をTEI全体の永久的な標準語彙として確定したことを意味しません。後続Driver、第一Web Target、第二Targetから得られるEvidenceによって、分類や所有境界は引き続き見直します。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

## 現在地

Application Pattern側のStage 1実証は完了しています。

現在mainで有効なPattern Kindは次の5種類です。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

いずれもCanonical model、Parser、Loader、Normalization、Workspace Validation、Source Index、実Reference、Golden Worldまで一続きに確認しています。

Reference Applicationでは次が共存します。

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

pattern order_entry.submit_order command {
    capability sales.order.submit
}

pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Application Pattern側の完了だけではGate A完了とはしません。Gate Aでは、既存認証Compatibility、Validation、CapabilityのBusiness Result / Execution Error境界などを合流させ、Driverへ一続きのApplication上の意味を渡せることを確認します。

## 共通の設計基線

### PatternはApplicationが所有する意図

PatternはSemanticそのものでも、Persistenceの定義でも、Target固有UIでもありません。

```text
Semantic / Query / Capability
          ↓ identityで参照
Application Pattern
          ↓ Driver / Projection
Realization・Target固有表現
```

各層が自分の意味だけを持ち、別の層がすでに保持する情報を不用意に複製しないことを優先します。

### PatternのCanonical model

現行Patternは次の型付き参照を持ちます。

```text
Pattern
- ID
- Kind
- Data
- Relation
- Query
- Capability
- Datasource
```

すべてのfieldをすべてのKindで使うわけではありません。Kindごとの意味に不要なfieldは暗黙に無視せず、ParserとWorkspace Validationの契約として受理しません。

任意の`role -> string` Mapへ一般化して型を失うことも避けます。

### Loader / Normalization

Patternの正本はApplication配下の`patterns/*.tei`です。

LoaderはPatternを所属Applicationへ統合し、Canonical ID順に正規化します。file nameやdirectory列挙順をCanonicalな意味にはしません。

### Source Index

Patternは明示的なCanonical IDを持ち、`application-pattern:<Canonical ID>` Subjectから既存Source Indexを通じてauthoring sourceへ戻れます。

PR #102では実Referenceのファイル名だけを変更し、Canonical WorldとPattern identityが変わらず、Source Artifact pathだけが追随することを確認しました。

file pathやEditor内部状態を第二のCanonical identityにはしません。

### ParserとWorkspace Validationの分担

Parserは、構文だけで判断できる条件を扱います。

- Kindの有効性
- 必須field
- 禁止field
- fieldの重複

Workspace全体を見なければ判断できない条件は共通Validationで扱います。

- Semantic Data / Relation / Capabilityの存在
- Queryの存在とApplication ownership
- Datasource aliasの存在とaccess
- `header-detail`が参照するRelationのcardinality

Pattern Diagnosticは既存`application-pattern:<Canonical ID>` Subjectを利用します。Pattern専用の別Diagnostic基盤は作りません。

## Pattern Kindごとの実証結果

### `browse`

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

契約は次です。

- `data`: 必須
- `query`: 必須
- `datasource`: 任意
- `relation`: 受理しない
- `capability`: 受理しない

実Referenceでは、Query `order_entry.recent_orders`がlogical datasource `sales`をすでに保持しているため、Patternへ同じDatasourceを重複記述していません。

`browse`から得た重要な判断は、既存Queryが持つ取得ロジックやDatasourceをPatternへ書き直さないことです。

### `record`

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

契約は次です。

- `data`: 必須
- `relation`: 受理しない
- `query`: 受理しない
- `capability`: 受理しない
- `datasource`: 受理しない

`record`は「ApplicationがどのSemantic Dataを一件として扱うか」という意図だけで成立しました。

将来Driverが一件取得を行うことやDatabase Contractに`find_by_id`が存在することは、取得QueryをPatternへ固定する理由にはなりませんでした。

### `reference`

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

契約は次です。

- `relation`: 必須
- `datasource`: 必須
- `data`: 受理しない
- `query`: 受理しない
- `capability`: 受理しない

Relation `sales.order.customer`がsource `sales.order`、target `customer.customer`、cardinalityを保持しているため、それらをPatternへ複製しません。

一方、どのlogical datasourceから参照するかはApplication固有の意図なので、`master`をPatternに保持します。Datasourceの`access read`自体はApplicationの`DatabaseRequirement`を正本とします。

`reference`は読み取り意図なので、write-only Datasourceは共通Validationで拒否します。read-write Datasourceは読み取り可能なので受理します。

### `command`

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

契約は次です。

- `capability`: 必須
- `data`: 受理しない
- `relation`: 受理しない
- `query`: 受理しない
- `datasource`: 受理しない

既存Capabilityは次の契約を持ちます。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

そのため、input、output、対象DataをPatternへ再記述しません。Capability implementationやDatasourceもBinding / Realization / Driver側の責務として分離します。

既存`output sales.order`はADR-0020に従うBusiness Resultとして維持し、`command`追加だけを理由にnamed Outcomeへ変更していません。Execution ErrorやObservationもPatternへ混在させません。

### `header-detail`

`header-detail`の前提となる業務Semanticは、Patternとは別の実証として先に追加しました。

```tei
type OrderLineId from std.identifier

data OrderLine {
    attribute orderLineId type sales.order_line_id
    attribute lineNumber type std.integer
    attribute itemCode type std.code
    attribute quantity type std.decimal
    attribute unitPrice type std.money
}
```

既存`Order`には次のRelationがあります。

```tei
relation lines to sales.order_line many
```

Application PatternはRelationだけを参照します。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

契約は次です。

- `relation`: 必須
- 参照Relationのcardinality: `many`を必須とする
- `data`: 受理しない
- `query`: 受理しない
- `capability`: 受理しない
- `datasource`: 受理しない

Relationがsource `sales.order`、target `sales.order_line`、cardinality `many`を持つため、Header Data、Detail Data、cardinalityをPatternへ重複記述しません。

`header-detail`はWebのHeader Form + Gridという画面テンプレートではありません。「この1:N Relationを一つのHeaderと複数Detailとして扱う」というApplication上の意図です。

## Semantic RelationとPersistenceを分離する

`Order 1:N OrderLine`のSemantic Relationを追加した段階でも、`header-detail` Patternを追加した段階でも、次を暗黙生成していません。

- `OrderLine` Persistence Mapping
- Database Foreign Key
- Database Contract拡張
- Migration

Semantic Relation、Application上の意図、物理DB上の実現方法は別の意味です。

`OrderLine`へ`orderId`属性も先回りして追加していません。物理DBでどの列やForeign Keyを使って関係を実現するかは、Persistence / Driver側の実証で判断します。

## 実証から得た共通原則

### 既存の意味を複製しない

- QueryのDatasourceを`browse`へ重複させない
- Relationのsource / target / cardinalityを`reference`や`header-detail`へ重複させない
- Capabilityのinput / outputを`command`へ重複させない
- Datasource accessをPatternへ複製しない

同じ意味を複数箇所へ手書きしないことを優先します。

### Kindごとに必要なfieldだけを持つ

実Referenceで成立した最小形は次です。

```text
browse        data + query
record        data
reference     relation + datasource
command       capability
header-detail relation
```

この形は現時点の実証結果であり、将来の全Applicationを網羅する完成catalogではありません。

### 実現方法をPatternへ持ち込まない

native SQL、Capability implementation、Persistence Mapping、Foreign Key、Migration、Web Widgetは、それぞれ既存の責務境界へ残します。

PatternはApplication上の意図だけを保持します。

### 未検証の一般化を先に固定しない

5種類が一つのReference Applicationで成立したことは重要なEvidenceですが、それだけで共有Pattern Catalog、継承、合成、汎用Form schema等を設計しません。

後続Driverと複数Targetから、どの意味が本当に共通化できるかを圧力試験します。

## Reference / Conformance Evidence

主な実証は次のPRで確認しています。

- `browse`: PR #101〜#103
- `record`: PR #105 / #106
- `reference`: PR #108 / #109
- `command`: PR #114 / #116
- `OrderLine` Semantic: PR #119
- `header-detail` Canonical契約: PR #121
- `header-detail`実Reference: PR #123

PR #119のCI #277、PR #121のCI #279、PR #123のCI #282を含め、主要な実証PRは`verify` / PostgreSQL integrationで成功しています。

Reference WorldはRepository自身の`tei world`から再生成でき、Pattern追加によるGolden差分をCanonicalな意味に限定しています。

## Gate Aへの引き継ぎ

Application Pattern側から後続へ渡せる情報は揃いました。

- browse対象
- record対象
- reference対象とlogical datasource
- header-detail Relation
- command Capability
- read / read-write Datasource境界
- PatternのCanonical identityとauthoring source

ただしGate A全体には、既存認証Compatibility #68とValidationのApplication上の構成が残っています。CapabilityのBusiness Result / Execution Error境界はADR-0020 / `CAPABILITY_OUTCOME_V1_DESIGN.md`を利用します。

Application Patternの完了を理由にDriver #69へ見切り発車せず、Gate Aの残条件をEvidenceで確認してから進みます。

## 非目標

このStage 1では次を確定していません。

- 5分類をTEI全体の永久的な標準語彙とすること
- Patternの網羅的なCatalog
- Pattern継承・合成
- 完成UI
- responsive design
- Detailの並び順やページング
- 汎用Form schema
- Persistence / FKの自動生成
- AIによるPattern推薦
- production-grade Workflow / Transaction設計

## 関連

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000010-command-application-pattern.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000012-header-detail-application-pattern.md`
- ADR-0017
- ADR-0018
- ADR-0020
- GitHub Issue #67 / #68 / #69 / #117 / #118
