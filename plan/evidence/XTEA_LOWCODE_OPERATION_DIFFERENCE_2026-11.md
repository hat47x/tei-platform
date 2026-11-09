# 宣言と実行の差の保持確認（2026-11-07）

## 対象

- 実装: `internal/command/applied_check.go`、`internal/command/operation.go`、`cmd/tei/cli_view.go`、`internal/studiohost/command.go`
- 試験: `internal/command/applied_operation_test.go`、`cmd/tei-studio/test/view-format.test.mjs`
- 対象版: `192a33a19`（`main`）
- 環境: Windows、Go `1.27.0`、Node.js `24.19.0`

## 実行

```powershell
go test ./internal/command -run 'TestReferenceDataExecutorReports|TestReferenceDataExecutorCascadeNamesEachRecord|TestInvokerRejectsAppliedOperationOutsideDeclaredKind|TestInvokerReportsMissingAppliedOperation|TestInvokerRecordsAppliedOperations|TestInvokerKeepsCascadeAsAnUnresolvedDifference' -count=1 -v
node --test cmd/tei-studio/test/view-format.test.mjs cmd/tei-studio/test/client.test.mjs
```

```powershell
tei view -workspace examples/xtea-lowcode -application order_entry `
  -pattern order_entry.order_delete -command sales.delete_order `
  -input order_id=ORD-1 -execute -allow-capability sales.delete_order `
  -period-state <open state file> -format json
```

## 観測

- `TestReferenceDataExecutorReportsAppliedOperation` 成功。参照Storeは適用したレコード（種別・対象・キー）を返す。
- `TestReferenceDataExecutorCascadeNamesEachRecord` 成功。カスケードは削除した関連レコードも列挙する。
- `TestInvokerRejectsAppliedOperationOutsideDeclaredKind` 成功。宣言外の種別は`UNDECLARED_COMMAND_EFFECT`で、Commit前に拒否される。
- `TestInvokerReportsMissingAppliedOperation` 成功。宣言効果に対応する操作が無い場合は`MISSING_COMMAND_EFFECT`。
- `TestInvokerRecordsAppliedOperations` 成功。記録に適用操作が残る。
- `TestInvokerKeepsCascadeAsAnUnresolvedDifference` 成功。カスケードは拒否されず、`Differences`へ`additional-scope`として残る（`sales.line`が1件、宣言は`sales.order`）。
- `view-format`と`client`の試験は8件すべて成功。参照実行の行に`differences`件数が出る。
- `examples/xtea-lowcode`の削除実行は成功し、`committed:true`・`persisted:false`・`durability:"none"`が返る。参照Storeが空のため`differences`は出ない（差が無い状態を偽って作らない）。

## 読み取れること

- 宣言と実行の差は、拒否と区別して未解決の差として残る。
- 差は実行記録（`OperationRecord.Differences`）、CLIの実行結果、Studioの`differences`から読める。
- 宣言外の種別は実行後に拒否されるが、その拒否は「適用されていない」と断定せず、効果不明として扱う（`COMMAND_EFFECT_UNKNOWN`）。

## 読み取れないこと

- 実行時に発行されたSQL文・CRUD操作そのものの記録（参照StoreはSQLを発行しない）。
- 行ごとの値の前後関係。
- 実Provider・実データベース・本番配備。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-8`の「差を未解決として保持する」は参照実装で確認済み、発行SQL文の記録は未実装です。
