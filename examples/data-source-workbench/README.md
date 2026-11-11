# 汎用データソース・ワークベンチ用の独立合成Fixture

- **状態：独立Expectedと生入力のみ作成済み。TEI Provider・CLI・Studio・実PostgreSQLでの合否はNOT RUN**。
- 対象計画：[Data Source Workbench設計 §17〜18](../../plan/design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)、Issue #3270。Plugin/CLI/Studio契約Ownerは#1251、実Read Ownerは#2586、独立期待値の資格は#2514。
- **出自**：この例のSQLとJSONLは検証用に新規作成した合成データであり、顧客情報・運用ログ・既存参照モデルの転載ではない。

## 入力

- `postgres_seed.sql`：PostgreSQLの**隔離DB専用**。テーブル1つ、物理主キー `id`、Customerを表す可能性がある文字列、金額、状態。CREATEとINSERTのみ。これが業務Key/Relation/Closed Domainの受理を意味するわけではない。既存スキーマや本番DBへ実行しない。
- `events.jsonl`：改行区切りJSON、5レコード。Event timeとIngest time、順序逆転、重複event_id、明示NULLと属性欠損を含む。ログの追尾/Cloud API等は別Providerで後から対照する。

## 予め固定した独立期待値（入力の構文から確認可能な範囲のみ）

| Case | 独立期待値 | 誤認を防ぐ内容 |
| --- | --- | --- |
| F-01 | SQLのINSERT対象の行は3件（`O-001`、`O-002`、`O-003`）、テーブル名は`sales_order` | SQLファイルを読めても実PostgreSQLに適用・読取成功したわけではない |
| F-02 | `amount`は`120.50`、`0.00`、`5.25`。NUMERICのスケール2を維持 | Go floatの四捨五入結果や文字列化による桁喪失を許容しない |
| F-03 | JSONLには行5件、ユニークevent_idは4種類。重複するのは`evt-B`の2件 | 意図的重複を暗黙の集合化で消さない。どちらを同一Occurrenceとみなすかは未受理 |
| F-04 | `evt-B`のevent_time 09:59:59Zは先に並ぶ`evt-A`の10:00:05Zより古い | 行順/取得順と発生時刻順を同一視しない |
| F-05 | `evt-C.ingest_time`は**明示NULL**、`evt-D.ingest_time`は**属性欠損** | NULLとmissingの違いを埋めない |
| F-06 | `evt-D.message`は属性欠損。SeverityはINFO/WARN/ERRORが存在 | サンプルにある値だけで業務上の完全閉値域を宣言しない |
| F-07 | `order_id`文字列はSQLの`id`の値と一致するものがある | FKや同じ業務Identity/更新関係が受理済みとはみなさず候補に留める |
| F-08 | `postgres_seed.sql`をDBへ適用しない場合でも、JSONLは読取対象になり得る | データソースReaderがDB初期化を必須前提にしない |
| F-09 | JSONLで最後のイベントを読まない範囲なら`O-003`に関する取得結果は部分集合 | 0件や不存在と断定せず、取得範囲・完全性を要する |
| F-10 | 両Sourceを読めても各Sourceの取得時点が不一致なら共通Snapshot保証はない | データ横断のAtomicity/同時点Claimを捏造しない |

## 実Product Query `order_entry.recent_orders` と汎用Fixtureを混同しない

現行[TEI Query宣言](../../reference/workspace/applications/order-entry/queries/recent_orders.tei)が指す[Native SQL](../../reference/workspace/applications/order-entry/queries/recent_orders.sql)は、**`public.orders(order_id, customer_id, ordered_at, status, total_amount)`**を読む。一方、上記の`postgres_seed.sql`は独立した汎用Workbench Fixture **`sales_order(id, customer_id, amount, status)`**であり、前者に必要な`public.orders`・`ordered_at`列・`:from`入力は存在しない。この二つを同じ「3行の実DB Provider Read」として扱うと、最初のFunction縦断試験が実行不能なまま誤って合格となる。

最初のProduct Read専用に**別ファイル**[`recent_orders_product_seed.sql`](recent_orders_product_seed.sql)を追加する。これは**新規／隔離／破棄可能なPostgreSQLのみ**へ適用可能な合成3行であり、元の汎用`sales_order`/F-01〜10と共通業務Meaning、同じDB Instance、同時Snapshotを宣言しない。テーブルを本番・既存参照Workspaceの実DBへ適用しない。

| Query Source-static期待値 | 独立Expected（入力からの固定条件） | ここで合格にしないもの |
| --- | --- | --- |
| PQ-01 Schemaと列 | `public.orders`、選択5列、`ordered_at TIMESTAMPTZ`、`total_amount NUMERIC(12,2)` | Schemaの本番所有、PKの業務Key受理 |
| PQ-02 `:from = 2026-11-07T09:00:00Z` | `ORDER BY ordered_at DESC`なら O-003, O-002, O-001 | 実SQL Executor/実Providerと結果順の資格 |
| PQ-03 `:from = 2026-11-07T10:05:00Z` | 境界一致を含め O-003, O-002 | Providerに実パラメータがBindされた証拠 |
| PQ-04 `:from = 2026-11-07T10:10:01Z` | 完全なFixture上の結果は0件 | 実Readerの返却0件、全Data Sourceでの不存在 |
| PQ-05 Source/Mapping | Query `order_entry.recent_orders`→`datasource sales`→Application `sales_core`宣言、5列のSemantic属性への宣言 | Function Contextへの明示Binding、Runtime Instance/実Grant、Result Mapping成功 |

### 同じQuery IDでもWorkspaceごとに実入力は別

[同名Query分離テスト](query_route_separation_test.go)では、異なる二つのSourceを独立に読み、**`order_entry.recent_orders`というIDの一致だけではSQL/入力/Rowset期待値を共有できない**ことを確認する。

| Sourceと用途 | 実SQL／入力 | 静的に確認できること | 実行前の不足 |
| --- | --- | --- | --- |
| `reference/workspace`＋`recent_orders_product_seed.sql` | `public.orders` 5列、`ordered_at >= :from`、日時降順 | 合成3行に対する`3/2/0`件の独立期待値。R0-Referenceの入力候補 | **Query宣言に`input from`が存在しない**。型・日時精度・Timezone・Runtime Binderが未確定。実PG適用／実Rowsetは未検証 |
| `examples/xtea-lowcode`（[Draft #4076](https://github.com/hat47x/tei-platform/pull/4076)） | `sales.order`から2列、`:status`・`:status_operator`・`:page_size`・`:page_offset`、状態順＋ページング | 宣言された4つの入力とPatternの`eq/ne`・`page-size 20`を別契約として照合 | この例が必要とする`"sales"."order"`の実DB/Readerは、**`public.orders`用Seedとは異なる**。Draftのopt-in実PGテストは未実行 |

Source-staticテストは、`reference/workspace`の`:from`未宣言を**明示的な未解決状態**として固定している。宣言が正しく追加されたときはテストの独立Expectedを審査して更新する。ここでSQL Dialectの完全解析、`order`という識別子の実DB引用規則、Plugin選択、Role/Grant、実Resultを証明したとは扱わない。

**実Readの優先順**：先行する1 Workspaceと実Queryを明示選択→専用の隔離・破棄可能DBに正しいSchema/Seedを適用→独立Readerで入力行を観測→そのWorkspaceでQuery宣言・Parameter Binderを確認→実SQL/Rows→Function Data Context→CLI受領。0件・ページ終端を業務不存在や全母集団の完全性へ無断昇格しない。正式Plugin Providerの選択とHost/Buildの検証は別の[#1251](https://github.com/hat47x/tei-platform/issues/1251)に保持する。

`product_query_fixture_test.go`は**実Query原文／宣言／Applicationと新合成Seedの静的整合**および固定3入力からの期待行順だけを検査する。PostgreSQLに接続せず、Native SQLを実行せず、`sql postgres file`や`datasource sales`が実時点の正しいProviderを選んだことも検証しない。正常0件と部分0件、Grant拒否、`:from`の型・変換・実Bind、同時刻の安定ページング、実Rowset→Function Data Context、CLI/Studio完全受領は[#2586](https://github.com/hat47x/tei-platform/issues/2586)／[#3270](https://github.com/hat47x/tei-platform/issues/3270)の次の独立試験とする。

## 利用条件と次の作業

1. `postgres_seed.sql`は明示的な隔離PostgreSQLのみで実行し、適用後に**別Reader**でSchemaと3行を観測する。SQLファイルをParseしただけなら`F-01`はSource-static資格に留める。
2. JSONL Readerでは元の行順、重複、NULL/欠損、offsetとread範囲、source fingerprintを保持し、Tree/Table/TimelineをFunction内の別Projectionとして検査する。行番号を更新対象の永続Identityとしない。
3. それぞれのProvider／Parser／Projection／CLI/Studio/Go Targetの**実選択・Link・Invoke・完全受領・独立Expected**を別票にする。`events.jsonl`を外部AIサービスへ送出することをProvider読取成功の必須条件にしない。
4. さらに`archive`と`primary`を別DB/異MappingにしたFixtureを追加する場合、#2611 PM-01〜06のOwnerに従い、同一`sales.order`の暗黙の先勝ち選択を拒否する。既存業務意味に採用しない。

**検証状況**：入力ファイルと独立期待値の静的照合に加え、`fixture_test.go`にJSONLの行数・重複・時刻逆転・NULL/欠損の回帰テストを記述した。実行結果は未取得（NOT RUN）。テストがPASSした場合でも検証できるのは**Fixtureの不変条件のみ**であり、実DB上の結果、Provider/Pluginの選択/Link/Invoke、CLI/Studio E2E、独立Result ConformanceのPASSにはならない。
