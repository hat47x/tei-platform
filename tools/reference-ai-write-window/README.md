# reference-ai-write-window（#2092）

Pluginが宣言した**書込み窓**（AIが書いてよい領域）に、AIの編集操作が収まっているかを検査する読取り専用の参照です。窓の外への書込みは受理前に拒否し、窓の内側でも意味・権限が変わる変更は人の判断へ戻します。受理候補は業務実行の許可ではありません。

編集を適用せず、受理・配備もしません。

## 実行

```bash
go run ./tools/reference-ai-write-window/cmd/tei-ai-write-window \
  -input tools/reference-ai-write-window/testdata/input-window.json
```

終了コード:

| 状態 | 終了コード |
| --- | --- |
| `candidate-accepted` | 0 |
| `human-decision-required` | 2 |
| `blocked` / `unresolved` | 1 |

## 入力

```json
{
  "schema": "tei.reference.ai-write-window-input/v0",
  "mode": "window",
  "window": {
    "id": "case.order-discount",
    "declaredBy": "human",
    "sourceRevision": "rev-10",
    "providerRevision": "order-plugin-1.0.0",
    "allowedPaths": ["src/order/discount.go"],
    "allowedExpressions": ["Order.line.total", "customer.tier"],
    "allowedTypes": ["decimal", "string"],
    "allowedEffects": ["read"],
    "allowedKinds": ["expression", "datatype"]
  },
  "edit": {
    "id": "candidate-1",
    "author": "ai",
    "baseSourceRevision": "rev-10",
    "baseProviderRevision": "order-plugin-1.0.0",
    "changes": [{"path": "src/order/discount.go", "kind": "expression",
                 "expressions": ["Order.line.total"], "types": ["decimal"], "effects": ["read"]}],
    "helpers": [{"name": "sumLines", "effects": ["read"], "resolved": true}],
    "tests": [{"command": "go test ./src/order", "head": "abc1234", "exitCode": 0,
               "unverified": ["real PostgreSQL"]}]
  },
  "targets": ["go", "wasm"],
  "targetCapabilities": {"go": ["read"], "wasm": ["read"]}
}
```

`mode`は`window`または`whole-generation`です。`window.declaredBy`は`human`でなければなりません。

## 判定

| 状態 | 条件 |
| --- | --- |
| `candidate-accepted` | 窓の内側で、人の判断が不要で、試験証跡があり、対象の能力が揃っている |
| `human-decision-required` | 窓の内側だが、意味または権限が変わる（A4） |
| `unresolved` | 情報が足りない（HelperのEffectが未解決、試験証跡が不完全、対象が不明、全体生成の往復一致・承認が無い） |
| `blocked` | 窓の外への書込み、版のずれ、AIによる窓の宣言、全体生成の本番書込み、破壊的操作の承認・退避の欠落 |

## 検査する内容

- 窓の宣言: `declaredBy`が`human`であること
- 版: `baseSourceRevision`と`baseProviderRevision`が窓の版と一致すること（ずれは`stale`）
- path・式・型・変更種別・Effect: 窓の許可リストとの一致
- 窓の内側でも`semanticChange`・`authorityChange`が真なら人の判断へ戻す
- Helper: 未解決なら`unresolved`、解決済みで窓の外のEffectを持つなら`blocked`
- 試験証跡: `command`・`head`・`exitCode`・`unverified`を記録する。自己申告だけでは足りない
- 破壊的操作: 人の承認と変更前の退避が無ければ`blocked`
- 全体生成: 本番へ直接書けない、隔離ブランチだけ、往復一致と人の承認が必要

## W08〜W11 との対応

| ID | 検査 |
| --- | --- |
| W08 | 許可式から呼ぶHelperのEffectが未宣言なら`unresolved`、確認済みの違反なら`blocked` |
| W09 | 参照Plugin/Providerの版が変わっていれば`stale`として拒否し、古い証拠を流用しない |
| W10 | 提案後にSourceが改訂されていれば`stale`として拒否し、Candidateは保持する（適用しない） |
| W11 | 対象ごとにEffectの可用性を表示する。あるTargetで使えなくても、業務実行の許可とは区別する |

## 範囲と非主張

- 入力は合成の例です。実在のPlugin・業務アプリの定義を写したものではありません
- 編集を適用せず、受理・配備もしません
- `candidate-accepted`は受理候補であり、意味の受理・権限の付与・業務実行の許可を示しません
- AIの自己申告や生成成功を、検証の証拠にしません
- 実Target・実Host・実Provider、完全自動の受理・配備は対象外です
