# TEIテスト基本設計：テスト専用製品の適用評価と実証計画（2026-11-07）

- 状態：**評価待ちの設計追補／採用前**。テスト専用製品をインストール・実行してTEI適合を確認した記録ではない。
- 主担当：[独立検証基準 #2514](https://github.com/hat47x/tei-platform/issues/2514)、[Driver Scenario SSOT #1180](https://github.com/hat47x/tei-platform/issues/1180)
- 関連：[単一改訂Evidence #1163](https://github.com/hat47x/tei-platform/issues/1163)、[Data Context取得範囲 #2506](https://github.com/hat47x/tei-platform/issues/2506)、[実Provider縦切り #2586](https://github.com/hat47x/tei-platform/issues/2586)、[外部UI制御 #2391](https://github.com/hat47x/tei-platform/issues/2391)、[状態・Snapshot #1131](https://github.com/hat47x/tei-platform/issues/1131)、[Harness境界 #2515](https://github.com/hat47x/tei-platform/issues/2515)、[開発工程 #1445](https://github.com/hat47x/tei-platform/issues/1445)
- 既存の設計：[経路間の業務機能完全性とScenario由来E2E](CROSS_ROUTE_FUNCTIONAL_PARITY_AND_TEST_DERIVATION.md)。こちらの文書は置換しない。
- 限界：本書の製品機能は2026-11-07確認時点の公式文書に基づく比較。性能、費用、TEIとの相互運用性、製品別ライセンス適合は未実測。OwnerのMeaning受理・実Host Grant・Production資格は含まれない。

<!-- tei-test-20261109-1120z-evidence-status -->
> **追加検収（2026-11-09 11:31Z。各PRの固定HEAD資格）**：main [#3749](https://github.com/hat47x/tei-platform/pull/3749)でCLI必須引数未指定の`INVALID_ARGUMENT`分類を統一し、main [#3763](https://github.com/hat47x/tei-platform/pull/3763)でMCP README説明文の旧Failure主張をGo試験で固定。main [#3731](https://github.com/hat47x/tei-platform/pull/3731)で同Plan二回Build結果を区別、main [#3760](https://github.com/hat47x/tei-platform/pull/3760)で明示Catalogからのin-process計画導出を限定実証。さらにmain [#3757](https://github.com/hat47x/tei-platform/pull/3757)で6 Environment宣言の解決と実Invocationの断絶をread-only調査、main [#3761](https://github.com/hat47x/tei-platform/pull/3761)で内部Hook未知結末名を拒否、main [#3750](https://github.com/hat47x/tei-platform/pull/3750)で`states`→`closes`の許容順序の解析不具合を修正。[#3772](https://github.com/hat47x/tei-platform/pull/3772)の一値flag重複拒否は**未統合・照会時CI進行中**。**採用先は§35.10〜35.11、次の実証はT-Discover/R-Read**。Source完全閉包/G4、外部MCP完全配送、未知TEST-T1B、実`recent_orders` DB Read、S2-product/Owner意味受理は未実証。今回の新Go/CLI/PG/Agent実行はNOT RUN。


<!-- tei-test-status-nov09-cli-knowledge-and-mcp -->
> **現行の検収上限（2026-11-09。以下の古い「queued／未統合」は当時の記録）**：main [#3685](https://github.com/hat47x/tei-platform/pull/3685)の実CLI knowledge-* 連鎖、main [#3633](https://github.com/hat47x/tei-platform/pull/3633)のtrace継続指紋、修正済み[#3693](https://github.com/hat47x/tei-platform/issues/3693)のfull-mode空配列、main [#3540](https://github.com/hat47x/tei-platform/pull/3540)のInspector **S2-reference** を局所実績として採用。一方で **knowledge入力SchemaのCLI自己記述、Markdown本文の意味取得、実Function Provider P0-R、Blind TEST-T1B、外部H3配送、S2-product/Current Grant、C1・Owner業務意味受理**は未成立。最新の受入順は§35.7〜35.9と各Owner Issueを優先し、新Reader/Runner/Registryを設けない。



<!-- tei-test-nov08-1136z-qualification-delta -->
> **検収更新（2026-11-08 11:36Z、旧queued記録を上書き）**：
> - **TT-154が局所実証へ進んだ**。[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365) exact HEAD `e8745c6cbb7596da48da62becdb48417f3ce8dd6` の[専用dogfood #42465435223](https://github.com/hat47x/tei-platform/actions/runs/42465435223)、[Health #42465435270](https://github.com/hat47x/tei-platform/actions/runs/42465435270)、[依存 #42465435235](https://github.com/hat47x/tei-platform/actions/runs/42465435235)は**すべてcompleted/success**。同一専用Runで13実CLI Reply／42 bounded Controls／6実Pinned Step／9 test-only受領目録対照。独立固定`STEP_OPERATIONS`を用いるOracleは、`expected_operations`とReceiptの`operation`を**同じ誤値へ変異**させた場合も拒否した。これは§35で「次の一件」とした**TT-154の局所試験合格**であり、なおDraft/未main統合。外部End/ACK、Product全Coverage、実Provider、T1-Bは未資格。
> - **Provider Source/Manifest負例はmainへ統合**。[merged #3479](https://github.com/hat47x/tei-platform/pull/3479) は2026-11-08 11:08Zに統合。最終PR HEAD `6f0a9735cae71cb577296ffc4dc947870331824d` の[Health #42470247004](https://github.com/hat47x/tei-platform/actions/runs/42470247004)と[再検証 #42474407926](https://github.com/hat47x/tei-platform/actions/runs/42474407926)は**completed/success**。ConsumerのWorld/IndexだけでImport ProviderのSource/Manifest raw revision不変を推定できない局所Go反例を採用。ただしmain pushの**別exact-head Runは本カードで未検収**。同一Atomic Nested Load、外部Consumer受領、全Input Closureは証明していない。
> - **次のゲートを更新**：①#3365のOwner検収・main統合後の別Runを確認、②#3273の**未知Gap Blind T1-B**を#2514独立Expectedと隠したMutantで一件実施、③#2586の実Provider Read(R)→#2509/#2515の外部受領(H)を同一Attemptで限定接続する。§35の「TT-154は未実施」は**当時の計画履歴**とし、最新解釈は本更新を優先する。新たな本番Go/CLI/Blind/H3実行は**この文書更新ではNOT RUN**。

<!-- tei-test-20261108-current-acceptance-card -->
> **現行受入カード（2026-11-08 08:45Z時点。下記の過去時点の記録より優先。CIはexact HEADごとに再検収）**
>
> | 対象 | 現在確認した資格 | 未証明のまま保持する範囲 |
> | --- | --- | --- |
> | Design Context / 一Function | **main #3482** の13実CLI Reply／42限定Controls、main push専用CI #42462861657 success。Required Functionの宣言・Source存在を局所検証 | Function実Invoke・Provider Read・全Required閉包・Blind T1-B |
> | 共通入口 | **main #3403/#3439/#3458** で参照stdio MCPとADR-0048、`validate`局所Parity | 外部H3完全受領、全Tool/Studio/Webの同一Operation/Grant/Effect |
> | Scenario Coverage | [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365) は実Pinned 6 Stepとtest-only受領目録。**最新HEAD `f7de811d0ac6c92c696e7f36069ae47350cca5f8`の専用CI #42463616468は確認時点queued**。旧HEADのCI成功を転用しない | Product Coverage、独立Step→Operation実Receiptの全閉包、外部End/ACK、同一AttemptのL/R/H |
> | 外部Source鮮度 | **main #3477** のWorkspace Source-only drift実証。[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479) はProvider Source/Manifestを分離。**最新HEAD `fff65f0ab6ae7bd9ad88d18df7a756e8fbb4ae31`の全体Health #42464022323は確認時点queued** | Consumer二FPだけによるImport Providerの鮮度断定、A→B→A、同一Snapshot/現在Grant |
> | 外部Harness | [Draft #3494](https://github.com/hat47x/tei-platform/pull/3494) はSource/受領/独立検証の差分を文書化。Skill限定CI #42463453193 success、全体Health #42463453204は確認時点in_progress | H2/H3実完全配送、Tool実Recipient ACK、外部Sourceの全Input Closure |
>
> **本書の正本運用**：テスト専用製品の比較は参考資料であり、実テスト義務は#2514、Scenarioは#1180、Design Contextは#3262、実Providerは#2586、Deliveryは#2509/#2515、構成判断と実行順は#1119および既存MASTER_PLANを正とする。§30〜34のTT番号は**横断参照用**であり、独立Oracle/Runner/Registryを増やさない。§34のL/R/Hは選定L・実取得R・受領Hの**集合と時点を分けた比較**であり、同一件数や`verified:true`のコピーから実到達を推定しない。
>
> **次の実装一件**：#3365の最新exact-head CI成功を確認したうえで、#1180が保持するStep→Operationを外部固定し、実CLI Resultとの**同一Attempt照合**を一件実行する（§33 TT-154に相当）。その後に#3262の限定Contextから#3273の未知Gap Blind T1-Bへ進む。#3479のImport Source/Manifest失効と#2586の実Provider Readは並行検証。**本カードは設計更新であり、新CLI/Go/H3/Blind T1-B実行ではない。**



> **最新の実証状態（2026-11-07末時点／下記各節の古い「未実行」を上書きする進捗注記）**：main統合済み #3284/#3296 は実CLI 9応答・Design Contextの部分資格と16種類の限定負例、#3299 はSource/Lockの実改変と旧Pin拒否、[#3304](https://github.com/hat47x/tei-platform/pull/3304) は既知の新Rule境界を**一時Candidate第7 Stepとして実TEI CLIで実行**し、独立Propertyと3種類の負例で判別した。#3304のPR HEAD `02d4c624...` について[専用CI #42378033965](https://github.com/hat47x/tei-platform/actions/runs/42378033965)はsuccess。ただし後続docs更新を含む最終HEADおよび統合後mainの別Runの成功は、本書では再検収していない。これは**T1-A（既知提案を実走）**に限定され、**T1-B（未提示の欠落をAIが独力で発見）／全Required閉包／Owner受理／実Provider・H3 Delivery／C1**ではない。§17〜19の旧T1未実施記述は各観測当時の記録として残し、最新解釈は§20を優先する。

> **最新の横断受入判断（2026-11-08）**：§22〜25に、TT-56の統合候補#3354、業務拒否と技術失敗を分ける#3351、CLI実Writeの#3333、Plugin能力の宣言/選択の#3337/#3350、Subprocess Buildの#3336、DSH Sinkの#3309を、**実証済み・Draft未検収・必要な実配線**に分けて整理した。旧§17〜21の状態説明は各記録時点のものとし、現在の受入の上限は最新節を優先する。

<!-- tei-test-compact-execution-baseline-20261108 -->
> **現在の実装基線と次の1件（2026-11-08／履歴章の状態表示より優先）**：
> ① **成立した局所証拠**：main統合 #3304 の既知境界T1-A（7番目の一時Step／独立負例3）、main統合 #3354 の11実CLI Reply＋32合成Consumer対照、main統合 #3353 の空集合でも未確認Handoffを`holds`にしない修正、main統合 #3360/#3362 の参照Stage Driver限定の再入／process-image対照、main統合 #3363 の24 Bindingに対する22 selected／2 missingという**参照算定**。これらは同一Product Attemptの合格を示さない。
> ② **新しく検収したDraft**：[TT-105/106実装 #3365](https://github.com/hat47x/tei-platform/pull/3365) HEAD `1e2531fede563ed4b1d32cca0e78ccaf69fe364c` の専用CI [#42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357)、全体Health [#42394673331](https://github.com/hat47x/tei-platform/actions/runs/42394673331)、既知脆弱性検査 [#42394673273](https://github.com/hat47x/tei-platform/actions/runs/42394673273) はいずれも **completed/success**。本PRは**未統合**。実6 Pinned Stepに対する試験用受信目録・7コピー負例の合格であり、`productCoverage=unknown`と`untestedConfirmed=[]`を維持する。External Tool/全Product/C1/真のT1-Bの証拠ではない。
> ③ **CLI契約の現行圧力**：[第2ラウンド #3356](https://github.com/hat47x/tei-platform/pull/3356)の実CLI調査ではD-9（管理MarkdownはLoaderの意味入力でない）とD-10（CLI JSON形状不統一）が残り、MCP Server入口は**未確認・MCP NOT RUN**。一方[#3366](https://github.com/hat47x/tei-platform/pull/3366)は`view -format json`にWorld/SourceIndex Fingerprintを加える**未統合PR**で、同HEADの限定/全体CI成功は現時点の検証証拠。見える欄が増えることとSource全件のReader資格は別。
> ④ **次の実行順（既存Ownerへ委譲）**：P0は #3365 のmain統合後Run資格確認→#3262の3 Required Rule/Source/実選択Modeと受領完了の局所判定→#3273の**未知Obligationを生成者から隠したT1-B**、同時並行で #2586 の実Product Function→SQL/Provider Read。P1は`End marker`・`verified`を呼出元booleanと実Receiver/Runの独立Witnessへ分ける否定対照、#3356のD-9/D-10と#3366の各Version/Scope相違をCLI/Skillに透過しない比較、#3279 Plugin実選択と実Invoke、#2831の`cmd/tei`再入を追加。**T1-B盲検／実Product Provider／外部H3完全受領／Go C1／Owner Meaningは依然未資格**。


<!-- tei-test-currentmain-api-13reply-42controls-20261108 -->
> **現行mainに基づく補正（2026-11-08、§30を優先）**：ADR-0048はCLI／MCP／HTTP／Desktopで共有する型付きアプリケーションAPI方針として**採用済み**（#3439）。実TEI MCP入口がないという旧調査は履歴であり、main #3403には4 read-only Toolの**参照MCP Adapter**が存在する。#3458は実CLI／参照MCPのvalidate正常・意味失敗・Source Load失敗の投影を限定的に揃えた。main #3482の[push CI #42462861657](https://github.com/hat47x/tei-platform/actions/runs/42462861657)は**13実CLI Reply、42限定Consumer Controls、Design Context Packet/9、既存Scenario6＋T1-A／Source-only失効**で成功。全Domainは引き続きpartial／qualifiedTestGeneration=false、実Function Invocation・Provider Read・外部H3 Delivery・T1-B・C1・Owner意味受理は未達。#3477の同Workspace Source-only改版と#3479のimport元Provider Raw改版は別の失効問題。Workbench旧Draft #3372はclosed/unmerged、後継は[#3486](https://github.com/hat47x/tei-platform/pull/3486)（Draft）である。過去節の「MCPなし」「11 Reply」「#3370未統合」「#3372現行」はそれぞれ記載当時の状態とみなす。

## 1. 守るべき不変条件

TEIのテストは、単なる「実行した／PASSした」の収集ではなく、**テストの判別力・実入力の同一性・対象範囲・証拠の現在資格**を分離して扱う。専門製品の導入で新しいSemantic正本、第二Driver、第二Test Scenario、第二Host、第二Authority、共通Runtimeを増やさない。

- 業務意味の正本はAccepted Canonical。テストの記述正本は保存・版管理したDriver Scenario（#1180）。
- Test CaseはScenarioの具体化、Verification Obligationは検証要求、Evidenceは観測結果、Qualificationは**特定のClaimと対象**に対する証拠の適用判断。これらは同一物ではない。
- Expectedは候補コード、実Evaluator、実行Actual、AI同士の一致から自動的に正当化しない。Source／Owner／定義域／時間・粒度・単位・NULL・取得集合／導出手順を保持する。
- CLI/Web/Desktop/Java/.NET/DB等のTarget固有Runnerは派生物。意味上のExpectedをTarget毎のテスト資産で二重管理しない。
- 旧PASSや旧Receiptを現在資格へ昇格しない。業務結果、Execution Error、Host Grant、Provider Effect、Receipt、独立Resource再読取を混同しない。
- 合成fixture・mock・service virtualizationは再現可能性の証拠だが、実Provider／現在の認可／本番環境の適合証明ではない。テスト製品の「self-healing」は要素探索などの候補更新であって、独立Expected／検証要求の無承認変更を許さない。
- 意図的に壊したMutant branchは**Draftのままmainへ絶対に統合しない**。

## 2. 既存の実証と未成立範囲（調査時点）

- [#2901](https://github.com/hat47x/tei-platform/pull/2901)のexact-headでFixture G0／AI追加Scenario G1／有限独立Expected G2を含むRepository Health成功。ただし一時統合Draft、正式main/全Providerには未転用。
- [#3017](https://github.com/hat47x/tei-platform/pull/3017)のrelation-unique判定結果反転、[#3128](https://github.com/hat47x/tei-platform/pull/3128)の末尾key切落しは、AI追加Scenarioで対象CaseのRedを実CIログで確認。各Mutantの局所的な検出実績であり全バグ率ではない。
- [#3175](https://github.com/hat47x/tei-platform/pull/3175)はRead Slice/Completeの境界をQ05〜07でcharacterizeしたDraft。合成返却集合であり、実Provider Read→取得閉包→Claim資格の成功ではない。
- 既存CLIのScenario `verify/generate/run/check`を再利用。GUI/CLI共通Driverが全Targetで動いた、全工程のEffect確認ができた、独立ExpectedがOwner受理済み、という主張は未成立。

## 3. テスト専用製品・フレームワークの比較

以下の「採用」は**設計原理または外部実行Adapterの候補**。正式な標準依存・購入・インストールの決定ではない。

| 製品・技法 | 公式に確認した特徴 | TEIへ抽出する要素 | 判断と境界 |
| --- | --- | --- | --- |
| Tricentis Tosca | Requirementsの重み付きリスクCoverage、TestCase-Designの属性・値の組合せと生成 | Requirement→Obligation→Case→Runのトレーサビリティ、リスク優先度、組合せ行列 | **設計採用候補/P1**。重み付きCoverageはScope閉包不明や未実行を隠す単一%にしない |
| GraphWalker | 状態/遷移グラフのpath生成、coverage停止条件、seed再生 | Business Procedure/Workflow/Grant失効を状態機械とし禁止経路・復旧・分岐を生成 | **外部比較PoC/P1**。グラフ経路完走≠実Effect確認 |
| Microsoft PICT | パラメータと制約からpairwise等の組合せ生成 | Null/未取得/非開示×Tenant×時点×権限×Targetのcovering array | **軽量PoC/P1**。複合事故・時間順序はpairwiseだけでは不十分 |
| Go rapid / Go fuzz | 性質ベースの生成・失敗最小化、Go標準のfuzz corpus | 構文raw/typed、集合、グラフ、数値、不変条件、shrunk counterexampleを固定回帰へ | **実装候補/P0-P1**。分母は生成試行数ではなく独立Obligation |
| Schemathesis | OpenAPI/GraphQL/gRPC定義から正負・状態連鎖APIテスト | TEI公開APIのprotocol-level反証・unexpected status/schema・再現request | **OpenAPI等が実在するTargetで試す/P1**。API仕様は業務意味正本ではない |
| Pact | Consumerが要求する相互作用を契約化しProviderに対して検証 | Consumer/Providerの版別互換性・要求された通信の実到達 | **外部Host/Provider接続時/P2**。契約テストだけでCommit後実状態や業務権限を証明しない |
| Testcontainers for Go | テスト内でPostgreSQL等を起動・終了し依存を管理 | 実DBの隔離実行・再現可能fixture・cleanup | **導入比較P0**。Docker実行前提を一般CLIの必須条件にしない |
| SmartBear ReadyAPI | APIモックとサービス仮想化、遅延・障害などを再現 | 不在/故障Providerの制御可能な負例／実サービスとの差の見える化 | **技法を採用候補/P2**。実接続試験と別Profile |
| Toxiproxy | Go製TCP proxyで遅延・切断などを故障注入 | ACK喪失・重複再送・Receipt未知・復旧の故障時系列試験 | **外部PoC/P1**。テスト環境のネットワークに閉じる |
| Stryker | killed/survived/no coverage/timeout等Mutantの区別 | Known Mutant Ledger、Target Test IDまでの検出証拠・未判定の保持 | **方法採用/P0**。TEIのGo誤実装をStrykerが直接実行できると仮定しない |
| Playwright + axe-core | Traceのbefore/after DOM・network・source、アクセシビリティ検査 | CLIのScenarioから派生するWeb E2EでAction/DOM/Network/Resultを分離 | **外部Runner候補/P1-P2**。スクリーンショット・a11y結果だけでEffect確定しない |
| Allure Report/TestOps・Qase | Case/Run/Result、履歴・retry・添付、Test planやdefect関連づけ | Evidenceの履歴表示、実行試行ごとの結果、旧版との比較、手動受理レビュー | **投影設計の参照/P2**。SSOTやOwner MeaningのAuthorityにはしない |
| ReportPortal | 失敗結果の自動分類・重複傾向・トリアージ | 大量並列AI生成テストの診断クラスタリング・flaky候補整理 | **レポート連携候補/P2**。分類AIはPASSの再判定権限を持たない |
| Grafana k6 | workload scenarioと性能のthresholdを明示しCIの失敗条件化 | CLI/Provider/APIの性能・負荷・回帰を機能の適合と別軸で評価 | **実Target成立後/P2**。未定義SLOの速度だけで品質を主張しない |
| Promptfoo | AI agent/LLMの入力セット、assertion、差分実行、red-team | AI生成Scenario/モデル提案の候補品質・指示追従の評価 | **AI生成工程限定PoC/P2**。モデル採点は業務正解の独立根拠にならない |

公式資料：
- Tosca Requirements <https://docs.tricentis.com/tosca-2026.1/en-us/content/requirements/rqm_introduction.htm>／TestCase-Design <https://docs.tricentis.com/tosca-2026.1/en-us/content/testcase_design/testcase_design_intro.htm>
- GraphWalker <https://graphwalker.github.io/graphwalker-rs/>、PICT <https://github.com/microsoft/pict>
- rapid <https://pkg.go.dev/pgregory.net/rapid>、Go fuzz <https://go.dev/doc/tutorial/fuzz>
- Schemathesis <https://schemathesis.readthedocs.io/en/stable/>、Pact <https://docs.pact.io/>、Testcontainers <https://golang.testcontainers.org/>
- ReadyAPI <https://smartbear.com/product/ready-api/api-virtualization/>、Toxiproxy <https://github.com/Shopify/toxiproxy>
- Stryker <https://stryker-mutator.io/docs/mutation-testing-elements/mutant-states-and-metrics/>、Playwright <https://playwright.dev/docs/trace-viewer>
- Allure <https://allurereport.org/docs/history-and-retries/>、Qase <https://docs.qase.io/en/articles/14442829-qase-concepts>、ReportPortal <https://reportportal.io/docs/analysis/>、k6 <https://grafana.com/docs/k6/latest/using-k6/thresholds/>、Promptfoo <https://www.promptfoo.dev/docs/getting-started/>

## 4. TEIに不足しているテスト設計機能（機能は既存Ownerへ分配）

### 4.1 Requirement/Obligation/Coverageを分けたread-only投影（#2514/#1180）

受理済みRule、Data構造、Function Operation、Business Procedure、状態遷移、Host Capabilityを**参照して**、検証要求（Obligation）を列挙する。新しいMeaningは生成しない。Obligationの必須/任意、範囲/時点、Target、Criticality、必要なObservationを列にする。

「宣言されている対象」「Sourceとして発見された対象」「実Loaderが読んだ対象」「必要なStepが生成された対象」「実行された対象」「観測結果が証拠を持つ対象」「現在資格がある対象」の分母を別管理する。**未発見・未読込・SKIP・unknown・not-disclosedを100%に含めない**。

Toscaのrisk weightはスケジューリングに利用するが、重みを上げても未確認をPASS化しない。出力はCaseカタログ、Requirement Coverage、Mutant判別、Target Parity、Qualificationを**別タブ相当の投影**にする。

### 4.2 テスト設計法をCandidate Generatorに分離（#2514/#1180/#1131）

- 固定独立Expected：必須の基線。評価器/AI出力からWantを生成しない。
- 境界値・同値分割：金額・丸め・NULL・時点・範囲境界。
- PICT相当の組合せ：Tenant × Grant × Presence × Collection completeness × Target等。制約を明示し、候補生成seed/入力を固定。
- GraphWalker相当の遷移：提出、承認、差戻し、取消、再送、失効、手動待機。辺到達と実Effect再読取を別扱い。
- 性質ベース/Go fuzz：assertionは独立不変条件。failed seedを最小化し、通常Scenarioの回帰候補として登録する。
- Metamorphic（変換前後の関係の検証）：合法な行順入替／分割JOIN／集計グループ再構成等、**意味を保つとOwnerが定義した変換に限定**。時間、bag重複、TOP、権限の変換を無条件の同値としない。
- 差分テスト：Go/Java/.NET/SQL等を同一固定Inputで比較。Target間一致のみでは正しさは確定しない。独立Expectedが必要。
- Mutation：意図的誤実装の種類・対象ファイル・baseline HEAD・mutant HEAD・Test名・診断・CI結果・安全隔離を記録。

AIによる生成は**提案**。Candidateを受理済みDriver Scenarioへ自動昇格させない。失敗ケースを修正する際、ExpectedをActualへ追従させない。

### 4.3 テスト結果・Evidence・Qualificationの三段階（#1163/#2514）

1. **Test execution result**：対象Caseが実際に呼ばれたか、PASS/FAIL/SKIP/TIMEOUT/BLOCKED/NOT-RUN、実argv、実Target、実入力。
2. **Evidence**：Source/Case/Fixture/Expected/actual build artifact/Plugin/Target/Provider/Consumerのbyte identity、Run ID、観測、raw/typed、Receipt、独立read、時点・開示範囲。特に結果に触れる実Codeの出典と、CI一般成功を分ける。
3. **Qualification**：どのClaimに対して対象集合・母集団閉包・権限・意味受理・鮮度の必要条件を満たすかを判定。履歴PASSと現在資格を区別。

単一の `passed=true`／百分率を総合資格の代替にしない。外部Allure/Qase/ReportPortalへ出力する場合も**派生ビュー**であり、元Evidenceと現在の資格を失わない。Retryは前回Runを上書きせず別Attempt。Flaky分類は「成功」と「再試行後成功」を同値にしない。秘密入力・Expected・holdoutを公開レポートへ添付しない。

### 4.4 Target間の同一Scenario・固有観測（#1180/#2391）

Scenario→typed Step/Operation→CLI argv／Web Playwright／Desktop専用Adapter／API contract-testへ派生。各生成器は**未対応Step・失われたBranch・拒否Case・Observation**を機械的に報告する。Target固有Locatorや画面文言で業務Expectedを複製しない。

WebはDOM/role/a11y/trace、Desktop/Canvasは実Host/UI操作・アクセス可能性、CLIはargv/stdin/stdout/exit、DB/Providerは実SQL/row set/transaction、外部Targetは実JAR/Assembly/Consumerを別のEvidenceとして保存。実UI操作を内部Command直呼びで代替しない。Production Buildにtest-only instrumentationを残さず、外部WebDriver・通常a11yは対象外として区別する。

### 4.5 資格を破る状態・効果の故障注入（#1174/#2506/#2586）

- 応答が届かなくてもEffectが既にCommit済み／未Commit／unknownの各Caseを区別。
- 権限が事前確認後に失効したとき、実Effect直前のHost側拒否を検証。
- `$top=1`、必要列の欠損、同件数異集合、page欠落、Replica遅延、非開示、Tenant相違で「全件真」を宣言しない。
- 同一Operationのretry／取消／部分成功／冪等性は、Providerが実行したか、Receiptを受けたか、独立Resource readで現在確認したかを分ける。
- Testcontainers/ReadyAPI/Toxiproxyは負例の制御手段。実Provider end-to-end試験を省略する根拠にならない。

## 5. 最初に固定する独立ExpectedとMutant候補

| ID | 入力・変異 | 独立して期待する判別 | Owner/対象 |
| --- | --- | --- | --- |
| TT-01 | relation key `[1,1]`。正常と判定反転 | 正常は重複拒否、Mutantは対象AI Stepで検出 | #2514、既存#3017 |
| TT-02 | 最終要素にだけ重複を置き、Evaluatorが末尾を無視 | AI Stepが具体的な誤判定を検出 | #2514、既存#3128 |
| TT-03 | 全件 `{A/3,A/9}` に対し `[A/3,A/3]` を返す | 同件数でも集合閉包不明→全件成立はunverified | #2506、#3175 |
| TT-04 | `$top=1`／`$select=orderId` に `status=READY` Claim | 先頭1件と必要列未取得から全件真を主張しない | #2506、#3175 |
| TT-05 | raw JSONで同名member・trailing token・missing/null | raw構文の受理と型付き入力・業務Ruleを分ける | #2514、#2077 |
| TT-06 | 同一業務操作でCLIは成功、Webは権限失効・UI stale | 経路固有失敗と意味・Host Grantの違いを報告 | #1180、#2391 |
| TT-07 | 送信→DB Commit→応答遮断→retry | ACK不明≠未Commit。Receipt／独立readで効果を調べる | #1174、#1163 |
| TT-08 | 同一Source・同一CaseでBuild側のみ変更、Consumerは旧Binary | byte一致を外れた実消費を検出し現在資格を拒否 | #1163、#2016 |
| TT-09 | 同じ業務グラフで辺欠落／順序逆転／循環 | 意味上の許可遷移とPath coverageを照合し不正経路を検出 | #1131、#1125 |
| TT-10 | 集計結果を異なる粒度で再集計、JOIN多重度を増加 | 正しいsum/count/grainの保存。値一致だけでは不可 | #2593、#2514 |

各Caseに正例・負例・必要時inconclusiveを持たせる。すでに実証したTT-01/02等を再実装せずEvidenceの参照に留める。TT-03/04の実Provider適用、TT-06〜10の正式横断E2Eは**NOT RUN**。

## 6. 実施計画と完了条件

### P0：既存能力の計測・最小導入（新正本なし）
1. 既存 `scenario verify/generate/run/check` とGoテストの対応を確認し、1 ScenarioについてRequired Stepの列挙/実行/証拠の差分表を出す。
2. #2901正常基線、#3017/#3128のMutant実証を、Case IDとbinary/source/CIログ付きの**read-only表示**で参照。Mutant PRは保持・merge禁止。
3. Go rapidまたはGo標準fuzzで、JSON raw/typed、NULL/partial、Collection identityについて**最小1性質・正負1組・shrunk counterexample**を試す。対象APIに合うものを選び、標準依存はPoC後決定。
4. Testcontainersの実PostgreSQLと既存fixtureによって`Provider read→Result Slice→Claim`を確認する（#2586等の担当と協調）。Dockerを基本CLIに強制しない。

完了は**実Go/CLIのテスト名、実入力、固定Expected、Run/HEAD、ログ、観測の到達範囲**で判定。計画文書だけではP0達成としない。

### P1：テスト設計の拡張（候補生成は独立）
1. PICT/GraphWalker相当の生成を、同一Scenarioの**候補Case提案**として比較。seed/constraints/coverage denominatorと非生成領域を残す。
2. ToxiproxyでACK-loss/再送/遅延、Schemathesisで仕様ベースのAPI負例、PlaywrightでWeb固有のtrace/a11yをそれぞれ1最小Referenceとして測る。
3. 固定Independent Expectedと3種以上の既知Mutantで、killed/survived/no-coverage/inconclusiveを分類。CI失敗だけでkilledと判定しない。
4. ScenarioからCLIと別Targetへ派生したStepごとに、Generation parity／executed parity／observed parityを**別列**で検証する。

### P2：Evidenceの利用・継続運用（採用方式は後決定）
1. Allure/Qase/ReportPortal等のような履歴・Attempt・defect関連づけを、既存Evidenceからread-only投影する。既存Repo/Issueを正本に残す。
2. 変更影響に応じたリスク優先テスト選択、Ownerの受理と自動Runの分離、sealed holdout漏出検査を加える。AIがExpected/Assertionを無通知更新した場合は失敗扱い。
3. k6による性能SLO、PactによるConsumer版互換、外部Host型付きAPIの契約テストを対象に応じて展開。テスト製品の購入・ライセンス/運用費用は、PoCの増分検出件数・保守時間・依存負担が得られてから判断する。

**受入ゲート（全段階共通）**：何を検査したか／実Source・byteと版／独立Expectedの出典とMeaning／どのCaseを実行したか／どんな誤実装を判別したか／どの母集団とTargetにだけ結果を使えるか／何はunknownか／誰が受理するか。この8列を欠く報告は、数値が高くても全体合格として扱わない。

## 7. 所有境界と文書反映

- #2514：独立検証基準、Known Mutant台帳、検証要求の判別力とClaim適用範囲。
- #1180：ScenarioのSSOT、Case候補からの受理、CLI/Web/Desktop生成差分とCoverage分母。
- #1163：同一改訂Source/Build/Run/ConsumerのEvidence・再資格。
- #2506/#2586：実Read Footprint、必要列、母集団閉包、Data Context→Claimの縦切り。
- #2391：外部Runner、UI trace/a11y、test instrumentationの実Build除去。
- #1174：Effect/Receipt/ACK loss/再送、現在Host Grant。
- #1445/#2515：大量並列AIのテスト工程・秘匿・実OS到達境界。
- #1131/#1125：状態・業務手順の実遷移と外部/手動Observation。

本書は専門製品の機能を所有Issueへ分配する**設計インデックス**。新製品の購入、共通サービスの新設、計画の一括移動、既存Issue本文の置換、意図的Mutantのmain統合を承認するものではない。


---

<!-- tei-test-cross-chat-witness-20261107 -->
## 8. 2026-11-07 他チャット横断：実証の進展を反映したテストの受入単位（追補／設計）

本節は、前節P0〜P2を置き換えず、最近のmain実装・他担当Draftに照らして「**何が検証済みか／どう検証が欠けているか**」を明確化する。参照先の設計ケースやEvidenceを複製しない。

### 8.1 現在の基線と、過去の未実装記述を混同しない

| 参照 | 確認できる限定事実 | 本書に残す未証明 |
| --- | --- | --- |
| main [#3155](https://github.com/hat47x/tei-platform/pull/3155) | Commit receipt再取得の `receipt-observed` と、別接続Resource現在値/Payload一致による `resource-observed` を分離。後者のみ `persisted:true`。Go試験・対象CIで実証。 | 更新対象全件、Versionの因果・時点・現在Read Grant、Owner業務完了。**旧「保存観測未実装」の記載を現在形で繰り返さない**。 |
| main [#3145](https://github.com/hat47x/tei-platform/pull/3145)、[#3167](https://github.com/hat47x/tei-platform/pull/3167) | `function-definition` pattern増加でも `tei operations` が8件 derived/0件 declaredのまま。Data IDだけ部分照合、Capability/Relationは一致せず。 | 宣言Pattern→Typed Operation→Action Offerの意味的結合、実Provider/Host。**未接続を「テスト対象0」と扱わない**。 |
| main [#3078](https://github.com/hat47x/tei-platform/pull/3078)／[#2621](https://github.com/hat47x/tei-platform/issues/2621) | Workspace file inventory、`readByLoader`等により発見/Loader消費を区別する経路が前進。 `*.tei.md`へ統一する方向。 | 実Parser Role、Accepted Meaning、Builder実入力・Consumer実使用、旧形式と新形式の等価性。 |
| [#3175](https://github.com/hat47x/tei-platform/pull/3175) | `EvaluateCompletion` に対する同件数異集合／TOP-1／述語列未取得の実Go API characterization。 | 合成rowsetでありFunction→Query→実Provider取得の閉包は未成立。 |
| [#3192](https://github.com/hat47x/tei-platform/pull/3192)、[#3222](https://github.com/hat47x/tei-platform/pull/3222)、[#3225](https://github.com/hat47x/tei-platform/pull/3225) | Shape／Representation Type／Function内Instance/Slot／Mappingと媒体軸／複数表示の協調を別契約に分けた設計（Draft）。 | Renderer・Provider・媒体同期・実更新のE2Eは未実証。RP/MVケースは各設計Ownerが保持する。 |
| [#2515](https://github.com/hat47x/tei-platform/issues/2515)／[TAI連携](https://github.com/hat47x/tai-pleroma) | 外部HarnessのTool結果切断・出力混在・Call/Result不一致、機密Evidenceの二次送出を、比較資格・開示の独立問題として特定。 | 実OS/全出口の隔離、全Step完全受領、H2/H3の独立比較資格、未計測Sinkの不存在。 |

古いDraftの「未実装」「NOT RUN」表現を現在mainへ無検査に継承しない。逆に、複数のPR/CIから得た局所PASSを別Attemptの証拠として合成しない。

### 8.2 第一級にするのは「検証要求の到達状態」、新Test Runnerではない

検証要求ごとに次の状態を**別観測列**として保存・表示する（以下は候補の表示概念であり、新しい永続Schema/第二Registryではない）。

1. **Required**：独立要求Source/Ownerのどの規則・対象についてテストが必要か。Required集合の根拠はCandidate Modelから逆算しない。
2. **Discovered/Loaded/Parsed**：Manifestが宣言したSource、列挙された実Source、Loaderが読んだByte、Parserが解釈したOccurrenceを分離。未Loader消費・未知Root・未列挙の場合、業務Missingとは断定せず `source-incomplete / unqualified`。
3. **Meaning/Binding**：Accepted MeaningとFunction Context、Relation Path、Pattern/Operation、Targetに接続できたか。表示できたこと／同Data IDであることから、Operation実行可能としない。
4. **Generated/Executed**：Driver Scenarioの必須Stepが各Targetの生成物へ落ち、実Runで各Stepに到達したか。単なるCase件数、exit 0、CI成功と分ける。
5. **Observed**：Business Result、Execution Error、UI状態、Provider rowset、Commit receipt、別接続Resource read、Tool受領の**各種類**と、対象同一性・時点・完全性を記録。未観測は不在でない。
6. **Qualified**：Independent Expectedの独立性、Known Mutant判別力、実Source/Build/Consumer一致、母集団/権限/Disclosure/Owner受理のうちClaimに必要な条件を確認。Scope外はunknownまたはunqualified。

これは一直線のPASS昇格ではなく、**各関係を根拠付きでjoinする評価票**。高位列が真でも低位列の欠測が残れば全体の主張を拡張しない。旧`PASS`・`resource-observed`・`Verified`を一つのBooleanへ還元しない。

### 8.3 標準データ構造と複数表現からテストを導出する際の制約

**Test Obligationを導出するための観点**（データ意味の新正本を生成するわけではない）：

- **Data topology**：Record/Collection/Relation/Graph、identity、関係多重度、grain（Entity/Relation occurrence/Path/Aggregate）、NULLと未取得、bag・JOIN増幅、Set closure。
- **Presentation topology**：Table/Tree/Graph/Chart/Panel/Tab/Audio/Video/Textの**型**と、Function内に実配置された**インスタンス**の関係。Headless Functionは0 Representation Instanceでもよい。
- **Binding topology**：Function Data ContextのRelation path/Field/Scope/必要列/epoch、Representation occurrenceとData identityの写像。ローカルSelection/Focus/SeekをDomain Mutationや正式Action Targetへ自動昇格しない。
- **Time/Media axes**：原資料の時間、取込/取得時点、再生時刻、文書内位置、同期用に生成した時間軸を混同しない。文書位置→音声再生時間は一般に情報を付加する派生写像であり可逆同値ではない。
- **Operation/effect**：どの表示が操作候補を出しても、実Operation binding、Host Grant、Effect実行、Receipt、Resource readが別条件。UI成功と保存完了を同一視しない。

この節は[#3192](https://github.com/hat47x/tei-platform/pull/3192) RP-01〜08や[#3222](https://github.com/hat47x/tei-platform/pull/3222) MV-01〜08のテスト定義を**保守しない**。本書はそれらの独立Expectedを#2514/#1180の要求に接続し、どの種別の観測が必要かだけを示す。

### 8.4 実行履歴と証拠の因果（ACK loss・分散Operation・外部Harness）

Jepsenは実SystemへのOperation invocation／completionの履歴と故障注入、独立Checkerを分ける。Hypothesis statefulは単一入力だけでなく複数Actionの順序と各Step不変条件を探索する。TEIではこれらを**既存Scenarioに対するテスト設計技法**として参照し、別の本番Workflowエンジンは作らない。

- ケースの順序、並行性、Operation ID、Invocation、Provider実Effect、応答の到着/欠落、Retry、Receipt lookup、独立Readを区別。応答喪失を未実行・失敗と推定しない。
- OpenTelemetryのTrace ID/Span/Linkは外部Runとの相関に有用。ただしTrace IDの存在をSource/Artifact同一性、Observerの完全性、現在Grant、独立Resource読取の**証明とはしない**。Samplingや紛失、複数Traceへの分断、Observer遅延を負例とする。
- TAI/DSH等のHarnessでToolが成功したように見えても、stdoutのJSON 8,192 byte切断、stderr混入、Call/Result件数不一致、Tool reply未受領なら`complete`にしない。**Tool実行と、検証器が実際に受領した全内容**を別Witnessにする。Runの途中で未受領のStepを暗黙に補完しない。
- Test・Trace・CI・Issueが機密値を含む可能性を考え、保護された生Evidenceと公開用の集約結果を分ける。非公開Expected/行数/拒否理由/秘密由来Digestを無条件に公開しない。全Sinkの監視範囲は#2515を参照する。

参考一次資料： [Jepsen core](https://jepsen-io.github.io/jepsen/jepsen.core.html)、[Hypothesis Stateful](https://hypothesis.readthedocs.io/en/latest/stateful.html)、[OpenTelemetry Overview](https://opentelemetry.io/docs/specs/otel/overview/)。

### 8.5 OKF Attested Computationとの接続（DocumentとRuntimeの責務分離）

[Open Knowledge Format v0.2仕様 §10](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)の`type: Attested Computation`は、計算内容・runtime/parameter・executor/receipt・決定論的attesterをConceptで参照し、実行Receipt自体はbundle外のRuntime Artifactとして扱う。**説明可能な計算契約と、実行の確証を分離する点**を採用候補とする。

TEIでは `Source+Accepted Function+Independent Expected` → `既存Driver/外部Hostでの実Run` → `Evidence/Receipt` → `決定論的検査` → `Claim相対Qualification` の関係をread-onlyに投影する。SHOは文書内リンク・引用を扱い、SEIは知識Claim/Evidenceの参照を担う。**OKFのfrontmatterに書いた`verified`やattester PASSを、Ownerの業務意味受理・現在Host Grant・実DBの完全性へ自動昇格しない**。OKFの軽量互換要件を破る必須TEI独自fieldや独立実行Registryを設けない。

## 9. 追加の独立検証基準（TT-11〜TT-20：別IssueのCaseを参照する受入票）

| Case | 入力・意図的欠損 | 独立して期待する結果／拒否する過大主張 | Owner・証拠 |
| --- | --- | --- | --- |
| **TT-11 Source closure** | Required`*.tei.md`をmanifestに宣言するがLoader/Parserが消費しない。別例は旧`*.tei`と新`*.tei.md`の同Subject異内容。 | validateが局所成功してもRequired完了ではない。未読込は`source-incomplete`、曖昧な二重正本はunresolved。 | #2621/#2823/#1180。Input byte・Loaded occurrence。 |
| **TT-12 Pattern→Operation** | Function patternを1件追加、`tei operations`は既存8 derived/0 declaredのまま。 | 追加機能の実Operation到達をPASSにしない。`binding unresolved`。成功件数の分母に未接続要求を含める。 | #2507/#1180、main #3145/#3167を再利用。 |
| **TT-13 Topology/grain** | 同一親のD1=2件・D2=3件、JOINでは6行。Chartは集計1行。 | JOIN6≠子実体6。Aggregate行に逆写像・対象確定が無い場合、更新Actionを導出しない。 | #2506/#2497/#3192 RP。 |
| **TT-14 Cross-view epoch** | TableでA選択→B選択。Aの遅延Read応答がBのContextに到着。Tabはpartial。 | stale AをBへ混入・同期成功と認定しない。partial Tabの非表示を「対象なし」としない。 | #3222 MV、#2506。 |
| **TT-15 Graph occurrence** | 同一Entityが異なる2本のRelation pathでTreeへ重複表示。 | 同一表示行番号・同labelのみでEntity/Update対象と推定しない。Path occurrenceが必要。 | #2497/#3222 MV。 |
| **TT-16 Media-time** | 文書の行位置とTTS/録音再生timeを関連づけ、テキスト更新後に旧Seek情報を再利用。 | 元Source位置≠再生time。写像根拠と版が無い旧Seekを現在の同時刻と扱わない。 | #3192 RP、#3222 MV、SHO/SUI側。 |
| **TT-17 Persistence state** | ACKを失うがreceiptは取得、Resourceは未読取／別接続で一致／後刻に別値へ更新。 | `receipt-observed`はpersistedを主張しない。Resource一致時のみ限定した`resource-observed`。後刻の現在値・全関係・Owner完了は別判断。 | main #3155、#1174/#1163。 |
| **TT-18 Harness result completeness** | Tool実行は成功、stdout JSONが8,192 byteで切断。56 Callに55 Reply、stderr混入。 | 完全受領と宣言せず未観測/要再照合。exit 0を独立Expected・完全Coverageの代替にしない。 | #2515/#1163、Tool受領範囲。 |
| **TT-19 Evidence as egress** | Mock canaryをTest/Tool/CI Traceに載せ、公開Issue向けReportを生成。 | 内部Evidenceの保持と公開Disclosureを分離。非許可Raw/拒否理由・秘密由来Digestを公開しない。未監視Sinkはunknown。 | #2515 H6-E13〜E15、#1178。 |
| **TT-20 History/checker** | 並行するsubmit/cancel/retry、Grant失効、ACK-lossを固定Historyと独立モデルで検査。 | Invocation/Completion/Effect/Observerを別列にし、停止時刻や部分観測を残す。合成HistoryのPASSで実DB線形化・全Workflow適合を主張しない。 | #1174/#1131/#1163。 |

各Caseに **positive / negative / inconclusive**、独立Expected、必要な実観測、現在対象資格を付記する。Ownerの既存RP/MV/H6/SourceテストとCase ID/fixtureを複製しない。上記TT-11〜TT-20は**受入候補であり今回の実行結果ではない**。

## 10. P0/P1/P2の完了基準を現在の実績へ合わせて更新

### P0最初の縦切り（現行main再利用。新Runnerなし）
1. **Closure表を読み取りで作る**：代表するAccepted RequirementとDriver Scenarioを1つ固定し、Required/Discovered/Loaded/Parsed/Bound/Generated/Executed/Observed/Qualifiedの各列を、現在の既存CLI/Go/Fixtureから**実観測またはunknown**で埋める。元Source byte、実readByLoader、必要なStep ID、Tool結果受領範囲を必ず別列。
2. **既存Mutationを参照**：#2901 Green、#3017/#3128 Redの実Case診断を区別する。新たな破壊的Mutantのmain取込み禁止。既知Mutantは各Claimの局所感度でしかない。
3. **実Provider境界**：#3175の合成Q05〜07は実Go characterizationで成立。次の本命は#2586の**同じFunction→Query→Provider実rowset→Result Slice→Claim**を同一Attemptでつなぐ。main #3155のResource readを再実装せず、別の保存Operation/Readで再利用できることを確かめる。
4. **Harnessの最小観測**：1件の代表Scenarioで実Tool request/responseのCall ID、受領byte、truncation、stderr、Result completenessを記録。機密なrawの公開は行わない。

### P1追加
- **Representation topologyテスト**：TT-13〜16をheadlessでData/Presentation/Binding別に判別し、CUI/Studio/SHO等の外部Consumer適用は独立に再評価。
- **Action sequence/property test**：Hypothesis stateful型の「一操作の値」ではなく「複数Operation順序」を参考にし、TEI Go用には既存Goテストに連結できる実現法を比較。時系列と並行HistoryのCheckerを分ける。
- **History fault injection**：Toxiproxy/Jepsen的fault injection→実Effect/Receipt/Resource observationの時系列を一Attemptで記録。モデル上の整合と外部Hostでの実証を別Profileとする。

### P2追加
- **Attested Computation投影**：OKF/SHO/SEIへ計算意味・Parameter/Executor・証拠種別・決定論的Attester判定・現在資格を*別軸*で表示。OKFを新Authorityにしない。
- **TraceとEvidenceの区別**：OpenTelemetry等の分散Traceは実行相関の補助。Source/Build byte、Independent Expected、実Effectと現在Grant/Disclosureは別の証拠。
- **秘匿・再資格化**：holdoutやTool/CI Evidenceを公共出力へ無条件伝播しない。Source/Plugin/Grant/Owner意味が変わる場合、旧PASSを保持しつつ現在資格を再評価する。

### 検証したと言える最小条件

単一のClaimに対して **必要なSourceの閉包／独立Expectedの妥当性／対象Case全件の生成・実到達／実SourceからConsumerまでの入力同一性／実観測の証拠種別・完全性／適用Scope・Tenant・時点・現在Authority** が束縛できる。欠けた列は明示的にunqualifiedとして残す。検証実験の局所PASSや外部Toolが「completed」を返しただけで全体資格を合格にしない。

**状態**：この追補は他チャット・GitHubの既存実証結果と外部専門技法を結んだ**設計計画**。今回の新しいGo/CLI/Provider/GUI/Media/Harness/OS隔離のE2Eは **NOT RUN**。正本（Accepted Canonical/Driver Scenario/Meaning/Host Grant）は既存Ownerに保持する。

<!-- tei-test-model-derived-obligations-20261107 -->
## 11. 2026-11-07 横断深化：ドメイン知識による検証要求・テストデータ生成と広域整合性（採用前の設計）

### 11.1 要求の導出元を固定し、候補生成と受理を分ける

三要素（Data / Function / Business）、値域・単位・強属性、Key/FD/Relation、業務Rule、Procedure/Workflow、組織/規範、Provider/Host/Consumerのモデルから**Verification Obligation候補**を導出する。ただし、モデルに存在しない業務意味・許可や「何が強属性か」をAIが補完して受理済みにしない。「強属性」は当該Accepted Modelが具体的に定義する性質・適用対象・反例を参照できるまでは、テスト観点の名称にとどめる。

- **業務Meaningの原典**：Accepted CanonicalのRule/Attribute/Relation/Key/Procedure、Authorityと対応Source Revision。暫定Model由来はprovisionalを明示。物理PK、業務Key、複合Key、Relation occurrence、表示行IDを識別・同一性の代用にしない（#1126）。
- **テスト要求の原典**：意味Ruleと利用シナリオに加え、Source/Read Closure、対象集合の完全性、時点、Host Grant、実Provider Effect/独立Observation、開示境界の各必要条件。Required集合を「生成器が作れたCase数」から逆算しない（#2514/#1180/#2506）。
- **候補生成**：型と値域の境界、NULL/未取得/非開示/空、正常・異常遷移、Relation/cardinality/FD、Rule間干渉、時間の前後、Membership/Delegation/Tenant、故障・順序・履歴、Metamorphic relation、Known Mutant。値域組合せを総当たりせず、各境界の独立対照＋意味のある交互作用のペア/高リスク多条件を選ぶ。
- **独立Expected**：Candidate/実装/同一EvaluatorからWantを自己生成しない。原契約、手計算・別アルゴリズム・独立データセット・Ownerレビューを使い、Expectedの定義域・出典・反例判別力を記録する。AI同士の一致や多数決は独立の根拠にならない（#2514）。
- **Scenario化**：提案Caseは既存Driver Scenarioの変更候補としてReviewし、採択後に同じScenarioからCLI/Web/Desktopへ投影する。未対応Stepや失われた分岐をsilent skipしない（#1180）。新Test DSL、永続Case Registry、新Runnerは作らない。

比較票の候補列は、Requirement Ref / Accepted Meaning Revision / Claim Scope / Obligation ID / Derivation Reason / Positive-Negative-Unknown / Expected Provenance / Fixture Ref / Scenario Revision / Target Step Binding / Observer / Actual Input Binding / Qualification / Limitations。これはread-only投影であり、公開Schema・新意味正本の採用ではない。

### 11.2 「生成済みのもの」ではなく「必要なもの」に対するCoverage

一つのClaimに対し、**Required → Candidate-generated → Owner-reviewed → Scenario-bound → Target-generated → Actually-executed → Actually-observed → Independently-checked → Currently-qualified**を別段階として扱う。各段の分母、除外された対象、欠落理由を出力する。Function成功後に手作業が残ればBusiness Procedureの完了Coverageには算入しない。Source未読込、Target非対応、欠けたTool Replyも分母から黙って除外しない。

- ケースが存在しない／生成済みだが未受理／Step未生成／未実行／実行したが未観測／観測したがClaimを支えない／旧Qualifiedが現在stale、を別理由として扱う。
- Rule・Relation・Branch・State・Time・Role・Grant・Source・Provider・TargetのCoverageは**母集団が異なる**。総合一数値で要件不足を隠さない。Mutation scoreは既知Mutant集合への局所感度。
- 存在証人は部分取得から成立する場合があるが、全称・不存在・全件集計は必要集合の閉包を要求する。空、未取得、非開示を「0件・真」と同一視しない。
- JOINによる粒度崩壊、表現型の誤った書戻し、時間/Grantの失効、Function成功≠Business完了は互いに独立する未達経路として追跡する。

### 11.3 独立正負対照（TT-21〜TT-29）

以下は**既存OwnerのFixture/Case IDを置換しない受入要求**。合成の値・業務Ruleは正式TEI意味としてOwnerが採用したものではない。**すべてNOT RUN**。

| 要求 | 合成入力・意図的差分 | 独立Expected／防ぐ誤主張 | 所有境界 |
| --- | --- | --- | --- |
| **TT-21 値域・属性意味** | 自作数量属性に仮Rule「整数、1以上」を置き1/0/負/小数、missing/null/未取得/非開示を区別 | 仮Ruleでは1のみ適合。入力変換・Rule拒否・取得不足は別結果。強属性等の意味が未Acceptedならprovisional | #1126/#1705/#2514 |
| **TT-22 Key/FD・粒度** | 同じ業務Keyの別物理PK、同Nodeの別Path、同額100の別Line、FD違反 | PK一致や値一致から実体/辺/集計対象を同一視しない。Key/FD未受理なら制約を推定しない | #1126/#2497/#2593 |
| **TT-23 三要素・手作業** | 仮Procedure「Function実行→人の確認→承認」、Function成功後も手作業未了／TaskEventとFunctionLaunchEventの差 | Function ReceiptだけでBusiness完了としない。人の作業を架空のFunctionに変換しない | #1125/#1180/#1131 |
| **TT-24 組織・時点・Tenant** | 同人が二法人に所属、代理権失効前後、自己承認、Tenantごとに同じCase ID、Preview後Grant失効 | Membership/Position/Delegation、業務Rule、実Host Grantを分離。Resource/Tenant/Timeが欠ければunqualified | #2510/#2386/#2390/#2514 |
| **TT-25 広域Ruleの読取閉包** | 親HのLine L1=100/L2=100は完全、独立Deliveryは3件中1件、非出力status/Grant未読込 | Line合計200は必要なLine閉包が証明されたClaimに限る。Delivery全件条件や未読込status依存Ruleは未確定 | #2506/#2586/#2582 |
| **TT-26 状態・同時更新** | 選択A→BとAの後着応答、承認/差戻し競合、ABA、expected-state一致/不一致、取消/再送 | stale混入、後勝ち上書き、未証明の順序を拒否。合成Checkerは実DBの線形化を証明しない | #1131/#2485/#1174 |
| **TT-27 派生・確定・効果** | Preview 200→承認Revision 200→保存、途中のRule改訂、ACK喪失、後続訂正 | Preview・承認済み値・Typed Request・Receipt・独立Resource Read・Owner完了を別Claimとする | #1129/#1174/#1163/#2671 |
| **TT-28 Source・負の依存** | 必須の新形式tei.md未読込、旧形式の競合、DB別SQLにspecificを追加、未使用Plugin変更 | 実選択が変わる依存では再評価。無関係な変更で過剰失効しない。依存未列挙ならunknown | #3030/#2621/#2586/#2582 |
| **TT-29 AI候補の自己検証** | AIが境界5 Caseを提案しWantに出典がない、末尾Case欠落、Tool Reply切断、holdoutを公開 | Source要求→提案→Review→Scenario版→実Target到達を照合。不確かなExpected/欠落Step/不適切開示でqualified不可 | #2514/#1180/#2515 |

TT-21〜29は既存MC/FC/ORX/RP/MV/H6/OrganizationおよびTT-01〜20のCaseをコピーして別SSOTを作るのでなく、**既存Referenceのどの出典・Case・Observerが要求を満たすか**を参照する。Owner固有のExpectedがない場合は合成Expectedをprovisionalにする。

### 11.4 P0を一つの縦断Fixtureへ収束させる

既存#2586のH／L1=100・L2=100／独立D2=3行の自作Fixtureを再利用する。追加の承認・Tenant RuleはAccepted Meaningの表現能力を問うための**仮置き**であり、Owner受理まで正式Expectedにしない。

1. **Source/Meaning**：必要なtei.md、Data/Function/Business、Rule、Key、各Revisionの出典とLoader実消費を確認。未定義の承認閾値・値域をAIが埋めない。
2. **Read/Claim**：D1の完全2行ならLine合計200。D2部分1/3ならD2全称はnot-established。JOIN由来600とSUM(DISTINCT amount)の100をLine合計と誤認しない。
3. **Function/Procedure**：Function成功後も手作業/承認/Host Grantを未確定に保つ。AIに渡すObligation表にも未完了Stepを残す。
4. **Action/Effect**：実書込みへ進むのは隔離Fixtureと正式Grant・Operation Target Bindingを確認した場合のみ。既存EffectCommit観測を再利用し、receipt-only、resource-observed、read-back-unknown、Owner完了を別々に扱う。
5. **Independent Evaluation**：独立Expected、Source→Build→Run→Consumer実入力、Tool Replyの受領、Required Step全到達、Tenant/時点の現在資格を同一Attemptで照合。欠測時は局所PASSを残してClaimはunqualified。

### 11.5 導入順序・停止条件

- **P0-a：設計の可検証化**。既存Accepted Meaning/Driver Scenario/Fixtureを参照する薄いread-only Obligation/Coverage比較票でTT-21/23/25/29の未達理由を判別。Sourceの実Loader消費が確認できなければunknown。
- **P0-b：実経路一往復**。#1180の1 Scenario/1 Claim→CLI派生、#2586のFunction→Query→実Read Slice→Claimへ接続。TT-23の手作業未完とTT-25の部分取得は混ぜず別理由で報告。main #3155の合成Resource Read成功を、別DBでの実証へ横流ししない。
- **P1：生成の品質**。Value/Time/Role/Graphの意味ある組合せをリスク別に提案。独立手計算/Oracle・既知Mutantとの比較、AIの見落とし率、人工的な偽陽性/偽陰性を測る。故意Mutantは隔離Branch/Fixtureのみに置く。
- **P2：変更影響と再資格化**。Source、Rule、Function/Procedure、Relation、非出力Query依存、Plugin/DB選択、Tenant Grantの**正負の依存辺**と実使用に基づきAffected Obligationを選ぶ。未列挙依存、A→B→A、Owner撤回は旧PASSを履歴保持し現在資格を再評価。選択実行と定期的な全体回帰を比較する。

**停止条件**：ExpectedがCandidateと同一経路、Requiredが生成済Case由来、Source未読込、Target Step silent skip、partial/非開示を全件扱い、手作業をFunction Receiptで代替、同人兼務をGrantへ自動昇格、ACK lossを未Commit扱い、旧Receiptを現在資格とする場合。新Runner/Host/Authority/Accepted Meaning Registryは追加しない。

**現状**：今回の差分は、他チャットと既存限定実証を受入要求へ**計画接続する設計更新**。新規Go/CLI/Provider/AI Harness/OS隔離/Web/Desktop実行はNOT RUN。TT-21〜29と統合P0はNOT RUN、Owner Meaning AcceptanceはNOT ACCEPTED。Draftでレビュー待ち。

<!-- tei-test-pattern-c0c1-20261107 -->
## 12. 2026-11-07 横断深化：最小指示のテストデータ・テスト規則作成とCoverageからのPattern提案（設計案／NOT RUN）

### 12.1 User Experienceを第一の受入要件にする

目標は **対象を1つ選ぶ→提案→必要な差分だけ確認→保存済みDriver Scenarioへ採択**。自然言語でも同じ経路を使い、既にFunctionを選択中なら再指定は不要。利用者が正常系・境界値・参照整合性・権限・失敗回復などを毎回列挙しないで済むよう、TEIの受理済みMeaningとRequirement/ObligationからPattern候補を逆引きする。生成AIにも同じheadless read-only入口を与える。追加質問は、意味や必要なGrantが本当に不足する条件だけに限る。

下記は**将来の構文候補であり、現行CLIに実装されているとの主張ではない**。

    tei test propose order_entry
    tei test propose order_entry --focus uncovered --budget small
    tei test patterns explain order_entry

同等の自然言語：「受注の未カバー領域を埋めるテストデータとテストを提案して」。最初の画面/CUI出力は「未カバーの要求」「上位3 Pattern」「追加すべきCaseと独立Expectedの根拠」「未受理/未観測」を短く表示する。原則として1回の選択→提案で見通しを得られ、詳細パラメータは必要時のみ展開する。UIの操作回数・Prompt token/文字数・質問往復数・Review/修正回数・計算コスト・有効提案率を測定対象とする。少ない指示で質が悪化するなら不合格。

### 12.2 Pattern Languageを標準Data Structureの「知識部品」として記述する

独立した第三の業務DSLやTest Runnerを追加せず、既存Record/Collection/Relation/Graph/Rule/Time/Facetの組合せで**Test Pattern候補**を説明・保存できるかを先にPoCする。Markdownの近傍表・短い宣言・自然言語はAuthoring Projectionであり、保存済みPatternとAccepted Meaning/Driver Scenario/Conformanceを別の正本にしない。Pluginが業界固有Patternを提供できることを目標に、Core固定enumの増殖は避ける。

**Patternの最小比較列（正式公開Schema未採用）**：

| 列 | 意味・独立受入条件 |
| --- | --- |
| Pattern ID / Revision / Origin | 知識の出典、提案者、Ownerの意味受理状態、Plugin/Source版。AIが生成したPatternはまずprovisional |
| Applies to | 対象Data/Function/Businessと型・Role・単位/精度・Relation/Cardinality・Time・Tenant・Contextの**適用述語**。同名一致だけで選択しない |
| Preconditions / Valid Baseline | 合法Datasetを作る拘束条件と必要Source/Scope、参照Rule。未読込・unacceptedはunknown |
| Variation / Violated Constraint | 正常ケースの変形／狙って壊す制約／保つ制約／多条件相互作用。負例が無関係Ruleまで壊したら診断 |
| Obligation / Independent Oracle | 埋めるべき機能/非機能要求・コード/モデル上の到達候補・独立Expectedの定義域/出典・必要Observer |
| Fixture / Scenario / Shrink | 既存#1131 Dataset・#1180 Scenarioへの提案写像、seed、Generator/Tool版、失敗入力の意味を保つ縮約 |
| Safety / Qualification | 合成データ規則、秘密・holdout非開示、実Target/Host Grant境界、再利用可否・stale |

まずPattern familyは baseline / boundary / missing-null-unloaded / relation-cardinality / bag-join-grain / decimal-unit / temporal-asof / state-transition / role-delegation / partial-read / concurrency-retry / projection-loss / plugin-binding / resilience-latency / disclosure として**候補**を列挙。Patternの命名・最小操作・適用条件は実験で絞る。基準となる正常Dataset→一つの狙った誤条件→必要時のみ交互作用、という認知順序を優先し、総当たりを初期規定にしない。

**作成・選択を迷わせない三段階**：A. 自動提案（Meaning/未達Coverageから上位3件を理由付き表示）、B. 短い選択（boundary等のFocusと小/中/大の予算だけ指定）、C. 定義拡張（専門家が近傍Markdown表を記す、AIが候補化し検証・Owner Review）。AIが既存Patternを作り直すよりも、既存部品の合成とパラメータ化を優先する。適用不能/競合/情報不足なら**なぜ生成できないか**を1つの根拠付き診断にして停止する。

### 12.3 C0/C1はコードの実測値。業務・モデルCoverageを別軸にする

用語は日本の一般的用法に合わせ **C0＝命令（statement）網羅、C1＝分岐（branch）網羅** とする。**C0/C1は実装コードの実際の通過状況**であり、DSL/業務Ruleのケース数、Graph Edge coverage、機能要件のカバー率、合成fixtureでの想定到達率と同一視しない。公式参照：
- IPA ソフトウェアテスト見積りガイドブック（C0/C1/C2）：https://www.ipa.go.jp/archive/publish/qv6pgp0000000yho-att/000005132.pdf
- Go coverage（標準ツールはstatement/basic block近似。C1をそのまま保証しない）：https://go.dev/blog/cover および https://go.dev/doc/build-cover

**Code coverage（実測）**：対象Artifact/Build/Source digest/Target/Tool版/計測対象package/実行Attempt/Instrumentation/実入力/Step/Counterを束縛する。Goの `go test -coverprofile` はC0相当のstatement coverage用として利用する。**Go標準のcoverage率をC1として表示しない**。C1には分岐Outcomeを測る別の計測器・Target Adapter・明示された計測意味が必要。新しいBranch Coverage Providerを導入する場合は、AST/CFGに対する正確さ、短絡評価、switch/select、panic/defer、生成ソース→元DSL位置、Instrumentationの影響を小PoCで確認する。無い場合は `C1=unmeasured` と表示する。

**Coverage Universeを分ける**：

| Universe | 分母（Required） | 実測・主張 | 禁止する取り違え |
| --- | --- | --- | --- |
| **C0** | 固定Buildに含む測定対象の実行可能Statement集合 | 実行で通過したStatement。非Instrumented範囲・外部依存はout-of-scopeと明示 | 行/試験件数をC0とみなす |
| **C1** | 同Buildの計測器が定義する各判定の分岐Outcome集合 | 正確に観測したTrue/False等の実分岐 | Go標準statement計測や単なる `if` 存在から推定 |
| **Model/Rule/Path** | Accepted RuleのDecision/Outcome、Relation path、業務遷移・禁止遷移 | モデル上のCase・Requirement/Scenarioの到達と独立Oracle | モデル分岐をコードC1と同一視 |
| **Functional** | 要求・受入条件・Claim・必要Operation/Target/Observerの集合 | 必要なStepが生成され実行され意味検証済みか | 全コード実行を全機能要件充足とみなす |
| **Non-functional** | 性能/SLO、復旧、セキュリティ、可用性、整合性、互換性、アクセシビリティ等の**測定条件つき要求** | Target/負荷/失敗注入/統計条件/実観測と閾値・期間を対応づける | C0/C1や通常Unit PASSでNFR達成と断定 |

分母は**実Source/Revisionから列挙したRequired集合**。未生成・未計測・到達不能／意図的除外・Source閉包不明を区別し、除外理由・Owner/Review・版を表示する。「未実装の要件」はコード上の分母に現れないため、Requirement/Obligation側で必ず残す。適合状態は「計画済／生成済／実行済／実観測／独立検証済／現在資格あり」を分ける。旧BuildのCoverageを新Buildへ足さない。コード→元DSL→Business Ruleの双方向Source Mapが失われれば関連付けはunresolvedとする。

### 12.4 未カバー領域を「必要なデータとテスト提案」へ逆写像する

Coverage gap→関連Requirement/Accepted Meaning→関係するFunction/Rule/Branch/状態/Source span→Pattern適用述語→境界・負例を含むDataset候補→Driver ScenarioのStep候補→Target instrumented Run/独立Oracleの必要条件、という**read-onlyの推論経路**を設計する。

| Gap | 第一候補Pattern | 追加すべき入力・観測 |
| --- | --- | --- |
| C0未実行Statement | 到達前提の合法Baseline、前提値境界、異常系 | 実行条件・対応Function・実際に通ったCounter |
| C1未実行False/異常分岐 | boundary / missing-null / invalid-transition | 意図的に条件を反転するDataset、分岐Outcome実測、Business Result/Errorの独立Expected |
| Function要件未到達 | operation-scenario / procedure-step | Function→Scenario→CLI/WebのStep対応、手作業・External Stepは別Observer |
| 多重Relation/集計が未検証 | relation-cardinality / bag-join-grain | Path occurrence、2×3 JOIN、独立Line合計、集合閉包 |
| テスト対象の組織/権限未検証 | role-delegation-tenant / grant-revoke | Valid time、Tenant、Subject/Caller/Resource、Host current Grant、拒否Evidence |
| 性能SLO未検証 | latency-under-load | 負荷条件、観測時間窓、統計量p95等、Hardware/Target/Provider版、基準値 |
| 回復/冪等/未知結果未検証 | concurrency-retry / ack-loss | Invocation/Effect/Receipt/独立Resource Read、retry・取消、停止・再開の観測 |
| 開示/秘密境界未検証 | disclosure / alternate-egress | DOM/Tool log/CI/report等の各Egress、機密/公開の許可範囲、観測不能はunknown |

Pattern推薦の順位は **未カバーObligationへの寄与・リスク/影響・独立判別力・実行安全性・生成/実行コスト・前提条件の既知性** の多目的評価とする。単一の“coverage %最大化”で最適化しない。上位案は「満たすGap」「根拠」「必要な追加情報」「期待される観測」「実施できない理由」を示す。C1未達Branchを埋める値を元コードの比較式から推測しても、独立ExpectedやRule受理を省略しない。

### 12.5 実証可能な独立対照 TT-35〜TT-43（新SSOTではない）

| ID | 独立の差分・負例 | Expected（受入の境界） | 既存Owner |
| --- | --- | --- | --- |
| TT-35 最小指示 | 同じFunction/Accepted Revisionに対し長文Promptと1対象指示+Pattern選択を比較 | 同じ固定独立Expected/Mutantsで検出力を落とさず、操作数・Tokenを減らせるか測定。生成AIの自己採点は不可 | #2514/#1180/#2018 |
| TT-36 Pattern適用の誤推奨 | 金額DecimalとQuantity/Time Pointを同型として推奨、RuleがSource未読込 | Unit/Time/Facetが不適合なら候補から除くかunknown。入力の名前のみで自動適用しない | #1705/#2495/#2507 |
| TT-37 目的のある負例 | 数量>=1の**仮Rule**で1と0を比較。他のRelation/Keyは維持 | 負例が狙ったRuleだけ破る。Rule未Acceptedならprovisional、Expectedを確定しない | #2514/#1126 |
| TT-38 C0/C1差分 | 真分岐で全Statementを通過、偽分岐未実行。Go標準profileのみ投入 | C0は高率/100%でもC1は**unmeasured**。別Branch Providerで偽分岐未通過が分かればC1 gap。数値捏造しない | #1180/#2514 |
| TT-39 要件分母の欠落 | Source上に未実装Business Requirement、実Codeに該当行なし | C0 100%でもFunctional Requirementはuncovered。テスト対象0件にはしない | #1119/#1180/#2514 |
| TT-40 NFRの閾値 | p95 200ms以下・指定負荷/時間窓の**仮SLO**を設置、Unit PASSのみ | 未負荷/未計測ならunverified、合格は同Target/負荷/集計条件の実測に限る | #1163/#2514/#1445 |
| TT-41 多段表現の非単射 | Graphの二Relation occurrence→Treeの一表示→Markdown | 表示PASSでも逆Patch対象の一意性は不成立。Mapping損失を下流へ伝える | #3236 CX/#2497 |
| TT-42 Plugin実Loadの不一致 | Contract C/Provider Bを選択したがConsumerはArtifact AをLoad | 静的Resolve成功と実Consumer合致を分離し、合成Conformanceを不適合/unknownにする | #951/#943/#2600 |
| TT-43 Generator/Pattern改版 | 同seed/ScenarioでPattern規則、生成器版またはRule meaningを変更 | 旧Data/Caseを現在のExpectedに無条件流用しない。無関係依存は過剰失効を防ぐ | #2582/#2514/#1180 |

上記IDはTT-01〜29や#3236 CX、#951 AV、#2515 H6等の具体データ/Expectedを**置換しない横断Obligation**。実行Ownerに既存Caseを参照して到達・観測させる。すべて**計画、NOT RUN、NOT ACCEPTED**。

### 12.6 実装順序と完了条件（既存基盤への増分接続）

- **P0-A Coverage現状の正直な計測**：実Go `go test -coverprofile` / `GOCOVERDIR` で固定Target/Commit/Instrumented範囲のC0相当値を報告。C1 Providerがなければunmeasuredを表示。Rule/Requirement未カバーと別列にする。対象数・分母・例外・Source→Build→Execution同一性は必須。
- **P0-B 1対象→推薦→Dataset**：#2586のH/L1/L2/D2を対象に、固定Pattern3〜5件の適用述語を比較し、基準・JOIN反例・部分取得のDataset候補を**同一Accepted Revisionから**構成。本人への追加質問なしで推薦理由と不足Sourceを示し、Datasetから独立Expectedを機械的に逆算しない。
- **P0-C Gap→Case→Scenario→実到達**：要求未カバーとC0未通過、C1未測定を**別Gap**として試し、上位推薦が既存Driver ScenarioのどのStep/Case候補を満たすか対応づける。Owner Review後、CLI生成・実行Counterと独立Observationへ追跡。正式C1計測器の採否はP1へ保留。
- **P1**：Go C1等のInstrumentation/正確なSource Mapの小PoC、Model/Rule branch coverage、Domain Patternの人/AI両用Authoring、Rule競合診断、意味を保持するShrinking。性能/可用性/セキュリティ/アクセシビリティ等の異質NFRへの推薦を追加し、実Target測定を各既存Ownerへ委譲。
- **P2**：全Target/Plugin/ProviderのCoverage/evidence現在資格の再計算、変更Impact、推薦順位改善。生成AIへの長文Promptと短い指示のA/B比較を同じReference Corpus/独立Expected/予算で評価。

**実測KPI（初期仮説。合否閾値はPoC後Owner判断）**：必要指示数、Pattern選択にかかる操作数、推薦精度（適用できた割合）、未達Obligationの減少、独立Mutant検出率、正常Baselineの意味保持、単一違反生成の正確さ、再現率、生成・実行コスト、stale誤認数、秘匿/無許可副作用0。C0/C1の数値のみを成果指標にしない。

**停止条件**：Goのstatement coverageをC1と呼ぶ、Source未Loaded/未計測を0%・達成済みに丸める、モデル上の未実装要件を分母から消す、コード通過からFunction/NFR完成と推測する、未受理RuleをAIが補って確定する、生成AIが自身のWantを採点する、Patternの提案を現在Grantに昇格する、実Provider/外部通信へ安全範囲外のテストデータを自動送出する。

**現状**：本節は既存TEI Issue/PRとIPA・Go公式資料に基づく設計追補。CLI構文・Pattern Library/Schema・C1計測・NFR Provider・生成AI最短操作は提案であり、実装/実Go試験/独立Oracle検証/Owner正式受理は**NOT RUN / NOT ACCEPTED**。

## 13. 経路合成との接続（#3236を受入条件として参照）

[統合Draft #3236](https://github.com/hat47x/tei-platform/pull/3236)のData→View→Consumer、Representation Instance、非単射・Loss/Presence・否定依存・Disclosureの独立Expectedを**新たなテスト正本として複製しない**。Patternは各変換で維持すべきIdentity/Time/Unit/Multiplicity/Scopeと、合成後に失われた/unknownとなる性質をObligationへ投影する。例えば二つのRelation occurrenceがTree一行へ畳まれれば、読取り表示テストは提案できても自動逆Patchテストを「成功見込」とせず、拒否/unknownをExpectedとする。

Plugin Compositionも既存Graph/Relation/Contract表現と#951の解決・#943/#2600の実Artifact・#1174の現在Grantを参照し、解決候補の存在と**実ConsumerがどれをLoadし実行したか**を分ける。新Graph Schema/第二Resolver/第二Authority/第二Test Scenarioは作らない。局所PatternのPASSから合成Target全体のPASSを推論しない。TT-41/42の専門Ownerが提供する実Resultで欠落の有無を判断し、実験未実施の部分をunknownとして残す。

<!-- tei-design-context-query-pretest-20261107 -->
## 14. テスト生成より前のP-1：Design Context QueryとCLI/MCP/Skill共通取得（#3262／設計）

### 14.1 最初に解決すべき対象情報の不足

テストパターン推薦の入力には、受理済みSource/Meaning、Function Data Context、Data/Function/Business、Unit/Time/Relation/Rule/Procedure/Organization、既存Scenario/Fixture、RequirementとC0/C1/NFRの未カバー集合が要る。しかし**既存のread-only CLI呼出を足しただけでは、必要な情報のすべてが実際にLoader/Parserへ到達し、同じ版で解釈された証拠にならない**。#2822空Workspaceとbuiltinの区別、#2823 managed inventoryと実Loaderの差、#2506部分取得、#2514独立Required集合の欠落を先に埋める。

実装済みの現行入口として、mainの cmd/tei/cli_contract.go に tei.cli.discovery/v0 selected-read-only-commands：model catalog、workspace-files、inspect、trace、source-index、requirements、validate、knowledge-context、plugin-lock/resolve等がある。ただし**全Capabilityの列挙ではない**。worldはraw JSON、version -jsonはidentity形、commandごとにmode・schemaVersionが異なり得る。view -execute等は大分類read-onlyの説明だけでargvの実File/Provider効果を保証しない。#1155を再実装せず、#1158の型付きRequest/Result境界の再利用可能性を調べる。

### 14.2 同じQuery、用途別の表現

新製品独自の知識正本を追加せず、**Design Context Query（仮称）**というread-only取得・合成の契約を検討する。入力＝Workspace/Subject Kind+Stable ID/Required Domain View/目的(test design等)/期待Source・Model版/Relation・依存閉包/要求Scope+as-of/Tenant/Consumer・開示資格/表示budget。出力＝一枚の概要、typed ref index、完全性・未達・影響、詳細へ辿れる同版参照。

- **Machine JSON**：Stable Subject/Relation occurrence、Source span、Model/Source/Loader/Parser/Accepted/Scenario/Target版、result kind、scope、grain、unit、時間、ProvisionalとAccepted、Known/Unknown、診断を失わない。読取り専用、失敗時も構造化して明示。
- **Human Markdown/表/図**：同じ意味の投影を「概要→関係→診断→詳細」の認知順に。省略した項目は明示し、詳細refへ戻れること。別の意味正本を文章Renderer内に作らない。
- **Harness Product Skill**：存在するCLIを発見してから手順を構成し、受領したResultの完全性を点検して不確実なClaimを拒否。Skillの説明・実Toolロード・実Source読込・実Tool返信・Owner権限を独立扱い。TEI単体実行にSkill/TAI/MCPを必須化しない。
- **MCP**：同じ型付きread-only Queryの将来Adapter。Tool/Resourceへの対応、pagination、Resource scope、Result errorsは実CLI一往復の成功後に選定。新しい第二Reader/Host/Authority/Loaderを組み込まない。

### 14.3 「漏れなく」の明確な意味と転送完了

Design情報の完全性は**指定したRequirement/Subject/Relation/Rule/Procedure・外部依存集合とScopeに相対的**。次を別々に観測する：

1. **Required Closure**：独立に要求されたData/Function/Business、非出力Rule列、Relation path、Function Context、Manual Step、Source Import、Plugin Provider、Scenario/Requirementの集合。requiredを発見済みCase数から逆算しない。
2. **Source Closure**：Manifest宣言、Discovery、実Loader read、実Parser occurrence、Accepted Binding。欠けた場合はnot-loaded/unresolved。builtinだけのvalidate.okを利用者Domain完成とはしない。
3. **Semantic/Observation Closure**：Requested Claimに必要な粒度、NULL・未取得・非開示、Query/ContextのResult Slice、実Provider/Hostは設計Sourceと分離。静的バインドを実Effectと誤認しない。
4. **Delivery Closure**：要求したRef/Case/Sourceが**Toolの受信者まで完全に到達**したか。ページ数、連番、受信byte、限定公開範囲のdigest、terminal marker、next cursorと呼出結果IDを検証。DSH #2509ではH3実Skill Readが観測済みだが、model catalogの8,192byte切断、56 Request/55 Reply、stderr混入、比較適格0/4が残る。旧「H3未ロード」ではなく、不完全なDeliveryを未確定として扱う。

大量応答の既定は**短いSummary＋Typed index＋追加ページ**。安定順・Page Budget（実Harness観測で決める）・同版Snapshot固定・明示continuationを要求する。Page間にSource/Grant/Disclosureが変わればstaleとし混ぜない。外部LLM/ログへのread-only送出も現在のDisclosure/実OS強制を要する。非公開情報の件数、digest、ID、存在・診断内容を未許可相手へ漏らさない（#2515/#1178）。

### 14.4 独立受入 DCQ-01〜10と最短の縦切り

**DCQ-01〜10の正負対照・Owner詳細は#3262を唯一のIssue正本とする。** ここでは同じD1 2行/D2 3行の合成FixtureにRequired Subject/Scenarioを加え、1 Subjectで次の順に受入：

- P-1a：実Binaryのcommands/version→workspace-files/Source Reader→model catalog/inspect/trace/requirements/validateを利用して独立Closure比較票を作成。Sourceの実Loader/Parser到達を捉えられない場合はunknown。
- P-1b：同じデータをMachine/人間向けに別投影し、欠けたData・Rule・Function Procedure・RequirementのRef/Unknownを隠さない。0/未取得/非開示とDescriptionだけの差を識別。
- P-1c：既存Product Skillを使う最小指示で同じ設計Contextを取得。H2/H3は#2509/TAI #168で比較、Skill有無の正答率とTool Call・Token・漏れ・切断誤認を別計測。
- P-1d：必要性が実測できた後、型付き同一QueryのMCP AdapterでCLIとの意味同値を評価。
- **その次がテスト生成P0**：Qualified ContextだけをTT-35〜43／Pattern推薦／#1180 Driver Scenario採用へ流す。Context未取得を空Fixtureや全件完了にしてはいけない。

**停止条件**：JSON parse／exit0／validate.ok／Catalog非0／1ページ／Tool successを全情報の証拠にする、未解析Source・hidden Relation・非出力Rule・Manual Procedure・未カバーNFRを消す、別版ページを連結、現Grantなしで機密Raw/件数を外部送出、未実装のMCPを実在するToolとしてSkillに記述する、という動作。

**状態**：本節は#3262の受入要件をテスト生成計画の前段として接続した**設計のみ**。実CLI一貫Read・Design Context Query・MCP・Harness全受領・新Test Pattern E2EはNOT RUN、正式なSchema/Owner受理はNOT ACCEPTED。

<!-- tei-test-t0-t1-integrated-dcq-20261107 -->
## 15. 2026-11-07 更新：T0実証からDesign Context P-1b→AI新規テストT1への一本化

### 15.1 最新の限定実証と未達を改版する

前節までの「新実行NOT RUN」は**本設計文書で追加した機能に対する記載**であり、既存TEI Test Ability T0全体が未実行という意味ではない。実証Ownerのmain統合PR #3264/#3266/#3268と、main push [CI #42359610226](https://github.com/hat47x/tei-platform/actions/runs/42359610226)を現時点の基線とする。

| 検証 | 実際のEvidence | 言える上限 |
| --- | --- | --- |
| Design Context P-1a限定 | 同一mainの実TEI binaryからread-only CLI 9種類、Required Source readByLoader、対象欠落/版差/Loader未読負例を照合（#3264） | 一つの公開合成SubjectのSource/Indexと限定Read。Function/Business/Ruleの全閉包とは言えない |
| Test Ability T0 | 既存Driver Scenarioの生成済みpinned 6 Stepを実CLIで実行、実CLI否定4件＋独立義務否定2件（#3266/#3268） | 既存Scenarioの実到達と既知欠陥への局所判別力。AIの新しいCase生成や製品全域の実証ではない |
| Code C0 | Go cmd/teiの**選択1テスト**のstatement C0相当10.1% | 対象package/テスト選択の局所的実測。全体C0、業務要件のCoverage、C1を示さない |
| C1・機能/NFR | C1 unmeasured、要件分母の全件実照合・非機能実測なし | 表・グラフの分岐、業務Rule分岐、HTTP Decision GraphのEdge/GuardもコードC1の代替にならない |
| Harness H2/H3 | #2509でH3 Skill/参照Tool Readは実測。ただしCatalog 8,192byte切断・56 Call/55 Result、比較適格0/4 | Skill実Loadと実CLI受信は限定事実。完全配送・独立採点・実OS隔離は未成立 |

上記のmain push RunのArtifactには公開合成Evidenceがあるが、保持期間7日のため永続ファイルが常に入手できるとみなさない。Run/CI/Source/Build/Scenario digestの資格を固定して参照し、後のHEADの新規テスト成功へ転用しない。

### 15.2 P-1b Design Context Packet（DCP、仮称）の利用者中心の出力契約

**「設計情報をすべて返す」という絶対保証は採らない**。利用者がFunction・Issue・Ruleのどれを指定したか、用途がテスト生成/仕様確認/影響分析のどれか、依存深さ・Sourceの範囲・時点とGrantを決め、**用途相対のRequired集合**をまず外部に根拠付きで確定する。

1. **Overview（常に小さく）**：Subject/Role、何を得たか、何が足りないか、受理された意味の範囲、現在の資格、必要な次の読取り。読む側が長大なJSONを毎回すべて展開しなくてよい。
2. **Typed Reference Index**：Data/Function/Business/Rule/Procedure/Query/Context/Scenario/Requirement/Provider/Artifact等の安定Ref、Source locationと版、Relation occurrence、依存の正負、必要な非出力属性、NFR/実C0/C1の観測Refs。これは閲覧や関係探索の**索引**であり、Role/意味やSource本文の完全読込に昇格しない。
3. **Closure Ledger**：Required、Discovered、Loader-consumed、Parsed occurrence、Accepted/Provisional Binding、Consumer-visible、Delivered/Tool-ACKを**集合として照合**。各差集合をmissing/not-loaded/unparsed/unresolved/permission-denied/unsupported/stale/truncatedで説明。分母が未確定なら割合を算出しない。
4. **Detail Pages**：必要Refを段階的に詳細取得し、Stable order・Snapshot/World/Index/Meaning/Disclosure binding・Continuation・End marker・Received bytesを検査。同じSource、Grant、Scopeを保証できない複数ページを継ぎ足さない。
5. **Claim Ceiling**：得られた証拠から許される最強の主張をscope付きで表示。例：syntax valid、indexed source exists、loaded/parsed、test candidate-ready、scenario-executed、observed-effect、business-acceptedを**別条件**とし、局所成功から上位へ自動昇格しない。

**宣言する完全性は直交する**：Required/Source/Meaning/Dependent Result Slice/Consumer Disclosure/Delivery。非公開情報の存在や件数はDisclosureが許す場合だけ示す。全世界の完全性は、許可された局所完全性とは異なる。Reasonがpermission-deniedでも秘密のIdentifier/件数を露出しない。

Responseを読むProduct Skillは、JSONのschemaVersion/modeとstdout JSON、stderr診断、Tool Result受領を別々に検証する。Tool切断なら「次ページあり」だけでなく「このページ自体が完全に届いたか」を検証できなければ停止する。CLI JSON・MCP Tool/Resource・人間向け表/Graphは**同一Queryからの用途別投影**であり、Human表示の省略を新しい意味正本にしない。

### 15.3 一つのT1独立試験へ接続する

**最初の対象を公開合成Workspaceの受注行番号Rule sales.order_line.line_number_positive に固定する**（[#3273](https://github.com/hat47x/tei-platform/pull/3273) §13、TG-01〜06が実行Owner）。既存T0の6 Stepを単にもう一度PASSさせることはT1としない。

| Stage | 必須受渡し | Fail-closed条件 |
| --- | --- | --- |
| T1-0 Required | Issue/Ruleの原典・独立な未カバーObligation、合法Baseline、Allowed Source、許可されるDisclosure | AI出力Case数をRequired分母としない。Rule Source未解析ならunknown |
| T1-1 Discover | 実CLIでSource/Loader/Role/Rule/Operator/World+Index fingerprintの適用範囲を観測し、Design Context Packetへ束縛 | Source Indexedだけ、Builtinのみ、Role未解析をRule資格にしない |
| T1-2 Recommend | AIには対象Rule・合法Baseline・許可ContextとCoverage gapのみ開示。出典つきPattern候補から**行番号0**の負例を提案 | Independent Expected/Mutant/holdoutをAI入力へ渡さない。数値を意味未受理のRuleから発明しない |
| T1-3 Prepare | 量・Key・Relation等の合法性を保持した一差分Caseを固定、seed/Generator/Pattern/Source/Scenario revisionを明示 | 行番号0以外の制約も壊すなら違反識別が曖昧として拒否 |
| T1-4 Execute | #1180の既存Driver Scenario SSOTへreview candidateを接続し、verify→generate→check→新pinned CLI Stepを**実行** | 生成だけ・exit0・旧Scenarioの再実行をT1成果にしない |
| T1-5 Qualify | AIと独立に事前固定：quantity_positive=satisfied、line_number_positive=not-satisfied、line_number_unique=satisfied。RuleごとのBusiness Result・独立観測・Source/Run/Toolの同一性を比較 | 値が一致しない/必要Obs未取得/Tool Reply切断ならfailまたはunqualified。OracleをActualから修正する場合は新Revと独立再審査 |
| T1-6 Review | Required Gapを1つ埋めたと根拠付きで報告。Ownerは受理を判断、未受理ならCandidate状態を維持 | AIの案/限定CLI成功をMeaning Acceptanceや全Coverageへ拡張しない |

T1への合流で**コードC0/C1の数値改善を必須成功条件にしない**。対応するコード分岐のInstrumentationが未整備ならC1はunmeasuredとして、モデルRule Branch/義務/独立Expectedによる判別の成立を別に記録。必要に応じて局所C0の変化を副次観測する。性能/可用性/機密開示等のNFRは独立Probe/Threshold/Target・同条件測定が揃うまでunverified。

### 15.4 複合構造の検証をT1の後続Patternに引き渡す

| 成熟した別レーン | Test Patternの選択規則 | 資格上限・Owner |
| --- | --- | --- |
| HTTP Response Decision Graph #3274（main設計#3275） | Guard/Node/Edge/Transport/Client Reactionを別Universeにして、422不適合／ACK-loss／再送不能を候補に。Node到達は業務Outcomeと異なる | HDG-01〜12を参照しExpectedをコピーしない。実HTTP/外部HostはNOT RUN |
| Verified Composition BOM #3267 | Resolved exact tuple 2 Provider版×2 Consumer版、偽互換・actual Loadとselected mismatchをPattern候補に。Pairwiseは未知3-wayをPASSにしない | static matrix/locked candidate ≠ real invoked/verified BOM。Owner #3267/#951/#2514 |
| Temporal/Derived Trait #3027・標準Data | as-of遡及訂正、Temporal履歴保護、Derived再生成/失効・Materialization削除を区別。Trait Hintを機能実装やRetention保証としない | Semantic Trait/Owner受理と実Provider/ストレージ実観測は別 |
| 汎用Data Source Workbench #3270（main設計#3276） | 実Provider ReadのScope/Row/Field/Grant→Function Data ContextをP0第一候補とし、DDL AppliedUnknownは独立再読取まで未確定 | Schema CRUD提案≠実適用、DBA運用は標準範囲外、#2586/#2892がOwner |
| Graph/複数表現 #3236・#2497 | 同Node別Path occurrence/多重Edge、Tree/Chart/MarkdownのLoss・逆Patch不適格、Hidden relationのDisclosure | View同値≠意味可逆・更新権限、独立CX/MP Caseを参照 |
| Self-Build #2831/#1119 | Required Source/Build Graph/Plugin/Toolchain→実消費Byte・Artifact・Execution Receiptの一致、欠けたBuild依存の負例 | Build成功≠Source closure/Stage収束/実Consumer現行資格 |

**優先順**は「多Domainを一度に実装する」ではなく、T0成功の実証を保持し、#3262 P-1bでT1-0/1の必要Contextを提供→#3273 T1-2〜6で一つの新しいCaseを検証→#2586でProvider Readと意味Contextを一辺接続→専門Plugin Patternを追加する。Graph/BOM/HTTP/Temporalの設計ケース群は並列で資料化してよいが、P0一往復の依存にはしない。

### 15.5 検証結果を自らテストする（Quality of Test Evidence）

- **Denominator honesty**：Required Function/Rule/NFR/C0/C1/Plugin tupleをそれぞれ独立Requiredとして列挙。未実装のFeatureを「コード対象外」だから全Requirement完了としない。
- **Generation independence**：AIに渡した設計Context、保持したHoldout、事前固定Expected、検証に使ったTool・Consumerを別Witnessにする。自己採点を通さない。
- **Completeness under delivery failure**：8,192byte切断、欠落1 Tool Reply、旧Source/別Grantの混入、非出力Rule、手作業などを同じ合成Worldで別々に注入。正例・負例・unknownを曖昧に一括PASSしない。
- **CI trigger quality**：#3268の広いpaths起動は実装済みの安全側基線であり、完全なDependency Graph/選択的再実行の証明ではない。漏れと過剰CI実行・費用を並行測定する。
- **Minimal instruction quality**：同一Source/Fixture/独立Expectedの長文Promptと「対象Rule/未カバーを埋めて」という短い指示を、提案の正確性・Tool Call/Token・不必要Source閲覧・Disclosure/欠損誤認で比較。H2/H3の既存実験資格を汚染しない別Runにする。

**横断の停止線**：新CLI/MCP名の実在偽装、Design Source未解析なのにcompleteを宣言、古いT0のPASSをT1へ引継ぐ、C0=10.1%を製品Coverageと呼ぶ、C1やNFRの推定合格、未受理Trait/Ruleを業務Meaningへ自動昇格、実Plugin/Provider/Hostを未実行のままqualifiedとする行為。

**実行状態**：本節の追記は、実mainのT0 CIと新しい他レーン設計を参照してT1への接続を**計画化したもの**。上表のT0は当該exact mainで限定実証済み。P-1b統合Query、T1新生成・実Pinned Rule結果、C1、MCP、実DB/Host/HTTP、Owner正式受理は**未実施／未受理**。既存#3273のT1実行Owner、#3262 Design Context Owner、#1180 Scenario SSOT、#2514独立Expectedを重複させない。


---

<!-- tei-testing-crosslane-req-id-matrix-mbt-20261107 -->
## 16. 2026-11-07 横断成熟差分：要求同一性、T1新規判別力、Version Matrix（設計と観測を分離）

本節は既存の§15に追加する**統合時の停止条件と受入票**。T1の実装・テストOwnerは[#3273](https://github.com/hat47x/tei-platform/pull/3273)、独立Expectedは#2514、Design Contextは#3262、BOMは#3267/#3269。新しいRunner、Test SSOT、Requirement Registry、Authorityを作らない。

### 16.1 実際に発見した製品要求IDの意味衝突：統合Blocker

2026-11-07にGitHubのファイルを直接比較した結果、mainの requirements/TEI_PRODUCT_REQUIREMENTS.md （blob 4893dac4a6bf31b24c8b510b68ce43ae408ddfd3）では **RQ-035 = 異種Data Source Workbench・開発者向けSchema CRUD**。mainへ統合済み[#3276](https://github.com/hat47x/tei-platform/pull/3276)の要求である。

しかしDraft [#3269](https://github.com/hat47x/tei-platform/pull/3269) のHEAD 895bdd6143ce2c9050820379662e7bdcedee5bfa の同ファイル（blob 3a675f181f7da1a03e8d06deb022cf4235e6b913）では **RQ-035 = Verified Composition BOM・複数Version組合せ試験**。異なる意味の二要求に同一IDが割り当てられている。GitHubの[#3269レビューコメント](https://github.com/hat47x/tei-platform/pull/3269#issuecomment-6706614977)へ是正を依頼済み。

これは「現在1ファイルに重複IDがない」検査を通過し得る**ID→Meaning置換**であり、単独BranchのGoテスト合格とは独立した統合阻害条件となる。**現main RQ-035の意味を維持**し、BOM側を独立Requirement（未使用のRQ-036等はOwner採番）として改訂し、Issue/Plan/Case等の参照を照合するまで統合不可。自動採番や他Ownerの要求本文上書きはしない。

最小回帰は baseline と candidate の **ID→Title/Meaning/Source** を比較して、明示Owner判断やMigration Mapのない別意味への置換を拒否する。異なるBranchの局所PASSをRequirementの同一性と誤認しない。

### 16.2 実main T0と未完了T1の分離

- **T0は限定成立**：main統合済み[#3264](https://github.com/hat47x/tei-platform/pull/3264)/[#3266](https://github.com/hat47x/tei-platform/pull/3266)/[#3268](https://github.com/hat47x/tei-platform/pull/3268)。固定mainの[CI #42359610226](https://github.com/hat47x/tei-platform/actions/runs/42359610226)で、実read-only CLI 9種・Product Skill静的14件・既存pinned Scenario 6 Step・実CLI負例4件/独立義務負例2件を観測。Go C0相当10.1%はcmd/teiの**選択された局所テスト**、C1はunmeasured。広いCI path triggerの存在は完全なDependency closureを証明しない。
- **T1は未成立**：[#3273](https://github.com/hat47x/tei-platform/pull/3273)の独立Gapから、新しい行番号0の負例をAIが提案し、独立Expectedと照合し、受理待ちのScenario Candidateとして実pinned CLIへ流す一件が最小課題。既存6 Stepや旧重複Keyケースを再実行しただけではT1としない。AIがActualに合わせてExpectedを修正した場合は独立性を再審査。
- **実PG Readは限定成立**：[#3238](https://github.com/hat47x/tei-platform/pull/3238)の隔離PostgreSQLでTenant A {3,9} と TOP1 {3} を実取得し、Q05〜07と非出力ordered_at WHERE正例を検証。ただしFunction→Query→実Provider→Claimを同一製品Attemptで完全に接続した証拠ではない。非出力PredicateのSQL評価と返却行集合のDisclosure資格は別判定。Q07の別status述語を本正例の成功で上書きしない。
- **Oracleの改訂履歴**：#3266初回CIはtyped Observationとdecode後のpendingの取り違えでFAIL。期待値修正後の成功は別HEAD・別Runであり、修正の根拠を独立Oracle/Output Contractから照合する。現在値への追従だけでQualifiedにしない。

### 16.3 Verified Composition BOMのPairwiseをテスト設計の分母へ接続

[#3269](https://github.com/hat47x/tei-platform/pull/3269)の隔離Go Referenceでは3軸各2値の全直積8通りに対し、供給されたexact Lock候補は4行であり、供給候補に現れる2項組合せが9/12、計画はその9/9を被覆する。**供給候補の9/9** と **宣言Universeの9/12** と **実行済み・適合済み** は別の集合である。供給されない3ペアを不可能・互換性なし・PASSへ自動変換しない。3-way以上の相互作用も未証明。

BOM-AとBOM-Bの個別合格から、実際に組み合わせたCompositeのQualified Unionを導かない。実Resolver Lock、Source/Build/Artifact/Consumerが実Loadした版、Scenario/Independent Expected、Target/Environment/Grantの一致を**同じ組合せ行・Attempt**で証明する。現在のPairwise Plannerはplanning-only・qualified:falseを維持し、Core Resolver/Hostの代わりにしない。

### 16.4 テストモデルの忠実度とonline/offlineを受入に明示

[ISO/IEC/IEEE FDIS 29119-8:2026](https://standards.iteh.ai/catalog/standards/iso/45b3e3a2-a490-40d3-8348-4357687bf137/iso-iec-ieee-29119-8)は**Final Draft**。正式規格の全面準拠を主張するのではなく、model fidelity（テストモデルと実対象の前提/入力/期待動作の対応）、抽象水準・追跡可能性・test-data・test環境・モデルCoverage、online/offline生成の区別を設計チェック項目として参照する。

TEIの既定は**候補生成→Owner Review→保存済みDriver Scenario→Target別実行のoffline型**。online探索は隔離合成Fixtureの未受理Candidateとして扱い、実Host副作用・業務Meaningへ自動昇格させない。モデル上に手作業、非出力依存、Source読込失敗、非機能要求が表現されていない場合、その未表現部分をmodel-fidelity gapとして保持する。

### 16.5 独立正負対照 TT-44〜TT-50（新しいCase正本ではなく横断Obligation）

| ID | 比較入力 | 独立Expected・判別する誤り | Owner |
| --- | --- | --- | --- |
| TT-44 Requirement identity | main RQ-035 WorkbenchをDraft RQ-035 BOMで置換。最終文書内の重複数は0 | **FAIL**：同IDで別Meaning。Owner採番/参照修復なしに統合不可。GitHubで既存衝突は**観測済み** | #3267/#3270/#1119 |
| TT-45 T0とT1 | 既存6 StepはPASS、未カバー行番号0の新Caseはなし | T0限定PASS、T1はNOT RUN。新Caseの実pinned実行と独立Rule診断が必要 | #3273/#1180/#2514 |
| TT-46 Oracle改訂 | 初回Expectedがtyped状態とdecode後pendingを混同し失敗、修正後PASS | Oracle出典と二HEAD/Runを比較、Actual追従だけでは独立検証基準としない | #3266/#2514 |
| TT-47 CI起動 | Required Source/Scenarioを変更したのにWorkflowが未起動/SKIP | Gate未実施として不適格。#3268のpath監視も実Runで発火と漏れを測る | #3268/#1119 |
| TT-48 非出力Predicate | 実WHEREで非出力ordered_atを使う正例と、statusが未評価のQ07 | SQL実評価の存在、Membership開示、Claim資格を別々に照合。別Predicateの過大受理を禁止 | #3238/#2506/#2515 |
| TT-49 Pairwise分母 | 4 exact Lockの供給候補9/9、宣言Universeは9/12 | unknown 3/12、qualified:false。未供給を不適合・合格としない | #3269/#3267 |
| TT-50 複数BOM合格の合算 | BOM-AとBOM-Bが別々にPASS、合成a+bは未構築/未実行 | CompositeはNOT RUN/unknown。実組合せのLock/Artifact/Consumer/Independent Expectedが必要 | #3267/#951/#1163 |

TT-44は**既存Sourceの意味衝突という現在確認済みの反例**。TT-45〜50の横断受入E2Eは今回未実施。#3273 TG、#3269 Pairwise Fixture、#3238 PG fixtureを複製しない。

### 16.6 既存計画への優先順位の変更

1. **P0-G Integration guard**：RQ-IDとMeaning非衝突を#3269で解消し、既存requirementsとCIに最小回帰を追加（新Registry不要）。これをT1やBOM製品完成と混同しない。
2. **P0-T1**：#3262のP-1bにSource/Meaning/Rule/Gapの閉包を要求し、#3273の新しい行番号0負例をDriver ScenarioへReview→実生成/実CLI→独立判別。実証済みT0は固定した比較基線とする。
3. **P1-Matrix**：既存Resolver/Lock→Build/Artifact→Test/Independent Expected→Consumer実Load/Host観測をMatrix行ごとに接続。実行できていない行を勝手にPASSや非対応としない。
4. **P1-CI/C0/C1**：#3268の保守的Triggerを実Changeで検証。C0局所・C1未測定・Functional/NFR未カバーを別Universeで表示。
5. **P2-Harness/Disclosure**：#2509/TAI #168の比較適格が成立するまではSkill説明・Tool完了を実完全受領/現在Grant/漏出不在へ昇格しない。

**今回の状態**：GitHub実Source比較（TT-44）、既存PRとRunの限定結果に基づく計画追補、Blob再取得のみ。新しいCLI/Go/PG/AI T1/OSテストは**NOT RUN**。正式採番・Owner Meaning Acceptance・BOM qualificationは**NOT ACCEPTED**。


---

<!-- tei-testing-p1a-readiness-actual-20261107 -->
## 17. 2026-11-07 P-1a実装済みReadinessの再評価とT1入口の厳密化

**更新理由**：§15/16の作成後に、Design Contextの実テスト[#3284](https://github.com/hat47x/tei-platform/pull/3284)が**mainへ統合**され、BOM Referenceの[#3269](https://github.com/hat47x/tei-platform/pull/3269)は**未mergeでclosed**となった。したがって§16の「Draft #3269をそのまま統合するBlocker」という説明は当時の競合発見の記録として保持するが、**現時点の統合予定PRであるとの前提は撤回**する。RQ-035の異Meaning再利用は新しいBOM採用Branchで再検査する。過去のGitHub Blob比較による衝突発見自体は消さない。

### 17.1 #3284の実行証拠と現在の資格

#3284の最終HEAD `eeb2295a900df45e134b65d3f96e677e1f0c886f` の[専用CI #42370519406](https://github.com/hat47x/tei-platform/actions/runs/42370519406)は **completed/success**。PR仮統合Checkout `9fc3a34462b18ab9f79d8459bbefb6bfebee0dd0`、Go単一TEI binary SHA-256 `bf6720f51bec404396b779a2ba74efecc944b1d333bccd2982565e1f05fc1e24`。9件のread-only CLI、14件のSkill静的検査、Source/Subject/Fingerprintの限定Contextと**7件の欠落・陳腐化・未観測負例**、既存pinned Scenario 6 Stepを同じCIで検証した。公開合成Artifactには `context-readiness.json` が追加された。

重要なのは、CIが成功しても出力は **`partial` / `qualifiedTestGeneration=false`** であり、外部Harness受領は`null`のままという点。これは「テストが失敗した」のではなく、**不足を正しく検出するTest Readiness Checkerが合格した**状態である。未取得Data/Function/Business、受理済みMeaning、実Provider Read、外部Tool Delivery、C1/NFRは資格外。Artifactの保持期間と現在mainの再実行を別扱いにする。

### 17.2 P-1bの入力を「AIに渡してよい」ものへ制限

P-1aが`partial`を返した場合、T1用Context Packetは次の**既知の部分集合**だけを機械可読で渡す。未知のRequirementやSourceが「存在しない」とは推測しない。

| 入力の分類 | 生成器への扱い | 合否 |
| --- | --- | --- |
| Loader実消費・Parser位置・同版SourceとAccepted Meaningが照合済み | 対象Scope/Grantが許す範囲でPattern候補の適用に使用 | candidate-ready（Target/Independent Expectedは別） |
| Source indexedだがLoader未読込、Meaning未受理、Function/Business閉包不明 | Source/Meaningの不足として報告。合法Baselineを仮定してExpectedを確定しない | blocked / provisional |
| Required分母が未確定、C1未計測、NFR Observerなし | `unknown/unmeasured`を維持し、Coverage改善率を数値化しない | not-qualified |
| 旧RevisionのPage、Tool Reply欠落、Disclosure未許可 | 同版Packetへ混ぜず受領/開示の失敗として停止 | fail-closed |

**局所的な候補生成は可能**：`qualifiedTestGeneration=false`は全てのアイデア出しの禁止ではない。たとえば公開合成Ruleの出典が明示されれば、行番号0を「未受理の負例Candidate」として提示できる。ただし**独立Expectedの正式採用、実Target実行、現在Qualified**とは別。ここを一つのbooleanで表現するとAIの提案価値と実証資格を両方失う。

### 17.3 テスト提案の最小の次段階（#3262→#3273→#1180）

1. **P-1b**：既存`context-readiness.json`のRequired/Observed/UnknownとSource Refを同版で受領し、Rule `sales.order_line.line_number_positive`に必要なSource/Rule/Allowed Contextを一つのtyped indexへ投影。新CLIを作る前に既存read-only command群で閉包が取得可能か確認する。
2. **T1新規性**：既存Scenarioにない`line_number=0`の負例を生成。合法Baselineの他Rule/Relationを維持し、狙ったRuleだけが`not-satisfied`となる独立ExpectedをAIへ秘匿して固定。CandidateはOwner Reviewまで未受理。
3. **実行・判別**：既存Driver Scenario `verify→generate→check`から新pinned CLIを実行し、実Tool Reply・Input/Source/Build digest・結果の観測を同Attemptへ束縛。既存6 Stepが再PASSしただけではT1を完了扱いしない。
4. **差集合を提示**：`required`、`candidate`、`reviewed`、`executed`、`observed`、`independently-checked`、`currently-qualified`を別集合で比較。C0は測定した範囲だけ、C1はunmeasuredのまま。
5. **Plugin/Trait/BOMとの連結は後続**：main統合済み[#3277](https://github.com/hat47x/tei-platform/pull/3277)/[#3281](https://github.com/hat47x/tei-platform/pull/3281)/[#3285](https://github.com/hat47x/tei-platform/pull/3285)のPlugin-first/原子的Traitは**設計成果**。実CLI/Studioの同Provider Load・Trait Fulfillment・合成Conformanceを既存#3278/#1251のOwnerが実証してからMatrix行へ投入。docs-onlyの成功を実Plugin Runtime PASSにしない。

### 17.4 追加の負例 TT-51〜TT-55

| ID | 入力 | 独立Expected | Owner |
| --- | --- | --- | --- |
| TT-51 Readiness成功≠生成資格 | #3284のCI成功、`partial`・`qualifiedTestGeneration=false` | Checker局所PASSを保持、T1の独立Expected/現在Qualificationは未成立 | #3262/#3273 |
| TT-52 部分Contextでの有用な提案 | 既知公開Ruleだけで行番号0の候補を提案。別の未読込Ruleが存在 | 提案はprovisional、未読込Ruleを存在しないと断定せず、独立Oracle確定/実行は保留 | #2514/#3273 |
| TT-53 Tool Receipt欠落 | Contextの一部は生成されたがHarness Tool Replyを未受領 | CLI実行成功でもDelivered/Observed/Qualifiedは別、欠測を合格にしない | #2509/#3262 |
| TT-54 BOM Draft消滅 | 旧#3269がclosed/unmergedでも旧PRにだけあったRQ-035を現mainの採用要求と誤認 | 現main Sourceを再取得してMeaning確認。旧衝突は履歴として残し、新BOM採用Branchで再検査 | #3267/#1119 |
| TT-55 Trait/Plugin設計を実装証明へ昇格 | mainにPlugin-first/Atomic Trait設計があるが、実Consumer Load・Manual Obligation Conformanceは未実施 | 設計PASS・静的候補・実Load/Invocation/Grant・現在Qualifiedを分離 | #3278/#1251/#2514 |

**現状と停止条件**：#3284の限定CIは実証済み。TT-51〜55の新たな統合E2E、T1生成・実pinned Rule試験、MCP/Studio、実Provider/Host/Plugin合成、Owner Meaning Acceptanceは**NOT RUN / NOT ACCEPTED**。Design Contextがpartialでも候補を提示することと、欠けた独立Expected/開示資格を補完して「正式合格」とすることは厳格に分離する。

<!-- tei-test-delivery-reachability-path-qualification-20261107 -->
## 18. 2026-11-07 追加：Design Contextの真の完了性と、Function/Entry/Decision Pathまでのテスト到達性（設計・未採用）

### 18.1 別チャットの実装・設計から抽出した、残る「空振り合格」

main統合済み#3284に続く[main #3296](https://github.com/hat47x/tei-platform/pull/3296)は、既存の `tools/reference-issue-test-ability/context_gate.py`（main Source blob `0c9873e7fcc129151668f3b0c4cbc74c74e9bc4a`）**一つだけ**を拡張し、既存CLI 9応答のschema、Source inventory + `readByLoader`、Subject、World/Index fingerprintを照合する。**既存7＋Manifest/Root 2＋Tool Reply byte改変7＝16件**の限定拒否条件は、exact main `d6a452f79023744e7926669c3556d2699f546697` の[push CI #42371592023](https://github.com/hat47x/tei-platform/actions/runs/42371592023)で成功。JSON重複キー・非有限値・余剰Object・途中切断等を**test-only Consumerで拒否**する。しかし合成copyの破損検出は、独立したRequired Universeの閉包や**実外部Tool受領**を証明する製品APIではない。特に、**構文的に正しいJSONへ再構成した末尾欠落**は、既知の対象だけでSource/Subject/FPの一致を維持し得る。必要な独立Required、Enumeration終端、開示可能なManifest、受信者ACKがなければ「その対象以外が存在しない」とは言えない。

並行して[Function/Build/Entry Draft #3283](https://github.com/hat47x/tei-platform/pull/3283)は、Function定義→意味→実Build収録→Entry公開→現在Grant→実Effectを**独立資格**とした。[HTTP Decision Graph main #3289](https://github.com/hat47x/tei-platform/pull/3289)のHDG-21は、静的 `onTerminal` ラベルが全種類揃い、`after` 順序ヒントがあっても、**各short-circuit実Pathを通ったAuditの因果証拠にはならない**という限定Characterization（V8 2/2）である。実HTTP HostのAuditバイパスが立証されたという意味ではない。

**結論**：テストCaseの実行・C0/C1/Graph EdgeのCoverageを増やす前に、1つのObligationが**本当に必要Sourceから、実配備Targetの必要Entry/Branch/Observerまで到達したか**を説明できる必要がある。Source Index、Build Plan、Service Contract、OpenAPI、静的Graph、単独CI合格を連結しただけでは、同一Attemptの実証にならない。

### 18.2 Test Design Contextの「完了」を偽装できない条件

- **独立したRequired Universe**：Userが選択したScope/Function/Requirementと受理済みMeaningから必要なSource/Role/Relation/非出力Rule/Procedure/Scenario/NFRを列挙。DiscoverできたリストだけをRequiredへコピーしない。そもそもRequiredが未確定なら `unbounded/unknown` とする。
- **Enumeration証拠**：対象Cursor/Scope/Source Revision/Visibilityと、読取元が保証する終端・件数・順序・可能なら内容照合値（権限のあるConsumerに限定）を別参照として保存。単なる `\n`、valid JSON、空の次ページ、自己申告の `complete:true` は完全性の独立証拠ではない。
- **Source・Loader・SemanticとDeliveryの独立性**：読めたbyte、Parserに届いたRole occurrence、Meaningの受理、Result投影、CLI stdout、MCP/外部Harnessの完全受領は別Witness。FP一致や最終Reply 2xxと外側Observerの受領済みを混同しない。受信者ACKが取得できなければ `delivery-unverified` を許す。
- **定義された出力形**：一枚のOverview + typed reference index + scope-relative Closure Ledger + 必要に応じるDetail Pages + `Claim Ceiling`。人が読むMarkdown/表/GraphとJSONは同じQueryの投影。自動省略項目・非公開Ref・未取得Sourceの存在と粒度を区別し、秘密のID/件数/digestは権限外に出さない。
- **変化と資格**：SourceがA→B→Aとなった、途中でSelection/Grant/Plugin/Target/Visibilityが切り替わった、Rule Dependency/否定依存が増えた場合、同じ見た目のDocumentから旧Qualifiedを復元しない。無関係Target/Plugin変更で無根拠に全件失効させることも避ける。

ここでの「完了」は*要求されたScopeの受理/開示可能な観測範囲内*だけ。Source不完全であっても、既知一Ruleについて**provisionalなテスト候補**を提案することは許す。候補生成と独立Expected確定、既存Driver Scenario採用、実Target結果確認、Owner Meaning Acceptanceを一つの `qualified` にまとめない。#3262がContextの品質を、#2605/#2509がSkill/実Harnessを、#2516/#2515がDisclosure/Egressを所有する。

### 18.3 FunctionのTest Coverageを「設計→配備→入口→分岐→観測」の二グラフで追跡

**Design/Requirement Graph**：Requirement/Business Procedure/Function/Data/Rule/Relation/Trait/Plugin Contract/Scenario Obligationの**必要な依存**（否定依存・未表出属性・時間/Scopeを含む）。これは何を検証するべきかの分母。

**Execution/Delivery Graph**：同じFunction/Ruleが実Build root→actual consumed source→Link済Artifact→Entry（CLI selector / HTTP route / Job）→現在Actor/Host Grant→実Request→Branch/Guard→Effect/Result→Response/Tool受領→独立Read/Observerへ流れた**観測された因果経路**。これは何を検証できたかの証拠であり、Design Graphの辺から自動的には導けない。

| Coverage / Qualification | Required Universe | Observedの必要条件 |
| --- | --- | --- |
| Function/Entry availability | Required Function×Target×Build Unit×Entry×Scope | SourceとMeaningの束縛、実Artifact収録、dispatch/route/Job実Entry到達、現在Grant。CLI成功をWeb/cronへ流用しない |
| Decision/Guard/Audit Path | 必須Rule Decision outcome、保護Guard、Terminal/Recovery path、Audit義務 | 実Path ID/Decision outcome、short-circuit時の必須監査経路、同AttemptのTraceと独立Expected。ラベル集合・topological orderのみは証拠不足 |
| Code C0/C1 | **実Instrumented Buildごと**のstatement / branch outcome | Target/Artifact/計測器/SourceMap/実Runを固定。Go標準statement値をC1としない。別Buildの%を単純足算しない |
| Business/Effect | Requirement/Scenario/Manual Step/Effect/Business Completion | Scenario step実到達、必要な人的ステップの別Observer、Receipt/Resource Read/Owner受理を個別に照合 |
| Non-functional | 明示NFR/SLO・環境/負荷/測定窓/統計/失敗回復条件 | 規定Target/Traffic/Duration/Sinkで実測。UIの表示やUnit PASSはNFR合格を示さない |
| Plugin/Trait/Composition | Required Trait Fulfillment/Contract×exact Provider/Target/BOM tuple | #3278のrequired/manual/default実装充足と独立Conformance、#3267の実Resolver/Build/Load/Invocation。各個別PASSやPairwise計画を合成Qualifiedにしない |

**C0/C1とGraph網羅は独立**：HTTP/Workflow/PluginのGraphで全LabelやEdgeが定義されたことは、実CodeのC1でも、実Host Auditへの到達でもない。必要Guardを実行前に必ず通過させる要求には、Path制約/実Traceの別検証がいる。異なるTargetに同じFunction意味を投影しても、生成コードの分岐母集団は同一でないことがあるため、C0/C1を集計表示するときは各Build/Toolの分母と対象割合を並記する。

### 18.4 重複を避けた追加横断負例 TT-56〜TT-63（個別Case/Expectedは各Owner）

| ID | 入力（具体的な違反） | 期待される判定 | Owner |
| --- | --- | --- | --- |
| **TT-56 parseable部分Response** | catalogが有効JSON・同FP・同Required Subjectを維持しながら**他の必須関係/Roleを省いた** | 既知一Subjectの局所一致を保っても、独立Required/終端の閉包がない限りDomain completeを拒否。newline/JSON parseでは不足 | #3262/#2509/#2514 |
| **TT-57 Function未収録** | Meaning上はFunctionあり、Build配置表にも名前があるが実Go Artifactには必要Symbol/実現がない | 設計上の存在を保持、Build-qualified/Entry実行資格はunverifiedまたはfailed。#3283のBuild Ownerに実Consumed witnessを要求 | #1130/#2831/#3283 |
| **TT-58 Entry非到達** | 実Binaryには実装があるが、当該CLI selector/Web route/Job triggerが未配線・旧Buildを呼ぶ | 当該Entryへの到達はnot-observed/unsupported。別EntryのPASSやAPI説明文を転用しない | #3283/#1158/#1180 |
| **TT-59 入口別意味ドリフト** | CLIとHTTPが同名Function/同じ表面Resultを返すが別Rule Revision/Provider版へbind | それぞれのSource/Build/Actual Input/Attemptを比較してparity不成立。HTTP status一致は十分条件でない | #1180/#951/#3274 |
| **TT-60 Terminal Audit見せかけ網羅** | HDG-21の `onTerminal` completed/rejected/errorが全列挙され、`after` で並ぶが拒否short-circuitのAudit因果経路未証明 | 静的Label合致は限定PASS、Runtime Path Auditはunknown。実Hostの脆弱性確定とは主張しない | #3274/#3289 |
| **TT-61 Trait実装欠落** | Manual `ConditionalWrite` がrequiredだが実装不足、他TraitはadvisoryまたはDefault-backed | RequiredのConformance不足を明示、advisory不足を同じ失敗にしない。実行Guard/Host Grantはさらに別 | #3278/#3281/#951 |
| **TT-62 Cross-Build Coverageの誤合算** | Go単一BinaryのC0と別Web/Job ArtifactのStatement/Branch Countを、分母や計測器を固定せず合算 | 各Target/Artifact/Source/Tool別Coverageしか表示しない。C1未計測をGraph Edge出現率で代替しない | #1180/#2514/#3283 |
| **TT-63 Manual+Async Outcomeの空振り** | Jobの受付ACK・HTTP 202・内部Step PASSがあるが、要承認Manual Step、Effect/独立Read、復旧Pathが未観測 | 受付/業務拒否/技術失敗/Effect unknown/実Business Completionを別Outcome。再送可能性を勝手に付与しない | #1130/#1174/#3274/#1125 |

TT-56〜63は**横断Verification Obligation**であり、#3283 Build/Entry試験、#3274 HDG、#3278 TC、#1180 Scenario、#3262 DCQの**新たなFixture正本を作らない**。既存Referenceへ要件と期待結果の**不足一辺**を伝えるための指標である。

### 18.5 実行順・最小受入と経済性

- **P-1b-a（最優先）**：main #3284の `context-readiness.json` と同じ公開Fixtureを用い、`TT-56`「構文的には正常だがRequired内の情報が欠ける」を独立RequiredとRead結果の差で拒否できるかテスト。まず既存CLI/Consumerで説明可能か試し、不足時のみ#3262へResponseの終端/Count等の最小追加を提案。Source/Actual Reply/外部Deliveryは別Witnessとして記録。
- **P0-T1（既存優先順位を変更しない）**：#3273の行番号0 Candidateを、T1 Required/Role/Source→Provisional Pattern→Review→既存Scenario→実Pinned CLI→独立Expectedへ一往復。#3284の `qualifiedTestGeneration=false` をAIが勝手にtrueへ変換せず、検証可能な局所Caseと未検証全体の両方を記録。
- **P1-Entry**：#3283のFunction一件をGo単一Build/CLI Entryで実照合し、次にWebまたはJob入口へ同一Scenarioを再生成。1→2→3 Buildは必要時の比較であり、初回T1のHard Dependencyにしない。Target別の実C0/C1/Branch/Guard/Outcomeに未測項目を残す。
- **P1-Decision**：#3289のHDG-21 Characterizationを前提とするが、実Hostのterminal-path Audit/Client Reactionの観測には別のActual Traceと独立Oracleを要求。Code C1導入は別のInstrumentation正確性PoCとして#2514へ依頼し、静的Graphで代用しない。
- **P2-Portfolio**：#3278 Trait required/manual missing、#3267 BOM exact matrix、#3270 Workbench Provider、#2586実Read Scopeを専門Patternとして採択し、#2018に指示量・Tool回数/Token・正しく検出したGap・誤PASS・測定費用を報告。

**停止条件**：正常なJSONだけで全Required確認済み、Function定義だけで全Build/Entry到達、Auditラベルだけでshort-circuit Path監査成立、他Target C1へGoのC0転用、HTTP 202/Job ACKを業務完了、Trait宣言を実Impl/Grant、個別Plugin PASSを合成BOM PASSとする主張。

**状態**：#3284/#3296の限定実main CI（後者は16 Controls）と#3289の2ケースV8 Characterizationは他Ownerの有限実証済み。#3283は設計Draft。今回提案したTT-56（**構文的に有効なままRequired要素を欠落させる独立反例**）〜TT-63、新P-1b/Entry/実Audit/T1の製品E2E・MCP/Studio・Owner意味受理は**NOT RUN / NOT ACCEPTED**。この節はGitHub Source/設計の突合に基づく計画追加であり、新Runtime/Resolver/Host/Registry/Runner/Scenario SSOTを作らない。

<!-- tei-test-t06-content-pin-effect-cross-source-20261107 -->
## 19. 2026-11-07 横断深化：T0.6実Source/Lock試験とAI用Contextの安全な入場判定（設計・未実装）

### 19.1 mainで前進したT0.6は本物の変更拒否。T1は依然未達

[#3299](https://github.com/hat47x/tei-platform/pull/3299)を統合したmain exact `04498c41475db7ac71fbf0fdd0c8919c40108258`の[push CI #42373611474](https://github.com/hat47x/tei-platform/actions/runs/42373611474)は**completed/success**。同一main・単一Go Binaryの9 read-only CLIと既存Context Readiness 16 Controls、生成済み6 pinned Scenario Step、**実CLI拒否6件（従来4＋実Canonical Source改変・実Plugin Lock byte改変の2件）＋独立義務/Oracle拒否2件**を実行。C0相当10.1%はcmd/teiの局所1 Test、C1はunmeasured。Artifact保存は7日保持の限定参照。実Evaluator Code Mutantを8件殺した意味ではない。

新2件は**実一時Workspaceファイルを生成Plan後に改変**した試験であり、§18の受領済みJSONコピーを破損させる16 Controlsとは異なる。Sourceの型変更ではWorld fingerprint、Plugin Lockの意味等価な空白変更ではLockの原byte SHAが変わり、旧pinned runが拒否された。**A→B→AでSource原byteへ戻すと旧pinは再び成功**した。よってContent Pinが示すのは必要byteの現在一致であり、履歴上の無変更、最新Accepted Meaning、Ownerの再受理、現在Host/Disclosure Grantではない。これらは独立した現在資格を要求するか、必要Evidenceがなければunknownに止める。新たな全変更History Storeを本テスト設計だけで導入しない。

### 19.2 AIに渡す最小入力を「不足を分類する取得票」とする

AIに与える最短指示（「このFunctionの未カバーテストを提案」）の前に、**取得したDomain Contextがなぜ十分/不十分なのか**を利用者にも機械にも同じ表で返す。既存#3262 Design Context Query／main `context_gate.py`を利用し、別のPacket Readerや同名SchemaのSSOTを追加しない。

| 独立に判断するGap | 欠けている証拠 | 直ちに薦める行動 | Candidateの扱い |
| --- | --- | --- | --- |
| **Required/Source gap** | Requirement、Business/Rule/Relation closure、Loader/Parser Role、Non-output dependency、Source revision | Design Contextの追加取得・Model/Ownerへの不足の引継ぎ | Ideaはprovisional。存在しないというNegativeやCoverage分母の完成を宣言しない |
| **Meaning/Oracle gap** | Contextの受理範囲、Unit/Time、Independent Expected・Baseline合法性、Holdout境界 | 候補の説明・第三者/Ownerによる期待値固定を先に要求 | AI生成Expectedで自己採点不可。正式Scenario採用は保留 |
| **Execution/Target gap** | 実Buildの収録、Entry/Dispatcher、Plugin Lockとactual Load、Tool Reply、Provider/Target Observation | 既存#1180の実Target接続か、最初の欠損辺の診断 | テスト案は出せるがexecuted/coveredと表示しない |
| **Freshness/Disclosure gap** | Source/Lock/Tool/Scopeの時点一致、Current Grant、受信者Complete Delivery | 同一Scopeの再取得または外部送出拒否 | 許可されないRaw/存在・件数・digestを出さない。Content Pin一致を現在Grantとしない |
| **Genuine uncovered obligation** | 適格なRequired集合には含まれるが、実Scenario/独立Observationが未到達 | Domain Patternから新Case候補を提案し、#1180へReview引継ぎ | **新規テスト提案の価値を評価できる唯一の入口**。旧Scenario再Runを生成実績としない |

この「診断優先のPattern推薦」は、あらゆる未知に対してテストを生成して埋める方式とは異なる。**未カバーの理由がSource不足なら追加取込、Oracle不足なら独立Review、Runtime不足なら接続試験、真のTest Gapなら新Case生成**を勧める。ユーザー向けMarkdown、生成AI向けtyped JSON、将来MCP Toolは同じRequirement/Ref/Statusの投影でよい。トップ3推薦等のUX順序は#3223 §12候補を維持し、新CLIコマンド名は未実装の案としてのみ扱う。

### 19.3 呼出しの安全性はTool名でなく「全argv／実File Effect」に束縛

現mainに対する実観測 [#3182](https://github.com/hat47x/tei-platform/issues/3182) では、tei commandsのread-only広告に載るviewが、実際は **-execute + -operation-log / -lifecycle-state** で名前付きファイルを生成できる。[#3302 Draft](https://github.com/hat47x/tei-platform/pull/3302)はread-only discoveryを非execute previewへ限定し、excludedFlagNamesを追加して3種flag表記を実ファイルEffectで検証する**修正候補**（main採用・新CI完了は本節では未確認）。したがって：

- T1のDesign Context探索は実binary identity・command＋**完全なargv**・flagsの意味・working directory・read対象・書込み先・効果観測をScopeに束縛。command catalogのread-only表記を実書込み禁止・OS隔離・外部送出許可と同一視しない。
- Skill/AIのread-only profileでは、非executeの読取経路だけを選び、実際に- execute相当のflagが混入した場合はpreflightを拒否。argvが後段で変わるなら実Process受領argvで再確認。flag名の除外広告をOSのwrite防止と呼ばない。
- 生成したTest Case/Scenarioの実行はread-only探索とは**別のEffect Profile**、Disposable Source・Output Path、独立Expected・Receiptを要求。Source/Plugin LockのA→B→AはContent Pinに現在Authを付与しない。
- 結果stdoutに個人情報や非公開Sourceが直接ない場合でも、Tool結果、stderr、Session Log、診断、集計件数/成否のSide Channelは個別Disclosure対象（#2515/#2509/#2516）。

### 19.4 同名Query／Graph／表現の間に「同じ実験である」という錯覚を作らない

[#2586](https://github.com/hat47x/tei-platform/issues/2586)のRMT検証で同じ `recent_orders` の名を持つものをE（examples/xtea-lowcodeのstatus引数SQL）、W（reference/workspaceのordered_at + from SQL）、C（Web Reference PostgreSQLの別物理Table/Tenant Scope）へ区別している。[#3238](https://github.com/hat47x/tei-platform/pull/3238)のCの実PG成功は、E/Wで実選択されたSQL/引数/物理Provider/Contextを証明しない。**Project/Workspace、Function/Query Source byte、DataSource+Provider Instance、param、Scope/epoch、Rowset/Consumer**が合致したときだけ一つのTest TraceへJoinする。

三視点図面・CRUDについては、旧総合Draft [#3236](https://github.com/hat47x/tei-platform/pull/3236) が**Closed/Unmerged**となり、現main向けの置換[Draft #3303](https://github.com/hat47x/tei-platform/pull/3303)へ引継がれた。[#3263](https://github.com/hat47x/tei-platform/pull/3263)の合成Entity9/Relation7/Function4/Business Action3のInputとExpectedはAI作成の別Blobであり、実Parser Source/独立Oracleの資格ではない。Graph/CRUD Matrixに見える欠落・Cell空欄を「業務上存在しない」へ昇格しない。P0-M（Source→図）、P0-R（実Read）、P0-C（両者の同一性）を**分けて計測**し、T1のHard Dependencyにはしない。

### 19.5 独立した横断負例 TT-64〜TT-72

| ID | 入力・負例 | 判定すべき事柄 | Owner |
| --- | --- | --- | --- |
| **TT-64 実Source TOCTOU** | Context取得・Plan生成後にCanonical Source型変更 | 旧World Pinを拒否し、実Run/Consumerはstale。旧Contextを現在Meaningと呼ばない。#3299 mainで一時Source変更の拒否は実測済み | #3262/#1180/#2514 |
| **TT-65 実Lock TOCTOU** | Plugin Lockを意味同値だが原byte違いに変更 | 旧Lock Pinを拒否。ただし実Provider Link/Invoke/Grantの不適格を新たに証明したとはしない。#3299で局所実測 | #3267/#951/#2600 |
| **TT-66 A→B→A** | Source変更後に原byteへ復元し旧pinned Planが再受理 | 現在Content一致だけ成立、途中無変更/現意味受理/Grantや旧Evidenceの無条件復活は禁止。#3299で再成功を観測 | #2600/#3262/#2514 |
| **TT-67 Tool argv Effect** | read-only Discoveryのviewを実-execute + named outputで呼ぶ | 読取専用Harness資格は拒否/別Profileへ。実File Effectを観測し、Discoveryと効果/隔離を別々に判定。#3182現挙動、#3302未統合候補 | #3182/#1155/#2605 |
| **TT-68 同名SQLの取り違え** | E/W/Cのrecent_orders・別SQL/Physical Providerの結果を一つに合算 | SQL実消費byte/params/Scope/epoch/Consumerが異なると別Claim。Cの実PG PASSをE/W製品へ転用しない | #2586/#2506/#2514 |
| **TT-69 三視点Source不在** | Synthetic Diagram/CRUDが自己整合PASSでも実TEI Loaded/Parsed Sourceから生成されない | 内部Fixture一致まで。Owner独立Expected・実Graph/Matrix Source ref・表示/Consumer実受領は不適格 | #3263/#3303/#2514 |
| **TT-70 Gap誤分類** | Requirementが未読込なのに未カバーテストとしてCase生成/coverage改善済みと表示 | Source gapとして提示、provisional案は可、独立Oracle/現在Coverage改善は不可 | #3262/#3273 |
| **TT-71 生成者自己採点** | AIが入力とExpected/Mutantを同じ文脈から生成し、旧6 Stepと同内容を新規Caseと申告 | 新規性・独立性不成立。Holdout独立評価、新しいline_number=0のCase/Step ID・観測が必要 | #3273/#2514/#1180 |
| **TT-72 不正な部分開示** | Toolに返すJSONは短くしたがSession LogやMarkdown診断に秘密Scopeの件数が含まれる | Disclosure/実Sink/OS隔離が未確認ならExternal-ready=false。完全性レジャーも秘密Countを公開しない | #2509/#2515/#2516 |

TT-64〜66の*限定実Source/Lock試行*のみ#3299の実証を引用する。TT-67は#3182実コード挙動観測、#3302は修正候補。TT-68はCのReference PG、TT-69は合成Fixtureに限定。他の**横断した生成/拒否/非開示のE2Eは未実施**であり、旧Caseの結果を新Caseの独立PASSに付け替えない。

### 19.6 最短の実装・検収順を更新

1. **現main T0.6を固定**：main `04498c...` の9 CLI、Context 16 Controls、Scenario 6 Step、CLI Negative6＋独立Negative2、Local C0/Unmeasured C1をSource・Build・Attemptごとに記録。これを新たなAI T1/Context Full Closureへ昇格しない。
2. **P-1b Source/Rule/Gap説明**：既存Reader/Context Gateでテスト用の部分的なDesign Contextを提示し、ReasonをSource/Oracle/Execution/Freshness/真のTest Gapへ分類。§18 TT-56の「parseableな必須情報欠落」は**独立Requiredが確認可能な一局所Scope**で先に反証。必要な追加契約だけ#3262へ戻す。
3. **T1は一つだけ実生成**：[#3273](https://github.com/hat47x/tei-platform/pull/3273)の新しい行番号0のRule負例を、実行できた現9 CLIと固定Skill/Allowed ContextからAIがprovisional生成し、別Holdout/Expectedを持つReviewer→既存Scenario SSOT→新pinned CLI→独立Rule結果へ接続。後段の実Source/Lock Pin拒否はT0.6の実挙動を再利用し、同じ実験Scopeで再照合。
4. **CLI Effect/Skillを先行対照**：#3302が採用される場合も、Discovery optional欄の一致だけでなく**完全argvの実ファイルEffect**を実検証する。read-only Skill比較の条件を変えるときは別版・別Runとし、旧H2/H3四Sessionに事後追加しない。
5. **D/T/R/P/B/Eを並列投影**：#3283の設計(D)、新Test(T)、実Provider Read(R)、Presentation(P)、実Build/Entry(B)、業務Effect(E)を**6軸**で記録し、未完成軸を消さない。P0-M三視点図とP0-R実Readは並列可、P0-Cの同一性は両方実観測後。
6. **経済性と昇格**：#2018へ(利用者指示回数/Tool ReplyとToken/欠損正答/False-qualified/Source再読・CI費用)を送る。新Test Case数だけでなく、**何の欠損を先に教えたか**、不足が修復した後に初めて独立Expectedへ到達できたかを評価する。

**横断停止線**：fingerprint一致→歴史・認可有効、read-only tool名→副作用なし、同名Query→同一Target SQL、合成Graph/CRUD一致→Product Source/独立Oracle、LLM自己Expected→Business Accepted、CLI T0.6→T1生成PASS、各Target/要件/C1/NFRのCoverage分母合算、未許可Scopeの存在/件数を説明に含めること。

**今回の実施状態**：本節はGitHub mainの実T0.6・#3182現行観測・別チャットのRMT/Modeler/Plugin/SHO議論を、既存#3262/#3273/#2514/#1180/#1119のOwnerに割当て直した**設計追補**。新Go/CLI/PG/実Harness/AI新Case/T1/Tool ACL/Function-Provider-MCP実連結は**NOT RUN**。#3302/#3303/#3263の正式採用、Owner Meaning Acceptanceも未成立。新Engine/Schema/Host/Resolver/Scenario/Oracle正本を追加しない。

<!-- tei-test-t1a-to-blind-t1b-20261107 -->
## 20. 2026-11-07 横断更新：T1-Aの実走を受けたT1-B盲検発見・証拠資格・Testabilityの次の受入

### 20.1 成熟した実績を、観測した範囲だけに引き上げる

| Gate | 観測済み | 今回も成立を主張しない項目 | Source/Owner |
| --- | --- | --- | --- |
| P-1a CLI/Context | #3284/#3296の9実CLI応答、Source/Subject/FPの限定照合、Contextの7+2+7=16拒否対照 | Required全体列挙、Parser Role/Meaning、Provider・外部Harness全受領 | #3262 |
| T0/T0.6 Scenario | 既存6 pinned Step、CLI 6種＋独立義務2種の拒否、#3299の実Canonical Source/Plugin Lock改変 | A→B→A間の歴史、現在Grant、正式業務Effect、全C1 | #1180/#2514 |
| **T1-A（今回の到達点）** | main統合済み#3304。既知の行番号正数Ruleに対する0境界を一時第7 Stepとして実CLI verify→generate→check→runし、偽Expected・偽Fixture・stale Pinを拒否。PRのCI #42378033965で限定成功 | AIが未提示Ruleを発見、別OwnerがScenario採択、真のProduct Code Mutant kill、実Source完全閉包 | #3273/#2514 |
| **T1-B（次の未達）** | 実績なし。提案時点のRuleや行番号0は既に公開されており盲検に使えない | 独立Holdoutの非開示、AIによるGap同定、再現・独立判別、対照試験の比較適格性 | #3273/#2509/#2018 |
| T2/T3 | T1と別受入 | 実Provider Query/Read・Host Effect・実Plugin/Trait合成・実Target配備・全体Self-Build | #2586/#2831/#3279 |

**証拠の読み分け**：#3304専用CI successは、同PRが後続文書更新を行う前の確定プログラムHEAD 02d4c624... に対応する。最終PR HEAD 2e79d61c... と統合後mainのWorkflowは別のAttemptであり、旧HEADの結果を流用しない。現在のGateの合否を示す際には対象Commit・実Checkout・Binary SHA・Tool/Scenario bytes・独立期待値・Actor/Scopeを同Attemptに束縛する。

### 20.2 T1-Bの最小実験は「盲検で未知Gapを選べるか」だけを独立に測る

- **入力の事前固定**：独立Reviewerは対象ScopeのRequired Rule/Relation、合法Baseline、未カバー境界の選定理由、独立Expected、Targetと許可Scopeを固定し、生成側には出さない。生成側へは許されたDesign Contextの実受領済部分と、明示したUnknown/不完全性だけを与える。公開済み行番号0の再提案はT1-Aの再生でありT1-B新規発見に加算しない。
- **発見と実行の分離**：生成AIは不足を診断し、Source/Rule ref・理由・最小差分・副作用となる他Ruleを提示する。Reviewerが候補を受け取った後に独立Expectedと照合し、採択は既存#1180 Scenario SSOT経由。実行は既存CLI pinned commandを使用し、第二Oracle/Runner/Authorityは設けない。Unknown Sourceから得た値はOwner受理済みExpectedへ昇格しない。
- **資格の独立性**：生成した数でなく、選ばれた新規Obligation、実際に触れたRule、合法Baselineの維持、既知Mutantの判別を評価する。人工のテスト入力でProduct Targetを判別できたこと、実Providerで正しいRead/Effectを生んだこと、全要件をカバーしたことは別Claimとする。
- **H2/H3比較の公平性**：同一Source版・Allowed Context・Model/Budget・独立Holdoutを固定し、Skill有無の実Sessionを順序効果と分離して比較する。ただし#2509で報告済みの8,192byte切断、56 Request/55 Reply等により、**全受領・非開示・外部Task完了が未資格化のSessionは評価不能（ineligible）**とし、AIの能力不足として誤採点しない。失われたResponseや秘匿値は自動補完しない。
- **漏えいと再現性**：外部Harness/Issue/CIへ公開するのは許されたScopeと秘匿情報を含まない結果要約のみ。Hidden Expected/非公開Source ID・Digest・件数・canaryは生成Agentにも一般Artifactにも置かず、Owner境界内の独立保管とする。必要な公開evidenceは対象Revision/Attempt IDと判定状態の参照だけ。#2515のTool出力自体のEgressを独立検証する。

### 20.3 Claim ledgerと差集合を一つのResultに保つ

| 集合／状態 | 既存実行から取得するWitness | 不在時の安全な帰結 |
| --- | --- | --- |
| Required | 独立ScopeとSource/Role/Meaningの受理範囲、Dependency/負の依存 | 分母 unknown。見つかったRuleだけで「全件」を作らない |
| Received | 実Tool Reply bytes、page/終端、ProducerとConsumer ACK、許可された開示Scope | delivery-unverified / partial。exit0やparseable JSONでは補えない |
| Proposed | 生成候補のRule ref・根拠・合法Baselineとの差・生成前に知られていた境界の区別 | provisional。旧Caseや公開済み行番号0の焼直しはnewly-discovered=false |
| Independently checked | 生成者から隔離したExpected、反例の予測結果、別版Mutantの検出条件 | expected-unqualified。生成者と同じ根拠の自己採点を不可 |
| Executed/Observed | Scenario/Fixture/World/Lock/Binaryのpin、実Target argv・Result・Error・独立Resource Read | 実行レベルの範囲のみ成功。Receipt=Resource=業務完了としない |
| Currently qualified | 現在Source/Model/Build/Scope/Grant/ConsumerとClaimの一致、Owner判断 | unknown/stale/not-runを保持。過去PASSを再利用しない |

**二つのグラフ**：Design/Requirement Graphで「必要」を、Execution/Delivery Graphで「実際に消費・配備・到達・観測」を保持する。Node表示・Graph edge・関係表・Function内Presentationへの投影が一致しても、実Read/実Dispatch/Guard/Observerの因果は独立に測る。C0は観測したGo statementのみ、C1は真の分岐計測がない限りunmeasured、業務Obligation Coverageは独立Required分母がない限りunknown。

### 20.4 重複しない横断の独立対照 TT-73〜TT-80

| ID | 最小反例・入力 | 独立に判別すべき状態 | Owner |
| --- | --- | --- | --- |
| TT-73 | #3304の行番号0 Candidateを「AIが未知Case発見」と報告 | 既知提案の実走=T1-A、未提示Gap発見=T1-B。既存第7 StepのPASSを盲検発見へ転用しない | #3273/#2514 |
| TT-74 | 行番号0だけ変え、Relation keyを旧値のままにする／Quantityも負にする | 複数Ruleを壊したFixtureを単一境界の判別力と数えない。独立Baseline・行とKeyの意味整合を確認 | #2514/#1180 |
| TT-75 | 実TargetがResultを返したが、Toolの最後のReplyまたはPageが失われた | Target実行成功と受信側の完全受領を分離。欠落Trialは不適格で、負例が発見できなかったとの判定に入れない | #2509/#3262 |
| TT-76 | Holdoutに事前公開RuleとExpectedを混ぜる／Generator自身が採点 | data leakage / shared-origin oracleを検知し、盲検新規発見資格を拒否 | #2514/#3273 |
| TT-77 | Skill有/無のSessionでSource版、Budget、Allowed Context、Model、実受領数が異なる | 両群を直接比較しない。資格の揃ったPairだけを指示数・費用・正負判別の分母とする | #2509/#2018 |
| TT-78 | validation.teiのテストコード内読込と宣言一致だけを製品Loader/Accepted Meaning扱い | T1-AのSource参照はtest-only witness。Product Reader/Parser/Meaning/実Rule Bindは独立にunknownを保持 | #3262/#2514 |
| TT-79 | Candidateは拒否を観測したが実行経路は常時fail／実Guardに到達していない | 正の対照・偽Expected・Fixtureの逆転・実Path到達を要求。exit1だけでMutant killと認めない | #1180/#2514/#1163 |
| TT-80 | H3のResultは適正でもCI/Issue/Session診断にHidden Holdout・非公開件数が流出 | 出力投影だけでなく各Sinkの許可と実送出を判定。秘密値は証拠に複写せず、非開示資格を別Gateとする | #2515/#2516/#2509 |

### 20.5 実装・検収の順序（既存Ownerに限定）

1. **P0／Mainの実績更新**：#3304の最終HEAD・統合後mainの専用CI Resultを別々に採録して、T1-Aの実観測1候補・3負例のCase×Attempt×Evidence票を#2514へ反映。PR CI #42378033965以外の新実行成功は取得前に主張しない。
2. **P0／T1-Aの限界可視化**：#3262の限定Contextにvalidation.teiのSource/Loader/Parser/Meaningと未収集分母を明示し、Rule自身のSource宣言をtest-onlyコードが読んだ事実をProduct Consumed Sourceへ変換しない。必要なReaderの最小実装判断を#3262へ渡す。
3. **P1／T1-Bの一件だけのBlind Holdout**：#3273/ #2509が生成者に非公開の独立Obligationを固定し、許可されたTool Replyの受領完全性確認→不足Rule候補提案→Owner Review→既存Scenario→独立判別の順で実施。欠測/未許可Scopeを不適格に分類し、比較適格0なら0と明記。
4. **P1／Coverage分母を実施前後で比較**：Required/Loaded/Bound/Proposed/Reviewed/Executed/Killed/Currently-Qualifiedの差集合を#2514へ出す。業務C0/C1、Go C0/C1、Path/Guard Coverage、BOM Version Matrixを混ぜない。未知は数値0にしない。
5. **P2／実Target・複合構造**：#2586の実Provider read、#3279のPlugin実Consumer、#3283 Function Build/Entry、HTTP Branch/Receipt/独立ReadをSource/Target/Scopeごとの独立Attemptとして検証し、#1163のEvidenceで結合可能な場合だけ上位Claimを評価する。

**停止線**：既知Case再現からAI新発見を主張しない。生成者がHoldoutを閲覧したら盲検は失効。Reader証拠なしの「Source閉包成功」、CI実行結果なしの「実測成功」、正例なしの「拒否成功」、C0をC1に読み替える表示、external tool名だけでread-onlyとする扱いは不採用。本文は**設計の追加／実績の検証済み範囲の反映**であり、T1-B新実行・Hidden Holdout/外部Harness・実PG/Host・Owner受理はNOT RUN。

<!-- tei-test-coverage-obligation-pattern-selection-20261108 -->
## 21. 2026-11-08：Coverage Obligationを先に確定する生成規則と、T1-Bの判別力検収

### 21.1 「カバレッジ」を一つの数字へ合成しない

テスト推薦に渡す最小単位は、**対象Scopeにおける検証義務（Obligation）**。これは新しいTEI Core型・第二Scenario正本の採用提案ではなく、#3262のDesign Context、#1180のScenario、#2514のIndependent Expected、#1163のEvidenceから得るread-only投影の概念的な行である。各行は少なくとも `Claim/Requirement ref、検証する条件・分岐・境界、対象Function/Rule/Relation/Guard、Sourceと受理範囲、Version/Target/Scope、独立Expected ref、必要な実Observation、既存Test/Receipt、取得・開示・資格状態` を参照する。判定前に、Source/解析・Meaning・独立Expected・実Entry・Tool受領・現在Grantの不足を別々に残す。

| 別々の指標 | 分母と観測方法 | 不正な昇格を止める条件 |
| --- | --- | --- |
| **Code C0** | 対象Build/Package/Source versionの計測可能な文・命令と、実際に実行した集合。Go statement coverageはここに限定 | 単一Packageの10.1%を全Productの値にしない |
| **Code C1** | 実instrumentした分岐ごとの結果（true/false、switch各辺、error path等）の到達。計測対象・除外根拠を保持 | Go標準のstatement coverage、Rule正負一件、Graph構成EdgeをC1と呼ばない |
| **Model/Rule/Guard** | AcceptedなRule境界と業務Decision/例外経路。実Result・必要なら実Guard到達で判定 | Ruleを宣言・Catalog列挙しただけではcoveredでない |
| **Requirements／Scenario** | 独立に特定された機能・非機能Obligation、Target/Consumerの必要観測 | Source未取得・Requirement unknownを分母ゼロや達成扱いにしない |
| **Discriminative／Mutation** | 独立Expectedの正例・負例と、対象Mutationを候補Case自身が識別した件数 | CI全体failure、常時拒否、別Caseのfailureはmutant killでない |
| **Plugin/BOM組合せ** | exact Contract/Provider/Consumer/Target/Lock tupleの必要組合せ | pairwise生成済み・部品単体PASSを実tuple合格へ加算しない |
| **NFR／Effect／UI** | 時間・負荷・中断・秘匿・可用性等の宣言SLO/測定環境／実Read/Effect／実表示操作 | 合成Fixtureやmock、warm cacheのみの結果を実業務・実UIへ昇格しない |

各分母は `required-known / required-unresolved / excluded-with-reason` を分ける。独立Requiredの閉包が未証明なら「既知部分に対する局所率」だけを対象・revision付きで示し、**全体率の確定値は unknown** とする。Coveredも `planned、generated、reviewed、executed、observed、independently-qualified、currently-usable` を混ぜない。Test生成によって単に候補数が増えたことは、C0/C1/業務Coverageの上昇ではない。

### 21.2 Pattern推薦は、まず「なぜ未カバーか」を選ぶ

#3262のRequired Closureと受信完全性を先に照合し、**Source/Parser不足、Meaning未受理、独立Oracle不足、実Target/Consumer未到達、Grant/Disclosure失効、真にTest未実行**を分類する。最初の五つに対しては不足を埋める操作・追加調査を優先推薦し、最後だけ新Test Candidateへ進む。AIはTool/DSL出力の形式から暗黙に意味・権限を補わない。

真のGapへ適用するTest Patternは `必要なMeaning/型/Unit/Grain/Scope、前提、最小入力変更、合法Baseline、変えてはいけない独立Invariant、Expectedの取得先、実Entry/Observer、故障注入可能点、失効条件` を持つ。初期の候補群は次のように一般化する。

- **Rule境界型**：範囲端点・NULL/欠損・精度・一意性／複合キー・時刻境界を、型とOwner受理済みRuleに応じて選ぶ。既知行番号0例を新発見に流用しない。
- **Topology／Query型**：1:N×1:Nの行増幅、Entity/Relation/Occurrence別粒度、空と未取得、部分集合・NOT EXISTS、別時点Sourceを反例化する。実ProviderとQueryの選択byteがない場合は「未実証」。
- **Decision／Client反応型**：HTTP/CLIの業務拒否、技術失敗、再試行、監査終了をGraphの宣言Edgeだけでなく実Guard・型付きResult・Client Reactionで照合する。
- **Trait／Plugin／Version型**：manual-required欠落、Target unsupported、同名/同Version異digest、Provider×Consumer exact tuple、失効後の旧Evidence誤Reuseを検出する。
- **非機能・Interaction型**：負荷・遅延・取消・アクセス拒否・秘密漏えい・GUI/CLI/MCP別受信／実表示・実Effectを対象環境ごとの別Obligationとして保持する。

推薦順位は単なるCase件数最大化ではなく、(a)未検証の重要なObligationが対象か、(b)前提・Oracle・Targetが揃うか、(c)他のRuleを壊さない独立判別力があるか、(d)実行と再現の費用、(e)情報開示・副作用制約で比較する。未閉包のObligationを低スコアで黙って捨てない。使いやすいPatternはPlugin/Dataとして交換可能にし、Code固定の巨大なTest Generatorや別Runnerへしない。

### 21.3 T1-B最小試験を、盲検発見と「誤って通るテスト」の検出に分解する

**事前登録**：Independent Evaluatorが、既出 `line_number=0` と公知#3017 Mutantとは異なる一つのQualified Scopeを選び、未カバーObligation、合法Baseline、正負Expected、比較用の正しい実装と意図的誤実装（可能なら無関係の対照変更）を固定。保管は#2514のOwner境界。AIにはRuleを理解するのに許されたSource/Design Contextは開示するが、選択された隠れたGap一覧、採点・Mutation・期待値表・Holdout手順は見せない。Rule Meaning自体を隠して不可能問題にしない。実受領が不完全なら `ineligible` とする。

**提案**：Generatorが `Rule ref、未カバー理由、合法Baseline→入力差分、正例＋負例、変えてはいけない他のRule/状態、実行入口・必要観測` を返す。既存Caseと等価なInput/Target/Expectationの重複を判定し、Case IDや文章が新しいだけでは `novel=false`。候補はprovisionalで、永続Scenario SSOTへ無審査追加しない。

**独立評価**：Reviewerは候補受付後にHoldoutへ照合し、既存#1180の一時Scenario→verify/generate/check→pinned実CLIを走らせ、(1)正常実装で合法Baselineが合格、(2)目的の負例だけが指定Rule/Guardを拒否、(3)実Targetに到達している、(4)偽Expected・無関係の変更・常時fail・別Case failureでは誤ってkill認定しない、(5)対象Mutantに対し**当該Caseが**独立Expectedと異なる結果を観測、を別々に判定する。Mutationを利用できない対象ではMutation結果はnot-runとして残す。

**比較**：AIなし／既存手順／AI＋Design Context／AI＋Skill等を比較する場合、Source・Allowed Scope・Target/Tool・Budget・Model・Oracle・実Tool全受領の適格性を揃える。例数1のT1-Bは技術成立の局所証拠であり、生成AI能力の一般的優位・全体網羅・業務Owner受理を主張しない。

### 21.4 変更・Reuseに対する義務単位の再資格化

Source・Rule・Relation/negative dependency・Target/Plugin Lock・Generator・Toolchain・Independent Expected・Observer・Audience Grantの変化を、Obligation→Scenario→実Build/Entry→Evidenceの依存関係へ投影する。変化のない局所テストは必要以上に失効させず、**古いPASSの再表示と現在の利用資格**を切り離す。A→B→A、実Readのpartial、同Version別byte、Tool Page欠落、別Tenant、Build中のSource変更で「現在の完全な資格」を自動回復しない。依存がunknownなら再利用判定をunknown/拒否とし、性能優位の評価はその後に行う（#2600/#3267/#3271）。

### 21.5 次の判別負例（TT-81〜TT-88、すべて新規実行NOT RUN）

| ID | 狙い | 正しい判定／主Owner |
| --- | --- | --- |
| TT-81 | Go statementの実行だけで分岐両辺C1達成と報告 | C1 unmeasured／#2514 |
| TT-82 | Required Rule未列挙でも「発見済Case数=分母」として100% | Product Coverage unknown／#3262 |
| TT-83 | Known Caseを別名で複製しT1-B新発見と申告 | novelty=false、T1-Aの再生／#2514/#1180 |
| TT-84 | 正しい実装でも常時失敗するTestがMutantも落とす | invalid baseline、killを認定しない／#2514 |
| TT-85 | Candidate自身は実Targetへ到達せず他のGo TestがMutantで落ちる | target-not-reached／inconclusive、kill不可／#2514/#1163 |
| TT-86 | GraphのException宣言は網羅、Client Reaction/実Guardが欠落 | Model coverageと実HTTP/Client coverageを分離／#3274/#1180 |
| TT-87 | Provider 2×2のPairwise計画を生成、実Runは3 PASS＋1 SKIP | verified=3、matrix-complete=false、実Target qualification不可／#3267 |
| TT-88 | NFR遅延をmock・warm cacheのみで測り実Provider・cold/warm・負荷のSLO合格と報告 | 環境別実測のみ採用、実Provider/NFR qualification不可／#2586/#2600 |

**実装の最短順**：(1) Draft #3324のexact HEADと統合後mainを別々に検収し、一Ruleの必要Source/Relation閉包が不足ならT1-B範囲を狭める → (2) 既出Caseと重ならない一Obligationを独立事前登録 → (3) 既存CLI/Skill受領でAI Candidateと正例・負例を生成 → (4) #1180/#2514/#1163でTarget到達・独立Expected・Mutation対照・Coverage差集合を観測 → (5) Source/Plugin/Target変更で再資格化判定 → (6) 必要時にだけ実Provider・HTTP・Studio/MCP・Version Matrixへ拡張。

**非主張とOwner**：この§21の追加はCoverage/Pattern/Blind Trialの設計を既存#3262（Context）、#2514（独立判別）、#1180（Scenario SSOT）、#1163（Evidence）、#2509（Harness配送）、#3274（HTTP）、#3267（BOM）、#2600（Reuse）へ割り当てるもの。新しい正式Schema・CLIコマンド、Test Engine/Oracle、Host/Grant、CI実行、T1-B発見、C1実測、正式Meaning Acceptanceは**実施していない**。

<!-- tei-test-required-slice-actual-ci-and-cohort-20261108 -->
## 22. 2026-11-08 横断統合：必須Rule Sliceの実CI、同時点性の未証明、Test/Evidenceの到達資格（Owner別計画）

### 22.1 §21の前提を更新する「既に実証された最小Required差集合」

[Draft #3326](https://github.com/hat47x/tei-platform/pull/3326) は旧9 CLI Replyでは**subjectモード**だった `source-index` を全件 **indexモード**でも取得する10番目の既存read-only CLI呼出しへ増やした。独立に事前固定した公開ReferenceのValidation 3件（`quantity_positive`、`line_number_unique`、`line_number_positive`）に対し、Source `semantic/sales/validation.tei` の `readByLoader`、Catalog `semantic-validation` の一意性とkind、indexの`definition` role/Source path、Source artifact/fingerprintの限定整合を判定する。既存16受領/Manifest負例、旧6 pinned Scenario、T1-A候補第7 Stepも同じWorkflowで実行する。

**新規実測**：当該PRのexact HEAD `c2d2da10cc93022697204e66e83c30248b3810a0`と現在Branchの同一性はGitHub compareで`identical`（ahead=behind=0）。[専用CI #42385726717](https://github.com/hat47x/tei-platform/actions/runs/42385726717)の`bounded-design-context` Jobは **completed/success**。既存7+2+7の16条件に加え、**7件の必須Rule欠落・Kind/Role/Path・重複・Loader未読込の拒否**と、**非必須Subject削除を過剰拒否しない1対照**を検査した。公開合成Artifact `tei-issue-test-ability` ID `13579711391`（保存時点expired=false、保持条件はActions側）へ結果を出力。`requiredValidationSlice.status=bounded-present` は**この公開固定3件だけ**。結果の`boundedReference=partial`、`testExecutionAdmission=exploration-only`、`qualifiedTestGeneration=false`、C1 unmeasuredは維持する。新CLI API・新Parser・Owner意味受理・外部Tool受領は実証していない。

**失敗を利用した学習**：先行[CI #42385502023](https://github.com/hat47x/tei-platform/actions/runs/42385502023)は、subjectモードのSource Indexを全件参照と誤用して `wrong-mode` とDefinition欠落を観測した。これはProductのRequired Rule欠落ではなく**試験用ReaderのQuery mode不足**。要求集合の不備／検索モードの誤り／Source実欠落／結果の実受領欠落を、一つの「未カバーRule」に合成しない。

**並行Draftの所有整理**：[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324) は一Ruleの`inspect-validation`投影とSource Anchor/Canonical属性まで照合する**別方式の候補**で、同じ`context_gate.py`とWorkflow/計画を変更している。現時点の[#42385502023](https://github.com/hat47x/tei-platform/actions/runs/42385502023)は#3326の初回FAILであり、#3324のCI結果ではない。**#3324のexact HEAD `4f4b114aa826b1a603044c5eef34407f536f186d` はBranchとidentical、[専用CI #42385606408](https://github.com/hat47x/tei-platform/actions/runs/42385606408)と[Repository Health #42385606448](https://github.com/hat47x/tei-platform/actions/runs/42385606448)がcompleted/success**。ただし1 RuleのInspect/Attribute・Relation観測と#3326の3 RuleのIndex検査は異なる局所Claimで、両者のCIを統合済み製品の証拠へ足し合わせない。正式一本化の比較基準は§23.1。`#3326`の実際に成功した**3 Rule+index mode**をP-1b局所基準として評価し、`#3324`の単一Rule Inspect/属性照合に不足する差分がある場合だけ移植する。一つのConsumer・一つの独立Required組に収束させ、別々の競合ファイルをそのまま並行mergeしない。どちらのDraftも正式Owner採用やmain上の完成機能とは主張しない。

### 22.2 次のP-1b受入は「同一時点」と「実消費」を一段だけ深める

今の10応答は単一Workflow内で**順次**呼び出した別Processの観測であり、各JSON fingerprintが一致しても全Readが同一原子的Snapshotから来た証拠ではない。TEI開発者Contextの[Draft #3176](https://github.com/hat47x/tei-platform/pull/3176) CC-14は、個々のSHAが正しくても共同成立しない`(T0,C1)`、CC-15はPath検証後の差替え、CC-16はDiscoverしたTestと実`go test -json`の結果を区別した**限定Go Characterization**。さらにCC-17はTest SourceだけをA→B→Aに変更し、旧PASS→変更中FAIL→再PASSを実Goイベントで観測したもの（[Focused CI #42388764022](https://github.com/hat47x/tei-platform/actions/runs/42388764022) completed/success）。**旧PASSの現在資格が履歴を越えて保存される証拠ではない**。これらをProduct Readinessへ無条件に転用せず、次の同一Workspace/Ruleの反例に縮約する。

| 一段先の到達性 | 最小の独立Expected | 失敗時の表現 |
| --- | --- | --- |
| **Query mode / Required** | `subject`で無いDefinitionは`index`で得られるが、必要3 RuleはCatalog/Index/Loaderに同版で存在し、非必須の省略は許す | wrong-modeとrequired-missingを分離 |
| **共同Read cohort** | Manifest/Source/Index/Catalog/Inspect/Ruleの実Consumerの読取世代が混合なら、個別SHA正でも同じClosureとして資格化しない。Query中にSourceをA→B→Aへ変えても共同世代の履歴は未観測のまま | cohort-unknown / stale、再取得条件を説明 |
| **Parser/Meaning/Rule実Bind** | Test-onlyのRule ID固定値と`readByLoader:true`だけでProduct Parsed/Bound/Acceptedを称さず、当該Validationの実Evaluatorへ届いたSource/RoleとFunction/Relation依存を既存Productで確認 | parser-or-binding-unqualified |
| **Test実行** | DiscoveryのTest名・Source集合と、指定Profileで現に走ったTest ID/Result/Pathが一致。exit0/no Testや別TestのRedを当該Ruleの判別力に加算しない | selected-but-not-executed / not-target-reached |
| **Delivery/Disclosure** | 実Harness受信側のRequest/Reply count、最終part/ordered-end、許可Scope/実Sinkが当該Attemptで照合済み | delivery-unverified / ineligible。Tool成功でも外部比較に加算しない |

原子的Snapshot、現在Grant、Meaning受理に対しては、**今回新たな実Product Gateが成功したという主張をしない**。受入不可な場合の「どのSource/Rule/Replyが追加で必要か」を分かる形で出すことが、生成AI用の最小指示UXと直接つながる（#3262/#2514/#2018）。

### 22.3 他チャットの「実際に壊れ得る」実行端を、Coverage母集団へ接続

- **Operation現在性**：[merged #3325](https://github.com/hat47x/tei-platform/pull/3325)のread-only参照実装は期待WorldFingerprint/ModelRevisionを与えれば`current/stale/unchecked`を区別し、属性変更では`basis`単独が不足することを検証済み。ただし既定の`tei operations`/Invoker/Current Grantへ未接続。静的Graph/CRUD/Functionが存在することを実Dispatch Coverageへ加算しない。
- **UIのepoch**：[Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)はWorkbench A→B→Aで旧Inspectの成功/失敗を採用しない局所Node回帰。これはUI Responseの可視資格であり、旧Host実行の中断、Context共同Read、Effect取消を意味しない。
- **Command効果不明**：[Draft #3328](https://github.com/hat47x/tei-platform/pull/3328)はExecutor後のRecorder/Stage書込み失敗を`EffectUnknown`へ伝播し、`MayReplay`と実Dispatchの食違いを修正する案。新Goテスト・exact-head CI・実PG隔離の成功は別検証。業務拒否と副作用不明/再送安全をTest Resultの異なる状態として扱う。
- **Build成功とSource Closure**：[merged #3331](https://github.com/hat47x/tei-platform/pull/3331)は旧X1証拠文書の文字化け重複を削除しただけで、記録値`buildSucceeded=true`でも`sourceClosureState=rejected` / `artifactProducedDespiteUnqualifiedClosure=true`を維持。局所Build exit0からG4/Source Closure/Artifact現在資格を推測しない。
- **業務Roleの実消費**：[merged #3318](https://github.com/hat47x/tei-platform/pull/3318)は休日・税率・為替・採番のRole付き例を既存Binder/Validationへ接続し、行を**テストが直接与える**限定Go試験を持つ。型/Role宣言≠実Provider行のValues/Precision/NULL/時間有効性や完全Readであり、同じObligationモデルで不足として提示する。
- **Graph/CRUD**：[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)の複数Canvas/parallel edge/CRUDは合成Fixtureの一貫性、実TEI Loader/Graph/SHO両Consumer/独立Expectedは未達。表示OccurrenceはSubject、業務Relation、実Effectのいずれのカバレッジ分母とも同一視しない。

### 22.4 独立の横断反例（TT-89〜TT-97：Case正本は各担当へ割り当て）

| ID | 独立に変える点 | 正しいOutcome / Owner |
| --- | --- | --- |
| TT-89 | `source-index` subject/index modeを間違え、同じWorld FPでValidationが一覧に出ない | wrong-query-mode。Rule実欠落と誤分類しない／#3262 |
| TT-90 | 全10 Reply JSON正常・個別SHA正、Source観測世代だけ混合 | cohort-unknown。同一Snapshot qualifiedは不可／#3262/#2533 |
| TT-91 | 必須3 Ruleのうち一つだけ欠落、非必須Itemを一件削除した場合と対照 | Required欠落はblocked、非必須欠落はbounded partial維持／#3262/#2514（#3326で限定実行済み） |
| TT-92 | Catalog/Loader indexは一致、Product Rule Parser/Bindが未観測 | Source tracedだがmeaning/binding unqualified／#3262/#2506 |
| TT-93 | `go test` exit0だが選定Testは0件／別TestだけRed | 選定Caseのexecuted/killはfalseまたはunknown／#2514/#2533 |
| TT-94 | Operationは同ID・`basis`一致でも属性が変化、期待World/ModelRevなし | `unchecked`または`stale`を区別、actual Invoke/Grantは未受理／#951/#2514 |
| TT-95 | UI A→B→Aに旧成功Replyが到着、別の旧Commandに副作用が残る | 画面採用拒否と効果/取消/Grantを別判定／#1824/#1174 |
| TT-96 | Executor成功後Recorder失敗／Command Stage不明、旧Receiptだけあり | EffectUnknown、再送安全・Resource現在値は未観測／#1174/#1163 |
| TT-97 | Build artifact生成exit0、独立Source閉包がrejected | Artifact存在はBuild局所事実のみ。Self-Build/G4資格はunqualified／#2831/#2514 |

TT-89/90/92〜97は**この横断結合を新たに実行したわけではない**。TT-91の限定ルール差集合は#3326のPR専用CIにおいて7 reject+1 negative controlを観測済み。旧Evidence/別Issueの局所テストを新しいE2E PASSへ合算しない。

### 22.5 導入順・担当・ブランチ統合の停止線

1. **P0（現在）**：#3262のTT-56実装候補#3326と#3324を**一つへ収束**し、#3326実CIのexact HEAD結果（7拒否/1非過剰拒否）を元に、差分として必要なInspect Canonical/Anchorだけ取り込むか判断。実装の二重Core化や別Rule正本を避け、main追随と統合後main Runは別検収。
2. **P1（次の最小実装）**：3 Ruleのうち1 Ruleについて、実Loader→Parser role→実Evaluator Targetと、その同Attemptの生成済Pinned Scenarioの接続（または明示unknown）を観測する。CC-14/15の共同Read/Path差替えを、必要なら同TestのNegativesとして追加。既存`context_gate.py`の`partial`を全製品`qualified`へ変更しない。
3. **P1（T1-B）**：観測・開示資格を満たすContext上で、事前未提示のObligationを1件、生成AIが提出→独立Reviewer→#1180 Scenario→#2514 Expected/Target Mutant→#1163 Evidenceの既存経路で判別。既出行番号0や公開済みRequired3件の再提案は新規発見に加算しない。
4. **P2（横断拡張）**：実Provider/Functionの接続#2586、Command EffectUnknown#1174、Operation drift#951、UI epoch#1824、BOM exact tuple#3267、三視点Graph/CRUD#3263、Self-Build#2831は必要な一辺ずつ追加。C0/C1、業務Obligation、Mutant到達、NFR/Effect、実Consumer/Deliveryは独立分母にする。

**今回の資格**：#3326 exact HEADのGitHub Actions局所Jobを照合し、既存#3176/#3325/#3327/#3328/#3331/#3318/#3263の限界を新たな依存順と横断反例へ整理した**設計追補**。本Draftの追記コミットに新Go/CLI/Browser/PG/H2/H3の実行は伴わない。#3324/#3326のmain統合、T1-B・実Provider・G4・Owner Meaning Acceptanceは未成立。新Runner、Resolver、Host、Authority、第二Scenario/Oracle正本を追加しない。

<!-- tei-test-c1-and-tt56-convergence-20261108 -->
## 23. 2026-11-08：TT-56二実装の収束・Go C1計測・T1-B実験の着手条件

### 23.1 現行PRとexact-head実証の再点検

| Draft | 検証方法 | 実証の状態 | 独立した価値／不足 |
| --- | --- | --- | --- |
| [#3324](https://github.com/hat47x/tei-platform/pull/3324) | 既存CLI inspect-validationを追加。1 RuleのCanonical positive/attribute、Source Definition Anchor、CatalogのRelationも必要とする。正常JSONでRule/Relationなど8件の欠落・曖昧を拒否 | HEAD 4f4b114a、[read-only dogfood #42385606408](https://github.com/hat47x/tei-platform/actions/runs/42385606408)・Health #42385606448・依存確認 #42385606458はsuccess | Ruleの意味上の限定属性とRelationを照合。一方、別Rule 2件のrequired completeness、必須外省略を許容する対照はこの8件では直接証明しない |
| [#3326](https://github.com/hat47x/tei-platform/pull/3326) | 既存CLI source-indexの全件modeを追加。事前固定した3 ValidationのCatalog/Definition Role/Loaderを結合。7件rejectと、Required外要素の省略はpartialを維持する1件non-overreject | HEAD c2d2da10、[read-only dogfood #42385726717](https://github.com/hat47x/tei-platform/actions/runs/42385726717)・Health #42385726733・依存確認 #42385726722はsuccess | Required集合の複数Rule間欠落と過剰拒否防止。一方、個別Rule inspect CanonicalとRelation検証は#3324とは異なる |

いずれも実CLIが返したJSONの試験用コピーに対する限定負例であり、実Loaderを壊した実業務End-to-End／全Functionの正本性・現在Disclosure・AIの新Case発見ではない。CI成功は各PRのexact HEADにのみ帰属させ、2PRの結果を実装済み一Consumerの証拠として合算しない。

**収束の推奨**：#3262が独立Requiredを一つだけ固定し、ひとつの既存context_gate.py・ひとつの専用CIへ収束させる。優先は (a) 3 Rule集合・Required外non-overrejectを満たす検査、(b) 1 RuleのCanonical属性とRelationの追加確認が当該用途に必要かを独立Expectedで判断、(c) subject inspect / full-indexのReply byte数・受信完全性・許可Audienceを計測して最小Reader選択、の順。追加CLIを無条件に11種へ増やさない。全件IndexにはSource存在やSecretの開示面が増える可能性があるため、実Consumerの許可Scopeを測定せず外部Harnessへ公開しない。片方のDraftを残すとしても、もう一方は差分を整理して閉じ、二つの恒久的Test Consumerを運用しない。

### 23.2 Go C0からC1を導出しない実装指針

Goの標準Go coverは**statement/basic-block相当の実行記録**であり、C1の独立した分岐Outcomeの完全計測ではない。Go公式のbuild-cover（Go 1.20以降のbinary integration coverage）はCLIアプリの実測範囲を広げられるが、出力の百分率はstatementsである。Go公式のcover説明では複合論理式内側の&&/||は直接probeしない。この違いを前提に以下の三経路を分離する。

| レーン | 具体策（実装前候補） | 評価できるClaim | 禁止する誤認 |
| --- | --- | --- | --- |
| G0 Go C0 | 同じexact Head/Buildでgo test -coverprofile、必要ならgo build -cover + GOCOVERDIRで生成済実CLIを実行。go tool covdata / coverでPackage/実行対象を明示 | そのinstrumented Go Source集合のstatement coverage | 同じTargetでもGo C1や全Source/全Nested Moduleの分母は保証しない |
| G1 Go C1 | 選択した小Packageに対し、独立に列挙したif true/false、switch全case/default、guard拒否/通過、early return、0/1回loop等のOutcomeを、別Instrumenterまたは検査可能な限定イベントで実観測。Static ASTカウントとRuntime TraceのIdentity・Instrumentation版を突合 | その列挙Scopeにおける実branch-outcome coverage。Instrumenter自身の誤計測/最適化差を負例で評価 | Statementの下流実行、テストのAssertion、Rule正常/拒否2件だけでGo C1を推認しない |
| M1 Model/Decision | Accepted Rule/HTTP Decision Graphの条件・Node/Edge/Guard・Client ReactionをScenario Step→typed Result→Trace/Observerに接続し、宣言Edgeと実際の到達を区別 | Graphの当該Edge/Decision Outcome coverage | Graph Edge coverageをGo C1に足さない。HTTP Statusやexit codeだけで実Guard到達とは言わない |

G1の最小比較対象は同一ifのtrue/false、false側が空のif、ifの&&内側未実行、switch default、早期returnの5種を含む一小Package。静的な分岐IDの安定性（同一Source・変更前後）、実際の未到達を検出できること、各入口で同じDecisionが相関できること、Instrumenter変更による副作用・Source位置の崩れを評価する。専用計測器を即Coreに採用しない。第三者のinstrumenterを評価するとしても版/Toolchain/License/生成Diffを固定し、Go標準coverageを機能上書きしない。

**率の記録**：対象Build/Instrumenter/Scope/Excludedと分母の出典を固定。Required BranchがunknownならC1率はunknown。既知部分のみ報告する場合は「bounded branch outcomes m/n」とし、全製品C1の百分率を名乗らない。真偽値の分岐網羅と各原子条件の独立効果（MC/DC）も別の指標。

### 23.3 Blind T1-Bの実験手順を4条件の比較に固定

独立Evaluatorが公開Sourceから妥当なMeaning・合法Baseline・許可Contextを定め、まだ生成者に提示していないObligation（正・負の期待とMutant target）を保管する。Source自体の意味を隠して解けない課題にはしない。生成者はExpected/評価対象Mutant/隠れたGapリストを読めない。今までのline_number=0/#3017を再利用した場合は新規発見に加算しない。

| 実験条件 | 生成者が受け取るもの | この比較が示すこと |
| --- | --- | --- |
| B0 | 許可されたRaw設計情報・CLI Discovery、追加の推薦なし | 現行ツールだけでのGap診断能力 |
| B1 | 同じRevision・Budget・Allowed Scopeにおける構造化Design ContextのPartial/Unknownを含む回答 | Context整形による差 |
| B2 | B1に独立Required/Gap理由の**開示可能な部分**と候補Pattern推薦を追加 | 生成支援による改善余地。隠れた正答・Mutantは非開示 |
| BX | 必須Reply欠落／別版混入／Scope外データの追加 | 正しい挙動は提案保留・ineligible・開示拒否。失敗と採点しない |

各Trialの受入条件：Tool全Replyの受領・終了marker／SourceとTool/Build版一致、出力からの秘匿情報流出なし、Candidateが既存Caseと意味的に異なる、正常対照がPASS、目的の負例が対象Rule/Guardを識別、実Target到達、独立Expectedとの照合、可能なら当該Caseが実Mutantをkill。適格Trial間だけで成功率・false-qualified率・Tool呼出回数・Token・実時間・追加されたCoverageの**局所差分**を比較する。B0〜B2が同条件で再現できない場合は比較不可。Trialが1件ならProof-of-Mechanismであり統計的優位ではない。

### 23.4 実装順序・担当・終結条件

1. **直近**：§23.1の二Draftの対象HEAD証拠を確定し、#3262に一本化方針を渡す。Test用一時JSONの差分判定を共通化してもSource／Reader／Runnerの第二正本を増やさない。Issue #3223のMarkdown escaped-backtick事故は本Branch上で修正し、修正後のHealthを別Runで確認する。
2. **P0**：#2514へC0（公式Instrumented CLI）とG1の局所C1を分けて引き渡す。Go計測の非対応を「全体C1=0%」とせずunmeasuredとする。最初のC1候補はRule Evaluator又は小さなHTTP Decision Handlerのうち、独立Outcome/負例を用意できる小Package一つ。
3. **P0**：#3273/#1180/#2509を既存Runner/Scenario/Skillで用い、一つの真の未知Obligationを別Evaluatorで事前登録。実Harnessの受信・開示が不適格なら内側の限定CLIで先行し、外部H3と同等のTrialと呼ばない。
4. **P1**：最初の局所T1-B結果をObligation差集合（required、received、proposed、reviewed、executed、independently-qualified、currently-usable）へ反映。必要なSource/Meaning/Target/Grantが欠落した場合は「次に何を取得すべきか」をテスト推薦の成果として記録する。
5. **P2**：M1のHTTP例外Graph/CLI/Web、Plugin/Trait/BOM 2×2、実Provider Read、Function/Businessの横断観測へ同じ検証原理を展開するが、他レーンの完了を最初のT1-BのHard Dependencyにしない。

**本節の位置付け**：#3324/#3326のCI成功、#3223のMarkdown原因を実GitHubで観測した結果にもとづく検証手順と判断案。新Code C1 instrumenter、Blind T1-Bの実行、正式Source/Scenario/Owner意味受理・全実Target・MCP/H3資格はNOT RUN。既存Gateの資格を自動昇格させない。

<!-- tei-test-branch-obligation-scoped-coverage-20261108 -->
## 24. 2026-11-08：業務拒否・技術失敗の実Go分岐義務と、per-test Coverageの資格境界

### 24.1 文書上のCoverage Definitionを実Evaluatorの10条件へ接続

[Draft PR #3351](https://github.com/hat47x/tei-platform/pull/3351)（初回Code HEAD `4d45124218e6311df1759ccf879ee1749aca5243`）に、既存`internal/semantic/validationeval.Evaluate`だけを対象とするtest-only `error_branch_obligations_test.go`を追加した。第二Runner/Host/Oracleを新設せず、既存`go test`で実行する。

| 独立Obligation | テスト入力（test-only Source） | 必要なExpected | 実測の資格 |
| --- | --- | --- | --- |
| VE-01 Source未指定 | nil Source | empty Result + Go error、Source callなし | 結果の型付けのみ |
| VE-02 数値取得エラー | `Number`がsentinel error | empty Result + wrapped error、Number call 1 | Source errorを業務拒否に変換しない |
| VE-03 数値未取得 | `Number`が`nil,nil` | empty Result + Go error、Number call 1 | Missing Valueと零値の混同禁止 |
| VE-04 正・ゼロ | `Number`が1/2または0 | 正= satisfied, 零=not-satisfied、errorなし | 同じRule内の正常／業務拒否対照 |
| VE-05 Relation取得エラー | `RelationKeys`がsentinel error | empty Result + wrapped error、Relation call 1 | Partial/失敗を空集合の一意性へ混入しない |
| VE-06 Relation重複・一意・空 | `RelationKeys`が3条件 | 重複=not-satisfied、他=satisfied、errorなし | test-onlyの明示的な完全Sliceでの結末に限定 |
| VE-07 unsupported kind | Rule Kindに未知値 | empty Result + Go error、Source callなし | unsupportedのsilent fallback禁止 |

具体的Go testは10 subtestsで、成功の判定は単なるexit0ではなく、`Result`、`error`の有無・`errors.Is`と実際に呼んだSource operation／回数を区別する。業務拒否は処理できた結果であり、業務結果が得られない技術的失敗とは別である。部分集合やProvider不完全性の一般論はこのtest doubleでは保証されない。

**Code C0/C1主張**：最初のDraftの実装・保存だけではGo Test PASSの証拠なし。Go Test実行結果が得られても、10個のScenario/独立Result条件を確認した範囲でのみ合格とし、`go test -cover`はstatementのC0相当、C1の分岐ごとの実観測は別。1 Test名に10 subtestsがあることは独立した10 Required Ruleや全Conditionの網羅を意味しない。正式なOwner Meaning Acceptance/実DB/Host/外部Harness/現在開示資格は別。

### 24.2 GoのCoverageツールを「観測の種類」と「実消費対象」で比較する

追加調査は一次情報に限定する。Go標準 https://go.dev/doc/build-cover 、[gobco](https://github.com/rillig/gobco)、[Tobari](https://github.com/goccy/tobari)。

| 候補 | 何を追加観測できるか | 向かないClaim・導入時に独立検査すること |
| --- | --- | --- |
| Go標準`go test -coverprofile`、`go build -cover`、`GOCOVERDIR` | Unit/Integration CLIで実行したstatement/blockの対象範囲・実際のGo Build/Package | Code C1ではない。Build Tags、nested modules、`-coverpkg`、除外条件と各Runを束縛 |
| gobco | 比較・論理条件の真偽別の**condition coverage**。標準Go coverの補助（BSD-2-Clause） | C1全Branch・MC/DCと同義でない。未使用Functionやselectの検出がないので別Required集合が必要。現Go Toolchain適合を先に試す |
| Tobari | TestまたはRequest単位のScoped Coverage、Test×Blockの到達と重なりを観測（MIT） | Scopeの分母は到達Functionの依存から導く候補であり、独立Requirement/Source Requiredの分母ではない。Shared worker goroutineでは別requestのBlockが混ざる既知制限あり。Runtime/Go/Library/CLIの同版Pinと正しいScopeを検証 |
| Model Decision Trace | TEIのAccepted Rule/Guard、HTTP Response Graphの実実行EdgeとClient ReactionをScenarioへ束縛 | Go C1に加算不可。宣言Graphだけ、Mock Resultだけ、外部Tool全Reply未受領なら実到達資格なし |

**採用判断を早めない**：既存Go標準が第一基線。gobcoはCondition Coverage PoC、TobariはAgentに特に有用な**Test→Block対応/冗長候補探索**のPoCである。両者ともC1完全実装の代用品とはせず、Backend対応版・ライセンス・Instrumentation副作用・ビルド閉包/排他性・Source改版・CI時間・未到達の偽陽性/偽陰性を計測してからPlugin候補とする。Test名単位のScopeはProductの同一Function/Operation/Actor/Targetに正しく対応しているか検査しない限り、業務Scenario単位のCoverageにならない。

**重要な非推論**：同じBlock集合を実行するテストが二つあっても、Independent Expected、Error Outcome、Temporal/Authorization/NULL、Source Identity、負例判別力が異なれば片方を削除してはいけない。`>95% overlap`はReview候補の推薦閾値になり得ても、削除資格にはならない。Tobariの依存到達可能性から推定したdenominatorも、#3262の独立Requiredや非出力依存・業務Function全件を自動補完しない。

### 24.3 最小PoCと偽陽性対照

- **CB-01**：#3351の10条件を同一Go Head・Toolchain・選択Packageで`go test -count=1 -run TestEvaluationErrorsAreDistinctFromBusinessRejections -v ./internal/semantic/validationeval`。named subtestの列挙、各結果、Go exit、Target SHAを証拠化。1 subtestを落としたときに計測上の差を観測。
- **CB-02**：Go標準`-coverprofile`で、追加テスト単独／既存Test suiteを別実行して対象Packageのstatement差分を比較。実Sourceの分母・`coverpkg`・編集前後を明記。
- **CB-03**：gobco/Tobariを現Go Toolchainにpinして隔離した短期試験。Source error/undefined value/zeroの3ケースについて同一Go Blockを共有していても別Independent Expectedであることを記録。Condition・Block・Branch/Guardを別列にし、計測の非互換/過大主張を検出。
- **CB-04**：Tobariのper-test結果が同じFunction内の10 subtestsを区別できるか。別Source/Target、shared worker、カバレッジ対象外Package、Build Tagの切替と、実Run選択を区別。未観測なら`unknown`。
- **CB-05**：ミニMutation（`nil Number`を`not-satisfied`へ誤変換、Source errorを握り潰す、`relation-unique`で重複を見逃す）を**本番ブランチへ統合せず隔離**して、対象subtest自身が独立Expected違反を観測することを確認。単なるCI FailureはKillでない。

`CB-01`以外のInstrumenter評価とMutationは設計候補で、**未実行**。CB-01もPR新規コミットの初回CIを確認してから「実Go PASSED」とする。既存#3017の意図的Mutant成功を今回の新subtestsへ流用しない。

### 24.4 最短の受入順とOwner

先に#3351のexact-head実Go Test/Healthを観測して、局所Obligation Resultだけを#2514へ伝える。次にGo標準Coverageの正しい分母とTarget/Sourceを固定してCB-02を実施。gobco/Tobariは候補として#2514/#3223に比較評価し、必要と測定される場合にのみPlugin/Skill化。#3262のRequired Rule Closureと#1180のScenario SSOT、#1163のEvidence、#2509の外部Tool受信と#3274のHTTP Graphはいずれも別Ownerを維持する。T1-B盲検採点には検証用test doubleを「実Provider受理」として渡さない。

**状態**：#3351はtest-only Draft。Code C1測定器、Go per-test scoped導入、実Mutation、Blind T1-B、実DB/Host/業務Owner受理は未達。新しいEngine/Runner/Host/Oracle/Scenario正本を増やさない。


<!-- tei-test-admission-effect-selection-harness-20261108 -->
## 25. 2026-11-08：Test AdmissionをSource・選択能力・実行効果・受領の四層で判定する（横断実装計画）

> **TT-56の統合後更新（2026-11-07 16:35 UTC）**：[PR #3354](https://github.com/hat47x/tei-platform/pull/3354) はcommit `39d4debae386442c11149c7d3995bcfffa711c20` でmainへ統合された。mainの`context_gate.py` blob SHA `b9d593c7b679f7f82c3ec8b804841ee9c0c5786f` は統合候補と完全一致。旧#3324/#3326は**closed/unmerged**。PR HEAD `0ab2b99f...`の[専用CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)はsuccessだが、**統合後mainの別push CI成功は本更新時点で未確認**。下表の「統合候補」は履歴上の候補として読む。Source全閉包・原子的共同Read・T1-Bは引き続き未成立。



### 25.1 今回の差分：単に「Ruleを取得した」だけでは、実行できるテストにならない

既存§22〜24が必要Rule・Source閉包・対象Caseの判別力を扱うのに対し、本節は**生成された候補をどの実行経路へ渡してよいか**を定める。独立Obligationを増やしても、実CLI/Plugin/Hook/Hostの実際の能力・副作用・保存先が不明なら、正式なRun/外部Harnessの比較資格はない。新たなTest Runnerや中央Authorityは作らず、既存#3262のContext、#1180のScenario、#2514のExpected、#1163のEvidenceへ用途相対のread-only判定を追加する。

| 層と問うこと | 現時点で根拠のある実装 | 必要だがまだ別資格の証拠 |
| --- | --- | --- |
| **S：Source/Required**　対象Rule・Relation・Functionが必要Scopeに存在するか | #3324/#3326各CIは局所PASS。統合候補[Draft #3354](https://github.com/hat47x/tei-platform/pull/3354)は11件の既存CLI Replyを1個の`context_gate.py`で消費し、3 Rule＋1 Focused Canonical/Relationを照合。exact HEAD `0ab2b99f028a867da6681239f8b79691b9a877b4` の[専用CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962) **completed/success**：既存16＋追加16＝32件の合成判別、既存6 pinned Step＋T1-A候補1 Stepの限定実行 | #3354のmain統合は確認済み。一方、統合後mainの別push CI・原子的Read cohort/Function/Business/Meaning全閉包は未証明 |
| **C：Capability/Invocation**　選択した機能が実際に到達するか | [merged #3337](https://github.com/hat47x/tei-platform/pull/3337)はWorkspaceの宣言能力と環境選択を参照一覧化。12能力中6を当時の算定基準でunselectedと報告 | [Draft #3350](https://github.com/hat47x/tei-platform/pull/3350)は既存`SupportsDatabaseProvider`を使い技術名→能力対応を是正すると選択なしは4件。いずれも**選択≠実Plugin Consumer Load/Invoke/Grant**。unselected≠欠陥 |
| **E：Effect/Outcome**　どのFlag/Entryで何を変更し、どのResultになるか | [merged #3333](https://github.com/hat47x/tei-platform/pull/3333)は`tei commands`の`writeFlags`を実ファイル生成で検証。[merged #3340](https://github.com/hat47x/tei-platform/pull/3340)は`-h`とDiscoveryが同じ引数値/既定値を参照。#3351 Draftは`validationeval.Evaluate`の10件で業務不成立と技術的失敗を分離 | Command全argv・明示Output・OS実書込・現在Grant/EffectUnknown・実Providerの独立Read。read-onlyという宣言や`-execute`という語だけで安全/禁止を判定しない |
| **R：Receipt/Delivery**　Observerは実結果を完全に受け取ったか | [Draft #3309](https://github.com/hat47x/tei-platform/pull/3309)は合成canaryを使いDSH headless stdoutと永続Session Log双方への保存を限定実測。連結zstd frameの一部だけ展開するとcanaryを見落とす | 実外部Harnessの全Reply/End/Actor/Scope、全SinkのDisclosure/保持、Host/Resource Receipt。内部CLI成功やLogへの一回の出現を完全配送・安全保持としない |

S/C/E/Rの全条件を一個の`qualified` booleanへ畳まない。まず`known`・`not-loaded`・`unsupported`・`not-selected`・`not-invoked`・`effect-unknown`・`delivery-unverified`等を、**対象とAudienceに必要な場合に限って**返す。例えばアプリ実行に不要な`editor-intelligence`が未選択でも、その業務Runの失敗へ転用しない。逆にRunnerが`view -execute -operation-log`を実行するなら、Discoveryのread-only広告だけではFile Effectなしを保証できない。

### 25.2 空集合・部分集合・エラーを業務上の完了へ短絡させない

- [Draft #3351](https://github.com/hat47x/tei-platform/pull/3351)の限定Go Testでは`Number() -> nil,nil`は**技術的に評価不能**、`Number() -> 0,nil`は**業務Rule不成立**、`RelationKeys() -> [],nil`は**テストが明示した完全Sliceに限り**一意条件成立。実Providerで`partial`や`not-observed`の返却を`[],nil`とみなし「全て一意」とすることを禁止する。
- [Draft #3353](https://github.com/hat47x/tei-platform/pull/3353)が扱う`EmptyPolicyHandoff`は、対象が空でも`EmptyJustified`と`HandoffConfirmed`を別証拠として要求する修正案。空集合証明・受領確認・実送信効果は別。コード案のCI結果・実Host受領は未確認。Completion/WorkflowのRuleを一般の`validationeval`へ混ぜない。
- [merged #3336](https://github.com/hat47x/tei-platform/pull/3336)はSubprocess Buildで`invoker`・`invoked`実行ファイルdigest/exitと実ディスクArtifact digest照合を記録する限定実証。**Stage1生成物が駆動したL3ではない**。旧Source Closure拒否が残るため、Build成功だけでTestを`currently-qualified`にしない。
- [Draft #3348](https://github.com/hat47x/tei-platform/pull/3348)はHook Phase順序と途中停止を`hook.Runner`直接試験で定めたが、CLI/Studio実入口・複数Plugin同位相の合成・Transaction/Effect境界は未検証。Source上のHook宣言数で実行Path Coverageを満たしたとしない。

### 25.3 新たな受入Caseは増やしすぎず、既存Ownerへ分配（TT-98〜TT-104）

| ケース | 変える対象と独立Expected | 対応先 |
| --- | --- | --- |
| **TT-98** | #3354で3 RuleとFocused Inspectが互いに整合する正常系、片方だけRule/Relation欠落する負例。同一`context_gate.py`実Runとexact HEADの証拠があるまで`partial` | #3262/#2514 |
| **TT-99** | 同じPluginが環境で選択されても実ConsumerがLoad/Invoke未実行。選択`present`と実Call`not-observed`を同時に返す。選択されない他環境の候補を欠陥と誤分類しない | #3279/#1251/#3267 |
| **TT-100** | `database-instance`の技術名を能力IDとして文字列比較してproviderをunselectedに誤分類。既存Productの`SupportsDatabaseProvider`で解決し、不一致・複数候補も区別 | #3279/#2514（#3350はDraft） |
| **TT-101** | CLI Discovery`effect=read-only`のまま、実argvに`-operation-log`等を与えた実ファイル書込み。名前付きOutputと意図しない書込み/外部送出を分離して判定 | #3182/#2515/#2605 |
| **TT-102** | `Number(nil,nil)` / Zero / RelationKeys error / 空だが不完全Sliceを反例化。Errorを業務拒否/成功へ、空を不存在/Completionへ変換しない | #2514/#2506/#1174 |
| **TT-103** | Hook/Operationを宣言・順序付けるが実Entryへ未接続。Declaredとexecuted Pathを分ける。途中Warn/Refuseで後続Phaseを実行したと推定しない | #951/#1180/#1163 |
| **TT-104** | DSH Tool結果を連結Frame全体ではなく先頭Frameだけ走査して「canary無し」と報告。実Sessionの保存済み結果と適格Disclosureの判断を分離する | #2515/#2509 |

### 25.4 MVPの作業順と停止条件（既存パイプラインに追加する最小横断）

1. **TT-56の一本化完了後の検収**：#3354はmain統合済み。11実CLI Reply・Context合成32条件・既存6＋T1-A第7 Stepの限定PR CIは成功。次は統合後mainの別CIと、Reader追加時のTool Budget/外部Disclosureの独立実測。旧#3324/#3326の別Branch PASSを新たな製品E2Eへ加算しない。
2. **実Run Admissionを一件だけ**：公開合成Rule `line_number_positive`で、必要Rule ref/Target/Source/CLI binary/全argv/実File Effect/Expected/Receiverを同じAttemptへ束縛。未知Source/Grant/Plugin実Loadは`unknown`として局所実験は継続可、**外部や製品全体のqualifiedには昇格しない**。
3. **実Provider接続の一辺**：#2586実ProviderからのQuery/Read Sliceを、`validationeval`の空/失敗/不成立と結合する前に`complete/partial/unavailable`を明示。#3351のtest double結果を実DBへ転用しない。
4. **T1-Bを独立実験**：事前未提示Rule Gapと秘匿Holdoutを独立Reviewerが固定し、既存Scenario/CLIで生成Agentの提案→Target到達→正負判別を測る。S/C/E/Rの必要証拠を満たさないTrialは不適格、C1未計測のままならC1改善を主張しない。
5. **計画反映・経済性**：#2018には「不足の診断が先に正しく出たか」「次に実行すべき合法な取得/試験を提案できたか」、Tool回数/Response bytes/外部受領/CI再実行費用を記録する。単なる候補件数、既存CI合格やPlugin選択率をMVPのKPIにしない。

**実績の停止線**：#3333/#3340/#3336/#3337および#3354のmain統合は各限定Capabilityの進展であり、#3351/#3350/#3353/#3348の未統合Draftの試験成功やT1-B製品資格ではない。本節は当該PR/Sourceの照合に基づく受入計画の更新であり、新しいGo/CLI/PG/外部H3/GUI/実Plugin複合試験、Owner Meaning Acceptanceは本追補自体ではNOT RUN。第二Runner・意味正本・認可系を作らない。

<!-- tei-test-epistemic-coverage-projection-20261108 -->
## 26. 2026-11-08 横断深化：不存在判断の適格性、検証義務の標準構造投影、二世界差分テスト

### 26.1 他レーンから一般化した問題――表示されない／返らない／実行されない ≠ 存在しない

三視点Model/CRUDと複数Canvas (#3263)、Function ContextのPartial Read/Join/Grain (#2506/#2586)、CLI Design ContextのRule目録 (#3262)、Trait FulfillmentとVerified BOM (#3279/#3267)、SHO/外部Harnessの公開Slice/実受信 (#2516/#2509)、Cache/Buildの再利用・履歴 (#3271/#2600)を横断すると、**観測可能な部分に対象が見つからなかったことを、世界全体の不存在や未実装・未カバーと読み替える**共通の偽陰性がある。以下をテスト推薦・Coverage投影における独立Expectedとして扱う。

- **存在の正の証人**：対象のstable semantic identity、Source/Role、Scope、時点/粒度、実ProducerとConsumerを束縛した観測なら、その観測範囲の存在を示せる。ただし現在のGrantや全件・持続性を推論しない。
- **閉じた範囲での不在**：Query mode・Predicate・Source Loader/Parser・Relation/negative dependency・ページ終端・Consumer受領・Source版/共同Read cohort・許可された開示が、対象に必要な範囲で閉じているときに限り「この範囲に見つからない」と言える。業務上の不存在や削除の確定は追加の権限/実Read条件を要する。
- **未取得／不完全／未知**：部分ページ、未Loaded、索引のsubjectモード、欠損Reply、未計測分岐、未発見Scenarioはゼロや不存在でなくunknown/partial。正常JSONや一致Fingerprintでも完全にはならない。
- **非開示**：Secret/Deniedは内部の未取得状態や件数と直結させず、閲覧者向けには既存Disclosure規則が許す範囲に抑える。存在/件数・Source path・Missing IDも漏えい面である。
- **不適用**：対象に要求が適用されないという、独立したSource/Rule/Target判断が得られた場合だけnot-applicableとする。テストがない・Providerが未選択という事実だけでは不適用にしない。

これは第二のClosed World Engineや新Core enumの提案ではない。既存のSource/Meaning/Context/Scenario/Evidenceから、目的・対象・粒度・Scope相対に導出する**read-onlyの判定規則**である。

### 26.2 Test Obligationはモデルからの派生投影であって独立の意味正本ではない

対象を「要求されているか」「観測可能か」「テストされたか」の三つの集合で照合する。Test Obligation一件の表示は次の参照を持つ（実Schema未採用）。

| 対象の軸 | 参照する既存Owner | 誤って合成しないもの |
| --- | --- | --- |
| **Semantic obligation** | Accepted Requirement/Rule/Data Relation/Function/Business/Atomic Trait IDとSource Role、Cardinality・単位・Time/NULL・条件分岐。#1120/#1126/#2495/#3279 | Rule名、同名SQL、CanvasのCard ID、型名だけを同一業務意味にしない |
| **Applicable Scope / Closure** | #3262のRequired/Loaded/Parsed/Boundと関係/否定依存、Query Mode、Result Slice、Version/Scope/As-of。#2506/#1706 | 0件、NOT EXISTS、未描画辺、未選択Pluginを完全な不在へしない |
| **Test design** | #2514独立Expected/合法Baseline、#1180 Scenario Source・Candidate/採択、Patternの前提と対象Target | AI自身のExpected、T1-A既知Case、別世界のテストをT1-B新発見へ加点しない |
| **Actual execution** | 実選択CLI/Plugin Provider/Go Build/Function Entry/Provider Read・Host/Client Reactionのexact Source/Lock/Artifact/Attempt。#951/#1163/#1174/#2586 | Dispatch宣言/exit0/Cache hit/HTTP 200だけで実Rule到達を保証しない |
| **Current qualification** | Observation+独立Expectedと必要なCurrent Grant/Consumer受信、失効・Disclose/Effectの資格。#1163/#2516/#2600 | 過去のPASS・Verified BOM・Owner未受理Candidateを現業務許可にしない |

同じSubjectを二つのCanvas/表・図・SHOに表示しても、別Obligationとして重複加算しない。反対に同じ両端/ラベルの**複数Relation Edge ID**、Entity/Relation/Path occurrence/Aggregate等の異なるGrain、同名異Provider/Query/Targetは別の検証単位となる。Data構造、Function内Binding、Presentation Topology、実Action/Effectの各辺に依存がある場合はそれぞれ独立の到達義務とする。SUI的な未分類Raw Itemは保存済みというだけで意味のRequired集合に自動参入させず、未整理/後日解釈として保持する。

**必要十分条件を誤らないCoverage判定**：
1. 「特定RuleのCaseがcovered」は、当該Rule/Scope/Targetの独立Expectedと実到達・観測が資格化した場合の**局所的な正のClaim**であり、全Ruleの発見完了を要求しない。
2. 「特定Ruleは未テスト」は、対象Ruleの存在・適用性と、そのScopeのScenario実行目録の完全性を別途確認して初めて主張可能。目録がpartialなら「現在受領したScenarioに一致Caseなし」と表現する。
3. 「必要なRule/Relationがソースで欠落」は、#3324/#3326の独立Requiredと限定CLI投影の差集合から診断できるが、完全な業務世界でRuleが存在しない証明ではない。
4. 「全要件がカバー済み」は、独立Required宇宙・依存/Target/Consumer範囲・実観測/独立Expectedの完全閉包がなければ不可。数値分母を勝手にゼロにしない。
5. 現在Grant/Audienceが未知または拒否なら、内容が正しくてもそのConsumer向けの「完全なCoverage Report」を公開しない。

### 26.3 推薦の停止条件と、状況に応じた次の最小操作

初めに「何をテストするか」を網羅生成するより、次の分岐を既存CLI/Skillのread-only Resultから提案する。

| 観測事実 | 先に推奨すること | 生成候補の資格 |
| --- | --- | --- |
| 必要Source/RuleがManifestにあるがLoader/Parser未確認 | 実Loader/ParserとSource Role、Full/Subject Index modeの確認 | 欠損の説明まで。Test Patternの自動採用不可 |
| Relation/Queryがpartial、NOT EXISTSが見かけ上true、結果Grainが不明 | Predicate/Scope/ページ終端/EntityとOccurrence/否定依存の観測 | 存在の局所証人は保持。不在を前提にしたExpected不可 |
| Trait mandatoryだがProvider/Manual Implementation/実Target不明 | #951/#3279のFulfillmentと実Consumer/Conformanceの不足照会 | テスト案は候補。提供済/Verifiedとは言わない |
| 同一ItemがCanvas二面にあり片面のみ見える／Graph edge非表示 | Subject/Edge identityとビューOccurrence/Sourceを独立照合 | 非表示を未カバー・不存在としない |
| 選択済Scenarioがあるが実CLI/Tool replyの完全受領なし | #1180のPinned命令・実named Test結果・#2509のReply Endを照合 | planned≠executed。全Coverage改善不可 |
| 独立Required/Expectedと実Targetが揃い、本当にCaseが未実行 | #3273/#2514の新規Test Pattern候補を提出 | T1-B Blind Reviewerへ引渡し可 |

QueryやGraphを扱う「不在に関するTest Pattern」は、Positive WitnessとNegative Witnessを分ける。たとえばD1=2、D2=3のJOIN 6行を6実体と数えず、D2の1/3ページを「残り0」としない。意図的に1要素欠落させた場合も、同じReaderが検証済みのRequiredだけに違反を示し、範囲外要素の変更には過剰拒否しない。

### 26.4 同一Subjectの二投影と二世界比較による独立検証

**一つの意味Sourceを、複数の異質なConsumerへ投影**するときのMetamorphicな期待値を整理する。Renderer名一致やJSONの見た目一致ではなく、何を不変にすべきかを先に定め、変更軸一つずつを独立に変える。これはテスト生成候補の型であり、OwnerのScenario/Oracle正本を複製しない。

| 比較 | 期待する不変／変化 | 非主張 |
| --- | --- | --- |
| 同一SubjectをCanvas A/Bへ二重配置、片側のみHidden | Semantic/Requirement ID・Obligation数は不変、View occurrence/Disclosureのみ差 | 二表示を二実体・二Ruleと数えない |
| 同じ端点のRelationをEdge ID違いで2本配置 | 多重Relationは2件を保持。片方だけ消えれば当該Edgeの差分を検出 | Graph線の描画順/レイアウトを意味としない |
| 同一QueryにPartially fetched 1/3と完全な0件を与える | 部分1件は存在証人、complete zeroだけがScope内の不在証拠 | SQL名やRaw JSON同一なら全件としない |
| 公開Slice不変で非公開Role/Item/Source件数のみ違うW0/W1 | 許可されるBody/Diagnostics/Status/図の可視差は生まない | 実Policy/Sink非干渉は未試験ならNOT PROVEN |
| Source/Provider Lock A→B→A、Artifact byteだけ同じ | 過去Evidenceは履歴として残し、現在のHost/Grantを自動復活させない | hash一致=更新履歴なし・最新許可ありではない |
| Pluginの未選択実装を追加／Provider選択だけ変更 | 実Build Closureが不変なら局所再利用余地あり。実選択Providerが変われば当該tuple資格再照合 | BOMの単体PASSを全組合せPASSにしない |
| RuntimeでEffectUnknownだがUIに成功表示 | UI表示と実Owner Commit/別Connection Readを別証拠 | UI success/Receipt受領だけでPersistedとしない |

### 26.5 新しい横断受入反例（TT-105〜TT-114、現時点は提案・NOT RUN）

| ID | 独立の対照入力 | 正しい判定／Owner |
| --- | --- | --- |
| TT-105 | Required Ruleが存在、受領Scenario目録は途中までで一致Caseなし | unobserved-in-partial-inventory。untested確定は拒否／#3262/#1180 |
| TT-106 | 観測済Rule Caseは局所実行済みだが、全Function Rule一覧が不完全 | 対象1 Ruleのqualified観測を保持し、全Coverageはunknown／#2514 |
| TT-107 | Queryが先頭ページ1/3、NOT EXISTS/0件を推論 | scope-without-absence-proof、存在証人だけ保持／#2506/#1706 |
| TT-108 | SUI/SHO/Studio二Canvasの同一Subjectカードを二件に加算 | semantic dedup、occurrence数は別／#3263/#2497 |
| TT-109 | 同端点・同LabelのGraph Edgeが2 ID、片方を隠す | Relation Requiredは2件、非描画を削除としない／#3263/#1126 |
| TT-110 | Trait required、宣言ありだが実Provider Load/Manual Obligation欠落 | unfulfilled/unknown、Conformance/Verified BOM不可／#3279/#3267 |
| TT-111 | 非公開Role件数だけ異なる二World、許可Sliceは同じ | 非公開件数/Source path/診断を外部Consumerへ漏らさない／#2516/#2515 |
| TT-112 | 同一名称のSQLを別Provider/Bindで実行しE/W/C結果を一つのCoverageへ合算 | 別Source/Target/Attemptの結果は別Claim／#2586 |
| TT-113 | Cache hitとCLI exit0、実Binary/Rule/Observer/Tool Replyが未確認 | executed/covered/reusable-qualifiedへ昇格不可／#2600/#1163 |
| TT-114 | Source A→B→Aで古いExpectedとGrantを無条件に再使用 | Content再一致は履歴や現在資格を回復しない／#2514/#1174 |

### 26.6 現実的な実施順とClaim ceiling

- **P0-A（既存実行）**：#3326/#3324のTT-56一つへ収束させた後、同じ3 Ruleと公開Source/同一CLIでRequired欠落・non-overreject・Query mode・Source/Index/Loader/Actual Rule Bindの区別を記録する。最初から全Data/Function/Businessを完全正本としない。
- **P0-B（最初の新しい局所判別）**：TT-105/99の「目録が不完全でも局所Coveredは成立、全体Untestedは未認定」を、既存Driver Scenario/Context Gateの試験用Consumerで再現。次にTT-107のpartial→completeを独立Expectedで観測する。既存#1180/#2514のCase命名・本体Runnerを再実装しない。
- **P1-C（AI検証）**：T1-BへはQualified Requirement範囲内の未提示Gapを一件だけ移し、B0/B1/B2/BXを実Tool受信・独立Expected/合法Baselineで評価。テストコードの提案数ではなく、真の未検証Obligationの解消・誤PASSを測定する。
- **P1-D（Plugin/BOM・二Consumer）**：同一Contract/ProviderのCLI/Studio実呼出を#1251/#951に、Trait requiredとexact 2×2 BOMを#3279/#3267に委譲。SUI/SHOの多重View・W0/W1非干渉の検査は#3263/#2516と別実測。
- **P2（実Product資格）**：Go C1局所実計測・実PG Provider/HTTP Client Reaction/Owner Effect/SHO実受領・NFRを必要なScopeごとに増やす。T1-BやPlugin D1の全完成を各々のHard Dependencyにしない。

**既存の資格境界を維持**：この節はモデル化されたObligation投影と判定負例をOwnerへ割り当てる設計差分。新しいTest Engine／第二Semantic/Scenario/Oracle／中央Registry／認可・Host/Resolver／実Plugin Bundleを作らない。#3324/#3326別々のCI success、main #3304の既知T1-A、Go局所C0相当10.1%は既存局所実績。TT-105〜107の新実行、T1-B/外部H3、全Product C0/C1・実Provider/SHO・Owner Meaning AcceptanceはNOT RUN/NOT PROVEN。

<!-- tei-test-tt56-main-merged-20261108 -->
### 26.7 最新のTT-56実装基線：二Draftの収束をmain統合として認識する

[PR #3354](https://github.com/hat47x/tei-platform/pull/3354) は2026-11-08 JSTに**merged**（merge commit 39d4debae386442c11149c7d3995bcfffa711c20）。旧[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324)と[#3326](https://github.com/hat47x/tei-platform/pull/3326)は**closed / unmerged**となった。したがって§22/§23/§25内の「一本化待ち」「#3354 Draft」という記述は**当時の履歴**とし、これから再び二実装を統合する計画として扱わない。

- 統合対象HEAD 0ab2b99f028a867da6681239f8b79691b9a877b4の[専用CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962) は completed/success。**既存11件の実CLI read-only Reply**を取得し、3 Validation RequiredのCatalog/全Index/Loader・Focused Canonical/Relation/Source照合を**1つの context_gate.py**で検査。既存16＋3 Rule差分8（7 reject/1非過剰拒否）＋Focused 8 reject＝**32件の限定合成Consumer Controls**。
- 同Runで既存6 pinned Scenario Step＋既知T1-A一時第7 Step/独立負例を維持。Go cmd/tei局所statement C0相当は**10.0%（当該HEAD）**で、以前の10.1%と単純な増減比較をしない。C1、全機能/非機能Coverage、原子的Source Snapshot、T1-B、実Provider/Host、外部H3 Tool Reply完全受領、Owner Acceptanceは別資格。
- 上の成功はPRの**exact HEADと仮統合Checkout**の試行。merge commit 39d4de... を対象とした独立push CIや新たなBinaryの結果は、同Runで確認したことにはしない。
- **次の最小P0対象を変更**：TT-56二候補統合ではなく、**TT-105/106**の「不完全Scenario目録から未テスト断定禁止」と「対象Ruleの局所Covered保持／全体Coverage unknown」を、main統合済みの1 Consumerと既存Scenario/独立Expectedの範囲内で実証する。次にTT-107の部分Query/NOT EXISTSへ拡張。T1-BとC1は独立並行。

**到達範囲**：Required 3 Validationの限定同一性照合まで。実行対象が完全に読み込まれた／実Functionに束縛された／外部へ開示してよいという上位資格は付与しない。


---

<!-- tei-test-source-and-proof-round2-20261108 -->
### 26.8 2026-11-08 第二ラウンドDogfoodingと実装境界：テスト生成の入力資格・実行経路・証拠の破れ目

**差分の性格**：§18〜26のRequired/Observed/Qualified原則を再定義せず、現行mainで新たに**実観測された不連続箇所**を既存テストOwnerへ割り振る。T1-B（未知Ruleの盲検発見）を新しい机上試験一覧ではなく、実Source Reader・実Toolと独立Expectedを使う一往復に限定する。各PRの状態は本節作成時点のGitHubで確認したものであり、今後のmainへの統合・CI結果を先取りしない。

#### 26.8.1 Sourceを渡したつもりでも、実Loaderが設計を読んでいない

[第2ラウンドDogfood #3356](https://github.com/hat47x/tei-platform/pull/3356) は実TEI CLIの再測定（Go 1.27.3 / WSL2）により、修復済みのD-1/2/4〜8と、未解決の**D-3（`view`のFingerprint）、D-9（設計文書の実Loader不消費）、D-10（JSON形状差）**を分離した。製品独自の開発用MCP Server/Commandは今回確認できず、**CLI実測・MCP NOT RUN**。[#3346](https://github.com/hat47x/tei-platform/issues/3346)、[#3345](https://github.com/hat47x/tei-platform/issues/3345)、[#3347](https://github.com/hat47x/tei-platform/issues/3347)が個別の実現Owner。

- **D-9**：`docs/`・`plan/`をWorkspace sourcesに列挙しても、Markdownは`managedFiles`へ現れるだけで`readByLoader:false`。`model catalog`/`inspect`/`trace`/`source-index`の**意味としては読めない**。Issue本文・設計Markdownの文字列を見たAgentが、TEIのAccepted Canonicalへ自動Bound済みとみなすことを禁止。一般Markdown/Raw Item/`*.tei.md`のTEI Authoring候補/既存`*.tei`は別資格（#2621/#2619）。P0はSource discoverabilityと意味解釈の有無を**実CLI応答で明示**する。Parser導入をここで勝手に決めない。
- **D-3**：`view -format json`の`semanticFingerprint`は利用Namespaceに限定した指紋で、catalog/inspect/validate等の`worldFingerprint`・`sourceIndexFingerprint`と**対象範囲が異なる**。等しくないこと自体を版漂流と判定せず、同種同Scopeの指紋が揃わない場合にのみstale/unknown。欄の追加/Schema v1は#3345の契約判断。
- **D-10**：`validate -json`の`diagnostics[]`は二重構造、`requirements.result`は配列、`model catalog`にbuiltin/stdが混在し、`plugin-lock`に独立形式版がある。単一共通`result.object`や全entries=業務宣言をConsumerが仮定すると**偽の欠損/過剰Required**を作る。実Command別Schema/Modeを正規化するのは読み手の明示責務だが、新しい第二意味正本・CLI偽MCP APIは作らない。

#### 26.8.2 「宣言した意味」「実装した作用」「実際のReader」は別の試験対象

| 新たに成立した／計画された実装証拠 | 本テスト設計へ取り込む区別・資格上限 |
| --- | --- |
| [main #3358](https://github.com/hat47x/tei-platform/pull/3358) Holiday Calendarの`ValueDomain`検証：型でValuesが宣言されている場合だけ宣言外を拒否、未宣言は**開集合**。Go局所Test 5件 | **値域未宣言≠入力値が不正／全値合法**。さらに`HolidayRowsForCalendar`への自動接続は未実装。単体の`Validate` PASSを実業務読取・型制約の自動強制へ昇格しない（#1126/#2514）。 |
| [main #3348](https://github.com/hat47x/tei-platform/pull/3348) Hook`RunSequence`位相順・Warn/Refuse停止、[Open #3359](https://github.com/hat47x/tei-platform/pull/3359) `ReadsFor`宣言属性のみに限定する実装候補 | Hookの`reads`宣言→実渡したSource値→Hook呼出→`produces`→Command/Effectは**別辺**。#3359はProviderから値を取得せずCLI/Studio/RunSequenceへ未配線。`on-failure warn`の続行や直接Runner Testを実入口の効果適合としない（#951/#1174/#2514）。 |
| [main #3357](https://github.com/hat47x/tei-platform/pull/3357) 共通JSONL Read/Append（局所Go試験6件） | 壊れた行で**前方の有効Recordとerrorを同時返却**する。呼出元がerrorを無視してprefixを「全履歴」と判定したら不適格。0600はOwner-only file permissionであって改竄不可能の証明ではない。ChainFileは別規則（#1163/#2515）。 |
| [Open #3360](https://github.com/hat47x/tei-platform/pull/3360) Stage1生成Driver自身がStage2を実駆動、[Open #3362](https://github.com/hat47x/tei-platform/pull/3362) 自称Invoker digestと実Process Imageの照合 | **限定ローカルGo Referenceの再入性**と`cmd/tei` G4は分離。`stage1 artifact == stage2 invoker`、同Child/Args、artifact一致を別判定。Self-claimの一致だけではL3を主張不可。Open PR局所Go PASSをmain/Release/G4へ加算しない（#2831/#1163）。 |
| [Open #3363](https://github.com/hat47x/tei-platform/pull/3363) 環境Binding×既存Supports*の参照Matrix。24 Bindings中22 selected、2 missing（`editor.database_workbench`） | **参照Candidate Matrix**と製品Validatorの責務を分離。現Validatorは`editor.sql_analysis`等の限定Checkであり、Reference上missing 2件は製品Rejectや実Plugin Load Failureを意味しない。selected→actual invoked/Artifact/Grantは未確認（#3279/#951/#3267）。 |
| [Open #3361](https://github.com/hat47x/tei-platform/pull/3361) Product Skill/Harness F0〜F5 | 実Source Read、Required充足、外部Tool完全受領、現在Disclosure、独立Expected、実Effectは6つの観測軸。H3限定Tool Read≠OS強制非漏洩・8,192byte切断解決・56/55 Reply欠落解決。新規H2の一Session実受領・Egressは未実証（#2509/#2515）。 |

#### 26.8.3 独立反例と受入条件 TT-123〜TT-130（既存Fixtureへ参照し、新Scenario SSOTを作らない）

| ID | 一つだけ変える対照入力 | 期待する局所診断・非主張 | 実行Owner |
| --- | --- | --- | --- |
| **TT-123 Document is not loaded** | docs/planのMarkdownをsourcesへ追加し管理Fileとして見せるがLoader未読込 | indexed/discoveredと`readByLoader:false`を保持。未解析ContentからRule/Requirement coveredや意味の欠損0を導かない | #3346/#3262 |
| **TT-124 Fingerprint scope confusion** | `view.semanticFingerprint`とcatalog World/Index FPが異なる | 異Scope FPを単純な不一致Failureにしない。同Scope/同Kind同士の改版/欠損はstale/unknown | #3345/#1163 |
| **TT-125 CLI shape false absence** | nested diagnostics/array result/builtin entriesの既知正常JSONを渡す | 型付きCommand別Adapterで部分/既存Stdを識別。Schema不一致はunsupported/error、空Requiredにしない | #3347/#3262 |
| **TT-126 Open versus closed ValueDomain** | 同じ値`C2`を、Values未宣言と`{C1}`明示型へ与える | 前者は制約不宣言・後者は拒否。直接ValueDomain Testと実HolidayRows経路の未接続を別列 | #3358/#2514 |
| **TT-127 Hook undeclared data leak** | `reads=[A]`のHookへ`A,B`を渡し、`on-failure warn`の合成と比較 | 宣言外Bは拒否。実Runnerが未配線なら実Host非開示としない。produces/実Effectは未証明 | #3359/#1174 |
| **TT-128 Truncated journal prefix** | JSONL 2正常行+1破損、前2件とReader errorが返る | prefixは保存、全件/効果不在の証拠としては拒否。0600と改竄耐性は別 | #3357/#1163 |
| **TT-129 Self-build claim substitution** | stage2の自称Process digestだけをStage1 artifactと合わせ、実Invokerが違う | L3拒否。Stage1 artifact→Stage2 process・child/argv・artifactの実独立照合が必要。Reference PASS≠`cmd/tei` G4 | #3360/#3362/#2831 |
| **TT-130 Plugin Matrix reference-only missing** | 24 Binding中`editor.database_workbench` 2件に対応宣言なし。Validatorは当該Featureを能力照合しない | 参照`missing=2`は保持、製品不適合はunknown/別判断。selected 22も実Load/Invoke資格なし | #3363/#3279 |

#### 26.8.4 次に走らせる最小縦切りと、実装の優先順

1. **P0-R（Source Admission実測）**：#3356 D-9/D-10の現状を同一実Binaryの`workspace-files`/`validate`/`source-index`/`model catalog`で再実行。意図的に管理Markdownだけを加える試験と、宣言3 Ruleの実`*.tei` Sourceを残す試験を対照にし、**未読込FileをTest Requiredへ昇格しない一方、独立に指定済みRequiredを消さない**ことを検収。一般MarkdownのParserをこのテストで勝手に採用しない。
2. **P0-C（Consumer契約）**：現main [#3354](https://github.com/hat47x/tei-platform/pull/3354)の11実CLI Reply/32合成Controlの同一Consumerから、D-3/D-10のmode/schema/FP Scopeの相違を**誤Missingと誤Completeの両方向**で試験。main pushの独立CI、実Response raw byteとTool Deliveryを別々に確認する。新Readerや第二Packetは追加しない。
3. **P0-T（T1-B盲検）**：Required Source/Rule/合法Baselineの独立Holdoutが確定して初めてAgentを実行。未知Gapを意図的に非提示にした場合の**候補発見率と、未Loaded/partial時の適切な保留率を両方**採点する。既知T1-Aの行番号0実行は基線として保存し、T1-B発見実証へ加点しない。
4. **P1-E（副作用・完全性）**：#3357 JSONL prefix+error、#3359 ReadsFor実入口、#3351業務拒否/技術評価不能、#3353空集合Handoffを異なる故障原因として実Target/Observerへ結合。単体Go PASSからProducer/Consumer/Persistedの資格を導かない。
5. **P1-B（Build・Plugin）**：#3360 stage-driver限定再入性→#3362 process-image→#2831 `cmd/tei`再入契約、#3363 Ref Matrix→#951実選択・Provider Load/Invoke→#3267 exact Tupleの順。各別PRのlocal PASS、Source・Artifact・Target・Granted Runtimeの合成適合を分ける。CLI/Studio双方で同じInspector Pluginを実呼出す最短D1を並行して進める。

**観測済と未実施の境界**：#3357/#3358/#3348/#3354はmain統合済みの限定成果。#3356はドッグフーディング第2ラウンドの実測を報告する**未統合PR**。#3359/#3360/#3361/#3362/#3363は各PRが報告した局所試験/提案であって、本節で新しく実行した試験ではない。TT-123〜130は**受入計画・NOT RUN**。外部H2/H3受領、MCP、T1-B、新しい実Provider/Host、`cmd/tei` G4、Owner Meaning Acceptanceは**未証明**。実施済みの局所結果を合算して全E2E/製品テストCoverageを作らない。

<!-- tei-test-partial-inventory-dogfood-20261108 -->
## 27. 2026-11-08：部分Scenario目録でも局所正証人を保持する実装（TT-105/106）

### 27.1 実装への昇格と責務の絞り込み

[新Draft #3365](https://github.com/hat47x/tei-platform/pull/3365) は、main統合済み[#3354](https://github.com/hat47x/tei-platform/pull/3354)の限定Design Contextの上で、既存#1180 Scenarioの実Go生成・Pinned実TEI CLI検証が取得した**6 Step各一件のEvidence**を使い、§26の「対象を観測できない」ことと「未テストが確定する」ことを区別する試験用read-only投影を追加した。新しい製品APIや第二Scenario/Runner/Oracle/Hostではない。

- 既存 `tools/reference-issue-test-ability/probe.py` に `scoped_step_coverage` を置き、**独立のRequired Step集合**、実CLIの `expect_real_evidence` が検査したStep結果、対象Scenario Source Digest、受信Endの有無を**別条件**で扱う。
- 正常な6 Step実入力から、`localObserved`はその6件とするが、`productCoverage`は常に`unknown`。これは**Scenarioの限定Step到達**でありRule条件のC1・Requirement網羅・現在Grantではない。
- TT-105は受信目録のコピーから `validate-denied` を外した場合、そのStepを `notObserved` とするが、`untestedConfirmed` は空。**削除されたReceiptは真の未実行を示さない**。
- TT-106は同じ部分目録でも実測 `validate-positive` の局所正証人は保持し、全ProductのCoverageはunknownとする。正の証拠を認めるのに全世界の分母完全性を要求せず、全体Coverageの推論を許さない。
- 追加対照は、必要なEndが欠ける6件の見かけ上の完備／欠落のある偽End／Source Digest不一致／重複Step Receipt／`verified`欠落。出力は既存 `test-ability.json` の `scenarioInventory` と `scenarioInventoryControls` に**試験用属性**を追加。Report既存Schemaの正式製品API採用とはしない。
- .github/workflows/tei-issue-design-dogfood.yml で現存の実Pinned CLI Step結果の後に7 ControlsとReportの入場判定を検証する。Code変更の本PR exact HEAD CI合否と、その後のmain統合可否は別途検収。

### 27.2 失敗診断をモデルのテスト不足から分離する

| 受領／実行の実態 | 範囲相対の正しい表現 | 追加取得すべき証拠 |
| --- | --- | --- |
| 特定Stepの実Pinned CLI Evidenceはverified、他のStep Replyは欠落 | 当該Stepのobservedを維持、欠落Stepはnot-observed-in-received-inventory | Request/Response/EOFまたは明示Skipped、全Step実選択 |
| 6件のStepが揃うがEnd未取得 | six-steps-seen、inventory partial | 実受信End marker（外部HarnessではRequest ID/ordered parts/digestも） |
| Endがあるが一件欠損／重複 | incomplete／ambiguous、未テスト確定不可 | Subject/Step ID、実選択・届いた独立Receipt |
| Scenario Source Digest/実Run Revision不一致 | stale/unqualified。古い局所証人を現対象へ転用しない | Source/Plan/Artifactの同一Attempt束縛 |
| Scenarioの6 Stepがすべて実行済 | 当該Pinned Scenario内の6 Stepについて局所観測 | 別途独立Rule/Requirementと適用Scopeの全分母 |
| Business拒否・技術失敗・EffectUnknown・Provider部分取得 | Status/実ResultとStepの履歴を独立保存 | #2514のIndependent Expected、#1174の実Effect/Receipt、#2506のSlice completeness |

重要な非推論：部分目録の一件欠損は、新規Test Patternの必要性を確定しない。一方、正しく観測済のStepを全体不明ゆえに消去する必要もない。**テスト不足候補を推薦する前に、どの集合（Requirement/Case Manifest/選択済Test/実行Receipt/外部受信）が不完全なのか説明**する。自律テスト生成の「最小指示・無駄なケースを増やさない」UXはここに置く。

### 27.3 今後の実装・検収順

1. **P0 現PR**：#3365の実CLI＋7対照を同一exact HEAD CIで実測し、Report中のStep局所正証人とProduct unknownの両立を確認。実CLI/Scenarioの別版・健康診断成功を本PRへ流用しない。新PRは作成時Draft・未受入。
2. **P0 H2/H3比較**：#2509/#2515に依頼する実外部Tool受信で、受領Step・End/Part・stdout以外のSink/Session再開も検査。今回は既存Runner内の合成受信目録なのでH2/H3が資格化したとは言わない。読取Providerがread-onlyでもログ・Cache・後続Agent sinkが保存/送出するEffectは別扱い。
3. **P1 AI Test Gap**：#3262がQualifiedな必要部分とPartial/Unknownを供給し、#3273/#1180のCase候補からまだ未提示の独立Obligationを#2514が盲検採点。**受信不足の修復**を新たな未テストの発見として採点しない。
4. **P1 多Consumer/Plugin**：同一独立ObligationをCLI/Studio/SHOやPlugin/Trait/BOMへ投影する場合、同じSubject IDの見かけの複製を分母へ加算せず、実読み込み/実呼出・View occurrence/Provider exact tupleは別軸で比較する。#1251/#951/#3279/#3267が実走Owner。
5. **P2 C1・NFR・Effect**：Go Code C1/Scenario Rule判別/HTTP Graph Guard/現DB Effect/NFRは、それぞれ独立分母と実行証拠で検収する。全Stepの実行だけで業務・C1・SLO達成を名乗らない。

**#3365 exact-head CI検収追補**：HEAD `1e2531fede563ed4b1d32cca0e78ccaf69fe364c`（`probe.py` blob `d6cebb6d45c099914a85d7a360b3522c2439eb57`）に対し、[専用CI #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357)、[Health #42394673331](https://github.com/hat47x/tei-platform/actions/runs/42394673331)、[依存 #42394673273](https://github.com/hat47x/tei-platform/actions/runs/42394673273)は**completed/success**。§27.3の「未確認」はDraft作成時点の状態。現HEADはCI成功した**未統合候補**であり、main統合後のCLIバイナリ・実Runは別検収。

**次の反証点（既存TT-105/106を強化する試験条件、独立Case正本は増やさない）**：`scoped_step_coverage`はtest-onlyのread-only Consumerであり、`end_received`と`row.verified`は**呼出元が供給する値**。同関数単独ではそのEndが実Receiverに到達したか、verifiedが実Pinned CLI検証に由来するかを認証しない。**「全Required Stepの偽Receipt＋偽End」**という全件揃った改変コピーを追加対照にし、単なる配列整合/booleanから**受領完全性または実行適格を自動認定しない**よう、実Invoker／Run Receipt／actual Tool Deliveryの独立Witnessが必要なことを明記する。現在の7 Controlsは一Step欠落や偽End（欠落同伴）を拒否するが、全件偽装への独立Producer照合の実証とは別。製品の偽造脆弱性が実証されたとの主張ではなく**テスト用Consumerの信頼前提の明示**である。

**Claim ceiling**：§27の新実装は既存実CLIの受領済みStepと改変コピーを扱うbounded test-dogfood。真の外部Tool配送欠落、実Provider部分故障、Product全体Rule Universe、Method/Holdoutを隠したT1-B、Go C1・正式Owner受理を証明しない。新Source/Plugin Resolver、Test Runner、認可/Authority、専用Proof Storeは導入しない。

<!-- tei-test-real-qualification-patterns-20261108 -->
## 28. 2026-11-08：新しい局所実装を「対になった反例」で評価し、T1-Bの真のTestabilityへ接続する

### 28.1 今回取り込む実装事実と、取り込まない上位Claim

最新mainの[#3354](https://github.com/hat47x/tei-platform/pull/3354)は既存 `context_gate.py` 一本にTT-56の3 Validation/Focused Rule+Relation、全Source Index/Loader/Canonical Anchor照合を集約し、[PR exact-head CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)で**11 CLI Reply・32局所Consumer判別**を確認した。main統合は確認済み、**統合後mainの別push成功は未確認**。既知T1-Aの第7 Stepも限定実行済みだが、**AIの未知Rule発見T1-Bではない**。

別レーンの具体的な実例には、[merged #3353](https://github.com/hat47x/tei-platform/pull/3353)の**空集合でも未確認Handoffを完了としない**修正、[merged #3358](https://github.com/hat47x/tei-platform/pull/3358)の**休日Calendarの値域が閉じている場合に限り宣言外値を拒否**する局所実装、[merged #3350](https://github.com/hat47x/tei-platform/pull/3350)の**Provider能力を技術対応で解決**するReference改善がある。3件とも特定Code/Referenceの限定成果で、実業務の他者受領、未知暦全Domain、実Provider Invoke/Grantの合格ではない。

進行中の[#3359](https://github.com/hat47x/tei-platform/pull/3359)はHookへ渡す宣言済み `reads` の事前検査候補だが、Providerからの取得やCLI/RunSequenceへの結線は未実証。[main統合済み #3363](https://github.com/hat47x/tei-platform/pull/3363)は既存Capability Supports*による24 Binding中selected 22/missing 2の**read-only参照算定**を提供し、`editor.database_workbench` のmissingを直ちに製品不具合とはしない。[main統合済み #3360](https://github.com/hat47x/tei-platform/pull/3360)は自己再構築可能な**参照Driver**についてStage1生成物がStage2を実駆動した局所L3 trueを実測したが、**cmd/teiのL3・全Source Closureではない**。それぞれの現実の観測を、テストPatternの独立対照へ変換する。

### 28.2 Patternを単なる入力値集ではなく「性質を変える一軸と不変部分」の組にする

Pattern候補の適用と判別に、少なくとも **(a) 対象のAccepted/Provisional obligation・Source、(b) 合法なPositive、(c) 変える一条件と期待するNegative、(d) 変えてはいけない他の条件、(e) 実Consumer/Entry/Expectedと必要Observer、(f) 適用条件/欠損時停止、(g) 版・Scope・Grant・開示、(h) 誤ってPass/Failになる既知対照** を要求する。これは#1180 Scenario／#2514 Independent Expectedを参照するread-onlyな推薦規則であり、新しいテストDSL、検証器、Oracle正本の追加ではない。

**正と負の双方を持つこと**だけで判別力が保証されるわけではない。両者が別のTarget/SQL版やTenant、異なるSourceを使った場合は比較不適格である。また一条件の変更が他のRule/Effectを壊す場合は「目的のRuleを検出した」とは認定しない。未確認値を既定値で埋めると、異常が成立せずfalse-qualifiedになるため、**missing・null・empty・not-observed・partial・permission-deniedを入力契約上分ける**。

### 28.3 他チャットの実装から得た「一軸差」候補

| Test Pattern / 対照（TT-115〜TT-122） | 最小Positive ⇄ Negativeと独立に判定すること | 実証境界・Owner |
| --- | --- | --- |
| **TT-115 空集合のCompletion** | Targets=0・EmptyJustified=true、HandoffConfirmed=true ⇄ false。他条件固定でfalseをCompletionUnverifiedへ。Denyは確定拒否として保持 | #3353は既存semanticの局所修正・テスト、**実RecipientのReceiptは別**。#1125/#2514 |
| **TT-116 開いた／閉じた暦値域** | 宣言されたCalendarDomainの合法値 ⇄ 宣言外値では拒否。一方Domain Valuesが未宣言ならopenであり「宣言外」として自動拒否しない | #3358の型値域局所実証、実HolidayRowsForCalendar自動配線・旧DBは未。#1126/#2514 |
| **TT-117 HookのDeclared Reads** | 宣言済みAttributeだけを値ごと渡す ⇄ 未宣言Attributeを追加/宣言値が未観測なら実装呼出前に拒否。コピーへの後追い追加も拒否 | #3359 Draftの局所8件。CLI/RunSequence/Provider取得・実Hostは未。Hook Owner/#951 |
| **TT-118 Plugin Capability選択の粒度** | DB Providerは`SupportsDatabaseProvider`で実対応 ⇄ 能力IDと技術名を文字列比較し誤unselected。選択済だがInvoke未観測の別Worldも比較 | #3350 main参照：unselected6→4（算定方式変更）。#3363は24 Binding中22 selected/2 missingのDraftで**製品Validationより広い参照**。#3279/#951 |
| **TT-119 実Build再入主体** | Stage2 invoker digest = Stage1 artifact digest、同child recipe・実disks再照合 ⇄ invoker違い／異recipe／artifact違い | #3360はReference Driver限定のL3 trueであり、実TEI `cmd/tei` L3/G4資格ではない。#2831 |
| **TT-120 技術失敗と業務不成立** | Numeric 0,nil はRule not-satisfied ⇄ Number nil,nil、Source errorはEvaluation failure。正常正値はsatisfied | #3351のGo test-only Draft、実Providerのpartial/permission deniedは別。#2514/#1180 |
| **TT-121 部分Journalと原子的Receipt** | 完全なLog/Chainを独立PinでRead ⇄ 中途に壊れたJSONL行を挿入。前行が返ってもErrorを無視して「ログ全件、完了」としない。末尾削除をPinなしで検知可能とは言わない | #3357の共有JSONL Readerはmain、#3364のChain共有化はDraft。Receipt Owner #1163/#1174 |
| **TT-122 同一Rule／別Consumerの非開示** | 同一許可Public SliceのW0/W1でPrivate Role件数のみ変更。CLI BodyだけでなくTool Reply・stderr・Session Log・保存sinkで許可外差異ゼロを要求 | #3309 DSH合成canaryは保存先の限定観測、OS/実外部Providerの安全非干渉は未。#2509/#2515/#2516 |

上表は**各Ownerの既存Source・テスト・Expectedを参照した適用候補**であり、Referenceの実測をあたかも本試験で再実行したようには扱わない。互いに異なるドメインのCaseを「成功したテスト件数」へ単純加算しない。Positive/Negativeが同一Attempt/同一Dataset上で同一変異として独立比較できない場合は、`non-comparable`として残す。

### 28.4 現実のT1-Bと製品Readを妨げない、二本の最短切り口

**A. 既存Context/Scenario上のTT-105/106を検収優先**：直前の§27と[実装Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)が、既存実CLIの6 Pinned Stepを受けた部分目録で**局所観測済みStepの正証人を保持**しながら、欠損Stepを`untested`と確定せず、全製品Coverageをunknownに保つ**test-only read-only判定**を既に担当している。第二Checker/Runner/Scenario正本は追加せず、まず#3365のexact HEAD CIで7対照、End/重複/Source digest/verified欠落を検収する。Required Rule/Function/Business・実外部Tool受領・Sourceの原子的共同Readは#3262の後続。

**B. 実Product Functionの最初のReadを別Ownerで並行**：#2586のE `examples/xtea-lowcode` の `order_entry.recent_orders` を固定し、Function→実選択Query/SQL byte→`:status` actual bind→physical Provider→Result Slice/Context→実Consumerの**同一Attempt**の最初の不足辺へ接続する。W `reference/workspace` の`:from` SQLとCの隔離PG Reference成功をEへ合算しない。Queryが0件ならScope/SQL Predicate/Target Readerのcompleteを証明するまでは「不存在」としない。図面/CRUDのP0-M、実ReadのP0-R、同一Identity投影のP0-Cは並列に進めてよいが、AのHard Dependencyにはしない。

**T1-B**はA/Bの**全完成を待たず**、対象一局所Ruleの必要Source/Meaningと受領資格が満たされる場合に、生成者へ未提示のGapを独立Reviewerが選び、前節のB0/B1/B2/BXで盲検提案・既存Scenario/CLI実行・独立判別を進める。既知T1-A=行番号0や公開済みMutantは盲検評価の加点にしない。テスト実行の候補が多いことより**独立に正しい不足を発見し、誤った完了Claimを出さないこと**を測定する。

### 28.5 C0/C1・現在資格への引渡し

Go標準 `go test -coverprofile` はstatement系Coverageであり、分岐の結果別到達をそのまま保証しない（[Go公式](https://go.dev/doc/build-cover)）。#3351の10分岐Outcomeは独立Expectedと実Goテストの対象であって、Target別全Code C1の数値ではない。§24のCB PoCでInstrumenter/SourceMapの対象とFalse Positive・Toolchain/Build版を固定する。新Coverageの集計では **Required-known / Unknown-required / Measured-denominator / Observed-reached / Independent-checked / Current-eligible** を別に表示する。閲覧者向けに非公開件数を明示できない場合は公開可能なScopeだけを表す。

**停止条件**：空集合のCase自体を完了、未宣言Enumをclosed、Hook宣言を実Provider Read、Plugin選択を実Invocation、Reference L3をcmd/teiのL3、JSONL途中までを全Journal、T1-Aの再提示をT1-B新発見、C0局所をC1/全製品Coverage、Owner未受理のCandidateを正式業務テストと呼ぶこと。

**本節の検証状態**：各Ownerのmain/Draftと本設計文書を照合した**計画更新のみ**。TT-105/106は§27・#3365で**実コードを持つDraft（exact-headのCIは別検収）**、本節TT-115〜122の横断適用・T1-B盲検Trial・A/B両経路のProduct実結合・実H2/H3外部受領・Code C1・現Host Grant/Effect・Owner正式Meaning AcceptanceはNOT RUN／NOT ACCEPTED。Mainに統合済みの個別試験結果はそのSourceの限定Claimに限って引用する。

<!-- tei-test-local-witness-vs-closed-absence-cross-consumer-20261108 -->
## 29. 2026-11-08：TT-105/106の実CIから「正の局所証拠／閉じた不在証明／複数Consumerの同一性」を実装優先へ（限定実証）

### 29.1 §27の計画から進んだ実証を、exact HEADへ限定して反映

[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)の現HEAD `1e2531fede563ed4b1d32cca0e78ccaf69fe364c`で、[専用Actions #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357) Job `128469383600` は**completed/success**、[Repository Health #42394673331](https://github.com/hat47x/tei-platform/actions/runs/42394673331) も**completed/success**（同PR HEADのPull Request Trigger）。専用Job実ログには、mainに既存の11 read-only CLI/32限定Context Controls、6実Pinned Scenario、実CLI拒否6＋独立義務/Oracle拒否2、**#3365追加のTT-105/106と7個の目録コピー対照**を検証する実Assertionがあり、`TT-105/106: complete-for-one-pinned-scenario; observed local steps without claiming complete Product coverage` と報告されている。既知T1-Aの候補境界も継続実行。

この結果は**§27の「exact-head CI未確認」を当該HEADについて更新する**もの。PRは本照会時点で**open/draft・main未merge**であり、統合後mainでの新Run、実外部H3の途中切断、全業務Required、実Code C1、実Provider/Host、未知テスト発見T1-B、Owner受理を示さない。

### 29.2 今回、TEIのテスト能力に必要な二つの「証明向き」を固定する

| 判定の向き | 成立に必要な最小Witness | 不成立時の扱い |
| --- | --- | --- |
| **存在・実行の正の局所証拠** | 対象Step/Ruleの安定Ref、実Run Source/Scenario/Artifactの比較可能な版、実Result/独立Expected、受信した検証済みReceiptの範囲。別対象のReceipt欠落は、既に確認した正証人を自動消去しない | 観測済みの局所Stepだけを`observed-in-scope`。対象Revision/Consumer/Scopeが変われば再資格化 |
| **存在しない／未テストの負の結論** | **独立Required Universe**、該当Source/Rule/Scenario選択集合の用途相対閉包、Enumerator/Loader/Parser、実実行予定と実Run目録の閉包、終端/継続・Recipient全受信、DisclosureとRevisionの比較可能性 | 一件のReply欠落なら`not-observed-in-received-inventory`。絶対的な`never-tested`や業務上の不存在を宣言しない |
| **Ruleがcoveredであるという結合証拠** | Step到達に加え、指定Rule/Decision outcomeへの**実Binder/実Evaluator**、入力/Target、Independent ExpectedとRule別Resultの照合 | Stepの`verified:true`、Model上のRelation、コードstatement C0だけではRule coveredにしない |
| **全Product Coverage** | 受理/独立Requiredの全分母・対応Targetと異質Source/未実装の独立観測、C0/C1/NFRと現資格の用途別集計 | #3365の`productCoverage:"unknown"`を保持。局所率を全体率へ外挿しない |

**局所的な正証人の単調性は条件付き**：同一Subject/Source/Attempt/Observer内で未関連ページを失っても有効な別Receiptまで自動で消さない。しかしSource revision・選択Agent/Rule・独立Expectedの入替え、Rev失効、開示権限の変更、重複/偽Receiptによる真偽の曖昧化まで同じ正証人として持ち越さない。#3365の実装はあくまで**6 Stepの受領済み証拠を試験用に投影**し、実外部Tool宛てのACK/endを構築したわけではない。

AIに対する推薦が`未カバーのテストを追加`か`まだ必要情報が取得できていない`かは、**Rule Required集合**、**選択済Scenario Case集合**、**実行実績**、**その受領目録**、**開示対象Consumer**を区別して決める。どの集合が閉じていないかを具体的に示せること自体が、UIとSkillに共通した価値である。

### 29.3 CLI・Studio・物理DB Reverseの新成果は「同じ証拠」ではなく対照Consumer

- **CLI result契約 [Draft #3370](https://github.com/hat47x/tei-platform/pull/3370)**：`tei requirements` の成功時`result`は**配列**であり、Discoveryへの`resultShape:"array"`追記と実Binary対照Testが候補。PR HEAD `dcf759a04084fc03134a6f7971864609c71497f5`の[read-only dogfood CI #42395222332](https://github.com/hat47x/tei-platform/actions/runs/42395222332)はsuccess。ただしRepository Healthは本照会時にin_progressであり、PRはDraft/未統合。Discoveryの外側`successShape`、`jsonContract`、内側`resultShape`は別。未宣言のResult Shapeをobjectと補完しない。正しい配列`[]`の受領成功と、独立Requiredの充足0件/不明を分ける。
- **Studio Inspector [Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)**：`internal/studiohost`のHost選択済み`InspectionService`へReaderを接続し、Factoryありで未提供/誤Subject/誤Fingerprint/誤Effectなら**暗黙builtin fallbackを禁止**する正負試験候補。HEAD `a42df66bf9faa4c55a888a9c4bb900252e450e8a`の[限定Issue dogfood CI #42395396794](https://github.com/hat47x/tei-platform/actions/runs/42395396794)はsuccessだが、Studio固有/Healthの全CI・実Plugin Resolver/Link/Invoke/CLI↔Studio二Consumer等は未確認。**実Host選択Service呼出≠実Plugin Provider選択/Current Grant**。
- **Workbench Physical Relation [Draft #3372](https://github.com/hat47x/tei-platform/pull/3372)**：合成`postgres.Provider.Introspect`→Workbench Projectionで**Observed FK/UNIQUE・複合Foreign Key・並列Relation・CapturedAt/Fingerprint**を保持するコード/Go test候補。先行HEAD `402a57f79f8d2e77493683d539ef22e1f1af7aec`から最新HEAD `5c1d94f2e65793102c9c6b25349d7c83fe556826`へ進行。最新HEADの主要Go/Health CI [#42396293679](https://github.com/hat47x/tei-platform/actions/runs/42396293679) は**照会時queued**、Wails Desktop smoke [#42396293510](https://github.com/hat47x/tei-platform/actions/runs/42396293510)のみsuccess。旧HEADやsmoke成功を新Go/Health/業務適合へ昇格しない。**合成物理Catalog→Workbenchの観測がData/Businessの受理済みRelation/Function CRUD/実Row Readへ昇格するわけではない**。
- **実Dogfood D-3/D-9/D-10 [PR #3356](https://github.com/hat47x/tei-platform/pull/3356)**：`view`のNamespace範囲Fingerprint、管理Markdownの`readByLoader:false`、CLIコマンド別JSON Shapeは、#3370/#3371/#3372の各Consumerが**False-Missing/False-Covered**へ落ちないための独立入力。実TEI専用開発MCP Serverは本再測定で確認できずMCP実試験はNOT RUN。各Source/Owner #3345/#3346/#3347、#3262を維持。

### 29.4 独立検収に引き渡すTT-131〜TT-138（既存Runner/Expected正本参照）

| ID | 最小正負対照／変える一軸 | 判定と必要な証拠 | Owner |
| --- | --- | --- | --- |
| **TT-131 局所正証人** | 6 Stepの実Receiptから1件を削除、別の正常Stepの実Receiptは残す | 正常Stepの局所observed維持、削除Stepはnot-observed、Product Coverage unknown。#3365 exact HEADで限定PASS | #3365/#2514 |
| **TT-132 False Never-Tested** | Step 1件が受信一覧にないが実選択/送信/外部ACKの閉包は未証明 | untestedConfirmed空。実外部Harnessが未検収なら別資格。#3365のコピー改変7対照は外部配送の実故障ではない | #2509/#3262 |
| **TT-133 Step→Rule誤写像** | Stepの実verified Receiptは正しいがRule Binding/実Evaluatorが異なる版を使用 | Step observedは保持、対象Rule coveredはunverified。Role/Source/Actual Input/Independent Resultが必要 | #1180/#2514/#3262 |
| **TT-134 CLI Array正常** | `requirements`成功`result:[]`を、object専用Consumer/未知resultShapeへ渡す | JSON Shape契約とRequired欠落を別票。正常配列を異常/全Requirement 0件と誤断定しない。Draft #3370の局所Product CIに委譲 | #3347/#3370 |
| **TT-135 Host選択Inspector** | Studio Factoryから選択Inspector Aを供給 ⇄ FactoryありだがInspectionなし/Subjectが異なる | A実callの正証人と、無断Builtinへの回帰禁止・typed errorを別判定。Plugin B actual LoadやCLI一致は未確認 | #3371/#1824/#951 |
| **TT-136 Physical FK≠業務Relation** | 同一Catalogに複合/並列FK2本、Function/Business Meaning未受理 | Physical Observationでは2辺と方向/Source保持、Diagram/CRUDのAccepted Meaningはunknown。Candidate→Owner判断を要求 | #3372/#1124/#2586 |
| **TT-137 構造Missingの偽陽性** | Raw MarkdownはmanagedでLoader未読込、Source Index mode違い、Result array・局所FP混在 | 不明/未Loaded/比較不能を独立診断。正常Query Shapeや異Scope FPを「Ruleは不存在」と誤判定しない | #3356/#3345/#3346/#3347/#3262 |
| **TT-138 部分目録の開示** | 許可Public Sliceは同一、秘密Step/Role数だけ異なる二WorldをCLI/Studio/Harnessへ投影 | Public Body・件数/診断・Tool/Session副送出の許可外差異ゼロを要求。外部Sink/OS強制未実測ならnot-proven | #2516/#2509/#2515 |

§28 TT-115〜122/§26 TT-123〜130と同じ番号・Fixture/Ownerを上書きしない。**TT-131/132の#3365限定CIのみ今回は既存他BranchのPASSとして根拠を採録**。TT-133〜138の横断統合E2Eは今回新実行していない。

### 29.5 最初の一往復を組織的に短くする

1. **P0-A #3365の限定正証人を基線化**：PR exact HEAD [CI #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357)、Health #42394673331は実success。main採用前の文書・Code/CI検収を#2514/#1180へ渡す。もしmainが進んでいれば、その新HEAD/main push CIは別Run。現`scoped_step_coverage`はtest-onlyで`productCoverage=unknown`を保持し、全業務Coverage Engineにしない。
2. **P0-B 正常入力をFalse Missingにしない**：#3370の実`requirements.result[]`、#3356のD-3/D-9/D-10を既存#3262 Consumerに通し、「必要Rule/Relationが本当に欠けた」と「ConsumerのSchema/Mode/Loader/FP Scopeの誤読」を対照検査。合法な非Required省略で過剰拒否しない。**二つ目のReaderは作らない**。
3. **P0-T Blind T1-Bを一局所Ruleで実行**：現在Allowed Sliceの実Source/Required/Meaning/Tool Replyを同一世界で束縛し、独立Reviewerが未提示ObligationとExpectedを事前固定。AIに渡したContext/Role/Source/秘匿Holdoutを分離し、提案→Review→既存#1180 Scenario→実Pinned CLI/独立Mutant→#2514 Evidenceへ。既知行番号0 T1-Aの再選択をnovel discoveryとしない。
4. **P1-D 二Consumer実Inspector／Reverse観測**：#3371でHost selected serviceと実Callの一致、CLIとの同意味・同版/Disclosure、Plugin exact Loadを別検査。#3372はPhysical Observation→Candidate→Accepted Meaning→Diagram/CRUDという変換を段階別Witnessにし、実DB接続・現Grant/Row/Business CRUDはそれぞれ別。
5. **P1-S 新Patternの適用適格**：#3353空Handoff/#3358 open Calendar/#3357 JSONL prefix+#3359 Hook reads/#3360 SelfBuild等の§28独立対照は、Required＋実Consumer/Artifact/Effectの根拠があるときだけ推薦。Case数やLink見かけを「新しいCoverage増分」と計上しない。

**更新した設計判断**：テスト基盤は「生成する/しない」のbinaryではなく、**対象となる業務Obligationに対して、局所的に確認できた正の事実、まだ閉じていない探索範囲、真正に未カバーと判断できる領域を分離し、次にどのReader/Observer/Reviewを必要とするか推薦する**。その同じ結果をCLI/Skill/MCP/Studioへ用途別に整形して渡す。別のSemantic Store、Coverage Engine、Host、Authority、Test Runnerを初手で増設しない。

**本節の資格**：#3365 exact HEAD CI＋Healthのsuccess、#3370の限定dogfood CI成功はGitHubから実確認した他Branchの結果。#3371は一部CIだけ成功、#3372の重要CIは未確定。**今回のGitHub作業は設計文書・Owner引渡しだけ**。新実CLI/Go/実Provider/CLI↔Studio同一Consumer、T1-B盲検、MCP・外部H3、Go C1、Owner正式意味受理はNOT RUN/NOT ACCEPTED。


---

<!-- tei-test-nov08-function-parity-source-provenance-p0 -->
## 30. 2026-11-08 横断統合：13 CLI/42 Controls、CLI/MCP局所同値、Source occurrence鮮度をT1-B入場基準へ反映

### 30.1 現mainの実証基線を更新（旧各章は観測当時の履歴）

| 分離するClaim | 2026-11-08の実装事実と限定Evidence | 依然成立しないClaim |
| --- | --- | --- |
| **対象Ruleの局所Required** | main統合[#3354](https://github.com/hat47x/tei-platform/pull/3354)は11実CLI Reply／32合成Consumer Controls。局所Validation 3件とFocused Relation/Canonical/Sourceを検査 | 全Data/Function/Business/Scenarioの独立Required閉包、原子的Read |
| **指定Functionの局所存在** | **main統合[#3482](https://github.com/hat47x/tei-platform/pull/3482)**は`sales.order.submit`の独立RequiredをCapability／Application／Sourceへ束縛。main push commit `8ed583790e63b63527c1dd8f8aecc1d6ce6e3260` の[専用CI #42462861657](https://github.com/hat47x/tei-platform/actions/runs/42462861657)は**completed/success**。13実CLI Reply・既存32+新10=**42合成Controls**、別Context Packet 9負例、6 Pinned Scenario＋T1-A、Source-only drift負例を同じmain Runで確認 | `sales.order.submit`の実Function Entry/Build収録・実Provider Invoke・全Business Rule/Host Grant・現行実Effect |
| **Scenario Stepの局所肯定** | [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のexact-head専用CI #42394673357とHealthはsuccess。実6 pinned Stepの試験用受領目録を使いTT-105/106を検査 | main統合、全Step送信/外部Tool完全受領、未知のTest未実施や全Product Coverage |
| **CLIとMCPの接続** | **main統合[#3403](https://github.com/hat47x/tei-platform/pull/3403)**で4 read-only Toolの**参照stdio MCP Adapter**が実在。**main統合[#3439](https://github.com/hat47x/tei-platform/pull/3439)**でADR-0048の共通Application API方針を採用。**main統合[#3458](https://github.com/hat47x/tei-platform/pull/3458)**で`validate`の正常・Load失敗・意味エラーに対してCLIとMCPの文書・成否を局所回帰 | 正式な全Transport共通APIの完成、4 Tool全応答のByte同値、実外部H2/H3 Harness完全受領、実OS/Host/Disclosure・配備資格 |
| **同Workspace Source-only drift** | **main統合[#3477](https://github.com/hat47x/tei-platform/pull/3477)**で`validation.tei`末尾改行だけの変更時、World同一／SourceIndex版差による旧Context拒否・実CLI Pin比較を移植 | Importした別Provider Sourceの変更、A→B→Aの履歴不変・原子的Snapshot、現在のMeaning/Grantの再受理 |
| **Imported Provider Source-only drift** | [Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)はConsumerのWorld/SourceIndex二Fingerprintが不変でもImport Providerの原文が変わる対照を設計し、Provider occurrence/sourceが別に必要とする | Draft exact-head Go/CI合格・nested source全Byte・real Consumer Source-use、全Plugin/BOMの再資格化 |
| **HTTP構成とGo Hostの対応** | main統合[#3478](https://github.com/hat47x/tei-platform/pull/3478)は実Go Hostの有限10経路を対象とし、main統合[#3485](https://github.com/hat47x/tei-platform/pull/3485)はE1とGo Hostで片側宣言が異なるケースのRoute/Action比較を追加 | 実E1宣言→正式TEI Loader→選択済Provider実Invoke、全Audit/Client Receipt・Effect結果 |
| **セルフビルド** | main統合[#3440](https://github.com/hat47x/tei-platform/pull/3440)は`cmd/tei`自身が駆動するstage0→stage1→stage2の限定実行を観測。三つの実プロセス/Artifact関係を区別 | Source Closure `rejected`、Trusted Supply、全G4再現性/配備/Owner資格。参照Driverだけの旧「cmd/tei L3なし」は履歴 |

**新旧状態の優先関係**：§14/§17/§25/§29の「MCP入口なし」「11 CLIが最新」「#3370はDraft」等は当時の記録であり、最新の実証上限は本§30を優先。main統合済み[#3370](https://github.com/hat47x/tei-platform/pull/3370)の`requirements.resultShape=array`は、正常配列の契約であって全Required存在の証明ではない。別PR/別CIの局所PASSを同一Attemptへ加算しない。

### 30.2 Design Contextの鮮度は一つのFingerprint比較では閉じない

T1-Bが参照した設計Sourceの**必要依存集合**を、少なくとも次の別集合で照合する。新World Registry/Provenance Engineは作らず、既存Source/Loader/Provider/Build/ConsumerのEvidenceへrefとして結び付ける。

- **World/Canonical**：実Loaderが消費したSemantic WorldのIdentity、用途相対のAccepted MeaningとSource/Role。World同値だけではSource byte同値を保証しない。
- **Source Index/Location**：Query mode、Source path・span、原Byte revision、index版。末尾LFだけの改訂でもSource Indexに依存するReader/Scenarioはstaleとなり得る。変更影響のない別対象の証拠を全失効させない。
- **Import Provider occurrence**：実選択した外部Source URL/Provider/Plugin/Revision/取得時点と必要なByte/Content fingerprint。Consumer World/Indexが不変でもProvider原文が変わり得る。Providerが未Load/未選択/未消費なら`unknown`のまま。A→B→Aは最後のByte同値で途中履歴を「無かったこと」にしない。
- **Execution/Delivery**：Scenario Pin、実CLI/MCPのAdapter/Tool版、Tool Resultの実受領、Selected→Loaded→Invoked、現在Grant/Disclosureと必要Effect/Read-back。CLI/MCPで`validate`の文書が一致しても外部HarnessのACKや実Provider読取は別資格。

**差分対応**：Src same/Import changedなら対象Imported Roleに依存するTest Claimだけ再資格化、Src changed/World sameならIndex/Locationを使うReaderの再検収、Source same/Tool shape changedならResult Shape/Adapterの再検収、Adapter same/Grant revokedなら実行・外部開示の現在資格だけ拒否。いずれも独立Requiredの分母を「観測できたもの」に縮めてはならない。

### 30.3 T1-Bを本当に盲検にするための一回の実験契約

**実行は既存#1180 Scenario SSOTと#2514 Independent Expectedの範囲内**。対象は公開合成の一Domainで、検証者が先に独立Required Gap・合法Baseline・Rule/Relation/必要Source・対象Target・Positive/Negativeと少なくとも一つの識別可能なKnown Mutantを固定する。生成者は**HoldoutのGap ID、変異箇所、Expected/Mutant答えを受け取らない**。公開済み行番号0 T1-Aや既知のRequired3 Ruleを繰り返したものは新規発見に加点しない。

1. **Pre-register**：Reviewerは必要Gapを独立記録し、固定Source/Tool/ScopeをPin。Probe入力とAIに渡すSource/Rule情報の許可部分集合を別票とする。未公開Hidden値はIssue/CI公開Artifactへ出さない。
2. **Readiness**：main #3482の13実CLI／42合成Controlsは限定初期Witness。T1-B対象の正確なRule/Source/Import occurrence/合法BaselineとTool Replyの適格性を別判定。Scope外/partialなら`insufficient-context`、開示失敗なら`ineligible`とする。
3. **Generate**：短い指示「対象Functionの必要な未カバー検証を提案して」に対し、生成者は根拠Ref、変更する一条件、変えないInvariant、PositiveとNegative、必要Observer、欠落情報を提示。追加Parameterを連続質問する前に既存Patternを合成する。
4. **Review/Execute**：候補を既存ScenarioのReview Candidateに写像。独立ReviewerがExpectedを別経路で比較し、採択した候補のみ実pinned CLIに渡す。Rule Resultだけでなく実Target/Source/Tool Receiptを照合する。
5. **Discriminate**：Fixed legitimate Positiveと目的Negative、Known Mutant、意図的にTarget未到達/Response欠落の対照を区別。生成Caseが実対象の欠陥に到達しない場合、独立Expectedが正しくても「Mutantを検出した」とはしない。
6. **Report**：required-known、required-unknown、found-valid、rejected-ineligible、executed、independently-discriminated、currently-qualifiedの差集合と、操作回数・Tool予算・開示/Source再取得費用を表示。**検証不足の修復と、新たなRuleテストの発見は別成果**。

外部Harness比較は同じHoldout/Source/Budget・現行Disclosure・Tool byte受信完全性が揃うSessionのみeligibleとする。#2509の欠落受信Sessionへ新しいProduct Skill成功率を付けない。Go C0は局所Statementのみ、Go C1・全NFRはunmeasuredのまま。正式AI Trial・生成者のBlind条件の成立は、この章の追記では**NOT RUN**。

### 30.4 最小追加の横断対照 TT-139〜TT-146（Ownerへ委譲）

| ID | Positiveと意図的差分 | 独立Expected・検出する偽Claim | Owner |
| --- | --- | --- | --- |
| **TT-139 Source-only index drift** | World固定、`validation.tei`末尾LFだけ改訂 | 古いContext/Source index資格を拒否。ただし旧Worldと新Worldの意味同値を「権限/歴史不変」と呼ばない。#3477は実CLI限定PASS | #3262/#3477/#2514 |
| **TT-140 Import occurrence drift** | Consumer World/Index双方不変、必要Import Providerの原Byteだけ変更 | Consumer二FPが同値でも当該外部Sourceの引用/Rule資格はstaleかunknown。Draft #3479の独立Go testと製品実Readは別 | #3479/#3262/#2600 |
| **TT-141 CLI/MCP validation parity** | 同Workspace正常、Load失敗、意味重複の三条件をCLI vs参照MCPで比較 | bodyの診断/文字列error code/成否を区別。主張は`validate`局所同値のみ。#3458の実回帰はmain、外部H3完全受領は未 | #3420/#1158 |
| **TT-142 Result shape≠Required zero** | `tei requirements`の正常な`result: []`、Source Requiredは別集合 | arrayをobject受信失敗と誤判定せず、0要素を「業務Required 0」と断定しない。#3370はmain | #3347/#3262/#2514 |
| **TT-143 Function declared≠invoked** | `sales.order.submit`のCapability/Application/Source Requiredが存在、実Entry未到達 | #3482の`bounded-present`は限定正の証人。実Function/Provider/Grant・業務結果Coverageは未証明 | #3482/#2586/#1180 |
| **TT-144 E1/Host route split** | E1のRoute/Action片側だけ変更、Go Hostが旧対応のまま | NodeとGo各ローカルPASSでも統合Bindingは不一致。#3485の合成二視点試験と実Plugin/Authorityを分ける | #3485/#3274/#951 |
| **TT-145 Blind test selection** | 既知T1-A=行番号0 ⇄ 独立Reviewerだけが知る新Gap（Sourceと合法Baselineだけ許可公開） | 既出Case再生を新発見に加点しない。独立Expected/実Pinned Run/Known Mutant到達とTool完全性が揃う場合だけ局所T1-B成立 | #3273/#2514/#1180 |
| **TT-146 Source cohort overmerge** | 同時点に見えるConsumer World/Indexを保持し、外部Import Source/Scenario/Tool Replyの版を差し替える | 独立Requiredから導出した必要依存Occurrenceごとの整合がなければ`unknown/stale`。CLI/MCP結果一致から原子的共同Readや現在Grantを推論しない | #3262/#3479/#1163 |

**受入経路と停止条件**：P0-I #3482の独立Required限定Contextを実消費→P0-T #3273真の盲検T1-B（対象1件、独立Holdout）をまず一往復。P0-R #2586の実Provider Readは並行。P1 #3420の残るCLI/MCP 3 Tool契約／#951実Plugin Invoke／#3274 E1→実Host Bindingを別Ownerで実施、P2 #3267 BOM exact TupleとC1/NFR。**各OwnerのCase/Scenario/Oracleを別ファイルへコピーしない**。資格の源泉はSourceと実行Attemptであり、Markdown中の合否欄ではない。

**今回の状態**：main #3403/#3439/#3458/#3477/#3482/#3485、#3440の実装・実証文書と既存PR証拠をGitHubで確認した設計更新。新しい本章のTT-139〜146統合E2E、外部H2/H3、T1-B盲検Agent、正式Plugin/Provider実Invoke、Go C1/NFR、現在のHost Grant/Effect/Owner Accepted Meaningは**NOT RUN / UNQUALIFIED**。Draft #3479/#3365はmain資格へ昇格させない。

<!-- tei-test-adr48-mcp-function-closure-two-source-drift-20261108 -->
## 31. 2026-11-08：ADR-0048共通Operation・実MCP・一Function宣言・二種Source失効をTest Admissionへ（現mainとの差分）

### 31.1 確定できた実績と、これから必要な一辺

| 最新のSource・実装 | 実際に成立した観測・CI | その証拠では成立しないこと |
| --- | --- | --- |
| [main #3439](https://github.com/hat47x/tei-platform/pull/3439) ADR-0048 | 共通の型付きApplication APIを正式な設計方針として採用。CLI/MCP/Web/DesktopをTransport/Presentation Adapterとし、ローカルHTTP常駐や新TEI Hostを必須にしない | 4 Transportの本番実装、全Operation同値、Provider/Host Grant |
| [main #3403](https://github.com/hat47x/tei-platform/pull/3403)／[main #3458](https://github.com/hat47x/tei-platform/pull/3458) | read-only MCP参照AdapterでCatalog/Inspect/Trace/Validateの4 Tool。Validateについて共有operations投影・実CLI／実MCP正常・Source load失敗・意味失敗の一部同値とMCP isError対応を検証 | catalog/inspect/traceの**全失敗・全wire同値**、未知Field/不正型/NULL/上限／MCP protocol全準拠、外部Harness H2/H3の受領・現Disclosure |
| [main #3482](https://github.com/hat47x/tei-platform/pull/3482) | **main exact 8ed583790e63b63527c1dd8f8aecc1d6ce6e3260**／[push実CI #42462861657](https://github.com/hat47x/tei-platform/actions/runs/42462861657) completed/success。13実CLI Reply・従前32＋10の**42 bounded Controls**。sales.order.submit Capability、order_entry Application、入出力型、3 Rule、sales_core Database申告、Loader／Role／Source Definitionを独立Requiredで限定照合。既存Design Context Packet/9、Scenario6＋T1-A/Source-only失効を保持 | **具体FunctionのInvocation/Target Entry/Query→Provider実Read**、原子的13 Reply共同Snapshot、全Business意味、外部Tool受領。boundedFunctionSlice.declarationStatus=bounded-presentでもqualifiedTestGeneration=false |
| [main #3477](https://github.com/hat47x/tei-platform/pull/3477) | 同一WorkspaceのSource末尾LF変更で**World Fingerprint不変／Source Index Fingerprint変更**。World-pinned Scenarioは実行できるが、取得済みDesign Contextの引用資格が失効する実CLI対照 | A→B→A履歴不変・意味再受理、Imported ProviderのRaw鮮度 |
| [Draft #3479](https://github.com/hat47x/tei-platform/pull/3479) | Import先Provider Rawの末尾LF変更により、Consumer World／Semantic／Consumer Source Indexが変わらない場合でもProvider Raw／LocateImportedSourceの現観測は変わるという**別反例を追加**。PR版別の実CIは個別判定 | 新しいImported Provider Readを原子的Snapshotと呼ぶこと、Consumer二指紋一致→Provider Source/Builder Input Closure。現PR未統合・全Source資格なし |
| [main #3478](https://github.com/hat47x/tei-platform/pull/3478)／[main #3485](https://github.com/hat47x/tei-platform/pull/3485) | 実Reference Go HTTP Hostの**10経路**のCallback/Audit・202後panic、E1 PlanとGo HostのRoute/Action差分を限定対照。#3485専用Node/Go [CI #42462531490](https://github.com/hat47x/tei-platform/actions/runs/42462531490) success | 実TEI E1→Host選択・実Provider Bind/Grant、202からBusiness Completion/Effect Receipt、全Path/C1 |
| [main #3440](https://github.com/hat47x/tei-platform/pull/3440) | cmd/tei自身のstage1成果物がstage2 invokerとして作動した**限定L3**、同Child・Artifact一致を独立欄で観測 | Full Source Closure／Seed Trust／Host実行資格／G4。sourceClosureState=rejected・G4=UNQUALIFIEDを保持 |
| [Draft #3486](https://github.com/hat47x/tei-platform/pull/3486) | 旧#3372 closed/unmergedを置換。物理FK/UNIQUE/Index・複合/多重参照・観測Source/版を合成Catalog→Workbenchで保持する製品修正候補 | 現PR HEADの全CI・実PG/Function Read・業務Relation/CRUD意味受理・DDL適用／Receipt |

上表はVersion/Scopeの異なる証拠の**並列表**であり、相互結合した一つのEnd-to-End試験ではない。Product Skillが受領する正規の答えは新Schemaではなく、既存Source/Operation/Scenario/Independent Expected/Evidenceを用途別に投影した結果と、その**不足理由・Claim Ceiling**である。

### 31.2 共通Operationをテスト生成の「取得口」として使う際の受入

ADR-0048の既定フローは **共通Application Operation → CLI/MCP/HTTP/Studio Adapter**。共通化するのは意味対象・入力、Typed Result/Failure、作用、必要な現在資格であり、argv／JSON-RPC数値Error／HTTP Status／画面書式そのものではない。共通APIをWeb ServerやTEI独自Runtimeへ集約しない。

**初回Contractの比較対象は4 read-only Operationだけ**。validateは#3458の投影を基線として、次はCatalog/Inspect/Traceについて同じ許可Workspaceで実バイナリ・Adapterを独立起動し、正常・不在・未対応Kind・Source Load失敗・Meaning Warning・部分・Unknownを比較する。差異はTransport固有表現と明示Reasonだけに許し、JSONの「ok」やTool annotationからMeaning/Disclosure/実Result資格を発行しない。

受入時には **(1) Source/Scope/Versionの意味同一性、(2) fail/error/isErrorの対応、(3) Result Shape/nullable/empty、(4) Recipientに届いたbyte/end、(5) Reader側に許された開示範囲** を別々に測る。参照MCPの入力にpath/rootがないことは任意のResourceへアクセス不能の証明ではなく、実Host隔離も別。#3420の残件（未知欄・不正型・NULL・サイズ制限・initialize/通知・protocol失敗）をそのまま利用する。MCPを必須Hard Gateとせず、T1-Bを同一実CLIで先行検証できる。

### 31.3 Source指紋とSnapshot資格を「何を含む指紋か」で切り分ける

| 変更の位置 | Consumer World | Consumer Source Index | 対象Provider Raw | 判定できる範囲 |
| --- | --- | --- | --- | --- |
| 同Workspace内Sourceの意味不変なRaw変更（#3477） | 同一になり得る | 変わる | 対象外 | Contextの旧Source引用を拒否。World pin実行成功とは別 |
| 別WorkspaceからimportしたProvider Rawのみ変更（#3479） | 同一になり得る | 同一になり得る | 変わる | Consumerの2指紋だけでは新Provider Source鮮度を認定しない |
| A→B→Aで同Rawへ復帰（#3299） | 戻り得る | 戻り得る | 戻り得る | 同byte現在一致と、変更履歴・Owner再受理・Current Grantを区別 |
| Function宣言は同じでProviderのSelected/Loaded/Invokedが変化（#3482/#951） | 同じことがある | 同じことがある | 同じことがある | 実Invocation/Build/Target/Grantの別Witnessが必要 |

よって実Source/Meaning/Build/外部引用を**一つのFingerprintだけで共通失効・共通再利用**しない。Consumer-owned SourceとImported Providerの**Source occurrence/Location/Raw revision**を区別し、必要なProvider実消費入力のClosureを独立Requiredと比較する。Rawが変わったから全Buildを失効させるのも、Worldが同じなら全Consumerの再利用を認めるのも禁止する。新しいFingerprint Service、Source Store、中央履歴Authorityは本計画で設けない。#2600/#3267/#2743/#3262が失効・実利用範囲の既存Owner。

### 31.4 Test Patternに落とす独立反例 TT-158〜TT-169

本節の旧TT-139〜TT-150は§30・§32の既存TT番号と衝突したため、**意味・Expected・Ownerを変えずTT-158〜TT-169へ再採番**した。章番号ではなくCase IDと独立Expectedで追跡し、別の対象を同一TTとして合算しない。

| ID | 一つだけ変える負例と正の対照 | 必要な判定・Owner |
| --- | --- | --- |
| **TT-158 CLI/MCP意味失敗の同値** | 同Workspaceのvalidate正常→重複宣言/Source Load失敗へ変える | CLI exit／Typed failure.codeとMCP isError／意味上codeが一致。#3458 main限定実証。残る3 Operationは#3420 |
| **TT-159 MCP入力を補正してしまう誤成功** | 正規Tool args→未知Field・型違い・NULL・Root変更／Protocol通知異常 | 不正入力はFail-closed、Workspace Root固定。Protocol errorと業務Failureを混同しない。#3420（新全ケース未実行） |
| **TT-160 Function宣言と実Invocation混同** | sales.order.submitのSource/Schema/3 Rule/DB Requirementが全Present、しかしEntry/Provider実選択が欠ける | bounded declaration-present維持・invocation/read/host effectをnot-observed。#3482 main限定実証／実Providerは#2586 |
| **TT-161 PacketのPartialをCompleteへ偽昇格** | 実13 CLI＋42限定Controlsと局所Packet/9が成功したが、全Function/Business/RoleのRequiredが未閉包 | ローカルpositive保持、qualifiedTestGeneration=false／missingを明示。#3262／#2514 |
| **TT-162 同Workspace Source-only drift** | World同一でSource Indexだけ改版したContextを候補生成前後に再使用 | 旧Context引用資格を拒否、旧World Pin Scenario自体の実行成功を否定しない。#3477 main |
| **TT-163 Imported Provider Raw-only drift** | Consumer World/Indexは不変だがProvider Raw改版／LocateImportedSource現在値変化 | 引用・Build input・実Providerの資格はunknown/stale。別Providerを自動的に再実行・再受理しない。#3479 Draft／#2743/#2600 |
| **TT-164 HTTP 202・監査と効果** | 202正常・202送信後panicでAudit Stageがcompleted⇄error | 実Host Path証人は保持、業務完了・Effect Receipt/Client受領はunknown。#3478 main/#1174 |
| **TT-165 E1／Go Host二視点の経路不一致** | E1 method/path/actionだけを変更、Go Host Policyは不変 | 両方の局所PASSを保ちながらCross-Binding不一致を検出。#3485 main限定／#3274 |
| **TT-166 self-buildの過剰なG4主張** | stage1成果物＝stage2 invoker＋artifact一致だがSource closure rejected | 局所L3 witness保持、G4と現在DeliveryはUNQUALIFIED。#3440/#2831 |
| **TT-167 物理FKからの業務Relation誤生成** | Fake Catalogで同対象への2 FK／複合Key・部分Indexを保持、Owner Meaningなし | Physical Observationは保持、Accepted Business Relation/Function CRUDはunknown。旧#3372でなく#3486が実装Owner |
| **TT-168 Harness二重送出・送出上限** | 同じTool内容がモデルInputとSession Log extensionに別載せ／maxBytes超でextension省略 | Raw HTTP request byteとModel/Extension/Sinkを別計測。Session完了≠外部送出なし、公開許可なしに生Sourceを出さない。#3415/#3470/#2515 |
| **TT-169 Legacy一致・Desired欠損の逆転** | 旧Runtime/新Targetとも同じRowを落とす⇄移行後Desiredでは当該Rowが必須 | Legacy Actualの一致とOwner Desired／Target Actual適合を独立判定。旧新同じバグを移植してもmigration-qualifiedにしない。REA #3394／#2514/#2586 |

上記は**既存OwnerへのTest Obligation引渡し**であり、正式な新Scenario/Oracle/Transport Schemaを採用しない。main各PRで実証された局所値と、本表の異なる横断ケースを一つのPASS数へ足し算しない。TT-158のvalidate限定、TT-160の宣言、TT-162の同Workspace Source-only、TT-164のGo Host10 Path、TT-166の限定L3が先行実績。それ以外の**製品実E2E・別Targetの独立Expected・外部H3 Receipt**は再検収が必要。

### 31.5 2026-11-08現在の実装順：最初の有用な返答→本当のT1-B→横断効果

1. **P0-F｜取得根拠の機械可読な一返答**：main #3482の実13 CLI／42 bounded controlsと既存Packetを使い、公開合成sales.order.submitの**宣言Requiredがどこまで届いたか**、**Function actual Entry/Query/Providerが未観測であること**、Source index mode／import/loaderの不足を一つの用途別read-only Briefへ。モデルの独立Required全世界を自称completeにしない。新MCP Tool名やSchemaを仮実装済みとして広告しない。
2. **P0-C｜同じ意味を返すCLI/MCPの局所対照**：main #3458 validate parityを維持し、#3420のCatalog/Inspect/Traceと不正入力のfail-closedを実stdio Adapter／実CLIで追加。返信の字面が完全一致しなくてもOperation結果と同じClaim/Failureを独立比較。Provider/Host/外部H3の成功を待たせない。
3. **P0-T｜T1-BのBlind一件**：#3273の既知line_number=0（T1-A）は除外。#3262の有界Required/Allowed Sourceと、Reviewerが事前固定した非公開Holdout/独立Expectedを分け、生成AIが未知Gapと最小一差分Caseを提案→#1180の既存Scenario→実Pinned CLI Rule結果→#2514/1163独立検証。Source insufficient／Reply partial／未Loadedの場合はCase乱造でなく次の取得行動を提案し、false-qualifiedを測る。§29の#3365 TT-105/106（PR exact HEAD成功・未統合）は局所Receipt目録の根拠にのみ利用。
4. **P1-R｜Product一Functionの未接続一辺**：#2586の実Source Query SQL byte/args→selected physical Provider→Rowset/Field・partial/empty/NULL→Context/Consumerを**同一Attempt**で。#3482のApplication namespace usesを具体Capability invocationに変換しない。Importer Source RawとConsumer版が不一致ならContextの引用・Test Qualificationを分けて停止。
5. **P1-D｜Plugin Inspector／三視点へ派生**：#1251/#951/#3371のSelected Provider B→実Invoked BをCLI/Studioで同条件検査し、暗黙Builtin fallbackを拒否。#3486の物理Metaと#3263/#3303の三視点Data/Function/Business・CRUD、REA Raw→Structural Candidate→Owner Acceptedを段階別に扱う。正式Meaningが未受理なら機械生成したTest PatternのExpectedを確定しない。
6. **P2-E｜Effect/Build/NFR/外部Harness**：#3478/#3485の実Go Host Path、#3440のL3、#3415/#3470の外部Session Log sinkを、実Receipt/Action/Current Grant、G4完全Source/Seed、外部H2/H3実Delivery、C1/NFRと別の評価軸へ。SEI判断移譲・KON/SUI/SHOの追加ConsumerはP0-TのHard Dependencyにしない。

**採用済みと未決**：共通アプリケーションAPIとAdapter方向は**ADR-0048で採用済み**、MCP4 read-only参照入口とvalidate同値はmainに存在する。Feature/Build/Sourceの実完全性、外部Disclosure/Host、MCP全契約、現在Owner業務意味、T1-B盲検生成、実Provider/全C1/G4は**NOT QUALIFIED**。本節は既存GitHub実証から**テスト要件と担当順序を更新した設計差分**であり、新Go/CLI/MCP/PG/Wails/H3実行や受理をこの変更自身で行っていない。新Core型/Host/Registry/Resolver/Parser/Reader/Runner/Oracle/History Storeの重複実装は要求しない。

<!-- tei-test-actual-operation-and-receipt-chain-20261108 -->
## 32. 2026-11-08：受領Step IDだけでなく実Operationへ束縛する――Scenario局所Witnessの強化

### 32.1 新しい実装差分：テスト用の「検証済み=true」を独立の実観測へ戻す

[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365) の初回HEAD `1e2531fede563ed4b1d32cca0e78ccaf69fe364c` は、[専用Dogfood #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357)、[Health #42394673331](https://github.com/hat47x/tei-platform/actions/runs/42394673331)、[依存確認 #42394673273](https://github.com/hat47x/tei-platform/actions/runs/42394673273) の3件が**completed/success**。ここまでの実証は、既存6件の実TEI Pinned CLI Stepと、そこから作った7件の試験用受領目録対照である。

追加調査で、旧 `scoped_step_selftest` は各Stepの実CLI Evidenceを `measured` に保持した後、目録を `{"stepId": ..., "verified": true}` と**試験側で再構成**していたことが判明。これでは後続の「局所Observed」は既存 `expect_real_evidence` の成功に間接依存するが、実Receiptの `verified` 値そのものには直接結合していなかった。また同じStep IDが別のOperationを意味する場合に、目録が正しいTargetへ到達したとは限らない。

[同PR追加commit `8adbf0d` → `1ffcba5` → `b81a764`](https://github.com/hat47x/tei-platform/pull/3365) では、**実TEI CLIが返した `stepId`／`operation`／`verified` をそのまま `measured` へ引き継ぎ**、既存Scenario Sourceで指定されたStep→Operationの対応と照合するよう変更。Step IDだけを同じにして `evaluate-validations` を `resolve-pattern` に差し替えた負例は `operation-mismatch` として局所認定を拒否する。従来のTT-105/106＋受領目録対照7件にこの1件を加え、**test-only 対照8件**を同一WorkflowでAssertion対象にした。

**重要な限界**：ここでの `end_received=True` は試験Consumerが渡す**合成終端フラグ**であり、外部Harnessのordered Tool part/endを観測した証拠ではない。Step実行の詳細Resultは既存 `expect_real_evidence` が照合した内容の範囲でのみ信頼し、`scoped_step_coverage` 自体を新Oracleとみなさない。Operation一致は**宣言されたStepとの整合性**であり、Providerの実Load/Invoke・Host Actor・Effect Receiptを証明しない。新変更HEADのCIは追加時点NOT RUN、旧HEADのsuccessを引き継がない。

### 32.2 Evidenceを「同じ名前」ではなく、必要依存の各辺で束縛する

他チャットで成熟した、外部Sourceの `knownFrom` とTEIの `observedAt` の分離（[Draft #3380](https://github.com/hat47x/tei-platform/pull/3380)）、Imported Provider Sourceだけの変化（[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)）、宣言した配備Routingと実Executorの断絶（[merged #3474](https://github.com/hat47x/tei-platform/pull/3474)）、HTTP 202/Host監査と資源効果の分離（[merged #3478](https://github.com/hat47x/tei-platform/pull/3478)）を、テストのClaim別の必須辺へ適用する。

| Witnessの辺 | 本当に別に必要な観測 | これだけでは不適格な主張 |
| --- | --- | --- |
| Source／受理 | Source原byteとoccurrence、Loader実消費、必要なRole/Relation、意味受理のScope。Imported Provider原Manifest/取得時点はConsumer World/Indexと別 | Fingerprint一致＝全依存同版、外部Sourceの履歴・受理済業務意味 |
| Test selection／実行 | 独立Required、Scenario Source digest、Step ID、**Step→Operation**、実選択/Run Target、実Receiptのverified/Outcome | Case名・Test名・exit0だけでTargetへ到達、AIが未知Gapを新発見 |
| Time／cohort | 外部記録のknownFrom、TEI観測のobservedAt、実Attempt時点、Consumer受領時点、Current Grantの有効時点を必要範囲で照合 | 過去のSource時点＝当時TEIが知っていた、同一日時文字列＝原子的Snapshot |
| Consumer／Delivery | CLI/MCP/Studio等のActor・Audience、ordered Tool Reply/end、Payload bytes、Session再開とSinkの保存・送出 | 内部Go/CLI PASS＝外部Agentに完全配送、Provider read-only＝Sinkも無副作用 |
| Provider／Effect | Selected/Built/Loaded/Invoked、実Provider結果・Scope/粒度/NULL/partial、CommitUnknown時の独立Receipt/Read-back | Route plan＝実Routing、HTTP 202／Audit completed＝業務完了、BOM検証済＝Host Grant |

許されたScopeで独立に検証した局所正のWitnessは残す。ただし**一つのReply/Source/Stepに未証明の辺があれば、当該上位Claimだけをunknown/unqualifiedに止める**。外部Source改版で全Workspaceを無条件失効させない一方、World二Fingerprintの一致だけで当該Importに依存するテスト結果を再利用しない。

KON Dataspaceは独立・任意のData Source候補であり、TEIのテスト基本能力の必須依存にはしない。SHOの文書投影もTEI Accepted MeaningやTarget実行の第二正本にしない。既存#1163 Evidence、#2514独立Expected、#3262 Design Context、#1180 ScenarioのAuthorityを維持する。

### 32.3 小さな次の独立対照（既存担当へ割当、製品実行を新たに認定しない）

| ID | 一か所を変える対照 | 判定と実装範囲 |
| --- | --- | --- |
| **TT-147 receipt target binding** | 実Pinned CLIの `stepId` はそのまま、`operation` だけ別の既存Operationにする | 同じIDでも `operation-mismatch`。#3365の試験用メモリコピーでコード化、**追加HEAD CI未検収** |
| **TT-148 receipt verified provenance** | `verified` を実CLI Evidenceから抜く／falseにする | 該当Stepの局所正証人を認定しない。合成対照は既存。外部受信者の署名/真正性は未実証 |
| **TT-149 temporal knowledge** | 上流Source `knownFrom` を昔にし、TEI `observedAt` だけ後日にする | 当時TEIが認識したという時点Claimを拒否。Draft #3380は別Target |
| **TT-150 imported Source/Provider only** | 外部Provider Manifest原Byteが変化、Consumer World/SourceIndexは不変 | Imported Source依存のClaimだけ再資格化を要求。Draft #3479へ委譲 |
| **TT-151 declared routing but not invoked** | Binding/Route planは正しいが実Executorは別経路を使う | plannedとactual runを別票。#3474の静的観測を実E2Eとみなさない |
| **TT-152 HTTP accepted≠business committed** | HTTP 202とAudit completedはあるが、Command Receipt・Client完全受領なし | 技術的応答とEffect/業務成功を分離。#3478の有限Host試験をG1/G2へ無条件に合算しない |

### 32.4 実装順序と停止線

**P0-1** #3365の追加HEADで、実CLI `stepId/operation/verified` が目録へ転記され、wrong-Operation負例が必要な理由で拒否されることをCIの同一Attemptで検収。**P0-2** #3273/#2514の本来の未知Gapを一つBlind Pre-registrationし、#3262の既存13 CLI等のContextを必要Subsetだけで使ってT1-Bへ進む。実Provider Readは#2586、Import Provenanceは#3479と並行。**P1** #2509で外部Tool End/Disclosure/Session Resume、#951/#1251で実Plugin Invocation、#3274で実Host/Client/Effectを別資格化。**P2** Go Code C1、Target version 2×2 BOM、NFRはそれぞれの独立分母で測定。

**今回実施した範囲**：PR #3365の既存test-only Code・専用CI Assertionの変更、および本計画への追記。既存初回HEADでの3 CI successは確認済み。追加HEADの結果、製品全体Coverage、T1-B盲検、外部H2/H3 Tool受領、Provider実Invoke、現在Grant、正式Owner Meaning、Go C1/NFRはこの文書だけでは認定しない。


---

<!-- tei-test-20261108-coverage-provenance-direct-binding -->
## 33. 2026-11-08 実コード照合で発見したCoverage証拠の出所ギャップ（TT-153〜157）

### 33.1 #3365の限定成功と、まだ未証明の直接結合

[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365) のHEAD `b81a7642b737f96a29e665807b55868907486d04` にある `tools/reference-issue-test-ability/probe.py` を直接確認した。`expect_real_evidence(result, step)` は実CLIの`executionMode=generated-pinned`、`selectedStepCount=1`、単一`evidence[0].stepId`、`verified is True`を検査する。一方、`scoped_step_selftest`はその後、`stepId/verified/operation`の3欄だけを**試験用Receiptへコピー**し、`scoped_step_coverage`へ渡す。

`scoped_step_coverage`は`required`、`scenario_sha`、`actual_source_sha`、`end_received`、`expected_operations`を**呼出元が与える値**として受け取る。自身で実CLIプロセス・Scenario Source・Result Body・Actual Input・StepのOperationを再照合しない。ここで`verified:true`があっても、その行が**指定Stepの指定Operationを実行した同一Attemptの実結果**だとは、投影関数単独では証明できない。これはread-only/test-onlyの関数責務としては妥当だが、Product Coverage Engineの受入証拠に昇格させると危険である。

特に、`expected_operations`と`row.operation`が**同じ誤値に変更**された場合、投影関数は両者の一致しか見ない。外部で固定したScenario定義と実CLI Resultの間の束縛は別途必要。逆に、独立に実証された他Stepの正証人を、別の未受領Stepのためにすべて消す必要はない。既存TT-105/106の「局所正の証人」と「不在証明の拒否」は維持する。

### 33.2 新たな独立反例（既存Scenario/Oracle正本を変更しない）

| ID | 変更する一軸 | 期待する診断・禁止する主張 | Owner |
| --- | --- | --- | --- |
| **TT-153 Receipt出所のすり替え** | `verified:true`とStep IDだけコピーした別AttemptのReceiptを混入 | `verified`だけでは`observed-in-current-attempt`に昇格不可。実CLI invocation/Source/Scenario/Resultへの参照と同一Attemptの一致が必要 | #3365/#1180/#2514 |
| **TT-154 Operation共謀誤値** | `expected_operations`とReceiptの`operation`を同じ別Operationへ置換 | 自己一致はPASSし得るが、独立Scenario SSOTのStep→Operation固定値と実Resultの照合で拒否 | #3365/#1180 |
| **TT-155 正常Stepの証拠と異常Receipt混在** | 正常Step Receiptは独立に検証済み、別Stepだけ重複/不正 | 不正行を含む**目録全体**はinvalid。ただし個別Attemptで独立検証済みの正常Step Witnessは、Source/Identity/Disclosureが比較可能なら別途保持。現`scoped_step_coverage`の全消去挙動を全体Coverage意味へ流用しない | #3365/#2514 |
| **TT-156 終端自己申告** | Callerが`end_received=True`、実外部受信にはend/ACKなし | `complete-for-one-pinned-scenario`を外部Tool Delivery完全性に昇格不可。受信Stream/Session/Count/ACKの別Witnessが必要 | #2509/#3365/#2515 |
| **TT-157 Function存在とRule到達の乖離** | main #3482のCapability/Application/Source Requiredは一致、実Scenarioが別Operation/Ruleを評価 | Function `bounded-present`を実Rule Coveredにしない。Scenario→実Target→Rule Evaluator→Independent Expectedの同一Attempt連鎖を確認 | #3262/#2586/#1180/#2514 |

### 33.3 実装への接続順序

**P0-1**：#3365に新しいCoverage/Receipt Authorityを作らず、既存実CLI結果と#1180 Scenario SSOTの**外部固定Operation**を照合する一つの直接負例を追加する。Fixture改変のみの局所テストと実CLI/実Runner改変を区別する。試験側が作った`verified:true`をProductの真正Receiptと呼ばない。

**P0-2**：局所`observed`を`Step/Operation/ScenarioRevision/ActualSource/Attempt/Result/IndependentExpected`の組として説明できる場合だけ、限定肯定を出す。複数Consumer間の同一性を検査する際も、単なるJSON Shape一致と実行同一性を分ける。

**P0-3**：#3273のBlind T1-Bは、独立未提示Gapの発見に加え、生成Caseが**実対象Operation/Ruleへ到達したこと**を判別力の分子条件にする。実到達不能なCaseは生成件数に含めても発見成功件数に入れない。

**P1**：#2509の外部Harness終端・ACK、#2515のDisclosure/OS送出、#2586の実Provider、#3420のCLI/MCP Transportは各OwnerのEvidenceを参照する。新しいReceipt Store/Runner/Authority/第二Coverage Engineは作らない。

**資格**：本節は実コードの静的照合と既存PRの試験結果に基づく**設計上の欠落指摘**。TT-153〜157の新しい実Go/CLI/外部H3テストは今回**NOT RUN**。#3365はDraft/未main統合であり、局所test-onlyのPASSを製品の実行証明に昇格させない。

<!-- tei-test-lrh-evidence-provenance-20261108 -->
## 34. 2026-11-08：実行結果の出所とHandoff対象集合を分けるTest Evidence契約

### 34.1 結論：観測が三段階に分かれるときの「覆われた」の意味

他チャットのHandoff/三視点モデルで確立した **L/R/H**（L=業務上選定された対象、R=実Providerが取得した集合、H=外部Consumerが受領確認した集合）を、テスト生成とCoverage判定で**独立した対照集合**として使う。集合の同一性は名称や件数一致ではなく、**Subject/Relation ID、Source/Provider revision、Predicate/Scope、時点、Grain、Recipient/Attempt**で照合する。空集合でもHandoff確認がなければHはunverified。宣言されたLが正しいこと、Rが実際にすべて取得されたこと、Hが相手に届いたことは三つの異なる証拠であり、単一ReceiptやCI成功で合算しない。

| 試験の問い | 必要な観測と期待する診断 | 所有する既存Owner |
| --- | --- | --- |
| Lの義務が何か | Accepted Meaningの範囲・Revision、Relationの多重性・時間・NULL、Function/Action/Ruleと否定依存。未受理/partialならLの全分母はunknown | #1120/#1126/#2514/#3262 |
| Rは対象を実取得したか | 実Providerのselected→loaded→invokedを区別、実SQL byte/Bind・Predicate/Scope・Source/Manifest/Rowset/Result completeness・Snapshot。Rにない対象を「DBに存在しない」としない | #951/#1251/#2586/#2506 |
| Hは実受領されたか | Consumer/Host/ToolのRequest/Reply ID・最後のPart/End・Audience/Disclosure・ACKを同Attemptに束縛。0件でもACK欠落なら未確認 | #2509/#2515/#1163 |
| 実行Targetに到達したか | Source/Scenario Revision、既存verify/generate/checkのPin、実Step ID/Operation/Evidence、Independent Expectedの一致。Policy/Grant/Effectも必要なClaimだけ別途照合 | #1180/#2514/#1174 |
| これらを再利用してよいか | Content/Provider raw/Toolchain/Rule policy fingerprint/Build Artifact/Consumer Scope/Current Grantのうち必要依存だけを再照合 | #2600/#3271/#3267/#1163 |

このL/R/H構造は独立の業務Expected・二次データストア・新Provenance Registryではなく、既存データ構造と観測Receiptを比較する**テスト判定の観点**。データ源が異なるときの同名snapshot token、設計と実行で違うProvider、事実として未公開のHは結合しない。Provider Rawのみ変更してConsumerのWorld/SourceIndexが同じケースは既存[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)の別版証拠を要し、「同じ指紋でFresh」には昇格しない。

### 34.2 TT-105/106の成果を実Resultへ繋ぐ境界

[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)の最初のexact HEAD `1e2531fede563ed4b1d32cca0e78ccaf69fe364c`は、[専用CI #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357)・[Health #42394673331](https://github.com/hat47x/tei-platform/actions/runs/42394673331)・依存確認 #42394673273がcompleted/success。既存実Pinned CLIの6 Stepを観測し、**試験用受領目録の7 Controls**で未受領・部分・End欠落・重複・未verifiedを「未テスト確定」へ変換しない局所結果。PRは**Draft/未統合**であり、あとから追加した8番目の`same stepId / wrong Operation`負例、Sourceの実行後再読取とそのHEADは**新CIの確認を要する**。

実Step入場条件では、Test側が`verified=True`を自作してはならない。実CLI EvidenceのStep ID/Operation/verifiedと元Scenarioの期待Operationを照合し、**名前が同じ別Operationの結果を転用しない**。Source digest比較も同じ変数の自己一致では証拠にならないので、生成前と実CLI完了後の別Readを区別する。ただしこの二Readの一致は途中のA→B→A、実Binaryが読んだbyte、Runtime/Host Grantを証明しない。Test-only受信Endは実外部ProtocolのEOF/Deliveryとは別。

次の独立の同一Attempt実験は、公開一Functionの独立Expectedを事前固定し、L対象ID、R実Query/Provider Rowset、H実受領を**それぞれ正/部分/空/未確認の対照**として観測する。Rが空でHも空でもHandoff ACKが欠ければ「実受領済の空」としない。Rがpartialで正の一件が確実ならその一件の局所観測は維持し、Lの全Coverage/不存在はunknown。BOMとPluginの`selected`だけではR=acquiredにせず、Owner Accepted/DB実Readがない段階で本番の意味適合を示さない。

### 34.3 直近の実装順・証拠の停止線

1. **P0 既存Codeを検収**：#3365の最新HEADに対し実6 Step Evidence、8つの限定copy-only負例、独立Scenario期待Operationの一致、実行後Source Readを同じ専用CIで検証。Go code / Workflow CI / Repo Healthを別票にし、他PR HEADの成功を移植しない。
2. **P0 担当差分を縮小**：Design Context #3262の13実CLI/42局所Controls・一Functionの宣言Sliceを基準にする。宣言されたCapabilityは実Provider選択/実Call/実Readではない。正しい正常Array、空の`[]`、subject/index mode、root/fingerprintの範囲を意図的な欠落と区別（CLI #3370 / #3467）。
3. **P1 L/R/Hで一Function**：#2586/#2509と共有する一つのFunction/Operation Scopeで、Provider実Read (R) と受信Consumer (H) を別接続し、その間の例外・欠落・重複・時点混合を故障注入。外部Toolのログ保存・別Recipient開示は別資格。
4. **P1 検証基準の独立性**：T1-Bの未提示Rule/Gap・Independent Expected/Target Mutantは#2514が保有し、TT-105コピー負例を「AIが未知Gapを発見した証拠」としない。正常Provider/Target到達、実Action/Effect・未知の依存を隠さず採点する。
5. **P2 領域拡張**：Plugin/Trait/BOMのRule policy fingerprint、Graph/CRUD/REA→SHOの多重Identity、Go HTTPの202/AuditとEffect、KON/SUI/SEIの二Consumer差分を既存Ownerの局所Claimに束縛する。C1/NFR/完全G4を分母なしの百分率で報告しない。

**状態**：本節は#3365の旧exact-head CI成功と新変更の未検収、および既存モデル/Source/Provider/Handoffの計画差分を統合したもの。L/R/H実Provider/実外部Handoffの同一Attempt、完全なSource/BOM Input Closure、現在Owner/Host Grant、T1-B、全Product C1/NFRは**NOT RUN/NOT QUALIFIED**。第二Runner/Reader/Source Store/Host/Authority/Oracleは追加しない。


---

<!-- tei-test-gate-checklist-20261108 -->
## 35. 2026-11-08：文書の増殖を止め、実証の受入ゲートを固定する（P0実行計画）

### 35.1 設計上の成熟度と実装上の成熟度を分ける

§1〜34のTTは、独立Expected/Scenario/Source/Deliveryの反例を十分に抽出した**設計候補**であり、番号が増えたことを実装進捗に数えない。これ以降の通常運用は**一つの実装PRにつき一つの新しい判別可能な負例**を優先し、未検収の章・TTを無制限に増やさない。既存Ownerに受入基準を引き継ぎ、同じテスト・実行結果を複数の設計書にコピーしない。

### 35.2 次の一件の厳密な実行票：独立Scenario Operationへの到達

| 観点 | 受入条件 |
| --- | --- |
| **Given** | #1180の既存Pinned Scenarioから、対象Step IDと**独立に固定した**期待Operation/Source/Scenario revisionを取得。実CLI binary/revisionと独立Expectedは#2514で固定 |
| **When 正常** | 同一CLI Attemptで指定Stepだけを実行し、実ResultのStep ID/Operation/verified/Targetと、外部固定の期待Operationを比較 |
| **When 異常** | `scoped_step_coverage`に渡す`expected_operations`とReceiptの`operation`を**同じ別Operationへ置換**。関数内自己一致だけでは通る場合があるため、独立Scenario Sourceに対する外側の検証が**明示的に拒否**すること |
| **独立Oracle** | 変異していない#1180 Scenario Source（実Byte/Revision）と#2514 Expected。生成者がコピーした`verified`や`end_received`を独立Expectedに使用しない |
| **出力** | actual binary/Scenario/Source revision、対象Step、実CLI invocation、元Result evidence、期待Operation、観測Operation、判定理由、CI exact HEADを含む最小Receipt |
| **合格上限** | `one-step-operation-bound`。Function全Rule Coverage、実Provider R、外部Consumer H、Host/Grant、C1、AI T1-Bは`unknown/unqualified` |
| **停止** | Source改訂が異なる、実CLI Replyが不完全、期待Operationの独立固定がない、Targetが未到達、CI exact HEAD未完了の場合、PASSではなく`ineligible`または`insufficient-evidence` |

この一件は既存[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のtest-only投影を製品のCoverage Engineへ拡張する提案ではない。既存#1180 Driver Scenarioと実CLIの境界で比較する。必要ならテスト専用Assertionのみ追加し、Production APIや新Runner/Registryは追加しない。

### 35.3 2026-11-08：exact-head検収と、実Read一機能の停止線（後続main統合を反映）

| 対象 | 実際に確かめた範囲 | まだ成立しない範囲 |
| --- | --- | --- |
| [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365) HEAD `e8745c6cbb7596da48da62becdb48417f3ce8dd6` | [専用dogfood #42465435223](https://github.com/hat47x/tei-platform/actions/runs/42465435223)、Health #42465435270、依存 #42465435235は**completed/success**。同一実CLIで13 Reply・42限定Context対照・6 Pinned Step・9 test-only目録対照、TT-154共謀誤値を独立Operation期待で拒否。Go `cmd/tei`局所statement C0相当9.4% | Draft/未main、C1/全Rule Coverage/実Provider/Tool実外部受領/AI盲検T1-B/Owner意味受理は未資格 |
| [Draft #3500](https://github.com/hat47x/tei-platform/pull/3500) HEAD `40c1acc5a2079d4e71aa9a2a9e9191779a3f48ba` | 実Native Organization Case Sourceを既存Pattern→Resolver→Query→PostgreSQL Executor→明示Semantic Column Mappingで接続。instrumented SQL Connectorによる[専用Go CI #42469646869](https://github.com/hat47x/tei-platform/actions/runs/42469646869)およびRepo Health #42469646964はsuccess | **実PostgreSQL Case試験は環境変数未設定でSKIP**。Function Data Context・実Provider Grant・Consumer Handoff・業務意味は未接続。別のStudio Wails系2ジョブにはfailureがあり全ジョブsuccessではない |
| main [#3482](https://github.com/hat47x/tei-platform/pull/3482)／[#3501](https://github.com/hat47x/tei-platform/pull/3501) | 前者は `sales.order.submit` Capability/Application/Sourceの限定宣言照合。後者は実WorkspaceのSubmit Pattern/Capabilityを既存Inspectorでそれぞれ取得し、架空ActionのSource不存在も識別 | **Submitの実Invocation・Host Effectの証拠ではない**。上のBrowse/Case実Readの証拠をSubmitへ移植しない |
| **main統合 [#3479](https://github.com/hat47x/tei-platform/pull/3479)**（merge `99635b8759a08b625e73eaea7813f6512dfd15da`） | Consumer World/Source Indexが変わらなくても、Imported ProviderのRaw Sourceのみ・Manifestのみ・両方の改訂を別々に観測し直す必要がある反例。PR HEAD `6f0a9735`の[Health #42470247004](https://github.com/hat47x/tei-platform/actions/runs/42470247004)・dogfood #42470246942、および後続[Health #42474407926](https://github.com/hat47x/tei-platform/actions/runs/42474407926)はsuccess | **mainへコードが入ったこと**と、実Consumerが同じAttemptで外部Raw/Manifestを再取得して使ったことは別。原子的Nested Input Closure・現Grant・Build/実Invokeは未証明。main merge commitの独立push実験とPR HEADは同一試行ではない |
| **main統合 [#3516](https://github.com/hat47x/tei-platform/pull/3516)**（merge `26bf47e12e17ebdf87ec2e2f6b4cb23aed643e4d`） | 実Workspace→Canonical認可→実Go Host→既存Driver→**合成Adapterの実Invoke**を同一HTTP試行で検証。HDG-I01〜I07の認可／CSRF／解決障害／呼出し障害／Session／不正Resultを、[限定Node/Go CI #42474818698](https://github.com/hat47x/tei-platform/actions/runs/42474818698)の実Goログで7/7 PASS。#3501/#3506からの段階的証拠 | **Host→Driver→test-only Adapter到達**の局所正証人であり、実SQL Provider Read／本番Plugin Build/Grant／DB更新・独立Read／Client完全受領／業務Meaningではない。#3516 HEADの[全体Health #42474818899](https://github.com/hat47x/tei-platform/actions/runs/42474818899)は**completed/failure**。実ログは`gofmt`/`go vet`/Workspace検証PASS、`go test ./...`の`TestPlanReadmesMentionEveryEntry`のみFAIL（AIチューニング設計文書が`plan/design/README.md`の一覧から欠落）。[修正Draft #3521](https://github.com/hat47x/tei-platform/pull/3521)で索引1行を補正し**新HEADのCIは別検収**。7経路実Goの専用成功と、全体Gateの失敗を両方保持 | 
| main [#3502](https://github.com/hat47x/tei-platform/pull/3502)／[Draft #3505](https://github.com/hat47x/tei-platform/pull/3505) | Trait/Pluginの独立Consumer検収条件を既存設計へ反映。図表/ナビゲーションを標準製品要求RQ-036の候補として、全体Map・標準/拡張/変更可能域・未取得/非公開を明示する受入をDraftで整理 | 現段階で実Plugin二Consumer Invoke、実Diagram/Navigation生成やユーザー操作試験は済んでいない。Mapで見える範囲≠全Domain Requiredの閉包 |

**Test Admissionの三種類の判定を分ける**：`passed`はそのHEAD・Scope・Attemptの実験が通過した場合だけ、`blocked`はSource/Grant/受領/CI整合などの前提欠落、`failed`は対象に到達した実検査が独立Expectedへ不一致だった場合とする。リポジトリ全体の`failed`をそのままBusiness Functionの失敗へ転写せず、局所成功を全体`passed`にも昇格しない。Skipped実DB試験は`blocked-or-not-run`であり`passed`でない。今回の#3516のIndex漏れを[#3521](https://github.com/hat47x/tei-platform/pull/3521)が修正する予定だが、後続CI未完了なら完了扱いしない。

**実行証拠の二系統を合算しない**：P0-RはBrowse/`order_entry.recent_orders`の**実選択Query→実Provider Read**、P0-IはCommand/`order_entry.submit_order`→`sales.order.submit`の**Host→Driver→Action Adapter Invoke**。#3516のP0-I限定到達は前進だが、これをP0-RのPostgreSQL Rowset観測やSubmitのDB Commitと呼ばない。どちらも同一Source Revisionというだけでは共通Execution Attempt・権限・業務Expectedを共有しない。

**次のP0-Rは `order_entry.recent_orders` の一機能・一Attempt**に限定する。main上に存在するBrowse Pattern、`recent_orders.tei`、SQL実ファイルとDatabase/QueryのBindingから始め、Source/Query/選択Provider/実SQL+args/Rowset/Function Data Contextを独立Expectedへつなぐ。#3500のOrganization Case実Executorは「既存実装経路がある」という**補助証人**であって、別Domainの`order_entry.recent_orders`実Readを実証しない。 `sales.order.submit` は**別のCommand/Effect実証**として#1174/#1127に接続し、Browse正例から業務登録成功へ昇格しない。

| HJ-09の既存Ownerに帰属する受入辺 | 必要な正の証拠 | 同じAttemptで落とすべき一つの負例 |
| --- | --- | --- |
| **D／設計Source** #3262/#2387 | 実Loaderが読んだPattern、Query、SQL fileとSource role・正確なrevision | Query IDは同じだが必要SQL Rawが改版／未ロード |
| **B／実選択・Binding** #951/#2622 | `sales` Datasourceが実Provider・対象DBに解決された証拠 | 同名Provider Aを宣言、別Provider Bが呼ばれた |
| **R／実Read** #2586/#2506 | 同じRequestから実ExecutorのSQL/args、実Row/Column、Mapper ResultとContext粒度が追跡可能 | Fake Connectorだけが行を返し、実DBを読んだと報告する |
| **S／完全性と意味** #2497/#2514 | 独立ExpectedでRowsetのgrain、必要列、null/欠損、時点・取得Scopeを比較 | 1/3部分Rowsetまたは0件を全件・不存在へ昇格 |
| **H／受領・表示** #2509/#2515/#3505 | 実Consumerに届いた受領範囲・End/Grantと、Map上での局所PASS/unknown表示 | UI/図では全体が可視に見えるが非公開・未取得の依存がある |

**試行の段階**：instrumented Connectorでの実Query/Mapper実行は`executor-mapping-bounded`、隔離PostgreSQLのSKIPなし実Readは`physical-read-observed`、必要Scopeを備えたFunction Data Contextと実Consumer受領はそれぞれ追加の独立資格。後段が不明でも前段の正証人は保持する。Goの局所statement C0とC1、Rule/Requirement Coverage、Product全体のE2、Owner Meaningはこれらから推論しない。

### 35.4 T1-Bへの昇格を認める条件

このP0 Operation負例が独立に検出され、実CLI/Scenario/Source/Resultの資格が揃った後にのみ、#3273で未知Gapを生成AIへ隠したBlind Trialを実施する。**新規テスト候補が生成されたこと**、**指定Targetへ到達したこと**、**独立ExpectedとMutantを判別したこと**、**実外部受領が成立したこと**を別列で記録する。候補の多さやドキュメント追記数は成功指標にしない。

**この章の実施状態（更新）**：#3365のGo/CLI既存検査と9つのtest-only controls、#3500のinstrumented SQL Connectorによる実Go検査は各exact-head CIで限定PASS。一方、**本章のHJ-09 `order_entry.recent_orders` 実Provider／Function Context／Consumer接続とT1-B Agent TrialはNOT RUN**。外部H2/H3独立完備比較、Source/Grantの同一Attempt、C1と正式意味受理も未資格。

<!-- tei-test-independent-operation-oracle-20261108 -->
### 35.5 独立Operation Oracleの実コードへの反映と次の合格線（2026-11-08）

[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)の旧HEAD `f7de811d0ac6c92c696e7f36069ae47350cca5f8`は、[専用dogfood CI #42463616468](https://github.com/hat47x/tei-platform/actions/runs/42463616468)・Repo Health #42463616418・依存確認 #42463616407 がcompleted/success。これは**実CLIの6 Stepと8つのtest-only copy controls**を含む局所証拠であり、その後の新実装の受入ではない。

新実装は、`probe.py`の公開6 Stepに対する**独立固定Step→Operation表**（`STEP_OPERATIONS`）を導入し、以下を同一実Pinned Scenarioの検査へ接続した。

- **正の条件**：Sourceから読んだScenario Step→Operationが固定Referenceの独立表と一致すること、実CLI Evidenceの`stepId`/`operation`/`verified`が要求と一致すること、実行後Scenario SourceのByte digestが読取時と一致すること。これらは実Provider/Host/異時点の完全な共同Snapshotではない。
- **判別負例 TT-154**：受領目録の`operation`と呼出元の`expected_operations`を**同時に別Operationへ変更**すると、`scoped_step_coverage`内の自己一致は見かけ上PASSする。だが**同じ独立Operation Oracleを実際に呼び出す**と拒否する。期待値と観測値を一緒に作る同源の検証基準では独立性を満たせないことを、test-only copyで対照化する。
- **CI受入**：既存7→8→**9つの限定Control**（TT-105/106の部分目録、終端、Source差、重複、未Verified、異Operation、TT-154共謀誤値）を、実6 Stepの成功後に現行WorkflowでAssertする。数値はProduct Mutant kill件数でも全Requirement分母でもない。

この新機構は**独立に固定した合成Referenceに対する検証**であり、業務Meaning Owner Acceptance、Scenario→実Function→Provider、実外部Harness end/ACKを証明しない。C1、Blind T1-B、Current Grant/Disclosureは引き続きunknown/NOT RUN。

**採用順（最新）**：#3365のexact HEAD CIにおける9対照・実6 Stepの限定成立は**2026-11-08のRun #42465435223で確認済み**。ただしDraft/未mainなので製品に統合済みとはしない。#2514/#1180へはこの限定受入範囲だけを伝える。次のP0-Rは本§35.3の `order_entry.recent_orders` 読取を先行し、`sales.order.submit` のCommand/Effect系とはOwnerもExpectedも分離する。宣言だけと実行を混ぜず、実呼出が未確認なら`not-observed`で止める。外部Import ProviderのRaw-only drift (#3479) とCLI/MCP参照Transportは、この同一Attemptの外側の別検収条件とする。**新Core/Reader/Runner/Oracle正本を作らない**。

<!-- tei-test-single-read-multi-purpose-admission-20261108 -->
### 35.6 実Readの一つのAttemptを、テスト・性能観測・表示の別資格に使う（2026-11-08 横断更新）

**今回の結論**：テスト生成、AI性能チューニング、Function Data Context、Studio/SHO表示が同じ業務データを使う場合も、**別試行で得たSource/Provider/Replyを一つの「実行済み」へ合成しない**。共有するのは新しい証拠正本ではなく、既存のSource/Operation/Provider/Context/Result/Receiptに付く**参照可能な同一Attempt識別子と必要条件の照合手順**とする。#2586のP0-R `order_entry.recent_orders`を先行Targetに固定し、Submit系のCommand/Effect実行とは分ける。

#### 35.6.1 先行実装の確定と範囲

| 実装/Owner | 確認できた限定成果 | テスト・AI観測へ流用できない未実証 |
| --- | --- | --- |
| [main #3479](https://github.com/hat47x/tei-platform/pull/3479) / #3262 | Consumer World/SourceIndexが同じでもImported ProviderのRaw/Manifest改訂を別途再観測すべきという実Go反例をmain統合 | 原子的Nested Input Closure、現在のGrant、実Consumer Load/Invocation、再資格化完了 |
| [main #3516](https://github.com/hat47x/tei-platform/pull/3516) / #3274 | 実WorkspaceのCanonical/認可→Go Host→既存Driver→**合成Adapter**へ到達した一HTTP試行。専用Go/Nodeで7 Caseの限定証拠 | 実SQL ProviderのRead/Write・本番Plugin Build/権限・Business Effect/外部Client完全受領 |
| [Draft #3500](https://github.com/hat47x/tei-platform/pull/3500) / #2586 | **別Domainの**Native Organization Case Query→実Resolver/既存Postgres Executor/Mapperにinstrumented Connectorを接続したGo試験。オプトイン物理PGも実装候補 | `order_entry.recent_orders`自身の実Read、物理PGのSKIPなし確認、Function Data Context/消費先同一Attempt。Draftまま |
| [main #3520](https://github.com/hat47x/tei-platform/pull/3520) / #3271 | Design Hint/申告されたProvider呼出・Statistics/Delivery/Disclosureの**19件の合成照合**を実装・main統合。専用CI #42476197836成功 | 実TEI Provider呼出・統計採取・外部Tool完全受領・性能改善。対応するHEADの脆弱性確認#42476197787はvuln DB取得時の通信失敗でworkflow failure、安全性PASSでない |
| [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365) / #2514 | 実6 Pinned Scenario Stepと9 test-only目録対照、独立Step→Operationを[exact HEAD CI #42465435223](https://github.com/hat47x/tei-platform/actions/runs/42465435223)でPASS | 6 Stepの外側のRule/Requirement分母、実Function Read、外部H2/H3配送、AIのBlind新規テスト発見 |

#### 35.6.2 P0-Rに渡す最小受入票（別のProduct Schemaにはしない）

同じ`order_entry.recent_orders`の一試行から、既存Ownerの記録を参照して次の属性を**個別に観測**する。すべてを一つの巨大なPacketに強制するのではなく、どの属性が同じAttemptで得られ、どの属性が別時点/未取得なのかを診断する。

| 辺 | 実際に観測するもの | 反例／保留 |
| --- | --- | --- |
| D（Source） | 実Loaderが消費したPattern・Query・SQL原Byte、Source/Role/Revision。外部Importが必要ならProvider Raw/Manifest occurrenceも別束縛 | Consumer World/Index同一でも実SQL/Import Rawが改版、またはSQL Role未取得 → stale/unknown |
| B（選択） | 実Request/Scope、QueryとDatasource alias、**選択されたProvider instance**、実呼出し先の関係 | Declared BなのにCalled A → 不一致。等しいRaw Rowsetでも一致認定不可 |
| R（実Read） | 実Executor/Statement+args、物理Column/RowとMapper出力、欠損/NULL・grain・時点とResult completeness | instrumented Connectorだけを物理PG実Readと報告しない。partial 1/3や0件で「全件・不存在」を推論しない |
| T（テスト） | 同じAttemptの実Target/Operation到達、別保管のIndependent Expected、正常/負例と実際に識別したMutant | Step IDだけ合致、生成AIが自作したExpectedと同源、別Operationの実成功 → TEST-PASS不可 |
| O（性能） | Provider実Readと結び付く測定点、Window/Count/Missing、計測負荷、同一Query/Scopeでの候補前後比較 | 合成19対照・N+1ヒント・p99推定のみでは実測改善不可。性能最適化提案と採用を混同しない |
| H/A（受領・開示） | Consumer別の実Part/End/ACK、現在のTenant/Scope/Grant/Disclosureと送出先 | 内部CLI exit 0や入力`end=true`を実外部受領としない。許可外ID/件数も出さない |

**局所の正の証拠は残す**：D/B/Rが確認できても独立ExpectedのTや性能のO、外部H/Aは自動で合格にならない。逆に、H/Aが未測でも独立に検証されたRの物理Rowset自体を消さない。ProductのFull Coverage/Benchmark/Effectは、必要な独立分母と実測が揃うまでunknown。

#### 35.6.3 呼称の衝突を防ぐ（既存Issue/履歴IDは改番しない）

他チャットで`T1-B`を**TEI Source化→再Parse→Source出典比較**の段階として使う計画と、本テスト基本設計の**生成AIによる未提示Gapの盲検発見**として使う計画がある。両者は別の受入試験であり、同じ`T1-B PASS`を共有してはならない。旧文書の番号を一斉変更せず、新しい横断報告では必ず**領域接頭辞とOwner**を添える。

- **SRC-T1B**（Source取込/再Parse）＝原文→正式Source→Parser/Source出典照合。Source採用・Owner Meaningはさらに別資格。
- **TEST-T1B**（#3273/#2514）＝未知の独立ObligationをAIが発見し、実Target・Pinned Scenario・Independent Expected/Mutantを判別。既知の行番号0 T1-Aは加点しない。
- **READ-P0R / HJ-09**（#2586）＝Function→Query→選択/実呼出Provider→実Rowset/Contextの一Attempt検証。上の二つの前提や代替成功ではない。
- **PERF-T0〜T4**（#3520/#3271）＝Hint/合成Observationから実Profiler/実Read/別Benchmarkへ段階的に昇格。TEST-T1Bの採点と区別。

#### 35.6.4 次の一件の判断と停止線

**最初の一往復**：既存`order_entry.recent_orders`の一Requestに対して D→B→R を実証し、`selected=B, invoked=A`を**同じ結果でも拒否**する。成功時に申告できるのは`read-observed-for-one-attempt`など範囲相対の資格まで。#3500の別Domain Connector試験や#3516のCommand合成Adapterが成功しても、その代替にはしない。実PGが無ければ`instrumented-connector-only`と明記する。

次に#2514の独立ExpectedでT（業務Rowset/NULL/粒度）を検収し、#3271の統計観測を**実Rに結び付けた場合だけ**Oを実測として受け入れる。外部H2/H3実配送、現在Grant、Owner Meaning、Source全閉包、Go C1、全NFRは並列Ownerの別Gate。**第二Runner/Resolver/Provider/Optimizer/認可・テスト正本を作らない**。

**今回の実施区分**：main/既存DraftのSource/CIをGitHubで確認して受入票へ反映した計画更新。新しい`order_entry.recent_orders`実Provider/PG/Function Context/外部Delivery/生成AI TEST-T1B試行/実性能Benchmarkは**NOT RUN**。

<!-- tei-ai-tuning-health-qualification-20261108 -->
#### 35.6.5 Synthetic試験の合格とRepository Healthの不一致を具体的に修復する

main統合[#3520](https://github.com/hat47x/tei-platform/pull/3520)の専用CI [#42476197836](https://github.com/hat47x/tei-platform/actions/runs/42476197836) は19合成対照がsuccess。一方、同HEADの[Repo Health #42476197803](https://github.com/hat47x/tei-platform/actions/runs/42476197803) は**completed/failure**。実ログで、(1)`plan/design/README.md`にAI性能観測文書が未掲載、(2)新WorkflowのCheckoutがcommit SHA固定でない、(3)Job timeout未設定、と切り分けた。Go Formatter/Vet/Workspace Validateの別PASSと、`go test ./...`/CI契約テストFAILを混同しない。また脆弱性確認[#42476197787](https://github.com/hat47x/tei-platform/actions/runs/42476197787)はvuln DB取得時の通信失敗でworkflow failureとなっており、安全性検証のPASSではない。

修正は、新しいFeature/Collector/Proof Storeを作らず**[Draft #3527](https://github.com/hat47x/tei-platform/pull/3527)**の2ファイルに限定：設計索引を1行補充し、既存と同一のcheckout固定SHA・`persist-credentials:false`・`timeout-minutes:10`へ統一。この修正PRのexact HEAD Health・専用19対照・依存確認を別々に検収する。**作成直後のCIはqueuedで、新HEADのPASSは未認定**。Syntheticの入力整合が実Provider Read・実性能改善やOwner受理を保証しないという停止線は維持する。

<!-- tei-test-p0r-case-matrix-observation-ceiling-20261108 -->
### 35.7 P0-Rの最小実行票を確定する：正常な0件と未取得を混同しない

**実装対象は一つだけ**。#2586の既存Read課題 `order_entry.recent_orders` を使い、同一Attemptの `Function Entry → Source-backed Query/SQL → Datasource Binding → selected Provider → actual invoked Provider → Executor/Mapper → Rowset → Function Data Context` を段階別に観測する。#3500の**別Domain**に対するinstrumented Connector Goテストや#3516のCommand系**合成Adapter**呼出成功は、P0-Rにおける実DB Readの代用品としない。

検証者が先に独立Expectedを固定し、実行対象とソースを一つずつ変えた次の**必要な正負対照**を既存#1180/#2514へ渡す。新しいCase/Oracle正本は作らない。

| 受入対照（#2586） | 実入力／観測の必要条件 | 成功時の主張と拒否すべき誤判定 |
| --- | --- | --- |
| **P0R-01 対象1件** | 既存Functionに対する同一Requestから、実SQL原Byte・Bind args・selected/actual Provider・実Row/Column・MapperのResultと粒度を記録。独立Expectedは要求Scope内の1行を固定 | 当該Query/Provider/Mapperの局所観測。実物理DBへ未接続なら `executor-mapping-bounded` まで。1件成功を全Dataset完全性としない |
| **P0R-02 正常な0件** | 認可済み・実Read完了・指定Filter/Scopeに実際に0行。独立Expectedも0行 | `empty-complete-for-query-scope` は成立し得るが、Source/Domainの不存在、隠れた行や他Tenantの0件を証明しない |
| **P0R-03 部分取得・途中障害** | 第1ページ取得後に第2ページ失敗、Tool/Mapper/Providerの通信途絶、またはCaller中断 | 最初の確認済み行の局所正証人は維持しても、全Rowset完了・0件・Business成功は禁止。失敗部位と受信終端を別記録 |
| **P0R-04 Datasource未解決** | SQL/Function宣言あり、必要Aliasが未Binding（既存診断の `UNBOUND_DATASOURCE` 等を照合） | 正式Provider実呼出しは起きず、未解決と診断。空Rowsetの成功に変換しない。既存実コードの安定Failure Code以外を新規創作しない |
| **P0R-05 選択と呼出しの相違** | declared/selected Bに対して実invoked Aが**同じRowset**を返す。正例はselected B→invoked B | 出力の値・SQL自体が同じでもProvider Binding不一致を拒否する。Target/Build/Plugin exact identity・Current Grantは別資格 |
| **P0R-06 同名Queryの異物混同** | examples/workspace/別DBの`recent_orders`が名前のみ一致、実SQL/引数/Database/Snapshot/Row scopeが異なる | Query name同値だけでTest/Performanceの母集団へ合算しない。物理・論理Rowのidentity/grain、Data Contextの依存を区別 |
| **P0R-07 Contextの鮮度差** | Query Source/Source Indexまたはimport Provider Raw/Manifestだけを読取前後に変更。World値が同じになる負例も用意 | 当該必要Source occurrenceに依存する資格だけstale/unknown。Source観測の二回取得は原子的Snapshotの証明ではない |

**数値・表示上の判定**：`0`（実際の結果件数）、`null`（業務値）、`missing`（必要Column欠損）、`partial`（結果一部のみ）、`unknown`（取得／観測資格なし）、`error`（実技術障害）を異なる状態で保持。とくにProviderから0件が返っても、Requestの当該Scopeと終端受領を確認するまでは`0 complete`へ昇格させない。

**既存工程への到達レベル**：`declaration-observed`（#3482）→`binding-observed`→`executor-mapping-bounded`（#3500の別Domainで局所先行）→`physical-read-observed`（隔離PGでSKIPなしの当該Query実行）→`function-context-observed`（Mapper/粒度/NULL/Scope一致）→`consumer-delivery-observed`（明示Audienceと受領証拠）。これらは説明用の**Claim Label案**であり、新しい製品Schema/Enumへ自動採用しない。同じAttempt内で接続できない段階は、別試行として残す。

### 35.8 Test・性能・Plugin・Consumerの間で共有するのはAttempt参照だけ

P0R-01〜07で得られたRead証拠を、#3273 **TEST-T1B**、#3271 **PERF-T0〜T4**、#1251 **Inspector D1**、#3262 **Design Context**の入力へ渡す際、**新しいEvidence StoreやTest Generatorの所有者を増やさない**。特定の試行に紐づくSource/Invocation/Rowset・Provider Selected/Invoked・Request Scope・Contract/Grantと、受け取ったConsumerを参照する。

- **TEST-T1B**：独立Reviewerが先に固定した未知ObligationとHoldout/Expectedを生成Agentへ開示しない。T1-A既知行番号0、Context42 test-only対照、Provider instrumented Connector成功は未知Gap発見と数えない。実Pinned Scenarioで新Ruleに到達し独立Mutantを判別したときのみ限定成功。
- **PERF**：実Readと紐づく計測点・期間・件数・条件と改善前Baselineがなければ、合成19対照やN+1候補が成功しても「性能改善」を宣言しない。同一Query/Scopeの変更前後で独立比較できるまで提案段階。
- **Inspector／Plugin**：main統合済み#3371/#3378/#3381の型付きService seamと局所同値Go検査は保持するが、通常CLI／Wails Studioが**選択Bを実際に呼んだ**というProduction S2に昇格しない。#951/#1016の実選択と#1251の実Call/Grantを必要な一辺として扱う。
- **CLI/MCP**：ADR-0048の共通Application Operationは採用済み。参照MCP 4 Toolと`validate`局所同値は実装済みだが、Read用の全Tool・外部Harnessでの実受領、権限/送出、完全JSON／Pagingは独立。Tool正常応答をProvider実ReadやBusiness成功へ使い回さない。
- **CI/Health**：main #3520専用合成19件の実成功と、元HEADの全Health不合格・脆弱性DB通信失敗を分離。[main統合 #3527](https://github.com/hat47x/tei-platform/pull/3527)はIndex/Workflowの衛生修正であり、既存失敗Runを遡及的PASSにせず、新HEAD/main pushの独立Runで検収する。

**実行優先順位は維持**：今すぐ別のテスト設計文書を作るのではなく、#2586へP0R-01〜05の一件ずつの**実Read境界**を返し、#3273へBlind TEST-T1B一件のHoldoutを別ルートで提示し、既存#1180/#2514で判別する。P0R-06〜07、PERF、外部H3、SHO等の追加Consumerは並列Pressureとし、最初の実Readを待たせない。**他製品の責務や詳細を、このTEI文書へ新規移植しない。**

**今回のEvidence資格**：既存GitHub実PR・main・CI、Source/計画の照合に基づく実装受入票。ここでの新P0R-01〜07、TEST-T1B新Trial、対象Queryの実PG/Consumer/物理Grant・Go C1・実性能改善・現在Owner Meaningは**NOT RUN／未受理**。新Core、第二Reader、Resolver、Host、Oracle、Runner、Provider Registryは追加しない。

<!-- tei-test-nov09-actual-knowledge-trace-tool-intake -->
### 35.9 2026-11-09 実CLIドッグフーディングをTEST-T1Bの「自律的な情報取得」入場試験へ接続

**新しいReader/Parser/Test Runnerを追加せず、既存のCLI/MCPと共通Application APIを受入対象とする。** 前節のP0-R `order_entry.recent_orders`（実Provider Read）はそのまま最優先の一件、TEST-T1B（生成AIが非公開の未知Gapを発見）は独立する。今回は既存実行の結果から、**生成AIが資料・Goソースや人手補完なしで必要Contextを得られるか**という追加の入場条件を採録した。実AI Trial自体は未実施。

| 実装・実ドッグフード（Owner） | 現在確認できた到達点 | 次の受入条件とClaim ceiling |
| --- | --- | --- |
| [Round 14 #3685](https://github.com/hat47x/tei-platform/pull/3685) / #3149 | mainの実CLIで `knowledge-context → review-plan → review-handoff → context -disclose source → qualify → review-disclose` を一巡。古いfingerprint、別Subject・不存在handoffを拒否 | `status:complete`は当該review-disclosureの完了であり、全Required/Meaning Acceptance/TEST-T1Bの準備完了ではない。実AIが独立Expectedを秘匿して設計した試行はNOT RUN |
| **発見不能な入力契約** [#3186](https://github.com/hat47x/tei-platform/issues/3186) / [#3419](https://github.com/hat47x/tei-platform/issues/3419) | `-qualification-requirements`はCLI discovery/helpから必須JSON形を辿れず、手動の正解は `{"expectedKind":"semantic-data"}`。未知欄をrejectするのは適切 | **Toolの説明だけで**既存v0のRequest型・必須項目・最小例/参照先を発見できることを、短い指示→Tool観測→合法Input→実Request成功で試す。既存ドキュメントを第二Schemaにしない |
| [Round 15/#3693](https://github.com/hat47x/tei-platform/issues/3693) / #3419 | mainが `trace -mode full` の0件を `rows:[]`として返す問題を解消。summaryでrowsが無いのは意図された契約 | 空配列／省略／null／unknownを用途とmodeに従って判別する。既存JSON golden byte順/旧v0 Reader互換を保ち、0件を「全業務Relationが存在しない」へ拡大しない |
| [Round 11/16 #3625](https://github.com/hat47x/tei-platform/issues/3625) / [merged #3633](https://github.com/hat47x/tei-platform/pull/3633) / #3420 | `trace.nextOffset`に並ぶ`trace.continuation.fingerprint`を次のページ取得に使用。内側 `graphFingerprint` は別目的で、旧 `trace.snapshotFingerprint` aliasは互換維持 | **正しい続きの値をTool結果だけから選べること**を検証。Scope/Direction/Source Raw変更は`STALE_SNAPSHOT`、summaryにoffsetを渡すと`INVALID_ARGUMENT`。別ページのsource/selector/versionが違えば集合を連結しない。Mode/offset/limit変更の現行トークン意味はRound16の限定実測に従う |
| [Round 13 #3149](https://github.com/hat47x/tei-platform/issues/3149) / [#2619](https://github.com/hat47x/tei-platform/issues/2619) / [#2621](https://github.com/hat47x/tei-platform/issues/2621) | 実 `.tei`宣言の定義位置はinspectで追えるが、管理MarkdownはSource Indexにあるだけで本文の語をinspect/traceできず、Repository全体にRoot Manifestも無い | 設計書本文を自動的にTEI Loaded/Parsed/Acceptedへ昇格しない。文書の影響範囲が不明なら`document-symbol-unresolved`相当の診断で止める（診断名は案）。必要なSource/Role取得は正式Authoring/Parserの受入へ委譲 |
| [merged #3540](https://github.com/hat47x/tei-platform/pull/3540) / #1251 | 通常CLI・Studio Hostが静的参照Inspectorを既存Compositionで選ぶ**S2-reference**がmainへ統合。PR exact HEADのHealth/Studio等5系統CI success | S2-reference≠外部PluginのS2-product。実Artifact Load/Link、別Provider B差替え、Current Grant、Conformance、Verified BOMは独立。Inspector成功を`recent_orders`実Readへ合算しない |

**既存TEST-T1Bへの最小追加試験（番号／Scenarioは増設しない）**：独立Reviewerが許可済み一RuleのRequiredと非公開Holdoutを固定し、生成Agentには「対象Functionの未カバー検証を1件提案して」とだけ伝える。Skill/CLI Consumerが(1) `commands`や型付き説明から要件ファイルの形を見つける、(2) knowledgeのContext→Review/Handoff→Qualificationのうち必要な処理を実行する、(3) traceを正しく完走するか、`partial/stale`と残りを報告する、(4) Markdown/Provider未取得を「存在しない」と言わない、(5) 適格なRuleだけをCandidateにして独立Expectedへ渡す、という**一往復**を観測する。必要契約が未公開ならそこで`blocked:input-contract-undiscoverable`、Source依存未取得なら`insufficient-context`とし、既知のT1-Aを再生成してPASSにしない。

**受入の所有と順序**：入力Schemaの発見可能性＝#3419、CLI/MCPの不正Input/Result/継続パリティ＝#3420、Source/独立Required＝#3262、Authoring本文意味化＝#2619/#2621、未知テスト生成＝#3273、Expected/Scenario/Evidence＝#2514/#1180/#1163。まず既存mainのread-only CLIで**一つの合法な自己完結Request**を作れるか実証し、その後MCPで同一Operationの意味結果を対照。#2586の実Readと並行可、外部H2/H3・正式Product S2/BOMまで一律に待機しない。

**停止線**：`knowledge-review-disclose:complete`≠Source/Meaning全閉包、`trace rows:[]`≠業務上不存在、Fingerprint一致≠Current Grant、Tool Replyを返したこと≠外部Agentが完全受領、S2-reference≠実Plugin/Read、設計試験の増設≠Test能力進捗。新しい本章の一往復・外部MCP/Harness/T1-B/実ProviderやC1をこの更新では実行していない。

<!-- tei-test-20261109-contract-close-vs-executable-gap -->
### 35.10 2026-11-09 夕方：実CLIの診断統一・文書ドリフト検出とBuild/Effect証拠を、次のTest Capability試行へ接続

**本節の狙い**：設計情報を集めるためのTool契約が曖昧なまま、生成AIに「不足を見つけてテストを作る」と期待することを避ける。同時に、既に実装されたCLI/MCP・参照Inspector・PostgreSQL・self-buildの**局所証拠**を使い、未知のTEST-T1B一件と実Function P0-R一件に必要な最初の欠損辺だけを解消する。別のTest Generator・検証正本・API Registry・実行Host・Plugin Resolverを作らない。

#### 35.10.1 追加の限定実装成果と、そのClaim Ceiling

| 既存Owner／最新の確定実績 | できたこと（現行main/PRに限定） | できたと宣言しないこと |
| --- | --- | --- |
| CLI引数分類 [merged #3749](https://github.com/hat47x/tei-platform/pull/3749)、[merged #3763](https://github.com/hat47x/tei-platform/pull/3763) | `inspect`・`knowledge-*`の必須引数**未指定**を`INVALID_ARGUMENT`へ統一。空白/不正Subjectの`INVALID_SUBJECT`、対象不在`SUBJECT_NOT_FOUND`、不正要件ファイル`KNOWLEDGE_BUILD_FAILED`は区別した。MCP READMEに残る旧コードの文章を修正し、戻し/説明削除の負例を拒否する実Goテストを追加 | これだけでMCP JSON-RPCの`-32602`とCLI Exitの完全共通化、全メソッド入力の厳密検査、実外部Skill受領まで合格にしない |
| Model/catalog表示 [merged #3754](https://github.com/hat47x/tei-platform/pull/3754) | `model catalog`の列挙成功と、内部`validation`判定を利用者向けに区別して説明。Contract拡張は必要に応じて#3419/#3420で評価 | catalog `ok:true`や `std` Builtin列挙を、業務Source Load済み・Meaning受理・Required充足としない |
| CLI/MCP read-onlyドッグフード [Round 16–18](https://github.com/hat47x/tei-platform/blob/main/plan/workstreams/TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11.md) | Trace継続のSource/Scope/Direction失効、full mode空`rows:[]`、CLI内Error分類、ドキュメントとコードの符号ずれを実測/補正。対応PRの局所Go実行記録あり | 実AgentがTool自己記述だけから `knowledge-qualify` 入力を一度で生成できる、管理Markdown内参照まで探索できる、外部MCP全受領を証明したわけではない |
| Inspector [merged #3540](https://github.com/hat47x/tei-platform/pull/3540)、[merged #3740](https://github.com/hat47x/tei-platform/pull/3740) | 通常CLI/Studioの静的参照S2-referenceと、ID完全一致/同Content別WorkspaceのOrigin問題を独立に計画化 | `Canonical.id`一致・`World`一致・`ArtifactDigest`一致で、実Source Kind、Plugin B実呼出、現在Principal Grant・S2-product適格性を自動認定しない |
| self-build [merged #3731](https://github.com/hat47x/tei-platform/pull/3731)、[merged #3753](https://github.com/hat47x/tei-platform/pull/3753)、[merged #3760](https://github.com/hat47x/tei-platform/pull/3760) | 同じPlanの2回Build結果を比較し`reproduced/mismatched/unperformed`を分離。呼出し元が明示したCatalogによるin-process Intent→Plan導出と、欠損/不一致/二重指定の拒否を限定検証 | 原入力Sourceの閉包・Catalog/Seed Trust・既定経路切替・G4の成立、製品配備/外部Hostの同等性ではない。再Build2回は計測費用も追加する |
| PostgreSQL／HTTP E3 [merged #3736](https://github.com/hat47x/tei-platform/pull/3736)、既存[PG実証の4系統](https://github.com/hat47x/tei-platform/issues/3274) | Row lock、CAS、Commit後ACK喪失、冪等再実行、Tenant scopeなどの**個別実DB試験**を計画の部分Evidenceに採用。Hook・Command Outcome分類も個別実装あり | 同一OperationのHTTP→Authorization→Executor→DB Commit→Receipt→Client受領が一続きで成功したこと、Business Completion、Hook全実Pathを証明しない |
| 異種宣言の実利用調査 [merged #3757](https://github.com/hat47x/tei-platform/pull/3757) | 宣言がConsumerで参照されることと実Dispatchされることに断絶があると確認 | 参照されたBinding/Instancesを実行へ繋ぐべきとのOwner選択、実Callback/Effect資格はまだ採用しない |

#### 35.10.2 AI/Skill/人間が同じ失敗を理解するための最小「Reason契約」

**新たなError Enumの採用ではなく既存CLI v0／MCP運用のExpected分類票**とする。まず次の**入力→結果→次の一手**を一つのread-only Operationと対象Workspaceで独立に対照する。

| 入力・必要Scope | 合法な結果の分類 | 次に選ぶ行動／誤判定防止 |
| --- | --- | --- |
| 必須`-subject-kind`/`-subject-id`等を**未指定** | CLI `INVALID_ARGUMENT`・exit 2。MCPは不正引数のProtocol Error `-32602`（同一wire表現ではない） | Tool説明・必須Parameterの発見に戻す。Source/Subjectの意味的欠損やRule不在と呼ばない |
| 指定Subjectが**空白等の不正値** | `INVALID_SUBJECT`など既存Operationの値検証Failure | 値を訂正。勝手にdefault Subjectへ代替しない |
| 合法IDだが**対象不存在** | `SUBJECT_NOT_FOUND`、認可されたScope内のnot-found | Selector/Source Role/Loader/Scopeを再照会。全Domainで「存在しない」と断言しない |
| `knowledge-qualify` の**JSON Shape/Requiredが不正** | `KNOWLEDGE_BUILD_FAILED`等、必要な独立入力の不適格 | #3419型付き契約・説明Refへ戻す。適当なEmpty Expectedを自動合成しない |
| `trace` full の**正常0件** | `rows:[]` + 明示Scope／Mode／指紋・終端の条件 | 「この指定Traceの0行」。未Loaded Relationや非公開Ruleの不存在としない |
| `trace` の**古い/異Scope/誤Fingerprint継続** | `STALE_SNAPSHOT`等の現在版不一致 | 正しい`continuation.fingerprint`で当該Scopeを再読。異版ページのRowを継ぎ足さない |
| Catalog/Inspectの**成功応答だが実Source不足** | Operation/Envelope成功、独立Required/Source/Meaningは`partial/unknown` | 既知の合法Roleだけにprovisional Test Candidate。不足したSourceの追跡を薦める |
| 実Provider/Effectが**未観測** | Inspector S2-reference、Capability宣言、Plan/Build Artifactの各局所正例は保持 | `order_entry.recent_orders`の同一Attempt Read、選択と実Invoke、独立DB行/Receiptへ進み、Step/Coverage成功を転用しない |

**引数欠落・Subject意味不在・Source未読・Tool送信失敗・Partial Deliveryは同一Unknownに丸めない**。一方で、認可されないObjectの存在/件数や、隠れたRequired項目をエラー診断から漏らさない。Reason分類の「実行可能な次の操作」は、現在のTool Discovery／Effect Profile／Audience/Grantを参照して制約する。

#### 35.10.3 「Test Abilityが成立した」と初めて言える最短の2試行

| Gate | 入力、実行、独立Expected | PASSの証拠 | FAIL/UNKNOWNへの停止線 |
| --- | --- | --- | --- |
| **T-Discover（TEST-T1B、#3273）** | 独立Reviewerが未提示のRequired Rule Gap/合法Baseline/Mutant・Expectedを固定。生成Agentには許可Subjectと短い依頼のみ。既存read-only Skill→共通Operation CLIからSchema/必要Source/traceを取得→本当に未知の一差分Caseを提案→#1180 Review/実Pinned Step→#2514独立検証 | 生成時の許可Context・非提示Holdout・Subject/Source/Attempt・新Case ID・対象Rule実Outcome・Mutant検出・Tool Complete Receipt。トークン/Tool CallとFalse-qualifiedも測定 | 入力形が発見不能なら`blocked`、Source/Authority/Query不足なら`provisional/unknown`。旧行番号0のT1-Aや9件Receipt Copyの再実行は新規発見に数えない |
| **R-Read（P0-R、#2586）** | `order_entry.recent_orders`で実Function Entry→同一Query Source/args→明示Datasource Alias→selected/actual Provider→実Read→Mapping/Context→独立Rowset Expected。正常1行／0行／部分＋error／未Binding／B選択A実Invokeは§35.7の既存7Caseを流用 | 同一AttemptのQuery原byte/Provider Identity/Scope/Rowset completeness/Parser/Mapper/Observer/Client受領。隔離PostgreSQLを使うならSKIPなし実Readと対象DBの証明 | 別Function `org_view.case_browse`、Fake Connector、静的Inspector、Command系実Call、PostgreSQL個別E3テストだけなら`real recent_orders Read`は**未確認** |

**製品全体のCoverage/C1/NFRをPASSの分母にしない**。T-Discoverの新規性とR-Readの実Data Contextは別指標だが、必要な一Attemptの実Source/Plugin/Tool Delivery/Expectedを既存Evidence参照で比較できるようにする。生成AIは判定/採用の代行権限を持たず、Ownerの業務意味受理と実Hostの現在認可は別ゲート。

#### 35.10.4 計画の持ち主と実行順は変更しない

**第1優先**：#3419に型付きRequestの自己記述、#3420にCLI/MCP不正値/Failureの非破壊対応表を集約し、既存実バイナリとMCPだけで合法Knowledge Requestを試す。PR #3763のREADME主張を実テスト化する方法を同じ契約書・参照説明へ適用する。Markdownの意味検索は#2619/#2621で継続、T-Discoverが全Markdown解析を要求しないようScopeを限定。

**第2優先（並列）**：T-Discover TEST-T1B一件とR-Read `recent_orders`一件を進める。まずSample/Required Universe/独立Expectedを固定し、結果0件・Error・Provider不一致を同じ「取得なし」に丸めず、実Tool/Consumer受領へ結ぶ。

**第3優先**：#2831 self-buildの再現性/予定Artifact/実消費Source閉包の差、#3274 E3のCommit→ACK損失とClient Receiptの断絶、#3278/#3267のTrait/BOM、#2582のNFR/性能へ**各Ownerの独立Pressure Case**として適用する。全製品横断E2Eが成立していないため、新Gateや中央基盤に吸収しない。

**今回の実施状態**：main/PRの既存実Go・CLI・DB試験結果を照合して本設計と既存Ownerの受入条件へ反映。ここで述べたT-Discover/R-Readの新実験、Knowledge JSON自己発見修正、MCP全面同値、独立本番Plugin、Host/Client E3連続実証、外部H2/H3、C1・Owner意味受理は**今回NOT RUN／NOT QUALIFIED**。

<!-- tei-test-20261109-declaration-execution-metatests -->
### 35.11 2026-11-09 11:31Z：宣言→実行の断絶、Hook境界、DSL順序変異をTest Patternの一組へ集約

前節で定義した **T-Discover／R-ReadのP0実行順は変更しない**。その入力・結果に対し、他チャットで**mainへ統合済みの三つの独立観測**から、横断的なTest Patternの適用条件を具体化する。新たなTest Runner/Executor/Oracle/Hook Engine/Environment Resolverを作らない。

#### 宣言が読めることと、実行が届くことの間を計測する

[main #3757](https://github.com/hat47x/tei-platform/pull/3757)のread-only Caller調査では、`internal/environment.Environment` に実装参照を示す**6種類の宣言**がある。これは6種類全部を実行に繋ぐべきという決定ではない。

| 宣言 | 製品コード・参照で確認した接続位置 | 未確認または存在しない辺／Claim上限 |
| --- | --- | --- |
| `DeploymentExecutors` | `routing.RouteCommand`で宣言からExecutor IDを解決する参照方式 | `Router.Dispatch`は参照実装のみ。現製品の実配備Operationまで実Call・Effectが届いたとは言えない |
| `ObservationProviders` | `observation.Binding`が実`cmd/tei/main.go`で宣言を解決する | `observation.Router`は非テストで未構築。実`Observe`／受領済みObservationは不明 |
| `EvidenceProviders` | `evidence.Binding`が実`cmd/tei/main.go`で宣言を解決する | `evidence.Router`は非テストで未構築。実`Collect`／外部Evidence受領は不明 |
| `ResourceBindings` | `resource.MatchBindings`は実装あり | Resourceのマッチ／貸与まで。業務Command/Observer実行の正証人ではない |
| `Instances` | 環境定義の参照・検証 | この調査では製品実解決／Invokeの対象外 |
| `Bindings` | 環境定義の参照・検証 | この調査では製品実解決／Invokeの対象外 |

この調査の「非試験Callerの有無」は**Sourceコード上の有界正負証拠**であり、Runtime経路の不存在・認可・外部配備状況を全世界に証明しない。特に `observation.Binding` の成功を `Router.Observe` 成功と見なさない。[判断Draft #3771](https://github.com/hat47x/tei-platform/pull/3771)は、この実態を**既存Composition計画へ持ち込む提案**であり、実行方式のOwner決定ではない。

#### TEST-T1Bが選べる「実行可能な負例」の選定規則

| Pattern群 | 先行の実装／実験 | 独立ExpectedとTest Candidateの判定 |
| --- | --- | --- |
| **Declaration→Resolver→Invocation→Receipt** | #3757の6宣言、既存#3516/#3692 C-3参照、および#1251/#2586 | Name/ID/Capabilityが読めてBindingも返っても、実Router/Provider/Consumerに届かなければ`invocation-unobserved`相当の局所Gap。**宣言の6種類を自動的に全て実行させるPatternを選ばない**。Ownerが当該用途で実呼出しをRequiredとして指定した時だけ未カバー義務にする |
| **Hook outcome vocabulary** | [main #3761](https://github.com/hat47x/tei-platform/pull/3761) の`ValidateOutcome`は語彙外の名前を`HOOK_OUTCOME_UNDECLARED`で拒否、名前無し不成立は正当に許容。6件の追加試験が局所PASS | `valueRejected`等の語彙外、語彙内、名前なし不成立を三分割。内部Hook boundaryの検査PASSを**実外部Plugin→CLI/JSON→業務Command結果**へ転用しない。Rule Accepted／Command Refused／技術Failureは別Outcome |
| **DSLの順序変異** | [main #3750](https://github.com/hat47x/tei-platform/pull/3750) で`states open closing closed`の**後ろ**に`closes on day 20`がある場合、後続語を状態名に取り込んでいたParserの誤りを修正 | 文法上許された二記述順序について、独立に固定した`states={open,closing,closed}`、`closingDay=20`、拒否Ruleの同じ意味への投影を要求。**どんな任意順序も同値**とする一般命題は置かない。Source raw/Spanの改版は同じ意味でもSource Indexと引用資格が変わる |
| **CLIの重複引数とCardinality** | [PR #3772](https://github.com/hat47x/tei-platform/pull/3772) HEAD `2643027f37f30092f0a6f988ce8161f8f0ae1e54`では一値flag重複を`INVALID_ARGUMENT`で拒否、`view -input`等の繰返しflagを`commands.repeatableFlags`で明示する変更が**提案中**。個別Go/実CLI試験はPRに記載、PRのHealth等CIは照会時 **in_progress** | 重複が黙って最後勝ちになる既存mainの挙動を「仕様」として採用せず、変更PRが統合されるまでは`pending`。生成AgentはflagのCardinality/argvをDiscoveryから確認し、単一値の重複でSource/Scopeを無言に変更しない。MCPのJSON objectとCLI argvの構造差を考慮し、両Transportの生byte同一性を求めない |
| **設定Dataの未宣言/重複/既定** | [main #3722](https://github.com/hat47x/tei-platform/pull/3722) の宣言済みKey read／[main #3750](https://github.com/hat47x/tei-platform/pull/3750) の`master.setting`例 | 宣言外Keyを空文字の正常値とせず、重複Key/同一Userを拒否し、User値→共有Defaultの選択を独立Expectedで検証。書込み監査/権限/実DBを未接続のまま正式Setting保存成功としない |

#### 最小検収手順と所有関係

1. **P0は変えず、宣言→実行を一つの有界Context行として付与**：#3262が返す`Function/Rule/Provider`の各Requiredについて、`declared/loaded/resolved/invoked/observed/received`のうち、**当該用途に必要**で実証された段階だけを表示。全段階を埋めるためにProduct Capabilityを勝手に発明しない。
2. **T-Discoverの未知Gapの分類へ適用**：単に宣言されただけの`ObservationProviders`を「テスト済みObserve」としない。Callerの静的調査を「業務機能は存在しない」に昇格させない。必要であれば#2514の独立Expected・#1180の既存Scenarioに一つの負例を渡す。
3. **独立した境界変異を追加する場合**：Hookの語彙・DSLの許容順序・CLI flag cardinality・Setting Keyは、それぞれ既存の担当が所有する契約に対する一差分の正負試験とする。Test Generator自身のExpectedやActualからExpectedを再導出しない。
4. **二試行の優先結果を指標化**：T-Discoverは「入力契約自己発見できたか／未知Gapが真にRequiredか／独立Mutant判別」、R-Readは「選択と実Invokeの同一性／実Query RowsetとScopeの完全性」。補助PatternのCase数で進捗を水増ししない。

**現時点の実証資格**：#3757はread-only Caller inventory、#3761は内部Hook境界試験、#3750/#3722は構文・設定の局所試験でmain統合済み。#3772は**未統合・CI進行中**。本節の横断Test Generator・新Scenario実Run／正式外部Plugin、実Provider DB、Studio/H3、C1/NFR、Owner Meaning Acceptanceは**NOT RUN / NOT QUALIFIED**。第二Owner/Runner/Registry/Authorityを追加しない。
