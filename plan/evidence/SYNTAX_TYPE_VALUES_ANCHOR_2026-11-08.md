# 値域付きの型宣言が Source Index と validate を丸ごと失敗させる（記録）

- 記録日: 2026-11-08
- 対象: `internal/syntax/semantic_anchor.go`（anchor 走査）
- 関連: #3345 の作業中に発見。`cmd/tei/product_slice_test.go` の合成 Workspace が該当する宣言を持つため、`TestCLIProcessRestartRefusesObservedCommandReplay` が失敗していた。

## 観測

`requirements/TEI_PRODUCT_REQUIREMENTS.md` の `type ... values { ... }` は parser が読む（`internal/syntax/parser.go` の `parseSemanticType` が `values` 節を消費する）。一方、Source Index の anchor 走査（`semanticSubjectAnchors`）は `type <sym> from <underlying>` までを読み、`values` を見落とす。

```
namespace sales

type Status from std.code values { "open" "closed" }

data Order { ... }
```

この Workspace では `tei source-index` と `tei validate` が終了コード2で失敗する。

```text
source semantic-data:sales.order in semantic/sales/order.tei: unexpected semantic declaration "values" at 3:27
```

`tei world` は成功する（parser 経由のため）。つまり、意味モデルとしては読めているのに、入力元索引と検証だけが落ちる。

## 原因

`semanticSubjectAnchors` の `case "type"` が、`from` と下位型の識別子を読んだ後に `values` 節を読み飛ばさないため、次のループ反復が `values` を宣言名として解釈し、`default` 分岐で失敗する。`internal/syntax/parser.go` の `parseSemanticType` は、`peek` で `values` を見たときだけ `{ "..." }` を消費する。

## 変更（このPR）

- `skipSemanticTypeValues` を追加し、`case "type"` の末尾で呼ぶ。`values` が続くときだけ `{` と文字列列を `}` まで消費する。続かないときは何もしない。
- `internal/syntax/semantic_anchor_values_test.go` を追加。値域付きの型を anchor 走査が通り、値域なしの型も通り、閉じない `values {` は失敗することを固定する。

## 検証

- `gofmt -l internal/syntax`: 空、`go vet ./internal/syntax ./cmd/tei ./internal/operations`: PASS、`go build ./...`: PASS
- `go test ./internal/syntax -count=1`: PASS
- `go test ./cmd/tei -run 'TestCLIProcessRestart|TestCLILifecycleState|TestCLIProcessJSONContract|TestContractDocument' -count=1`: PASS
- 実測: 値域付き型を含む合成 Workspace で `tei source-index -workspace <dir>` が終了コード0・`ok:true`（修正前は終了コード2）。

## 主張しないこと

- この修正は anchor 走査の読み飛ばしだけである。`values` の値集合を検証（`validation`）が意味的に検査しているかは確認していない。
- 実データベース・実 Provider・受入済み業務意味の確認は行っていない。
