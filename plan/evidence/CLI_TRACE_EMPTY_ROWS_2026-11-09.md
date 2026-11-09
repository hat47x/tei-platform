# `trace -mode full` が参照0件のとき `rows` を出さなかった（修正）

- 記録日: 2026-11-09
- 対象: `internal/modelref/trace.go`
- 関連: #3693、#3456（`source-index` の `references` を `null` から `[]` にした同型の修正）、#3625

## 観測（修正前）

`trace -mode full` で、対象に宣言された参照が0件のとき `result.trace.rows` が**出ませんでした**。`Rows []TraceRow` が `json:"rows,omitempty"` であり、空スライスも省略されるためです。

```json
{"ok":true,"result":{"trace":{"mode":"full","total":0,"unresolved":0, ...}}}
```

`mode=full` は「全行を返す」契約なので、読み手は `rows` の不在を「0件」と「欄が無い（この版の応答）」のどちらとも解釈できませんでした。

## 変更

- `TraceResult` に `MarshalJSON` を追加した。`mode == "full"` のときだけ `rows` を明示的に書き出し、nil／空スライスでも `[]` になるようにした。
- フィールドを**宣言順に明示**して marshal する。埋め込み構造体で後から `rows` を足す方法は JSON のキー順を変え、既存のゴールデン（`tools/reference-cli-trace-benchmark`）を壊したため採用しなかった。
- `mode == "summary"` は従来どおり `rows` を出さない（件数のみを返す契約）。

## 検証

- `gofmt -l internal cmd`: 空、`go vet ./cmd/tei ./internal/modelref`: PASS、`go build ./...`: PASS
- `go test ./internal/modelref ./internal/operations ./tools/reference-cli-trace-benchmark ./tools/reference-mcp-readonly-adapter -count=1`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS（402 秒、全件）
- 追加試験:
  - `internal/modelref/trace_rows_test.go`
    - full・0件で `rows` が `[]` として**存在**すること。
    - summary で `rows` が無く、`total` はあること。
    - 行があるときのキー順が宣言順のままであること（バイト比較をする既存ツールの前提）。
  - `cmd/tei/cli_trace_empty_rows_test.go`
    - 同じ3点を CLI の応答と**実バイナリ**で確認。
- 負例: `MarshalJSON` の `full` 分岐を無効化すると、単体・CLI・実バイナリの3試験が失敗する（コードは戻した）。
- ゴールデン（`tools/reference-cli-trace-benchmark/testdata/golden.json`）は**変更していない**。キー順を保ったため、既存のハッシュがそのまま一致した。

## 主張しないこと

- `summary` に `rows: []` を足していない（契約が「件数のみ」であるため）。
- `source`／`relatedSources`／`projectionUnresolved` など、他の空リストの扱いは変えていない。
- 実 Host・認可・外部送出は NOT RUN。
