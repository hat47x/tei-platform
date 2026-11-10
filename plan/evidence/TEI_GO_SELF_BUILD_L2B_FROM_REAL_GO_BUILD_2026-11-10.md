# 実buildで`l2b.claimed=true`が立った

§4.39の段1（計画を読むbuilder）を実装した後、**段2**として、そのbuilderをcatalogのproviderとして宣言し、`tei self-build -build-catalog`で鎖を通しました。本記録は、**実際の`go build ./cmd/tei`で`l2b.claimed=true`が立った**ことを残します。

## 何を通したか

| 段 | 内容 |
| --- | --- |
| 宣言 | `subject`は`tei.native.runtime.reference@1.0.0`、`recipe.kind=go-command`、`entrypoint=./cmd/tei`、`outputKind=runtime-artifact`、sourceは`plugin-definition`（既存の`plugin.tei`）と`go-package`（`cmd/tei`） |
| catalog | `tools/reference-build-catalog-authoring`で**能力宣言から作成**（providerの`artifact`は実際のbuilderバイトから計算） |
| 計画 | `tools/reference-build-authoring-intent`で解決（`planFingerprint=0a6fec065bdea0f8fc501dac90c7ee6b99e245daac4f226d540a87637b9883a1`） |
| build | `tei self-build -build-catalog … -child tools/reference-go-build-from-plan -child-arg '["-plan",…,"-intent",…,"-repo",…,"-go",…]' -supply … -rebuild-check=true` |

## 結果

```
ok: true
l2b.claimed: True
l2b.unmet: None
planReceipt: resolved (entry-in-process)
planCrossCheck: matched
planArtifactBinding: bound
artifactReproducibility: reproduced
childRun.artifactDigest: sha256:88ad1d9836b8ba468301a3e2c63dabceb62439101a1bb6fb4ff51b3aa0e62e2b
artifactCrossCheckMatched: true
```

**五条件がすべて成立しました。** 対象は`./cmd/tei`そのものです。

## 途中で分かったこと（記録）

| # | 分かったこと | 対応 |
| --- | --- | --- |
| 1 | 宣言の`definitionSource`は**`plugin-definition` kindのsourceを指さなければならない**。commandのビルドには定義が無いので、既存の`plugin.tei`を定義sourceとして使った | **宣言モデルがplugin前提**であることの現れ。commandだけの宣言は書けない |
| 2 | `reference-go-build-from-plan`は`GOPROXY=off`の固定設定のため、依存のある`./cmd/tei`をビルドできなかった | `gobuilder.BuildPackageWithSupply`を足し、toolに`-supply`を追加 |
| 3 | `-child-arg`に`-supply`を書くと**entryが拒否**する（entryが自分で供給を渡す） | entryの`-supply`を使う |

## 残っている穴

| 穴 | 内容 |
| --- | --- |
| `definitionBinding: unperformed` | 新しいbuilderは`definitionBytesDigest`を申告していない。**§4.36のentry側照合がこのbuilderでは効いていない** |
| **回帰試験が無い** | 本記録は**一度通しただけ**である。CIで毎回は通っていない |
| TOCTOU | 照合とcompileの間にファイルが変わり得る |
| 宣言モデル | commandのビルドに`plugin-definition`を要求する形は、意味として正しくない可能性がある |

## 検証

- 上の実行結果（`ok=true`・`l2b.claimed=true`・`unmet`空）
- `go test ./tools/reference-go-build-from-plan`：**PASS**（43.0秒）
- `go test ./tools/internal/gobuilder`：WSLのworktreeでは`git`が`.git`のWindowsパスを解決できず2件が失敗する（環境の制約）。**同じ2件をWindowsで実行してPASS**

**NOT RUN**：CI、`darwin`・`arm64`。

## 次の一歩

**回帰試験に載せること**（§4.39の段3）。載せない限り、この到達点は「一度通した」ままである。あわせて`definitionBytesDigest`の申告を足し、§4.36の照合をこのbuilderでも効かせる。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.39
- 段1：[`TEI_GO_SELF_BUILD_BUILDER_FROM_PLAN_2026-11-10.md`](TEI_GO_SELF_BUILD_BUILDER_FROM_PLAN_2026-11-10.md)
- 能力の一覧：[同 §4.38](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
