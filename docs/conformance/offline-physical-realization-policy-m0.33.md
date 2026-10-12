# M0.33 オフラインでの物理構造生成方針の準拠確認

M0.33では、保存先への対応付けから期待する物理構造を生成する際の提供元固有の知識を、保存処理の中核部分と接続を伴う実行時の提供元接続口から分離しました。

この文書では、それぞれの責務が実装上どのように分けられているかを確かめます。

## 対象となる境界

期待する物理構造の生成は、次の三層に分かれます。

```text
Persistence Mapping / persistence.RealizeDesired
          |
          | PhysicalRealizationPolicyだけを見る
          v
provider-owned offline policy
          ^
          |
          +-- postgres.RealizationPolicy
          +-- oracle.RealizationPolicy

外側のcomposition
          |
          +-- database/realization
              provider ID -> policy -> persistence.RealizeDesired
```

保存処理の中核部分はPostgreSQLやOracleのパッケージを取り込みません。データベースごとの物理型、索引の既定値、接続せずに確定できる表記統一の規則は、提供元ごとのパッケージが管理します。

## 接続せずに完結する処理

`PhysicalRealizationPolicy`には、次の情報を渡しません。

- `context.Context`
- `DatabaseInstance`
- DB接続
- 認証情報
- runtime Provider object

そのため、期待する物理構造を生成するだけなら、稼働中のデータベースへ接続する必要はありません。

作業領域の読み込みや検証にも、M0.33によって新たなデータベース接続は発生しません。保存先の定義を読み込む処理と、必要な提供元に合わせて期待する物理構造を生成する処理は、引き続き分かれています。

## 提供元が担う責務

M0.33で`PhysicalRealizationPolicy`へ移した責務は、現時点で必要な次の範囲に限ります。

- 提供元に依存しない`StorageType`から物理型への変換
- 索引方式の既定値
- 一意索引におけるNULLの扱いの既定値
- 接続なしで確定できる索引キーの再現可能な表記統一

PostgreSQLでは、従来は保存処理の中核部分にあった次の規則を、この方針へ移しました。

- `int64 -> pg_catalog.int8`
- `varchar -> pg_catalog.varchar`
- `text -> pg_catalog.text`
- `datetime-tz -> pg_catalog.timestamptz`
- `decimal -> pg_catalog.numeric`
- `json -> pg_catalog.jsonb`
- 索引方式の既定値を`btree`とする
- 一意索引でのNULLの扱いの既定値を`distinct`とする
- B-treeの`ASC NULLS LAST`を実際に適用される既定値の表現へそろえる
- B-treeの`DESC NULLS FIRST`を実際に適用される既定値の表現へそろえる
- `pg_catalog.default`の照合順序を既定の表現へそろえる

Oracleについても、既存の保存型の変換規則を、Oracle側の接続を必要としない方針へ移しました。

## 組立て側が担う責務

`internal/database/realization`は、提供元の識別子から`PhysicalRealizationPolicy`を選び、物理構造の生成処理へ渡すための組立て部分です。

提供元ごとの選択処理をここに集約するため、CLIや結合検証用のツールは個別の方針の型を知る必要がありません。また、その選択処理を保存処理の中核部分へ戻すこともありません。

未対応の提供元は、汎用的な既定値で代用せず、明示的にエラーとします。意味を確定できない構造を、見かけだけもっともらしい共通値へ変換しません。

## 単体での準拠確認

`internal/database/realization/realize_test.go`では、少なくとも次を確認します。

1. PostgreSQLの識別子から、接続不要のPostgreSQL用方針を取得できる
2. Oracleの識別子から、接続不要のOracle用方針を取得できる
3. 汎用の提供元に組込みの方針を割り当てない
4. 保存上の意図が同じでも、PostgreSQLでは`pg_catalog.int8`、Oracleでは`NUMBER(19)`へ変換される
5. PostgreSQLでの索引方式と一意索引のNULLの扱いの既定値を維持する
6. PostgreSQLのB-treeにおける昇順・NULLの順序・既定の照合順序の表記統一が、組立て部分を通しても維持される

保存処理のパッケージを単体でテストするときは、提供元固有の方針を外から直接渡します。これにより、物理構造の生成が提供元の識別子による分岐ではなく、明示した接続口に依存して動作することも確かめます。

## PostgreSQL実環境での準拠確認

既存の`reference-postgres-integration`は、M0.33以降、`database/realization`を通して期待する物理構造を生成します。

その期待構造と、PostgreSQL 17の実際のデータベースの管理情報から読み取った構造を比較し、基準状態で差分が0件になることを引き続き確認します。

これにより、方針を分離した後も、少なくとも次の内容が実際のデータベースで維持されているかを確認できます。

- physical type
- Primary Key / UNIQUE constraint / Foreign Key
- simple / unique index
- Advanced Indexの観測
- PostgreSQLのeffective default normalization

## 実行時の提供元接続口との分離

`database/provider.Provider`は実際のデータベースとのやり取りを担う実行時の接続口であり、M0.33では変更しません。

期待する構造を生成するために`Capabilities`や`Introspect`を呼び出すこともありません。接続を必要としない方針と、実行時に接続する提供元の機構を、一つの万能なインターフェースへ統合しないことを準拠条件とします。

## 提供元固有の式との関係

ADR-0012とM0.32では、ProviderExpressionの`authored`と`provider-normalized`を区別し、比較結果を三つの状態に分けました。

M0.33では、利用者が記述した任意の式を、提供元による復元表現へ変換する構文解析器や復元機構は追加しません。提供元の版やサーバーとの往復を含む条件が、まだ確定していないためです。

`PhysicalRealizationPolicy`は、現時点で確定している接続不要の物理構造生成だけを扱います。式以外の提供元固有の識別子や、実際に適用される既定値の追加規則は、Issue #58で引き続き扱います。

## 非目標

M0.33では、次の機能を実装しません。

- runtime Provider SPIの統合・拡張
- Workspace load時のDB接続
- arbitrary SQL expressionのcanonicalization
- production databaseを使ったnormalization用DDL
- external provider policyのplugin discovery
- Advanced Index DDL rendering

これらを一度に取り込まず、期待する物理構造を生成する責務の境界を先に安定させます。
