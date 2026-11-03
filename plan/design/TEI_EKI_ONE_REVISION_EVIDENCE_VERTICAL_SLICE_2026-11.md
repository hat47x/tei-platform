# TEI→EKI→Provider：一改訂でつなぐ最小実行・期限・実結果の受入縦断

- 2026-11-03／Status：**他レーン実証と照合した計画／独立Expected設計。新試験は全件NOT RUN。製品実装・Owner意味受理・実EKI Worker/Provider実行の達成とは異なる。**
- 狙い：これまで増えたTEI-EKI設計（AI Job、Headless Batch、Pool-first委任、リードタイム／予算／回復、TEI Ops/Simulation、外部Action資格、X-TEA Data Script）を、新しい共通Runtime/Registry/Schedulerを作らず、**同一Source/Meaning/Artifactに束縛された最小の実行証拠**として検収できる順序へ整理する。
- 既存TEI Owner：[Batch #1130](https://github.com/hat47x/tei-platform/issues/1130)（JL-01〜07とAJ境界）、[Data lifecycle #1129](https://github.com/hat47x/tei-platform/issues/1129)（XL-01〜03）、[Delegated Authority #2499](https://github.com/hat47x/tei-platform/issues/2499)（DA-1〜6）、[Independent Oracle #2514](https://github.com/hat47x/tei-platform/issues/2514)、[Semantic Action PR #2616](https://github.com/hat47x/tei-platform/pull/2616)（AJ-01〜08/§14）、[Everything is Data PR #2667](https://github.com/hat47x/tei-platform/pull/2667)（§31/35/36と本体Runtimeの実証区分）、[#1174](https://github.com/hat47x/tei-platform/issues/1174)（Effect Fence）。
- EKI Owner：[External projection #112](https://github.com/hat47x/eki-fabric/issues/112)、[MVP #57](https://github.com/hat47x/eki-fabric/issues/57)、[Control wiring #242](https://github.com/hat47x/eki-fabric/issues/242)、[Workload Suitability #58](https://github.com/hat47x/eki-fabric/issues/58)、[Provider cluster #67](https://github.com/hat47x/eki-fabric/issues/67)、[History/Shadow #19](https://github.com/hat47x/eki-fabric/issues/19)、[Recovery fairness #155](https://github.com/hat47x/eki-fabric/issues/155)、[Queued visibility #249](https://github.com/hat47x/eki-fabric/issues/249)。
- このブランチの正本に対する参照：[Job/FlowとEKI権限](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)、[Pool-first委譲](TEI_AI_WORKLOAD_EKI_AGGREGATE_DELEGATION_2026-11.md)、[Deadline/Budget/Recovery](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)、[未解決契約Gates](TEI_EKI_AGGREGATE_DELEGATION_UNRESOLVED_GATES_2026-11.md)。
- GitHub変更は**TEI Draft文書のみ**。他レーンのPRやEKIのIssue/ADR/実行コードをこの文書から変更・採用しない。

## 1. 隠れた不一致を閉じる最小方針

**一つのJOBを同じ意味のまま3経路へ投影**する：
- **T0 Direct**：TEI生成Goバイナリまたは既存Hostが、純粋な計算を自分で実行する。EKI未導入でも成立すべき正常経路。
- **T1 EKI Native**：同じ入力・Executor/Artifact版・Expectedを保持し、EKIのTask→Attempt→Resultへ委任する。EKI内部のScheduler/Worker/Result取得まで**実際に確認できた場合のみ**「実行済み」とする。Input/Output bytesが渡せなければ`unsupported`。
- **T2 EKI→Managed Pool**：EKIは登録済みResource Delegate/Cluster Groupを選び、Slurm/Kueue/Cloud等のProviderが内部Node配置を行う。Provider予約/ExecutionRef/結果本体とEKI AcceptedResultを別観測にする。T1が成功してもT2を成功としない。

**優先するのは最小の一つの純粋計算**。複雑なAI tool chain、Data Script changed-edge、Flow/業務DB Commit、複数Pool最適化、生成AI運用操作は後続の独立Gateへ分割する。特にTEI #1129のX-TEA hook実行意味はSource-staticだけで決定できず、初回fixtureの計算を業務Sourceの真の意味へ昇格しない。EKI #112の投影SpikeでInput Artifact 8/9が422、Result取得が501だった履歴は**現在の実行能力を再検査すべき根拠**であり、この文書で解決したとは言わない。

## 2. 縦断で残す識別子は多いが、正本は増やさない

| 証拠の切片 | 帰属・束縛先 | 同一視してはいけないもの |
| --- | --- | --- |
| Source / Language / Model Revision | TEI原Source byte hash・参照時点/locator・Function/Job/Flow ID・Plugin/Operator version | Sourceの同名移動・A→B→AのABAや旧Toolchainの再利用を同じ意味受理とみなさない |
| Accepted Meaning | TEI Business/Domain Ownerの意思決定、適用Scope/粒度、NULL/missing/partial、受理・撤回・Revision | 構造候補、AI提案、Generated Codeのhashを意味の受理とみなさない |
| Build Artifact / Executor | Input dependency closure、生成器・Toolchain、Target（Go native／Java Host等）、Artifact byte digest・Function/Operation binding | `Source同じ`だけで生成器・Executor・Target間の互換性を保証しない |
| Action / Flow / Request | Context ID/Subject・Operation ID・Step ID、Payload/Scope/Data Class、not-before/target/latest-useful、Effect class、Owner/Invoker | 同じ表示ラベル・Action Offerを実行資格としない。Request対象IDとPayload内IDを不整合にしない |
| Resource/Policy | EKI DLG-1 delegate ID/class/policy revision、BUD-1物理Budget、CAP-1の有償Capacity、Provider Offer/Admission Rev、Local Governor | Offer／予測／公開`queued`／BUD残高と実GPU予約を同じにしない |
| Attempt / Provider | EKI Task ID、Attempt/Lease/fence、Provider Job/ExecutionRef、Provider receipt、実行権の期限 | EKI Lease発行＝実起動、Provider accepted＝結果取得、Lease期限切れ＝外部副作用不在ではない |
| Output / Business Effect | Result Artifactの**body bytes＋digest**、EKI ResultClaim/VerifiedResult、TEI Business Provider Commit/Receipt/独立Read | ResultClaimやEKI CompletedをTEI側の業務CommitやQuality PASSにしない |
| Evaluation / Simulation | World/Policy/Job/Estimator revision、観測対象・欠測/打切り、独立Expected/Oracle版、Scenario/Seed | 同じRuleから作ったExpected、過去のSimulation PASS、LLMの一致を現在の実Grant/実成功の証拠にしない |

上表は既存Model/Receipt/Task/Build/Observationに付けて照合する**検収用の参照票**であり、独立した中央`Job Provenance Registry`や新Credential Storeを要求しない。生成物内の論理Job IDと配布・実行単位が異なる場合もSource/Owner identityは保持し、`Build Targetを変更したから業務意味を改版した`とは推論しない。

## 3. 固定合成Fixture W-1：純粋計算の3経路を対照

**この値は業務モデルの実データではなく、テストだけの独立Expected**：

- 人が先に固定する入力 `F1={quantity:2, unitPrice:"300.00"}`、金額精度（2桁）、丸め・単位を契約として明記、Expected `amount="600.00"`。浮動小数を黙って使わない。
- 負例として`quantity=0`→0、`quantity=null`、`quantity=missing`、`unitPrice="not-a-number"`、部分Inputを、**各Input Contractで区別**する。NULLと欠落は独立のExpected（拒否／未知／Nullableを先にOwnerが指定し、実装の結果から期待を後付けしない）。
- Job/ExecutorはGo nativeを初期Targetとし、必要なSource/Toolchain/Artifact digestをpinする。これをTEIが完全なAI Job Builderを実装済みと主張する根拠に使わない。
- **T0**：同じFixed InputのDirect ExecutionとOutput body/digestを取得する。
- **T1**：EKI External APIの適切なData/Artifact経路でInput bodyを渡し、Executor digestを束縛し、Task→Attempt→Lease/Actual Worker→Output body/digest/検証を辿る。未実装エンドポイントは`unsupported`と記録し、実装があるようにfake APIを使わない。
- **T2**：一つの外部管理Poolで、Provider Job/ExecutionRef、Provider実Reservation、EKI Attemptとの相関、Output body、二重Retryしない境界を確認する。Providerの`completed`だけでT1に合格しない。
- 同じRule/Expectedを使って`T0/T1/T2`のInput→Result→Test/Oracleを比較し、結果が数値等価でも**Auth、入力の同一性、Outputの真正性、費用・所要時間**の受入資格は別で記録する。

初期では**実DB更新を混ぜない**。T0の実Go/CLIとT1のEKIのSource/Artifactを同一版で比較できた場合のみ、次のCommit/Trigger Gateへ進む。

## 4. 固定合成Fixture W-2：Data Script changed-edgeは「Jobを発火してよい」という実事実が必要

TEI Data Model/Scriptの`ready`仮想状態を想定し、`false→true`の業務Ruleを**合成例として**使用する。参照X-TEAや特定業務の受理済みMeaningと偽装しない。

- 変更前の元Fact（対象ID/Key、NULL/missing、Relationの子集合、Time Scope）と変更後のSnapshotを**同じ必要範囲/Completeness**で取得し、Rule/Operator/Dependency Revisionを固定する。子集合が0件であっても、`NOT EXISTS`や集計・参照元が何であったかという否定依存が残る。0件≠変更なし。
- Providerが業務Writeをcommitし、実Data Script導出EdgeがOwner受理のRuleと一致する**確定事実**を観測した場合に限って、必要ならTransaction Outbox/Change Eventを識別子付きで発行する。`before=true/after=true`の再評価・rollback・初回値unknownを変化Eventへ推定しない。
- EKI委任が必要なら**Event identity と Job Step/Operation ID、元FactのRevision/Commit Receipt**で重複配送と真の再発火を区別。A→B→Aや`false→true→false→true`の二つの有効Eventを同じ値digestで潰さない。
- TEIが常駐しない構成では選択済みHost/OS Scheduler/EKI等の起動Coordinatorを明示。OutboxがないProviderで厳密changed-edgeを実行できるかは独立評価し、`Execution enabled`やStudioのFlow表示だけではBackground Runnerの実在を証明しない。
- JobがEKIで完成しても業務DBのUpdate/Quality Gateが完了したとはいえない。Effect Fence/独立Read/UNKNOWNをOwner契約に従って継続保持する。

このW-2はTEI [#1129](https://github.com/hat47x/tei-platform/issues/1129)のXL-01〜03、[#1130](https://github.com/hat47x/tei-platform/issues/1130)のSJ-01〜08、[#2616](https://github.com/hat47x/tei-platform/pull/2616)のAJ-01〜08、PR #2667 §31/27.6（Scope completeness、Quality Gate）へ引き渡す。**本稿だけでTx/CDC/Outbox APIを採用しない**。

## 5. 固定合成Fixture W-3：リードタイム・費用・回復判断を同じ実Artifact上で

**完全に人工の計算例でありEKIの実Cost/性能/SLAではない。**

- テストClock：09:00に受理した純粋Job、`not_before=09:00`、`target_complete=09:50`、`latest_useful=10:00`。結果採用条件はEKI Verified Result bodyの取得、業務DB Commitを含まない。
- Opportunistic Pool Oの初回試行は準備＋転送＋計算＋Output/Verificationで20分（確率/性能は**未校正の合成値**）。Managed Provider MのFallbackは**確保＋転送＋実行＋検証で26分**、O側の失敗検出/Lease Fence/Provider照合に追加5分必要と仮定。
- 最遅Fallback判断時刻は、この人工Scenarioで`10:00−(26+5)分=09:29`。09:32にOの失敗が分かったら`09:32+5+26=10:03`となり、必要な採用期限を過ぎる。**予測に基づくAlertやDirect Hostへの変更案**は出せるが、期限延長やCAP-1支出の資格は作れない。
- 同一のGPU Classの物理Budget 60分、O実消費最大20分、M実消費30分を仮定する場合、O+M=50分で収まるが、O+M+M=80分は超過。異なるGPU ClassのGPU時間、現金、Cloud料金は単一の同種物理予算として足さない。
- 同時にJob BがMの最後のCapacityを予約したら、Oに「Budgetがある」だけではMへ切替できない。Provider ReservationとEKI BUD-1/CAP-1を別々に確認し、既存部門Fair-share/Quotaを侵害しない。ProviderのCapacity Reservationが無ければ**`fallback feasible`ではなく`fallback speculative`**。
- Pool OとMが同一Object Store/Network/Identity基盤に依存するなら、2つの独立失敗と計算しない。再試行はEffect-freeであり、Providerの内部RetryがEKIの新Attemptと重複しない条件が必要。
- EKI #58の`DIRECT`候補を必ず含める。数ミリ秒のFunctionに数十ミリ秒のControl/Decision overheadが生じる場合、Directを正常に選ぶ。

最適化の価値を実証するには、T0/T1/T2を**同一Job/同じ実結果**で比較し、実行待機/発見/処理/結果検証・実Provider使用量/外部Cloud費用・利用者の端末妨害・失敗/回復を観測する。SimulationのみのPareto優位やMLモデルの`shadow rank`から有償容量確保を推論しない。

## 6. 反証と停止条件（VS-01〜14、現時点で全件NOT RUN）

| Case | 注入する対照・入力 | 固定すべきExpected／証拠 |
| --- | --- | --- |
| **VS-01** T0 vs T1 Input | 同じ論理Function、異なるInput bytes/digest/欠損、Toolchain rev違い | Artifact/input bindingが合致しなければ拒否。同名Job/同じ出力文字列だけで互換性PASSとしない |
| **VS-02** EKI API未対応 | 旧Spikeと同等のInput Artifact/Result retrieval 422/501 | 外部APIの現実を`unsupported`として示す。モック成功を実E2Eと主張しない |
| **VS-03** W-1型/NULL | `quantity=0/null/missing`、金額不正文字列、部分Input | Expectedを事前固定し、NULLとmissingを混同しない。単なる拒否は契約適合範囲内に限定 |
| **VS-04** X-TEA virtual changed-edge | Before false→After trueとtrue→true/unknown→true/rollbackを並列 | 実Commit/同一scope/Rule Revがある場合のみChanged Event、その他はno-event／unknownを区別 |
| **VS-05** 0件/否定依存 | 子集合0件、別のProviderで子Relation追加、NOT EXISTS Rule、部分RLS | 0件=全世界空集合/依存無しとしない。差分取得不能なら`edge unknown/unsupported` |
| **VS-06** Duplicate trigger | 同じOutbox Eventを3回Relay、二つの別Txで同じ値へのEdge | 同一Eventは一つの論理Operationへ収束、異なる正当Eventを値digestだけで同一化しない |
| **VS-07** Action/Authority | UIはAction表示、Issuerの権限失効、Request対象A/Payload B、Executorの改版 | Offer/Planは実Grantではない。Host/Provider/current Policyで実拒否、実Effect=0を独立に確認 |
| **VS-08** T1/T2 Provider ACK loss | ProviderがTask受理後にACK消失、EKI Attemptがretry候補 | ExecutionRef/Operation ID照会とEffect Fenceを優先。新Provider Jobの無条件二重発火は拒否/hold |
| **VS-09** EKI Result≠TEI Business | T1のOutput body/digest verified、DB Commit unknown、部分Output | EKI結果受理とTEI業務完了を分離。Quality/Business結果がUnknownならFlow成功としない |
| **VS-10** Fallback deadline | 09:32にO失敗、残り28分、再照合5分＋M26分 | 現時計で`10:03`予想、10:00期限には間に合わない。過去の09:29切替点やBudgetから保証を作らない |
| **VS-11** Capacity/Quota競合 | Job A/Bが同じProvider最後のGPU slotを同時予約、BUD残高あり | Providerが二重Reservationを防ぐ。片方は待機/代替/期限リスク、Budget≠Capacity |
| **VS-12** 共通故障/Storm | O/Mが同Network、PCの勤務開始時一斉reclaim、Recovery Window restart | 独立故障として成功確率を過大評価しない。Retry storm/Budget WindowをResetせず、Owner/Local Governorが優先 |
| **VS-13** Service/Flow非常駐 | TEI CLI終了、EKI稼働、未登録Guardを含む後続Flow、Callback後着 | EKI Taskは続行できてもTEI業務Guardは推定実行しない。Coordinator実在とどこで停止するかを表示 |
| **VS-14** No distribution / disclosure | 同一Pure JobはDirectの方が低費用、一般利用者に下位Node数非開示 | `DIRECT`正常結果。EKIの内部Trust/Pool inventory/Topologyは予測詳細やAI prompt/SHO/Tooltipで漏らさない |

**受理境界**：VS-01〜14の独立ExpectedはOwner／実Providerの振る舞いから先に固定し、TEI生成RuleやEKI Schedulerの予測から自動算出しない。Source-static／synthetic pure／real CLI／real EKI Native／real Provider／real Business Commit／real external Consumerの各資格を**合格件数で加算せず、行ごとに確認**する。2026-11-03時点、この追加14件は**全件NOT RUN**。

## 7. 「実装する価値がある最小切片」への統合順とOwners

| Phase / Gate | 作業範囲と実証する最小の事実 | 合格の資格・Owner |
| --- | --- | --- |
| **V0 Scope/Expected固定** | W-1固定Input/Output/nullable/Source/Meaning/Artifact/Target、EKI APIの現状、既存TEI/EKI Unit/Synthetic観測を同じ表へ。X-TEA Sourceと業務秘密を新公開文書へ転載しない | TEI #2514/#1121/#1130、Buri/#1129のSource/意味Owner。合成TargetのPASSを実業務へ昇格しない |
| **V1 Direct Go実行** | TEI単一Go Targetを実起動、Byte/Toolchain/Output digestと同じ独立Expectedを確認、異版拒否を観測 | TEI既存Go/CLI/Build evidence、#2616 OB-07A、#2667の実CLI。ただし別PRの旧HEAD PASSを本HEAD合格に転用しない |
| **V2 EKI Native Job I/O** | Public API→Input Artifact body→Digest/Executor一致→実Attempt/Lease/Worker→Result body/digest/verificationを一つのTask/Attemptで相関 | EKI #112/#57/#242。422/501、Worker未配線なら`unsupported`と止める |
| **V3 Provider Pool委任** | Managed HPC/Cloudの一つのPoolでProvider Reservation/ExecutionRef/ResultとEKI Taskを紐付け、ACK loss・二重Retryと取消を対照 | EKI #67、Task State Model、DLG-1/BUD-1/CAP-1。内部NodeをEKIに複製しない |
| **V4 非常駐Business Trigger** | TEI #1129 XL-03の実旧Driver/新Provider対応、Before/After/Commit/Outbox→EKI、実Effect Receiptと後続Flowまでの同一Owner判断 | TEI #1129/#1174/#2499、#1130。実Tx/外部Effectの採用資格はV1〜3と独立 |
| **V5 時間/費用/実成果** | Direct/EKI/Providerを同一W-1（後にW-2）でBenchmark。Deadline/回復/Quotaの合成反証→実失敗測定→Shadow calibrationの順 | EKI #58/#19/#50、TEI Ops/Simulation。再計画/AI Reviewは当該証拠の範囲でのみ評価 |

**スケジュールではなく依存Gate**。V4がV2後でなくてもTEI Provider単体で検証できる部分は分離して進めてよいが、`TEI Event → EKI Task`のE2Eを名乗るには双方の実証が必要。V1が通ってもV2/V3にWorkerがいるとは限らず、V2が通ってもV4の業務意味が受理されたわけではない。

**実装差分を出す順番**：最初は現行EKI #112の未対応Contractと既存TEI #1130のArtifact/Result Referenceとの比較表だけを作る。何が足りないかを確定できた場合に限り、既存API/Adapter/Result ingestionへ**最小の差分**を出す。新しいJob Controller、EKI Admin Portal、AI Runtime、常駐TEI Host、中央Provenance Registry、独自Trace DB、価格/通貨EngineはこのV0〜V2の前提にしない。

## 8. 監査可能な状態と次の判断

- **Confirmed source/design**：EKIの委任/期限・予算/Local Governor/不明Effectの製品要求・一部Pure Core、TEIのGo/CLI・Action Surfaceの限定Go実証、X-TEA旧Codeの静的Hook/source構造、各PR/Issueが実施済みと報告する局所試験。
- **Proposed**：W-1〜3とVS-01〜14、三経路T0/T1/T2の完全な固定Source/Artifact/Provider比較、Job Evidenceの突合せ、期限・回復計画の単一Reference。正式External SchemaやShared Runtime ABIではない。
- **Not demonstrated**：EKIのTask I/O Full Byte E2E、TEI→EKI→Managed HPC/Cloud、TEIのBusiness Commit/Outbox、AI Job Builder全機能、MDM Worker Pool実管理、Modeler/SHOの総合動作、期限成功確率/支出削減実績、Owner受理。
- **Stop**：見かけ上のResult successで実bodyがない、旧/新Artifact revisionが混在、外部Action実行範囲とRequest対象が異なる、Source未取得/partialの業務RuleでChanged Edgeを推定、資源を実予約できないのに保証、Effect unknownで自動再実行、Data Class違反、MDM/Local Governor失効、予測値をAdmin/Lease権限へ昇格。このときも許された範囲のread-only診断は維持する。

**採用のOwner質問**：T0/T1/T2のどのWorkload classを初期製品Gateとするか、nullable/rounding/qualityの独立Expectedの受入先、最低限のResult body取得とHash/Verifier相関、Provider ExecutionRef/Lease/Cancel不明の最小対照、Ownerが期限として測りたいOutcomeはEKI結果か業務Commitか。答えが未受理なら、正式Contract/ABIを推測で採用せず**V0の比較票とsynthetic fixtureまでに留める**。
