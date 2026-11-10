# TEI×EKI：資源群の委任を実運用可能にする未解決Gate／独立反証（2026-11）

- 日付：2026-11-03
- Status：**既存議論の監査と実装前の契約整理／すべて設計提案・実EKI/TEI/HPC/MDM試験 NOT RUN**。GitHubのADRやIssueの採否を本稿で変更しない。
- Scope：[AIジョブ構築・Pool-first広域委任](TEI_AI_WORKLOAD_EKI_AGGREGATE_DELEGATION_2026-11.md)、[TEI/EKI Job/Flow境界](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)、[EKI Resource Operations](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)、[Cloud/On-prem Application Launch](TEI_EKI_APPLICATION_DEPLOYMENT_FABRIC_2026-11.md)、[Market Fit](EKI_ENTERPRISE_COMPUTE_MARKET_FIT_2026-11.md)。
- 適用基準：TEIがAI/Data/Flow/意味のOwner、EKIは**委任の核と広域Resource Broker**、HPC/Cloud/Endpointは個々のノード・実行・資産の専門Owner。EKIの利用は任意。直接実行が適切ならそのまま利用。TEI固有Host／EKIの第二HPC Scheduler／独自ITAM／新しいFederation PKI・通貨市場を先行実装しない。
- **事実の基礎**：EKI [ADR-0013](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0013-product-core-and-faces.md)の委任の核、[採用ADR-0014](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0014-authorities-for-delegation-budget-capacity.md)のDLG-1/BUD-1/CAP-1/ADM-1、[Contract Boundaries](https://github.com/hat47x/eki-fabric/blob/main/architecture/CONTRACT_BOUNDARIES.md)、[Task State Model](https://github.com/hat47x/eki-fabric/blob/main/architecture/TASK_STATE_MODEL.md)、[Product Requirements](https://github.com/hat47x/eki-fabric/blob/main/requirements/PRODUCT_REQUIREMENTS.md) RQ-014/018/034/037/038/047/048/051/058/059/060/063、[Control/Data Plane](https://github.com/hat47x/eki-fabric/blob/main/architecture/CONTROL_DATA_PLANE.md)、[Deployment Profiles](https://github.com/hat47x/eki-fabric/blob/main/architecture/DEPLOYMENT_PROFILES.md)。

## 1. 既存案の未成熟箇所を、重複しないGateへ整理

| 優先・論点 | 前回案では未確定だった失敗の境界 | 既存Owner／本稿の結論候補 |
| --- | --- | --- |
| **G0 実行Artifactと結果本体** | TEI Jobに`executor.digest`・ワークスペースを要求したのにEKI External APIで取り込めない／実Resultが取得できない | **最優先の外部契約Gap**。EKI [#112](https://github.com/hat47x/eki-fabric/issues/112)：TEI仕事9種中8種の入力が422、結果取得501というSpike観測。理想のPool Brokerより先にPure Task 1件のbyte-bound E2Eを通す。TEI [#1121](https://github.com/hat47x/tei-platform/issues/1121)/[#1130](https://github.com/hat47x/tei-platform/issues/1130)と対照 |
| **G1 資源Offer→実予約** | Pool平均・古いOffer・同一Cloud quotaの二重表示から実行可能と誤認。二つのControlで同じGPU slotを確保 | EKI採用ADR-0014の**DLG-1**登録、**BUD-1**枠、**CAP-1**有償容量とProvider admissionを独立する。**Offerは予定、Reservationは実Authority** |
| **G2 委任先への多段指示・実行帰属** | EKI Task→下位Scheduler Job→内部Attempt/Retry→外部Effectの意味が一つの`completed`に潰れる | EKI Task/Attempt/Lease正本、Provider Execution Ref、TEI Business Operation/Receiptを**別IDと別Outcome**。既存EKI Task State Model、#67、TEI #1174 |
| **G3 MDM/Endpoint Securityとの連携** | `compliant=true`を`resource available`／Worker assuranceと混同。削除・通信断で新Taskを実行 | Asset/Compliance情報は登録根拠のみ。**今のLocal Governor/Agent登録・Trust Bundle・実行資格**を別検査。EKI #222、#466/#467の修正済み事例と実mTLS・失効の運用Gate |
| **G4 TEI Flowと非常駐・外部Provider** | TEIが停止したらFlowが進まない、EKIとProviderが同じStepを二重Retry。User希望の「設定レス」で監視主体が消える | Coordinator選択をDeploymentごとに固定。TEI Flow/GuardはTEI意味正本、EKIはTask/Attempt、外部Providerは内部Job状態。既存TEI #1129/#1130のCommit/Outbox/Callbackとは独立Gate |
| **G5 組織間委任・公平性** | A→B、B→CのGrantがA→Cへ推移する。Pool Budget・下位Quotaが二重消費。提供量をEKI内の金額へ換算 | EKI ADR-0014のBUD-1/CAP-1、RQ-034/037/038、[#125](https://github.com/hat47x/eki-fabric/issues/125)。TEI組織モデルの時点付きAssignment/Authority、EKIのDelegate permission、実Provider costを分離 |
| **G6 TEI管理コンソール／AI安全レビュー** | 観測Poolを現在の実Grantに昇格。非開示Node件数がGraph/AI promptで漏れる | TEI Modeler/Graph/Scenarioはread-only ProjectionとProposal。EKI Admin Authority／独立read／Providerへの実反映が実操作正本。TEI #2497/#2514/#2620 |
| **G7 事業性／オーバーヘッド** | EKI仲介でLatency・Data Egress・重複Reservation・運用工数が増え、Directより劣る | EKI [#58](https://github.com/hat47x/eki-fabric/issues/58)のSuitability Gateを入口にし、Provider別の費用/SLO/evidenceの比較で`Direct`を正常な選択とする |

**前提となる既存状態**：EKI #109（登録済Pool/RQ-058）／#140（広域最適化）は**closedで要求が整理されている**が、対応する実Pool ProviderのE2Eをここから確定しない。#67は二種類の管理クラスタProvider接続を要求するopen Issue。#112は現APIにTEI Job Artifact/Result取得のGapを記録している。#465（Runner→Body送信の静的問題）は**closed**となったため、過去所見を現HEADの現存不具合と断定せず、Attempt IDと実Output bytesの結合を継続E2E確認項目にする。#466のFingerprint計算と#467のTrust Bundle読取障害は**過去のSource-static所見であり、両Issueは修正報告と試験の後にclosed**。実攻撃や現在の全経路のfailureは未実証。

### 1.1 現main再確認によるG0/G3の証拠資格更新（2026-11-03）

[実行証拠の縦断§12](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)で、旧EKI #112/#242の不足を**時点・実装構成別**に再資格化した。現EKI `main`の`service/run.go`は`Results:store`と条件付き`Dispatch`を接続、`service/dispatch.go`は配置・Lease・WorkOrderの発行ループを組立て、`conformance/process/http_submit_agent_test.go`は別OSプロセスでHTTP Taskから固定Executorを走らせるTest-profileを持つ（**このターンのGo試験はNOT RUN**）。ただし同テストはADMITTED→READYを直接SQLite CASし、Result Acceptanceもテストから直接呼ぶ。**Product-onlyでResultまで自動進行するとは証明していない**。Output Artifact GETも`Worker.ArtifactsDir`／Meta/Owner/期限がそろえばHandler経路は存在するが、Input Artifactの`POST ingest-intents`・`PUT artifact-uploads`はまだ`routeNotImplemented`。以前の「Result取得501／Worker未配線」を現mainの一律Gapとせず、G0は**Input ingest＋Per-Job Executor binding＋手動補助なしの実AcceptedResult/Result Body取得**に絞る。

G3についても、現`service/run.go`の`currentTrustBundle`は未導入（NotFound）と**取得障害（error）**をSource上で区別している。旧EKI #467の失敗報告を現Source上の継続不具合と断定せず、実mTLS Handlerによるエラー伝播・revocationの負例と#466 fingerprintを別試験で評価する。EKI `clientapi/README.md`にはOutput GETは501／Result artifacts常に空という**最新Sourceと不一致の説明**が残るため、APIの真偽はOpenAPI・Handler・Process試験・実レスポンスを横並びで確認する。新しいContract/Registryは作らない。

## 2. 同じ「Resource Pool」でも四つの資格を一つにしない

**Logical pool ≠ fungible physical machine ≠ available capacity ≠ authority to run.** 運用Surfaceでは次を別Role／Source／Revision／有効時刻として扱う。

1. `RegisteredDelegate`：DLG-1/管理者が`pool ID`、所有者、Data Class、Assurance上限、許可Client、Issuer/Trust Domain、選択可能性を登録した状態。**誰が登録したか**をProvider自己申告にしない。
2. `ProviderCapability`：Providerが報告するRuntime Class（OS/arch/toolchain）、GPU Family/Partition/VRAM floor、CPU/NUMA・RAM、Container/OS isolation、Executor digest、Data locality。**申告/実測/Attestedの区別**を保持し、知らない能力はfalse相当ではなく`unavailable/unknown`として保守的に扱う。
3. `CapacityOffer`：一定時点と時間窓における需要増減込みの供給見込み。`offer_id, provider_revision, measured_at, observed_at, expires_at, availability_window, resource_class, lower_bound_if_guaranteed, burst_estimate, data_residency, cost_basis, provenance, completeness`等の**概念項目**を要求。新しい正式Schema/Registryの採用ではない。
4. `AllocationAuthority`：EKI側のDLG-1許可＋BUD-1 budget reservation＋必要ならCAP-1 cost-bearing approval＋現Provider Admission/Reservation＋実WorkOrder/Lease/Local Governorの強制結果。**OfferだけからLeaseを発行しない**。

**集約の規則**：Pool全体のCPU 256coreでも単一SMP Processに128coreを与えられるとは限らない。GPU 24GB×4は96GB単一GPUではない。異なるOS/arch、Data Class、Region、Tenant、GPU隔離ModeのOfferを一つの可換Capacityとして合計しない。必要なら`Pool/{ResourceClass}/{Trust/Data boundary}/{TimeWindow}`など**表示上のSub-offer**へ分ける。Sub-offerへの分割は第二の物理Node台帳を意味せず、Providerが公開を許した抽象的なResource Shapeで足りる。Topology confidentialな場合、Clientへは`satisfies/rejected/needs-recheck`だけを返し、正確なPool数・Node種別・内部順位を漏らさない。

**変動Poolの容量**：`guaranteed=0, estimated burst=20`はあり得るが、20台に対してTaskを予約済みとは言えない。予定された容量`can_offer`・確認済みの`can_reserve`・現在の`running`・実際に完了した`verified`を異なる時間軸で表示する。古いObservationで現在の供給能力を再利用しない。予測の信頼区間をSecurity Assuranceに昇格しない。

**Sourceの状態**：`fresh/stale/partial/unavailable/not-found/permission-denied`と`last-known`は異なる。特に`unavailable`（観測できない）を`not-found`（存在しない）や`zero-capacity`に変換しない。TenantのDisclosureにより非開示なら、非開示存在数そのものを漏らさない。

## 3. EKI委任と下位Providerの二重予約を避ける最小Hand-off

**提案**：独立した分散Txを作らず、権限の移動を**Intent→Provider Admission→現在性の再検査→限定Execution→独立Outcome**と段階表示する。具体的なResponse名や同期方式は未決。既存EKI External Contract/SPIを改変する前に、対応表とFixtureで検証する。

| 境界 | 決定主体・記録 | 許される結論 | 禁止する結論 |
| --- | --- | --- | --- |
| D0 TEI Job Declaration | TEI Flow Meaning ID、Input/Output/Executor digest、Effect class、Expected、Source pin、Ownerの実行意図 | あるJobが**定義・ビルドされた** | JobがEKIに登録・Providerで起動済み |
| D1 EKI Admission | ADM-2 TaskRequest/TaskIR保存・Idempotency、DLG-1 RegisteredDelegate/Trust、BUD-1 nested budget、SUIT Gate | EKI Task IDがあり、委任対象・予算・適格性を**評価**した | 下位クラスタのCPU/GPUが確保された |
| D2 Provider Reservation | Cap/Resource Typeに合うPool Adapter、実Provider Reservation/Queue Job Ref、CAP-1が必要なら有償確保の承認、Providerの現在Policy | 下位実行環境が**一定条件で受理/予約**した証拠がある | `provider queued`＝Running、またはLocal Governorの同意が確定 |
| D3 Attempt/Lease/WorkOrder | EKIのAttempt ID・fence/revision・上限・expiry、下位Provider execution refとの束縛。起動直前にProvider/Local Governorが再検査 | 当該Attemptを条件内で動かしてよい**限定権限** | Grantの期限超過で副作用が絶対起きない、古いAttemptが消えた |
| D4 Claimed/Verified Result | Provider result/Artifact bytes/Claim、EKIのRVR-1/RAC-1照合、Attempt/Task遷移 | EKI Resultが所定の検証に合格した範囲でAccepted | Worker/Slurm`COMPLETED`だけでEKI採用、さらにTEI Business Commit |
| D5 TEI Business Consequence | TEI Flow/業務Providerの現Meaning/Scope/Grant再評価、DB Commit/Effect Receipt、独立Read | 業務上の次工程・副作用が**別に**確認された | EKI Task acceptedだけで次工程を無条件開始 |

**相関Identityは最低4種**：TEI Business Operation/Flow Step、EKI Task/Attempt、Provider Job/Reservation/Lease相当、Artifact/Effect Receipt。下位Providerが`JobID`を再利用する場合はCluster/Provider ID・Job version/SubmittedAt等を加え、同名だけで同一対象としない。Agentの秘密鍵やProvider credentialをTEI ScriptやEKI Task metadataへ埋め込まない。Budget reservationはBUD-1が管理し、Provider側Quotaは独立の実制約。EKI予算残高を増やすのにProviderの資源を勝手に解放したと解釈しない。

**取消／失効／ACK消失**：Cancel requestとProvider actual stop、EKI Lease expiredとactual effect、CAP-1 release requestとcloud billable stopを分ける。Providerが`accepted`を返す直前の応答断は`admission_unknown`であり、別Jobの無条件再提出は不可。Providerが本番Job idempotency/lookupを提供しないなら、横断exactly-onceを保証しない。再照合できないEffectは`unknown/held`に残し、必要なOwner承認まで後続を止める。

**二重Scheduler**：EKI側は**Pool/Delegate選択**、Slurm/Kueue側は**内部Nodeの割当**。両者のAutoscale/Retry/Quotaはどちらが最終判断者かをProvider Bindingに明記。典型的には下位Pod/Batch Jobの再試行はProvider内の同一Executionを扱い、EKIが新Attemptを作るのはProviderが終端・再照合可能と確認した後。ただし再試行条件と参照IDの同等性はProvider固有で、全Providerに一律の二重排除があると主張しない。

## 4. Dynamic Endpoint Poolの「集約」は管理権限の丸投げではない

**既存Asset/MDM/EDRとの連携を、三種類の独立Provider能力として区別**：

| 能力 | 一次Owner | EKIが受け取るもの | ここから導出してはいけないこと |
| --- | --- | --- | --- |
| 資産・組織の所属・Compliance | ITAM/CMDB/MDM/EDR、組織・人事の正式Identity Provider | 管理者が許可した`device group`/Deployment Scope/Policy rev、compliance時点、端末の同意要件 | Compliance＝自由なCPU/GPU容量／Worker実行の正当性／顧客データへのアクセス許可 |
| 接続・Agentの実行資格 | EKI Worker Enrolment/Trust Authority（直接Worker）または管理されたSite Gateway Provider（内部Agent） | 認証済みIdentity/Issuer/Trust rev、失効・Signature/Bundleの現在性、Provider abilityの範囲 | Tokenが生きている＝本人の利用同意／旧CAのrevocationを無視してよい |
| 実容量・優先使用者の拒否 | Local Governor／OS実測／Provider admission | 短命Resource Offer、Time Window、拒否/Stop/Lease Evidence | MDMの`healthy`/EKIの前回Forecast＝現在のCPU/VRAMや実利用可能量 |

**連携方式候補**：社内既存のDevice Group/IT Asset Collectionを、管理者が委任対象として登録する**Group Reference Adapter**。Group内の個体Identity・サインイン主体・エンドポイントのIP/hostnamesは外部ITAM/MDM正本へ残す。EKIに必要な情報はClass・数の公開可能範囲・合計供給予測・Gateway Admission・失効通知の最小集合である。Agentを直接EKIへ登録するProfileでは[#222](https://github.com/hat47x/eki-fabric/issues/222)のWorker Enrollment要件が残り、`external group`だから無認証でもよいとはしない。

**修正報告のある歴史的なSource-static所見と残る実TLS確認**：[EKI #466](https://github.com/hat47x/eki-fabric/issues/466)はTrust BundleのIssuer FingerprintでSHA256(DER)とSHA256(SHA256(DER))が食い違うSource-static所見、[#467](https://github.com/hat47x/eki-fabric/issues/467)はTrust Bundle読取障害を未設定と同じfalseへ畳み、**従来のCA/Registry検証を通った相手に対する追加の版検査だけ**飛ばし得るSource-static所見。いずれも実攻撃未実証であって「すべてのmTLS検証が迂回できる」という主張ではない。**直接WorkerやSecurity Gatewayで高いTrustに昇格する前の既存EKI Owner Gate**として独立Oracle/実TLS経路を先に確認する。Trust Bundleが意図的に未導入／利用中なのにDB読取障害／対象Issuerが除外済みの三状態を別に検査。実装済み保証へ書換えない。

**失効シナリオ**：MDMがNode-Xを隔離→Device Groupの表示が更新されても、EKI側には30秒前のOffer、Node-X上には未期限のLeaseが残る場合がある。EKIは**新しい高機密JobをAdmitしない**、Gateway/Local Governorは再検査不能なら実行を拒否、既実行Taskについては停止Receiptか`unknown`の独立観測が必要。単にGroup membershipのUI色を赤くするだけで実停止が成立したと宣言しない。

## 5. 部門・組織・経済圏：予算と所有は階層、Trustは非推移

**組織階層とResource Pool階層は別**。TEIがモデル化するPerson/Position/Assignmentや組織時点と、EKIのDLG-1登録・BUD-1予約・CAP-1支出は、参照される`organization identity, valid-time, policy revision, authority proof`で結ぶ。組織の異動・承認代理の失効、Owner変更は業務意味の再受理と実委任の可否を別に判断する。部署/Group名が同じだから同じOwner・Grantとして継承しない。

**EKI採用ADR-0014を拡張せず使う**：
- `DLG-1`：管理者がResource Delegate・Trust水準・許可Client／失効を登録し、Trustを自動上昇させない。
- `BUD-1`：全体予算から組織／部門／案件／Task Groupへ**物理量**で枠を配分。上位枠・下位枠の二重予約と、重複するPool/ProviderのCapacity二重算定を別に検査。
- `CAP-1`：Cloud scale-upなど費用が生じる容量確保を、依頼者の枠＋組織の枠＋下位Provider制限で止め、必要なら人の承認を要求。EKIに請求/通貨/市場を内蔵しない（RQ-038）。
- `ADM-1`：管理者権限の限定・有期限委任状。TEIで記述した「組織Workflowが承認した」というClaimから、EKIへの管理者Grantを自動生成しない（TEI [#2499](https://github.com/hat47x/tei-platform/issues/2499)はOwner別Issuerの資格を分離）。

**Federation非推移**：組織AのCompute Grant→組織BのPool、組織B→組織CのPoolがあっても、Aの仕事をCへ移すには、AのData Authority/Policy/委任状が**Cを許す別の証拠**が必要。異なる管理DomainのBudgetや生メトリクスを中央集権で集めることを前提としない。TEIの組織Modelは再帰的所属・時間と版を表現できる候補だが、部署への所属＝秘密情報の閲覧権ではない。AdminのRead-only Graphへは該当OwnerのDisclosureを適用し、見えない隣接NodeやPoolの件数・場所を漏らさない。

**Fair Shareの位置**：組織間の「公平」は自動算出される物理法則ではなく、各Ownerが決めたAllocation/Minimum/Maximum/Borrowing Policy。EKI #125でopen。Contributionの量は物理観測であり、Credit/金銭価値への写像は外部制度（EKI RQ-037/038）。Ownerが承認していない`提供台数比で利用権を自動付与`を実装しない。

## 6. TEIでの操作・Simulation・AIレビューは、実実行の資格から隔離する

[TEI Resource Ops](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)のNode Level可視化とAI Reviewは、**管理する対象を増やす要請ではなく、必要なScopeでPool/Resource Delegateの状態を十分に理解するための投影**。Graph/Tabular/Timelineには`meaning subject ID`、`source/provider rev`、`observed_at`、`completeness`、`visibility scope`、`simulation revision`を保つ。

| Surfaceの状態 | 利用者に伝えてよいこと | 禁止する操作・解釈 |
| --- | --- | --- |
| `read-only observed` | 許されたPoolの出所・観測時刻・現在性・実行結果・未知 | 昨日のCapacityから今日のPlacementを承認する |
| `structural candidate` | AI/TEIがGraph/関係/Policy整合の候補を提示、出典と反例を付す | AIの候補だけでDLG-1/Policyを変更する |
| `simulated` | What-ifのCapacity/Deadline/CostとモデルLimitationsを表示 | Simulation PASS＝EKI Provider保証／CAP-1費用確保／Business Commit |
| `accepted meaning` | Ownerが何を業務・AI Job／組織Policyとして受理したか | その意味受理をEKIのWorkOrder/Leaseへ自動昇格 |
| `currently authorized action` | Host/EKI Admin Authorityが現在認可・CAS前提を確認した操作を申請 | UIの操作候補表示だけでExecution Grantがあると見なす |
| `execution receipt` | EKI／Provider／TEI Effect Authorityから得た各段階の観測事実 | ACK/ClaimをProvider Effect/TEI業務保存成功に丸める |

**AIレビューの最小対象**：オーバーブッキング、Capabilityの偽合算、古いOffer、Trust Bundle読取障害、再帰Org Grantの不正継承、Data transfer cost、二重Retry、Topology disclosure、Effect unknown。AIが見つけた反例は**独立Expected**へ落とし、既存Contract/Hostが停止させるか検査する。AI自身の回答、別AIとの一致、モデルのSimulated OutcomeをSecurity Authorityにしない。Providerの全Node構成を生成AIへ送らず、公開許可の集約情報だけで足りるか比較する。

**Developer/Operationsの経路**：
`TEI Model/Job Draft → Syntax/Meaning/Target validation → EKI Resource/Offer Observation → Simulation + Independent Oracle → AI Review → Preview / Owner Acceptance → Current Provider/Delegate Auth → EKI Submit/Reserve/Lease → Provider Actual → EKI Verified Result → TEI Business Effect/Receipt`。

## 7. 独立Fixture: テーブルで受入/失敗を先に固定（DG-01〜14、NOT RUN）

| Case | 固定入力・障害注入 | 必須Expected・停止条件 |
| --- | --- | --- |
| **DG-01** TEI Job byte-binding | 同じTaskRequestに別Executor digest、入力Artifact本体不在、Result取得501 | 対応不足は`unsupported`、不一致digestは拒否。入力/出力bytesの受け渡しが確実になるまで「AI Job実行成功」としない |
| **DG-02** Poolの偽合算 | Pool: 4×GPU24GB、Workload: GPU80GB 1枚 | Pool合計96GBで予約成功としない。ProviderのResourceClass適合確認なしなら`needs-recheck/unsupported` |
| **DG-03** 予約競合 | EKIとCloud別Clientが同じProviderの最後のGPU slotへ同時予約 | ProviderのAtomic Reservationで一方だけ成功、EKIのBUD-1/Provider Quotaは別。双方成功は不合格 |
| **DG-04** 曖昧Submission | ProviderがJobを受けた後でEKIへACKが戻らない | `submission_unknown`、Provider ID照合まで二重投入不可。照会APIなしなら`held`、無条件Retry禁止 |
| **DG-05** 二階層Retry | EKI Retry条件とSlurm requeueが同じEffectful Taskへ適用 | Provider execution identityが一致するまで新Attemptの外部Effect開始禁止。古いResult到着は別Attemptへ付け替えない |
| **DG-06** Endpoint失効 | MDMでNode-X noncompliant、EKIのOffer 30秒古い、Local Governor 0 | 新しい機密Task拒否。実Running停止はLocal/Gateway証拠を待つ。MDM更新の成功＝停止成功は不合格 |
| **DG-07** Trust失敗3種 | Bundle未導入／導入済み読取I/O失敗／旧Issuer除外済みの各Case | 従来互換・取得不能・明示拒否を区別。読取失敗で追加Bundle検査を黙って飛ばす挙動は不合格。EKI #466/#467の実TLS経路で検査 |
| **DG-08** 実装済/未実装区別 | #465がClosed、同AttemptのClaimとArtifact body senderが別起動、再起動/配送遅延を注入 | Closedを未修正と断定せず、各Attemptのbody bytes/digest/保存先を実E2Eで独立確認。Claim受理とBody取得を別観測 |
| **DG-09** Org time change | Task登録後にOwner部門/代理許可が失効、旧RunがCloud支出要求 | Meaning rev／Org valid-time／現在のIssuer/BUD-1/CAP-1を再検査。元申請時のOwner名だけで費用発生不可 |
| **DG-10** Federation経由 | A→B、B→Cのみ許可。Aの機密JobをCへ再委任 | A→CのData Authority/Delegationがないなら拒否。AのBudgetがCで消費されるように見せない |
| **DG-11** Visibility / AI漏えい | 利用者ScopeはPoolの粗い提供可否だけ、Provider内部Worker数は非開示 | Node/件数/位置/実Raw telemetryがGraph/tooltip/CSV/SHO/LLM Prompt/エラーへ漏れない |
| **DG-12** Cost / oversharding | TEI純粋処理 5～11ms、EKI API roundtrip 10.2ms（既存Spikeの観測）。Remote/Directを比較 | 同等の結果品質でRemoteが遅い/高いなら`DIRECT`が正常。単にEKIを通したことを成功KPIとしない |
| **DG-13** Group/Flow stop | TEI Coordinatorが停止、EKI Taskは完了、次TEI Guardは未再評価 | EKIだけで業務Guardを推定せず`awaiting-domain-evaluation`。事前許可済みStep Graph以外は進行不可 |
| **DG-14** Cloud cost-bearing hold | ProviderはScaleOut成功、EKI CAP-1承認はタイムアウト／取消中 | 実費用停止やCapacity ReleaseをReplyだけで確定しない。独立Provider Observation、予算枠の調停、unknown/Operator要確認を維持 |

**Fixtureの独立性**：ExpectedのByte/Digest/時刻/ID/予算・物理容量/Authority/拒否/Effectを事前固定。TEI生成RuleやEKI Schedulerの出力からExpectedを生成しない（TEI [#2514](https://github.com/hat47x/tei-platform/issues/2514)）。`Unknown`は成功でも失敗でもない。正例・負例・失効後・途中通信断・A→B→Aの二重revisionを含める。Synthetic合格は実MDM/HPC/Cloud/OS/APIの保証ではない。

### 7.1 時間と予算で変わる計画の受入条件

[EKI Deadline/Budget/Recovery Coordinator](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)のLB-01〜26をDG-01〜14と接続する。**LB-18〜26は複数Jobの予約競合、実回復Window予算、部門飢餓、期限未開始とEffect不明、複数Poolの共通障害、TEI業務Flowまでの完了境界を追加する。**未登録Pool/誤GPU class/MDM失効/下位Admission unknown等を避けた上で、**許容リードタイムと失敗後の残余時間を見積り、回復のための物理予算を残す**。費用だけ安いがDeadline/SLO未達の候補は最適としない。Estimatorの実績はPool/Time/Workload等級ごとにObservationとして保持し、少数サンプル/打切り・ドリフト・共通故障を扱う。EKI #108/#109/#140の採用済み要件に加える実装Gateであり、新しいPolicy Authority／Budget Registryは作らない。

### 7.2 Source/Artifact→実Providerの最小証拠を別実行資格で渡す

[縦断受入VS-01〜14](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)で、以前のG0〜G7/DG-01〜14に対する**実装前の一本のReference**を用意。W-1（純粋数量×金額、NULL/missingは別Expected）をT0直接Go→T1 EKI Native→T2 Managed Providerで同一Source/Artifact/Input/Oracleのまま比較し、W-2 changed-edge/Outbox/Business Effectは別のOwner Gateへ分離する。Providerが入力/結果のbodyや実予約を提供できないとき、Pure GoのPASSをEKI/Providerの合格に合算しない。プラグインと外部Providerに独立の実Grant/Revision確認を求めるが、新中央Provenance Registryは設けない。

### 7.3 既存Job進捗とGroup Budgetを混同しない（2026-11-04）

[実行縦断§13](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)と[Deadline/予算§11](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)を追加照合。TEI `internal/execution/runplan.go`の`RunPlan`、`runlog.File`の`RunKey/Steps/Attempt/FenceEpoch`は現`main`に存在し、Step再開の限定機能が実装済み。一方、同時別プロセスのCAS、外部Effect CommitとJournal記録の原子性、Plan/Executor Revisionの保持、実EKI Workerへの委譲はそのSourceだけでは証明できない。EKI #267/#381は**closed**、Task phaseと失敗時の`ready`戻しの試験が#695で報告された。EKI #115のProvider内部Retry制御、#126のGroup予算・期限・部分結果、#155の回復公平性は**open**。TEIのBusiness FlowとEKIのTask Groupを共通Scheduler/台帳へ統合せず、Owner別にVS-23〜32とLB-27〜32を配分する。

**Security再資格化**：EKI #466/#467はそれぞれPR #678/#675によって修正・試験が報告され**closed**。当初のSource-static指摘は歴史的Evidenceへ移し、更新後Trust/Issuer失効・実mTLSとGroup Gatewayの統合試験を継続する。修正済みIssueを未修正安全Gapとして新設・再度計画化しない。

### 7.4 ABI損失をP1 Pure Job Wireの実装停止線へ明示（2026-11-04）

[一改訂の実行縦断§14](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)を先行Gateに追加。TEI Draft [#2787](https://github.com/hat47x/tei-platform/pull/2787)はEKI実OpenAPI blob `388f2e3ad32ba663b4777fcfb4304b0cf6f48f3f`の固定証人に対する**WP-01〜19の19/19 exact-head CI PASS**を報告したが、これは実Task Process/Runtime適合ではない。P0 `parameters`はWorkOrder/Workerへ未伝達、`effect_mode`は`none`のみ、Executor `version_constraint`はAdmissionで未評価、`deadline`は単一時刻、Input ingestはHandler実装が501、Provider/Data AuthorityはTask自己申告では発行されない。**Source→意味→Workerで必須情報が一つでも黙って失われるJobは委任不適合**とする。

**新制度を増やさない**：本計画のG0は`Existing EKI ABI snapshot + TEI semantic checklist + real Worker bytes`という比較に絞り、WPの静的試験とRE/VSの実行試験を重複計上しない。初期Referenceは①Parameters不要なPure Job、②Typed input Pure Job（現Gap）、③Effectful Job（現P0不適格）。EKI #112を外部API Ownerとし、TEI #1130/#2514/#1180/#2811へ必要なFixture/独立Expectedだけを渡す。新しいUniversal ABI/Host/Artifact RegistryやAI自律実行権限は作らない。


### 7.5 Cache/Toolchainの局所性はProvider能力とData権限に分割する（2026-11-04）

[Deadline/予算§12](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)のCL-01〜08をG0/G1/G7へ接続する。EKI [#112](https://github.com/hat47x/eki-fabric/issues/112)の実験でビルドCacheあり/なしの時間差と、Toolchain再選択が記録されている。EKI [#58](https://github.com/hat47x/eki-fabric/issues/58)はInput TransferがCompute短縮を上回る確定的な反例を持つ。**Poolに空きGPUがあること・Cacheがあると申告されたことだけでは、TEI生成Jobを今使える保証にならない**。必要なOS/arch/Go Toolchain/Artifact Digest・Entry・Input Data Class/Grant/可用性と実Provider Capacity Reservationを別の現在性で評価する。`ProviderCapability`に未知がある場合はOptimizerの高順位を許可とせず、`needs_measurement/not_eligible`で止める。EKI RQ-029に反し、任意のWorker側取得でToolchainやWorkspaceを黙って補完しない。Cache HitはSecurity Authorityではない。**CLの実験はNOT RUN**、新しいStage ServerやRuntime CompilerをEKI Coreに加えず、既存EKI #112/#58／TEI #2788を実装Ownerとする。

## 8. 作業順・各Ownerへの引渡し・保留判断

| Phase | 依存と実装/受入の最小目標 | 反映先（**既存Owner**） | 先行条件 |
| --- | --- | --- | --- |
| **P0 source-to-contract** | 既存EKIのDLG-1/BUD-1/CAP-1、TEI Data/AI Job/Flow、Provider/MDM/Orgの所有境界をRead-only対応表へ。新Core型/新RegistryやAI/HPC Kernelを実装しない | EKI ADR-0014／#109/#140（採用設計）、TEI #1121/#1130/#2620/#2499 | Source/Revisionと提案の区別 |
| **P1 pure Job wire** | 一個のTEI純粋Executor＋Input Artifact digest/body＋EKI Task Admission＋Output bytes→TEI独立Oracleまで、既存Public APIでできる範囲と不足を実観測 | EKI #112/#57/#465（ClosedだがE2Eは別）、TEI #1121/#1130/#2514 | APIの422/501ギャップを成功扱いしない |
| **P2 one managed pool** | 管理されたSlurm/Kueue等一系統のDelegateでProvider Admission/Reservation→EKI Attempt/Lease→Result検証・Cancel/Retryを相関。第二Providerは差異が明確なものを選ぶ | EKI #67/#58、ADRs、TEI #1130 | 実Cost/Outcome/Provider ReceiptとAuthority |
| **P3 dynamic endpoint** | MDM/Endpoint ProviderのGroup reference、Gateway Offer、Local Governor、Trust Bundle失効の三境界を合成→実機で検査 | EKI #222/#466/#467/#118/#354、既存Endpoint管理Owner | 修正済み#466/#467の統合・実TLS・版交代試験 |
| **P4 higher-level coordination** | TEI Flow継続・EKI Task／Provider Jobの二重Retry無し、期間限定Grant/Cancel、組織異動/部門予算・Federation越境の反証 | EKI #125/#130/#140、TEI #2499/#1129/#1174/#1130 | 人/組織の委任Policy・実Credential Owner |
| **P5 TEI operations value** | Pool-first Graph/Table/Timeline、What-if、AI Review、Preview→Current Auth→Effect Receiptと独立Read。市場Pilotで既存ツール/Directと比較 | TEI #1124/#2497/#2514/#2620、EKI #58/#124／Market Fit | 開示と独立実Observed Stateが成立 |

**採用保留**：EKI External Contractの`ResourceDelegate/Offer`正式Schema、下位Scheduler↔EKIの二層Leaseマッピング、HPC ProviderをEKI Workerと等置するか、端末Gateway trust root、Credential形態、組織間fair-share数式、AIレビュー自動化閾値、TEI生成アプリのCloud Deployment Profile、APIのretry/Callback/Outboxを**本稿だけでAcceptedにしない**。先行P1/P2の反証が現行Contractに収まらない箇所だけ、EKI/TEIの既存Issueへ必要最小の差分を渡す。Buri／X-TEA継承は依然として別Gateであり、この計画で移植済みと主張しない。

**Done（本稿）**：既存採用済みAuthorityへ責務を戻し、曖昧な`Pool Resource`を現在性・容量と実利用権限で分け、Provider多段Hand-off／MDM/Trust／Organization/Cost／AI Operationを実行可能な負例と段階へ分解した。**製品コード・実ネットワーク・Provider/Worker/DB・Test/CI・顧客検証は未実施**。Draft planの段階で継続し、Owner受理と本番許可を行わない。
