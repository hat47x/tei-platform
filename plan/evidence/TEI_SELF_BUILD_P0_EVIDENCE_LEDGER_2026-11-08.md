# セルフビルドP0の証拠台帳（SB-0a〜SB-C1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のP0順序（SB-0a → SB-0b → SB-A → SB-B → SB-C）のうち、2026-11-07にmainへ統合された証拠と実装を、**Claim別に一か所へ並べた索引**です。

**これは索引であって、新しい主張ではありません。** 各項目の主張と限界は元の記録が定めます。本稿は「どこまでが統合済みで、どこが未着手か」を読み取り、次の作業を選ぶために置きます。

**L3の成立は対象が限定されています。** `reference-go-build-stage-driver`自身についてL3の形が成立しましたが、`cmd/tei`のL3ではありません（§2.2）。

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
| **SB-C1（L3・driver自身）** | stage1のartifactがstage2の**invoker**として実起動され、同一child・同一レシピでartifactが一致する。**対象は`reference-go-build-stage-driver`自身** | [`TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md`](../evidence/TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md)（`e039a7f9c`） | `tools/reference-self-build-two-stage` | 統合済み（**対象限定**） |
| SB-C0（設計） | 再入contractの外形・拒否条件・L3に必要な追加情報 | [`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)（`62e727be5`、訂正 `d033dd011`） | — | 統合済み（設計のみ） |
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

### 2.2 成立しているもの（L3の範囲・対象限定）

**`reference-go-build-stage-driver`自身**を対象に、L3の形が成立している（SB-C1）。

| 判定 | 値 |
| --- | --- |
| `stage1ArtifactIsStage2Invoker` | `true`（stage2のinvoker＝stage1のartifact） |
| `sameChildAndArguments` | `true` |
| `artifactsEqual` | `true` |
| `l3Claimed` | `true` |

**これは`cmd/tei`のL3ではない。** 対象は「自分自身を再構築できるdriver」であり、`cmd/tei`はBuild入口を持たない（SB-C0 §2）。同じ形を`cmd/tei`へ移すには入口の決定が先である。

### 2.3 成立していないもの

- **既定経路の閉包資格**：`ResolveSourceClosure`は`rejected`のまま。opt-in経路（案C）で閉包を受けられるが、既定は変わっていない
- **`go.sum`のTrust根拠**：誰がいつ`go.sum`を受理したかはGit履歴でしかない
- **L2（TEI-defined build）**：Build Intent/Graphから実Buildへの結線は無い。`cmd/tei`に汎用Build入口が無いことを実測した
- **`cmd/tei`のL3**：入口が無いため、同じ形を`cmd/tei`へ当てる実験は未実施
- **Seed Trust**：stage0のGoは外部seedである。TEI自身によるGoコンパイラのself-hostingではない

### 2.4 残差

L3の**形**はdriver自身について成立した。`cmd/tei`へ広げるために残るのは次の三点である。

| 必要なもの | 現状 | 次の作業 |
| --- | --- | --- |
| invoker/invokedの区別と記録 | 済 | — |
| 子の主張とディスク実物の照合 | 済 | — |
| 自称digestと実行imageの照合 | 済（SB-N13） | — |
| Stage1の生成物をinvokerに据える | 済（driver自身について） | — |
| **`cmd/tei`のBuild入口** | **未**（設計のみ） | A-1（`gobuilder`移動）か、案B（`cmd/tei`が既存toolをsubprocess起動） |
| `cmd/tei`を対象にしたL3 | 未 | 入口の決定後、同じ`reference-self-build-two-stage`の形を当てる |
| Seed Trustと受入 | 未 | 別のClaim |

**「driver自身のL3」と「TEI本体のL3」を混同しない。** 前者は対象を限定した成立であり、後者は入口の決定を待つ。

## 3. この台帳の使い方

- 各段階の「状態」が`統合済み`でないものは、**Claimとして引用しない**
- 「実装」列は、そのClaimを支えるコードの置き場所である。**コードがあることはClaimの成立ではない**（`qualified`・`l3Claimed`が常にfalseであることと同じ理由）
- Claimを追加するときは、この台帳の行を増やすのではなく、元のEvidenceを更新して行の参照先を変える

## 4. 未決事項

- **`cmd/tei`にBuild入口を足すか**（A-1の`gobuilder`移動か、案Bのsubprocessか）。driver自身のL3は成立したが、`cmd/tei`は入口を持たないため対象を広げられない
- `cmd/tei`を対象にしたL3の実験手順（入口の決定後に`reference-self-build-two-stage`の形を当てる）
- Seed Trustの扱い（stage0のGoは外部seedであり、TEI自身によるGoコンパイラのself-hostingではない）
- `go.sum`のTrust根拠
- 別OS・別toolchainでの再観測
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
