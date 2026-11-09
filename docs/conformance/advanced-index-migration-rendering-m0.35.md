# M0.35 高度な索引のマイグレーションSQL準拠確認

## 目的

M0.35では、高度な索引の期待する物理構造をPostgreSQL用のDDLへ変換し、実際のPostgreSQLへ適用した後に、期待する構造と一致することを確認します。

`CREATE INDEX`の文字列を生成できるだけでは不十分です。期待する構造の意味を失わず、比較未確定や運用上の判断が必要な場合は処理を停止します。適用後の実際の構造を再取得し、差分がなくなることまで確かめます。

## 責務の境界

共通部分の`internal/database/migration`には、PostgreSQL固有のDDL構文を持たせません。

`migration.Render`は、次の二つを担当します。

- `Diff.Unresolved`が残っている場合にSQL生成を拒否する
- 対象の提供元を選び、その提供元が管理するSQL生成処理へ委ねる

PostgreSQL固有の構文は`internal/database/providers/postgres`が管理します。

## `CREATE INDEX`で保持する意味

PostgreSQL用のSQL生成処理は、物理構造に記録された次の索引の要素をSQLへ変換します。

- 列キー
- 式を使うキー
- キーの順序
- collation
- operator class
- `ASC` / `DESC`
- `NULLS FIRST` / `NULLS LAST`
- `INCLUDE`
- 部分索引の条件式
- 一意索引の`NULLS NOT DISTINCT`
- B-tree以外のaccess method

B-treeはPostgreSQLで既定の索引方式です。物理構造にある索引方式が`btree`の場合は、`USING btree`を省略してもよいものとします。

## ProviderExpression

式を使うキーと部分索引の条件式は、ProviderExpressionとして扱います。

PostgreSQL用のSQL生成処理では、次の条件を検査します。

- 提供元が`postgres`であること
- SQLの方言が`postgresql`であること
- 元の式が空でないこと
- 表現の種類が既知であること
- `provider-normalized`の場合は表記統一の規則を識別できること

元の式の本文は、SQLの構文解析器がない状態で推測して書き換えません。ProviderExpressionはレビュー対象のデータベース定義コードとして扱い、実行時に渡される任意の入力とは信頼の境界を分けます。

## 識別子の引用

照合順序と演算子クラスは、任意のSQL断片として埋め込まず、識別子として適切に引用します。

この段階で安全に扱える名前の形式は次のとおりです。

- `name`
- `schema.name`

三階層以上の名前や、現在の物理構造だけでは意味を確定できない表現は、推測せず拒否します。

## `DROP INDEX`の安全境界

`DROP INDEX`は親テーブルのスキーマ名を使い、必ずスキーマを明示した名前で生成します。

例は次のとおりです。

```sql
DROP INDEX "sales"."customers_name_idx";
```

構造差分が持つテーブルの識別情報に、`customers`のようにスキーマ名が含まれていない場合は、`search_path`に依存する削除を避け、SQL生成を拒否します。

## 通常のマイグレーションに含めない運用判断

M0.35では、`CREATE INDEX CONCURRENTLY`と`DROP INDEX CONCURRENTLY`を自動的に選択しません。

`CONCURRENTLY`の指定は、ロックの時間、トランザクションの境界、失敗後の復旧方法などに関わる運用上の判断です。期待する索引の構造とは分けて扱います。

また、実際の索引の`valid`、`ready`、`live`に異常があることを示す`index-state-drift`も、自動的な削除・再作成へは変換しません。復旧方法を一意に決められないため、運用上の判断が必要な状態としてSQL生成を停止します。

## 単体での準拠確認

`internal/database/providers/postgres/migration_sql_test.go`と関連テストでは、少なくとも次を確認します。

- 高度な索引の主要な要素を、一つの`CREATE INDEX`へ反映できる
- B-tree以外のaccess methodを`USING`で生成できる
- 従来の単純な索引の生成結果を維持する
- PostgreSQL以外の提供元固有の式を拒否する
- B-tree以外の索引にB-tree専用の並べ替えやNULLの順序を指定した場合は拒否する
- `Diff.Unresolved`が残る場合は拒否する
- `index-state-drift`からDDLを自動生成しない
- スキーマ名を明示した`DROP INDEX`を生成する
- スキーマ名を特定できない`DROP INDEX`を拒否する
- 通常のSQL生成では`CONCURRENTLY`を指定しない

## PostgreSQL実環境での準拠確認

自動検査で使うPostgreSQL 17では、`tools/reference-postgres-migration-rendering`によって、次の一連の処理を実行します。

```text
Desired Physical Schema
  -> Actualとの差分
  -> PostgreSQL migration SQL
  -> SQLを実PostgreSQLへ適用
  -> Introspect
  -> Desiredと再比較
  -> Changes = 0 / Unresolved = 0
```

試験用データでは三つの索引を作り、次の内容を実際のPostgreSQLで検証します。

1. 照合順序、既定でない演算子クラス、`DESC NULLS LAST`、`INCLUDE`、部分索引の条件式を持つB-tree索引
2. `lower(email)`を使う式による索引
3. `NULLS NOT DISTINCT`を指定した一意索引

自動検査では、生成したSQLを実行できるだけでなく、実際の状態を再取得したときに三つの索引が存在し、構造差分が完全になくなることを確認します。

## 既存の準拠確認との関係

M0.35は、次の既存の規則を前提とします。

- M0.29: Advanced Index Actual / Diff
- M0.30: Advanced Index Desired authoring
- M0.31: 比較未確定とChangeの分離
- M0.32: ProviderExpression representation / comparison
- M0.33: provider-owned offline realization policy
- M0.34: collation / operator classの安全な比較契約

比較段階で確定できない違いを、DDLの生成へそのまま渡さないことが、M0.31以降の共通した安全上の原則です。

## 現時点で対象にしないこと

M0.35では、次の機能は扱いません。

- 本番データベースへの自動適用
- テーブルの規模や負荷に応じた`CONCURRENTLY`の自動選択
- 無効な索引について復旧方法を自動決定すること
- 分割索引に固有の運用
- operator class parameter
- 提供元固有の識別子を完全に構造化すること
- 任意のSQL入力を安全に扱うためのSQL構文解析器

これらは必要性と責務の境界を確認したうえで、後続の課題として扱います。
