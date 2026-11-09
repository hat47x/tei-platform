# `source-index` の空の参照一覧が `null` で返る（記録）

- 記録日: 2026-11-08
- 対象: `cmd/tei/source_index.go`
- 関連: #3456（応答間の `null` と `[]` の混在）、#3347（応答の形の不統一。Closed）

## 観測（修正前）

`source-index` の `mode=index` は、参照が1件も無い Workspace で `result.references` を `null` で返していた。

```bash
# plan/ と docs/ だけの Workspace（意味宣言なし）
tei source-index -workspace /tmp/ws-docs
# "references": null
```

原因は `internal/sourceindex.Index` の `References []Reference` に `omitempty` が無く、nil スライスが `null` として直列化されること。`inspect` と `trace` は `make(...,0,...)` で組み立てるため `[]` を返す。同じ「参照の一覧」が入口によって `null` と `[]` に分かれていた。

## 変更

- `sourceIndexIndexView` を追加し、`mode=index` の `result` をその型で書き出す。`artifacts` と `references` が nil のとき `[]` にする。
- `mode=subject` も、参照が無い subject では `references` と `artifacts` を `[]` にする。
- `sourceIndexIndexView` は**応答の投影だけ**であり、`sourceIndexFingerprint` は `internal/operations` が `sourceindex.Index` 自体から計算する。したがって `worldFingerprint`・`sourceIndexFingerprint`・`modelRevision` は変わらない（`reference/workspace` で実測: `sha256:c92de09f…` / `sha256:1827f4b4…` / `sha256:f414e2b0…` がいずれも従来と同じ）。

## 検証

- `cmd/tei/source_index_test.go` を追加・更新。
  - `docs` だけの Workspace を一時ディレクトリに作り、`mode=index` の `artifacts` と `references` が `null` ではなく `[]` で出ることを検査。
  - 参照の無い subject で `mode=subject` の `references` が `[]` であることを検査。
  - 全体索引の型が `sourceIndexIndexView` であることを検査。
- 負例: `newSourceIndexIndexView` を外すと `inspection type=sourceindex.Index` で失敗、`view.References` の正規化だけを外すと `references:null` で失敗する（いずれもコードは戻した）。
- `go test ./cmd/tei -count=1` PASS（265秒）。`gofmt -l cmd internal` 空、`go vet ./cmd/tei ./internal/operations ./internal/sourceindex` PASS。
- 実測: `reference/workspace` の `source-index` は `references` に配列を返し続け、指紋3種は不変。

## 主張しないこと

- `null` を「未取得・未対応」の意味に限定するか、全コマンドで空配列へ統一するかは決めていない。ここでは `source-index` の2モードだけを `[]` にした。
- `proposals`・`diagnostics`・`sourceRootFindings` の空の扱いは変えていない。
