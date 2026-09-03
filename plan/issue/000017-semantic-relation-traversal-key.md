# ISSUE-000017: Semantic Relationをinstance間で辿るkey対応を明示する

- 状態: In Progress
- GitHub Issue: #178
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 後続Issue: `ISSUE-000023` / GitHub Issue #233（keylessな`many` Relationのruntime traversal）
- 関連RQ: RQ-005 / RQ-009 / RQ-012

## 背景

Application Driverで`reference` Patternを実行へ進めると、Semantic Relationがtarget Dataとcardinalityだけでは、source instanceからtarget instanceを一意に辿れないケースがあることが分かりました。

Driverはattribute名、型の類似、物理Foreign Key、Persistence Mappingからこの対応を推測しません。sourceとtargetのSemantic attribute同士に意味のあるidentity対応が存在する場合、その対応をSemantic Relation自身が明示的に所有できるようにします。

ただし、**RelationKeyをすべてのRelationに必須なruntime実装情報とは扱いません**。

`customer`のように両端のSemantic attributeが自然に対応するRelationと、`lines`のようにtarget Dataへ親identity attributeを露出しなくても業務Relationが成立するケースを区別します。

## 第一Referenceで確定した境界

`order_entry.customer_reference`では、次をRelation keyとして明示しました。

```tei
relation customer to customer.customer one
    key sales.order.customer_id -> customer.customer.customer_id
```

意味は次です。

```text
sales.order.customer_id
        ↓
customer.customer.customer_id
```

これは物理Foreign KeyやDatabase Queryを表すものではありません。source Data instanceとtarget Data instanceの間に存在する、Semantic identityの対応です。

## ここまでに実装したこと

- `semantic.RelationKey` model
- Relation key authoring構文
- source / target Semantic attributeの存在確認
- source attributeがRelationのsource Dataへ属することの確認
- target attributeがRelation target Dataへ属することの確認
- source / targetのSemantic型identity一致確認
- 完全重複pairの拒否
- 同一target attributeへ複数source attributeを対応付ける曖昧性の拒否
- keyを持たない概念的Relationを引き続き有効とする境界
- 第一Reference `sales.order.customer`へのkey適用
- Reference Contract Query inputへtarget keyを明示bindingする横断Validation
- `browse` mapped rowのsource key値を、Relation key経由でContract Query parameterへ渡すruntime境界

## 責務分離

Relation keyは次だけを所有します。

```text
source Semantic attribute
        ↓
target Semantic attribute
```

次は所有しません。

- SQL / Database API / stored procedure
- Database Contract Query名
- Application datasource alias
- physical table / column / Foreign Key
- runtime connection
- target Dataに存在しない親identity attributeの代用品

Database Contract Queryのinput / result / cardinality / realizationは`ISSUE-000018`、Applicationのreference PatternからQueryを呼び出すbindingは`ISSUE-000019`で扱います。

## header-detailで見つかった境界

`sales.order.lines`は次のSemantic Relationとして成立しています。

```text
sales.order.lines
  source: sales.order
  target: sales.order_line
  cardinality: many
```

一方、`sales.order_line`には親Order identityに相当するSemantic attributeがありません。

`ISSUE-000011`では、DriverやPersistenceの都合だけで`OrderLine.orderId`を追加しないことを明示的に選びました。したがって、header-detail runtimeを進めるためだけに次を追加することはしません。

```text
sales.order.order_id
        ↓
sales.order_line.order_id
```

この対応が業務Semanticとして独立に必要だと確認できればRelationKeyとして表現できます。しかし、そうでない場合はkeylessな`many` Relationを別の明示的なtraversal realizationや取得Contractへ接続する必要があります。

この判断は`ISSUE-000023` / GitHub Issue #233へ切り出します。

## RelationKey modelの現在地

第一Referenceの`customer` Relationについては、model、authoring、Validation、runtime parameter準備まで一続きに接続できています。

そのため、`ISSUE-000017`の責務は「すべてのRelationをRelationKeyで実行可能にすること」ではありません。Semantic attribute同士に明示可能な対応が存在するRelationについて、その対応を名前・型・物理構造から推測せず表現することです。

keylessなRelationをどう実行するかは別の設計問題として扱います。

## CIの扱い

2026年9月3日現在、GitHub Actions workflowはRepositoryから明示的に削除されています。CI未実行をgreenとは扱いません。

CIが別の明示的な判断で再有効化された後、RelationKey model / authoring / Validation / Reference runtime Conformanceを通常runnerで再実行します。

## 完了条件

- [x] Relation keyをCanonical modelで表現できる
- [x] Relation keyをauthoring sourceから読み込める
- [x] source / target attributeの所属とSemantic型identityを検査できる
- [x] target側からsource値を一意に逆引きできないkey定義を拒否できる
- [x] keyを持たない概念的Relationを無理に補完しない
- [x] 第一Referenceのcustomer Relationで実値をQuery inputまで運べる
- [x] header-detailへtarget側親identityをDriver都合だけで追加しない境界を明確化する
- [x] keylessな`many` Relationのruntime traversalを`ISSUE-000023`へ分離する
- [ ] CIが明示的に再有効化された後、関連Conformanceが通常runnerで成功する

## 非目標

- physical Foreign KeyをSemantic Relationの正本にすること
- attribute名や型からRelation keyを自動生成すること
- すべてのRelationへRelationKeyを必須化すること
- target Dataに存在しないattributeをruntime都合で追加すること
- 汎用ORMのassociation model
- UI上の親子表示仕様

## 関連

- GitHub Issue #69 / #117 / #178 / #179 / #185 / #233
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000019-reference-contract-query-invocation.md`
- `plan/issue/000023-header-detail-relation-traversal-boundary.md`
- `reference/workspace/semantic/sales/order.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `internal/semantic/model.go`
- `internal/validation/semantic_relation_keys.go`
- `internal/driver/reference_contract_query_parameters.go`
