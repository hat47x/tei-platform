# catalogの7つの欄は、リポジトリのどこにも宣言されていない（§4.29）

§4.25は「既定経路をどこまで強くするか」の判断に足りないものとして、**catalog 3点セットを既定で用意できるか**を挙げました。本記録はその問いを、**catalogの各欄がリポジトリのどこで宣言されているか**という形で測ります。

[`TEI_GO_SELF_BUILD_CATALOG_PROVENANCE_2026-11-09.md`](TEI_GO_SELF_BUILD_CATALOG_PROVENANCE_2026-11-09.md)は「catalogの出所がどこかに**あるか**」を探して**無い**ことを示しました。本記録はそこから一歩進み、**欄ごとに**どこに無いのかを示します。

## 測ったこと

`BuildCatalog`の`BuildProvider`が持つ7つの欄について、`internal/`と`cmd/`から`"欄名"`を検索し、**catalogの型定義以外に宣言面があるか**を確かめました。

## 結果

| 欄 | catalogの型以外の宣言面 |
| --- | --- |
| `contractMajor` | **無い** |
| `authorityScope` | **無い** |
| `recipeKinds` | **無い** |
| `profiles` | **無い** |
| `targets`（build対象） | **無い**（`internal/index`と`internal/modelref`の`"targets"`は**意味関係の名前**で無関係） |
| `artifactDigest` | **ある**（`internal/plugin`の`BuildArtifactLock`・`LockedBuild`、宣言の`builder.pin`、runtime service carrier） |
| `contract`（名前） | 宣言にある。ただし**「どのproviderがどのcontractを提供するか」**はどこにも無い |

**四つ（`contractMajor`・`authorityScope`・`recipeKinds`・`profiles`）は、`internal/buildintent/catalog.go`とその試験、`cmd/tei`の試験用catalogにしか現れません。** `targets`も同じで、他の出現は別概念でした。

## 何が分かったか

**catalogは既存の宣言から導出できません。** 7つのうち5つに宣言面が無いためです。

| 案 | 何が要るか | 本記録の評価 |
| --- | --- | --- |
| 既存の宣言から生成する | **できない**。5つの欄の出所が無い | 不可 |
| `plugin`定義にbuilder能力を宣言させる（§4.22 K-2） | **構文拡張**。`contract`・`contractMajor`・`authorityScope`・`recipeKinds`・`targets`・`profiles`をplugin定義へ | 可能だが構文と意味の追加 |
| Workspaceにcatalog欄を設ける（§4.22 K-1） | `tei.workspace.yaml`の拡張 | 可能。Workspaceが「どのbuilderが在るか」の正本になる |
| 外部の配布物として持つ | catalog自体を成果物として作り、digestで名指しする | **§4.22 K-3が既にこれを前提にしている** |

**したがって「既定で用意できるか」の答えは「いいえ、今のリポジトリの内容からは用意できない」です。** 用意するには**宣言面を一つ増やす**決定が要ります。

## これは配線の問題ではない

§4.25の表現を借りれば「catalog 3点セットを既定で用意できるか」は、**誰がcatalogを書くか**ではなく、**catalogを書く場所を製品に作るか**という問いです。

§4.22は「**entryがcatalogをどこから読むか**」をK-3（呼び手が名指しする）に決めました。本記録はその決定を変えません。変わるのは**「そのcatalogを誰がどう作るか」が未決のまま残る**ことの重みです——既存の宣言からは作れないので、**明示的に作る場所を決めるまで、既定の試行でClaimは立てられません**。

## この実測が言えること

- catalogの7欄のうち**5欄に、catalog以外の宣言面が無い**
- したがって**catalogは既存のリポジトリ内容から導出できない**
- 既定化には**宣言面を増やす決定**が要る（K-1かK-2か、外部配布物か）

## この実測が言えないこと

- **どの案が良いか。** K-1・K-2・外部配布物のどれを採るかは契約判断であり、本記録は判断しない
- **外在するconsumer**の扱い
- catalogの中身が正しいこと（§4.22 K-3、SC0-26）
- `-strong`の名前と粒度（§4.25の三つ目の不足分。未着手）

## 検証

- `internal/`と`cmd/`の`*.go`から`"欄名"`を検索し、出現ファイルを列挙
- 各出現がcatalogの型定義・試験・fixtureのいずれかを確認
- `internal/index`・`internal/modelref`の`"targets"`が`SemanticDependency.Relation`の値であり、build対象と無関係であることを行単位で確認
- `go test ./tools/repository-plan-check -count=1`：PASS
- 文字化け検査：該当なし

**NOT RUN**：外在consumerの調査、宣言面を増やす実装（判断前のため）。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.29
- 出所が無いことの先行記録：[`TEI_GO_SELF_BUILD_CATALOG_PROVENANCE_2026-11-09.md`](TEI_GO_SELF_BUILD_CATALOG_PROVENANCE_2026-11-09.md)
- catalogの出所の決定：[同 §4.22](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 三つの未決：[同 §4.25](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
