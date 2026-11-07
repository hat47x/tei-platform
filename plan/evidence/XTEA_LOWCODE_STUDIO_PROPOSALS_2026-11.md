# Studio検証結果の修正案表示の確認（2026-11-07）

## 対象

- 実装: `internal/studiohost/readonly.go`（`ValidationResponse.proposals`）、`cmd/tei-studio/assets/view-format.mjs`（`validationSections`）
- 試験: `internal/studiohost/validation_proposal_test.go`、`cmd/tei-studio/test/view-format.test.mjs`
- 対象版: `bab43e9bb`（`main`）
- 環境: Windows、Go `1.27.0`、Node.js `24.19.0`

## 実行

```powershell
go test ./internal/studiohost -run 'TestValidateWorkspacePairsProposalsWithTheReadRevision' -count=1 -v
node --test cmd/tei-studio/test/view-format.test.mjs
```

## 観測

- `TestValidateWorkspacePairsProposalsWithTheReadRevision` 成功。検証結果の案の件数が診断件数（直接診断を含む）と一致し、各案の`revision`が`worldFingerprint`と一致し、`accepted`は`false`。版を変えた受理は拒否される。
- `view-format`の「validation view lists diagnostics and unanswered proposals」成功。診断表と「修正案（未受理）」表が出て、案の状態は「未受理」と表示される。
- `view-format`の7件すべて成功（既存6件を含む）。

## 読み取れること

- Studioの検証結果は、CLIと同じ版付きの未受理案を返す。
- 画面は、診断と案を別の表として出し、案が受理済みでないことを明示する。

## 読み取れないこと

- 案を受理する操作（`validation.Accept`を呼ぶ画面・API）は未実装。
- Wails RPCの実HTTP経路での確認。
- 案の内容の妥当性（案は診断・対象・版の記録で、修正パッチではない）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-9`の「診断の版結合・自動受理しない」と`AC-12`の「生成物を未受理で置く」の接点は参照実装で確認済み、受理操作は未実装です。
