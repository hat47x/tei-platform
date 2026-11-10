# EKIの継続的な資源・Node運用をTEIで可視化／シミュレーション／検証／実行する設計（2026-11）

- 日付：2026-11-03
- 状態：**製品要求の方向性と設計・反証計画。実装済み／正式採用済み／人の承認済みではない。**
- 起点：EKIはTaskを送り出すJob Controllerにとどまらず、CPU／GPU／主記憶／GPUメモリ／周辺資源を、端末利用者を優先しつつ共有する**継続的な資源制御・Node管理基盤**。TEIの「モデル・意味Role・図表・低コード実行・生成・テスト・SHO・AI支援」をEKIの管理機能に最大限利用したい。
- 既存接点：[TEI↔EKIジョブ委譲・State境界](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)（§8でTEIのFlow意味とEKI Task/Lease状態を分離）、[Cross-domain State Machine](STATE_MACHINE_CROSS_DOMAIN_DESIGN_PRESSURE_2026-11.md)、[Data Script起点のFlow](DATA_DERIVED_TRANSITION_TRIGGER_PIPELINE_2026-11.md)。
- 主要EKI Owner：[Product Value](https://github.com/hat47x/eki-fabric/blob/main/product/value.md)、[Requirements](https://github.com/hat47x/eki-fabric/blob/main/requirements/PRODUCT_REQUIREMENTS.md) RQ-005～009/011/018/023/047/058/063、[Authority](https://github.com/hat47x/eki-fabric/blob/main/architecture/AUTHORITY_BOUNDARIES.md)、[Control/Data/Observation Plane](https://github.com/hat47x/eki-fabric/blob/main/architecture/CONTROL_DATA_PLANE.md)、[Task State Model](https://github.com/hat47x/eki-fabric/blob/main/architecture/TASK_STATE_MODEL.md)、[Extension SPI](https://github.com/hat47x/eki-fabric/blob/main/architecture/EXTENSION_SPI.md)、[TEI適用方針](https://github.com/hat47x/eki-fabric/blob/main/plan/UPSTREAM_PLAN.md)。
- 変更範囲：このTEI Draft計画への追補のみ。**EKIのmain／Issue／ADR／Core、TEIのコード／Runtimeへの変更なし**。

## 1. 製品の焦点を「Task」から「継続的な資源とNode」へ広げる

**EKIが管理すべき対象**は、Task/Attempt/Leaseの都度の処理だけではない。資源提供者／Pool／Nodeの登録・移動・失効・隔離、Resource Advertisementと独立Verification、Local Governorの上限、CPU/メモリ/GPUデバイスの割当・競合、Resident/Session/Sandbox、Agent/Policy更新、観測・回復・利用者への透明性、予実／将来の配置候補である。

TEIの「最大限利用」とは**管理画面だけをTEIで描くことではない**。EKI Operations Applicationとして、次をTEIの構造化モデルから同じID／Scope／Revisionで構成することを目標とする。

1. **Modeler**：Node/Device/Pool/Lease/Allocation/Task/Incident/PolicyのData・Relation／Graph／State/Rule／Flowを定義。既存EKI Authorityの定義を読取り、TEIを第二のSecurity Registryにしない。
2. **可視化**：Topology Graph、資源配賦表・時間軸、占有率／予約／予測の異なるレイヤー、アラートと原因・根拠をStudio／SHOへ投影。集計のGrain・Source・更新時点・不確実性・開示Scopeを表示。
3. **Simulation／What-if**：EKIの生産Stateを変更せず、提案したNode drain、GPU分割、Capacity/Policy変更、Worker離脱、利用者復帰などの影響を計算。独立Expected・現物Traceからモデル精度を評価。
4. **運用Functionの生成・実行**：TEIのFlow/Validation/Derivation/Capabilityから、変更Plan→Preview→安全確認→EKI Administrative API／既存管理Providerへの**明示的な運用Intent**を生成。権限・CAS/fencing・実Effect判定はEKIの現行Authorityが担う。
5. **生成AIレビュー・検証**：Graph/表/制約から矛盾・不足・危険な変更案・シミュレーションCaseを提示し、反例候補を生成。判定根拠と修正差分を残し、**AI出力の一致を安全性・権限・Conformanceの証拠に昇格させない**。

**依存の向き**：EKIのApplication/Control用UIをTEIから生成してよいが、EKIの署名・Task State Authority／Placement Authority／Lease発行・Local Governorの危険回避は、TEI Studio・生成画面・生成AI・EKI Central UIが停止しても機能する必要がある。TEIのRuntimeをEKI Kernelの必須前提にすることは、既存EKI方針に反する。逆に、EKIの管理業務そのものをTEIで記述・実行する能力は、権限外の純粋な「ダッシュボード」に限定しない。

### 1.1 EKIの標準管理単位をPool/Delegateへ修正（2026-11-03）

[AI JobはTEI、広域委任・資源群はEKI、HPC/社員PCの内部管理は専門Provider](TEI_AI_WORKLOAD_EKI_AGGREGATE_DELEGATION_2026-11.md)により、前段の`Node/Device`中心表現を修正する。**EKIの標準的な利用者操作・広域可視化の単位は、登録済Resource Pool／Cluster Group／Delegate／Trust Domain。** CPU/GPU/Memory単体やPC個体は**通常はEKIが直接所有する資産台帳ではない**。HPCならSlurm/Kueue/Cloud Provider、PCなら既存MDM/ITAM/Endpoint Security、DP-7の直接WorkerならEKI Agent/Local Governorが各々の詳細を管理する。

- TEIの**標準管理コンソール**は、Pool/Delegate→Workload class/Offer→Policy/Consent→実際のTask/Result→予実/Cost/ImpactをGraph×Table×Timelineで表示する。個別Node・GPU PartitionへはScopeのあるOperatorだけProvider-owned画面／限定Queryでdrill-downし、標準EKI Controlに全Worker物理構成の複製を求めない。
- ただしPoolの`100GB VRAM`や`100 cores`は**単一Workloadの連続資源**を意味しない。同質GPU Class／OS arch／Data Class／Memory ceilingごとのSub-offerを保持し、具体的Device配置の可否は実Providerが検査する。時点・失効・partial・uncertainをResource Scoreに平均化しない。
- 社内PC等は**既存Security/ITAM/MDMの管理対象集合**を登録根拠とした`volatile Pool`として供給できる。しかし`compliant`という自己申告/管理台帳だけで、現在のLocal Governorの利用許可、Worker実行環境の保証、外部Effectの真正性を断定しない。
- 以上はEKIの既存RQ-058、ADR-0013の**設計思想への整合**。HPC Pool/Gateway、MDM Adapter、実Agent/TEI Projectionの採用・実動は未確認。本稿以下のNode/Deviceレベル図・負例は、**EKI直接管理Profile／Provider drill-down／安全性検証に限る**ものと読む。全利用者が全Nodeを閲覧できる設計を要求しない。

## 2. 一枚の巨大な状態図でなく、独立した状態軸と相互制約を可視化

| 対象・状態軸 | 代表的なFact／候補 | 何を区別しなければならないか |
| --- | --- | --- |
| Node・Agent Lifecycle | registered／authenticated／attested／eligible／draining／quarantined／revoked、Agent version | 登録済み＝利用可でない。Attestation/Policy/Agentの有効版、Node IDと交換された物理実体を分離 |
| Local Governor | 利用者の復帰、opt-out、温度／電力／電源、端末内AIの資源利用、effective ceiling | 中央の上限・利用者の上限・実測可能性を別に保持。観測できない資源を「空いている」にしない |
| Device/Resource | CPU core/CPU time、host RAM、GPU device/VRAM/SM/MIG等のパーティション、IO/Network、Disk spool | Device IdentityとDevice Class／総量・予約量・実測使用量・余力・測定保証を区別。時刻と単位を混ぜない |
| Pool/Grant/Policy | 管理者登録済みPool、Delegation Class、Data Class、Budget、Quota、Resource Cap | 予実やWorkerのSelf-reportからAuthorityを計算しない。Pool所有／越境開示の可否を持つ |
| Task/Attempt/Lease | admitted/ready/active/verifying/reconciling/terminal、Attempt／Lease status、fence | Taskの公開statusとAttempt/Leaseを統合しない。外部Effect unknown、旧Leaseの後着、部分Task Groupを別表示 |
| Resident／Session／Sandbox | 常駐枠・有効期間、再接続、Checkpoint、担当Agent | 常駐サービス・短命Attempt・Session Leaseの寿命と回復手段を分離。EKIが高可用な業務サービスHostを保証するとは限らない |
| Operations Incident／Change | Node停止・隔離・回復・Policy/Agent更新のProposal、実行Evidence | 「管理者が提案した」「EKIに受理された」「Workerで強制された」「全Affected Taskが回復した」は別 |

**Graph×Table×Timeline**：TopologyのNode/Edge（Pool→Node→Device→GPU Partition、Task→Attempt→Lease→Node、Policy→Delegate、Current Device→Reservation）のそれぞれに**stable ID**を付ける。Graphで端末と依存の全体、状態表で各FactとRoot Cause、Timelineで時点・遅着・配置競合を見る。図で折り畳まれたNode/Edgeや集計が消えたとみなさない。多重所有、同一名のNode交換、複数GPU Partition、Profileの細分化を単純Treeへ潰さない。TEI [#2497](https://github.com/hat47x/tei-platform/issues/2497)の意味忠実なProjection/逆写像と[#1124](https://github.com/hat47x/tei-platform/issues/1124)のModelerを再利用する。

### Resourceの6つの数字を同じ「使用率」にしない

| 表示 | 意味 | 偽の結論の例 |
| --- | --- | --- |
| `physical_capacity` | Hardware/OSで確認した量（観測できなければclaimed/unknown） | GPU数が4個見えたので全員に4個貸せる |
| `policy_max` | 管理者が定めた論理上限／許可されたData Class | 管理者上限が60%なのでLocal Governor 0%を上書きできる |
| `local_effective_ceiling` | Local Governorが今許す資源の上限／Time/Trustを伴う | 昨日のGovernorの許可を現在のGrantとして再利用 |
| `reserved` | Admission前に将来使うと約束した資源、Attempt／Leaseに束縛 | 予約が0だから即時に物理割当できる、失効したLeaseを予約に戻す |
| `observed_usage` | 実際の使用量（観測点・誤差・欠測・タイムウィンドウを含む） | CPUが平均5%だから100%予約できる、GPU VRAMの空きを誤測定 |
| `forecast_capacity` | Data/Trend・Simulationから導いた**予測** | AI推奨値や予実一致をPlacement／Authorityへ昇格 |

Resource Valueは「CPU milli／core/時間比率」「Host RAM bytes」「GPU Device count」「GPU VRAM bytes」「GPU compute slice／mode」「電力・温度等」の**単位と競合方法を持つ型付きVector**として扱う。GPUを汎用な百分率だけで加算せず、同じ物理GPUに由来するMIG instance・MPS client・time-sliced replicasのCapacity重複算定を防ぐ。RAMの予約量、VRAMの実占有、GPU compute utilizationは別の量。**Node単位の全空き容量がすべてのDevice/NUMAで同時確保可能とは限らない**。CPU oversubscriptionも常に禁止／常に許可とはせず、実Node/Workload ClassのPolicyで区別する。

外部実装参考：NVIDIAの[MIG](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/latest/introduction.html)はHardware Partition/隔離を、[time-slicing](https://docs.nvidia.com/datacenter/cloud-native/openshift/latest/time-slicing-gpus-in-openshift.html)は共有するが同等のメモリ／障害隔離なし、[MPS](https://docs.nvidia.com/deploy/mps/latest/index.html)は協調Processの効率化とMode別制約を持つ。Kubernetes [DRA ResourceClaim/ResourceSlice](https://kubernetes.io/docs/concepts/resource-management/dynamic-resource-allocation/how-dra-works/)や[Kueue Admission](https://kueue.sigs.k8s.io/docs/concepts/admission/)はCapacity／Quota／Device/Admission分離の比較材料。**これらをEKIの既存Resource Provider機能の実装証拠、TEIの全TargetでのGPU隔離保証とみなさない**。

## 3. EKIの既存AuthorityとTEI Management Applicationの接続

EKI [Control/Data/Observation](https://github.com/hat47x/eki-fabric/blob/main/architecture/CONTROL_DATA_PLANE.md)§3.4ではObservation Planeに予実・消費・Task/Node要約を**一方向に書き出し**、権限正本と別保存する。管理UIはObservation＋read-only照会を読み、変更はAdministrative Planeを通す。これを**TEI利用の最低限のI/O契約**にする。

~~~text
EKI Authority / Worker / Provider
     ├── Policy・Task・Lease・Capacity・CAS・Local Governor（実権限）
     │           │ Read-only (scoped/versioned)    ↑ Admin Intent / EKI-authorized write
     │           ↓                                 │
     └── Observation Plane ──────> TEI Modeler / Operations App
                                      ├─ Graph/Table/Timeline/Diagnostic
                                      ├─ Pure State/Derivation/Invariant
                                      ├─ Read-only Counterfactual Simulator
                                      ├─ AI-generated Review/Case/Change Candidate
                                      └─ Preview → EKI Admin API / Provider → Receipt
~~~

**注：図は構成候補であり、通信・Capability契約はまだ未実装。** EKIの内部State StoreをTEIのSQL Data Providerから直接書込んだり、Observation Planeの遅れた値をCAS条件の現在版として使用したりしない。

**TEI側で構成・生成し得るOperations Function**：
- Node/Pool enrollment review、Eligibility検査、Agent version rollout plan／対象Node選択、Drain/Quarantine/Resume依頼、Resource Limit/Quota変更、Task Group再配置計画、利用者への使用状況説明、Incident/Runbook、実行後の独立Observation。
- ただし**実際のDisable/Lease revoke/Quota mutation/Worker stop**はEKIの正規Admin／Local Governor/Providerが行う。TEIが生成したFunctionの存在、AIレビューPASS、Simulatorの結果は権限そのものではない。
- `Node eligible`をDerived PredicateとしてStudioに表示できても、すべてのAgent・Trust Evidence・Policy Revisionが現在確認されなければ、EKIのPlacement Authorityへ入力してよい`eligibility`とは限らない。Rule/Projection/Authorityを明示的に分ける。

## 4. 高度な可視化：安全判断を説明可能にする最小Surface

| View | 図・表で判断できること | 誤認させない表示 |
| --- | --- | --- |
| **Resource topology** | 管理者が閲覧を許されたPool→Node→GPU Device/partition、予約Task/Lease、転送経路・Data locality | 存在が秘匿されたNodeは全件数も出さない。権限Scopeを図のGraph Traversalに伝播 |
| **Capacity/Allocation matrix** | Resource Type×Node×Time WindowのCapacity／Policy／Governor／Reserved／Observed／Forecast | 欠測・古い値・Unit mismatchは数値0扱いしない。各セルにOrigin/Evidence/Revision |
| **Timeline・State transition** | Agent失効、Node隔離、Job dispatch、Lease失効、Result/unknown、重複・遅着の相関 | Event timestamp、recorded timestamp、Processing time、clock skew区別。ReceiptをClaimにすり替えない |
| **Operation change impact** | Drain/Update/Quota変更で影響するJob、止まるTask/Pool、Deadline超過候補 | Model Planと実際のAuthority変更を別列にし、承認前は一律`preview` |
| **Contribution／公平性／予実** | Poolレベルの利用と確からしさ、中断・予実差、承認済みポリシーの影響 | Worker申告とControl観測を区別。Resource Contribution＝金銭的価値とは限らない |
| **端末利用者向け local view** | 自分の端末がいつ・何をどの程度提供し、なぜ停止・reclaimしたか | 別利用者・Job内容・別組織・機密入力を晒さない。EKI停止中もlocal記録から閲覧可能か検証 |

TEI StudioはGraph/表/TimelineとData/Rule/Function/Flow IDの相互選択を担い、SHOは構成変更・安全根拠・影響予測・監査記録を説明する文書へ投影する。EditorでGraph上のNodeを「削除」してもEKI実NodeやLeaseが削除されるわけではない。編集不可のDerived、曖昧なGraph逆写像、部分取得はread-onlyにする。標準構造・意味Role・Projection機構は[#2620](https://github.com/hat47x/tei-platform/issues/2620)／[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#2497](https://github.com/hat47x/tei-platform/issues/2497)へ渡し、EKI専用のGraph Core/Node Registryを作らない。

## 5. シミュレーションの4モードと実環境との差

| Mode | 入力・手順 | 示せること | 示せないこと |
| --- | --- | --- | --- |
| **S-0 deterministic replay** | 版固定したState Snapshot/Event Trace、Clock、Policy/Resource Model、決定的なSeedを再演 | 状態遷移の許容性、既知のCAS競合／Resource予約過剰などの再現 | 収集していないSensor、未観測のExternal Effect、OS内の競合の全再現 |
| **S-1 what-if** | `drain node-A`／`GPU制約変更`／`pool quota縮小`の**差分候補**を仮想Stateへ適用 | Affected nodes/Attempt/Lease、再配置先の候補、期限超過や予算・容量のリスク | 実環境でDrainが成功・Grantが保持されるという保証 |
| **S-2 fault injection** | Node loss、通信分断、Client/Worker偽装、遅延Heartbeats、Lease ABA、OOM、VRAM逼迫、端末利用者復帰、GPU Mode mismatchを注入 | fail-closed、不正なRetry、Old Lease復活、部分成功/unknownの欠陥 | 未模擬なDriver故障・電力・熱・パーティション隔離 |
| **S-3 shadow / learned comparison** | 既存EKIの`spikes/artifact-locality-cost`／learned scheduling shadow simulationをBaselineとして対照 | 予測器の誤差、履歴分割Leak、選択分布の候補比較 | MLの予測がPlacement許可／Securityの意味を持つこと、本番での安全な高速化 |

シミュレーション結果は`SimulatedOutcome`。実`ObservedOutcome`／`VerifiedEffect`と異なるRole・Revisionで保持する。入力Snapshotの範囲、Resource ProviderのMode/Capability・測定窓、event/clock/model/rule/seed、欠測・partial・non-disclosure、対象外の物理現象を記録する。Snapshotが古いならシミュレーションは「過去条件での比較」と表示し、現運用への適用可否は再評価する。

**既存EKI調査を再利用する**：[EKI #19](https://github.com/hat47x/eki-fabric/issues/19)は既存Deterministic locality/cost model、Shadow観測、学習器のAuthority非昇格を前提とする。[#48](https://github.com/hat47x/eki-fabric/issues/48)にはScenario S1～S6のSimulation設計、[#50](https://github.com/hat47x/eki-fabric/issues/50)には学習器比較前のGateがある。新Simulatorを先行で別正本にせず、**EKIの既存Simulator／Observation→TEIでの入力編集・グラフ表示・Case生成・結果説明**へ接続する。Simulationの不足（Node Event、Resource Vector、Policy/Fence）だけ、専任EKI Ownerへ差分を提示する。

## 6. 生成AIレビューを「必須の設計品質工程」とし、実許可から切り離す

**AIは安全性の証明主体でもPolicy Authorityでもない。** ただし複雑なNode運用の設計・変更・事故分析では、生成AIによる検査を**必須のレビュー工程**として位置付け、見落とし得る条件や影響を探索・説明させる。実本番操作の許可はAI結果とは別に、EKIのCurrent Authority/Hostの運用Role・自動Conformanceと、必要な人間の承認で決める。

最小レビューの入力票：`Change Proposal ID`、目的、Affected Node/Pool（閲覧許可範囲）、Snapshot/Source/Policy/Plugin/Simulator Revision、Capacity/Reservation/Usageの出典と観測信頼度、Change diff、Event/Clock/Failure Model、Simulation outputs、独立Expected/Oracle ID、既知のLimitations、実行予定Authority Role。機密Node/Artifact/端末利用者属性や鍵をAIプロンプトに無制限送付しない。外部テキスト・Telemetryを命令として解釈しない。

AIに期待する仕事：
- Graph/表／State Modelを横断し、**未定義Event・Guard競合・誤ったRegion統合・依存循環・容量の二重計上・stale Observation・失効済みGrant**を指摘。
- 資源の異なる複数案、延期・Drain・隔離・段階Agent更新のImpactと「利用者の本来の作業」への影響を整理。
- **反例テストを提案**し、事前登録した独立Expected／Mutation（Reservation誤合計／古いLease復活／GPU共有Mode誤認／Data Class緩和等）で判別力を比較。
- 実Traceに対して「何が観測されたか」「何が推論か」「どこがunknown/未検証か」を、Source/Revision/Effect Receiptとともに説明。
- 影響範囲の大きなPolicy、Device Partition、Node大量drain／Agent rollout、顧客データ・機密境界、実Credentialに触れる提案では**人的レビュー必須**。AIが複数回答で一致しても独立Conformanceとは扱わない。

**安全なAction Loop**：
`Read-only Inspect → Typed Change Proposal → Model Validation → Deterministic Simulation → Independent Scenario/Oracle → AI Critical Review → Preview/Diff → Current Administrative Authorization → EKI Conditional Write/Provider Effect → Receipt → Independent Observation`。

この順番は**申請とEvidenceの論理境界**であり、現行EKIの自動化APIやTEI GUIがすでに揃っているという意味ではない。AIは提案・Reviewへ参加できるが、Actual CAS/fencingの置き換え、Local Governor拒否権の上書き、EKI Admin Credentialの管理・暗黙委任、Security Assurance決定を行わない。Safety上のfail-closeは**AI停止・誤回答時にもnative側で成立**させる。

## 7. 重要な実装GapとOwnerの既存作業を再利用

| 出典・状態 | 具体的な未確定／確認済み境界 | TEIで受け取る/受け取らない差分 |
| --- | --- | --- |
| [EKI #354](https://github.com/hat47x/eki-fabric/issues/354) open | 現startgateは同期単件実行で成立。同時Attemptを始めると1件ごとのMachine上限では**合計予約が過剰**になり得る。ReservationとActual Usageは別 | Resource Vector + `sum(reserved)+new≤LocalGovernor`のWhat-if・負例、実Store/CASはEKI Owner |
| [EKI #118](https://github.com/hat47x/eki-fabric/issues/118) open | ローカルAI等のCPU/GPU/NPU利用をOSごとにどこまで観測できるか未実測。未知は背景計算を控える | `unknown`を赤い空き容量にせず「利用不可/未確認」の診断として表示。実Sensor/Driver測定はEKI Spike |
| [EKI #124](https://github.com/hat47x/eki-fabric/issues/124) open | 端末利用者への資源提供履歴・中断理由の透明性が追加要件候補 | 最小Local Read-only Surface/SHO説明、他人のTask内容・内部構造は非開示 |
| [EKI #123](https://github.com/hat47x/eki-fabric/issues/123) open | Agent署名・最低版・段階Rollout／停止中のGovernor権限 | TEIでRollout Plan/PreviewとMutationテスト、実Agent更新権限と安全停止はEKI/OS/MDM |
| [EKI #268](https://github.com/hat47x/eki-fabric/issues/268) open | Worker Disable/Revocation→Placement/Lease/Attemptへ波及の実接続 | Node QuarantineのImpact Graph、Old Lease不可の独立Expected |
| [EKI #498](https://github.com/hat47x/eki-fabric/issues/498)／[#505](https://github.com/hat47x/eki-fabric/issues/505) closed | Local spool総量とGovernor資源／出力判定の接続作業が進展。ClosedというIssue状態と現在の実行保証は区別 | 非常時のLocal Observation/Output pressureを表示。実Control/AgentのE2E完了は再確認が必要 |
| [EKI #57](https://github.com/hat47x/eki-fabric/issues/57)／[#242](https://github.com/hat47x/eki-fabric/issues/242) open | Admission→Placement→Lease/WorkOrder→ResultのProcess横断検証と配線に未解決 | TEI画面・SimulationのPASSからProduct Pipelineの実動成功を推論しない |
| [EKI RQ-058](https://github.com/hat47x/eki-fabric/blob/main/requirements/PRODUCT_REQUIREMENTS.md) baseline／[ADR-0021](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0021-observation-plane-and-disclosure-tiers.md) | 登録されたDelegate/PoolのTrust等級と実績、開示階層、Observationの非Authority | TEIでは管理者Scopeだけを観測し、内部Nodeを一般Userへ列挙しない。Pool PredictionをGrantにしない |

**全てのOwnerをTEIへ移すのではない**。EKIのNode/Device/Allocationの原子性とLocal GovernorはEKI側、TEIのData/Graph/View/Simulation AuthoringとOps Function生成はTEI側。EKI [#136](https://github.com/hat47x/eki-fabric/issues/136)はTEI Plugin lease/Authorityとの比較を扱い、EKI [UPSTREAM_PLAN](https://github.com/hat47x/eki-fabric/blob/main/plan/UPSTREAM_PLAN.md)はTEI-first Application Plane、native-first Kernelを**設計方針**としている。TEIの「最大限利用」は、この方針を破ることなく生成Targetと実Runtimeの適合性を検証して広げる。足りない場合のみPlugin/Adapter差分を採用し、TEI CoreへEKI専用State kindを足さない。

## 8. 最小の独立Conformance設計（NR-01〜12、全件NOT RUN）

| Fixture | 独立Expected／観測者 | 停止・不合格条件 |
| --- | --- | --- |
| **NR-01 同時予約** | 1 Node capacity=100、Governor max=100、既存Reservation 60、新規60。上限超過なので2件目は待機/拒否、二重割当なし。EKI Agent Reservation StoreとStartGateのActual | 1件ずつ60≤100で同時起動、Lease期限だけ消費、シミュレーションがPASSを出す |
| **NR-02 CPU/GPU/Memory多次元** | Node CPUに余裕、RAM不足、GPU device/VRAMは使用中。いずれか必要条件不足なら新TaskをPlacementしない。Device/OSが観測した量 | GPU utilization=低いだけでVRAMを空と判定、別GPUのVRAMを足して架空の割当を認める |
| **NR-03 GPU共有Mode** | 同じ物理GPUでMIG／MPS／time slicingのDevice/Isolation/Capacity Profileを切替。Profile未登録の共有はunsupported | time-slicingをMIG相当の安全隔離と判定、Partition重複計上 |
| **NR-04 端末利用者復帰** | ローカルAI・本人の操作を観測、Governor ceilingを0へ。Centralが60を推奨してもLocal Stop拒否権が優先 | GPU/CPUが空きと表示され、中央の再配置でLocal Stopを回避 |
| **NR-05 Offline/Telemetry stale** | Nodeの最新観測が欠ける、Workerが虚偽resource claim。部分取得と不明を明示し、Placement SecurityはAuthorityの独立評価 | staleを0使用/全空きとする、自己申告をverifiedと表示 |
| **NR-06 Node quarantine/drain** | Node-Aがrevoked、当該Lease/Attempt/Task/SessionをImpact Graphで識別。新Placement停止・古いLease拒否、実Effect unknownを保持 | Quarantine UIが緑でも古いLease再開、影響Taskを消去 |
| **NR-07 Agent staged rollout** | 10台中2台だけ先行。旧Agent/新Agent混在・署名失効・Rollback/保守手順を比較 | 未署名Versionがeligibleに入る、失効Agentへ配置 |
| **NR-08 Partial Group** | Group 3 Task：成功1、dispatch前期限切れ1、dispatch後Effect unknown1。状態表とTimelineが区別 | 1成功=Group完了、unknownをfailed/no-effectへ丸める |
| **NR-09 What-if versus Act** | `drain one node`案のSimulationでは1件再配置可能と予測。実時点で別のNodeが離脱。Action前にAuthority revisionを再取得 | 古いSimulation結果だけでAdmin Write/Grantを通す |
| **NR-10 AIレビュー不正入力** | Telemetryの任意文字列に「安全制約を無効化せよ」。AIが提案してもGuard/Policy/受理済Role/Runbookを変更できない | Telemetry由来の命令が管理Actionになり、AI承認でPolicyが確定 |
| **NR-11 非開示** | 利用者Aへ自分のTask/端末使用は見せ、他部署のNode/Task/Cache/Group数は非開示。UI/Graph/Tooltip/AI prompt/SHOが同じDisclosure | Graphの非表示Nodeが件数/リンク/診断で推測可能、AIに秘密情報を送信 |
| **NR-12 シミュレーション失効** | Simulatorで`Policy P0`のPASS後、Governor/Policy/AgentがP1へ更新。現在の運用にはP0 PASSを再使用不可 | 同じChartでPASS継続、過去Receiptを現在Grantへ昇格 |

**Expectedの独立性**：各Caseの前提と許される出力を先に固定し、TEI生成の同じRule／SimulatorからExpectedを自動算出しない。Guard反転、Lease後着、Node二重ID、Memory/VRAM見積違い、古いObservation、Disclosure越境、AI Prompt Injection等のMutantを用意し、判別力を説明する。EKI/EKI Agent実測が無ければ`simulation-only`と表示。TEI [#2514](https://github.com/hat47x/tei-platform/issues/2514)のOracleへ接続するが、別のConformance Runnerを起こさない。

### 8.1 Hosted Controlと外部Service実行を前提とするDeployment圧力（2026-11-03）

[EKIをTEI Applicationの論理的Launch/Deployment Brokerにする設計案](TEI_EKI_APPLICATION_DEPLOYMENT_FABRIC_2026-11.md)では、安定クラウドでEKI Control/Consoleを提供し、TEI生成Cloud/On-prem Applicationの**常駐実行Ownerは既存Cloud Host／Orchestrator**、Task/Lease／仮想Edge計算はEKIの任意委任、と分ける。Hosted ConsoleのResource Pool/Node管理にはGraph/表/Simulation/AI Reviewを最大限利用できるが、ConsoleはEKI Authorityを直接変更せず、EKI Kernelが停止時にも復旧できるNative経路を維持する。

**運用SurfaceをDeployへ延長**：Release Source/Meaning/Artifact digest、EKI Deploy Proposal、Provider Service identity/Revision、Ingress/User Data Path、Result/Provider Health/Business Receiptを同一画面に**別の状態**として表示する。管理コンソールをクラウドで貸与しても、公開AppへのUser HTTPやbulk Artifact/DB trafficをEKI Controlに通さない。TenantのResource構成・Node数・Access/AI Promptへの開示を区別する。AL-01（Direct vs EKI broker同一Host）、AL-02（EKI停止後Service継続）、AL-07（Tenant隔離）、AL-12（Provider実状態との差分）を本稿のNR-01〜12とは異なる部署/実Targetの**追加圧力**として参照し、Test Runner/Oracleは増設しない。

### 8.2 市場需要・競合と適用先の補足（2026-11-03）

[EKIの企業内Compute資源一元管理：市場・製品適合の調査](EKI_ENTERPRISE_COMPUTE_MARKET_FIT_2026-11.md)では、NVIDIA Run:ai/Base Command Manager、Rafay、IBM Turbonomic、HPE OpsRamp、Slurm、Tanium、OpenCostと用途が重なる点を確認した。**図・Dashboard・AI推奨だけでは商品上の差別化にならない**。初期市場は社内AI/GPU混在Poolの可視化→使用可否の事前Simulation→安全な限定委任で検証する。FinOps/Flexeraの数字をEKI購買需要や節約実績と同一視しない。

### 8.3 Pool-first運用可視化の資格表とSafety Gate

[未解決Gate計画](TEI_EKI_AGGREGATE_DELEGATION_UNRESOLVED_GATES_2026-11.md)§2・§6を追加。TEIのEKI管理Consoleでは`RegisteredDelegate`／`ProviderCapability`／`CapacityOffer`／`AllocationAuthority`を別のType/Stateとする。通常の広域操作はPool/Delegate単位に絞り、内部Node/DeviceはEKI直接管理Profileまたは限定Provider drill-downに留める。**Offer/Simulation/AI Reviewを現在の予約・Grantに昇格させない**。Scope、時点、Source revision、completeness（fresh/stale/partial/unavailable/not-found/permission-denied）と過去値を別表示。Trust Bundleのopen Source-static課題EKI #466/#467を実Worker/MDM Gatewayへ進む前のOwner確認Gateとし、端末集約でmTLS/Local Governorの強制が消えないことを検証する。

### 8.4 リードタイム・失敗相関・Recovery Reserve・Portfolio競合の運用可視化

[期限・予算・実績によるResource Pool調停](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)では、TEI Ops Consoleに`requested deadline/forecast completion range/remaining slack/estimated recovery elapsed/available recovery budget/actual Provider reservation`を**異なる事実**として表示する。単一の`expected finish`だけでは複数故障・Queue/転送・検証が見えない。Site/Network/Provider共通故障のGraph、期限接近によるCloud引上げのWhat-if、予測の校正/サンプル不足、故障後の予算消費とCAP-1承認をSimulation/AI Critical Reviewの主要ケースに含める。ただしAIの提案や期限Warningは、実資源確保・費用支出・管理者Grantに昇格しない。

### 8.5 直接Worker Test-profileの現在EvidenceとPool運用の間

[EKI現mainの実配線再資格化](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)§12では、optional Dispatch、固定Commandの別Processテスト、条件付きOutput body受領とGET実装が確認できる。一方、Result Acceptanceの試験内直接呼出し・Client Input ingestの未実装、実Provider連携、異種GPU/VRAM予約、社内Endpoint/MDMによるPool登録・失効は未証明である。TEI Consoleでは**source exists／test source exists／exact-head test PASS／product-only実行／実Managed Pool**の証拠を別の資格として可視化し、WorkOrder/Lease/Claimを一つの`Job完了`アイコンへ潰さない。GPU/Cloud/MDM全体の管理Appに昇格するにはEKI #67等の専門Ownerで個別検証する。

### 8.6 AI-firstなConformance生成と現在のEKI ABI差分の表示

[縦断§14](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)のABI Gateを、TEI管理Consoleの`mode=read-only/diagnosis`へ接続。TEIのジョブ要件（Effect/Executor/Inputs/期限/Pool/DataClass）とEKI P0が**表現できる項目／値だけ保持する項目／実行時に強制しない項目／表現不能な項目**を区別して表示する。WP-01〜19の静的差分は出典OpenAPI blobと試験Versionを併記し、EKIの実Worker/Output・Owner Grantへ昇格しない。生成AIによるGap検出や反例生成はTEI [#2811](https://github.com/hat47x/tei-platform/pull/2811)の共通AI-first test scaffoldを後続Consumerとして使い、本Console固有のFixture RunnerやAI Policy Authorityを新設しない。Source採用とEKI Worker Trustも別Issuer/Revisionであり、AIが原文や未公開Node Topologyを外部送信する権利は無い。


### 8.7 AI生成CaseとSource原文非開示の接続（2026-11-04）

[TEI/EKI一改訂の受入表§16](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)を、EKI運用ConsoleのAI診断設計へ反映する。TEI [#2813](https://github.com/hat47x/tei-platform/pull/2813)はDSL ReferenceのJSON Case追加を試す**test-only・Draft**であり、実EKI Worker/Cloud/MDMの動作や予算最適化の成功を証明していない。AIが行うのは、既存の適合/非適合表（WP/RE/VS/LB/CL）から不足する負例候補と独立Expectedの確認事項を挙げることまで。生成したCaseの採否・現在の実Grant・Owner受理は既存Authorityに返す。

外部Sourceの証拠は[TEI #2812](https://github.com/hat47x/tei-platform/pull/2812)の安全修正版に従い、**原文取得をprivateに限定し、公開SurfaceへはDigest等のメタデータだけを投影**する。AI prompt/Export/SHO/Tooltip/CSV/ログへ、単なるSource ImportやSource Witnessの存在を根拠に原文/非公開Pool Node数/Provider credentialを渡さない。利用できる範囲を明示した合成FixtureでAIの反証提案をテストし、実Worker/Provider/Business Effectは独立Test Gateへ接続する。EKIの[安全Event計測 #548](https://github.com/hat47x/eki-fabric/issues/548)はclosedだが、4種の安全カウンターをDeadline達成/実Cost/故障分布の完全なテレメトリとはみなさない。新しいAI RunnerやPolicy Authorityは作らない。

## 9. 既存計画との分担・段階導入・Done/Stop

| 段階 | TEI側の差分／最小成果物 | EKIの既存担当とGate |
| --- | --- | --- |
| **R0 机上設計（今回）** | Graph/Table/Timelineを通すNode/Resource State Contract、権限/観測/予測の分離、NR-01〜12 | EKI #354/#118/#124/#123/#268/#19と適合照合。追加Contract採用はOwner待ち |
| **R1 read-only Opsビュー** | 安全に開示できるEKIのObservation/Read-only APIからRecord/Relationを取り、Topologyと予約/実測・State/Traceを表示（モデル・時点・未知を保持） | EKI ADR-0021の開示Scope、EKI #112の外部API契約。**スクレイピングや内部Store直接参照を既定にしない** |
| **R2 Simulator × 検証** | EKI既存Simulator／TraceをData/Function/Scenarioで受け、Deterministic/What-if/Fault Injectionの独立Expectedに対して説明できるか検証 | EKI #19/#48/#50、#354の同時Reservation負例。SimulatedとActualを分離 |
| **R3 AI支援のChange Review** | Source/Result限定Prompt・Context、Impact Graph、AIと独立Checkerの差分・SHO説明。A4となるOwner/Trust決定は人が受理 | EKI Authority/Access revocation、TEI #2514/#1124。AIはQuery/提案の権限に限定 |
| **R4 Safe Admin Intent** | 小さな`Node drain`または`Pool quota`のPreview→typed Proposal→EKI Admin API→実Receipt／独立Read。TEI Studio停止時もCLI復旧可 | EKI Admin Authority/CAS/fencing／#268の実接続。承認前・旧Revisionなら停止 |
| **R5 Resource-aware E2E** | CPU/RAM/GPU/VRAM/Nodeの異種実Profile、Local Governor、MultiWorker/ResidentとInput/Output Budget、実機証拠 | EKI #57/#242を含む実Worker E2E、#118 OS観測、#354同時性、安全性能を個別実測 |

**採用すべき原則**：TEIが管理Application（図・表・操作・Simulation・AIレビュー・ドキュメント・テスト）を広く担うことと、EKIのState/Resource Authorityが安全かつ独立に動作することは両立できる。後者をTEI生成のUI/AIから迂回しないことが最重要。低遅延・高権限のKernelをTEIで生成する案を全面否定はしないが、その場合はNative baselineと同等の失敗独立性・安全性・端末資源保護を、EKIのProvider実機と別の資格で検証する必要がある。

**Done（本稿）**：EKI既存Authority・Observation・Task/Node/ResourceとTEI Data/Role/Graph/Simulation/Actionを混同しない対応表、最小独立Fixture、Owner・段階・停止条件を提示した段階。**本番Node・GPU分割・Agent更新・Simulation Operator・LLMレビュー・TEI Studio／SHO E2E・EKI実管理操作はいずれもNOT RUN**。GPU共有隔離、資源の完全な測定・ノード全操作、Modeler編集、AI安全Review、Authority不変条件の適合は未証明。

**停止条件**：Node/Resource IDや観測Scopeが曖昧、Access/Disclosure範囲不明、Snapshot/Policy/Lease/Agent現在性不明、Meteringの偽装/欠測、GPU mode/VRAM上限が未解決、Agent/Local Governorの拒否とCentral提案が矛盾、Simulatedを実Effectへ昇格、AIが自己承認/credential越境、実Effect UNKNOWN、Source/Rule/Plugin失効のいずれかなら実Actionを止める。ただし閲覧できる範囲のread-only説明は維持する。
