# TEI 設計Context → Test Skill/Harness → 独立検証の統合計画（P-1b→T1）

更新：2026-11-07。対象：Issue #3262/#3223/#1180/#2514/#2605/#1119。実装正本・業務意味Owner・Test Scenario SSOTは置き換えない。

## 1. 最短の縦断目的と現在の証拠

Issueの要求を、生成AIが最低限の指示で**漏れを見つけながら**テスト候補へ変換し、正答を自己生成して追認しないまま、TEI自身の実CLI/将来各Targetで検証する。テスト対象はTEIの構築工程そのものにも徐々に拡大する。

2026-11-07までにmainへ統合されたT0：#3264、#3266、#3268。main push CI #42359610226の実観測はCLI read-only 9件、Product Skill静的14件、生成済Pinned Scenario 6 Step、実CLIが検出する誤り4件と独立義務が検出する誤り2件。局所Go statement C0相当10.1%。**C1未計測**。これは既知Referenceの再実行と判別であって、IssueからAIが未知のTestを新規生成できたという証拠ではない。

他チャットの成果を受け、次の不足を**一本のクリティカルパス**として扱う：

`Required Issue/Scope → 実Source/Role/Parser/Meaning/DependencyのClosure → 軽量Design Context Packet → Test Pattern Proposal → Scenario Candidate → Review → 既存Scenario verify/generate/check → 実Pinned CLI/Target → 独立Expected/Mutant → 観測/未Coverage/再資格化`

各矢印は実行結果または明確な未充足Edgeを記録する。文書生成・スクリプト生成・CLI起動・Outcome受理は別のClaim。

## 2. P-1b：既存Design Context Readiness Consumerの欠落検出を成熟させる

先行統合された[#3284](https://github.com/hat47x/tei-platform/pull/3284)の`tools/reference-issue-test-ability/context_gate.py`は既に**9 CLI実応答を読む試験用Consumer**であり、7種の合成負例と実Runnerの`exploration-only`資格を返す。第二の`packet.py`を置かず、この既存入口を補強する。

今回の[先行Draft #3293](https://github.com/hat47x/tei-platform/pull/3293)は並行実装の重複を発見したため**未mergeで閉鎖**。本PRは既存`context_gate.py`のSource/Deliveryの境界に限る差分のみ。

| 項目 | 既存Context Gate | 今回の成熟差分 |
| --- | --- | --- |
| Source/Subject | `TARGET`と`REQUIRED_SOURCE`をテスト用独立Requiredとして固定。Loader観測とCatalog/Inspectを区別 | 同一Required SourceのInventory重複、`workspace-files.result.sourceRootFindings`**だけ**が報告した欠落を判別 |
| Tool受領 | 9実応答のraw SHA256とJSON・schema検査。7負例中のTool欠落はメモリ上の模擬 | **実受領Byte列を書き換え**、欠落・途中切断・改行欠落・JSON重複Key・Unicode escape同値重複・NaN・末尾追加JSONを独立テスト |
| 資格 | `boundedReference=partial`、`testExecutionAdmission=exploration-only`、`qualifiedTestGeneration=false` | 異常Byte/Sourceで`blocked`になり、テスト生成/探索実行とも拒否することを実行で判別 |
| Meaning/Disclosure | `productMeaningAccepted=false`、`externalHarnessDeliveryComplete=null` | **維持**。local JSONが正しいことから外部Tool受信・開示資格を導出しない |
| Source Closure | Loader読取はParsed Occurrence/Role/Meaningを証明しない | **維持**。製品のTyped Context Queryは#3262の後続P-1b1で実装 |

`context_gate.py`は製品APIでも、TEI Source Reader/Resolverでも、業務意味Oracleでもない。失敗/未知/不明をResultで表す、製品の実CLI JSONを消費する検証用試作に留める。既存Command Catalogに含まれないPlugin機能や外部Providerの不存在は推論しない。

**新しい阻害点**：Source Root findingは`validate`と`workspace-files`という異なる投影に現れる可能性がある。片側だけを読んだConsumerが「必要Sourceは揃っている」と誤判定すれば、候補生成の前提資格を誤る。今回の最小修正で両方をチェックし、片側だけに不正が出る合成負例を追加した。これは**本当のRequired Source/Parser/Meaning閉包が実現した証拠ではない**。

**外部Harness受領**：#2509実H3 Skill Loadの限定観測は先行するが、8,192byte Catalog途中切断やRequest/Reply欠落を解決したとは主張しない。ローカルTool Reply単位で安全でも、外部AIへのDeliveryは現在もunknownとする。

## 3. T1：未知のテストを生む能力の判別

[#3273](https://github.com/hat47x/tei-platform/pull/3273)は同じ計画文書へ§13を追加する**既存Draft担当**。対象は公開Referenceの`sales.order_line.line_number_positive`を独立に満たさない行番号0。**今回のPacket PRで別のScenario/Oracle/Runnerを追加して同じT1を実装したとは言わない**。

T1受入を、AIが提案したケース数でなく以下の一回の成功またはFailで測る。

1. Qualified Design Contextから**Ruleの必要なSource/粒度/Scope/関係/単位**を提示する。未閉包なら生成AIへ`unknown`を出し、対象なしと断定しない。
2. **入力最小化**：利用者の指示は「この対象の未カバーを検出してテスト案を作る」の一件にし、型/参照/設計を再手入力させない。生成AgentとHoldout Ownerは別の役割とログにする。
3. AIは`Candidate`の根拠・期待する分岐・依存・失敗分類を提案する。Candidateが業務正本のExpected/Owner Acceptedに自動昇格してはならない。
4. **独立Expected/Holdoutを候補生成Agentに見せない**。OracleがCandidateのExpectedをそのまま採用しないよう、異なるRule条件・欠損/NULL・版・対象/Scope違い・常時PASSな偽Attesterを検証する。
5. Review後だけ#1180のScenarioへ保存し、TEIが生成したCLI argv+stdinをPinned実Binaryで実行。Exit/業務拒否/実Provider効果/独立再Readを別列にする。
6. Coverage分母を`Qualified Requirements`に限定し、Unknown/非公開/未解析を「0件」へ変換しない。C0(statement)、C1(branch)、Decision GraphのNode/Edge/Guard、Scenario/Requirement、NFR（時間・リソース・失敗時・Disclosure）を**別メトリクス**で報告する。

T1不合格例：候補は作ったがRuleに届かない、正答Holdoutの混入、Scenario Generated Planの改ざん、実CLIは未起動なのにexecuted、Mutantが生き残る、未知のNFR分母を完全Coverageとする。これは#2514の独立検証基準と#1180のScenario SSOTを再利用し、第二検証Engineを作らない。

## 4. 複数領域への展開はCaseを複製せず、共通Observation型で扱う

| 他チャットの領域 | 既存Ownerへ戻す必要Observationと独立負例 |
| --- | --- |
| **標準Data/表現型/Topology** | Data側のgrain/Relation多重度と、Function内の表現型Topology・明示Mapping。単位・軸・NULL・非開示を維持。Viewに表示されない行をSource不存在としない。#1124/#2586/#3223 |
| **HTTP Decision Graph** | Server/Client/Recoveryを3つの参照Graphとして同一構造に束縛。Outcome→Execution/Effect→HTTP→Reaction→Observationを独立列で評価。Node/Edge/Guard・偽HTTP成功/ACK喪失をCase化。既存Graph/HTTP Ownerへ委譲（#1043/#1656） |
| **原子Trait/Plugin/BOM** | TraitはGenerated/Provided/Manual/External等のFulfillmentと実Load/Runtime使用証拠を分離。BOMはexact version tuple/Builder/Consumer/Artifact/Targetと独立Caseの集合を確認。2×2の全行実Run前はverified:trueにしない。#3267、Plugin Owner |
| **DB・Schema Workbench** | Schema metadata表示≠実Row Read/Function Context/Operation。History unknown≠applied、Providerの能力宣言≠現在Grant。実Row/Scope→Action→Receipt→独立再取得を分離する。#3270/#2586/#2892 |
| **Cache/Derived Snapshot** | 旧Snapshot hit≠意味・権限fresh。Source Revision/selected Provider/TTL/Invalidation/Consumer Receipt別。#3271 |
| **SHO/モデル図/CRUD表** | Mermaid/Graphvizは標準Graphの可逆表現・参照であって第二意味正本ではない。3視点Data/Function/Business・CRUDクロスリファレンスはSource/Rule/Operationの到達に紐付け、不可視Edgeを未取得扱い。#1124とSHO独自担当 |
| **全体Self-Build** | T1のScenario成功だけでL2/L3を証明しない。Stage1実TEIプロセスがstage2を再入駆動した実Byte/Graph/Builder/Artifact/Receiptと、テスト定義の再生成・再実行証拠を別に取る。#2831 |

これらをT1のHard Dependencyにはしない。領域ごとのQualificationに必要な場合のみ追加する。新巨大Runner、Graph Workflow Engine、中央Host、中央認可Registryを作らない。対象外のテストCaseを実質的な完成要件へ膨らませない。

## 5. 計画への反映と受入Gate

| Stage | 明示したExit | 状態 |
| --- | --- | --- |
| **P-1a** 実CLI部分取得 | 9実応答、Binary/Source指紋とSource Loaderの区別。誤completeの負例 | **限定main T0既実行** |
| **P-1b0** Consumer Readiness強化 | 既存独立Required/9実CLI応答を使い、Byte異常7種＋Source Inventory異常2種を新規判別（既存7件は変更なし） | **本PRの新実験。CI結果が必要** |
| **P-1b1** 製品Typed Design Context | Requirement/Function/Business/Rule、Source occurrence/Parser/Role/Meaning/Disclosure/Deliveryの用途相対Closure | **NOT RUN** |
| **P-1c / T1** AI Test Proposal + Holdout | 実Harness/Skill Load、Tool完全受領、候補新Case、独立ExpectedでMutant判別、false-qualified、Token/時間 | **NOT RUN**（#3273 Draftは設計のみ） |
| **T2** cross-target | 同一ScenarioのCLI/Web/Desktop/HTTP・実Provider/Host Evidenceと独立再Read | **NOT RUN** |
| **T3 / G4** self-build | TEI表現によるBuild/Test定義、L3実プロセスStage再入 | **NOT RUN** |

**最小ループ**：Source／Ownerを読む→不足の最初の一辺だけIssueへ戻す→修正→同一独立Required/Expectedで再検査→Evidence Digest/実BinaryとAttempt→Qualification更新。main PR Healthや新Artifactを他ブランチへ無断流用しない。

**既存Owner**：#3262はContext/Closure、#2605/TAI #168/#2509は外部Skill/Harness、#1180はScenario SSOT、#2514は独立Expected/Mutant、#3223はテストパターン/カバレッジ、#1163はEvidence、#3267はBOM、#2831はSelf-Build、#1119はGate総括、#2018は効率・コスト。重複Issueや第二Source Readerを作らない。
