# フック宣言の機能定義書への反映の確認（2026-11-09）

## 対象

- 実装: `internal/artifact/functiondef.go`（生成規則`function-definition/v1`の行種別に`hook`を追加）
- 試験: `internal/artifact/functiondef_hook_test.go`
- 対象版: `39afead4d`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 変更の要点

- 機能定義書（`function-definition`成果物）の各`pattern`の下に、その対話が宣言したフックを`  hook <phase> reads … uses … produces … on-failure …`として出すようにしました。
- 並びは、実行時の位相順（`hook.PhaseOrder`）です。成果物と実行が同じ順序を使うので、定義書の読み順と実行順が食い違いません。順序に無い位相は末尾へ置き、黙って途中へ混ぜません。
- 生成規則の行種別一覧に`hook`を加えました。`period`・`permission`・`run-plan`を加えたのと同じく、版（`v1`）は据え置きです。規則の指紋は、規則文字列が変わったため変わります（`sha256:82e4a3…`）。
- 試験を3件追加。

## 実行

```powershell
go test ./internal/artifact ./internal/operations ./internal/hook ./internal/application -count=1
go vet ./internal/artifact
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
go run ./cmd/tei artifacts -workspace examples/xtea-lowcode -application order_entry
```

## 観測

- `TestRenderFunctionDefinitionShowsHooks` 成功。宣言したフックが、patternの下に位相・`reads`・`uses`・`produces`・`on-failure`つきで出る。
- `TestRenderFunctionDefinitionOrdersHooksByPhase` 成功。宣言順が`after-update`→`before-read`でも、成果物は`before-read`→`after-update`の順に出す。
- `TestRenderFunctionDefinitionWithoutHooksHasNoHookLine` 成功。フックの無いPatternには行を足さない。
- 既存の`function-definition`の試験（決定性・未解決・期間・効果規則・実行計画・権限）はすべて成功。
- `examples/xtea-lowcode`の`order_entry`で、成果物に実際に次の行が出る。

```text
pattern order_entry.order_record record
  hook before-update reads sales.order.status uses sales.submit_order produces sales.order.customer_id on-failure reject
```

- `go vet`と`gofmt -l .`は無出力（Go 1.26.10）。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- フックの宣言は、対話ビューだけでなく機能定義書にも出る。
- 定義書の並びは実行時の位相順と一致する。

## 読み取れないこと

- フックの実行結果（`hook.Runner`の配線）と、実Pluginの接続。
- 複数Pluginが同じ位相に登録した場合の合成。
- 実Provider・実DB。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の宣言の投影を、対話ビューから機能定義書へ広げた記録です。
