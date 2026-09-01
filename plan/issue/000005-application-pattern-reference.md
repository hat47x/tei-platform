# ISSUE-000005: `order_entry`でApplication Patternを実証する

- 状態: Done
- GitHub Issue: #67（完了）
- 設計・実装基線: [`APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)
- Pattern所有境界: [ADR-0017](../adr/0017-application-pattern-ownership.md)
- Authoring source / identity: [ADR-0018](../adr/0018-application-pattern-authoring-source.md)

## 目的

Application Patternを画面テンプレートとして先に固定せず、ApplicationがSemanticや既存業務資産をどのような意図で利用するかを、Driverや複数Targetから再利用できる単位として成立させられるかを`order_entry`で実証する作業です。

v1では次の5種類を候補として一つずつ実Referenceから検証しました。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

5種類すべてが実Referenceまで成立したため、このIssueは完了です。ただし、5分類をTEI全体の永久的な標準語彙として確定したわけではありません。

## 実Reference

### `browse`

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

既存Queryがlogical datasourceを保持しているため、実ReferenceではDatasourceをPatternへ重複記述していません。

### `record`

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

「どのSemantic Dataを一件として扱うか」という意味だけで成立しました。取得QueryやDatasourceはPatternへ固定していません。

### `reference`

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

Relationのsource / target / cardinalityはSemantic側を正本とし、Application固有の参照経路であるlogical datasourceだけをPatternに保持します。

### `command`

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

Capabilityのinput / output、implementation、DatasourceはPatternへ複製していません。

### `header-detail`

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Relationのsource `sales.order`、target `sales.order_line`、cardinality `many`をSemantic側の正本から利用します。Header Data、Detail Data、cardinalityをPatternへ重複記述していません。

## `header-detail`の前提Semantic

`header-detail`だけは、既存Referenceに明細側の業務Semanticがなかったため、Patternとは別の作業として先に追加しました。

`ISSUE-000011` / GitHub Issue #117 / PR #119で次を追加しています。

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

この段階ではPersistence Mapping、Database Foreign Key、Database Contract、Migrationを追加していません。

新しいSemanticの不足とApplication Patternの設計を別の実証単位に分けたことで、Patternの都合から物理DB構造を逆算することを避けられました。

## 実証した共通基盤

### Canonical model

PatternはApplicationが所有する子要素として保持します。

現行modelはID、Kind、Data、Relation、Query、Capability、Datasourceを型付き参照として持ちますが、Kindごとに必要なfieldだけを受理します。

### Loader / Normalization

- `applications/*/patterns/*.tei`を読み込む
- 所属Applicationへ統合する
- Canonical ID順に正規化する
- `application-pattern:<Canonical ID>`のsource mappingを作る

### Source Index

Pattern identityはfile pathから独立したCanonical IDです。

Source Indexからauthoring sourceへ戻れます。file nameを変更してもCanonical WorldとPattern identityが変わらないことを実Referenceで確認済みです。

### Parser / Workspace Validation

Parserは構文とKind固有の必須・禁止fieldを扱います。

Workspace全体が必要な参照整合性は共通`validation.Validate`で扱います。

- Semantic assetの存在
- Query ownership
- Datasourceの存在・access
- Capabilityの存在
- `header-detail` Relationの`many` cardinality

同じ原因のDiagnosticをPattern専用入口と共通入口から重複して返さない構造を維持します。

## 実証から得た設計基線

### PatternはApplication上の意図

Semanticそのものでも、Persistenceでも、Target固有UIでもありません。

### 既存資産の意味を複製しない

Query、Relation、Capability、Datasourceがすでに持つ意味をPatternへ書き写さず、Canonical IDで接続します。

### Kindごとに必要なfieldだけを持つ

実Referenceで成立した最小形は次です。

```text
browse        data + query
record        data
reference     relation + datasource
command       capability
header-detail relation
```

### 実現方法をPatternへ持ち込まない

native SQL、Capability implementation、Persistence Mapping、Foreign Key、Migration、Web Widgetはそれぞれの責務境界へ残します。

### Semantic RelationとDatabase Foreign Keyを同一視しない

`Order 1:N OrderLine`のSemantic Relationが存在しても、Persistence MappingやFKを暗黙生成しません。

### PatternをUI layoutへ固定しない

`header-detail`をWebのHeader Form + Gridと定義しません。ほかのKindも同様に、Targetが変わっても残る可能性があるApplication上の意図として扱います。

## Evidence

主要な実証は次です。

- `browse`: PR #101〜#103
- `record`: PR #105 / #106
- `reference`: PR #108 / #109
- `command`: PR #114 / #116
- `OrderLine` Semantic: PR #119 / CI #277
- `header-detail` Canonical契約: PR #121 / CI #279
- `header-detail`実Reference: PR #123 / CI #282

主要PRは`verify` / PostgreSQL integrationで成功しています。

Reference WorldはRepository自身の`tei world`から再生成できます。

## 完了条件の確認

1. 5種類のPatternを一つの`order_entry`で表現できる — 完了
2. Application上の意図とSemanticを分離できる — 完了
3. PatternとPersistence / Realizationを分離できる — 完了
4. PatternとProjection / Target固有情報を分離できる — 完了
5. Patternの正本とCanonical identityを追跡できる — 完了
6. Source Index / Diagnosticからauthoring sourceへ戻れる — 完了
7. Reference WorldをRepository内の正本から再生成できる — 完了
8. 後続DriverへTarget非依存のPattern情報を渡せる — Application Pattern側は完了

## Gate Aとの関係

このIssueが完了してもGate A全体は完了しません。

Gate Aでは、Application PatternのEvidenceに加えて次を合流させます。

- Validation
- Authentication / Authorization
- read / read-write Datasource境界
- Capability Binding
- Business Result / Execution Error境界

既存認証Compatibilityは`ISSUE-000003` / GitHub Issue #68から受け取ります。

Capability結果境界はADR-0020 / `ISSUE-000009`を利用します。

ValidationをPattern自身のfieldとして持つ必要性は今回の5 Patternから確認されなかったため、後続の具体的な業務ruleからApplication上の構成を決めます。

Gate AをEvidenceで通過した後、`ISSUE-000006` / GitHub Issue #69のApplication Driverへ進みます。

## 関連

- [`../APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)
- [`../V1_REFERENCE_APPLICATION_PLAN.md`](../V1_REFERENCE_APPLICATION_PLAN.md)
- `ISSUE-000003`
- `ISSUE-000006`
- `ISSUE-000009`
- `ISSUE-000010`
- `ISSUE-000011`
- `ISSUE-000012`
- GitHub Issue #67 / #68 / #69 / #117 / #118
