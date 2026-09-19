# Actor Context R3BA — Explicit Post-reconciliation Evidence Acceptance Witness / Candidate Boundary

- Status: ISSUE-000053 / Phase 3 post-reconciliation evidence acceptance witness pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AZ_RECONCILIATION_EVIDENCE_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AY_RECONCILIATION_EXECUTION_OBSERVATION_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AV_OUTCOME_ACCEPTANCE_WITNESS_CANDIDATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AZではexecution Evidence / current-state Evidenceそれぞれについてacceptance authority Gateを分離した。

R3BAではsingle-qualified Gateでも自動accept/rejectせず、explicit witnessが存在する場合だけtest-only Evidence acceptance candidateを作る。

```text
single-qualified Evidence acceptance authority
  + exact R3AY raw Evidence
  + explicit accept / reject witness
  + acceptance provenance
  -> test-only Evidence acceptance candidate

candidate
  != authoritative state synchronized
  != Accepted Decision current
  != retry / rollback authority
```

## 2. Evidence planeを明示する

test-only witness / candidateは:

- `execution`
- `current-state`

のEvidencePlaneを明示する。

execution witnessをcurrent-state Gateへ適用せず、current-state witnessをexecution Gateへ適用しない。

```text
same reconciliation candidate
  != same Evidence plane
```

である。

## 3. execution Evidence acceptance witness

test-only witnessは少なくとも:

- EvidencePlane = execution
- ReconciliationCandidateIdentity
- AcceptanceAuthorityEvidenceID
- ReconciliationExecutionObservationRef
- ReportedExecutionOutcome
- Decision
- EvidenceAcceptanceProvenanceRef

を持つ。

Decisionはtest-onlyに:

- `accept-reconciliation-evidence`
- `reject-reconciliation-evidence`

だけをPressureする。

## 4. current-state Evidence acceptance witness

current-state witnessは少なくとも:

- EvidencePlane = current-state
- ReconciliationCandidateIdentity
- AcceptanceAuthorityEvidenceID
- CurrentStateObservationRef
- ObservedDecisionRevision
- ObservedStateFingerprint
- RevisionAuthorityRef
- Decision
- EvidenceAcceptanceProvenanceRef

を持つ。

execution planeのraw outcome語彙を補完しない。

## 5. exact binding

execution candidate化には:

- single-qualified execution R3AZ Gate
- exact authority EvidenceID
- exact R3AY execution ObservationRef
- exact ReportedExecutionOutcome
- exact ReconciliationCandidateIdentity

を要求する。

current-state candidate化には:

- single-qualified current-state R3AZ Gate
- exact authority EvidenceID
- exact CurrentStateObservationRef
- exact ObservedDecisionRevision
- exact ObservedStateFingerprint
- exact RevisionAuthorityRef
- exact ReconciliationCandidateIdentity

を要求する。

wrong plane / authority / observation / raw value / revision bindingはfail closedする。

## 6. no witnessでauto-acceptしない

single-qualified Gateでもwitnessがなければcandidateを作らない。

```text
Gate qualified
  != Evidence acceptance intent exists
  != Evidence accepted
```

である。

## 7. accept / rejectをraw Evidenceから推測しない

```text
reconciler-reported-applied
  != accept Evidence

reconciler-reported-conflict
  != reject Evidence

matching current revision
  != accept current-state Evidence

divergent current revision
  != reject current-state Evidence
```

Decisionはexplicit witnessだけから得る。

## 8. reject candidateの意味

```text
reject Evidence candidate
  != retry
  != rollback
  != compensation
  != write failed
  != reconciliation failed
```

である。

rejectはそのraw Evidenceをauthoritative acceptanceへ進めない意図だけを表す。

## 9. candidate

test-only candidateは少なくとも:

- EvidencePlane
- ReconciliationCandidateIdentity
- AcceptanceAuthorityEvidenceID
- RawEvidenceRef
- RawEvidenceClass
- ObservedDecisionRevision（current-stateのみ）
- ObservedStateFingerprint（current-stateのみ）
- RevisionAuthorityRef（current-stateのみ）
- Decision
- EvidenceAcceptanceProvenanceRef
- EvidenceAcceptanceCandidateIdentity
- AuthorityLimits

までを持てる。

ExecutionEvidenceAccepted / CurrentStateEvidenceAcceptedは持たない。

## 10. both-plane acceptでもstate synchronizationしない

execution / current-state両方にaccept candidateが存在しても:

```text
execution Evidence accept candidate
  + current-state Evidence accept candidate
  != StateSynchronized
  != RevisionAccepted
  != CurrentAuthority applied
```

とする。

plane間のjoin / final acceptanceは次段の別boundaryである。

## 11. duplicate / conflict

same plane / same raw Evidence / same authorityについて:

- exact duplicate witness -> normalize可
- Decisionが異なる -> fail closed
- EvidenceAcceptanceProvenanceRefが異なる -> fail closed
- current-state revision / fingerprint bindingが異なる -> fail closed

last writer / majority / latest observationでwinnerを選ばない。

## 12. current Repository Evidence

current Repository-backed R3AZではproduction acceptance authorityはunresolvedである。

したがってR3BAのcandidate caseはsynthetic single-qualified R3AZ GateによるPressureであり、current production acceptanceを意味しない。

## 13. 禁止するshortcut

R3BA witness / candidate shapeは次を持たない。

- EvidenceAccepted / AcceptanceApplied
- StateSynchronized / Reconciled
- RevisionAccepted / RevisionApplied
- AcceptedDecision / CurrentDecision
- CurrentAuthority / AppliedAuthority
- Retry / RetryEligible / RetryRequired
- Rollback / Compensation
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 14. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ba_reconciliation_evidence_acceptance_witness_candidate_pressure_test.go`

Pressureするcase:

1. unresolved R3AZ Gateではcandidateを作れない
2. single-qualified Gateでもno witnessではcandidateを作らない
3. explicit execution accept / reject witnessをcandidate化する
4. explicit current-state accept / reject witnessをcandidate化する
5. wrong plane / authority / observation / raw value / revision / fingerprintはfail closed
6. raw outcome / revision relationからDecisionを推測しない
7. both-plane accept candidateでもStateSynchronizedへ進めない
8. multiple-qualified planeをwitnessで迂回しない
9. exact duplicate normalize、decision/provenance conflictはfail closed
10. repeated build / input orderでcandidate identityを変えない
11. acceptance application / synchronization / retry shortcutを追加しない

## 15. current conclusion

R3BAで固定するのは:

```text
single-qualified R3AZ plane Gate
  + exact explicit Evidence acceptance witness
  -> test-only plane-local acceptance candidate

candidate
  != Evidence accepted authoritatively
  != state synchronized
  != current authority applied
```

である。

production evidence acceptance workflow、state synchronization service、decision-store mutation、retry / rollback engineは追加しない。

## 16. 次

次段ではexecution/current-state両candidateが揃っても自動joinせず:

```text
plane-local Evidence acceptance candidates
  != authoritative reconciliation result

explicit join / current-state acceptance authority
  -> separate Evidence Gate
```

をPressureする。
