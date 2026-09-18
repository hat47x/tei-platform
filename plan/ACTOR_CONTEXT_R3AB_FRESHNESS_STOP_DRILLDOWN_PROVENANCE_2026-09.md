# Actor Context R3AB — Freshness Stop Drill-down Provenance Boundary

- Status: ISSUE-000053 / Phase 3 freshness-stop drill-down provenance pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3V_CROSS_AUDIENCE_REASON_IDENTITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3X_AUTHORIZATION_CONTEXT_IDENTITY_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Z_AUTHORIZATION_FRESHNESS_PRECEDENCE_EVIDENCE_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AAではsame-tupleのAuthorization freshness authority unresolvedを、R3M〜R3Pの既存Operational stop / handoffへ接続した。

次はHuman / Agentのどちらからfreshness stopをdrill downしても、同じprovenance chainへ戻れることをPressureする。

```text
freshness stop
  -> R3X context identity boundary
  -> R3Y freshness authority boundary
  -> R3Z precedence Evidence Gate
```

ただし三つのsource roleを一つの「Authorization authority source」へ平坦化しない。

## 2. provenance root roleを分ける

test-onlyには少なくとも次のroleを持つ。

### R3X — context identity boundary

```text
role = authorization-context-identity-boundary
```

Business address / subject / action / resource tupleの一致・不一致を説明する。

これは:

```text
context identity evidence
  != reevaluation directive
  != reuse permission
```

である。

### R3Y — freshness authority boundary

```text
role = evaluation-freshness-authority-boundary
```

reevaluation authorityとprior Observation reuse authorityを別axisとして説明する。

R3Y plan自体はcurrent production directiveではない。

### R3Z — Evidence Gate

```text
role = freshness-precedence-evidence-gate
```

precedence mechanismを作るためのEvidence充足状態を説明する。

```text
Evidence Gate
  != Authorization decision authority
  != effective precedence rule
```

である。

## 3. Human / Agentでroot identityを共有する

R3PではHuman orientationとAgent resolution-inspectionでdetail disclosureが異なる。

しかしR3AA freshness stopのselected `source-refs` については:

```text
Human drill-down roots
  = Agent drill-down roots
```

を要求する。

これはuniversal packetを意味しない。

```text
same provenance roots
  != same disclosure set
```

である。

## 4. presentation wordingをroot identityへ含めない

Human表示文はderived presentationである。

concise / plain等の表示variantが変わっても:

- stop machine reason
- resolution need
- source ref
- root role

は変わらない。

```text
presentation
  != provenance identity
```

を維持する。

## 5. unknown source refはfail closed

freshness stopのdrill-down source refが既知roleへ分類できない場合、推測で:

- authority
- policy
- owner
- runtime Observation

へ分類しない。

```text
unknown source role
  -> drill-down provenance unresolved
```

としてfail closedする。

## 6. source roleから値を補わない

R3X / R3Y / R3Zへ到達しても:

- current Authorization outcome
- reevaluation required?
- reuse permitted?
- precedence winner
- handoff owner

をdrill-down resolverが生成しない。

drill-downは「どのsource / boundaryへ戻るか」を示すだけである。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ab_freshness_stop_drilldown_provenance_pressure_test.go`

Pressureするcase:

1. freshness stop source refsをR3X / R3Y / R3Z roleへ分類する
2. Human / Agentでroot identityが一致する
3. Human display wording変更でroot identityが変わらない
4. R3Z Evidence GateをAuthorization authorityへ昇格させない
5. R3X context identityをfreshness directiveへ昇格させない
6. unknown source refを推測分類しない
7. Truth / EffectiveDecision / HandoffOwner / NextAction等を追加しない

## 8. current conclusion

R3ABで固定するのは:

```text
same stop reason
  + same source refs
  -> same drill-down provenance roots

source role preserved
  != authority flattened

same provenance
  != same disclosure
  != same presentation
```

である。

production navigation schema、UI routing、ticket link、cross-agent protocolは追加しない。

## 9. 次

次段ではdrill-down rootからactual Repository source / runtime Evidenceへ戻る際に、static sourceとruntime Observationを同じlocator contractへ潰さないことをPressureする。

特にR3Rのfreshness planeとR3ABのdrill-down roleを接続する。
