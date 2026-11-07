# セルフビルドの再入（SB-C0）の入口契約（設計候補）

- 対象：[#2831](https://github.com/hat47x/tei-platform/issues/2831) の **SB-C0**（再入contract）、§35.3のL1/L2/L3区分
- 関連：[#943](https://github.com/hat47x/tei-platform/issues/943)、[#979](https://github.com/hat47x/tei-platform/issues/979)、[#1080](https://github.com/hat47x/tei-platform/issues/1080)、[#1090](https://github.com/hat47x/tei-platform/issues/1090)、[#1097](https://github.com/hat47x/tei-platform/issues/1097)、[#1103](https://github.com/hat47x/tei-platform/issues/1103)、[#1121](https://github.com/hat47x/tei-platform/issues/1121)
- 作成日：2026-11-07
- 状態：**設計候補。採用・実装・qualified PASSではない**
- 前段：[SB-0bの供給境界](GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md)、[X1の記録](../evidence/TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)、[再現性の記録](../evidence/TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md)

## 1. この文書が解く問題

SB-0a〜SB-Bの実測で、次が分かっている。

- `cmd/tei`は供給と`go.sum`一致だけで実Buildできる（X1）
- 二回の独立した試行で成果物byteが一致する（SB-N8）
- しかし`ResolveSourceClosure`は`rejected`のままである

足りないのは「Buildできる」ことの先、**Stage1で作ったTEI自身がStage2のBuildを駆動したと言えるか**である。§35.3はこれをL3と呼び、L1（Go toolchainで実artifact）・L2（TEI標準構造のBuild Intentで宣言）から分けている。

本稿が決めるのはL3の手前、**SB-C0（再入contract）**だけである。すなわち「Stage1の生成物が、Stage2のBuild Plan解決と既存adapterへの実行要求を、実際に自分のプロセスで行える入口が、既存経路のどこにあるか」。

## 2. 現行mainの静的観測（2026-11-07）

実装の有無を推測せず、import関係とCLI広告から確認した。

| 観測 | 結果 | 意味 |
| --- | --- | --- |
| `cmd/tei`から`tools/internal/gobuilder`へのimport | **0件** | CLIはGo build adapterを持たない |
| `internal/`から`tools/`へのimport | **0件**（`internal/operations/dependency_test.go`が禁止を固定） | Operation境界はtransport/tooling非依存を保つ |
| `cmd/tei`の`cli_contract.go`が広告するコマンド | read-only 19件（`version`〜`plugin-runtime-plan`） | `build`/`self-build`は無い |
| `os/exec`を持つ実装 | `internal/plugin/runtime/process_stdio.go`（Plugin実行）、`internal/query/analyzer/client.go`（Query基盤） | **Process起動の既存機構はある**。ただしBuild用ではない |
| Buildの汎用入口 | `pluginruntime.BuildPlan`（Plugin RuntimeのEnvironment Plan） | **Plugin前提**であり、非PluginのBuild Planではない |
| 非Plugin Buildの参照実装 | `tools/`側に5件（`reference-go-build-with-supply`、`reference-go-build-reproducibility`、`reference-go-module-supply-verify`、`reference-go-module-closure-preflight`、`reference-build-execution-binding`） | すべて独立したCLI。`cmd/tei`からは呼ばれない |

**結論（観測であって不存在の証明ではない）**：`cmd/tei`自身に汎用Build入口は無い。あるのは`tools/`配下の独立CLIで、それらは`go run`/`go build`で起動する外部プロセスである。

## 3. 三つの候補と、選ばない理由

再入contractを成立させる方法は三つある。**本稿は第一候補を推すが、採用はしない。**

| 案 | 内容 | 長所 | 短所 |
| --- | --- | --- | --- |
| **A: 薄いCLI接続** | `cmd/tei`へ`build`サブコマンドを1個足し、`tools/internal/gobuilder`を呼ぶ | Stage1バイナリが自分でStage2を起動できる（L3の形に最も近い） | `internal/`→`tools/`のimportを新設するか、`gobuilder`を`internal/`へ移す必要がある。後者はADR-0033の配置判断に触れる |
| B: 既存toolのSubprocess起動 | Stage1が`tools/reference-go-build-with-supply`を子プロセスとして起動 | 既存の`os/exec`機構（`StdioProcessLauncher`）を再利用できる。`internal/`の依存境界を変えない | Build主体が**別のバイナリ**になる。L3の「Stage1生成TEI自身が駆動した」の証明には、子プロセスのexecutable digestを別途固定する必要がある |
| C: Plugin経路へ載せる | BuildをPlugin Capabilityとして宣言し、`pluginruntime`から実行 | 既存のLock/Authority/実行機構をそのまま使える | 非PluginのBuild subjectをPluginへ偽装することになる。§35.2が明示的に禁じている |

**Aを第一候補にする理由**：L3の定義は「stage1で出力されたTEI自身のバイナリがstage2のBuild Plan解決・実Build起動を駆動」であり、Bはこの主語を別バイナリへ移す。Cは`cmd/tei`をPluginとして偽装する。

### 3.1 訂正（2026-11-07）：AはGoのinternal規則に抵触する

当初の§3は「`cmd/tei`から`tools/internal/gobuilder`をimportするのは既存の規則に抵触しない」と書いた。**これは誤りである。** 実測で確かめた。

`tei/tools/internal/gobuilder`は`tei/tools/`配下の`internal`であり、Goのinternal規則により**`tei/tools/`の外からはimportできない**。`cmd/tei`は`tei/cmd/`にあり、外側である。

~~~text
$ mkdir -p .tmp_probe/cmd && cat > .tmp_probe/cmd/probe.go  (import "tei/tools/internal/gobuilder")
$ go vet ./.tmp_probe/cmd
package tei/.tmp_probe/cmd
	.tmp_probe/cmd/probe.go:4:2: use of internal package tei/tools/internal/gobuilder not allowed
~~~

この検査は`tei-wt-intcheck`のworktree（main `96d276b74`）、Go 1.27.3 linux/amd64で実行し、probeは削除した。interpreterの判定はコンパイル時のパス規則であり、`internal/operations/dependency_test.go`の禁止とは**別の層**にある。

したがってAの前段は二択になる。

| Aの変種 | 内容 | 影響 |
| --- | --- | --- |
| A-1 | `gobuilder`をモジュール直下の`internal/`配下へ移す | 既存15ファイルのimport pathが変わる。`tools/`の側から新pathをimportできる（module root 配下なので、`tools/`からも`cmd/`からも届く） |
| A-2 | `gobuilder`の必要な関数だけを新しい`internal/`パッケージへ写し、`tools/`側は既存のまま残す | **同じ実装が二つになる**。§35.2が禁じた「第二Runner」に近づく |
| A-3 | `internal/`側に薄いwrapperを置き、`go`コマンドを直接execする | `ResolveSourceClosure`等の既存検査を再実装することになる |

**A-1を本稿の推奨にする。** 移動は一度きりで、import pathの変更は機械的に追える。A-2は二重実装、A-3は検査の再実装であり、どちらも「一つのadapterを共有する」という前提を壊す。

ただしA-1は`tools/internal/gobuilder`の**15ファイルすべてのimport pathを変える**ため、既存Evidence（`plan/evidence/TEI_GO_*`）が引用するコマンドの実行手順も変わる。移動そのものを本稿で実施せず、影響範囲を別途洗う必要がある。移動は[ADR-0033](../adr/0033-core-and-plugin-subprojects.md)の配置判断に触れるため、**採用はしない**。

### 3.2 再評価：Bは「別バイナリ」ではなく「別プロセス」の問題になる

A-1のコストを踏まえると、B（`tools/reference-go-build-with-supply`のsubprocess起動）の短所も読み直す必要がある。

- BではBuild主体が別バイナリになるが、L3の定義は「Stage1で生成したTEI自身のProcessが、Stage2のBuild Plan解決と既存adapterへの実行要求を駆動した」ことである。**subprocessを起動するのがStage1のTEIなら、駆動主体はStage1のまま**である
- 証明に必要なのは「Stage1のTEIが、どのchild executableを、どのargvで起動したか」の記録であり、child自身のdigestである。これは`StdioProcessLauncher`で既に取得できる形に近い
- 逆にA-1では、Stage1のTEIが**同じプロセス内で**Buildを実行する。駆動と実行が同一プロセスになり、区別が弱くなる

**したがってAとBの優劣は「どちらがL3に近いか」では決まらない。** Aはプロセス内、Bはプロセス間で、L3の証拠の取り方が違う。本稿の第一候補は**Bへ変更する**。理由は、A-1の移動コストが大きく、Bは既存の依存境界を変えずに駆動主体の記録まで取れるためである。

ただしBにも制約がある。Stage1のTEIがchildを起動するには、既存の`StdioProcessLauncher`を**そのまま使えるとは限らない**（Plugin用のwire protocolを前提にしている）。実際に使えるかは未確認であり、§6の未決事項に移す。

## 4. SB-C0の最小契約（案）

`cmd/tei`へ足す1コマンドの外形を固定する。名称は提案であり、確定ではない。

~~~text
tei self-build
  -workspace <path>           対象Workspace（Build Intentの宣言元）
  -intent <id>                Build Intent ID（§35.3のL2が要求する宣言）
  -supply <dir>               外部Module供給（SB-0bの照合済み供給）
  -state <dir>                試行ごとの隔離状態（必須。既定値なし）
  -output <path>              成果物の出力先
  -json                       機械可読出力
~~~

### 4.1 効果の分類

| 分類 | 値 | 理由 |
| --- | --- | --- |
| Effect | `writes-artifact` | ファイルを書く。ただし**RepositoryとWorkspaceは変更しない** |
| 必要な権限 | 出力先への書込み | 既存のAccess境界（`internal/access`）を再利用する |
| 広告 | `cliCommandCatalog`へ追加 | discoveryの既存機構（`Modes`/`SuccessShape`）に合わせる |

### 4.2 拒否条件（Buildを開始しない）

1. `-workspace`が未指定、または存在しない
2. `-intent`が解決できない（Build Intentが宣言されていない）
3. `-supply`が指定され、かつSB-0bの照合が通らない（`corrupt-supply`を含む）
4. `-state`が未指定、または既存の試行と共有されている
5. `-output`が既存ファイルを上書きする（no-clobber）
6. WorkspaceのSourceが未受理（`validate`がerrorを返す）
7. 起動するchild executableが`-builder`で明示されていない（Bの場合。PATH解決に委ねない）

7はBを選んだ場合の追加条件である。`StdioProcessLauncher`の既存方針（「Executableとargvを分けたまま渡し、shellを経由しない」）に合わせ、**PATHからの暗黙解決を許さない**。childのexecutable digestは起動前に記録する。

5は`reference-plugin-build-supply`のno-clobber規約と同じ扱いにする。上書きfallbackを持たない。

### 4.3 出力（案）

~~~text
tei.self-build/v0
  schemaVersion
  nature                  "one build attempt driven by this TEI binary; no acceptance, no runtime authority"
  intent                  : Build Intent ID と解決済みfingerprint
  plan                    : exact Build Plan fingerprint
  toolchain               : Go実行ファイルdigest、GOOS/GOARCH
  supply                  : SB-0bの照合結果（classification、module数）
  closure                 : 対象Module側digest、外部入力側digest（別値）
  artifact                : digest、byte数
  attempt                 : attemptId（乱数）、stateDir
  qualified               : **常にfalse**（§35.3のL3は別途証明する）
  boundary                : 何を主張しないかの一文
~~~

`qualified`を常にfalseにするのは、[X1の記録](../evidence/TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)と同じ理由である。artifactが在ることを閉包資格やL3の証明と読み替えない。

### 4.4 L3の証明に必要な追加情報

SB-C0の出力だけではL3を主張できない。L3には次が要る。

| 項目 | 取得元 | SB-C0での扱い |
| --- | --- | --- |
| Stage1実行ファイルのdigest | `tei version -json`の`vcsRevision`ではなく、**実行ファイル自身のdigest** | 出力へ含める。ただし`tei version`はbuild情報しか返さないため、別途`-self-digest`を記録する |
| Stage1プロセスの起動記録 | `os.Exit`前のchild PID・argv・exit | Reference側で記録。本契約では必須にしない |
| Stage2が消費したSource/Intent | `intent`・`closure` | 出力に含める |
| Stage1/2 artifactの比較 | digest | 別Attemptとして記録する |

**Stage1のdigestを自己申告で出力するだけではL3にならない**（SB-N13）。実行プロセスのimageと照合する第三者の観測が必要で、それはSB-C1の範囲である。

## 5. 追加の負例（試験仕様）

| ID | Given / 操作 | Expected |
| --- | --- | --- |
| SC0-01 | `tei self-build`を`-intent`無しで実行 | 引数エラー。Buildを開始しない |
| SC0-02 | `-supply`に破損Moduleを渡す | `corrupt-supply`で停止（SB-0bの分類を再利用） |
| SC0-03 | `-state`を二回の試行で共有 | 拒否。共有stateを「二回」と数えない |
| SC0-04 | `-output`が既存ファイル | no-clobberで拒否。上書きfallbackなし |
| SC0-05 | `validate`がerrorのWorkspace | Buildを開始しない |
| SC0-06 | 成功時の出力 | `qualified: false`。artifact成功をL3・受入・権限と表現しない |
| SC0-07 | Stage1のdigestを自己申告だけ | L3の証明にしない（SB-N13の再掲） |
| SC0-08 | `cmd/tei`から`tools/internal/`をimportする変更 | Goのinternal規則が拒否する（§3.1で実測）。規則を緩めるのではなく、`gobuilder`の移動かwrapperかを選ぶ |
| SC0-09 | Bで`-builder`を省略しPATHから解決させる | 拒否。child executableのdigestを記録できない起動を許さない |
| SC0-10 | Stage1が起動したchildのdigestと、記録したdigestが違う | L3不可。SB-N13と同じ扱い |

## 6. 未決事項

- コマンド名（`self-build` / `build` / `artifact build`）
- Build Intentの解決方式（`-intent`でID指定か、Workspace内の宣言から解決か）
- **A-1（`gobuilder`の`internal/`への移動）とB（subprocess）のどちらを採るか**。§3.1・§3.2でBを推奨へ変更したが、A-1の移動コストを測っていない
- `internal/plugin/runtime`の`StdioProcessLauncher`をBuild用child起動へ**そのまま使えるか**（Plugin用wire protocol前提であるため、未確認）
- `-self-digest`の取得方法（実行ファイル自身のdigestをどう自己参照するか）
- 出力のschemaVersionと、既存`tei.cli.*`との命名一貫性
- SB-C1（Stage1実行によるStage2駆動）の実測手順

## 7. この文書の終了条件

- [x] §3の三案の比較と、Aの前提がGoのinternal規則に抵触することの実測（§3.1）
- [ ] A-1とBのどちらを採るかの決定（ownerは[#1090](https://github.com/hat47x/tei-platform/issues/1090)と#2831）
- [ ] `gobuilder`移動の影響範囲（15ファイルのimport pathと、`plan/evidence/TEI_GO_*`が引用する実行手順）の洗い出し
- [ ] `tei self-build`の実装とSC0-01〜10の実試験
- [ ] `cli_contract.go`の広告と実応答の一致（既存のdiscovery照合へ載せる）
- [ ] SB-C1の実測手順

本書は計画の正本を増やすものではない。SB-C0の**入口の外形と拒否条件**だけを固定し、#2831のP0順序（SB-0a → SB-0b → SB-A → SB-B → SB-C）のうちSB-C0を詰めた設計候補である。最終判断は元計画・ADR・actual Evidenceへ返す。
