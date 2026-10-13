# PostgreSQL行スコープ付き読込：実DB検証手順（#1769）

このReferenceは、[#1656](https://github.com/hat47x/tei-platform/issues/1656)のWeb API契約と既存Driverに、**ひとつの明示SQL realization**を接続する試験である。OData準拠の実装でも、正本の認可Providerでも、あらゆるSchemaで使用できる汎用SQL Compilerでもない。

## E9. 何が実装されているか

- `postgres_scoped_reference.go` はCanonical `order_entry.recent_orders` QueryのID、Source形状、Datasource、Dialect、Databaseを厳密に照合する。別QueryのSQLを自動的に書き換えない。
- 信頼済みで注入された正の `CustomerID` を、**SQLの最初の行絞込み条件** `o.customer_id = $1` として値束縛する。以後の日時条件（`$2`）、許可された `orderId` 比較（`$3`）、Order、Top（最後のbind引数）もクライアント文字列をSQLへ連結しない。
- `orderId`だけの取得要求では `customer_id`、`ordered_at`、`status`、`total_amount` に `NULL AS <column>` を使う。Driverの厳密なResult Column Mappingが期待する**列名は維持しつつ、選択権限のない値を物理結果として取得しない**。識別子と投影式は既知5項目の固定リストからのみ組み立てる。
- Principal ID、Scope ID、Policy Revision、Customer ID、対象Query/Database/SQL形状、選択Field、演算子、件数制限、Queryパラメータのいずれかが不正なら、DBへのQueryContext前に拒否する。認可Grantそのものを新たに発行しない。

## E9-A. 通常CI（実DB未接続）

```sh
go test -count=1 ./reference/web-api-contract
go vet ./reference/web-api-contract
```

`postgres_scoped_reference_test.go` は `database/sql` の偽Connectorで発行SQL・bind引数と不正要求時のSQL未実行を検証する。**実PostgreSQLに接続した証拠ではない。**

## E9-B. 実PostgreSQL（明示opt-in）

本テストは**廃棄可能な独立DB**以外では実行しないこと。実装は`public.orders`を作成するので、運用DBや既存の受注データがあるDBを指定してはならない。

1. 別途管理する使い捨てPostgreSQLインスタンス内に、データベース`tei_web_scoped_e2e`を新規作成する。PostgreSQLが実際に利用できるかと必要な権限を事前確認する。
2. 下記のように、**完全一致のDB名**と別の明示確認フラグを指定してテストする。ログやIssue本文に接続文字列、認証情報を転記しない。

```sh
TEI_WEB_SCOPED_PG_DSN='postgres://USER:REDACTED@HOST:5432/tei_web_scoped_e2e?sslmode=require' \
TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED=tei_web_scoped_e2e \
go test -count=1 -run '^TestScopedPostgresAgainstIsolatedDatabase$' -v ./reference/web-api-contract
```

3. テストは実際の接続先の`current_database()`が`tei_web_scoped_e2e`であること、`public.orders`が**存在しないこと**を確認してからテーブルを作る。既存の同名テーブルは変更しない。実行完了時には作成したテーブルをDROPする。テストが異常終了した場合は廃棄DBそのものを削除し、共有DBで再実行しない。
4. 顧客11と顧客22の受注を混在投入し、高い受注番号を持つ顧客22の行が顧客11の降順TopやFilterに影響しないことを検証する。公開応答だけでなく、**非選択項目が物理結果でもNULLであること**を検査する。標準出力にはDB名とPostgreSQLの`server_version`のみ記録する。
5. 実証記録には、実行した正確なGit commit SHA、Go version、PostgreSQL server version、test実行コマンドの秘密を除いた部分、結果を記載する。DSN未設定による`SKIP`は**実DB検証未完了**であり、CI成功と混同しない。

## E10. Tenant SQLの実PostgreSQL検証（#1769後続）

`scoped_postgres_integration_test.go`は、`scoped_postgres_reference.go`の**Tenant向けSQLそのもの**を既存の`database/sql` PostgreSQL接続で実行する追加テストである。従来のCustomer向け実DB試験では検証できなかった経路を対象にする。

- `TEI_WEB_SCOPED_PG_DSN`と`TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED=tei_web_scoped_e2e`の両方が揃わないと実行しない。実DB名の厳密確認、対象テーブルの未存在確認後にだけ、リポジトリに含まれる使い捨てfixture SQLを実行する。
- テナントA/Bそれぞれの異なるPrincipal/Scopeへの**固定のテストGrant**を用い、相手の大きな受注番号がFilter/Sort/Topに影響しないことをDB実結果で検査する。古いPolicy RevisionのReplayは拒否する。
- `orderId`以外が未選択のとき、公開結果のみならず**SQL実行のRaw Resultでも未選択列がNULL**であることを確認する。テナント参照Providerも、Canonial Raw Result Column Mappingの列名を固定した`NULL AS`へ変更している。
- 既存の`TestScopedPostgresAgainstIsolatedDatabase`（Customer向け）と同じ**隔離DB名**を使用するが、別のテーブル`public.tei_web_scoped_orders_reference`を作成して除去する。テーブルが既存なら何も破壊せず拒否する。
- `SQLTenantScopeAuthority`には静的なテスト実装を注入するため、**本番の権限正本からTenantを導出できた証拠にはならない**。RLS、認可失効とSQLトランザクションの原子性も未実証。

この二つの実DB試験を同じ使い捨てDBで実行するコマンド（接続情報はログへ保存しない）：

```sh
TEI_WEB_SCOPED_PG_DSN='postgres://USER:REDACTED@HOST:5432/tei_web_scoped_e2e?sslmode=require' \
TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED=tei_web_scoped_e2e \
go test -count=1 -run '^TestScoped(Postgres|TenantPostgres)AgainstIsolatedDatabase

- `PostgresCustomerScope` の顧客IDが、認証済み利用者の正本権限・Tenant所属・現在のPolicy Revisionから導出されたこと。このReferenceでは信頼済み**テスト入力**である。
- PostgreSQL RLSや広範なABAC/RBAC、行・利用目的・条件付き項目権限。今回は特定顧客を限定する明示SQL述語の実証に限る。
- 全Query/Datasource、Count/Aggregate/Expand、OData 4.01適合、Command書込、実Host認証・CSRF、Production承認。
- コンカレントな権限変更とQuery実行の原子性。現行のPolicy Revision一致はプロセス内の照合で、DB側の権限正本に同期した失効保証ではない。

**実DB試験が未実行なら、Issue #1769の実DB完了チェックは埋めない。**


## Tenantの参照例との関係

`scoped_postgres_reference.go`と、この文書が扱う`postgres_scoped_reference.go`は、異なる隔離fixtureを使う二つの検証例である。共通の入口は既存の`ReadPlan`と`ScopedBrowseQueryExecutor`であり、通常のExecutorを暗黙にスコープ対応へ置き換えない。

| 例 | スコープ値の入力 | 物理テーブル | 検証の範囲 |
|---|---|---|---|
| Tenantの例 | 注入した`SQLTenantScopeAuthority`から得る文字列`TenantID` | `public.tei_web_scoped_orders_reference` | 偽ConnectorでSQL/Grant照合。実DB向けのopt-in越境・Filter/Sort/Top・物理NULLマスクの試験を追加（実行結果は別途記録） |
| Customerの例 | 信頼済み試験入力`PostgresCustomerScope.CustomerID` | 専用DBに作る`public.orders` | 偽Connectorと明示opt-inの実PostgreSQLで行の越境、Order/Top、非選択列のNULLマスクを確認 |

Customer IDをTenant IDや認可Grantとして扱わない。Tenantの例のSQLをCustomerの実DB試験が実行したとは主張しない。二つの実装を明示選択するReferenceとして併置できるが、正式Providerを二種類採用したことにはならない。

製品の行スコープを共通化するには、現在の権限正本からどの値を導出し、Query・Scope ID・Policy Revisionへどう拘束するかを先に定める必要がある。権限の失効と実SQLの原子性、実Host受理、RLS、正式Providerの選択は[#1656](https://github.com/hat47x/tei-platform/issues/1656)と[#1769](https://github.com/hat47x/tei-platform/issues/1769)に残す。
 -v ./reference/web-api-contract
```

**通常CIでDSN未設定のためSKIPされた場合、Tenant向け実DB検証は完了していない。** 実行証拠が得られた後に、正確なhead SHA、server_version、SQL実行結果を記録する。

## なお未実証

- `PostgresCustomerScope` の顧客IDが、認証済み利用者の正本権限・Tenant所属・現在のPolicy Revisionから導出されたこと。このReferenceでは信頼済み**テスト入力**である。
- PostgreSQL RLSや広範なABAC/RBAC、行・利用目的・条件付き項目権限。今回は特定顧客を限定する明示SQL述語の実証に限る。
- 全Query/Datasource、Count/Aggregate/Expand、OData 4.01適合、Command書込、実Host認証・CSRF、Production承認。
- コンカレントな権限変更とQuery実行の原子性。現行のPolicy Revision一致はプロセス内の照合で、DB側の権限正本に同期した失効保証ではない。

**実DB試験が未実行なら、Issue #1769の実DB完了チェックは埋めない。**


## Tenantの参照例との関係

`scoped_postgres_reference.go`と、この文書が扱う`postgres_scoped_reference.go`は、異なる隔離fixtureを使う二つの検証例である。共通の入口は既存の`ReadPlan`と`ScopedBrowseQueryExecutor`であり、通常のExecutorを暗黙にスコープ対応へ置き換えない。

| 例 | スコープ値の入力 | 物理テーブル | 検証の範囲 |
|---|---|---|---|
| Tenantの例 | 注入した`SQLTenantScopeAuthority`から得る文字列`TenantID` | `public.tei_web_scoped_orders_reference` | 偽ConnectorでSQL/Grant照合。実DB向けのopt-in越境・Filter/Sort/Top・物理NULLマスクの試験を追加（実行結果は別途記録） |
| Customerの例 | 信頼済み試験入力`PostgresCustomerScope.CustomerID` | 専用DBに作る`public.orders` | 偽Connectorと明示opt-inの実PostgreSQLで行の越境、Order/Top、非選択列のNULLマスクを確認 |

Customer IDをTenant IDや認可Grantとして扱わない。Tenantの例のSQLをCustomerの実DB試験が実行したとは主張しない。二つの実装を明示選択するReferenceとして併置できるが、正式Providerを二種類採用したことにはならない。

製品の行スコープを共通化するには、現在の権限正本からどの値を導出し、Query・Scope ID・Policy Revisionへどう拘束するかを先に定める必要がある。権限の失効と実SQLの原子性、実Host受理、RLS、正式Providerの選択は[#1656](https://github.com/hat47x/tei-platform/issues/1656)と[#1769](https://github.com/hat47x/tei-platform/issues/1769)に残す。
