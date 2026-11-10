# Go版とアーキテクチャを独立に動かして解決を比べた

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7 は「`darwin`・`arm64`・別のGo版の組合せでの観測」を残しています。§4.26 は二つのホストを比べましたが、**ホストとGo版が同時に動いていました**（`go1.27.0` と `go1.27.3`）。本記録は**版とアーキテクチャを独立に動かし**、4通りの組み合わせを同じコミットで比べます。

## 何を用意したか

| 用意したもの | 入手方法 |
| --- | --- |
| `go1.26.10` | `dl.google.com` から tarball を取得し、ホームディレクトリへ展開（**root不要**） |
| `qemu-aarch64-static` 7.2.0 | `multiarch/qemu-user-static` のリリースから静的バイナリを取得（**root不要**） |

`archive.ubuntu.com` へは到達できませんが、`dl.google.com` と `github.com` へは到達できました。**パッケージマネージャを経由せずに揃えています。** `go.mod` は `go 1.26.10` を要求するので、`go1.26.10` はビルド可能な下限にあたります。

## 実測：2×2の行列

同じコミット（`4f6f56ed3`）で、同じフィクスチャ（`tools/reference-build-authoring-intent/testdata`）に対して `tools/reference-build-authoring-intent` を実行しました。ビルドキャッシュは4通りとも分離しています。

| Go版 | アーキテクチャ | 終了コード | 出力 | receipt の sha256 |
| --- | --- | --- | --- | --- |
| `go1.27.3` | `amd64` | **0** | 2,880 B | `597e44ee16cd5709f7a014dfda1eff145d7d8629cf8cd105f0cfcaaf831e98ce` |
| `go1.26.10` | `amd64` | **0** | 2,880 B | 同上 |
| `go1.27.3` | `arm64`（qemu） | **0** | 2,880 B | 同上 |
| `go1.26.10` | `arm64`（qemu） | **0** | 2,880 B | 同上 |

**4つともバイト単位で一致しました。** 一致した値には `intentFingerprint`・`sourceClosureFingerprint`・`planFingerprint`・`resolutionContextFingerprint`・`builderResolutionFingerprint`・`dependencyResolutionFingerprint`・`plan.sources[].digest` が含まれます。

## この一致が意味すること（と、意味しないこと）

**一致したのは「解決」の経路だけです。** そして receipt には**ツールチェーンの欄がありません**（`formatVersion`・`authority`・各fingerprint・`plan` などが並ぶが、Go版やGOARCHを記録しない）。つまりこの比較が示すのは次です。

| 示すこと | 示さないこと |
| --- | --- |
| 解決の値は**Go版にもアーキテクチャにも依存しない** | **ビルド成果物の digest が一致すること** |
| アーキテクチャ依存の挙動（mapの反復順、浮動小数点の書式、パスの正規化）が解決に紛れ込んでいない | 同じソースから同じバイトができること |

**後者は依然として未観測です。** コンパイルされた成果物は、アーキテクチャが違えば当然バイトが違います。Go版が違えば変わり得ます。**本記録はそこを一切主張しません。**

## 何を言っていないか

- **`darwin` は依然として未観測。** macOS の実行環境が無く、クロスコンパイルは通るが走らせられません
- **実 build の `artifactDigest` は未観測。** receipt にある `artifactDigest` はフィクスチャの catalog が名指しした合成値（`sha256:1111…`）です
- **`l2b.claimed` は未観測。** 解決だけを走らせており、child のビルドを行っていません
- **ネイティブの arm64 実機ではない。** qemu のユーザーモードエミュレーション（x86_64 カーネル上の aarch64 ユーザー空間）です
- **`go1.27.0`（Windows）との比較はしていません。** 本記録の4通りはすべて `linux` です

## 途中で踏んだ誤り（手順の記録）

**最初の試行は比較になりませんでした。** `go1.26.10` のパスを `$HOME/go1.26.10/go/bin/go` と書いていましたが、実際は `$HOME/go1.26.10/bin/go` でした。このとき `go1.26.10` 側は exit 127 で出力 0 バイトでした。

**手順が「両方が exit 0 かつ非空のときだけ比べる」形だったため、`NOT COMPARABLE` と判定され、空同士の一致を一致として記録しませんでした。** 前回の記録で同じ罠を踏んだので、判定を機械化してあります。

## 検証

- 4通りをビルド（すべて exit 0）、同一引数で実行（すべて exit 0・2,880 B）
- `cmp` による完全一致、`sha256sum` による同一ハッシュの確認
- receipt にツールチェーンの欄が無いことを `grep` で確認
- receipt の各値は機械的に抽出

**NOT RUN**：`darwin`、実 build の `artifactDigest`、`l2b.claimed`、ネイティブ arm64。

## 限界

- エミュレーションであり実機ではない
- 観測したのは解決の経路だけ
- Go版は `1.26.10` と `1.27.3` の2点のみ。離れた版（1.2x 系）は未確認
- receipt がツールチェーンを記録しないため、**「どのツールチェーンで得た値か」を receipt からは辿れません**。本記録の対応は実行ログに依存します
- G4は `UNQUALIFIED` のまま。
