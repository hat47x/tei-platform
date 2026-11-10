# 計画が名指すsourceを、builderが読んでいなかった

§4.8(c)は「**その計画がそのartifactを生んだ**」を主張します。本記録は、その連鎖に**入力の照合が無かった**ことを示し、塞いだ結果を残します。

## 見つけたこと

計画は`Sources`に**source closureのdigest**を持ちます。entryは`-build-source-root`からsourceを解決してこのdigestを作ります。

**ところがbuilder childは、計画が名指すdigestを入力と突き合わせていませんでした。** 参照childは`-plugin`で渡されたファイルを読み、そのまま解析して成果物を作ります。

```go
definitionSource, err := os.ReadFile(pluginPath)   // 読む
definition, err := syntax.NewParser(string(definitionSource)).ParsePlugin()  // そのまま解析
artifact, err := plugin.MaterializePluginPackage(definition, plan)           // そのまま作る
```

つまり、**計画が名指すsourceと、実際に読んだファイルが別でも、誰も気づきません**。

| 観測 | この状態で成立するか |
| --- | --- |
| `planCrossCheck=matched` | **成立する。** 計画fingerprintはchildが計画ファイルから取る |
| `planArtifactBinding=bound` | **成立する。** 成果物digestはchildの申告とディスクが一致する |
| `artifactReproducibility=reproduced` | **成立する。** 二回とも同じ「別のファイル」を使えば同じバイト |

**どれも「入力が計画のものか」を見ていません。** 計画がSを名指し、childがS'を読んで成果物Xを作っても、三つの観測はすべて通ります。Xは「Sから作られるはずの成果物」ではありません。

## 直したこと

**childが、自分が読んだdefinitionのdigestを計画の値と突き合わせ、違えば成果物を作る前に停止する。**

| 変更 | 内容 |
| --- | --- |
| `buildintent.SingleFileSourceDigest` | 単一ファイルsourceのdigest。**resolverと同じ枠組み**（`len(名前):名前\n長さ\n`＋バイト）で計算する |
| child | 読んだdefinitionのdigestを計画の`plugin-definition` sourceと比較。**違えば停止**（成果物を作らない）。計画がdigestを持たない場合も停止 |
| child の報告 | `definitionBytesDigest`と`definitionMatchesPlan`を追加 |

**照合をchildに置いたのは、計画と入力の両方を持つ唯一の場所だからです。** entryはchildが何を読んだか見えません。

**枠組みを一か所にしたのは、二か所に書くと必ず食い違うからです。** 実際、当初は生の`sha256`で比較して食い違い、resolverの枠組み（メンバー名と長さを先に書く）を知って直しました。

## 副産物：三つのfixtureが「存在しないbuild」を記述していた

| 場所 | 何が起きていたか |
| --- | --- |
| `internal/buildintent/catalog_test.go` | — |
| `tools/reference-plugin-package-build-child/main_test.go` | 計画が`sha256:aaaa…`という**実在しないdigest**を名指ししていた（検査が無かったので通っていた） |
| `cmd/tei`のe2e | 検証は通るが、実際には計画と入力の対応を見ていなかった |

**fixtureが実在しないdigestを持っていたこと自体が、この検査が無かった証拠**です。実ファイルのdigestに直しました。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestSingleFileSourceDigestAgreesWithTheResolver`（同じバイトで**resolverと一致**） | **PASS** |
| `TestSingleFileSourceDigestFollowsTheBytes`（バイトが変われば変わる） | **PASS** |
| `tools/reference-plugin-package-build-child` 全試験（計画と入力が食い違う計画を拒否） | **PASS** |
| `cmd/tei` 全suite | **PASS**（565.8秒） |
| `internal/buildintent`・参照ツール2件 | **PASS** |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## この修正が変える主張

**変わらない**：(c)の意味。

**強くなる**：計画が名指す入力と実際に読んだ入力の一致が、**成果物を作る前に**確かめられる。

**新しく不成立になる場合**：計画のdigestと実際のファイルが違うbuild。従来は三つの観測すべてが通った。

## 言えないこと

- **childの実装が正しいこと**（従来どおり。childが自分で申告し、自分で止まる）
- source closureが**ディレクトリ**の場合の照合。今回は単一ファイル（`plugin-definition`）だけを塞いだ。複数メンバーのclosureは同じ枠組みで計算できるが、child側の照合は未実装
- 計画が名指す他のsource（`go-package`など）の照合
- 別ホスト・別Go版での成立

**したがって本修正は、単一ファイルのdefinitionについて「計画の入力＝実際の入力」を確かめる。それ以上は主張しない。**

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.8
- 再現の観測：[`TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md`](TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md)
- 前ラウンドの修正：[`TEI_GO_SELF_BUILD_REPRODUCIBILITY_STATE_2026-11-10.md`](TEI_GO_SELF_BUILD_REPRODUCIBILITY_STATE_2026-11-10.md)
