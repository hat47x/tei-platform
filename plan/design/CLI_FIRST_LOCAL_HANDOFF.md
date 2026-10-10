# CLI-first C0/C1 ローカル検証・引継ぎ

対象: Draft PR #1159 / issues #1155, #1158, #1165。GitHub Actionsは停止中。以下は**未実行の手順**であり、合格を保証しない。

## 1. 作業ツリーを確認する

```sh
git fetch origin
git switch feat/cli-first-discovery-and-json-validation
git status --short
git log -1 --oneline
go version
```

未コミット変更がある場合は、破棄せず隔離する。

## 2. 静的検査と対象テスト

```sh
gofmt -l cmd/tei/cli_contract.go cmd/tei/cli_validation.go cmd/tei/cli_contract_test.go cmd/tei/cli_validation_test.go cmd/tei/cli_process_test.go cmd/tei/source_index.go cmd/tei/validation_output.go internal/operations/validate.go internal/operations/validate_test.go internal/operations/trace.go internal/operations/trace_test.go
go test ./internal/operations ./cmd/tei -count=1
go vet ./internal/operations ./cmd/tei
```

gofmtの出力は空であること。失敗時はまず既存fixtureと新規テストの前提を照合する。テスト結果を保存し、実行せずに合格扱いしない。

## 3. 全体回帰

```sh
go test ./... -count=1
go vet ./...
go build -o ./tei-local ./cmd/tei
./tei-local commands
./tei-local validate -workspace reference/workspace -json
./tei-local source-index -workspace reference/workspace -subject-kind query -subject-id order_entry.recent_orders
```

Windowsでは出力名を `tei-local.exe` に変更する。生成物をコミットしない。終了コード0/1/2の区別、stdoutに単一JSON、stderrに予期しない混入がないこと、既存text validateの互換性を確認する。Referenceのwarningは失敗と同一視しない。

## 4. レビュー重点

- `internal/operations` がCLI依存せず、HTTP/IPCでも再利用できるか。
- Source Indexが唯一の参照元であり、Canonical Worldにファイルパスを混入させていないか。
- `source-index` 全体取得と絞り込みの既存出力互換性（JSON配列の空値表現を含む）。
- 構文エラーに架空のSubjectを付けていないか。
- `commands` が完全なコマンド登録簿と誤認されないか。
- `validate -json` のフラグ解析失敗時に非JSON stderrが出る場合は、契約の範囲を明記するか改善する。
- 読取りコマンドがファイル・lock・cacheを更新しないか。
- C1 `inspect` は未実装。現時点のTraceはソース位置追跡であり、モデル意味論・依存関係追跡ではない。

## 5. マージ・クローズ条件

全体テスト、vet、出力互換性、read-only性、差分レビューが完了するまでDraft解除・mainマージを行わない。既存の並列PRによるmain変更を取り込み、衝突と回帰を再検査する。#1155はC0受入基準を満たした場合にのみクローズする。#1158と#1165は、Web/desktop adapterやinspectが未完成ならオープンのまま残す。

ローカル検証で問題が出たら失敗コマンド、Go version、標準出力/標準エラー、コミットSHAをIssue/PRへ記録する。
