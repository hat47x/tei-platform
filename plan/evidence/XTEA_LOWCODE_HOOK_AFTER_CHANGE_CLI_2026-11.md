# CLIから`after-*`フックへ到達する配線の確認（2026-11-10）

## 対象

- 実装: `cmd/tei/cli_view.go`（`-reference-store`、二巡目のフック実行）、`cmd/tei/cli_view_hooks.go`（`runDeclaredHooksHalf`・`hookHalf`・`mergeHookReports`）、`cmd/tei/cli_flag_contract.go`
- 試験: `cmd/tei/cli_view_hooks_half_test.go`
- 対象版: `208c02b3e`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 前段で残っていたこと

前段（[`after-*`へ到達する実行経路](XTEA_LOWCODE_HOOK_AFTER_CHANGE_2026-11.md)）で、フック層に`RunBeforeChange`・`RunAfterChange`・`AppliedChangeReach`を入れました。ただし**ホストの配線は未着手**で、CLIもStudioも`ReferenceReach`と`RunReachable`のままでした。

## 配線の方針

CLIの`view -execute`は「何も書かない参照実行」と表明しており（`persisted:false`・`persistence:not-applicable`）、既定の挙動を適用する経路へ変えるのは製品の意味の変更になります。そこで**明示的な任意指定のフラグ**を足し、既定は変えないことにしました。

`-reference-store`（`-execute`と併用）:

- 宣言された効果を**メモリ上の参照Store**（`command.NewMemoryStore`）へ適用する。
- フックを**変更の前後で二巡**する: `RunBeforeChange` → コマンド実行（適用）→ `RunAfterChange`＋`AppliedChangeReach`。
- Storeは実行ごとに作り、終了時に捨てる。**永続ではない**（`persisted:false`・`persistence:not-applicable`は変えない）。

`-execute`を伴わない`-reference-store`は引数の誤りとして`INVALID_ARGUMENT`（終了コード2）で拒否します。実行の中身を変えるフラグを、実行しない呼出しで黙って無視しないためです。

## 実測

同じfixture（`sales.create_order`、`before-create`と`after-create`のフック）で:

| 呼出し | `hooks.executed` |
| --- | --- |
| `-execute`（従来） | `["before-create"]`、`after-create`は`"the reference run writes nothing..."`でskip |
| `-execute -reference-store` | `["before-create","after-create"]` |
| `-reference-store`単独 | `INVALID_ARGUMENT`、終了コード2 |

`-execute`単独の挙動は変わっていません。`-reference-store`を付けた場合も`persisted:false`・`persistence:"not-applicable"`のままで、Storeが永続でないことを応答が述べています。

`update`効果を空のStoreへ適用しようとすると`record-missing`で拒否されます（適用先が無いので正しい）。

## 試験（4件）

- `TestRunDeclaredHooksHalfSplitsAroundTheChange` — 前半は`before-update`だけを実行し、後半をskippedとして報告しない。後半は`after-update`を実行する。
- `TestRunDeclaredHooksAfterHalfStaysSkippedWithoutAChange` — `ReferenceReach`では後半は理由つきでskipされ、実行されない。
- `TestMergeHookReportsKeepsBothHalves` — 併合した報告が両方を実行順に持つ。
- `TestMergeHookReportsHandlesOneMissingHalf` — 片方が無い場合にpanicも空報告もしない。

## 実行

```powershell
go test ./cmd/tei -run 'TestRunDeclaredHooksHalf|TestRunDeclaredHooksAfterHalf|TestMergeHookReports' -count=1
go test ./internal/hook ./internal/command ./internal/studiohost ./internal/validation -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./cmd/tei ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 読み取れること

- CLIから、変更を適用した後の`after-*`フックを実行できる（明示指定した場合だけ）。
- 既定の参照実行は変わらず、`after-*`は理由つきでskipされる。

## 読み取れないこと

- **Studio側の配線**。`internal/studiohost/hooks.go`は`ReferenceReach`と`RunReachable`のままで、`after-*`へは到達しません。
- 実Provider・実DBへの適用。Storeはメモリ上だけで、`persistence`は`not-applicable`のままです。
- `after-read`の位置づけ（`isBeforeChange`は`before-read`だけを前半とする）。
- 効果適用の取り消し（補償）。`-reference-store`は適用してからafter-*を走らせるだけで、after-*が拒否しても適用を戻しません。この点は未実装です。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「ホストから`after-*`経路への配線」を、CLIの明示指定フラグとして繋いだ記録です。Studio配線と、after-*拒否時の補償は未着手のまま残ります。
