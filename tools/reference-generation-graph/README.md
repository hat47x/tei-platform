# DSLの生成時・実行時二用途と変換工程 Reference（#1063）

本ディレクトリは **phase（authoring／validation／generation／runtime）付きの型付き変換グラフを読み取り専用で検査する小さな参照** であり、TEIの正式な生成機構、第二のPlugin Resolver、第二のBuild Lock、業務意味の受入台帳ではありません。実装主体は `generationgraph.go`、受入は `generationgraph_test.go`、起動点は `cmd/tei-generation-graph/` です。Operatorを実行せず、生成物を出力せず、生成候補を受理済みの業務意味へ昇格させません（#988/#979/#1016/#998の責務を複製しない）。

## 検査する意味（採用前）

| 対象 | 扱い |
| --- | --- |
| 純粋なOperator（Effectなし） | 生成時と実行時で同じ契約を共有できる（`compare`で確認） |
| Effectを持つOperator（`io`／`time`／`random`／`network`／`persistence`） | 宣言した段階でのみ動く。実行時へ無条件に漏れない |
| Operatorの型契約（`inputType`／`outputType`） | 段違いの接続を拒否（空の型は本参照では検査しない） |
| 生成stepの依存 | 循環を許さない |
| 未確定要件 | 解決せず、下流へ伝播して保持する |
| 規則版（`currentRuleVersion`／`incomingRuleVersion`） | 版が変わる場合は黙って旧出力を維持せず `RECOMPUTE_REQUIRED` |
| 反実行性 | 構造とOperator契約のfingerprint（`sha256:`）で固定する |

| 条件 | 結果コード |
| --- | --- |
| 構造検査に合格 | `CHECK_PASSED`（traceと未確定一覧を併記） |
| 同じOperator契約が両段階で一致 | `MEANING_SHARED` |
| 契約の内容が段階間で異なる | `MEANING_CHANGED` |
| Operator版が異なる | `OPERATOR_VERSION_CHANGED` |
| Effectが実行時グラフに現れる | `EFFECT_IN_RUNTIME` |
| Operatorがその段階を許可しない | `PHASE_NOT_ALLOWED` |
| 未宣言のOperator参照 | `OPERATOR_UNRESOLVED` |
| 同名Operatorの重複宣言 | `DUPLICATE_OPERATOR` |
| 型契約の不一致 | `OPERATOR_CONTRACT_MISMATCH` |
| 生成step依存の循環 | `TRANSFORM_CYCLE` |
| 規則版の変更 | `RECOMPUTE_REQUIRED` |

## 合成Fixture（受入条件との対応）

| 操作 | 本参照の結果 |
| --- | --- |
| 純粋Operator `list-order-lines` を同一fixtureで generation／runtime の両方へ接続 | 両方 `CHECK_PASSED`、`compare` は `MEANING_SHARED` |
| `merge-persisted-settings`（persistence effect）を runtime へ配線 | `PHASE_NOT_ALLOWED` |
| Effectを持つOperatorを宣言ごと runtime に置く | `EFFECT_IN_RUNTIME` |
| 受注要件 `requirement:R1` が未確定 | `req-order` と下流 `spec-lines` が unresolved のまま保持 |
| 規則版 `rules-v1` → `rules-v2` | `RECOMPUTE_REQUIRED`。同一版の再呼出しは同一fingerprint |
| Operator版 `v1` → `v2` の比較 | `OPERATOR_VERSION_CHANGED` |

`Trace` は各stepの `id <- operator @ origin` の一覧で、要件・設計・設定からの由来を辿れる形を保持します。

## 実行方法

```sh
go run ./tools/reference-generation-graph/cmd/tei-generation-graph -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例（runtimeグラフの検査）：

```json
{
  "currentRuleVersion": "rules-v1",
  "incomingRuleVersion": "rules-v1",
  "graph": {
    "phase": "runtime",
    "operators": [
      { "name": "list-order-lines", "version": "v1", "phases": ["generation", "runtime"], "inputType": "order-key", "outputType": "order-line-set" }
    ],
    "steps": [
      { "id": "req-order", "operator": "list-order-lines", "origin": "requirement:R1" }
    ]
  }
}
```

`"mode": "compare"` と `"compareTo"` を加えると、二つの段階で共有するOperator契約の一致を検査します。

## 責務と分界

- DSL自体のPlugin化とOperator意味契約は #988、決定論的変換ビルダーとBuild Plan／Lock／Artifactは #979、案件の構成選択は #1016、業務仕様テンプレートとAccepted意味の分離は #998 が所有します。
- 本参照は新Core kind・生成計画schema・実行機構の採用を主張しません。受理（#267）への接続、生成物の適用、案件固有変更の保持と再生成の実装は未実装です。
- 決定的な再生成、変更影響の局所化、生成物の追跡のうち本参照が扱うのは構造の検査とfingerprintの固定までです。
- 実行専用Operatorが生成時でどんな計算をするかは本参照の埒外です。Escape hatchは宣言（`Effects`）だけです。

## 限界

- **Operatorを実行しません。** fingerprintはグラフ構造と契約の固定であって、計算結果の再現性を直接の証拠にはしません。固定入力・固定実装での同一出力の実測は別証拠が必要です。
- 未確定、候補、受理済み意味、実行観測の区別は合成入力に依存する声明です。実Workspace・実DSL・実PluginでのHELP確認は未実施です。
- 型契約の検査は両側の型が宣言されている場合に限ります（空は未検査）。入力が外部source入力の場合も未検査です。
- `-race` は cgo を伴わない可搬Goでは実行していません。
