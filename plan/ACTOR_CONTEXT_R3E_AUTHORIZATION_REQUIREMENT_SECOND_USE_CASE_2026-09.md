# Actor Context R3E — Authorization Requirement Second-use-case Pressure

- Status: ISSUE-000053 / Phase 3 second-use-case pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3D_AUTHORIZATION_REQUIREMENT_AUTHORING_SCOPE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Dでは、Authorization requirementについて:

```text
Business Step scope
  != Application Pattern scope
```

を分離した。

しかしcurrent `order_entry.customer_reference` はBusiness Procedure上の利用がBP-02一件しかなく、どちらのscopeを選んでもaffected setが同じになる。

R3Eでは、production fieldを追加せず、同じApplication Patternを複数Business contextから利用する第二の現実的なuse caseをtest-onlyに置き、scope選択に必要なEvidenceをPressureする。

## 2. 二つのPressure case

### Case A — 一方だけrequirementが明示される

同じ `order_entry.customer_reference` を:

```text
order-entry.order-handling / BP-02
customer-inquiry.customer-confirmation / CI-01
```

から利用する。

BP-02だけがexplicit `required`、CI-01は`unresolved`とする。

このとき:

```text
Business Step scoped requirement
  -> BP-02 only

Application Pattern scoped requirement
  -> BP-02 + CI-01
```

となる。

したがって、一つのBusiness contextでAuthorization requirementが成立したことだけからPattern全体へscopeを昇格すると、別contextの未解決authorityを暗黙に埋める。

```text
same Pattern reused
  != same Authorization requirement already accepted
```

である。

### Case B — 複数contextが独立に同じrequirementを持つ

次に、BP-02とCI-01の両方が、それぞれのBusiness authorityからexplicit `required`を持つcaseを置く。

この場合、Business Step scopeでは同じrequirementが複数箇所へ現れるため、Pattern-level invariant候補を検討するpressureが生じる。

ただし:

```text
same status repeated twice
  != Pattern authority proven
```

である。

偶然同じ、現時点で同じ、同一security policyで実現される、という事実だけではApplication interaction invariantを証明しない。

Pattern側へ上げるには少なくとも:

- Patternの意味そのものにrequirementが属すること
- Business context差で変化しないこと
- explicit exception / override semanticsが必要か
- provenanceを失わないこと

を別Evidenceで確認する必要がある。

## 3. unresolvedをnot-requiredへ変換しない

Case Aの第二contextは`unresolved`であり、`not-required`ではない。

これは重要である。

```text
no requirement authored in this context
  != explicitly not required
```

R3Eは、Pattern scope採用によってunresolvedを自動requiredへ変えることも、Business Step scope採用によってunresolvedをnot-requiredへ変えることも禁止する。

## 4. requirement applicabilityとevaluation cadenceを分ける

将来、同じAuthorization decisionを複数Stepで再利用できるとしても、それは:

```text
Authorization applicable here
  != Authorization must be re-evaluated here
```

である。

R3Eではevaluation reuse / session scope / cache / prior-step inheritanceをrequirement authorityへ混ぜない。

## 5. test-only source

- `tools/reference-business-procedure-check/actor_context_r3e_authorization_second_use_case_pressure_test.go`

ここではproduction fixtureを変更せず、test-onlyのuse recordsを組み立てる。

Pressureする不変条件:

- Pattern reuseだけで別Business contextへrequirementを伝播しない
- unresolvedをrequired / not-requiredへ補完しない
- 複数contextで同じrequirementが明示された場合はduplication pressureとして観測する
- duplication pressureをPattern authority成立へ読み替えない
- supporting capability `security.authorize` の一致をscope authorityへしない

## 6. current conclusion

R3Eで確定するのは次までである。

```text
one explicit Business requirement
  + same Pattern reused elsewhere
  != Pattern-wide requirement authority

multiple independent explicit requirements
  -> Pattern invariant candidate pressure
  != Pattern authority proven

unresolved
  != not-required
  != inherited required
```

したがって、R3DのBusiness Step / Application Patternの二択をまだproduction schemaへ固定しない。

## 7. 次

次段では、Pattern invariant候補を成立させるために必要なEvidenceと、Business contextによるvariationを許す場合のprecedenceを分離してPressureする。

候補として:

```text
Pattern minimum invariant
Business Step additional requirement
explicit exception / override
```

の関係を比較するが、R3E時点ではproduction relation / precedence ruleを追加しない。
