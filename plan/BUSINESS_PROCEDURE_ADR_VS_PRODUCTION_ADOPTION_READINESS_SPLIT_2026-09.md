# Business Procedure ADR vs Production Adoption Readiness Split — 2026-09

Status: **Lane C / readiness boundary clarification / no semantic candidate change**

## 1. 目的

Business Procedure / Handoffのcurrent minimum candidateは次まで絞れている。

```text
D4 + singular narrow O1 + S4-A + H1/R3
```

一方、これまでのPre-ADR文書では次の三段階が一部混在していた。

```text
Contract ADR readiness
Production adoption readiness
PAS-01 / Studio / AI Shared Acceptance end-to-end readiness
```

さらに`Gate C`という名称が二つの別概念に使われていた。

本書はsemantic candidateを変更せず、**何をいつEvidenceとして要求するか**だけを分離する。

## 2. Gate Cの名称衝突を解消する

### Lane C Pre-ADR Gate C

`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`内のlocal audit gate。

対象:

```text
Procedure-scoped Step address
Handoff authority
Handoff from/to/condition sufficiency
second realistic TEI-original Handoff
```

### Product Gate C

`GATE_C_EVIDENCE_CONTRACT.md`が定義する第一Web Target / Projection / ConformanceのProduct development gate。

`ISSUE-000027` AI Proposal / Acceptance Boundaryと`ISSUE-000046` Studio shared Acceptance authoring loopが待っている`Gate C Ready`はこちらである。

したがって:

```text
Lane C Pre-ADR Gate C
  != Product Gate C
```

であり、Lane CのHandoff contract ADRがISSUE-000027 / 000046のproduction implementation完了を前提にする循環依存は作らない。

## 3. Contract ADR readiness

目的は、次のcontract decisionを起こしてよいかを判断すること。

```text
Business Procedure minimum meaning
O1 protected-target representation
Procedure-scoped Step address
thin Handoff relation
separate normalized Business semantic plane
migration-adapter / candidate-only-loader architecture direction
freshness / authority-cutover invariants
```

ADR前に必要なのは**contractを反証できるEvidence**であり、最終production substrateの完成ではない。

最低限:

1. actual Repository mechanical green。
2. D4 + singular O1 / S4-A / H1/R3のactual losslessness / fail-closed tests。
3. multi-source normalization、source freshness、Conformance-base freshness、coordinated change-setのactual tests。
4. legacy + candidate simultaneous authority rejectのactual test。
5. candidate-only sourceをlegacy migration inputなしに別process相当で再構成できるfeasibility Evidence。
6. second realistic TEI-original Handoffで`from/to/condition`がlosslessであること。
7. derived Data / CRUD guardrailがactualに推測を抑制すること。

を要求する。

ただし次の**production完成**はContract ADRの前提にしない。

- generic/shared Acceptance service implementation完成
- Studio / AI authoring integration
- production multi-file writer完成
- production repository transaction API完成
- final hash / ETag / revision-token schema
- production migration command UX

これらが無いことはimplementation gapであり、Business schema field追加Evidenceではない。

## 4. Production adoption readiness

ADRでcontractを決めた後、実source authorityとして採用する前に次を要求する。

```text
Business-specific production authoring adapter
  -> workspace / Source Index discovery reuse
  -> strict production decode / normalize

Shared Acceptance / repository boundary
  -> complete stale envelope
  -> Validation / cross-view Conformance
  -> atomic candidate source apply
  -> atomic authority discovery cutover

fresh process
  -> candidate-only steady-state load
  -> legacy migration adapter unreachable
  -> same accepted Business semantic revision
  -> current World Conformance
```

失敗時にpartial source write / partial authority switchを残さないこともここで実証する。

これはContract ADR readinessより強い条件である。

## 5. PAS-01 / Studio / AI integration

`ISSUE-000027` / `ISSUE-000046`はProduct Gate C後にshared Acceptance BoundaryをStudio / AIへ接続するworkである。

Lane Cが今固定するのは次の**integration obligation**まで。

```text
Business authoring candidate
  -> AI専用経路を作らない
  -> Studio専用正本を作らない
  -> shared Acceptance principleへ接続可能であること
```

Product Gate C前にStudio / AI end-to-endを実装してBusiness Procedure ADRを証明する必要はない。

逆に、Business Procedure ADRができてもPAS-01をacceptedへ上げない。

## 6. C/U/D gapの扱い

Current TEI-original corpusにはqualifying C / D / non-state U attribution caseがない。

これは:

```text
Business Procedure schemaにCRUD fieldが必要
```

というEvidenceではない。

むしろcurrent ADRでは:

```text
unknown / insufficient operation authority
  -> unknownのまま保持
  -> Database write / Pattern kind / Lifecycleから推測しない
```

ことを要求する。

したがって**qualifying C/U/D caseの不存在そのものはBusiness Procedure Contract ADR blockerではない**。

将来generic mutation/effect contractを採用する場合のreopen triggerとして保持する。

## 7. ir.World.Businessの扱い

Current Evidenceではseparate normalized Business semantic planeでもcross-view Conformanceをfail closedできる方向が最も強い。

したがって:

```text
ir.World.Businessが必要だというEvidenceがない
```

ことはblockerではない。

これはcurrent decision:

```text
separate Business planeを選ぶ
ir.World embeddingをdeferする
```

を支持するnegative Evidenceである。

Concrete insufficiencyが出た場合だけplacementをreopenする。

## 8. Current blockersを二段に分ける

### Contract ADR blocker

Pre-ADR Auditのcurrent判定に合わせ、material blockerは二つに縮約する。

```text
A. actual Repository runner green未取得
   - source-uniqueness Pressureを含む
   - candidate-only fresh-process feasibility actualを含む

B. second realistic TEI-original Handoff未充足
```

Fresh-process feasibilityはADR前に必要な独立Evidence項目であるが、source-level実装済みであり、現在はGate A actual executionで同じrevision上に実行Evidenceを得る課題であった（2026-09-20更新: Gate A actual executionは取得済みでPASS。R = `e2856e6e568b6be966c6b8689ebea479ec184a4a`、判定の正本は[Pre-ADR Audit](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)§13）。このためmaterial blockerを別立てして三重管理しない。

### Production adoption blocker

Contract ADR blockersに加えて:

```text
C. production Business-specific adapter未実装
D. complete production stale envelope未実装
E. atomic source apply / authority cutover未実証
F. production fresh-process candidate-only reload未実証
G. partial failure safety未実証
```

### Blockerではないもの

```text
qualifying C/U/D caseがない
ir.World.Business必要性Evidenceがない
Studio / AI Shared Acceptance end-to-endがまだない
final revision-token shapeが未決定
```

これらをNot Ready理由へ混ぜない。

## 9. Current decisionへの影響

Semantic candidateは変えない。

```text
minimum candidate
  = D4 + singular narrow O1 + S4-A + H1/R3

placement candidate
  = separate normalized Business semantic plane

migration architecture
  = legacy-read migration adapter
    -> accepted one-time cutover
    -> candidate-only steady-state loader
```

Readinessだけを次へ分離する。

```text
Contract ADR
  = Not Ready
  reason = second realistic Handoff（actual Gate A greenは2026-09-20に充足）

Production adoption
  = Not Ready
  reason = Contract ADR blockers + production adapter / atomic apply / stale envelope / reload

PAS-01 / Studio / AI end-to-end
  = Product Gate C後の別work
```
