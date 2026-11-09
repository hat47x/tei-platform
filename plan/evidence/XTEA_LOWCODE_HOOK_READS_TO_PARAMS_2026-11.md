# フックの`reads`から能力のパラメータへの対応規則の確認（2026-11-10）

## 対象

- 実装: `internal/hook/calls.go`（`DeclaredCalls`・`BindInputsFromReads`）、`internal/hook/reach.go`（位相の`reads`から呼出し入力を導く）、`internal/hook/run.go`（`Runner.CallInputs`を削除）、`cmd/tei/cli_view_hooks.go`、`internal/studiohost/hooks.go`
- 試験: `internal/hook/calls_test.go`、`internal/hook/reach_uses_test.go`、`internal/hook/reach_call_test.go`、`internal/studiohost/hooks_test.go`
- 対象版: `33e330ede`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 背景

`AC-2`の残る未確認に「`reads`から能力のパラメータへの対応規則」がありました。前段（#3843）でフックが宣言した能力を実際に呼ぶ配線を入れましたが、**能力へ渡す入力は呼出し元が与えた`-input`をそのまま渡していた**ため、フックの`reads`と能力のパラメータがどう結びつくのかは決まっていませんでした。

## 見つけた対応規則

対応は**既に宣言に書かれていました**。フックを持つ`record` Patternのinteractionにある

```text
command <capability> {
    input <パラメータ名> <属性>
}
```

が、「能力のパラメータ`<パラメータ名>`は属性`<属性>`から供給される」と述べています。フックの`reads`は属性の並びなので、**同じ属性を読んでいればそのままそのパラメータへ渡せる**。新しい宣言構文も、新しい意味も要りませんでした。

なお、`command` Pattern（`Kind: command`）はinteractionを持てません（`APPLICATION_PATTERN_INTERACTION_NOT_ALLOWED`）。したがって対応は`command` Pattern側ではなく、フックを宣言したPatternのinteractionにあります。当初これを取り違えて実装し、実測で気づいて直しました。

## 変更の要点

- `DeclaredCalls`が、位相の`reads`（属性→値）から、能力ごとのパラメータ入力を導きます。
- `BindInputsFromReads`が対応を適用します。interactionの`command <capability> { input <name> <attribute> }`を引き、`reads[<attribute>]`があれば`<name>`へ入れます。
- **読んでいない属性のパラメータは入れません。** 推測して渡すのではなく、欠けたままにして、Command境界に「入力が足りない」と言わせます。
- `Runner.CallInputs`（呼出し元が対応表を持ち込む欄）を削除しました。対応は宣言が持つので、ホストが別の対応を持ち込む余地を残しません。
- CLIとStudioは、位相の`reads`をそのまま`RunReachable`へ渡します（前段のまま）。
- `uses`の能力に`command` Patternが無い場合と、呼出し元が無い場合は、位相を`NotEvaluated`として理由つきで記録します（実行済みとはしません）。
- 試験を追加（対応規則3件、`uses`解決の失敗1件、呼出し元ありの実呼出し1件）。

## 実行

```powershell
go test ./internal/hook ./internal/command ./internal/studiohost -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook ./cmd/tei ./internal/studiohost ./internal/command
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestBindInputsFromReadsUsesTheDeclaredBinding` 成功。interactionが「パラメータ`order`は属性`sales.order.order_id`」と述べており、フックがその属性を読むと`order`に値が入る。
- `TestBindInputsFromReadsOmitsAnUnreadParameter` 成功。読んでいない属性しか無ければ、どのパラメータも埋めない。
- `TestBindInputsFromReadsIgnoresAnotherCapability` 成功。呼ぶ能力を名指すinteractionの`command`だけを読む。
- `TestRunReachableRecordsAnUnresolvableCapability` 成功。`command` Patternの無い`uses`は`NotEvaluated`で理由つき、境界へは届かない。
- `TestRunReachableCallsTheDeclaredCapabilityWhenACallerIsConfigured` 成功。呼出し元があれば、対応規則だけで能力が境界へ届く。
- **実CLIで観測（正例）**: `command sales.submit_order { input order sales.order.order_id }` と `hook before-update { reads sales.order.order_id uses sales.submit_order }` を宣言したfixtureで`view -command -execute`を実行し、`"calls":{"before-update":[{"capability":"sales.submit_order","called":true}]}` と `"executed":["before-update"]` を確認しました。呼出し元が対応表を持ち込まなくても呼べています。
- **実CLIで観測（負例）**: 同じfixtureで対応だけを`input order sales.order.status`へ変え、フックは`sales.order.order_id`を読むままにすると、`"calls":{"before-update":[{"capability":"sales.submit_order","called":true,"error":"INVALID_COMMAND_INPUT: missing inputs: [order]"}]}` となり、`ok:false`・`refused:"before-update"` で止まりました。**値を推測して渡していない**ことが、Command境界の拒否として現れています。
- 既存の`internal/hook`・`internal/command`・`internal/studiohost`・`cmd/tei`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- フックの`reads`は、宣言済みの対応（interactionの`command`の`input`）を通して能力のパラメータになる。
- 対応に無い値は渡さない。足りなければCommand境界が拒否し、フックの位相は止まる。

## 読み取れないこと

- 1つの属性を複数のパラメータへ、あるいは複数の属性を1つのパラメータへ束ねる規則（現在は`name → attribute`の単純な対応のみ）。
- 型変換。属性の型とパラメータの型が違う場合の変換規則は未定義です（Command境界の値検査に委ねています）。
- 実Pluginからの供給路、`reads`・`produces`の実データ観測、`after-*`へ到達する実行経路。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「`reads`から能力のパラメータへの対応規則」を、既存の宣言から導く形で埋めた記録です。
