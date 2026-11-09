# Data Derivation D7：導出値の説明根拠・開示境界 Reference（#1696）

本ディレクトリは **一つの導出値の説明Projectionを読取り専用で組み立てる小さな参照** であり、TEIの正式な分散トレーシング、第二のSource正本、認可判定、UI、受入済みAPIではありません。実装主体は `derivationevidence.go`、受入は `derivationevidence_test.go`、起動点は `cmd/tei-derivation-evidence/` です。`正本`を書き換える経路はありません。

## 責務と分界

- **#1165（Canonical trace／inspect）／#1178（provenance開示）／#1180（Scenario SSOT）**：一般のTrace・Scenarioは委譲先です。別の追跡グラフや権限系を作りません。
- **#1637（D1）**：導出の意味契約を前提にします。**#1641（D3）**はキャッシュ・鮮度、**#1693（D5）**は権限判断を所有し、本参照は「計算された一つの値が何に由来するか」の開示境界に絞ります。
- 全計算の常時詳細トレース、全入力値ログ、情報源の完全な再現保証はしません。`正本`を書き換えません。

## 区別する三つの由来

| 種別 | 例 | 本参照での扱い |
| --- | --- | --- |
| 定義の由来（Design Provenance） | 規則の記述元（Rule ID・受理版・適用先） | 機械可読に示すが、実行値の真正性の証明にはしない |
| 計算の入力根拠（Evaluation Evidence） | 実現identity・入力identity/version・評価文脈 | 追跡可能な範囲を示し、不明はUnknownに残す |
| 結果の業務上の確定（Business Fact） | 受注確定で保存された額 | #1641／#1129 が所有。本参照は上書きしない |

## 判定する意味（採用前）

| resultState | 診断 | 備考 |
| --- | --- | --- |
| `present` | `RESULT_PRESENT` | 入力versionが既知なら追跡可能 |
| `input-missing` | `INPUT_MISSING` | NULLや結果0へ潰さない |
| `null-declared` | `NULL_DECLARED` | 宣言されたNULL規則の結果 |
| `null-undeclared` | `NULL_UNDECLARED` | NULLを0として扱わない |
| `unauthorized` | `UNAUTHORIZED` | 値と入力を開示しない |
| `compute-error` | `COMPUTE_ERROR` | 原因不明をNULLへ潰さない |
| `multiplicity-violation` / `type-mismatch` / `unsupported` | 各診断 | 個別に区別する |

## 開示境界

- `admin`：結果が`present`のときだけ`inputValue`／`unitPrice`を開示する。`input-missing`・NULL・`compute-error`・`unauthorized`などの非present状態では、値が無いため伏せる。
- `general`：`inputValue`／`unitPrice`／`rowExistence` を伏せる（`redacted`）。
- `unauthorized`：`admin`でも`rowExistence`を伏せ、値と入力を開示しない。
- エラー文言は、入力値・単価を含めない。
- 入力versionが不明なときは `traceable=false` とし、`unknown=[inputVersion]` を残す（完全再現を主張しない）。
- 規則のSource位置が移動してもCanonical identityと意味が同一なら説明identityは不変。

## 合成Fixture（受注明細）

数量2・単価300 → 金額600。規則 `line-amount`、対象 `order-line.amount`、受理版 `R`、実現 `X`。

| 操作 | 本参照の結果 |
| --- | --- |
| 正常値の説明 | `RESULT_PRESENT`、`ruleId`／`targetId`／`realization` を開示 |
| 単価未取得 | `INPUT_MISSING`（NULLと区別） |
| 単価NULL（宣言あり／なし） | `NULL_DECLARED` と `NULL_UNDECLARED` を区別 |
| 計算不能 | `COMPUTE_ERROR`（NULL・未取得と区別） |
| 管理者と一般利用者 | 一般の `trace` は単価・入力値・行存在を伏せる |
| 非present状態の管理者 | 値が無いため `inputValue`／`unitPrice` を伏せる（`unauthorized`は`rowExistence`も） |
| 入力version不明 | `traceable=false`、`unknown=[inputVersion]` |
| Source移動（意味不変） | 説明identityは不変 |

## 実行方法

```sh
go run ./tools/reference-derivation-evidence/cmd/tei-derivation-evidence -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "ruleId": "line-amount",
  "acceptedRevision": "R",
  "targetId": "order-line.amount",
  "realization": "X",
  "inputVersionKnown": true,
  "resultState": "present",
  "inputValue": "2",
  "unitPrice": "300",
  "requester": "general"
}
```

## 限界

- 合成入力に対する静的な判定です。実Providerで取得できない根拠の完全再現は保証しません（Unknownとして残します）。
- 合成試験の成功を本番値の保証にはしません。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- M2（CLI／標準UIの機械可読Diagnosticとの対応、実Providerでの未知扱い）は本PRの範囲外であり、未実証として残します。
