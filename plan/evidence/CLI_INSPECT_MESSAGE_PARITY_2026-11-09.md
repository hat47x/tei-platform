# `inspect` の失敗メッセージが CLI と MCP で違っていた（修正）

- 記録日: 2026-11-09
- 対象: `cmd/tei/cli_inspect.go`、`internal/inspectoradapter/{selected,local_host}.go`
- 関連: #3420（CAPI-2）、#3400（MCP 入口）、ADR-0048 §2.3

## 観測（修正前）

同じ Workspace・同じ入力で、`inspect` の失敗メッセージだけが2つの入口で違っていました。

| 入力 | CLI | MCP |
| --- | --- | --- |
| `-subject-kind bogus` | `inspection service reported UNSUPPORTED_SUBJECT_KIND` | `inspect does not support subject kind "bogus"` |
| 存在しない subject | `inspection service reported SUBJECT_NOT_FOUND` | `canonical subject semantic-data:no.such not found` |

`trace`・`model catalog`・`validate` は一致していたため、`inspect` だけが例外でした。

## 原因

通常の CLI 経路（`inspectWorkspaceJSONSelected`）は Host が選んだ実装（`inspectoradapter.Selected`）を束縛します。`cli_inspect.go` はメッセージをそのまま出す条件を「サービスが `builtinCLIInspectionService` であること」にしていたため、Host 経由の実装はすべて「未検証」と扱われ、`inspection service reported <CODE>` に置き換えられていました。MCP Adapter は `operations.InspectSubject` を直接呼ぶので本物の文言が出ていました。

この置き換え自体は必要です（未検証の実装がホスト固有のパスや資格情報をメッセージへ入れられないようにするため）。足りなかったのは「この実装は正本の Operation のエラーをそのまま返す」と宣言する経路です。

## 変更

- `inspectoradapter.Registration` に `CanonicalMessages` を追加し、`Selected.CanonicalInspectionMessages()` で読めるようにした。
- ローカル参照 Host は自分の登録が正本の Operation そのものであるため `CanonicalMessages: true` を設定した。
- `cli_inspect.go` の判定を `canonicalInspectionMessages(service)` にまとめた。組み込みサービスは常に真、Host 経由は宣言があるときだけ真、宣言が無ければ従来どおり伏せる。

## 検証

- `gofmt -l cmd internal`: 空、`go vet ./cmd/tei`: PASS、`go build ./...`: PASS
- `go test ./internal/inspectoradapter ./cmd/tei -count=1 -timeout 30m`: PASS
- 追加試験 `cmd/tei/cli_inspect_message_parity_test.go`:
  - 正本を宣言したサービスはメッセージをそのまま返す。
  - 宣言の無いサービスは `inspection service reported …` に伏せ、`secret` を含む本文が漏れない。
  - 組み込みサービスは常に正本、nil と未宣言は正本でない。
  - **実バイナリ**の `inspect -subject-kind bogus` が `UNSUPPORTED_SUBJECT_KIND` と本物の文言を返す（伏せ字でない）。
- 負例: `canonicalInspectionMessages` の分岐を無効化すると、単体と実バイナリの2試験が失敗する（コードは戻した）。
- 実測（実バイナリ2つ）: `inspect` の2ケース、`trace` の2ケース、`catalog`、`trace` の `STALE_SNAPSHOT` の計6ケースで、CLI と MCP の**応答文書が完全一致**（`documents_equal: true`、文言も一致）。

## 主張しないこと

- `INSPECTION_PROVIDER_UNAVAILABLE` など Host 初期化の失敗は、MCP に対応する経路が無いため比較していない。
- 未検証実装のメッセージを伏せる既定は変えていない（安全性を弱めていない）。
- 実 Host・認可・外部送出は NOT RUN。
