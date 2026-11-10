# `realize ... with custom go` が参照グラフの辺になっていなかった（修正）

- 記録日: 2026-11-10
- 対象: `internal/modelref/derivedbridge/projection.go`
- 関連: #3983、#3985（証拠記録）

## 観測（修正前）

`realize X with custom go` は「この capability はこのアプリケーションが実装する」という宣言ですが、`trace` に一切現れませんでした。

```text
tei trace -workspace reference/workspace -scope c \
  -subject-kind semantic-capability -subject-id security.authenticate -direction incoming -mode full
# total: 0
```

`reference/workspace` には4つの realize 宣言があり、索引（`index.Derived.CustomGoBindings`）はそれらを保持していましたが、投影が1度も参照していませんでした（`grep -c CustomGoBindings projection.go` が 0）。

## 変更

`FromIndex` で `d.CustomGoBindings` を辺にした。

```go
for _, b := range d.CustomGoBindings {
    if b.Application == "" || b.Capability == "" {
        p.Unresolved = append(p.Unresolved, "incomplete-realization:"+string(b.Capability))
        continue
    }
    addEdge("application", string(b.Application), "realizes", "semantic-capability", string(b.Capability))
}
```

**向きと語彙の判断**:

- 辺は `application --realizes--> semantic-capability`。宣言は「アプリケーションが能力を実装する」であり、能力を尋ねる読み手は `-direction incoming` で実装者を得る。
- 最初は `capability --realizes--> adapter`（adapter ファイル名をノードにする形）で実装したが、`unmaterialized-to:adapter:...` になった。adapter のファイル名とフック名は**生成されるファイルの名前**でありモデルの subject ではないため、ノードにしてはいけない。索引のメタデータとして残す。
- 実装者を問う読み手のために `application` を from にしたことで、既存の `application` ノードと `semantic-capability` ノードだけで完結する。

## 検証

- `gofmt -l internal`: 空、`go build ./...`: PASS
- `go test ./internal/modelref/... ./internal/operations ./tools/reference-cli-trace-benchmark ./tools/reference-mcp-readonly-adapter -count=1`: PASS
- `go test ./cmd/tei -count=1 -timeout 30m`: PASS
- 実測（実バイナリ、`reference/workspace`）:

| capability | incoming の辺 |
| --- | --- |
| `security.authenticate` | `order_entry --realizes-->` |
| `security.authorize` | `order_entry --realizes-->` |
| `sales.order.submit` | `order_entry --realizes-->` と `order_entry.submit_order --capability-->` |
| `order_entry.compose_order_summary` | **辺なし**（下記） |

- 追加試験 `internal/modelref/derivedbridge/realize_edge_test.go`:
  - 束縛が `application -realizes-> semantic-capability` の辺になり、規則も登録されること（規則が無いと `invalid-edge-type` になる）。
  - adapter のファイル名がノードにも辺の subject にもならないこと。
  - capability が宣言されていない束縛は `unmaterialized-to:semantic-capability:...` として報告されること。
  - application が無い束縛は `unmaterialized-from:application:...` として報告されること。
- 負例: `addEdge` の呼び出しを無効化すると `no realizes edge from {c {application order_entry}}` で失敗する（コードは戻した）。

## 見つけた別の欠陥（未修正・報告のみ）

`order_entry.compose_order_summary` は**どのファイルにも capability として宣言されていません**。束縛だけが存在します。修正後は `projectionUnresolved` に `unmaterialized-to:semantic-capability:order_entry.compose_order_summary` と出るので、**「実装だけあって宣言が無い」状態が観測できる**ようになりました。

これは `reference/workspace` の内容の問題であり、`reference-shared-acceptance` の受理経路が要るため、この変更では触っていません。

## 主張しないこと

- `reference/workspace` の内容（4つ目の realize の宣言漏れ）は直していない。
- adapter のファイル名・フック名をグラフへ出す設計は採っていない（生成物の名前であり subject ではない）。
- `custom go` 以外の実装種別（`HeaderDetailContractQueryBindings` 等）は辺にしていない。今回の Issue が指すのは `realize ... with custom go` だけである。
- 実 Host・認可・外部送出は NOT RUN。
