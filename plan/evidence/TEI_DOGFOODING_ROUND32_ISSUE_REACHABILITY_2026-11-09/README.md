# trace の行 issue 11種はすべて CLI から観測できない（2026-11-09、第32ラウンド）

- 対象: [#3906](https://github.com/hat47x/tei-platform/issues/3906)（第31ラウンドで起票）、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3「失敗」。
- 前段: [第31ラウンド](../TEI_DOGFOODING_ROUND31_TRACE_ISSUE_REACHABILITY_2026-11-09/README.md)。そこで「6種のうち4種は到達不能、`invalid-source` は未確認」と記録した。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **前ラウンドの「未確認」を閉じた。結論は「11種すべてが CLI から観測できない」。**

## 1. 目的

第31ラウンドで `invalid-source` を「到達させる方法が見つかっていない」として残した。これを閉じる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `f04c51049`（`origin/main`） |
| CLI実行物 | `sha256:3e17ceff6d402edb7a57f0910a3dc77859fa93513e7b97498313d2f2b14f4738` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 行の `issue` に入りうる値は 11 種

`internal/modelref/trace.go` の switch が6種、`internal/modelref/lookup.go` の `resolve` が5種を返す（後者は `dest, issue := lookup.resolve(edge.To)` でそのまま `row.Issue` になる）。

| 出所 | 値 |
| --- | --- |
| `trace.go` の switch | `invalid-source`・`ambiguous-source`・`invalid-target`・`invalid-edge-type`・`invalid-rule`・`conflicting-rule` |
| `lookup.resolve` | `invalid-reference`・`wrong-kind`・`missing-reference`・`ambiguous-reference`・`version-mismatch` |
| `evidenceDiagnostics` | 任意のコード（証拠の Authority が不正なとき） |

## 4. 到達不能である理由（それぞれ実測またはコード読解で確認）

### 4.1 `invalid-source` / `ambiguous-source`

`counts` は `graph.Nodes` の各 `Key` を1回ずつ数える。

```go
counts := make(map[Key]int, len(graph.Nodes))
for _, node := range graph.Nodes {
    counts[node.Key]++
}
```

`projection` は `nodes` マップで重複を弾くので、同じ `Key` は1つしか入らない。

```go
k := modelref.Key{Scope: scope, Subject: ...}
if nodes[k] {
    return // repeated use of the same Application is not a duplicate definition
}
```

したがって `counts[edge.From] > 1`（`ambiguous-source`）は起きない。`counts[edge.From] == 0`（`invalid-source`）も、`addEdge` が `!nodes[from]` を先に弾いて `unmaterialized-from` として `projectionUnresolved` に落とすため、行にならない。

さらに `TraceDirect` は**選択した subject 自体**の重複を `SUBJECT_AMBIGUOUS` で拒否する。

### 4.2 `invalid-edge-type` / `invalid-rule` / `conflicting-rule`

`derivedbridge.FromIndex` が**投影した辺そのものから規則を作る**。

```go
rule := modelref.Rule{Relation: relation, FromKind: fromKind, ToKind: toKind}
ruleSet[rule] = true
```

規則は辺から合成されるので、規則の欠落・空欄・重複が生じない。

### 4.3 `invalid-target`

`lookup.resolve` が空キーを返す条件は「Kind か ID が空白」「 Scope に前後空白」。投影は空の `id` を `add` で捨て、`scope` は `trace` の引数として検証済み（下記 4.5）。到達させるには不正な `Ref` を持つ辺が必要だが、投影はそれを作らない。

### 4.4 `missing-reference` / `wrong-kind` / `ambiguous-reference`

いずれも「辺の到達先がグラフに無い／種類が違う／同名が複数」で起きる。しかし投影は `!nodes[to]` を `unmaterialized-to` として `projectionUnresolved` へ落とし、`semantic-element` の解決は `semanticKindsByID` が1件のときだけ通す。

実測:

```tei
data a { relation r to app.no_such_data many }
```

```text
validate: UNKNOWN_SEMANTIC_RELATION_TARGET
trace:    ok:true total:0 unresolved:0
          projectionUnresolved: ["unmaterialized-to:semantic-data:app.no_such_data"]
```

行にならず、`projectionUnresolved` に出る。

### 4.5 `invalid-reference`（`lookup.resolve` 側）

`Ref.Scope` に前後空白がある場合。`trace` の `-scope` は引数検証で拒否される。

| 入力 | 結果 |
| --- | --- |
| `-scope " bad "` | exit 2 `INVALID_ARGUMENT` |
| `-scope $'a\nb'` | exit 2 `INVALID_SUBJECT` |

したがって投影に不正な scope は入らない。

### 4.6 `version-mismatch`

`Ref.Version` を使う経路が CLI の投影に無い（`modelref.Ref` の `Version` は投影で設定されない）。

### 4.7 `evidenceDiagnostics`

辺の `Evidence` の `Authority` が既定の集合外のとき。CLI の投影は `Evidence` を設定しない（`p.Graph.Edges = append(..., modelref.Edge{...})` に `Evidence` が無い）。

## 5. 実測: 実 workspace では1件も出ない

`reference/workspace` の全 subject を `trace -mode full` で走査した結果、`issue` は**1件も観測されなかった**。

```text
issues observed: none
```

## 6. 結論

**`trace` の行が持つ `issue` は、CLI 経路では 11 種すべてが観測不能。**

`Status` は常に `resolved` で、`issue` は常に空。未解決の実体は `result.projectionUnresolved` にのみ現れる。第31ラウンドで契約文書に書いた「未解決の判断は `projectionUnresolved` で行う」は、**この意味で正しかった**（`issue` を見ても何も分からない）。

## 7. 起票しない理由

既に [#3906](https://github.com/hat47x/tei-platform/issues/3906) で扱っており、今回はその「未確認」を閉じた追加実測である。重複Issueを作らない方針に従い、3906 に追記する。

## 8. 読み取れないこと（NOT RUN）

- `modelref` を直接使う他経路（`derivedbridge` を通さない呼出し）でこれらの `issue` が意味を持つか。`modelref` は汎用の型であり、テストが直接 `Graph` を組めば到達しうる。**CLI 経由で観測できない**という主張であって、コードが死んでいるという主張ではない。
- `evidenceDiagnostics` の実際のコード集合（証拠付きの辺を投影する将来の経路）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

第31ラウンドの「未確認」を閉じた。`trace` の `issue` は CLI からは常に空であり、未解決は `projectionUnresolved` にのみ現れる。**契約文書の記述が正しかったことを、独立した経路（11種すべての到達性）で裏付けた。**

なお `modelref` 自体は汎用の型なので、CLI 以外の呼出し元では `issue` が意味を持ちうる。今回の主張は「CLI 経由で観測できない」に限定している。
