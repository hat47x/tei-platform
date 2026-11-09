## 調査結果

指定の `reference/workspace` はカレントに存在しなかったため、実体の `workspace/` を `-workspace .\workspace` として参照しました。使用したCLIはすべて読み取り専用です。

### 1. 実際に呼び出したCLIと結果

| コマンド | 結果の要点 |
|---|---|
| `.\bin\tei.exe version -json` | `go1.27.0`、`vcsRevision: 50ebf8014340fd1906f6b35cfa30173e0ee2fc6c` |
| `.\bin\tei.exe commands` | 利用可能なコマンドは `selected-read-only-commands`。今回使った `model catalog` / `inspect` / `source-index` / `trace` / `validate` 等は read-only |
| `.\bin\tei.exe model catalog -workspace .\workspace -namespace sales` | `semantic-data:sales.order` は `symbol: Order`、`sourceStatus: indexed`。`modelRevision: sha256:f414e2b0...` |
| `.\bin\tei.exe inspect -workspace .\workspace -subject-kind semantic-data -subject-id sales.order` | 属性6件・関係2件を取得。定義元は `semantic/sales/order.tei` |
| `.\bin\tei.exe inspect -workspace .\workspace -subject-kind semantic-attribute -subject-id sales.order.total` | `sales.order.total` は `symbol: total`、`type: std.money`。定義は `order.tei` 12行目 |
| `.\bin\tei.exe inspect -workspace .\workspace -subject-kind semantic-type -subject-id std.money` | `kind: primitive`、`sourceStatus: not-indexed`、artifactなし |
| `.\bin\tei.exe source-index -workspace .\workspace -subject-kind semantic-data -subject-id sales.order` | artifactは `semantic/sales/order.tei` 1件。定義参照は line 7, col 6–11、role `definition` |
| `.\bin\tei.exe trace -workspace .\workspace -scope full -subject-kind semantic-data -subject-id sales.order -mode full` | 解決済みエッジ4件、未解決0件。`projectionUnresolved` は17件 |
| `.\bin\tei.exe validate -workspace .\workspace -json` | `ok: true`、警告3件、エラー0件 |
| `.\bin\tei.exe workspace-files -workspace .\workspace` | `semantic/sales/order.tei` の存在を確認 |

### 2. 要素が表すもの

`semantic-data:sales.order` は、namespace `sales` におけるデータ宣言 `Order` です。受注ヘッダを表し、以下の属性と関係を持ちます。

- 属性: `orderId`、`customerId`、`orderedAt`、`status`、`total`、`externalId`
- 関係: `customer`（`customer.customer` への one）、`lines`（`sales.order_line` への many）

`trace` では、`order_entry.order_record` と `order_entry.recent_orders` からの `data` 参照、および `sales.order.submit` の `input` / `output` 参照がすべて解決済みでした。

### 3. 定義元ファイルと出典の手掛かり

- 定義元: `semantic/sales/order.tei` の7行目 `data Order {`（line 7, col 6–11、offset 124–129）
- artifact fingerprint: `sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33`
- `source-index` が返す定義参照は上記1件のみで、roleは `definition`
- ファイル内には上流の出典・著者・ライセンス・URLの記載はなく、**それ以上の出典は未確認**

関連する下流の手掛かりとして、`databases/sales-core/contracts/order.tei` は `read sales.order` / `write sales.order.status` を宣言し、`releases/databases/sales-core/4.2.0.tei` が `sales_core.order v4.2` を提供しています。ただしこれらは定義元ではなく、下流の参照です。

### 4. 注文の合計額を表す属性

- 識別子: `sales.order.total`
- シンボル: `total`
- 型: `std.money`
- 定義位置: `semantic/sales/order.tei` 12行目（line 12, col 15–20）

`std.money` は `primitive` とされていますが、`sourceStatus: not-indexed` でartifactがありません。したがって、通貨単位・精度・スケールなどの具体的な意味は**未確認**です。なお、DB側の `persistence/order.tei` では `sales.order.total` を `total_amount` / `decimal` にマッピングしていますが、これは保存表現であり、semantic typeそのものではありません。

### 5. Workspaceの検証結果

`validate` は `ok: true` で、エラーはありません。警告は以下の3件です。

1. `DATABASE_RELEASE_CONTRACT_NOT_CURRENT`: `sales_core` release 4.0.0 の `sales_core.order_legacy` に現行定義がない
2. 同 4.1.0 について同じ警告
3. `APPLICATION_RELEASE_CONTRACT_NOT_CURRENT`: `order_entry@3.7.0` が現行定義のない `sales_core.order_legacy` を要求

いずれも `sales.order` 自体ではなく、旧contract `sales_core.order_legacy` に関する警告です。

### 6. 残る診断・不明点

- `std.money` の実体定義はWorkspaceに索引されておらず、金額の具体的仕様は**未確認**。
- `trace` の `projectionUnresolved` に `unmaterialized-from:database-contract:sales_core.order`、`unmaterialized-to:database-contract:sales_core.order` などが残る。DB層への投影は完全にはmaterializeされておらず、`source-evidence-not-bound` / `business-procedure-not-in-current-world` / `derived-index-is-not-authoritative-world` の制約もある。
- `semantic/sales/order.tei` に上流出典のメタデータがないため、Workspace外の原典・版は**未確認**。
- 指定の `reference/workspace` は存在せず、実際は `workspace/` を参照した（パス相違）。

### 7. 参照したモデル・Sourceの版の区別

| 種別 | 値 | 出所 |
|---|---|---|
| CLI build | `go1.27.0` / `vcsRevision 50ebf801...` | `version -json` |
| モデル全体の版 | `sha256:f414e2b09d2f6ea5d0f5f282aea55612985c0e8155877ad8d310250633cf3fd2` | `model catalog` |
| Source index | `sha256:1827f4b46169710cbe26179498bf30ef775aecba858c3beff8e2e0d53ca23531` | `model catalog` / `inspect` |
| World snapshot | `sha256:c92de09f9595b1f46bfc18e6f99260d8391113b6d2639357d372abf634a1d65a` | `model catalog` / `inspect` / `validate` |
| 定義ファイル `order.tei` の版 | `sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33` | `source-index` / `inspect` |
| trace snapshot | `sha256:fe15b202fc7d26f93debea90d19db30a4ef93fa69b02bb62218fab8fe7526668` | `trace` |
| trace modelref snapshot / rule | `sha256:a6975be7...` / `sha256:6a71839a...` | `trace` |
| DB release / contract | `sales_core 4.2.0` / `sales_core.order v4.2` / `schema-fingerprint "reference-sales-4.2"` | `releases/databases/sales-core/4.2.0.tei` |
| App release / source revision | `order_entry 3.8.0` / `source-revision "reference-app-3.8.0"` | `releases/applications/order-entry/3.8.0.tei` |

以上より、`sales.order` の定義と合計額属性はWorkspace内で確認できましたが、`std.money` の実体定義とWorkspace外の原典は**未確認**です。