# 必須引数の不足が3つの符号で返っていた（統一）

- 記録日: 2026-11-09
- 対象: `cmd/tei/cli_inspect.go`、`cmd/tei/cli_result.go`、`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md`
- 関連: #3733、#3420（CAPI-2b）、#3605（空文字列の拒否）、#2354

## 観測（修正前）

「フラグを書いていない」という同じ失敗が、コマンドによって別の符号で返っていました。

| 呼出し | exit | `error.code` |
| --- | ---: | --- |
| `inspect`（`-subject-id`／`-subject-kind`／`-workspace` 無し） | 2 | `INVALID_SUBJECT` |
| `trace`／`model catalog`／`source-index`／`requirements` | 2 | `INVALID_ARGUMENT` |
| `knowledge-context`（`-subject-id` 無し） | 2 | `KNOWLEDGE_BUILD_FAILED` |
| `knowledge-qualify`・`knowledge-review-*`（同じ） | 2 | `KNOWLEDGE_BUILD_FAILED` |

`INVALID_SUBJECT` は「subject が不正」と読めるため、`-workspace` の付け忘れを subject の問題と誤診します。`KNOWLEDGE_BUILD_FAILED` は知識構築の失敗と同じ符号になり、原因を切り分けられません。

## 変更

- `inspect` の引数ガードを `INVALID_ARGUMENT` にした。`INVALID_SUBJECT` は「値は渡されたが subject として不正」に残す。
- `requireKnowledgeSubject` を追加し、`knowledge-context`・`knowledge-qualify`・`knowledge-review-plan`・`knowledge-review-handoff`・`knowledge-review-disclose` の `-subject-kind`／`-subject-id` 欠落を `INVALID_ARGUMENT` で返すようにした。`KNOWLEDGE_BUILD_FAILED` は構築失敗だけに残る。
- `validate`（`-json` 無し）は従来どおり標準エラー1行、`-json` 付きは `INVALID_ARGUMENT` の1 JSON という既存の区別を維持。
- 終了コード2は変えていない（符号だけを揃えた）。
- 契約文書 §4 に規則を追記した。

## 検証

- `gofmt -l cmd`: 空、`go vet ./cmd/tei`: PASS、`go build ./...`: PASS
- `go test ./cmd/tei -run 'TestInspect|TestKnowledge|TestArgumentFailure|TestContractDocument|TestDiscovery' -count=1`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 追加試験 `cmd/tei/cli_missing_argument_code_test.go`:
  - `inspect`×2、`trace`、`knowledge-context`・`knowledge-qualify`・`knowledge-review-plan`・`knowledge-review-handoff`・`knowledge-review-disclose`、`model catalog`、`requirements` の10経路すべてが exit 2・`INVALID_ARGUMENT` であること。
  - 値が渡された空白 subject は引数不足の経路（`inspect requires …`）で報告されないこと（`INVALID_SUBJECT` の役割を保つ）。
- 負例: `knowledge-context` のガード呼び出しを無効化すると `knowledge-context_without_-subject-id` が失敗する（コードは戻した）。
- 実測（実バイナリ）: 12の欠落ケースすべてが exit 2・`INVALID_ARGUMENT`。

## 主張しないこと

- `INVALID_SUBJECT` を廃止していない。subject の値が不正な場合の符号として残している。
- MCP 側の符号は変えていない（もともとすべて `-32602`）。
- `validate`（`-json` 無し）の標準エラー出力は、既存の決定として維持した。
- 実 Host・認可・外部送出は NOT RUN。
