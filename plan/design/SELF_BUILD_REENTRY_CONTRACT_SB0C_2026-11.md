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

### 3.3 Bの形は既に実測で成立している（2026-11-08 追補）

§3.2で「Bの証拠の取り方」を推したが、その後、Bの形そのものが実装・実測された。[`tools/reference-self-build-two-stage`](../../tools/reference-self-build-two-stage/README.md)がそれである。

| stage | 何が動くか |
| --- | --- |
| stage0 | 外部Go seedでdriverを構築（driven processではない） |
| stage1 | stage0のdriverがchildを起動し、driver自身を再構築 |
| stage2 | **stage1で得たdriver実行ファイル**をinvokerとして起動し、同じchildで再構築 |

実測（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64、[記録](../evidence/TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md)）では、stage2のinvoker digestがstage1のartifact digestと一致し、`l3Claimed=true`が成立した。

**したがって§3.2の「Bにも制約がある」は、少なくともdriver自身については解消された。** `StdioProcessLauncher`をそのまま使えたのではなく、`reference-go-build-stage-driver`が独自にchildを起動してinvoker/invokedを記録する形で成立している。Plugin用のlauncherを流用する必要は無かった。

### 3.4 本稿の第一候補を「B（`cmd/tei`からのsubprocess起動）」に確定する

§3.1でAの前提が崩れ、§3.3でBの形が実測で成立した。**本稿の第一候補をBに確定する。** A-1（`gobuilder`の移動）は採らない。

理由は次の三点である。

1. A-1は29ファイル（Go 16・Markdown 12・CI 1）に触れ、Evidenceの実行手順の意味も変える。Bはそれらを変えない
2. Bの形は`reference-self-build-two-stage`で実測済みであり、stage1のartifactがstage2のinvokerになることをdigestで示せた
3. `cmd/tei`側に必要なのは「childを起動して、そのinvoker/invokedを記録する」入口だけであり、`reference-go-build-stage-driver`がその記録形式を既に持っている

**ただし、これは`cmd/tei`のL3が成立したという意味ではない。** 対象はdriver自身であり、`cmd/tei`はBuild入口を持たない。Bを採る場合の作業は「`cmd/tei`からdriver相当を起動できる薄い入口を足す」ことであり、§4の外形はその入口の提案である。

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
| 広告 | `acceptedCommands`へ追加し、`cliCommandCatalog`へは**載せない**（§4.5） | 現行catalogは`scope`と`effect`でread-onlyを機械固定している。書込み系は既存慣行どおりdispatch-onlyとする |

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

### 4.5 発見面（discovery）の制約：`writes-artifact`は単純に追加できない

§4.1は「`cliCommandCatalog`へ追加」と書いたが、実測すると**そのままでは追加できない**ことが分かった。

現行の`tei commands`は、次の二点を機械的に固定している。

| 固定されていること | 場所 |
| --- | --- |
| `scope`は`selected-read-only-commands` | `cmd/tei/cli_contract.go`、`cmd/tei/cli_contract_test.go:12`、`cmd/tei/cli_process_test.go:53` |
| **全entryの`effect`が`read-only`** | `cmd/tei/cli_contract_test.go:17`（`cmd.Effect != "read-only"`で失敗） |

したがって`effect: writes-artifact`のentryを同じcatalogへ足すと、この二つの試験が落ちる。`writeFlags`が既にあるのは「`read-only`は正本の意味を書き換えないという意味であり、プロセスが書かないことまでは意味しない」（#3182）という整理に基づくもので、**catalogが書込み系を広告してよいという意味ではない**。

選択肢は三つある。

| 案 | 内容 | 影響 |
| --- | --- | --- |
| D-1: catalogの`scope`を変える | `selected-commands`へ広げ、entryごとの`effect`で区別する | `scope`は公開契約の一部であり、`read-only`だけを信じて呼ぶ既存consumerの前提が変わる。**contract decision**である |
| D-2: 別のdiscovery面を作る | `tei commands -include-effects`や第二のcatalogで`writes-artifact`を出す | 既存consumerの前提を壊さない。ただし「第二のregistry」を作らないというcatalogの設計方針と衝突しないか確認が要る |
| D-3: catalogへ載せない | 既存の`deployment-*`や`issue-grant`と同じ扱い（dispatchするがadvertiseしない）にする | 既存慣行と一致する。ただし呼び手は`tei self-build`の存在を`-h`か文書でしか知れない |

**現行の慣行はD-3である。** `acceptedCommands` 49件のうちcatalogへ載っているのは19件で、`deployment-start`・`issue-grant`・`artifacts`などはdispatchされるがadvertiseされない。本稿は**D-3を既定案**とし、D-1/D-2は別の契約判断として切り出す。

D-3を採る場合、§4.1の「広告」欄は「`acceptedCommands`へ追加し、catalogへは載せない」に読み替える。`tei self-build -h`は既存の`-h`経路（終了コード0・標準出力）で自分のflagを説明する。

### 4.6 実装した最小の入口（2026-11-08）

§4の外形（`-intent`・`-supply`・`-state`・`-output`）は**まだ実装していない**。代わりに、その**一段下**の入口を実装した。`cmd/tei self-build`は、child実行ファイルを起動してinvoker/invokedを記録するだけで、Build Planの解決もBuild Intentの解釈もしない。

| flag | 意味 |
| --- | --- |
| `-child` | 起動する実行ファイル。**PATH解決しない** |
| `-child-arg` / `-child-arg-file` | childへ渡す引数一式（JSON配列） |
| `-artifact` | childが書くartifactのパス。既存なら**起動前に拒否**（no-clobber） |
| `-claimed-self` | このバイナリが自称するdigest。実行中imageと照合する |

出力は`tei.cli.self-build-entry/v0`であり、`tei.cli.*`のエンベロープを名乗らない。catalogへ載せないため、read-onlyコマンドと同じ成功形を名乗ると読み違えを招く。

`l3Claimed`は常に`false`である。この入口は**駆動の記録**を作るところまでで、stage1の生成物をinvokerに据える実験（SB-C1）は別に行う。

実装の検証は`cmd/tei/cli_self_build_test.go`が固定する。少なくとも次を含む。

| 試験 | 確かめていること |
| --- | --- |
| 発見面 | `self-build`が`acceptedCommands`にあり、`commandCatalog`に**無い** |
| invoker/invoked | 二つのdigestが別々に記録され、argvの先頭がexact child |
| artifact照合 | 子の主張が違えば`artifactCrossCheckMatched=false`と`failureCause` |
| SB-N13 | 誤った`-claimed-self`は`invokerClaimMatched=false`と`failureCause` |
| 一致する自称 | `invokerClaimMatched=true`でも`l3Claimed=false` |
| no-clobber | 既存artifactは起動前に拒否 |
| 引数不正 | 必須flag欠落・位置引数は`INVALID_ARGUMENT`・終了コード2 |

**§4の外形とこの入口の差**が、次に埋めるべき部分である。`-intent`の解決、`-supply`の照合（SB-0bの分類）、`-state`の分離、`qualified`欄は未実装である。

### 4.7 `-intent` を入口へ足す前に解くべき接続（2026-11-08）

`-intent`を足すには、Build Intentの宣言を読んで確定計画を導出する処理を`cmd/tei`から呼ぶ必要がある。その呼び出しは**現状のままでは組めない**。実測で確かめた。

| 観測 | 結果 |
| --- | --- |
| Build Intentの宣言を読む実装 | `tools/reference-build-authoring-intent`にある（`ParseIntent`・`Resolve…`） |
| そのパッケージ種別 | **全ファイルが`package main`**（8ファイル） |
| 別パッケージからのimport | `import "tei/tools/reference-build-authoring-intent" is a program, not an importable package` で**拒否**（`go vet`で実測） |
| もう一つの実装 | `tools/reference-plugin-build-supply/stage1.go`にも`ParseIntent`があるが、これも`package main` |

つまり`-intent`を`cmd/tei`へ足すには、次のいずれかが要る。

| 案 | 内容 | 影響 |
| --- | --- | --- |
| I-1: intent解析を`internal/`のパッケージへ切り出す | `reference-build-authoring-intent`の解析部分をモジュール直下の`internal/`配下の新パッケージへ移し、`tools/`側はそれをimportする | 既存参照実装のimport pathと試験が動く。`internal/`は`tools/`からも`cmd/`からも届く |
| I-2: `cmd/tei`が参照実装を**childとして起動**する | `-intent`の解決もsubprocessに任せ、結果のJSONを読む | 入口の記録形式（invoker/invoked）をそのまま使える。ただしintent解決が別プロセスになり、`qualified`の意味が薄まる |
| I-3: `cmd/tei`側に解析を**書き直す** | 同じJSON契約を読む第二実装を持つ | 二重実装。§35.2が禁じた「第二Runner」に近づく |

**I-2を本稿の既定案とする。** 理由は次の三点である。

1. `tei self-build`は既にchild起動とinvoker/invokedの記録を持つ。intent解決も同じ形に載せれば、新しい記録形式を作らない
2. I-1は`tools/`側の参照実装の配置を変える。SB-C0 §3.1と同じく、移動は影響範囲が広い
3. I-3は二重実装であり、宣言の解釈が二箇所に分かれる

I-2の制約は明記する。**intent解決をchildへ委ねると、「この`cmd/tei`が宣言を解釈した」とは言えなくなる。** 言えるのは「この`cmd/tei`が、指定したchildに宣言を渡し、その結果を受けた」ことだけである。L2のClaimを立てるときは、この差を明記する必要がある。

### 4.8 L2のClaimをどこで切るか（2026-11-08）

I-2を採ると、L2のClaimは**二段**に分かれる。混ぜると、childが解釈した宣言を「この`cmd/tei`が解釈した」と読ませてしまう。

| 層 | 何を言えるか | 何を言えないか |
| --- | --- | --- |
| **L2-a（受け渡し）** | この`cmd/tei`が、指定したchildへ**特定の宣言を渡した**。渡したバイトのdigestと、childの出力を受けたことを記録できる | 宣言が意味的に妥当か。childがその宣言をどう解釈したか |
| **L2-b（解釈）** | childが宣言を読み、確定したBuild Planを導出し、その照合値を返した | この`cmd/tei`が解釈したこと。導出の正しさ（宣言と照合値の突合が要る） |

L2-aは既存の入口（`tei self-build`）が既に持つ情報（invoked.argv・childの出力digest）で記録できる。**L2-bはchild側の記録（`reference-build-authoring-intent`の`ResolvedBuildReceipt`）を入口が受け取り、宣言のdigestと突き合わせて初めて成立する。**

したがって`-intent`を実装するときは、次の三つを一つの出力に含める。

1. 渡した宣言の**byte digest**（`IntentFingerprint`ではなく、渡したファイルそのもののdigest）
2. childが返した`IntentFingerprint`と`BuildPlanFingerprint`
3. 両者の突合結果（宣言のdigestがchildの読んだものと一致するか）

2だけを記録すると、childが**別の宣言**を読んで別の計画を返した場合を検出できない。1と3がその検出を担う。

**この節はL2のClaimを立てていない。** L2-a/L2-bの切り分けと、必要な記録の列挙だけである。

### 4.9 実装済みの状態（2026-11-09）

§4.8の三項目は、次の形で`tei self-build`に載った。**どれもL2-bを主張しない。**

| §4.8の項目 | 実装 | 出力欄 | 記録 |
| --- | --- | --- | --- |
| 1. 渡した宣言のbyte digest | 起動前にファイルを読み、`sha256`を取る | `intentBinding.intentBytesDigest` | [#3544](https://github.com/hat47x/tei-platform/pull/3544) |
| 2. childが返した`IntentFingerprint`と`BuildPlanFingerprint` | 解決child（`-intent-resolver`）のreceiptから読む | `planReceipt.intentFingerprint`・`planReceipt.planFingerprint` | [#3571](https://github.com/hat47x/tei-platform/pull/3571) |
| 3. 両者の突合結果 | 解決childの`planFingerprint`とbuilder childの申告を比べる | `planCrossCheck.state`（`matched`/`mismatched`/`unperformed`） | [#3584](https://github.com/hat47x/tei-platform/pull/3584) |

**項目2と3は別のchildの発言を突き合わせている。** 項目3が`matched`でも、それは解決childとbuilder childが同じfingerprintを名乗ったことに過ぎない。**builderがその宣言からその計画を導いた証明ではない**（`planCrossCheckBoundary`）。このため`l2bClaimed`は常に`false`である。

`-intent-resolver`を指定しない場合、`planReceipt`と`planCrossCheck`は出力されない。**欄が無いことを合格と読ませない**ためであり、`supplyCheck`の`delegated-to-child`と同じ扱いである。

**未了のまま残るもの**：builder childが自分の読んだ宣言と導いた計画を報告する経路。実測に使った`tools/reference-go-build-with-supply`は`planFingerprint`を返さないため、実プロセスでは`planCrossCheck`が`unperformed`になる。単体試験では報告するfixtureで`matched`を確かめている。

### 4.10 計画突合を`unperformed`から抜ける道筋（2026-11-09）

§4.9の未了事項を詰める。結論を先に書く。**「報告するchildを用意すれば済む」は成り立たない。** 既存のtoolを走査すると、`tei self-build`のchild文法（`-artifact-out`と`-state`でartifactを書き、1件のJSONをstdoutへ出す）を満たしながら計画fingerprintも申告するtoolが**1つも無い**。

### 4.10.1 走査した結果

`tools/reference-*/main.go` を機械的に走査した（stdoutへJSONを出すか、`PlanFingerprint`を扱うか）。

| tool | stdout JSON | plan fingerprint | `tei self-build`のchildとして使えるか |
| --- | --- | --- | --- |
| `reference-go-build-with-supply` | あり | **無し**（`BuildEvidence`に欄が無い） | **使える**（`-artifact-out`・`-state`・`-supply`）。ただし計画を名乗らない |
| `reference-build-execution-binding` | あり | あり（`ExecutionPlanFingerprint`） | **使えない**。flagを取らず、固定fixtureを出すだけ |
| `reference-build-attempt-evidence` | あり | あり（13箇所） | **使えない**。flagを取らず、失敗を模擬して常にexit 1 |
| `reference-plugin-self-build` | あり | あり（13箇所） | **使えない**。`-repo`・`-go`等はあるが`-artifact-out`・`-state`を取らない |
| `reference-plugin-package-builder` | **無し**（stderrのみ） | **無し** | 使えない。`-plugin`・`-plan`・`-out`でartifactは書くが、結果をstdoutへ出さない |
| `reference-build-authoring-intent` | あり | あり | 使えない。**計画を導出する側**でartifactをbuildしない |

**計画を名乗るtoolは、いずれも`tei self-build`のchild文法を持たない。** child文法を持つtoolは、計画を名乗らない。この非対称が`unperformed`の実質的な原因である。

### 4.10.2 三つの道筋

| 案 | 内容 | 何が確かめられるか | 何が残るか |
| --- | --- | --- | --- |
| **P-1: 計画を消費するchildを用意する** | `internal/plugin`の`BuildPlan`を入力として受け、`-artifact-out`と`-state`でartifactを書き、`planFingerprint`をstdoutへ返すchildを新設する | **1回の試行で**artifact照合と計画突合が両方観測できる | 新しい参照toolを1つ増やす |
| P-2: `reference-go-build-with-supply`に`planFingerprint`欄を足す | 呼び手が`-child-arg`で渡した値を転記する | 欄が埋まる | **転記に過ぎない。** このchildは計画を解釈しないので、単体試験のfixtureと同じ意味しか持たない |
| P-3: 二つのchildを1回の試行に並べる | 計画を名乗るchildとartifactを書くchildの両方を起動する | 両方の観測を1回で取れる | `tei self-build`は**childを1つしか起動しない**。契約の変更が要る |

**P-2は採らない。** 計画を解釈しないchildに欄を足しても、「builderがその計画を導いた」証拠にはならない。§4.8の(a)(b)(c)のうち(b)を形式的に埋めるだけで、実測の意味が増えない。**この案を採ると`unperformed`が`matched`に変わるが、主張の強さは変わらない**——それこそが避けたい読み違えである。

**P-3は本稿では採らない。** childを2つにすると`tei self-build`の契約（1 child・1 artifact・no-clobber）が変わり、SC0-03（state分離）とSC0-04（no-clobber）の意味も変わる。契約変更は別の設計判断である。

**P-1を次の一歩とする。** ただし新設するchildは**artifactを書く主体**でもあるため、`tei self-build`の1 child・1 artifactの契約を変えずに済む。

### 4.10.3 P-1で観測できること・できないこと

P-1のchildが実装できたとして、観測できるのは次である。

| 観測 | 言えること |
| --- | --- |
| `planCrossCheck.state=matched` | 解決childとbuilder childが**同じplan fingerprintを名乗った** |
| `planReceipt.planMatchesExpected=true` | 呼び手の宣言値と解決計画が一致した |
| `childRun.artifactCrossCheckMatched=true` | childの申告digestとディスク実体が一致した |

**それでもL2-bではない。** L2-bを言うには§4.8の(a)(b)(c)が要る。

- (a) builderが読んだ宣言byteの申告 — P-1のchildが返せば記録できる
- (b) entryの保持byteとの一致 — `intentBinding`が担う
- **(c) artifactがその計画の出力であること** — これは`matched`では出ない。計画とartifactの結び付けは、`plugin.LockedBuild`（`PlanFingerprint`と`ArtifactDigest`を束ねる型）のような**別の観測**を要する

したがってP-1を実装しても、`l2bClaimed`は`false`のままでなければならない。**次に詰めるのは(c)である。**

## 5. 追加の負例（試験仕様）

| ID | Given / 操作 | Expected |
| --- | --- | --- |
| SC0-01 | `tei self-build`を`-intent`も`-intent-file`も無しで実行 | 引数エラー（`SELF_BUILD_PRECONDITION_FAILED`）。Buildを開始しない（実装済み） |
| SC0-02 | `-supply`に破損Moduleを渡す | `corrupt-supply`で停止。分類は`-supply-verifier`のexactな実行ファイルへ委譲し、`verified: false`なら停止（実装済み） |
| SC0-03 | `-state`を二回の試行で共有 | 拒否。共有stateを「二回」と数えない |
| SC0-04 | `-output`が既存ファイル | no-clobberで拒否。上書きfallbackなし |
| SC0-05 | `validate`がerrorのWorkspace | Buildを開始しない |
| SC0-06 | 成功時の出力 | `qualified: false`。artifact成功をL3・受入・権限と表現しない |
| SC0-07 | Stage1のdigestを自己申告だけ | L3の証明にしない（SB-N13の再掲） |
| SC0-08 | `cmd/tei`から`tools/internal/`をimportする変更 | Goのinternal規則が拒否する（§3.1で実測）。規則を緩めるのではなく、`gobuilder`の移動かwrapperかを選ぶ |
| SC0-09 | Bで`-builder`を省略しPATHから解決させる | 拒否。child executableのdigestを記録できない起動を許さない |
| SC0-10 | Stage1が起動したchildのdigestと、記録したdigestが違う | L3不可。SB-N13と同じ扱い |
| SC0-11 | `tei self-build`を`cliCommandCatalog`へ`effect: writes-artifact`で足す | `cli_contract_test.go:17`が`read-only`以外を拒否して落ちる（§4.5）。D-1の契約判断なしにcatalogへ載せない |
| SC0-12 | 既存の`scope: selected-read-only-commands`を「書込み系も含む」と読み直す | 既存consumerの前提を変える。D-1の契約判断なしに行わない |
| SC0-13 | childが**別の宣言**を読んで別のBuild Planを返す（渡した宣言のdigestとchildの読んだものが違う） | L2-b不可。`intentBinding.intentByteCheckMatched`がfalseになり、`planCrossCheck`が`mismatched`になる（実装済み） |
| SC0-14 | childの`IntentFingerprint`だけを記録し、渡した宣言のbyte digestを記録しない | L2-b不可。entryは`intentBinding.intentBytesDigest`を必ず記録し、childの`IntentFingerprint`だけでは突合しない（実装済み） |

## 6. 未決事項

- コマンド名（`self-build` / `build` / `artifact build`）
- Build Intentの解決方式（`-intent`でID指定か、Workspace内の宣言から解決か）
- ~~A-1とBのどちらを採るか~~ → **§3.4でBに確定**（A-1は採らない）
- ~~`StdioProcessLauncher`をBuild用child起動へそのまま使えるか~~ → **§3.3で不要と判明**。`reference-go-build-stage-driver`が独自にchildを起動して記録する形で成立した
- ~~catalogへ`effect: writes-artifact`を足せるか~~ → **§4.5でD-3（catalogへ載せない）を既定案に**。D-1/D-2は別の契約判断
- ~~`cmd/tei`からchild driverを起動する薄い入口~~ → **§4.6で実装済み**（`tei.cli.self-build-entry/v0`、#3433）
- ~~`cmd/tei`を対象にしたL3~~ → **実測済み**（[記録](../evidence/TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md)、`bb7fc01bb`）
- ~~`-intent`を入口へ足せるか~~ → **§4.7でI-2（childとして起動）を既定案に**。L2のClaimは§4.8の二層に分ける
- ~~`-intent`の実装（I-2）と、宣言byte digest・childの照合値・突合結果の三つを一つの出力に載せること~~ → **§4.9で実装済み**（`-intent-resolver`のreceiptと`planCrossCheck`。`l2bClaimed`は常にfalse）
- **builder childが自分の読んだ宣言と導いた計画を報告する経路**。これが無いと`planCrossCheck`は実プロセスで`unperformed`のままである（§4.9）
- `-self-digest`の取得方法（実行ファイル自身のdigestをどう自己参照するか）
- 出力のschemaVersionと、既存`tei.cli.*`との命名一貫性
- Seed Trustの扱い（stage0のGoは外部seed。TEI自身によるGoコンパイラのself-hostingではない）

## 7. この文書の終了条件

- [x] §3の三案の比較と、Aの前提がGoのinternal規則に抵触することの実測（§3.1）
- [x] Bの形が実測で成立することの確認（§3.3、[`reference-self-build-two-stage`](../../tools/reference-self-build-two-stage/README.md)）
- [x] 第一候補をBに確定（§3.4、A-1は採らない）
- [x] discovery面の制約の実測と既定案の決定（§4.5、D-3）
- [x] `cmd/tei`からchild driverを起動する薄い入口の実装（§4.6、`tei.cli.self-build-entry/v0`。SC0-04・SC0-06・SC0-07・SC0-09・SC0-10相当を試験で固定）
- [x] `-intent`を足す前の接続制約の実測と既定案の決定（§4.7、I-2）
- [x] L2のClaimを二層に分ける設計（§4.8、L2-a/L2-bと必要な記録の列挙）
- [x] `cmd/tei`を対象にしたL3の実測（[記録](../evidence/TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md)、`bb7fc01bb`）
- [x] `-intent`（I-2）の実装。SC0-01〜05は外形の拒否条件として、SC0-13・SC0-14は`intentBinding`と`planCrossCheck`として実装・実試験済み（§4.9）
- [x] §4の外形（`-supply`・`-state`・`-output`・`qualified`欄）の実装（[#3463](https://github.com/hat47x/tei-platform/pull/3463)・[#3476](https://github.com/hat47x/tei-platform/pull/3476)）
- [x] 計画突合の3状態を実プロセスで観測（[記録](../evidence/TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_ACTUAL_2026-11-09.md)、`3a4ddeba6`）
- [x] 既存toolの走査と、`unperformed`の実質的な原因の特定（§4.10.1）
- [ ] **P-1**（`internal/plugin`の`BuildPlan`を消費し、artifactを書き、`planFingerprint`を返すchild）の実装と、実プロセスでの`planCrossCheck=matched`観測
- [ ] **(c) artifactがその計画の出力であること**の観測（`plugin.LockedBuild`のような束ね型を使う。§4.10.3）
- [ ] D-1（catalogの`scope`拡張）を採るかどうかの契約判断（採る場合のみ、catalogの広告と実応答の一致を照合へ載せる）

**A-1（`gobuilder`移動）は採らないため、影響範囲の洗い出しは終了条件から外した。** 29ファイルの内訳は[記録](../evidence/TEI_GO_INTERNAL_RULE_AND_BUILDER_MOVE_2026-11-07.md)として残す。I-1（intent解析の`internal/`への切り出し）も、I-2を既定案にしたため本稿では行わない。

**P-2（`reference-go-build-with-supply`への欄追加）は採らない。** 計画を解釈しないchildに欄を足すと`unperformed`が`matched`に変わるが、主張の強さは変わらない（§4.10.2）。

本書は計画の正本を増やすものではない。SB-C0の**入口の外形と拒否条件**だけを固定し、#2831のP0順序（SB-0a → SB-0b → SB-A → SB-B → SB-C）のうちSB-C0を詰めた設計候補である。最終判断は元計画・ADR・actual Evidenceへ返す。
