# realize の実装束縛を参照グラフの辺にした（2026-11-10、第42ラウンド）

- 対象: [#3983](https://github.com/hat47x/tei-platform/issues/3983)（第41ラウンドで起票）、[#3884](https://github.com/hat47x/tei-platform/issues/3884)。
- 前段: [第41ラウンド](../TEI_DOGFOODING_ROUND41_REALIZE_NOT_IN_GRAPH_2026-11-10/README.md)（発見と起票）。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **前ラウンドで起票した欠陥を修正。**

## 1. 目的

前ラウンドで「`realize` による実装束縛が `trace` に現れない」ことを記録した。本ラウンドで、案 A（`DeclaredDependencies` に辺を追加）を実装する。

## 2. 実装前の確認（読んでから書く）

前3ラウンドで「読まずに書く」失敗を繰り返したので、次を先に確かめた。

| 確認 | 結果 |
| --- | --- |
| 誰かが着手済みでないか | `gh pr list` に該当なし、`git log` に #3983 参照なし |
| `BindingIndexEntry` の形 | `Application`・`Capability`・`Adapter`・`Hook` |
| `DeclaredDependency` の形 | `FromKind`・`From`・`Relation`・`ToKind`・`To`・`Alias` |
| 既存の関係名 | `attribute`・`capability`・`input`・`outcome`・`owns-pattern` など（kebab-case、動詞） |
| **`Application` は常に入るか** | パーサは `CustomGoBinding{Capability: id}` しか返さないが、loader が `b.Application = a.ID` を設定する（`loader.go:747`） |
| **`application` はノードになるか** | なる。ただし `ApplicationPatterns` 経由でのみ `add("application", ...)` される |
| 対象 application は実在するか | `order_entry`・`repair_intake` がノードとして存在 |

**「`Application` が空なら辺を作らない」条件が必要**だと分かった。ambiguous な空ノードを指す辺を作ると、投影が `unmaterialized-from` として報告してしまう。

## 3. 修正

`internal/index/index.go` の束縛走査に1行の依存を足した。

```go
for _, b := range w.Bindings {
	d.CustomGoBindings = append(d.CustomGoBindings, BindingIndexEntry{...})
	if b.Application != "" && b.Capability != "" {
		d.DeclaredDependencies = append(d.DeclaredDependencies, DeclaredDependency{
			FromKind: "application", From: b.Application,
			Relation: "realizes",
			ToKind:   "semantic-capability", To: b.Capability,
		})
	}
}
```

関係名は `realizes`（DSL の `realize` に対応する動詞、既存の kebab-case 慣習に合わせた）。

## 4. 修正後

### 4.1 3つの束縛が辺になった

```bash
tei trace -scope c -subject-kind semantic-capability \
  -subject-id security.authenticate -direction incoming -mode full
```

```text
ok: True  total: 1  unresolved: 0
   application order_entry - realizes
```

| capability | realizes 辺 |
| --- | --- |
| `security.authenticate` | 1（`order_entry`） |
| `security.authorize` | 1（`order_entry`） |
| `sales.order.submit` | 1（`order_entry`） |
| `order_entry.compose_order_summary` | **0** |

### 4.2 4つ目が0なのは正しい

`order_entry.compose_order_summary` は**どこにも宣言されていない**（`grep` しても `default.tei` の `realize` 行だけ）。つまり束縛の対象が存在しない。投影はそれを正しく報告する。

```text
projectionUnresolved: ["unmaterialized-to:semantic-capability:order_entry.compose_order_summary"]
```

**これは修正前から存在するワークスペース側の不整合**で、今回の変更が作ったものではない。修正前は束縛がグラフに入っていなかったため、この不整合が**見えなかった**。今回の変更で可視化された。

`application order_entry` の outgoing は14件で、うち `realizes` が3件、`owns-pattern` が5件、`uses-semantic` が3件、`uses-validation` が3件である。

## 5. 試験

`internal/index/realizes_test.go` の3件。

| 試験 | 内容 |
| --- | --- |
| `TestCustomGoBindingBecomesARealizesEdge` | `application order_entry -realizes-> semantic-capability security.authenticate` が存在する |
| `TestCustomGoBindingRemainsIndexed` | 辺を足しても `CustomGoBindings` の記録（`Application`・`Capability`・`Adapter`・`Hook`）が変わらない |
| `TestBindingWithAnEmptySideProducesNoEdge` | `Application` が空の束縛は辺を作らない |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 辺の追加を削除（元の欠陥） | FAIL `no realizes edge from order_entry` |
| M2 | 空側ガードを `if true` に | FAIL `a binding with no application produced "" -> "security.authenticate"` |

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/index ./internal/modelref/... ./internal/operations` | すべて ok |
| `./cmd/tei`（契約・golden を含む） | ok（170.8s） |
| `go vet ./internal/index` | 指摘なし |
| `gofmt -l .` | 空 |

`reference/standard-crud/workspace` ほかの他ワークスペースも、私の変更前後で同じ応答だった（引数の渡し方を誤っていたので、両バイナリで同一の `INVALID_ARGUMENT` を確認した）。

## 7. 互換性

**意図的な挙動変更を含む。**

- `trace` の `graphFingerprint` が変わる（辺が増えるため）。保存済みの継続トークンは失効する。
- `projectionUnresolved` に `unmaterialized-to:semantic-capability:order_entry.compose_order_summary` が新たに現れる。これは**既存のワークスペース側の不整合が可視化されたもの**で、修正が持ち込んだ欠陥ではない。

## 8. 読み取れないこと（NOT RUN）

- `order_entry.compose_order_summary` の束縛が正しいか（対象の宣言漏れか、束縛の綴り誤りか、廃止予定か）。**ワークスペースの Owner の判断**であり、私は触っていない。
- `realize` 以外の3種の束縛（`HeaderDetailContractQueryBinding`・`ContractQueryExecutorBinding`・`ContractQuerySQLRealization`）も同様に辺にすべきか。今回は `Bindings` のみ。
- `application` ノードが `ApplicationPatterns` 経由でしか作られない点。pattern を持たない application の束縛は `unmaterialized-from` になる。今回は該当なしだが、将来の穴になりうる。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

第41ラウンドで見つけた「索引にあるがグラフに無い」欠陥を、写像1箇所の追加で閉じた。ADR-0048 の Binding/Realization 層が `trace` で辿れるようになり、「この capability は何で実装されているか」に CLI で答えられる。

副産物として、**ワークスペース側の既存不整合（存在しない capability への束縛）が可視化された**。修正前は束縛がグラフに入っていなかったため、これを見る手段が無かった。これは「見えないこと」自体が欠陥だったことの証拠でもある。
