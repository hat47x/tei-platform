# データ層のノードを materialize し、未解決を12件から1件にした（2026-11-10、第50ラウンド）

- 対象: [#4068](https://github.com/hat47x/tei-platform/issues/4068)（起票 → 本ラウンドで解消）、[#4001](https://github.com/hat47x/tei-platform/issues/4001)（依存元）、[#4024](https://github.com/hat47x/tei-platform/issues/4024)、[#3180](https://github.com/hat47x/tei-platform/issues/3180)。
- 前段: [第49ラウンド](../TEI_DOGFOODING_ROUND49_DATA_LAYER_UNMATERIALIZED_2026-11-10/README.md)（発見と起票）。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **前ラウンドで起票した欠陥を修正。**

## 1. 目的

第49ラウンドで起票した #4068 を解消する。`database`・`database-contract`・`database-contract-query` がノードにならず、それらを端点にする辺が全て落ちていた。

## 2. 実装前に読んだこと

| 確認 | 結果 |
| --- | --- |
| 誰かが着手済みでないか | `gh pr list` に該当なし、`projection.go` の直近コミットは `102e5fa09` |
| 案 B（ノードは作るが kind には加えない）は可能か | **不可**。`supports_kind_test.go` が「`SupportsSubjectKind` は `FromIndex` が materialize する集合と一致すべき」と明記 |
| kind を増やすと何が固定されているか | `tools/reference-mcp-readonly-adapter/subject_kind_enum_test.go` の `len(TraceSubjectKinds) != 10`、`trace_workspace.go` の拒否メッセージ |
| 公開値は単一情報源か | はい。`commands`・`-h`・MCP adapter は全て `TraceSubjectKinds` を参照 |

案 B は既存の不変条件に反するため採らず、**案 A（ノードを作り、kind にも加える）**を実装した。

## 3. 修正

`internal/modelref/derivedbridge/projection.go`。

```go
// endpointOnlyKinds are subject kinds the Derived Index names as edge endpoints
// without carrying a dedicated entry list for them.
var endpointOnlyKinds = map[string]bool{
	"database":                true,
	"database-contract":       true,
	"database-contract-query": true,
}
```

`addEdge` の前に、依存リストが名前を挙げている端点を materialize する。

```go
for _, e := range d.SemanticDependencies {
	addEndpointNode(add, e.FromKind, string(e.From))
	addEndpointNode(add, e.ToKind, string(e.To))
}
for _, e := range d.DeclaredDependencies {
	addEndpointNode(add, e.FromKind, string(e.From))
	addEndpointNode(add, e.ToKind, string(e.To))
}
```

`SupportedSubjectKinds` に3種を追加し、`subject_kind_enum_test.go` の固定値（10→13）と拒否メッセージを同時に更新した。**公開値は単一情報源なので、`commands` と `-h` は自動で追随した。**

## 4. 修正後

### 4.1 application からデータ層が見える

```bash
tei trace -workspace reference/workspace -scope c \
  -subject-kind application -subject-id order_entry -direction outgoing -mode full
```

| | 修正前 | 修正後 |
| --- | --- | --- |
| `total` | 14 | **19** |
| `targets-database` | なし | **2** |
| `requires-contract` | なし | **3** |
| `projectionUnresolved` | 12件 | **1件** |

### 4.2 database を直接問い合わせられる

```bash
tei trace -workspace reference/workspace -scope c \
  -subject-kind database -subject-id sales_core -mode full
```

```text
total: 2
  order_entry - targets-database -> sales_core
  sales_core  - uses-semantic    -> sales
```

修正前は `UNSUPPORTED_SUBJECT_KIND` だった。

### 4.3 残った1件は本物の不整合

```text
projectionUnresolved: ["unmaterialized-to:semantic-capability:order_entry.compose_order_summary"]
```

これは第42ラウンドで見つけた「存在しない capability への束縛」で、**ワークスペース側の不整合**。私の変更が作ったものではない。#4001 の依存元として残る。

## 5. 試験

`internal/modelref/derivedbridge/data_layer_test.go` の3件。

| 試験 | 内容 |
| --- | --- |
| `TestDataLayerEdgesAreProjectedNotDropped` | `targets-database`・`requires-contract`・`contract-query` がグラフに入り、`database` を含む未解決が出ない |
| `TestDataLayerNodesAreQueryable` | 3種のノードが存在し、`SupportsSubjectKind` が受け付ける |
| `TestEveryProjectedNodeKindIsSupported` | 投影された全ノード種が `SupportsSubjectKind` で受け付けられる |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 端点 materialize を削除（元の欠陥） | FAIL `edge "targets-database" to the data layer was dropped; declared edges must reach the graph`（3関係すべて） |
| M2 | ノードは作るが kind 一覧から削除 | FAIL `database has a node but SupportsSubjectKind refuses it; trace would reject a subject the graph contains`（3種すべて） |

## 6. 回帰確認

`derivedbridge` を import する全11パッケージ。

| パッケージ | 結果 |
| --- | --- |
| `cmd/tei`（契約・help・discovery） | ok（165.9s） |
| `internal/modelref/derivedbridge`・`internal/operations`・`internal/workspace` | ok |
| `reference/application-composition`・`reference/multilingual-overlays` | ok |
| `tools/reference-business-procedure-check`・`reference-change-assurance`・`reference-cli-trace-benchmark`・`reference-order-multilingual-operation`・`reference-shared-acceptance` | ok（80.3s を含む） |
| `tools/reference-mcp-readonly-adapter` | ok（33.9s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. 互換性

**意図的な公開契約の変更を含む。**

- `trace` の `subject-kind` が 10 → **13** に増えた。`commands`・`-h`・MCP adapter の enum が同時に広がる。
- `graphFingerprint` が変わる（ノードと辺が増えるため）。保存済みの継続トークンは失効する。
- `trace.unresolved` の意味は**変わっていない**。投影で落ちた辺は依然として数えない。今回は落ちる辺をほぼ無くしたので影響が小さいが、**この数え方自体は未修正**である。

## 8. 読み取れないこと（NOT RUN）

- `trace.unresolved` が投影の取りこぼしを数えられない点（#4068 の (b)）。今回は (a) を直して取りこぼしを1件まで減らしただけで、**数え方の欠陥は残る**。
- `inspect` は `database` 系を受け付けない（`InspectionSubjectKinds` は12種のまま）。`trace` と `inspect` で扱える種が異なる状態が続く。
- `persistence` の列対応を辺にすること（#4001）。端点は用意できたが、辺の設計は未着手。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

第49ラウンドで「`unresolved: 0` は正常ではなく『何も見ていない』だった」と書いた。本ラウンドで**見えるようにした**ので、`unresolved: 0` は今ほぼ正直な値になった。

ただし**数え方の欠陥は残した**。取りこぼしを減らしただけで、`TraceDirect` が `graph.Edges` しか走査しない構造は変わっていない。「穴を塞いだ」ではなく「穴を1つに絞った」と記録する。
