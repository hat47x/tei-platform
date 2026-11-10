# フックが宣言した能力を呼ばないまま「実行した」と報告しない確認（2026-11-09）

## 対象

- 実装: `internal/hook/reach.go`（`RunReachable`が`uses`を扱う、`PhaseReport.Uncalled`、`ReasonCallerUnavailable`・`ReasonDeclaredCallNotMade`）、`cmd/tei/cli_view.go`と`cmd/tei-studio/assets/view-format.mjs`（未実行の呼出しを表示）
- 試験: `internal/hook/reach_uses_test.go`、`internal/studiohost/hooks_run_test.go`、`cmd/tei/cli_view_hooks_text_test.go`、`cmd/tei-studio/test/view-format.test.mjs`
- 対象版: `34783baf3`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、Node v22

## 見つけた欠陥

CLIとStudioの参照実行が使う`hook.RunReachable`は、フックの`uses`（呼ぶ能力の宣言）を**見ていませんでした**。

- `Runner.Caller`は両ホストとも設定していない。
- `hook.CommandCaller`と`hook.Guarded`は`internal/hook`に単体試験つきで存在するが、呼ばれていない。
- それでも、位相は`executed`として報告されていた。

つまり、`hook before-update { reads … uses sales.submit_order }`と宣言したフックは、能力を一度も呼ばずに「実行された」と報告されていました。検証（`APPLICATION_PATTERN_INTERACTION_HOOK_USE_WITHOUT_COMMAND`）は「`uses`の能力に`command` Patternが無ければ拒否する」としており、呼べるはずの能力が呼ばれないまま合格と読める状態は、宣言と実行の食い違いです。

## 変更の要点

- `RunReachable`が、`uses`を宣言した位相を**`executed`として報告しない**ようにしました。代わりに:
  - `NotEvaluated`と`Skipped`に入れる。
  - `PhaseReport.Uncalled`に、呼ばれなかった能力を位相ごとに記録する（宣言順を保つ）。
  - 理由を、「呼出し元が無い」（`ReasonCallerUnavailable`）と「呼出し元はあるが実装が呼ばない」（`ReasonDeclaredCallNotMade`）で分ける。
- CLIのテキストに`    uncalled <capability>`、Studioの画面要約に`uncalled at <phase> <capability>`を足しました。
- 読み取りだけを宣言したフックは、これまでどおり`executed`になります。新しい拒否が通常の場合を飲み込まないことを試験で固定しました。
- 試験を6件追加（Go 5件、Node 1件）。Studioのfixtureを二つに分け、`uses`のある形とない形を別々に確かめられるようにしました。

## 実行

```powershell
go test ./internal/hook ./internal/studiohost ./internal/command -count=1
go test ./cmd/tei -count=1 -timeout 30m
node --test cmd/tei-studio/test/view-format.test.mjs
go vet ./internal/hook ./cmd/tei
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRunReachableDoesNotClaimADeclaredCallItDidNotMake` 成功。`uses`を宣言した位相は`executed`に入らず、`NotEvaluated`に入り、`Uncalled["before-update"]`が`sales.submit_order`を名指す。修正前は`executed`に入っていた。
- `TestRunReachableNamesTheCallWhenACallerExistsButWasUnused` 成功。呼出し元があっても実装が呼ばなければ理由が変わる。
- `TestRunReachableStillRunsAHookThatDeclaresNoCall` 成功。`reads`だけのフックは`executed`のまま。
- `TestRunReachableUncalledKeepsTheDeclarationOrder` 成功。宣言順が保たれる。
- `TestExecuteCommandDoesNotClaimADeclaredCallItDidNotMake` 成功（Studio）。`uncalled`に`sales.submit`が入る。
- `TestWriteHookReportTextNamesAnUncalledDeclaration` 成功（CLI）。
- Nodeの新規テスト成功。`hooks`行に`uncalled at before-update sales.submit_order`が出る（計10件）。
- 既存の`internal/hook`・`internal/studiohost`・`internal/command`・`cmd/tei`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- `uses`を宣言したフックは、能力が実際に呼ばれない限り「実行された」と報告されない。
- 呼ばれなかった能力は、CLI・Studio・JSONの三つで名指しされる。

## 読み取れないこと

- **フックから能力を実際に呼ぶ経路そのもの**。`hook.CommandCaller`と`hook.Guarded`は存在するが、両ホストはまだ`Runner.Caller`を渡していません。本変更は「呼んでいないのに呼んだと書かない」までで、「呼べるようにする」は別の作業です。
- 実Pluginからの供給路。
- 実データの読取り。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「実Pluginからの供給路」に隣接する、**宣言と実行の食い違い**を塞いだ記録です。能力を実際に呼ぶ配線は未着手のまま残ります。
