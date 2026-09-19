# Actor Context R3AY — Reconciliation Execution Observation / Current-state Evidence Boundary

- Status: ISSUE-000053 / Phase 3 reconciliation execution observation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AX_RECONCILIATION_WITNESS_CANDIDATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AW_POST_WRITE_RECONCILIATION_AUTHORITY_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AT_WRITE_ATTEMPT_OBSERVATION_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AXではexplicit reconciliation witnessからtest-only reconciliation candidateまでを作り:

```text
reconciliation candidate
  != authoritative state synchronized
  != revision applied
  != current authority applied
```

を固定した。

R3AYではcandidateの後段をさらに分離する。

```text
reconciliation intent prepared
  != reconciliation execution observed
  != post-reconciliation current-state observed
  != authoritative state current
```

とする。

## 2. execution observationとcurrent-state observationを別axisにする

R3AYは少なくとも二つのraw Evidence planeを分ける。

### A. reconciliation execution observation

test-onlyに:

- ReconciliationCandidateIdentity
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- ReconciliationExecutionObservationRef
- ReportedExecutionOutcome

を持つ。

ReportedExecutionOutcomeはraw classとして:

- `reconciler-reported-applied`
- `reconciler-reported-not-applied`
- `reconciler-reported-conflict`
- `reconciler-reported-unknown`

だけをPressureする。

### B. post-reconciliation current-state observation

test-onlyに:

- ReconciliationCandidateIdentity
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- CurrentStateObservationRef
- ObservedDecisionRevision
- ObservedStateFingerprint

を持つ。

これは「現在観測したstate」のEvidenceであり、R3AX candidateのapply結果ではない。

## 3. 二つの観測は独立

```text
execution observation present
  != current-state observation present

current-state observation present
  != execution observation present
```

とする。

一方だけでもraw Evidenceとして保持できる。

片方をもう片方から補完しない。

## 4. reported outcomeはauthoritative synchronizationではない

```text
reconciler-reported-applied
  != StateSynchronized
  != RevisionAccepted
  != CurrentAuthority applied

reconciler-reported-not-applied
  != rollback required
  != retry authorized

reconciler-reported-conflict
  != compensation required

reconciler-reported-unknown
  != failed
```

である。

raw reportはattributed observationに留める。

## 5. current revision一致もsynchronization acceptanceではない

current-state observationのObservedDecisionRevisionが:

```text
R3AX.PostWriteDecisionRevision
```

と一致しても:

```text
revision matches candidate context
  != reconciliation execution proven
  != revision accepted
  != Accepted Decision current
```

とする。

不一致でもretry / rollback / failureを推論しない。

ObservedStateFingerprintもopaqueなobservation fingerprintでありAccepted Decision fingerprintではない。

## 6. exact binding

execution / current-state observationはそれぞれexact current R3AX candidateへbindingする。

少なくとも:

- ReconciliationCandidateIdentity
- AcceptedDecisionSinkRef
- RevisionAuthorityRef

が一致しなければfail closedする。

execution reportのObservationRef、current-state ObservationRefを別々に必須にする。

同じsinkだけでは不足する。

## 7. accept candidateだけをexecution-observation入口にする

R3AX:

- `accept-post-write-state-reconciliation`
- `reject-post-write-state-reconciliation`

のうち、R3AYでexecution observationへ進めるのはaccept candidateだけとする。

```text
reject reconciliation candidate
  != execution attempt
  != rollback
  != compensation
  != retry
```

reject candidateについてexecution observationを生成しない。

## 8. duplicate / conflict

同じcandidateについて:

- exact duplicate raw observation -> normalize可
- same ObservationRef + different ReportedExecutionOutcome -> fail closed
- same CurrentStateObservationRef + different revision / fingerprint -> fail closed

last writer / latest revision / majorityでwinnerを選ばない。

execution observationとcurrent-state observationの間にprecedenceを作らない。

## 9. current Repository Evidence

current Repository-backed R3AXはReference-onlyのcandidate boundaryであり、production reconciliation executor / actual post-reconciliation state observerをauthoritative Evidenceとして確認していない。

したがってR3AYのapplied / conflict / state-observed caseはsynthetic pressureである。

Gate C ReadyやFirst Web Target EvidenceをOperational Accepted Decision reconciliation executorへ転用しない。

## 10. read-only result

test-only resultは:

- ReconciliationCandidateIdentity
- ExecutionObservations
- CurrentStateObservations
- AuthorityLimits

までを持てる。

次は持たない。

- StateSynchronized / Reconciled
- RevisionAccepted / RevisionApplied
- AcceptedDecision / CurrentDecision
- CurrentAuthority / AppliedAuthority
- AcceptedWriteOutcome
- Retry / RetryEligible / RetryRequired
- Rollback / Compensation
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ay_reconciliation_execution_observation_pressure_test.go`

Pressureするcase:

1. R3AX accept candidateだけではexecution/current-state observationをfabricateしない
2. exact execution witnessをraw reported outcomeとして保持する
3. exact current-state witnessをrevision/fingerprint observationとして保持する
4. execution-only / current-state-onlyを独立に保持する
5. reported-appliedからStateSynchronizedを推論しない
6. candidate revision一致からRevisionAcceptedを推論しない
7. revision不一致からretry / rollbackを推論しない
8. reject candidateはexecution-observation入口にならない
9. wrong candidate / sink / revision authorityはfail closed
10. exact duplicate normalize、same-ref conflictはfail closed
11. repeated build / input orderでnormalized raw Evidenceを変えない
12. synchronization / current authority / execution shortcutを追加しない

## 12. current conclusion

R3AYで固定するのは:

```text
R3AX reconciliation candidate
  + attributed execution observation
  + attributed current-state observation
  -> raw post-candidate Evidence

raw post-candidate Evidence
  != authoritative synchronization
  != Accepted Decision current
  != retry / rollback authority
```

である。

production reconciliation executor、decision-store mutation、state observer service、authority projection mutation、retry / rollback engineは追加しない。

## 13. 次

次段ではraw execution/current-state Evidenceが揃っても:

```text
raw reconciliation Evidence
  != reconciliation outcome accepted
  != current authoritative state accepted
```

を維持し、post-reconciliation Evidence acceptance authorityを別GateとしてPressureする。
