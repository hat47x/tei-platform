# `semantic-data:sales.order` 調査報告

## 0. 実行環境と版の識別

| 項目 | 確認値 |
|---|---|
| 実行物 | `C:\Users\yhata\AppData\Local\Temp\h2-participant-20261107\bin\tei.exe`（12,006,912 bytes、SHA-256 `186D06CE8C79A259A834C79DE28B675103113D1218B78876737B15BC4CE69707`） |
| CLI版 | `tei.cli.version/v0`、`effect=read-only`、Go 1.27.0 / windows/amd64、`vcsRevision=50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`（`source=build-info`） |
| cwd | `C:\Users\yhata\AppData\Local\Temp\h2-participant-20261107` |
| Workspace | `.\workspace`（`tei.workspace.yaml`: `workspace: tei_reference`） |

**注意**：依頼文の `reference/workspace` というパスは今回のcwdには存在せず、実際に読んだのは `.\workspace` です。両者を同一と推定せず、以下の結果は `tei_reference` に対する観測です。

## 1. 実行したread-onlyコマンド（すべて exit 0）

```
tei.exe version -json
tei.exe commands
tei.exe model catalog -workspace .\workspace
tei.exe inspect -workspace .\workspace -subject-kind semantic-data -subject-id sales.order
tei.exe inspect -workspace .\workspace -subject-kind semantic-attribute -subject-id sales.order.total
tei.exe source-index -workspace .\workspace -subject-kind semantic-data -subject-id sales.order
tei.exe source-index -workspace .\workspace
tei.exe trace -workspace .\workspace -scope reference-observation -subject-kind semantic-data -subject-id sales.order -direction both -mode summary
tei.exe validate -workspace .\workspace -json
tei.exe workspace-files -workspace .\workspace
tei.exe knowledge-context -workspace .\workspace -subject-kind semantic-data -subject-id sales.order
```

`commands` の `scope` は `selected-read-only-commands` であり、全CLI能力の一覧ではありません。

## 2. 要素が表すもの

`inspect` の `canonical` と `semantic/sales/order.tei` の実体から：

- `data Order`（namespace `sales`、symbol `Order`）。注文ヘッダを表すデータ要素で、`sales.order_line` を明細として持つ集約ルート。
- 属性：`orderId`(`sales.order_id`)、`customerId`(`customer.customer_id`)、`orderedAt`(`std.datetime`)、`status`(`sales.order_status`)、`total`(`std.money`)、`externalId`(`std.text`)。
- 関係：`customer` → `customer.customer`（cardinality `one`、キー `sales.order.customer_id` → `customer.customer.customer_id`）、`lines` → `sales.order_line`（`many`）。
- 能力 `sales.order.submit` の input/output が `sales.order`。

## 3. 定義元ファイルと出典の手掛かり

- 定義ファイル：`semantic/sales/order.tei`
- 定義位置：line 7, column 6–11（offset 124–129）、`role=definition`
- ファイル指紋：`sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33`
- `source-index` は上記 path・fingerprint・range を返し、`knowledge-context` も同じ artifact を出典として列挙。

**出典の限界**：ソースファイル自体には著者・URL・ライセンス等の外部出典メタデータはありません。確認できた「出典」はWorkspace内の相対パスと指紋までです。外部原典・来歴は**未確認**です。

下流の手掛かりとして `knowledge-context` は次を返しています（いずれも派生Projectionであり正本受理ではありません）：
- 永続化：`sales_core.order` → table `public.orders`（`databases/sales-core/persistence/order.tei`）
- incoming：`order_entry.order_record`、`order_entry.recent_orders`（data）、`sales.order.submit`（input/output）、`sales_core.order`（reads）

## 4. 合計額属性の識別子と型

- 識別子：`sales.order.total`（symbol `total`）
- 型：`std.money`
- 定義位置：`semantic/sales/order.tei` line 12, column 15–20（offset 326–331）
- 永続化での対応：`total_amount`、storage `decimal`、`required`（`sales_core.order` / `public.orders`）

**不明点**：`std.money` は `semantic-type` として `sourceStatus=not-indexed`（`std` namespace が未索引）です。このWorkspace内に `std.money` の具体定義（精度・通貨単位等）は確認できず、**未確認**です。

## 5. Workspace検証結果

`validate -workspace .\workspace -json` は **exit 0 / `ok=true`** ですが、**Warning 3件**があります（エラーなし）。

| code | severity | 内容 |
|---|---|---|
| `DATABASE_RELEASE_CONTRACT_NOT_CURRENT` | warning | `sales_core` release 4.0.0 が現行定義のない `sales_core.order_legacy` を提供 |
| `DATABASE_RELEASE_CONTRACT_NOT_CURRENT` | warning | 同 4.1.0 |
| `APPLICATION_RELEASE_CONTRACT_NOT_CURRENT` | warning | `order_entry@3.7.0` が現行定義のない `sales_core.order_legacy` を要求 |

したがって「Warningなし」「業務受理済み」とは言えません。exit 0 はWarningを含みます。

## 6. 残る診断・不明点

- `trace`（direction both, summary）：`total=4`、`unresolved=0`。ただし `projectionUnresolved` に `unmaterialized-from/to` が多数（`sales_core.order`、`sales_core.order_line`、`master_data.customer`、各database等）。これはTrace Graphの未解決辺ではなく**Projection側の未materialize**として区別が必要です。
- `knowledge-context` は `resolutionStatus=resolved` / `evidenceCoverageStatus=complete` を返す一方、次を `unresolved` と明示：
  - 関係の `businessMeaning` / `temporalSemantics`
  - `businessOccurrenceIdentifier`、`candidateKeys`、`functionalDependencies`、`compositeDomainConstraints`、`temporalHistoryConditions`
- `std.money` の実体定義は未索引（上記）。
- 依頼文の `reference/workspace` と実読取先 `.\workspace` の関係は**未確認**。
- 外部出典（著者・原典URL等）は**未確認**。

## 7. 版・指紋の対応（混同しないための記録）

| 種別 | 値 |
|---|---|
| `worldFingerprint`（catalog/inspect/trace/validate共通） | `sha256:c92de09f9595b1f46bfc18e6f99260d8391113b6d2639357d372abf634a1d65a` |
| `sourceIndexFingerprint`（同上） | `sha256:1827f4b46169710cbe26179498bf30ef775aecba858c3beff8e2e0d53ca23531` |
| `modelRevision`（catalog） | `sha256:f414e2b09d2f6ea5d0f5f282aea55612985c0e8155877ad8d310250633cf3fd2` |
| 定義ファイル `semantic/sales/order.tei` | `sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33` |
| `trace` 外側 `snapshotFingerprint` | `sha256:c8f86c3d4bc7ec00c7a654daecbf28d186350faf75615b622d1a9ccfc155e62e` |
| `trace` 内側 `trace.snapshotFingerprint` | `sha256:1b5736a19ec80ad78539421f60669691df67bf881a590dee0e07ea5508fceab2` |
| `knowledge-context` dataDesign fingerprint | `sha256:6c419c22b46eee554651f6245ec00423f413a54d9979a9d62c7ed692f714c33e`（`tei.knowledge.data-design/v7`） |

これらは読取り時点の同一性の手掛かりであり、内容の意味受理や書込み権限を意味しません。

## 8. 実施しなかったこと

ファイル書換え・生成、DB/Providerへの業務操作、外部送信は行っていません。モデル変更・受理・コード生成・配備も未実行です。