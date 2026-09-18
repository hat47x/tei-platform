# Actor Context R3A — Principal / Authorization Target Boundary

- Status: ISSUE-000053 / Phase 3 follow-up pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3_READ_ONLY_BUSINESS_ACTOR_CHALLENGE_2026-09.md`
- Related: ADR-0019 / `docs/conformance/legacy-authorization-v1.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3では、BP-02のBusiness Actionからread-only Contract Queryまでのstatic routeを解決した。

次に必要なのは、routeを理解したAgentがruntime principalとAuthorizationを捏造せず、既存TEIのAuthentication / Authorization境界へ接続できるかをPressureすることである。

R3Aではactual Authorization invocationもread query invocationもまだ行わない。

## 2. 既存TEIにすでにある境界

Reference Workspaceには次のSemantic Capabilityが存在する。

```text
security.authenticate
  -> authenticated security.authenticated_subject
  -> denied

security.authorize
  input subject
  input action
  input resource
  -> allowed
  -> denied
```

ADR-0019は、Application利用者のAuthentication / AuthorizationとPlugin / Runtime Permissionを別責務としている。

したがってR3Aのために新しいPrincipal schema、Role model、Permission modelを作らない。

## 3. Business actor labelはruntime principalではない

Business Procedure Referenceには、

```text
actor.status = candidate
actor.label = sales operator
```

がある。

これはProcedureを説明するBusiness actor候補であり、実行時のidentityではない。

```text
Business actor label
  != authenticated subject
  != Agent principal
```

R3Aでは、明示された `security.authenticate / authenticated` Observationにnon-empty `subjectId` がある場合だけprincipal boundaryを解決する。

Authenticationがない場合、deniedの場合、subjectIdが空の場合は `actor-principal-unresolved` を保持する。

## 4. authenticated subjectだけではAuthorizationにならない

authenticated subjectを得ても、次はまだ別である。

```text
authenticated subject
  != authorization target
  != authorization evaluated
  != allowed
```

BP-02のcurrent routeには、

- Pattern = `order_entry.customer_reference`
- Application datasource access = `read`

がある。

しかしcurrent Business Procedure / Application Referenceには、BP-02についてAuthorizationへ渡す明示的な

- action
- resource

の組を定義していない。

したがって、

```text
Application access = read
  != Authorization action = browse
```

として扱う。

語彙の似ている `read` / `browse` をAgentが補完しない。

## 5. Authorization policyはmissing targetの正本ではない

current Legacy Authorization fixtureは、

- `browse / order_entry.recent_orders`
- `edit / order_entry.order_record`
- `execute / order_entry.submit_order`

を扱う。

`order_entry.customer_reference`はcurrent policy targetではない。

さらに重要なのは、将来policy側へ同名resource ruleが追加されたとしても、それだけを根拠にBP-02のBusiness/Application Authorization targetを生成しないことである。

既存のAuthorization target authority pressureと同じく、

```text
explicit Business/Application target
  -> Authorization policy may corroborate

Authorization policy rule only
  != missing Business/Application target authority
```

とする。

## 6. test-only shape

production schemaではない。

```text
PrincipalAuthorizationProbe

  procedure actor
    status
    label

  runtime subject
    subjectId
    principalSource
    principalResolved

  authorization target
    action
    resource
    targetResolved

  authorization
    evaluated

  runtime gaps
  authority limits
```

R3A current challengeでは、authenticated subjectが得られた場合でも、

```text
authorizationAction = empty
authorizationResource = empty
authorizationTargetResolved = false
authorizationEvaluated = false
```

を維持する。

## 7. Pressure scenarios

### P1 — Authentication Observationなし

- candidate actor labelをsubjectIdへ変換しない
- principal unresolvedを保持

### P2 — denied Authentication Outcome

- payloadらしきsubject文字列があってもprincipalへ採用しない

### P3 — explicit authenticated subject

- `security.authenticate / authenticated` のnon-empty subjectIdだけをruntime subjectとして受理
- candidate actor labelとは別identityとして保持
- principal gapだけを解消
- Authorization gapは解消しない

### P4 — Application access = read

- `read`から `browse` を生成しない
- Pattern identityからAuthorization resourceを自動生成しない

### P5 — policy-only target injection

test-onlyに `browse / order_entry.customer_reference` allow ruleを追加しても、明示的なBusiness/Application targetがない限りAuthorization targetをbackfillしない。

## 8. Runtime contextとstatic freshnessを混ぜない

R3の `RequiredInputFingerprints` はstatic read routeのfreshnessを表す。

一方、authenticated subjectはruntime Observationであり、Repository content fingerprintではない。

```text
static source freshness
  != runtime principal freshness / session validity
```

R3AではAuthentication ObservationをR3のstatic fingerprint集合へ追加しない。

将来runtime Evidenceを保持する場合は、session / Observation provenance、expiry、issuer等を別境界として扱う。

## 9. 次に残る設計判断

principalを安全に解決できても、BP-02のAuthorization action/resource authorityは未解決である。

次段では少なくとも次を比較する必要がある。

- Business Procedure側でprotected targetを明示する
- Application Pattern側でAuthorization targetを明示する
- 別のaccepted cross-view mappingを持つ

このR3Aからwinnerを選ばない。

また、action/resource authorityが定まる前に `security.authorize` をinvokeしない。

## 10. Evidence高さ

```text
Design Evidence
  principal / Authorization target authority boundary

Source Evidence
  test-only pressure source

Executable Evidence
  actual test PASS

Integration Evidence
  explicit authenticated subject + explicit target + Authorization invocation

Operational Evidence
  repeated runtime authorization with correct expiry / failure / handoff
```

R3AではDesign / Sourceまでを対象とする。
