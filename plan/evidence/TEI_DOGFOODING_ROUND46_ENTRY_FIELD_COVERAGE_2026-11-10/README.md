# 索引の entry 型が World の下位構造を捨てていないかを検査で固定した（2026-11-10、第46ラウンド）

- 対象: [#4001](https://github.com/hat47x/tei-platform/issues/4001)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)。
- 前段: [第45ラウンド](../TEI_DOGFOODING_ROUND45_WORLD_FIELD_COVERAGE_2026-11-10/README.md)（`World` の欄を読むかの検査。残り半分として明記した段）。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **前ラウンドで「半分だけ塞いだ」と書いた残りを塞いだ。**

## 1. 目的

第45ラウンドで次の連鎖の1段目だけを検査した。

```text
ir.World --[検査済み]--> index.Build --[未検査]--> 参照グラフ
```

2段目、つまり **索引の entry 型が World の下位構造を捨てていないか** を機械的に検出する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `cdc22174d`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 実測: `PersistenceIndexEntry` は10欄のうち6欄を捨てている

| 型 | 欄 |
| --- | --- |
| `persistence.Mapping`（World 側） | `ID`・`Database`・`SemanticData`・`Table`・`Columns`・`RelationStorage`・`PrimaryKey`・`UniqueConstraints`・`Indexes`・`ForeignKeys`（10） |
| `PersistenceIndexEntry`（索引側） | `Mapping`・`Database`・`SemanticData`・`Table`（4） |

**6欄が捨てられている。** そのうち `Columns` が #4001 の直接原因（列対応がグラフに出ない）。`ID` は `Mapping` に改名されて保持されている。

対照として `realization.CustomGoBinding` → `BindingIndexEntry` は2欄とも保持し、`Adapter`・`Hook` を加えている。**捨てていない。**

## 4. 追加した試験

`internal/index/entry_field_coverage_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestEntryTypesKeepOrExplainEverySourceField` | 各 source 型の全欄が entry 型に存在するか、`renamed` か `dropped` に記録されているか。どれでもなければ落ちる。逆方向（存在しない欄を記録）も落ちる。`renamed` の移動先が実在するかも見る |
| `TestEntryTypesArePresent` | 型の改名で表が空振りしないよう、両側に欄があることを確認 |

```go
{
	source:  reflect.TypeOf(persistence.Mapping{}),
	entry:   reflect.TypeOf(PersistenceIndexEntry{}),
	renamed: map[string]string{"ID": "Mapping"},
	dropped: map[string]string{
		"Columns":           "column mappings; #4001 records that these do not reach the graph",
		...
	},
}
```

**捨てている欄には理由を必須**にした。第45ラウンドと同じ方針（「無い」を失敗にせず「判断を書け」を要求する）を、下位構造にも適用している。

### 4.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `Mapping` に未登録の欄を追加 | FAIL `Mapping.UndeclaredField neither appears in PersistenceIndexEntry nor is listed as dropped` |
| M2 | `Columns` の dropped 行を削除 | FAIL `Mapping.Columns neither appears in PersistenceIndexEntry nor is listed as dropped` |
| M3 | `ID` の rename 行を削除 | FAIL `Mapping.ID neither appears in PersistenceIndexEntry nor is listed as dropped` |

## 5. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/index ./internal/modelref/... ./internal/operations` | すべて ok |
| `go vet ./internal/index` | 指摘なし |
| `gofmt -l .` | 空 |

## 6. これで2段とも塞がった。ただし #4001 の欠陥自体は残る

```text
ir.World --[第45ラウンド]--> index.Build --[第46ラウンド]--> entry 型
```

**両方の段が「判断を書かなければ通らない」になった。** 新しい World 欄も、entry 型が捨てる下位構造も、理由なしには通らない。

**しかし #4001 の欠陥（列対応が辺にならない）は直っていない。** 今は `dropped` に理由付きで記録されているだけで、`trace` は依然として列対応を返さない。試験は「穴があること」を可視化・固定したに過ぎない。

**穴を塞ぐには、辺の両端を何にするかの設計判断が要る。** `database` は `SupportedSubjectKinds` に無いため、`persistence -stores-> semantic-data` を作るには kind の追加も伴う。私は材料を提示するに留める。

## 7. 読み取れないこと（NOT RUN）

- `#4001` の辺の設計（`persistence` を subject-kind に加えるか、`database` を加えるか、両端をどうするか）。
- 他の entry 型（`ApplicationPatternIndexEntry`・`PluginIndexEntry`・`QueryIndexEntry` など）と source 型の対応。今回の表は2組だけを扱っている。全 entry 型へ広げるには、対応する source 型を1つずつ決める必要がある。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

第45ラウンドで「半分だけ塞いだ」と明記した残りを塞いだ。**これで「World に在るのにグラフに出ない」型の検出は2段とも機械化された。**

ただし検出と修正は別である。今回は検出を完成させ、修正（#4001）は設計判断として残した。前ラウンドまでの「個別に辺を足す」を繰り返さないためには、この順序が正しいと考えている。
