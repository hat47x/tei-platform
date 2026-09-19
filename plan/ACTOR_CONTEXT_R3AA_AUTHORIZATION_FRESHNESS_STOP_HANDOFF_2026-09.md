# Actor Context R3AA — Authorization Freshness Authority Operational Stop / Handoff Boundary

- Status: ISSUE-000053 / Phase 3 freshness-stop projection pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3Z_AUTHORIZATION_FRESHNESS_PRECEDENCE_EVIDENCE_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3M_OPERATIONAL_STOP_EXPLANATION_HANDOFF_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3N_STAGE_AWARE_OPERATIONAL_STOP_TAXONOMY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3O_STOP_PROJECTION_CORE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3P_PURPOSE_BOUND_STOP_PROJECTION_OMISSION_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3X〜R3Zで、same Authorization tupleでもevaluation freshness authorityが未解決になり得ることを固定した。

current unresolved state:

```text
Authorization tuple = compatible
prior Observation = exists
reevaluation authority = unresolved
reuse authority = unresolved
```

は、deniedでもnot-evaluatedでもObservation mismatchでもない。

R3AAではこれを既存R3M〜R3PのOperational stop / handoff projectionへ接続する。

## 2. 新しいstop meaning

test-only taxonomyに次を追加する。

```text
stage = authorization-evaluation-freshness-authority
class = evaluation-freshness-authority-unresolved
reason = authorization-evaluation-freshness-authority-unresolved
resolution need = authorization-evaluation-freshness-authority
```

これはproduction API vocabulary採用ではない。

## 3. deniedへ丸めない

prior Observationがallowed / deniedであっても、そのObservationをcurrent decision evidenceとして利用できるfreshness authorityが未解決なら:

```text
freshness authority unresolved
  != current denied
  != current allowed
```

である。

R3AA projectionはcurrent runtime outcomeを捏造しない。

## 4. retry / approvalへ変換しない

freshness authority unresolvedから:

- retry-required
- reauthorize-now
- human-approval-required
- handoff owner
- security team

を自動生成しない。

```text
resolution requested
  != resolution recipient known
  != resolution method selected
```

をR3M同様に維持する。

## 5. earlier blockerを優先する

requirement applicabilityがunresolved、tuple mismatch、principal / action / resource authority gap等がある場合は、そのearlier / more specific stopを維持する。

```text
requirement transition unresolved
  -> requirement authority stop
  != freshness stop

tuple mismatch
  -> evaluation Observation mismatch
  != freshness stop
```

freshness stopは、same tuple前提が成立し、freshness authorityだけが未解決の場合に限る。

## 6. R3O / R3Pへ再利用する

R3AAは別packetを作らない。

R3M explanation + R3N taxonomyからR3O common coreを作り、R3Pの:

- Human orientation
- Agent resolution-inspection

へ通す。

Human / Agentでdetail disclosureは異なっても:

```text
machine reason
resolution need
source refs
```

は同じauthority-preserving coreへ戻る。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3aa_authorization_freshness_stop_handoff_pressure_test.go`

Pressureするcase:

1. R3Y both unresolved + tuple match -> freshness authority stop
2. R3M explanationがresolution needを正確に保持
3. R3N taxonomyがdenied / not-evaluated / reuse-unresolvedと区別
4. R3O common coreへ損失なく接続
5. Human / Agent bounded projectionが同じmachine reasonを保持
6. handoff recipientはunresolved
7. current runtime outcomeをinventしない
8. requirement transition / tuple mismatchをfreshness stopへ昇格させない
9. Retry / Approval / CacheValid / ShouldReevaluate等を追加しない

## 8. current conclusion

R3AAで固定するのは:

```text
freshness authority unresolved
  -> explainable stop
  -> explicit resolution need
  -> unresolved handoff recipient

  != denied
  != retry directive
  != approval request
  != hidden cache decision
```

である。

production Agent protocol、notification transport、ticket routing、automatic reevaluation schedulerは追加しない。

## 9. 次

次段ではfreshness stopのdrill-downが:

- R3X context identity
- R3Y freshness authority axes
- R3Z Evidence Gate

へ同じprovenance chainで戻れるかをPressureする。

Human / Agentの入口差から別のtruthを作らない。
