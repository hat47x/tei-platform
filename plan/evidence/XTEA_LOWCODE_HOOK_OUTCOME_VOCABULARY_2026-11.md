# フックの結末名とCommand拒否名の対応の確認（2026-11-09）

## 対象

- 実装: `internal/hook/outcome.go`（`NotSatisfiedOutcomes`・`CommandOutcomeFor`・`OutcomeOf`・`OutcomeUnmet`・`OutcomeValueRejected`）、`internal/hook/reference.go`（結末名を定数へ）
- 試験: `internal/hook/outcome_test.go`
- 対象版: `9cabf6cf4`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 背景

`AC-5`の残る未確認に「フックの結末名とCommand結末名の対応規則」がありました。

- フックは`hook.NotSatisfied.Outcome`に業務上の結末名を持つが、**自由な文字列**でした。`value-rejected`と`valueRejected`が混在しても誰も止めません。
- Command側は`command.FailureOutcome`が拒否符号から結末名を一つに決めており、CLIは`COMMAND_REFUSED`でそれを報告します。
- 両者の対応を書いた場所が無く、フックの結末名がCommandの結末名と同じ意味かは、読み手の推測に任されていました。

## 変更の要点

- `NotSatisfiedOutcomes`で、フックが名乗ってよい結末名を列挙しました。Command側の`command.FailureOutcome`が返す名前を正としています。
- `CommandOutcomeFor`が、フックの結末名をCommandの結末名へ写します。**写しの規則は`command.FailureOutcome`を呼ぶだけで、ここで発明しません**。綴りが変われば`CommandOutcomeFor`の結果が変わり、試験が落ちます。
- `not-satisfied`だけは対応を返しません。Command側に対応する拒否符号が無いので、Commandの結末として報告すると「フックが条件を満たさない」が「Commandの拒否理由」に化けます。**対応が無いことを型で示す**（`false`を返す）ようにしました。
- `OutcomeOf`で、`NotSatisfied`以外のエラーが結末名を持つと読まないようにしました。実行の失敗を業務上の結末にしないためです。
- `ReferenceRule`が名乗る結末名を、文字列リテラルから定数（`OutcomeUnmet`・`OutcomeValueRejected`）へ変えました。
- 試験を8件追加。

## 実行

```powershell
go test ./internal/hook ./internal/command ./cmd/tei ./internal/studiohost -count=1 -timeout 30m
go vet ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestNotSatisfiedOutcomesMapToCommandNames` 成功。列挙した結末名が、`command.FailureOutcome`の綴りと一致する（`not-satisfied`だけ対応を持たない）。
- `TestCommandOutcomeForNamesTheReferenceRefusals` 成功。`not-found`・`already-exists`・`value-rejected`・`rule-rejected`・`undeclared`がそれぞれ同じ名前へ写る。
- `TestCommandOutcomeForRefusesAnUnknownName` 成功。`valueRejected`と空文字は対応を持たない（勝手な名前がCommandの拒否理由にならない）。
- `TestReferenceRuleUsesTheDeclaredOutcomes` 成功。参照規則が名乗るのは語彙に載る名前だけ。
- `TestOutcomeOfIgnoresAnExecutionFailure` 成功。実行の失敗と`nil`は結末名を返さない。
- `TestOutcomeOfReportsNoOutcomeWhenTheHookNamedNone` 成功。名前を付けなかった不成立に、こちらで名前を補わない。
- `TestEveryVocabularyNameIsReachableFromFailureOutcome` 成功。逆向きにも、語彙の名前が`command.FailureOutcome`の返す名前であることを確かめる。
- `TestNotSatisfiedOutcomesHasNoDuplicates` 成功。
- `internal/hook`・`internal/command`・`internal/studiohost`・`cmd/tei`の全試験が`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- フックの結末名の語彙が一箇所に定まり、Commandの結末名との対応が明示された。
- 対応の無い名前（`not-satisfied`や打ち間違い）は、Commandの拒否理由に化けない。
- 二つの語彙が離れたら、試験が落ちる。

## 読み取れないこと

- 実Pluginが名乗る結末名。実装は参照規則だけで、外から来る名前を検査する経路は未接続。
- プロセスを跨いだ結末名の受け渡し（CLIのJSON出力とCommand境界の照合）。
- 多結果・多重一致の表現（`AC-5`の別の残り）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-5`の「フックの結末名とCommand結末名の対応規則」を、語彙の列挙と一方向の写しで埋めた記録です。
