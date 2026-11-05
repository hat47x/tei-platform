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
