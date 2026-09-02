# ISSUE-000017: Semantic Relationをinstance間で辿るkey対応の正本を決める

- 状態: Ready
- GitHub Issue: #178
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 関連Issue: `ISSUE-000016` / GitHub Issue #167、`ISSUE-000019` / GitHub Issue #185
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Application Driverで`browse`の実行結果を`record / reference / header-detail`へ渡す境界を検討したところ、現在のSemantic Relationには、概念上のつながりは表現されている一方、**実データ上のinstanceをどのkeyで辿るか**を示す正本がないことが分かりました。

現行のRelationは、identity、target Data、cardinalityを保持します。

```text
Relation
  ├─ identity
  ├─ target Data
  └─ cardinality
```

しかし、source側のどのSemantic attributeの値をtarget側のどのSemantic attributeへ対応付けるかは保持していません。

この不足をDriverがattribute名、型、物理column名、Foreign Key等から推測して補うことはしません。

## Referenceで露出した二つのRelation

### OrderからCustomerを参照するRelation

Semanticには次があります。

```text
sales.order.customer
  source: sales.order
  target: customer.customer
  cardinality: one
```

実データ上では、Referenceとして次の対応が想定されます。

```text
sales.order.customer_id
        ↓
customer.customer.customer_id
```

Application PatternはRelationとlogical Datasourceを明示しています。

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

`sales -> sales_core`と`master -> master_data`を跨ぐRelationであるため、sales側の物理Foreign KeyをそのままRelation traversalの正本にはできません。

このRelationのtarget keyを、どのDatabase Contract Queryのどのinputへ渡すかはRelation自身の責務には含めません。その接続は`ISSUE-000019`のReference invocationが所有します。

### OrderからOrderLineを辿るRelation

Semanticには次があります。

```text
sales.order.lines
  source: sales.order
  target: sales.order_line
  cardinality: many
```

header-detail Patternは次です。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

一方、現行`sales.order_line`には`orderId`相当のattributeがありません。

```text
sales.order_line
  order_line_id
  line_number
  item_code
  quantity
  unit_price
```

この状態では、Driverが`Order.orderId`とtarget側の何かを名前や型から対応付けることすらできません。

## このIssueで決めること

Relationの概念的意味を壊さず、instance traversalに必要なkey対応をどの層が正本として所有するかを、実Referenceから決めます。

少なくとも次の三案を比較します。

### A. Semantic Relation自身がkey対応を持つ

概念例:

```text
relation sales.order.customer
  source-key sales.order.customer_id
  target-key customer.customer.customer_id
```

業務上のRelationとidentity対応が不可分なら、最も直接的な表現です。

### B. Relationの実現・Projectionを別に持つ

Semantic Relationは概念上の関係だけを保持し、Application、Database、または別のRealizationがinstance traversal方法を明示します。

同じRelationを異なる物理方式や外部APIで実現できる必要がある場合、この分離が適する可能性があります。

### C. Persistence Mappingや型一致から導出する

物理Foreign Key、Persistence Mapping、attribute型等から推定します。

cross-datasource Relationや`OrderLine.orderId`の欠如を説明しにくいため、少なくともv1の主たる正本にはしない方向で検証します。

## Reference invocationとの責務分離

`ISSUE-000019`では、`reference` Patternが利用するDatabase Contract Queryと、そのinputへのbindingを扱います。

```text
Semantic Relation
  source-key / target-key
        ↓
Reference invocation
  利用するContract / Query
  target-key -> Query input
        ↓
Database Contract Query
```

このIssueが所有するのは、source instanceとtarget instanceの間を結ぶSemantic key対応までです。どのQueryを呼び出すか、どのQuery inputへ値を渡すかはApplication側のReference invocationへ分離します。

これにより、同じSemantic Relationを異なるDatabase Contractや将来の別実現で利用する場合でも、RelationへDatabase固有の呼び出し情報を埋め込まずに済みます。

## `OrderLine.orderId`を先に追加しない

このIssueでは、Driverやheader-detailを動かすためだけに`OrderLine`へ`orderId`を追加しません。

追加する場合は、それが

- 業務上のreferenceとして本当に存在する意味なのか
- 物理Foreign KeyをSemanticへコピーしただけなのか

を区別します。

親子関係がSemantic上の包含として成立し、物理的な結合keyだけがRealization側に必要な可能性も残します。

## 実証順

1. `sales.order.customer`でcross-datasourceなone Relationを検討する
2. `sales.order.lines`でmany Relationを検討する
3. 二例で共通に必要な意味と、実現方式にだけ必要な情報を分ける
4. source / target attributeの存在と所属をValidationできるようにする
5. authoring sourceとSource Index上の来歴を確認する
6. `ISSUE-000019`のReference invocationがtarget keyを明示bindingとして参照できることを確認する
7. Driverが名前・型・物理schemaを推測せずRelation traversalへ進めることを確認する
8. 長期的な所有境界を固定する必要があればADRへ昇格する

## 完了条件

- [ ] Relation traversalのsource key / target keyを名前や型から推測しない
- [ ] `Order → Customer`をcross-datasourceで明示的に辿れる
- [ ] `Order → OrderLine`のmany Relationを明示的に辿れる
- [ ] key対応の正本とauthoring sourceを追跡できる
- [ ] 物理Foreign KeyをSemantic Relationの正本と自動的に同一視しない
- [ ] RelationへDatabase Contract Queryのidentityやinput名を埋め込まない
- [ ] `ISSUE-000019`の明示bindingからtarget keyを参照できる
- [ ] `master_data`のread-only境界を維持する
- [ ] DriverがRelation traversal固有の命名規則を持たない
- [ ] `record / reference / header-detail`のScenarioへ安全に渡せる
- [ ] 必要ならADRへ昇格できるEvidenceが揃う

## 非目標

- 汎用ORM
- generic join engine
- SQL JOINの自動生成一般化
- Foreign Key DDLの設計
- Database Contract Queryの選択やinput binding
- production向けDataLoader / cache / batching
- GraphQL風resolver framework
- Target UIのnavigation設計

## 関連

- GitHub Issue #69 / #167 / #178 / #179 / #185
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000016-native-query-runtime-boundary.md`
- `plan/issue/000019-reference-contract-query-invocation.md`
- `reference/workspace/semantic/sales/order.tei`
- `reference/workspace/semantic/customer/customer.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `reference/workspace/applications/order-entry/patterns/order-lines.tei`
- `reference/workspace/databases/sales-core/persistence/order.tei`
