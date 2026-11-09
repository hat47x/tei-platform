# フックが提案する値の宣言照合の確認（2026-11-08）

## 対象

- 実装: `internal/hook/values.go`（`Produced`・`ProducingImplementation`・`ProducesFor`・`Runner.RunWithDeclaredValues`）
- 試験: `internal/hook/values_test.go`
- 対象版: `13447ec32`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 変更の要点

- フックが宣言する`produces`について、実装が提案した値が宣言に載っているかを照合する境界を置きました。宣言していない属性を提案すると`HOOK_PRODUCE_UNDECLARED`で拒否します。宣言した値が提案されないことは拒否ではありません（空の提案は「何も作らない」）。
- 値の提案は`ProducingImplementation`（`RunProducing`）に分けました。読み取りだけのフックは従来の`Implementation`のままで動きます。
- `Runner.RunWithDeclaredValues`は、`ReadsFor`で読み取りを絞り、`ProducesFor`で提案を照合します。実装の失敗は`wrapFailure`を通すので、`on-failure warn`は警告のまま、不成立は不成立のままです。
- 試験を9件追加。

## 実行

```powershell
go test ./internal/hook -count=1
go vet ./internal/hook
gofmt -l .
go test ./internal/hook ./internal/validation ./internal/command ./internal/applicationview -count=1
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestProducesForAcceptsDeclaredValues` 成功。宣言した属性の提案は通る。
- `TestProducesForRefusesAnUndeclaredValue` 成功。宣言していない属性の提案は`HOOK_PRODUCE_UNDECLARED`で拒否される。
- `TestProducesForAcceptsAnEmptyProposal` 成功。空の提案は拒否ではない。
- `TestProducesForCopiesTheValues` 成功。返る提案は複製で、入力と共有しない。
- `TestRunWithDeclaredValuesReturnsTheProposal` 成功。Runnerは照合済みの提案を返す。
- `TestRunWithDeclaredValuesRefusesAnUndeclaredProposal` 成功。宣言外の提案は返らない。
- `TestRunWithDeclaredValuesRefusesAnUndeclaredRead` 成功。読み取りの照合が実装の前に走る。
- `TestRunWithDeclaredValuesRunsAPlainImplementation` 成功。値を提案しない実装は従来どおり動き、提案は空。
- `TestRunWithDeclaredValuesKeepsTheWarnRule` 成功。提案側の失敗も`warn`の規則に従う。
- 既存の`internal/hook`の試験はすべて成功。`go vet`・`gofmt -l .`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- フックが提案できる項目は、宣言の`produces`で決まる。
- 宣言外の提案は、呼出し側へ返す前に拒否される。
- 値の提案を返す契約は、読み取りだけのフックを壊さない。

## 読み取れないこと

- 提案した値の採用（確定）。`ProducesFor`は提案を運ぶだけで、保存や上書きは行いません。
- 提案が実データへ反映されたかの観測。
- CLI・Studioの実行経路への配線。
- 実Plugin・実Provider・実DB。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の宣言外拒否を、値の提案側にも広げた記録です。
