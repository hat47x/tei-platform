# 再現の「別state」も、確かめていなかった

前ラウンドは再現の記録から「二回のbuildが**同じbuilder**だった」ことを確かめるようにしました（[記録](TEI_GO_SELF_BUILD_REPRODUCIBILITY_BUILDER_2026-11-10.md)）。本記録は、同じ観測の残る前提である「**別state**」も**確かめていなかった**ことを示し、直した結果を残します。

## 見つけたこと

§4.21の境界は「two runs of the same child executable from the same plan, **in separate state directories**」と述べています。

実装では、entryが二回目に**新しい`-state`を渡す**ところまでで、**childがその`-state`を使ったかは見ていませんでした**。

```
secondState := filepath.Join(work, "state")     // 新しいstateを作る
args := withArtifactAndState(childArgs, secondOutput, secondState)  // 渡す
driveSelfBuild(...)                             // 使ったかは確かめない
```

**childが`-state`を無視すれば、二回のbuildは同じstateを共有します。** そのとき同じバイトが出ても、それは「独立な二回」ではなく**同じ観測の繰り返し**です。名前と境界が「別state」と言っているのに、記録はそれを示していませんでした。

## 直したこと

**参照childは`stateDir`を申告します。** それを使わない手はありません。

| 変更 | 内容 |
| --- | --- |
| `childStateDir`を child evidence へ追加 | childが申告したstate |
| `reproducibilityStateVerdict(passed, reported)` | 申告が**無ければ不成立**、**渡した先と違えば不成立** |
| `firstStateDir`・`secondStateDir`を記録 | 二回それぞれに**渡した**state |
| 境界の記述 | 記録に照らして読めるようになった |

**申告が無い場合を不成立にする**のは、SC0-18（欄が無いことを一致と読ませない）と同じ規則です。**確かめていない分離を、確かめた分離として扱わない**というだけのことです。

## 副産物：二つの擬似childが「行儀の良いchild」を模すようになった

`cmd/tei`の試験用childのうち二つ（`selfBuildChildSource`・`reproducibleChildSource`）は`-state`を無視していたため、この検査で落ちました。**childが自分のstateを申告する**のは参照childと同じ振る舞いなので、二つを合わせました。

**これはテストの辻褄合わせではなく、擬似childが実childの文法に追随するという意味を持つ。**

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestReproducibilityStateVerdict`（同じ／末尾`/`違い／別ディレクトリ／未申告） | **PASS** |
| `TestSelfBuildRebuildCheckReportsReproduced`（決定的なchild） | **PASS** |
| `TestSelfBuildRebuildCheckReportsMismatch`（二回目が違うバイト） | **PASS** |
| `TestSelfBuildResolvesInProcessFromANamedCatalog`（in-processは`-rebuild-check`が既定） | **PASS** |
| `TestSelfBuildEstablishesL2BEndToEnd`（authoring経由） | **PASS** |
| `go test ./cmd/tei -run 'SelfBuild\|Reproducibility\|AssessL2B\|ApplyEvidenceDefaults'` | **PASS**（32.1秒） |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## この修正が変える主張

**変わらない**：`reproduced`の意味（同じplanを二回buildして同じバイト）。

**強くなる**：「別stateで二回」が**申告に基づいて検査される**ようになった。

**新しく不成立になる場合**：`-state`を無視する（またはstateを申告しない）child。**これらは従来`reproduced`になり得ました。**

## 言えないこと

- childが**申告どおりのstateを使ったこと**の証明。childの自己申告である（builderの他の申告と同じ限界）
- childがstateを**実際に分離して使った**こと（同じディレクトリを別名で申告する、など）
- 別ホスト・別Go版での成立

**したがって本修正は、境界の「別state」を「childがそう申告した」まで強める。それ以上は主張しない。**

## 検証

- 上の表の5試験：PASS
- `gofmt`差分なし／`go vet ./cmd/tei`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.21
- 前ラウンドの修正：[`TEI_GO_SELF_BUILD_REPRODUCIBILITY_BUILDER_2026-11-10.md`](TEI_GO_SELF_BUILD_REPRODUCIBILITY_BUILDER_2026-11-10.md)
