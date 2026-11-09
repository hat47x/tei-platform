# `model catalog` の名前空間不在が CLI と MCP で別の成否になっていた（記録）

- 記録日: 2026-11-08
- 対象: `internal/operations/model_catalog.go`、`cmd/tei/cli_model.go`、`tools/reference-mcp-readonly-adapter`
- 関連: #3420（CLI/MCP の結果・失敗・成否の一致）、#3419、ADR-0048。

## 観測（修正前）

同じ Workspace と同じ `-namespace nope` に対して、2つの入口が別の成否を返していた。

```text
tei model catalog -workspace reference/workspace -namespace nope
# exit 1, {"schemaVersion":"tei.cli.model/v0","ok":false,
#          "error":{"code":"NAMESPACE_NOT_FOUND","message":"namespace nope is not a semantic namespace in this workspace"}}

MCP tei_model_catalog {"namespace":"nope"}
# isError なし, {"schemaVersion":"tei.cli.model/v0","ok":true,
#   "result":{...,"namespaceMatched":false,...,"entries":[]}}
```

原因は、名前空間の不在判定を CLI（`cmd/tei/cli_model.go`）だけが持っていたこと。`internal/operations.CatalogWorkspace` は `NamespaceMatched=false` を返すだけで、失敗としては返さなかった。MCP 参照 Adapter はその戻り値をそのまま包んだため、`ok:true` の成功として返していた。

## 変更

- `internal/operations/model_catalog.go` に `CatalogError`・`CatalogCodeNamespaceNotFound`・`ModelCatalog.NamespaceFilterError()` を追加。名前空間の不在を**投影自身の判定**にした（コードと文言は CLI が持っていたものを移した）。
- `cmd/tei/cli_model.go` は自前の分岐を消し、`NamespaceFilterError()` の結果を CLI の失敗へ変換するだけにした。CLI の応答は変わらない（文言・コード・終了コード1が同一）。
- `tools/reference-mcp-readonly-adapter` の `tei_model_catalog` は、`NamespaceFilterError()` があれば `isError:true` と CLI と同じ文字列コードで返す。`errorEnvelope` も `CatalogError` を認識するようにした。
- `catalog_parity_test.go` を追加。実 CLI と実 Adapter を起動し、(a) 名前空間不在が両方で `NAMESPACE_NOT_FOUND`・同じ文言・`ok:false`、(b) 宣言済みの名前空間（`std`）は両方で成功、(c) 対象不在の `inspect`・`trace` は両方で `SUBJECT_NOT_FOUND` であることを検査する。

## 検証

- `gofmt -l tools cmd internal`: 空、`go vet ./tools/reference-mcp-readonly-adapter ./internal/operations ./cmd/tei`: PASS、`go build ./...`: PASS
- `go test ./tools/reference-mcp-readonly-adapter -count=1`: PASS
- `go test ./cmd/tei -run 'TestCLIModel|TestModel' -count=1`: PASS
- `go test ./internal/operations -run 'Catalog' -count=1`: PASS
- 負例: Adapter の `NamespaceFilterError` 呼び出しを外すと `TestCatalogNamespaceFailureMatchesTheCLI` が「missing namespace を成功として返した」で失敗する（コードは戻した）。
- 実測: 両入口が同じ JSON（`schemaVersion`・`ok:false`・`error.code`・`error.message`）を返す。

## 主張しないこと

- `catalog` の `entries` の中身が両入口で同値であることは、ここでは検査していない（既存の `TestAdapterMatchesTheSameOperationResults` が指紋と件数を比較している）。
- 入力の型・未知欄・上限の検査は #3420 の別項目（`tei_trace` の `direction` など）で、この変更では扱っていない。
- 実 Host・認可・外部送出は NOT RUN。
