# ISSUE-000011: `sales.OrderLine`とOrder 1:N RelationをSemantic Referenceへ追加する

- 状態: Done
- GitHub Issue: #117（完了）
- 実装PR: #119
- 上位Issue: `ISSUE-000005` / GitHub Issue #67

## 目的

`header-detail` Application Patternを実証する前提として、受注明細側の業務SemanticをPatternや物理DB設計から独立して成立させる作業です。

既存Reference Workspaceには`sales.order`はありましたが、明細側のDataと1:N Relationがありませんでした。

Patternの都合から物理tableやForeign Keyを先に設計するのではなく、まず業務上の関係そのものをSemantic Referenceとして追加しました。

## 追加したSemantic

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

Canonical Worldでは、`sales.order.lines`が次を保持します。

- source: `sales.order`
- target: `sales.order_line`
- cardinality: `many`

## なぜPatternと分けたか

`header-detail` Patternと`OrderLine`を同時に追加すると、次の二つを切り分けにくくなります。

1. 業務Semanticそのものが不足している問題
2. Application Patternとしてどのfieldが必要かという問題

そこで先にSemanticだけを成立させ、後から成立済みRelationをPatternが参照する順序にしました。

この分離によって、Application上の都合からSemantic modelを逆算することを避けています。

## `orderId`を先に追加しなかった理由

`OrderLine`に`orderId`属性は追加していません。

現時点で`Order`への所属関係の正本はSemantic Relation `sales.order.lines`です。

物理DBでこの関係をどの列、Foreign Key、indexで実現するかはPersistence側の設計です。物理実現を先に想定し、それをSemanticへ逆流させる必要性はこの実証から確認されませんでした。

## この作業で追加しなかったもの

- `OrderLine` Persistence Mapping
- `order_lines`物理table
- Database Foreign Key
- Migration
- Database Contract拡張
- Application Pattern
- Driver / UI

Semantic Relationと物理DB上の実現方法は別の意味単位として維持します。

## Conformance

PR #119で次を確認しました。

- `sales.order_line`がSemantic Dataとして読み込まれる
- 5属性が既存`std.*`型で表現できる
- `sales.order.lines`のsource / target / `many`を追跡できる
- 既存`sales.order.customer` Relationが変わらない
- 既存`sales.order.submit` Capabilityのinput / outputが変わらない
- `OrderLine` Persistence Mappingが暗黙生成されない
- Database Contractへ`OrderLine`が暗黙公開されない
- Reference WorldをRepository自身から再生成できる

CI #277は`verify` / PostgreSQL integrationとも成功しました。

## 完了条件の確認

1. `sales.OrderLine`をSemantic Dataとして表現できる — 完了
2. `Order 1:N OrderLine`をSemantic Relationとして表現できる — 完了
3. Semantic RelationとPersistence / FKを分離できる — 完了
4. 既存Semantic / Capabilityを壊さない — 完了
5. Reference WorldをRepository自身から再生成できる — 完了
6. 後続`header-detail`実証へRelation identityを渡せる — 完了

## 関連

- GitHub Issue #67 / #117 / #118
- GitHub PR #119
- `ISSUE-000005`
- `ISSUE-000012`
- ADR-0017 / ADR-0018
