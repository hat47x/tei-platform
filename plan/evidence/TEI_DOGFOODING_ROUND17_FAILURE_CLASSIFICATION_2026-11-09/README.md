# 必須引数の不足が3つの符号に分かれていた（2026-11-09、第17ラウンド）

- 対象: [#3733](https://github.com/hat47x/tei-platform/issues/3733)、[#3420](https://github.com/hat47x/tei-platform/issues/3420) CAPI-2b、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3「失敗」・§5 T05。
- 前提: [第10ラウンド](../TEI_DOGFOODING_ROUND10_CAPI2A_PARITY_2026-11-09/README.md)で「必須引数の欠落だけが層の違い」と記録し、#3420 の未決として残していた。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の比較実測と、失敗符号の統一**。

## 1. 目的

「引数の不足」という1つの条件が、コマンドによって別の符号で返っていないかを全コマンドで確かめ、揃える。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source（修正前） | `3c8bebefe`（`origin/main`） |
| CLI実行物（修正前） | `sha256:9ef1f8b6bc941b438a5aa1434cff6de761c0820eeab8a7695f64620a91cc9538` |
| CLI実行物（修正後） | `sha256:782fc6d2cd88bfcb16a28c3e9d45b75fb1a6c347c4654775ca404f717477d9d6` |
| MCP実行物 | `sha256:ef7f11d407074510bd3908244955d1d03962d17b847c9267938e582bec1974b4` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt`・`01-binaries.txt` | 対象 commit と修正前の実行物 |
| `10-cli-cases.txt` | 修正前: 11コマンドの失敗応答（生JSON） |
| `11-validate-noargs.out`・`12-validate-noargs.err` | 修正前: `validate` 引数無しの stdout／stderr |
| `13-validate-json-noargs.out`・`14-validate-json-noargs.err` | `validate -json` 引数無し（構造化される） |
| `20-mcp-cases.txt` | MCP の同じ入力 |
| `30-…` | 修正後の実行物 |
| `31-fixed-binary.txt`・`32-after-cases.txt` | 修正後の11コマンド |

## 4. 観測（修正前）

| 呼出し | 終了コード | `error.code` |
| --- | --- | --- |
| `inspect`（`-subject-id` 無し） | 2 | **`INVALID_SUBJECT`** |
| `inspect`（`-workspace` 無し） | 2 | **`INVALID_SUBJECT`** |
| `trace`（`-scope` 無し） | 2 | `INVALID_ARGUMENT` |
| `model catalog`（`-workspace` 無し） | 2 | `INVALID_ARGUMENT` |
| `source-index`（`-workspace` 無し） | 2 | `INVALID_ARGUMENT` |
| `requirements`（入力無し） | 2 | `INVALID_ARGUMENT` |
| `knowledge-context`（`-subject-id` 無し） | 2 | **`KNOWLEDGE_BUILD_FAILED`** |
| `validate`（引数無し・`-json` 無し） | 2 | **JSON を出さない**（stderr に1行） |

同じ `inspect` でも、`-subject-id` に空文字列を渡すと `INVALID_ARGUMENT`（#3605 の修正）。**「書いていない」と「空文字列」が別の符号**という逆向きの不揃いもあった。

MCP は同じ入力すべてを `-32602`（invalid params）で返す。CLI 側の3つの符号は、MCP の1つに対応する。

## 5. 修正

`cmd/tei/cli_argument_error.go` を追加した。`argumentError` 型は「呼出し元が引数を間違えた」ことを表す。

```go
type argumentError struct{ message string }
func (e *argumentError) Error() string { return e.message }
func newArgumentError(message string) error { return &argumentError{message: message} }
```

- `inspect` の引数ガードを `INVALID_SUBJECT` から `resultCodeInvalidArgument` へ。`INVALID_SUBJECT` は「subject の値が使えない」場合に残す。
- `knowledge-context`・`knowledge-qualify`・`knowledge-review-plan`・`knowledge-review-handoff`・`knowledge-review-disclose` の「`-flag` が足りない」を `newArgumentError` に変え、`knowledgeFailure` がそれを `INVALID_ARGUMENT` へ写す。知識の構築に失敗した場合は `KNOWLEDGE_BUILD_FAILED` のまま。

**変えないと決めたもの**（試験で固定した）:

| 条件 | 符号 | 理由 |
| --- | --- | --- |
| `-subject-id` が空白のみ・前後に空白あり | `INVALID_SUBJECT` | 値は渡されたが使えない。名前どおりの意味を保つ |
| 実在しない subject | `SUBJECT_NOT_FOUND`・exit 1 | 入力の誤りではなく「無い」という所見 |
| 要件ファイルが不正 | `KNOWLEDGE_BUILD_FAILED` | 引数の不足ではなく、中身の失敗 |
| `validate`（`-json` 無し） | 非JSONのまま | text 出力の契約。`-json` を付けたときだけ構造化する現在の区別を維持 |

## 6. 修正後の実測

「`-flag` が足りない」11件すべてが `INVALID_ARGUMENT`・exit 2 になった（`32-after-cases.txt`）。`INVALID_SUBJECT`・`SUBJECT_NOT_FOUND`・`KNOWLEDGE_BUILD_FAILED` は上の表のとおり据え置き。

MCP の `-32602` とは層が違うままである。今回揃えたのは**CLI 内部の一貫性**であり、CLI と MCP の符号の統一（`INVALID_ARGUMENT` と `-32602` の対応付け）は #3420 の残りとする。

## 7. 試験

`cmd/tei/cli_failure_classification_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestMissingRequiredFlagIsAlwaysInvalidArgument` | 11通りの欠落がすべて exit 2・`INVALID_ARGUMENT` |
| `TestUnusableSubjectValueStaysInvalidSubject` | 空白のみ・前後空白は `INVALID_SUBJECT` のまま |
| `TestAbsentSubjectIsAFindingNotAnInputError` | 実在しない subject は exit 1・`SUBJECT_NOT_FOUND` |

既存の `TestCLIProcessJSONContract`・`TestResultEnvelopeCommandsJSONContract` の期待値を新しい符号へ更新した（どちらも「引数不足」を固定していた）。

`go test ./cmd/tei -count=1` は ok（121.1s）。gofmt・vet の指摘なし。

## 8. 互換性

`INVALID_SUBJECT` を `INVALID_ARGUMENT` へ変えるのは破壊的である。`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の `inspect` 行は**もともと `INVALID_ARGUMENT` を挙げており `INVALID_SUBJECT` を挙げていなかった**ため、今回の修正は実装を自らの契約へ合わせたものになる。`trace` 行は `INVALID_SUBJECT` を挙げているが、`trace` の引数ガードは元から `INVALID_ARGUMENT` を返しており、この行の `INVALID_SUBJECT` は `operations`／`modelref` の subject 検証に対応する。契約文書の更新は不要と判断した。

## 9. 読み取れないこと（NOT RUN）

- CLI と MCP の符号の対応付け（`INVALID_ARGUMENT` ⇔ `-32602`）。
- HTTP・Desktop の入口。
- 実DB・実Provider・実Host、意味の受理・権限・配備。

## 10. 位置付け

「引数が足りない」という1つの条件を、11コマンドで同じ符号に揃えた。`INVALID_SUBJECT`（値が使えない）・`SUBJECT_NOT_FOUND`（無い）・`KNOWLEDGE_BUILD_FAILED`（構築に失敗）は、それぞれ別の意味を持つものとして残した。この変更は意味の受理や権限を与えない。
