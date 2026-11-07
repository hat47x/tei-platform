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

## 利用条件と次の作業

1. `postgres_seed.sql`は明示的な隔離PostgreSQLのみで実行し、適用後に**別Reader**でSchemaと3行を観測する。SQLファイルをParseしただけなら`F-01`はSource-static資格に留める。
2. JSONL Readerでは元の行順、重複、NULL/欠損、offsetとread範囲、source fingerprintを保持し、Tree/Table/TimelineをFunction内の別Projectionとして検査する。行番号を更新対象の永続Identityとしない。
3. それぞれのProvider／Parser／Projection／CLI/Studio/Go Targetの**実選択・Link・Invoke・完全受領・独立Expected**を別票にする。`events.jsonl`を外部AIサービスへ送出することをProvider読取成功の必須条件にしない。
4. さらに`archive`と`primary`を別DB/異MappingにしたFixtureを追加する場合、#2611 PM-01〜06のOwnerに従い、同一`sales.order`の暗黙の先勝ち選択を拒否する。既存業務意味に採用しない。

**検証状況**：本PRで確認するのは入力ファイル構造とExpectedの手動・静的照合のみ。実DB上の結果、Go Test、Pluginの選択/Link/Invoke、CLI/Studio E2E、独立Result Conformanceは未実施。
