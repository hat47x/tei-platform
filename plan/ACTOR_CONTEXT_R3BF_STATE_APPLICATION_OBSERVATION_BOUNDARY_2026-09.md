# Actor Context R3BF — State-application Execution / Authority-projection Observation Boundary

- Status: ISSUE-000053 / Phase 3 state-application observation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3BE_STATE_APPLICATION_WITNESS_CANDIDATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3BD_AUTHORITATIVE_STATE_APPLICATION_AUTHORITY_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AY_RECONCILIATION_EXECUTION_OBSERVATION_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3BEではexplicit witnessからtest-only state-application candidateを作った。

しかし:

```text
state-application candidate
  != state application executed
  != revision accepted
  != CurrentAuthority projected
```

である。

R3BFではcandidate後段のraw Evidenceを二つの独立planeへ分ける。

```text
A. state-application execution observation
B. authority-projection/current-state observation
```

どちらもauthoritative resultではない。

## 2. apply candidateだけをobservation入口にする

R3BE:

- `apply-authoritative-state`
- `reject-authoritative-state-application`

のうち、R3BFでexecution / projection observationへ進めるのはapply candidateだけとする。

```text
reject state-application candidate
  != execution attempt
  != rollback
  != retry
  != compensation
```

である。

## 3. execution observation plane

test-only execution witnessは最低限:

- StateApplicationCandidateIdentity
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- StateApplicationExecutionObservationRef
- ReportedApplicationOutcome

を持つ。

ReportedApplicationOutcomeはraw classとして:

- `state-applicator-reported-applied`
- `state-applicator-reported-not-applied`
- `state-applicator-reported-conflict`
- `state-applicator-reported-unknown`

だけをPressureする。

```text
state-applicator-reported-applied
  != authoritative state applied
  != revision accepted
  != CurrentAuthority projected
```

である。

## 4. authority-projection observation plane

test-only authority-projection/current-state witnessは最低限:

- StateApplicationCandidateIdentity
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- CurrentAuthorityProjectionLocusRef
- CurrentAuthorityObservationRef
- ObservedDecisionRevision
- ObservedStateFingerprint
- ObservedAuthorityProjectionFingerprint

を持つ。

ObservedAuthorityProjectionFingerprintはopaqueなobservation fingerprintであり:

```text
observed projection fingerprint
  != CurrentAuthority accepted
  != CurrentAuthority current
```

である。

## 5. two planes remain independent

```text
execution observation present
  != authority-projection observation present

authority-projection observation present
  != execution observation present
```

である。

片方から他方を補完しない。

## 6. exact binding

両planeはexact current R3BE candidateへbindingする。

最低限:

- StateApplicationCandidateIdentity
- AcceptedDecisionSinkRef
- RevisionAuthorityRef

を一致させる。

authority-projection planeではさらに:

- CurrentAuthorityProjectionLocusRef

をexact matchさせる。

same sink / same revision familyだけでは不足する。

## 7. revision / fingerprint relation does not decide authority

```text
ObservedDecisionRevision == candidate ObservedDecisionRevision
  != RevisionAccepted

ObservedStateFingerprint == candidate ObservedStateFingerprint
  != StateApplied

ObservedAuthorityProjectionFingerprint present
  != CurrentAuthority projected
```

不一致でもretry / rollback / failureを推論しない。

## 8. duplicate / conflict

same candidateについて:

- exact duplicate raw observation -> normalize可
- same execution ObservationRef + different raw outcome -> fail closed
- same CurrentAuthorityObservationRef + different revision/state/projection fingerprint -> fail closed

last writer / newest observation / majorityでwinnerを選ばない。

## 9. current Repository Evidence

current Repository-backed R3BEはReference-only candidate boundaryであり、production state applicator / authority projection observerをauthoritative Evidenceとして確認していない。

したがってR3BF positive caseはsynthetic pressureである。

## 10. read-only result

test-only resultは:

- StateApplicationCandidateIdentity
- ExecutionObservations
- AuthorityProjectionObservations
- AuthorityLimits

までを持てる。

次は持たない。

- StateApplied / StateSynchronized
- RevisionAccepted / RevisionApplied
- AcceptedDecision / CurrentDecision
- CurrentAuthority / AppliedAuthority
- CurrentAuthorityAccepted
- RuntimeOutcome / AuthorizationResult
- Retry / RetryEligible / RetryRequired
- Rollback / Compensation
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3bf_state_application_observation_pressure_test.go`

Pressureするcase:

1. R3BE apply candidateだけではobservationをfabricateしない
2. reject candidateはobservation入口にならない
3. exact execution witnessをraw outcomeとして保持する
4. exact authority-projection witnessをraw observationとして保持する
5. execution-only / projection-onlyを独立保持する
6. wrong candidate / sink / revision authority / projection locusはfail closed
7. reported-appliedからStateAppliedを推論しない
8. matching revision/state fingerprintからRevisionAcceptedを推論しない
9. projection fingerprintからCurrentAuthorityを推論しない
10. exact duplicate normalize / same-ref conflictはfail closed
11. repeated build / input orderでnormalized Evidenceを変えない
12. state application / CurrentAuthority / retry shortcutを追加しない

## 12. current conclusion

R3BFで固定するのは:

```text
R3BE state-application candidate
  + attributed execution observation
  + attributed authority-projection/current-state observation
  -> raw post-application Evidence

raw post-application Evidence
  != authoritative state applied
  != revision accepted
  != CurrentAuthority current
```

である。

production state applicator、authority projection service、decision-store mutation、retry / rollback engineは追加しない。

## 13. 次

次段ではraw post-application Evidenceをauthoritative state / CurrentAuthority evidenceとして受理する前に:

```text
raw application Evidence
  + explicit post-application Evidence acceptance authority
  -> acceptance-authority Gate

Gate qualified
  != state applied
  != CurrentAuthority current
```

をPressureする。
