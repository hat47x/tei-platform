# TEIがAI Workloadを構築し、EKIが組織・経済圏単位のResource Delegateへ委任する設計

- 日付：2026-11-03
- Status：**設計案／既存EKI Baselineへの整合確認／反証Fixture（NOT RUN）**。正式なEKI External API／Provider SPI／TEI DSL／HPC Node Scheduler／組織間経済制度の採用・実装を意味しない。
- 発端：AI Job・PipelineはTEIがデータ・関数・状態・依存・検証と共に構築する。EKIはそれを実行させるための**限定委任の核**に集中し、HPC管理・ノード単位スケジューラ・社内端末資産台帳を自前で再実装しない。EKIの主要委任・管理単位を、管理者が登録した**Cluster／Provider／Pool／Organization Domain（論理資源群）**に寄せる。社内PC等の動的な低信頼資源は既存Endpoint Security/MDM/CMDBによる管理を一次根拠とし、EKIは合意済みの集合を「供給量が急変するResource Delegate」として扱う。
- 出典：[EKI ADR-0013](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0013-product-core-and-faces.md)（委任の核・管理者登録の資源プール）、[RQ-058、RQ-059、RQ-063](https://github.com/hat47x/eki-fabric/blob/main/requirements/PRODUCT_REQUIREMENTS.md)、[EKI #109](https://github.com/hat47x/eki-fabric/issues/109)（資源プールと予実、**closed：採用した要求と本番実装は別**）、[#140](https://github.com/hat47x/eki-fabric/issues/140)（広域資源最適化）、[#125](https://github.com/hat47x/eki-fabric/issues/125)（部門間Fair Share）、[#67](https://github.com/hat47x/eki-fabric/issues/67)（既存管理クラスタ）、[#58](https://github.com/hat47x/eki-fabric/issues/58)（分散しない判断）、[#222](https://github.com/hat47x/eki-fabric/issues/222)（Worker enrollment/security、open）、[Deployment Profiles](https://github.com/hat47x/eki-fabric/blob/main/architecture/DEPLOYMENT_PROFILES.md)、[Authority Boundaries](https://github.com/hat47x/eki-fabric/blob/main/architecture/AUTHORITY_BOUNDARIES.md)、[Control/Data Plane](https://github.com/hat47x/eki-fabric/blob/main/architecture/CONTROL_DATA_PLANE.md)。
- TEI関連：[EKI Resource Ops](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)、[EKI deployment target](TEI_EKI_APPLICATION_DEPLOYMENT_FABRIC_2026-11.md)、[TEI/EKI Job delegation](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)、[EKI market fit](EKI_ENTERPRISE_COMPUTE_MARKET_FIT_2026-11.md)。
- 外部実装比較：[Slurm federation](https://slurm.schedmd.com/federation.html)、[Kueue MultiKueue](https://kueue.sigs.k8s.io/docs/concepts/multikueue/)、[Kueue admission](https://kueue.sigs.k8s.io/docs/concepts/admission/)、[Nomad node pools/federation](https://developer.hashicorp.com/nomad/docs/architecture/cluster/node-pools)、[Open Cluster Management Placement](https://open-cluster-management.io/docs/concepts/content-placement/placement/)。これらは**EKIが採用済みという証拠ではなく、Providerに専門性を残せることの比較材料**。

## 1. 配置する抽象を二段階だけに抑える

**TEIがAI Jobの意味を定義する／EKIが「どの管理主体・Resource Delegateへ、どの条件で委任するか」を決める／下位Providerが具体的Worker・GPU/CPU/Memoryを割り当てる**。

| 領域 | 正本・責任 | EKIに渡すもの／EKIが解釈しないもの |
| --- | --- | --- |
| **AI Workload／Flow authoring** | TEIのData Model＋Data Script/Functions/Operators/Validation/Derivation/State/Flow/Scenario/Independent Expected＋生成Artifact（TEI #2620/#1121/#1130） | 処理を実行可能単位へLoweringしたTask要求、入出力Artifact/契約、期限、業務上の再試行可能性、Side-effect区分。LLM prompt意味・モデル選択・Data Join・Tool呼出依存DAGをEKIの第二正本にしない |
| **委任先の登録と広域配置** | EKIのDelegation/Admission/Suitability/Placement/Assurance/Task State、管理者が登録したPool／Provider/Trust Domain | **可視・選択可能な論理資源単位**はPool/Registered Delegate。EKIはJobID→委任先・予算・期限・Outcomeを追い、個々のNodeのHPCスケジューリングを標準責務にしない |
| **HPC・専用GPUクラスタの内部実行** | Slurm/Kubernetes/Kueue/Ray/既存Cloud Managed JobのResource Provider/Orchestrator（実Job・内部Node状態の正本） | Providerが報告できる可用量・資源Class・上限/Queue・Job/ExecutionRef・Result Claim/Evidence。下位Queue/Quota/Preemption/RetryをEKIが二重発行しない |
| **動的PC群・企業Endpoint** | Endpoint Security／ITAM／MDM等が資産・Compliance・管理対象・失効の正本。Local Gateway/Worker Local Governorが即時拒否・測定・隔離を強制 | EKIには**組織的に承認されたPoolの変動するCapacity Offer**と実契約で発行されるTask Outcome。MDM inventoryをWorker保証や実容量へ自動昇格しない |
| **組織間Federation／「経済圏」** | 各OrganizationのTrust/Policy/Budget/Data Authority。別制度のContribution/Credit/Settlementは各Owner | EKIはFederationでの許された**委任関係と有効枠**をBrokerする。信頼を推移的に継承しない。資源量の記録＝価値・現金・支払義務ではない |
| **管理・可視化・シミュレーション** | TEIで構築したEKI Application Plane。Read-only資料やWhat-ifはSimulation/Observation | Cluster/Providerの利用可能枠・履歴・コスト・不確実性をGraph/表/Timelineで表示。必要な範囲でProviderへの深掘りを**権限に応じて委譲**し、NodeInventory正本をEKI/TEIにコピーしない |

**例外**：EKIが一台で動くDP-7やWorkerを直接登録するProfileでは、EKI内部のWorker/Attempt/Leaseの**現実の証拠とfence**が必要。利用者の通常操作・広域選択単位をPoolとすることと、EKIが直接責任を持つWorkerの安全検査を放棄することは別。**「EKIが個々のNodeを管理しない」は「下位Workerの認証、失効、効果不明を検査しなくてよい」を意味しない**。外部Providerが内部Workerを完全に隠す場合、EKIが証明できるのはProvider境界までであり、個別Worker attestation相当の保証を暗黙に付与しない。

## 2. Resource Delegateの最小共通契約（概念・正式Schemaではない）

**同じIDの個別MachineをEKIに全部登録するのではなく、管理者が承認した`Delegate ID / Pool ID`を基本の見える対象とする。** EKIは少なくとも以下のCategoryを読み書き分ける必要がある。

| Category | 必要項目の例 | 意味・Authority |
| --- | --- | --- |
| Identity/Owner | `delegate_id`, `trust_domain`, `provider_id`, `admin_registration_rev`, issuer, scope | **管理者が登録**。Dataの顧客が見える範囲を制限。動的なWorker自身の`pool=trusted`自己申告不可 |
| Workload Capability | runtime/OS/arch（Go linux-amd64/arm64等）、CPU/GPU class/VRAM floor、memory minimum、artifact transport、allowed data class/region | 自己申告機能と確認済み機能を別。異種Poolの場合は**同種の許可済Sub-offer**に分割し、平均GPU/VRAMで偽の適格性を作らない |
| Capacity Offer | `as_of`, `expires_at`, `capacity_class`, `min_guaranteed`または`0`, `max_burst`, `reservation/ref`, `confidence/evidence`, `availability_window` | **供給量は時間変化する予測/Proposal**。実Reservation/Admissionは外部ProviderとLocal Governorに再確認し、予約を二重計上しない |
| Admission/Execution | Task/Operation ID、idempotency、lease / delegation authority、provider execution ref、limits、deadline、allowed retry/reclaim | EKIが受理してもProvider実容量を確保したとは限らない。Providerが`accepted`→`reserved`→`started`→`finished`のどれを保証できるか分ける |
| Result/Effect | claimed result, verified result, artifact digest, provider receipt, unknown/partial/requeued | **ProviderのCompleted≠EKI AcceptedResult≠TEI Business Commit**。Ack loss/Cancel/再送/二重Schedulerを別に検査 |
| Security/Revocation | 認証されたProvider/Delegate、Data Class、Attestation refs、currentness、credential rotation/revoke、local opt-out | EKIがAdmin向けTokenをWorkerに渡さない。Endpoint管理が`non-compliant`へ変わったらOffer/Grantは保留・失効する方向、**届かない場合はfail-closed** |
| Provenance/Cost | region/site granularity（開示可範囲）、estimated walltime/transfer/egress/verification/cost/energy、実観測Revision | Cost見積と実計測を別表示。ユーザーへ下位Worker ID・所在・Graphノード数を漏らさない（EKI RQ-018） |

**Pool＝1台の仮想Supercomputerではない**。Pool内の総CPU/RAMやGPU/VRAMを単純に合計して、任意の単一Jobが利用可能だと判断しない。単一Task要件（例：1ホストの128GB RAM／1枚の80GB VRAM）を満たすPhysical Placementは下位Providerで検証。EKIの粗いOfferは`could-admit`まで、実`admitted`はProviderの現在時点のReservation/Binding・ResourceClass・Trustを要する。跨いだ処理をまとめるにはTEI Workloadの分割可能性/通信/Checkpoint等の明示Contractが必要。

## 3. AI Jobの複雑さはTEIが所有し、EKI APIは「Taskの実行条件」に絞る

```text
TEI Authoring（AI/データ/意味）
  Data → Data Script/Validation/Derivation → AI Function/Model/Tool/Batch
       → Flow/DAG/Test/Oracle → selected Workload Artifact/Execution Contract
                                  |
                                  v
EKI（組織・経済圏横断の委任／資源調停）
  Task Contract → Suitability (local/existing vs delegated)
                → Admin-approved Delegate/Pool Choice
                → Time/Cost/Trust/Data/Quota Boundaries
                → Submit/Query/Cancel/Receipt Correlation
                      |                 |
                      v                 v
    Managed HPC/GPU/Cloud Pool      Dynamic Corporate-PC Pool
    Slurm/Kueue/Cloud Provider      Managed Endpoint/Local Gateway
      → native Node scheduling       → PC membership/Local Governor
      → ExecutionRef/Result          → permitted ephemeral execution
                      \                 /
                       v               v
                   EKI Result Verification
                       |
                       v
              TEI Flow/Business Result (separate)
```

**TEIのAI Job構築**には、学習データ準備、推論/学習Operator、モデル・Prompt・Tool chain、評価・検証、Function/Actionの業務意味、並列化/分割/Join、再実行の業務正当性、Target別Artifact、独立テストが含まれる。TEIがすでにこれらを全て製品として実装したという主張ではない。既存Plugin/外部AI/HPC Frameworkの結合を優先し、TEI CoreへLLM固有エンジンを固定しない。

**EKIに渡す抽象の最小化**：`Workload/ExecutorDigest, Input/OutputArtifact references, Resource/Runtime/Trust/Data/Deadline/Budget, retry/checkpoint/effect class, idempotency/causation, result assurance`。Workload内容が分からなくても契約を検証できることが望ましい。EKIは内部のBranchやAI Tool選択を解釈しない。TEI側にCoordinatorがあればStep単位でTaskをSubmitし、非常駐で限定的なStep依存を委譲したい場合だけEKIの受理済み限定Graph契約を検討（[委譲境界§8](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)）。**EKIが「単純なJob Controller」で済むのは意味解釈を薄くできるという意味であり、Authority/Lease/Safety/Outcomeを省略できるという意味ではない**。

## 4. 低信頼・高変動Endpointを「変動供給のCloud Pool」として使う

**外部Endpoint Security / Asset systemを真のOwnerとし、EKIにその補助データを投影**：

1. **Discovery**：既存Endpoint Security/IT Asset/MDMの管理対象Node、OS/arch、Owner/Org、機密Policy、Device Compliance/Quarantineなどを、許可された`Asset/Compliance Reference`としてProvider Adapterから取得。Inventoryの現Revision/Time/Sourceと取得完全性を明示。
2. **Pool Registration**：Security/Resource Ownerが、特定の動的Group/Policy/Device Classを`delegate_id`へ承認。EKIが「合格した全PC」を勝手に追加しない。Namespace／policy origin／Tenant/Org Scopeを固定。
3. **On-device Enforcement**：実資源の観測・端末利用者の優先・即時停止はLocal Governor/Managed Agent/必要ならSite Gateway。EKIがCMDBの`compliant=true`だけを根拠に実利用上限を増やさない。
4. **Capacity Supply**：時点付きClass別Offer（`0～N`台、予測分布、実予約可能下限、Burst/Time Window、Expiry）をEKIへ。重要なTaskは`guaranteed=0`のPoolに置かない。段階的な配備・Site Gatewayでデバイスの細部を集約可能。
5. **Execution/Result**：EKIがPoolに依頼したらGatewayが端末/Workerを選択し、Owner/Local Governor/Secret/Data Class/署名済WorkOrderの実行時資格をチェック。中断・失踪・期限切れは通常の結果。外部へ書込むJobは失敗時の二重効果防止をさらに要求。
6. **Revocation/Offline**：Endpoint管理側で隔離／契約終了／User opt-out→Registration／Grant失効が非同期に伝わる。同期不能・Asset sync stale・Provider未知なら**新しい機密処理のPlacementを止める**。Already-running workerのKill/StopはOn-device Enforcementが担当、EKIへResult/Effect unknownを返す。すべてのActive Lease停止をEndpoint管理の一行更新のみで保証しない。
7. **Oversight**：TEIのAdmin Portalでは組織／Pool/Class単位の有効Offer・故障率・Reason/Forecast/CostとImpact／Simulationを表示。一般利用者にはPCのIdentity/hostnames/数を不用意に開示しない。権限を持つAsset Operatorは**外部ITAMへ深掘り**でき、EKIに冗長なEndpoint Directoryを複製しない。

MDM等は**資産管理/ComplianceのAuthority**、Pool Gatewayは**接続の実行・容量実測とLocal enforcement**、EKIは**委任/Task/Group Result Authority**を担当する。外部管理Productの単なる`healthy`表示は、Workerの実挙動／Resultの真偽・現在のLocal Governor許可を証明しない。登録時と利用時を分けた強制境界が必要。

## 5. HPC/Cloud/組織をまたぐ委任単位と二重Schedulerを避ける条件

- **HPC**：Slurm/KueueなどのJob/Node/Queue管理をそのまま使う。EKIは「登録済HPC Resource Delegateへ依頼する」一段上のBroker。下位の`submitted`/`pending`/`running`/`completed`をEKI Task/Attemptと同一視せず、QueueRef/ProviderRef・Owner・Capacity/Quota/Reservationで相関。内部GPU分割、NUMA、MPI rank/通信の専門性は下位Scheduler／TEI Workload targetが担当。
- **Cloud**：Region/Account/ResourcePoolをDelegateとして登録。Cloud Autoscalerは下位Controller、EKIは予算・期限・用途・Trustを照合。複数Orchestratorのautoscale/Preemption/Retryを二重に発火させない。TEI常時稼働ServiceのReleaseは既存HostがAvailability Owner（EKI #130）。
- **Organization**：一つの管理者Scope＝一つの硬いTenantとは限らない。複数Department/Provider/Poolへ委任するが、Team/Org budget/Consentの優先順位を受理済みPolicyで決める（EKI #125）。**経済圏**は一つの論理Federation集合／Exchange of compute grantsを指す候補であり、強制的な資源共有・統一価格・通貨発行を意味しない（EKI RQ-031/034/037/038）。
- **Trustの非推移**：Org A→Bの許可とB→Cの許可があっても、Aの機密JobをCへ勝手に流してはならない。CapacityやResultはProviderの証拠の強度に従って表示。下位Member IDの公開は参照Scopeがある場合だけ。
- **Cost / Data Locality**：EKI自身がHPC/Cloud Schedulerを包む追加コストを補う価値がなければDirect Executionを選ぶ（EKI #58）。巨大InputをEKI中央Control経由で転送せず、既存Data Plane/Site Cache/ArtifactGrantを使う。GPU Poolで通信の高い分散MLが不利なら既存HPC局所のまま実行する。

外部Schedulerの実装例は、[Slurm federation](https://slurm.schedmd.com/federation.html)（複数クラスタ、originが重複起動を協調）、[Kueue MultiKueue](https://kueue.sigs.k8s.io/docs/concepts/multikueue/)（ManagerとWorker Cluster、下位Quota/Admission）など。これらの性質を利用可能なProvider Adapterに結ぶ一方、**二重Quota/重複Job/下位PreemptionのUnknown**をEKIの最低反証とする。EKIのタスク提出＝下位SchedulerのReservation、下位Job completed＝業務意味受理という短絡は不合格。

## 6. 修正が必要なTEI/EKI計画の焦点

- 以前のResource Ops計画はEKIそのものがNode/Device/GPU partitionを見られる構成を広く候補にしていたが、**標準UI/ControlはPool/Delegate firstに再配向**する。個別NodeのGraph/CPU/VRAM内訳は、EKIが直接管理するDP-7などのProfile、または権限を持つ外部IT Asset/HPC管理者がProviderに深掘りする専門UI。**中央が全Workerを何も知らずに並列Dispatchしてよいことにはしない**。
- 以前のCloud deployment計画L2/L3/L4でNodeの細部までEKI Controlが所有する前提を置かず、**Managed Cloud Pool/Managed HPC Pool/Dynamic Endpoint Pool**へのDeploy/Task Bindingを明示。安定Web/App HostはCloud/既存Orchestratorへ、可中断BatchはEKI Workerへ。
- 初期Market Fitは**S1 GPU混在Poolの管理者向けのread-only横断可視化**で検証。Endpoint群は既存Security/ITAM integration経由、HPCは既存Scheduler integration経由でのみ広げる。EKI独自HPC Provisioner／Endpoint CMDB／PaaS Runtimeを最初から作らない。
- **経済圏単位の可視化**では、金銭/価値/業務意味をEKI Coreが生成しない。Policy Authority、Data Authority、Account ownerが異なる環境を混同しない。TEIはGraph/Table/Timeline/Simulation/AI reviewerで大きな関係を可視化し得るが、非開示Poolがあるときに全体の総量/Node件数を勝手に漏らさない。
- **高い可観測性を維持しつつ省コスト**：Resource Providerから集約した状態、Task/Lease/Job Result、Failure EvidenceとImpactを、EKIの非Authority Observation Planeへ提供。個別Nodeのmetricsを全てEKIへ二重コピーしない。診断にはProvenance/Time/CompletenessとProvider deep-link/限定Queryを付け、使用者へ「なぜ今使えないか」を説明できる範囲のみ提供。

## 7. 最小検証ケース（AP-01〜12、未実行／独立Expected）

| Case | 対象Input・意図 | Expected（禁止推論） |
| --- | --- | --- |
| **AP-01 AI JobのOwner** | TEIで2段AI Job（前処理→評価）をモデル化。EKIに`stage A`を委任 | EKIはTask EnvelopeとResource条件だけ理解。Step Bが進むのはTEI側の受理済みFlow/選択Coordinatorによる。EKI task accepted/worker successを業務条件と同一化しない |
| **AP-02 HPC pool first** | 同一ClusterにWorker 100台、GPU種類混在。EKIへは委任先`HPC-A`を登録 | クライアントはWorker IDを列挙できず、Executor/Resource class不一致Jobを拒否。HPC Provider側のPlacement/Queue/Nodeの詳細はOwnerが保有 |
| **AP-03 Node合計の偽容量** | HPC内CPU合計256core、個別Node max64。単一Taskが128core shared-memoryを要求 | Nodeの合算でAdmission成功と判定しない。MPI等を使うならTEIが明示分割とProviderの実保証を要求 |
| **AP-04 GPU異種能力** | GPU VRAM24GB×2とVRAM80GB×1、Job要件VRAM64GB×1 | 集約`128GB VRAM`だけで、どのGPUも適切と偽装しない。Resource Class/Sub-offerで実適格性を判定 |
| **AP-05 Dynamic PC pool** | 社内Managed PC 20→2→0→10台と変動。実Owner使用が再開 | `0 guaranteed`、失効Time付Offer。EKIは有償SLOを偽装しない。Local Governor 0がCentral推奨を上書き |
| **AP-06 External quarantine** | MDMが端末`noncompliant`、Capacity Offerは5分古い | 新しい機密Jobを止める。MDM `compliant`または古いOfferを実Placement資格にしない。実行中TaskはLocal enforcementのReceipt／Unknown |
| **AP-07 多段失効** | AからBへ、BからCへCompute Grant。A→Cの許可はなし | Cへ委譲しない。Aggregate Count/Capacityの表示から隠れたCのNode数/Locationを漏らさない |
| **AP-08 Provider vs EKI success** | Slurm Jobは`COMPLETED`、Result Artifact未検証、TEI側Business Commit不明 | 三者の`completed`／`verified`／`committed`を混同せず、業務Flowの成功を宣言しない |
| **AP-09 HPC二重Retry** | EKI上でretry、Slurm側もrequeue、ACK消失 | 同一Operation/ExecutionRefに対する二重Effectsを検知し、unknownならreconcileで止める。下位requeueを新Jobへ黙ってコピーしない |
| **AP-10 Asset leak** | 一般利用者がPool見込みを要求、組織全Node一覧は非開示 | Pool提供可能な見込み/利用者に許されたClassのみ開示。Node IP/hostname/件数/AI prompt/Tooltipから秘密が漏れない |
| **AP-11 Provider unavailable** | 一つのHPC Providerは停止、他のPoolは正常 | 不明のHPC capacityを0/healthyへ変換せず、無関係Poolの正当な処理は継続。HPCの代替がData Class/Cost/Target条件を満たさなければ止める |
| **AP-12 Cost/latency baseline** | 100msのTEI処理を直接 vs EKI→Managed Poolへ投げる、同等のInput/Result | End-to-end latency/transfer/verification/cost含め利点無しならDirectを選ぶ。単なる高い稼働率を成果にしない |

**Owner境界**：TEI #1121/#1130/#2620/#2499、EKI #109/RQ-058、#140、#125、#67、#222、#58/#106、#57/#112、#130、Authority/Deployment Profiles。AP Fixtureは**今後の独立Oracle候補**であり、実Slurm/Intune/MDM/Agent/Cloud/EKI/TEI間のE2Eを確認したものではない。

### 7.1 反証の優先順位：Pool-firstを実行可能にする依存Gate

[資源群委任の未解決Gate・実予約・MDM/Trust・Federation反証](TEI_EKI_AGGREGATE_DELEGATION_UNRESOLVED_GATES_2026-11.md)を追加した。これまでのAP-01〜12は抽象境界を試すが、**TEI Job byte/Result E2E（EKI #112）、Offer→Provider Reservation（DLG-1/BUD-1/CAP-1）、Provider二層Retry/Unknown、MDM情報とLocal Governorの分離、Trust Bundle読取障害（EKI #466/#467）、時点付き組織Policyと予算、AI運用への非開示**を単独のG0〜G7・DG-01〜14として掘り下げた。**P1 Pure Job Wireを実証する前に管理Pool構想を実装済みと称しない**。既存EKI #465はclosedであり過去のソース所見を現HEADの未修正不具合と断定しない。

### 7.2 リードタイムと回復可能性を条件に、資源群を動的に選ぶ

[期限・予算・実績・連鎖失敗を考慮するEKI調停設計](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)を追補。**TEIがJobの完了条件・許容リードタイム・再実行の業務条件を定義**し、EKIはDLG-1で認められたPoolを対象に、BUD-1/CAP-1の枠、Queue/Transfer/Verify/Recoveryの残余時間、過去のPool予実を使い、Direct／余力／管理Cluster／Cloudへの配置・待機・期限前の引上げを**Proposal**として比較する。外部HPCが内部Workerを選び、EKIはClusterを1論理Delegateとして見る。履歴による期限内成功見積はTrust/AssuranceのAuthorityではなく、予測母集団・打切り・故障相関・期限切れを明示。安全なFallbackが予約できなければ達成保証を主張しない。

## 8. 段階と採否判断

- **P0（今回）**：既存EKI「委任の核」「登録済Pool」「一時資源・クラウドの広域最適化」を再利用し、TEI AI Job/Delegate Pool/HPC Provider/企業Endpoint Asset Systemの所有境界・負例を設計書へ反映。新Core Enum/Global Registryの追加は保留。
- **P1（合成Read-only）**：TEIのAI Jobの段階・依存と、二種類のEKI Delegate（Managed HPC/GPU Pool と Dynamic Endpoint Pool）の`capability/time/capacity/trust`を、同じ型付きRef／シミュレーションで比較。Endpoint個体IDを漏らさず、独立Expected AP-01/02/05/10を検査。
- **P2（専門Provider 1件）**：登録済SlurmまたはKueue Providerから、Admission/ExecutionRef/Resultを読み、EKIのTask IDと相関。Retry/Claim/Verification/CostはEKIの既存Task vertical sliceと比較する。HPC内Nodeの実Scheduler/Queueは自前実装しない。
- **P3（社内資産連携）**：一つの既存MDM/ITAM/Security ProviderとGroup/Device Complianceの**read-only** Scope/Revision/Change Notificationを接続。Local Governorの拒否・新Job停止・失効連絡を合成で負例にし、実端末Client/AgentのNode-by-node検収はEKI Owner側へ渡す。実認証はEKI #222の採用後。
- **P4（Multi-org/経済圏）**：Ownerが登録したTrust Domain間の非推移的Grant／予算・期限・リージョン・Data Classを、実Federation/Providerで検収。組織間請求・通貨・業務意味をEKI内部で勝手に生成しない。
- **P5（事業価値）**：顧客のAI GPU Pool・Cloud・Endpointの競合ツールと比較し、実Jobの同等品質/SLO、管理時間、Data transfer/verification cost、資源確保時間、利用者妨害/安全Incident、支払意思を測定。市場データをEKI自体の商業的成功へ読み替えない。

**停止条件**：Pool identityが曖昧／Provider管理者の登録証拠なし／該当GPU・OS archが不明／容量が単純合計でしかない／MDMの古いcompliance／Local Governor拒否が伝わらない／Grantの非推移性を証明できない／Provider RefとTaskの相関なし／内部Node公開範囲不明／未確定の実Effect／二重Retry／EKIを挟むことで直接より不利か不明。UIはread-onlyの限定説明を継続しても、Actual Placement/Permission/Executionは明示拒否または保留。

**証拠上限**：EKIにPool/RQ-058、広域最適化、Network/Control/Authorityの採用済み方針・部分Referenceはある。**EKI Delegateを抽象的な1リソースとして扱うExternal API適合、Slurm/MDM/Cloud/Low-trust Gatewayの正式なAdapter、TEI AI Jobの全演算・Simulation、跨経済圏Federation、実Lease・Resultの一連E2Eはこの稿ではNOT RUN**。P0は設計整合を表すにとどめる。
