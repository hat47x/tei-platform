# `inspect` の Workspace 読み込み失敗だけが CLI と MCP で違っていた（修正）

- 記録日: 2026-11-09
- 対象: `cmd/tei/cli_inspect.go`
- 関連: #3420（CAPI-2）、#3400、ADR-0048 §2.3

## 観測（修正前）

4つの tool（`catalog`・`inspect`・`trace`・`validate`）× 5つの Workspace 状態（健全・構文エラー・source-root 欠落・重複宣言・load 失敗）の20通りを実測したところ、**`inspect` の load 失敗2件だけ**が文書不一致でした。

| 入力 | CLI | MCP |
| --- | --- | --- |
| 存在しない Workspace | `WORKSPACE_LOAD_FAILED` / `workspace inspection failed` | `WORKSPACE_LOAD_FAILED` / `resolve Workspace root: …` |
| 構文エラーの Workspace | 同上 | 同上 |

符号は一致していたため `error.code` だけ見る呼出し元は気づけませんが、原因を示す文言が CLI だけ失われていました。`trace`・`catalog`・`validate` は同じ状況で本物の文言を返していました。

## 原因

`cli_inspect.go` の失敗報告は、既定値として `Message: "workspace inspection failed"` を置き、`InspectionError` の分岐だけがメッセージを差し替えていました。`ValidationFailure`（Load・Source Index）の分岐は `Code` だけを変え、メッセージを更新していませんでした。

この既定値自体は必要です。未検証の Provider がホスト固有のパスや資格情報をメッセージへ入れられないようにするためで、既存の2つの安全性試験（`TestCLIInspectDoesNotExposeUnknownProviderErrors`・`TestCLIInspectMasksForgedProviderFailureCodeAndMessage`）がそれを固定しています。

## 変更

- `ValidationFailure` の `StageLoad`・`StageSourceIndex` の分岐で `failure.Message = err.Error()` を設定した。この2段階は**このプロセス自身による Workspace の読み取り**の失敗であり、Provider の文言ではないため、報告して安全かつ MCP と一致させる必要がある。
- `InspectionError` の分岐（未検証 Provider の可能性がある）は従来どおり伏せる。
- 既定の `"workspace inspection failed"` はそのまま残した。

## 検証

- `gofmt -l cmd`: 空、`go vet ./cmd/tei`: PASS、`go build ./...`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 既存の安全性試験2件が引き続き PASS（未検証 Provider の本文は漏れない）。
- 追加試験 `cmd/tei/cli_inspect_load_failure_test.go`:
  - Load 失敗で実際の読み取り文言（対象パスを含む）が返り、汎用文言に置き換わらないこと。
  - 未検証 Provider の文言は今も伏せられること。
  - **実バイナリ**の `inspect -workspace <存在しない>` が本物の文言を返すこと。
- 負例: 段階分岐のメッセージ設定を無効化すると、単体と実バイナリの2試験が失敗する（コードは戻した）。
- 実測（再実行）: 20通りの失敗分類すべてで `documents_equal: true`（不一致0）。成功5文書も完全一致。プロトコル6入力（通知・未知 method・壊れた JSON・バッチ・initialize・ping）も期待どおり。

## 主張しないこと

- `InspectionError` を返す未検証 Provider のメッセージを出すようにはしていない（安全性の既定を維持）。
- Host 初期化失敗（`INSPECTION_PROVIDER_UNAVAILABLE`）は MCP に対応経路が無いため比較していない。
- 実 Host・認可・外部送出は NOT RUN。
