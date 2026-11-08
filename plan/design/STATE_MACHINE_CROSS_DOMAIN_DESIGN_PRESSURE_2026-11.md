# 非業務領域を含む状態機械と状態図・状態表の交差投影（設計・反証計画）

- 日付：2026-11-03
- 状態：**設計候補・未採用**。汎用StateMachine Core、標準DSL構文、共通Runtime、状態表Schema、生成器、Studio機能の実装済み／採用済みを意味しない。
- 発端：S2Buriのデータ中心Lifecycleと分岐・合流の静的調査を、業務状態以外の**組込／ゲーム／UI／通信**へ拡張する。
- 対応：[Buri→TEI設計継承](BURI_TO_TEI_LIFECYCLE_TRANSFER_2026-11.md)、[業務Lifecycle v1](BUSINESS_LIFECYCLE_V1_DESIGN.md)、[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)。
- 目的：**状態遷移の共通の記述・検査可能性**と、領域ごとの**実行・保存・時間・安全性の違い**を両立させる。新しい汎用Workflow Engine／TEI Host／Registryを前提にしない。

## 1. 設計上の見直し：Business Lifecycleは汎用State Machineの全体ではない

BuriではWorkflow/業務状態を中心に活動を表すが、TEIではステートマシンを他のドメインの可視化・生成・検証にも使用し得る。以前の「汎用状態テーブルを採らない」は**業務データの物理保存方式**についての原則であり、**抽象的な状態遷移モデルを扱わない**という意味ではない。

- **State Machine（候補の意味Role）**：どの状態構成に、どのEvent／Input／条件が作用し、どの遷移・Actionが許されるかを表す。状態の所有者／Scope、条件の評価方法／時点、観測結果を明示する。
- **Business Lifecycle（業務固有Role）**：受注・承認・取消などに意味付けされた状態と遷移。Business Rule、Owner、データ保存、Auth、Commit／Receiptとの接続は別の業務契約。
- **Business Procedure**：人間／外部工程を含む手順のMeaning。Stepの前後関係を表せても、全Stepが状態機械の実行トークンとは限らない。
- **Runtime Workflow**：必要な場合に限る永続活動Instance・待機・相関・分岐合流。汎用State Machineを表せるからといって全領域にWorkflow Instance表を求めない。
- **UI・Protocol・Game／Device State**：それぞれのRuntimeが責任を持つ状態。UIの通信中表示と実Socketの接続状態、Gameの認知状態と運動状態、組込の制御状態と物理センサ状態は同一とは限らない。

**過去議論との整合**：単純なIF／条件分岐を既存Function・Data Ruleで自然に表せるならそのまま使い、状態が複雑になって**許容遷移の網羅性、到達可能性、イベント順、影響範囲の説明が重要になる場合**だけ明示的なState Machine表現を比較する。Spring StateMachine等の特定Runtimeの概念／階層状態一式をTEI Coreへ自動輸入しない。一方、処理非依存のState Modelを再利用する発想を捨てない。

## 2. 最小の意味契約（固定公開Schemaではない）

| 情報 | 共通化を検討する意味 | Domain／Bindingによって変わるもの |
| --- | --- | --- |
| Machine IDとSubject | どの対象・インスタンス／対象集合の状態か。Sourceから再解決できる安定ID | 業務レコード／UI Component／Socket Session／Game Actor／Device ChannelのIdentityと寿命 |
| StateとState configuration | 許容状態の識別、初期化・終了の候補、同時に有効な状態集合／Region | 単一状態か複数直交状態か、階層・履歴・排他制約の採用条件 |
| EventとPayload | 起動原因のID、Payload型、発生元、scope・epoch、必要な順序情報 | UI操作、Network ACK、Frame/Tick、割込み、Timeout、業務Command |
| Transition | source configuration、Event、Target configuration、条件、Action参照、安定Transition ID | 優先順位、内部遷移、entry/exit、競合Eventの裁定、選択不能の結果 |
| Guard / Invariant | 状態と型付き入力Factに対する評価。true／false／unknown／errorを混同しない | 認可・時間・センサ不明・外部Read・安全性条件の評価責任と現在性 |
| Action / Effect | 遷移の許可判断と実際のEffectを分離。既存Function/Capability/Providerへの明示Binding | 描画、音、DB更新、ネットワーク送信、アクチュエータ駆動等の副作用 |
| Time / Ordering | Event time／処理時点／観測時点とReplay規則を、要求がある範囲で区別 | monotonic clock／deadline、tick、業務valid time、queue、遅着、retry |
| Evidence / Version | Source／Role・Operator版／意味受理／生成Artifact／実Event Traceを個別に保持 | 永続Receipt、telemetry、device-local trace、frame trace、UI Request epochの保存義務 |

最初から共通の総称Event Bus、長期保存表、Event Sourcing、Task Scheduler、Statechart実行器を構築しない。抽象の共有は**意味IDと検査できる遷移契約**に留め、実行方法はTarget／既存Host／専門PluginへBindingする。多Regionや階層化、State history、deferred event、優先順位は必要性を確認してからProfile別に導入する。導入しない機能を暗黙の既定規則で補わない。

## 3. 状態管理図と状態管理表を「交差」させる

TEI Modelerでは**状態図と状態表を同一Accepted Meaningの相補的なProjection**として扱うことを検討する。「二つの編集正本を用意する」案ではない。

- **状態図（Graph）**：Stateをnode、Transitionをedge、Region／階層をgroupとして、到達経路、分岐、合流、孤立状態、循環、構造上の複雑さを視覚化する。線の位置・配置・色はView metadataであって業務・機械意味の正本ではない。
- **状態一覧表（State catalog）**：State ID、説明、親／Region、初期・終端、禁止条件、入出時効果、状態不変条件、出典・版・未決を表示する。
- **状態遷移表（Transition table）**：Transition ID、Source、Event、Guard（型付き参照）、Target、Action／Effect分類、優先／競合方針、未知・例外時の扱い、Evidenceを1行の検証単位とする。
- **交差マトリクス（State × Event）**：行＝Source State／configuration、列＝Event種別。セル＝適用候補Transition IDの集合。空欄は「仕様上未定義／明示拒否／意図的無視／該当不能」を区別し、無断でNo-opとみなさない。Guard違いの複数遷移や多Regionを1セル1遷移へ潰さない。
- **Scenario・検証表示**：Graphのedge、表の行／セル、独立Expected、Source occurrence、生成Targetの該当箇所、実行時の観測Eventを同一Transition IDから相互移動する。未取得・未検証・未受理・Target非対応を表示する。

**交差照合する不変条件**：
1. 状態図の各edgeとTransition tableの各行が、同じ定義版・Transition IDとSource/Targetを参照する。図にだけある遷移、表にだけある遷移を診断する。ただし図が折り畳み／Filter中で見えないことは欠落としない。
2. 状態一覧のState集合と図のNode／Region、表のSource/Targetを相互検査。未定義、孤立、未到達、デッドエンドを候補診断とし、合法な終端や意図的待機はOwner／Profile判断を求める。
3. State×Eventマトリクスで未定義Event、Guard重なり、優先順位欠落、Event未消費、unknown/error時の扱いを検査。**完全網羅の主張は対象Event集合・入力Domain・時間・RegionのScopeが確定した場合だけ**。
4. 図で表しきれない優先条件、Actionの順序、Clock／Concurrency、受理根拠は表と出典に残し、図が簡潔なだけで意味を失わない。
5. 表／図の**編集逆写像**は1対1に特定でき、Version fenceを満たす場合だけPatch候補とする。複数Guardのedge削除や階層を畳んだ状態の一括変更など非一意・非可逆操作はread-onlyまたは明示差分レビューへ戻す。ユーザーの現在Grantも別判定。
6. Markdownに書かれた「説明用の状態図／表／テスト表」を、Role・Source owner・Accepted Meaningがないまま実行定義にしない。図・表の表示内容が同じでもSource・Meaning・Projection・Actual Traceは異なる。

**責務**：[構造とRole #2620](https://github.com/hat47x/tei-platform/issues/2620)、[Markdown/表/図のSource #2621](https://github.com/hat47x/tei-platform/issues/2621)、[Projection・編集逆写像 #2497](https://github.com/hat47x/tei-platform/issues/2497)、[Modeler #1124](https://github.com/hat47x/tei-platform/issues/1124)、[設計文書 #1803](https://github.com/hat47x/tei-platform/issues/1803)で既存機構を使う。Graphviz／Mermaid／表組みは**出力手段の候補**であり、DSLやGraph形式を一種類へ固定しない。SHOは同じ意味と出典から説明文・図表を導出するConsumerで、別の状態機械正本を持たない。

### 交差表示の小例（通信Session、説明用。正式TEI DSLではない）

状態図の対応（Transition ID）：

~~~text
Closed --[T1: connect]--> Connecting
Connecting --[T2: ack]--> Established
Connecting --[T3: timeout]--> Closed
Established --[T4: link-lost]--> Closed
~~~

| ID | Source | Event | Guard／対象 | Target | Effect／不確実性 |
| --- | --- | --- | --- | --- | --- |
| T1 | Closed | connect | attempt IDを新しく採番 | Connecting | 接続要求Intent。送信は別Effect |
| T2 | Connecting | ack | 同じattempt ID、期限内 | Established | 接続確立の観測。旧ACKなら拒否／破棄 |
| T3 | Connecting | timeout | 対象attemptのdeadline経過 | Closed | Retryするかは別Policy。外部接続解除の完了は推測しない |
| T4 | Established | link-lost | 現sessionの断線観測 | Closed | 再接続を自動許可しない |

**交差上の負例**：同じtick／処理窓でT2とT3が到着した場合、Event order・deadline判定・処理優先が未宣言なら図の線だけで次状態を決めない。古いattemptのACKが新しいConnectingへ届くABA、EventとSocketの物理状態の不一致、複数Regionの状態直積も別に扱う。これは説明用の小例であり、TCP等の実プロトコル適合や実装済みを主張しない。

## 4. 異なる4領域のPressureを同じ言葉で比較する

| Domain／最小モデル | 図表で検査する反例 | Runtime／保存・性能の固有条件 | 既存Owner |
| --- | --- | --- | --- |
| **UI 非同期操作**：Idle→Loading→Success/Error、Cancel | Submit A→B、Aの遅延応答がBの表示へ混入。LoadingからCancelした後の旧応答でSuccessにならない。UI stateとBackend Command結果は別 | Component／画面の一時State、Request epoch、再描画、Cancelは外部効果の取消保証ではない | #1124/#2497/#997、#1131 |
| **通信Session**：Closed→Connecting→Established、Timeout | 旧ACK・重複ACK・Timeout競合、再送、順不同、断線後の古いイベント | Session ID、イベント順、Monotonic deadline、Protocol Provider、接続寿命。DB status列やReceiptを一律に要求しない | #2620/#988/#1121、#2514 |
| **ゲームActor**：移動（Idle/Run）×警戒（Calm/Alert/Attack） | 二つの直交Regionがあるのに単一列に潰して合法状態を失う。更新順で結果が変わる、Tick途中のActionが再入、未定義Guardをtrueにする | Frame/Tick、入力snapshot、RNG seed・再現性、CPU/メモリ予算、認知遷移と運動出力のBinding。特定ゲームEngineへ専用Coreを作らない | #2493/#2620/#988/#1121/#2514 |
| **組込Device**：Off→Starting→Running→Fault、Reset | Fault中のStartが危険、センサ未取得を正常とみなす、Timeout後に再起動して危険出力、複数イベント同着 | 期限・安全側停止、Interrupt／Tick、bounded allocation、実Device Provider／保護装置の責任。形式検証やHard real-time保証は別資格 | #2493/#2620/#1121/#2514、専門Provider Owner |

業務受注／組織Membership／長時間Workflowの既存[BL-01〜08](BURI_TO_TEI_LIFECYCLE_TRANSFER_2026-11.md)とは、**同じ表現上の構造を使っても意味・効果が異なるか**を比較する。上の四領域は現時点では**検証設計・反例候補**であり、テスト番号・新Issue・新Role種別をこの文書だけで固定しない。既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Oracle／Conformance ProfileへOwnerから引き渡す。

## 5. 状態機械の候補を受理可能とする最低条件

**単純IF方式と明示状態機械の比較**を行う。「状態機械の方が高度だから採る」という理由は不可。実際の設計者／Domain expertが次を改善できるかを検査する。

1. **型・構造**：未知State/Event、重複Transition ID、Source/Target未束縛、Region外遷移、初期構成／終了状態の矛盾を診断できるか。
2. **遷移決定性**：同じEventに複数のGuardがある場合、重なり・優先・選択不能・未知を表示できるか。nondeterministic選択を必要とするDomainなら明示Policyと再現Witnessを持てるか。暗黙の行順／描画順で選ばない。
3. **時間・同時性**：Event ordering、deadline、タイムアウト／ACKの競合、遅着、再入、同一Tickの多Region更新を再現・説明できるか。説明可能な範囲だけ機械保証する。
4. **不変条件・到達可能性**：Safety invariant、デッドエンド、到達不能State、未処理Event、意図的例外を区別。静的解析で証明できない経路は未検証として残す。
5. **データ・効果**：純粋Guard／Transition decisionと外部Effectを分離。Effect未受領・失敗・未知を状態進行成功へ丸めない。DB/通信/描画/Device駆動はそれぞれ既存Provider/Hostの保証へ接続する。
6. **Source→図表→生成→観測の同一性**：同じMeaning Revision、Transition ID、Operator/Plugin lockを追い、図の表示位置と表の行位置の変化だけでMeaningを変えない。生成Goの同一byteでも業務/Device意味の受理と実Execution資格は別。
7. **Target適合**：Go単一バイナリ／他Host向けライブラリの双方で、選択された意味Roleの対応範囲を表示。組込のリアルタイム制約、ゲームのTick、UI非同期などを汎用Provider能力が満たさなければunsupportedにする。Go AST／専用Runtime／外部Engine採用は先行確定しない。

**静的診断とIndependent Expectedを分ける**：表の網羅チェックがPASSでも、イベント時刻や現物センサの正しさを証明しない。生成側の同じState TableからExpectedを自動写した場合、Oracleは独立でない。独立した手書きの入出力列、Mutation（Guard反転／ACK遅着／State重複等）、実Event Traceで判別力を試す。

## 6. 既存計画への追補先・順序・判断Gate

| 既存先 | 差分の責務 | 新たに採用しないもの |
| --- | --- | --- |
| [#2619](https://github.com/hat47x/tei-platform/issues/2619)／[#2620](https://github.com/hat47x/tei-platform/issues/2620) | 業務以外のState Machineを、標準構造＋明示意味Roleの**評価対象**に加える。構造共有と意味受理を分離 | StateMachine Core種別の即時固定、全域共通の実行器 |
| [#988](https://github.com/hat47x/tei-platform/issues/988)／[#2621](https://github.com/hat47x/tei-platform/issues/2621) | 専門家向け簡潔なAuthoring、状態表／図のRole識別、Source provenance／再解決 | Go文法の露出、図・表それぞれの第二意味正本 |
| [#2497](https://github.com/hat47x/tei-platform/issues/2497)／[#1124](https://github.com/hat47x/tei-platform/issues/1124)／[#1803](https://github.com/hat47x/tei-platform/issues/1803) | State Graph×Catalog×Transition Table×Matrixの相互参照、情報損失と編集可否、SHO文書へのProjection | 状態機械専用巨大Editor、GraphレイアウトのSemantic化 |
| [#1121](https://github.com/hat47x/tei-platform/issues/1121)／[#1063](https://github.com/hat47x/tei-platform/issues/1063)／[#2077](https://github.com/hat47x/tei-platform/issues/2077) | Role/Operator→Target別Lowering・適用能力・実行保証の区別、Go標準Targetと外部Host Libraryを検証 | 共通Scheduler/Host、すべてのTargetで同じリアルタイム保証 |
| [#1131](https://github.com/hat47x/tei-platform/issues/1131)／[#2514](https://github.com/hat47x/tei-platform/issues/2514) | State/Event Scenarioと独立Expected、異質Domainの反例、図・表の相互一致・Mutationの判別力 | 図からの自己生成Expectedを独立Oracleと主張する |
| [ADR-0024/0025](../adr/0024-business-lifecycle-ownership.md)／[Lifecycle v1](BUSINESS_LIFECYCLE_V1_DESIGN.md) | 業務Lifecycleの局所的所有判断は**提案のまま保持**。後に一般State Machineとの共通契約が示されれば意味の重複を再評価 | 業務Lifecycle新要素を汎用State Machine実装済みとみなす |

**進め方**：
- **S0（現在、設計のみ）**：受注・通信・UI・ゲーム・組込のState/Event/Transitionを同一の検討表へ並べ、共通／固有／未解決を区別。図↔表↔Sourceの失われる情報、現行TEI Referenceとの差を記録。**全新横断試験 NOT RUN**。
- **S1（小さな意味検証）**：UI非同期と通信Sessionを、既存Graph/Record/Operator/Roleに限定してread-only解釈。遷移表／図の相互ID、unknown、曖昧Guardの反例、独立Expectedを比較。必要なら専門PluginのRole候補を比較。
- **S2（直交性と安全性）**：ゲームの二Regionと組込Faultを比較し、階層化・直交Region・時間制約をCoreへ先に義務化せず、Pluginごとの対応可否・表現不足を測る。
- **S3（生成・実行は別資格）**：採用された有限subsetだけ、既存Lowering→Go artifact／他Host向けLibraryをTarget別に生成し、Source／Artifact／Actual Event traceと独立Oracleを照合。実Device／通信接続／フレーム予算の評価は専門環境で別。
- **S4（UI/SHO検証）**：同じAccepted Meaningを状態図・状態表・マトリクス・設計説明へ投影し、編集可否とSource逆写像の判別力を試す。S1/S2から派生可能だが、Owner意味受理とTarget実行を代替しない。

**採用判断**：明示State Machineが、IF中心実装よりも許容遷移・変更影響・異常Event・安全不変条件・レビュー理解を改善し、追加するRole/Parser/Generatorの複雑さに見合うことを、少なくとも二つの**異質なDomain**の独立Expectedと具体的不足で示す。小Domainで不利ならIF/Ruleのままにする。未解決のGuard優先、Clock、Event queue、実Target能力、Ownerを勝手に既定値へしない。

**停止条件**：未受理のDomain意味、曖昧な状態/Region、Event集合・入力Domainの不完全性、外部Reference版不明／stale、Graphと表の相互不一致、Source逆写像不能、Target非対応、安全上未知のGuard、現在Grant／Effect未確認のいずれかを検出したら、該当操作をunresolved/unsupportedとして止める（read-onlyの表示自体は対象Scope/未解決を明記して可）。既存Product Gate／Owner受理を変更しない。

## 7. Validation／Derivation／Flowと、ヒアリング中の設計修正から実装へ渡す契約

### 7.1 Buriの資料・ソースから確認できる範囲と不足

**確認済みの静的事実（Buri `fbaf32d`）**：
- [マニュアル §3-1-3～3-1-4](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/doc/%E3%83%9E%E3%83%8B%E3%83%A5%E3%82%A2%E3%83%AB.txt)に、JaWEでXPDLのProcess／Activity／Transitionを描き、ManualとAutomatic、条件式（`#data`参照）を定義する操作手順がある。**この図は保存・実行されるフロー定義の編集入口**であり、見た目だけの説明図ではない。
- 同マニュアル§4-1-5/6ではAND/XOR、`#data`／`#action`の条件、ToolでOGNL等の処理を呼び出す。[`wakanagoProcess.ftl`](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/resources/ftl/wakanagoProcess.ftl)は条件・例外遷移・Tool・Manual待機をJava生成へ接続する。
- 同マニュアル§4-2-3と[`ExcelBaseParticipantProvider`](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/common/participantprovider/impl/ExcelBaseParticipantProvider.java)に、Excelの権限階層と有効期間を読み込んでActivity候補の認可判定へ使う経路がある。Excel表はフロー内Ruleのすべてを代替していない。
- 同マニュアルの特徴一覧は「Excelにルールを記述するだけのルールライブラリを搭載」と記す。二次資料にはExcelのDecision Tableという説明もあるが、この調査で直接見つかったExcel解析コードは主として**ParticipantProviderの権限表**である。Excelを使う任意のValidation/Derivation/Flow決定表の実行器とその実動は、このBuri Repo内の根拠で実装確認できていない。
- [`BuriScriptComponentImpl`](https://github.com/seasarorg/s2buri/blob/fbaf32dd3ec2a815dfe3f4426f10d6b0ce53547d/buri-core/src/main/java/org/escafe/buri/component/impl/BuriScriptComponentImpl.java)はToolのExtended Attributeをスクリプト呼出しコードへ生成する。Toolに任意コードを置けるということと、顧客が図表を修正するだけでそのロジック全体を安全に生成できることは同義でない。

**未確認の帰属**：`Validation`／`Derivation`／`Flow`がBuriでそのままの名称・厳密な3モードで独立実装されていたという主張は、確認した公開マニュアル・主要ソースでは裏付けられていない。これは**ユーザーが示した重要な設計上の三分類**としてTEIへ導入を検討する。Buri／S2Coffee／外部デシジョンテーブル実装の役割分担を混同しない。今後の史料照合は追加の参照資料が得られた場合のみ行い、Buri旧環境の実動復元をGateにしない。

**これまで漏れていた本質**：図面を描けること以上に、ヒアリングで加筆訂正した**説明可能な宣言そのもの**が、型・対象・Rule・実装部品の対応を保ち、実装・検査・文書へ連続して引き継がれる点を、TEIの価値と評価対象にする。

### 7.2 三つの意味形態（閉じたCore enum／正式命名の決定ではない）

| 観点 | Validation／制約 | Derivation／導出 | Flow／進行・状態・手順 |
| --- | --- | --- | --- |
| 利用者が決めたい問い | **何が成立する／しない／未判定か** | **何から何を求め、なぜその値・分類になるか** | **何が契機で、次に何が起き／進められるか** |
| 表・図の候補 | 条件・判定表、制約マトリクス、例外と診断一覧 | 算定表、決定表、依存グラフ、式・単位・丸め規則 | 状態図×遷移表、処理Flow、時系列、手作業／外部Actionの区別 |
| 共通の実装対象 | Semantic Subject/Scope＋Predicate/Validation Rule | Derived Attribute/Dataset/Function＋純粋Operator依存 | Procedure/Action、State Transition Candidate、Event/Trigger＋既存Capability |
| 入力と結果 | typed facts、complete/scope/epoch、satisfied/not-satisfied/unknown/error | typed inputs、精度・NULL・欠損・時点、値＋出所・診断 | State/Event/Guard＋Version、遷移候補・拒否・Effect Intent／Observation |
| Effect | 判定自体は副作用を持たない（ただし拒否を強制する場所は別） | 計算本体は純粋。表示用計算と保存済派生値を区別 | Flowの決定は副作用を起こさない。Actionの実EffectはProvider/Hostに委譲 |
| 主な診断 | 条件重複・抜け、矛盾、未知／不完全入力 | 循環参照、型・単位・精度、非可換な集計、欠損 | 未到達・欠落イベント・分岐競合、順序、再入、実行不能・Effect不確実性 |
| 主要既存Owner | #1129/#2507/#2514、既存Validation | #2622/#988/#2600、DerivedとFunction | #1125/#1444、Lifecycle v1、#1121/#1174 |

三つは**同じ定義へ複数Roleで重ね得る**が、意味・Effect・実行時点まで統合しない。例：`注文可能 = 在庫数 >= 注文数`を導出値とする、`注文可能`を受注確定のValidationとして参照する、Flowで`確定要求`が来た際に評価する、という三者のBindingは明示する。**Validation=trueはAuthorizationではなく、Derivation結果は常に現在値ではなく、Flowの遷移候補は実Commitではない。**

例として業務以外にも次が成り立つかを比較する：
- **UI**：`formCanSubmit`導出、フォーム入力Validation、Idle→Submitting→Done/ErrorのFlow。ボタン表示・enabledがHostによるwrite permissionに昇格しない。
- **通信**：`retryDeadline`導出、packet/attempt ID Validation、Closed→Connecting→EstablishedのFlow。旧ACK、timeout、ABAとEffect結果を分ける。
- **ゲーム**：入力・環境から`threatLevel`を導出、攻撃条件をValidation、警戒／運動RegionのFlow。Frame order・物理運動・乱数seedを固定した範囲で再現性を評価する。
- **組込**：センサから`overheat`を導出、運転許可の安全条件、Running→FaultのFlow。欠測/時刻/故障時の安全側出力を純粋なGuard成功と混同しない。

純粋ValidationとDerivationは共通Operator／Rule基盤を再利用しやすいが、FlowとEffectはTime・Concurrency・Targetが強く関与する。三分類を**すべての業務Ruleを覆う排他的分類**とはしない。Sourceの観測、業務上の命令、外部Action、独立Expected、Authorityは別の意味Role／証拠である。

### 7.3 ユーザーヒアリングの図表を「そのままシステム化」する最短の往復

目指すのは「手書きの会議図から即座に本番実行」ではない。**ヒアリングで理解された同じ定義を、再転記・再解釈せず、段階的に資格を与えて実装へ使えること**と定義する。

| 段階 | 一つの設計の扱い | 何ができる／何をしてはいけない |
| --- | --- | --- |
| H0 質問・共同編集 | TEI Modelerで図にノード・矢印追加、表に条件・導出・例外追記。SHOの文章とSource位置を対応させる | Draft／出典／発言者が未受理なら候補表示。自然言語・見た目の図を実行定義と見なさない |
| H1 構造化・診断 | 変更を既存Record／Graph／Expression＋Role候補に写像し、対象・型・Status／Event・入出力とRelationを検査 | Source span・Stable ID・意味候補・曖昧箇所を一覧にする。自動で穴を埋めない |
| H2 例示と合意 | 同じ表・図に、正常／境界／例外の具体例、独立Expected、Ownerの受理対象/範囲を紐付ける | 説明用表、AI案、受理済規則、独立Oracleを区別。整合性チェックを意味受理としない |
| H3 生成・検証 | 受理済みMeaningをLanguage/Operator/Provider/Target固定とともに必要なGo/Host Library・SHO設計文書・Conformanceへ派生 | ModelとDocだけ再生成して済む変更とTarget再Buildが必要な変更を区別。生成成功を実運用可能としない |
| H4 実現・承認 | 実Consumerで利用したArtifact、現在Provider／Authority、保存Effect、独立Read、レビュー済み文書の配布Revisionを照合 | 人手受理、外部実行権限、テストEvidenceを混ぜた単一`approved`を作らない |

**共同編集の最小操作**：一つのState/Rule/Derived Attributeを選択すると、図・表・説明・Source・Example・Expected・Generated targetの対応箇所へ移動できる。図のedge削除→遷移表の一行削除候補、表のGuard変更→図の該当edgeハイライト、導出式変更→利用するValidation／Flow／Targetの影響一覧、といった**差分PlanとPreview**を出す。曖昧な逆写像、非可逆集計、循環Dependency、未取得Required Factは自動確定せず診断する。

**訂正の粒度**：文言・配置・コメント、例示値、独立Expected、Guard／Derivation式、Rule／Role版、Provider Binding、Target Templateの変更を別扱いし、意味変更を伴わない表示修正から実行時の再承認を自動要求しない。逆に式の1字修正がEffect/Guardへ波及する場合は必ず影響閉包を示す。非対称な図↔表編集は唯一のAuthoritative SourceへPatchとして記録する。SHOは同じモデルから説明・図表・改版履歴を生成し、承認済み納品文書のRevisionを保つ。

### 7.4 検証を既存Conformanceへ割り付ける

以下は**新Runner／新テスト正本を増やすものではない**。既存[#2620](https://github.com/hat47x/tei-platform/issues/2620)／[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#2497](https://github.com/hat47x/tei-platform/issues/2497)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)に割り付ける受入圧力であり、現時点では全て**NOT RUN**。

| 既存領域 | 最小の反例・独立Expected | 反証する誤成功 |
| --- | --- | --- |
| #2620/#988 | 同じ入力項目を、Validation表・Derivation表・Flow表の別Roleで解釈。Type/Operator lock・出典を揃えた範囲だけ意味に束縛 | 同形表だから同じ意味／表があるだけで実行可能 |
| #2497/#1124 | 遷移図のedge T2と遷移表T2を相互選択。重複edge・複数Guard・折り畳み表示・曖昧削除を検査 | 図が一致して見えるからSourceが一致／複数遷移を1セルに潰す |
| #2621/#1803 | ヒアリングで説明文・表・図を変更し、Source→Meaning→SHO文書と生成Bundleを版別比較 | レイアウト修正だけで全Meaning変更／式変更後も旧成果物のまま |
| #2514/#1131 | 計算の独立Expected、Validationのfalse/unknown、Flowの分岐差分を別Oracleで採点 | 計算表から自動生成した値を独立Expectedとする |
| #1121/#2600 | あるDerivationの変更で参照するValidation/Flow/Go Artifactのみを再資格化。RuntimeのRead/Grantは別判定 | byte同一＝意味同一、原因の異なる旧Receipt再利用 |
| #1129/#1174 | 会議で定義したFlowのActionが、現在Host Authorization／実Provider書込み／Receiptを伴って実行できるか別検収 | accepted diagram＝本番実行許可／Provider ACK＝業務完了 |

**Gate**：最初は一つの「入力値→導出→Validation→状態遷移」だけを、図・表・Markdown説明・独立Expectedの四Projectionで同じSource/Meaning IDとして追う。次に非業務UIまたは通信の異質Domainで同じ意味Roleを使う。続いて再編集・再生成と必要なTargetを検証し、初めて実Provider／HostのEffectに進む。**ヒアリングで即編集できる利用価値**と**正しい実装を生成するConformance**を独立に評価する。


### 7.5 添付されたDecision Workbook調査との照合（2026-11-03）

[Excel決定表調査のTEI適合レビュー](DECISION_WORKBOOK_INTAKE_REVIEW_2026-11.md)を追加した。WorkbookにはValidation 3件／Derivation 3件／Lookup 1件／Flow 4遷移／Test 14件があり、図・表・Matrixを同じIDで追う**説明用Profile**として有用。ただし汎用Decision Table Engine／Sourceの正式受理／実TEI評価は証明していない。

- `positive`の自由比較演算子、`derivationref`の`lookup`、`date→year`、Flowの「初期1・終端必須・(Lifecycle,From,Capability)一意」は、現行Semanticで受けられる範囲／簡易業務Lifecycle用Profile／非対応を分ける。
- 一般的なUI・通信・ゲーム・組込のState MachineをWorkbookのLifecycle表に無理に押し込まない。Event Payload、Time、Region、Guard overlap、SafetyをDomain別Pressureとして維持する。
- Excelを唯一のAuthoring Sourceとも`.tei`を全Domain唯一のSourceとも固定しない。Managed Sourceと受理済Meaningから、Excel・Studio・Markdown・SHOの図表と可逆な編集提案へ渡す。
- 添付のlint 0/0はWorkbook生成ツールの報告。独立Expectedの実評価、Runtime実行、Owner受理への昇格はしない。

### 7.6 仮想Attribute起点のFlow／Batch（2026-11-03、設計追補）

[仮想状態・Data Script・Trigger・Pipelineの設計／反証計画](DATA_DERIVED_TRANSITION_TRIGGER_PIPELINE_2026-11.md)に、**「実データ変更／導出値の変化→Flow起動→UIなしJob」**の要求と検証境界を追加した。

- Data ModelはVirtual/Derived Attributeの存在・型・対象・read-onlyを宣言し、Data Script側は純粋な値導出と依存関係を定義する。Flow側がそのAttributeの`changed false→true`等を明示的に購読する。意味宣言とTrigger Bindingと実Effectは別。
- `physical write`／`derived edge`／`level condition`／`domain commit`／`timer`を区別する。導出値を再計算できても過去の変化の発生・時刻・一度限りのJob起動は保証されないため、Before/After・Read一致・Event Identity・Idempotency/Receiptを要件に応じて別検収する。
- 同じData/Statusを重複保存しない原則を強化する。ただし実行進捗・外部通知・手動取消・Actorの承認など、計算だけで復元できない**独立した事実**は保存が必要。Materialized View/CacheとBusiness Truthは区別する。
- Flow中心の機能構築はData/Rule/Function/Capabilityの既存契約の**異なる合成入口**として扱い、新たな万能Scheduler/Workflow Hostを先行採用しない。#1130（Batch）／#1129（Effect phase）／#2622（Derived）／#2582（依存）へ渡す。

### 7.7 TEIとEKIの状態機械は同じRuntimeではない（2026-11-03、Owner判断）

[TEI→EKI限定委譲とSpring Statemachine比較](TEI_EKI_JOB_DELEGATION_BOUNDARY_2026-11.md)§8で、**ステート管理に基づくJob/Flowの定義・進行判断は基本TEIの責務**と整理し直した。TEIは必ずしも常駐しないため、生成Application／既存Host・OS scheduler・EKI・他Providerなどから、個別Deploymentに適した起動・進行主体を明示選択する。EKIが必要なのは主にTask/Attempt/Lease／Result/Effect/Recoveryの技術的な状態遷移と、部分委譲した場合のTimer・遅延Event・限定Step依存である。

Spring Statemachineの階層、直交Region、Fork/Join、History、Deferred Event、Timer、永続化などはEKIの複雑な分散Job運用に参考となるが、TEIの標準DSL/Coreへ一括輸入せず、EKIの既存独立状態軸・CAS/fencingで不足が実証された場合にだけ比較する。Spring Statemachine本体は2026-07-05に公開Repoがarchivedとなったため、直接依存でなく概念・反証の参照とする。**EKIがRich State Machineを採用した＝TEIの業務Flowを解釈してよい**、とはしない。

### 7.8 EKI資源共有の状態機械をTEIで検証する（2026-11-03）

[EKI Resource/Node運用の可視化・Simulation・AIレビュー計画](TEI_EKI_RESOURCE_OPERATIONS_SIMULATION_2026-11.md)に、**EKI自体がResource/Node/Local Governor/Task/Lease/Residentなど複雑な継続状態を安全に管理する**という実例を追加した。TEIはEKIのApplication PlaneのGraph/表/Time-series、状態Role/Rule、Simulation Scenario、生成AI Critical Review、変更Previewと運用Function生成に用いる。ただし**EKI Task State Authority・Lease・Local Governor/Providerの実権限をTEIの意味候補やAIによって上書きしない**。

TEI CoreでSpring Statemachineの全機構を先行採用するのではなく、EKIの既存Task/Attempt/Lease等の独立State軸に、Graph×Transition/Invariant表×Event Traceを投影できるかを評価する。EKI側でDeferred Event/Fork/Join/History等が必要ならEKI固有Conformanceへ引き渡す。GPU shared memory・VRAM・CPU/Memory予約／OS保護を一つの「使用率」列に潰さず、Simulationの計算結果と実Device観測・Authority Decisionを分離する。

### 7.9 Cloud/Edge DeploymentとState Ownerの分離（2026-11-03）

[TEI Application→EKI論理Launch Target／Cloud Control＋On-prem/Pi Fabricの比較](TEI_EKI_APPLICATION_DEPLOYMENT_FABRIC_2026-11.md)を追加。State/Flowの意味はTEI、常駐ApplicationのRuntime/Service readinessはCloud／既存Orchestrator、Task/Attempt/LeaseとResource/Node stateはEKI、DB/Business EffectはDomain Providerが保持。**どの管理コンソールに登録したか**ではRuntime状態の所有者は変わらない。Web/APIの可用性とPi/PCの機会的Task実行はState Machineの実行Profileとして区別する。Release Intent→Provider Actual→Consumer Observation／現在資格の差を、Graph×Tableの相関表示と独立Expectedで検収する。


### 7.10 Data ScriptのIFからDecision Table／State Graphへの投影資格（2026-11-03、IF-first方針の反証計画）

<!-- IF_DECISION_TABLE_PROJECTION_20261103 -->

**位置づけ**：[TEI #1129の「Data ScriptのIFを基本とし、状態機械は保守性が増す範囲だけ選択」](https://github.com/hat47x/tei-platform/issues/1129#issuecomment-6588906800)と§7.5のDecision Workbook（Validation/Derivation/Flowの提案・レビュー面）を接合する。IF→Decision Table／State Graphを原文の自動置換ではなく、まず**同じ受理済み意味のread-only Projection**として検証する。結果値が一致しても、評価順序・短絡・未取得・診断・副作用・実Readの差を無視した意味同値は認めない。Excelのdraft行や「合格」ラベルをAuthoritative Sourceの編集権限・Owner承認・実行資格へ昇格しない。

**変換の資格**：

| 段階 | 扱い | 必要な証拠・禁止事項 |
| --- | --- | --- |
| **P0 非実行の投影** | 元Source occurrence、byte span、文法/Revisionを保持したIF→Rule/Transition Table・Graphの表示候補 | Else、優先順位、重複Match、未解釈構文、Source位置、Loss/Unresolvedを保存。元Scriptに書戻さない |
| **P1 条件付き同値** | 同じInput/Context/Role版に対し、IFと表が何を保存できるかの反証 | Resultだけでなく、Error/Diagnostic、評価順序、Short-circuit、実Read、公開可能な理由、Effect Phaseが同等である範囲を検証。副作用があるGuardは純粋な論理表に勝手に配置しない |
| **P2 明示的編集Intent** | 表・Graphの編集から元Sourceの局所Patch候補を提案 | 対象Subject＋Source occurrence＋Source revision＋UTF-8 spanの一意性、影響解析、変更対象外byte不変、Ownerの受理を別に確認。#2621/#1124/#1935の既存境界を再利用 |
| **P3 実実行／Effect** | Host/Providerが現在の権限と条件を再検査して実行 | old/new値、Rule Phase、Tx/CAS/Idempotency、Receipt、独立Readを区別。表のValidationの成功をCommitやJob実行成功と呼ばない |

**固定小例（正式TEI DSLではない）**：受注提出時に「①statusがclosedならreject_closed、②quantityがNULLならunknown_qty、③quantity<=0ならreject_qty、④それ以外はallow」を順次IFで評価する。closedの行でquantityがnot-fetchedの場合、短絡したIFはquantityを読む必要がないが、表が全条件を先に評価するとRead/errorが増える。closedかつquantity=-1なら条件が二つMatchしてもIFは最初のreject_closedを採用する。順序指定のない決定表を同値と扱えない。

| ID | 比較Fixture／独立Expected | 既存Owner |
| --- | --- | --- |
| **DT-01 First-match** | closed & quantity=-1：IFは最初の拒否。無順序の2行Matchを同じ判定と誤認しない | #988、#1129 |
| **DT-02 短絡・非出力Read** | closed & quantity=not-fetched：IFはquantityを読まない。Eager表が読んでLookup errorを起こす場合、実行同値ではない | #2622、#2497 |
| **DT-03 NULLと未取得** | draft & quantity=NULLは値の観測、not-fetchedは未観測。どちらもfalseにして一致させない | #2622、#2514 |
| **DT-04 拒否Reason/Diagnostic** | Resultがrejectでも最初のReason・発生箇所・公開可能な情報が違う場合は、診断や利用者説明の同値未成立 | #1129、#1649、#2515 |
| **DT-05 Edge vs Level** | old=draft→new=submittedの変化とcurrent=submittedを分ける。導出再計算は履歴Eventの実発生と同義でない | #2582、#1129、#1130 |
| **DT-06 同時更新** | 同じ旧revisionから2回submitする場合、CASで確定できた回数以上のEffectを作らない。2行Matchは実Commit2回を意味しない | #1129、#1174 |
| **DT-07 Source編集** | No-op Saveはzero write/byte一致。1セル編集は元ScriptのOccurrenceが一意に特定できる場合のみ局所Patch。周囲のコメント・書式・構文糖を崩さない | #2621、#1124、#1935 |
| **DT-08 別Domain** | UI/通信/ゲーム等のEvent priority・timeout・並行Regionに業務Decision WorkbookのLifecycle一意制約を押し付けない | #2497、#988、#2514 |

**証拠訂正（2026-11-03）**：前版の「Python標準unittestで8件PASS」という記載は、当該時点の実行ログ・スクリプト実物を確認できないため**撤回する**。報告された8件を実行済みのProbeとして扱わず、この文書での結果は**設計上の反証候補、NOT RUN**とする。いずれのDT-01〜08も実Workbook、TEI Parser、Studio、Go/Java Target、実DB/Provider/Owner意味受理での製品試験は**NOT RUN**。今後PASSと記録する場合には実行したSource/Expected、固定入力、Tool/Version、Command、Exit、出力、git commit/test blobの可検証な証跡を保存する。

**最短の接続順**：①既存Source/WorkbookでIF→表のread-only行・Source位置を照合、②DT-01〜04で順序/Read/欠測/診断の違いを固定、③DT-05〜06を#1129/#1174実Effectへ、④DT-07は#1935 Shared Acceptanceのupdate演算が実証された範囲だけに限定、⑤DT-08で別Domainへの強制的な意味拡張を防ぐ。これを先に新しいDecision Table EngineやState Machine Runtimeを作る口実にしない。Source正本・Meaning Authority・Provider実効果・文書投影のOwnerを重複させない。対応不能なIFはLoss付きread-onlyまたはunsupportedに留める。



### 7.11 入力起点と実行先をData Scriptの意味から分離する：Function Invocation・Interaction・現在Targetの縦断（2026-11-03）

<!-- tei-20261103-invocation-interaction-binding-crosslane -->

**他レーンの成熟判断と本計画での新しい差分**：

- [RMT-2一試行計画 Draft #2783](https://github.com/hat47x/tei-platform/pull/2783)は「二入口・一つのData Rule・一Owner Write」をE0〜E6で比較し、PA-11〜14で反例集合・実Observer・Trigger/Flow・EKI Body/Deadlineを区別した。ここでE0〜E6の別正本を作らず、**同じFunction/Actionの起動入口とInteractionが、実行対象・意味・権限を汚染しないか**という画面/Job/CLI境界を特定する。
- [Data Script/状態の方針 #1129](https://github.com/hat47x/tei-platform/issues/1129)はIF-firstの単一Meaningを維持し、Function/Flow図はProjection／Bindingとする。[Action Surface Draft #2616](https://github.com/hat47x/tei-platform/pull/2616)と[実Command Target漂移 #2598](https://github.com/hat47x/tei-platform/issues/2598#issuecomment-6601453060)には、表示/入力側の受注O1に対し実RequestがO2を更新し得る**限定Go特性試験**がある。これは適格拒否が実装できた証拠ではない。
- [組織由来の推移的可視性 Draft #2781](https://github.com/hat47x/tei-platform/pull/2781)は、表示・read-only・例外・deny・期間・Scopeを分けた合成33ケースの提案であり、Provider/Hostの実効認可ではない。**「画面に見える」こと、Data ScriptのValidation成立、実Actionの書込Grantは別契約**。
- [非常駐JobとEKI委譲計画 #1130](https://github.com/hat47x/tei-platform/issues/1130)はTEIがJob意味を持ち、EKIにTask/Attempt/資源・期限を必要に応じ委任する。ボタン押下もTimerも「Script専用Host」起動の別言語仕様にせず、型付きFunction/Action Invocationへ正規化する。

**唯一の業務Ruleを共有し、入口の違いを隠さない**：

| 契約面 | 固定する意味・証拠 | 起動形態による違い |
| --- | --- | --- |
| Data/Rule | Order Subject、型/粒度、IF評価順、必須Read、State/Validation/DerivationとRule版 | Web/CLI/Batchが別にRuleを再定義しない |
| Function/Action | Purpose（submit/cancel/correct等）、typed parameters、対象集合とScope、期待するBusiness Effect | 一つのBusiness Functionを複数起点から呼べるが、異なるPurposeを同じupdate操作に圧縮しない |
| Invocation Trigger | user button、API、CLI、Timer、Data Eventなどの起動契機、Actor/Correlation/Idempotency | Triggerは受付契機であり現在Authorizationや実Data Eventの証明ではない |
| Interaction | 型付きInput要求、確認/選択/進捗/取消の必要能力、失敗時の応答Contract | Web/DesktopはInteraction Providerを利用し、CLI/Headlessは事前Inputを与えるかinteractive-requiredを返す。UI待機をData Scriptの純粋Ruleへ埋め込まない |
| Request/Effect | Target Subject/Provider Mapping、Grant、ExpectedVersion、Payload、Period、Commit/Replay/Receipt | 各入口で同じ最終Owner Resource/Meaningへ束縛する。TEIのPreview/参照ExecutorのCommittedは実DB Commitではない |
| Async/Delivery | Resource予算・期限・Job Instance/Attempt、Body/Result/Owner Verified | EKIや外部Hostの実行完了から業務上の成功・納品・Rule現在性を推論しない |

**最小縦断ケース（新規Harness/Engineではなく既存PA/SJ/OBの横断照合票）**：

| Case | 具体的な反証 | 期待結果とOwner |
| --- | --- | --- |
| **IV-01 二入口一Rule** | Webの提出ボタンとHeadless CLI/APIで同一Order O1/同一Rule版を呼び出す | 入力値/Rule相対Expectedと業務拒否理由は一致。ただしActor/Grant、Interaction、結果の提示は別。#1129/#2783 |
| **IV-02 Interaction不足** | Functionは利用者の追加確認を必須とするがHeadless実行へ確認済みInput/Providerがない | 明示的interactive-required／不受理。確認なしで既定承認しない。#997/#1130/#2502 |
| **IV-03 Purpose/Target漂移** | 表示/選択はO1のsubmit、PayloadはO2のcancel、Provider Aliasも複数 | Request構築または実Commit直前に現Owner/Grant/Target不一致を拒否する必要がある。現行のGo負例PASSは拒否の製品PASSではない。#2598/#1174 |
| **IV-04 組織Read-only** | O1は可視だが部門Scopeがread-only、または組織関係の部分Readで授権経路不確定 | 表示許可からMutation Grantを導出しない。実Host/Providerが現在のScope/Permission/Policy版を検査。#2386/#2781/#1174 |
| **IV-05 Period/Revision** | Actionを提示した後に期間がclosing/closed、Rule/Source/Grantが改版 | Offer/PreviewをExecuteの現在資格に転用しない。必要な現在状態が不明ならfail closed。#2598/#2784 |
| **IV-06 Virtual Edge疑似イベント** | SQL Schema/Mapping改訂によりDerived Fieldの表示値が変わるが、Business Write/Commitがない | UIの変更表示・Derived Edge候補からOutbox/Jobを発火させない。#2387/#1129/本Draft Data-derived §7.4 |
| **IV-07 Effect不明・Replay** | Commit後ACK-loss、Replicaでは旧値/0件、再SubmitボタンやTimer再送が発生 | 新OperationIDの二重Commitに進まず、既存Receipt/Idempotencyと因果Readを照会。#1174/#2598 |
| **IV-08 非常駐とEKI** | GUI停止中に登録済みJob/Timerが発生、EKI Task.completedだがBody/業務効果未確認 | Deploymentで選んだ実Coordinatorの起動を検証し、Task/Artifact/Business Verifiedを分離。TEI独自常駐Hostを追加しない。#1130/EKI #112 |

**最短の実装順・止めどころ**：まず公開合成Order 1件について、元Data RuleとFunctionの意味・Source位置を固定して同じInvocationをHeadless二入口から呼ぶ（#2783 PA-01/13を再利用）。次に現在のGo Action Surface/Command AdapterへIV-03とIV-05の**意図した拒否を独立Expected**として与える。ここでSourceを誤参照/対象違い/Grant欠測なら業務Effectを止める。実Provider（初回Owner Write/ACK-loss）を通過した後で初めてWeb/Desktop Interactionと、オプションのEKI Task/Artifact/Deadlineへ拡張。図表での「Action可」ラベル・Decision Tableのallow・Rule CheckSetの0違反・EKI Task.completed・Probeのexit=0は、正式Commit/Owner Completedを代替しない。

**採否・証拠の状態**：これは既存Source/Issue/他Draftの結合計画であり、新しいTEI API/Interaction Plugin/Job Scheduler/Rule正本を実装した記録ではない。IV-01〜08の同一Attempt Web/CLI/Provider/Host/EKI E2E、実組織Policy enforcement、実Command拒否、元X-TEAとの同義検証、Owner受理はすべて**NOT RUN／NOT ACCEPTED**。既存の#1119 Gateを引き上げない。


### 7.12 11月7〜8日横断：Raw Item・Role・状態図からDesign Context／Plugin実呼出し／Provider実観測へ（計画、未受理）

<!-- 20261108-state-raw-role-inspector-context-handoff -->

**11/3方針に対する成熟差分**：Data ScriptのIF-first／決定表・状態図は意味保存Projection／InvocationとEffectを分ける構成は維持する。その上で11/7の「Everything is Data」「未整理Item/複数Canvas」「原子的Trait」「Core以外Plugin-first」「必要設計情報の完全性」「三視点図/CRUDと実Provider Read」を、同一構造の**異なる資格段階**として扱う。新しい巨大Core Kindや第二Workflow Engineを追加しない。

| 段階 | 保持・確認する事実 | 自動的に昇格させないもの |
| --- | --- | --- |
| **C0 Raw Source/Item** | 未整理Text/Markdown/外部Reference/添付の保存、Content bytes・Occurrence・Revision・Scope、Canvasの配置・分割/参照 | 保存・表示できたことからParsed/Accepted/Runnableを推論しない |
| **C1 Structural/Role Candidate** | 実際に選択したSyntax/Language PluginでのRead/Parse/BindとSource位置。ItemにTask/Action/Event/State/Transition等の拡張Role候補を対応付ける | LabelやGraph Edgeから実Business Transition、Committed Event、Task実行資格を作らない |
| **C2 Accepted Meaning/Model** | Owner受理済みData/Function/Business/Rule、Subject/Role/時点/Scope、IFとDecision Table/Graphの意味保存条件 | 可視化の同値・同名・Trait宣言・AI提案をOwner MeaningやGrantにしない |
| **C3 Composition/Run/Effect** | Required Trait→Provider選択→Load/Link→実Call→CLI/Studio Consumer、現在Host/Provider Grant、Write/Receipt/独立Read。外部EKIはTask/Attempt/Lease/Bodyを自分で管理 | 「build成功」「selected」「Reference Committed」「EKI completed」「結果0件」だけで実業務完了としない |

Itemは未整理の情報を乱暴に保持してよい一般の単位であり、Task/Action/Eventを固定した物理レコード分類にしない。Roleは組み合わせ・拡張可能。ただし **Event事実とEvent発火条件、Action Intentと実Effect、TEI上のTask RoleとEKIのTask IDは区別**する。同一Itemを二つのCanvasへ置いても、二つの業務Eventを発生させない。Canvasの位置変更・リンク削除はEntity削除/Relation削除/Business Transitionではない。

**現行の確かな再利用資産と未達**：
- [merged #3277](https://github.com/hat47x/tei-platform/pull/3277)：Plugin-firstの設計/製品要求はmainへ反映。現行CLI静的dispatch、Studio固定Bridgeから共通Inspector Contract/Providerへの実接続は残る。#951/#1251/#1158の既存Resolver/Providerを使い、第二Inspector Engineを作らない。
- [merged #3354](https://github.com/hat47x/tei-platform/pull/3354)：限定された実CLI 11 Reply／32 Consumer Controlsは、独立Required 3 RuleとCanonical/Sourceの欠落を検出する有限な証拠。全Source/Function/Business閉包、単一原子的Snapshot、実Provider Read、外部Harness完全受領、AI盲検新規発見の証明ではない。#3262のRequired/Observedへ接続する。
- [merged #3353](https://github.com/hat47x/tei-platform/pull/3353)：空TargetsでHandoffConfirmed=falseなのにCompletionHoldsとなる不整合を局所修正。空集合が正当でも、別の主体の引継ぎ受領証拠は独立。Caller申告と外部相手/実Provider Receiptを混同しない。
- [#3278/#3279](https://github.com/hat47x/tei-platform/issues/3279)：Atomic TraitのDefinition/Requirement/Binding/Fulfillment/Evidenceは設計候補。manual-required未実装、Plugin契約版不一致、宣言のみで実Load/Invokeが無い場合はunfulfilled/unverified。現在Grant/Owner受理はさらに別。#3267 BOMにも実Target別Evidenceが必要。
- [#3271](https://github.com/hat47x/tei-platform/issues/3271)：派生Snapshot/Cacheの再生成・失効は確定業務履歴やCommitted Eventではない。Mapping/Schema/Cache版変更でVirtual Valueが動いてもBusiness Write/Outbox/EKI Jobを自動発火しない（本Draft DS-01〜08を再利用）。
- [#2497](https://github.com/hat47x/tei-platform/issues/2497)/#1124/#1125/#2586：Canvas・State Graph・Transition Table・三視点図・CRUD・SHO説明を元Subjectへの複数Projectionとして扱う。Loss/逆写像不能ならread-only。実Provider行読取は別の実証。

**追加の独立Expected候補（CR-01〜08、全件新規NOT RUN）**

| ID | 反例 | 期待する結果と既存Owner |
| --- | --- | --- |
| CR-01 | 未整理/未知RoleのRaw Markdownを保存 | Sourceは保全・閲覧可能、MeaningとExecutionは未資格。#2621 |
| CR-02 | 同一Itemを二Canvasへ配置、片側Layout削除 | 元Item/Entity/Relation/Business Eventは消えない。Action二重発火なし。#2497/#1124 |
| CR-03 | 説明TextにAction/Task/Eventという語がある | 解釈/Owner受理なしでは実Action/Eventへ昇格しない。#2620 |
| CR-04 | JSON/FP正常でもRequired RuleやSource Anchorが未観測 | #3354の限定実Consumer相当ではreject。全Domain閉包の主張なし。#3262 |
| CR-05 | Inspector宣言は選択済みだが実Provider別、manual Trait欠落 | selected≠loaded≠called≠fulfilledを検査。#951/#3279/#3277 |
| CR-06 | Schema/Cache改訂でVirtual Fieldがfalse→true、業務Writeなし | Business Event/Outbox/EKI Jobは発行しない。#1129/#3271 |
| CR-07 | Handoff対象0件、完全性正当化済、受領未確認 | Completionはunverified、独立Handoffを別観測。#1125/#3353 |
| CR-08 | Studio Reference実行成功、EKI Task.completed、Bodyや実Effect不明 | Task/Artifact/Owner Completedは別。#1130/#1174/EKI #112 |

**着手順（既存#1119 Gateを増やさない）**：P0-Mでは一つの公開合成SourceのRaw Item/Accepted Rule/Business ProcedureをGraph・表・CRUDへread-only投影。P0-Dでは#3262/#3354で独立Requiredと実ReadのGapを照合し、partialを維持。P0-Pでは#3277の最初のInspectorを既存Resolver経由で選択→実Load/Call→CLI/Studio同じConsumerへ渡すが、未接続先はNOT RUNとする。P0-Rでは[#2586](https://github.com/hat47x/tei-platform/issues/2586)のorder_entry.recent_orders一Function/一Queryを実Provider Readへ接続し、Rowset/Scope/Completeness/経路を独立観測。P0-Eの書込/Jobは、現在Target/Grant、Tx/CAS/Receipt、Handoff、EKI Body/Owner完了の別資格が揃うまで実行成功と主張しない。

**非採用/保留**：Canvas固有の必須Core型階層、固定Task/Action/Event enum、Raw Textの自動意味受理、中央Trait Resolver/BOM/Host/新認可Engine、全Domain共通State Machine Runtime。複数Canvasの実UI・協働編集、正式Role Syntax、CLI/Studio共通Plugin実呼出、実Query Provider、Owner受理は本追補では未採用/未実証。

**Evidence ceiling**：GitHub現行Issue・main統合PRの限定証拠を引用したdocs-onlyの横断設計。CR-01〜08の実製品試験、Studio/Web/SHO/EKI E2E、Provider/Host実行、CI、正式意味受理はこの変更で**NOT RUN**。過去PRが証明した限定試験を本PRで実行したとは記載しない。

### 7.13 SUI探索・REA原典解析からTEIモデル、SEI判断へ至る「翻訳と決定権限」の分離（2026-11-08、既存契約の受渡し）

<!-- 20261108-crosslane-sui-rea-sei-decision-handoff -->

**今回の確定的な設計制約と新しい接続点**：
- SUIの探索/Canvas/Item/関係候補はTEIの受理済み業務意味へ**無審査で自動昇格させない**（未受理CandidateをTEIへ直接渡すAdapterは許容）。人間または生成AIが出典と未決を見ながら翻訳してTEIへの入力候補を作り、TEI側のParser/Role/Owner Acceptanceを経る。SHOは必要時の文章化・出典保存・Reviewを担うが必須の中継Hostではない。Document/Item/Graph/Canvasの共通コンポーネント利用は可能でも、SUIの解釈やTEIの業務意味を同一DBレコード・同一ID・同一Authorityへ統合しない。
- 既存システムの解析は[TEI #3394 REA入場](https://github.com/hat47x/tei-platform/issues/3394)で外部Provider Candidate/Observationとして扱う。REA保存Evidence・選択View・SQL/旧DBの観測範囲を元Sourceと別に記録し、静的なPotential CRUDをRuntime Commit事実に昇格しない。旧画面に「実行可能」とあることも、移植先の許可されたActionを証明しない。
- [TEI #3475 Ontology/Decision Twin](https://github.com/hat47x/tei-platform/issues/3475)は、**今はO0設計研究と既存契約へのFit/Gapのみ**。Ontology DSL/SDK・動的Twin・予測/シミュレーション・実SEI委譲の製品化は後続OwnerのGateに従う。TEI全体セルフビルド #2831とSUI基本機能の実受入前に、O2/O3の開発を本PRから起動しない。既存P0業務縦断やGo単一バイナリ基本構成を遅らせない。
- 判断の主体は原則として**SEI**に置く。[SEI #677](https://github.com/hat47x/sei-cognition/issues/677)のGoal/Constraint、異論/未決、Knowledge照合、用途別受渡しを尊重する。ただし人間が受理した決定的Data Ruleで完結する通常のValidation/単純処理にSEI照会を義務化しない。新しい裁量判断・相反する目標・追加知識を要する場合はTEIが**Decision Inquiry候補**を作ってSEIに判断を依頼する。Inquiry作成/送信/ACK/SEI受領/判断/Host実認可/実Effectはそれぞれ別の事実。
- 神経反射に相当する現場即応は**あらかじめOwner/実Hostが認めた対象・作用範囲・期限・Fail-safeを限定した手続きのみ**。通信断を理由に権限を拡大しない。現場の実行結果/Receipt/観測を後でSEIへ伝えても、後送ACKだけで判断や正当性を遡及承認したことにはならない。EKIが資源/Taskを実行しても、TEIの業務RuleやSEIの意思決定主体をEKIへ移譲したことにはしない。

**一つの対象を通す時の所有権と証拠**（新公開SchemaやMessage Transport実装を決めるものではない）

| 相 | 最小入力・証拠 | 判断/Stopと既存Owner |
| --- | --- | --- |
| S0 Explore | SUI Item/Canvasの元Source/Revision/Occurrence、出典、保留・異論、部分的Relation | Itemの表示/連結≠TEIの正式Entity/State。SUIが探索・保存を所有 |
| S1 Observe | 必要ならREAの実Raw Evidence、実Read/解析Profile/版/coverage、旧Code/SQL/DBの時点 | 静的Query候補やSource Snapshotを外部Commitとしない。TEI #3394、#3270 |
| S2 Translate | 人間/AIによるIdentity・Field/Relation・Rule・Actionの翻訳案、元文脈、loss/unknown、出典範囲 | **直接Auto-Promote禁止**。TEI #267/#2620/#2497/#1124のMeaning/Binding/SourceへCandidateとして提出 |
| S3 Accept & Inspect | Ownerが受理したScope/Rule/State/Version、Function必要ContextとSourceのRequired/Observed差分 | #3262/#3354の限定CLI等で必要なSliceが揃わない場合、完全モデルと呼ばない。業務意味の採否はOwner |
| S4 Decide | 既知の決定的IF/Guardなら現在Rule適用。裁量を要する場合はSEIへのInquiry（Question/Goal/Constraint/Evidence/Unknown/Correlation） | SEI受信ACK≠判断済み。旧Rule/Source版・非開示・相反条件なら再確認・保留。SEI #677/#767、TEI #3475 |
| S5 Execute | SEI判断または受理済みRuleに基づくAction Intent、実Target/Provider/Grant/CAS、必要なら限定EKI Task | SEIが決定しても実Hostの現在権限/Tx/Receiptは別。実Effect UNKNOWNでは無条件再試行を避ける。TEI #1174/#1129、EKI #112 |
| S6 Explain & Reconcile | 実Effect/独立Readと不明範囲を元Source/SHO説明/SEI判断履歴へ照合 | Model更新、説明文更新、Canvas更新、業務Event、SEI Memory更新はそれぞれ独立。遅れた旧応答で現在版を上書きしない |

**既存Projectへ非重複で返す反証（SH-01〜08、今回新規実装はNOT RUN）**

| ID | 合成圧力 | 独立Expectedの要点 |
| --- | --- | --- |
| SH-01 | SUIで関係を結んだがRule/Owner/元Source範囲が不明 | TEIへ翻訳Candidateのみ。Graphの操作で業務状態は変わらない |
| SH-02 | 同一Itemが2 Canvas、SHOに別引用Occurrence | Source identityと表示/引用Occurrenceを区別。重複Decision/Actionなし |
| SH-03 | REAは一Source解析成功、別DB時点未取得／旧Runtime未観測 | 外部候補の範囲のみ。TEIモデル・旧新同値・業務実Commitを成功としない |
| SH-04 | REA/SUIが同じ名前の別Entity、時点・Scopeが異なる | Stable ID/Relationの自動結合禁止。翻訳/受理までunresolved |
| SH-05 | 機械的Ruleは判定可能、別件は目標間で衝突 | 前者はSEI必須化せず既存IF Guardへ、後者はDecision Inquiryの候補にしてSEI応答待ち。必要なAuthorityは別 |
| SH-06 | Inquiry送信ACK後にRule/観測が改版、旧版でSEIがallow | 古い回答を現在の実行資格として使わず、再照合/保留 |
| SH-07 | 通信断で現場の停止が必要だが、反射Profileの範囲外 | 実Hostでfail-safe/明示拒否。許可済みの狭いProfileは効果と実Receiptを別に観測しSEIへ後送 |
| SH-08 | Twin風のScenario/仮想EdgeまたはEKI Task完了 | Simulation/Derived/Task/Artifact/業務確定の区別。Self-build/SUI基礎未達ならOntology Twin本格実装へ進まない |

**実行順と非目標**：現行のP0-M/P0-D/P0-P/P0-R/P0-E（§7.12）を維持する。まず公開合成のSUI探索記録と旧Source Evidenceについて、*人/AI翻訳Candidate → read-only TEI Data/Function/Business候補 → Source/Meaning必要条件の不足表示 → SHO任意引用*を**設計比較票**として評価する。TEI実Providerの1 Function/1 Readが未接続なら、SEIへの実InquiryやTwin運用を先に作らない。後続のSEI連携は実Decision Ownerの#3475、外部REAは#3394、SHO文書の実保存はSHO #4/#6のOwnerに戻す。第二Shared Registry/Workflow Host/Decision Engine/TEI内シミュレータや、SUI候補を受理済み意味へ無審査で昇格するImporterは提案しない。出典・未受理を保持する限定的なSUI→TEI直接Adapterは妨げない。

**Claim ceiling**：これは既存GitHub Issue/採用済み境界と過去会話を照合した文書上の実装順/反証計画であり、REAの実解析、SUIからTEIの実翻訳・Parser/Owner受理、SHO実Writer、TEI生成SDK、SEI実Message、EKI実Task、独立Provider Commit/Receipt、Ontology Twinはこの追補では**すべてNOT RUN／NOT ACCEPTED**。将来Gateの技術成功をOwnerの意思決定や本番の実権限へ昇格しない。


### 7.14 外部Evidence・意思決定Artifact・Transportの「到達証拠」と実権限を分ける受入票（2026-11-08）

<!-- tei-20261108-handoff-claim-witness-and-real-path-admission -->

§7.13でSUI/REA→TEI→SEI→実Host/EKIという**責務**を確定した。今回は「それぞれの境界で何が実際に到達・解釈され、何がまだ起きていないか」を同一Attemptで示す受入列へ絞る。新たなTransport、Decision Schema、Evidence Registryを実装する計画ではない。単なるメッセージ送信、静的Analysis、選択Plugin、テストFixtureのPASSを後段の成功へ誤伝播させないことを目的にする。

**現行の具体的な実装差分（各Issueの正本にのみ帰属）**：

1. **外部解析の二経路**：[TEI #3394](https://github.com/hat47x/tei-platform/issues/3394)はREAのRaw Evidence読取 **T1-A External Observation** と、専門家が選択・翻訳した候補をTEI Authoring Sourceへ投入する **T1-B** を区別している。標準Authoringの *.tei.md 方針は[main #3030](https://github.com/hat47x/tei-platform/pull/3030)で設計採用済みだが正式Reader/Parserの製品接続は未完成。REA JSONを拡張子だけ変えて正式TEI Sourceにしたことにはならない。T1-AはT1-Bを待たず実証できる一方、旧Runtimeとの同値はさらに別のT2である。
2. **必要設計情報の到達**：[TEI #3262](https://github.com/hat47x/tei-platform/issues/3262)/[merged #3354](https://github.com/hat47x/tei-platform/pull/3354)の限定11 Reply・3 Required Rule・32 Consumer Controlsは、読取り結果の欠落を検出する基線。完全なTEI Function/Business closure、atomic Snapshot、外部Harness配送まで証明したわけではない。実Source/Provider版とReplyの最終受領を独立に確認する。
3. **SEIに届くこととDecision recordになること**：[SEI #785](https://github.com/hat47x/sei-cognition/issues/785)のTAI由来Decision投影artifactは、現在のIssueでSEI内部契約・実使用箇所・Decision recordへの変換が未決。平坦なRecordにbasis_refs/scope等の必須根拠が無いまま「受理済みSEI Decision」へ自動変換することを禁じる。先行するDecision Inquiry自体も[TEI #3475](https://github.com/hat47x/tei-platform/issues/3475)のO0設計比較にとどめ、実Messageは後続。
4. **EKI Data Deliveryの基線**：[EKI #753](https://github.com/hat47x/eki-fabric/issues/753)は固定6 EventのReference意味と改変負例を[merged #785](https://github.com/hat47x/eki-fabric/pull/785)で試験したが、**Local IPC/Connectの実Data Adapter配送は未検証**。実Lease/Grant/Transport遅延・実Payload受領やTEI Business Receiptではない。通信方式の採否は[#754](https://github.com/hat47x/eki-fabric/issues/754)で実Adapter計測後とし、本PRから先行してWebRTC/Transportを選ばない。EKI外部Task Input/Result取得は[#112](https://github.com/hat47x/eki-fabric/issues/112)の独立した未解決Contractである。

**同一Attemptの「ClaimとWitness」の最小列**（製品Schemaでなく既存#1163/#3262の受入票に追加する比較軸）：

| 対象のClaim | Positive Witnessに必要なもの | 誤った代用品 |
| --- | --- | --- |
| 外部Sourceを観測 | 元Source cohort/byte・実解析Provider/argv/版・実Bundle readback・coverage/unknown | REA summary、JSONの名前変更、Tool exit 0のみ |
| TEI正式Authoringとして解釈 | 選択された *.tei.md byte/spanの正式Loader consumed→Parser/Role/Source Index結果＋意味受理の独立記録 | Markdownファイルが保存されている／旧 *.tei ReaderがPASS |
| 必要なDesign Contextを取得 | 用途・対象相対の独立Required集合とObserved Loader/Rule/Relation/Anchor、分割終端・実Consumer受領 | 同一Fingerprint、catalog件数、11 Replyの一部だけ |
| 判断InquiryがSEIへ到達し判断された | Inquiry scope/basis/unknown/Request版と、実受信→受理/保留→Decision Recordを独立観測 | TAI Projection JSON、sent ACK、単なるDecision文 |
| EKI上で実Data/Taskが配送された | 実AdapterのInput/Output byte/digestとAttempt/Lease epoch・Result retrieval・実Consumer受領 | Reference Test 7 PASS、Task admitted/completed、Fixtureのみ |
| 実業務Actionが完了 | 現在Grant/Target/ExpectedVersion、Owner write Tx/Receipt、独立Provider ReadとCompletion/Handoff | SEI allow、EKI Event fresh、Replicaの空結果 |

**反証候補（HC-01〜08、すべて今回の製品実試験NOT RUN）**

| ID | Pressure | 独立Expected／受入先 |
| --- | --- | --- |
| HC-01 | REAのRaw JSONを order.tei.md として保存 | T1-AのEvidenceを残すがT1-B正式Parse成功/Meaning受理を拒否。#3394/#2621 |
| HC-02 | 外部Source二件が別時点・一方partial、SHO文書だけ最新版 | 同一Business Snapshotとしない。文書Revision≠Source/Meaning current。#3394/SHO #6 |
| HC-03 | 11 Reply内のRule詳細が一件欠落、または外部Toolで途中切断 | #3262のRequired/Observed不足と受領欠落を区別し、完全なContext成功にしない |
| HC-04 | SEI向けArtifactにDecision文はあるがbasis_refs/scopeがない | #785決定前には正式Decision/採用済みMeaningへ変換しない。問合せは保留/未対応 |
| HC-05 | SEIにACKされた後にRule/Source版が変わり旧allowが返る | Reply受領は記録するが現在Grant/Owner writeを禁止し、影響分だけ再審査 |
| HC-06 | EKI Reference意味のテストはPASS、Local IPC/Connect実Adapter未接続 | Reference PASSのみ。実Delivery/Latency/Transport選定はNOT RUN。#753/#754 |
| HC-07 | 旧Lease epochのResultが接続後に到着、freshと誤判定される | Identity/epoch/Attemptを検査し、旧Claimを受理しない。実Provider Effect有無は別 |
| HC-08 | EKI Task.completedまたはSEI Decision済、外部Output/Owner DB Receiptなし | Job/Decisionの履歴は保持、Owner Completedを保留。#1130/#1174/#1163 |

**実装順と停止条件**：P0は既存§7.12のRead-only Required/Observed、Plugin実呼出、一Function実Provider Read、実Effectの順を維持。今回まず **HC-01/03** を同一公開合成Sourceについて「External Observation T1-A、Authoring Admission T1-B、Design Context受領」の別列に置く試験設計へ接続する。HC-04/05のSEIは#3475のO0契約比較、HC-06/07のEKIは#753の実Data Adapterと#112の外部API Ownerが実証してから連結する。実SEI Handoff/新Ontology SDK・Simulation/デジタルツインの着手はセルフビルド/SUI基礎の既存Gateを越えない。TEIの実行資格が不明ならread-only観測だけを保持して止める。

**証拠の上限**：本節はGitHub現行OwnerのIssue/PRと設計本文の照合であり、今回のREA実CLI、TEI Authoring Parser、Source→SEI実転送、EKI実Data Adapter、実PG/Host Provider、組織によるDecision受理・製品CIは**NOT RUN／NOT ACCEPTED**。他Repoのテスト結果をこのDraftのPASSとして再主張しない。Source/Schema/Transport/Grant変更後の旧Witnessを現在資格へ自動再利用しない。

### 7.15 11月8日実装進展の再照合：Plugin実呼出とTEI自身の再ビルドを分離する（横断計画）

<!-- tei-20261108-d1-and-real-selfbuild-g4-boundary -->

**現mainの進展（先行§7.12〜14の「未実装」を一律には引き継がない）**：main [#3363](https://github.com/hat47x/tei-platform/pull/3363)では既存Capability Support関数を使い、Environment Binding 24件のうち22件selected／2件missingをread-only参照で実測。ただし2件の editor.database_workbench は現Validationの検査対象外であり、**missingを製品の不具合・拒否済みと断定しない**。Draft [#3381](https://github.com/hat47x/tei-platform/pull/3381)のCLI typed Inspection Service注入口とmain統合済み[#3371](https://github.com/hat47x/tei-platform/pull/3371)のStudio Host-selected Inspection Service経路は、それぞれ局所コード・テストを追加したが、CLI Draft #3381の局所Composition Adapterでは**B選択→Bのin-process Go Call／B選択→Aの拒否**まで候補実装が進んだ。ただし**通常CLI dispatch／Studio Bridgeの両実Call-siteでのB実呼出し、実Build組込み、二Consumerの同一Provider資格は未達**。

一方、セルフビルドの現状は以前の「Reference driverのみ」から前進した。main [#3440](https://github.com/hat47x/tei-platform/pull/3440)では、**stage1で生成された実 `cmd/tei` binary自身をstage2のinvokerとして起動**し、同じchildでTEIを再構築・artifact digest一致をWSL2/Go1.27.3の一環境で実測した。main [#3463](https://github.com/hat47x/tei-platform/pull/3463)は `tei self-build` の外形・拒否条件・実child起動を製品CLIへ追加した。**ただし `-intent` は記録のみでBuild Intent未解決、`-supply`未照合、sourceClosureState=rejected、`qualified=false`、別環境独立二回試行・Seed Trust・G4正式受理は未達**。先行§7.12〜14のP0の優先順は変えず、L3の局所実証をL2/G4へ自動昇格させない。

**最新の実装・計画Ownerとの接続（同じ§7.15の精密化）**：main [#3482](https://github.com/hat47x/tei-platform/pull/3482)は`sales.order.submit`の限定Required/Observedを**13実CLI Reply・42 test-only Controls**へ拡張し、専用CIおよびmain push専用再実行がsuccess。これは1 Functionのbounded declaration/context evidenceであって実Function Invoke、Plugin実Provider、Studio第二Consumer、全Business閉包の資格ではない。main [#3439](https://github.com/hat47x/tei-platform/pull/3439)の**ADR-0048**はCLI/MCP/Web/Desktopの共通Application Operation契約を採用し、各Transport Adapterと実Host/Grantを分離した。CLI/MCPの限定validate比較（main #3458）をInspector実Providerの資格へ流用しない。

<!-- tei-20261108-d1-s2-context13-api-adr-refinement -->

**実行順の正本はmain統合済み[#3481](https://github.com/hat47x/tei-platform/pull/3481)の[MASTER_PLAN「一機能で検収する横断の最短経路」](https://github.com/hat47x/tei-platform/blob/main/plan/MASTER_PLAN.md)**とし、P0-D/M/R/C→P1-P/T/C→P2-Bの順序を本Draftで再定義しない。Inspector D1の局所的なS0〜S3補正は**main統合済み[#3499](https://github.com/hat47x/tei-platform/pull/3499)**がMASTER_PLAN作業領域Gで担当する。S0=CLI/Studio typed Service、S1=選択Providerの局所in-process Call、**S2=通常CLI dispatchとStudio Bridgeの実Call-siteを同一Contract/Sourceで結線**、S3=Build/Source/Authority/独立Expected/BOMの資格化。したがって本節のPX-01/02は新たな並行実装Issueを起こさず、#3499の**S2最優先**と#1251/#951/#1016のOwnerに接続する。main統合済み[#3378](https://github.com/hat47x/tei-platform/pull/3378)のCLI/Studio六Subject shadow parityは**exact-head Health成功の限定実Go試験**だがBuiltin/direct-Operation基準であり、Provider実呼出しの代用品ではない。

**現在性の分離**：SourceIndexだけの変更（main #3477）、外部Providerの原文・Manifest変化（Draft #3479）、判定規則Fingerprint変化（main統合済み #3490）は異なる失効原因。World FPや同じJSONだけで選択・実呼出し・受理の現在性を再利用しない。各Ownerの既存検査へ委譲し、新鮮度Registryや第二Evidence Engineを作らない。

**直近CI・実装の資格境界（11月8日追認）**：main統合済み[#3508](https://github.com/hat47x/tei-platform/pull/3508)は**Go Test成功とgofmt起因のRepository Health失敗を分離**し、旧HEADの結果を修正後HEADへ転用しない方針をMASTER_PLANへ反映した。Draft [#3381](https://github.com/hat47x/tei-platform/pull/3381)ではCLI注入HelperとStudio Host APIを使う六Subject×二Consumerの局所Go Call試験が追加されたが、**実Provider Bの通常CLI dispatch・実Wails BridgeへのProduction Wiringではない**。このGo試験の結果も当該exact HEADのCIで確認するまでPASSと記さない。さらに[Draft #3507](https://github.com/hat47x/tei-platform/pull/3507)はLifecycleを型付きGraph／不変条件／Plugin Graph Patch／Target Generatorへ分離する設計を既存Ownerへ委譲している。これは本節の状態・Flowを拡張する**設計候補**であり、実Graph CompilerやVUI/Chat実行の採用・実装完了ではない。Graphの図表投影と実Provider Invokeを同一の完了票に混ぜない。

**次に実施する一対の局所試行（設計・NOT RUN）**：

| ID | 実際に比較する入力・結果 | 独立Expectedと資格上限 |
| --- | --- | --- |
| PX-01 | 一公開合成SubjectでInspector Provider Bを選択、実CLI binary/argv/selected implementation digest/actual invoked implementation digest/Resultを記録 | B選択→B実Callのみ局所適格。B選択→A実CallはResult JSONが同一でもFAIL。Provider欠落・契約違反はfail closed。CLI既定Builtin成功をPlugin実呼出しと数えない |
| PX-02 | PX-01と同じContract/Source/TargetをStudioへ渡し、Host factoryが選んだ実Providerと実Call-siteを別に記録 | CLIとStudioの結果一致だけでProvider同一を推論しない。#3371/#3381のDraft単体テストを二Consumer実製品PASSに合算しない |
| SB-17 | main #3440のstage1 invoker実測を、`tei self-build -workspace ... -intent ...` のBuild Intent解決・実Source closure・外部module供給と別列で照合 | 再入L3の局所証拠は保持するが、未解決Intent・未照合Supply・`qualified=false`ならL2/G4はUNQUALIFIED |
| SB-18 | 同一binary digestでもSource/Provider/Toolchain/実argv/供給元・Current Grantが異なる別Attemptを比較 | digest一致だけで同じMeaning、実Provider資格、Owner受理、配備認可を推論しない |

**実装先は既存Ownerへ**：PX-01/02は#951/#1016/#1251およびDraft #3381／main統合済み #3371、Source/Required/Consumer受領は#3262/#1163、SB-17/18は#2831/#3142。SUIのItem/CanvasやREAのRaw Observationは直接AdapterでTEIの**未受理Candidate**へ渡してよいが、SHOを必須中継点にせず、Accepted Meaning/Grant/Eventへ自動昇格しない。SUI→TEIの直接データ経路の可否と、意味受理の禁止を混同しない。SEIは判断Owner、EKIは配送・資源実行の独立Ownerであり、これらの完了をTEI Provider EffectのReceiptとしない。

**状態**：本節は既存main/並行Draftの実証範囲を照合した設計・試験順の追補。PX-01/02・SB-17/18の新規試験、CLI/Studio共通Production Provider、実Provider Read/PG、G1/G2/G4の統合一Attempt、正式Owner受理は**NOT RUN／UNQUALIFIED**。新Host/Resolver/Registry/Runner/独立Evidence正本は作らない。

## 8. 今回の結論と検証の状態

**成熟した設計入力として置くもの**：状態機械の適用対象は業務に限らない／図・状態表・遷移表・交差マトリクスは同一意味を異なる形で投影する／意味Core・実行・保存・UI・文書を直結せず明示Bindingする／業務IFを全部置換しない／実Runtimeの一般化は別の採用判断とする。

**保留**：標準State Machine Role／DSL構文、階層・Region・履歴・Event orderingの採用範囲、形式的検証の範囲、組込のhard real-time・安全認証、Game TickとUI非同期のRuntime契約、GUI編集・逆写像、Targetごとの生成方式、業務Lifecycle ADRの再設計。

**今回の検証**：既存TEI Issue／ADR・Buri静的ソース／関連過去議論との設計照合のみ。TEIの新実装、UI生成、テスト／CI、通信・ゲーム・実DeviceのRuntime、Owner受理は**NOT RUN／NOT ACCEPTED**。Buriの旧環境復元・実動確認は採用Gateにしない。
