# TEIテスト基本設計：テスト専用製品の適用評価と実証計画（2026-11-07）

- 状態：**評価待ちの設計追補／採用前**。テスト専用製品をインストール・実行してTEI適合を確認した記録ではない。
- 主担当：[独立検証基準 #2514](https://github.com/hat47x/tei-platform/issues/2514)、[Driver Scenario SSOT #1180](https://github.com/hat47x/tei-platform/issues/1180)
- 関連：[単一改訂Evidence #1163](https://github.com/hat47x/tei-platform/issues/1163)、[Data Context取得範囲 #2506](https://github.com/hat47x/tei-platform/issues/2506)、[実Provider縦切り #2586](https://github.com/hat47x/tei-platform/issues/2586)、[外部UI制御 #2391](https://github.com/hat47x/tei-platform/issues/2391)、[状態・Snapshot #1131](https://github.com/hat47x/tei-platform/issues/1131)、[Harness境界 #2515](https://github.com/hat47x/tei-platform/issues/2515)、[開発工程 #1445](https://github.com/hat47x/tei-platform/issues/1445)
- 既存の設計：[経路間の業務機能完全性とScenario由来E2E](CROSS_ROUTE_FUNCTIONAL_PARITY_AND_TEST_DERIVATION.md)。こちらの文書は置換しない。
- 限界：本書の製品機能は2026-11-07確認時点の公式文書に基づく比較。性能、費用、TEIとの相互運用性、製品別ライセンス適合は未実測。OwnerのMeaning受理・実Host Grant・Production資格は含まれない。

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
