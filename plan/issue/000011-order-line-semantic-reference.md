# ISSUE-000011: `sales.OrderLine`とOrder 1:N RelationをSemantic Referenceへ追加する

- 状態: In Progress
- GitHub Issue: #117
- 実装PR: #119
- 上位Issue: `ISSUE-000005` / GitHub Issue #67
- 後続Issue: `ISSUE-000012` / GitHub Issue #118

## 目的

`header-detail` Application Patternを実証する前に、受注明細そのものの業務上の意味をSemantic Referenceとして成立させます。

Patternの都合からData構造を逆算するのではなく、まず`OrderLine`というDataと、`Order`が複数の`OrderLine`を持つ関係をSemantic上の正本として表します。

この段階では、Persistence Mapping、Database Foreign Key、Database Contract、画面構造を同時に追加しません。Semantic上の関係と、その関係を物理DBやApplicationでどのように利用するかを分離したまま検証します。

## 現在の実証候補

PR #119では、次のSemantic Referenceを実装しています。

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

既存`Order`には次のRelationを追加します。

```tei
relation lines to sales.order_line many
```

このRelationにより、Canonical Worldから次の意味を追跡できる状態を目指します。

```text
sales.order
    1
    │
    └── many sales.order_line
```

## `orderId`を先に追加しない理由

この段階では、`OrderLine`へ`orderId`属性を追加しません。

`OrderLine`が`Order`に属するという業務上の関係は、まず`Order.lines` Relationを正本とします。物理DBでその関係をどの列やForeign Keyによって実現するかはPersistence側の設計です。

`orderId`を早い段階で必須属性として置くと、Semantic上の意味と物理的な関連付け方法を暗黙に同一視するおそれがあります。

後続Driverで保存・取得を実証するときに、Semantic上にも`orderId`という業務属性が必要だと分かった場合は、そのEvidenceをもとに改めて判断します。

## 属性の位置づけ

現時点のReferenceでは、業務上の明細らしさを保ちながら周辺モデルを広げすぎない範囲で、次の属性を利用します。

- `orderLineId`: 明細自身を識別するidentity
- `lineNumber`: 受注内の業務上の明細番号
- `itemCode`: 品目を表す業務コード
- `quantity`: 数量
- `unitPrice`: 単価

`lineNumber`をUI上の行番号や配列indexとは定義しません。

`itemCode`についても、このIssueだけを理由に商品Masterや在庫領域まで追加しません。

数量専用型、単位、税、値引、明細金額などは、後続の実例で必要性が確認されるまで先に固定しません。

## 確認すること

- `sales.order_line`をSemantic Dataとして読み込める
- `sales.order.lines`のsourceが`sales.order`である
- Relation targetが`sales.order_line`である
- cardinalityが`many`である
- `OrderLine`の属性を既存`std.*`型で表現できる
- 既存`sales.order.customer` Relationを変更しない
- 既存`sales.order.submit` Capabilityのinput / outputを変更しない
- Reference WorldをRepository自身から再生成できる
- Source-to-Canonical Traceabilityを維持できる
- Semantic追加だけではPersistence Mappingを自動追加しない
- Semantic追加だけでは`order_lines` tableやDatabase Foreign Keyを自動生成しない
- Semantic追加だけでは` sales_core.order ` Contractを暗黙に拡張しない

## Persistence / Contractとの境界

このIssueでは次を変更しません。

- `sales_core`のPersistence Mapping
- Database Foreign Key
- `sales_core.order` Database Contract
- Migration
- Driverの保存・取得処理

Semantic Relationが存在することと、物理DBで参照整合性をForeign Keyとして強制することは別の判断です。

同様に、Semantic Dataを追加したことだけを理由に、既存ApplicationがそのDataを読み書きできる契約を自動的に獲得するわけではありません。

## Application Patternとの境界

このIssueでは`header-detail` Patternを実装しません。

まずSemantic Relationを独立したReferenceとして成立させます。その後、`ISSUE-000012` / GitHub Issue #118で、ApplicationがこのRelationをHeader / Detailの一まとまりとして扱うために必要な情報だけを抽出します。

この順序によって、Semantic Dataの不足とApplication Patternの設計問題を切り分けます。

## 非目標

- `header-detail` Pattern Kindの実装
- 完成した販売管理の明細schema
- 商品Master / 在庫 / 税 / 値引の設計
- `order_lines`物理table
- Database Foreign Key
- Database Contractの拡張
- Migration
- Header / Detailの画面設計
- Driverでの明細編集

## 完了条件

次をReference / Conformanceから再現できることを完了条件とします。

1. `sales.order_line`をSemantic Dataとして表現できる
2. `sales.order.lines -> sales.order_line many`をSemantic Relationとして表現できる
3. Relationのsource / target / cardinalityをCanonical Worldから追跡できる
4. 既存`Order`、customer Relation、submit Capabilityの意味を壊さない
5. Persistence Mapping / FK / Contract / UIを追加せずSemanticだけで関係を成立させられる
6. Reference WorldをRepository自身から再生成できる
7. 後続`header-detail`実証が、このRelationを既存Semanticとして参照できる

## 関連

- GitHub Issue #67: v1 Application Pattern実証
- GitHub Issue #117: Semantic Reference実証
- GitHub Issue #118: `header-detail` Application Pattern実証
- GitHub PR #119: `OrderLine` Semantic Reference実装
- `ISSUE-000005`: Application Pattern全体
- `ISSUE-000012`: `header-detail`実証
- ADR-0017: Application Pattern ownership
- ADR-0018: Application Pattern authoring source
- RQ-001, RQ-003, RQ-014, RQ-017, RQ-021
