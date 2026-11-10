# フック位相の到達範囲を一箇所に置く確認（2026-11-09）

## 対象

- 実装: `internal/hook/reach.go`（`Reach`・`ReferenceReach`・`PhaseReport`・`Runner.RunReachable`）
- 呼出し側: `cmd/tei/cli_view_hooks.go`、`internal/studiohost/hooks.go`
- 試験: `internal/hook/reach_test.go`
- 対象版: `c90bb502d`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 背景

`AC-2`の「CLI参照実行からのフック呼出し」（#3603）と「Studio参照実行からのフック呼出し」（#3635）は、どちらも同じ位相の選び方を別々に持っていました。

- 到達できる位相（`before-*`）だけを実行する
- 到達しない位相（`after-*`）は理由つきで`skipped`にする
- 宣言した読み取りに値が無い位相は`notEvaluated`にする
- 不成立で止め、`on-failure warn`は記録して続ける
- 順序に無い位相を黙って落とさない

同じ規則が二箇所にあると、片方だけ直したときに CLI と Studio の答えが食い違います。規則を`internal/hook`へ移しました。

## 変更の要点

- `Reach`は「その実行境界が位相へ到達できるか」を型で言わせます。`ReferenceReach`は`before-*`へ到達し、`after-*`へは到達しないと答えます。
- `Runner.RunReachable`が、到達できる位相だけを`PhaseOrder`の順に実行し、`PhaseReport`を返します。
  - `Executed`・`Warned`・`Skipped`・`NotEvaluated`・`Reasons`・`Refused`・`Outcome`を持つ。
  - `Skipped`（原理的に到達しない）と`NotEvaluated`（到達しうるが前提が欠けた）を分ける。
  - フックを一つも宣言していない Pattern、および`command`など対話を持たない Pattern は`nil, nil`を返す。フックが無い実行を「フックを見た実行」と読ませない。
- CLI と Studio は、それぞれ `PhaseReport`を埋め込んだ報告型に自前の文言（境界の注記）だけを足す形にしました。位相の選び方は持ちません。
- 試験を7件追加。

## 実行

```powershell
go test ./internal/hook ./internal/studiohost -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook ./internal/studiohost ./cmd/tei
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRunReachableRunsTheReachablePhasesInOrder` 成功。宣言順が`before-update`→`before-read`でも、実行は`before-read`→`before-update`になる。
- `TestRunReachableRecordsTheUnreachablePhasesWithAReason` 成功。`after-update`は`Skipped`で、理由は`ReasonAfterChangeUnreachable`。
- `TestRunReachableRecordsAMissingReadAsNotEvaluated` 成功。読み取りの無い位相は`NotEvaluated`で、理由が属性名を示す。
- `TestRunReachableStopsAtARefusal` 成功。不成立で止まり、`Refused`と`Outcome`を残し、それ以前の位相は`Executed`に残る。
- `TestRunReachableWithNoHooksReportsNothing` 成功。フックの無い Pattern は`nil`を返す。
- `TestRunReachableRecordsAnUnsupportedPhase` 成功。順序に無い位相は`Skipped`で記録される。
- `TestRunReachableKeepsOnFailureWarnGoing` 成功。`on-failure warn`の位相は`Warned`に記録して続く。
- 既存の CLI 6件・Studio 2件のフック試験は、呼出し側を差し替えた後もすべて成功。
- `cmd/tei`の全試験と`internal/hook`・`internal/studiohost`の全試験が`ok`。`go vet`と`gofmt -l .`は無出力。

## 読み取れること

- CLI と Studio は同じ位相の選び方を使う。片方だけを直して食い違わせる余地が減った。
- 「到達しない位相」と「前提が欠けた位相」は、共通の型で区別される。

## 読み取れないこと

- 実データの読取り。参照実行は渡された`inputs`だけを使う。
- 規則Pluginの実供給路（参照規則だけ）。
- `after-*`へ到達できる実行境界（書き込む実行）の実装。`Reach`はその境界を受け取れる形にしたが、実装はまだ無い。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の配線を、二つの呼出し側で同じ規則に揃えた記録です。
