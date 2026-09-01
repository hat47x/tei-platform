# ISSUE-000011: `sales.OrderLine`とOrder 1:N RelationをSemantic Referenceへ追加する

- 状態: Done
- GitHub Issue: #117（完了）
- 実装PR: #119（merge済み）
- 上位Issue: `ISSUE-000005` / GitHub Issue #67
- 後続Issue: `ISSUE-000012` / GitHub Issue #118

## 目的

`header-detail` Application Patternを実証する前提として、受注明細そのものの業務上の意味をSemantic Referenceとして成立させました。

Patternの都合からData構造を逆算するのではなく、まず`OrderLine`というDataと、`Order`が複数の`OrderLine`を持つ関係をSemantic上の正本として表しています。

この実証では、Persistence Mapping、Database Foreign Key、Database Contract、画面構造を同時に追加していません。Semantic上の関係と、その関係を物理DBやApplicationでどのように利用するかを分離したまま確認しました。

## 成立したSemantic Reference

PR #119で次を追加しました。

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

Canonical Worldでは、`sales.order.lines`を次の関係として追跡できます。

```text
sales.order
    1
    │
    └── many sales.order_line
```

## `orderId`を追加しなかった理由

`OrderLine`へ`orderId`属性を先回りして追加していません。

`OrderLine`が`Order`に属するという業務上の関係は、まず`Order.lines` Relationを正本とします。物理DBでその関係をどの列やForeign Keyによって実現するかはPersistence側の設計です。

`orderId`をこの段階で必須属性にすると、Semantic上の意味と物理的な関連付け方法を暗黙に同一視するおそれがあります。

後続Driverで保存・取得を実証するときに、Semantic上にも`orderId`という業務属性が必要だと分かった場合は、そのEvidenceをもとに改めて判断します。

## 属性の位置づけ

今回のReferenceでは、明細としての業務上の意味を保ちながら周辺モデルを広げすぎない範囲で、次の属性を採用しました。

- `orderLineId`: 明細自身を識別するidentity
- `lineNumber`: 受注内の業務上の明細番号
- `itemCode`: 品目を表す業務コード
- `quantity`: 数量
- `unitPrice`: 単価

`lineNumber`をUI上の行番号や配列indexとは定義しません。

`itemCode`についても、この実証だけを理由に商品Masterや在庫領域まで拡張していません。数量専用型、単位、税、値引、明細金額なども、後続の実例で必要性が確認されるまで固定しません。

## 確認できたこと

Repository内のReference / Conformanceでは、次を確認しました。

- `sales.order_line`をSemantic Dataとして読み込める
- `sales.order.lines`のsourceが`sales.order`である
- Relation targetが`sales.order_line`である
- cardinalityが`many`である
- `OrderLine`の属性を既存`std.*`型で表現できる
- 既存`sales.order.customer` Relationを変更していない
- 既存`sales.order.submit` Capabilityのinput / outputを変更していない
- Semantic追加だけではPersistence Mappingを自動追加しない
- Semantic追加だけでは`order_lines` tableやDatabase Foreign Keyを自動生成しない
- Semantic追加だけでは`sales_core.order` Contractを暗黙に拡張しない
- Reference WorldをRepository自身の`tei world`から再生成できる

PR #119のCI #277は`verify` / PostgreSQL integrationとも成功しました。

## Persistence / Contractとの境界

この実証では次を変更していません。

- `sales_core`のPersistence Mapping
- Database Foreign Key
- `sales_core.order` Database Contract
- Migration
- Driverの保存・取得処理

Semantic Relationが存在することと、物理DBで参照整合性をForeign Keyとして強制することは別の判断です。

同様に、Semantic Dataを追加したことだけを理由に、既存ApplicationがそのDataを読み書きできる契約を自動的に獲得するわけではありません。

## Application Patternとの境界

このIssueでは`header-detail` Patternを実装していません。

Semantic Relationを独立したReferenceとして先に成立させたことで、後続`ISSUE-000012` / GitHub Issue #118は、`OrderLine`の意味を再設計せず、既存RelationをApplication上のHeader / Detail意図として利用できるかだけに集中できます。

## 完了条件の確認

1. `sales.order_line`をSemantic Dataとして表現できる — 完了
2. `sales.order.lines -> sales.order_line many`をSemantic Relationとして表現できる — 完了
3. Relationのsource / target / cardinalityをCanonical Worldから追跡できる — 完了
4. 既存`Order`、customer Relation、submit Capabilityの意味を壊さない — 完了
5. Persistence Mapping / FK / Contract / UIを追加せずSemanticだけで関係を成立させる — 完了
6. Reference WorldをRepository自身から再生成できる — 完了
7. 後続`header-detail`実証が、このRelationを既存Semanticとして参照できる — 完了

## 関連

- GitHub Issue #67: v1 Application Pattern実証
- GitHub Issue #117: Semantic Reference実証（完了）
- GitHub Issue #118: `header-detail` Application Pattern実証
- GitHub PR #119: `OrderLine` Semantic Reference
- `ISSUE-000005`: Application Pattern全体
- `ISSUE-000012`: `header-detail`実証
- ADR-0017: Application Pattern ownership
- ADR-0018: Application Pattern authoring source
- RQ-001, RQ-003, RQ-014, RQ-017, RQ-021
