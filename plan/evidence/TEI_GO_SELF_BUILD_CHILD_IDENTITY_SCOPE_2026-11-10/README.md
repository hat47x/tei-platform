# child同一性の範囲を再導出する

[`../TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md`](../TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md) の二つの観測を、リポジトリから再導出するための一式です。

## 走らせ方

リポジトリのルートから。

```bash
bash plan/evidence/TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10/run.sh proxy
bash plan/evidence/TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10/run.sh direct
```

| モード | 何をするか | 期待 |
| --- | --- | --- |
| `proxy` | `-child`のプロキシが**自分のバイトを変えずに**起動先を切り替える | **`l2b.claimed=true`**（差し替えは見えない） |
| `direct` | `-child`のファイル自体が初回に書き換わる | **`l2b.claimed=false`**、`mismatched`、`mismatchReason`が実行者違いを名指しする |

> **`proxy` の期待は、`raw/` を取ったコミットでの値です。** その後、宣言したbuilderと起動したファイルを結ぶ検査が入りました（[記録](../TEI_GO_SELF_BUILD_BUILDER_BINDING_2026-11-10.md)、SB-C0 §4.47）。このスクリプトはcatalogに`builder`を宣言してプロキシを起動するので、**現行では `l2b.claimed=false`**（`builderBinding=mismatched`、再現は`reproduced`のまま）になるはずです。**現行のコードでは未実行**で、`raw/` と `run.sh` の出力欄は更新していません。

エミュレータは要りません。どちらもエントリの記帳に関する観測で、childのアーキテクチャに依存しないためです。

## 何をするか

一時ディレクトリに次を組み立て、`tei self-build` まで通します。リポジトリへ生成物を残しません。

```text
能力宣言 -> catalog（reference-build-catalog-authoring）
         -> plan（reference-build-authoring-intent）
         -> tei self-build（二回目のbuildは既定。§4.30）
         -> l2b の判定
```

同じソースから、**バイトが違うが振る舞いが同じ**構築器を二つ作ります（`-trimpath` の有無のみ）。振る舞いが同じなので**成果物は一致**し、`builder` の同一性だけが判定を分けます。

## 中身

| パス | 内容 |
| --- | --- |
| `run.sh` | 上記の手順。`proxy`・`direct` を引数に取る |
| `fixture/tei.workspace.yaml` | 試行に渡す作業領域 |
| `fixture/plugin.tei` | 対象プラグインの記述 |
| `fixture/implementation.go.txt` | `go-package` の source |
| `fixture/intent.json` | ビルド宣言 |
| `raw/commit.txt` | 下の生出力を取ったコミット |
| `raw/proxy-summary.txt` | `proxy` の出力（`run.sh` の出力そのまま） |
| `raw/direct-summary.txt` | `direct` の出力（同上） |

## 生出力から読めること

| 項目 | `proxy` | `direct` |
| --- | --- | --- |
| `l2b.claimed` | **`true`** | **`false`** |
| `l2b.unmet` | 空 | `the artifact was not reproduced from the plan (mismatched)` |
| `reproducibility` | `reproduced` | `mismatched` |
| `mismatchReason` | — | `the two runs used different builder executables` |
| `firstChildDigest` = `secondChildDigest` | **等しい**（プロキシのバイトが変わらない） | **異なる** |
| `firstArtifactDigest` = `secondArtifactDigest` | **等しい** | **等しい** |

**どちらのモードでも成果物は一致しています。** 違うのは `builder` の同一性だけで、それが判定を分けます。

## この一式が主張しないこと

- **`darwin` は扱いません。** `linux/amd64` のみです
- **悪意ある呼び手の一般論ではありません。** 示すのは「名指しされたファイルと実際に走るものが一致しない構成があり得る」ことだけです
- **振る舞いが違う構築器への差し替えは、成果物の不一致として別途止まります**（本一式はそこで止まる場合を扱いません）
- **`qualified` は `false`** です。受理・Runtime Authority・L3 を主張しません

## 値はコミットに依存する

構築器と各ツールはこのリポジトリのソースからビルドするので、**コミットが変われば digest も変わります**。`raw/` はその時点の実行です。**再導出で確かめられるのは、上の表の「どちらが一致し、どちらが止まるか」という構造**です。
