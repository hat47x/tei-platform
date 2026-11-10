# 「builderの同一性」は、名指しされたファイルの同一性である

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.37 は、L2-bの条件5（artifactの再現）が何を止めるかを表にしている。その中に次がある。

> `二回の間に実行物を差し替える` → 条件5の**builder同一性**

**本記録は、これが「`-child` に名指しされたファイル」の同一性であって、「実際に走った構築器」の同一性ではないことを実測で示します。**

> **後続の記録による更新**：本記録の実測は、書かれた時点の観測です。その後、[宣言したbuilderと起動したファイルを結ぶ検査](TEI_GO_SELF_BUILD_BUILDER_BINDING_2026-11-10.md)（§4.42）が入りました。本記録のプロキシの構成（catalogは`builder`を宣言し、起動するのはプロキシ）は、**現行では `l2b.claimed=false` になります**。条件5の範囲についての本記録の結論はそのまま成り立ちます。

## なぜ差が出るか

エントリの child 検証は**正規ファイルであること**だけを見ます。

```go
info, err := os.Stat(exactChild)
if !info.Mode().IsRegular() {
    return ..., fmt.Errorf("child executable is not a regular file: %s", exactChild)
}
childDigest, err := digestOfFile(exactChild)
```

**シェルスクリプトは `IsRegular()` を通ります。** そして digest は `exactChild`、つまり `-child` に渡されたパスの byte です。

したがって `-child` が**別の実行物を起動するプロキシ**であれば、エントリが記録し比較する値は**プロキシの digest** です。プロキシの背後で何が走ったかは、この値に現れません。

## 実測

同一のソースから、**バイトが違うが振る舞いが同じ**構築器を二つ作りました（`-trimpath` の有無だけが違い）。

| 構築器 | digest（先頭16桁） |
| --- | --- |
| `builder-a` | `e0d7878b9f0c6b01…` |
| `builder-b` | `5e1cab04092fff20…` |

`-child` に渡したのは**プロキシ**（シェルスクリプト）で、呼ばれた回数に応じて起動先を切り替えます。1回目は `builder-a`、2回目は `builder-b` を `exec` します。

**1回の試行で、二回の build が別々の構築器で走りました。**

| 項目 | 結果 |
| --- | --- |
| プロキシの起動記録 | `1 -> builder-a`、`2 -> builder-b` |
| `firstChildDigest` | `sha256:19ee317c…`（**プロキシの digest**） |
| `secondChildDigest` | `sha256:19ee317c…`（**同じ値**） |
| `firstArtifactDigest` | `sha256:5eed4c17…` |
| `secondArtifactDigest` | `sha256:5eed4c17…` |
| `artifactReproducibility.state` | `reproduced` |
| **`l2b.claimed`** | **`true`** |
| `l2b.unmet` | 空 |

**二つの異なる構築器が走ったのに、Claimが立ちました。** 理由は一つです——同一性の検査が**名指しされたファイル**を見ており、それが両方の試行で同じプロキシだったからです。

## 対照：プロキシを挟まない差し替えは止まる

上の実験だけでは「差し替えは一切止まらない」と読めます。**そうではありません。** `-child` のファイル自体を二回の間で差し替えると、止まります。

`-child` に、**初回に自分自身を別のバイト列へ書き換えてから** `builder-a` を `exec` するスクリプトを渡しました。二回目の build は書き換わった後の内容で走ります。

| 項目 | 結果 |
| --- | --- |
| `-child` の digest（実行前 → 実行後） | `5e0ccca1…` → `b0d445dc…` |
| `firstChildDigest` | `sha256:5e0ccca1…` |
| `secondChildDigest` | `sha256:b0d445dc…`（**異なる**） |
| `firstArtifactDigest` / `secondArtifactDigest` | `sha256:5eed4c17…` で**同じ** |
| `artifactReproducibility.state` | `mismatched` |
| `mismatchReason` | `the two runs used different builder executables` |
| `l2b.claimed` | **`false`** |
| `l2b.unmet` | `the artifact was not reproduced from the plan (mismatched)` |
| `tei self-build` の終了コード | **1** |

**成果物は一致しているのに、`builder` の同一性だけで止まりました。** 差し替えの検出そのものは働いており、**効かないのは「名指しされたファイルの背後で別の実行物が走る」場合だけ**です。

エントリは build ごとに `-child` を digest し直すため、ファイルが変われば二つの値が食い違います。**§4.37 の行はこの意味では正しく、範囲が `-child` のパスまでだというだけです。**

## これは何を意味するか

| 言えること | 言えないこと |
| --- | --- |
| 条件5の「builder同一性」は、`-child` のパスの byte 同一性である | **実際に走った構築器の同一性である** |
| プロキシを挟めば、背後で構築器を替えても Claim は立つ | その Claim が「同じ構築器で二回buildした」ことを意味する |
| 成果物のバイト一致は依然として検査されている | 成果物を生んだ**プログラム**が同じであること |

**成果物の一致（`firstArtifactDigest == secondArtifactDigest`）は本物の検査です。** 本記録はそれを否定していません。否定しているのは、**その一致を「同じ構築器」に結び付ける鎖**です。

## 撤回・対処の方向（本記録は決めない）

- **`-child` に正規ファイル以上の制約を課す**（ELF/Mach-O であることの確認など）。ただしスクリプトを正当に使う呼び手を排除します
- **childに自分の実行物の digest を申告させる**。ただし**自己申告**が増えるだけで、条件2〜4と同じ弱さを持ちます
- **何もしない**。`-child` は呼び手が選ぶものであり、呼び手を信頼するなら検査の対象外である

**本記録は測っただけです。** どれを採るかは §4.37 の持ち主が決めます。

## 何を言っていないか

- **`darwin` は扱っていません。** `linux/amd64` のみです
- **悪意ある呼び手の一般論ではありません。** 示したのは「名指しされたファイルと実際に走るものが一致しない構成があり得る」ことだけです
- **手順・フィクスチャ・生出力はリポジトリにある。** `plan/evidence/TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10/` の `run.sh`（`proxy`・`direct`）が両方の観測を再導出します（[README](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10/README.md)）。**構築器はリポジトリのソースからビルドするので、digest はコミットで変わります。**
- **本記録の構築器は振る舞いが同じです。** 振る舞いが違えば成果物が変わり、`reproduced` にならないので Claim は立ちません
- **`qualified` は `false` のまま**です。本記録は受理・L3・Runtime Authority を主張しません

## 副次的な発見

§4.37 の表の条件5の「塞いだ節」欄に **`(直近2件)`** という**未完成の参照**が残っています。どの節を指すのか本文から特定できないため、**本記録では修正していません**（推測で埋めると誤った参照を固定するため）。

## 検証

- 二つの構築器の digest が異なることを `cmp` で確認
- プロキシの起動記録で、1回目と2回目が別の構築器だったことを確認
- `tei self-build` が exit 0、`l2b.claimed=true`、`unmet` 空、`reproducibility=reproduced`
- `firstChildDigest == secondChildDigest`、`firstArtifactDigest == secondArtifactDigest`

**NOT RUN**：`darwin`。

## 限界

- `linux/amd64` の一回の観測です
- プロキシが意図的に切り替える構成を作っており、通常の利用ではありません
- 成果物が一致する範囲でしか Claim が立たないため、**振る舞いの違う構築器への差し替えは成果物の不一致として止まります**
- G4は `UNQUALIFIED` のまま。
