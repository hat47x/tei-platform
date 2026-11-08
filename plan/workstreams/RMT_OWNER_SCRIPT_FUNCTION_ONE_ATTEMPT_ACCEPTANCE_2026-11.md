# RMT-2 一試行受入：Data Script／標準Function／Owner Effectの忠実移植と実行資格（2026-11）

- **状態**：S3横断の実行計画案・未採用／実TEI一試行未検収。既存意味正本、Owner、Issue、Gate、製品Schemaは変更しない。
- **目的**：旧参照モデルのData/Function/Businessを一つの業務例へ接合し、Source転記や合成Rule試験だけで「移植完了」と判断しない。設計の追加よりも、同じ入力・改訂・試行に対する互換性の判別を優先する。
- **責務の正本**：[RMT Source/CrossView #950](https://github.com/hat47x/tei-platform/issues/950)／[RMT Relationship計画](TEI_RMT_RELATION_CROSSVIEW_ASSURANCE_PLAN_2026-11.md)／[Business DFD #1125](https://github.com/hat47x/tei-platform/issues/1125)／[Data Rule/Owner Write #1129](https://github.com/hat47x/tei-platform/issues/1129)／[Effect/Receipt #1174](https://github.com/hat47x/tei-platform/issues/1174)／[Critical Path #1119](https://github.com/hat47x/tei-platform/issues/1119)／[one-revision Packet #1163](https://github.com/hat47x/tei-platform/issues/1163)。
- **受理権限**：原典のSource事実と移植候補は自動受理しない。旧処理の正解・Business Actionの完了・欠落した実行契約はOwnerと独立Expectedが必要。原典にライセンス不明／非公開内容が含まれる場合は公開Repoに転載しない。

## 1. 移植を三種類の変更に分ける（同一対象・同一Expectedで比較）

| 区分 | 位置付け | 必須の比較と停止 |
| --- | --- | --- |
| **P: 忠実移植（Parity）** | 旧Data Model側Script、Function、業務手順の**観測・意味を保存する**。SourceのScript字面/Locator/宣言Attachment/Trigger、Field/Relation/Key/SortKeyのraw差を失わない | 旧Runtime実測・独立Expectedがなければ「同義確認済」と言わず、対応関係を未確認とする。Scriptが読めても発火時点やTransactionを推測しない |
| **A: 実行契約への適合（Adaptation）** | TEIで必要なContext Mapping、型/NULL/部分集合、現在Grant、Owner Write、Request/Operation ID、Tx・Receipt・Result/Errorを明示して安全に実現する | 旧Runtimeに曖昧な例外や正しい回復契約が無い場合、「移植忠実度P」と「TEI安全契約A」を**別結果**にする。Adaptationを当初から原典の意味だと書き換えない |
| **E: 将来拡張（Extension）** | 複数Target、EKIジョブ委譲、専用状態投影、一般Data Operator等の追加機能 | P/Aを検収する前提にしない。別OwnerのIssueへ渡し、不要な汎用Host・State/Workflow Registry・強制StateMachine定義を作らない |

**構成原則**：X-TEAのTable Scriptによる凝集性を、TEIではData Modelへ添付する**一つのData Script/Rule**と、これを共有する標準Functionへ移植する。各Function／GUI／API／Batchに業務ScriptとSQLを複製しない。Data Scriptは人間のSource、Data Ruleは解釈された型付き意味の候補であり、二重Authoringを設けない。IF主体のScriptが唯一の業務状態更新正本でよい場合、状態図はread-onlyの部分Projectionに留める。明示Transitionを正本に選ぶ場合はIFの業務意味を並行所有させず、OwnerがMode変更を受理する。これらは[#1129 Script-authoritative/Transition-authoritative](https://github.com/hat47x/tei-platform/issues/1129)と[既存#2776 Source計画](https://github.com/hat47x/tei-platform/pull/2776)を利用する提案であって、新しいCore Kindの採用ではない。

**Data Model／SQLの位置**：Table/Field/Key/Relation等は直接宣言し、標準Functionの通常のRelation取得を可能な限り明示Context Bindingから導出する。SQLはLINQ型変換に利用できる**第一級の選択肢**だが、TEI DSLへSQLネイティブ構文を強制しない。StudioからのDBクライアント操作、SQL→型付きData Operator→Target SQLの候補、Native SQL原文保持を区別し、複雑な集計・DB固有処理ではNative SQLを許容。SQLの字面・同じResultからNULL、Bag/順序、元Entity/集計Grain、Schema/Dialect/実Read、移行の同義を自動受理しない（[#2387](https://github.com/hat47x/tei-platform/issues/2387)、[#2622](https://github.com/hat47x/tei-platform/issues/2622)、[Draft #2778](https://github.com/hat47x/tei-platform/pull/2778)）。

## 2. 最小の一試行は「二入口・一Rule・一Owner確定」に限定

**合成用の業務題材**：一つの注文（親）に二つの明細、提出可能性のValidation、状態更新IFと、条件に応じた同じOwner側の書込み。UI風標準保守FunctionとAPI/バッチ風Functionの**二入口**を比較する。具体的な旧モデル固有の識別子・業務式・旧Expectedは固定原典からOwnerが別途照合する。合成正例を旧仕様に読み替えない。

**試行の固定キー**（既存#1163のEvidence欄を使用し、新Schemaを発行しない）：Source file/raw digestとSource location、Application/Domain/typed Subject key（tenant・key順序・型・null扱い）、Source Index/Meaning Revision/Role・Operator・Rule Pin、Function/Context InstanceとRule ID・phase、Read Set/非出力Predicate/Relation Path/必要な全対象集合L/Complete Cohort、実Provider Route/Target/Authority、投入した**不変のRequest byte**・OperationID・ExpectedVersion、Attempt、K/Receipt、別Read Oと時点・実Target、Consumerが読み取ったArtifact bytes、Owner判断と独立Expected。異実Provider・別Attemptのログを「同一試行」と合成しない。

| 手順 | Producer／既存テスト・契約 | 観測して次へ渡すもの | 未到達時の判断 |
| --- | --- | --- | --- |
| **E0 Source** | #950 Reader/Transcription、#1031 Business→Function→Data、#2776 RV-S1〜03 | Script raw/Attachment/phase、複合Key・Relation候補、元Source版とLocator、opaque部分 | unknown、転記成功≠意味受理 |
| **E1 独立期待** | #2514 検証基準／原典旧Driver evidence #940、Owner #267 | 同じ入力の旧実動又は独立Expected。拒否/不明/例外、NULL、集合、時刻/業務対象Lを含めた期待 | 旧実動が無ければ legacy=unobserved。TEI結果でExpectedを作らない |
| **E2 Rule/Context** | #1129 Data Rule Phase、#2506 Function Mapping、#2387/2622 Query、#1126 direct Model | 二入口が**同一Rule ID+版+Owner**を参照し、必要Read/Rule非出力入力を満たすか。pure Derive/Validation/Transition/in-Tx/post-commitの区分 | partial/hidden/NULL/意味未受理ならunresolved。未取得をfalseへ潰さない |
| **E3 Action/Request** | #2485/#2598、[mainに統合された#2664 submit caller](https://github.com/hat47x/tei-platform/pull/2664)、既存Driver Binding | 選択したActionのBusiness Targetと実Provider書込み対象、typed input→Request byte/operation、現在Host GrantのBinding | Capability/型が一致しても対象・Grant・意味対応未確認ならexecution-unqualified |
| **E4 Effect/Recovery** | #1174実PG、#1129 Owner-write、#1163 Attempt | 同じRequestで初回書込み・否認・Conflict・ACK-loss・Receipt K／独立Read O／Lookup。二入口が同じOwner側Invariantを破らず、部分Kを全業務完了Bへ昇格しない | K不明でも「未実行」へ倒さない。安全なRecoveryが未証明なら自動再Submit禁止 |
| **E5 完了Claim** | #1125 Business条件、[Business Evidence §11〜15の#2671](https://github.com/hat47x/tei-platform/pull/2671)、#2582 negative依存 | Owner対象Lの完全性、外部Handoff/手作業、Query時点、K→O因果可視性、0件と不存在・Replicaの遅れ、二Providerの同時整合を別列にする | K/O一致・Client保存表示・空ReadだけでB=passedにしない |
| **E6 受領・納品** | #2604 Artifact集合/読者、SHO Composer、#2516 Disclosure、#1121 Target Lowering | 同じAccepted MeaningからGo単一バイナリ／必要時.NET/Java/WASM LibraryとSHO文書を生成したSource/Artifact版、実Readerが消費したbytes、Audience current Grant | 生成成功≠実Loader/外部Host権限/利用者受理。意味未確定の文書は提案として表示 |

E0/E1/E2の証拠は他のE3〜E6の実行許可ではない。各段の**同一Revision・同一Subject・同一Attemptの接合辺**を確認する。既存partial successの件数を足してG1/G2に昇格しない。実旧Sourceと正式Owner意味が無い場合は、P parity **保留**のまま、Aの合成/実Provider技術適合のみ前進できる。

## 3. 反証に使う差分ケース（独立ExpectedはOwner/検証基準で固定）

| Case | 最小変化 | 検出すべき誤った成功 |
| --- | --- | --- |
| **PA-01 正対照** | 同じRule ID＋版・同じ完全な明細集合・正しい実Request・確定K/O | 二入口の共有Rule/Owner書込みが同一Meaningとして確認されないままBへ昇格 |
| **PA-02 Ruleの分岐** | Script Ruleだけ変更、両Functionの表示/入力は同じ | 入口にコピーした旧IF/Ruleが残って一方だけ古い拒否になる |
| **PA-03 hidden依存** | 表示Amount不変でHold 0→1、Relation retarget、NULL/部分Page | UIの同値ResultやReadCount=0からValidation/完了を認定 |
| **PA-04 Phase相違** | pure virtual value変化、in-Tx履歴、after-commit外部通知 | 算定値変化/参照Command成功を確定Eventや外部Effectの証拠にする |
| **PA-05 旧Script未解釈** | 動的Call・例外・旧Table Hookの実起動条件が不明 | Syntax parse・IF図の部分Projectionを旧Runtimeの動作同義として採用 |
| **PA-06 書込み競合** | 他入口からExpectedVersion変化、再送ACK-loss、既存KのLookup終了障害 | validation false/Host denied/technical error/unknownを混同し、別OperationIDで重複Effect |
| **PA-07 空/Replica** | Primary Commit後Replica旧空集合、RLS不可視、0→1→0 | Read=[]を未保存・履歴無変更・安全な再作成と認定 |
| **PA-08 状態保存の適合** | 本体status列から状態付替え、別Workflow Instance／履歴・複合Key | Buri式中央Storeを常設、又は単一statusへ並列Occurrenceをlossyに縮約 |
| **PA-09 Provider差** | 同じMeaningを異DB/SQL Dialect/JSON Path/ProviderへBinding、同数値position | 同じResult/Positionでatomic Tx・Target/Read Scope・因果可視性を共有と宣言 |
| **PA-10 Consumer** | Code/SHOで別Artifact世代や開示Grant失効、Source locatorだけ移動 | 以前のMeaning fingerprintや内部Receiptだけで今の配布/開示資格を再利用 |

**初回の受入上限**：PA-01〜03、06、07について一Provider・一Rule・二入口を優先。共有Ownerの二書込みが単一Provider Transactionに参加できない場合は atomicity=unsupported / unresolved と明記し、Saga等を先行導入しない。PA-04〜05は旧Driver側の根拠待ち、PA-08〜10は後続圧力で十分。Case番号は**本計画だけ**の仮ラベルで、新Gate/Fixture Frameworkではない。

## 4. 実行手順・現物への接続・やらないこと

1. **P0-a #1174**：既存 `internal/security/effectcommit/postgres/trusted_owner_integration_test.go`、`tools/reference-first-web-target/runner_order_submit_postgres_test.go`、統合済み `internal/security/effectcommit/submit/submit.go` をProducerとして用いる。**初回post-commitの不正/未知Result→K→独立O→Lookup**の実PG負例を、同じRequestで実行できる範囲に限定。旧Receiptの**読取相当Tx終了error**は別[#2680 RR-P5](https://github.com/hat47x/tei-platform/pull/2680)（memory Provider）であり、実PG初回効果の証明へ混入しない。合成Grant/TEST-ONLY DDLは本番Authorization/Owner意味ではない。
2. **P0-b #1129**：同一Data Ruleを呼ぶ標準FunctionとAPI形FunctionをBindingし、両入口の拒否・状態更新・Owner側追加行・VersionとReceiptを**同じ独立Expected**で比較。Owner Resourceが二つならtransaction participant/Writerを明示し、同一Tx不可の場合の部分Effectと回復資格を隠さない。
3. **S0 #950／Work H2/J5**：原典Source provenance/旧実動・正式Target参照をprivateの固定版で検証し、公開Reportへraw内容・個人名・非公開モデルIDを出さない。Source Factの存在/曖昧度とMeaning受理を分離。
4. **E2E #1163/#1125**：一改訂・一AttemptのEvidence表へE0〜E6の出典、Case、検証したTool/Process、exit、読み取ったbyte、役割、現在性を追記。UIやSHOが旧版を読んでも「完了」と表示しない。相違は原Source・Rule/Mapping・Provider Bindingのどこを修正すべきかへ戻す。

**状態**：この計画はSource-staticな実装箇所と各Draft/Issueの報告を整理したもので、**新たなGo/PG/旧Driver/Studio/SHO実行・独立Expected採用・Ownerによる実業務意味受理は NOT RUN**。#1119 G0〜G4、#1174 P0-a、#1129 P0-bの未達を維持する。TEI専用Host、中央Workflow/State Store、二重Rule Source、全域CDC/共有Snapshot Registry、新Script Runnerは追加しない。

## 5. 11/3 他レーンの追加圧力：反証一件・Flow起動・実効果対象・外部Job

前節のPA-01〜10に対し、**従来計画が暗黙の成功と見なしかねない辺**を四つだけ追加する。既存[#2777 BP-05とOnline/Offline検証](https://github.com/hat47x/tei-platform/pull/2777)、[#1129 Data Script/Effect](https://github.com/hat47x/tei-platform/issues/1129)、[#1130 Batch/Job](https://github.com/hat47x/tei-platform/issues/1130)、[#2745 DSL/Studio Source/Effect](https://github.com/hat47x/tei-platform/pull/2745)のOwnerと契約を共有する。ここで新しいValidation/Job/Effect Engineを設計しない。

| 追加圧力 | 正しい限定判断と既存Owner |
| --- | --- |
| **PA-11 存在する反例と全称成立の非対称性** | Owner対象集合Lの一要素が所属・条件・実Readにより確実に同定されていてRuleが不成立なら、他の要素が未取得でも「L全員成立」は反証できる場合がある。一方、観測した全行が成立しても、Lの被覆・非空・全述語・版/時点が未証明なら全称成立はunknown。「不成立の証明」と「成立の証明」を対称にComplete要求するか、部分Readの正例をBへ昇格させるのは誤り。空集合を許すか、数量非零条件を別にするかはOwnerのEmptyPolicy。#2514/#1125/#2777へ |
| **PA-12 SQL Observerの対象漂移** | 同じCount=1のSQL観測でも、過去の別Order/別OperationのEffectであれば今の対象のObservationではない。現mainのSQLEffectObserverのcount>0は**設定Queryの述語内の存在**のみを示し、Operation/Subject/Receiptの自動束縛ではない。Command AdapterがCommittedとReplayedを同じ外向きResultへ写像しても「新規保存」は証明されない。実Request Target・Query/Args・Read時点・Receipt・Owner Lへ接合するまでK/O/Bを分ける。#1129/#1174/#2598へ |
| **PA-13 評価と実発火の相違** | oldValue/valueChanged・virtual derived field・Flow/StateのPreview・GUIでの操作候補は、実DB EventやJob起動の証拠ではない。Source ScriptのIFを起点に、Guard（false/unknown/error）→Owner Action確定→Commit Receipt→Event/Outbox発生→Job Accepted→Job Result→Owner Verifiedを別欄で検証する。再計算やSQL Schema driftによるDerived差分だけでイベント実行しない。XF000後継のUIなし起動はFunction/Action Invocationで表現し、専用Host/Lancherを前提としない。#1129/#1130/#1174/#940へ |
| **PA-14 外部委譲の成果物／時限保証** | EKIへTaskを委譲したこと、Task Accepted/Completed、body出力が保存されたこと、依頼主体が実bytesを取得・検証できたこと、業務全件完了は別。Deadline/Recovery/Costの予算は TEIのJob意味と運用条件から引渡すが、配備/資源選択はEKI、業務意味・最新ReadはTEIと実Providerに残す。body不達、別Attemptのbody混入、期限後結果、部分成功を判別。P0-a/P0-bの検収にEKIを必須化せずP1外部実行に回す。#1130/EKI既存Issueへ |

**受入に追加する列（#1163既存Attempt票の補足であり、新Schemaではない）**：対象Lの量化とEmptyPolicy／反例の実所属証拠／全称成立時のCoverage、旧/新値・Derived Preview・業務Event・Commit Receipt・Outbox/JobのPhaseと版、SQL Observerの実Query/Argsと対象Operation ID・実対象、外部Task/Attempt/Result Bodyのdigest・実Reader・期限・回復余地、判定をしたOwnerと用いた独立Expected。**実行されていないPhaseは未実行として保持**し、違うAttempt/Targetの証拠で埋めない。

この4圧力は将来拡張Eだけではない。PA-11〜13のうち旧X-TEA Data Script/Validation/更新Hookと対応する範囲は忠実移植Pの必要な受入に含め、TEIの新しい安全要求や証拠強化はAへ区別する。PA-14は旧UI起動との同義性Pを要求せず、非画面Jobの追加適合Aと任意EKI実行Eを分離する。

**実測限界**：PA-11〜14はこの段落では設計上の比較条件であり、原X-TEA実Driver、実ProviderのObserver対象束縛、EKI実Runner/Body/Deadline、Owner意味は**NOT RUN / NOT ACCEPTED**。局所SQLite参考試験が成功してもTEI本体/旧Driverの一試行PASSにはならない。

## 6. 局所SQLiteの実行済み反証（PA-11〜13）

[実行用Python/SQLite試験](../fixtures/rmt-one-attempt/test_rmt_validation_effect_scope_sqlite.py)を別ファイルに置いた。**自作の小さなProof境界モデルと使い捨てSQLite**であり、TEIの実Provider・Data Script/Rule・旧Driverを呼ぶものではない。

- **PA-11**：同じ注文の完全2行では全称正例、先頭1行だけではunknown。一件の所属確認済みquantity=0はPartialでも反証。別Tenantで同注文番号の不正行は対象の反証に流用不可。空集合はOwner EmptyPolicyによって結論が異なり、NULLはunknown。
- **PA-12**：無条件のCOUNT>0が、異Operation/異Orderの既存効果だけでtrueになる。OperationIDが同じでも別Tenantなら対象のEffectを認定しない。実対象の行が見つかってもK/因果的O/Owner Bは別。
- **PA-13**：old/new値のPreviewでchange=trueでもOutboxは0。合成TxでReceiptとOutbox行を作成した場合のみSQL行を観測する。ただし外部Job実行は証明しない。ACK=unknownでもReceiptがDBに残る対照は、旧効果の否定や新OperationIDによる再実行を許可する根拠ではない。

実行記録：**Python 3.13.5、sqlite3標準モジュール、py_compile PASS、unittest 8/8 PASS**。実行source 7392 byte、Git blob `6ecb3f988e354860211399066a9a89cf4b089dc5`、SHA-256 `542dca721fc23b86ea532174922df1a355b6cc0dcf46eea5f232fee421c077fe` をローカルで記録し、GitHub再取得blob一致。これは**SQLiteで限定した反証の実測**であって、Business OwnerのEmptyPolicyの採用・TEI SQL実行・原Runtimeの実動・実Host Authority・同一AttemptのEffect証明を示さない。

次のP0-a/P0-bでは、この試験の表現をそのまま製品Interfaceに採用するのではなく、#2514の独立Expectedと実ProviderのRead/Receiptへ置換する。Script/Function側で足りないField・Phase・Identityを先に列挙し、追加Coreを必要とするかは実際の失敗から判断する。

## 7. 「一試行」の正確な範囲：Legacy・TEI・Consumerを同じAttempt IDにしない

ここまでの「同じSource/Subject/Attempt」は、**あるTEI実行の内部**でRequest→Effect→独立Readを因果的に接合する要求である。旧Driver試行、TEI試行、SHO/Consumerの別プロセス読込、任意のEKI Task/Attemptは**実際には別のAttempt**であり、単一の識別子へ潰さない。横断比較に必要なのは「一つの受入Scenarioに対応する各試行の証拠を、明示的な比較辺で結び付けること」である。

| 対象 | 固定するIdentity | 許される比較・許されない飛躍 |
| --- | --- | --- |
| **受入Scenario/Case** | Scenario ID、独立Expectedの採用版、合成/旧実体の区別、入力Fixture・対象集合L・Rule目的、対象となるlegacy/TEIのSource/Meaning対応 | 旧・新で値や文面が一致しても同じ受理意味とは限らない。Accepted Mappingと差分表を別保存 |
| **Legacy実行** | 実Driver/JAR/DB Snapshot、Source bytes、Legacy Attempt/Trigger/実Read・書込み・例外、対象Key | TEI実行と同一Attemptだと宣言しない。旧版未起動なら legacy result=unobserved |
| **TEI Rule評価／実行** | TEI Source/Accepted Meaning revision、Rule/Function/Context/Provider/Host版、TEI Attempt、実Target、OperationID、不変Request bytes | 同じFunction名、OperationID文字列、入力表示値だけで実対象の一致や再送資格を推定しない |
| **Commit・独立Read** | そのTEI Attemptの実Request→K/Receipt/Transaction→当該Provider Read/Scope/位置→Oの因果接合。異ProviderのReadは明示別Route/Cohort | 違うRead/業務条件の「それぞれPASS」をBへ足さない。失敗後のLookupは別Invocationでも同一不変OperationID/Requestに帰属し得る |
| **Target生成・Consumer** | Generator/Target Toolchain/Artifact Set revision、実配布file bytes/manifest digest、独立Consumer invocation/read、現在Audience/Authority | 同じMeaningから生成しても別Targetや後続Buildを同じ実行Attemptと呼ばない。Manifestの宣言≠Readerの実受領 |
| **外部Job（必要時）** | TEI Job Semantic IDと実委譲条件、EKI Task/Attempt/Lease、Result Body digest/受領Attempt/時点/期限 | EKI task/AttemptはTEIのBusiness OperationIDではない。Task Completed≠Body取得≠Owner Business Verified |

**継ぎ目の判定**：各段に「Claim、対象・Scope、比較する版、Producer、実行Context/Attempt、実際に消費したbyte、資格を証明できる辺、missing/partial/unknown」を記録する。横断比較のためのScenario IDは中央Proof Registryではない。**比較の根拠が未取得ならfail-closedの判断を保持**し、似た値・同じ業務Key・同じ時刻から無理に結線しない。公開IssueやSHO出力ではOwner/機密情報の開示を別審査する。

**具体的な負例**：同じSource/Ruleで別AttemptのKを借りる、同じOperationIDを別Tenantで使う、旧Driverの表示成功とTEIの実DB Commitを同一試行と呼ぶ、EKIのResult Bodyが次のAttemptから届く、A→B→AのSource byte一致で観測期間の連続性を宣言する。これらは同じScenarioに属していても、因果接合できない限りProduct PASSにしない。

受入票の物理構造は既存#1163/#2514の成果に委ね、本節で新たな共通Evidence Schema/APIを採用しない。

## 8. 11/5次課題：移植ゲートの現況を実Source／CI／実DB／Windows／Owner別に再資格化

照合基点：2026-11-06 JST取得のmain `8aa18ca150f8689e9875d2e0319fb20e704086f5`。**本節は前節の古い「実PG未実行」の一般文言を、個別試験が存在する現況へ更新するための追補**であり、§1〜7のP/A/E・PA反例や既存#1119 G0〜G4のGateを変更しない。mainの新しい証拠と、本Draftの将来採用を別扱いする。

### 8.1 証拠資格（実装 ≠ PR ≠ CI ≠ 実DB ≠ 旧Windows ≠ 意味受理）

| 該当 Claim | mainの実装または既存成果 | 試験・CIで独立確認できたこと | 未了・昇格禁止 |
| --- | --- | --- | --- |
| **Data／Source→Function／Business** | #950 Source Adapter、#1031 CrossView、#1126意味モデルの候補、`reference/workspace/applications/order-entry/realizations/submit-order.tei`（`realize sales.order.submit with custom go`） | 固定Sourceのread-only構造・公開集計の局所証拠 | 原典Script/Hookの実発火、XEAD↔XEAF Role対応・正式Meaning・Functionへの同一受理帰属は未検収。Sourceのpresence/typed path/Business目的の混同は禁止 |
| **Source declared→actually consumed** | main既存Workspace loader/CLI、#2822「宣言0件でもvalidate ok」・#2823「source一覧≠実読込」・#2824 Discovery契約差の実現状 | 実Source読込のないvalidate結果が成功し得るという再現票 | `validate ok`、std catalog 13件、Workspace-files列挙、WorldFPだけで「業務Ruleが評価入力へ入った」と判定禁止。必須Sourceの実Index/Parse/Rule bind/実Consumptionを同版で確認 |
| **Online保存前・Rule／EmptyPolicy** | #1129/ #2622、[merged #2849](https://github.com/hat47x/tei-platform/pull/2849)でAction宣言`emptyPolicy`をParser/Semanticへ実装 | #2857暫定E2Eのunit_price 0：実Rule拒否・Runner未到達・Effect 0を試験観測。#2849の構文/意味試験 | `emptyPolicy`の宣言が`CompletionClaim(TargetSet L, EmptyPolicy, Predicate P)`へ実評価接続済みとは言えない。未保存Patch、全子集合、非表示Guard、同Txの最終postimage、二Function共通Ruleは別要検証 |
| **Command／実PG／Receipt** | [merged #2853](https://github.com/hat47x/tei-platform/pull/2853)の製品Invoker→`commandadapter`→`effectcommit.Runner`→実PG、[merged #2859](https://github.com/hat47x/tei-platform/pull/2859)のRunner Store startup schema check | [#2853 実PG CI](https://github.com/hat47x/tei-platform/actions/runs/41858621216)と[#2859 Runner実PG CI](https://github.com/hat47x/tei-platform/actions/runs/41929846644)はexact head `completed/success`。CAS conflict、模擬ACK loss→Unknown、同OperationID再照会、S02 startup gate等の限定試験名をログで確認 | 本番Grant/Period/Target・真正業務Rule・実Web起動経路のS02 startup拒否・全Owner効果/補償・ネットワーク／プロセス故障は別。Adapterは`committed/replayed`を空Resultへ写すため、外向きResult成功だけで新規Writeを認定しない |
| **一改訂Web→別読取** | [merged #2857](https://github.com/hat47x/tei-platform/pull/2857)のprovisional D8〜D11・実PG Web E2E | 実Web child・Order1001更新と別OSプロセスread・Receipt 1、S05拒否／N11 commit後Observation障害を試験観測。[Repository Health #41928122125](https://github.com/hat47x/tei-platform/actions/runs/41928122125) PASS | 出力監査`INCOMPLETE / productGate=UNQUALIFIED`。provisional Grant/Acceptanceは正式Owner/Host受理ではない。最新mainの全機構をこの過去の同一E2Eで再実行した証拠ではない |
| **再起動／別Key／ABA** | [merged #2870](https://github.com/hat47x/tei-platform/pull/2870)と[merged #2854](https://github.com/hat47x/tei-platform/pull/2854) | 実PG限定：再起動後のmemory Lifecycle unknownが消える。同keyはProvider Receiptで二重適用を回避。別key＋A→B→Aで再適用・Receipt 2件を再現 | 永続Owner/Fence、同Business Operationの別key重複防止、正式自動retryは未受理・未実装。unknownから任意新keyでre-submit禁止 |
| **旧Windows X-TEA** | #940 XD-01〜04、XF310-D01 H1〜H5a/H5bの固定受入票。JAR/sourceはディスク上の指紋記録あり | Windows Computer Useの2026-11-03再試行では初期化`-32602`でapp list前に停止 | 実ロードJAR／GUI／SQL／Tx／別接続DB／Windows実Driver観測は**NOT RUN**。Linux/Go実PGの合格を旧Driver互換証拠へ転用禁止 |
| **業務完了／公開** | #1125 Business Procedure、#267 Owner Decision、#2514 検証基準、#2516 Consumer、#1163 Packet。#2671や本#2783は**未統合** | 合成Rule／SQLiteの局所対照とPR単独Healthはある | 同じ受入Scenarioで原典実動→TEI実際の書込み→Owner完了→実Reader受領の全辺は未検収。G1/G2をPASSへ変更しない |

### 8.2 既存`order_entry.submit_order`を使う次の一個の実行Slice

**Scenario VS-RMT-01**（ケース番号は本計画の引用名であり、新テスト基盤ではない）。**同じScenarioに属するLegacy Attempt、TEI Attempt、Consumer Readを独立識別**する（§7）。合成業務Fixtureで開始し、原典固有のSourceは公開しない。最初の実TEI結合では**一Provider、一Tx、一つのOrder、明細二行、同一Data Ruleを参照する二入口**を対象とする。入口は標準Functionとheadless API Functionを想定するが、両入口が既存実装で同じRule ID／Revisionを実消費できると確認できない場合はその不足を記録して停める。無理に二重IF実装しない。

1. **Source/Interpretation**：#950/#1031の元Source bytes＋locator（private）→実Workspace manifestの必須Source list→`Parsed/Resolved`→本番候補のFunction Data Context/Ruleが実消費したSource Index/Meaning Revision。実読込の観測が無ければ、合成TEIケースでも`Source binding unqualified`。
2. **Independent Expected／Owner**：固定したScenario／Business Key（Tenantを含む複合Keyなら型と全member）／物理PKとの対応／Action Purpose／明細全集合L、空集合Policy、許容status移行、業務拒否／無作用／最終Owner postimageを、Candidate実装の計算結果と独立に記す。原典旧Driver未実行なら**P=unobserved**、TEIのAだけ検証可能。
3. **Before-write／Context**：保存済みHeader/Linesに未保存Patchを適用した**提出予定後状態**を同じ対象Scopeで評価。数量/単価正値・必須全明細・隠れたHold/Status/Relation Role・欠落/NULL/未開示・部分Pageを識別する。全称成立には完全なL、ただし所属確定済み違反一件による反証は可能。Rule Previewや`validate ok`は実Command前資格ではない。
4. **Owner Action→Request**：2入口のFunctionは同一受理済みRuleを参照する。選択Order1001/Purpose=submit、Headerと全子集合、入力Payload、Grant Resource/Period、Provider Target、ExpectedVersion、OperationID、Byte-identical Requestを束縛。**反例：Validation O1・Payload O2、別Tenant同OrderNo、未保存Line変更後Header CAS同一**。合法な派生書込みはOwnerが宣言したRelation/Roleを使い、単純ID一致規則を強制しない（#2598）。
5. **実Effect/Receipt**：#2853の製品Invoker/Runner + #2859 schema precheckを既存#2857 Web入口と同じ**実バイナリ、Source R1、Provider/DB、Principal、Attempt**で再実行。最初に正常→拒否→Conflict→実行時S05/S02→Commit uncertainの順。Owner側書込みが二つあれば、両方が同じTx内の参加者か、分離Txの明示部分成功かを確認。**同Tx内で後続更新が先のValidation結果を破る場合**の最終Invariantを独立検査し、最終postimageをOwner判断から省かない。
6. **Recovery/独立Read**：Receipt K、status/version、関連Owner行、他Order非更新を**別接続**で実測。N11ではKが確定していても独立Observer障害ならV/Oは未確定であり、業務完了Bに昇格させない。模擬ACK-loss→`unknown`は同じOperationID・同じ不変RequestでLookupを試し、Read/Lookup障害なら`not-recorded`に倒さない。実Network cut／Process kill／ABA別keyは独立負例として未実証を残す。
7. **Business Procedure／Consumer**：#1125のManual/External Step、Role≠Authority Role、Completion`(L, E, P)`を確認。Owner受理、観測対象Lのcomplete read／negative dependency、Current Grant、実Consumerが受領したbytesとManifest/Artifact Revisionを別判断。Client/SHOに`completed`と表示してもK/O/Bの証明ではない。

### 8.3 失敗時の最小独立Expectedと停止・回復

| Case（既存E0〜E6欄に記録） | 期待する技術結果／独立観測 | 停止／自動回復を許さない条件 |
| --- | --- | --- |
| **R0 明示Source欠落／実読込なし** | `validate ok`でも必要業務Rule不在を発見。実Consumed Inputs=不足、E0はunknown | Sourceを補う前にProduct Qualification/Effectへ進まない。#2822/23にLoader診断を委譲 |
| **R1 preflight拒否** | 未保存Patch反映後の量化Rule違反／Current Grant/Target不一致ならProvider Write=0、Receipt新規0、別Read不変 | Validation false、missing/unknown、denied、technical failureを単一Resultにしない |
| **R2 in-Tx後続変化／rollback** | Validation PASS後のconsequenceで最終Invariant破れ。必要なら同Tx rollback→別接続で双方Effect 0・Receipt 0 | rollback API呼出し・errorだけでDB未作用を断定不可。rollback結果不明は観測資格保留 |
| **R3 Commit ACK消失** | Providerで実CommitしてもCommandはunknown。実K/別Connection Readで実効果と一Receiptを確認、同OperationID Lookupでreplayed | 自動新OperationIDのSubmit・別Key retry禁止。Receipt/Read unavailable時はunknown保持 |
| **R4 read-only replay/Lookup終了失敗** | 旧Receiptを読んだ後のTx終了障害ではResult/Lookup失敗と効果再適用の有無を別記録（merged #2680 memory reference） | Lookup errorを`not-recorded`やEffect=0へ変換しない |
| **R5 ABA／Process restart** | 同key replayはProvider Receipt 1、別key ABAは再作用し得ることを対照。memory Stageの消失を明示 | 永続Owner/Grantと一度限りの業務同一性の正式方針が未決なら自動resume禁止 |
| **R6 Observer/業務Step欠落** | SQL count>0の対象不一致、RLS/Replica旧Read、Manual Step未完了、Consumer旧ArtifactのいずれかでB/D=unqualified | Kの存在だけで完了済み・配送済みを宣言しない |

**終了分類**：安全な拒否（Effect=0を独立確認）、技術的未作用（Txが確実にrollbackされ独立Readで確認）、Commit済み観測、Commit結果不明（再照会可能／照会不能）、部分Effect、旧Runtime未観測、Owner意味未受理を分ける。これは既存#1119/#1163の証拠欄に記す**判断語**であり、製品に新enum/第二Gateは追加しない。

### 8.4 横断受渡し／Workが必要なWindows実作業

- **#950/#1031/#1126/#2822/#2823**：宣言／実読込／Source→Data/Function/Business CrossView、Business Keyと物理PK・Role粒度。S0は公開Source Factとprivate provenanceだけ。
- **#1129/#2506/#2586/#2622**：共通Data Rule、二つのFunction Context、未保存Patch/全明細L/非出力依存、Online必須とDeferred Qualityの分離、in-Tx final postimage。DB別SQL選択は別OwnerのDraft #2862で、初回一Providerをブロックしない。
- **#2598/#1174/#2564/#2146**：Purpose/対象と実Payload・Grant・CAS/Tx/Receipt・unknown/replay/ABA・永続Ownerを既存Runnerへ接合。新Host/Proof Store/Transaction Coordinatorは追加しない。
- **#1125/#267/#2514/#2516/#1163**：Business Procedure/Manual/Handoff、Owner意味とEmptyPolicy、Expected独立性、実Reader、同一改訂Packet。初回のP0でSHO/EKI、多DB、全Function Pattern、G4 Selfbuildを先行Hard dependencyにしない。
- **#940 + Windows Work担当**：Computer Use正常初期化→app list／画面読取→実ロードCodeSource・JAR/Java/JDBC/DB版の固定→read-onlyの旧Data/Function/Script/Hook段階観測→隔離された使い捨てDBで正常/拒否/例外/rollback→別ConnectionのH5a（外側Commit後、Session終了前）とH5b（別試行Session終了後）の読取。既存XF310-D01の#940五段階受入票を再利用し、成功ログ・画面表示・正常Session終了を未作用/Commitの代用にしない。2026-11-03までComputer UseがMCP `-32602`でapp list到達前停止、**Windows旧Driver実動はNOT RUN**。GUI制御障害時に破壊的コマンドで迂回せず停止する。

**実装と判断の現在地**：#2853/#2859/#2857の実PG/CIは局所PASSED、#2849 EmptyPolicyは構文・意味に導入済み、#2870/#2854は再試行危険の**実際の反例**。#2783/#2671/#2776などは別HEADのDraft/Openであり、この§8追加をmain統合済みと呼ばない。VS-RMT-01の実同一業務・正式Owner意味・旧Driver比較・Web/Provider/Consumer全接合は**NOT RUN／UNQUALIFIED**。今回は計画改訂のみでGo/PG/Windows/Owner検証を新たに実施していない。

## 9. 11/6横断深化：Requiredの独立性／Guard被覆／Source実消費の「証明すべき辺」を固定する

本章は§8のVS-RMT-01を**合格票として閉じるための必要条件**を定める。新たな受理Schema、Proof Engine、Test Runnerや共通Hostは作らない。近接する[Script起動設計Draft #2884](https://github.com/hat47x/tei-platform/pull/2884)（Required/Supplied/Evaluated/Enforced-at-Effect）、[自己記述Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)（候補自身のRequired自己申告は不可）、[テスト設計Draft #2811](https://github.com/hat47x/tei-platform/pull/2811)（Required Case／SKIP／検証基準独立性）、[DB別Query Draft #2862](https://github.com/hat47x/tei-platform/pull/2862)（DB実対象とSQL Realization選択）のうち、**参照モデル移植が判定する部分だけ**を受け取る。

### 9.1 Successを計算する前にRequiredを確定する

**必要検証項目（Required）を、検証される候補の宣言からだけ作ってはならない。** 例えばScript/AI生成Candidateが`required=[]`、RunPlanの`CombinePreconditions`にPeriod checkを渡さない、Source manifestにRule fileを登録しない、Operator自身が`pure=true`と宣言する、Query候補が自分の依存Predicateを報告しない、といった場合にも、`provided checks all PASS`が業務検証PASSを意味しない。

必要項目を確定する出典は、Scenarioの独立Expected、Owner受理済みAction/Rule/Phase、Hostの認可条件、実Workspace/Providerの制約、既存のConsumer用途契約とする。**Owner未受理の項目は「未確定のRequired候補」であり、空集合へ勝手に落とさない**。各Caseについて既存#1163の票に次の列を付ける（新Schemaでなく比較票の欄）。

| 必須Claim／必要項目の独立出典 | Supplied / Evaluatedとして観測する実物 | Effect時にEnforcedであるための追加証拠 | Missing時のClaim ceiling |
| --- | --- | --- | --- |
| **S: 業務Source閉包**：#950/#1031の必須Data/Function/Action/Hook、Accepted Meaning/Source revision、必要Query/Ruleの外部Ref | Manifest, actual loaded parsed bytes/Source Index, Rule/Function Context実消費read。#2822/23の未読込対照 | 実Web/CLI/Generatorの**実入力とReadに使ったbyte**・対応Source/Meaning revision（#2595）。単純`validate ok`や`manifest.Source` digestだけでは不可 | S unqualified → RMT ParityもA全体も未資格 |
| **G: 保存前・効果前の必須Guard集合**：Owner ActionのRequired predicates、全TargetSet L/EmptyPolicy/Phase、Grant/Period/Schema/ExpectedVersion/Observer要求 | 実Preconditionの列挙、Supplied/Evaluated status、使用したbefore/after row、子集合scope/epoch | Provider effectを出す直前に必要なGuardが**現在値で強制**されていること。UI Preview/過去Preflightに合格しても同一Tx時点の強制ではない | G未被覆 → safety/Owner Write資格なし。前Step Effectがあれば`no-effect`とも言えない |
| **T: Business Subject→実Target**：受理済みAction Purpose/Role/Relationと、Business Keyと物理PKの対応 | Function selection、未保存Patch、Request bytes/hash、Payload、OperationID、Providerの実Scope/SQL args | 実書込Target/CAS条件がValidationで検査した対象とOwner許容の派生対象に対応。O1選択→O2 Payload、同名別Tenant、Line membership変化を負例 | T不足 → Receiptがあっても「正しい対象のEffect」未確定 |
| **K/O: 実Commitと観測**：#1174 Effectの原子性/結果区分、#2671のU/K/V/O/B/D区別 | Commit・Rollback・ACK-loss各Invocation、実Receipt、独立Connection readと時点/Scope/Provider position | Receiptの同一OperationIDと実postimage、必要Owner行・子集合全件、Readerの因果資格。SQL Observerの非対象COUNTやReplica旧Readは不可 | Kのみ成立してもOwner Bはundischarged、O障害でEffect=0としない |
| **B/D: 業務完了・開示**：#1125/#267のmanual/handoff、Owner `(L,E,P)`、#2516実Consumer | 実Step/Manual決定、独立Expected、Consumerが受け取った実bytes/digest/開示Grant | 必須Step・対象集合/空集合証拠・実Effect・結果受領を時点とAudienceに対して判定。旧Artifactと現在Grantを別 | 技術Commit PASSでもBusiness Completion/DisclosureはUNQUALIFIED |

判定は各Claimごとに**Required ／ Supplied ／ Evaluated ／ Enforced ／ Observed ／ Undischarged ／ allowed conclusion**を出し、欠けた項目を隠す単一`success`を禁止する。異なるPR／HEAD／ProviderのPASSを一つの実Attemptへ足し合わせない。必要最小のClaimだけ閉じ、他領域を不必要なHard dependencyへ加えない。

### 9.2 現行mainの新しい実装と、未検証の「Required全件」問題

- [merged #2882](https://github.com/hat47x/tei-platform/pull/2882)の`internal/execution/runplan_workspace_test.go`には、WorkspaceからPeriod/Effectを取得して**明示的に渡した**`PeriodPreconditions`＋`EffectPreconditions`で`RunPlan`が実Step 0のBlock／正常Completeへ分かれる**合成Go試験**がある。これは**渡されたGuardの効果**を確認したもので、業務上必要なGuardをすべて列挙した証拠、Provider Commit直前の認可・対象強制、全RunPlanの無副作用保証ではない。
- [merged #2819](https://github.com/hat47x/tei-platform/pull/2819)は`manifest.Source=D1`と実`go build input=D2`の不一致が現行Generator Selectで拒否されない反例。Business Sourceが実Parserへ渡ったとしても**実Binaryへ取り込まれたSource閉包を自動保証しない**。初回は実Product Web/Goバイナリの固定HashとSource Input Closureの照合を優先し、Selector修正は#2595へ委譲。
- [merged #2835](https://github.com/hat47x/tei-platform/pull/2835)は実PG試験の`go test exit=0`で必須Caseが`SKIP`され得る反例と、それを拒否するCIの限定対処。VS-RMT-01の実PGスライスは、**実際に走った必須Test ID／subtest・実DSN Provider・PASS/SKIP/FAIL・負例の判別力**を記録し、単なる`go test green`へ帰属させない。#2514独立Expectedと同由来の自動検証基準は別。

### 9.3 最小4段実証を既存Ownerへ引き渡す（新Frameworkなし）

| 順番／既存Owner | 一つの固定Scenarioで行う実対照（すべて本章では**NOT RUN**） | 次段へ渡せる限定資格 |
| --- | --- | --- |
| **U0：Source/Rule Binding** #950/#2822/#2823/#2595/#2586 | Sourceを必要集合から1件取り除いても`validate ok`が出る負例。次に宣言・実Loaded byte・Index・Rule Context consumed・実Build binary Inputの同一Snapshotまたは明示版Lockを対照。Source移動・内容同一・A→B→Aは別に記録 | 正しいSource/Ruleが実入力に含まれる証拠。**旧Runtime Parityと業務意味受理は別** |
| **U1：Required Guard Coverage** #1129/#2622/#1130/#1180 | 同一Rule・同一L/未保存Patchで、一つの必須Period/Grant/Target/Effect Observer checkを**意図的にPreconditionsへ渡さない**。Supplierが返したPASSだけでは検収不可。全Guardを渡した拒否ケースで実Step=0/Provider未到達を確認し、途中Blockでは先行Step Effectを保持 | 実Preflightで必要Guardの被覆を説明できること。Commit前現在資格はU2で別確認 |
| **U2：同一Web/PG Effect+Recovery** #1174/#2564/#1163 | main統合済み#2853/#2857/#2859の実部品を**同じSource R1・実Binary・PG Connection/Grant/Attempt**で走らせる。Validation O1→Payload O2拒否、Rule成立→後続Owner書込み最終不変破れ→rollback、ACK-loss→同じOperationID Lookup、Observer失敗でもReceipt保持、同Key replayと別Key ABAを独立実行 | Providerの限定K/別Read O/unknown区別。正式Owner B/旧Driver Parityは未合格 |
| **U3：原典とOwner/Consumer** #940/#950/#1125/#267/#2514/#2516 | Windows Workでは旧Driverの実Load/JDBC・Source→Hook→SQL/Commit/独立Readを取得、同一ScenarioのTEIとは**別Attempt**で独立Expected比較。OwnerがBusiness Procedure／EmptyPolicy／許容差分と失効時再受理を判断し、実Consumer bytesを確認 | Pの旧互換、AのTEI適合、Bの業務完了、Dの開示を**別資格**で判定 |

**停止条件**：U0 Source closureなし、U1必要Guard未列挙／未強制、U2不変Request/Target/Grant未固定または実PG required Case SKIP、U3旧Driver未観測／Owner受理なしのいずれでも、そのClaimを`unqualified`として保持する。別レーン（SHO、EKI、二DB、G4 self-build）の全機能完成はU0〜U2の先行条件にしない。

**この追補の状態**：上記は他Issue/PRの実装・CI・設計を照合した検収計画であり、**本PRで新しいGo/PG/Windows/AI Test実行は行っていない**。#2882の合成実Go経路、#2819の不一致反例、#2835の実PG skip検査はそれぞれ局所的証拠。Ownerによる独立Requiredの正式受理、VS-RMT-01 U0〜U3の同一Scenario連結、旧X-TEA移植完了、G1/G2合格は**NOT RUN／NOT ACCEPTED**。

## 10. 11/6次段：最小実Read→Owner書込みの検収台本とEvidence鮮度

§9のU0〜U3を**最初の実検収に使える着手手順へ縮める**。既存`VS-RMT-01`以外の業務Scenario正本は増やさず、原典の個別業務ID・Script本文は#941の非公開境界に置く。採用するのは**二段階**：まず一Functionの実Provider ReadでSource→Query→Contextを確定し、次に同じSource/Subject/RuleをOwner writeと既存実PG EffectCommitへ延ばす。これは実装に着手する順序であり、全体Gateを分離・新設する意味ではない。

### 10.1 VS-RMT-01 の初回実行対象と独立Expected

初回は公開合成データの受注Header一行、明細二行、別Tenantに同じ業務番号の非対象一行、他注文一行を用意する。**Business KeyとPhysical PK、Tenant/Period/Relation Role**は独立Expectedの識別子に置き、Queryの実返却Row IDから逆算して「正しい対象集合」を構築しない。モデル由来Function/Ruleは原典候補であり、旧Driver実動が無ければParity（原典互換）は`unobserved`。次の表の値は設計用の合成例であり、実業務の採用済み金額・税・状態規則としない。

| Case / 段階 | 独立Expected（先に固定する） | 必須の実Witness／否定対照 | この段で合格としないこと |
| --- | --- | --- | --- |
| **RMT-RD1 正常Read** | 固定Tenant/Orderの明細2件のみ、order1002と別Tenant同番号を含まない。NULL/Quantity/Line IDと元Revisionを保持 | 実Source bytes→Loader→Rule/Query/Function Context consumed、選択したProvider/DB AliasとRealization identity＋SQL/Args、別接続のRow identity、Read scope/完全性、実Reader version | Source `validate` 成功だけ、画面2行表示だけでRead全体を保証しない |
| **RMT-RD2 不完全Read** | 一頁1件、NULL、非開示、Query中の隠れたWHERE/ORDER依存をそれぞれ独立に識別。全称成立は`unknown`、所属確定の反例1件は不成立を示し得る | Query/Providerのページ・snapshot/cohort/継続token、選択SQL改版、Scope/field availability/negative dependenciesの実Read | 0件から不存在、既知部分の合計から総額、旧SQL選択から現在の許可を導かない |
| **RMT-WR1 prewrite拒否** | 未保存Patch後の1明細が数量0ならCommandとProvider書込み0。標準FunctionとAPI Functionが同じRule ID/版を使用 | 保存前GuardのRequired/Supplied/Evaluated/実Step到達数、実Request/Payload/Target、Receipt件数の事前・事後、別接続の非対象対照 | 一方の入口が拒否できても二入口同等や旧Driver互換を証明しない |
| **RMT-WR2 normal Owner write** | Header/statusの許容遷移、必要な子行/後続Owner効果、final postimageの不変条件、Receiptちょうど1 | 受理Business Purpose→Grant/Period/Target/ExpectedVersion/不変Request→同一Provider Tx・CAS→K→別Conn O。書込み先とReaderのSQL/Args・Tenant/PKを相互照合 | `Command completed`／`COUNT>0`／`RunPlan completed`だけでBまたはD合格なし |
| **RMT-WR3 failure／recovery** | precommit拒否は確定未作用、ACK失落は`unknown`、同ID再照会は二重作用なし。別Key ABA・途中Step既存Effectは未作用と呼ばない | rollback要求+結果／Commit不明→同OperationID Lookup→Receipt状態/版→別Conn postimage。Process restartと実Network cutは未試験なら明示 | `Go test exit 0`や個別CI結果の合算で、同一実Attemptの回復成立を認定しない |

**初回に実装するテスト単位はRD1とWR1**を優先する。RD1の具体的な実Read経路が成立していない場合、WR2のWeb/PG組合せへ無理に飛ばず「必要Input未取得」「実Contextで消費したRuleが不明」というGapを#2586/#2622へ返す。RD2は同じReaderの否定対照として並行、WR3は#2853/#2854/#2870の既存故障試験を先に引用し、当該同一Web実行では別途検証する。ユーザー操作・画面を必須にしない初回Headless検収を採用し、旧Windows/SHO/EKI/二DBをHard dependencyにしない。

### 10.2 多入口・RunPlanと「途中で止めたから無作用」の誤判定

[Draft #2889](https://github.com/hat47x/tei-platform/pull/2889)の`internal/execution/org_case_runplan_boundary_test.go`は、**実Go RunPlan＋合成toy Step/Journal**で以下の3つを確認した（[専用CI #42100472921](https://github.com/hat47x/tei-platform/actions/runs/42100472921) 3/3 PASS）。(1)必須Host Grant checkを渡さなければ`RunPlan`はそれを自動発見できずtoy Stepが動く、(2)Effect Observerを渡しただけでは`ObserveEffect`呼出し/Commit後Readを証明しない、(3)後続Step Blockでも先行Stepのtoy side-effectが残る。**同PRの全体Repository Health [#42100472849](https://github.com/hat47x/tei-platform/actions/runs/42100472849) はgofmt driftでFAIL**（vet/Go test/workspace validateはPASS）。したがってこの3件は限定技術特性の再現であって、正式統合・実組織DB・TEI受注Owner書込み・全体Health資格を示さない。

RMTで再利用するのは**判別条件**だけ。初回VS-RMT-01はOwner writeを単一Txに収める。将来の二Step/二Txでは`blocked`の前に確定したEffect/Receiptと、後段の未実行をそれぞれ別の観測として報告し、Run全体失敗を補償成功や無作用へ書き換えない。必要なら#1130/#1174の既存部分効果モデルへ渡す。旧DriverのSession正常終了はPending Commitし得るため、同様に失敗時の正常終了を自動的なrollbackとして扱わない（#940）。

### 10.3 AIによるCase追加と「Expectedが正しい」の独立性

[Draft #2890](https://github.com/hat47x/tei-platform/pull/2890)では、既存[#2828](https://github.com/hat47x/tei-platform/pull/2828)のJSON候補に対して**JSONのみ**の明細行番号重複負例を提案（新規テストGo本文は変更しない）。これは`Quantity>0`と`LineNo duplicate`を分離する良いMutation候補だが、Agentが書いたExpected自体は独立検証基準／Owner受理ではない。本日時点で#2890のexact-head CIは確認できない。

VS-RMT-01へ追加する場合は#1180既存Scenario runnerと#2514の独立Expectedを使用し、候補`line_number_unique`を固定Owner Rule・Sourceと結び、実discovered→attempted→PASS/FAIL/SKIPを各Stepで確認する。候補がActualと不一致ならExpectedをActualに合わせて自動修正しない。対象Ruleが存在しないモデルにAI Candidateだけを追加して業務Coverageを水増ししない。

### 10.4 再利用できる証拠の最小キーと失効条件

**同じID／値／hashでも別の主張は資格を引き継がない**。既存#1163の一改訂Evidence Packet内で、次の5つの一致・差分を常に併記する（新共通Schema/Registryを追加しない）。

| 失効を判定する対象 | 保存する実Witness | 再資格化・停止の条件 |
| --- | --- | --- |
| **原典Source／Mapping（P）** | 原byte指紋＋実Locator/Source occurrence、Legacy Runtime版、Data/Function/Business候補Relation、Owner決定 | 元Source移動（byte同一もlocator再解決）、Script/Hook変更、原典Source↔TEI対応変更、旧Driver実観測差。旧SourceのA→B→Aで旧意味受理を自動復活しない |
| **実TEI Input／Build（U0）** | Workspace manifestと実Loaded bytes/Source Index、必要Rule/Function Context consumed、actual Build input/Bundle/Toolchain、Binary hash | #2819が示す宣言`manifest.Source`と実ビルド入力の不一致、暗黙Source脱落、Plugin/Operator版変更、未選択SQLの選択変更。データ読取だけで実Binary closureを保証しない |
| **Runtime Guard／Provider（U1/U2）** | Accepted Meaning/Rule版、必要Guard集合、Period/Grant epoch、DB Alias・選択SQL、Provider/Schema/Target/Transaction、OperationID・Request bytes | Authority/Period/Grant/Schema/Realization変更、Predicate/Scope/SQL変化、Context epoch/対象集合変更。古いPreview/Validation PASSをCommit直前の実行権限へ再利用しない |
| **Effect／Observer（K/V/O）** | 同じOperationIDのReceipt/Commit status、DB postimage row identity+version、別接続Reader Snapshot/Scope/Position/完全性 | Provider observation障害、Replica遅延/部分Page、対象外COUNT、異Tenant、別OperationまたはABA新Key、後続修正。過去Kは歴史として残し、現在O/Bだけを再資格化 |
| **Owner B／Consumer D** | `(TargetSet L, EmptyPolicy E, Predicate P)`、Business Procedure/Manual Step、Owner受理版、Audience/Disclosure Grant、実配布/実受領Artifact bytes | Owner受理撤回、Action/Rule/EmptyPolicy/必要L変更、Manual Step未了、Consumer対象変更、旧Artifact/Reader受領不明。K/O成立でもB/D自動合格なし |

**更新方式**：一部の証拠が変化したら、影響Claimのみを再検収し、履歴を改竄しない。World Fingerprint、Model Revision、Artifact Revision、実Observation・Acceptanceは異なる。同byteの戻り（A→B→A）を「変更なし」として経過中の権限失効を見逃さない。証拠不足は`unqualified`を保持し、未実行・失敗・拒否・非開示を混同しない。

### 10.5 担当への具体的な受渡し（#1119の先行順は変えない）

- **P0-r 実Read**：#950/#1031 Source対応、#2586/#2506 Context→Function、#2622/#2611 DB Binding／Query Realization（初回は一Provider）にRD1/2のExpectedと実Reader scopeを渡す。SQLファイル名からProviderを暗黙推定せず、異DB/異SQLへ無音fallbackさせない（Draft #2862のOwner領域）。
- **P0-w 実書込み**：#1129 Data Rule/Owner final postimage、#2598 Request Target、#1174/#2564/実PG既存RunnerへWR1〜3を渡す。S02 startup check、prewrite S05/Grant拒否、同OperationID Lookupを現在Source/Targetに接続。
- **P1 旧Runtimeと業務Owner**：#940 Windows Workで旧Driver/JDBC/Session/別接続readを採集。#941公開条件、#2514独立検証基準、#1125/#267業務完了受理、#2516実Consumerを別Attempt・別Authorityで照合。
- **横断のEvidenceとCI**：#1163 Packetを一つのScenarioにまとめ、#1119でG1/G2を判定。#2835の必要PGテストSKIP検出を再利用し、CI実行HEAD、Test IDs、実Provider/DB、Scope、Environment、Required/Observed、Negative Control、期限を記録。#2889はfocused CIのみPASSで全Health FAIL、#2890はGo/CI未確認、#2783直前HEAD`b9d6306b7feb3aed43a9d0df9205df272017b15e`の[Health #42101164557](https://github.com/hat47x/tei-platform/actions/runs/42101164557)は**completed/success**（ただし文書とPython fixtureだけ）。**これらのCIの合算で新しいVS-RMT-01を実行したとは主張しない**。

**本章の資格と差分**：文書のみの追加。RD1〜WR3の実TEI Data Context/SQL/Provider統合、同一Business Write、旧Windows Driver実行、Owner受理、現在Host正式Grant、実Consumer受領は**NOT RUN／UNQUALIFIED**。既存IssueのOwnerとPRの判断を変更せず、#2783はDraft維持、mainには取り込まない。

## 11. 11/6検収実装引渡し：現在のProduct APIから何が実行可能かを先に判別する

§10のRD1〜WR3は今後のAcceptance Caseであって、**現Productにそのまま動く一つのデータ取得APIがあるという宣言ではない**。GitHub main `8aa18ca150f8689e9875d2e0319fb20e704086f5`で既存Go Sourceを再照合し、実装のある入口／契約の検証までの入口／未接続を以下に固定した。実装担当は先にこのFit/Gapを実際のGo試験で確認し、存在しないインターフェイスを計画から勝手に生成しない。

### 11.1 実装上の証拠境界（static inspection）

| 現在の確認済み入口 | 実際に確認できること | RD1/WR1へ移る前の不足 |
| --- | --- | --- |
| `internal/workspace/loader.go`の`Load`、`SourceContributions`、`sourceSnapshot`、`ManifestIdentity()` | Workspace管理SourceとLoad時Snapshot・manifest identityを分けて保持する機構がある | Declared/Discovered/Loadedと**選択Function/Ruleが実消費したSource**、Generatorへ渡ったBuild inputsは別のwitness。#2822/#2823/#2819の欠落負例を解消したとは言えない |
| `reference/workspace/applications/order-entry/realizations/submit-order.tei` | ファイル内容は`realize sales.order.submit with custom go` **一行** | これはRealization選択宣言で、実Query/実SQL読取・保存前Data Rule実行・Owner Effectの証明ではない。利用可能な実Function Read入口を#2586/#2506で確定する |
| `internal/realization/contract_query_sql.go` `ContractQuerySQLRealization.ValidateContract` | Target+ProviderとQuery Signature、input parameter、result columnsの**構造的整合**を検査する | Query選択、実DB alias、実行SQL bytes/args、結果行・粒度/完全性、現在Scope/Grantは別。Source名・Query IDだけでDB/SQLを決めない（#2622/#2611） |
| `internal/security/effectcommit/commandadapter/executor.go` | `Executor.Payload`を明示した場合は`request.Inputs`からの組立てより優先し、GrantのResourceを実Runner Scopeへ渡す | `Function selection O1`対`Payload O2`をAdapterが自動的に拒否するとは言えない。Owner Actionが認めるTarget対応の証拠・負例を#2598/#1129で要求 |
| `internal/execution/observe.go` `SQLEffectObserver` | SQLが返すcountが正ならObservedをtrueにし得る | countが**このOperation/Tenant/Target**に帰属するかは実Query/Argsと独立Expectedが必要。単純`COUNT>0`でRecipient/Business Completion合格禁止 |
| `internal/security/effectcommit/postgres/command_lifecycle_integration_test.go` | 実PG Providerを伴う製品Command CAS／Conflict／simulated ACK-loss／同OperationID Takeoverの既存限定結合Fixture | 生のResource文字列のCAS例であり、二Functionの実Data Context・完全な子集合/Rule・Owner意味・正式Current GrantとWebの同一改訂を示さない |

**結論**：RMT-RD1の最小作業は「新SQL/Context DSLを設計する」ではなく、**すでに存在するFunction→Query→Context→実Provider Readの実入口を特定し、同じTenant/対象/Ruleを実Consumed Source/SQL/Rowsで実測する**こと。実Productにその接合APIが無ければ、`Function Context→Query Provider execution not connected`と**明示的なGap**を記録し、#2586/#2622のOwnerへ限定差分を渡す。Fake/Go直接SQL Readerの局所成功を製品RD1合格にしない。

### 11.2 最初に実装担当へ渡す「試験二件」と否定対照

以下は*新規採用ではない*既存#1180 Scenario／#2514 independent Expected／#1163 Packetへの引渡し票。試験名は**提案名**であり、Repoに実装済みの`Test...`と誤認しない。

| 順序／試験候補 | 独立Expectedと成功の上限 | 必須negative control / 判別できなければ未合格 |
| --- | --- | --- |
| **P0-R1 `TestRMTFunctionReadUsesBoundDataContext`（提案）** | 合成Tenant A/Order1001、Header1＋子2、Tenant Bの同Order番号1と別Order1002を除外。Source byte／Semantic ID／Rule・Function Context／DB Alias／選択Realization／実SQL+bind args／実Read Row ID、grain、Snapshot/Scopeを一連の**実Product Read**で取得 | **N-R1a**：別Tenantの同番号を誤取得。**N-R1b**：JOINで2×3を6 Entityと誤認、または1頁だけで全件成立を主張。**N-R1c**：同Query IDでSQL/Provider/Schemaが変わったのに古いReadを資格化。どれかの実witnessが欠ければ`context/read-incomplete`として保留 |
| **P0-R2 `TestRMTPreWriteRefusalUsesSameBoundTarget`（提案）** | R1で実Readした対象に未保存Patch（子数量を0）を適用し、Owner受理済みRuleのRequiredを固定。**二つのFunction入口**で同一Rule/Versionが実評価されること、実Request/Targetが同じScopeに属すること、Provider call=0／Receipt new=0／別ConnectionのDB前後不変を確認 | **N-R2a**：Callerが必須Period/Grant/Line全件Guardを供給せず、残存CheckだけPASS。**N-R2b**：Validation対象O1、明示Payload O2。**N-R2c**：失敗RunPlanの前段がすでにEffectしたのに`no-effect`と呼ぶ。現Runnerに必要Guard/Target束縛機能がなければ実Gapとして#1129/#2598/#1174へ |
  
**最小実行順**：既存Source/Context fixtureを選び固定→独立Expectedを実Candidateから別に作成→P0-R1をHeadlessに実行→Negative controls N-R1a/bを注入→P0-R2を同じSource/Subject/Providerへ接続。**最初から本番PG書込みを要求せず、書込みは使い捨てDBでopt-inかつ非共有環境**。P0-R2の`Provider call=0`はmockだけではなく、可能な範囲で実DBのBefore/AfterとReceipt件数を別witnessにする。実Readと実Writeの両方を一回で走らせるのはR1/R2の足場がつながった後だけ。

### 11.3 担当者が固定して提出する具体的な成果物

1. **Code path map**：実に呼んだ`workspace.Load → Query/Context Binding → Provider Read`（存在する実関数名で記載）、Go source/branch HEAD、実Test ID。途中が仮Adapter／SQL直叩きならその境界を明記し、製品API完成を主張しない。
2. **Fixture＋独立Expected**：一つの公開隔離Scenario、Owner/Case受入版とWant／実Input／negative controls／Mutant検出可否／Tenant+Subject+Operationの識別。生成AIがExpectedをCandidateから追認しない。
3. **Actual Environment／Result**：Go Toolchain、OS、Provider type/version、実DB接続先の秘密以外の識別、SQL/Query Realizationのdigest、`Required tests discovered/attempted/PASS/SKIP/FAIL`、実Rows/Effect/K/O。DBのDSNをログ／Issueに出さない。
4. **差分判定**：`read-complete/read-partial/unavailable`、`guard-covered/guard-missing`、`correct-target/unbound`、`committed/unknown/not-recorded-only-after-qualified-lookup`、`owner-accepted/unqualified`を**Claim別の説明欄**で表現（新製品enumは追加しない）。
5. **最終引渡し**：#2586/#2506→#1129/#2598→#1174/#1163→#1119、Legacyは#940 Work、Owner Bは#1125/#267。正式Business、Windows、SHO/Consumerの合格票を実Product Read/PG局所結果に付け足さない。

### 11.4 CIの確実性と改訂時の再利用判断

- 本Draft #2783の**一つ前の確定HEAD** `b9d6306b7feb3aed43a9d0df9205df272017b15e`は[Repository Health #42101164557](https://github.com/hat47x/tei-platform/actions/runs/42101164557) `completed/success`（Docs＋自作SQLite Fixtureの範囲）。§10 HEAD `102b51349da7fba56b23d0f2bcf0aa248679553b`の[Health #42106904868](https://github.com/hat47x/tei-platform/actions/runs/42106904868)は本節確認時点**in_progress**。この§11の新HEADも独立に確認し、過去のgreenを転用しない。
- [Draft #2889](https://github.com/hat47x/tei-platform/pull/2889)の専用Go/RunPlan [#42100472921](https://github.com/hat47x/tei-platform/actions/runs/42100472921) `success`と同HEADのRepository Health [#42100472849](https://github.com/hat47x/tei-platform/actions/runs/42100472849) `failure`（gofmt drift）は**両方記録**。局所Go特性がPASSしても整形/全体CI合格ではない。
- 同Sourceが移動してbyte同一、ReferenceがA→B→A、Provider別SQLが切替、Manifestと実Build Inputが乖離、Owner Meaning撤回、後続DB UpdateでO/Bが失効したときは、**必要なClaimだけ再検収**。historical Receiptは消さず、同じDigestだけでcurrent Grant/観測資格を復活させない。

**実施状況**：この追補はSource-staticなFit/Gapと実装受入票である。P0-R1/R2は**提案のみ・NOT RUN**、実Product Data Context Read、実データ書込み、Windows Driver、正式Owner判定、二Target資格は未実証。設計方針は従来どおり、一Provider・一Function Read先行、同Rule二入口、実PGの局所既存Runnerを最大限再利用し、強制SQL記述・新Host/Resolver/Runner・中央Proof Registryは導入しない。

## 12. 11/6追補：Readの入口資格をSource・Role・Reader・独立Expectedで閉じる

§11のP0-R1/R2で「Product APIが存在するか」を先に調べる判断を維持し、他レーンの新しい実Goの**反証**を、追加のシステム共通仕様ではなく**同じRead一往復の停止条件**として組み込む。照合したmainは`af969b88bb89d8262501086b67c2ba6add16e210`（従来§11の基準`8aa18ca...`より3 commit進展）。本章では新規ケース群を別正本化せず、既存RMT-RD1/RD2/WR1、#2586 HJ/QX、#2514 検証基準、#1163 Packetを参照する。

### 12.1 一回のRead資格を三つの独立条件に分ける

| 必要条件（独立の根拠） | 当該Function実行で残す最小Witness | NGの反例・Claim ceiling |
| --- | --- | --- |
| **Input Integrity**：Owner/Source LedgerのData・Relation・Rule・Actionの必要出典と実受理範囲 | Manifest/Import、exact loaded Snapshot bytes、Source Index/Parsed source spans、Role/Type/Ruleの当該Occurrence、Functionが実ConsumedしたModel/Query/Rule、実Build/Entrypointに結び付いたInput Closure | Requiredに無いSourceを「読んだ」とする、Parse後の別byte Fileを同版と称する、Roleの重複宣言が一つに潰れて曖昧性を失う → Source/Meaning資格保留 |
| **Read Fidelity**：Function Context定義と別に固定したBusiness Key/物理PK/Relation Role/Result Grain/対象集合/Predicate/Scope | Provider Alias・Target/Selected Realization＋実SQL bytes/Bind Args、Result Slice（行ID、取得状態、時点、欠損/NULL、完全性、Pagination/Read cohort）、Readerが実受領したRows | 非表示WHERE/ORDER依存、JOIN fanout、partial/hiddenで0件、未観測Read/Read失敗、別Tenant同番号、Query選択変更 → 「正しい全件Read」不認定 |
| **Independent Discrimination**：候補Go/Rule自身から作らない固定Expected、少なくとも一つのKnown Mutant・反証対象の根拠 | Sourceと区別したExpected/Owner status、実Go実行HEAD/Test ID/Case、Required caseがDiscovered/Attempted/PASS（SKIPではない）、mutant rejected、現在Grant/開示に応じたConsumer Witness | 生成AIがCandidate Expectedをそのまま再計算、Fixtureの重複JSON memberが暗黙上書き、別HEADのGreen流用 → 独立検証基準/実Consumer Claim保留 |

**三条件の論理関係**：Sourceのbyte一致だけではProvider Read成功を示さず、実ReadのRows一致だけでは必要な非出力Predicate/Scopeが読まれたと示さず、独立Expectedの局所PASSだけでは正式Business Meaningを受理しない。三条件が各Claimの必要範囲で揃って初めて**限定RMT-RD1 Read witness**を認める。正式Owner業務互換、二入口Owner-write、現在Grant/Commit、Consumer配布は別資格。

### 12.2 他レーンの新しい実証をRead一試行へ最小転写する

- **Load時と後Readの時間差／旧Source Indexの扱い**：[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)の実Go Referenceは、`workspace.Load`時のSnapshotと、その後に同じPathのbyteを書き換えた直接Readが不一致になるcharacterizationを追加した。旧IndexがLoad時bytesを保持することは限定的に確認するが、**全Workspaceのatomic snapshot／消費時の自動TOCTOU拒否**は示さない。RMT-RD1では「ロード時と実Runtime消費時のbyte」または明示Lock世代を照合し、後Readの同Pathだけで旧結果を復活させない（#950/#2823/#2595）。
- **Role重複による情報落ち**：現main `internal/syntax/parser.go`の`parseSemanticData`は`case "role"`ごとに`role = roleTok.Text`としており、重複宣言の最後の値がCandidateに残る。[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)のGo限定反証は「Role一意性を証明済み」という意味ではない。Reference Data/Function Relation Roleは業務対象粒度とAuthorityを変え得るため、重複/競合Sourceがあるときは**Role未受理・Writer/Action対象未確定**に留める。診断実装は#988/#2620、Source編集は#2621へ委譲。
- **実Query/Contextの負依存**：[Draft #2896](https://github.com/hat47x/tei-platform/pull/2896)のSystem CompositionはData Read/Command/Job/Publication/Manual Handoffを異なる成立条件として整理し、#2586 HJを初回実Readに指定する。[Draft #2885](https://github.com/hat47x/tei-platform/pull/2885)はSource閉包とRuntime Read閉包を分け、Not Exists/LEFT JOIN/hidden SQL/partial/Read Cohortを別資格とした。RMTではこれらを**RD2の単一Provider負例**にだけ借り、Plugin方式比較・全Domain Component・二DB実行をP0の必須条件へ追加しない。
- **ExpectedとFixture入口の自己追認**：[Draft #2895](https://github.com/hat47x/tei-platform/pull/2895)は`[1,2]`／`[1,1]`を独立の有限集合性で判定し、Expectedだけ反転／Fixtureだけ反転したnegative controlを追加。これはOwner業務意味の採用ではない。[Draft #2891](https://github.com/hat47x/tei-platform/pull/2891)はFixture JSONの重複**object member**の暗黙上書きを拒否する変更。一方、#2890の`[1,1]`はJSON配列の合法な重複**値**であり、業務Ruleに対する正当な負例。混同しない。#2514/#1180の既存検証基準/Scenarioへ接続し、本PR内で第二Go Harnessを追加しない。
- **DB接続準備と実Readの非同値**：現mainに[merged #2894](https://github.com/hat47x/tei-platform/pull/2894)の`tei dbcheck`がある。実コードは`pgx.Connect`、`SELECT version()`、**`CREATE TEMPORARY TABLE`／`DROP TABLE`**を実行する。よって`dbcheck PASS`は接続・基本操作の環境Readyを示すに限り、業務Query/Readや完全なread-only実行を証明しない。書込み不能な本番接続で無条件に起動しない。DSN未指定exit 2を実Readの合格/不合格に換算しない（#2622/#2586）。
- **Windowsの「緑」と旧Driverの実動**：現mainの[merged #2893](https://github.com/hat47x/tei-platform/pull/2893)はUnix signal依存テストをWindowsでskipする。Windows全Test exit 0でも対象Caseが未実行の可能性がある。#940 X-TEA Driverのactual GUI/JDBC/Tx/独立Readが完成した証拠ではない。必要Windows Test IDsと`PASS/SKIP/NOT RUN`を#1163/Owner票で別管理する。

### 12.3 P0実装担当が最初に提出する差分と停止条件

**対象は同じScenario VS-RMT-01の**P0-R1（読み取り）**であり、RMT-RD2を境界負例、P0-R2を後段へ配置する。新しいAPIを発明する前に、次の4資料を現在のGo実行・正本へ結び付ける。

| 最初のDeliverable | 実行後の受入基準 | 未達なら次に渡すOwner |
| --- | --- | --- |
| **D1: exact Source/Role Bind** | 同じModel/Function/Relation Role/RuleへのRequired出典、Loader actual bytes、Parser Source span、Role重複なし/衝突未解決の分類、実Consumer（Function/Build）までのReachabilityを比較。Byte同一だけでMeaning Acceptedにしない | #950/#1031/#2823、Role#988/#2620、実Build#2595 |
| **D2: actual Function Read** | 実際に呼んだ製品API／Provider／SQLと`Query target/Business Key/Physical PK/Relation Role`の一対一又はOwner宣言対応、固定2子行と負対照を実行。Row grain/pagination/cohort、NULL/未取得/非開示を報告 | #2586/#2506、Data Operations#2622、Mapping#2611 |
| **D3: 検証基準/CI witness** | Candidateから独立したExpectedとmutant（別Tenant/重複Role/Partial/SQL候補変更）、Required Test IDの実Discovered/Attempted/Resultとexact HEAD、実Environment/Providerを保存 | #2514/#1180/#1163、Skipped CI#2835 |
| **D4: Write handoff（まだ実行しない）** | D1〜D3が成立した実Data Context/対象L/未保存Patchと同一Rule版を、二入口の保存前拒否P0-R2へ渡し、検証O1と実Payload/Grant/Owner Targetの同一性を証明することを依頼 | #1129/#2598/#1174、正式Business#1125/#267 |

**No-go**：RMT-RD1のProduct Reader接続が存在しないなら`product-read-not-connected`という計画上のGapとして止め、Go直接DBアクセスや自作FakeをProduct PASSとしない。Source/Role/SQL/Bound Targetがambiguous・partial・staleなら、ProductのData Read/Owner Write Claimを保留。旧X-TEA DriverをWindowsで操作できない場合はP（忠実移植）だけ`legacy-unobserved`を維持し、TEIのA（適合）の局所試験を止めるHard dependencyにはしない。

### 12.4 新HEADへ引き継がない証拠と更新状態

- §11を書いたHEAD `43462eecbd015e72004a93c91617849eb5fe9947`の[Repository Health #42107676122](https://github.com/hat47x/tei-platform/actions/runs/42107676122)は**completed/success**。証明対象は当該Docs/SQLite FixtureのRepo Healthだけであり、実Read P0-R1/R2ではない。
- **後着CI訂正（2026-11-06）**：#2895 HEAD 8d3dc2fcの[Health #42109027966](https://github.com/hat47x/tei-platform/actions/runs/42109027966)はcompleted/success（独立有限検証基準の限定テストを含む）。#2891旧HEAD 2f677921のHealth失敗に対し、修正HEAD 16b8d554の[Health #42110507652](https://github.com/hat47x/tei-platform/actions/runs/42110507652)はcompleted/success（JSON Fixtureの重複object member拒否の限定実装）。#2860 HEAD f5ec1e8aの[Health #42109350029](https://github.com/hat47x/tei-platform/actions/runs/42109350029)はcompleted/failureで、実ログの原因はreference/dsl-language-plugin/self_description_workspace_boundary_test.goのgofmt drift。go vet・go test ./...・Workspace ValidateはPASS。Role重複をProductで拒否できた証明ではない。これらの他HEAD・別契約の結果をRMT Product Read/業務Owner受理へ加算しない。
- 本§12は既存一計画Markdownのみの変更。独立Source/Role問題は現mainの実Code-static観測と別Draftの特性試験を引用しているが、**VS-RMT-01 D1〜D4の実Code path観測、実Function Read、Owner Write、正式Windows Driver、業務Owner受理は新たにNOT RUN／UNQUALIFIED**。新Plugin/Role Resolver/Query Engine/Runner/Host/検証基準 Authorityの導入、main統合、G1/G2 Gate変更は行わない。

### 12.5 最初の実行票への圧縮：P0-R1を先に成立させる

追加の設計正本ではなく、§11のD1〜D4と§12の三条件を**一つの実検収・一つの欠落辺報告**へ縮める。独立Expectedは#2514/#1180、結合証拠は#1163、Gateは#1119を既存Ownerとする。初回は**単一の受理候補Function／一意のDB Alias／一Provider／Readのみ**とし、DB移行・二Provider・Job/SHO/G4完成を先行依存にしない。

| 着手順 | 完成させる最小実物と負例 | Stop／次の担当 |
| --- | --- | --- |
| **1. 実Go入口の確定** | 実Workspace Load byte／Source Index／Function→Context→Query Binding→Provider Readerの実呼出し列と、実SQL/Args・実Read Scopeをコードの**正確な関数名・HEAD**で記録。契約をValidateできることと実Readerを呼べることを区別する | 経路が未接続ならproduct-read-not-connectedを不足の一辺として記録し、#2586/#2506/#2622へ。Fakeや直接SQL ReadをProduct成功にしない |
| **2. 固定Scenarioの実Read** | Tenant AのOrder1001/Header1＋二明細、別Tenantの同番号と別Orderを独立Expectedに固定。**別Tenant混入**と**部分PageまたはJOIN fanoutのEntity数誤認**の2反例を最低限走らせる。実Rows/PK/Relation occurrence/grain/NULL/snapshot/完全性を確認 | 必須Source/Roleの競合、部分取得、対象SQL/Providerの不一致、非表示Predicateの未検査なら限定Readも未資格。#2586/#2514/#1163へ |
| **3. Requiredテストと検証基準** | 実Case ID・Go exact HEAD・必須TestのDiscovered/Attempted/PASS・負例の実Rejected、独立Expectedの作成根拠を保存。#2895のKey重複有限検証基準と#2891のJSON重複member拒否は**別の局所検査**として再利用し、Business MeaningはOwner判定 | Go test全体緑でも必須Case SKIP/未実行なら未検収。Candidateと同じEvaluatorからExpectedを再計算しない。#2514/#1180/#2835 |
| **4. 同Sourceから保存前拒否へ** | 上記Readerと同じOwner Subject/Relation Role/対象集合L、未保存Patch（数量0）、同Rule版を二Function入口へ渡す。必要Guardの全件充足、Validation対象＝Request/Provider実Target、Provider Effect 0/新Receipt 0/別Read不変を確認 | 二入口／Actual Providerに接続できないなら実装Gap。実PG CAS/Unknown/Replayの別既存成功を本Caseに合算しない。#1129/#2598/#1174 |
| **条件付き：DB移行・複数実行先** | 実対象に複数Snapshot/DB Alias/旧新Contractがあるとき**だけ**、DB Instanceと観測世代・Provider・Contractが一意に束縛されたか検査。CompatibilityとMigration Historyの判定は実Read・Current Grantの証拠ではない | main release CheckApplicationは同DB State/ProvidedContractの入力でmap後勝ち、History CompareHistoryはAppliedUnknownを成功扱いし得る。#2897（Draft #2900で修正候補）/#2892/#2555へ。単一DBのP0-R1には無関係なHard dependencyとしない |

**証拠の現況**：#2783 HEAD c3410291の[Repository Health #42110579065](https://github.com/hat47x/tei-platform/actions/runs/42110579065)はcompleted/success。#2895/#2891の修正後CIもそれぞれcompleted/success、#2860はgofmt driftによる全Health失敗。これは各々のSource/テストに限定され、**P0-R1のProduct Go Reader実行・P0-R2保存前拒否・正式Legacy Windows Driver・Owner完了は未実施**。mainの確認時点はaf969b88bb89d8262501086b67c2ba6add16e210。PR #2783は古いbaseから分岐しているため、merge前にはmainとの差分再確認と**更新後HEADのCI**が必要。

**引渡しと停止線**：#2586へ実コード経路と不足1辺、#2514へ独立Expected、#1163へ実Source/Context/Provider/Readerの一試行Packet、#1119へClaim ceilingを返す。欠落証拠は未資格のまま、旧Driver未観測は忠実移植Pだけ保留しTEI適合Aの局所検証を止めない。新Host/Query Engine/RunPlan/Registry/共通受入Schemaは追加しない。

## 13. 11/6実装進展による再配分：X-TEA受容例と本来のRMTを混同せず、実Readの未接続を検収する

基準main：`14238478528e9d0a7a283e1141edb0203467964d`。本章は§12.5のP0-R1→P0-R2の順序を**変更しない**。11/6にmainへ統合された参照Workspace、Command認可・実行段階、実PG試験を再照合し、「既にある前段を再実装しない」と「まだない一つの接続」を分ける。新Scenario SSOT／Host／Rule Engine／Query Engine／Proof Registryは設けない。

### 13.1 新しい実物の位置付け：受容例（E）／原典移植（P）／実製品（A/K）は別

| 実物と現在の実証 | RMTで再利用できる範囲 | これだけでは受理できないClaim |
| --- | --- | --- |
| **main [#2940](https://github.com/hat47x/tei-platform/pull/2940)ほかの`examples/xtea-lowcode`**：`semantic/sales/order.tei`（Order/OrderLine/Payment）、`order_entry.recent_orders`（`datasource sales`、`sql postgres file "recent_orders.sql"`）、CLI/Studio Record・Command Plan、MemoryStore create/delete・cascade/restrict、閉期間拒否。AC-1〜11/RQ-C1〜3の[仮受入票](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/XTEA_LOWCODE_ACCEPTANCE_TRACE_2026-11.md) | 既存Go Loader/Parser/Context/Commandの**合成・MemoryStore基線**としてRead入口候補を絞る。実Query Sourceと宣言DataSourceの対応箇所を実コードで調べる。「候補のテスト足場」として参照 | `UNBOUND_DATASOURCE`警告があり実DB Binding／Query実行なし。`xtea_example_test.go`の実行は`applicationview.Build`、`BuildCommandPlan`、`ReferenceDataExecutor`+`NewMemoryStore`。**旧Driver実動でも実PostgreSQL Function Readでもない**。EのOrder/LineデータをRMT-2の原典Source/Owner同一性に無断転用しない。例にはRMTのTenant負例や二入口共有Ruleの受入済み契約も未確認 |
| **main [#2942](https://github.com/hat47x/tei-platform/pull/2942)・[#3008](https://github.com/hat47x/tei-platform/pull/3008)**：DurableLifecycleとStudio Host合成のstage保存。新Hostインスタンスでも同attemptを拒否する限定Go試験あり | Unknown/Observed段階とOwner/Generationを再起動を跨いで扱う既存Lifecycle契約をWR3で参照。新Lifecycleを実装し直さない | Wails Desktopでの実store合成、PG Effect Receiptとの**同一Attempt連結**、複数Hostでの原子性、原典Legacy Driverの再起動挙動は未証明。Studio既定Command経路はReferenceExecutor/MemoryUnitOfWorkで実外部DBを変更しない |
| **main [#2978](https://github.com/hat47x/tei-platform/pull/2978)・[#2987](https://github.com/hat47x/tei-platform/pull/2987)・[#3001](https://github.com/hat47x/tei-platform/pull/3001)・[#3004](https://github.com/hat47x/tei-platform/pull/3004)**：任意`Authorizer`、CLI/Studioの参照AllowList、報告Effectと宣言Effectの照合・Operation Record | WR1/WR2の**事前拒否の位置**とActivity traceの既存実入口を再利用。選択したCapability/Period/Source/Requestを記録する | `Authorizer`未設定またはCLI/Studio AllowList空の従来経路は**未認可を意味しない**。AllowListは能力単位のReference検査で、Current Host Principal+Resource/Target Grant/Provider Commit fenceを代替しない。Operation Recordは実DB Receipt/Owner完了ではない |
| **main [#2907](https://github.com/hat47x/tei-platform/pull/2907)・[#2910](https://github.com/hat47x/tei-platform/pull/2910)**：異Payload・同OperationID拒否の実PG adapter、Socket-level COMMIT応答喪失の限定実PG test。main [#2977](https://github.com/hat47x/tei-platform/pull/2977)の暫定D8–D11実PG E2E監査 | WR3の実故障注入/Receipt照合は**既存Testを先に使う**。模擬ACK-loss以外のSocket断証拠も得られたため「実接続断の証拠なし」という古い一般記述は撤回。ただし同一業務RMT Caseとして再確認する | #2910は実TCP Socket断であり実OS process死・別Key ABA完全防止・二Host分散fenceではない。#2977のAuditは`INCOMPLETE/productGate UNQUALIFIED`、provisional Grant・意味受理・保証は正式資格でない |
| **main [#3005](https://github.com/hat47x/tei-platform/pull/3005)・[#3012](https://github.com/hat47x/tei-platform/pull/3012)**：Period/Effect/RunPlan/Permissionの実PG向けFixture 4系列、記録先行のGrant Issue契約 | U1 GuardのRequired/Supplied実証のテスト基線、付与の出典と実使用を別票化する材料 | #3005のテストは`TEI_TEST_POSTGRES_DSN`が無ければSKIPで、[実DB判断計画](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/REAL_DATABASE_RUNTIME_DECISION_PLAN_2026-11.md)も実行NOT RUN。#3012のIssuerは発行主体の第三者承認でなく、Grantの発行記録があるだけでは当該Functionの現在のProvider Grantを証明しない |

### 13.2 Product Function Readに関する実装Gapを狭める

**初回P0-R1は追加コードではなく先に既存コードの呼び出し線を証拠化する。** `examples/xtea-lowcode/applications/order-entry/queries/recent_orders.tei`には`datasource sales`と`postgres` SQLファイルの明示宣言があるが、例の環境には実データソースのBindingがない。ここは「SQL定義不足」ではなく、**選択Context→Alias/DB Instance→Read Provider→SQL実行→Result Slice**という経路が同一Productで成立するかの実測Gapとする。#2586/#2506/#2622/#2611の既存契約を順に使用し、ない接合点だけ最小変更にする。

対象は二つの**異なる**比較対象として記録する：
- **E（参考用合成）**：現在の`examples/xtea-lowcode`。Loader/Query宣言/MemoryStoreとCLI/Studioの局所試験を再利用する。実DBを接続していないまま「SQLを実行した」と報告しない。例のSourceとOwner/Legacy移植に必要なOriginal Sourceは同一物ではない。
- **R（原典に基づくVS-RMT-01）**：#950/#940のprivate原典Source/Function/Rule/Owner Mappingを固定し、Tenant A/Order1001＋明細2行など既存§10の独立Expectedでテストする。Eの成功でRの業務意味受理・原典Parity（P）を自動受理しない。LegacyとTEIは別Attempt。

**RMT-RD1出力の最小列**：`actual Product call path`、`Source bytes/Role occurrence/Function/Query Context`、`Alias/Instance/Provider/selected SQL bytes+Args`、`Row identity/Business Key/Physical PK/Relation grain/Scope/Completeness/Read epoch`、`Independent Expected/negative control`。Product APIが未接続なら、Provider契約の静的検証やGo直接SQL Readerを「完成」とせず`product-read-not-connected`を保持。作業開始時は**一Provider・一Tenant/Orderのread-only固定Case**に留める。

### 13.3 P0-R2後段の拒否・権限・Receiptは二重実装しない

P0-R1に成功しても自動的にWrite Allowは発行しない。同じRuleで二つのFunction入口を比較する際、`Required Guards`（Period/Owner Target/Rule/Current Grant/ExpectedVersion/Observer等）は独立の業務/Host契約から確定し、`Supplied/Evaluated`と`Effect直前Enforced`を分離する。

特に**AllowListに設定がないReference実行**、Grant発行記録だけ、Studioでの`committed`表示、`OperationRecord`、Durable Stage保存は**実ProviderのEffect/Receipt Kの代用にならない**。WR1の拒否をProvider effect 0／new Receipt 0／別DB接続不変で検収し、WR2/3では既存`commandadapter.Executor→effectcommit.Runner`経路と同一OperationID、Current Grant、実postimage、Lookupを要求する。Source/Target/Payload不一致やACK失落後の新Key盲再送は拒否。実Source/Consumer/Ownerにより確認されない限りG1/G2の正式判定はUNQUALIFIED。

### 13.4 実装引渡しの最小差分と失効条件

| 優先 | 担当と次の実物 | 判定が止まる条件 |
| --- | --- | --- |
| **P0-A** | #2586/#2506/#2622：既存X-TEA受容例で`Query宣言→実選択/Binding`のコードパスを特定し、**実Reader呼出しの欠けた一辺**と必要Adapter数を報告。RMT Sourceとは別のE Probeであることを明記 | Alias未束縛、SQL静的検証のみ、Read Scope未確定、Query同一IDでも違うSQL/Provider、Line粒度/0件/partial不明 |
| **P0-B** | #1129/#2598/#1174：RMTのR Scenarioに同じData Ruleの二入口、未保存Patch、対象O1→Payload O2負例、完全なGuard被覆を結び、実PG Effectと別Readで検収 | Reference AllowList／MemoryStore／Durable Stage／Operation Logだけの合格。S02/S05/Current Grant/Provider Targetと実Commitが同一Attemptに無い |
| **P1** | #1163/#2514/#1119：E（受容例）とR（原典移植）を別Attemptで保持したEvidence Packet、実CI Test IDs（SKIPの区別）、原典#940、Business Owner B・Consumer Dを別資格で集約 | 異HEAD・別FixtureのPASSを加算、Provisional Grant/Atomic guaranteeを正式受理へ昇格、Source A→B→A/Role/SQL/Grant改版で再資格化を省略 |

**変更履歴の読み方**：この§13は2026-11-06 main実コードとmerged PRの再分類であり、P0-R1の実Product SQL Read・RMT Owner write・Legacy Windows Driverの新実行結果ではない。Draft #2783自体の前HEAD `6d9af6dec77b7460cc5b9293f491392af455557f`は[Health #42112265037](https://github.com/hat47x/tei-platform/actions/runs/42112265037) `completed/success`（Docs/Fixture）。本追補のCIは新HEADで別確認する。mainの進捗と大きな分岐差があるため、**このDraftを無検証でmerge/rebaseしない**。Owner判断が必要な原典互換・受理範囲・旧Runtimeは留保を維持。


## 14. 初回Queryの静的配線を固定し、Header Readと明細の完了証拠を分ける（2026-11-07）

本章は§12.5／§13のP0-R1を**現mainの実Sourceで補正する一つの検収差分**。既存E（合成受容例）／R（原典忠実移植）／K（実Provider確定）の三つのEvidenceを混合しない。実装・Loader刷新・Second Query Engine・新Runnerを採用するものではない。

### 14.1 初回E Probeは「Query選択の宣言がない」段階ではない

2026-11-07時点のmainに次の静的な連鎖がある（Sourceは実GitHubファイルを直接確認。各Sourceの意味受理・実行済みを推定しない）。

| 段階 | 現行Sourceと内容 | 未検証の境界 |
| --- | --- | --- |
| Workspace | `examples/xtea-lowcode/tei.workspace.yaml`：`semantic/applications/databases`をsourcesへ指定 | 読込から実RunまでのInput byte・選択／版 |
| Pattern | [`applications/order-entry/patterns/recent-orders.tei`](https://github.com/hat47x/tei-platform/blob/main/examples/xtea-lowcode/applications/order-entry/patterns/recent-orders.tei)：`pattern order_entry.recent_orders browse`、`query order_entry.recent_orders`、`data sales.order`、`filter sales.order.status {eq ne} parameter status`、`page-size 20` | **Pattern宣言のQuery参照が、実Runtime ReaderのQuery選択へ渡った証拠はない** |
| Query/SQL | [`queries/recent_orders.tei`](https://github.com/hat47x/tei-platform/blob/main/examples/xtea-lowcode/applications/order-entry/queries/recent_orders.tei)：`input status std.code`、`datasource sales`、`sql postgres file "recent_orders.sql"`、`result order_id/customer_id` → `sales.order`属性 | 実引数・SQL実消費byte・Row/Field Mappingと実Statementの同一性 |
| App → DB定義 | [`application.tei`](https://github.com/hat47x/tei-platform/blob/main/examples/xtea-lowcode/applications/order-entry/application.tei)：`database sales {target sales_core; access read-write}`；[`database.tei`](https://github.com/hat47x/tei-platform/blob/main/examples/xtea-lowcode/databases/sales-core/database.tei)：`database sales_core {kind relational; ownership managed-here; uses semantic sales}` | **論理Alias→実DB Instance/接続・Tenant/Scheme・Credential/Current Grantは別**。定義上targetがあることをRuntime Binding済みとしない |
| SQL原文 | [`recent_orders.sql`](https://github.com/hat47x/tei-platform/blob/main/examples/xtea-lowcode/applications/order-entry/queries/recent_orders.sql)：`select order_id, customer_id from sales.order where status = :status` | `order`という識別子の方言上の扱い、実Schema名/物理table、`:status`からProvider PlaceholderへのBindingを実Dialect/Executionで検証。**現時点でPG実行成功／SQL不適合を断定しない** |
| 既存Go Test | [`cmd/tei/xtea_example_test.go`](https://github.com/hat47x/tei-platform/blob/main/cmd/tei/xtea_example_test.go)：Workspace Validate、`order_entry.order_record`の`applicationview.Build`、Command Plan、Package、MemoryStore CRUD/cascade | このTestは**recent_ordersの実Query Read／実PostgreSQL Rowsetを実行していない**。例の`UNBOUND_DATASOURCE`は未解決 |

**精密化したGap**：`Pattern→Query`と`App Alias sales→logical DB sales_core`の**宣言辺は存在**する。一方、実製品での`Pattern/Function Context→selected Query Runtime Binding→physical DataSource Instance/Provider→realized SQL+params→Rowset/Scope`は未証明。静的な宣言辺を「不存在」と誤報しない一方、宣言を実行成功へ昇格させない。実Product Code Pathの不存在までをこのSource-static探索から断定せず、#2586/#2622の実Call Path調査で初めて`product-read-not-connected`か結線済みかを分類する。

### 14.2 最初の独立ExpectedはHeader Read限定、Completion対象Lとは分離

このQueryのResultは`order_id`／`customer_id`の**受注ヘッダ相当2列**であり、`sales.order_line`・明細ID・数量・承認／Handoff証拠を返さない。`page-size 20`も完全母集団を宣言しない。したがって§10/§12.5の「Order1001＋二明細」は**次の明細Data Context用の別Read要件**とし、この初回Queryの成功だけで`CompletionEvidence.Coverage=complete`または`EmptyJustified=true`を供給してはならない（merged #2926はCallerが渡したCoverage/Empty証拠を評価するだけ）。

| 既存#1180/#2514に渡す対照ID | 初回実証の合格条件／反例 | Claim上限・停止 |
| --- | --- | --- |
| **RMT-HR-01 Header query positive** | 実Source/SQL digest・`status`の入力→実bind args→実ReaderのRow/Column values・Provider/Scope・独立Expectedを同一Attemptで確認。Caseは自作固定ヘッダを使用 | **Header Read限定**。2列の値一致≠明細全件・業務Rule・Owner完了 |
| **RMT-HR-02 filter mismatch** | 独立Fixtureで`status=open/closed`を区別し、誤ったbind、未束縛、別Statementによる偽一致を検出。欠損入力／NULLを無断defaultへ補完しない | Input契約/SQL実行未確認なら未資格。Query宣言のtest PASSだけでは不可 |
| **RMT-HR-03 page/cohort** | 21件以上の候補・同じstatus・変動ページ／返却0件を対照し、`page-size 20`とSQL原文の（明示的ORDER BYなし）境界を実Runtimeで確認。Snapshot/Stable orderingがない場合は未確定を表示 | Page1が20件/0件というだけで全件/不存在の証拠を作らない |
| **RMT-HR-04 physical scope** | 実DB Alias・Schema/physical table・Current Tenant/Reader filtering/RLS等、明示的Scopeを計測。SQL原文にTenant predicateが無い点は、実TargetのTenant分離保証と分ける | 多Tenant環境なら混入負例必須。単一Tenant限定なら根拠を付けN/A。Alias `sales`だけで隔離を推定しない |
| **RMT-HR-05 details completion handoff** | `order.lines`の全Target L・grain・実Version/Scopeを**別Function Data Context／Query／Read witness**で完全取得した後、独立Expected・`EvaluateCompletion`へ渡す | Header SQLの成功・Paging 0件は`CoverageComplete`/Handoff確認へ流用不可。部分集合なら`unverified`保持 |

**P0最小手順**：HR-01＋HR-02を同じ実Reader Caseで先に実行、条件が成立してからHR-03/04を追加し、HR-05は明細Contextが実際に接続した後段へ置く。First ReadとOwner Write/Windows旧Driverを一度に動かす新Dependencyは作らない。

### 14.3 11/6夜のCommand/Grant/監査実装を後段の正しい資格へ配置

- [merged #3033](https://github.com/hat47x/tei-platform/pull/3033)の`ReferenceDataExecutor→AppliedOperation→OperationRecord`は**参照Storeのレコード単位の自己報告**。`checkAppliedOperations`は`Applied=[]`を無検査で通し、同種別の追加Data操作もRelation/Targetを検証しない。参照報告なし≠Effect 0・合法な全Effectとせず、#1174の実PG Effect/Receipt/別Read負例へ引き渡す。
- `command.Invoker.execute`のApplied/Outcome検査は`Executor.Execute`**後**。すでに別物理TxでCommitしたProviderへの後段検査Errorを`no-effect`としない。実`commandadapter.Executor`はcommitted/replayed両者を空Command Resultに潰すので、Command Result単独ではK/Replay/Owner完了を証明しない（#1174/#1129/#1163）。
- [merged #3034](https://github.com/hat47x/tei-platform/pull/3034)はCommandの事前`Authorizer`を追加。[merged #3032](https://github.com/hat47x/tei-platform/pull/3032)と[merged #3035](https://github.com/hat47x/tei-platform/pull/3035)で`issuedBy`/宣言内Grant解決は進んだが、Issuerとresolved rootは署名された外部Authorityではない。Preflight許可は実Effect直前の同Tx current Grant・Target一致を代用しない。
- [merged #3025](https://github.com/hat47x/tei-platform/pull/3025)はprovisional chainを実PG＋exact-head Health Receiptで再実行し、`TRUSTED_FULL_HEALTH_NOT_RECORDED`を解消。ただし**provisional 3診断は残り`productGate=UNQUALIFIED`**。本文§13.1に残る「実DB判断計画は未実行」は、[#3032 実PG CI success](https://github.com/hat47x/tei-platform/actions/runs/42235336807)の**その限定実装に関して更新**する。RMTの同一Attempt/正式業務Grant/Owner Bではない。

### 14.4 `*.tei.md`移行・UI後着を同じ受入へ足し合わせない

Ownerが11/6に`*.tei.md`統一を決定し、[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)は移行計画、[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)は未対応Parserを`TEI_MD_PARSER_UNBOUND`で拒否する**限定Go CI成功**。しかし`*.tei.md`によるNative Meaning実読込・Query実行はNOT RUN。現`*.tei`のHR-01を先行し、移行後は同Expected・別Source Revision/AttemptでParser→Query→Readerを再資格化する。`*.tei`のSourceが改名されても意味が不変と断定せず、実消費byteとLocator/Source Indexが一致するか確認する。

Studioの[Draft #3036](https://github.com/hat47x/tei-platform/pull/3036)は旧RPC応答の表示失効を扱うが、UIの旧結果不採用はProvider Requestの取消、DB Rollback、現在Grantの維持ではない。Wails／Windows旧Driverの実操作観測は#940の別Laneとして保持。

### 14.5 具体的な引渡しと証拠上限

- **Product Owner #2586/#2622/#2611**：HR-01の実Source Chain（Pattern→Query→Application Alias→logical DB）を起点とし、最初に欠けている**Runtime Binding/Physical DB/actual SQL Execution/Rowset**一辺を関数名・blob/HEADとともに報告。修正が必要なら既存Go APIへ小さいAdapterと独立Expectedを追加し、Fake/直接DB接続をProduct Successにしない。
- **Data/Completion #2506/#1125/#267**：HR-05の明細全件性/空証明を、Inputの`CoverageComplete`文字列ではなくProvider Scope/snapshot/Provider resultの観測で裏付ける。最初のHeader Probe成功では業務Completionを評価しない。
- **Host/Effect #2598/#1129/#1174**：P0-R2で「Executor**前**に拒否→Effect/Receipt 0」と「Executor**後**検査Err→実Effect unknown/committed」を別Caseにし、Current Grant/Target/Receipt/独立Readで判断。Grant発行ログ・参照OperationRecord・UI ResultをEffect Receiptに換算しない。
- **統合/検証基準 #1163/#2514/#1180/#1119**：同じExpectedでもE/原典R/新Authoring Sourceは別Input/Attempt。Observed Test IDs/PASS/SKIPと実Binary/SQL/Provider/Read Cohortを束縛し、足りないClaimは`unknown/not-run/unqualified`とする。

**この追補の資格**：Source-staticなGitHub現mainの宣言／Go Test、既存merged PR・限定CIを再照合し、Draft #2783の既存計画本文だけ追補。**新たなGo test実行・PG Rowset／業務SQL実行・初回HR-01〜05・Windows Driver・実Web/Studio GUI・Owner正式意味受理はNOT RUN**。既存§12.5 P0-R1を優先し、G1/G2/原典parityは未資格。既存の別担当PR・Fixture/実コードに変更せず、Draftのままとし、**新HEADのtrusted healthとmainとの差分照合前にReady/mergeしない**。

## 15. 11/7横断再資格：空集合、Command Unknown、Applied差分、Product実Readを接続する

§12.5〜14の優先順（**P0-R1実Reader→P0-R2同Rule保存前拒否→実PG Effect/Receipt/独立Read→原典Rとの比較**）は不変。今回の変更は2026-11-07のmain実Source・merged PRに基づく**誤った旧記述の訂正と検収対象の限定**。E（合成受容例）とR（原典忠実移植）は引き続き異Source・異Attempt。新Completion Engine、Provider、Runner、Gate、権限正本を追加しない。

### 15.1 P0-R1：Workspace管理・Loader読込・実行消費を三段で検収

[merged #3085](https://github.com/hat47x/tei-platform/pull/3085)の`workspace-files.readByLoader`は、管理対象の存在とLoad中の読込を区別できる補助証拠になった。しかし`readByLoader:true`は**ParserのMeaning受理・Build入力・Runtime Query選択・実SQL実行を証明しない**。

最初のRMT-HR-01/02は、現行`examples/xtea-lowcode`の宣言済み`pattern order_entry.recent_orders browse→query order_entry.recent_orders→datasource sales→logical sales_core`を出発点とする。実Go Function/Pattern入口→選択Query→**物理**DB Instance/Provider→選択SQL statementと実bind`status`→実Rowset/Field・Scope/時点・独立期待結果までを**同一Attempt**に束縛する。SQLファイルをLoaderが読めたこと、`dbcheck`、参照`view`、検証成功、`UNBOUND_DATASOURCE`警告の残存を実Reader成功としない。

HR-01はHeaderの`order_id/customer_id`取得、HR-02は`status=open/closed`・missing/NULL/誤bindの負例。20件のBrowse pageと明細全件性はHR-03〜05の別Evidence。現在Queryが返すHeader2列だけでは、明細対象集合`L`の完全性を主張できない。Product call path不明なら最初に欠落する`Runtime Query selection/Physical Binding/Reader`の辺だけを#2586/#2622へ報告し、Fake/直接SQLの動作をProduct E2としない。

### 15.2 完了判定：空の観測を業務否定へ昇格しない修正は統合済み

[merged #3051](https://github.com/hat47x/tei-platform/pull/3051)でmain`internal/semantic/completion.go`の`EmptyPolicyDeny`は、`Targets=[]`かつ`Coverage!=complete`で`fails`ではなく**`unverified`**を返すようになった。これは既存RMC-03の旧実装懸念への限定修正。対象内で確定した違反`TargetResult.Holds=false`なら部分取得でも`fails`、本当に空でCoverageCompleteならdenyの`fails`を維持する。

ただし`CoverageComplete`と`EmptyJustified`の真正性はCallerが供給する証拠に依存する。したがってRMC-02（部分PageなのにComplete申告）、RMC-04（異Scopeの空証明）、RMC-05（空Handoffと外部確認のMeaning）は**実Provider/Owner受理の別試験**として維持する。空集合の分類はBackendの技術的完全性と業務完了の意味を跨ぐので、Owner判断をEvaluationから自動推測しない（#1125/#267/#2506）。

### 15.3 Command：古い「未検査・拒否」から、現行の「前置拒否・事後Unknown・差分記録」へ更新

| 既存F系列を再利用する対照 | mainの実装済み範囲 | 次に必要な独立検証基準 |
| --- | --- | --- |
| **F-Pre：認可・期間欠落** | [merged #3042](https://github.com/hat47x/tei-platform/pull/3042)：`Authorizer==nil`→`COMMAND_AUTHORIZER_UNAVAILABLE`、期間宣言に対し`Periods==nil`→`COMMAND_PERIOD_STATE_UNAVAILABLE`。認可が期間より先に実行される | 当該HostでのExecutor未呼出と、実DB Effect 0／新Receipt 0。`ReferenceAuthorizer{}`は参照用の明示的全許可であり現在の正式Grantではない |
| **F-Post：Executor後に宣言外Effect/Outcome/Applied** | #3042で`EffectUnknown=true`としてLifecycle`unknown`へ送る。旧来の「拒否＝無作用」の誤分類は**修正済み** | 既に実Providerが別TxでCommitしたかは`unknown`だけでは分からない。同じOperationID/RequestHashのReceipt Lookupと独立再読取を実施し、新ID盲再送を止める |
| **F-SameKind：宣言外Dataへの同種操作** | [merged #3055](https://github.com/hat47x/tei-platform/pull/3055)：`additional-scope`を`OperationRecord.Differences`に残す。合法cascadeを許容するため、このDiff単独で実行拒否しない | 対象Relation/Grant/Source Scopeに合う合法連鎖か、無関係Dataへの作用かをOwner受理と実Provider/Oで区別。`Differences`記録は許可や完全性の証明ではない |
| **F-NoReport：Appliedが空** | main`checkAppliedOperations`は長さ0で検査を省略し、Recorderは`AppliedUnobserved`を記録できる。`effectcommit/commandadapter`はCommitted/Replayedを空Command Resultへ変換 | 未報告≠Effect 0、Command Success≠新規Commit/Replayの判別。実K・独立O、Request identityを別観測 |
| **参照Commitの読み違い** | [merged #3054](https://github.com/hat47x/tei-platform/pull/3054)：参照CLI/Studioの`committed:true`は`persisted:false, durability:none`と併記。Memory UnitOfWorkのCommit呼出と永続化を分けた | 実Provider経路の`committed/persisted`定義と独立読取保証は別途受理。参照表示をK/Oに転記しない |
| **Grantの申告とHost有効性** | [merged #3047](https://github.com/hat47x/tei-platform/pull/3047)にDeclared/Issued permissionの`GrantAuthorizer`局所試験（CLI/Studio/Provider未結線）。[merged #3035](https://github.com/hat47x/tei-platform/pull/3035)のIssuer/ResolvedByは署名無し | Current Subject/Resource/Target/Period/Grant epochをProvider Commit直前に拘束し、選択O1→明示Payload O2やGrant撤回を実DBで検収。文字列`issuedBy`を外部承認証明としない |

[merged #3053](https://github.com/hat47x/tei-platform/pull/3053)の参照CLI`COMMAND_EFFECT_UNKNOWN`と`COMMAND_LIFECYCLE_REFUSED`、[merged #3090](https://github.com/hat47x/tei-platform/pull/3090)の参照Store`COMMAND_REFUSED`は、**Consumerに返す診断の区別**であって実ProviderのEffect 0／Receiptの新証拠ではない。既存#1174のF-Pre/Post/NoReport/StaleGrant/Issuer/SameKindを流用し新Fixture正本を作らない。

### 15.4 実PostgreSQLの更新された到達点・残るG1/G2

[merged #3088](https://github.com/hat47x/tei-platform/pull/3088)はmainのmerge commit `a512dd491a92b3ed0dc775596380262968bfac3c`／PostgreSQL 16.15で、閉/開期間、Observer不能、再Dispatch拒否、Grant無し/有りの**6試験PASS、SKIP 0**を記録。以前の「実DB判断はDSN無しで全て未実行」という記述は**この限定6件**について修正する。ただしこれらはRMTの同一Source/Function/Query/Operation/Consumer/Ownerに揃った実行ではない。

[merged #3025](https://github.com/hat47x/tei-platform/pull/3025)のProvisional Chainは健康診断Receiptを確認済みだが、Provisional Acceptance/Grant/Guaranteeの3診断が残り`productGate=UNQUALIFIED`。[merged #3093](https://github.com/hat47x/tei-platform/pull/3093)ではProvisionalの資格不備とAtomic不足を**独立した診断**として正負試験で区別した。形式的なHealth PASS・他Attemptの実PG Testの成功をG1/G2へ足さない。#1163の既存一改訂Packetに、Actual Source/Runtime Binary/Provider/Required Guard/Receipt K/別Reader O/Business B/Disclosure Dと必須Test IDのPASS・SKIPを対応づける。

### 15.5 引渡し順とソース形式・テスト用語

**最初の一手は依然HR-01/HR-02だけ**。#2586は実Function→Query→物理Providerの呼出し線と最初の欠落辺、#2622/#2611は実Binding、#2514/#1180は実装から独立した期待結果、#1163は同Attempt Packetを担当。成功後に明細Contextの完全性とCompletionを検査し、最後に同Rule二入口のP0-R2と実Effect Fenceへ進める。Windows原典Driverは別Attemptであり、RMTの忠実移植Rのみ未観測として保持。

`*.tei.md`への標準移行は[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)が形式・検収、[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)がParser未束縛時の明示拒否を担当。現`*.tei`のP0-R1は形式移行待ちにしない。移行後は同じ独立Expectedでも別の実消費Source Revision/Attemptで再資格化する。

#3039／[merged #3040](https://github.com/hat47x/tei-platform/pull/3040)の用語方針に合わせ、テスト判定の呼称は「独立検証基準」「独立期待結果」「業務判定基準」に寄せる。DB製品名や外部コード識別子は変更しない。

**実証状態**：前計画HEAD`d2c992fbeeb3bb4cc978454ea3eadfea823d0e4a`の[Repository Health #42256444830](https://github.com/hat47x/tei-platform/actions/runs/42256444830)はcompleted/success。**本節を追加した後のHEADのCIは別に資格化する**。この追補は既存計画Markdownのみの変更で、新Go/実Function SQL Read/同業務実PG K/O/Windows実Driver/Owner意味受理はNOT RUN、G1/G2はUNQUALIFIED、Draftのままmain未統合。

## 16. 11/7実行境界の具体化：BrowseViewの定義解決と実Rowset、効果報告の二つの未観測

本章は`14–15のP0-R1（既存X-TEA合成受容例Eの`order_entry.recent_orders` Header Read）を、**確認した製品Go call path**で具体化する。Function/Business忠実移植R、実Provider K、Owner BをEの成功に合算しない。第二Query Engine、Runner、検証基準、独自Hostを導入しない。

### 16.1 Code-staticで確定した停止辺：BrowseのQuery選択は実装済み、CLIの実DB Readではない

現main [`internal/applicationview/browse.go`](https://github.com/hat47x/tei-platform/blob/main/internal/applicationview/browse.go)の`BuildBrowse(world, applicationID, patternID)`は、Application→Browse Pattern→`pattern.Query`に一致する`world.Queries[i].ID`を選び、Application一致とResultColumnsのSemantic属性、Filter/Sort/PageSizeを検証して**表示用`BrowseView`**を返す。従来の「Pattern→Queryの宣言しかなく実Go選択がない」という一般化は正しくない。**定義の選択・表示投影は現物としてある**。

一方、[`cmd/tei/cli_view.go`](https://github.com/hat47x/tei-platform/blob/main/cmd/tei/cli_view.go)の`runViewCommand`は、`PatternBrowse`の場合`BuildBrowse`を呼び、`report.Result=view`へ投影を入れる。この実枝にSQL実行・実Provider Rowset取得はない。現`BrowseView.Query`や結果列は**定義情報**であって実取得行ではない。したがって、この特定入口の次の欠落辺は**選択済Query→Runtime Execution Binding→物理DataSource/選択SQL+実args→実Provider Reader→Rows/Scope**。全Productの別経路まで不存在と断言せず、#2586が既存実Reader Callerの有無を実APIで判断する。

### 16.2 HR-01/02へ足す独立検証基準（新Fixture正本は作らない）

| 検査の焦点 | 同一Attemptの期待と負例 | 禁止する昇格 |
| --- | --- | --- |
| **HR-01／正対照** | 現行Workspace→`BuildBrowse.Query`と選択SQLの原byte→実Aliasと物理DB Instance→`status`引数→実ProviderのSQL実行→`order_id/customer_id`のRowset・Consumer Bodyを**一試行**で独立Expectedと照合する | `BrowseView`成功や`readByLoader:true`をProduct実ReadのPASSにしない |
| **HR-02／同ID・異入力** | 同じQuery IDでもSQL WHERE／Schema列／Dialectを一因子ずつ変え、Request/Source fingerprint・Revision・Worker実体・実消費SQLを別に固定。古い結果を新Requestへ結び直さない | [merged #2657](https://github.com/hat47x/tei-platform/pull/2657) QX-14のGoサブプロセス**模擬Worker**は同IDだけでは入力帰属を証明しない局所反例。実SQLGlot／実PGの適合ではない |
| **HR-02／非出力依存** | 出力Header2列が同じでもWHERE/ORDER、Rule、秘匿列の実SQL ReadをContext用途別に観測する | [merged #2665](https://github.com/hat47x/tei-platform/pull/2665)の実SQLite 14/14 PASSは`SQLITE_READ`**コンパイル時の列参照**。実Row/Storage/PG RLS/Host開示の証明ではない |
| **HR-03〜05／ページと明細** | 20件のBrowse View、Header0件・Page1の成功から`sales.order_line`対象集合Lの完全取得/空集合正当化を推論しない。別Data ContextのRow identity/Scope/cohort/Read epochを要求 | 完了判定[merged #3051](https://github.com/hat47x/tei-platform/pull/3051)の空Deny未取得→`unverified`は修正済みだが、CallerのCoverage/Empty申告をそのまま信頼しない |

Source A→B→Aでは同byteでもObservation時点・選択Revisionが別となる。初回HR-01/02の実製品SQL Reader試験は本章では**NOT RUN**。`BuildBrowse`のGo選択済みという証拠と、本番Providerの実Readを分離する。

### 16.3 Commandの「効果宣言あり／報告なし」と「Applied未列挙」は直交する

[merged #3112](https://github.com/hat47x/tei-platform/pull/3112)により、[`internal/command/invoke.go`](https://github.com/hat47x/tei-platform/blob/main/internal/command/invoke.go)と`operation.go`は`plan.Effect!=nil && result.Effect==nil`なら`ReportedEffectMissing=true`を記録。これは**Executorが宣言効果を報告しなかった**印で、`AppliedUnobserved`（効果を宣言したが操作を列挙しなかった）とは別。後者は[merged #3104](https://github.com/hat47x/tei-platform/pull/3104)で`plan.Effect!=nil`の場合だけ立てるよう修正済み。`Differences/additional-scope`は[merged #3055](https://github.com/hat47x/tei-platform/pull/3055)の独立の差記録。

| 当該Commandで観測できたこと | 正しく示す状態 | 主張できないこと |
| --- | --- | --- |
| 効果宣言あり、`Result.Effect=nil`、`Applied=[]` | `reportedEffectMissing=true`と`appliedUnobserved=true` | 効果0、結果成功＝新規Commit |
| 効果宣言あり、報告Effectあり、`Applied=[]` | 効果を**自己報告**したが、個別操作は未列挙 | 実DBへ宣言効果が確かに適用されたこと |
| 効果宣言なし、`Applied=[]` | 比較対象なし。`appliedUnobserved=false` | 外部効果0を独立観測したこと |
| 同Kind別Dataに追加適用し`Differences`あり | 未解決の追加Scopeを記録 | 合法Cascade、Grant/Relation/Owner意味の確定 |

重要：実`effectcommit/commandadapter.Executor`は`Committed/Replayed`を空`command.Result{}`へ変換している。報告欠落を**一律EffectUnknown拒否にする案は現行正常経路を壊す**ため採用しない（#3112）。実ProviderでCommit/Replay識別、Operation ID/Request Hash、Receipt K、別接続Postimage O、Current Grant/Targetを別証拠とし、未報告を勝手に「無作用」にしない。`ReferenceAuthorizer`、MemoryStore、Studioの`committed:true,persisted:false,durability:none`は本番保存Receiptではない。

### 16.4 並行レーン成果の適用上限

- [merged #3108](https://github.com/hat47x/tei-platform/pull/3108)は`hook.CommandCaller`によるHook→Commandの局所委譲契約を提供する。ただし`hook.Runner`の**実配線未実装**、参照認可使用、実DB未接続。RMTの二Function実入口が成立したとしない（#1125/#1129）。
- [PR #3113](https://github.com/hat47x/tei-platform/pull/3113)はDSH外部Harnessから固定TEI実行物をread-onlyで32回起動し、CLI `commands/version/model catalog/inspect/source-index/validate`を実観測した独立Attempt。これはCLI到達証拠で**Product Query ReadやEffect/Owner受理ではない**。実行範囲外の読み取り可能パスへアクセスできた点は参加者隔離の未達であり、Harness試験の適格性はそのOwnerで管理する。
- [merged #3115](https://github.com/hat47x/tei-platform/pull/3115)はX-TEA Modelerの設計書/DDL/行列等の**派生出力受容設計**。Source版・生成規則・未解決を残す提案であり、旧Driverの原業務実動、TEIの意味Parity、実Provider Readの新証明ではない。原典R/SHOの別受入へ渡す。

### 16.5 次の一手と証拠の状態

最初は**既存HR-01/02のみ**：#2586/#2622が`runViewCommand→BuildBrowse`を固定し、既存Product Readerへ至るCallerがあれば実SQL/Rows/Scopeと同じRunで検証、無ければこの入口に限った`browse-query-execution-unbound`を既存`product-read-not-connected`の内訳として返す（新Status enumは作らない）。#2514/#1180は同ID異SQL・非表示Columnの独立Expected、#1163はSource/Request/Provider/Consumer実行とSKIP/NOT RUNの一改訂Packetを所有。P0-R1が実証できてから#1174の事前拒否／Executor後Unknown／二つの未報告欄／K/O/Grantを同一業務Attemptで照合し、旧Driver/Windows/Owner意味受理はさらに別。

**資格**：現行mainの製品Go Sourceの静的追跡、[merged #3112 exact-head Repository Health #42308140870](https://github.com/hat47x/tei-platform/actions/runs/42308140870) completed/success、[前計画HEAD 1fe1e455のHealth #42301279950](https://github.com/hat47x/tei-platform/actions/runs/42301279950) completed/successを**それぞれの範囲**で参照。今回の新HR実Reader・実PG業務K/O・Windows Driver・Owner判断はNOT RUN／UNQUALIFIED。本計画Markdown以外は変更せず、G1/G2未受理、Draft/main未merge。新HEAD Healthは結果を確認するまで未確定。

### 16.6 Receiptの再照会と実データのPost-imageは別証拠（11/7 main追補）

[merged #3118](https://github.com/hat47x/tei-platform/pull/3118)により`commandadapter.Executor.Effect`へ**呼出側が渡した宣言効果**を、Committed/Replayedの`Result.Effect`として返せるようになった。報告漏れ`ReportedEffectMissing`は減らせるが、これは実更新先の列挙`Applied`・実DBのpost-image・受理済みOwner Targetの観測ではない。

[merged #3124](https://github.com/hat47x/tei-platform/pull/3124)は`command.Invoker`へ`PersistenceState`（`not-applicable`／`not-observed`／`observed`）と`DurableStore`、`PersistenceObserver`を導入した。`DurableStore`はCallerの申告で、`PersistenceObserver.Observe`がnilを返すと`observed`へ進む。`not-observed`は「Commit失敗」ではなく、未構成・読取失敗などの**確認不足**。

さらに[merged #3127](https://github.com/hat47x/tei-platform/pull/3127)の実Go `commandadapter.Executor.Observe`（`Lookup=true`）は、`Runner.Lookup`が`StatusReplayed`を返すとnilを返す。このLookupは**同一OperationID/RequestHash/GrantでReceiptを読む**が、`objects`や受注の**現在値を読まない**。それでも`Invoker.observePersistence`はnil→`PersistenceObserved`とし、CLI/Studioの`persisted`は`observed`でtrueに変換される。実試験`TestExecutorObservesACommittedReceiptByLookup`もこのReceipt-only fake Providerを使用する。**「Receiptの存在をobservedと表示する」実装は成立したが、業務レコードが期待どおり存在・更新されているという意味での独立Read Oを満たさない**。現状は表示上の`observed/persisted:true`をRMTのO資格として採用しない。

| 既存F-Post/NoReportと同じOperationで追加する対照（新Fixture正本なし） | 独立期待結果・確認対象 | 誤った結論を拒否 |
| --- | --- | --- |
| **K-Lookup-01 / Receipt正例** | 実Provider Commit、同OperationID・RequestHash・現GrantのReceipt再照会。`StatusReplayed`なら**Kの照会成功**のみを記録 | これを`order.status`／`order.lines`の実Read O、Business Completion Bへ自動変換しない |
| **K-Lookup-02 / Receiptあり・値不一致** | Commit後、別許可Operationが同じTargetを更新、又はレコードが削除された反例。元ReceiptのLookupが成功しても、別Connectionの現在Post-imageが元Expectedと違えば**K confirmed / O mismatch** | `persistence:observed`や`persisted:true`だけで現在値・業務成立・再実行不要を保証しない |
| **K-Lookup-03 / Recorder/Observer不可** | 実Commit済みでLookup不能（権限失効/通信障害/構成無し）なら**Kが持つ確定度を保持し、O unverified**。観測失敗をCommit取消・効果0としない | `not-observed`を`not-persisted`や新OperationIDでの再送許可へ変換しない |
| **K-Lookup-04 / Replayと宣言自己報告** | 同ID/同RequestのReplayed Receiptと、`Executor.Effect`に渡された宣言効果（自認）を分け、Providerの独立Post-image/Target/Scope/Schemaを再照合 | Replayed=新規Commit、`Result.Effect`の存在=Providerが物理変更を検証済み、としない |
| **K-Lookup-05 / 複合Effect** | 同じ業務Operationが主DB行と関連Data/外部Serviceを変更する場合、Receipt観測可能な範囲を明示し、各EffectのConfirmed/Failed/Unknownと独立Readを分ける | 単一Receiptの`observed`をすべての派生/外部作用へ適用しない |

**製品契約の補正候補（Owner判断まで未採用）**：#2998/#1174では、「Receipt確認」専用の観測結果と、「物理Record/Targetのpost-imageを独立読取で一致確認」の結果を別の証拠として扱う。既存`PersistenceObserved`／`persisted`の意味を**Receipt-onlyへ縮小する**ならConsumer文書・CLI JSON互換性をレビューする。従来どおり**宣言効果の実データ観測**の意味に保つなら、Receipt-only Observerではその資格へ昇格させず、別ConnectionのRead/Scope/Revision/target一致を確認できるObserverへ渡す。ここで新共通Enumや二重Receipt Storeは設けない。

[merged #3125](https://github.com/hat47x/tei-platform/pull/3125)のX-TEA標準業務（会計月・暦・税・採番等）の固定版静的読解は、原典Rの意味候補としてのみ受け取る。RMTの期間判定に関係する`FIRST_MONTH`・`ALLOWED_FISCAL_MONTH`とTEIの`sales.month`を同じ意味と推定せず、実Rule・適用時点・Owner受理で別途対応付ける。税の符号別丸めや採番桁あふれの旧実装特性をTEIの安全な既定値へ無断移植せず、原典Parityと改善設計を区別する。これらは**P0-R1実Readの前提条件にしない**。

**資格と次一手**：#3127 exact HEAD `c15bc4e074c5d7a86b190277c2f518df460a5876`の[Repository Health](https://github.com/hat47x/tei-platform/actions/runs/42316661948)はcompleted/success。実PostgreSQL上のReceipt Lookup＋受注Post-imageを**同一RMT Attemptで**検証した証拠ではない。`BuildBrowse→実Provider HR-01/02`の実Readを優先し、書込み後段の受理では**Receipt K と別実Read O**を最低限独立に残す。文書のみの追補で、新Go/PG-RMT/Windows原典Driver/Owner意味受理はNOT RUN、G1/G2 UNQUALIFIED。

### 16.7 実PostgreSQLで資源現在値の別読取が成立：KとOの境界を更新（11/7）

`16.6で「資源Post-image Reader未接続」と記した状態は**現在のmainでは更新が必要**。[merged #3155](https://github.com/hat47x/tei-platform/pull/3155)により、Receipt-only照会と資源値照会は別`PersistenceMethod`になった。現[`internal/command/persistence.go`](https://github.com/hat47x/tei-platform/blob/main/internal/command/persistence.go)は次の4状態：`not-applicable`、`not-observed`、`receipt-observed`、`resource-observed`。**`persisted:true`は`resource-observed`だけ**（[merged #3157](https://github.com/hat47x/tei-platform/pull/3157)の`docs/status.md`にも反映済み）。

現[`commandadapter.Executor.ObservePersistence`](https://github.com/hat47x/tei-platform/blob/main/internal/security/effectcommit/commandadapter/executor.go)は、`ResourceRead=true`なら`Runner.ReadResource`の返却Payloadと当該呼出側が固定した実Commit Payloadの**byte一致**を確認して`resource-observed`とする。`ResourceRead=false`かつ`Lookup=true`なら`receipt-observed`（同Operationの記録の照会のみ）。どちらも失敗・未設定なら`not-observed`（効果0ではない）。[`postgres.Provider.ReadResource`](https://github.com/hat47x/tei-platform/blob/main/internal/security/effectcommit/postgres/provider.go)は書込後に**別pool接続**で`objects`の`version,value`を読む。

実証は[PR #3155 exact-head `19f00c7`の実PG Workflow #42328707393](https://github.com/hat47x/tei-platform/actions/runs/42328707393)（completed/success）の実job logまで確認し、`TestPostgresCommandObservesACommittedReceiptByLookup`と`TestPostgresCommandObservesTheStoredResource`が**実行されPASS、SKIPなし**であることを確認した。後者はCommit→別接続でのPayload一致の正例と、同じ資源値を`changed`に変更した後の観測失敗を検証する。これは現実のPostgreSQLでの**局所的なEffect Commit資源観測**が成立したという進歩であり、単なる設計案やモックのPASSではない。

しかし**受注RMTの業務Post-image Oは依然未資格**。現実装の以下の条件は、この局所成功からは導けない。

| 既存K-Lookup受入票への差分 | 現Sourceで確認したこと | RMT側の独立Expected／最小の追加対照 |
| --- | --- | --- |
| **K/O-Resource-01（正対照）** | `ResourceRead=true`ではResource現在Payload byte一致を別pool Readで観測済み。Receipt-onlyは`receipt-observed`であり`persisted:false` | 実RMT受注ContextとOwner受理されたTarget/SQL/Scopeに結び付けた上で`resource-observed`を業務Oへ接続。`effectcommit.objects`の単一Resource値を`sales.order`／明細Lの取得済み完全集合と読み替えない |
| **K/O-Resource-02（ABA・Version）** | PostgreSQL Readerは`version,value`を返すが、`Executor.ObservePersistence`が比較するのは**`state.Payload`だけ**で、`state.Version`を元ReceiptのVersionと比較していない | 書込A→別OperationでB→別OperationでA、現在ValueがAへ戻ってVersionが進んだ反例では、Payload一致で現コードは`resource-observed`へ到達し得る。**同一版のEffect Post-imageと主張しない**。Version/Receiptの帰属は#2564/#1174が判断 |
| **K/O-Resource-03（読取Scope/権限）** | `postgres.Provider.ReadResource`の`WHERE`は`application=$1 AND resource=$2`で、メソッドコメントにも**読取時のAuthority再検査なし**とある。`Scope`の`Subject/Action/Capability`は空でないことを検査するがSQL絞込には含まれない | 読取主体/Grant epoch/業務Tenantと開示可否、Resource ID→実Record/Business keyの解決を別Host/Provider根拠で検査。内部Observer結果の生Payloadを無許可Consumerへ公開しない |
| **K/O-Resource-04（構成）** | `ResourceRead`はOptional bool、`DurableStore`はCaller申告、`PersistenceObserver`も明示構成が必要 | Productの当該入口で設定されていなければ`not-observed`（正常処理と観測未実施を分離）。`resource-observed`を全Command/全Providerで実施済みとしない |
| **K/O-Resource-05（Replayと複合Effect）** | `ResourceRead`は現在値とCaller Payloadを比較する。旧Receipt Replay、複合更新、外部API/明細全件は自動取得しない | 同じOperationID/RequestHashのK、Owner対象集合Lの実取得完了、当該EffectのVersion/Reader、外部作用ごとの結果を独立欄に保持。旧Receiptが存在するだけでB/全Effectを`holds`にしない |

上記K/O-Resource-02/03は**現Sourceを読んだ設計上の反例案であり、その新しい実DB故障注入を実行したという主張ではない**。新Persistence状態・Validator・権限エンジンは増やさず、既存#2998（公開契約）、#1174（Commit/Grant/Target）、#2564（CAS/ABA）、#1163（Evidence）の独立Expectedへ回す。

### 16.8 Pattern／Operation Surfaceの同一性はまだ未採用：P0-R1の別入口へ誤束縛しない

[merged #3145](https://github.com/hat47x/tei-platform/pull/3145)・[merged #3167](https://github.com/hat47x/tei-platform/pull/3167)は実TEI CLIの固定Worldで、`tei operations`の導出Operationと宣言Function Patternが**まだ一対一に結ばれない**ことを観測した。`reference/workspace`へ`order_entry.preview_order_quote`候補Patternを追加すると、宣言Patternは5→6件、World fingerprintは変わるが、導出Operationは8件のまま／`summary.declared=0`。`Binding`は効果を宣言しないread-only Commandでは空となる。前掲受注BrowseとOperation SurfaceのProduct実行資格は**異なる入口**であり、`BuildBrowse`のQuery選択が成立しても`tei operations`が同じPatternからOperatorを選択したことにはならない。

[PR #3172](https://github.com/hat47x/tei-platform/pull/3172)はW3対応表の設計候補（現時点**未採用・未実装**）。Data ID一致だけで同一Dataの`browse`と`read`を選別できず、Capability IDが効果なしread-only Commandでは消え、Relation targetとSemantic targetの変換も別規則が必要。**名前類似・先頭一致・first-winnerで未宣言のFunction/Effectを勝手に結ばない**。#1127/#1129の既存W3 Ownerが規則/版/優先・衝突を受理するまで、RMT HR-01/02は`BuildBrowse→Query→実Reader`という既存表示入口に限定し、Operation Surface経由で実業務行動が可能と報告しない。新Operation Engineや標準表現型の中核列挙を追加しない。

[merged #3171](https://github.com/hat47x/tei-platform/pull/3171)では`ApplicationView.periods`が`calendar`の宣言IDを投影できるようになり、受容例`sales.month calendar sales.fiscal`をCLI表示で確認済み。ただし**表示上のID保持であって暦Ruleの解決/期間強制や旧DriverとのMeaning同一性ではない**。原典の`FIRST_MONTH`・`ALLOWED_FISCAL_MONTH`を推測で置換せず、#2144/#1125の元々の期間対照へ引き渡す。P0-R1実Readの新必須依存にしない。

**2026-11-07の資格と引継ぎ**：旧Draft Head `1b3f5a2661ed4e51b23276e8e6bd7d9f30d3b9ff`の[Repo Health #42317691986](https://github.com/hat47x/tei-platform/actions/runs/42317691986)はcompleted/success。今回本節は既存RMT受入計画Markdownだけ変更し、Go/Fixture/Runner/Owner受理を触らない。**現Mainの実PG Resource Receipt/Value一致試験成功（#3155）はE2の一部成立、RMT Function Read HR-01/02とBusiness B、Windows旧Driver、正式G1/G2は依然NOT RUN/UNQUALIFIED**。優先順は従来どおり実Read→明細完全性→同Rule二入口保存前拒否→Receipt K/資源O→業務B。新HeadのCIは再確認前にPASSとは記録しない。

## 17. 11/7再資格：Pattern/Operation実Crosswalk、Read Sliceの完全性、Function内Representationの非昇格

本章は第14〜16章の一Function・一Query・一実Providerを優先するRMT P0-R1を維持し、並行レーンの**実現済みSource**と**未統合検証・設計候補**を同じ資格にしない。X-TEA合成受容例E、旧原典Driver忠実移植R、実データK/O、業務Owner Bは引き続き別Attempt・別Claim。新しい第二Query/Operation Engine、Scenario正本、UI State Engine、Host/Authority、Fixtureは導入しない。

### 17.1 第16.8節の現在地を更新：参照Crosswalkは動いたがW3/Product Actionは未統合

[main merged #3195](https://github.com/hat47x/tei-platform/pull/3195)は`tools/reference-operation-pattern-crosswalk`を実装済み。既存`operations.DeriveOperationSurface`を呼んでData ID／Capability ID／Relation targetだけを突き合わせ、結果をlinks/unlinked/conflicts/ambiguousへ出す**読取専用の参考ツール**であり、製品の`tei operations`で宣言Patternを優先選択するW3は**未実装・未採用**。実Workspaceの操作8件・宣言Pattern5件でlink 4／unlinked 2／conflict 0。追加候補Patternを載せると8件・6件でlink 4／unlinked 3となる。4件の一致には`recent_orders→list.sales.order`、`order_record→read.sales.order`、`order_lines→list/read.sales.order_line`が含まれる。これは**Readのための設計候補間Identity対照**であって、Query/SQLがProviderへ渡った実行記録ではない。

[main merged #3204](https://github.com/hat47x/tei-platform/pull/3204)はその同一規則を5つの宣言追加で反証：effectを持つcommandは対応Capabilityにリンク可能、Relation先DataのPersistence追加でreferenceがreadにリンク可能、同Dataを指す2 browseは**衝突・双方未採用**、効果の宣言だけでPersistenceが無ければmissing-input。**名前類似・暗黙first-winner・非宣言Effectの合成を禁止する**。旧第16.8節の「手掛かりだけで結合規則が動かない」という一般化は、**参照Crosswalkに限っては訂正**する。ただしW3の採否/版管理・`Origin=declared`の製品反映・実行可能Action Offer/現在Grantを自動成立にしない（#1127/#1129/#951）。

P0-R1の実Reader呼出しは`BuildBrowse(Query)→selected SQL+actual args→physical Provider→Rows/Scope/Consumer`をそのまま先行。**Crosswalk link 4件をHR-01のPASSに足さず、W3の採否をHR-01の新しい前提にも置かない**。

### 17.2 Completionの「Caller申告」をRead Sliceの独立証拠へ縫い付ける

[Draft #3175](https://github.com/hat47x/tei-platform/pull/3175)は現`semantic.EvaluateCompletion`と`AuthorizeBrowseRead`の実Go APIで、`ReadPlan($top=1,$select=orderId)`と**合成Rowset**を切り離したCDE-Q05〜07の限定Characterization。固定HEAD`defbf7297fb98d444879bb1766c6af95d2a17707`の[Repository Health #42334653630](https://github.com/hat47x/tei-platform/actions/runs/42334653630)はcompleted/successだが、**Draft未merge・実PostgreSQL RowsetはNOT RUN**。mainの一般的な完成保証ではない。

| 既存HR-03〜05/PA-11への受渡し | 独立Expectedの要点 | `EvaluateCompletion`だけでは確認しない事実 |
| --- | --- | --- |
| **Q05：件数一致／識別子欠落** | Owner対象`L={A/3,A/9}`に対し観測`R=[A/3,A/3]`なら`count(R)=count(L)`でも`R≠L`。Callerが`CoverageComplete`を誤って渡した局所`holds`は採用しない | 重複除去、Row Identity、Tenant/Scope別対象集合の被覆 |
| **Q06：Pageと全称の差** | `LIMIT 1`の`R=[A/9]`を全Lへ変換せず、Coverage不確定/Partialなら`unverified`。最初のPage/0件は空集合証拠ではない | 実Read PlanのPagination・次Page・全件存在、否定依存 |
| **Q07：非出力Predicate** | Consumer`$select=orderId`だけでは`status=READY`の評価結果は導けない。`Holds=true`の自己申告は不適格 | ただし**認可済SQL WHERE／Operatorがstatusを実評価し、Query/Operator版、入力、Scope、結果の証拠を記録したならConsumerへのstatus列公開は不要**。非表示≠未評価 |

P0-R1の**同一Attempt Read Slice最小証拠**は`Actual Source bytes+Revision / selected Query ID+SQL bytes / datasource alias+physical Provider / status bind actual / Read Plan(SQL/filter/limit/selection) / acquired Rows(業務Key・多重度・Field grain) / Reader Scope(Tenant・Grant・Read epoch) / Coverage basis(LとRの識別集合・Page/negative dependency) / Predicate evaluation origin / Consumer disclosure`。Roleと対象が異なるReadのPASSを結合してCompletionの`CoverageComplete`を自動入力しない。これらは`#2506 Context/Result Slice`、`#2514 Independent Expected`、`#1163 Evidence Packet`、`#1180 Scenario`を**再利用**。Readが動くようになってもこの欄が欠ければ、Business Bを自動合格にしない。

### 17.3 Data topology・Representation topology・Function Binding topologyを混同しない

[Draft #3179](https://github.com/hat47x/tei-platform/pull/3179)はMaster Planの全体ロードマップ、[Draft #3225](https://github.com/hat47x/tei-platform/pull/3225)はData/Function/Business詳細引継ぎを担当し、[Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)が表現型と媒体軸、[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)が複数表現間の選択同期を担当。**これらのDraftは現行Productの受理済み構造ではない**。本RMT計画は以下の受入境界だけ共有する。

- **Data topology**：1:1／1:N／Graph等の実Data/Relation/粒度とOwner意味。Source行・JOIN Fanout・Parent/ChildのIdentityを保つ。
- **Presentation type**：Panel/Table/Tree/Chart/Media等の機能非依存な種類。種類の存在はUI Widget固定列挙をCoreに増やす根拠ではない。
- **Function-bound instance / presentation topology**：具体的な表示Slot・Field/Axis/SelectionがどのFunction Context Instance・Result Sliceに束縛され、粒度/単位/欠損/媒体時刻のどこが失われるかを明示する。Headless FunctionならSlot=0も成立する。Multi-viewのSelection epochは**業務ID・実更新対象・Source Mutation・現在Grantを自動で決めない**。

最初の受注Browseは、選択されたQueryの**実Read結果**をFunction Contextへ束縛できた時点で、Table表示への投影検収に進める。Map/Audio/Media／別Functionの高度表現を**HR-01のブロッカーにしない**。画面が正しい／資料として同じ意味に見えることを、実ProviderのRow/Scope/Completenessや別ConsumerのDisclosureへ転用しない。

### 17.4 残る実行・自律・永続化の条件

[main merged #3197](https://github.com/hat47x/tei-platform/pull/3197)が型コメントでも明確にした通り、`command.Executor`に`Invoker.UnitOfWork`へのTx参加の契約はない。外側RollbackやLifecycleの`unknown`は実Providerの別Tx Commitを巻き戻さない。PR #3155の`resource-observed`は実PGの単一Resource Payload照合として成立したが、宣言外の別Data更新、RMT Header/Lines全件、Version ABA、Current Read Grantの独立証拠にはしない（第16.7節）。

[merged #3220](https://github.com/hat47x/tei-platform/pull/3220)の観測ではPlugin CapabilityKindとOperation Kindの語彙検査は別系統。前者は現行閉じた列挙、後者は規則内の生成語彙で未知種別検査なし。これを「Plugin追加で標準Operation Kindも自動拡張された」と解釈しない。[PR #3231](https://github.com/hat47x/tei-platform/pull/3231)の読み取り専用調査は`unattended`が現`list/read`から導かれ、明示Service主体や`human-gated`を評価していない範囲を示したが、**読取専用だから認可不要・外部Consumerへ公開可能とはしない**。自律区分／Host実Grant／Disclosureは別資格のまま。

**次の単一試行と資格**：現mainのCrosswalk参照・PG resource observer・Go独立負例は進展として採用し、いずれも**RMT実ReaderのHR-01/02代用にしない**。#2586がまず受注Browseのphysical Provider/実SQL+status/Rowsへ接続、#2506/#2514がQ05〜07に対応するRead Sliceと独立Expected、#1163が単一Attempt Packetを持ち、その後に明細Coverage→同Rule二入口/保存前拒否→実Receipt K/Resource O/Business Bを積む。旧Driver忠実移植RとOwner Meaning・正式G1/G2は未受理。`#3175`はDraftであり、mainへの新Go Test統合と実PG Product Readを実行したとは主張しない。

**文書変更の範囲**：本Draft #2783の既存RMT Markdown一ファイルのみ。Go/Fixture/Scenario/Provider/Studio/Windows原典Driverは今回変更・実行しない。変更前HEAD`3505ac2bb9e8914b0296919ea125fe1082b51cf2`の[Repository Health #42334571590](https://github.com/hat47x/tei-platform/actions/runs/42334571590)はcompleted/success。変更後HEADは別にCIを確認するまでUNVERIFIED。Draft/main未merge、G1/G2 UNQUALIFIED。

## 18. 11/7夜の再資格化：同名Queryの別SQL出典／限定PG Read Slice／AIテスト能力

第14〜17章のE（X-TEA合成受容例）／R（旧Driver忠実移植）／K（実Provider Effect）／O（独立観測）／B（Owner業務意味）の区別を保ちつつ、他チャットのDesign Context Query、標準構造・表現型、テスト判別力、Data Source Workbench、三視点CRUD/図面の議論を**最初の実Product Readerとその検収**へ集約する。新Query Engine・Provider・Authority・Scenario正本・Fixtureを増やさない。G1/G2の最優先P0-R1は維持する。

### 18.1 同じrecent_orders名の3経路は同じSQL／Targetではない

現mainの実ファイルと、固定HEADの実PG試験を再照合した。**同じ受注・Query IDに見えても、別のWorld／SQL byte／入力／対象表／取得粒度である**。

| 経路 | Sourceで確認できたこと | 現時点の成立範囲 |
| --- | --- | --- |
| **E：合成X-TEA例** | examples/xtea-lowcode の Queryは status を input に宣言し、SQLは sales.order から order_id/customer_id を status=:status で抽出。Browse PatternはPage size 20（第14章） | Go の BuildBrowse はPattern→Query/Result属性を解決するが、CLIの当該Browse分岐は表示定義を返すだけ。実Provider Rowsetを取得しない。**HR-01/02の対象はこのEのstatus正負例** |
| **W：reference/workspace** | reference/workspace/applications/order-entry/queries/recent_orders.tei はResult 5列、datasource sales、SQL file recent_orders.sql。対応SQLは public.orders から ordered_at >= :from で読み、ordered_at DESCで順序付ける。現Query定義に input from 宣言は見当たらない | SQL側の **:from の由来・型・実Binding**をProductが解決できるかは未検証。Eの :status を自動代入せず、W Routeの独立検収とする |
| **C：Web Referenceの限定実PG** | [Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)は別の public.tei_web_scoped_orders_reference を使う。Tenant Aの実行済みRowset TOP2=[[3],[9]]、TOP1=[[3]]、非出力 ordered_at のWHERE評価で from=2026-10-02 → [[9]]を独立Expectedへ照合 | PR exact HEAD d3b67d5809478fad828052ce01bdafe576b2f934 の[専用PG CI #42351796330](https://github.com/hat47x/tei-platform/actions/runs/42351796330)と[Health #42351796383](https://github.com/hat47x/tei-platform/actions/runs/42351796383)はcompleted/success、専用PG試験SKIPなし。ただし**Product E/WのFunction選択からこのReaderを呼んだ実証ではない**。PRはDraft・未main統合 |

**最初の停止辺はEのQuery選択後**。現mainの cmd/tei/cli_view.go のBrowse分岐は BuildBrowse→BrowseViewをResultへ入れ、SQL・実Providerを呼ばない。別の製品Caller全体を未調査のまま不存在とは断定しないが、Eの **selected Query→SQL実原byte→alias→physical Provider Instance→実SQL+status bind→Rows/Scope→Function Data Context** は少なくともその分岐では未接続。Cの成功をEの実Read成功に合成しない。

### 18.2 #2586が次に結線する一本と、独立Expectedの最小セット

**実行先行：E Routeで最初のHeader SQL Read一往復**。既存Query・Provider・Context契約を用い、第二SQL生成器／Runnerは作らない。実行時にSource/World版、選ばれたQueryとSQLの生byte/版、実Inputの値・型・placeholder対応、logical datasource aliasとphysical Instance、実SQL、返却列/Row ID、Tenant/Read epoch、結果のConsumer受領を一つのAttempt証拠に束縛する（#1163 W1〜W5）。この時点ではHeader二列の有限Reader証拠に限り、明細全件/業務完了B/二入口Parityへ拡張しない。

| 既存HR-01〜05の追加判別 | 独立Expectation／停止条件 |
| --- | --- |
| **E01 正例** | status OPEN/READYを個別Bindし、実Providerの実返却行と事前固定Expectedを照合。SQL byte/Target/Result Columnが異なるC/Wルートとのすり替えを検出 |
| **E02 入力負例** | status 欠損、NULL、型誤り、誤値、未束縛alias、SQL版違いを区別。0件を即座に不存在/Business failへ昇格しない |
| **W01 SQL :fromの起源** | W RouteのSQLには :from があるがQuery定義に入力宣言が見当たらない。実Resolver/CallerからのParameter供給根拠がなければ unbound-query-input 相当で停止。Eの statusとは別 |
| **R1-Drift** | [main #3254](https://github.com/hat47x/tei-platform/pull/3254)ではWorld/Model Revisionが変わってもOperation basisが不変な例を観測。Query ID・basis一致を実SQL Source同一性/実行時意味一致としない。Source、実選択、実Executable、SQL+args、Readerの因果を別照合 |
| **L-Whole** | Header Rowsetのみでは order.lines のOwner対象集合L、Page後続、集合の多重度・権限・時点を確定しない。Lの実Read後にのみCompletionへ接続 |

### 18.3 Q05〜09の実証済み範囲を訂正し、開示の漏れを別検収にする

旧§17.2のQ05〜07は、[新Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)に移植された。限定Referenceでは**実PG**が full={3,9} / top1={3} を返し、テスト上はCompleteならholds、Partialならunverified、top1へ誤ってComplete自己申告すれば局所holdsとなり得ることを実証した。これは「製品FunctionのResult Slice完全性が自動証明される」ことではない。fullとtop1は**異なるRead Attempt・Query Plan**で、同じCI/DBでも同一Snapshotの保証はない。

- **Q05**：同件数でもR=[3,3]はL={3,9}を覆わない。件数≠Entity ID集合、JOIN fanout/Relation occurrenceを区別。
- **Q06**：TOP1を全件として扱わず、Pagination/Source Scope/Snapshotと対象集合が確立するまではunverified。
- **Q07**：status=READYの根拠がないままCallerのHolds=trueを信じない。非出力列をConsumerへ露出しなくても、SQL/Operatorが実評価した出典と結果があれば適格なことはある。
- **Q08**：非選択列にSQL NULL AS statusを投影したRaw slot=nilは、DB格納statusがNULLという意味ではない。not-selected/not-fetched/redacted/permission-denied/実NULLを区別し、ReadPlan.Selectedと実SQL列alias・Cell provenanceを結ぶ。
- **Q09**：別の非出力 ordered_at を実SQL WHEREで評価し、from=2026-10-02でID 9だけを返す正例は専用PGでPASS。ただし同じ[9]がTOP1等で出る場合と区別するには実SQL+args+Filter+Limitが必要。これはQ07のstatus条件を実評価した証拠ではない。

**追加の停止線：秘匿Predicateで絞り込んだ行の存在・件数がConsumerに見えてよいとは限らない**。表示列を落とすだけでは非干渉を証明できない。非開示条件だけが異なる二世界で、公開Body、件数、空集合、診断、Error、Timing、Tool結果/Session Logまでの実差を検証する責務は#2516/#2514/#1180へ渡す。実ProviderでPredicateが成立した事実とConsumerへの開示資格を別にする。

### 18.4 実テスト能力T0の前進を採用し、T1とG1/G2を混同しない

[main #3266](https://github.com/hat47x/tei-platform/pull/3266)と[main #3268](https://github.com/hat47x/tei-platform/pull/3268)は、**main exact SHA 69a2996187852a036057b7155d6e847527b67346**の[Push CI #42359610226](https://github.com/hat47x/tei-platform/actions/runs/42359610226)で実走済み。実TEI CLI 9 read-only、Skill静的14、pinned Driver Scenario 6 Step、実CLI負例4・独立義務負例2をPASS。局所cmd/teiのGo statement C0相当10.1%は観測されたが**C1未計測**。実DB Product Function、Owner Business、全Requirement/NFRは対象外。このT0を「未実施」へ戻さない。

[Draft #3273](https://github.com/hat47x/tei-platform/pull/3273)のT1は、#3262の**必要Design Contextを完全性・Source版付きで取得した後**、未カバーRule（例：line_number_positiveの0）からAI/人間が**新たな**Scenario負例を選び、#1180/ #2514の独立Holdout Expected/Mutantと実CLIで判別する設計。T1のAI新規生成・C1・未取得Function/Business/Rule/Procedure閉包は未達。RMTではE01/E02に必要なSource・status/Scope/Context/Non-output依存をRequired/Supplied/Observed/Undischargedへ投影するだけで、第二Test Engine/新Scenario正本を作らない。

[merged #3276](https://github.com/hat47x/tei-platform/pull/3276)のRQ-035/Data Source Workbench設計と[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)の三視点CRUD・図面は、共通のData/Function/Business Source、declared Read/CUDとactual Reader/Effect、Result Slice/Consumerの区別を補助する。WorkbenchやGraph/Studio/SHOの完成をE01/02のHard Gateにしない。実Read後に同一Function ContextをForm/Tableへ投影するConsumer検収に回す。

**既存Owner・受入順**：#2586 Product Reader、#2611/#2622 physical Provider、#2506 Function Data Context、#2514/#1180 Scenarioと独立Expected、#1163 Evidence、#2516 Disclosure、#3262 Design Query、#3273 T1、#1119 Gate。最初は**E Route Product実SQL＋status HR-01/02**、次にRow/Scope/Field grain、Page/非出力依存、明細完全集合L、同Rule二入口、実K/O、Business B。今回の変更は**既存RMT Markdown一ファイルのみ**。新Go/本RMT実PG/旧Windows Driver/Host/SHO/正式Owner受理はNOT RUN。C Route実PGとmain T0の過去実測は各々の限定成果とし、更新後HEADのHealthを別に確認する。

## 19. 11/8横断再資格：実行可能なテスト・Workspace・Tool EffectをE Routeへ接続

第18章のE（X-TEA合成受注例のstatus Query）／W（別Workspaceのfrom Query）／C（Web Referenceの隔離PostgreSQL）を異なる実行経路として保持する。Eの最初の未接続は「Function/Patternが選択したQuery → SQL実原byteとstatus引数 → physical Provider → Header実Rowset → Function Data Context/Consumer」である。本章は並行レーンの成熟成果を、その代わりではなく実証の前段・後段へ限定して組み込む。

### 19.1 T1-Aは実走、T1-Bは未成立

[main #3284](https://github.com/hat47x/tei-platform/pull/3284)のDesign Context Gateは実TEI CLIの9読取応答と、欠落・版競合・Root不完全などの合成負例を実測した。ただし正常時もcontextはpartial、qualifiedTestGeneration=false。Sourceの読込・列挙から必須Function/Rule/Provider/Grantの完全な閉包を推定しない。

[main #3304](https://github.com/hat47x/tei-platform/pull/3304)は、既存計画#3273で既に提案した「明細行番号は正数」というRuleに0を与える一時Scenario第7Stepを、生成済みPinned TEI CLIで実実行した。旧固定HEAD 02d4c624の[専用CI #42378033965](https://github.com/hat47x/tei-platform/actions/runs/42378033965)は成功し、行番号1の正対照・0の不成立、偽Expected／偽Fixture／古いPinの3負例を検出した。**T1-A（既知Caseの実行・判別）は成立、T1-B（AIが未見Ruleから独立した新Caseを盲検発見）は未成立**。正式Scenario採用、全要件/C1 Coverage、Owner受理ではない。#3304最終HEADの[全体Health #42378254383](https://github.com/hat47x/tei-platform/actions/runs/42378254383)はfailureであり、専用CI成功と混同しない。

RMTではEのstatus入力定義とSQLのstatusプレースホルダをRequiredに固定し、E01のOPEN/READY実Bind対照、E02の欠損/NULL/誤Bind・異SQL Sourceの負例を候補化する。**候補の合成Scenario PASSは実Product SQL/Provider HR-01/02の成功ではない**。T1-Bの検証は#3262 Design Context、#1180 Scenario、#2514独立Expected／Holdout／Mutantに委ねる。

### 19.2 取得・選択・実行・受領の四境界を独立にする

現mainのinternal/operations/inspect.goにはWorkspace識別、worldFingerprint、sourceIndexFingerprintの別欄がある。[PR #3311](https://github.com/hat47x/tei-platform/pull/3311)のManifest版/hashと名前付き出力広告は追加候補であり未統合。Workspace名・Manifest Hash・Query IDが同じでもSQL実原byteや実Providerの取得を証明しない。[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)はviewをread-onlyと広告してもexecute指定でoperation-log等へ実書込できる負例と、広告を限定する修正案。外部Harnessがexecute付きviewを無条件read-onlyと扱わず、Source非変更・名前付きファイル書込・Command/DB Effectを別に検査する。

| Eの同一Attemptの欄 | 必須となる実観測 | 禁止する推論 |
| --- | --- | --- |
| W：WorkspaceとSource | Root/Manifest、World/Index指紋、Loader実取得Source | 同名・宣言あり＝全Required読込済み |
| Q：実Query | 選択SQL原byte/版、Dialect、status実型・値・束縛 | 同Query ID＝同SQL・同実行 |
| P：実Provider | sales→sales_core→物理Instance、実SQL/引数、Grant/Target/Tenant/時点 | BuildBrowse投影成功＝実DB読取成功 |
| R：ResultとConsumer | Row Identity/粒度/多重度、Page、Predicate由来、完全なReply受領 | TOP1・0行＝全母集団、返却＝受領、非出力＝開示安全 |

Required→Loaded/Parsed、Selected→Provider Execute、Returned→Consumer Received、Result Slice→Completion Claimを四つの独立した境界として検収する。不明・partialなら後段をunqualifiedに保つ。新たなEvidence Registry/APIを作らず、#1163/#2506/#2586/#2516へ受け渡す。

### 19.3 Hook、図面、CIの進展と資格上限

[PR #3306](https://github.com/hat47x/tei-platform/pull/3306)はHook usesに対応するCommand Patternの不在を検証拒否する未統合の改善候補。複数Command曖昧性、Phase、実Runner/Provider接続は未検収。[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)・[Draft #3303](https://github.com/hat47x/tei-platform/pull/3303)の三視点Diagram/CRUD P0-Mと実Read P0-R、同一Identity/Scopeでの接合P0-Cを別Gateとする。図の宣言Rセル、Canvas選択、合成Fixtureを製品実Reader/Host Grantの証拠にしない。P0-MをEの実Readerに先立つ必須条件へ追加しない。

直前計画HEAD 2f3c9494の[Health #42370557015](https://github.com/hat47x/tei-platform/actions/runs/42370557015)はfailure。実Jobログで原因はRQ-035のPAS/Gap未登録と設計索引4件不足であり、gofmt・go vetと他のGo Package試験はPASS。**現mainではこれらの登録済み状態をGitHubで直接確認**した。旧HEADのFAILは履歴として保持し、索引は既存修復Ownerと重複変更せず、新HEAD CIで別判定する。

次一手は#2586のE Route実Provider Header Read HR-01/02。その後#2506/#2514のPage・明細完全集合・非出力Predicate、#2516開示、#1125/#1174の同Rule二入口／Receipt K・資源O／業務Bへ進む。#3238のC専用実PGと#3304のT1-A成功は局所成果として認めるが、E実Reader、旧Windows Driver、Owner Meaning受理、G1/G2合格へは合算しない。今回はMarkdown計画だけの変更で、新Go・実E Provider・旧Driver・Owner正式試験はNOT RUN。

## 20. 11/8再資格：Design Contextの閉包反例、実CLI効果、Commit後Unknownと原典暦を受入順に統合

第18〜19章で固定したE／W／Cの異SQL・異Target・異Read Attemptの区別、及び最初のE Route（examples/xtea-lowcode の order_entry.recent_orders、SQL status=:status）Product Read HR-01/02を変更しない。本章で取り込む新成果は、いずれも**局所の実Go/CI、または未統合Draftの提案**であり、製品のFunction→実SQL/Provider Rowset、一つのOwner業務効果と同じ資格にしない。新Scenario正本、Source Index／Plugin Resolver／Transaction Coordinator／Hostを追加しない。

### 20.1 TT-56：Fingerprint一致・正常なJSONでも、必要なRuleが欠落し得る

[Draft #3326](https://github.com/hat47x/tei-platform/pull/3326)は、既存CLI9件のContext Gateから**既存source-indexの全Indexモードを追加した10応答**へ広げ、事前に固定した公開合成Validation 3件のRequiredとCatalog・SourceIndexのDefinition Role・Source Path・Loader実Readを照合した。初回[専用CI #42385502023](https://github.com/hat47x/tei-platform/actions/runs/42385502023)は失敗したが、それは**subjectモードのSource Indexを全件Indexと誤認した試験用Consumerの不備**であり、「必要Ruleが実Sourceから消えた」という証拠ではない。修正後の固定HEAD c2d2da10cc93022697204e66e83c30248b3810a0 に対する[専用dogfood #42385726717](https://github.com/hat47x/tei-platform/actions/runs/42385726717)と[Repository Health #42385726733](https://github.com/hat47x/tei-platform/actions/runs/42385726733)はcompleted/success。**ただしDraft未統合**であり、現mainのDesign Contextが全Rule closureを保証するという意味ではない。並行[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324)も近い問題を扱うので、**両案を独立した二つの正本として取り込まない**。

E Routeの検証に使う最小Requiredを、宣言Status Input／SQL Sourceだけでなく、そのQueryのPredicate・Function Context・Validation Rule・Relation・Result Field依存について別指定する。少なくとも以下を同一AttemptのReadinessとRowset Executionの**別票**にする。

| 対照 | Request / Contextの差分 | 許されるClaim |
| --- | --- | --- |
| **DC-R01** | RequiredのValidationだけをCatalogの正常JSONから省略。World/Index Fingerprintは元Replyのまま | Catalog schema OK・fingerprint一致でも未取得RuleはNOT QUALIFIED。実Sourceに当該Ruleが存在しないとも断定しない |
| **DC-R02** | SourceIndexがsubjectモードのままで全件Definition照合を要求 | wrong-mode/insufficient-responseとしてContextを保留し、Source欠落に置換しない |
| **DC-R03** | 10 Replyの各Source FPは同じだが、途中でSource/Grant/Selectionが更新 | 個別Reply同一性≠原子的な一World Snapshot／実Query実行時点の権限 |
| **DC-R04** | Required Rule/Relationが揃ってもProviderに実接続されず、EのSQL＋statusは未実行 | Context局所Readinessは進んでもHR-01/02はNOT RUN。必要Source/Parser/Roleの証拠とProduct実Read証拠を分ける |

既存#3262が必要設計情報の分母・完備条件、#2514が独立Expected、#2586が物理Reader、#1163が実行PacketのOwner。選択したRule1件の受領証拠を全Data/Function/Businessの意味受理に拡大しない。

### 20.2 CLIのFile Effect表示はmainで実装済み、実Command Effectの広告は未完

旧第19章の「名前付き出力の広告は未統合」を**局所的に訂正**する。[main merged #3333](https://github.com/hat47x/tei-platform/pull/3333)は既存 tei.cli.discovery/v0 の view に writeFlags（-operation-log、-lifecycle-state）を追加し、実CLIで引数を指定した際のJSONファイル／SQLite Stageファイルの生成と、指定しない場合の非生成を対照した。固定HEAD 78bf41c99964255c4372ac0a71556538d40b83d2 の[Health #42386374310](https://github.com/hat47x/tei-platform/actions/runs/42386374310)はsuccess。現mainの cmd/tei/cli_contract.go でも read-only と writeFlags の両欄を確認した。

ただしこの read-only は**正本の意味変更ではない**との定義に限られる。Caller指定Pathへの物理File書込みを可視化しただけであり、-executeのCommand/DB Effect・実Grant・外部Consumerが見た結果は網羅していない。[未統合 #3302](https://github.com/hat47x/tei-platform/pull/3302)の excludedFlagNames によるInvocation Variantのread-only範囲制限は、mainへ入った writeFlags と同じ解決策として**採用確定させない**。RMTのHost/Harnessは必ず実argv、writeFlags、出力先、Command／Provider Targetを照合し、広告されたCommand名だけで副作用なしと認定しない。

### 20.3 RC-03/RC-05：実行後の記録障害は、以前の拒否ではなく効果不明として扱う

現main internal/command/invoke.go の Recorder.Record 失敗は依然 COMMAND_RECORD_FAILED を返すが、**EffectUnknownを付けない経路**がある。main lifecycle.go の MayReplay はStageDispatchedをtrue扱いする経路があり、実Dispatchの再送拒否と一致しない。[Draft #3328](https://github.com/hat47x/tei-platform/pull/3328)は前者を NewUnknownExecutionError に変更し、後者を既存Attemptが存在する限りMayReplay=falseに合わせ、LifecycleのObserve/MarkUnknown失敗を効果不明として伝播する5 Go対照を加えた。固定HEAD 1b8ff4e68d840a6981f4c2b2ea2fb310c3e1765f の[Health #42385861954](https://github.com/hat47x/tei-platform/actions/runs/42385861954)はsuccessだが、**Draft未統合／TestはReference Executor＋Memory UoW/Stage Fake**。実PG Commit ACK消失／Journal crash／再起動・別Host競合は未実証。

新しい受入はRMTの既存F-Post/RC-03/RC-05へ次の限定の正負対照を付加する：Executor前の権限拒否→Effect 0/Rejected、Executor後Recorder失敗→Effect Unknown/同Attempt安易な再送禁止、Commit後Stage Observe失敗→元Stage Dispatchedでも再Dispatch禁止、Unknown記録自体失敗→外側返却でもUnknown保持。**Stageが永続化されない場合、ローカルの再Dispatch拒否だけでは別Host再送を阻止できない**。Receipt K・独立Postimage O・Owner Meaning Bを回復するまで確定成功/失敗を作らない。既存#1174/#2996/#2502/#1163のOwnerに再実験を渡し、第二Journal/Coordinatorは作らない。

### 20.4 Hookと暦の標準機能はX-TEA由来の意味対照へ接続する

[main merged #3306](https://github.com/hat47x/tei-platform/pull/3306)は、Hookのusesが指すCapabilityに対応するApplication Command Patternが**存在しない**場合に検証拒否を加えた。これは旧第19章の「未統合」を訂正できるが、複数同Capability Commandの曖昧性、位相とCommandの結合、実Runner/Plugin/Providerの呼出しを解決したわけではない。[PR #3348](https://github.com/hat47x/tei-platform/pull/3348)のPhase順実行と途中停止のGo実装は**現時点未統合**であり、TxnとPhase、Plugin合成順序、実Product Callerは未資格。対応するData Script/Hookの実動作Parityは、Read/Update二入口の本番同Rule実行時に別検収する。

[main merged #3343](https://github.com/hat47x/tei-platform/pull/3343)では、X-TEAの暦コードに相当するholiday-calendar Roleを追加し、複数暦が同じ休日Dataを共有するとき、Calendar指定なしの全行読込を拒否して別暦休日の混入を防ぐ。これはDate Role・timezoneとあわせた**参照用Binder/Row変換**の前進であり、旧DriverのKBCALENDAR文字列/日付書式/実DB/年会計期間の意味Parityとはまだ別。暦Dataに紐付くCalendar IDの実妥当性と旧Provider/Owner Expectedは未受理。HR-01/02のstatus Queryに関係ない暦全機能を先行Hard Gateにしない。

### 20.5 実Operation DriftとPlugin能力選択は物理Source/Provider消費とは別

[main merged #3325](https://github.com/hat47x/tei-platform/pull/3325)のread-only参照drift検査は、Callerが渡すWorldFingerprintとModelRevisionを現在値と照合し、current／stale／uncheckedを区別する。Basisだけでは属性変更を検出できないという独立負例がある。**これは実Commandの実行直前検査でも選択SQL raw bytesの検査でもない**。EのActual Source SQLと現在Provider/Operationを別に記録する。

[main merged #3337](https://github.com/hat47x/tei-platform/pull/3337)は宣言Plugin CapabilityとEnvironmentの選択を参照的に棚卸しできるが、未選択は直ちに欠陥ではなく、**選択済み＝実Provider Loader/権限/接続成功ではない**。database-providerの技術名とCapability IDの比較差は[未統合 #3350](https://github.com/hat47x/tei-platform/pull/3350)が既存ValidationのSupportsDatabaseProviderで改める候補。RMTのsales→sales_core物理BindingはWorld上のPlugin候補の選択ではなく、**当該Attemptの実Provider Instanceと実行SQL/Rowset**で検収する。

**変更後の最短順序とClaim ceiling**：#2586 E Route Product Query/SQL+status/Provider/Rowset HR-01/02 → #2506/#2514の対象集合L・Result Slice・negative dependencies/Scope → #1125/#1174の二入口同Ruleと実K/O/Business B。#3262 TT-56をContextの前段品質、#3182 CLI writeFlagsをTool Scope、#1174 RC-03/05をCommit後回復、#1125のHook・暦を独立Meaning対照へ分担する。**今回新Go/実E Product PG/旧Windows Runtime/実Owner受理の実験はNOT RUN。G1/G2は引き続きUNQUALIFIED**。本追補は既存RMT Markdown 1ファイルのみ変更、過去のCI成功を新HEADへ流用しない。

## 21. 11/8追補：Context11応答のmain統合、空集合Handoffの停止線、原典Source/実Providerの独立性

本章は第20章の最新実装状態を更新し、TEI参照モデル完全移植のRMT P0-R1を進めるために**どのClaimを補強でき、どれを補強できないか**を明確化する。E Route（examples/xtea-lowcode の order_entry.recent_orders、SQL status=:status）、W Route（reference/workspace の :from）とC Route（Web Reference実PostgreSQL）は引き続き別のSource・SQL・Provider・Scope。優先度はE Product Function→実Query/SQL→実物理Provider→Header Rows→Function Contextで変更しない。新Parser/Context Engine/Scenario SSOT/Workflow Host/Authorityは設けない。

### 21.1 TT-56はDraft二件からmainの単一Gateへ統合済み

旧第20.1節の「#3326 Draft／#3324並行候補」は**現在地が変わった**。[main merged #3354](https://github.com/hat47x/tei-platform/pull/3354)が既存context_gate.pyに両者の限定成果を統合し、旧#3324/#3326を**closed/unmerged**に整理した。重複した2つのGate／独立Expected正本を作らない。実際の構成は**11件のTEI CLI read-only Reply**（元の9＋全Indexモードの source-index＋Focused semantic-validation inspect）、独立Required 3 Validation、Rule Canonical・Relation・Definition SourceとそのLoader/Source Index/Fingerprintの相互照合、及び**32件の合成Consumer対照**。固定HEAD 0ab2b99f028a867da6681239f8b79691b9a877b4 の[専用CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)と[Health #42392009764](https://github.com/hat47x/tei-platform/actions/runs/42392009764)はcompleted/success。現在main tools/reference-issue-test-ability/context_gate.pyにも source-index-all／inspect-validation を確認した。

この前進は**必要と先に決めた公開合成3 RuleのScopeに限定した局所Context Readiness**。32件はCLI応答の合成改変に対するConsumer Controlsであり、製品コードMutant killや未知RuleのAI発見数ではない。結果は引き続き boundedReference=partial／testExecutionAdmission=exploration-only／qualifiedTestGeneration=false。CLI11応答を順に取得しただけで**原子的な同一Snapshot／実Providerまたは現在Host Grant／外部Harness完全受領**は証明できない。

RMT E01/02の独立Requiredには、単に名前が見つかっただけでなくQuery/SQLに必要なRule・Relation・Result属性、Source Location、実Loaded/Parsed、Current Selection/Runtime Inputと不在・未取得の区別を含める。source-indexのsubjectモードとindexモードを区別し、正常JSON・一致FingerprintであってもRequired Rule欠落を不適格にする。Context ReadyとProduct Readerのactual SQL/Rowsetは**別票**である。

### 21.2 空集合の証明と外部Handoff受領は二つの義務

[main merged #3353](https://github.com/hat47x/tei-platform/pull/3353)は internal/semantic/completion.go の **EmptyPolicyHandoff＋Targets=[]＋EmptyJustified=true** において、HandoffConfirmed=falseなのに CompletionHolds へ進み得た例を修正した。空集合枝でHoldsに達しても引継ぎ確認がなければ**CompletionUnverified**。Denyによる確定Failsを上書きせず、非空Targetの既存確認を維持する。固定HEAD cf3de9a61906167b3b5cc2fbd246e238a5018d5f の[Health #42394613242](https://github.com/hat47x/tei-platform/actions/runs/42394613242)はcompleted/success。実Go Codeをmainから再読取して当該停止枝を確認した。

| 受注／業務引継ぎ受入の独立対照 | 対象集合の成立 | Handoff受領／判定の資格 |
| --- | --- | --- |
| **HC-01 未取得の0件** | 1 Pageが0件、CoverageがPartial/Unknown | 空集合Lを証明しない。Handoff・業務Completionは未確認 |
| **HC-02 真の空集合・Handoff未確認** | Owner対象Lが空であることを独立に示したと仮定しても、HandoffConfirmed=false | Handoff型はUnverified。空集合の証拠を受領へ代用しない |
| **HC-03 CallerのHandoffConfirmed=true** | Callerが真を報告し局所EvaluatorがHoldsを返し得る | 実Host/相手方Receipt・受領者・時点・対象集合・Disclosureは**独立に未証明**。G2/Bへ昇格しない |
| **HC-04 明示Deny／履行失敗** | Independent Expectedが拒否／確定違反を示す | 安易にUnknownへ丸めない。実行前の拒否と実行後の効果不明も分離 |
| **HC-05 受領だけが新しい／Readが古い** | HandoffとReadが別Epoch・別Scope・別Tenant | 受領フラグがあっても同じ対象集合・Operation/Actorに束縛されるまで完了資格なし |

これは既存#1125 Business Procedure／#2506 Result Slice／#2514 Independent Expected／#1163 Evidenceへの追加検証観点。**Target集合Lの完全性、Predicate実評価、空集合の独立証拠、Handoffの実受領、現在Grantは5つの別義務**とし、証拠がない箇所をCallerのboolで埋めない。Universal Readの0件とHandoff Receiptの0件も同義ではない。

### 21.3 Plugin Providerの選択規則変更・原典暦の値域を実行資格から切り離す

[main merged #3350](https://github.com/hat47x/tei-platform/pull/3350)は、read-only参照のPlugin能力選択を技術名とCapability IDの文字列比較ではなく、既存の SupportsDatabaseProvider に従って解決するよう修正した。参照Worldで宣言12能力・未選択6→4となり、Postgres/Oracleの database-provider が選択されたと数えられる。**以前の6件観測を当時の方式による履歴として保持**し、今回の4件を実Provider binary/Load/接続/SQL ExecuteのPASSへ変換しない。RMTでは sales datasource alias→sales_core→実DB Instance/Target SQL/Subject Grant/Rowsetを独立観測する。

[main merged #3358](https://github.com/hat47x/tei-platform/pull/3358)はX-TEAの暦コードに対応するHoliday Calendar属性のValueDomainを型の宣言値域から解決する。宣言された閉値域外のコードを報告でき、値域宣言がない型は**開集合のまま**で勝手に全拒否しない。固定HEAD c3e39a4a4047bcff834bc3ac1198e0f7091b5e66 の[Health #42393319879](https://github.com/hat47x/tei-platform/actions/runs/42393319879)はsuccess。ただし **HolidayRowsForCalendarへのValueDomain.Validate自動組込み、実Workspace例の値域、旧Driver/実DB/Ownerの意味Parityは未実装**。暦機能は原典R/Bの別受入に割り当て、Eのstatus Header Readの前提にしない。

### 21.4 Raw資料／Tool受領／Journalを別扱いにする

[未統合 PR #3356](https://github.com/hat47x/tei-platform/pull/3356)のCLI dogfooding第2回は、単一ビルド実行物でread-only CLI面を再実測し、**MCP専用Server入口は現状未確認でMCP自体NOT RUN**と記録。CLIが使えることをMCP／外部Harness Tool配送成功とみなさない。一般MarkdownがmanagedFilesにあってもreadByLoader=falseなら、原典Java/SQLや設計Markdownの**Raw存在／実Reader取得／構造解析／TEI Accepted Meaning／Business Source束縛**を別に検収する。[Design Intake Draft #3355](https://github.com/hat47x/tei-platform/pull/3355)のP0-D Raw資料到達性、P0-I既存CLI設計情報、並行P0-M三視点/CRUDとP0-R実Read、P0-C同一Subject/Scope接合を参照する。Raw保持やGraph/SHO出力をE Product ReadのHard Gateにしない。MCPを欠くことはCLI経路でHR-01/02を実証する妨げにはしない。

[main merged #3357](https://github.com/hat47x/tei-platform/pull/3357)はAccess/Issuance/Runlog/EffectlogのJSON Lines読書きを共有実装へ整理し、0600、欠損ファイル空、破損行はエラー＋既読Prefix、という既存の扱いを保つ。これは**改竄検知、厳密な原子公開、効果とReceiptの因果照合を実現したわけではない**。特に「壊れた行より前の記録が返った」をJournal全件正常・当該Command結果確定・再送安全へ昇格させない。第20.3節の[未統合 #3328](https://github.com/hat47x/tei-platform/pull/3328)が扱うExecutor後Recorder/Stage書込み失敗と組み合わせ、当該AttemptのRecord/Stage/実DB Receipt/別Connection Oを独立確認する。#1174のRC-03/05は引き続き未受理。

### 21.5 次の一試行・実証済み境界・停止位置

| 順 | Owner／既存試験の前進 | 今回の判定 |
| --- | --- | --- |
| **P0-I：Required Context** | #3354 main、11 Reply/32局所対照の限定PASS | 3 Rule Scopeの局所検査は達成。全Required Closure/実行時Grant/Consumer受領は未資格 |
| **P0-R：実Product E Read** | #2586 HR-01/02、Query定義とBuildBrowse表示解決 | **未達：mainのCLI PatternBrowse分岐はBuildBrowse Viewを返すが実Provider Rowsetを呼ばない**。Query実Byte→Bind→physical Readerが最初の欠落 |
| **P0-B：Completion／Handoff** | #3353 main、空集合Handoffの誤Holds防止 | 実Read全L、外部受領、同Attempt Host Scope、Owner Meaningは未資格 |
| **P0-Effect/Recovery** | #3155単一Resourceの実PG別読取、#3328未統合の効果不明修正 | RMT二入口の全Effect K/O、実Journal crash/再送、Business Bは未資格 |
| **原典Meaning/Target** | #3350 Provider選択規則、#3358暦値域、#3357 Journal整流 | 旧Windows Driver実挙動とOwner受理、別Target/Host同値性は未資格 |

この章で**E Product SQL実行、実Header/Lines Rowset、旧X-TEA Windows Driver、実Host/MCP/SHO、Owner正式意味受理は新たに実施していない**。C Route隔離PGはReferenceの有限成功、Provider選択は宣言解決、Context11 Replyは限定候補取得。G1/G2はUNQUALIFIED。次は#2586のE Route Query実SQL＋status→実PG Reader→Function Data Context／Consumer一試行を、#1163同一Attempt Packetと#2514独立Expectedで実証。その後に明細全L・Handoff/Disclosure・二入口保存前拒否・実K/O/業務Bへ接続する。新Go/Fixture/Scenario/Hostの第二正本は追加しない。

今回の変更は本RMT計画Markdown 1ファイルのみ。変更前HEAD fc8f8b83f4f1b82ab1920027ce8529bb15b19250 の[Health #42391972484](https://github.com/hat47x/tei-platform/actions/runs/42391972484)はcompleted/success。**新HEADのCIは別結果が出るまでPASSとしない**。

## 22. 11/8追加：Browse操作契約の実効性と部分テスト受領をP0-R1の次の判別にする

§14/§18/§21ですでに特定した**唯一の初回不足辺**（Eの`BuildBrowse`定義投影後、選択Query/SQL Source→実物理Provider Read→Rowset/Function Contextが未接続）を置き換えない。2026-11-08現行main `5c28427b4e705f0428d2dc2f8e065892fc96d607`の4 Source（`patterns/recent-orders.tei`、`queries/recent_orders.tei`、`queries/recent_orders.sql`、`internal/applicationview/browse.go`と`cmd/tei/cli_view.go`）を直接再照合。**新しい発見は、実Readができた後もPatternのフィルタ・整列・範囲契約の各一項目を別に検収しなければならないこと**である。

| 現行Sourceで確かめた契約 | Source-staticに確認した差 | 初回Product Read後の既存Caseに追加する独立判別・合格上限 |
| --- | --- | --- |
| **Filter / parameter** | Patternは`filter sales.order.status {eq ne} parameter status`、Queryは`input status std.code`、現SQLは`where status = :status`。`BuildBrowse`は`FilterRef`を返すが入力値や演算子を実SQLへ実適用しない | **HR-01/02**：`eq`の値・型・selected SQL+bindを実証する。`ne`は許可UI選択肢だが、固定SQLが`ne`を実現したとは認定しない。実Provider/上位Operatorの明示実装と負例がなければ`ne`実行資格はunknown。未bind、NULL、誤値、異Tenant/同名別SQLによる偽一致を分離 |
| **Order / page** | Patternは`sort sales.order.status asc`と`page-size 20`を宣言。一方、現SQL原文には`ORDER BY`/`LIMIT`がなく、BrowseViewはSortRef/PageSizeの**定義値**を保持するだけ | **HR-03**：21件以上・同statusのtie・2ページ・途中追加/削除の反例で、実行側のOrder/Limit/Continuation/Snapshotを計測する。単なる昇順という表示の存在や「20件返った」だけで安定Order/完全性としない。SQL外の実変換があるなら実呼出し位置／処理済み行を証拠化 |
| **非出力列 / 対象粒度** | 現Query resultは`order_id`/`customer_id`の2列、`status`は非出力のFilter/Sort依存。SQLにTenant述語はなく、明細Lineも出力しない | **HR-02/04/05**：非出力statusが実DB Predicate/Sortで評価されたかとConsumerの公開2列を分ける。Tenant分離は物理Instance/RLS/明示Scopeの実証がある場合のみ主張し、**EのSourceに存在しないTenant列やLine2件をEの独立Expectedに紛れ込ませない**。RのOwner TargetSet L/明細は別Read/別Claim |
| **SQL source と実Dialect** | `sql postgres file "recent_orders.sql"`のSourceと`:status`、`sales.order`という表記は見える。Product CLI `view`のBrowse分岐は`applicationview.BuildBrowse`を呼んで定義だけ返す | **HR-01/02**：実SQL byte→選択Dialect／実テーブル識別・Named Parameter変換→Provider SQL/args→別観測のRow IDを一致させる。原文のPostgreSQL適否・escape/quote変換・実接続を静的推測しない。UNBOUND_DATASOURCEは別の未充足Guard、宣言上のPlugin selectedは物理Connectionではない |

### 22.1 他レーンの「限定PASS」を一つのRead資格に足し算しない

- [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)は既存6-Step pinned Scenarioへの**部分受領目録**を試験し、`notObserved`と`untestedConfirmed=[]`、局所Stepの肯定と全体Coverage `unknown`を分離。exact HEAD `1e2531fede563ed4b1d32cca0e78ccaf69fe364c`の[専用Dogfood CI #42394673357](https://github.com/hat47x/tei-platform/actions/runs/42394673357)と[Repository Health #42394673331](https://github.com/hat47x/tei-platform/actions/runs/42394673331)はcompleted/success。ただし**test-only投影、部分目録コピーの7反例、Draft未統合**。E HR-01/02で実Reader Caseが見当たらないことを「未テストが確定」へ変換しない。必要Caseがdiscovered/attempted/actual PASSかは既存#1180/#2514/#1163へ確認する。
- [Draft #3378](https://github.com/hat47x/tei-platform/pull/3378)はCLIとStudio Hostで同じData/Rule/Relationの**直接Inspect**結果を比較する製品Go試験候補。両方とも`operations.InspectSubject`へ直接進むため、成功してもPlugin Provider呼出し／`order_entry.recent_orders`の実SQL Read／WebViewへの実Row受領にはならない。照合時exact HEAD `d871afc75ca196fd05d0a4329cb7efb33020d271`のHealth #42396412440はqueued、**PASS未認定**。D1 InspectorをP0-R1の前提にしない。
- [Draft #3379](https://github.com/hat47x/tei-platform/pull/3379)の過去`database-provider`選択Evidenceは、`ruleFingerprint`が変われば同じPlugin/Version/能力IDでも再利用失格となるcharacterization候補。現main #3350のselected数補正は既存の履歴として維持し、実Provider Build/Load/Invoke/現在Grant証明とは分ける。照合時Health #42396431442はqueued、**PASS未認定**。P0-R1では実選択Rule/Provider版をW1〜W4へ束縛して流用不可を判定するだけで、Verified BOMを先行Hard dependencyにしない。

### 22.2 既存Issueへの最小受渡し／No-go

- **#2586 HR-01/02を先行**：Eに一意の実Connection/Source SQL+Status Bindを与える合成隔離環境で、固定Independent Expectedと実Provider 2列Rowsの一往復を先に検収。実Readerへの未接続なら`product-read-not-connected`で止め、`BuildBrowse`のResult Projectionを実Rowsetに格上げしない。**次の限定受入**として`eq/ne`、sort/page、hidden `status`の実行適合をHR-02/03へ追加する。新CLI Command/第二SQL Engineを勝手に設計しない。
- **#2622/#2611**：`sales`→`sales_core`→Physical Instance、実選択Dialect/Placeholder/SQL/args、statusの非出力依存、`ne`・stable pagingの真のRuntime能力を`supported/unsupported/unknown`で実測。未Binding、未許可、partial、empty、data-redacted、無結果とSQL失敗を同じ0件へ正規化しない。
- **#1163/#2514**：Query原文、Pattern操作意図、実Applied Filter/Sort/Page/Scope、Readerの実Rowset/Read epoch/extent、独立Expected、Case receiptの実受領と完全性を**同じE Attempt**へ束縛。別W/C実PGのPASS、CLI/Studio Inspect、Draft部分目録の局所PASSをEの実Read資格に加算しない。必要なら#2516が低権限公開Bodyの二世界対照を担う。R原典Parity／K EffectReceipt／Owner Bは別Claimで保留。

**現況**：Source-staticの契約比較と既存別HEAD CI確認を実施。新しいE Product Go/SQL/実PG/Studio実Read、Windows旧Driver、実Owner受理、同一Functionの独立PG Rowsetは**NOT RUN／UNQUALIFIED**。既存G1/G2/G4、#2586/2622の正式Owner基準、新Runner/Host/Proof Registry/Scenario正本を変更しない。#2783はDraft維持。変更後HEADのCIは新結果が出るまで未認定。

### 22.3 11/8現行mainのRead前段資格：入口で使う情報の意味を狭くする

基準main `2a9849dafdfe2c2d2d6a51991669c9dddd739de2`。既存P0-R1、RMT-HR-01〜05、E/W/CのSource分離、P0-R2の後段配置を維持する。次の5実装は**main統合済みの限定機能**として受け入れるが、いずれも`order_entry.recent_orders`の実SQL/Provider/Rowsetの代用にしない。

| main実装／確定した局所結果 | 今のHR-01/02への利用と独立停止条件 |
| --- | --- |
| [#3390](https://github.com/hat47x/tei-platform/pull/3390) `internal/syntax/semantic_anchor.go` | `type Status from std.code values {...}`のSource Index anchor走査を修正し、当該合成Workspaceの`source-index`/`validate`が以前のexit 2から成功に変化。**値域の合法性/業務Role/全Loader消費/実Query実行を保証しない**。ロード直後と実Reader時のSource/SQL原byte・Source Indexをなお照合（Draft #3383はSource-only driftの別負例、全Health [#42396735631](https://github.com/hat47x/tei-platform/actions/runs/42396735631) はfailureで未統合）。 |
| [#3404](https://github.com/hat47x/tei-platform/pull/3404) `internal/operations.ModelCatalogEntry.prelude` | `examples/xtea-lowcode`のCatalog全76 entry中、標準13 entryを`prelude:true`で区別可能。**必要業務Rule・Query・Relationの分母は独立Requiredから固定**し、13標準型が列挙されたことを利用者Sourceの存在証明にしない。preludeはSource Indexに現れないためSource absenceとは別。 |
| [#3403](https://github.com/hat47x/tei-platform/pull/3403) `tools/reference-mcp-readonly-adapter` | CLIと同じCatalog/Inspect/Trace/Validateの4読取結果を参照MCP Adapterで比較し一致。**製品MCP契約・実Harness受領・Function Query/Provider実Readではない**。P0-R1は実CLI経路で先行し、MCP提供をHard Gateとしない。将来のMCP Consumersは同Source/Required/実Result Replyへの完全受領を別検収。 |
| [#3408](https://github.com/hat47x/tei-platform/pull/3408) Feature selection実測 | `editor.sql_analysis`のPlugin欠落は`ENVIRONMENT_FEATURE_PLUGIN_MISSING`になるが、`editor.database_workbench`の同種欠落は`tei validate exit 0`になり得た（構成コピーを使った限定観測）。**この事例はEditor Featureの選択検査の非対称性であり、sales DB Providerの実Missingを直接示さない**。一般則として`validate ok`／`Capability selected`で全Required Provider Loaded/Invoked/Current Grantを証明しない。Eの`UNBOUND_DATASOURCE`も独立に検出・停止。 |
| [#3401](https://github.com/hat47x/tei-platform/pull/3401) `hook.RunSequence`／`ReadsFor` | 各Hook位相は宣言されたReadだけを受け取り、不足時は`HOOK_READ_NOT_OBSERVED`で停止。**値はCaller提供で実DB取得せず、`produces`/CLI・Studio/実Plugin・実Providerは未接続**。Rの同Rule二入口／Data Script/Hook受理で参照するが、E HR-01をHook依存にしない。 |

以上のmain限定機能は各実装PRのRepository Health成功を確認済み（#3390 [#42440925085](https://github.com/hat47x/tei-platform/actions/runs/42440925085)、#3404 [#42438438836](https://github.com/hat47x/tei-platform/actions/runs/42438438836)、#3403 [#42440216932](https://github.com/hat47x/tei-platform/actions/runs/42440216932)、#3408 [#42439322869](https://github.com/hat47x/tei-platform/actions/runs/42439322869)、#3401 [#42445671940](https://github.com/hat47x/tei-platform/actions/runs/42445671940)）。**別HEADであり、E Product SQL Readのテストは今回NOT RUN**。

### 22.4 Product Readへの先行順をさらに限定する

既存Query/SQLと`BuildBrowse`があるのに実Readの結果が無い現況で、**必要な新実装は「一つの実Binding→Provider Readerの接続」であり、Context/Reader/MCP/Hookの別Engineではない**。最初の正常Caseは`status`のeq実入力・必要なSchema/物理Instance/Provider確定・SQL byte/実args/Row identity/Read scopeを既存#2586 HR-01/#2514/#1163に束縛する。失敗理由は`no-instance-binding`・`no-product-reader-invocation`・`bind-invalid`・`sql-failed`・`read-partial`・`read-empty-qualified`・`host-denied`・`independent-read-not-run`を**Packet上の診断例**として区別（新製品enumを追加しない）。実Rowsetを一度得た後に、§22.1の`ne`、sort asc、page20、hidden status、Tenant/Disclosureを各Operatorの独立負例で検収する。Result 2列の成功をOrderLine全L/Business Bへ再利用しない。

追加の**Source-only drift**の圧力：Draft [#3383](https://github.com/hat47x/tei-platform/pull/3383)ではWorldの正規化JSONが同じでもSourceIndex fingerprintが変わるLF差分をCharacterize。固定Fixtureの旧Scenarioが走ることと現在Contextの資格が維持されることは別。**このPRはDraft・全Health failure**なので当RMT一Attemptの実成功へ加算せず、Source実消費・Build Input・選択SQLの現在版を照合する負例としてのみ使用する。

### 22.5 第22章前HEADのCI失敗と統合判断

本Draft前HEAD `a2ee4cc4fec4c07241ff9dbd80e5310136308b88`の[Repository Health #42396891587](https://github.com/hat47x/tei-platform/actions/runs/42396891587)は**completed/failure**。Job logでは`go vet ./...`、`go test -count=1 -timeout 60m ./...`、`tei validate -workspace reference/workspace`はPASSし、**Go 1.26.10の`gofmt -l .`が`tools/reference-plugin-capability-coverage/coverage_test.go`のdriftを検出**したことが失敗原因。RMTのMarkdown追加やProduct Reader test失敗を原因としない。mainでは[#3391](https://github.com/hat47x/tei-platform/pull/3391)/[#3398](https://github.com/hat47x/tei-platform/pull/3398)等でCI Go版の整形差へ対処済み。ただし**古いDraft branchが自動的に修正されたわけではない**。

最終統合では、古いBranchのまま連続追補をMergeせず、mainの最新Go Toolchain/Source/Planとの必要差分を再検査し、**RMT自身のMarkdown＋既存SQLite Fixtureだけを意図した変更として再資格化**する。現在Branchの大量behindを無検証のRebase/Force pushで処理しない。新HEADのHealthが成功しなければ`docs-branch-health-unqualified`、新実Provider Readが未接続なら`product-read-not-connected`をそれぞれ独立に保持。製品Main／DB／Windows/旧Driver／正式Owner Bは本追補では変更・実行せず、G1/G2はUNQUALIFIED。

## 23. 11/8再資格：Imported Sourceの鮮度・Function Required・実Hostを一試行の別証拠に

前章のE（examples/xtea-lowcode の recent_orders／status）、W（reference/workspace の from）、C（Web Reference実PG）を別Source/SQL/Scopeとして維持する。最優先は **E Product Function→選択SQL原Byte/実引数→physical Provider Read→Header Rowset→Function Context/Consumer** で変えない。横断Master Planは[main統合 #3481](https://github.com/hat47x/tei-platform/pull/3481)ですでにP0-D→並行P0-M/R→P0-Cを保持するため、重複修正しない。

### 23.1 同一WorkspaceのSource-only改訂は実証済み、外部ImportのRaw鮮度は別

[main #3477](https://github.com/hat47x/tei-platform/pull/3477)は、隔離Workspaceのvalidation.teiに末尾LFだけ追加してもCanonical Worldは変わらず、Source Index fingerprintだけが変化するため、古いWorld-pinned Scenarioは動いても旧Design Contextは拒否すべきことを**実CLI**で検証した。A→B→Aで原Byteと指紋が戻っても中間履歴や現在Grantは証明できない。[Health #42461032028](https://github.com/hat47x/tei-platform/actions/runs/42461032028)はcompleted/success。前章の旧Draft #3383の状況を現在の実装状況と誤認しない。

一方、[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)は**外部Import Providerの原文がConsumer側Source Indexに現れない**境界を検証中。Consumer World／Semantic Digest／Source Index／ModelRevision不変でも、Provider Raw末尾LFの変更で原Byte SHA256とLocated Source fingerprintが変わり、BuildSourceIndexFromLoadがErrImportsNotBoundを返すケースを既存externalPairで提示している。照合時HEAD 1451bf69fdf53592366be5a4a549e7e7d9560249 の[Health #42463036749](https://github.com/hat47x/tei-platform/actions/runs/42463036749)はqueued。**Draft・Go実テスト結果未確認**であり、main #3477の成功を外部Imported Sourceへ流用しない。

| Case | 不変なもの／変化するもの | 独立Expectedと停止 |
| --- | --- | --- |
| **SC-L1：ローカルRawのみ変更** | World FP不変、同一Workspace SourceIndex FP変化 | 旧Context拒否。#3477の限定Go CLIでPASS |
| **SC-I1：Imported Provider Rawのみ変更** | Consumer World/Index不変、Provider raw digest/Located Source改訂 | Imported Sourceを必要とする資格は再検査。#3479は未統合/CI待ち |
| **SC-I2：Import未束縛** | Consumer側の一見正常なIndex／ErrImportsNotBound | Source未取得を0行・不存在・成功へ変換しない。Nested Load共同Snapshotは未証明 |
| **SC-I3：A→B→A** | 現在byte・fingerprint復帰 | 中間履歴／Owner受理／実Grantを自動再利用しない |

**現在のE受注SQL recent_orders.sqlはローカルファイル**。Imported Provider Sourceの追加ReadinessをHR-01/02の新しいHard Gateにしない。外部Importが実際に依存として登場したAttemptのみ、Provider側Source occurrence・原Byte・Revision/Location・Load観測とConsumer Scopeを#1163へ追加する。後時点の再Fetchを最初のLoadの証拠にしない。

### 23.2 bounded Function Requiredは13 CLI Replyへ。Product SQL実行へは未接続

[main #3482](https://github.com/hat47x/tei-platform/pull/3482)は独立Requiredに、sales.order.submit の入力order／出力sales.order、order_entry Applicationのsales namespace・3 Validation・sales_core DB requirementを追加。既存11実CLI ReplyにTyped Inspectを2件追加し、**13実CLI Reply・42件の局所合成Consumer Controls**を実行した。PR専用[CI #42462672188](https://github.com/hat47x/tei-platform/actions/runs/42462672188)とmain squash 8ed583790e63b63527c1dd8f8aecc1d6ce6e3260 の[push専用CI #42462861657](https://github.com/hat47x/tei-platform/actions/runs/42462861657)はsuccess。全体Health #42462835497は本照合時in_progressなので合格を先取りしない。

これは**Command/FunctionのSource宣言の局所存在**であり、uses semantic sales は特定CapabilityのInvoke Bindingを示さず、13回のReadが一つの原子的Snapshotとも限らない。現在も boundedReference=partial、qualifiedTestGeneration=false。42件はProduct Go Code MutantではなくConsumer Replyの合成判別である。何より **sales.order.submitの宣言観測は、Eのrecent_orders Query/SQL/Provider実Readではない**。

| 既存レーン | 成立した限定事実 | 最初の不足 |
| --- | --- | --- |
| **P0-I Design Context** | #3482によるCapability/Application/Rule必須Sourceの局所検査 | 全Function/Business閉包、現在Grant・実Invoke |
| **P0-R E Browse** | Pattern/Query/SQL SourceとBuildBrowseの定義投影 | **#2586：status Bind→実PostgreSQL Provider→Header2列Rows→Function Context/Consumer** |
| **P0-E Command** | #3155の単一Resource別読取、[main #3472](https://github.com/hat47x/tei-platform/pull/3472)のCommand/Numbering型Adapter | 一つの実DB Tx／Receipt K／別Connection資源O／Owner B |
| **P0-X Consumer** | [main #3403](https://github.com/hat47x/tei-platform/pull/3403)の参照MCP Adapter、[main #3458](https://github.com/hat47x/tei-platform/pull/3458)のvalidate CLI/MCP文書同値 | 実Function Rowsetの同一Scope・Epoch・Disclosureと完全受領 |

### 23.3 実Go HostのHTTP応答とREA/SHO向けSchemaは別資格

[main #3478](https://github.com/hat47x/tei-platform/pull/3478)は実Go Host Handlerの10 HTTP経路とAudit/Status/Stage順を[専用CI #42461546535](https://github.com/hat47x/tei-platform/actions/runs/42461546535)で検証。**HTTP 202＋Audit completedはHost Requestの終結でありBusiness Effect／Receiptではない**。202送出後panicではAudit errorでもResponse Statusは202のままになり得るため、#1174のEffectUnknownと#1125の外部Handoffへ独立負例として渡す。[main #3485](https://github.com/hat47x/tei-platform/pull/3485)はE1 PlanとGo Hostが個別に正しくてもmethod/path/actionIdのペアが誤る合成二視点負例を追加し、[専用Node/Go CI #42462531490](https://github.com/hat47x/tei-platform/actions/runs/42462531490)はsuccess。ただし実TEI Graph→Host Binding、現在Grant、実Provider、Client受領は未証明。

[Draft #3486](https://github.com/hat47x/tei-platform/pull/3486)は既存Physical Schema Snapshotの複合PK/UNIQUE/FK、式・条件・Include・状態付きIndexをread-only Workbench Modelへ保持する実装候補。Fake Catalog→Provider.Introspect→Projectionの局所テストであり、**実PostgreSQL Catalog／REA業務Role・三視点CRUD・SHO意味のOwner受理、実Function Rowsetではない**。旧#3372を二重採用しない。P0-Mの構造Candidateとして参照し、物理FKから業務Key・Function CUD・現在Grantを自動確定しない。

### 23.4 次の一Attemptと判定上限

既存#2586が**E Route Product BuildBrowse→Selected Query→SQL原Byte+status実Bind→sales→sales_core→physical Provider→order_id/customer_id Rows→Function Context/Consumer**を一試行で実証する。#2514の独立Expectedと#1163の同Attempt Evidenceを使用。その後に第22章のne／sort／page20・非出力status・明細完全集合L、#2516 Disclosure、#1125二入口/Handoff、#1174 Receipt KとResource Oへ進む。

#3477/#3482/#3478/#3485はmainの**限定実証**、#3479/#3486は**未統合Draft**。製品E SQL Read・旧Windows Driver・実Host Provider/Studio/SHO Consumer受領・Owner正式Meaningは今回NOT RUN、G1/G2 UNQUALIFIED。本Draftへの変更は本Markdown一ファイルのみ、mainに統合された#3481 Master PlanやScenario正本へ重複変更しない。直前HEAD 20144e15e2e7a5329165524396828051fab9b205 の[Health #42461008246](https://github.com/hat47x/tei-platform/actions/runs/42461008246)はcompleted/success。新HEADのHealthは別判定。

## 24. 11/8追加：Functionは導出View、Historyは実Schemaでない、UI値は実Action値でない

第22〜23章の先行関係を変更しない。P0-R1最初のEの`recent_orders`は、実Query SQLのSource byteと`status`引数を物理Providerへ渡し、独立ExpectedのHeader行をFunction Context／Consumerが受領した一Attemptでのみ成立する。**概念Modelの宣言、Plugin能力選択、Schema移行履歴、表示UI、HTTP応答の個別PASSをこの一Attemptに足さない**。本章は他チャットのEverything is Data・共通Application API・Migration/Workbench・標準表現型・三視点/REA・Harnessの横断成果から、追加の不整合を独立テストへ配分する。新Function Registry、第二Runtime/Query/Host/Authority/Schema正本を導入しない。

### 24.1 導出Functionの同一性と実Operation/ProviderのIdentityを分ける

現行の[ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)を前提に、[PR #3495](https://github.com/hat47x/tei-platform/pull/3495)が現mainで整理したように、業務Functionを新しいCanonical Element/Registryへ格上げするのでなく、**既存Application／Pattern／Query／Command／Procedure／Rule／Dataの参照関係から導かれるView**として扱う。CLI/MCP/Web/Desktopは共通Application APIの別Transportであり、結果形が一致しても**Callerの選択したOperation、現在のProvider、実Invocation、Clientへの完全配送**は別資格。#3495はdocs-onlyであり、共通APIの全Transport実装完了ではない。

RMT Eでは`FunctionView=order_entry.recent_orders`の宣言存在、`Pattern→Query`のIdentity、選択SQL byteと`status`値／演算子、実Provider Instanceと行、Consumer Replyを**異なるWitness欄**で観測する。#3482の13 CLI/42限定Controlsが証明する`sales.order.submit`のCapability/Application/Source存在は、E Browseの実SQLやK/O Effectには流用しない。三視点[Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)は実WorkspaceのQuery／Relationを`operations.InspectSubject`で読む限定M1a testを加えたが、**Graph/CRUDのProduct導出やSource逆編集、Function実Provider Readではない**。旧#3263を二重統合せず、P0-Mの同一Source候補として並走する。

### 24.2 Migration Historyの未確定状態を適用済みSchemaへ昇格させない

2026-11-08のmain`internal/database/migration/history.go:CompareHistory`を直接Readすると、履歴`AppliedFailed`とChecksum不一致を除く`default`で`RepoApplied`が返り得る。よって**AppliedUnknown、State空、未知StateでもChecksumが一致すると、成功したDDLと誤認する経路がある**。さらにHistory取得自体がUnavailable/Partialなのに空`Entries`と見なされた場合、`RepoPending`の判断も実欠損と混同する。`CompareHistory`のコメント自体が「移行履歴の比較はActual Schemaの一致を推論しない」と明記している。

同じOwner #2892に[Draft #3489](https://github.com/hat47x/tei-platform/pull/3489)（結果に`RepoUnknown`を追加して下流に渡す方式）と[Draft #3497](https://github.com/hat47x/tei-platform/pull/3497)（`CompareHistory`が未確定Stateを**errorでfail-closed**する方式）が並行し、**結果型とConsumerが異なる**。両者を独立してそのままmergeしない。どちらを採るかは既存#2892/#2887 Ownerで、Workbench/CLI/Schema ApplyのConsumerが未確定を実行不可にできるか、`AppliedFailed`/`RepoDiverged`/Orphan/Pendingの従来意味を維持できるか、履歴Readerの独立Scope/Completenessを扱えるかで判断する。両Draftの最新exact-head CI/実DB Applyは今回未合格・未統合として扱う。

| RMT/DBクライアントの独立対照 | 入力 | 受入上限 |
| --- | --- | --- |
| **MH-01 偽Applied** | 同Migration ID＋同ChecksumだがHistory StateがUnknown/空/未知 | `RepoApplied`や実Schema適合にしない。方式はerror／明示Unknownいずれもfail-closedなら候補 |
| **MH-02 偽Pending** | History ReadがUnavailable/Partialなのに`Entries=[]` | 完全な空履歴ではない。履歴取得範囲・Provider・Grantを別票にする |
| **MH-03 偽同版** | History Succeeded＋Checksum一致、Actual Table/Column/Constraintが違う | 履歴成功≠Actual Schema。物理Catalogの独立Readと列・型・制約比較が必要 |
| **MH-04 事故の先送り防止** | Fixtureが既知Schemaで実ReadできるがMigration Toolの本番資格は未取得 | **P0-R1の隔離PG fixtureを別途固定し、実Catalog/Rowsを読む実験は先行可能**。正式DB移行/Cutoverの適格は別Gate |

[Draft #3486](https://github.com/hat47x/tei-platform/pull/3486)のFK/Index Workbench構造保持はread-only Physical Snapshotの**有限Fake Catalog投影**であり、実PostgreSQL Schema取得・REA Business Role・DDL Applyの証拠ではない。これをRMTの`sales.order`テーブルの実存在/許可の代用にしない。DB初期化が固定Fixtureなら、そのDDL原Byte／実Catalog／実Statement/Rowsを一Attemptで計測できる。Migrationプラットフォーム全機能の完成を初回Header ReadのHard Gateにはしない。

### 24.3 UIの整数・表示値とActionへ実渡した値を別Witnessにする

[Draft #3491](https://github.com/hat47x/tei-platform/pull/3491)は現mainのReact Starter`reference/ui-customization/web-react/starter.mjs`で`row`を正規表現検証後`Number(values.row)`へ変換する経路を指摘。`9007199254740993`がIEEE754整数化で`9007199254740992`になり、任意の後段`invoke`へ原値と異なる番号が渡り得る。既存`action-port.mjs`は`Number.isSafeInteger`で独立拒否するため、**現在のorder_entry専用Action Portで誤ったDB更新が起きた証拠ではない**。#3491の早期拒否とNode負例は現時点Draft・新HEAD CI未確認。並行React修正との統合は別Owner。

RMTの正負対照は、入力Raw文字列、型付き値/BigInt／Decimalの精度契約、実Action Invokeの引数、実ProviderのPK/Row Identity、UI/Consumer表示を**区別**して記録する。大きな行番号・採番・金額/数量の丸めは、表示時の仕様・業務単位／精度／Owner Meaningで決める。`MAX_SAFE_INTEGER`超は汎用JS Numberなら拒否または損失のない別表現を用い、丸めたままActionに渡さない。**RMT最初のRead-only Header E01/02**にUI Starter修正を新しいHard dependencyとせず、更新・二入口・明細Selectionや採番を検収する時点の独立受入に置く。

### 24.4 Plugin/BOMのRule Producer差をImported Sourceと切り分ける

[Draft #3490](https://github.com/hat47x/tei-platform/pull/3490)は[main #3350](https://github.com/hat47x/tei-platform/pull/3350)のPlugin能力選択規則変更によって、論理Capabilityと表示Rule Versionが同じでも`ruleFingerprint`が異なる過去Evidenceを流用しない局所Go試験を候補化。**Provider Source rawの変更**を扱う[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)とは異なる失効原因である。Consumer World/SourceIndex/ModelRevision、Imported Provider Raw occurrence、Plugin選択Rule Producer、実Build Artifact/Target、Host実Grant/Receiptを**相互代入しない**。#3479は照合中にさらにHEAD更新があり、最新実CIはqueuedなので前版PASSを現在HEADへ移さない。

既存#951/#1251のProvider選択・実Callと#3267のBOMで扱い、P0-R1では**実選択Provider＋SQL実行結果**を先に計測する。選択済みCapabilityや同Ruleバージョンだけで実Link/Invokeを語らない。BOM 2 Provider版×2 Consumer版の四tupleは、初回E Read後のTarget互換性検収で独立Expectedにより比較し、先行の単一Readを無資格へ押し戻す一括Gateにはしない。

### 24.5 最短順序と反証の保存先

**実施順を維持**：#2586（E Product SQL raw byte＋`status` bind→隔離PG actual Provider→Header 2列Rows→Function Context/CLI Consumerの最初の不足辺）→#2506/#2514/#1163（page、隠れ`status`、対象集合L/Scope、独立Expectedと受領）→#1125/#1174（同Rule二入口、EffectUnknown、Receipt K／別Connection O、Owner Business B）へ進む。

新たなMH-01〜04は既存#2892/#2887の履歴と実Schema、UI精度負例は既存#997/#1000/#2500、Plugin Rule失効は#1251/#3267、Graph M1aは#1124/#2497/#3492へ各Owner別に渡す。#3495のFunction派生View・共通APIは移植計画の機能Identity方針に取り込むが**新Canonical Kindや単一Hostへ組み換えない**。Read-only Source/Contextがpartialでも有限な探索・独立Expectedの用意は可能。ただし必須DB/SQL/ScopeやTool/Consumer実受領が未観測ならProduct ReadのPASS、G1/G2の受理はしない。

**今回の作業範囲**：現mainのMigration History/React Starter/Query BrowseをGitHub Sourceで確認し、関連PRの実/未統合/CI結果を確認したうえで、このRMT Markdown一ファイルだけ更新。新Go/Node/実E Product PostgreSQL/Windows旧X-TEA Driver／UI実ReactDOM／Host Provider／外部SHO・H3／正式Owner Meaning試験は**NOT RUN**。前HEAD 4d9168e6c86f34eb4502f0dcc93517b9d066f60c の[Health #42463387148](https://github.com/hat47x/tei-platform/actions/runs/42463387148)は本作業確認時queued。旧HEADのCIを今回の新HEADへ転用せず、新HEADのCI完了まで`docs-branch-health-unqualified`。G1/G2はUNQUALIFIED。
