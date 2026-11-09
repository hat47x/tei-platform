# 読み取り専用datasourceへの書込み拒否の確認（2026-11-07）

## 対象

- 実装: `internal/command/datasource_access.go`
- 試験: `internal/command/datasource_access_test.go`
- 対象版: `63bd8bdee`（`main`）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/command -run 'TestInvokerRefusesWriteThroughReadOnlyDatasource|TestInvokerAllowsWriteThroughReadWriteDatasource|TestInvokerDoesNotGuessADatasource|TestInvokerReportsReadOnlyDatasourceWithTheDeclaration' -count=1 -v
```

## 観測

- `TestInvokerRefusesWriteThroughReadOnlyDatasource` 成功。Applicationが`access read`と宣言したdatasourceへの書込みCommandは`COMMAND_DATASOURCE_READ_ONLY`で拒否され、Executorへ到達しない。
- `TestInvokerAllowsWriteThroughReadWriteDatasource` 成功。同じCommandが`access read-write`では実行される。
- `TestInvokerDoesNotGuessADatasource` 成功。対象Dataを覆うdatasource宣言が無い場合は拒否しない（推測で結び付けない）。
- `TestInvokerReportsReadOnlyDatasourceWithTheDeclaration` 成功。拒否メッセージにApplicationのalias（`sales`）、Database（`sales_core`）、対象Data（`sales.order`）が含まれる。
- パッケージ全体は`ok`。

## 読み取れること

- 検査は宣言に基づき、データベースへ接続しない。
- 読み取り専用の宣言がある場合、書込みは実行前に止まる。
- 宣言が無い場合は「許可」ではなく「判断しない」として扱い、推測で拒否も許可もしない。

## 読み取れないこと

- データソース側での実行時強制（三段目の検査）。
- 実データベースの権限・ロールとの一致。
- `access write`（読取不可）の宣言に対する読取りの扱い（reference Patternの既存検査の範囲）。
- 実Provider・本番配備。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-6`の「宣言」と「実行前」は参照実装で確認済み、実行時の強制は未実装です。
