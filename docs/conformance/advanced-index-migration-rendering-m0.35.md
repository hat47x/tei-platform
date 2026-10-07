# M0.35 Advanced Index マイグレーションSQL Conformance

## 目的

M0.35では、Advanced IndexのDesired Physical SchemaをPostgreSQLのDDLへ変換し、生成したDDLを実際のPostgreSQLへ適用した後もDesiredへ収束することを固定します。

重要なのは、単に`CREATE INDEX`の文字列を組み立てられることではありません。Desiredが持つ意味を欠落させず、比較未確定や運用判断を必要とする状態では安全側に停止し、実行後のActualを再観測して差分が消えるところまでを確認します。

## 責務の境界

共通の`internal/database/migration`は、PostgreSQL固有のDDL文法を持ちません。

`migration.Render`が担うのは次の2点です。

- `Diff.Unresolved`が残っている場合にSQL生成を拒否する
- 対象プロバイダーを選び、プロバイダー固有のSQL生成処理へ委譲する

PostgreSQLの文法は`internal/database/providers/postgres`が所有します。

## CREATE INDEXで保持する意味

PostgreSQL rendererは、Physical Indexに表現された次の要素をSQLへ戻します。

- 列キー
- expression key
- キーの順序
- collation
- operator class
- `ASC` / `DESC`
- `NULLS FIRST` / `NULLS LAST`
- `INCLUDE`
- partial indexのpredicate
- unique indexの`NULLS NOT DISTINCT`
- B-tree以外のaccess method

B-treeはPostgreSQLの既定access methodなので、Physical Index上のmethodが`btree`である場合は`USING btree`を省略してよいものとします。

## ProviderExpression

expression keyとpredicateはProviderExpressionとして扱います。

PostgreSQL rendererは次を検査します。

- providerが`postgres`であること
- dialectが`postgresql`であること
- sourceが空でないこと
- representationが既知であること
- `provider-normalized`の場合はnormalization authorityがあること

source本文は、SQL parserを持たない状態で推測による書き換えを行いません。ProviderExpressionはレビュー対象となるデータベース定義コードとして扱い、任意のruntime入力とは別の信頼境界に置きます。

## 識別子の引用

collationとoperator classはSQL断片として扱わず、識別子として引用します。

現在安全に扱う形は次です。

- `name`
- `schema.name`

3階層以上の名前や、現在のPhysical Schemaから構造を確定できない表現は推測せず拒否します。

## DROP INDEXの安全境界

`DROP INDEX`は親tableのschemaを使い、必ずschema-qualified nameとして生成します。

例は次のとおりです。

```sql
DROP INDEX "sales"."customers_name_idx";
```

Schema Diffのtable identityが`customers`のようにschemaを含まない場合は、`search_path`に依存した削除を行わずSQL生成を拒否します。

## 通常のマイグレーションに含めない運用判断

M0.35では`CREATE INDEX CONCURRENTLY`と`DROP INDEX CONCURRENTLY`を自動選択しません。

`CONCURRENTLY`は、ロック時間、transaction境界、失敗時の回復方法などに関わる運用上の選択です。DesiredなIndex構造とは分けて扱います。

また、Actualで`valid` / `ready` / `live`の異常を観測した`index-state-drift`も、自動的なdrop/recreateへ変換しません。回復方法を一意に決められないため、運用判断が必要な状態としてSQL生成を停止します。

## Unit Conformance

`internal/database/providers/postgres/migration_sql_test.go`と関連テストで、少なくとも次を確認します。

- Advanced Indexの主要要素を1本の`CREATE INDEX`へ保持できる
- B-tree以外のaccess methodを`USING`で生成できる
- simple indexの既存生成結果を維持する
- PostgreSQL以外のProviderExpressionを拒否する
- B-tree以外にB-tree専用のsort / NULL orderingを付けた場合は拒否する
- `Diff.Unresolved`が残る場合は拒否する
- `index-state-drift`からDDLを自動生成しない
- schema-qualifiedな`DROP INDEX`を生成する
- schemaを確定できない`DROP INDEX`を拒否する
- 通常のSQL生成で`CONCURRENTLY`を付けない

## 実PostgreSQL Conformance

CIのPostgreSQL 17では、`tools/reference-postgres-migration-rendering`を使って次の閉ループを実行します。

```text
Desired Physical Schema
  -> Actualとの差分
  -> PostgreSQL migration SQL
  -> SQLを実PostgreSQLへ適用
  -> Introspect
  -> Desiredと再比較
  -> Changes = 0 / Unresolved = 0
```

fixtureでは3本のIndexを作成し、次の意味を実際のPostgreSQLで検証します。

1. collation、non-default operator class、`DESC NULLS LAST`、`INCLUDE`、partial predicateを持つB-tree Index
2. `lower(email)`を使うexpression Index
3. `NULLS NOT DISTINCT`を持つunique Index

CIは、生成したSQLが実行可能であるだけでなく、再観測後に3本のIndexが存在し、Schema Diffが完全に収束することを確認します。

## 既存Conformanceとの関係

M0.35は、次の既存契約を前提とします。

- M0.29: Advanced Index Actual / Diff
- M0.30: Advanced Index Desired authoring
- M0.31: 比較未確定とChangeの分離
- M0.32: ProviderExpression representation / comparison
- M0.33: provider-owned offline realization policy
- M0.34: collation / operator classの安全な比較契約

比較段階で確定できない差をDDL生成段階へ押し流さないことが、M0.31以降の共通した安全境界です。

## 現時点で行わないこと

M0.35では次を扱いません。

- production DBへの自動適用
- table sizeや負荷を見た`CONCURRENTLY`の自動選択
- invalid indexの自動回復方法の決定
- partitioned index固有の運用
- operator class parameter
- provider-native識別子の完全な構造化
- 任意のSQL入力を安全化するためのSQL parser

これらは必要性と責務境界を確認したうえで、後続Issueとして扱います。
