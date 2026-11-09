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

`scoped_postgres_integration_test.go`は、`scoped_postgres_reference.go`の**Tenant用SQL自体**を実PostgreSQLへ接続するオプトイン試験である。従来のCustomer用実DB試験では検証していなかった経路を扱う。

- 専用使い捨てDB `tei_web_scoped_e2e`の実DB名、明示確認フラグ、テーブル未存在を検証してから、リポジトリの`fixtures/postgres_scoped_orders.sql`を適用する。既存テーブルの上書きは拒否する。
- Tenant A/B別のPrincipal・Scope・Policy Revisionから構成した**固定のテストGrant**を使い、相手の受注番号がFilter・Sort・Topへ混入しないことを実DBで検査する。古いRevisionの再利用は失敗させる。
- Tenant用のSELECTは未選択列を`NULL AS <alias>`に変え、Driverが期待するCanonical結果列名を維持しつつ、権限外の生値を物理取得しない。実DBテストでもRaw ResultがNULLになることを検査する。
- Customer用とTenant用は同じ使い捨てDBを利用できるが、作成・廃棄するテーブルは別である。
- `SQLTenantScopeAuthority`はテストfixtureを用いるため、**実業務の認可正本からTenant Grantを導出できた証拠ではない**。Policy失効とQueryの原子性、RLS、Host受理は未実証。

秘密を含まない実行例（DSNは使い捨てDB専用の接続文字列へ置換する）：

```sh
TEI_WEB_SCOPED_PG_DSN='postgres://USER:REDACTED@HOST:5432/tei_web_scoped_e2e?sslmode=require' \
TEI_WEB_SCOPED_PG_CONFIRM_ISOLATED=tei_web_scoped_e2e \
go test -count=1 -run 'TestScoped.*AgainstIsolatedDatabase' -v ./reference/web-api-contract
```

**DSN未設定でのSKIPは実DB検証の成功ではない。** 実行したexact commit SHA、server_versionとGoバージョンを証拠に残す。

## E11. 既定の並びは非公開列に従う（既知の限界の実証）

`postgres_scoped_inference_integration_test.go`は、**公開していない項目の値が、公開項目だけを選んだ応答から推測できるか**を、実PostgreSQLで確かめる特性化試験である。この試験が固定するのは現在の参照実装の既知の限界であり、望ましい動作の証明ではない。

### 経緯

Issue #1769のコメントは、`NULL AS`による非選択列のマスクについて、「生値の物理取得は抑えるが、既定の`ORDER BY o.ordered_at DESC`が`orderedAt`を公開投影から外しても結果の並びに影響する」ことを未検証の懸念として挙げていた。

この参照では、公開する項目は`orderId`（選択・絞込み・整列）と`customerId`（選択）だけで、`orderedAt`・`status`・`total`は公開していない。呼出し元は、公開していない項目を`$select`にも`$orderby`にも指定できない。

### 実証した内容

公開項目（`orderId`、`customerId`）が全く同じで、公開していない`orderedAt`の大小関係だけが異なる二つのデータセットに、同じ要求（`$select=orderId`）を送る。

| 要求 | データセットA（9が最新） | データセットB（3が最新） |
|---|---|---|
| `$select=orderId`（`$orderby`なし） | `[9, 5, 3]` | `[3, 5, 9]` |
| `$select=orderId&$top=1` | `[9]` | `[3]` |
| `$select=orderId&$orderby=orderId asc` | `[3, 5, 9]` | `[3, 5, 9]` |

- `$orderby`を指定しないと、応答の並びは、公開していない`orderedAt`の降順に従う。呼出し元は、公開項目だけで`orderedAt`の大小関係を知ることができる。`$top=1`だけでも、どの行が最も新しいかが分かる。
- 公開項目で明示的に並べると、非公開項目の違いは応答に現れない。
- 別顧客の行は、どの場合も応答に現れない（行スコープの隔離は破れていない）。

### 実行の記録

実行したexact commit、PostgreSQLのserver_version、Goのバージョンは、この変更のプルリクエストに記録する。DSN未設定ではSKIPし、実証したことにはならない。実行手順はE9-Bと同じ環境変数を使い、`-run TestScopedPostgresDefaultOrder`を指定する。

### 扱い

- 解消は未決である。選択肢は、既定の並びを公開項目に限る、並びに使う項目の権限を要求する、非公開項目に依存した並びを許す条件を契約に明記する、などで、既存のCanonical Queryの意味を変えるため、責務の分担を確認してから決める。
- 解消する変更を入れたら、この試験は、その変更が意図したものであることを確かめたうえで更新する。試験の失敗メッセージにも、その旨を書いている。

### この試験が確かめていないこと

- Tenant用の参照と`from`の経路は、E12で確かめた。
- Count、Aggregate、Expand、複数列の並びなど、ほかの間接推測の経路。

## E12. Tenant用の参照でも並びは非公開列に従う／`from`は呼出し元から渡せない

`postgres_scoped_tenant_inference_integration_test.go`は、E11の二つの未確認点を実PostgreSQLで確かめる特性化試験である。E11と同じく、固定するのは現在の参照実装の既知の限界であり、望ましい動作の証明ではない。

### Tenant用の参照（`ScopedPostgresOrdersReference`）

コード上の予想どおり、既定の`ORDER BY o.ordered_at DESC, o.order_id DESC`は、公開していない`orderedAt`の大小関係を、公開項目だけの応答の並びへ出す。`tenant-a`の注文3・9について、公開項目を変えずに`orderedAt`の大小だけを入れ替えた。

| 要求 | 9が新しい | 3が新しい |
|---|---|---|
| `$select=orderId`（`$orderby`なし） | `[9, 3]` | `[3, 9]` |
| `$select=orderId&$top=1` | `[9]` | `[3]` |
| `$select=orderId&$orderby=orderId asc` | `[3, 9]` | `[3, 9]` |

`tenant-b`の行は、どの場合も応答に現れない（Tenantの隔離は破れていない）。

### `from`による推測は、この参照では成立しない

`from`は`o.ordered_at >= :from`として非公開列への述語になるが、呼出し元は要求で渡せない。要求の入口（`BindRead`）は`$select`・`$filter`・`$orderby`・`$top`以外のキーを拒否し、`$filter`は`orderedAt`を許さない。試験は次の要求がすべて拒否されることを確かめる。`$select=orderedAt`、`$orderby=orderedAt desc`、`$filter=orderedAt ge ...`、`from=...`、`$from=...`、`tenant_id=tenant-b`。`from`は`ExecuteScopedCanonicalRead`の引数としてHostが渡す。

この結論は、入口がこの4つのキーに限られる間だけ成り立つ。`from`などを呼出し元に開く変更を入れるときは、その時点で推測の経路を再評価する。

### 実行の記録とE11と同じ扱い

実行したexact commit、server_version、Goのバージョンは、この変更のプルリクエストに記録する。DSN未設定ではSKIPし、実証したことにはならない。解消は未決で、E11の「扱い」と同じ判断に従う。実行は`-run TestScopedTenantPostgres`を指定する。

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
| Tenantの例 | 注入した`SQLTenantScopeAuthority`から得る文字列`TenantID` | `public.tei_web_scoped_orders_reference` | 偽ConnectorによるSQL/Grant照合に加え、実DB用オプトイン越境テストと非公開列NULLマスクを追加（実行結果は別途確認） |
| Customerの例 | 信頼済み試験入力`PostgresCustomerScope.CustomerID` | 専用DBに作る`public.orders` | 偽Connectorと明示opt-inの実PostgreSQLで行の越境、Order/Top、非選択列のNULLマスクを確認 |

Customer IDをTenant IDや認可Grantとして扱わない。Tenantの例のSQLをCustomerの実DB試験が実行したとは主張しない。二つの実装を明示選択するReferenceとして併置できるが、正式Providerを二種類採用したことにはならない。

製品の行スコープを共通化するには、現在の権限正本からどの値を導出し、Query・Scope ID・Policy Revisionへどう拘束するかを先に定める必要がある。権限の失効と実SQLの原子性、実Host受理、RLS、正式Providerの選択は[#1656](https://github.com/hat47x/tei-platform/issues/1656)と[#1769](https://github.com/hat47x/tei-platform/issues/1769)に残す。
