# Application Pattern v1 設計・実装基線

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で進めるApplication Patternについて、v1で守る設計境界、Repository上で確認済みのEvidence、現在の実証課題を一つの基線として整理するものです。

Application Patternは、画面テンプレートの分類を先に決めるための仕組みではありません。ApplicationがSemantic上の意味をどのような意図で利用するかを、WebなどのTarget固有表現やPersistence上の実現方法から切り離して保持するための層です。

v1では`order_entry`を実際のReference Applicationとして使い、具体的な業務シナリオから必要なPattern Kindとfieldを一つずつ抽出します。`browse`、`record`、`reference`、`command`、`header-detail`という5種類は実証候補であり、TEI全体の永久的な標準語彙として先に固定しません。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

## 現在地

Stage 1は進行中です。

`browse`、`record`、`reference`、`command`の4種類は、Canonical model、Parser、Loader、Normalization、Validation、Source Index、実Reference、Golden Worldまで一続きに成立しています。

最後の候補である`header-detail`の前提となる業務Semanticも、`ISSUE-000011` / GitHub Issue #117 / PR #119で成立しました。

```text
sales.order
    1
    │
    └── many sales.order_line
```

現在は`ISSUE-000012` / GitHub Issue #118で、成立済みRelation `sales.order.lines`をApplication上のHeader / Detail意図として利用する最小契約を実証しています。

現時点でmainに有効なPattern Kindは次の4種類です。

- `browse`
- `record`
- `reference`
- `command`

`header-detail`は実装・検証中です。

## これまでに成立した共通基盤

### Canonical model

PR #87で`application.Pattern`と`Application.Patterns`を追加し、PatternをApplicationが所有する子要素として扱う構造を導入しました。

現在のPatternは、実Referenceから必要性を確認できた参照だけを型付きで保持します。

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

すべてのfieldをすべてのKindで使うわけではありません。Kindごとの意味に不要なfieldは暗黙に無視せず、契約として受理しません。

Semantic定義、Query本文、Capability契約、Database Contract本文、Persistence Mapping、file path、Target固有PresentationはPatternへ複製しません。

### LoaderとNormalization

PR #88で次を実装しました。

- `applications/*/patterns/*.tei`をWorkspace Loaderへ接続
- 読み込んだPatternを所属Applicationへ統合
- PatternをCanonical ID順に正規化
- `application-pattern:<Canonical ID>`のsource mappingを登録

ファイル名やdirectoryの列挙順をCanonicalな意味にはしません。

### Source Index

PR #89で、PatternのCanonical IDから宣言中のID tokenへ戻るSource Index経路を実装しました。

Patternのidentityはfile pathから独立しています。PR #102では実ReferenceのPatternファイルを改名し、次を確認しました。

- Canonical Worldは変わらない
- Pattern Subject identityは変わらない
- Source Artifact pathだけが新しいファイル名へ追随する
- 古いpathは再構築後のSource Indexへ残らない

Canonical modelへline / columnなどのsource locatorは埋め込みません。

### 共通Validation

PR #95、#99、#103以降、Pattern固有の参照整合性は共通`validation.Validate`から一度だけ実行されます。

Parserは構文だけで確定できる条件を検査し、Workspace全体を見なければ判断できない参照整合性はValidation層で扱います。

PatternのDiagnosticは既存の`application-pattern:<Canonical ID>` Subjectを利用し、Source Indexを通じて元のauthoring sourceへ戻れます。

## Pattern Kindごとの実証結果

### `browse`

実Referenceは次です。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

現時点の契約は次です。

- `data`: 必須
- `query`: 必須
- `datasource`: 任意
- `relation`: 受理しない
- `capability`: 受理しない

Query `order_entry.recent_orders`がlogical datasource `sales`を正本として保持しているため、実ReferenceではPattern側に同じDatasourceを重複記述していません。

PR #101で実ReferenceとGolden Worldを追加し、PR #102で実Referenceのfile rename invarianceを確認しました。

`browse`から得た重要な判断は、**既存Queryがすでに持つ意味をPatternへ二重記述しない**ことです。

### `record`

Issue #104、PR #105 / #106で実証しました。

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

現時点の契約は次です。

- `data`: 必須
- `query`: 受理しない
- `datasource`: 受理しない
- `relation`: 受理しない
- `capability`: 受理しない

`record`は、「ApplicationがどのSemantic Dataを一件として扱うか」という意図だけで成立しました。

既存Database Contractに`find_by_id`があることや、将来Driverが一件取得を行うことは、Patternへ取得経路を固定する理由にはなりませんでした。

同様に、`sales.order`とlogical datasource `sales`の対応もPatternへ重複記述していません。

### `reference`

Issue #107、PR #108 / #109で実証しました。

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

現時点の契約は次です。

- `relation`: 必須
- `datasource`: 必須
- `data`: 受理しない
- `query`: 受理しない
- `capability`: 受理しない

Semantic Relation `sales.order.customer`は、source `sales.order`とtarget `customer.customer`をすでに保持しています。そのため、sourceやtargetをPatternへ重複記述しません。

一方、logical datasource `master`はApplication固有の参照経路なので、Patternが保持する意味として残しています。

`reference`は読み取り意図です。そのため、選択Datasourceが`write`専用である場合は`APPLICATION_PATTERN_DATASOURCE_NOT_READABLE`として拒否します。`read-write`は読み取り可能なので受理します。

access modeそのものはPatternへ複製せず、Applicationの`DatabaseRequirement`を正本とします。

実Referenceでは、次の接続もConformanceで確認しています。

```text
sales.order.customer
        │ target
        ▼
customer.customer
        ▲
        │ read
master_data.customer v2.4
        ▲
        │ requires ^2
order_entry datasource master
```

「特定のreference Patternが選んだDatasourceのRequired Contractが、そのRelation targetをread対象として公開している」という横断条件は、現時点ではReference Conformanceで観測しています。一例だけを理由にPattern専用の重複Contract体系は追加していません。

### `command`

Issue #110、PR #114 / #116で実証しました。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

現時点の契約は次です。

- `capability`: 必須
- `data`: 受理しない
- `relation`: 受理しない
- `query`: 受理しない
- `datasource`: 受理しない

Capability `sales.order.submit`はSemantic側で次の契約を持ちます。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

そのため、対象Data、input、outputをPatternへ再記述していません。

DatasourceやCustom Go等の実装方式もPatternへ持ち込まず、Capability Binding / Realization / Driver側の責務として分離します。

Capabilityの既存`output sales.order`はADR-0020に従うBusiness Resultとして維持しました。`command`追加だけを理由にnamed Outcomeへ変更せず、Execution ErrorやObservationもPatternへ混在させていません。

PR #116では4種類のPattern共存、Source Index、Golden Worldの再生成まで確認しています。

## `OrderLine` Semantic Referenceで確認した境界

`header-detail`だけは、既存Reference Workspaceに明細側の業務Semanticがなかったため、Patternと新しいSemanticを同時に追加しませんでした。

`ISSUE-000011` / GitHub Issue #117 / PR #119で、先に次のSemanticを独立して成立させています。

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

既存`Order`には次を追加しました。

```tei
relation lines to sales.order_line many
```

この実証では、Semantic上の関係だけを追加し、次を変更していません。

- `OrderLine` Persistence Mapping
- Database Foreign Key
- `sales_core.order` Database Contract
- Migration
- Application Pattern

`OrderLine`へ`orderId`属性も先回りして追加していません。`Order`への所属関係はまずSemantic Relationを正本とし、物理DBでどの列・Foreign Keyを使うかはPersistence側のEvidenceへ残しました。

PR #119のCI #277は`verify` / PostgreSQL integrationとも成功しています。

## 現在の実証 — `header-detail`

`ISSUE-000012` / GitHub Issue #118では、成立済みRelation `sales.order.lines`をApplication上のHeader / Detail意図として扱えるかを検証しています。

第一候補はRelation-onlyです。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Relationがすでに次を保持しています。

- source: `sales.order`
- target: `sales.order_line`
- cardinality: `many`

そのため、Header Data、Detail Data、cardinalityをPatternへ重複記述しません。

### `many`はValidationで確認する

`header-detail`は一つのHeaderと複数Detailを扱う意図として実証します。

RelationのcardinalityはPatternへコピーしませんが、参照するRelationが`many`であることはWorkspace Validationから確認します。

ParserはRelationの実体を知らないため、`relation` fieldの必須性までを扱います。`one` Relationを指定した場合は、Workspace全体を見られるValidation層から`APPLICATION_PATTERN_RELATION_NOT_MANY`を返す方針です。

### Datasource / Query / Validationを先に追加しない

`header-detail`がRelationだけでApplication上の意図として成立するなら、取得・保存経路をPatternへ固定する必要はありません。

Datasource、Query、Validation identityは、後続Driverで必要性が確認されるまで追加しません。

### Persistence / FKを前提にしない

Semantic Relation、Application Pattern、Persistence Mapping、Database Foreign Keyは別の意味単位です。

`header-detail`が成立したという理由だけで、`order_lines` table、Persistence Mapping、FK、Migration、Database Contract拡張を自動生成しません。

## 現在のAuthoring構文

### `browse`

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

### `record`

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

### `reference`

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

### `command`

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

### `header-detail` — 実証中

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

表層構文そのものより、次の性質を優先します。

- Pattern IDを明示する
- Pattern Kindを明示する
- 既存資産を型付きidentityで参照する
- 同じ意味を複数箇所へ手書きしない
- file pathをidentityにしない
- Persistenceの実現方法をApplication上の意味へ持ち込まない
- Target固有UIをApplication上の意味へ持ち込まない

## 正本の配置

ADR-0018に従い、Patternの正本はApplication配下の`patterns/*.tei`です。

現在のReferenceは概ね次の構成です。

```text
applications/order-entry/
├─ application.tei
├─ patterns/
│  ├─ recent-orders.tei
│  ├─ order-record.tei
│  ├─ customer-reference.tei
│  └─ submit-order.tei
├─ queries/
└─ realizations/
```

`header-detail`の実Referenceが成立した後は`order-lines.tei`が加わる見込みですが、実装前のファイル名をCanonical identityにはしません。

v1では一ファイル一Patternから始めています。

Editor内部状態を第二の正本にはしません。Editorは当面、定義sourceのProjectionと編集支援を担います。

## Kindごとのfieldは共通化しすぎない

4種類の実証から、Patternに共通するのは主にidentityとKindであり、必要な参照はKindごとに異なることが確認できました。

そのため、次のどちらにも寄せません。

- すべてのPatternに同じfieldを必須化する
- 任意の`role -> string` Mapで意味を曖昧にする

必要なfieldだけを型付きで追加し、不要なfieldは明示的に拒否します。

`header-detail`でも新しい専用fieldを先に増やさず、既存の型付きRelation参照だけで成立するかを最初に確認します。

## Validationの責務

### Parserで確認すること

- Pattern IDが有効なCanonical IDである
- Kindが現在有効な値である
- field名がそのKindで扱える
- 同じfieldが重複していない
- Kindに必要なfieldが欠けていない
- そのKindでは意味を持たないfieldが混入していない

### Workspace Validationで確認すること

- 参照するSemantic Dataが存在する
- 参照するSemantic Relationが存在する
- Queryが存在し、対象Applicationとの所有関係が整合する
- Capabilityが存在する
- Datasource aliasがApplicationに存在する
- `reference`のDatasourceが読み取り可能である
- `header-detail`のRelationが`many`である

ParserへWorkspace全体の知識を押し込みません。

## Source IndexとDiagnostic

PatternのDiagnosticは`application-pattern:<Canonical ID>` Subjectを使用します。

Source Indexから元のPattern定義へ戻れるため、Canonical IDを保ったままファイル名や配置を変更できます。

後続Kindでも、Pattern専用の別系統のsource trackingを増やさず、既存のSource Index / Subject契約を再利用します。

## Normalizationと再現性

PatternはCanonical ID順に正規化します。

Reference WorldはRepository自身の`tei world`から再生成し、手作業でGolden JSONを編集しません。

`browse`、`record`、`reference`、`command`の各実Reference追加では、Golden差分が追加したPatternのCanonicalな意味だけに限定されることを確認しています。

`OrderLine` Semantic Referenceでも、Golden差分はSemantic追加に対応する純追加だけで、Persistence / Contract / Migrationへ差分を広げていません。

## #67の実装単位と現在の状態

| 実装単位 | 状態 | Evidence / 次の作業 |
| --- | --- | --- |
| Patternの最小Canonical model | 完了 | PR #87 |
| `patterns/*.tei` Loader / Normalization | 完了 | PR #88 |
| Source Index / identityとfile pathの分離 | 完了 | PR #89、#102 |
| 共通Validation / Diagnostic projection | 完了 | PR #95、#99、#103 |
| `browse`実Reference | 完了 | PR #101、#102 |
| `record`実証 | 完了 | Issue #104、PR #105、#106 |
| `reference`実証 | 完了 | Issue #107、PR #108、#109 |
| `command`実証 | 完了 | Issue #110、PR #114、#116 |
| `OrderLine` Semantic Reference | 完了 | Issue #117、PR #119 |
| `header-detail`最小契約 | 進行中 | Issue #118 |
| `header-detail`実Reference | 未完了 | 最小契約成立後に追加 |
| 5 Pattern Evidence監査 | 未完了 | `header-detail`完了後 |
| Gate Aへ渡すEvidence監査 | 未完了 | #68の既存認証Compatibility等と合流して評価 |

## まだ固定しないこと

実装Evidenceが増えるまで、次は固定しません。

- 5 Patternを最終的な標準分類とすること
- Pattern Catalogの共有方式
- Patternの継承・合成
- Pattern versioning
- UI field grouping
- Validation / AuthorizationをPatternの直接参照にするか
- `reference`とDatabase Contractの横断整合を共通Coreへ昇格する最終条件
- `header-detail`にDatasource / Queryが必要か
- Editorでの最終編集方式
- Aggregate / Transaction一般モデル

## Gate Aへ渡す条件

#67だけでGate Aを完了とはしません。

Application Pattern側では、少なくとも次をRepository上のEvidenceから説明できる状態にします。

- 何を`browse`するか
- 一つの`record`として何を扱うか
- どのRelationを、どのDatasourceから`reference`するか
- どのCapabilityを`command`として呼ぶか
- HeaderとDetailをどのRelationで一まとまりとして扱うか
- どのValidationを実行するか
- どのDatasourceがread-onlyか
- Patternの正本とCanonical identityをどのように追跡するか

Authorizationは`ISSUE-000003`の既存認証Compatibilityから合流させます。

CapabilityのBusiness Result / Execution Error境界はADR-0020 / `ISSUE-000009`から受け取ります。

## 完了条件

#67の完了条件は、`order_entry`の業務シナリオで候補となる5 Patternを一続きに表現し、Application上の意図とSemantic、Realization / Persistence、Projection / Target固有情報の境界を説明できることです。

さらに、Patternの正本、Canonical identity、Source Index、Reference Worldの再現性をRepository内のReference / Conformanceから確認でき、その同じApplication上の意味を後続Driverが利用できる必要があります。

現在は4種類のPatternと、`header-detail`の前提Semanticまで実証済みです。Stage 1、Gate A、Milestone 1の完了はまだ宣言しません。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000010-command-application-pattern.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000012-header-detail-application-pattern.md`
