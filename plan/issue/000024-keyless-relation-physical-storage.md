# ISSUE-000024: keyless header-detail RelationのDatabase-ownedな物理保存境界を定める

- 状態: Ready
- GitHub Issue: #244
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前段: `ISSUE-000023` / GitHub Issue #233
- 関連Issue: `ISSUE-000017` / #178、`ISSUE-000018` / #179
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

`ISSUE-000023`では、`sales.order.lines`のようなkeylessな`many` Semantic Relationを、Application PatternへDatasourceやQueryの責務を戻さず、独立した`HeaderDetailContractQueryBinding`から実行へ接続できることを確認しました。

現在のReferenceでは、論理的な実行経路は次の形まで成立しています。

```text
sales.order.lines
  ↓
order_entry.order_lines
  ↓
HeaderDetailContractQueryBinding
  sales.order.order_id -> input order_id
  ↓
sales_core.order_line / find_by_order
  ↓
development / sales_core.dev
  ↓
database.query.postgres
```

Driverはこの経路を、RelationKeyやphysical Foreign Keyを推測せずに解決できます。0件は正常な空collection、複数件は正常な`many` result、Executorの失敗はExecution Errorとして区別できます。

一方、managed-here Databaseである`sales_core`にOrderLineをどのように保存し、`find_by_order`が利用する親Orderとの物理的な対応をどこへ記述するかは、まだ決まっていません。

## 問題

現行Persistence Mappingは、基本的にSemantic attributeとphysical columnを一対として記述します。

```tei
column sales.order.order_id -> order_id storage int64 required
```

しかし、`sales.order_line`には親Order identityに相当するSemantic attributeを置いていません。これは実装漏れではなく、`ISSUE-000011`と`ISSUE-000023`で、DriverやForeign Keyの都合だけを理由に`OrderLine.orderId`をSemanticへ追加しないことを確認した結果です。

この状態で、たとえばOrderLine tableへ`order_id`というsupport columnを追加しても、現行Persistence Mappingには、そのcolumnをSemantic Relationの物理実現として表す正しい置き場がありません。

したがって、次の三層を分離したまま、Database-ownedな物理保存境界を定める必要があります。

```text
Semantic
  sales.order.lines
        ↓
Application traversal
  sales.order.order_id -> find_by_order(order_id)
        ↓
Database storage realization
  親子関係をどのtable / column / constraint / indexで保持するか
```

## 比較する選択肢

### A. `OrderLine.orderId`をSemantic attributeとして追加する

OrderLine自身が、業務上独立した意味として「所属するOrderのidentity」を属性として持つ必要が確認できた場合には自然です。

ただし、Driver traversal、SQL、Foreign Key、Migrationを実装しやすくする目的だけでは採用しません。

### B. Persistence MappingへRelation-ownedなphysical storageを追加する

現時点の第一候補です。

概念上は次のような境界です。

```text
persist sales_core.order_line
  semantic sales.order_line
  table ...

  relation-storage sales.order.lines
    source sales.order.order_id
    column order_id
```

この形では、target Dataへ新しいSemantic attributeを追加せず、DatabaseがRelationを保存するために必要なsupport columnをDatabase-ownedなrealizationとして保持できます。

実際の構文・modelはこのIssueの実証で最小化します。上記構文を先に仕様として固定しません。

### C. Relation専用のlink storageへ分離する

OrderLine本体とは別のlink table等で関係を保存する案です。

同じSemantic Relationを別の物理構造へ写せる点では一般性がありますが、第一Referenceには過剰になり得ます。B案で表現できない実例が現れるまで先回りして一般化しません。

### D. SQL realizationだけにphysical columnを埋め込む

採用しません。

`find_by_order`のSQLだけに`order_id`等を記述すると、managed-here Databaseであるにもかかわらず、そのcolumnやconstraintの正本がschema / migration側に残りません。Queryが暗黙のschema前提を持つ状態になるためです。

### E. physical Foreign Keyや命名規則から自動推論する

採用しません。

physical schemaをSemantic Relationの権威へ逆転させ、既存DBの偶然の命名や制約有無に意味解釈が依存するためです。

## 上位原則

このIssueでは、次を維持します。

- `sales.order.lines`を業務Relationの正本とする
- `OrderLine.orderId`を物理実装の都合だけでは追加しない
- physical table / column / Foreign KeyはDatabase-ownedなdecisionとして扱う
- SQL realizationだけにschema前提を隠さない
- provider固有の物理表現をSemanticやApplication Patternへ持ち込まない
- physical storageが未定義なら未解決として停止し、名前やFKから補完しない
- Source Indexから物理realizationのauthoring sourceへ戻れるようにする
- 将来、別providerや別storage方式へ再選択できる余地を残す

## 実証順

1. 現行Persistence Mapping / parser / Validation / migration生成の責務を確認する
2. keyless Relation向けphysical storageに必要な最小Canonical modelを比較する
3. modelだけで、Semantic attributeとsupport columnを混同しないことをConformance化する
4. 参照整合性Validationを追加する
5. authoring sourceとSource Index追跡を追加する
6. PostgreSQL migration / schema表現へ接続する
7. Referenceの`sales.order_line` persistenceを追加する
8. 必要なindex / Foreign KeyをDatabase-ownedなdecisionとして明示する
9. `find_by_order`のSQL realizationを、確定したphysical storageから明示的に作成する
10. raw result column → `sales.order_line` Semantic attribute mappingを明示する
11. 実PostgreSQLで0件・複数件・Execution Errorを確認する
12. 通常CIが明示的に再有効化された後、関連Conformanceを通常runnerで再検証する

## 完了条件

- [ ] keyless Semantic Relationのphysical storage責務所有者を決める
- [ ] `sales.order.lines`をReferenceで保存できる最小authoringを定める
- [ ] OrderLineのSemantic attributeとDatabase support columnを混同しない
- [ ] migration / schema生成または明示DDLへ接続できる
- [ ] `find_by_order` SQL realizationを推測なしで定義できる
- [ ] result column mappingを`sales.order_line`へ明示できる
- [ ] physical Foreign Keyを採用する場合もDatabase-ownedなdecisionとして保持する
- [ ] 0件・複数件の実PostgreSQL readをConformanceで確認する
- [ ] Source Indexから保存realization / SQL realizationへ追跡できる
- [ ] 通常CI再開後に関連Conformanceを再実行する

## 非目標

- 汎用ORM association
- 全Relationの物理保存方式を一度に一般化すること
- UIのheader-detail layout
- physical schemaからSemantic Relationを自動生成すること
- Semantic modelへphysical schemaを逆流させること
- external-observed `master_data`の物理EvidenceをこのIssueで扱うこと

## 関連

- GitHub Issue #69 / #117 / #178 / #179 / #185 / #233 / #244
- GitHub PR #235〜#243
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000023-header-detail-relation-traversal-boundary.md`
- `reference/workspace/semantic/sales/order.tei`
- `reference/workspace/databases/sales-core/persistence/order.tei`
- `reference/workspace/databases/sales-core/contracts/order-line.tei`
