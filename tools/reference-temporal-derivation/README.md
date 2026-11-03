# reference-temporal-derivation（#1727）

過去時点の再計算・規則版・確定保存値の**時間意味**を区別する読取り専用の参照です。業務有効時点（valid-time）、記録・観測時点、規則の意味版、評価実行時点、業務確定事実を混同せず、再計算の結果を`historical-replay` / `what-if` / `current-computation` / `confirmed-fact-read`としてラベルします。

再計算結果で保存済みの確定金額を書き換えません。履歴DB・別のRule Registry・分散Transactionを作りません。

## 実行

```bash
go run ./tools/reference-temporal-derivation/cmd/tei-temporal-derivation \
  -input tools/reference-temporal-derivation/testdata/historical-replay.json
```

結果はJSONで標準出力へ出します。

## 入力

```json
{
  "kind": "historical-replay",
  "quantity": "2",
  "asOf": "2026-04-15",
  "ruleVersion": "1",
  "priceHistory": [
    {"validFrom": "2026-04-01", "price": "300"},
    {"validFrom": "2026-06-01", "price": "350"}
  ],
  "historicalPricesAvailable": true,
  "rules": [
    {"version": "1", "multiplier": "1"},
    {"version": "2", "multiplier": "0.9"}
  ],
  "confirmedSaved": {"value": "600", "validTime": "2026-04-15", "ruleVersion": "1"},
  "authorization": "allowed",
  "evaluationAt": "2026-11-01T10:00:00Z"
}
```

## 結果の種別

| `kind` | 使う入力 | 結果 |
| --- | --- | --- |
| `historical-replay` | `asOf`時点の価格＋当時の規則版 | `HISTORICAL_REPLAY`、`isHistoricalReplay=true`、`reproductible=true` |
| `what-if` | 過去データに指定した（現在の）規則版 | `WHAT_IF`、当時の確定結果とも再現とも呼ばない |
| `current-computation` | 現在の価格＋指定した規則版 | `CURRENT_COMPUTATION`、過去の確定金額と無関係 |
| `confirmed-fact-read` | 保存済み事実 | `CONFIRMED_FACT_READ`、再計算しない |

## 判定する境界

| 条件 | 結果 |
| --- | --- |
| 認可されていない | `AUTHORIZATION_DENIED`（値も履歴の存在も開示しない） |
| 過去価格をProviderが提供できない／`asOf`が最初の価格より前 | `UNKNOWN_HISTORICAL_PRICE`（現行価格から推定しない） |
| 当時の規則版が取得できない | `UNKNOWN_RULE_VERSION`（別版の値で再現を装わない） |
| `historical-replay`に`asOf`が無い | `INVALID_REQUEST` |
| 未知の`kind` | `INVALID_REQUEST` |

`evaluationAt`（評価実行時点）は参考情報であり、`asOf`（業務有効時点）の代用にしません。`priceValidFrom`で、どの価格時点を使ったかを示します。

## 不変性

`updatesSavedFact`は常に`false`です。`historical-replay`・`what-if`・`current-computation`のいずれも、`confirmedSaved`の値を書き換えません。`confirmed-fact-read`は保存済み事実をそのまま返し、再評価しません。

## 合成Fixture

商品Pの価格履歴：2026-04-01から300円、2026-06-01から350円。受注Oは2026-04-15に数量2で確定し、保存済み金額600円。規則v1は`qty×price`、v2は`qty×price×0.9`。

| 要求 | 結果 |
| --- | --- |
| `historical-replay`（2026-04-15、v1） | 600円（`priceValidFrom=2026-04-01`） |
| `what-if`（2026-04-15、v2） | 540円。保存済み600円は不変 |
| `current-computation`（v1） | 700円（現在価格350） |
| `confirmed-fact-read` | 600円。再計算しない |
| 価格履歴なし（`historicalPricesAvailable=false`） | `UNKNOWN_HISTORICAL_PRICE` |
| 規則v1が取得できない | `UNKNOWN_RULE_VERSION` |

## 範囲と非主張

- 合成入力に対する静的な意味の判定です。実Providerの履歴API、実DBのbitemporal表現、実Actionの保存を観測した証拠ではありません
- 過去価格の自動復元、Event Sourcing、全履歴保存、規則版の永久保持、過去権限の復活、確定値の自動訂正はしません
- #1641（鮮度・保存）、#1708（Snapshot）、#1696（説明）の既存契約を置き換えません
- `-race`はcgoを伴わない可搬Goでは実行していません
