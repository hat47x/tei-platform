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
| Outcome / feedback | EKI Verified ResultとTEI Business Receipt、实际Queue/耗時/失敗/転送/利用量/費用/干渉を分離して記録、予実/校正へ | 最適化Proposal/AI一致を正式なAssurance・Grantに昇格しない |

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

## 8. 最小の独立Conformance（LB-01〜16、全件NOT RUN）

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
| **LB-12 Provider ack loss** | EK​​IからSlurm Job投入、Providerが受理後応答断。Deadline逼迫 | Provider ExecutionRef lookupを優先。再提出はCorrelation/Idempotencyがなければ`unknown/held` |
| **LB-13 Direct wins** | Local/Existing 5ms、EKI Decision/Dispatch往復10ms以上の同等仕事 | `DIRECT`正常判定。高価なOptimizer探索で実Jobより遅くなるのは不合格 |
| **LB-14 Fair-share override** | 部門Aが`urgent`と宣言し部門Bの保証枠を横取りしようとする | User preferenceだけでEKI Admin Fair-share/BUD-1を超えない。期限が危険でも無権限支出不可 |
| **LB-15 Forecast disclosure** | Pool内部Node非開示、Clientへ詳細な確率分布/下位Site数を返そうとする | RQ-018/058のDisclosureで抑制。最適化内部モデルがNode秘密を漏らさない |
| **LB-16 Sim→Act mismatch** | SimulatorはM available、実CAP-1承認時にProvider capacity消失 | Simulation/AI PASSでLeaseやProvider Reservationを発行しない。新しいAuthority検査→`unavailable`/再Plan |

ExpectedはPlan/Estimator/LLMと**独立な固定Oracle**を用意し、実故障分布・異種Provider／実時刻/実消費は実測で別受入。合成Score/ScenarioのPASSを本番Deadline達成率へ昇格させない。失敗Caseは[既存TEI #2514](https://github.com/hat47x/tei-platform/issues/2514)のOracleとEKIの既存Simulation/Conformanceへ接続し、新Runnerや別検証Registryを作らない。

## 9. 既存Issue/ADRとの非重複・導入順・保留判断

| Phase | 最小成果物／検査 | Ownerと受入Gate |
| --- | --- | --- |
| **0 契約照合** | TEI Job/Batchの時刻・Retry/Checkpoint/Effect/Result Spec、EKI RQ-057/058とDLG-1/BUD-1/CAP-1のMapping、Direct-vs-Broker固定Oracle | TEI #1130/#1121、EKI #108/#109/#140（既存要求）、#58 |
| **1 実Job Wire** | TEI純粋Job1件のInput/Output bytes/Executor digest→EKI Task/Resultの受渡し | EKI #112/#57/#242。**現422/501が未解消ならここで止める** |
| **2 Deterministic time/budget** | Pool/Provider 2候補のQueue/Transfer/Run/Verify所要とBUD-1/CAP-1・Local Governor/資格を使い、Deadline-aware feasibility/Direct comparatorを決定論で出す | EKI #58/#67/#108/#140。Reason/Estimator Revisionと独立Expected |
| **3 Recovery-aware planning** | 2故障／共有Site障害／費用枠残余／Retry・Checkpoint・Provider ack unknownをSimulatorで評価、実Provider/WorkerのRetryを実測 | EKI #52/#354、#19/#48、実Provider/Lease/Result Owner |
| **4 Shadow calibration** | Pool×Runtime/Workload Class×Time Windowの実Outcome、打切り・ドリフト/相関を分けたHoldoutとCalibration。Actual Placementへ影響させない | EKI #19/#50。学習モデルのAuthority昇格を行わない |
| **5 TEI Ops／Pilot** | 期限/予算/実結果の可視化、What-if・AI Critical Review、Fallback原因/Plan diff、Deadline Miss/Recovery SLO/物理Cost・顧客価値 | TEI #1124/#2514/#2620、EKI #125/資源管理Owner、Business Owner |

**Metrics（提案）**：`deadline_miss_rate`、`on_time_verified_completion_rate`（母集団と打切りを固定）、`p50/p95 end-to-end lead-time`、`p95/p99 recovery elapsed`（観測できた範囲と母集団）、`escalation rate/time`、`remaining recovery budget`、`deadline_at_risk lead warning time`、`compute/transfer/verification physical cost per accepted result`、`provider admission failure`、`decision overhead`、`owner-interference`、`unauthorized allocation=0`、`unknown-effect duplicate retries=0`。構成・入力・環境・失敗母集団が違う数字を単純比較しない。

**明示的保留**：TEI/ EK​​Iの正式DSL/APIの名前、確率閾値や統計推定器、分散最適化Solver/学習Model、Cloud料金の自動入力、SLA/保証、recovery budgetの物理表現、Hedgeと独立性、Ownerの全自動再配置許可。**現行EKIのAuthority/実Providerの完了資格・持ち主の拒否権は不変**。どのアルゴリズムでも変更する前に既存Issueと独立TestでOwner判断を行う。

**停止条件**：実Artifact/Result不可、必要なProvider/Trust/Data/Deadline/Consent不明、Source stale/partial、現在のCAP-1/Provider予約なし、物理予算不足、実External Effect Unknown、故障相関不明で必要な回復確率を保証できない、Fallback開始時の必要時間超過、Estimator校正不足、Solver overhead/費用逆転、一般ClientへのPool秘密漏洩。読取可能な状態の説明・期限リスクWarningは維持するが、実権限なしで高額Scaleoutや危険な再試行を行わない。
