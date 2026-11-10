# 外部要件による基盤評価とAI-firstヘッドレス一気通貫の設計（2026-11）

- **状態**：設計・検証計画の候補。製品契約の採用、外部要件の完全充足、各Targetでの実行・人的受理を意味しない。
- **基本方針**：[開発全体計画](../MASTER_PLAN.md#基本開発方針2026-11-01owner方針)。本書は、その方針を既存Issueの責務と検証可能な小さなシナリオへ落とす。
- **製品要求**：[RQ-001〜RQ-034](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)を参照。特にRQ-001〜008、012〜015、018〜021、025〜026、031〜034。
- **既存の評価資産**：[参照モデル移植時の不足分類](REFERENCE_PORT_IMPLEMENTABILITY_GAP_TRIAGE_2026-10.md)、[技術検証用サンプル](TECH_VALIDATION_SAMPLES_2026-10.md)、[AI向けプラグイン合成](AI_NATIVE_PLUGIN_COMPOSITION_DESIGN_2026-10.md)、[CLI-first](CLI_FIRST_INTERFACE_DESIGN.md)。
- **責務**：#1126（Data/Relation・識別）、#1127（標準操作の導出）、#2485（Driver実行）、#951（能力契約とAuthority）、#1121（Lowering）、#1129（副作用とData lifecycle）、#1130（Batch/Report）、#997（Projection/UI Kit）、#1124（Modeler）、#1180（Scenario）、#1155/#1158/#2354（CLI/Operation）、#1445（AI開発の比較）、#2391（UI外部制御）、#1119（横断資格）。個別の実装責任、受理判定、検証記録は各Ownerの正本へ置く。

## 1. 評価単位：機能名でなく、仕事・意味・実現条件を重ねる

外部要件の収集対象を、異なる規模・業種・実行環境・データ形態・運用制約を持つ製品群へ広げる。比較元の製品名・固有モデル識別子・原典の再構成資料は、この公開文書やIssueへ掲載せず、公開可能なTEI側の要求として抽象化する。出典照合に必要な詳細は、権利・公開範囲を確認した管理元で保持する。

机上評価で「要件をすべて扱った」と言える範囲は、**ある時点で明示した収集対象集合と、その要件それぞれの対応・未解決・根拠の記録**までである。未知の外部機能や未調査の動作も適合済みとは扱わない。広い収集と、個別の採用・意味受理は別の活動である。

一つの評価単位は、次の観点を横断する具体的な仕事（requirement slice）とする。

| 観点 | 調べること | 主な再利用先 |
| --- | --- | --- |
| 意味・同一性 | 何を識別・分類するか、型・値域・関係・制約・有効時点・所有者 | RQ-001/002、#1126 |
| 操作と作用 | 入力、前提、状態遷移、拒否、部分成功、補償、監査 | #951、#1129、#1174 |
| データ取得と更新 | 単一/複数Source、集合・階層・時系列、Snapshot、整列・集計、更新の逆対応 | #1127、#2387、#982 |
| 機能と表示の合成 | 反復ブロック、一覧・単票・図表・予定・文書等の投影、選択・ナビゲーション | #997、#1124、#1127 |
| Pluginと実現方式 | Provides/Requires、合成、版、Lifetime、Host/Target依存、生成・配布 | #951、#1016、#1121、#2390 |
| 利用者・組織・時間 | ロール、所属、兼務、代理、委任、認可時点、異動・履歴 | #1125、#1174、#1129 |
| 開発・検証 | AIの実装方法探索、CLI/直接編集、Scenario、独立観測、Target間の意味比較 | #1155、#1445、#1180、#1119 |

観点の表は評価漏れを発見するためのものであり、そのままCoreの新しい要素種別や固定enumとしない。特に、論理的なData、Query結果、表示Context、更新可能性、物理的な保存先を同一のオブジェクトへ押し込めない。

## 2. 机上適合表と判定

公開可能な要件IDを用い、既存のIssueや設計文書上で次の欄を追跡する。新しい一元管理DBや二つ目のGateを設けず、既存の台帳・Issue・検証根拠に参照を置く。

| 欄 | 内容 |
| --- | --- |
| 要件・適用条件 | 公開可能な要件ID、仕事の流れ、制約、Target、境界条件 |
| 根拠 | 確認済み資料と版、静的な仕様／実測／未確認の区別。非公開資料のlocatorは公開しない |
| TEIへの写像候補 | Semantic、Function/Action、Query、Projection、Plugin、Binding、外部実装の対応先 |
| 保全すべき意味 | 識別、時間、順序、値の丸め、欠損・NULL、権限、失敗・副作用、逆更新条件 |
| 複合圧力・反例 | 他の要件との重ね合わせ、隠れた暗黙前提、失敗を注入する条件 |
| 証拠と状態 | 机上で説明可能／未解決／実装済み／実行観測済み／意味受理済みを根拠ごとに分離 |
| Ownerと次の確認 | 既存の担当Issue、必要な最小fixture、失敗時に戻る契約 |

判定の際は、(a)原典の事実が不足、(b)意味の対応・受理が未確定、(c)意味は表せるが適切なProvider/Targetが未接続、(d)現行モデル・Pluginで表現できない反例、(e)検証未実施、を区別する。特に(c)や(e)をCoreの表現力不足として数えない。対象集合の全件に説明が付くことと、実環境で全件が動くことも別である。

## 3. 基盤構造へ圧力を与える代表シナリオ

次は現時点の**候補**であり、実行済みの受入試験一覧ではない。各行を固有の新IssueやGateにせず、既存の担当による反例設計・最小fixtureへ使う。

| ID | 重ね合わせる要求と成立させたい振る舞い | 見落としやすい負例・境界 | 既存の主な担当 |
| --- | --- | --- | --- |
| P1 | 一つの機能に複数の独立した1:N集合を置き、各集合を一覧・編集・集計・図表・予定へ投影する。各集合の選択と元レコードの同一性を維持する | Query結果一つに全Contextを押し込み、対象集合・選択行・更新対象がずれる | #1126、#1127、#997 |
| P2 | Query・関係・時間を含むデータの表示を、CTE/階層/集計などから派生させ、必要なContextとSourceに結び付ける | 任意の派生・集計結果へ更新先を推測して書込み可能と誤表示する | #2387、#1127、#1129 |
| P3 | 値域・単位・精度・NULL/欠損を持つ型を、SQL/DSL/外部言語へ展開して同じ結果を得る | 文字列・整数・十進数、通貨・丸め・時点の差を暗黙に同一視する | #1126、#1121、#2077 |
| P4 | 複数Database/APIの参照と更新、保留・部分成功・障害復旧を業務Actionへ接続する | 古い読取り、応答喪失、rollback不明を単純な成功/失敗へ丸める | #951、#1129、#1174、#1119 |
| P5 | 時点別組織、兼務、委任、代理・権限失効を業務Actionと結び、実行直前のAuthorityを確認する | モデル上の所属や事前承認を、そのまま現在の実行権限へ昇格させる | #1125、#951、#1174 |
| P6 | Dataの制約・導出・ライフサイクルに従属するロジックを、Scriptや外部実装を含めて合成する | イベント時点・副作用順序・実行Context・資源所有の不一致を隠す | #1129、#951、#1121 |
| P7 | 同一の受理済み意味から複数の生成Target・外部Host向けLibraryを生成し、生成成果物と契約を同じ試験で比較する | Mainクラス等の一部分のbyte一致を成果物全体の再現性と読み替える | #2016、#2077、#2019、#943 |
| P8 | 開発時のモデル検証と標準Pluginの合成・業務操作をCLIから一巡し、後続でGUIを重ねる | CLI成功を実UI、実DBの永続化、Owner受理、Production資格へ転用する | #1155、#1158、#1180、#2391 |
| P9 | バッチ、帳票、外部通知、対話操作を同じ意味・Capability境界から利用する | Targetごとに業務規則やResultをコピーし、二つ目の意味正本を作る | #1130、#1121、#951 |
| P10 | 領域固有の意味拡張・Pluginを標準と固有で共通のContractにより差し替える | 標準機能だけCoreの固定名・特権経路を増やし、依存や版の追跡が失われる | #951、#2390、#1251 |

### 複数集合を持つFunction Data Contextの最初の圧力

P1は、従来の単一一覧／単票／明細パターンだけでは見つからない不足を探す試金石とする。一つの機能Contextに、独立に取得できる複数の1:N集合があり、それぞれにQuery・識別・Snapshot・表示方式・入力/更新方針がある状況を考える。モデル上のData/Relation、データ取得結果、表示上の選択/焦点、業務Actionの対象を分け、それらの対応を追跡する。

表示形式の差し替えは、表から図表・予定へ拡張できる形を望ましい候補とする。ただし、AggregationやVirtual Queryの結果が可逆な更新写像を持つとは限らない。更新の対象識別・編集可能項目・逆写像・競合検出が解決できる場合にだけ編集操作を提供する。複数のSourceを合成した表示では、取得時点と整合性の適用範囲を明示する。

これらは**新しい万能ContextのCanonical種別を採用する決定ではない**。まず既存のSemantic Relation、Query、Application Pattern、Projection、Operation/Bindingの組合せによって実現できる範囲を検証する。表現力が不足した場合は、具体的な反例を該当Ownerへ返す。

## 4. 基盤から一気通貫までの並行開発

| 工程 | 進め方 | 到達と証拠 |
| --- | --- | --- |
| E0 机上評価 | P1〜P10を多様な外部要件へ照合し、対応と残余を既存資料へ記録する | 明示した対象集合への対応表。未解決・未知の出典を保持 |
| E1 最小基盤 | 意味/Source/識別、Pluginの契約と合成、typed Operationを、必要な範囲で先行実装する | 型・参照・依存・権限境界をfocused fixtureで検査。Core固定化の反例をレビュー |
| E2 CUI＋標準Plugin並行 | AIの探索、直接編集、CLI Validationと、標準データ操作・Query・ProjectionなどのPluginを同じ小さなWorkspaceで接続する | 記述→発見→構成→診断→変更→再読込が通常Shell/Gitで成立 |
| E3 ヘッドレス統合 | 受理済みScenarioを参照し、導出したOperation→Driver→Provider→独立Readを同一revisionと効果識別で結ぶ | 正常、拒否、競合、権限失効、応答喪失などの結果を別証拠として記録 |
| E4 GUI追加 | E3の機能契約とfixtureを再利用し、Web/Desktop等をAdapterとして接続する | 同じ操作意味と結果に対し、表示・入力・a11y・GUI固有失敗だけを追加観測 |

E1とE2は最小の共通面が成立した時点から重ねて実施できる。E0は広い要件集合を先に机上照合するが、全世界の要件が調査済みになるまで実装を停止する意味ではない。追加の外部要件が基本構造を揺さぶった場合は、その反例を使い設計へ戻る。既存のGate A/B/CやIssue #1119の資格を改定せず、各工程の証拠を既存のOwnerへ結び付ける。

## 5. AIの認知負荷を製品品質として比較する

同じ要件、同じ受入条件、同じ試験環境で、通常コーディングと「TEIのファイル直接編集」「TEI CLI併用」を比較する。開発者向けMCPは必要な時点で別条件とする。#1445/#1168の評価と共通化し、独立した工程エンジンは作らない。

- **機能発見**：既存Plugin/Contract/モデル/レシピの探索で目的に到達するまでの試行と誤選択。
- **編集と接続**：変更対象ファイル数、必要な追加記述、Plugin/Bindingの迷い、再作業の回数。
- **実行・検証**：最初の成功までの反復だけでなく、負例・独立Read・改訂後の回帰までを含む完了までの負担。
- **品質**：意味の欠落、Targetごとの不一致、誤って成功扱いした未確認事項。
- **総費用**：処理時間、トークン・Tool呼出し量、追加の人間判断を必要とした理由。意味の正式受理に必要な判断は、単純な「AIの失敗」と数えない。

最初の製品品質目標は、**同じ品質条件を満たす課題について、AIがTEIの実装方法を調べ、接続し、検証する負担を通常コーディング以下に近づけること**。実測値が得られるまでは達成と称さない。Token量のみの削減が品質・再現性を犠牲にする場合は改善とは判定しない。

## 5.1 DeepSeek HarnessでのAI-first入口とPlugin合成の追加圧力（2026-11-01、提案）

[DSH公式Sourceを固定した横断調査](../research/DEEPSEEK_HARNESS_TEI_CROSS_PRODUCT_ADOPTION_2026-11-01.md)を、AI向けCUI/Plugin/Source-Evidence/Hostの複合要件として追加する。既存[AI-native Plugin Composition](AI_NATIVE_PLUGIN_COMPOSITION_DESIGN_2026-10.md) §8・§13と[ADR-0032](../adr/0032-composition-kernel-replacement.md)で採用済みの**Cordis型Kernelの方向性は継承**し、今後の実証はDSHを「外側のHarness」、TEIを「意味と契約を持つ能力提供者」として比較する。

最初は**DSHの既存headless Profile×TEIのread-only CLI一件**を合成Fixtureで実行できるかを評価する。DSH Session LogのEventとJSON Projection、TEI CLIのtyped Result/失敗分類、固定したモデルSource、Contract/Conformance、実Artifactの資格、Ownerの意味受理を**別の観測列**として扱う。

次の差分圧力をP1〜P10へ重ねる（新しいGateやCore enumを設けない）。

| 論点 | 増やす比較条件 | 既存の責務 |
| --- | --- | --- |
| AIが選択できる能力 | Profile/Bundle/Config差分による公開Tool/Contract/Providerの変化。失われた設定と必要なScopeを検出 | #951/#1223/#2492 |
| CLIとMCPの同値性 | DSH標準MCP ClientからTEIの任意read-only Adapterを呼ぶ場合、CLIと同じMeaning/Failure/Unknownを返せるか | #1155/#1158/#2354/#2390 |
| 実行Contextの確かさ | Agent/Sessionに登録されたToolが、実Hostの権限と最新のGeneration/Scopeに一致するか | #1121/#1174/#1223 |
| 開示と監査 | 「モデルに見えた情報がログへ残る」方式と、企業内Rawを非開示とするOS/Host境界。最小Projectionで作業できるか | TAI #232、#2391 |
| 副作用の分離 | DSH Tool result/Cancel/RetryがTEI Commit/Provider Result・Owner受理へ昇格しないこと | #1129/#1174/#1119 |
| 専門家・別Harness接続 | SEIの専門性不足の認知をDSH側のSubagent/MCP/人間照会へ渡し、限定された根拠とUnknownを返す | SEI #381/#677、TAI #168 |

既存Harnessで足りる場合はTEI専用Agent Loopを増やさず、DSH実験で**機能上の増分と負担**を測る。上記は研究・実証計画候補で、DSH/TEI E2Eの実測、標準MCP必須化、正式契約採用を意味しない。

## 5.2 DSHのGoal・Scheduleと安価な大量推論を前提とする並列開発（2026-11-01、Proposed）

[DSH Goal/Schedule/通信境界の一次資料調査](../research/DSH_SCALE_GOAL_SCHEDULE_EGRESS_RESEARCH_2026-11-01.md)を、P1〜P10の外部要件適合とAI-first CUIの実証計画へ追加した。具体的なread-only実験を[#2509](https://github.com/hat47x/tei-platform/issues/2509)、複数Workstream/Goal/Schedule比較を[TAI #234](https://github.com/hat47x/tai-pleroma/issues/234)が担当する。要件ポートフォリオ自体の拡大と横断割当ては既存[#2493](https://github.com/hat47x/tei-platform/issues/2493)へ保つ。

**Goal、Tool、Schedule、Resultを同じ「自動化済み」として束ねない。** DSHの`sdk-minimal`はGoal driverを含まず、`base/headless`ではGoal関連Pluginを利用できる構成がある。ただしGoalは一Session一件で、Round上限はToken/金額上限ではなく、Restart時の再開には明示的なre-armが必要。Web側ScheduleはHost稼働中のSessionへの指示配送であり、作業の完了・失敗復旧・正式業務結果は別。

**新たな通信負例**：公式DSHの`dsh-session-log-deepseek`は既定で送信対象Eventをモデル入力とは別の`dsh_session_log` Request拡張に含め得る。既定最大8MiB/Request、`enabled:false`後のin-flight、再有効化時のBacklogは[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)と合成・隔離環境で検査する。`sdk-minimal`の既定`danger-full-access`はHost/OS機密境界を保証しない。公開可能な合成Inputを用いた`TEI CLI単体 / DSH Headless / DSH SDK-minimal`の同一Source対照から始める。

拡大する実装・Issue探索はSource/Model/Contractの共通正本を保ち、**別Session×別branch/worktree×別fixture**で並列化する。結果のMerge・独立試験・正式な意味受理、Host権限、実Effectは従来のOwner境界に留める。探索された候補数、実証済みの変更数、統合競合と人の確認負担を別々の品質指標として扱う。

**状態**：公式Sourceと設計比較まで。実DSH起動、OS通信/権限検証、DGA-01〜06、実AI並列開発の性能/費用比較、Owner正式受理はすべてNOT RUN。

## 5.3 Experimental Agent Teamsと用途相対Contextによる追加Pressure（2026-11-02、候補）

[DSH Agent Teams＋専門家連携・Construction Contextの横断調査](../research/DSH_AGENT_TEAMS_CONTEXT_EXPERT_HANDOFF_2026-11-02.md)から、P1〜P10を補完する**候補P11〜P14**を追加する。実行試験や採用済みCore enumではない。

| ID | 合成Pressure（要求） | 反証すべき不成立・既存Owner |
| --- | --- | --- |
| **P11 Team並列** | 同一WorkstreamにTeammate複数＋DAG/CAS、異WorkstreamをSession×worktreeへ分離し、Source/Fixture/Contractを追跡 | DSH Teamの`writeScopes`は警告にすぎず実FS更新を保護しない。Task完了≠検証済み成果。[#2493](https://github.com/hat47x/tei-platform/issues/2493)/[#2017](https://github.com/hat47x/tei-platform/issues/2017)/[TAI #234](https://github.com/hat47x/tai-pleroma/issues/234) |
| **P12 Context投影** | 同一TEI意味からLead・fresh/fork teammate・ACP・SEI・SHO・CLIへ、目的ごとに必要情報だけを渡す | Source/Claimの粒度・時点・許可外項目・投影漏れ、TeamのRoster/Task全文Broadcast、非公開原文のSession/HTTP送信。[#2516](https://github.com/hat47x/tei-platform/issues/2516)/[#2515](https://github.com/hat47x/tei-platform/issues/2515) |
| **P13 専門知識協調** | SEIが法域・専門性・時点・必要根拠・Unknownを識別し、外部DSHがMCP/ACP/人間専門家へ照会、意見をCandidateとして統合 | Toolが返した結果・専門Agentの自己申告・人間の見解をOwner意思・正規権限・現在の業務意味へ昇格。SEI #660/#677、TEI #2516、TAI #168 |
| **P14 進捗・実効結果** | Goal/Schedule/Team Task/Tool/CI/実Provider結果を分離し、再起動と再試行後も同じSourceに戻る | Message受付だけで実処理成功、Goal完了だけで意味受理、停止/ACK-lossで二重副作用。TAI #234、TEI #1174/#1119 |

**実証の差分**：[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)を`CLI単体 / DSH標準Headless / DSH sdk-minimal`の対照入口として維持。Agent Teamsは**任意の別レーン**とし、実験的Pluginが有効でない条件と有効条件を実Profileごとに確認する。合成・非機密Fixtureを用いて`T-01〜T-12`を既存Ownerへ配分し、並列度と独立検証成果の相関を測る。既存契約・受理境界を変えるための結論にはOwnerの判断を要する。


## 5.4 AI Proposal入力・受理・実行の接続境界（2026-11-06、設計／新E2E NOT RUN）

### 採用済み契約と今回の追加範囲

[ADR-0022](../adr/0022-ai-proposal-canonical-acceptance-boundary.md)と[#267](https://github.com/hat47x/tei-platform/issues/267)の**Shared Acceptance Boundary**を唯一のauthoritative source変更経路とする。[ADR-0030](../adr/0030-context-scoped-epistemic-knowledge.md)と[#958](https://github.com/hat47x/tei-platform/issues/958)は由来・主張種別・確認履歴・適用文脈・現在性を扱う。[#988](https://github.com/hat47x/tei-platform/issues/988)はLanguage Plugin/Operator/Source span診断、[#2619](https://github.com/hat47x/tei-platform/issues/2619)は標準データと意味Role、[#2622](https://github.com/hat47x/tei-platform/issues/2622)はData Operations/Provider契約を担当。新しいAI Acceptance Engine、Provider直結Tool、Canonical正本、Host、Policy Registryは作らない。

**境界の列**：`Source選択・実消費→用途別Disclosure/外部送信→AI Candidate→決定論的Validation→曖昧・矛盾の保留→Human/Owner Review→Accepted Source更新→reload/Conformance→型付きAction Intent→外部Hostの現在認可→Provider Effect/Receipt→独立read`。Pure read-onlyな説明CandidateはAction以降へ進まない。各列は前列の成功から自動的にPASSにならない。

### 入力/出力の最小参照票（正式Schema・新APIではない）

| 段階 | 同一Attemptへ記録する必要な情報 | 誤って確定してはならないもの |
| --- | --- | --- |
| Input selection | task/purpose、principal/consumer、Source ID・Revision・raw digest、Source range/span、外部Reference revision/観測時点、Semantic/Operator Lock、選択・実消費byte | 列挙されたSourceすべてを実際にLoad/Parseしたこと |
| Context/disclosure | Allowed *proposal/read* capabilities、用途別送出可否、実送信した範囲/byte・診断種別、必要Input Closure（observed/partial/unbound/unknown） | Modelへの入力が「全Source」、またはread-only＝外部送信自由 |
| AI output | Candidate/Attempt ID、base model revision、typed proposal/diff、各Claimの出典位置・原文Quote/意味対象、前提・unsupported/unknown、参照したSource IDs | 引用位置が存在するだけでその意味が裏付けられたこと |
| Validator | parse/type/ref、現行Source照合、Lock/Operator契約、Semantic Diff、依存Context/必要根拠、引用妥当性、曖昧・矛盾/不足 | validation合格＝Human/Owner acceptance |
| Review / Source edit | Review target/actor/decision/authority/scope、expected base revision、採用したPatch、source write receipt、再Load/Conformance | AI生成者＝review actor、過去Reviewの自動継承 |
| Host / Effect | 実Request/target identity/expected version/current grant、Provider/transaction outcome、operation identity、Receipt、独立read | 候補/保存成功＝現在の実行許可・Commit確定 |

*Allowed capabilities*はAIへ見せる提案・参照対象の上限であり、既存Host/OSの実権限Grantではない。AI入力や引用対象のSourceは**文書の内容**であり、命令・Authorityには昇格しない。Tool resultと外部文書も同様に扱い、秘密のSource path/hash、存在件数、内部エラーや拒否理由も用途相対に公開判定する（#2515/#2516/#1178）。AI用のPrompt・ログ・再生成履歴へ無許可Raw・Credentialを持ち出さない。

### Candidateの生成・適用・再開

- **引用の二重検査**：Source position/Revision/Digestが存在することと、引用文が提案したRule・Relation・条件を支持することを別列で検証。間接引用や外部Referenceのfreshnessも再確認する。Indexだけの観測や`validate ok`（未読込Source、宣言0件の場合を含む）をFull Input Closureとして扱わない（#2822/#2823/#2812）。
- **Meaningを勝手に補わない**：欠落、NULL、非開示、unknown、partial、conflicting、ambiguousを区別。Ownerの直接入力（要件・決定）は現在文脈の前提として使えるが、貼り付けた外部文書やAI文章の出所をuser確認済みへ変換しない（ADR-0030、#1440）。
- **レビュー対象を固定**：Candidate semantic identityだけでなく、Base Source Revision、実参照Scope、Review対象と判断時点を記録。Source改版・関係経路変更・Role/Operator/Grant/Disclosure変更・撤回時には当該用途の適用資格を再照合する。SourceのA→B→A、EvidenceのE1→E2→E1でfingerprintが戻っても途中の無変更や旧Review復活を推定しない（#958/#2600）。
- **再生成・再開時**：Candidate本文が同じでもAttemptは別。過去の拒否/保留をモデルの言換えによって迂回させず、再生成ごとに現在のSource/必要根拠/Allowed capabilityを照合。履歴を復元できなければ`not established`のまま。候補からAccepted Sourceへの反映は#267の書込み・reload検証だけ。
**既存Referenceの具体的な未解決点**：[#958 R3](https://github.com/hat47x/tei-platform/issues/958)では、Evidence E1→E2→E1のABA後にReview subject fingerprintが元に戻り、**旧Review CommandがReference v0で受理され得る**ことがCharacterizationされている。この現状を「既に旧Reviewを拒否できる」と記録しない。AIB-02/08では、同一digestの再登場とReviewの有効性を分離し、観測履歴/現在subjectの資格が不足すれば自動適用を止めることを**追加の受入目標**とする。修正方式・新Core状態は既存Owner判断まで保留。

- **Provider直結を構造的に排除**：AIが出せるのはtyped Action Intent **候補**まで。AIへのread-only Tool公開・Proposal生成から、Provider write/execute/commit、Shell/network egress、Host credentialを獲得する暗黙経路を設けない。実Actionは#2598の対象束縛と#1174/#1129の現在Grant/Effect Fence経由でのみ成立。ネットワーク/OS分離はUI上の宣言ではなく後段の実測対象。

### UIに投影する区別（新Core enumの採用ではない）

| 表示上の状態 | 必ず併記する根拠 | 可能にする操作 |
| --- | --- | --- |
| Source一部未取得／引用不明 | 取得したSource、欠落・部分Scope、元Revision | read-only確認・再取得 |
| Candidate生成／修正候補 | 提案Diff、Source span、生成Attempt、未解決前提 | 比較・再生成・破棄 |
| 構文検証済み・意味未確定 | passed/failed/unknownの検証対象 | Human/Owner照会、保留 |
| Review済み・Source未反映 | Review対象、判断主体、base revision | 既存Acceptanceの適用検討 |
| Source受理・再読込済み | 実Source更新・Reload/Conformanceの別Evidence | 当該意味の利用可否の確認 |
| 実行未認可／拒否／結果不明 | 現在のGrant/Request/Receipt・非開示範囲 | 安全な再照会、必要時のみ再申請 |

表示上の「verified」「green」「accepted」を単一状態にまとめず、編集成功、受理済み意味、利用資格、実Effectを別の列として示す。UIの実行ボタン活性やHuman reviewer候補の表示をHost Authorityへ昇格しない。

### 最小反証Fixtureと実証順序（AI-01〜12の設計を実Ownerへ接続）

| Case | 固定Source・独立Expectedの対照 | 必ず失敗させる誤昇格／Owner |
| --- | --- | --- |
| **AIB-01** | 根拠位置あり・別の意味／Source span不存在 | 位置一致だけで意味受理（#988/#267） |
| **AIB-02** | rev17 Candidate→rev18 Source／A→B→A | 同Digest/旧baseから現在適用を推定（#267/#958/#2600） |
| **AIB-03** | 宣言Inputと実parsed input不一致、0件/unknown import | CLI/Source選択成功を全依存Loadと推定（#2812/#2822/#2823） |
| **AIB-04** | partialまたは非開示の子集合でNOT EXISTS | 取得0件から普遍的不存在を推定（#2622/#2506） |
| **AIB-05** | 同一Sourceを公開Consumerと内部Reviewerへ投影 | 非公開Field/存在件数/拒否理由の漏えい（#2516/#2515） |
| **AIB-06** | Source本文/Tool返値に`Providerへ書け`の命令 | 引用本文をsystem命令/実Grantとして扱う（#267/#1174） |
| **AIB-07** | Allowed proposal/readのみ、AIがwrite Intent要求 | Tool availabilityをwrite Authorityへ昇格（#2619/#2622/#1174） |
| **AIB-08** | CandidateをReject→同内容を別Attemptで再生成 | 再生成で拒否・検証を回避（#267/#958） |
| **AIB-09** | Human直接回答と貼付AI原文を対照 | ingestion actorとcontent provenanceを混同（#958/#1440） |
| **AIB-10** | CandidateとExpectedを同じ生成Ruleで作成、既知Mutant投入 | 自己追認を独立検証基準合格とする（#2514/#1180） |
| **AIB-11** | Accepted Source/Reload成功後にHost Grant失効 | 受理・保存成功から実行可能と推定（#2598/#1174） |
| **AIB-12** | Host Commit ACK喪失／二重Retry | unknownを未実行とみなして重複Effect（#1174/#1129） |

**P0**：固定された合成SourceでAIB-01〜09をAI生成なしのCandidate Packetとして作り、既存parse/Validation/Shared Acceptance・Source再読込に実際に接続できる範囲を測定。AIの出力品質とは分ける。**P1**：AIB-10の独立Expectedと既知Mutantで誤PASSを検出（#2514/#1180）。**P2**：別Source版・Principal・開示範囲で二世界の返答/Tool候補/診断/外部送信を実観測（#2515/#2516）。**P3**：AIB-11/12を実Provider/外部Hostの別Attempt・Receipt/独立Readへ接続（#1174）。**P4**：その後で初めて実モデルによるCandidate正確性、誤引用率、必要な保留・拒否率、修正負担・Token/時間を評価する。平均精度値や成功件数を未実行のまま記入しない。

**受入停止条件**：Source/引用/実消費Inputの連鎖が不明、必要根拠未取得、Actor authority未確認、Denied/unknown Effect、外部送信のPolicy/OS実測不足は各必要範囲で`not established`／`denied`とし、未観測のProviderや本番可用性を合格としない。#267はCanonical Source書込み・Owner受理、#988は構文/意味候補、#2619は意味Role/境界、#2622はOperation/Provider、#958はepistemic履歴、#2516は開示、#1180/#2514はScenario/独立検証基準、#1174は実Effectを所有する。**本節は計画更新のみ、AIB-01〜12の実行・実AI/CLI/OS/外部Host/Owner統合受理はNOT RUN／NOT ACCEPTED**。

## 5.5 `*.tei.md`統一に伴うAI入力・Authoring・受理境界の再資格化（2026-11-06）

### 決定の更新と対象範囲

2026-11-06 12:10 UTC頃のOwner方針：**TEI向けAuthoringの標準ファイル名を`*.tei.md`へ統一し、従来の`*.tei`の新規利用を原則停止、既存資産は段階的に移行する**。Markdown上の表・文章・図をAIが段階的に定式化しやすくし、通常の`*.md`との差をファイル名から認識できるようにする。これは従来[#2621](https://github.com/hat47x/tei-platform/issues/2621)末尾に残る「`.md`と`.tei`の別ファイル／`.tei.md`標準化せず」という**2026-11-06 12:04 UTC以前の比較案を置き換える**。古い比較は履歴として保持し、今回の明示決定を現行方針として扱う。

ただし**決定済みは命名・移行方向**であり、現Parserが任意の`*.tei.md`のMarkdown全体を読み込める、既存`*.tei`を単純renameできる、IDE/Build/SHO/Studioの全Consumerが対応済み、という意味ではない。既存`*.tei`は移行完了まで所在・参照・正本を追跡し、黙って削除/再解釈しない。OKF `type`/`tags`/任意`tei`メタデータは併用候補だが、Filename/Tag/OKF適合を意味受理・Source実選択・Host Grantの代替にしない。OKFの規格判定とTEIのLanguage/Role解釈も別。

### 同一Source・一つの正本を守るためのAI/CLI契約

1. **Discoveryと実解釈の分離**：`*.tei.md`はTEI向け文書を探す規約であって、Parser/Operatorを自己認証する仕組みではない。Workspace/Manifest/Seed/Language Pluginの実SelectionとPinを[#988](https://github.com/hat47x/tei-platform/issues/988)・[Draft #3028](https://github.com/hat47x/tei-platform/pull/3028)のK0〜K6に照合する。未対応文法・Version mismatch・選択不能は`unsupported/unresolved`として検出し、AIの推測で有効化しない。
2. **Authoring表面と意味Roleの分離**：Markdown全体を一つのAccepted Meaningへ昇格しない。文章、説明表、独立Expected、Rule/Model Candidate、受理済み定義、手作業のProcedure、非実行の図を、**範囲・対象・Role**で区別する。構造化Candidateは元のSection/Row/Code fence/Source span/Revisionへ戻せるようにし、未選択範囲を変更しない（#2620/#2621/#267）。表層の文章が変わらないことと、解釈Plugin/Operator版が同じことも区別する。
3. **Migrationと引用の二段階**：`old.tei → new.tei.md`のPath/参照移動・引用位置は、論理Source identityとraw byte/Revisionに照らして再解決する。改行/BOM/Unicode/frontmatterの差、複数同名/同内容の別Source、A→B→Aは明示的なRebase/Conflictへ。Pathの移動だけでOwnerの旧Reviewを別Occurrenceへ転写しない。同時に旧`.tei`と新`.tei.md`を編集正本としない。旧形式がMarkdownでなければ**構文変換＋独立Expected＋Source確認**を要し、単純rename適合とはしない。
4. **AI入力・開示とTenant**：Source内の文章・表・コードフェンス・OKF値は**untrusted content**であり、`Providerへ直接更新`等の文字列を命令として実行しない。`*.tei.md`のread/authoring権限、AIへの提示、外部LLM送信、Plugin Composition/Organization scope、別Tenantへの共有、Runtime Host Grantを独立の判断とする。兼務者のMembership/Occupancyや同PluginのDistributionは、他Tenantの秘密参照権限・代理権・実行権限を付与しない（#2386/#2515/#2516/#1174）。
5. **Owner受理と実効結果**：Candidate Diff→引用/構文/意味/独立Expected→#267 Shared Acceptance→実保存Sourceのreload→実Build input/Consumer読取→#2598のRequest Target→#1174のcurrent Grant/Provider Effect/Receipt/独立Readの順に、**必要な範囲だけ**別Evidenceを保存する。受理済みのMarkdown内Ruleがあっても、任意の表/説明まで自動で実行対象にしない。Preview後のGrant撤回・表示Epoch変更・Commit unknownの扱いは後段のOwnerに帰属。

### 最小差分Fixture（AMF-01〜07、独立Expected・新規実走なし）

| ID | 同一Source/対照条件 | 独立Expected／誤成功の検出 | Owner |
| --- | --- | --- | --- |
| **AMF-01** | 同じ説明本文を`plain.md`と`work.tei.md`へ配置、TEI Role宣言なし | 後者だけを根拠にRule実行可としない。Source選択・Parsed Meaning・Owner受理を別結果として表示 | #2621/#988 |
| **AMF-02** | 旧`work.tei`がMarkdown非互換、`work.tei.md`へのrenameのみ | rename≠Parse成功・意味同等。変換不能は診断し、旧正本を保持 | #988/#2621 |
| **AMF-03** | CRLF/BOM/UTF-8日本語を含む表でSection/Row位置移動、旧引用あり | 実raw byte/Logical Source/Revision/Spanを照合し、別行への誤対応を拒否。単なる表示位置変更とMeaning変更を区別 | #2621/#267 |
| **AMF-04** | 同じ`.tei.md`内に受理済み表・AI候補表・独立Expected・未解決手順が共存 | ファイル全体をAccepted/Executableとせず範囲別Role/Review。ExpectedとCandidateの共有由来は独立検証基準失格 | #2621/#2514/#267 |
| **AMF-05** | Source内の引用/コードフェンスが`ToolにProvider.write実行を命令`、Allowed=read/propose | 注入文はCandidate引用に留め、実Capability/外部送信/Host Grantは追加しない | #2515/#2619/#1174 |
| **AMF-06** | 同PluginをTenant A/Bで利用。AIはAのSourceに閲覧権限、Bには許可なし。兼務者のOrg Roleは両方に見える | Bの実Source/診断/候補/件数を開示しない。AのOwner Review/Plugin InstallからBのHost Grantを推定しない | #2386/#2516/#1174 |
| **AMF-07** | `.tei.md` Ruleの受理・Buildは成立、Preview後にGrant撤回／旧Receiptは残存 | 意味資格と現在の実行資格を別評価。実Commit不明はEffect lookup/別ReadまでUnknown、旧Receiptで新Attempt成功にしない | #2598/#1174 |

**実証を優先する最小Slice**：AMF-01/02/03/04でfile-native Authoring→Source Index/Plugin Selection→read-only解析・引用→Candidate→#267のShared Acceptanceとreloadを同一Revisionで確認。その後AMF-05/06のAI/Tool/OS送出・Tenant境界を、AMF-07の現Host/実DB/Receiptをそれぞれ既存Ownerで別Gateとして実証する。固定Source/Fixture/独立Expectedを先に定義し、AI生成品質は最後に別評価する。新しいMarkdown Parserや移行Writerをこの計画PRへ実装しない。

**証拠水準**：拡張子統一はOwnerの新方針。Migration可否、既存Parserの受容、OKF・SHO連携、AI/OS/Consumer実送信、Tenant別認可、Owner受理、実Provider/Host E2E、AMF-01〜07は**NOT RUN／NOT ACCEPTED**。[#988](https://github.com/hat47x/tei-platform/issues/988)のBS-01〜15、[#2621](https://github.com/hat47x/tei-platform/issues/2621)のFM-01〜08、前節AIB-01〜12は既存Ownerの別Caseとして維持し、今回の命名決定をその過去の試験PASSへ遡及適用しない。

## 5.6 AI-first受入を「一つのSource、一つのAttempt、独立Expected」に縮約する（2026-11-07）

### 先行成果の接続と、証拠の上限

前節のAIB-01〜12とAMF-01〜07を無制限に増やす前に、既存の[#3030](https://github.com/hat47x/tei-platform/pull/3030)（`*.tei.md`へのM0〜M6、TM-01〜14）、[#3028](https://github.com/hat47x/tei-platform/pull/3028)（Seed/Parser/Operator、BS-01〜15）、[#267](https://github.com/hat47x/tei-platform/issues/267)（Shared Acceptance）、[#1180](https://github.com/hat47x/tei-platform/issues/1180)（Scenario）、[#2514](https://github.com/hat47x/tei-platform/issues/2514)（独立検証基準）の**同一Source縦切り**を先行させる。

現状の独立Evidenceを一つの製品横断PASSへ合算しない。
- [#3030](https://github.com/hat47x/tei-platform/pull/3030)は**docs-only Draft**で、現mainのLoaderが`*.tei`固定名・globに依存し、`filepath.Ext("x.tei.md")`は`.md`、TEI Lexer/ParserはMarkdown全文を受け付けないことを確認した。**新`*.tei.md`の実Parser/Writer/CLI成功は未実証**。固定Gate資産と`.tei/decisions/`は一括変更対象外。
- [#2901](https://github.com/hat47x/tei-platform/pull/2901)の固定Greenと、[意図的Mutant #3017](https://github.com/hat47x/tei-platform/pull/3017)の固定Redでは、公開合成`relation-unique`の**一件のEvaluator誤実装をAI追加Scenario自身が検出**した。これは当該Product Rule Mutantに限る判別力であり、新Markdown Reader・実Provider Read・業務Ownerの検証基準・本番適格性を示さない。**Mutant branchをmergeしない**。
- [#2853](https://github.com/hat47x/tei-platform/pull/2853)等の既存実PostgreSQL試験は`CommitUncertainError`のCommand unknown伝播と同OperationIDのReceipt Lookupを限定実証。実Provider/外部Hostの全Grant/Guard/Effect/After-commitの保証ではない。同じSubjectの別Key再送や前段Stepの部分効果も除外できない。

### 一枚の受渡し票：証拠の有無と、主張できる範囲

以下は新しいCore Schema、Audit Registry、Run APIではなく、既存[#1163](https://github.com/hat47x/tei-platform/issues/1163)／[#1119](https://github.com/hat47x/tei-platform/issues/1119)のEvidence Packetに対して必要な**読取り専用の照合列**。

| 証拠列 | 実際に照合する対象 | 不足時の停止線 |
| --- | --- | --- |
| **S: Source** | `*.tei.md` raw byte、Source ID/Revision、Manifest/Root選択、Markdown断片元span、Language/Operator exact Pin、実Read/Parsed Input、依存閉包 | filename/タグ/Discovery/validate okだけではSource利用資格なし |
| **C: Candidate** | AIまたは固定CandidateのAttempt、Typed Diff、引用品と根拠の一致、未確認前提、引用・Review対象のScope、Audience別Disclosure | 引用不成立・部分文脈・外部送信資格不明なら保留／非開示 |
| **V: Validation/独立検証基準** | 同一Source版の決定論的Validation、独立Expectedの作成由来、既知Mutantの検出／非検出 | Parser/CI合格でも業務意味未受理、検証基準の自己追認なら不適格 |
| **M: Meaning** | #267 Shared AcceptanceのHuman/Owner Decision、対象・有効範囲・判断時点、Source書込みと実Reload | Candidate acceptedに見えてもSourceに未反映ならMeaning未確定 |
| **G: Current Grant/Binding** | 現在Principal/Tenant、必要Guard集合と供給/評価済み集合、Request/Target/Version/Operation ID | Owner受理とTool存在から実行資格を推定しない |
| **E: Effect/Consumer** | Provider実行とCommit/Unknown、Receipt/independent read、Consumerが現在のWorkspace/selection epochで実際に読んだbytesと開示可否 | 旧Receiptや後着Responseで現在のOperation/表示を成功扱いしない |

**二つの独立軸**：`M (業務Meaning accepted?)`と`G (当該Attemptの現在Grant/Target/Guard valid?)`はどちらも独立。M=Yes/G=Noは「定義は受理済み、実行拒否」、M=No/G=Yesは「権限があっても業務意味を未受理」、M=No/G=Noは双方不足、M=Yes/G=Yesでも**実Effect Eの成立は別観測**。Read-onlyのCandidateレビューはGが未検査でも成立し得るが、書込みは別。MはHuman判断であり機械PASSから自動導出しない。

### 次の実際のP0：一つのRead-only Source→Candidate→Review Slice

1. **前提の正確な判定**：現mainは新形式MarkdownのLoader/Parser非対応なので、まず#3030 M0/M2/M3の最小実装が必要である。Readerがないうちは`*.tei.md`のraw Draftを確認できても`Parsed TEI Source=verified`としない。先にファイル名だけ切り替えたSynthetic PASSは目標ではない。
2. **最小の固定Fixture**：注文一件の説明・型付きTEI断片・検証用表・不明な承認条件を一つの`*.tei.md`に置く。既存TM-02/05、AMF-03/04、AIB-01/03/10を**同一Fixtureと独立Expected**へ参照し、別の12件のCaseを新設しない。Markdownの未選択部分の保持、Source span/byte、`required` Input集合、Ref/Role/Version不一致、引用妥当性を別列で確認する。
3. **LLM抜きの接合**：人手で作った固定Typed Candidateを#267へ渡し、`not established`、`rejected`、Owner受理の3条件を対照する。Owner Decisionが得られない場合は「Authoring/Validationの限定成功」で止まり、意味受理済みとしない。受理済み変更は実Source再読込後に同じ意味が得られることを確認する。
4. **独立検証基準による誤成功検出**：#2901/#3017の同一Case・同一Expected・一行Mutantという**既存の有限実績**を、独立検証基準とMutantの実験設計参照にする。ただし元の`*.tei`または旧Scenarioで成立したGreen/Redを、新`*.tei.md` Readerに継承しない。新Sourceの実消費版を使う場合は改めて実測する。
5. **後段の二独立レーン**：P1 Consumer Disclosure/実送信を#2515/#2516、P2 実Host Grant/RunPlan/Provider Receiptを#2598/#1174へ渡す。前者の公開安全はSource/AIプロンプト以外のTool/ログ/Session/CI出口を含み、未観測はunknown。後者の`blocked`はRunPlan全体のno-effectではなく、このStepの事前停止となり得る。UIの旧選択Epochへ後着した応答も現表示の成功に使わない（#1824）。

### 受入判定・計画変更の限界

- **最初の統合Doneの最小条件**：`*.tei.md`実Loader/Parserで選択した断片が読まれ、固定Candidateの出典を照合し、独立Expectedを持つValidationで正常・誤引用・部分/未知を正しく分け、#267のAccepted Source更新を実行した場合のみReloadと一致を確認する。これらのうち実行していない列は**NOT RUN**、Owner Decisionが得られなければ**NOT ACCEPTED**。Read-only検証の成立をHost実行資格へ昇格しない。
- **優先順位**：今はAI生成トークン数やUI画面数を増やすより、この一件で`Source selected ≠ Source consumed ≠ Candidate valid ≠ Meaning accepted ≠ Host currently authorized ≠ Effect confirmed`を証拠付きで示す。P0の停止位置を測り、現在の#3030・#267・#1180・#2514・#1163のOwnerに戻す。
- **未受理**：この節はDocsの接続計画のみ。`*.tei.md` Reader/Parser/Editorの実装、同一Fixtureの実CLI・AI/Harness・OS/External LLM・Studio/SHO・PG/Host、M/Gの実結合、生成精度や性能、Owner受理を実証・採用したと主張しない。第二のReader/Registry/Proof Engine/Runner/Host/独立検証基準Engine、固定Gateの改変はしない。

## 5.7 既存形式の実Provider検証と新Authoring移行を分離し、完全性証拠で合流する（2026-11-07）

### 5.7.1 前節P0の依存関係を修正する：二レーンと一つの合流条件

§5.6の「`*.tei.md` Reader/Parser実対応を初回P0の前提とする」は**新形式Authoring→Shared Acceptanceを主張する場合**に限定する。旧`*.tei`で現行Loader/Function/Providerを観測する**P0-L（Legacy Runtime Read）**を、Reader未完成のため停止させない。一方で新`*.tei.md`を使った**P0-M（Markdown Source Admission）**の結果を旧`*.tei` Readerで代替しない。二レーンを分ける判断は、既存[Script起動 Draft #2884 §10](https://github.com/hat47x/tei-platform/pull/2884)のL/Mに整合する。

| レーン | 最初に行う既存経路 | 認定できる最大のClaim | 認定できないClaim |
| --- | --- | --- | --- |
| **P0-L：Legacy** | 固定済み`*.tei` → 実Workspace Load/Language・Role Selection → 既存Function/Query/Provider read-only → 独立Reader/Expected | **実際に読んだ旧Source・実Provider Read範囲**での値／診断適合 | `*.tei.md`のMarkdown Reader/Parser、対象全件の網羅、Owner正式意味受理、新形式との同値性 |
| **P0-M：Migration** | [#3030](https://github.com/hat47x/tei-platform/pull/3030)のTM-02/03/04/14と、[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)の未対応Source Guard → 真のMarkdown Container Reader/選択Scope/Plugin Pin → Parser/実消費の逐次照合 | `*.tei.md`を**どのSource Kind・実byte/spanで読み取ったか／未対応なら拒否したか**という限局Claim | 実Provider結果、旧Sourceとの意味同等、Candidate/Review/Host/Business受理 |
| **P0-J：合流** | L/Mで別々のSource Revision・実消費byte・Language/Operator Pin・必要対象集合を観測し、**同じ業務Requirement/独立Expected**へ対応付ける | 同一Meaningを主張する**範囲を明示した条件付き比較** | 同値値/同一Canonical ID/単一CI成功だけで全面移行完了と宣言すること |

**移行計画のOwner（2026-11-07更新）**：`*.tei.md`移行仕様・M0〜M6・TM-01〜14の主担当は[#3030](https://github.com/hat47x/tei-platform/pull/3030)。[#2667](https://github.com/hat47x/tei-platform/pull/2667)のTMD-01〜12は横断的な追跡と既存Caseの対応表であり、第二のMigration Gateではない。Authoring ProfileのAU-01〜08も#3030が比較し、本書で第三のFixture集合を設けない。#3031は未対応の新Sourceを黙って検証PASS扱いしない**暫定Guard**であって、Markdown Reader実装ではない。

### 5.7.2 Source／Target／Guardの「必須集合」と「実観測」を分ける

[main統合 #2926](https://github.com/hat47x/tei-platform/pull/2926)の`semantic.EvaluateCompletion`は、与えられた`TargetSet × EmptyPolicy × Predicate`に対する`holds/fails/unverified/contradiction`を判断する有限評価器として利用できる。ただし**Callerが`CoverageComplete=true`を渡したことは、ProviderからOwner対象全件を取得した証明ではない**。必要対象を列挙した根拠、実取得Row/Field/Relation、負例・空証明・未観測対象、Snapshot/Principal/RLS/Provider Scopeを別列で照合しなければ、完全性は`unverified`のまま。

| 連鎖の段階 | 必要な集合・観測値 | 誤って合格させない条件 |
| --- | --- | --- |
| Source/Import/Plugin | `required source closure` vs `discovered` vs `managed` vs `loader-read` vs `parsed` vs `actually consumed` | Catalogやvalidateの0件成功、未選択Source/Pluginを無関係と誤認 |
| Role/Rule/Function | Required Rule・Referenced Subject・Operator Pin vs Evaluated Rule/Context/Field | 出力Fieldだけを見て非表示Predicate/NOT EXISTSの依存を省略 |
| Provider/Target Set | Ownerから独立に決まるRequired Target/Identity/Scope vs 実Read Row/Field/Complete Slice/Empty Evidence | 取得0件・行Subset・CallerのCoverageComplete自己申告を全件証明へ昇格 |
| Action/RunPlan | 必須Guard vs 供給Guard vs 当該Attemptで実評価済みGuard、current Grant/Target/Period | 検証済み一部Guard、Observer存在、RunPlan completedをProvider Effect確定へ昇格 |
| Consumer/Result | 実Artifact bytes/Generation/Selection epoch/Disclosure vs 返却Request/Receipt/独立Read | 旧Workspaceへ由来する後着Responseや過去Receiptを現在結果とする |

この表は新しいProof EngineやUniversal Completion APIではない。#1180＝必要Obligation/Scenario SSOT、#2514＝独立Expected/Mutant判別力、#2595＝実Build Source、#2823/#3031＝実Loader/未対応拒否、#2586/#2628＝実Provider Read、#1125/#2926＝Completion評価、#2598/#1174＝実Target/Grant/Effect、#1163＝同一Attempt証拠の連結に帰属させる。

### 5.7.3 最小の対照を既存Fixtureに重ねる（新規独立Caseを起こさない）

1. **Legacy正例＋部分取得負例**：同じ旧`*.tei`のSource/Model/Queryで期待されるTargetSetをAI生成Actualと独立に固定。実Provider read-onlyが取得したSubject/Field/Scopeを記録し、真部分集合を意図的に与えても`holds`に昇格させない。Provider接続未取得ならSource-static/Referenceに留まる。
2. **新形式の拒否→正常受付**：同じ`*.tei.md` Sourceを現行Reader未対応条件で`TEI_MD_PARSER_UNBOUND`（#3031）とし、将来Reader/Pinが成立したときに限り真にParserが実消費した範囲を観測。**拒否が正しいことと、受け付けたとき正しく意味解釈できることを正負対照**にする。
3. **旧新合流**：旧形式／新形式それぞれのActual Source bytesとSource spanを独立に記録し、同じIndependent Expectedを使って**対象範囲・意味差分**を比較する。生成したMarkdown/変換結果自身がExpectedを記述した場合、独立検証基準として採点しない。旧Sourceと新SourceのID一致／0件／CI総Greenだけで意味移行済みとしない。
4. **Completionの最小負例**：TargetSetが3件と独立に確定しているのに2件しかLoader/Providerから到達しない場合、`CoverageComplete`が機械的にtrueでも、**対象集合のCoverage主張は未立証**。別途与えた不正1件を検出して`fails`となる局所ケースと、全件確認した`holds`を混ぜない。空集合はOwnerが選んだ`EmptyPolicy`と独立空証明の有無で判断する。
5. **業務/実行は後段で区分**：P0-L read-only成功とP0-M Parser適合の結合後、固定Candidate→#267 Shared Acceptance→Source Reloadを実証。Script Publish/Queue/Start Admission→current Grant→Provider Commit/Receiptは#2884/#1174へ引き渡す。**Publish承認やQueue時のGrantを実Dispatch/Commitの現在Grantとしない**。Cancel/Task Completeはno-effectの証明ではない。

### 5.7.4 現行の確認できた実装・未実証範囲

- **実装された限定Guard**：[#3031](https://github.com/hat47x/tei-platform/pull/3031) fixed HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae`。Focused GoおよびWorkspace packageの試験・vet、Actions #42217379093 は限定success。ただし**Draft／未統合、現mainの安全性として流用しない**。
- **Completion評価器**：[#2926](https://github.com/hat47x/tei-platform/pull/2926)はmain統合済み、限定`TestEvaluateCompletion`成功。`TargetSet`/実Coverage/実Provider read/Hand-offは未接続。
- **Bootstrap**：[#3028](https://github.com/hat47x/tei-platform/pull/3028)のBS-02/03/06は既存Go APIに正負対照を持つDraft。新Parser実受入、全BSの実証、実Plugin有効化、全実Binary同一性は別。
- **Source/AI独立検証基準**：#2901 Green→#3017故意Redは一件のProduct Evaluator誤実装をAI追加Scenario自身が検出した限定結果であり、`*.tei.md`/実Provider/全Target Coverageを認定しない。#2514が以後の検出力とExpected独立性を所有。
- **新P0-L実Provider→P0-M実Markdown Reader→P0-J旧新合流→実Candidate/Shared Acceptance→Host**の同一版統合試験、実OS/外部LLM送出、実Tenant/Policy、Owner業務意味受理は**NOT RUN／NOT ACCEPTED**。Go単一バイナリ標準、外部Hostの権限主体、第二Meaning SSOT/新Host/Registry/Proof Engineなし。

## 5.8 実装済みProposalの限定契約と、Shared Acceptance／実行Effectへの橋渡し（2026-11-07）

### 5.8.1 三種類の「受理」を同一視しない

[main統合 #3041](https://github.com/hat47x/tei-platform/pull/3041)により`internal/validation/proposal.go`の**`validation.Propose`／`validation.Accept`**と`tei validate -json`の`proposals`が利用可能になった。これはAI専用Proposal形式やSource修正Patchの実装ではない。

| 境界 | 現実のFit/Gap | 受入の停止線 |
| --- | --- | --- |
| **V0 診断→提案** | `validation.Propose(diagnostics, worldFingerprint)`は各診断に`Code/Severity/Subject/Diagnostic/Revision`を結び`Accepted=false`で生成。`Guidance`は任意。**Source引用span・外部Source revision・依存閉包・Allowed capability・AI Attemptは含まれない** | `proposals`非空／個数一致／JSON表示だけでは引用の正当性、Source全件読取、実行可能性を主張しない |
| **V1 局所フラグ** | `validation.Accept(&proposal, acceptedRevision)`は同じ文字列版・非空・二重受理禁止を検査して**メモリ上の`Accepted`をtrueにする**。この関数自体にはReview actor、署名/Authority、Accepted Sourceの永続書込み、Reload/Conformance、Source ABA履歴の照合がない | `Accepted=true`を[ADR-0022](../adr/0022-ai-proposal-canonical-acceptance-boundary.md)／[#267](https://github.com/hat47x/tei-platform/issues/267)のHuman/Owner正式Meaning Acceptanceや承認Receiptと解釈しない |
| **M 正式意味受理** | 既存Shared AcceptanceのCandidate identity、Review fingerprint、現Base revision、Actor authority、Scoped Decision、実Source反映、Reload後Conformanceを別証拠とする。AI/Validatorの局所`Accepted`の真偽はこれを代替しない | V1からMへの自動昇格・AI独自Writerを認めない |
| **G/E 現行実行とEffect** | [main統合 #3042](https://github.com/hat47x/tei-platform/pull/3042)はCommand Authorizer未設定を`COMMAND_AUTHORIZER_UNAVAILABLE`で拒否し、宣言PeriodのObserver不在を`COMMAND_PERIOD_STATE_UNAVAILABLE`で拒否。宣言外のEffectは実Executor後に分かる場合があるため`EffectUnknown`／Lifecycle`unknown`で保全 | 明示`ReferenceAuthorizer{}`はReference専用で**真正なHost Grantの検証ではない**。拒否やUnknownを`no effect`に読み替えない |

`worldFingerprint`一致は**当該Comparisonで使ったRevision文字列一致**の証拠に限り、Source path移動、外部Referenceのfreshness、Owner Review撤回、Role/Plugin版、A→B→Aの観測連続性、実Byte閉包まで包括しない。真のAI向けCandidateは§5.4の`Source/Revision/Allowed capabilities/引用/依存Context`に基づく**別の上位契約候補**であり、`validation.Proposal`に全項目を今すぐ押し込まない。

### 5.8.2 参照実行の「実効果との差」を消さない

[main統合 #3060](https://github.com/hat47x/tei-platform/pull/3060)の[固定Evidence](../../plan/evidence/XTEA_LOWCODE_OPERATION_DIFFERENCE_2026-11.md)は、参照Storeの`OperationRecord.Differences`とCLI/Studioの件数投影を試験で確認した。宣言に無い追加Scopeは履歴上の未解決差として残すが、発行SQL、実DBの更新前後、実Provider効果を検証していない。`committed:true`かつ`persisted:false`・`durability:none`のReference CLI結果を、実DB CommitやBusiness Completionの証拠にしてはならない。実Providerでの未宣言効果、部分成功、ACK喪失は#1174のOperationID/Receipt/独立Readの検証対象。Consumer UIには`Proposal`／`Review`／`Accepted Source`／`Request admitted`／`Effect observed`を別の状態として表示する。

### 5.8.3 既存Caseを再利用する最初の差分検収

- **AIB-01/03（出典・完全性）**：`validate -json`に診断とProposalが出た正例を、未読込必須Sourceを混入させた負例と対照。診断0件/Proposal0件を「必要なSourceがすべて正常」と認定しない。Legacy Provider read-only P0-Lは並行してよい。
- **AIB-02/08（古い版・Reject/再生成）**：同一Proposalを誤Revision／空Revision／二度の`validation.Accept`で拒否する機械対照と、**真正なReview actorを伴うShared Acceptance**を別Case列で測定。V1局所trueでもMに未到達なら正式受理なし。Source ABA・撤回の実再資格化は#958/#267。
- **AIB-07/11/12（実権限と副作用）**：Authorizer=nil／宣言Period observer=nilでは拒否、`ReferenceAuthorizer{}`を本番Grantとして認めない。Executor後の宣言外Effectはunknown/Receipt Lookupの対象にし、無確認再送を合格としない。#3042のWindows限定Go PASSを実PostgreSQL/Host同一Attemptへ流用しない。
- **AMF-03/04／TM-02/03/14（Markdown移行）**：`*.tei.md`の実Parser/Source span/Required Inputは[移行主担当#3030](https://github.com/hat47x/tei-platform/pull/3030) M0〜M6/TM-01〜14で判定。[#2667](https://github.com/hat47x/tei-platform/pull/2667)のTMD-01〜12は**横断的な追跡・歴史的対応表**であり、別のMigration Gateや唯一の移行受入正本ではない。旧§5.7/PR説明にある「#2667が移行横断Owner」という記載は、#3030の最新Owner確定に対して補正する。#3031は未対応形式のGuardでありReaderではない。

**用語統一**：[docs/naming.md](../../docs/naming.md)の2026-11-07方針に従い、テスト判定の説明では「独立期待値」「独立検証基準」「適合判定」を用い、`Oracle`は製品名**Oracle Database**の意味だけに用いる。旧説明文の表記は本文で更新するが、歴史的Case ID・実装識別子・URLは機械的に改名しない。なお現mainの同`docs/naming.md`には旧`.tei`標準の説明が残る。これは命名規約の表示更新が`*.tei.md`の実Parser/正式Migrationを先取りしないよう、[#2621](https://github.com/hat47x/tei-platform/issues/2621)／#3030のOwnerへ照合すべき**別の文書整合課題**。

**証拠水準／次の実行**：#3041のGo試験・#3042のWindows Goテスト・#3060のReference記録はそれぞれ別HEAD/別Claimの既存限定証拠。本節は**採用済みAPIの意味と受入上限を照合する文書追補**のみ。新`*.tei.md`実Reader、実Candidateからの正式Source書込み、同一Source/Owner/Host/Provider/Consumer E2E、実外部LLM、OS隔離・Producer同一性はNOT RUN。第二Accepted Source正本・Authorization Registry・AI Host・万能Completion Engineを新設しない。

## 5.9 生成AIの三つの経路とOKF Attested Computationの限定接続（2026-11-07）

### 既存Product SkillのS1実証を新しい境界へ反映

[#2605](https://github.com/hat47x/tei-platform/issues/2605)で、TEIのビルド済Go CLIをTAIの別リポジトリCIから実起動し、固定TEI HEAD `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`・公開`reference/workspace`・`semantic-data:sales.order`についてCatalog/Inspect/Validateの実結果、Source Index path・指紋と原Source byte SHAの一致を限定確認した（[TAI CI #42101157330](https://github.com/hat47x/tai-pleroma/actions/runs/42101157330)、合成S1-W01〜08も別評価）。これは**外部リポジトリからの実Product CLI read-only利用**であり、外部AI HarnessによるTool選択・Tool Request→argv、Skill本体/参照文書実ロード、Owner意味受理、全Source閉包、実OS/Provider/Hostを証明しない。従来の「CLIも全てNOT RUN」という旧Statusをこれからの計画に引き継がない。

### AIが返す三種類のものを同じ`Accepted`に畳まない

| AI経路 | AIに許す範囲と正式な判断主体 | 検証できる実績の範囲と停止線 |
| --- | --- | --- |
| **A: 自由な意味・設計のProposal** | Source/Role/Rule/Query/Authoringの**候補Diff**と引用を生成。#267のHuman/Owner Shared Acceptanceが業務Meaningの正式受理を所有 | `validation.Propose/Accept`の局所フラグやAI文章は正式Source変更ではない。Source/引用・Revision・決定主体・保存後再Readが不足なら未受理 |
| **B: 正規定義済みComputationのParameter Bind** | 事前に選定・適格性確認した計算定義の**宣言された型付き引数だけ**を埋める。AIがSQL/Script/Computationを自由に書き直した場合はこの経路を利用できずAとして再提案・受理が必要 | 正規定義のSource/byte・Runtime/Pin、実Bind、Executor実実行/Job/Receipt、非LLM Attester実検査、結果の独立再Readを**別Witness**とする。Attester PASSは定義の業務Meaning・対象集合全件・現在Grantの証明ではない |
| **C: Provider・外部ActionのEffectful Intent** | 型付きOperation Intent候補まで。正式なRequest/Target/現在Grant/Period/Effect Fenceは#2598/#1174・外部Hostが所有 | OKFのAttestationやSkill手順の存在でProvider write/commit権限を付与しない。拒否/Unknown/partial effect、再送・Receipt/補償はHost側で独立検証 |

OKF v0.2 [一次仕様 §10](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)の`type: Attested Computation`は、`runtime`、`parameters`、本文または参照先の`computation`、`executor`と`executor.receipt`、`attester`という契約を記述する**知識Concept**であり、実TEI Provider/Hostを自動構築しない。OKF v0.2は実行/Attesterのwire format、ABI、Sandbox、Attestation Cacheを規定していない。新TEI CLIコマンド、共通Receipt Format、第二Attester Store、全Plugin常駐を本節で採用しない。

### 定義の現在性・実行の忠実性・結果の正しさを独立に評価

1. **文書由来と利用資格**：OKFの`sources[].id/resource`とClaim footnote、`generated.by/at`、`verified[].by/at`、`status/stale_after`は発見・引用・書き手／検証者の来歴に利用する。Actorの`human:`文字列、`verified`、`status: stable`はTEIの正式Owner Acceptanceや現在のHost Grantではない。期限内でもSource/Operatorが変われば現在資格を再評価し、期限切れは無条件の過去Receipt否定にしない。未認識type/任意key・broken linkでOKF Bundle全部を拒否しない一方、個別の実行資格は未確立とできる。
2. **計算の同一性**：AIが自由に作ったSQL文字列ではなく、認められたComputationの**正確な実byte／Binding Semantics／Plugin・Operator Pin**を基準として、Executorが実際に受け取った文/Artifact/typed parametersと照合する。単純なSQL文字列類似・表示Result値の一致・モデルによる「実行した」の説明では代用しない。動的`executor.resource`を未信頼文書から無条件に起動せず、外部送信と実Runは別Host/Disclosure判断を通す。
3. **AttesterとReceiptの証拠**：実Provider/Executor由来のJob ID・実行内容・Result/Receipt location、Attester自身の実Byte/Version/実呼出し、Job IDを使った独立Result再Readを別に固定。AIが生成したReceiptの文字列、自己申告`attested=true`、同じAIが作ったExpectedによる自己評価は合格しない。存在しない`attester`/未知Runtime/未観測Jobは`unsupported/unknown/NOT RUN`。
4. **業務意味とScope**：実際に規定の計算を忠実に行ったことと、**その計算定義が業務上正しいか、NULL/部分集合/時間条件/RLS/粒度/対象集合を満たすか**は別。#2514の独立期待値と#2622/#2586のProvider Readを用い、値一致でも実Readの完全性が不明なら不足として残す。正しい計算結果の開示範囲と、内部Receiptの可視性も#2515/#2516で別に制限する。
5. **UIと再開時**：`Candidate proposed`／`Definition verified (as-of)`／`Current Source/Host qualified`／`Execution observed`／`Attestation verdict`／`Business Meaning accepted`／`Effect confirmed`は別表示。古い定義の検証履歴や旧Attestationは履歴として残すが、新しいRequest/現在Principal・Tenantの資格へ自動継承しない。

### 実験順序を増殖させない

**P0（今実施すべきこと）**：[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)の固定Participant Task・実CLI/Tool Profile/Skill bytesを維持し、**外部AI Harnessからの最初のH2 Tool-only実Tool呼出し1 Session**でTool request→実argv/cwd/exit/CLI stdout fingerprintを同Sessionに束縛。その後H2/H3を2 Sessionずつ計4回実施し、H3ではSkill rootとReferencesの**実ロード**、H2ではSkill非露出を証拠化。割付条件（R）と実提供条件（D）を分け、未ロード・誤答・Tool未観測・Warning喪失を都合よく除外しない。4件の比較から普遍的な因果効果を主張しない。実Product S1の限定成功はP0の代替ではない。

**P1（後続・初回H2/H3のHard dependencyにしない）**：[TAI #514](https://github.com/hat47x/tai-pleroma/issues/514)の既存AC-01〜12を参照して、**公開1 Concept・read-only計算1件**の`discover→load→typed bind→executor receipt→deterministic attest→independent read→Consumer`を既存TEI Evidenceへ対応付ける。正常/不正SQL/捏造Job ID/改変Result/旧Receipt/型不正/部分集合/未信頼executorの正負対照を用い、追加Case SSOTを作らない。Builder/Diagram生成やDB更新・削除など副作用へ同じAttestation契約を一般化するかはこの最小PoC後に判断。

**責務**：TEI Product Skill実装/実CLI＝[開いている#2605](https://github.com/hat47x/tei-platform/issues/2605)、外部Harness H2/H3＝TAI #168、OKF横断契約/AC＝TAI #514、独立期待値＝#2514、実入力・消費Evidence＝#1163、Meaning Owner＝#267、Host Effect＝#1174、Disclosure＝#2515/#2516。TEI #2614は**closed/duplicate**なので再開・二重実装しない。OKF Conceptのread-only閲覧と正式`*.tei.md` Lexer/Parser合格は別で、未対応Sourceは#3030/#3031の条件で止める。

**今回の証拠上限**：既存S1実Product CLI/Indexed Source成功は固定公開Source・HEADに限る。本節は追加の設計・計画統合のみで、実H2/H3外部AI Harness、OKF Executor/Attester、Job/Receipt独立再Read、実Source移行・Owner Meaning Acceptance、現在実Host/ProviderのE2Eは**NOT RUN／NOT ACCEPTED**。

## 5.10 実Harness接続の限定達成と「比較適格H2」への再設定（2026-11-07）

### 5.10.1 前節§5.9の未実施表記を限定更新

前節の「外部AI Harnessから最初の実Tool呼出し」および「実H2/H3外部Harnessは全てNOT RUN」という記述は**2026-11-07の新Evidenceより前の状態**。現在は[TEI実Evidence PR #3113](https://github.com/hat47x/tei-platform/pull/3113)（Open/未merge）と[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)から、**DSHからTEI実行物へのTool transport・実CLI起動とResult受信を1探索Sessionで限定観測済み**と訂正する。ただし**TAI SK-02の比較適格H2/H3試験は0/4で、Skill効果や独立正答は未確立**。この二つのClaimを同一の`H2 PASS`へまとめない。

| Claimの段階 | 固定した実証・証拠 | 現在の上限 |
| --- | --- | --- |
| S1 実Product CLI・Source束縛 | [TEI #2605](https://github.com/hat47x/tei-platform/issues/2605)／TAI CI #42101157330、公開Subject `sales.order`、実BinaryとSource Index fingerprint | **限定成立**。外部AI Tool利用ではない |
| H2-T 外部Harness Tool transport | [PR #3113のEvidence README](https://github.com/hat47x/tei-platform/blob/docs/dsh-h2-readonly-evidence-20261107/plan/evidence/DSH_H2_TOOL_ONLY_READ_ONLY_2026-11-07/README.md)、固定TEI `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`、DSH 0.2.2-rc.1 Windows、Participant Task digest `38a0b238…`。探索Session `h2-08`は`completed`、Tool call 56/Tool result 55、TEI実Binary起動32。CLI `commands/version/catalog/inspect/source-index/validate`を観測。実byteコピーと元worktreeでWorld/SourceIndex fingerprint一致 | **transportの限定成立**。同時Snapshotや全Source閉包、正答/権限の証明ではない |
| H2-Q 比較適格なTool-only | 物理的にSkill・独立期待値・過去回答が不可視で、Tool/CLI/モデルTaskの割付、出力と実Consumer受領、現在開示範囲を保証する一Session | **NOT QUALIFIED：0/4**。作業ディレクトリ限定は実OS拒否でなかった |
| H3 Product Skillあり | 固定Skill root+Referencesの実Loader読取とTool呼出しを、H2と同じTask/Tool Profileで観測 | **NOT RUN** |
| SK-02 比較価値 | 新Trial Groupで割付`H2→H3→H3→H2`の4独立Sessionを全件計上、独立期待値/Warning/Unknown/Token/人手/実費用を別列で採点 | **NOT RUN**。探索9回から成功Sessionを事後抽出しない |

**失敗の観測は活用するが比較の分母に混ぜない**：PR #3113の9探索AttemptはParticipant Task完了1、短い接続Task完了2、`MALFORMED_RESPONSE` 5、`INVALID_CONFIG` 1。モデルがTool call JSONを連結する問題は`maxParallelToolCalls:1`でも残った。Tool結果の成功だけで参加者Taskの完了・精度を認めない。未完了/Tool結果未受信と、実Toolの存在を別列にする。CLI `validate ok`／warning 3件はOwner Meaning・`*.tei.md`移行やRequired Input完全性を証明しない。

### 5.10.2 次の最小Exitは「比較できるH2を一件取得」

[TAI #168の既存Runbook §19〜24](https://github.com/hat47x/tai-pleroma/blob/docs/yi-harness-skills-20261102/docs/yi-constellation/first-real-skill-qualification-runbook-2026-11-02.md)をTrial/Task/Allocationの唯一の正本とし、#3113は既存探索RunのEvidence原本・限定技術的到達点、[#2509](https://github.com/hat47x/tei-platform/issues/2509)はDSH固有Profile・通信/Tool境界を所有する。新Trial Runner/基準票/Second Receipt Formatは作らない。

1. **実到達の拒否を先に計測**：参加者H2からSkill本体/Reference、Reviewer-onlyの独立期待値、旧Session answer/log、別Workspace・元Worktreeへ物理Pathを変えてアクセス試行し、**拒否が実OS/Harness/Hostで観測された**範囲だけH2隔離適格とする。H2のisolated cwd、`read-only`設定、Tool allowlist、単なるSourceコピー、未配布という説明は拒否の証拠ではない。可能なら安全な合成Canaryを使う。Windows `workspace-write`の`SetNamedSecurityInfoW failed (Win32 5)`を設定成功と扱わない。禁制Pathへ到達できる場合は正式比較を開始しない。
2. **開示と通信を別測定**：`session-log-deepseek.enabled:false`は「Request/Session Log/Telemetryの外向きbyteが0」の実証ではない。Tool stdout・stderr、DSH Request追加字段、保存Log、Child Process、ネットワークSink、CI artifactのうち実計測できたものと不明なものを[#2515](https://github.com/hat47x/tei-platform/issues/2515)/[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)へ分離。許可された公開合成Sourceの正常なProvider送信まで一律に「不正」とは扱わず、現在のAudience/目的に対する**許容範囲**を先に固定する。実OS全出口未計測なら全出口の非漏えいは未確立。
3. **Trialを事前に固定**：新Trial Group、固定Participant Task raw SHA、公開Source/実Binary/Tool schema、DSH/model/provider/請求・Credential経路、H2非Skill/H3 Skill root+Refs版、current Disclosureと独立期待値/Review窓を試行**前**に記録。探索`h2-08`の正答らしい回答やTool不正JSONを、成功した比較Trialとして再利用しない。Model/Adapter修正は別Version Groupとし、H2/H3途中で片方のみ変更しない。
4. **第一適格H2の一往復**：Harness request→OS/CLI実argv/cwd/Process identity/exit→Tool reply digest→実モデル最終回答→適格Consumerの実受領→独立Expected/Warning/Unknownの採点を同Sessionで照合。Tool call 56に対してresult 55という探索の欠落も独立観測。成功Sessionだけを残さず、事前割付された失敗・timeout・malformedをITT分母に残し、実提供D未達はPP区分へ分離。
5. **その後H3・四Session**：H3の`SKILL.md`とReferenced Filesは**実Loaderから読まれたbyte/版**が必要。H2にはロードされない物理分離を担保。同じ独立ExpectedでSkillの効果/費用を比較し、優位性を前提にしない。DSHのモデル/価格/Tool Profile結果を他Harnessへ外挿しない。

### 5.10.3 SourceとCompletionの新しい限定実装を独立に扱う

[main統合 #3051](https://github.com/hat47x/tei-platform/pull/3051)では`semantic.EvaluateCompletion`の`deny + Targets=[]`について、**Coverage未確立なら`unverified`、完全と確認できた空集合なら`fails`**とする小修正が入り、部分取得中でも明白な反例1件があれば`fails`を維持。これは§5.7の「観測0件≠Owner対象0件」を実コードへ一部反映した成果だが、`CoverageComplete`をCallerが真と申告した場合の真偽を実Reader/Providerで検査するものではない。独立に決めた対象ID・Relation・Tenant/Snapshot・必要Field/Predicate・取得ページと実Provider Readを[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#2506](https://github.com/hat47x/tei-platform/issues/2506)で照合しなければ、全件性は未立証。

[Draft #2935](https://github.com/hat47x/tei-platform/pull/2935)の`readByLoader`は`managed vs actual loader-read`という限定観測であり、Parse/Meaning/Build/Runtime actual inputは別。新`*.tei.md`のReader/ParserとSource切替後のConsumer資格は引き続き[#3030](https://github.com/hat47x/tei-platform/pull/3030)/[#3031](https://github.com/hat47x/tei-platform/pull/3031)が所有。DSL選択がPlugin Bになっても実Build/実Consumerが古いAを使用していれば無資格とし、Selected/Consumedの同一性を既存[#988](https://github.com/hat47x/tei-platform/issues/988)/#2595へ返す。

**設計の停止線**：新たに認めるのは「DSH→実TEI Binary transportの探索成功」と「Completionの未知空集合処理の限定修正」であり、**比較適格H2/H3 0/4、独立回答採点、実OS拒否・Egress不在、Product Skill Load、OKF Attester、Provider read/write、Owner Meaning受理は未達**。AIの`read-only`能力、Host Grant、Provider Effect/Receipt、Audienceへの出力許可は別の実観測が必要。今回の§5.10も計画文書のみを更新し、実コード・本番設定・新Trial実行はしない。

## 5.11 実CLI・Studio・認可・Persistenceの最新接続を証拠の段階で検収する（2026-11-07）

### 5.11.1 限定実装の進展を別Claimで更新する

前節§5.10の**実DSH→TEI Tool transport限定成功**は変えず、[Evidence PR #3113](https://github.com/hat47x/tei-platform/pull/3113)が**main統合済み**になったことを状態更新する。ただし探索9 Attempt／比較適格なH2/H3 **0/4**／独立Expected未採点のまま。以降の新実装も「接続済み」の粒度を区別して扱う。

| 既存製品面・実装 | 確認できた最大の事実 | 禁止する上位資格 |
| --- | --- | --- |
| **AI修正案の表示**：[main #3120](https://github.com/hat47x/tei-platform/pull/3120)・[Evidence #3122](https://github.com/hat47x/tei-platform/pull/3122) | StudioHost `ValidationResponse.proposals`はWorld fingerprintに結びつく未受理案を返し、`view-format`が診断/修正案（未受理）を別表で表示。限定Go/Node試験成功 | UIに表示された・`validation.Accept`の局所`Accepted`＝#267の正式Shared Acceptance/Source保存、Wails RPC実HTTP、業務Meaning/実Host Grantとしない |
| **Grantの配線**：[main #3121](https://github.com/hat47x/tei-platform/pull/3121) | 非Pluginの参照Toolへ環境Grantを宣言すると`PERMISSION_GRANT_CAPABILITY_NOT_RESOLVED`。現ValidatorはPlugin catalogから解決するため止まる | ToolはPluginと同義と決定しない。環境GrantをCLIに渡した/Toolを公開したことから実Authorityを推定しない。契約選択と正式実接続は未決 |
| **認可表示**：[main #3143](https://github.com/hat47x/tei-platform/pull/3143) | `authorizationMode`＝`reference-allow-all`／`reference-allow-list`／`external`をCLI/Studioへ投影 | `external`が実Principal・Grant/Policy版を検査したとの証明はない。参照用の許可を本番Grantへ昇格しない |
| **永続化の観測**：[main #3124/#3127/#3131/#3155](https://github.com/hat47x/tei-platform/pull/3155) | `not-applicable`／`not-observed`／`receipt-observed`／`resource-observed`を区別。独立Receipt Lookupと別pool接続の現在Resource読取が限定実PostgreSQL conformanceで確認され、`persisted:true`は`resource-observed`のときのみ | Commit成功／同Operation Receipt／現在値一致／業務完了をひとつの緑状態にしない。Observerが構成されない`not-observed`を失敗・未Commitと推定しない。独立Readでも取得時点・Grant・対象集合は別 |
| **アクセス/記録**：[main #3123/#3133/#3136/#3152](https://github.com/hat47x/tei-platform/pull/3152) | `access.FileSystem`のRoot外拒否を記録し、`os.OpenRoot`によってリンク/ジャンクション経由の脱出を拒否。Reference Toolの`ChainFile`は中途改変を検出 | このAccess APIの閉じ込め＝DSH `pwsh/read`や子Process/NetworkのOS隔離ではない。鎖はhead外部Pinなしの末尾削除、全再Hashを防げず、ログの内容の真正性も証明しない |
| **RunPlan所有世代**：[main #3139](https://github.com/hat47x/tei-platform/pull/3139) | 書込み前の`runlog.Owns`照合で旧Attemptの遅着完了記録を局所停止する | Journal読取→追記のatomicityやProvider実Effect、Concurrent Crash/Fenceの全保証ではない |
| **CLI Product contract**：[main #3148/#3160](https://github.com/hat47x/tei-platform/pull/3160)、[Dogfooding Draft #3162](https://github.com/hat47x/tei-platform/pull/3162) | `tei commands`が`modes`・`successShape`を広告。Dogfoodingの一固定実Binaryでは、CLIで実Sourceを読める一方、Workspace Identity/Manifestが応答に不足、`view effect:read-only`でも`-execute -operation-log`や`-lifecycle-state`が作業ファイルを書いた | Commandの広告だけで応答Schema・対象Workspace・副作用なしを確定しない。CLI単体の実測を全Tool Profile/DSH・実Providerの不作用へ継承しない。Draft #3162はCI/正式受理前 |
| **Model Source版**：[main #3163](https://github.com/hat47x/tei-platform/pull/3163) | `modelRevision`は管理Source Indexのbyte由来の版でありWorld版ではない。Prelude-onlyの空Workspaceでも`model catalog ok:true`とstd entriesがあり得る。実宣言の有無は`declarationCounts`を確認 | ModelRevision一致・Catalog非空・validate okを必要User Subject/Source closureの実Parsed/Consumed成功へ昇格しない |

**開発自律性と製品受理を区別**：[採用済みADR-0035／main #3150](https://github.com/hat47x/tei-platform/pull/3150)の方針では、リポジトリ内PoC・固定Fixture/文書/参照検証は機械照合できるA3の範囲で自律進行でき、**外部に取り消せない効果**（本番配備、秘密送出、削除・破壊的移行、対外公開/第三者ライセンス等）はA4として人へ確認する。これを「製品の業務Owner MeaningがAI生成内容で自動受理された」とは解釈しない。#267のShared Acceptance／外部Host Authorityは**製品側の資格**であり、PoC工程の全変更に人的Blocking Reviewを新設しない。変更時にADR-0035のPoC工程と、対象製品の適用資格/実効果を別証拠列で提示する。

### 5.11.2 DSH Mock egressの実測で次のH2適格条件を修正

[main Evidence #3153](https://github.com/hat47x/tei-platform/pull/3153)と[main Evidence #3161](https://github.com/hat47x/tei-platform/pull/3161)は、**DSH 0.2.2-rc.1／ローカル127.0.0.1 Mock Endpoint／ダミーCredential**で次を実観測した：

- `session-log-deepseek.enabled:true`なら要求の`dsh_session_log`に約45 KB/19 eventが含まれる。Toolの合成markerは**モデル入力1箇所に加えてSession Logの2箇所**（`tool/result`の`data.message`と`data.meta`）へ投影された。これはモデル入力のMaskだけでは防げない第二の送出経路。
- 同じSessionを`enabled:false`で再開した間は当該fieldが無かったが、`true`へ戻すと止めた区間のeventsが**再送された**。既定Profile（`--patch`なし）でもSession Log fieldが載った。
- Mockが503を返した場合、同じ水位（`afterSeq`/ `throughSeq`）の約45KBが次要求で再送された。200応答でも`message_stop`なしのStream切断では水位が進まず再送した。**disabled once＝過去ログが消える／送信されない、という推論は禁止**。
- これらは**Mockの限定wire挙動**であり、実Providerでの保存、全送信先/Telemetry、in-flight、8MiB超の分割、実OS Egress拒否・公開Policy資格にはまだ昇格しない。Session LogのRaw本文を公開Artifactへ載せず、内部Audienceに保存した原本と公開した`field/size/hash`投影を別管理する（#2515/#2516/TAI #232）。

**次のH2比較適格試験の必要条件**：
1. 固定Participant Task/TEI binary/Workspace/Tool allowlist/DSH Version/Model/請求経路を事前Pinし、Reviewer-only独立期待値・Skill原文・旧回答へ**実Toolから拒否された**ことをOS/Harness側で観測。TEIの`access.FileSystem`だけでDSH実Processを閉じ込めたとはしない。
2. 実送出の許可対象（公開合成入力と許可されたPrompt等）・Audienceを用途ごとに明示し、`dsh_session_log`の既定ON、Off→On backlog、503/切断再送による**複数Request/複数Sink**を測れるProfile/Mockと実OS拒否の必要範囲を選ぶ。外部Providerへ秘密を送って実験しない。
3. 比較用の新Trial GroupのH2→H3→H3→H2は[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168) Runbookのみが正本。探索9 Attemptは比較試験へ事後混入しない。Tool-onlyとSkillありで使用するTool/Skill/Source実Byte・Permission/Profileを同一条件にし、Skillそのものの差以外で要件を変えない。
4. H2の`View/Inspect/Validate`という自然言語・広告上のread-onlyと、**実`argv`・暗黙のローカル書込み・外部送信**を区別。初回は`-execute`など不要なAction候補をTool側で提供せず、実Tool commandsの型付きAllowlistを使う。Allowlist設置だけで実OS/Host拒否が成立するとはしない。
5. Source/Model Revision、実Parsed/Actual Consumed、Return JSON successShape/mode、Stdout/Stderr/exit、Answer/Consumer受信、独立Expected、Unknown・警告を同一Sessionへ束縛する。Studio Proposal表示や実CLI Modelがあることだけで意味受理・Action資格は認めない。

**Ownerへ渡すべき未決**：外部Tool GrantをPlugin catalogに束縛するか（#3121/ #951）、`external` Authorizerの実根拠をどこまでConsumerへ公開するか（#3143/#1174/#2516）、`receipt-observed`／`not-observed`のUI警告・再照会停止条件（#3155/#1174）、Tool/OS Sandbox実拒否の最小実装（#2515/TAI #232）。別の中央Host/Registry、AI独自承認Engine、Receipt SSOTを先に作らない。

**証拠水準**：上記のmain局所実装・DSH Mock wire・PR #3113探索を**各Source/HEADの限定成果**として採用。新たなH2/H3比較適格Session、実OS/Harness全面隔離、実ProviderへのEgress、Studio Wails end-to-end、外部Host current Grant、`*.tei.md`実Reader/Owner業務Meaning/DB業務完了は**NOT RUN／NOT QUALIFIED**。本節は既存Ownerへの統合計画であり、新Schema・新Runtime・新コードを追加しない。

## 6. 採否と現在の限界

- **Owner方針として確認済み**：外部要件の十分な机上評価を基盤実装に先行させること、CUIと標準Pluginを並行開発すること、ヘッドレス実証を基準にGUIを後続実装すること。
- **本書の設計候補**：七つの評価観点、P1〜P10の代表圧力、E0〜E4の実証整理、AI認知負荷の比較方法。
- **既存実装の一部**：CLIの読み取り操作、特定のDriver/Plugin/Targetの参照試験等は存在する。製品としての統合モデラー、全用途のCLI操作、横断的な実Provider/Target資格は未完成（[実装状況](../../docs/status.md)）。
- **今回未実行**：机上要件全件の照合、P1〜P10の統合テスト、新たな標準Plugin・CUI機能の実装、実DBや実Target、GUI操作、Owner受理。既存の実測がある場合もその版と対象に限定する。

十分な反例または実証が得られた内容のみ既存の製品要求・ADR・Issueへ差分として取り込み、機構やGateの増殖を避ける。
