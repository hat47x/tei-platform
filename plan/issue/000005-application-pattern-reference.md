# ISSUE-000005: `order_entry`でApplication Patternを実証する

- 状態: Done
- GitHub Issue: #67
- 設計・実装基線: [`APPLICATION_PATTERN_V1_DESIGN.md`](../design/APPLICATION_PATTERN_V1_DESIGN.md)

## 目的

v1 Reference Application `order_entry`を使い、Application Patternを画面テンプレートではなく、ApplicationがSemantic上の意味をどのような意図で利用するかを表すTarget非依存の単位として成立させられるかを実証しました。

実証候補としたのは次の5種類です。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

5種類はすべて同じReference Workspace上で成立しました。ただし、これをTEI全体の永久的な標準語彙として確定したわけではありません。Driverや複数Targetから得られるEvidenceによって、分類や所有境界は今後も見直せます。

Patternの所有境界は[ADR-0017](../adr/0017-application-pattern-ownership.md)、authoring sourceとfile pathからのidentity分離は[ADR-0018](../adr/0018-application-pattern-authoring-source.md)に従います。

## 関連するProduct Requirement

- RQ-001 業務上の意味
- RQ-003 暗黙の複数正本を作らない
- RQ-005 Application ↔ Database N:M
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance

## 実証結果

### `browse`

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

QueryがDatasource `sales`を正本として持つため、PatternへDatasourceを重複記述していません。

### `record`

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

「どのSemantic Dataを一件として扱うか」という意味だけで成立しました。取得QueryやDatasourceはPatternへ持ち込んでいません。

### `reference`

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

Relationがsource / target / cardinalityを保持し、PatternはApplication固有のlogical datasourceだけを追加します。Datasourceの`access read`はApplication定義を正本とします。

### `command`

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

Capabilityがinput / outputを保持するため、対象DataやresultをPatternへ重複記述していません。Capability implementationやDatasourceもPatternへ固定していません。

### `header-detail`

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Relationがsource `sales.order`、target `sales.order_line`、cardinality `many`を保持するため、Header Data、Detail Data、cardinalityをPatternへ重複記述していません。

## `header-detail`のSemantic substrate

`header-detail`を実証する前に、Issue #117 / PR #119で業務SemanticをPatternとは独立して追加しました。

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

既存`Order`には次のRelationを追加しています。

```tei
relation lines to sales.order_line many
```

このSemantic追加だけを理由にPersistence Mapping、Database Foreign Key、Database Contract、Migrationは生成していません。

## 成立した設計基線

### PatternはApplicationが所有する意図

PatternはSemanticそのものでも、Web画面などのTarget固有Presentationでもありません。

```text
Semantic
    ↓ identityで参照
Application Pattern
    ↓ Driver / Projection
Target / Runtime
```

### 同じ意味を複数箇所へ書かない

- QueryのDatasourceを`browse`へ重複させない
- Relationのsource / target / cardinalityを`reference`や`header-detail`へ重複させない
- Capabilityのinput / outputを`command`へ重複させない
- Datasource accessをPatternへ複製しない

### Kindごとに必要なfieldを共通化しすぎない

実証した最小形は次です。

```text
browse        data + query
record        data
reference     relation + datasource
command       capability
header-detail relation
```

一つの大きな汎用Patternへ全fieldを詰め込まず、不要なfieldは各Kindの契約として明示的に拒否します。

### ParserとWorkspace Validationを分ける

Parserは構文とKindごとの必須・禁止fieldを確認します。

参照先の存在、Datasource access、Capabilityの存在、`header-detail` Relationが`many`であることなど、Workspace全体を見なければ判断できない条件は共通Validationで確認します。

### Source / identityをfile pathから分離する

Patternは明示的なCanonical IDを持ち、`application-pattern:<Canonical ID>` Subjectから既存Source Indexを通じてauthoring sourceへ戻れます。

file pathやEditor内部状態を第二のCanonical identityにはしません。

### Semantic RelationとDatabase Foreign Keyを分離する

Semantic Relationが存在しても、Persistence MappingやFKを暗黙生成しません。Application Patternを成立させるために物理DB構造を先取りする必要もありませんでした。

### PatternをUI layoutへ固定しない

`header-detail`はWebのHeader Form + Gridを意味しません。5 Patternはいずれも、Targetが変わっても残る可能性のあるApplication上の意図として扱います。

## Reference / Conformance

一連の実証で次を確認しました。

- 5 Patternが同じ`order_entry`で共存する
- PatternをCanonical ID順に正規化できる
- Patternから既存Semantic / Query / Relation / Capabilityを追跡できる
- Pattern Diagnosticからauthoring sourceへ戻れる
- file renameだけではCanonical identityを変えない
- Reference WorldをRepository自身の`tei world`から再生成できる
- Golden差分へfile pathや生成都合を混ぜない
- Pattern追加だけでは不要なPersistence / Contractを暗黙生成しない

主要な実証PRのCIは`verify` / PostgreSQL integrationとも成功しています。

- `browse`: PR #101〜#103
- `record`: PR #105 / #106
- `reference`: PR #108 / #109
- `command`: PR #114 / #116
- `OrderLine` Semantic: PR #119
- `header-detail`: PR #121 / #123

## 完了条件

1. 5 Patternを`order_entry`で一続きに表現できる — 完了
2. Application上の意図とSemanticの境界を説明できる — 完了
3. Application PatternとPersistence / Realizationの境界を説明できる — 完了
4. Application PatternとProjection / Target固有情報の境界を説明できる — 完了
5. Patternの正本とCanonical identityを追跡できる — 完了
6. Source Index / Diagnosticからauthoring sourceへ戻れる — 完了
7. Reference WorldをRepository内の正本から再生成できる — 完了
8. 後続Driverへ渡せるTarget非依存情報を揃える — Application Pattern側は完了

## Gate Aへの引き継ぎ

このIssueの完了だけではGate Aを完了としません。

Application Pattern側のEvidenceは揃いました。次は既存認証Compatibility [`ISSUE-000003`](000003-legacy-authentication-compatibility-reference.md) / GitHub Issue #68を進めます。

#68では、共通基盤であるCapability Outcome / Execution Error #77を先に閉じ、Authentication / Authorizationを`order_entry`へ接続します。そのEvidenceとこのIssueの成果を合わせてGate Aを監査します。

## 関連文書

- `plan/design/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/design/LEGACY_AUTH_V1_DESIGN.md`
- `plan/design/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
