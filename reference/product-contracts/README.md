# プロダクト契約のTEIモデル

TAI配下の5製品をTEIで再表現できるかを検討するための参照workspaceです。実装を置き換えるものではなく、各製品の主要な契約をTEIの型として並べ、共通部分と製品固有部分を分けて見ます。

## 含めるもの

| 製品 | 主な型 |
| --- | --- |
| 共通 | `Evidence` / `Authority` |
| EKI | `TaskRequest` / `Attempt` / `ResultClaim` / `AcceptedResult` |
| SUI | `Document` / `Card` / `Island` / `Edge` |
| SEI | `Decision` / `Unresolved` |
| MMI | `Observation` / `Selection` |

## 能力

| 能力 | 入力 | 出力 |
| --- | --- | --- |
| `product_contracts.eki_task_submit` | `TaskRequest` | `Attempt` |
| `product_contracts.eki_attempt_execute` | `Attempt` | `ResultClaim` |
| `product_contracts.eki_result_accept` | `ResultClaim` | `AcceptedResult` |
| `product_contracts.sui_document_put` | `Document` | `Document` |
| `product_contracts.sei_decision_record` | `Decision` | `Decision` |
| `product_contracts.mmi_observation_select` | `Selection` | `Selection` |

## アプリケーション

`applications/product-flow/application.tei`は、`product_contracts`のsemanticモデルを参照する最小のアプリケーションです。能力の呼出し順や実装はまだ持ちません。

## 確認

```text
go run ./cmd/tei validate -workspace reference/product-contracts
```

## 境界

- 型の宣言であり、各製品の実装・権限・検証を置き換えません。
- 共通の`Evidence`と`Authority`は最小限の項目だけを持ちます。
- 実際の製品への移行は、契約ごとの意味と責務を確認してから行います。
