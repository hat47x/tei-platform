# Actor Context R3AN — Assertion Acceptance Authority / Decision Locus Evidence Gate

- Status: ISSUE-000053 / Phase 3 acceptance-authority Evidence Gate / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AM_QUALIFIED_EVIDENCE_AUTHORITY_ASSERTION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3L_AUTHORIZATION_ACTION_LOCUS_EVIDENCE_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AL_RESPONSE_EVIDENCE_QUALIFICATION_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AMでは:

```text
qualified evidence
  -> provenance-bound authority assertion

asserted authority value
  != accepted authority value
  != Accepted Decision
```

を固定した。

R3ANではassertionをAccepted Decision候補として扱う前に:

```text
assertion provenance available
  != acceptance authority available
```

をRepository-backed Evidence GateとしてPressureする。

## 2. current corpus

current Reference / plan / fixtureでは、R3AM assertionとは別に:

- assertion acceptance authority declaration
- acceptance scope
- acceptance decision locus
- accept / reject decision semantics

を同時に満たすrepository-backed sourceを確認できない。

したがってcurrent statusは:

```text
acceptance-authority-unresolved
```

とする。

これはassertionがincorrectという意味ではない。

## 3. qualifying evidence requirements

test-only acceptance authority evidenceがqualifyするには少なくとも:

### E1 repository-backed

candidateがrepository-backed evidenceである。

synthetic runtime observationやLLM suggestionだけでは不可。

### E2 explicit authority declaration

```text
AuthorityKind = assertion-acceptance-authority
AuthorityDeclarationRef != empty
```

が必要。

AssertionSourceRefやQualificationSourceRefの存在だけからacceptance authorityを推定しない。

### E3 explicit assertion-class coverage

candidateがR3AMのexact AssertionClassを明示的にcoverageする。

近い語彙、same information class、same source familyでは補完しない。

### E4 explicit scope coverage

current BusinessAddressをscopeが明示的にcoverする。

test-only scopeは:

- exact-business-address
- any-business-address

だけをPressureする。

scope specificityはqualify条件の一部であってprecedenceではない。

### E5 explicit decision locus

DecisionLocusRefが必要。

source locationやfile pathそのものをdecision locusとして推測しない。

### E6 explicit decision semantics

少なくとも:

- accept
- reject

をdecision semanticsとして明示する。

「approve」という語が本文にあるだけ、あるいはassertion valueと同じ語彙があるだけでは不可。

## 4. current assertion sourceはacceptance authorityではない

R3AMの:

- EvidenceRef
- AssertionSourceRef
- AssertionIdentity

はassertion provenanceである。

```text
assertion source
  != acceptance authority declaration
  != decision locus
```

である。

same artifactが将来explicit acceptance authorityも宣言すること自体は排除しないが、そのauthority declarationを別Evidenceとして明示する必要がある。

## 5. corroboration countはauthorityにならない

同じAssertedValueを複数のqualified Evidenceが主張しても:

```text
more assertions
  != acceptance quorum
  != accepted value
```

である。

R3ANはassertion件数からauthorityを生成しない。

## 6. scope specificityはwinnerを作らない

exact-business-address candidateとany-business-address candidateが双方qualifyする場合:

```text
multiple acceptance-authority evidence qualified
  -> precedence unresolved
```

とする。

exact scopeを自動winnerにしない。

```text
more specific scope
  != precedence authority
```

である。

## 7. source locationもwinnerを作らない

Business Procedure近傍、Application source近傍、Authorization fixture近傍などのfile locationはacceptance authority / precedenceを意味しない。

Legacy policyも、明示されたacceptance authority declarationがない限りR3AM assertion acceptanceのauthorityへ昇格させない。

## 8. gate result

test-only gate resultは:

- AcceptanceAuthorityStatus
- QualifiedEvidenceIDs
- UnresolvedReasons
- AuthorityLimits

までを持てる。

statusは:

```text
acceptance-authority-unresolved
acceptance-authority-evidence-qualified
acceptance-authority-multiple-qualified-precedence-unresolved
```

をPressureする。

```text
authority evidence qualified
  != assertion accepted
```

である。

## 9. GateはAccepted Decisionを作らない

R3ANは次を作らない。

- AcceptedAuthorityValue
- AcceptedDecision
- AcceptanceDecision
- SelectedAuthority
- PreferredAuthority
- Winner
- CurrentAuthority
- ReevaluationAuthorityStatus
- ReuseAuthorityStatus
- Approved / Rejected
- Resolved
- Apply / AutoApply
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3an_acceptance_authority_evidence_gate_pressure_test.go`

Pressureするcase:

1. current corpus / no acceptance authority evidence -> unresolved
2. assertion source / assertion countだけではqualifyしない
3. complete repository-backed acceptance authority evidenceはqualifyできる
4. missing E1–E6はいずれもqualifyしない
5. wrong AssertionClass coverageはqualifyしない
6. wrong BusinessAddress scopeはqualifyしない
7. exact + any scopeの複数qualifying evidenceはprecedence unresolved
8. scope specificityでwinnerを選ばない
9. input orderでnormalized gate identityを変えない
10. gate shapeにAccepted Decision / execution shortcutを追加しない

## 11. current conclusion

R3ANで固定するのは:

```text
qualified assertion
  + repository-backed acceptance authority evidence
  -> acceptance authority evidence qualified

acceptance authority evidence qualified
  != assertion accepted
  != Accepted Decision
```

である。

current corpusではacceptance authority evidenceはunresolvedのまま。

production decision store、approval workflow、authority directory、precedence engine、tool invocationは追加しない。

## 12. 次

次段ではacceptance authority evidenceがqualifyしたという仮定の下で:

```text
explicit acceptance witness
  -> exact AssertionIdentity
  + exact acceptance authority evidence
  + acceptance provenance
```

を要求し、accept / reject decisionをassertion内容やscope specificityから推測しない境界をPressureする。
