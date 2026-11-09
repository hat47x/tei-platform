# D13 一括導出の行別失敗・部分結果・集合演算の完全性 Reference（#1724）

**状態: 合成Fixtureの読取り専用Reference。D1のError enum、D2のQuery Planner、D5の認可、D7の説明を置き換えない。**

このパッケージは、一括Query・一覧・帳票で行ごとに導出の成否が分かれるとき、**一行の導出状態**と**Query要求が完全に成立したか**を区別する最小契約を固定する。100行中1行の失敗を暗黙のNULLや0として返す、失敗行だけ除外する、不完全な結果を完全な結果として扱うことを防ぐ。

## 行の結果区分

`RowOutcomeOf` は D1 の結果とエラーから一行の結果を読む。D1のエラー集合を再実装しない。

- `value` / `null`: 正常値と、宣言された正当なNULL。正当なNULLは失敗ではない。
- `input-unavailable`: 入力未取得・参照先なし・多重度違反。
- `evaluation-failed`: 型・演算子・循環・副作用などの評価失敗。

未取得と評価失敗をNULLや0へ潰さない。認可拒否は #1693 が所有し、ここでは判断しない。

## Query利用と完全性

`Plan` は導出値の用途ごとに、バッチ結果を完全として返せるか、明示的に部分的な表示Projectionとして返せるか、全体を失敗させるかを決める。

- `display-only` かつ明示的に部分許可があるときだけ、全行を保ったまま評価不能行を `unevaluated` として識別した部分Projectionを返す。`partial=true` 相当の状態を別に示し、完全な計算済み表と偽装しない。
- `filter` / `sort` / `aggregate` / `group` / `count-derived` / `page-cursor` は導出値を読むため、一行でも評価不能なら**Query全体を不成立**として扱い、部分データを返さない。`aggregate` の空集合・NULL方針そのものは D9 #1706 が所有する。
- `count-source-rows` は導出値を読まないため、導出の評価不能に影響されない。単純な元行Countと、導出Filter後のCountを区別する。

失敗時の `Decision` は空であり、行データを返さない。これにより、失敗したバッチが部分結果を漏らさない。

## 合成Fixture

| 行 | 数量 | 単価 | 導出amount | 区分 |
| --- | ---: | ---: | --- | --- |
| A | 2 | 300 | 600 | value |
| B | 3 | 未取得 | - | input-unavailable |
| C | 0 | 300 | 0 | value |
| D | NULL | 200 | 宣言されたNULL | null |

| 操作 | 期待 | 試験 |
| --- | --- | --- |
| 表示用の任意 `amount` 列のみ（部分許可あり） | A=600, C=0, D=NULL, B=評価不能、`partial` を識別 | `TestD13PartialDisplayKeepsEveryRowAndMarksUnevaluated` |
| 同上（部分許可なし） | 全体失敗 | `TestD13DisplayWithoutAllowanceFailsWhole` |
| `filter amount > 0` | Bを黙って除外せず全体失敗 | `TestD13SetOperationsFailOnUnevaluated` |
| `sort amount DESC` / `page-cursor` | Bの最大値・順序が不明なため全体失敗 | 同上 |
| `SUM(amount)` | Bを0/NULLへ変換して600と返さず全体失敗 | 同上 |
| `count rows`（元行） | 導出失敗に影響されない | `TestD13SourceRowCountIsUnaffected` |
| 評価不能行 | 0やNULLではない | `TestD13UnevaluatedIsNotZeroOrNull` |

## 焦点試験

```sh
gofmt -l reference/derivation-batch/*.go
go test -count=1 ./reference/derivation-batch
go vet ./reference/derivation-batch
```

## 未実装・依存

- Query計画、Filter/Sort/Aggregate/Limitの実行順、再計画は #1640 が所有する。このReferenceは評価済みの行集合を受け取るだけで、クエリを解析・実行・最適化しない。
- 集合の形・完全性・多重度は D9 #1706、診断の観測範囲と根拠表現は #1696、診断情報の認可と行存在の秘匿は #1693 が所有する。
- 一括書込みのRollbackと副作用は #1129 / #1174 が所有する。ここでは読取りQueryの完全性だけを扱う。
- 中断・タイムアウト時の途中結果の再利用条件は #1641 / #1708 へ委譲し、このReferenceでは扱わない。
- 合成Fixtureのみ。実Provider、実DB、実SQL、CLI/標準一覧（#1127）、X-TEA #940の実動作との同等性は未検証。部分Projectionの正式なschema名はReference後に決める。

実行の有無、exact SHA、成功／失敗／未検証を Issue #1724 と対象PRへ記録する。
