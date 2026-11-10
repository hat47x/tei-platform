# MCP の `subjectKind` だけが閉じた集合を宣言せず、CLI の広告と食い違っていた（修正）

- 記録日: 2026-11-10
- 対象: `tools/reference-mcp-readonly-adapter/main.go`
- 関連: #3420（CAPI-2）、#3242（CLI の argumentValues 広告）、#3400

## 観測（修正前）

`inputSchema` の `subjectKind` は `{"type":"string"}` だけで、**閉じた集合を宣言していませんでした**。同じ file の `mode`・`direction` は enum を持っていたため、`subjectKind` だけが抜けている状態でした。

| 入口 | `subjectKind` の広告 |
| --- | --- |
| `tei commands` → `inspect` | 12値（`argumentValues["subject-kind"]`） |
| `tei commands` → `trace` | 10値 |
| MCP `tools/list` → `tei_inspect` | **enum なし** |
| MCP `tools/list` → `tei_trace` | **enum なし** |

MCP の `inputSchema` は client が読む契約です。enum が無いと、client は「どの kind が有効か」を CLI の別経路か prose から推測するしかありません。

## 変更

- `toolList()` の `subjectProps`・`traceProps` の `subjectKind` に `enum` を追加した。
- 値は**手書きせず、操作の閉じた集合をそのまま参照**する（`operations.InspectionSubjectKinds`・`operations.TraceSubjectKinds`）。同じ事実を2か所に書かない。
- `trace` の `subjectKind` は `derivedbridge.SupportedSubjectKinds` を経由するため、投影側の集合が変われば広告も追従する。

## 検証

- `gofmt -l tools/reference-mcp-readonly-adapter`: 空、`go build ./...`: PASS
- `go test ./tools/reference-mcp-readonly-adapter -count=1`: PASS（170 秒）
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 追加試験 `tools/reference-mcp-readonly-adapter/subject_kind_enum_test.go`:
  - `tei_inspect`・`tei_trace` の `subjectKind` に enum があり、操作の集合と一致すること（`tei_inspect` 12値・`tei_trace` 10値）。
  - **過剰宣言の検出**: 集合の要素数が変わったら、CLI の広告とこの固定を同時に更新するよう失敗する。
- 実測（実バイナリ2つ、変更後）:
  - `tei_inspect` enum=12、`tei_trace` enum=10、`mode`・`direction` も一致。**CLI 広告との一致は全項目 `agree=True`。**
  - **広告した22値すべてが受理される**（`UNSUPPORTED_SUBJECT_KIND` を返した値は0件）。
  - enum 外の `bogus` は `tei_inspect`・`tei_trace` とも `UNSUPPORTED_SUBJECT_KIND` で拒否（CLI と同じ符号）。
- 負例: `tei_inspect` の enum を `{"type":"string"}` に戻すと `TestSubjectKindEnumsComeFromTheOperations/tei_inspect` が「no enum」で失敗する（コードは戻した）。

## 併せて確認したが変更していないこと

同じ `tools/list` と CLI 広告を突き合わせ、他は一致していました。

- `mode`（`summary`/`full`）・`direction`（`outgoing`/`incoming`/`both`）は MCP と CLI で同一。
- catalogue が列挙する72 subject の全 kind を `inspect` と `trace` に投げ、拒否は0件。catalogue・inspect・trace の3者は kind の扱いで一致。
- `tei commands` の `inspect` は `semantic-candidate-key`・`semantic-functional-dependency` を広告し、`trace` は広告しない。両者とも実際に受理するため、広告は正しい（差は投影の対象範囲の違い）。

## 主張しないこと

- MCP の**実行時**の kind 検証は変えていない（元から `internal/operations` の判定を使っている）。
- `inputSchema` の他の欄（`description`・`required` 等）は触っていない。
- 実 Host・認可・外部送出は NOT RUN。
