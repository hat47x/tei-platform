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
## 2. P-1b：Design Context Packetの最小境界

今回のReferenceは`tools/reference-design-context-packet/packet.py`で、**既存TEI CLIの9種類のJSON出力を読み取るConsumer**を構成する。TEI Source/DSLの新Parser、第二Resolver、権限判定器、製品MCP Serverではない。Obligationの独立定義は`tools/reference-design-context-packet/obligations.json`に固定し、観測済Catalogの行からRequired集合を逆生成しない。

| 投影フィールド | 意味 | 現時点の主張上限 |
| --- | --- | --- |
| `obligations` | 対象Kind/ID、独立Required Sourceと必須観点 | 今回は`semantic-data:sales.order`と`semantic/sales/order.tei`のみ。Issue全体Requiredは未解決 |
| `cliEvidence` | 実CLI応答ごとのByte長・SHA256と完全受領 | CI Runner内のローカルJSON。外部Harnessの受領ではない |
| `versionAgreement` | catalog/inspect/trace/validateのWorldとSource Index指紋 | 同一文字列であってもSourceの読み取り中変化や実Target資格を証明しない |
| `sourceLedger` | Required SourceのInventoryと`readByLoader` | `readByLoader=true`でもParsed occurrence/Role/Meaningは未観測 |
| `data/facets` | Dataの実Subject同定、Function/Business/Rule/Dependency等の未充足 | `observed-subject-only`はData全閉包でもOwner採用でもない |
| `delivery` | ローカルToolから外部LLMへの完全受領・Disclosure | このReferenceでは`null`（unknown）。外部送出しない |
| `state/claimCeiling` | 不完全・版競合・部分観測と、昇格できない資格 | 本Referenceは`partial`上限。complete/test-generation-qualifiedを偽装しない |

合成負例は9件：Subjectが消える、Required Sourceが未Load/消滅、SourceIndex指紋競合・World指紋欠落、Source root findingがある、Tool reply欠落・切断・別Schema。部分受領を空集合へ畳まず、欠落時は`incomplete`、競合時は`stale`へ倒す。ローカル完全JSONだけでは#2509の8,192byte途中切断や56 Request/55 Replyを解決しないので、**外部Deliveryは引き続きunknown**。

**成熟してもまだ不足する点**：新Packetが`partial`を返すことで「誤ったcomplete」を拒否できるが、意味的なRequired Domain/Function/Business/Rule全閉包を生成する機能自体を実装したわけではない。正式P-1bでは既存read-only Typed Application ServiceからParsed/Source occurrence/Dependency/Current Disclosureを取得し、CLI/MCP/Skill/人間向け表へ**同じ意味の投影**として提供する。中核をPython Consumerへ固定しない。

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
| **P-1b0** Consumer Packet | 独立Requiredと9実応答を結び、9負例でfail closed、Artifact保管 | **本PRの新実験。CI結果が必要** |
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

**受領経路の追加**：従来の9件に含まれる`source-index -subject-kind semantic-data -subject-id sales.order`は**subjectモードであり**、必要な別ValidationのDefinition Referenceは含まない。そのまま9件だけでRequiredを検証済みと称することはできない。そこで実CLIの**既存全件モード** `tei source-index -workspace reference/workspace`を10番目のread-only Replyとして独立に取得する。CLI自体の新設や第2Source Index実装ではない。Tool Reply 10件それぞれの生byte digestを限定Evidenceに含めるが、順次取得のため原子的Snapshotとは主張しない。

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

## 9. TT-56の二系統を統合：独立Required 3 Ruleと、個別Ruleの構造・Relation照合（2026-11-08）

### 9.1 併走Draftを単一のテストConsumerへ収束

2026-11-07時点で[#3324](https://github.com/hat47x/tei-platform/pull/3324)と[#3326](https://github.com/hat47x/tei-platform/pull/3326)は**別々に**既存の`tools/reference-issue-test-ability/context_gate.py`と同じCI/本計画を編集した。両PRのdedicated CIは各HEADで**成功**したが、二つを独立にmainへ順番mergeして一方のRequiredを上書きしてはならない。今回の統合候補は**#3326の「3 Ruleの外部独立Required」**を基底とし、#3324の「Canonical/Relation/Definitionへの到達」を同一Consumerの追加検査として融合する。

- **既存9 read-only CLI Reply**の他、`source-index -workspace`（既存のIndex全件モード）と`inspect -subject-kind semantic-validation -subject-id sales.order_line.line_number_positive`（既存のtyped Inspect）を各1回追加し、**11本の実TEI CLI Reply**を同一Binaryで受領。11回の逐次Readなので、原子的Snapshotや実H3 Tool完全受領ではない。
- #3326のテスト専用独立Required 3件（`quantity_positive`、`line_number_unique`、`line_number_positive`）は維持。Catalogの一意性・Declared Kind、SourceIndex全件の同Source定義Referenceの一意性・`role=definition`・正確なSource Path、InventoryのLoader Readを検査。7拒否＋1過剰拒否防止の**8 controls**を維持。
- #3324由来の追加Requiredは、既存T1-Aが実行した`line_number_positive`の`canonical.id/kind/attribute`、`sales.order.lines` Relationの一意性とParent=`sales.order`、実`inspect.sources`と全SourceIndexの**同じDefinition Reference/Artifact Fingerprint**であること。**8 controls**（Relation省略・二重・Parent違い、Rule Canonical属性/kind違い、Inspect側Definition/Artifactの欠落・不整合、Inspectだけ改版）を別検出し、既存8件と意味の同一な検査を重複加算しない。
- 先行ベースの**16 controls**（7基本、2 Manifest、7受領Byte）も同一CIで実行。11 Replyの実Byte SHA256、Rule slice／Focused Structure、各Control判定を既存Artifactへ保存し、第二Packet Store、第二Resolver、第二DSL Parser、新Host/Authorityは作らない。

### 9.2 データの同一性を巡る明示的な境界

**SourceがCatalogにある**、**定義SourceがLoadされた**、**Source Indexに位置がある**、**Canonicalが期待形である**、**Rule/Relation間の親が対応する**、**世界とIndexのFingerprintが一致する**は相互に必要だが、どれか一つで他を代用できない。反例として、応答側でRuleだけ削除してもFingerprint文字列が変わらない可能性を別に検出する。これは「ハッシュが壊れている」というProduct欠陥ではなく、**非原子的・部分的なConsumer Deliveryを独立Required集合と照合する必要性**を示す。

受入資格は引き続き `boundedReference=partial`、`testExecutionAdmission=exploration-only`、`qualifiedTestGeneration=false`。新検査の`focusedStructureSlice=bounded-present`は**この一つの公開合成Rule/Relationの既存CLI投影が独立Requiredと合致した**ことだけを表す。Function/Business/全Rule／実Provider・NULL/partial／全Source Parser／権限とDisclosure／外部H3のReply完結／Owner Meaning Acceptedを全資格へ昇格しない。A→B→Aの中間履歴やPlugin Lock byte一致を現在のGrantに変換しない。

### 9.3 他レーンへの最小出力

| Owner | 渡す確認済み必要条件と次の未達 |
| --- | --- |
| #3262／#3223 | TT-56の独立Required 3 Rule+Relation/Canonical/Source Anchor差分、同FPでの部分欠落・過剰拒否防止、11実CLI受領。次に1 Functionの必要Role/Relation/Rule/Source閉包を用途ごとに定義し、`known/partial/unknown/unsupported`で返す |
| #3273／#1180／#2514 | T1-Aの既知行番号0 Candidate実走を維持。**T1-Bは別の独立Required**と非開示Holdoutで未知テスト提案を採点。Candidate/Execution/Independent Expected/Code Mutant Killは別証拠、Owner採否を第2 Scenarioへ保存しない |
| #2509／TAI Harness | CLI単独の11 Replyは原子Readや外部Tool受領の証明でない。Request/Reply対、Part Digest、Reply End/EOF、Current Disclosure/Grantの確認ができる実H2/H3だけを比較母集団とする |
| Plugin／Trait／BOM（#3278/#3267） | Rule Source/Revision照合の型を流用できるが、Plugin exact tuple/Builder/Runtime Load/Consumer Invocation/Target/Grantは独立の事実。2×2全行PASSへ飛ばさない |
| Function／Graph／SHO／Build（#2586/#3283/#2831） | Data/Function/Business 3視点、CRUD、Function→Query/Provider、Build Stage Driverを独立必要集合で追跡。Graphの表示欠落をモデル不存在としない。Source/CLI Contextが一部揃っても実Target資格やStage1→2 L3は別 |

### 9.4 受入Gate

新PRのexact HEADでPython Consumerが実11 CLIを読み、7+2+7+8+8（合計**32個の試験用判別Control**）と、従前の6 Pinned Scenario＋T1-A一時Candidate/独立3負例を正常に再実行することが当面の受入条件。**32は独立したProduction Mutant Kill数ではない**。新たに追加したRule/Relation構造上の8負例は**CLI応答の一時コピー**を改変しており、実Sourceファイルの変更、実H3の部分配送や製品の新Behavior修正を意味しない。

ベース#3326／#3324の各PR専用CI成功を、統合候補の実CI成功と取り違えない。PRのexact HEAD、統合後main push、実BinaryとEvidence Artifactは別々に確認する。統合候補が確認できた場合に限り、**#3324/#3326の別々のDraftを重複統合せず、後継PRへ置換してclose/unmerged**とする。監査証拠は各既存PRとmainの参照を残す。

**main統合後の更新（2026-11-08）**：本§9の統合変更は[PR #3354](https://github.com/hat47x/tei-platform/pull/3354)によりmain `39d4debae386442c11149c7d3995bcfffa711c20`へ統合され、[main push CI #42392054820](https://github.com/hat47x/tei-platform/actions/runs/42392054820)は**success**。この成功は11実CLI応答・32局所Controls・既存Scenario6＋既知T1-Aの限定範囲で、完全設計受領／Owner意味受理の資格ではない。T1-Bの生成AI盲検発見、C1/全NFR、外部Harness Reply End、実DB/Host、Owner受理・G4 Self-buildの新実証はまだない。

## 10. Context→Scenario Replay：Source Index改訂と実行時点の資格を分離する（2026-11-08）

### 10.1 問題と先行検証

[#3354](https://github.com/hat47x/tei-platform/pull/3354)の11実read-only CLI・32局所Control／既知T1-A Caseは、**Contextを取得した時刻と、Scenarioを実行した時刻の同一性**までは保証しない。実`cmd/tei/scenario.go`の生成Planは、Scenario byte、Canonical World、Plugin Lock byte、Fixture byteをPinする。一方、**Source Indexの改訂（Source byte/位置・ReferenceのRole）をそのPlanはPinしていない**。業務モデルが同一でもSourceの末尾改行でSource Indexが変化する反例があり、実行結果の一致をSource最新版やMeaning再受理へ昇格させてはならない。

旧[Draft #3383](https://github.com/hat47x/tei-platform/pull/3383)の[専用dogfood CI #42396735765](https://github.com/hat47x/tei-platform/actions/runs/42396735765)は**completed/success**（実TEI CLIによるWorld不変・Source Index改版・旧Context拒否・既存World Pinの実Scenario実行を観測）。ただし同PRの**全体Repo Healthは`gofmt -l . is empty`でfailure**、mainもその後多数更新された。旧PRをそのままmergeして本計画・CI新Packetを上書きしない。**主張は専用dogfoodの実Runに限定し、現在mainへ非競合部分だけを移植して別CIで再検収する**。

### 10.2 必要契約／実行する負例

| 観測の箇所 | 参照する既存正本・実Product操作 | 受入条件／棄却条件 |
| --- | --- | --- |
| Context取得時 | 既存`context_gate.py`が保存した`observations.observedFingerprints` | `partial/exploration-only`、全Required資格は`false`のまま。WorldとSource Indexの両指紋を記録 |
| Candidate作成直前 | **実TEI CLI** `model catalog`の再取得 | World/Source Indexが上記Contextと一致する。保存済み`partial`フラグ単独では許可しない |
| Candidate生成 | 実CLI `scenario generate` → `worldSha256` | Context World FP = `sha256:` + 生成PlanのWorldSHA256（同じCanonical World JSONをGo側がSHA-256） |
| 実際のPinned実行後 | 実CLI `model catalog`の再取得 | World/Source Indexが依然Contextと一致する。2度のReadは**原子的Snapshotではない** |
| Source-only変更 | 一時Workspaceコピーの`semantic/sales/validation.tei`末尾へLF追加 | 実CLIで`World FP unchanged`かつ`Source Index FP changed`を観測し、**旧Contextを拒否** |
| 元World Pinを使うScenario | 上記一時Workspaceに対し生成済み旧Pinned Argvを実行 | World/Lock/Scenario/Fixture Pinだけでは**実行が成立し得る**。成功を旧Source Indexの新鮮性へ昇格しない |
| Source A→B→A | コピー内の原byteへ復元・実CLI再取得 | 元FP復帰は認めるが、途中の変更歴・現在のConsumer/Host Grant、Owner意味受理、実H3 Tool受領を証明しない |

今回の変更は**試験専用`t1_line_number_candidate.py`と既存CIの受入条件だけ**。**第二Source Reader/Parser、Scenario SSOT、独自Host、権限Registry、追加Artifact Storeは作らない**。新しい観測は公開合成Workspaceの隔離Copyのみを変更し、正式Scenario・実運用DBは変更しない。3つの既存独立Expected偽装負例（期待値・Fixture・Pin）は保持し、Source改訂の負例を実Go EvaluatorコードMutation kill数に合算しない。

### 10.3 他の進行レーンへの引渡し

- **[#3262](https://github.com/hat47x/tei-platform/issues/3262)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)**：Source/Role/Revisionの最新性と、テストの真の検出力／独立Holdoutを別Gateとする。T1-Bでは未知のRequiredとHoldoutを生成側に伏せ、実Product Evaluatorのコード変異に対する検出率を別採点する。
- **[#3365](https://github.com/hat47x/tei-platform/pull/3365)**：Scenario部分目録の未観測/未試験とC0/C1/NFRの分母は別Draftが担当し、今回の`probe.py`は変更しない。
- **[#3361](https://github.com/hat47x/tei-platform/pull/3361)／[#2509](https://github.com/hat47x/tei-platform/issues/2509)**：CLIを逐次11回取得しただけでは実外部HarnessのTool Request/Reply/Part/EOF完全受領・開示Grantは成立しない。
- **[#3372](https://github.com/hat47x/tei-platform/pull/3372)／SHO・REA・KON**：逆解析で得た物理FK/UNIQUEと業務Relation、ObservedとAcceptedを区別する。同じSource Identity/Revision/Meaning/DependencyでContextへ渡す設計候補に利用できるが、実Workbench→三視点編集→SHO再開・同一モデル生成は別E2E受入。
- **[#3267](https://github.com/hat47x/tei-platform/issues/3267)／Plugin**：選択済み能力／Lock byte一致を同一Consumerの実Load/InvocationやExact Tuple Verified BOMへ置換しない。現在Scope/Target/Grantは別。
- **[#3360](https://github.com/hat47x/tei-platform/pull/3360)／[#2831](https://github.com/hat47x/tei-platform/issues/2831)**：Stage1生成**参照ビルドドライバ**によるStage2再入は実証済みだが、`cmd/tei`本体／TEI Build Intent GraphでのL2/L3・G4資格は未達。Context FreshnessからG4成功を推論しない。

**受入ゲート**：新main基底のPR HEADで実11 CLI／32局所Control＋既存6 pinned Scenario＋T1-Aと**上記Source-only改訂の新実CLI negative**がPASSし、同PRの差分に旧main計画やWorkflow Packetの逆行がないこと。専用dogfoodと全体Repo Healthは別々に結果を記録し、統合後main pushを再実行して確認する。**検証前の新PR成功は未主張**。C1/全業務・NFR要件、Blind T1-B、実DB/Host、Owner Accepted Meaning、現在Grantは引き続き未資格。

## 11. P-1b1単一Function：宣言上の必要集合と実行可能性を分離する（2026-11-08）

### 11.1 現行mainの実基線と独立Required

[#3477](https://github.com/hat47x/tei-platform/pull/3477)のmain統合版 `29d367198af2281fc8a390eda959ba91088d8542` は、従来11 read-only CLI、32試験用Negative/Over-rejection Control、限定Design Context Packet/9負例、6実Pinned Scenario＋T1-A既知行番号0候補、**Worldが同じでSource Indexのみ変化する実CLI Replay負例**を統合した。[main push専用Run #42461058083](https://github.com/hat47x/tei-platform/actions/runs/42461058083)は**completed/success**。PRへの組込みに使ったRun #42460897028も成功した。ただし局所C0相当9.4%、C1/未知T1-B/外部H3 Provider/Owner Meaning/全Requiredは未資格。

次の最小独立Requiredを既存Product DSLの**公開Referenceから、試験用の独立契約**として固定する。生成したCatalogやScenarioのExpectedをRequiredの出典にはしない。

| 必要な独立要素 | 固定した正答（試験用） | 実TEI CLIの観測・残る資格 |
| --- | --- | --- |
| Capability | `semantic-capability:sales.order.submit`、入力 `order:sales.order`、出力 `sales.order` | 既存`tei inspect -subject-kind semantic-capability` Canonical + Catalog一意性。**業務実行可能とは言わない** |
| Application | `application:order_entry` | 既存`tei inspect -subject-kind application` Canonical + Catalog一意性 |
| NamespaceとRule依存 | Applicationの`semanticUses`に`sales`、3 Validationを一意に参照 | これは**Namespace Uses**であり**Capability invocation bindingではない**。直接call/Entryは未観測 |
| DB requirement（宣言のみ） | `alias=sales`、`target=sales_core`、`access=read-write` | DB契約の存在やアクセス申告を、現在のProvider実Read／Host Grant／DB Commitへ変換しない |
| Source原文・Role | `semantic/sales/order.tei`と`applications/order-entry/application.tei` | `workspace-files`のLoader Read、`source-index`全件の`definition` Role、Inspect Source Reference/Artifact Fingerprintとの一致 |
| 既存対象Rule・Relation | 既存の3 Validation + `sales.order.lines` Relation | #3354の局所Required検査を再利用。Function/Business全Graph/Query全閉包には拡張しない |

**実装する差分**：既存`tools/reference-issue-test-ability/context_gate.py`へ`boundedFunctionSlice`と10判別Controlを追加。実CLI Replyは従来11に上記**既存Inspector操作を2つだけ**追加して計13。11+2は同一製品CLI実Binaryだが連続Snapshotではない。従来32 Controlに**9拒否+1過剰拒否防止＝10**を追加し、累計42試験用判別Controlとして区別する。`tools/reference-design-context-packet/packet.py`の9 Reply／限定Data Packet正本を改変しない（PacketはFunction閉包の受理ではない）。

### 11.2 テストとして固定する不成立・過剰拒否防止

| Case | 独立に変える受領済みCLI JSONの試験用Copy | 必要なReject／非Reject |
| --- | --- | --- |
| F-01 | Catalogの必須Capabilityだけを削除 | missing（FP文字列が同じでもblocked） |
| F-02 | CatalogのApplicationだけを削除 | missing（FP同じでもblocked） |
| F-03/F-04 | Capabilityの入力型／出力型を変更 | Canonical signature不一致でblocked |
| F-05/F-06 | Applicationから`sales` uses／必須Validation参照を一つ欠落 | 宣言上の必要集合不一致でblocked |
| F-07 | Applicationの`sales` DatabaseのTargetを変更 | Requiredデータソース申告不一致でblocked |
| F-08 | CapabilityのInspect Source definition Referenceだけを削除 | CatalogがあってもSource Role/Anchor未充足でblocked |
| F-09 | Application SourceがInventoryにあるが`readByLoader:false` | 管理対象ファイルの存在と実Readを区別してblocked |
| F-10 | 必要集合外の`semantic-type:std.money`のみCatalogから削除 | **この限定Requiredでは過剰拒否しない**。Workspace全件完全性の主張は依然false |

上記は**Sourceファイルを破損させるE2Eではなく、実TEI CLIから取得したJSONのメモリ上Copyを変更するConsumer判別テスト**。`FPの一致`が必要集合の完備を意味しないことを確認する。FPへの署名・改訂・全依存グラフの原子性も新たに認定しない。

### 11.3 現在の資格・次の縦断辺

局所`boundedFunctionSlice.declarationStatus=bounded-present`は、**指定したCapability宣言/入出力型/Application uses/Source・Role**という観測範囲の一致に限定する。最終的な`boundedReference`は依然`partial`、`testExecutionAdmission=exploration-only`、`qualifiedTestGeneration=false`。`functionInvocationBinding=not-observed`、`queryOrProviderRead=not-observed`、`hostGrantAndEffect=not-observed`を明示する。

次に[#2586](https://github.com/hat47x/tei-platform/issues/2586)と照合すべき不足辺は、**Capability宣言がどのFunction/Build Entry/Query・Providerを通って実Consumerに到達するか**の有界な必要集合である。Applicationの`uses semantic sales`から**具体Capabilityを実呼出し済みと推定しない**。Query/Rowsetの部分観測・0件・NULL、現在Provider選択/実呼出し、Host/Disclosure/Owner meaning、Scenario独立Expectedを別Witnessとして保持する。

この選定は他チャットのPlugin-first Inspector（CLI/Studioで実選択Bと呼出Bを一致させる）、REA/SHO三視点と物理FK/UNIQUE観測、KON交換、SEI委譲にも共通する。いずれも**モデルの存在やSource Index一致をAction資格やデジタルツイン全意味に自動昇格させない**。参照用Build DriverのStage再入成功も`cmd/tei`自身のL2/L3/G4資格とは分離。

**新PRの受入**：actual CLI 13 Reply、従前のContext Packet/9負例・既存32 Control・Scenario6＋T1-A/SourceIndex失効に追加10判別が、**同じ実CIで全てPASS**したこと。そのうえでexact PR HEADとmain pushの実バイナリdigest/Artifactを別々に検査する。新PRの実CI成功を確認するまでは未検証とする。
**現時点**：本文は統合変更の受入規定。新統合HEADのCI結果を検収するまでは**NOT QUALIFIED / NOT MERGED**。T1-Bの生成AI盲検発見、C1/全NFR、外部Harness Reply End、実DB/Host、Owner受理・G4 Self-buildの新実証はまだない。

## 10. Source Index版を含むContext→Scenario引渡し：意味の変わらないSource変更を見逃さない

### 10.1 mainで確認済みの先行成果（2026-11-08 JST）

[#3354](https://github.com/hat47x/tei-platform/pull/3354)はmain `39d4debae386442c11149c7d3995bcfffa711c20`へ統合済み。main push [Actions #42392054820](https://github.com/hat47x/tei-platform/actions/runs/42392054820)、Job `128461490209` は**completed/success**、checkoutは同じcommit。11 read-only CLI Reply（9＋全Source Index＋Validation Inspect）、既存16＋3 Rule独立Requiredの8＋Canonical/Relation/Source照合の8＝**32の限定判別Control**、6既存Pinned Scenario Step＋旧6実CLI拒否・2独立拒否、さらにT1-A候補1 Step＋3拒否がPASS。Artifact ID `13581529076`、7日保持。Go局所statement C0相当**10.0%**、C1未計測。これは新しいFunction/Business全Required資格ではなく、外部Harness実受領・Owner Meaning Acceptanceも未実証。

### 10.2 新しく明確になった欠落：同じWorldでも異なるSource Index

**Contextの時点**：既存`context_gate.py`は各CLI応答の`worldFingerprint`と`sourceIndexFingerprint`を照合し、対象3 Rule/Relation/Sourceの必要要素を確かめた。ただし**Contextとその後のScenario実行は別時点の観測**であり、Contextの成功判定だけを保存して再利用することは不適切。

**Scenarioの時点**：実`cmd/tei/scenario.go`の生成済みPlanは、Scenario source byte、正規化WorldのJSON byte、Plugin Lock byte、Fixture byteをpinする。一方、**Source Index（Source byte・位置・role）RevisionはPlanのPinに含まれていない**。これはPlanの意味評価範囲を超えたSource Authoringの全版資格を、成功したScenario実行から推論できないという境界である。

したがって次の二段階を同じ試験Consumerに追加する。

1. 実TEI CLIの`model catalog`を**Scenario候補を作る直前とPinned実行直後**に改めて呼び、Context Reportの`observations.observedFingerprints`中の**WorldとSource Index両方**に完全一致することを求める。既存Model CatalogをConsumerが読むだけで、第2パーサー・Scanner・Hostは作らない。
2. 実TEI CLIの`scenario generate`が返した`worldSha256`と、元Contextの`worldFingerprint`（`sha256:`＋hex）を照合する。両者は既存Goの`json.Marshal(normalized World)`に対するSHA-256であり、**同じSourceから別々のReadが作成したWorldの一致**を検査する。Plugin Lock/Fixture/Scenarioの既存Pinを変更しない。

### 10.3 業務意味が不変でもSource改訂が必要な負例

公開合成Workspaceの**破棄する一時コピー**で、既存`validation.tei`の末尾にLFを1文字追加する。実TEI CLIの`model catalog`を変更前後で呼び、**Canonical World FPは同じ／Source Index FPだけ異なる**ことを観測する。このとき元PlanのWorld/Lock/Fixture/Scenario Pinを付けた`scenario run`は、別Workspaceの同一Canon Worldに対し**成功し得る**。これはRuntime Rule Evaluationの失敗ではないが、**過去のDesign ContextからSource位置・Source Revisionまで現在と認定してよい理由にはならない**。

独立Requiredの照合側はWorldが同じでもSource Index改訂を拒否する。一時Copyの元Source byteへ復元するとA→B→AでSourceIndex指紋は元に戻るが、それは**中間変更がなかったこと、現在Owner Grantが継続すること、外部Harnessが同版を完全受領したことの証明にならない**。別Workspaceの同名ManifestはWorkspace rootの同一性も保証しない。テストは版・内容の必要条件を示すだけで、根拠記録がないSource epochやCurrent Host Authorizationを捏造しない。

| 観測 | 新しいExpected | 禁止する主張 |
| --- | --- | --- |
| Context→Scenario生成 | Context World FP = Scenario Planの`sha256:`＋WorldSHA256 | World一致だけでSource位置・Grant一致とはしない |
| Scenarioの前後 | 現在CLI CatalogのWorld/Source Index両FPがContextと一致 | 連続時間の原子的Snapshot、A→B→A不存在を推論しない |
| Source末尾LF追加 | Canonical World不変、SourceIndexだけ変化 | 処理結果が正しいことをSource Revision同一に代用しない |
| 元Plan＋改変後WorkspaceのScenario Pinned実行 | World同一条件では実行が成立し得る事実を照合 | 実行成立を旧Contextの最新性・Owner意味受理へ昇格しない |
| Consumerの失効判定 | SourceIndex FPが異なる限り対象Contextの再利用を拒否 | 自動再解釈してAccepted Meaning/実Provider資格を与えない |
| Source A→B→A | 内容指紋が復元する | 中間履歴/Authorization/Tool完全受領が復元すると仮定しない |

### 10.4 後続Issueとの調整

- **#3262**：Context Readinessの正本を増設せず、`t1_line_number_candidate.py`が限定的に**既存CLIを直前・直後に再観測**する。将来の製品Scope Queryはatomic transactionの有無、Source Snapshotと実ConsumerのVersion/Role/Grantを明記する。
- **#2514/#1180**：業務拒否の独立Expected・Product Mutant killと、Context RevisionのFreshness負例を**別種類**として計測。今回のSource LFはGo実EvaluatorコードMutantでなく、Source Index変更の失効テスト。
- **#3365**：Scenario目録の欠落とCoverage分母（TT-105/106）は別Draftで検討中。今回のContext→Pin比較は目録・カバレッジを推測せず、既存T1-A候補の時間境界だけ担当する。競合する`probe.py`を変更しない。
- **#3361/#2509**：外部AI Harnessが受領した実Request/Reply/EOF/Disclosure/Scopeと、この**ローカルCLI**の前後観測を混同しない。
- **#3360/#2831**：Stage1で再構築した**reference-go-build-stage-driver**がStage2を駆動した実プロセス試行は成立している。一方、対象は`cmd/tei`ではなく、TEI標準Build Intent/Graphを使ったTEI本体の再入L2/L3は**未実証**。外部Go Seed Trust、単一OS/toolchain/Recipeの制約を残し、G4全体は`UNQUALIFIED`。Context→Scenario成功をSelf-build資格と混同しない。
- **#3267/#3278**：Plugin Lock byte差分を2×2Verified BOM・実Provider Consumer/Invocationへ昇格しない。Source ContextとProvider Contract/TargetのVersion/Current Grantは別Witness。

**今回の変更予定**は`tools/reference-issue-test-ability/t1_line_number_candidate.py`と既存dogfood CI・本計画に限定。新実CI成功確認までは実験結果を**NOT RUN**とする。外部H3の盲検T1-B新Case生成、C1/NFR/全Requirement、実DB/Provider、Owner Accepted Meaning、現在Grantは新規受入しない。
