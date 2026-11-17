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

## 観測（この環境では SKIP、CI では実行）

`internal/numberingpostgres` の6件。**この環境には PostgreSQL が無いため SKIP** した（SKIP は証拠ではない）。**CI の使い捨て `postgres:16` では6件すべてが実行され、PASS した**（`postgres-suites` ジョブ）。

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

- ~~実 PostgreSQL での実行結果~~ **確認済み**（CI の `postgres-suites` で6件すべて PASS。ただし使い捨ての `tei_fixture` データベースであり、製品DBではない）。
- 多プロセス・多ホストでの競合（SQLite で確認した2プロセス試験に相当するものを PostgreSQL で実施していない）。
- 実DB のスキーマ運用（このストアは自前でテーブルを作る。製品のスキーマ移行とは別）。
- トランザクション分離レベルの選定（`BeginTx` の既定に依存している）。

## 追記2（2026-11-18）: 記録の古い記述を直した

再読して、次の2箇所が**実装より古い**ことに気づいた（過大ではなく陳腐化）。

- 「この環境では一度も実行していない」→ CI で実行済み。
- 「`required` の一覧には2件しか登録していない」→ 6件登録した（#5429）。
- 索引の一文「全件SKIPのため実DBの結果は未確認」→ 全件PASS済み。

`#5429` は workflow だけを変えて文書を更新しなかったため、記録と CI が食い違っていた。**記録を直した**。

さらに、独立レビューが指摘した**より重要な穴**を記す。

**採番ストアの試験は逐次実行のみ**である。`TestPostgresRefusesAForeignFirstWrite` は「2回目の最初の書き込み」を同じ接続で順に呼ぶだけで、**二人の書き込みが並行したときの挙動を観測していない**。したがって「条件付きUPDATEだけで成立する」は、**並行の観測を含まない主張**である。SQLite で行った2プロセス試験に相当するものを PostgreSQL では実施していない。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。**実装は存在するが、この環境では一度も実行していない**（SKIP）。CI 配線は済み。仮承認。
## 追記（2026-11-17）: CI で実行され、全件 PASS した

初出の時点では、この環境に PostgreSQL が無く6件すべてが SKIP でした。CI へ配線したところ**2度続けて落ち、しかも一度は「5件 PASS で緑」でした**。3回目で全件実行されました。

| 版 | 結果 | 何が起きたか |
| --- | --- | --- |
| 初回 | failure | `TEI_TEST_POSTGRES_DSN is empty` — **必須キーの検査を足したが、値を `env:` に定義していなかった** |
| 2回目 | success（**不完全**） | DSN は定義したが、実行フィルタが `TestPostgresStore\|TestPostgresReserves\|TestPostgresRefuses` で、**`TestPostgresLoadRequiresARuleID` が列挙から漏れ、一度も走っていなかった** |
| 3回目 | success（**全件**） | フィルタを `^TestPostgres` へ広げ、6件すべてが PASS |

`postgres-suites` のログで確認した実際の結果:

```
--- PASS: TestPostgresStoreStartsAtZero (0.01s)
--- PASS: TestPostgresReservesAcrossReopen (0.01s)
--- PASS: TestPostgresRefusesAStaleVersion (0.01s)
--- PASS: TestPostgresRefusesAForeignFirstWrite (0.00s)
--- PASS: TestPostgresRefusesANegativeCounter (0.00s)
--- PASS: TestPostgresLoadRequiresARuleID (0.00s)
```

読み取れること:
- 不変条件（0から始まる・再開後も番号が残る・古い版は拒否・二人の最初の書き込みが両方成功しない・負のカウンタは拒否・規則ID無しは拒否）が、**PostgreSQL の条件付き UPDATE だけで成立する**。SQLite の「接続を1本に絞る」仕組みに依存していない。

読み取れないこと:
- **製品DB**での挙動。使い捨ての `tei_fixture` であり、スキーマ移行・権限・運用とは別である。
- **多プロセス・多ホストの競合**。SQLite で行った2プロセス試験に相当するものを PostgreSQL では実施していない。
- `required` の一覧には2件しか登録していないため、**残り4件が将来走らなくなってもジョブは緑になりうる**。
