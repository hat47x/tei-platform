# World の全欄と参照グラフの対応を検査で固定した（2026-11-10、第45ラウンド）

- 対象: [#3419](https://github.com/hat47x/tei-platform/issues/3419)（型付き契約）、[#3983](https://github.com/hat47x/tei-platform/issues/3983)、[#4001](https://github.com/hat47x/tei-platform/issues/4001)。
- 前段: 第41〜44ラウンドで「モデルにあるが参照グラフに無い」型を4回検出した。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **同じ型の再発を機械的に検出する試験を追加。**

## 1. 目的

第28ラウンド（`capability.outcome`）・第41〜42（`realize` 束縛）・第43（他の3種の束縛）・第44（永続化の列対応）で、**同じ型の欠陥を4回**見つけた。個別に辺を足すだけでは、次に `ir.World` へ欄が増えたときに同じ手順を繰り返す。再発を機械的に検出する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `85b90d89c`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 実測: 17欄のうち3欄が読まれていない

`ir.World` の全欄について、`index.Build` が読むか（`w.<欄>` の出現）を機械的に数えた。

| 欄 | `index.Build` が読む |
| --- | --- |
| `Plugins`・`Bundles`・`Semantics`・`Applications`・`Databases` | 読む |
| `Persistence`・`Migrations`・`MigrationProviders` | 読む |
| `ApplicationReleases`・`DatabaseReleases`・`SolutionReleases` | 読む |
| `Environments`・`Queries`・`Bindings` | 読む |
| **`HeaderDetailContractQueryBindings`** | **読まない** |
| **`ContractQueryExecutorBindings`** | **読まない** |
| **`ContractQuerySQLRealizations`** | **読まない** |

**14/17 が読まれ、読まれない3欄は第43ラウンドで特定した3種と完全に一致した。** 数え方が正しいことの裏付けでもある。

## 4. 追加した試験

`internal/index/world_field_coverage_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestEveryWorldFieldIsAccountedFor` | `reflect` で `ir.World` の全欄を列挙し、`worldFieldCoverage` に無い欄があれば落ちる。逆方向（表にあるが World に無い欄）も落ちる |
| `TestUnreadWorldFieldsAreExplained` | `read: false` の欄には理由が必須。理由の無い穴を許さない |

`worldFieldCoverage` は「読む／読まない」と「読まない理由」を1箇所に持つ。新しい欄を足した人は、ここに判断を書かないと `go test ./internal/index` が落ちる。

```go
"HeaderDetailContractQueryBindings": {
	whyUnread: "indexed nowhere; #3983 records this as a known gap",
},
```

**「読まれていない」を失敗にせず、明示を要求する形にした。** plugin・release・environment のように、参照グラフの subject ではない欄が正当に存在するため。

### 4.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `ir.World` に未登録の欄を追加 | FAIL `ir.World.HypotheticalNewField is not in worldFieldCoverage; decide whether index.Build reads it and say so here` |
| M2 | 未読の欄から理由を削除 | FAIL `HeaderDetailContractQueryBindings is marked unread without a reason` |

## 5. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/index ./internal/modelref/... ./internal/operations` | すべて ok |
| `go vet ./internal/index` | 指摘なし |
| `gofmt -l .` | 空 |

## 6. これは何を保証し、何を保証しないか

**保証する**: `ir.World` に欄が増えたとき、「その欄をグラフに反映するか」の判断を書かずに通ることがない。第43ラウンドで私がやった「隣の欄を数える」作業が、機械的に強制される。

**保証しない**: 読むと書いた欄が、実際に**辺として**グラフへ出るか。`Persistence` は「読む」だが、`#4001` のとおり列対応は辺になっていない（`Mapping.Columns` を索引が捨てている）。つまり**この試験は「索引が World を読むか」までで、「World の内容がグラフに出るか」は見ていない。**

```
ir.World ──[この試験が見る]──> index.Build ──[見ていない]──> 参照グラフ
```

`#4001` の穴はこの試験では捕まらない。**半分だけ塞いだ**ことを明記する。

## 7. 読み取れないこと（NOT RUN）

- 「読む」と書いた欄の内容が辺になるかの検査（`#4001` の `Mapping.Columns` はここで落ちる）。索引の各 entry 型が World の下位構造を捨てていないかを見る試験が別途要る。
- 参照グラフに出る関係の閉じた集合を契約にすること（#3419 の本体）。今回は実装側の検査に留めた。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

4ラウンド続けた個別修正を、**再発検出**に変えた。ただし塞いだのは「索引が World を読むか」の1段だけで、`#4001` が示した「索引が下位構造を捨てる」段は残っている。

次の段（entry 型が World の下位構造を落としていないか）は、`PersistenceIndexEntry` のような型ごとの判断が要るので、別の設計として起票・判断するのが妥当と考えている。
