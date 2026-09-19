# Actor Context R3AM — Qualified Evidence Authority Assertion / Acceptance Boundary

- Status: ISSUE-000053 / Phase 3 qualified-evidence assertion pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AL_RESPONSE_EVIDENCE_QUALIFICATION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AK_REQUEST_CORRELATION_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ALではcorrelated responseのEvidenceRefを:

```text
observed
  -> qualification-unresolved
  -> explicit-evidence-qualified
```

まで分離した。

R3AMではexplicitにqualifiedされたEvidenceがauthority valueをassertする場合でも:

```text
asserted authority value
  != accepted authority value
  != Accepted Decision
```

を固定する。

## 2. assertion witness

test-only assertion witnessは最低限:

```text
EvidenceRef
AssertionClass
AssertedValue
AssertionSourceRef
```

を持つ。

例:

```text
AssertionClass = reevaluation-authority
AssertedValue = explicit-reevaluation-required
```

と記述できる。

ただしこれは:

```text
qualified Evidence says X
```

を記録するだけで:

```text
TEI accepts X
current authority = X
```

を意味しない。

## 3. exact EvidenceRef binding

assertion witnessはR3AL assessmentのexact EvidenceRefへだけ適用する。

```text
same response
same information class
similar locator
```

だけで別Evidenceへassertionを伝播しない。

current assessmentに存在しないEvidenceRefへのassertionはfail closedする。

## 4. qualification prerequisite

R3AMでqualified-evidence assertionとしてmaterializeできるのは:

```text
QualificationStatus = explicit-evidence-qualified
```

のEvidenceだけである。

```text
qualification-unresolved
  + assertion witness
  != qualified assertion
```

とし、fail closedする。

これはassertion内容の正しさを判定するためではなく、R3ALのqualification境界を迂回させないためである。

## 5. assertion provenance

AssertionSourceRefを必須にする。

```text
EvidenceRef
  != AssertionSourceRef
```

であり得る。

Evidenceがqualifiedでも、誰／どのartifactがそのauthority value assertionを供給したかが追跡できなければmaterializeしない。

## 6. current authorityへ書き戻さない

asserted valueがR3Yの既存語彙と一致しても:

```text
AssertedValue = explicit-reevaluation-required
  != ReevaluationAuthorityStatus = explicit-reevaluation-required
```

である。

同様に:

```text
AssertedValue = explicit-reuse-permitted
  != ReuseAuthorityStatus = explicit-reuse-permitted
```

とする。

R3AMはR3Y authority resultを書き換えない。

## 7. conflictをwinnerへ丸めない

異なるqualified Evidenceが同じAssertionClassについて異なるAssertedValueを主張し得る。

例:

```text
evidence-a -> reevaluation-authority = explicit-reevaluation-required
evidence-b -> reevaluation-authority = test-only:not-required
```

R3AMでは両assertionを独立して保持する。

```text
conflicting assertions
  != choose winner
  != last writer wins
  != stronger evidence wins
```

である。

exact duplicate assertionだけはnormalized identityとして一つにできる。

## 8. assertion identity

test-only assertion identityは:

```text
RequestIdentity
RequestRevisionIdentity
ResponseRef
EvidenceRef
AssertionClass
AssertedValue
AssertionSourceRef
```

の正規化された組で表現できる。

presentation wording、input order、audienceはidentityへ入れない。

## 9. no assertion response

explicitly qualified Evidenceが存在してもassertion witnessがなければ、assertion listは空でよい。

```text
qualified evidence
  + no authority assertion
  != authority resolved
```

generic Accepted / Rejectedへ丸めない。

## 10. 禁止するshortcut

R3AM assertion shapeは次を持たない。

- AcceptedAuthorityValue
- AcceptedDecision
- AcceptanceStatus
- Approved / Rejected
- Sufficient
- Resolved / ResolutionComplete
- ReevaluationAuthorityStatus
- ReuseAuthorityStatus
- CurrentAuthority
- Apply / AutoApply
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Retry / Reauthorize
- Confidence / Priority / Severity / Winner

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3am_qualified_evidence_authority_assertion_pressure_test.go`

Pressureするcase:

1. explicit-qualified exact EvidenceRefからauthority assertionをmaterializeする
2. qualification-unresolved Evidenceからのassertionはfail closed
3. unknown EvidenceRefへのassertionはfail closed
4. AssertionSourceRef欠落はfail closed
5. asserted reevaluation valueをR3Y current authority statusへ書き戻さない
6. asserted reuse valueをprotected operation execution authorityへ昇格させない
7. conflicting qualified assertionsをwinner選択せず独立保持する
8. exact duplicate assertionだけをnormalizeする
9. no assertion witnessならqualified Evidenceがあってもassertion listは空
10. input order / repeated materializationでidentityを変えない
11. acceptance / execution shortcutを追加しない

## 12. current conclusion

R3AMで固定するのは:

```text
qualified evidence
  -> provenance-bound authority assertion

authority assertion
  != accepted authority value
  != Accepted Decision
  != current Authorization authority
  != side effect
```

である。

production Accepted Decision store、approval workflow、authority registry、trust store、Command executionは追加しない。

## 13. 次

次段では複数のqualified assertionをAccepted Decision候補として扱う前に:

```text
assertion provenance available
  != acceptance authority available
```

を維持し、Repository-backedなacceptance authority / decision locusのEvidence GateをPressureする。
