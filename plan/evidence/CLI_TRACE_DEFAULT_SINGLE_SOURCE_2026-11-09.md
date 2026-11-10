# `trace` の既定値が3か所に分かれていた（一元化）

- 記録日: 2026-11-09
- 対象: `internal/modelref/trace.go`、`cmd/tei/main.go`、`cmd/tei/cli_contract_values.go`、`tools/reference-mcp-readonly-adapter`
- 関連: #3605（空文字列の拒否）、#3242（引数値の広告）、#3419（共通契約）

## 観測（修正前）

`trace` の `mode` の既定値が3か所に別々に存在していた。

| 場所 | `mode` の既定 |
| --- | --- |
| CLI の flag 既定値（`cmd/tei/main.go`） | `summary` |
| `tei commands` の `argumentDefaults`（`cmd/tei/cli_contract_values.go`） | `summary` |
| MCP Adapter の省略時（`tools/reference-mcp-readonly-adapter`） | `summary` |
| `modelref.normalizeTraceRequest` の空文字列の扱い | `full` |

CLI と MCP は #3605 以降、明示の空文字列を拒否するため内部の `full` へは到達しない。しかし値の定義が4か所に散っており、片方だけを直すと広告と適用が食い違う構造が残っていた。

## 変更

- `internal/modelref` に `TraceDefaultDirection = "both"`・`TraceDefaultMode = "summary"` を追加し、**広告と適用の単一の出所**にした。
- CLI の flag 既定値・`tei commands` の `argumentDefaults`・MCP Adapter の省略時が同じ定数を読むようにした。
- `normalizeTraceRequest` の空 `Direction` は同じ定数へ寄せた。空 `Mode` は `full` のまま（直接の in-process 呼出し向けの歴史的な値であり、アダプタの既定ではない）と、コメントで区別した。
- `internal/modelref/trace_defaults_test.go` と `cmd/tei/cli_trace_defaults_test.go` を追加。
  - 既定値が閉じた集合の要素であること（広告した既定が不正値として拒否されない）。
  - 省略時の既定と明示の `summary` が同じ trace になること（`mode`・`total`・行数・指紋）。
  - `commandCatalog()` の `argumentDefaults` が定数と一致し、`argumentValues` の中にあること。
  - **実バイナリ**で `-mode` を省いたときの `result.trace.mode` が `TraceDefaultMode` と一致すること。

## 検証

- `gofmt -l cmd internal tools`: 空、`go vet ./cmd/tei ./internal/modelref ./tools/reference-mcp-readonly-adapter`: PASS、`go build ./...`: PASS
- `go test ./cmd/tei -run 'TestHelp|TestContractValues|TestDiscovery|TestTrace|TestArgumentFailures|TestCLIModel' -count=1`: PASS
- `go test ./internal/modelref -count=1`: PASS
- `go test ./tools/reference-mcp-readonly-adapter -count=1`: PASS
- 負例: `argumentDefaults` の `mode` を `"full"` に変えると `TestDiscoveryArgumentValuesMatchImplementation` と `TestHelpShowsTheSameArgumentValuesAsDiscovery/trace_-mode` が失敗する（コードは戻した）。

## 主張しないこと

- `normalizeTraceRequest` の空 `Mode` → `full` は残した。直接の in-process 呼出しの挙動を変えるかは別の判断で、アダプタからは到達しない。
- `trace` 以外のコマンドの既定値を同じ仕組みへ移してはいない（`view` の `format: text` は CLI 内の1か所のみ）。
- 実 Host・認可・外部送出は NOT RUN。
