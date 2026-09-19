# ISSUE-000024: keyless header-detail RelationのDatabase-ownedな物理保存境界を定める

- 状態: Done
- GitHub Issue: #244
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前段: `ISSUE-000023` / GitHub Issue #233
- 実Provider Evidence: GitHub Issue #308
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

`ISSUE-000023`では、`sales.order.lines`のようなkeylessな`many` Semantic Relationを、Application PatternへDatasourceやQueryの責務を戻さず、独立した`HeaderDetailContractQueryBinding`から実行へ接続する方針を確定しました。

managed-here Databaseである`sales_core`では親Orderとの関係を物理的に保存する必要があります。しかし、`OrderLine.orderId`を物理実装の都合だけでSemantic attributeとして追加すると、Databaseの保存方式がCanonicalな業務意味へ逆流します。

このIssueでは、Semantic Relation、Application traversal、Database storage realizationを分離したまま、keylessな親子Relationを保存・取得できる境界を成立させました。

## 採用した境界

Persistence Mappingへ、通常のSemantic attribute列とは別に**Database-ownedなRelation Storage**を置きます。

```tei
relation-storage sales_core.order_line {
    relation sales.order.lines
    component sales.order.order_id -> order_id storage int64 required
}
```

意味の流れは次です。

```text
Semantic Relation
  sales.order.lines
        ↓
Database-owned Relation Storage
  sales.order.order_id -> public.order_lines.order_id
        ↓
Database Contract Query
  sales_core.order_line / find_by_order(order_id)
        ↓
PostgreSQL SQL realization
        ↓
Driver header-detail runtime
```

ここで次を混同しません。

- `sales.order.lines`: 業務上のSemantic Relation
- `sales.order.order_id`: Relation source側のSemantic identity
- `public.order_lines.order_id`: Database-owned support column
- `find_by_order(order_id)`: Database Contract Query
- native SQL / PostgreSQL: physical realization

`order_id`という物理列名をSemantic Relationのidentityにはせず、`sales.order_line.order_id`のような親Order identityをruntime / FK都合だけでSemanticへ追加しません。

## 成立した実装

### Canonical model / Validation

Persistence MappingにRelation Storageを追加し、少なくとも次を明示的に保持します。

```text
RelationStorageMapping
  ├─ Relation
  └─ Components[]
       ├─ SourceAttribute
       ├─ Column
       ├─ Storage
       └─ Nullability
```

局所ValidationとWorld-level Validationにより、次を検査します。

- Relation identityとcomponent定義
- source attribute重複
- support column衝突
- storage type / nullability
- Relationの存在
- Relation target DataとPersistence MappingのSemantic Dataの一致
- source attributeがRelation source Dataに属すること

RelationKeyやphysical Foreign Keyの存在は要求しません。keylessなRelationでも、Relationとsource attributeが明示されていれば成立します。

### Desired Physical Schema

Relation Storageのsupport columnはDesired Physical Schemaへ投影します。

`public.order_lines.order_id`はtable columnとして現れますが、Primary Key、Unique、Index、Foreign Keyへ暗黙参加しません。必要なconstraint / access pathはDatabase-ownedな別decisionとして明示します。

### Authoring / Source Index

Relation Storageは次から読み込みます。

```text
databases/<database>/relation-storage/*.tei
```

第一Referenceの正本:

```text
reference/workspace/databases/sales-core/relation-storage/order-line.tei
```

Source Indexでは次のsubjectからauthoring sourceへ戻れます。

```text
persistence-relation-storage:<mapping>/<relation>
```

### `find_by_order` realization

第一Referenceでは次を正本とします。

```text
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.tei
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.sql
```

SQLはDatabase-ownedな`order_id` support columnを検索条件として利用し、resultにはOrderLine自身のSemantic attributeだけを返します。

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

Contract inputからphysical parameterへの対応も明示し、raw resultからSemantic attributeへのmappingも明示します。Relation support columnの`order_id`をSemantic resultへ漏らしません。

PR #299で、計画文書に残っていた旧Reference pathもこの正本へ揃えました。

## physical Foreign Keyの扱い

第一Referenceではphysical Foreign KeyをRelation Storage成立の必須条件にしません。

Semantic Relation、Database support column、physical Foreign Keyは異なる責務です。FKを必要とする具体的な整合性要件が確認された場合は、Database-ownedな別decisionとして扱います。

既存FKを発見しただけでSemantic Relationを生成することもしません。

## Evidence toolの安全境界

実PostgreSQL engineでruntime Evidenceを取得するため、次を用意しています。

```text
tools/reference-postgres-header-detail-runtime
```

実Provider Evidence自体は#308へ分離しましたが、Evidence toolの安全境界は本Issueの成果として成立済みです。

PR #294 / #300:

- 既存`public.order_lines`があればfail closed
- 既存tableをDROPしてfixtureへ置換しない
- tool自身がCREATEしたfixtureだけをcleanupする
- `evidenceNature = real-postgres-ephemeral-fixture`を明示する

PR #306:

- Workspace Load後に通常の`validation.Validate`を実行する
- Validation ErrorがあればPostgreSQLへ接続しない
- ValidなCanonical Workspaceだけを前提にfixture Evidenceへ進む

このtoolはproduction既存Databaseの物理事実を観測するものではありません。

## 実Provider Evidenceの分離

実PostgreSQL engineで2件 / 0件 / Execution Errorを確認する作業は、境界設計の未実装ではありません。

そのため、次へ責務を分けます。

- GitHub Issue #308: real PostgreSQL engine + tool-owned ephemeral fixture Evidence
- `ISSUE-000025` / #255: generated Reference artifact再現性
- `ISSUE-000029` / #273: Gate B総合監査

実Provider Evidenceが未取得であることを理由に、本IssueをIn Progressのまま保持しません。

## 完了条件

- [x] keyless Semantic Relationのphysical storage責務をDatabase-owned Relation Storageとして定めた
- [x] `sales.order.lines`をReferenceで保存できる最小authoringを定めた
- [x] OrderLine Semantic attributeとDatabase support columnを分離した
- [x] Relation StorageをDesired Physical Schemaへ接続した
- [x] `find_by_order` SQL realizationを推測なしで定義した
- [x] raw result mappingを`sales.order_line`へ明示した
- [x] physical Foreign KeyをSemantic Relationと混同しない
- [x] Source Indexからstorage / SQL realizationへ追跡できる
- [x] 実PostgreSQL engine用Evidence toolを用意した
- [x] Evidence toolが既存資産を破壊しないfail-closed guardを持つ
- [x] Invalid WorkspaceではDB接続前に停止する
- [x] Evidence natureをephemeral fixtureとして明示する
- [x] Reference追跡先を現行正本へ揃えた
- [x] 実Provider Evidenceを#308へ分離した

## 非目標

- 汎用ORM association
- 全Relationの物理保存方式を一度に一般化すること
- UIのheader-detail layout
- physical schemaからSemantic Relationを自動生成すること
- Semantic modelへphysical schemaを逆流させること
- physical Foreign Keyを全Relationへ必須化すること
- production既存Databaseをfixtureで上書きすること
- #214 external-observed `master_data` Evidenceを代替すること

## 関連

- GitHub Issue #69 / #117 / #178 / #179 / #185 / #233 / #244 / #273 / #308
- GitHub PR #235〜#253 / #294 / #299 / #300 / #306
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000023-header-detail-relation-traversal-boundary.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `reference/workspace/databases/sales-core/persistence/order-line.tei`
- `reference/workspace/databases/sales-core/relation-storage/order-line.tei`
- `reference/workspace/databases/sales-core/contracts/order-line.tei`
- `reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.tei`
- `reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.sql`
- `tools/reference-postgres-header-detail-runtime/`
