# 行単位フックの`uses`を二つの入口で同じに判定する確認（2026-11-10）

## 対象

- 実装: `internal/hook/rows_uses.go`（`refuseRowScopeWithUses`・`CodeRowScopeUsesUnsupported`）、`internal/hook/rows.go`、`internal/hook/reach_rows.go`
- 試験: `internal/hook/rows_uses_parity_test.go`、`internal/hook/rows_uses_test.go`
- 対象版: `2fdca879c`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 見つけた欠陥

同じ宣言が、入口によって**成功も失敗もする**状態でした。

| 入口 | 経路 | 結果 |
| --- | --- | --- |
| `Run`（`Runner.Rows`あり） | `RunRows` → `RunWithDeclaredValues` → `declaredCaller` | **成功**（能力は呼ばれる） |
| `RunReachable`（`Runner.Rows`あり） | `applyReachableRows` → 自前の拒否 | **失敗**（`HOOK_SCOPE_ROW_USES_UNSUPPORTED`） |

`scope row` + `uses` という一つの宣言が、`Run` では通り、`RunReachable` では拒否されていました。

## 実測（修正前）

```text
--- FAIL: TestRowScopeWithUsesIsJudgedTheSameByRunAndRunReachable
    the same declaration got two answers: Run err=<nil>,
    RunReachable err=HOOK_SCOPE_ROW_USES_UNSUPPORTED: hook before-update
    declares scope row with uses, which this path does not call per row
```

## どちらへ寄せたか（判断）

**拒否（fail-closed）に寄せました。** 理由:

- `uses` は「その能力を呼ぶ」という約束です。レコード経路は `RunDeclaredCalls` で位相ごとに一度だけ果たします。
- 行ごとに果たす実装はありません。1ページ分を1回だけ呼ぶと、**どの行の約束も果たしていない**のに「果たした」ように見えます。
- 呼ばないまま位相を実行済みにすると、X-TEAの「宣言は在るが実行時に何もしない」（知識目録D-9）と同じ状態になります。

したがって `RunRows` 側を、`applyReachableRows` と同じ拒否へ揃えました。

## 変更

- 判定を`refuseRowScopeWithUses`一つにまとめ、両方の行入口がそれを呼びます。**二箇所に同じ判断を書いたことが食い違いの原因**だったので、判断を一箇所にしました。
- 符号`CodeRowScopeUsesUnsupported`を定数にしました（片方だけ文字列で持っていた）。
- 試験を追加:
  - `TestRowScopeWithUsesIsJudgedTheSameByRunAndRunReachable`: 両入口の答えが一致すること。
  - `TestRunRowsRefusesUses`: `RunRows`が拒否し、実装が一度も呼ばれないこと。
  - `TestRunRowsWithoutUsesStillRuns`: `uses`を宣言しない行単位フックは従来どおり行ごとに動くこと。

## 実行

```powershell
go test ./internal/hook ./internal/command ./internal/studiohost ./internal/validation -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRowScopeWithUsesIsJudgedTheSameByRunAndRunReachable` 成功（修正前は失敗）。
- `TestRunRowsRefusesUses` 成功。`RunRows`が`HOOK_SCOPE_ROW_USES_UNSUPPORTED`を返し、実装の呼出しは0回。
- `TestRunRowsWithoutUsesStillRuns` 成功。2行に2回。
- 既存の行経路の試験（`TestRunReachableRefusesRowScopeWithUses`・`TestRunReachableAppliesRowScopeToEveryRow`・`TestRunRoutesARowScopeToRunRows` など）はすべて`ok`。
- `internal/hook`・`internal/command`・`internal/studiohost`・`internal/validation`は`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`は`ok`。

## 読み取れること

- `scope row` + `uses` は、どちらの入口でも同じく拒否される。
- 行単位フックの`uses`の約束は、現時点では果たさない（果たせないので拒否する）。

## 読み取れないこと

- **行ごとに`uses`を果たす実装**。行ごとに入力を組み立てて能力を呼ぶ経路は未着手です。今回の変更は「果たせない約束を、果たしたように見せない」ところまでです。
- ホストが`Runner.Rows`を設定する経路（`cmd`・`internal/studiohost`ともに0件）。CLI・Studioの参照実行は行集合を渡さないため、`row`は理由つきでskipされます。
- `header-detail`の`row`／`header`、実Pluginの行単位フック。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。同じ宣言が入口で別の答えになる欠陥を、判断を一箇所へ寄せて直した記録です。行ごとの`uses`実行は未着手のまま残ります。
