# ISSUE-000023: header-detail Relation traversalの実現境界を定める

- 状態: In Progress
- GitHub Issue: #233
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 関連Issue: `ISSUE-000011` / #117、`ISSUE-000012` / #118、`ISSUE-000017` / #178、`ISSUE-000018` / #179、`ISSUE-000019` / #185
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Gate Bへ向けて`order_entry.order_lines`をruntime traversalへ進めると、`sales.order.lines`というSemantic Relationだけでは、具体的なOrder instanceから複数のOrderLine instanceを取得する方法がまだ定まっていません。

現在のSemanticは次の形です。

```tei
data Order {
    attribute orderId type sales.order_id
    ...
    relation lines to sales.order_line many
}

data OrderLine {
    attribute orderLineId type sales.order_line_id
    attribute lineNumber type std.integer
    attribute itemCode type std.code
    attribute quantity type std.decimal
    attribute unitPrice type std.money
}
```

`OrderLine`には、親Orderのidentityを表すSemantic attributeを置いていません。

これは単なる実装漏れではありません。`ISSUE-000011`では、Driverや物理Databaseの都合だけで`orderId`をOrderLineへ追加せず、`sales.order.lines`というRelation自体を親子関係のSemanticな正本として成立させました。`ISSUE-000012`でも、header-detail PatternはRelation-onlyで成立し、DatasourceやQueryを先回りして持たせないことを確認しています。

一方、`ISSUE-000017`で追加した`RelationKey`は、sourceとtargetのSemantic attribute同士に意味のある対応が存在するRelationで有効です。第一Referenceのcustomer Relationでは、次が自然に成立します。

```text
sales.order.customer_id
    -> customer.customer.customer_id
```

これを`lines`へ機械的に適用すると、`sales.order_line.order_id`のようなtarget attributeを追加したくなります。しかし、そのattributeに独立した業務Semantic上の必要性が確認されていない状態で追加すると、runtime traversalやphysical Foreign Keyの都合をSemanticへ逆流させることになります。

このIssueでは、**keylessな`many` Semantic Relationを実行へ接続する責務をどこへ置くか**を決めます。

## 現在確認できている事実

- `sales.order.lines`はSemantic Relationとして成立済み
- cardinalityは`many`
- `order_entry.order_lines`はRelation-onlyのheader-detail Patternとして成立済み
- `sales.order_line`には親Order identity属性がない
- `sales.order_line`のPersistence Mappingはない
- `sales.order`には`sales.order.order_id`というsource identityがある
- Driverの`ResolvePattern`はheader-detailについてRelation source / target / cardinalityだけを解決する
- physical FKやtable構造をSemanticの正本にしない方針は既に確定している
- `HeaderDetailContractQueryBinding`によって、Pattern本体を変えずにsource attributeからDatabase Contract Query inputへの対応を表現できる
- `sales_core.order_line/find_by_order`をtyped QueryとしてReferenceへ置き、result Data=`sales.order_line`、cardinality=`many`を契約できる
- Binding authoring sourceは`applications/<unit>/realizations/header-detail/*.tei`から読み込み、Source Indexで追跡できる

## 比較した選択肢

### A. OrderLineへ親Order identityをSemantic attributeとして追加する

```text
sales.order.order_id
    -> sales.order_line.order_id
```

OrderLine自身が業務上「どのOrderに属するか」というidentityを保持する必要が、runtimeやPersistenceとは独立して確認できる場合には自然です。

ただし、Driver traversalやphysical Foreign Keyだけを根拠に採用しません。

### B. header-detail Patternが取得Contractとbindingを明示する

Applicationが、このheader-detailで利用するlogical Datasource、Database Contract Query、source attributeからQuery inputへのbindingをPattern自身に持つ案です。

```text
header-detail Pattern
  relation sales.order.lines
  datasource sales
  Contract Query find_by_order
  sales.order.order_id -> input order_id
```

OrderLine側の親IDを要求しない利点があります。一方、Relation-onlyで成立しているPatternへ取得責務を戻すため、既存のApplication intentを不必要に重くします。

### C. Relation traversal realizationを別modelとして置く

Semantic RelationとApplication Patternを維持し、独立realizationが取得方法を明示する案です。

```text
Application Pattern / logical Datasource
        ↓
Database Contract / Query
        ↓
source Semantic attribute -> Query input
        ↓
result Semantic Data = sales.order_line many
```

この案では、OrderLineへ親IDを強制せず、Database固有の実現方法もSemantic RelationやPatternへ逆流させません。

### D. aggregate / nested materializationとして一括取得する

Orderを読み込むQuery自身がOrderLine collectionまでSemantic resultとしてmaterializeする案です。

将来のAPIやTargetでは有効ですが、現在のflatなQuery result境界を大きく広げるため、v1の第一候補にはしません。

### E. physical FK / Persistence Mappingから自動推論する

採用しません。

実装の容易さと引き換えに、物理構造をSemantic Relation traversalの権威へしてしまうためです。

## 現時点の実証結果

### C案をruntime実証前の暫定採用とする

PR #235〜#237で、C案を最小model・Validation・authoringまで実証しました。

追加した境界は次です。

```text
HeaderDetailContractQueryBinding
  Application
  Pattern
  Datasource
  Target: Database / Contract / Query
  Inputs[]
    SourceAttribute -> Query Input
```

Binding identityは`Application + Pattern`です。DatasourceやContract Queryは選択内容であり、identityには含めません。

この構成では、`order_entry.order_lines`は引き続き次のRelation-only Patternのままです。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

取得方法は別authoring sourceへ置きます。

```tei
header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
    input sales.order.order_id to order_id
}
```

### RelationKeyを要求しない

Validationでは、Bindingの`SourceAttribute`がRelation source Dataに属することを確認しますが、`sales.order.lines`にRelationKeyが存在することは要求しません。

これにより、`sales.order_line.order_id`を追加せず、source側の`sales.order.order_id`をcollection取得Queryのinputへ明示的に渡せます。

### Contract Queryの意味契約を分離する

Referenceでは次のtyped Queryを追加して実証します。

```tei
contract sales_core.order_line v1.0 {
    read sales.order_line
    query find_by_order {
        input order_id type sales.order_id
        result sales.order_line many
    }
}
```

ここに物理table名、column名、Foreign Key、SQL本文は含めません。

### 全件Queryへの退行を防ぐ

header-detail traversalでは、少なくとも一つの`SourceAttribute -> Query input` bindingを必須としました。

Query inputが0件のContractを暗黙に利用し、親Orderと無関係な全OrderLineを取得する形はheader-detail traversalとして認めません。

### Source traceabilityを維持する

Bindingは次のSubjectでSource Indexへ登録します。

```text
header-detail-contract-query-binding:<Application>/<Pattern>
```

Validation Diagnosticから独立realization sourceへ戻れることまでConformanceで固定しました。

## 現時点の判断

v1ではC案を基準にruntime実証を続けます。

ただし、現時点ではADRへ昇格させません。次の点を実行経路で確認した後に、境界が十分安定したかを判断します。

- mapped Order instanceから`sales.order.order_id`の実値を取り出せること
- Bindingだけを正本として`find_by_order(order_id)`のparameterを準備できること
- Database Contract Query executorの既存compositionを再利用できること
- 0件を正常な空collectionとして扱えること
- 複数件を`sales.order_line[]`として保持できること
- Execution Errorを空collectionと混同しないこと
- physical PersistenceやFKを追加しなくてもruntime境界を説明できること

これらが成立すれば、AやBへ責務を戻す必要性は低いと判断できます。

## 評価基準

採用する境界は、少なくとも次を満たす必要があります。

- `sales.order.lines`を業務Semanticの正本として維持する
- `OrderLine.orderId`をDriver都合だけで追加しない
- physical FK / table / column名からtraversalを推測しない
- Application Query / Database Contract Query / Semantic Relationのidentityを混同しない
- source instanceから必要な値を明示的に取得できる
- resultが`sales.order_line`かつ`many`であることを検証できる
- logical DatasourceとEnvironment上のDatabase instanceを混同しない
- realization / bindingのauthoring sourceを追跡できる
- PostgreSQL固有の実現方法をCoreへ固定しない
- header-detail PatternへUI layoutを持ち込まない
- 同じRelationを別Targetや別Database realizationでも再利用できる余地を残す
- 汎用ORMやgeneric join engineへ拡張しない

## 実証順

1. A / B / Cについて必要なCanonical identityと責務を比較する — 完了
2. `OrderLine.orderId`を追加しないケースでも成立する最小Conformanceを作る — 完了
3. source attribute → traversal inputの明示bindingをmodel上で表す — 完了
4. result Data / cardinalityのValidation境界を確認する — 完了
5. Pattern本体へ取得情報を追加せず、別realizationで分離できることを確認する — 完了
6. 採用境界をReference authoringへ適用する — 実施中
7. Source Indexからrealization / Contract Queryへ戻れることを確認する — Bindingは完了
8. Driverで`Order -> OrderLine[]`を0件・複数件・Execution Errorを区別して実行する — 未着手
9. 実証で設計が安定した時点でADR化の要否を判断する — 未着手

## 完了条件

- [x] `OrderLine.orderId`をruntime都合だけでは追加しない境界を確認できる
- [x] keylessな`many` Relationを実行へ接続する責務所有者を独立realizationとして試作できる
- [x] source instance → traversal inputの対応を名前・型推測なしで表現できる
- [x] result Data / cardinalityを明示的に検証できる
- [x] physical FKをSemantic Relation keyとして自動採用しない
- [x] 最小Reference authoringを作り、BindingをSource Indexまで追跡できる
- [ ] Driverで`Order -> OrderLine[]`を推測なしで実行できる
- [ ] 0件を正常な空collectionとして扱える
- [ ] Execution Errorと空collectionを区別できる
- [ ] 通常CIが明示的に再有効化された後、関連Conformanceを実行する

## 非目標

- 汎用ORM association
- generic join engine
- UI Grid / Form layout
- physical FKの自動発見
- OrderLine table設計をこのIssueだけで確定すること
- 全種類のcollection Relationを一度に一般化すること

## 関連

- GitHub Issue #69 / #117 / #118 / #178 / #179 / #185 / #233
- GitHub PR #235 / #236 / #237
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000012-header-detail-application-pattern.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `reference/workspace/semantic/sales/order.tei`
- `reference/workspace/applications/order-entry/realizations/header-detail/order-lines.tei`
- `reference/workspace/databases/sales-core/contracts/order-line.tei`
- `internal/driver/resolver.go`
