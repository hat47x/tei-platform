# Actor Context R3I — Authorization Target / Evaluation Cadence Boundary

- Status: ISSUE-000053 / Phase 3 target-cadence pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3H_AUTHORIZATION_EXCEPTION_EVIDENCE_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3A_PRINCIPAL_AUTHORIZATION_BOUNDARY_2026-09.md`
  - `reference/business-system-design/order-entry-protected-target-overlay.json`
  - `reference/legacy-auth/authorization.json`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3C〜R3HでAuthorization requirementのapplicability authorityを分離した。

次に再接続するのは:

```text
Authorization requirement
  -> protected target/action authority
  -> Authorization evaluation
```

である。

R3Iでは、この三段を再び一つへ潰さない。

```text
requirement required
  != target/action resolved
  != evaluation current
  != allow/deny outcome
```

## 2. BP-06はrequirementとresource targetを持つがactionは別

current BP-06は:

```text
realization = security-boundary
capability = security.authorize
  -> requirement = required
```

である。

またtest/reference overlayには:

```text
BP-06
  -> applicationPatternTarget = order_entry.submit_order
```

がある。

これはprotected resource候補のEvidenceになる。

しかしoverlayはAuthorization actionをauthorしていない。

Legacy policyには:

```text
execute / order_entry.submit_order / allow
```

があるが、R3Aと同じく:

```text
policy action/resource pair
  != missing Business/Application target authority
```

である。

したがってR3Iでは:

```text
requirement = required
resource = order_entry.submit_order
action = unresolved
  -> Authorization invocation remains blocked
```

とする。

## 3. requirement scopeからtarget scopeを継承しない

将来Pattern-level requirementが成立しても:

```text
Pattern requirement applies
  != Pattern identity is Authorization resource
  != Application access word is Authorization action
```

である。

Business Step requirementについても同じである。

requirementがどのscopeに属するかと、Authorizationに渡すaction/resourceを誰がauthorするかは独立にする。

## 4. evaluation cadenceはさらに別authority

action/resourceが明示されAuthorization evaluationが成功したとしても、その結果をどこまで再利用できるかは別問題である。

少なくとも:

```text
same principal
same action/resource
same Application Pattern
same Procedure family
```

だけでは、別Step / 別Procedureでprior evaluationを自動再利用しない。

必要なのは別のruntime authority / validity contextである。

候補となり得る概念:

- same invocation only
- same Business Step instance
- same Procedure instance
- session scoped
- explicit reusable authorization grant

ただしR3Iではproduction cadence modelを選ばない。

## 5. prior evaluation Observationを静的authorityへしない

Authorization evaluation resultはruntime Observationである。

```text
Repository source fingerprint
  != Authorization decision freshness
```

である。

R3Iではprior evaluationをstatic source fingerprintへ混ぜず、current contextで再利用可能かを別状態として扱う。

## 6. Pressure cases

### Case A — required + resource only

```text
requirement = required
resource resolved
action unresolved
  -> authorization-action-unresolved
```

policyの `execute` をbackfillしない。

### Case B — required + explicit action/resource + no evaluation

```text
requirement = required
action/resource resolved
evaluation = absent
  -> authorization-not-evaluated
```

### Case C — prior evaluation exists in another Business context

```text
prior evaluation = observed
current context = different Step / Procedure
reuse authority = unresolved
  -> current-context-evaluation-unresolved
```

same targetだからといってreuseしない。

### Case D — test-only explicit reuse authority

比較のためtest-onlyにreuse authorityを与えることはできるが、それをproduction cadenceとして採用したとは扱わない。

またreuse authorityがあっても、そのObservationがexpired / revoked / principal-changedでないことまではR3Iで証明しない。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3i_authorization_target_cadence_pressure_test.go`

Pressureする不変条件:

- requiredからaction/resourceを生成しない
- BP-06 resource overlayからactionを生成しない
- Legacy policyからmissing actionをbackfillしない
- Pattern identityをresourceへ自動変換しない
- prior evaluationを別Business contextへ暗黙reuseしない
- reuse authorityとruntime resultを分離する
- `Authorized` / `Allowed` / `CanExecute` / `CacheValid` shortcutを作らない

## 8. current conclusion

R3Iで固定する順序は:

```text
requirement applicability resolved
  -> principal if required
  -> action/resource authority resolved
  -> Authorization evaluation for current context
  -> downstream invocation
```

である。

さらに:

```text
evaluation observed once
  != evaluation reusable elsewhere
```

を保持する。

production target schema、evaluation cache scope、session semantics、TTL / revocation modelは追加しない。

## 9. 次

次段では、current BP-06のresource targetに対してaction authorityをどこに置けるかをPressureする。

候補は:

- Business Procedure protected action
- Application Pattern operation semantics
- accepted cross-view target mapping

だが、Legacy policy actionを正本へ昇格させない。
