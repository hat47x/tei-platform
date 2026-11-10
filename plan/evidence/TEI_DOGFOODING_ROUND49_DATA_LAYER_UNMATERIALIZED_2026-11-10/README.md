# データ層のkindがノードにならず、辺が全て落ちる（2026-11-10、第49ラウンド）

- 対象: [#4068](https://github.com/hat47x/tei-platform/issues/4068)（起票）、[#4001](https://github.com/hat47x/tei-platform/issues/4001)（前提を訂正）、[#3983](https://github.com/hat47x/tei-platform/issues/3983)、[#4024](https://github.com/hat47x/tei-platform/issues/4024)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **read-only の実測。新しい欠陥を検出して起票し、既存起票の前提を訂正した。コードは変更しない。**

## 1. 目的

前ラウンドで「公開前に現在の main で主張を再実測する」を手順化した。本ラウンドは #4001 の解消を試みたが、**着手前の再実測で自分の主張が過大だったことが分かった**。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `e613e23b8`（`origin/main`） |
| CLI実行物 | `/tmp/tei50`（`e613e23b8` からビルド） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 訂正: `tei world` は既に答えていた

#4001 で私は「CLI で答えられない」と書いた。**`trace` だけを試して一般化したのが誤りだった。**

```bash
tei world -workspace reference/workspace
```

```text
FOUND: sales.order.total -> sales_core.public.orders.total_amount
mappings=2 columns=11
```

`persistenceMappings` は列対応を含めて出力される。**「この属性はどの列か」は既に CLI で答えられる。**

`trace` が運ばないのは事実で、そこは訂正不要である。

```text
trace -scope s -subject-kind semantic-attribute -subject-id sales.order.total -direction incoming
total: 0
```

## 4. 根本原因の発見

### 4.1 `order_entry` の `targets-database` 辺が出ない

`order_entry` は `database sales { target sales_core }` を宣言し、索引も辺を宣言している。

```go
// internal/index/index.go:386
DeclaredDependency{FromKind: "application", From: a.ID,
	Relation: "targets-database", ToKind: "database", To: r.TargetDatabase, Alias: r.Alias}
```

しかし trace の応答は次のとおり。

```text
total: 14  unresolved: 0
relations: {owns-pattern: 5, realizes: 3, uses-validation: 3, uses-semantic: 3}
targets-database present? NO
```

`projectionUnresolved` に理由があった。

```text
unmaterialized-to:database:master_data
unmaterialized-to:database:sales_core
unmaterialized-to:database-contract:sales_core.order
unmaterialized-to:database-contract:master_data.customer
unmaterialized-to:database-contract-query:master_data.customer/find_by_id
unmaterialized-from:database:sales_core
...
```

### 4.2 全 subject での規模

72 subject（scope `c`、`-direction both`）を走査した。

```text
subjects traced: 72
sum of trace.unresolved across them: 0
distinct projectionUnresolved messages: 12

unmaterialized-from  database-contract        216
unmaterialized-to    database-contract        216
unmaterialized-from  database                 144
unmaterialized-to    database                 144
unmaterialized-to    database-contract-query   72
unmaterialized-to    semantic-capability       72
```

**12種の未解決参照が 72 subject すべてで報告され、それでも `trace.unresolved` の合計は 0。**

## 5. 原因（2つ）

### (a) kind が materialize されない

`SupportedSubjectKinds` に database 系が無い。`FromIndex` は端点が `nodes` に無ければ辺を追加せず `unmaterialized-*` を積む。索引は database ノードを1つも作らないため、database を端点にする辺は**必ず**落ちる。

### (b) `trace.unresolved` は投影の取りこぼしを数えない

`TraceDirect` は `graph.Edges` を走査して `Status == "unresolved"` を数える。**投影で落ちた辺は `graph.Edges` に入らないので数えられない。** `unresolved` は「辺はあるが解決が曖昧」だけを数え、「辺自体が存在しない」を数えない。

```go
for _, edge := range graph.Edges {
	row := TraceRow{Status: "unresolved", Edge: edge}
	...
	if row.Status == "unresolved" {
		unresolved++
	}
}
```

## 6. 影響

- **データ層が参照グラフに一切現れない。** 「この application はどの DB を使うか」「この contract は何を読むか」を `trace` で答えられない。
- **`unresolved: 0` を「すべて解決済み」と読むと誤る。** 私は第41〜47ラウンドで `unresolved: 0` を見て正常と判断し続けた。実際には12種の未解決が常にあった。**この表示が #4001 の根本原因を隠していた。**
- 第46ラウンドの `entryFieldCoverage` は「索引が World を読むか」「entry 型が欄を捨てたか」の2段だけを見ており、**3段目（投影が端点を作れるか）を見ていない**。

## 7. 起票した案

| 案 | 内容 | 判断が要る点 |
| --- | --- | --- |
| A | database 系を materialize し `trace` の subject-kind にも加える | `tools/reference-mcp-readonly-adapter/subject_kind_enum_test.go` が `len(TraceSubjectKinds) != 10` を固定しており、外部ツールも同時に直す必要がある |
| **B** | ノードは作るが subject-kind には加えない | 契約を変えずに `targets-database` 辺は見える |
| C | 索引が database 系の辺を宣言するのをやめる | データ層は永久に見えない |
| **D** | `trace.unresolved` に投影の取りこぼしを含める | 小さく、誤読は防げる |

**B + D を推した。** 契約を変えずにデータ層の辺が可視化され、誤読も防げる。**C は採るべきでない** — 見えないという設計判断なら `AuthorityLimits` に書くべきで、索引が黙って辺を宣言して落とす現状は誰にも利益がない。

## 8. 読み取れないこと（NOT RUN）

- 案 A/B/C/D の採否。Owner の設計判断。
- `unmaterialized-to:semantic-capability:order_entry.compose_order_summary`（第42ラウンドで見つけた、存在しない capability への束縛）の扱い。
- 3段目の検査を機械化できるか（#4068 の設計が決まってから）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**「解消しようとしたら、着手前の再実測で自分の起票が過大だったと分かった」回**である。#4001 の「CLI で答えられない」は誤りで、`tei world` は答えていた。`trace` に限れば正しかったので、Issue は依存関係を明示して残した。

同時に、**私が7ラウンド見落としていた `unresolved: 0` の意味**を特定した。数えている対象が「曖昧な辺」だけで、「存在しない辺」を数えていなかった。この表示は「正常」ではなく「何も見ていない」だった。
