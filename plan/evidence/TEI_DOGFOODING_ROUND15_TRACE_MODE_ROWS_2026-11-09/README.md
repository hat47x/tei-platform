# trace -mode full が参照0件のとき rows を出さない（2026-11-09、第15ラウンド）

- 対象: [#3693](https://github.com/hat47x/tei-platform/issues/3693)、[#3456](https://github.com/hat47x/tei-platform/issues/3456)（同じ種類の先行修正）、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3・§5 T01/T07。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の比較実測と、1つの型のJSON表現の修正**。

## 1. 目的

第14ラウンドまでで `knowledge-*` の連鎖を通した。このラウンドは、その連鎖の手前にある `trace` の応答が、関係0件の Workspace でも機械的に読めるかを確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source（比較の基準） | `cd45e05edf4a8d2fec998580b8eed3a9ae57c80f`（`origin/main`） |
| 修正前の実行物 | `sha256:49c0ef003b8799df66adfeb90a56f0505a57edf0af9c9f81e8d37c2f5bf9243d` |
| 修正後の実行物 | `sha256:45a6a0ced055eded5365fc6124e361f30225d8ab338f0eb972618b4a6a82c7d5` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

対象Workspaceは2つ。関係を持たない最小Workspace（`/home/hat47x/dogfood/r15-ws`）と、関係を持つ `reference/workspace`。

```tei
namespace sales

data order {
    attribute orderId type std.text
    attribute customerId type std.text
    attribute note type std.text
}

data customer {
    attribute customerId type std.text
}

data product {
    attribute productId type std.text
}
```

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt` | 対象 commit |
| `30-validate.json` | 最小Workspaceの validate（`ok:true`、宣言9件） |
| `31-before.json` | 修正前: 関係0件の `-mode full` |
| `33-summary.json` | 修正前: 同じWorkspaceの `-mode summary` |
| `40-fixed-binary.txt` | 修正後の実行物 |
| `41-full-after.json` | 修正後: 関係0件の `-mode full` |
| `42-summary-after.json` | 修正後: `-mode summary`（rows は出ない） |
| `43-ref-after.json` | 修正後: 関係ありの `-mode full`（rows 2件、`nextOffset`、`continuation`） |
| `44-beyond-after.json` | 修正後: `-offset 99`（rows は `[]`） |
| `45-mutations.txt` | 修正をわざと壊した3種の検出 |
| `46-packages.txt` | 影響パッケージの試験 |
| `47-gofmt.txt` | 0 byte |
| `01-inspect-missing-id.json`・`02-inspect-empty-id.json`・`03-inspect-absent.json` | 必須引数欠落の分類（副次的な観測） |

## 4. 観測（修正前）

関係0件のWorkspaceで `-mode full` を実行すると、`result.trace` に `rows` が**存在しない**。

```json
{"ok":true,"result":{"trace":{"mode":"full","total":0,"unresolved":0,
  "graphFingerprint":"sha256:0c717ba1…","snapshotFingerprint":"sha256:0c717ba1…"}}}
```

`mode=summary` も `rows` を持たない。両者は `mode` と `total` で区別できるが、**`rows` という欄の有無は同じ**である。

原因は `internal/modelref/trace.go` の `Rows` に付いた `json:"rows,omitempty"` タグ。`omitempty` は nil だけでなく**長さ0のスライスも落とす**ため、`mode=full` の「0件という答え」と「欄が無い」が同じ表現になる。

`mode=summary` が `rows` を出さないのは契約どおりである。summary は全体を数えるだけで行を確保しない（`internal/modelref/trace.go` の `if options.Mode == "summary" { continue }`）。

## 5. 修正

`TraceResult` に `MarshalJSON` を足し、**`mode=full` かつ行が0件のときだけ** `rows` を `[]` として書く。それ以外は既定の変換にそのまま委ねる。

```go
func (r TraceResult) MarshalJSON() ([]byte, error) {
	type alias TraceResult
	plain := alias(r)
	if r.Mode != "full" || len(r.Rows) > 0 {
		return json.Marshal(plain)
	}
	type withRows struct {
		alias
		Rows *[]TraceRow `json:"rows"`
	}
	empty := []TraceRow{}
	return json.Marshal(withRows{alias: plain, Rows: &empty})
}
```

**行が1件以上ある場合は上書きしない。** 最初は `mode=full` のすべてを上書きする実装にしたが、埋め込み構造体が `rows` を最後に回すため、既存の full 応答のバイト列が変わった（`tools/reference-cli-trace-benchmark` の golden が `small/full-limit1` と `medium/full-limit1` で FAIL）。ゼロ件のときだけ上書きする形に絞り、既存の応答は1バイトも変えていない。

`omitempty` は外していない。外すと `mode=summary` にも `rows: null` が出て、「行を確保していない」ことが「0件」に見える。

## 6. 修正後の実測

| 入力 | `rows` | 備考 |
| --- | --- | --- |
| 関係0件・`mode=full` | **`[]` が出る** | 今回直した箇所 |
| 関係0件・`mode=summary` | 出ない | 契約どおり |
| 関係あり・`mode=full -limit 2` | 2件 | 変更なし。`nextOffset: 2`、`continuation` も従来どおり |
| 関係あり・`mode=full -offset 99 -limit 2` | **`[]`** | 最終ページを超えた場合も空配列 |

## 7. 試験

`internal/modelref/trace_rows_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestFullTraceAlwaysEmitsRows` | nil／空スライス／1件のいずれでも `rows` が存在し、`null` でないこと |
| `TestSummaryTraceOmitsRows` | summary は `rows` を出さないこと |
| `TestMarshalTraceKeepsEveryOtherField` | 他の全欄が既定の変換と同一であること（欄の消失・改名を検出） |

わざと壊した3種はいずれも検出された（`45-mutations.txt`）。

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `MarshalJSON` を無効化 | FAIL: `rows is absent for a full trace` |
| M2 | summary でも上書きする | FAIL: `summary emitted rows: []` |
| M3 | nil を `[]` にしない | FAIL: `rows was written as null` |

## 8. 影響範囲の試験

`./internal/modelref`・`./internal/operations`・`./cmd/tei`・`./tools/reference-mcp-readonly-adapter`・`./tools/reference-cli-trace-benchmark`・`./tools/repository-plan-check` がすべて ok（`46-packages.txt`）。gofmt 1.26.10 の指摘は0件。

## 9. 読み取れないこと（NOT RUN）

- HTTP・Desktop の入口。
- 実DB・実Provider・実Host、意味の受理・権限・配備。
- 関係0件の Workspace で `trace` 以外のコマンドが同じ種類の欄落ちをしないことの全面確認（`source-index` は #3456 で修正済み、他は未確認）。

## 10. 位置付け

「`mode=full` は全行を返す」という契約に対し、0件のときだけ欄が消えていた。空配列を返すようにして、0件と欄の不在を区別できるようにした。既存の full 応答のバイト列は変えていない。
