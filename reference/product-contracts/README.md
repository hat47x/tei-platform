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

`applications/product-flow/application.tei`は、`product_contracts`のsemanticモデルを参照するアプリケーションです。

6つの`command` patternが、それぞれ1つのcapabilityを参照します。

- `eki_task_submit`
- `eki_attempt_execute`
- `eki_result_accept`
- `sui_document_put`
- `sei_decision_record`
- `mmi_observation_select`

patternは能力を参照します。`realizations/`では、各capabilityを`custom go`の実装へ結びつける宣言だけを持ちます。呼出し順・副作用・実装本体はまだ持ちません。

## 確認

```text
go run ./cmd/tei validate -workspace reference/product-contracts
```

## 境界

- 型の宣言であり、各製品の実装・権限・検証を置き換えません。
- 共通の`Evidence`と`Authority`は最小限の項目だけを持ちます。
- `tei validate`は、宣言した型名の実在までを確認します。能力の出力型と次の入力型の接続は検査しません。たとえば`eki_attempt_execute`の出力を別の型へ替えても`ok`になります(`cmd/tei`の試験で確認、[#2410](https://github.com/hat47x/tei-platform/issues/2410))。接続の宣言形式は未決で、Ownerの判断を待ちます。
- 実際の製品への移行は、契約ごとの意味と責務を確認してから行います。
