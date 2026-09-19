# Actor Context R3AF — Category Outcome / Freshness Authority Explanation Boundary

- Status: ISSUE-000053 / Phase 3 category-outcome explanation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AE_ROOT_CHANGE_CATEGORY_VALUE_OUTCOME_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3P_PURPOSE_BOUND_STOP_PROJECTION_OMISSION_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AEでは:

```text
root delta
  -> selected re-evaluation reason
  -> category recomputation
  -> changed / unchanged-after-reevaluation
```

を分離した。

R3AFではcategory outcomeとR3AD residualをOperational explanationへ戻す。同時に、projection meaningの再評価結果とAuthorization freshness authorityを別軸として保持する。

```text
unchanged-after-reevaluation
  != fresh
  != safe
  != reusable
  != Authorization freshness resolved

changed-after-reevaluation
  != stale
  != error
  != retry-required
```

## 2. explanation shape

test-only explanationは少なくとも次を分ける。

```text
shared freshness stop reason
resolution need class

reevaluation authority status
reuse authority status

category outcome identity
unselected changes
residual root deltas
```

category outcome identityは:

- category
- changed / unchanged status
- before / after fingerprint
- exact reason identity

を保持する。

fingerprintはAgent resolution-inspectionではinline表示できるが、Human orientationではinline省略できる。

## 3. freshness authorityは別軸

current R3AA scenarioでは:

```text
reevaluation authority = unresolved
reuse authority = unresolved
resolution need = authorization-evaluation-freshness-authority
```

である。

machine-reasonsが `unchanged-after-reevaluation` でも、このauthority stateは変えない。

```text
projection meaning unchanged
  != prior Observation reuse permitted
```

である。

## 4. changed outcomeもresponse actionを選ばない

category meaningが実際に変わっても:

- retry
- auto reauthorize
- rebuild
- human approval
- handoff owner
- protected operation continuation

を自動選択しない。

```text
meaning changed
  != response action selected
```

を維持する。

## 5. residual / unselectedを吸収しない

R3ADのResidualRootDeltaとR3T UnselectedChangesをcategory outcomeへ合成しない。

prior Authorization Observation revisionのようなno-binding residualは:

```text
residual root delta
  != category unchanged
  != category changed
  != current runtime outcome
```

である。

selected categoryがunchangedでもresidualが残る場合:

```text
selected meaning unchanged
  != all observed changes resolved
```

とする。

## 6. Human / Agent

双方でselectedされるcategory outcome identityは同一である。

Human:
- shared category/status/reason identityを保持
- raw fingerprintはinline省略可
- Agent-only categoryを追加しない

Agent:
- fingerprint detail
- Agent-only known-authority-facts outcome
- residual detail

をより厚く保持できる。

```text
more detail
  != stronger authority
```

である。

## 7. global verdictを作らない

changed / unchanged / residualが同時に存在しても、全体を:

- fresh
- stale
- safe
- degraded
- reusable

の一値へ畳まない。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3af_category_outcome_explanation_pressure_test.go`

Pressureするcase:

1. requirement Reference change + unchanged machine-reasons -> shared unchanged outcome + freshness authority unresolved
2. Humanはfingerprintをinline省略し、Agentは保持してもoutcome identityは同じ
3. actual machine-reasons change -> changed outcome + freshness authority unresolved
4. prior Observation no-binding residual -> category outcomeなし / residual保持
5. Agent-only known-authority-facts outcomeをHumanへ強制しない
6. UnselectedChanges / ResidualRootDeltasをresolvedへ丸めない
7. Fresh / Safe / Reusable / Retry / NextAction / HandoffOwner等を追加しない

## 9. current conclusion

```text
category outcome
  = projection meaning comparison

freshness authority status
  = separate Authorization authority question

unchanged category meaning
  != freshness authority resolved

changed category meaning
  != next action selected
```

production UI contract、Agent protocol、notification transport、ticket routingは追加しない。

## 10. 次

次段では、このexplanationをread-only Agentがmachine-readableに消費するとき:

```text
understand unresolved state
  != autonomously choose remediation
```

を維持し、resolution needをnext actionへ変換しないことをPressureする。
