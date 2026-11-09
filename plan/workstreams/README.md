# 複数作業にまたがる計画

機能やIssueをまたいで、進め方・移行順・調査対象を定める計画文書を置きます。採用済みの設計判断は[`adr/`](../adr/README.md)、機能の内部設計は[`design/`](../design/README.md)、個々の作業と完了条件はGitHub Issuesを参照してください。

この下の一覧は、このディレクトリの**全計画文書**を載せています（`README.md` を除く）。`tools/repository-plan-check` の `TestPlanReadmesMentionEveryEntry` が一致を検査します。

- [AI大量並列開発と要件ポートフォリオ：外部要求からTEIの共通構造を育てる計画](AI_HIGH_THROUGHPUT_REQUIREMENTS_AND_PLUGIN_EXPANSION_2026-11.md)（2026-11-01の実行計画案。正式なメタモデル・業務意味・受入条件の採用ではない）
- [クリティカルパスの暫定代行判断（2026-10-23）](TEI_PROVISIONAL_DELEGATED_DECISIONS_2026-10-23.md)（所有者の指示により開発エージェントが代行した判断の記録。所有者の確認で覆せる）
- [TEI技術調査Phase A: 一改訂の実行・観測証拠の受渡し](TEI_RESEARCH_PHASE_A_EVIDENCE_HANDOFF_2026-10-16.md)（実施準備・未認定）
- [参照モデルRelationship移植: Type固有構造・CrossView・証拠接合の実証計画](TEI_RMT_RELATION_CROSSVIEW_ASSURANCE_PLAN_2026-11.md)（計画はmain統合済み。新追補は提案）
- [設計可読性・意味Authoring・標準Consumerの横断接続計画](TEI_SEMANTIC_AUTHORING_CONSUMER_INTEGRATION_2026-11.md)（Draft。新しいCanonical schema等の採用ではない）
- [Data・Function・Businessの意味整合と移植・変更管理の受入引渡し](TRIAD_MEANING_AND_CROSSLANE_ACCEPTANCE_HANDOFF_2026-11.md)（既存Ownerへ渡す計画候補。業務意味の採用やGate合格ではない）
- [X-TEA知見の移行計画](XTEA_LOWCODE_TRANSFER_PLAN_2026-11.md)（計画案・未採用）
- [Everything is Dataの横断接合・同一版受入の計画](EVERYTHING_DATA_CROSSLANE_HANDOFF_2026-11.md)
- [業務状態の保存方式・時点・権限を同一Scenarioで比較する受入ハンドオフ](BUSINESS_STATE_PERSISTENCE_DECISION_GATE_2026-11.md)
- [定義を中心にしたフィードバックループの計画](DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md)
- [定義と派生物の一覧](DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md)
- [フィードバックの振り分け一覧](DEFINITION_FEEDBACK_ROUTING_CATALOG_2026-09.md)
- [知識移植の初期境界](KNOWLEDGE_PORTABILITY_R0_BOUNDARY_2026-09.md)
- [業務手順の参照実験](ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md)
- [SUI移行設計の案内](SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md)
- [部分モデルを再検討する条件](SUBSYSTEM_CANONICAL_REOPEN_GATE_2026-09.md)
- [三視点ローコード実証の計画](TEI_PRODUCT_CRITICAL_PATH_2026-10.md)
- [増分ビルドの再利用・Evidence・再資格化の実行対照手順](INCREMENTAL_REUSE_EVIDENCE_EXECUTION_PROTOCOL_2026-11.md)
- [Issueの深掘り対象の選定と作業順序](ISSUE_DEEPENING_2026-11.md)
- [深掘り設計の上流接続](UPSTREAM_CONNECTION_2026-11.md)
- [TAI上流への立ち戻りと、5課題の整合](TAI_UPSTREAM_ALIGNMENT_2026-11.md)
- [PoCの仮説と、上流へ戻る条件](POC_HYPOTHESES_2026-11.md)
- [判断材料: マイルストーンと契約枠への位置づけ](MILESTONE_CONTRACT_DECISION_BRIEF_2026-11.md)
- [判断材料: 操作面の未決4件と、決める順序](OPERATION_SURFACE_DECISION_BRIEF_2026-11.md)（公開面・語彙の置き場所・`unattended` の条件・drift 照合の既定について、`main` 上の4つの実行記録を束ね、決めると後続が解ける順序と、最初に答えるべき「照合値をどこから渡すか」を示す。決定そのものは行わない）
- [判断材料: 実行の側をどこで宣言へ繋ぐか](COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)（6本の実行記録から、宣言→検証→計画→実行の間に繋ぎ目が無い形を示し、C-1 routing.Router・C-2 servicehost・C-3 能力IDの解決・C-4 繋がない、の4案と決める順序を整理する。どれも推薦せず、C-4を正当な選択肢として扱う。決定はGateと成立状況の担当）
- [上流計画から担当作業への展開](UPSTREAM_BREAKDOWN_2026-09.md)
- [X-TEA設計を参照する際の代替手順](UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md)
- [TEI自身の設計をCLIで扱うドッグフーディングの実測と課題](TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11.md)（#3149。read-only CLIでTEI自身の設計記述を発見・読取り・検証し、10件の課題を起票対象として整理。修正は未着手）
- [TEI自身の設計をCLIで扱うドッグフーディング 第2ラウンド](TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11.md)（#3149。現行main `4fd88c2b1` の実行物で再実測し、D-1・D-2・D-4〜D-8 が閉じたことと、D-3・D-9・D-10 が残ることを確認。MCP入口は未実装のため CLI 経路のみ実測）
- [Script/Macroの製品化と実証計画](TEI_SCRIPT_MACRO_PRODUCT_PLAN_2026-11.md)
