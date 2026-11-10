# 永続化マッピングの列対応が参照グラフに現れていなかった（修正）

- 記録日: 2026-11-10
- 対象: `internal/index/index.go`
- 関連: #4001、#3989（同じ「索引にあるが辺にしない」型の欠陥）、#3906

## 観測（修正前）

`column <attribute> -> <name>` は「この属性はこの列に格納される」という宣言ですが、`trace` で辿れませんでした。

```text
tei trace -scope s -subject-kind semantic-attribute \
  -subject-id sales.order.total -direction incoming -mode full
# total: 0
```

`sales.order.total` は `databases/sales-core/persistence/order.tei` と `applications/order-entry/queries/recent_orders.tei` の2箇所から参照されていました。全20属性のうち incoming を持つのは3件だけで、それは `semantic-validation -attribute->` の辺を持つものだけでした。

## 原因

2段階で落ちていました。

1. `index.PersistenceIndexEntry` が `Mapping`・`Database`・`SemanticData`・`Table` しか持たず、**列対応（`Mapping.Columns`）を捨てていた**。
2. `modelref/derivedbridge` の投影が `PersistenceMappings` を一度も参照していなかった。

## 変更（#3989 と同じく索引層で解く）

- `PersistenceIndexEntry` に `Columns []PersistenceColumnIndexEntry`（`Attribute`・`Column`）を追加し、`Mapping.Columns` を保持する。
- 各列対応を `DeclaredDependency` に写す: `semantic-data --maps-column--> semantic-attribute`、列名は `Alias` に載せる。
- 属性が空・列名が空の組は辺にしない（投影が `unmaterialized` として報告するだけの辺を作らない）。

**向きの判断**: `data` を from、`attribute` を to にした。宣言は「データの属性がこの列に入る」であり、両端とも既存ノード（`semantic-data`・`semantic-attribute`）で完結する。列名は subject ではないので `Alias` に載せ、ノードにしない（#3989 で adapter のファイル名をノードにしなかったのと同じ理由）。

## 検証

- `gofmt -l internal`: 空、`go build ./...`: PASS
- `go test ./internal/index ./internal/modelref/... ./internal/operations ./tools/reference-cli-trace-benchmark ./tools/reference-mcp-readonly-adapter -count=1`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 実測（実バイナリ、`reference/workspace`）:
  - `trace -subject-kind semantic-attribute -subject-id sales.order.total -direction incoming` が `sales.order --maps-column--> sales.order.total` を返す。
  - **incoming を持つ属性が 3件 → 12件**（全20件中）。
  - 残る8件（`customer.customer.name` など）は**列対応が宣言されていない**属性であり、辺にならないのが正しい。`customer.customer.name` は `attribute name type std.text` の宣言だけで、`column` 宣言が無いことを確認した。
- 追加試験 `internal/index/column_mapping_test.go`:
  - 列対応が `maps-column` の辺になり、`Alias` が列名を運ぶこと。
  - 索引のデータとしても `Columns` が残ること（辺だけでなく直接読める）。
  - 列対応を持たないマッピングは辺を作らないこと。
  - 属性または列名が空の組は辺にも索引にも残らないこと。
- 負例: `Relation: "maps-column"` を別名に変えると `no maps-column edge; declared edges: [...]` で失敗する（コードは戻した）。
- 既存の網羅性テスト `TestEntryTypesKeepOrExplainEverySourceField`（main の並行変更で追加）が、`Columns` を「落としている」と記録していたため失敗した。実際に保持するようになったので、その記録から `Columns` を外した。**この表は「どのフィールドを落としたか」の判断記録であり、判断を変えたときに更新するもの**である。

## 主張しないこと

- 残る8属性に列対応を足していない（`reference/workspace` の内容変更は `reference-shared-acceptance` の受理経路が要る）。
- `-subject-kind` に `persistence` を追加していない。Issue が挙げていたが、今回の欠陥は属性側の incoming が0件であることで、`semantic-attribute` で辿れるようになった時点で解消する。マッピング自体を subject として引く需要は別の判断である。
- `RelationStorage`（中間テーブル等）は辺にしていない。Issue が指すのは `column A -> B` の対応である。
- 実 Host・認可・外部送出は NOT RUN。
