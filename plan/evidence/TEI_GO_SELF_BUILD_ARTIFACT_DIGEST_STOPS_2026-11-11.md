# 「成果物と違うdigestを申告する」は、ディスクとの比較で止まる（追認）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.37 は、条件5の三つの要素のうち **成果物のバイト一致は entry が自分で確かめる**としています。**本記録はその記述を実測で追認します。**

**直近2周は「entry が確かめていない」側の限界を測ってきました。本記録は逆側を確かめ、表の非対称が実測に基づくことを示します。**

## 実測

child をラッパーにし、**書いた成果物とは違う `artifactDigest` を申告**させました（実物の digest を `sha256:1111…` へ置換）。

| 項目 | 結果 |
| --- | --- |
| 終了コード | **1** |
| `ok` | **`false`** |
| `error.code` | **`SELF_BUILD_CHILD_FAILED`** |
| `error.message` | `child-reported digest differs from the artifact on disk; the child's own lock fi…` |
| `artifactCrossCheckMatched` | **`false`** |
| **`l2b.claimed`** | **`false`** |
| `l2b.unmet` | `['the plan was not bound to the artifact (mismatched)', 'the artifact was not reproduced from the plan (mismatched)']` |

**entry がディスクの byte から digest を計算し直し、申告と比べています。** 申告を偽っても通りません。

## 表の非対称は実測に基づく

§4.37 の「検査で止められるもの」表には、**止まるもの**と**止まらないもの**が混在しています。直近の2周で「止まらない」側の2件（builder同一性・state分離）を実測しました。**本記録は「止まる」側の代表例を確かめました。**

| 条件5の要素 | entry が自分で見るか | 実測 |
| --- | --- | --- |
| **成果物のバイト一致** | **見る**（ディスクから再計算） | **本記録：止まる** |
| builder同一性 | 見る（`-child` のパス） | [止まる／プロキシだと止まらない](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md) |
| state分離 | **childの申告と文字列一致** | [申告が正しければ止まらない](TEI_GO_SELF_BUILD_STATE_SEPARATION_SCOPE_2026-11-11.md) |

**「entry が自分で見る」と「止まる」は同じではありません。** builder同一性も entry が見ていますが、見ている対象（`-child` のパス）が実態と一致しない構成がありました。**成果物のバイト一致だけは、見ている対象が実態そのもの（ディスクの byte）です。**

## 何を言っていないか

- **この検査が「計画から出た成果物か」を確かめるとは言っていません。** ディスクの byte が**計画から出たか**は、条件5の再現と §4.36 の source 照合が**間接的に**支えるだけです（§4.37 の「検査で止められないもの」2）
- **child が成果物を書かなかった場合は別です。** 本記録は「書いたが違う digest を申告した」場合だけです
- `linux/amd64` の一回の観測です
- **`qualified` は `false`** です

## 検証

偽の `artifactDigest` を申告する child で、`ok:false`・exit 1・`claimed:false`・`artifactCrossCheckMatched:false` を確認しました。
