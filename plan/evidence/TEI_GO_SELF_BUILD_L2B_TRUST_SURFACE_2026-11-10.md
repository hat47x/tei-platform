# L2-bの五条件が何を確かめ、何を信じているかの整理（§4.37）

§4.23の述語は五条件の同時成立を要求します。本記録は、**各条件がどこまで検査で、どこからが自己申告か**を一か所にまとめたものです。**新しい主張はせず**、`main`の`533177b16`の実装を読んで確認した内容を固定します。

## なぜ整理するか

**§4.21・§4.24・§4.36で三つの穴を塞いだ結果、条件ごとの中身が変わっています。**

| 塞いだ穴 | 節 | 何が変わったか |
| --- | --- | --- |
| 「同じbuilder」ではなくentryのdigestが入っていた | §4.21 | 条件5が**builder同一性**を検査するようになった |
| 「別state」を渡すだけで使ったか見ていなかった | §4.21 | 条件5が**state分離**を検査するようになった |
| 計画が名指すsourceをchildが読んでいなかった | §4.36 | **入力の一致**がchild側で検査されるようになった |

**欄の名前と中身が一致したので、読む人が「欄があるから確かめた」と読まないよう、現状を固定する必要がある。**

## 条件ごとの内訳

| 条件 | 検査（entryが自分で見る） | 自己申告（childの言い分） |
| --- | --- | --- |
| 1. entry自身が計画を導いた | `derivedBy`・`state`・`unmet`の整合 | — |
| 2. 宣言byteの一致 | 二つのdigestの**比較** | **childが実際にそのbyteを読んだこと** |
| 3. 計画fingerprintの一致 | 二つの値の**比較**（両方空なら`unperformed`） | **childの計画fingerprint** |
| 4. plan→artifactの束縛 | **ディスクのbyteのdigest**とlockの整合・検証 | childの`artifactDigest` |
| 5. artifactの再現 | 二回の**バイト一致**・**builder同一性**・**state分離**・ディスクの一致 | childの`stateDir` |

（表は設計文書§4.37に、各節への参照つきで置いた）

## 検査で止められるもの／止められないもの

**止められる**：宣言と違うbyteを申告する、計画と違うfingerprintを申告する、成果物と違うdigestを申告する、二回で違うバイトを出す、二回の間に実行物を差し替える、`-state`を無視する、**計画と違うdefinitionを読む**（childが成果物を作る前に停止）。

**止められない（信じている）**：childの「自分が何をしたか」という申告そのもの。具体的には (1) 宣言を実際に読んだこと、(2) 計画から成果物を作ったこと、(3) 申告どおりのstateを使ったこと、(4) §4.36の停止が働くこと。

**つまりL2-bは「entryの観測」と「childの申告」の合成であり、childを敵と見なす安全性ではない。** これは§4.8がL2-bを「解釈の層」として定義したことの帰結です。

## 敵対的なchildを想定するなら

| 必要になるもの | 閉じるもの |
| --- | --- |
| OSレベルの読み取り制限 | 「読んだこと」 |
| child実行の監視（open/execve） | 「作ったこと」 |
| entryが用意したstateをchildに渡さない | 「state」 |
| §4.36の照合をentry側でも持つ | 「停止が働くこと」 |

**いずれも本設計の範囲外です。** 敵対的builderへの防御は別の層（供給のTrust・Seed Trust・Runtime Authority）の仕事であり、いずれも未主張のままです。

## 検証

- 設計文書§4.37として記録した
- `go test ./tools/repository-plan-check -count=1`：PASS
- `doc-reference-probe` 未解決参照0件／文字化け検査 該当なし

**NOT RUN**：CI（文書のみ）。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.37
- 述語：[同 §4.23](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 三つの穴：[`TEI_GO_SELF_BUILD_REPRODUCIBILITY_BUILDER_2026-11-10.md`](TEI_GO_SELF_BUILD_REPRODUCIBILITY_BUILDER_2026-11-10.md)・[`TEI_GO_SELF_BUILD_REPRODUCIBILITY_STATE_2026-11-10.md`](TEI_GO_SELF_BUILD_REPRODUCIBILITY_STATE_2026-11-10.md)・[`TEI_GO_SELF_BUILD_SOURCE_BINDING_2026-11-10.md`](TEI_GO_SELF_BUILD_SOURCE_BINDING_2026-11-10.md)
