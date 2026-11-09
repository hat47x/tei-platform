# フックが宣言した能力を実際に呼ぶ配線の確認（2026-11-09）

## 対象

- 実装: `internal/hook/calls.go`（`RunDeclaredCalls`・`CallOutcome`・`DeclaredCalls`）、`internal/hook/reach.go`（`RunReachable`が宣言した能力を呼ぶ、`PhaseReport.Calls`）、`internal/hook/run.go`（`Runner.CallInputs`）、`cmd/tei/cli_view_hooks.go`と`internal/studiohost/hooks.go`（`CommandCaller`の配線）
- 試験: `internal/hook/calls_test.go`、`internal/hook/reach_call_test.go`、`internal/studiohost/hooks_run_test.go`
- 対象版: `dc0856af5`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 背景

前段（#3828）で「宣言した能力を呼ばないまま `executed` と報告しない」ようにしました。しかし、**呼ぶ配線そのものが無い**ままでした。`hook.CommandCaller`と`hook.Guarded`は単体試験つきで存在しましたが、CLIもStudioも`Runner.Caller`を設定していませんでした。

## 変更の要点

- `Runner.CallInputs`を追加。能力ごとに渡す入力を呼出し側が与えます。
- `RunDeclaredCalls`を追加。宣言の`uses`を、宣言で制限済みの呼出し元（`Declared`）を通して呼び、呼出しごとの`CallOutcome`（能力・呼んだか・理由・エラー）を返します。
- `DeclaredCalls`を追加。能力ごとに`CommandPatternFor`で`command` Patternを解決し、宣言外・曖昧な能力を推測せずに拒否します。
- `RunReachable`が、呼出し元と入力がある場合は**実際に呼びます**。呼べた呼出しは`PhaseReport.Calls`に残り、位相は`executed`になります。呼べなかった呼出しは従来どおり`Uncalled`＋`NotEvaluated`。呼出しが拒否されたらその位相で止まります。
- CLI（`view -command -execute`）とStudioの参照実行が、`hook.CommandCaller`を`Runner.Caller`へ渡すようにしました。両ホストとも`CommandPatternFor`で能力からPatternを解決します。
- 能力へ渡す入力は、**フックの入力（`-input`）をそのまま使います**。フックの`reads`から能力のパラメータへの対応は宣言に無いので、ここで発明していません。対応が無い能力は「呼ばなかった」として理由つきで残します。
- 試験を9件追加（`calls_test.go` 7件、`reach_call_test.go` 2件）。Studioの既存試験1件は、呼出しが成功するようになったため「呼ぶこと」を確かめる形へ書き換えました。

## 実行

```powershell
go test ./internal/hook ./internal/command ./internal/studiohost ./tools/repository-node-tests -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook ./internal/studiohost ./cmd/tei
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRunDeclaredCallsCallsTheDeclaredCapability` 成功。宣言した能力が境界へ届き、`CallOutcome.Called`が真になる。
- `TestDeclaredCallsResolvesTheCommandPattern` 成功。`command` Patternの無い能力は`HOOK_PATTERN_NOT_DECLARED`で拒否される。
- `TestCommandCallerRunsTheDeclaredCallInOnePath` 成功。`CommandPatternFor` → `CommandCaller` → `command.Invoker`の一続きが通る。
- `TestRunReachableCallsTheDeclaredCapability` 成功。`RunReachable`が実際に呼び、位相は`executed`、`Calls`に記録が残る。
- `TestRunReachableStopsAtACallRefusal` 成功。呼出しが拒否されたらその位相で止まる。
- `TestRunDeclaredCallsRefusesADeclaredButUnboundCapabilityNotBound` 成功。入力の無い能力は推測して呼ばず、理由つきで「呼ばなかった」。
- `TestExecuteCommandCallsTheDeclaredCapability` 成功（Studio）。`calls`に`sales.submit`が`Called:true`で入る。
- **実CLIで観測**: フックが`uses sales.submit_order`を宣言するfixtureで`view -command -execute`を実行し、JSONに`"calls":{"before-update":[{"capability":"sales.submit_order","called":true}]}`が出ることを確認しました。テキストでも`hook before-update: executed`。
- `internal/hook`・`internal/command`・`internal/studiohost`・`tools/repository-node-tests`・`cmd/tei`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- フックが`uses`で宣言した能力は、CLIとStudioの参照実行から**実際にCommand境界を通って呼ばれる**。
- 呼ばれなかった呼出しは、理由つきで区別される。呼出しの拒否は位相を止める。

## 読み取れないこと

- **フックの`reads`から能力のパラメータへの対応**。宣言に無いため、フックの入力をそのまま渡しています。この対応規則は未設計です。
- 実Pluginからの供給路。規則は参照実装だけです。
- 実データの読取り、`after-*`へ到達する実行経路。
- 呼出しが引き起こす効果の観測（参照実行は何も書きません）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「フックから能力を実際に呼ぶ配線」を埋めた記録です。
