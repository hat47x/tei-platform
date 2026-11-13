# `-state-boundary` は書込を閉じるが、既存ディレクトリの「読取共有」は閉じない

[#4486](https://github.com/hat47x/tei-platform/issues/4486) の S2 を実装しました。**`-state-boundary` を付けると、2回目のbuildが「rootが読み取り専用で、渡したstateと成果物ディレクトリだけ書ける」mount namespace の中で走ります。** 実測で**期待どおりに動く範囲と、動かない範囲**が分かれました。

## 実装

| 部分 | 内容 |
| --- | --- |
| launcher | `stateBoundaryLauncher`（`unshare -rm sh -c <script> sh`）。root を ro にし、`ALLOW_STATE`・`ALLOW_ART` だけ rw に戻し、`ALLOW_FIRST_STATE` を空のtmpfsで覆う |
| 起動 | `driveSelfBuildIn`（記録するbuilderは**`-child`のまま**） |
| 入口 | `tei self-build -state-boundary`（opt-in） |
| 報告 | `stateSeparation.basis` が **`host-boundary`**、`scope` は「何をしたか」だけ |

## 実測（同一fixture、4ケース）

| ケース | 終了 | `claimed` | `basis` |
| --- | --- | --- | --- |
| 同梱child・境界なし | 0 | `true` | `entry-post-hoc` |
| **同梱child・境界あり** | **0** | **`true`** | **`host-boundary`** |
| **B（1つのstateを共有＋空ディレクトリ）・境界あり** | **0** | **`true`** | **`host-boundary`** |
| B・境界なし | 0 | `true` | `entry-post-hoc` |

**正例は両方通り、B は境界を付けても落ちません。**

## なぜ B が落ちないか（重要）

**この B は、共有ディレクトリに対して何も書きません。** 同梱childは `-state` を `mkdir` するだけで、そのディレクトリは1回目のbuildで既に存在します。**書込が発生しないので、書込を禁じる境界では捕まりません。**

[#4486](https://github.com/hat47x/tei-platform/issues/4486) の B2 実験で B が `read-only file system` で落ちたのは、**その負例が実際に書いていた**からです。

**したがって境界が閉じるのは「書込による共有」だけです。** 既存ディレクトリを**読む**共有は閉じません（#4486 の C5 残余「読取共有」のまま）。

## 記録の書き方

`basis=host-boundary` は**「何をしたか」**（root ro・許可ディレクトリのみ rw・1回目のstateをtmpfsで被覆）を述べ、**「分離が証明された」とは述べません**。B が通る以上、後者を書けません。

## 副産物：launcher の実バグを Step Code のレビューが発見

初版は `Launcher[1:] + child` を `argv[1:]` で exec しており、**launcher の第1引数が落ちていました**（`sh -c script sh` では script がファイル名扱い）。**修正済み。**

## していないこと

- **読取共有は閉じていません**（本記録の中心的な限界）
- ネットワーク・常駐Process・環境変数は境界外
- `linux/amd64`（WSL2）の各1回。**Windows・`darwin`は対象外**（mount namespace はLinux限定）
- 境界は**opt-in**で、既定の挙動は変えていません
- **`qualified` は `false`**、G4は `UNQUALIFIED` のまま

## 独立レビュー（Step Code / `step-3.7-flash`）と、その指摘への対応

**実装担当とは別のモデルに、diff と制約を渡してレビューさせました。** 指摘は3件で、**2件は実在、1件は誤り**でした。

| 指摘 | 判定 | 対応 |
| --- | --- | --- |
| **1. `-state-boundary` が `-rebuild-check` を暗黙に有効にしておらず、単独で渡すと黙って無視される** | **実在** | `parsed.StateBoundary` のときに `RebuildCheck` を立てるよう修正。ヘルプの「implies `-rebuild-check`」と実装を一致させた |
| **2. `scope` の文言が機構より強い。`/proc`・`/sys`・`/dev` など別マウントは書き込み可能なまま** | **実在** | 文言を「root tree を ro にし、許可ディレクトリを rw に戻した。**root と別にマウントされたファイルシステムは対象外**」へ変更。「他に何も書けなかった」とは書かない |
| **3. `checkArtifactReproducibility` の引数名が `stateBondary` という綴り誤り** | **誤り** | 実物は `stateBoundary`。**指摘に合わせて直す必要は無いので直していません** |

**3件目を「直さなかった」ことを記録します。** レビューの指摘を機械的に適用すると、**存在しない欠陥に合わせてコードを歪める**ことになります。**指摘は実物と突き合わせて採否を決める**という手順を踏みました。

指摘1・2は、いずれも**「静かに無効化される」種類の欠陥**（フラグが効かない／文言が実態より強い）で、実装者の視点では見落としやすいものでした。
