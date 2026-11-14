# TEIの開発全体計画

## 目的

この文書は、TEI（鼎／テイ）の製品価値と要求を開発上の作業領域へ整理し、優先する作業と並行できる作業を示します。

期限は設けません。優先順位は実装量ではなく、**TEIの中心価値に関する不確実性をどれだけ減らせるか**を基準に見直します。

過去の判断を本文に重ねて記すのではなく、現在の計画の基準を示します。以前の判断とその理由は、Gitの履歴、設計判断記録、課題からたどれます。

関連文書は次のとおりです。

- [Studio／CLI／WebのBackend子プロセス・Plugin管理・配布更新の統合設計](design/STUDIO_CLI_BACKEND_PROCESS_DISTRIBUTION_UPDATE_2026-11.md)：DesktopはWails親からローカル子BackendへIPC接続する前提を置き、CLIはWails非依存の共通Backendを直接／自己子起動／再利用Sessionで比較する。WebはServer配信Clientと認証済みAPIを分離し、遠隔CLIはServer側のPlugin構成を使用する。G0〜G6は既存#1158/#1824/#2044/#2589/#2831/#4583の実装Ownerに接続し、**CLI子起動の既定化・共有Daemon・Updater製品採用は実測とOwner判断待ち**とする。Studio Stage1→Stage2セルフビルドのP0を変更しない。

- [横断プラグインの役割別構成・実行境界・Studio自己再構築](design/PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)：**Studio Desktopを正式な非Plugin Build対象としてStage1→Stage2→実Windows WebView2まで検証することをP0**とし、通常CLI／Studioから選択した実Providerへの到達を並行P0とする。Build時にだけ必要なGeneratorと生成製品に収録するProviderを分け、Lifecycle Graphを既存Operation／Hookに接続する（§1〜12）。**§13では構文・意味・演算・データ・表示・実行・検証の貢献別選択、Source／Anchor／Graphの往復編集、変更内容に応じた局所再検証、Studio／AIの対象固定と現在資格を整理**し、PCX-01〜10を既存#1180/#2514の独立Expected候補に引き渡す。**§14では必要Contract→exact Provider選択→Trait履行→工程・Target別の依存集合→実Builder／Provider→Consumer受領を、既存ResolverとBuildへ接続する。** Verified Composition BOM（RQ-037）、静的Go標準と外部Process／Wasm／Native配布の比較、DSL自己記述B0〜B4、PCC-01〜10の検証候補を加え、P0のセルフビルドを妨げない。新Grammar Engine／Resolver／Host／ABIの採用、実DB Effect、Owner意味受理は未実施。 **§15では依存先が契約で公開した図解・Relation・Obligationを、再公開許可・出典・版・現在開示の範囲で継承する。§16ではCLI/MCPからTEI定義→実コード生成→Plugin自身の生成割合拡大を先行実証し、正式Desktop Stage2の合格条件とは区別する**（PX-38〜47へ接続）。 **§17〜18ではApplication Graphに基づく横断拡張の対象選択と、Plugin説明・上位依存・短いAI Skillから現行Contractを辿る入口を整理する。**

- [Script/Function起動・Generator・セルフビルドの接続計画](design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)：**StudioのStage1→Stage2自己再構築を全体P0**とし、#4495の製品pure Function Entry→Host選択の実Go Provider→通常`tei` CLIと独立Consumerの縦断を並行P0とする。#4629/#4636でCapability宣言の実行前構造拒否はmain統合済み。追加の[§20 DSL/Workspace/Runtime四層診断](design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md#20-dsl-authoring診断と実行前宣言検査を一つの不正定義で照合する2026-11-14)は[#4652](https://github.com/hat47x/tei-platform/pull/4652)でmain統合済みだが、Workspace重複Input診断[#4655](https://github.com/hat47x/tei-platform/pull/4655)とSource→Adapterの回帰[#4654](https://github.com/hat47x/tei-platform/pull/4654)は別のGo Draftであり製品Entry受理ではない。実ReadはRA-0[#4626](https://github.com/hat47x/tei-platform/pull/4626)（17/18成功・Browser実APIの`sqlgen`依存除外FAIL）→RA-1/RA-2R統合検査[#4644](https://github.com/hat47x/tei-platform/pull/4644)（限定3/3成功・実PG16 Reference）→RA-2P製品Read→RA-3独立Consumerの順。取得範囲[#4647](https://github.com/hat47x/tei-platform/pull/4647)は観測された制限を`partial`にできてもNative SQLの完全性を保証せず、生成CLI[#4559](https://github.com/hat47x/tei-platform/pull/4559)は旧Sourceの未宣言入力を補うReferenceにとどまる。Studio[#4587](https://github.com/hat47x/tei-platform/pull/4587)は最新HEADで生成・通常Windows WebView2の両Jobが成功（B0、Draft）したがStage2ではない。Toolchain[#4648](https://github.com/hat47x/tei-platform/pull/4648)とState[#4537](https://github.com/hat47x/tei-platform/pull/4537)の別HEAD各8/8もStage2へ加算しない。**上位の完了条件は[実Consumer受領の横断設計](design/CONSUMER_QUALIFIED_CROSS_LANE_ACCEPTANCE_2026-11-14.md)に従い、Source・Authority・実Call・取得範囲・開示・Consumerを別Claimで検収する**。
- **2026-11-14 Script Launch実装接合**：[Script Launch計画§21〜25](design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)で、Function/Entry/TriggerとExecution Bindingの分担、実行前の八検査、Provider不適合時の拒否、RDB/NoSQLの入力完備性とReceipt、実Go Provider/通常tei CLIの独立Witnessを明確化する。main統合済みの[Process Lifecycle設計](design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)と[実行モデルの図解](design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)を再利用し、新しいHost・Registry・Scheduler・Schemaは追加しない。**優先順位は、初回の意味実装・自己構築＝TEI CLI（MCPは任意）から*.tei.mdの図表・Rule→実Codegen→実CLI動作→Stage1 CLIによるStage2 CLI再構築とPlugin実Call、製品全体の到達目標＝Studio Stage1→Stage2 Windows実再構築、並行する起動P0＝#4495公開pure Entry→Host-selected actual Go Provider→型付きAPI／通常製品CLI、後続＝#2586実Read/Consumer・#4497 Trigger・#1174 Effect。Studio GUIをCLI初回実証の前提にしない。** 追加内容は検証計画であり、実ProviderやOwner業務意味の新たな受理ではない。 **§23では実コードに基づきAdapter呼出し前の拒否と呼出し後のResult契約違反を区別**し、後者を未実行・Rollback・安全なRetryとみなさない。入力値／Output Payloadの意味検証、Effect Receipt、実Consumer受領は既存Ownerに残し、#4495の最小実Call証拠と#1174の回復資格で別々に検収する。 **§24は#2831/#4675/#4676のCLI先行・プラグインTEI主導生成・定義の簡素化の計画と接続**し、Goへの委譲だけ、偽のStage1再入、上位Pluginの非公開図面の誤継承を合格にしない。上位契約から公開された図解・制約は参照可能だが、実行GrantとSource編集権限は継承しない。 **§25は現行製品CLIの実参照経路を棚卸しし、Product実Callとの区別を実装作業に固定**する。tei view -executeはReferenceExecutor/MemoryUnitOfWork、tei self-buildは子Process/Artifact観測であって、公開Entry→actual Go ProviderやTEI主導CLI Stage1→Stage2の合格に昇格させない。#4495の型付きAPI→実Providerを先に閉じ、通常製品tei別Processを次の差分にする。既存Failure Contractを保持し、Adapter後Result違反やEffect unknownを無条件Retryにしない。
- [標準データ構造・構文・図解・編集の意味保存](design/SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)：構造／評価／変更の同値性、Graph／Tree／Table・JSON Schema・複合Projectionの損失、Slot別編集資格を既存#2495/#2497/#4122へ接続する。Definition／Effective／Observedの型付きGraph、横断Hook、Plugin文書参照は#4198/#951/#1124へ渡す。**`*.tei.md`は通常Markdownとして可読**を基本条件とし、階層は入れ子リスト、小表はMarkdownテーブル、大表はCSV／TSVへの相対リンク、文書が大きければファイル分割を標準候補とする。ER／シーケンス／Flow等はGitHub対応Mermaidを**直接のAuthoring Source**にできる範囲と、型付きSourceから**導出図**とする範囲を区別し、意味の二重正本を作らない。**渡辺式ER図は用途別View**で、**親を左・子を右、同じ段階を同列で上下に整列**する。横位置は確認した親子関係から決め、PK継承の識別関係とFKのみの非識別参照を異なる線で示す。複数親は子の左辺へ複数線を結ぶ。箱内はEntity名の右側に項目をカンマ区切りで表示し、主キーは下線、二次識別子は`{...}`、導出／仮想項目は`(...)`とする。Mermaid記述の初出順は主に**同じ列の上下順**に用い、Studioの初期配置は同一Source／Layout Profileで決定論的に再現する。GitHubではMermaid Renderer自身の配置に従い座標一致を要求しない。将来の同列GUI並替えは意味を保つ局所Source Patch候補、自由な座標移動はView Hintの別検討とする。#3030／#2621／#4370へMD-0〜MD-4、SM-01〜70を既存#1180/#2514の独立Expected候補として渡す。新Graph Compiler／Source Writer／固定座標Schemaは先行採用せず、**全体P0のStudio Stage1→Stage2セルフビルドと通常`tei` CLI→実Provider→独立Consumerを優先**する。
- [数理・統計・計画・科学向けツールキットの設計・検証計画](design/CROSS_DOMAIN_MATH_STATISTICS_PLANNING_TOOLKITS_2026-11.md)：ウィンドウ集計と帳票、WBS/PERT/CPM、Solver・実験計画法、数式・物理系・生命科学を題材に、既存のDSL・標準表現型・Operator・Providerの適用範囲を確かめる（[#4660](https://github.com/hat47x/tei-platform/issues/4660)）。独立した期待結果と、Mockから実際の計算Providerへ移る際の受入条件を先に定義する。TEI Studioと一般アプリケーションの双方で再利用できる設計を目指すが、Studio DesktopのStage1→Stage2セルフビルドを最優先とする方針は変えない。
- [汎用プログラム実行モデルの図解・Target間移植](design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)：Bootstrap／Trigger／Entry／Operation／Effectを型付き関係と実行規則から投影し、意味・実効計画・実Trace・図面を分離する設計案（#4198／#4502／#4193）。§16〜19では処理分割・言語移行の契約保存と局所再検証、§20〜23ではSource・静的解析・実Trace・AI推定の対応資格、Projectionの情報損失、Target同等性の観測範囲、SQL/NoSQLの粒度・取得条件、PX-30〜37を既存Ownerへ引き渡す。**Studio Stage1→Stage2と実Entry→実ProviderをP0**とし、新Graph Compiler・Host・Runtime・Hard Gateは先行導入しない。実Target検収と正式意味受理は未実行・未採用。 **§24〜27では上位Pluginの公開Facet・継承の情報境界、差分Graph合成とObligation維持、TEI-native Plugin自己生成のHeadless実証、PX-38〜47を既存Ownerへ引き渡す。** それぞれの実装・検証・正式採用は未完了。 **§28〜32でApp/Route/Entry/Operationの全体Graph、任意計測と必須監査の差、Target/Consentごとの適用、CLI・Studio・Harnessの対象固定、PX-48〜58を既存Issueへ接続する。** **§33〜36では定義・選択・実現・実観測を同一Subject・Target・Attemptで照合し、問いに応じたGraph完全性、CLI/Studio/HarnessのInstruction・Subject・Capability三閉包、Markdown優先の未実装Readerの扱い、最初の一枚のread-only検証票を既存Ownerへ引き渡す。**
- [理想の利用像を製品の受入へつなぐ計画](workstreams/PRODUCT_COMPLETION_PLAN.md)と[製品設計](design/PRODUCT_COMPLETION_DESIGN.md)は、設計書の編集、実接続、標準構成、帳票、複合表現、分野拡張、運用・費用をG01〜G71の作業へ結ぶ。最新の型・表現、DSL、ETL、プラグイン管理の段階計画へ接続し、申請等の一巡から添付、検索、通知、外部契約、継続イベント、デバッグ、利用者の管理を補う。評価は [#3796](https://github.com/hat47x/tei-platform/issues/3796)と各担当Issueで行い、既存のヘッドレス実証からGUIへ進む順序と、実Readを含む現在の主線を保つ。
- [Issue #2586](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6773141846) ：Plugin・Function実Read・三視点Model Map・実Effect・Build・Consumerを、**宣言／実選択／実入力／実呼出し／受領／現在資格**で区別する横断受入計画。§9はS2-reference（#3540）・限定L2-b（#3785/#3795）・限定L3・Hookの実績を反映。**S2-product／Function R0／全体G4は未資格**。各担当は#1251/#2586/#2831/#1124/#2600等を維持し、新Gate/第二正本を設けない。
- [Lifecycle Graph・AI Harness・三視点Modelerの統合資格とS0〜S2実装引渡し](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md#23-lifecycle-graphai-harness三視点modelerの統合資格2026-11-12) ：§23〜32.5で固定Selection Context、Graph Patchの構造/意味/実行資格、Source/Plugin/Artifact/EffectのEvidence Join、既存Owner別A〜Eバックログ、**テーブル列削除→モデル写像削除・UIは未マッピング警告（§27.7の六診断状態・§27.8のCRUDレビュー資格分離）**、**Function/Script/Entry/Trigger/実Artifactまでの失効伝播（§27.6、#4502/#4495）**、三要素分析法のMethod/Plugin化、**通常開発相当の意味レビューとAI候補の出典確認**、**観測参照0件≠未使用という問い相対の判定**、**Graph→Generatorの適格性をG1規約→G3独立Conformance→G2合成へ分割（§30、#4544/#4198）**、**#4539実PG Readのmain統合と製品`tei` CLI→Studioへの残差（§31）**、**Studio Desktopの生成部品実使用P0-A→TEI Stage1実ProcessによるStage2起動P0-B→X-TEA三要素Plugin化P0-C、DS-S0〜S4の受入（§32、#1824/#2831/#1124）**、**Source Indexからの高速参照移動とPin→AI Preview（§29）**、**§29.6の参照ナビゲーション・近傍直接テスト・実Entry起動・実Read・Effectの証拠を分離した統合受入（#4543/#4502）**を扱う。**#2586の横断受入、#1251の実Consumer、#1124/#1126のModeler/Schema、#3503のLifecycle、#1180/#2514のScenario、#267/#4143のAI/OKF契約を置き換えない**。S0〜S2、DMU-01〜15、NAV-01〜10、#4472 TE-CR-01〜06との交差条件、および#4502/#4495の既存起動負例との交差照合は検証用Reference Slice/独立Expected候補。局所合成実績とS2-product/Function R0/全体G4の未資格を区別し、汎用Coreや通常コーディングに三要素の必須制約を加えない。§29のP1a限定修正（#4392/#4524）はmainマージ済み。実Consumerは未実証。§29の実装順はP0実Provider/実Readを維持し、P1a Map/Locate/曖昧IDの整合（#4392/#4524）→P1b 一件のread-only参照往復（#4543の近傍直接テスト照合は並行可能）→P1c Pin送出Preview（#4479/#2516）→P2安全な逆編集→P3実Workspaceの性能・認知負担測定とする。**§32.5では新しいSource束縛Function生成Draft #4559の限定実績と残差を確認し、最優先D-W（実Wailsで生成部品を実利用）・独立F-B1（PG read-only Function生成/手書きReference比較）・並行B-S（Stage1自身によるStage2再入場）を分離。** 現行#4559はDraft/未統合、入力fromは未宣言、SQLはSource Index対象外、生成READ ONLYの実書込拒否と変異生成器実検出は未確認。D-Wの実証は**既存のPoC B生成Workbench→generated server/Chromium**と**通常Windows Desktop→WebView2**（さらにPoC A CIのgenerated Windows exe Buildのみ）という分かれた基線を、同じSource/Generator/実Build/実Windows WebView2 Consumerへ結ぶ**未接続の一辺**に限定する。既存compose-ui/connect-workbench等を再利用し、生成Assetの未import・手書き同値代替・serverのみの合格・旧Workspace後着を独立負例にする。#4559の実Rowset一致をEnvironment実BindingやStudio実証へ昇格しない。OwnerはD-W=#1824/#1121、F-B1=#1121/#2586、B-S=#2831の既存Issueのまま。文書のみ・通常CLI/Studio/Stage1→Stage2・X-TEA移植実接続、DS-S0〜S4/NAV/Script/Generator GG/D-W/性能の新試験NOT RUN。
- `product/`
- `requirements/`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/MILESTONES.md`
- `plan/research/TECH_RESEARCH_INTEGRATION_2026-10.md` ：直近の調査結果を製品価値や既存の課題に結び付ける統合計画
- [DeepSeek Harnessに学ぶ構成・AI-first連携・横断検証（研究案）](research/DEEPSEEK_HARNESS_TEI_CROSS_PRODUCT_ADOPTION_2026-11-01.md) ：外側のAgent HarnessとTEIのService Contract/Plugin/Host Authorityの境界を比較。正式要件・実実行の受理は既存担当へ
- [DSHのGoal・Schedule・通信境界と大規模AI並列開発（研究案）](research/DSH_SCALE_GOAL_SCHEDULE_EGRESS_RESEARCH_2026-11-01.md) ：軽量Profileの自律性、独立Workstream、既定Request拡張によるSession履歴送信、検証責務を明示。実証担当は[#2509](https://github.com/hat47x/tei-platform/issues/2509)／[TAI #234](https://github.com/hat47x/tai-pleroma/issues/234)。
- [DSH実験的Agent Teamsの並列開発・専門家連携・用途別Context（研究案）](research/DSH_AGENT_TEAMS_CONTEXT_EXPERT_HANDOFF_2026-11-02.md) ：Team Task DAG/CAS/共有Checkout・非強制writeScope、SEIの専門家照会、#2515実効隔離・#2516用途別Projectionを統合したT-01〜T-12反証案。既存Ownerと受理を維持する。
- [外部Harnessの結果継承と選択的再資格化（2026-11-02研究案）](research/HARNESS_EVIDENCE_CONTINUITY_AND_SELECTIVE_REQUALIFICATION_2026-11-02.md) ：DSH Headless NDJSONの投影/截断/Session再開と実Git/TEI/Oracle Database/Providerの証拠の差を識別。#2514独立検証基準、#2582 Reactive、#2595実Source、SEI #767の用途別根拠十分性に接続し、Claim単位で局所再評価する。
- `plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md` ：成熟した外部の参照モデルをプラグイン中心で移し替え、TEIの表現力や暗黙の前提を横断的に検証する計画
- `plan/design/ARTIFACT_SET_SCOPE_AND_CONSUMER_READ_EVIDENCE_2026-11.md` ：導出成果物の対象集合・内部整合・公開世代・実Consumer読取を分けて検証する**提案中**の計画（#2482／#2604。製品Gateや受理条件の変更ではない）
- `plan/workstreams/TRIAD_MEANING_AND_CROSSLANE_ACCEPTANCE_HANDOFF_2026-11.md` ：Data・Function・Businessの意味を保ち、移植・DB移行・組織Relation・Plugin境界における既存Ownerの検収条件を照合する**提案中**の横断引渡し（#1125／#1126／#1129／#2619。新Gate・業務意味の採用ではない）

- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md`：AI Tuning Hint/限定合成観測Reference、性能要求×Workload×実現×独立Evidence、Typed Lifecycle/Plugin/Cache/Query/Headless比較（§1〜19）。§20でCacheの速度・費用・正しさ、Commit/Hook/Outbox/CDCと失効の区別、AT-33〜36を定義。§21で一FunctionのNo Cache対Request-local/Memory（必要時Query/Batch）をW1〜W8の同一Workload/Expectedで比較。§22でH0（AIなし）/H1（DSH等外部Harness）/H2（別Harness/Provider）の任意接続とTool完全受領・開示・費用の別資格を定義。**§23（2026-11-12）**でStudio/CUIのFocus非依存の固定Selection（Qualified ID/Source/Revision/Scope/Grant）と、Product別権限を引き継がない共有Harness Session、stale時の明示再選択、Tool欠落/未承認Apply負例を接続。#2586の実Function→実PG→Rowset→ConsumerをP0とし、固定Selection/AI送出は後続。**§24（2026-11-12）**でGitに小さなManifest/Expected、Raw Trace等はLocal CAS/NAS/S3へ選択保管し、Identity/Storage Resolution/Qualification/Disclosure/Lifecycleを分離。EV-01〜08は既存Scenarioへの反例候補で、Local-onlyを正規経路とし、外部HarnessへのRaw Evidence自動送出を禁止。**§25（2026-11-13）**でDB Read Snapshot/Subject Pin/Row Edit Selection/Pending Changes/Commit Attempt/Receipt/独立Post-commit Read/Cache Projectionを分離。R0実Product PG Read→R1 Headless読取/Pin→R2単一行条件付き更新→R3独立Read/Cache失効→R4任意AI Harnessの順とし、RW-01〜08を既存Scenarioの未実行反例候補へ接続。Read-only A/Bは編集完了を一律前提とせず、更新鮮度のClaimにはR3相当の実証を要求。既存AT-01〜36、T0〜T4、#2586/#3262/#3271/#1180/#1452/#1824をOwnerとし、新Host/Registry/Hard Gateを追加しない。実Read E2E、Cache性能、実GUI/CUI固定選択、外部Harness完全受領、Owner受理は **NOT RUN / NOT ACCEPTED**。
- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md` ：生成AI向け設計ヒント・19件の限定合成観測Reference（§9〜10）、性能要求×Workload×実現×独立Evidence、Typed Lifecycle、Plugin/Cache/Query/Headless比較（§11〜18）を維持。**§19（2026-11-09）**で#2586のM1b→R0（Function→Query/Relation→実Provider→Rowset→Consumer）を最優先の一Attemptに固定し、merged #3689/#3761のHook実位相/Outcome検査、merged #3757のBinding実呼出し範囲、merged #3717の実PostgreSQL試験用CI設定を各資格上限付きで統合。CacheのTTL/Hook/Build成功≠鮮度・実Read・性能改善を維持する。旧版に混入した重複見出し/旧T0〜T4表を正規化し§1〜19へ整理。**§20（2026-11-09）**でM1b→R0後の一Function A=No Cache/B=Request-localまたはMemoryの再現可能な性能実験（W1〜W8）、独立Expected・実Call・p95/p99/Origin負荷・測定費の二段階採否、必要時のC=Query/Batch改善対照を定義。E0〜E4は既存T0〜T4の内訳であり実測未実施。**§21**ではStudioのUI要素選択→用途限定Design Context→DSH等外部Harness→AI改善候補→TEI独立検証をH0（AIなし）/H1（外部Harness）/H2（別Harness/Provider）で比較。外部送出/Tool完全受領/OS隔離/契約上の接続可否/コストを別資格とし、AIを必須Runtime化しない。AT-01〜32と既存T0〜T4のOwnerを継続し、新Optimizer/Core/Host/Hard Gateを設けない。実Function→実DB、キャッシュ負荷計測、AI自動適用、正式Owner受理は **NOT RUN / NOT ACCEPTED**。
- [高度B2C向けHeadless・Conversion Engineering設計（Issue #3509）](https://github.com/hat47x/tei-platform/issues/3509) ：独自Frontend/標準表現型の任意選択、Journey/Experiment/Effectの別証拠、販促変更レーン、実Consumer受入と反例を扱う**提案中の設計・計画**。旧PR #3512の全文は[Issue内に保存](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6772845918)し、設計書を第二正本として復活させない。新Runtime/Commerce EngineやCVR改善の実証を意味しない。
- `plan/design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md` ：原子的Traitの実装義務、Manual/Generated/Providerでの履行、合成・不適合診断を既存#951/#2495/#3271/#3267へ接合する**設計・実証候補**（[#3278](https://github.com/hat47x/tei-platform/issues/3278)）。新Resolver/新Hard Gateや実装完了を意味しない。
- `plan/design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md` ：Web/Desktop APIの例外・Client Reaction・Graph/Table/Traceの**横断設計と段階的な実証計画**（[#3274](https://github.com/hat47x/tei-platform/issues/3274)）。**P-1a/b**：E1の構成時Guard DAGとRuntime HTTP Decision Pathを区別し、[実Node専用CI #42396614454](https://github.com/hat47x/tei-platform/actions/runs/42396614454)の限定HDG-21 CharacterizationはPASS。全体Healthは同PRでgofmt不一致（別Source）によりFAIL、Go vet/test/Workspace validateはPASS。**P-1c**：既存Go Reference Hostのdefer Auditと10経路Callback/Status/Stageを、[Node+Go専用CI #42461546535](https://github.com/hat47x/tei-platform/actions/runs/42461546535)で限定PASS。**P0-R（限定実証）**：E1・Goの独立宣言は両側が局所的に構成可能でも不一致になることを[Node/Go実CI #42462531490](https://github.com/hat47x/tei-platform/actions/runs/42462531490)で確認（Node 7/7＋Go package全PASS）。**P0-S（限定CI合格）**：実`reference/workspace`の`application-pattern:order_entry.submit_order`と`semantic-capability:sales.order.submit`・Source Index/Canonical参照を既存`operations.InspectSubject`で検査し、[Node/Go専用CI #42464774706](https://github.com/hat47x/tei-platform/actions/runs/42464774706)＋同HEAD全体Health #42464774715でPASS。**P0-B（限定CI合格）**：既存`driver.ResolveCommandBinding`を実Workspaceで通し、欠落・重複・有効だが意味的に違うCapabilityの負例を[専用CI #42470312187](https://github.com/hat47x/tei-platform/actions/runs/42470312187)でPASS（同HEAD全体HealthもPASS）。**P0-I（限定CI合格）**：実`NewHost`・`NewCanonicalActionAuthorizer`・実Driver`InvokeCapability`から合成`Capability Adapter.Invoke`までを一つのHTTP Attemptで通し、Session/CSRF/認可拒否・Adapter障害・Result契約違反を[専用CI #42474818698](https://github.com/hat47x/tei-platform/actions/runs/42474818698)でGo/Node PASS。**同HEADの全体Health #42474818899はfailure**（当時`plan/design/README.md`に並行追加AI性能設計書の目次登録なし。現main目次には登録済み）。**P0-E（限定CI合格）**：同一実Host＋実Driver＋合成Adapterで、202/503と合成状態の5つの反例・明示Observation readの失敗を[専用CI #42486408608](https://github.com/hat47x/tei-platform/actions/runs/42486408608)でPASS（Node 7/7＋Go全PASS）。**同HEADの全体Health #42486408590はfailure**：全体gofmt/vet/test/Workspace validateはPASSだが当時の別AI性能観測Workflowが未ピン留めaction・timeout欠落でCI契約違反。**現mainのWorkflowは修正済み**（別HEADであり過去CIをsuccessに変更しない）。HTTP status／Audit／正常Result≠永続業務効果、read natureはsynthetic、外部Receipt・Client完全受領は未実証。**上流Composition接続判断優先**：[判断資料 PR #3549](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)のC-4（実行を外部Hostへ委任する現状維持）を含め、C-4→C-1/C-2→C-3の順で製品実行経路を定めるまでHTTP専用Registry/Routerを増設しない（[設計§20](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）。**C-1参照実行の成熟差分（PR #3573経由）**：Environment Binding→既存`routing.Router.Dispatch`→Host登録の合成Executorという技術的成立を確認したが、製品CLI/実外部Providerへは未接続。新しい[設計§21](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)と[Composition判断資料§9](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)では、Executor入場後のエラー／不正ACKを`refusedCount`（入場前拒否）に数えない**Reference v1補正**と合成負例BXE-01〜06を追加し、`unverifiedCount`（`ranCount`の内数）として保持する。[限定Go CI #42505232348](https://github.com/hat47x/tei-platform/actions/runs/42505232348)で実`Router.Dispatch`の6反例・既存Router回帰・Go vet/gofmtがPASS（v1 Reference限定）。**C-1/C-2は別のOperation役割**：配備Command（C-1）とService Request（C-2）は独立し得るため、二者択一を一律強制しない。同一Operationで実行Ownerを重複させず、C-4外部Host委譲も対象／Targetごとに未決。C-2参照はCarrier入場とProvider Process実呼出しを分離し、`Schema/Rule/RuleFingerprint`をv1として保存済みv0の版を流用しない。[C-1/C-2限定CI #42555954660](https://github.com/hat47x/tei-platform/actions/runs/42555954660)で両ReferenceのGo正負・vet/gofmtがPASS（[判断資料§11](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)／[HTTP設計§22](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）。**両案とも製品への採用判断は未決**であり、HTTP 202/503・Executor/Carrier ACK・実Effect Receiptを一つの成功/失敗へ畳まない。**横断S2の現況と次のGate**：[main統合PR #3540](https://github.com/hat47x/tei-platform/pull/3540)で通常CLI `inspect`と通常・生成Studio Hostから**静的Reference Inspector Provider**を既存Composition経由で選ぶS2-referenceが成立し、exact-head Health/Wails/WebView/CLIはPASS。一方、正式Profileで**Provider B選択→両実ConsumerがB実呼出し**するS2-product、Plugin実Artifact/Entry/Current Grant/2×2 BOMは未検証。HTTP側はこの残差を先行Gateとし、結果分類・Effect unknownをドッグフーディングする。**11/09成熟差分（[HTTP設計§23](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）**：C-3 [#3692](https://github.com/hat47x/tei-platform/pull/3692)/[#3720](https://github.com/hat47x/tei-platform/pull/3720)はBinding由来Executor IDを既存Route/BuildPlan/Supportsで解決した**実行前Reference**（v0）であり、C-1/C-2 v1より遅れた未実行機能ではない。固定IDと実解決IDが異なる反例を保持する。Hook [#3721](https://github.com/hat47x/tei-platform/pull/3721)は`NotSatisfied`→Command拒否語彙を限定対応し、`not-satisfied`と技術ErrorをCommand拒否へ昇格しない。外部Plugin語彙検査、CLI/HTTP Clientへの実連動は未実証。実PostgreSQL [#3717](https://github.com/hat47x/tei-platform/pull/3717)の[専用CI #42570360880](https://github.com/hat47x/tei-platform/actions/runs/42570360880)は4系統でDSN設定・非SKIP・有限実試験PASSだが、別DB・別OperationでありHTTP Host→同一Operation Commit/Receipt/Client反応のE2Eではない。[Design Context継続Token #3723](https://github.com/hat47x/tei-platform/pull/3723)の`STALE_SNAPSHOT`も現在Grant/実Provider資格とは別。**HDG-X01〜06（NOT RUN）**を独立Expectedとして#1016/#1043/#1158/#1174/#3262へ引渡し、実Consumerを最優先にする。**11/09直近成果（[HTTP設計§24](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）**：[main #3761](https://github.com/hat47x/tei-platform/pull/3761)の実`hook.Runner`が`ValidateOutcome`で未知名を`HOOK_OUTCOME_UNDECLARED`として拒否し、語彙内／空／`not-satisfied`とCommand拒否を分離（[全体CI #42606902787](https://github.com/hat47x/tei-platform/actions/runs/42606902787)成功）。外部Plugin Transport→HTTP/IPC/Clientへの受信・表示は未実証。[#3757](https://github.com/hat47x/tei-platform/pull/3757)はEnvironmentの6種Bindingを調査し、`ObservationProviders/EvidenceProviders`は通常CLI`-plan`で**解決済み**だが、製品`observation.Router/evidence.Router`の実組立てと`Observe/Collect`の実Invokeは未実証（[判断資料§12](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)）。[#3760](https://github.com/hat47x/tei-platform/pull/3760)のself-build in-process Catalog+digestでPlan Fingerprint一致、`bound`は限定確認済みだが**`qualified=false`／L2 claimなし**。Plan導出を実Runtime Artifact/Current Grantに昇格しない。前段[#3736](https://github.com/hat47x/tei-platform/pull/3736)の全体Health #42576080511はsuccess、ただしHDG-X01〜X06はNOT RUN。次の[HDG-X02a/X03a/X05a/X05b](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)は独立Expected・未実行として#1251/#1016/#3421/#1174の各Ownerへ渡し、Product S2 Provider Bの実呼出しと独立Observation Consumerを優先する。**P0以降**：実CLI Design Context（#3262/#3354）の必須集合、標準Graphの多重Edge/出典・実Consumer（#1124/#2497）、現Command Unknown（#1174）、Trait/Provider/BOMの実資格（#3279/#3267）を接続する。E1 plan→Go HostのBinding、実TEI Graph Loader、Actual Provider/Client/Effectは未証明。既存Gate B/Cと各Ownerを維持する。

## TEI標準要件：モデル全容の図表化と、標準／カスタムの理解可能性（2026-11-08）

**基本要求（[RQ-036](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-036-ドメインの全容を図表とナビゲーションで理解拡張できること)、利用者明示）**：TEIのすべての標準機能は、業務／システムの対象構造、イベント群、状態遷移、Function、Pattern、Data、Rule、Operation、Providerと外部連携が、**TEI自ら導出する一貫した図表体系**の中に位置づけられることを前提とする。各Patternだけを個別に説明しても標準機能として十分ではない。利用者が事前知識なしに「全体として何をするか・何が起きるか・どこを変えられるか」を理解・探索できる標準ガイド／ナビゲーションを備える。Web/Desktop/API/CLI/MCP/VUIなど媒体ごとに別の意味正本を作らず、異質なConsumerにも同じ型付き参照と適用資格を渡す。

**図表を全体のガイドとして保つ（2026-11-09、利用者明示）**：大きなFlow・状態遷移図・構成図は、個別要素の確認・編集中も全体の道案内として働く。詳細を開いても全体像の中の現在地が見え、概観の同じ位置へ戻れ、一枚で読めない図は意味上の区画で概観と内訳に分ける。生成AIには同じ概観・現在地・近傍をCLI/MCPから構造化テキストで渡す。図表は実装である以前に設計書であり、要素から実装・試験の所在と未実装の範囲へ辿れる実装のガイドとする。具体化と受入（MAP-17〜23）は[三要素Method §19.8](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)に置く。

**標準とカスタムを見分けられること**も機能要件とする。標準固定の意味・Guard・Authority/Effect境界、設定可能な既定値、明示Extension Slot/Hook、案件で置換可能な部分、観測のみの部分を、Flow・Graph・Matrix・CLI説明で同じように区別する。差し込みのある個所は**標準フロー／差分／合成後の実効フロー**を相互参照可能にし、Plugin/Generator/Targetの版と受理資格を隠さない。ドメイン別のEvent/State定義は必要に応じてカスタムできるが、型・Source・現在権限・実副作用の検査をPresentation変更として迂回しない。

**完全性の意味**：指定したWorld／Source Scope／Consumer公開資格に対し、要求される図表・関係・Pattern／Event／Stateの対象集合と、実際に解決／表示した集合、未解決・非公開・未対応の集合を区別して表示すること。必要集合自体が未確認なら「全体像はpartial」と説明し、表示したものだけで完全性を主張しない。図表・表・CLIは共通のAccepted Meaning/Sourceから導出し、手書きのガイド図やAI説明を第二正本にしない。

**実装／受入の標準接続**は[三要素Method設計§19](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)に集約する。具体的な初回受入案はMAP-01〜10（全容Catalog、Event→Lifecycle、Pattern位置づけ、固定/Custom/実効フロー、初心者の段階ナビゲーション、Scope・Coverage、複数Consumerの一致）を基本とし、**MAP-11〜16**でLifecycleを可変の型付きGraphとして組み立てる際の保証、Plugin競合、到達不能、編集中と実行中のRevision差、AIチャットのInteractionと業務Effectの分離を反証する。固定するのは**不変の保証**であり、全Domainに一律の手順列を強制しない。#1124をDiagram/Navigation、#1125をEvent/Business Flow、#1126をData、#1127/#2507をFunction/Pattern、#2497を投影意味の保持、#2619/#2620をEverything is Data/役割、#2514/#1180を独立Expectedの既存Ownerとする。最初は**read-onlyのModel Map→明示Gap→複数図表へのジャンプ**を検証し、その後にHook差分・Typed Edit・Target実効動作を扱う。現時点では標準要求を計画へ反映した段階で、全図表Generator／全媒体Navigation／全Patternの実装完了は**未確認**。既存Gate B/Cを遡及して変更しない。

<!-- tei-b2c-headless-plan-20261108 -->
## 高度B2Cの自由度とConversion Engineeringの独立検証（2026-11-08、提案段階）

高度B2Cにおいては、競争力を生む独自Frontend・SEO・体験実験・頻繁な販促更新のため、TEI標準Renderer/DOM/Framework/Runtimeを使わない完全Headlessも正規利用として認める。標準UIを使う省力化と、UIを全面的に独自開発して型付きData/Operation/Effect/Source/Scenarioだけを活用する方式を、同じContract上で比較する。B2CのためにTEIの第二Host・Resolver・Commerce/Analytics Engineを追加しない。

**検証を優先する価値**は、単なる汎用EC画面生成ではなく「施策変更→配信Variant→実表示→Intent→権威あるOperation Effect→計測・評価→採用/撤回」を既存のMeaning/Provenance/Plugin/Conformanceで安全に結べるかである。Experience IntentとBusiness Meaning、画面Rollbackと注文Effectの補償、通常の販促変更と価格/決済/同意を伴う変更を別責務とする。全体図はRQ-036の共通Model Mapへ位置づけ、固定保証と拡張Hookの実効Graphを区別する。生成AI/Harnessは提案/テスト/測定の薄いConsumerとし、現在権限や業務意味の受理を代行しない。

**順序**：B0 read-only Fit/Gap → **B1a** 独自Frontend＋既存の型付きRead/Operationの一往復（標準Rendererなし） → **B1b** Commerce SandboxのCart/Checkout・実Effect/独立Read（B1aと別の資格） → B2 部分標準UI/Experiment/旧Client/Consent/配信Cache・Browser回帰 → B3 AI性能・変更時間・計測品質の同条件比較 → B4 ECと予約/Subscriptionの異質Consumer比較を経て外部Adapterを製品化判断。#2586の実Function→Query→Provider→Data Contextと#4350の表現・写像Fit/Gap、既存#1119 E2や#951/#1251の実Provider検証を優先し、新規P0 Gateにしない。

**資格・正本**：[#3509](https://github.com/hat47x/tei-platform/issues/3509)が現行B2C受入Owner。旧#3512の設計本文は[保存コメント](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6772845918)へ全文移管され、#3529も横断計画への引渡し後に未マージでクローズされた。11月12日の敵対的検証・BC-25〜34はIssue本文で管理し、既存Scenario/Independent Expectedを再利用する。**実Native Browser、Commerce Provider、実Effect/別Read、Field Experiment、CVR改善、Owner受理はNOT RUN/NOT ACCEPTED**。


**2026-11-12の敵対的レビューで強化した契約**： (1) Headlessを「専用SDK/常駐JS必須」にすり替えず、Mandatory TEI Runtime/DOM/Router依存0のNative経路を実Artifactで確認する（TEI利用の総コスト0を意味しない）、(2) 変更レーンは文言/CSS等のファイル分類ではなく、実効のRule/Consent/Price/Effect/Disclosure依存で判定する、(3) 生成AIのWeb/Studio操作対象はFocus/DOM座標でなくSource Subject・Scope・Selection epoch・Revisionに固定し、古い選択からの書込みを拒否する、(4) Experiment assignment、実Exposure、Operation Effect、業務成果・因果推論を独立観測し、同意・欠測を「非購入」に変換しない、(5) Screenshot/実験ログ等のEvidenceは[#4307](https://github.com/hat47x/tei-platform/issues/4307)のlocal/NAS/S3とGitリンクの分離を使い、GitHubへの実体・存在情報の送出を必須にしない。[#4198](https://github.com/hat47x/tei-platform/issues/4198)のLifecycle Graph、[#3278](https://github.com/hat47x/tei-platform/issues/3278)のTrait/Hook、[#997](https://github.com/hat47x/tei-platform/issues/997)のUI Kitは各Ownerのまま。詳細と独立Expected BC-25〜34は[#3509](https://github.com/hat47x/tei-platform/issues/3509)に集約する。B2C独自の第二Workflow/Graph Compiler/Analytics Engineを追加しない。

## 基本開発方針（2026-11-01、Owner方針）

**外部要件による十分な机上評価 → 基盤構造の実装 → CUIと標準プラグインの並行開発・一気通貫検証 → 人間向けGUIの構築**を、新規機能開発の基本順序とします。これは過去のGate A/B/Cの判定を変更するものでも、独立した新しいGateを設けるものでもありません。実装済み・検証済みの範囲は[現行の実装状況](../docs/status.md)、個別の進行・受入は各Issueと既存Gateで確認します。

机上評価の観点、複合要件の反例、CUI・標準Pluginの一気通貫検証の具体案は[外部要件による基盤評価とAI-firstヘッドレス検証](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)に整理します（設計候補・未実行）。

基本データ構造の机上評価は、通常のプログラミング言語の型階層より上位にある**構造形式・数学的構造・数量／単位・トポロジー**の概念空間から検討します。具体的な構造には複数の親や適合契約を認めつつ、分類する問い・関係の意味・構成と実現方式を区別します。候補と未決点は[基本データ概念空間の横断設計](design/FOUNDATIONAL_DATA_CONCEPT_SPACE_2026-11.md)で管理します。これはCoreの新型や多重継承実装を確定するものではありません。

特に、同じ値集合が**異なる演算の束縛によって同種の数学的構造を複数持つ**場合（例：自然数の加法・乗法モノイド）、Facet名や多重親の列挙だけで演算・公理・適合を決定しません。数学的Theoryの要件、具体的OperatorへのBinding、成立するLaw、解釈Scope、根拠の種類を区別し、[多重Facetの整合と公理の検証 #2544](https://github.com/hat47x/tei-platform/issues/2544)に具体的反例を管理します。広域の構造探索は[#2501 Corpus](https://github.com/hat47x/tei-platform/issues/2501)、構造保存・編集逆写像は[#2497](https://github.com/hat47x/tei-platform/issues/2497)、分類そのものは[#2495](https://github.com/hat47x/tei-platform/issues/2495)を利用します。現段階はいずれも未採用の検証計画です。

**構造のMECE性・複数Projectionの整合**については、固定のData Type列挙より、名目ID／構成子／Facet／演算束縛／制約／解釈文脈／出典を区別し、各判断の適用範囲と具体的反例を残します。比較のための正規形が計算できても、それを業務意味の同一性・Owner受理へ昇格させません。異なる構造が同じ意味を投影する場合（例：[TEIの多値Record投影 #2576](https://github.com/hat47x/tei-platform/pull/2576)と[SUIの所属表現](https://github.com/hat47x/tai-pleroma/pull/253)）は、**どのRelation/Sourceを正本とするか、列挙順・重複・完全性・Version・更新Intentをどう保存するか**を別に判断します。既存[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#2544](https://github.com/hat47x/tei-platform/issues/2544)/[#2497](https://github.com/hat47x/tei-platform/issues/2497)と[Draft PR #2496](https://github.com/hat47x/tei-platform/pull/2496)が比較・反例の担当です。実Targetで意味が保存されたことや両表現の意味上の等価性は未実証です。

**Projectionの値忠実度と対象範囲**も、基本データ構造の机上評価からCUI/標準Plugin・実Target検証へ引き渡す項目です。Fieldの値型、必須/任意と未設定・明示NULL・空、単一/多値、重複・順序・完全集合性、数値精度・単位、Source/Consumer、編集時の変更Intentを区別し、特定の用途で許容する情報損失を明示します。既に実装された[Record Projectionの多値・数値・optional・boolean（#2576/#2578/#2580/#2581）](https://github.com/hat47x/tei-platform/pull/2578)を[#2585](https://github.com/hat47x/tei-platform/issues/2585)の局所Conformanceで検証し、[#2497](https://github.com/hat47x/tei-platform/issues/2497)の構造保存条件と[#2498](https://github.com/hat47x/tei-platform/issues/2498)の実Consumer適合へ接続します。SUIの読み取り専用artifactと既存SUI元データのような**部分Projection／複数Source合成**では、フィールドごとの正本・未投影項目・時点・版を説明できるかを確認します。厳密な金額計算の型契約は#1705、業務DBのmissing/nullは#2056の成果を再利用し、GUIやProviderの正式な意味適合は別途実証します。現在の追加項目は受入設計であり、実装・実証完了を宣言するものではありません。

**時間変化するProjectionの意味保存**は、静的なValue/Presence適合の次の検証対象です。同じQuery/Functionから生成するCUI・Studio・SHOの複数Consumerについて、Source意味版、Query/Result SliceのScope/Completeness、Snapshot/read-set、Event/Processing/Observation Time、Selection epoch、実Consumer読込版を分け、Claimの支持・暫定・撤回・置換・失効を説明できるかを確認します。先頭ページの0件を全体不存在へ昇格させないこと、JOINの重複・LEFT JOIN仮NULL・TOP-Nの変更で旧結論を適切に撤回すること、古い非同期応答を新しいContextへ混ぜないことを反例とします。[#2497の§12](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)と[#2501のPC-97〜108](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)に整理し、単一FunctionのHeadless実証は[#2586](https://github.com/hat47x/tei-platform/issues/2586)、Data Contextは#2506、Freshnessは#1708、版間対応は#2505、異質Consumerは#2498/#2516の既存Ownerに帰属させます。追加12件は反証候補であり、実試験PASSや業務意味受理ではありません。

**Measureの再集計適合とClaimの開示適合**を、値忠実度・時間変化するProjectionに続く検証軸として追加します。同じDecimal型・同じ表示値でも、平均・比率・重複排除人数・時点残高・分位数は集計軸と必要な中間状態が異なります。[#2593](https://github.com/hat47x/tei-platform/issues/2593)のMeasure再集計を[#2497の§13](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)に接続し、[#2501のPC-109〜120](https://github.com/hat47x/tei-platform/issues/2501)を独立反例として整理します。また、内部で正しく導出したClaim・根拠へのアクセス資格・外部Consumerへの開示資格・開示後の根拠充足・実際のConsumer読込を別々に検証します。Construction Context/AI CUI/SHOは[#2516](https://github.com/hat47x/tei-platform/issues/2516)、外部開示は#1178、SHO原本は#1803、実Consumerは#2498の既存Ownerに帰属させ、第二のMeasure/Authority/証拠Registryを作らず、受入・実送信・実Targetの観測を独立管理します。今回の反例は机上計画であり、実装/Owner受理は未実施です。

**Consumer間Claim比較・文書Occurrence・開示非干渉性**を、構造保存とConstruction Contextの追加検証軸として位置づけます。同じ表示値でも対象・演算・Grain・Scope・Snapshot・Source版が異なれば比較不能であり、異値でも部分取得と全件確定のように両方正当な場合があります。比較可能な同一Claimの不一致だけを独立検証基準へ返します。また、同じTEI意味要素がSHO文書に複数出現するとき、Semantic IDと文書内Occurrence ID、局所編集・再生成・レビュー現在性を区別します。内部で読める根拠が外部Consumerへ開示できるとは限らず、非公開情報の有無が比較結果・拒否理由・Traceから漏れないかを既存開示Ownerで検証します。[#2497の§14](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)と[#2501のPC-133〜144](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)へ整理し、実Consumer横断は#2516 W8、SHO文書はSHO #4/#6、数値忠実度は#2585、情報開示は#1178へ帰属させます。追加反例は設計段階で、実SHO/OS/AI/GUI試験やOwner受理は未実施です。

**Functionの依存閉包・物理Read・開示・Effect対象束縛**は別々の受入義務として扱います。表示用Data Mappingだけでは非表示Validation/Rule入力を閉じられず、同じ公開JSONでも実ProviderのOverfetch有無は判定できません。Rowset completeとField observed、宣言したallと実際の取得完了、表示対象と実Request/Effect対象、公開回答と秘密Action候補の開示可否を区別します。[#2497の§17](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)へDC-01〜08を整理し、既存Corpus PC-157〜168と重複起票しません。Rule/Mappingは#2624、Provider実Readは#2628/#2622、Contextは#2506、View→実Requestは#2598、外部開示は#2516、Authoringは#2621のOwnerを維持します。隔離合成試験のPASSを実Provider/Host/OS/AIのE2Eや業務意味受理へ昇格させません。

**SQL→Data Operatorの意味保存と複数Function編集の合成不変条件**を、別々の実証義務として追加します。SQLのJOIN/EXISTS、NOT IN/NOT EXISTS、LEFT JOIN、NULL三値論理、Bag多重度、DISTINCT、Filter/Sort/LIMIT順序、平均の再構成は、構文や表示値が一致しても一般に意味同等とは限りません。SQL正本とTarget条件を保持し、候補Data Operatorを独立検証基準・実Providerで検証します。また、複数Functionの個別保存成功は並行実行後の業務不変条件を保証しません。Editing Session、Read Predicate、実Transaction境界、Commit不明、Effect Lookup、独立read-backを区別します。[#2497 §18](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)と[#2501 PC-169〜180](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)へ反例を配置し、SQLは#2387/#2622、独立期待は#2514、編集は#2597、実Effectは#1174の既存Ownerへ戻します。隔離SQLite試験のPASSは実TEI/PG/Host/Owner受理を示しません。第二Query Runtime・共通Transaction Coordinator・TEI独自Hostは導入しません。

**空結果の否定依存・選択的再資格化・Product Skillの実利用資格**を分離して検証します。0件のQueryにもWHERE/NOT EXISTS/Relation/Scopeの依存があり、0→1→0の同値観測から途中の変更なしを推定できません。変更影響と現在のConsumer利用資格を別扱いとし、依存証拠のあるClaimのみ選択的再評価を候補とします。さらにSemantic Role、Operation、CLI、Product Skill読込、実Tool呼出し、外部Consumerへの開示、Host Effectを段階別に観測し、Skill説明やCLI存在を実行許可・完了へ昇格させません。[#2497 §19](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)、[#2501 PC-181〜192](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)へ反例を追加。Change/Queryは#2582/#2387、Consumer資格は#2516、再利用は#2018、Product Skillは#2605/#2619とTAI #271、実Effectは#1174のOwnerを維持します。実Provider/Harness/SHO/Host統合試験は未実施で、新しいNegative Cache/Reactive Engine/独自Hostは設けません。

**Artifact Envelopeの同一性・物理公開一式・実Consumer読取は三つの別資格**として受入を計画します。既存`Artifact.Digest`はContent raw hashではなくKind/InputRevision/Rule/Generator/Content/UnresolvedのEnvelope fingerprintであり、現行`tei artifacts -out`の個別保存byteとManifestだけではEnvelope検証入力を完全復元できません。Content同一でRule/Generator/Unresolvedが変化する負例、Manifest取得中の世代切替、ExpectedMemberの欠落、SHO/Skillの旧版実消費を[#2497 §20](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のAE-01〜08へ整理しました。Artifact整合は#2482/#2630、公開一式は#2604、SHO/Consumer読取は#2516、Skill消費は#2605へ帰属します。#2633 main統合済みCharacterizationと#2659/#2607 Draftを区別し、別HEADのCI成功を現HEADの証拠にしません。実外部Consumer・障害注入・Owner受理は未実施で、第二Artifact正本や独自Hostは追加しません。

**Data Model正本とQuery二経路、生成アクセス性能の受入**：Table/Field/Key/Relationは直接宣言するData Modelを正本とし、SQL解析由来は候補に留めます。QueryはTEI QueryとNative SQLを第一級の別Authoring経路とし、AI変換は元SQLを保持した意味比較候補です。通常の取得はFunction Data ContextにTarget/Relation path/Field/Predicate/Scopeを明示し、複雑集計・方言固有処理ではNative SQLを維持します。生成アクセスのN+1、JOIN fanout、Read Scope、Query count、p50/p95を実Providerで測定し、静的に証明できる契約の重複ランタイム検証は前提・失効条件付きで省略候補とします。ただし実行時Grant、外部Input、Provider部分取得、Transaction/Effectの境界検証は省略しません。[#2497 §21](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のDM-01〜10は設計票でNOT RUN。既存#1126/#1129/#2387/#2622/#2624/#2628/#2018/#1174へ帰属し、第二Data Model正本・新SQL Runtime・TEI独自Hostを設けません。

### 外部要件を先に受け止める（要求・構造の机上評価）

CUIやGUIの実装に入る前に、**データ構造、意味上の識別・型・値域・関係・制約、操作・状態・副作用、プラグインの契約・合成・ライフサイクル、Target/Hostとの境界**など、後戻りの費用が大きい基本構造を十分に検討します。

成熟した参照設計が示す豊かなデータ・機能・業務モデルと、一般的な業務製品・サービス・オープンソース実装等の多様な外部要件を、単なる機能名称の一覧ではなく、具体的な利用シナリオ・制約・組合せ・失敗条件として収集します。一製品の実装方式をそのままTEIの中核に固定せず、外部要件をTEIの共通概念、拡張可能な意味モデル、標準／固有プラグイン、外部実装のどこで受け止めるかを比較します。

対象となる外部要件集合について、**全件の対応先と残余を追跡可能にする**机上適合表を用意します。最低限、要件と出典、期待する振る舞い、TEIでの表現・実現経路、複合条件、失われる意味、未解決点、反例・検証方法、判断状態を対応付けます。「全要件を受け止められる」という判断は、選定・確認した集合と証拠範囲に限ります。資料の欠落や未知の製品仕様まで完全性を推測しません。

基本構造に関わる重要な未解決事項は、契約の固定や汎用化の判断より先に検討します。拡張で表現できること、実行できること、互換性を実証できたことは別々に記録します。既存の[参照モデル移植時の不足分類](design/REFERENCE_PORT_IMPLEMENTABILITY_GAP_TRIAGE_2026-10.md)、[技術検証用サンプル](design/TECH_VALIDATION_SAMPLES_2026-10.md)、[設計原則](../product/principles/TEI_PRODUCT_PRINCIPLES.md)を再利用し、要件や判定の第二正本を作りません。外部資料の権利・公開範囲も既存の規則を維持します。

### 基盤を先行実装する

机上評価で得た構造上の判断を踏まえ、Canonicalな意味、Source/ID/Trace、型・関係・制約、プラグインの発見・契約・合成・Binding、検証・生成・実行との接続など、**各種製品機能が共有する最小限の基盤**を優先します。個別製品の機能に合わせた中核要素の追加や、X-TEA固有の技術構造への固定は避けます。

机上評価は設計上の適合可能性の確認です。実装の成立、複合要件、性能、実Provider・実Hostでの振る舞いは、後続の実験で独立に確かめます。基盤全体の完成を待つのでなく、標準プラグインとCUIに接続できる最小の縦切りから実装を進めます。

### 生成AI向けCUIと標準プラグインを並行開発する

生成AIが通常のコーディングと同等以下の探索・判断負担で実装できることを、操作系の強い要件とします。**CLIを基本の開発入口**とし、通常のファイル／DSL／SQL直接参照・編集、Git、必要に応じたDB直接参照・独立検証を第一級の経路として維持します。生成AIが機能・定義位置・実装方法・必要な検証を自力で発見できる、版付きの機械可読な診断・操作契約を整備します。開発者向けMCPは必須とせず、利用者向けMCPやComputer Useは目的に応じたAdapterとして評価します。

基盤の最小接続部が成立した段階から、**CUIと標準プラグインを並行して開発**します。プラグインの契約だけを先に完成扱いにせず、CLIから定義・選択・合成・実行・観測まで接続し、標準機能の実用性と基盤の抽象化の妥当性を相互に検証します。標準プラグインが中核の特権的な別経路にならないことも確かめます。

### ヘッドレス一気通貫を基準にする

各縦切りでは、**要件／Source → モデルの作成・変更 → Validation → Plugin/Binding解決 → 生成または意味シミュレーション → 実行 → 独立した結果観測**を、CUIで追える形にします。正例だけでなく、不適合、権限拒否、未対応、競合、部分成功、結果不明などの負例を対象に含めます。

同じrevision・fixture・Contract・生成物・実行条件を追跡し、机上の表現可能性、合成試験のPASS、実DB効果、実Hostでの成立、Ownerの意味受理を混同しません。既存のScenario、Driver、Source Index、Evidence、Gateを再利用します。生成AIが試作・修正・検証を自律反復できるようにしながら、正式な意味受理、権限付与、本番配備などの判断は[AGENTS.md](../AGENTS.md)の責任境界に従います。

### 人間向けGUIはヘッドレスの実証後に構築する

Web／Desktopの編集画面や利用者向けGUIは、対応する機能的な振る舞いがCUIで再現・検証できる基盤を基準として追加します。GUIは同じ型付きOperation・意味・検証経路を利用し、表示・入力・対話・アクセシビリティ等の**GUI固有の差分**を独立して確認します。GUIで初めて現れる要件は、必要に応じて基盤へ還元して検証します。過去に実施したGUIの参照試験や並行する調査は保持しますが、新たな基盤機能の完了条件としてGUI実装を先行させません。
- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md` ：生成AIによる性能改善のための設計ヒント、低負荷な統計収集、Source/Function/Query/Providerの観測接続、AI提案の独立Benchmark・適用・Rollbackを扱う**横断設計・実証候補**（[#3271](https://github.com/hat47x/tei-platform/issues/3271)、[#3262](https://github.com/hat47x/tei-platform/issues/3262)、[#2582](https://github.com/hat47x/tei-platform/issues/2582)）。新Optimizer/Core/Hard Gateや性能改善の実証完了を意味しない。
- `plan/design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md` ：原子的Traitの実装義務、Manual/Generated/Providerでの履行、合成・不適合診断を既存#951/#2495/#3271/#3267へ接合する**設計・実証候補**（[#3278](https://github.com/hat47x/tei-platform/issues/3278)）。新Resolver/新Hard Gateや実装完了を意味しない。
- [Issue #3503](https://github.com/hat47x/tei-platform/issues/3503#issuecomment-6772886371) ：Lifecycle Graphの型付きRole・不変Obligation・Plugin Patch競合/合成・Generator入力・L0〜L6縦断検証・LG-01〜16独立Expectedを、既存#3503/#1158/#951/#1121/#1180へ引渡すDraft。#3509のB2C Headless、AI Web Chat/VUIも異質Targetとして比較。既存P0を置換しない。
- [Issue #3503](https://github.com/hat47x/tei-platform/issues/3503#issuecomment-6772855707) ：API/Web/Desktop/CLI/MCP/VUI横断の共通Operation、型付きLifecycle Graph/Hook Binding、状態付きInteraction/AI Web Chat、実行Event/Effect意味論を、既存#1158/#2090/#951/#1121/#1180へ接続する**設計・実証候補**（#3503、Draft PR #3507）。標準/拡張/実効/観測の区別と、第二Host/Registry/新Gate禁止を維持する。
- [Issue #4198](https://github.com/hat47x/tei-platform/issues/4198) ：Statecharts/SCXML、Process Variants、Graph Transformation、Temporal、AG-UI/A2UI等から**Lifecycle Graph自身をPluginで拡張してGeneratorへ渡す**設計を検証する研究。図表編集からSource/Buildへの帰還、AI Web Chat/VUIの異質性を扱うが、正式Graph Compiler/実Target/Owner採用は未完了。
- `plan/design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md` ：Web/Desktop APIの例外・Client Reaction・Graph/Table/Traceの**横断設計と段階的な実証計画**（[#3274](https://github.com/hat47x/tei-platform/issues/3274)）。**P-1a/b**：E1の構成時Guard DAGとRuntime HTTP Decision Pathを区別し、[実Node専用CI #42396614454](https://github.com/hat47x/tei-platform/actions/runs/42396614454)の限定HDG-21 CharacterizationはPASS。全体Healthは同PRでgofmt不一致（別Source）によりFAIL、Go vet/test/Workspace validateはPASS。**P-1c**：既存Go Reference Hostのdefer Auditと10経路Callback/Status/Stageを、[Node+Go専用CI #42461546535](https://github.com/hat47x/tei-platform/actions/runs/42461546535)で限定PASS。**P0-R（限定実証）**：E1・Goの独立宣言は両側が局所的に構成可能でも不一致になることを[Node/Go実CI #42462531490](https://github.com/hat47x/tei-platform/actions/runs/42462531490)で確認（Node 7/7＋Go package全PASS）。**P0-S（限定CI合格）**：実`reference/workspace`の`application-pattern:order_entry.submit_order`と`semantic-capability:sales.order.submit`・Source Index/Canonical参照を既存`operations.InspectSubject`で検査し、[Node/Go専用CI #42464774706](https://github.com/hat47x/tei-platform/actions/runs/42464774706)＋同HEAD全体Health #42464774715でPASS。**P0-B（限定CI合格）**：既存`driver.ResolveCommandBinding`を実Workspaceで通し、欠落・重複・有効だが意味的に違うCapabilityの負例を[専用CI #42470312187](https://github.com/hat47x/tei-platform/actions/runs/42470312187)でPASS（同HEAD全体HealthもPASS）。**P0-I（実Host→Driver→合成AdapterのInvocation検証候補）**：実`NewHost`・`NewCanonicalActionAuthorizer`・実Driver`InvokeCapability`から合成`Capability Adapter.Invoke`までを一つのHTTP Attemptで通し、Session/CSRF/Canonical認可拒否・Adapter解決障害・実Invoke障害・Canonical Result契約違反の独立Expectedを[設計§18](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)で試験する。実Adapter呼出しが確認できても本番Plugin Provider/Artifact/Grant・永続DB Effect/Client完全受領・Graph Required全閉包ではない。**P0以降**：実CLI Design Context（#3262/#3354）の必須集合、標準Graphの多重Edge/出典・実Consumer（#1124/#2497）、現Command Unknown（#1174）、Trait/Provider/BOMの実資格（#3279/#3267）を接続する。E1 plan→Go HostのBinding、実TEI Graph Loader、Actual Provider/Client/Effectは未証明。既存Gate B/Cと各Ownerを維持する。

## 現在の主線

第1版の参照アプリケーション`order_entry`に基づく判定段階と既存の検証レーンは次のとおりです。**新規開発の基本順序は上記のAI-first・要件先行方針**に従います。

```text
Product Value / Requirements
        ↓
Application Pattern
        ↓
Gate A — Ready
        ↓
Application Driver
        ↓
Gate B — 3 Required Evidence / Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
Handover Baseline B at revision R — 固定済み
        ↓
First Web Target build T1 — Evidence取得済み（T = 0ba87a4e0a96…）
        ↓
Gate C — PAS-04 Evidence — Ready
        ↓
        ├──────── Authoring / Knowledge lane
        │           ├─ Studio minimal loop — PAS-01
        │           ├─ knowledge-first exploration — PAS-06
        │           └─ AI Proposal / Acceptance — PAS-01 / PAS-06
        │
        └──────── Second Target pressure-test lane — PAS-04 / RQ-024
```

判定段階Cの後は、定義の記述・知識の活用と、二つ目の実行先の検証を並行して進めます。互いの完成を待つ必要はありません。最初の実行先で確認した違いと検証根拠を共有します。

### Studio Desktopセルフビルドを最優先にする現行の実証順（2026-11-13）

#### 2026-11-14 第13次更新：SaaSの複数Worktree・DB実行対象を段階分離しP0 Self-buildを保護

**最新の限定基線**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`のWindows N2→N3実Bundle/EmbedFiles前後byte照合・改変コピー拒否・実WebView2はCI **12/12 success**。[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a`（Builder/Child事前pin）は**8/8 success**、[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) HEAD `2d403f49`（Resolver不正応答拒否）は**5/5 success**、ただし別HEAD/Draft・未merge。[Draft #4627](https://github.com/hat47x/tei-platform/pull/4627) HEAD `9491bf9e081bf98f39056d0a4142c2764c31289d`のWails Server `StudioBridge` exported 15 Method inventory/Service登録レビューtestは[Wails Host Run #43544229895](https://github.com/hat47x/tei-platform/actions/runs/43544229895)を含むCI **5/5 success**。**Method棚卸しの合格≠共有Web認証・実HTTP/WS公開遮断・DB/Workspace隔離**。現main WorkbenchはTypeScript/esbuild/CodeMirrorで、ReactもLitも依存していない。

**直列P0は変えない**：①#4623＋#4624を一HEADで宣言→Builder選択→実子起動前拒否まで統合、②N0 Inspector TS→N2選択Input（H0）のProducer/Consumer Role/digest/Attemptを結ぶ、③N1/N2 Node/esbuild/N3 Go/Wailsの**非Plugin Desktop**実Builder・Consumed Source・ArtifactSetを既存#1080/#1097 BuildExecutionへ段階接続、④生成Stage1 TEI自身の実Process→Stage2 Windows Desktop→当該Stage2実WebView2/BridgeをB2/B3として独立検収する。H1限定PASSをOwner受理/全Closure/完全セルフビルドへ拡大しない。SaaS・WASM・UI Framework変更は新Hard Gateにしない。

**並行P1：Desktop→複数対象・共有Webへの順序を追加**：まずローカルStudioでHost許可済み**2 Worktree×2 Database Bindingのread-only preview**を成立させ、Project/Repo/Worktreeの実Root・Git branch ref＋解決commit・Source revision・Environment/Profile→実Database instanceの束縛を**Viewer Selection・Host解決Context・Build/Operation Attempt**として別記録する。Git branch文字列/DB alias/Client入力だけでCurrent Grantとみなさず、別worktree同名SubjectやA→B→A、同alias別DB、Dirty Buffer/保存Source差を拒否。Provider/Environment#2586、Source/Root CAS#1529、Studio Host#1824の既存Ownerを使い、新Git/DB Manager/Global Transactionは導入しない。

次に[ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)の共通Go OperationでDesktop IPCとWails Serverまたは薄いWeb Adapterの**選択read-onlyメソッドだけ**を比較。現`cmd/tei-studio/main.go`のloopback Wails Serverは全`StudioBridge`（Command/SHO含む）登録で、`studiohost.ReadOnly`はper-browser主体を持たない。#4627のMethod Inventoryを維持しつつ、次は実Binder/HTTP/WSで禁止Method・別Principal/Workspace/Browser Contextの拒否とイベントAudienceの境界を検証する。Server Buildができても即公開・多利用者資格にはならない。さらに将来SaaSのBuilder実Process/ArtifactはServer Hostの許可済み作業領域へ固定し、2 Principal×2 Workspace×2 Worktree×2 DB Bindingの隔離、Credential秘匿・Quota/取消/Recovery・Source Revision競合を独立検収する。**Wails ServerのローカルPoCをWeb SaaSの全セキュリティ基盤とみなさない**。

**公開前の依存版レビュー**：現nested `cmd/tei-studio/go.mod`が固定するWails v3.0.0-beta.29は、公式[v3.0.0セキュリティ修正案内](https://github.com/wailsapp/wails/releases)より旧い。公開Webや多利用者Serverを受け入れる前に、公式修正・残る制約・Upgrade後のBinder/Origin/IPC/WS/実Windows回帰を確認する。既存P0を無審査で依存Upgradeすることも、旧版のPoC successを公開安全とすることも避ける。これはWeb版のP1安全性評価であり、現P0セルフビルドへ新Hard Gateを増やさない。

**UI/Go-WASMの技術比較はP2候補**：現在のplain TypeScript＋CodeMirrorを基線とし、Web Components/Litの代表PaneとReact対照を、同一Operation/FixtureによるPin裏返信/Editor/Graph/IME/Keyboard/ARIA、JS Bundle/再Build、生成AI修正の局所性で比べる。React採用必須でもLit全面移行必須でもない。純粋Graph/Filter/Source projectionのみGo native vs TypeScript vs browser Go/WASMをベンチし、Source/Meaning/Current Grant/Build資格はHostに残す。実反復改善がなければ現行TSを維持し、採用前にP0生成・WebView2を壊さない回帰を要求。

**受入・Owner**：詳細な三Context、D0〜D3段階とSCX-01〜07の負例は[Desktop契約 Draft #4498 §0.13](https://github.com/hat47x/tei-platform/blob/docs/desktop-product-contract-20261112/plan/design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)へ置き、#1180/#2514の既存Expectedに割り当てる。最優先#4583/#2831、Studio/Web#1824/#3421、Worktree/Source#1529、DB#2586、Pin#4515に責務を戻す。全Targetで同じSource/Meaning/Authority正本を二重実装せず、Server/Clientの局所Context世代・Operation実Attemptを明示。**Docs-only**：新SaaS/DB/Git実動、React/Lit/WASM実比較、公開Web認可、Stage1→Stage2実行、Owner受理、main統合は行っていない。

#### 2026-11-14 第12次更新：H1実Windows PASS後のBuildIntentとDesktop／Web／WASM共通化

**最新実証（前回P0-3の限定完了）**：[Desktop統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`／[Windows CI #43542664470](https://github.com/hat47x/tei-platform/actions/runs/43542664470)でgenerated/ordinary Windows Wails WebView2両Job success、HEADのPR Workflowは**12/12 completed/success**。N2 esbuild実Bundle出力を`n2-bundle-producer.json`にRole/format/digest/HEAD/Attempt付きで固定し、N3の`go list -tags studio_generated`が選んだ`EmbedFiles`の実byteと**Go build開始前・後**に照合。`n2-n3-bundle-handoff.json`に同一Roleの受領・変造コピー拒否（元を変更せず別コピーに文字列追記）・`readTimeClosureProved=false`を記録。N2選択24件、N3 Go選択312件、生成exe SHA `e2158bc8e20a5f2fe153ca96d08aad1571d2bacc9c87672b288fbfcbe5566205`、実WebView2配信JS SHA `755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`を同HEADに紐付け。**H1（N2→N3 pre/post SHA/Role/Attemptの局所観測）PASS**であり、read-time完全閉包／正式BuildIntentによるN1/N2/N3駆動／Stage1→Stage2 B2／当該Stage2 B3は**NOT RUN**。PRはDraft未マージ。

**P0の未完了優先順へ更新**： (1) [Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) `f1e219a`の`-bind-resolved-builder`（CI 8/8 success）と[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) `2d403f49`の宣言/Resolver整合（CI 5/5 success）を、#4587の最新mainベースと**同一HEAD/実Attempt**で統合・正負検査。 (2) N0実Generator output `inspection-reader.generated.ts`→N2実esbuild selected input（H0）をRole/入力bytes/Source RevisionでN2実行前後に接続し、旧TS/誤TS・未選択inputを拒否。 (3) N1、N2 Node/esbuild、N3 nested Go/Wailsの実Builder/argv/ArtifactSetを**既存#1080/#1097の非Plugin Build Execution/ArtifactSetへ一工程ずつ**接続し、Workflow Shell直起動を正式BuildIntentへ置換。H1の証拠は失わず、Plugin主体BuildPlanにDesktop exeを偽装しない。 (4) 生成Stage1 TEI自身の実ProcessがStage2 native Windows Desktopを構築し、**そのStage2自身**のWebView2/Bridge/生成Inspector受領をB2/B3として独立検証する。Wails単体成功、Build Digestの一致、CI proxy呼出しを再入成功としない。

**並行するDesktop/Web共通化（P0の新Hard Gateにしない）**：[採用ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)に従い、共通Go `internal/operations`＋StudioHost/Provider/Shared Acceptanceの型付きOperation/Receiptを**Desktop Wails Bridge・Web HTTP/stream Adapter・CLI/MCP**へ使う。既存`cmd/tei-studio/main.go`のWails Serverモードは`127.0.0.1`以外を拒否する**ローカルPoC**であり、これをWeb公開Host/Session/Origin/CSRF/CORS/HTTPSの検証済み実装と誤称しない。Workbench UI・read-only Build Preview・Pin/Focusの表示規則を共有し、WailsのDesktop OS WindowとブラウザーServerの実行・開示境界は分離。**既存`StudioBridge`には`ExecuteCommandWithPolicy`、`ExecuteCommandWithReferenceStore`、SHOのOpen/Resume等のexported methodが存在するため、Wails Serverの全Serviceをそのまま公開Webへ公開しない**。Web第一試作は明示したread-only代表OperationのMethod/Event公開リストを作り、ServerのActor/Workspace/Session/Grant・Origin/CSRF/CORS/HTTPS・表示先限定を独立検収する。全Browser向けbroadcast可能なEventへSourceやAI結果を流さない。**Go/WASMは非UIすべての移行先と決めず**、大量Graph/Filter/Projection等純粋関数のTS/Go Host/Go WASM対照（結果同値、cold/warm、memory、IPC bridge/copy cost）から局所採否を決める。WASM局所Pin/Context/CacheはViewer Snapshotであり、Canonical Source、Meaning受理、Current Grant、Build Builder選択、Artifact/Effectの正本・実行許可にしない。

**共通化の最初の検査**：AI OFFでDesktop/Webの同一Operationを実行し、型付きValue/Failure・Source Revision/Scopeを比較。Pin A裏応答がB Editor/Graph/Dirty Bufferに作用0、Web Session A/BのWorkspace/Principal非干渉、WASM旧SnapshotとHost R1の不一致失効、通信断後Effect unknownを独立Expectedへ引渡す。Web Host Security/Wails Server可用性は別環境実証、WASM最適化とB2/B3 Self-buildを直列Hard Gateにしない。詳細は[Desktop製品契約 Draft #4498 §0.12](https://github.com/hat47x/tei-platform/blob/docs/desktop-product-contract-20261112/plan/design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)。Ownerは#2831/#4583（Build）、#1824（Desktop）、#1158/#3421（共通Operation/Web）、#4515（Pin）、#1180/#2514（既存Expected）を維持。**本追補はDocs-only**。新Go/WASM/Web Browser/Stage2実行・Owner意味受理・main統合は行わない。

#### 2026-11-14 第11次更新：最新main追従DesktopとN2→N3 Artifact実受渡しを先行

**現状（exact HEADで独立検証）**：[Windows統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `efe1de03c23d7e8bd9271e465fc0b8436f9a993c`はmain merge `03679ccd`後の`mergeable=true`。実CI pathに`cmd/tei/**`と`internal/buildintent/**`を含め、CLI/Resolverが変化したらDesktop Windows CIが実行される。**再照合した新HEADの12 CIは11 success・Health 1 cancelled**。[Windows WebView2 Run #43541884048](https://github.com/hat47x/tei-platform/actions/runs/43541884048)のgenerated/ordinary両Jobがsuccess、2 Artifactあり。[Self-build Run #43541884152](https://github.com/hat47x/tei-platform/actions/runs/43541884152)もsuccess。[Health Run #43541884124](https://github.com/hat47x/tei-platform/actions/runs/43541884124)はcancelledで再検収対象。旧Windows CI #43450255223のPASSは新HEADへ流用せず、新HEAD自体の限定Windows成功として扱う。[共有CLI Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a`のBuildIntent→実子pre-start pinは**8/8 success**、[Resolver整合 Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) HEAD `2d403f49`は**5/5 success**。**三つは別HEAD・Draft・未マージ**で、単一Desktop E2E資格ではない。

**新P0順序（既存Issue/Scenario/Build機構の範囲）**：
1. **P0-1 本統合HEADのWindows CI受入**：#4587のgenerated/ordinary Desktop、実WebView2/Bridge、Health/Self-buildの完了を同一HEADで判定。以前のmain競合は解消済みなので旧競合解消タスクは再要求しない。ただし#4587内N0はWorkflow Caller pin、N1〜N3はWorkflow直接実行であり、正式`BuildIntent`による全工程Buildではない。
2. **P0-2 Builder/Resolver同一HEAD**：#4623`-bind-resolved-builder`と#4624の宣言/Resolver応答整合拒否を同一HEADへまとめ、Intent→Resolved Plan→実子image/argv→Artifactを一Attemptで正負検証。正しいBuilder Bを選んだのに実子A、またはResolverが宣言と異なるAを返すときは子Call 0/Artifact 0。両PR個別Greenの合算は禁止。Catalog Authority/Meaning/現在Grantは独立。
3. **P0-3 最初の非Plugin Artifact辺はN2→N3**：実`n2-esbuild-metafile.json`の唯一の`cmd/tei-studio/assets/workbench/main.js`出力Role/digestを、実`go list -tags studio_generated`の`EmbedFiles`選択＋N3 Go build前後のbundle実byteと結ぶ。N2出力A/N3 embed B、同byte長の別bundle、PowerShell Step間変数消失、実行後に古いJSへ差替えを独立Expectedで拒否する。次にN0 Inspector TS→N2選択Source、N3 exe→実Windows配信JSへ拡張。現在の`b1-stage-input-observation.json`等の**既存Evidenceを拡張**し、新独自Manifest/Build Engineを作らない。`tools/reference-build-execution-binding/main.go`の`BuildExecutionPlan/ArtifactMember/ArtifactSet`は参照`package main`であるため製品APIへの直接importは不採用。非Plugin Desktop Targetは既存#1080/#1097担当との薄いAdapterとして設計。
4. **P0-4 Stage再入と実Consumer**：生成Stage1 TEIの実ProcessがStage2 Windows Desktopを構築した履歴（選択Intent/Builder/実invoker/Output exe）と、**そのStage2実exe**のWindow・WebView2配信JS/Bridgeの消費を別検収。Stage0/CI代理Go buildや旧Windowは不合格。OS Loader TOCTOU、Node/Go外部依存read-time全閉包、Runtime Provider Grantはunknownを維持。

**並行Studio/Web実装**：AI OFFのPin A裏再確認でFocus B/Editor/Graph/Dirty Buffer維持、保存済みCanonical Sourceの明示Build、実Build/Artifact/Consumer Receiptのread-only Preview、旧exeの暗黙置換禁止。Wails Desktopと将来のWeb Serverで同一Operation/Evidenceを使い、UI側のFocus/Pin/Viewer世代とHost Source/Build/Authorityを別寿命とする。X-TEA三要素・SHO/SUI・Plugin Profile/DB・AI Harnessは並行するが、P0セルフビルドへ新Hard Gateを増やさない。具体的File/Role/負例、H0 N0→N2・H1 N2→N3・H2 N3→実Consumerは[Desktop製品契約 Draft #4498 §0.11](https://github.com/hat47x/tei-platform/blob/docs/desktop-product-contract-20261112/plan/design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)へ一本化。**今回の改訂はDocs-only。新Go/Node/Windows実行・Stage1→Stage2再構築・Owner意味受理/本番統合は実施なし。**

#### 2026-11-14 最新差分：Resolver応答の宣言適合とResolved Builderの実行前束縛

- **実装更新A：選択Plan→実子**。[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) **exact HEAD `f1e219a25976946bcc8a19ef57f5490333ec8a84`**は、`tei self-build -bind-resolved-builder`を明示したとき、同じTEIプロセスがCaller指定のBuild CatalogとSourceから解決した`ResolvedBuildReceipt.Plan.Builder.ArtifactDigest`を実Child画像の起動前SHA照合へ渡す。Selected B/Actual Aは`SELF_BUILD_BUILDER_SELECTION_MISMATCH`で拒否し、PlanReceiptを保全、ChildRunとArtifactを作らない。**当該HEADの8系統GitHub Actionsはすべてcompleted/success**。ただしCaller指定Catalogの意味/Authority、BuilderのOS Loader実read瞬間、Node/esbuild/Go/Wails多工程とStage1→Stage2は別資格。PRはDraft/未統合。
- **実装更新B：宣言→Resolver**。[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) exact HEAD `2d403f4996c205514dc57d5fc1237127c6bd994b` は`internal/buildintent.Resolve`で、可換Resolverの返すSource Kind/ID・Builder Contract/Major/指定Provider/Implementation/Pin/Pinned・依存Kind/ID/Majorが元Intentと違う場合、Resolved Receiptを返す前に拒否。PR記載のGo/race/CLI限定テストは成功。**当該HEADのGitHub Actionsは確認時点で実行中のJobがあり、全体CI成功・#4623との統合成功は未主張**。Recipe/Target適合、実Source raw byte、現在Grant/Meaningは別検査。
- **現行主線への影響**：前回の「Resolverが選択したBuilder digestを実子へ渡すコードがない」という不足は**#4623のOpt-in一工程に限って解消**。新たな残差は①#4623と#4624の別Draftを最新版mainで統合し、宣言→Resolver→Pin→子→Artifactを**一つのHEAD/Attempt**で検収、②N0生成TS→N2実esbuild→N3 Go embedの**非Plugin Desktop Artifact edge**を既存BuildExecution/ArtifactSetへ接続、③生成Stage1実TEI→Stage2 native Windows Desktop→そのStage2自身のWebView2/IPC受領をB2/B3として別検収すること。旧[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)はmain競合と共有CLI重複があり、新pin付きWindows統合HEADのCIが未観測。#4623 Greenと旧#4587 Greenを合算しない。
- **既存Ownerへの引渡し**：#2831/#4583（Build Intent/実Builder/Artifact）、#1824（Windows Window/IPC）、#4515/#3788（Focus/Pin/Dirty Buffer）、#1251/#4570（Stage2 Runtime Plugin実Consumer）、#1180/#2514（独立Expected）。最初のUIは同じOperation/Receiptのread-only Build PreviewとPin A裏確認中のB閲覧保持。Builder byte一致をOwner意味受理や現在Grant/Disclosureへ昇格せず、Plugin主体のBuildPlanにDesktop exeを偽装しない。段階別E1〜E5、Provider/AI/SHOの資格分離は[Desktop製品契約 Draft #4498 §0.10](https://github.com/hat47x/tei-platform/blob/docs/desktop-product-contract-20261112/plan/design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)に集約。**今回の追加は文書更新のみ**、新Go/Windows E2E・Owner受理・main統合は行っていない。

#### 2026-11-14 更新：選択Builder実行前Pinの限定CI成功とB1-B実結線

[共有CLI Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) exact HEAD `929fcf857cd0f6a792a69fb23cadfb5eccc05e90`では`tei self-build` primitiveの`-expected-child-digest`が、子Builder実exeのSHAを**起動前**に照合し、Selected B/Actual A・不正pin・境界Launcher迂回を拒否する。実CIは同HEADの**8系統すべてcompleted/success**（Self-buildとRepository Healthを含む）。ただし**Caller指定のPinであり、BuildIntent Resolver由来の選択や全Desktop経路は未成立**。最新mainからの共通CLI差分のみを#4623に置き、Desktop統合[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の旧HEADと二重のCLI差分を合算・採用しない。#4587のpin統合Windows検収は新HEADで未確認、merge conflictの解消とCI再実行が必要。

**次の実装順**は (1) #4623の共通CLI pinをOwnerレビュー・回帰後に統合判断し、#4587の同じ2ファイルの重複を解消／最新mainへ追従、(2) 既存`internal/buildintent.Resolve`が導出する`Plan.Builder.ArtifactDigest`を**対応可能な一工程**の実Invoker Pinへ渡し、Plan FP・選択Builder ID/Contract・実子image/argv/Artifact roleを一Attemptに束縛、(3) N0実TS→N2 esbuild→N3 Go embedの成果物・後段実入力を最小非Plugin Target Adapterで結ぶ、(4) B2では**生成Stage1 TEI実プロセス**によるStage2 Windows Desktop Build、(5) B3では**そのStage2 exe自身**のWindows WebView2/IPC/生成Inspectorを検収する順。非Plugin DesktopをPlugin Subjectに偽装せず、既存`BuildExecution/ArtifactSet`・Build Intent/Builderを再利用する。前後SHA、Go/esbuild申告Inputsだけでread-time閉包やWindows OS強制を主張しない。対応不能工程は`unsupported`、未観測依存は`unknown`。

**既存Scenarioに渡す最小負例**：ResolverがBuilder Bを選択→actual A起動を事前拒否（子/Launcher/Artifact生成0）、Caller pinだけをL2へ昇格しない、#4623のGreen＋#4587旧Windows Greenの別HEAD合算を拒否、N2 output A→N3 embed Bを拒否、Stage2生成成功＋旧Windowを当該Stage2 Consumerと誤認しない。AI OFFのread-only Navigate/Pin（B閲覧中のPin A裏再確認でBのDirty Buffer不変）、Dirty/Saved/Meaning/実Builder/Runningの版分離は#4515/#1824/#3788の並行レーンで続行し、AI/SHO/X-TEA/実DB/Plugin全対応をP0の新Hard Gateにしない。判断とOwner別の反例は[Desktop契約 Draft #4498 §0.9](https://github.com/hat47x/tei-platform/blob/docs/desktop-product-contract-20261112/plan/design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)へ集約。**この追補はDocs-only・Draft未マージ**で新しい製品実装PASSやOwner受理を意味しない。

**2026-11-14の現行差分（Desktop第8次横断統合）**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) exact HEAD `24bfaed8fadcce5687fb7f02a121a21fd70f2861`／[Windows CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)で、通常Go Buildで用意した**実TEI CLIプロセス**がprimitive `tei self-build -child`を通じてN0 Inspector Generatorを起動し、親/子の実image・argv・exit・生成TS SHA、別Process比較、no-clobber負例を検証した。同一CIではN2 esbuildの24選択入力、N3 Goの303選択入力、生成native Desktop/実WebView2も限定成功。**Build Catalog/Intentによる選択とN1〜N3のTEI自身による実Builder起動、生成Stage1→Stage2は未実証**。B1全工程の`qualifiedBuildIntent=false`、閉包`unknown`、B2/B3 `NOT RUN`を保持する。従前の「N0はTEI外部からしか動かない」という記述はこの固定HEADについて更新するが、L2/L3へは昇格しない。

**次のP0を一辺に限定**：#4583/#2831がN0実Inspector TS→N2 esbuild→N3 Go embedで、宣言Build Intent/選択Builder、実Process image/argv、実消費入力・出力Artifact role/digestを**同一Attempt**へ結線する。既存#1080/#1097のBuild Execution/ArtifactSet契約を参照し、現`BuildIntent`（一Subject/一Recipe、Plugin主体）が表せない非Plugin Desktop工程に必要な**最小Target Adapter**のみ設計する。CI Shellを正式Build Planへ偽装せず、N2 metafile・N3 go list・事後SHAからread-time全依存閉包／OS強制を推論しない。次にB2では**生成Stage1実TEI**がStage2 exeを構築し、B3では**当該Stage2実exe**のWindow/WebView2/IPC/配信JS/生成Inspector利用を独立受領する。異HEAD・旧Window・Stage0代理実行を合算しない。

**Desktop実用の並行レーン**：#1824/#4515/#3788はAI OFFでNavigate→明示Pin→裏Inspect→再Navigateを同じSource/Qualified Subjectで実Workbenchへ通し、B閲覧中のPin A再確認でもB Editor/Graph/Dirty Bufferを変えない。Build Previewは既存read-only Operation/Evidenceの投影から始め、Dirty Buffer・保存済みCanonical Source・Accepted Meaning・Builder実入力・生成Artifact・現在Windowを別版として示す。未保存Bufferの自動Save/Build/Acceptance、旧Replyの自動資格復活、Stage2生成直後の旧Studio上書きを認めない。SHO/X-TEA三要素/DB Read/外部Harnessは既存Ownerの並行レーンとし、P0 Self-buildの新Hard Gateにしない。UIはFocus/Pin/Reviewを区別し、Declared/Observed/Enforced、PASS/FAIL/NOT RUN/UNKNOWNを既存Receiptから表示する。詳細な製品契約と既存DT/ED/PIN/BDへのExpected引渡しは[Desktop契約 Draft #4498 §0.8](https://github.com/hat47x/tei-platform/blob/docs/desktop-product-contract-20261112/plan/design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)に集約する。**今回の反映は文書のみ**で、新たなGo/Node/Windows/実Provider試験、Owner正式受理、main統合は行っていない。

**最上位の製品目標は、TEI Studio Desktopを実OS上で構築・利用し、TEI自身が次のStudioを再構築すること**。この目的に対し、最初から新しいDSL/汎用Interpreter/独立Build Hostや全機能のX-TEA・B2C実装を要求しない。既存Wails、Workspace/Source Index、標準BuildIntent/Builder、型付きOperation/Providerを使い、以下の証拠を段階的に接続する。段階の記述は新しい全体Hard Gateではない。

1. **実Desktopへの生成部品接続**：[Draft PR #4572](https://github.com/hat47x/tei-platform/pull/4572)で、TEI契約Source→既存の決定論Transform→Inspector reader（TypeScript）→esbuild bundle→Go `//go:embed`→**実Windows WebView2 Desktopでの利用**を局所実測。生成前bundleを使った負例とSource digestの対照も報告され、PR exact-headのHealth/Workbench Embed/Windows WebView Smoke等は成功。ただし**Draft・main未統合**であり、実Wails表示に成功したこととStudio全体のTEI自己生成は別資格。
2. **実Builder入力の閉包**：#2831の既存BuildIntent/Builderへ、前項のTransform/生成Sourceだけでなく、**実際に実行したNode/esbuild・Wails Build・nested Studio module・Go embed**を、必要入力・実呼出し・出力Artifactとして明示的に結ぶ。#4572ではbundling/Wails BuildがまだTEI Build Graphの外にあり、`l2b`合格やBuilder完全性を宣言しない。別のBuild Engineを作らず、既存Adapterが実行できる必要工程を一つずつ通す。
3. **自己再入場とOS境界**：Stage1として実生成したTEIの**プロセスイメージ**がStage2のBuildPlan解決・Builder要求・実Artifact生成を駆動したことを実行証拠で確認する（#2831）。[main PR #4575](https://github.com/hat47x/tei-platform/pull/4575)のentry事後状態観測は、空ディレクトリを作るだけの偽独立を防がない。[PR #4578](https://github.com/hat47x/tei-platform/pull/4578)の境界launcherも現時点で入口Flagが未結線。Child申告、二回のByte一致、Stage0製Go Build、局所Reentryだけで**全体セルフビルド／OS隔離**としない。
4. **異質Consumerで製品の基盤を試す**：[main #4539](https://github.com/hat47x/tei-platform/pull/4539)は実PostgreSQL→一時的Function Data Context→限定Reference CLIを実装済み。次は#2586/#4495で製品`tei` CLI・Typed APIへ結線し、#1124の実Studio IPCと同一結果の受領を独立Expectedで確認する。[X-TEA三視点プラグインDraft #4570](https://github.com/hat47x/tei-platform/pull/4570)はStudio Bridge/CRUD/Source Navigateの局所試験があるが、実DesktopでのX-TEA原典受入は未実施で、exact-headの全体CIに失敗が残る。X-TEAを正本通り再現しつつ、Studioセルフビルドの先行必須条件としない。
5. **Source/選択の失敗を隠さない**：[Issue #4577](https://github.com/hat47x/tei-platform/issues/4577)では`tei source-index`のSubject指定時に不存在を`ok:true`・exit 0と報告する不一致が実測されている。解消までは`inspect`/`trace`等の存在検査を別に行い、空のReference集合をSubject不存在・取得不能・未使用の証拠に昇格させない。B2Cの[main #4550](https://github.com/hat47x/tei-platform/pull/4550)は**申告Graph内**のRequired多段上流依存を検証する限定Referenceで、実Native/SQL全依存の閉包を保証しない。

#### 2026-11-13 B0同一HEAD CI成功とB1の実コードFit/Gap（#4583）

- **B0の実証基線**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `b26e87781e00b19bf142b22a35cfc62a857e88a3` の [Windows CI #43368714294](https://github.com/hat47x/tei-platform/actions/runs/43368714294) は `generated-desktop-webview2-smoke` と元の `desktop-webview2-smoke` がともに成功。生成InspectorのContract digest/実受理回数と、実WebView2配信JSのSHA-256一致（682,327 bytes、`755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`）を**同じHEAD・生成版Job**で確認。Repository Health・Embed等の関連CIも同HEADでsuccess。#4572の追跡済みPython `.pyc` 2件は統合PR上で除去。ただし#4587はDraft・未merge、Stage2 Desktopビルドの証拠ではない。
- **B1-A／N0の実生成を同一Windows CIで追加検収**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)をHEAD `708ca27eb39ae31a039565429876f214a421ca52`へ更新し、[Windows Actions #43385876034](https://github.com/hat47x/tei-platform/actions/runs/43385876034)の生成版・通常版WebView2 Jobは両方completed/success。追加した生成版JobのN0で**既存Go Generator実行・独立Goテスト、契約/Transform raw digestと生成器報告値、再生成TSとWorkbenchが使うTSのバイト一致**を検査し、ログ上のN0出力 SHA-256 `6d3dfe806ef758dc4746ccc34b5ae646533eb056b78cbbc098630993d384586b`を確認。同じJobのWebView2実配信JS 682,327 bytes／SHA-256 `755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`も一致。**N0実生成→実Desktop Consumerの限定Claimは成立。ただしN0〜N3がBuildIntentを通ること、Stage2 Desktop buildは未実施**。PRはDraft・main未統合。
- **B1-Aの指定ファイル目録も実Windows CIで成功**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `d7ad13ceabc39b3917a129409630879564f8384c` の [Run #43388804188](https://github.com/hat47x/tei-platform/actions/runs/43388804188) で、generated/ordinary Windows WebView2両Jobがsuccess。生成JobはN0（契約・Transform・生成TS）、N1（Fixture・生成Go/TS）、N2（Inspector TS・main TS・Node lock・bundle）、N3（root/nested Go module・生成Go/JS・Windows exe）の**名前を固定した実ファイル**をハッシュ化し、`b1-stage-input-observation.json`を生成・Artifactにアップロード。ログ上のexe SHA-256 `9c86cd2b4e3894eb9dcca1f5c7fd5a7ed18d0c274af068a83406e9f25967f437`、WebView2配信JSは682,327 bytes／`755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`。**これは指定入力のWorkflow観測であって、依存閉包やTEI BuildIntent実駆動の証明ではない**。Evidence自体も`qualifiedBuildIntent=false`／`dependencyClosure=unknown`と記録し、Stage1によるStage2 Desktopは未実施。
- **B1-A／N0の生成物引渡しも実CIで成功**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `7babc6ad889960c5e75c1c3cbdd00a1518adeec5` の [Windows CI #43394244960](https://github.com/hat47x/tei-platform/actions/runs/43394244960)は生成版・通常版WebView2がともに成功。N0で実再生成したInspector TypeScriptをチェックイン済みInputとbyte一致検査したうえで、**実生成ファイルそのものをWorkbenchがimportするパスへ上書き配置**し、配置後SHA-256も照合した。後続のNode/esbuild→Wails native Desktop→実WebView2が同一生成版Jobで成功。実N0 part SHA-256 `6d3dfe806ef758dc4746ccc34b5ae646533eb056b78cbbc098630993d384586b`、Windows exe SHA-256 `ef545b98746a97dac27aff587b61dada206df959f034f97ad07d8397ca2b7cf0`。これで**N0→N2の限定的な実ファイル受け渡し**は成立。ただしN0〜N3がTEI BuildIntentで実駆動されたこと、esbuild/Go全Required input実読取とOS強制閉包、Stage1→Stage2 Desktop Buildは依然**NOT RUN／UNKNOWN**。工程外のCIが実行したという事実をL2/L3へ昇格しない。
- **参照Build Execution型を製品APIと混同しない**：`tools/reference-build-execution-binding/main.go`には`BuildExecutionPlan`／`ArtifactSet`等があるが、**`package main`のReference専用型**であり、`internal/buildintent`や`cmd/tei`から直接importして製品実行できない。`internal/plugin.BuildPlan`はPlugin主体の一成果物契約。B1で必要な汎用Build Execution接合はこの既存参照を評価基準とし、製品側の最小共有型／薄いAdapterへの昇格と実Invoker束縛を**具体的な差分**として判断する。Wails exeをPluginへ偽装しない（[Plugin Role設計Draft #4601](https://github.com/hat47x/tei-platform/pull/4601)も参照）。

- **B1は単一BuildIntentの増記だけでは成立しない**：現mainの`internal/buildintent/intent.go`は`BuildIntent`一件につきSubject一件、`plugin.BuildRecipe`は`kind/entrypoint`でNode/Edgeを記述しない。`Resolve`は現状`go-command`のpackage Sourceと、その他のPlugin Definition Sourceを別扱いする。`internal/buildintent/catalog.go`はBuilder候補/依存/Targetの選択を解決するが、複数工程を実行しない。`cmd/tei/cli_self_build_outer.go`は`-build-catalog`から自プロセスで一Planを導出できても、実行は指定された`-child`側であり、**Node/esbuild→Go embed→nested Wailsという全工程の実消費・順序・失敗を自動で証明しない**。
- **最小接合**：#4583 B1でN0（Inspector契約/Transform→生成TS）、N1（PoC B fixture→生成Go/TS）、N2（Node/esbuild→埋込JS）、N3（nested `cmd/tei-studio` Wails native exe）の**入力、出力Artifact role、実Invoker/Target、宣言できる既存BuildPlan/依存と未表現Gap**を実Workflowと比較。既存#1080/#1097のExecution/ArtifactSet/Subject Binding、#943のBuilder、#2831の実Build入口を優先再利用し、必要になったTarget Adapterだけを小さく追加する。Node工程を隠した単一shell child／自己申告の`required=[]`／Goの成功だけでL2とはしない。未表現は`unsupported`、未観測は`unknown`、未試行は`not-run`。
<!-- tei-master-b1b-observation-phase-20261113 -->
- **N2 esbuild実選択Inputと工程間Artifactの境界（#4587）**：[Windows CI #43410465181](https://github.com/hat47x/tei-platform/actions/runs/43410465181)、HEAD `7e77d341` の**生成版・通常版WebView2 Jobはともに成功**。生成版の実esbuildビルドで出力された`metafile`は24件の選択入力を申告し、`main.generated.ts`と`inspection-reader.generated.ts`を含むことを独立確認。選択ファイルの**ビルド後の**SHA-256は`n2-esbuild-selected-sources.json`に記録し、実配信JSとのdigest一致も維持。これにより「名指しファイルが存在する」から「実BuilderがそのSourceを選択した」へ進んだが、Builder自己申告＋事後Hashであるため**read-time byte、未宣言Inputなし、actual Provider選択、L2は未証明**。次は既存Build Execution/ArtifactSetに、実Invoker/工程間Output role/契約Targetと同じAttemptを結ぶ。
- **暗黙runspace共有を捨てる**：続く[Windows Run #43411158181](https://github.com/hat47x/tei-platform/actions/runs/43411158181)は、N2の選択出力をN3の観測結果へ表示しようとした際、PowerShellの別Stepに`$bundleOutput`が持ち越されないため**生成版JobがFAIL**（通常版はPASS）。修正は次のStepで前の変数を参照せず、**N2が検査済みの実Artifact pathと保存したmetafile**から読み直すこと。Stage/Processを跨ぐ受渡しは暗黙のスクリプト変数ではなく、明示的なSource/Artifact/Receiptで固定する。[修正HEAD `8480bdcd`のWindows CI #43411524719](https://github.com/hat47x/tei-platform/actions/runs/43411524719)で**生成版・通常版両Jobともsuccess**。N2選択入力24件、esbuild報告のmain.js出力先と実ファイルbyte長照合、N3生成exe SHA-256 `1d6722b79eafa4c0b74ab4c237cf393e1dcada27bd084d1fc9928558f7597b39`、実WebView配信JS 682,327 bytesの一致を同一HEADで検証した。失敗Attemptと修正後成功Attemptを混ぜず、B1 BuildIntent実駆動／Stage1→Stage2 Desktopは未実施とする。
- **N3 Goの実選択入力まで観測範囲を拡大（#4587）**：[Draft #4587 HEAD `001b40ce`](https://github.com/hat47x/tei-platform/pull/4587) の [Windows CI #43426489587](https://github.com/hat47x/tei-platform/actions/runs/43426489587)は生成版・通常版WebView2両Jobがsuccess。N3の**実Goビルドと同じnested module／`studio_generated` tags**で`go list -deps`が選択したリポジトリ内ソース・embed等**303ファイル**を記録し、実`go build`前後の各byte SHA-256を比較。Go選択に`main.go`とN2出力の`assets/workbench/main.js`を必須とし、`n3-go-selected-sources.json`とN2のesbuild選択24件を同一生成版Artifactに収録。生成版exe digest `e49fbbbe9ac7ccfdbefc0955a2844ffa71af8a36077e14c002ff33301769bbe0`、WebView2配信JSは682,327 bytes／`755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`。**到達はB1-Aの選択集合／前後照合**であり、実read-time byte、TOCTOU/ABA、Go SDK/Module Cache等外部依存、Node環境閉包、actual Builder Provider強制は未証明。特に現`BuildIntent.Validate`は`go-command`定義Source分岐があってもSubjectをPlugin限定とし、`plugin.BuildPlan`がPlugin Identityを要求するため、**Desktop exeをPluginに偽装してB1-Bを通すことはしない**。#1080/#1097のReference-only BuildExecution/ArtifactSetを製品側Adapterの検討材料とし、#2831の既存実Build/Invokerに**非Pluginの工程出力roleと実呼出し**を束縛する薄い接続から進める。B2 Stage1→Stage2 Windows Desktop／B3そのexeの実WebView2は未実施。
- **B1/N0に『TEI自身が実Generatorを起動』する細い実経路を追加（#4587）**：[Draft #4587 HEAD `24bfaed8`](https://github.com/hat47x/tei-platform/pull/4587)／[Windows CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)は**生成版・通常版Windows WebView2 Jobともsuccess**。既存`cmd/tei`と`reference-studio-part-generator`をWindows実exeにビルドし、**TEI実Processの既存`self-build -child`**からN0のGenerator子を起動。TEI自身が報告した親・子実行ファイルbyte digest、子argv/exit、子生成TS digestとTEIによるディスク独立再読をWorkflowで実照合し、同一Outputへの再実行拒否・byte不変も検査。別プロセスでのInspector独立再生成Reportは実ProcessのReceiptとファイルを分けて保持し、**TEI子が生成した実TSを後続esbuildへ引き渡す**。同一生成版JobのN2 esbuild選択入力24件、N3 Go選択入力303件、生成exe SHA-256 `f7acb50feb74f9e93afe61ff60fc14718b5ffaf7cf6a88c38f837a6a8f757209`、実WebView2配信JS 682,327 bytes／`755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`を確認した。これは従来のCI直Generator起動から、**限定一工程だけTEI CLIが実InvokerになったB1の進展**。ただし`tei self-build`のprimitive childであり、`BuildIntent`／Provider CatalogからBuilderを解決していない。N1/N2/N3は依然Workflow直起動、TEI CLI exeもStage1生成物ではない。**全工程B1-B／B2生成Stage1→Stage2 Desktop／B3そのStage2 WebView2はNOT RUN**。#2831既存実行入口と#1080/#1097 BuildExecution/ArtifactSetの非Plugin Target接合を続ける。AI候補の採用やOwner意味受理には昇格しない。
- **B1/N0で『宣言された実Builder byte pin』の事前強制を追加（#4587、検証待ち）**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `4dcaa199d0de655e47ee7446da6cbd5160e6218f`。既存TEI CLI `self-build` primitiveへ`-expected-child-digest sha256:…`を**任意指定**で追加し、実子のファイル内容SHA-256と**exec開始前に比較**。不一致／不正形式は拒否し、成果物を書かない。一致したAttemptのReceiptへ、Caller指定のpinと`childPinChecked/childPinMatched`を実子digestと区別して記録する。Go単体負例`TestSelfBuildSelectedChildPinRefusesMismatchBeforeRunning`でSelected B／Actual A・不正pin・一致pinを検証対象にし、実Windows統合JobのN0でもGenerator実exeの事前SHAをCLIへ渡して、**誤pinのN0生成拒否・Artifact不作成**を試す。前HEAD `24bfaed8`／[Windows CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)のN0実TEI起動成功とは別の変更であり、**本HEADのCIは確認できていないためPASSと記載しない**。このpinはWorkflowが明示した選択意図で、BuildIntent Resolver/Builder Catalogからの選択ではない。検査とOS loaderの間の差替えTOCTOU、Node/Go外部入力の閉包、再試行中の内容変化も未保証。次は既存`BuildIntent.Resolve`が得るBuilder `ArtifactDigest`を実子pinへ結合し、工程間Artifact role／Inputの実Consumerに連結する。Wails exeをPluginとして偽装せず、非PluginのBuildExecution/ArtifactSetへ薄く接続する（#1080/#1097/#2831）。
<!-- tei-b1b-builder-pin-mainline-extraction-20261114 -->
- **共通CLI側の実Builder事前pinを最新mainから独立検証（#4583 B1-Bの局所下地）**：旧[Desktop統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)はmainとの競合があり、旧ブランチの`cmd/tei/cli_self_build.go`がmainに追加された**OS境界Launcher／`driveSelfBuildIn`**を欠いていた。旧コードへのそのままの機能追加を採用せず、mainのLauncher経路を保全して選択子hashの**実行前照合だけ**を追加。共通CLIの修正を最新mainからの[独立Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `929fcf857cd0f6a792a69fb23cadfb5eccc05e90`（2ファイル、mergeable）に切り出した。Mismatch／不正pin／一致pin＋虚偽Artifact／境界Launcher拒否のGo負例を追加。Caller指定`-expected-child-digest`は実子ファイルbyteと`exec`より前に照合し、Resultには選択pin／照合実施／一致を独立記録する。**BuildIntent Resolverから選択したpinではない**ためB1-B完了ではない。Child fileのcheck→OS LoaderにTOCTOUが残り、全依存閉包やOwner意味受理を保証しない。新PRのCIを独立に判定し、#4587の既存Windows N0→N3→WebView2実装にはCI未確認の統合変更として残す。#4623にStudio／Wailsの広い差分を持ち込まず、#4587の本線は競合解消後に別AttemptでWindows CI検収する。次は`BuildIntent.Resolve`のBuilder ArtifactDigestを実起動子と照合し、非Plugin BuildExecution/ArtifactSetへ接合する。新Host／Runner／Gateなし。
<!-- tei-b1b-buildintent-resolved-builder-binding-20261114 -->
- **B1-B／BuildIntent選択→実子Builderの起動前照合を初めて実接続（選択限定）**：[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a25976946bcc8a19ef57f5490333ec8a84`では、既存`tei self-build`外側経路に任意`-bind-resolved-builder`を追加。`-build-catalog/-build-catalog-digest/-build-source-root`を指定した**製品内`buildintent.Resolve`**が作った`ResolvedBuildReceipt.Plan.Builder.ArtifactDigest`を、既存`driveSelfBuild`の`-expected-child-digest`相当の実子起動前照合へ渡す。**実Bundle/Builder新Registryは設けない**。選択SHAと実子ファイルbyte不一致時は、実子`exec`前に拒否し、受け取ったBuildIntent/Plan Receiptと`SELF_BUILD_BUILDER_SELECTION_MISMATCH`を同Attemptへ残す（ChildRunなし・Artifactなし）。一致時も`selectedChildDigest`、`childPinChecked`、`childPinMatched`を記録し、Builder一致をArtifact/業務意味/Runtime Authorityへ昇格しない。実Go試験には**正例／Selected B・Actual A負例／Catalogなし拒否**を追加し、旧CLIの未指定動作は維持。前コミット`929fcf85`の8系統CIは全成功だが、**本HEADのCIは別実行で判定**する。現在のBuildPlanはPlugin主体のため、この実例はPlugin buildの限定経路であり、**Windows Wails Desktop.exeをPluginに偽装した証拠ではない**。さらにBuilder実読取時のTOCTOU/ABA、Source/依存の完全閉包、N0～N3全工程実起動、生成Stage1→Stage2 Windows DesktopとそのWebView2は引き続き未証明。次は#4583のDesktop非Plugin工程を既存#1080/#1097 BuildExecution/ArtifactSetに束縛し、選択Target／Step Artifact role／実Invoker／Consumed bytesを一つのAttemptへ連結する。新Host/汎用Runner/第二Gateなし。
<!-- tei-desktop-selfbuild-E1-E5-crosschat-handoff-20261114 -->
- **横断チャットの成熟した境界をE1～E5の実工程で統一（2026-11-14）**：E1＝保存済みBuildIntent→`buildintent.Resolve`が導いたBuildPlan（Candidate/編集中BufferやOwner意味受理とは分離）、E2＝解決Planの選択Builder `ArtifactDigest`→起動前の実子Image比較、E3＝N0実TEI CLI生成TS→N2 esbuildが選択したInspector入力、E4＝N2の生成Workbench bundle→N3 Go `studio_generated`の実`EmbedFiles`選択、E5＝**TEI生成Stage1実Process自身**がStage2 Desktopを再構築し、その**Stage2**を実WebView2で検証する。E1/E2はPlugin主体の限定経路で[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a`の**8/8 CI成功**（[Self-build](https://github.com/hat47x/tei-platform/actions/runs/43539856841)／[Health](https://github.com/hat47x/tei-platform/actions/runs/43539856799)を含む）。E3は[Desktop Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `efe1de03`で[Windows run #43541884048](https://github.com/hat47x/tei-platform/actions/runs/43541884048)の**生成版・通常版とも実WebView2 PASS**。ただし実読取時の全入力閉包やE5を示さない。E4は#4587新HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`で`n2-bundle-producer.json`へN2成果物の`role=workbench.bundle`・format・実SHA/byte長・HEAD・Attemptを**N2終了時点**に保存し、N3の`go list -tags studio_generated`が選んだEmbedFilesの**Go Build開始前digest／byte長／HEAD／Attempt**と照合、不一致ならN3実行前拒否する実装を追加。別コピーに1バイトを加えた負例を**同じ照合関数**で拒否し、N3実行後も再ハッシュを記録（`n2-n3-bundle-handoff.json`）。[Windows検証run #43542664470](https://github.com/hat47x/tei-platform/actions/runs/43542664470)は**生成版・通常版Windows WebView2 Jobともcompleted/success**。生成版の実ログ：`N2-producer-bundle-recorded`と`N2-N3-embed-handoff-verified-before-go-build`は双方`sha256:755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`、改変コピーの負例とGo build後digest照合もpass、実WebView2配信JSは682,327 bytes／同一SHA。N2 esbuild選択入力24件、N3`go list`リポジトリ内選択入力は今回**312件**（前HEADの303件を流用しない）、生成Windows exe SHA-256は`e2158bc8e20a5f2fe153ca96d08aad1571d2bacc9c87672b288fbfcbe5566205`。E4成功でも、N3 Go compilerのread-time捕捉・cache/external dependency完全性・実exeのbyteにembedされたことの独立証明は別票。E5は**NOT RUN**。**Build包含／実行利用／Authority/Effect／データ保持の8状態を混同しない**：BuildへJSを含めたこと、WebView2が実JSを使ったこと、Ownerが意味受理したことはそれぞれ別判定。StudioはCandidate/PreviewのUIであり意味正本でも独自Host/Approvalでもない。NonPlugin Desktop対象は#1080/#1097のReference `BuildExecutionPlan`/`ArtifactSet`を最小採用し、旧Plugin BuildPlanを偽Plugin IDで使わない。保存済みSource・Generator/Runtime Provider・行動Authority・Consumerを分離し、E4実検収→E5生成Stage1→Stage2へ最短接続する。新Build Engine/Generic Runner/第二Approval Gateは設けない。
<!-- tei-studio-main-merged-b1-n0-webview-ci-20261114 -->
- **2026-11-14 現行Desktop統合HEADへmainを実merge（#4583）**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)は、mainの現行HEAD `e888bb8f21367b721f0984953530da9627e2634d`を第二親とするmerge commit `03679ccd12b2780b4b6e4890792d49518f4461be`で**behind=0／mergeable=true**に復帰。22差分ファイルのうちmain側変更と交差したのは`cmd/tei/cli_self_build.go`と`plan/evidence/README.md`の2件のみ。前者は最新mainの境界Launcher／`driveSelfBuildIn`を残し、共通Builder pin [Draft #4623](https://github.com/hat47x/tei-platform/pull/4623)の実子SHA照合と同じGit blobを保持。後者はmain側の全索引更新を残してInspector生成文書への一行を追加。**単なるファイル上書き／旧mainへの復帰は行っていない**。続く実装HEAD `efe1de03c23d7e8bd9271e465fc0b8436f9a993c`ではWindows Desktop Workflowの`pull_request/push paths`へ、N0で実際に使う`cmd/tei/**`と`internal/buildintent/**`を追加。変更検知から実TEI CLI/BuildIntentが漏れた欠陥を解消した。[新HEAD CI（Windows含む）](https://github.com/hat47x/tei-platform/actions/runs/43541884048)は**main merge後の別Attempt**として試験中であり、古いWindows success（#43450255223）を新HEADの結果に流用しない。**共有CLI PR #4623 HEAD `f1e219a`の8系統CIは別HEADとして全success**（[#43539856841](https://github.com/hat47x/tei-platform/actions/runs/43539856841) self-build、[#43539856799](https://github.com/hat47x/tei-platform/actions/runs/43539856799) Healthを含む）。ただし#4623の正式BuildIntent `-bind-resolved-builder`は未mergeで#4587のN0 Windows Jobには含まれず、そこではprimitiveのCaller pinを使う。BuildIntent選択→非Plugin Desktop全工程の実Invoker連結、Stage1生成TEI→Stage2 Windows Wails構築→**そのStage2** WebView2は依然未受理。N0～N3のArtifact roleとRead-time消費を既存#1080/#1097 BuildExecution/ArtifactSetに薄く束ねるのが次のP0であり、Wailsを偽Plugin扱いせず新Runner/Gateを増やさない。
- **B1-Bで先に塞ぐ『事後Hash＝実読取』の誤認**：[#4587のB1-A実装レビュー](https://github.com/hat47x/tei-platform/pull/4587#issuecomment-6847421815)で、`b1-stage-input-observation.json`の`n2.inspectorInputSha256`・`n3.bundleInputSha256`・`n3.generatedGoServiceInputSha256`は、**N3 Windows Go Build後に名前付きファイルを再読取した値**であることを確認した。`invoker`も現状はWorkflow由来の説明文字列で、実Child image/argvの証跡ではない。B0/N0の実Windows Consumer成功は保持しつつ、この目録を実esbuild／Go Builderの入力閉包・現在資格へ昇格しない。B1-Bの最小作業は、各実Invokerの直前・直後の入力byte／選択Builder／出力を同一Attemptで照合し、差替え・未宣言transitive import・Selected B／Actual Aを既存[#4583の独立Expected](https://github.com/hat47x/tei-platform/issues/4583#issuecomment-6847431909)で拒否すること。**pre/post一致でもTOCTOU/ABAや実Readの全閉包は未証明**であり、#2595/#4486に従い不明を残す。`tools/reference-build-execution-binding/main.go`のExecution/ArtifactSetはReference-only（`package main`）なので、これを正式Product APIとして直接importせず、#1080/#1097の非Plugin Build実行／Subject束縛の境界を維持して小さなAdapterを検討する。
- **B2/B3の停止線**：Stage1のTEI実ProcessがStage2の**Windows Desktop**を起動・構築したこと（実Process image/argv、実Builder消費入力、Stage2 exe digest）と、**そのStage2 exe**が生成Inspectorを実WebView2で使ったことを別確認する。#4572のLinux局所部品再入や#4587のWindows外部Workflowビルドはこの代わりにならない。remote実行を行うなら別Target/Authority契約を明示し、暗黙のWindows fallbackをしない。
- **対象発見の独立性（main修正済み）**：#4577で観測された不存在Subjectに対する`source-index`の`ok:true`・空参照は、[merged #4584](https://github.com/hat47x/tei-platform/pull/4584)で修正された。Subject指定モードは既存`InspectionFromLoad`で実Canonical対象を確認し、不存在なら`SUBJECT_NOT_FOUND`／exit 1、unsupported kindならexit 2を返す（PR HEAD `dcfe1d69`の対象8 CIはsuccess）。**存在するSubjectの参照0件と、Subject自体の不存在を分離**してPin/AI/SHOへ渡す。ただし`internal/operations/trace.go`の参照列挙や`source-index`成功だけで、別Workspace Rootの同一性、現在Scope・Grant、必要な外部Sourceの取得完全性までは証明できない。固定Pin・生成Intent・Handoffでは各Productがそれらを再確認する。

**実装のOwner**：B0〜B3は[#4583](https://github.com/hat47x/tei-platform/issues/4583)、全体Builder/Stage再入は#2831、Windows Desktopは#1824、独立Expectedは#1180/#2514を維持する。X-TEA三要素のModelerプラグイン#4570、Function生成#4559、固定Subject Pin#4515は並行し、B1の新Hard Gateにしない。本追補は実ソースと既存CIの照合に基づく計画であり、**B1〜B3の新しい実BuildやSelfBuild資格・Owner受理は未実施**。

**検収票の分離**：生成したSource、実Builder入力、生成したArtifact、Stage1実Process、Stage2実呼出し、実Desktop WebViewでの使用、外部Provider/Consumerの受領を別欄に記録する。各段階に対応した実失敗・Source改版・未取得/partial・同名別Workspace・旧bundleの負例を独立Expectedで確認する（#1180/#2514）。**今回の計画追補は実装・GUI・CLI・Stage2の新たな製品E2Eを実行したものではない**。

## 現在の判定段階と引継ぎ

### 判定段階A：アプリケーションの意味を実行・検証環境へ渡せる

**Ready**です。

アプリケーションの構成パターン、データ接続先、検証規則、認証・認可、能力と実装の対応付け、業務上の結果と実行時エラーなどについて、画面固有の定義を必要とせず実行・検証環境へ渡せる意味の基準が成立しています。

### 判定段階B：画面なしで主要な業務シナリオを再現できる

**Ready**です（2026年9月19日の再監査。候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

必須の検証根拠は次の三つです。いずれも、変更のない同じ候補版`R`を対象とした一度の実行で成立しています。

1. #310 リポジトリの決定論的な再現性に関する検証根拠
2. #255 生成成果物の再現性に関する検証根拠
3. #308 実際のPostgreSQL提供実装と一時的な試験環境による検証根拠

`#387`での最終確認と、同じ実行で取得した子処理の成果物の独立監査を踏まえた`Ready`の判定記録は、`GATE_B_READINESS_AUDIT.md`を正本とします。

PostgreSQL以外の実際のデータベースによる検証根拠も重要です。最初の参照事例では、Oracle Databaseに関する#214・#371・#376が該当します。ただし、判定段階Bの第四の必須検証経路には含めません。未完了の範囲は`unresolved`として引き継ぎます。

### 引継ぎの基準：判定段階BからCへ

引継ぎは、新しい番号付きの判定段階ではありません。

判定段階Bで`Ready`となった候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`を、引継ぎ基準版`B`として固定しています。正式な記録は`GATE_B_TO_C_HANDOVER_BASELINE_B.md`にあります。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
```

引継ぎ記録を、意味についての第二の正本にはしません。

### 判定段階C：最初の実行先で生じる違いを検証根拠で説明できる

**Ready**です（T = `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026年9月19日）。判定段階Bと引継ぎ基準版を起点に、最初のウェブ実行先の構築版`T1`について、意味・機能・表示の検証根拠を同じ来歴で揃えました（[判定段階Cの監査記録](gate/GATE_C_READINESS_AUDIT.md)）。この`Ready`は、本番運用が可能であることや、実際の外部環境で命令が成功したことを意味しません。実行先に依存しない表示の分類体系の完成、二つ目の実行先の不要化、表示の忠実度に対する人の承認も示しません。これらの限界は同じ監査記録に残しています。

最初のウェブ実行先では、引継ぎ基準版`B`と対象版`R`を意味の基準として使います。

ウェブ向け実装の版は`T1`として、基準版とは別に追跡します。

```text
Baseline meaning B at R
        ↓
First Web Target build T1
        ↓
Semantic / Functional / Visual Evidence
```

`T1`で基準版の対象の意味を変更する必要が生じた場合は、そのまま作業を続けず、新しい引継ぎ基準版を発行します。

判定段階Cでは、次の三種類の検証根拠を区別します。

1. 意味の準拠確認
2. 操作・機能の検証根拠
3. 表示の忠実度の検証根拠

画像を評価するAIの判断や画面画像の評価点だけで、意味や機能の検証を合格とは判定しません。

## 作業領域A：製品の基準と追跡可能性

### 目的

製品価値・要求・製品受入シナリオ・設計判断記録・計画上の課題・参照事例・検証根拠・判定段階を、一続きに追跡できる状態を維持します。

### 重点

```text
Product Requirement
        ↓
Product Acceptance Scenario
        ↓
ADR / Plan Issue
        ↓
Reference / Fixture
        ↓
Evidence
        ↓
Gate / Product claim
```

PASは単なる網羅率の指標ではありません。要求と実証を結び付ける追跡単位として扱います。

**CI・main健全性の横断運用（2026-11-12、実行計画案）**：製品の実受入と開発時CIの状態を混同せず、現mainのexact HEAD／run attempt／必要scope／環境・ツールチェーンを結ぶ。過去失敗・未実行・意図的停止・runなしを合格と誤認しない。PR→mainの再観測、差分に応じた検証、AI Harnessの実Tool/Consumer証拠、NAS/S3等への大容量Evidence退避と取得可否の扱いは、[main健全性とCI Evidenceの横断運用計画](workstreams/CI_MAIN_HEALTH_AND_EVIDENCE_CROSS_REPO_2026-11.md)にまとめる。既存AGENTS/品質設計・#2018/#2514/#2515/#2600の責務を維持し、新Gate・第二のRegistry・全リポジトリCI自動再有効化は行わない。

## 作業領域B：意味の定義・記述・知識

### 目的

定義元のソースから業務上の意味を再構成し、複数の表示形式や編集方法で同じ正本の意味を扱える基盤を整えます。

### 対象

- 意味を扱う中核
- ソース索引と診断
- 入力元から意味の正本への追跡可能性
- 分野固有の部品、意味の構成パターン、方針の候補
- 出所と対象の版
- 再利用できる意味情報の探索

第1版では、アプリケーションの構成パターンをアプリケーション側で管理します。最初の参照事例だけを根拠に、意味を扱う中核の標準的な概念一覧へ移すことはしません。

AI向けの知識・検索索引・埋め込み表現・処理履歴などは派生情報として扱い、意味の第二の正本にはしません。

判定段階Cの後に行う読取り専用の探索は、`ISSUE-000047`でPAS-06として実証します。

## 作業領域C：アプリケーションと既存資産の互換性

### 目的

新しい理想形だけを対象とせず、既存の業務資産を保持しながら段階的に移行できることを確認します。

### 対象

- アプリケーションの構成パターンと意図
- アプリケーションとデータベースの関係
- データベース契約と元形式の問い合わせ
- データ接続先の読取りと書込み
- 既存の認証と認可
- 既存のSQLと外部のAPI
- 機能と実装の対応付け
- 段階的な置き換え

実際の外部資産については、観測した事実と、正本に記述した契約を区別します。

実際に観測していない物理的な構成要素・列・SQL・データ型は、推測して補いません。

成熟した外部の参照モデルを移し替える作業は、`ISSUE-000078`と`REFERENCE_MODEL_TRANSFER_GATE_2026-10.md`で扱います。必要な業務・設計上の概念は、原則として意味の拡張やプラグインで表し、X-TEA固有のスキーマを中核へ直接追加しません。外部モデルの本体を公開の試験データへ複製せず、元資料の照合値・参照位置・対応付け・失われた意味・未解決事項・準拠確認の根拠から追跡できるようにします。

## 作業領域D：データベースと提供実装の変更管理

### 目的

期待する状態・実際の状態・差分・移行・提供実装の境界を、提供実装固有の知識を中核へ持ち込まずに成立させます。

PostgreSQLは詳細な検証に用いる提供実装として利用します。

異なる提供実装は、既存資産との互換性と、特定の提供実装に依存しない境界を検証するために利用します。Oracle Databaseは、最初の参照事例の`master_data`が利用する既存資産の一例であり、検証対象をOracle Databaseに限定するものではありません。MySQLやSQL Serverなどの追加は、具体的な資産と利用条件が明らかになってから判断します。検証では、物理的な事実の観測・読取り専用の制約・実行環境との接続・実際の読取りを区別します。

Oracle Databaseの実環境での検証が未完了であることだけを理由に、固定版での判定段階BのReadyを取り消しません。

### 汎用データソースの開発者向け管理範囲（2026-11-07、#3270）

SQL/NoSQLのデータ、ログファイル、クラウド情報源は別々の製品機能を増やすのでなく、既存Data Operations・Provider・標準データ構造と表示型から構成する。**スキーマ／テーブル／索引／制約の作成・変更・削除をTEI標準に含める**。これらはアプリケーション設計・データ管理の責務であり、DBA/インフラ運用（サーバ、クラスタ、バックアップ、レプリケーション、DBユーザー管理など）は標準外とする。実行時の対象・権限と破壊性は別途検査する。

[汎用データソース・ワークベンチの設計・縦断実証案](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)は#3270の計画として、①現状契約棚卸し・read-only構造観測、②実DBでのReverse/Schema Plan、③Disposable DBでのCREATE/ALTER/DROPと独立読取り、④ログ/NoSQLの部分取得・時系列・型揺れ、⑤CLI/MCP/Studioの共通投影、⑥組合せBOM/Cache資格の検証を段階化する。#1126・#1803・#2622・既存Schema Diff/Migration・#1641・#3267へ責務を戻し、第二のSchema/Query/Cache/Authority正本を作らない。**既存の判定段階B/Cと主線優先順位は変えず、設計・受入案と実装済みを区別する。**

**対応する製品要求は[RQ-035](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-035-一般開発者が異種データソースのデータと構造を管理できること)**。2026-11-07に現行コードを静的確認した結果、`internal/database/provider/provider.go`のDB専用SPI、PostgreSQL `Introspect`、`internal/workbench/database/model.go`の派生投影、`cmd/tei/main.go`のSchema Diff/Migration入口は存在する。一方、異種Sourceを含む実Provider Read→完全性付きFunction Data Context→標準表現型の一往復は未確認であり、#2586と最初の不足一辺を検証する。Schema Historyのunknown→applied誤分類候補は既存#2892へ渡す。Trait・Decision Graph・C0/C1・独立期待値への接続とWB-15〜25は上記設計書の11〜15節に集約し、ここへ第二の契約正本を作らない。静的確認は実Provider/Go/GUIの実行成功を意味しない。**

### Core外PluginによるWorkbenchの同一契約実証（2026-11-07、#3270×#1251）

[#3277](https://github.com/hat47x/tei-platform/pull/3277)の[Plugin-first CLI/Studio/Runtime設計](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) D1/D2/D4/D5とWorkbench W0〜W5を**依存関係で接続**する。最初にCLI/Studio共通のInspector一操作を同一Contract/Providerで通し、次に**PostgreSQL実ReadとJSONLファイルRead→Function Data Context→Table/Tree/Timeline**を、同じContract選択と異質なProviderで比較する。成果物を読む`workbench-model`と実Provider Row Readは別資格。次に既存Schema Diff/Migrationの隔離DB適用、最後に同じRead/Validator契約を開発用Surfaceなしの生成Goバイナリへ組み込む。

[設計書の§16〜18](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)に実ソースの接続点、Selected/Linked/Invoked、異Alias競合、CLIの呼出別Effect表示、WB-26〜37の独立期待値と停止線を配置し、[合成SQL＋JSONLのFixture](../examples/data-source-workbench/README.md)を受入入力に固定する。Provider/Plugin/Contract Resolverの実装は#951/#1251、実Row→Data Contextは#2586、異DB Aliasは#2611、CLI Effect語彙は#3182、Migration Historyは#2892、検証済みBOMは#3267がOwner。新しいPlugin Registry、Host、Query/Cache/Authority正本を作らない。**この段階では静的コード確認＋Fixture作成のみで、実Plugin選択/Build/Provider接続/Studio/Goテストは未実施**。
### 実Catalogの物理Relationから三視点候補への引渡し（2026-11-08、#3270）

SQL/NoSQL/ログ等のリバース結果は、**Physical Observation→Structural Candidate→Accepted Data/Function/Business→Diagram/CRUD/SHO**の各段階で資格を分ける。物理FK/UNIQUEやログ中の同一文字列だけから業務Relation、FunctionのCRU(D)、実行権限を生成しない。Catalogの0件や未取得を業務上の「関係なし」にしない。

[Workbench詳細設計§19〜20](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)へWB-42〜54と三経路（Schema/実Data Read/三視点）を登録。現行`postgres.Provider.Introspect`→`databaseworkbench.Build`の**合成Catalog read-only経路**でCapturedAt/Fingerprint、UNIQUE/FK・複合参照・IndexのNative式/部分索引/Readinessを落とさずTable Viewへ投影する狭いProduct Goコードはmainに存在するが、その検証は合成Catalogに限られ、実DB/Scope・Row Read・Schema Applyの資格とは異なる。実Provider/Studio/SHOの適格やOwnerによるMeaning受理は別。機能Readは#2586、Required Design Contextの完全性は#3262、三視点Diagram/CRUDのDraftは#3263/#3303、Plugin-first D1は#1251がOwner。実DBのSchema定義/変更操作は標準、DBA運用は非標準の境界を維持する。

**2026-11-08 横断調整**：[Workbench設計§21](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)にREAの外部原本→Evidence→正式TEI Source→実Read/SHO比較、SUI/SEI/KONの任意Consumer境界、共通Operation（ADR-0048）、Migration適用直前のInstance/History/Grant再照合を集約した。最短P0は実Provider Read→Function Data ContextとPlugin-first Inspector二Consumerであり、SUI・SEI・KON・SHOやOntology DSL製品完成を新たなHard dependencyにしない。**

**2026-11-12／REA外部候補の入場を横断計画へ反映（#3394、計画・受理境界）**：実REA 6.7.0の単独JSONと6.8.0の7 Record＋JSONL終端の版別検査は[main #4257](https://github.com/hat47x/tei-platform/pull/4257)で限定実証済み。一方、[PR #4322](https://github.com/hat47x/tei-platform/pull/4322)の`operations.ExternalCandidates`は保存実捕捉Fixtureからの外部Observation 1件＋方向・Edge ID付きRelation 1件の型付きread-only受領を局所実装・試験しているが、**2026-11-12確認時点でopen/unmerged**。元raw byteと派生入力のdigest、Producer/Attempt、Source位置・版、External Node/Edge ID、parallel relation、coverage/unknown/partial、選択部分集合を保持し、外部CandidateをCanonical Subject・業務FK/CRUD・Accepted Meaning・実DB観測へ自動昇格させない。CLI entryの局所成功をMCP/Studioの独立Consumer成功とは数えない。

**最小の検収順**：①#4322のexact HEADとmain差分を照合し、版違い・終端欠落・外部ID混同・向き逆転／平行辺縮約・unknownの観測化・1+1を全Graphとする偽PASSを拒否した上でmainへの受入を判断、②同一型付きOperation結果をCLIと独立したMCPまたはStudio Consumerで実受領し、送出範囲・Disclosure、selected/invoked/received、現在のSource改版・A→B→A失効を別に検収、③選択・Owner受理した候補だけを正式Authoringへ提案。正式`*.tei.md` Reader（#3346）、三視点Modeler（#1124）、SHOのSave/reopen、外部Provider実Rowset（#2586）・旧新DB Migration（#4125）の資格を足し合わせない。[TEI文書Draft #4317](https://github.com/hat47x/tei-platform/pull/4317)・[再投影方針 #4467](https://github.com/hat47x/tei-platform/pull/4467)・SHO #36/#40のmain統合は再利用するが、著者文の自動マージ、引用自動解決、Occurrence ID永続化、Meaning受理は行わない。Legacy Observed Actual／Owner Accepted Desired／Target Observed Actualの三状態は#4125の既存G0〜G5で別々に照合する。

**責務と停止線**：REAの詳細な受入票・独立Expected・最新状態の正本は[#3394の2026-11-12横断追補](https://github.com/hat47x/tei-platform/issues/3394#issuecomment-6813994847)。共通API=#1158/#3419、Provider実Call=#1251、Source/Context=#3262、実Function Read=#2586、DB移行=#4125に帰属。第二Parser/Host/Registry/Meaning Ledger/移行Gateは作らない。新規Consumer実E2E・正式Source Loader・実DB/Owner受理は**NOT RUN**。既存P0の一Function実ReadとPlugin-first実Consumerを、REA T1-B/SHO完成でブロックしない。


### REA受領Readerのmain統合とDesktop優先の実装順（2026-11-13、#3394）

**実装済み（本節の追補時点）**：[PR #4547](https://github.com/hat47x/tei-platform/pull/4547)をmainへ統合（squash `030936b3a48690b63150a8495efa4dc8634eb1c2`）。既存REA 6.7.0/6.8.0の保存済みバイト列からReceipt／構造候補を読む `internal/externalanalysis` と実捕捉Fixture・負例、既存共通JSONLの受領済みbyte読取を再利用可能にした。全体Health・セルフビルド・キャッシュ・依存検査・ドッグフードの**5系統CIが合格**した固定HEADに基づく統合である。ただし**Readerのmain入り≠製品Operation・通常tei CLI・Studio/MCP独立Consumer・Owner意味受理・実Provider/DB観測の完了**。

**次の小分割**：[PR #4548](https://github.com/hat47x/tei-platform/pull/4548)をmain起点の独立5ファイル差分へ組み直した。外部Observation 1件＋有向Relation 1件のread-only `operations.ExternalCandidates`、Producer ID/Derived Handle重複の**曖昧選択拒否**、期待Evidence ID・受領Raw Digest・Producer宣言Source Revisionによる**旧Pinの選択前拒否**と独立Go負例が対象。専用Go契約の成功は確認済みだが、**本追補時点ではmain向けの全体Health・セルフビルド・Wails検証を実行中**として、main合格や実Consumer受領を先取りしない。ハッシュ一致は過去の受領byte一致であり、元Sourceの現在性／A→B→A／Workspace Root／開示資格／実行権限／業務Meaning受理ではない。

**製品全体の順序**：最優先は**TEI Studio Desktopの実Wails Desktop接続とTEIによるセルフビルド**（#1824/#3421；S0 実build→起動→Catalog/Inspect→共通API、以後S1〜S4で再生成／再入場／実OS検収）。REA R1/R2はこれを阻害しない独立read-only線であり、REA/SHO/三要素分析法すべての完成をDesktopセルフビルドや実Provider Read→通常CLI（#2586）の前提にしない。R2のmain受入後のR3は**通常CLIと独立した第二Consumerで、同じSelected/Invoked/Received・Digest/Attempt/Coverageを実受領**する最小縦断とし、独自のHost/Registry/Parser/Grant Engineを作らない。REA観測から汎用Graph・三要素分析法のData/Function/Business、SHO文書へ進む際は、Source根拠・外部Identity・Scope・意味の明示的採否を分離する。最新版・Ownerは[#3394](https://github.com/hat47x/tei-platform/issues/3394)を参照。

### REA受領から編集までの変更伝播規約（2026-11-12、#3394×#1124×#1126×#3270）

**対象を三つに分ける**。①REA／外部DB Reverseの`Observed Schema`、②TEIで編集・受理する`Table Definition`、③Table Definitionを写像した`Data Model`は、同じ名前でも別の出典・版・適格性を持つ。実DBとの差分は同期候補であり、**外部観測の欠落やREAのpartialをTEI定義の削除として扱わない**。正式な`Table Definition`の変更は、その同一版に束縛されたData Model投影と、機能モデルUIのBinding・CRUD・Query参照・SHO引用へ**変更影響**を通知する。外部REA Node/Edge IDと正式TEI要素ID、Canvas上のOccurrence IDを混用しない。

**GUI/CLI共通の操作意味を固定する**。画面上でUI項目を追加しても、テーブルや列は自動生成しない。必要なら「新しいテーブル項目を作成して結び付ける」を明示的に指示でき、Studioの操作とCLI/CUI・AIハーネスからの操作提案は**同じ型付きIntent・Preview・診断・更新前Revision検査**を通る。明示された編集によってTEI Table DefinitionのField IDを削除したときは、Data Modelの対応項目を同じ編集結果から消し、機能モデルのUI項目は**残したままBindingを解除し未マッピング警告**を出す。元のFieldと同名の別IDへ自動付替えしない。参照するQuery・Rule・CRUD・画面処理・文書・固定Selectionには影響／失効を表示し、削除された列への暗黙保存は拒否する。変更前に依存先をPreviewできるようにするが、Authoringでの削除が**実DB DDL Apply済み**であることは意味しない。

**実DB側への反映は別のEffect**。Workbench／Migrationの既存`Schema Diff→Plan→Apply→独立Read-back`を再利用し、破壊的DDL、対象Alias・Principal・Grant・読取時点と履歴は別資格で確認する。DB側で列の不在が観測されても、TEI定義・Function UI・Owner受理は自動変更せずDriftとして示す。逆にTEI定義を削除しても、実DBはApplyと再観測が済むまで`Target Observed Actual`として旧状態の可能性を保持する。Modelerの表示一致やREA EvidenceはDDL・DB Read・Meaning受理の代替ではない。これは過剰な承認プロセスの導入ではなく、通常開発の変更確認と生成AIの誤推定防止である。

**独立Expectedと偽PASS防止（仕様候補、未実装）**：

| Case | 操作・反例 | 必ず成り立つこと |
| --- | --- | --- |
| REA-MAP-01 | 機能モデルにUI項目だけを追加 | Data Model／Table Definition／実DB列は増えない。UIは未マッピングを表示 |
| REA-MAP-02 | 明示操作でTable DefinitionにFieldを追加してUIへ結合 | 事前Previewと版照合を経て、Field IDとUI Bindingが一致。物理DDLは未適用扱い |
| REA-MAP-03 | Table Definitionの参照中Fieldを削除 | Data Model対応項目を削除、UI自体は存続しBinding解除＋診断。旧FieldへのWrite拒否 |
| REA-MAP-04 | 削除済みFieldと同名の別IDを再作成 | UIの自動再結合なし。Pin Selectionも旧IDを指し、別対象へ黙示変更しない |
| REA-MAP-05 | REA／実DB Reverseのpartial・permission-denied・0件 | 不在を削除と断定せず、Observed差分・unknownを保持 |
| REA-MAP-06 | 同一Field IDのrenameと、別Fieldへのreplace | renameはID基準でBindingを保持、replaceは明示対応付けがなければ未マッピング |
| REA-MAP-07 | TEI定義を削除したが実DBのDROPは未実施・失敗・結果不明 | Definition/Projection更新とDBのActualを区別し、移行成功の偽PASSなし |
| REA-MAP-08 | 旧Revisionで作成したUI/AI編集案をSource変更後に適用 | staleを検出し再Preview／再指定を要求。旧選択の自動付替えなし |
| REA-MAP-09 | 同じ変更をGUIとCLIから提案 | 同じField ID・影響集合・診断意味・効果境界を返す。GUIだけの隠れた更新なし |
| REA-MAP-10 | SHO著者が説明文を編集した後にField削除／REA再解析 | 文書引用はstale表示、著者文の無断書換えはしない。正式Source再解決を別確認 |

**責務と実行順**：定義とData Modelの連動・Modeler投影は#1126/#1124、UI Binding／変更逆写像は機能モデルと#2500、外部観測とREA入場は#3394、実DBとの差分とDDLは#3270/#4125、Source Revision／失効とCLI・AIの固定選択は既存Operation契約、SHOは#4317/#4467の再投影に配賦する。最初の実装出口は`REA-MAP-01/03/04/05`を同じread-only Inspector・編集Preview上で独立Expectedに照らすこと。その後、明示Add/Edit・GUI/CLI一致・実DB Read-back・SHO再読込へ進む。**原典の三要素分析法に忠実な機能と、TEI独自の編集支援を区別する**。本節はPlanであり、実製品E2E・Owner意味受理・DDL適用は**NOT RUN**。

### モデル変更からScript起動までの連鎖を最小化する受入順（2026-11-12、#3394×#1124×#4515×#4502）

**既存二経路を混ぜない。**「外部REA／Workbenchによる観測→候補→TEI定義への明示的採用」は#3394の外部Evidence入場であり、「受理済みTEI Table Definition／Function／UI宣言の編集→参照切れ診断→必要時の再接続」は#1126/#1124の編集経路とする。どちらも`Preview / Change Intent / Source Revision check / Authoritative Source write / Reprojection / Diagnostics`の既存契約の範囲で検討するが、同一Operationで自動的に外部Sourceや実DBへ書き込まない。OKFは文書や根拠のIdentity・Version・Links・Lintを通常のGit／レビュー／試験と併用する。専用Owner意味承認Engine・第二Ledgerは設けない（#4143のK0〜K5は用途別Admissionと独立に検収）。

**実装と機能公開の連鎖を切る。** UIの新規追加・画面変更・REAのFunction候補発見は、テーブルFieldの新設・公開Entry・Trigger/Script起動の作成を**暗黙に実施しない**。Function／Entry／Fieldを削除・rename・移動・版変更した場合は、依存するFunction UI Binding、Script/Entry Input Mapping、Trigger Binding、Query/CRUD、SHO引用のうち各々の**未解決参照を残して診断**し、同名・同offset・同表示位置の別IDへ自動付替えしない。依存先の意味と実体を同定できるrenameはStable Identityと版の検査によって継続を判定し、同定できないreplace/split/mergeは候補提示へ戻す。Triggerは保存・発火しても現在のEntry公開/Host Grant/Artifact資格を示さず、Script Launch P0の実起動Ownerは#4495、manual/schedule/event/batchのbindingは#4497/#4502へ渡す。

**Pinは段階的に適格化する。** #4515の現行Studio Workbenchにおける最初の実装対象は`WorkspaceRef + Subject(kind,id)`の**read-only Pin Intent**。ユーザーの明示Pin/Unpinと一時選択・ハイライトは独立とし、Workspace変更、Source改版、同offset別Subject、A→B→A、後着返信では`needs-requalification/unresolved/unknown`を返す。現在のInspection事実を現在のGrant、実行権限、外部Harness送信許可へ昇格させない。Studioの`parent:<subjectKey>`のような**描画用Edge IDを正式RelationのOccurrence Identityとみなさない**。Relation/CRUDセル/SHO spanのPinは、別Ownerで出典裏付けのある正式Relation識別子を導入・再解決できた後の受入対象とする。Subject Pinの初期実装をGraph全機能の完成に依存させない。

**次の小さな独立Expected（上記REA-MAPの拡張、採用済み・試験済みではない）**：

| Case | 入力・操作 | 期待・拒否 |
| --- | --- | --- |
| REA-X-01 | UI項目を増やす／UI項目のみを削除 | Table Definition・実DB・公開Entryの増減なし。明示したUI変更のみ |
| REA-X-02 | Fieldを明示削除し、同Fieldを複数UI/Query/CRUD/Script Inputが参照 | Data Model写像からFieldを除き、UIは残して未Mapping警告。すべての依存先の参照切れを検出し暗黙Write/Launchを拒否 |
| REA-X-03 | Function/Entryの削除・rename・同名別IDでTriggerを再評価 | Bindingの一意な現在性が確かめられなければunresolved。自動再公開、再実行、別Functionへの付替えなし |
| REA-X-04 | REAの外部Graphに平行EdgeがあるがStudio描画EdgeにしかIDがない | Subject Pinは許容しうるが意味Relation Pinはunsupported。業務Relation受理と誤表示しない |
| REA-X-05 | Subject Pin後にWorkspace切替・Source改版・同offset A→B→A・旧Reply後着 | Pin Intentを維持しつつ現在資格を失効。silent retarget/旧返信の現在表示なし |
| REA-X-06 | GUI・CLI・AIから同じField編集案を要求 | 変更前・後の対象Identity、影響集合、診断意味、Source版、Action/Effect境界が同じ。AI案だけに権限追加なし |
| REA-X-07 | 外部REAのpartial/unknown、またはOKF broken linkを含む文書を読む | 形式の取得成功と「全項目不在」「意味受理」「Current Grant」を混同せず、用途に応じunknown/needs-review |
| REA-X-08 | Triggerが発火したがEntry未公開／Artifact版不一致／現在Grant欠落 | 起動しない。Trigger occurrence・Invocation/Attempt・Effect Receiptを独立観測し、ACK成功を業務成功としない |

**非重複の実装配賦・受入ゲート**：P0-A=#4322からREA Observation 1件＋有向Relation 1件の現main移植（製品API・受領bytesの独立照合）。P0-B=#4515で**Subject Pin**の初回GUI実Consumer（read-only）。両者は**並行**し、互いの未完了をHard dependencyにしない。P1=#1126/#1124で正式Table Definition更新のMapping解除とCLI/Studio診断の同値性（REA-X-01〜03/06）、#4495のpure Function二入口Invokeは別レーンで先行。P2=#3394の外部Graph候補から正式Relation/三要素分析法のモデル・CRUD/SHOへの**選択的**取込、#3270の実DB観測/Apply/Read-back、#4497のTrigger Effect・#4143のOKF K2以降を段階的に結合する。X-TEA Modeler原典に基づく三要素分析法の図面・機能階層・業務モデル・相互参照は#1124/#1125/#1031の忠実性検証を維持し、TEI独自のAI/Pin/Script拡張を原典機能だと記述しない。**本計画は仕様候補であり、実Go/Studio/REA Product E2E、実Provider Read、Owner受理、実DDL/実EffectはNOT RUN。**

### REA Evidence取得経路とStore契約の分離（2026-11-12、#3394×#4307×#4503）

[REA受領PR #4322](https://github.com/hat47x/tei-platform/pull/4322)の現実装は`ExternalCandidateRequest.EvidencePath`を受け`os.ReadFile`してから、`externalanalysis.Receive(raw, tool, at)`と同一Readerで解釈する。これは**保存済みローカル証拠を読む初期Adapter**として有用だが、ローカル／NAS／S3等の任意Evidence Storeの共通入力契約と同一視しない。Evidence実体・内部Locator・digest・件数をGitHubに必ず送る設計にしない（Storeの所有者=#4307、Local CAS実証=#4481、NAS/S3比較=#4494、用途別の取得・固定選択・送出=#4503）。

**最小の接続順**：P0：#4322のReceiver／型付きExternalCandidatesと、版別6.7.0/6.8.0実捕捉Fixtureを**現mainへ最小差分で切り出し**、ローカル保存byteを読み、Observation 1件＋有向Relation 1件を、Coverage/Unknown/Unselectedを保って共通read-only Operationへ投影する。41ファイルを一括マージせず、`externalanalysis`の受領・候補Readと`operations`のProjectionを先に再利用し、CLI・Document・専用CIは小さく後続へ分割する。P1：同じ受領・投影ロジックへ、既存Local CASの実Readerが読み出して**取得元と照合済みの限定byte列**を渡す内部経路を検証。Path専用APIを正本に固定せず、外部Producer形式の第二Parser／巨大な公開raw-bytes Envelopeを増やさない。P2：#4307/#4503の用途別Fetch Plan／選択Scope／送出許可が揃ってから、NAS/S3 Provider、CLI/Studio/Harness独立Consumerを比較。REA側で新しいCAS/Host/Grant Engineを作らない。

**判定に必要な分離**：①Source Revision（外部解析対象のproducer-declared byte版）、②受領Evidence raw digest、③Store objectのlocator/版・実Readback時のdigest、④REA version/protocol/attempt、⑤選択されたObservation/Relationの外部Identityとsubset coverage、⑥TEI Sourceへの候補対応、⑦現在の用途・開示資格。Digest一致は内容同一の証拠であって、Source位置、業務Meaning、Current Grant、全Graph取得、Git HEAD/Dirty入力との一致を保証しない。Pin Intentの維持は開示や取得の許可ではなく、Catalog/Source改版やA→B→Aは再資格化する。

**受入候補（新製品試験はNOT RUN）**：`REA-STORE-01`同一Evidence byteをローカルPathとLocal CAS Readerから受領し、選択Observation/有向Relation・raw digest・Coverageが一致／Locatorは別、`02`同digest別Source Revision／同Evidence ID別raw digestは別Attempt、`03`CAS member欠損・破損／metadataのみ存在をcompleteにしない、`04`Remote Access拒否・offline・partial・tool-absentの区別、`05`Pin対象の現在性／用途別Disclosureなしでは外部AI送出0、`06`1+1の部分投影を全Graphまたは正式TEI Relation/CRUDへ自動受理0。実Store読取り、外部送信、実REA→正式TEI Consumerの縦断試験はこの追記では実施していない。REA実装PR #4322は2026-11-12確認時点で**ahead 10 / behind 254、41ファイル、mergeable=false**。元PRの一括統合を完了条件とせず、現mainからの小分割PRで独立に検収する。

<!-- rea-ref-coverage-20261112 -->
### REA由来参照とTEI参照索引の区別・高速移動（2026-11-12、#3394×#3788×#1124）

**参照の出典と資格を保持する。** REAが発見した静的Relation／同名・近接の構造候補、TEIで受理したData・Function・Businessのモデル参照、実DBのFK／Query観測、実行時の依存は同じ「参照」ではない。既存Source Index／ModelRef／trace／三要素分析法の横断参照を再利用し、表示や検索では`origin=external-observation/candidate/accepted/runtime-observed`（分類候補）、`subject-kind+stable-id`、外部Edge IDと向き・多重性、Source span/revision、確認済み範囲・未解決・必要な開示範囲を区別する。REAのEdge IDや描画用Edge IDを正式TEI Relation IDへ暗黙変換しない。新しい意味正本・Graph Engine・LSPを作らない。

**件数と「未使用」は閉じた範囲でのみ確定する。** 参照数は`confirmed / candidate / unresolved`と検索範囲・Snapshot/Revision・Coverageを併記する。外部REAのpartial、権限不足、未取得Source、動的参照、未観測Pluginが残るときは「確認済み参照0件（未確認あり）」と表示し、「未使用確定」や安全な削除の根拠にしない。Candidateを正式参照数へ足し込まず、欠落をゼロに変換しない。AI向けCUIとStudioでは同じ型付き検索結果・診断・固定Selectionを扱い、引用元・参照先・参照元・SHO上の出典へキーボード主体で移動し、戻る／再移動の履歴を保持する。Pinは現在性・用途別Disclosure・改版で再資格化し、古い結果を最新の画面に上書きしない。

**修正済みの基底と、これから検収する製品経路を分ける。** [#4513](https://github.com/hat47x/tei-platform/pull/4513)（CRUDでのData索引）、[#4534](https://github.com/hat47x/tei-platform/pull/4534)（矛盾の全件報告と構造OK／要レビュー区別）、[#4551](https://github.com/hat47x/tei-platform/pull/4551)（明示CRUD-only組の表示と`UnusedData`）、[#4524](https://github.com/hat47x/tei-platform/pull/4524)（同Kind同ID衝突）、[#4507](https://github.com/hat47x/tei-platform/pull/4507)（Fingerprint入力非破壊）、[#4514](https://github.com/hat47x/tei-platform/pull/4514)（Sourceだけ改版した図面のstale扱い）は2026-11-13確認時点で**main統合済み**。いずれも実装した経路・固定HEADでの限定検証であり、三要素分析法の**全Consumer横断で参照数が正しいこと**や、CRUDのレビュー要否をStudio／SHO／AIが適用停止へつなぐことは未実証。特に`ok=true`は構造診断または投影成功の範囲に限り、`crudConflictStatus=conflicts-require-review`を意味受理・変更許可へ読み替えない。#3788を通常Editorの定義移動・逆引き・安全なrenameのOwner、#1124をModeler投影、#3394を外部REA由来の出典分類、#4503をEvidence Pin/DisclosureのOwnerとする。Source/Indexの増分失効とPluginの遅延読み込みを基本に、参照移動のp50/p95・Index更新範囲・起動時Plugin負荷を実測する。REA入場、#2586の実Function Read、#4515のSubject Pinを参照UI完成でブロックしない。

**独立Expected（基底の一部は修正・単体/CLIで試験済み。REA→参照一覧→Studio/SHO/AIの横断製品試験はNOT RUN）**：

| Case | 反例・操作 | 期待結果 |
| --- | --- | --- |
| REA-REF-01 | 外部REAにRelationだけ存在し、TEI Meaning未受理 | 外部候補として辿れるが、正式Modeler Relation／CRUDの確定参照数に混入しない |
| REA-REF-02 | 同名異ID、同端点の有向平行Edge、逆向きEdge | ID・向き・多重性・出典を保ち、単一Edgeへ縮約せず対象間違いをしない |
| REA-REF-03 | 調査範囲にpartial／permission-denied／動的参照／未観測Pluginがある | confirmed=0でも「未使用確定」を出さず、未解決／範囲不足を併記する |
| REA-REF-04 | 元Source A→B→A、改版、旧応答後着、旧Pinから参照へ移動 | 現在性を再確認し、旧結果のsilent retarget／最新表示への上書きを拒否 |
| REA-REF-05 | 明示CRUDのみ、Data/Business同一ID、同Kind同ID重複、Fingerprint計算前後 | #4472/#4475の負例で件数と名前の取り違えを拒否。Fingerprint計算は元モデル不変 |
| REA-REF-06 | CLI/CUIとStudioから同じ参照元一覧・参照数・依存変更Previewを取得 | 同一Scope/Revision/Grantで結果・unknown/diagnosticが一致。ショートカット操作は追加権限を持たない |
| REA-REF-07 | 大量参照・Plugin未ロード・1 Sourceのみ変更 | 変更影響のある範囲だけ失効・再索引し、不要Pluginの起動を要求しない。実測p50/p95と全件照合の一致を記録 |

**検収順**：P0はmain統合済み#4472/#4475の回帰を固定基線として、#3788の既存Source Indexから一Subjectの参照元・参照先一覧（Scope/unknown付き）を**通常CLIで実受領**し、列挙した個々のRelationと対応する元Source位置を突合する。#4548の型付き`ExternalCandidates`はmain受入の可否を独立に判定し、候補1 Observation＋有向Relation 1件のRead成功を参照索引への正式登録とみなさない。P1は#4515のSubject Pinを利用し、同じQualified Subject・出典・RevisionでStudioの定義移動／逆引き／戻るとCLIの結果を照合する。P2は外部候補と正式参照を**別の参照集計列**で並列表示し、SHOの引用・編集Preview・モデル統計・AIの用途別開示へ接続する。未受理外部Graphの部分取得を「全体把握・全件未使用確定」と呼ばない。

<!-- rea-question-bound-reference-20261113 -->
### 外部REAの参照・未使用判定を「問いに必要な証拠」で検収する（2026-11-13、#3394×#3788×#4472）

**存在の証拠と不存在の証拠は非対称。** REAの同一実Attemptで捕捉したExternal Relation 1件は、「指定Source/版/解析範囲に、指定向き・Edge IDの候補が観測された」という**存在の限定Witness**になる。一方「関連がない」「未使用」「直接Writerがいない」「移行後も同等」は、該当Subject・参照Kind・Source/Dependency・未ロードPlugin・動的到達・必要な業務意味・現在用途について、**その問いに必要な集合の取得と未解決数**が確認できる場合に限って判定する。検索結果の空配列、confirmed=0、Validation.OK、Delivery success、単独digest一致を全称／不存在／完全性の証拠にしない。必要集合を列挙できないときはunknown/not-examinedを明示し、判断不能を正常なゼロに変えない。これは#4527の問い相対の閉包と#4487の三要素横断Evidenceの議論を、REAの参照画面へ適用する提案であり、新たなVerifierやEvidence正本を作らない。

**最近mainへ入った修正と残る不足を区別する。** #4507（Fingerprint純粋性）、#4524（同Kind同ID曖昧性拒否）、#4513（CRUD Data索引誤用）、#4534（CRUD衝突全件報告と構造的OK／要レビューの別表示）は限定修正として統合済み。したがって前節REA-REF-05の全項目を一律「未修正」としない。ただし**明示的なCRUDDefinitionだけが存在しFunction Operationsが空のとき、派生CRUD表・横断参照から行が脱落する問題**（#4472/#4403）と、Derived Consumer/Studioの採用・変更影響判定は残る。Function-level CRUD、業務Action、SubsystemのDirect Writer候補、実行時のEffect/Rowsetをひとつの確定Writer件数に縮約しない（#3250/#4520）。okScope=structural-diagnostics-onlyも業務整合性・Owner受理を意味しない。

**受入候補（独立Expected、実Product試験はNOT RUN）**：

| Case | 入力・反証 | 必須の観測結果 |
| --- | --- | --- |
| REA-QC-01 | REAで1件の外部有向Edgeを観測、TEI正式Relationは未受理 | 指定Attempt・Source版における候補の存在のみ成立。正式参照・CRUD・Current Grantへの昇格0 |
| REA-QC-02 | 一件も参照が見つからないがSource範囲がpartial、Plugin未観測、動的参照が残る | confirmed=0と未探索集合・unknownを別記し「未使用確定」/安全削除を拒否 |
| REA-QC-03 | 明示CRUD UのみでFunction Operations空、または矛盾CRUD U/D複数 | 派生表0行・単一行を「直接更新なし」「矛盾なし」と扱わず、脱落/要レビューを独立表示 |
| REA-QC-04 | 同じEvidence IDとraw digestだがSource現在版／Workspace／Disclosureが変化 | #4548の受領Receipt Pin一致だけでは現在のSource/閲覧・配信・編集資格PASSにしない。必要な再解決で失効またはunknown |

**実装順**：P0は#4547のmain統合済みReceiverと#4548の型付きOperationの限定受入を優先し、実際のSource/Reader/Consumer受領に対するclaim上限を明記。P1は#4472/#4403の明示CRUD-only脱落を修復・判別し、#3788で一Subjectの参照/被参照を**閉包の範囲とセット**で返す最小read-only CLI。P2は#4515のStudio Subject Pinと独立Consumerを結び、REA候補と正式参照を並列表示して高速移動・参照数・変更影響を検収する。未受理Relation Edgeの正式Pin、外部Harness送出、Source書戻し、実DB Apply、SHO著者文変更、Owner意味受理は別の資格。**TEI Studio Desktop実Wails起動・セルフビルド、および#2586の実Function Readをこれら全体の完成でブロックしない。**

**検討中の横断追補（DBアクセス・Query/2way SQL）**：対象DataSourceの既存Binding解決、DB別Native SQLの選択規約、SQL編集時の診断と分岐検証、異種DBでの意味同値性を、[DBアクセス・Query選択・2way SQLの最小適合計画](https://github.com/hat47x/tei-platform/issues/2622#issuecomment-6772875802)で小さな実証へ分解する。#2611・#2622・#2387・#2667に既存の責務を残し、新しいCore API・正式DSL構文・実DB対応の採用・完了を意味しない。
**2026-11-09／Product Queryの物理入力乖離を解消（独立Source Fixture段階）**：[Workbench設計§22](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)と[隔離PG Product Fixture](../examples/data-source-workbench/README.md)に、既存`postgres_seed.sql`の`sales_order`4列と、実`order_entry.recent_orders` SQLが要求する`public.orders`5列＋`:from`の**物理Schema不一致**を反映した。従来F-01〜10を壊さず、専用`recent_orders_product_seed.sql`を別Sourceとして追加し、3行／境界一致2行／後方0行の独立Source-static ExpectedをGoで判別する。**これは実PostgreSQL接続・Provider Invoke成功ではない**。最初の未成立実行辺は、固定Function/Queryの実Consumed SQL+args→Alias/物理Instance/Provider→実Rowset/Function Data Context。Ownerは#2586、実ProductのCLI/Studioは#1251、Schema適用は#2892。main #3540の静的Inspector S2-referenceは実CLI/Wails経路まで前進したが、外部Plugin Build/Grant/2×2実BOMを証明せず、異質JSONLは後続比較とする。新Gate/Host/Query正本なし。


### XTEA browse実Readのmain欠落を解消する（2026-11-12、#2586×#3270）

[Workbench設計§24](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)で、未merge終了の[#4076](https://github.com/hat47x/tei-platform/pull/4076)が「mainへ既存取込済み」と判断されたことと、現mainの**実際のSource存在**を分離した。実際には旧PRにある`internal/driver/query_execution_environment.go`、`internal/driver/browse_parameters.go`、`internal/functioncontext/functioncontext.go`、`cmd/tei/cli_view_browse_read.go`の経路がmainで見当たらない。旧Branchの隔離PG 30行・eq/ne・2ページ・通常CLIの**旧版限定実測**を、main製品のPASSへ流用しない。さらに旧Branchの`FROM sales.order`は**現mainの修正済み`FROM sales."order"`を退行**させるため、24ファイル一括上書きではなく必要辺を現在の既存型/APIへ小分けに戻す。

**最初のP0実装**：一方のWorkspace（`reference/workspace`の`:from/public.orders` 3/2/0、または`examples/xtea-lowcode`の`status/sales."order"` 30行と2ページ）を指定し、Environment/Instance→明示Query Input→実Executor/Read-only PostgreSQL→Rowset→Function Data Context→通常CLI出力・独立Expectedを同一Attemptで検収する。各WorkspaceのSQL/Fixture/Grain/Roleは別。0行/最終Pageの`declared-page-contract`をグローバルな完全性・不存在へ昇格させない。正式PluginのBuild/Entry/Grant、Studio/Harnessの**固定対象選択と誤解除防止**、NAS/S3等の外部Evidence格納・Source Revision、Schema DDL History/Applyは**別Consumer/資格**として並列に進める。#2586が実Read、#1251/#951が正式Plugin、#1124/#1824/#1158がContext選択、#2600/#1163がEvidence、#2892/#2887がDDL、#2514が独立Expectedを担当。既存WB-01〜54・HJ/QXとRQ-035 PASを再利用し、新Registry/Host/Runtime/Query正本/Gateを導入しない。

**現状**：物理FK/Index read-only投影とProduct Queryの静的Seedはmainに存在。XTEA実Read経路の旧PR成果は**mainで未確認／上記Path不在**。今回の記載だけでは新Go実装・実PG/CLI/Studio・Owner受理は成立しない。


## 作業領域E：設計の実行と検証

### 目的

画面を実装する前に設計内容を実行し、アプリケーションの意味を検証できる環境を維持します。

### 重点

- 一覧表示・記録・参照・見出しと明細・操作の構成パターン
- 検証
- 認証と認可
- 機能の呼出し
- 業務上の結果と実行時エラー
- 独立した状態の観測
- 再現可能で決定論的なシナリオ

設計用の実行・検証環境は、本番の実行基盤を単に簡略化したものではありません。設計の妥当性を検証するための基準です。

## 作業領域F：実行先に依存しない表示

### 目的

同じ正本上のアプリケーションを複数の実行先へ展開し、共通する意味と実行先固有の違いを検証根拠で区別します。

### 最初の実行先

最初のウェブ実行先は、PAS-04で初めて実証した対象です。

判定段階Cの検証では、引継ぎ基準版`R`とウェブ実行先の構築版`T1`を区別して追跡します。

### 二つ目の実行先

判定段階Cの後、`ISSUE-000026`で二つ目の実行先を選定します。

比較する観点は次のとおりです。

- アプリケーションの正本上の意味
- 実行先の能力
- 実行時の対応付け
- 実行条件と信頼の境界
- 実行先固有の実装
- 実行環境の管理機構
- 状態の観測
- 意味・機能・表示の検証根拠
- RQ-024に関する性能測定

最初の実行先だけの検証結果から、実行先に依存しない分類体系が完成したとは判断しません。

### 編集環境の実行基盤とアプリケーションの実行先を分ける

編集環境が開発者向けのデスクトップ基盤としてWailsを使っても、それを二つ目のアプリケーション実行先の検証根拠には数えません。

Wailsを使って生成したデスクトップアプリケーションを採用する場合は、それを独立した実行先とし、意味・機能・表示の検証根拠を取得します。

## 作業領域G：編集環境と開発体験

### 目的

厳格な意味モデルを、開発者が多くの情報を安全に扱える編集環境へつなぎます。

### 原則

編集環境そのものを、意味の正本を管理する主体とはしません。

```text
Studio / file-native / CLI / AI
        ↓ candidate
Shared Acceptance Boundary
        ↓
authoritative source
        ↓ reload
Canonical meaning
```

判定段階Cの後の最初の実証は、`ISSUE-000046`で行います。

最小限の操作手順は次のとおりです。

```text
Select
  ↓
Propose
  ↓
Preview Semantic Diff / impact
  ↓
Accept / Reject
  ↓ Accept only
Shared Acceptance Boundary
  ↓
reload / Conformance
```

人が記述する安定識別子は必須とせず、編集環境専用の受理機構も導入しません。

### 2026-11-07 Plugin-first D1：実CLI／Studioの最初の不足辺を固定する（実装・受理前）

[main統合済みPR #3277](https://github.com/hat47x/tei-platform/pull/3277)の[Plugin-first移行計画](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) D0〜D6と、[#1016の構成→実使用Witness](https://github.com/hat47x/tei-platform/issues/1016#issuecomment-6704666016)、[#1251の実Go D1引継ぎ](https://github.com/hat47x/tei-platform/issues/1251#issuecomment-6704672969)を、**作業領域Gの一つの実装切り口**として接続する。main統合[#3300](https://github.com/hat47x/tei-platform/pull/3300)のD1 Fit/Gap（AS-01〜12）と[#3367](https://github.com/hat47x/tei-platform/pull/3367)の追加条件（AS-13〜20）を検収正本とし、[Desktop Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)および[mainのPlugin-first設計§16](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)（旧Draft #3369は閉鎖・未統合）と重複する仕様を増やさない。既存#1119のG0〜G4／B/C判定を変更しない。

**コード上の出発点（2026-11-07、GitHub mainの静的照合）**：
- `internal/operations/inspect.go` の `InspectSubject(root,subject)` はWorld/Source Indexのread-only投影で、`worldFingerprint`、`sourceIndexFingerprint`、`sourceStatus`を返す。ただし**Required Domain ClosureがすべてLoaded/Parsed/Boundされた証明**でも、expected Revisionでの実行前CASでもない。
- `cmd/tei/cli_commands.go` の `inspect`は手書きの受付一覧にあり、`cmd/tei-studio/main.go` の `StudioBridge.InspectSubject`は`internal/studiohost.ReadOnly`へ委譲する。現時点で**両方が一つのexact選択ProviderをComposition経由で実呼出しした**とは観測されていない。
- `internal/composition`の`Contract{ID,Version,Shape}`／`Provider{ID,Scope,Offers}`／`Selection{Revision,Epoch,...}`は宣言上の候補・選択に使えるが、現Host Grant、実Builder Source閉包、実Provider Callbackの資格そのものではない。
- [#3247](https://github.com/hat47x/tei-platform/pull/3247)はWorkspace名・manifest照合情報の限定CLI応答をmainへ統合済み、[#3284](https://github.com/hat47x/tei-platform/pull/3284)／[#3296](https://github.com/hat47x/tei-platform/pull/3296)／[#3354](https://github.com/hat47x/tei-platform/pull/3354)はDesign Contextの限定試験を段階的にmainへ統合済み。ただしWorkspace名は一意のRoot/Source identityではなく、Contextの実11 Replyと32判別ControlはRequired全域の同一原子的Snapshot／全業務Meaning／Plugin実Invokeを証明しない。

**P0の対象を一つに固定**：D1の候補をまず**既存`inspect`**とする（`model catalog`は比較対象であって同時に置換しない）。変更内容は次の不足一辺だけを解くこととし、汎用Plugin Manager／新Registry／新CLI構文／Studio用意味Service／第二Scenarioを先に作らない。

| 順 | 実変更・観測点 | 独立した合否／停止線 |
| --- | --- | --- |
| **P0-a: Source/Context基線** | 1 Workspace、1 Subject(kind＋qualified ID)、1 Source版。CLI実コマンドとStudio allowlist内のWorkspace ID→信頼済みRootを同じ対象へ束縛する | `worldFingerprint`値一致や`workspace.name`一致だけで同じWorkspace・同じSource集合・現在Grantとみなさない。Root実体/Manifest/Source occurrenceを保護された内部Evidenceで照合し、開示先には#2516で許可された情報だけ投影。Required Closure不足は#3262へ返す |
| **P0-b: exact ContractとProviderの限定導入** | 既存`InspectSubject`を一つのread-only Typed Service Adapterとして提供し、#951/#1016の既存解決・選択を経由させる | Providerなし／二候補曖昧／Contract shape・scope不一致／旧epochを拒否。Referenceの`Registry.Select`だけを実Call・Conformance・現在Authorityに昇格しない |
| **P0-c: 2つの実Consumer** | CLIの`inspect`とStudio `StudioBridge → studiohost`から、選択Providerの**実Invocation**へ到達。read-only旧経路とのshadow比較後、切替対象に暗黙fallbackを残さない | CLIがJSON化した結果をStudioが表示しただけでは「独立2 Consumer」にならない。実CLI Target／Studio Targetそれぞれのentry、Selection、Provider実Call、Input/Resultを別Attemptで照合。Wails Desktop E2Eは実起動を伴う別資格 |
| **P0-d: 不整合と実Build** | B selected→B invokedの正例、B selected→A invokedの負例、未選択AのBundle同梱と`init`混入、Root/Source/epoch変更、途中切断を既存テストへ接続 | 値が等しくても誤Providerなら不合格。Build成功・実Source members観測・Binary含有・Callback実行を別Claimとする。#3217の外部Module member観測（qualified=false）をG4のSource Closureへ転用しない |
| **P1: 同じConsumerを用いた展開** | #3262設計Context／#3270 Data Workbench／#3267 Verified BOM／#1124 Graphと三視点を**後から**同じOperation Contractへ適用 | D1 `inspect`のread-only PASSをDDL、外部Host、実DB Read、BOM組合せ、Owner意味受理へ拡大しない |

**特に混同しない2つの「2×2」**：D1でいうCLIとStudioは**二つのSurface/実Consumer経路**。#3267のBOM-1が検証する「Provider二版×Consumer二版」は**同じSurface二種類の試験ではない**。BOMの4 exact tupleには版・Target・Artifact・Independent Expectedとactual Invocationを独立に記録し、CLI/Studioが同じ結果を示しただけでは4組合せをVerifiedにしない。未実施／SKIP／途中切断は不合格・未知として残す。

**最初の実行成果物**：#1251／#1158／#1824で`inspect`への実Provider経由の入口を最小変更し、#1180の既存Scenario＋#2514の独立Expectedで上表P0-a〜dの正負を検査する。#3262はRequired/Loaded/Parsed/Bound/ReturnedとTool完全受領の照合、#951/#1016はexact契約・選択、#943/#1090/#2831は物理Build/Source資格、#2516/#2515は現在の公開／送出資格をそれぞれ所有する。**一つの列が欠けても他列の成功で補完しない。** 固定命令の削除やCore分割・Trait新Schema採用は、ここで実Consumerの不足と変更量を測ってから判断する。

**現時点の資格上限**：上記は実コード・既存PRを読んだP0実装順序の統合。新しいCLI/Studio共通Provider、実D1 Target、CI、実Wails、Source Closureの正式資格、Host Grant、業務Owner意味受理は**NOT RUN／NOT PROVEN**。自動的なmain移行・正式採用を示すものではない。

#### 2026-11-08 横断更新：Capability選択の実証と実Consumer資格の間を分ける

既存の実Referenceから観測できた前進を受入基線へ採用するが、**正本の検証規則・実Consumerへの接続は未完成**として区別する。

| 成熟した実証（main／別HEADの適用範囲） | これを根拠にしてよいこと | 追加で必要な反証 |
| --- | --- | --- |
| [#3337](https://github.com/hat47x/tei-platform/pull/3337)→[#3350](https://github.com/hat47x/tei-platform/pull/3350)：Capability Referenceで宣言12／未選択6→4（選択ルール改訂に伴う比較値）。DB instanceの技術名と能力IDを等号で比べず既存`SupportsDatabaseProvider`を使用 | **判定規則の変更で同じSourceでも結果が変わる**。選択されない能力は直ちに欠陥ではない。Catalog/判定器の版をEvidenceに明記 | 宣言Plugin → Capability Kind/ID → Environment binding → exact Contract/Provider Selection を同一視しない。[#3363](https://github.com/hat47x/tei-platform/pull/3363)はmain統合済みの限定参照であり、24 Binding中22 selected/2 missingと報告した。Workbench feature 2件は現Product Validatorと異なる探索範囲であり「製品Error 2件」ではない |
| [#3354](https://github.com/hat47x/tei-platform/pull/3354)：実CLI 11 read-only Reply／Consumer 32局所判別と限定独立3 Rule／Canonical／Source Index | 既存Context Gateが、正常JSON・同FPでも必須Rule/Relation欠落を有限範囲で拒否できる。CI上の限定合格を次のfixture設計へ使える | 11のCLI起動は**単一Snapshotの原子的取得ではない**。Required Universe、Loaded/Parsed/Bound、Source/Grantの時点差、受信Sink、外部Harness Tool Reply完全性は#3262/#2509で別途検証。qualifiedTestGeneration=falseを保持 |
| [#3333](https://github.com/hat47x/tei-platform/pull/3333)：`tei commands view`に`writeFlags`、実File/SQLite作成の正負実測 | command名の`read-only`広告と、引数Variantによる**ローカル出力作用**を区別できる | D1 `inspect`をread-onlyで開始しても、将来のTool/Plugin/CLI wrapperにおける出力・Session Log・外部Egressは別Effect/Disclosure。単なる`effect:read-only`を「書込み・送出ゼロ」に拡大しない |
| [#3336](https://github.com/hat47x/tei-platform/pull/3336)：子Build driverのinvoker/invoked実行ファイルdigestと子成果物再Read一致 | Build駆動主体と子実行物を分離して観測する実Reference。Stage2成果物の子報告だけを信用しない | これはPluginがLink/InvokeされたこともL3 TEI self-build成立も証明しない（l3Claimed=false）。選択Pluginの実Build closure・init・entrypointは#1251/#2831で別試験 |
| [#3327](https://github.com/hat47x/tei-platform/pull/3327)／[Desktop計画Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)：Workbench/frontendの実UI受入 | Studioに`/`のPoC Aと`/workbench/`の別Frontend/RPC Consumerがあること、および別HEADでNode20/20等の局所試験を確認できる | 別HEADではWails埋込asset byte不一致が残る。D1で**実際に選んだFrontend ArtifactとWails起動時のRPC/Provider**を照合し、BridgeのGo単体試験だけで実Desktop二Consumer合格にしない |

**D1実配線の最小順序（main統合済み#3300/#3367のD1a→D1bを維持）**：①現`tei inspect`と`StudioBridge.InspectSubject`の入力RootをHost側で確定→②Required/Loaded/Parsedの用途相対不足を表示→③#951のexact Contract/Provider Scope選択を通す→④Host-owned Adapterから実Providerへ到達→⑤それぞれの実Target/Artifact/AttemptのInvocationを確認→⑥旧直接経路を切替対象から外して再試験、の順にする。[#3300](https://github.com/hat47x/tei-platform/pull/3300)で確認済みの旧Service Carrierが`application-runtime`以外を拒否する仕様を**文字列置換で迂回しない**。Authoring Host Adapterは既存`studiohost.RootServices/ServiceFactory`の信頼境界を利用し、Renderer/AIからTrusted RootやSelection Authorityを受け取らない。将来のMCP/生成Runtimeは同一Contractで別Consumer/Grant/Target資格を得る。

**追加の独立検収義務（既存Scenario/Expectedへの観点追加であり第二SSOTではない）**：
- **PGD-01：宣言とEnvironment選択**。一つのPluginが複数Capabilityを提供、ある一Kindのみが選択される／未選択でも正常、技術名だけのDB選択一致を拒否する。実Providerに到達したとは報告しない。
- **PGD-02：曖昧・Validator非対称**。同Bindingの適格Provider二件は`ambiguous`、0件は`missing`を候補として記録する。ただし参照集計の`missing`を既存Validatorの正式拒否に昇格させず、対象・種類・規則版を照合する。
- **PGD-03：選択と実呼出し**。Provider Bをexact選択→CLI・Studio双方でB実Callの正例。B選択・A実Callは結果一致でもFAIL。Registry選択成功、同一メソッド共用、Bundle同梱のみならunknown。
- **PGD-04：Source/受領と版**。Workspace名／WorldFP一致でもRoot・Required Relation欠落、異なるIndexモード、11返信の間のSource ABA、Tool Reply途中切断があればContextをcompleteにしない。公開先のSecret・件数は#2516に従う。
- **PGD-05：実Desktop**。CLI実Targetと`/workbench/`実Frontendの各call-site、Wails embedしたassetのbyte、UI ABA、RPCの実Provider Invocationを独立照合。Go Bridge単体や別HEADのNode Testは製品E2Eの代替ではない。
- **PGD-06：副作用とBuildの別資格**。read-only入力/argvと出力Sink、未選択PluginのGo import/init、子Build実行物と生成Artifactを分離。Plugin Build/Runtime資格、Verified BOM 2×2（#3267、[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)の旧RQ-036案（現行ではRQ-037へ再採番必須））、Host GrantとOwner Meaningはそれぞれ別欄のまま。
- **PGD-06：副作用とBuildの別資格**。read-only入力/argvと出力Sink、未選択PluginのGo import/init、子Build実行物と生成Artifactを分離。Plugin Build/Runtime資格、Verified BOM 2×2（#3267、[旧Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)のBOM採番旧RQ-036（現行RQ-037）候補）、Host GrantとOwner Meaningはそれぞれ別欄のまま。

**Owner／完了線**：#1251/#1158/#1824がD1コードと2実Consumer、#951/#1016は正式Selection、#3262はContext必要集合とReply受領、#2514/#1180はExpected/Scenario、#2515/#2516はSink・Disclosure、#943/#2831はBuild、#3267はBOMを担当する。[main統合#3368](https://github.com/hat47x/tei-platform/pull/3368)はPlugin選択≠Capability選択≠一意Provider解決≠Required Trait履行として3つのReference負例を追加したが、Go実行・Trait履行の完了は未証明。#3278/#3279はRequired/Default/manual/hintと合成の義務のOwnerとし、Go固定Kindや新Trait Runnerを作らない。Graph/三視点/Function表現型（#1124/#2497/#2507）は同じ出典Factの別投影であって、それを実Selection/実Effectの証拠にしない。


**今回の計画から外へ委譲した詳細**：AS-01〜20の逐次・Provider／Scope負例は[main #3300/#3367のEvidence](evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)に帰属し、本節のPGD-01〜06はそれらを独立に再定義するテスト群でなく**横断の完了票**とする。原子的Traitの組合せ詳細とTC-16〜20は[main #3368](https://github.com/hat47x/tei-platform/pull/3368)、Provider×Consumer版Matrixは[#3267](https://github.com/hat47x/tei-platform/issues/3267)／[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)、CLI/StudioのOfferとFunction内表現Bindingは[main統合#3338](https://github.com/hat47x/tei-platform/pull/3338)、Studio/Wailsの実Frontend資格は[Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)が保有する。複数Issue/PRの件数やPASSを一つのContract/Functionの合格へ足し合わせない。

**P0開始時の不変条件**：既存`internal/plugin/runtime/service_carrier.go`は`application-runtime`以外のScopeを拒否する。Tooling/Authoring Inspectorの実装は当該制約を緩めず、既存`composition` SelectionとHost-owned`studiohost` Seamの不足辺だけを修正する。画面がSourceを読めたことと、Function Data ContextやOwner Accepted Meaningを満たすことは異なる。

**当時の計画変更の上限**：PGD-01〜06を同一HEAD・同一AttemptのCLI/Studio/Frontend/Hostで検証したわけではない。なお後続[main #3440](https://github.com/hat47x/tei-platform/pull/3440)で**cmd/tei自身の限定3段Self-build L3実測**は成立したが、Plugin Bの実Link/Call、Build Source全閉包、現在のGrant、2×2 BOM、Owner業務受理への資格移転はない。実B呼出のProduct二Consumerは引き続き **NOT PROVEN**。

#### 2026-11-08 D1の実装差分：局所Provider Callから実二Consumerへ（最新mainに基づく引継ぎ）

後続で[ADR-0048（main #3439）](adr/0048-shared-application-api-and-transport-adapters.md)が採用され、CLI/MCP/API/Web/Desktopは**共通Application Operationの意味**とTransport表現を分ける方針になった。[main #3458](https://github.com/hat47x/tei-platform/pull/3458)は`validate`のCLI/MCP間での限定投影同値に成功したが、`inspect`のPlugin経由実行・二Consumerの資格を代替しない。

| 段階と責務 | 現時点の実装候補・確認対象 | 合格としない状態／次の不足辺 |
| --- | --- | --- |
| **S0 Typed Port**（#1158/#1824） | [CLI Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)の`inspectWorkspaceJSONWithService`、[Studio main統合 #3371](https://github.com/hat47x/tei-platform/pull/3371)の`RootServices.Inspection`。指定Service欠落・出力契約不一致・未許可Root時はfail-closedを検査 | **Studio Hostの型付きService seamのみmain統合済み**（squash `080d162c`、Health #42470378877 success）。CLI側はDraftで、通常CLI `main.go`はBuiltin経路。Service注入だけでは実Plugin選択の証拠にならない |
| **S1 selected Provider（局所）**（#951/#1016/#1251） | CLI Draft #3381の`internal/inspectoradapter`が既存Registry/Selection/SelectedCoordinator/Host guardを使ってB選択→B実Go Call／B選択→A登録拒否／epoch/Grant失効を**局所in-process**で検査する候補 | 合成ID/digest・登録Functionだけでは配布Binaryの実Member/Entry/Conformanceを証明しない。正常なReturnが得られてもTarget実Entry・Authorityは別 |
| **S2 CLI/Studio実Consumer**（#1251/#1824） | HostだけがTrust Rootとexact Selectionを保持し、**通常起動のCLI dispatchと実Wails Studio Bridge**を同じ契約・各自のTargetへ接続する。[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)（squash `0f3455e132d2146834d65089c68cae40c31f3055`）の従来Direct経路を6 Subject＋Not Foundで比較するGo Testは、exact-head Healthが成功した**移行前Baseline**としてのみ利用 | UIがCLI JSONを表示しただけ／factory nilでBuiltinへフォールバック／別revisionから得た同値JSONを二Consumer成功としない。B selected→A Callは同値ResultでもFAIL |
| **S3 Source・Artifact・Effect実証**（#3262/#2600/#3267） | [main #3482](https://github.com/hat47x/tei-platform/pull/3482)の限定Function Required（13実CLI Reply/42局所Controls）と[main #3477](https://github.com/hat47x/tei-platform/pull/3477)のSource-only失効を、実S2 Attemptで得たSource/Provider/Artifact/Scope/独立Expectedへ照合 | 13 Replyは原子的SnapshotでなくFunction実行資格もない。[main #3479](https://github.com/hat47x/tei-platform/pull/3479)の外部Provider raw/Manifestのみ改訂、[main #3490](https://github.com/hat47x/tei-platform/pull/3490)の選択規則fingerprint変更は追加の失効反例であり、Consumer World FPの一致だけでfreshにしない |

**2026-11-08 D1 CI検収の前方移植（現在のS0〜S3と分離）**。[main統合済み #3499](https://github.com/hat47x/tei-platform/pull/3499)のS0〜S3／D1B-ERR・TRF・CALLを変更せず、並行PRの**各checkoutと実Actions Job Log**から「Go Test PASS・Health FAIL・統合済みReference」を別々に記録する。

| 対象 | 試験済みの具体的なcheckoutと合否 | 現在の判断／次の一手 |
| --- | --- | --- |
| [Studio #3371](https://github.com/hat47x/tei-platform/pull/3371) | 旧HEAD `c7acfd6b`の[Health #42464668701](https://github.com/hat47x/tei-platform/actions/runs/42464668701)：全Go Test／実Workspace validate **PASS**、`gofmt` 2ファイルで全Health **FAIL** | 書式修正HEAD `86f9261e`の[再CI #42470378877](https://github.com/hat47x/tei-platform/actions/runs/42470378877)が**success**、squash `080d162c`で**main統合**。S0 Typed Service／Error秘匿のGo検査であり、実Selected BのStudio Bridge呼出しはS2の別条件 |
| [CLI #3381](https://github.com/hat47x/tei-platform/pull/3381) | HEAD `5cdd3b9e`の[Health #42463722188](https://github.com/hat47x/tei-platform/actions/runs/42463722188)：全Go Test／validate **PASS**、`gofmt` 2テストファイルでHealth **FAIL** | 同PR内でGo1.26 `gofmt`修正後にexact HEADを再実行。S1の局所B Go関数実CallはGo Test PASS、通常`cmd/tei/main.go`でBを選ぶS2は未接続 |
| [Imported Source #3479](https://github.com/hat47x/tei-platform/pull/3479) | 旧HEAD `fff65f0a`の[Health #42464022323](https://github.com/hat47x/tei-platform/actions/runs/42464022323)：Source／Manifest各改訂を含む全Go Test／validate **PASS**、`gofmt` 1テストファイルでHealth **FAIL** | 書式修正HEAD `6f0a9735`の[再CI #42470247004](https://github.com/hat47x/tei-platform/actions/runs/42470247004)が**success**、squash `99635b87`で**main統合**。独立Source×Manifest 2×2は外部Importの同一Atomic Load閉包／実Provider Readを証明しない |
| [BOM政策 #3490](https://github.com/hat47x/tei-platform/pull/3490) | HEAD `6b2e0b18`の[Health #42464016653](https://github.com/hat47x/tei-platform/actions/runs/42464016653) **success**、PRはmain統合済み | 選択規則RuleFingerprintと旧Evidence再利用の限定Go Reference。BOM Provider×Consumer版4組合せ／同一Attemptの実Build・Invoke・現在Grantではない |

**判断と実装順**：上記CI状態の変化を`S1 Go Test PASS`、`Repository Health PASS`、`S2製品実Call PASS`、`S3実Source/Artifact/BOM/Grant qualified`へ単一の合否で昇格させない。Studio #3371と外部Source #3479は修正後exact-head Healthが成功・main統合済み。CLI #3381は新HEAD `602ddc30`のexact-head CI／main追随・衝突判定を残す。次に既存#1251/#1824の**通常CLI Dispatch→実Studio Bridgeにおける同一authoring Provider Bの実Call**を独立Expectedで検収する。部分Required・Imported Source／Manifest・Policy RuleはS3の用途別再資格化へ委譲。新Host／Resolver／Runner／Trait固定Enum／第二Scenarioを足さない。本追記はdocs-onlyで製品E2E、実PostgreSQL、実外部Harness、Owner意味受理はNOT RUN。
**次の一件をS2の実装と検収に固定**：同じ公開`reference/workspace`・同じ`semantic-data:sales.order`・exact`authoring-tool` Contractを用い、(a)CLI通常dispatch、(b)Studio Hostの実Bridgeから、それぞれ実選択BのGo関数へ入ったことを**別のCall-site記録**で確認する。BをAへ取り違える／Serviceなし／Source ABA／Host Guard失効／部分応答の各負例では、旧Builtinへ自動fallbackせず診断を返す。公開されるCall証拠は受信先の権限と#2516のDisclosureに従い、内部Root/Secretを誤開示しない。実バイナリとWebViewの適格性はS3で別に測る。

**予定の帰属**：S0はStudio main #3371／CLI main #3381、S1はmain #3381のComposition選択B→Host登録Bへの局所実Go呼出とCLI/Studio Host API二経路試験（製品S2ではない）、S2のProduct統合は#1251/#1824、S3のDesign Requiredは#3262、履歴Sourceと再利用判定は#2600、版Matrix/BOMは#3267、Trait実装義務はmain #3480/#3278、Scenario/独立Expectedは#1180/#2514。新しいResolver/Registry/Host/Scenario SSOT、HTTP常駐必須化、Legacy `application-runtime` CarrierのScope緩和は行わない。本節自体は計画の更新で、S2/S3の実試験成功・Owner Meaning受理を**主張しない**。
**予定の帰属**：S0はStudio main #3371／CLI Draft #3381、S1はCLI Draft #3381の局所実Go試験、S2のProduct統合は#1251/#1824、S3のDesign Requiredは#3262、履歴Sourceと再利用判定は#2600、版Matrix/BOMは#3267、Trait実装義務はmain #3480/#3278、Scenario/独立Expectedは#1180/#2514。新しいResolver/Registry/Host/Scenario SSOT、HTTP常駐必須化、Legacy `application-runtime` CarrierのScope緩和は行わない。本節自体は計画の更新で、S2/S3の実試験成功・Owner Meaning受理を**主張しない**。

**2026-11-09 C-1/C-2・Hook・公開Tool契約の横断追補（設計引継ぎ）**：[Trait設計§20](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)で、新しい[入口比較実測](evidence/CONNECT_POINT_C1_C2_2026-11-09/README.md)と[Service Carrier限定実行](evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09/README.md)を踏まえ、**C-1配備動作とC-2世代固定Service要求は必ずしも代替ではない**と精密化した。採否・C-4外部実行は引き続き未決。main #3579に入った機能定義書のHook位相/reads/uses/produces表示は**宣言の投影**であり、実Hook/業務Effect・Trait履行は未実証。main #3563/#3582のCLI/MCP失敗整合と未統合#3590のTool Discovery差分を踏まえ、広告／Required Closure／実Provider／結果を別々に観測する。TCR-01〜06は#3278/#3279→#2110/#1129/#3419/#3420/#1180/#2514へ委譲し、新Core Trait/Host/Runnerを追加しない。

**2026-11-09 原子的Traitの実行主体・BOM資格の境界（受入計画）**：[Trait設計§19](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)で、Required Traitの実装義務と、製品内実行／外部Host実行の選択（[構成の繋ぎ目判断材料](workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)のC-1〜C-4）を独立させた。C-4「TEIでは実行に接続しない」も正当な候補であり、本計画がその採否を決めない。#3540は通常CLI/StudioのS2へ向けた**未統合Draft**、#3538/#3541のRQ-037 BOM要求・Matrixも**未統合候補**。実ProviderのTrait履行、外部Receipt、同一Attempt/TargetのQualified BOM、現在Grantをそれぞれ別に検証し、TEB-01〜06は#3278/#3279→#1251/#3267/#1180/#2514への未実行受入観点として引き渡す。新Host/Resolver/Runner/Hard Gateなし。製品S2と外部Hostのどちらかを他方の前提にしない。

**2026-11-08 実装の新しい基線（main #3381／Plugin計画 #3525）**：[PR #3381](https://github.com/hat47x/tei-platform/pull/3381)がmain（統合 `f8896886c4528920a0d0cb3a2f67181971402987`）へ入り、`internal/inspectoradapter.Selected`で`authoring-tool`のexact Selection/Epoch/Host guard/Host登録Go実関数を接合、CLI注入入口とStudio Host Serviceで、Data・Rule・Relation・Capability・Application・Patternの6 Subjectを**二つの局所Go call-siteから同一Bへ合計12呼出し**する試験コードを収録した。B選択→A登録、Scope/Contract/Artifact不一致、Host Grant撤回、Epoch失効、Secret型付きError等の拒否もテストに含む。これらはS0と**限定S1**の実装であり、通常CLI `main.go`からのB自動選択、実Wails `StudioBridge`配線、実Plugin Binary/Source/Current Grantの真正性、外部Tool配信、2×2 Verified BOMはなお**S2/S3未成立**。PR旧HEADのGo package試験は一部索引漏れだけで全Health FAILし、索引を補正した新HEADを統合したが、**新HEAD全体CIの最終結論はこの追記時点では未確認**。この関係は[現行実装状況](../docs/status.md)と[Plugin計画 §19](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)へ接続した。

**次に着手する唯一の本番経路**：既存#951/#1016のResolver/ProfileをHostが受け、CLIの通常`inspect`とStudioの実Wails Hostで**別々に**同じ選択済みProvider Bを注入する。その際「選択」が実Artifact/Provider Entryの真正性を意味しないため、Selection→actual Binding→実Call/Source/Effect/受領のReceiptを分割したまま#1180/#2514の独立Expectedで評価する。Lifecycle Graphの型付きPatch/Hook（#3503/#3510）、B2C Headless（#3509）、AIチューニング（#3520）は、最初のS2を阻害しない後続Consumer/圧力試験とする。新たなResolver、共通常駐Host、Inspector専用の意味受理機構、第二Runnerは作らない。
| **S0 Typed Port**（#1158/#1824） | [CLI Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)の`inspectWorkspaceJSONWithService`、[Studio Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)の`RootServices.Inspection`。指定Service欠落・出力契約不一致・未許可Root時はfail-closedを検査 | 両方**未main統合**。通常CLI `main.go`はBuiltin経路。型付きService注入が可能でも実Plugin選択をしたことにはならない。#3371現HEADの全Go Test/Workspace validateは通過するが、`gofmt`不一致でRepository Health FAIL（下記実測） |
| **S1 selected Provider（局所）**（#951/#1016/#1251） | CLI Draft #3381の`internal/inspectoradapter`が既存Registry/Selection/SelectedCoordinator/Host guardを使ってB選択→B実Go Call／B選択→A登録拒否／epoch/Grant失効を**局所in-process**で検査する候補 | 局所B実Call・B選択/A登録拒否を含む現HEADの全Go Test/Workspace validateは通過、ただし`gofmt`不一致でHealth FAIL。合成ID/digest・登録Functionだけでは配布Binaryの実Member/Entry/Conformanceを証明しない。Target実Entry・Authorityは別 |
| **S2 CLI/Studio実Consumer**（#1251/#1824） | HostだけがTrust Rootとexact Selectionを保持し、**通常起動のCLI dispatchと実Wails Studio Bridge**を同じ契約・各自のTargetへ接続する。[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)（squash `0f3455e132d2146834d65089c68cae40c31f3055`）の従来Direct経路を6 Subject＋Not Foundで比較するGo Testは、exact-head Healthが成功した**移行前Baseline**としてのみ利用 | UIがCLI JSONを表示しただけ／factory nilでBuiltinへフォールバック／別revisionから得た同値JSONを二Consumer成功としない。B selected→A Callは同値ResultでもFAIL |
| **S3 Source・Artifact・Effect実証**（#3262/#2600/#3267） | [main #3482](https://github.com/hat47x/tei-platform/pull/3482)の限定Function Required（13実CLI Reply/42局所Controls）と[main #3477](https://github.com/hat47x/tei-platform/pull/3477)のSource-only失効を、実S2 Attemptで得たSource/Provider/Artifact/Scope/独立Expectedへ照合 | 13 Replyは原子的SnapshotでなくFunction実行資格もない。[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)の外部Provider raw/Manifestのみ改訂は**現HEAD Go Test通過／gofmt FAIL**、[Draft #3490](https://github.com/hat47x/tei-platform/pull/3490)の選択規則fingerprint変更は**現HEAD Repository Health成功**の限定Go Reference。ともに未main統合であり、Consumer World FPの一致だけでfresh／Qualified BOMにしない |

**2026-11-08 exact HEAD CI再検収（S0/S1/S3の統合条件を訂正）**。旧計画の「CI未確認」を放置せず、GitHub Actionsの各**PR HEADに対応する実Job Log**を確認した。

| PR / 対象HEAD | exact HEADで実際に確認したこと | 現在の阻害・次の作業 |
| --- | --- | --- |
| [Studio #3371](https://github.com/hat47x/tei-platform/pull/3371) `c7acfd6b` | [Health #42464668701](https://github.com/hat47x/tei-platform/actions/runs/42464668701)：`go test ./...`／Workspace validate PASS | Health FAILの唯一の記録原因はGo1.26.10での`internal/studiohost/readonly.go`、`readonly_inspection_service_test.go`の**gofmt drift**。**当該PRで整形→再実行**。既存Wails smoke/Studio PoC Aなど別Workflowはsuccessだが実Selected BのS2 Callを証明しない |
| [CLI #3381](https://github.com/hat47x/tei-platform/pull/3381) `5cdd3b9e` | [Health #42463722188](https://github.com/hat47x/tei-platform/actions/runs/42463722188)：`go test ./...`／Workspace validate PASS | Health FAILは`cmd/tei/cli_inspect_service_test.go`、`internal/inspectoradapter/selected_test.go`の**gofmt drift**。**当該PRで整形→再実行**。S1局所B実Go CallのGo PASSと通常CLI S2未接続を区別 |
| [External Import #3479](https://github.com/hat47x/tei-platform/pull/3479) `fff65f0a` | [Health #42464022323](https://github.com/hat47x/tei-platform/actions/runs/42464022323)：`go test ./...`／Workspace validate PASS。Source／Manifest各改訂のGo testがそのcheckoutで実行された範囲 | Health FAILは`internal/workspace/import_source_context_revision_test.go`の**gofmt drift**。**当該PRで整形→再実行**。Go PASSでも複数Loadの原子的Closure／Current Grantなし |
| [BOM政策参照 #3490](https://github.com/hat47x/tei-platform/pull/3490) `6b2e0b18` | [Health #42464016653](https://github.com/hat47x/tei-platform/actions/runs/42464016653) **completed/success** | Scope／Policy RuleFingerprintのReference限定PASS。選択Ruleと実Provider Source／実Artifact／実Invoke／BOM 2×2の同一Attemptは未認定 |

**CIの実務順**：新しい抽象設計やProvider追加の前に、上記3 Draftを**各Ownerの既存PR内でGo1.26のgofmtだけ修正**し、exact HEAD Healthと各専用Workflowを再検収する。Go TestがPASSしたことをCI全体PASSとは呼ばず、Health PASS後もS2通常CLI／Studio実B呼出し・S3同一Input Closureと現在権限は別の欠落辺として残す。CI修復のために第二Test Runner／新Plugin Runtimeを設けない。
**次の一件をS2の実装と検収に固定**：同じ公開`reference/workspace`・同じ`semantic-data:sales.order`・exact`authoring-tool` Contractを用い、(a)CLI通常dispatch、(b)Studio Hostの実Bridgeから、それぞれ実選択BのGo関数へ入ったことを**別のCall-site記録**で確認する。BをAへ取り違える／Serviceなし／Source ABA／Host Guard失効／部分応答の各負例では、旧Builtinへ自動fallbackせず診断を返す。公開されるCall証拠は受信先の権限と#2516のDisclosureに従い、内部Root/Secretを誤開示しない。実バイナリとWebViewの適格性はS3で別に測る。

**予定の帰属**：S0はStudio main #3371／CLI Draft #3381、S1はCLI Draft #3381の局所実Go試験、S2のProduct統合は#1251/#1824、S3のDesign Requiredは#3262、履歴Sourceと再利用判定は#2600、版Matrix/BOMは#3267、Trait実装義務はmain #3480/#3278、Scenario/独立Expectedは#1180/#2514。新しいResolver/Registry/Host/Scenario SSOT、HTTP常駐必須化、Legacy `application-runtime` CarrierのScope緩和は行わない。本節自体は計画の更新で、S2/S3の実試験成功・Owner Meaning受理を**主張しない**。


**2026-11-08 実装の新しい基線（main #3381／Plugin計画 #3525）**：[PR #3381](https://github.com/hat47x/tei-platform/pull/3381)がmain（統合 `f8896886c4528920a0d0cb3a2f67181971402987`）へ入り、`internal/inspectoradapter.Selected`で`authoring-tool`のexact Selection/Epoch/Host guard/Host登録Go実関数を接合、CLI注入入口とStudio Host Serviceで、Data・Rule・Relation・Capability・Application・Patternの6 Subjectを**二つの局所Go call-siteから同一Bへ合計12呼出し**する試験コードを収録した。B選択→A登録、Scope/Contract/Artifact不一致、Host Grant撤回、Epoch失効、Secret型付きError等の拒否もテストに含む。これらはS0と**限定S1**の実装であり、通常CLI `main.go`からのB自動選択、実Wails `StudioBridge`配線、実Plugin Binary/Source/Current Grantの真正性、外部Tool配信、2×2 Verified BOMはなお**S2/S3未成立**。PR旧HEADのGo package試験は一部索引漏れだけで全Health FAILし、索引を補正した新HEADを統合したが、**新HEAD全体CIの最終結論はこの追記時点では未確認**。この関係は[現行実装状況](../docs/status.md)と[Plugin計画 §19](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)へ接続した。

**次に着手する唯一の本番経路**：既存#951/#1016のResolver/ProfileをHostが受け、CLIの通常`inspect`とStudioの実Wails Hostで**別々に**同じ選択済みProvider Bを注入する。その際「選択」が実Artifact/Provider Entryの真正性を意味しないため、Selection→actual Binding→実Call/Source/Effect/受領のReceiptを分割したまま#1180/#2514の独立Expectedで評価する。Lifecycle Graphの型付きPatch/Hook（#3503/#3510）、B2C Headless（#3509）、AIチューニング（#3520）は、最初のS2を阻害しない後続Consumer/圧力試験とする。新たなResolver、共通常駐Host、Inspector専用の意味受理機構、第二Runnerは作らない。

**Atomic TraitとRQ-036全体マップの次の受入（既存S0〜S3の上で検証）**：[Trait設計§18](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)に、main #3381／#3531の**局所S1到達**と通常CLI／実Wailsの**製品S2未接続**を明記した。次のS2一件は既存`inspectoradapter.Selected`を再利用し、通常CLI／実Wailsそれぞれのactual B invokeを確認する。Traitの`manual-required`欠落診断は#3279の既存Contractに薄く接続する。RQ-036（main #3505）の全体ナビゲーションへTraitを表示する場合は、既存Source／Required Closure／Provider／Conformanceの**read-only参照**に限定し、Model Map Draft #3518のpartial観測を「全Trait実装済み」にしない。TMP-01〜07は既存#1180/#2514への**受入候補・未実行**であり、Source/実Provider/Build/Host Grant/意味受理の新資格を発行しない。

**2026-11-08 S2追補：D1a既存基線とTyped Error秘匿・Atomic Trait二Consumerの判定**（既存S0〜S3の変更なし）

- **D1a基線を実績として固定する**：[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)では既存CLI／Studioの直接Operation経路でData/Rule/Relation/Capability/Application/Patternの6 Subjectを読み取り、Canonical・Source位置・World/Source Index FP・EffectとNotFoundの同値を検証。exact HEAD [Health #42463508044](https://github.com/hat47x/tei-platform/actions/runs/42463508044)はcompleted/success。**D1a direct同値PASS**であって、D1b Provider Bの実Selected／actual Invokeではない。D1aを作り直さず、S2では同じSubject/Expectedで呼出ProviderのIDと実Call-siteを追加観測する。
- **Host-selected Inspectorの失敗時開示を独立P0にする**：[Studio main #3371](https://github.com/hat47x/tei-platform/pull/3371)のGo Sourceに、Providerが`operations.InspectionError`のCodeとMessageへ絶対Path/Token/内部Causeを埋め込める反例が追加された。非nil Factoryの選択Serviceだけに適用する`selectedInspectionFailure`は4つの許容Codeを固定Messageに変換し、それ以外を`OPERATION_FAILED`とする。通常CLIも別ConsumerとしてPublic Code＋秘密Message／Private Code＋秘密Message／Generic Errorの一因子負例でClient実受領を確認し、**Providerからの型付きErrorを既に安全な公開診断だと扱わない**。Source差分のexact-head CIはsuccessでmain統合済み。ただし実Wails BridgeでのProvider B実呼出しと公開結果は独立検証を要求する。
- **Traitは二種類の実Consumerで検収する**：[main統合 #3502](https://github.com/hat47x/tei-platform/pull/3502)により、P0-T=CLI/Studio read-only Inspector、P0-R=`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`→`recent_orders.sql (:from)`→`datasource sales`→`application order_entry.sales_core` Alias→実Provider Rowset→Function Contextを**別のRequired/Expected/Attempt**に固定。共通化するのはTraitのDefinition／Binding／Implementation／Invocation／Conformanceの証拠形だけ。Provider Instance/Source/Target/Current Grantは共通正本化しない。`sales.order.submit`のMutation/Transaction TraitをBrowseに暗黙継承させず、Manual Required不足とProvider未選択と実Readerの0件／partialを別に拒否する。P0-Rは#2586/#3270、Trait履行の正本は#3278/#3279。

**受入の次の最小追加条件（独立Expected #2514へ委譲）**：D1B-ERR=Typed Provider ErrorのCode/Message両方を非信頼として匿名化、D1B-TRF=TとRのTrait Required集合を別検査、D1B-CALL=同じ6 SubjectでB選択→B実Invokeを各通常CLI/Studio Call-siteで証明。**局所S1だけの成功、暫定Builtin Studio Host、Direct D1a同値、二Consumer間のTrait証拠形式同値をD1bやP0-R実Provider Readの合格へ昇格させない**。新Resolver/Runner/Host/二重Scenarioを追加せず、旧Carrierの`application-runtime`限定も緩めない。本追補は計画文書のみ、追加Go/実DB/実Wails/実Provider/Owner意味受理はNOT RUN。
**実装候補の進捗（2026-11-08）**：[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)では既存`studiohost.RootServices/ServiceFactory`へ型付き`InspectionService`を追加し、Factoryを設定した場合の旧`operations.InspectSubject`への無音fallback拒否、Subject/Effect/Schema/Fingerprint等の不一致拒否をGoコードと試験へ具体化している。ただし**未統合Draft／作成時点Go CI未検収**であり、実Compositionで選択したProviderのHost側供給、CLIの同じProviderへの接続、WebView/Build/actual Invocationの証拠は別。P0-b/cの候補として既存#1251/#1824が検証し、Studio seamの追加だけでD1bを完了扱いしない。

**今回の上限**：main統合済み参照結果の引用と、実Consumerに残る不足の計画反映。PGD-01〜06を同一HEAD・同一attemptで実CLI/Studio/Go/Frontend/Host検証したわけではない。新Contract Schema/ABI、実B呼出、G4 L3、BOM組合せの本番資格、Owner業務受理は **NOT RUN／NOT PROVEN**。
- **Host-selected Inspectorの失敗時開示を独立P0にする**：[Studio Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)のGo Sourceに、Providerが`operations.InspectionError`のCodeとMessageへ絶対Path/Token/内部Causeを埋め込める反例が追加された。非nil Factoryの選択Serviceだけに適用する`selectedInspectionFailure`は4つの許容Codeを固定Messageに変換し、それ以外を`OPERATION_FAILED`とする。通常CLIも別ConsumerとしてPublic Code＋秘密Message／Private Code＋秘密Message／Generic Errorの一因子負例でClient実受領を確認し、**Providerからの型付きErrorを既に安全な公開診断だと扱わない**。Draftの新HEADのCIとWailsの実公開結果は独立検証を要求する。
- **Traitは二種類の実Consumerで検収する**：[main統合 #3502](https://github.com/hat47x/tei-platform/pull/3502)により、P0-T=CLI/Studio read-only Inspector、P0-R=`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`→`recent_orders.sql (:from)`→`datasource sales`→`application order_entry.sales_core` Alias→実Provider Rowset→Function Contextを**別のRequired/Expected/Attempt**に固定。共通化するのはTraitのDefinition／Binding／Implementation／Invocation／Conformanceの証拠形だけ。Provider Instance/Source/Target/Current Grantは共通正本化しない。`sales.order.submit`のMutation/Transaction TraitをBrowseに暗黙継承させず、Manual Required不足とProvider未選択と実Readerの0件／partialを別に拒否する。P0-Rは#2586/#3270、Trait履行の正本は#3278/#3279。

**受入の次の最小追加条件（独立Expected #2514へ委譲）**：D1B-ERR=Typed Provider ErrorのCode/Message両方を非信頼として匿名化、D1B-TRF=TとRのTrait Required集合を別検査、D1B-CALL=同じ6 SubjectでB選択→B実Invokeを各通常CLI/Studio Call-siteで証明。**局所S1だけの成功、暫定Builtin Studio Host、Direct D1a同値、二Consumer間のTrait証拠形式同値をD1bやP0-R実Provider Readの合格へ昇格させない**。新Resolver/Runner/Host/二重Scenarioを追加せず、旧Carrierの`application-runtime`限定も緩めない。本追補は計画文書のみ、追加Go/実DB/実Wails/実Provider/Owner意味受理はNOT RUN。


### トポロジーと複合図表の評価順（未採用の追加検証候補）

[トポロジー・複合図表・開発パイプラインの共通表現](design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)を、既存契約を重複実装しないための横断的な設計・検証候補として参照します。現行のIssue優先順位、製品受入条件、実装済み機能の説明は変更しません。

最初は既存の型付き参照・Graph・Projection・Build/Compositionを棚卸しし、同じ構造からGraph・表・関係表を読み取り専用で投影します。次に、対象版とscopeを固定した実効パイプラインの解決完全性を評価し、完全性が証明できない場合に図表上の欠落を「不存在」と誤認しないことを検証します。編集・受理、認知負荷の比較、メタ生成の導入は、その後に具体的な不足と効果が観測された場合だけ進めます。

図表間リンクは、同一qualified要素の複数表示、別要素間の意味的な関係、表示用のナビゲーションを区別します。複合図表の関連グラフ・対応表も第二の意味正本としません。これらは採用済みの新Core SchemaやStudio編集APIを要求するものではありません。

### 三要素MethodのScript-first圧力試験（2026-11-06、評価待ち）

[三要素MethodのTEA-V0 / Self-build Fit/Gap](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)を、作業領域Gの**最小縦断実証の追加候補**として位置づける。Data/Function/Businessは同じauthoritative Script/標準データのRole別Projectionであり、専用図形を第二正本としない。既存#1124 M0〜M4と#1125/#1126/#951/#267がOwnerであり、新しい横断Gate・Resolver・Acceptanceは作らない。

最優先の不足は、①Source byteと意味が同一でもSource移動により旧定義ジャンプが失効すること、②可逆でないER/DFD線編集をSourceへ誤適用しないことを、**同一の実Consumerにおける正負の試験**で証明する点にある。並行するread-only評価では、**T06（三視点二世界の開示非干渉）・T07（同一Sourceへの複数Roleでのunknown保持）**を先に反証し、Policy/Role ResolverをTEA独自に実装しない。その後にバッチ/ファイル連携と業務図の役割区別、Method Pluginの実Build/Loadを扱う。Plugin self-buildのSB-1/2/3は既存Ownerの独立資格であり、TEAの初期read-only利用のブロッカーにはしない。 [T06/T07の限定Reference（Draft #2972）](https://github.com/hat47x/tei-platform/pull/2972)では、事前公開資格済みPublic Sliceの投影・unknown Roleの局所性に加え、Private metadata混入拒否をGitHub保存ファイルと同一BlobのNode 9件で確認。ただし実Source→公開SliceのPolicy非干渉／実Workbenchは未実証。#1124の実CLI→同一Consumer接合を直接優先とし、この安全性の二世界比較を#2516/#2620と並行Pressureに置く。現行の全体優先順・Gate B/C判定は変更せず、具体的な着手・採否は担当Issueの判断へ返す。

**Trait Intentと実装の齟齬の三視点照合（評価待ち、2026-11-06）**：横断Trait設計の[main統合済みPR #3027](https://github.com/hat47x/tei-platform/pull/3027)を唯一のTrait Definition/Intent/Observation/Assessment Ownerとし、三要素Methodは[設計§18〜19](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)のT10〜T13を**読み取り専用の横断受入候補**として参照する。最小の`sales.order.status`では、Businessの閉値域Intent、DataのJava EnumとDB制約、Functionの書込みValidator/未確認Writerを**同じ意味対象・Scope・版・公開資格**へ接続し、制約保証の欠落を実データ違反と誤同一視しない。MECEの母集団/排他/網羅も別Claims。原典未確定の「強属性」を標準Facetとして採用せず、業務Identity・CandidateKey・不変性は独立して審査する。既存の#1124・#1126・#2516・#2620の受入を上書きせず、Studio M0→M1の直接P0や既存Gate B/C優先順を変更しない。T10〜T13の[合成read-only Trait結合Reference（Draft #2972）](https://github.com/hat47x/tei-platform/pull/2972)では、既存の公開Slice投影から三視点Occurrenceを得て、外部資格済みIntent/Observation/Assessment参照の対応をNode 10件で局所検査済み。別Scope・旧Snapshot・非公開Claim混入の拒否を含むが、実Source Reader／Java・DB観測／Policyが正しい公開Sliceを生成すること／実Workbenchへの受領は**NOT RUN**。方法論の受入、セルフビルドや既存OwnerのGate合格へは昇格しない。

**三視点の世代結合とサブシステム分割候補（2026-11-08、並行評価）**：[三要素Method設計§19.4・§20](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)と[Reference Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)を接続。Traitを貼る時は`Projection/Envelope/Claim`の呼出元Snapshot三者一致を要求し、同じView IDでも旧図への新Trait貼付を拒否する。GitHub exact sourceを用いたV8テスト互換ハーネス**22/22件PASS**（[exact HEADのNode 22専用CIはcompleted/success](https://github.com/hat47x/tei-platform/actions/runs/42385331823/job/128441389821)、Repository Health・Product E2Eは別資格）。このStampは実Source/現在Grantを証明しないため、実CLI→同一Consumer/Policy/旧DOM失効は#1124の直接P0に残す。並行する[#3250](https://github.com/hat47x/tei-platform/issues/3250)/[Draft #3255](https://github.com/hat47x/tei-platform/pull/3255)の論理サブシステム分割は、TEA側でDataの所有候補・Functionの**direct CRUDと委譲/Effective Effect**・Businessの手順/Actorを別Evidenceとして投影するT14〜T16（NOT RUN）だけを追加。最適化・Owner判断・実Providerを再実装せず、既存優先順位/Stage B・Cは変更しない。

**Workspaceの読取文脈と三視点資格の分離（2026-11-09、評価待ち）**：[設計§21](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)へ、[実CLIの異Workspace同Snapshot負例 #3321](https://github.com/hat47x/tei-platform/pull/3321)を受けて、`Workspace観測`と`Snapshot`の別照合を追加。合成TEA Reference [Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)は公開用`workspaceObservation`をPublic Slice・Projection・Trait Envelope/Claimで照合し、同じSnapshot/表示IDでも異Workspaceなら拒否。GitHub実ソースのV8互換検査**27/27 PASS**に加え、[PR #2972のexact-head専用Node CI](https://github.com/hat47x/tei-platform/actions/runs/42591779028)も**completed / success（公開Slice12件＋Trait Bind15件）**。ただしRepository Health・実Policy・実Studioの製品受入は別資格。これは同じ物理Root・Read Attemptの証明ではない。[#3543](https://github.com/hat47x/tei-platform/pull/3543)の実CLI A/J→同一Workbench **Initial/List 0/0/1件**という限定証拠をM0→M1の既存進展として再利用し、残るGraph/Command・旧DOM・Source/Policy現在性・独立Readの実接合へ直接P0を絞る。[#3728](https://github.com/hat47x/tei-platform/pull/3728)の`Catalog read成功`と`Validation NG`、[#3518](https://github.com/hat47x/tei-platform/pull/3518)の部分Model Mapを別Claimとして示すT18〜T20は未実行。Studioに別Reader/Policy/Validator/新Ownerを作らず、既存Stage B/CとSelf-build/ETLの独立Gateは変更しない。

**Read・Validation・Source結合・開示の独立資格（2026-11-09夜）**：[merged #3728](https://github.com/hat47x/tei-platform/pull/3728)の実Catalog read成功＋`validation.ok=false`と、[merged #3492](https://github.com/hat47x/tei-platform/pull/3492)のSource Index構造PASSでも偽装Subjectへの実Inspectが拒否される反例を[TEA設計§22](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)へ接続。T21 ReadとValidateの別表示、T22 Locator Byte妥当性とSemantic Subject実在性の別検収、T23非公開診断件数の二世界非干渉を**新規Consumer受入候補（NOT RUN）**とした。既存[Reference Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)にはSubject/Relation/Claim/ViewのArray accessor/iterator/holesを入力資格外として拒否する4負例を追加し、GitHub byteを用いたV8 shim **31/31 PASS**。さらに[合成T18 Catalog/Validation表示Reference（Draft #2972）](https://github.com/hat47x/tei-platform/pull/2972)へ6件を追加し、GitHub実ソースのV8互換試験**37/37件PASS**（Public14 / Trait17 / Catalog6）。Read成功とValidation NGを別表示し、未観測ReadからValidationを捏造しない。**新head Native Node CI、実Catalog JSON受領、現在Policy/GUIは別資格**。#1124のGraph/Command/旧DOM・現在Source/Policyの直接P0を最優先に維持し、ValidatorやSource Registry、第二PolicyをTEAへ複製しない。

## 作業領域H：AIによる開発支援

### 目的

AIを大量の成果物を生成するだけの仕組みにはしません。既存知識の探索・変更案の作成・検査・意思決定の支援に利用します。

### 二段階

1. `ISSUE-000047`：人とAIが共通に利用できる、既存知識を優先した意味情報の探索（PAS-06）
2. `ISSUE-000027`：AIによる変更案と共通の受理の境界（PAS-01・PAS-06）

AIによる提案には、意味の正本を変更する権限はありません。

検索順位・類似度・モデルの確信度も、意味の正しさを示す検証根拠ではありません。

AIの自律的な作業と実行権限を区別します。AIがツールを直接呼び出せることを、能力や利用許可の仕組みの代わりにはしません。

### AI-firstの現行受入：TT-56実CLI閉包→盲検T1-Bと実Providerを分ける（2026-11-08）

[AI-first資格・実測対応設計§10](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)を、作業領域Hの**受入順と担当への引渡し**として参照する。旧[PR #3241](https://github.com/hat47x/tei-platform/pull/3241)の複数段追記は主張の歴史として設計本文へ保持し、本計画へは重複掲載しない。現行mainのIssue/ADR・TAI Runbookが受入・実験の正本である。

**確認済みの最小成果**：[main統合 #3354](https://github.com/hat47x/tei-platform/pull/3354)は旧#3324/#3326を一つのContext Gateへ統合し、固定公開Sourceについて**11実CLI Reply＋32局所Consumer Controls**（既存16・Required三Rule 8・Rule/Relation/Source詳細8）を[専用exact-head CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)で成功、Healthも別Runで成功。旧二PRはclosed/unmerged。これは一時コピーの欠落/改変を検出した実績であって、実Harnessの全Byte配送・全Domain/Function/BusinessのRequired Closure、AIのT1-B新規発見は**未証明**。`boundedReference=partial`／`qualifiedTestGeneration=false`を維持。

**P0-A/P0-B/P0-Cを分離**：(A) #3262の実Loader/Rule/Relation/Source/Index/Canonical取得と独立Requiredを一Subjectの版に束縛し、不足理由を返す。(B) TAI #168・TEI #2509/#2515で独立Expected/未公開Gap/旧回答への**実OS拒否**、許可`Command+Flag`/argv・Tool Reply全Byte/終端・Session Log/Egress Sink/Consumer受領を実測して初めて比較/盲検資格を与える。(C) #3273/#1180/#2514で**未知GapのT1-B Candidate一件**を独立Expected/Mutantで採点。既知行番号0のT1-A成功は未知Gapの発見でない。[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)の部分Scenario目録TT-105/106はfixed-head Dogfood/Health/依存CIがいずれも成功したが、**部分受領＝notObservedであってuntestedConfirmedではない**、全Coverageはunknown、Draft未統合。

**別レーンの実資格**：[main統合 #3353](https://github.com/hat47x/tei-platform/pull/3353)は空Targetの`EmptyPolicyHandoff`に`HandoffConfirmed=false`なら`unverified`を返す局所修正を加えたが、実相手方Receipt・Target Set全Readではない。[main #3350/#3363](https://github.com/hat47x/tei-platform/pull/3363)の参照Capability/Bindingは12宣言・未選択4、24 Bindingの選択22/欠落2（`editor.database_workbench`）、ただしReferenceの`missing`≠現Validatorの違反≠実Plugin Loader。[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)はStudio InspectorをHostの選択InspectionServiceへ接続する候補だが、CLIは依然直接経路であり、**一Pluginのselected→actual loaded/invoked→CLI/Studio二Consumer**を実証したものではない。#1251/#951/#3292へ実Invoke試験を引き渡す。

**Inspector D1の実接続（2026-11-08追補）**：[AI-first資格・実測対応設計§11](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)へ、CLI/Studio双方の**既存直結Inspectorの応答基準**[Draft #3378](https://github.com/hat47x/tei-platform/pull/3378)、**CLIの型付き注入入口**[Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)、**StudioHostのHost選択Service入口**[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)を別資格で接続。両Surfaceの応答一致・Host Factoryからの委譲は、選択済みPlugin/Providerの実Build/Load/Invokeを証明しない。次は#951/#1251/#3292が所有する**一read-only Inspectorの`Provider B selected→B actually invoked`**をCLI/Studio二Consumerで実測し、B選択→旧Builtin A呼出し／明示Service未提供→暗黙Fallbackの負例を拒否する。既存`inspect_projection`の共有Result検証へ寄せ、第二Registry/Hostを作らない。#3378/#3381/#3371の旧exact-head Repository Healthは**それぞれ失敗**（前二者にはGo整形問題、Studioには既存Service注入契約の失敗もある）。他Workflow PASSと合算しない。[Draft #3370](https://github.com/hat47x/tei-platform/pull/3370)の`requirements.resultShape=array`は実Reply形状の改善候補で、外側の成功Envelope・Source完全性・権限とは独立。**P0のT1-B盲検/H2隔離と並列**とし、D1の完成を前提にしない。

**Tool公開とReaderの停止線**：[main #3333/#3340](https://github.com/hat47x/tei-platform/pull/3340)の`writeFlags`・Help/Discovery整合は安全な実argv/OS非書込みの証明ではない。[第2回CLIドッグフード #3356](https://github.com/hat47x/tei-platform/pull/3356)は一部改善とD-3/9/10残存を記録し、**現在の実入口はCLI、MCPはNOT RUN**と分類。Managed Markdown、ReaderでParsedされたSource、実Consumed Meaningを混同しない。旧`*.tei`の一Function→実Provider read-only（#2586）と新`*.tei.md` Reader移行（#3030/#3031）は並行し、T1-BのHard dependencyにしない。

**PoCのA3自律作業**は採用ADR-0035に従い可能だが、製品のHuman/Owner Shared Acceptance（#267）・実Host current Grant/Effect（#1174/#2598）・外部不可逆効果の人的確認は別。**今回の追加はDocs/Planのみ**。新H2/H3四Session・実T1-B・実DB/Plugin/Host・正式Meaning Acceptance・C1はNOT RUN。Scenario/Expected/Registry/Host/Runbook/Readerの第二正本は設けない。
### AI Proposalの入力・検証・受理・Host境界（2026-11-06横断追補）

[ADR-0022](adr/0022-ai-proposal-canonical-acceptance-boundary.md)と[#267](https://github.com/hat47x/tei-platform/issues/267)の**Shared Acceptance**を唯一の正式なSource変更経路として維持し、AI専用の書込み・受理・Provider直結経路を追加しません。[ADR-0030](adr/0030-context-scoped-epistemic-knowledge.md)／#958の由来・確認履歴・用途相対資格を、生成AI向け入力にも適用します。具体的な最小横断仕様・UI状態・AIB-01〜12の負例は[AI-first設計§5.4](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)へ一本化します。

- **Inputの現在性と外部送信**：Source ID/実消費byte/Revision/fingerprint/span、Semantic/Operator Lock、依存閉包、purpose/Principal、Allowed *proposal/read* capability、開示範囲を別々に固定します。宣言Input＝実Load/Parse済みではなく、内部read-only＝外部LLMへの送出許可でもありません。Source本文・Tool結果に埋め込まれた命令はAuthorityではありません。
- **Candidateと検証**：Candidate/Attempt/base revision、Typed Diff、根拠Source span/意味対象、前提/未解決を保持。構文/型/参照と、引用の意味妥当性、Source鮮度/完全性、Rule衝突を別判定します。欠落・部分取得・曖昧・矛盾は保留し、再生成・A→B→Aでも旧Reviewを自動継承しません。
- **受理とEffect**：人によるReview/Owner意味受理→Source反映→reload/Conformance→型付きAction Intent候補→外部Hostの現在Grant/Target確認→実Provider Effect/Receipt/独立readを段階別に検収します。受理済み意味は現在の実行許可を保証せず、Commit応答不明では無確認再試行を行いません。
- **Owner分担・優先順**：#988＝構文/Operator、#2619＝意味Role、#2622＝Data Operations、#267＝Shared Acceptance、#958＝認識状態、#2515/#2516＝Disclosure、#1180/#2514＝Scenario/独立検証基準、#2598/#1174＝実Target/Effect。P0固定CandidateのHeadless負例→P1独立検証基準/既知Mutant→P2実Consumer/外部送信・二世界非干渉→P3実Host/Receipt→P4実AI生成品質の順とし、低レベル試験のPASSを製品統合へ合算しません。

**状態**：設計と計画の接続のみ。AIB-01〜12、実AI入力/生成精度、OS・外部LLM送信、実Provider/Host、Studio/SHO、Owner正式受理の新しい横断E2Eは**NOT RUN**。新Schema・Core enum・Acceptance Engine・Policy Engine・TEI独自Hostの採用決定ではありません。

### `*.tei.md`統一後のAI-first Authoringと受理資格（2026-11-06追加決定）

**Ownerの現行方針**：TEI向けAuthoringを`*.tei.md`へ統一し、`*.tei`の新規使用を原則停止、既存資産は段階移行する。従来の「Markdownで検討し、別`.tei`へ切出し」「`.tei.md`は標準にしない」という#2621の旧比較案は現行方針ではない。ただし**拡張子の決定とParser/Build/Studio/SHOの実対応は別資格**であり、`*.tei.md`という名称、OKFタグ、AI候補をOwner Meaning/実Runtime Grantへ昇格しない。

**最小Authoring Slice**は、`*.tei.md`のSource発見→raw byte・論理Source ID・Revision固定→Seed/Manifest/Language Plugin実Selection/Pin→Section/Row/Source spanごとのRoleとRead-only解釈→根拠付きAI Candidate/独立Expected→#267 Shared Acceptance→Source保存と再Load→実Build/Consumer読取の照合とする。通常の`*.md`は説明/SHO/OKF用途として保持し、旧`*.tei`の移行は原内容の構文差・引用/Link再対応・Source SSOTを照合してから行う。単純renameや同ファイル内の一部受理をファイル全体の意味受理としない。Markdown中の指示文/コードフェンスや外部OKF metadataはAIへの命令・権限として扱わない。

**既存Owner/受入**：#2621はAuthoringと配置Lint、#988/#3028はLanguage/Bootstrap、#2620はRole、#267はShared Acceptance、#2514は独立Expected、#2515/#2516は用途別外部送出、#2386と#1174はTenant/Host Grant、#2598は実Request対象。新規差分は[AI-first設計§5.5のAMF-01〜07](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)に限定する。AIB-01〜12・#2621既設FM-01〜08・#988 BS-01〜15とCaseを二重実装しない。

**状態と停止条件**：命名統一はOwnerの決定。既存`*.tei`の一括変換・Parser実対応、同一Source/Revisionでの実CLI/Writer/Consumer、OS/LLM送出、Tenant別実Provider、Host Grant/Receipt、Owner意味受理、AMF-01〜07は**NOT RUN／NOT ACCEPTED**。対応不能は明示して旧Sourceの正本性を保持し、新しい独自Host/Resolver/Registry/Parser/受理Engineを前提としない。

### AI-first初回縦断の最小Done：Source実消費→候補→受理（2026-11-07）

過去のAIB-01〜12／AMF-01〜07を増殖させるより、[AI-first設計§5.6](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)を**最初に実行する一つの検収手順**として優先します。旧`*.tei`から新`*.tei.md`への移行は[#3030](https://github.com/hat47x/tei-platform/pull/3030)（docs-only Draft）のM0〜M6がOwner。現mainの`*.tei`固定Loader・`filepath.Ext`・Markdown非対応のTEI Parserという実際の障害を解消するまでは、新Source形式の実運用やAI定式化E2Eを合格としません。

**第一Slice**：注文一件の`*.tei.md` raw Sourceを固定し、実Manifest/Language Plugin/Source Indexが選択した断片の`actual bytes・Revision・span・Required Input`を観測。次にLLM非依存の固定Typed Candidate・誤引用・部分/未知の独立Expectedを#1180/#2514で検査し、#267のHuman/Owner Shared Acceptanceを通った場合に限り実Source更新→reload/Conformanceを検証。AI生成品質や外部LLM送信はこの初回Doneの前提にしない。Owner意味受理が未取得なら機械的な解析/検証合格までで停止する。既存TM-02/05・AMF-03/04・AIB-01/03/10を**一Fixtureへ再利用**し、新Parser/Runner/第二意味正本を設けない。

**Done以降を独立に判定**：#2515/#2516で外部AIへ実送出した内容とOS/Tool/CIの各Sink、#2598/#1174で現Principal/Target/Required-Supplied-Evaluated Guards、実Provider Effect/Receipt/独立Read、#1824で後着Resultの現Selection Epochを評価する。業務Meaning受理とHost current Grantは独立した二軸であり、その両方が成立してもCommit/業務完了の証拠にはならない。既存#2901 Green／#3017 Red（AI追加Scenarioの**一件の実Product Mutant kill**）、#2853の限定PostgreSQL unknown伝播は、いずれも当該対象・HEADのEvidenceであって本Sliceの実行PASSには数えない。

**現状**：本追補は設計・基本計画の更新のみ。新`*.tei.md` Loader/Parser、固定Candidateの実Shared Acceptance、AI/OS送信、実Host・Providerとの同一Attempt、Owner正式受理は**NOT RUN／NOT ACCEPTED**。P0の実行後に初めて対象Revision・Consumer・不足Guardと証拠を明記して次の作業へ進む。

### P0二レーン実証とCompletionの証拠資格（2026-11-07修正）

前節の「初回P0は`*.tei.md`実Reader/Parserが先決」という停止条件は、**新Authoring形式からのShared Acceptanceの合格条件**へ限定する。旧`*.tei`の実Provider read-only／独立Expected検証は、[Script起動 Draft #2884 §10](https://github.com/hat47x/tei-platform/pull/2884)に合わせた**P0-L（Legacy）**として並行着手できる。**P0-M（Migration）**は`*.tei.md`の正しいSource発見・Markdown解釈・実Parser/Build入力まで別試験とし、最後に**P0-J**で旧新それぞれのSource byte/Revision/実消費範囲と同じ業務Requirement/独立Expectedを照合する。LのPASSをMへ継承せず、MのParse PASSを実Provider成立へ拡大しない。#2667が移行横断TMDと合流・Legacy退役を管理し、#3030がAuthoring固有のTMと表層、#3031が未対応Sourceの暫定拒否を担当する。第三のMigration Case SSOTは設けない。

**必要対象の完全性を独立に検査**する。[main統合 #2926](https://github.com/hat47x/tei-platform/pull/2926)の`EvaluateCompletion`は、渡されたTargetSet/Coverage/EmptyPolicyの評価器であり、Callerが宣言した`CoverageComplete=true`・行数0・表面上の`holds`からProviderがOwner対象全件を読んだとは推定できない。`required→discovered→loaded→parsed→actual consumed`、Required/Observed TargetSet/Row/Field、必要Guard/実評価Guard、空集合の独立証拠、現Principal/Tenant/Scopeを同じClaimに照合する。未観測なら`unverified`、矛盾なら別扱い。#1180はRequired Obligation、#2514は独立Expected/判別力、#2586/#2628は実Provider Read、#1163は同一Attempt、#1174は実Effectを担当。新しいCompletion Engine/Proof Registryを作らない。

[AI-first設計§5.7](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)へ実証順・既存Fixture再利用・Evidence上限を集約。**限定既存成果**：#3031の固定HEAD `63953ccb`の未対応`*.tei.md` Guard Go/CI success（Draft/未merge）、#2926のCompletion evaluator main統合と局所Go PASS、#3028のBootstrap正負対照Draft。これらから新Reader/実TargetSet/Owner meaning・実Host合流は導かない。P0-L実Provider、P0-M実Reader、P0-J旧新Meaning比較、同一Source改訂のCandidate→Shared Acceptance／Consumer、実OS/AI送出・Host/Provider Effectは**NOT RUN**。Go単一バイナリと既存外部Host権限境界を維持する。

### 既存Proposal・Command境界を使った初回検収の修正（2026-11-07）

[AI-first設計§5.8](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)へ、mainに統合された**実APIと実証の上限**を接続した。新たなAI修正案APIを作る前に既存`validation.Propose(diagnostics, worldFingerprint)`・`validation.Accept(&proposal, acceptedRevision)`（[#3041](https://github.com/hat47x/tei-platform/pull/3041)）を限定利用する。前者は診断に版を付け、後者は**局所`Accepted`フラグを立てるに過ぎず、Human/Owner authority・正式Source更新・Shared Acceptance／Reloadとは別**。修正PatchやSource引用span/入力閉包/Allowed AI capabilityもこの実APIには無い。真正なDecisionは引き続き#267、Source/引用の現在資格は#958/#988がOwner。

[Commandの修正 #3042](https://github.com/hat47x/tei-platform/pull/3042)により、Authorizer未設定は`COMMAND_AUTHORIZER_UNAVAILABLE`、宣言PeriodのObserver未設定は`COMMAND_PERIOD_STATE_UNAVAILABLE`として拒否され、実Executor後の宣言外Effectは`EffectUnknown`で保全される。ただし参照用`ReferenceAuthorizer{}`は現在の実Host発行者・Policy・実権限の証明ではない。「拒否・unknownだからno Effect」は誤り。差の保持について[AC-8実証 #3060](https://github.com/hat47x/tei-platform/pull/3060)は参照Storeの`OperationRecord.Differences`／CLI/Studio投影まで。実SQL/DB Commit/Provider/Owner業務完了ではない。既存#2598/#1174のTarget/Grant/Receipt/独立Readで後段を別検証する。

**初回Doneの更新**：P0-L旧`*.tei`の実Provider read-onlyとP0-M新`*.tei.md` Readerを並行維持。新形式の移行・TM-01〜14／M0〜M6の**主担当は#3030**（#2667 TMDは横断追跡対応表）。#3031は未対応新形式の暫定Guardである。既存AIB-01/02/03/07/08/11/12とTM/AMFを同一Fixture・独立期待値で照合し、`diagnostic proposal created → local flag accepted → owner source accepted → host request admitted → provider effect observed`を**五つの別資格**として報告する。新AI API／第二Acceptance/Host/Registry/Runnerは設けない。

**用語**：[#3039](https://github.com/hat47x/tei-platform/issues/3039)と[docs/naming.md](../docs/naming.md)に従いテストの「Oracle」を「独立期待値」「独立検証基準」に置換。**Oracle Database**はDB製品名として区別し、歴史的コード名・API・URLは改名しない。設計/実証と命名更新を混同しない。#3041/#3042/#3060の既存限定Go・Reference証拠を、`*.tei.md`新Reader／人の正式受理／実外部Host Providerの同一Attempt成功へ自動継承しない。**当該統合はNOT RUN／NOT ACCEPTED**。

### AI Proposal・Product Skill・Attested Computationを三経路として接続する（2026-11-07）

[AI-first設計§5.9](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)に、外部Harness実測と[OKF v0.2 Attested Computation](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)の利用契約を接続。**AIの自由な意味Proposal**（#267のHuman/Owner Shared Acceptance）と、**受理・Pin済みComputationへの型付きParameter Bind**（Executor実Receipt＋決定的Attester）と、**Provider/外部Actionへの型付きIntent候補**（#2598/#1174の現在Host Grant/Effect）を明確に分離する。計算をAttestできたことから業務Meaning正当性・Provider対象集合全件・現在の実行資格を導かず、`validation.Accept`の局所`Accepted=true`からComputation承認や正式Source受理を導かない。

**実績更新**：TEI #2605／[TAI実S1 CI #42101157330](https://github.com/hat47x/tai-pleroma/actions/runs/42101157330)では、TEI固定HEAD `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`の実Go CLIを外部リポジトリCIから起動し、公開`semantic-data:sales.order`のSource Index path/fingerprintと原byte照合を限定確認済み。これは**S1の実CLI read-only**であって、外部AI Harness自身の実Tool request/argv、Product Skill root/参照文書の実Loader読取、Source必須全集合、現在Disclosure、Owner受理ではない。

**優先する最短手順**：既存P0-L（旧`*.tei`実Provider Read）／P0-M（新`*.tei.md`のReader/Guard）／P0-J（旧新同一業務の比較）の各Ownerと並行して、AI Skillレーンは**TAI #168のH2 Tool-only最初の1実Session**を最優先で観測する。その後、固定Task/実Binary/Skill bytes/Tool profile/独立期待値を変えず、H2/H3各2 Sessionの計4回で実Tool使用・Skill root/Refsロード・誤答/Warning/Unknown/Costを検収する。割付した条件と実際に提供された条件を分け、失敗Sessionを採点から恣意的に除外しない。初回Tool/Skill比較の途中でOKF用Skill説明を書換えない。

**後続の任意read-only PoC**：[TAI #514](https://github.com/hat47x/tai-pleroma/issues/514)の既存AC-01〜12に従い、正規Computation/typed parameters/実Executor bytes・Job/Receipt/Attesterの実Identityと独立再Readを一件だけ照合する。OKFの`generated`/`verified`/`stale_after`は知識Conceptの来歴・現在性であり、実Actorの認証やTEI Owner Meaning/Host Grantではない。OKFはAttester ABI・wire format・sandboxを定めないため、第二Host/Receipt Registry/新共通CLIを本計画から追加しない。Product SkillのOwnerは**open #2605**、TEI #2614はclosed/duplicate。P0の実Harness・H3ロード・Attester/Job/Receipt E2Eは**NOT RUN**。Go単一バイナリ、既存Shared Acceptance、外部Hostの責務を維持する。

### 外部AI Harnessの実Tool接続済みと比較試験の未適格を区別（2026-11-07）

[AI-first設計§5.10](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)に、従来の「初回H2 Tool実起動は未実施」という状態の**明示的な訂正**を記録した。[TEI実測PR #3113](https://github.com/hat47x/tei-platform/pull/3113)／[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)のDSH 0.2.2-rc.1 Windows headless探索では、固定TEI `50ebf801…`と公開Source/固定Taskで、完了した`h2-08`のTool call 56／result 55、実TEI Binary起動32件とCLI/Source Indexの読み取りを確認した。**外部AI Harness→Tool request→実TEI CLI→Tool Resultの技術的接続は限定達成済み**。PR #3113はEvidenceのOpen/未マージであり、探索9試行中の短い接続確認2、長いTask完了1、Tool JSON連結エラー5、無効Config1は試行別の歴史的証拠として記録する。

**次のBlocking Exitを変更**：最初の「実Tool Callを一件取る」から、**比較資格を持つH2 Tool-onlyを一件取る**へ移す。探索Sessionはread-onlyでも作業Dir外のWorktreeを読めたため、Skill/独立期待値/旧回答の物理非露出・OS隔離が実証されていない。Session logの`enabled:false`は実通信送出0の証明ではなく、Windows `workspace-write` SandboxはWin32 5で初期化失敗。TAI #168の既存SK-02 RunbookをTrial正本として、(1)禁止Path/Canary実アクセス拒否、(2)許可された外部送出と未観測Sinkの分離、(3)固定Task/Source/CLI/Tool/Model/Skill／Reviewer-only期待値、(4)実Tool→回答→Consumer受領・独立採点を同一Sessionで確認、(5)その後H3実Skill root/References Loadと`H2→H3→H3→H2`四Sessionを開始する。探索9試行から成功例を事後選抜せず、**比較適格Trial 0/4**のままとする。#2509がDSH固有の実証、#2515/TAI #232が実OS・開示、#2605がTEI Skill、TAI #168が比較Owner。MALFORMED_RESPONSEはTool連結JSONという実失敗であり、`maxParallelToolCalls:1`でも回避できなかった。異なるProfile/Provider/モデルへの修正は新Pinned Trial Groupで行う。TEI独自Harness Runtime/Runner/Validatorは増設しない。

[main統合 #3051](https://github.com/hat47x/tei-platform/pull/3051)のCompletion評価は、未確立Coverageの空Targetを`unverified`、完全と確認した空Targetの`deny`を`fails`へ区別するよう改善済み。明白な反例が1件あれば部分取得でも`fails`。ただしCaller自身の`CoverageComplete`真偽や全Source/Target/Field/Tenant/Snapshot・実Provider Readを証明する改修ではない。[Draft #2935](https://github.com/hat47x/tei-platform/pull/2935)の`readByLoader`もLoader読取の限定目印であり、Parse/Build/実Consumer読取へ昇格させない。P0-L（既存`*.tei`実Provider）とP0-M（新`*.tei.md` Reader）を並行維持し、#3030は移行Owner、#2514/#2506は独立期待値・母集団検証を担当する。

**現在の証拠と未達**：DSH→TEI CLI技術的接続は探索1件で限定実測済み、#3051 Completionの局所修正はmain統合済み。一方、比較適格H2/H3、H3実Skillロード・効果/コスト、実OS拒否・Egress不在、`*.tei.md` Reader、全Provider対象集合の完全性、Owner意味受理・実Host Effectは**NOT RUN／NOT QUALIFIED**。前節§5.9の「H2実Tool呼出し未実施」は新Evidence以前の歴史状態であり、現在の適格性判定を上書きしない。

### AI-first実装境界の新しい到達点と次の一試行（2026-11-07）

[AI-first設計§5.11](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)で最新mainの実装と実証を接続し、前節の「H2 Tool接続自体は限定達成、比較適格H2/H3は0/4」を維持。**新しい次のBlocking Exit**は、禁止Source/独立期待値/Skill/過去回答への実アクセス拒否と、用途別の実送出範囲が確認された環境で、**H2 Tool-onlyを比較適格な一Sessionとして実行すること**である。TAI #168の既存Trial/Task/割付を唯一の比較正本とし、探索9回の事後選別をしない。

**根拠を持つ実装進展**：[main #3120/#3122]のStudio版付き修正案（あくまで未受理）、[#3143](https://github.com/hat47x/tei-platform/pull/3143)の認可種別`reference-allow-all/reference-allow-list/external`、[#3124/#3127/#3131/#3155](https://github.com/hat47x/tei-platform/pull/3155)の永続化観測`not-applicable/not-observed/receipt-observed/resource-observed`（`persisted:true`は現在Resource一致の場合だけ）を、**個別の製品Evidence**として採用する。実PostgreSQLでのReceipt Lookup/Resource別接続観測の限定経路があるが、Business Completion/current Host Grantは未実証。ツール環境Grantの[main #3121](https://github.com/hat47x/tei-platform/pull/3121)は`PERMISSION_GRANT_CAPABILITY_NOT_RESOLVED`で停止しており、ToolをPluginと決めるかは未決。AI提案→Source変更は依然#267 Shared Acceptance、Host実行は#1174。PoCコード/Fixtureの開発を全部Owner待ちにしない。[採用ADR-0035/#3150](https://github.com/hat47x/tei-platform/pull/3150)に従いA3の機械検証・自律作業と、対外不可逆効果に必要なA4の人的確認を区別し、製品業務Meaning/実Grantの資格を機械PASSで偽装しない。

**実効隔離と送出の反証を先行**：[main #3152](https://github.com/hat47x/tei-platform/pull/3152)のTEI`access.FileSystem`は`os.OpenRoot`でリンク/ジャンクション経由のRoot逸脱を拒否できるが、DSHのShell・子Process・Socketまでは閉じ込めない。Access journalの改変検出も独立headの固定がなければ末尾削除や全再Hashを防げない。[main #3153/#3161](https://github.com/hat47x/tei-platform/pull/3161)のDSH 0.2.2-rc.1 **ローカルMock実測**は、既定Profileでの`dsh_session_log`約45KB送出、Tool markerのモデル入力とは別の2箇所への複写、off→on再開/503/stream切断後のbacklog再送を観測した。これは「`enabled:false`にしたから秘密送出が0」「read-onlyだから外送なし」の否定材料であり、実Providerでの漏えい確定やOS全出口の保護実証ではない。#2515/TAI #232で許可Audience・実測Sink・未測定を同Attemptに記録し、公開用EvidenceへMock rawを無制限に転送しない。

**実Source・Tool資格も別照合**：[main #3148/#3160](https://github.com/hat47x/tei-platform/pull/3160)のCLI discoveryは`modes`/`successShape`を広告するが、実argv・stdout/stderr/exit・Response Mode・実Workspaceを別検証。[Dogfooding Draft #3162](https://github.com/hat47x/tei-platform/pull/3162)は`view effect:read-only`でも`-execute -operation-log`や`-lifecycle-state`によるローカル書込みを観測した。初回H2 Tool allowlistは不要な`-execute`を公開せず、構文上のread-only・参照API・実OS/Host制約を分ける。[main #3163](https://github.com/hat47x/tei-platform/pull/3163)により`modelRevision`は**管理Source Indexの版であってWorldの版ではない**と明確化。空Workspaceで`ok:true`＋`std` entriesが出ても`declarationCounts.total=0`なら必要Subjectの実Loaded/Parsed/Consumed資格は未確立。旧`*.tei` Provider P0-Lと新`*.tei.md` P0-M/P0-Jは引き続き別レーンで進める。

**今回のDone/NOT DONE**：docs/planの契約追補のみ。実Tool探索とMock egress/限定PG・CLIは**既存mainの別HEAD**で観測された結果であり、このPRの同一End-to-End成功ではない。比較適格H2/H3、実OS隔離・全送信拒否、Studio Wails実通過、正式`*.tei.md` Reader、Owner Meaning、実Host・Business Effectは**NOT RUN／NOT QUALIFIED**。新AI Host/Policy Registry/Approval Engine/Runner/Fixture正本は増やさない。PR #2492はmainの[ADR-0035採用 #3150](https://github.com/hat47x/tei-platform/pull/3150)等より古いbaseから発展しているため、統合前にcurrent mainへの差分・旧A4記述と新受入境界の衝突を**別途rebase/レビューで照合**し、旧PRの既存CIを現main資格とみなさない。
### AI-first資格分離の初回実証（2026-11-07、現在mainへの成熟差分）

[AI-first資格・実測対応](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)を横断受入の現行基線として参照する。旧[PR #2492](https://github.com/hat47x/tei-platform/pull/2492)は現行mainから大きく乖離しているため、**全追補を機械移植しない**。本節は成熟したAI受入条件だけを移すもので、同PRに含まれる別件の要求・基礎データ・研究の自動採否ではない。既存Owner/ADR・TAI Runbookと競合する第二の正本・中央Agent Hostを作らない。

**三経路**：AIの自由なMeaning Proposalは#267のShared Acceptance、受理済みComputationの型付きParameter Bindは正規実行物・Receipt・独立Attester、Provider等のEffectful Intentは現在Host Grant/OperationID/Receipt・Resource再Readへ分ける。`validation.Accept`局所フラグ・Studio未受理案、PoCの機械検証によるADR-0035 A3、OKFの`verified`、本番のMeaning/Grant/Effectを相互に代替しない。

**現行実績と制限**：DSHからTEI実BinaryへのTool transportは[#3113](https://github.com/hat47x/tei-platform/pull/3113)で探索限定成功。ただしCatalog Tool Result 8,192 byteの途中終了、Tool call 56/Result 55、stdout/stderr付着、参加者の禁制資料へのアクセス拒否未確立により、TAI #168のH2/H3比較適格は**0/4**。[#3200](https://github.com/hat47x/tei-platform/pull/3200)のTool子Process環境実測、[#3153/#3161](https://github.com/hat47x/tei-platform/pull/3161)のDSH Session Log既定送出・再送は、モデルPrompt以外のSinkと実OS隔離を明示確認する根拠。これらは実Providerの全Egress非漏えいやSkill増分価値の証明ではない。

**Tool選択・Source完全性**：[main #3210](https://github.com/hat47x/tei-platform/pull/3210)ではCLI受理53／公開discovery19という実差があり、非公開34は意図的な選択。AI側がCLIで呼べることと公開Tool/Host資格は別。[Draft #3162](https://github.com/hat47x/tei-platform/pull/3162)の`read-only`表示でも`-execute`でローカル書込みがある反例を残す。SourceはRequired→Discovered→Loaded→Parsed→Actual Consumed、ProviderはRequired Target Set→Observed Result Slice→Predicate Evaluation/Scopeを照合。[現main向けDraft #3238](https://github.com/hat47x/tei-platform/pull/3238)は旧#3175合成Q05〜07を再配置し、隔離PostgreSQLでTenant A `{3,9}`の`$top=2`対`$top=1`を実Reader/独立Expectedで確認（[限定CI #42346176936](https://github.com/hat47x/tei-platform/actions/runs/42346176936)）。**有限Reference実Readを実Product Function→Query全件保証へ拡張しない**。[Draft #3156](https://github.com/hat47x/tei-platform/pull/3156)の実Build成功・Source閉包拒否も、artifactと資格を分ける実例として扱う。

**2026-11-07最新の資格差分**：[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)の4 Session原EvidenceでH3-A/BのProduct Skill本文各5,482 byte・参照文書の**実Tool Readは限定成立**。しかしH2-Aは旧Session再使用で事前隔離・割付が揃わず、Skill効果/独立回答の比較適格性は**0/4のまま**。新TrialではG0読取拒否→G1許可送出→G2 Tool Result全体性を満たし、Skill Source原byteとLoader wrapper/実消費本文を区別する。[再開の実測 #3237](https://github.com/hat47x/tei-platform/pull/3237)はSession Logの既ACK区間が非再送でも、再開`messages`へ旧Canaryが再出現。**Log非再送≠モデル入力非再開示**なので、再開時Audience/Disclosureを別に照合する。

**CLI/実行時のCurrentness**：[実装PR #3247](https://github.com/hat47x/tei-platform/pull/3247)はWorkspace名・manifest SHA・形式版を5つのCLI応答へ追加して限定実験済みだが未統合で、名称自体は一意IDではなく`source-index`の対象なし応答にも未対応。[Issue #3242](https://github.com/hat47x/tei-platform/issues/3242)は`tei commands`の引数型・enum・default未広告を担当。Hostへ渡すTyped Intentの元となるWorkspace/CLI/Tool Requestを**実解決・実消費した版**で束縛する。[main実測 #3254](https://github.com/hat47x/tei-platform/pull/3254)ではSource属性追加でWorld/Model版が変わってもOperation `basis`と件数は不変で、現実行面に期待Revision照合入口はない。#2598/#1125/#1174のOwnerへ、保存済み操作面の選択時点・実行直前World/Model・現Target/Grantの照合を渡す。**実装方式や既定の拒否判断は未決**、参照`basis`一致・CLI/Studio表示だけを実行資格としない。

**次の小さいDone**：(1) TAI #168／TEI #2509/#2515/#2605で、OSの禁止Source拒否・許可Egress/Tool Result完全性・Answer/Consumer受信を備えた**比較適格H2一Session**を新Trialで成立させる。(2) #2586/#2514/#2506で旧`*.tei`の一Function→実Provider read-onlyと独立期待値を照合する。(3) #3030/#3031で新`*.tei.md` Reader/旧新意味比較、#267で正式Source受理/再Loadへ接続する。(4) #1174/#2598で現在Host Grant・Receipt／Resource観測・業務Outcomeを段階別に確認する。(5) OKF Attested Computationは初回H2/H3比較条件を変更せず、TAI #514の後続read-only PoCへ置く。

**停止線**：隔離・独立採点を満たした新H2/H3の4Session（探索の限定4Sessionとは別）、実非公開Egress拒否、実Providerの全件閉包、`*.tei.md`実Parser、AI正式Meaning受理・本番Host一往復は**NOT RUN／NOT ACCEPTED**。PoC内A3の自律開発を停止させず、対外不可逆効果のA4と製品側資格を別に評価する。[Draft #3179](https://github.com/hat47x/tei-platform/pull/3179)の標準構造・機能内表現のMASTER_PLAN差分とは独立の節とし、merge前に最新mainとの差分を照合する。

**Design Context→T1新負例の最短経路（2026-11-07更新）**：[AI-first資格設計§6](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)へ、#3262の**用途相対Required Design Context read-only投影**を、AI Test Generatorより先の前提として配置。P-1a＝[main #3264](https://github.com/hat47x/tei-platform/pull/3264)の実CLI 9応答／Subject・Source Index一致／`readByLoader:true`／欠落・異版・未読込の3負例は限定実証済み、**全Data/Function/Business/Rule/Procedure閉包・新Context APIは未実証**。T0＝[main #3266](https://github.com/hat47x/tei-platform/pull/3266)の実Binary pinned Scenario 6 Step／実CLI4負例＋独立義務2負例／局所statement C0相当10.1%（C1未計測）。[main #3268](https://github.com/hat47x/tei-platform/pull/3268)でSource/CLI/Skill等変更時のCI再発火範囲を拡張したが、完全な依存Closureや全要求Coverageではない。

**次の新規検証はT1一件**：公開`sales.order`の「行番号は正」のRuleについて、#3262の実消費Source/Rule/Requirementと必要Context/未取得を読取り専用で照合し、**既存Scenarioが網羅していない行番号0の負例**を最小CLI/AI操作で候補生成する。独立Expected/Holdoutを生成Agentへ渡さず、#1180 Scenarioの実CLI実行と#2514の既知Mutant/独立判別で「提案が新しい欠落を検出した」と示す。P-1bの機械／人間両用の整形、T1の新Candidate実行、C1/NFR Coverageは未実装・未検収。案の競合[Draft #3258/#3259/#3260/#3261](https://github.com/hat47x/tei-platform/pull/3258)を第二のTest/Expected/Runner正本にせず、実装Owner #1131/#1126/#1125/#1180/#2514/#3223/#2811へ帰属させる。

**後続・並行レーン**：[三要素Diagram/CRUD Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)は同じSource/RoleからのProjectionにとどめ、Function文脈の表現型Instance、SHO図解・表示/編集を初回T1のBlockingにしない。[基本機能 #3267](https://github.com/hat47x/tei-platform/issues/3267)のVerified Composition BOM／[Reference Draft #3269](https://github.com/hat47x/tei-platform/pull/3269)のexact-lock／pairwise候補は`qualified:false`の計画・表示段階であり、実Plugin組合せPASSや現Host Grantではない。[Snapshot/Cache #3271](https://github.com/hat47x/tei-platform/issues/3271)も実消費・Input/Artifact・現在資格の後続再利用で、新P0 Hard Gateにしない。既存P0-L実ProviderとP0-M`*.tei.md` Reader、および比較適格H2/H3は相互に独立して並走する。**本追加はDocsのみ、P-1b/T1実Run・BOM実Build/Provider/Host・意味受理はNOT RUN**。

### T1-Aの実走とT1-B盲検・Plugin実Consumerの分岐（2026-11-08）

[AI-first資格・実測対応§7](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)へ、既存テストとPlugin-firstの新しい証拠を追加する。**新たな実装達成を意味するのではなく、実証した範囲だけの受入順補正**である。

- **Design Contextの準備**：[main #3296](https://github.com/hat47x/tei-platform/pull/3296)は実CLI9応答＋合成不足/Tool Reply改変16負例を限定CIで検出し、[main #3299](https://github.com/hat47x/tei-platform/pull/3299)はSource/Plugin Lock実byteの変化後に旧Scenarioを拒否した。しかしContextは`partial`、`qualifiedTestGeneration=false`。外部Harnessの切断Tool Result完全性、実Plugin Loader・Operator実消費、Current Grant、全Requirement/Source閉包は別。
- **T1の実績更新**：[main #3304](https://github.com/hat47x/tei-platform/pull/3304)の旧HEAD専用CIは、既存6 Stepと別の一時第7 Step「行番号=0」を実CLIで判別し、偽Expected/Fixture/旧Pinの3負例を検出。これは**既提案境界のT1-A実走**であり、AIが非公開Holdoutを見ずに未知条件を発見する**T1-BはNOT RUN**。#3262の一Subject・用途限定Design Context→#1180 Scenario→#2514独立Expected/既知Mutant→同Attempt受領の最初の盲検試行を次の実験とし、C0 10.1%局所・C1未計測・NFR未測定をそのまま表示する。外部H2/H3比較とは並列に進め、第二Generator/Runnerを作らない。
- **Plugin-firstの次のConsumer**：[main #3277](https://github.com/hat47x/tei-platform/pull/3277)の開発機能Plugin化要求に従い、[#3292](https://github.com/hat47x/tei-platform/pull/3292)のread-only Inspector一機能で**Selected Providerと実Invoked Provider、CLIとStudioの実Consumer**を照合する。2 SurfaceとProvider×Consumer 2×2は別の検証軸。Workbenchの[main #3287](https://github.com/hat47x/tei-platform/pull/3287)は合成Fixtureまで、実Provider/共通Invokeは未実証。標準業務Role/型・Dataの[main #3308/#3318](https://github.com/hat47x/tei-platform/pull/3318)を後続Domain負例に利用するが、DB実Rowとはみなさない。
- **既存Owner・未統合を尊重**：[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)の旧RQ-035 Workbench／RQ-036 BOM記述（現行要求はRQ-035 Workbench／RQ-036 Model Map／RQ-037 BOM）と2×2 Witnessは既存担当に保持。`qualified:false`を正式BOMへ変えない。[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)のread-only `view -execute`除外案や、closed/unmerged[#3315](https://github.com/hat47x/tei-platform/pull/3315)のWorkspace Identity欄を、現mainに導入済みと扱わない。#3241の設計PR、#3292/#3297の計画Draft、#3303三視点図面Draftは各Ownerの役割が異なり、二重の正本を作らず統合時に最新mainとの差分を比較する。
- **既存Owner・未統合を尊重**：[旧Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)のBOM旧RQ-036/PAS/Gap Audit（現行RQ-037へ再調整する履歴）と2×2 Witnessは既存担当に保持。`qualified:false`を正式BOMへ変えない。[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)のread-only `view -execute`除外案や、closed/unmerged[#3315](https://github.com/hat47x/tei-platform/pull/3315)のWorkspace Identity欄を、現mainに導入済みと扱わない。#3241の設計PR、#3292/#3297の計画Draft、#3303三視点図面Draftは各Ownerの役割が異なり、二重の正本を作らず統合時に最新mainとの差分を比較する。

**残るP0**：T1-Bの初回盲検Candidate、比較適格H2/H3の4Session、旧`*.tei`一Function→実Provider read-only、新`*.tei.md` Reader/実意味比較は**未成立**。既存局所CI Greenを正式Meaning/Authority/EffectやBuild G4適格に昇格させず、ADR-0035のPoC A3自律実装を不必要に止めない。

### 正常JSONでも必要Ruleが欠ける場合の停止条件とTool実効果（2026-11-08）

[AI-first資格・実測対応§8](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)で、**T1-Bへの入力の実資格と、AIに渡すToolの実効果**を二つの独立したP0に具体化した。先行する[main #3296](https://github.com/hat47x/tei-platform/pull/3296)の9件の実CLI応答＋合成欠落/破損16負例は、正常JSON/同Fingerprintのまま必要Rule・Relationだけ欠落する反例を検査しきらない。[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324)は`inspect-validation`を加えた10応答で、`validation.tei` Loader実読取・Rule/Relation Identity/Canonical Attribute/Source Anchor/Definition Roleを比較し、Tool Reply一時コピー8負例を拒否する変更候補。ただし**Draft/CI未確認、Context`partial`/`qualifiedTestGeneration=false`**であり、実Tool配送・Function/Business全Requirement・新AIテストの資格へ昇格しない。

**実Toolの権限・副作用**：[main #3333](https://github.com/hat47x/tei-platform/pull/3333)は`tei.cli.discovery/v0`に`view.writeFlags=[-operation-log,-lifecycle-state]`を互換追加し、実ローカルFile書込みを限定Go試験で検証した。`effect:read-only`はCanonical Meaning非変更であり、File/Network無変更の証明ではない。H2/T1-Bでは許可Command/Flag/実argv・File/Session Log・Audienceを明示し、Holdout/Reviewer-only資料へ**実OS/Harnessで到達拒否できた**場合のみBlind資格を与える（#2509/#2515/TAI #168）。旧Draft #3302の別提案欄は現mainの実Contractではない。

**次の最小Done**：P0-A=#3262/#3324による一Ruleの独立RequiredとActual Source/Rule closure・不足理由の検査（まずDraft exact-head CI）、P0-B=#2509/#2515のHoldout物理隔離/Tool Result全体性・副作用/送出境界、P0-C=#3273/#1180/#2514の**未知Rule Gapからの盲検T1-B候補一件→独立Expected/Mutantの判別**。一件の狭いSourceが成立しても全Workspaceの`qualifiedTestGeneration`は別。#3304の既知行番号0のT1-A成功をT1-Bに転用しない。設計Contextが`partial`なら取得済みScopeの探索に限り、全域PASSを返さない。従来の適格H2/H3比較と、旧`*.tei`実Provider read-only、新`*.tei.md` Parser移行は**別の並行レーン**。

**後続Plugin/Consumer**：[main #3334](https://github.com/hat47x/tei-platform/pull/3334)の税・為替・採番・休日DataのApplication/Studio投影は実装済みだが、行取得/実DB Bind・Wails E2Eではない。[Draft #3292](https://github.com/hat47x/tei-platform/pull/3292)のInspector一機能で選択Plugin/Providerと実Loaded/Invoked Provider、CLI/Studio二Consumer、Current Scope/Independent Expectedを別々に確認。Studio ABAの[Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)は旧UI応答を採用しない2件の局所成功だけで、Hostの旧Effect取消ではない。Verified BOM/Workbench/Cache/OKF/図面はP0-CのHard dependencyにしない。**今回の変更はDocs/Planのみで、新P0-A/B/C、実Provider・正式Owner受理・Host E2E・C1はNOT RUN**。追加Runner/第二Scenario/Registry/Hostなし。

### TT-56の二候補を一本化し、実Capability選択とBuild主体を分離する（2026-11-08）

[AI-first資格・実測対応§9](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)に、**Draftの二つのTT-56実装候補を統合前に一本化する**採否手順を記録した。[#3324](https://github.com/hat47x/tei-platform/pull/3324)は必要な一Ruleの`inspect-validation`/Canonical Attribute/Relation/Source Anchor、[#3326](https://github.com/hat47x/tei-platform/pull/3326)は独立Required 3 Ruleと`source-index` **indexモード**の全件照合を実装候補とする。**両PRの最終固定HEADで、専用Dogfood／Repository Health／依存チェック3系統ともcompleted/success**（#3324 CI #42385606408/#42385606448/#42385606458、#3326 CI #42385726717/#42385726733/#42385726722）。ただし**Draft未マージ・合体HEAD未試験**。#3262を唯一のDesign Context Required/Observed Ownerとし、#3326の独立Required集合＋#3324の特定Rule/Relation/属性/Source Spanを重複なく一つの検証へ照合する。必須でないRuleの省略は正例に残し、重複Caseを二重計数しない。連続するCLI readは原子的Snapshotの証明ではない。

**P0の次の順序**：① #3262でRequired 3 Rule全件Indexと一Rule詳細を実CLI/Source版/返却byteで確認、合同HEAD・main pushは別CI。② #2509/#2515/TAI #168で独立Expectedへの物理アクセス拒否、Tool Reply終端・実argv/Session Log Egressを確認。③ #3273/#1180/#2514でAIが未予告Gapから独立に新Candidateを発見したか**T1-B盲検**を測る。#3304の既知行番号0 T1-A成功を再利用しない。Sourceが`partial`なら可観測Scopeの探索に限り、`qualifiedTestGeneration=false`のまま。不適格なRuleやTool全体を不可視のまま合格としない。

**選択・起動・Buildの実証差**：[main #3337](https://github.com/hat47x/tei-platform/pull/3337)の参照Capability Coverageでは12宣言中6件が**その参照方式では未選択**。後続[Open #3350](https://github.com/hat47x/tei-platform/pull/3350)は既存`SupportsDatabaseProvider`の技術名対応を使って未選択を4件とする改善候補で、現main結果とは異なる。**Declared→Selected Capability→実Plugin Loaded/Invoked→Provider/Consumer適合**は別資格。未選択は必ずしも欠陥でなく、選択済みも実装・権限・BOM合格ではない。[main #3336](https://github.com/hat47x/tei-platform/pull/3336)の子Build駆動記録はInvoking/Invoked executable digest・argv/exit・実Artifactを区別するが`l3Claimed=false`／Source Closure未資格。Plugin-first Inspectorの実Invoke（#3292）とSelf-Build L3（#2831）は後続別検証で、**T1-B開始条件にしない**。

**Tool/標準Dataの横断反例**：[main #3340](https://github.com/hat47x/tei-platform/pull/3340)の`-h` Value/Default とDiscoveryの同一変数化はTool説明の整合改善。実Flag許容・writeFlags #3333・OS/Network送出は別。標準Dataの[main #3343](https://github.com/hat47x/tei-platform/pull/3343)では複数Calendarが一Holiday Dataを共有できるが、Calendar省略の曖昧Readは拒否。**同じData ID／件数でも選択Contextが違えばProviderの実Sliceは別**とする。実DB Row、Studio実Provider、業務完了は未実証。

**新たな実装・試験の上限**：今回の変更は本計画とAI-first設計のDocs更新のみ。合同TT-56、T1-B、比較適格H2/H3、実CLI/Studio共通Provider、Go L3/外部Host/Owner AcceptanceはNOT RUN。新Runner/第二Test/Resolver/Hostを作らず、既存Draftの採否・main追随は個別Ownerが担う。

## 作業領域I：周辺技術との連携と本番運用の境界

### 目的

中核と外部の実行基盤を分ける境界を、実際の利用に耐えられるよう段階的に整備します。

### 対象

- プラグインの配布方法と信頼性
- プラグインが定める機能契約（ISSUE-000080。中核に固定の列挙値を増やさず、記述時、構築時、実行時の範囲を分離する）
- 本番向け提供実装の接続部品
- 永続的な保存機構と高可用性
- 実行状態の観測
- 配備機構との連携
- ソフトウェアの供給経路に関する検証根拠
- 互換性の方針

外部のオーケストレーターやエージェント基盤にある汎用機能を、TEI独自の価値として重複実装することはしません。

## 作業領域間の依存関係

```text
A Product Baseline
      ↓
B Semantic / Knowledge ───┬── C Application / Legacy
                          ├── D Database / Provider
                          └── E Driver
                               ↓
                    F First Target / Gate C
                       ┌───────┴────────┐
                       ↓                ↓
                 G Studio          F Second Target
                       ↕                │
                 H AI / Knowledge ──────┘
                       ↓
                 I Production Boundary
```

すべての作業を順番に終える必要はありません。判定段階Cの後は、編集環境・AI・既存知識の探索と、二つ目の実行先の検証を並行して進められます。

## 2026年10月の技術調査を製品価値につなぐ

直近の調査結果を、新しいAI基盤や判定段階を増やす理由にはしません。「意味の正本を一つに保つ」「特定技術に依存しない」「人が変更を確認できる」「AIを使っても検証結果を再現できる」という製品価値に照らして、既存の作業領域へ割り当てます。

詳細は[TECH_RESEARCH_INTEGRATION_2026-10.md](research/TECH_RESEARCH_INTEGRATION_2026-10.md)を正本とします。

知識の探索とAIの利用については、次の順序で必要な手段を段階的に追加できるか検証します。

~~~text
exact / explicit
   ↓ unresolved only
deterministic inference
   ↓ unresolved only
lightweight associative / statistical candidate generation
   ↓ unresolved only
bounded local cognition
   ↓ unresolved only
local SLM / LLM
   ↓ unresolved only
external LLM
~~~

どの段階でも、候補の品質や確信度を、意味上の適用可能性・受理・書込み権限・実行権限と同一視しません。候補の適格性・検証・受理・権限は、同じ入力から同じ結果を得られる規則で最終確認します。

実行先と実行基盤については、Wails・WebView・ネイティブ実行・Wasmなどの違いを意味の正本へ持ち込みません。実行環境・接続部品・生成した実行物・実行物間の規約・実行設定の違いとして観測します。意味上は同じ操作を、異なる実行物や環境でも追跡できるかを、二つ目の実行先で検証します。

これらの調査結果は、RQ-010・013・015・016・021・024・027〜030に残る不足を減らすために活用します。研究を参照しただけで検証の達成段階を変更することはしません。


### 2026-10-20の国内ローコード調査の反映（追加検証案）

国内製品の公開文書には、設計情報の資産化、生成と手書き処理の分離、AIの編集範囲の限定が既に説明されています。TEIでは、それらを掲げるだけでなく、業務規則・標準部品・実現先の変更を通して案件の知識と処理を保ち、調査・修正・再試験を減らせるかを確かめます。調査資料の読解を、TEIや他製品の実動作の証拠とは扱いません。

| 依存の順序 | 確かめること | 作業領域・既存の入口 | 次へ渡すもの |
| --- | --- | --- | --- |
| 1 | 派生名の衝突、意味の未記入・曖昧な対応を読取り専用で示す | B・D、#2094・#2095・#1120・#1808 | 対象の識別子、根拠、規則の版、未評価の範囲 |
| 2 | 再生成で案件実装を保ち、標準定義更新時の案件修正を比較する | C・G、#2090・#2091・#2016・#1920 | 保全対象、互換性、衝突、取消し可能な範囲 |
| 3 | AI編集の対象と操作を検査し、開き直しても意味を保つ | G・H、#2092・#267・#1935 | 基準の版、差分全体、必要な検証、受理に残る条件 |
| 4 | 能力不足を配備前に示し、AI向けの参照経路を既存の認可と比較する | D・E・F・I、#2096・#2093・#1180・#1174 | 不足と未確認の区別、限定した公開範囲、同等性の比較結果 |
| 5 | 同じ受注変更を更新と引継ぎまで比較する | A・E、#2018 | 人の調査・再適用・再試験・維持の負担と不確かさ |

要求の具体化は[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、担当境界と受入候補は[変更の継続性設計](design/CHANGE_CONTINUITY_DESIGN_2026-10.md)、測定は[生産性ベンチマーク](design/PRODUCTIVITY_BENCHMARK_PROTOCOL_2026-10.md)へ集約します。新しい汎用の受理口や上書き演算は先に作りません。

この表は追加検証の依存順です。現在の主線と優先順を変更せず、既存のOwner Action、Validationの依存、拒否の分類、Hostの権限が未決の場合は、その担当の判断を待つ条件として扱います。参照専用の試作、合成試験、実Hostでの成立は別々に記録し、調査を理由にGateや実装状況を格上げしません。

2026年10月21日には担当参照を#2090〜#2096へ更新しました。10月23日には、mainに統合された生成器契約の設計候補と適用範囲をそろえました。契約案との役割分担、失効時の証拠の扱い、再利用できる参照実装と追加検証は、[継続性設計の4.6節・5.1節](design/CHANGE_CONTINUITY_DESIGN_2026-10.md)で確認します。生産性記録ツールの同梱計画は初回4タスクで、計画の要素数も4件に固定されています。追加の更新タスクを同ツールの対応済みとして計画せず、実施時は対象と記録方式を別に固定します。

### 2026-10-09の追加検証候補（評価待ち）

直近の仕様確認の研究、画面操作の評価、WebAssemblyの接続規約の標準化は、**新しい判定段階や独立した基盤を増やすためではなく、既存の業務上の参照事例をより厳しく検証するため**に利用します。製品価値との関係と調査根拠は、[技術調査の統合方針第9節](research/TECH_RESEARCH_INTEGRATION_2026-10.md)に記します。

- **REF-SPEC-01：仕様の不足を実装前に発見する。** 既存の受注の参照事例を使い、人が欠落・曖昧さ・矛盾・古い前提を発見できるか比較します。関連課題はISSUE-000075・#982・#998・#915・#267です。最初に検証する価値は生成量ではなく、意味を誤って確定させないための変更の確認です。
- **REF-INTERACTION-01：画面操作を業務結果まで確認する。** 表示・操作・権限による拒否・業務上の結果・観測結果・画面の評価を分けて記録します。関連課題は#997・#1000・#1001・#259です。最初のウェブ実行先で得た判定段階Cの結果を遡って変更することはしません。
- **REF-PORTABLE-01：異なる実行物や環境で意味を保つ。** Goのネイティブ実行物やTinyGoのWasmなど、対応可能な方式を小規模な構成で比較します。WASI 0.2／0.3・バイナリ間の規約・実行環境から取り込む機能・依存関係・権限・資源・性能を分けて記録します。関連課題は#259・#943・#973・#976・#1016です。編集環境用のWailsと二つ目の業務用実行先を混同せず、HikeをGoコンパイラとして扱いません。

評価は、まず**業務変更時に役立つREF-SPEC-01**、次に**操作の成立を示すREF-INTERACTION-01**、続いて**可搬性を確かめるREF-PORTABLE-01**の順とします。ただし、これは研究候補の評価順であり、既存課題への着手承認・並行作業・保護された統合手順は変更しません。ISSUE-000075の候補Bでは業務規則の変更が受理されましたが、一連の処理を通した実証は未完了です。REF-SPEC-01はその受理を置き換えず、受理済みの意味と業務手順の記述の違いを読取り専用で示します。

**判定：** 文書への反映は実施済みです。三つの参照実験はいずれも設計案であり、実行・利用者評価・性能測定・要求充足を示す検証根拠の取得は未実施です。小規模な実証で中心価値に関する不確実性を減らせない場合は、方式や作業範囲を広げる前に見送ります。

## 外部の参照モデルを使った横断的な検証

小規模な参照事例だけでTEIの表現力を判断しないため、判定段階Cの後にISSUE-000078を横断的な検証として進めます。利用する外部参照モデルについては、公開の可否と利用条件を個別に確認します。外部モデルの本体や詳細な派生資料を、この公開リポジトリへ含めることはしません。

これは新たな製品判定段階Dではなく、既存の要求をまとめて厳しく検証する取組みです。X-TEAのスキーマや実行基盤に、TEIの意味を決める権限を与えるものではありません。データ・機能と操作・業務・元形式の既存資産・実行基盤と環境・表示形式を一式で移し替えても、意味を失わず説明できるか確認します。

移行時に見つかった不足は、業務手順・変更周期・変更の影響・実行先・意味の拡張などに関する既存課題で扱います。実行基盤や開発用ツールの機能を中核に固定された列挙値として増やさないための境界は、ISSUE-000080で扱います。外部の参照モデルに存在するという理由だけで、中核の基本要素を増やしません。利用条件を確認せずに原モデルを公開リポジトリへ転載することもありません。元資料の照合値・対応付け・失われた意味・未解決事項・TEI独自の試験データを検証根拠として保持します。

作業の目的と公開境界はISSUE-000078で管理します。

## 業務設計の実用性を基準にした範囲の再評価（2026-10-08、評価待ち）

[製品要求の整理#982](https://github.com/hat47x/tei-platform/issues/982)では、データ・機能と操作・業務手順の相互検証、早い段階で実行できる設計、人が理解できる表示を重視し、RQ-001〜033と検討中のRQ-034について第1版で着手する範囲を見直しています。**ここでは製品要求の削除や既存の優先順位の変更を確定しません。**

検討の入口は既存のISSUE-000075とします。実証Aでは、2026年10月5日に合成データを使った開発環境で実行記録を取得しました。候補Bでは2026年10月8日に管理主体が業務規則の変更を受理しましたが、ウェブ画面を含む一連の処理・すべての明細への規則適用・三つの視点からの意味の整合性確認は未完了です。実証Bの詳細と未完了範囲はISSUE-000075を参照します。

汎用的なプラグインの依存解決・構築器の自己構築・AIによる業務実行・複数の実行先など、長期的な能力を否定するものではありません。ただし、それらの完成を最小限の業務実証の前提にする場合は、**どの具体的な不足を解消するか**を明らかにしたうえで採否を判断します。現在の判定段階・各作業の適格性確認・保護された統合条件は変更しません。

## 現在の優先順


**2026-11-11／実ReadとPlugin選択の現在の最短不足を再確定**：main [#3862](https://github.com/hat47x/tei-platform/pull/3862)で`reference/workspace`の`public.orders`用隔離Seedと3/2/0件の独立Expected、main [#3866](https://github.com/hat47x/tei-platform/pull/3866)で型付きM0 Model Mapの局所Source照合が成立した。一方[旧Draft #4076](https://github.com/hat47x/tei-platform/pull/4076)は**別Workspace** `examples/xtea-lowcode`でEnvironment→Database instance→実SQL Executor→Function Data Context→CLI `view -execute`を試みたが、現在は**closed/unmerged**であり、現mainの製品実行経路として扱えない。現在のXTEA SQLは`sales."order"`／`:status`の等価検索で、当該旧Draftの`operator/page-size/offset`等の入力追加はmainに存在しない。両方とも`order_entry.recent_orders`のIDだが**SQL/物理Relation/パラメータ/Fixture/Expectedが別**で、同じ成功票を共有しない。さらに`reference/workspace`側SQLの`:from`は現Query宣言に`input from`がなく、実Binder/型契約未成立。これを[Workbench §23](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)と[独立Source-static回帰](../examples/data-source-workbench/query_route_separation_test.go)へ固定した。

**P0-Read（#2586）**：まず明示した一つのWorkspaceだけで、隔離実PG Schema/Seed→別Reader独立観測→該当QueryのInput宣言/Runtime Bind→実行Statement/params→Raw Rows/Presence/NULL/Scope→Mapping/Grain→Function Data Context→CLI受領の**一Attempt**を確認する。#4076のopt-in実PG試験は環境変数未設定でskipされた記録を実Read成功に足さない。`declared-page-contract`による`coverage=complete`・`emptyJustified`の推定を、RLS/Role/全母集団の独立完全性や業務不存在に拡張しない。`SourceRevision=unknown`はfreshではない。

**P0-Plugin（#1251/#951/#1016）**：[Draft #4202](https://github.com/hat47x/tei-platform/pull/4202)は既存CompositionでのA/B Entry選択、実Goシンボル照合、Host独立Manifest再観測の局所実装を加えたが、SP-04（一貫した偽Entry）・SP-05（同一Manifest/異Source）を検出できないと明示。正式Profile→対象Binaryに存在する実Provider Entry、同一Load Source Origin、現在Grant、CLI/Wails二Consumerの実Callを別列で検収する。選択ID・DB Provider種別・Host登録名・実Entry・返却IdentityをAdapterが上書きして偽一致を作らない。責務は既存[Composition判断 §13](workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、実Artifact/BOM #943/#3267、Source #3524、独立Expected #1180/#2514のOwnerに帰属し、第二Host/Registry/Runnerを作らない。M0・Hook・共有Harness/SEI/SUI/SHO/KONは有用な後続Consumerであり、最初の実Readや正式Provider選択の追加Hard Dependencyにはしない。


**2026-11-12／同名Queryの実Readと派生Consumerの検収を現在の実装へ補正（旧実績の取り消しではない）**：P0-Rは引き続き[#2586](https://github.com/hat47x/tei-platform/issues/2586)が唯一の一Function実Read統合Owner。mainの実Sourceは、**Reference** `reference/workspace/applications/order-entry/queries/recent_orders.tei`（Blob `a570bceddc7d61aae62ac315a886cc9aedcac082`）が`public.orders`・`:from`・5列・日時降順で、**XTEA** `examples/xtea-lowcode/applications/order-entry/queries/recent_orders.tei`（Blob `0bf3d516cc81c55493e45a4b525f6dd69bc6ab83`）が`sales."order"`・`:status`・2列という**同IDの別Source／別SQL／別引数／別Expected**である。Reference側は実SQLに`:from`があるがQuery宣言に`input from`がない。main [#4258](https://github.com/hat47x/tei-platform/pull/4258)のSource-static分離と[#4344](https://github.com/hat47x/tei-platform/pull/4344)の引用修正は実証済みの局所成果だが、旧[#4076](https://github.com/hat47x/tei-platform/pull/4076)のEnvironment実行入口・`internal/functioncontext`は**mainへ未統合**。新たな検証・実装の順は、①Workspace＋Query ID＋SQL原byteと入力宣言を固定、②型付き引数・Alias→実Instance/Providerを実Callerで追跡、③独立Seedと実QueryのRaw Rows/NULL/Scope/0件を確認、④Function Data Context→CLI Consumerの実受領を比較。**最初に欠けた辺だけを既存Driver/Bindingで補う**。現在[PR #4465](https://github.com/hat47x/tei-platform/pull/4465)のApplication DB要件未bind警告は未マージの修正候補で、`validate ok`やwarningの有無をRuntime Bind・Provider実Read成功と解釈しない。

**派生情報は「表示に成功」より「問いに必要な情報を保持したか」で検収する**：[main #4468](https://github.com/hat47x/tei-platform/pull/4468)は`overview`で読取可能／不能の種類を明示し、[main #4469](https://github.com/hat47x/tei-platform/pull/4469)は`trace`の2種類の未解決を`unresolvedScopes`で区別する。[実CLI調査 #3900](https://github.com/hat47x/tei-platform/pull/3900)では、`reference/workspace`の対象行の`unresolved=0`とGraph投影時の`projectionUnresolved=17`が**同時に成立**した。対象行に未解決がないことを、Graph／CRUD／仕様書／AI Contextの依存閉包・全Relation完結へ昇格しない。各Consumerは対象のOccurrence、必要なRelation/依存集合、Source/Model Revision、観測範囲・omitted/unknown/非開示、実受信を既存[#3262](https://github.com/hat47x/tei-platform/issues/3262)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)で用途別に対照し、欠落を空集合や意味不存在としない。

**SHO/SUIへ渡す実装済み範囲を明記**：[merged #4317](https://github.com/hat47x/tei-platform/pull/4317)とSHO [#36](https://github.com/hat47x/sho-composer/pull/36)は、TEI受理Sourceから一要素・一関係を**一Workspace Load**で文書Draftへ投影し、SHOで編集・保存・再開、TEIで再照合する限定経路を実装。SHO側[相互実CLI CI #42995269509](https://github.com/hat47x/sho-composer/actions/runs/42995269509)は当該一往復に限りsuccess。一方、`three-element-diagram`文書種別は入力JSONを投影し**TEI Workspaceを読まない**。文書保存や`frozen`が実Modelの全要素・全Relationの自動受理やSource書戻しを証明するわけではない。[main #4445](https://github.com/hat47x/tei-platform/pull/4445)のSUI向けRecord/Disclosure/Scopeは本文を運ばない**Reference**で、SUIの業務意味をTEIへ直接変換した資格ではない。これらの成果は[#1124](https://github.com/hat47x/tei-platform/issues/1124)/[#2497](https://github.com/hat47x/tei-platform/issues/2497)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expectedへ引き渡し、AI選択の固定・共有Harness Session・現在Grantを混同しない（進行中[Draft #4473](https://github.com/hat47x/tei-platform/pull/4473)/[#4477](https://github.com/hat47x/tei-platform/pull/4477)/[#4478](https://github.com/hat47x/tei-platform/pull/4478)は各別Owner）。**P0の実Read・正式Plugin実CallをSHO/SUI/共有Harnessの実装完了待ちにしない**。第二Model/Graph正本、Reader/Resolver/Host、Test Runner、横断Grant Engineを追加しない。今回の新Product実DB/GUI/外部Harness/Owner意味受理の横断試験はNOT RUN。

**2026-11-12／標準PreludeとImported Provider由来を実Readの必要閉包へ分解（#2620・#3262・#2586の所有範囲を維持）**：main `internal/stdlib/semantic.go`（Blob `b460998679c5e58cc3f637f063783636462bfc3a`）は12個の`std.*` primitive型を**Semantic Coreの種別ではなくGoプロセス内の標準Library Prelude**として構成する。XTEA側`recent_orders.tei`の`input status std.code`を検証するとき、`std.code`がSource Indexに存在しないことだけで「型／意味の不存在」「必須Source未取得」と判断しない。ただしBuiltinで解決できたことは**独立したOwner Meaning受理、Imported Sourceの実消費、SQL `:status`実Bind、Provider実Read**の証拠でもない。現PreludeがCompiled Standardである限り`standard-definition-origin=builtin-prelude`相当の**既存証拠票での分類**と実Build版を対照し、標準型の自己記述Sourceへの置換は[#2620](https://github.com/hat47x/tei-platform/issues/2620)のK0→K3受入へ委譲する。[Draft #4485](https://github.com/hat47x/tei-platform/pull/4485)の自己記述設計は**未統合・未実行**であり、`std.*`の現在の宣言経路が自動的にSource化したとは扱わない。

**外部Sourceの鮮度は、参照している必要閉包だけを判定する**：[merged #3479](https://github.com/hat47x/tei-platform/pull/3479)はImported ProviderのRaw／Manifestだけ変更してもConsumer World/Source Indexが不変となる局所負例を確認した。同一`order_entry.recent_orders`のProduct Readでも、**要求されたProvider/Language Plugin/Importが実選択・実消費された場合**だけ、そのSource Revision/Manifest・Artifact・実InstanceとSource/SQL引数を同じAttemptに照合する。Consumer側World一致をFreshnessや正しいProvider実Invokeへ昇格させない反面、**そのFunctionに不要な別Providerの改訂で無関係なRead候補まで一律失効させない**。必要性がSourceから未判別の場合は対象Claimだけ`unknown`とし、read-onlyのRaw探索・不変のQuery Sourceの事実は保持する。具体的な受入対照は①`std.code`が組込みPreludeで型解決しても`status`のSQL Bind未観測ならRead未資格、②必要ImportのManifest/Raw変更でWorld不変でもRead資格を再評価、③非選択Providerだけの変更では過剰失効させない、の3件（**未実行**）。#3262 Required Context、#1251/#951実選択、#2514独立Expectedへ引き渡し、新Prelude Registry／Resolver／Host／Role Engine／Runner／Hard Gateは追加しない。P0の実Function Provider ReadをこのK1〜K3自己記述移行で待たせない。

**2026-11-12／実Readの一Attempt証拠とGit外保管・CI再実行の境界（#2586の既存検証票を拡張）**：main統合済み[性能・Evidence保管 #4491](https://github.com/hat47x/tei-platform/pull/4491)では、Gitに最小Manifest・独立Expectedを置き、Raw Trace/Rowset等をLocal CAS/NAS/S3の許可された保管先に分離する。これを`order_entry.recent_orders`のP0-Rへ適用する場合、**保管場所と証拠の資格を分離**する。同一AttemptのWorkspace Root/Query Source byte・実Binary/Provider Artifact・選択DB Instance・型付き引数・実Read Scope/Cohort・Result/Field Presence・Consumer受領・独立Expected・Disclosureを既存台帳の参照先へ対応づけ、**必要なRawに実際にアクセス・再検証できたか**を別判定する。Content Digest一致だけで現在Grant、データの完全性、別DB・別Toolchainでの再利用、SourceのA→B→Aの中間履歴を証明しない。Local-onlyは正常な選択肢であり、公開Git・外部HarnessへのRaw SQL/Parameter/Rowset・NAS URLの無条件送出を要求しない。

**試験判定と保存Logを同一Attemptに固定する**：[Draft #4492](https://github.com/hat47x/tei-platform/pull/4492)はCache検証CIで判定用`go test`と証拠保存用`go test ... || true`の**二回実行**が混在し得ることを指摘し、一回の実行を`tee`で保存する修正候補（未統合）である。P0-Rの最初の実PostgreSQL/CLI検証にも同じ注意を適用し、**CI Verdict・保存されたstdout/stderr・Source/Toolchain/Provider版・実DB Target/Attempt**を一回の実行に束縛する。部分/失敗時にも診断Logを残すが、後続の再実行結果で元Attemptの成功・失敗を改ざんしない。最小負例は①同HEADで二回実行し片方だけ失敗、②同Source DigestでDB/Grantだけ変化、③保管Manifestは存在するがRaw Blob欠落／受信Reply不完全、の3つ（既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)のIndependent Expectedに委譲、**新実DB/CI横断試験NOT RUN**）。#4491の保管方式導入や#4492のCI修復を**P0実Readに先行する新Hard Gateにはしない**。新Evidence Registry・Host・Runner・Telemetry Storeは追加しない。

**2026-11-09／Traitの現main資格と受入統合（既存P0の更新）**：[原子的Trait設計§21](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)で重複した旧§18を整理し、main #3540の**静的Inspector S2-reference成立**（通常CLI／Wails Host経路）と**正式ProviderでのS2-product／S3未成立**を一つの基線に統一した。main #3728/#3754の`model catalog.ok`と`validation.ok`の分離、#3761のHook結末語彙の境界拒否（実Plugin/DB効果は未証明）、#3757のEnvironment六Bindingの解決と実行到達の差、#3760の呼び手指定catalogによるself-build in-process計画導出（`qualified:false`／L2-b未認定）を、**Trait Required→選択→実Consumer→独立Expected→現在Grant**の別資格に接続した。次のP0-Aは#951/#1016の正式選択Provider Artifactを通常CLI／Studio双方で`selected B=actual B`へ接続する一辺、P0-Fは#3279の原子`manual-required`欠落検出。TQ-01〜08は#1180/#2514への未実行反証候補。RQ-036、#3267 BOM、#2831 Buildは既存Ownerを維持し、新Registry/Host/Runner/Hard Gateは作らない。

**2026-11-09／Trait実装義務の到達点更新（新規Hard Gateなし）**：[Trait設計§21.5](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)へ、main #3828/#3843の**Hook `uses`→参照Command Caller実Call**と未実行`Uncalled/NotEvaluated`の差、#3802の`produces`候補表示、#2784のDispatch直前期間Guard、#2812の同一Provider Loadに束縛した非公開parsed Source（ただし`FullInputClosure=false`）、#3839の実CLI Build閉包、#3848の`attemptedCount/complete`（試行した分の全件成功のみ・Required集合完全ではない）を接続した。`CallOutcome.Called=true`は境界到達の申告であり、実Provider/Effect/Commitではない。RQ-036のAI向け「現在地→Trait Required→実装・試験所在」は既存#3847／#1124の導線を使用し、第二Registryを作らない。**TFX-01〜07は未実行の受入観点**で、P0-Aの正式Provider実Artifact→CLI/Wails実Invoke、P0-Fの`manual-required`欠落診断を引き続き先行する。実Plugin/DB・現在Grant・BOMを合成参照で代替しない。

**2026-11-10／Inspector S2-referenceのCanonical／Workspace境界をmainへ統合（#1251、#3181、#2586）**：[PR #3735](https://github.com/hat47x/tei-platform/pull/3735)（統合`e25278d59f5b9b6174d92191dcde4b266394a559`）で、`operations.ValidateInspectionProjection`が**要求Subject IDとCanonical top-levelの小文字・単一`id`を厳密一致**させ、異ID・欠落・大小文字別名・重複キー・JSON null/arrayを拒否する。 CLI／StudioのProvider注入境界にも反例を追加。同一Provider・同一Subject IDの二Workspace A/Bを実際にBind／Studio Bridgeで別々に読ませ、AのSourceファイルを移動するとAのWorld/Canonicalは不変だがSourceIndex FPとArtifactPathが変化、Bは不変という局所対照を追加。 PR exact-head `c70761c71e0d79d869a0f91c7a9c6726d6457541`の[Repository Health](https://github.com/hat47x/tei-platform/actions/runs/42578414117)、[Wails Host](https://github.com/hat47x/tei-platform/actions/runs/42578414227)、[Studio参照編集](https://github.com/hat47x/tei-platform/actions/runs/42578414161)、[Windows WebView](https://github.com/hat47x/tei-platform/actions/runs/42578414217)、[実CLI](https://github.com/hat47x/tei-platform/actions/runs/42578414183)はsuccess。 **これはS2-referenceの入力／投影整合とHostが束縛した二Rootの局所観測まで**。悪意あるProviderの異Root虚偽応答、Importのsame-Load raw Origin、正式ProfileからのProvider B→B実Entry、実Build Member・現在Grant・Disclosure・Owner意味受理・外部Tool受領は未検証であり、PR exact-head CIと統合後main CIの資格は別。

**二つの独立したP0とProvider identityの非洗浄原則**：Function一件の実Read（#2586）は`order_entry.recent_orders`について**Function→Query/Relation→`:from`付き実SQL→実PostgreSQL Provider→Rowset→Data Context→Consumer受領**を同一Attemptで実証する。現[Draft #3862](https://github.com/hat47x/tei-platform/pull/3862)はProduct SQLに合わせた隔離Seed／3・2・0行の独立Expectedを準備した**Source-static**成果で、実DB接続・パラメータ束縛・Rowset・Function Contextは未実行。 並行するPlugin S2-product（#1251/#951/#1016）は選択Providerと実Invoke Providerの同一性、別Artifact/Host Grantの資格を実証する。両者を同じ完了票へ合算しない。別系統のmain [#3936](https://github.com/hat47x/tei-platform/pull/3936)／[#3945](https://github.com/hat47x/tei-platform/pull/3945)が**観測Provider ID／配備Executor IDをAdapterが無条件に上書きして同一性拒否を無効化する不具合**を修正した事実を横断負例に採用し、`selected`／`registered`／`returned`／`actually-invoked`のIdentityを互いに上書きして整合を捏造しない。実Providerの真のプロセス／DB／Receiptは独立に追跡する。

**受入の上限**：現行の12種類のInspector Canonicalはtop-levelの小文字`id`を持つが、**その文字列が要求IDと一致することと、要求Kindに属する実Sourceから導出されたことは別**。同じIDはData・Relation・Query等の別Kindでも現れ得る。選択済みProviderのJSONを検査しても、実Source Root・Parser消費byte・Providerの実体・独立Grantを自動認証しない。対照の独立Expectedは既存#1180/#2514を利用する。

**次の一辺は変えない**：既存#951/#1016で選択したProviderとHost登録の**実Artifact Entry**を、通常CLIとStudioの二Consumerで同一Attemptに結び、B選択→A誤呼出しを拒否する。Workspace Source Revision・Manifest・Provider raw、Source Read Root、現在Host Authority、Function→Query→Provider Rowset（#2586のM1b/R0）を各別資格で観測する。C-1/C-2/C-4やKON/SHOを一律必須としない。Independent Expectedは#1180/#2514、Build/BOMは#3267が所有し、第二Resolver・Host・Runnerを増やさない。

**2026-11-09更新：Plugin-first InspectorのS2-referenceをmainへ統合。** [PR #3540](https://github.com/hat47x/tei-platform/pull/3540)（統合`2eeec81c36db6dfe8c4b6f5c464e0280f8facd87`）により、前回の限定S1（main #3381）から、通常`tei inspect` dispatch／通常・生成Studio Host／Wails Bridgeが**共通Contract・静的参照Provider・既存Composition**を使用する経路まで前進した。PR exact-headで全体Health、Wails Host、Windows Desktop WebView、Studio参照編集、Action Surface CLIが成功。実装・証拠の現在地は[docs/status.md](../docs/status.md)を正本とし、[D1 §15](evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)はその実装手順の履歴とする。下記に残る「通常CLI/Studio未配線」「Draft #3381」という文言は**記述当時の旧状態**であり、新しい現行状態へは読み替えない。

**次のP0はS2-product・S3の不足分だけに限定する。** まず#951/#1016の**既存Profile/Provider選択**から、固定された実Build Artifact/実Provider Entryへ束縛し、通常CLI・Studioの別Consumerで`selected B → actual B`と`selected B → actual A`（出力同一でもFAIL）を独立Expectedで再検証する。続いてProvider Source raw/Manifest・選択Ruleの変更、現在Host Grant、Required Source/Role/Disclosure、2×2 Verified BOMと実Build memberを各別資格として照合する。Host実行ファイルから読めたSHA-256とPlugin実Link/署名・本番認可を混同しない。Trait Required/Hookは実装義務と実経路でのみ資格化し、Graph描画やAI統計の自己申告を実呼出しに昇格しない。CarrierのC-1/C-2、外部Host C-4は用途別に選択可能なまま維持し、新Resolver・共通常駐Host・第二Authority・新Hard Gateを増やさない。実験のOwnerは#1251/#951/#1016/#1824、独立Expectedは#1180/#2514、BOMは#3267。KON/SHO/SUIを通常のTEI読込みに必須の実行依存にしない。

1. （完了、2026-09-19）判定段階Bの最終的な実環境での検証根拠を、同じ版（R = `e5231839`）についてそろえた
2. （完了、2026-09-19）引継ぎの基準`B`と対象版`R`を固定した
3. （完了、2026-09-19）最初のWeb向け実行先をビルド版`T1`（T = `0ba87a4e`）として実装し、判定段階Cの検証根拠を得て`Ready`と判定した
4. 判定段階Cの`Ready`後に継続する既存の検証レーン（現在地。新規実装は上記の外部要件机上評価→基盤→CUI・標準プラグイン並行→ヘッドレス一気通貫→GUIの順序を優先する。各課題の状態は、その課題の`状態`欄を正本とする）
   - `ISSUE-000046`：編集環境での最小限の記述・変更・受理の流れ
   - `ISSUE-000047`：既存知識を優先する意味情報の探索。厳密・決定的な方法を基本とし、必要に応じて軽量な認知機構からモデルへ段階的に進む
   - `ISSUE-000027`：AIによる提案と受理。手段によらず最小限の意味変更を共通の受理境界で扱う
   - `ISSUE-000026`：二つ目の実行先による検証。実行先の違いだけでなく、可搬な実行物と実行環境の違いも比較する
   - `ISSUE-000053`：AIが開発を再開し、業務知識を利用するための仕組み。契約、実装、設定、現在の権限、観測結果を再現可能にする
   - `ISSUE-000078`：外部の参照モデル全体を移し替え、表現力、失われる意味、暗黙の前提を横断的に検証する
5. 二つ目の実行先、定義の記述、AI、参照モデルの移行で得た検証根拠を相互に照合し、実行先からの独立性、記述方法、意味の拡張の境界を再評価する
6. 本番運用と外部技術との連携の境界を、必要な検証根拠とともに広げる

判定段階BとCの`Ready`判定と過去の検証根拠は、それぞれの対象版に固定されています。現在は、判定段階Cの後に進める定義の記述・知識の活用・AI・二つ目の実行先の作業を重視します。以前の判定対象版で得た根拠を、現在の作業に対する検証根拠として無条件に引き継ぐことはしません。

### 2026-11-08 一機能で検収する横断の最短経路（既存Ownerを横につなぐ実装順）

判定段階B/Cの確定記録とマイルストーン5〜7の並行方針は維持する。そのうえで、直近の設計・実装・他製品連携の議論を、**一つの機能を通して最初の未接続箇所を特定する順序**へ収束させる。詳細契約・試験の正本をここに複製せず、各Issueを担当とする。本節は優先する**検証計画**であり、新しい製品Gate、Runner、Parser、Plugin Resolver、Host、意味の承認機構を設ける判断ではない。

最初の対象候補は`order_entry.recent_orders`と関係する`sales.order`の一改訂とし、**同一Workspace名・Subject名だけで各出典や実Providerを接合しない**。Source/外部解析/設計/実Read/表示/Actionは、依存する実Revision・Scope・Attemptで比較する。Sourceを読み込めない場合でも、確認済みの局所情報は保持し、全体完了・不存在・実行許可は保留する。

| 順序・同時実行性 | 次に実証する「一つの辺」 | 判定・引渡し先 |
| --- | --- | --- |
| **P0-D**（他P0と並行） | 管理対象ファイルと、実Loader/Parserが読んだSourceを区別し、Data/Function/Business/Ruleの必要閉包とSource位置を返す。通常Markdown、正式`*.tei.md`候補、従来Source、外部逆解析Rawの資格を混同しない | [#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#3262](https://github.com/hat47x/tei-platform/issues/3262)／[設計取込の後継Draft #3483](https://github.com/hat47x/tei-platform/pull/3483)（旧#3355は歴史的Draft）。main [#3354](https://github.com/hat47x/tei-platform/pull/3354)の11実CLI・32限定判別を基線に、[main #3482](https://github.com/hat47x/tei-platform/pull/3482)では`sales.order.submit`の3 Rule・Capability/Application/Sourceの限定閉包について13実CLI・42 Consumer対照が専用CIで成功。Function実Invoke/実Provider Read・全Requiredは未成立 |
| **P0-M** と **P0-R**（互いを待たない） | **M**：実Source→Data/Function/Business三視点とCRUD→元Relation/CellのSource locator。**R**：`order_entry.recent_orders`を初回対象に、**Pattern→Query Source／SQL raw→実Provider選択→実SQL+args→物理Rowset／Semantic Mapper→Function Data Context**の各辺を、独立Expectedと同一Attempt/Scopeで検収。`sales.order.submit`の宣言・Validation成功をBrowse実Readと合算しない。Graph表示・別DomainのExecutor成功・fake SQL Connectorを実PostgreSQL取得としない | M=[#3263 Draft](https://github.com/hat47x/tei-platform/pull/3263)／[#1124](https://github.com/hat47x/tei-platform/issues/1124)。全容Map要求は[main #3505（RQ-036）](https://github.com/hat47x/tei-platform/pull/3505)で採用済み、既存Worldからのread-only typed M0は[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)で検証中（`partial-projection`であり実三視点/CRUD・State/Hook・Consumer全体の完成ではない）。R=[#2586 HJ-09](https://github.com/hat47x/tei-platform/issues/2586)／[#2506](https://github.com/hat47x/tei-platform/issues/2506)。[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)は**別Domain Native Case**の既存Executor/Mapper＋instrumented Connectorの局所Go Testが成功したが、隔離PG試験は**SKIP**。**P0-C**で同じSource cohort・Scope・Read時点・Consumerが揃うときだけ接合 |
| **P1-P**（P0-Dと並行試行可） | 開発ツールのread-only Inspector一操作を、CLIとStudioで **Contract→Provider選択→実ロード→実Handler Invoke→結果**まで共通契約でつなぐ。選択B／実呼出Aは結果が同じでも不適格。Commandのread-only表示と実argvのFilesystem効果は別に測る | [#3277 merged](https://github.com/hat47x/tei-platform/pull/3277)は設計・要求の基線。[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)が実装・適合のOwner。[main #3381](https://github.com/hat47x/tei-platform/pull/3381)は型付きInspector注入・Host選択Go呼出を限定実装したが、通常CLI Dispatcherと実Studio Bridgeが同じ選択済みProviderを使う製品S2は未実証（[main #3531](https://github.com/hat47x/tei-platform/pull/3531)）。認証方式の選択・実Principal・Identity Link・Session・業務認可を分ける試験は新規[#3412](https://github.com/hat47x/tei-platform/issues/3412)および既存#1043/#1174が担当し、業務Roleや現在Grantへの自動昇格は禁止 |
| **P1-T**（必要な設計が得られた範囲） | 設計入力の**D**（必要Source/Role/Ruleを実取得）・Tool/Consumer受領の**T**（Replyの完全受信）・生成候補の**G**（独立Expected/未知Rule/Mutant判別）を別評価。既知T1-A合格から未知T1-BやC1を認定しない | [#3262](https://github.com/hat47x/tei-platform/issues/3262)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)／[#1180](https://github.com/hat47x/tei-platform/issues/1180)。main [#3477](https://github.com/hat47x/tei-platform/pull/3477)の**同一Workspace** Source-only失効試験と、外部Import Provider原文・Manifestの改訂がConsumerの指紋に反映されない[main #3479](https://github.com/hat47x/tei-platform/pull/3479)の限定Go反例は別資格として扱う。同一Nested Provider Loadのprivate原文証拠は[Draft #3524](https://github.com/hat47x/tei-platform/pull/3524)で前方移植中であり、実Consumerの完全なSource引用資格を自動回復しない |
| **P0-D**（他P0と並行） | 管理対象ファイルと、実Loader/Parserが読んだSourceを区別し、Data/Function/Business/Ruleの必要閉包とSource位置を返す。通常Markdown、正式`*.tei.md`候補、従来Source、外部逆解析Rawの資格を混同しない | [#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#3262](https://github.com/hat47x/tei-platform/issues/3262)／[設計取込後継Draft #3483](https://github.com/hat47x/tei-platform/pull/3483)。main [#3482](https://github.com/hat47x/tei-platform/pull/3482)は13実CLI応答・42限定判別でCapability／Application／Sourceの一部Requiredを実照合済み。ただし全Source閉包・Function実Provider呼出しではない |
| **P0-M** と **P0-R**（互いを待たない） | **M**：実Source→Data/Function/Business三視点とCRUD→元Relation/CellのSource locator。**R**：同一Product Function→実Query/args→実Provider Rowset→Function Context。Graph表示や別Reference SQLの成功を実Product Readとしない | M=[#3263 Draft](https://github.com/hat47x/tei-platform/pull/3263)／[#1124](https://github.com/hat47x/tei-platform/issues/1124)、R=[#2586](https://github.com/hat47x/tei-platform/issues/2586)。**P0-C**で同じSource cohort・Scope・Read時点・Consumerのときだけ接合する |
| **P1-P**（P0-Dと並行試行可） | 開発ツールのread-only Inspector一操作を、CLIとStudioで **Contract→Provider選択→実ロード→実Handler Invoke→結果**まで共通契約でつなぐ。選択B／実呼出Aは結果が同じでも不適格。Commandのread-only表示と実argvのFilesystem効果は別に測る | [#3277 merged](https://github.com/hat47x/tei-platform/pull/3277)は設計・要求の基線。[CLI入口Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)／[Studio入口Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)は型付きService Port・誤応答・欠落時fallback拒否の候補であり、**実Compositionのexact Provider Binding／両Consumer実Invokeは未接続**。[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)が実装・適合のOwner。認証方式・Principal・Identity Link・Session・業務認可は[#3412](https://github.com/hat47x/tei-platform/issues/3412)と#1043/#1174で別検証し、現在Grantへ自動昇格しない |
| **P1-T**（必要な設計が得られた範囲） | 設計入力の**D**（必要Source/Role/Ruleを実取得）・Tool/Consumer受領の**T**（Replyの完全受信）・生成候補の**G**（独立Expected/未知Rule/Mutant判別）を別評価。既知T1-A合格から未知T1-BやC1を認定しない | [#3262](https://github.com/hat47x/tei-platform/issues/3262)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)／[#1180](https://github.com/hat47x/tei-platform/issues/1180)。main [#3477](https://github.com/hat47x/tei-platform/pull/3477)の**Consumer管理Source-only**失効と、[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)の**別Import Provider Raw／Manifest-only**改訂（ConsumerのWorld/Index二指紋が不変）を区別。Provider実原文・ManifestのRevisionは用途相対に独立照合し、別Load時点を同一Snapshotと扱わない |
| **P1-C**（上記の結果を用途別に渡す） | 既存資産逆解析からの**観測事実→構造候補→Ownerが受理した意味→正式Authoring**を分離。SHOは出典付き文書のRead/候補Patch/正式Save・再Readを段階確認する。外部認知・意思決定へ渡すときも、局所即応・意思決定Intent・現在許可されたAction・実Effect/Receiptを別にする | [REA #3394](https://github.com/hat47x/tei-platform/issues/3394)／[TEI-EKI設計Draft #2744](https://github.com/hat47x/tei-platform/pull/2744)／[SHO計画 #2](https://github.com/hat47x/sho-composer/issues/2)。SHOの限定Reader/Plannerに加え、Working Copy保存・別Open [main #25](https://github.com/hat47x/sho-composer/pull/25)、公開TEI原文Reader [main #27](https://github.com/hat47x/sho-composer/pull/27)、Save As相対リンク差分preflight [main #28](https://github.com/hat47x/sho-composer/pull/28)、Editor UTF-16選択→Raw UTF-8位置写像 [main #29](https://github.com/hat47x/sho-composer/pull/29)が限定試験として統合済み。ただし正式Source Writer／実Editor Save/reopen／TEI Meaning受理・現在Disclosureは別資格 |
| **P2-B**（P1-Pの実呼出し成立後） | 実Plugin 2版×Consumer 2版の**4つのexact tuple**を、実Build・Invoke・独立Expectedでそれぞれ評価。PASS/FAIL/SKIP/NOT RUNを保持し、Current Host Grant・実DB Effect・Consumer開示は追加資格として検査する | [Verified BOM #3267](https://github.com/hat47x/tei-platform/issues/3267)／#951／#2514。[Draft #3379](https://github.com/hat47x/tei-platform/pull/3379)のCapability判定規則変更による旧Evidence失効もProvider版×Consumer版とは別の前提。候補一覧・静的Version一致・2件の試験・限定TEI Self-build L3を「互換な4件」としない |

**テスト生成のP0実証との接合（2026-11-08更新）**： [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のHEAD `e8745c6c`は[専用dogfood CI #42465435223](https://github.com/hat47x/tei-platform/actions/runs/42465435223)、全体Health、依存確認がsuccess。既存13件の実CLI read-only応答、42件の限定Context対照、6実Pinned Scenario Step、**9つのtest-only受領対照**を同一Runで検証した。独立したStep→Operation期待値が、ExpectedとReceiptを同時に誤変更するTT-154を拒否する。これは`test-ability`の**局所正証人**であり、正式Function実Invoke・Data Context/実DB Read・外部完全受領や盲検T1-Bの証拠ではない。詳細Expected/Caseは[#2514](https://github.com/hat47x/tei-platform/issues/2514)、Scenarioは[#1180](https://github.com/hat47x/tei-platform/issues/1180)、受入票は[既存テスト基本設計§35](https://github.com/hat47x/tei-platform/pull/3223)を参照し、このMasterには複製しない。[main #3502](https://github.com/hat47x/tei-platform/pull/3502)のTrait/Provider二Consumer設計や、[main #3505](https://github.com/hat47x/tei-platform/pull/3505)で採用済みの全体Map/RQ-036要求も、実Entry/Rowset・現在Grant・全Domainの不存在証明を代替しない。Mapを表示する場合は**観測済・未取得・部分取得・非公開・未実行**の状態を対象Scopeごとに明示し、表示カスタマイズで実Coverageや権限の判定を変更しない。Source-only/Manifest-only失効の[PR #3479](https://github.com/hat47x/tei-platform/pull/3479)は書式修正後HEADのCIがsuccessだが、外部Providerの原文をConsumerが実消費した同一Attempt/Source閉包は依然別資格である。

**次に通す実製品の最小3 Probe（2026-11-08 現行mainとの差分）**：詳細手順・Independent Expectedは既存Ownerの#1180/#2514へ置き、本節は順序と停止条件だけを示す。旧局所CIのPASSを集計して全体PASSを作らない。

| Probe | 最小の正常/負例と、最初に失敗した場合の停止箇所 | 次の実装を所有する場所 |
| --- | --- | --- |
| **D1-S2／実Inspector** | 同じScope/Contractに対するProvider Bの選択から通常`tei inspect`と実Studio Bridgeの**両Consumerのactual B Invoke**を記録し、B selected→A invoked、Host登録解除/版ずれを拒否。同じJSON・注入Helper Go Testだけでは不可 | #1251/#1824。main #3381/#3531のS0/S1を再実装せず、最初に途切れる**通常製品Dispatch→実Provider**を測る |
| **M0→M1／モデルの現在地** | 同じ`workspace.Load`とSource Indexに由来するData/Function/Business・Relationのtyped IDとSource位置を提示し、同名別ID・多重Edge・未知Event/Stateは欠落診断。M0成功を全容Map/CRUD・可逆編集・Generator成否としない | 採用済RQ-036・#1124とDraft #3518/#3263。まずread-onlyの`partial-projection`を実Consumerで受け、実Loader→Graph/CRUDの未接続辺を特定 |
| **HJ-09／Product実Read** | `order_entry.recent_orders`のPattern→Query raw/args→選択・実呼出Provider→実SQL/Rowset→Mapper→Function Contextを同じAttemptで照合。Rowset 0件/partial、Provider B選択→A実行、接続失敗を正常な不存在としない。物理PostgreSQLが未実行なら`NOT RUN` | #2586/#2506。別Domain Go ExecutorのPASSやInstrumented/Fake Adapterを本Functionの物理Readとしない |

この3 Probeは**D1-S2とHJ-09を並行優先、M0/M1をread-onlyで並行**させる。どれか一つが不適格でも確認済み局所Claimは残し、未取得・実行不能を空集合／不存在／権限ありへ変換しない。共通APIのTransport同値、Plugin Selection/actual Invoke、Graph Projection、DB Read、外部Reference currentnessは別の資格として照合する。Lifecycle/Hook Graph（#3503、[Draft #3507](https://github.com/hat47x/tei-platform/pull/3507)）、AI性能観測[main #3520](https://github.com/hat47x/tei-platform/pull/3520)、完全Headless B2C #3509、SHO/KON連携[Draft #3528](https://github.com/hat47x/tei-platform/pull/3528)は異質な**後続/並行Consumer試験**であり、この一Function実証に新しい常駐HostやCore依存を追加しない。

**要求IDの衝突を明示**：全体MapのRQ-036はmain [#3505](https://github.com/hat47x/tei-platform/pull/3505)で採用済み。[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)が過去に用いた「RQ-036=Verified BOM」は現mainの要求番号と衝突している。BOM要件の採用・再採番は#3267/#1119のOwnerへ委譲し、旧Draftの要件文で現RQ-036を置換しない。2×2適合の実証は引き続き未成立である。
**製品要求とBOMの受入同期（2026-11-08／[RQ-037](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-037-plugincontractの検証済み組合せを構成bomとして再利用できること)）**：mainに存在するRQ-036は**全容Model Map／標準・カスタムFlow**の要求であり、Plugin構成BOMではない。BOMは独立した**RQ-037**として既存[#3267](https://github.com/hat47x/tei-platform/issues/3267)へ帰属させる。Primary PAS-06、PAS-02/04/01/05補助、Product GapはDesign Only。BOM Draft #3297は未統合で旧RQ-036を重複定義しており、mainに取り込む場合は要求番号・PAS・Gap Audit・Design Indexとgo build差分を現基線へ再調整することを**Hard stop**とする。図表RQ-036をBOM用へ改名・上書きしない。

**BOMのS2→S3へ渡す二つの未達**：main統合済み[#3381](https://github.com/hat47x/tei-platform/pull/3381)はCompositionを選択したProvider Bの局所Go実呼出とCLI Helper/Studio Host API二経路の限定証拠を持つが、通常`tei inspect` Dispatchと実Wails Bridgeの製品S2ではない。加えて、[#3490](https://github.com/hat47x/tei-platform/pull/3490)のCapability Rule鮮度、[#3477](https://github.com/hat47x/tei-platform/pull/3477)のConsumer Source-only、[#3479](https://github.com/hat47x/tei-platform/pull/3479)のImported Provider変更検査は、各Producerの歴史的証拠を現在へ再使用する際の必要条件に過ぎず、実Version 2×2の4 Attemptを検証したことにはならない。最初はS2一機能で**実選択B→通常CLI実Call B／実Studio Bridge実Call B**と誤Aへのfallback拒否を検証し、S3では実Build/Source closure/Toolchain/Target/Independent Expected/Attemptを対象ごとに照合する。主な受入負例は[既存PAS-06のRQ-037節](UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)、詳細の唯一の実装Ownerは#3267/#1251/#1180/#2514。

**RQ-037 Verified Composition BOMの追跡修復（2026-11-09）**：製品RQ-036の正本は**全容Model Map／Navigation**である。PAS/Gap Auditに後から混入した「RQ-036=BOM」を修正し、BOMを**RQ-037、主PAS-06、Gap Tier=Design Only**へ区分する（[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、[#3267](https://github.com/hat47x/tei-platform/issues/3267)）。この整理は旧BOM Draft #3297をそのままmainへ統合することではない。限定Go Referenceは[#3541](https://github.com/hat47x/tei-platform/pull/3541)でexact-head Repo Health/依存確認が成功したが、最新mainとの差分を再調整する必要があり`qualified:false`である。

**次の実CallとBuildの受入境界**：通常`tei inspect`と実Wails Studio Hostへの配線は[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)の**S2-reference**として実装候補があるが、固定コンパイル済みInspector ProviderのHost側Bindingや実行ファイルdigestは、別Providerへ切替可能な製品S2・実Link Closure・真正なConformanceの証拠ではない。[main #3583](https://github.com/hat47x/tei-platform/pull/3583)による既存実行入口の再照合ではC-1 `routing.Router`は配備Command、C-2 `servicehost`はService Requestと異なるため、両者を無条件に相互排他的なResolver候補としない。外部Hostへ委譲するC-4も許容し、実行Receiptの未取得は`delegated/not-observed`のまま留める。S2の実Provider選択／実Consumer受領をまず確定し、BOM 2×2の4ケースで各Target/Artifact/Independent Expected/Attemptを実照合する前にS3/Qualifiedへ上げない。

**テスト生成のP0実証との接合（2026-11-08更新）**： [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のHEAD `e8745c6c`は[専用dogfood CI #42465435223](https://github.com/hat47x/tei-platform/actions/runs/42465435223)、全体Health、依存確認がsuccess。既存13件の実CLI read-only応答、42件の限定Context対照、6実Pinned Scenario Step、**9つのtest-only受領対照**を同一Runで検証した。独立したStep→Operation期待値が、ExpectedとReceiptを同時に誤変更するTT-154を拒否する。これは`test-ability`の**局所正証人**であり、正式Function実Invoke・Data Context/実DB Read・外部完全受領や盲検T1-Bの証拠ではない。詳細Expected/Caseは[#2514](https://github.com/hat47x/tei-platform/issues/2514)、Scenarioは[#1180](https://github.com/hat47x/tei-platform/issues/1180)、受入票は[既存テスト基本設計§35](https://github.com/hat47x/tei-platform/pull/3223)を参照し、このMasterには複製しない。[main #3502](https://github.com/hat47x/tei-platform/pull/3502)のTrait/Provider二Consumer設計や、[Draft #3505](https://github.com/hat47x/tei-platform/pull/3505)の全体Map/RQ-036候補も、実Entry/Rowset・現在Grant・全Domainの不存在証明を代替しない。Mapを表示する場合は**観測済・未取得・部分取得・非公開・未実行**の状態を対象Scopeごとに明示し、表示カスタマイズで実Coverageや権限の判定を変更しない。Source-only/Manifest-only失効の[PR #3479](https://github.com/hat47x/tei-platform/pull/3479)は書式修正後HEADのCIがsuccessだが、外部Providerの原文をConsumerが実消費した同一Attempt/Source閉包は依然別資格である。

**一改訂の引渡し票**には、少なくとも対象Subject・使用目的・Source/外部参照の実Revision・World/Source Indexの比較範囲・Requiredの取得/未取得・Meaning受理状態・選択したContract/Providerとactual Invoke・Target/Artifact・Attempt・Consumer受領・独立Expected・現時点の権限/Effectを、**観測できた列だけ**記録する。これは[#1163](https://github.com/hat47x/tei-platform/issues/1163)の証拠と[#3262](https://github.com/hat47x/tei-platform/issues/3262)の設計Context等を束ねる表示・検収の観点であり、第二の台帳や共通Schemaではない。
**一改訂の引渡し票**には、少なくとも対象Subject・使用目的・Consumer側Source/外部Import Providerの**RawとManifestそれぞれの観測Revision（未取得も明示）**・World/Source Indexの比較範囲・Capability選択の判定規則指紋・Requiredの取得/未取得・Meaning受理状態・選択したContract/Providerとactual Invoke・Target/Artifact・Attempt・Consumer受領・独立Expected・現時点の権限/Effectを、**観測できた列だけ**記録する。これは[#1163](https://github.com/hat47x/tei-platform/issues/1163)の証拠と[#3262](https://github.com/hat47x/tei-platform/issues/3262)の設計Context等を束ねる表示・検収の観点であり、第二の台帳や共通Schemaではない。

**2026-11-08の判定補正**：同じ`sales.order`に対し、①Consumer Source Indexだけ変わる改訂（main #3477）、②Import Provider Raw/Manifestだけ変わりConsumer World・Source Indexが変わらない改訂（Draft #3479）、③Capability選択ルールだけ変わる改訂（main #3350、Draft #3379の旧Evidence拒否）の三つは、**同一の失効イベントではない**。観測した入力閉包・判定規則・実Targetに応じて必要な再検査を選ぶ。Raw変更で全生成物を常に失効させる／World同一なら何でも再利用する、の両極は採らない。G4には[main #3440](https://github.com/hat47x/tei-platform/pull/3440)のTEI CLI自身の限定stage0→1→2実測があるが、供給信頼・独立二試行・既定Source Closure・Plugin/BOM適合は引き続き未資格。外部データ基盤KON Dataspaceはこの証拠・Meaning/Host Authorityの第二正本にも、D1の必須Runtime依存にも置かない。

最初に判定すべき停止条件は、(a)実Loaderに届かない設計、(b)同一Workspaceの指紋では見えない外部Provider改訂、(c)図・CRUDから実Readに到達しない辺、(d)選択Providerと実呼出し先の不一致、(e)Tool正常終了だがConsumer受領欠落、(f)認証済み主体を業務権限へ誤写像、(g)未実行のBOM組合せをPASSとする誤判定、の七つとする。新たな抽象化より先に各Ownerの既存Scenario/独立Expectedで**一対の正常・負例**を実行し、最初に失敗した辺と次の必要なProbeを示す。古い確定Gate B/C、合成テスト、Draftの資料を現在の製品E2Eへ転用しない。

#### 共通アプリケーションAPIの採用と既存Harness比較の入口（2026-11-08、ADR-0048／新Host不要）

**採用済みの設計判断**：[ADR-0048](adr/0048-shared-application-api-and-transport-adapters.md)により、CLI・MCP・HTTP/API・Web・Desktopから原則として**同じ型付きアプリケーション操作**を利用する。これはHTTPサーバーを常駐させる判断ではなく、localはプロセス内、remoteは明示した接続先で呼ぶ。開発者向けCLI、Editor/Studio、外部Harnessは同じ意味の操作を使うが、HostのWorkspace解決・開示・認可・副作用をこのAPIの内部に暗黙統合しない。現段階で全入口の同値性が実装・検証されたという意味ではない。

既存[#3419](https://github.com/hat47x/tei-platform/issues/3419)／[#3420](https://github.com/hat47x/tei-platform/issues/3420)のCAPI-1/2では、最初の4操作を **catalog / inspect / trace / validate** に限定する。特に`validate`で、呼出しの正常終了、意味Validationの合否、warning、Source root不在、missing/unknown、`failure.code`とsource範囲を型付きで区別し、CLI v0 wireは互換投影で保持する。かつて存在したCLI/MCPの`validate`成功・失敗の差は[main #3458](https://github.com/hat47x/tei-platform/pull/3458)で**正常・読込失敗・重複意味宣言の限定ケースについて実CLIと実MCP Adapterの投影一致**が確認された。残るcatalog/inspect/traceの完全同値、未知欄/null/上限、実Host認可・外部Harnessの配送は未資格のまま残す。CAPI-3[#3421](https://github.com/hat47x/tei-platform/issues/3421)のHTTP/StudioとCAPI-4[#3422](https://github.com/hat47x/tei-platform/issues/3422)のProposal／受理／実行はその後の別資格とし、read-onlyだけで現在Grant/Effectを許可しない。Plugin側[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)の選択→実Handler呼出しとも一つに潰さない。
既存[#3419](https://github.com/hat47x/tei-platform/issues/3419)／[#3420](https://github.com/hat47x/tei-platform/issues/3420)のCAPI-1/2では、最初の4操作を **catalog / inspect / trace / validate** に限定する。特に`validate`で、呼出しの正常終了、意味Validationの合否、warning、Source root不在、missing/unknown、`failure.code`とsource範囲を型付きで区別し、CLI v0 wireは互換投影で保持する。MCP参照Adapterはmain [#3403](https://github.com/hat47x/tei-platform/pull/3403)で4読取操作の限定値一致を観測し、その後main [#3458](https://github.com/hat47x/tei-platform/pull/3458)で**validateのCLI/MCP応答の限定的な形・診断・成否の差**を修正・試験済み。ただし全4操作の完全Body同値、外部Tool受領終端・実Host Grant・Plugin実Invokeまでは未資格。CAPI-3[#3421](https://github.com/hat47x/tei-platform/issues/3421)のHTTP/StudioとCAPI-4[#3422](https://github.com/hat47x/tei-platform/issues/3422)のProposal／受理／実行はその後の別資格とし、read-onlyだけで現在Grant/Effectを許可しない。Plugin側[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)の選択→実Handler呼出しとも一つに潰さない。

**独自AI Harnessは当面、実装決定ではなく追加価値の比較対象**とする。判別したい問いは「TEIの製品操作・Plugin・Skillで提供できない、再現可能な価値が独自Harnessの追加責務にあるか」である。固定Fixture/同一Source・Task・モデル/予算・Tool権限・開示条件を揃え、次を別々に評価する。

| 対照 | 実行する範囲 | 対象とする価値・比較不可の範囲 |
| --- | --- | --- |
| **B0** 決定的な共通API/CLI直接操作 | 4 read-only操作の取得、Source/診断/版/Requiredの確認 | データ取得・判別の基線。未知Ruleの自律生成は対象外であり、その非生成を「Harness品質不足」と採点しない |
| **B1** 既存外部Harness＋同じCLI/Tool（Skillなし／あり） | 独立Sessionで同じD（設計入力）・T（Tool受領）を検証し、G（未知Ruleのテスト候補生成）は別に盲検採点 | 既存Skillの純増分、Tool呼出回数、Token、待ち時間、コスト、誤肯定／誤不存在、不要操作を測定。単なるSkillロードやTool Exit0を増分成果にしない |
| **B2** 既存Harnessの薄いTool/Plugin Adapter（必要時だけ） | B1で反復して確認された一つの不足（例：型付き失敗/受領境界）が共通API＋薄いAdapterで解決するか対照 | Adapterで足りれば独自Agent Runtimeは作らない。Model/Providerを追加した改善と専用Harnessによる改善を混ぜない |

既存[DSH #2509](https://github.com/hat47x/tei-platform/issues/2509)のH2/H3では実Tool読取とSkill本文の限定一致を観測しているが、`model catalog`の各8,192-byte切断、56 Tool Request対55 Result、stderr付着、再利用Sessionの混入により**独立比較適格は0/4**。その過去4件を再採点して価値証明に転用せず、[#2515](https://github.com/hat47x/tei-platform/issues/2515)が持つ実OS/Network/Log送出と、[#2514](https://github.com/hat47x/tei-platform/issues/2514)が持つ独立Expected／Mutantを満たす**新しい独立4 Session**を優先する。Dを取得できてもTが途中切断なら、Gの成否と推定ROIを`not-qualified`にする。

最初の比較Taskは、P0-Dの一Function・3 Required RuleのContextと、T1-Bの**未提示Rule由来の新規負例を1件選択する能力**を別々の問いに分割する。Harnessに独立ExpectedやBlind Mutantを渡さず、旧Source改訂、欠落Relation、0件/partial Rowset、Selected Provider Bと実Invocation Aの取り違えを既存Scenarioへ委譲する。結果票には成否だけでなく、対象Run/Sourceのbyte・契約・actual Tool/Provider・応答終端・独立採点、実隔離と送出、使用Token/時間/費用、再試行を残し、任意の過大主張を拒否する。Domainを一つ追加しても差が残るか確認し、用途特化が必要なら汎用Harnessではなく**Domain Plugin/Skill/Policy Adapter**で足りる可能性を先に評価する。

**採否の停止線**：B1で既存機構の不足が独立に再現され、B2でも解消しない明確な機能差があり、少なくとも異なる2種類のTask/Domainで正確性・再現性・費用/運用負担・実効隔離の改善を確認するまでは、独自共通Harnessや新常駐Agent RuntimeをTEIの必須配備に追加しない。比較用の局所PoCは継続できるが、新しい証拠台帳・Owner・Test SSOTは作らず、#2509/#2514/#2515/#3262の既存経路へ結果を記録する。**今回の追加は計画のみ**。共通API実運用、CLI/MCP/Studioの完全同値、外部Harnessの新比較、独自Harnessの効果、OwnerのMeaning受理はNOT RUN/未検証。



## 2026-11-01 深掘り設計の位置づけ（評価待ち）

[Issueの深掘り対象の選定](workstreams/ISSUE_DEEPENING_2026-11.md)で選んだ5件と、その上流への接続（[深掘り設計の上流接続](workstreams/UPSTREAM_CONNECTION_2026-11.md)）を、既存の作業領域へ割り当てます。**優先順位と判定段階の条件は変更しません。**

| 設計 | 主な作業領域 | #982 の一往復での位置 |
| --- | --- | --- |
| [#1127](https://github.com/hat47x/tei-platform/issues/1127) 標準CRUD | E（設計の実行と検証）、F（実行先に依存しない表示） | 標準Function / UIの即時実行 |
| [#1129](https://github.com/hat47x/tei-platform/issues/1129) 効果規則 | B（意味の定義・記述）、E | 整合の確認（副作用と所有者） |
| [#1130](https://github.com/hat47x/tei-platform/issues/1130) 実行計画 | I（本番運用の境界）、E | 帳票・非画面業務 |
| [#2144](https://github.com/hat47x/tei-platform/issues/2144) 期間の状態 | B、I | 締めと訂正（追補候補） |
| [#1803](https://github.com/hat47x/tei-platform/issues/1803) 設計成果物 | B、G（編集環境と開発体験） | 不整合の発見と、意味の差分の確認 |

この割当は、5件の設計が提案段階であることを変えません。5件の実装が進んでも、判定段階B・Cの合格条件と、`docs/status.md`の実装状況は別に更新します。

上流に足りない決定は、[深掘り設計の上流接続](workstreams/UPSTREAM_CONNECTION_2026-11.md) 6章に置きました。マイルストーンの到達条件、#982 の一往復の担当、#1119 の契約枠への追加は、未決の判断として残ります。PoCの作業は、既定案を選んで進め、後から覆せる形で実装と検証を続けます。既定案を選んだ理由と覆す条件は、関連するIssue・PRへ記録します。

<!-- tei-data-form-read-handoff-20261106 -->
## 2026-11-06 データ様式と実Readの横断引渡し（採用前・既存Ownerに帰属）

データ様式の詳細な構造保存・変換・反例は[既存#2497の設計Draft #2496](https://github.com/hat47x/tei-platform/pull/2496)、組織Relationの業務意味は[#2389](https://github.com/hat47x/tei-platform/pull/2389)、参照モデル移植のRead-first検収は[#2783](https://github.com/hat47x/tei-platform/pull/2783)に帰属させる。本節は**最新main上の既存開発順へ渡す最小契約**のみであり、第二の構造正本、Query/Role Resolver、Host、Evidence Registry、Scenario/Gateを作らない。

**関係のIdentityと編集対象を構造ごとに区別する**：

| 元データの様式 | 保存すべき対象と不変条件 | 暗黙に推論してはいけないこと |
| --- | --- | --- |
| **Direct Relation**（#2681のNative `CaseRecord.handlingUnit`） | 所有Case ID、Relation Role、現在値、対象Source/Context。更新Intentには実Providerに束縛された対象行と必要なExpected Version | Graph描画線・JOIN行から独立Edge IDを発明しない。同NativeのCaseRecordには`rowVersion`属性がなく、合成Provider例の7→8は正式Field/CASの証拠ではない |
| **Materialized Relation**（`Membership`・`Occupancy`） | Membership IDなど独立した関係DataのIdentity、多重Occurrence、Relation向き・Field・粒度。後段の集計/編集が使うものを落とさない | 同じPerson/Unit/Labelの辺を勝手に一つへ統合しない。`validFrom`/`knownFrom`の属性宣言だけから二時点Query・Provider pushdown・時点Owner受理を推論しない |
| **Derived Relation/View**（Graph/Tree/SQL JOIN/集計/SHO） | 元SubjectのIdentifier/Occurrence（存在する場合）、Mapping、Scope/Grain、Source occurrence/Revision、Lossと用途相対Claim | 表示線・Row index・派生Count/0件だけから書戻しTarget、全件不存在、現在Grantを決めない。元への可逆写像が未成立ならread-only/明示Action Intentに留める |

**最短の製品受入順**：[#2586](https://github.com/hat47x/tei-platform/issues/2586)の既存同一Attemptを唯一の実Read統合Ownerとし、(R0)独立Expectedと負対照・Source/Provider Targetを固定 → **(R1) 1 Function・1 Query・1 Provider**の実Loaded/解析/Consumed Source、Function Data Contextの明示Relation path/Field/Grain、実Provider Rowset＋Field状態・Scope/Read Cohortを対照 → (R1v)同Queryへの二つの**明示的**SQL Realization切替・負の選択依存を対照 → (R1p)Page1/2継続の同一Realization/Cohort/完了根拠を検査 → Consumer開示やWrite/Effectは別資格、という順番で進める。

最初のR1は[#2783](https://github.com/hat47x/tei-platform/pull/2783)の合成受注（Tenant A/Order1001、子2件、他Tenantの同業務番号を負対照）を主線とする。**組織Case E2a**（Case C1→OrgUnit D1、C2→D2を負対照）は[#2681](https://github.com/hat47x/tei-platform/pull/2681)の別Subject/別Expected/別Attemptとして並行可能。**組織Membership E2b**のvalid/known・多重出現は後続。これらを一つの実行で成功したかのように合算しない。mainの`examples/xtea-lowcode`は`UNBOUND_DATASOURCE`を明示し、org NativeのCaseRecord Persistence/Provider Readも未接続であるため、R1実Readを成功済みとしない。`tei dbcheck`の接続とTEMP TABLE確認、Parser/OperationSurface/合成Scenario/旧Go CI PASSをR1物理Readの代替にしない。

**判定方法**：対象Claimが必要とする`Required（独立検証基準/業務意味）→ Declared → Resolved/Actually Consumed → Observed（Rowset/Field/Scope/Cohort）→ Current Consumer Qualification`を既存#1163/#2514/#1180の一試行証拠で対応付ける。Declaredが空でもRequiredが不明ならempty PASSとしない。部分/未取得/NULL/非開示/完全な0件、Entity Count/Relation occurrence Count/Join増幅、過去のEvidence/現在Grantはそれぞれ分ける。Product Readerの呼出し辺が見つからなければ`product-read-not-connected`、Case Providerが未接続なら`physical-read-unavailable`と**具体的に不足する辺を記録**し、合成テストの成功で補わない。

<!-- tei-data-form-evidence-absence-outcome-separation-20261107 -->
**未観測・不在・0件・失敗を混ぜない横断負例（既存#2586/#2514の判定票へ付加）**：

| 実際の観測 | 独立検証基準で許される結論 | 禁止推論／実装の所在 |
| --- | --- | --- |
| `*.tei.md`がManagedFilesに存在するだけ／#3031 Guardが`TEI_MD_PARSER_UNBOUND` | `not-loaded/unsupported`、必要Sourceの未充足。Raw Draftは保存対象になり得る | `validate ok`、Catalog空、`proposals=[]`からMeaning 0件・正常終了を推定しない（#3031 Draft、#2823） |
| `model catalog -namespace no.such.ns` | `NAMESPACE_NOT_FOUND`で`ok:false`、終了コード1。既存の宣言済み空Namespaceとは区別 | main統合[#3052](https://github.com/hat47x/tei-platform/pull/3052)の限定CLI変更。全Readerの必須Source完結や`v0`互換の最終判断までは示さない |
| 必要対象を一件も取得していない／一部のみ観測 | 集合の空・全件完了とも限らず`unverified` | [main統合済み #3051](https://github.com/hat47x/tei-platform/pull/3051) exact HEAD `581e699b62b126c86388c88c7c357674394ad252` の[限定Go/Health CI](https://github.com/hat47x/tei-platform/actions/runs/42256518693)はsuccess。ただしCallerの`CoverageComplete`自己申告を独立に裏付けるものではない |
| 対象内で`Holds:false`を一件観測 | 被覆が部分でも当該条件の`fails`証人になり得る | これを「全Targetの取得完了」や他Claimの普遍否定へ拡張しない（#3051の同一Go試験） |
| 実Rowsetが0行で返る | Selection/Scope/Cohort/同一Snapshot／Page終了条件が完全と実証された場合に限り対象の空を候補化 | 実Providerを呼べない`UNBOUND_DATASOURCE`、別Tenant/Replica・途中Page・JOIN増幅から完全な不存在を結論しない（#2586/#2611） |
| 参照Commandが`committed:true`を返す | 参照UnitOfWorkのCommit呼出しは完了し得る | main統合[#3054](https://github.com/hat47x/tei-platform/pull/3054)は`persisted:false`/`durability:none`を別表示。実DB永続化・独立再Readではない。main統合[#3053](https://github.com/hat47x/tei-platform/pull/3053)の`COMMAND_EFFECT_UNKNOWN`と再送拒否、[#3060](https://github.com/hat47x/tei-platform/pull/3060)の未解決`Differences`は**ReadのE2a合格とは別の効果境界** |

**受入の最小順**：#2586では①Requiredの実Target/Field/Scopeと独立期待値を先に固定、②宣言したPattern/Query/DB Aliasと実選択・実Provider消費の辺を対照、③行・列・集合・Page/Cohortの完全性を観測、④対象Claimだけを`holds/fails/unverified`へ判定する。Command/Receipt/Current Grantは同一Read成功から自動で継承せず、既存#1174/#2516へ分離する。上表は新Public Status Enum・新Test Runner・新Hostを提案せず、既存CLI/Completion/Source/Effectの機能差を同じ独立Expectedで比較する**計画**。実Function→Provider Product R1と`*.tei.md` Reader E2Eは今回**NOT RUN**。

<!-- tei-data-form-selected-alias-write-preflight-versus-runtime-effect-20261107 -->
### 2026-11-07 多重DB Aliasの書込前検査と実Provider対象を切り分ける（#3070／#2611、受入案）

main統合済み[#3070](https://github.com/hat47x/tei-platform/pull/3070)の`internal/command/datasource_access.go`は、DataEffectがcreate/update/deleteの場合にApplicationの`database <alias> {target ... access read}`とDBの`SemanticUses`を照合し、最初に一致したDB要件がread-onlyなら`COMMAND_DATASOURCE_READ_ONLY`で**Executorより前**に拒否する。単一Aliasの拒否／read-write／非該当宣言の局所Go試験はPRで報告済み。ただし**実DBへの接続、物理Provider権限、現在Host Grant、実Read/Write Effectの同一性は観測していない**。該当DB宣言を見つけられずreturn nilする経路は、書込許可の証拠でなく、宣言照合の**非拒否**にすぎない。

**新たなSource-staticの曖昧性**：`checkDatasourceAccess`は`app.Databases`を先頭から走査し、`databaseUsesNamespace`が真の**最初の一件**で拒否または`return nil`する。同じ`sales.order`を参照するRead Replica（`archive:access read`）とWrite Primary（`primary:access read-write`）が併存し、**既存World契約がこの構成を許容する場合**、宣言順を`[archive,primary]`から`[primary,archive]`に反転すると、実Selected Aliasを確認せずにPreflightの拒否/非拒否が逆転し得る。これはGoコードからの**反例予測**であり、現実に有効なFixture/実Go試験/実Providerへの越境を確認したものではない。複数AliasそのものがWorld Validatorで拒否されるなら、第一の独立Expectedはその明示診断となる。

**重複しない検収の最小追加**：既存[#2611](https://github.com/hat47x/tei-platform/issues/2611) PM-03/04（Read Replica vs Write Primary、Alias順反転）に`checkDatasourceAccess`を含む同一World固定Fixtureを追加し、①有効なWorldか、②実際に選択されたAlias/DB/Operation/Scopeはどれか、③read-onlyの宣言で拒否したか、④非拒否ならどのHost Grant/Provider実Targetが実際に許可・強制したかを別列にする。Alias順だけで実対象・認可・業務意味を推論しない。**新Provider Resolver/Permission Engineは設けない**。

またmain統合済み[#3068](https://github.com/hat47x/tei-platform/pull/3068)は、Command Executorが適用操作を列挙しなければ`OperationRecord.AppliedUnobserved=true`とし、CLI/Studioへ`difference-check: not observed`を伝える。従って`Differences=[]`から**「適用対象と宣言に差がない」**と判断できるのは、必要なOperation/Scopeが実際に列挙・観測された場合に限る。これは参照実装の証拠品質改善であり、実SQL・値の前後・DB Commit・独立再読取は未実証。必要な取得対象を実Readerが列挙できない場合の`CoverageUnestablished`と**同じ非推論原則**を共有するが、Read R1とCommand E4の実行資格を合算しない。

**配置／証拠上限**：多重Alias/Targetの選択判定は#2611/#2611 PM-03/04、Functionの実Provider Readは#2586、Host GrantとEffectは#1174/#2516、独立検証基準は#2514に保持する。今回の検証はmainの実Source静的照合と既存PRの限定試験記録であり、**多重Alias新Go試験、実Provider/DB、Owner受理はNOT RUN**。

**Owner／状態**：実ReadとFunction Contextは#2586/#2506、Provider/DB Aliasと明示SQLは#2622/#2611、構造の保存/Source編集は#2497/#2621、組織の意味とCaseは#2389/#1126、独立Expectedは#1131/#1180/#2514、Writer/Host Effect/Receiptは#1129/#1174。追加された本節は設計引渡しであり、実TEI Read/DB/Consumer/Owner意味受理の実験は**NOT RUN**。#2496は古いDraftであり、この節を同Draft全体のmainへの自動採用とみなさない。

<!-- tei-data-form-e1b-candidate-vs-e2-physical-read-20261106 -->
### 2026-11-06 Read候補と実取得の接続不足を一段具体化（#3024／#2586、採用前）

[#2681](https://github.com/hat47x/tei-platform/pull/2681) のNative `.tei`では、`org_view`の`org_read`（`org_store`、read）とMembershipのPersistenceだけがある。元のCaseRecordはEntity/直接Relation定義のみでPersistenceがない。現mainの`internal/operations/surface.go`にある`SurfaceWorkspace→workspace.Load→DeriveOperationSurface`は、**Readable Alias＋対応Persistence＋Data Key**からread/listの**候補**を導出するread-only経路であり、業務FunctionのQueryを発行したりProvider Rowsetを受領したりする処理ではない。

[積上げDraft #3024](https://github.com/hat47x/tei-platform/pull/3024) は#2681を親とし、実Native Workspaceの一時コピーへ`persist org_store.case_record`を加え、実Loaderと`SurfaceWorkspace`がCaseRecordのlist/read候補（Read Inputは`organization.case_record.case_id`）を導出できるかを試験する。基線でMembershipの2候補、Mapping追加後は計4候補となることを、[#3024 exact HEAD `cc10afc61f` のNative Case Mapping CI #42204092007](https://github.com/hat47x/tei-platform/actions/runs/42204092007) **completed/success**（job `native-case-mapping`のSource/整形確認、Native Go特性テスト、checkout清浄確認が成功）で検証済み。**この合格はE1bの候補導出に限定**し、Provider Binding/Row/Field/Scope/Cohort/Consumer/現在Grantを証明しない。親#2681も未統合であり、#3024はmain向け単独差分ではない。

このため組織E2aの不足は2段で表示する：（1）**E1b/Candidate**：CaseRecord Mapping→実Loader/OperationSurfaceの候補と読み取り入力Identityを限定検証（#3024）；（2）**E2a/Actual Read**：明示Function Data Context→選択DB Alias/Instance→Provider/Query/Args→実Case C1/D1のRow+Field/Scope/Cohort→独立Expected/Consumer（#2586/#2506/#2622）。途中の`Binding=""`・`Effects=[]`・`read-only`は実Read/Host Grantを示さない。Owner/HostのE4 WriteやRowVersion Bindingはさらに別である。

**横断の選択基準（既存Sourceの宣言辺を除外しない）**：受注RMTの合成受容例Eは、実Sourceの`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`、`application order_entry`の`database sales {target sales_core}`、Queryの`datasource sales`と`recent_orders.sql`が**宣言済み**（[#2783 §14](https://github.com/hat47x/tei-platform/pull/2783)、[#2586引継ぎ](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6690394573)）。「Pattern→QueryやAlias→論理DBが一切未接続」とは書かない。一方、`UNBOUND_DATASOURCE`は**実RuntimeでのProvider Instanceへの束縛・実Query呼出しが未証明**であることと整合する。最初の実行時不足辺は (1) ContextのQuery選択／実Consumed、(2) Alias→物理Instance/Provider、(3)選択SQL・引数、(4)実Rowset/Scope/Consumerの順で照合する。Org Caseは#3024のE1b Mapping/候補とE1c Query/SQL Source受容までは局所Go実証済み、Function→Query選択・実Providerは未接続。両者の**最初の不足辺は同じと仮定しない**。#2586の既存一Attemptへ最初の欠落Edgeのみを渡し、二SQL選択／Page継続は製品実Read成立後に検証する。第二Reader/Host/Query EngineやRelation一律IDは加えない。

**R1引継ぎへ加える最小の反例**：#3024が一時Workspaceへ追加した`persist org_store.case_record`は、`case_id`を単一Primary Keyとする**単一Mappingの局所特性化**である。導出されたRead Input `organization.case_record.case_id` は、そのMappingでの検索候補を示すにとどまり、同じ文字列`C1`が別Tenant/DB Instanceに存在する場合のグローバル一意性・可視性・業務所有権を証明しない。最初のC1→D1実Readに対して、**選択Alias/Instance/Provider＋Tenant/Scope＋Case ID＋Relation Role＋実Row Identity**の対応を同一Attemptで保存し、別Tenant C1→D2を混入させない。必要なTenant Scope/物理Keyは業務モデルとProviderごとに決め、全DomainへTenant列や独立Edge IDを一律強制しない。#2389 §71/#2586/#2611/#1656の既存負例・Ownerへ引き渡し、第二Fixture正本を作らない。

**証拠の版**：本PR #3023 旧HEAD `43775e2daf1341243aad639ab504cef3266698e8` の[Repository Health #42203726377](https://github.com/hat47x/tei-platform/actions/runs/42203726377)は **completed/success**。#3024のPASSは別HEAD（`cc10afc61f`）・別Scopeの局所Go検証であり、同一Product Readの合格へ加算しない。本追補は計画Markdownのみ。**追補後の新HEADのCIは別途確認**。実DB/Provider/Consumer E2E、実Host Grant、Owner意味受理は今回NOT RUN。

<!-- tei-data-form-e1c-and-tei-md-migration-20261106 -->
### 2026-11-06 E1c実Query読込と `*.tei.md` 統一方針の引継ぎ（決定と実装資格を分離）

**新しい局所実証**：[積上げDraft #3024](https://github.com/hat47x/tei-platform/pull/3024) のexact HEAD `705933667d5b081036314a9161280323e08a64bd`、[Actions #42208159977](https://github.com/hat47x/tei-platform/actions/runs/42208159977)は**completed/success**。実Native `workspace.Load`を用いて、コピー先へ追加した`persist org_store.case_record`のE1bに加え、`query org_view.case_read`（input `case_id`、datasource `org_read`、Case ID/handlingUnit IDのResult Mapping）・別`case_read.sql`のSourceContributionをE1cとして読み込んだ。SQL WHERE句の本文だけを改訂した場合、同Query ID/Result MappingのままWorld FingerprintとModel Revisionが変化する局所回帰を含む。旧試験では入力名`caseId`が文法に不適合だったため`case_id`へ修正して成功。**今回証明されたのは宣言とSource版変化のLoader経路であり、Function Context→Query Binding、SQL実Provider選択・実行、Rows/Fields/Scope/Cohort/Consumerは一度も製品として検証されていない**。#2586のE2aは次の一辺を保持する。

**Owner決定（Authoringの将来形式）**：`*.tei`の新規採用を原則停止し、TEI文書を**`*.tei.md`へ統一する方向**とする。`*.tei.md`は単なる汎用`.md`との区別をファイル名で維持し、Markdown編集・表・図・説明とTEIの構造化データを共存させる入口とする。これはユーザーの形式選択であって、現行Language Plugin/Parser、`formatVersion`、既存Workspaceの実装・正式Owner Meaningや実行Grantが移行済みという意味ではない。従来[#2621](https://github.com/hat47x/tei-platform/issues/2621)の「.teiを厳密Sourceとして残す／.tei.mdを標準化しない」**未採用比較案**より、この方針を優先し、既存の説明を新しい決定と混同しない。OKF frontmatter/tagsは文書の意図を示す追加情報であり、`tags:[tei]`やファイル名から意味受理・Host権限を自動発生させない。

**現行mainの具体的移行Gap**：`internal/workspace/loader.go`は`semantic`を`strings.HasSuffix(name, ".tei")`、`applications/.../queries`・`persistence`などを`*.tei`、`application.tei`/`database.tei`/`plugin.tei`等を**固定名**で列挙し、発見した原文を`syntax.NewParser`へ直接渡す。`scanManagedFiles`の`filepath.Ext("model.tei.md")`は`.md`となる。よって**現状の`*.tei.md`はNative意味Sourceとして発見・解釈できると主張しない**。Migrationで`*.tei`の一斉rename、旧ParserへのMarkdown本文直接投入、`.tei.md`を汎用mdとして無言skip、または旧新両方を同じSemantic IDで二重採用することを禁止する。

**移行計画の担当整合（最新文書で解消）**：移行専用[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030) の`TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md`最新冒頭は **M0〜M6/TM-01〜14を移行の主たる実装・受入候補**とし、[横断Draft #2667](https://github.com/hat47x/tei-platform/pull/2667) の`TEI_MARKDOWN_SOURCE_MIGRATION_2026-11.md`も **M0〜M5/TMD-01〜12を先行研究・履歴・対応表**として#3030へ委譲した。以前の「双方が相手を正本と呼ぶ」矛盾は**両ブランチの現行本文上で解消**。ただし両PRはDraft・main未統合であり、個別TM/AUの正式な実装・Owner業務意味受理は別である。#3031の暫定Loader停止ガード、#2823の実Loaded Coverage、#988のGrammar/Pin、#2497の構造保存、#2586の実Function Readは引き続き各担当に帰属させる。

<!-- tei-crosslane-migration-owner-conflict-and-terminology-20261107 -->
**実装された暫定停止線（別Draftの限定CI）**：[PR #3031](https://github.com/hat47x/tei-platform/pull/3031) は、選択されたWorkspace Source rootsを走査した`ManagedFiles`に `*.tei.md` があると、未選択Parserのまま意味が空になる誤成功を防ぎ`TEI_MD_PARSER_UNBOUND`＋Source pathで拒否する暫定Guardを追加。exact HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae` の[専用CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093) は**completed/success**（Focused 4件/7 Domain圧力、Workspace package試験、vetの限定結果）。これは**未マージDraft上でのFail-closed**であり、現行mainの`*.tei.md`解析・Parser選択・Meaning/Build資格が成立したことではない。Source roots外の文書はこのGuardが観測しない。Manifestの誤Root/optional未探索の完全性は引き続き#2823の別Claim。

<!-- tei-markdown-parse-reject-vs-validation-proposal-20261107 -->
**暫定Load拒否と診断修正案の非同一性**：main統合済み[#3041](https://github.com/hat47x/tei-platform/pull/3041)の`validation.Propose`は診断に**World Fingerprintを結び、`Accepted=false`で提案**する限定実装（`tei validate -json`の`proposals`）。一方、#3031の`TEI_MD_PARSER_UNBOUND`はWorldの正式Parse前にLoadを拒否するため、**`proposals`が空でも「修正不要」「全必須Source読込済み」ではない**。将来、暫定Guardを新Markdown Readerへ置換するときも、Required Sourceと実Loaded/Parsed、Diagnosticと未検査Scope、版付きProposalとOwner意味受理を区別し、#3030のTM-02/12/14と#2823の既存試験へ加える。#3041の`validation.Accept`は版一致検査を提供するが、修正Patchの適用・実Provider Read・業務Ownerの意味受理・Host Grantを証明したものではない。

**M1とM2/M3の順序補正（#3030への提案）**：`*.tei.md`を**下書き文書の新規作成既定**にすることは可能だが、Markdown Reader/選択Parser/Loaderが実Sourceとして消費できるまでは、選択済み実行Workspace rootへ「実行可能な既定Source」として投入しない。Raw Draft用の未選択領域とLoaded Worldの要求集合を混同しない。もし選択済みrootに旧`.tei`と新`.tei.md`が同居するなら、現#3031暫定Guardは**同内容でも拒否**する。将来、両方が解析可能になった段階では、同Subjectの二重正本を拒否またはOwnerが一方を明示選択し、先勝ち/後勝ちしない。

**R1との最小接続**：#3024の旧形式E1b/E1cは、固定されたLegacy Workspace/Source Revisionでの`CaseRecord Mapping→case_read Query→SQL SourceContribution`までの証拠。移行試験ではTM-02/03/05/09/12/14を再利用し、(a)宣言Sourceが選択・実消費されたか、(b)旧新SourceのSemantic/Query/Result Mappingが独立Expected上で整合するか、(c)同じSQL IDでもSQL-only変更後のRevision失効が生きるか、(d)raw byte/Markdown SpanとSQL相対出典が失われないかを、**移行後の別HEAD**で検収する。旧形式E1c PASSと新形式E1c PASSは別資格。#2586の最初の実Provider Readは検証可能な旧形式の固定Workspaceでも先行できるが、その成功を新形式の移行適格性として再利用しない。

**順序の衝突回避**：Markdown容器文法とLoader移行の未完成を、Read-first P0-R1全体の停止条件にしない。実製品Readerが未接続の間は既存`.tei`でSource/Query/実Readの不足一辺を確認してよく、その証拠にSource形式の旧版と適用範囲を明記する。新形式移行後には、旧E1b/E1cのSource/Query/SQL出典と実Readの適用範囲をそれぞれ再資格化する。Authoringの検収候補は#3030/#2621/#988、Source/構造保存は#2497、実Readは#2586、独立期待値・独立検証基準は#2514に帰属させる。新全域Parser/Host/Registry/第二Query Engineを増やさない。

<!-- tei-data-form-source-read-witness-and-cli-v0-20261107 -->
### 2026-11-07 Sourceの管理・実読込み・意味適格性を現行CLIで検収する（#2823／#3073／#3078／#3081）

**mainで新たに観測できる事実**：
- [#3073](https://github.com/hat47x/tei-platform/pull/3073)は`validate -json.sourceRootFindings`に、存在しない宣言root（`severity:error`、`ok:false`）、既知一覧にないが実在するroot（`severity:warning`）、`.tei.bak`/`Application.tei`等の非解釈ファイルを区別して返す。未知rootの警告は**正常なLoaded Sourceの証拠ではなく**、警告のみなら`ok:true`を維持し得る。Containment・symlink escapeの実拒否はこの欄への警告に格下げしない。
- [#3078](https://github.com/hat47x/tei-platform/pull/3078)は`ManagedFile.readByLoader`を、`readSourceForLoad`が実読取したバイトの`sourceSnapshot`への存在に基づいて立て、`workspace-files`へ`managedFiles[]`と`sourceRootFindings[]`を返す。**`readByLoader:true`は実Readバイトの証拠だけ**で、Parsed Occurrence・Role/Meaning採用・Operation/Build実消費の証拠ではない。逆に`false`はファイル名が管理対象でも現Loaderが解釈していない状態で、宣言無しの証明ではない。
- [#3081](https://github.com/hat47x/tei-platform/pull/3081)は宣言0件に`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS` warning、`declarationCounts.total=0`を追加するが、`ok:true`・exit 0を維持する。警告があっても失敗しないCLI契約と、**実FunctionのRequired Sourceが揃ったか**の判定を別に行う。

**同じ公開合成Workspaceで比較する最小試験（#3030 TM-02/03/04/12/14と#2823の既存Fixtureを利用）**：

| 組合せ | 期待される観測と資格上限 |
| --- | --- |
| (A) 既存`semantic/order.tei`＋通常`notes.md` | `order.tei`は`managed=true/readByLoader=true`、`notes.md`は`managed=true/readByLoader=false`となる既存#3078正例。Reader実Consumeは認定できるが、業務Function/Provider到達とは別 |
| (B) `order.tei`を`order.tei.md`へ移すだけ | 現**main**では`managed`でもLoaderがまだ読まず、必要Sourceは未Loaded。`sourceRootFindings`警告/宣言0件warningがあっても`ok:true`はあり得る。別の未統合[#3031](https://github.com/hat47x/tei-platform/pull/3031)適用Checkoutなら`TEI_MD_PARSER_UNBOUND`で明示拒否。二つのブランチの結果を一つの製品状態と呼ばない |
| (C) 旧新が同じSource rootに併存 | 旧SourceだけでWorldが成立し、新Markdown側のbyte改訂がSource Indexには現れても**実Worldのモデル変化がない**反例を対象にする。旧新同一Subjectの意味衝突・Owner Selectionは新Readerの独立試験として未決。宣言順やFilenameで黙って採用しない |
| (D) manifestに存在しないroot／実在する未知rootを指定 | 存在しないrootはerror。実在する未知rootはwarningだが`readByLoader=false`とRequired不足を別に確認する。warningを必要Source集合が空である証拠へ変換しない |

**外部CLI Consumerの互換性に関する既知の未決**：#3078で`tei.cli.workspace-files/v0`の`result`が**配列→`{managedFiles,sourceRootFindings}`オブジェクト**へ変わった。[#3092/#3095](https://github.com/hat47x/tei-platform/pull/3095)のDiscoveryと実応答`schemaVersion`一致試験は、この**同じv0の内側JSON型の互換性**までは示さない。旧Readerが配列として解釈し空集合へ縮約すると、Loader対象の欠落が隠れる。旧Binary×旧Consumer、新Binary×旧Consumer、新Binary×新Consumer、旧Binary×新Consumerの**既存[#2824](https://github.com/hat47x/tei-platform/issues/2824)互換判定**をOwner判断で行う。`schemaVersion`一致やIndex Fingerprint変化だけをLoader/Consumer互換のPASSとしない。

**実Readとの最短接続**：#2586では`Required Source/Query/Field/Relation/Scope → Manifest Selected → Managed → readByLoader → Parsed Source/SQL Byte・args → Functionから実選択 → Provider Row/Field/Cohort → Consumer独立検証基準`の**初めて不足する一辺**を同一Attemptで示す。E1b/E1cと本節のSource観測はR1への入力であり、**実Provider取得の代替にならない**。正式`*.tei.md` Reader、TM新形式正例、外部Consumer旧新実実行、実Product R1/Owner受理は**NOT RUN**。新CLIレジストリ、Parser、独立Fixture群を本PRで実装しない。

**本節の証拠上限**：既存`internal/workspace/loader.go`と#3024 E1c Go/CIの実照合に加え、#3031暫定Guardの固定HEAD限定CIと、#3030移行設計の実Sourceを確認済み。実`*.tei.md` Markdown Reader/意味Parser・Builder・Writer、移行後Org E1c、実Provider/DB、SHO/Consumer、Owner意味受理は**NOT RUN／未成立**。この文書は設計引継ぎであり、#3030/#3031が既に担当する詳細契約・Goコードを二重定義しない。

<!-- tei-design-context-test-ability-t0-5-20261107 -->
### 2026-11-07 Issue→設計→テスト能力：Main実証とMVPへの統合（#3262／#2514）

**実装／実証済みの小さい縦切り**：[#3264](https://github.com/hat47x/tei-platform/pull/3264)は既存TEI CLI 9 read-only操作で一SubjectのSource/Inventory/Loader/Versionを限定検証。[#3266](https://github.com/hat47x/tei-platform/pull/3266)は既存#1180 Scenario SSOTから生成した**実別プロセス pinned CLI 6 Step**と、実CLI拒否4件／独立期待値検査2件をCIで確認。[#3268](https://github.com/hat47x/tei-platform/pull/3268)はSource/CLI/Scenario/Skill/Go依存変更を監視対象に追加してmain push再試験を実証。これらを全機能、正式意味、外部ProviderのE2Eとはしない。

**独立Requiredを満たしたかで次段の資格を決める**：[#3284](https://github.com/hat47x/tei-platform/pull/3284)は実9 CLI Replyを読む**テスト専用Consumer**をmainへ統合し、Source未読込、Subject欠落、Fingerprint版混合、Tool Reply欠落、Schema変更、Source root欠損を安全に拒否する6合成対照＋外部Provider/Harness/Grant未観測の資格保留1対照を検証。正常でも**`boundedReference=partial`／`testExecutionAdmission=exploration-only`／`qualifiedTestGeneration=false`**。Source Read≠Parsed/Accepted、同Fingerprint≠A→B→Aを含む時点・権限の完全保証、CLI受信≠H3外部Harnessへの完整なDelivery。局所`cmd/tei`1テストのStatement C0相当は10.1%、**C1未計測**。

| 段階 | Mainに取り込む/検証する成果 | 完了を禁じる不足条件と既存Owner |
| --- | --- | --- |
| **T0 実行・判別** | 固定公開Scenarioの実CLI pinned 6 Step、独立Propertyと意図的失敗の判別、正例・業務拒否の区別 | 実コードMutant・別Target・Owner/実DBの包括PASSにはしない。#1180/#2514 |
| **T0.5 Test Readiness** | 実CLIのSource/Subject/Version/必要Replyと独立Requiredを照合し、未Loaded/欠落/新旧混合を検出。必要な情報が未取得なら`blocked`又は`partial` | Product test-generation全資格、Plugin実Consumer Load、外部Harness受領／Grant／機密送出安全性は未実証。#3262/#2605/#2509 |
| **T1 AIによるCase発見** | 必須Data/Function/Business/Rule/Relationの出典・意味・未取得を同Scopeで回答し、最小指示で適用理由付きPattern／Candidate Scenarioを生成。独立Holdout非開示でMutant/false-qualified/未カバーを測定 | プロンプトからのExpected自己採点、Schema/件数・C0だけからのCoverage完了、AIがOwner Meaningを承認したふりをすることを拒否。#3223/#2514/#1180 |
| **T2 実製品横断** | Function→Query→実Provider Reader→Row/Field/Scope/Completeness、同ScenarioのCLI/Web/Studio分岐、API例外系のGraph＋表＋遷移検査。実Grant、Effect、Receipt、資源再読取を別証拠化 | 局所合成Query/HTTP静的Graph/旧Receiptや実PG別Reference成功から全経路PASSへ昇格しない。#2586/#3274/#1174/#2516 |
| **T3 自己記述・Self-Build** | TEI自体の要件/Rule/Graph/Scenario/Buildを標準Dataに記述・適切に投影、Stage1実TEI ProcessからStage2 Build/Artifactへの再入を検証 | `go build`、標準Plugin SB-3、GraphレンダリングだけでG4=L3合格としない。#2831/#1119 |

**順序と並列性**：T0.5はT1の**品質ゲート**だが、実Product ReaderのP0-R1（#2586）、Host Effect/Receipt、Plugin・Trait設計、DSL `*.tei.md`移行、Self-Build G4を全てT1の後に直列化する新Hard dependencyではない。別Ownerは並列に独立Fixtureを前進できるが、横断Claimを認定するときだけ`Required→Observed Source/実Consumer→Target Revision/Grant→Independent Expected→Claim Ceiling`の**同一試行**で照合する。T0.5のtest-only Python Readerは製品用Design Context Query・第2Parser/Verifier/権限正本に昇格させない。

**正本／次の引渡し**：[実装・証拠・詳細T0〜T3](design/ISSUE_TO_DESIGN_DOGFOOD_MVP_2026-11.md)を参照。T1では#3262のtyped read-only Contextの要件Closure／Unknownを先に実証し、#3223のPattern推薦→#1180採用Scenario→#2514の独立判別へ移す。外部H2/H3でのSkill効果はTAI #168／TEI #2509が固定Task・Skill実ロード・完全Reply・送出資格を揃えた**独立実験**を行う。Source/Rule/Plugin/Owner受理、C1・NFR、Function実Provider、外部H3 Test-Case生成は現時点で**NOT QUALIFIED**。
<!-- tei-data-shape-presentation-r1-20261107 -->
## 2026-11-07 標準データ構造・表現型・機能内Presentationの最初の受入（設計候補）

**目的と位置づけ**：X-TEAのテーブル間1:1／1:N関係、単票・明細やXF系機能の対応関係を、TEIでは**物理テーブルに依存しないData Shape**、**機能から独立して定義できるPresentation Type**、**特定Functionに束縛されたPresentation Instance**へ分解する。これは採用済みCore Schema／実装完了の宣言ではない。詳細型／Pattern適合の正本は[#2506](https://github.com/hat47x/tei-platform/issues/2506)・[#2507](https://github.com/hat47x/tei-platform/issues/2507)、標準CRUD／機能は[#1127](https://github.com/hat47x/tei-platform/issues/1127)、UI実現は[#997](https://github.com/hat47x/tei-platform/issues/997)に置き、本節は**製品横断の検収順序**だけを束ねる。既存のデータ様式設計・Read-first詳細は[旧横断Draft #3023](https://github.com/hat47x/tei-platform/pull/3023)を履歴・検討根拠として参照し、同じ大量の受入票を重複管理しない。

| 層 | 意味・責務 | これだけから推論しないこと |
| --- | --- | --- |
| **標準データ構造／Data Shape** | Scalar/Record/Collection、1:1・1:N等のRelation／Graph・Tree・時系列・空間／多次元・表、Identity・Cardinality・Grain・単位・NULL/欠損・出現ID・Temporal/ScopeのFacet。DB Table/列とのMappingは別 | テーブル・FK・ラベルや見た目からBusiness Meaning、正当な更新対象、必要全集合、現在Grantを決めない |
| **標準表現型／Presentation Type** | Panel/Form、Tabs、Text Input、単一/複数選択、On/Off、Slider、Table、Tree、Chart、Map等の型付き表示・入力方式。表示器／TargetやPluginへ実現を委譲 | UI種別からFunctionの業務責任、保存/承認Action、DB Bindingを生成しない。型の定義はFunctionに依存しないが、**実際の業務PresentationはFunctionの外に単独存在させない** |
| **機能内Presentation／Bound Instance** | Function Data Contextの対象／Relation path／Field／Grain／Selection／必要入力、Projection、Presentation Type Binding、Action Intent、状態/Revision・Consumerを**明示**して実際の表示を構成 | 表示可能≠変更可能、Operation候補≠現在Dispatch可、Save click≠Commit・Receipt・保存後対象データ確認 |
| **Provider/Host/Consumer** | Query/Target/Realizationの選択、実Provider Rowset/Field/Scope/Cohort、Host Grant/Effect/Receipt、実Consumerでの限定Projectionを各担当契約で連結 | Alias名やOperation ID、Source/Receiptの指紋一致から現在の実データ同一性や開示許可を認定しない |

**参照実装で観測された不足**：[main統合 #3145](https://github.com/hat47x/tei-platform/pull/3145)は固定Workspaceと見積候補を比較し、Model Revision・World Fingerprint・`function-definition`のPattern（候補側6件）は変わる一方、`tei operations`は**8件／gaps 0／conflicts 0／declared 0、全8件がderived**で不変と実測。つまり**既存OperationSurfaceに宣言Pattern/Business Actionを取り込むW3経路は、この限定観測では成立していない**。これを「全機能／全Actionを構成できる」証拠にしない。逆に、PatternをOperationへ無条件に複製する設計を決めたわけでもない。最初の関連は`function-definition`/宣言Sourceを通じて照合し、Action Purpose、KindとEffect、粒度/Context、Authorityの対応が揃うまでは`unresolved`として保持（#2507/#1127）。既存`surface.go`のderived Operationと実Service Contract/Commandの所有関係は別。

### 同一Fixtureでの最小受入順（試験IDの第二正本を作らない）

| 順位・Owner | 一つの正例と、先に固定する負例 | 成功の上限／停止線 |
| --- | --- | --- |
| **R0→R1／#2586・#2506・#2514** | 受注`Order1001`と明細2件（または組織`Case C1→Unit D1`）を独立Expectedに固定。Function Context→明示Query→Alias/実Provider→実Row/Field/Scope/Cohortの**同一Attempt**。別Tenant同番号、未取得、部分集合、NULL、完全な0件、Join増幅を対照 | まず**一Function・一Query・一実Provider**。宣言のみ、`readByLoader`、`UNBOUND_DATASOURCE`、型付きQuery候補、`tei dbcheck`、合成Go PASSではR1実Read合格でない。R1v二SQL・R1p Page継続は後続 |
| **P1／#1127・#2507・#997** | **同じFunction Context／Selection／業務ID**をPanel/FormとTableの二Presentationへ投影。列/Role/Unit/欠損/出現ID・Scopeを比較し、Renderer変更でもMeaning・Actionの正本は一つに保つ。Table↔Chartの集計／Tree↔GraphのPath多重性は既存FP-01/02へ | 2表示の局所投影が一致してもFunctionが実Read/書込可能とは言わない。集約の書戻しやTree node dragから更新対象を推論しない。Native Canvas等の例外はUI Kit/Targetが所有 |
| **P2／#2500・#2598・#1174** | Form編集／Tableセル編集→`ViewChange→型付きIntent→既存Action`。同一対象ID/Revision、必須Rule、現在Host Grant、選択Provider実Target、取消/再送・部分Effectの負例を対照 | read-only・適用候補・実Dispatch・実Effect/Receipt・Owner意味受理を分離。Widgetのeditable状態や`committed:true`を許可/保存の証拠にしない |
| **M／#3030・#2823・#988** | 旧`.tei`の固定R1/Presentation Fixtureと、新`*.tei.md`の同じ独立Expectedを比較。Required Root→Managed→`readByLoader`→Parsed Occurrence/Source Span→実Builder/Function消費を分離 | [#3073/#3078/#3081](https://github.com/hat47x/tei-platform/pull/3078)はRoot診断・Loader読込印・宣言0件Warningまで、[#3031](https://github.com/hat47x/tei-platform/pull/3031)は未統合の未束縛拒否Guard。新Markdown実Parserや移行後R1 PASSは未証明 |

**保存後の確認は最低二つに分ける**：[main #3124](https://github.com/hat47x/tei-platform/pull/3124)の`persistence`（`not-applicable/not-observed/observed`）と[main #3127](https://github.com/hat47x/tei-platform/pull/3127)の`effectcommit.Runner.Lookup`による独立**Receipt再照会**は、記録されたOperation ID・Payloadの照合であり、**業務対象Recordの現在値を独立取得した証拠ではない**。`persistence:observed`を出しても、Consumerへ「受注保存後の内容を確認済み」と説明するには**別のProvider Read／対象Identity/Scope/Version/Fieldの独立Expected**を要する。さらに`AppliedUnobserved`と`Differences=[]`の非同一性（main #3068/#3104）を維持する。

**優先順位・資格**：R0/R1は旧`.tei`で先行でき、Markdown ReaderやGUI完成はP0のブロッカーにしない。P1はR1と同一のMeaning/Context/Expectedを再利用し、表現を切替えても各Consumerの許容Scopeを変えない。P2とMはそれぞれ別Claimとして後続検証。構造・型・Role/Patternの提案と、機能内Presentationの業務意味受理はOwner判断。**今回の更新は既存Source/PR試験記録に基づく計画であり、P1/P2/Mの実製品E2E・実Provider R1・実DB値の独立Read・Owner正式受理はNOT RUN**。新Core UI種別、万能Presentation Engine、第二Parser/Host/Reader、影のAction/Authority正本を先に増やさない。

## 2026-11-07 標準データ構造・標準表現型と合成写像の統合検証（採用前）

X-TEAのDB 1:1／1:N→XF単票・明細という対応を、**物理テーブルと無関係な標準データ構造**、**Functionから独立した標準表現型の定義**、**Function Data Contextに所有される表現インスタンス**、**両側の構造間Mapping**へ一般化する検証を進めます。Container／Table／Tree／Chart／文書／音声等が同じ元Dataを異なる形に見せる際、表示能力・意味の保持・同期・編集候補・現在のHost Grant・Provider Effectを独立に評価します。

### 成果を一本の現行main起点の候補計画へまとめる

- [標準表現型と媒体時間軸](design/STANDARD_REPRESENTATION_TOPOLOGY_AND_MEDIA_AXES_2026-11.md)：Data/Representationの構造、Function-owned Instance、元データ時刻／取得時刻／Playback時間／文書位置、軸保持・変換・生成を分ける。旧[Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)の成熟した設計記録を再利用。
- [複数表現の選択・状態・同期](design/REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md)：Subject/Result/Path/View occurrence、selection epoch、Presentation-only変更と業務Actionを分ける。旧[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)の成熟した設計記録を再利用。
- [合成写像の適格性・非単射・否定依存・開示](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)：Source→表示→他Consumerの各段でloss・Scope・unknown・Permissionを検収。局所表示成功から、合成後の完全性や逆編集を推論しない。SUI Tree/Island→SHO Outline、D1/D2 partial、空結果の否定依存、Research/Full Autoの操作モード、全出力面のDisclosureを実例とする。

**受入順（新Gateではない）**：P0 HeadlessでCX-01〜03/08の非単射・集計・空集合の判別 → P1 CX-04〜06/10のConsumer非干渉・操作Mode・epoch/版 → P2 SHO/SUI/Mediaを異質ConsumerとしてCX-07/09へ接続 → P3 #2586の実Function→Query→Provider Readを経て、編集が必要な場合のみ#2500→#2598→#1174のIntent/実Request/Receipt/独立Readへ接続。旧PRのRP/MVと新CXは[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)の既存独立Expected/Scenarioへ**差分引継ぎ**し、第二のテスト正本を設けません。

**担当を変更しない**：構造のFacet=#2495、Loss/Projection=#2497、Context/Result Slice=#2506、Pattern=#2507、変更影響=#2582、Renderer=#997、用途別開示=#2516、実Read=#2586、Writer/原文受理=SHO/TEI各Owner。今後のUI生成・Operation開発を早めるための設計であり、Core型列挙・共通Plugin Registry/Runtime/Host・第二Action/権限Engine・新Gateは追加しません。

**証拠の上限**：これは現行mainを起点とした設計資料と計画の整備であり、CX-01〜10の新試験、実Provider/Media/Renderer/Host接合、SHO/SUI実受領、Owner意味受理は**NOT RUN**。局所PRのGo/CI実績や見かけの同一表示を、実Product E2E／現在の適合資格へ自動昇格させません。並行変更がある場合は統合前に同一HEAD/内容差分を再照合します。

### 製品実Readから二表現へ渡す初回受入（2026-11-07追補）

上記の幅広いRP/MV/CX圧力ケースは維持するが、**最初にProduct資格を得る一本の実証**は[#2586 HR-01/02](https://github.com/hat47x/tei-platform/issues/2586)が所有する`order_entry.recent_orders` Headerの**Function→実選択Query/SQL・args→Runtime DataSource alias/物理Provider→actual Rowset/Scope/Field Presence→Function Data Context**に固定する。その後、同じContextからForm/Tableの**二つのread-only表現**を投影し、Identity・Grain・NULL/未取得・Page/部分集合・選択epoch・開示の一致と、情報損失を#2506/#2507/#2497で検収する。最初に欠ける接続辺はunresolvedとしてSource/Ownerとともに記録し、Headlessの参照・合成成功を実Product Read資格にしない。

現main統合[#3195](https://github.com/hat47x/tei-platform/pull/3195)/[#3204](https://github.com/hat47x/tei-platform/pull/3204)のPattern↔Operation突き合わせ参照実装は8 Operation／5 Patternの局所結果があるが、**正式なFunction/Action dispatchとSQL Readerではない**。参照用実PG Readの[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)もProduct HR-01/02とは別。反対に、[main統合#3155](https://github.com/hat47x/tei-platform/pull/3155)は資源独立Readによる`resource-observed`を既に持つため「永続化の観測機構が全TEIで未実装」とは記載しない。**Resource Readの局所成立≠受注FunctionのHeader/明細完全Read≠業務Outcome**とする。

順序はR1実Read→R2同一Contextの二read-only表現→R3必要な場合のみViewChange/typed Intent/現在Grant→R4 Effect/Receipt/独立Read→R5 SHO/SUI/Graph/Media異質Consumer。独立Expectedと否定依存／同件数異集合／LIMIT 1／Non-output Predicateの反例は既存#2514/#1180を再利用。詳細は[合成写像の適格性 §8](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)。新しい実行結果の主張ではなく、実Provider、Product二表現、Owner受理は**NOT RUN**。

### Workspace帰属・read-onlyの実効果・ER/CRUD図の資格（2026-11-07更新）

標準表現型の初回実証をR1 Product Header Read→R2同一ContextのForm/Table read-onlyとする順序は維持する。追加で三つの**適用資格**を検査する。

- **Workspace/Source帰属**：[#3149](https://github.com/hat47x/tei-platform/issues/3149)/[#3181](https://github.com/hat47x/tei-platform/issues/3181)のCLIドッグフーディングで、別Workspaceの同一 `sales.order` Subjectに対しmanifest/Workspace identityが応答に欠落する問題が観測された。Context Instance、Manifest identity、Source revision、実Provider Scope、Read epochを明示し、同名・同内容を同一実対象へ無音結合しない。CLI Contractの変更は#3181/#3242/#2824がOwner。第二Workspace Registryは作らない。
- **副作用の種類**：[#3182](https://github.com/hat47x/tei-platform/issues/3182)で、`view`のread-only広告と明示`-execute`＋ログ/状態ファイル出力が別であることが実測された。Business read-only、環境ファイル・送出効果、現在Host Grant/実Requestを独立評価する。Form/Tableを読み取り専用で投影できても`zero OS effects`、自動Action dispatch、業務完了を宣言しない。
- **異質な図表Consumer**：[#3250](https://github.com/hat47x/tei-platform/issues/3250)のSubsystem候補/ER/CRUD/依存図は、標準構造をGraph/Tableへ展開する追加圧力例として利用する。論理Subsystem/Owner候補・表示レイアウト・直接CRUD・間接Effect・実Provider Effect・物理配置を区別し、折畳み/図の移動をSource Rewriteとしない。未知・部分・非開示の越境Edgeを「越境0件」と推測しない。専用Optimizerと受理権限は#3250/#267のままとする。

**前進した実証**：[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)固定HEADの[#2586実行記録](https://github.com/hat47x/tei-platform/issues/2586)で、隔離PG 16.15→物理Read→局所Completion、Top 2/Top 1の区別が限定CI成功。これはReferenceの一部達成で、**Product Function→Query/Provider→Context→二表現のE2E合格ではない**。前節の「#3238 PGの結果は未確認」は作成時点の履歴とし、本節の限定成功を現在資格として優先する。

同じ[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)のIndependent Expected/Scenarioへ、[合成写像検収§9](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のWX-01〜06を一因子差分として引き継ぐ。新Go/実Product DB→二表現・跨Workspace UI・ER Editorの試験とOwner Meaning Acceptanceは**NOT RUN**。複数並行PRのMaster Plan編集は統合前に内容比較し、同一受入を二度追記しない。

### 実SQL Sourceと隔離PG Readの対応証拠（2026-11-07：初回実証の停止線）

[合成写像検収§10](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)に、現mainの**実Sourceファイル比較**から得られた不足条件を追加した。Product `reference/workspace/applications/order-entry/queries/recent_orders.tei` は `datasource sales` と別ファイル `recent_orders.sql` を参照し、そのSQLは **`FROM public.orders`、`WHERE ordered_at >= :from`、`ORDER BY ordered_at DESC`**。一方[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)の実PG Referenceは専用の `public.tei_web_scoped_orders_reference` とTenant限定の実行により、`orderId`ソート・TOPを比較する。Logical Query IDや`sales_core`の一致、実PGのGreenだけで、**同じSQL byte/母集団/Time Window/Physical Provider/現在認可**とは断定しない。

初回HR-01/02では、**Function/Pattern実選択→Query Source + SQL fileの実消費bytes→`:from`のBinding/Time Zone→runtime datasource sales/physical Provider/実行statement→受領Rowset・Scope・Read epoch→Context/Result Slice**を一Attemptとして固定する。原Queryの`ordered_at`順とReferenceの`orderId`順が異なるなら、先頭ページの同数・同一IDだけで表示同等・全集合Coverageを宣言しない。QueryのSQL-only変更は別Source Contributionとして失効。Contextが同じでも複数Readerの**joint snapshot**は自動成立しない。

既存#2586（Product E2 Read）、#2387/#2862（SQL Source/Variant）、#2611（Provider binding）、#2506/#2507（Context→表現）、#2514/#1180（独立Expected/Scenario）の担当へ、QS-01〜06を追加の一因子差分として引き継ぐ。これは**出典に基づく設計更新のみ**であり、QSの新Go/実Provider/二View E2、Owner意味受理はNOT RUN。新SQL Executor/Reader/Registry/Gateは作らない。

**実測で更新した表示・操作資格**：[SUI Draft #3396](https://github.com/hat47x/sui-sensemaking/pull/3396)の固定HEAD CIでは実Product `buildContextProjection` の非単射（元Source cue／Island所属が異なっても公開投影が同じ）と正の変更対照を**3/3 PASS**で観測した。[main統合済みTEI #3253](https://github.com/hat47x/tei-platform/pull/3253)はこれを研究基線に帰属する。**公開投影が正しく同じことは元Source Identityの一致・逆編集の一意性を証明しない**。一方[main統合済み#3254](https://github.com/hat47x/tei-platform/pull/3254)の操作面drift実測ではモデル属性追加でWorld/Model fingerprintが変わっても5 Operationの`basis`が不変。現在の実Request資格を旧`basis`一致から導かず、必要依存・Source/Version/Selection・実Grantを確認する。影響外と証明できるread-only表示は局所維持を認め、すべての変更で無条件に再取得・拒否する第二Gateを設けない。

**開発上の接続**：[合成写像検収§11](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)にSUIの局所PASS、TEIの操作面照合未接続、Product E2未実証を別票で保存。#2497/#2507/#2582/#2598・独立Expected #2514/#1180へ引き渡す。並行しているドメイン意味駆動のテスト生成案[#3258](https://github.com/hat47x/tei-platform/pull/3258)〜[#3261](https://github.com/hat47x/tei-platform/pull/3261)はOwner #1131/#2514との採否・重複統合を待つ**別Draft**であり、#3236の第二データ生成器やScenario正本にはしない。TEI Product二View/実逆Patch/Hostの新試験・Owner受理はNOT RUN。

### 2026-11-07 三視点Diagram/CRUDと実Provider Readを並列P0で検収する（main成果との整合）

X-TEA Modeler由来の**Data/Function/Businessの3図とCRUD cross-reference**はTEIモデラーの基幹価値であり、実Providerが先に接続できることを前提にしない。[三視点受入 Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)（M-01〜10）を既存#1124/#1125/#1126/#2497/#1803へ振り分け、**P0-M**として受理済み限定Sourceから3図・CRUD MatrixをHeadless/read-onlyで導き、Source/Role/Relation/Occurrence/Revision・宣言/派生候補/実観測Effectの根拠を同時に返す。Graphの辺・多重性/1:Nの関係Roleを保存し、行列の空欄、未取得/非開示/partialを「CUDなし」の証拠にしない。図座標・Mermaid風文書の見た目を元Source更新としない。可逆な編集はTyped ProposalとOwner Acceptanceを経る。SHOの独立図解は既存SHO #5/#6/#19 Consumerで検収する。

従来の **P0-R**（#2586実Product Function→Query/SQL/Provider→Result Slice→同一ContextのForm/Table）も維持する。**P0-C**では三図/CRUDの`sales.order`と実Read結果を型付きSubject/Source Revision/Scope/Read Attemptで照合するが、Query実行がない`R`宣言を実Reader成功と呼ばず、同IDでもWorkspace/Providerが違えば接合しない。M/R両レーンは共通の出典・欠損診断を利用し、製品Mainの新Schemaや第二Reader/Runnerは立てない。

前段の設計完全性は[#3262](https://github.com/hat47x/tei-platform/issues/3262)が所有。**main統合済み[#3264](https://github.com/hat47x/tei-platform/pull/3264)の実CLI9呼出し・Skill静的14・Required Source Loader確認**と**[#3266](https://github.com/hat47x/tei-platform/pull/3266)/[#3268](https://github.com/hat47x/tei-platform/pull/3268)の既存Scenario 6 step／実CLI負例4＋独立義務2／対象pathのCI再起動**は限定のT0基線であり、Design Contextの全Source Closureや三図/CRUD実生成、MCP/外部Harness完全受領、AIの新Case生成を実証しない。Goの局所Statement Coverage C0相当をC1や機能/NFR Coverageへ昇格させない。未カバーRuleから新Caseを作るT1は[Draft #3273](https://github.com/hat47x/tei-platform/pull/3273)と#1131/#2514/#1180が担当し、第二Oracleを作らない。

[合成写像検収§12](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のDM-01〜08を既存独立Expectedへ引き渡す。**今回の新たな実P0-M Graph/CRUD出力・P0-R Product Provider接続・P0-C Crossview・Studio/SHO・Owner意味受理はNOT RUN**。mainと並行Draftの更新内容を失わないよう、旧ブランチのMaster Planを最新mainに基づき統合し、PRの再比較とCIは別に確認する。

### 三視点Fixtureを現行CLI/Graph/Design Contextの実装入口へ接続（2026-11-07）

[三視点計画Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)にはData Entity 9／Relation 7（同端点の別Role二辺）／Function 4／Business Action 3、独立に記述した`expected-v0.json`、12種のKnown Mutant候補がある。しかし**合成JSONはTEI Parser入力でなく、Input/ExpectedともAI草案**。整合性15項目の静的検査は独立Expectedの正当性やTEI製品Graph/CRUDの実生成の証拠ではない。

実mainの `internal/modelref/model.go` はScoped Ref/Relation Evidence、`tools/reference-er-table-crossview/projection.go` はread-only ER/物理表比較、`tools/reference-business-dfd/diagram_context.go` はHandoff候補DOT、`reference/tei-studio-graph/graph.ts` はnode/edge IDと多重辺・Node選択を保持する。ただし**実三視点Source→Loader/Role→Graph/CRUD Projection→元Relation/CellのSource復帰→Studio/SHO実Consumer**はまだ未接続。GraphでEdge select/Field slotのSource復帰は現在のnode-only Referenceと別資格。既存CLI discoveryに三視点diagram/CRUD製品Commandは広告されていない。

**新しい検証前段**：[main統合#3284](https://github.com/hat47x/tei-platform/pull/3284)・[#3296](https://github.com/hat47x/tei-platform/pull/3296)の既存`context_gate.py`は、9 read-only CLI Replyを使用し、不正/欠落Root・古いSource指紋・途中切断・JSON重複Member等の**計16の合成負例を有限CIで判別済み**（[#3296固定HEAD CI](https://github.com/hat47x/tei-platform/actions/runs/42371310577)）。しかしRequiredの中心は`semantic-data:sales.order`で、Function/Business/Rule/Relationすべての閉包ではない。Diagram/CRUDの`complete`やテスト生成可へ昇格させず、現行Gateへ対象Requiredを拡張しても実Loader/Consumerが支持した範囲だけ資格化する。第二Packet/Runnerは作らない（重複PR #3293は未マージClose）。

最初は(1)合成FixtureのTEI Source/Loaderへの**表現可能部分**を特定、(2)型付きGraphの同端点多重辺・同名異IDの保持とSource復帰、(3)Function×Data CRUDで`declaredCrudComplete=false`を未知と表示、(4)Human/AI/Studio/SHOのConsumer別受領／編集の資格を検証。JOIN 2×3の6表示行≠実体6件。独立ExpectedとMutant評価は#2514/#1180を利用し、[#3236設計§13](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のDGX-01〜04を順次検収する。**新製品Diagram/CRUD/GUI/Provider実行、Go/CLI/Owner受理は今回NOT RUN**。

## 組織意味と全容Model Mapの現行main接続（2026-11-09、設計・実行未受理）

<!-- org-crossview-currentmain-20261109 -->
[組織意味モデルの2026-11-09横断追補](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)で、組織所属／職位／案件担当／業務承認／Host認可／実引継ぎの**異なるRelation Identity**と時間・Owner・証拠を分離する。RQ-036が定めた全容Model Mapの要件は[main統合#3505](https://github.com/hat47x/tei-platform/pull/3505)、read-only型付きProjection候補は[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)、三視点/CRUDの実Relation照合は[Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)に分担。現時点で組織Event/State/Hook、Graph GUI、SHO図の全経路実Productは未証明。Case C1のD1→D2担当変更は、Personの所属変更から暗黙に生成しない。古い承認候補／Grant、Replica Read、遅延・重複ACKを現行Effect/全件Handoffへ流用しない。

**実証優先の補正**：TEI全体の最初のP0-Rは、既存#2586 HJ-09の **order_entry.recent_orders**（Pattern→選択/実呼出Provider→SQL/Rows→Mapping→Function Context→Consumer）。組織Native C1→D1/D2（[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)のinstrumented Connector/E1f）は、その縦断契約の**次の分野別Pressure**にする。#3500の隔離PostgreSQLは既存確認でSKIP、Orgの実Provider/Trusted Scope/Consumer受領は未成立。P0-Mは同じSource/Scopeの型付きGraph・Function/Business・CRUDにread-only Source navigateと欠落/非開示を保持する比較を並行。P0-Cで二者の実受領と独立Expectedを同一Attemptへ統合。P1以降に#1174/#2499の現在Grant/Effect、#1125/#1444のHandoff、#1124のTyped編集・Acceptance/再Projectionを行う。

**所有先・重複回避**：組織Meaning=#2386、Pressure Corpus=#2510、Read=#2586/#2506、Model Map/三視点=#1124/#1125/#3492、Lifecycle/Trait=#3278/#3507、SubSystem分割は#3250、独立Expected/Scenario/Attempt=#2514/#1180/#1163。旧[Draft #3243](https://github.com/hat47x/tei-platform/pull/3243)は現mainより大幅に乖離しているため、旧MASTER_PLAN/README/組織Profileを丸ごと再投入しない。**本追補はdocs-only設計・作業順序**であり、新CI・実Provider/GUI/Host/Owner意味受理はNOT RUN。RQ-036全容MapとRQ-037 Verified Composition BOMの番号整合は別[Draft #3592](https://github.com/hat47x/tei-platform/pull/3592)へ委譲する。

### 組織責務切替の次段受入（2026-11-09、設計・独立Expected案）

<!-- org-cutover-tr-plan-20261109 -->
[組織意味設計のCutover追補](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)に、**所属更新≠Case handlingUnitの担当変更≠実行Grant切替≠相手方Handoff**を実効責務の四象限（旧側/新側がyes/no）として定義。単独責任を全組織へ強制せず、Ownerが共同責任・合議・移行中保留を選択可能にする。二重有効（単独責任時）／両者無効／現在性未照合／A→B→A旧Receipt復活を分離し、ORG-TR-01〜08を追加した。旧Draft #3243のORG-CF/MAPを第二Scenario化せず、#1180/#2514の既存独立Expectedへ差分接続する。

**他レーンの現在基線**：mainの #3540 は通常CLI/Studioの静的Inspector参照選択、#3635 はStudio参照Hookの executed/notEvaluated/skipped/refused を区別。どちらも組織実DB/現在Grantの資格ではない。旧#3496（closed/unmerged）のREA 6.7.0 T0実解析・Artifact監査はSource候補の限定証拠。T0-05は[Draft #4257](https://github.com/hat47x/tei-platform/pull/4257)で6.7.0の単独JSON／6.8.0の7行＋jsonl/v1終端を同じ公開合成JSの別2 Jobで実証（[Actions #42942865412](https://github.com/hat47x/tei-platform/actions/runs/42942865412)）。旧監査・候補投影コードはmainに無く、Owner意味受理/TEI Product Intakeは未成立。#3662のSubsystem固定Owner数理候補は現業務責任や物理Service分割でない。

**優先は据え置き**：全TEIのHJ-09一Function実Read（#2586）を先行し、組織C1→D1 read-onlyとRQ-036 Model Map／三視点の同一Source照合は次段、実Case切替・旧Grant/SoD・二件Handoffは#1174/#2499/#1125/#1444のP1。実行主体と認可Sourceの比較可能なrevisionが証明されない場合は保留し、HookのnotEvaluated/partialを許可やEffectに昇格しない。新Org Status Enum/Authority Engine/Workflow Host/Proof Store/Hard Gateを設けない。本追補はdocs-onlyでORG-TRの実Go/PG/Studio/外部Host/Owner受理は未実行。
## 2026-11-07 標準データ構造と標準表現型の対応（設計・検証候補）

X-TEAのテーブル1:1/1:N→XF単票/明細の対応を、**テーブル非依存の標準データ構造**と**機能から独立して定義する標準表現型**の二つのトポロジー間のMappingへ拡張して検証します。表現型の定義/Paletteは独立してよい一方、用途・対象を持つ具体的な表現インスタンスはFunctionに配置し、Function Data Context/Result Slice/Operationと明示的に結び付けます。データ側のEntity/Relation/Occurrenceと表現側のPanel/Tab/Table/Tree等の配置を同一視しません。

- **B・F・G領域の横断**：[#2495](https://github.com/hat47x/tei-platform/issues/2495)が構造Facet、[#2497](https://github.com/hat47x/tei-platform/issues/2497)が構造写像・情報損失・逆写像、[#2506](https://github.com/hat47x/tei-platform/issues/2506)/[#2507](https://github.com/hat47x/tei-platform/issues/2507)がFunction内Binding、[#997](https://github.com/hat47x/tei-platform/issues/997)がTarget別Presentationを担当します。第二のValidator/Renderer/Host/Scenario正本は作りません。
- **時間・媒体の追加検証**：元データの時刻、取得時刻、再生/生成時刻、文書の段落/ページ位置を分離。音声・動画・文章・ページ間では、軸を保持する写像、明示的な時刻変換、新たな再生時間軸の生成を区別します。朗読時間やreflowされたページ番号を元データの本質的な時間とみなしません。
- **順序と完了条件**：P0=分類比較、P1=Headless read-only mapping、P2=Function/Rendererへの配置、P3=逆編集/実Provider Effect、P4=SHO/SUI等の異質Consumerへの展開。先にRP-01〜08（JOIN fanout、DAG→Tree、数値→Slider、音声/動画同期、文書→TTS、Sonification、partial Tab、Function跨ぎ）の独立Expectedで誤実装を落とします。既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)へ接続し、新Gateは追加しません。

詳細は[設計・反証比較票](design/STANDARD_REPRESENTATION_TOPOLOGY_AND_MEDIA_AXES_2026-11.md)。**今回の追加は計画のみ**であり、RP-01〜08の実行、実Media生成/再生、Provider/Host接続、Owner意味受理、公開契約の採用を示しません。既存の判定段階A/B/Cの評価は変更しません。
## 2026-11-07 表現型間の選択・状態連携（検証候補）

標準データ構造と標準表現型を独立させ、Function内に表現Instanceを配置する[前段の設計Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)に対し、**複数Viewの同期が業務対象・取得Scope・書込み資格を暗黙に共有する誤り**を検出する計画を追加します。前段PRは本節記載時点でDraft・未統合であり、正式採用済みではありません。

- **設計上の差分**：Data Subject／Relation occurrence、Query Result occurrence／grain、Representation occurrence（行・Graph/Tree経路・集計点）、Context Instance／selection epoch、操作Intentを区別。Panel/Tab/Canvasの配置・Focus・Zoom・SeekをDomain Mutationへ自動変換しない。異なるViewへの同期は用途・粒度・出現ID・Read Scope／Disclosure・現在の版に従い、exact／複数候補／aggregate関連／unknownを区別する。
- **既存責務を活用**：[#2506](https://github.com/hat47x/tei-platform/issues/2506) Context/Result Slice、[#2507](https://github.com/hat47x/tei-platform/issues/2507) Binding、[#2497](https://github.com/hat47x/tei-platform/issues/2497) Projection、[#2582](https://github.com/hat47x/tei-platform/issues/2582) Change/Refresh、[#2500](https://github.com/hat47x/tei-platform/issues/2500)/[#2598](https://github.com/hat47x/tei-platform/issues/2598) 更新Intent/Request、[#997](https://github.com/hat47x/tei-platform/issues/997) Rendererに分配し、第二のState Engine/Authority/Scenarioを作りません。
- **受入順**：C0 MV-01〜04（Table/Form、DAG→Tree、Aggregate、partial）Headless → C1 MV-05/06（Selection ABA、非開示の同期漏えい）→ C2 MV-07/08（SUI Canvas・SHO文書・朗読区間）異質Consumer → C3 既存Function→実Provider Read/Action/Receiptの縦切りへ接続。独立Expectedは[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)が所有します。

[表現型間連携・変更影響の設計比較票](design/REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md)を参照。**今回の成果は計画・比較票のみで、MV-01〜08、実Renderer/Provider、Owner受理はNOT RUN**。既存Gate A/B/CおよびProduct実装状況の判定には反映しません。
## ドメインモデル由来のテスト生成と開発者支援（設計・未実装）

TEIの三要素分析（Data / Function / Business）を、単なる入力データの生成にとどめず、値域・型・強属性の意味候補・業務Key/FD・Relation・業務Rule・手作業/承認を含むProcedure・広域のデータ整合性から**検証義務と正負テスト候補を導出する開発者支援**へ展開する。独立Expected、実Runner、実Provider、Source/Attempt Evidence、Owner受理の正本は増設しない。

最初は#1131の型付きDataset/Snapshot・候補生成、#1126/#2495の制約、#2514の独立Expected、#1180のScenario、#1163の実証Evidenceを利用し、**受理済みConstraintだけを決定論的に検査して正常例・境界値・一因子違反の反例を生成する**。次に#1125の業務手順・手作業/承認、#2506/#2586のContext/実Read、複数Tenant/Provider・変更/同時実行への横断検査を段階的に足す。AIは反例探索と変更影響分析を支援し、業務意味の採否や本番の副作用は実行しない。

**「強属性」の原典定義は未受理**であるため、業務Key・FD・必須属性と自動同一視しない。未取得、業務NULL、非選択SQL NULL、部分取得、異Snapshot、非公開条件の照合も区別する。詳細と16件の独立対照候補、P0〜P2の開発順、実行済み/未実行境界は [ドメインモデル由来のテスト生成・開発者支援計画](https://github.com/hat47x/tei-platform/issues/4134#issuecomment-6772836751) にまとめる（Draft、**生成機能は未実装・NOT RUN**）。

### 2026-11-08 標準表現型の残差：実Source入場と共通Consumerを最小検収へ接続

[合成写像の検収§20](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)へ、mainで実証した限定Factと未接続Edgeを統合した。**第1〜13章と基本P0-M/R/Cは現mainに既に存在するため二重導入しない**。

- **Source**：main #3482は13実CLI Reply／42合成Consumer Controlsで`sales.order.submit`の宣言・Application・Rule/Source位置を限定確認したが、Product Function→SQL/物理Provider実Readを証明していない。main #3477は同Workspace Source-only改訂を検知。Import Provider原文だけの改訂は未マージ#3479が別の反例として扱う。Managed Markdown（#3346）は`readByLoader:false`で、物理FK/Index（後継Draft #3486、旧#3372は未マージClose）は業務意味の**Candidate**に留まる。
- **Consumer**：main #3403にread-only MCP参照Adapter4 Toolが存在し、ADR-0048 (#3439)はCLI/MCP/API/Web/Desktopの共通Operation方針を採用。#3458は`validate`のCLI/MCP正常・失敗の局所Parityを検証。残るTool、実Web/Desktop/外部H3/Hostの受領・効果・現在Grantまで同値とはしない。
- **最初の縦切り**：実TEI Source→read-only型付きGraph/Relation locator（P0-M）と、同一Product Function→実SQL/Provider→Data Context/Form/Table（P0-R）を並列実証し、同じSource/Scope/Revision/Read Cohortが観測できた部分だけP0-Cで結ぶ。Graph/CRUDの未取得はunknown、物理観測は未受理Candidate、参照CLI/MCPの値一致は現在の実Invoke/Effectではない。
- **担当と停止線**：#3263三視点、#2586実Read、#3262必要Source、#3479 Import鮮度、#3486 Workbench候補、#1158/#3420共通API、#2514/#1180独立Expectedを再利用。IA-01〜04は一因子の負例候補。新Source Registry/Parser/API/Host/Runnerを作らず、追加Product E2EとOwner意味受理は**NOT RUN**。
## 2026-11-07 組織表現と業務手順の非自動接続（#2386/#1125、受入提案）

[組織設計§84.6〜84.9](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)へ、組織の**Data/Relationトポロジー**、再利用可能な**標準表現型**、**Function内のPresentation/Contextインスタンス**、一階層の**業務DFD（業務間データ授受）**、内部**Business Procedure**、実**Workflow Instance/Host/Provider Effect**を分ける最小受入を接続した。画面Treeの部門後継線、案件担当Pickerの選択、DFDのデータ授受矢印、Procedureの引継ぎStep、実Case移管と人手の受領は、似た図形や同じC1/D1/D2でも同じ出来事ではない。

**具体的な検証は既存Ownerへ委譲**：#2506/#2507/#997は組織Tree/TableとCase Panel/PickerでFunction ContextとTarget・Scope/epochの保持を検査、#2500/#2598はViewChange→受理済みAction/実Requestの不一致を検査、#1125/#1444はDFD→Handoff/Procedure（人手/外部/機械）の**read-only対応**を検査、#1174/#1163は現在Grant/Target・Effect/Receipt/独立再読取を所有する。既存ORG-SU/SM・APP/TB・VE/RITの独立期待を再利用し、新Workflow/Policy Engine/専用Org Core型は設けない。

対象集合を使う業務完了判定では、#3238が扱う重複ID・TOP1・非出力Predicate、mainのCompletion APIへ渡すCaller提供Coverageの独立資格を§84.8で追加確認する。Receipt再照会だけで保存後資源の独立Readや人手引受を確定しない。

#3236の組合せ表現では、Data→Function View→SHO/AIの各写像が局所成功しても全体のRelation/Scope/開示を保証しない。#1129の**Data-attached Rule Attachment**はMembership発生やCaseの直接Relationへ明示Bindingし、read-time導出・validation・before-write・Tx内効果・post-commitを区別する。表示変化やSQL JOINの重複を発火・完了の証拠にしない（§84.7/84.9）。

Sourceの論理Workspace IDと物理root/realpathも別資格とし、Manifestの名前が同じでも実読込Sourceの位置・内容・現在許可範囲を検査する（組織設計§84.5.1、#2823/#2605/#2017）。

**製品E2の優先は不変**：main統合#2681のNative組織E1を起点に、[#2586](https://github.com/hat47x/tei-platform/issues/2586)で**一件のC1→D1を実Function→Query→Provider Read→Consumer**まで同じSource/Attemptで検証。旧[#3024](https://github.com/hat47x/tei-platform/pull/3024)と設計PR[#2389](https://github.com/hat47x/tei-platform/pull/2389)は**closed・unmerged**であり、その旧CI/未マージ草案から現main Product E2 PASSやOwner意味採用を主張しない。ここでの追補は**docs-only／独立期待・接続案**で、組織二Functionの実UI・実DB/Host・Procedure Instance、正式Owner受理はNOT RUN。

## 2026-11-07 組織トポロジー・単位系と標準表現型の横断受入（#2386/#2495/#2593、採用前）

[組織設計§85](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)と[組織トポロジー・数量・表現型合成Profile](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)に、**同じ組織の多軸Graph/多項Relation/時間・空間と、人数・FTE・予算・工数等のQuantity Kind／Unit/Basis／Rollup lawは直交するが互いに適合条件を持つ**という比較票を追加。組織を専用Tree Core型へ限定せず、#2495/#2544/#1705/#2593の一般Facetを組み合わせ、組織ProfileはRelation Role・用途別多重度・集計規範の不足分だけを与える。

独立Expectedの最小例は合成P1(A1 D1=0.6 FTE,A2 D2=0.4 FTE)、P2(A3 D1=1.0 FTE)。**Unique Person全体2人／Membership3件／FTE2.0**を混同しない。H→D1→CとH→D2→Cの二Pathは、Cの同一0.4 FTE Factを自動で0.8へ増やさない。現mainの`Membership.allocation std.decimal`はFTE Unit/分母・60と0.6の換算Ruleを宣言していないため、合成期待値を実Sourceの業務計算済み資格へ昇格しない。#2510/§16/39のTopologyケース、#2593 MC、#2497 CX、#2514の独立Expectedへ差分を引き渡し、第二テスト正本を作らない。

**表現型適合**はGraph、Tree、Matrix、Table、Chart、Timeline、Map、SHOの平易なMermaid風テキストを同じDataから構成し、用途/Scope/Unit/Path/時間/Disclosure・Lossと逆編集資格を別々に検証する。TEI表現インスタンスはFunction Data Contextに所属し、SHOは別Consumerとして再資格化。Readableな図の生成成功≠元Relationの再構成可能性／現在Grant／実編集Effect。標準の表現型と専門Profileを分離し、新Renderer/Topology Engineを組織専用で複製しない。

**三要素・図面・CRUD横断の受入差分**：[組織合成Profile§7](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)で、Data/Function/Businessの図面・SHO図解・CRUD等クロスリファレンスを同一Semantic Sourceから導出するときの意味、必要Source/Role/Quantity、Source逆編集資格を明示。合成例Org Explore／Case Handling／Workforce Report／人手Handoffでは、Rは**宣言された依存**、U?は**未受理の操作候補**であり、実Read/実Effect/Host Grantではない。Graphで二経路へ出現する同一C、D1のFTE集計セル、DFDの移管矢印、CRUDのセルは自動で同一更新対象にならない。ORG-XV-01〜07（軸だけ変更、FTEだけ変更、Path多重、DFDのみ変更、集計セル編集、非公開差、ABA）を既存#1124/#1125/#1803/#2497/#2593/#2514の検証へ接続する。モデル図面とCRUD表は派生表示であり、**一意なSource対象を持つ型付き変更候補だけ**既存Preview/Acceptance/reparseへ渡す。C0/C1の未カバー範囲は#1180/#2514の既存独立Expected/Scenarioの側で判定し、新Org専用Runner/Canonical/生成器は作らない。

**組織モデル由来の独立テスト義務**：[組織Profile§8](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)に、他チャットの未統合テスト生成Draft #3258〜#3261を組織の具体的反証へ接続した。ORG-TG-01〜08はHC≠Membership≠FTE、二Path配賦、軸だけの変更、欠落した集合、非単射図編集、DFD≠Effect、Public/Private差とABAを、**手で固定した独立Expected＋既知の誤実装＋常時unknownを防ぐ正例**で評価する候補。#1131のDataset候補→#1180のScenario正本→#2514の独立検証基準/Mutant→#2811のAIテスト入口→#1163の実Attempt Evidenceを再利用する。**コードC0/C1と、受理済み業務ObligationのCoverageと、Mutant判別力は別分母**。単なるAI生成ケース増加・全SKIP・未実行は充足に算入しない。現Native `allocation std.decimal`だけからFTE意味/単位系を自動採用せず、必要条件不明なら候補のまま停止する。新Org Test Generator/Runner/Expected Store/第二Scenarioを採用しない。

**優先順位**：#2586の実C1→D1 Function→Query→Provider Read→Consumerを先行し、その後P0合成のGraph/Headcount/FTE、P1二Function/SHO表示、P2原本への一意なIntent/Host Effectを段階検証。#2506/#2507/#2516/#2510/#2497/#2593が対応Owner。新Go/DB/GUI/SHO実行、Owner Quantity/Meaning受理、実Provider実証は**NOT RUN**。既存の組織計画§84・§85と並行採用可否を評価し、無根拠な大量Core新設やP0実Readの延期を避ける。

## 2026-11-08 組織Product E2の三ゲートと汎用Referenceからの証拠移転禁止

[組織Profile §9](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)へ、他チャットの#3263（三視点図/CRUDの実Source生成）、#3330（P0-M/P0-R/P0-Cの独立ゲート）、#3262（Design Context必要集合の取得完全性）、#2586（実Function→Query→Provider→Context）を、**単一Ownerを維持した実証依存**として接合した。P0-M（三視点Diagram/CRUD Source provenance）とP0-R（実行された物理ReadとRow/Field/Scope）は並行可能。P0-Cは同じMeaning ID／Source版／Snapshot／粒度／許可Scopeのもとで両証拠と独立Expectedを照合したときだけ成立する。図面表示PASS・Local Completion Holds・別DomainのPG PASSを組織E2へ流用しない。

**現Sourceで判明した具体的欠落**：main `reference/org-headless-workspace/semantic/organization/model.tei` には `CaseRecord.handlingUnitId→OrgUnit` の関係があるが、検査済み組織Source/現Operation SurfaceはCase用Function、Query、実Persistence Mapping、Provider Read／Host Grantを示していない。したがってC1→D1は、現時点では**意味を表す合成Expectedであって実DBから読み取った案件ではない**。#2586の既存受注Referenceでの汎用E2検査を先行可能とする一方、組織側に必要なFunction/Bindingを具体化し、P0-C接合前は `product-read-not-connected` を保持。何が実Workspace全体に存在しないかは#3262の範囲付き探索を経ずに断言しない。

**受入報告はM/R/Cの各状態と未解決を独立記録**：Model/Diagram Source、実SQL・args・Provider・row IDs/fields・read extent、必要母集団/非出力Predicate、Consumer Disclosure、exact HEAD/Attempt、Owner採否。#3238のTOP1・重複ID・隠れたPredicateはR→Cの負例として再利用するが、Caller申告 `CoverageComplete` を完全性のProducerとしない。実組織Provider/DB・三視点UI/SHO・全件業務完了・正式Owner受理は**NOT RUN**。

**実装回帰の追加（2026-11-08）**：[main統合済み #3353](https://github.com/hat47x/tei-platform/pull/3353)（squash `6cf510d6301a1e133ebbe69ad7bf004f03361b54`）で、`EvaluateCompletion`が`EmptyPolicyHandoff`・0件・`EmptyJustified=true`で、引継ぎ受領なしでも`holds`になり得た分岐を修正した。6ケースで`allow/requires-evidence/deny`と受領有無を対照し、確定`fails`を損なわない。**空集合の独立証拠≠相手方の受領証拠**を組織Profile§9.4のP0-C検収へ追加。ただし`HandoffConfirmed`/`CoverageComplete`はなおCaller申告なので、局所Go PASSだけを実Case引継ぎ・製品E2完了へ昇格させない。#3353のPR HEAD `cf3de9a61906167b3b5cc2fbd246e238a5018d5f` の[Repository Health #42391982442](https://github.com/hat47x/tei-platform/actions/runs/42391982442)はcompleted/success（gofmt、go vet、Go全体テスト、モデル検証）。ただし実案件Reader、相手方の受領、組織Product E2、Owner正式受理はNOT RUN。

**全件Handoffの残余条件（組織Profile§9.5）**：Local Evaluatorの`HandoffConfirmed`はClaim単位のboolなので、`L={C1,C2}`に対してC1だけのACK・同じACK二重計上・適用集合不明の一括ACK・旧版のACKでも、Callerが`true`と報告すれば局所的に`holds`を返し得る。#1125/#1444はOwnerが受理した対象別／正確な集合に束縛した一括Handoff証拠`H`を、#2586/#2622は実Read集合`R`を、#2514は必要集合`L`の独立Expectedを所有する。`L/R/H`のID・Scope・Revision・確認者／適用時点の適格性が揃うまでProducerは全件`HandoffConfirmed`をTrueにしない。#3353の6 GoケースとORG-HC-01〜07は別検証段階であり、Hの実読取や組織製品一気通貫は未実施。新専用Ledger/Completion Engineは作らない。

**外部Source・二時点・引継ぎ集合の再資格化（2026-11-08）**：[組織Profile§11](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)に、mainへ統合された[#3482](https://github.com/hat47x/tei-platform/pull/3482)の**限定Function Design Context（実CLI 13 Reply／42局所Control、宣言上の必要集合のみ）**と[ADR-0048](adr/0048-shared-application-api-and-transport-adapters.md)／main [#3403](https://github.com/hat47x/tei-platform/pull/3403)の**共通API／CLI-MCP Adapter**を、組織C1の次の未接続辺へ接合する。前者はCapability宣言を実Function→Query→ProviderへBindingした証拠でなく、後者の同値JSON・Tool広告は実Consumer受領／現在Authorityではない。

[#3380](https://github.com/hat47x/tei-platform/pull/3380)の二時点合成（現HEAD Node CI [#42462892409](https://github.com/hat47x/tei-platform/actions/runs/42462892409)成功、main未統合）、外部Provider Source-only driftの[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)、複合/並列FKの物理Catalog投影[Draft #3486](https://github.com/hat47x/tei-platform/pull/3486)は**別種の観測**。HRが申告する`knownFrom`≠TEI受信時刻、物理FK≠Owner受理Relation、Consumer World一致≠Imported Provider原Sourceの現在資格。Owner必要集合`L`／実Read`R`／対象に束縛された受領`H`と、必要Design Source`D`・Provider/Manifest観測を照合するORG-EC-01〜07を§11に配置した。正式Scenario/Runnerを複製せず#3262/#2586/#1125/#2514へ渡す。

**優先はP0-R一件Case実Readのまま**。組織実Case Function/Provider Binding・同一Snapshot観測・複数Consumerの実受領、全件Handoff/正式Owner受理は未成立。局所GoやSynthetic NodeのPASSをOrg E2/業務完了へ流用しない。

<!-- org-cf-plan-20261109 -->
**2026-11-09 組織改編・承認の時間整合差分（設計・未実行）**：組織Profile [§13](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215) にORG-CF-01〜08を追加。C1→D1/D2の担当切替・旧Read・失効Grant・Replica遅延・比較不能revision・動的職務分掌・重複ACK・遡及Source訂正を、業務valid/known、TEI observed、実Provider Scope/Read Cohort、Business Rule、現在Host Grant、相手方Handoffの**別証拠**で判定する。異なるAuthority domainの時刻／版の大小を因果順序とみなさない。**#2586の一件P0-R実Readを先行**し、ORG-CF-01/04の読み取り観測を既存Attemptへ添付。その後、実行時再確認は#1174/#2499、受領と時間訂正は#1125/#1444/#3380、独立Expectedは#2514/#1180へ渡す。新Token、Global Clock、Authorization Engine、Provider、Gateを作らず、CF新テスト・実DB/Host・Owner受理はNOT RUN。
### 2026-11-08 TEI／SHO／KONの境界とHeadless／静的公開の最小受入（横断統合・KON実装は未着手）

TEIを静的サイト生成やブラウザ利用へ拡張する際に、**「TEI全体のWebアプリ化」「KON経由の必須Gateway」「標準UIを経由しないFrontend禁止」を採用しない**。これはWeb利用の否定ではなく、次の独立した仕事と責務を混ぜないための方針である。

| 役割 | 最初に所有する事実と境界 | 今回確約しないもの |
| --- | --- | --- |
| **TEI** | Source/Accepted Meaning、標準Data・Function/Operation・Trait/Provider Contract、生成Artifact、CLI/Studio等の型付きApplication API。既存ADR-0048に従いAdapterのTransportと意味Operationを分離 | TEIの全Authoring/Runtimeを新しい中央Web Hostで動かすこと、各WebページにTEI製Renderer/JSを同梱すること |
| **SHO Composer** | 自然言語文書・図表・引用を構成し、外部SourceのIdentity/Revision/受理済み部分を保持するDocument Consumer。分散・自由な執筆を妨げない | 文書が参照するデータのDB真値や現在Grant、TEIモデルの意味受理をSHO文面から自動確定すること |
| **KON Dataspace**（`hat47x/kon-dataspace`） | 足場のあるデータ保持・共有・版付き参照・公開を必要とする場合の**選択可能な**共通データ基盤。KONの実提供範囲は別リポジトリの実証で決める | TEI/SHO/SUIの必須サービス化、全Sourceの一括移管、別の意味正本・一元認可をTEIへ導入すること |
| **Web/Native Frontend・配信先** | Route/Brand固有のUX、DOM/Router、配信/SEO/SSR/Client・Edge/CDN、外部の実閲覧。B2Cでは完全Headless利用を正規の使い方とする | 標準表現型への強制マッピング、既存Commerce/CMS/決済の置換、TEIが配信状態を自己申告するだけで公開完了とすること |

**拡張契約の語彙を混同しない**：Hookはライフサイクル上の拡張点、Traitは満たすべき原子的な実装義務、Providerは選択・組込み・実呼出しが必要な実装、Pipelineはそれらの実行順・遷移である。静的サイトGenerator、Publish Adapter、Web編集UI、B2C Journeyはそれぞれ必要なHook/Operation/Providerを既存Contractへ載せる。図表で記述したLifecycleやWebチャット型Interactionも、Diagramだけで稼働と認めず、Generator入力→実Target/Hostの選択されたAction→Effect/Receipt→独立Readで実証する（#3503/#3278/#951）。Go単一バイナリのローカルTEIを基本経路として保持し、KON常駐やBrowser実行を導入前提にしない。

**利用形態は要求に相対的に選ぶ**。技術者はローカルTEI CLI→既存Git/CI→静的Artifact→外部配信を選べる。一方、Git運用を前提にできない執筆者向けには、**編集・Preview・Revision・公開申請・履歴参照等の限定操作面だけ**をブラウザへ出す案を検証する。その際、TEIの意味/Build処理を共通Operationに委譲し、保存が必要ならKONまたは既存CMS/ストレージをProviderとして接続する。BrowserがTEI製品の第二Modeler/Second Host/Authorityになることを前提にしない。B2Cの完全Native UIはTEI Core/Rendererをロードしなくても業務Action Contractを使えることを条件にする（既存[#3509](https://github.com/hat47x/tei-platform/issues/3509)／Draft[#3512](https://github.com/hat47x/tei-platform/pull/3512)のOwnerへ圧力を返す）。

**公開成功の判定は4段階**：①対象Source/Meaning/依存とDisclosure資格に対して生成・PreviewしたArtifact、②Host/Providerによる保存・Revision受領、③独立したPublish／Deploy ACK、④配信URL等から別Consumerが受け取った外部実Byte/版。Git push成功、SHO本文の保存、KONへの格納、Build Artifactの存在だけで④をPASSにしない。異なるSourceや外部Provider Raw/Manifestが変わった場合、World FP一致だけで公開物・引用・BOMの鮮度を継承しない（[#3477](https://github.com/hat47x/tei-platform/pull/3477)／[#3479](https://github.com/hat47x/tei-platform/pull/3479)／[#3490](https://github.com/hat47x/tei-platform/pull/3490)）。認可・公開範囲・未取得・partial・not-loadedを空集合または既存Grantと見なさない。

**既存Scenario/独立Expectedで検証する最小の横断反例（未実行）**：

| ID | 試行と独立に固定するExpected | 既存Owner |
| --- | --- | --- |
| **KW-01** | 同一受理Source→TEI CLI静的生成→Git/外部配信の実Byteを再取得。Build・Publish・公開後Readを別版・別証拠にし、外部未反映ならPublish完了を拒否 | #943/#1163/#1180、配備は対象Host |
| **KW-02** | Browser編集面で既存の型付き保存Operationを呼ぶ。KON/ストレージが未選択でも、別適格Providerがあれば機能を拒否しない。権限外/Version staleは拒否。UIからSource Meaningを再定義しない | #1158/#951/#1016、保存・公開Host |
| **KW-03** | 外部Providerの引用Rawだけ変更、TEI World FPは同一。SHO引用Revisionと公開物の再利用資格を差分として失効。必要Sourceがunavailableなら未取得を明示 | #2044/#3262/#3479、SHO Composer |
| **KW-04** | Native B2C Routeは標準Renderer/KON/TEI Browser Runtimeを搭載せず、外部Commerce ProviderのTyped Operationを利用。零の追加Runtime依存と実Effect/Consentを別に測り、CVR向上は実験なしで主張しない | #3509/#1158/#1174/#2514 |
| **KW-05** | 同一Function/SourceのCLI・Studio・限定Web操作で**選択Provider Bと実呼出B**を突合。B選択→A実呼出（結果同値）を失格、未受理/未Loadedはunknown。表現が同じでも逆編集の一意性を仮定しない | #1251/#951/#2497/#3267、既存D1受入 |

**高度B2Cの変更レーンと検証順（[#3509](https://github.com/hat47x/tei-platform/issues/3509)／[設計Draft #3512](https://github.com/hat47x/tei-platform/pull/3512)から受領）**：施策・Creative・Presentationだけの差分は、業務意味・Provider・契約・Host Grantに影響しないことを差分で証明できる場合、無関係なMeaning再受理を要求しない。一方、価格・在庫・決済・同意・注文Effectに触れる変更は通常の業務Operation／Effect／Disclosure／独立Expectedの再資格化を省略しない。実際のVariant配信、実表示・Intent、権威あるOrder Effect、計測対象・同意・帰属、採用・撤回は別の観測段階とする。画面のRollbackが実注文の補償になるとは扱わず、CVR向上は実験による独立測定の前には主張しない。

**B0〜B4の圧力試験**：B0=Native HeadlessのRead-only Fit/Gap、B1=独自FrontendからTyped Operationを実Provider経由で呼び別Consumerが実結果を再読取、B2=一部標準UIとの比較とExperiment/Publish/Browser・Cache負例、B3=AIチューニングの統計品質・独立Benchmark、B4=外部サービスAdapterとしてのPlugin採否判断。初手はECの一操作とし、予約・会員等は後続の異質Consumer候補。KW-04との独立Expectedを#2514/#1180へ、仕様・業務効果のOwnerを#3509へ返す。B0〜B4は全件未実行であり、#1251の通常CLI/Studio S2、#2586の実Function ReadのHard Gateを増やさない。

**実装された範囲と横断判断を更新する（2026-11-09）**：[main #3579](https://github.com/hat47x/tei-platform/pull/3579)はFunction定義書へHook宣言を既存Phase順で導出する実GeneratorコードとGo試験を統合した。ただし**宣言が出力されることと、Runnerが実Targetで実Hookを呼んだことは別**であり、Provider実呼出・実Effect・B2C配信を認定しない。さらに[main #3549](https://github.com/hat47x/tei-platform/pull/3549)／[#3588](https://github.com/hat47x/tei-platform/pull/3588)でC-1 Router／C-2 ServiceCarrier／C-3 Deployment Binding／C-4非接続を選択前の比較候補として保持し、C-2は試験外の限定Reference組立に進んだ。**TEI/KON/Headless全体のためにC-1/C-2を両方常設したり、中央Gatewayを必須化したりしない**。どの入口を製品化するかは#951/#1016/#1251と実二Consumer・Host現在権限の証拠で別判断する。公開CLIとStudioへ静的参照Inspectorを配線する[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)はS2-referenceの実装候補だが、正式Profileでの任意Provider切替と実Target/BOMのS2-productは未実証である。

**実行順は現行P0を変えない**：まず[#3378](https://github.com/hat47x/tei-platform/pull/3378)の既存CLI/Studio直結Baseline、main[#3371](https://github.com/hat47x/tei-platform/pull/3371)のHost選択Service、main[#3381](https://github.com/hat47x/tei-platform/pull/3381)のComposition選択Inspector（局所Go二Consumer実呼出まで）を、**通常CLI dispatchと実Studio Bridge双方の実Invoke**へ接続（#1251のS2）。並行して#2586の実Function→Query/Provider Read→Context/表現を検証する。KW-01〜04はこの結果から必要な小さなConsumer/Targetだけ実験し、KON全製品・B2C全機能・静的サイト配信サービスを新Hard Gateにしない。Performance/AIのHint→統計→独立測定はmain[#3520](https://github.com/hat47x/tei-platform/pull/3520)の限定Referenceを再利用し、計測資格なしの最適化やCVR効果を成功とは呼ばない。

**証拠状態**：本節は横断した責務・受入の計画整理。KONリポジトリ作成は実Web/保存/公開APIの実装証拠ではない。KW-01〜05、TEI Web限定面、SSG外部公開、実Native B2C、KON/SHO/TEI一貫実行・Owner受理は**NOT RUN**。新Core、第二Host／Registry／Resolver／Auth、独立Scenario正本は追加しない。

### 推移的Record Scope × 担当変更の接合（2026-11-09、設計のみ）

<!-- org-visibility-cutover-plan-20261109 -->
既存[組織意味設計§58](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)の**用途を指定した広域Org Policy・明示例外・Read-only**（合成[PR #2781](https://github.com/hat47x/tei-platform/pull/2781)のVP/VE）と、ORG-TRの**Case担当切替**を新ORG-VTR-01〜06で交差検証する。Membership異動≠CaseのR1→R2変更≠現Read/Update Grant≠Handoff。旧D1のCache/表示、D2の新関係、A→B→A復帰、適用除外/not-applicable、共同責任・多重Pathによる件数増加、Host/Provider行Scopeの未確認を同一Caseで判別。広域Policyの「既定適用」は**Ownerが対象指定したData/用途内**だけであり、任意Tenantへの既定許可ではない。

**既存工程**：HJ-09の最初のProduct Function/Provider read-first (#2586)は維持し、組織C1の次段P0-Rで#2506/#1656のRow Filter実施行点、SQL/実Rowset・Scope、List/Count/Page/Report/Export、Consumer/Cache現在性を独立照合。Current Grant/Effectは#1174/#2499、Model Map・UI/SHOからの開示は#1124/#2516、独立ScenarioとExpectedは#1180/#2514へ。Providerに行Scopeを強制できない保護Routeは拒否候補とし、UIで隠すだけで合格にしない。合成VP/VEのfixed-head PASS、mainのInspector/Hooks局所実装、実PostgreSQLの別Operationの成功は、Org Caseの実Policy Enforcement/Consumer/Owner受理を意味しない。第二Policy Engine/Host/Test Runnerを追加しない。**新VTRの実行・製品適合はNOT RUN**。

### 組織CaseのSource→実Read→Decision候補→Effect→Handoff：証拠を接合するが、権限と正本を増やさない（2026-11-09）

<!-- org-evidence-join-plan-20261109 -->
[組織意味設計の新追補](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)で、S（Source/Meaning）・Q（選択/実呼出Provider＋実Rowset/Cohort）・D（受理済Rule相対の判断候補）・E（現Principal/Grant・OperationID・Commit/Receipt）・H（必要L/実Read R/対象ACK H・実Consumer）を**別の同一性と資格**として整理。Case ID/World FP/同名・別Run成功の寄せ集めで、同一Snapshot/Authority/Attemptを捏造しない。遡及訂正は「訂正後の履歴解釈」を変え得るが、過去の実Effect/送信履歴を消さず、旧Receiptを現在有効化しない。SUI/外部REAのCandidate→SEI/人間のProposal→TEI Owner-accepted Rule→Host Effectの昇格はそれぞれ別決定で、長期Ontology DSL/Decision TwinはP0 Readの前提にしない。

**最初の製品優先**は引き続き#2586のorder_entry.recent_orders 1 Function/1 Provider/実Consumer。次段組織Case C1では、同WorkspaceのRelation occurrence、Function/Queryとselected/invoked Provider、SQL/args/Scope、Rowsetのpartial/0件、§58のRecord Policy（実Provider施行は別資格）、一Consumerの受信・Disclosureまでを**同じAttempt**で確認。判断・実Effect・L/R/H Handoffは後続P1で#1174/#1125/#1444/#2514/#1180/#1163の既存Ownerへ段階接続する。#3717のPostgreSQL4系統非SKIP成功、#2781の合成Org Policy成功、mainのInspector/Hook実装は各々の限定Witnessであり、Org全体の現在Grant/Provider Enforcement/Client E2Eではない。本追補はdocs-only、新規実Go/DB/Host/SEI/SHOテストはNOT RUN。

### 外部Observationから組織Relationへの候補写像を段階受入する（2026-11-09）

<!-- org-external-observation-t1a-plan-20261109 -->
[組織設計の外部Admission追補](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)で、[REA T1-A前段 #3496](https://github.com/hat47x/tei-platform/pull/3496)の**実REA 6.7.0・固定HEAD 4/4 Job・独立監査＋候補試験42/42 PASS**、valid **13/14**・partial **11/12**の二時点Source byte付きCandidateと各1 unbound Rootを、組織意味の**正式採用前の観測**として接続した。元データは公開JS合成Sourceであり、組織HR/実Caseの外部Producerが動いた証拠ではない。Source前後一致はA→B→AやProvider Actual Inputの原子的Snapshotを証明しない。旧#3496はclosed/unmergedで試験用Projectorはmain未搭載。Model Map #3866のmain統合と足しても外部Observation→正式Model Map/製品CLI実受領は未成立。TEICanonicalSubject=null／acceptedMeaning=false／sourceClosure=not-qualifiedを維持。

**次の欠落辺は1 Observation＋1 Edgeのみ**：既存共通Application API（ADR-0048/#1158）へProvider版・Evidence ID・Node/Edge個別ID・方向/多重性・Source path+raw digest+span・coverage/unknown・Scope/Workspaceを損失なく渡す限定read-only実Consumer。#3394が外部Producer/Consumer照合を所有し、#2386がPerson/Membership/Occupancy/CaseへのOwner意味写像、#3492/#3518/#1124がAccepted/候補を混同しない三視点Model Map、#2516が公開Slice/診断も含む開示、#2514/#1180が独立負例を所有する。ORG-EXT-01〜06で同名別ID、多重Edge、unbound端点、partial 0件、同byte別Workspace、A→B→Aを対照。既存InspectSubjectへ外部JSONを偽Canonicalとして流さず、独立Graph Store/Resolver/Parser/Hostを作らない。

全TEI最初の実Product Read #2586 HJ-09は**並行して最優先のまま**。External候補が得られても、実組織のCase C1 read、現在Principal Grant、Rule受理、Effect/Handoff L/R/H、SHO/SEIの正式受信まで昇格しない。上記外部Candidate→TEI Operation／組織Owner採用／Org Map実表示の新試験はNOT RUN。


### 共有Harnessで製品間を移る際の組織Scope／権限の再資格化（2026-11-10、設計のみ）

<!-- org-shared-harness-authority-plan-20261110 -->
[組織意味設計：ORG-HS-01〜08](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)に、他チャットの**製品共有Harness Session／Activity Contextの分離**を、既存§58 Record Policy・ORG-VTR Case切替・#2516 Construction Context/Consumer Disclosureへ接合した。SEI/SUI/TEI/CSW間で活動の目的、候補Selection、出典、未決、復帰位置は引き渡せるが、**Verified Principal/Issuer/Tenant、現行Grant、Policy epoch、Source実体/版、Recipientへの開示権、Operation Effect/Receiptは引き継がない**。Harness自体をTEIに内蔵せず、製品別UI/APIと既存Adapterを利用する。共有Session・選択済みCase・前Consumer成功は、実Provider Scope/Read/Count/ExportやCommitの許可ではない。

**追加する最小検収**：HS-01移動先のRead拒否／HS-02組織異動後の旧Cache／HS-03 Privateの件数・図・診断漏えい／HS-04目的・受信先変更／HS-05 partial 0件の不存在誤認／HS-06 Issuer/Tenant越境／HS-07 Selection ABA・後着AI応答／HS-08失効中のCommit uncertain再試行。既存#1131/#1180/#2514を独立Expectedの正本として使用し、#1163で同Attemptを照合。成果を新しい全製品共通Session Store、Org Grant Engine、全Route共通Policyの追加要件にはしない。

**順序とOwner**：#2586 order_entry.recent_orders の初回実Function→Provider→Consumer R0をP0最優先に維持。並行して公開合成C1をheadlessでCandidate→TEI read-only Operationへ引き渡す**試験準備**（#2386/#2516/#1124）。後続でCase C1実Provider/Scope、Product Consumer（#2506/#1656）、Current Grant・Effect（#1174/#2499）、ACK/Handoff（#1125/#1444）、Harness実Tool/送出・Log（#2509/#2515）をそれぞれ実証する。通常Org Data全体への秘密指定、全Session切替での一律再取得、旧送達済みCopyの消去保証はしない。

**資格**：ORG-HSは設計／受入候補のみ。共有Harnessの実製品間連携、実Grant、Row Scope Enforcement、Egress、Owner受理、HS-01〜08試験はNOT RUN。既存M0、合成Org Policy、DSHのTool Read、別Operation実PostgreSQL成功をこの縦断のPASSとして集計しない。


### SUI/CSWの探索的理解をTEI組織Meaning候補へ受け渡す（2026-11-11、受入設計のみ）

<!-- org-affinity-admission-plan-20261111 -->
[組織Meaning設計のORG-AG-01〜07](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)で、SUI Card/Island・CSWのKJ法的グループ／仮説と、TEI Person/OrgUnit/Membership/Case担当・承認Ruleの**正式Relationを非同一**とした。許可済みの理解パックは、元CardのSource/Review資格、反証・保留・孤立残差、Candidateと未解決、Purpose/Audienceを保って利用目的別に構成する。組織MeaningへのMappingは任意・非可逆で、Owner受理された部分だけがTEI正本を変更する。TEIの逆編集からSUIの元Islandを黙って書き換えず、SUI/CSWをTEIへ全面移行させない。

**単一の最小試行**：公開合成Card数件をSUIの実許可read-only Projectionから取得→実Consumerへの送達・受領・要約での反証残存を別々に観測→TEI既存read-only OperationにSource-backed Mapping Candidateを照合。SUI現context-projection.v1のSource資格はL0まで（#3368）。元Sourceを取得したかのようにL1/L2へ昇格しない。類似順位候補の成功（SUI #3157）やReviewは組織受理を意味しない。反例はAG-01同名島とOrgUnit／02 C1→D2未受理案／03孤立反証落ち／04L0根拠不足／05生成と実受信の差／06Private公開差／07新材料による再検討を、既存#1131/#1180/#2514で評価する。

**既存Owner・実施順**：探索/実投影はSUI #3368/#3157、用途別構成・開示はTEI #2516、TEI組織意味は#2386、Model Mapは#1124、実Tool/Consumerと不確実性は#1163/#2516。#2586 order_entry.recent_orders のFunction→実Provider→Consumer R0は依然P0最優先。組織C1実Provider/Scopeと現Grant/Effect/Handoffは後続#2506/#1656/#1174/#1444へ。本計画は新Affinity Runtime、共通Session Store、第二Meaning正本、別Gateを設けない。実SUI/CSW→TEI/SHO/SEI E2E、ORG-AG新試験、Ownerの正式採用、実Org DB/Hostは**NOT RUN**。


## 2026-11-12 高度組織モデリング：構造受入から業務責任・三視点・Consumer縦断へ（#2386/#4490）

<!-- org-crosslane-acceptance-20261112 -->

**段階計画の正本**：[高度組織モデリングの段階計画 §10](design/ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md#10-2026-11-12-横断統合組織責任実行三視点知識consumerを一案件で接合する採用前計画)。#2386の意味Ownerと既存Data/Relation/Rule/Operationを維持し、組織専用Core/Workflow Engine/Policy Engine/第二Expected正本は追加しない。

- **基線の訂正**：PR #4342の設計計画、#4399の構造テスト、#3741の責任Cutover設計、#4256のSUI/CSW候補境界、TEI→SHOの#4317/#4467はmain統合済み。ただし#4399の構造test PASSは二時点Query結果・承認候補・委任拒否等の製品上の振る舞いを保証しない。#4479の固定対象AI指示はDraft。
- **次の独立Evidence**：#4490でORG-BEH-01/02（valid/knownの結果差・WHERE/ORDER/EXISTSの非出力依存）を最初に実行。次にDirective/候補/代理/Case移管・ABA。WU-11〜14は子Issue #4501 に分離し、業務手順・Handoff ACK・loss-aware Projection・固定対象と再投影・保護Field例外を既存Ownerへ引き渡す。
- **Case C1の接合**：Membership変更≠Case責任移管≠Rule候補≠現行Host Grant≠Effect Receipt≠Handoff ACK。Source/Meaning、実Query/Read Cohort、Business Decision、Authority/Effect、Consumer受信のそれぞれを同一Attemptで明示的に照合。単なるCase ID/World hash一致は同一資格の証拠ではない。
- **優先順位**：全体の実Provider Read #2586 HJ-09を最優先に保ち、組織C1の合成Behavioral Expectedを並行作成。続いて実Org Rowset/Scope #2506/#2622、三視点/GUI/SHO #1124/#2497/#2516、Host Grant/Effect #1174/#2499、業務Handoff #1125/#1444へ接続。各段階は実測・未実行・Owner未受理を分ける。普通の組織Dataを組織管理モジュールというだけで特別な秘密情報にしない。

**三要素分析法・REAの補強（§10.7）**：X-TEA Modelerの三要素分析法（Data/Function/Business）を方法論名として正確に用い、TEI独自のSource/CRUD/Evidence接合を拡張として明示。Task契機≠Function起動、Function成功≠Business完了、REA Agent≠認証Principal、DB FK/SQL観測≠業務意味受理、Studio/SHO/AI Selection≠Source更新資格をCX-13〜18で判別する。実行計画は#4501、#1125/#1444、#2586/#2387、#1124/#2516の既存Ownerへ配分し、新Core/Runtimeを増やさない。

**組織FTE換算の命名衝突（2026-11-13、§10.8.1/§10.11.3）**：r7 `allocations[].basisPoints` は現行合成データ内でraw 100=1 FTEとする旧フィールド名であり、金融のbasis point（100 bp=1%=0.01）とは倍率が違う。Field名だけで汎用bpへ自動変換しない。#1705/#2495/#2593へUnit/Scale/Basis判別を接合し、既存JSON一括変更や新Unit/Query Engineは先行させない。同一Allocation ID重複は衝突、knownAt訂正によるD1の値変化と同条件D2値不変はClaim単位で分け、実Providerの同一Read cohortを推測しない。新たな合成試験はORG-CAP-UNIT-01/02、ORG-CAP-07b。元の§10.8 r7修正と7件のmain済み試験資格を維持し、追加の現行HEAD実行結果は別途確認する。

**組織の運営能力・数量・Scenario（§10.8、fixture整合修正）**：既存 `reference/org-semantic-views/source-fixture.json` の `org-synthetic-r7`（validAt=2027-06-15、knownAt=2027-07-01）に揃え、P1→D1(A1=1.0)、P2→D1(A2=0.6)・D2(A3=0.4)、P3→D2(A4=1.0)とした。全体Unique Person=3、Membership=4、Allocated FTE=3.0（D1 1.6、D2 1.4）。前稿の2人/3所属/2.0 FTEは既存fixtureにない別仮定だったため撤回。欠員はP1の利用可能配賦相当D1=0.6、仮配賦変更はP2 A2 0.6→0/A3 0.4→1.0でD1=1.0/D2=2.0とし、確定Membership/Case/Grant/Effectを改変しない。knownAtを2027-07-03へ進めた同一validAtではA2が外れ、Membership=3/FTE=2.4。実行可能な既存fixture合成テストを**main統合済みPR #4532**（Node v22.16.0、ローカル7件PASS、main squash `2e52ca47`）に切り出し、#2510/#4490/#4501とExpectedを重複させない。ORG-CAP-06/08、実Provider/Studio/SHO/Host、Owner意味受理は未成立。一方、合成Node Referenceは#4532 head `fa2d8ce98ff77536f51e32095748f63d935a2d07` の [Actions #43242282586](https://github.com/hat47x/tei-platform/actions/runs/43242282586) で313/313件PASS（新ORG-CAP 7件を実行ログで確認）。これは製品上のQuery/Provider/Grantを証明しない。#2495/#2593標準Quantity、#2586実Read先行、#2386 Rule/責任、#1124/#2516投影/開示、#1174/#2499実EffectのOwner境界を維持。

**組織C1のSchema/UI Mapping変更追補（§10.9/§10.9.4）**：実テーブル定義とDesired Schema、各版の**テーブル定義モデルFieldそのもの**、上位の業務意味/Computed Field、Function/UI要素とBindingを分離する。正本でFieldが削除されたDesired版、または完全観測で削除を確認したActual版では、**その版のテーブル定義モデルからFieldを除去**する。一方、UI要素は残して未マッピング警告、CRUD/派生Query/SHO/AI対象の依存を診断する。外部DBの変化を理由にDesired Sourceを無断更新しない。partial/unavailableを削除確定とせず、同名再作成・Schema ABAで旧Bindingを復活させない。画面追加だけでは列を作らず、明示的開発者操作→Preview→Migration/Grant→実DB再観測へ進める。GUI/CLI/実Targetは同じ操作意味。#4515の**Subject Pin先行**と描画Edgeの非意味ID・Relation Pin unsupportedを織り込み、#1143の横断変更資格、#2497の逆写像、#4125の旧新Client互換と照合する。ORG-MAP-01〜10は既存Expectedを改訂・再利用（実試験NOT RUN）。#1126/#1124/#3270/#1452/#4479/#4501へ配分し、#2586の実Read優先を維持。三要素分析法は業務向けの方法論/PluginでありTEI汎用Coreを専用固定しない。

**組織モデルの参照・未使用診断とStudio性能（§10.10）**：#4472の三要素CRUD索引/明示定義・矛盾表示と#4475の同Kind同ID/Fingerprint純粋性を先に検証し、その後#1177/#3788の既存Source Index/Traceで順方向・逆方向参照、Source位置への高速移動、Scope完全時だけの「参照数0・未使用候補」を実証。0件/partial/unknown/未解析Pluginを混同せず、静的Unusedから自動削除しない。#4515のSubject Pinを先行、意味Relation ID未確認の描画Edgeから編集しない。大規模性能はcold/warm p50/p95、CPU/メモリ、Index構築と実Query/描画を分離計測。必要なPanel/Pluginだけ遅延起動する案を#1824で比較し、第二Index/Parser/Hostは作らない。WU-12/13とORG-MAP/CXのExpectedを再利用し、#2586実Read優先を維持。実ナビゲーション/性能・Owner意味受理はNOT RUN。

**三要素・REA・組織参照の実証資格（§10.10.4）**：三要素Fingerprint入力破壊はmain統合 #4507 で限定修正済み、CRUDのData索引3箇所は#4513で修正・5 Workflow成功を確認後main統合（squash `7e36f85a`）。明示CRUD-only組の派生表脱落/Unused誤判定は#4551で修正しmain統合済み（squash `1a2271d8`）。矛盾の全件列挙と構造`ok`/CRUDレビュー資格分離は#4534（main merge `5f35d5dc`）で実装済み。ただしStudio/SHO/AI Consumerによる要レビュー診断の正しい表示・適用停止は未検証。CLIのtrace.unresolvedは返却行の未解決、projectionUnresolvedは全Graph投影の脱落を示す（#3906）。一方のみが0でも全体完全とは判定しない。外部REA 6.7/6.8のProducer Protocol検証 #4257 と、実TEI受領・Owner Accepted Meaning・正式CRUD参照は別（#3394）。ORG r7のNode 7件はPR #4532の固定HEAD fa2d8ce.../Actions #43242282586で合成CI成功、実Provider/Studio/Hostの代替ではない。組織参照の次のExitは「Accepted参照／外部候補／未解決・partial」の三群を混同せず、#4472/#4475→#3262/#1177→#3394実Receiver→#4515/#3788の順に既存Expectedで段階確認する。P0 #2586は維持。追加した§10.10.4は設計/現況の照合で、新製品試験はNOT RUN。

**UI起点のデータ文脈設計とClaim別再評価（§10.11）**：業務向けModelerではUI部品から既存Field／新しい永続Field候補／Derived／Transientの4経路を選択し、**新Fieldの明示設計操作**が行われた場合に限ってData Context→Desired Schema→Impact Preview→既存Migration/Grant→実DB独立再観測へ進む。UI配置だけではDDL/実DB変更なし。SQLの`SELECT *`/`alias.*`/`NATURAL JOIN`やWHERE/NOT EXISTS等の暗黙依存も追い、#2582/#4518/#4526のClaim別Required Closureでaffected/scoped-unaffected/unknownを判別。無関係なUnknownで全画面を過剰失効させず、未知の依存を無影響としない。SHO #50 QH-08〜10で同Markdown別Source・TEI実再束縛・実Recipient受領の資格を分離。#4524（同Kind同ID）、#4513（CRUD Data索引）、#4514（SHO図Source-only drift）、#4534（CRUD衝突診断）は範囲を限定した修正としてmain統合済み。ただしConsumer全面Gateではない。#4532（r7合成Node）も正確なPR HEADの4 Workflow成功を確認後main統合済み（squash `2e52ca47`）。ORG-MAP/ORG-CAP/ORG-BEH/CX/PIN既存Expectedの追加assertのみとし、別Engine/Accepted意味正本を作らない。#2586の実Read P0優先を維持。新しいProduct E2EはNOT RUN。

**REAの名称区別と組織/Subsystem責務（§10.7/§10.11.4）**：Resource–Event–Agent（業務・会計の分析モデル）と、#3394/#4322の外部解析ツールREAを別概念として扱う。解析ツールのObservation/Node/EdgeはSource-backed候補であり正式Membership/CRUD/Agentへ自動昇格しない。重複Node/Relation/並行Edge由来の候補ID衝突は任意の一件を採用せず拒否（#4542は親#4322へ統合、#4322はmain未統合）。同一Attemptの実TEI read-only API受領・Owner意味受理は別。組織の所属/Case取扱責任、論理SubsystemのDirect CUD・Owner、実DB配置、Host Grantを異なる軸として、組織改編・Subsystem表示名/IDの付替えだけで権限/責務移転を推測しない。#3250のSP-X23/24 Draftおよび#4501 CX/ORG-BEHの既存Expectedを引き継ぎ、Required/Supplied/Missing Guardと実Attempt/Effect/ACKを別検証（#1129/#1174）。変更影響は#2582/#4518のClaim別Closure、物理Field変更は§10.9/10.11に従う。外部REA/Product/Hostの新しい実接合はNOT RUN、新Core/第二意味正本を作らない。

**実PostgreSQL Readと組織Queryを接続する実装順（§10.10.4/§10.11）**：#2586のmain統合済み #4539（squash `ee1af44c`）は、受注Query `order_entry.recent_orders`を一つのEnvironment/Database Instanceと実PG16.15 Providerへ明示結合し、Raw Rows→Function Data Context→隔離CLI `tools/reference-browse-read` で実読取した限定実装。使い捨てPGで0件、NULL/欠落、RLS、権限拒否、Source SQL差を対照し、該当HEADの7 WorkflowはSUCCESS。ただし現在の製品`tei` CLI/Studio/SHO/AI Harnessには未結線、Env↔実Providerはhost-asserted、coverage/不存在/current Grantは未確立、参照実装のmain統合は確認済み。ただし通常CLI/Studio/組織固有Queryは未完。P0の次の一手は**通常CLIで選択と実Invokeと同一AttemptのQuery/params/Rowset/Context受領を一致させる**こと。最初の実例は受注Queryであり、組織C1のData/Case/二時点業務Queryへの実移植は別受入。#4534のCRUDレビュー資格をHost許可と混同しない。新Runtime/独立Expectedなし、GUI完成をCLI P0の前提にしない。

**採否**：新規Source/Rule/公的契約や業務責任の解釈はOwnerの意味受理が必要。本追補はdocs-onlyであり、実Org Provider/Host・SHO/HarnessとのOrg C1 E2E・Owner意味受理は **NOT RUN**。

**Studio Desktopの実構築→TEIセルフビルド／組織C1受入の最短接続（組織計画§10.12）**：製品の最優先目標は**Wails/Go版TEI Studio Desktopの実OS起動**（#1824）と、既存TEI Typed Build Graphを使った**stage1 TEI実プロセスからstage2 Studioを再生成するセルフビルド**（#2831）。Go CLIのセルフビルドCI成功をDesktop起動・D2再生成成功とは見なさない。**生成Workbenchのserver/Chromiumでの成功と通常Wails Desktop/WebView2での成功も合算しない**。#4541のGD-0〜4に沿い、生成assetをserverなしのDesktopへ組み込み、実WebView2→IPC→Go Catalog/Inspect応答まで同一HEAD・実Process・Selectionで検証する。組織C1のread-only Data/三要素CRUDは#4551 main統合の派生結果と接続し、#4539の実PG受注Queryは通常tei CLI→Studioの別Consumer経路に結ぶ（同一業務Queryだと誤認しない）。X-TEA三要素は業務向けPluginとして移植し、汎用Coreを固定しない（#1124/#1125/#1251）。D0実Desktop、D1 Model/Provider受領、D2セルフビルド、D3 X-TEA Plugin、D4 Pin/SHO/AI適用の各資格を独立Expectedとexact HEADで判定する。#2586の通常CLI Provider実ReadはGUI完成待ちにせず並行P0。#4500/#4551のmain統合は確認済みだが、D0〜D4の実行証拠は未成立。新Build Engine/Host/Scenario正本を作らない。

<!-- org-six-scope-headless-master-20261113 -->
### 2026-11-13 組織意味とPlugin六つの適用範囲・B2C Headlessを接合する（受入設計）

[段階計画 §10.13](design/ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)に、配布（Distribution）・設計編集（Authoring）・アプリ構成（Composition）・組織関係（Organization）・実行（Runtime）・用途別開示（Disclosure）を**別の適用範囲**として照合する方針を追加した。六つは比較・受入観点であり、固定Core enum、第二の認可基盤、TEI独自Hostではない。兼務Personや同一Plugin配布から別Tenant Grantを推定せず、B2Cの購買者をMembershipに押し込まない。専用Frontendが型付きApplication APIを呼ぶ完全Headlessを選べることを維持する。

最初に既存r7組織Sourceと追加の合成Tenant/Consumer/Host Scenarioを区別し、#4490 ORG-BEH、#4501 CX、#2903 SCOPE/JOIN、#3509 B0〜B4を**第二Expected正本なし**で横断照合する。#2586の通常 tei CLI→実Provider Read→一ConsumerをP0として先行させ、その後Org C1の実Queryと複数Consumer、Host現Grant/Effect/Receipt、Disclosure非干渉性を既存Ownerに引き渡す。組織Dataを無条件で秘密にせず、保護を要する項目と接続先に応じて制御する。UI選択・AI提案・Plugin配布・局所Node/Go PASSは実Readや現在Grantの代替としない。

**状態**：設計追補のみ。実Org C1 Provider、実Multi-Tenant/Headless B2C、Host/Disclosure、CI、Owner意味受理は未実施。TEI Desktopセルフビルドと初回実Readの優先順位を変更しない。

## 先に固定しないもの

- 最初のWeb向け実行先で用いる開発基盤
- 表示に関する完全な中間表現
- 実行先の機能の完全な分類体系
- 編集環境の最終的な画面構成
- AIの提供元
- ベクトルデータベース
- 変更提案の最終的なスキーマ
- 人が記述する安定識別子やUUIDの必須化
- 差分の適用だけを認める記述方式
- WailsをTEIの標準実行基盤とすること
- 二つ目の実行先の種類
- 一律の性能基準値
- 汎用の業務処理の実行機構
- 本番運用の高可用性を共通機構として扱うこと

参照事例や検証根拠によって必要性を確かめるまでは、これらの方式を固定しません。
