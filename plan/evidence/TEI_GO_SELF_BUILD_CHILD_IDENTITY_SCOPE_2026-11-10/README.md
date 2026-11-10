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

> **現行の期待は、宣言したbuilderと起動したファイルを結ぶ検査（SB-C0 §4.47、[記録](../TEI_GO_SELF_BUILD_BUILDER_BINDING_2026-11-10.md)）の後の値です。** 上の表の `proxy` の期待（`true`）は、検査が入る前の観測です。`raw/*-summary.txt` はその時点の出力で、`raw/*-summary-with-builder-binding.txt` が検査の後の出力です（下の表）。

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

## 検査の後の生出力（`*-summary-with-builder-binding.txt`）

| 項目 | `proxy` | `direct` |
| --- | --- | --- |
| `l2b.claimed` | **`false`** | **`false`** |
| `l2b.unmet` | `the file this entry started was not shown to be the builder the plan declares (mismatched)` のみ | 再現の不一致と、上の宣言との不一致の二つ |
| `reproducibility` | `reproduced`（再現だけでは見えない） | `mismatched`（`the two runs used different builder executables`） |

- `proxy` を止めているのは**宣言との照合だけ**です。成果物は一致し、二回の child digest も等しいままです。
- `direct` は、catalog が宣言した `builder` ではなく**シェルのラッパー**を `-child` に渡すので、再現とは別に宣言との不一致も立ちます。ファイルの差し替えは従来どおり再現で止まります。
- **取得環境**：WSL2（Ubuntu 24.04、linux/amd64）。**ネイティブのLinux版Goではなく**、Windows版Go 1.27.0に`GOOS=linux`を渡すラッパー経由でビルドしました（`raw/commit-with-builder-binding.txt`）。構造は確かめましたが、digestの値はネイティブ版のツールチェーンとは異なり得ます。
## この一式が主張しないこと

- **`darwin` は扱いません。** `linux/amd64` のみです
- **悪意ある呼び手の一般論ではありません。** 示すのは「名指しされたファイルと実際に走るものが一致しない構成があり得る」ことだけです
- **振る舞いが違う構築器への差し替えは、成果物の不一致として別途止まります**（本一式はそこで止まる場合を扱いません）
- **`qualified` は `false`** です。受理・Runtime Authority・L3 を主張しません

## 値はコミットに依存する

構築器と各ツールはこのリポジトリのソースからビルドするので、**コミットが変われば digest も変わります**。`raw/` はその時点の実行です。**再導出で確かめられるのは、上の表の「どちらが一致し、どちらが止まるか」という構造**です。
