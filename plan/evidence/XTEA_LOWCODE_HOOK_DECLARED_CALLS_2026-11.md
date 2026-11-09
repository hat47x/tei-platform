# フックの宣言外呼出し拒否の確認（2026-11-07）

## 対象

- 実装: `internal/hook/declared.go`
- 試験: `internal/hook/declared_test.go`
- 対象版: `5af4dc4f8`（`main`）
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/hook -run 'TestDeclaredHook' -count=1 -v
```

## 観測

- `TestDeclaredHookCallsTheDeclaredCapability` 成功。宣言した能力は呼出し先へ届く。
- `TestDeclaredHookRefusesAnUndeclaredUse` 成功。宣言外の能力は`HOOK_UNDECLARED_CALL`で拒否され、呼出し先へ届かない。
- `TestDeclaredHookRefusesWithoutADeclaration` 成功。宣言の無いフックは何も呼べない。
- `TestDeclaredHookReportsMissingCaller` 成功。呼出し先の不在は`HOOK_IMPLEMENTATION_UNAVAILABLE`で、拒否と区別される。
- `TestDeclaredHookPassesTheCallerError` 成功。呼出し先のエラーはそのまま返る。
- パッケージ全体は`ok`。

## 読み取れること

- フックが到達できる能力は、宣言（`uses`）で決まる。
- 宣言外の呼出しは、実装の裁量ではなく境界で拒否される。
- 宣言の欠落は許可ではなく拒否として扱われる。

## 読み取れないこと

- `hook.Runner`（位相の解決と失敗時の扱い）とCommand境界への接続。
- 実際の能力呼出しの内容・副作用。
- 実Provider・実データベース・本番配備。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「宣言外の能力呼出しの拒否」は境界として実装済み、Command境界への接続は未実装です。
