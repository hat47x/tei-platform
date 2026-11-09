# reference-generator-extension-points（#2090）

生成器が宣言した**拡張点**に、案件側のコードを生成物の外から結び付けるための読取り専用の参照です。契約の宣言、再生成での保持、契約変更の検出、拡張点の外の直接編集の検出、業務結果の証拠を扱います。

生成物を書き換えず、再生成も適用もしません。

## 実行

```bash
go run ./tools/reference-generator-extension-points/cmd/tei-generator-extension-points \
  -input tools/reference-generator-extension-points/testdata/input-bound.json
```

結果はJSONで標準出力へ出します。`conflict`または`unresolved`のときは終了コード1、`passed`のときは0です。

## 入力

```json
{
  "schema": "tei.reference.generator-extension-points-input/v0",
  "generator": {
    "id": "std.order.generator",
    "version": "1.0.0",
    "extensionPoints": [
      {"id": "compute-derived", "version": "1", "order": 10,
       "inputs": ["amount"], "outputs": ["derivedAmount"], "default": "amount"}
    ],
    "artifact": [{"path": "generated/order.go", "content": "..."}],
    "outcomes": [
      {"point": "compute-derived", "outcome": "order-total",
       "default": "100", "withExtension": "120"}
    ]
  },
  "caseExtensions": [
    {"point": "compute-derived", "path": "case/derived.go", "content": "...",
     "contract": {"version": "1", "order": 10,
                  "inputs": ["amount"], "outputs": ["derivedAmount"]}}
  ],
  "workspaceFiles": [
    {"path": "generated/order.go", "content": "..."},
    {"path": "case/derived.go", "content": "..."}
  ]
}
```

- 拡張点は`id`・`version`・`order`（呼ばれる順）・`inputs`・`outputs`・`default`を持ちます
- 案件側の拡張は`artifact`の外の`path`に置きます。`artifact`と同じpathは`conflict`です
- `contract`は、案件が結び付けた時点の契約です。現在の宣言と比べます

## 検査する内容

| 検査 | 結果 |
| --- | --- |
| 宣言 | 拡張点の識別子・版・入出力・呼び出し順の一意性。重複や欠落は`unresolved` |
| 結び付け | `contract`の版・順・入力・出力が現在の宣言と一致するか。違えば`conflict` |
| 再生成 | 3回生成し、生成物と案件側の拡張が別のままで、byte単位に変わらないこと |
| 直接編集 | `workspaceFiles`の生成物が生成器の出力と違えば`direct-edit`、無ければ`missing`。再生成で上書きされる前に報告する |
| 業務結果 | 拡張の有無で変わる`outcome`を、既定値と拡張時の値つきで記録する |

案件側の拡張ファイルを編集した場合は`extension-changed`として記録します。生成物の外なので`conflict`にはしません。

## 状態

| 状態 | 意味 |
| --- | --- |
| `passed` | 契約が一致し、直接編集も欠落も無い |
| `conflict` | 契約変更、拡張点の削除、生成物と拡張のpath重複、直接編集、生成物の欠落 |
| `unresolved` | 入力の矛盾（拡張点の順序重複、識別子の重複など） |

## ADR-0033（Plugin）との境界

この参照が扱うのは、**生成器の契約に含まれる差し込み点**です。既存のPlugin拡張機構（[ADR-0033](../../plan/adr/0033-core-and-plugin-subprojects.md)）との違いは次のとおりです。

| 内容 | 扱い |
| --- | --- |
| 生成器の選択・版・パラメータ | Pluginの選択・設定として表現できる |
| 生成物の外にある案件コードを、ビルドに含める | 生成物の外の資産として表現できる。本参照の対象 |
| 実行時に新しいCapability・権限を足す | 拡張点ではなく、Pluginの契約の変更として扱う。本参照の対象外 |
| 生成器の契約に無い場所へ差し込む | 拡張点ではない。直接編集として検出する |

本参照は、この表の境界を判定せず、記録した契約と成果物だけを比べます。

## 範囲と非主張

- 入力は合成の例です。実在の生成器・Pluginの定義を写したものではありません
- 生成物を書き換えず、再生成・適用・受理もしません
- `passed`は、宣言と成果物の整合を示すだけで、生成物の意味の受理や実行権限を示しません
- 実Target、実Host、実Provider、配布物の同一性は対象外です
- 拡張の有無による業務結果の差は、入力に記録された値の比較であり、実システムでの観測ではありません
