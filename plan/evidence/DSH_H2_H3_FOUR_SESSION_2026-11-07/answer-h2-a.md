## 結論

`semantic-data:sales.order` は、`sales` 名前空間で定義されたセマンティックデータ要素 `data Order`（表示記号 `Order`）です。

CLI の `inspect` で確認した構造は次のとおりです。

- 属性: `orderId`、`customerId`、`orderedAt`、`status`、`total`、`externalId`
- 関係: `customer`（`customer.customer` への 1）、`lines`（`sales.order_line` への 多）
- 機能: `sales.order.submit` の入力・出力

モデル内に説明文はありませんが、名前と構造からは**注文ヘッダを表すデータ要素**と読めます。

**注文の合計額に対応する属性**は次のとおりです。

| 項目 | 値 |
|---|---|
| 属性識別子 | `sales.order.total` |
| 記号 | `total` |
| セマンティック型 | `std.money`（プリミティブ） |
| 物理実現（派生コンテキスト） | `sales_core.order` → `public.orders.total_amount`、storage `decimal`、required |

`sales.order_line.unit_price` は明細の単価であり、注文合計ではありません。

## 定義元と出典の手掛かり

- 定義ファイル: `semantic/sales/order.tei`
- `Order` の定義: 7行目 `data Order {`、offset 124–129、6–11列、`role: definition`
- `total` の定義: 12行目 `attribute total type std.money`、offset 326–331、15–20列
- Source artifact fingerprint: `sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33`
- ファイル形式: `tei`、863 bytes

当該TEIファイル内には、外部文献・著者・ライセンスなどの出典記載はありません。また、この要素について `source-index` が返した定義元は当該ファイルのみで、外部原典への参照は返っていません。したがって**外部原典は未確認**です。

## 実行したCLIと結果

CLI は `tei <command> [options]` の順で実行しました。`-workspace` をコマンドより前に置くと `exactly one of -workspace or -world is required` で失敗したため、コマンドを先にしています。

| コマンド | 結果 |
|---|---|
| `tei commands` | `scope: selected-read-only-commands`。一覧のコマンドはすべて `effect: read-only` |
| `tei version -json` | revision `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`、`go1.27.0 windows/amd64` |
| `tei model catalog -workspace=<REF>` | `ok: true`。`sales.order` は `semantic-data` として `sourceStatus: indexed` |
| `tei inspect -workspace=<REF> -subject-kind semantic-data -subject-id sales.order` | 上記の属性6個・関係2個を取得。定義範囲は `order.tei` 7行目 |
| `tei inspect -workspace=<REF> -subject-kind semantic-attribute -subject-id sales.order.total` | `type: std.money`、定義範囲は `order.tei` 12行目 |
| `tei source-index -workspace=<REF> -subject-kind semantic-attribute -subject-id sales.order.total` | artifact `semantic/sales/order.tei`、fingerprint `f206…`、`role: definition` |
| `tei knowledge-context -workspace=<COPY> -subject-kind semantic-data -subject-id sales.order` | 物理実現、incoming 参照、未解決事項を取得 |
| `tei validate -workspace=<REF> -json`（text も実行） | `ok: true`、終了コード `0`、警告3件 |

`<REF>` は `D:\GIT\hat47x\tei-wt-dsh-h2-20261107\reference\workspace`、`<COPY>` は実行用コピー `C:\Users\yhata\AppData\Local\Temp\h2-participant-20261107\workspace` です。両方で `model catalog`・`inspect`・`source-index`・`validate` を実行し、`worldFingerprint` と `sourceIndexFingerprint` が一致することを確認しました。

## モデルとSourceの版の区別

| 種別 | 版の手掛かり |
|---|---|
| CLI | `tei.cli.version/v0`、revision `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c` |
| モデル（派生World） | `modelRevision: sha256:f414e2b09d2f6ea5d0f5f282aea55612985c0e8155877ad8d310250633cf3fd2` |
| World全体 | `worldFingerprint: sha256:c92de09f9595b1f46bfc18e6f99260d8391113b6d2639357d372abf634a1d65a` |
| Source Index | `sourceIndexFingerprint: sha256:1827f4b46169710cbe26179498bf30ef775aecba858c3beff8e2e0d53ca23531` |
| Source `semantic/sales/order.tei` | `sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33`、863 bytes |
| Workspace manifest | `tei_reference`、`formatVersion: 1` |

Source には人間可読な版番号がなく、内容の SHA-256 fingerprint で識別されています。`modelRevision` は派生モデルのリビジョンであり、Source ファイルの版そのものではありません。なお `reference/README.md` 9行目には、参照Workspaceを **M0.22.1** の設計要素を組み合わせたものと説明していますが、これは個々のモデル・Sourceの版番号ではありません。

## 検証結果と残る診断・不明点

`validate` は `ok: true`、終了コード `0` でした。エラー重大度の診断は出力されず、警告は次の3件です。

1. `DATABASE_RELEASE_CONTRACT_NOT_CURRENT` — `sales_core` 4.0.0 が履歴契約 `sales_core.order_legacy` を提供しているが、現行定義がない
2. `DATABASE_RELEASE_CONTRACT_NOT_CURRENT` — 同じく `sales_core` 4.1.0
3. `APPLICATION_RELEASE_CONTRACT_NOT_CURRENT` — `order_entry@3.7.0` が履歴契約 `sales_core.order_legacy` を要求しているが、現行定義がない

いずれも `sales_core.order_legacy` に関する過去契約の警告で、`sales.order` 自体の定義欠落ではありません。`reference/README.md` 17行目も、過去契約の警告が出ても最後に `ok` なら意図した状態と説明しています。

残る不明点は次のとおりです。

- `std.money` はプリミティブ型としてカタログにありますが、`std` 名前空間は `sourceStatus: not-indexed` です。**通貨・精度・丸めなどの型定義はWorkspaceのSource Index上では未確認**です。
- `knowledge-context` は、`businessOccurrenceIdentifier`、`candidateKeys`、`functionalDependencies`、`compositeDomainConstraints`、`temporalHistoryConditions` を `unresolved` としています。関係の `businessMeaning` と `temporalSemantics` も `unresolved` です。
- 当該TEIファイルには説明文・外部出典の記載がないため、**名前と構造以上の業務的な意味や外部原典は未確認**です。
- これらは `validate` のエラーではなく、派生コンテキストの未解決・情報不足です。CLI自身も「派生コンテキストは権威ではない」と明示しています。

調査は閲覧のみで、ファイルの書換え・生成、DB/Providerへの業務操作は行っていません。