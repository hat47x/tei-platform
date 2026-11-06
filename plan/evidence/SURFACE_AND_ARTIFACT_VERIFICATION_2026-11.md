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
| 実DBの競合 | `TEI_TEST_POSTGRES_DSN=... go test ./tools/reference-expected-state-concurrency -count=1 -v` | `ok`（PostgreSQL 16.14、2件。初回の試験同期不足を修正） |

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
## 6. 期間の状態の通し確認（2026-11-03 追記）

期間の拒否が、宣言から実行入口まで一本につながることを確かめました。

| 確認 | コマンド | 結果 |
| --- | --- | --- |
| 宣言から拒否まで | `go test ./internal/execution -run WorkspacePeriod` | 通る（`period` → 操作面 → 束縛 → ガード → driver を呼ばない拒否） |
| ガードの単位 | `go test ./internal/operations -run 'GuardCapability|CheckPeriods'` | 通る（操作全体・能力単位・出所経由） |
| CLI の判定 | 一時作業領域で `tei periods -period-state-file` | `closed` → 2件拒否・終了コード1、`open` → 0件・0、未知の状態 → エラー |
| 状態ファイル | `go test ./cmd/tei -run PeriodStateFile` | 通る（読取り・未知の状態の拒否・欠落の拒否） |

**確かめていないこと**
- 実際の能力呼出しを行うツールへの適用（`tools/reference-data-first-persistence` は未適用）
- 実データベースでの期間と競合の確認（`TEI_TEST_POSTGRES_DSN` 未設定）
- 期間の状態を誰が・いつ記録するかの運用（現状は JSON ファイルを運用者が置く想定）
## 7. 効果の規則の確認（2026-11-03 追記）

`effect-rule`（#1129）を P1〜P3 で実装し、ライブラリの範囲で確かめました。

| 確認 | コマンド | 結果 |
| --- | --- | --- |
| 宣言の解析 | `go test ./internal/syntax -run EffectRule` | 通る |
| 検証（6診断） | `go test ./internal/validation -run EffectRule` | 通る（正例は診断なし、負例は該当の診断） |
| 操作面への反映 | `go test ./internal/operations -run EffectRules` | 通る（束縛された操作だけが規則を載せる） |
| 成果物への反映 | `go test ./internal/artifact -run EffectRules` | 通る（`effect-rule` 行） |
| 結末で証人にならない効果 | `go test ./internal/execution -run EffectsOutsideTransaction` | 通る |
| 実行と観測の分離 | `go test ./internal/execution -run RunEffect` | 通る（成功コードだけでは `Observed` にならない） |
| 記録 | `go test ./internal/effectlog` | 通る（往復・欠落・壊れた行） |
| 補償の選択と実行 | `go test ./internal/execution -run 'Compensations|Compensate'` | 通る（同一の能力・同一のデータだけ） |
| 観測（SQL） | `go test ./internal/execution -run SQLEffectObserver` | 通る（件数・零件・読取り失敗・未設定） |

**確かめていないこと**
- 実データベースでの効果の実行と観測（`TEI_TEST_POSTGRES_DSN` 未設定）
- 効果の対象を特定する引数の組み立て（`Args` の実装）
- 再試行の上限、補償の完了を待つ境界
## 8. 実行計画の確認（2026-11-04 追記）

`run-plan`（#1130）の P1〜P2 を実装し、ライブラリの範囲で確かめました。

| 確認 | コマンド | 結果 |
| --- | --- | --- |
| 宣言の解析 | `go test ./internal/syntax -run RunPlan` | 通る |
| 検証（8診断） | `go test ./internal/validation -run RunPlan` | 通る（正例は診断なし、負例は該当の診断） |
| 実行の同定・判定 | `go test ./internal/runlog` | 通る（開始・再開・拒否・fence・壊れた行） |
| 実行ループ | `go test ./internal/execution -run RunPlan` | 通る（順次実行・途中停止・完了の記録・再開） |
| 無人の停止 | `go test ./internal/execution -run 'Unattended|Attended'` | 通る（確かめる手段が無ければ `blocked`） |
| 期間との接続 | `go test ./internal/execution -run PeriodPreconditions` | 通る（閉 → blocked、開 → 実行、期間なし → 素通り） |
| 効果の前提との接続 | `go test ./internal/execution -run EffectPreconditions` | 通る（観測不能 → blocked、観測可能 → 実行） |
| 停止条件の合成 | `go test ./internal/execution -run CombinePreconditions` | 通る（期間が開でも効果が観測不能なら止まる） |
| 出力の投影 | `go test ./internal/execution -run ProjectOutputs` | 通る（件数・部分失敗・手段なし・出力なし） |
| 成果物への反映 | `go test ./internal/artifact -run RunPlans` | 通る（`run-plan` 行、省略項目は出さない） |

**確かめていないこと**
- 実データベースでの計画の実行
- 出力（`output`）の生成
- 効果の前提を停止条件へ接続すること
- 無人実行の承認（誰が無人を許したか）の記録
## 9. 三つの宣言を通した実行計画の確認（2026-11-06 追記）

作業領域のテキストから、期間・効果の前提を持つ実行計画が**無人で止まる／進む**ことを通しで確かめました（`internal/execution/runplan_workspace_test.go`）。

| 確認 | 結果 |
| --- | --- |
| 期間が閉じた作業領域で `unattended` の計画 | `blocked`（step は一度も呼ばれない）。実行記録も `blocked` |
| 期間が開いた作業領域 | 実行して `completed` |
| 確定後効果を観測できない作業領域（期間は開） | `blocked`。理由に規則の識別子 `s.order.stock` を含む |

この三つは、期間の宣言（#2144）・効果の規則（#1129）・実行計画（#1130）が**同じ操作面の上で噛み合う**ことを示します。合成は `CombinePreconditions` で行い、どれか一つでも満たされなければ実行しません。
## 10. 実データベースの確認手順（2026-11-06 追記）

実データベースでの確認は**まだ行っていません**。手順と、確かめること・確かめないことを固定しました（[実データベースでの確認手順](REAL_DATABASE_VERIFICATION_PROCEDURE_2026-11.md)）。

- 入口は `tei dbcheck`（DSN 未設定なら終了コード 2 で、何も実行しない）。
- 試験は `tools/reference-expected-state-concurrency`。
- 確かめるのは一つの行と `READ COMMITTED` の範囲。分離水準の変更・実際の実行経路・性能は**確かめない**と明記。
## 11. 四つの前提を持つ実行計画の確認（2026-11-06 追記）

期間・効果・権限の停止条件を**すべて備えた**実行計画が、どれか一つでも欠ければ止まることを一つの試験で固定しました（`internal/execution/runplan_all_preconditions_test.go`）。

| 欠けているもの | 結果 |
| --- | --- |
| 期間が閉じている（他は揃っている） | `blocked`（理由に "closed"）。step は呼ばれない |
| 確定後効果を観測できない（期間は開） | `blocked`（理由に `s.order.stock`） |
| 権限の付与が無い（期間は開、観測は可能） | `blocked`（理由に `db:5432`） |
| すべて満たす | 実行して `completed` |

四つを `CombinePreconditions` で合成し、どれか一つでも満たされなければ**実行しない**ことを示しています。**無人実行の前提は、期間の状態・確定後効果の観測性・権限の付与のすべて**です。