# `trace` の「未解決」2欄の意味が読めず、`unresolved: 0` を「未解決なし」と誤読できた（契約へ明記）

- 記録日: 2026-11-10
- 対象: `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md`、`internal/operations/trace_unresolved_meaning_test.go`
- 関連: #3906、#3420

## 観測（修正前）

`trace` の応答には「未解決」を表す欄が2つあり、名前が似ているため取り違えられます。

| 欄 | 読める意味（修正前の文書） |
| --- | --- |
| `result.trace.unresolved` | 未解決の件数 |
| `result.projectionUnresolved` | 投影で落ちた参照 |

`reference/workspace` の全 subject を実測した結果は次のとおりでした。

```text
subjects in catalogue: 72
subjects traced: 66   (application-pattern は trace の対象外)
trace.unresolved > 0: 0
result.projectionUnresolved non-empty: 66
rows carrying an issue: 0  (issue の内訳: 空)
```

つまり **`trace.unresolved` は常に0** で、`projectionUnresolved` は**全66件で非空**でした。

さらに重要な点を2つ測りました。

1. `projectionUnresolved` は **66 subject すべてで同一の17件**（distinct payloads = 1）。`-scope` を `s` と `other-scope` に変えても同じ17件。**グラフ全体の一覧であり、要求した subject に限定されない。**
2. 17件は**要求した subject を名指していない**（`sales.order.customer` を trace しても `order.customer` を含まない）。

## 原因（コードで確認）

`rows[].issue` に入りうる値は6つですが、CLI 経路では大半が構造的に発生しません。

- `internal/modelref/derivedbridge/projection.go:99` が **辺から規則を作る**（`rule := modelref.Rule{Relation: relation, FromKind: fromKind, ToKind: toKind}`）。したがって `invalid-edge-type`（規則が無い）・`invalid-rule`（欄が空）・`conflicting-rule`（規則の重複）は起こりません。
- 同名要素の二重宣言は `validate` が `DUPLICATE_SEMANTIC_ELEMENT` で先に拒否するため、`ambiguous-source` も trace まで届きません。
- 未解決の outcome payload は `unmaterialized-to` として `projectionUnresolved` に落ちるため、`invalid-target` の行になりません。

## 変更（案 C: 契約への明記）

実装の意味は変えず、契約文書 `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` §4 に次を明記しました。

- `result.trace.unresolved` は**返した行のうち未解決だった行の件数**（`rows[].issue` が入っている行を数える）。
- `result.projectionUnresolved` は**グラフ全体の投影で落ちた参照の一覧**で、要求した subject に限定されない（実測: 66 subject すべてが同一の17件、`-scope` を変えても同じ）。
- **`trace.unresolved: 0` は「Workspace に未解決が無い」を意味しない。** `projectionUnresolved` が空でない場合、未解決は存在するが、その subject の行としては現れていない。両方を読むこと。
- `rows[].issue` の6値のうち、CLI 経路で到達できるのは一部だけ。到達不能な値を「起こりうる」と読まないこと。

## 検証

- `gofmt -l internal`: 空、`go vet ./internal/operations`: PASS、`go build ./...`: PASS
- `go test ./cmd/tei -run 'TestContractDocument' -count=1`: PASS
- 追加試験 `internal/operations/trace_unresolved_meaning_test.go`:
  - **グラフ全体性**: 3 subject（data・relation・type）で `projectionUnresolved` が同一であること。
  - **行の件数であること**: `trace.unresolved` が `issue` を持つ行の数と一致し、`4 rows / unresolved=0 / projectionUnresolved=17` の状態を1つの応答から観測できること（試験ログに実測値を出す）。
  - **到達不能性**: 行が `invalid-edge-type`・`invalid-rule`・`conflicting-rule` を持たないこと、宣言外の値も持たないこと。
- 実測（実バイナリ）: `reference/workspace` の72 subject を trace し、上表の数値を確認。

## 主張しないこと

- 案 A（欄の改名）と案 B（`projectionUnresolved` を `trace` の内側へ統合）は**実施していない**。応答形の変更であり、`tei.cli.trace/v0` の扱いを決める必要がある。
- `projectionUnresolved` を subject 単位に絞っていない（挙動は不変）。
- 到達不能な `issue` 値を削除していない（`modelref` の宣言はそのまま）。
- 実 Host・認可・外部送出は NOT RUN。
