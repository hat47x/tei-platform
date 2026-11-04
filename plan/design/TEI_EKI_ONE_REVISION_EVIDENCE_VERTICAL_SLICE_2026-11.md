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

**優先するのは最小の一つの純粋計算**。最新EKI Sourceによる再資格化は本稿§12が優先し、旧#112/#242の実装不足表現は歴史的な報告として読む。複雑なAI tool chain、Data Script changed-edge、Flow/業務DB Commit、複数Pool最適化、生成AI運用操作は後続の独立Gateへ分割する。特にTEI #1129のX-TEA hook実行意味はSource-staticだけで決定できず、初回fixtureの計算を業務Sourceの真の意味へ昇格しない。EKI #112の投影SpikeでInput Artifact 8/9が422、Result取得が501だった履歴は**現在の実行能力を再検査すべき根拠**であり、この文書で解決したとは言わない。

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

## 6. 反証と停止条件（VS-01〜14、追加VS-15〜22は§11、現時点で全件NOT RUN）

| Case | 注入する対照・入力 | 固定すべきExpected／証拠 |
| --- | --- | --- |
| **VS-01** T0 vs T1 Input | 同じ論理Function、異なるInput bytes/digest/欠損、Toolchain rev違い | Artifact/input bindingが合致しなければ拒否。同名Job/同じ出力文字列だけで互換性PASSとしない |
| **VS-02** EKI API再資格化 | 旧SpikeのInput ingest 422／Result GET 501と、最新のConfig有無・Output GET・Output bodyの各場合を分離 | 実際に存在するAPI機能と旧歴史的Gapを区別し、現行未対応Input ingestは`unsupported`。モック成功を実E2Eと主張しない（§12） |
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

**受理境界**：VS-01〜32の独立ExpectedはOwner／実Providerの振る舞いから先に固定し、TEI生成RuleやEKI Schedulerの予測から自動算出しない。Source-static／synthetic pure／real CLI／real EKI Native／real Provider／real Business Commit／real external Consumerの各資格を**合格件数で加算せず、行ごとに確認**する。2026-11-03時点、この追加14件は**全件NOT RUN**。

## 7. 「実装する価値がある最小切片」への統合順とOwners

| Phase / Gate | 作業範囲と実証する最小の事実 | 合格の資格・Owner |
| --- | --- | --- |
| **V0 Scope/Expected固定** | W-1固定Input/Output/nullable/Source/Meaning/Artifact/Target、EKI APIの現状、既存TEI/EKI Unit/Synthetic観測を同じ表へ。X-TEA Sourceと業務秘密を新公開文書へ転載しない | TEI #2514/#1121/#1130、Buri/#1129のSource/意味Owner。合成TargetのPASSを実業務へ昇格しない |
| **V1 Direct Go実行** | TEI単一Go Targetを実起動、Byte/Toolchain/Output digestと同じ独立Expectedを確認、異版拒否を観測 | TEI既存Go/CLI/Build evidence、#2616 OB-07A、#2667の実CLI。ただし別PRの旧HEAD PASSを本HEAD合格に転用しない |
| **V2 EKI Native Job I/O** | Public API→Input Artifact body→Digest/Executor一致→実Attempt/Lease/Worker→Result body/digest/verificationを一つのTask/Attemptで相関 | EKI #112/#57/#242。現mainはoptional Dispatch/実Process test/条件付きOutput GETを持つため、**RE-01〜08でConfig別に再資格化**。Input ingest未実装やProduct-only result未結合なら該当境界で止める |
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

## 9. Cross-lane確定差分：Job意味・Build単位・Launch単位を別正本にする（2026-11-03追補）

他チャットで進んだ[TEI #1130のBU-01〜09](https://github.com/hat47x/tei-platform/issues/1130)、[Build artifact多論理Subject #1097](https://github.com/hat47x/tei-platform/issues/1097)、[Application別Go実配備E1 #1167](https://github.com/hat47x/tei-platform/issues/1167)を、この実行縦断へ接続する。**標準はGo単一バイナリ**、マルチビルド設定は配備・起動・更新の実理由が生じてから導入する。業務上のSystem/Subsystem/Function/Flowは意味単位、`BuildUnit/DeployUnit/Invocation Entry`は技術単位であり、互いに恒等写像ではない。

| 区別 | 正本／現在の証拠 | この縦断の合格境界 |
| --- | --- | --- |
| Logical Job/Function | TEI Ownerが受理するJob/Function/Flow IDと版。X-TEA旧意味を合成式から推定しない | 同じFunctionをT0/T1/T2で呼び、同じInput/Expectedを適用できる。Job名の一致だけでは不足 |
| Build Intent／Physical Artifact | TEI #1130 BU、#1097 Shared Build Subject、#1167 Application別実Source Closure。**Go単一バイナリ内に複数Entryが存在し得る** | `Job ID → selected Entry/member/Artifact digest → toolchain/Source closure`を版付きで辿れる。共有Binaryだから実行Grantを共通化しない |
| Deploy/Launch Target | 既存Go CLI/HTTP Host/cron/Cloud service/HPC Executor/EKI Taskの選択 | Entryが実Artifactに存在し、Targetが対応するOS/arch、Runtime、Input/Output契約を満たす。Build選択をRuntimeに隠して再解釈しない |
| Actual invocation | OS Cron/Service Manager、既存Host、EKI Adapter/Providerの現Credential/Capability/Source & Meaning revision | 起動トリガー・Job Dispatch・実Args/Payload・起動時資格・実Worker/Process・Result Body/Exit/Receiptを別列で証明する |
| Selected build outputs | 一つのBuildが全Functionを含む場合、後から`order_only`等の別Targetを必要に応じ追加 | 影響外Dependencyが**物理的に除外**された証拠は`go list -deps`＋`go build`＋必要に応じSymbol/Byteを確認した場合に限る。起動時`-application`選択だけで除外を主張しない |

**V0票追加**：`logical_function_id, flow_step_id, entry_selector, build_intent, build_profile, target_runtime, artifact_member, artifact_digest, source_dependency_closure, invocation_transport, current_authority_issuer, operation_id, actual_result_digest`。あくまで既存発行Source/Build/Host/EKI Evidenceの**照合用フィールド候補**であり、新しい公開Manifest/Dispatcher/Registryの要求ではない。Source/Artifactが同じでもEntrypointの違いと現在Grantを検査する。TEI #1130 BU-09の`Job v1 queue → Binary v2 deploy → retry`では、再開時の旧JobのPinを維持し、単に最新バイナリに存在する同名Functionへ付け替えない。

**結果分類の既存証拠**：[TEI #2747](https://github.com/hat47x/tei-platform/issues/2747)はStudio Plan/ExecuteCommandの**参照実行**を統合済みだが、実業務Provider/DB Commitではない。[Draft TEI #2780](https://github.com/hat47x/tei-platform/pull/2780)には`Committed/Replayed`の区別がCommand Result表面では失われる一方、CommitUncertainErrorの型付き原因は残るという**限定Go Characterization（PR内報告はexact-head 4/4 PASS）**がある。だから`Studio Command Committed=true`／`Go Command Result{}`／`EKI Result Completed`／`Business Provider Commit`を同じ肯定値へ正規化しない。#2780の限定Go PASSも、本PRで実DB/EKIが接続された証拠ではない。

## 10. 品質・取得範囲・権限をリードタイム計算へ接続する（2026-11-03追補）

他チャットの[横断PR #2667 §27.6/§27.7](https://github.com/hat47x/tei-platform/pull/2667)、[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)、[Transitive record visibility reference #2781](https://github.com/hat47x/tei-platform/pull/2781)、[Semantic Action #2616 §14](https://github.com/hat47x/tei-platform/pull/2616)により、EKIで早く計算が終わるだけでは**業務に必要な母集団について安全に結果利用できる**とは限らないと判明する。目的が「期限内の使える結果」である以上、計算終了と、取得・検査・開示・業務利用の資格を分ける。

**品質/Scopeの4集合**：

| 集合／Scope | 定義のOwner | 最小反証 |
| --- | --- | --- |
| `required_population` | TEI Business/Quality Meaning Owner：どの行/期間/部門/Data Classで品質検査と業務結果が必要か | 要求が全社で、取得は一部門だけ。品質良好を全社合格へ昇格不可 |
| `authorized_read_scope` | 実Host/Identity/Provider Grant＋組織のvalid/known time | ViewerのRLSによる非表示を、Dataの不存在/0件とみなさない |
| `actually_acquired_scope` | 実Provider physical Read/QAPI、Snapshot/Watermark、Field/Relation/Source Version/partial | 必要Readを宣言しても実未取得であれば`unknown/partial`。Providerが非公開Fieldを実読んだ事実は隠せない |
| `permitted_disclosure_scope` | Consumer/Roleと実Disclosure Authority、出力/Tooltip/Count/Logs/LLMへの開示制約 | TEI Studio/SHO/LLMへ表示されないだけで、実Providerの読取り許可を得たとみなさない |

品質の要件はTEIが分類する：**Required Online**（Business Effectの前に現在資格を検査）、**Bounded Deferred**（所定Deadlineまでに検査をJobで実行する）、**Advisory/Best-effort**（警告と未検査を区別）、**Lean/Sampled**（全件ではなく対象Subsetでの保証）。これらは**UI装飾ではなく、結果の採用範囲・締切・実行可否の意味条件**である。新Quality Engineは作らず、既存Data/Function/Batch/Oracleから生成し、必要な部分だけEKIに計算委任する。

**時間契約**：`TEI Job request deadline`と`Business Quality Deadline`が異なることがある。例としてEKI Task Result bodyを10:00までに取得しても、必要な全体のQuality Auditが10:05までに合格しないなら、業務側のQuality PASSや「期限内に採用完了」を主張しない。一方、Advisory/LeanとしてOwnerが許した結果は、その**制限されたScope**でだけ利用可能と表現する。明示的なOwner受理なしに`Required Online`をEKI Deferredへ変えてはならない。

**実行時性能との整合**：TEI RuntimeのすべてのRequestで全Relation/全期間を検査しない。Required Onlineは最低限の安全なHost/Provider実行資格・Target/Period/CAS/Effect Fenceを保つ。全社・時系列・外部横断の検査は、Ownerが許す`Deferred/Advisory`として独立Batch Jobへ配置可能。ただしCommit unknownや実Access Grant不明を高速化のために省略・緩和しない。検査の配置を変更する場合も、Source/Owner受理/Scope/Deadline/Artifact/Result Receiptを維持する。

**AIレビュー**：AIが生成するCoverage/Diffや異常検知の候補はStructural Candidateであり、実Data Providerの取得範囲やSecurity Grantではない。EKIのResource Poolへの配備がどれほど効率的でも、`coverage unknown`・`source stale`・`owner not accepted`は高品質・実行資格へ昇格しない。生データをAIに送るかは`permitted_disclosure_scope`で独立審査し、単に画面に表示できるという理由では許されない。

## 11. 追加の独立Expected（VS-15〜22／全件NOT RUN）

| Case | 対照Fixture | 成立するExpected・停止線 |
| --- | --- | --- |
| **VS-15 単一Binary/Entry拒否** | 同じGo BinaryにJob-A/BのEntry、CallerにはAのみ許可 | BのCodeがPhysical Artifactに含まれても`Run B`を拒否。Build同居とRuntime Grantは別 |
| **VS-16 Build Unit分割・再開** | Job v1 queue後にArtifact v2 deploy、Flow StepのMeaningは同じ、Executor byte変更 | 旧Runを黙ってv2へ再束縛しない。Source/Artifact/member Pin・Provider Capability/旧Resultの採用条件を比較 |
| **VS-17 Artifact/Dependency部分再生成** | 未関連Functionの変更、選択済Go Entry・Shared Rule変更の差分を比較 | 依存閉包に従って再Build/再利用を区別。最小Buildによる物理除外は実Go dependencyの証拠が必要 |
| **VS-18 Studio Reference Committed** | Studioの参照Commandが`Committed=true`、実DB Providerなし | 参照実行でありBusiness EffectのCommit Receiptとしない。EKI Job実行可能という意味もない |
| **VS-19 Replayed／Commit unknown** | Operation Oに対しCommitted→Replayed、別TrialでACK消失 | CommandResultの表示同値から二度書いたと断定せず、typed cause/実Owner Lookup/Receiptで判定。Unknownなら無条件再提出禁止 |
| **VS-20 Quality required vs acquired** | 必須100行、RLSで40件取得、取得された40件は品質PASS | `40/40 inspected`は`100/100 business PASS`ではない。Quality Coverage/非開示/不明を区別 |
| **VS-21 Deferred Quality Deadline** | EKI Result 09:59 Verified、Quality Audit Deadline10:05、Auditは10:08又は取得不明 | EKI期限内完了とBusiness Quality期限達成を分離。Owner許可なしにRequired検査をDeferredへ変更しない |
| **VS-22 AI・Disclosure/Perf** | AIがGraph非開示ノード数/生ログを求め、性能重視でFull Scope検査省略を提案 | `permitted_disclosure_scope`を守り、最低限の実Host Grant/CAS/Effect Fenceは省略しない。Deferred化はOwnerにより採用された範囲のみ |

**採用・停止**：VS-01〜32はあくまで本稿の独立Expected候補。PR #2667/#2616/#2780/#2781の既存試験と本稿の試験を**重複成果と数えない**。V0にはBuild/Entry/Scope/Qualityの受入票を加えるが、V2（EKI Native I/O）が未達なら本稿全体を`TEI→EKI製品実証完了`と呼ばない。Source-static/限定Go Characterization/合成Scope/独立Provider Receiptはそれぞれ異なる証拠資格である。

## 12. 現行EKI main再照合：旧Spike/Issueの未実装判定を再資格化する（2026-11-03）

**今回のSource比較は、EKIの旧検証基線`c0776fdd1be9c6e83d9e0ae803f9ff650ee7def1`（Issue #242の参照）から現`main`まで681コミット先行・差分300ファイル（GitHub比較APIによる取得範囲）**。したがって**#242の「実配線なし」／2026-10-19 TEI投影Spikeの「Result 501・Worker未実装」／EKI #112の422/501は、当時の実行結果としては正しいが、現mainの全Profileに対する現在の確定判定ではない**。コード存在を製品の実動PASSへ自動昇格するのも同様に誤り。

| 機能／検査対象 | 現`main`でSourceから確認できる事実 | 実行資格と残るGap |
| --- | --- | --- |
| **Task受理・実行配置** | [`internal/control/service/run.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/service/run.go)で`admission.Config.Results=store`、`cfg.Dispatch != nil`なら`startDispatch`。[`dispatch.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/service/dispatch.go)が配置署名／Lease予約・発行／WorkOrder署名を組み合わせ、Tickループを実行 | **Source implemented/optional**。`dispatch=nil`ならTaskを進行させない。署名鍵・Trust Domain・ExecutorApproval設定と各Provider前提の実起動、Capacity/Lease/Recoveryを個別確認。全Profile常時動作やT2 HPC対応の証明ではない |
| **実Agent・Claim** | [`conformance/process/http_submit_agent_test.go`](https://github.com/hat47x/eki-fabric/blob/main/conformance/process/http_submit_agent_test.go)に`TestHTTPTaskSubmissionIsExecutedByTheAgent`。POST Task→Dispatch→Agentが`/bin/echo`相当固定Commandを実行→Claim記録までを別OSプロセスで試す**テストコードが存在** | **Test-profile fixture exists／このターンのexact-head Go実行はNOT RUN**。同テストは`ADMITTED→READY`を試験からSQLite CASで明示的に進め、結果についても試験コードが`resultacceptance.Accept`を直接呼ぶ。製品の通常設定だけでAdmissionから結果受理まで無人E2Eがつながった、と扱わない |
| **Worker Output body** | [`conformance/process/execution_body_test.go`](https://github.com/hat47x/eki-fabric/blob/main/conformance/process/execution_body_test.go)は同AttemptのClaimとControl側に保存されたbody bytes/size/digestを照合するテスト。[`service/run.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/service/run.go)でmTLS Worker+`artifacts_dir`があると受入側Storeへ接続 | **Body受信のプロセス試験コードあり**。TEI-generated ExecutorのInput受渡し、Client結果取得と同一Taskの全工程、長期Retention、Cloud/別Host実配備は別Gate |
| **Client Result GET** | [`handler.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/clientapi/handler.go)の`GET /v1/tasks/{id}/result`は`ResultsEnabled`で分岐。[`run.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/service/run.go)は`Results:store`を渡す。AcceptedResultありなら200、未受理／未終端等は409など | **Result APIは条件付きで実装済み**。GETで200を返せる設計と、通常Dispatch/Agent経路が独立VerificationからAcceptedResultを生成することは別。旧Spikeの501を現`main`で恒久的な仕様としない |
| **Client Output Artifact GET** | [`clientapi/artifact.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/clientapi/artifact.go)が`GET /v1/artifacts/{digest}`のOwner/期限/撤回判定とbody streamを実装。`handler.go`の`artifactsOf`はArtifactStoreのMetaを使ってResult内の可取得Artifactを投影 | **`Worker.ArtifactsDir`と格納されたMeta/bodyがあるProfileでは条件付きで取得可能**。設定無しでは501、未登録/失効/Owner違いは規定の拒否。Artifact bodyを持つことと業務Resultの受理は別。Client APIのREADMEに残る「全Artifact GETは501／Result artifactsは常に空」は現在のhandler実装と**文書ドリフト** |
| **Client Input Artifact ingest** | [`handler.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/clientapi/handler.go)は`POST /v1/artifacts/ingest-intents`、`PUT /v1/artifact-uploads/{id}`を`routeNotImplemented`へ送る。[`spec/external/openapi.json`](https://github.com/hat47x/eki-fabric/blob/main/spec/external/openapi.json)にはこれらの外部Contractの項目が存在 | **現在Sourceで未実装**。TEIが送る任意Input bytes/digest/権利のあるArtifactを、この入口から登録できると主張しない。W-1/T1の最初の未解決外部境界 |
| **Executor binding** | [`dispatch.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/service/dispatch.go)は起動設定`Dispatch.ExecutorApproval`を使用し、`ExecutorDigest`をWorkOrder発行に渡す | **単一実行体のFixtureでの拘束と、任意のTEI Function/Jobごとに受理済みExecutor digestを束縛できることは別**。Client任意指定はEKI #112の既存契約・管理者承認・WorkOrderの実検証と対照する。Callerが`effect_mode:none`と偽装しただけで安全なExecutorとしない |
| **Trust Bundle read failure** | [`service/run.go`](https://github.com/hat47x/eki-fabric/blob/main/internal/control/service/run.go)の`currentTrustBundle`は`ErrNotFound`のみ互換未導入、その他の読取失敗は`error`を返す実装へなっている | **EKI #467の旧Source-static不具合を現実の全経路の未修正事項と断言しない**。実mTLS Handlerがerrorをfail-closedに伝播するかの実負例、Issuer fingerprint #466、Agent更新競合は別に検証 |
| **Managed HPC/Cloud/PC Pool** | 上記は主にEKIの直接Worker／固定Command Test-profileのSourceと限定プロセス試験 | **T2実HPC/Cloud/MDM連携・Resource PoolのProvider Reservation／経済圏の複数Org調停・端末OSの実使用感は依然未証明**。EKI #67／#115／#125等の専任Ownerへ引き渡す |

**Sourceを見直した結果の優先順位の修正**：従来の「Worker/Result実装がないので何も始められない」ではなく、**現在あるTest-profileの実Job経路を正しく再利用し、試験用CAS・Result Acceptanceの手動呼び出しとInput ingestの未実装を切り分ける**。新しくGeneric Job Engine/Registryを立てない。

### 12.1 次に実行すべき最小Process受入（RE-01〜08、全件NOT RUN）

| Test | 用意する切片 | 独立Expected／失敗時の判定 |
| --- | --- | --- |
| **RE-01 Config off** | `Dispatch=nil`、Client POST、`Worker=nil` | 受理できても実Attempt/Lease/WorkOrder/Resultが生成されたとしない。`queued`を期限保証へ変換しない |
| **RE-02 Config on** | `Dispatch!=nil`・署名鍵等を明示・mTLS Worker・固定Executor | 署名済みPlacement/Lease/WorkOrderと実Agent Claimの同一Task/Attempt関係を観測。**試験側のADMITTED→READY補助CASの有無**をログで記録 |
| **RE-03 Product-only acceptance** | RE-02を元に、試験コードが`resultacceptance.Accept`を直接呼ばずClient GETを観測 | 200＋受理済Result/出力digest/bodyまで製品が自力で到達すればその条件を記録。409/保留なら未結合。**synthetic verifierを本番独立検証と称しない** |
| **RE-04 Output GET conditions** | mTLS Worker+`ArtifactsDir`設定あり／なし、Metaあり／なし、期限切れ、別Principal、撤回済 | 取得可能時だけbody/digest/size一致。501/404等のOwner/設定・保持条件を別に固定。Backendに保存された単一Fileは実External Artifact ingestionの証拠ではない |
| **RE-05 Input ingest** | `POST /v1/artifacts/ingest-intents`と`PUT /v1/artifact-uploads/{id}`の現APIでW-1の入力bodyを渡す | 現Sourceでは501想定。`unsupported`を明示し、Output GETが動くことをInput Uploadが動くことへ読み替えない |
| **RE-06 Per-Job Executor identity** | 登録済Executor A/Bの別digest、Client Task要求とDispatch Configの現選択、旧Job v1 vs Binary v2 | 同名Function/管理者の静的`ExecutorApproval`だけでTaskごとのDigestが一致すると仮定しない。Mismatch/未登録なら拒否、古いJobを新版Executorへ黙って替えない |
| **RE-07 Current Trust failure** | 最新Trust Bundle未登録・読み出し故障・旧Issuer失効の3種、直接Worker mTLS | 読出し故障を`not configured`に変換せず停止/拒否。旧Source-static報告を現実の再現証拠なしで`未修正`としない |
| **RE-08 Test-profile vs production** | Loopback test profileの固定`/bin/echo`と、別Hostの管理クラスタProviderを対照 | 前者のGo Process test PASSは後者のProvider Reservation・再実行制御・Assurance/SLO／業務Effect成功の証明ではない。EKI #115の隠れRetryは引き続き評価 |

**証拠資格**：このターンで確認したのは**接続GitHub現`main`のGo Source/テストコード/READMEと旧Issue/Spikeの差分**であり、RE-01〜08の実Goテスト、現mainのexact-head CI、TEI生成Go→EKIの実Input/Result全件結合、実HPC/MDM/Cloud/EKIプラットフォームの運用は**NOT RUN**。試験コードの存在をGo実行済みと表現しない。

### 12.2 既存計画・Ownerに対する最小修正

- [EKI #112](https://github.com/hat47x/eki-fabric/issues/112)：**Input ingestが現在も`routeNotImplemented`、Client Executor要求・Result provenanceとのギャップ**へ焦点を絞る。Output GET/Client Result GET/Dispatch全般の「一律未実装」は撤回。古い422/501は発生当時のSpike事実として保持する。
- [EKI #242](https://github.com/hat47x/eki-fabric/issues/242)／[#57](https://github.com/hat47x/eki-fabric/issues/57)：**現在はoptional Dispatch Loopと実Process Testがある**。残存課題は手動`ADMITTED→READY`、手動`ResultAcceptance.Accept`を含まない本当のProduct-only縦断／持続・運用Profileと証拠を整理すること。現行Issue記述の旧参照`c0776fdd`の未配線分析を最新実装の不在証明にしない。
- [EKI #465](https://github.com/hat47x/eki-fabric/issues/465) closed／[#467](https://github.com/hat47x/eki-fabric/issues/467) open：閉じたIssueと最新Sourceに改善の見える経路を先に調べ、過去不具合の採番のみから未実装と断定しない。Current Runtime・TLS・Worker Effectへの適合は別受入。
- [TEI #1130](https://github.com/hat47x/tei-platform/issues/1130)／[#2616](https://github.com/hat47x/tei-platform/pull/2616)／[#2667](https://github.com/hat47x/tei-platform/pull/2667)：W-1の`Source/Artifact/Entry/Input/Expected`を固定し、**まずRE-02〜06と一つのDirect Go T0の独立比較**を行う。実Provider/Business Commitへの拡張はそれぞれのOwner Gatewayを通る。
- **適合完了の判定**：`Source implemented`／`test source exists`／`actual Go test PASS on exact HEAD`／`real process in test profile`／`product-only process no manual setup`／`actual TEI Job/Result body`／`managed provider pool`／`business commit/owner acceptance`を別列で管理。欠落がある工程から先へ「完成」を伝播しない。


## 13. TEI RunPlan実装とEKI Attemptを接合する際の再開・部分結果の境界（2026-11-04）

**直近の実装を踏まえた再評価。** TEI [#2798](https://github.com/hat47x/tei-platform/pull/2798)/[#2802](https://github.com/hat47x/tei-platform/pull/2802)/[#2805](https://github.com/hat47x/tei-platform/pull/2805)/[#2807](https://github.com/hat47x/tei-platform/pull/2807)はclosedで、現`main`の[`internal/execution/runplan.go`](https://github.com/hat47x/tei-platform/blob/main/internal/execution/runplan.go)には`RunPlan`が`semantic.RunPlan`のOperations順に`StepFunc`を呼び、`runlog.File` Journalに完了Stepを記録し、中断時に既完了Stepを飛ばす実装がある。[`internal/runlog/runlog.go`](https://github.com/hat47x/tei-platform/blob/main/internal/runlog/runlog.go)は`RunKey(planID,keyValues)`、`Attempt/FenceEpoch`、`Owns`を持つ。実行結果は`StepFunc`が返した`resolved/unresolved`で判定する。**このSource上の実装は前稿の「Headless Batch未実装」という一般化を修正するが、実Provider業務Effect・外部EKI成功・自動安全復旧の証明ではない。**

**重要な未証明境界：**
1. **Stepの意味・実行資格**：`RunPlan`自体はOperation IDを`StepFunc`へ渡す汎用進行器。Stepが何を実行したか、誰がCurrent Grantを与えたか、External ProviderがCommitしたか、Source/Artifact/MeaningのRevisionが同じかは`RunPlan`が保証しない。`StepResult.Status="resolved"`は業務Receiptではない。
2. **Journalの永続性と競合**：`runlog.File.Append`はJSON Lines追記であり、`Last`を読んで`Decide`する経路に「同時別プロセスが同じRunKeyを開始／再開しない」ためのCAS/transactionは**Sourceでは確認できない**。存在する`Owns(last,attempt,fenceEpoch)`も各実Effectの直前にRunnerから検査される実装ではない。**Fence値が存在する＝外部効果が古いAttemptから拒否される、ではない**。単一Caller向けの参照実装と分散実行契約を別に扱う。
3. **Crash window**：`step(ctx,operation)`の実ProviderがCommit済みでも、`journal.Append`前にCrashすると、そのStepは未完了と見える。再開時のskip規則だけでは二重Effectを防げない。事前に業務OperationID/Effect Fence/Idempotency/Provider Lookupの契約が必要。`Append`済みでもOutput bytes・品質検証・Business Receiptが無ければEKIに`completed`を送れない。
4. **版と順序**：現在のRunKeyはPlanID＋Key値であり、`runlog.Record`の主な永続情報は`PlanID,Key,Attempt,FenceEpoch,Status,Steps`。**Plan Definition/Source/Executable ArtifactのRevisionやStep順序・契約はRecordで固定されていない**。再開時に同一PlanIDのStep集合/順序が変われば、`Steps`照合で旧Stepだけを飛ばす一方、新Rule/引数で残りが走り得る。再開の前にSource/Meaning/Artifact/Profileの適合を既存Build/Receiptから照合し、不一致時は安全側停止・明示Migrationを要求する。**中央Run Registryや新公開schemaを前提にしない**。
5. **TEIとEKIのFenceは異なる**：TEI`RunKey/attempt/fenceEpoch`は同じ実行計画の進捗資格、EKI`TaskID/Attempt/Lease/fence`は委任先Workerの物理実行権限。数値が同じでも等価でなく、どちらかのFenceの更新は他方のEffect/Grantを自動失効させない。`FlowStep/OperationID ↔ EKI TaskID/AttemptID ↔ Provider ExecutionRef`の相関を必要とするが、EKIにTEIのBusiness Flow意味・Step Ledgerを持たせない。
6. **委任先による内部Retry**：[EKI #115](https://github.com/hat47x/eki-fabric/issues/115)のRQ-054はopen。Cloud Function/QueueやManaged HPCが内部Retryを制御できないとき、副作用あり／再実行根拠不明なStepは配置せず`unsupported/held`。EKI外側の再投入、Provider内部のretry、TEI Plan Resumeを足し合わせて同一Effectを複製しない。
7. **EKIの実Task phase**：[EKI #267](https://github.com/hat47x/eki-fabric/issues/267)、[#381](https://github.com/hat47x/eki-fabric/issues/381)、[PR #695](https://github.com/hat47x/eki-fabric/pull/695)はclosed。現`main`では`admitted→ready→active`を配置側、`active→verifying`をClaim受理側のCASで扱い、拒否後に`active→ready`/Exhaustedへ戻す経路とテストを確認している。#695はWSL clean cloneで`./scripts/verify.sh` PASSと報告。**前稿の「readyへは試験が手動CASしないと移れない」は現在の全構成の未実装判定ではない**。ただしTEI JobのOutput/Provider Business Commit/Group Outcomeには別受入が必要。
8. **EKIのTrust修正**：[EKI #466](https://github.com/hat47x/eki-fabric/issues/466)/[#467](https://github.com/hat47x/eki-fabric/issues/467)はclosed、[PR #678](https://github.com/hat47x/eki-fabric/pull/678)/[#675](https://github.com/hat47x/eki-fabric/pull/675)の修正・個別試験が報告されている。**旧Source-staticの二重fingerprint／読取障害のsilent bypassを現在の未修正不具合として扱わない**。更新・Issuer失効・Agentの真正検証までの実運用は別Gate。
9. **Task GroupとPlanは異なる**：[EKI #126](https://github.com/hat47x/eki-fabric/issues/126)はopenで、予算・期限・取消・部分的な結果をTask群として束ねる契約の検討段階。TEI RunPlanは**順序付き業務Operation**であり、Task Groupは**複数独立Taskの資源・期限・部分結果の集約**。一つのJobが多数の評価Taskを含む場合でも、EKI Group APIが完成したと見なさない。初期はTEI側の非Authority集約表示と既存Task/Result IDの関係だけで比較し、EKI Groupの予算・取消等の強い一体性は#126 Ownerが受理するまで主張しない。

### 13.1 三層の再試行で守る不変条件

| 階層 | 実行/結果の所有者 | Retry/Resumeしてよい条件・影響 |
| --- | --- | --- |
| TEI Plan／Flow | 業務Meaning/Operation、Stepの依存・品質・Commit期限、結果採用とSource Revision | RunPlanの再開は既完了Stepの**記録**を読み飛ばすだけ。未記録Effect／Source改版／Unknown CommitならHost/Provider Lookup＋Owner受理までhold |
| EKI Task／Attempt | タスクの受理・能力/Policy・Lease/Fence・Worker Claim/Result、BUD-1/Recovery Window | Retryable/Effect Class/残予算・最新Admission、旧AttemptのEffect照合と独立に判断。TEIの全Flow StepをEKI内部Schedulerへ転送しない |
| 外部Provider Job／Reservation | Cloud/HPC等が持つ内部ExecutionRef・Queue・preemption・internal retry・課金 | 内部Retryを停止/観測できるかProvider Contractで確認。Provider成功Report≠TEI Business Commit、ACK unknownなら追加Job発行前にLookup |

**初期の安全な接続**は、再実行可能・副作用なしの一つのStepを、1つのEKI Task/Attemptへ委譲し、実Output body/digest/独立Expectedを取得すること。**副作用ありStepのResume/EKI Retry/Provider内部Retryを同時には有効化しない。** 通信不能時の`unknown`を単なる「未実行」に変換しない。

### 13.2 追加の固定反証（VS-23〜32、全件NOT RUN）

| Case | 故障・対照入力 | Independent Expected／保留条件 |
| --- | --- | --- |
| **VS-23 Journal commit gap** | TEI Stepの外部EffectはCommitted、直後Crashで`runlog.Append(resolved)`未達 | 再開しても無条件にStepを再実行しない。Business OperationID/Provider Receiptと照合、未知ならhold |
| **VS-24 Duplicate runners** | 同じRunKeyを別プロセスが同時に`Last→Start`。Fileに両方`attempt=1/fence=1` | File Journalの現在実装だけでexclusiveとは主張しない。実Process/CAS OwnerのGateまで副作用Stepを同時実行不可 |
| **VS-25 Plan edit while paused** | 同じPlanIDでv1はA→B、再開前にv2でA→C→Bへ変更 | 旧Step Aの記録だけで新C/Bを勝手に続行せず、Meaning/Artifact/Step Contract一致を受理するまで停止 |
| **VS-26 Fence mismatch across planes** | TEI Run attempt 2/fence 2、EKI Task Attempt 1/fence 7、Provider JobRef-X | 独立Identity/Currentnessを検査し、整数の一致で横断Grant/Resultを認めない |
| **VS-27 Provider internal retry** | 副作用ありJobをProviderが自動2回再実行可能、TEIもResume、EKIもRetry | EKI #115が求める内部Retry制御がなければ配置しない。最終のBusiness Effectは二重実行しない |
| **VS-28 EKI phase source progress** | 現`main`の`admitted→ready→active→verifying→terminal`、失敗→ready・同時Cancel | #267/#381の報告済みPASSと本稿の再試験を区別。CAS失敗・reconcile後も二重AcceptedResultなし |
| **VS-29 Trust false-positive** | Bundle I/O ErrorまたはDER fingerprint違いで旧イシュー文を参照 | 新Source/修正済PRで拒否・移行を再評価し、過去のIssueだけから未修正と断定しない |
| **VS-30 Task Group partial** | 10 Taskのうち6 verified、2 failed、2 unknown、期限終了 | 成功数6、失敗2、不明2を保持。10/10正常完了やunknown=空結果としない。グループ全体の完了認定はOwner受理範囲に限定 |
| **VS-31 Group budget and duplicate** | 10個のTaskを2系統Providerへ投入。5件は完了後に重複Callback、3件は未実行、合算Budget不足 | 重複Callbackで完了数・消費予算を二重計上しない。BUD-1をグループの擬似台帳で置換せず、不足時は実予算Gateで止める |
| **VS-32 Partial utility/quality** | 100件の候補評価中80件成功。業務品質は「全100件必須」と「20件以上の独立候補で可」が別Job | 前者のBusiness PASS不可、後者は選定された母集団/証拠/Qualityを満たす範囲でだけ部分成果利用可。分母100を無かったことにしない |

**独立Oracle**：VS-23〜32は既存[TEI #2514](https://github.com/hat47x/tei-platform/issues/2514)、[EKI #57](https://github.com/hat47x/eki-fabric/issues/57)/[#126](https://github.com/hat47x/eki-fabric/issues/126)/[#115](https://github.com/hat47x/eki-fabric/issues/115)の受入へ**差分を引き渡す**。この文書に新Runner、新Task Group正式Schema、新Hostや中央Receipt Registryを作らず、未採用の分布推定・Deadline確率・3段再試行自動Policyを既存実装済みとして扱わない。

### 13.3 最小受入順序とGateの更新

**V1’（実Go/RunPlanの限定証拠）**：`runplan.go`の既存Step進行と`runlog.go`のRunKey/Steps/Attempt/Fenceを正本とした固定Direct/Pure fixtureで、記録・再開・改版・2プロセス競合を独立Expectedへ照合。現GoのPR試験PASS（既存報告）と新VSの`NOT RUN`を別資格として保持。業務Providerの効果・許可は未証明。

**V2’（EKI Nativeの現在実装へ合わせる）**：以前の「Taskがreadyに動かない」「Trust旧二件未修正」を停止理由にしない。#267/#381/#695のTask phase実装とAcceptedResult Store/Output bodyを既存Sourceに基づき再資格化し、Input Artifact外部登録・TEI Per-Job Executor binding／Product-only Result E2Eを独立に調べる。

**V3’（Managed Pool/Task GroupはOwner別）**：#115の内部Retry無効化/観測、#126の一群予算/取消/期限・部分結果の契約、#67のCluster Provider、#155のRecovery Fairnessを別Gateにする。現Task APIの複数呼出し結果をTEIで束ねても、EKI-native Group DeadlineやGroup Budgetの実装済み主張はしない。

**Stop**：直接Go/RunPlanのStep resolvedがあるだけ、同名RunKeyにPlan変更がある、同じPlanへ別プロセス同時起動、External Effect/Provider ACK unknown、Task Groupの部分結果のQuality Scope不明、EKI Trust/Policyの現在Revision不明、EKI Result body/TEI Business Receipt未取得のいずれかで、最終的な`finished/assured`表示へ昇格しない。

## 14. 実EKI P0 ABIの非損失ゲートを、実ジョブ試験の前提に置く（2026-11-04）

### 14.1 今回の新しい実証：契約の非同値を19ケースで固定したが、実Jobはまだ走っていない

並走の[TEI Draft #2787](https://github.com/hat47x/tei-platform/pull/2787)は、EKI[実OpenAPI `spec/external/openapi.json`](https://github.com/hat47x/eki-fabric/blob/main/spec/external/openapi.json)の**実Git blob `388f2e3ad32ba663b4777fcfb4304b0cf6f48f3f`**をtest-only snapshotとして固定し、`tools/reference-tei-eki-contract/probe.test.cjs`の**WP-01〜19**でTEI→EKI投影の損失を検出する。PR #2787の報告ではexact-head `4b4c6d3f24e2d0994a4f5a1b2a674af3320ace52`、[Actions run 41641607381](https://github.com/hat47x/tei-platform/actions/runs/41641607381)で**19/19 PASS**。今回、接続GitHubのEKI`main`を再読したOpenAPI blob SHAも同一値だった。これは**ソース構造の負例を検出する静的契約試験**に限る。現EKI Worker/ProviderにTEI生成Jobが実投入できること、費用/期限/回復、OwnerのMeaning Acceptanceは証明しない。

**P0非損失判定表**（TEI側で意味が要求される場合。EKIのSourceについては2026-11-04の参照）：

| 受渡し意味 | 現EKI P0外部契約の観察 | 下位へ無断投影した場合の損失 | 最小の判断 |
| --- | --- | --- | --- |
| 入力パラメータ | `TaskRequest.parameters`は存在するがOpenAPIに「Workerへ渡さない」と明記 | TEIの`quantity`等がClient側にはあるだけでExecutorが受け取らない | **`not-representable-for-execution`**。Input Artifact等で実際に渡せる別経路が証明できるまでJobを拒否 |
| Executor一致 | `ExecutorRequirement.version_constraint`はfingerprintに記録するがAdmissionで評価しない。TaskRequirementに厳密`executor.digest`なし | TEIのArtifact/Entrypointの指定版が実行されると誤認 | **`wire-shape-only`**。EKI Agent WorkOrderの実Approved Executor digestとProvider側Actualを独立照合できる場合だけ適合候補 |
| Effect class | `reliability.effect_mode`は`none`のみ | TEI DB更新・外部支払い等のEffectful Jobを「純粋計算」と虚偽申告 | **`not-representable`**。外部副作用はEKI P0の委任対象外。既存Hostで実行／明示Owner判断 |
| 時間契約 | `TaskRequest.deadline`は単一値。TEIの`not_before/target_complete/latest_useful/recovery`とは異なる | 時刻を勝手に最終時刻へ潰すと、早期開始禁止／優先目標／業務完了・回復の意味が失われる | **`partial wire only`**。TEIの残余意味を外部の実Ownerが保持できる場合に限り予測候補。自動SLA/保証・状態確定は不可 |
| Artifact入出力 | OpenAPIがArtifact GETとInput ingest endpointを宣言する一方、実HandlerではInput ingestを501へ送る。Output GETはOwner/期限/Store条件下で存在 | `output_digest`や契約上のpath存在を、Input実登録・Result body取得に誤変換 | **`input-unavailable`／`output-conditional`**。同一AttemptのInput/Output bytesとdigestを実Processで照合し、未対応経路は停止 |
| 資源提供者/データ領域 | `TaskRequest.resources`は物理要件だが、任意Provider選択やData Residency GrantをCallerの自己申告で認可する契約ではない | 「CPU/GPU要件を書いた」＝希望Clusterが認可・予約されたと誤認 | **`authority-owned-elsewhere`**。EKI DLG-1/BUD-1/CAP-1とProvider Admission/Reservationを別に照合 |

**適合性の処理順**：①TEI Business/Quality/Source Meaningの必須意味を列挙、②EKI current OpenAPIとAdapterで`lossless / representable-but-not-enforced / not-representable / unknown`に分類、③欠けた**必須意味は黙ってdefaultへ変えず**`unsupported`としてDirect/既存Hostへ戻す、④残余が適合可能な純計算Taskのみ実EKI Process受入に進める。上記はAdapterの**受入基準**であり、新`TEI-EKI universal ABI`や第二Schema正本を採用するものではない。`extensions`や`client_request_id`に未保証意味を詰め込んでもEKIが解釈・強制する証拠にはならない。

**エビデンスの混同を禁止**：WP 19/19 PASS（Schemaの静的対照）≠TEI Job Lowering完了≠Product-only Task start/result verified≠Input/Output body≠HPC Provider reservation≠Business Commit。旧EKI #112の422/501と現API Handlerの条件付き経路を区別した§12を優先し、今回の静的差異が実APIの正常性を保証しないことを記録。

### 14.2 AI-first Test Scaffoldへ接続するが、テスト実装を二重化しない

[TEI Draft #2811](https://github.com/hat47x/tei-platform/pull/2811)は生成AIのinspect→Case生成・拡張→独立Expected/Oracleとの比較→縮小→OwnerレビューをP0とする**文書案**（AI-01〜12、実CLI/Web/AI実装NOT RUN）。本レーンの**WP-01〜19（実CIの静的ABI対照）**・**VS-01〜32（本PR未実施の実行・Effect縦断）**・**LB-01〜32（期限/予算/回復）**を、新規Runnerを起こさず既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)のScenario/Target導出、[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Oracle、[#1131](https://github.com/hat47x/tei-platform/issues/1131)のFixtureへOwner分配する。

**生成AIに最初に解放する作業は**、`Required semantic field × current EKI OpenAPI × mapping status × negative counterexample`の**read-only Gap抽出・候補Fixture生成**。実Providerへ自律Submit/Cost-bearing reserve/Grant変更/Effectful Job再試行する権限は与えない。生成CandidateはSource revision/独立Expected/再現コマンド/期待失敗・実Observedを別列にし、あるAIの回答を別AIが追認しただけではOwner受理としない。

### 14.3 外部Source受理とWorker Trustは別のversion/currentness

[TEI Draft #2812](https://github.com/hat47x/tei-platform/pull/2812)は、同一Provider Loadのselected Namespace原文／Manifestを**privateのSource Witness**として保持し、公開のDescribeImportedParsedLoadはDigest等のメタデータに限定する実装へ改訂された。**Full Input Closureでも原文開示許可でもなく、CI成否は対象HEADに限定する**。同じSourceを後から再取得してbyte一致したからといって、最初のLoadの証拠にはできない（[#2801](https://github.com/hat47x/tei-platform/pull/2801)）。TEIのReference/Meaning/Artifact鮮度と、EKI Workerの信頼・Execution Grantは個別判定を継続する。

EKI側の[PR #694](https://github.com/hat47x/eki-fabric/pull/694)はWorker Trustの署名鍵集合をAgentの`anchor.json`という**配備所有者の設定**で管理し、受信した版自体に「次の信頼鍵」を宣言させないと決定。[PR #692](https://github.com/hat47x/eki-fabric/pull/692)には複数鍵のいずれかで検証して受理する局所実装と試験の報告がある。**TEIのSource/Accepted MeaningをEKIのTrust Anchorへ自動昇格させず、EKIの署名済みWorkOrderもTEIのビジネス上の意味受理としない**。複数鍵・期限切れ・失効・I/O ErrorでのFail-closedの運用受入はEKI Ownerの責務。こうしてTEIのPlugin/Reference/Sourceを含め、同じ`version`という名称でも誰が発行し、何を守る版かを明示する。

### 14.4 P0の小さな受入票：3ジョブで「できる」と「できない」を切り分ける

| Test Class | Minimal fixture／Independent Expected | 現在の判定 |
| --- | --- | --- |
| **A. Parameterless pure** | 固定Executor・空Parameters・適合Resource、実入力なし／簡易Output。EKI配置・Lease・Claim・Result/Artifact body/Ownerの因果を観測 | Sourceでは経路が存在、Product-onlyの再試験が必要。**本稿NOT RUN** |
| **B. Typed-input pure** | TEI Go Function `quantity=2,unitPrice=300.00`→`600.00`、Byte/digest一致、NULL/missing負例。入力受渡しのActual WorkOrder/Agent Readingを必須 | EKI P0のParametersはWorker未配送、Input ingestも未対応。**現契約で未解決/実行適合を未証明** |
| **C. Effectful business** | 承認済Operation ID/Host Grant/外部DB Commit/Receiptを要するJob、Result再送とACK lossを対照 | EKI P0 `effect_mode:none`のみ。**EKIを実行先にしない**。TEI/既存HostのEffect Providerを別に使う |

**次の最小実装の方向**：Bを製品機能とする前に、EKIの既存Owner [#112](https://github.com/hat47x/eki-fabric/issues/112)・外部Artifact ingest/WorkOrderという**既存Contractの必要最小の差分**だけを明らかにする。高価なSolver／複数Pool Federation／自前AI Job RuntimeはA・BのE2E成立を前提に後続へ回す。A/B/Cの3種類を同じ「Job成功」カウントで合算しない。新ケース番号を乱立させずWP/VSの既存IDへ参照で引き渡す。

**この章の証拠区分**：EKI mainの実OpenAPI source blob、TEI #2787のexact-head WP 19 PASS（外部PR報告）、#2811/#2812の文書・限定Sourceコード提案、EKI #692/#694の修正・運用決定、既存TEI RunPlan/Provider検証からの**設計提案**。本Draftの新しいGo/Node/実EKI/実AIテストは**NOT RUN**。Owner受理、メインline統合、実Worker I/Oと金銭的最適化は主張しない。

## 15. Build/Toolchain/Data Localityを実行可能性の独立Gateへ（2026-11-04）

[Deadline/Budget/Recovery§12](TEI_EKI_DEADLINE_BUDGET_RECOVERY_COORDINATION_2026-11.md)のCL-01〜08を、T0/T1/T2の**同一Artifact比較に必要な実行前の条件**として追加。TEI [Job/Build設計 #2788](https://github.com/hat47x/tei-platform/pull/2788)の単一Go Binary既定・必要時のみ分割と、EKI [#112](https://github.com/hat47x/eki-fabric/issues/112)のビルドキャッシュ差（過去Spikeで0.8/12.5秒）・Toolchain版差、[EKI #58](https://github.com/hat47x/eki-fabric/issues/58)の転送費用をつなぐ。**通常EKI Workerでユーザー任意URLからGo ToolchainやInputを勝手に取得しない**（[RQ-029](https://github.com/hat47x/eki-fabric/blob/main/requirements/PRODUCT_REQUIREMENTS.md)）。Executor/Runtime Class/Artifact digest/入力のData Grantと、実予約は別検査。

- **T0**：既存Go CLIで同一Source/Meaning/Artifact/Entry・Runtime Class/Toolchain/Input/Output bytesを固定。Build/Execution timeを別計測。
- **T1**：EKI P0 `parameters`はWorkerへ配送されず、Input ingest経路も未実装という§14の非損失Gateを先に通す。Task/WorkOrderに`Go Source Name`を記述しただけでは、EKI Workerが同じGo Binary/Entry/Dependenciesを実行した証拠にならない。
- **T2**：管理Pool Providerの`staged artifact ready`、`current authorized Data access`、`capacity reserved`、`Task/Attempt/Lease`、`Result body`を**別Receipt**として取得。Cache HitやWarm Statusは、現在のTrust/Quota/Leaseの代用品にしない。実Providerを確認する前に`deadline recovery feasible`としない。

**一つの実行単位で観測する最小データ**：`logical_function_id, source/meaning/artifact/toolchain revision, runtime class/OS/arch, executor digest, input digest+size, stage/cache evidence age, data access scope, transfer bytes/time, provider reservation/attempt, output digest+size, verification and deadline, BUD-1/CAP-1 physical consumption`。いずれも**既存正本へのリンクを張るread-only票**とし、別Registryや新しい公開ABIの採用ではない。3経路で`cold / warm / unavailable`を同じ意味で検証し、TEIの選択済み外部Import原文証拠（Draft #2812）をFull Input Closure／ビルドキャッシュ資格へ誤昇格させない。**CL-01〜08はこの追補でNOT RUN**。初回の合格基準はP0のPure Job一件に固定する。

## 16. AI生成テスト・Source開示・実EKI受入のゲート統合（2026-11-04）

他レーン[TEI #2813](https://github.com/hat47x/tei-platform/pull/2813)はtest-onlyのGo Referenceで、JSON Fixture baseline 9件と追加候補2件により、Go側のテストコードを増やさずCase候補を追加する最小の足場を示した。4件のSource/ExpectedはGo側にも固定しており、未知契約・重複Case ID・空Case・不正Stageの拒否を含む。**#2813のPR本文はexact-head CI/実Go PASSの確認前であり、実行済みとは記録しない。** [AI-first計画#2811](https://github.com/hat47x/tei-platform/pull/2811)と[Scenario SSOT #1180](https://github.com/hat47x/tei-platform/issues/1180)の後続Consumerとして用いる。DSLのJSON形式をEKIの正式Test Runner／API Schemaへ流用せず、Caseの追加・独立Expected確認という**実装パターンのみ**を再利用する。

[TEI #2812](https://github.com/hat47x/tei-platform/pull/2812)の最新P0bでは、同一Provider Loadで読んだselected Semantic Source/Manifestの**原文取得経路をprivateに制限**し、公開DescribeImportedParsedLoadをDigest等の**metadataのみ**へ修正した。初期案の「原文証拠があるからAI・Studio・EKIへ内容を渡せる」は誤り。Source WitnessはFull Input Closure、現在のData/Disclosure Grant、Owner Accepted Meaningのどれでもない。[EKI #321](https://github.com/hat47x/eki-fabric/issues/321)/[#433](https://github.com/hat47x/eki-fabric/issues/433)はclosedでWorker署名鍵・Trust Bundleの所有境界が整理されているが、TEI Source DigestをAgent Trust Anchorへ昇格させる理由にはならない。

### 16.1 新規実行基盤を作らない受入判定

| Gate | 既存Owner/証拠 | 合格範囲と停止線 |
| --- | --- | --- |
| **A Source / Meaning** | TEI Source/Reference #2812、Owner受理Meaning、Artifact Build証拠 | 原文開示の権限と同一LoadのFingerprintを別管理。partial / 未受理なら業務意味の確定不可 |
| **B Independent Tests** | TEI #2813 TestData拡張パターン、#1180 Scenario、[#2514](https://github.com/hat47x/tei-platform/issues/2514)独立Oracle | AI生成の負例はCandidate。Expectedが同一Generator/Actual由来なら採点不可。新しいGo/CLI/Web実行は別資格 |
| **C ABI no-loss** | TEI #2787のWP-01〜19、EKI OpenAPIの対象Blob/Rev | Parameter Worker未配送、Effect none限定、Executor版の強制不足、Deadlineの意味縮約、Input ingest未対応を黙って変換しない。不適合時はDirect/既存Host |
| **D Real Execution** | RE-01〜08／VS-01〜32、EKI Task/Attempt/Provider、Output body、TEI Business Receipt | Product-only実Jobで実Input/Artifact/Executor/Result/Current Grantを照合。Go参照RunPlan完了や静的WP PASSを業務Commitへ合算しない |

**最小動線**：公開可能な合成Jobの固定Input/Outputと独立Expected → AIが不足するCaseの候補を追加 → 既存Scenario/FixtureでSchema/ID/Scope/Oracleを検査 → 対象EKI OpenAPI RevでWP静的no-lossを判定 → 不適合ならここで停止 → 適合する純粋JobだけをT0 Direct/ T1 EKI Native（RE）で実検証 → T2 Provider/Deadline/Costは独立Gate後に進む。EKIのAPI/Artifact/Trust改版時は旧SnapshotのPASSを再利用せず、現Revisionで再確認する。

**実装責務**：テスト生成・差分縮小はTEI #1180/#2514/#2811/#2813、ABIの追加はEKI #112、実Result/ProviderはEKI #57/#67、Deadline/Recoveryは既存#58/#115/#126。新中央Evidence Registry、新AI Policy Authority、新CLI/Web Driver、新EKI Job Engineは設けない。AIへ非公開Source本文/下位Worker Topologyや有償Capacity変更権限を渡さない。WP静的CI報告と#2813 test-only P0、#2812 private Source Witnessは別証拠であり、**この追補のTEI→EKI実行・AI生成・Worker/Hostの結合実験はすべてNOT RUN**。
