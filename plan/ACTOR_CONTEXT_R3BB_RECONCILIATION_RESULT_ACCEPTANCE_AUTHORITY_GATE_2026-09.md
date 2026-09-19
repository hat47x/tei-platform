# Actor Context R3BB — Reconciliation Result / Current-state Join Authority Evidence Gate

- Status: ISSUE-000053 / Phase 3 reconciliation-result join-authority pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3BA_RECONCILIATION_EVIDENCE_ACCEPTANCE_WITNESS_CANDIDATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AZ_RECONCILIATION_EVIDENCE_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AX_RECONCILIATION_WITNESS_CANDIDATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3BAではexecution / current-stateのraw Evidenceを、それぞれexplicit witnessによるplane-local acceptance candidateまで進めた。

両planeにaccept candidateが存在しても:

```text
execution Evidence accept candidate
  + current-state Evidence accept candidate
  != authoritative reconciliation result
  != StateSynchronized
  != CurrentAuthority applied
```

である。

R3BBでは、両candidateをauthoritative reconciliation resultへjoinする前に、Repository-backedなresult/current-state acceptance authorityが存在するかをEvidence GateとしてPressureする。

## 2. entry condition

R3BBへ入れるのは同じR3AX `ReconciliationCandidateIdentity` に属する:

- execution plane candidate
- current-state plane candidate

がそれぞれexactly one存在し、両方のDecisionが:

```text
accept-reconciliation-evidence
```

である場合だけとする。

片方がmissing / rejectならGateを組まない。

```text
reject plane-local Evidence
  != reconciliation failed
  != retry / rollback entry
```

である。

## 3. qualifying authority evidence

test-only reconciliation-result acceptance authority evidenceは少なくとも:

- EvidenceID
- EvidenceRole
- RepositoryBacked
- AuthorityKind
- AuthorityDeclarationRef
- exact ReconciliationCandidateIdentity coverage
- exact ExecutionEvidenceAcceptanceCandidateIdentity coverage
- exact CurrentStateEvidenceAcceptanceCandidateIdentity coverage
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- BusinessAddress scope
- ReconciliationResultDecisionLocusRef
- CurrentStateAcceptanceLocusRef
- ResultAcceptanceProvenanceRef
- DerivedOnly

を持つ。

AuthorityKindはtest-onlyに:

```text
operational-reconciliation-result-acceptance-authority
```

を要求する。

## 4. exact binding

qualifying evidenceはexact:

- R3AX ReconciliationCandidateIdentity
- R3BA execution EvidenceAcceptanceCandidateIdentity
- R3BA current-state EvidenceAcceptanceCandidateIdentity
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- BusinessAddress scope

へbindingする。

同じraw ObservationRef / same revision / same sinkだけでは補完しない。

plane-local acceptance provenanceの類似からresult acceptance authorityを推測しない。

## 5. current Repository Evidence

current Repository-backed chainには:

- R3AX reconciliation candidate
- R3BA execution acceptance candidate shape
- R3BA current-state acceptance candidate shape

はある。

しかしproduction authoritative reconciliation-result/current-state acceptance locusを確認していない。

したがってcurrent inspected evidenceでは:

```text
reconciliation-result-acceptance-authority-unresolved
```

を維持する。

R3AW reconciliation authority、R3AZ plane-local Evidence acceptance authority、Gate C / First Web Target EvidenceをR3BB authorityへ暗黙転用しない。

## 6. plane-local accept candidateはauthority sourceではない

R3BA candidatesはderived candidateである。

```text
execution Evidence accept candidate
  != result acceptance authority

current-state Evidence accept candidate
  != current-state authoritative acceptance locus
```

である。

candidate自身をRepositoryBacked authorityへ昇格させない。

## 7. revision / fingerprint / raw outcomeからjoin authorityを推測しない

```text
current-state revision matches R3AX post-write revision
  != result acceptance authority

execution reported-applied
  != result acceptance authority

execution + current-state both accepted as candidates
  != result acceptance authority
```

とする。

## 8. two loci remain explicit

qualifying evidenceは:

- `ReconciliationResultDecisionLocusRef`
- `CurrentStateAcceptanceLocusRef`

を別々に保持する。

一つのgeneric `AuthorityLocus` に畳まない。

```text
result decision authority
  != current-state acceptance authority
```

という内部境界を維持する。

ただしR3BBでは両locusを一つのRepository-backed authority evidenceが明示しているpressure caseを扱う。

## 9. multiple-qualified

複数authority evidenceがqualifyした場合:

```text
reconciliation-result-acceptance-authority-multiple-qualified-precedence-unresolved
```

とする。

scope specificity / latest revision / source path / candidate provenance / corroboration countからwinnerを選ばない。

## 10. Gate result

test-only Gate resultは:

- ReconciliationResultAcceptanceAuthorityStatus
- QualifiedEvidenceIDs
- ReconciliationResultDecisionLocusRefs
- CurrentStateAcceptanceLocusRefs
- ExecutionEvidenceAcceptanceCandidateIdentity
- CurrentStateEvidenceAcceptanceCandidateIdentity
- InsufficientEvidenceIDs
- AuthorityLimits

までを持てる。

これはauthoritative resultではない。

## 11. 禁止するshortcut

R3BB Gate / evidence shapeは次を持たない。

- ReconciliationResultAccepted
- CurrentStateAccepted
- StateSynchronized / Reconciled
- RevisionAccepted / RevisionApplied
- AcceptedDecision / CurrentDecision
- CurrentAuthority / AppliedAuthority
- RuntimeOutcome / AuthorizationResult
- Retry / RetryEligible / RetryRequired
- Rollback / Compensation
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 12. test-only source

- `tools/reference-business-procedure-check/actor_context_r3bb_reconciliation_result_acceptance_authority_gate_pressure_test.go`

Pressureするcase:

1. one-plane candidateだけではGate entryできない
2. reject candidateをresult acceptance entryへ変換しない
3. current inspected derived candidatesだけではauthority unresolved
4. complete synthetic result/current-state authority evidenceはGateだけqualifyできる
5. wrong reconciliation candidate / execution candidate / current-state candidate / sink / revision authority / scopeはqualifyしない
6. missing declaration / result locus / current-state locus / provenanceはqualifyしない
7. revision/fingerprint/raw outcome/candidate countからauthorityを推論しない
8. multiple-qualifiedはprecedence unresolved
9. input orderでGate resultを変えない
10. synchronized state / current authority / retry shortcutを追加しない

## 13. current conclusion

R3BBで固定するのは:

```text
plane-local Evidence acceptance candidates
  + explicit repository-backed result/current-state acceptance authority
  -> reconciliation-result acceptance Evidence Gate

Gate qualified
  != authoritative reconciliation result accepted
  != state synchronized
  != current authority applied
```

である。

production state synchronization service、decision-store current-state acceptance、authority projection mutation、retry / rollback engineは追加しない。

## 14. 次

次段ではsingle-qualified R3BB Gateでも自動acceptせず:

```text
explicit reconciliation-result acceptance witness
  + exact R3BA plane candidates
  + exact R3BB authority
  + explicit accept / reject
  + provenance
  -> test-only reconciliation-result acceptance candidate

candidate
  != StateSynchronized
  != CurrentAuthority applied
```

をPressureする。
