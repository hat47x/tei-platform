# Actor Context R3J — Authorization Action Authority Evidence Gate

- Status: ISSUE-000053 / Phase 3 action-authority Evidence Gate / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3I_AUTHORIZATION_TARGET_EVALUATION_CADENCE_2026-09.md`
- Related:
  - `plan/LANE_B_AUTHORIZATION_TARGET_AUTHORITY_AUDIT_2026-09.md`
  - `reference/business-system-design/order-entry-business-procedure.json`
  - `reference/business-system-design/order-entry-protected-target-overlay.json`
  - `reference/workspace/applications/order-entry/patterns/submit-order.tei`
  - `reference/legacy-auth/authorization.json`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3IではBP-06について:

```text
Authorization requirement = required
resource = order_entry.submit_order
action = unresolved
```

までを分離した。

R3Jでは、current RepositoryにAuthorization actionのaccepted authorityが存在するかを監査する。

焦点は、複数の一致する手掛かりを足し合わせてもmissing authorityを生成しないことである。

## 2. current Repositoryにある三つのhint

### Business wording

BP-06:

```text
determine whether the authenticated subject may execute the submit application action
```

自然言語として `execute` を含む。

### Application Pattern kind

`order_entry.submit_order` は:

```text
pattern order_entry.submit_order command
```

である。

これはApplication interactionがcommandであることをauthorするが、AuthorizationAction語彙の `execute` をauthorしない。

### Legacy policy

Legacy Authorization fixtureには:

```text
execute / order_entry.submit_order / allow
```

がある。

これはpolicy realization / compatibility Evidenceであり、Business/Application action authorityではない。

## 3. 三つが整合してもauthorityにはならない

current evidenceは人間には非常に自然な対応に見える。

```text
Business wording: execute
Application Pattern: command
Legacy policy: execute
```

しかし:

```text
multiple corroborating hints
  != accepted AuthorizationAction authority
```

である。

二つ以上の非authoritative sourceを多数決・一致率・confidenceで正本化しない。

特に生成AIは語彙一致から `execute` を補完しやすいため、明示的なnegative pressureが必要である。

## 4. current gate result

current Business/Application sourceには、BP-06 protected targetに対して:

```text
AuthorizationAction = execute
source = accepted Business/Application authority
```

と明示するsourceがない。

したがってR3J current gateは:

```text
action authority = unresolved
authorization invocation = blocked
```

である。

これは `execute` が将来不適切だという意味ではない。

```text
likely value
  != authoritative value
```

を維持する。

## 5. test-only explicit action candidate

設計順序をPressureするためだけにtest-only explicit action candidateを与えることはできる。

例:

```text
action = execute
source = test-only:business-protected-action
```

この場合でも:

```text
action resolved
  != Authorization evaluated
  != allowed
```

である。

R3I boundaryへ接続すると次のblockerは `authorization-not-evaluated` になる。

## 6. authoring locusはまだ選ばない

候補は少なくとも:

1. Business Procedureのprotected-action relation
2. Application Patternのexplicit authorization-operation relation
3. accepted cross-view mapping artifact

である。

しかしcurrent一例だけではscope / duplication / reuseの差を十分にPressureできない。

R3Jでは:

- production field
- generic relation
- action enum mapping
- `command -> execute` convention
- policy-to-Business backfill

を追加しない。

## 7. Source Evidence

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3j_authorization_action_authority_gate_pressure_test.go`

Pressureすること:

- BP-06 wordingに `execute` が存在することを確認する
- submit Patternが `command` であることを確認する
- Legacy policyに `execute / order_entry.submit_order` があることを確認する
- 三者がそろってもactionを生成しない
- explicit test-only actionにはsource provenanceを要求する
- action解決後もevaluationを自動成立させない
- confidence / voting / winner fieldを作らない

## 8. 次

action authorityのproduction locusを選ぶには、少なくとも別のAuthorization-protected interactionで:

- action語彙が同じか異なるか
- Pattern kindとの対応が一対一か
- Business context固有actionが必要か
- policy providerを変更してもmeaningが維持されるか

をPressureする必要がある。

qualifying second caseが得られるまではR3J gateを未充足として保持する。
