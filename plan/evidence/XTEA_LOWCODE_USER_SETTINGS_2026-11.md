# 利用者設定の宣言と結合の確認（2026-11）

## 対象

- 実装: `internal/usersetting/usersetting.go`、`internal/validation`の`setting-key`・`setting-value`・`setting-user`役割
- 試験: `internal/usersetting/usersetting_test.go`
- 対象版: `main` の `381ce9dcb` から分岐した`feat/user-setting-declaration-20261109`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/usersetting -count=1 -v
go test ./internal/validation -count=1
```

## 観測

- `TestResolveRequiresKeyAndValue` 成功。鍵または値の役割が無いDataは、**役割名つきで**拒否される。未宣言のDataも拒否。
- `TestBindRequiresAKey` 成功。鍵の空の行は行番号つきで拒否される。
- `TestValuePrefersTheUsersRow` 成功。利用者`alice`の行`200`が共有の既定`50`に勝ち、行を持たない`bob`は共有の`50`を読む。
- `TestValueReportsAnUndeclaredKey` 成功。`page_sise`のような綴り誤りは空文字ではなく**報告**になる。
- `TestValueReportsAKeyDeclaredForAnotherUser` 成功。他利用者だけが宣言する鍵は、空ではなく報告になる。
- `TestNewSettingsRefusesADuplicateKey` 成功。同じ鍵（同じ利用者）の2行は拒否される（行順に依存させない）。
- `TestKeysAreStable` 成功。宣言済みの鍵一覧が整列して返る。
- `internal/validation`は`ok`。

## 読み取れること

- 設定の鍵は**モデルが宣言**する。X-TEAは`setVariant`でシステム変数、`setUserValueOf`で利用者値を任意のキーで書けた（`Session.java` L2112、`XFScriptable.java` L56–60）ため、存在する鍵の一覧がどこにも無く、綴り誤りは「未設定」として読めました。
- 利用者の行が共有の既定に勝つ。これは`setUserValueOf`の意味を、宣言された範囲で保つものです。
- 宣言されていない鍵は、空文字ではなく報告になります。画面の設定が静かに消えることがありません。

## 読み取れないこと

- 設定の**書き込み**（受理された変更としての更新・監査）。今は読み取りだけです。
- 設定値の型の絞り込み（`page_size`が整数であること等）。今は値の型を問いません。
- Workspace例への設定Data追加。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。利用者設定の宣言と読み取りは参照実装で確認済み、書き込みと例への追加は未実装です。
