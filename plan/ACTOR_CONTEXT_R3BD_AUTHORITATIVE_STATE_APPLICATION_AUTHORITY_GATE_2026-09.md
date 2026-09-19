# Actor Context R3BD — Authoritative-state Application Authority Evidence Gate

- Status: ISSUE-000053 / Phase 3 authoritative-state application authority pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3BC_RECONCILIATION_RESULT_ACCEPTANCE_WITNESS_CANDIDATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3BB_RECONCILIATION_RESULT_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AW_POST_WRITE_RECONCILIATION_AUTHORITY_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3BCではexplicit witnessからtest-only reconciliation-result acceptance candidateを作り:

```text
reconciliation-result acceptance candidate
  != authoritative result applied
  != state synchronized
  != current authority applied
```

を固定した。

R3BDではR3BC `accept-reconciliation-result` candidateが存在しても:

```text
accepted result intent
  != authoritative-state application authority
  != state application
  != CurrentAuthority projection
```

を維持し、authoritative stateへ適用できるRepository-backed authority locusが存在するかをEvidence GateでPressureする。

## 2. reject result candidateはapplication入口ではない

R3BC `reject-reconciliation-result` candidateはresult acceptanceを進めない意図である。

```text
reject-reconciliation-result
  != state application
  != rollback
  != compensation
  != retry
```

である。

R3BD Gateはaccept candidateだけを対象にする。

## 3. current Repository Evidence

current Repository-backed Phase 3には:

- R3AX ReconciliationCandidateIdentity
- R3BC ReconciliationResultAcceptanceCandidateIdentity
- AcceptedDecisionSinkRef / RevisionAuthorityRef
- ObservedDecisionRevision / ObservedStateFingerprint

は存在する。

しかしproduction authoritative-state application authority locusは確認していない。

したがってcurrent stateは:

```text
authoritative-state-application-authority-unresolved
```

を維持する。

Gate C / R7 Evidenceやderived R3 candidateをOperational state application authorityへ転用しない。

## 4. qualifying evidence

test-only authority evidenceは少なくとも:

- EvidenceID
- EvidenceRole
- RepositoryBacked
- AuthorityKind
- AuthorityDeclarationRef
- ReconciliationCandidateIdentity coverage
- ReconciliationResultAcceptanceCandidateIdentity coverage
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- ObservedDecisionRevision coverage
- ObservedStateFingerprint coverage
- BusinessAddress scope
- StateApplicationLocusRef
- RevisionAcceptanceLocusRef
- CurrentAuthorityProjectionLocusRef
- StateApplicationProvenanceRef
- DerivedOnly = false

を要求する。

AuthorityKindはtest-onlyに:

```text
operational-authoritative-state-application-authority
```

を要求する。

## 5. exact binding

qualifying evidenceはexact:

- R3AX ReconciliationCandidateIdentity
- R3BC ReconciliationResultAcceptanceCandidateIdentity
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- R3BC ObservedDecisionRevision
- R3BC ObservedStateFingerprint
- BusinessAddress scope

へbindingする。

same sink、same revision family、same fingerprint format、same result authorityだけでは補完しない。

## 6. three loci remain distinct

qualifying evidenceには:

- StateApplicationLocusRef
- RevisionAcceptanceLocusRef
- CurrentAuthorityProjectionLocusRef

を別々に要求する。

```text
state application locus known
  != revision accepted

revision acceptance locus known
  != CurrentAuthority projected

CurrentAuthority projection locus known
  != CurrentAuthority applied
```

である。

generic authority locusへ畳まない。

## 7. observed state does not create application authority

```text
ObservedDecisionRevision matches expected context
  != application authority

ObservedStateFingerprint present
  != application authority

result acceptance candidate = accept
  != application authority
```

である。

R3BC candidate自体はDerivedOnly evidenceとして扱い、authority sourceへしない。

## 8. multiple-qualified

複数authority evidenceがqualifyする場合:

```text
authoritative-state-application-authority-multiple-qualified-precedence-unresolved
```

とする。

scope specificity / revision / fingerprint / source path / corroboration countからwinnerを選ばない。

## 9. Gate result

test-only Gate resultは:

- StateApplicationAuthorityStatus
- QualifiedEvidenceIDs
- StateApplicationLocusRefs
- RevisionAcceptanceLocusRefs
- CurrentAuthorityProjectionLocusRefs
- AcceptedDecisionSinkRefs
- ObservedDecisionRevision
- ObservedStateFingerprint
- InsufficientEvidenceIDs
- AuthorityLimits

までを持てる。

これはapplied state recordではない。

## 10. 禁止するshortcut

R3BD Gate / evidence shapeは次を持たない。

- StateApplied / StateSynchronized / Reconciled
- RevisionAccepted / RevisionApplied
- AcceptedDecision / CurrentDecision
- CurrentAuthority / AppliedAuthority
- AuthoritativeState
- RuntimeOutcome / AuthorizationResult
- Retry / RetryEligible / RetryRequired
- Rollback / Compensation
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3bd_authoritative_state_application_authority_gate_pressure_test.go`

Pressureするcase:

1. current inspected derived candidateだけではGate unresolved
2. reject result candidateはapplication Gate対象にならない
3. complete synthetic authority evidenceはGateだけqualifyできる
4. wrong reconciliation/result candidate / sink / revision authority / revision / fingerprint / scopeはqualifyしない
5. missing application / revision-acceptance / CurrentAuthority-projection locusはqualifyしない
6. result accept / matching revision / fingerprint presenceからauthorityを推論しない
7. multiple-qualifiedはprecedence unresolved
8. input orderでGate resultを変えない
9. state application / CurrentAuthority / retry shortcutを追加しない

## 12. current conclusion

R3BDで固定するのは:

```text
R3BC accept result candidate
  + repository-backed authoritative-state application authority evidence
  -> authoritative-state application Evidence Gate

Gate qualified
  != state applied
  != revision accepted
  != CurrentAuthority projected
```

である。

production state applicator、decision-store mutation、authority projection mutation、transaction / retry / compensation engineは追加しない。

## 13. 次

次段ではsingle-qualified R3BD Gateでも自動適用せず:

```text
explicit state-application witness
  + exact result candidate
  + exact application authority
  + explicit apply / reject decision
  + provenance
  -> test-only state-application candidate

candidate
  != state applied
  != CurrentAuthority projected
```

をPressureする。
