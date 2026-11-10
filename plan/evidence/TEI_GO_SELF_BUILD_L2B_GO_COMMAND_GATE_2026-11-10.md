# 実buildのL2-bを回帰試験に載せた（§4.39 段3）

§4.39の段3です。段2で`l2b.claimed=true`が立ったことを**一度測っただけ**では、次に検査を足したときに黙って止まります（§4.38の3番目）。本記録は、**その鎖を毎回通る試験に載せた**ことを残します。

## 足したもの

| 対象 | 内容 |
| --- | --- |
| `cmd/tei/cli_self_build_l2b_go_command_test.go` | 宣言→catalog→計画→実build→五条件→`l2b.claimed`までを一本で通す |
| `cmd/tei/testdata/l2b-go-command/main.go` | ビルド対象。**機構を見るのが目的なので、内部ツリー全体をビルドしない**小さなcommand |
| `reference-go-build-from-plan` | `definitionBytesDigest`を申告するようにした（§4.36のentry側照合をこのbuilderでも効かせる） |

## 試験が確かめること

| 項目 | 期待 |
| --- | --- |
| `planReceipt.state` | `resolved`（entryがin-processで導いた） |
| `planCrossCheck.state` | `matched` |
| `planArtifactBinding.state` | `bound` |
| `artifactReproducibility.state` | `reproduced` |
| `definitionBinding.state` | **`bound`**（builderが申告したdefinitionをentryが計画と比較） |
| `l2b.claimed` | **`true`** |
| 成果物 | 存在する |

**`definitionBinding`を`bound`にしたのは段3の副産物です。** 段2では`unperformed`で、§4.36のentry側照合がこのbuilderでは効いていませんでした。

## 実測

| 試験 | 結果 | 時間 |
| --- | --- | --- |
| `TestSelfBuildEstablishesL2BFromARealGoBuild` | **PASS** | **66.8秒** |
| 同（`-short`） | **SKIP** | 0.0秒 |
| `go test ./cmd/tei -run 'SelfBuild\|Reproducibility\|AssessL2B\|DefinitionBinding'` | **PASS** | 160.5秒 |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## なぜ小さいcommandを対象にしたか

**`./cmd/tei`を対象にすると101.6秒かかりました**（内部ツリー全体のコンパイルが二回）。**機構は同じ**なので、対象を小さなcommandへ変えて66.8秒にしました。`./cmd/tei`そのものの観測は[段2の記録](TEI_GO_SELF_BUILD_L2B_FROM_REAL_GO_BUILD_2026-11-10.md)に残っています。

**`-short`では飛びます。** 実buildを二回することが観測そのものなので、短縮は「ゲートを承知で捨てる」選択です。

## 残っている穴

| 穴 | 内容 |
| --- | --- |
| TOCTOU | 照合とcompileの間にファイルが変わり得る（§4.39の案B） |
| 宣言モデル | commandのビルドに`plugin-definition`を要求する形は意味として怪しい |
| 計画に載っていないsource | 読んでいないことを示していない（`go list -deps`との突き合わせは案のまま） |
| 対象 | 回帰試験の対象は小さなcommandで、`./cmd/tei`ではない |

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet ./cmd/tei`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.39
- 段2：[`TEI_GO_SELF_BUILD_L2B_FROM_REAL_GO_BUILD_2026-11-10.md`](TEI_GO_SELF_BUILD_L2B_FROM_REAL_GO_BUILD_2026-11-10.md)
