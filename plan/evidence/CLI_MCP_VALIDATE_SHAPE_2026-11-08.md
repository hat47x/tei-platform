# `validate` の応答が CLI と MCP で別の文書になっていた（記録）

- 記録日: 2026-11-08
- 対象: `cmd/tei/cli_validation.go`、`internal/operations/validation_report.go`、`tools/reference-mcp-readonly-adapter`
- 関連: #3420（CLI/MCP の結果・失敗・成否の一致）、#1158、ADR-0048。

## 観測（修正前）

同じ Workspace に対して、CLI と MCP 参照 Adapter が別の文書を返していた。

```text
tei validate -workspace reference/workspace -json
# {"schemaVersion":"tei.cli.validate/v0","ok":true,"diagnostics":[...],"worldFingerprint":...,
#  "sourceIndexFingerprint":...,"proposals":[...],"declarationCounts":{...},"workspace":{...}}

MCP tei_validate
# {"schemaVersion":"tei.cli.validate/v0","ok":true,"result":{"worldFingerprint":...,
#  "sourceIndexFingerprint":...,"diagnostics":[...],"declarationCounts":{...},"sourceRootFindings":[]}}
```

差分は3つ。

1. **形**: CLI は最上位に `diagnostics` などを置き、MCP は `result` で包んでいた。
2. **失敗の符号**: MCP の `error.code` は数値 `1` で、文字列の符号を `message` の先頭へ埋めていた。CLI は `error.code` に文字列（`WORKSPACE_LOAD_FAILED` など）を置く。
3. **成否の対応**: MCP は `isError` を操作のエラーだけで決めていたため、意味エラー（`diagnostics` に error がある `ok:false`）でも `isError:false` になった。CLI は同じ場合に終了コード1を返す。

## 変更

- `internal/operations/validation_report.go` に `WorkspaceValidationReport` と、その投影 `WorkspaceValidationReportFrom`、失敗文書 `WorkspaceValidationFailureReport` を置いた。CLI が書いていた欄名・並び・並べ替えの鍵をそのまま移し、`WorkspaceValidationResult` からの純粋な変換にした（再読取りなし）。
- `cmd/tei/cli_validation.go` は型を `operations` の別名にし、本文は共有の投影を呼ぶだけにした。CLI の応答はバイト単位で従来と同じ（実測で確認）。
- `tools/reference-mcp-readonly-adapter` は `tei_validate` で同じ投影を返し、`isError` を `!report.OK` にした。失敗時は `operations` の失敗文書（`diagnostics: []` を含む）を返す。JSON-RPC の数値 `error` はプロトコル用に分離したまま。
- `validate_parity_test.go` を追加。実 CLI と実 Adapter を起動し、(a) 正常時の文書が同じ欄・同じ値・同じ診断件数になること、(b) 存在しない Workspace が `isError:true` と CLI と同じ `WORKSPACE_LOAD_FAILED` を返すこと、(c) 重複宣言の意味エラーが両方で失敗（CLI exit 1・`ok:false`、MCP `isError:true`・`ok:false`）になり診断件数が一致することを検査する。

## 検証

- `gofmt -l cmd internal tools`: 空、`go vet ./cmd/tei ./internal/operations ./tools/reference-mcp-readonly-adapter`: PASS、`go build ./...`: PASS
- `go test ./tools/reference-mcp-readonly-adapter -count=1`: PASS（`TestAdapterMatchesTheSameOperationResults` を含む）
- `go test ./cmd/tei -run 'TestValidate|TestCLIProcess' -count=1`: PASS
- `go test ./internal/operations -count=1`: PASS
- 負例: `return report, !report.OK` を `return report, false` に変えると `TestValidateFindingIsAFailedAnswerOnBothTransports` が失敗する（コードは戻した）。
- 実測（CLI）: `reference/workspace` の `validate -json` は修正前と同じ最上位欄、`nosuch` は exit 2 と `WORKSPACE_LOAD_FAILED`。

## 主張しないこと

- MCP の protocol 版・実 Host・認可・外部送出は扱っていない。ここで一致を示したのは、同じローカル参照 Adapter の `tei_validate` と CLI だけである。
- `catalog`・`inspect`・`trace` の MCP 側の形は変えていない（それぞれ CLI と同じ `result` 包みで、既存試験が値の一致を検査している）。未知欄・型不正・上限などの入力検査、`initialize`／通知の網羅は #3420 の残作業。
- `sourceRootFindings` は CLI では省略可能（`omitempty`）、MCP も同じ投影なので同じ扱いになる。空配列を常に出すかは決めていない。
