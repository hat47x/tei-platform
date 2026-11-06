# 既存認可規則とCommand境界の接続確認（2026-11-07）

## 対象

- 実装: `internal/security/legacyauthorization/command_authorizer.go`
- 試験: `internal/security/legacyauthorization/command_authorizer_test.go`
- 対象版: `e96166bf3`（`main`）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/security/legacyauthorization -run 'TestCommandAuthorizer|TestInvokerRejectsUngranted' -count=1 -v
```

## 観測

- `TestCommandAuthorizerLetsAGrantedSubjectRun` 成功。付与のある主体はExecutorへ到達する。
- `TestInvokerRejectsUngrantedCommandBeforeExecutor` 成功。PatternがCommandを宣言していても、付与が無ければ`COMMAND_UNAUTHORIZED`でExecutorへ到達しない。
- `TestCommandAuthorizerStopsWhenRulesCannotBeEvaluated` 成功。規則の評価不能は拒否へ丸めず、`ErrNoGrant`ではないエラーで返る。
- 3試験の実行時間はそれぞれ`0.00s`で、パッケージ全体は`ok`。

## 読み取れること

- Command境界は、メニュー・Patternの宣言ではなく`Authorizer`の判断で実行可否を決める。
- 拒否と判定不能は区別される。
- `security.authorize`の既存規則（subject・action・resource）を再利用している。

## 読み取れないこと

- 権限の正本（誰がどの権限を持つか）の管理。
- 既存システムの実データでの認可判断。
- `security.authorize`の`EpochReader`を用いた版照合（試験では未設定）。
- 実Provider・実データベース・本番配備。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-11`の「メニュー非依存の拒否」は参照実装で確認済み、権限の宣言・付与の管理は未接続です。
