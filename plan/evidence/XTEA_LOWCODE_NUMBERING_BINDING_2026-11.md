# 採番Dataの宣言と行の結合の確認（2026-11）

## 対象

- 実装: `internal/numbering/binding.go`
- 試験: `internal/numbering/binding_test.go`
- 対象版: `main` の `7403549a7` から分岐した`feat/numbering-data-binding-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/numbering -count=1 -v
```

## 観測

- `TestResolveReadsTheDeclaredRoles` 成功。`numbering-counter`・`numbering-digits`・`numbering-prefix`・`numbering-check`・`numbering-overflow`の役割から属性を解決する。
- `TestResolveRefusesAMissingRequiredRole` 成功。カウンタ・桁数が欠けたDataと未宣言のDataは拒否される。
- `TestBindBuildsTheContractAndState` 成功。1行から`Contract`（桁5・接頭辞`ORD-`・モジュラス10検査数字）と`State`（41）を作り、次の番号が発行でき、状態が42へ進む。
- `TestBindDefaultsAreNotInventedFromEmptyColumns` 成功。任意列が空のとき、接頭辞なし・検査数字なし・桁あふれは停止（既定）になる。空欄から`wrap`を捏造しない。
- `TestBindReadsTheOverflowRule` 成功。`wrap`を宣言した行だけ1へ戻る。
- `TestBindReportsUnparsableValues` 成功。桁数の空・非数値、桁数0、検査数字`maybe`、負のカウンタは拒否される。
- パッケージ全体は`ok`（14件）。

## 読み取れること

- 採番表の列（`NRCURRENT`・`NRNUMDIGIT`・`TXPREFIX`・`FGWITHCD`）は、**宣言された役割**から決まる。X-TEAは表名をシステム変数から取り、列名を直接読んでいた（`Session.java` L1980–2009）。
- 任意の役割が空のときは、契約の既定（接頭辞なし・検査数字なし・停止）になる。空欄から別の規則を推測しない。
- 非数値・負のカウンタは発行前に拒否される。X-TEAは例外を握りつぶして空文字を返した（同 L2004–2007）ため、失敗が「番号なし」と区別できなかった。

## 読み取れないこと

- Workspace例（`.tei`）への採番Dataの追加。
- 取引内での発行と取消（`State`を戻す経路）、同時実行の競合（`UPDCOUNTER`の楽観的更新）。
- 複数の採番規則が1つの表にある場合の、行の識別（`IDNUMBER`に相当する役割は未定義）。
- 実DB・旧Runtimeでの採番の実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。採番のData宣言との結合は参照実装で確認済み、取引・同時実行・例への追加は未実装です。
