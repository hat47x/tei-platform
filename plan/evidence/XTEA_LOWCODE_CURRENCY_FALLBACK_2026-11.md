# 為替の年次フォールバックの宣言の確認（2026-11）

## 対象

- 実装: `internal/currencyrate/fallback.go`、`Set.Fallback`
- 試験: `internal/currencyrate/fallback_test.go`
- 対象版: `main` の `7452b2b9b` から分岐した`feat/currency-annual-fallback-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/currencyrate -run 'TestFallback' -count=1 -v
```

## 観測

- `TestFallbackDefaultIsNone` 成功。フォールバックを宣言しない場合、月次行の欠落は報告される（別の期間を勝手に使わない）。
- `TestFallbackAnnualUsesTheAnnualRow` 成功。`annual`を宣言すると年次行（月順序0）を使い、月次行がある期間では月次を優先する。
- `TestFallbackAnnualReportsAMissingAnnualRow` 成功。年次行も無い場合は0を返さず報告する。
- `TestFallbackRejectsAnUnknownName` 成功。`monthly`のような未知の名前は拒否される。
- `TestFallbackKeepsTheSystemCurrencyAtOne` 成功。システム通貨はフォールバックの影響を受けず率1。
- パッケージ全体は`ok`。`internal/validation`・`cmd/tei`も`ok`。

## 読み取れること

- 年次への切替は**宣言**であり、既定では起きない。X-TEAは月次率が0のとき年次表へ切り替え（`Session.java` L2296–2298）、「行が無い」と「率が0」を区別できなかった。
- 年次行も月次と同じDataに置ける（月順序0で区別）。
- フォールバックが答えられない場合は、0ではなく報告になる。

## 読み取れないこと

- 為替Dataの宣言（`.tei`）から`Fallback`を導く語彙。今は`Set`を組む呼出し側が設定します。
- 月次と年次を別Dataに分ける場合の結合。
- 実DB・旧Runtimeの実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。フォールバックの宣言と参照実装の動作は確認済み、宣言語彙と例への追加は未実装です。
