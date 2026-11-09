# 診断の修正案と受理境界の確認（2026-11-07）

## 対象

- 実装: `internal/validation/proposal.go`、`cmd/tei/cli_validation.go`
- 試験: `internal/validation/proposal_test.go`、`cmd/tei/cli_validation_test.go`
- 対象版: `edae81e1c`（`main`）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/validation -run 'TestPropose|TestAccept' -count=1 -v
go test ./cmd/tei -run 'TestValidateWorkspaceJSONProposesWithoutAccepting' -count=1 -v
```

## 観測

- `TestProposeNeverAccepts` 成功。`Propose`は案を作るだけで、`Accepted`は常に`false`。
- `TestAcceptRequiresTheAcceptedRevision` 成功。同じ版での受理は通り、版が変わると`RevisionMismatchError`で拒否。
- `TestAcceptRefusesMissingRevision` 成功。案・受理側のどちらかが版を持たない場合は拒否。
- `TestAcceptRefusesRepeatedAcceptance` 成功。二重受理を拒否。
- `TestValidateWorkspaceJSONProposesWithoutAccepting` 成功。`tei validate -json`の`proposals`件数が診断件数と一致し、各案の`revision`が`worldFingerprint`と一致し、`accepted`は`false`。

## 読み取れること

- 診断は読み取った版に結び付く。
- 修正案は生成された時点では受理されておらず、受理には版一致と明示的な呼出しが要る。
- 版が変わった後の案は、そのままでは受理できない。

## 読み取れないこと

- 案を採る作成者向けの画面・手順。
- 意味を受理する権限を持つ主体の判定。
- 案の内容が妥当かどうか（案は修正パッチではなく、診断・対象・版の記録）。
- 実Provider・実データベース・本番配備。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-9`の「版への結合」と「自動受理しない境界」は参照実装で確認済み、作成者向けの受理操作は未実装です。
