# 上流工程文書のブレイクダウン — 2026-09

Status: **planning breakdown / 既存正本を変更しない / 新しいRequirement・Gate・Canonical schemaを定義しない**

## 1. この文書の役割

`plan/`直下に蓄積した上流工程文書（pressure / audit / trace 群）の結論を、**計画層**と**下流の担当**へ割り当てる導出表です。

- 何の正本か: 上流結論と下流担当の**対応付け**の正本
- 何の第二正本ではないか: Requirement（`requirements/`）、PAS trace（`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`）、Gate判定（各Readiness Audit）、Lane C判定（`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`）、作業状態（各Plan Issue）のいずれでもありません
- 扱う範囲: 上流文書の分類、結論の受け皿、受け皿が無い結論の下流起票、reopen triggerの所有先、配置上の課題
- 扱わない範囲: 個々の上流結論の再判定、Evidence取得状況の再記述、schema決定

可変な現在地（Gate判定、actual Evidence取得、Issue進行）はここへ複製しません。各正本を参照します。

## 2. 導出鎖

```text
Product Value / Vision / Principles      product/
        ↓
Product Requirement (RQ-xxx)             requirements/
        ↓
Product Acceptance Scenario (PAS-xx)     UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md
        ↓
上流pressure / audit（反証と境界の絞り込み）  plan/直下の BUSINESS_* / LANE_* / XTEA_*
        ↓
計画（順序・Gate・Milestone）              MASTER_PLAN / MILESTONES / GATE_*
        ↓
下流の担当（作業定義・設計判断・機能設計）    plan/issue/ · plan/adr/ · plan/design/
        ↓
Reference / Conformance / Evidence        reference/ · tools/ · docs/conformance/
        ↓
Gate / Product claim
```

上流文書は**候補を絞るための反証材料**です。pressure文書がPASSしたことをRequirement充足やGate通過へ読み替えません。

## 3. 上流文書の分類と受け皿

`plan/`直下の上流文書は現在95件前後あります。family単位で受け皿を固定します。

| Family | 件数目安 | 主な内容 | 結論の受け皿 |
| --- | --- | --- | --- |
| `BUSINESS_PROCEDURE_*` | 約35 | Business Procedure / Step address / Handoff / O1 / identity の候補絞り込み | Lane C判定は Pre-ADR Audit、Reference / Evidence は [ISSUE-000042](issue/000042-business-procedure-cross-view-reference.md)、契約起票は [ISSUE-000048](issue/000048-business-procedure-contract-adr-readiness.md) |
| `BUSINESS_ACTION_*` | 約8 | Business Action × CRUD / Data projection / ownership lens | ISSUE-000042（A80〜A83系のprojection）、Subsystem再検討は本書§6 |
| `BUSINESS_AUTHORING_*` | 約9 | authoring authority / loader / discovery / production採用条件 | [ISSUE-000049](issue/000049-business-authoring-production-adoption-readiness.md) |
| `BUSINESS_MODEL_*` | 約7 | Business semantic plane の配置と受理境界 | ISSUE-000048（契約決定時にADRへ）、placement reopenは本書§6 |
| `BUSINESS_HANDOFF_*` | 約5 | Handoff authority / second case 十分性 | [ISSUE-000050](issue/000050-second-realistic-handoff-corpus.md) |
| `LANE_B_*` | 約17 | A-set単位のtrace監査、RQ-002 / RQ-025 / RQ-026 trace | PAS trace正本は `RQ025_RQ026_ACCEPTANCE_TRACE.md`、A-set正本は ISSUE-000042 |
| `UPSTREAM_*` | 4 | PAS trace、vitality監査、trace監査、X-TEA fallback protocol | 本書と `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` |
| `XTEA_*` | 2 | X-TEA原典由来のEvidence register | ISSUE-000042 / 本書§6のreopen trigger |
| `GATE_*` / `MASTER_PLAN` / `MILESTONES` / `V1_*` / `PRODUCT_REQUIREMENT_GAP_AUDIT` | 10 | 計画管理文書 | それ自身が計画層の正本 |

同じ結論を複数の受け皿へ複製しません。上流文書の本文はそのまま残し、**判定と作業だけ**を下流へ移します。

## 4. Lane Cの三段readinessを下流へ割る

`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`が分離した三段を、担当Issueへ割り当てます。これがこのブレイクダウンの中心です。

```text
① Contract ADR readiness         → ISSUE-000048（新規）
② Production adoption readiness  → ISSUE-000049（新規）
③ PAS-01 / Studio / AI 統合      → ISSUE-000027 / ISSUE-000046（既存、Product Gate C後）
```

| 段 | 何が揃えば次へ進めるか | 担当 | 判定の正本 |
| --- | --- | --- | --- |
| ① | contractを反証できるEvidence。最終substrateの完成は前提にしない | ISSUE-000048 | Pre-ADR Audit |
| ② | production authoring adapter / shared Acceptance境界 / fresh process の3条件。partial write・partial authority switchを残さない | ISSUE-000049 | ISSUE-000049の完了条件 |
| ③ | shared Acceptance BoundaryへStudio / AIを接続 | ISSUE-000027 / 000046 | Product Gate C（`GATE_C_EVIDENCE_CONTRACT.md`） |

`Gate C`の名称衝突は上流で解消済みです。**Lane C Pre-ADR Gate C**（Handoff / address audit）と**Product Gate C**（第一Web TargetのProjection / Conformance）は別物で、Lane Cの契約ADRがISSUE-000027 / 000046のproduction実装を前提にする循環依存は作りません。

ISSUE-000042との担当境界は次のとおりです。

- ISSUE-000042 が所有: cross-view Reference、checker、Pressure test、A-set、operational checklist
- ISSUE-000042 が所有しない: Contract ADR readinessのReady / Not Ready判定（Pre-ADR Auditが正本）
- 新規3件が所有: ①②の到達条件と、second corpusという別Evidenceの取得

## 5. 受け皿が無かった上流結論

| 上流結論 | 出典 | 新しい受け皿 |
| --- | --- | --- |
| Contract ADR blocker A / B（actual runner green。fresh-process/source-uniqueness actualを包含、second realistic Handoff） | Pre-ADR Audit / readiness split §8 | ISSUE-000048 |
| ADR前に要求する7件のEvidenceと、前提にしないproduction完成6件 | readiness split §3 | ISSUE-000048 |
| production採用前の3条件（adapter / acceptance / fresh process） | readiness split §4 | ISSUE-000049 |
| second realistic TEI-original Handoffの`from/to/condition` losslessness | Handoff family / readiness split §8-B | ISSUE-000050 |
| C/U/D欠落は契約blockerではない、という判断 | readiness split §6 | ISSUE-000048の非目標 + 本書§6 |
| `ir.World.Business` embeddingのdefer判断 | readiness split §7 | ISSUE-000048（ADR起票時に記録） + 本書§6 |

Fresh-process feasibilityはADR前に必要なEvidenceである一方、current source-level implementationではGate A actual executionまで縮約している（2026-09-20更新: Gate A actual executionは取得済みでPASS。R = `e2856e6e568b6be966c6b8689ebea479ec184a4a`、判定の正本は[Pre-ADR Audit](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)§13）。このためISSUE-000048でmaterial blockerをAとは別の第三blockerとして重複管理しない。

## 6. reopen triggerの所有先

「今は入れない」判断は、再検討の条件と一緒に保持します。上流文書がtriggerを定義し、起動判断は下流が持ちます。

| defer中の対象 | trigger定義 | 起動時の受け皿 |
| --- | --- | --- |
| explicit `Subsystem` / `SubjectArea` | `SUBSYSTEM_CANONICAL_REOPEN_GATE_2026-09.md` | 新規ADR + 新規Issue（trigger成立時に起票） |
| generic mutation / effect contract（C/U/D field） | readiness split §6 | ISSUE-000048で判断、必要ならADR |
| `ir.World.Business` embedding | readiness split §7 | 同上 |
| generic ref-role taxonomy | ISSUE-000042 checklist 19 | ISSUE-000042 |

trigger未成立を「永久に不要」へ、trigger成立を「即採用」へ読み替えません。

## 7. 計画層へ反映すること

- `plan/issue/README.md`の主線は2026-09-04時点の記述です。Gate B三系統に加え、Lane B / Lane Cと本書で割り当てた3件を主線へ載せます。
- `MASTER_PLAN.md` / `MILESTONES.md`へは、①②③の順序依存（③はProduct Gate C後）だけを反映し、上流の個別結論は持ち込みません。
- Gate B必須Evidence三系統（ISSUE-000032 / 000025 / 000031）とLane Cのactual runner greenは**別のEvidence**です。同じrunnerが動くことで両方をgreenへ読み替えません。

## 8. 配置上の課題（提案）

`AGENTS.md`は「`plan/`直下は計画管理文書だけに限定する」と定めていますが、現在は上流pressure / audit文書が約85件、直下に並んでいます。計画管理文書（10件前後）が埋もれ、読み手導線が機能していません。

提案は次です。実行は本書の作業に含めず、別作業として判断します。

```text
plan/                 計画管理（Master Plan / Milestone / Gate / 本書）
  upstream/           pressure / audit / trace の上流工程文書
    business/         BUSINESS_* family
    lane-b/           LANE_B_* family
  design/ issue/ adr/ 既存のまま
```

移動時は相対リンクの張り直しが必要です（上流文書間の相互参照が多いため、リンク検査を伴う一括作業になります）。

## 9. 非主張範囲

- 上流文書の結論をここで再判定していません。原文の判断を移していません。
- pressure testやchecker実装の存在を、actual runner greenやGate通過として扱いません。
- ①②③の割り当ては担当の整理であり、どの段が今Readyかを主張するものではありません。
- 新規Issue 3件は起票であり、着手・完了を意味しません。

## 10. 関連文書

- [`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md) — RQ → PAS → Evidence trace
- [`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`](BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md) — 三段readinessの原典
- [`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md) — Contract ADR判定の正本
- [`SUBSYSTEM_CANONICAL_REOPEN_GATE_2026-09.md`](SUBSYSTEM_CANONICAL_REOPEN_GATE_2026-09.md) — Subsystem reopen trigger
- [`RQ025_RQ026_ACCEPTANCE_TRACE.md`](RQ025_RQ026_ACCEPTANCE_TRACE.md) — PAS-07 Evidence contract
- [`issue/000042-business-procedure-cross-view-reference.md`](issue/000042-business-procedure-cross-view-reference.md) — Lane C Reference / Evidenceの親
- [`README.md`](README.md) — 計画層の入口
