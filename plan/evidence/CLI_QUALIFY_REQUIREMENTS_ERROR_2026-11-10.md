# 読めない `-qualification-requirements` が `KNOWLEDGE_BUILD_FAILED` になっていた（修正）

- 記録日: 2026-11-10
- 対象: `cmd/tei/knowledge_context.go`
- 関連: #3943、#3944（help 文言。統合済み）、#3749（引数不足の符号統一）

## 観測（修正前）

`-qualification-requirements` に読めないパスや JSON 文字列を渡すと、**`KNOWLEDGE_BUILD_FAILED`** が返っていました。これは「知識の構築に失敗した」という符号で、引数の誤りとは別の原因を示します。

| 入力 | exit | `error.code`（修正前） |
| --- | ---: | --- |
| 存在しないパス | 2 | `KNOWLEDGE_BUILD_FAILED` |
| JSON 文字列を値として渡す | 2 | `KNOWLEDGE_BUILD_FAILED` |
| 形の違う JSON ファイル | 2 | `KNOWLEDGE_BUILD_FAILED` |
| `-qualification-requirements` を省く | 2 | `INVALID_ARGUMENT` |

Issue #3943 の報告者が3回失敗したのはこの状態で、「引数の誤り」と「モデルの問題」を区別できませんでした。

## 変更

`decodeStrictKnowledgeJSONFile` の2か所を `newArgumentError` にした。

- `os.Open` の失敗 → `-qualification-requirements must name a readable JSON file: <理由>`
- デコードの失敗 → `-qualification-requirements is not the expected JSON document: <理由>`

`knowledgeFailure` は既に `argumentError` を `INVALID_ARGUMENT` に写す仕組みを持っていたため（#3749 で導入）、符号の対応表を増やさずに済みました。`knowledge-qualify` と `knowledge-review-handoff` は同じ loader を使うので両方に効きます。

## 検証

- `gofmt -l cmd`: 空、`go vet ./cmd/tei`: PASS、`go build ./...`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 追加試験 `cmd/tei/qualification_requirements_error_test.go`:
  - 存在しないパス → `INVALID_ARGUMENT`、メッセージが flag を名指す
  - `knowledge-review-handoff` でも同じ
  - 形の違う JSON → `INVALID_ARGUMENT`、「expected JSON document」と述べる
  - **逆向きの区別**: 読める requirements ＋ 読めない Workspace は `KNOWLEDGE_BUILD_FAILED` のまま
  - 正常系は `ok:true`
- 負例: `newArgumentError` を素の `err` に戻すと `error.code="KNOWLEDGE_BUILD_FAILED", want "INVALID_ARGUMENT"` で失敗する（コードは戻した）。
- 実測（実バイナリ）: 3ケースすべて `INVALID_ARGUMENT`、真の構築失敗は `KNOWLEDGE_BUILD_FAILED`、正常系は `ok:true`。

## 主張しないこと

- Issue の期待2番（requirements の**スキーマを `commands` で広告する**）は実施していない。`argumentType` は閉じた集合のための欄であり、ファイルパスや文書形を載せるには契約の追加が要る。Issue 自身が「1だけで私の失敗は防げた」と書いており、1（help 文言、#3944）と3（符号、本変更）で実害は解消する。
- requirements の文書形（`expectedKind`・`application`・`requiredDependencies`）は依然として `internal/knowledgecontext/qualification.go` を読む必要がある。
- 実 Host・認可・外部送出は NOT RUN。
