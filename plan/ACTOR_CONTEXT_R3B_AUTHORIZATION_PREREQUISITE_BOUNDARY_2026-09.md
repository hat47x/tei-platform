# Actor Context R3B — Authorization Prerequisite / Fail-Closed Boundary

- Status: ISSUE-000053 / Phase 3 follow-up pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3A_PRINCIPAL_AUTHORIZATION_BOUNDARY_2026-09.md`
- Related: `plan/LANE_B_AUTHORIZATION_TARGET_AUTHORITY_AUDIT_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Aでは、explicit Authentication Observationからruntime principalを解決できても、BP-02のAuthorization action/resource authorityは解決しないことを確認した。

R3Bでは、この不足をAgentが推論で埋めず、actual invocationより前でfail closedできるかをPressureする。

```text
authenticated subject
  + understood read route
  + missing explicit Authorization target
  -> stop before security.authorize
  -> stop before Contract Query
```

R3Bはactual Authorization / Query invocationを実装しない。

## 2. current BP-02に存在するもの

BP-02についてcurrent Referenceから静的に解決できるもの:

- Business Action
- `order_entry.customer_reference`
- datasource `master`
- Application access `read`
- Contract `master_data.customer`
- Query `find_by_id`
- input semantic mapping
- result cardinality

R3Aを通じてruntimeで追加できるもの:

- explicit authenticated `subjectId`

一方、current Business/Application authorityにはBP-02用の明示的な:

- Authorization action
- Authorization resource / protected target

がない。

したがって `read -> browse`、Pattern identity -> policy resourceという補完は行わない。

## 3. BP-06 protected-target pressureとの関係

既存のtest-only protected-target overlayには、BP-06について:

```text
BP-06
  supporting capability = security.authorize
  protected target      = order_entry.submit_order
```

が明示されている。

これは、

```text
supporting semantic capability
  != protected application target
  != Application Function realization
```

をPressureする既存Evidenceである。

しかし同overlayにはBP-02 targetがない。

したがってR3Bでは:

```text
BP-06 explicit target
  != BP-02 target

adjacent / same Procedure target
  != reusable authorization authority
```

として扱う。

## 4. Legacy Authorization policyはstop reasonを解消しない

current policyにはBP-02 `order_entry.customer_reference` ruleがない。

さらにtest-onlyで:

```text
browse / order_entry.customer_reference / allow
```

を追加しても、それだけではBusiness/Application target authorityにならない。

policyは明示targetをcorroborateできても、missing targetをauthor/backfillしない。

## 5. R3B test-only projection

`InvocationBoundaryProbe`は結果を一つの `canExecute` に畳まない。

保持するのは:

- principal resolved?
- Authorization target resolved?
- Authorization evaluated?
- Authorization invocation attempted?
- Contract Query invocation attempted?
- Observation received?
- completion unresolved?
- stop reasons
- authority limits

current BP-02では:

```text
principal resolved                  = true
authorization target resolved       = false
authorization evaluated             = false
authorization invocation attempted  = false
contract query invocation attempted = false
observation received                = false
completion unresolved               = true
```

とする。

## 6. stop / handoffの意味

R3Bで確定するのは「ここでは進めない」という境界までである。

```text
stop reason
  = authorization-action-resource-unresolved
```

から、

- どの人へhandoffするか
- どのteamがtargetをauthorするか
- targetをBusiness ProcedureとApplicationのどちらへ永続化するか

を推論しない。

既存Lane B auditが示す候補を比較し、accepted authorityが得られた時点で次段へ進む。

## 7. Source Index追随

R3 sourceがSource Index subject resolutionへ移行したため、R3A/R3Bも同じread-route derivationを利用する。

```text
semantic subject identity
  -> Source Index definition
  -> current locator

runtime Authentication Observation
  != static Source Index definition
```

principal/runtime contextをstatic freshness fingerprintへ混ぜない。

## 8. Evidence高さ

この段階はDesign / Source Evidenceである。

actual `security.authorize` invocation、Contract Query invocation、Observation、Business continuationはまだ存在しない。

actual runnerでtest PASSを取得するまではExecutable Evidenceとも呼ばない。

## 9. 次

次段では、BP-02にAuthorizationが本当に必要かを含め、protected-target/action authorityをどのaccepted sourceへ置くべきかを既存Lane Bのpressureと統合して比較する。

authorityが成立するまではinvokeしない。
