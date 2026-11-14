# 機能設計文書

ここには、複数のIssueにまたがる機能について、意味モデル、記述方法、検証、実行時の境界をまとめた設計文書を置きます。採用前の案も、対象となるIssueと文書の状態を冒頭に明記して管理します。

- 作業内容と完了条件：GitHubのIssue
- 方針を採用した理由：[`../adr/`](../adr/README.md)
- 作業の順序と必要な検証根拠：[`../README.md`](../README.md)
- 現行実装の構成と技術的な境界：[`../../docs/architecture.md`](../../docs/architecture.md)
- 実装と検証根拠の現状：[`../../docs/status.md`](../../docs/status.md)
- 画面の試作例：[`mockups/`](mockups/README.md)

設計文書はIssueやADRの代わりにはなりません。作業の進行状況はIssue、採用した設計判断はADRを正本とします。ここには、それらの記録だけでは一つにまとまらない、機能単位の設計本文を置きます。小さな機能については新たな文書を作らず、Issue本文に記します。

`docs/`とは役割を分けます。`docs/`では実装済みの構成と利用方法を説明し、ここでは採用前の案や変更に必要な機能設計を扱います。実装状況が変わった場合は、この設計文書を現行構成の説明に流用せず、`docs/architecture.md`を更新します。

- [重要Issueの実装仕様：確定結果・再送・操作契約の発見](CRITICAL_HEADLESS_ISSUE_IMPLEMENTATION_SPEC_2026-11.md)（#1174/#2564/#3419の設計候補。実装・受入完了ではない）

## 体裁

新しい文書は[`docs/README.md`](../../docs/README.md)の「設計文書の基本体裁」に沿って作ります。冒頭で文書の役割、正本との関係、対象範囲、現在の結論、根拠、何を主張しないかを明らかにします。形式上の項目を埋めること自体を目的にはしません。

文書の状態は各設計文書の冒頭に記し、その文書自身を状態の正本とします。この索引には、変わり得る状態を重複して記載しません。状態を二択に固定せず、文書の性格に応じて`実装済みの現行基線`、`実装前設計`、`評価待ち（未採用）`などと区別します。

ここで示すのは**文書の性格**です。Issueの進行状況、判定段階の結果、実際に得られた検証根拠など、変化する状況は対応するIssue、`docs/status.md`、判定段階の監査記録などを正本とします。設計文書の状態欄には重複して記載しません。

## 一覧

設計文書の一覧です。文書の状態は各設計文書の冒頭に記し、その文書自身を状態の正本とします。

| 文書 | 対象 |
| --- | --- |
| [`PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md`](PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md) | Trigger・Invocation・State／Effect・Outcome／Responseを共通の意味から図・DSL・Mock・Targetへつなぐ検討。既存Application API／Script LaunchのOwnerへ返す |
| [`CROSS_DOMAIN_MATH_STATISTICS_PLANNING_TOOLKITS_2026-11.md`](CROSS_DOMAIN_MATH_STATISTICS_PLANNING_TOOLKITS_2026-11.md) | 数理・統計、ウィンドウ集計、工程計画、Solver・実験計画法、物理系・生命科学を、既存のDSL・標準表現型・Operator・Providerと照合し、Studioと一般アプリケーションで検証する計画（[#4660](https://github.com/hat47x/tei-platform/issues/4660)、評価待ち） |
| [`SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md`](SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md) | Script/Functionの公開Entry→製品Go Binary実起動と、OKF/DSL近傍の直接テスト・Driver Scenario結合テストの責務分担（#4502/#4495/#1180、実装前設計） |
| [`PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md`](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md) | 汎用実行構造の定義・実効計画・Trace・図解、Model of Computation、安全な処理抽出、言語/Target移行の意味保存と独立検収（#4198/#4502/#4193。新Host・Graph Compilerは未採用） |
| [`CONSUMER_QUALIFIED_CROSS_LANE_ACCEPTANCE_2026-11-14.md`](CONSUMER_QUALIFIED_CROSS_LANE_ACCEPTANCE_2026-11-14.md) | 定義から実Provider・取得制限・開示・独立Consumerの受領までを、Source／Authority／実行Attemptごとに検収する横断設計（#4502／#2586／#4495／#4583、既存Ownerへ引渡し） |
| [`ETL_BATCH_FUNCTION_COMPOSITION_DESIGN_2026-11.md`](ETL_BATCH_FUNCTION_COMPOSITION_DESIGN_2026-11.md) | ETL・Batch処理を既存Function・Data Operations・Operator・Pluginの合成として表す要件・基本設計・実装計画（[#1130](https://github.com/hat47x/tei-platform/issues/1130)・[#2622](https://github.com/hat47x/tei-platform/issues/2622)・[#4194](https://github.com/hat47x/tei-platform/issues/4194)、実装前設計・Owner未採用） |
| [`AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md`](AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md) | 生成AIによる性能チューニングの設計ヒント・統計の実測と、Source／Function／Query／Provider／受領の資格を分けた独立Benchmark・改善検証（#3271・#3262・#2586、設計・実証候補） |
| [`DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md`](DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md) | 汎用データソース・ワークベンチの標準範囲（接続・閲覧・構造定義・Migration）と縦断実証の設計（[#3270](https://github.com/hat47x/tei-platform/issues/3270)、[RQ-035](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)、設計・検証計画） |
| [`HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md`](HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md) | HTTP応答のServer/Client例外契約と、Graph/Table/Traceの相互検証（[#3274](https://github.com/hat47x/tei-platform/issues/3274)、設計候補） |
| [`TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md`](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md) | 原子的Traitの実装義務（手動/自動/Provider）、Profile合成、不適合診断、DerivedData/Pluginでの受入負例（[#3278](https://github.com/hat47x/tei-platform/issues/3278)、設計・実証候補） |
| [`AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md`](AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md) | AI-firstのSource/Tool/Independent Expected/Meaning/Grant・Effect受入を、mainのTT-56実CLIと盲検T1-B・Plugin実呼出しへ接続（§10現行状態、採用判断は既存Owner） |
| [`COMMAND_BOUNDARY_SAFETY_2026-11.md`](COMMAND_BOUNDARY_SAFETY_2026-11.md) | Command境界の未設定認可・期間状態の拒否、宣言外効果をunknownとして記録する判断と、参照実行の明示的認可（[#2996](https://github.com/hat47x/tei-platform/issues/2996)、[#2997](https://github.com/hat47x/tei-platform/issues/2997)、参照実装） |
| [`EVERYTHING_DATA_MARKDOWN_SOURCE_FIDELITY_2026-11.md`](EVERYTHING_DATA_MARKDOWN_SOURCE_FIDELITY_2026-11.md) | Markdownの元シンタックスシュガーと書式・Source occurrenceを保存で保持する契約、局所Patch・競合・往復受入 |
| [`PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md`](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) | CLI／Studioの限定read-only OperationからApplication Runtimeまで同一Plugin契約を再利用する実装検証計画（#1251・#951） |
| [`PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md`](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md) | Pluginの役割別提供・静的Composition・Studio自己再構築、Lifecycle Graphの実行境界、構文／意味／演算／Data／表示／実行／検証の貢献、Source往復と局所再評価を既存Owner・Scenarioへ結ぶ設計（§13、PCX-01〜10。正式Schema・実機検収は別） |
| [`REA_THREE_ELEMENT_CANDIDATE_INTAKE_2026-11.md`](REA_THREE_ELEMENT_CANDIDATE_INTAKE_2026-11.md) | REA同一Attemptの外部Observation/有向RelationからData/Function/Business構造候補を作り、意味受理前にレビューする読み取り専用の取り込み（[#3394](https://github.com/hat47x/tei-platform/issues/3394)、実装は候補抽出まで） |
| [`ISSUE_TO_DESIGN_DOGFOOD_MVP_2026-11.md`](ISSUE_TO_DESIGN_DOGFOOD_MVP_2026-11.md) | Issue→Design Context→Scenario/独立Expected→実CLIを通してMVPのCLI/CUI・Skill・Coverageを検収する計画（[#3273](https://github.com/hat47x/tei-platform/issues/3273)） |
| [`TEI_PRODUCT_SKILL_HARNESS_BINDING_2026-11.md`](TEI_PRODUCT_SKILL_HARNESS_BINDING_2026-11.md) | TEI read-only Product Skillを既存CLIへ正確に結び、外部Harness/Consumerの受理・許可・開示・実実行と分ける設計候補（[#2605](https://github.com/hat47x/tei-platform/issues/2605)、未採用） |
| [`TEI_MEANING_TO_CAPABILITY_SKILL_WITNESS_2026-11.md`](TEI_MEANING_TO_CAPABILITY_SKILL_WITNESS_2026-11.md) | Everything is Dataの意味Role→標準Operation→限定CLI発見→Product Skill/Harness→実Host/SHO受渡しの資格差とSource-static・合成検証（#2619/#2605/#2614、未採用） |
| [`ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md`](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md) | Everything is DataにおけるRole/Operator自己記述の有限bootstrap、eager/deferred循環、Pinと実利用資格を分離する設計案（[#2619](https://github.com/hat47x/tei-platform/issues/2619)／[#2620](https://github.com/hat47x/tei-platform/issues/2620)、未採用） |
| [`TEI_CONFIGURATION_DISCUSSION_REVIEW_2026-10-28.md`](TEI_CONFIGURATION_DISCUSSION_REVIEW_2026-10-28.md) | 10月28日時点の構成議論の再精査。三規則合成正常系の訂正、統合された参照実装と次の比較 |
| [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md) | アプリケーションの構成パターンの設計・実装基準（ISSUE-000005 / ADR-0017 / ADR-0018） |
| [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md) | 既存の認証仕様を保ち、認証・認可の機能へ接続する設計（ISSUE-000003 / ADR-0019） |
| [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md) | 機能の業務結果・名前付きの結果と実行エラーを分離する設計（ISSUE-000009 / ADR-0020） |
| [`APPLICATION_DRIVER_V1_DESIGN.md`](APPLICATION_DRIVER_V1_DESIGN.md) | 構成パターンの解決から、機能と契約に基づく問い合わせの実行、観測までの境界（ISSUE-000006 / ADR-0019 / ADR-0020 / ADR-0021 / ADR-0023） |
| [`DATABASE_CONTRACT_QUERY_V1_DESIGN.md`](DATABASE_CONTRACT_QUERY_V1_DESIGN.md) | 契約に基づく問い合わせの形式、SQLによる実現、実行器との対応付け、提供実装の実行基盤という四層（ISSUE-000018〜ISSUE-000021 / ADR-0007 / ADR-0021） |
| [`DATABASE_DIALECT_PLUGIN_DESIGN_2026-10.md`](DATABASE_DIALECT_PLUGIN_DESIGN_2026-10.md) | PostgreSQLとSQLiteを対象に、関係データベースの共通契約と製品ごとの方言プラグインを分ける検討案（識別子、方言の面、製品の版、現行コードの仕分け、実施順序） |
| [`SQL_DIALECT_LIMITATIONS_2026-11.md`](SQL_DIALECT_LIMITATIONS_2026-11.md) | SQL方言の共通契約（`internal/database/sqlgen`）が運ぶ構成要素（CTE・名前付きウィンドウ・集合操作）、未対応の構成要素、DBごとの差、呼出し側が担う契約（検証済み識別子の受渡し、未配線の名前付きパラメータ） |
| [`EXPLICIT_FUNCTION_DATA_MAPPING_AND_STANDARD_ROLES_2026-11.md`](EXPLICIT_FUNCTION_DATA_MAPPING_AND_STANDARD_ROLES_2026-11.md) | Functionごとに必要なRelation Path／Field／Scopeを明示し、標準Data構造へ意味Roleを適用するHeadless検証。Read Footprint・Claim Closure・実Provider／開示資格の区別（[#2506](https://github.com/hat47x/tei-platform/issues/2506)、[#2620](https://github.com/hat47x/tei-platform/issues/2620)、評価待ち） |
| [`DATABASE_QUERY_INPUT_NULLABILITY_2026-10.md`](DATABASE_QUERY_INPUT_NULLABILITY_2026-10.md) | Query入力のmissingと明示NULLを区別し、presence/nullを契約で明示してSQL実行前に拒否する検討案（[#2056](https://github.com/hat47x/tei-platform/issues/2056)、未採用） |
| [`GENERATOR_REVOCATION_APPROVAL_BOUNDARY_2026-10.md`](GENERATOR_REVOCATION_APPROVAL_BOUNDARY_2026-10.md) | 生成器の失効承認recordをregistry編集と分離し、対象一致・監査・再生成の完了条件を整理する検討案（[#2076](https://github.com/hat47x/tei-platform/issues/2076)、未採用） |
| [`CONTRACT_ONLY_GENERATIVE_PROVIDER_POC_2026-10.md`](CONTRACT_ONLY_GENERATIVE_PROVIDER_POC_2026-10.md) | 実装を持たない契約と独立した適合試験を先に共有し、案件内でAIまたは人が必要なProviderだけを補完するPoC案（[#1975](https://github.com/hat47x/tei-platform/issues/1975)、未採用） |
| [`BUSINESS_LIFECYCLE_V1_DESIGN.md`](BUSINESS_LIFECYCLE_V1_DESIGN.md) | 業務の状態遷移を意味の正本として定義し、実行・観測する設計（ISSUE-000038 / ADR-0024 / ADR-0025） |
| [`DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`](DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md) | RQ-026の複数視点の照合を、対象を限定した証明上の条件として設計案の候補を絞り込む処理へ接続する、評価待ちの設計 |
| [`ARTIFACT_SET_SCOPE_AND_CONSUMER_READ_EVIDENCE_2026-11.md`](ARTIFACT_SET_SCOPE_AND_CONSUMER_READ_EVIDENCE_2026-11.md) | #2482・#2604の導出Artifactについて、Application/Consumerの必要集合とManifest内部整合、公開一式の実読取を区別する検証計画（提案中・未採用） |
| [`TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md`](TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md) | トポロジーの基本構造・複合図表と関連表・Complect解消・開発パイプラインの完全性を、既存契約へ照合する未採用の設計・段階的実証計画 |
| [`PROJECT_PLUGIN_COMPOSITION_DESIGN.md`](PROJECT_PLUGIN_COMPOSITION_DESIGN.md) | 案件ごとのプラグイン、画面部品一式、仕様テンプレート、構成プロファイルの境界（[GitHub #1016](https://github.com/hat47x/tei-platform/issues/1016)、評価待ち） |
| [`MULTILINGUAL_SEMANTIC_RESOURCE_DESIGN_2026-10.md`](MULTILINGUAL_SEMANTIC_RESOURCE_DESIGN_2026-10.md) | 意味モデルと表示資源を分離する多言語対応、言語資源の所有、代替、配布、翻訳の確認（[GitHub #1396](https://github.com/hat47x/tei-platform/issues/1396)、評価待ち） |
| [`MULTILINGUAL_BUSINESS_DATA_AND_AI_TRANSLATION_2026-10.md`](MULTILINGUAL_BUSINESS_DATA_AND_AI_TRANSLATION_2026-10.md) | 多言語の業務属性、記述の版・保存・検索・公開と、生成AIによる翻訳候補・確認・受理（[GitHub #1453](https://github.com/hat47x/tei-platform/issues/1453)、評価待ち） |
| [`MULTILINGUAL_TRANSPARENT_PLUGIN_OVERLAY_2026-10.md`](MULTILINGUAL_TRANSPARENT_PLUGIN_OVERLAY_2026-10.md) | 通常プラグインへ多言語の表示情報を後付けする意味参照・適用許可・競合解決と合成（[GitHub #1551](https://github.com/hat47x/tei-platform/issues/1551)、参照実装検証中） |
| [`CAPABILITY_INVOCATION_CONTEXT_DESIGN.md`](CAPABILITY_INVOCATION_CONTEXT_DESIGN.md) | 機能に渡す意味上の要求と、ホストが注入する実行時の文脈を分ける、評価待ちの設計（[GitHub #1118](https://github.com/hat47x/tei-platform/issues/1118)） |
| [`TEI_DESIGN_CONTEXT_TEST_HARNESS_INTEGRATION_2026-11.md`](TEI_DESIGN_CONTEXT_TEST_HARNESS_INTEGRATION_2026-11.md) | 設計ContextからTest Skill/Harnessを経て独立検証へ至る統合計画（[#3262](https://github.com/hat47x/tei-platform/issues/3262)、[#3223](https://github.com/hat47x/tei-platform/pull/3223)と接続） |
| [`PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md`](PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md) | プラグイン間の依存を、受け渡す項目・パラメータ・値の出どころ・対応付けと変換・多重度・版の変化まで含めて整理する方式設計候補（[GitHub #1223](https://github.com/hat47x/tei-platform/issues/1223)、評価待ち） |
| [`PLUGIN_NAMESPACE_AND_OPTIONAL_COMPOSITION_2026-10.md`](PLUGIN_NAMESPACE_AND_OPTIONAL_COMPOSITION_2026-10.md) | `tei.std` / `tei.experimental` / `tei.contrib` の管理名前空間と、Principal・DB・TransactionをCoreへ強制しない案件選択型Compositionの設計候補（提案中） |
| [`SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`](SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md) | SUIの既存実装を生かしながらTEIへ移行するための設計基準 |
| [`SUI_TEI_MIGRATION_IMPLEMENTATION_PACKET_2026-11.md`](SUI_TEI_MIGRATION_IMPLEMENTATION_PACKET_2026-11.md) | SUI→TEI移行のM1〜M6を、変更範囲・型・試験・受入・Owner判断・完了条件までPR単位に具体化した実装前の提案（[#3401](https://github.com/hat47x/sui-sensemaking/issues/3401)、[#2497](https://github.com/hat47x/tei-platform/issues/2497)、未採用） |
| [`CROSS_VIEW_CHANGE_QUALIFICATION_REFERENCE_DRAFT.md`](CROSS_VIEW_CHANGE_QUALIFICATION_REFERENCE_DRAFT.md) | Issue #1143で検討中の、複数の視点にまたがる変更の適格性確認案。参照用の契約としては未採用 |
| [`STUDIO_CONSUMER_FRESHNESS_BOUNDARY_2026-10.md`](STUDIO_CONSUMER_FRESHNESS_BOUNDARY_2026-10.md) | #1124の非Host Studio Consumerが非同期応答・Graph描画選択・Source/Accepted/readinessの鮮度を別々に検査する未採用の設計補足 |
| [`PLUGIN_MANAGEMENT_CONTRACT_2026-11.md`](PLUGIN_MANAGEMENT_CONTRACT_2026-11.md) | プラグイン管理の状態分離・理由コード対応・role契約対応表・データlevel導出を1箇所にまとめる管理契約（[#2044](https://github.com/hat47x/tei-platform/issues/2044)、S1は[#4376](https://github.com/hat47x/tei-platform/issues/4376)で実装、S2〜S5は追記） |
| [`BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md`](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md) | 副作用を伴う実行の段階、確定後の外部効果、冪等の識別子、確定状態、補償、一括処理の再開をまとめる、評価待ちの設計（[GitHub #1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1130](https://github.com/hat47x/tei-platform/issues/1130)） |
| [`BUSINESS_ACTION_COMPLETION_EVIDENCE_DESIGN_2026-11.md`](BUSINESS_ACTION_COMPLETION_EVIDENCE_DESIGN_2026-11.md) | Business Actionの完了述語、必要対象集合の完全性、Function利用箇所・実確定Effect・外部証拠の接合を区別する未採用の設計（[#1125](https://github.com/hat47x/tei-platform/issues/1125)、[#1119](https://github.com/hat47x/tei-platform/issues/1119)） |
| [`EXECUTION_CORRELATION_DESIGN_2026-10.md`](EXECUTION_CORRELATION_DESIGN_2026-10.md) | 一つの業務操作を実行先から提供元まで追跡し、意味の版・ビルド・ロックへ戻るための識別子と記録の規則。担当Issueは無く、評価待ち |
| [`SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md`](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md) | 受理した意味の変更と既存の業務データの両立性の判定、影響の観測、段階的な移行。担当Issueは無く、評価待ち |
| [`KNOWLEDGE_VARIATION_DESIGN_2026-10.md`](KNOWLEDGE_VARIATION_DESIGN_2026-10.md) | 再利用した知識の変動点、適用先での選択と分岐、再利用元の更新の扱い。RQ-027・RQ-029に対応する、評価待ちの設計 |
| [`GENERATOR_CONTRACT_BOUNDARIES_DESIGN_2026-10.md`](GENERATOR_CONTRACT_BOUNDARIES_DESIGN_2026-10.md) | 生成器の契約に宣言する、派生名の規則・拡張点・AIの書込み窓・要求する能力と縮退。評価待ちの設計（[GitHub #2090](https://github.com/hat47x/tei-platform/issues/2090)、[#2092](https://github.com/hat47x/tei-platform/issues/2092)、[#2094](https://github.com/hat47x/tei-platform/issues/2094)、[#2096](https://github.com/hat47x/tei-platform/issues/2096)） |
| [`BUSINESS_DATA_IMPORT_DESIGN_2026-10.md`](BUSINESS_DATA_IMPORT_DESIGN_2026-10.md) | 人が用意した業務データと初期データの取込みを、通常のCapabilityの検証・認可・結果の分離を通して行うための段階案（[GitHub #1807](https://github.com/hat47x/tei-platform/issues/1807)） |
| [`POSTGRES_CONSTRAINT_RENDERING_DESIGN_2026-10.md`](POSTGRES_CONSTRAINT_RENDERING_DESIGN_2026-10.md) | PostgreSQL提供元が主キー・一意制約・外部キーを出力できない現状と、制約の名前の決め方、段階的な解消案（[GitHub #1803](https://github.com/hat47x/tei-platform/issues/1803)） |
| [`AUTHORING_TOOL_FEATURE_TRANSFER_MAP_2026-10.md`](AUTHORING_TOOL_FEATURE_TRANSFER_MAP_2026-10.md) | 既存の設計ツール（X-TEA ModelerとEditor）の機能を、担当Issue・新規Issue・移植しない機能へ仕分けた対応表（[GitHub #1802](https://github.com/hat47x/tei-platform/issues/1802)） |
| [`AUXILIARY_DESIGN_INFORMATION_PLACEMENT_2026-10.md`](AUXILIARY_DESIGN_INFORMATION_PLACEMENT_2026-10.md) | 用語、改訂履歴、注記、区分などの補助情報を、意味の正本・導出・表示情報・運用の記録のどれに置くかの案（[GitHub #1808](https://github.com/hat47x/tei-platform/issues/1808)） |
| [`GENERATIVE_LOWCODE_PRODUCT_STRATEGY_2026-10.md`](GENERATIVE_LOWCODE_PRODUCT_STRATEGY_2026-10.md) | 生成AIネイティブなローコード開発基盤としての価値仮説、適用範囲、優先順位、測定と撤回条件（#2016〜#2021、未採用） |
| [`CHANGE_CONTINUITY_DESIGN_2026-10.md`](CHANGE_CONTINUITY_DESIGN_2026-10.md) | 更新・再生成で案件の知識と処理を保つ機能、AI編集と業務経路、派生名と意味への対応、能力の照合、担当境界と受入候補（#2090〜#2096） |
| [`DSH_DEVELOPMENT_PLUGIN_PROPOSAL_2026-10.md`](DSH_DEVELOPMENT_PLUGIN_PROPOSAL_2026-10.md) | DeepSeek HarnessをTEI開発の薄いAdapterとして使うためのプラグイン・スキル提案（未実装） |
| [`PRODUCTIVITY_BENCHMARK_PROTOCOL_2026-10.md`](PRODUCTIVITY_BENCHMARK_PROTOCOL_2026-10.md) | 従来型コーディング・ローコード・TEIを同じ業務変更で比較する手順、指標、不確かさ、撤回条件（[GitHub #2018](https://github.com/hat47x/tei-platform/issues/2018)、未実施） |
| [`DELIVERY_PLUGIN_COMPOSITION_2026-10.md`](DELIVERY_PLUGIN_COMPOSITION_2026-10.md) | 個人・SIer・SaaSの提供形態を、形態別実装ではなくプラグインの選択で構成する設計案（[GitHub #2020](https://github.com/hat47x/tei-platform/issues/2020)、[ADR-0036](../adr/0036-delivery-forms-as-plugin-composition.md)） |
| [`TECH_VALIDATION_SAMPLES_2026-10.md`](TECH_VALIDATION_SAMPLES_2026-10.md) | ピーキーな要件の目録と、それを一つの作業の流れで同時に起こす技術検証用サンプル(既存の検証用資産の再編と新設)の構成案(未採用) |
| [`AI_NATIVE_PLUGIN_COMPOSITION_DESIGN_2026-10.md`](AI_NATIVE_PLUGIN_COMPOSITION_DESIGN_2026-10.md) | 標準機能を含む製品構成を交換可能にし、生成AIの設計・実装・検証の足場にもする最小プラグイン規約の設計候補（[GitHub #1223](https://github.com/hat47x/tei-platform/issues/1223)ほか） |
| [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md) | 業務手順の設計品質を、業務意味を根拠を失わずに説明・検証・修正できる程度で評価するモデル（RQ-025・RQ-026） |
| [`CLI_C2_DISCLOSURE_BOUNDARY_DESIGN.md`](CLI_C2_DISCLOSURE_BOUNDARY_DESIGN.md) | CLIのinspect・traceで、開示してよい内容を既存のAuthorization Capabilityで判定する境界の設計候補（[GitHub #1178](https://github.com/hat47x/tei-platform/issues/1178)） |
| [`CLI_JSON_OUTPUT_CONTRACT_2026-10.md`](CLI_JSON_OUTPUT_CONTRACT_2026-10.md) | `tei commands`が広告する読取りコマンドのJSON出力の現状一覧、`tei.cli.*/v0`の互換変更の区別の規則案、schemaVersion追加と失敗時形式の決定案（[GitHub #2354](https://github.com/hat47x/tei-platform/issues/2354)、提案中） |
| [`CLI_FIRST_INTERFACE_DESIGN.md`](CLI_FIRST_INTERFACE_DESIGN.md) | CLIを主入口とし、画面・API・CLIで共通にする意味（対象ID・版・検証結果・受理の成否など）を分けるインターフェース設計の初期案 |
| [`CORDIS_CONTINUOUS_PLUGIN_UPDATE_EVALUATION_2026-10.md`](CORDIS_CONTINUOUS_PLUGIN_UPDATE_EVALUATION_2026-10.md) | Cordis由来の合成機構とTEIプラグインの継続更新の比較検討記録（[ADR-0032](../adr/0032-composition-kernel-replacement.md)で前提が置き換えられた） |
| [`CORE_SUBPROJECT_ISOLATION_PROBE_2026-10.md`](CORE_SUBPROJECT_ISOLATION_PROBE_2026-10.md) | 合成基盤を独立して構築できるかを確かめる局所的な検証手順（[ADR-0033](../adr/0033-core-and-plugin-subprojects.md)に関連） |
| [`CRITICAL_LOWERING_CONTRACT_DESIGN.md`](CRITICAL_LOWERING_CONTRACT_DESIGN.md) | 宣言からTargetの実現へ段階的に接続するCritical Lowering契約の候補 |
| [`CROSS_ROUTE_FUNCTIONAL_PARITY_AND_TEST_DERIVATION.md`](CROSS_ROUTE_FUNCTIONAL_PARITY_AND_TEST_DERIVATION.md) | 経路間の業務機能の完全性と、Scenarioから導くE2E試験の設計。既存`tei scenario verify/generate/run/check`による限定Referenceと、未実装のWeb/GUI・実Provider E2Eを区別する |
| [`CROSS_SEARCH_AND_REPLACE_PROPOSAL_2026-10.md`](CROSS_SEARCH_AND_REPLACE_PROPOSAL_2026-10.md) | 横断検索と、置換を変更案へ変換する最小経路の設計案（[GitHub #1804](https://github.com/hat47x/tei-platform/issues/1804)、PoC） |
| [`CROSS_SEARCH_ACCEPTANCE_EXECUTION_2026-10.md`](CROSS_SEARCH_ACCEPTANCE_EXECUTION_2026-10.md) | 横断置換の変更案を受理境界へ渡し、受理後に各視点へ再投影する受理実行段の設計案（[GitHub #1804](https://github.com/hat47x/tei-platform/issues/1804)、PoC） |
| [`CROSS_VIEW_TYPED_REFERENCE_CONTRACT_2026-10.md`](CROSS_VIEW_TYPED_REFERENCE_CONTRACT_2026-10.md) | 三視点の意味参照と変更影響の初期契約。参照契約の実装・試験は導入済みで、受理済みモデルの第二の正本ではない（[GitHub #1120](https://github.com/hat47x/tei-platform/issues/1120)） |
| [`DATA_DERIVATION_REFERENCE_CONTRACT.md`](DATA_DERIVATION_REFERENCE_CONTRACT.md) | データ導出規則の最小意味契約（検証案）。導出関係の意味契約の正本 |
| [`DATA_DERIVATION_CONTRACT_REFERENCE.md`](DATA_DERIVATION_CONTRACT_REFERENCE.md) | データ導出規則の参照実装と検証範囲の案内（[GitHub #1637](https://github.com/hat47x/tei-platform/issues/1637)）。意味契約は上の文書が正本 |
| [`EVALUATION_CONTEXT_DETERMINISM_2026-10.md`](EVALUATION_CONTEXT_DETERMINISM_2026-10.md) | データ導出の評価文脈の明示入力と決定的な再評価の設計整理（[GitHub #1720](https://github.com/hat47x/tei-platform/issues/1720)、採用前） |
| [`TEMPORAL_MEANING_DISTINCTION_2026-10.md`](TEMPORAL_MEANING_DISTINCTION_2026-10.md) | 導出値の時間意味(過去時点の再計算・規則の版・確定した保存値)を区別する提案（[GitHub #1727](https://github.com/hat47x/tei-platform/issues/1727)） |
| [`DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md`](DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md) | Desktop Studio Hostの操作・状態・資源の境界の設計候補。実機の起動と操作の検証は未実施 |
| [`DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md`](DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md) | DSLの言語プラグインと演算子の合成についての設計候補 |
| [`THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md`](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md) | 三要素分析の意味拡張／方法論・図法／図面投影・配置・編集候補を分離し、既存Studio・Source受理・旧資産Adapterへ接続する設計候補（#1124/#1125/#1126、未採用） |
| [`EXAMPLE_ER_SLIDE_M1_2026-10.md`](EXAMPLE_ER_SLIDE_M1_2026-10.md) | 具体例からER図のスライドへつなぐ参照実装（M1）の候補 |
| [`EXTERNAL_SEMANTIC_ACCEPTANCE_HANDOFF_2026-10.md`](EXTERNAL_SEMANTIC_ACCEPTANCE_HANDOFF_2026-10.md) | 外部の業務意味資産の選択と、案件での受理を結ぶ証拠契約の候補 |
| [`GENERIC_CAPABILITY_MIGRATION_DESIGN.md`](GENERIC_CAPABILITY_MIGRATION_DESIGN.md) | Generic Capabilityへの移行設計（評価待ち・未採用） |
| [`LOGICAL_RULE_SOURCE_CUT_PREFLIGHT.md`](LOGICAL_RULE_SOURCE_CUT_PREFLIGHT.md) | 候補キー・関数従属性の撤回に向けた、正本ソースの範囲を特定する読み取り専用の参照機構（[GitHub #1126](https://github.com/hat47x/tei-platform/issues/1126)） |
| [`SEMANTIC_LOGICAL_IDENTITY_FD_AUTHORING.md`](SEMANTIC_LOGICAL_IDENTITY_FD_AUTHORING.md) | 論理候補キーと関数従属性を正本ソースで明示宣言する、読み取りと静的検証の最小実装（[GitHub #1126](https://github.com/hat47x/tei-platform/issues/1126)） |
| [`RELATION_HISTORICAL_IMPACT_2026-10.md`](RELATION_HISTORICAL_IMPACT_2026-10.md) | Relationの削除・改名・分割で、変更前の状態から過去の参照を取り出す読み取り専用の拡張（[GitHub #1126](https://github.com/hat47x/tei-platform/issues/1126)） |
| [`PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md`](PLUGIN_SELF_BUILD_PRODUCTION_CONTRACT_2026-10.md) | プラグインのセルフビルドの最小契約と移行Gateの検討案。production仕様の採用ではない |
| 文書 | 対象 |
| --- | --- |
| [`VERIFIED_COMPOSITION_BOM_AND_MATRIX_DESIGN_2026-11.md`](VERIFIED_COMPOSITION_BOM_AND_MATRIX_DESIGN_2026-11.md) | 検証済みPlugin構成BOM、多版Matrix、Pairwise、独立Expected、実Artifactと同一Attempt、Trait/HTTP例外Coverageの資格分離（[#3267](https://github.com/hat47x/tei-platform/issues/3267)、評価待ち） |
| --- | --- |
| [`THREE_VIEW_DIAGRAM_CROSS_REFERENCE_PROJECTION_2026-11.md`](THREE_VIEW_DIAGRAM_CROSS_REFERENCE_PROJECTION_2026-11.md) | 三視点Data/Function/Business Diagram・CRUD Cross-referenceのSource追跡、複数Canvas・可逆編集の受入と合成Fixture（[#1124](https://github.com/hat47x/tei-platform/issues/1124)、Draft #3263・未採用） |
| [`EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md`](EXTERNAL_GO_MODULE_ADMISSION_CONTRACT_2026-11.md) | 外部Go Module入力をSource閉包へ入れられるかの判断材料。観測済みの事実と三つの選択肢・既定案・覆す条件を並べた検討案で、方針の採用ではない（[GitHub #2831](https://github.com/hat47x/tei-platform/issues/2831)） |
| [`REFERENCE_PORT_IMPLEMENTABILITY_GAP_TRIAGE_2026-10.md`](REFERENCE_PORT_IMPLEMENTABILITY_GAP_TRIAGE_2026-10.md) | 参照モデルの実装可否と欠落機能を、原典側で止まった理由から製品側の責務へ渡す照合方法の設計案（[GitHub #1969](https://github.com/hat47x/tei-platform/issues/1969)） |
| [`REPOSITORY_ARCHITECTURE_DEPENDENCY_GUARD_2026-10.md`](REPOSITORY_ARCHITECTURE_DEPENDENCY_GUARD_2026-10.md) | リポジトリの依存方向を守る最小ガードの実装候補 |
| [`RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md`](RUNTIME_TARGET_BINARY_COMPOSITION_DESIGN.md) | 多様な実行環境に向けた最小ランタイムと配備単位の設計候補（評価待ち） |
| [`SEI_REFERENCE_ADOPTION_STAGE0_DESIGN.md`](SEI_REFERENCE_ADOPTION_STAGE0_DESIGN.md) | SEIの既存実装をTEIの参照として取り込む段階0（Resume・Review Projection）の設計。固定した版のみに対する検証 |
| [`STANDARD_MESSAGE_AND_LOCALIZATION_PLUGIN_2026-10.md`](STANDARD_MESSAGE_AND_LOCALIZATION_PLUGIN_2026-10.md) | 標準メッセージと多言語拡張をプラグインとして分離する候補契約とGo参照実装 |
| [`SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md`](SUI_MIGRATION_EXTENSION_AND_SPLIT_DESIGN_2026-10.md) | SUIの移管に向けたTEIの拡張方針とSUIの分割案（評価待ち・未採用） |
| [`SUI_TEI_MIGRATION_TASKS_2026-11.md`](SUI_TEI_MIGRATION_TASKS_2026-11.md) | read-only参照の実装知見（SUI PR #3429）を*.tei.md＋Pluginの移行タスクへ分解したもの。M0〜M6を「目的・変更境界・依存・契約案・受入・負例・未実施・委譲先・撤退条件」の作業票に整形し、順序とブロッカーを明記する（評価待ち・未採用） |
| [`TYPE_SPLIT_AND_REPLACE_PROPOSAL_2026-10.md`](TYPE_SPLIT_AND_REPLACE_PROPOSAL_2026-10.md) | 意味の型の分割・置換を変更案として扱う最小経路の設計案（[GitHub #1806](https://github.com/hat47x/tei-platform/issues/1806)、PoC） |
| [`WEB_APPLICATION_COMPOSITION_DESIGN.md`](WEB_APPLICATION_COMPOSITION_DESIGN.md) | Webアプリケーションの構成キットとして、安全な要求経路と認証方式を組み合わせる設計候補（評価待ち） |
| [`WEB_READ_WRITE_API_ADAPTER_DESIGN.md`](WEB_READ_WRITE_API_ADAPTER_DESIGN.md) | 標準Webの読込・書込API Adapterの設計候補と合成Reference。OData準拠・実DB書込・本番適合は未確認 |
| [`XTEA_STARTUP_JAR_DISTRIBUTION_EVIDENCE_2026-10.md`](XTEA_STARTUP_JAR_DISTRIBUTION_EVIDENCE_2026-10.md) | X-TEAのstartup JARの配布・版・更新に関する、外部Referenceの静的な証拠整理。実行検証・TEIの更新機構への受入証明ではない |
| [`OPERATION_SURFACE_DESIGN_2026-11.md`](OPERATION_SURFACE_DESIGN_2026-11.md) | 操作面（何ができて何が足りないか）の導出規則・来歴・未解決の分類・自律区分を定める共通設計（[#1127](https://github.com/hat47x/tei-platform/issues/1127)〜[#2144](https://github.com/hat47x/tei-platform/issues/2144)、提案中） |
| [`DERIVED_CRUD_OPERATION_DESIGN_2026-11.md`](DERIVED_CRUD_OPERATION_DESIGN_2026-11.md) | Data-firstから標準CRUD・検索・保守画面までの導出と、同時更新・廃止・提示の受入（[#1127](https://github.com/hat47x/tei-platform/issues/1127)、提案中） |
| [`EFFECT_RULE_AUTHORING_DESIGN_2026-11.md`](EFFECT_RULE_AUTHORING_DESIGN_2026-11.md) | 効果規則の宣言・所有者境界の診断・仮想項目の保存境界。実行時の段階は業務副作用の設計が正本（[#1129](https://github.com/hat47x/tei-platform/issues/1129)、提案中） |
| [`RUN_PLAN_AND_UNATTENDED_EXECUTION_DESIGN_2026-11.md`](RUN_PLAN_AND_UNATTENDED_EXECUTION_DESIGN_2026-11.md) | 実行計画の宣言、業務キーによる実行の同定、再開・二重起動の拒否、帳票の決定性、無人の実行条件（[#1130](https://github.com/hat47x/tei-platform/issues/1130)、提案中） |
| [`PERIOD_STATE_SEMANTICS_DESIGN_2026-11.md`](PERIOD_STATE_SEMANTICS_DESIGN_2026-11.md) | 業務日付・会計期間の状態・締めの一貫点・確定した行の不変性・赤黒の訂正・遅延到着の方針（[#2144](https://github.com/hat47x/tei-platform/issues/2144)、提案中・中核変更のためA4判断） |
| [`DERIVED_DESIGN_ARTIFACTS_DESIGN_2026-11.md`](DERIVED_DESIGN_ARTIFACTS_DESIGN_2026-11.md) | 受理済みの意味から表定義書・機能定義書・対応一覧・DDLを決定的に導出し、差分と未解決を読む契約（[#1803](https://github.com/hat47x/tei-platform/issues/1803)、提案中） |
| [`XTEA_STANDARD_BUSINESS_SEMANTICS_2026-11.md`](XTEA_STANDARD_BUSINESS_SEMANTICS_2026-11.md) | X-TEAの標準業務ライブラリ（暦・税・通貨・採番・日付計算・利用者設定）を、宣言された意味と契約へ受け止める設計案（未採用）。年度開始・休日のData化、税率の適用期間の両端、率の種類、採番の桁あふれと欠番の契約を含む |
| [`XTEA_DERIVED_OUTPUT_RECEPTION_2026-11.md`](XTEA_DERIVED_OUTPUT_RECEPTION_2026-11.md) | X-TEAの設計書・DDL・行列・差分・一覧の往復を、TEIの導出成果物の契約（版・規則・未解決・非昇格）へ受け止める設計案（未採用） |
| [`XTEA_AUTHORING_SURFACE_RECEPTION_2026-11.md`](XTEA_AUTHORING_SURFACE_RECEPTION_2026-11.md) | X-TEAのModeler・Editor・Driverを読み、TEI Studioの作者環境と標準業務の基本機能へ受け止める形を整理した設計案（未採用）。配置検査の面分割、検索・置換の版境界、編集履歴のまとまり、雛形生成の未受理扱い、標準業務の宣言化を含む |
| [`XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md`](XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md) | X-TEAから回収したローコードの知見を、Core・Pattern・生成器・Plugin・Studio・要件のどこで受け止めるかの設計案（未採用）。パッケージ・対話フック・項目の役割・結果型・標準業務意味の判断候補を含む |
| [`DATA_OPERATIONS_KEY_SPACE_AND_MULTIPLICITY_LOSS_2026-11.md`](DATA_OPERATIONS_KEY_SPACE_AND_MULTIPLICITY_LOSS_2026-11.md) | MS-09：多様なData Structure間のKey空間・多重度・逆更新の喪失を明示する |
| [`GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md`](GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md) | セルフビルドで外部Go Moduleを入力として認める条件（供給manifest・`h1:`照合・`go mod verify`・`corrupt-supply`分類・成果物digestを供給同一性の根拠にしない負例）の設計候補（[#2831](https://github.com/hat47x/tei-platform/issues/2831) SB-0b、未採用） |
| [`EVERYTHING_DATA_GO_LANGUAGE_REFERENCE_BRIDGE_2026-11.md`](EVERYTHING_DATA_GO_LANGUAGE_REFERENCE_BRIDGE_2026-11.md) | Everything is Dataの複数表層を既存Go Language Plugin Referenceに接続する限定試験設計（#2619/#2620/#988、Draft #2639。新Core/正式意味受理・実Product/Hostは未実証） |
| [`QUERY_QX31_DYNAMIC_PATH_AND_NEGATIVE_DEPENDENCY_2026-11.md`](QUERY_QX31_DYNAMIC_PATH_AND_NEGATIVE_DEPENDENCY_2026-11.md) | QX-31：SQLのRelation path再束縛、相関NOT EXISTSの0件・否定依存、Read Cohortと現在Claimの分離をSQLiteで反証する補助設計（[#2387](https://github.com/hat47x/tei-platform/issues/2387)、#2582、未採用） |
| [`QUERY_STRUCTURAL_MODEL_AND_FIXTURE_WORKBENCH_2026-10.md`](QUERY_STRUCTURAL_MODEL_AND_FIXTURE_WORKBENCH_2026-10.md) | Query構造Explainのread-only投影と、QX-01〜32のSQL/共通Data Operator/Function Context/Provider・Target同義性、証拠・開示・完全性の境界を分ける継続検証計画（[#2387](https://github.com/hat47x/tei-platform/issues/2387)、Draft #2388、未採用） |
| [`QUERY_QX33_ANALYZER_INVOCATION_BINDING_AND_VERTICAL_HANDOFF_2026-11.md`](QUERY_QX33_ANALYZER_INVOCATION_BINDING_AND_VERTICAL_HANDOFF_2026-11.md) | QX-33：SQL Analyzerの実呼出し・入力byte帰属とQuery→Context縦断の受渡し（2026-11-03） |
| [`QUERY_QX33_V0_V1_WORKER_READINESS_AND_ACCEPTANCE_2026-11.md`](QUERY_QX33_V0_V1_WORKER_READINESS_AND_ACCEPTANCE_2026-11.md) | QX-33 V0/V1：実SQL解析Producerの所在・入力帰属・Go DerivedIR縦断受入票 |
| [`CROSSLANE_REAL_CLI_OPERATION_ARTIFACT_2026-11.md`](CROSSLANE_REAL_CLI_OPERATION_ARTIFACT_2026-11.md) | Everything is Data：実TEI CLIでのOperation→Artifact受渡しを同一版で検証する |
| [`FUNCTION_MAPPING_SQLITE_READ_WITNESS_2026-11.md`](FUNCTION_MAPPING_SQLITE_READ_WITNESS_2026-11.md) | Function Data MappingとProvider側Read Witnessの接合検証（2026-11） |
| [`QUERY_QX14_INPUT_ORIGIN_EVIDENCE_2026-11.md`](QUERY_QX14_INPUT_ORIGIN_EVIDENCE_2026-11.md) | QX-14追補：SQL解析の「入力帰属」と演算・Data Contextへ渡せる証拠 |
| [`GENERIC_STUDIO_OPEN_WORLD_PROJECTION_CONFORMANCE_2026-11.md`](GENERIC_STUDIO_OPEN_WORLD_PROJECTION_CONFORMANCE_2026-11.md) | TEI Studio標準UIの開放的拡張性：標準データ構造／意味Plugin／汎用Projectionの適合計画（2026-11-02） |
| [`EXPLICIT_DATA_FOOTPRINT_VS_PROVIDER_READ_WITNESS_2026-11.md`](EXPLICIT_DATA_FOOTPRINT_VS_PROVIDER_READ_WITNESS_2026-11.md) | Function Read FootprintとProvider実アクセスの差を観測する（2026-11-02） |
| [`EVERYTHING_DATA_SYNTAX_NORMAL_FORM_EQUIVALENCE_2026-11.md`](EVERYTHING_DATA_SYNTAX_NORMAL_FORM_EQUIVALENCE_2026-11.md) | Everything is Data：表層構文からプレーンな標準データへ収束する条件の限定Reference（2026-11-02） |
| [`EVERYTHING_DATA_INTERPRETATION_AND_SOURCE_BINDING_2026-11.md`](EVERYTHING_DATA_INTERPRETATION_AND_SOURCE_BINDING_2026-11.md) | Everything is Data：引用データ・意味Role適用・参照束縛・Source編集の境界（2026-11-02） |
| [`SCRIPT_FIRST_GENERIC_STUDIO_PROJECTION_REFERENCE_2026-11.md`](SCRIPT_FIRST_GENERIC_STUDIO_PROJECTION_REFERENCE_2026-11.md) | Script-first + 標準データ構造からの汎用Studio投影：Headless受入案（2026-11-02） |
| [`SEMANTIC_CONTEXT_ACTION_AFFORDANCE_2026-11.md`](SEMANTIC_CONTEXT_ACTION_AFFORDANCE_2026-11.md) | Semantic Contextに基づく汎用Action発見・UI Affordance・実Request接合（2026-11-02） |
| [`SOURCE_REVISION_CORRESPONDENCE_AND_CONSUMER_IMPACT_2026-11.md`](SOURCE_REVISION_CORRESPONDENCE_AND_CONSUMER_IMPACT_2026-11.md) | Source改版追跡とConsumer別の再適用・再生成（設計候補） |
| [`FUNCTION_PATTERN_ASSUME_GUARANTEE_COMPOSITION_2026-11.md`](FUNCTION_PATTERN_ASSUME_GUARANTEE_COMPOSITION_2026-11.md) | Function Patternの前提・保証と合成可能性：異質Domainの反例検証（2026-11） |
| [`COMPOSITE_FUNCTION_VERSIONED_EVOLUTION_2026-11.md`](COMPOSITE_FUNCTION_VERSIONED_EVOLUTION_2026-11.md) | Composite Function の改版・状態移行・実行世代の交差検証（2026-11） |
| [`FUNCTION_CONTEXT_EDIT_SESSION_SAVE_BOUNDARY_2026-11.md`](FUNCTION_CONTEXT_EDIT_SESSION_SAVE_BOUNDARY_2026-11.md) | 複数Data Contextの編集セッションと保存境界：機能合成の横断設計（2026-11） |
| [`MEASURE_GRAIN_AGGREGATION_CONFORMANCE_2026-11.md`](MEASURE_GRAIN_AGGREGATION_CONFORMANCE_2026-11.md) | 測定値の集計軸・粒度・時間・再集計適合：Function Data Contextの横断圧力試験 |
| [`PLUGIN_REGISTRY_PUBLICATION_VERIFICATION_DESIGN_2026-11.md`](PLUGIN_REGISTRY_PUBLICATION_VERIFICATION_DESIGN_2026-11.md) | 分散型Plugin公開・登録・検証プロトコル（設計候補、2026-11-02） |
| [`FUNCTION_DATA_CONTEXT_PATTERN_PRESSURE_2026-11.md`](FUNCTION_DATA_CONTEXT_PATTERN_PRESSURE_2026-11.md) | Function Data Contextと機能パターンの横断設計・圧力試験（2026-11） |
| [`JAVA_LIBRARY_PORTABLE_CONSUMER_AND_STORE_GATES_2026-10.md`](JAVA_LIBRARY_PORTABLE_CONSUMER_AND_STORE_GATES_2026-10.md) | Java Libraryの外部Consumer・Artifact Store受入計画（2026-10-30） |
| [`ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md`](ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md) | 組織関係・時点・統制の標準意味モデル：横断設計と最小検証計画 |
| [`ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md`](ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md) | 高度組織モデリングの段階計画：5段階の基本設計（Identity・Relation・Cardinality・Time・Validation・Query・表示形式）と作業単位WU-01〜14（[#2386](https://github.com/hat47x/tei-platform/issues/2386)、設計・未受理） |
| [`DSL_NEUTRAL_UNICODE_AUTHORING_AND_SAFE_AI_2026-11.md`](DSL_NEUTRAL_UNICODE_AUTHORING_AND_SAFE_AI_2026-11.md) | TEI DSL：グローバル共通構文・Unicode表層名・Studio表示・AI入力安全性の統合計画（2026-11-03） |
| [`ETL_ELT_MFT_CROSSBOUNDARY_CONTRACT_PRESSURE_2026-11.md`](ETL_ELT_MFT_CROSSBOUNDARY_CONTRACT_PRESSURE_2026-11.md) | ETL/ELT・MFT/ファイルリレー横断契約の圧力試験（設計候補・2026-11-04） |
| [`PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md`](PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md) | Plugin機能の実現・安全な最適化に必要な契約（2026-11-06、設計/要件候補） |
| [`SCRIPT_INVOCATION_FAILURE_AND_RECOVERY_DESIGN_2026-11.md`](SCRIPT_INVOCATION_FAILURE_AND_RECOVERY_DESIGN_2026-11.md) | Script起動の失敗状態・再実行・回復設計（2026-11） |
| [`SEMANTIC_TRAIT_CLAIMS_AND_CONFORMANCE_2026-11.md`](SEMANTIC_TRAIT_CLAIMS_AND_CONFORMANCE_2026-11.md) | 意味的特性の宣言・観測・適合：閉値域／MECE／属性同一性の横断設計（2026-11） |
| [`TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md`](TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md) | TEI Authoring Source を .tei.md へ統一する段階移行計画 |
| [`TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_LOWERING_2026-11.md`](TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_LOWERING_2026-11.md) | *.tei.md ドメイン知識の図表表現からコード生成・ビルドへ到る縦断設計（設計候補、2026-11-11） |
| [`TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_IMPLEMENTATION_GUIDE_2026-11.md`](TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_IMPLEMENTATION_GUIDE_2026-11.md) | *.tei.md ドメイン図表→Build 実装導線（P0-A成立後、P1-B〜P3-Dの受入・依存Owner・再利用Fixture） |
| [`TEI_SHO_OKF_DOMAIN_LINT_DOCUMENT_UNIT_2026-11.md`](TEI_SHO_OKF_DOMAIN_LINT_DOCUMENT_UNIT_2026-11.md) | TEI/SHOのOKF知識バンドルとドメインLint：段階導入の検証計画 |
| [`TEI_SHO_DOCUMENT_DRAFT_INTEGRATION_2026-11.md`](TEI_SHO_DOCUMENT_DRAFT_INTEGRATION_2026-11.md) | TEI受理済みモデルの一要素と一関係を、SourceとRevisionを保持した説明文書DraftとしてSHOへ渡し、編集・保存・再開・モデル改版時の再確認までを行う最小統合機構（[SHO #19](https://github.com/hat47x/sho-composer/issues/19)・[SHO #4](https://github.com/hat47x/sho-composer/issues/4)・[SHO #6](https://github.com/hat47x/sho-composer/issues/6)、[#3394](https://github.com/hat47x/tei-platform/issues/3394)のSHO文書段、実装済みの最小基線） |
| [`TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md`](TEI_SHO_DOCUMENT_DRAFT_NEXT_STAGES_2026-11.md) | 上の契約を段階2〜5（CRUD表・三要素分析図・仕様書・固定納品版）へ伸ばす実行計画と、相互CI・Studio実組込み・逆方向禁止・`*.tei.md`取込みの横断課題、Owner判断事項の一覧（実装前設計） |
| [`TEI_STUDIO_SHO_EMBEDDING_FORM_2026-11.md`](TEI_STUDIO_SHO_EMBEDDING_FORM_2026-11.md) | TEI Studio から SHO 文書編集部品を利用する形態の実測と決定（子プロセス起動を維持）・失敗分類6種・未測定（配布・Linux/macOS）（代行決定済み） |
| [`DOCUMENT_DRAFT_CITATION_AND_OCCURRENCE_ID_2026-11.md`](DOCUMENT_DRAFT_CITATION_AND_OCCURRENCE_ID_2026-11.md) | Draft 引用の解決（`recorded-not-resolved` のまま）と Occurrence ID の永続性（snapshot-local のまま）の代行決定と根拠・撤回条件 |
| [`TEI_MD_READ_SIDE_DESIGN_2026-11.md`](TEI_MD_READ_SIDE_DESIGN_2026-11.md) | SHO 保存物（`*.tei.md`）の TEI 側読み取り設計。読み取り側が答えるべき4つ（R-01〜04）・Parser完了後も変えない境界・Parser 入出力契約。実装は Parser（#3346）完了後（設計のみ） |

## 関連する参照事例と評価

`DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`の評価順序は、後続の読み取り専用の検証結果に合わせ、R1～R6へ反映済みです。当初、変更や古い状態の再評価として扱う予定だった論点は、R3～R5を先に個別に評価したうえで、R6のスナップショットの失効として扱います。

R3の初期構想にある`proposal_origin`は、現在の本番用フィールドの候補ではありません。R5と、異なる種類の候補が混在する場合の精緻化を優先し、候補が表す意味と、発見経路・出所の記録を別の観点で扱います。

## 設計案の統合で判定段階C前に立ち止まる地点

設計案の統合とローカル認知を扱う作業領域Dでは、判定段階Cより前に安全に進められる読み取り専用の設計評価として、R1～R6、S1・S2の候補群の検証、異なる種類の候補が混在する場合の文脈の精緻化、提供実装の実現可能性と結果の分類に関する検証、S1～S5の受理に関する素案までを実施しました。

ここまでの評価で確認した現行の境界は、次のとおりです。

```text
candidate discovery
  -> exact / Pattern / associative provenanceを分離

candidate qualification
  -> current accepted authorityへ戻る

cross-view proof
  -> supported / contradiction / unresolvedを分離

candidate contraction
  -> exact scopeのcontradictionだけを決定論的に除外
  -> unresolvedは残余dimensionとして保持
  -> frontierを必ず1 candidateへ縮約しない
  -> survived != supported
  -> supported existing candidate != preferred / winner / accepted candidate

provider feasibility
  -> semantic candidate != realization / binding variant
  -> missing provider Evidenceはsemantic contradictionにしない
  -> provider-specific incompatibilityはmatching realization variantへ閉じる
  -> hard Target / Provider requirement時だけcomposite candidate eliminationへ拡張する
  -> provider limitationからBusiness / Canonical meaningを逆authorしない
  -> same error surfaceをsame resultへ畳まず、reason / owner / scopeへ戻す
  -> missing Evidence / operational recoveryはunresolvedとして保持する
  -> observed incompatibility / structural unsupportedだけをexact realization claimへ縮約する

context packet
  -> candidate subject != discovery trace != candidate-local proof / qualification
  -> R3初期の単一proposal_originへ戻さない
  -> supported existing + unresolved proposalをflattenせずS3 / human reviewへ渡す

proof freshness
  -> required accepted authorityのsemantic fingerprintへ束縛
  -> stale contradictionをcandidate eliminationへ使わない

LLM boundary
  -> bounded residualをProposalとして扱う
  -> minimum semantic deltaはS2 proofではなくS3 / human comparisonで扱う
  -> Acceptance / authorityを所有しない
```

判定段階Cより前には、これらの結果だけを根拠として、次の項目を本番向けの契約として固定しません。

- 正本に新設する要素の種類としての`DesignCandidate`
- 本番用の`DesignContextPacket`スキーマ
- 候補の出所を表す排他的な列挙値
- 証明の有効期間や結果を保存する列挙値
- 永続化する`AcceptanceSnapshot`や証明の評価時点の記録
- 本番用の構成パターンの記憶スキーマ
- ローカル・外部の言語モデルを提供する仕組みの契約
- ベクトルデータベース、グラフデータベース、汎用の制約解決機構
- 共通の受理境界での自動的な受理
- 意味の差分を永続化するモデル
- フィードバックの出来事を分類する体系
- 本番用の`ProviderFeasibility`列挙値や候補の階層
- 提供実装のエラーと結果の分類体系を本番向けに固定すること

新しい読み取り専用の検証を追加するのは、R1～R6、S1・S2の候補群の検証、混在する候補の文脈の精緻化、提供実装の実現可能性と結果分類の検証だけでは説明できない問題が、現在の参照事例で見つかった場合に限ります。対象となるのは、具体的な情報の欠落、権限上の矛盾、出所・情報の鮮度・対象範囲・担当への振分けに関する不足です。将来必要になりそうだという理由だけで、抽象化や試験専用の構造を増やしません。

### 判定段階Cの後に引き継ぐ条件

判定段階Cの後に`ISSUE-000047`と`ISSUE-000051`を再開するときは、R5・R6で確認済みの境界を、別のスキーマや認知処理の層として設計し直しません。

`ISSUE-000047`には、次の確認済みの条件を引き継ぎます。

- 意味上の候補と、候補を発見する方式・発見経路の記録を分ける
- 同じ候補を正確な照合、構成パターン、連想処理などの複数の経路で発見しても、候補を複製せず、複数の発見経路を記録できる
- 参照パターンの出所、連想検索の表現を構築した入力の出所と対象範囲内での鮮度、現在のM1の適格性確認をそれぞれ区別する
- 検索の評価点、類似度、過去の利用回数を、適用可能性・検証根拠・受理の判断へ直接変換しない
- 正確に一致する既存資産も、存在するだけで適用可能とは判断しない。現在の識別子、依存関係、対象範囲、必要な受理済みの権限・検証根拠に戻って適格性を確認する

`ISSUE-000051`には、次の確認済みの条件を引き継ぎます。

- 派生した認知用の表現はM2の再生成可能な資源であり、M1で意味を決定する権限ではない
- 候補を検索する方式を変えても、意味上の候補の識別子や受理済みの意味を変えない
- 派生索引を構築した入力の出所・鮮度・失効の判定と、M1で受理済みの権限・証明の鮮度を混同しない
- 派生した表現が最新でも、現在の適用可能性や複数視点の証明は、必要な受理済みの権限と検証根拠に戻って独立に確認する
- R6における証明の鮮度は、証明が依存する受理済みの権限の照合値に関する問題であり、認知用索引のためにリポジトリ全体で共有する改訂番号を導入する根拠にはしない
- 索引を削除したり検索方式を変更したりしても、意味の正本全体が変わらないことを維持する

したがって、再開時には次の仕組みを改めて導入するところから始めません。

```text
candidate.origin = exact | pattern | associative
という排他的origin enum

retrieval score -> confidence / evidence strength
というauthority変換

repository全体のglobal revision counter

cognitive index / pattern memoryを
accepted semantic sourceとする構造
```

実際の判定段階Cの参照事例で、これらの境界による情報の欠落が観測された場合に限って再検討します。判定段階Cの後は、境界を設計し直すのではなく、まず実際のS1の候補群に`ISSUE-000047`の現行の適格性確認を適用します。必要性が確認された場合に限り、`ISSUE-000051`で検索方式を比較します。

### 再開条件

通常は製品の判定段階Cの後に再開します。少なくとも次の条件が実際に成立してから、S1～S5の参照事例を用いた検証に進みます。

```text
Gate B Ready
  -> Handover Baseline B / baseline revision R
  -> First Web Target build T1
  -> Gate C Evidence
```

```text
S1 realistic intent / candidate frontier
  -> S2 incremental cross-view contraction
  -> actual information-lossを見てbounded packetを調整
  -> S3 local LLM / minimum semantic delta
  -> Shared Acceptanceで小さいsemantic changeを1件Accept
  -> S4 implementation / test derivation
  -> S5 one observable feedbackをmeaning ownerへroute
  -> closed loopを1周
```

この一連の改善循環を実際に検証した根拠が得られるまでは、R1～R6の試験専用の構造や、S1・S2の候補群、混在する候補の精緻化、提供実装の実現可能性に関する概念上の構造を、本番用のスキーマには採用しません。

## まだ設計文書を置いていない領域

ここには、実装、ADR、準拠性の検証資料は存在するものの、機能単位の設計本文はまだ置いていない領域を示します。設計本文は必要になった時点で追加し、それまでは対応するADRと準拠性の検証資料を正本とします。

| 領域 | 現在の正本 |
| --- | --- |
| 情報源から正本の意味への追跡と入力元索引 | [ADR-0004](../adr/0004-source-to-canonical-traceability.md)、[`docs/conformance/source-traceability-m0.23.md`](../../docs/conformance/source-traceability-m0.23.md) |
| データベースの物理スキーマに関する期待状態・実際の状態・差分・未解決事項、高度な索引、移行用の生成処理 | ADR-0008〜ADR-0015、`docs/conformance/`のm0.26〜m0.35 |
| プラグインとプロセスの実行基盤、権限、資源の仲介、古い実行担当の排除・担当の交代 | [ADR-0005](../adr/0005-supervisor-and-runtime-fencing-authority.md)、[ADR-0006](../adr/0006-process-command-and-argv.md)、`docs/conformance/`のm0.22.1 / m0.24 / m0.25 |
| 実行先と表示・操作の境界 | 判定段階Cより前のため未確定。[Gate C Evidence契約](../gate/GATE_C_EVIDENCE_CONTRACT.md)とISSUE-000026 |
| [`AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md`](AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md) | （追記） |
| [`COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md`](COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md) | （追記） |
| [`FOUNDATIONAL_DATA_CONCEPT_SPACE_2026-11.md`](FOUNDATIONAL_DATA_CONCEPT_SPACE_2026-11.md) | （追記） |
| [`REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md`](REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md) | （追記） |
| [`SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md`](SCRIPT_MACRO_AUTHORITY_AND_EFFECT_TARGET_BINDING_2026-11.md) | （追記） |
| [`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) | （追記） |
| [`STANDARD_REPRESENTATION_TOPOLOGY_AND_MEDIA_AXES_2026-11.md`](STANDARD_REPRESENTATION_TOPOLOGY_AND_MEDIA_AXES_2026-11.md) | （追記） |
| [`HASH_DERIVED_ID_SCHEME_DESIGN_2026-11.md`](HASH_DERIVED_ID_SCHEME_DESIGN_2026-11.md) | （追記） |

判定段階の前に確定していない領域について、先回りして設計文書は作りません。

## 直近の構成議論

- [TEI構成議論の整理と次の設計判断（2026-10-24）](TEI_CONFIGURATION_DISCUSSION_SYNTHESIS_2026-10-24.md)：独自Host非必須、TEI内での業務シミュレーション、DSL・プラグインの交差的利用、同じ版からのコード・設計書出力を既存Issueへ対応付ける検討メモ。採用・実装完了の宣言ではありません。
- [定義形式の版と変換の境界（2026-11）](FORMAT_VERSION_CONVERSION_2026-11.md)：`.tei`と作業領域manifestの形式版、中心表現経由の変換、プラグインが実行する変換境界、三状態属性の文法提案と残作業（#1920）。
- [実行先の現状調査](HOST_TARGET_SURVEY_2026-11.md)（Web・デスクトップ・サービス・Wasmの現在地。モバイル・組み込みは未着手）
- [権限の現状調査](PERMISSION_ENFORCEMENT_SURVEY_2026-11.md)（宣言の検証はあるが、実行時の強制は無い）
- [アクセス経路の棚卸し](ACCESS_PATH_INVENTORY_2026-11.md)（実行の中核が直接触るファイル操作は1件。実際のアクセスはアダプタ側）
- [権限の強制の適用計画](PERMISSION_ADOPTION_PLAN_2026-11.md)（要求＝能力、付与＝環境。実行時に再確認して記録する）
- [動的な出力パスと宣言スコープ](DYNAMIC_PATH_GRANT_DESIGN_2026-11.md)（環境の付与は静的、実行時のパスは動的。両者の不一致と選択肢）

## 機能設計文書

ここには、複数のIssueにまたがる機能について、意味モデル、記述方法、検証、実行時の境界をまとめた設計文書を置きます。採用前の案も、対象となるIssueと文書の状態を冒頭に明記して管理します。

- 作業内容と完了条件：GitHubのIssue
- 方針を採用した理由：[`../adr/`](../adr/README.md)
- 作業の順序と必要な検証根拠：[`../README.md`](../README.md)
- 現行実装の構成と技術的な境界：[`../../docs/architecture.md`](../../docs/architecture.md)
- 実装と検証根拠の現状：[`../../docs/status.md`](../../docs/status.md)
- 画面の試作例：[`mockups/`](mockups/README.md)

設計文書はIssueやADRの代わりにはなりません。作業の進行状況はIssue、採用した設計判断はADRを正本とします。ここには、それらの記録だけでは一つにまとまらない、機能単位の設計本文を置きます。小さな機能については新たな文書を作らず、Issue本文に記します。

`docs/`とは役割を分けます。`docs/`では実装済みの構成と利用方法を説明し、ここでは採用前の案や変更に必要な機能設計を扱います。実装状況が変わった場合は、この設計文書を現行構成の説明に流用せず、`docs/architecture.md`を更新します。
- [設計書から実行結果までをつなぐ製品設計](PRODUCT_COMPLETION_DESIGN.md)

- [成果物の入力契約（設計記録）](ARTIFACT_INPUT_CONTRACT_2026-11.md)
- [SQLite introspection の制限と未対応事項](SQLITE_INTROSPECTION_LIMITATIONS_2026-11.md)
- [共通の例外・失敗モデル（2026-11）](UNIFIED_FAILURE_MODEL_2026-11.md)
- [2waySQL-like記述との整合性](WAY_SQL_ALIGNMENT_2026-11.md)
- [DriverシナリオのMarkdown記述（設計候補）](DRIVER_SCENARIO_MARKDOWN_AUTHORING_2026-11.md)
- [組織データ保護の境界（2026-11）](ORGANIZATION_DATA_PROTECTION_BOUNDARY_2026-11.md)
