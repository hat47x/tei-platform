# Data Derivation D10：導出項目のProjection意味境界 Reference（#1707）

本ディレクトリは **一つのProjectionイベントを読取り専用で検査する小さな参照** であり、TEIの正式なEditor、CRUD生成器、永続化実行器、Renderer、受入済みAPIではありません。実装主体は `derivationprojection.go`、受入は `derivationprojection_test.go`、起動点は `cmd/tei-derivation-projection/` です。`正本`を書き換える経路はありません。

## 責務と分界

- **#1124 / #997 / #1127**：モデラー編集、UI Widget、標準業務画面の実装は委譲します。本参照は意味状態の判定だけを担います。
- **#1641 / #1129**：確定保存のOwnerは委譲先です。本参照は確定値を上書きしない境界を示します。
- **#1693**：項目権限は委譲し、権限不足は `AUTHORIZATION_DENIED` とします。
- **#267**：提案→受理のフローは既存経路を消費し、第二Editorを作りません。
- Actionの永続化実行器、Command/Transaction、汎用Reactive計算エンジンは実装しません。

## 四つの意味状態

| 状態 | 意味 |
| --- | --- |
| `declared` | Authoritative Source上の規則とBinding。実行結果ではない。 |
| `preview` | 未保存入力の一時評価。保存・確定とはみなさない。 |
| `runtime-derived` | 受理済み規則を現在の文脈・認可で評価した表示値。 |
| `confirmed-saved` | 業務Actionで保存された事実。再描画・入力変更・計算式更新で上書きしない。 |

## 判定（採用前）

| 入力イベント | 期待 | 結果コード |
| --- | --- | --- |
| 未保存の入力変更 | プレビューだけ更新 | `PREVIEW_UPDATED` |
| 実行時再評価 | 実行時導出値を更新 | `PROJECTION_UPDATED` |
| 導出値セルへ直接入力 | 編集不可。書込みを拒否 | `EDIT_FORBIDDEN` |
| 確定保存値への再評価／マスター変更 | 保存値は不変 | `SAVED_FACT_IMMUTABLE` |
| 業務Actionで確定 | 確定保存値として凍結 | `CONFIRMED_SAVED` |
| 確定時の入力未取得／計算不能 | 確定を拒否 | `RECOMPUTE_UNAVAILABLE` |
| 権限不足 | 古い値を確定値として見せない | `AUTHORIZATION_DENIED`（値は不明） |
| 受理前の提案 | Canonical・保存値は不変 | `PROPOSAL_NOT_APPLIED` |
| 受理後の反映 | Projectionへ反映 | `PROPOSAL_APPLIED` |
| 別Rendererへ同じ意味を渡す | 意味状態が同一 | `RENDERER_AGNOSTIC` |
| Rendererが異なる値を表示 | 二重正本として拒否 | `RENDERER_CONTRACT_VIOLATION` |

## 合成Fixture（受注明細）

明細数量2、単価300 → 一時表示金額600。

| 操作 | 本参照の結果 |
| --- | --- |
| 数量を3へ未保存変更 | `PREVIEW_UPDATED`、`value=900`（DB上の保存事実は不変） |
| 導出金額セルに直接 `999` | `EDIT_FORBIDDEN` |
| 未取得単価で再評価 | `RECOMPUTE_UNAVAILABLE`、値は不明（古い600を継続表示しない） |
| 確定600後にマスター単価300→350 | `SAVED_FACT_IMMUTABLE`、`value=600`（新規未確定の計算は700） |
| 受理前のAI生成式プレビュー | `PROPOSAL_NOT_APPLIED`（保存値不変）／受理後 `PROPOSAL_APPLIED` |
| Web／CLI／専用UIへ同じ意味 | `RENDERER_AGNOSTIC`、`value=600` |

## 実行方法

```sh
go run ./tools/reference-derivation-projection/cmd/tei-derivation-projection -input request.json
```

`-input -`（既定）は標準入力から読みます。入力例：

```json
{
  "state": "confirmed-saved",
  "value": "600",
  "event": "master-change",
  "newValue": "700"
}
```

## 限界

- 合成入力に対する静的な判定です。実Renderer、実UI、実DBでの意味同等性の証拠ではありません。
- `-race` は cgo を伴わない可搬Goでは実行していません。
- M2/M3（#1124／#1127／#997 への接続、案件固有UIのOverride確認）は本PRの範囲外であり、未実証として残します。
