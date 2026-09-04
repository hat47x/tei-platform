# ISSUE-000024: keyless header-detail RelationのDatabase-ownedな物理保存境界を定める

- 状態: In Progress
- GitHub Issue: #244
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前段: `ISSUE-000023` / GitHub Issue #233
- 関連Issue: `ISSUE-000017` / #178、`ISSUE-000018` / #179
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

`ISSUE-000023`では、`sales.order.lines`のようなkeylessな`many` Semantic Relationを、Application PatternへDatasourceやQueryの責務を戻さず、独立した`HeaderDetailContractQueryBinding`から実行へ接続する方針を確定しました。

一方、managed-here Databaseである`sales_core`では、親Orderとの関係を物理的に保存する必要があります。ここで`OrderLine.orderId`を物理実装の都合だけでSemantic attributeとして追加すると、Databaseの保存方式がCanonicalな業務意味へ逆流します。

このIssueでは、Semantic Relation、Application traversal、Database storage realizationを分離したまま、keylessな親子Relationを保存・取得できる最小境界を実証します。

## 採用した境界

v1では、Persistence Mappingへ**Relation-ownedな物理保存境界（Relation Storage）**を追加します。

通常のSemantic attribute列は、従来どおり`ColumnMapping`で表します。

```tei
persist sales_core.order_line {
    database sales_core
    semantic sales.order_line
    table public.order_lines

    column sales.order_line.order_line_id -> order_line_id storage int64 required
    ...
}
```

親Orderとの関係を保存する`order_id`は、OrderLineのSemantic attributeへ昇格させず、別authoring sourceでRelation Storageとして明示します。

```tei
relation-storage sales_core.order_line {
    relation sales.order.lines
    component sales.order.order_id -> order_id storage int64 required
}
```

意味の流れは次のようになります。

```text
Semantic Relation
  sales.order.lines
        ↓
Database-owned Relation Storage
  sales.order.order_id -> public.order_lines.order_id
        ↓
Contract Query SQL realization
  find_by_order(order_id)
```

`order_id`という物理列名自体をSemantic Relationのidentityにはしません。Relation Storageは、どのSemantic Relationを、source側のどのSemantic attributeから、どの物理support columnへ保存するかを明示します。

## 実装済みの境界

### Canonical model

Persistence Mappingに次を追加しました。

```text
RelationStorageMapping
  ├─ Relation
  └─ Components[]
       ├─ SourceAttribute
       ├─ Column
       ├─ Storage
       └─ Nullability
```

component順序を保持するため、将来composite source identityが必要になっても、物理列名やFKから対応順を推測する必要はありません。

### 局所Validation

Persistence Mapping単体では、次を検査します。

- Relation identityの形式
- componentが1件以上あること
- source attributeの重複がないこと
- support column名が空でないこと
- 通常のSemantic attribute列や別Relation Storageと物理列名が衝突しないこと
- storage type / nullabilityが妥当であること

### World-level Validation

Semantic Worldとの参照整合性として、次を検査します。

- Relationが実在すること
- Relation target DataとPersistence MappingのSemantic Dataが一致すること
- source attributeが実在すること
- source attributeがRelation source Dataに属すること

RelationKeyやphysical Foreign Keyの存在は要求しません。keylessなRelationでも、Semantic Relationとsource attributeが明示されていれば成立します。

### Desired Physical Schema

Relation Storageのsupport columnは、通常のSemantic attribute列とは別にDesired Physical Schemaへ投影します。

`public.order_lines.order_id`はDesired tableのcolumnとして現れますが、Primary Key、Unique、Index、Foreign Keyへ暗黙参加しません。制約やaccess pathが必要な場合は、Database-ownedな別decisionとして明示します。

### AuthoringとSource Index

Relation Storageは次の配置から読み込みます。

```text
databases/<database>/relation-storage/*.tei
```

既存の明示Persistence Mappingへ合流し、Source Indexでは次のsubjectを持ちます。

```text
persistence-relation-storage:<mapping>/<relation>
```

第一Referenceでは次のsourceへ追跡できます。

```text
databases/sales-core/relation-storage/order-line.tei
```

### Reference Persistence

`sales_core.order_line`は`public.order_lines`へ保存します。

通常のSemantic attribute列は、OrderLine自身の5属性だけです。

- `order_line_id`
- `line_number`
- `item_code`
- `quantity`
- `unit_price`

親Orderとの関係を保持する`order_id`は通常の`ColumnMapping`には入れず、Relation Storageだけが所有します。

### `find_by_order` SQL realization

確定したDatabase-owned storageを正本として、`sales_core.order_line / find_by_order`のPostgreSQL realizationを追加しました。

```sql
SELECT
    order_line_id,
    line_number,
    item_code,
    quantity,
    unit_price
FROM public.order_lines
WHERE order_id = :order_id
ORDER BY line_number, order_line_id;
```

Contract inputからphysical parameterへの対応も明示します。

```text
input order_id -> physical parameter order_id
```

raw resultはOrderLine自身のSemantic属性だけへmappingします。Relation support columnの`order_id`はSELECT resultへ含めず、`sales.order_line.order_id`のような存在しないSemantic属性を生成しません。

### Reference統合Conformance

実際のReference Workspaceを通常Loaderで読み込み、次を確認するConformanceを追加しました。

- Relation Storageが`sales_core.order_line`へ合流する
- 通常のSemantic attribute列に`order_id`が混入していない
- Relation StorageとSemantic RelationがValidation上整合する
- `find_by_order` SQL realizationとQuery signatureが整合する
- Relation StorageとSQL realizationのauthoring sourceへSource Indexから戻れる

## physical Foreign Keyの扱い

v1の第一Referenceでは、physical Foreign KeyをRelation Storage成立の必須条件にはしません。

理由は、Semantic Relation、Database support column、physical FKは同じものではないためです。FKを必要とする具体的な整合性要件が確認できた場合は、Database-ownedなdecisionとして別途明示します。

したがって、FKがないことを理由にSemantic RelationやRelation Storageを未定義とは扱いません。また既存FKを発見したことを理由にSemantic Relationを自動生成しません。

## 実PostgreSQL Evidence

停止前のGitHub Actionsでは、PostgreSQL integrationに共通して`TEI_TEST_POSTGRES_DSN`を使用していました。この規約を再利用し、header-detail経路を実PostgreSQLへ通す手動Evidence toolを追加しました。

```text
tools/reference-postgres-header-detail-runtime
```

このtoolは次を一続きに実行します。

```text
Reference Workspace
  → order_entry.order_lines
  → HeaderDetailContractQueryBinding
  → sales_core.order_line/find_by_order
  → development / sales_core.dev
  → database.query.postgres
  → instance-aware runtime Registry
  → SQL realization
  → PostgreSQL Runner
  → raw result
  → Semantic OrderLine[]
```

fixtureでは`order_id=1001`に2件、別Orderに1件を用意し、対象Orderの2件取得と、存在しない`order_id=9999`の0件取得を確認する構成です。

ただし、2026年9月4日現在、GitHub Actions workflowはRepositoryで明示的に停止されています。また、このチャットの実行環境から実PostgreSQLサーバーへ接続してtoolを実行する経路もありません。

そのため、**実PostgreSQLでの成功Evidenceはまだ取得していません**。toolを実装したことと、実DBで成功したことは分けて扱います。

## 現在残っている作業

### 1. 実PostgreSQLでの実行確認

`TEI_TEST_POSTGRES_DSN`を与えられる環境で`tools/reference-postgres-header-detail-runtime`を実行し、少なくとも次を確認します。

- `order_id=1001`で2件取得できる
- `order_id=9999`で0件を正常な空collectionとして扱える
- Environment / Database instance / executor Capabilityが明示identityのまま保たれる
- parameter値がEvidenceへ保存されない
- `order_id` support columnがOrderLineのSemantic属性として返らない

### 2. CI再開後の通常runner検証

CIをこのIssueから独断で再有効化しません。別の明示的な判断でActionsが再開された後、既存の`postgres-integration`規約へこのtoolを接続し、通常runnerでも再現できるEvidenceを残します。

## 完了条件

- [x] keyless Semantic Relationのphysical storage責務をDatabase-ownedなRelation Storageとして定めた
- [x] `sales.order.lines`をReferenceで保存できる最小authoringを定めた
- [x] OrderLineのSemantic attributeとDatabase support columnを分離した
- [x] Relation StorageをDesired Physical Schemaへ接続した
- [x] `find_by_order` SQL realizationを推測なしで定義した
- [x] result column mappingを`sales.order_line`へ明示した
- [x] v1ではphysical Foreign Keyを必須化せず、採用時はDatabase-ownedな別decisionとして扱う方針を明示した
- [x] Source Indexから保存realization / SQL realizationへ追跡できる
- [x] 実PostgreSQL用の再現可能な手動Evidence toolを用意した
- [ ] 0件・複数件の実PostgreSQL readを実際に確認する
- [ ] 通常CI再開後に関連Conformanceと実PostgreSQL Evidenceを再実行する

## 非目標

- 汎用ORM association
- 全Relationの物理保存方式を一度に一般化すること
- UIのheader-detail layout
- physical schemaからSemantic Relationを自動生成すること
- Semantic modelへphysical schemaを逆流させること
- physical Foreign Keyを全Relationへ必須化すること
- external-observed `master_data`の物理EvidenceをこのIssueで扱うこと

## 関連

- GitHub Issue #69 / #117 / #178 / #179 / #185 / #233 / #244
- GitHub PR #235〜#253
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000023-header-detail-relation-traversal-boundary.md`
- `reference/workspace/databases/sales-core/persistence/order-line.tei`
- `reference/workspace/databases/sales-core/relation-storage/order-line.tei`
- `reference/workspace/databases/sales-core/contracts/order-line.tei`
- `reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.tei`
- `reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.sql`
- `tools/reference-postgres-header-detail-runtime/main.go`
