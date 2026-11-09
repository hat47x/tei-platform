# 到達できる位相の経路で提案を捨てない確認（2026-11-09）

## 対象

- 実装: `internal/hook/reach.go`（`PhaseReport.Produced`・`PhaseReport.Proposed`、`RunReachable`が`RunWithDeclaredValues`を通る）
- 試験: `internal/hook/reach_proposes_test.go`
- 対象版: `9ece5adc6`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 見つけた欠陥

CLIとStudioの参照実行が使う`hook.RunReachable`は、位相の実行に`RunWithDeclaredReads`を呼んでいました。この経路は**読み取りだけ**を扱うため、次の二つが起きていました。

1. `produces`を宣言したフックが値を返しても、提案が捨てられ、`PhaseReport`にも残らない。
2. `hook.ProducesFor`の宣言照合が走らない。つまり、**宣言していない属性を提案しても拒否されない**。

`hook.ProducesFor`と`RunWithDeclaredValues`は`internal/hook`にあり、単体試験もありましたが、実際に使われる経路（`RunReachable`）からは呼ばれていませんでした。宣言と実行が食い違う状態を、成功として通していたことになります。

## 変更の要点

- `RunReachable`が`RunWithDeclaredValues`を通るようにしました。これで`ReadsFor`と`ProducesFor`の両方が効きます。
- `PhaseReport.Produced`を追加し、位相ごとに提案を残します。位相をまたいで混ぜません。
- `PhaseReport.Proposed(phase)`を追加。
- 試験を4件追加。
- 既存の`Runner.Run`など、提案を扱わない経路は変えていません。

## 実行

```powershell
go test ./internal/hook -count=1
go test ./internal/studiohost ./internal/command -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRunReachableKeepsTheProposalADeclaredHookMade` 成功。宣言した`produces`の値が`PhaseReport.Produced["before-update"]`に残る。修正前は捨てられていた。
- `TestRunReachableRefusesAnUndeclaredProposal` 成功。宣言していない属性の提案が`HOOK_PRODUCE_UNDECLARED`で拒否される。修正前は通っていた。
- `TestRunReachableRecordsNoProposalForAReadOnlyHook` 成功。提案しないフックは`Produced`に何も足さない（空の地図を「提案した」と読ませない）。
- `TestRunReachableKeepsProposalsPerPhase` 成功。二つの位相が同じ属性を提案しても、位相ごとに分かれて残る。
- 既存の`internal/hook`の試験12件、`internal/studiohost`、`internal/command`、`cmd/tei`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- 参照実行の経路でも、`produces`の宣言照合が効く。
- 宣言した提案は捨てられず、位相ごとに報告される。

## 読み取れないこと

- 実データからの読み取りと、提案の実データへの反映。参照実行は`-input`の値だけを使い、提案を確定しません。
- 提案を採用する側（画面・帳票の再計算など）の実装。
- 実Pluginの`ProducingImplementation`。実装は試験用のものだけです。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「`reads`・`produces`の実データ観測」のうち、**宣言照合が実際の経路で効いていなかった**点を直した記録です。実データ観測そのものは未着手のままです。
