# 位相ごとにフックの実装を供給できる経路の確認（2026-11-10）

## 対象

- 実装: `internal/hook/run.go`（`Runner.ByPhase`・`implementationFor`）、`internal/hook/sequence.go`・`internal/hook/values.go`（実装の解決）
- 試験: `internal/hook/byphase_test.go`
- 対象版: `b0b770019`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 背景

`AC-2`の残る未確認「実Pluginからの供給路」に着手しました。

`hook.Runner`は`Implementation`を一つだけ持ち、すべての位相を同じ実装で賄う前提でした。しかし設計（[受容アーキテクチャ](../design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)のD2）は「フックの本体は規則Pluginが供給する」とし、位相ごとに別の規則があり得ます（読取前の検証と更新前の検証は別の規則）。

設計は同時に「規則Pluginを別の契約で束縛するかはD2の採用時に決める」としており、**規則Pluginの契約そのものは未決定**です。契約を勝手に発明すると、後で設計と食い違います。

## 変更の要点

- `Runner.ByPhase`（`map[string]Implementation`）を追加しました。位相の名前で実装を選べます。
- 位相の実装が無ければ`Implementation`へ落ちます。どちらも無ければ`HOOK_IMPLEMENTATION_UNAVAILABLE`で止まります（黙って成功にしません）。
- `Runner.Run`・`runDeclared`・`RunWithDeclaredValues`の三経路が同じ解決を使います。片方だけ直すと、単一位相の入口と位相順の入口で挙動が変わります。
- **`PhaseOrder`が持たない位相名を鍵にしても実装は差し込まれません。** 宣言に無い位相は実行されないので、任意の文字列で別の本体を呼ばせる経路になりません。
- 試験を5件追加。

## 設計上の判断（仮定として記録）

規則Pluginの契約（Pluginが何を実装し、どう版を束縛するか）は**決めていません**。ここで入れたのは「位相ごとに実装を供給できる」という最小の口だけです。契約を決める作業はD2の採用時に行うもので、実装の都合で先に固定しません。

## 実行

```powershell
go test ./internal/hook ./internal/command ./internal/studiohost -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook ./cmd/tei ./internal/studiohost
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRunnerUsesThePhaseImplementation` 成功。`before-read`と`before-update`に別の実装を渡すと、それぞれの実装が自分の位相でだけ呼ばれる。
- `TestRunnerFallsBackToTheWholeImplementation` 成功。位相別の指定が無い位相は`Implementation`で実行される。
- `TestRunnerRefusesWhenNoImplementationIsAvailableForAPhase` 成功。どちらも無い位相は`HOOK_IMPLEMENTATION_UNAVAILABLE`で止まり、`Refused`にその位相が入る。指定のあった位相は実行済みとして残る。
- `TestRunnerIgnoresAnImplementationKeyedByAnUnknownPhase` 成功。`during-update`という鍵は使われず、両位相が共通実装で実行される。
- `TestRunnerUsesThePhaseImplementationForRun` 成功。単一位相の入口（`Runner.Run`）も同じ解決を使う。
- 既存の`internal/hook`の試験はすべて`ok`。`internal/command`・`internal/studiohost`も`ok`。`go vet`と`gofmt -l .`は無出力。

## 実装中に踏んだ誤り

`Runner.Run`だけ`r.Implementation`を直接参照したままにしていたため、位相別の実装だけを渡した試験でnil参照になりました。三経路すべてが`implementationFor`を通ることを試験で確かめています。

## 読み取れること

- 位相ごとに別のフック本体を供給できる。
- 供給が無い位相は、共通実装へ落ちるか、明示的に止まる。
- 宣言に無い位相名で実装を差し込むことはできない。

## 読み取れないこと

- **規則Pluginの契約**（Pluginが何を宣言し、どう版を束縛するか、どのように`Implementation`へ適合させるか）。未決定です。
- 実Pluginの読み込みと束縛（`plugin`パッケージの宣言から`hook.Implementation`を構成する経路）。
- 実Pluginが呼ばれたことの観測、`reads`・`produces`の実データ観測、`after-*`へ到達する実行経路。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「実Pluginからの供給路」のうち、**位相ごとに本体を供給できる口**を用意した記録です。Plugin契約と束縛は未着手のまま残ります。
