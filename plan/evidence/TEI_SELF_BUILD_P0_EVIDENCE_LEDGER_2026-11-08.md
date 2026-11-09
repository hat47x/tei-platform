# セルフビルドP0の証拠台帳（SB-0a〜SB-C1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のP0順序（SB-0a → SB-0b → SB-A → SB-B → SB-C）のうち、2026-11-07にmainへ統合された証拠と実装を、**Claim別に一か所へ並べた索引**です。

**これは索引であって、新しい主張ではありません。** 各項目の主張と限界は元の記録が定めます。本稿は「どこまでが統合済みで、どこが未着手か」を読み取り、次の作業を選ぶために置きます。

**L3の形は二つの対象で成立しています**（§2.2）。`reference-go-build-stage-driver`自身と、`tei self-build`を使った`cmd/tei`自身です。ただし**どちらも「L2（Build Intentから実Buildへの結線）」の成立ではありません**（§2.3）。**L2はさらに二層（受け渡し・解釈）に分かれ、両方とも未成立です**（§2.5）。

- 計画の正本：`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md` §35.2・§35.3
- 判定の正本：[`docs/status.md`](../../docs/status.md)
- Gate：[#1119](https://github.com/hat47x/tei-platform/issues/1119)。**G4は`UNQUALIFIED`のまま**

## 1. Claim別の台帳

「Claim」は§35.3のL1/L2/L3と、§35.2のSB段階を混ぜないための区分です。**同じ行の証拠が別の行のClaimを支えることはありません。**

| 段階 | Claim（何を言えるか） | 証拠（mainの統合commit） | 実装 | 状態 |
| --- | --- | --- | --- | --- |
| SB-0a | `cmd/tei`の選択Module閉包は、隔離観測では`missing-module`。自己完結toolは`supported` | [`TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md`](../evidence/TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)（`c81659678`） | `gobuilder`の五分類と`tools/reference-go-module-closure-preflight` | 統合済み |
| SB-0b（照合） | 実選択12件は`go.sum`の`h1`と一致し、供給だけでオフライン再解決できる | [`TEI_GO_MODULE_SUPPLY_VERIFICATION_2026-11-07.md`](../evidence/TEI_GO_MODULE_SUPPLY_VERIFICATION_2026-11-07.md)（`8b7edf867`） | `tools/reference-go-module-supply-verify` | 統合済み |
| SB-0b（供給の実測） | 供給の有無で分類が変わる。**成果物digestの一致は供給の同一性を証明しない** | [`TEI_GO_MODULE_SUPPLY_OBSERVATION_2026-11-07.md`](../evidence/TEI_GO_MODULE_SUPPLY_OBSERVATION_2026-11-07.md)（`cbfbf5dc9`） | 同上 | 統合済み |
| SB-0b（破損の分類） | archive破損は`corrupt-supply`として供給不足と別に分類される | [`TEI_GO_MODULE_CORRUPT_SUPPLY_2026-11-07.md`](../evidence/TEI_GO_MODULE_CORRUPT_SUPPLY_2026-11-07.md)（`e3e94204f`） | `gobuilder`の`corrupt-supply` | 統合済み |
| SB-A1 | 選択入力はリポジトリ内318件・外部Module 12件・325メンバーに分かれる。**資格は常に`false`** | [`TEI_GO_SOURCE_CLOSURE_OBSERVATION_2026-11-07.md`](../evidence/TEI_GO_SOURCE_CLOSURE_OBSERVATION_2026-11-07.md)（`543db98df`） | `gobuilder`の`ObserveSourceClosure` | 統合済み |
| SB-B（X1） | `cmd/tei`は供給だけでBuildできる。同じadapterのSource閉包は`rejected` | [`TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md`](../evidence/TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)（`5ad04cbdf`） | `tools/reference-go-build-with-supply` | 統合済み |
| SB-N8 | 二つの独立試行が同じartifact byteを出す。実行主体の重複を除く | [`TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md`](../evidence/TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md)（`43765839f`） | `tools/reference-go-build-reproducibility` | 統合済み |
| SB-B（案C） | 供給を明示した呼出しだけが外部Moduleメンバーを閉包へ入れる。**既定は拒否のまま** | [`TEI_GO_EXTERNAL_MODULE_ADMISSION_OPTIN_2026-11-07.md`](../evidence/TEI_GO_EXTERNAL_MODULE_ADMISSION_OPTIN_2026-11-07.md)（`e19dccc38`） | `gobuilder`の`ResolveQualifiedSourceClosure` | 統合済み |
| SB-C0/C1（駆動） | invokerとinvokedの実行ファイルdigest・argv・exit・PIDを別々に記録し、子の主張とディスク実物のdigestが一致する。**`l3Claimed`は常に`false`** | [`TEI_GO_BUILD_STAGE_DRIVER_2026-11-07.md`](../evidence/TEI_GO_BUILD_STAGE_DRIVER_2026-11-07.md)（`04f75301f`） | `tools/reference-go-build-stage-driver` | 統合済み |
| SB-N13（自称照合） | 駆動主体が自称するdigestは実行中process imageと比較され、不一致は`failureCause`に残る。**一致しても`l3Claimed`は`false`** | [`TEI_GO_STAGE_DRIVER_INVOKER_CLAIM_2026-11-08.md`](../evidence/TEI_GO_STAGE_DRIVER_INVOKER_CLAIM_2026-11-08.md)（`26469d411`） | 同上（`-claimed-self`） | 統合済み |
| **SB-C1（L3・driver自身）** | stage1のartifactがstage2の**invoker**として実起動され、同一child・同一レシピでartifactが一致する。**対象は`reference-go-build-stage-driver`自身** | [`TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md`](../evidence/TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md)（`e039a7f9c`） | `tools/reference-self-build-two-stage` | 統合済み |
| **SB-C0（薄い入口）** | `tei self-build`がchildを起動し、invoker/invokedのdigest・argv・exit・PIDと、子の主張とディスク実物のdigestを別々に記録する。**`l3Claimed`は常に`false`** | [`cmd/tei/cli_self_build.go`](../../cmd/tei/cli_self_build.go)、[`TEI_GO_STAGE_DRIVER_INVOKER_CLAIM_2026-11-08.md`](../evidence/TEI_GO_STAGE_DRIVER_INVOKER_CLAIM_2026-11-08.md)（`26469d411`） | `cmd/tei`（#3433、`55a48aba6`） | 統合済み |
| **SB-C1（L3・`cmd/tei`自身）** | stage1のartifact自身をstage2の**invoker**として実起動し、同一child・同一レシピでartifactが一致する。**対象は`cmd/tei`** | [`TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md`](../evidence/TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md)（`bb7fc01bb`） | `tei self-build`＋`tools/reference-go-build-with-supply` | 統合済み |
| SB-C0（設計） | 再入contractの外形・拒否条件・L3に必要な追加情報 | [`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)（`62e727be5`、訂正 `d033dd011`） | — | 統合済み（設計のみ） |
| SB-C0 §4（外形） | workspace・intent・supply・state・outputの外形とSC0-01〜05の拒否条件。**`qualified`は常に`false`** | [`TEI_GO_SELF_BUILD_OUTER_SHAPE_2026-11-08.md`](TEI_GO_SELF_BUILD_OUTER_SHAPE_2026-11-08.md)、[`TEI_GO_SELF_BUILD_INTENT_AND_SUPPLY_2026-11-08.md`](TEI_GO_SELF_BUILD_INTENT_AND_SUPPLY_2026-11-08.md) | `cmd/tei`（#3463・#3476） | 統合済み |
| SB-C0 §4.7（I-2） | 宣言を解決childへ渡し、`derived-cache`のreceiptを受け取る。未知authorityは拒否、宣言済みplan fingerprintと不一致ならbuilderを起動しない | [`TEI_GO_SELF_BUILD_INTENT_RESOLUTION_2026-11-09.md`](TEI_GO_SELF_BUILD_INTENT_RESOLUTION_2026-11-09.md) | `cmd/tei`（#3571） | 統合済み |
| SB-C0 §4.8（計画突合） | 解決childとbuilder childのplan fingerprintを比べ、`matched`／`mismatched`／`unperformed`を区別する。**不一致は事後の発見**として`ok=false` | [`TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_2026-11-09.md)、[`TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_ACTUAL_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_ACTUAL_2026-11-09.md)（`3a4ddeba6`） | `cmd/tei`の`planCrossCheck` | 統合済み |
| SB-C0 §4.8（L2-a再観測） | 別の作業領域・別builderで、同じ宣言から同じ`planFingerprint`が出ることを再確認。**`l2Claimed`は`false`** | [`TEI_GO_SELF_BUILD_L2A_REOBSERVATION_2026-11-09.md`](TEI_GO_SELF_BUILD_L2A_REOBSERVATION_2026-11-09.md)（`04ea1470b`） | `tei self-build` | 統合済み |
| SB-C0（internal規則） | `cmd/tei`から`tools/internal/gobuilder`はimportできない。移動なら29ファイルに触れる | [`TEI_GO_INTERNAL_RULE_AND_BUILDER_MOVE_2026-11-07.md`](../evidence/TEI_GO_INTERNAL_RULE_AND_BUILDER_MOVE_2026-11-07.md)（`4519e1ff1`） | — | 統合済み（観測のみ） |
| SB-B（方針の判断材料） | 外部Moduleを閉包へ入れる三案と、判断に必要な未取得の観測 | [`EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md`](../design/EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md)（`392af6ea9`） | — | 統合済み（材料のみ） |

## 2. 台帳から読み取れること

### 2.1 成立しているもの（L1の範囲）

`cmd/tei`について、次は**実測**で成立している。

- 供給と`go.sum`一致だけで実Buildできる（X1）
- 二回の独立試行で成果物byteが一致する（SB-N8）
- 選択入力の全体像（リポジトリ内318件・外部325メンバー）を列挙できる（SB-A1）
- 子プロセスとしてBuildを駆動し、invoker/invokedを区別して記録できる（SB-C0/C1）
- 駆動主体の自称digestを実行中imageと照合できる（SB-N13）

これは§35.3の**L1（reproducible Go build）**に相当する。ただし**同じマシン・同じtoolchain・同じ供給での一致**であり、別OS・別toolchainへの一般化はしていない。

### 2.2 成立しているもの（L3の範囲）

L3の形は、**二つの対象**で成立している。どちらも同じ三判定（invoker一致・同一レシピ・artifact一致）を別々に読んだ結果であり、`tei self-build`の`l3Claimed`は常にfalseのままである。

#### 2.2.1 `reference-go-build-stage-driver`自身

| 判定 | 値 |
| --- | --- |
| `stage1ArtifactIsStage2Invoker` | `true`（stage2のinvoker＝stage1のartifact） |
| `sameChildAndArguments` | `true` |
| `artifactsEqual` | `true` |
| `l3Claimed` | `true` |

記録：[`TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md`](../evidence/TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md)（`e039a7f9c`）、実装は`tools/reference-self-build-two-stage`。

#### 2.2.2 `cmd/tei`自身

`tei self-build`（[#3433](https://github.com/hat47x/tei-platform/pull/3433)）で、**stage1のartifact自身をstage2のinvokerとして実起動**した。

| 欄 | 値 |
| --- | --- |
| stage0 TEI digest | `sha256:e6afc83f…` |
| stage1 artifact digest | `sha256:47b01ebf…` |
| **stage2 invoker digest** | **`sha256:47b01ebf…`（stage1のartifact）** |
| stage2 artifact digest | `sha256:47b01ebf…` |
| child executable digest（両段） | `sha256:9f7cf994…` |

| 判定 | 値 |
| --- | --- |
| invoker一致 | `true` |
| 同一レシピ | `true` |
| artifact一致 | `true` |

記録：[`TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md`](../evidence/TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md)（`bb7fc01bb`）、入口は[`cmd/tei/cli_self_build.go`](../../cmd/tei/cli_self_build.go)。

**これは「Build Intentから実Buildへ結線した」L2の成立ではない。** 入口はchildを起動してinvoker/invokedを記録するだけで、Build Planの解決もBuild Intentの解釈もしない（SB-C0 §4.6）。

### 2.3 成立していないもの

- **既定経路の閉包資格**：`ResolveSourceClosure`は`rejected`のまま。opt-in経路（案C）で閉包を受けられるが、既定は変わっていない
- **`go.sum`のTrust根拠**：誰がいつ`go.sum`を受理したかはGit履歴でしかない
- **L2（TEI-defined build）**：Build Intent/Graphから実Buildへの結線は無い。入口は宣言を読んで解決childへ渡し、計画の突合まで記録するが、**「builderがその宣言からその計画を導いた」ことも「artifactがその計画の出力である」ことも言えない**（§2.5）
- **Seed Trust**：stage0のGoは外部seedである。TEI自身によるGoコンパイラのself-hostingではない
- **二回の独立環境での比較**：L3の二つの実測は、どちらも同じ供給と同じchildを使う。独立した環境の比較ではない

### 2.5 L2を二層に分けた現在地

SB-C0 §4.8でL2を二層に切り分けた。**両層とも未成立**だが、記録できる材料の量が違う。

| 層 | 何を言えるか | 現状 | 根拠 |
| --- | --- | --- | --- |
| **L2-a（受け渡し）** | この`cmd/tei`が指定childへ**特定の宣言**を渡した。渡したbyteのdigestと、childの出力を受けたことを記録できる | **材料あり**（未成立のClaimとしては扱わない） | [`TEI_GO_SELF_BUILD_L2A_REOBSERVATION_2026-11-09.md`](TEI_GO_SELF_BUILD_L2A_REOBSERVATION_2026-11-09.md) |
| **L2-b（解釈）** | childが宣言を読み確定Build Planを導出し照合値を返した | **未**（計画突合はP-1で実測`matched`になったが、(a)builderが読んだ宣言byteの申告と(c)artifactと計画の結び付けが未） | [`TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md)・[`TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_ACTUAL_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_ACTUAL_2026-11-09.md) |

**「二つの自己申告が一致した」はL2-bではない。** 一致を言うには、少なくとも(a)builderが読んだ宣言のbyteをbuilder自身が申告し、(b)それがentryの保持したbyteと一致し、(c)artifactと計画の結び付けが別の観測で示される必要がある。(a)は`-child-arg`で報告引数を渡せば記録できるが、`reference-go-build-with-supply`は報告しないため実測では`unperformed`になる。**P-1のchild（[`reference-plugin-package-build-child`](../../tools/reference-plugin-package-build-child/README.md)）を使うと計画突合は`matched`になるが、それでも(a)と(c)は埋まらない。** `matched`を`(c)`の代わりに読ませない。

### 2.4 残差

L3の**形**は二つの対象で成立した。残るのは次の三点である。

| 必要なもの | 現状 | 次の作業 |
| --- | --- | --- |
| invoker/invokedの区別と記録 | 済 | — |
| 子の主張とディスク実物の照合 | 済 | — |
| 自称digestと実行imageの照合 | 済（SB-N13） | — |
| Stage1の生成物をinvokerに据える | 済（driver自身と`cmd/tei`の両方） | — |
| **SB-C0 §4の外形**（`-intent`・`-supply`・`-state`・`-output`・`qualified`） | 済（[#3463](https://github.com/hat47x/tei-platform/pull/3463)、[#3476](https://github.com/hat47x/tei-platform/pull/3476)）。Intentは宣言のまま（`resolved: false`）、供給はexactな検証器へ委譲 | 未決は§4の項目を参照 |
| **SB-C0 §4.7・§4.8**（解決の委譲と計画突合） | 済（[#3571](https://github.com/hat47x/tei-platform/pull/3571)・[#3584](https://github.com/hat47x/tei-platform/pull/3584)・[#3610](https://github.com/hat47x/tei-platform/pull/3610)。P-1のchildで`planCrossCheck=matched`を実プロセスで観測、[記録](TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md)） | L2-bは別のClaim。`(c)`が未 |
| Seed Trustと受入 | 未 | 別のClaim |
| 別OS・別toolchainでの再観測 | **OS軸は済**（Windows／Linuxで計画側の値が一致、[記録](TEI_GO_SELF_BUILD_CROSS_OS_2026-11-09.md)）。toolchain軸と二回試行の比較は未 | 独立した比較 |

**「L3の形が成立した」と「L2が成立した」を混同しない。** 前者は駆動主体の同一性であり、後者は意味からBuildへの結線である。**L2をさらに二層に分けた現在地は§2.5が正本である。**

## 3. この台帳の使い方

- 各段階の「状態」が`統合済み`でないものは、**Claimとして引用しない**
- 「実装」列は、そのClaimを支えるコードの置き場所である。**コードがあることはClaimの成立ではない**（`qualified`・`l3Claimed`が常にfalseであることと同じ理由）
- Claimを追加するときは、この台帳の行を増やすのではなく、元のEvidenceを更新して行の参照先を変える

## 4. 未決事項

- ~~SB-C0 §4の外形の実装~~ → **済**（[`TEI_GO_SELF_BUILD_OUTER_SHAPE_2026-11-08.md`](TEI_GO_SELF_BUILD_OUTER_SHAPE_2026-11-08.md)、[`TEI_GO_SELF_BUILD_INTENT_AND_SUPPLY_2026-11-08.md`](TEI_GO_SELF_BUILD_INTENT_AND_SUPPLY_2026-11-08.md)）。`-intent-file`の宣言検査と`-supply-verifier`の分類は起動前条件であり、`qualified`は常に`false`
- **L2（Build Intentから実Buildへの結線）は依然として未**。入口は宣言を読んでdigestを取り、解決childへ渡してreceiptと計画突合を記録するが、`resolved: false`のまま。供給の分類も複写せずexactな検証器へ委譲する。**宣言が読めたこと・供給が照合できたこと・二つの自己申告が一致したことは、L2の成立ではない**（§2.5）
- Seed Trustの扱い（stage0のGoは外部seedであり、TEI自身によるGoコンパイラのself-hostingではない）
- `go.sum`のTrust根拠
- 別OS・別toolchainでの再観測、および二回の独立環境での比較
- この台帳の更新責任（#2831の横断担当か、各段階のownerか）

## 5. 実行した検証

- 各Evidenceの統合commitを`git log -1 --format='%h' origin/main -- <path>`で確認
- 各実装ディレクトリの存在を`ls`で確認
- `go test ./tools/repository-plan-check -count=1`：PASS（本稿の相対リンクを含む）

**NOT RUN**：本稿は索引であり、記載した各Claimの再実行はしていない。

## 参照

- 計画：[`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) §35.2・§35.3
- 設計：[`GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md`](../design/GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md)、[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 判定の正本：[`docs/status.md`](../../docs/status.md)
