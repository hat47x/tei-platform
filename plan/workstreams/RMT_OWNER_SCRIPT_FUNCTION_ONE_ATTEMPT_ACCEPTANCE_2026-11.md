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
| **E1 独立期待** | #2514 Oracle／原典旧Driver evidence #940、Owner #267 | 同じ入力の旧実動又は独立Expected。拒否/不明/例外、NULL、集合、時刻/業務対象Lを含めた期待 | 旧実動が無ければ legacy=unobserved。TEI結果でExpectedを作らない |
| **E2 Rule/Context** | #1129 Data Rule Phase、#2506 Function Mapping、#2387/2622 Query、#1126 direct Model | 二入口が**同一Rule ID+版+Owner**を参照し、必要Read/Rule非出力入力を満たすか。pure Derive/Validation/Transition/in-Tx/post-commitの区分 | partial/hidden/NULL/意味未受理ならunresolved。未取得をfalseへ潰さない |
| **E3 Action/Request** | #2485/#2598、[mainに統合された#2664 submit caller](https://github.com/hat47x/tei-platform/pull/2664)、既存Driver Binding | 選択したActionのBusiness Targetと実Provider書込み対象、typed input→Request byte/operation、現在Host GrantのBinding | Capability/型が一致しても対象・Grant・意味対応未確認ならexecution-unqualified |
| **E4 Effect/Recovery** | #1174実PG、#1129 Owner-write、#1163 Attempt | 同じRequestで初回書込み・否認・Conflict・ACK-loss・Receipt K／独立Read O／Lookup。二入口が同じOwner側Invariantを破らず、部分Kを全業務完了Bへ昇格しない | K不明でも「未実行」へ倒さない。安全なRecoveryが未証明なら自動再Submit禁止 |
| **E5 完了Claim** | #1125 Business条件、[Business Evidence §11〜15の#2671](https://github.com/hat47x/tei-platform/pull/2671)、#2582 negative依存 | Owner対象Lの完全性、外部Handoff/手作業、Query時点、K→O因果可視性、0件と不存在・Replicaの遅れ、二Providerの同時整合を別列にする | K/O一致・Client保存表示・空ReadだけでB=passedにしない |
| **E6 受領・納品** | #2604 Artifact集合/読者、SHO Composer、#2516 Disclosure、#1121 Target Lowering | 同じAccepted MeaningからGo単一バイナリ／必要時.NET/Java/WASM LibraryとSHO文書を生成したSource/Artifact版、実Readerが消費したbytes、Audience current Grant | 生成成功≠実Loader/外部Host権限/利用者受理。意味未確定の文書は提案として表示 |

E0/E1/E2の証拠は他のE3〜E6の実行許可ではない。各段の**同一Revision・同一Subject・同一Attemptの接合辺**を確認する。既存partial successの件数を足してG1/G2に昇格しない。実旧Sourceと正式Owner意味が無い場合は、P parity **保留**のまま、Aの合成/実Provider技術適合のみ前進できる。

## 3. 反証に使う差分ケース（独立ExpectedはOwner/Oracleで固定）

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
| **業務完了／公開** | #1125 Business Procedure、#267 Owner Decision、#2514 Oracle、#2516 Consumer、#1163 Packet。#2671や本#2783は**未統合** | 合成Rule／SQLiteの局所対照とPR単独Healthはある | 同じ受入Scenarioで原典実動→TEI実際の書込み→Owner完了→実Reader受領の全辺は未検収。G1/G2をPASSへ変更しない |

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

本章は§8のVS-RMT-01を**合格票として閉じるための必要条件**を定める。新たな受理Schema、Proof Engine、Test Runnerや共通Hostは作らない。近接する[Script起動設計Draft #2884](https://github.com/hat47x/tei-platform/pull/2884)（Required/Supplied/Evaluated/Enforced-at-Effect）、[自己記述Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)（候補自身のRequired自己申告は不可）、[テスト設計Draft #2811](https://github.com/hat47x/tei-platform/pull/2811)（Required Case／SKIP／Oracle独立性）、[DB別Query Draft #2862](https://github.com/hat47x/tei-platform/pull/2862)（DB実対象とSQL Realization選択）のうち、**参照モデル移植が判定する部分だけ**を受け取る。

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
- [merged #2835](https://github.com/hat47x/tei-platform/pull/2835)は実PG試験の`go test exit=0`で必須Caseが`SKIP`され得る反例と、それを拒否するCIの限定対処。VS-RMT-01の実PGスライスは、**実際に走った必須Test ID／subtest・実DSN Provider・PASS/SKIP/FAIL・負例の判別力**を記録し、単なる`go test green`へ帰属させない。#2514独立Expectedと同由来の自動Oracleは別。

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

[Draft #2890](https://github.com/hat47x/tei-platform/pull/2890)では、既存[#2828](https://github.com/hat47x/tei-platform/pull/2828)のJSON候補に対して**JSONのみ**の明細行番号重複負例を提案（新規テストGo本文は変更しない）。これは`Quantity>0`と`LineNo duplicate`を分離する良いMutation候補だが、Agentが書いたExpected自体は独立Oracle／Owner受理ではない。本日時点で#2890のexact-head CIは確認できない。

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
- **P1 旧Runtimeと業務Owner**：#940 Windows Workで旧Driver/JDBC/Session/別接続readを採集。#941公開条件、#2514独立Oracle、#1125/#267業務完了受理、#2516実Consumerを別Attempt・別Authorityで照合。
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

§11のP0-R1/R2で「Product APIが存在するか」を先に調べる判断を維持し、他レーンの新しい実Goの**反証**を、追加のシステム共通仕様ではなく**同じRead一往復の停止条件**として組み込む。照合したmainは`af969b88bb89d8262501086b67c2ba6add16e210`（従来§11の基準`8aa18ca...`より3 commit進展）。本章では新規ケース群を別正本化せず、既存RMT-RD1/RD2/WR1、#2586 HJ/QX、#2514 Oracle、#1163 Packetを参照する。

### 12.1 一回のRead資格を三つの独立条件に分ける

| 必要条件（独立の根拠） | 当該Function実行で残す最小Witness | NGの反例・Claim ceiling |
| --- | --- | --- |
| **Input Integrity**：Owner/Source LedgerのData・Relation・Rule・Actionの必要出典と実受理範囲 | Manifest/Import、exact loaded Snapshot bytes、Source Index/Parsed source spans、Role/Type/Ruleの当該Occurrence、Functionが実ConsumedしたModel/Query/Rule、実Build/Entrypointに結び付いたInput Closure | Requiredに無いSourceを「読んだ」とする、Parse後の別byte Fileを同版と称する、Roleの重複宣言が一つに潰れて曖昧性を失う → Source/Meaning資格保留 |
| **Read Fidelity**：Function Context定義と別に固定したBusiness Key/物理PK/Relation Role/Result Grain/対象集合/Predicate/Scope | Provider Alias・Target/Selected Realization＋実SQL bytes/Bind Args、Result Slice（行ID、取得状態、時点、欠損/NULL、完全性、Pagination/Read cohort）、Readerが実受領したRows | 非表示WHERE/ORDER依存、JOIN fanout、partial/hiddenで0件、未観測Read/Read失敗、別Tenant同番号、Query選択変更 → 「正しい全件Read」不認定 |
| **Independent Discrimination**：候補Go/Rule自身から作らない固定Expected、少なくとも一つのKnown Mutant・反証対象の根拠 | Sourceと区別したExpected/Owner status、実Go実行HEAD/Test ID/Case、Required caseがDiscovered/Attempted/PASS（SKIPではない）、mutant rejected、現在Grant/開示に応じたConsumer Witness | 生成AIがCandidate Expectedをそのまま再計算、Fixtureの重複JSON memberが暗黙上書き、別HEADのGreen流用 → 独立Oracle/実Consumer Claim保留 |

**三条件の論理関係**：Sourceのbyte一致だけではProvider Read成功を示さず、実ReadのRows一致だけでは必要な非出力Predicate/Scopeが読まれたと示さず、独立Expectedの局所PASSだけでは正式Business Meaningを受理しない。三条件が各Claimの必要範囲で揃って初めて**限定RMT-RD1 Read witness**を認める。正式Owner業務互換、二入口Owner-write、現在Grant/Commit、Consumer配布は別資格。

### 12.2 他レーンの新しい実証をRead一試行へ最小転写する

- **Load時と後Readの時間差／旧Source Indexの扱い**：[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)の実Go Referenceは、`workspace.Load`時のSnapshotと、その後に同じPathのbyteを書き換えた直接Readが不一致になるcharacterizationを追加した。旧IndexがLoad時bytesを保持することは限定的に確認するが、**全Workspaceのatomic snapshot／消費時の自動TOCTOU拒否**は示さない。RMT-RD1では「ロード時と実Runtime消費時のbyte」または明示Lock世代を照合し、後Readの同Pathだけで旧結果を復活させない（#950/#2823/#2595）。
- **Role重複による情報落ち**：現main `internal/syntax/parser.go`の`parseSemanticData`は`case "role"`ごとに`role = roleTok.Text`としており、重複宣言の最後の値がCandidateに残る。[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)のGo限定反証は「Role一意性を証明済み」という意味ではない。Reference Data/Function Relation Roleは業務対象粒度とAuthorityを変え得るため、重複/競合Sourceがあるときは**Role未受理・Writer/Action対象未確定**に留める。診断実装は#988/#2620、Source編集は#2621へ委譲。
- **実Query/Contextの負依存**：[Draft #2896](https://github.com/hat47x/tei-platform/pull/2896)のSystem CompositionはData Read/Command/Job/Publication/Manual Handoffを異なる成立条件として整理し、#2586 HJを初回実Readに指定する。[Draft #2885](https://github.com/hat47x/tei-platform/pull/2885)はSource閉包とRuntime Read閉包を分け、Not Exists/LEFT JOIN/hidden SQL/partial/Read Cohortを別資格とした。RMTではこれらを**RD2の単一Provider負例**にだけ借り、Plugin方式比較・全Domain Component・二DB実行をP0の必須条件へ追加しない。
- **ExpectedとFixture入口の自己追認**：[Draft #2895](https://github.com/hat47x/tei-platform/pull/2895)は`[1,2]`／`[1,1]`を独立の有限集合性で判定し、Expectedだけ反転／Fixtureだけ反転したnegative controlを追加。これはOwner業務意味の採用ではない。[Draft #2891](https://github.com/hat47x/tei-platform/pull/2891)はFixture JSONの重複**object member**の暗黙上書きを拒否する変更。一方、#2890の`[1,1]`はJSON配列の合法な重複**値**であり、業務Ruleに対する正当な負例。混同しない。#2514/#1180の既存Oracle/Scenarioへ接続し、本PR内で第二Go Harnessを追加しない。
- **DB接続準備と実Readの非同値**：現mainに[merged #2894](https://github.com/hat47x/tei-platform/pull/2894)の`tei dbcheck`がある。実コードは`pgx.Connect`、`SELECT version()`、**`CREATE TEMPORARY TABLE`／`DROP TABLE`**を実行する。よって`dbcheck PASS`は接続・基本操作の環境Readyを示すに限り、業務Query/Readや完全なread-only実行を証明しない。書込み不能な本番接続で無条件に起動しない。DSN未指定exit 2を実Readの合格/不合格に換算しない（#2622/#2586）。
- **Windowsの「緑」と旧Driverの実動**：現mainの[merged #2893](https://github.com/hat47x/tei-platform/pull/2893)はUnix signal依存テストをWindowsでskipする。Windows全Test exit 0でも対象Caseが未実行の可能性がある。#940 X-TEA Driverのactual GUI/JDBC/Tx/独立Readが完成した証拠ではない。必要Windows Test IDsと`PASS/SKIP/NOT RUN`を#1163/Owner票で別管理する。

### 12.3 P0実装担当が最初に提出する差分と停止条件

**対象は同じScenario VS-RMT-01の**P0-R1（読み取り）**であり、RMT-RD2を境界負例、P0-R2を後段へ配置する。新しいAPIを発明する前に、次の4資料を現在のGo実行・正本へ結び付ける。

| 最初のDeliverable | 実行後の受入基準 | 未達なら次に渡すOwner |
| --- | --- | --- |
| **D1: exact Source/Role Bind** | 同じModel/Function/Relation Role/RuleへのRequired出典、Loader actual bytes、Parser Source span、Role重複なし/衝突未解決の分類、実Consumer（Function/Build）までのReachabilityを比較。Byte同一だけでMeaning Acceptedにしない | #950/#1031/#2823、Role#988/#2620、実Build#2595 |
| **D2: actual Function Read** | 実際に呼んだ製品API／Provider／SQLと`Query target/Business Key/Physical PK/Relation Role`の一対一又はOwner宣言対応、固定2子行と負対照を実行。Row grain/pagination/cohort、NULL/未取得/非開示を報告 | #2586/#2506、Data Operations#2622、Mapping#2611 |
| **D3: Oracle/CI witness** | Candidateから独立したExpectedとmutant（別Tenant/重複Role/Partial/SQL候補変更）、Required Test IDの実Discovered/Attempted/Resultとexact HEAD、実Environment/Providerを保存 | #2514/#1180/#1163、Skipped CI#2835 |
| **D4: Write handoff（まだ実行しない）** | D1〜D3が成立した実Data Context/対象L/未保存Patchと同一Rule版を、二入口の保存前拒否P0-R2へ渡し、検証O1と実Payload/Grant/Owner Targetの同一性を証明することを依頼 | #1129/#2598/#1174、正式Business#1125/#267 |

**No-go**：RMT-RD1のProduct Reader接続が存在しないなら`product-read-not-connected`という計画上のGapとして止め、Go直接DBアクセスや自作FakeをProduct PASSとしない。Source/Role/SQL/Bound Targetがambiguous・partial・staleなら、ProductのData Read/Owner Write Claimを保留。旧X-TEA DriverをWindowsで操作できない場合はP（忠実移植）だけ`legacy-unobserved`を維持し、TEIのA（適合）の局所試験を止めるHard dependencyにはしない。

### 12.4 新HEADへ引き継がない証拠と更新状態

- §11を書いたHEAD `43462eecbd015e72004a93c91617849eb5fe9947`の[Repository Health #42107676122](https://github.com/hat47x/tei-platform/actions/runs/42107676122)は**completed/success**。証明対象は当該Docs/SQLite FixtureのRepo Healthだけであり、実Read P0-R1/R2ではない。
- **後着CI訂正（2026-11-06）**：#2895 HEAD 8d3dc2fcの[Health #42109027966](https://github.com/hat47x/tei-platform/actions/runs/42109027966)はcompleted/success（独立有限Oracleの限定テストを含む）。#2891旧HEAD 2f677921のHealth失敗に対し、修正HEAD 16b8d554の[Health #42110507652](https://github.com/hat47x/tei-platform/actions/runs/42110507652)はcompleted/success（JSON Fixtureの重複object member拒否の限定実装）。#2860 HEAD f5ec1e8aの[Health #42109350029](https://github.com/hat47x/tei-platform/actions/runs/42109350029)はcompleted/failureで、実ログの原因はreference/dsl-language-plugin/self_description_workspace_boundary_test.goのgofmt drift。go vet・go test ./...・Workspace ValidateはPASS。Role重複をProductで拒否できた証明ではない。これらの他HEAD・別契約の結果をRMT Product Read/業務Owner受理へ加算しない。
- 本§12は既存一計画Markdownのみの変更。独立Source/Role問題は現mainの実Code-static観測と別Draftの特性試験を引用しているが、**VS-RMT-01 D1〜D4の実Code path観測、実Function Read、Owner Write、正式Windows Driver、業務Owner受理は新たにNOT RUN／UNQUALIFIED**。新Plugin/Role Resolver/Query Engine/Runner/Host/Oracle Authorityの導入、main統合、G1/G2 Gate変更は行わない。

### 12.5 最初の実行票への圧縮：P0-R1を先に成立させる

追加の設計正本ではなく、§11のD1〜D4と§12の三条件を**一つの実検収・一つの欠落辺報告**へ縮める。独立Expectedは#2514/#1180、結合証拠は#1163、Gateは#1119を既存Ownerとする。初回は**単一の受理候補Function／一意のDB Alias／一Provider／Readのみ**とし、DB移行・二Provider・Job/SHO/G4完成を先行依存にしない。

| 着手順 | 完成させる最小実物と負例 | Stop／次の担当 |
| --- | --- | --- |
| **1. 実Go入口の確定** | 実Workspace Load byte／Source Index／Function→Context→Query Binding→Provider Readerの実呼出し列と、実SQL/Args・実Read Scopeをコードの**正確な関数名・HEAD**で記録。契約をValidateできることと実Readerを呼べることを区別する | 経路が未接続ならproduct-read-not-connectedを不足の一辺として記録し、#2586/#2506/#2622へ。Fakeや直接SQL ReadをProduct成功にしない |
| **2. 固定Scenarioの実Read** | Tenant AのOrder1001/Header1＋二明細、別Tenantの同番号と別Orderを独立Expectedに固定。**別Tenant混入**と**部分PageまたはJOIN fanoutのEntity数誤認**の2反例を最低限走らせる。実Rows/PK/Relation occurrence/grain/NULL/snapshot/完全性を確認 | 必須Source/Roleの競合、部分取得、対象SQL/Providerの不一致、非表示Predicateの未検査なら限定Readも未資格。#2586/#2514/#1163へ |
| **3. RequiredテストとOracle** | 実Case ID・Go exact HEAD・必須TestのDiscovered/Attempted/PASS・負例の実Rejected、独立Expectedの作成根拠を保存。#2895のKey重複有限Oracleと#2891のJSON重複member拒否は**別の局所検査**として再利用し、Business MeaningはOwner判定 | Go test全体緑でも必須Case SKIP/未実行なら未検収。Candidateと同じEvaluatorからExpectedを再計算しない。#2514/#1180/#2835 |
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
- **統合/Oracle #1163/#2514/#1180/#1119**：同じExpectedでもE/原典R/新Authoring Sourceは別Input/Attempt。Observed Test IDs/PASS/SKIPと実Binary/SQL/Provider/Read Cohortを束縛し、足りないClaimは`unknown/not-run/unqualified`とする。

**この追補の資格**：Source-staticなGitHub現mainの宣言／Go Test、既存merged PR・限定CIを再照合し、Draft #2783の既存計画本文だけ追補。**新たなGo test実行・PG Rowset／業務SQL実行・初回HR-01〜05・Windows Driver・実Web/Studio GUI・Owner正式意味受理はNOT RUN**。既存§12.5 P0-R1を優先し、G1/G2/原典parityは未資格。既存の別担当PR・Fixture/実コードに変更せず、Draftのままとし、**新HEADのtrusted healthとmainとの差分照合前にReady/mergeしない**。
