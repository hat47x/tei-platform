# Application Pattern v1 設計・実装基線

- 状態: 実装済みの現行基線

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で実証したApplication Patternについて、v1で成立した設計境界とRepository上のEvidenceを、後続DriverやTargetへ引き継げる形で整理するものです。

Application Patternは、画面テンプレートを分類するための仕組みではありません。ApplicationがSemantic上の意味をどのような意図で利用するかを、Target固有のPresentationや物理的な実現方式から切り離して保持する層です。

v1ではReference Application `order_entry`を使い、具体的な業務シナリオから必要なKindとfieldを一つずつ抽出しました。

実証した5種類は次です。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

5種類はすべて実Referenceまで成立しています。ただし、これをTEI全体の永久的な標準語彙として確定したわけではありません。Driver、第一Target、第二Targetから得られるEvidenceによって、分類や所有境界は今後も見直します。

Patternの所有境界は[ADR-0017](../adr/0017-application-pattern-ownership.md)、authoring sourceとfile pathからのidentity分離は[ADR-0018](../adr/0018-application-pattern-authoring-source.md)に従います。

## 現在地

Application Pattern Stage #67は完了しています。

`order_entry`には次の5 Patternが共存します。

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

Application Pattern側から後続Driverへ渡すTarget非依存の意味は揃いました。

ただし、Gate Aはまだ完了していません。既存認証Compatibility #68とCapability Outcome / Execution Error #77のEvidenceを合流させてから評価します。

## Canonical model

PatternはApplicationが所有します。

```text
Application
├─ ID
├─ SemanticUses
├─ Databases
└─ Patterns
   ├─ browse
   ├─ record
   ├─ reference
   ├─ command
   └─ header-detail
```

現行のPattern modelは、次の小さな型付き参照だけを持ちます。

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

すべてのfieldをすべてのKindで使うわけではありません。Kindごとに必要なfieldだけを受理し、不要なfieldは明示的に拒否します。

Semantic定義、Query本文、Capability契約、Database Contract本文、file path、Target固有PresentationはPatternへ複製しません。

## Pattern Kindごとの契約

### `browse`

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

契約:

- `data`: 必須
- `query`: 必須
- `datasource`: 任意
- `relation`: 受理しない
- `capability`: 受理しない

実ReferenceではQuery `order_entry.recent_orders`自身がDatasource `sales`を正本として持つため、Patternへ同じDatasourceを重複記述していません。

`browse`から得た判断は、**既存Queryが持つ実行経路の意味をPatternへ二重記述しない**ことです。

### `record`

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

契約:

- `data`: 必須
- `relation`: 受理しない
- `query`: 受理しない
- `capability`: 受理しない
- `datasource`: 受理しない

`record`は、「ApplicationがどのSemantic Dataを一件として扱うか」という意図だけで成立しました。

一件取得のQueryやDatabase Contractが存在しても、それをPatternへ固定する必要はありませんでした。

### `reference`

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

契約:

- `relation`: 必須
- `datasource`: 必須
- `data`: 受理しない
- `query`: 受理しない
- `capability`: 受理しない

Relation `sales.order.customer`がsource `sales.order`、target `customer.customer`、cardinality `one`を保持するため、それらをPatternへ複製しません。

一方、logical datasource `master`はApplication固有の参照経路なのでPatternが保持します。

Datasource accessはApplicationの`DatabaseRequirement`を正本とします。`reference`は読み取り意図なので、`write`専用DatasourceはValidationで拒否し、`read` / `read-write`を受理します。

実Referenceでは、`master`が要求する`master_data.customer ^2`のContractがRelation target `customer.customer`を`read`対象として公開することまでConformanceで追跡しています。

この一例だけを理由にPattern専用のContract体系は追加していません。

### `command`

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

契約:

- `capability`: 必須
- `data`: 受理しない
- `relation`: 受理しない
- `query`: 受理しない
- `datasource`: 受理しない

Capability `sales.order.submit`はSemantic側で次を保持しています。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

そのため、input / outputや対象DataをPatternへ複製しません。

Capability implementationがCustom Go、Plugin、Database、外部API等へ変わっても、Application上のcommand identityを変えないため、実装方式やDatasourceもPatternへ持ち込みません。

既存`output sales.order`はADR-0020に従う通常のBusiness Resultとして維持し、`command`を追加しただけの理由でnamed Outcomeへ変換していません。

### `header-detail`

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

契約:

- `relation`: 必須
- `data`: 受理しない
- `query`: 受理しない
- `capability`: 受理しない
- `datasource`: 受理しない

Relation `sales.order.lines`はsource `sales.order`、target `sales.order_line`、cardinality `many`を保持しています。

そのため、Header Data、Detail Data、cardinalityをPatternへ複製しません。

`header-detail`は複数Detailを扱う意図なので、参照Relationが`many`であることをWorkspace Validationで確認します。`one` Relationは`APPLICATION_PATTERN_RELATION_NOT_MANY`として拒否します。

`header-detail`はWebのHeader Form + GridというUI layoutを意味しません。

## `OrderLine`をPatternとは独立して追加した理由

`header-detail`の実証では、新しい業務Semanticが必要でした。

そこでPatternを先に作るのではなく、Issue #117 / PR #119で`sales.OrderLine`と1:N Relationを独立して追加しました。

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

```tei
relation lines to sales.order_line many
```

この段階では次を追加していません。

- `OrderLine` Persistence Mapping
- Database Foreign Key
- Database Contract拡張
- Migration
- UI layout

Semantic Relationと物理DB上の実現方法を分離するためです。

`OrderLine`へ`orderId`属性も先回りして追加していません。現在のSemantic上の所属関係は`Order.lines` Relationを正本とし、物理FKの都合をSemanticへ逆流させない境界を確認しました。

## Kindごとのfieldを共通化しすぎない

5種類の実証から、必要な参照はKindごとにかなり異なることが分かりました。

```text
browse        data + query
record        data
reference     relation + datasource
command       capability
header-detail relation
```

この結果から、次の両極端を避けます。

- すべてのKindへ同じfieldを持たせる
- 任意の`role -> string` Mapで意味を曖昧にする

共通化よりも、Referenceで確認できた意味を型付きで小さく保持することを優先します。

## 正本とAuthoring source

ADR-0018に従い、Patternの正本はApplication配下の`patterns/*.tei`です。

現在のReferenceは次の構成です。

```text
applications/order-entry/
├─ application.tei
├─ patterns/
│  ├─ customer-reference.tei
│  ├─ order-lines.tei
│  ├─ order-record.tei
│  ├─ recent-orders.tei
│  └─ submit-order.tei
├─ queries/
└─ realizations/
```

Pattern identityは明示的なCanonical IDであり、file pathやファイル名から導出しません。

Editor内部状態を第二の正本にはしません。Editorは定義sourceのProjectionと編集支援を担います。

## Loader / Normalization

Workspace Loaderは`applications/*/patterns/*.tei`を読み込み、所属Applicationの`Patterns`へ統合します。

Canonical WorldではPatternをCanonical ID順に正規化します。directory列挙順やOS依存の順序を意味にはしません。

Reference WorldはRepository自身の`tei world`から再生成し、Golden JSONを手作業で編集しません。

## ParserとWorkspace Validation

### Parserの責務

Parserは、そのsourceだけで確定できる条件を確認します。

- Pattern IDが有効なCanonical IDである
- Kindが有効である
- field名を認識できる
- 同じfieldが重複していない
- Kindに必須のfieldがある
- Kindでは意味を持たないfieldが混入していない

### Workspace Validationの責務

Workspace全体を見なければ判断できない条件は共通Validationで確認します。

- Semantic Dataの存在
- Semantic Relationの存在
- Queryの存在とApplication所有関係
- Capabilityの存在
- Datasource aliasの存在
- `reference` Datasourceが読み取り可能であること
- `header-detail` Relationが`many`であること

ParserへWorkspace全体の知識を押し込みません。

## Source IndexとDiagnostic

Patternは`application-pattern:<Canonical ID>` Subjectを使います。

Source IndexからPattern宣言のCanonical ID tokenへ戻れるため、Canonical identityとfile pathを分離できます。

実Referenceではfile renameだけを行い、次を確認しています。

- Canonical Worldは変わらない
- Pattern Subject identityは変わらない
- Source Artifact pathだけが新しいpathへ追随する
- 古いpathは再構築後のSource Indexへ残らない

Pattern Kindごとに別のsource tracking機構は作っていません。

## Application / Semantic / Persistence / Targetの境界

Application Patternの実証から、次の分離を維持できました。

```text
Semantic
  Data / Relation / Capability
          │
          ▼ identityで参照
Application Pattern
  browse / record / reference / command / header-detail
          │
          ├─ Driver
          └─ Projection
                │
                ▼
          Web / Desktop / API / ...
```

Persistence MappingやDatabase Foreign KeyはSemantic RelationやApplication Patternから暗黙生成しません。

Target固有のWidget、画面配置、レスポンシブ指定、HTTP status等もPatternへ持ち込みません。

## Reference / Conformance Evidence

主要な実証は次のPRで閉じました。

| 実証 | Evidence |
| --- | --- |
| Pattern model / Loader / Source Index基盤 | PR #87〜#89 |
| 共通Validation / Diagnostic | PR #95 / #99 / #103 |
| `browse` 実Reference / rename | PR #101〜#103 |
| `record` | PR #105 / #106 |
| `reference` | PR #108 / #109 |
| `command` | PR #114 / #116 |
| `OrderLine` Semantic substrate | PR #119 |
| `header-detail` | PR #121 / #123 |

主要なPRでは`verify`とPostgreSQL integrationが成功しています。

## 完了したこと

Application Pattern Stageでは、次をRepository上のEvidenceから説明できる状態になりました。

- 何を`browse`するか
- 一つの`record`として何を扱うか
- 何をどのlogical datasourceから`reference`するか
- どのCapabilityを`command`として扱うか
- どの1:N Relationを`header-detail`として扱うか
- Patternの正本とCanonical identityをどのように追跡するか
- PatternをSemantic / Persistence / Target固有情報からどのように分離するか

## まだ固定しないこと

- 5種類をTEI全体の永久的な標準一覧とすること
- Pattern Catalogの共有方式
- Patternの継承・合成
- Pattern versioning
- Validation / AuthorizationをPatternが直接参照する方式
- Detailの並び順やページング
- Target-neutral Presentationの最終構造
- Editorでの最終編集方式

これらはDriverと複数TargetからEvidenceを得てから判断します。

## Gate Aへの引き継ぎ

Application Pattern側のStage 1は完了しましたが、Gate Aは未完了です。

次は既存認証Compatibility #68を進めます。Authenticationで必要となるnamed OutcomeとExecution Errorの共通境界はIssue #77 / ADR-0020で先に実装・検証します。

```text
Application Pattern #67  完了
            │
            ├──────────────┐
            │              │
            ▼              ▼
      Outcome #77      Legacy Auth #68
                           │
                           ▼
                        Gate A
                           │
                           ▼
                     Driver #69
```

Gate AではApplication Patternに加えてAuthentication / Authorization、read/write境界、CapabilityのBusiness Result / Execution Errorを一続きに説明できることを確認します。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/design/LEGACY_AUTH_V1_DESIGN.md`
- `plan/design/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
