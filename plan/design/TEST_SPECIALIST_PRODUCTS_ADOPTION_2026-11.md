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


---

<!-- tei-test-cross-chat-witness-20261107 -->
## 8. 2026-11-07 他チャット横断：実証の進展を反映したテストの受入単位（追補／設計）

本節は、前節P0〜P2を置き換えず、最近のmain実装・他担当Draftに照らして「**何が検証済みか／どう検証が欠けているか**」を明確化する。参照先の設計ケースやEvidenceを複製しない。

### 8.1 現在の基線と、過去の未実装記述を混同しない

| 参照 | 確認できる限定事実 | 本書に残す未証明 |
| --- | --- | --- |
| main [#3155](https://github.com/hat47x/tei-platform/pull/3155) | Commit receipt再取得の \`receipt-observed\` と、別接続Resource現在値/Payload一致による \`resource-observed\` を分離。後者のみ \`persisted:true\`。Go試験・対象CIで実証。 | 更新対象全件、Versionの因果・時点・現在Read Grant、Owner業務完了。**旧「保存観測未実装」の記載を現在形で繰り返さない**。 |
| main [#3145](https://github.com/hat47x/tei-platform/pull/3145)、[#3167](https://github.com/hat47x/tei-platform/pull/3167) | \`function-definition\` pattern増加でも \`tei operations\` が8件 derived/0件 declaredのまま。Data IDだけ部分照合、Capability/Relationは一致せず。 | 宣言Pattern→Typed Operation→Action Offerの意味的結合、実Provider/Host。**未接続を「テスト対象0」と扱わない**。 |
| main [#3078](https://github.com/hat47x/tei-platform/pull/3078)／[#2621](https://github.com/hat47x/tei-platform/issues/2621) | Workspace file inventory、\`readByLoader\`等により発見/Loader消費を区別する経路が前進。 \`*.tei.md\`へ統一する方向。 | 実Parser Role、Accepted Meaning、Builder実入力・Consumer実使用、旧形式と新形式の等価性。 |
| [#3175](https://github.com/hat47x/tei-platform/pull/3175) | \`EvaluateCompletion\` に対する同件数異集合／TOP-1／述語列未取得の実Go API characterization。 | 合成rowsetでありFunction→Query→実Provider取得の閉包は未成立。 |
| [#3192](https://github.com/hat47x/tei-platform/pull/3192)、[#3222](https://github.com/hat47x/tei-platform/pull/3222)、[#3225](https://github.com/hat47x/tei-platform/pull/3225) | Shape／Representation Type／Function内Instance/Slot／Mappingと媒体軸／複数表示の協調を別契約に分けた設計（Draft）。 | Renderer・Provider・媒体同期・実更新のE2Eは未実証。RP/MVケースは各設計Ownerが保持する。 |
| [#2515](https://github.com/hat47x/tei-platform/issues/2515)／[TAI連携](https://github.com/hat47x/tai-pleroma) | 外部HarnessのTool結果切断・出力混在・Call/Result不一致、機密Evidenceの二次送出を、比較資格・開示の独立問題として特定。 | 実OS/全出口の隔離、全Step完全受領、H2/H3の独立比較資格、未計測Sinkの不存在。 |

古いDraftの「未実装」「NOT RUN」表現を現在mainへ無検査に継承しない。逆に、複数のPR/CIから得た局所PASSを別Attemptの証拠として合成しない。

### 8.2 第一級にするのは「検証要求の到達状態」、新Test Runnerではない

検証要求ごとに次の状態を**別観測列**として保存・表示する（以下は候補の表示概念であり、新しい永続Schema/第二Registryではない）。

1. **Required**：独立要求Source/Ownerのどの規則・対象についてテストが必要か。Required集合の根拠はCandidate Modelから逆算しない。
2. **Discovered/Loaded/Parsed**：Manifestが宣言したSource、列挙された実Source、Loaderが読んだByte、Parserが解釈したOccurrenceを分離。未Loader消費・未知Root・未列挙の場合、業務Missingとは断定せず \`source-incomplete / unqualified\`。
3. **Meaning/Binding**：Accepted MeaningとFunction Context、Relation Path、Pattern/Operation、Targetに接続できたか。表示できたこと／同Data IDであることから、Operation実行可能としない。
4. **Generated/Executed**：Driver Scenarioの必須Stepが各Targetの生成物へ落ち、実Runで各Stepに到達したか。単なるCase件数、exit 0、CI成功と分ける。
5. **Observed**：Business Result、Execution Error、UI状態、Provider rowset、Commit receipt、別接続Resource read、Tool受領の**各種類**と、対象同一性・時点・完全性を記録。未観測は不在でない。
6. **Qualified**：Independent Expectedの独立性、Known Mutant判別力、実Source/Build/Consumer一致、母集団/権限/Disclosure/Owner受理のうちClaimに必要な条件を確認。Scope外はunknownまたはunqualified。

これは一直線のPASS昇格ではなく、**各関係を根拠付きでjoinする評価票**。高位列が真でも低位列の欠測が残れば全体の主張を拡張しない。旧\`PASS\`・\`resource-observed\`・\`Verified\`を一つのBooleanへ還元しない。

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
- TAI/DSH等のHarnessでToolが成功したように見えても、stdoutのJSON 8,192 byte切断、stderr混入、Call/Result件数不一致、Tool reply未受領なら\`complete\`にしない。**Tool実行と、検証器が実際に受領した全内容**を別Witnessにする。Runの途中で未受領のStepを暗黙に補完しない。
- Test・Trace・CI・Issueが機密値を含む可能性を考え、保護された生Evidenceと公開用の集約結果を分ける。非公開Expected/行数/拒否理由/秘密由来Digestを無条件に公開しない。全Sinkの監視範囲は#2515を参照する。

参考一次資料： [Jepsen core](https://jepsen-io.github.io/jepsen/jepsen.core.html)、[Hypothesis Stateful](https://hypothesis.readthedocs.io/en/latest/stateful.html)、[OpenTelemetry Overview](https://opentelemetry.io/docs/specs/otel/overview/)。

### 8.5 OKF Attested Computationとの接続（DocumentとRuntimeの責務分離）

[Open Knowledge Format v0.2仕様 §10](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)の\`type: Attested Computation\`は、計算内容・runtime/parameter・executor/receipt・決定論的attesterをConceptで参照し、実行Receipt自体はbundle外のRuntime Artifactとして扱う。**説明可能な計算契約と、実行の確証を分離する点**を採用候補とする。

TEIでは \`Source+Accepted Function+Independent Expected\` → \`既存Driver/外部Hostでの実Run\` → \`Evidence/Receipt\` → \`決定論的検査\` → \`Claim相対Qualification\` の関係をread-onlyに投影する。SHOは文書内リンク・引用を扱い、SEIは知識Claim/Evidenceの参照を担う。**OKFのfrontmatterに書いた\`verified\`やattester PASSを、Ownerの業務意味受理・現在Host Grant・実DBの完全性へ自動昇格しない**。OKFの軽量互換要件を破る必須TEI独自fieldや独立実行Registryを設けない。

## 9. 追加の独立検証基準（TT-11〜TT-20：別IssueのCaseを参照する受入票）

| Case | 入力・意図的欠損 | 独立して期待する結果／拒否する過大主張 | Owner・証拠 |
| --- | --- | --- | --- |
| **TT-11 Source closure** | Required\`*.tei.md\`をmanifestに宣言するがLoader/Parserが消費しない。別例は旧\`*.tei\`と新\`*.tei.md\`の同Subject異内容。 | validateが局所成功してもRequired完了ではない。未読込は\`source-incomplete\`、曖昧な二重正本はunresolved。 | #2621/#2823/#1180。Input byte・Loaded occurrence。 |
| **TT-12 Pattern→Operation** | Function patternを1件追加、\`tei operations\`は既存8 derived/0 declaredのまま。 | 追加機能の実Operation到達をPASSにしない。\`binding unresolved\`。成功件数の分母に未接続要求を含める。 | #2507/#1180、main #3145/#3167を再利用。 |
| **TT-13 Topology/grain** | 同一親のD1=2件・D2=3件、JOINでは6行。Chartは集計1行。 | JOIN6≠子実体6。Aggregate行に逆写像・対象確定が無い場合、更新Actionを導出しない。 | #2506/#2497/#3192 RP。 |
| **TT-14 Cross-view epoch** | TableでA選択→B選択。Aの遅延Read応答がBのContextに到着。Tabはpartial。 | stale AをBへ混入・同期成功と認定しない。partial Tabの非表示を「対象なし」としない。 | #3222 MV、#2506。 |
| **TT-15 Graph occurrence** | 同一Entityが異なる2本のRelation pathでTreeへ重複表示。 | 同一表示行番号・同labelのみでEntity/Update対象と推定しない。Path occurrenceが必要。 | #2497/#3222 MV。 |
| **TT-16 Media-time** | 文書の行位置とTTS/録音再生timeを関連づけ、テキスト更新後に旧Seek情報を再利用。 | 元Source位置≠再生time。写像根拠と版が無い旧Seekを現在の同時刻と扱わない。 | #3192 RP、#3222 MV、SHO/SUI側。 |
| **TT-17 Persistence state** | ACKを失うがreceiptは取得、Resourceは未読取／別接続で一致／後刻に別値へ更新。 | \`receipt-observed\`はpersistedを主張しない。Resource一致時のみ限定した\`resource-observed\`。後刻の現在値・全関係・Owner完了は別判断。 | main #3155、#1174/#1163。 |
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
