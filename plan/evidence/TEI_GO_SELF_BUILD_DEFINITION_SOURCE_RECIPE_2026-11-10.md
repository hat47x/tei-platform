# 定義sourceの種類はrecipeが決める（§4.40）

段2で`./cmd/tei`の実buildを鎖に通したとき、**宣言の`definitionSource`が`plugin-definition` kindを要求する**ために、commandのビルドにも既存の`plugin.tei`を定義sourceとして差し込む必要がありました。**計画は、そのビルドが読まないファイルの指紋を運んでいました。**

本記録は、その規則を**recipeに応じて変えた**ことを残します。

## 規則

| recipe | 定義sourceに要求するkind | 計画の`pluginDefinitionFingerprint`の出どころ |
| --- | --- | --- |
| `go-command` | **`go-package`**（ディレクトリ可） | **sourceのidentityそのもの** |
| それ以外 | `plugin-definition`（単一ファイル） | 解析した宣言の`BuildDefinitionFingerprint` |

**「主体を定義する入力」は、何を作るかで変わります。** pluginは宣言が定義であり、commandはコンパイルされるpackageが定義です。**単一の規則を緩めるのではなく、recipeごとの規則にしました。**

## 変えたところ

| 場所 | 変更 |
| --- | --- |
| 宣言検証（`internal/buildintent`） | kindの検査を`validateDefinitionSourceKind(recipe, source)`へ |
| 解決 | `go-command`では定義sourceを**pluginとして解析しない**。単一ファイル要求も無い。指紋はsourceのidentityから |
| 計画 | `pluginDefinitionFingerprint`の意味は「主体を定義する入力の同一性」で、**recipeが導出法を決める** |

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestValidateAcceptsTheDefinitionKindEachRecipeNeeds` | **PASS** |
| `TestValidateRefusesTheDefinitionKindTheRecipeDoesNotUse`（**両方向**で拒否し、必要なkindを名指す） | **PASS** |
| `TestValidateStillRequiresADeclaredDefinitionSource` | **PASS** |
| `TestResolveDerivesTheDefinitionIdentityPerRecipe`（同じ入力で安定・bytesが変われば変わる） | **PASS** |
| `go test ./internal/buildintent ./tools/reference-build-authoring-intent ./cmd/tei -run 'Definition\|Resolve\|Intent'` | **PASS** |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## 言っていないこと

- **既存の宣言の移行**。`plugin-definition`を使う宣言は今までどおりで、変更は要らない
- `go-command`以外の非plugin recipeの規則（今は`go-command`だけを分けた）
- **欄の名前**。commandでは`pluginDefinitionFingerprint`はpluginを指さないが、名前を変えると計画の形が変わるため、意味の説明で補った
- 受理・Runtime Authority・供給のTrust。いずれも未主張のまま

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.40
- 段2の記録（この穴が見つかった場所）：[`TEI_GO_SELF_BUILD_L2B_FROM_REAL_GO_BUILD_2026-11-10.md`](TEI_GO_SELF_BUILD_L2B_FROM_REAL_GO_BUILD_2026-11-10.md)
