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

## 6. 再資格化の最初の実Source試験（2026-11-07／T0.6）

### 6.1 重複実装を避けた新たな成熟差分

main [#3284](https://github.com/hat47x/tei-platform/pull/3284) と [#3296](https://github.com/hat47x/tei-platform/pull/3296) は、9つの実CLI応答を既存の`context_gate.py`で読み、計**16種類の受領・欠落・陳腐化・資格保留を識別する限定Reference**まで拡張した。これは「**Tool応答を後から改変した場合**」の拒否力であり、**生成済みテストを実行するまでに現実のWorkspace Sourceが変化した**負例とは異なる。旧#3293の第二`packet.py`は重複と判明して閉鎖済みであり、再導入しない。

この差分は製品Design Context API、Plugin Contract、Source Resolver、Test Oracleへ機能を足すのではなく、既存実行入口`scenario generate → run`のSource/Lock版固定の判別力を**隔離した実ファイル改変**で確認する。

| 新実験 | 固定根拠 | 新たな操作と独立Expected | Claim Ceiling |
| --- | --- | --- | --- |
| **T0.6-S1** Canonical Source TOCTOU | 元Scenario・World Fingerprint、合成`semantic/sales/order.tei` | 一時Workspaceで旧Plan生成・正常pinned実行 → `externalId`の型を変更 → World Fingerprint変化確認 → **旧pinned実CLIの拒否** → 原byte復元 | 使用時のWorld Pin差異の検出のみ。Source全件/Parser/Role/Owner/Provider閉包は非証明 |
| **T0.6-S2** Plugin Lock Byte TOCTOU | 元Scenario・Workspaceの`.tei/plugin-lock.json`の実byte | 別一時Workspaceで旧Plan生成 → JSONと意味的に等価な空白変更 → Lock SHA差確認 → **旧pinned実CLIの拒否** | Lock byte pinが機能したのみ。Plugin/Providerの**実Consumer Load・Runtime Link・組合せConformance**は非証明 |
| **T0.6-S3** A→B→A | Source原byte、変更履歴と旧Plan | Bの間は拒否、Aへの**byte完全復元時に旧Planが再び成立**する事実を独立に記録 | 「中間版が存在しなかった」「古いOwner承認・Grantが現在も有効」は**推論禁止**。Observation epoch・Effect/Current Stateは別Owner |

**区別する実行段階**：CLI JSONの局所受領・Source Inventory・Parser消費、ScenarioのWorld/Lock byte Pin、Provider実Read/操作、実Host/現在Grant、BOMの検証済み組合せ、Owner Meaningは、同じ`verified`へ畳まない。Sourceが一瞬BになってAに戻った場合の履歴を、content-based fingerprintのみで証明することはできない。

### 6.2 Verified Composition BOMとテスト生成の依存関係

[#3267](https://github.com/hat47x/tei-platform/issues/3267)の標準Verified Composition BOM（構想）では、Reusable Pattern／exact resolved tuple＋Lock／組合せMatrix／**同じAttemptの実Link・Load・Invocationと独立Expected**／Qualified BOMを区別する。今回の`Plugin Lock fingerprint differs`は**そのうちLock byte差分の失効検出だけ**である。

後続T1のPattern推薦では`Required Capability`/Traitの原子的Fulfillment、Provider/Consumer/Target/Versionの適用述語を明記する。最初の組合せ2×2を試す際も **個々のProvider PASSから4行の組合せPASSを合成しない**。組合せと実行したTest Case、Current Grant/Scope、Target/Artifact/RuntimeをEvidenceとして束縛し、ReleaseとOwnerは別判断。BOM製品機能・PluginテストMatrix・Trait強制ValidatorをT0.6の前提にはしない。

### 6.3 Gateと未解決

- **T0.6独立Exit**：実ビルドTEI CLIで、Source／Lockの実物改変が旧Planを拒否し、復元時のA→B→Aでは履歴を自動的に証明できないことを記録。2種類の改変はrunnerの廃棄する**公開合成Workspaceコピーだけ**。main SourceやSecret/外部Providerには触れない。
- **後続P-1b1**：同じSubject/Scopeで実Source Identity・Parser Read・Function/Business/Rule/Relation closureを束縛し、Source Root移動・リンク/パス同名異byte・Source内容は同じだがParser/Toolchainが異なる反例を追加。中央Scannerや第二Source/Meaning Resolverを作らない。
- **後続T1**：必要Subjectの欠落/NULL・一意・時間/金額Unit・グラフ多重性・HTTP例外/再試行とProvider partial/emptyを**独立Holdout**として分母に入れ、同一Instruction Budget下でAI生成CaseのC0/C1・実Mutant kill/survive・誤Acceptedを測定。既存#3273のT1 Case Ownerを複製しない。
- **後続T2／T3**：Plugin組合せBOMは#3267、DB/実Resource Readは#2586/#3238、Consumer Disclosureは#2515/#2516、SHO保存/再開とData/Function/Business図面は既存SHO/Modeler Owner、TEI実Self-Build L3は#2831へ渡す。同じ実行Runから取れない資格はunknown/not-runのまま維持する。

**計測値の扱い**：新CIが走る前の本節は**試験実装・期待結果の計画**であり、実測成功ではない。後からPR exact HEAD、main、実Binary、Artifact、失敗理由をそれぞれ記録する。既存の局所C0=10.1%を新testのC1や全Requirement母集団のCoverageに換算しない。

## 7. T1-A：既知だが未実行のRule境界をAI支援でScenario候補へ変換する最小実走

### 7.1 現時点の現実的な入口と独立性の限界

2026-11-07のmainでは、[#3273](https://github.com/hat47x/tei-platform/pull/3273)が**未カバーRule `sales.order_line.line_number_positive`の行番号0負例**をT1候補として提示している。ただしDraftはdocs-onlyであり、生成済み実TEI CLI Scenarioでの新Case実行証拠ではない。T0/T0.6の6 pinned Stepを何度再実行しても、当該境界を新たに検査したとはいえない。

この差分は、既知候補を**生成AIが参照したDesign Contextから試験用Candidateに落とし込む**初回ドッグフーディングとして、実ファイルに基づく独立の基準と実TEI CLIを接続する。

- 入力の根拠：公開`reference/workspace/semantic/sales/validation.tei`の`validation sales.order_line.line_number_positive positive`（現実のRule構文）と、[#3273](https://github.com/hat47x/tei-platform/pull/3273)の未実施境界。Canonically解釈するのは既存TEI Loader/Evaluatorであり、試験用Pythonは**第二DSL Parserではない**。
- 候補：既存`order-entry-pattern-smoke.json`の**一時コピー**へ`validate-line-number-zero-proposal`を1 Stepだけ追加。合成Fixtureは既存の数量正・Relation Key一意を保持し、`line_number=0`だけを変える。CandidateはOwner採用前の試験用表現であり、#1180の保存済みScenario正本を勝手に書き換えない。
- 独立性：期待する正数特性は、Candidateに書かれたExpectedの自己コピーでなく、数値`line_number>0`の独立Propertyとの照合を行う。行番号1で正常、0で業務not-satisfiedを確認。さらに**Expectedだけ偽装**、**Fixtureだけ正常に変更**、**Plan後Fixture改変**の3つの負例が実CLIで拒否されることを照合。
- Product実行：既存CLIの`scenario verify`では未評価扱い、`generate`でScenario/World/Plugin Lock/FixtureのPinを生成し、`check`で改ざんなし、`run`で実`validationeval.Evaluate`から3 Ruleの状態を取得する。
- 事前資格：#3262の既存9 CLI Reply/16欠落・陳腐化検査で`partial/exploration-only`を維持し、`qualifiedTestGeneration=false`。これは**T1候補の実行許可を限定的に与えるだけ**で、全設計の自動テスト生成資格を意味しない。

### 7.2 受入と、今回取得しない証拠

| 軸 | 最小Expected | 判定上限 |
| --- | --- | --- |
| **新たなCaseの実TEI CLI実行** | 既存6 Stepを保存したまま候補7番目を一時挿入し、generate/checkから正確なPinned run、業務`line_number_positive=not-satisfied`、他2 Ruleはsatisfied | **候補1件の実行能力**。全Rule/全要件/意味Ownerは未受理 |
| **テスト判別力** | 独立Property（0 vs 1）、嘘Expected、Fixtureのみ変更、古いFixture Pinの**3負例**が実CLIで拒否される | Rule Evaluatorへの**実GoコードMutant killではない**。常時拒否の偽陽性を防ぐ正常対照を保持 |
| **生成AIの貢献** | 既存#3273が示す未実行境界を、AIが実Scenario/Fixture/CLI手順に翻訳して検収 | **盲検の未知Case発見ではない**。AI Agent/Harnessが新規にこのObligationを発見したか、外部H3のSkill実ロードにより品質改善したかは未計測 |
| **SourceとHarness** | validation.teiの原byteハッシュ、基点Scenario SHA、実CLI Binary SHA、設計Context状態、実生成Pin、CI run/Artifactを記録 | Source→Parser/全Required閉包、現在Consumer Grant、外部Harness受領完全性、独立Holdout秘匿は未資格 |
| **CoverageとSelf-Build** | 新規Rule境界1件、Go局所statement C0とC1を別報告 | 未確定Required分母を覆わず、実DB/Host/HTTP/Web、G4 L3を推論しない |

試験実装は`tools/reference-issue-test-ability/t1_line_number_candidate.py`。**設計の正本・既存Rule・Scenario本体は変更しない。** 候補の採用は#1180/#3273のOwner判断。将来この候補を恒久Scenarioに追加する場合は、本試験の6→7の一時コピーと独立Expectedを、保存済みScenarioの版・Case IDへ昇格する差分としてレビューする。統合前にCaseを勝手にAccepted Meaningへ変更しない。

### 7.3 次に必要な盲検T1-B

1. #3262 P-1b1で、**別の未カバーRuleを含む**狭いFunction ContextのSource/Role/Parser/Relation閉包をQualifiedにする。欠損があれば既存`context_gate.py`の`unknown`に返す。
2. 生成AIの要求入力・出力Schema・Tool予算・Context Revisionを固定し、**独立Holdoutの具体値/欠陥版を生成側に知らせず**候補を提案させる。旧#3273の行番号0という先行ヒントを与えたSessionは、真の「未知発見」ベンチマーク分母から除く。
3. 候補をOwner Reviewして既存Scenario契約へ登録し、実CLI/必要Targetに生成→Pinned実行する。#2514の第三者Holdoutに対し実EvaluatorコードMutant・Source欠落/NULL・scope/粒度/単位違いの検出とfalse-qualifiedを測定。
4. Skillなし/H3 Skillありの比較は[#2509](https://github.com/hat47x/tei-platform/issues/2509)の同一Participant/同一Tool受領・実OS/Network開示資格が整った時だけ実施する。外部Tool受領が切断されたSessionの失敗/時間/コストはITТ分母に残し、Skill有効性の比較には採用しない。

**新実走前の状態**：この節は候補の試験実装と受入条件。実CI/run/Goの新結果は専用PR HEADを確認して追補する。T1-B、実GoコードMutant、H3の盲検比較、C1、全Required/NFR、Owner Meaning Acceptanceは**NOT RUN**。

### 7.4 最初の実行記録（2026-11-07／候補の実行、T1-A）

- **候補プログラム実測HEAD** `02d4c624339fc2d6241703e9a9148ccf46fed99e` の[GitHub Actions #42378033965](https://github.com/hat47x/tei-platform/actions/runs/42378033965)、Job `128419576184` は**completed/success**。仮統合Checkout `b9ff3b32af3d1b88056848751b87fbc351899e11`、実TEI CLI SHA256 `5e0bf9a15a2f83417c8303ea1f58df25ab80bd43c09bf51649621bb2a6ca01b2`。
- **同一CI内で確認できたこと**：既存9 read-only実CLI応答とContext拒否16件、保存済みScenario 6 pinned Stepと実CLI拒否6＋独立条件2は従来通りPASS。その後、#3273候補を基点Scenarioの**一時コピーへ7番目として追加**し、実TEI CLIの`scenario verify`（未評価）→`generate`→`check`→`generated-pinned run`を実行して`sales.order_line.line_number_positive=not-satisfied`、数量正・Relation一意2 Ruleはsatisfiedを観測。
- **実際の判別**：新候補の`expectValidations`だけを偽装した場合、合成Fixtureの行番号/対応Relation Keyを両方1に変えた場合、生成後Fixtureだけを更新した場合の**3負例**は、実CLIがそれぞれ正しい理由で拒否。合成Fixtureは`line_number=0`と当該行のRelation Key `0`を一致させ、2行のKey `0`/`2`を一意に保持。単なる正数Ruleの成功自己申告ではない。
- Artifact `tei-issue-test-ability` ID `13576656749`、約18KiB、7日保持に`t1-line-number-candidate.json`と従前の限定証拠を保存。業務データ／正式Scenarioを変更していない。
- **到達上限**：これは**既存Issueで選定されていた境界1件を実CLIで初めて検収**したこと。外部AI Harnessで未知Ruleを盲検発見した証拠ではない。候補の恒久採用/#1180正本更新、Source→Functionの全Required閉包、C1/全業務要件・NFR、実Provider/Host、独立Product実装コードMutant、Owner Meaning受理は**NOT RUN**。Contextは引き続きpartial/exploration-only、`qualifiedTestGeneration=false`を維持。
- **注意**：本追記は実Run後の別commit。上記は先行HEADの成功であり、**この文書追補を含む最終PR HEADおよび統合後mainの合格は、各Runを確認するまで主張しない**。

## 8. P-1b1最小差分：JSON正常・同一Fingerprintでも欠ける必須Ruleを検出（TT-56）

### 8.1 設計判断：独立Requiredと既存Product投影の**差集合**

既存T0/T0.5は`semantic-data:sales.order`一件のCatalog・Inspect・Source Loader照合と、9 CLI Reply/Manifest/Transportの16種の負例検出が成立した。T1-A（main [#3304](https://github.com/hat47x/tei-platform/pull/3304)）は`line_number=0`の一候補を実Pinned CLIで実行した。ただし**テストで必要とする別RuleをCatalogが表示しなくても、表面的なData対象・Fingerprint一致だけなら「条件がそろった」と誤認し得る**。これはTool Replyを壊す負例やSourceを実改変してWorld SHAを失効させる負例とは異なる。

最初の照合対象を公開合成`sales.order`のValidation評価3件とし、**必要な3 Subjectの集合はCatalogや候補Scenario結果から導出しない**。

| 独立Required（test-only） | 正本での表現 | 既存Product CLIで観測するもの |
| --- | --- | --- |
| `sales.order_line.quantity_positive` | `validation.tei`のpositive Rule | Catalogの`semantic-validation`正本存在、`sourceStatus:indexed`、Source Indexの当該`definition`参照 |
| `sales.order.lines.line_number_unique` | 同Relation内の`relation-unique` Rule | 同SubjectのSource Anchor/Role、Inventoryの`readByLoader` |
| `sales.order_line.line_number_positive` | 行番号正数Rule | 同SubjectのCanonical Kind/Index Reference、T1-Aの候補と同じSource Bundle |

この3つに対し、`workspace-files.result.managedFiles`の`semantic/sales/validation.tei`が1件で`readByLoader:true`、`source-index`が`mode:index`で同Source Artifactを1件、各Definition Referenceの`subject/kind/role/artifactPath`を一意に持つことを**既存CLI投影の値だけ**から確認する。受領したCatalog/Index JSONを検査するため、DSL構文をPythonで再パースせず、Source位置と意味正本は依然TEI既存Loader/Source Indexが保有する。

**決定的な非推論**：CatalogやSource Indexで一致したことは、その3件についての**限定的な読取・追跡可能性**の証拠であって、全Function/Business/Rule/ProcedureのRequired集合、業務意味受理、Source解析の完全性、Graphトポロジーの全辺、現在Host権限/Plugin実Load、実DB Readを証明しない。`boundedReference`は`partial`、`testExecutionAdmission`は`exploration-only`、`qualifiedTestGeneration=false`を**維持**する。

### 8.2 独立の反証と、欠陥を過剰に検出しない対照

| TT-56 | 独立に変えるもの | 期待する判別 |
| --- | --- | --- |
| A | Catalogから必須`line_number_positive`だけを削除 | `required-rule-catalog`でblocked。JSONもFPも正常 |
| B | Catalogの必須SubjectのKindだけを変更 | Subject名が同じでも別Kindなのでblocked |
| C | Source Indexの当該Definition Referenceだけを削除 | SourceがInventoryに残っていてもblocked |
| D | Definition ReferenceのRoleを`derived`へ変更 | Source参照の存在だけでRole充足としない |
| E | Definition ReferenceのArtifact Pathを別Sourceへ変更 | Subject文字列一致だけでSource束縛充足としない |
| F | Catalogに同一必須Subjectを重複追加 | 一意選択不能はblocked、first/last winsにしない |
| G | `validation.tei`はInventoryにあるが`readByLoader=false` | 掲載だけでは実Loadedでないためblocked |
| H | **必須集合の外側**の無関係なCatalog要素だけ削除 | **この限定Requiredの判定はpartialを維持**。全Workspace閉包を暗黙に評価しない |

A〜Hは既存TEI CLIが返したJSONの**test-onlyメモリ上のコピー**を変更する。A〜Gは7件の拒否、Hは1件の過剰拒否防止。World/SourceIndex Fingerprint欄を変更せず、`Catalog+Indexの表層指紋一致`だけで必須集合の完備を認定しない。実Toolの実配送・本番権限・Sourceファイルの実改変やConsumerの実ロードをテストしたものではない。

### 8.3 他チャット領域のRequired/Gapを同じ構造へ統一する

今後のTEI標準Data/Function/Business・Topology/単位系・標準表現型・Graph/SHO/CRUDの情報も、特別な第2Context Readerを増やすのでなく、必要な構造の**独立Required→実Product投影でのAvailable/Indexed/Loaded/Parsed/Consumed→未解決Gap**で管理する。ただし各領域の全件性を3 Ruleの試験だけで証明しない。

| 領域とOwner | 後続の具体的な同型負例／依存 |
| --- | --- |
| **標準Data/Topology/Unit**（#1124/#3223） | Relation向き・多重性・同名別Identity、NULL vs missing、集計grainとFTE/金額単位。図の不可視辺/空Tableを「存在しない」としない |
| **Function→Query→Provider**（#2586/#3283） | Function Source宣言と実Target Build収録・Entry、Query SQL byte/bind、実Provider Read、部分集合/0件、Consumer現在Grantを別義務。合成Browseは実DB結果としない |
| **Plugin/Trait/BOM**（#3278/#3267） | Contract/Provider/Atomic Traitの宣言・手動実装必須・同一Bundle実Load/Call、exact version tupleと2×2の独立Conformanceを別列。Lock byte一致を4組合せVerifiedとしない |
| **HTTP Decision Graph／SHO**（#1043/#1656、SHO担当） | Node/Transition/Guardの必要集合、Runtime Audit／HTTPレスポンス／Client Reaction／Receiptの別証拠。Mermaid等は表示投影で意味正本ではない |
| **CLI/MCP/H3 Harness**（#2605/#2509） | 実Tool Reply ID・完全Part・EOF／開示Grant／Source Revisionを対照。受領済みJSONの一時改変試験で8,192byte途中切断や未返答Requestを解消したとしない |
| **TEI自身のTest/Build**（#1180/#2514/#2831） | 同一義務からCase生成・独立Expected・Mutant判別。Stage1実TEI→Stage2再入Buildの実Artifact/Receiptと一般`go build`を別資格にする |

今後の候補の第一出力は「テスト候補一覧」だけでなく、`missing source / wrong role / not-loaded / unaccepted meaning / not actually consumed / unknown disclosure / true uncovered test` の**正しい不足理由**を先に示す。IssueやAIの要約だけでProductの実Requiredを捏造せず、Ownerの意味・現在Host Grantを自動承認しない。

### 8.4 受入と残りの最短実行順

- **TT56/T0.7（今回）**：既存`context_gate.py`に限定Rule Slice照合を追加。P0 Scenarioの旧6 StepとT1-A Candidateは一切書き換えず、同じCIで16既存Control＋7拒否／1非過剰拒否の追加検査を行う。実PR HEADとmain pushの**各別CI**からEvidenceを残す。これは実測成功確認まではNOT RUN。
- **P-1b1次段階**：一Functionを独立Requiredで定義し、Data/Function/Business/Relation/Rule/Source Anchor/Build Entryを**操作用途に必要な範囲**で列挙。不明と未対応、Consumer非開示を混同せず、既存typed operationsと同一Source IndexからCLI→MCPへ投影。下流への受領・Deliveryと実Provider Readerは別Gate。
- **T1-B**：そのRequired集合を生成AIに与え、候補の独立Hidden Holdout・Source Revision/Scope/Provider/Build Target一致を、別OwnerのExpected/Mutantで検収する。#3273の既知行番号0を盲検未知発見の成功数に算入しない。
- **開発上の停止線**：新Core Type/第二Query/Parser/Resolver/Scenario SSOT/Host/Authority/証拠Storeを先行設置せず、1つの実不足辺が解消する度に同じIndependent Requiredで再測定する。C0(statement)・C1(branch)・Graph Node/Edge・Scenario Requirement・NFRの分母を相互換算しない。
