# フックの位相ごとの読み取り範囲の確認（2026-11-08）

## 対象

- 実装: `internal/hook/sequence.go`（`RunSequence`が位相ごとに`ReadsFor`を通す）
- 試験: `internal/hook/sequence_reads_test.go`
- 対象版: `58d2a8238`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 変更の要点

- `RunSequence`は、各位相の宣言を見て、その位相が宣言した属性の値だけを渡すようにしました。呼出し側は Pattern 全体の値を一度渡しますが、狭い宣言の位相が広い宣言の位相の値を受け継ぐことはありません。
- 位相が宣言しているのに値が無い場合は、`ReadsFor`が`HOOK_READ_NOT_OBSERVED`で拒否し、その位相で順序を止めます。

## 実行

```powershell
go test ./internal/hook -count=1
go vet ./internal/hook
gofmt -l .
go test ./internal/hook ./internal/validation ./internal/command ./internal/applicationview -count=1
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRunSequenceGivesEachPhaseItsOwnDeclaredReads` 成功。2属性を読む位相と1属性を読む位相が同じ Pattern にあり、後者は1属性だけを受け取る。
- `TestRunSequenceDoesNotPassAnUndeclaredRead` 成功。別の位相が読む属性の値を、宣言していない位相へ渡さない。
- `TestRunSequenceStopsOnAMissingRead` 成功。宣言した読み取りの値が無ければ`HOOK_READ_NOT_OBSERVED`で止まり、`Refused`にその位相が入る。
- 既存の`TestRunSequence...`・`TestReads...`・`TestRun...`・`TestDeclared...`・`TestGuarded...`・`TestReferenceRule...`・`TestCommandPatternFor...`はすべて成功。
- `go vet`と`gofmt -l .`は無出力（Go 1.26.10）。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- 位相が受け取る読み取りは、その位相の宣言で決まる。Pattern 全体の値がそのまま流れない。
- 未観測の読み取りは、実行前にその位相で止まる。

## 読み取れないこと

- 実データの読取り。値は呼出し側が渡します。
- `produces`（書き込み側）の照合。
- CLI・Studioの実行経路への配線。試験は`hook.Runner`を直接呼んでいます。
- 実Plugin・実Provider・実DB。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の宣言外拒否を、位相ごとの読み取りにも広げた記録です。
