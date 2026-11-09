# フックが名乗る結末名の検査の確認（2026-11-09）

## 対象

- 実装: `internal/hook/outcome.go`（`ValidateOutcome`、`HOOK_OUTCOME_UNDECLARED`）、`internal/hook/sequence.go`（`wrapFailure`が判定を通す）
- 試験: `internal/hook/outcome_validate_test.go`
- 対象版: `af852163f`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 背景

`AC-5`の残る未確認に「実Pluginが名乗る結末名の検査」がありました。前段（#3721）で語彙（`NotSatisfiedOutcomes`）とCommand拒否名への写し（`CommandOutcomeFor`）を入れましたが、**語彙は宣言されただけで、外から来る名前を照合していませんでした**。実装が`valueRejected`のような勝手な名前を名乗っても、そのまま呼出し側へ通っていました。

`hook.ReadsFor`が宣言外の読み取りを`HOOK_READ_UNDECLARED`で拒否し、`hook.ProducesFor`が宣言外の提案を`HOOK_PRODUCE_UNDECLARED`で拒否するのと同じ立場が、結末名には欠けていました。

## 変更の要点

- `ValidateOutcome`を追加。名乗った結末名が`NotSatisfiedOutcomes`に載っているかを照合し、載っていなければ`HOOK_OUTCOME_UNDECLARED`を返します。語彙は一箇所だけが持ち、ここでは照合するだけで新しい名前を足しません。
- 名乗らなかった（空の）結末名は拒否しません。理由だけを返す不成立は正当だからです。
- `wrapFailure`が、不成立を通す経路でこの照合を行います。すべての拒否がここを通るので、実装が語彙を広げる抜け道がありません。
- あわせて`outcome.go`の先頭に残っていた、存在しない定数`OutcomeDone`の説明を削除しました。
- 試験を6件追加。

## 実行

```powershell
go test ./internal/hook -count=1
go test ./internal/command ./internal/studiohost -count=1
go vet ./internal/hook
gofmt -l .
```

## 観測

- `TestRunnerRefusesAnOutcomeOutsideTheVocabulary` 成功。`valueRejected`を名乗る実装が`HOOK_OUTCOME_UNDECLARED`で拒否され、拒否文が名乗った名前を示す。
- `TestRunnerAcceptsADeclaredOutcome` 成功。語彙に載る名前は`NotSatisfied`としてそのまま呼出し側へ届く。
- `TestRunnerAcceptsARefusalWithoutAnOutcome` 成功。名前を付けない不成立は通る。
- `TestValidateOutcomeAcceptsTheVocabulary`・`TestValidateOutcomeRefusesAnInventedName` 成功。語彙の全名前が通り、`valueRejected`・`rejected`・`ok`・先頭空白付きは通らない。
- `TestRunReachableRecordsTheRefusalItStoppedAt` 成功。照合を挟んだ後も、`PhaseReport`の`Refused`と`Outcome`が保たれる。
- 既存の`internal/hook`の試験、`internal/command`、`internal/studiohost`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。

## 読み取れること

- 実装が語彙の外の結末名を名乗っても、`HOOK_OUTCOME_UNDECLARED`で止まる。
- 空の名前（理由だけの不成立）は正当として通る。

## 読み取れないこと

- プロセスを跨ぐ経路（CLIのJSON出力、Plugin境界の受け渡し）での検査。ここで入れたのは`internal/hook`の境界までです。
- 実Pluginの実装。実装は参照規則だけで、外部から来る名前を持つ実装はまだありません。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-5`の「実Pluginが名乗る結末名の検査」を、フック境界での照合として埋めた記録です。
