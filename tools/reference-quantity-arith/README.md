# Data Derivation D8：数値・通貨・単位・丸めの意味注釈 Reference（#1705）

本ディレクトリは **合成入力を読取り専用で検査する小さな参照** であり、TEIの正式な型体系、通貨レジストリ、単位換算、式実行器、SQL生成ではありません。実装主体は `quantity.go`、受入は `quantity_test.go`、起動点は `cmd/tei-quantity-arith/` です。`正本`を書き換える経路はありません。

## 既存責務との接続

- **D1 #1637**：`internal/semantic/derivationref` の入力状態（`present`/`null`/`missing`/`absent-relation`/`multiple-relations`）と `Multiply` を流用します。`missing` などは `INPUT_UNAVAILABLE`、`null` は `NULL_PROPAGATED` とし、**欠損を数値ゼロへ変換しません**。0個と欠損は別の結果になります。
- **#988 / #1126**：型レジストリは新設しません。`unit`・`per`・`currency` は呼び出し側が与える**合成のラベル**であり、標準型名やDSL構文として採用済みとはみなしません。
- **D6 #1694 / D3 #1641 / D7 #1696**：通貨換算、消費税、日付依存単価などの外部参照は所有しません。換算レートの出所・時点・確定保存の境界はこれらのIssueへ返します。
- **#1121 / D2 #1640**：SQL式の生成・書換えはしません。意味をTargetへ委譲できる条件だけを、`CheckDelegation`で宣言された桁・丸め・正確な十進との比較として検査します。CASTの生成やnative SQLの書換えは行いません。

## 検査する意味の候補表（採用前）

| 入力・規則 | 期待 | 本参照の結果コード |
| --- | --- | --- |
| 2個 × 300 JPY/個 | 600 JPY | `OK`（`isAmount=true`, `currency=JPY`） |
| 0個 × 300 JPY/個 | 0 JPY（NULLではない） | `OK`（`isAmount=true`） |
| 1.25個 × 0.4 JPY/個 | 0.5 JPY（途中計算は正確） | `OK` |
| 1kg × 300 JPY/個 | 単位不一致として拒否 | `DIMENSION_MISMATCH` |
| 100 JPY + 2 USD | 明示的FX規則なしでは拒否 | `CURRENCY_MISMATCH` |
| 1.005を小数第2位へ丸め | HALF_UP→1.01、HALF_EVEN→1.00 | `OK`（丸め方を同一視しない） |
| DECIMAL(5,2)に1999.98を格納 | 精度超過として拒否。暗黙truncate不可 | `PRECISION_OVERFLOW` |
| 単価NULL／未取得 | D1の誤り状態を維持 | `NULL_PROPAGATED` / `INPUT_UNAVAILABLE` |
| -1.005を小数第2位へ丸め | HALF_UP→-1.01（0から遠い方）、HALF_EVEN→-1.00 | `OK`（符号でも方針差を保つ） |
| 宣言した桁・丸めをTargetが満たす | 委譲してよい | `DELEGATION_READY` |
| 桁・scale・丸め・正確な十進のどれかが満たせない | 黙ってCASTせず、未証明として返す | `DELEGATION_UNPROVEN`（`reasons` に不足条件） |

`JPY/個`、`個`、`kg` はfixture上の意味注釈であり、標準型名・DSL構文として採用済みではありません。

## 次元の扱い

各型をトークンの指数ベクトルとして扱います。`個` は `{個:+1}`、`JPY/個` は `{JPY:+1, 個:-1}` です。乗算は指数を加算し、同じトークンの `+1` と `-1` を相殺します。そのため `個 × JPY/個 = JPY` が成立します。

- 乗算の結果に複数の通貨が現れる場合は `CURRENCY_MISMATCH`。
- 乗算の結果に通貨と、通貨以外の残存トークンが同時に現れる場合は `DIMENSION_MISMATCH`（`kg × JPY/個`）。
- 加算・比較は両辺の次元が一致することを要求します。

## 実行方法

```sh
go run ./tools/reference-quantity-arith/cmd/tei-quantity-arith -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "currencies": ["JPY", "USD"],
  "types": {
    "個": { "unit": "個" },
    "JPY/個": { "currency": "JPY", "per": "個" }
  },
  "operation": {
    "op": "multiply",
    "left": { "type": "個", "state": "present", "decimal": "2" },
    "right": { "type": "JPY/個", "state": "present", "decimal": "300" }
  }
}
```

`op` は `multiply`、`add`、`compare`、`round`、`store` です。`round` は `scale` と `rounding`（`HALF_UP` / `HALF_EVEN`）、`store` は型の `precision` と `scale` を使います。`decimal` は十進文字列で与え、内部では `math/big` で正確に扱います。

### Targetへの委譲（`-mode delegation`）

```sh
go run ./tools/reference-quantity-arith/cmd/tei-quantity-arith -mode delegation \
  -input tools/reference-quantity-arith/testdata/delegation-ready.json
```

Targetの能力（`maxPrecision`・`maxScale`・`roundingModes`・`exactDecimal`）と、結果型の宣言（`precision`・`scale`）を比べます。次のいずれかが満たせない場合は`DELEGATION_UNPROVEN`とし、`reasons`に全ての不足条件を並べます。

- `precision-not-declared` / `scale-not-declared`（宣言なしを暗黙の既定にしない）
- `precision-exceeds-target` / `scale-exceeds-target`
- `rounding-mode-unsupported` / `rounding-mode-unknown`
- `target-not-exact-decimal`（floatへの暗黙変換を許さない）
- `result-type-unknown` / `target-not-declared`

委譲の判定は**合成の能力宣言**に対するもので、実SQL方言の検証ではありません。CASTを生成せず、SQLも書き換えません。

## 限界

- 合成入力に対する静的な検査です。実Provider、実DB、SQL方言、実Wasmでの挙動の証拠ではありません。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- 型・通貨・単位の追加宣言は必要な案件に限る、というD8の境界は満たしていません。本参照は候補であって採否の判断ではありません。
