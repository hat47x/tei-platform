# M0.26 PostgreSQLの実状態と期待構造の差分確認

M0.26では、既存のデータベースの管理手順を実際のPostgreSQLに接続します。保存先への対応付けから導出した期待する物理構造と、データベースの管理情報から観測した実際の物理構造を、一連の処理で比較します。

新しいデータベースの正本モデルを導入するのではなく、既存の境界が実際のデータベースでも成立するかを確かめる実証です。

## 検証する経路

```text
Persistence Mapping
  ↓ persistence.RealizeDesired(..., postgres)
Desired Physical Schema

actual PostgreSQL
  ↓ database/sql
SQLCatalogReader
  ↓ postgres.Provider.Introspect
Actual Physical Schema

Desired + Actual
  ↓ schemadiff.Compare
Schema Diff
```

`SchemaSnapshot`は再取得できる物理構造の観測結果であり、意味の正本である全体モデルではありません。

## database/sqlドライバーの境界

`internal/database/providers/postgres`は引き続き標準の`database/sql`にだけ依存し、具体的なドライバーを選びません。

参照実装の結合検証では`github.com/jackc/pgx/v5/stdlib`を組み合わせます。`database/sql`に対応する接続部を通じて実際のPostgreSQLを利用するため、提供元との接続規約や意味の正本にドライバーの識別情報を持ち込みません。複数の文を含む試験用SQLを扱うため、共有接続設定では簡易通信方式を選びます。

接続文字列は、ホスト側から`TEI_TEST_POSTGRES_DSN`で与えます。接続先、パスワード、ホスト名などは、物理構造の観測結果や差分には保存しません。

## 実状態を再現する試験データ

`tools/reference-postgres-integration`は、結合テスト用に専用のデータベース構造を準備します。試験データを用意するためにDDLを実行しても、TEI自身にデータベース構造を変更する機能が実装されたことにはなりません。

試験用データには、少なくとも次を含めます。

- schema-qualified tables
- bigint
- varchar
- text
- timestamptz
- numeric
- jsonb
- primary keys
- ordinary indexes
- foreign key

期待する構造は手書きの物理構造記録ではなく、`persistence.Mapping`から`RealizeDesired`で生成します。

## 主キーと索引を二重に数えない

実際のPostgreSQLを使った結合検証では、主キーを支えるために自動生成された索引を、`SQLCatalogReader.Indexes`が通常の索引としても取得している問題が見つかりました。

TEIの物理構造モデルでは、主キーと索引を区別します。そのためPostgreSQLの管理情報を読み取る接続部では、`indisprimary`が示す索引を通常の索引一覧から除外します。

これはテストを通すためだけの文字列補正ではありません。同じ物理的な構成要素を主キーと索引の両方として数えないため、提供元固有の表現を整える処理です。

## 基準状態と差分

参照実装の結合検証は、二段階で行います。

### 基準状態

保存先への対応付けから生成した期待する構造と、同じ形に準備した実際のPostgreSQLを比較し、`schemadiff.Compare`が返す変更項目が0件であることを要求します。

### 差分

その後、テスト用の仕組みが実際のデータベースにだけ`drift_note text`という列を追加します。

TEIは実際の構造を再取得し、期待する構造にない列として`drop-column`の差分を検出します。ここでいう`drop-column`は「期待する構造に合わせるなら、実際の構造から削除する対象」を示す差分であり、TEIが自動的に削除命令を実行するわけではありません。

M0.26では、差分を検出しても移行処理を自動的に採用・実行しません。

## 外部キーの確認状況

PostgreSQLの接続部は、実際の外部キーを`SchemaSnapshot.ForeignKeys`へ記録します。参照実装の結合検証では、外部キーを1件取得できることを確かめます。

一方、M0.26時点の`schemadiff.Compare`は外部キーの差分を比較しません。そのため、この段階では実際のデータベースから外部キーを読み取れることまでを確認し、外部キーの差分検出は別の課題としました。

この制約はM0.26時点のものです。その後、M0.27とADR-0008で保存先への対応付けから期待する外部キーを明示し、提供元に依存しない構造差分の比較へ外部キーを追加しました。現在の扱いは[M0.27の準拠確認](foreign-key-drift-m0.27.md)を参照してください。

## 継続的な自動検査

常設の自動検査には、PostgreSQLのサービスを使う独立した実行単位を設けます。

この実行単位では、次を確かめます。

- actual PostgreSQL serverが起動する
- Reference `database/sql` driverで接続できる
- provider.Introspectがactual catalogを読み取れる
- baseline diffが0件
- drift columnを検出
- foreign keyを1件観測
- connection dataがSnapshot / Diffへ混入しない

通常の単体テストや参照用データの検証とは実行単位を分け、外部サービスとの結合部分で起きた失敗を特定できるようにします。

## 非目標

M0.26では、次の機能を本番運用できる品質で実現したとは主張しません。

- connection pool sizing / failover
- TLS / certificate policy
- production credential management
- automatic schema apply
- migration executor
- PostgreSQL server administration
- backup / replication / role management
- Oracle等ほかのDB providerのactual integration
- Foreign Key diffの一般化

目的は、TEIが区別する期待状態・実際の状態・差分の境界が、実際のPostgreSQLでも保たれることを確かめることです。
