# `l2b.claimed` を `arm64` でも観測した（連鎖全体を実プロセスで）

直前までの記録は、`arm64` で**解決**と**パッケージの `artifactDigest`** を個別に観測しました。本記録は**連鎖全体**——`tei self-build` のエントリ自身——を `arm64` で走らせ、`l2b.claimed` を読みます。

## どうやって走らせたか

`tei` を `linux/arm64` へクロスコンパイルし、`qemu-aarch64-static` 上で実行しました。**障害が一つありました。**

**エミュレートされた arm64 プロセスは、別の arm64 バイナリを `execve` できません。** `binfmt_misc` の登録が必要で、それは root を要します。`tei self-build` は child を直接 `exec` するため、そのままでは child の起動で止まります。

**回避策**：`-child` に `/bin/sh` のラッパーを渡しました。ラッパーは x86_64 なのでネイティブに動き、その中で qemu を明示的に起動して arm64 の child を走らせます。

```sh
#!/bin/sh
exec /home/hat47x/qemuprobe/qemu-aarch64-static /tmp/l2b-arm64/bin/child "$@"
```

**`tei` 本体・child・catalog 著者ツール・resolver はすべて arm64** です。ラッパーは起動の仲介だけをします。

## 実測：両アーキテクチャで連鎖が成立した

| 項目 | `linux/amd64` | `linux/arm64`（qemu） |
| --- | --- | --- |
| `tei self-build` 終了コード | **0** | **0** |
| `ok` | `true` | `true` |
| `qualified` | `false` | `false` |
| **`l2b.claimed`** | **`true`** | **`true`** |
| `l2b.unmet` | 空 | 空 |
| `artifactReproducibility.state` | `reproduced` | `reproduced` |
| artifact のバイト数 | 463 B | 463 B |

**両方で Claim が立ち、`unmet` は空でした。** 二回目の build も既定（`-build-catalog` を渡したため §4.30）で走り、`firstArtifactDigest == secondArtifactDigest` になっています。

## 何が一致し、何が違うか（ここが要点）

| 値 | amd64 | arm64 | 判定 |
| --- | --- | --- | --- |
| `intentFingerprint` | `0909a60d…` | `0909a60d…` | **一致** |
| `sourceClosureFingerprint` | `034ca748…` | `034ca748…` | **一致** |
| `catalogDigest` | `69019e4a…` | `b1ae953e…` | **相違** |
| `planFingerprint` | `18607f38…` | `797099f6…` | **相違** |
| `artifactDigest` | `f5bc2a7f…` | `c0cabc71…` | **相違** |
| child の digest | `a2a09bab…` | `77011940…` | **相違** |

**きれいに二つに分かれました。**

- **宣言とソースのバイトだけに依存する値**（`intentFingerprint`・`sourceClosureFingerprint`）は**一致**する
- **builder のバイトに依存する値**（`catalogDigest`・`planFingerprint`・`artifactDigest`）は**相違**する

相違は**欠陥ではありません**。catalog は「実際に走る builder のバイト」を名指しし（§4.33）、amd64 の builder と arm64 の builder は別のバイトだからです。plan はその catalog を参照するので plan の照合値も変わり、artifact も builder の同一性を写すので変わります。

## この分離が意味すること

**`l2b.claimed` は、アーキテクチャを跨いだ成果物の一致を要求していません。** 要求しているのは**連鎖の内部整合**です——entry が自分で解釈したこと、宣言のバイトが一致したこと、計画が一致したこと、artifact に束縛されたこと、その計画から再現されたこと。

したがって「arm64 では Claim が立たない」理由は原理的に無く、**実測でも立ちました**。一方で「arm64 の artifact が amd64 と同じバイトになる」ことは**要求されておらず、実際になっていません**。

**この二つを混同しないことが、本記録の要点です。**

## 何を言っていないか

- **`darwin` は依然として未観測。** macOS の実行環境がありません
- **ネイティブの arm64 実機ではない。** qemu のユーザーモードエミュレーションです
- **`qualified=true` ではない。** 両方とも `false` であり、本記録は受理・Runtime Authority・L2-c・L3 を主張しません
- **クロスアーキテクチャでの成果物の再現性は主張しません。** 主張したのは「各アーキテクチャの内部で二回の build が一致した」こと（`reproduced`）だけです
- **child は `/bin/sh` ラッパー経由で起動した。** `binfmt_misc` を使う構成（root あり）とは起動経路が違います。child の実行内容は同じですが、`tei` から見た実行ファイルの同一性（`-child` のパス）は異なります
- **フィクスチャはリポジトリに無い。** 一時ディレクトリで組み立てました

## 検証

- amd64 と arm64 の双方で `tei self-build` が exit 0、`l2b.claimed=true`、`unmet` 空
- `artifactReproducibility.state=reproduced`、`firstArtifactDigest == secondArtifactDigest`
- 6つの照合値を両アーキテクチャから機械的に抽出し、一致/相違を分類
- `planArtifactBinding.artifactDigest == artifactDigestOnDisk`
- `planCrossCheck.resolverPlanFingerprint == builderReportedPlanFingerprint`

**NOT RUN**：`darwin`、ネイティブ arm64、`binfmt_misc` を用いた起動経路。

## 限界

- エミュレーションであり実機ではない
- child 起動がラッパー経由（`binfmt_misc` 無しの環境では、これが arm64 連鎖を走らせる唯一の方法）
- フィクスチャが一時ディレクトリにあり、リポジトリから再現できない
- G4は `UNQUALIFIED` のまま。
