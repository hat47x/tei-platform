# catalogを名指しすれば`-rebuild-check`を書かなくてもClaimが立つ（§4.30）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.25は既定経路の強化を四案に整理し、決定を保留しました。本記録は**一つの場合に限って既定を強くした**結果を測ります。

## 決めたこと

**`-build-catalog`を渡した試行は、`-rebuild-check`を明示しなくても二回目のbuildを行う。** `-rebuild-check=false`を明示すれば従来どおり一回で終わります。

| 呼び手がしたこと | 二回目のbuild |
| --- | --- |
| `-build-catalog`を渡した（`-rebuild-check`に触れない） | **行う** |
| `-build-catalog`と`-rebuild-check=false` | 行わない |
| `-build-catalog`と`-rebuild-check` | 行う |
| `-intent-resolver`（catalogなし） | **行わない**（Claimが立たないため） |
| どちらも無し | 行わない |

## 観測結果

同じfixtureで三つの場合を実プロセスで走らせました。

| 場合 | `reproducibility` | `l2b.claimed` | `unmet` |
| --- | --- | --- | --- |
| **catalogのみ（`-rebuild-check`を書かない）** | **`reproduced`** | **`true`** | 空 |
| catalog＋`-rebuild-check=false` | `unperformed` | `false` | `the artifact was not reproduced from the plan (unperformed)` |
| catalog＋`-rebuild-check` | `reproduced` | `true` | 空 |

**一つ目の場合が本節の要点です。** これまでは`-rebuild-check`を書き忘れると、**何も言われずにClaimが立たない**状態でした。今は**catalogを渡せばClaimが立つ**のが既定です。

### 理由の文言も分けた

`-rebuild-check=false`のときの`unperformedReason`は次のようになります。

```text
the caller turned off the second build (-rebuild-check=false)
```

「誰も頼まなかった」と「呼び手が断った」を区別します。in-processでは二回目のbuildが既定なので、**無いことは選択の結果**だからです。

## なぜこの一つだけか

**catalogを名指しすることは、既に「強い経路を選ぶ」という表明です。** §4.22のK-3によりcatalogは呼び手が用意するデータで、渡すには手間がかかります。その手間を払った呼び手には、**同じ経路で取れる最後の観測を既定で取る**のが費用と便益の一致する形です。

**child経路は違います。** そちらは`l2b.claimed`が立てられないので、二回目のbuildを足しても費用だけが増えます。§4.25が「P-2は非対称」と呼んだ点を、**既定に持ち込まない**形にしました。

## 何が変わっていないか

| 項目 | 状態 |
| --- | --- |
| 既定の導出経路 | **childのまま**（R-2据え置き） |
| 何も渡さない試行 | 計画も導出せず、二回目のbuildも行わない |
| `qualified` | **`false`** のまま |
| catalogの出所 | K-3のまま。**中身を誰が作るかは未決**（§4.29） |

**「既定の試行でClaimが立つ」ようになったわけではありません。** catalogを渡すという一手が要ります。§4.29が示したとおり、そのcatalogを用意する場所はまだ製品に無い。

## 覆す条件

- catalogを渡す呼び手が二回目のbuildを常に不要とする場合（費用が受容できない）
- 二回目のbuildが失敗しやすい場合（部分的な成果を返すchildなど）

いずれも`-rebuild-check=false`で回避でき、**既定を戻すのは一行の変更**です。

## 検証

- 新規試験`TestApplyEvidenceDefaults`：**6ケース**（in-process×未指定／明示false／明示true、child経路×未指定／明示true、導出なし）。**ヘルパーが他の引数を変えないこと**も確認
- 新規試験`TestParseSelfBuildOuterArgsRecordsWhetherTheCallerSpoke`：未指定・`=false`・`true`の3通りで`RebuildCheckSet`が正しいこと
- `TestSelfBuildEstablishesL2BEndToEnd`：**`-rebuild-check`を渡さずに**`l2b.claimed=true`・`unmet`空・`qualified=false`を確認（実child）
- `TestSelfBuildWithoutTheRebuildCheckNamesTheMissingCondition`：`-rebuild-check=false`で`claimed=false`、**理由が「turned off」**であることを確認
- 実行3回：上の表のとおり
- `gofmt -l cmd/tei`：差分なし／`go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -timeout 3600s`：**PASS**（全suite）
- `go test ./tools/repository-plan-check -count=1`：PASS
- 文字化け検査：該当なし

**NOT RUN**：Windows（`TestApplyEvidenceDefaults`は純粋関数なのでOS非依存）、実build childでの既定化の所要時間（§4.28の1.926倍がそのまま当てはまる）、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.30
- 四案の整理：[同 §4.25](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 費用の実測：[`TEI_GO_SELF_BUILD_REBUILD_CHECK_COST_2026-11-09.md`](TEI_GO_SELF_BUILD_REBUILD_CHECK_COST_2026-11-09.md)・[`TEI_GO_SELF_BUILD_GO_BUILD_REBUILD_COST_2026-11-09.md`](TEI_GO_SELF_BUILD_GO_BUILD_REBUILD_COST_2026-11-09.md)
- catalogの出所（未決）：[`TEI_GO_SELF_BUILD_CATALOG_FIELD_PROVENANCE_2026-11-09.md`](TEI_GO_SELF_BUILD_CATALOG_FIELD_PROVENANCE_2026-11-09.md)
