# Actor Context R3M — Operational Stop Explanation / Handoff Boundary

- Status: ISSUE-000053 / Phase 3 operational explanation pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3L_AUTHORIZATION_ACTION_LOCUS_EVIDENCE_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3B_AUTHORIZATION_PREREQUISITE_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3I_AUTHORIZATION_TARGET_EVALUATION_CADENCE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3J_AUTHORIZATION_ACTION_AUTHORITY_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3A〜R3Lで、read-only AI Business Actorはmissing authorityを推測で埋めず、最も早い未解決境界でfail closedするようPressureしてきた。

次の課題は、停止したこと自体をどう外部へ説明するかである。

AI Business Actorは単に:

```text
cannot proceed
```

と返すだけでは、Human Reviewや別agentが何を確認すべきか分からない。

一方で、停止理由から:

- handoff owner
- approval authority
- missing action value
- missing policy
- next executable Business Action

を推測してはならない。

R3Mでは、**stop stateをauthority-preservingなexplanation / handoff request projectionへ変換する境界**をPressureする。

## 2. explanationはderived projection

R3M explanationは正本ではない。

```text
accepted source / runtime observation
  -> stop state
  -> explanation projection
```

であり:

```text
explanation projection
  -X-> author Business meaning
  -X-> author Authorization action
  -X-> select handoff owner
  -X-> approve continuation
```

とする。

projectionを削除しても、元のsource / runtime stateから再構成できることを前提にする。

## 3. current BP-06 stop example

current BP-06では:

```text
requirement = required
requirement source = business-procedure-security-boundary

resource = order_entry.submit_order
resource source = reference-pressure-overlay:BP-06

action = unresolved
```

である。

したがってcurrent stopは:

```text
stop stage = authorization-action-authority
blocking reason = authorization-action-authority-unresolved
```

と説明できる。

しかし:

```text
likely action = execute
owner = security team
ask ORDER_EDITOR
continue to BP-07 after approval
```

などは生成しない。

Business wording / Pattern kind / Legacy policyはcorroborating hintsとしてdrill-down可能でも、missing authorityの値やownerを埋める材料にしない。

## 4. explanation projectionの最小内容

test-only R3M projectionでは少なくとも次を保持する。

```text
Business address
stop stage
blocking reasons
known authority facts
resolution need class
source refs
downstream operations not attempted
handoff recipient status
authority limits
```

### known authority facts

例:

- Authorization requirement = required
- protected resource = order_entry.submit_order

値にはprovenanceを付ける。

### resolution need class

例:

```text
authorization-action-authority
authorization-evaluation
```

authority gapではauthority classを、runtime evaluation未実施ではruntime Evidence needを表す。

これはmissing valueやresolution ownerそのものではない。

```text
resolution need class known
  != missing value known
  != resolution owner known
```

を維持する。

### downstream operations not attempted

current stopより後段の:

- Authorization invocation
- protected Application command
- Business completion

を成功・失敗いずれにも偽装しない。

## 5. handoff requestとhandoff ownerを分離する

R3Mでは、未解決authorityに対して「外部解決が必要」というrequestを作る余地は認める。

ただし:

```text
resolution requested
  != handoff recipient known
```

である。

current sourceに明示的なowner / responsibility mappingがない限り:

```text
handoff recipient status = unresolved
```

とする。

停止理由から:

- security team
- application owner
- business owner
- human operator
- administrator

のいずれも自動選択しない。

Business Procedure actor label `sales operator` もruntime / governance handoff ownerへ転用しない。

## 6. blocker vocabularyをpreserveする

explanationのためにblockerを自然言語へ要約しても、machine-readable reasonは失わない。

例えば:

```text
authorization-action-authority-unresolved
```

を:

```text
authorization failed
```

へ丸めない。

前者は:

- action valueがないのか
- action provenanceがないのか
- evaluationがdeniedなのか

を区別する。

R3Mではsource blocker vocabularyを保持し、表示文はderived detailとしてのみ扱う。

## 7. earliest blockerを別blockerへ昇格させない

R3Bからの原則を維持する。

例:

```text
requirement unresolved
  -> explain requirement authority gap
  != principal gap as current blocker
  != target gap as current blocker
```

同様に:

```text
action authority unresolved
  -> explain action authority gap
  != authorization-not-evaluated as current blocker
```

後段の未実行状態は `downstream not attempted` として説明できるが、現在のblocking reasonへ混ぜない。

## 8. Evidence / drill-down

R3M projectionは、known factとstop reasonを確認できるdrill-down先を保持する。

current BP-06では少なくとも:

- Business Procedure source
- protected target overlay
- R3J action-authority gate source

へ戻れる。

Legacy policyはcorroborationとして参照可能だが、action authority sourceとして表示しない。

```text
source ref role
  = authority / corroboration / runtime observation
```

を混同しない。

## 9. test-only source

- `tools/reference-business-procedure-check/actor_context_r3m_operational_stop_explanation_pressure_test.go`

Pressureする不変条件:

- action authority unresolvedをexact blockerとして保持する
- known requirement / resourceへprovenanceを保持する
- missing action valueを推測しない
- Legacy policyの `execute` をexplanation上のmissing valueへ昇格させない
- Business actor labelからhandoff ownerを生成しない
- downstream Authorization / command / completionをattempted / succeededへしない
- earliest blockerとdeferred downstream stateを混ぜない
- Approved / Safe / CanExecute / HandoffOwner / SuggestedAction等のshortcut fieldを作らない

## 10. current conclusion

R3Mで固定するのは:

```text
fail closed
  != opaque failure

explain resolution need class
  != infer missing authority value
  != misclassify runtime Evidence need as authority

request resolution
  != select resolution owner

show downstream not attempted
  != promote downstream gap to current blocker
```

である。

production agent protocol、ticket routing、human task assignment、notification transportは追加しない。

## 11. 次

次段では、同じexplanation projectionを:

- requirement unresolved
- action authority unresolved
- evaluation absent
- evaluation denied
- Observation mismatch

へ適用し、**stop taxonomyが異なる状態を一つの「権限エラー」へ潰さないこと**をPressureする。

その後に初めて、human / agent operational UXへ必要な最小projection shapeを検討できる。
