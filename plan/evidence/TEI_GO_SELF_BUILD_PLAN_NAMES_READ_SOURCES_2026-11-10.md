# 計画は、ビルドが読むsourceだけを名指す（§4.46）

§4.41は「ビルドが計画に載っていない package を読む」を拒否しました。**逆向きの食い違い** — 計画が、ビルドが読まない source を名指す — は残っていました。§4.40で`definitionSource`が`go-package`になった後も、宣言は`plugin-definition`を計画に残していた（写すだけで、ビルドは読まない）ためです。

## 規則

**`go-command`の計画は、`go-package`と`go-module`だけを名指せる。** それ以外の kind があれば**ビルドしません**。

| なぜ | 内容 |
| --- | --- |
| 記述の一致 | 使わない source を載せると、計画は「ビルドより多い」ことを言う。読み手はどちらが本意か分からない |
| §4.41との対称 | あちらは「ビルドが計画より多い」、こちらは「計画がビルドより多い」。**どちらも計画とビルドが別物** |

**`go-command`以外の recipe は従来どおり**です（規則を決めた範囲だけに適用）。

## あわせて直したこと：定義sourceのkindを両側で揃える

`definitionBytesDigest`の照合（§4.36・§4.40）が**`plugin-definition`固定**だったため、宣言を外すと`definitionBinding`が`unperformed`になりました。**builderとentryの両方で、recipeに応じてkindを決める**ようにしました（`go-command`なら`go-package`）。

**片側だけ直すと、静かに`unperformed`へ落ちます。** 実際にその順序で踏みました。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestCheckSourcesAreReadAcceptsTheSourcesACommandBuildUses` | **PASS** |
| `TestCheckSourcesAreReadRefusesASourceTheCommandBuildNeverReads` | **PASS** |
| `TestCheckSourcesAreReadLeavesOtherRecipesAlone` | **PASS** |
| `TestRunRefusesAPlanWithAnUnusedSource`（試行ごと拒否） | **PASS** |
| `TestSelfBuildEstablishesL2BFromARealGoBuild`（宣言を外した計画で`l2b.claimed=true`・`definitionBinding=bound`・`memberCount=3`） | **PASS**（26.6秒） |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## 言っていないこと

- **`go-command`以外の recipe の「読む source」の集合**。決めていない
- 計画に載る`go-package`が**複数**の場合の扱い（今は全部写す）
- 記録側（`sources`）の意味の変更。**計画の source は、ビルドが読むものだけになった**

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.46
- 逆向きの規則：[同 §4.41](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 定義sourceのkind：[同 §4.40](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
