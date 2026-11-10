# `tools/call` の params・name 欠落が誤った理由で拒否されていた（修正）

- 記録日: 2026-11-09
- 対象: `tools/reference-mcp-readonly-adapter`
- 関連: #3420（CLI/MCP の結果・失敗・成否の一致）、#3400（MCP 入口）

## 観測（修正前）

`tools/call` の params を欠落・null・型違いにしたとき、および `name` を欠落・null・空にしたとき、返るメッセージが実際の誤りを指していませんでした。

| 入力 | 修正前のメッセージ |
| --- | --- |
| `params` なし | `invalid tools/call params: unexpected end of JSON input` |
| `params: null` | `unknown tool ""` |
| `name` なし | `unknown tool ""` |
| `name: null` | `unknown tool ""` |
| `name: ""` | `unknown tool ""` |
| `name: 7` | `invalid tools/call params: json: cannot unmarshal number into Go struct field …` |
| `arguments: []` | `invalid arguments: json: cannot unmarshal array into Go value of type map[string]json.RawMessage` |

`unknown tool ""` は「その名前の tool は無い」と読みますが、実際は名前が無いのであり、クライアントの誤りを隠します。デコーダの文言も Go の型名を露出するだけで、何を直せばよいかを示しません。

## 変更

- `callTool` を書き直した。
  - `params` が空: `tools/call requires params with a name`。
  - `params` が `null`: `tools/call requires params with a name, not null`。
  - params がオブジェクトでない、または `name` が文字列でない: `tools/call params must be an object with a string name`。
  - `name` が無い／`null`: `tools/call requires a name`（ポインタで「無い」と「空」を区別）。
  - `name` が空文字列: `tools/call name must not be an empty string`。
  - `arguments` がオブジェクトでない: `tools/call arguments must be an object`。
  - `arguments` の省略と `null` は同じ「引数なし」として扱う（tool は引数なしで動く）。
- すべて JSON-RPC `-32602`（invalid params）で返し、tool 結果の `isError` とは区別したままにした。

## 検証

- `gofmt -l tools`: 空、`go vet ./tools/reference-mcp-readonly-adapter`: PASS、`go build ./...`: PASS
- `go test ./tools/reference-mcp-readonly-adapter -count=1`: PASS
- `call_params_test.go` を追加。
  - 9つの不正入力すべてが `-32602` で、期待するメッセージを含み、`unknown tool ""`・`unexpected end of JSON input`・`cannot unmarshal` を含まないこと。
  - `arguments` の省略・`null`・`{}` の3通りがすべて `tei_validate` の成功になること。
- 負例: `name == nil` を空文字へ置き換える旧挙動へ戻すと、`no_name` と `name_null` が失敗する（コードは戻した）。
- 実測（実バイナリ）: 9入力すべてが上表の新メッセージを返し、正常系は変わらない。

## 主張しないこと

- `params` の他の形（配列の中にオブジェクトがある等）や、`id` の型検証は扱っていない。今回直したのは params/name/arguments の形だけである。
- JSON-RPC の仕様が求める `-32600`（invalid request）と `-32602`（invalid params）の使い分けは、既存の `-32602` のままにした。仕様上の分類を変えるなら別の判断が要る。
- 実 Host・認可・外部送出は NOT RUN。
