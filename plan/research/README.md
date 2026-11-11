# 調査と設計候補の評価

技術や設計の候補を比べ、採用判断に必要な根拠を集める文書を置きます。ここにある評価結果だけで設計方針が採用されたことにはなりません。採用した判断は[`adr/`](../adr/README.md)に記録します。

この下の一覧は、このディレクトリの**全文書**を載せています（`README.md` を除く）。`tools/repository-plan-check` の `TestPlanReadmesMentionEveryEntry` が一致を検査します。

- [DeepSeek Harnessの構成・Headless・MCP/ToolとTEIの責務比較](DEEPSEEK_HARNESS_TEI_CROSS_PRODUCT_ADOPTION_2026-11-01.md)
- [DSH Experimental Agent TeamsのTeam Scope・共有Checkout・CAS/Task・SEI専門照会・用途別Context](DSH_AGENT_TEAMS_CONTEXT_EXPERT_HANDOFF_2026-11-02.md)
- [DSHのGoal・Schedule・外部送信・大規模並列に関する調査](DSH_SCALE_GOAL_SCHEDULE_EGRESS_RESEARCH_2026-11-01.md)
- [Harness証拠の連続性と選択的な再資格化](HARNESS_EVIDENCE_CONTINUITY_AND_SELECTIVE_REQUALIFICATION_2026-11-02.md)
- [記述判断の根拠評価](AUTHORING_DECISION_AUTHORITY_EVIDENCE_GATE_2026-10.md)
- [高度組織を扱う商用製品の要求比較とTEI適合性監査（2026年11月）](COMMERCIAL_ORGANIZATION_PLATFORM_REQUIREMENTS_2026-11.md)：調査記録・採用前Requirements Probe。各社の公開資料と現行設計を照合した暫定的判断で、要求の採用や受入条件の決定ではありません。
- [データ決定子と参照受入条件の評価](DATA_DETERMINANT_ASSOCIATION_REFERENCE_ACCEPTANCE_CONTRACT_2026-10.md)
- [データ決定子の中核契約案](DATA_DETERMINANT_CORE_CONTRACT_CANDIDATE_2026-10.md)
- [データ決定子の昇格条件](DATA_DETERMINANT_PROMOTION_GATE_2026-10.md)
- [意味拡張の依存解決に関する事前評価](SEMANTIC_EXTENSION_DEPENDENCY_PRE_ADR_2026-10.md)
- [意味拡張の依存解決に関する再評価](SEMANTIC_EXTENSION_ADVERSARIAL_REVIEW_2026-10.md)
- [現行設計の空白と外部方式の調査（2026年10月）](DESIGN_GAP_RESEARCH_2026-10.md)
- [プラグイン間の依存と入出力に関する他のOSSの調査](PLUGIN_DEPENDENCY_IO_OSS_SURVEY_2026-10.md)
- [技術調査の統合方針（2026年9月）](TECH_RESEARCH_INTEGRATION_2026-09.md)
- [技術調査の統合方針（2026年10月）](TECH_RESEARCH_INTEGRATION_2026-10.md)
- [Model Viewの読取・検証・同一性・被覆・操作の横断資格（2026年11月）](MODEL_VIEW_CROSS_CONSUMER_QUALIFICATION_2026-11.md)：実CLI・Catalog・Studio・Model Mapの局所成功を過剰な設計Claimへ昇格しない受入候補。
- [直近の技術研究を検証可能な作業に落とす計画（2026年10月16日）](TECH_RESEARCH_EXECUTION_PLAN_2026-10-16.md)
- [X-TEA（XEAD）のローコード機構に関する知識目録（2026年11月）](XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)
- [空集合と業務完了の意味（2026年11月）](EMPTY_SET_COMPLETION_SEMANTICS_2026-11.md)：全称条件の空集合（vacuous truth）とLeanでの扱いを、業務類型・EmptyPolicyとの関係で整理する調査ノート。採用判断ではありません。
- [TEI × SEI：ローカルCPU認知による実装再利用・標準部品化候補の研究](LOCAL_CPU_COMPONENT_REUSE.md)：研究仮説。未実装で、製品への採用は決まっていません。
- [TEIの汎用モデリング能力を業務外領域で確かめる限定実証（CSW・VSクマ）](TEI_CROSS_DOMAIN_PROBE_CSW_VS_BEAR_2026-11.md)：CSWとVSクマを独立対象にした研究・比較・read-only probe。製品アーキテクチャを変更せず、採用判断も行っていません。
