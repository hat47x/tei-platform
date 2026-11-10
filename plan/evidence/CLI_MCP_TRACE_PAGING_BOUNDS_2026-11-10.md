# `trace` の `offset`／`limit` だけが値域を宣言せず、負値の拒否を client が読めなかった（修正）

- 記録日: 2026-11-10
- 対象: `tools/reference-mcp-readonly-adapter/main.go`
- 関連: #3420（CAPI-2）、#3906（`requestedOffset`）、#3851 / #3928（`inputSchema` の宣言と実行の一致）

## 観測（修正前）

`tei_trace` の `offset`・`limit` は `{"type":"integer"}` だけで、**値域が宣言されていませんでした**。同じ Adapter の `tei_model_locate` の `limit` は `{"minimum":1,"maximum":500}` を宣言しており、非対称でした。

| tool | 宣言（修正前） |
| --- | --- |
| `tei_trace` | `offset`・`limit` とも `{"type":"integer"}` |
| `tei_model_locate` | `limit` は `{"minimum":1,"maximum":500}` |

実測した挙動は次のとおりです。

| 入力 | 結果 |
| --- | --- |
| `tei_trace` `limit=-1` | 文書の `error.code: INVALID_ARGUMENT`（`isError:true`） |
| `tei_trace` `limit=0` | 成功（0 は「未指定」の意味） |
| `tei_trace` `offset=-1` | 文書の `INVALID_ARGUMENT` |
| `tei_model_locate` `limit=0` | **プロトコル `-32602`**（`limit must be between 1 and 500`） |
| `tei_model_locate` `limit` が文字列 | **プロトコル `-32602`** |

つまり `trace` は負値を「意味上の失敗」として文書で返し、`model locate` は「入力の誤り」としてプロトコルで返します（Adapter README の「入力の検査」に書かれた区別どおり）。しかし `trace` 側の値域が宣言されていないため、client は 0 が有効で負値が拒否されることを `inputSchema` から読めませんでした。

## 変更

- `tei_trace` の `offset`・`limit` に `"minimum": 0` を宣言した（0 は「未指定」を意味し受理される）。
- 上限は付けていない。`trace` に上限が無いため、無い制約を宣言しない。
- `tei_model_locate` の `minimum:1`・`maximum:500` はそのまま。**2つの `limit` は意図的に異なる**ので、それぞれが自分の値域を宣言する形にした。

## 検証

- `gofmt -l tools`: 空、`go build ./...`: PASS
- `go test ./tools/reference-mcp-readonly-adapter -count=1`: PASS（132 秒）
- 追加試験 `tools/reference-mcp-readonly-adapter/trace_paging_bounds_test.go`:
  - `offset`・`limit` に `minimum: 0` があること。
  - **2つの `limit` が異なることを固定**（`trace` は 0、`model locate` は 1 と上限あり）。
  - **宣言した最小値が実際の境界と一致すること**: 最小値（0）は受理され、その1つ下（-1）は拒否される。`dispatch` を通した実挙動で確認する。
- 実測（実バイナリ）: `trace` の宣言は `{"minimum":0}`、`model locate` は `{"minimum":1,"maximum":500}` のまま。`limit=-1`・`offset=-1` は拒否、`=0` は成功。**全ケースで宣言と挙動が一致**（`all consistent: True`）。
- 負例: `offset` の `minimum` を外すと `TestTracePagingBoundsAreDeclared` が「no minimum」で失敗する（コードは戻した）。

## 主張しないこと

- **`model locate` の「プロトコル `-32602`」と `trace` の「文書 `INVALID_ARGUMENT`」の使い分けは変えていない。** 数値の範囲は Adapter が入力として検査し、意味上の拒否は操作の結果として返す、という既存の区別をそのまま保った。この非対称が望ましいかは未決で、変えるなら両入口の契約判断になる。
- `trace` の `limit` に上限を設けていない（操作に上限が無い）。
- 実 Host・認可・外部送出は NOT RUN。
