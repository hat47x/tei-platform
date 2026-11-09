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

### 4.11 (c)をどう実装したか（2026-11-09）

(c)を「plan fingerprintとartifact digestを一つの検証済み束ね型にする」として実装した。`planCrossCheck`とは**別の欄**にしたのが要点である。計画の一致とartifactの結び付けを同じ欄に混ぜると、`matched`を(c)の代わりに読ませてしまう。

| 材料 | 出所 | entryが独立に確かめたこと |
| --- | --- | --- |
| 計画のidentity（Plugin・outputKind・target・profile） | 解決childのreceiptの`plan`本体 | 要約欄ではなくplan本体から読む |
| builderの`planFingerprint` | childのstdout | 解決childの値と一致するか（不一致は`mismatched`） |
| builderの`artifactDigest` | childのstdout | **ディスクのバイト**と一致するか |

これらから`internal/plugin`の`BuildArtifactLock`を組み立て、`Validate`と`BuildArtifactLockFingerprint`を通す。lockが検証を通らない、またはディスクが読めない場合は`unperformed`であり、`bound`として扱わない。

| `state` | 意味 | `ok`への影響 |
| --- | --- | --- |
| `bound` | 検証済みlockが組め、artifact digestがディスクと一致した | 影響なし |
| `mismatched` | 計画fingerprintが解決値と違う、またはartifact digestがディスクと違う | **`ok=false`・`failureCause`** |
| `unperformed` | receiptが無い／childが申告しない／lockが検証を通らない／ディスクが読めない | 影響なし（理由を`unperformedReason`へ） |

**`bound`でもL2-bではない。** lockに載る`planFingerprint`と`artifactDigest`は**どちらもbuilder childの自己申告**である。entryが独立に確かめたのは、artifact digestがディスクと合うことと、lockが`internal/plugin`の検証を通ることだけである。よって`l2bClaimed`は`false`のままであり、境界文がそう述べる。

**(c)を`bound`で満たしたと読まないこと。** (c)が本来問うのは「**その計画がそのartifactを生んだか**」であり、`bound`が示すのは「**その計画fingerprintとそのartifact digestが一つのlockとして整合し、digestが実体と合う**」ことである。この差は、builderが自己申告を偽らないという前提に依存している。前提を観測で置き換えるには、builderの出力そのものをentryが検証する経路（例: builderが`plugin.LockedBuild`を自分で作って返す）が要る。

### 4.12 child自身がlockを作る（2026-11-09）

§4.11の末尾が挙げた経路を実装した。**builder childが自分の`BuildArtifactLock`を組み立て、`internal/plugin`の検証を通してから返す。entryはそれを、自分が独立に導いたlockと比べる。**

| 段 | §4.11の実装 | §4.12の実装 |
| --- | --- | --- |
| lockを作る主体 | entryのみ | **childも作る** |
| lockの検証 | entryが`Validate` | **childも`Validate`**。通らなければreportを出さずexit 1 |
| 突合 | なし | **entryのlock fingerprintとchildのlock fingerprintを比較** |

#### child側

reportへ`buildArtifactLock`を足した。`validated`・`lockFingerprint`・`planFingerprint`・`artifactDigest`・`boundary`を持つ。lockが`Validate`を通らない場合は**artifactを書いた後にreportを出さずexit 1**にする。reportを返すと、検証できなかった束縛が「返ってきた」と読まれるためである。

#### entry側

`planArtifactBinding`へ次を足した。

| 欄 | 意味 |
| --- | --- |
| `childLockChecked` | childのlockを比べたか |
| `childLockValidated` | childが検証済みと申告したか |
| `childLockMatched` | entryのlock fingerprintと一致したか |
| `childLockFingerprint` | childのlock fingerprint |

childのlockが無い場合は`childLockChecked=false`のままにする（**欄が無いことを合格と読ませない**）。あるのに一致しない場合は`mismatched`として`ok=false`・`failureCause`にする。

#### この段で増えた観測と、増えなかったもの

**増えた観測**：経路の違う二つの組み立てが**同じlock fingerprint**になったこと。childは自分が読んだplanと自分が書いたバイトから、entryはreceiptのplan identityとchildの申告からlockを作る。一致すれば、両方の読みが同じ束縛を指している。

**増えなかったもの**：**L2-b**。`l2bClaimed`は`false`のままである。

- builderが読んだのは`-plan`のファイルであり、entryが保持した宣言そのものではない
- **(c)が問う「その計画がそのartifactを生んだか」は、lockが一致しても出ない。** lockが示すのは「計画identity・planFingerprint・artifactDigestの三つ組が、二つの独立した組み立てで同じになった」ことである
- **childのlockもchildの申告に依存する。** entryが比べられるのは、entry自身が導いた値との一致までである

#### 残る限界と次の一歩

entryのlockもchildの`planFingerprint`と`artifactDigest`から作るため、依存は完全には消えていない。計画側もentryが独立に持つには、`-intent-resolver`のreceiptではなく**entry自身が宣言から計画を導く経路**が要る。それは§4.7でI-1（intent解析の`internal/`への切り出し）を採らなかった制約に戻る。

### 4.13 entryが宣言から計画を独立に導くには（I-1の実測見積り）

§4.12の残る限界を詰めるため、I-1の**実際の作業量**を測った。結論から書く。**I-1は「配置の移動」ではなく「パッケージ境界の新設」であり、`tools/`側の参照実装を分割する作業になる。**

#### 対象の実測

`tools/reference-build-authoring-intent`は8ファイル・2205行（うち試験1074行）である。

| ファイル | 行数 | 関数 | 役割 |
| --- | --- | --- | --- |
| `intent.go` | 384 | 9 | `BuildIntent`の型・`ParseIntent`・`NormalizeIntent`・`IntentFingerprint` |
| `resolve.go` | 267 | 4 | `Resolve`・`MarshalReceipt`・`VerifyDerivedReceipt` |
| `fixture.go` | 406 | 21 | `DirSourceResolver`・`ParseFixtureCatalog`・`NewFixtureContext` |
| `input_json.go` | 58 | 3 | `transform.ParseJSON`を使う厳密JSON読み |
| `main.go` | 79 | 2 | CLI |
| 試験4ファイル | 1082 | 34 | — |

**32のexported関数がある。** `internal/`へ移すのはこのうち再導出に必要な部分だけでよく、CLI（`main.go`）と試験は`tools/`に残せる。

#### 依存の実測

`internal/`側から見ると、参照実装が使うのは次の4つだけである。

| import | 用途 |
| --- | --- |
| `tei/internal/core` | `ParseCanonicalID`・`ValidNumericVersion` |
| `tei/internal/plugin` | `BuildPlan`・`BuildPlanFingerprint`・`BuildDependency` |
| `tei/internal/plugin/transform` | `ParseJSON`（厳密JSON） |
| `tei/internal/syntax` | Plugin定義の構文解析 |

**すべて`internal/`配下であり、`cmd/tei`からも届く。** つまり新しいパッケージを`internal/`に置けば、`cmd/tei`と`tools/`の両方からimportできる。§3.1で問題になった`tools/internal/gobuilder`とは違い、**internal規則が障害にならない**。

#### 三つの案と採否

| 案 | 内容 | 作業量 | 何が得られるか |
| --- | --- | --- | --- |
| **I-1a: 解析だけを移す** | `ParseIntent`・`IntentFingerprint`・`ParseFixtureCatalog`を`internal/`の新パッケージへ移し、`Resolve`は`tools/`に残す | 中（型と解析の移動、試験の分割） | entryが**宣言の解釈と照合値の導出**を自分でできる。計画の導出はchildのまま |
| I-1b: 導出まで移す | `Resolve`も`internal/`へ移す | 大（`ResolutionContext`とfixture catalogも移す） | entryが**計画まで**自分で導ける。L2-bの依存が最も減る |
| I-1c: 移さない | 現状維持（I-2） | なし | — |

**I-1aを次の一歩とする。** 理由は次の三点である。

1. **§4.12の依存は計画側にある。** entryがchildの`planFingerprint`を受け取らずに済むには、entryが計画を**導出**する必要がある。しかしその前段の「宣言を厳密に読み、意図の照合値を出す」は独立にでき、これだけで**宣言byteの突合をentryが行える**
2. **I-1bは`ResolutionContext`の設計まで動かす。** `fixture`は名前のとおり参照実装の固定入力であり、`internal/`へ移すと「fixture catalogが製品の一部」に見える。この誤読は避けたい
3. **I-1cは残る限界をそのまま残す。** §4.12が明示した依存が消えない

#### I-1aで増える観測と、増えないもの

**増える観測**：entryが`ParseIntent`と`IntentFingerprint`を**自分で**呼べるため、childの`intentFingerprint`申告に依存せずに「entryが読んだ宣言の照合値」を持てる。`intentBinding`が今は「childの申告との一致」しか言えないのに対し、**entry自身の導出値**を基準にできる。

**増えないもの**：**L2-b**。計画の導出は依然childにあり、`planCrossCheck`の限界（二つの自己申告の一致）は変わらない。**(c)も変わらない。**

**I-1aの後に残るもの**：計画の導出をentryが行う経路（I-1b）。これは`ResolutionContext`の設計判断を伴うため、別の設計判断として切り出す。

#### I-1aの作業範囲（見積り）

| 作業 | 対象 |
| --- | --- |
| 新パッケージの作成 | `intent.go`の型・`ParseIntent`・`NormalizeIntent`・`IntentFingerprint`、`input_json.go` |
| `ParseFixtureCatalog`の扱い | これは`BuildPlan`の解決に使う。I-1aでは**移さない**（`tools/`に残す）。entryが宣言だけを読む用途では不要 |
| `tools/`側の追随 | `intent.go`の残りが新パッケージをimportする形へ変更。`Resolve`はそのまま |
| 試験 | `intent`関連の試験を新パッケージ側へ複製せず、**新パッケージの試験として書き直す**（`tools/`側は`Resolve`の試験を残す） |

**この見積りは本節の範囲では実行しない。** 実装は次の周へ回す。

### 4.14 I-1aを実装し、(a)(b)を埋めた（2026-11-09）

§4.13の見積りどおりにI-1aを実装し、続けて**§4.8の(a)(b)**を埋めた。二つは独立の変更だが、狙いは同じである——**entryが宣言について自分で確かめられる範囲を広げる**こと。

#### I-1a：宣言readerを`internal/buildintent`へ

| 対象 | 扱い |
| --- | --- |
| `BuildIntent`とその型・`ParseIntent`・`ValidateSourceRef`・`NormalizeIntent`・`IntentFingerprint`・`FingerprintJSON`・`ValidateInputJSON` | **`internal/buildintent`へ移動** |
| `Resolve`・`ResolutionContext`・`DirSourceResolver`・`ParseFixtureCatalog`・`NewFixtureContext` | **`tools/`に残す**（fixture catalogは製品の概念ではない） |

`tools/reference-build-authoring-intent/intent.go`は**型aliasと薄い委譲**だけになった。型は複製ではなくaliasなので、宣言の定義はリポジトリに一つだけである。`Resolve`の試験は**無変更で通った**。

**entryの宣言読みが変わった。** `cmd/tei`の`readIntentDeclaration`は`map[string]json.RawMessage`で一部の欄だけを見ており、**未知キー・大文字小文字の別名・導出値の手書きを見逃していた**。共有readerへ置き換えたことで、entryも同じ規則で拒否する。これは挙動の変更なので、厳密readerを通らない既存fixtureを実際の宣言へ差し替えた。

entryは導出した宣言の照合値を`intentDeclaration.intentFingerprint`として記録する。**childの申告に依存しない基準**をentryが持つことになる。

#### (a)(b)：builder childが宣言を読む

`tools/reference-plugin-package-build-child`へ`-intent`（省略可）を足した。渡すとchildは共有readerで宣言を読み、`intentBytesDigest`（childが読んだbyte）・`intentFingerprint`（childが導いた照合値）・`intentBoundToPlan`を申告する。

| 条件 | 挙動 |
| --- | --- |
| 宣言のsubjectがplanのPluginと一致しない | **artifactを書いた後にreportを出さずexit 1**。一致し得ない二つの値を「読んだ値」として返さないため |
| `-intent`を渡さない | 宣言の欄を**出さない**。欄が無いことを合格と読ませないため |

entry側（`cmd/tei`）は**変更不要**だった。childが申告するようになったことで`intentBinding`が自動的に埋まる。

#### 観測結果（WSL2 Ubuntu 22.04 / go1.27.3）

| 欄 | 値 |
| --- | --- |
| `intentDeclaration.digest`（entryが保持） | `sha256:d5131c505d7d013d713ae5196e76aaa5a0ff238b51a82378a7786a130082940b` |
| `intentBinding.childIntentBytesDigest`（childが読んだ） | **同じ値** |
| `intentBinding.intentByteCheckPerformed` | **`true`** |
| `intentBinding.intentByteCheckMatched` | **`true`** |
| `intentDeclaration.intentFingerprint`（entryが導出） | `c4cf576f5e6946484ad321e6f1b3dc7ec4d711052be1460dd1db9da210a65dff` |
| `intentBinding.childIntentFingerprint`（builder childが導出） | **同じ値** |
| `planReceipt.intentFingerprint`（解決childが導出） | **同じ値** |

**三つの独立した導出が同じ値になり、entryとbuilder childのbyte digestも一致した。**

#### この段で埋まったもの・埋まらないもの

**埋まった**：§4.8の(a)と(b)。childが読んだ宣言byteのdigestをchild自身が申告し、それがentryの保持したbyteと一致した。

**埋まらない**：**L2-b**。`l2bClaimed`は両欄とも`false`である。

- (a)(b)が埋まったことは「**childが同じ宣言を読んだ**」ことを示す。(c)の「その計画がそのartifactを生んだか」は変わらない
- しかし**「childがその宣言からその計画を導いた」ことの証明にはならない。** childは`-plan`を入力として受け取り、宣言は**identityの照合にだけ**使う。計画の導出はchildの別の入力（`-plan`）に依存している
- childの申告が真である保証もない。entryが比べられるのは自分が導いた値との一致までである

#### 次に詰めること

残る段差は**childが宣言から計画を導くこと**、すなわち**I-1b**（`Resolve`を`internal/`へ移し、entryまたはchildが計画まで独立に導く）である。`ResolutionContext`の設計判断を伴うため、別の設計判断として切り出す。

**より小さい一歩**として、childがsubject identityに加えて**`outputKind`・`target`・`profile`もplanと比べる**ことができる。§4.8の(c)を直接埋めはしないが、**宣言とplanが同じbuild座標を指す**ことを増やせる。

### 4.15 宣言とplanのbuild座標を照合する（2026-11-09）

§4.14末尾の「より小さい一歩」を実装した。§4.14ではsubject identity（PluginのIDとversion）だけを比べていたが、**identityが一致しても別のbuildを指す場合がある**（`outputKind`・`target`・`profile`）。

| 座標 | 何が変わるか |
| --- | --- |
| `outputKind` | 配布物か実行物か。消費経路とauthorityが変わる |
| `target` | どのOS・arch向けか |
| `profile` | releaseかdebugか |

**三つを一つずつ比べ、違えばartifactを書いた後にreportを出さずexit 1にする。** reportを返すと、照合できなかった組み合わせが「返ってきた」と読まれるためである。どの座標が違ったかはstderrで名指しする。

**`profile`の省略は一致と読まない。** 宣言が`profile`を書かないことは「planのprofileと同じ」ではない。これは`unperformed`を`matched`と読ませない原則と同じ扱いである。

**それでもL2-bではない。** 座標の一致が示すのは「宣言とplanが同じbuildを指す」ことであり、`l2bClaimed`は`false`のままである。(c)も`bound`では出ない。宣言の他の欄（`sources`・`builder`・`recipe`・`dependencies`・`parameters`・`secretRequirements`）は未照合である。

**座標照合をさらに広げるより、I-1bとしてまとめて扱う方がよい。** 欄ごとに照合を足していくと「照合した」が「導出した」に見えやすくなる。§4.8が禁じた混同そのものである。

### 4.16 entryがin-processで計画を導くか（判断材料、2026-11-09）

§4.13のI-1bを実装した（[#3654](https://github.com/hat47x/tei-platform/pull/3654)）ことで、`cmd/tei`は**subprocessを起動せずに計画を導ける**。`-intent-resolver`によるchild起動が既定のままなので、**既定経路を切り替えるかどうか**が新しい判断項目になった。本節は判断を下さず、材料を測る。

#### 実測：四つのportの実装は今どこにあるか

| port | 実装している場所 |
| --- | --- |
| `SourceResolver` | `tools/reference-build-authoring-intent`（`DirSourceResolver`）のみ |
| `BuilderResolver` | 同上（`FixtureCatalog`）のみ |
| `DependencyResolver` | 同上（`FixtureCatalog`）のみ |
| `TargetCatalog` | 同上（`FixtureCatalog`）のみ |

`internal/`と`cmd/`には**一つの実装も無い**。

#### 実測：`internal/plugin`はBuilder catalogの代わりになるか

ならない。`internal/plugin`の`Requirement`は`Plugin`と`CompatibleMajor`だけを持ち、**`recipeKinds`・`targets`・`profiles`・`authorityScope`を持たない**。`BuilderResolver`が解く問い（contract・recipe kind・target・profileに適合するProviderはどれか）に答えられない。

#### 三つの案

| 案 | 内容 | 何が要るか | 何が増えるか |
| --- | --- | --- | --- |
| **R-1: 既定をin-processへ** | `cmd/tei`が`internal/buildintent`の`Resolve`を直接呼ぶ | **`internal/`に四つのportの実装を新設**（Source解決・Builder catalog・Dependency catalog・Target catalog）。fixture catalogは参照実装専用なので移せない | subprocess起動が消え、entryが**自分で計画を導いた**と言える |
| R-2: 既定はchild起動のまま | 現状維持 | なし | 何も変わらない |
| R-3: 併存 | `-intent-resolver`があればchild、無ければin-process | R-1と同じ実装 | 経路が二つになり、**どちらで導いたかが出力の別欄になる** |

#### 測ったうえでの所見

**R-1は「配線」ではなく「catalogの製品化」である。** portは`internal/`にあるが、実装はfixtureにしかない。in-processへ切り替えるには、fixture catalogに相当するものを`internal/`へ新設する必要があり、それは§4.13が避けた「fixtureが製品に見える」問題を**別の形で**招く。今度はfixtureではなく**本物のcatalog**が必要になるが、その本物がどこから来るか（Workspaceの宣言か、外部の配布物か）は未決である。

**R-3はR-1より危険が大きい。** 二つの経路が同じ出力欄を書くと、`planReceipt.resolverDigest`のような「どの実行体が導いたか」の記録が経路によって意味を変える。§4.8が避けた混同（自己申告と導出の混同）に近い。

**したがって既定はR-2のまま据え置き、R-1の前提（catalogの出所）を先に決める。** これは`docs/status.md`や既存ADRの範囲ではなく、**新しい契約判断**である。§7に未決として名指しする。

#### この節が言っていないこと

- R-1を採らない、とは言っていない。**前提が未決**だと言っている。
- in-processで導いても**L2-bではない**。(c)も変わらない。導出の主体が変わるだけで、「その計画がそのartifactを生んだ」ことは別の観測である。

### 4.17 宣言の他の欄をどこまで照合するか（2026-11-09）

§4.15末尾の残作業を、**照合できる欄**と**照合の意味が未決の欄**に分けた。

#### 照合した欄（[#3661](https://github.com/hat47x/tei-platform/pull/3661)）

builder childが`compareIntentFields`で比べる。

| 欄 | 比べる内容 | 何を検出するか |
| --- | --- | --- |
| `recipe.kind`・`recipe.entrypoint` | 完全一致 | 別の作り方を要求する宣言 |
| `sources` | kind と id（集合・件数） | 宣言が別のSource集合を指している |
| `dependencies` | kind と id（集合・件数） | 宣言が別の依存集合を指している |

不一致は**欄を名指し**して拒否し、**reportを出さない**。拒否した宣言について`bound`と報告しないためである。

#### 照合しなかった欄と、その理由

| 欄 | 理由 |
| --- | --- |
| sourceの`locator`・`digest` | **plan側は導出値**。宣言は`ref`を書き、planは解決済みのlocatorとbyte digestを持つ。比べると**正しいbuildをすべて不一致と呼ぶ** |
| dependencyの`version`・`artifactDigest` | 同じくplan側の導出値 |
| `builder.contract`・`compatibleMajor` | planが持つのは**解決済みの`BuilderIdentity`**（`ContractVersion`・`ArtifactDigest`）であり、宣言の要求（contract名とmajor）と形が違う。比べるには「要求」と「解決結果」の対応規則を決める必要がある |
| `secretRequirements` | planは**名前だけ**を`SecretRequirementNames`として持つ（`ResolvedBuildReceipt`）。`plugin.BuildPlan`自体には無い。宣言の`Authority`はplanに対応する欄が無い |
| ~~`parameters`~~ | **実測して訂正した。** どちらの側も名前順ソートのみで値を変えない（`buildintent.NormalizeIntent`・`plugin.NormalizeBuildPlan`）。名前で突き合わせれば順序に依存しないので、**照合する**（[#3664](https://github.com/hat47x/tei-platform/pull/3664)、§4.18） |

**「比べられる欄」と「比べるべき欄」は違う。** 導出値を比べると常に不一致になり、形の違うものを比べると定義が要る。比べないことを明示するのが、黙って比べないより安全である。

- ~~残る欄を永久に照合しないとは言っていない。`parameters`は正規化の意味を決めれば照合できる~~ → **`parameters`は測ると正規化がソートだけだったので実装した**（§4.18）。`builder`は要求と解決結果の対応規則を決めれば照合できるが、それは**測れば決まる**種類ではない（§4.18末尾）。
- 照合の拡大は**L2-bを進めない**。宣言とplanが同じ意味を指すことの確認であり、「その宣言がその計画を導く」ことではない。

### 4.18 `parameters`を照合する（2026-11-09）

§4.17は`parameters`を「正規化の意味が未決」として残した。**測ると前提が誤っていた。**

| 側 | 正規化 | 何をするか |
| --- | --- | --- |
| 宣言 | `buildintent.NormalizeIntent` | `Parameters`を**名前でソートするだけ** |
| plan | `plugin.NormalizeBuildPlan` | `Parameters`を**名前でソートするだけ** |

`resolve.go`は`Parameters: n.Parameters`と代入するだけで、値を変換しない。**したがって値の違いは本物の違いである。** 順序だけが正規化の対象なので、**名前で突き合わせれば順序に依存しない**。

照合は名前の存在・値・件数を見る。不一致は**名前と両方の値**を名指しする。

**正規化の性質そのものを試験で固定した。** 照合が「名前順ソートのみ」という性質に依存しているため、将来`NormalizeIntent`が値を正規化するようになったら、その試験が落ちて気づける。依存を暗黙にしない。

#### 残る2欄は「測れば決まる」種類ではない

| 欄 | 何が要るか |
| --- | --- |
| `builder.contract`・`compatibleMajor` | planが持つのは**解決済みの`BuilderIdentity`**（`ContractVersion`・`ArtifactDigest`）であり、宣言の要求と**形が違う**。「要求」と「解決結果」の対応規則を決める必要がある |
| `secretRequirements` | planは名前を`SecretRequirementNames`として持つが`plugin.BuildPlan`には無い。宣言の`Authority`に対応する欄が無い |

**`parameters`は測れば決まった。この2つは規則そのものを決める必要がある。** 同じ「未決」でも種類が違う。
### 4.19 `builder`の契約を照合し、`secretRequirements`の不可能性を示す（2026-11-09）

§4.18は`builder`と`secretRequirements`を同じ「規則を決める必要がある」欄として残した。**踏み込むと種類が違った。**

#### `builder`は規則が既にあった

`fixture.go`は宣言の`contract`と`compatibleMajor`を次のように解決する。

```go
ContractVersion: fmt.Sprintf("%s/v%d", p.Contract, p.ContractMajor),
```

**決めるべき規則は既にコードにあり、測れば読めた。** childはこの規則を自分でも適用して、組み立てた綴りとplanの`ContractVersion`を比べる。**child側にも規則を書いたのは意図的である**：resolverが綴りを変えたらこの照合が壊れる場所になり、その場所を試験で固定できる。

| 比べる | 比べない |
| --- | --- |
| `builder.contract` + `compatibleMajor` から組み立てた `<contract>/v<major>` | planの`builder.ID`・`Version`・`ArtifactDigest`（**解決済みの実体**。宣言は方針を書く） |
| `builder.provider.plugin`（指定時）・`builder.pin.artifactDigest`（指定時） | — |

**方針と実体を区別する。** 方針から実体は決まらないので、`builder.ID`・`Version`・`ArtifactDigest`そのものを宣言の不一致と読まない。ただし**方針を指定した場合**は、その方針が解決結果に反映されたかを比べる（`provider.plugin`と`pin.artifactDigest`）。`ResolveBuilder`が候補をこの二つで絞るため、**同じ宣言から導いたplanでは必ず一致する**。したがってこの比較が発火するのは、planがその宣言由来でない場合だけである。

**この節の初版は「`provider`と`pin`は照合の対象外」と書いていた。** `#3670`が両方を照合する実装を入れ、`#3700`が試験を実装に合わせたため、記述を実装へ合わせて直した。

#### `secretRequirements`は比べる相手が無い

| 事実 | 帰結 |
| --- | --- |
| `plugin.BuildPlan`に`SecretRequirements`欄が無い | plan側に比べる相手が無い |
| 名前は`ResolvedBuildReceipt.SecretRequirementNames`に載る | **receiptはchildへ渡っていない**（childが受け取るのは`-plan`と`-intent`） |
| 宣言の`Authority`（取得元）に対応する欄がplanにもreceiptにも無い | 比べる対象が存在しない |

**これは「測れば決まる」問題ではない。** 測って分かったのは「比べる相手が無い」ことである。照合するには、receiptをchildへ渡すか、planへ欄を足すか、いずれかの**設計変更**が要る。§4.18が「規則を決めれば照合できる」と書いたのは`secretRequirements`については誤りだった。

#### 宣言の欄の到達点

| 欄 | 状態 |
| --- | --- |
| `subject` identity・`outputKind`・`target`・`profile` | 照合済み（§4.14・§4.15） |
| `recipe`・`sources`・`dependencies` | 照合済み（§4.17） |
| `parameters` | 照合済み（§4.18） |
| `builder.contract`・`compatibleMajor` | 照合済み（[#3684](https://github.com/hat47x/tei-platform/pull/3684)） |
| `builder.provider`・`builder.pin` | 方針であり、解決結果と対応しない（照合の対象外） |
| `secretRequirements` | **比べる相手が無い**（設計変更が要る） |

**照合の拡大はL2-bを進めない。** 宣言とplanが同じ意味を指すことの確認であり、「その宣言がその計画を導く」ことではない。

### 4.20 欄ごとの照合は何を保証するのか（`secretRequirements`の扱いを決める、2026-11-09）

§4.19で分かった性質は、**`secretRequirements`だけの話ではなく、§4.17以降に足してきた欄ごとの照合すべてに当てはまる。** ここで一度、何を保証しているのかを言い直す。

#### 補足：同じ宣言から導いたplanでは、欄照合は決して失敗しない

**同じ宣言からresolverが導いたplanに対して、この照合は決して失敗しない。** 要求を満たさない宣言からはresolverがplanを作らないためである（実測：`compatibleMajor`を2にすると`no compatible builder provider for contract tei.plugin-build/v2`で停止する）。

したがってこの照合が効くのは、**planがその宣言から導かれたものではない**場合だけである。具体的には次の三つ。

- 宣言がplanの解決後に変わった
- planが別の宣言から導かれた（§4.8のSC0-13が問う状況）
- planがresolver以外（手書き・別ツール）で作られた

**これは弱い検査ではなく、狙いどおりの検査である。** childは「渡されたplanが渡された宣言と対応しているか」を確かめており、対応していなければ止まる。

#### 言い直し

宣言のbyteが一致していれば（`intentByteCheckMatched=true`）、そのbyteから導かれる値は**すべて**一致する。**planがそのbyteから導かれたものである限り**、§4.14以降に足してきた`recipe`・`sources`・`dependencies`・`parameters`・`builder`・build座標の照合は**必ず通る。**

つまり欄ごとの照合は、**独立の証拠を増やしていない。** 増やしているのは次の一点だけである。

> **そのplan（またはreceipt）が、保持したbyteから導かれたものではない場合に、どこが食い違うかを名指しで示す。**

これは価値がある。食い違いを「byteが違う」とだけ言われるより、「宣言の`target`が違う」と言われる方が原因を追える。だが**保証の強さは変わらない。**

#### したがって`secretRequirements`は照合しない

§4.18は`secretRequirements`を「規則を決める必要がある」として残したが、**照合しても保証が増えない以上、規則を決める理由がない。**

三つの選択肢を比べた結果は次のとおり。

| 案 | 内容 | 判定 |
| --- | --- | --- |
| **S-1: childで照合しない** | planに対応欄が無いことを理由付きで記録する | **採る** |
| S-2: childがreceiptも受け取る | childの入力契約が変わり、「planを消費するchild」ではなくなる | 採らない。**保証が増えないのに契約だけ広がる** |
| S-3: `plugin.BuildPlan`に秘密名を持たせる | 製品の型の意味変更 | 採らない。**比較のためだけに製品型へ欄を足す**ことになり、その欄に消費者が無い |

**S-2とS-3は「照合を増やすために契約を広げる」取引である。** 保証が増えないなら、取引は割に合わない。§4.19が示したのは「この種の照合は原因究明を助けるが、保証は増やさない」ということであり、その結論は**照合を広げる方向に進まない**ことを支持する。

#### これ以上の欄照合は足さない

§4.17以降、照合できる欄を順に足してきた。**ここで打ち切る。** 理由は三つ。

1. **保証が増えない。** byte一致が成り立てば必ず通る
2. **「照合した」が「導出した」に見える危険がある。** 欄が増えるほど、planの全項目を確かめたように読める。実際に確かめているのは「保持したbyteとの対応」だけである
3. **欄を増やすより、通らない場合に原因を名指しする方が効く。** 食い違いの原因を示す仕組みは既にあり、欄を足す必要はない

**残る段差は欄照合では埋まらない。** 埋めるのはI-1bの先、すなわち「そのplanがそのartifactを生んだか」（(c)）と、entryの既定経路（§4.16のR-2）である。**欄を足すことは、このどちらにも近づかない。**

#### この節が言っていないこと

- 既に足した照合を外す、とは言っていない。**原因究明の道具として残す**価値はある
- `secretRequirements`の意味が不要だとは言っていない。**照合の相手がplanに無い**と言っている

### 4.21 (c)を再現で観測する（2026-11-09）

§4.20は「残る段差は欄照合では埋まらない」と書き、埋める対象として(c)とentryの既定経路を挙げた。**本節は(c)を埋める。**

#### lockでは足りない理由を、あらためて

`planArtifactBinding=bound`（§4.11・§4.12）が示すのは「plan fingerprintとartifact digestが一つの検証済みlockとして整合し、digestがディスクと合う」ことである。**そのplanを実行するとそのバイトが出る、という観測ではない。** 二つの値を結び付けただけであり、builderが申告を偽らないという前提に依存する。

#### 再現で置き換える

**plan Pでbuilderを二回、別のstateで走らせて同じバイトDが出て、ディスク上のartifactがDであれば、そのartifactはPの出力である。** 決定性が観測されたことと、digest衝突がないことを条件に。

| 段 | 動き |
| --- | --- |
| 一回目 | 従来どおり`-output`へ書く（no-clobberは保たれる） |
| 二回目 | **同じchild実行物**を、**新しい一時state**と**一時出力先**で起動する |
| 比較 | 二回目の申告digest == 一回目の申告digest、かつ ディスク上の`-output` == 一回目の申告digest |

`-artifact-out`と`-state`はentryがchild argvへ足している値なので、**二回目だけを差し替える**。呼び手が渡した他の引数はそのまま使うので、条件の違うbuildを比べているのではない。

**二回目は呼び手の`-output`へ書かない。** 成功した一回目のartifactを検査が上書きしないためである。ただしchildが`-artifact-out`を無視する場合は一回目の出力先へ書くため、**二回目の後にディスクを読み直して**一回目の申告と比べる。

#### 出力

`artifactReproducibility`を**常に**出す。

| `state` | 意味 | 効果 |
| --- | --- | --- |
| `reproduced` | 二回のbuildが同じバイトを出し、ディスク上も一致した | なし（成功） |
| `mismatched` | 一致しない | **`ok=false`・`failureCause`** |
| `unperformed` | 実行していない／実行できなかった | 理由を`unperformedReason`へ |

`-rebuild-check`を渡さない場合も`unperformed`と理由を出す。**欄が無いことを一致と読ませない**ためである（SC0-18と同じ規則）。

#### 実測

[記録](../evidence/TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md)。同じplan、同じchild実行物、別々のstateでの二回のbuildが同じdigest `sha256:7d2f6ff73176e1c817242a225d1499230579fd54ae2c69ff0f845535482f3ae4` を出し、ディスク上のartifactも同じdigestだった。`secondChildDigest`は`tei`実行物と同一で、二回目も同じchildを使ったことが読める。

#### これで(c)が埋まる。だがL2-bではない

**(c)が問うのは「そのplanがそのartifactを生んだか」であり、本節はそれに答えた。** 範囲は`boundary`が明記する——同じホスト・同じbinary・二回の試行で観測した決定性であり、別ホスト・別Go版・別builder版での再現は確かめていない。

**L2-bは埋まらない。** (c)はplanとartifactの関係であり、**「entryが宣言を解釈してそのplanを導いた」ことではない。** 後者は依然として解決childの申告に依存する。`l2bClaimed`は`false`のままである。**(c)が埋まったことをL2-bの成立と読ませない。**

#### 既定を`-rebuild-check`にするかは別の判断

**本節は既定を変えない。** 二回目のbuildは時間とI/Oを倍にするため、呼び手の明示が要る。

ただし**既定にすべき理由はある。** (c)を観測しない限り、`bound`が(c)の代わりに読まれる危険が残る。既定を切り替えるかは、費用と証拠の重みの取引であり、**別の契約判断**として残す。

#### この節が言っていないこと

- 決定性を証明したとは言っていない。**二回の観測**である
- 再現性がL2-bや受理を意味するとは言っていない

### 4.22 catalogの出所を決める（R-1の前提、2026-11-09）

§4.16はR-1（既定をin-processへ）の前提として「**本物のcatalogがどこから来るか**」を未決として残した。本節は測ったうえで決める。

#### 実測：catalogに相当するものは、まだfixtureにしかない

| 探したもの | 結果 |
| --- | --- |
| `internal/`・`cmd/`に`authorityScope`・`recipeKinds`を持つ型 | **無い** |
| plugin定義（`internal/syntax`）がbuilder能力を宣言する構文 | **無い** |
| `internal/composition`の`Providers` | 構成の競合判定であり、builder catalogではない |
| 実際に在るもの | `tools/reference-build-authoring-intent`の`FixtureProvider`だけ |

**Builder catalogが答える問いは6つある。** contract・compatibleMajor・recipeKind・target・profile・（任意でproviderとpin）に適合する**exactなbuilder artifact**はどれか。この6つを持つ型は`FixtureProvider`だけである。

#### 実測：移す範囲は既に切り分けられている

`tools/reference-build-authoring-intent/fixture.go`は407行・21宣言である。**ホストI/Oを持つのは先頭の3つだけ**で、残りは純粋なデータ処理である。

| 範囲 | 内容 | ホストI/O |
| --- | --- | --- |
| 31〜142行 | `DirSourceResolver`（`os.Root`・`fs.WalkDir`・symlink拒否） | **あり** |
| 144〜406行 | `FixtureCatalog`・`ParseFixtureCatalog`・`ResolveBuilder`・`ResolveDependency`・`CheckTarget`・3つのfingerprint・port adapter | なし |

つまり**catalogの機構そのものは既に製品非依存**であり、`fixture`という名前が「参照実装の固定入力」に見せているだけである。

#### 四つの案

| 案 | 出所 | 何が要るか | 何が増えるか |
| --- | --- | --- | --- |
| **K-1** | Workspaceの宣言 | `tei.workspace.yaml`にcatalog欄を新設 | Workspaceが「どのbuilderが在るか」の正本になる |
| K-2 | 導入済みpluginの能力宣言 | **plugin定義の構文拡張**（contract・recipeKinds・targets・profiles・authorityScope） | 構文と意味の追加。受理済みpluginからcatalogを導ける |
| **K-3** | **呼び手が名指しするcatalogデータ＋そのdigest** | catalogのJSONを行列の入力として受け取り、digestを必須にする | 新しいAuthorityを作らない。既存の`-supply`・`-intent-resolver`と同じ形 |
| K-4 | fixtureのまま（R-2据え置き） | なし | in-process経路を持たない |

#### K-3を採る

**理由は三点である。**

1. **新しいAuthorityを作らない。** entryは既に「自分で発見せず、渡されたものを記録する」形を採っている（`-child`・`-supply`・`-intent-resolver`はすべて呼び手が名指しする）。catalogも同じ形にすれば、**entryが勝手にbuilderを選ぶ経路が生まれない**
2. **K-1とK-2は「誰がbuilderの存在を決めるか」を製品へ移す。** それは§4.5のD-3（catalogの`scope`）やD-1と同じ種類の契約判断であり、いま決めるべきものではない
3. **K-3なら移す作業が機械的である。** 上の表のとおり、機構は既に製品非依存で、名前から`fixture`を外すだけで済む

**digestを必須にする。** 呼び手が名指しする以上、「どのcatalogで導いたか」を記録できなければ、`planReceipt.resolverDigest`と同じ種類の曖昧さが入る。`-build-catalog`と`-build-catalog-digest`を**対**で要求し、片方だけなら起動しない。

#### 移行の形

| 段 | 内容 |
| --- | --- |
| 1 | `fixture`を外した名前で`internal/buildintent`へ移す（`BuildCatalog`・`ParseBuildCatalog`・`ResolveBuilder`・`ResolveDependency`・`CheckTarget`・`DirSourceResolver`）。`tools/`側は型aliasと薄い委譲にする（§4.14のI-1aと同じ形） |
| 2 | `cmd/tei`に`-build-catalog`・`-build-catalog-digest`を足し、**in-processで計画を導く経路**を作る |
| 3 | 既定をin-processへ切り替えるか（R-1）を別途決める。§4.16のR-2据え置きは段2の完了まで維持する |

**`DirSourceResolver`も移す。** symlink拒否・root外参照の拒否は**方針**であり、entryが独自実装すると方針が二箇所に分かれる。ソースを読むのはホストI/Oだが、それは「製品でない」理由にならない。

#### これでL2-bがclaimできるようになるか

**なる。ただし範囲を明示する。**

段2が終わると、entryは自分で宣言を読み、自分でcatalogを引き、自分で計画を導く。**「このentryが宣言を解釈してその計画を導いた」と言えるようになる。** それが§4.8のL2-bが要求していた内容である。

**段2は実装済みである**（[記録](../evidence/TEI_GO_SELF_BUILD_INPROCESS_RESOLUTION_2026-11-09.md)）。entryが導いた計画のfingerprintは参照実装のchildが出した値と一致した。ただし**Claimを立てるかは別の判断**であり、`l2bClaimed`は`false`のまま据え置いている（欄の意味を変える判断をしていない）。

言えないものは変わらない。

- **意味の受理・Runtime Authority**（`qualified`は`false`のまま）
- **供給のTrust・Seed Trust・`go.sum`のTrust根拠**
- **catalogそのものの信頼性**。K-3は「呼び手が名指ししたcatalogを使った」ことまでしか言えない。**そのcatalogが正しいという主張はしない**

#### この節が決めていないこと

- 既定経路をin-processへ切り替えるか（段3。§4.16のR-1/R-2/R-3は据え置き）
- catalogの**中身**を誰が作るか（配布物か、Workspaceから生成するか）。K-3は「呼び手が名指しする」ところまでである

### 4.23 L2-bのClaimをいつ立てられるか（2026-11-09）

§4.8はL2を二層に分けた。L2-aはentryが宣言を引き渡したこと、**L2-bはその宣言が計画として解釈され、その計画がartifactを生んだこと**である。§4.14以降、L2-bの材料は一つずつ揃ってきたが、**`l2bClaimed`は定数`false`のまま**だった。

本節はその状態を変える。**ただし、単一の観測がL2-bになるわけではないという原則は変えない。**

#### なぜ定数だったか、なぜ今は違うか

`l2bClaimed`を`false`に固定していたのは、**一つの観測をL2-bと読ませない**ためだった。`intentByteCheckMatched`だけでも、`planArtifactBinding=bound`だけでも、`artifactReproducibility=reproduced`だけでもL2-bではない。この判断は正しく、**今も変わらない**。

変わったのは、**すべてが同時に成り立つ試行を作れるようになった**ことである。

| 材料 | どこで入ったか | 何を確かめるか |
| --- | --- | --- |
| 宣言byteの突合（(a)(b)） | §4.14 | builderが読んだ宣言がentryの保持したbyteと同じ |
| 計画の突合 | §4.9・§4.11 | builderの申告した計画がentryの計画と同じ |
| plan→artifactの束縛 | §4.11・§4.12 | plan fingerprintとartifact digestが一つのlockで整合し、ディスクと合う |
| artifactの再現 | §4.21 | そのplanを実行するとそのバイトが出る |
| **entry自身の解釈** | §4.22 段2 | その計画を導いたのがentryである |

最後の一つが入るまで、**「entryが宣言を解釈した」とは言えなかった**。段2がそれを埋めた。

#### 述語

`l2b.claimed`が`true`になるのは、次の**五つがすべて成り立つ**ときである。一つでも欠ければ`false`で、**欠けた条件をすべて`l2b.unmet`に列挙する**。

| # | 条件 | 欠けたときの述べ方 |
| --- | --- | --- |
| 1 | entry自身が計画を導いた（`planReceipt.derivedBy=entry-in-process`かつ`state=resolved`） | 誰が導いたかを名指しする |
| 2 | 宣言byteの突合が**実施され、一致した** | 未実施か不一致かを区別する |
| 3 | builderの計画がentryの計画と**一致した** | 突合の状態を述べる |
| 4 | plan→artifactの束縛が`bound` | 状態を述べる |
| 5 | artifactが再現された（`reproduced`） | 状態を述べる |

**条件を「欄が存在する」で判定しない。** 欄が無いことを一致と読ませない原則（SC0-18）を、述語にも適用する。`unperformed`はすべて不成立として扱う。

#### 単一の観測の欄は`false`のまま

`planArtifactBinding.l2bClaimed`・`intentBinding.l2bClaimed`は**`false`のまま据え置く**。それぞれの欄は「その観測だけではL2-bではない」ことを述べており、**その記述は正しい**。組み合わせの判断は新しい`l2b`欄が持つ。

**述語はそれらの欄を読まない。** 読むと、`false`を書くことが目的の欄が主張の入力になり、意味が循環する。試験で固定した。

#### 実測

[記録](../evidence/TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md)。三つの場合を実プロセスで観測した。

| 場合 | `l2b.claimed` | `l2b.unmet` |
| --- | --- | --- |
| in-process＋`-intent`＋`-rebuild-check` | **`true`** | 空 |
| 同じで`-rebuild-check`のみ無し | `false` | `the artifact was not reproduced from the plan (unperformed)` |
| `-intent-resolver`経由（entryは解釈していない） | `false` | `the plan was derived by "", not by this entry` |

**一つ目の場合、`planArtifactBinding.l2bClaimed`は`false`のまま**である。単一観測の欄と組み合わせの欄が別のことを述べていることが、出力に現れている。

#### このClaimが言っていないこと

`l2b.boundary`が明記する。

- **受理・Runtime Authorityではない**（`qualified`は`false`のまま）
- **catalogが正しいという主張ではない**（§4.22 K-3、SC0-26）
- **供給のTrust・Seed Trustの判断ではない**
- **L2-cでもL3でもない**
- **このホスト・このbinaryでの観測**である

#### この節が決めていないこと

- `l2b.claimed`を**既定の試行で立てるか**。既定はin-processではなくchild経路のままであり、`-rebuild-check`も既定offである。**Claimを立てるには呼び手が明示する必要がある**
- 既定経路の切り替え（§4.16のR-1/R-2/R-3）と`-rebuild-check`の既定化（§4.21末尾）。いずれも本節の対象外

### 4.24 Claimを回帰試験にする（2026-11-09）

§4.23はL2-bを五条件の述語にした。**述語が正しくても、実際の試行で五条件が揃わなければClaimは立たない。** 本節は、**リポジトリ内の参照catalogと参照builder childで五条件が揃うこと**を試験に固定したことを記録する。

#### 固定したもの

`cmd/tei`の試験が、実際に動く実行物を組み立てて**一度の`self-build`を走らせる**。

| 手順 | 内容 |
| --- | --- |
| 1 | `cmd/tei`と`tools/reference-plugin-package-build-child`と`tools/reference-build-authoring-intent`を`go build`する |
| 2 | ワークスペース・宣言・source root・参照catalog（`testdata/catalog.json`）を一時ディレクトリに置く |
| 3 | 参照実装を**直接**起動して、builder childが消費するplanファイルを作る（entryの経路ではない） |
| 4 | `tei self-build`を**in-process＋`-intent`＋`-rebuild-check`**で走らせる |
| 5 | `l2b.claimed=true`・`l2b.unmet`空・`qualified=false`・`ok=true`・artifactが存在することを確かめる |

**参照実装はplanファイルを作るためだけに使う。** entryは自分で計画を導く（それが条件1である）。したがってこの試験は「entryが解釈した」ことを、参照実装に依存せずに確かめている。

#### 何が保証されるか

- 五条件が**リポジトリ内の材料だけで**同時に成立する
- どれか一つが壊れたら、**名指しされた条件**として試験が落ちる（`l2b.unmet`を失敗メッセージに出す）
- `go test ./...`に含まれるため、**`tools/repository-health-check.sh`の経路で自動的に回る**

#### 何が保証されないか

- **別ホスト・別Go版**での成立（§4.21の範囲）
- **catalogが正しいこと**（参照catalogは参照実装用のデータである）
- 受理・Runtime Authority・Trust
- **既定の試行**でClaimが立つこと。既定はin-processでも`-rebuild-check`でもない（§4.23末尾）

#### この試験が落ちたときに読むもの

`l2b.unmet`の文字列がそのまま失敗メッセージに入る。どの条件が欠けたかが、試験の出力だけで分かる。

### 4.25 既定経路をどこまで強くするか（三つの未決を一つの判断として、2026-11-09）

§7に三つの未決が並んでいる。**同じ根に三つ実がある。**

| 未決 | 問い |
| --- | --- |
| `-rebuild-check`を既定にするか | 再現を毎回測るか |
| entryの既定経路をin-processへ切り替えるか | 導出を誰がするか |
| `l2b.claimed`を既定の試行で立てられるようにするか | 主張を既定で立てるか |

**どれも「既定の試行をどこまで強くするか」を問うている。** 個別に決めると、片方を強くしたせいでもう片方の前提が変わる。本節は材料を測り、**一つの判断として扱う**ことを提案する。ただし決定はしない。

#### 実測：既定の試行が今なにをするか

`tei self-build`は、`-intent-resolver`も`-build-catalog`も無ければ**計画を導出しない**。両方あるときは拒否する（一行の明示が要る）。つまり:

| 試行 | 計画の導出 | 再現 | `l2b.claimed` |
| --- | --- | --- | --- |
| 既定（どちらも無し） | **しない** | しない | 立てられない |
| `-intent-resolver` | child | 呼び手次第 | 立てられない（entryが解釈していない） |
| `-build-catalog`一式 | **entry** | 呼び手次第 | 五条件が揃えば立つ |

#### 実測：強くすると何が増えるか

| 強くすること | 増えるもの | 増えないもの |
| --- | --- | --- |
| `-rebuild-check`を既定に | **childの実行が二回**になる（同じ条件で別state） | 供給のTrust、catalogの正しさ |
| in-processを既定に | 呼び手はcatalog 3点セットを渡す義務を負う | 同じく別の層 |
| `l2b.claimed`を既定で | 上の二つが既定で要る | 意味の受理・Runtime Authority |

**`-rebuild-check`の既定化は、build時間をほぼ倍にする。** 二回目は同じchild実行物を新しいstateで起動するので、ビルドが高価なほど素直に二倍近くになる。決定論的なchildでは同じバイトが出るが、**その保証は時間と引き換え**である。

#### 三つの組み合わせ

| 案 | `-rebuild-check` | 導出 | 既定でclaim | 何を優先するか |
| --- | --- | --- | --- | --- |
| **P-1: 現状維持** | 明示 | 明示 | 立たない | **速さ**。呼び手が強さを選ぶ |
| P-2: 再現だけ既定 | **既定** | 明示 | 立つ（in-process時） | (c)の取り違えを減らす |
| P-3: 両方既定 | **既定** | **既定** | **立つ** | 主張の強さ。時間を払う |
| P-4: 既定は軽く、`-strong`で全部 | 明示 | 明示 | 明示時のみ | 一つのflagで強さを選ぶ |

#### 測ったうえでの所見

**P-3は今の製品の性質を変える。** 既定でchildを二回起動し、catalog 3点セットを要求する。これは「既定は軽く、強さは呼び手が選ぶ」という既存の形（`-supply`・`-intent-resolver`・`-state`）と**逆向き**である。

**P-2は非対称。** 再現だけ既定にすると、child経路でも毎回二回ビルドするのに`l2b.claimed`は立たない（entryが解釈していないため）。**コストを払って主張が増えない**組み合わせである。

**P-4が既存の形と整合する。** 既定は軽いまま、呼び手が`-strong`（仮称）で「再現も測り、可能ならclaimも立てる」を選ぶ。ただし新しいflagを一つ増やすので、その名前と意味を決める必要がある。

**したがって本記録はP-4を推すが、決定はしない。** 理由は、P-1〜P-3のどれも「既定の試行の意味」を変える契約判断であり、§4.16がR-1/R-2/R-3を据え置いたのと同じ種類だからである。

#### 決めるために足りないもの

| 足りないもの | なぜ要るか |
| --- | --- |
| 実buildの所要時間 | `-rebuild-check`の倍化が許容できるかの判断材料 |
| catalog 3点セットを既定で用意できるか | in-process既定化の前提。誰がcatalogを作るかは§4.22末尾で未決 |
| `-strong`の名前と粒度 | P-4を採る場合の契約 |

**実buildの所要時間は測った**（[記録](../evidence/TEI_GO_SELF_BUILD_REBUILD_COST_2026-11-09.md)）。WSL2の参照fixtureで plain 192ms・`-rebuild-check` あり 346ms、**1.80倍**。ただし**このfixtureのchildは即座に終わる**ので、実buildが高価な場合は倍率の意味が変わる。内訳は計測点を分けていないため未計測である。

#### この節が決めていないこと

- 三つのどれも決めていない。**一つの判断として扱うことを提案しただけである。**
- `-strong`の名前・粒度・既定値。
- 既定経路を変えることの後方互換への影響（既存の呼び手は何も渡さずに走っている）。

### 4.26 二つのホストで同じ計画とartifactが出る（2026-11-09）

§4.21・§4.23・§4.24は、いずれも「**このホスト**での観測」と範囲を限定していた。本節はその限定を**二つのホスト**に広げる。

#### 測ったこと

同じ宣言・同じsource・同じcatalogから、`windows/amd64`と`linux/amd64`で四つの照合値を導き、比べた。

| 値 | 何が決めるか | Windows | Linux |
| --- | --- | --- | --- |
| `intentFingerprint` | 宣言の正規化 | `22e346d1…` | **同じ** |
| `sourceClosure` | sourceの相対member pathと内容 | `582ec417…` | **同じ** |
| `planFingerprint` | 上記＋catalogのfingerprint | `e61096a4…` | **同じ** |
| `artifactDigest` | 計画から作るバイト | `sha256:675dc815…` | **同じ** |

**Go版も違う**（`go1.27.0`と`go1.27.3`）。版をまたいで一致したことになる。

`cmd/tei`のエンドツーエンド試験（§4.24）も**WindowsのGoで直接**実行し、`TestSelfBuildEstablishesL2BEndToEnd`が**PASS**した。つまり`l2b.claimed=true`になる試行が両方のホストで成立する。

#### なぜ一致するのか

偶然ではない。三つの性質から説明できる。

| 値 | ホスト非依存である理由 |
| --- | --- |
| `sourceClosure` | 相対member pathと内容だけから決まり、rootの場所を含まない（§4.14で`DirSourceResolver`を移した際の性質） |
| `planFingerprint` | `NormalizeBuildPlan`が列挙順を除いた構造から決まる |
| `artifactDigest` | `MaterializePluginPackage`が正規化した定義を直列化した結果で、改行やパスの混入が無い |

**実測はこの説明を確かめたことになる。** 「一致した」だけでなく「一致するはずだと言える」状態になった。

#### 範囲の更新

Claimの`boundary`文言を、**このホスト**から**観測した二つのホスト**へ改める。

| 欄 | 変更 |
| --- | --- |
| `artifactReproducibility.boundary` | 「the host that ran it」＋「linux/amd64 と windows/amd64 で同じバイトを観測した」 |
| `l2b.boundary` | 「the host that ran it」＋「観測したのは linux/amd64 と windows/amd64。他のホストは未証明」 |

**観測していないホストを含めない**という原則は変えない。`darwin`・`arm64`・別のGo版の組合せは未観測である。

#### この節が言っていないこと

- すべてのOSで一致するとは言っていない。**二つのホスト**である
- catalogが正しいこと（§4.22 K-3、SC0-26）、受理・Runtime Authority・Trustは変わらない
- 既定の試行でClaimが立つことは変わらない（§4.23末尾）

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
| SC0-15 | childが自分のlockを作るが`Validate`を通らない | reportを出さずexit 1。検証できなかった束縛を「返ってきた」と読ませない（§4.12） |
| SC0-16 | childのlock fingerprintとentryのlock fingerprintが違う | `planArtifactBinding.state=mismatched`・`childLockMatched=false`・`ok=false`（§4.12） |
| SC0-17 | builder childがentryの保持した宣言と**別の宣言**を読む | `intentBinding.intentByteCheckMatched=false`・`ok=false`。childが読んだbyteをself-reportさせることで初めて検出できる（§4.14） |
| SC0-18 | builder childへ`-intent`を渡さず、沈黙を合格と読む | `intentByteCheckPerformed=false`と理由が残る。欄が無いことを一致と読ませない（§4.14） |
| SC0-19 | builder childが読む宣言がplanのPluginと別のsubjectを指す | childがreportを出さずexit 1。一致し得ない二つの値を「読んだ値」として返さない（§4.14） |
| SC0-20 | 宣言の`compatibleMajor`を、planを作ったときと変える | childがreportを出さずexit 1。**同じ宣言から解いたplanでは起こらず**、planがその宣言由来でない場合にだけ検出できる（§4.19） |
| SC0-21 | `secretRequirements`の照合を足そうとする | 足さない。byte一致が成り立てば必ず通るため保証が増えず、childの入力契約か`plugin.BuildPlan`を広げることになる（§4.20） |
| SC0-22 | `-rebuild-check`を渡し、二回目のbuildが違うバイトを出す | `artifactReproducibility.state=mismatched`・`ok=false`。再現しないartifactはそのplanの出力と言えない（§4.21） |
| SC0-23 | `-rebuild-check`を渡さず、沈黙を再現の成功と読む | `state=unperformed`と理由が残る。**欄が無いことを一致と読ませない**（§4.21） |
| SC0-24 | `artifactReproducibility=reproduced`をL2-bの成立と読む | 読まない。再現が示すのは「そのplanがそのartifactを生んだ」こと（(c)）であり、entryが宣言を解釈したことではない。`l2bClaimed`は`false`のまま（§4.21） |
| SC0-25 | `-build-catalog`だけを渡し、digestを渡さない | 起動しない。**どのcatalogで導いたかを記録できない経路を許さない**（§4.22） |
| SC0-26 | 呼び手が名指ししたcatalogを使ったことを「そのcatalogが正しい」と読む | 読まない。K-3が言えるのは「名指しされたcatalogを使った」ことまでである（§4.22） |
| SC0-27 | `l2b.claimed=true`を受理・Runtime Authorityと読む | 読まない。`qualified`は`false`のままで、`l2b.boundary`が範囲を明記する（§4.23） |
| SC0-28 | 単一の観測（`bound`だけ、`reproduced`だけ）でL2-bを主張する | 主張しない。述語は五条件の同時成立を要求し、欠けた条件をすべて列挙する（§4.23） |
| SC0-29 | `l2bClaimed`を`false`に固定したまま「L2-bは未成立」と述べ続ける | 述べ続けない。ただし**Claimを立てるには呼び手がin-processと`-rebuild-check`を明示する必要がある**（既定は変わっていない。§4.23） |
| SC0-30 | 二つのホストで一致したことを「すべてのOSで一致する」と読む | 読まない。観測したのは`linux/amd64`と`windows/amd64`で、他のホストは未観測である（§4.26） |

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
- ~~builder childが自分の読んだ宣言と導いた計画を報告する経路~~ → **§4.14で実装済み**（`-intent`でchildが宣言byte digestと照合値を申告する）
- ~~`builder.contract`・`compatibleMajor`の照合~~ → **§4.19で実装済み**（対応規則はresolverに既にあり、`BuilderContractVersion`として一箇所に定義した）
- ~~`secretRequirements`の照合~~ → **§4.20でS-1（childで照合しない）に決定**。保証が増えないため、childの入力契約も`plugin.BuildPlan`も広げない
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
- [x] **P-1**（`internal/plugin`の`BuildPlan`を消費し、artifactを書き、`planFingerprint`を返すchild）の実装と、実プロセスでの`planCrossCheck=matched`観測（[記録](../evidence/TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md)、`e4b478581`）
- [x] (c)の一歩目（`planArtifactBinding`。検証済みlockで計画fingerprintとartifact digestを束ね、digestをディスクと突き合わせる。§4.11）
- [x] `planCrossCheck=matched`と`planArtifactBinding=bound`の**同時**実測（[記録](../evidence/TEI_GO_SELF_BUILD_MATCHED_AND_BOUND_2026-11-09.md)、`99b220ae0`）
- [x] builderが自分で`BuildArtifactLock`を作って返す経路（§4.12。entryのlockとの一致を実測。[記録](../evidence/TEI_GO_SELF_BUILD_CHILD_LOCK_2026-11-09.md)、`669bc83f0`）
- [x] I-1の実測見積りと案の選択（§4.13。**I-1a（解析だけを`internal/`へ移す）**を次の一歩に）
- [x] **I-1a**（`ParseIntent`・`IntentFingerprint`を`internal/`の新パッケージへ移し、entryが宣言を自分で読む）の実装（§4.14、[記録](../evidence/TEI_GO_SELF_BUILD_ENTRY_INTENT_FINGERPRINT_2026-11-09.md)、`f522e8c65`）
- [x] §4.8の(a)(b)（builder childが宣言byteを申告し、entryの保持byteと一致する）の実装（§4.14、[記録](../evidence/TEI_GO_SELF_BUILD_CHILD_DECLARATION_2026-11-09.md)、`ee7148138`）
- [x] entryまたはchildが宣言から**計画まで**独立に導く経路（I-1b。[#3654](https://github.com/hat47x/tei-platform/pull/3654)で`Resolve`を`internal/buildintent`へ移し、fixture catalogは`tools/`に残した。§4.13の反対理由はcatalogについてであり、portについては当たらなかった）
- [x] (c)の観測：**同じplanを二回、別stateでbuildして同じバイトが出た**（§4.21、[記録](../evidence/TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md)。`-rebuild-check`で`artifactReproducibility=reproduced`。lockの`bound`が示せなかった「そのplanがそのartifactを生んだ」を再現で観測した。**L2-bではない**）
- [ ] `-rebuild-check`を**既定**にするかどうかの判断。**§4.25で三つの未決を一つの判断として扱う案（P-1〜P-4）を示し、実測コスト（1.80倍）を足した**。判断は未（§4.25末尾）
- [x] catalogの出所の決定（§4.22。**K-3: 呼び手が名指しするcatalogデータ＋必須のdigest**。新しいAuthorityを作らず、既存の`-supply`・`-intent-resolver`と同じ形）
- [ ] **entryの既定経路をin-processの`Resolve`へ切り替えるか**の判断（§4.16のR-1/R-2/R-3）。前提だったcatalogの出所は§4.22で決めた（K-3）。切り替え自体は段2の完了後に別途決める
- [x] **段1**：`fixture`を外した名前で`BuildCatalog`・`ParseBuildCatalog`・`ResolveBuilder`・`ResolveDependency`・`CheckTarget`・`DirSourceResolver`を`internal/buildintent`へ移した（`tools/`側は型aliasと薄い委譲。参照実装の試験は**無変更で通る**。§4.22の移行表、[記録](../evidence/TEI_GO_SELF_BUILD_CATALOG_MOVE_2026-11-09.md)）
- [x] **段2**：`cmd/tei`に`-build-catalog`・`-build-catalog-digest`・`-build-source-root`を足し、in-processで計画を導く経路を作った（`-intent-resolver`とは排他。§4.22、[記録](../evidence/TEI_GO_SELF_BUILD_INPROCESS_RESOLUTION_2026-11-09.md)）。**entryが導いた計画のfingerprintは参照実装のchildが出した値と一致した。** `planRecompute`・`planCrossCheck`・`planArtifactBinding`はin-process経路でも成立する
- [x] L2-bのClaimを立てるかの判断と実装（§4.23。**五条件の同時成立**を述語にし、`l2b.claimed`と`l2b.unmet`として出す。[記録](../evidence/TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md)。単一観測の`l2bClaimed`欄は`false`のまま据え置き、述語はそれらを読まない）
- [x] L2-bのClaimを**回帰試験**にする（§4.24。参照catalogと参照builder childで五条件が揃うことを`cmd/tei`の試験で固定し、`go test ./...`経由で健康性確認に載る。[記録](../evidence/TEI_GO_SELF_BUILD_L2B_GATE_2026-11-09.md)）
- [x] 二つのホスト（`linux/amd64`・`windows/amd64`）で同じ計画とartifactが出ることの実測（§4.26。[記録](../evidence/TEI_GO_SELF_BUILD_CROSS_OS_DIGESTS_2026-11-09.md)。`intentFingerprint`・`sourceClosure`・`planFingerprint`・`artifactDigest`の四つが一致し、L2-bのエンドツーエンド試験もWindowsでPASSした。**Go版も違う**）
- [ ] `darwin`・`arm64`・別のGo版の組合せでの観測（§4.26の範囲外）
- [ ] `l2b.claimed`を**既定の試行**で立てられるようにするか（今は呼び手がin-processと`-rebuild-check`を明示する必要がある。§4.23末尾）
- [x] 宣言とplanの**build座標**（`outputKind`・`target`・`profile`）の照合（[#3652](https://github.com/hat47x/tei-platform/pull/3652)。childが三座標を一つずつ比べ、違えばartifactを書いた後にreportを出さずexit 1。[記録](../evidence/TEI_GO_SELF_BUILD_BUILD_COORDINATES_2026-11-09.md)）
- [x] 宣言の他の欄のうち、planが同じ形で持つもの（`recipe`・`sources`・`dependencies`）の照合（[#3661](https://github.com/hat47x/tei-platform/pull/3661)。§4.17）
- [x] `parameters`の照合（[#3664](https://github.com/hat47x/tei-platform/pull/3664)。測るとどちらの側も名前順ソートのみで値を変えないと分かったので、名前で突き合わせる。§4.18）
- [x] `builder`（要求と解決結果の対応規則）の照合（[#3670](https://github.com/hat47x/tei-platform/pull/3670)。対応規則はresolverに既にあり、`BuilderContractVersion`として一箇所に定義した。§4.19、[記録](../evidence/TEI_GO_SELF_BUILD_BUILDER_REQUIREMENT_2026-11-09.md)）
- [x] `secretRequirements`の扱いの決定（§4.20。**S-1: childで照合しない**。byte一致が成り立てば欄ごとの照合は必ず通るため、保証が増えない。childの入力契約も`plugin.BuildPlan`も広げない）
- [x] **欄ごとの照合をここで打ち切る**決定（§4.20。原因究明の道具として既存は残すが、これ以上足さない。残る段差は欄照合では埋まらない）
- [x] **entryが計画側も独立に持つ**経路（[#3729](https://github.com/hat47x/tei-platform/pull/3729)）。lockの計画側はbuilderの申告を優先し、申告が無ければentryの再計算値を使う。どちらを使ったかを`planSideSource`で記録するので、弱い根拠を強い根拠と読ませない（[記録](../evidence/TEI_GO_SELF_BUILD_ENTRY_PLAN_SIDE_2026-11-09.md)）
- [ ] D-1（catalogの`scope`拡張）を採るかどうかの契約判断（採る場合のみ、catalogの広告と実応答の一致を照合へ載せる）

**A-1（`gobuilder`移動）は採らないため、影響範囲の洗い出しは終了条件から外した。** 29ファイルの内訳は[記録](../evidence/TEI_GO_INTERNAL_RULE_AND_BUILDER_MOVE_2026-11-07.md)として残す。I-1は「採らない」から「I-1aとして着手」へ改めた（§4.13）。`ParseFixtureCatalog`（fixture catalog）は`tools/`に残す。

**P-2（`reference-go-build-with-supply`への欄追加）は採らない。** 計画を解釈しないchildに欄を足すと`unperformed`が`matched`に変わるが、主張の強さは変わらない（§4.10.2）。

本書は計画の正本を増やすものではない。SB-C0の**入口の外形と拒否条件**だけを固定し、#2831のP0順序（SB-0a → SB-0b → SB-A → SB-B → SB-C）のうちSB-C0を詰めた設計候補である。最終判断は元計画・ADR・actual Evidenceへ返す。
