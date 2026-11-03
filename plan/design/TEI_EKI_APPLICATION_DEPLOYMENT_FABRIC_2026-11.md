# TEI成果物のEKIローンチTarget：Cloud Control・Managed Service・On-prem/小型Nodeを統合する配置契約（設計案）

- 日付：2026-11-03
- 状態：**統合案／設計レビュー用。正式なEKI Application Hosting契約・API・Deployment Provider・公開Service/PaaS・実ネットワーク・Owner受理・性能実測は未成立**。
- 要求：EKI管理ノード・コンソールを安定したクラウドなどで提供し、TEIで生成したApplicationの**論理的ローンチ先**にできるようにする。処理本体はクラウド／既存管理クラスタへ委譲し、EKIを介する余分なリソース・通信コストを抑える。同じFabricには、安全なネットワーク経由で社内PC、GPUワークステーション、Raspberry Pi等の小型Linux計算機、オンプレの一時資源Poolを接続できるようにする。
- 関連設計：[TEI/EKI状態・ジョブ委譲](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)、[EKI資源／Nodeの可視化・Simulation・AIレビュー](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)、[Data ScriptとVirtual Attribute起点のFlow](DATA_DERIVED_TRANSITION_TRIGGER_PIPELINE_2026-11.md)。
- 適用の前提：TEIは既存Hostへ組み込める成果物とGo単一バイナリを生成する（**TEI独自Hostは不要**）。EKIは管理・限定委任の独立製品であり、TEIの意味正本、業務DB/独立Provider、外部クラウドのサービス運用Stateを奪わない。

## 1. 出典から確定できることと、今回はじめて提案すること

| 根拠・現状 | この資料での扱い |
| --- | --- |
| EKI [`research/integration/orchestrator-integration-and-layering.md` §7.1–7.3](https://github.com/hat47x/eki-fabric/blob/main/research/integration/orchestrator-integration-and-layering.md)：TEI→EKIのExecution Binding、TEI製Application本体のEKIによる常時実行は対象外。「EKIが配置先を選び証拠化」する役割は**未決** | **ローンチの意味をDeploy/Placement Brokerとして限定する**。EKIが常駐業務Appのプロセス管理者／SLA保証者になる、と決めない。既存の明示的な非目標を黙って書き換えない |
| EKI [Product Value](https://github.com/hat47x/eki-fabric/blob/main/product/value.md)・[Issue #130](https://github.com/hat47x/eki-fabric/issues/130)：可用性・業務状態正本を持つ常駐Applicationは既存Orchestrator。中断できる状態非正本のresident処理だけ限定Lease連鎖候補 | **Service/Resident/Batch**の実行契約を区別。低遅延Web/APIを遊休PCやLease再取得のみでHAと呼ばない |
| EKI [ADR-0013](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0013-product-core-and-faces.md)・[Deployment Profiles](https://github.com/hat47x/eki-fabric/blob/main/architecture/DEPLOYMENT_PROFILES.md)：外部Resource Provider、DP-1〜DP-7、Worker outbound-first、Worker間直接通信は標準非採用 | Stable Cloud ControlとOn-prem Workerの**組合せが設計上適合候補**。multi-tenant Hosted Consoleが運用済みと推測しない |
| EKI [Control/Data Plane](https://github.com/hat47x/eki-fabric/blob/main/architecture/CONTROL_DATA_PLANE.md) §4／§8／§15：Control metadataと大容量Artifact transferを分離。Site cacheと短命Grant、Observation≠Authorityを保持 | **余分なEKI経由のbulk data proxyを標準にしない**。Cloud/Edge直送はEKIのGrant/Access Brokerを通した限定アクセスであり、Data access自由化ではない |
| EKI [Issue #58](https://github.com/hat47x/eki-fabric/issues/58)（Suitability）／[#106](https://github.com/hat47x/eki-fabric/issues/106)：負担増なら分散しない。TEI小Functionは5〜11ms、EKIのAdmissionのみ中央値10.2ms（当時のSpike） | **EKIの通信・処理を挟む経済性の判断が不可欠**。これは既存Spikeの例で、現環境の遅延や最終配置性能を証明しない |
| TEI [#1016](https://github.com/hat47x/tei-platform/issues/1016)、[#1167](https://github.com/hat47x/tei-platform/issues/1167)、[#1182](https://github.com/hat47x/tei-platform/issues/1182)、[#1121](https://github.com/hat47x/tei-platform/issues/1121) | Application Composition、単一ApplicationのBrowser/API別配備単位、Lowering/Artifact・Host Targetを再利用。**新しいBuild/Deployment意味正本・TEI Host/Registryを増やさない** |

今回の**新提案**：EKIを「管理されたアプリケーション公開・配置・運用を選択できる**論理Target／Deployment Broker**」とする。ただし、その**公開面／App catalog／Deployment操作／Instance寿命**がEKI現行APIに存在するかは未証明。Hosted Controlという新しいEKI Application Faceの要件と、EKI Core Kernelの採用決定を分離して検証する。

## 2. 五つの「ローンチ形態」と実行主体を分ける

| Launch Profile（仮称） | 利用者から見える操作 | 長寿命の実行・サービス継続Owner | EKIが保持し得る事実 |
| --- | --- | --- | --- |
| **L0 Direct** | TEIから直接既存Host／Cloud／OSへDeploy | Cloud Service／既存Host／OS Service Manager | EKI利用なし。無駄なBroker経由を避けるBaseline |
| **L1 Batch/Task via EKI** | TEI成果物の一部をTaskとしてEKIへSubmit | EKI Task/Attempt/Lease、下位Worker/Resource Provider | Job、結果／可用性ではなくTask Outcome、資源・証跡 |
| **L2 Managed Application Launch via EKI（新候補）** | TEIからEKI管理コンソールでApplicationを選び、Release／Placementを申請 | **選択された既存Cloud/On-prem Orchestrator**がWeb/API常駐、Health、Rollout、Autoscaling/再起動、Ingress、SLOを担当 | Deploy Intent/Proposal、Target selectionと根拠、Provider Request/ID、観測されたRelease status、最小運用Evidence |
| **L3 On-prem/Edge Fabric via EKI** | 組織内の小型/端末Poolを安全に登録し、利用可能なBatch/一時Taskを割当 | Edge Agent/Local Governor/Owner、必要なら既存Orchestrator | 登録済Delegation/Resource Class、短命Lease、Task/Result、Node/Agent失効・安全な容量 |
| **L4 Hybrid** | Cloudの安定Web/API＋Edge/SBCの補助計算を一つのApplicationとして設計 | **Web/APIは安定Host**、断続的な分散TaskはEKI、Domain DB正本は独立Provider | 紐付けたTask/Artifact/ResultとResource Observation。Web Business CommitはEKIの正本ではない |

**最初の採用候補はL2で「既存Cloud HostへApplicationを配置するDeployment Broker」を限定すること。** いきなりEKI自身をPaaS Runtime、Load Balancer、Domain DB、Availability Controllerにしない。TEI ApplicationにEKI利用必須も課さない。

Cloud/APIサービスの通常のClient通信は**Cloud Load Balancer／Gateway→Provider-hosted Service**へ直接送る。管理者のDeployment Plan/Policy変更・定期的ObservationだけEKI Controlに届く。EKIがすべてのUser HTTP要求、WebSocket、DB Query、Artifact bytesを中継する構成は**非目標**。Security上必要なGatewayは独立のAccess/Ingress Providerが管理し、その明示した費用・経路を記録する。

## 3. 物理構成：安定Control、外部実行、直接Data Path

~~~text
                 ┌──────────────────────────────────────┐
 Developer / Ops │ Hosted EKI Management Console       │
  (TEI-generated)├─ Application Catalog / Deploy Plan │
                 │ Read-only Topology/Simulation/AI    │
                 └────────────┬─────────────────────────┘
                              │ Admin (identity, diff, current authority)
                  ┌───────────▼──────────────────────────┐
 Stable Cloud     │ EKI Native Control / State / Lease   │
                 │ Placement / Broker / Evidence         │
                 │ Independent bootstrap/CLI/recovery    │
                 └───┬───────────────┬───────────────────┘
                     │ Provider/API  │ Worker outbound registration/control
        ┌────────────▼───────┐ ┌────▼───────────────────────────┐
 Cloud  │ External Managed   │ │ Local/On-prem Edge Fabric       │
        │ K8s/Container/App  │ │ PC/GPU node/ARM64 SBC/Pi       │
        │ Service+Ingress    │ │ Site Gateway/Cache (optional)  │
        │ Cloud DB/Secrets   │ │ Local Governor+Agent           │
        └───────────┬────────┘ └──────────────┬─────────────────┘
                    │                        │
 Public/Private ────► Provider Service        └─── limited Data access → site store
 Client traffic      (NOT via EKI)
                            Data/Artifact transfers with scoped Grants:
        Provider/Worker  ───────────────► Cloud Object Store / Site Cache / DB
                            (NOT through EKI Control metadata path)
~~~

**ホスト基盤との相性**：KubernetesのService LoadBalancer／Gateway/Ingressは、選択したCloud Provider上で外部EndpointとBackend Serviceを結ぶ。EKIが同じデータ面のLoad Balancerを再実装する必要はない（[Kubernetes Service](https://kubernetes.io/docs/concepts/services-networking/service/)）。既存Kubernetes／KueueやSlurm等は、専用のEKI Resource Provider Adapter越しに利用する。**KubernetesのNamespace/QuotaとEKI Budget・Retryを二重に実行**していないかを検証する。

**EKI自身のControl/Console**：安定したCloud VM/Managed Cluster等で起動・更新・復旧する。Provider Identity/Secret/Metadata Store/HA、ログ監査、ネットワーク分離、Backup/Disaster Recoveryは別の運用義務。Consoleを**EKI Control Planeの可用性に巻き込まない独立配備／Break-glass CLI**を保つ。TEI Studioの稼働やTEI製ConsoleからしかEKIを復旧できない構成は不適合。

### 3.1 マネージドConsoleの貸与：Tenantごとの権限と開示

Hosted Management Consoleは、Tenantごとに権限・Application/DeploymentのScope・Environment（dev/stage/prod）・Provider Account/Grant・Secret reference・Quota/Budget・監査を分離し、**管理画面を開けること≠Admin操作権限**とする。一般利用者にはNode count／Host/IP／他TenantのArtifact existenceを表示しない。TEIのAI Review/SHO等が必要とするPrompt Contextも同じDisclosure制約を通す。

Client-facing操作（ユーザーアプリを利用）、TEI DeveloperのRelease提案、EKI OperatorのPolicy/Pool変更、Provider側の実Deploy・Cloud IAMを**同じPrincipalで済ませない**。EKI Admin UIがProviderの長寿命Root credentialsを保持・ブラウザへ配布しない。認可可能なProvider Adapterへ限定的・短期・目的拘束のCredentialを渡す。

提供・課金・請求等の商用機能は未設計。**コンソール貸与**は先にTenant Isolation/Operation Scope/利用Evidenceを意味し、料金体系／Application hosting商品の販売可能状態を含意しない。

## 4. TEI Release→EKI Deploy Intent→外部Host Actualの束縛

**受理済Application Meaning ≠ 同じバイナリ ≠ 配置されたサービス ≠ 業務上の成功。** 必要なID/証拠とOwnerの切り分け：

| 段階 | 具体内容／Identity | 成功の上限 |
| --- | --- | --- |
| A. TEI Build | Application/Source/Meaning Revision、Browser/API/Worker Artifact Set digest、OS/arch/Toolchain、Target Host Contract、Build Evidence/Attestation | 生成物の同一性のみ。署名/検証の現在資格は別 |
| B. TEI Deployment Plan | Release ID、Environment、Provider Selectionの制約（Data Class、Latency、Cost、Uptime、CPU/GPU/RAM/Arm64等）、必要なSecret reference、Migration/rollback policy | **Proposal**。EKIへのRegistrationや実Provider準備はまだ |
| C. EKI Broker/Administrative Admission | Proposal ID、Approval/Current Policy、Scope/Quota/Budget、候補ProviderのSuitability、Provider Agreement、Task/Deploy Intent ID | EKIが申請を受理しただけ。Provider Resourceの実確保・Host権限は別 |
| D. Provider Deployment | Kubernetes Deployment/Service/Gateway等の実Resource ID/Revision、Container digest、Cloud Account/Region、Runtime Service Identity | Providerがdesired設定を受理しただけ。Pod healthy≠業務契約成立 |
| E. Actual Observe | Rollout readiness、Endpoint/TLS、DB migration（必要な場合のみ独立管理）、実Consumer HTTP/Web/CLI検証、Owner-current/Effect、Receipt、independent read | **観測された範囲**だけ実配備合格。EKIの`accepted`／Providerの`running`だけでTEIの業務完了にしない |
| F. Lifecycle | Revision-pin、Scale/stop、reconcile、Provider失効、rollback、Release取り消し、Domain Migration、DR | EKI BrokerはCloud/K8sの実Service状態を独自の正本へ上書きしない。Intent→Provider actual→Observationの差分を保持 |

**重要**：TEIの過去設計にある`Release Graph→Durable Deployment Run→Executor Command→Observation/Evidence→Reconcile`は、**意味の要求・実際の変更・現在の観測を分離する考え方**として再利用する。現在Mainに同名の完全なProduct APIが実装済みだとは主張しない。GitHub上の[#1167](https://github.com/hat47x/tei-platform/issues/1167)はApplication別Goコンパイル結果の差分、[#1182](https://github.com/hat47x/tei-platform/issues/1182)はBrowser/API二物理配備単位の未実証部分を別追跡。EKIへLaunchすることで双方を自動的に満たしたと扱わない。

## 5. Cloud・On-prem・PC・SBC/Piの異種資源と安全な接続

### 5.1 Nodeを同じWorkerと仮定しない

| Node Tier（提案） | 最初に向くWorkload | 明示すべき実装制約・非目標 |
| --- | --- | --- |
| Managed Cloud compute/managed Kubernetes | Web/API常駐、DB接続、期限・Availabilityの厳しい処理、GPU Batch | 本来のAvailability/Scaling/IngressはCloud Providerが保証する範囲。EKIが同じ基盤のControl Costを加えるだけならL0を選ぶ |
| On-prem管理サーバ／GPU Workstation | CPU/GPU Batch、Data-localな解析、管理されたPool | Site Identity/Owner、Policy、Quota、Driver/VRAM、障害隔離、Endpoint/Artifact転送、管理クラスタの独立Scheduler |
| 従業員PC・個人PC | 割込可能なTask、再送可能・漏洩可能Data Classに合う計算 | Local Governorの停止・User優先、本人AI/backup負荷。常駐Web/API/強い期限の保証源にしない |
| Raspberry Pi等のLinux/ARM64 SBC | 軽いCPU Task、データ前処理、並列Monte Carlo、軽いAgent、site-local Cacheの一部 | ARM64 Linux・使用可能RAM/CPU/SSD/Network/crypto/OSが足りること。x86_64 Artifactをそのまま実行不可。GPU/VRAMや高速AI推論能力を想定しない |
| Piより制約の厳しいMCU／RTOS／超低価格機器 | Sensor/Event/軽いEdge推論・命令実行など、**薄いData/Device Provider候補** | フルEKI Worker Agent、Go単一バイナリ／TLS/Artifact verification／Linux Serviceが必ず動くとは限らない。必要なら**Site Gateway越しの端末**として扱い、Worker Identity/Agentと同一視しない |

**異種Targetの適合**：TEIは選択されたWorkloadごとに`OS/architecture`、ABI、CPU feature、必要Memory/Storage、Sandbox/Runtime、GPU Device mode、Network/Trust、Executor Artifact digestを宣言し、適合するTargetをビルドする。多アーキテクチャOCI Image Indexを用いれば適合するImage VariantをNodeが選べる構成もあるが、**配布可能なmanifestがあることと各archでの実動互換・実性能は別**（[Google Cloud multi-arch guide](https://docs.cloud.google.com/kubernetes-engine/docs/how-to/build-multi-arch-for-arm)）。未ビルドarchや必須Memory不足のWorkerへ意図せずFallback配置しない。

### 5.2 Secure networkは「Worker outbound-first」を基本にする

EKI [Deployment Profiles §5](https://github.com/hat47x/eki-fabric/blob/main/architecture/DEPLOYMENT_PROFILES.md)では、WorkerからControlへの**outbound-first**が基本で、任意Worker間P2P直結は標準採用していない。社内NATや家庭Routerの下でも、Inbound portを無差別に開けることを不要にする構成を優先する。

- **Enrollment／Identity**：Worker/Nodeごとの検証可能なDevice Identity、Managed Enrollment、期限、revocation、Agent digest、Tenant/Trust DomainへのBinding。EKI [#222](https://github.com/hat47x/eki-fabric/issues/222)ではmTLS／OAuth-DPoP／workload identity等を**比較中**。固定Bearerの試験用Profileを本番の登録要件へ昇格させない。
- **Control channel**：Outbound authenticated session（mTLS等は選定待ち）とLease/WorkOrder/Policy署名・nonce・fencing。通信切断時のLocal Governor fail-closed、旧Lease/revoked Nodeの再参加拒否。
- **Data channel**：仕事のInput/OutputはEKI Controlを通さず、ArtifactGrantとData Access Brokerから短命Scoped Accessを得てCloud Object Store／Site Cacheへ。生の社内Storage CredentialをWorkerへ渡さず、Workerの任意Internetアクセスを許可しない。
- **On-prem Site Gateway**：多数の小型Node／不安定LANのAggregator/Cache候補。統一して登録できても**Gatewayが全Workerの受理・実行結果・Authorityを自分の任意判断で捏造しない**。Resource/Result Claimと根拠はNode/Providerごとに追う。Worker間P2Pや共有FS/Shared MemoryをEKIが提供する前提にしない。
- **Private application ingress**：オンプレにWeb/APIを公開したい場合は、別途許可されたVPN/Zero-trust Tunnel/Reverse Proxy等の既存Ingress Providerを用いる。Worker接続用のcontrol channelで公開WebのUser Dataを全転送する設計を既定としない。Outbound-only Tunnelは成立し得るが、**端点の本人認証・アプリ認可・ネットワーク経路/帯域/Relay費用は別**（[Cloudflare Tunnel concepts](https://developers.cloudflare.com/tunnel/concepts/)は比較例でありEKIの採用Targetではない）。

**Hybridでの状態正本**：Raspberry Piの一時離脱やClient再起動によっても、業務DBやService identityがPiへ移されてはならない。PiはWorkerとしてTaskを受けるか、外部の安定Serviceへの依存先として使われるかを別契約とし、ローカルPCの利用者優先とデータ分類を守る。

## 6. 「EKIを介することで無駄な消費をしない」を測定可能な製品品質にする

### 6.1 EKIを外した直接運用との比較を必須にする

**3つの候補を同じWorkload/Host/Artifact/Network/Measured priceで比較**：
- `DIRECT`：TEI Artifact→Cloud/既存Hostへ直接Deploy/Run
- `EKI_BROKER_SAME_HOST`：EKIでDeploy申請し、**同じHost**へ配置。User traffic/Data pathは直接であり、追加はAdmin/Observationだけ
- `EKI_FABRIC`：EKIが別Worker/On-prem/Cloud Providerへ分散し、Cache/Transfer/Verify/Retryを伴う

初期指標は**独立**に取る：`total wall time`、P50/P95 interactive latency、Job end-to-end、worker execution CPU/GPU time、EKI Control CPU/memory/DB/ops cost、data transfer bytes/egress fees、Artifact duplicate transfer、Provider managed service/Ingress charges、retries/preemption、warm-up/cold-start、User impact、Availability/Recovery、Security assurance差。相殺する単一Scoreにすぐ統合しない。

**比較式（説明用）**：
`Total cost ≈ Host compute/storage/ingress + EKI control/console/observation + extra transfer/egress + verification/retry + operational overhead`。
`Elapsed ≈ baseline runtime + control/admission + transfer/queue/cold-start + verification/recovery`。
配置先・Networkが異なるケースでは、応答遅延の単純和や利用料一律比較が成立しないので、実TraceのCritical Pathと課金単位で測定する。

**採用停止条件**：EKI_BROKER_SAME_HOSTで管理対象が増えただけなのに全Client HTTPがEKI proxyを経由／secretが共有／EKI停止で既に公開済みCloud Serviceの通常利用が止まる／管理費用が削減便益を上回る、という場合はそのBroker機能を採用しない。**管理コンソールの便利さに対する費用はゼロにはならない**が、処理本体への不要な追加Hop・CPU/GPU占有を最小化する。

EKI [Issue #106](https://github.com/hat47x/eki-fabric/issues/106)の既存TEI Spikeでは、数msの純粋TEI Functionを遠隔SubmitするのはAdmissionだけでも同程度の時間になり得る。**小さい処理はそのままローカル／既存Host実行**、大きい処理のみ分散する選択を基本に、既存[#58](https://github.com/hat47x/eki-fabric/issues/58) Suitability Gateを再利用する。

### 6.2 EKI Control障害・Cloud障害・Site切断を分離

| 故障 | 望ましい縮退動作 |
| --- | --- |
| EKI Hosted Console障害 | 既に稼働中のCloud Business ServiceはProvider側で継続。EKI Native Controlに影響しない構成を検証 |
| EKI Control API/Metadata Store障害 | 新規Broker Deploy・Placementはpending/unknown。Providerに正当に登録済みのServiceはEKIを経由しないUser trafficで継続。Local Governorは独立 |
| Cloud実行Provider障害 | Provider/Host/SLOの責務でRecovery。EKIが候補再配置を提示する場合も、移すData/DB/Secret、Region/Trust、DNS/Ingress切替のOwner承認・Receipt必須 |
| On-prem Site/PC/Pi断絶 | 失効したWorker/Leaseの再利用を拒否し、Taskはunknown/reconcileまたは別Workerへ再試行。Local GovernorはControl到達を前提にしない |
| EKIの管理Consoleに使うTEI生成物の障害 | 別の独立Native CLI／Provider Management APIからEKI Control/Providerを復旧可能。TEI Runtime不在で署名・Lease・Recoveryが動く |

**Control/Consoleのクラウド安定化と、Cloud Providerの実行保証は別**。EKI自身のControl NodeのHAがあっても、Node WorkerのAvailability・業務DB Transactionを保証しない。Cloud Region障害／Managed Cluster停止で復旧に使うPlaneがなくならないよう、Provider-native break-glassと管理面のデータバックアップ／資格の分離を計画する。

## 7. 最小独立Fixtureと採否Gate（全件NOT RUN）

| Case | 想定条件 | 独立Expected／負例 |
| --- | --- | --- |
| **AL-01 同一Host direct vs EKI broker** | TEI同一Artifact／Cloud Host、DirectとEKI Brokerで配置 | 公開User HTTPは同じProvider Ingressを通り、EKI経由bulk traffic=0。追加EKI CPU/Memory/Latency/DB費用を分離計測 |
| **AL-02 EKI Control停止後もService継続** | EKI APIを遮断、Cloud Managed Serviceの既存Deploymentが稼働 | 既存Web/API ConsumerとProvider Healthが独立成功。EKI新規Deploy pending、Service間通信はEKIを要しない |
| **AL-03 Workerのみ断続参加** | 社内PC、ARM64 Pi、Cloud Worker、3種類のNode | Worker/AgentのOS/Arch/Memory/Network/Trust/Executor digestに適合したTaskだけ受理。Piにx86 BinaryやGPU不可Taskを配置しない |
| **AL-04 Site offline/復帰** | NAT下のOn-prem Worker Siteを切断→復帰。古いLeaseが残る | Outbound-firstのみ。再接続時に旧Lease/currentnessを検査。見えないWorkerをavailableへ補完しない |
| **AL-05 新しいServiceのDeploy時にProvider失敗** | EKI proposal accepted→Cloud/K8s拒否・部分成功・Timeout | EKI accepted≠実Deploy。実Provider ID/RevisionとReadを独立照合し、unknownをsuccessにしない |
| **AL-06 Browser/API別Artifact** | TEI同じAccepted ApplicationからBrowser+Go APIをCloudへ | BrowserにDB/Secret/Server実装が混入せず、実Origin/TLS/認証・結果を検証。#1182のDoneへ勝手に昇格しない |
| **AL-07 Tenant情報の隔離** | 2 TenantがHosted Consoleを共用 | AのNode数、Host名、GPU構成、Task/Artifact、Source/Secret、Resource budgetをBへ開示しない（URL/Graph/集計/AI Prompt/ログ含む） |
| **AL-08 Node Resource競合** | Pi/PC含むWorkerでCPU/RAM/VRAM差、同時Reservation 60+60>100 | EKI #354のReservation Vector/Local Governorで不適合を止める。Dashboardの予測値で予約できない |
| **AL-09 管理者資格・Cloud Secret失効** | Provider Tokenの失効、EKIから古いDeploy再送 | Provider実権限に現在性を検証。Browser/WorkloadへRoot Secretを露出しない。旧Deploy許可再生禁止 |
| **AL-10 Transfer/Egress非効率** | 同一Hostで処理するケースと別Region/Piへ大量Artifactを送るケース | Network bytes/cost、cache hit、retry stormを実測。Securityやデータ所在を満たしても性能劣化ならDirectを選ぶ |
| **AL-11 Pi/MCUの能力不足** | RAM/Storage/crypto不適合、RTOSでGo Agent非対応 | Unsupportedを診断。MCUは薄いDevice Providerとして別境界、全面Workerとして偽登録しない |
| **AL-12 Reconcile/rollback** | EKI Proposalは旧Revision、Provider rollout中に別部署が直接変更 | Source/Artifact/Policy/Provider actual/current Evidenceを別々に照合。自動上書き・二重Deploymentを止める |
| **AL-13 ジョブと常駐の混同** | stateful Web APIを中断可能LeaseだけでPiへLaunch | 非適合：stable Availability/DB/Ingress Owner不在。既存Cloud ServiceまたはDomain SLOを満たすHostへ戻す |
| **AL-14 Hybrid Cloud+Onprem** | Cloud Web/APIの非同期TaskをOn-prem Resource Poolへ転送 | Web request pathはCloud直通、EKIはTaskとResourceを管理、業務DBのCommit/Result unknownはTEI Host/Providerで別検査 |
| **AL-15 Low-level Stop権限** | EKI Control down／UserがローカルPCでopt-out | Local Governorは自律停止。Cloud側のDeployment OwnerがPCを強制再開させない |
| **AL-16 多数Piの大規模Pool** | 同時100 Node、混在arch/OS/断続回線、Site Gateway/cache | Single Gatewayが全Workerを偽装しない。Enrollment/Registry/revoke/Heartbeat、Task split/Result verifier/Network costの実測。数値の成功保証は実験後 |

Fixture/図のExpectedは**TEI生成DSL/実Simulatorの式をコピーせず**、Independent Oracleの根拠で固定（TEI [#2514](https://github.com/hat47x/tei-platform/issues/2514)）。すべて机上設計・**NOT RUN**。Adapterが未存在なら`unsupported`を出し、Fake providerでPASSしても実Cloud/API/Workerの適合済みとは扱わない。

### 7.1 Resource DelegateをEKIの標準的な外部単位にする（2026-11-03）

[TEIのAI JobとEKIのPool-first広域委任契約](TEI_AI_WORKLOAD_EKI_AGGREGATE_DELEGATION_2026-11.md)を受け、Cloud/SBC/PC/HPCを個別Workerの横並びではなく、**Managed Cloud/Managed HPC/Dynamic Corporate-EndpointのResource Delegate単位**として登録・選択する案へ修正する。EKIがHPC各NodeのProvisioning、GPU分割、端末資産台帳を持つことを標準目標にしない。既存HPC Scheduler（Slurm/Kueue等）・Managed Cloud・端末Security/ITAM/MDMのOwnershipを温存し、EKIは複数Providerのうちどこに期限・予算・Data Class・Trust・Result要件に沿って委任するかを判断する。

**Cloud Stable Control/Console**と**Delegateが所有する実Execution/Data Plane**は独立する。Managed Applicationの実Host／Ingress/DBのRuntime Authorityも元Providerのまま。非常に安価なArm/SBC群は、命令/メモリ/通信/電力を満たし、明示登録されたDelegateの下でのみ扱う。Workerへの個別認証やLocal Governorの実拒否権が必要なProfileでは、**管理単位がPoolでも実行の正当性は省略しない**。

EKIの内部を細部なしの“単一仮想マシン”と見なすことはできず、必要なWorkload class/OS-arch/単一Device Memory/Trusted Data Class、実Admission/Reservation/費用の差をSub-offerとして伝達する。Kueue MultiKueueやSlurm Federationの既存協調方式を参考とするが、EKIに内部Schedulerを移植する案ではない。

### 7.2 実際のLaunch可否とManaged Poolの予約・支出を二段階に分ける

[未解決Gate](TEI_EKI_AGGREGATE_DELEGATION_UNRESOLVED_GATES_2026-11.md)§3・§5により、EKI Consoleで`Cloud/HPC Pool`が表示されたことを、実ProviderがリソースをReserve/Deployできる保証と混同しない。TEI ReleaseのArtifact digest→EKI Admission→BUD-1/CAP-1費用枠→下位Cloud/Slurm/KueueのReservation/Deployment actual→Provider Receipt／独立ヘルスを順序を分けて表示。**Provider ACK喪失は`unknown`、費用確保解除はProviderの実Observationまで保留**。TEI生成サービスの可用性・Ingress/DBは既存Hostの責任。高変動Endpoint Poolは`guaranteed=0`を許容する可中断Taskに絞り、SLO/実動資源をCloudサービス同等に見せない。

### 7.3 Cloud Controlの時間・費用判断は、実HostのSLOを置換しない

[Deadline/Budget/Recovery-aware Coordination](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)により、Managed Cloud/Managed HPC/Dynamic Endpointの選択は単純なCPU/GPU割当でなく、Queue/Provision/Network/Verified ResultのEnd-to-end時間と、失敗後の再確保・予算・期限前エスカレーションを比較する。実運用のApplication Serviceは既存Providerの可用性契約を維持。**EKI Cloud Controlが提供した予測はHost SLAでもCloud Capacity reservationでもない**。Scale-outはBUD-1/CAP-1とProvider quota/Cost Authorityを満たした場合だけ、停止・解除・支出反映はProvider receiptによって確認する。


### 7.4 事前Stage・有償Capacity確保・Launchを分離する（2026-11-04）

[期限・予算計画§12](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)へ準拠。TEI生成Go実行体のBuild ArtifactをCloud/HPC/Endpoint Poolへ`Stage`することと、BUD-1/CAP-1・Provider Reservationによる**有償・実CPU/GPU Capacity確保**、さらにEKI Task/Leaseによる**実行権限**は、費用・許可・失効・Receiptがそれぞれ異なる。Stageに成功しただけでCapacity/SLO/Runを保証しない。機密Source/Inputを管理者未承認の低信頼PC Poolへ「Cacheのため」配布しない。Workerから任意URL・Toolchainを取得させず、承認されたData Plane Store/ArtifactGrant、既存Provider配備機構を使用する。Cloud ControlのControl PlaneをGB単位のArtifact Proxyへしない。実Stage/Launch/Resultがない限り`ready`を単一の真値として表示しない。初期実証は一つのGo Binary・一つのPure JobのDirect vs EKIに限定する（CL-01〜08すべてNOT RUN）。

## 8. 段階導入とIssue/ADR責務の所有範囲

| Gate | 最小の出来上がりと受入条件 | 主要Owner |
| --- | --- | --- |
| **G0 方向の確認（今回）** | `EKI=Launch Target`を**Logical Placement/Deployment Broker**として限定。既存EKI「service hosting非目標」との衝突・追加判断、Managed Consoleの必要権限・User Data Pathの非通過を記録 | EKI [orchestrator§7.3/§10](https://github.com/hat47x/eki-fabric/blob/main/research/integration/orchestrator-integration-and-layering.md)／[ADR-0013](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0013-product-core-and-faces.md)、TEI #1016/#1121 |
| **G1 Cloud Direct Baseline** | 同じTEI Release/Hostへ直接Deploy。Go API/Browserの二単位、Startup、Provider Health／独立Consumer、実行権限・Secretを検収 | TEI [#1167](https://github.com/hat47x/tei-platform/issues/1167)／[#1182](https://github.com/hat47x/tei-platform/issues/1182)／#943/#951、既存Cloud Provider |
| **G2 EKI Placement-only Spy** | 同一Cloud HostへBroker経由の**read-only candidate**を提示し、コスト・Authority・Data Pathを比較。実Deployは人間/Providerの承認で | EKI #58（Suitability）、[#112](https://github.com/hat47x/eki-fabric/issues/112)（External Contract）、TEI #2497/#1124 |
| **G3 Minimum EKI Deployment Broker** | Release proposal→EKI Admin acceptance→外部Providerの正式Deploy API→実Provider/Consumer再観測。Idempotency・Rollback/unknown・Auth/Issuerを独立Fixtureで検証 | EKI Admin/Resource Adapter（**新機能採用が要る**）、TEI #2499/#1174、EKI #57/#242現MVP Gateを妨げない |
| **G4 Private/Onprem Pool** | Cloud Controlとoutbound Site/PC/Pi Agentを安全登録し、実CPU Task 1つを配分・中断/復旧。arm64とx86の異種Artifact・証拠を固定 | EKI [#222](https://github.com/hat47x/eki-fabric/issues/222)/#118/#354/#123、DP-1/2/7とSite Cache、TEI #943/#259 |
| **G5 Managed Console／Hybrid E2E** | TEI-generated ConsoleでImpact/Simulation/AI Review→型付き管理Proposal、Provider側RuntimeへRelease/Cloud Service、On-prem Poolへ補助Task。EKI Control停止・Tenant隔離・Cost/LatencyのIndependent Benchmarkを通す | TEI [EKI資源運用計画](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)、EKI ADR-0021、TEI #2514/#1124 |
| **G6 本格採否** | Hosted Serviceの可用性/費用とセキュリティを**実測**してDirect vs EKI Brokerと比較。望ましい価値がない構成はL0を選び、使わないRuntime層を削除 | EKI Product Owner/TEI Product Owner/組織のProvider Operator |

**EKI側へ渡す追加Gap（未反映）**：「EKIへのTEI Application Launchは、常駐App Hosting機能か、**Placement/Deployment Providerの一元管理**か」を明示決定。Hosted Console Tenant isolation、Deploy proposal→Provider actual→Receipt API、外部ProviderのService Runtime Owner、管理者が選んだPoolとWorkerの匿名化、Egress/Execution overhead実測、Site/Pi Enrollmentと異種Target、Admin Planeの資格とRecovery。既存[EKI orchestration research§7.3](https://github.com/hat47x/eki-fabric/blob/main/research/integration/orchestrator-integration-and-layering.md)、[#58](https://github.com/hat47x/eki-fabric/issues/58)、[#112](https://github.com/hat47x/eki-fabric/issues/112)、[#222](https://github.com/hat47x/eki-fabric/issues/222)、[#130](https://github.com/hat47x/eki-fabric/issues/130)のOwnerに対応し、新しい第二Fabric Control Plane/TEI Hostを起こさない。

**TEI側へ渡す差分**：[#1016](https://github.com/hat47x/tei-platform/issues/1016) Composition/Architecture Profile、[#1121](https://github.com/hat47x/tei-platform/issues/1121) Lowering、[#1167](https://github.com/hat47x/tei-platform/issues/1167)/[#1182](https://github.com/hat47x/tei-platform/issues/1182) Native App/Browser/API成果物、[#943](https://github.com/hat47x/tei-platform/issues/943)/[#1104](https://github.com/hat47x/tei-platform/issues/1104) Artifact/Attestation、[#1043](https://github.com/hat47x/tei-platform/issues/1043)/[#2499](https://github.com/hat47x/tei-platform/issues/2499) Host Auth/Bounded Grant、[#2497](https://github.com/hat47x/tei-platform/issues/2497)/[#1124](https://github.com/hat47x/tei-platform/issues/1124)/[#2514](https://github.com/hat47x/tei-platform/issues/2514) UI/Conformance、[#1130](https://github.com/hat47x/tei-platform/issues/1130) Batch。過去の構成案のRelease Graph／Durable Invocationの思想は再利用するが、今のProductに未受入APIを実在扱いしない。

**反証／STOP**：TEIで生成したAppがEKIに「登録できた」だけでCloud ProviderのServiceを実配備したとみなす／状態の正本をEKIへ移す／EKIに全HTTP/Artifactを中継させる／Providerの同じJobを二重Schedulerでretry／Cloud Region・DB・Host権限不明のまま自動Failover／arm64 Workerへx86 Artifactを配置／ConsoleのUserがPool・他Tenant Nodeの情報を列挙／CredentialをClient/AI Promptへ流す／ObservationをDeployment Authorityへ昇格／EKI停止で既存Cloud Serviceが停止／Local Governorの拒否権が消える、のいずれかを満たしたらその結合案を拒否し、read-only説明を残す。

**現時点の証拠**：GitHub上のEKI Architecture/ADR/Research/Issue、TEI既存Issue、外部Kubernetes/Cloud Tunnel/multi-arch技術文書の**読取と設計照合のみ**。TEI Go Native/Browserビルド、EKI新Deployment API、Cloud VM/Managed Service、Pi/ARM64、Site VPN/Agent、Ingress、AI Console、実Data transfer/費用、Fault injection、AL-01〜16はすべて**NOT RUN／NOT ACCEPTED**。今回の変更はDraft計画文書のみで、正式EKI要件・mainへの統合・実動保証ではない。
