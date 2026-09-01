# Application Pattern v1 設計・実装基線

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で進めるApplication Patternについて、v1で守る設計境界、Repository上で確認済みのEvidence、次に実証する内容を一つの基線として整理するものです。

Application Patternは、画面テンプレートの分類を先に決めるための仕組みではありません。ApplicationがSemantic上の意味をどのような意図で利用するかを、WebなどのTarget固有表現から切り離して保持するための層です。

v1では`order_entry`を実際のReference Applicationとして使い、具体的な業務シナリオから必要なPattern Kindとfieldを一つずつ抽出します。`browse`、`record`、`reference`、`command`、`header-detail`という5種類は実証候補であり、TEI全体の永久的な標準語彙として先に固定しません。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

## 現在地

Stage 1は進行中です。

`browse`、`record`、`reference`の3種類については、Canonical model、Parser、Loader、Normalization、Validation、Source Index、実Reference、Golden Worldまで一続きに成立しています。

次に実証するのは`command`です。`header-detail`は、新しい`sales.OrderLine`と対応RelationをSemantic側へ追加した後に扱います。

現時点で有効なPattern Kindは次の3種類です。

- `browse`
- `record`
- `reference`

`command`と`header-detail`はまだ有効化していません。

## これまでに成立した共通基盤

### Canonical model

PR #87で`application.Pattern`と`Application.Patterns`を追加し、PatternをApplicationが所有する子要素として扱う構造を導入しました。

現在のPatternは、実Referenceから必要性を確認できた参照だけを持ちます。

```text
Pattern
- ID
- Kind
- Data
- Relation
- Query
- Datasource
```

すべてのfieldをすべてのKindで使うわけではありません。Kindごとの意味に不要なfieldは受理しません。

Semantic定義、Query本文、Database Contract本文、file path、Target固有PresentationはPatternへ複製しません。

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

Query `order_entry.recent_orders`がlogical datasource `sales`を正本として保持しているため、実ReferenceではPattern側に同じDatasourceを重複記述していません。

PR #101で実ReferenceとGolden Worldを追加し、PR #102で実Referenceのfile rename invarianceを確認しました。

`browse`から得た重要な判断は、**既存のQueryがすでに持つ意味をPatternへ二重記述しない**ことです。

### `record`

Issue #104、PR #105 / #106で実証しました。

実Referenceは次です。

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

`record`は、「ApplicationがどのSemantic Dataを一件として扱うか」という意図だけで成立しました。

既存Database Contractに`find_by_id`があることや、将来Driverが一件取得を行うことは、Patternへ取得経路を固定する理由にはなりませんでした。

同様に、`sales.order`とlogical datasource `sales`の対応もPatternへ重複記述していません。

### `reference`

Issue #107、PR #108 / #109で実証しました。

実Referenceは次です。

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

Semantic Relation `sales.order.customer`は、所属Data `sales.order`とtarget `customer.customer`をすでに保持しています。そのため、sourceやtargetをPatternへ重複記述しません。

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

既存共通Validationは、Semantic Relation target、Database ContractのSemantic参照、Required Contractの存在、提供Database、compatible majorをそれぞれ検査します。

「特定のreference Patternが選んだDatasourceのRequired Contractが、そのRelation targetをread対象として公開している」という横断条件は、現時点ではReference Conformanceで観測しています。一例だけを理由にPattern専用の重複Contract体系は追加していません。

## 現在のAuthoring構文

### browse

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

### record

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

### reference

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

表層構文そのものより、次の性質を優先します。

- Pattern IDを明示する
- Pattern Kindを明示する
- 既存資産を型付きidentityで参照する
- 同じ意味を複数箇所へ手書きしない
- file pathをidentityにしない
- Target固有UIをApplication上の意味へ持ち込まない

## 正本の配置

ADR-0018に従い、Patternの正本はApplication配下の`patterns/*.tei`です。

現在のReferenceでは概ね次の構成になっています。

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

v1では一ファイル一Patternから始めています。

Editor内部状態を第二の正本にはしません。Editorは当面、定義sourceのProjectionと編集支援を担います。

## Kindごとのfieldは共通化しすぎない

`browse`、`record`、`reference`の実証から、Patternに共通するのはidentityとKind程度であり、必要な参照はKindごとにかなり異なることが確認できました。

そのため、次のどちらにも寄せません。

- すべてのPatternに同じfieldを必須化する
- 任意の`role -> string` Mapで意味を曖昧にする

必要なfieldだけを型付きで追加し、不要なfieldは明示的に拒否します。

## 次の実証 — `command`

次は既存Semantic Capability `sales.order.submit`へ接続する`command`を実証します。

第一候補は、Capability identityだけを保持する最小形です。

```text
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

ただし、この構文やfield名は実装前の候補です。実Referenceを成立させるために他の意味が必要かを先に確認します。

### 最初に確認する問い

- `command`の意味はCapability identityだけで十分か
- 対象Data `sales.order`をPatternへ重複記述する必要があるか
- DatasourceをPatternへ持たせる必要があるか
- Capabilityの既存`output sales.order`をそのまま利用できるか
- named Outcomeを新設する必要が本当にあるか
- AuthorizationやValidationとの順序関係をPattern自身へ持たせるべきか、それともDriverのScenarioで構成すべきか

Capability結果の共通境界は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に従います。

`command`を追加するという理由だけで、現行`sales.order.submit`の単一`output`をnamed Outcomeへ書き換えません。

## `header-detail`と`OrderLine`

`header-detail`は、新しい`sales.OrderLine`と対応RelationをSemantic側へ追加した後に実証します。

概念上は次の関係です。

```text
Order 1 ── N OrderLine
```

Semantic Relation、Application上の`header-detail`、Database Foreign Keyは別の意味単位です。

Relationが存在するという理由だけで、FK、画面構造、編集操作を暗黙生成しません。

`OrderLine`という新しい業務Semanticの追加とPattern基盤の変更を分け、どの境界から問題が生じたかを追跡できるようにします。

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
- Datasource aliasがApplicationに存在する
- `reference`のDatasourceが読み取り可能である
- 後続`command`ではCapability参照が実在する

ParserへWorkspace全体の知識を押し込みません。

## Source IndexとDiagnostic

PatternのDiagnosticは`application-pattern:<Canonical ID>` Subjectを使用します。

Source Indexから元のPattern定義へ戻れるため、Canonical IDを保ったままファイル名や配置を変更できます。

今後のKindでも、Pattern専用の別系統のsource trackingを増やさず、既存のSource Index / Subject契約を再利用します。

## Normalizationと再現性

PatternはCanonical ID順に正規化します。

Reference WorldはRepository自身の`tei world`から再生成し、手作業でGolden JSONを編集しません。

`browse`、`record`、`reference`の各実Reference追加では、Golden差分が追加したPatternのCanonicalな意味だけに限定されることを確認しています。

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
| `command`実証 | 次 | `sales.order.submit`を利用する |
| `header-detail`実証 | 未着手 | `OrderLine`とRelation追加後 |
| Gate Aへ渡すEvidence監査 | 未完了 | #68の既存認証Compatibilityと合流して評価 |

## まだ固定しないこと

実装Evidenceが増えるまで、次は固定しません。

- 5 Patternを最終的な標準分類とすること
- `command` / `header-detail`の最終DSL
- Pattern Catalogの共有方式
- Patternの継承・合成
- Pattern versioning
- UI field grouping
- Validation / AuthorizationをPatternの直接参照にするか
- referenceとDatabase Contractの横断整合を共通Coreへ昇格する最終条件
- Editorでの最終編集方式

## Gate Aへ渡す条件

#67だけでGate Aを完了とはしません。

Application Pattern側では、少なくとも次をRepository上のEvidenceから説明できる状態にします。

- 何を`browse`するか
- 一つの`record`として何を扱うか
- どのRelationを、どのDatasourceから`reference`するか
- HeaderとDetailをどのようなApplication上の単位として扱うか
- どのCapabilityを`command`として呼ぶか
- どのValidationを実行するか
- どのDatasourceがread-onlyか
- Patternの正本とCanonical identityをどのように追跡するか

Authorizationは`ISSUE-000003`の既存認証Compatibilityから合流させます。

## 完了条件

#67の完了条件は、`order_entry`の業務シナリオで候補となる5 Patternを一続きに表現し、Application上の意図とProjection / Target固有情報の境界を説明できることです。

さらに、Patternの正本、Canonical identity、Source Index、Reference Worldの再現性をRepository内のReference / Conformanceから確認でき、その同じApplication上の意味を後続Driverが利用できる必要があります。

現在は`browse`、`record`、`reference`まで実証済みです。Stage 1、Gate A、Milestone 1の完了はまだ宣言しません。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
