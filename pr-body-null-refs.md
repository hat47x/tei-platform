# `source-index` の空の参照一覧を `[]` で返す（`null` をやめる）

`source-index` の `mode=index` は、参照が1件も無い Workspace で `result.references` を **`null`** で返していました（#3456）。`internal/sourceindex.Index` の `References` に `omitempty` が無く、nil スライスがそのまま `null` になるためです。同じ「参照の一覧」を返す `inspect`・`trace` は `[]` を返すので、読み手は `null` と `[]` を別々に扱う必要がありました。

## 変更

- `cmd/tei/source_index.go` に `sourceIndexIndexView` を追加し、`mode=index` の `result` をその型で書きます。`artifacts`・`references` が nil のとき `[]` にします。
- `mode=subject` も、参照の無い subject では `references`・`artifacts` を `[]` にします。
- この型は**応答の投影だけ**です。`sourceIndexFingerprint` は `internal/operations` が `sourceindex.Index` 自体から計算するので、指紋は変わりません（`reference/workspace` で `sha256:c92de09f…`／`sha256:1827f4b4…`／`sha256:f414e2b0…` が従来と同一であることを実測）。

## 検証（この head、Windows 11、go1.27.0）

- `gofmt -l cmd`: 空、`go vet ./cmd/tei ./internal/operations ./internal/sourceindex`: PASS、`go build ./...`: PASS
- `go test ./cmd/tei -count=1`: PASS（265 秒、全件）
- `go test ./tools/repository-plan-check -count=1`: PASS
- 追加試験: `docs` だけの Workspace を一時ディレクトリに作り、`mode=index` の `references` が `[]` で出ること、参照の無い subject で `mode=subject` の `references` が `[]` であることを検査。
- 負例: `newSourceIndexIndexView` を外すと `inspection type=sourceindex.Index` で、`view.References` の正規化だけを外すと `"references":null` で、それぞれ試験が失敗する（コードは戻した）。
- 実測: 意味宣言の無い Workspace で `"references": []`（修正前は `null`）。

## 決めないこと

- `null` を「未取得・未対応」の意味に限定するか、全コマンドで空配列へ統一するか。ここでは `source-index` の2モードだけを直しました。
- `proposals`・`diagnostics`・`sourceRootFindings` の空の扱いは変えていません。
