# TEI×EKI：許容リードタイム・実績・予算・連鎖失敗を考慮する資源群コーディネーション

- 日付：2026-11-03
- 状態：**設計・独立反証計画。最適化ロジックの実装／実資源配置／校正／性能測定／Owner受理はNOT RUN。**
- ユーザーの要求：AI/非AIジョブの**許容リードタイム、過去実績、成功の可能性、故障が重なったときの許容時間内の回復**を、管理者が許す資源／予算の範囲で賢く調停する。単に最安のWorkerを選ぶのではなく、時点に応じて、待機／実行／確保／引上げ／中止と知らせる判断を改善する。
- 既存責務：[TEIはAI Job意味、EKIは資源群への委任、HPC・Cloud・MDMは専門実行Owner](TEI_AI_WORKLOAD_EKI_AGGREGATE_DELEGATION_2026-11.md)、[多段委任・実予約・失効のGate](TEI_EKI_AGGREGATE_DELEGATION_UNRESOLVED_GATES_2026-11.md)、[EKIへのジョブ実行委譲境界](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)、[TEI Ops/Simulator](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)。
- EKI既存要求：[RQ-010/012/013/014/052/057/058/060](https://github.com/hat47x/eki-fabric/blob/main/requirements/PRODUCT_REQUIREMENTS.md)。採用済み[ADR-0014](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0014-authorities-for-delegation-budget-capacity.md)のDLG-1（登録委任先）、BUD-1（物理予算）、CAP-1（有償容量）、ADM-1（管理者限定委任状）を必ず再利用。[#108](https://github.com/hat47x/eki-fabric/issues/108)は期限接近時の引き上げ／物理消費、[#109](https://github.com/hat47x/eki-fabric/issues/109)はPool単位予実、[#140](https://github.com/hat47x/eki-fabric/issues/140)は広域最適化で、それぞれ**要求整理のIssueはclosed**。本番アルゴリズムやProvider E2E完了は意味しない。
- 既存評価：[EKI #19](https://github.com/hat47x/eki-fabric/issues/19) learned estimate in shadow、[#48](https://github.com/hat47x/eki-fabric/issues/48) closed deterministic S1–S6 simulation/preregistered baseline、[#50](https://github.com/hat47x/eki-fabric/issues/50) open learned comparison、[#58](https://github.com/hat47x/eki-fabric/issues/58) Suitability/direct option、[#125](https://github.com/hat47x/eki-fabric/issues/125) fair share、[#67](https://github.com/hat47x/eki-fabric/issues/67) managed-cluster Provider、[#354](https://github.com/hat47x/eki-fabric/issues/354) concurrent worker reservations、[#112](https://github.com/hat47x/eki-fabric/issues/112) TEI Job I/O External API Gap。既存Simulator/Estimatorは再利用対象であり、MLをPlacement Authorityにする決定ではない。

## 1. 「最安」ではなく、予算と許容時間内で実現できる計画を選ぶ

**目的は各Taskの価格最小化ではない**。最初にOwner/Consent、Data Class、実行体/Resource Class、Lease/Effect、Provider稼働資格、予算／フェアシェアを**hard constraints**として選別する。次に、TEIで指定されたJobの許容時間に対して、Queue/Transfer/Setup/Compute/Verification/Commit確認/Recoveryを含む**end-to-endの完了可能性**を評価する。**条件を満たせる候補の中で**支払・資源消費・待機・中断・リカバリ等を最適化する。失敗確率の見積りが不十分なら未知として保留・再計測／既存Hostへ戻す。

- **許容リードタイム**：Jobが許された起点時点 `not_before`／提出時点 `submitted_at` と、利用者・業務側が期待する `target_complete_by`、厳密な `latest_useful_complete_by`、結果受理条件。単に`run duration ≤ lead time`としない。起点・clock/zone/単位・変更可能主体を明示し、`requester wishes`から優先権を作らない。
- **納期／SLO**：Business Target（TEI）、Service Class/許容遅延と通知方針（管理者承認）、Provider実Availability契約、予測の信頼区間を区別。`desired_probability`はTEI Policyで表せる候補でも、**EKIが期限保証のSLAを自動発行するわけではない**（RQ-010/057）。
- **回復の許容時間**：単独失敗での検出遅延、次の資源確保、checkpoint/resumeまたは再計算、追加I/O・検証を含む最大残余時間。複数Pool/Region同時不調や勤務開始の一斉reclaimも考える。ユーザー希望の`recovery_within`と、その根拠となるBudget/実Provider能力を別に評価。
- **分散しない選択**：Direct existing server/cloud/localの基準を常に比較する（RQ-014/#58）。小Jobの重いSolver・Telemetry取得・Control API往復がWorkloadより高くつくなら`DIRECT`。決定処理の追加コストも目的関数に含める。

**優先順位の原則**：安全とAuthority ＞ 物理/有償予算・管理者Policy ＞ TEI受理済み効果/データ意味 ＞ 期限を満たすための確からしさ ＞ 計算量・電力・転送・待機・運用の資源効率。利用者から与えられた`urgent=true`やモデルが推奨する最安ProviderでHard Constraintを緩めない。複数の有効候補でのみ目的間の重みを管理者Policyが選択。

## 2. 最小入力契約：Job Specの意味はTEI、実運用の権限はEKIとProvider

| 分類 | 入力・状態（名称は概念。公開API/DSL Schema未採用） | 誰が正本を持つか |
| --- | --- | --- |
| Workload／Outcome | Job/Operation/Step identity、TEI Meaning/Artifact/Executor digest、入力Data Class、検証結果の採用条件、副作用/再実行/Checkpoint能力、Workload Resource Profile、ターゲットOS/arch | TEI生成と業務Owner。EKIへ`TaskRequest`/登録済Executor制約へLowering。実行結果の業務受理はTEI/Host |
| Lead time／Delivery | `submitted_at, not_before, target_complete_by, latest_useful_complete_by, max_recovery_delay, priority_class, late_policy`、Run/Step全体の制約。Queue/Transfer/Verify/Result fetchを含む | 業務期待はTEI/Owner。優先Class／実Budget/Resource権はEKIに登録した管理者Policyが制限 |
| Budget／Authority | `budget_ref`、物理CPU時間/GPU時間/Memory×Time/通信/Task数、Provider reservation、BUD-1 nested share、費用発生可否/CAP-1 threshold、owner consent | **DLG-1/BUD-1/CAP-1** と下位Providerの実Quota/Cost。通貨の価格換算・請求はCore外のTEI/Financial Provider |
| Candidate Offer | 管理者登録済Pool・Trust/Data Class、Resource Sub-offer、容量予測/実ReservationとOffer期限、Provider setup/queue/cold start/locality/egress、部分・欠測・停電/通信の時点情報 | EKI DLG-1とProvider Admission/Observation。TelemetryやWorker自己申告はAuthorityでない |
| Historical outcomes | 同じResource Class/Artifact/Provider profileでのqueue/run/transfer/verify/retry/detection/cancel/interrupt、見込みと実績差、料金・物理消費、観測可能範囲/対象期間 | EKI Control Planeの実ObservationとProviderの署名・受理証拠。**Self-report ≠ measured performance ≠ trusted eligibility** |
| Plan／Decision | Plan ID/version、候補/基準、予測分布・不確実性、回復分岐、予約上限、決定理由、観測Revision、適用可能時点 | EKIの非Authority optimizer Proposal。現在のAdmission/Placement/CAP-1/Lease Authorityが別に採否決定 |
| Monitoring／Feedback | Task/Attempt/Lease/Provider Execution Ref/JobRun、時刻・結果・失敗原因・recovery elapsed、Model version、予測対実測 | Source別に所有し、EKIのOutcomeへ相関。Observationは実GrantやSLAに昇格しない |

**状態の粒度**：一つのPoolを候補にしても、同じ物理GPUを重複したGPU Sub-offerで数えず、Workloadの単一デバイスVRAM制約／GPU Mode／Arch/NUMA／Data regionの要件を下位Providerで検査。Group総容量の和だけで「納期に間に合う」ことを推論しない。現Reservable CapacityとExpected Burstの信頼性も分ける。

## 3. 見込み完了時間と失敗後の残余時間を分解する

**Job end-to-end**（概念）：
`T_complete = T_wait + T_approval + T_provision + T_input + T_start + T_compute + T_output + T_verify + T_result_visibility + T_business_commit_if_required`。
ただし認可の未確定、Cloud CAP-1の人手承認、Data sourceがunavailable、Provider reservationなしなら、定数時間へ勝手に置換しない。業務CommitはEKI結果と別であり、EKIはTEI DBのCommit所要を保証しない。

**Recovery plan**：障害が起きる時間`t_fail`に対し、残余時間`deadline - t_fail`から
`T_detect + T_fence/reconcile + T_new_admission + T_acquire + T_restore_or_restart + T_transfer + T_recompute + T_reverify + T_publish`
を見積もる。`Effect UNKNOWN`で安全な再実行条件が不明なら`recovery_infeasible`／`held`。Lease expiryは副作用不在の証明ではない。**失敗のたびに同じ「初期残余時間」を使い直さない**。

**共通故障**：同じSite電源、ネットワーク出口、Cloud Region、Identity Provider、Artifact Store、Provider Queue、朝の端末一斉reclaimは「異なるPool ID」でも独立とは限らない。`P(A failure and B failure)`を`P(A)×P(B)`と置くには別Site・別Failover経路の証拠が必要。Modelが知らない共通故障はリスク不明。複製を増やすほど成功率が上がるという単純公式は用いない。

**期限前に安全な警告を出す**：候補/回復経路の最短の確実な許容見積でも`deadline`に間に合わない、Cloud Budget不可、適格なPoolなし、Estimatorが校正不足なら`deadline_at_risk/needs_measurement`を返し、Ownerへ通知する。通知/Operator reviewが失敗を後付けで「達成」へ変えない。

## 4. Policyとしての予算最適化：物理量と外部価格を分ける

**EKI Coreには価格形成・請求・金額換算を入れない**（RQ-038/057、ADR-0014）。Coreで拘束するのは有期限の**物理量予算・Quota・Grant**、委任先のリソース制約、CAP-1の有償Capacity承認と実Provider上限。TEI/EKI Application PlaneのFinancial/Cost Providerが実料金/請求単価を型付きに読み、`estimated_cost`と`observed_cost`を明示し、管理者が採用するPolicyとして`cost_weight/reliability_weight/late_penalty/energy_weight`を提案できる。ただし価格テーブルが未取得・通貨/地域・課金粒度が曖昧なら金銭上の最適性は主張しない。最適化提案がCAP-1/Provider Credentialの権限を取得するわけではない。

**候補Planの比較（設計案）**：
1. Policy/Trust/Data/Resource/Consent/現在の予算に不適格な候補を**最初に除外**する。
2. 同じ検証済みWorkload/Profileに対して、完了時刻と消費の**分布または保守的な上下限**を比較する。失敗/中断が多数なら平均値だけの所要時間は使わない。
3. `P(verified_result_before_deadline | current evidence, plan) ≥ target`を品質要件候補にできるが、その`P`が校正済み・母集団適合でなければ**未判定**。この数式を法的SLAや実保証と混同しない。最新の稼働能力は実Placement段階で再チェックする。
4. 成立する候補に限り、観測/外部料金、物理リソース、制御/Decision overhead、転送、再試行、電力/人への影響、回復余力とFair-shareを比較する。**安価だが期限違反が多い候補より、予算内で間に合う可能性が十分高い候補**を選ぶ。
5. 有効な候補なしなら`cannot_meet_requested_lead_time`を明示。納期を勝手に延長せず、予算上限引上げ・締切変更・処理の段階化・Direct Hostへ戻す等、権限あるOwnerへの**選択肢**を提示する。

**予算を使い尽くさない**：確実に必要な保護容量や再試行予算を、先行のOpportunistic Attemptですべて消費してしまうと失敗時にCAP-1が使えない。Job/Sub-Flow単位の`initial execution budget`／`recovery reserve budget`をBUD-1の既存枠内で**配分**する提案は有用。ただしBudgetの新正本は作らず、同一Budgetが重複予約されないこと、失敗時に予約が実際に消滅したとは推論しないことを検査する。

## 5. 時間と実績に応じた調停：Planは静的でなく、再評価する

| 段階・時点 | 提案する判断 | 不変条件 |
| --- | --- | --- |
| Submit | TEIのJob/Deadlineと既存Direct Host基準を照合。必要リソース/Retryability/Result Profileに応じてPool/Provider候補、期待遅延・危険度・物理Budgetを予備比較 | Taskの受理は実Capacityを予約しない。Provider/Job/Artifact digestの版を記録 |
| Initial placement | 早い/安いOpportunistic Poolを試すか、より確実なManaged Poolを先に使うか、Queue/待機/投入時点を比較 | DLG-1/ADM-2/Placement/Leaseと実Provider Reservationの適格性を別に承認 |
| Progress monitoring | Compute進捗（Claim）、heartbeat（Observation）、Provider Queue/actual、残り時間/残予算とLocal Governorによる縮小を観測 | Claimは実完了・安全な中断を証明しない。観測が途絶えたらrisk上昇/unknown |
| Deadline pressure | 期限への余裕`slack`を、Queue/Provision/Transfer/Verification＋**回復時間**を含んで再計算。`slack ≤ fallback setup + conservative runtime`に近づけば昇格候補を提示 | 前もって実有償Capacityを確保する場合はCAP-1と二重Budget/Provider制約。自動Scaleoutは事前承認済みScopeのみ |
| Failure/reclaim | 同時失敗・未知Effect、Provider timeout、Site outage/一斉reclaimを分類し、独立PoolのFallback、Checkpoint/Resume、Task再投入と隔離を比較 | 旧Lease/Attemptの結果が遅着したら新Attemptとして受理しない。副作用Unknownは勝手にRetryしない |
| Outcome / feedback | EKI Verified ResultとTEI Business Receipt、実際のQueue待ち時間／処理時間／失敗／転送／利用量／費用／利用者への影響を分離して記録、予実/校正へ | 最適化Proposal/AI一致を正式なAssurance・Grantに昇格しない |

**待機も戦略**：Queueが短くなる見込みのある同一クラスタで待機した方が良い場合がある。価格の安い夜間Windowを待つ候補もあり得るが、Deadline/Quota/現在のGrantで認められなければ不可。**早く開始するほど良いとは限らず、安く遅く開始するほど良いとも限らない**。Control Planeは小さい規則から始め、評価計算のコスト／再評価頻度に上限・ヒステリシス/Minimum dwellを設け、Policy境界内で揺れを抑える。Solver overhead自身でリードタイムを失わない。

**冗長実行（hedging）**：同一Taskの並列複製は、純粋・冪等・独立検証可能な処理であっても、予算と共有障害を考慮し、既存EKI multi-Attempt/Result/verificationの適合が確認できる場合のみ比較。副作用Jobや不明なCommitでは**安全性が証明されないため既定禁止**。複製が別のCPU/GPU Providerに分散しても同じArtifact Store障害なら有効な冗長化ではない。Managed fallback資源を先に`reservation`するか`only escalate when needed`かは費用・期限・容量保証の別Scenario。

## 6. 実績とAI/MLの扱い：予測は学習できても、権限は学習しない

**履歴観測の最小レコード**：`Task/Attempt/Provider/Pool/Class`、Queue/Dispatch/Worker Start/Last Progress/Claim/Verified/TEI receiptの各Event時刻、失敗原因/端末人優先reclaim/通信喪失/Timeout、Checkpoints、検証不一致、物理消費/External Cost、`eligible/rejected`、Data Class、観測時点・欠測/打切り、Rule/Model/Estimator rev、Source/Evidence。一般利用者への開示はPool集約の必要な部分のみ（RQ-018）。個々のWorker自己申告/予測が高Trust等級を作ることはない（RQ-058）。

**統計上の失敗に注意**：
- **生存者バイアス**：完了したJobだけで平均所要時間を算出すると、途中消失や期限切れが消え、過度に楽観的になる。`unfinished/censored/unknown`は別に扱う。UnknownをFailure/Successへ黙って再ラベルしない。
- **選択バイアス**：従来Schedulerが選んだPoolだけの実績を、新Poolへ一般化できない。選ばれなかった候補の実結果は直接観測されない。共通Workload/eligible populationを固定したReplay/Shadow Comparisonで限界を示す。
- **少数/Drift**：端末Poolは季節・勤務時間・ユーザー帰還で分布が変化。Pool/Class/時間窓/Region/Workloadを分け、少数は`insufficient evidence`。A→B→Aや同名Pool別revisionの観測を混ぜない。
- **失敗相関**：別Workerでも同一拠点/ネットワーク、複数Providerでも同じCloud/Secret/Identity Providerに依存すれば同時喪失のリスク。独立と断定する前に共通依存をModelerのGraphで確認する。
- **Confidence/calibration**：出すのであれば`P(deadline success)`の予測確率をHoldoutの時間別・Pool別・難易度別で較正検査。Brier/Calibration/coverageやDeadline-miss rate、p95遅延の**実証した範囲**を併記し、単一平均点で保証を主張しない。
- **ML導入Gate**：EKI #19/#48/#50のDeterministic baseline→Residual/Completion Model→Shadow ranking→校正/対照→実適用の順。導入後もEligibility/Authorityは現行Coreが決定。AIによる説明・反例/Scenario生成はTEI Application Planeで行い、Human/Owner/Providerの実許可を代行しない。

## 7. コーディネーション例：40分の期限に、安い余力とManaged Backupを組み合わせる

**完全に合成した説明例であり、実EKI性能・確率や費用ではない。**

- `Job J`は提出から40分以内の結果確認が望ましく、Data Classは両Poolで許容、純粋・再実行可能。Jobの意味・40分の制約はTEIの受理済みWorkload。EKIでの実実行資格は別。
- `Pool O`：不安定な余力。初回試行が通常12分、ただし勤務開始・通信断で途中喪失する。安価だが完了時間の尾部が長い。単一Poolの`p95=12分`と断定しない。
- `Pool M`：管理されたクラウド/クラスタ。現在の予約・立上げ＋転送＋処理＋結果検証を保守的に22分と仮定。ただし22分は**SLOではなく合成Scenarioの固定見積**。
- 障害検出・旧Attemptとの照合で5分、Mへの移行準備を含むなら残りの許容時間はさらに短い。最も遅いFallback開始を`40 − 22 = 18分`とみなすのは、**Mの実容量がその時点に確保でき、旧Attempt/Effectが解消され、追加準備が22分に全て入っている場合に限る**。そうでなければ安全な切替点は前倒し、または最初からMへ委任する。
- 管理者の物理資源予算は同種`GPU-class-X minutes`で最大80、Opportunistic一試行20、Managed一回35（合成）。一度Oで20消費した後のMは55で予算内だが、M二回の追加は90となり予算超過。**一回の成功待ちに全予算を浪費せず、残余の回復計画を予算内で守る**。異なるGPU Class/円を勝手に同単位で足さない。
- A→Bの業務Flow後続はTEIが所有し、EKIがResultを受理してもTEI Business Commit/次段の開始は別Gate。Managed Provider自身のRetry/Queueも最終`ProviderRef`から相関を取る。

**判断の例**：40分の期限が緩く、Oの中断リスクと22分のFallbackが十分検証でき、物理予算が残るなら、まずO→監視→早い時点のMへ切替が候補。Oの中断率が高い、共通障害が大きい、回復予算が不足、Data ClassやCAP-1が不適格なら**最初からM**または**期限達成不能を前倒しで知らせる**。常にO優先という固定アルゴリズムではない。

## 8. 最小の独立Conformance（LB-01〜17、追加LB-18〜26は§10.6、全件NOT RUN）

| Case | Fixture／Fault Injection | Independent Expected |
| --- | --- | --- |
| **LB-01 Lead time origin** | NotBefore 09:10／Submit 09:00／Deadline 09:40、TZ違い・遅着Event | `T_total`にQueue/起動/転送/検証を含む。時刻取り違えで未受理の早期起動・期限上書きなし |
| **LB-02 Cheap but deadline-risky** | Oは安価・平均10分、ただし同条件で期限超過の尾部が大きい。Mは高資源・所要20分 | 平均Cost最小化だけでOに固定しない。充分な校正がなく`meets SLO`を発行しない |
| **LB-03 Late fallback** | Deadline 40m、O失敗19m、M保守的必要22m＋照合5m | 19+5+22=46m > 40m。間に合うと宣言せず早期Warning/既存Host候補を提示 |
| **LB-04 Reserve budget** | 同一GPU class Budget 80min、O 20、M 35、M追加35 | 20+35=55許可可能。20+35+35=90拒否。事前Reservation解放はProvider実Receiptを要求 |
| **LB-05 Common-mode outage** | 二つのEndpoint Poolが同じSite電源/Networkに依存し同時離脱 | 独立失敗として成功確率を積み上げない。別Failure DomainのBackup／Managed直接を評価 |
| **LB-06 Group reclaim storm** | 勤務開始でPC Poolの70%が短時間でreclaim。期限付き100Task | RQ-052：Retry storm/新Capacity不足/Provider Quotaを制限し、jitter/backoff/優先・回復 Budgetを守る。多数の同時CAP-1支出は拒否 |
| **LB-07 No-retry effectful** | 外部DB更新JobがACK前に通信断、Effect UNKNOWN | Jobの期限が迫っても無条件再試行／並列Hedgeせず`held/reconciling`。TEI Effect Receipt/Owner判断が必要 |
| **LB-08 Missing history** | 新Poolに成功Task1件、Failureは未観測、Estimatorが`P=99.9%`申告 | サンプル不十分なら`unknown/needs_measurement`。高Assuranceへ昇格不可 |
| **LB-09 Censored failure** | 100Task中完了70、タイムアウト20、結果不明10。完了分だけ短時間 | 70完了の平均から100件の期限成功確率を作らない。不明/打切りを区別 |
| **LB-10 Drift** | 同Pool ID/新Agent version・勤務時間変更・Policy rev変更、古い予測あり | 古い`p95`を現在の合格保証にしない。Requalification/再校正を要求 |
| **LB-11 Double quota** | EKI BUD-1枠残100、Cloud側Quota満杯、CAP-1提案あり | BUD-1があるだけではProvider Capacity受理できず、Cloudは拒否。Quotaを両方に二重予約した成功は不可 |
| **LB-12 Provider ack loss** | EKIからSlurm Job投入、Providerが受理後応答断。Deadline逼迫 | Provider ExecutionRef lookupを優先。再提出はCorrelation/Idempotencyがなければ`unknown/held` |
| **LB-13 Direct wins** | Local/Existing 5ms、EKI Decision/Dispatch往復10ms以上の同等仕事 | `DIRECT`正常判定。高価なOptimizer探索で実Jobより遅くなるのは不合格 |
| **LB-14 Fair-share override** | 部門Aが`urgent`と宣言し部門Bの保証枠を横取りしようとする | User preferenceだけでEKI Admin Fair-share/BUD-1を超えない。期限が危険でも無権限支出不可 |
| **LB-15 Forecast disclosure** | Pool内部Node非開示、Clientへ詳細な確率分布/下位Site数を返そうとする | RQ-018/058のDisclosureで抑制。最適化内部モデルがNode秘密を漏らさない |
| **LB-16 Sim→Act mismatch** | SimulatorはM available、実CAP-1承認時にProvider capacity消失 | Simulation/AI PASSでLeaseやProvider Reservationを発行しない。新しいAuthority検査→`unavailable`/再Plan |
| **LB-17 連鎖した二回失敗** | Deadline 50m、初回失敗15m、再試行2回目失敗後に残余12m。再々実行の検出・確保・検証込みで16m必要 | 現在残余12<16を明示。初回の50mをリセットせず、既存Host／部分結果利用／期限リスク通知のうちOwner許可範囲を提示。旧Attemptの副作用不明なら再々実行しない |

ExpectedはPlan/Estimator/LLMと**独立な固定Oracle**を用意し、実故障分布・異種Provider／実時刻/実消費は実測で別受入。合成Score/ScenarioのPASSを本番Deadline達成率へ昇格させない。失敗Caseは[既存TEI #2514](https://github.com/hat47x/tei-platform/issues/2514)のOracleとEKIの既存Simulation/Conformanceへ接続し、新Runnerや別検証Registryを作らない。

## 9. 既存Issue/ADRとの非重複・導入順・保留判断

| Phase | 最小成果物／検査 | Ownerと受入Gate |
| --- | --- | --- |
| **0 契約照合** | TEI Job/Batchの時刻・Retry/Checkpoint/Effect/Result Spec、EKI RQ-057/058とDLG-1/BUD-1/CAP-1のMapping、Direct-vs-Broker固定Oracle | TEI #1130/#1121、EKI #108/#109/#140（既存要求）、#58 |
| **1 実Job Wire** | TEI純粋Job1件のInput/Output bytes/Executor digest→EKI Task/Resultの受渡し | EKI #112/#57/#242。**旧422/501を現環境で再評価**し、Input ingest・手動補助のない実Resultが未対応なら該当境界で止める（§9.2） |
| **2 Deterministic time/budget** | Pool/Provider 2候補のQueue/Transfer/Run/Verify所要とBUD-1/CAP-1・Local Governor/資格を使い、Deadline-aware feasibility/Direct comparatorを決定論で出す | EKI #58/#67/#108/#140。Reason/Estimator Revisionと独立Expected |
| **3 Recovery-aware planning** | 2故障／共有Site障害／費用枠残余／Retry・Checkpoint・Provider ack unknownをSimulatorで評価、実Provider/WorkerのRetryを実測 | EKI RQ-052／#354、#19/#48、実Provider/Lease/Result Owner |
| **4 Shadow calibration** | Pool×Runtime/Workload Class×Time Windowの実Outcome、打切り・ドリフト/相関を分けたHoldoutとCalibration。Actual Placementへ影響させない | EKI #19/#50。学習モデルのAuthority昇格を行わない |
| **5 TEI Ops／Pilot** | 期限/予算/実結果の可視化、What-if・AI Critical Review、Fallback原因/Plan diff、Deadline Miss/Recovery SLO/物理Cost・顧客価値 | TEI #1124/#2514/#2620、EKI #125/資源管理Owner、Business Owner |

**Metrics（提案）**：`deadline_miss_rate`、`on_time_verified_completion_rate`（母集団と打切りを固定）、`p50/p95 end-to-end lead-time`、`p95/p99 recovery elapsed`（観測できた範囲と母集団）、`escalation rate/time`、`remaining recovery budget`、`deadline_at_risk lead warning time`、`compute/transfer/verification physical cost per accepted result`、`provider admission failure`、`decision overhead`、`owner-interference`、`unauthorized allocation=0`、`unknown-effect duplicate retries=0`。構成・入力・環境・失敗母集団が違う数字を単純比較しない。

**明示的保留**：TEI/EKIの正式DSL/APIの名前、確率閾値や統計推定器、分散最適化Solver/学習Model、Cloud料金の自動入力、SLA/保証、recovery budgetの物理表現、Hedgeと独立性、Ownerの全自動再配置許可。**現行EKIのAuthority/実Providerの完了資格・持ち主の拒否権は不変**。どのアルゴリズムでも変更する前に既存Issueと独立TestでOwner判断を行う。

**停止条件**：実Artifact/Result不可、必要なProvider/Trust/Data/Deadline/Consent不明、Source stale/partial、現在のCAP-1/Provider予約なし、物理予算不足、実External Effect Unknown、故障相関不明で必要な回復確率を保証できない、Fallback開始時の必要時間超過、Estimator校正不足、Solver overhead/費用逆転、一般ClientへのPool秘密漏洩。読取可能な状態の説明・期限リスクWarningは維持するが、実権限なしで高額Scaleoutや危険な再試行を行わない。

### 9.1 Deadline/Cost判断を、実Source/Artifactが一致する対照試行へ接続

[一改訂のDirect／EKI Native／Managed Pool縦断と独立Oracle](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)のW-3/VS-10〜12では、09:00開始、10:00最終期限、O→M切替の検出5分＋M実行26分という**合成値**を固定し、09:29が最遅判断候補、09:32故障判明なら10:03となる負例を設計した。**予測の比較対象は同じTEI Input/Executor digest/Result検証条件**であり、出力本体のないEKI`completed`とDirectの実Outputを比較しない。T0/T1/T2の実経路が成立するまで費用・期限達成率・回復確率の製品実績を主張しない。LB-01〜26の予算/相関/公平性を再実装せず、VS Fixtureは実験の縦断証拠だけを扱う。

### 9.2 歴史的なEKI 422/501と現Dispatch/Output GETの区別

[実行縦断§12](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)で現EKI `main`のConfig別証拠を再確認した。Dispatchはoptionalだが実配線コードとTest-profile実Process試験は存在し、Result GETも`Results:store`なら実装経路があり、Output GETもArtifact Store/Owner/期限条件の下で存在する。**旧Spikeの422/501を現在の全Profileに当てはめない**。一方Input ingest-intents/Artifact uploadのHandlerはまだ501、テストでResult Acceptanceを直接呼ぶため**製品の完全な自動受理は未証明**。本稿のDeadline/Budget/Recovery実績はこれらRE-01〜08を受理してからしか測れない。将来費用を先に最適化しても、Input bytesやAcceptedResultがないTaskを`成功`に数えない。

## 10. 横断深化：単体Jobから複数Job・期限付き約束へ（2026-11-03、第2段）

TEIの別議論（[#1130](https://github.com/hat47x/tei-platform/issues/1130)のHeadless Batch／Trigger／Simulation、[#2499](https://github.com/hat47x/tei-platform/issues/2499)の外部Authority、[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Oracle、組織の時点付きMembershipとDecision WorkbookでのFlow意味受理）と、EKIの実装進捗を照合した結果、**単一Jobの安価なPool選択だけでは、複数の期限付きJob、部門別最低枠、回復用Capacity、Providerの競合が同時に起こる場面を扱えない**。しかしEKIに新しい汎用業務Workflow Engine／HPC Node Scheduler／Budget Authorityを作る必要もない。以下は既存契約に従う**短い時間窓ごとのPortfolio-level Coordination Proposal**を定義する。

### 10.1 「期限に間に合う見込み」「資源を予約した」「期限を約束した」を区別する

| 事実・状態 | 正本と有効範囲 | 伝えられること／伝えてはいけないこと |
| --- | --- | --- |
| TEI Business/Flow Deadline | TEIのOwner受理済みJob／Flow Definition・Meaning/Source/Artifact Revision | `target`、`latest_useful`、Step critical pathの依存・検証・業務Commit。**Taskのdeadlineを勝手にFlow成功へ読み替えない** |
| Pool Forecast | EKIのObservation＋Provider Offer、Model Revision、Sample/Time Window、観測時点 | `deadline_at_risk`とPlan候補。**予測は物理Capacity・SLA・Trustを証明しない** |
| Budget Reservation | 採用ADR-0014のBUD-1、必要に応じCAP-1とProvider-side Quota | 物理量の使用権限と有償確保の承認境界。**Budgetを確保しただけでGPU slotが取れたわけではない** |
| Provider Capacity Commitment | 外部HPC/Cloud/Site Gatewayが認めた予約・時間窓・Resource Class、Provider Revision | 予約の強度（soft offer／queue acceptance／hard reservation）、失効・取消条件。**将来のOwner reclaim、Provider停止、絶対的な可用性保証と同一視しない** |
| EKI Attempt/Lease | EKIのTask/Attempt/Lease/Recovery Store、現fence/expiry | 特定試行の有限な実行資格。**Leaseは完了時刻や外部Effect無発生の証明ではない** |
| Verified/Business Outcome | EKI AcceptedResult／TEI業務Provider Commit Receipt／実Independent Observation | 最終的に得た結果と適用できる意味。**Estimator hit／Callback／Job submittedで完了にしない** |

**結果の達成時点**はTEIが採用した条件で異なる：`EKI Result accepted`まででよい純計算と、`TEI Provider business commit verified`まで必要な処理を区別する。同じ期限をこれら全てに一律適用しない。

### 10.2 ポートフォリオの「実行希望・資源予約・回復枠・部門公平性」を分離する

**重要な差**：前節§4の`recovery reserve budget`（将来の回復用に使い切らない**物理量予算の取り置き**）は、次のどちらとも異なる。

- **EKI Recovery Window Permit/Budget**：回復が一斉に発生した際のAttempt release数／transfer bytes／compute millis／時間窓を制限する**安全側のレート・増幅防止**。既存ADR-0015（[関連EKI #154](https://github.com/hat47x/eki-fabric/issues/154)、closed：`Pending`、予算・Permitの永続化設計／実MVP E2E保証とは別）をOwnerとする。BUD-1の物理量Budgetを複製・置換しない。
- **Provider Capacity Reservation**：特定のGPU/CPU/Worker Classに対する実枠・時間窓を、外部Providerが予約する行為。BUD-1の利用可能残高や「回復予算を保持した」という内部Planは、将来の物理Capacity獲得を保証しない。有償CapacityならCAP-1承認とProvider側の請求/費用開始・解放Receiptも必要。

**Portfolio単位のProposal**には、同じPoolに並ぶTaskと部門／Service Class、期限付きCritical Jobs、利用中の予約、回復Reserve、CandidateのData/Trust・Requester Scopeを入れる。部門ごとの最低割当・上限・Borrowingは未決の[EKI #125](https://github.com/hat47x/eki-fabric/issues/125)がOwner。回復の窓をまたいだBest-effort飢餓は[EKI #155](https://github.com/hat47x/eki-fabric/issues/155)（open）がOwner。期限が近いTaskを**全資源への優先権**へ自動変換しない。組織の時点付きAssignmentとResource Grantは別権限、古い部門所属や利用者申告の`urgent`から公平性枠を引き上げない。

**MVP判断順序（新Solver不要）**：
1. 管理者許可／Data/Runtime/Resource/Local Governor／DLG-1／BUD-1／CAP-1／Providerの現在Eligibilityで**hard filter**。独立Expectedが無ければ`unsupported`を返す。
2. `DIRECT`実行と各管理済みPool/動的Poolの`queue+setup+transfer+compute+verify+visibility`、必要なら業務Commitを比較。各EstimateはSource、測定期間、Calibration、欠測、Confidence、Time Windowを付記。
3. **全Jobの既存Commitmentと公平枠**を固定した上で、候補ごとに残り時間と回復に必要な最小準備時間、ProviderのCapacity Commitmentを評価。優先期限のみ見て他の予約Jobを押し出す変更は、別途権限と影響検証が必要。
4. その条件内で、安価なOpportunistic、予約済Managed Pool、Direct Host、Cloud CAP-1を、転送/再起動/料金/電力/利用者干渉とともに**比較するProposal**を作る。目的の重みは管理者Policy／OwnerのBusiness Deadline。EKI Coreで通貨・価格や組織Fair-shareの恣意的な重みを自作しない。
5. 実実行時には**予約直前の現在性再評価と条件付きProvider Admission**が必要。Planは予約を発行せず、Current Authorityは最後の段階で別決定。拒否・失効・Observation staleなら再Plan／期限リスク通知、実完了を保証しない。

**予約の棚卸し**：Planを再計算しても、過去のCAP-1予約・Provider側の稼働費用・未確認Effectを自動消去しない。旧予約と新予約が重なる期間／Cloud終了処理が非同期で課金される期間を含めて、上限を越えないか確認する。Reservationを変更する場合の`old release request`と`provider release confirmed`は別状態。

### 10.3 回復の見込みは「未来の時間」を前借りできない：Rolling Horizonで再計算

**各判定の有効区間**：`Plan = (Task/Flow rev, resource/Policy rev, budget rev, provider offer rev, evidence window, now, horizon)`。新しいAttempt・Owner変更・Clockの進行・価格表変更・Provider停止で`Plan stale`となり、そのPlanの見込みから新しい実行資格を生成しない。

**実行後の再評価**では、末尾期限から現在時刻までの`remaining_time`と、当該Task/Flowで既に消費した物理Budget、旧予約、未確認Effectを入力に戻す。特定のJobだけでなく、他部門・他Jobの既存Commitmentを含める。再Planは前のPlanと同じ`Provider Execution Ref/Attempt`が現存している可能性を扱うため、**後から受け取ったResult・旧Lease・失効の二重処理**を避ける。

**見込みの品質**：未知の故障率や失敗相関に、仮の`0%`を入れて「99.9%成功」と表示しない。Deadline success probabilityの数値化は、同等Workload/Pool/時間帯の適切なHoldout、打切り・選択バイアス、共通障害と予測校正を検証できる場合に限る。結果を知らないTaskは`censored/unknown`として扱い、`Task refused`や`Telemetry delay`と一律なFailureにしない。実工場・事務所等の労働時間や端末一斉回収はFuture Demandに影響するが、**MDMの前回Inventory=現在の実可用性**ではない。

**ヘッジ／冗長化**は純粋・複製安全なTaskのみ、実Providerの二重課金とResult verify費用を含む場合に比較する。Effectful/UNKNOWNなTaskに期限制約を理由に多重Attemptを勝手に追加しない。同じSite/Storage/Identity Gateway依存ならバックアップのPool IDを増やしても障害分散ではない。

### 10.4 期限・待機・実行見込みなしを依頼元へ誤認させない

[EKI #249](https://github.com/hat47x/eki-fabric/issues/249)（open）は受理済み`queued`を配置待ちと見込みなしで区別できない問題、[#111](https://github.com/hat47x/eki-fabric/issues/111)（open）は未開始の期限超過状態と`実行されたか不明`を分離する要求を扱う。したがって、現在のExternal APIに以下の全部が存在すると**主張しない**：

| Client向けに表したい概念（Proposal） | 事実源と実判断条件 | 誤った説明 |
| --- | --- | --- |
| `waiting_eligible` | Taskは受理済み、適格なPool/Provider候補が存在し、今はQueue/Wait | `queued`＝必要なら必ず開始できる |
| `blocked_by_policy/budget` | Admin Policy/物理Budget/Admission条件で止まる。非開示Poolの有無は見せない | `no eligible resource`からNode数・所在・他部門Quotaを漏らす |
| `no_current_plan` | Candidate不足・Provider unavailable・能力未確認・Model校正不足のいずれか。Reason Scopeを限定 | Providerが`unavailable`だからWorkload不適格が確定 |
| `deadline_at_risk` | 現時計と現在の実Provider Commitment／回復計画を照合し、期限内完了を立証できない | `likely late`＝Task failed／自動費用支出の許可 |
| `deadline_missed_not_started` | **どのAttemptにもdispatchの証拠がない場合だけ** | 期限切れ＝外部副作用が無かった |
| `deadline_missed_outcome_unknown` | dispatch済み/外部Effectの可能性あり、確定Receiptがない | Deadline経過＝safe retry or cancelled |

TEI Studioでは`source observation`／`simulated plan`／`EKI accepted/admitted`／`Provider reserved`／`EKI verified result`／`TEI committed business effect`を縦に並べ、Reason/出典/時点/不明を別に表示する。External APIの公開Status enumは、EKI #249/#111のOwnerが正式受理するまで変えない。**実行見込みと管理者向けの詳細Root Causeを同じ開示範囲にしない**。

### 10.5 既存Measurementの実装位置と、改善効果の検証を混同しない

EKI ADR-0039（最小安全観測）と[#548](https://github.com/hat47x/eki-fabric/issues/548)（closed、発生点で計測する配線と検証が報告済み）は`admission.rejected`／`lease.reclaimed`／`claim.refused`／`limit.refused`の4種類の実安全Eventを実発生点で数えることを扱う。**これら4カウンターだけでEnd-to-endの完了時間分布・故障相関・Budget節約・Deadline達成確率は学習できない。** さらに、End-to-end時刻・Provider Queue・Transfer・Result検証・Business Receiptの観測要件を**別のOutcome Sampling Profile候補**として整理する。ただし、新しいEKI計測DB/Registryや個体追跡の全量Logを最初から作らない。EKIの既存Task/Attempt/Provider相関・Observation Planeと[#19](https://github.com/hat47x/eki-fabric/issues/19)のshadow-schemaを再利用する。

**評価順序**：①固定Deterministic/最少2PoolとDirectの比較、②共通障害／複数Job／Recovery WindowのScenario、③実記録を使ったShadow（別のTrialではCurrent Authorityへ影響無し）、④Holdout校正とProvider実測、⑤Ownerが支出/期限条件の自動化閾値を受理する場合のみ実Pilot。単純な`GPU utilization increased`、LLMによる`safe`、過去のSimulation PASSは実用価値や安全性の証明ではない。

**現在の受入判定**：#548の実装・試験の完了は、4つの安全Eventの発生件数に限る。Deadline内の業務品質・Queue/転送・実費用・回復時間・故障相関・Provider Reservationの観測が同時に揃ったわけではない。実績に基づくPlannerの予測精度や利用額の節減を、このIssueのclosed状態から導出しない。

### 10.6 追加の独立Expected（LB-18〜26、全件NOT RUN）

| Case | 固定条件と注入 | Expected：満たすべき区別 |
| --- | --- | --- |
| **LB-18 二つの期限ジョブの同時競合** | あるManaged Poolに時間窓内1枠のみ、Job A/Bとも20分の処理と30分の期限を申告、Aの予算枠は許可・Bは異なる部門で最低保証あり | 別々の単体Plannerが同じ物理枠を二回Reserveしない。両方期限内を約束せず、Provider/部門Policyに従って待機・代替・期限リスクを説明 |
| **LB-19 Recovery Reserve≠Capacity Reserve** | AのBUD-1物理予算に再実行分の余力あり、Provider実GPUは他JobにReservation済み | 回復予算があるからFallback GPUを確保できたと表示しない。現在のProvider Reservation/Quotaが無ければ保証なし |
| **LB-20 Recovery Window Budget耐久性** | 大量reclaim後にControl再起動、同一Time Windowでrelease 10件、毎回`Budget{}`へ戻したくなる | EKI #154のWindow/Pending/Permitの耐久不変条件を維持。独立BUD-1を迂回せず、restartでrelease/transfer budgetをリセットしない |
| **LB-21 Recovery Fair Share vs Urgency** | Urgentが継続流入、部門Bのbest-effortは何回もWindow跨ぎで延期 | #155/#125の未採用Policy下で「公平性保証済」と表示しない。UrgentによるGrant/Quota越え不可。飢餓の検出/Policy候補を分ける |
| **LB-22 期限切れで未dispatch/済み不明** | 未開始Taskと、dispatch後Lease expiry・外部Effect不明のTaskが同じDeadlineを越える | 前者`not-started`相当、後者`unknown/reconciling`を区別。期限経過を無効果/自動retryへ縮約しない |
| **LB-23 予測の打切りとBias** | Pool Oは完了20／消失40／結果不明40、Estimatorは20完了の平均のみで`on-time=100%`申告 | 少数の完了だけで`deadline success 100%`を出さない。Censored/Unknown/Eligible populationとCalibration不足を表現 |
| **LB-24 再計画と失効** | Plan P0でProvider A Soft Offer、Plan P1直前にAのQuota/Pricing/Agent Revが更新 | 旧Planを実Current Authとしない。BUD-1/CAP-1とProvider Confirmationを再評価、既支出分は消さない |
| **LB-25 共通障害の誤分散** | 二つのPoolが同一Artifact Store/Identity Provider/電源/Networkを共有、同時失敗 | `independent`の成功確率積を使わない。Shared dependencyをGraph上で表示、Risk unknownまたは別Fault Domainを候補に |
| **LB-26 TEI FlowとJob Deadlineの境界** | A→B、EKIのA結果が期限内、TEI Bの必須DB Commitが未確認／TEI Coordinatorが停止 | `EKI verified A`≠`TEI Flow complete`。Owner/HostがBを検証するまでBusiness CompletionやSLO達成を宣言しない |

**最小の裁定**：LB-18/19/20/22/26は既存契約の反証であり、独自のOptimizerや新Core型がなくても判断できる。LB-21はEKI #125/#155のOwnerがFair Shareの規則を採用するまで結論保留。LB-23/25は既存#19/#48/#50のTrace/Shadow評価、LB-24は既存Admission/CAP-1/Provider CASの契約へ渡す。**全体最適の理論的保証、確率校正、実Deadline/SLA達成率は未検証**。

### 10.7 Deadlineの型と実EKI P0単一deadlineは一致しない

[実外部契約の静的対照WP-01〜19](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)§14により、EKI P0 TaskRequestの時間項目は**単一`deadline`**であり、TEIが管理する`not_before/target_complete_by/latest_useful_complete_by/recovery_within`を無条件にLoweringできないことを固定。**エンドツーエンド業務期限を単一実行期限に置換しない**。TEI Application/実Provider/Policy Ownerのどこが残余制約を強制するか具体的な引受が無ければ、EKIは`deadline-aware placement`の保証を与えず`cannot-enforce/needs-measurement`を返す。期限付き予約・CAP-1費用・Provider内部Retryが実在するかは別のAuthority/Receiptで検証。#2787の19/19はSource ABI対照であり、過去実績の確率予測・回復成功率・金銭最適化の証明ではない。

## 11. Task Groupの部分成果・計画再開・リカバリを結ぶ予算評価（2026-11-04）

[実行縦断§13](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)により、TEI `run-plan`のStep進捗とEKI `Task/Attempt`の再実行は既存実装が進んでいることを確認した。ただし、TEIのStep完了記録と実ProviderのBusiness Receiptは別であり、`runlog.File`はPlanの版・外部Effect/Outputや複数プロセスCASを証明しない。**「再開できる」は「安全に副作用を再実行できる」ではない。** 現状の継続的なコーディネーションでは、実行計画／Task状態／Provider状態／業務完了の異時点を別の根拠として維持する。

### 11.1 期限内の「使える成果」の正本はTEIのJob/Quality Meaning

[EKI #126](https://github.com/hat47x/eki-fabric/issues/126)（open）が扱うTask Groupは、**一群のTaskに予算・期限・取消・部分結果を付ける**構想であり、TEI業務上のRunPlan/Flowそのものではない。Groupの最小経済性評価は、以下の違いに依存する。

| 業務受理条件（TEI／Owner側） | 予算配分に対する意味 | Provider/EKIのTask outcomeとの差 |
| --- | --- | --- |
| **All required** | 1件でも未取得/unknownなら、部分実行が安くても最終成果未達。残余期限・再試行予算を評価 | 80/100完了を`100/100 quality PASS`へ変換しない |
| **Minimum useful subset** | `min_verified`件・対象範囲/独立性/品質などのOwner契約を満たせば、未完了の追加Taskへ支出しない戦略も許容され得る | 最初の20件だけが偏ったPool/母集団なら、20件の成功で全体の推論品質を保証しない |
| **Progressive/anytime** | 既に検証済みの結果を段階提供し、残余予算と期限までの追加価値を比較 | 部分Snapshotを`final result`と混同せず、Result Revision/Quality Coverageを明示 |
| **No partial usefulness** | 期限までに必要な独立Expectedを満たせない見込みなら、早期停止・Direct fallback・Owner通知を提案 | EKI `Task completed`の数が多くてもBusiness Outcomeは0かもしれない |

上記の`min_verified/coverage/value_of_next_result`は**TEIで受理済み業務Meaningの候補**であり、EKI #126の正式API/Schema・Runtime保証ではない。重み、価値、単価をEKI Coreに閉じ込めず、Financial Provider/TEI Application Planeで評価する。**Groupの期限や予算の強い一体性は、EKI側で採用されるまで`proposed`として扱う**。

### 11.2 予算に加えるべき「追加1件の価値」と「再試行コスト」

**事前に確保された現在GrantとOutcome品質を守ったうえで**、`Candidate action`の比較票に`marginal_verified_utility`、`remaining_deadline_slack`、`remaining_physical_budget`、`provider_capacity_commitment`、`correlated_failure_risk`、`detection_and_reconcile_time`、`provider_retry_visibility`を追加する案。**効用値はOwner受理したモデルの値に限り、AIが高い成功確率を出しただけで支出しない**。

- `start another task`：新規に得られる**独立な**情報量が、転送・検証・資源消費・期限遅延に見合うか。
- `retry existing task`：旧Attempt/外部Provider内部Retry/未知Effectを照会し、追加結果の価値と、二重支出・Group Budget/Recovery Windowの競合を比較する。
- `escalate to managed pool`：Provider Reservationと費用開始/解除の実Receipt、全Task間のBudget競合を再評価。**BUD-1物理枠があることはCloud/GPU Slotがある証拠ではない**。
- `stop or accept partial`：残りのTaskが意味・品質に必須でなければ、検証済み部分結果をOwner契約により採用し、実未完了/unknownを消さずに終了または保留する。取消要求は外部Effectが生じなかったことを意味しない。

**複数のTaskの部分成果**を一つのQuality Claimにする場合、`verified/success`、`failed`、`not started`、`cancel requested`、`unknown`、`excluded from required scope`は異なる。未完了・不明の分母を減らして`100% completed`にしない。疑似集計だけでは、本当のGroup Budget/Group Cancellation/Group Result Authenticationは得られない。

### 11.3 連鎖失敗と再開の受入基準（LB-27〜32、全件NOT RUN）

| Case | 入力・故障 | 独立Expected |
| --- | --- | --- |
| **LB-27 Partial success/Quality** | Task 100件のうち80 verified、20 unknown、TEI品質がAll-requiredとMinimum-usefulで異なる | All-requiredは不合格/未確定。Minimum-usefulは母集団/独立性/Quality条件を確認した場合のみ限定採用 |
| **LB-28 Group budget conflict** | 同時TaskがBUD-1の共通残額を使い切る、Provider予約済み・キャンセルACK不明 | Proposalの合算だけで二重支出可としない。BUD-1とProviderの実Reservation/解放Receiptで停止 |
| **LB-29 Three-level retry storm** | TEI RunPlanがResume、EKIが新Attempt、Cloud Queueが内部Retry | Effect-freeでないJobはProvider内部Retry停止/観測を証明できなければ不適格。期限逼迫でも二重Effect禁止 |
| **LB-30 Crash after business Commit** | Provider業務Commit後にTEI Step記録前でCrash、RunPlanは未完了と判断 | 同じOperationID/Provider Lookup・Receiptで照合。記録上未実施を「安全な再実行可」に読み替えない |
| **LB-31 Plan revision ABA** | Plan A→B→Aで同一PlanID、Stepの入力・Executorと時点異なる | 同一表示Plan/IDや古い`steps`だけで実行資格再利用しない。Source/Meaning/Artifact/Owner Scopeを再資格化 |
| **LB-32 Early marginal stop** | 20件の検証済み独立結果でOwnerの最低Quality達成、追加80件は期限・物理予算圧迫 | 最小値達成を全体完了や100件の真実と偽らず、追加実行/中止/延長を実Policy内で比較。未実Taskを`failed`にしない |

**Phaseの判断**：①TEI実`RunPlan`の現RunKey/Step記録とVS-23〜32を直接Goで反証、②EKI #267/#381実装のTask Phaseと一つの実AcceptedResultまでの現在資格を実Processで再実証、③EKI #115の内部Retry/Provider Ack unknown、④EKI #126のTask Groupの必要最小Contract、⑤Deadline/部分効用/相関失敗を既存#19/#48/#50のDeterministic→Shadowで比較する。**Phase①〜④の実E2Eが通る前に、確率最適化/有償Autoscaleの実運用価値を主張しない**。

**非目標**：TEIにEKI Task Group台帳を複製、EKIにTEIのBusiness RunPlan/Workflow Engineを移植、PlanRunとTaskを同じIDにする、全Providerに完全な2PCを要求する、未知Effectを再実行の許可に変える、独立OracleのExpectedをScheduler/AIが生成する、正式なDSL/API/価格体系を本稿で採用すること。

## 12. 実行体・依存物・データの局所性を期限・予算の調停に組み込む（2026-11-04）

他レーンの[TEI Job/Buildの段階割当 Draft #2788](https://github.com/hat47x/tei-platform/pull/2788)、[TEI #1130 BU-01〜09](https://github.com/hat47x/tei-platform/issues/1130)、[EKI #112 TEI workload projection](https://github.com/hat47x/eki-fabric/issues/112)、[EKI #58 Suitability Gate](https://github.com/hat47x/eki-fabric/issues/58)、[EKI Control/Data Plane](https://github.com/hat47x/eki-fabric/blob/main/architecture/CONTROL_DATA_PLANE.md)、[EKI RQ-029](https://github.com/hat47x/eki-fabric/blob/main/requirements/PRODUCT_REQUIREMENTS.md)を横断。前節までのPool選択が計算資源・queue・失敗見込み中心だったことに対し、**実行可能なArtifact・Toolchain・入力・結果の置き場所と事前準備**を、費用/時間/Authorityの独立制約として補う。

### 12.1 Sourceの意味、ビルド成果物、実行可能な配置は別

**TEI**はFunction/Job/Flowの意味、単一Go Binary既定、Build Target/Artifact Byte/Toolchain/依存閉包、Entry/InvocationとIndependent Expectedを管理。**EKI**は実行リソースの適格性と委任資格／Task/Attempt/Lease/Resource/Transfer budgetを管理。**Provider**は実際のRuntime Class・Data Plane Store/Cache・Executorを準備し実Jobを起動する。EKIがTEI Sourceを解釈してGoビルドを再実行する前提にしない。複数Buildは必要になった段階でTEI Build管理Pluginが提案し、業務上のFunctionの名前で物理配備を分断しない（TEI #2788）。

**候補Poolにおける「実行準備済み」**は、次の独立Witnessが揃う場合だけ判断する：
- `source/meaning/build`：TEI受理済みJob意味、完全性を持つ入力closure、Go Toolchain/OS/arch/Artifact Digest・Entry/Executorの版。TEI外部Importのselected parsed Source Witness（Draft [#2812](https://github.com/hat47x/tei-platform/pull/2812)）は**Full Closureでない**ため、そのままBuild再利用資格には使わない。
- `executor/runtime`：Providerによる対応OS/arch/runtime/toolchain・Approved Executor/Artifact digestの現時点の束縛。EKI P0の`version_constraint`はAdmission未評価、`parameters`はWorkerへ未配送という静的差異を先行[非損失Gate](TEI_EKI_ONE_REVISION_EVIDENCE_VERTICAL_SLICE_2026-11.md)§14で拒否／保留する。
- `data`：Input artifact bytes/size/digest、Data Class・Residency・Owner/Grant、Store/Cacheの現在性、実Provider Data Planeへの安全な取得経路。WorkerのClient指定URLを自動取得しない（EKI **RQ-029**）。同じDigestのCache hit自己申告は実データ存在とSecurity Authorityを証明しない。
- `capacity`：DLG-1委任先登録／BUD-1物理予算／CAP-1費用支出許可／実Provider CPU/GPU/Memory Reservation/Local Governor。Artifactが用意できてもGPUが取れなければ`execution-ready`ではない。逆も同様。
- `outcome`：Result本体をStoreへ戻し、検証/Owner開示を通過できる出力転送経路。入力だけでなくOutputの保持費用・期限・検証遅延も含む。

上記を**新しいRuntime/Registry・共通Manifestの正式Schemaとして採用しない**。既存のTEI Build/Entry Witness、EKI Provider Resource Class、Data Plane Store/ArtifactGrant、Runtime／Actual Resultを、読み取り専用の判断票に束ねる。

### 12.2 Cacheを考慮したDeadline/Costモデル：coldとwarmを混ぜない

**時間の予測（Proposal）**：`T_total = T_admission + T_queue + T_capacity_reserve + T_artifact_stage_or_build + T_input_transfer + T_start + T_compute + T_output_transfer + T_verify + T_result_visibility + T_business_commit_if_required`。実行体が既に配布済みならStage部分は減るが、**安全・同一Digest・Grantを確認した実Cache Hit**でなければゼロと置かない。`T_artifact_stage_or_build`を二重計上しないよう、TEIで事前Buildした場合はEKI Workerでの追加Buildを前提にしない。複数Taskで共通Artifactを使う場合はStageコストをTaskごとに無条件再計上せず、一方初回配布・Cache miss・Eviction・時刻帯別Loadを見積もる。

[EKI #112](https://github.com/hat47x/eki-fabric/issues/112)には、TEI projection Spikeの**ビルドキャッシュ有無で約0.8秒と12.5秒（約15倍）**という過去の観測がある。これは当該環境での事例であり、全Poolの一般的な性能値／実Go Binaryの配布速度ではない。同じIssueはToolchainが`go.mod`指定によって変わる実例を持ち、EKI RQ-029は**任意のWorker-side network fetchを既定で認めない**。必要なToolchainをWorkerに黙って取得させるPlanは不適格。事前に承認済みData Plane/Package Sourceへ配備・Buildするか、既存Host/Directを選ぶ。

[EKI #58](https://github.com/hat47x/eki-fabric/issues/58)の既存Suitability合成例は、Local実行60秒に対し、分散候補は計算20秒だが入力8GiBを100MiB/sで転送するため**入力転送だけで約81.92秒、単純合計で101.92秒**になる。つまりGPU/CPUが速いことはEnd-to-endの優位性を示さない。**Output転送・検証・再試行・キャッシュ準備・Planner自身の遅延**まで含んだ同一Workload/Result Qualityの対照でのみ「EKIの方が速く/安く完了する」と主張する。

**Placement候補の区分（正式enumではない）**：`direct-existing`、`warm-artifact-and-data`、`warm-executor-cold-input`、`requires-authorized-staging`、`unknown-version-or-cache`、`not-eligible`。Cache/Bundleが古い、Body bytes不明、Toolchain不適合、Source closure不足、Data Region違反の場合、`cache present`だけを根拠にeligibleへ昇格しない。

### 12.3 事前配備と計算資源の予約を同一の行為としない

期限逼迫時にManaged HPC/Cloudを**先に使える状態へ準備する**ことは有用だが、実行許可・費用発生の条件が異なる：
1. `prepare/stage proposal`：必要Artifact/Toolchain/InputのHash・Provider Capability・Data Classを検査。転送・Build・事前保存にもOwner/Quota/Transfer Budgetが必要。明示承認なしのプリウォームを低信頼PC群／他組織へ撒かない。
2. `authorized stage`：既存のData Plane Grant／Approved Store/Provider、Artifact Byte・Digest/Scopeと配備先を限定し、実Observationで完了を確認。**Stage成功≠Run Grant**、Stageを実WorkOrder起動として記録しない。
3. `capacity reserve`：上記の準備と別にEKI BUD-1/CAP-1・Provider Admissionで実計算枠・時間窓を確認。ProviderがCapacityを将来保証しないなら、Stage済みでもFallbackは`speculative`。
4. `execute and verify`：Source/Meaning/Executor/Artifact/Policy/Leaseの現Revision、Local Governor、Input/Output、Result Qualityを再確認。旧Stageがあるだけで旧TrustやGrantを延命させない。
5. `release/reconcile`：Stageされた機密Artifactの保持/削除と、Cloud有償Capacityの課金終了は別Receipt。Stageを失敗したRunの一部として二重計上しない一方、実転送費用・格納量を予算から消さない。

**予算が厳しい場合の推奨順**：まず無権限・実行不能・Source/Artifact不適格候補を排除し、Direct/Warm/Coldの完成見込みと物理消費を比較する。後から別Poolへ移す場合はArtifact事前準備・予約コストの回収可否、同じSourceの再利用回数、Deadline余裕・回復枠を考慮。単に`GPU utilization`最大・同一論理PoolのCPU合算最大を選ぶことはしない。公平性の改善をCost最小化で上書きしない。

### 12.4 独立Fixture（既存LBへの追加CL-01〜08、全件NOT RUN）

| Case | 試験条件・障害 | Independent Expected |
| --- | --- | --- |
| **CL-01 Cache warm/cold** | 同じTEI Go Executor digest、同じProvider Class。1回目Stage miss、2回目verified Hit。Queue等は固定 | Stageの時間/物理消費を区別し、Hit根拠なしのゼロ推定は禁止。#112の0.8/12.5秒を普遍定数にしない |
| **CL-02 Unauthorized toolchain fetch** | Worker OSに指定Go toolchainがない。Clientが任意ネットURLを与えて自動入手させようとする | RQ-029により既定拒否。承認済みToolchain ProviderやPre-staged Binaryがないなら`not eligible` |
| **CL-03 Cached bytes wrong digest** | CacheのEntry名は同じ、実Body digestが新Artifact Revisionと違う | 旧Cacheを使わず拒否／再Stage。表示名やHit申告をSource/Executable Authorityへ昇格しない |
| **CL-04 Locality vs fast compute** | Local 60秒。Remote 20秒＋8GiB/100MiB/sの新規Input転送81.92秒 | #58の固定反例で`FABRIC_NOT_JUSTIFIED`相当。Remoteを高速な候補と偽らず、Result/Verify費用込みで比較 |
| **CL-05 Partial Source closure** | TEI #2812でselected parsed Sourceのみ取得、推移Import/否定依存は不明。新Inputは同Digest名 | Full ClosureがないならBuild skip/Output受理へ昇格しない。`needs source requalification`、再実行資格は別 |
| **CL-06 Staged but no capacity** | Managed PoolでArtifact/InputのStageは成功、同じ時間窓のGPU slotは他Taskに予約済み | `stage-ready` ≠ `reservation-confirmed`。Deadline/Recovery保証なし、二重予約不可 |
| **CL-07 Low-trust prewarm** | 管理者承認のない社内PC Groupへ機密Executor/Inputを事前配布しようとする | DLG-1/Data Class/Local Governor/Transfer Budget/Provider grantを満たさず拒否。Poolの集約表示は配布許可でない |
| **CL-08 Replan and orphan stage** | A Poolに有償Stage/Cache格納、途中でB Poolへ変更、AではData Retention/Cloud支出継続 | Bへ切替成功でもAの転送/残余格納費を消さない。Provider release/retentionの実Receiptまで`unknown/outstanding` |

**実装順の上書きではない**：P0は既存[EKI #58](https://github.com/hat47x/eki-fabric/issues/58)のSuitability-0モデル（Input bytes/Cache/Throughput/Compute）に、TEI Build Artifact/Runtime ClassとLocal baselineを結ぶ読み取り専用比較を追加する。P1は現EKI P0 Input ingestとApproved Executor identityの不足を[#112](https://github.com/hat47x/eki-fabric/issues/112)で解消する。P2で初めて実ProviderのStage/Reserve/Result/Cache MetricsをE2E観測し、P3でDeadline/RecoveryのPlanner・Predictorに反映する。新Cost Engine、Container Registry、Package Downloader、Workload-specific Go CompilerをEKIへ追加しない。WP-01〜19は静的ABI試験PASSの報告、CL-01〜08は**新規実行NOT RUN**。
