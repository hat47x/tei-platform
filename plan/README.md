# TEI Planning

[プロダクト開発ロードマップ（HTML）](product-roadmap.html)は、期限を設けず、着手依存と完了条件で開発順を整理した横断ガイドです。[想定仕様](../docs/specification/index.html)と相互参照し、正式なGate・採用判断は既存正本に従います。

`plan/`には、Product Value / Requirementsを実際の着手順、Product Acceptance Scenario、ADR、Reference、Evidence、Gateへ接続するplanning文書を置きます。

- 何を守るか: `product/`
- 何を満たすか: `requirements/`
- どのScenarioで確かめるか: `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- どの順で不確実性を減らすか: `plan/`

## 最初に読む文書

1. [`MASTER_PLAN.md`](MASTER_PLAN.md) — 現在のworkstream /主線
2. [`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md) — `order_entry`でのv1実証順
3. [`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md) — RQ → PAS → Evidence trace
4. [`PRODUCT_REQUIREMENT_GAP_AUDIT.md`](PRODUCT_REQUIREMENT_GAP_AUDIT.md) — 現在のRequirement Evidence / Gap
5. [`MILESTONES.md`](MILESTONES.md) — 期限なしのCapability horizon
6. [`UPSTREAM_BREAKDOWN_2026-09.md`](UPSTREAM_BREAKDOWN_2026-09.md) — 上流pressure / audit文書の結論を計画と下流の担当へ割り当てた導出表

## 現在のv1主線

```text
Application Pattern Stage 1 — Done
        ↓
Gate A — Ready
        ↓
Application Driver — major semantics ready
        ↓
Gate B — Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
Handover Baseline B / revision R — 固定済み
        ↓
First Web Target build T1 — Evidence取得済み（T = 0ba87a4e0a96…）
        ↓
Gate C — PAS-04 — Ready
        ↓
   ┌──────┼──────────────┐
   ↓      ↓              ↓
Studio  knowledge / AI  Second Target
PAS-01  PAS-06 / 01     PAS-04
```

Gate C後のStudio / knowledge-first / AI Proposalと第二Targetは並行可能です。

## Gate A

- [`GATE_A_READINESS_AUDIT.md`](GATE_A_READINESS_AUDIT.md)
- 現在: **Ready**

Application Pattern、Datasource、Validation、Authentication / Authorization、Capability Binding、Business Result / Execution Error等をDriverへ渡せる意味基線が成立しています。

## Gate B

### 正本文書

- [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md) — 何のEvidenceが必要か / 何を証明するか
- [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md) — 唯一の具体実行手順
- [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md) — Ready / Not Ready判定
- [`issue/000037-gate-b-three-lane-final-evidence.md`](issue/000037-gate-b-three-lane-final-evidence.md) — three-lane final convergence

### 現在

**Ready**です（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

Required Evidenceは次の三つで、いずれもsame-run same-Rで成立しています。

| Evidence | Issue | 状態 |
| --- | --- | --- |
| deterministic Repository | `ISSUE-000032` / #310 | 成立 |
| generated artifact reproducibility | `ISSUE-000025` / #255 | 成立 |
| real PostgreSQL Provider + ephemeral fixture | `ISSUE-000031` / #308 | 成立 |

#387 final convergenceと独立artifact audit（`artifactAuditPassed = true`）の詳細は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を参照してください。

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、Gate Bの第四Required laneではなく、未完了のままHandoverへ`unresolved`として渡しています。

## Gate B → Gate C Handover

- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) — Handoverの規則・原則
- [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](GATE_B_TO_C_HANDOVER_BASELINE_B.md) — 実際に固定したBaseline `B / R`の記録
- [`issue/000030-gate-b-to-c-handover-baseline.md`](issue/000030-gate-b-to-c-handover-baseline.md)

Gate B Ready candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`をHandover Baseline `B`として固定済みです。

Handoverは新しい番号付きGateでもCanonicalの第二正本でもありません。

## Gate C / First Web Target

- [`GATE_C_READINESS_AUDIT.md`](GATE_C_READINESS_AUDIT.md) — Ready判定の記録
- [`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)
- [`GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`](GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)
- [`issue/000007-first-web-target-projection.md`](issue/000007-first-web-target-projection.md)
- [`issue/000028-first-web-target-conformance-evidence.md`](issue/000028-first-web-target-conformance-evidence.md)

Gate Cは2026-09-19に**Ready**と判定しました（`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`）。判定記録は[`GATE_C_READINESS_AUDIT.md`](GATE_C_READINESS_AUDIT.md)です。Visual Fidelityの評価者は単一のAI評価者で、人による確認ではありません。

First Web Targetのbuild revision `T1`をbaseline revision `R`と分けて追跡します。

```text
Handover B at R
      ↓ meaning baseline
Web build T1
      ↓
Gate C Evidence
```

Gate Cでは次を分離します。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence
4. 必要に応じたPerformance Evidence

Target側でbaseline meaningを変更する必要がある場合、Web固有field / ruleで埋めず、successor Handoverを発行します。

## Business system design / PAS-07

この領域はGate B / Gate C Required Evidenceへ追加する新しいGateではなく、RQ-025 / RQ-026を現実的なBusiness Referenceで圧力試験するparallel workstreamです。

根幹から次の順で読みます。

1. [`../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md) — RQ-025 / RQ-026の要求解釈
2. [`../docs/business-system-design.md`](../docs/business-system-design.md) — Data / Function-Interaction / BusinessのArchitecture境界
3. [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](design/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md) — false completenessを避け、meaning / evidence / authority / actuality / responsibilityをreviewする品質モデル
4. [`issue/000042-business-procedure-cross-view-reference.md`](issue/000042-business-procedure-cross-view-reference.md) — Lane C Reference / Pressure inventory
5. [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md) — second realistic ReferenceのRRB-P1 / P2 / P3 acceptance / maturity条件
6. [`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md) / [`issue/000048-business-procedure-contract-adr-readiness.md`](issue/000048-business-procedure-contract-adr-readiness.md) — Contract ADR readiness
7. [`issue/000050-second-realistic-handoff-corpus.md`](issue/000050-second-realistic-handoff-corpus.md) — acceptance contractを製造 / 在庫candidateへ適用し、independent Business seedからsecond realistic Handoff Evidenceを取得する計画（2026-09-21: 実施済み。実施記録あり）

現在の品質上の中心は、Business Procedureを「完全に埋めたflow」として作ることではありません。

```text
known / accepted meaningをexactに保つ
+ derived / proposal / unresolvedを混同しない
+ Business instruction / system result / external actuality / Business acceptanceを短絡しない
+ responsibility Handoffをchronologyやactor差から発明しない
```

second corpusもHandoff fixtureから始めず、Handoffが無くても価値を持つindependent Business need / seedを先に成立させます。**Lane C Business Procedure Gate A**のactual green前はprovenance / candidate design question / RRB-P1適用準備までに留め、accepted Procedure / HandoffをBlocker解消のために捏造しません。

ここでいうLane C Business Procedure Gate Aは、本README前半のApplication Pattern **Gate A — Ready**とは別のPre-ADR execution gateです。名称を理由に両者のEvidence statusを混同しません。

## Definition-centered feedback / AI era

Studio、knowledge-first探索、AI Proposal、非LLM認知方式を別々のAI機能として育てず、**端的なauthoritative definitionを中心にした一つのfeedback loop**へ接続します。

設計本文:

- [`../docs/definition-centered-feedback-loop.md`](../docs/definition-centered-feedback-loop.md) — concise definition / three-memory separation / feedback routing / minimum semantic delta / Reference Modelの役割
- [`DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md`](DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md) — 上流の複数設計案をローカル認知で縮約し、local LLMとsemantic-to-implementation/test derivationへ接続する上位Product Goal
- [`DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`](DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md) — Gate C前後の実証順と既存Issueの責務統合

中心flow:

```text
concise authoritative definition
  ↓
Canonical meaning
  ↓
derived projection / runtime / simulator / index
  ↓
human review / Observation / Reference pressure
  ↓
route feedback to owning concern
  ↓
exact / pattern / cognition / LLM candidates
  ↓
deterministic qualification + Preview
  ↓
Shared Acceptance Boundary
  ↓ accepted minimum semantic delta
authoritative source → reload / regenerate
```

ここで、LLMやcognitive indexを新しいauthorityにしません。Reference Modelはanswer libraryではなく、worked example・実行可能content・Conformance pressure・pattern evidence・Platform feedback sourceとして扱います。

Design Synthesisの長期目標は、上流purposeから複数のstructured design candidateを作り、exact / constraint / pattern / cross-view / associative cognitionで候補空間を狭め、local LLMには少数案の比較・統合・minimum semantic delta提案を担わせることです。accepted meaningからimplementationとtestをできる限り再生成し、人間の反復authoringをmeaning selection / Acceptanceへ圧縮します。

Gate C前はdefinition inventory、feedback routing、memory separation、Reference pattern provenance等のread-only設計準備までです。Gate C後、`ISSUE-000046` / `000047` / `000027`を一つの小さいsemantic changeで合流させ、その後に`ISSUE-000051`のnon-LLM cognition比較へ進みます。

## Gate C後のplanning

### Studio / PAS-01

- [`issue/000046-studio-shared-acceptance-authoring-loop.md`](issue/000046-studio-shared-acceptance-authoring-loop.md)

最小loop:

```text
Select → Propose → Preview → Accept / Reject
                    ↓ Accept only
          Shared Acceptance Boundary
                    ↓
          authoritative source → reload
```

StudioはCanonicalの第二正本ではありません。Definition-centered loopでは、whole-model editingよりselected definition / local neighborhoodを基本にし、AIを使う場合もwhole file再生成ではなくreview可能なminimum semantic deltaを優先します。

### knowledge-first / PAS-06

この領域は、個別algorithmやfeature設計からではなく、Product上の趣旨から順に読みます。

1. [`DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md`](DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md) — 複数設計案、proposal-space contraction、local-first認知、local LLM、semantic-to-code/test derivationという上位Product Goal
2. [`DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`](DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md) — なぜ非LLM認知方式を検討するか、Canonical / LLM / Authorityとの境界、個別課題への展開順
3. [`DERIVED_COGNITIVE_REQUIREMENT_DRAFT_2026-09.md`](DERIVED_COGNITIVE_REQUIREMENT_DRAFT_2026-09.md) — SEI Cognition / SUI / ハエ脳等の関連検討からTEIへ取り込む性質を抽出した要件素案と所有境界
4. [`issue/000047-knowledge-first-semantic-exploration.md`](issue/000047-knowledge-first-semantic-exploration.md) — Human / AI共通のknowledge-first Product Reference
5. [`issue/000051-derived-cognitive-retrieval-evaluation.md`](issue/000051-derived-cognitive-retrieval-evaluation.md) — candidate retrieval方式を比較するarchitecture probe
6. [`DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md`](DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md) — 共通の評価問題とoracle
7. [`DERIVED_COGNITIVE_FEATURE_EXTRACTION_CONTRACT_2026-09.md`](DERIVED_COGNITIVE_FEATURE_EXTRACTION_CONTRACT_2026-09.md) — algorithm-neutralなfeature抽出詳細

Human / AIが同じauthoritative Source / Canonical / Evidenceをread-only探索し、`applicable / incompatible / unknown`を分けることは最初の実証スライスです。FlyHash、HDC/VSA等はその候補生成部分に価値があるかを比較する方式であり、最終目標はretrieval単体ではなく、設計候補の生成・縮約・Acceptance・implementation/test derivationまでfeedback loopを閉じることです。

### AI Proposal / PAS-01 / PAS-06

- [`adr/0022-ai-proposal-canonical-acceptance-boundary.md`](adr/0022-ai-proposal-canonical-acceptance-boundary.md)
- [`issue/000027-ai-proposal-acceptance-boundary.md`](issue/000027-ai-proposal-acceptance-boundary.md)

AI ProposalはCanonical authorityではなく、file-native / Studio / CLI等とShared Acceptance Boundaryへ収束します。AIの標準的な価値はwhole modelを再生成することではなく、current revision・explicit refs・Evidenceに基づく**小さい意味差分、比較、説明**を作ることです。

### Second Target / PAS-04

- [`issue/000026-target-capability-execution-profile-boundary.md`](issue/000026-target-capability-execution-profile-boundary.md)

First TargetだけでTarget-neutral taxonomyを完成させず、第二TargetでTarget Capability / Runtime Binding / Execution-Trust / RQ-024 performanceを圧力試験します。

Studioのdeveloper-facing Wails HostをApplication Target Evidenceへ数えません。

## ADR / Plan Issue / 機能設計

- [`adr/`](adr/) — 長く残す設計判断
- [`issue/`](issue/) — 複数PR / Reference / Evidenceへまたがる作業定義
- [`design/`](design/README.md) — 機能単位の設計本文。Architectureの第二正本にはしない

planning-onlyの将来作業まで不要にGitHub Issueへ二重起票することを目的にはしません。

Plan Issueの構成ルールは[`issue/README.md`](issue/README.md)を参照します。

## 判定文書と調査記録の役割

「監査」というファイル名だけで保管先を決めない。現在の判定や受入条件を管理する文書と、特定時点に実施した調査記録を区別する。

| 役割 | 正本または参照先 | 扱い |
| --- | --- | --- |
| Gate A・B・Cの判定 | [Gate A](GATE_A_READINESS_AUDIT.md)／[Gate B](GATE_B_READINESS_AUDIT.md)／[Gate C](GATE_C_READINESS_AUDIT.md) | 判定した対象・版・Evidenceに結び付く正式記録。現在の`main`の検証済みを意味しない。 |
| 製品要求の不足と根拠 | [要求別の不足監査](PRODUCT_REQUIREMENT_GAP_AUDIT.md) | 更新型の照合文書。要求と実証の正本を置き換えない。 |
| 業務手順の契約ADR作成可否 | [ADR作成前の判定](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md) | 当該契約の判定正本。他の調査や独立レビューだけでReadyに変更しない。 |
| 特定時点の調査結果 | [監査記録一覧](../docs/audits/README.md) | 判断を裏付ける参考記録。現行の採用判断やGateの第二正本にしない。 |

独立レビューや証拠の引継ぎに使う文書は、関連するIssue・判定文書との関係を確かめてから配置を変更する。旧パスの移動案内は参照をつなぐだけであり、本文の正本ではない。

## current snapshot方針

Master Plan、v1 Plan、Milestones、Gap Audit、Gate契約等の上位文書は、判断履歴を本文へ無制限に追記せず**現在の正規baseline**を優先します。

過去の判断経緯はGit history、ADR、完了 / superseded Plan Issueから追跡します。

同じ実行手順やsemantic boundaryを複数文書で競合する正本にしません。