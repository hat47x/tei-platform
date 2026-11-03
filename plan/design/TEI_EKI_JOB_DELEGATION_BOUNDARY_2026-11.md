# TEIのData／FlowとEKIジョブ委任・分散資源制御の接続契約（設計・反証案）

- 日付：2026-11-03
- 状態：**設計提案（Decision待ち／実装・実行検証なし）**。EKIの公開API変更、TEI Plugin SPI変更、ジョブエンジン実装、相互依存の採用済みADRを意味しない。
- 目的：TEIのData Model／Data Script（項目Validation・仮想導出・更新時Rule）→Flow／Capabilityから、EKIを**ジョブ管理が必要な構成の第一候補となる実行・資源調停・結果検証Provider**として委任できるかを検討する。EKI自身のControl/Application PlaneはTEIのローコード能力を活用し得るが、EKI Kernelの権限正本と停止・復旧をTEIへ移譲しない。
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

**実行形態は選択的**：同期／単発のローカルTEI Go単一バイナリや既存Hostと、常駐JobのEKI管理を区別する。常駐Job管理を必要とする場合の基本BindingはEKIであり、他のJob Providerは明示的に選ぶ代替Targetであって、TEI本体の隠れたSchedulerではない。低遅延のUIや同一DBの単純更新まで常にEKIへ委譲しない。EKIのResource Suitability Gateが「分散しない」と判定する場合に、勝手にセキュリティ条件を落とさず別の許された実行方法へ戻す。

**既存Orchestratorとの関係**：EKIはKubernetes+Kueue、Slurm、Nomad等をResource Providerとして繋ぐ上位委任Brokerが基軸。Business Step依存を長期間耐久化する高度なJob/Workflowが必要なら、**TEI側へ常駐Flow runnerを既定で新設するのではなく**、EKI Job Controllerの将来のDAG/依存Run Contract（またはEKIが接続する既存Workflow Provider）で実現できるかを評価する。TEIは業務Flowの意味を保持し、実行可能なTask依存へLoweringする。現行EKI Task Groupが依存DAGの連鎖を実装済みとは扱わず、対応不可なら`unsupported`とする。二層Scheduler間の自動再試行・preemption・予算の二重計上はEKI既存調査の負例を再利用する。

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

## 8. 設定レスのEKIジョブ管理と常駐主体の所在（2026-11-03、Owner要求による修正）

**設計の優先順位を更新する**：TEI生成物には必ずしも常駐プロセスがない。ゆえに、TEIのData Script／Flowに「毎日動く」「変更時に起動」「前のTaskが終わったら次へ」という記述があっても、それだけでは予定時刻を監視・再起動・再送する主体は現れない。**ジョブ管理を利用する構成では、EKIをその常駐・永続実行主体として第一候補に据え、TEI本体へJob Scheduler／Run Registry／Worker監視を埋め込まない。**

ただし、**委譲が完全であること＝EKIがTEIの業務意味を所有することではない**。Data/Rule/Flowの意味・Data Providerの実Tx／Effect正本はTEI生成Application側、スケジュール登録・起動判定の実務管理・Job Run/Attempt/Retry/Deadline/Group/DAG実行・資源配置・回復・ジョブ結果の記録はEKI、という契約を候補にする。TEIがState Machine/Flow図から、EKIが実行可能な**実行依存グラフ／JobSpec／TriggerSpec（形式は未決）**を生成する余地はある。Business MeaningそのものをEKIのTask Stateへ投影しない。

### 8.1 「設定レス」の意味を4段階に分ける

| 場面 | 望ましい最小挙動 | 入力／準備の境界 | 既存の保証 |
| --- | --- | --- | --- |
| **Z0：初回起動・Job未登録** | EKIが一台で起動し、状態Store/契約検査/Health・限定的なStatus APIを提供。ジョブは**勝手に走らない** | InstallerによるOSアカウント・秘密鍵・Store権限・実行体隔離の構成は必要。利用者が設定ファイルを毎回書かない、という意味のZero-config | [EKI ADR-0018](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0018-single-machine-delegation-profile.md)でDP-7（単一マシン分離）が**採用**。実装済みではなくS1予定。現在のDP-0を本番安全性として使わない |
| **Z1：承認済み単発Job** | EKIの標準Task APIで受理・永続化し、ローカルで実行・結果を照会する**デフォルトProfile候補**。外部Worker/クラスタは不要 | TEI/他ClientがJob内容とExecutor digest等を指定。本人のAuthority、ExecutorのAdmission、Resource上限は省略不可。未承認の任意コマンド自動実行なし | 方向はEKI #57・#112・#242。現Productの一連の実行完了は**未実証** |
| **Z2：定期実行・依存Job** | 登録されたTriggerSpec／JobSpecの次回起動時刻・重複・永続Run・Job間依存をEKIが管理し、**TEI Clientが終了しても継続**する設計候補 | 頻度・Event条件・Time zone・Missed-run/Concurrency policyはJob定義に必要。ユーザーにEKI固有の運用設定を強制しないことが目標 | **新たに比較するJob Controller要件**。EKIの現行Task API/Task Groupだけで実現済みとはしない |
| **Z3：外部Data変更・複数Worker・大規模DAG** | EKIが登録されたSignal source／Cursor、Run、Group、資源配賦、Backpressure、回復を管理する構想 | DB CDC/Outbox等のProducer接続、Provider権限、外部WorkerのEnrollmentとGrant、データ所在/機密・予算が必要。存在しないSourceは自動検出できない | EKI #126（Group）、Resource Provider Adapter研究等の**将来設計／検証待ち** |

「完全設定レス」を**すべての外部DB・業務資格情報・Worker・実行プログラムを自動発見して任意の副作用を実行する**意味にしてはならない。望ましいUXは**安全な初期値＋必要条件に応じた宣言的Job登録＋未設定箇所の型付き診断**である。初回起動・ローカル操作に複雑なYAMLや専用クラスタを強要しない一方、認可・秘密鍵・Worker隔離はInstaller/OSで確実に実現する。EKIがローカルClientと同一OSユーザーでControlの鍵／Storeを共有する設定はDP-7の代替にならない。

### 8.2 Cron／systemdとの関係と二つのBootstrapping方式

OSの定期起動機構は実在するが、**TEI生成プログラムがcronやsystemdを暗黙に所有しているわけではない**。

| 方式 | 起動・予定時刻の管理 | 長時間／依存ジョブ・回復とポータビリティ | 採用判断 |
| --- | --- | --- | --- |
| OS Cronや`systemd.timer`を**Jobごと**に使う | OSがTEIの単発CLI、またはEKI Submit CLIを起動。Linux固有の定期実行環境に依存 | 依存DAG、永続Operation ID、実Task Resultの検証、Cross-Worker配置・Recoveryは別。Timer再起動・同時実行・欠落時の意味がOSの契約で異なる | **軽量Fallback/外部Integration Target**として残す。EKI完全委譲の標準経路にはしない |
| `systemd`／Windows Service Manager／launchd等で**EKIサービス自体**を起動 | OSはEKIの起動・監視、EKIはJob Schedule、Trigger、Run/Attemptを永続管理 | TEIが稼働・常駐しなくても継続。EKI Native Kernelが責任を持ち、下位OS service managerはBootstrapだけ。インストール権限とCredential構成は必要 | **推奨する役割分担の設計候補**。EKI Timer・DAG Job Catalogは現実装済みとしない |
| EKIの単発CLIだけを利用 | 利用者または既存Hostから必要時にEKIへSubmit | 単発Jobの委任は可能。**定期実行を誰も起動しない場合は動かない** | Z1の最小出口。Z2やZ3を自動達成と見なさない |

`systemd.timer`の`Persistent=true`は停止期間に発火機会があったことを検出して起動時にServiceを**一回**起動するなどの機能を提供するが、未実行分の全Run列挙・業務上の二重防止・EKIのJob Receiptまでは保証しない（[systemd.timer man](https://www.freedesktop.org/software/systemd/man/latest/systemd.timer.html)）。OS cronに同等の欠落補完があると前提化しない。

**System SchedulerとEKI Schedulerを同じJobの二重正本にしない**。systemdはEKI自体を確実に起動するか、単純Jobのみを外部で起動する。EKIに登録したScheduleと同じ予定をOS timerへ無断複製すると、再起動後の二重起動／Owner不明が生じる。Delegationしない実装TargetではOS scheduler用Unit/cron設定の生成を明示的に選べるが、その場合の正本と運用責務はEKIではなく外部Schedulerである。

### 8.3 Jobを登録した後の所有権とCallbackなしでの継続

**TEIの役割**：Data ModelでVirtual属性を宣言し、Data Scriptで更新時Validation/DerivationとTrigger/Flow Bindingを記述する。ビルド成果物には**Job Definition/Trigger Subscription/Flow Graph/Executor Artifact/Ownerとの紐付け**を含められる。ただし記述やコンパイル＝EKIへの登録／有効化ではない。Deployment/Registrationは現在Grantと契約版を照合し、EKIが受理したJob/Trigger/Run識別子を返す。

**EKIの役割（設計要件候補）**：登録済Job/Triggerの有効期限・Revision・次回時刻・入力SourceのCursor・最後に確定したEvent・Run/Step依存・Attemptとretry・Result/Receiptを**TEI常駐なしに保持・照会**。異なるJobから一つの実Workloadを共有する場合もTask ID／JobRun ID／Business Operation IDを混同しない。EKIがSourceの業務意味を計算するのではなく、TEI生成の承認済みEvaluator／Provider Adapterを許可された入力と実行体として起動する。

**Signal境界**：Data ProviderのcommitがEKIのJob登録・Task受理と原子的に結合するとは限らない。可能ならSource DBのTransaction内Outbox、できなければ権限付きCDC/監視とReconciliationによって更新事実を届ける。**Data Scriptそのものを常駐監視プロセスとして要求しない**。EKIが持つのはSignal subscription／配送・再走査／Job起動記録の管理であって、未登録DBを暗黙に観測する能力ではない。Providerが strict derived edge のbefore/after証拠を出せない場合は`unknown/unsupported`、Ownerが認めたlevel reconciliationのみFallback候補。

**Callback境界**：TEI生成物が存在しない／停止している場合にもEKIはRunを継続し、結果を永続保持してQueryできること。Callbackは**常駐しているTEIアプリケーション等のConsumerが明示登録した場合だけ**送る任意の通知。TEIが常駐しない場合、EKIは保存されたFlow/Job定義に基づき次StepのJobを進め、必要なBusiness Actionは承認済みExecutor/Providerへ新たに依頼する。EKIのRun完了だけで業務DBの状態を成功に変更しない。通知のACKとJob Result、Business Effect Receiptを分離し、Retryする主体はTask/Attempt=EKI、業務Effect=Provider/Target Hostと固定する。

**Job Definitionの版管理**：TEIのFlow Meaning Revision、Target Artifact Revision、EKI Job Registration Revision、Runの開始時Pin、Provider Rule/Policy/Executorの現在資格を独立に記録する。将来のJob定義更新が走行中Runの意味を勝手に変更しない。登録削除と起動停止、旧Run取消、途中のVersion Migrationは別Operationであり、現在のEKIのTask Group APIでこれらが実装済みとは言わない。

### 8.4 初期の重要な非目標と追加Conformance（NOT RUN）

- **初期の非目標**：EKIが任意の業務状態・他ProviderのTransactionを所有すること、EKIのKernelをTEI成果物の稼働に依存させること、設定不要で外部機密DBへ接続すること、全Scheduler/Workflow製品の置換、Cronの全保証を再実装すること。
- **ZC-01** 初回インストール後、外部クラスタ/管理者手編集なしでDP-7安全条件を満たすローカル構成の起動・Healthが可能か。未登録Executorや外部Network Jobは拒否。OSアカウント／鍵の分離を独立検査。
- **ZC-02** TEI生成の承認済みJobSpecをEKIへ一度登録し、TEI CLI／Studioを終了。EKIだけで単発Jobを実行・保存・Query。Restart後も同じJobRunに収束し、二重Attempt/Receiptを作らない。
- **ZC-03** 一つの`daily at 02:00`と一つの`Data-commit`契機について、timezone/DST、service停止中の遅延・欠落、起動重複、Outbox Relay重複を独立Expectedで区別。未登録Source、部分取得、Commit Unknownでは起動成功を装わない。
- **ZC-04** TEI Runtimeが一切稼働しない状態で、EKIの受理済みDAG A→Bが進むか。業務Side Effectを伴うBはProvider認可・Commit/Receiptを別に検収。EKIにDAG機能が無い間は`unsupported`であり実行を捏造しない。
- **ZC-05** EK​​I Kernelが停止したとき、TEIで作られた運用UIが無くてもNative CLI/OS service managerで原因追跡と安全な復旧が可能。古いJob/Lease/Trust rootが復活しない。
- **ZC-06** OS cron/systemd Job-firstとEKI Job-firstを比較し、同じJobを双方が発火させた場合の重複、停止時の追いつき、可搬性、運用負荷とTime-to-first-taskを測る。未起動Runと`unknown`を混同しない。

**EKIへ求める追加契約のOwned Gap**：[EKI #57](https://github.com/hat47x/eki-fabric/issues/57) MVPを先に成立させ、[ADR-0018](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0018-single-machine-delegation-profile.md)／DP-7のSingle Machine導入へ、`local-default Task Admission`／`safe installer bootstrap`の要件候補を渡す。JobSpec／TriggerSpec／Schedule persistence／DAG next-step／Callback不使用のdurable result は[EKI #112](https://github.com/hat47x/eki-fabric/issues/112)のExternal Contract、[EKI #126](https://github.com/hat47x/eki-fabric/issues/126)のGroup、[EKI #136](https://github.com/hat47x/eki-fabric/issues/136)のTEI Plugin比較を**既存Owner候補**として比較。EKIの採用済ADR/Requirementsへの変更はEKI Ownerの判断を要する。TEI #1130は`EKIを持たないTEI生成物が勝手に常駐・監視する`という要件を置かず、`Job実行基盤へのBindingと独立したTarget`を受入単位にする。

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

2026-11-03時点で接続GitHubの**EKI README/ADR-0013/MVP計画/Architecture/External OpenAPI/既存Issue**と**TEIデータ・Flow/Batch/Effectの既存Issue**をread-only照合した。EKI本体にはGoによるControl・Task/Lease/State等の開発成果があるが、Task Group／外部Callback／TEI固有のDerived-edge→EKI Job委任の実Product E2E、Workerの本番能力、分散Job Controller全体の完成を確認したものではない。EKI #381によればTask `admitted→ready→active` の実装が進む一方、Result検証・回復の後続状態経路に未解決箇所がある。EKI #242も起動配線の不足を扱う。READMEの古いM0説明・過去Issueだけで現HEADの全動作を断定しない。

TEIは実動するEKI Adapter／Callback Endpoint／Outbox連携を今回追加していない。GitHubのDesign Draftへの追記のみ。**JD-01〜09は全件NOT RUN**、人による運用・Security Reviewも未実施、E2E実装済み・Owner意味受理・EKI要件採用・main統合の証拠とはしない。
