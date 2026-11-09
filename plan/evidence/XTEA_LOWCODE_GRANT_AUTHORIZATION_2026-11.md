# 能力の宣言許可と発行許可の照合確認（2026-11-07）

## 対象

- 実装: `internal/command/grant_authorizer.go`
- 試験: `internal/command/grant_authorizer_test.go`
- 対象版: `928c0a9c4`（`main`）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/command -run 'TestGrantAuthorizer' -count=1 -v
```

## 観測

- `TestGrantAuthorizerRequiresEveryDeclaredPermission` 成功。部分許可では拒否され、欠けている許可（`network`）が拒否メッセージに含まれる。
- `TestGrantAuthorizerAcceptsCoveringGrants` 成功。`read-write`の許可は`write`の要求を満たす。
- `TestGrantAuthorizerRefusesUndeclaredCapability` 成功。無宣言の能力は拒否される（沈黙を許可とみなさない）。
- `TestGrantAuthorizerRefusesInvalidDeclaration` 成功。不正な宣言は判定不能として返り、拒否とは区別される。
- `TestGrantAuthorizerRunsBeforeTheExecutor` 成功。拒否時は`COMMAND_UNAUTHORIZED`で、Executorへ到達しない。
- パッケージ全体は`ok`（`1.026s`）。

## 読み取れること

- 能力が宣言した許可を、実行時に発行済み許可と照合している。
- 部分許可・無宣言・不正宣言は拒否または判定不能として扱われ、実行へ進まない。
- 拒否は`ErrPermissionNotGranted`で識別できる。

## 読み取れないこと

- 許可の発行・保存（誰がいつどの許可を発行したか）。
- CLI・Studioの実行経路への接続。
- 既存システムの実データでの許可判断。
- 実Provider・実データベース・本番配備。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-11`の「権限の宣言」と「実行時検査」は参照実装で確認済み、許可の管理と実行経路への接続は未実装です。
