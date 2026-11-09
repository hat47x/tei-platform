# Data Derivation D13：一括導出の行別失敗と結果完全性の Reference（#1724）

本ディレクトリは **合成入力を読取り専用で検査する小さな参照** であり、TEIの正式なQuery planner、SQL実行器、認可判定、Scenario正本ではありません。実装主体は `completeness.go`、受入は `completeness_test.go`、起動点は `cmd/tei-batch-completeness/` です。`正本`を書き換える経路はありません。

## 既存責務との接続

- **D1 #1637**：`internal/semantic/derivationref` の入力状態と `Multiply` を流用し、行の導出値と状態を得ます。**正常なNULLは失敗ではありません**。入力未取得などは評価不能として区別し、0やNULLへ変換しません。
- **D2 #1640**：Query計画（Filter／Sort／Aggregate／Count／Page）は所有しません。導出列が集合演算に使われるかどうかだけを受け取り、計画の可否を判定します。
- **D9 #1706 / D5 #1693 / D7 #1696**：集合の完全性、診断の認可、計算根拠の表現はこれらのIssueが所有します。本参照は認可判断や診断の公開可否を決めません。
- **#1127 / #1180**：標準一覧・帳票とScenario正本は再実装しません。

## 判定する意味（採用前）

| 導出列の使われ方 | 評価不能行がある場合 | 本参照の結果コード |
| --- | --- | --- |
| 単純な元行 Count（`count-rows`） | 導出失敗は影響しない | `OK`（`complete=true`） |
| 任意表示列のみ（`display`、部分Projectionを明示許可） | 成功行の値＋行別診断を返す | `PARTIAL_PROJECTION`（`partial=true`） |
| 任意表示列のみ（`display`、許可なし） | 全体を不成立とする | `INCOMPLETE_DISPLAY` |
| Filter／Sort／Aggregate／Count 導出（`filter`/`sort`/`aggregate`/`count-derived`） | 完全な結果・件数を返せない | `INCOMPLETE_SET_OPERATION` |
| 評価不能行が無い | 完全な結果を主張できる | `OK`（`complete=true`） |

## 合成Fixture

| 行 | 数量 | 単価 | 導出 amount | 行の状態 |
| --- | ---: | --- | --- | --- |
| A | 2 | 300 | 600 | `present` |
| B | 3 | 未取得 | （値なし） | `input-unavailable` |
| C | 0 | 300 | 0 | `present`（0はNULLではない） |
| D | NULL | 200 | （値なし） | `null` |

導出できない行の状態は、原因ごとに分けます。

- `present` / `null`：正当な値。NULLは失敗ではありません
- `input-unavailable`：入力未取得（上のB）
- `type-error`：型・演算子の失敗（例：数量`abc`）。入力未取得と区別します
- `execution-unavailable`：規則を実行できない状態。入力未取得と区別します

- 表示用の任意列で部分Projectionが明示的に許可されれば、A=600、C=0、D=NULL、B=評価不能と返し、`partial=true` と該当行を識別します。
- `filter`（例 `amount > 0`）は、Bを黙って除外して「Aのみ・Count 1」と返しません。全体を不成立とします。
- `sort`（例 `amount DESC limit 1`）は、Bが最大か不明なためAを「最大1件」と返しません（`TestSortDescLimitOneDoesNotClaimMax`）。
- `aggregate`（例 `SUM(amount)`）は、Bを0やNULLへ変換して合計を返しません（`TestSumDoesNotSubstituteZeroOrNullForMissingRow`）。
- 同じfixtureで`sort`と`SUM`を再実行し、同じ結果になることを確かめます（`TestSortAndSumNegativesAreDeterministic`）。
- `count-rows` は元行数を返し、導出失敗に影響されません。`count-derived` は導出に基づく件数なので、評価不能行があれば不成立です。

## 実行方法

```sh
go run ./tools/reference-batch-completeness/cmd/tei-batch-completeness -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "rows": [
    { "id": "A", "quantity": { "state": "present", "decimal": "2" }, "price": { "state": "present", "decimal": "300" } },
    { "id": "B", "quantity": { "state": "present", "decimal": "3" }, "price": { "state": "missing" } },
    { "id": "C", "quantity": { "state": "present", "decimal": "0" }, "price": { "state": "present", "decimal": "300" } },
    { "id": "D", "quantity": { "state": "null" }, "price": { "state": "present", "decimal": "200" } }
  ],
  "query": { "op": "display", "partialProjectionAllowed": true }
}
```

`op` は `display`、`filter`、`sort`、`aggregate`、`count-rows`、`count-derived` です。

## 限界

- 合成入力に対する静的な可否判定です。実Provider、実DB、実SQL、実Rendererでの挙動の証拠ではありません。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- Filter／Sort／Aggregateの**実際の結果値**は計算しません（M1は計画の可否判定のみ）。M2で #1640 の最小Query実行へ接続する場合は別の作業です。
