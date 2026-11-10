# `-rebuild-check`の実コストを測る（§4.27）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.25は「既定経路をどこまで強くするか」の三つの未決を一つにまとめ、**決めるために足りないもの**を三つ挙げました。本記録はその一つ、**実buildの所要時間**を測ります。

## 何を測ったか

同じ試行を`-rebuild-check`あり・なしで繰り返し、**中央値**を比べました。7回ずつ、ウォームアップ1回の後。

| 項目 | 測り方 |
| --- | --- |
| 対象 | 参照catalog＋参照builder child（`reference-plugin-package-build-child`） |
| 環境 | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| 計時 | `self-build`一回の壁時計時間（プロセス起動を含む） |
| child単体 | childを直接起動した壁時計時間（比較のため） |

## 観測結果

| 条件 | 中央値 | 最小 | 最大 |
| --- | --- | --- | --- |
| `-rebuild-check`なし | **136.4 ms** | 118.3 ms | 177.7 ms |
| `-rebuild-check`あり | **246.5 ms** | 230.8 ms | 327.9 ms |
| child単体 | **3.4 ms** | 3.1 ms | 3.9 ms |

| 値 | 結果 |
| --- | --- |
| 中央値の差 | **+110.1 ms** |
| 中央値の比 | **1.807倍** |

同時に、意図したとおりに動くことも確認しました。

| 欄 | `-rebuild-check`なし | あり |
| --- | --- | --- |
| `artifactReproducibility.state` | `unperformed` | **`reproduced`** |
| `l2b.claimed` | `false` | **`true`** |

## 分かったこと：増分はbuildではない

**child単体は3.4 msしかないのに、増分は110.1 msです。** つまり二回目のコストの**大半はchildの仕事ではなく、entry側の固定費**です。

二回目のためにentryが行うこと:

- 一時ディレクトリの作成と削除
- child argvの差し替えと起動（`driveSelfBuild`の一式）
- 二回目のreportの解析
- artifactの再読とdigest計算

**したがって「build時間が倍になる」という§4.25の表現は、参照childについては正確ではありません。** 参照childのbuildは3.4 msで、倍にしても6.8 msにしかならない。実際に倍になるのは**entryの一回分の固定費**です。

## 実buildではどうなるか（推論であり実測ではない）

増分は「entryの固定費 ＋ childの仕事」です。したがって:

| childの仕事 | 増分 | 比 |
| --- | --- | --- |
| 3.4 ms（参照child、実測） | +110 ms | 1.8倍 |
| 1秒程度（小さなGo build） | 約 +1.1秒 | 約 1.8倍（Attemptが約1.2秒なら） |
| 数十秒（大きなGo build） | 約 +build時間 | **約2倍に近づく** |

**buildが重いほど比は2倍に近づきます。** これは測定ではなく、増分の内訳からの推論です。**実buildでの測定は行っていません。**

## この実測が判断に与えるもの

§4.25が挙げた「実buildの所要時間」は、**参照childでは110 msの固定費**であることが分かりました。判断に効く形に直すと次のようになります。

| 問い | 本記録が与える材料 |
| --- | --- |
| 参照childで既定にできるか | **+110 ms**。試行が136 msから246 msになる。**絶対値としては小さい** |
| 実buildで既定にできるか | **未測定。** 増分はbuild時間に比例して増える。**大きなbuildでは約2倍**になる |
| 既定化の是非 | **本記録は決めない。** 費用は「誰の試行か」に依存する |

**「絶対値が小さいから既定にしてよい」とは言いません。** 参照childの110 msは、このPoCの最小のbuildだからです。実際のGo buildを伴う試行では比が2倍に近づくため、**同じ判断を同じ根拠で下せません。**

## この実測が言えないこと

- **実build（Go buildを伴うchild）の所要時間**。測っていない
- **別ホスト**の時間。プロセス起動の費用はOSで変わる
- `-rebuild-check`を既定にすべきか。**費用の材料であり、判断ではない**
- catalog 3点セットを既定で用意できるか（§4.25の二つ目の不足分。未着手）
- `-strong`の名前と粒度（三つ目。未着手）

## 残る不足分

§4.25が挙げた三つのうち、本記録が埋めたのは**一つ目だけ**です。

| 足りないもの | 状態 |
| --- | --- |
| 実buildの所要時間 | **参照childについて測定済み**（本記録）。実buildは未測定 |
| catalog 3点セットを既定で用意できるか | **未着手。** §4.22末尾の「catalogの中身を誰が作るか」と同じ問い |
| `-strong`の名前と粒度 | **未着手** |

## 検証

- 7回ずつ計測、ウォームアップ1回、中央値で比較
- 計測スクリプトは`self-build`を実際に起動し、戻り値のJSONで`artifactReproducibility.state`と`l2b.claimed`も確認
- `go test ./tools/repository-plan-check -count=1`：PASS
- 文字化け検査：該当なし

**NOT RUN**：実build（Go buildを伴うchild）の計時、Windows・macOSでの計時、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.27
- 三つの未決：[同 §4.25](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 再現の設計：[同 §4.21](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
