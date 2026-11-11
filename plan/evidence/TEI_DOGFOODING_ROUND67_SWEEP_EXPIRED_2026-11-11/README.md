# SQLite の sweep が「期限切れを返さない」断続失敗を、単一文化で解消（2026-11-11、第67ラウンド）

- 対象: [#4252](https://github.com/hat47x/tei-platform/issues/4252)（第66ラウンドで起票）。
- 前段: [第66ラウンド](../TEI_DOGFOODING_ROUND66_CLI_TEST_TIME_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **原因を特定し、`SweepExpired` を単一文にした。再現 3/3 → 0/8。**

## 1. 目的

第66ラウンドで起票した flake は、CI のログに `SQLITE_BUSY` が出ていたため「ロック競合」と推定した。**本ラウンドで実際に再現したところ、別の原因だった。** 推定を実測で訂正する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `29b321464`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`、`modernc.org/sqlite v1.34.5` |

## 3. 再現（推定の訂正）

| 試行 | 結果 |
| --- | --- |
| 素の `go test` 40 回 | **0/40 失敗**（再現しない） |
| `TestParallelClaim` 単独 50 回 | ok（14.99s） |
| **`-count=5`（1 プロセスで 5 回）** | **FAIL** |
| `-count=10` を 3 並列 | **3/3 FAIL** |

**再現条件は「同一プロセス内での繰り返し」だった。** CI のログに見えた `SQLITE_BUSY` は別の run のもので、私が最初に読んだ行は `TestParallelClaim` のものだったが、**実際に再現したのは `TestSweepExpired` の別の失敗**だった。

```text
--- FAIL: TestSweepExpired (0.05s)
    lease_test.go:76: SweepExpired len = 0, want 1
```

## 4. 原因

```go
_, err := s.db.ExecContext(ctx, `UPDATE claims SET state='expired', updated_at=datetime('now') WHERE ...`)
rows, err := s.db.QueryContext(ctx, `SELECT ... WHERE state='expired' AND updated_at = datetime('now')`)
```

**`datetime('now')` は秒単位**である。UPDATE と SELECT の間で秒が変わる（または同じ秒のうちに別の sweep が走る）と:

- **秒が変わった場合**: UPDATE した行の `updated_at` は前の秒、SELECT は新しい秒を探すので **0 行**。作業は成功しているのに「何も期限切れでない」と答える。
- **同じ秒に 2 回目**: 1 回目が期限切れにした行を、2 回目が**もう一度**返す。

**2 つの文に分かれていること自体が誤り**だった。`updated_at` を時計として照合する設計が、秒の境界で壊れる。

## 5. 修正

**単一の `UPDATE ... RETURNING` にした。**

```sql
UPDATE claims SET state='expired', updated_at=datetime('now')
WHERE state IN ('working','review','blocked') AND lease_expires < ?
RETURNING issue_number, role, model_id, harness, session_id, lease_expires,
          heartbeat_at, state, base_sha, head_sha, pr_number
```

**返す値は「この文が期限切れにした集合」そのもの**になり、2 回目の照合が不要になる。`rows.Err()` も確認するようにした（単一文になったので）。

## 6. 試験

`tools/reference-external-role-dispatcher/internal/sweep_expired_test.go`:

| 試験 | 内容 |
| --- | --- |
| `TestSweepExpiredReportsOnlyWhatItExpired` | 1 回目は 1 件、**同じ秒での 2 回目は 0 件**（旧実装では 1 件返る） |
| `TestSweepExpiredLeavesActiveClaimsAlone` | 有効な claim は sweep されず `working` のまま |

### 6.1 変異試験

**旧実装（2 文）に戻すと、10 回反復で FAIL する。**

```text
--- FAIL: TestSweepExpiredReportsOnlyWhatItExpired
    second sweep = [{... State:expired ...}], want none:
    a sweep must report only newly expired claims
```

## 7. 安定性の測定

| 実行 | 修正前 | 修正後 |
| --- | --- | --- |
| `-count=5` を 3 回 | **3/3 FAIL** | — |
| `-count=5` を 8 回 | — | **0/8 FAIL** |

`TestParallelClaim`（`SQLITE_BUSY`）は 50 回単独で通り、**`busy_timeout` と `SetMaxOpenConns(1)` は既に適切に設定されていた**（`lease.go` 43–67 行）。**ロック競合は原因ではなかった。**

## 8. 回帰確認

`go build ./tools/reference-external-role-dispatcher/...` が通り、パッケージの全テストが ok。

## 9. #4252 に書いた4案との関係

| 案 | 判定 |
| --- | --- |
| A `busy_timeout` を設定 | **既に設定済みだった**（不要） |
| B `SQLITE_BUSY` の再試行 | **原因が違う**（不要） |
| C テストの直列化 | **不要**（並行性は壊れていない） |
| D WAL モード | **既に設定済み** |

**4案はいずれも当てはまらなかった。** 起票時に「A か B が筋」と書いたが、**実測すると原因は別**だった。Issue を訂正する。

## 10. 読み取れないこと（NOT RUN）

- CI 上での再現率（ローカルのみ測定）。
- 他の `datetime('now')` 照合パターンが同じ脆弱性を持つか（未調査）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 11. 位置付け

**起票時の推定が間違っていたことを、実測で訂正した回**である。CI のログに `SQLITE_BUSY` があったため「ロック競合」と判断し、4案を書いた。実際に再現すると、原因は**秒単位の時刻照合**で、ロック競合ではなかった。

**再現条件（同一プロセス内の反復）を特定できたことが決定的だった。** 素の 40 回では 0 回、`-count=5` では 3/3 失敗。CI の `./...` は 1 プロセスで各パッケージを 1 回しか走らせないので、**ローカルで同じ形を作らないと出ない** flake だった。
