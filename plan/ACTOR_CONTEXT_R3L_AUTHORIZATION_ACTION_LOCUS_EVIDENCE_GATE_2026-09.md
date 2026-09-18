# Actor Context R3L — Authorization Action Locus Evidence Gate

- Status: ISSUE-000053 / Phase 3 action-locus Evidence Gate / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3K_AUTHORIZATION_ACTION_AUTHORING_SCOPE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3J_AUTHORIZATION_ACTION_AUTHORITY_GATE_2026-09.md`
  - `reference/business-system-design/order-entry-business-procedure.json`
  - `reference/business-system-design/order-entry-follow-up-procedure.json`
  - `reference/business-system-design/order-entry-protected-target-overlay.json`
  - `reference/legacy-auth/authorization.json`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Kでは、将来Authorization action authorityをauthorする場合でも、そのscope候補には少なくとも:

```text
Business Step
Application Pattern
explicit cross-view edge
```

があり、それぞれreachが異なることをPressureした。

次の課題は、三候補のどれかをproduction locusとして選ぶことではない。

まず、**locus選択に耐えるsecond repository-backed protected interactionが存在するか**をGateとして明示する。

R3Lでは:

```text
semantic scope difference
  != production locus qualified
```

を固定する。

## 2. current BP-06が与えるEvidence

current ReferenceではBP-06について:

```text
Authorization requirement = required
protected resource = order_entry.submit_order
Authorization action authority = unresolved
```

である。

したがってBP-06は:

- required Authorization interaction
- Business Step scoped security boundary
- protected resource Evidence

を与えるが、accepted Authorization action authorityまでは与えない。

Business wordingの `execute`、Pattern kindの `command`、Legacy policyの `execute` はR3Jどおりcorroborationであり、action authorityに昇格させない。

## 3. recent_orders reuseはscope pressureだがqualifying second caseではない

current Repositoryには:

```text
order-entry.order-handling/BP-00
order-entry.follow-up/FU-00
  -> order_entry.recent_orders
```

というcross-Procedure reuseが存在する。

また:

```text
Pattern kind = browse
Legacy policy action = browse
```

というcorroborating hintsもある。

しかしBP-00 / FU-00はAuthorization requirementが `unresolved` である。

したがって:

```text
repository-backed Pattern reuse
  + action hints
  != repository-backed required protected interaction
```

である。

R3Kのscope差をPressureするには有効だが、production locusのsecond caseには数えない。

## 4. Action Locus Evidence Gate

production action-authoring locusを比較対象へ上げるため、少なくとも二つの独立interaction recordについて、各recordが次を満たすことを要求する。

### E1 — requirement authority

```text
Authorization requirement = required
```

がBusiness / Application側のexplicit authorityから成立している。

Legacy policy ruleの存在からrequiredを逆算しない。

### E2 — protected resource authority

Authorizationのresourceが明示され、provenanceを持つ。

```text
Pattern used by the Step
  != automatically protected resource
```

を維持する。

### E3 — explicit action authority

Authorization actionがBusiness / Application authorityとして明示され、provenanceを持つ。

Business wording、Pattern kind、Legacy policyからactionを生成しない。

### E4 — repository-backed case

test-only candidateだけではなく、current / future accepted Reference sourceで成立する。

### E5 — locus-discriminating variation

second caseは単なる複製でなく、少なくとも一つ:

- 異なるPattern kind
- 異なるAuthorization action語彙
- cross-Procedure reuse shape
- Business-context固有action
- same Pattern / different Business action relation

のいずれかを持ち、Business Step / Pattern / cross-view locusの意味差を実際にPressureできる。

### E6 — provider independence

Legacy policy providerを取り除いても:

- requirement
- resource
- action

のmeaningをBusiness / Application authorityから説明できる。

## 5. Gate outcome

R3Lでは三状態を区別する。

```text
insufficient-evidence
single-qualifying-interaction
action-locus-pressure-qualified
```

### insufficient-evidence

qualifying interactionが0件、またはrecord自体にE1〜E4 / E6の不足がある。

### single-qualifying-interaction

一つのinteractionだけがE1〜E4 / E6を満たす。

これはaction authorityの存在をPressureするには有効だが、authoring locus比較には不足する。

### action-locus-pressure-qualified

二つ以上のrepository-backed qualifying interactionがあり、かつE5のvariationが観測できる場合のみ到達する。

ただし:

```text
action-locus-pressure-qualified
  != Business Step locus selected
  != Pattern locus selected
  != cross-view locus selected
```

である。

Gate通過後に初めてlocus候補をEvidence-backedに比較してよい。

## 6. current Repository gate result

current Referenceでは:

```text
BP-06
  required = yes
  resource authority = yes
  action authority = no

BP-00 / FU-00
  required = unresolved
  action authority = no
```

である。

したがってcurrent gateは:

```text
action locus Evidence Gate
  = insufficient-evidence
```

に留まる。

BP-06へLegacy policyの `execute` をbackfillしてsingle caseを作らない。
recent_ordersへPattern kind / policyの `browse` をbackfillしてsecond caseを作らない。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3l_authorization_action_locus_evidence_gate_pressure_test.go`

Pressureする不変条件:

- current BP-06はaction authority unresolvedのためqualifying recordにならない
- current BP-00 / FU-00はrequirement unresolvedのためqualifying recordにならない
- policy / Pattern kind / wordingのhintを不足Evidenceの補完に使わない
- one qualifying caseだけでproduction locusを選ばない
- two qualifying casesでもvariationがなければlocus selection pressureを開かない
- two qualifying + variationで初めてcomparison pressureを開く
- Gate resultにWinner / PreferredLocus / Confidenceを持たせない

## 8. current conclusion

R3Lで固定するのは次である。

```text
action authority candidate exists
  != action authoring locus qualified

one protected interaction
  != reusable locus rule

corroborating hints
  != missing authority

second repository-backed qualifying interaction
  + discriminating variation
  -> locus comparison may begin
  != locus winner selected
```

production field、generic mapping artifact、action enum mapping conventionは追加しない。

## 9. 次

current gateが未充足なので、action-locus branchはここで停止できる。

read-only AI Business Actor本線では、BP-06 action authorityが未解決である以上actual Authorization invocationへ進まない。

次に進める候補は二つある。

1. qualifying second protected interactionを、別のBusiness requirement / Reference seedが成立した後に再監査する。
2. 現在すでに成立している境界を使い、AI Business Actorが**どの未解決authorityで停止したかをhandoff / explanation projectionとしてどう提示するか**をPressureする。

R3Lでは後者を次の安全な進行方向とする。
