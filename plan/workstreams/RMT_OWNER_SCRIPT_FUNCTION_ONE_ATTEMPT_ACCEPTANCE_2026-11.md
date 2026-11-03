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

1. **P0-a #1174**：既存 \`internal/security/effectcommit/postgres/trusted_owner_integration_test.go\`、\`tools/reference-first-web-target/runner_order_submit_postgres_test.go\`、統合済み \`internal/security/effectcommit/submit/submit.go\` をProducerとして用いる。**初回post-commitの不正/未知Result→K→独立O→Lookup**の実PG負例を、同じRequestで実行できる範囲に限定。旧Receiptの**読取相当Tx終了error**は別[#2680 RR-P5](https://github.com/hat47x/tei-platform/pull/2680)（memory Provider）であり、実PG初回効果の証明へ混入しない。合成Grant/TEST-ONLY DDLは本番Authorization/Owner意味ではない。
2. **P0-b #1129**：同一Data Ruleを呼ぶ標準FunctionとAPI形FunctionをBindingし、両入口の拒否・状態更新・Owner側追加行・VersionとReceiptを**同じ独立Expected**で比較。Owner Resourceが二つならtransaction participant/Writerを明示し、同一Tx不可の場合の部分Effectと回復資格を隠さない。
3. **S0 #950／Work H2/J5**：原典Source provenance/旧実動・正式Target参照をprivateの固定版で検証し、公開Reportへraw内容・個人名・非公開モデルIDを出さない。Source Factの存在/曖昧度とMeaning受理を分離。
4. **E2E #1163/#1125**：一改訂・一AttemptのEvidence表へE0〜E6の出典、Case、検証したTool/Process、exit、読み取ったbyte、役割、現在性を追記。UIやSHOが旧版を読んでも「完了」と表示しない。相違は原Source・Rule/Mapping・Provider Bindingのどこを修正すべきかへ戻す。

**状態**：この計画はSource-staticな実装箇所と各Draft/Issueの報告を整理したもので、**新たなGo/PG/旧Driver/Studio/SHO実行・独立Expected採用・Ownerによる実業務意味受理は NOT RUN**。#1119 G0〜G4、#1174 P0-a、#1129 P0-bの未達を維持する。TEI専用Host、中央Workflow/State Store、二重Rule Source、全域CDC/共有Snapshot Registry、新Script Runnerは追加しない。
