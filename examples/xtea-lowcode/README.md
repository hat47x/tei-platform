# X-TEAローコード知見の受容例

X-TEAのローコード知見から受容した表現を、小さな作業領域で確認するための例です。設計は[受容アーキテクチャ](../../plan/design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)、知識の出典は[知識目録](../../plan/research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)、受容条件は[受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)を参照してください。

## 構成

- `semantic/sales/order.tei`: 受注と明細、作成・削除・`submit_order`能力、`sales.month`期間、受注の候補キー、明細から受注への`on-delete cascade`とkey対応。
- `semantic/customer/customer.tei`: 顧客。
- `applications/order-entry/`: 一覧、単票、参照、見出し明細、command、hook、Package、作成・削除のcommand。
- `applications/order-entry/queries/`: 一覧の問い合わせとSQL。
- `databases/sales-core/`: 参照先データベース。

## 確認

```bash
go run ./cmd/tei validate -workspace examples/xtea-lowcode

go run ./cmd/tei view \
  -workspace examples/xtea-lowcode \
  -application order_entry \
  -pattern order_entry.order_record \
  -format text

go run ./cmd/tei view \
  -workspace examples/xtea-lowcode \
  -application order_entry \
  -pattern order_entry.order_record \
  -command sales.submit_order \
  -input order=ORD-1 \
  -format text

go run ./cmd/tei view \
  -workspace examples/xtea-lowcode \
  -application order_entry \
  -package order_entry.list_to_record \
  -format text
```

`validate`は`UNBOUND_DATASOURCE`を警告として返します。この例は環境へのデータベース束縛を持たないためです。

### 期間の閉塞

`sales.month`は`update`を閉期間で拒否します。

```bash
printf '{"sales.month":"closed"}' > /tmp/xtea-periods.json

go run ./cmd/tei view \
  -workspace examples/xtea-lowcode \
  -application order_entry \
  -pattern order_entry.order_record \
  -command sales.submit_order \
  -input order=ORD-1 \
  -execute \
  -period-state /tmp/xtea-periods.json \
  -format text
```

`COMMAND_PERIOD_CLOSED`で拒否されます。

## 範囲

- 表示・Command計画・Package投影の参照例です。
- 実データベース、実Provider、認可、receiptは含みません。
- `view -command -execute`は参照Executorを使うため、この例の`update`効果は記録済みデータがない状態では成功しません。
- 受容判断は仮承認であり、最終採否は別途です。

