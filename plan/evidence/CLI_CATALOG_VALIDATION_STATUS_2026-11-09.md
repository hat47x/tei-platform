# `model catalog` が検証NGの Workspace でも成功を返し、検証成否を判別できなかった（案A で修正）

- 記録日: 2026-11-09
- 対象: `internal/operations/model_catalog.go`、`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md`
- 関連: #3674、#3420、#2822、#3181

## 観測（修正前）

未定義の型を参照するファイルを含む Workspace で、`validate` は失敗するのに `model catalog` は成功し、結果から検証成否を読み取れませんでした。

| 実行 | 結果 |
| --- | --- |
| `validate -workspace <ws> -json` | `ok:false`、`UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE`、exit 1 |
| `model catalog -workspace <ws>` | `ok:true`、exit 0、検証に関する欄なし |

両者の `worldFingerprint` は同じ値なので、指紋の一致から検証成否は分かりません。

## 変更（選択肢 A）

- `ModelCatalog` に `validation`（`ok`・`errorCount`・`warningCount`）を追加した。
- 判定は `validate` と同じ規則（error 重大度の診断が無く、error 重大度の source-root finding も無い）で、同じ `workspace.Result` から計算する。
- **`model catalog` は検証NGでも成功のまま**（`ok:true`・exit 0）にした。編集中のモデルを閲覧する用途を妨げないため。健全性は `validation.ok` で読む。
- 診断の詳細は複製しない。`validate -json` が返す。
- 契約文書 §4 に規則を追記した。

## 検証

- `gofmt -l internal cmd`: 空、`go build ./...`: PASS
- `go test ./internal/operations -count=1`: PASS
- `go test ./tools/reference-mcp-readonly-adapter -count=1`: PASS（MCP も同じ `CatalogWorkspace` を使うため欄が自動的に揃う）
- 追加試験 `internal/operations/model_catalog_validation_test.go`:
  - 健全な Workspace: `validation.ok:true`、error 0・warning 0。
  - 未定義型参照: `validation.ok:false`、error 1。
  - `ValidateWorkspace` との一致（`ok` と件数の両方）。
  - 検証NGでも `entries` が返り、`validation.ok:false` になること。
- 実測（実バイナリ）: 健全 `{ok:true,errorCount:0,warningCount:0}`、未定義型 `{ok:false,errorCount:1}`、`reference/workspace` `{ok:true,warningCount:3}` で、いずれも `validate` の結果と件数が一致。
- 契約文書の一致試験（`TestContractDocument`）PASS。

## 主張しないこと

- 診断の一覧・メッセージは `model catalog` に載せていない。
- MCP 側の入力スキーマや応答の形は変えていない（同じ投影を包むため `validation` がそのまま現れる）。
- Studio が `validation` を表示するかは決めていない（この変更は欄の追加のみ）。
- 実 Host・認可・外部送出は NOT RUN。
