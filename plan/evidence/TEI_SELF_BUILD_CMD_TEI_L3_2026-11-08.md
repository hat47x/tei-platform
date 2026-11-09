# `tei self-build` が駆動する stage0→stage1→stage2 の実測（SB-C1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) の**SB-C1**を、`cmd/tei`の薄い入口（[#3433](https://github.com/hat47x/tei-platform/pull/3433) の`tei self-build`）で実際に走らせた記録です。設計は[SB-C0の入口契約](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.6・§3.3に対応します。

**対象は`cmd/tei`自身です。** 既存の[`reference-self-build-two-stage`](../../tools/reference-self-build-two-stage/README.md)は対象がdriver自身でしたが、本記録は**TEI CLIの実バイナリがstage1の生成物としてstage2を駆動した**ことを示します。

## 段の構成

| stage | 何が動くか |
| --- | --- |
| stage0 | 外部Go seedで`cmd/tei`を構築（driven processではない） |
| stage1 | **stage0のTEI**が`tei self-build`でchildを起動し、childが`cmd/tei`を再構築（stage1 artifact） |
| stage2 | **stage1のartifact自身**をinvokerとして起動し、同じchildで`cmd/tei`を再構築（stage2 artifact） |

childは`tools/reference-go-build-with-supply`のビルド済み実行ファイルである。`tei self-build`はchildを起動してinvoker/invokedを記録するだけで、Build Planの解決はchildが行う。

## 観測条件

| 項目 | 値 |
| --- | --- |
| checkout | `D:\GIT\hat47x\tei-wt-l3`（main `55a48aba6`、`#3433`の統合後） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`（`/usr/local/go/bin/go`） |
| 供給 | `~/go/pkg/mod/cache/download` の複製（582MB）。`-supply`へ`cache/download`を指す |
| 対象package | `./cmd/tei` |
| 固定設定 | child側の`BuildPackage`と同じdeny-by-default（`GOPROXY=file://<供給>`、試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`） |
| stage0 | `/usr/local/go/bin/go build -trimpath -o /tmp/l3-stage0-tei ./cmd/tei` |
| 独立state | stage1は`/tmp/l3-st1-child`、stage2は`/tmp/l3-st2-child`（共有しない） |

**供給ディレクトリの指定に注意が要る。** 当初`/tmp/l3-supply`を丸ごと複製して`-supply /tmp/l3-supply`と指定したところ、childのBuildは`reading file:///tmp/l3-supply/github.com/jackc/pgx/v5/@v/v5.9.2.zip: no such file or directory`で失敗した。`GOPROXY=file://`は指定ディレクトリを**そのままmodule proxyのrootとして読む**ため、`cache/download`を指す必要がある。これは実測で判明した手順上の誤りであり、`-supply /tmp/l3-supply/cache/download`へ直すとBuildは成功した。

## 観測結果

| 欄 | 値 |
| --- | --- |
| stage0 TEI digest | `sha256:e6afc83f737e3607c0dccce7f3f5b090c335bac2060a3382c59e3be11ee19e01` |
| stage1 invoker digest | 同上（stage0のTEI） |
| stage1 artifact digest | `sha256:47b01ebfc8e2ea1bc142c198ca424fd4f38cb65d1df44ab2e93955dac5870d46` |
| **stage2 invoker digest** | `sha256:47b01ebfc8e2ea1bc142c198ca424fd4f38cb65d1df44ab2e93955dac5870d46`（**stage1のartifact**） |
| stage2 artifact digest | `sha256:47b01ebfc8e2ea1bc142c198ca424fd4f38cb65d1df44ab2e93955dac5870d46` |
| child executable digest（両段） | `sha256:9f7cf994dcc5f3f30b496b43f0b58576ad234ebee1bdfd50b670283ea0256819` |
| `stage1ArtifactIsStage2Invoker` | `true` |
| `artifactsEqual` | `true` |
| `sameChild` | `true` |
| stage1 / stage2 のartifact照合 | 両方`true`（childの主張とディスク実体が一致） |
| stage1 / stage2 のchild exit | 両方`0` |
| 両段の`l3Claimed` | **`false`**（entry自身は常にfalse。L3の判定は下記で別に行う） |

stage2の`tei self-build`出力では、`invokerPath`が`/tmp/l3-stage1-tei`、`invoked.argv[0]`が`/tmp/child-build-with-supply`である。**駆動主体（stage1の生成物）と被駆動主体（child）が別の実行ファイルとして記録されている。**

## 三つの判定を分けて読む

| 判定 | 値 | 意味 |
| --- | --- | --- |
| invoker一致 | `true` | stage2を駆動した実行ファイルがstage1の生成物そのもの |
| 同一レシピ | `true` | 両段が同じchild・同じ対象package・同じ供給 |
| artifact一致 | `true` | stage1とstage2の生成物byteが一致 |

三つが揃ったので、この対象について**L3の形**が成立している。ただし`tei self-build`の`l3Claimed`は常にfalseであり、この判定は**この記録の中で三つの値を突き合わせた結果**であって、entryが資格を発行したのではない。

## 主張しないこと

- **Seed Trust**：stage0のGoは明示した外部seedである。TEI自身によるGoコンパイラのself-hostingではない。
- **受入・Runtime Authority**：artifactのdigestが一致しても、受理済みでも配備可能でもない。
- **供給元のTrust・署名**：`go.sum`と実byteの一致はSB-0b（#3142、統合済み）が担う。本記録はそれを前提にしている。
- **一般化**：1台・1 toolchain・1 childレシピでの観測である。Windowsでの同型実行、別Go版、別供給は未確認。
- **二回の独立試行**：stage1とstage2は同じ供給と同じchildを使うため、「同じSourceを二つの独立環境で構築した」ことの証明ではない。別Attempt性は入力の同一性で担保されていない。
- **`cmd/tei`の既定経路の閉包資格**：`sourceClosureState`は依然`rejected`である（[X1](TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)）。本記録はartifactと駆動の話であり、閉包資格を動かしていない。

## 残差

| 必要なもの | 現状 |
| --- | --- |
| `cmd/tei`のBuild入口 | 済（`tei self-build`、#3433） |
| invoker/invokedの区別と記録 | 済 |
| Stage1の生成物をinvokerに据える | **済（この記録）** |
| Stage1/2のartifact比較 | 済（三判定で別々に読む） |
| `§4`の外形（`-intent`・`-supply`・`-state`・`-output`・`qualified`） | **未**。入口はchild起動の記録までで、Build Intentの解釈をしない |
| Seed Trust | 未 |
| 既定経路の閉包資格 | 未（`rejected`のまま） |

## 実行した検証

- `go build -trimpath -o /tmp/l3-stage0-tei ./cmd/tei`：PASS
- `go build -trimpath -o /tmp/child-build-with-supply ./tools/reference-go-build-with-supply`：PASS
- 上記の三段実行：stage1・stage2ともchild exit 0、artifact照合一致
- `sha256sum`による両段のartifact digestとinvoker digestの突合：上表のとおり

**NOT RUN**：Windows実行、別Go版、別供給、二回の独立環境での比較、CI、trusted full checkout。

## 参照

- 入口の実装：[`cmd/tei/cli_self_build.go`](../../cmd/tei/cli_self_build.go)、試験は[`cli_self_build_test.go`](../../cmd/tei/cli_self_build_test.go)
- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §3.3・§4.6
- child：[`tools/reference-go-build-with-supply`](../../tools/reference-go-build-with-supply/README.md)
- 台帳：[`TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md`](TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md)
