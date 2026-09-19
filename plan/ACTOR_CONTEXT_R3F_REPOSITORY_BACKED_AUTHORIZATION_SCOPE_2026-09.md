# Actor Context R3F — Repository-backed Cross-Procedure Authorization Scope

- Status: ISSUE-000053 / Phase 3 repository-backed scope pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3E_AUTHORIZATION_REQUIREMENT_SECOND_USE_CASE_2026-09.md`
- Related:
  - `reference/business-system-design/order-entry-business-procedure.json`
  - `reference/business-system-design/order-entry-follow-up-procedure.json`
  - `reference/legacy-auth/authorization.json`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Eでは、同じApplication Patternを二つのBusiness contextから使うsecond-use caseをtest-only recordとしてPressureした。

Repositoryを再点検すると、同種のreuseはすでにcurrent Referenceに存在する。

```text
order-entry.order-handling / BP-00
  determine which order requires attention
  -> order_entry.recent_orders

order-entry.follow-up / FU-00
  find the order that requires follow-up
  -> order_entry.recent_orders
```

R3Fではこの既存Referenceを利用し、R3D/R3Eのscope差をsynthetic recordだけでなくRepository-backed Source Evidenceへ近づける。

## 2. 同じPattern、異なるBusiness context

二つのStepは同じApplication Patternを使うが、Business Action wordingとProcedure purposeは異なる。

したがって:

```text
same Application Pattern
  != same Business Action
  != same Business context
```

である。

Business Step scoped requirementをBP-00へ置くならaffected addressは:

```text
order-entry.order-handling/BP-00
```

だけである。

Application Pattern scoped requirementを `order_entry.recent_orders` へ置くならaffected setは:

```text
order-entry.order-handling/BP-00
order-entry.follow-up/FU-00
```

となる。

これにより、R3Dでtest-onlyに作った第二Stepなしでもscope差を実Repository上で観測できる。

## 3. Procedure-scoped Step addressを維持する

cross-Procedure pressureではStep IDだけをglobal identityとして扱わない。

```text
Procedure + Step local address
  != global Step identity
```

既存Business Procedure設計のaddress boundaryに従い、R3FのBusiness Step scopeも `Procedure/Step` で識別する。

## 4. Legacy policyが既に同Patternをtargetにしている

current Legacy Authorization policyには:

```text
browse / order_entry.recent_orders / allow
```

が存在する。

これはR3Fにとって重要なnegative pressureになる。

BP-00 / FU-00のcurrent Business Procedure sourceはいずれも `security.authorize` requirementを明示していない。

したがって:

```text
shared policy resource exists
  != BP-00 Authorization requirement = required
  != FU-00 Authorization requirement = required
  != Application Pattern scope selected
```

とする。

policyがPattern IDをresourceとして使っている事実は、policy realization / corroboration上の事情であり、Business/Application側のapplicability scopeをauthorしない。

## 5. current requirement state

current Referenceに対して既存R3C derivationをそのまま適用すると:

```text
BP-00 = unresolved
FU-00 = unresolved
```

である。

これは:

```text
unresolved
  != not-required
```

を維持する。

またpolicy ruleの存在によって `required` へ昇格させない。

## 6. R3Eとの関係

R3Eのsynthetic second-use recordは、次の極端な二ケースを独立にPressureするため依然有効である。

- 一方だけexplicit required
- 両方がindependently required

R3Fはそれを置き換えるのではなく、current Repositoryにも実際のPattern reuseがあることをcorroborateする。

ただしcurrent BP-00 / FU-00はいずれもrequirement unresolvedなので、R3FだけではBusiness Step scope / Pattern scopeのwinnerを選べない。

## 7. second corpus / Pre-ADR blockerとの境界

R3Fは:

- 同一 `order_entry` Application
- 同じsales/order-entry Reference family
- 既存primary + follow-up Procedure

の範囲内である。

したがって、Business Procedure Contract側で要求しているindependent second realistic corpus / second-domain Evidenceではない。

```text
cross-Procedure reuse in first-domain Reference
  != independent second corpus
  != Pre-ADR blocker B resolved
```

R3Fを理由にISSUE-000048 / ISSUE-000050のEvidence gateを閉じない。

## 8. current conclusion

R3Fで強まるEvidenceは次である。

```text
repository-backed same-Pattern reuse
  -> Business Step scope and Pattern scope have observably different reach

existing browse policy for same Pattern
  != applicability authority
  != scope authority

current two Business contexts both unresolved
  -> production locus still undecided
```

production schema、override / precedence、Application-wide profileは追加しない。

## 9. 次

次段では、Pattern-level minimum invariantとBusiness-context variationを同時に表す必要が本当にあるかをPressureする。

その際も:

```text
minimum invariant
  != default policy
  != inherited runtime decision
```

を分離し、precedence mechanismをEvidenceなしに先取りしない。
