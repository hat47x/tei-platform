# 採番の PostgreSQL 実装（2026-11-17）

## 対象

- 実装: `internal/numberingpostgres/store.go`（`Open`・`Store.Load`・`Store.Save`・`quoteIdentifier`）
- 試験: `internal/numberingpostgres/store_test.go`
- CI: `.github/workflows/tei-postgres-unrun-suites-ci.yml`（使い捨て `postgres:16` で実行）

## これまでの状態

採番の永続格納は SQLite だけで確認していた（2プロセス同時実行で同一番号を出さない、カウンタが発行数に一致する）。PostgreSQL は未実装で、設計文書の残り表にも「PostgreSQLを先にするか、多プロセス構成を広げるか」とあった。

## 実装

`numbering.CounterStore`（`Load`・`Save`）を PostgreSQL で実装した。**不変条件は SQLite と同じ、仕組みは違う。**

| | SQLite | PostgreSQL |
| --- | --- | --- |
| 二人の書き手を分ける手段 | `_txlock=immediate` ＋ 単一接続 ＋ `busy_timeout` | **`WHERE version = ?` の条件だけ** |
| 最初の書き込みの競合 | INSERT が主キーで拒否 | INSERT ... ON CONFLICT DO NOTHING が 0 行を返す |
| 同時性の担保 | 接続を1本に絞る | 接続プールのままでよい |

**SQLite の安全性は「接続を1本に絞る」ことに依存していた**ので、そのまま PostgreSQL へ持ち込む必要はない。条件付き UPDATE だけで足りる。だから `SetMaxOpenConns(1)` は置いていない。

## 観測（DSN 未設定時は SKIP）

`internal/numberingpostgres` の6件。**この環境には PostgreSQL が無いため全て SKIP** した（SKIP は証拠ではない）。

| 試験 | 確かめること |
| --- | --- |
| `TestPostgresStoreStartsAtZero` | 行の無い規則は 0・版 0 |
| `TestPostgresReservesAcrossReopen` | 発行した番号が再接続後も残り、次は 0002 |
| `TestPostgresRefusesAStaleVersion` | 古い版は拒否され、カウンタも変わらない |
| `TestPostgresRefusesAForeignFirstWrite` | 二人の「最初の書き込み」が両方成功しない |
| `TestPostgresRefusesANegativeCounter` | 負のカウンタは DB へ届く前に拒否 |
| `TestPostgresLoadRequiresARuleID` | 規則 ID 無しは拒否 |

## CI へ配線した

既存の使い捨て PostgreSQL ジョブ（`tei-postgres-unrun-suites-ci.yml`）へ加えた。

- `TEI_TEST_POSTGRES_DSN` を必須キーの一覧へ追加（空なら「スイートが skip する」と落とす）。
- `go test ... ./internal/numberingpostgres` を実行。
- `required` の一覧へ `TestPostgresStoreStartsAtZero`・`TestPostgresRefusesAStaleVersion` を追加（名前を書くことで、rename や呼出し削除で「何も走らず緑」にならないようにする）。
- YAML はパースできることを確認。

## 読み取れること

- 採番の永続格納が **SQLite と PostgreSQL の両方**で同じ契約（`CounterStore`）を満たす実装を持つ。
- 同時実行の担保が**別の仕組み**（接続制限ではなく条件付き UPDATE）であることを、コードと試験の両方で区別している。

## 読み取れないこと

- **実 PostgreSQL での実行結果**。この環境に PostgreSQL が無く、6件すべて SKIP した。**CI での結果は未確認**（このPRのマージ後に確認する必要がある）。
- 多プロセス・多ホストでの競合（SQLite で確認した2プロセス試験に相当するものを PostgreSQL で実施していない）。
- 実DB のスキーマ運用（このストアは自前でテーブルを作る。製品のスキーマ移行とは別）。
- トランザクション分離レベルの選定（`BeginTx` の既定に依存している）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。**実装は存在するが、この環境では一度も実行していない**（SKIP）。CI 配線は済み。仮承認。
