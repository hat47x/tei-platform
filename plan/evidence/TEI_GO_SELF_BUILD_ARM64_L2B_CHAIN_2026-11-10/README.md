# `arm64` で L2-b 連鎖を再導出する

[`../TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10.md`](../TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10.md) の観測を、リポジトリから再導出するための一式です。

**記録は「フィクスチャがリポジトリに無い」ことを限界として挙げていました。** このディレクトリがそれを埋めます。

## 走らせ方

リポジトリのルートから。

```bash
plan/evidence/TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10/run.sh amd64
plan/evidence/TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10/run.sh arm64
```

`arm64` には aarch64 のエミュレータが要ります。

| 方法 | 指定 |
| --- | --- |
| 既にある | `QEMU=<qemu-aarch64-static のパス>` |
| 取得する（root 不要） | `QEMU_DOWNLOAD=1`（`$HOME/qemuprobe` へ） |
| 無い | 理由を出して**終了コード 2** で止まる（結果を主張しない） |

## 何をするか

一時ディレクトリに次を組み立て、`tei self-build` まで通します。リポジトリへ生成物を残しません。

```text
能力宣言（capability-declaration.json）
  -> catalog（reference-build-catalog-authoring が著者）
  -> plan（reference-build-authoring-intent が解決し、child へ渡す plan を書く）
  -> tei self-build（entry 自身が in-process で plan を導出し、二回 build する）
  -> l2b の判定
```

`tei`・child・著者ツール・resolver は**すべて対象アーキテクチャ向けにビルド**します。

## なぜ child をラッパー経由で起動するか

**エミュレートされた arm64 プロセスは、別の arm64 バイナリを `execve` できません。** `binfmt_misc` の登録が必要で、それは root を要します。`tei self-build` は child を直接 `exec` するため、そのままでは止まります。

そこで `-child` に `/bin/sh` のラッパーを渡します。ラッパーは x86_64 なのでネイティブに動き、中でエミュレータを明示的に起動します。**連鎖の全ツールは arm64** で、ラッパーは起動の仲介だけをします。

## 中身

| パス | 内容 |
| --- | --- |
| `run.sh` | 上記の手順。`amd64`・`arm64` を引数に取る |
| `fixture/tei.workspace.yaml` | 試行に渡す作業領域 |
| `fixture/plugin.tei` | 対象プラグインの記述 |
| `fixture/implementation.go.txt` | `go-package` の source |
| `fixture/intent.json` | ビルド宣言 |
| `fixture/capability-declaration.json` | 能力宣言（catalog の入力） |
| `raw/commit.txt` | 下の生出力を取ったコミット |
| `raw/amd64-summary.txt` | amd64 の実行結果（`run.sh` の出力そのまま） |
| `raw/arm64-summary.txt` | arm64 の実行結果（同上） |

## 生出力から読めること

`raw/` は `4d04036e4` での再実行です。

| 値 | amd64 | arm64 | 判定 |
| --- | --- | --- | --- |
| `l2b.claimed` | `True` | `True` | 一致 |
| `l2b.unmet` | 空 | 空 | 一致 |
| `reproducibility` | `reproduced` | `reproduced` | 一致 |
| `intentFingerprint` | `0909a60d…` | `0909a60d…` | **一致** |
| `sourceClosure` | `034ca748…` | `034ca748…` | **一致** |
| `catalogDigest` | `84177d3b…` | `65278fa8…` | **相違** |
| `planFingerprint` | `96f28e0d…` | `f4fa9db6…` | **相違** |
| `artifactDigest` | `dce79cda…` | `634ff2ad…` | **相違** |

**宣言とソースのバイトだけに依存する値は一致し、builder のバイトに依存する値は相違します。**

## 値はコミットに依存する

**builder に依存する値は、リポジトリのコミットが変われば変わります。** 記録本体の表は `706fab080` での観測、この `raw/` は `4d04036e4` での再実行であり、`catalogDigest`・`planFingerprint`・`artifactDigest` の値は一致しません。**一致するのは「どちらが一致し、どちらが相違するか」という構造です。**

## この一式が主張しないこと

- **`darwin` は扱いません。** macOS の実行環境が要ります
- **ネイティブの arm64 実機ではありません。** エミュレーションです
- **`qualified` は `false`** です。受理・Runtime Authority・L2-c・L3 を主張しません
- **クロスアーキテクチャの成果物再現性は主張しません。** 主張するのは各アーキテクチャの内部で二回の build が一致したこと（`reproduced`）です
