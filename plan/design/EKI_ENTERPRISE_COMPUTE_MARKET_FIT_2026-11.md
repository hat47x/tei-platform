# EKIの社内・ハイブリッド計算資源可視化／運用一元化：既存製品・需要・製品境界の調査（2026-11-03）

- 状態：**公開競合・市場資料と既存EKI/TEI設計の照合に基づく製品仮説**。製品採用／顧客課金意向／売上見込み／競合評価の実測には至っていない。
- ユーザーの製品意図：EKIはジョブコントローラーだけでなく、社内CPU/GPU/メモリ・クラウド・オンプレ・低価格ノード・一時参加資源を、持ち主優先・安全な限定委任で共有する。TEIのモデル化・可視化・Simulation・AIレビューを、管理コンソール／運用Applicationとして最大限利用。TEI生成Applicationをクラウド等へ配置する入口としてEKIを介在できるとよいが、**EKIを通ることが無駄な資源消費を生まない**。
- EKI既存方針：[製品価値](https://github.com/hat47x/eki-fabric/blob/main/product/value.md)、[RQ-005/006/014/018/058/063](https://github.com/hat47x/eki-fabric/blob/main/requirements/PRODUCT_REQUIREMENTS.md)、[Workload Suitability EKI #58](https://github.com/hat47x/eki-fabric/issues/58)、[既存Orchestrator接続§7](https://github.com/hat47x/eki-fabric/blob/main/research/integration/orchestrator-integration-and-layering.md)、[Cloud/On-prem Deployment Profile](https://github.com/hat47x/eki-fabric/blob/main/architecture/DEPLOYMENT_PROFILES.md)、[Control/Data Plane](https://github.com/hat47x/eki-fabric/blob/main/architecture/CONTROL_DATA_PLANE.md)、[Human-first Governor](https://github.com/hat47x/eki-fabric/blob/main/architecture/AUTHORITY_BOUNDARIES.md)。
- 対応TEI計画：[Resource Ops/Simulation](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)、[ジョブ委譲](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)。
- **易経の「卦」は製品哲学を考える補助軸**であり、今回の調査だけからEKIに割り当てた特定の卦名・辞を確定したとはしない。製品思想の支持と市場の需要検証は別の証拠を使う。

## 1. 既存製品はある。空白ではなく重なり合う成熟市場である

| 市場・代表 | 公開資料で確認できた中核機能 | EKIの直接競争部分／まだ差がある仮説 |
| --- | --- | --- |
| **AI/GPU共有とクラスタ運用**：[NVIDIA Run:ai](https://www.nvidia.com/en-us/software/run-ai/)／[NVIDIA Base Command Manager](https://www.nvidia.com/en-us/data-center/base-command-manager/) | Run:aiはGPU分割/Quota/優先/AI Workload/Kubernetes、Base Command ManagerはNode provisioning・監視・Slurm/Run:ai連動・Cloud cluster on-demand。**2026-10-13にRun:aiはOSS化された**([NVIDIA発表](https://developer.nvidia.com/blog/nvidia-runai-is-now-open-source-as-part-of-nvidia-dsx-ai-factory-platform)) | GPU scheduling/GUIを新規製品固有の差と主張しない。EKIは弱い信頼・一時的端末・多種Workload・持ち主拒否権を明示する点を検証 |
| **AI Compute Platform / Multi-tenant GPU cloud**：[Rafay Platform](https://rafay.co/platform/rafay-paas) | Kubernetes、VM/Slurm、GPU/CPU・AI model service、Quota、観測、サービスCatalog、Tenant/SKU、Chargeback。すでに非常に近い競合 | EKIのオープンな外部Provider連携、持ち主優先・安全な端末余力・構造的な事前評価が差になるかは**未実証** |
| **ハイブリッドIT監視/AIOps**：[HPE OpsRamp](https://www.hpe.com/us/en/products/software/opsramp-software.html) | クラウド/オンプレ、GPU/Kubernetes、Service Map、AIイベント分析、運用Automation | 可視化・AIレビューだけでは差別化しない。実計算Resourceの限定提供と配置をできるかが別 |
| **アプリ視点の資源最適化**：[IBM Turbonomic](https://www.ibm.com/products/turbonomic) | Hybrid/MulticloudのApplication/VM/Container/GPU等のResource demand、依存、rightsizing、実自動化とPolicy | EKIはTask受理/Lease/検証・一時参加のTrustに焦点。省資源化・自動配置自体は競合既存 |
| **Endpoint IT資産統制**：[Tanium Asset](https://www.tanium.com/products/tanium-asset) | PC・端末のハード/ソフト在庫、使用状況、運用・Security向けInventory | Endpoints可視化だけでは強い競合。**本人の作業を守りながら使ってよい余力を計算に変える**点を独立検証 |
| **HPC Resource Scheduler／Portal**：[Slurm](https://slurm.schedmd.com/overview.html)、[Open OnDemand](https://www.openondemand.org/about-us) | SlurmはQueue/Reservation/CPU/GPU scheduling/Accounting、Open OnDemandはWeb Portal/Job submission/monitoring。成熟したHPC用解 | EKIは既存Schedulerを再実装せずBroker・限定委任・異種Poolへ接続。Slurm活用前提の顧客に二重Schedulingを強制しない |
| **計算コスト可視化**：[OpenCost](https://opencost.io/) | Kubernetes CPU/GPU/Memory等の利用配賦・Chargeback・On-prem見積 | Cost/Chart自体の差別化は低い。End-user端末を含むAllocation・Job結果・Policy・異種Provider間の価値/影響が仮説 |

**市場見立て**：全部入りの「一元管理」だけでは、複数の成熟競合を同時に相手にする。EKIが最初に証明する差は「異種の資源を**持ち主の実効同意と信頼水準の下で**安全に使える余力へ変換し、**使う価値がない場合には分散しない**と判断し、既存Resource Providerへ委任できること」。GPU FractioningやDashboardなどの重複実装は既存OSS/Adapterを優先。

## 2. 公開された需要指標と、それらが証明しないこと

| 2026年公開資料 | 数字／本文から確認できるニーズ | 根拠の種類と読み違えてはいけないこと |
| --- | --- | --- |
| [FinOps Foundation, State of FinOps 2026](https://data.finops.org/)（Linux Foundation、2026-02-19発表） | 対象1,192回答者。98%がAI支出管理、57%がPrivate Cloud、48%がData Centerも管理対象。「単一画面で異種技術費用を扱いたい」「GPUの細かい稼働とAI費用」「導入前の設計コスト見積」がツール不足の上位 | **FinOps従事者の調査**。企業全体の98%／EKI購入意向98%とは言えない。費用管理と物理CPUの余力活用は別の需要 |
| [Flexera State of the Cloud 2026](https://info.flexera.com/CM-REPORT-State-of-the-Cloud)（2026-03-18、753名） | 73%がHybrid Cloud、推計IaaS/PaaS無駄支出が29%、71%がCCOEを持つ | **対象者の自己推計／Vendor調査**。クラウド費用の29%をEKIによって回収できると主張しない |
| [CNCF Annual Survey 2025/26](https://www.cncf.io/reports/the-cncf-annual-cloud-native-survey/)（2026-01-20） | 回答したContainer利用組織で82%が本番Kubernetes。AI/クラウドネイティブ運用の成熟を示す | k8sは既存前提として適合が必要。K8sの代替Scheduler需要の証拠ではない |
| [VentureBeat Intelligence 2026-09調査](https://venturebeat.com/vb-pulse-infrastructure-and-compute-report) | 企業（従業員100名以上）調査、production AI利用者にGPU idleの自己申告が多い；インフラROI検証も弱い | **媒体調査・サンプル小、自己申告、GPUの利用状況定義は統一保証なし**。全企業のGPUが一様に空いていると推定しない |
| [SB C&S国内販売パートナー調査](https://cas.softbank.jp/information/260423_01/)（2026-04-23、販売パートナー179名） | AI Infrastructure運用人材不足70.4%（複数回答）、顧客向けAI案件の本格導入が進むとの回答は7.3% | **顧客への直接需要調査ではなくチャネル側の認識**。国内SMB全体の割合とは言わない |
| [学術調査: AI accelerator resource sharing, ACM CSUR 2025](https://doi.org/10.1145/3721427) | 共有Acceleratorの公平性・Security・効率の課題を整理。既存実例のGPU利用率25〜50%を引用 | 研究で引用された複数事例であり、2026年の全企業利用率の代表推計ではない |

**需要が確認できた領域**：ハイブリッドIT/AI支出の管理、GPU資源の可視化と組織別配賦、導入前のシナリオ比較、専門運用人材不足への対処。**まだ直接の需要証拠がない領域**：社員PCの余力を集めて一般的な業務計算を大量実行したい企業の割合、Raspberry Pi群の総経済性、EKIの持ち主優先GrantやSimulationが購買判断をどの程度変えるか、TEIベース管理コンソールへの支払意向・単価。

## 3. 「理念×需要」で狙う初期ターゲットを具体化

| Segment／Buying center | 現在の痛みと成約仮説 | 市場競争・技術制約 | 第1目標 |
| --- | --- | --- | --- |
| **S1 社内AI/GPU混在Pool**（AI Platform/Infra Lead、CIO、FinOps、情報Security） | 複数部門に別々のGPU/CPU Server、オンプレ＋クラウド、空き・予約・使用と予算が別々で、待ち行列と費用の見通しが悪い。組織内Resourceの**現状可視化→安全な未使用分活用**を一続きに提示 | **最優先**。Run:ai/Rafay/IBMと直接競争。独自Kernelの完成前に既存Scheduler/AgentのData接続で検証 | 3種類の資源（GPU/RAM/CPU）＋2つのPool＋CloudのInventory/Capacity/Forecast/Reservationを、出典/精度つきで一画面に表示。安全な候補操作1件 |
| **S2 企業端末・研究室の一時リソース**（IT Operations、Endpoint Security、R&D） | 端末の稼働状況を守りつつ、夜間/遊休時間に切断可能なJobを受けたい。買い足す前の選択肢 | 社内PCへのAgent導入・監視・業務割込み/機密・電源コスト・反射的停止が障壁。ITAM競合とは異なる利用許可を提供 | まず管理者が許可した小規模・非機密Task。Local Governorによる即時停止と資源取り戻しの実機 |
| **S3 オンプレ小型／Armノードとクラウドの混在**（Engineering/R&D/Edge Operations） | 大きな設備がない現場で、小型CPUノードや省電力機を安全に借用し処理を分けたい | Arm64/Linux/Memory制約、Taskのバイナリ適合、通信量/電力/管理コストで転送負けも多い。GPU/LLM Trainingに無理をさせない | 低依存・分割可能・再試行安全なJobでCluster化。Cloud/CPU既存実行との費用・所要時間比較 |
| **S4 TEI生成Application配備管理**（App Platform/SIer） | TEIから生成したService/Batchを、単一画面で既存Cloud/On-prem実行環境にDeploy/Observe | **EKIが高可用Serviceの状態正本を取らない**（EKI ADR-0013・#130）。Kubernetes/VM/既存Hostの成熟機能を利用、Data PathをEKI経由にしない | EKI Console経由でCloud Managed Targetへ配備Candidate/Revision/Resultを提示し、実Serviceは既存Host Authority。止められる背景JobだけEKIにResidency委譲 |

**推奨初期事業仮説はS1**。S2はEKIの持ち主優先哲学との一致が強いがAgentやSecurityによる導入摩擦が高く、最初の有償買い手に選ばれるかは未検証。S3は技術上の実証に向くが、単なる安価なCPUの寄せ集めでCloud代替を保証しない。S4はTEIとの相乗効果が大きいが、EKIが新しいKubernetes/Container PaaSを一式開発しない。

## 4. 差別化は「見える→判断できる→許された余力を使う→価値を測る」

**4段階に分けた顧客が買う機能**：

- **Visibility (Read-only)**：CPU/GPU/Host RAM/VRAM/Node/Pools/Lease/Actual/Reserved/Capacity/Cost/Locationを、同じ単位・Resource Class・Scopeで表示。Tanium/OpsRamp/OpenCost/Run:aiが既に提供する項目について差別化と主張しない。強み候補は**異種Provider/一時参加とCurrent Authorityも同時表示**。
- **Explain/Plan**：何故そのJobが待つのか、誰がなぜ余力提供を許したか、今の有効な余力と予測の違い、Data transferと検証/電力/管理費を含む「使う価値があるか」をシミュレーション。FinOpsの事前費用比較ニーズと直接対応。
- **Safe sharing (Opt-in)**：管理者登録Delegate/Poolに限定し、OwnerのLocal Governor拒否を超えず、承認済みTaskをEKIが既存Provider/Kueue/Slurm/Cloud/Agentへ安全委任。購入者にとっての差別化は利用率だけでなく、**使ってはいけないResourceを使わない保証**。実行Credential/実機Fencingは別の検収。
- **Value proof**：独立実測で、承認済みTaskの**到達時間・完了時間・完了数／GPU-hour/CPU-hour当たり成果／回収時間／運用工数／中断影響**を記録。企業価値とConservation（人の仕事の妨げ・情報漏えい・誤配置）を同じグラフで説明。E2Eなしにコスト削減率を謳わない。

**運用UIの実装者としてTEIを活用**：TEIはData/Graph/Rule/Flow/Modeler/SHO/AI reviewを一体に提供するが、EKIの観測が直接GrantやPerformance Truthを発行するわけではない。AIはSimulation/Change Impactの反証案を作成し、EKIのCurrent Admin Policy・Local Governor/CASを上書きしない。管理ConsoleはCloud上でもよいが、Key/Lease/Kernelsの起動をTEI生成Admin UIに依存させない。

## 5. 配置と費用：EKIを介することで遅く・高くなってはいけない

**三層を分離**：
1. **Control/Console（安定Cloud可）**：EKI Admission/Placement/Lease/Policy/Observability（実Authority正本はEKI側）、TEI管理画面/Simulation/Change Review。SaaS化・マルチテナントは新Security Reviewが必要。
2. **Execution Provider**：既存Kubernetes/Cloud managed service、オンプレSlurm、既存Host、承認されたローカルPC/Arm/Linux Worker。長期HA・業務状態正本をもつServiceは**既存HostがAvailability Owner**（EKI #130・ADR-0013）。EKI管理ConsoleがServiceのDeployを制御する場合も、実Service Traffic/Database/Data PlaneをCentral EKI経由には流さない。
3. **Data Plane/Network**：ArtifactはWorker↔認可されたObject Storage/site cache等の直接/制限経路を基本にし、Control Planeを大量データのProxyにしない（EKI Control/Data Plane）。Local Workerはoutbound-first。Worker間の任意P2Pを標準化しない（EKI Deployment Profiles）。EdgeやRaspberry Piは秘密Key、OS証明、十分なメモリ、電力/熱/帯域を個別確認。

**資源利用判断の対照式（計測項目）**：
`Benefit = (local-or-existing baseline total cost / completed output) − (EKI-selected total cost / completed output)`。
Total CostはCompute（Cloud/On-prem amortization）＋Data Egress/Ingress＋Control/Ops＋Energy＋Retry/Verification＋Idle reservation＋Failure/Recovery。Output quality／deadline/SLOが一致しないJobの単価比較は不可。配布・監視のオーバーヘッドと追加障害経路を監査する。数百ミリ秒以下の小さな仕事を一件ずつ外部Submitするのは不利な可能性があり、EKIの[投影Spike #106](https://github.com/hat47x/eki-fabric/issues/106)では5〜11msの決定的TEI操作に対しTask API受理の往復だけで中央値10.2msとの観測。**直接実行／Session／Bundle化の選択**と #58 の`FABRIC_NOT_JUSTIFIED`を入口に置く。

**サービス配備と背景計算は別**：顧客向け常時稼働API、厳密SLO/DB正本/長期SessionはCloud/既存Host。止められる推論バッチ・資料変換・テスト・並列数値計算はEKIと相性がよい。すべてをEKI Workerへ常駐させたり、極端に小さなArmノードに分割してネットワーク通信量を上回る収益が出る前提にしない。

## 6. 顧客発見と製品判断を混ぜないGate

**需要証拠 → 支払者/意思決定者の特定 → 既存製品比較 → 有償Pilot → 広域展開**の順を守る。

| Gate | 必要な事実／独立Expected | 止める条件 |
| --- | --- | --- |
| **M0 競合・市場（今回）** | 数字は出典／対象母集団／時点を固定し、既存Productの比較表。市場が空白ではないと記載 | 29%無駄支出→EKI29%節約、FinOps98%→全企業の導入需要、と推測 |
| **M1 顧客発見** | S1に属する企業のAI Platform/SRE/FinOps/Endpoint Securityの担当者に、既存GPU数・Provider/Queue・空き予約・困る意思決定・既存ツール・購買主体をヒアリング。顧客固有値は非公開 | 「見える化は便利」という反応だけで予算/購買意向を推定 |
| **M2 Read-only pilot** | 2つのResource Provider、実GPU/RAM/CPU、予約/実測/未知、部門別Consumption/Policy、時間差を表示。既存Grafana/Tanium/Run:ai等と比べて手間・洞察を測定 | 正確でないInventory、開示越境、既存ツールを組み合わせる方が安い |
| **M3 Simulation/Value** | 同じ実WorkloadでDirect/Cloud/Current SchedulerとEKI CandidateのCost/Latency/Deadline/Local-interferenceを、独立ログで比較。1件の安全なWhat-if変更 | Simulation PASSを実Savingsに格上げ、Local Ownerの利用拒否を超える |
| **M4 limited execution** | 非機密の分割TaskをEKIへ登録し、実Local Governor/Task/Lease/Receipt/Providerを書き分け。外部Data Planeの費用も記録 | MultiWorkerの二重配置、Unknown Side Effect、旧Worker再登場、費用逆転 |
| **M5 有償判断** | 導入主体・財務承認者のPilot有償化/継続意向、短期導入費・運用工数・差別化の費用対効果、販路・継続率 | 競合の無償・既存保有機能で需要を満たせる、セキュリティ導入費が利益を上回る |

**North-star候補**は「許された資源で、顧客のSLOと安全性を守りながら、追加設備なしに完了させた仕事の価値」であり、単純なGPU稼働率・総登録ノード数・総転送byte数ではない。KPI例：`verified useful work per constrained resource-hour`、`avoided spend (independently baselined)`、`p95 time to first successful output`、`human-interference incidents=0`、`unauthorized allocation=0`、`unknown-effect double retry=0`。

**未確定の投資判断**：市場全体規模、価格/契約形態、Run:ai OSSとの差、GPUメモリ共有・低価格ノードの経済性、Cloud Control SaaSの法規/データ所在、非TEI ClientとのInterop、顧客の既存クラスタ移行負担。実需要を顧客ヒアリングとPaid Pilotで測るまでは`product-market fit achieved`としない。

### 6.1 顧客価値を資源群・組織単位で証明する追加仮説（2026-11-03）

[TEIがAI Jobを構築、EKIが資源群へ委任する設計](TEI_AI_WORKLOAD_EKI_AGGREGATE_DELEGATION_2026-11.md)を受け、**EKIは個々のGPU/PCを管理する新製品である必要はない**と捉える。購入主体は一つ上のAI Platform／IT/Infra Governanceであり、複数の既存GPUクラスタ、Cloud Account、Managed Endpoint Poolを**同じ実権限と実費用に照らして委任できること**を差別化候補にする。Slurm/Kueue/ITAM/MDM/EDRを置換する管理ソフトの開発コストを避け、必要なProvider AdapterとTEIベースのSimulation/運用を価値の核にする。

顧客発見では`Pool/Orgレベルの利用可否判断に困っているか`、`既存製品のライセンス/運用負担を下げるか`、`Nodeの詳細をEKIに渡さないデータ主権・Security上の価値があるか`、`TEIのAI Pipeline設計と一体の購買意思があるか`を別に検証する。**「国内企業は余剰PCを多数持つ」から`動的PC Cloud`へ有償の需要があるとは推定しない**。Clientが承認済Poolを指名する方式と、EKIが最適Delegateを提案する方式の双方を、現行EKI RQ-059と照合する。経済圏におけるContributionや金銭的価値への換算は別の制度・Ownerが決め、EKI Coreが通貨や市場を構築しない。

### 6.2 最初に購入価値を判定する実証は、Pool選択より前のJob Wireと予約

[TEI/EKIの未解決契約と独立反証](TEI_EKI_AGGREGATE_DELEGATION_UNRESOLVED_GATES_2026-11.md)を補助判断票にする。顧客に`全企業GPUが見える`と言えても、実TEI Jobの入出力Artifact・EKI #112のAPI Gap、下位Providerの予約と二重Retry、Owner/EDR/MDMの現在性が未検証なら安全な共有製品としての価値は証明できない。Pilotでは**既存Provider直接利用 vs EKI Broker介在**を同じWorkload/結果で測り、Task完了率・p95待ち時間・費用/完了件数・実User干渉・Security incident・権限/支出への影響を別の軸で評価する。過剰なPool inventory収集、未承認GPU連携、組織間Credit制度を初期MVPの必須前提にしない。

### 6.3 顧客の購買判断につながるのは「期限と回復リスクを含めた予算内の仕事完了」

[期限・実績・予算・Recovery Coordination](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)を市場検証の差別化候補へ。GPU使用率や見掛けの空き台数より、**同一品質のJobについて、許容リードタイム内にVerified Resultを得た割合、失敗後の回復時間、実際の物理消費・外部料金、不要なCloud引上げの抑制、運用負担**を、Direct/現行HPCとEKIで比較する。現在のEKI要求RQ-057は期限内完了を保証せず、予算内での引上げと期限前のリスク通知を求める。価格換算はCore外で行い、独自通貨／無条件の金銭的最適性をプロダクトの既定機能として主張しない。

## 7. Issue・ADRとの接続（重複開発を避ける）

- **EKI側で維持**：#58 Suitability、#112 External TEI projection、#57/#242 Task/E2E、#130 stop-safe resident、#106 Session/小Task、#222 Worker identity/onboarding、#109/#125/#140 resource pools/economics、#19/#48/#50 simulation、#354 concurrent reservations、#118 CPU/GPU/Local Observability、#124 human transparency、#136 TEI plugin comparison。EKI repo/ADRの正式変更はEKI Owner判断待ち。
- **TEI側で維持**：#1016 Application Composition、#1121 Source→Target Lowering、#1130 Batch/Run、#1105 Build artifact、#1043 Web Host/Auth、#2499 delegated business authority、#1124 Modeler、#2497 projections、#2514 Oracle、#2620/#2621 Data/Role/Authoring、#1174 Effect/Receipt。TEIとEKIの第二Host/Registry/Resolverを作らない。
- **本稿の変更**：Draft文書の市場・製品仮説のみ。実顧客発見、購買意向調査、Source/Model受理、EKI管理API、新しいAdapter/Graph UI、Simulation E2E、安全Conformance、収益実証は**NOT RUN**。

## 8. 根拠の種類と明示的な限界

- **公開市場データ**：FinOps Foundation／Flexera／CNCF／VentureBeat／SB C&S。調査対象と自己申告性、Vendor利害を明示。
- **競合機能**：各社公式Web（2026-11-03閲覧）での**提供表明**。製品の独立実機・価格・SLA・競合比較ベンチマークを行ったものではない。
- **EKI/TEI現況**：接続GitHubの現行Issue/設計文書。採用済みRequirement/ADR、Spike/Reference、Issue Open/Gapsを区別し、EKIのCloud Console／TEI Host Launch／GPU hardware controlが製品実装済みとは推論しない。
- **提案と理念**：Ownerが将来採用する候補。易の卦の特定や将来の市場成功を外部調査から証明するものではない。
