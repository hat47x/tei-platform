# フックが読むのは宣言した属性だけであることの確認（2026-11-08）

## 対象

- 実装: `internal/hook/reads.go`（`ReadsFor`・`Runner.RunWithDeclaredReads`）
- 試験: `internal/hook/reads_test.go`
- 対象版: `6bed8f296`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.27.0`

## 変更の要点

- `ReadsFor`は、宣言の`reads`と渡された値を照合します。
  - 宣言した属性の値が無ければ`HOOK_READ_NOT_OBSERVED`で拒否します。未観測を空の値として通しません。
  - 宣言していない属性の値が渡されれば`HOOK_READ_UNDECLARED`で拒否します。フックは宣言より広いデータを見ません。
  - 返すのは複製で、呼出し側が検査後に値を足せません。
- `Runner.RunWithDeclaredReads`は、宣言を解決して`ReadsFor`を通した値だけを実装へ渡します。`on-failure warn`と不成立の扱いは`Runner.Run`と同じ規則です。

## 実行

```powershell
go test ./internal/hook -count=1
go vet ./internal/hook
gofmt -l internal/hook
go test ./internal/hook ./internal/validation ./internal/command ./internal/applicationview -count=1
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestReadsForPassesTheDeclaredReads` 成功。宣言した属性の値はそのまま実装へ届く。
- `TestReadsForRefusesAMissingRead` 成功。宣言した読み取りが未観測なら`HOOK_READ_NOT_OBSERVED`で拒否される。
- `TestReadsForRefusesAnUndeclaredRead` 成功。宣言していない属性の値は`HOOK_READ_UNDECLARED`で拒否される。
- `TestReadsForRefusesWithoutADeclaration` 成功。宣言の無い読み取りは拒否される。
- `TestReadsForCopiesTheValues` 成功。返る値は複製で、入力とは共有しない。
- `TestRunWithDeclaredReadsHandsTheDeclaredValues` 成功。Runner経由でも宣言した値だけが実装へ届く。
- `TestRunWithDeclaredReadsRefusesBeforeTheImplementation` 成功。宣言外の値は実装へ届かず、実装は呼ばれない。
- `TestRunWithDeclaredReadsKeepsTheWarnRule` 成功。`on-failure warn`の扱いは変わらない（警告として返る）。
- 既存の`internal/hook`の試験はすべて成功。`go vet`・`gofmt -l`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- フックが実装へ渡せる読み取り値は、宣言の`reads`で決まる。
- 未観測と宣言外は、どちらも黙って通さない。
- 検査した値は、その後の書き換えで広がらない。

## 読み取れないこと

- 実データの読取りそのもの。`ReadsFor`は渡された値を照合するだけで、Providerから値を取得しません。
- `produces`（書き込み側）の照合。
- `RunWithDeclaredReads`をCLI・Studio・`RunSequence`へ配線すること。
- 実Plugin・実Provider・実DB。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「宣言外の能力呼出しを拒否する」の読み取り側の対称として、宣言外の読み取りを拒否する境界を置きました。
