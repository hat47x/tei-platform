# 操作面・成果物・束縛実行の検証記録（2026-11-02）

- 状態: **検証の記録**。合否の判定ではなく、実行したコマンドと結果を残すものです。
- 対象: ブランチ `work/upstream-connection-2026-11`（`origin/main` 上、rebase 済み）
- 目的: 統合の判断材料として、変更範囲の健全性を確かめる

## 1. 実行した確認

| 確認 | コマンド | 結果 |
| --- | --- | --- |
| 全体のビルド | `go build ./...` | 成功（出力なし、exit 0） |
| 内部パッケージの静的解析 | `go vet ./internal/...` | 成功（出力なし、exit 0） |
| 書式 | `gofmt -l internal tools` | 出力なし |
| 影響パッケージの試験 | `go test ./internal/artifact ./internal/operations ./internal/driver ./internal/semantic/... ./tools/repository-plan-check -count=1` | すべて `ok` |
| 参照用作業領域 | `go run ./cmd/tei validate -workspace reference/workspace` | `ok`（既存の警告のみ） |
| 負例の作業領域 | `go run ./cmd/tei validate -workspace reference/standard-crud/workspace` | `ok` |
| 文書の検査 | `go test ./tools/repository-plan-check -count=1` | `ok` |
| 実DBの競合（枠組み） | `go test ./tools/reference-expected-state-concurrency -count=1` | **スキップ**（`TEI_TEST_POSTGRES_DSN` 未設定） |

## 2. この記録が示すこと

- 変更範囲（操作面の導出、能力への束縛実行、設計成果物の生成と整合検査）のビルド・静的解析・試験が通ること。
- 参照用作業領域と負例の作業領域の両方が、`validate` で読めること。

## 3. この記録が示さないこと

- **実データベースでの競合の防止**。実装は合成の保存領域で確かめただけで、PostgreSQL での並行更新は未実行です。
- **意味の妥当性**。整合検査は成果物同士の矛盾を見るだけで、意味が正しいことは保証しません。
- **受入シナリオの達成**。実装の存在を、判定や受理の証拠にしません。
- **未実行の範囲**: `#1129`・`#1130`・`#2144` は設計のみです。Scenario に書き込みの語彙はありません。

## 4. 再現

```sh
go build ./...
go vet ./internal/...
gofmt -l internal tools
go test ./internal/artifact ./internal/operations ./internal/driver ./internal/semantic/... ./tools/repository-plan-check -count=1
go run ./cmd/tei validate -workspace reference/workspace
go run ./cmd/tei validate -workspace reference/standard-crud/workspace
```
## 5. 期間の状態の検証（2026-11-03 追記）

期間の状態（#2144）を P1〜P3 で実装し、次の範囲を確かめました。

| 確認 | コマンド | 結果 |
| --- | --- | --- |
| 宣言の解析 | `go test ./internal/syntax -run Period` | 通る（`period` の各項目を読み取る） |
| 検証 | `go test ./internal/validation -run Period` | 通る（正例は診断なし、負例は5種の診断） |
| 操作面への反映 | `go test ./internal/operations -run Period` | 通る（`period`・`periodRejects`・前提 `period:<id>:open`） |
| 成果物への反映 | `go test ./internal/artifact -run Period` | 通る（機能定義書の期間行） |
| 既存作業領域への影響 | `go run ./cmd/tei validate -workspace reference/workspace` | `ok`（期間を宣言していないため影響なし） |

**確かめていないこと**: 閉じた期間の**実行時の拒否**。操作面は前提を宣言しますが、実行の入口で前提を検査する層はまだありません。実データベースでの期間の扱いも未確認です。

**設計判断（仮承認D-2）**: 期間は意味の層（`data`・`persist` の宣言）に閉じ、CLI の段階は増やしていません。閉じた期間を `human-gated` に読み替えず、自律区分（誰が判断するか）と期間の状態（いつ実行できるか）を別の軸として扱います。