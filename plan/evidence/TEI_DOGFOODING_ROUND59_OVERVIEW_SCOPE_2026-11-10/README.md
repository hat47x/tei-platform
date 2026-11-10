# #4128 の E 案: `model overview` が自分の対象範囲を明示する（2026-11-10、第59ラウンド）

- 対象: [#4128](https://github.com/hat47x/tei-platform/issues/4128)（E 案を実装）。
- 前段: [第58ラウンド](../TEI_DOGFOODING_ROUND58_INSPECT_DELIVERY_2026-11-10/README.md)で C 案を実装した。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **E 案を実装。C・E が完了。#4128 の残りは A・B・D。**

## 1. 目的

第57ラウンドで「`model overview` が `sectionTotal: 8` を返すので『これで全部』と読める」と記録した。E 案は**実装を変えずに、対象範囲を応答と `-h` で明示する**。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `886abf56d`（`origin/main`） |
| CLI実行物（修正前） | `sha256:fc778a88cc0ee2911ce2f2e5271858239cc9c98e282214058e17ddf04a3d166d` |
| CLI実行物（修正後） | `sha256:7c2f43dff7591dcdd07b66d4706d244a6759a5d98fa46d84afd9ac376a9cbd7c` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 実装

応答に `sectionScope` を足した。**3つとも計算で出す**ので、説明が実装からずれない。

```go
type ModelSectionScope struct {
	RootsAre        string   `json:"rootsAre"`
	CoveredKinds    []string `json:"coveredKinds"`
	NotCoveredKinds []string `json:"notCoveredKinds"`
}
```

- `rootsAre` — セクションの選択規則（`catalog-root-entries`）。
- `coveredKinds` — **catalog の entries から実際に集めた**種。
- `notCoveredKinds` — **World を見て、宣言があるのに catalog が載せていない種**。`declaredLayerKinds` の表（14層）と World の件数の積で出す。

`-h` にも明示した。

```text
sections mirror the model catalog roots. The response's sectionScope names
the kinds the catalog covers and the declared kinds it does not, so a section
count is never a count of every declaration.
```

## 4. 実測（CLI）

```text
sectionTotal: 8
rootsAre: catalog-root-entries
coveredKinds: application, application-pattern, query, semantic,
              semantic-attribute, semantic-capability, semantic-data,
              semantic-relation, semantic-type, semantic-validation
notCoveredKinds: plugin, bundle, database, persistence-mapping, migration,
                 migration-provider, application-release, database-release,
                 solution-release, environment, custom-go-binding,
                 header-detail-contract-query-binding,
                 contract-query-executor-binding, contract-query-sql-realization
```

**導出であることの確認**: 配備層（`databases`・`releases`・`plugins`・`environments`・`bindings`）を消した写しでは `notCoveredKinds` が14件から3件に縮んだ。固定文言ではない。

## 5. 試験

| 試験 | 内容 |
| --- | --- |
| `TestOverviewSectionScopeIsStated` | `rootsAre` が規則を名指しし、`coveredKinds` が空でなく、reference が宣言している migration・database-release・solution-release が `notCoveredKinds` に載る |
| `TestEverySectionRootIsACatalogEntry` | 全セクションの根の種が `coveredKinds` に含まれる（説明と実際の一致） |
| `TestDeclaredLayerKindsAreAccountedFor` | `declaredLayerKinds` の各層で、宣言があれば covered か notCovered のどちらかに必ず現れる |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `notCoveredKinds` を常に空にする | FAIL `"migration" is declared ... but is not named as not covered: []` |
| M3 | 表から `migration` を落とす | FAIL 同旨（一覧から migration だけ消える） |
| M4 | `rootsAre` を `every-declaration` に変える | FAIL `rootsAre is "every-declaration", want catalog-root-entries` |
| M2 | 被覆済みの種を `notCoveredKinds` に混ぜる | **反証できず**（下記） |

### 5.2 恒真な試験を1つ削除した

最初に `TestSectionScopeListsAreDisjoint`（2つの一覧が重複しない）を書いた。変異で壊そうとして**壊せない**ことが分かった。

`sectionScope` は `covered[layer.kind]` が真なら `continue` するので、**被覆済みの種が `notCoveredKinds` に入る経路が構造的に存在しない**。表に被覆済みの種（`application`）を足しても、`continue` により一覧は変わらない。

**反証できない試験は偽の安心を与えるので削除した。** 残した3つは、いずれも変異で落ちることを確認している。

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/operations` | ok |
| `./cmd/tei` | ok（109.6s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. #4128 の残り

| 案 | 状態 |
| --- | --- |
| C `inspect` の受理種を広げる | 完了（第58ラウンド） |
| **E 範囲を明示する** | **完了（本ラウンド）** |
| A `model catalog` に種を加える | 未着手（名前空間フィルタ設計に触れる） |
| B `overview` にセクションを加える | 未着手 |
| D 専用コマンド | 未着手 |

## 8. 読み取れないこと（NOT RUN）

- A・B・D の採否と実装。
- `notCoveredKinds` に載る層が実際に読めるようになること（E は「載っていない」と言うだけで、読めるようにはしない）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**「数えている範囲を、数える側が言う」ようにした回**である。`sectionTotal: 8` は嘘ではないが、8が全部だと読まれる余地があった。`notCoveredKinds` が14層を名指しするので、その誤読は閉じた。

**恒真な試験を自分で見つけて削った**のも記録に値する。変異試験を書かなければ、反証不能な試験を「守り」として数えていた。
