# セルフビルドの再入（SB-C0）の入口契約（設計候補）

- 対象：[#2831](https://github.com/hat47x/tei-platform/issues/2831) の **SB-C0**（再入contract）、§35.3のL1/L2/L3区分
- 関連：[#943](https://github.com/hat47x/tei-platform/issues/943)、[#979](https://github.com/hat47x/tei-platform/issues/979)、[#1080](https://github.com/hat47x/tei-platform/issues/1080)、[#1090](https://github.com/hat47x/tei-platform/issues/1090)、[#1097](https://github.com/hat47x/tei-platform/issues/1097)、[#1103](https://github.com/hat47x/tei-platform/issues/1103)、[#1121](https://github.com/hat47x/tei-platform/issues/1121)
- 作成日：2026-11-07
- 状態：**設計候補。採用・実装・qualified PASSではない**
- 前段：[SB-0bの供給境界](GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md)、[X1の記録](../evidence/TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)、[再現性の記録](../evidence/TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md)

**読み方。** 本稿は**書いた順に積んだ記録**であり、各節は**その時点の状態**を述べる。後の節が前の節の問いを閉じていることがあるため、中間の節にある「未着手」「未決」は現在の状態とは限らない。**残っている項目は§7の一覧が示し、そこへ至る最後の決定節が現在の判断を示す。** 節ごとの記述を今の状態として引用しない。

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

#### 補足：二回のbuildが同じbuilderだったことを記録する（2026-11-09）

本節の観測は「**同じchild実行物**を二回」である。しかし実装では、`artifactReproducibility.secondChildDigest`に**entry自身（`tei`）のdigest**が入っており、二回のbuilderが同じだったことを**比較する検査が無かった**（[記録](../evidence/TEI_GO_SELF_BUILD_REPRODUCIBILITY_BUILDER_2026-11-10.md)）。

修正して、`firstChildDigest`と`secondChildDigest`に**実際に起動したchildのdigest**を入れ、両者を比較するようにした。**違えば不成立、未記録も不成立**である（SC0-18と同じ規則）。食い違いの理由は`mismatchReason`に述べる。

**この修正は主張を弱めない。** 二回の間に実行物が差し替わった試行が`reproduced`になり得た点だけを塞ぐ。

同じ観測の残る前提である「**別state**」も、確かめていなかった。entryは二回目に新しい`-state`を渡すが、**childがそれを使ったかは見ていなかった**（`-state`を無視すれば二回は同じstateを共有し、同じ観測の繰り返しになる）。修正して、childが申告する`stateDir`が**渡した先と一致することを検査**するようにした。**申告が無ければ不成立**である（SC0-18と同じ規則）。

**これは「childがそう申告した」までである。** 申告どおりに使ったことの証明ではない（builderの他の申告と同じ限界）。

**この検査を足した時点で、実buildを走らせる`reference-go-build-with-supply`は`stateDir`を申告していなかった**ため、実buildの再現は`unperformed`になっていた。このchildは`-state`の中に`GOCACHE`・`GOMODCACHE`・`GOPATH`・`HOME`を置いて実際に分離しており、申告が無いだけであった。**申告を足して、実buildで`reproduced`が戻った**（[記録](../evidence/TEI_GO_SELF_BUILD_GO_BUILDER_STATE_2026-11-10.md)）。

**これはP-2（計画を解釈しないchildに`planFingerprint`欄を足す）とは違う。** stateの申告は「この実行が何をしたか」の事実であり、計画の解釈ではない。P-2を採らない判断は変わらない。

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

> **後続の節で決定済み。** 本節はP-4（`-strong`）を推したが、§4.30が「catalogを名指しすれば最強の観測が既定で揃う」を実現し、§4.34が**`-strong`を採らない**と決めた。本節のP-1〜P-4の比較は当時の材料であり、現在の決定ではない。既定の扱いは§4.34を参照。

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

**実buildの所要時間は測っていない。** 本節は構造と組み合わせを示しただけで、時間の実測を含まない。

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

### 4.27 `-rebuild-check`の実コストを測る（2026-11-09）

§4.25は「既定経路をどこまで強くするか」の三つの未決を一つにまとめ、**決めるために足りないもの**を三つ挙げた。本節はその一つ目、**実buildの所要時間**を測る。

#### 実測

同じ試行を`-rebuild-check`あり・なしで7回ずつ繰り返し、中央値で比べた（ウォームアップ1回の後）。

| 条件 | 中央値 | 最小 | 最大 |
| --- | --- | --- | --- |
| なし | **136.4 ms** | 118.3 ms | 177.7 ms |
| あり | **246.5 ms** | 230.8 ms | 327.9 ms |
| child単体 | **3.4 ms** | 3.1 ms | 3.9 ms |

| 値 | 結果 |
| --- | --- |
| 中央値の差 | **+110.1 ms** |
| 中央値の比 | **1.807倍** |

同時に、`artifactReproducibility.state`が`unperformed`から`reproduced`へ、`l2b.claimed`が`false`から`true`へ変わることも確認した。

#### 分かったこと：増分はbuildではない

**child単体は3.4 msしかないのに、増分は110.1 msである。** 二回目のコストの大半は**childの仕事ではなくentry側の固定費**である——一時ディレクトリの作成と削除、child argvの差し替えと起動、二回目のreport解析、artifactの再読とdigest計算。

**したがって§4.25の「build時間をほぼ倍にする」は、参照childについては正確ではない。** 参照childのbuildは3.4 msで、倍にしても6.8 msにしかならない。倍になるのは**entryの一回分の固定費**である。

#### 実buildではどうなるか（推論であり実測ではない）

増分は「entryの固定費 ＋ childの仕事」である。したがってbuildが重いほど比は2倍に近づく。

| childの仕事 | 増分 | 比 |
| --- | --- | --- |
| 3.4 ms（参照child、**実測**） | +110 ms | 1.8倍 |
| 数十秒（大きなGo build） | 約 +build時間 | **約2倍に近づく** |

**これは推論であり、実buildでの測定は行っていない。**

#### 判断への材料

| 問い | 本節が与える材料 |
| --- | --- |
| 参照childで既定にできるか | **+110 ms**。試行が136 msから246 msになる |
| 実buildで既定にできるか | **未測定。** 増分はbuild時間に比例して増える |
| 既定化の是非 | **決めない。** 費用は「誰の試行か」に依存する |

**「絶対値が小さいから既定にしてよい」とは言わない。** 110 msはこのPoCの最小のbuildでの値であり、Go buildを伴う試行では比が2倍に近づくため、**同じ判断を同じ根拠で下せない**。

#### 残る不足分

§4.25の三つのうち、本節が埋めたのは**一つ目だけ**である。

| 足りないもの | 状態 |
| --- | --- |
| 実buildの所要時間 | **参照childは測定済み**。実buildは未測定 |
| catalog 3点セットを既定で用意できるか | **未着手**（§4.22末尾の「catalogの中身を誰が作るか」と同じ問い） |
| `-strong`の名前と粒度 | **未着手** |

#### この節が言っていないこと

- 実buildの所要時間（測っていない）、別ホストの時間
- `-rebuild-check`を既定にすべきか。**費用の材料であり、判断ではない**

### 4.28 実buildで`-rebuild-check`の比は1.93倍になった（2026-11-09）

§4.27は参照childで`-rebuild-check`のコストを測り、「実buildでは増分がbuild時間に比例し、比は2倍に近づく」と**推論**した。本節はその推論を実buildで確かめる。

#### 実測

`tools/reference-go-build-with-supply`をbuilder childとして`tei self-build`を走らせた。このchildは**実際に`go build`を実行**し、`-artifact-out`と`-state`を尊重する（`-rebuild-check`が差し替えるのはこの二つである）。

| 項目 | 値 |
| --- | --- |
| 対象package | `./internal/core`（外部module依存なし） |
| 環境 | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| 試行 | 5回ずつ、ウォームアップ1回の後、中央値 |

| 条件 | 中央値 | 最小 | 最大 |
| --- | --- | --- | --- |
| なし | **5852.7 ms** | 5562.9 ms | 6739.0 ms |
| あり | **11272.4 ms** | 10406.8 ms | 11792.5 ms |
| child単体（Go build） | **5568.2 ms** | 5352.4 ms | 6770.8 ms |

| 値 | 結果 |
| --- | --- |
| 中央値の差 | **+5419.7 ms** |
| 中央値の比 | **1.926倍** |

#### §4.27の推論は当たっていた

| 予想（§4.27） | 実測（本節） |
| --- | --- |
| 増分は「entryの固定費 ＋ childの仕事」 | 増分5419.7 ms ≒ child単体5568.2 ms ＋ 固定費 |
| buildが重いほど比は2倍に近づく | **1.926倍** |
| 参照childでは固定費が支配的（+110 ms、1.807倍） | 実buildでは**buildが支配的**（+5.4秒、1.926倍） |

**同じflagが、childの重さによって意味の違う費用になる。** 参照childでは「entryの段取り代」、実buildでは「buildをもう一度」である。

#### 副産物：Go buildのartifactも再現した

`artifactReproducibility.state`は**`reproduced`**で、二回のbuildのdigestが一致した（`sha256:1bcf33c2…`）。**plugin packageだけでなく、Go buildの成果物も同じplanから同じバイトになった。** §4.21の再現は参照childに限った話ではなかったことになる。

`l2b.claimed`は`false`のまま。entryが解釈していない（catalogを渡していない）ためで、述語どおりの動作である。

#### 既定化の判断に効くこと

| 問い | 材料 |
| --- | --- |
| 参照childで既定にできるか | +110 ms（§4.27） |
| **実buildで既定にできるか** | **+5.4秒。試行が5.85秒から11.27秒になる** |
| `-rebuild-check`を既定にすべきか | **依然として決めない。** 費用が2桁違う |

**「実buildでは約2倍」が実測で裏づけられた。** 既定化は「すべての試行の壁時計時間をほぼ倍にする」判断になる。§4.25のP-2（再現だけ既定）は、この費用を全試行に課すことになる。

#### この節が言えないこと

- 別のpackage・別ホストでの時間。`./internal/core`は小さく、重い対象では絶対値が変わる
- 既定化の是非。**費用の材料であり、判断ではない**
- catalog 3点セットの出所・`-strong`の名前と粒度（§4.25の残る二つの不足分。未着手）

### 4.29 catalogの欄はリポジトリのどこにも宣言されていない（2026-11-09）

§4.25は判断に足りないものとして「**catalog 3点セットを既定で用意できるか**」を挙げた。本節はその問いを、**catalogの各欄がどこで宣言されているか**という形で測る。

#### 実測

`BuildProvider`の7欄について、`internal/`と`cmd/`から`"欄名"`を検索し、**catalogの型定義以外に宣言面があるか**を確かめた。

| 欄 | catalogの型以外の宣言面 |
| --- | --- |
| `contractMajor` | **無い** |
| `authorityScope` | **無い** |
| `recipeKinds` | **無い** |
| `profiles` | **無い** |
| `targets`（build対象） | **無い**（`internal/index`と`internal/modelref`の`"targets"`は**意味関係の名前**で無関係） |
| `artifactDigest` | **ある**（`internal/plugin`のlock、宣言の`builder.pin`、runtime service carrier） |
| `contract`（名前） | 宣言にある。ただし**「どのproviderがどのcontractを提供するか」**はどこにも無い |

**四つ（`contractMajor`・`authorityScope`・`recipeKinds`・`profiles`）は、catalogの型・その試験・`cmd/tei`の試験用catalogにしか現れない。** `targets`も同じで、他の出現は別概念だった。

#### 何が分かったか

**catalogは既存の宣言から導出できない。** 7つのうち5つに宣言面が無いためである。

| 案 | 何が要るか | 評価 |
| --- | --- | --- |
| 既存の宣言から生成する | 5欄の出所が無い | **不可** |
| `plugin`定義にbuilder能力を宣言させる（§4.22 K-2） | **構文拡張** | 可能だが構文と意味の追加 |
| Workspaceにcatalog欄を設ける（§4.22 K-1） | `tei.workspace.yaml`の拡張 | 可能。Workspaceが「どのbuilderが在るか」の正本になる |
| 外部の配布物として持つ | catalog自体を成果物にしdigestで名指しする | **K-3が既にこれを前提にしている** |

**「既定で用意できるか」の答えは「今のリポジトリの内容からは用意できない」である。** 用意するには**宣言面を一つ増やす**決定が要る。

#### これは配線の問題ではない

§4.25の表現を借りれば、この不足分は「誰がcatalogを書くか」ではなく、**catalogを書く場所を製品に作るか**という問いである。

§4.22は「**entryがcatalogをどこから読むか**」をK-3（呼び手が名指しする）に決めた。本節はその決定を変えない。変わるのは**「そのcatalogを誰がどう作るか」が未決のまま残る**ことの重みである——既存の宣言からは作れないので、**明示的に作る場所を決めるまで、既定の試行でClaimは立てられない**。

#### この節が言っていないこと

- **どの案が良いか。** K-1・K-2・外部配布物のどれを採るかは契約判断であり、本節は判断しない
- 外在するconsumerの扱い
- catalogの中身が正しいこと（§4.22 K-3、SC0-26）
- `-strong`の名前と粒度（§4.25の三つ目の不足分。未着手）

### 4.30 catalogを名指ししたら、二回目のbuildも既定で行う（2026-11-09）

§4.25は既定経路の強化を四案に整理し、P-4（`-strong`で強さを選ぶ）を推したが決定しなかった。§4.27・§4.28が費用を測り、**参照childで+110 ms、実buildで1.93倍**と分かった。

本節は**一つの場合に限って既定を強くする**。

#### 決めたこと

**`-build-catalog`を渡した試行は、`-rebuild-check`を明示しなくても二回目のbuildを行う。** `-rebuild-check=false`を明示すれば従来どおり一回で終わる。

| 呼び手がしたこと | 二回目のbuild | 理由 |
| --- | --- | --- |
| `-build-catalog`を渡した（`-rebuild-check`に触れない） | **行う** | その経路だけがL2-bを立てられる |
| `-build-catalog`と`-rebuild-check=false` | 行わない | 呼び手が明示的に断った |
| `-build-catalog`と`-rebuild-check` | 行う | 明示 |
| `-intent-resolver`（catalogなし） | **行わない** | その経路ではClaimが立たない |
| どちらも無し | 行わない | 計画を導出しない |

#### なぜこの一つだけか

**catalogを名指しすることは、既に「強い経路を選ぶ」という表明である。** §4.22のK-3により、catalogは呼び手が用意するデータであり、渡すには手間がかかる。その手間を払った呼び手に対して、**同じ経路で取れる最後の観測を既定で取る**のは、費用と便益が同じ側にある。

**child経路は違う。** そちらは`l2b.claimed`が立てられないので、二回目のbuildを足しても費用だけが増える。§4.25が「P-2は非対称」と呼んだのはこの点であり、本節はその非対称を**既定に持ち込まない**形にした。

#### 何が良くなるか

これまでは、catalogを渡した呼び手が`-rebuild-check`を書き忘れると、**何も言われずにClaimが立たなかった**。`l2b.unmet`には理由が出るが、呼び手は「渡したのに立たない」ことに後から気づく。本節の変更で、**catalogを渡せばClaimが立つ**のが既定になる。

費用は実buildで約2倍（§4.28）。**catalogを渡す呼び手だけが払う。**

#### 何が変わらないか

| 項目 | 状態 |
| --- | --- |
| 既定の導出経路 | **child のまま**（R-2据え置き。in-processは呼び手が名指しする） |
| `-intent-resolver`の試行 | 二回目のbuildは既定で行わない |
| 何も渡さない試行 | 計画も導出せず、二回目のbuildも行わない |
| `qualified` | `false` のまま |
| catalogの出所 | §4.22 K-3のまま。**中身を誰が作るかは未決**（§4.29） |

**「既定の試行でClaimが立つ」ようになったわけではない。** catalogを渡すという一手が要る。§4.29が示したとおり、そのcatalogを用意する場所はまだ製品に無い。

#### 覆す条件

- catalogを渡す呼び手が、二回目のbuildを**常に**不要とする場合（費用が受容できない）
- 二回目のbuildが**失敗しやすい**場合（部分的な成果を返すchildなど）——このときは`-rebuild-check=false`を既定にする

いずれも`-rebuild-check=false`で回避できるため、**既定を戻すのは一行の変更**である。

#### 実測

[記録](../evidence/TEI_GO_SELF_BUILD_INPROCESS_DEFAULT_EVIDENCE_2026-11-09.md)。`-rebuild-check`を**渡さずに**`l2b.claimed=true`になること、`-rebuild-check=false`で`unperformed`の理由が「呼び手が断った」に変わることを確かめた。

### 4.31 `l2b.claimed`に必要なflagを実測で確かめる（2026-11-09）

§7は`l2b.claimed`について「今は呼び手がin-processと`-rebuild-check`を明示する必要がある」と書いていた。**§4.30の後、後半が成り立たない。**

#### 何が変わったか

§4.30は「`-build-catalog`を渡した試行は、`-rebuild-check`を明示しなくても二回目のbuildを行う」と決めた。五条件の五つ目（再現）が**`-build-catalog`だけで満たされる**ことになる。

#### 実測

**`-build-catalog`のみ（`-rebuild-check`を渡さない）**

| 項目 | 結果 |
| --- | --- |
| `artifactReproducibility.state` | **`reproduced`** |
| `l2b.claimed` | **`true`** |
| `l2b.unmet` | 空 |

**`-rebuild-check=false`を明示**

| 項目 | 結果 |
| --- | --- |
| `state` | `unperformed` |
| 理由 | **`-rebuild-check=false`を名指し**（「渡されなかった」ではなく「呼び手が断った」） |
| `l2b.claimed` | `false` |

**断り方が区別されている。** 同じ`unperformed`でも、flagが無い場合と明示的に断った場合で理由文が違う。

#### 記述の訂正

| 旧 | 新 |
| --- | --- |
| in-process と`-rebuild-check`を明示する必要がある | **in-process経路を明示すれば足りる**（再現は§4.30が既定にした） |

#### 残る問い

**「何も渡さない試行」でclaimを立てるかは未決である。** 現在の状態は次のとおり。

| 試行 | claim |
| --- | --- |
| 何も渡さない | 立たない（計画を導出しない） |
| `-build-catalog`一式 | 立つ（五条件が揃えば） |
| `-intent-resolver` | 立たない（entryが解釈しない） |
| `-build-catalog` + `-rebuild-check=false` | 立たない（呼び手が断った） |

**既定化には§4.29の「catalogを書く場所」の決定が前提になる。** 何も渡さない試行にcatalogを供給できなければ、その試行は計画を導出できず、claimも立てられない。

#### この節がしていないこと

- **§7の判断をしていない。** 記述の訂正と実測だけである。
- 「既定でclaimを立てるべき」とは言っていない。§4.29の決定を待つ。

### 4.32 catalogを書く場所は製品の外に置く（2026-11-09）

§4.29は、catalogの7欄のうち5欄に**宣言面が無い**ことを示した。本節は「ではどこに書くか」を、**既存readerへの影響**という軸で決める。

#### 測ったこと：三つの案は、どれも既存readerを壊しうる

| 案 | 何を変えるか | 既存readerへの影響（実測） |
| --- | --- | --- |
| **K-1** Workspaceに`builders:`を足す | `tei.workspace.yaml` | **壊れる。** manifest parserは未知のentryで**fail closed**する（`internal/workspace/imports.go`の`unsupported workspace manifest entry`）。さらに`formatVersion`は1と2だけを受理し、未知の版もfail closed（`internal/workspace/loader.go`）。**古いbinaryがWorkspaceを読めなくなる** |
| **K-2** plugin定義に構文を足す | `.tei`の構文 | **壊れる。** parserは構文ごとのfield集合を明示し、未知のfieldでエラーにする（`internal/syntax/parser.go`の`unknown lease-policy field`と同じ形）。**古いbinaryがplugin定義を検証できなくなる** |
| **外部成果物** catalogを独立したファイルとして渡し、digestで名指しする | **何も足さない** | **壊れない。** 既存のどのファイルにも新しいentryが増えない |

**Workspaceもplugin定義も、未知の入力を拒否する設計である。** これは「分からないことを成功として扱わない」という本リポジトリの原則そのもので、正しい。だからこそ、**既存のファイルへ欄を足す案は、古いbinaryとの互換を壊す**。

#### 決めたこと

**catalogは製品の外で作り、呼び手がdigestで名指しする。** §4.22のK-3が既にこの形を実装しているので、**製品側の変更は要らない**。

| 項目 | 決定 |
| --- | --- |
| catalogの置き場所 | **製品のファイル形式の外**（独立したJSON） |
| 誰が作るか | **ツール**。能力宣言を入力に取り、catalogを組み立ててdigestを付ける |
| entryの読み方 | 変わらない（`-build-catalog`＋`-build-catalog-digest`＋`-build-source-root`、K-3） |
| Workspaceの`formatVersion` | 変えない（1と2のまま） |
| plugin構文 | 変えない |

**製品の契約を一つも変えずに、catalogを用意できるようにする。** これが「最短経路」である。

#### なぜこれで足りるのか

catalogは**データ**であり、entryはそれを**名指しされたものとして読む**（§4.22が「entryはbuilderを自分で探さない」と決めた）。**誰がそのデータを作ったかは、entryの観測の対象ではない。** 作る手段が製品の外にあっても、entryの振る舞いは変わらない。

能力宣言（どのcontractを、どのrecipe kindで、どのtarget/profileに向けて提供するか）は、**catalogを作る者の入力**であって、Workspaceの意味でもpluginの意味でもない。だから製品の構文に入れない。

#### この決定が言っていないこと

- **能力宣言の形式**（ツールの入力の形）。本節は決めない
- **catalogが正しいこと**。§4.22 K-3、SC0-26のまま。呼び手が名指ししたデータを使った、という記録しかない
- **誰が能力を宣言してよいか**（権限）。catalogは「buildできると名乗るデータ」であり、**その名乗りの妥当性は本節の範囲外**
- 既定の試行でのClaim（§4.30のまま。catalogを渡す一手が要る）

#### 覆す条件

- 能力宣言を**Workspaceの意味として扱う必要が出た場合**（例：Workspaceの妥当性検査がbuilderの有無に依存するようになる）。そのときは`formatVersion`の改版と一緒にK-1を採る
- 能力宣言を**pluginの意味として扱う必要が出た場合**（例：pluginの起動時に自分のbuild能力を検証する）。同じくK-2を版付けで採る

いずれも**後から足せる**。逆（製品に足してから外す）は互換を壊すので、順序はこちらが正しい。

### 4.33 capability declarationからcatalogを作る（2026-11-09）

§4.32はcatalogを**製品の外のデータ**と決めた。その決定が残した「誰が書くか」を、`tools/reference-build-catalog-authoring`が埋める。

#### 何をするか

**能力宣言**（build toolが何をできるか）を読み、**artifactのdigestを実ファイルから計算して**、catalog JSONを書く。

| 欄 | 扱い |
| --- | --- |
| `plugin`・`version`・`implementation`・`contract`・`contractMajor`・`recipeKinds`・`targets`・`profiles` | 宣言から写す |
| `authorityScope` | **書かせない。** 常に`build-tool`（build toolとして名乗れる値は一つだけ） |
| `artifact` | `-root`からの**相対パス**。このファイルのdigestが`artifactDigest`になる |
| `artifactDigest` | **書かせない。** 書くと拒否する |

**digestを宣言させないことが要点である。** 手で書いたdigestを許すと、リポジトリに無いbuilderを名乗るcatalogが作れてしまう。ツールは実ファイルを読み、そのバイトのdigestを入れる。

#### 拒否するもの

| 入力 | 理由 |
| --- | --- |
| 宣言の`artifactDigest` | 読んでいないバイトを名乗れなくする |
| `-root`の外を指す`artifact` | ルートが持たないファイルをcatalogに載せない |
| 存在しない・通常ファイルでない`artifact` | 同上 |
| 既存の`-output` | 上書きしない |
| 組み立てたcatalogが`ParseBuildCatalog`で読めない | entryが読めないcatalogを「catalogがある」と誤解させない |

#### 実測：宣言からClaimまで一本で通った

| 段 | 結果 |
| --- | --- |
| 能力宣言を書く | 1 provider |
| ツールでcatalogを作る | `catalogDigest=sha256:6d1e5929…`、`artifactDigest=sha256:84db7752…` |
| **宣言のartifactと実ファイルのdigest** | `sha256:84db7752…` で**一致**（`sha256sum`で確認） |
| 参照実装がそのcatalogを解決 | `planFingerprint=a6b756b8…` |
| **`tei self-build`（`-rebuild-check`を書かない）** | `derivedBy=entry-in-process`・`artifactReproducibility=reproduced`・**`l2b.claimed=true`**・`qualified=false` |

**§4.29で「catalogを書く場所が無い」と分かってから、製品の契約を一つも変えずに、宣言からL2-bのClaimまでが一本で通った。**

#### この形式は製品の形式ではない

能力宣言には**互換性の約束も`formatVersion`も無い**。製品が読むのは、このツールが出力するcatalogだけである。宣言の形を変えても、catalogの形（`BuildCatalog`）を変えなければ製品には影響しない。

#### 回帰試験に載せた

**宣言からClaimまでの鎖は、`cmd/tei`のエンドツーエンド試験が毎回通す。** 試験は参照catalogをコピーせず、**ツールでcatalogを作る**。

| 段 | 試験が確かめること |
| --- | --- |
| 能力宣言 | builder childの実体を宣言が名指しする |
| authoring | ツールがcatalogを書き、`catalogDigest`を返す |
| **catalogの中身** | **`artifactDigest`が、実際に走るbuilderのバイトのdigestと一致する** |
| `self-build` | `l2b.claimed=true`・`unmet`空・`qualified=false` |

**digestの一致を試験で確かめるのは、下流が気づけないからである。** builderの解決はdigestで照合しないため、catalogが別のバイトを名指ししていても試行は通ってしまう。鎖の意味は「名指ししたバイトが実際に走る」ことにあるので、そこを固定した。

#### 二つのホストで同じcatalogになる

同じ宣言と**同じバイトのbuilder**から、`linux/amd64`と`windows/amd64`でcatalogを作った。

| 値 | 結果 |
| --- | --- |
| `catalogDigest` | **両ホストで同じ**（`sha256:f7e1c879…`） |
| catalogのバイト数 | 667（同じ） |
| `artifactDigest` | **両ホストで同じ**（`sha256:cabf793a…`） |

**catalogのバイトが一致するのは、`artifact`が相対パスで、ツールが絶対パスをcatalogへ書き込まないからである。** 同じ宣言と同じbuilderがあれば、どのホストでも同じ`catalogDigest`を呼び手に渡せる。

鎖そのものも**WindowsのGoで直接**通した（`TestSelfBuildEstablishesL2BEndToEnd`が25.13秒でPASS）。観測したのは`linux/amd64`と`windows/amd64`だけで、`darwin`・`arm64`は未観測である。

#### この節が言っていないこと

- **catalogが正しいこと**（§4.22 K-3、SC0-26のまま）。名指しされたデータを使った記録しかない
- **誰が能力を宣言してよいか**（権限）。catalogは「buildできると名乗るデータ」であり、名乗りの妥当性は範囲外
- **builderの実装が正しいこと**。digestを計算するだけで中身は見ない
- 既定の試行でのClaim（§4.30のまま。catalogを渡す一手が要る）
- 能力宣言の**版付け**。必要になった時点で決める
- **すべてのOSでの成立**。二つのホストを観測しただけである

### 4.34 既定に関する残りの判断を閉じる（2026-11-09）

§7に残っていた既定の問いに、**測った材料の範囲で**答える。いずれも**今の振る舞いを変えない**判断で、覆す条件を添える。

#### 判断1：`-rebuild-check`をすべての試行で既定にしない

| 材料 | 内容 |
| --- | --- |
| 費用（§4.27） | 参照childで+110 ms（1.807倍） |
| 費用（§4.28） | 実buildで+5.4秒（**1.926倍**） |
| 効果 | child経路では`l2b.claimed`が**立てられない**（entryが解釈していない） |

**child経路に二回目のbuildを既定で足すのは、費用だけを倍にして主張を増やさない。** §4.25が「P-2は非対称」と呼んだとおりである。§4.30で**catalogを名指しした試行に限って**既定にしたので、Claimを立てたい呼び手は既に自動で最強の観測を得る。

**決定：既定のまま（catalogを名指ししたときだけ行う）。**

**覆す条件**：child経路がL2-bを立てられるようになったとき。

#### 判断2：既定の導出経路をin-processへ切り替えない（R-1は据え置き）

in-processを既定にするには、**entryがcatalogを自分で選ぶ**必要がある。しかし§4.22は「entryはbuilderを自分で探さない」と決め、K-3でcatalogは**呼び手が名指しするデータ**とした。§4.29はcatalogの5欄に宣言面が無いことを示し、§4.32はcatalogを製品の外に置いた。

**したがって「既定でin-process」は、entryが名指しされていないデータを読むことに等しく、K-3を取り消さない限り実装できない。**

**決定：R-1は据え置き。既定はchild経路のまま。** 強い経路は呼び手が`-build-catalog`一式で選ぶ。

**覆す条件**：呼び手が**一度だけ設定する**catalogの場所が製品の契約として決まったとき（Workspaceの`formatVersion`改版を伴う決定）。

#### 判断3：`l2b.claimed`を既定の試行で立てない

判断2の帰結である。既定はchild経路で、その経路ではentryが解釈しないので五条件の一つ目が成立しない。

**決定：既定では立てない。** Claimは**呼び手の行為**であり、`-build-catalog`一式を渡すことがその表明になる。§4.30により、渡した時点で残りの観測は自動で揃う。

**覆す条件**：判断2を覆すとき。

#### 判断4：D-1（catalogの`scope`拡張）を採らない

| 材料 | 内容 |
| --- | --- |
| 既存consumer | `tools/reference-mcp-readonly-adapter`が`effect=read-only`を前提にしている |
| `scope`の意味 | `selected-read-only-commands`。「選んだ読取り専用」であって「dispatchする全部」ではない |
| 外在consumer | **測れない** |
| 失われるもの | `-h`と文書で知れるので**発見性だけ**。§4.33でヘルプに全flagが出るようにした |

**決定：D-3のまま（catalogへ載せない）。** `scope`を広げると、外在consumerの前提を測れないまま変えることになる。発見性は`-h`と`reference/plugin-self-build/README.md`で確保した。

**覆す条件**：外在consumerの一覧ができ、`scope`ではなく`effect`を見ることを契約に書けるようになったとき。

#### 閉じないもの

| 項目 | 状態 |
| --- | --- |
| `darwin`・`arm64`・別のGo版の観測 | **未観測のまま。** 本環境では実行できない |
| 能力宣言の版付け | **未決のまま。** 第二のconsumerが現れた時点で決める |
| `-strong`（§4.25のP-4） | **採らない。** §4.30が「catalogを名指しすれば最強の観測が既定で揃う」を実現したので、強さを選ぶ別flagは要らない。第二の強さの段が要るようになったら再考する |

### 4.37 L2-bの五条件は、何を確かめ、何を信じているか（2026-11-09）

§4.23の述語は五条件の同時成立を要求する。本節は、**各条件がどこまで検査で、どこからが自己申告か**を一か所にまとめる。

**§4.21・§4.24・§4.36で三つの穴を塞いだ結果、条件ごとの中身が変わっている。** 読む人が「欄があるから確かめた」と読まないよう、現状を固定する。

#### 条件ごとの内訳

| 条件 | 検査（entryが自分で見る） | 自己申告（childの言い分） | 塞いだ節 |
| --- | --- | --- | --- |
| 1. entry自身が計画を導いた | `derivedBy`・`state`・`unmet`の整合 | — | §4.22段2 |
| 2. 宣言byteの一致 | 宣言のdigest（entryが保持）と、childが申告したdigestの**比較** | **childが実際にそのbyteを読んだこと** | §4.14 |
| 3. 計画fingerprintの一致 | 二つの値の**比較**（両方とも空なら`unperformed`） | **childの計画fingerprint** | §4.9・§4.11 |
| 4. plan→artifactの束縛 | **ディスクのbyteのdigest**と、lockの整合・検証 | childの`artifactDigest` | §4.11・§4.12 |
| 5. artifactの再現 | 二回の**バイト一致**、**builderの同一性**、**stateの分離**、ディスクの一致 | childの`stateDir`・`artifactDigest` | §4.21（#4002・#4015）・§4.36 |

**「直近2件」は #4002 と #4015 を指す。** 本節を書いた時点で、`cmd/tei/cli_self_build_reproducibility.go` を最後に変更した2件がこの二つであり（`git log` で確認できる）、どちらも §4.21 の変更として記録されている。**欄が未完成のまま残っていたので、名指しに置き換えた。**

#### 検査で止められるもの

| childがしようとしたこと | 止まる場所 |
| --- | --- |
| 宣言と違うバイトを読んだ**と申告する** | 条件2で不一致 |
| 計画と違うfingerprintを申告する | 条件3で不一致 |
| 成果物と違うdigestを申告する | 条件4（ディスクとの比較）で不一致 |
| 同じバイトを二回出さない | 条件5で不一致 |
| 二回の間に実行物を差し替える | 条件5の**builder同一性**（**`-child`に名指ししたファイルのbyte**。そのファイルが別の実行物を起動するプロキシだと、背後での差し替えは**条件5では**止まらない——[実測](../evidence/TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md)。宣言したbuilderと違うファイルを起動した場合は、§4.42の`builderBinding`がClaimを止める） |
| `-state`を無視して同じstateで二回buildする | 条件5の**state分離**（申告が合わない） |
| **計画と違うdefinitionを読んでbuildする** | childが**成果物を作る前に停止**（§4.36） |

#### 検査で止められないもの（信じている部分）

**childの「自分が何をしたか」という申告は、すべて信じている。** 具体的には:

1. **宣言を実際に読んだこと**（条件2は「読んだ」という申告のdigestを比べるだけで、読んだ行為は見えない）
2. **計画から成果物を作ったこと**（条件4は「申告したdigest」と「ディスクのbyte」を比べる。ディスクのbyteが計画から出たかは、条件5の再現と§4.36のsource照合で**間接的に**支えるだけ）
3. **使ったstate**（条件5は申告が渡した先と一致するかを見る。申告どおりに使ったかは見えない）
4. **§4.36の停止が働くこと**（childが自分で止まる。止まらないchildは、そもそもこの検査を実装していない）

**つまりL2-bは「entryの観測」と「childの申告」の合成であり、childを敵と見なす安全性ではない。** これは§4.8がL2-bを「解釈の層」として定義したことの帰結である。

#### 敵対的なchildを想定するなら何が要るか

| 必要になるもの | 何を閉じるか |
| --- | --- |
| OSレベルの読み取り制限（計画のsourceだけをbind mountする） | 申告1（読んだこと） |
| childの実行をentryが監視する（open/execveの記録） | 申告2（作ったこと） |
| stateディレクトリの実体をentryが用意し、childには渡さない | 申告3（state） |
| §4.36の照合を**entry側でも**持つ（childの申告に依存しない） | 申告4（停止が働くこと） |

**いずれも本設計の範囲外である。** §4.8はL2-bを「宣言が計画として解釈され、その計画がartifactを生んだこと」の**解釈層**として定義しており、敵対的builderに対する防御は別の層（供給のTrust、Seed Trust、Runtime Authority）の仕事である。

#### この節が言っていないこと

- childが**正直であること**の保証。していない
- 敵対的なbuilderへの防御。**層が違う**
- 受理・Runtime Authority・供給のTrust。いずれも未主張のまま

### 4.36 計画が名指すsourceを、builderが確かめる（2026-11-09）

§4.8(c)は「その計画がそのartifactを生んだ」を主張します。本節は、その連鎖に**入力の照合が無かった**ことを記録し、塞いだ結果を残す。

#### 見つけたこと

計画は`Sources`に**source closureのdigest**を持ちます。entryは`-build-source-root`からsourceを解決してこのdigestを作ります。**ところがbuilder childは、計画が名指すdigestを入力と突き合わせていませんでした。**

| 観測 | 「計画がSを名指し、childがS'を読む」とき |
| --- | --- |
| `planCrossCheck=matched` | **成立する。** 計画fingerprintはchildが計画ファイルから取る |
| `planArtifactBinding=bound` | **成立する。** 成果物digestはchildの申告とディスクが一致する |
| `artifactReproducibility=reproduced` | **成立する。** 二回とも同じS'を使えば同じバイト |

**どれも「入力が計画のものか」を見ていない。** 三つの観測がすべて通ったまま、計画が記述しない成果物ができる。

#### 直したこと

**childが、読んだdefinitionのdigestを計画の値と突き合わせ、違えば成果物を作る前に停止する。**

| 変更 | 内容 |
| --- | --- |
| `buildintent.SingleFileSourceDigest` | 単一ファイルsourceのdigest。**resolverと同じ枠組み**で計算する |
| child | 計画の`plugin-definition` sourceと比較し、**違えば停止**。計画がdigestを持たない場合も停止 |
| 報告 | `definitionBytesDigest`と`definitionMatchesPlan` |

**照合をchildに置いたのは、計画と入力の両方を持つ唯一の場所だからである。** entryはchildが何を読んだか見えない。

**枠組みを一か所にしたのは、二か所に書くと必ず食い違うからである。** 当初は生の`sha256`で比較して食い違い、resolverの枠組み（メンバー名と長さを先に書く）を知って直した。

#### 副産物：fixtureが存在しないbuildを記述していた

childの試験の計画が`sha256:aaaa…`という**実在しないdigest**を名指ししていた。検査が無かったので通っていた。**fixtureが実在しないdigestを持っていたこと自体が、この検査が無かった証拠**である。実ファイルのdigestに直した。

#### 補足：同じ照合をentry側にも置く（2026-11-09）

上の照合は**builderが自分で止まる**ことで成り立つ。**その検査を実装していないbuilderは、この穴を塞がない。** §4.37の「信じている4項目」のうち「§4.36の停止が働くこと」がこれに当たる。

そこで、childが申告する`definitionBytesDigest`を**entryが自分の計画の値と比較**するようにした（`definitionBinding`）。不一致は**試行の失敗**として扱い、両方のdigestを記録に残す。**未申告は`unperformed`**（一致として扱わない）。

**これで「§4.36の停止が働くこと」は信じる対象から外れた。** 残る信じている項目は、宣言を実際に読んだこと・計画から成果物を作ったこと・申告どおりのstateを使ったことである。

#### この節が言っていないこと

- **childが本当にそのファイルを読んだこと**（digestはchildの申告である）
- **ディレクトリのclosure**の照合。今回は単一ファイル（`plugin-definition`）だけを塞いだ
- 計画が名指す他のsource（`go-package`など）の照合
- 別ホスト・別Go版での成立

### 4.35 能力宣言に版を付けない（配るのはcatalogだから、2026-11-09）

§4.33で足した能力宣言について、§7は「版付けは必要になった時点で決める」としていました。しかし**実測すると、必要になった時点では決められません**（[記録](../evidence/TEI_GO_SELF_BUILD_DECLARATION_VERSIONING_2026-11-10.md)）——宣言は`DisallowUnknownFields`で解析されるので、`formatVersion`を足した宣言は**版を名乗ろうとした瞬間に拒否**されます。

本節は、その未決を**「宣言を誰に配るか」という問いとして決めます。**

#### 測ったこと：版を持つべきなのは、配る側である

| 対象 | 配るか | 版の扱い（実測） |
| --- | --- | --- |
| **catalog**（entryが読む） | **配る**。呼び手が名指しし、digestで固定する | `formatVersion`を持ち、**未知の版を拒否する**（`unsupported build catalog format 2`） |
| **能力宣言**（ツールの入力） | **配らない**。ツールと同じリポジトリに置く | 版を持たない。キー集合に一致しない入力を拒否する |

#### 決定

**能力宣言に版を付けない。配るのはcatalogであり、catalogは既に版を持つ。**

| 理由 | 内容 |
| --- | --- |
| 配る側が版を持つ | 版を必要とするのは「作った時と読む時が離れている」データである。catalogはその条件を満たし、宣言は満たさない |
| 宣言はツールと同じ場所で変わる | ツールを上げるときに宣言も同じリポジトリで直す。版を名指す相手がいない |
| **版を足すこと自体が拒否される** | 後から足すと、その時点で宣言を持つ全員にとって破壊的変更になる（実測） |

**宣言を配ることになった場合は、配る前に`formatVersion`を足す（D-2）。** これは「後から決める」ではなく「**配る前に決める**」であり、判断の時期を「必要になった時点」から「配布の決断時」へ移す。

#### 実測

| 入力 | 結果 |
| --- | --- |
| 版を書かない宣言 | **成功**。catalogを生成 |
| `formatVersion`をトップレベルに足した宣言 | **`declaration: json: unknown field "formatVersion"`** で拒否 |
| `formatVersion: 2`のcatalog | **`refused: unsupported build catalog format 2`** |

**配る側（catalog）は未知の版を拒否し、配らない側（宣言）は未知のキーを拒否する。** どちらも「分からない入力を成功として扱わない」という同じ原則の現れである。

#### この節が言っていないこと

- 宣言を**配らない**ことの妥当性は、§4.32の「製品の外のデータ」という決定に依拠する。**配布する製品になった場合は本節を覆す**
- catalogの版を**上げる方法**（版番号の意味、移行手順）は決めていない
- 誰が能力を宣言してよいか（権限）は範囲外のまま

### 4.38 どのchildが、どの条件を満たせるか（2026-11-09）

L2-bの五条件は、childの能力に依存する。本節は、**参照実装のどれがどの条件を満たせるか**を一覧にする。

**§4.21でstate分離の検査を足したとき、実buildの観測が黙って`unperformed`になった**（[記録](../evidence/TEI_GO_SELF_BUILD_GO_BUILDER_STATE_2026-11-10.md)）。childが申告しない欄を条件にすると、**そのchildを使った観測は止まる**。条件を足す前に、この表を確かめる。

#### 一覧

| child | 条件1（in-process） | 条件2（宣言byte） | 条件3（計画fingerprint） | 条件4（plan→artifact） | 条件5（再現） |
| --- | --- | --- | --- | --- | --- |
| `reference-plugin-package-build-child` | 呼び手が`-build-catalog`を渡せば満たす | **満たす**（`-intent`で宣言を読む） | **満たす**（`-plan`で計画を読む） | **満たす**（lockを自分で作る） | **満たす**（`stateDir`を申告） |
| `reference-go-build-with-supply` | 同上 | **満たさない**（宣言を読まない） | **満たさない**（計画を解釈しない） | **満たさない**（lockを作らない） | **満たす**（`stateDir`を申告。§4.21補足） |
| `reference-go-build-reproducibility` | — | — | — | — | **満たす**（`stateDir`を申告） |

**`l2b.claimed`が立つのは、上の表で五条件すべてを満たすchildだけである。** 現在それは`reference-plugin-package-build-child`だけであり、**これはplanを解釈する参照実装**であって、TEI自身をビルドする道具ではない。

#### この表が意味すること

**TEI自身（Goのバイナリ）を、五条件を満たして自己構築することは、今はできない。** 理由はflagの不足ではなく、**計画を解釈してGoをコンパイルするbuilderが存在しない**ことである（§4.10.2のP-2を採らない判断と同じ理由）。

**最短経路でこれを実現するには、次の三つを持つbuilderが要る。**

| 必要なもの | 何のためか |
| --- | --- |
| 計画を読む（`-plan`） | 条件3のfingerprintを自分の入力から出す |
| 宣言を読む（`-intent`） | 条件2のbyte digestを自分の読みから出す |
| 計画のsource closureから`go build`する | 条件4・5を自分の成果物で満たす（§4.36の照合もここに入る） |

**`reference-go-build-with-supply`に欄を足すのでは足りない。** それは転記であり、§4.10.2がP-2として退けた案である。

#### 検査を足すときに確かめること

1. **既存の観測が止まらないか。** 新しい条件がchildの申告を要求するなら、そのchildが申告するか（本節の表）
2. **止まるのが正しい停止か。** 止まること自体は悪くない。悪いのは、**止まったことに気づかない**ことである（§4.21の場合は実buildの観測が失われた）
3. **そのchildを使う試験があるか。** 無ければ、止まったことをCIが知らせない

#### この節が言っていないこと

- 表に無いchildの能力。**測ったものだけを載せている**
- 三つを持つbuilderの実装。**まだ無い**
- 受理・Runtime Authority・供給のTrust。いずれも未主張のまま

### 4.39 TEI自身をビルドする計画の形（2026-11-09）

§4.38は「五条件を満たしてTEI自身を自己構築することは今はできない。理由はflagの不足ではなく、**計画を解釈してGoをコンパイルするbuilderが存在しない**こと」で終わった。本節は、**そのbuilderが受け取る計画を先に決める。**

#### 測ったこと：計画の型は既に「commandをビルドする」形を受ける

次の計画を`plugin.BuildPlan`として解析し、`Validate()`と`LockBuildOutput`に通した（[記録](../evidence/TEI_GO_SELF_BUILD_GO_COMMAND_PLAN_2026-11-10.md)）。

| 欄 | 値 |
| --- | --- |
| `recipe.kind` | `go-command` |
| `recipe.entrypoint` | `./cmd/tei` |
| `outputKind` | `runtime-artifact` |
| `sources[0]` | `{kind: "go-package", locator: "cmd/tei", digest: "sha256:…"}` |

**結果：`Validate()`は通り、fingerprintが出て、`LockBuildOutput`も通った。** 型を足す必要は無い。`LockBuildOutput`は`runtime-artifact`を要求するので、**commandのためのlockの種類も既にある**（`Lock`は`runtime-artifact`専用として検証される）。

#### builderが満たすべき五条件

| 条件 | builderがすること |
| --- | --- |
| 1. entryが計画を導く | 呼び手が`-build-catalog`を渡す。**builder側の仕事ではない** |
| 2. 宣言byteの一致 | `-intent`で宣言を読み、`intentBytesDigest`を申告する |
| 3. 計画fingerprintの一致 | `-plan`で計画を読み、`BuildPlanFingerprint`を計算して申告する |
| 4. plan→artifact | 成果物を書き、`artifactDigest`を申告し、**自分でlockを作って検証する** |
| 5. artifactの再現 | `-artifact-out`と`-state`を尊重し、`stateDir`を申告する |

#### 入力の照合をどこまでやるか

**計画の`Sources`は、builderが`DirSourceResolver`で同じrootから解釈し直して突き合わせる。** §4.36は単一ファイルだけだったが、`DirSourceResolver`はディレクトリも扱うので、**`go-package`のsource（`cmd/tei`のようなディレクトリ）でも同じ照合ができる。**

| 案 | 内容 | 何が違うか |
| --- | --- | --- |
| **A: その場で照合してbuild** | 作業ツリーのファイルを計画と突き合わせ、一致したら**そのツリーで**`go build`する | 実装が小さく、今日のtoolchainの挙動をそのまま使える。**照合とcompileの間にファイルが変わり得る**（TOCTOU） |
| B: 閉包を別の木へ写してbuild | 計画のsource closureだけを隔離した木へ写し、`go.mod`を合成してbuildする | 照合とcompileの間に何も変わらない。**module解決の合成が要る**（依存の取り込み方を含む） |

**最初の一歩はAとする。** 理由は三つ:

1. **Bの合成規則（module path・置換・依存の取り込み）は、それ自体が設計対象**であり、先に決めると計画が重くなる
2. Aでも「計画に載っていないsourceを読んでいないか」は**列挙で確かめられる**（`go list -deps`の結果を閉包と比べる）
3. **Aの残る危険（TOCTOU）は名前を付けて記録できる**。Bはその危険を消すが、代わりに別の危険（合成の誤り）を持ち込む

#### 実装が要るもの（順）

| 段 | 内容 | 何が満たされるか | 状態 |
| --- | --- | --- | --- |
| 1 | `tools/reference-go-build-from-plan`: `-plan`・`-intent`・`-repo`・`-artifact-out`・`-state`を受け、計画とsourceを照合して`go build`し、5項目を申告する | 条件2・3・5と、条件4の材料 | **実装済み**（[記録](../evidence/TEI_GO_SELF_BUILD_BUILDER_FROM_PLAN_2026-11-10.md)） |
| 2 | そのtoolをcatalogのproviderとして宣言し、`tei self-build -build-catalog`で鎖を通す | 条件1・4・**`l2b.claimed`** | **一度通した**（[記録](../evidence/TEI_GO_SELF_BUILD_L2B_FROM_REAL_GO_BUILD_2026-11-10.md)。`./cmd/tei`の実buildで`l2b.claimed=true`） |
| 3 | 回帰試験（`cmd/tei`のエンドツーエンド）に載せる | 毎回通ること | **実装済み**（[記録](../evidence/TEI_GO_SELF_BUILD_L2B_GO_COMMAND_GATE_2026-11-10.md)。66.8秒） |

**段1で確かめたこと**：計画が名指すsourceと実際のsourceが違えば**ビルドしない**。実buildが通り、`planFingerprint`・`artifactDigest`・`stateDir`・自分で作って検証したlockが出る。**`DirSourceResolver`をそのまま呼んでいる**ので、枠組みがentryと食い違わない。

#### 段2で分かったこと

**`./cmd/tei`の実buildで`l2b.claimed=true`が立った。** 五条件がすべて成立し、`unmet`は空である。ただし次の四点が残った。

| # | 残ったこと | 意味 |
| --- | --- | --- |
| 1 | 宣言の`definitionSource`は**`plugin-definition` kind**を要求する | **宣言モデルがplugin前提**である。commandだけの宣言は書けない（既存の`plugin.tei`を定義sourceにして通した） |
| 2 | `gobuilder`の固定設定が`GOPROXY=off` | 依存のある対象はビルドできない。`BuildPackageWithSupply`と`-supply`を足した |
| 3 | `definitionBinding`が`unperformed` | 新しいbuilderが`definitionBytesDigest`を申告しない。**§4.36のentry側照合が効いていない** |
| 4 | **回帰試験が無い** | 到達点は「一度通した」ままである |

**段3で回帰試験に載せた**（[記録](../evidence/TEI_GO_SELF_BUILD_L2B_GO_COMMAND_GATE_2026-11-10.md)）。あわせて`reference-go-build-from-plan`が`definitionBytesDigest`を申告するようにして、**§4.36のentry側照合がこのbuilderでも効く**ようにした（段2では`unperformed`だった）。

**試験の対象は小さなcommandである。** `./cmd/tei`を対象にすると101.6秒かかったため、機構が同じで軽い対象に変えて66.8秒にした。`./cmd/tei`そのものの観測は段2の記録に残っている。**`-short`では飛ぶ**（実buildを二回することが観測そのものだから）。

**§4.38の表の「満たさない」は、段1と2で埋まる。** 段3が無いと、次に検査を足したときに黙って止まる（§4.38の3番目）。

#### この節が言っていないこと

- **段2・3がまだ無い。** 段1だけでは`l2b.claimed`は立たない
- **AのTOCTOUを閉じていない。** 閉じるにはBが要る
- `go list -deps`の結果と閉包の突き合わせは**案**であり、実装していない
- 受理・Runtime Authority・供給のTrust。いずれも未主張のまま

### 4.40 定義sourceの種類は、recipeが決める（2026-11-09）

段2で`./cmd/tei`の実buildを鎖に通したとき、**宣言の`definitionSource`が`plugin-definition` kindを要求する**ために、commandのビルドにも既存の`plugin.tei`を定義sourceとして差し込む必要があった（[記録](../evidence/TEI_GO_SELF_BUILD_L2B_FROM_REAL_GO_BUILD_2026-11-10.md)）。**計画は、そのビルドが読まないファイルの指紋を運んでいた。** 本節は、その規則を**recipeに応じて変える**。

#### 規則

| recipe | 定義sourceに要求するkind | 計画の`pluginDefinitionFingerprint`の出どころ |
| --- | --- | --- |
| `go-command` | **`go-package`**（ディレクトリ可） | **sourceのidentityそのもの**（`FingerprintJSON(source)`） |
| それ以外（`go-package`など） | `plugin-definition`（単一ファイル） | 解析した宣言の`BuildDefinitionFingerprint` |

**「主体を定義する入力」は、何を作るかで変わる。** pluginは宣言が定義であり、commandはコンパイルされるpackageが定義である。**単一の規則を緩めるのではなく、recipeごとの規則にする。**

#### 変えたところ

| 場所 | 変更 |
| --- | --- |
| `internal/buildintent`の宣言検証 | kindの検査を`validateDefinitionSourceKind(recipe, source)`へ |
| 解決 | `go-command`では定義sourceを**pluginとして解析しない**。単一ファイル要求も無い。指紋はsourceのidentityから |
| 計画 | `pluginDefinitionFingerprint`は上の表のとおり。**欄の意味は「主体を定義する入力の同一性」で、recipeが導出法を決める** |

#### 実測

| 試験 | 結果 |
| --- | --- |
| `TestValidateAcceptsTheDefinitionKindEachRecipeNeeds`（plugin宣言＋`plugin-definition`／command＋`go-package`） | **PASS** |
| `TestValidateRefusesTheDefinitionKindTheRecipeDoesNotUse`（**両方向**で拒否し、必要なkindを名指す） | **PASS** |
| `TestValidateStillRequiresADeclaredDefinitionSource` | **PASS** |
| `TestResolveDerivesTheDefinitionIdentityPerRecipe`（commandの計画に定義identityが入り、**同じ入力で安定**し、**bytesが変われば変わる**） | **PASS** |

環境：WSL2 Ubuntu 22.04、`go1.27.3`。

#### この節が言っていないこと

- **既存の宣言の移行**。`plugin-definition`を使う宣言は今までどおりで、変更は要らない（試験で確認）
- `go-command`以外の非plugin recipe（将来増える場合）の規則。**今は`go-command`だけを分けた**
- `pluginDefinitionFingerprint`という**欄の名前**。commandではpluginを指さないが、名前を変えると計画の形が変わるため、意味の説明で補った
- 受理・Runtime Authority・供給のTrust。いずれも未主張のまま

### 4.41 計画が、ビルドが実際にコンパイルするpackageを覆っているか（2026-11-09）

§4.39の案Aは「計画と作業ツリーを照合してから、そのツリーで`go build`する」形にした。しかし**照合するのは計画が名指したsourceだけ**であり、**ビルドが計画に載っていないmodule内packageを引き込んでいないか**は見ていなかった。

本節は、その検査を足したことを記録する（TOCTOUを閉じるものではない。§4.39の案Bは別）。

#### 何をするか

| 段 | 内容 |
| --- | --- |
| 1 | `go list -deps -f '{{if not .Standard}}{{.ImportPath}}\t{{.Dir}}{{end}}'`を、ビルドと同じ隔離環境・同じ供給で走らせる |
| 2 | module内（root配下）の各packageのdirが、計画の`go-package` sourceの**いずれかの配下**にあるかを調べる |
| 3 | **覆われていないpackageがあればビルドしない**。理由にpackageとdirを並べる |

**標準ライブラリは対象外**（`-f`の`{{if not .Standard}}`）。**外部moduleは`external`として別に記録する**——供給から来るものであり、計画のsourceが覆う対象ではない。

#### 実測

| 試験 | 結果 |
| --- | --- |
| `TestCheckDependencyCoverageRefusesAnUncoveredModulePackage`（`./internal/buildintent`を、無関係なsourceだけを名指す計画でビルドしようとする → **覆われていないと判定**） | **PASS** |
| `TestCheckDependencyCoverageAcceptsASelfContainedPackage`（module内依存の無いpackageは自分のsourceで覆われる） | **PASS** |
| `TestRunBuildsFromThePlanAndReportsWhatItDid`（依存の無いprobeをビルド） | **PASS** |
| `TestSelfBuildEstablishesL2BFromARealGoBuild`（回帰試験） | **PASS**（16.1秒） |

#### この検査が意味すること

**`./cmd/tei`のビルドは、`cmd/tei`だけを名指す計画では通らない。** 実際に試すと、`internal/`配下の30弱のpackageが「覆われていない」と出る。**これは正しい。** 計画が`cmd/tei`だけを名指している以上、その計画は「`cmd/tei`をコンパイルする」ことを記述していない。

**したがって、宣言の`sources`はmodule内の閉包を含まなければならない。** これは宣言を書く側の負担増だが、**「計画が記述していないビルド」を通さない**という意味を持つ。

#### この節が言っていないこと

- **TOCTOUは閉じていない。** 照合と`go list`と`go build`の間にファイルは変わり得る
- 外部moduleの完全性。**供給のTrustの範囲**であり、ここでは`external`として並べるだけ
- 覆われていないpackageを自動で計画へ足すこと。**計画は呼び手が書くもの**である
- `vendor/`の扱い。**今は対象から外している**

### 4.42 照合した木をビルドする（案Bの第一歩、2026-11-09）

§4.39の案Aには**TOCTOU**が残る——計画と作業ツリーを照合してから`go build`するまでの間に、ファイルは変わり得る。本節は、**案Bの第一歩**を実装したことを記録する。

#### 何をするか

`-isolate`を付けると、**計画が名指すsourceだけを新しい木へ写し、その木をビルドする。**

| 段 | 内容 |
| --- | --- |
| 1 | 各sourceを`DirSourceResolver`で読み、**locatorの形を保って**新しい木へ書く |
| 2 | 写した木を**同じresolverで読み直し**、計画のdigestと一致することを確かめる。**違えばビルドしない** |
| 3 | `go.mod`・`go.sum`を作業ツリーから取り、**そのdigestを記録**する |
| 4 | その木で`go list -deps`と`go build`を行う（§4.41の被覆検査もこの木に対して行う） |

**「確かめたもの」と「ビルドしたもの」が同じ木になる。** 作業ツリーの後からの変更は、ビルドに届かない。

#### 素直でない点：module contextは計画のsourceではない

`go build`には`go.mod`が要る。**これはsourceではない**ので計画は名指さない。そこで**作業ツリーから取り、`moduleFiles`としてdigestを記録**する。**境界に明記**した——「module contextは主張に含まれない」。

**これは穴を塞ぎ切っていない。** `go.mod`を後から変えれば、依存の解決は変わり得る。塞ぐにはmodule contextも計画の一部にする（`Dependencies`の拡張）必要があり、それは未着手である。

#### 実測

| 試験 | 結果 |
| --- | --- |
| `TestStageSourcesCopiesExactlyThePlannedBytes`（写した木が計画と同じdigestで、module contextが記録される） | **PASS** |
| `TestStageSourcesRefusesACopyThatDoesNotMatchThePlan`（写した木が計画と違えば**拒否**） | **PASS** |
| `TestRunWithIsolateBuildsFromTheStagedTree`（`-isolate`で実際にビルドし、木のdigestと境界が記録され、被覆検査も通る） | **PASS** |
| `go test ./tools/reference-go-build-from-plan -count=1` | **PASS**（20.6秒） |

#### 既定にしない理由

**`-isolate`は既定にしていない。** 理由は二つ:

1. **module contextが計画の外にある**ので、隔離しても主張は完全には閉じない。既定にすると「閉じた」と読まれやすい
2. 依存のある対象では、**module解決の合成規則**（§4.39の案Bの本体）がまだ無い

**既定にできるのは、module contextを計画の一部にした後である。**

#### この節が言っていないこと

- **TOCTOUを完全には閉じていない。** `go.mod`・`go.sum`は作業ツリーから取る
- 複数sourceが同じdirectoryを共有する場合の合成規則（今はlocatorごとに独立して写す）
- `vendor/`・置換（`replace`）の扱い
- 既定化。**上の二つの理由で見送っている**

### 4.43 module contextも計画のsourceにする（2026-11-09）

§4.42の`-isolate`は、**`go.mod`・`go.sum`を作業ツリーから取る**ので、隔離してもTOCTOUが残った。本節は、**module contextも計画のsourceにする**ことで、その例外を消す。

#### 規則

**`kind: "go-module"`のsourceとして`go.mod`・`go.sum`を宣言できる。** 宣言すれば、隔離ビルドはそれらを**他のsourceと同じように写し、同じようにdigestを照合する**。

| 計画がmodule contextを名指すか | 隔離ビルドの`moduleFiles` | 何が言えるか |
| --- | --- | --- |
| **名指す** | **空** | **木は計画のsourceだけでできている**（作業ツリーからの持ち込みが無い） |
| 名指さない | `go.mod:<digest>`など | 木は計画＋作業ツリーのmodule contextでできている（境界に明記） |

**「例外は無い」と言えるのは、名指した場合だけである。** 名指さない場合は従来どおりで、記録がそのことを示す。

#### 実測

| 試験 | 結果 |
| --- | --- |
| `TestIsolatedTreeTakesNoModuleFileFromTheWorkingTreeWhenThePlanNamesThem`（`go.mod`・`go.sum`を`go-module` sourceとして宣言 → `moduleFiles`が**空**、写したmembersは**3件**＝package＋2ファイル） | **PASS** |
| `go test ./tools/reference-go-build-from-plan -count=1`（全試験） | **PASS**（79.4秒） |

#### 途中で直した欠陥

**単一ファイルのsourceを写す処理が、そのパスをディレクトリとして作っていた。** ディレクトリのsourceしか写していなかったので表面化していなかったが、`go.mod`をsourceにした瞬間に`is a directory`で落ちた。**ファイルとディレクトリを分けて書く**ように直した。

#### この節が言っていないこと

- **`go-module` sourceの検証**（`go.mod`という名前に限るか、任意のmoduleファイルを許すか）。今は**kindの意味だけ**を決め、名前は呼び手に任せている
- `go.sum`が無いmodule（依存の無いmodule）での扱い
- `-isolate`の既定化。**module contextを名指す計画が一般的になってから**
- `replace`・`vendor/`の扱い

### 4.44 隔離ビルドでもL2-bが立つことを回帰試験で確かめる（2026-11-09）

§4.42・§4.43で隔離ビルドを作ったが、**その経路で五条件が立つかは確かめていなかった。** 本節は、回帰試験を隔離ビルドへ切り替え、**隔離した木からビルドしても`l2b.claimed=true`が立つ**ことを固定した。

#### 変えたこと

| 対象 | 変更 |
| --- | --- |
| 宣言 | `go-module`のsourceとして`go.mod`・`go.sum`を追加（§4.43の規則） |
| child args | `-isolate`を追加 |
| 試験 | 主張に加えて、**childが計画を読むbuilderであること**（`childSchema`）を確かめる |

#### 実測

| 項目 | 値 |
| --- | --- |
| `TestSelfBuildEstablishesL2BFromARealGoBuild` | **PASS**（14.5〜71.3秒。環境の負荷で変動） |
| `l2b.claimed` | **true** |
| `planCrossCheck` / `planArtifactBinding` / `artifactReproducibility` / `definitionBinding` | すべて成立 |

#### この試験が示していないこと

**隔離が実際に行われたことを、entryは見ていない。** entryはchildのstdoutをdigestとして記録するが、`isolatedTree`の中身は解析しない。**隔離そのものの観測は、ツール自身の試験（写した木が計画と一致する・`moduleFiles`が空・実buildが通る）が担う。**

**したがって本節が言えるのは「隔離ビルドの経路でも鎖が成立する」までで、「この試行が隔離された」ではない。** 両者を一つの试验で言うには、entryがchildの隔離記録を読む必要がある（未着手）。

### 4.45 entryが、builderの隔離記録を読む（2026-11-09）

§4.44は「隔離ビルドの経路でも鎖が成立する」までしか言えなかった。**entryがchildの`isolatedTree`を解析していなかった**からである。本節は、それを読んで記録するようにしたことを残す。

#### 何をするか

| 段 | 内容 |
| --- | --- |
| 1 | childの`isolatedTree`（`root`・`members`・`moduleFiles`・`boundary`）を解析する |
| 2 | **`moduleFiles`が空なら`isolated`**、あれば`isolated-with-external-context`（理由に何を持ち込んだかを並べる） |
| 3 | 報告の`childIsolation`として出す。**五条件には加えない** |

#### 五条件に加えない理由

**条件を増やすことは契約の変更**であり、§4.23の述語を変える。**記録を足すだけなら、述語はそのままで、読む人が「この試行は隔離されていた」を確認できる。**

**加えるべきかは、`-isolate`を既定にする判断と同じ場所で決める**（§4.43）。今は記録だけを置く。

#### 実測

| 試験 | 結果 |
| --- | --- |
| `TestReadChildIsolation`（未申告は記録なし／`moduleFiles`空で`isolated`／あれば`isolated-with-external-context`と理由） | **PASS** |
| `TestSelfBuildEstablishesL2BFromARealGoBuild`（`childIsolation.state=isolated`・`moduleFiles`空・`memberCount=4`を確認） | **PASS**（12.8秒） |

**`memberCount=4`の内訳**：package・計画が持つ宣言・`go.mod`・`go.sum`。**宣言は§4.40で定義sourceではなくなったが、計画のsourceとしては残っている**（ビルドは読まないが、写す対象ではある）。

#### この節が言っていないこと

- **entryが隔離木を見たこと**。見えるのはchildの申告だけである（§4.37の「信じている」に属する）
- 五条件への追加。**契約変更であり、別途決める**
- `-isolate`の既定化
### 4.47 宣言したbuilderと、起動したファイルを結ぶ（2026-11-10）

§4.33は「builderの解決はdigestで照合しないため、catalogが別のバイトを名指ししていても試行は通ってしまう」と書き、§4.37の実測は「`-child`がプロキシなら背後の差し替えは止まらない」ことを示した。本節は、この二つが**同じ一つの欠落**であると整理し、塞いだ範囲と残る範囲を分ける。

#### 何が欠けていたか

同一性には三つの対象がある。

| 対象 | 値の出どころ | 変更前の扱い |
| --- | --- | --- |
| **宣言したbuilder** | 計画の`builder.artifactDigest`（catalogから解決。entryが自分で導く） | **どこにも比べられていなかった** |
| **起動したファイル** | entryが`-child`のbyteを起動の直前に読んだdigest | 二回の起動どうしを比べるだけ（条件5） |
| **実際に動いた構築器** | OSが読み込んだプロセスイメージと、それが起動したもの | 観測していない |

条件5は二番目を**それ自身と**比べていた。一番目と二番目を結ぶ検査が無かったので、計画がbuilder Dを宣言し、entryがbyte Eのファイルを起動しても、**「この計画がこのartifactを生んだ」というClaimが立った**。プロキシはその一例であり、catalogが別のbyteを名指しする場合も同じ欠落である。

**これは実装の欠落である。** 両方の値をentryが既に持っており、childの申告を一つも要さずに比べられた。保証範囲の限定ではない。

#### 足したもの

`builderBinding`という記録を一つ足した。計画の`builder.artifactDigest`と、起動したファイルのdigestを比べる。

| 状態 | 意味 |
| --- | --- |
| `bound` | 起動したファイルのbyteが、計画の宣言するbuilderのbyteと同じ |
| `mismatched` | 違う。ラッパー、プロキシ、別ビルドのbuilderはいずれも「別のファイル」 |
| `unperformed` | 計画が無い、計画がdigestを持たない、起動したファイルのdigestが無い |

**`l2b.claimed`は`bound`を要求する**（§4.23の五条件に並ぶ六つ目の条件）。新しい型、新しい実行器、新しいEvidenceの形式は足していない。値は既存の`plugin.BuildPlan`と、既存の起動記録（`invoked.executableDigest`）である。

**`mismatched`は試行を失敗にしない。** childは動き、成果物は照合されている。落とすのはClaimだけである。理由は、catalogを渡さない試行（計画が無い）が今も`ok`であることと揃えるため、また宣言と違うファイルを起動する正当な構成（エミュレータのラッパー）を「buildの失敗」と呼ばないためである。**試行ごと止めるべきだと分かったら、`appendSelfBuildFailure`を一行足せば覆せる。**

#### 止まるもの、止まらないもの（実プロセスの試験で固定）

| 構成 | 結果 | 止める検査 |
| --- | --- | --- |
| catalogがbuilder Aを宣言し、`-child`もA | Claimが立つ | — |
| catalogが別のbyteを宣言（§4.33の穴） | **Claimが立たない**（試行は`ok`） | `builderBinding` |
| catalogがAを宣言し、`-child`は1回目にA・2回目にBを起動するプロキシ | **Claimが立たない**（再現は`reproduced`のまま） | `builderBinding` |
| `-child`のファイル自体が二回の間に書き換わる | **Claimが立たず、試行も失敗** | 条件5（1回目は`bound`） |
| **catalogがプロキシ自体を宣言**し、プロキシが背後でA→Bと起動 | **Claimが立つ** | 止まらない（下記） |

#### Claimが言うこと、言わないこと

**言う**：entryが起動を依頼したファイルは、起動の直前に読んだ時点で、計画が宣言するbuilderと同じbyteだった。二回とも同じだった。

**言わない**：

1. **OSが読み込んだイメージがそのbyteであること。** entryはファイルを読んでから、パスで起動する。この二つは不可分ではない（TOCTOU）。閉じるには保持したイメージから起動する仕組みが要り、それはRuntime側（`internal/plugin/runtime`の保持イメージ検証、#973）の仕事である
2. **起動したファイルが何を起動したか。** 宣言されたbuilderが別のプログラムへ処理を渡すなら、それは**宣言されたbuilderの振る舞い**である。Goのbuilderがコンパイラを起動するのと、プロキシが別のbuilderを起動するのを、entryは区別しない。区別するにはexecの監視が要る（§4.37「敵対的なchildを想定するなら」の2行目と同じ）
3. **catalogが正しいこと。** プロキシを宣言するcatalogは、プロキシをbuilderとして宣言している。それを受け入れるかは供給のTrustの判断である

したがって「同じ構築器で二回buildした」は、**「同じ宣言済みファイルを二回起動した」**までしか言えない。`l2b`と再現の境界文は、この表現に直した。

#### 既存の記録への影響

- arm64のL2-b連鎖（§4.25以降の記録）は、catalogが`builder`を名指しし、entryはラッパーを起動した。**現行では同じ構成でClaimは立たない**。Claimを立てるには、catalogがラッパーを宣言する必要がある（その場合、宣言されたbuilderはラッパーである）。過去の記録は当時の観測として残す
- §4.23・§4.37・§4.38・§4.39の「五条件」は当時の述語である。現行の述語は六条件である

#### この節が言っていないこと

- 上の1〜3を閉じた、とは言っていない
- `-child`に通常ファイル以上の制約（実行形式の判定）は課していない
- 受理・Runtime Authority・供給のTrust。いずれも未主張のまま

### 4.46 計画は、ビルドが読むsourceだけを名指す（2026-11-09）

§4.41は「ビルドが計画に載っていないpackageを読む」を拒否した。**逆向きの食い違い**——計画が、ビルドが読まないsourceを名指す——は残っていた。§4.40で`definitionSource`が`go-package`になった後も、宣言は`plugin-definition`を計画に残していた（写すだけで、ビルドは読まない）。

#### 規則

**`go-command`の計画は、`go-package`と`go-module`だけを名指せる。** それ以外のkindがあれば**ビルドしない**。

| なぜ | 内容 |
| --- | --- |
| 記述の一致 | 使わないsourceを載せると、計画は「ビルドより多い」ことを言う。読み手はどちらが本意か分からない |
| §4.41との対称 | あちらは「ビルドが計画より多い」、こちらは「計画がビルドより多い」。**どちらも計画とビルドが別物** |

**`go-command`以外のrecipeは従来どおり**（規則を決めた範囲だけに適用する）。

#### あわせて直したこと：定義sourceのkindを両側で揃える

`definitionBytesDigest`の照合（§4.36・§4.40）が**`plugin-definition`固定**だったため、宣言を外すと`definitionBinding`が`unperformed`になった。**builderとentryの両方で、recipeに応じてkindを決める**ようにした（`go-command`なら`go-package`）。

**片側だけ直すと、静かに`unperformed`へ落ちる。** 実際にその順序で踏んだ。

#### 実測

| 試験 | 結果 |
| --- | --- |
| `TestCheckSourcesAreReadAcceptsTheSourcesACommandBuildUses` | **PASS** |
| `TestCheckSourcesAreReadRefusesASourceTheCommandBuildNeverReads`（`plugin-definition`を名指すと拒否し、理由を述べる） | **PASS** |
| `TestCheckSourcesAreReadLeavesOtherRecipesAlone` | **PASS** |
| `TestRunRefusesAPlanWithAnUnusedSource`（試行ごと拒否） | **PASS** |
| `TestSelfBuildEstablishesL2BFromARealGoBuild`（宣言を外した計画で`l2b.claimed=true`・`definitionBinding=bound`・`memberCount=3`） | **PASS**（26.6秒） |

#### この節が言っていないこと

- **`go-command`以外のrecipeの「読むsource」の集合**。決めていない
- 計画に載る`go-package`が**複数**の場合の扱い（今は全部写す）
- 記録側（`sources`）の意味の変更。**計画のsourceは、ビルドが読むものだけになった**

### 4.47 計画がmodule contextを名指すなら、隔離ビルドを既定にする（2026-11-09）

§4.42は`-isolate`を**既定にしなかった**。理由は「module contextが計画の外にある」だった。§4.43でそれが計画の内側に入ったので、**条件が変わった。**

#### 規則

| 計画が`go-module`を名指すか | 既定 | 記録 |
| --- | --- | --- |
| **名指す** | **隔離ビルド**（`buildSource=plan-only`） | 木は計画のsourceだけ |
| 名指さない | 作業ツリーを照合してビルド（`buildSource=working-tree`） | 隔離しない |

**`-isolate=false`で明示的に断れる。** 明示された指定は既定に優先する。

#### なぜ「条件付きの既定」なのか

**無条件に既定にすると、module contextを持たない計画で「隔離した」と読まれる。** 実際には`go.mod`を作業ツリーから取ることになり、§4.42が避けたかった読み違えそのものになる。**計画が名指すときだけ既定にする**ことで、既定の意味が「計画だけでビルドした」に固定される。

#### 五条件は変えない

**隔離を6番目の条件にはしない。** 条件を増やすのは§4.23の述語の変更であり、**隔離できないbuilderを一律に排除する**（§4.38の表が示すとおり、childの能力に依存する）。**既定を強くする**ことで、通常の経路が隔離になり、記録（`childIsolation`・`buildSource`）がそれを示す。

#### 実測

| 試験 | 結果 |
| --- | --- |
| `TestIsolationIsTheDefaultWhenThePlanNamesTheModuleContext`（`buildSource=plan-only`・`moduleFiles`空） | **PASS**（34.2秒） |
| `TestIsolationStaysOffWhenThePlanNamesNoModuleContext`（`buildSource=working-tree`） | **PASS**（14.1秒） |
| `TestIsolationCanBeTurnedOffExplicitly`（`-isolate=false`が既定に勝つ・木を作らない） | **PASS**（9.1秒） |
| 既存の隔離試験・回帰試験（`./tools/reference-go-build-from-plan`・`cmd/tei`） | **PASS** |

#### この節が言っていないこと

- **依存のある対象**（外部moduleを引く）の隔離ビルド。合成規則が要る（§4.39の案B本体）
- `replace`・`vendor/`の扱い
- 隔離を6番目の条件にすること。**今はしない**（判断の材料は揃いつつある）

### 4.48 隔離木が再現できないmodule contextは、ビルドしない（2026-11-09）

§4.43でmodule contextが計画のsourceになった。しかし**計画が名指していても、その中身が隔離木では再現できない場合**がある。本節は、その二つを**拒否**することを決める。

#### 何を拒否するか

| module contextの中身 | なぜ再現できないか |
| --- | --- |
| **`replace`**（`replace x => ../x`） | 置換先は計画のsourceではない。隔離木にそのパスは無く、あっても「計画が名指したもの」ではない |
| **`vendor/`** | 依存がmoduleの隣に置かれている。隔離木は`vendor/`を写さないので、ビルドは供給の側から取ることになる |

**どちらも「ビルドは通るが、計画が記述したビルドではない」**種類の食い違いである。§4.41・§4.46と同じ族に属する。

#### 実測

| 試験 | 結果 |
| --- | --- |
| `TestCheckModuleContextRefusesAReplaceDirective` | **PASS** |
| `TestCheckModuleContextRefusesAReplaceBlock`（同じことを別の書き方でした場合） | **PASS** |
| `TestCheckModuleContextRefusesAVendorDirectory` | **PASS** |
| `TestCheckModuleContextAcceptsAPlainModule`（`require`だけのmodule contextは通る） | **PASS** |
| `TestCheckModuleContextIgnoresAMissingModuleFile`（`go.mod`が無いことは別の話。ビルドが報告する） | **PASS** |
| 既存の隔離試験・回帰試験 | **PASS**（29.8秒／27.5秒） |

#### TEIのmodule contextはこの規則を通る

**`go.mod`に`replace`は無く、`vendor/`も無い。** したがってこの規則は、TEI自身を対象にした隔離ビルドを妨げない。**依存のある対象でも、`require`だけで解決できるなら隔離ビルドできる道が残る。**

（依存のある対象を実際に隔離木でビルドすることは、まだ測っていない。§4.47の記録のとおり未対応である。）

#### この節が言っていないこと

- **依存のある対象の隔離ビルドの実測**。今回拒否したのは「再現できない形」であり、残る「再現できる形」を試したわけではない
- `replace`を**計画で名指せるようにする**案。置換先をsourceとして宣言できれば再現できるが、その設計はしていない
- `vendor/`を計画に含める案。同上

### 4.49 依存のある対象でも、計画だけからビルドできた（2026-11-09）

§4.47の記録は「依存のある対象の隔離ビルドは未対応」で終わっていた。本節は、**それを実測した**結果を残す。

#### 何が要だったか

§4.46により、計画は**ビルドが読むsourceだけ**を名指さなければならない。`./cmd/tei`はmodule内だけで**73のpackage**を読む。手で並べるのは現実的でない。

そこで**計画を書くツール**を足した（`tools/reference-build-plan-authoring`）。

| 段 | 内容 |
| --- | --- |
| 1 | `go list -deps` でmodule内のpackageを集める |
| 2 | 各sourceを`DirSourceResolver`で解決し、**digestは実ファイルから計算**する（宣言はできない） |
| 3 | `go-module`として`go.mod`・`go.sum`も名指す |
| 4 | `BuildPlan`として書き出す |

#### 実測

`./cmd/tei`を対象に、計画を書いてから隔離ビルドを実行した。

| 項目 | 値 |
| --- | --- |
| 計画のsource数 | **75**（`go-package` 73・`go-module` 2） |
| `buildSource` | **`plan-only`** |
| `buildSucceeded` | **true** |
| `isolatedTree.moduleFiles` | **なし**（作業ツリーからの持ち込みゼロ） |
| 被覆検査 | **covered=73・uncovered=0**、external=36（供給から来る外部module） |
| lock | **validated** |
| artifact | `sha256:3e9a8d07a714300d8103fc4381b98a525ea19ec333842524f21325e413f266ee` |

**依存のある対象でも、計画のsourceだけでビルドできた。** module contextも計画から来ており、作業ツリーからは何も取っていない。

#### 何が言えるようになったか

| 以前 | 今 |
| --- | --- |
| 「隔離ビルドは依存の無い対象だけ」 | **依存のある対象（`./cmd/tei`全体）でも隔離できる** |
| 「計画を書くには73件を手で並べる」 | **ツールで作れる**（digestは実ファイルから） |

**ただし、これはビルドの話である。** 五条件のClaim（`l2b.claimed`）はまだ小さなcommandでしか立てておらず、`./cmd/tei`を対象にしたClaimは再測していない（§4.39段2の観測は隔離前のもの）。

#### この節が言っていないこと

- **`./cmd/tei`を対象にした五条件のClaim**。隔離ビルドが通ることと、Claimが立つことは別である
- 外部moduleの完全性（供給のTrust）
- 計画の自動生成が「読むもの全部」であることの証明。**ビルド時の被覆検査が確かめる**（§4.41）
- `replace`・`vendor/`（§4.48で拒否）

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
- [x] **実build**での`-rebuild-check`のコスト測定（§4.28。`reference-go-build-with-supply`をchildにして`./internal/core`を実際に`go build`。中央値5852.7 ms→11272.4 ms、**+5419.7 ms・1.926倍**で、§4.27の推論が当たった。二回のbuildのdigestは一致し`artifactReproducibility=reproduced`。[記録](../evidence/TEI_GO_SELF_BUILD_GO_BUILD_REBUILD_COST_2026-11-09.md)）
- [x] `-rebuild-check`の実コストの測定（§4.27。[記録](../evidence/TEI_GO_SELF_BUILD_REBUILD_CHECK_COST_2026-11-09.md)。参照childでは中央値136.4 ms→246.5 ms（+110.1 ms・1.807倍）で、**増分の大半はchildの仕事ではなくentryの固定費**（child単体3.4 ms）。実buildは未測定）
- [x] `-rebuild-check`の既定を**catalogを名指しした試行に限って**強くする決定と実装（§4.30。`-build-catalog`を渡せば二回目のbuildが既定になり、`-rebuild-check=false`で断れる。child経路と何も渡さない試行は変わらない）
- [x] `-rebuild-check`を**すべての試行で既定**にするかどうかの判断（**§4.34で閉じた：既定にしない。** child経路では費用が倍になるだけでClaimが増えない。catalogを名指しした試行では§4.30が既定で行う） **§4.25がP-1〜P-4を示し、§4.27が参照child、§4.28が実buildのコストを実測した**（それぞれ1.807倍・1.93倍）。**§4.30がcatalog経路だけ既定にした**ので、残る問いはそれ以外の経路でも既定にするかである
- [x] catalogの出所の決定（§4.22。**K-3: 呼び手が名指しするcatalogデータ＋必須のdigest**。新しいAuthorityを作らず、既存の`-supply`・`-intent-resolver`と同じ形）
- [x] **entryの既定経路をin-processの`Resolve`へ切り替えるか**の判断（**§4.34で閉じた：R-1据え置き。** entryが名指しされていないcatalogを読むことはK-3の取り消しになる。既定はchild経路のまま） 前提だったcatalogの出所は§4.22で決めた（K-3）。切り替え自体は段2の完了後に別途決める
- [x] **段1**：`fixture`を外した名前で`BuildCatalog`・`ParseBuildCatalog`・`ResolveBuilder`・`ResolveDependency`・`CheckTarget`・`DirSourceResolver`を`internal/buildintent`へ移した（`tools/`側は型aliasと薄い委譲。参照実装の試験は**無変更で通る**。§4.22の移行表、[記録](../evidence/TEI_GO_SELF_BUILD_CATALOG_MOVE_2026-11-09.md)）
- [x] **段2**：`cmd/tei`に`-build-catalog`・`-build-catalog-digest`・`-build-source-root`を足し、in-processで計画を導く経路を作った（`-intent-resolver`とは排他。§4.22、[記録](../evidence/TEI_GO_SELF_BUILD_INPROCESS_RESOLUTION_2026-11-09.md)）。**entryが導いた計画のfingerprintは参照実装のchildが出した値と一致した。** `planRecompute`・`planCrossCheck`・`planArtifactBinding`はin-process経路でも成立する
- [x] L2-bのClaimを立てるかの判断と実装（§4.23。**五条件の同時成立**を述語にし、`l2b.claimed`と`l2b.unmet`として出す。[記録](../evidence/TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md)。単一観測の`l2bClaimed`欄は`false`のまま据え置き、述語はそれらを読まない）
- [x] L2-bのClaimを**回帰試験**にする（§4.24。参照catalogと参照builder childで五条件が揃うことを`cmd/tei`の試験で固定し、`go test ./...`経由で健康性確認に載る。[記録](../evidence/TEI_GO_SELF_BUILD_L2B_GATE_2026-11-09.md)）
- [x] 二つのホスト（`linux/amd64`・`windows/amd64`）で同じ計画とartifactが出ることの実測（§4.26。[記録](../evidence/TEI_GO_SELF_BUILD_CROSS_OS_DIGESTS_2026-11-09.md)。`intentFingerprint`・`sourceClosure`・`planFingerprint`・`artifactDigest`の四つが一致し、L2-bのエンドツーエンド試験もWindowsでPASSした。**Go版も違う**）
- [x] **`arm64`と別のGo版**での観測（§4.26の範囲外）。**`arm64`では連鎖全体が成立した。** `tei`・child・著者ツール・resolverをすべてarm64へビルドしqemu上で`tei self-build`を走らせると、**`l2b.claimed=true`・`unmet`空・`reproduced`** になり、amd64と同じ結果だった（[記録](../evidence/TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10.md)）。**宣言とソースに依存する値は一致し、builderに依存する値は相違する**——`catalogDigest`は走るbuilderのバイトを名指しするので、アーキが違えば別の値になる。**Claimはクロスアーキテクチャの成果物一致を要求しない。**版は`go1.26.10`と`go1.27.3`で2×2を観測済み（[記録](../evidence/TEI_GO_SELF_BUILD_GO_VERSION_AND_ARCH_MATRIX_2026-11-10.md)）。
- [ ] `darwin`での観測。**未観測のまま**（macOSの実行環境が無い。クロスコンパイルは通るが走らせられない）。**この一項目だけが残っている。**
- [x] `l2b.claimed`を**既定の試行**で立てられるようにするか（**§4.34で閉じた：立てない。** 既定はchild経路で五条件の一つ目が成立しない。Claimは呼び手が`-build-catalog`一式を渡す行為） **呼び手が要るのはin-process経路（`-build-catalog`一式）だけで、`-rebuild-check`は§4.30が既定にした**（実測：`-build-catalog`のみで`claimed=true`・`-rebuild-check=false`で`false`。[記録](../evidence/TEI_GO_SELF_BUILD_L2B_DEFAULT_PREMISE_2026-11-09.md)）。残る問いは「何も渡さない試行」で立てるかであり、§4.29のcatalog決定が前提（§4.31）
- [x] 宣言とplanの**build座標**（`outputKind`・`target`・`profile`）の照合（[#3652](https://github.com/hat47x/tei-platform/pull/3652)。childが三座標を一つずつ比べ、違えばartifactを書いた後にreportを出さずexit 1。[記録](../evidence/TEI_GO_SELF_BUILD_BUILD_COORDINATES_2026-11-09.md)）
- [x] 宣言の他の欄のうち、planが同じ形で持つもの（`recipe`・`sources`・`dependencies`）の照合（[#3661](https://github.com/hat47x/tei-platform/pull/3661)。§4.17）
- [x] `parameters`の照合（[#3664](https://github.com/hat47x/tei-platform/pull/3664)。測るとどちらの側も名前順ソートのみで値を変えないと分かったので、名前で突き合わせる。§4.18）
- [x] `builder`（要求と解決結果の対応規則）の照合（[#3670](https://github.com/hat47x/tei-platform/pull/3670)。対応規則はresolverに既にあり、`BuilderContractVersion`として一箇所に定義した。§4.19、[記録](../evidence/TEI_GO_SELF_BUILD_BUILDER_REQUIREMENT_2026-11-09.md)）
- [x] `secretRequirements`の扱いの決定（§4.20。**S-1: childで照合しない**。byte一致が成り立てば欄ごとの照合は必ず通るため、保証が増えない。childの入力契約も`plugin.BuildPlan`も広げない）
- [x] **欄ごとの照合をここで打ち切る**決定（§4.20。原因究明の道具として既存は残すが、これ以上足さない。残る段差は欄照合では埋まらない）
- [x] 依存の更なる削減：**entryが計画側も独立に持つ**経路（[#3729](https://github.com/hat47x/tei-platform/pull/3729)）。lockの計画側はbuilderの申告を優先し、申告が無ければentryの再計算値を使う。どちらを使ったかを`planSideSource`で記録するので、弱い根拠を強い根拠と読ませない（[記録](../evidence/TEI_GO_SELF_BUILD_ENTRY_PLAN_SIDE_2026-11-09.md)）
- [x] catalogの欄がどこで宣言されているかの実測（§4.29。7欄のうち**5欄にcatalog以外の宣言面が無く**、既存の宣言からは導出できないと分かった。[記録](../evidence/TEI_GO_SELF_BUILD_CATALOG_FIELD_PROVENANCE_2026-11-09.md)）
- [x] catalogを**書く場所**の決定（§4.32。**製品の外**に置く。Workspaceもplugin構文も変えない。既存readerは未知のentryと未知の`formatVersion`を拒否するため、K-1・K-2は古いbinaryとの互換を壊す。K-3の読み方はそのままで、製品の契約を一つも変えない）
- [x] 鎖の**Windows**での観測と、catalogのホスト間一致（§4.33。`TestSelfBuildEstablishesL2BEndToEnd`がWindowsのGoでPASSし、同じ宣言とbuilderから作ったcatalogの`catalogDigest`が`linux/amd64`と`windows/amd64`で一致した。観測は二ホストで、`darwin`・`arm64`は未観測。[記録](../evidence/TEI_GO_SELF_BUILD_CHAIN_WINDOWS_2026-11-10.md)）
- [x] 宣言からClaimまでの鎖の**回帰試験**（§4.33。`cmd/tei`のエンドツーエンド試験がツールでcatalogを作り、**`artifactDigest`が実際に走るbuilderのバイトと一致すること**を確かめてから`l2b.claimed`を要求する。builderの解決はdigestで照合しないので、下流では気づけない）
- [x] catalogを作る**ツール**の能力宣言の形式と実装（§4.33。`tools/reference-build-catalog-authoring`が能力宣言からcatalogを作り、**artifactのdigestは実ファイルから計算**する。宣言にdigestを書くことと`-root`の外を拒否する。宣言から`l2b.claimed=true`まで一本で通った。[記録](../evidence/TEI_GO_SELF_BUILD_CATALOG_AUTHORING_2026-11-09.md)）
- [x] 能力宣言の**版付け**（**§4.35で閉じた：版を付けない。** 配るのはcatalogで、catalogは既に`formatVersion`を持ち未知の版を拒否する。宣言はツールと同じ場所で変わるので版を名指す相手がいない。**宣言を配ることになったら、配る前に`formatVersion`を足す**） **実測すると「必要になった時点」に決められない**——`formatVersion`を足した宣言は`json: unknown field`で拒否され、版を名乗る手段が導入時点で使えない。宣言を配るなら今足すのが最も安い（D-2）。**判断には「宣言を誰に配るか」が要り、これは§4.32が範囲外とした問い**（[記録](../evidence/TEI_GO_SELF_BUILD_DECLARATION_VERSIONING_2026-11-10.md)）。**§4.34が未決のまま閉じ、覆す条件を「第二のconsumerが現れた時点」とした**ので、本項目はその条件を待つ
- [x] D-1（catalogの`scope`拡張）を採るかどうかの契約判断（**§4.34で閉じた：採らない。** 外在consumerの前提を測れないまま変えない。発見性は`-h`とREADMEで確保した） （採る場合のみ、catalogの広告と実応答の一致を照合へ載せる）

**A-1（`gobuilder`移動）は採らないため、影響範囲の洗い出しは終了条件から外した。** 29ファイルの内訳は[記録](../evidence/TEI_GO_INTERNAL_RULE_AND_BUILDER_MOVE_2026-11-07.md)として残す。I-1は「採らない」から「I-1aとして着手」へ改めた（§4.13）。`ParseFixtureCatalog`（fixture catalog）は`tools/`に残す。

**P-2（`reference-go-build-with-supply`への欄追加）は採らない。** 計画を解釈しないchildに欄を足すと`unperformed`が`matched`に変わるが、主張の強さは変わらない（§4.10.2）。

本書は計画の正本を増やすものではない。SB-C0の**入口の外形と拒否条件**だけを固定し、#2831のP0順序（SB-0a → SB-0b → SB-A → SB-B → SB-C）のうちSB-C0を詰めた設計候補である。最終判断は元計画・ADR・actual Evidenceへ返す。
