# `artifactDigest` を `arm64` でも実測した（plugin-package について）

直前の二つの記録は、`arm64` で観測できたのは**解決**の値だけで、**実 build の `artifactDigest` は未観測**だと述べました。本記録は、その一部を埋めます。

**埋まったのは `plugin-package` という成果物の種類についてだけです。** コンパイルされた `runtime-artifact` については、依然として未観測です（後述）。

## 何をしたか

リポジトリの `tools/reference-plugin-package-build-child` は、planから**パッケージを materialize** する child です。外部コマンドを呼ばず（`os/exec` を使わない）、決定的なデータを書きます。したがって**アーキテクチャを跨いで比べられます**。

| 段 | 使ったもの |
| --- | --- |
| 能力宣言 → catalog | `tools/reference-build-catalog-authoring`（`amd64` で著者し、両方で読む） |
| 宣言 → plan | `tools/reference-build-authoring-intent`（両アーキで実行） |
| plan → artifact | `tools/reference-plugin-package-build-child`（両アーキで実行） |

catalog は**一度だけ** `amd64` で著者しました（`catalogDigest=sha256:f9c9d3ab…`）。両アーキテクチャの resolver が同じ catalog を読みます。

## なぜ既製のフィクスチャを使わなかったか

`tools/reference-build-authoring-intent/testdata` の宣言は `recipe.kind=go-package`・`outputKind=runtime-artifact` です。この child は `plugin-package` しか materialize しないので、**既製のフィクスチャでは走りません**（実測：`plugin package materialization requires plugin-package output, got runtime-artifact` で exit 1）。

そこで、`tei-plugin-package` を広告する catalog と、それに合う宣言を用意しました。**製品の契約は変えていません**。catalog は製品の外のデータであり、`§4.32`・`§4.33` が決めたとおりです。

## 実測

| アーキテクチャ | 解決 | child | artifact | `artifactDigest` |
| --- | --- | --- | --- | --- |
| `linux/amd64` | exit 0 | **exit 0** | 493 B | `sha256:7623c4512540e7bd2e2631a33925d24a96d812660de9649e1ff61a71760d0442` |
| `linux/arm64`（qemu） | exit 0 | **exit 0** | 493 B | **同じ値** |

**artifact のバイトは完全に一致しました**（`cmp` で同一、`sha256sum` も同一）。

これで §4.26 が比べた四つの値のうち、`artifactDigest` を除く三つ（`intentFingerprint`・`sourceClosure`・`planFingerprint`）に加えて、**`artifactDigest` も `arm64` で一致した**ことになります。ただし対象は下記の種類に限ります。

## 何を言っていないか

- **コンパイルされた成果物の `artifactDigest` は未観測。** `runtime-artifact` は Go のビルド出力であり、**アーキテクチャが違えばバイトが違うのが当然**です。本記録の一致は、artifact が**決定的なデータの直列化**だからです。**「build すればどのアーキでも同じバイトになる」とは言っていません**
- **`darwin` は依然として未観測。** macOS の実行環境がありません
- **`l2b.claimed` は未観測。** child を直接起動しており、`tei self-build` の五条件を評価していません
- **ネイティブの arm64 実機ではない。** qemu のユーザーモードエミュレーションです
- **本記録のフィクスチャはリポジトリに無い。** 一時ディレクトリで組み立てた宣言と catalog であり、再現手順は本記録の記述に依存します

## 何が確かめられたか

| 確かめられたこと | 根拠 |
| --- | --- |
| catalog の読み取りがアーキテクチャ非依存 | 同じ `catalogDigest` の catalog を両方の resolver が読んで同じ plan を出した |
| 解決がアーキテクチャ非依存 | 直前の記録（2×2の行列）と同じ結論 |
| **`plugin-package` の materialize がアーキテクチャ非依存** | **artifact がバイト単位で一致** |
| child の自己申告 digest がディスクのバイトと一致 | `artifactDigest` と `sha256sum` が同じ値 |

## 検証

- 4つのバイナリ（resolver・child × 2アーキ）をビルドし、すべて exit 0
- catalog を `amd64` で著者（672 B）、両アーキの resolver が同一 plan を導出
- 両アーキの child が exit 0、493 B の artifact を書いた
- `cmp` による完全一致、`sha256sum` と child の `artifactDigest` が一致

**NOT RUN**：`runtime-artifact` の build、`darwin`、`l2b.claimed`、ネイティブ arm64。

## 限界

- エミュレーションであり実機ではない
- 一致したのは**データ成果物**についてだけ。コンパイル成果物は対象外
- フィクスチャが一時ディレクトリにあり、リポジトリから再現できない
- G4は `UNQUALIFIED` のまま。
