# Data Derivation D6：規則と実現の意味適合 Reference（#1694）

本ディレクトリは **規則の宣言した業務意味と、指定された実現（実装）が適合するかを読取り専用で検査する小さな参照** であり、TEIの正式なOperator登録所、Plugin Lock権威、Build API、汎用式言語、受入済みAPIではありません。実装主体は `realizationcompliance.go`、受入は `realizationcompliance_test.go`、起動点は `cmd/tei-realization-compliance/` です。`正本`を書き換える経路はありません。

## 責務と分界

- **#988**：一般Operatorの意味契約の正本。本参照は既存の登録所を参照し、別のOperator登録所を作りません。
- **#1121**：Target Lowering／固定は委譲先です。Plugin Lock・Build APIを新設しません。
- **#951／#1016／#943**：Plugin Capability契約と選択は委譲します。必須の汎用Expression Engine、重いプラグイン認証制度を導入しません。
- **#1637（D1）**：規則の意味契約を前提にします。**#1640（D2）**はQuery計画の正しさの正本です。
- 生成AI作成のテスト通過を独立した権限授与や汎用適合証明として扱いません。`正本`を書き換えません。

## 区別する対象

- **規則identity**（受理済みの意味：出力型・NULL方針・丸め・pure・fixture Oracle）
- **実現identity**（Plugin identity／版、Capability、実装方式 go／sql／wasm）— 規則本文を複製せず入替え可能

## 判定する意味（採用前）

| 条件 | 期待 | 結果コード |
| --- | --- | --- |
| 実現が契約とOracleに適合 | 受理 | `COMPLIANT` |
| 実現が一つのみ | 第二実現の同等性は未証明 | `COMPLIANT`＋`equivalenceUnproven` |
| 二実現のfixture結果が一致 | 同等性を確認 | `COMPLIANT`（`implementations` 併記） |
| 二実現の入力集合・結果が食い違う | 適合未証明として差替えない | `EQUIVALENCE_UNPROVEN` |
| fixture結果がOracleと不一致 | 拒否 | `FIXTURE_MISMATCH` |
| 出力型不一致 | 拒否 | `TYPE_MISMATCH` |
| NULL方針不一致 | 契約変更として再受理 | `NULL_POLICY_MISMATCH` |
| 丸めモード不一致 | 同等実装の差替えとして認めない | `MEANING_MISMATCH` |
| pureな規則に副作用を要求 | 拒否 | `IMPURE_REALIZATION` |
| 未登録Plugin | 黙って落とさない | `REALIZATION_NOT_REGISTERED` |
| 固定版の欠落／版不一致 | 拒否 | `REALIZATION_LOCK_MISSING`／`REALIZATION_VERSION_MISMATCH` |
| 現在の権限・能力で実行できない | 別実装へ黙って落とさない | `REALIZATION_NOT_PERMITTED` |

## 合成Fixture

規則 `line-amount(quantity decimal, unit_price money[JPY])`、同一通貨・無丸め。

| 入力 | 期待 |
| --- | --- |
| `2×300` | `present` 600 |
| `0×300` | `present` 0 |
| `0.2×0.5` | `present` 0.1 |
| `NULL×300` | `null`（宣言方針に従う） |
| `未取得×300` | `missing`（NULLと区別） |
| `half-up` を宣言した実現 | `MEANING_MISMATCH`（同等差替えを認めない） |
| Go と SQL が同じ集合と結果 | `COMPLIANT`、`implementations=[go,sql]`、`equivalenceUnproven=false` |
| coverage が異なる二実現 | `EQUIVALENCE_UNPROVEN` |
| 規則identityを固定したまま実装identity（plugin・版・言語）を入替え | 同じ規則のfixtureで検査し、`RuleID`は変わらない |
| 現在の権限・能力で実行できない実現 | `REALIZATION_NOT_PERMITTED`（別実装へ落とさない） |

## 実行方法

```sh
go run ./tools/reference-realization-compliance/cmd/tei-realization-compliance -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "ruleId": "line-amount",
  "ruleOutputType": "amount",
  "ruleNullPolicy": "propagate",
  "ruleRounding": "half-even",
  "rulePure": true,
  "realizations": [
    {
      "plugin": "case-line-amount-go",
      "version": "3",
      "language": "go",
      "outputType": "amount",
      "nullPolicy": "propagate",
      "rounding": "half-even",
      "pure": true,
      "registered": true,
      "locked": true,
      "lockedVersion": "3",
      "cases": [
        { "quantityState": "present", "quantity": "2", "unitPrice": "300", "resultState": "present", "amount": "600" }
      ]
    }
  ]
}
```

## 限界

- **等価性の検出範囲は、二実現が報告したfixture結果とその集合の比較に限ります。** 片方だけが照らす入力や、Oracleに書かれないNULL・精度境界は検出できません。
- 合成入力に対する静的な判定です。実SQL方言・実Wasm・実Pluginでの実行同等性の証拠ではありません。第二実現を接続する実測は未実施です。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- M2（実際の案件需要で第二実現の差分比較とTarget接続を追加する）は本PRの範囲外であり、未実証として残します。
