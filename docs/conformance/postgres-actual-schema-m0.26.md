# M0.26 PostgreSQLの実状態と期待構造の差分確認

M0.26では、既存のDatabase lifecycleをactual PostgreSQLへ接続し、Persistence Mappingから導出したDesired Physical Schemaと、実DB catalogから観測したActual Physical Schemaをend-to-endで比較します。

新しいCanonical DB modelを導入するのではなく、既存境界が実DBでも成立するかを確認するvertical sliceです。

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

`SchemaSnapshot`はregeneratableなphysical observationであり、Canonical Worldではありません。

## database/sqlドライバーの境界

`internal/database/providers/postgres`は引き続き標準`database/sql`へだけ依存し、具体driverを選びません。

Reference integrationのassemblyとして`github.com/jackc/pgx/v5/stdlib`を利用します。`database/sql`互換adapterを通じてactual PostgreSQLへ接続するため、provider SPIやCanonical modelへdriver identityを持ち込みません。fixtureの複文SQLを維持するため、共有接続設定でsimple protocolを選択します。

接続文字列は`TEI_TEST_POSTGRES_DSN`からHost-sideに与えます。DSN、password、host等をPhysical Schema SnapshotやSchema Diffへ保存しません。

## 実状態を再現する試験データ

`tools/reference-postgres-integration`はintegration harnessとして専用schemaを準備します。fixture setupのDDLは、TEIがschema-change capabilityを実装したことを意味しません。

fixtureは少なくとも次を含みます。

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

Desired側は同じ構造を手書きPhysical Snapshotにせず、`persistence.Mapping`から`RealizeDesired`で生成します。

## 主キーと索引を二重に数えない

actual PostgreSQL integrationによって、PostgreSQLがPrimary Keyのために自動生成するbacking indexを`SQLCatalogReader.Indexes`が通常Indexとしても観測していた継ぎ目が見つかりました。

TEI physical modelではPrimary KeyとIndexは別概念です。そのためPostgreSQL catalog adapterでは`indisprimary`のindexを通常Index一覧から除外します。

これはtestを通すための文字列補正ではなく、同じphysical objectをPrimary KeyとIndexとして二重に表現しないためのprovider normalizationです。

## 基準状態と差分

Reference integrationは二段階で検証します。

### 基準状態

Persistence Mappingから生成したDesiredと、同じ形に準備したactual PostgreSQLを比較し、`schemadiff.Compare`のchangesが0件であることを要求します。

### 差分

その後、test harnessがactual側だけに`drift_note text` columnを追加します。

TEIは再introspectionし、Desiredには存在しないactual-only columnとして`drop-column`差分を検出します。ここでいう`drop-column`は「Desiredへ一致させるならactual側から消える対象」というDiff表現であり、TEIが自動的にDROPを実行する意味ではありません。

M0.26では差分検出後にmigrationを自動採用・実行しません。

## 外部キーの確認状況

PostgreSQL providerはactual foreign keyを`SchemaSnapshot.ForeignKeys`へ観測します。Reference integrationでは1件のFKが実際に取得できることを確認します。

一方、現行`schemadiff.Compare`はForeign Key差分を比較対象にしていません。したがってM0.26は「FK introspectionは実DBで成立する」ことまでを確認し、FK drift generalizationは別課題とします。

この制約はM0.26時点のものです。後続M0.27 / ADR-0008でPersistence MappingからDesired FKを明示し、provider-neutral Schema DiffへForeign Keyを統合しました。現在の境界は [M0.27 Conformance](foreign-key-drift-m0.27.md) を参照してください。

## 継続的な自動検査

常設CIにはPostgreSQL serviceを使う独立jobを置きます。

このjobでは次を確認します。

- actual PostgreSQL serverが起動する
- Reference `database/sql` driverで接続できる
- provider.Introspectがactual catalogを読み取れる
- baseline diffが0件
- drift columnを検出
- foreign keyを1件観測
- connection dataがSnapshot / Diffへ混入しない

通常のunit / Reference fixture検証とはjobを分け、外部service integrationの失敗境界を明確にします。

## 非目標

M0.26では次をproduction-readyと主張しません。

- connection pool sizing / failover
- TLS / certificate policy
- production credential management
- automatic schema apply
- migration executor
- PostgreSQL server administration
- backup / replication / role management
- Oracle等ほかのDB providerのactual integration
- Foreign Key diffの一般化

目的は、TEI固有のDesired / Actual / Diff境界が実PostgreSQLでも壊れないことを確認することです。
