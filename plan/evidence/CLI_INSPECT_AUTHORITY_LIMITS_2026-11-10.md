# `inspect` だけが「この応答が何を許さないか」を表明していなかった（修正）

- 記録日: 2026-11-10
- 対象: `internal/operations/inspect.go`、`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md`
- 関連: #1178（C2 の開示境界）、#3400、ADR-0004、`plan/design/CLI_C2_DISCLOSURE_BOUNDARY_DESIGN.md`

## 観測（修正前）

3つの読取り投影を同じ Workspace で実行し、`result` のキーを比べました。

| コマンド | `authorityLimits` |
| --- | --- |
| `model catalog` | あり（4項） |
| `trace` | あり（3項） |
| **`inspect`** | **なし** |

`inspect` は `sources.references` に **Source のパスと件数**を返す最も具体的な投影です。`trace` は `source`／`relatedSources` に同じ情報を持つため「位置は意味の受理でも書込み権限でもない」と表明していますが、`inspect` だけが無表明でした。

`docs/getting-started.md` と契約文書は `inspect` の出力形を説明していますが、この限界については何も書いていませんでした。

## 変更

- `SubjectInspection` に `authorityLimits`（`[]string`）を追加した。
- 内容は**固定の4項**で、呼出し元ごとの判定ではない。
  - `inspection-is-a-derived-read-only-projection`
  - `source-location-is-not-semantic-acceptance`
  - `source-location-is-not-write-authority`
  - `local-read-is-not-disclosure-permission`
- 契約文書 §4 に追記した。**4項目目は認可の判定結果ではない**ことを明記した: ローカルで読めることは別の相手へ渡してよいことを意味せず、開示の可否を決めるのは既存の Authorization Capability である（設計文書の「第二 Policy Engine を作らない」に従う）。

## 検証

- `gofmt -l internal`: 空、`go build ./...`: PASS
- `go test ./internal/operations ./internal/inspectoradapter ./tools/reference-mcp-readonly-adapter -count=1`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- `go test ./cmd/tei -run 'TestContractDocument' -count=1`: PASS
- 追加試験 `internal/operations/inspect_authority_limits_test.go`:
  - 4項が順序どおり返ること。
  - ワイヤ上の名前が `authorityLimits` であること。
  - Source を持たない subject（`std.text`）でも表明が付くこと（「位置が見つかったか」ではなく「応答が何を許さないか」の表明であるため）。
  - 許可リスト外の項目が混ざらないこと（認可判定を主張しないことの固定）。
- 負例: `AuthorityLimits` の代入を `nil` にすると `TestInspectStatesWhatItDoesNotGrant` が失敗する（コードは戻した）。
- 実測（実バイナリ2つ）: CLI の `inspect` が4項を返し、MCP Adapter の同じ呼出しと**応答文書が完全一致**。

## 主張しないこと

- **開示の可否を判定していない。** `-scope` は認可scopeでも tenant 境界でもなく、`inspect`／`trace` は信頼できるローカル読取りの候補のままである。
- 実 Provider・実 Host・複数 Transport での fail-closed 実測、監査、Cache、送信先別 egress は未実施（#1178 の残り）。
- 保留ブランチ `park/issue-1178-c2-reference-unverified` の Reference は再評価していない。
- 実 Host・認可・外部送出は NOT RUN。
