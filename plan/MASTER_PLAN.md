# TEIの開発全体計画

## 目的

この文書は、TEI（鼎／テイ）の製品価値と要求を開発上の作業領域へ整理し、優先する作業と並行できる作業を示します。

期限は設けません。優先順位は実装量ではなく、**TEIの中心価値に関する不確実性をどれだけ減らせるか**を基準に見直します。

過去の判断を本文に重ねて記すのではなく、現在の計画の基準を示します。以前の判断とその理由は、Gitの履歴、設計判断記録、課題からたどれます。

関連文書は次のとおりです。

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

### トポロジーと複合図表の評価順（未採用の追加検証候補）

[トポロジー・複合図表・開発パイプラインの共通表現](design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)を、既存契約を重複実装しないための横断的な設計・検証候補として参照します。現行のIssue優先順位、製品受入条件、実装済み機能の説明は変更しません。

最初は既存の型付き参照・Graph・Projection・Build/Compositionを棚卸しし、同じ構造からGraph・表・関係表を読み取り専用で投影します。次に、対象版とscopeを固定した実効パイプラインの解決完全性を評価し、完全性が証明できない場合に図表上の欠落を「不存在」と誤認しないことを検証します。編集・受理、認知負荷の比較、メタ生成の導入は、その後に具体的な不足と効果が観測された場合だけ進めます。

図表間リンクは、同一qualified要素の複数表示、別要素間の意味的な関係、表示用のナビゲーションを区別します。複合図表の関連グラフ・対応表も第二の意味正本としません。これらは採用済みの新Core SchemaやStudio編集APIを要求するものではありません。

## 作業領域H：AIによる開発支援

### 目的

AIを大量の成果物を生成するだけの仕組みにはしません。既存知識の探索・変更案の作成・検査・意思決定の支援に利用します。

### 二段階

1. `ISSUE-000047`：人とAIが共通に利用できる、既存知識を優先した意味情報の探索（PAS-06）
2. `ISSUE-000027`：AIによる変更案と共通の受理の境界（PAS-01・PAS-06）

AIによる提案には、意味の正本を変更する権限はありません。

検索順位・類似度・モデルの確信度も、意味の正しさを示す検証根拠ではありません。

AIの自律的な作業と実行権限を区別します。AIがツールを直接呼び出せることを、能力や利用許可の仕組みの代わりにはしません。

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

上流に足りない決定は、[深掘り設計の上流接続](workstreams/UPSTREAM_CONNECTION_2026-11.md) 6章に置きました。マイルストーンの到達条件、#982 の一往復の担当、#1119 の契約枠への追加は、Owner の判断を待ちます。

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
