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

## 6. 採否と現在の限界

- **Owner方針として確認済み**：外部要件の十分な机上評価を基盤実装に先行させること、CUIと標準Pluginを並行開発すること、ヘッドレス実証を基準にGUIを後続実装すること。
- **本書の設計候補**：七つの評価観点、P1〜P10の代表圧力、E0〜E4の実証整理、AI認知負荷の比較方法。
- **既存実装の一部**：CLIの読み取り操作、特定のDriver/Plugin/Targetの参照試験等は存在する。製品としての統合モデラー、全用途のCLI操作、横断的な実Provider/Target資格は未完成（[実装状況](../../docs/status.md)）。
- **今回未実行**：机上要件全件の照合、P1〜P10の統合テスト、新たな標準Plugin・CUI機能の実装、実DBや実Target、GUI操作、Owner受理。既存の実測がある場合もその版と対象に限定する。

十分な反例または実証が得られた内容のみ既存の製品要求・ADR・Issueへ差分として取り込み、機構やGateの増殖を避ける。
