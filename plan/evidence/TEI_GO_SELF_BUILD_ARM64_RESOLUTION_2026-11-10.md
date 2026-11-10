# `arm64` での解決を実測した（darwin は依然として未観測）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7 は「`darwin`・`arm64`・別のGo版の組合せでの観測」を未決として残しています。前回の記録（[クロスコンパイルの範囲](TEI_GO_SELF_BUILD_CROSS_COMPILE_SCOPE_2026-11-10.md)）は「このホストでは実行できない」と結論しました。**`arm64` については、その結論を一部撤回します。**

## 何が変わったか

`qemu-aarch64-static` を**root なしで**入手できました。`multiarch/qemu-user-static` のリリースから静的バイナリを取得し、ホームディレクトリへ置くだけです（`sudo` 不要）。

```
qemu-aarch64 version 7.2.0
```

`archive.ubuntu.com` へは到達できませんでしたが（`apt-get download` がタイムアウト）、GitHub へは到達できたため、パッケージマネージャを経由せずに取得しています。

**動作確認**：`GOOS=linux GOARCH=arm64` で作った Go バイナリは、直接実行すると `Exec format error`、qemu 経由では `hello from linux/arm64` を出力しました。**本物の aarch64 コードが動いています。**

## 実測：同じ解決を二つのアーキテクチャで走らせた

`tools/reference-build-authoring-intent` を `linux/amd64` と `linux/arm64` の両方へビルドし、**同じフィクスチャ**（`tools/reference-build-authoring-intent/testdata` の `intent.json`・`catalog.json`・`src/`）に対して実行しました。

| 項目 | amd64 | arm64（qemu） |
| --- | --- | --- |
| 終了コード | **0** | **0** |
| 出力サイズ | 2,880 B | 2,880 B |
| receipt の sha256 | `597e44ee…` | `597e44ee…` |

**出力はバイト単位で一致しました。**

一致した照合値（抜粋）。

| 値 | 結果 |
| --- | --- |
| `intentFingerprint` | `c22275d8f1e4c806a91aa0835a433b3bca0b01d60b5c40652ab024cd71ebeab8` |
| `sourceClosureFingerprint` | `4e11f859d7707f0214771c19ce164d7dc3331a899e722bf644dfcdde92418dcf` |
| `planFingerprint` | `38c999fbc35263d55fb4cb7b26e43b97c3303c651e3e65ca655e3f2c6aff8169` |
| `resolutionContextFingerprint` | `bdc269c49d87487e8427cd6f62a28a30edb650f2a29862aa47e017caddf87615` |
| `builderResolutionFingerprint` | `cf874026cc005424a75e12c2ec3bb43edaa377773d724ef5c8a0888b95af0387` |
| `dependencyResolutionFingerprint` | `b2a6752a6d2de415c79edddd83415f969df6e9d824db4089e79e4cb90b600203` |
| `plan.sources[0].digest` | `sha256:eaf206359da421c9e5e815d3dd3734894749846dc0f412005923aab9a7287b8c` |
| `plan.sources[1].digest` | `sha256:7d1a595c8926467c9640b491f0800bceef02c79ad99b427541d2bfda49ec3829` |

## 手順で気をつけたこと

**最初の実行は両方とも失敗しました。** `-source-root` に `testdata` を渡していたため、`source path component "plugin.tei" does not exist` で終了コード 1、出力は 0 バイトでした。

**そのとき両アーキテクチャの出力は「一致」していました。** どちらも空だったからです。**空の一致を一致として記録しないよう、終了コードと出力サイズを先に確認し、両方が 0 で非空のときだけ比べる**手順にしました。

## 何を言っていないか

- **`darwin` は依然として未観測。** macOS の実行環境が無いという制約は変わっていません。クロスコンパイルは成功しますが、走らせる手段がありません
- **別の Go 版も未観測。** `go1.27.3` だけを使いました
- **実 build の `artifactDigest` は測っていません。** 本記録が観測したのは**解決**の値です。上の receipt にある `artifactDigest` はフィクスチャの catalog が名指しした合成値（`sha256:1111…`）であり、**ビルドが生んだ値ではありません**
- **ネイティブの arm64 実機ではない。** qemu のユーザーモードエミュレーションであり、x86_64 カーネル上の aarch64 ユーザー空間です。命令は本物の aarch64 ですが、ハードウェアもカーネルも arm64 ではありません
- **`l2b.claimed` は測っていません。** 解決だけを走らせており、child のビルドは行っていません

## なぜ一致したと考えられるか（推論であり、観測ではない）

`sourceClosure` と `plan.sources[].digest` は**ファイル内容の sha256**、`intentFingerprint` は**正規化した JSON の digest** です。定義からしてアーキテクチャに依存しません。

**だからこそ、この実測の価値は「一致するはずだ」の確認ではなく、「アーキテクチャ依存の挙動（map の反復順、浮動小数点の書式、パスの正規化など）が紛れ込んでいないこと」を実際に確かめた点にあります。** 紛れ込めば receipt のバイトが変わり、この比較で捕まります。

## 検証

- `GOOS=linux GOARCH=arm64` の Go バイナリが qemu 上で動作（`hello from linux/arm64`）
- 両アーキテクチャで `tools/reference-build-authoring-intent` をビルド（5,104,878 B / 4,838,124 B）
- 同一フィクスチャ・同一引数で実行し、**両方 exit 0・2,880 B・`cmp` で完全一致**
- 照合値は receipt から機械的に抽出

**NOT RUN**：`darwin` での実行、別の Go 版、実 build の `artifactDigest`、`l2b.claimed`。

## 限界

- エミュレーションであり、実機の arm64 ではない
- 観測したのは解決の経路だけ。ビルドを伴う経路は未観測
- Go の版を変えた場合の一致は未確認
- G4は `UNQUALIFIED` のまま。
