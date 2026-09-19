# Actor Context R3B — Authorization Prerequisite / Fail-Closed Boundary

- Status: ISSUE-000053 / Phase 3 follow-up pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3A_PRINCIPAL_AUTHORIZATION_BOUNDARY_2026-09.md`
- Refined by: `plan/ACTOR_CONTEXT_R3C_AUTHORIZATION_REQUIREMENT_AUTHORITY_2026-09.md`
- Related: `plan/LANE_B_AUTHORIZATION_TARGET_AUTHORITY_AUDIT_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Bは、read-only Operational Projectionが未解決のAuthorization前提をAgent推論で埋めず、actual invocationより前でfail closedできるかをPressureする。

重要な順序は次である。

```text
Authorization requirement?
  unresolved -> stop here

  required
    -> principal?
    -> action/resource target?
    -> Authorization evaluation?
    -> downstream runtime prerequisites

  explicitly not-required
    -> Authorization-specific prerequisitesを要求しない
    -> downstream runtime prerequisitesへ
```

この段階列はtest-only pressureであり、production enum / workflow engineではない。

## 2. BP-02ではrequirement自体が未解決

BP-02から静的に解決できるもの:

- Business Action
- `order_entry.customer_reference`
- datasource `master`
- Application access `read`
- Contract `master_data.customer`
- Query `find_by_id`

一方、current Business ProcedureではBP-02は:

```text
realization = application-use
applicationPatterns = [order_entry.customer_reference]
```

であり、`security.authorize` requirementを明示していない。

ここから:

```text
security.authorize absent
  != Authorization not required

Application access = read
  != Authorization required
  != Authorization not required
```

とする。

したがってcurrent BP-02の最初のblocking reasonは:

```text
authorization-requirement-unresolved
```

である。

## 3. BP-06は対照例

BP-06は:

```text
realization = security-boundary
capabilities = [security.authorize]
```

を明示する。

このBusiness Step自身については、Authorization requirementを `required` と扱える。

さらにtest-only protected-target overlayはBP-06のtargetとして `order_entry.submit_order` を明示する。

ただし:

```text
BP-06 requirement / target
  != BP-02 requirement / target
```

であり、同一Procedure内だからといって流用しない。

## 4. principalはrequirementに従属する

R3Aは、explicit authenticated subjectからprincipalを安全に解決する方法をPressureした。

しかし:

```text
principal can be resolved
  != principal is required for every Action
```

である。

Authorization requirementが `required` と確定した場合にだけ、Authorization用principalをprerequisiteとして要求する。

requirementが未解決なら、まずrequirement authorityを解決する。

将来explicit `not-required` authorityが成立した場合は、Authorization用principal / target / evaluationをその経路の必須条件にはしない。

## 5. target/action/evaluationもrequirementに従属する

R3Aで確認した:

- `read -> browse` を生成しない
- Pattern identityをresource authorityへしない
- policyだけでtargetをbackfillしない

という境界は維持する。

さらにR3Bでは、requirement自体が未解決の段階で:

```text
authorization-action-resource-unresolved
authorization-not-evaluated
```

を**現在のblocking reason**へ昇格させない。

それらはruntime contextとして保持できるが、Authorizationがrequiredと判明した後にprerequisiteとして評価する。

## 6. policyはrequirement authorityではない

Legacy policyにresource ruleが:

- あること
- ないこと

のどちらもBusiness/Application側のAuthorization requirementをauthorしない。

```text
policy rule present
  != Authorization required

policy rule absent
  != Authorization not required
```

policyはaccepted target/actionがある場合のrealization / corroboration Evidenceである。

## 7. test-only projection

`InvocationBoundaryProbe`では:

- Authorization requirement status
- principal resolved
- target resolved
- evaluated
- invocation attempted
- query invocation attempted
- Observation received
- completion unresolved
- current blocking reasons
- deferred runtime gaps
- authority limits

を分ける。

一つの `canExecute` / `ready` / confidenceには畳まない。

current BP-02では:

```text
authorization requirement     = unresolved
principal resolved             = true  // explicit Authentication Observationを与えたpressure case
current blocker                = authorization-requirement-unresolved
authorization invocation       = not attempted
Contract Query invocation      = not attempted
Observation                    = none
Business completion            = unresolved
```

principalが解決済みでも、それがAuthorization applicabilityを決めないことが重要である。

## 8. stop / handoff

stop reasonから:

- handoff owner
- authoring owner
- production field
- policy role

を推論しない。

R3Bは「現在のauthorityではここから先へ進まない」ことだけを示す。

## 9. Evidence

Design / Source Evidence。

actual `go test` PASS、Authorization invocation、Query invocation、Observationはまだ主張しない。
