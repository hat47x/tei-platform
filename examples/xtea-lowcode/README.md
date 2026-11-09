# X-TEAローコード知見の受容例

X-TEAのローコード知見から受容した表現を、小さな作業領域で確認するための例です。設計は[受容アーキテクチャ](../../plan/design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)、知識の出典は[知識目録](../../plan/research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)、受容条件は[受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)を参照してください。

## 構成

- `semantic/sales/order.tei`: 受注・明細・支払、作成・削除・`submit_order`能力、`sales.month`期間、`sales.fiscal`暦（年度開始4月・週開始月曜・休日Data）、`sales.month`の20日締め、受注の候補キー、明細から受注への`on-delete cascade`、支払から受注への`on-delete restrict`とkey対応。
- `semantic/customer/customer.tei`: 顧客。
- `semantic/master/standard.tei`: 標準業務のData。休日、税率（適用期間の両端・端数処理）、為替（率の種類・会計年度・月順序）、採番（規則識別・桁・接頭辞・検査数字・桁あふれ）、利用者設定（設定の鍵・値・利用者）。属性は列名ではなく`role`で結合先を示す。
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
- `view -command -execute`は参照Executorを使うため、この例の`update`効果は記録済みデータがない状態では成功しません。応答の`committed`は参照用の取引Commit呼出し、`persisted`は常に`false`、`durability`は`none`です。
- 標準業務のDataは宣言と結合の例であり、税率・為替・採番の行データは含みません。`internal/taxrate`・`internal/currencyrate`・`internal/numbering`の`Resolve`で役割を解決するところまでを例で確かめられます。`view -application order_entry`は、宣言された標準業務Dataを`standardBusiness`（種別・Data・属性）として投影します。
- 受容判断は仮承認であり、最終採否は別途です。
## Studioで開く（任意）

Studioのサーバービルドから、この作業領域を読み取り専用で開けます。

```bash
cd cmd/tei-studio
CGO_ENABLED=0 go build -mod=readonly -tags server -o tei-studio-server .
./tei-studio-server -workspace ../../examples/xtea-lowcode -port 43123
```

別ターミナルで`http://127.0.0.1:43123/`を開きます。単票ビュー、Command計画、参照実行、Packageビューを確認できます。

Studioはloopback限定の読取り専用Hostであり、接続者認証・業務操作の認可・利用者間の分離を提供しません。詳細は[Studio README](../../cmd/tei-studio/README.md)を参照してください。
server buildと`/health`の確認記録は[Studioサーバー起動確認](../../plan/evidence/XTEA_LOWCODE_EXAMPLE_STUDIO_SMOKE_2026-11.md)にあります。画面表示・Wails RPC・Desktop WebViewは未確認です。



