# データ層の subject がノードにならず、その辺が全て落ちていた（修正）

- 記録日: 2026-11-11
- 対象: `internal/index/index.go`、`internal/modelref/derivedbridge/projection.go`
- 関連: #4068、#4001（本件が根本原因）、#3983・#4024（同じ型）、#3910（`trace.unresolved` の意味）

## 観測（修正前、`origin/main` `d160581ca`）

`database`・`database-contract`・`database-contract-query` がノードにならないため、それらを端点にする辺が全て落ちていました。

```text
tei trace -scope c -subject-kind application -subject-id order_entry -direction outgoing -mode full
# total: 14  unresolved: 0
# relations: {owns-pattern:5, realizes:3, uses-validation:3, uses-semantic:3}
# targets-database: NO  requires-contract: NO
```

`order_entry` は `database sales { target sales_core }` を宣言しているのに、`targets-database` が出ません。

## 原因

投影は辺の両端がノードであるときだけ辺を残します。索引は database 系の subject を1つも記録していなかったため、**必ず落ちていました**。落ちた辺は `projectionUnresolved` に出るだけで、`graph.Edges` に入らないので `trace.unresolved` には数えられません。

## 変更（#4068 の案 B）

- `index.Derived` に `DatabaseElements []DatabaseElementIndexEntry`（`Kind`・`ID`）を追加し、`w.Databases` から次を記録する。
  - `database` = `db.ID`
  - `database-contract` = `c.ID`
  - `database-contract-query` = `c.ID + "/" + q.Name`（workspace の命名に合わせる。`internal/workspace/loader.go` と同じ形式）
- 投影はこれをノードとして materialize する。
- **`subject-kind` の公開列挙は変えない**（案 B）。ノードは辺の端点として存在するが、`trace -subject-kind database` は引き続きできない。公開文法の拡張は契約変更なので、別の判断とする。
- `DatabaseElements` は決定的にソートする（グラフ指紋が map 順に依存しないように）。

## 検証

- `gofmt -l internal`: 空、`go build ./...`: PASS
- `go test ./internal/index ./internal/modelref/... ./internal/operations ./tools/reference-mcp-readonly-adapter ./tools/reference-cli-trace-benchmark -count=1`: PASS
  - `tools/reference-mcp-readonly-adapter` の `subject_kind_enum_test.go` が `len(TraceSubjectKinds) != 10` を固定しているが、**列挙を変えていないので通る**（案 B の確認）。
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 前後比較（同じ `reference/workspace`、`main` のバイナリと修正後のバイナリで実測）:

| | 修正前（main） | 修正後 |
| --- | ---: | ---: |
| `order_entry` outgoing の行数 | 14 | **19** |
| `targets-database` | 出ない | **2** |
| `requires-contract` | 出ない | **3** |
| `projectionUnresolved` の件数 | 12 | **1** |
| 72 subject 走査の `projectionUnresolved` 合計 | **864** | **72** |
| 未解決の種類 | **6種**（`unmaterialized-from/to: database`、`database-contract`、`database-contract-query`、`semantic-capability`） | **1種**（`semantic-capability`） |
| `sum(trace.unresolved)` | 0 | 0 |

- 残る1種は #3983 で確認した**実在する宣言漏れ**（`order_entry.compose_order_summary` が capability として宣言されていない）で、72件は「subject ごとに1回報告される」ことによるもの。グラフは正しくなり、報告は真の陽性だけになった。
- 追加試験:
  - `internal/index/database_elements_test.go`: 3種の ID が記録されること、空のクエリ名を捨てること、決定的にソートされること。
  - `internal/modelref/derivedbridge/data_layer_nodes_test.go`: `targets-database`・`requires-contract`・`contract-query` の辺が残ること、**`SupportsSubjectKind` が database 系を広告しないこと**、`DatabaseElements` を空にすると `unmaterialized-to:database:...` が報告されること。
- 負例: `Kind: "database"` の記録を消すと `database = "", want "sales_core"` で失敗する（コードは戻した）。

## `trace.unresolved` の意味は変えない（案 D を採らない理由）

#4068 の (b) は `trace.unresolved` に投影の取りこぼしを数えさせる案（D）ですが、採っていません。

- #3910 で `trace.unresolved` は「行はあるが解決が曖昧な件数」、`projectionUnresolved` は「グラフ全体の取りこぼし」と定め、`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` に記録済み。ここを変えると `tei.cli.trace/v0` の意味が変わる。
- (b) が問題になったのは「12種の未解決が常にあったのに `unresolved: 0` だった」ため。**(a) を直した結果、未解決は真の陽性1種だけになり、誤読の実害は消えた**。
- 表示の改善がなお必要なら、`trace.unresolved` の意味を変えるのではなく、投影の取りこぼし件数を別の欄として足すのが筋。今回はその必要が生じていないので、契約を触らない。

## 主張しないこと

- `trace -subject-kind database` はできないまま（案 B）。マッピングや database を subject として引く需要は別の判断。
- `reference/workspace` の `compose_order_summary` の宣言漏れは直していない（`reference-shared-acceptance` の受理経路が要る）。
- `RelationStorage`・`ForeignKeys` 等、索引が持っていないデータ層の詳細は辺にしていない。
- 実 Host・認可・外部送出は NOT RUN。
