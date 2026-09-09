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
Gate B — Not Ready
  #310 + #255 + #308 same-R actual Evidence待ち
        ↓
Handover Baseline B / revision R
        ↓
First Web Target build T1
        ↓
Gate C — PAS-04
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

**Not Ready**です。

Required Evidenceは次の三つだけです。

| Evidence | Issue | 状態 |
| --- | --- | --- |
| deterministic Repository | `ISSUE-000032` / #310 | actual clean checkout run待ち |
| generated artifact reproducibility | `ISSUE-000025` / #255 | actual clean check-only待ち |
| real PostgreSQL Provider + ephemeral fixture | `ISSUE-000031` / #308 | Evidence専用空PostgreSQL actual run待ち |

三つを同じclean candidate revision `R`へ揃え、#387 final convergenceで確認します。

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、Gate Bの第四Required laneではありません。未完了ならHandover `unresolved`へ渡します。

## Gate B → Gate C Handover

- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)
- [`issue/000030-gate-b-to-c-handover-baseline.md`](issue/000030-gate-b-to-c-handover-baseline.md)

Gate B Ready candidate revision `R`をHandover Baseline `B`として固定します。

Handoverは新しい番号付きGateでもCanonicalの第二正本でもありません。

## Gate C / First Web Target

- [`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)
- [`issue/000007-first-web-target-projection.md`](issue/000007-first-web-target-projection.md)
- [`issue/000028-first-web-target-conformance-evidence.md`](issue/000028-first-web-target-conformance-evidence.md)

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

StudioはCanonicalの第二正本ではありません。

### knowledge-first / PAS-06

- [`issue/000047-knowledge-first-semantic-exploration.md`](issue/000047-knowledge-first-semantic-exploration.md)

Human / AIが同じauthoritative Source / Canonical / Evidenceをread-only探索し、`applicable / incompatible / unknown`を分けます。

search ranking / similarityをSemantic Evidenceへ格上げしません。

### AI Proposal / PAS-01 / PAS-06

- [`adr/0022-ai-proposal-canonical-acceptance-boundary.md`](adr/0022-ai-proposal-canonical-acceptance-boundary.md)
- [`issue/000027-ai-proposal-acceptance-boundary.md`](issue/000027-ai-proposal-acceptance-boundary.md)

AI ProposalはCanonical authorityではなく、file-native / Studio / CLI等とShared Acceptance Boundaryへ収束します。

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

## current snapshot方針

Master Plan、v1 Plan、Milestones、Gap Audit、Gate契約等の上位文書は、判断履歴を本文へ無制限に追記せず**現在の正規baseline**を優先します。

過去の判断経緯はGit history、ADR、完了 / superseded Plan Issueから追跡します。

同じ実行手順やsemantic boundaryを複数文書で競合する正本にしません。

