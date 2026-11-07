## 対象と実行環境

- 対象: `semantic-data:sales.order`
- 実体Workspace: `C:\Users\yhata\AppData\Local\Temp\h2-participant-20261107\workspace`  
  指定の `reference/workspace` はこの環境に存在しなかったため、実在する `workspace` を使用しました。
- 使用CLI: `.\bin\tei.exe`  
  SHA-256 `186D06CE8C79A259A834C79DE28B675103113D1218B78876737B15BC4CE69707`  
  `version -json`: `vcsRevision=50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`、`go1.27.0 windows/amd64`、`effect=read-only`

## 実行したCLIと結果

すべて終了コード `0` でした。

- `model catalog -workspace .\workspace`  
  `semantic-data:sales.order`、symbol `Order`、`sourceStatus=indexed` を確認。
- `source-index -workspace .\workspace -subject-kind semantic-data -subject-id sales.order`  
  定義artifact `semantic/sales/order.tei`、`role=definition`、定義範囲は7行目6〜11列、artifact fingerprint `sha256:f2063418...` を確認。
- `inspect -workspace .\workspace -subject-kind semantic-data -subject-id sales.order`  
  属性・関係を確認。
- `inspect ... -subject-kind semantic-attribute -subject-id sales.order.total`  
  `total`、型 `std.money` を確認。
- `trace ... -mode full -limit 50`  
  直接辺は `total=4`、`unresolved=0`。ただし `projectionUnresolved` は17件。
- `validate -workspace .\workspace -json`  
  `ok=true`、診断3件（すべてwarning）、エラーなし。

## (1) 何を表す要素か

`namespace sales` の `data Order` で、注文を表すデータ要素です。

- 属性: `orderId`、`customerId`、`orderedAt`、`status`、`total`、`externalId`
- 関係: `customer`（`customer.customer` へ1件）、`lines`（`sales.order_line` へ多数）
- 注文明細そのものは別要素 `sales.order_line` で定義されています。

## (2) 定義元と出典の手掛かり

- 定義元: `semantic/sales/order.tei` の7行目 `data Order`
- Source index上の定義範囲: 7行目6〜11列
- Source artifact fingerprint: `sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33`
- 同ファイルの直接SHA-256も上記と一致（863 bytes）
- 同ファイル27行目には capability `sales.order.submit` もあります。
- 補助的に直接読んだ `databases/sales-core/contracts/order.tei` には `contract sales_core.order v4.2 { read sales.order ... }` がありました。これはCLIのtrace辺ではなく、下流contractの参照です。

### 参照した版・指紋

- `worldFingerprint`: `sha256:c92de09f9595b1f46bfc18e6f99260d8391113b6d2639357d372abf634a1d65a`
- `sourceIndexFingerprint`: `sha256:1827f4b46169710cbe26179498bf30ef775aecba858c3beff8e2e0d53ca23531`
- `modelRevision`: `sha256:f414e2b09d2f6ea5d0f5f282aea55612985c0e8155877ad8d310250633cf3fd2`

`catalog`、`inspect`、`trace`、`validate` の `worldFingerprint` と `sourceIndexFingerprint` は一致しています。

## (3) 注文の合計額属性

- 識別子: `sales.order.total`
- シンボル: `total`
- 型: `std.money`

`std.money` はCLI上 `primitive` かつ `sourceStatus=not-indexed` です。通貨・精度・スケールの具体的表現はSourceからは未確認です。

## (4) Workspaceの検証結果と残る診断・不明点

`validate` は `ok=true`、終了コード `0` ですが、warningが3件あります。

- `DATABASE_RELEASE_CONTRACT_NOT_CURRENT`: `sales_core` 4.0.0 の `sales_core.order_legacy` に現行定義がない
- 同 4.1.0 について同じwarning
- `APPLICATION_RELEASE_CONTRACT_NOT_CURRENT`: `order_entry@3.7.0` が現行定義のない `sales_core.order_legacy` を要求

`sales.order` を名指しする診断はありません。エラーはありません。

残る不明点:

- `trace` の直接辺は4件すべて解決済みですが、`projectionUnresolved` が17件あります。`database-contract`、`database`、`database-contract-query` の未materialize参照が残り、層をまたぐ依存の完全な具体化は未確認です。
- `std.money` の通貨・精度・スケールは未確認です。
- `model catalog -namespace semantic-data` は `entries` が空でした。namespaceフィルタの意味は未確認のため、対象同定はフィルタなしのcatalogで行いました。
- CLIの `canonical` はread-only projectionであり、業務意味の正式受理・書込み権限・DB/Providerへの業務Effectは未確認です。

今回は閲覧のみで、ファイル書換え・生成、DB/Provider操作、外部送信は行っていません。