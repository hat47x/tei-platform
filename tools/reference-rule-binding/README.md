# Data Derivation D4：導出規則の適用先解決 Reference（#1692）

本ディレクトリは **合成入力を読取り専用で検査する小さな参照** であり、TEIの正式なResolver、式言語、Plugin適合レジストリ、受入済みAPIではありません。実装主体は `rulebinding.go`、受入は `rulebinding_test.go`、起動点は `cmd/tei-rule-binding/` です。`正本`を書き換える経路はありません。

## 責務と分界

- **#1637（D1）**：規則と適用先を分離する最小契約を前提にします。本参照は受理済みRule identity／type／versionを、型付きSemantic IDの適用先へ結び付けます。
- **#1120／#1121／#988**：Semantic ID、受理→Lowering、汎用Operatorは委譲先です。第二のResolver・Canonical正本を作りません。
- **#1016／#951／#267**：Application Composition、Plugin契約、受理の判断は委譲します。Plugin提案は自動受理しません。
- 物理列名、同名Symbol、Source file path、登録順序による暗黙推定をしません。SELECTやデータ更新の実行器は新設しません。

## 判定する意味（採用前）

| 条件 | 期待 | 結果コード |
| --- | --- | --- |
| 同一Ruleを複数TargetへBinding | 適用先ごとに解決し、規則本文を複製しない | `RESOLVED` |
| 同一Targetに複数規則（明示の優先なし） | 登録順先勝ちにしない | `AMBIGUOUS` |
| 一意な明示Priority／一意なReplace | 宣言に従って解決 | `RESOLVED` |
| 案件Pluginが実装だけをGo→Wasmへ置換 | Rule ID・適用先・意味は不変 | `IMPLEMENTATION_REPLACED` |
| NULL方針／型を変える候補を実装差替えとして投入 | 契約変更として再受理要求 | `CONTRACT_CHANGE_REQUIRES_ACCEPTANCE` |
| 別名前空間の同名Symbol | 別IDとして暗黙推定しない | `SAME_NAME_DIFFERENT_ID` |
| 適用先の型不一致 | 拒否 | `TYPE_MISMATCH` |
| 未受理Rule | 拒否 | `RULE_NOT_ACCEPTED` |
| 未登録Plugin | 自動受理しない | `PLUGIN_NOT_REGISTERED` |
| Bindingなし | 未解決 | `UNBOUND` |
| 未登録Target | 拒否 | `TARGET_UNKNOWN` |

## 合成Fixture

規則 `sales.line-amount`（quantity × unit_price）を、適用先 `sales.order-line.amount` と `query.order-list.line-total` にそれぞれBindingします。

| 操作 | 本参照の結果 |
| --- | --- |
| 2つのTargetを解決 | どちらも同一Rule ID `sales.line-amount` を `RESOLVED`（規則本文は複製しない） |
| 同一Targetへ2規則を競合 | `AMBIGUOUS` |
| 一方に明示Priority | `RESOLVED` |
| 案件Pluginの実装差替え（Go→Wasm） | `IMPLEMENTATION_REPLACED`、`implementations=[go,wasm]`、意味不変 |
| NULL方針を変える候補 | `CONTRACT_CHANGE_REQUIRES_ACCEPTANCE` |
| 別名前空間 `other.line-amount` | `SAME_NAME_DIFFERENT_ID` |

## 実行方法

```sh
go run ./tools/reference-rule-binding/cmd/tei-rule-binding -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "rules": [
    { "id": "sales.line-amount", "version": "1", "accepted": true, "type": "amount" }
  ],
  "targets": [
    { "id": "sales.order-line.amount", "type": "amount" }
  ],
  "bindings": [
    { "ruleId": "sales.line-amount", "targetId": "sales.order-line.amount" }
  ],
  "resolveTarget": "sales.order-line.amount"
}
```

## 限界

- 合成入力に対する静的な判定です。実Application Query・実Data属性への接続や、実Pluginの実行同等性の証拠ではありません。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- M2（#1120／#1121／#988 の既存実装への接続と、二つ目のTargetでの再利用検証）は本PRの範囲外であり、未実証として残します。
