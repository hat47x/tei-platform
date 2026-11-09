# 為替Dataの宣言と行の結合の確認（2026-11）

## 対象

- 実装: `internal/currencyrate/binding.go`
- 試験: `internal/currencyrate/binding_test.go`
- 対象版: `main` の `43765839f` から分岐した`feat/currency-data-binding-20261108`
- 環境: Windows、Go `1.27.0`

## 実行

```powershell
go test ./internal/currencyrate -count=1 -v
```

## 観測

- `TestResolveReadsTheDeclaredRoles` 成功。`currency-rate-currency`・`-fiscal-year`・`-month-sequence`・`-middle`・`-buying`・`-selling`の役割から属性を解決する。
- `TestResolveRefusesAMissingRequiredRole` 成功。通貨・年度・月順序・中値のいずれかが欠けたData、および未宣言のDataは拒否される。
- `TestBindBuildsRatesFromRows` 成功。保存行からRateを作り、買値・売値は宣言されている行にだけ値が入る（無い行に0を捏造しない）。結合した行は既存の`RateFor`で引ける。
- `TestBindReportsANonPositiveRate` 成功。`0`と負の率は結合時に拒否される。
- `TestBindReportsAnImpossibleMonthSequence` 成功。月順序0・13は拒否される。
- `TestBindReportsAnUnparsableRow` 成功。通貨の欠落・年度の非数値・率の非数値は、行番号と属性名つきで報告される。
- パッケージ全体は`ok`（14件）。

## 読み取れること

- 為替表の属性は、列名ではなく**宣言された役割**から決まる。X-TEAは`KBCURRENCY`・`DTNEND`・`DTMSEQ`・`VLRATEM`等を列名で直接読み、表名もシステム変数から取っていた（`Session.java` L2253–2313）。
- **率0を制御値にしない**。X-TEAは月次率が0のとき年次表へ切り替え（同 L2296–2298）、「行が無い」と「率が0」を区別できなかった。結合時に非正の率を拒否するので、0が制御値として紛れ込まない。
- 任意の役割（買値・売値）は、宣言されている行にだけ値を入れる。無い行に0を入れないので、`KindBuying`の要求は結合後の照会で失敗する。

## 読み取れないこと

- Workspace例（`.tei`）への為替Dataの追加。新しい役割名を含む宣言は未追加。
- 年次表と月次表の優先順位（フォールバック）の宣言。参照実装は呼出し側が期間を渡し、`MissingPeriods`で欠落を示す形。
- 実DB・旧Runtimeでの為替表の実測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。通貨のData宣言との結合は参照実装で確認済み、例への追加とフォールバックの宣言は未実施です。
