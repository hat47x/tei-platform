# TEIのData／FlowとEKIジョブ委任・分散資源制御の接続契約（設計・反証案）

- 日付：2026-11-03
- 状態：**設計提案（Decision待ち／実装・実行検証なし）**。EKIの公開API変更、TEI Plugin SPI変更、ジョブエンジン実装、相互依存の採用済みADRを意味しない。
- 目的：TEIのData Model／Data Script（項目Validation・仮想導出・更新時Rule）→Flow／Capabilityから、EKIを**必要時に選択できる実行・資源調停・結果検証Provider**として委任できるかを検討する。EKI自身のControl/Application PlaneはTEIのローコード能力を活用し得るが、EKI Kernelの権限正本と停止・復旧をTEIへ移譲しない。
- TEI関連：[仮想状態・Data Script・Flow／Batch](DATA_DERIVED_TRANSITION_TRIGGER_PIPELINE_2026-11.md)、[状態機械と図表交差](STATE_MACHINE_CROSS_DOMAIN_DESIGN_PRESSURE_2026-11.md)、[Buri継承](BURI_TO_TEI_LIFECYCLE_TRANSFER_2026-11.md)；Issue [#1129](https://github.com/hat47x/tei-platform/issues/1129)／[#1130](https://github.com/hat47x/tei-platform/issues/1130)／[#1174](https://github.com/hat47x/tei-platform/issues/1174)／[#2582](https://github.com/hat47x/tei-platform/issues/2582)／[#2622](https://github.com/hat47x/tei-platform/issues/2622)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)／[#988](https://github.com/hat47x/tei-platform/issues/988)。
- EKI一次資料：[`ADR-0013`](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0013-product-core-and-faces.md)、[`research/integration/orchestrator-integration-and-layering.md`](https://github.com/hat47x/eki-fabric/blob/main/research/integration/orchestrator-integration-and-layering.md)、[`plan/UPSTREAM_PLAN.md`のTEI適用](https://github.com/hat47x/eki-fabric/blob/main/plan/UPSTREAM_PLAN.md)、[`architecture/EXTENSION_SPI.md`](https://github.com/hat47x/eki-fabric/blob/main/architecture/EXTENSION_SPI.md)、[`architecture/TASK_STATE_MODEL.md`](https://github.com/hat47x/eki-fabric/blob/main/architecture/TASK_STATE_MODEL.md)、[`spec/external/openapi.json`](https://github.com/hat47x/eki-fabric/blob/main/spec/external/openapi.json)、Issue [EKI #136](https://github.com/hat47x/eki-fabric/issues/136)、[#112](https://github.com/hat47x/eki-fabric/issues/112)、[#126](https://github.com/hat47x/eki-fabric/issues/126)、[#57](https://github.com/hat47x/eki-fabric/issues/57)、[#242](https://github.com/hat47x/eki-fabric/issues/242)、[#381](https://github.com/hat47x/eki-fabric/issues/381)。

## 1. 結論候補：EKIを実行・資源の専門基盤へ委任する。ただし業務意味と基盤のKernelは交換しない

**適切な役割分担**：TEIで生成・受理したData/Rule/Flow／Batch定義はTEIのMeaningの正本に残す。EKIは受理したTaskのAdmission、資源適合、Placement、Lease／fence、Attempt、reclaim／retry、結果の検証・実行記録・予算制約を担当する。異なる既存クラスタ（Kubernetes/Kueue、Slurm、Nomad等）との統合はEKI側Resource Provider Adapterとし、必ずしもEKI自身で下位Schedulerの全機能を実装しない。

| 関心事 | 意味／権限の所有者 | EKIへ渡してよいものと渡せないもの |
| --- | --- | --- |
| Data/AttributeのVirtual宣言・導出・Validation | TEI Model／Data Scriptの受理済Meaning、対象業務Owner | 必要なInput Artifactと実行体契約は可。値の判断権と業務ルール正本は渡さない |
| Data変化によるFlow起動・対象集合・分岐・次工程 | TEI Flow／業務Application（または既存Domain Orchestrator） | Task/Group依頼と結果のObservationは可。EKI Task stateを業務Lifecycleへ変換しない |
| 分散Job実行、Resource Budget／Deadline、Placement／reclaim | EKI委任Kernel（Task/Attempt/Lease） | Taskの資源要求・データ分類・実行体固定等を受け、EKI独自にAdmission／Authorityを判断 |
| Job成功／成果物検証 | EKIのVerification/AcceptedResult | 実装としてJobが実行された、結果が指定Validationを通過したという範囲のみ。業務の受理・保存・通知完了は別 |
| 業務DB書込み・独立承認・外部副作用 | TEIのCapability/Target Host/Providerと既存Owner | EKIがWorker成功・ResultClaim・Lease終了だけから業務Commitを宣言してはいけない |
| EKI Console／Job Catalog／運用画面／Policy Review | EKIのApplication Plane。TEIで生成する候補 | 権限の編集面・観測・SimulationにTEI全能力を活用できるが、EKI Kernelの鍵・Lease発行やWorker停止権はEKI native-first |

**「EKIがTEIのローコード全能力を使う」の解釈**：TEIを用いてEKIの管理UI、ジョブ定義画面、Job依存図、資源Poolの可視化、Policy Review、監査、Simulation/検証、Operator手順等を制作し得る。ただし**採用判断は個別Conformance**。EKIのWorker、Local Governor、署名・Trust Root、Lease／fencing、Control Plane起動・復旧をTEI成果物の稼働に依存させると循環障害になる。EKI停止時に障害修復用のCLI・鍵管理・最低限の監視を残す（EKI ADR-0013、UPSTREAM_PLAN、orchestrator研究§7.4）。TEIが生成した**Native Artifact**なら実装候補にはなり得るが、Kernel適合／安全性を通さず一律TEIへ移す決定はしない。

## 2. インターフェース：EKI一意の核＋TEI Pluginの任意Bindingを二層化

| 方式 | 利点 | 採用を止める要因 | 判定 |
| --- | --- | --- | --- |
| A：EKI側で全て一意のJob API／契機／Callbackを固定 | Task/Attempt/Lease／結果の意味とVersionを1箇所で管理しやすい | TEI固有のVirtual state・Provider/Host変種・各種WorkflowをEKIのAPIへ押し込み、EKIが業務意味を持つ。高速な用途別拡張を妨げる | **共通核の範囲に限り採る** |
| B：TEI PluginごとにJob依頼／Callback／権限・再試行を任意拡張 | Provider固有の契機やTransportに適合しやすい | Idempotency、Task identity、Lease／fencing、Unknown、ResultAcceptanceがPlugin間で分裂し、EKIが統治できない | **任意拡張を核の代替にしない** |
| **C：二層（候補）** EKI External Contractを一意のAuthority境界とし、TEI Integration PluginがAuthoring/Binding／Transport／Result変換を担う | 依存はTEI→EKI、EKIの契約を守りつつ、Source/Provider/Target固有の編集・変換はPlugin-first | 契約Version／必要Capability／再試行の委譲先を、曖昧なFallback無しで照合する必要 | **推奨する設計比較軸。正式採用は未決** |

**層の切り方**：
1. **EKI公開面**：Task submission／query／cancellation／result照会、Task／Attempt／Lease／Resultの意味・Authority・Idempotency・予算・期限をEKIが一意に規定。TEIに限らず他Clientも同じ契約を使える。EKIに固有のExecution SPIとResource Provider AdapterはEKI内部契約へ委譲する。
2. **TEI統合面**：既存Language Plugin／Operator Contract／Capability／Data Providerとは別責務の**Execution Integration Binding候補**。Data Script Trigger参照／Flow定義からEKI TaskRequestへ適合変換し、結果ObservationをTEIのMeaning／Effect/Independent Expectedへ相関させる。既存Plugin机制を使い、第二のTEI Runtime／Global Registryを作らない。
3. **発火通知・Callback面**：EKIからのイベント通知は将来の**任意Transport**であり、Taskの正本やOutcome資格の代替ではない。初期は既存APIの**pull/query + durable TEI-side correlation**を基準として検証。Push/Webhook/Queueはそれが必要なService Profileで追加し、失敗時pull/照会で回復できる契約とする。

**拡張可能とするのは** `InputAdapter`、`Executor/Artifact mapping`、`Trigger policy/profile`、`Result verification要求`、`Callback transport`、下位Resource Provider。**拡張不能な不変条件**は、EKIが決めるAdmission/Lease/Authority、TaskとAttemptの分離、ResultClaim≠AcceptedResult、古いLeaseの失効、retry budget、未知Effectの安全な回復、EKI/TEI意味正本の独立。

**Version/Capability negotiation**：正式APIに無い`DAG`、`TaskGroup`、`callback`、`effectful-executor`、`strict-derived-edge`をClientが勝手に申告して成功扱いしない。Runtime提供側の公開仕様・配備能力・現在資格を照合し、未対応なら`unsupported`／別Provider候補／安全な非分散ローカル実行へ戻す。保証を弱める自動Fallbackは行わない。EKIの内部`SPI`をTEI Pluginの公衆APIと取り違えない（EKI [#136](https://github.com/hat47x/eki-fabric/issues/136)は研究状態）。

## 3. Data更新→Job発火のTxと時点：五つの境界を分離する

概念上の発火経路（具体Runtime未実装）：

~~~text
TEI Data Model virtual/read-only attribute と Data Script pure Rule
    ↓  同じSubject/Scope/必要Read set・Versionで評価（before/after）
TEI Validation → Operation Intent（期待版と現在Grant）
    ↓
Provider transaction / conditional update（業務Factを書込）
    ├─ 同一Tx Outbox/JobIntent記録（可能なProviderに限る）
    └─ COMMIT  ← この成否・未知を区別
    ↓  committed intent relay / periodic reconciliation
TEI Execution Binding → EKI Task Admission（Idempotency-Key）
    ↓  （受付と実処理、実資源配置、Leaseは別）
EKI Task → Attempt/Lease/Worker → ResultClaim → Verification → AcceptedResult
    ↓  callback notification（任意）またはread/query
TEI Result Consumer（version/Authority/Effect fenceの再評価）
    ↓
業務上の後続Flow判定／書込／Receipt／独立Read
~~~

**Txの責務表**：

| 境界 | 許される処理 | 禁止する結論 |
| --- | --- | --- |
| **V0 pre-write**：項目Validation／純粋Derivation／Derived-edge候補の計算 | Type／Nullable／Relation完全性／Snapshotの検査、diff候補と独立Expected | TriggerがtrueだからEKIの実Jobを開始、任意Script/Callbackで外部Effectを先行 |
| **V1 write transaction**：本体・必要な履歴・outbox | 原子性があるProviderで`Business Write + durable Intent`を一緒にcommit。RollbackならIntentも消える | DBとEKI Task APIが**同一Tx**である／Outbox行=EKI受理済み |
| **V2 after commit**：outboxを届けEKIへAdmission | EKIでTask受理・重複抑止・期限・予算／Data Class制約を確認。EKI停止ならIntentをpendingで保持 | Submit応答ACKやTask ID=ジョブ完了、Commit不明で無条件再送 |
| **V3 EKI実行・結果** | Task/Attempt/Lease、配置・中断/再試行、ResultClaimの検証・AcceptedResult、Trace。CallbackはChange Notification | Worker終了Code／Lease expired=業務保存済み、検証済みArtifact=業務完了、Callback到着=確定 |
| **V4 TEI Business Completion** | 結果とCausationを照合し、対象の現Revision・Meaning版・Owner／Host Grantを再検証。必要な後続業務Effectは別Tx/Receipt | 旧Resultが現在のFlowを無条件進める、外部通知が実配達済みとみなす |

同一DBへのOutboxが利用できないProvider、二つの異なるProviderのread-setが同一Snapshotではない場合は、原子的な`derived false→true`イベントを保証できない。strict edge要件なら`unsupported`、業務で許容された場合だけ**level reconciliation + 冪等なOperation**を比較する。Null／missing／0件／partial／non-disclosure、A→B→A、Rule版更新だけの再評価を、実データの新規edgeやCommit成功へ変換しない。

**技術的参照**：Transactional Outboxは送信を業務更新と同じDB取引へ記録して後から配達するが、Relayは重複配送し得るため受信側の冪等性を要する（[AWS](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html)）。`exactly once`をTEI/EKI全体で無条件に主張しない。

## 4. 委譲Contractに必要な最小の概念（API Schema案ではない）

**TEI側で保持するCorrelation**：`Meaning/Flow/Rule revision`、`Source occurrence`、`Data/Subject ID`、対象の`Record/Scope/Expected row revision`、`Trigger id`、`causation event/commit id`、`operation id`、`EKI task id`、任意`group/step id`、`Result revision`、TEI`consumption status`。生の顧客情報・Provider credential・秘匿Source locatorをEKI Task metadataへそのまま渡さない。

**EKIへ申請する**：実行するJobの機械可読Workload/Executor binding、Input Artifact digest／権利付きAccess、必要Resource Class/Limit、Data Class、許可された配置Profile、Deadline/Budget、Retry可能性／Effect classification、希望する検証保証、Clientの冪等識別子。これらは**既存EKI OpenAPIで全部表現できるという主張ではない**。EKI [#112](https://github.com/hat47x/eki-fabric/issues/112)ではArtifact ingestやExecutor digest、effect mode等にFit/Gapが残る。未採用拡張は最小Scenarioを使ってEKIの既存Issueへ追補する。

**EKIから返る事実**：Submissionの受付か拒否、Task/Attemptの観測、検証前ResultClaim／検証後AcceptedResult、Result Artifact digest／Availability、未知・取消・期限超過・reclaimなどの状態。これらはTEIの`business accepted`／`commit done`とは別。EKI正本はEKI側が保持し、TEIは必要な相関と受領Evidenceのみを保持する。

**Callbackの分類**：
- `onAdmission`：EKIに依頼が受け付けられた通知。Token／Leaseを与えるものではない。
- `onProgress`：進捗はObservation／best-effortであり、順不同・欠落が起こり得る。Flow完了の判定材料にしない。
- `onResultAvailable`：Resultが観測・照会可能になった通知。署名・発行元検証、現在のTask revision/Result Verificationの再照合を経て初めて利用。EKI側のWebhook APIは**現行未確認・新規提案**。
- `onTerminal`／`onUnknown`：Taskとしての終端／実行効果不明を伝えるが、TEIの業務取消／完了とは異なる。
- **再送・再入**：`source+event-id`（またはTask＋Event revision）と受領Inboxで重複抑止、subjectごとのversion fence、古いAttempt/Lease/旧Meaningの結果拒否または保留。Callbackへの2xx応答は**受信耐久化のACK**であって業務Effect完了ではない。

通知Envelopeの相互運用候補に[CloudEvents 1.0](https://github.com/cloudevents/spec/blob/main/cloudevents/spec.md)（source＋id、type、subject等）を比較できるが、EKI固有Trust／Lease／Task Resultの意味をCloudEvents形式だけでは表現できない。現時点で標準採用しない。Webhookを必須にせずQueryから回復可能なBaselineを先に証明する。

## 5. 「大規模Job Controller」の機能分担と必要な反証

| 能力 | TEI側の役割 | EKI側の責務（現在契約／将来候補を分離） |
| --- | --- | --- |
| Data-derived Event／Flow Graph、業務Guard、Step依存の意味 | Model／Data Script／Flowが正本。Diagram×TableとSHOへProjection | Business-Flowの意味は解釈しない |
| Step／Task提出・結果照会・取消 | TEI EKI Integration Binding、後続Flowを進める判定主体 | EKI Task/Attempt/Lease/Resultの標準契約。現実のMVP E2Eは進行中 |
| 10万対象分割、fan-out/fan-in、期限／予算／一群の結果 | 対象集合の完全性・各Step Meaning、再実行の業務適否を指定 | EKI `Task Group`は構想とBaselineの一部。外部API実行保証・依存DAGの所有は未確定（EKI #126）。**GroupとWorkflow DAGを同一視しない** |
| Placement、資源共有、公平分配、優先、reclaim、retry | 最小限の制約と利用者の目的だけを渡す | EKI委任Kernel＋Resource Provider Adapter。下位Schedulerの全機能をEKIで置換しない |
| Job Taskの長期実行・再開・Checkpoint | 再開可能な业务Step／中断時意味／Subjectを決定 | EKI Attempt/Recovery、外部Checkpoint、Worker性質・Data Class。再開できると無条件には保証しない |
| 業務DBの確定・外部通知／銀行API | TEI Capability／実Provider/Host、現権限・Tx/Receipt、必要なら補償 | EKIのEffect Receipt/Provider観測は参考Evidence。EKIだけでBusiness Commit／external exactly-onceにしない |
| EKI自身のJob Console、管理・観測・Policy Review | TEIのLow-Code/SHO/Graph/Tableで生成候補 | EKI Application PlaneはTEI利用可能。ただしKernelの復旧・署名・Lease/Local GovernorはNative-first |

**実行形態は選択的**：同期／単発のローカルTEI Go単一バイナリや既存Hostと、常駐JobのEKI管理を区別する。常駐・永続Job管理を要する場合は、そのDeploymentで既に存在するHost Scheduler／EKI／OS Schedulerのいずれかを**明示選択**し、TEIの意味上のFlowと実行管理のOwnerを別々に固定する。EKIは専門Providerの有力候補であり、唯一の必須経路ではない。低遅延のUIや同一DBの単純更新まで常にEKIへ委譲しない。EKIのResource Suitability Gateが「分散しない」と判定する場合に、勝手にセキュリティ条件を落とさず別の許された実行方法へ戻す。

**既存Orchestratorとの関係**：EKIはKubernetes+Kueue、Slurm、Nomad等をResource Providerとして繋ぐ上位委任Brokerが基軸。Business Step依存を長期間耐久化する高度なJob/Workflowが必要なら、**TEI側がFlow意味と進行判断の責務を保持し**、常駐中の既存Host/生成Application内Coordinator、EKIへの限定委譲、または明示の外部Workflow Providerを比較する。TEIは業務Flowの意味を保持し、実行可能なTask依存へLoweringする。現行EKI Task Groupが依存DAGの連鎖を実装済みとは扱わず、対応不可なら`unsupported`とする。二層Scheduler間の自動再試行・preemption・予算の二重計上はEKI既存調査の負例を再利用する。

## 6. 二つの製品の循環依存・失敗独立性

**依存図**：TEI生成物の通常業務実行は`TEI Flow Adapter → EKI External API`の一方向。EKIがTEIを用いて生成したConsole／Job Catalogは**EKI Application Planeの成果物として**、EKIのKernelと独立に配布・起動できること（必要ならEKI停止中もローカルのreadonly表示／保守CLIで復旧）。EKIのControl Plane起動、Auth鍵・Lease発行、Result accept判断はTEI Runtime/Studioの稼働を前提にしない。

EKIが停止してもTEIのData Model／純粋Derivation／ローカルValidation／同一Providerの必要な業務Txは継続可能な契約を残す。EKI必須の非同期Jobは`pending / unavailable / unknown`で止め、Eventを喪失・実行済みと誤表示しない。

**保証／可用性／責任の分離**：EKIのJob上でDB Effectを起こす場合、ETLの純粋変換と違い重複試行・権限変化・Commit ACK喪失がある。下位Schedulerの`Succeeded`／EKIの`AcceptedResult`／TEIの`Business Completion`は別。TEIのFlowの再実行判断はEKIのAttempt retryと独立であり、二重Retryを作らない。

## 7. 最小Contract/Conformance：現行MVPを壊さずに比較する

| ケース | 独立Expected／観測器 | 成立しなければどう止めるか | Owner |
| --- | --- | --- | --- |
| JD-01 pure Task | TEIの固定入力Artifact＋Executor digestで計算→EKI Task accepted→ResultClaim→検証済み結果→TEI Digest/型と独立Oracle照合。Task IDとOperation IDを分ける | 結果取得不可なら業務へ進まない。既存EKI #112のArtifact/Result不足を明記 | EKI #57/#112、TEI #1130/#2514 |
| JD-02 Rollback | TEI Field更新とOutboxを同じDB TxでRollback。EKIへTask未送信・未受理を独立観測 | Rollback後にFlow実ジョブが動いたら不合格 | TEI #1129/#1174 |
| JD-03 commit→outbox→crash | Commit済み直後に送信前Crash。再起動後、同じEvent/Idempotencyで一つの論理Taskへ収束 | Outbox未処理が消える、二つのTaskを作るなら不合格 | TEI #1129、EKI Idempotency |
| JD-04 callback重複／順不同 | 同じResultが2回通知、古いAttemptから後着、通知喪失を照会で回復。TEIは現在Task/Result/Meaningを再検査 | 古いResultが後続業務状態を進めたら不合格 | TEI #2582/#2514、EKI #57 |
| JD-05 Virtual-edge | 2/3明細のみ取得→`unknown`。3/3完了のafter/readとbeforeを同じScope/Rule版で比較、strict edgeとlevel方式を対照 | empty/partialを`false→true`へ補完しない | TEI #2622/#1708/#2582 |
| JD-06 effectful retry/unknown | Workerが外部書込み後にACK消失。EKI ResultClaim・cancel/timeoutの観測と、Effect AuthorityのReceipt/独立Readを対照 | Lease失効・Worker失敗から副作用不在を断定して再実行しない | EKI #57、TEI #1174 |
| JD-07 Group/DAG/大量対象 | 100,000件を分割。Group budget上限、partial result、timeout/unknown、Backpressure／Checkpoint、Flow Step依存はTEIの独立Expectedで照合 | Group外部API／DAGの実装がない間はunsupported。全完了に偽装しない | EKI #126、TEI #1130 |
| JD-08 EKI停止／TEI復旧 | EKI API不可時もTEI pure/Sync処理を続け、未送信Jobをpending保持。EKI復旧後Intentを調停 | EKI KernelがTEI Studio/生成Admin UI起動を要求したら循環依存不合格 | EKI ADR-0013、TEI #1121 |
| JD-09 plugin互換と権限 | EKI API Version／Executor capability／Callback transportを替え、既存Meaning ID不変。予算・Auth・Resultの不変条件が保持されることを検査 | pluginがTask IDやLeaseの意味を上書きする／保証を弱めるなら拒否 | EKI #136/#112、TEI #988 |

番号は**本稿ローカルの反証ケース**で、EKIのPAS/RQやTEIのConformance番号へ自動登録しない。すべて**NOT RUN**。実Provider／実EKI公開APIの範囲、今後の正式Conformance Ownerが必要。

## 8. TEIをジョブ意味・進行の基本所有者とし、EKIには必要な運用能力を委譲する（2026-11-03、Owner修正）

**前版§8の「常駐Job管理が必要ならEKIが基本」「EKIがJob定義・Trigger・Flow依存を全て管理する」という一般化を撤回する。** ユーザーが明示した基本線は、**ステート管理に基づくジョブ管理の設計責務は原則TEI側**というもの。TEIはData Model／Data Script／Validation／Derivation／Flow・状態遷移、Jobの意味・Step依存・発火条件・業務上の再実行可否を定義・生成・管理する。生成Applicationが既存Hostに常駐し得る場合、状態機械の進行判断やJob Trigger処理もそのHostのTEI生成成果物で担える。

ただし「TEI側の責務」≠「TEI Platform本体が永続サーバを内蔵する」。TEIはModel／Artifact／Plugin/Bindingとして意味を提供するのであり、**Go単一バイナリや埋込みLibraryは常駐を保証しない**。常駐する主体の無い構成では、OS scheduler・既存Host・EKI・他Providerのいずれかへ、**必要な部分だけ**委譲する。EKIを呼ばない単発Job・既存HostのWorkflowまで、EKI登録を必須としない。

### 8.1 三つの所有権を区別して二重状態管理を避ける

| 責務 | 正本・所有者 | 委譲の可否と禁止推論 |
| --- | --- | --- |
| **① Domain／TEI Jobの意味状態** | TEI Data/Attribute・Script・Rule・Flow、Ownerが受理したMeaning、実業務Fact | TEIの生成成果物／既存Hostが解釈。EKIに業務的な`approved`／`billing_ready`の意味正本を移さない |
| **② Flow進行・Job間依存** | TEI側のState/Flowの定義・遷移選択・発火規則。実際の進行記録は当該Deploymentで**選ばれた一つのCoordinator**が管理 | 同一DeploymentでTEI生成HostがCoordinatorになるか、EKI/既存Orchestratorへ**版固定された一部の進行実行**を委譲するかを選ぶ。両者が同一Stepを独立に再試行しない |
| **③ EKI自身の実行状態** | EKIのTask／Attempt／Lease／Result Verification／External Effect／Cancellation／Recoveryの各Authorityと永続Store | EKIがTask受理・配置・fencing・reclaim・Attempt再試行・Result受理を独自管理する。EKI `completed` ≠ TEI Flowの業務完了、EKI Group ≠ 業務Workflow |

**委譲の粒度**は`execution-only`（TEIがStepを決めEKIへTask依頼）、`trigger/schedule`（TEIが発火意味を決め、EKIが時間・変更通知を保持して依頼を起こす）、`bounded-step-coordination`（TEIが受理したステップ依存の**限定した実行表現**をEKIが保持・進める）の三候補。後者は**EKI公開Contractと実E2Eで対応できる場合に限る**。EKIに業務データの意味や任意Scriptを実行する自由権限を与えない。EKI側が依存DAGに未対応なら`unsupported`とし、TEIのCoordinatorがないまま黙って進めない。

### 8.2 Deployment別の最低限の起動主体と安全な設定レス

| Deployment／具体例 | 時間・変更を監視する主体 | ジョブ状態・次Stepを進める主体 | 受入・制約 |
| --- | --- | --- | --- |
| TEI生成の常駐サービス／既存Java・.NET Host | 生成Application／既存Hostのタイマ・イベント／DB subscription | **TEIのFlow Contractを実装したHost側Coordinator**。EKIを使う場合も単なるTask実行先 | 既存Host自身の再起動、Job/Stateの保存と再実行Contractで検証。TEI本体に常駐サービスを追加しない |
| TEI単発Go CLI・非常駐Artifact | OS `systemd.timer`／cron、外部業務Host、EKI等の明示Binding | 単発なら起動元。複数Step・永続Runなら**選択されたCoordinatorが必要** | `timer`をDSLに書いただけでは実行されない。設定・有効化・Credential・障害再開の責任を明示 |
| EKIに部分委譲したJob | 通常はTEI側のCommitted outbox/event、必要に応じEKIが明示登録されたschedule/signalを保持 | **TEI HostのJob Coordinator**が進行を決め、EKIはTask/Attempt/Leaseを実行（execution-only） | EKI状態をTEI意味状態へ勝手にコピーせず、結果をTEI側が再受理する |
| TEI側が停止している間も持続すべきJob | EKIの明示登録Trigger/Job、または既存Orchestrator | **限定的に委譲したStep進行はEKI/Orchestrator**、委譲されない業務判断は`awaiting-domain-evaluation`／保留 | TEIが生成した版固定Executor/Guardと必要なProvider GrantをEKIが安全に実行できる場合のみ進める。EKIの現行APIで対応済みとは主張しない |

**EKI設定レスの最小目標は存続**：EKI未導入のTEI利用を妨げず、必要になった時は単一マシン構成が**管理者の個別設定ファイル編集を必須にせず**安全な既定値で起動する。EKI [ADR-0018](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0018-single-machine-delegation-profile.md)のDP-7はOSの別アカウント・Key/Store保護・Executor分離を要求し、**設計採用済みだが本番実装済みではない**。初回起動で未登録Jobを勝手に走らせない。標準Executorが承認されている単発ローカルTaskのみを段階的な実装Goalにする。外部DB CDC／Worker登録／Data Class／任意コード実行は明示設定・Grantが必要。

`systemd`等はEKI自身のサービス起動にも、EKIを使わないTEI Jobの明示的なスケジュールにも利用できる。しかし**EKIへ委譲したのと同じJobをOSタイマとTEI Hostが重複発火しない**ように一つの発火Ownerと`JobRun/Operation ID`を固定する。Cron／systemd timerの起動成功だけではTask受理・業務Commit・全欠落期間の再実行を証明しない。

### 8.3 Spring Statemachineからの比較：TEIへの一括輸入ではなくEKIの実行状態圧力として評価

[Spring Statemachine 4.0公式Reference](https://docs.spring.io/spring-statemachine/docs/4.0.x/reference/)には階層状態、直交Region、Guards/Actions、Choice/Junction、Fork/Join、History、Deferred Event、Timer Trigger、Listener/Interceptor、State Machine Persister等がある。**機能一覧の存在≠EKIへの必要性／製品要件の証拠**。また[Spring StatemachineのGitHubリポジトリ](https://github.com/spring-attic/spring-statemachine)は2026-07-05にarchivedとなり、メンテナンス終了を明記しているため、Java Libraryへの直接依存を今回の推奨にしない。比較するのは**機構と反例**である。

| 機構 | TEI側での考え方 | EKI側での必要性・現行境界 | Gate |
| --- | --- | --- | --- |
| Flat State/Event/Guard/Action | **基本対象**：業務外も含むState/Rule/Flowの意味表現。IF中心記述と比較 | Task phaseの既存検証・transition subject／CASを再利用 | **両者の共通な検査可能な核**。実Runtime共有は未決 |
| Hierarchical State | TEIにはDomain要求がある場合のみ。見せ方の折畳みとSemantic hierarchyは別 | EKI Job/Task/Groupの集計・状態可視化に役立つ可能性 | Group aggregateのみで十分なら階層Machineにしない |
| Orthogonal Regions | UI／Gameなど必要Domainで評価 | **EKIは既にTask／Attempt／Lease／Result／Effect／Recoveryを独立軸に分離**。これを単一巨大State Enumに潰さない | 直交Region Runtimeを追加する前に、既存独立Store/Invariantで反例を満たせるか比較 |
| Choice／Fork／Join | TEIがFlow分岐・step依存・独立Expectedを定義 | 分散Task Groupの完了集約・結果待ち・複数Attempt/Verifierの技術的合流に候補 | 全員成功／部分成功／未知／期限切れを業務上のJOIN条件へ転用しない |
| Timer／Deferred Event | Flow側は時間条件・Scope/Clock・延期の意味を定義 | Lease expiry、Recovery retry、期限・延期Event・一部利用者の不在中TriggerにはEKIが実行Ownerのとき高適合 | Event喪失/重複/順序・再起動・期限越えのEvidenceが必要。in-memory timerだけでは不十分 |
| Shallow/Deep History | TEI Domainの状態復帰が必要なときだけ | EKIのCheckpoint、再起動後のAttempt/Lease/recoveryに似た圧力だが、**同一Attemptの状態復活を意味しない** | 永続Receipt・Fenceから再構築する方式を優先し、古いLease/Authorityの復活を禁止 |
| Persist/Interceptor、Event Callback | TEI側ではBusiness Commit/Provider Effect Fenceと独立 | EKIはState Revision/CAS、atomic Attempt+Task transition、Outbox/Recoveryが重要。Callbackは通知でありState Authorityでない | Persistence Interceptorの呼出し＝DBと外部Effectの単一Txにはならない |
| Distributed State Machine | TEI自体に単一の分散State Engineを標準装備しない | EKIのControl Plane複数Instanceで正しいAuthority・CAS/fencing・reconciliationを確保する必要 | **共有状態機械の導入を既定にしない**。EKIの既存State Storeとそれを検査するContractが先 |

**優先判断**：EKIにおける高度な状態機械の必要性は、業務Flowの表現力より、**長寿命の複数Task/Attemptと同時Event、部分失敗、期限、失効、回復、複数Controllerの競合**で生じる。ただし現行[EKI Task State Model §3](https://github.com/hat47x/eki-fabric/blob/main/architecture/TASK_STATE_MODEL.md)は明示的に「一つの巨大なState Enumを使わない」と決め、[ADR-0032](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0032-task-phase-transition-subjects.md)は遷移の主体・CAS境界を定めている。**高度機構はこの正本を置換するためではなく、独立状態軸の安全な進行に不足する条件がある場合のみ検証**する。EKIにStatechart一式を内蔵・Spring製品を採用したことにはならない。

### 8.4 委譲インターフェースは「状態機械そのものの共有」より、実行資格の限定を優先

**TEI正本から必要な部分だけLoweringする案**を第一候補にする。意味上のStable Transition/Step ID、Rule/Flow/Artifact Revision、Trigger ID/Clock/Scope、Data Class、業務Ownerと実行Hostの区分を残し、EKI側のTask/Attempt/Lease/Resultとは`operation id`で相関させる。

- `execution-only`：TEI Hostが事前Guard/意味判断・次Step選択を行い、EKIが型付きTaskを実行して結果を返す。Callbackは任意（照会で回復可能）。TEIが停止中なら**後続の意味上の進行は停止**するがEKI中のTaskは継続する。
- `delegated-coordination`：TEI生成の**特定Revisionで固定した有限Step Graph、既に判定済み条件、公開されたEvent種類、承認済Executor**のみをEKIが永続Runとして進める候補。動的な業務Guard・新データ取得・権限判断は、TEIの受理済みExecutor/既存Providerを明示呼び出し、実Meaning判定結果が不明なら保留。**任意Ruleの意味をEKI Kernelに実装しない**。
- `timed/signal-trigger`：TEIがTriggerを定義。EKIが常駐していれば登録されたscheduleやCommitted Signalの到着・Cursorを保持できる契約を将来比較。ただし実DBのcommitとEKI登録は同一Txではない。Outbox/CDC/再照合と重複抑止はProvider能力と共に受入が必要。
- **State source of truth**：TEI FlowのMeaningはTEI、実RunのCoordination OwnerはそのDeploymentで選択した一つ、EKIのAttempt/LeaseはEKI。EKI AcceptedResultやCallbackの到着はTEI業務効果の確定ではない。

**Owner Gate**：状態機械の意味正本、状態管理の永続化・Runner Owner、同一Stepの再試行を許す主体、タイマの登録と失効、Flow definition更新中のRun移行、External Effect Receiptの相関を、全てDeployment契約へ記載する。EKI独自のCommon APIを使うかTEI Pluginの拡張を使うかは前節§2の「EKI Common Authority Contract＋TEI Plugin Binding」で扱い、EKIの外部APIをTEI固有Statechart形式にしない。

### 8.5 独立Expected／Conformance（追加案、NOT RUN）

| Case | Fixture | 期待される区別・禁止推論 |
| --- | --- | --- |
| SD-01 | TEI生成Hostが稼働し、業務Flow A→B、Task AだけEKIに送る | **TEIが次Step判定を維持**。EKIはAの実行Attemptを管理。EKI Task completedのみで業務Bを自動確定しない |
| SD-02 | TEI CLI終了、EKI未導入、cronで単発Job起動 | Cronが起動Owner。TEIのFlow意味は維持するがEKIで動いたふりをしない。常駐依存は`unsupported` |
| SD-03 | TEI停止中、EKI上に受理済み有限A→B Graph＋Executorのpinあり | **有限表現に限り**EKIがRunを進める設計候補。未受理Guard/Provider不可なら保留、勝手に業務確定しない |
| SD-04 | AのEKI ResultClaim成功、Lease失効、Callback重複／順不同、業務Commit unknown | AcceptedResult/ReceiptとTEI Business Completionを分離。ResultClaim=成功は不合格 |
| SD-05 | EKI Task/Attempt/Lease/Eventが同時更新、Crash直後にrecovery | 独立軸のCAS、Attempt再作成、古いLease fence、UNKNOWN維持を確認。単一State enumからの再計算で旧Authorityを復活させない |
| SD-06 | Groupの3タスク：成功、期限切れで未送信、送信済みEffect UNKNOWN | Group集約は部分結果・未知を残す。単純JOIN=successや「3分の2完成＝業務承認」を禁ずる |
| SD-07 | 業務Rule改版とEKI登録済みGraphの走行中更新 | Meaning RevisionとRun Pinを照合。新旧混合実行／旧Owner権限での後続Effectを防ぐ |
| SD-08 | 同じTEI FlowをHost-running方式とdelegated-coordination方式で実行 | State/Transition ID・独立Expected・Effect outcomeと負例を比較。同じUI図表でも不可搬機構を`unsupported`と示す |

**停止・採否**：EKIがBusiness Flowの理解を必要とする／TEI側に隠れた常駐Runnerが必要／同じStepを二主体が再試行する／partialまたはunknownをsuccessへ変換する／Springの便利機能が独立AuthorityとTx境界を破るならその委譲案を採らない。まず現行EKI #57/#112/#242のTask実行縦断を検証し、Group・Timers・Deferred・DAGは必要な最小Caseが失敗する際のみ追加要求へ昇格する。

### 8.6 EKIの継続的な資源・Node運用へのTEI適用（2026-11-03、Owner追加）

[EKI Resource／Node運用のTEI可視化・Simulation・AIレビュー計画](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)を参照。**EKIはJob Controllerだけでなく、CPU/GPU/Memory、Worker Node／Local Governor、Pool／Delegate、Resource Reservation、Resident/Session、権限の失効と回復を継続的に管理する製品**。TEIは、そのEKI自身の**Application Plane**として、運用Model/Graph/表・時間軸・シミュレーション・生成AIレビュー・運用Function/Runbook・安全なAdmin Intentを最大限実装できる候補である。TEIをただの管理画面描画器に矮小化しない。

ここでも依存は逆転させない。EKIのObservation Planeは非Authority、Resource/Lease/Placement/Local Governorの実権限はEKI Kernel／OS／Providerに残す。TEIが生成した可視化・AI候補・Simulationを、WorkerやDeviceの直接制御・Lease発行・Admin権限に格上げしない。EKIのControl PlaneやLocal GovernorはTEI/AIが停止していても端末を保護する。

特に現在のEKI[#354](https://github.com/hat47x/eki-fabric/issues/354)は同時Attemptの**合計予約**、[#118](https://github.com/hat47x/eki-fabric/issues/118)はCPU/GPU/NPU等の観測可能性、[#123](https://github.com/hat47x/eki-fabric/issues/123)は安全なAgent更新、[#124](https://github.com/hat47x/eki-fabric/issues/124)は端末利用者への提供履歴と説明を課題としている。これらをTEI Simulationの独立Expected／危険な操作のPreview・停止条件へ接続するが、未実装のEKI管理API／状態変化をTEI画面が提供済みとはしない。

### 8.7 TEI Applicationの「EKIへLaunch」は実行Hostの新設ではなくDeployment Broker候補（2026-11-03）

[Cloud Control＋On-prem/小型Edge FabricへのTEI Application Launch設計](TEI_EKI_APPLICATION_DEPLOYMENT_FABRIC_2026-11.md)を追補。EKIの既存[Orchestrator調査§7.3](https://github.com/hat47x/eki-fabric/blob/main/research/integration/orchestrator-integration-and-layering.md)は、TEI生成の**常駐業務Application本体をEKIが直接運用することを対象外**としつつ、配置先の選択とEvidence記録を未決の候補としている。今回の「EKIがLaunch Target」を**Logical Deployment/Placement Broker + Management Console**として試作するのはこの制約と両立するが、EKIの正式なPaaS Hosting APIが採用済みという意味ではない。

TEIのApplication Composition/Artifactから`Release/Deployment Intent`を導出し、EKIのAdmin/Provider Selectionを経て、**実際のWeb/API常駐・Ingress・Autoscale・DB/SLOは選ばれたCloud/On-prem Hostの責務**とする。EKIへのSubmit成功、下位ProviderのResource作成、Actual Service Readiness、TEI Business Outcomeを別証拠へ束縛する。EKI自体が生成ApplicationのHost/Reverse Proxy/DBを引き受けない。公開Client HTTPやbulk ArtifactをEKI Controlに余分に通さず、同一Cloudへの直接Deployとの費用・遅延・障害依存の比較を必須にする。

Workerにはoutbound-firstの社内PC/ARM64 Raspberry Pi/On-prem資源Poolも候補だが、**低信頼・端末利用者優先／断続的な資源**にStateful ServiceのAvailabilityを委ねない。小型Linux WorkerのOS/arch/RAM/Sandbox/Executor digestの適合、Agent enrollment/revoke、Node/Data/Trust/Lease Authority、Site Cache/Network費用を検証する。厳しいSLOはCloud/Managed Host、再実行可能な補助TaskはEKIが配分、というHybridの具体Caseから開始する。

### 8.8 TEI成果物のLaunch先とEKI Cloud Control（2026-11-03、市場適合追補）

[EKIの市場適合・Cloud/On-prem配備方式の照合](EKI_ENTERPRISE_COMPUTE_MARKET_FIT_2026-11.md)を追加。TEI生成ApplicationをEKI Consoleから**既存Cloud/オンプレHostへ配備管理**できることは有力な利用体験だが、常時稼働Web/API/DBのAvailability/業務State正本をEKIのTask Leaseへ無条件に移さない（EKI #130／ADR-0013）。EKIのControl/ConsoleはCloudに置ける構成候補でも、Worker ExecutionとData PlaneをControl経由のProxyにしない。Linux/Arm/小型PC/一時Workerは登録・認証・Local Governor/Resource Suitability Gateを通った範囲に限定する。**常駐App配備と止められる背景Jobを異なるProduct Profile**として検証し、Cloud ControllerやEKI-hosted TEI Serviceを実装済みとは扱わない。

### 8.9 AI Jobの複雑さはTEIへ、HPC/Nodeの専門性は下位Providerへ（2026-11-03）

[AI Job構築とPool-first委任の設計](TEI_AI_WORKLOAD_EKI_AGGREGATE_DELEGATION_2026-11.md)に基づき、EKIの最低限の責務を再確認する。TEIはAI/LLM JobのData Context、Model/Prompt/Tool/Operator、Function/State/Flow/DAG、依存と独立Expected、Artifact/Executorを生成する。**EKIは任意AI Operator/DAGの解釈を行わず、Task Envelope/必要資源/信頼・Data Class/期限・予算/試行回復/結果検証を扱う委任Brokerでよい**。ただしAuthority/Admission/Task/Attempt/Lease/Fence/Unknown/Resultは**EKIが省略できないKernel責務**であり、「単純なJob Controller」を無認可Queueと同一視しない。

大規模HPCクラスタのCPU/GPU/Memory粒度やMPI/NUMA・GPU分割/Worker Queueは、Slurm/Kueue/Kubernetes／外部Cloud等、既存の専門Providerに残す。通常のEKI資源選択・開示対象は**登録済Pool/Cluster Group/Delegate**。EKI自身がWorker直接実行を担当するProfileでのみWorker/Lease実証が必要。低信頼のPC資産登録・隔離は既存IT Asset/MDM/EDR等のSecurity Authorityに保持し、EKIは動的な供給Offerを持つResource Delegateとして扱う。外部Compliance≠実Worker Assurance／現在Local Governorの許可と区別。

組織・経済圏を跨ぐ場合も、管理者登録済Delegate、非推移的Federation Grant、予算/データ制約、本人のopt-outを尊重する。EKIのPoolは単一巨大GPU/Memory機械ではないため、Workload条件を満たす下位Sub-offerの検査が必要。**TEI Flow意味、EKI Delegation Authority、HPC Node Scheduler、Endpoint Asset Authorityの四者を別正本として保持**し、重複Controller・安全性保証の格下げ・資源の二重予約を防止する。

### 8.10 委任先PoolとProviderの多段Hand-off／実結果の未解決契約

[TEI/EKI/下位Providerの未解決Gate](TEI_EKI_AGGREGATE_DELEGATION_UNRESOLVED_GATES_2026-11.md)§3・§7を参照。**TEI Business Operation／EKI Task・Attempt・Lease／下位Provider Job・Reservation／Effect Receipt**を四種類の独立IdentityとOutcomeとして追跡する。EKI Admissionは下位GPU予約ではない。ProviderのAck消失は`submission_unknown`、Lease失効はEffect不在ではない。下位Slurm等のrequeueとEKI Retryが同一Effectを二重発生させないため、Owner・execution ref・lookup・idempotency条件をProvider Bindingに固定する。**現在のEKI #112ではInput Artifact 8/9=422、結果取得501のSpike Gap**があり、最初の実用Gateは純粋Job一件のinput/output bytes/digest E2Eである。外部APIの新Schemaや分散Tx実装はGateで不足が確定するまで保留する。

### 8.11 TEIのJob期限・再実行条件をEKIの回復Aware Plannerへ渡す

[予算・リードタイム・回復調停](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)へ詳細を分離。TEI Job/Flowの`not_before/target_complete/latest_useful`、End-to-endの検証・業務完了条件、再実行/Checkpoint/Effect ClassはTEIの意味契約。EKIのPool Brokerは現在の許可済みDLG-1／BUD-1／CAP-1、Queue/Transfer/Provision/Verifyと失敗後の復旧余力を用い、配置候補／早期引上げ／期限リスクWarningを返す。**単独Jobの候補順位だけでなく、複数部門の既存予約・回復Window・公平枠を横断し、Planの予測とProvider実予約／EKI Authority／TEI Business Completionを別の事実として返す**（詳細は追補§10）。**予定通りの完了は保証しない**（EKI RQ-057）。実外部効果がUnknownの場合、期限逼迫は勝手な再試行許可にならない。Provider内の再試行とEKI Attemptの再試行を二重化せず、TEI Business CommitのOwnerも変更しない。

### 8.12 他レーンの実Go CLI／Action／BatchとEKIの証拠を一つの試行で照合

[TEI→EKI→Providerの一改訂・実行証拠の縦断](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)を追加。TEI #1130のJL-01〜07とAJ、#1129のData Script XL、#2616の実Go CLI/Action Offer（限定Go試験と実Host未検証の区別）、#2667のSource/Meaning/Artifact/Consumer境界を、**同じ純粋JobのT0 Direct／T1 EKI Native／T2 Managed Provider**へ接続する。EKIのTask acceptedはResult body取得・Provider実予約・TEI Business Commitとは異なる。まず#112の旧Artifact/Result 422/501 Gapを解消したか同じInput bytes/digestで検証し、新Host/Registry/Schedulerを前提にしない。Job/Business MeaningのOwnerもEKIへ移さない。

### 8.13 最新EKIの実装を根拠に、委任前提を再資格化する

[一改訂実行縦断§12](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)に、現EKI `main`の**optional Dispatch実配線・別ProcessでHTTP→Agent固定Command/Claimを確認するテスト・Result GETにAcceptedResult Storeを接続する実装・条件付きOutput Artifact GET**をSource根拠とともに反映した。旧EKI #242の「起動経路未配線」、#112の「入力422/Result501」は**旧Source/Spikeの時点事実**であり、現在の全構成の実装状況と同一視しない。逆にTest-profileではTaskをADMITTED→READYへ試験用CASし、Result Acceptanceを試験内で直接呼び出しているので、TEIから通常のProduct起動だけで任意Jobが自動で業務Resultまで達したと主張しない。

最初のGateは**W-1 Go Direct vs 現EKI RE-01〜08**。現在Input Artifact ingest-intents/uploadはHandlerが501、Output GETはStoreがあると実経路がある。ExecutorApprovalはDispatch起動設定から取得するため、TEIごとのFunction/Job/Entry DigestとCurrent Grantが実確定したかは別。完了した限定Processテストの存在を足掛かりに、既存EKI #112/#57/#242へ不足分だけを引き渡す。新しいTEI HostやEKI第二Job Engineは作らない。

### 8.14 TEIの限定RunPlanとEKIの実Task Groupは別レベルの契約

[最新の一改訂縦断§13](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)で、TEI `RunPlan`の現Go実装（Stepごとの進捗・Checkpoint Resume・RunKey/Attempt/Fence）が確認できることを追加した。ただしRunKeyの現在実装にはSource/Artifact Revisionや業務Effect Receiptが含まれず、File Journalの`Last→Append`は分散CAS/Txを証明しない。**Step resolvedはEKI AcceptedResultでも業務Commitでもない**。TEI Plan Resume／EKI Task Attempt Retry／外部HPC・Cloud Providerの内部Retryを、Operation ID/Effect Class/現在Grant/Receipt/ExecutionRefで別に管理し、#115の内部Retryが無効化／観測できないeffectful Stepはholdとする。

EKI #267/#381/#695のTask phase実装と限定試験は確認済み。旧文書の「検証・回復後のTask phase未配線」を現時点の一般事実にしない。EKI #126 Group Budget/Deadline/Partial Resultはまだ**openの要求**であり、TEI FlowのFork/Join、Groupの一体Budget/Cancel、業務品質範囲をEKIが実装済みとは扱わない。100件中80件Verifiedでも、TEIのOwnerが全件必須と受理していれば結果は未完了。RunPlanは標準Go単一Binary／既存Host起動を妨げず、EKIが不要なPure/短時間JobはDirectを維持。

### 8.15 EKI P0外部APIのLossless Lowering Gateを追加

[一改訂実証§14](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)にEKI[現実の外部OpenAPI](https://github.com/hat47x/eki-fabric/blob/main/spec/external/openapi.json)とTEI Draft #2787の**WP-01〜19、19/19 exact-head Node CI PASS**を接続。これは**ABI非同値を検出する証拠**であり、実EKI Job/Worker PASSではない。TEIのTyped Input、Executor digest、Effect class、not-before/target/latest-useful、Data Class/Pool AuthorityはEKI P0 TaskRequest/WorkOrderがそのまま実行強制できるとは限らない。特に`parameters`はWorkerへ渡らず、`effect_mode=none`限定、`version_constraint`はAdmissionで未評価。**必須意味が表現・強制されない場合、Adapterはsilent default/extensionでごまかさず`unsupported`／既存Host・Directへ回す**。Input Artifactの実受け渡しや結果取得は別RE/VS実Process Gateとして保持する。WPの固定OpenAPI blobがEKI更新で変わったら再比較が必要であり、AIが現Authorityを生成しない。

## 9. 設計判断のGate、実装順、Issue追補候補

- **G0／今回の設計比較**：EKI ADR-0013・一方向依存・native-first kernel とTEI no-custom-Host/Plugin-firstを保持し、上記二層案を未採用の候補とする。TEI Draft PRへの資料追加だけで、EKIのRequirements/ADR/Mainは書換えない。
- **G1／既存APIで純粋Taskを通す**：EKI #112のInput Artifact ingest/Result retrieval/Executor digestを実Targetで検査。Task admissionとResult acceptance、TEIの独立Oracleを分離。ここが未達ならCallback APIに着手しない。
- **G2／トランザクション境界**：一Providerの`Business Write+Outbox`／同一Idempotency／Rollbackを実観測。Virtual Edgeは同一Scope/Read-setで証明できる範囲だけ。X-TEAから移植したData Scriptの静的Hookをそのまま外部JobのExactly-once契約にしない。
- **G3／Callbackが必要か判定**：pull queryによる回復の限界（頻度・遅延・API費用）が実測できた場合のみ、EKI Webhook/Queue通知をCapabilityとして比較し、受領Inbox・署名・再送・時系列・旧Attemptを検証する。**CallbackをEKI core内業務Ruleにしない**。
- **G4／Task Group・分散Job**：EKI #126の外部Group Contractと実装、#57の縦断reclaim/verified resultを検証。DAGはTEIのFlowとEKIのInfrastructure Task依存を比較し、共通化利益と二重Scheduler負担を測ってから採否判断する。
- **G5／EKIをTEIで作る範囲**：EKI Console/Policy Editor/Operations WorkflowはTEI Modeler/SHO/Function/Roleの採用候補。Kernelの動的権限・WorkOrder署名・Local GovernorをTEIで置換するなら独立の安全性／循環障害／性能試験とEKI Ownerの正式判断が必要。

**Owner間の成果物**：TEIは`Flow Trigger/Rule→EKI TaskRequest`の候補対応票（Scope／Meaning Version／Expected Effect／必要Context）、EKIは`Task Contract/Version/Capability/Evidence/Unknown`の公開可否票を、それぞれのrepoで持つ。共通Schemaを新しい一箇所の正本として設けず、EKI External APIの所有権とTEI Integration Bindingの所有権を分ける。EKI側の追補先は [#112](https://github.com/hat47x/eki-fabric/issues/112)（外部Contract）、[#136](https://github.com/hat47x/eki-fabric/issues/136)（Lease/Plugin比較）、[#126](https://github.com/hat47x/eki-fabric/issues/126)（Group）、[#57](https://github.com/hat47x/eki-fabric/issues/57)／[#242](https://github.com/hat47x/eki-fabric/issues/242)（MVP実装）、[research/integration/orchestrator…](https://github.com/hat47x/eki-fabric/blob/main/research/integration/orchestrator-integration-and-layering.md)（TEIの位置）。この文書だけで既存Issueを実装済みにしない。

**未決**：EKI External Contractに既存非同期Notificationを設けるか、Client side pullだけか／Task GroupがDAG依存を持つか／Flow Coordinatorの配置／Effectful ExecutorのJob Retry許容条件／Sourceのbefore-after整合／別Providerのstrict-edge／EKIがTEI生成Control Planeを利用できる範囲／EKI停止時のRTO・RPO。いずれもOwnerの設計・運用・権限の判断に戻す。

## 10. 検証・出典の状態

2026-11-03時点で接続GitHubの**EKI README/ADR-0013/MVP計画/Architecture/External OpenAPI/既存Issue**と**TEIデータ・Flow/Batch/Effectの既存Issue**をread-only照合した。EKI本体にはGoによるControl・Task/Lease/State等の開発成果があるが、Task Group／外部Callback／TEI固有のDerived-edge→EKI Job委任の実Product E2E、Workerの本番能力、分散Job Controller全体の完成を確認したものではない。EKI #267/#381はclosedで、Task `admitted→ready→active→verifying`・拒否後の再投入とResult受理の配線・試験を#695で照合済み。ただしEKI #242の旧未配線記録とProduct-only／実TEI生成Job/Provider/業務Receiptの受入は別。READMEの古いM0説明・過去Issueだけで現HEADの全動作を断定しない。

TEIは実動するEKI Adapter／Callback Endpoint／Outbox連携を今回追加していない。GitHubのDesign Draftへの追記のみ。**JD-01〜09は全件NOT RUN**、人による運用・Security Reviewも未実施、E2E実装済み・Owner意味受理・EKI要件採用・main統合の証拠とはしない。
