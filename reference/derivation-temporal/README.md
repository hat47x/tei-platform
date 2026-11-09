# D14 過去時点再計算・規則版・確定保存値の時間意味 Reference（#1727）

**状態: 合成Fixtureの読取り専用Reference。bitemporal DB、履歴保存、Rule Registry、Commitを実装しない。**

このパッケージは、業務値の時間軸と結果の意味を分ける。当時の有効価格と当時の規則で再計算した値、現在の規則を過去へ適用した試算、現在の計算値、保存済みの確定事実を、単一の `asOf` へ集約しない。

## 区別する時点と意味

- `valid-time`（業務有効時点）: 取引日時に適用される単価・料率。元データが無ければ過去有効値を推定しない。
- `recorded-at`（記録・観測時点）: Providerが入力を把握・記録した時点。valid-time と同一視しない。
- `evaluated-at`（評価実行時点）: 再計算を今行った時刻。
- `Rule revision`: 当時受理済みの規則版と現在の規則版を区別する。現在の規則を過去データへ自動適用した値は「当時計算」ではない。
- `confirmed fact`: 当時のActionが保存・承認した金額。再計算値が一致しても元の保存事実の証拠とはせず、一致しなくても勝手に上書きしない。

## 結果区分（Reference上の候補名）

| Kind | 意味 | Replayable |
| --- | --- | --- |
| `current-computation` | 現在の入力＋現行規則 | false |
| `historical-replay` | 当時の入力版＋当時の規則版。再現条件を満たす場合のみ | true |
| `what-if` | 過去データに別の規則を適用するなど意図的な組合せ | false |
| `confirmed-fact-read` | 保存済み事実そのものの参照（導出実行とは別） | false |

## 非捏造の境界

- `historical-replay` は当時の価格と当時の規則版の両方を要求する。片方が無ければ `ErrHistoryUnknown` / `ErrRevisionUnknown` とし、現在の価格や現行規則で代用しない（`PriceBasis` を明示し、暗黙fallbackをしない）。
- `what-if` は常に what-if として表示し、確定事実を自分の値として運ばない。
- `confirmed-fact-read` は入力版・規則版が取得不能でも「再評価の成功」ではない。Kind と Replayable で区別する。
- 過去レコード参照の権限拒否は #1693 が所有する。このReferenceは認可を判断しない。

## 合成Fixture

商品Pの価格履歴: 2026-04-01から300円、2026-06-01から350円。受注Oは2026-04-15に数量2で確定し、保存済み金額600円。規則v1は `qty×price`、v2は `qty×price×0.9`。

| 要求 | 期待 | 試験 |
| --- | --- | --- |
| 当時価格300・v1で historical-replay | 600 / Replayable | `TestD14HistoricalReplayUsesThenPriceAndRule` |
| 旧価格300・v2で what-if | 540 / 確定600を変更しない | `TestD14WhatIfDoesNotChangeConfirmedFact` |
| 現在価格350・v1で current-computation | 700 / 過去の確定金額と無関係 | `TestD14CurrentComputationUsesCurrentPrice` |
| Providerに履歴なし | `ErrHistoryUnknown`（300と推定しない） | `TestD14MissingHistoryIsNotEstimated` |
| v2のみ存在しv1再現を要求 | `ErrRevisionUnknown`（当時のv1再現と称さない） | `TestD14MissingRevisionIsNotSubstituted` |
| 保存済み600の参照 | confirmed-fact-read / Replayableでない | `TestD14ConfirmedFactReadIsNotAReplay` |

## 焦点試験

```sh
gofmt -l reference/derivation-temporal/*.go
go test -count=1 ./reference/derivation-temporal
go vet ./reference/derivation-temporal
```

## 未実装・依存

- 現在の計算値・キャッシュ・確定保存値は #1641、同一実行内の読取りSnapshot整合は #1708、固定時計・TZは #1720、説明Projectionは #1696、受理済み規則とPlugin Lockは #1121 が所有する。
- bitemporalの汎用DB、過去価格の自動復元、Event Sourcing、全履歴保存義務、規則版の永久保持、過去権限の復活、確定値の自動訂正は行わない。
- 合成Fixtureのみ。実Providerの履歴API、実DB保存Evidence、監査・帳票接続、X-TEA #940の実動作との同等性は未検証。M2は履歴APIを持つ案件でのみ版固定した読取り再計算を行う。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1727 と対象PRへ記録する。
