# reference-capability-degradation（#2096）

既存のデータベースやProviderに業務アプリを後付けするとき、業務操作が要求する保証（競合検出、監査、トランザクション境界など）と、観測した能力を照合する読取り専用の参照です。能力が足りないときは、**操作を生成しない**か、**明示の弱い保証を宣言する**かを選び、結果を定義とEvidenceに残します。知らせないまま弱い保証で成功させる経路を作りません。

データベースを変更せず、コードを生成しません。縮退した結果を「同等」と認証しません。

## 実行

```bash
go run ./tools/reference-capability-degradation/cmd/tei-capability-degradation \
  -input tools/reference-capability-degradation/testdata/input-existing-db.json
```

結果はJSONで標準出力へ出します。`blocked`または`unresolved`のときは終了コード1、`passed`または`degraded`のときは0です。`degraded`は宣言された縮退であり、失敗ではありません。

## 入力

```json
{
  "schema": "tei.reference.capability-degradation-input/v0",
  "targetId": "existing-db",
  "tables": [
    {"id": "orders", "observed": true,
     "capabilities": ["read", "insert", "update", "delete", "optimistic-lock-column", "audit-column"]},
    {"id": "legacy_orders", "observed": true,
     "capabilities": ["read", "insert", "update", "delete"]}
  ],
  "operations": [
    {"id": "orders.update", "table": "orders", "action": "update",
     "requires": ["update", "optimistic-lock-column"]},
    {"id": "legacy.update", "table": "legacy_orders", "action": "update",
     "requires": ["update", "optimistic-lock-column"]},
    {"id": "legacy.audit-update", "table": "legacy_orders", "action": "update",
     "requires": ["update", "optimistic-lock-column", "audit-column"],
     "allowDegradation": true,
     "weakerGuarantee": "更新のみ。競合検出と監査列は使わない"}
  ]
}
```

- `observed`が偽の表は、能力を確認していません。不足を「確認済みの事実」にしません
- `action`は`read`・`insert`・`update`・`delete`です

## 判定

| 状態 | 条件 |
| --- | --- |
| `generated` | 要求する能力がすべて揃っている |
| `withheld` | 能力が不足し、弱い保証の宣言が無い。操作を生成しない |
| `degraded` | 能力が不足し、`allowDegradation`と`weakerGuarantee`が宣言されている。弱い保証を記録する |
| `rejected` | 定義の誤り。未知の表、`requires`の欠落、更新・削除が`optimistic-lock-column`を要求していない、不足が無いのに縮退を宣言している |
| `unresolved` | 表が未観測。不足を確認済みの事実にできない |

`update`と`delete`は、`optimistic-lock-column`を`requires`に含めなければ`rejected`です。要求そのものを書き忘れた定義で、後勝ちの更新を通さないためです。

## 報告

- `operations[]` — 操作ごとの状態、不足する能力、弱い保証、理由
- `degradations[]` — 縮退した操作の一覧。表・不足する能力・処分（`withheld`／`degraded`）・弱い保証・`confirmed`
- `unresolved[]` — 未観測の表に依存する操作
- `certifiedEquivalent` — 常に`false`。`equivalence`は常に`not-certified`

`confirmed`は、観測済みの表で能力の不在を確認した事実だけに付きます。未観測の表は`unresolved`に分け、`degradations`へ入れません（P11）。

## 規則

- 能力が不足するとき、生成・拒否・明示の弱い保証のどれを選んだかを、定義とEvidenceに残す
- 縮退した操作を「同等」と認証しない（#1180、#1969の分類と整合する）
- 不足を、未確認（`unresolved`）と確認済み（`confirmed`）に分ける
- 既存の表を自動で改修しない。排他制御の方式を統一しない

## 範囲と非主張

- 入力は合成の例です。実在のデータベースの観測結果ではありません
- データベースを変更せず、コードを生成しません
- 同時更新の実動作は試験しません。列の無い表で更新操作を生成しないことを、定義の側で確かめます
- 実Provider・実Target・本番配備、経路間の同等性（#1180）、副作用フェンス（#1174）は対象外です
- `degraded`は弱い保証の宣言であり、意味の受理や実行権限を示しません
