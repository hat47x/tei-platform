# フック宣言の印刷面への反映の確認（2026-11-07）

## 対象

- 実装: `internal/applicationview/print.go`（record の印刷面にフック宣言の節を加えた）
- 試験: `internal/applicationview/print_hook_test.go`
- 参考: `internal/applicationview/html.go`（画面面は以前から同じ内容を出している）
- 対象版: `f63c7723b`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.27.0`

## 変更の要点

- record ビューの印刷面（`-format print`）に、画面面と同じフック宣言の節を加えました。
- 節は、フックが宣言されているときだけ出ます。宣言が無ければ、空の見出しを出しません。
- 表示する内容は画面面と同じ（位相・`reads`・`uses`・`produces`・`on-failure`）で、意味の解釈は `applicationview` の投影に任せています。

## 実行

```powershell
go test ./internal/applicationview -count=1
go vet ./internal/applicationview
gofmt -l internal/applicationview
go run ./cmd/tei view -workspace examples/xtea-lowcode -application order_entry -pattern order_entry.order_record -format print
```

## 観測

- `TestRenderPrintHTMLCarriesTheHooks` 成功。印刷面に「フック」の節と、宣言した位相・能力名が出る。
- `TestPrintHTMLWithoutHooksHasNoEmptySection` 成功。フック宣言が無いときは、その節を出さない。
- 既存の `TestRenderPrintHTML`・`TestRenderPrintHTMLCarriesTemplate`・`TestPrintHTMLCarriesTheSemanticFingerprint` は成功のまま。
- `examples/xtea-lowcode` の `order_entry.order_record` を `-format print` で投影すると、画面面と同じ内容が出る。

```text
<section><h2>フック</h2><ul><li>before-update: reads sales.order.status uses sales.submit_order produces sales.order.customer_id on-failure reject</li></ul></section>
```

- `go vet` と `gofmt -l` は無出力。

## 読み取れること

- 同じ宣言から、画面面と印刷面が同じフックの記述を出す。
- フックが無い Pattern では、画面にも印刷にも余計な節を出さない。

## 読み取れないこと

- 実印刷・用紙面の見た目（余白・改ページ・部品）の一致。ここで確かめたのは、投影されたテキストの一致です。
- フックの実行そのもの（`hook.Runner` の配線）と、実 Plugin の接続。
- Studio の画面・Desktop 表示。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「画面と帳票で同じ結果になること」のうち、宣言の投影が同じになる範囲を確かめました。実行結果の一致は未確認です。
