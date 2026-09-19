# Actor Context R3AH — Cross-audience Outcome Drill-down Chain

- Status: ISSUE-000053 / Phase 3 cross-audience outcome drill-down pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AG_PURPOSE_BOUND_OUTCOME_EXPLANATION_MATERIALIZATION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3V_CROSS_AUDIENCE_REASON_IDENTITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AD_ROOT_CHANGE_PROJECTION_IMPACT_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AB_CROSS_AUDIENCE_FRESHNESS_DRILLDOWN_PROVENANCE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AGではHuman / Agentで表示密度を変えても、shared category outcomeは同じunderlying identityへ戻るようにした。

R3AHではdrill-downを実際に一段ずつ接続する。

```text
R3AG summary outcome identity
  -> exact R3AF category outcome
  -> exact R3T re-evaluation reason
  -> existing R3R/R3S dependency key
  -> explicit R3AD root binding
  -> R3AB provenance root
  -> current locator
```

このchainはHuman / Agentのpresentationから独立である。

## 2. summary handleはauthorityではない

R3AGの`OutcomeIdentity`はexact outcomeへ戻るためのderived handleである。

```text
summary outcome identity
  != Canonical source identity
  != authority
  != dependency key
```

drill-down開始時に、summary handleとexact R3AF outcomeから再導出したidentityが一致しなければfail closedする。

## 3. exact reasonからexisting dependencyへ戻る

R3AHはR3T reasonの:

- category
- change class
- dependency key
- dependency reason

をそのまま保持する。

dependency keyはR3ADで既に採用したR3R/R3S keyを使い、R3AH用のparallel keyを作らない。

```text
reason identity
  -> reason.DependencyKey

  != nearest source
  != role-based guessed dependency
```

である。

## 4. dependencyからrootはexplicit bindingだけ

R3ADの`RootAddress -> DependencyKey` bindingを逆向きに辿る。

同じdependencyへ複数rootが明示bindingされる場合は全て保持し、winnerを選ばない。

bindingが存在しない場合:

```text
dependency root provenance unresolved
  -> fail closed
```

とし、locator文字列やroleからrootを推測しない。

## 5. root identityとcurrent locator

R3AB / R3ACどおり:

```text
root identity
  != locator
```

である。

R3AH traceは:

- root address
- root semantic identity
- role
- claim kind
- current locator

を保持する。

locator-only move後もroot identityは同じままcurrent locatorを更新できる。

## 6. Human / Agent invariance

Human orientationはfingerprint / exact reasonをinlineに持たなくても、summary outcome identityからdrill-downできる。

Agent resolution-inspectionはexact detailをinlineに持つが、同じshared categoryについて得られるtraceはHumanと同一でなければならない。

```text
different presentation
  -> same shared outcome identity
  -> same reason identity
  -> same dependency key
  -> same provenance root
```

Agent-only `known-authority-facts` はAgentだけ追加traceを持てる。Humanへ強制開示しない。

## 7. residualは別chain

R3AD ResidualRootDeltaにはprojection dependency bindingがない場合がある。

prior Authorization Observation revisionのようなresidualを:

```text
residual root
  -> fabricated dependency
```

へ変換しない。

residual drill-downは:

```text
residual reason
  -> root address / role
  -> current locator
```

までを別chainとして保持する。

```text
residual trace
  != selected category reason trace
```

である。

## 8. fail closed

次のいずれかはfail closedする。

- summary handleとexact outcome identityが不一致
- selected reasonのdependency keyが空
- dependency keyへのexplicit root bindingがない
- binding先rootがcurrent root setに存在しない
- current locatorが取得できない

R3AHはmissing linkをLegacy policy、Source Index類似、role名から補完しない。

## 9. 禁止する短絡

drill-down traceは説明であり、次を生成しない。

- Fresh / Stale
- Safe / Reusable
- ShouldReevaluate
- Retry / Reauthorize / Rebuild
- ApprovalRequest
- NextAction / HandoffOwner
- Authorized / Allowed / CanExecute
- Confidence / Priority / Winner

```text
traceability
  != authority
  != remediation
```

である。

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ah_cross_audience_outcome_drilldown_pressure_test.go`

Pressureするcase:

1. Human machine-reasons summaryからexact outcome / reason / dependency / root / locatorへ辿れる
2. Agentのshared machine-reasons traceがHumanと一致する
3. Agent-only known-authority-facts traceをHumanへ強制しない
4. tampered outcome handleはfail closed
5. missing explicit dependency-root bindingはfail closed
6. binding先root欠落はfail closed
7. prior Observation no-binding residualはdependencyを捏造せずroot / locatorへ辿れる
8. repeated / input-order variationでnormalized trace identityを変えない
9. action / authority shortcutを追加しない

## 11. current conclusion

R3AHで固定するのは:

```text
presentation handle
  -> exact outcome
  -> exact reason
  -> existing dependency
  -> explicit root binding
  -> provenance root / current locator

cross-audience presentation difference
  != provenance chain difference

residual provenance
  != selected dependency provenance
```

である。

production provenance service、global graph、UI routing、Agent invocation protocolは追加しない。

## 12. 次

次段ではAgent resolution-inspection materializationとR3AH traceをread-only reasoning inputとして消費するとき:

```text
understand unresolved state
  != select remediation
```

を維持できるかをPressureする。
