# Actor Context R3AO — Explicit Acceptance Witness / Exact Binding Boundary

- Status: ISSUE-000053 / Phase 3 explicit acceptance-witness pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AN_ACCEPTANCE_AUTHORITY_EVIDENCE_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AM_QUALIFIED_EVIDENCE_AUTHORITY_ASSERTION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AN_ACCEPTANCE_AUTHORITY_EVIDENCE_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ANではacceptance authority evidenceがqualifyしても:

```text
authority evidence qualified
  != assertion accepted
```

を固定した。

R3AOではaccept / rejectを観測するために、test-only explicit acceptance witnessを:

```text
exact AssertionIdentity
+ exact qualified acceptance authority evidence
+ explicit decision
+ acceptance provenance
```

へbindingする。

ただし:

```text
explicit acceptance witness observed
  != Accepted Decision applied
  != current authority rewritten
```

である。

## 2. current corpus

current Repository-backed R3AN stateは:

```text
acceptance-authority-unresolved
```

である。

したがってcurrent corpusだけからR3AO witnessを生成しない。

accept / reject caseはR3AN GateそのものをPressureするsynthetic complete acceptance-authority evidenceを用いる。

```text
qualified assertion exists
  + acceptance authority unresolved
  -> no acceptance witness observation
```

である。

## 3. witness shape

test-only witnessは最低限:

```text
AssertionIdentity
AcceptanceAuthorityEvidenceID
Decision
AcceptanceProvenanceRef
```

を持つ。

Decisionは:

- accept
- reject

だけをPressureする。

assertionのAssertedValue、scope specificity、Evidence件数からDecisionを推測しない。

## 4. exact assertion binding

witnessのAssertionIdentityはR3AM assertionのexact identityと一致しなければならない。

```text
same AssertionClass
same AssertedValue
same EvidenceRef family
```

だけでは不足する。

stale / other assertion identityはfail closedする。

## 5. exact acceptance authority evidence binding

witnessが参照するAcceptanceAuthorityEvidenceIDは:

- R3AN gateがsingle `acceptance-authority-evidence-qualified`
- GateのQualifiedEvidenceIDsがexactly one
- witnessのEvidenceIDがそのexact IDと一致
- current candidate setにexact IDが存在
- current AssertionClass / BusinessAddressをcandidateが実際にcover
- DecisionLocusRefがcandidateに明示されている

ことを要求する。

source locationやscope類似から別candidateを補完しない。

## 6. multiple-qualified precedence unresolvedを迂回しない

R3ANが:

```text
acceptance-authority-multiple-qualified-precedence-unresolved
```

なら、witnessが一つのEvidenceIDを名指ししていてもR3AOはbindingしない。

```text
witness names one candidate
  != precedence resolved
```

である。

R3AOはR3AN precedence boundaryを上書きしない。

## 7. explicit decisionだけを保持する

witness Decisionが空なら:

- asserted valueがpositive-looking
- acceptance authority scopeがexact
- candidateのDecisionSemanticsにacceptがある

としてもacceptへ推論しない。

同様にrejectも推論しない。

unsupported decisionはfail closedする。

## 8. acceptance provenance

AcceptanceProvenanceRefは必須である。

```text
AcceptanceAuthorityEvidenceID
  != AcceptanceProvenanceRef

AssertionSourceRef
  != AcceptanceProvenanceRef
```

であり得る。

authority definitionとactual decision witnessのprovenanceを分離する。

## 9. no witness

R3AN authority evidenceがsingle-qualifiedでもwitnessがなければ:

```text
acceptance witness = absent
```

のままでよい。

```text
no witness
  != reject
  != accept
  != assertion invalid
```

である。

## 10. conflicting witnesses

同一AssertionIdentity / AcceptanceAuthorityEvidenceIDについて:

- accept witness
- reject witness

が別provenanceから存在し得る。

R3AOは両方を独立したattributed observationとして保持し:

```text
conflicting explicit witnesses
  != last writer wins
  != accept wins
  != reject wins
  != current authority
```

とする。

exact duplicate witnessだけをnormalizeする。

conflict resolutionは別authorityが必要である。

## 11. observed witness projection

test-only observed witnessは:

- AssertionIdentity
- AssertionClass
- AssertedValue
- AcceptanceAuthorityEvidenceID
- AuthorityDeclarationRef
- DecisionLocusRef
- Decision
- AcceptanceProvenanceRef
- WitnessIdentity
- AuthorityLimits

までを持てる。

これはAccepted Decision recordではない。

```text
observed accept witness
  != AcceptedAuthorityValue
  != current R3Y authority

observed reject witness
  != Authorization denied
  != RuntimeOutcome = denied
```

である。

## 12. 禁止するshortcut

R3AO shapeは次を持たない。

- AcceptedAuthorityValue
- AcceptedDecision
- AcceptanceStatus
- CurrentAuthority
- ReevaluationAuthorityStatus
- ReuseAuthorityStatus
- RuntimeOutcome
- AuthorizationResult
- Applied
- Apply / AutoApply
- Resolved / ResolutionComplete
- CanExecute / Authorized / Allowed
- ToolCall / Command / NextAction
- HandoffOwner
- Confidence / Priority / Severity / Winner

## 13. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ao_explicit_acceptance_witness_boundary_pressure_test.go`

Pressureするcase:

1. current unresolved Gateではwitnessをbindingできない
2. single-qualified authority evidence + exact explicit witnessをbindできる
3. no witnessではaccept / rejectを生成しない
4. wrong AssertionIdentityはfail closed
5. non-qualified / unknown authority evidence IDはfail closed
6. multiple-qualified precedence unresolvedではbindingしない
7. missing decision / unsupported decisionはfail closed
8. missing AcceptanceProvenanceRefはfail closed
9. assertion内容やexact scopeからdecisionを推測しない
10. conflicting accept / reject witnessesをwinner選択せず保持する
11. exact duplicate witnessだけnormalizeする
12. input order / repeated bindingでidentityを変えない
13. Accepted Decision / execution shortcutを追加しない

## 14. current conclusion

R3AOで固定するのは:

```text
qualified acceptance authority evidence
  + exact explicit witness
  -> provenance-bound acceptance witness observation

acceptance witness observation
  != Accepted Decision applied
  != current authority
  != Authorization outcome
  != execution authority
```

である。

production approval workflow、decision store、authority rewrite、Command executionは追加しない。

## 15. 次

次段ではexplicit acceptance witnessを観測できても:

```text
acceptance witness observed
  != accepted value committed to authoritative source
```

を維持し、Accepted Decisionとしてcommitするためのauthoritative write / provenance boundaryをEvidence GateでPressureする。
