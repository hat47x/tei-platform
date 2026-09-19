# Actor Context R3AZ — Post-reconciliation Evidence Acceptance Authority Gate

- Status: ISSUE-000053 / Phase 3 post-reconciliation evidence-acceptance authority pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AY_RECONCILIATION_EXECUTION_OBSERVATION_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AU_WRITE_OUTCOME_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AX_RECONCILIATION_WITNESS_CANDIDATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AYではpost-reconciliationのraw Evidenceを:

```text
execution observation
current-state observation
```

の二平面に分離した。

R3AZではそれぞれをauthoritative evidenceとして受理するためのauthorityを別GateとしてPressureする。

```text
raw reconciliation execution Evidence
  != execution Evidence accepted

raw post-reconciliation current-state Evidence
  != current-state Evidence accepted

acceptance authority qualified
  != Evidence accepted
  != authoritative state current
```

である。

## 2. Gateを二軸に分ける

test-only Gate resultは少なくとも:

```text
ExecutionEvidenceAcceptanceAuthorityStatus
CurrentStateEvidenceAcceptanceAuthorityStatus
```

を別々に持つ。

execution側のauthorityがqualifyしてもcurrent-state側をqualifyしない。

current-state側のauthorityがqualifyしてもexecution側をqualifyしない。

```text
one plane qualified
  != other plane qualified
```

である。

## 3. execution Evidence acceptance authority

execution planeのqualifying evidenceは少なくとも:

- EvidenceID
- RepositoryBacked
- explicit authority declaration
- AuthorityKind = `operational-reconciliation-execution-evidence-acceptance-authority`
- exact ReconciliationCandidateIdentity coverage
- exact ReconciliationExecutionObservationRef coverage
- exact ReportedExecutionOutcome coverage
- AcceptedDecisionSinkRef
- BusinessAddress scope
- ExecutionEvidenceDecisionLocusRef
- EvidenceAcceptanceProvenanceRef
- DerivedOnly = false

を要求する。

raw execution observation自身はDerivedOnlyでありauthorityにならない。

## 4. current-state Evidence acceptance authority

current-state planeのqualifying evidenceは少なくとも:

- EvidenceID
- RepositoryBacked
- explicit authority declaration
- AuthorityKind = `operational-reconciliation-current-state-evidence-acceptance-authority`
- exact ReconciliationCandidateIdentity coverage
- exact CurrentStateObservationRef coverage
- exact ObservedDecisionRevision coverage
- exact ObservedStateFingerprint coverage
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- BusinessAddress scope
- CurrentStateEvidenceDecisionLocusRef
- EvidenceAcceptanceProvenanceRef
- DerivedOnly = false

を要求する。

raw current-state observation自身はDerivedOnlyでありauthorityにならない。

## 5. current Repository Evidence

current Repository-backed Phase 3では:

- R3AX reconciliation candidate
- R3AY raw execution observation model
- R3AY raw current-state observation model

はある。

しかしproduction post-reconciliation Evidence acceptance authority locusを確認していない。

したがってcurrent stateは:

```text
execution evidence acceptance authority
  = unresolved

current-state evidence acceptance authority
  = unresolved
```

を維持する。

Gate C Ready / First Web Target Visual・Semantic・Functional EvidenceはこのOperational acceptance authorityへ転用しない。

## 6. raw Evidenceはauthorityではない

```text
reconciler-reported-applied
  != accepted execution outcome

ObservedDecisionRevision matches candidate
  != accepted current state

ObservedStateFingerprint present
  != Accepted Decision fingerprint
```

である。

R3AY raw observation refをAuthorityDeclarationRefへ流用しない。

## 7. exact binding

execution authority evidenceはexact:

- ReconciliationCandidateIdentity
- ReconciliationExecutionObservationRef
- ReportedExecutionOutcome
- AcceptedDecisionSinkRef
- BusinessAddress scope

へbindingする。

current-state authority evidenceはexact:

- ReconciliationCandidateIdentity
- CurrentStateObservationRef
- ObservedDecisionRevision
- ObservedStateFingerprint
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- BusinessAddress scope

へbindingする。

同じsink / same revision family / same fingerprint formatだけでは補完しない。

## 8. revision relation / raw outcomeはauthorityを生成しない

```text
reported-applied
  != execution Evidence acceptance authority

reported-conflict
  != retry authority

current revision == candidate post-write revision
  != current-state Evidence acceptance authority

current revision != candidate post-write revision
  != current-state Evidence rejected
  != rollback / retry authority
```

である。

## 9. multiple-qualified

各planeで複数authority evidenceがqualifyする場合、それぞれ:

```text
...multiple-qualified-precedence-unresolved
```

とする。

execution側のmultiple-qualifiedをcurrent-state側のsingle-qualifiedで解決しない。

scope specificity / source path / revision / fingerprint / outcome class / corroboration countからwinnerを選ばない。

## 10. Gate result

test-only Gate resultは:

- ExecutionEvidenceAcceptanceAuthorityStatus
- CurrentStateEvidenceAcceptanceAuthorityStatus
- QualifiedExecutionEvidenceIDs
- QualifiedCurrentStateEvidenceIDs
- ExecutionEvidenceDecisionLocusRefs
- CurrentStateEvidenceDecisionLocusRefs
- InsufficientEvidenceIDs
- AuthorityLimits

までを持てる。

これはaccepted Evidence recordではない。

## 11. 禁止するshortcut

R3AZ Gate / evidence shapeは次を持たない。

- ExecutionEvidenceAccepted / CurrentStateEvidenceAccepted
- AcceptedExecutionOutcome
- AcceptedCurrentState
- StateSynchronized / Reconciled
- RevisionAccepted / RevisionApplied
- AcceptedDecision / CurrentDecision
- CurrentAuthority / AppliedAuthority
- Retry / RetryEligible / RetryRequired
- Rollback / Compensation
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 12. test-only source

- `tools/reference-business-procedure-check/actor_context_r3az_reconciliation_evidence_acceptance_authority_gate_pressure_test.go`

Pressureするcase:

1. current inspected raw Evidenceだけでは両Gate unresolved
2. execution raw observation自身はexecution authorityにならない
3. current-state raw observation自身はcurrent-state authorityにならない
4. complete synthetic execution authorityはexecution Gateだけqualifyする
5. complete synthetic current-state authorityはcurrent-state Gateだけqualifyする
6. 両authorityが揃ってもGate qualificationまでに留める
7. wrong candidate / observation ref / outcome / revision / fingerprint / sink / revision authority / scopeはqualifyしない
8. missing declaration / decision locus / provenanceはqualifyしない
9. raw outcome / revision relationからauthorityを推論しない
10. planeごとのmultiple-qualifiedはprecedence unresolved
11. input orderでGate resultを変えない
12. accepted Evidence / synchronized state / retry shortcutを追加しない

## 13. current conclusion

R3AZで固定するのは:

```text
R3AY raw execution Evidence
  + explicit execution Evidence acceptance authority
  -> execution acceptance-authority Gate

R3AY raw current-state Evidence
  + explicit current-state Evidence acceptance authority
  -> current-state acceptance-authority Gate

Gate qualified
  != Evidence accepted
  != authoritative state synchronized
```

である。

production evidence registry、trust/signature validation、reconciliation acceptance workflow、state synchronization service、retry / rollback engineは追加しない。

## 14. 次

次段ではsingle-qualified Gateが存在しても自動acceptせず:

```text
explicit Evidence acceptance witness
  + exact R3AY observation
  + exact R3AZ acceptance authority
  + explicit accept / reject
  + provenance
  -> test-only Evidence acceptance candidate

candidate
  != state synchronized
  != current authority applied
```

をPressureする。
