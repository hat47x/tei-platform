# Actor Context R3AI — Read-only Agent Consumption / Remediation Non-selection Boundary

- Status: ISSUE-000053 / Phase 3 read-only Agent reasoning pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AH_CROSS_AUDIENCE_OUTCOME_DRILLDOWN_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AG_PURPOSE_BOUND_OUTCOME_EXPLANATION_MATERIALIZATION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AGではAgent / resolution-inspection向けにexact outcome detailをmaterializeでき、R3AHではそこからexisting dependency / provenance root / current locatorへdrill-downできるようにした。

R3AIでは、それらをread-only Agent reasoning inputとして消費しても:

```text
understand unresolved state
  != select remediation

resolution need
  != next action
```

を維持する。

これはPhase 4 side-effect Actionへ入る前のread-only境界である。

## 2. Agentがread-onlyに保持できるもの

test-only reasoning viewはR3AG / R3AHから少なくとも次を保持できる。

```text
freshness stop reason
resolution need classes
reevaluation authority status
reuse authority status

selected category outcome identity
exact before / after fingerprints
exact re-evaluation reason identity

unselected changes
residual root deltas
dependency / provenance traces
authority limits
```

これらは理解・説明・照合の入力であり、Commandではない。

## 3. resolution needはdiagnostic class

current freshness stopの:

```text
resolution need
  = authorization-evaluation-freshness-authority
```

は「何が未解決か」を示す。

ここからAgentが自動的に:

- Authorization providerを呼ぶ
- reauthorizeする
- retryする
- policyを変更する
- human approvalを依頼する
- ownerへhandoffする
- prior Observationをreuseする

と決めてはならない。

```text
resolution need class
  != remediation operation
  != owner
  != tool
  != workflow step
```

である。

## 4. explicit freshness directiveもAgent実行権限ではない

R3Yで明示的に:

```text
reevaluation authority = explicit-reevaluation-required
```

が得られた場合、R3AIはそのauthority factを保持できる。

しかし:

```text
reevaluation required
  != this Agent may invoke reevaluation
  != schedule authorization call
  != retry protected operation
```

である。

同様に:

```text
reuse authority = explicit-reuse-permitted
  != protected operation execution authorized
  != current Authorization result allowed
```

とする。

freshness authorityとoperation execution authorityを統合しない。

## 5. category outcomeをresponse policyへ変換しない

```text
unchanged-after-reevaluation
  != continue
  != reuse
  != no-op authorized
  != fresh

changed-after-reevaluation
  != retry
  != rebuild
  != escalate
  != unsafe
```

である。

category outcomeは「再計算後の意味が変わったか」を示すだけで、response policyではない。

## 6. residual / unselected / traceを消さない

R3AG Agent materializationの:

- UnselectedChanges
- ResidualRootDeltas

とR3AHのprovenance traceをread-only viewでも保持する。

prior Authorization Observationのno-binding residualは:

```text
prior Observation residual
  != current runtime decision
  != current Authorization result
```

である。

traceが詳しくてもauthorityは増えない。

## 7. repeated consumptionでauthorityは増えない

同一materialization / traceを複数回消費しても:

```text
repeated observation
  != corroboration quorum
  != stronger authority
  != confidence override
```

とする。

入力順序の違いもsemantic identityを変えない。

## 8. read-only shapeにActionを入れない

R3AI reasoning viewは次を持たない。

- ToolCall
- InvocationPlan
- Command
- SuggestedAction
- NextAction
- Remediation
- Retry
- Reauthorize
- Rebuild
- Escalate
- ApprovalRequest
- HandoffOwner
- CanExecute
- Authorized / Allowed
- Continue
- Safe / Fresh / Reusable
- Confidence / Severity / Priority

Agentが外部へ情報を要求する必要がある場合も、そのrequest設計は次段で「情報要求」と「行為要求」を分離してPressureする。

## 9. source authorityを変えない

R3AIはderived read-only reasoning viewである。

```text
Agent consumed fact
  != Accepted Decision
  != Business authority
  != Authorization authority
  != Canonical source
```

R3AH traceのcurrent locatorを読めることも、そのsourceを新しいauthorityへ昇格させない。

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ai_read_only_agent_consumption_pressure_test.go`

Pressureするcase:

1. Agent materialization + R3AH traceをsemantic identityを失わずread-only viewへ保持する
2. resolution need classを保持してもNextAction / Remediation / HandoffOwnerを生成しない
3. unchanged outcomeからreuse / continue / freshを推論しない
4. changed outcomeからretry / rebuild / escalateを推論しない
5. prior Observation no-binding residualをcurrent runtime decisionへ昇格させない
6. explicit reevaluation-requiredを保持してもToolCall / schedule / invocation authorityを生成しない
7. explicit reuse-permittedを保持してもCanExecute / Authorized / Allowedを生成しない
8. repeated consumption / input orderでnormalized identityを変えない
9. source traceの詳細度をauthority strengthへ変換しない

## 11. current conclusion

R3AIで固定するのは:

```text
purpose-bound materialization
  + exact provenance trace
  -> read-only Agent understanding

read-only understanding
  != remediation selection
  != invocation authority
  != side effect
```

である。

production Agent protocol、planner、tool router、approval workflow、ticket routing、Command executionは追加しない。

## 12. 次

次段ではunresolved stateについて外部へ問い合わせる必要がある場合でも:

```text
request for missing authority information
  != request to execute remediation
```

を維持する。

特にhandoff recipientが未解決でもownerを推測せず、情報要求のscope / recipient-status / evidence needだけを表現できるかをPressureする。
