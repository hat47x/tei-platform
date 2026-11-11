# main が赤かった: `TestParallelClaim` は毎回失敗していた（2026-11-11）

- 対象: [`tools/reference-external-role-dispatcher/internal/lease.go`](../../../tools/reference-external-role-dispatcher/internal/lease.go) と `TestParallelClaim`
- 位置づけ: **製品コード（参照ツール）の不具合修正と実行記録**。
- 計測した版: `raw/head.txt`（`origin/main` `b10e53b5` の作業ツリー＋このPRの変更）。

私の前のPR（#4245）の CI が赤になりました。原因は**私の変更ではなく、main に既に入っていた別のPR（#4218）**でした。**clean main の別 worktree で再現**して確認し、そのまま直しました。

## 1. 何が起きていたか

`TestParallelClaim` は**2つの goroutine が同じ issue を同時に claim**し、**勝者が1つ、敗者が `ErrAlreadyClaimed`** になることを要求します。

clean main で5回連続実行した結果、**5回とも失敗**しました。

```text
--- FAIL: TestParallelClaim (0.04s)
    lease_test.go:133: unexpected error from worker 0: clear terminal claims: database is locked (5) (SQLITE_BUSY)
```

**敗者が `ErrAlreadyClaimed` ではなく `SQLITE_BUSY` を受け取っていました。** これは flake ではなく**毎回**です。

## 2. 原因

`ClaimIssue` は `BEGIN` して `DELETE` と `SELECT` と `INSERT` を行います。2つの接続が同時に読取りロックを取り、**書き込みロックへ昇格しようとして衝突**します。SQLite は待たずに `SQLITE_BUSY` を返します。

```text
db.SetMaxOpenConns(1)              … 接続を1本にして直列化する
PRAGMA busy_timeout=5000           … 負けた側が勝者の commit を待つ
```

**どちらも欠けていました。**

## 3. 修正

`NewLeaseStore` に2つ足しました。

```go
db.SetMaxOpenConns(1)
if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil { ... }
```

**呼出し側は変えていません**（`cmd/dispatcher` も試験も同じ）。

## 4. 観測

### 修正後（5回連続）

```text
run 1: ok 0.057s    run 2: ok 0.049s    run 3: ok 0.051s
run 4: ok 0.048s    run 5: ok 0.076s
```

### 修正を戻したとき

```text
removed the single connection and the busy timeout
--- FAIL: TestParallelClaim (0.05s)
    lease_test.go:133: unexpected error from worker 0: clear terminal claims: database is locked (5) (SQLITE_BUSY)
FAIL
restored lease.go
```

**2つを外すと、同じ失敗が再現します。** この2つが原因であることを実測しました。

## 5. この記録が示すこと

- **main は私の変更と無関係に赤かった。** clean main の別 worktree で5回とも再現したので、私の差分のせいではありません。
- **「flaky だから再実行」で済ませると、この不具合は残ります。** 5回とも失敗するので、再実行では直りません。
- **修正は store の性質（接続数と待ち時間）で、呼出し側の責務ではありません。** 直列化と待機は、lease という排他機構の前提です。

## 6. この記録が示さないこと

- **`busy_timeout=5000` が十分か。** 5秒を超える保持があると、まだ `SQLITE_BUSY` になりえます。**PoC の範囲では足りますが、上限の根拠は測っていません。**
- **`Release`・`Heartbeat` の同時実行**。同じ store を使うので単一接続で直列化されますが、**それらの競合を試験していません**。
- **本番の Dispatcher 運用**（複数プロセスが同じ DB を開く場合）。この修正は1プロセス内の接続を直列化しますが、**プロセス間の排他は busy_timeout に依存**します。
- プラグイン基盤本体（`internal/plugin`）への影響。**この修正はプラグイン基盤とは独立**です。

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（修正後の5回と、修正を戻したときの失敗） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/after.txt`・`raw/after-1..5.txt` | 修正後の5回 |
| `raw/before.txt` | 修正を戻したときの失敗 |

## 8. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `797bf953ca86a0922c60fd31bd63478615b8177d0bcb13228f5e23c361aa5ba0` | 2428 |
| `raw/head.txt` | `bff092477edffc3ec9306afbf1b13119f3b8eba41c60d9536b3117d4cbf06308` | 73 |
| `raw/pre-status.txt` | `1b19bdb569cab490d255e5d3b446120a215c1c856c512fea54b9471ce7c471ba` | 86 |
| `raw/after.txt` | `926d9355d52c635318fe411a86e8bcef3f075d32e8b9cfab4352916dbabaf7a3` | 365 |
| `raw/after-1.txt` | `f497071ce5f51287602e607c05d5d5165fadc121506205705671cf770e64aa05` | 66 |
| `raw/after-2.txt` | `f38fc82d267322b6ea4d12c938a14c966077aabf3329f0dedc5a22bf4cded072` | 66 |
| `raw/after-3.txt` | `dc609a13147076c6f3131c73210083925d51e738a40439e6e6bfa8d2126a26d5` | 66 |
| `raw/after-4.txt` | `e5c169f961458ec922f8a4d6c2876b913c2e598e2c463f7bdb997d808fbbcbeb` | 66 |
| `raw/after-5.txt` | `de633aa7b3351b39b3b6e2b7f547fb7557e4507cf19897cff92b4007144a74e5` | 66 |
| `raw/before.txt` | `1ec9c5f97e9c2b89ae6b65f2b0e408cd62ba6fd5b2cdb11e1e970a026af7fd7b` | 296 |
