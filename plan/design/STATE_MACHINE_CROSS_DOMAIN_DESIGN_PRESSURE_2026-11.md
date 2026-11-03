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

## 7. 今回の結論と検証の状態

**成熟した設計入力として置くもの**：状態機械の適用対象は業務に限らない／図・状態表・遷移表・交差マトリクスは同一意味を異なる形で投影する／意味Core・実行・保存・UI・文書を直結せず明示Bindingする／業務IFを全部置換しない／実Runtimeの一般化は別の採用判断とする。

**保留**：標準State Machine Role／DSL構文、階層・Region・履歴・Event orderingの採用範囲、形式的検証の範囲、組込のhard real-time・安全認証、Game TickとUI非同期のRuntime契約、GUI編集・逆写像、Targetごとの生成方式、業務Lifecycle ADRの再設計。

**今回の検証**：既存TEI Issue／ADR・Buri静的ソース／関連過去議論との設計照合のみ。TEIの新実装、UI生成、テスト／CI、通信・ゲーム・実DeviceのRuntime、Owner受理は**NOT RUN／NOT ACCEPTED**。Buriの旧環境復元・実動確認は採用Gateにしない。
