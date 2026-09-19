# Actor Context R3BC — Explicit Reconciliation Result Acceptance Witness / Candidate Boundary

- Status: ISSUE-000053 / Phase 3 reconciliation-result acceptance witness pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3BB_RECONCILIATION_RESULT_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3BA_RECONCILIATION_EVIDENCE_ACCEPTANCE_WITNESS_CANDIDATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AX_RECONCILIATION_WITNESS_CANDIDATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3BBではexecution/current-state plane-local accept candidatesをjoinするためのresult/current-state acceptance authority Gateを追加した。

R3BCではsingle-qualified R3BB Gateでも自動accept/rejectせず:

```text
single-qualified R3BB Gate
  + exact plane-local candidate identities
  + exact qualified result authority
  + explicit accept / reject witness
  + acceptance provenance
  -> test-only reconciliation-result acceptance candidate

candidate
  != StateSynchronized
  != Accepted Decision current
  != CurrentAuthority applied
```

をPressureする。

## 2. explicit witness

test-only witnessは少なくとも:

- ReconciliationCandidateIdentity
- ResultAcceptanceAuthorityEvidenceID
- ExecutionEvidenceAcceptanceCandidateIdentity
- CurrentStateEvidenceAcceptanceCandidateIdentity
- ReconciliationResultDecisionLocusRef
- CurrentStateAcceptanceLocusRef
- Decision
- ResultAcceptanceProvenanceRef

を持つ。

Decisionはtest-onlyに:

- `accept-reconciliation-result`
- `reject-reconciliation-result`

だけをPressureする。

## 3. exact binding

candidate化には:

- R3BB Gate status = single-qualified
- exact qualified EvidenceID
- exact R3AX ReconciliationCandidateIdentity
- exact R3BA execution acceptance candidate identity
- exact R3BA current-state acceptance candidate identity
- exact ReconciliationResultDecisionLocusRef
- exact CurrentStateAcceptanceLocusRef

を要求する。

Gateが保持していないcandidate / locusへwitnessが差し替わる場合はfail closedする。

## 4. Gate qualifiedだけではcandidateを作らない

```text
R3BB Gate qualified
  != result acceptance intent exists
  != reconciliation result accepted
```

である。

witnessがない場合はcandidateなしを正常系とする。

## 5. Decisionをcandidate内容から推測しない

```text
execution/current-state candidates both accepted
  != accept reconciliation result

matching revision
  != accept reconciliation result

reported-applied
  != accept reconciliation result
```

Decisionはexplicit witnessだけから取得する。

## 6. reject candidate

```text
reject reconciliation result candidate
  != retry
  != rollback
  != compensation
  != write failure
  != reconciliation failure
```

である。

rejectはresult acceptanceを進めない意図だけを表す。

## 7. candidate

test-only candidateは少なくとも:

- ReconciliationCandidateIdentity
- ResultAcceptanceAuthorityEvidenceID
- ExecutionEvidenceAcceptanceCandidateIdentity
- CurrentStateEvidenceAcceptanceCandidateIdentity
- ReconciliationResultDecisionLocusRef
- CurrentStateAcceptanceLocusRef
- ObservedDecisionRevision
- ObservedStateFingerprint
- Decision
- ResultAcceptanceProvenanceRef
- ReconciliationResultAcceptanceCandidateIdentity
- AuthorityLimits

を持てる。

ObservedDecisionRevision / ObservedStateFingerprintはcurrent-state plane candidateからtraceとして保持するが、Accepted Decision / synchronized stateではない。

## 8. duplicate / conflict

same exact result authority + same exact plane candidatesについて:

- exact duplicate witness -> normalize可
- Decision conflict -> fail closed
- result/current-state locus conflict -> fail closed
- acceptance provenance conflict -> fail closed

last writer / latest revision / majorityでwinnerを選ばない。

## 9. multiple-qualified Gateを迂回しない

R3BB Gateがmultiple-qualifiedの場合、witnessが特定EvidenceIDを示してもcandidateを作らない。

```text
witness selected one authority
  != precedence authority exists
```

である。

## 10. current Repository Evidence

current Repository-backed R3BB authorityはunresolvedであり、production reconciliation-result acceptance locusは確認していない。

したがってR3BC candidate caseはsynthetic single-qualified R3BB GateによるPressureである。

Gate C / R7A / R7B EvidenceをこのOperational result acceptance witnessへ流用しない。

## 11. 禁止するshortcut

R3BC witness / candidate shapeは次を持たない。

- ReconciliationResultAccepted / AcceptanceApplied
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

- `tools/reference-business-procedure-check/actor_context_r3bc_reconciliation_result_acceptance_witness_candidate_pressure_test.go`

Pressureするcase:

1. unresolved R3BB Gateではcandidateを作れない
2. single-qualified Gateでもno witnessではcandidateなし
3. explicit accept / reject witnessをcandidate化できる
4. wrong authority / reconciliation candidate / execution candidate / current-state candidate / result locus / current-state locusはfail closed
5. Decision / provenance missingはfail closed
6. candidateからDecisionを推測しない
7. multiple-qualified Gateをwitnessで迂回しない
8. exact duplicate normalize、decision/locus/provenance conflictはfail closed
9. input order / repeated buildでcandidate identityを変えない
10. synchronization / current authority / retry shortcutを追加しない

## 13. current conclusion

R3BCで固定するのは:

```text
single-qualified R3BB Gate
  + explicit exact result acceptance witness
  -> test-only reconciliation-result acceptance candidate

candidate
  != authoritative result applied
  != state synchronized
  != current authority applied
```

である。

production result acceptance workflow、state synchronization service、decision-store mutation、authority projection mutation、retry / rollback engineは追加しない。

## 14. 次

次段ではaccept candidateがあっても自動適用せず:

```text
reconciliation-result acceptance candidate
  + explicit state-application authority
  -> authoritative-state application Evidence Gate

Gate qualified
  != state applied
```

をPressureする。
