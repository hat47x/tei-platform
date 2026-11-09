# reference-order-entry-persistence

`order_entry`の注文を実PostgreSQLへ書き込み、**別のprocess**が読み戻すことを確かめる参照ツールです（[#1127](https://github.com/hat47x/tei-platform/issues/1127)）。クリティカルパス（[TEI_PRODUCT_CRITICAL_PATH_2026-10.md](../../plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) §31）のうち、正式Accepted R1から実DBへ保存し、別invocationで読む一本の縦切りの、DB側を扱います。

## 何を確かめるか

一回の試行を四つのprocessに分けます。DBへ接続するmodeは、専用の空PostgreSQL（`TEI_TEST_POSTGRES_DSN`または`_FILE`）と、cleanなリポジトリのrevisionを要求します。

| mode | 内容 |
| --- | --- |
| `prepare` | 専用の空DBであることを確認し、`order_entry`の**Persistence Mappingから導出したDDL**で`public.orders`・`public.order_lines`を作る。所有印を付け、pendingの注文1001と明細2行を入れる |
| `write` | Source revisionと物理schemaを**別々の段階で**確かめ、解決済みの`sales.order.submit` BindingをDriver経由で実行する。書込み直前に許可を確認し、注文をpendingからsubmittedに更新する |
| `read` | write とは**別のprocess・別のDB接続**で注文を読み、Commandの成功応答とは別に観測状態を判定する |
| `cleanup` | 所有印が一致するときだけfixtureを削除し、DBが空に戻ったことを確かめる |
| `join` | writeとreadの二つのreceiptが、同じ一回の試行（Command ID・注文key・Source revision・repository revision）で、別processから得たものかを判定する |

観測状態は`observed`・`read-failed`・`mismatch`・`command-not-committed`・`source-revision-mismatch`です。Commandが成功しても、読取りが失敗・不一致・別Sourceなら`observed`にしません（N11）。

## 拒否する場合

- 物理schemaに差がある（S02の物理側）。Source revisionの不一致（S02のSource側）とは別の理由で、書込み前に拒否します。
- 書込み直前の許可が失われている（S05）。更新は行われず、行は変わりません。
- 接続先がprepareで作った所有印のfixtureでない。
- 手書きoverlayの証拠（`reference-data-first-persistence`のreceipt）や、別Application・別Source・同一processのreceipt。`join`は結合しません。

## 実行

```bash
export TEI_TEST_POSTGRES_DSN='postgres://…/専用の空DB'   # 空で専用のDBだけ
go build -o oe-persist ./tools/reference-order-entry-persistence
./oe-persist -mode prepare > prepare.json
./oe-persist -mode write   -prepare-receipt prepare.json > write.json
./oe-persist -mode read    -write-receipt write.json > read.json
./oe-persist -mode join    -write-receipt write.json -read-receipt read.json
./oe-persist -mode cleanup -prepare-receipt prepare.json > cleanup.json
```

CIは[tei-order-entry-postgres-ci.yml](../../.github/workflows/tei-order-entry-postgres-ci.yml)で、`postgres:16-alpine`に対して上記を別processとして実行します。DB不要の試験（`go test ./tools/reference-order-entry-persistence`）は、DSNが無い場合に実PostgreSQLの試験を`NOT_RUN`で飛ばします。

## 主張しないこと

- **Sourceは合成**で、人的Acceptanceではありません。Host Current Accepted Source Readerも使っていません（#267）。
- **書込み直前の許可は合成**（常にcurrent）で、実Hostの現在の効果許可ではありません。許可が失われたときの拒否は試験で確かめます。
- **通常のWeb経路は通りません。** 同じ注文を`reference-first-web-target`の`referenceCommandScenarioRunner`が扱いますが、そちらのDB書込みは任意のstoreを差し込めるだけで、Observationは合成のままです。
- DDLはPersistence Mappingから導出しますが、**CRUDは生成していません**。列名を固定したseedとUPDATEは手書きです。
- 生成した物理schemaの移行（Source変更に伴うmigration）、Product E2、本番配備は対象外です。
- 使い捨てのPostgreSQL fixtureに対する確認であり、他のProvider・Targetの成功を示しません。
