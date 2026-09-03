# ISSUE-000023: header-detail Relation traversalの実現境界を定める

- 状態: Ready
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
- Database ContractにもOrderLine read契約はない
- Driverの`ResolvePattern`はheader-detailについてRelation source / target / cardinalityだけを解決する
- `sales.order`には`sales.order.order_id`というsource identityがある
- physical FKやtable構造をSemanticの正本にしない方針は既に確定している

## 比較する選択肢

### A. OrderLineへ親Order identityをSemantic attributeとして追加する

```text
sales.order.order_id
    -> sales.order_line.order_id
```

OrderLine自身が業務上「どのOrderに属するか」というidentityを保持する必要が、runtimeやPersistenceとは独立して確認できる場合には自然です。

ただし、Driver traversalやphysical Foreign Keyだけを根拠に採用しません。

### B. header-detail Patternが取得Contractとbindingを明示する

Applicationが、このheader-detailで利用するlogical Datasource、Database Contract Query、source attributeからQuery inputへのbindingを所有する案です。

```text
header-detail Pattern
  relation sales.order.lines
  datasource sales
  Contract Query find_by_order
  sales.order.order_id -> input order_id
```

OrderLine側の親IDを要求しない利点があります。一方、Relation-onlyで成立しているPatternへ取得責務を追加するため、Pattern本体へ持たせる必然性を確認する必要があります。

### C. Relation traversal realizationを別modelとして置く

Semantic RelationとApplication Patternを維持し、下位のrealizationが取得方法を明示する案です。

```text
Application / Relation / logical Datasource
        ↓
Database Contract / Query
        ↓
source Semantic attribute -> Query input
        ↓
result Semantic Data = sales.order_line many
```

この案では、OrderLineへ親IDを強制せず、Database固有の実現方法もSemantic Relationへ逆流させません。

一方、Application固有性、logical Datasource、Environment差し替え可能性をどのidentityへ持たせるかを整理する必要があります。

### D. aggregate / nested materializationとして一括取得する

Orderを読み込むQuery自身がOrderLine collectionまでSemantic resultとしてmaterializeする案です。

将来のAPIやTargetでは有効ですが、現在のflatなQuery result境界を大きく広げるため、v1の第一候補にはしません。

### E. physical FK / Persistence Mappingから自動推論する

採用しません。

実装の容易さと引き換えに、物理構造をSemantic Relation traversalの権威へしてしまうためです。

## 初期仮説

現時点では、Aを前提化せず、**BまたはCのようにsource側の明示Semantic identityから取得Contract inputへbindingし、取得したtarget rowsをContract result Dataとして受け取る形**が有力です。

この形なら、`sales.order.lines`を業務Relationの正本として維持しつつ、OrderLineへ物理FK由来の親IDを強制せずにruntime traversalを構成できます。

ただし、初期仮説をそのままADRへ昇格させません。最小Referenceで責務の重複とSource traceabilityを比較し、最も小さく再選択可能な境界を採用します。

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

1. A / B / Cについて、必要なCanonical identityと責務を最小modelとして比較する
2. `OrderLine.orderId`を追加しないケースでも成立する最小Conformanceを作る
3. source attribute → traversal inputの明示bindingをmodel上で表せるか確認する
4. result Data / cardinalityのValidation境界を確認する
5. Application Pattern本体へ取得情報を追加する必要があるか、別realizationで分離できるか比較する
6. 採用境界をReference authoringへ適用する
7. Source IndexからRelation / invocationまたはrealization / Contract Queryへ戻れることを確認する
8. Driverで`Order -> OrderLine[]`を0件・複数件・Execution Errorを区別して実行する
9. 実証で設計が安定した時点でADR化の要否を判断する

## 完了条件

- [ ] `OrderLine.orderId`がSemanticとして必要かどうかをruntime都合とは分離して判断できる
- [ ] keylessな`many` Relationを実行へ接続する責務所有者を決める
- [ ] source instance → traversal inputの対応を名前・型推測なしで表現できる
- [ ] result Data / cardinalityを明示的に検証できる
- [ ] physical FKをSemantic Relation keyとして自動採用しない
- [ ] 最小Reference authoringを作り、Source Indexまで追跡できる
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
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000012-header-detail-application-pattern.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `reference/workspace/semantic/sales/order.tei`
- `internal/driver/resolver.go`
