# 実buildで`-rebuild-check`の比は1.93倍になった（§4.28）

§4.27は参照child（plugin package、buildは3.4 ms）で`-rebuild-check`のコストを測り、**増分の大半はentryの固定費**だと記録しました。同時に「実buildでは増分がbuild時間に比例し、比は2倍に近づく」と**推論**しました。

**本記録はその推論を実buildで確かめます。**

## 何を測ったか

`tools/reference-go-build-with-supply`をbuilder childとして`tei self-build`を走らせました。このchildは**実際に`go build`を実行**し、`-artifact-out`と`-state`を尊重します（`-rebuild-check`が差し替えるのはこの二つである）。

| 項目 | 値 |
| --- | --- |
| 対象package | `./internal/core`（外部module依存なし） |
| child | `reference-go-build-with-supply`、`-go`にexact Goを指定 |
| 供給 | 空のsupplyディレクトリ（`GOPROXY=file://…`） |
| 環境 | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| 試行 | 5回ずつ、ウォームアップ1回の後、中央値で比較 |

## 観測結果

| 条件 | 中央値 | 最小 | 最大 |
| --- | --- | --- | --- |
| `-rebuild-check`なし | **5852.7 ms** | 5562.9 ms | 6739.0 ms |
| `-rebuild-check`あり | **11272.4 ms** | 10406.8 ms | 11792.5 ms |
| child単体（Go build） | **5568.2 ms** | 5352.4 ms | 6770.8 ms |

| 値 | 結果 |
| --- | --- |
| 中央値の差 | **+5419.7 ms** |
| 中央値の比 | **1.926倍** |

## §4.27の推論は当たっていた

| 予想（§4.27） | 実測（本記録） |
| --- | --- |
| 増分は「entryの固定費 ＋ childの仕事」 | 増分5419.7 ms ≒ child単体5568.2 ms ＋ 固定費 |
| buildが重いほど比は2倍に近づく | **1.926倍** |
| 参照childでは固定費が支配的（+110 ms、1.807倍） | 実buildでは**buildが支配的**（+5.4秒、1.926倍） |

**同じflagが、childの重さによって意味の違う費用になる。** 参照childでは「entryの段取り代」、実buildでは「buildをもう一度」である。

## 副産物：Go buildのartifactも再現した

`artifactReproducibility.state`は**`reproduced`**で、二回のbuildのdigestが一致しました。

| 欄 | 値 |
| --- | --- |
| `firstArtifactDigest` | `sha256:1bcf33c219385ba5cd1ff6f4c230e43f056b7d2ec57548a557991a2407acffa0` |
| `secondArtifactDigest` | **同じ値** |

**plugin packageだけでなく、Go buildの成果物も同じplanから同じバイトになった。** §4.21の再現は参照childに限った話ではなかったことになる。

`l2b.claimed`は`false`のままです。entryが解釈していない（catalogを渡していない）ためで、述語どおりの動作です。

## 既定化の判断に効くこと

| 問い | 材料 |
| --- | --- |
| 参照childで既定にできるか | +110 ms（§4.27） |
| **実buildで既定にできるか** | **+5.4秒。試行が5.85秒から11.27秒になる** |
| `-rebuild-check`を既定にすべきか | **依然として決めない。** 費用が2桁違う |

**「実buildでは約2倍」が実測で裏づけられました。** 既定化は「すべての試行の壁時計時間をほぼ倍にする」という判断になります。§4.25のP-2（再現だけ既定）は、この費用を全試行に課すことになります。

## この実測が言えないこと

- **別のpackage・別ホスト**での時間。`./internal/core`は小さく、`./cmd/tei`のような重い対象では絶対値が変わる
- **既定化の是非**。費用の材料であり、判断ではない
- catalog 3点セットの出所（§4.25の二つ目の不足分。未着手）
- `-strong`の名前と粒度（三つ目。未着手）
- 意味の受理・Runtime Authority・Trust

## 検証

- 5回ずつ計測、ウォームアップ1回、中央値で比較
- すべての試行で`ok=true`を確認（`ok_light`・`ok_heavy`とも`true`）
- 実行したchildは実際に`go build`を行い、成果物のdigestを申告する
- `go test ./tools/repository-plan-check -count=1`：PASS
- 文字化け検査：該当なし

**NOT RUN**：重いpackage（`./cmd/tei`等）での計時、Windows・macOSでの計時、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.28
- 参照childでの測定：[`TEI_GO_SELF_BUILD_REBUILD_CHECK_COST_2026-11-09.md`](TEI_GO_SELF_BUILD_REBUILD_CHECK_COST_2026-11-09.md)
- 三つの未決：[同 §4.25](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
