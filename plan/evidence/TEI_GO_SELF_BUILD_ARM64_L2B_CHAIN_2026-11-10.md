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
| `-child` に渡したファイルの digest | `a2a09bab…` | `77011940…` | **相違**（下記） |

**`-child` の digest については、何を数えた値を書く必要があります。** この試行は child を `/bin/sh` のラッパー経由で起動したので、エントリが記録する値は**ラッパーの byte digest** であり、arm64 の builder バイナリの digest ではありません（`cli_self_build.go` の `childDigest, err := digestOfFile(exactChild)`、`exactChild` は `-child` のパス）。

**相違の原因は二つあります。** 一つは arm64 と amd64 で builder のバイトが違うこと、もう一つは**ラッパーの内容がアーキテクチャごとに違う**こと（arm64 版はエミュレータのパスを埋め込む）。したがってこの行は、builder のバイトについて何も述べていません。

**そして、catalog が名指しした builder のバイトと、実際に起動したファイルは照合されません。** 本試行では catalog が `builder`（arm64 の builder バイナリ）を名指しし、エントリは `child-run`（ラッパー）を起動しました。それでも Claim が立ったのは、**エントリが両者を digest で照合しないから**です。これは §4.33 が明記しているとおりで（「builderの解決はdigestで照合しないため、catalogが別のバイトを名指ししていても試行は通ってしまう」）、対応は**試験**が固定しています。**本記録はその対応を確かめていません。**

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
- **child は `/bin/sh` ラッパー経由で起動した。** `binfmt_misc` を使う構成（root あり）とは起動経路が違います。そのため `childDigest` はラッパーの digest であり、**arm64 の builder バイナリの digest ではありません**
- **catalog が名指しした builder のバイトは、本試行では照合されていない。** エントリは `-child` のパスと catalog の `artifactDigest` を比べません（§4.33）。この対応を確かめるのは試験であり、本記録の範囲外です
- **フィクスチャと手順はリポジトリにある。** `plan/evidence/TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10/` の `run.sh`・`fixture/`・`raw/` が再導出の一式です（[README](TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10/README.md)）。**ただし builder に依存する値はコミットで変わる**ので、`raw/` の再実行（`4d04036e4`）では `catalogDigest`・`planFingerprint`・`artifactDigest` が本記録の値と一致しません。一致するのは構造です

## 検証

- amd64 と arm64 の双方で `tei self-build` が exit 0、`l2b.claimed=true`、`unmet` 空
- `artifactReproducibility.state=reproduced`、`firstArtifactDigest == secondArtifactDigest`
- 6つの照合値を両アーキテクチャから機械的に抽出し、一致/相違を分類
- `planArtifactBinding.artifactDigest == artifactDigestOnDisk`
- `planCrossCheck.resolverPlanFingerprint == builderReportedPlanFingerprint`

**NOT RUN**：`darwin`、ネイティブ arm64、`binfmt_misc` を用いた起動経路。

## 再導出

手順・フィクスチャ・生出力を `plan/evidence/TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10/` に置きました。リポジトリのルートから `run.sh amd64` と `run.sh arm64` を実行すると同じ連鎖を再導出できます（`arm64` はエミュレータが要り、無い場合は終了コード 2 で理由を出して止まります）。

**値はコミットに依存します。** `raw/` は `4d04036e4` での再実行であり、本記録の表（`706fab080`）とは `catalogDigest`・`planFingerprint`・`artifactDigest` が一致しません。**再導出で確かめられるのは「どちらが一致し、どちらが相違するか」という構造です。**

## 限界

- エミュレーションであり実機ではない
- child 起動がラッパー経由（`binfmt_misc` 無しの環境では、これが arm64 連鎖を走らせる唯一の方法）
- フィクスチャが一時ディレクトリにあり、リポジトリから再現できない
- G4は `UNQUALIFIED` のまま。

## 2026-11-13 の再確認（`main` `059b96fc3`）

**条件5の表を全行実測した後、同じ `run.sh amd64` が今の `main` でも同じ結論を出すかを確かめました。**

| 項目 | 値 |
| --- | --- |
| `ok` | **`true`** |
| **`l2b.claimed`** | **`true`** |
| `l2b.unmet` | **`null`**（空） |
| `artifactReproducibility` | **`reproduced`** |
| `qualified` | **`false`** |
| `catalogDigest` | `sha256:1ba0cbd2…` |
| `planFingerprint` | `3d44ab8a…` |
| `artifactDigest` | `sha256:e2e098b4…` |

**`catalogDigest`・`planFingerprint`・`artifactDigest` は本記録の表とも `raw/` とも一致しません。** 記録済みのとおり、これらの値はコミットに依存します。**一致したのは「Claimが立ち、`unmet`が空で、再現が成立する」という結論の側です。**

`qualified` が `false` のままである点も変わりません（§7の`darwin`が未観測）。

### 両アーキテクチャでの再確認

同じ `run.sh` を2アーキテクチャで実行しました（`QEMU=/home/hat47x/qemuprobe/qemu-aarch64-static`）。

| 項目 | amd64 | arm64（qemu） |
| --- | --- | --- |
| `ok` | **`true`** | **`true`** |
| **`l2b.claimed`** | **`true`** | **`true`** |
| `l2b.unmet` | **`null`** | **`null`** |
| `artifactReproducibility` | **`reproduced`** | **`reproduced`** |
| `qualified` | **`false`** | **`false`** |
| `catalogDigest` | `sha256:1ba0cbd2…` | `sha256:6189107b…` |
| `planFingerprint` | `3d44ab8a…` | `301c80d8…` |
| `artifactDigest` | `sha256:e2e098b4…` | `sha256:256c3bb3…` |
| `intentFingerprint` | `0909a60d…` | **`0909a60d…`（一致）** |
| `sourceClosure` | `034ca748…` | **`034ca748…`（一致）** |

**`intentFingerprint` と `sourceClosure` は両アーキテクチャで一致しました。** `catalogDigest`・`planFingerprint`・`artifactDigest` は一致しません（builderのバイナリが違うため。**Claimはアーキテクチャ間の成果物一致を要求していません**）。

**`qualified` は両方とも `false`** です（`darwin` 未観測）。

