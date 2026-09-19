# Actor Context R3AT — Write-attempt Observation / Accepted Outcome Separation Boundary

- Status: ISSUE-000053 / Phase 3 write-attempt observation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AS_WRITE_ATTEMPT_CANDIDATE_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AR_APPLICATION_AUTHORITY_REVISION_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AQ_ACCEPTED_DECISION_COMMIT_CANDIDATE_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ASではexplicit witnessからtest-only write-attempt candidateを作ったが、writerを実行していない。

R3ATではwriterから返ったと仮定するtest-only observationを受けても:

```text
write attempt observation
  != accepted write outcome
  != Accepted Decision applied
```

を維持する。

R3AT自身はwriterを呼ばない。

## 2. observationはattempt identityへexact bindingする

test-only observationは最低限:

- WriteAttemptCandidateIdentity
- AcceptedDecisionSinkRef
- IdempotencyIdentity
- WriterObservationRef
- ReportedOutcome
- PostWriteRevisionObservationRef
- PostWriteObservedDecisionRevision

を持つ。

WriteAttemptCandidateIdentity / sink / IdempotencyIdentityがR3AS candidateとexactに一致しなければfail closedする。

## 3. ReportedOutcomeはraw observation class

test-only raw classとして:

- writer-reported-applied
- writer-reported-not-applied
- writer-reported-conflict
- writer-reported-unknown

をPressureする。

これはwriter側のreported observationを識別する語彙であり、Accepted Decision write outcomeのauthority enumではない。

```text
writer-reported-applied
  != Accepted Decision committed

writer-reported-conflict
  != retry authorized

writer-reported-not-applied
  != rejected authority

writer-reported-unknown
  != failed
```

である。

## 4. post-write revision observationは別axis

PostWriteObservedDecisionRevisionをReportedOutcomeと分離して保持する。

```text
reported applied
  + revision changed
  != authoritative commit accepted

reported applied
  + revision unchanged
  != automatically invalid observation

reported conflict
  + revision changed
  != conflict resolved
```

とする。

R3ATではoutcome classとrevision changeの整合ルールをproduction contractとして固定しない。

## 5. attribution必須

WriterObservationRefとPostWriteRevisionObservationRefを必須にする。

attributionのないraw outcome / revisionは受け取らない。

ただし:

```text
attributed observation
  != trusted authority
  != accepted outcome
```

である。

sender identity / transport / signature validationは追加しない。

## 6. idempotency identity echo

observationのIdempotencyIdentityはR3AS candidateとexact一致を要求する。

```text
same idempotency identity observed
  != dedupe succeeded
  != exactly-once proven
```

異なるidentityをsame attemptのretryとして自動集約しない。

## 7. duplicate / conflicting observation

same WriteAttemptCandidateIdentityについて:

- exact duplicate observation -> normalize可
- ReportedOutcomeが異なる -> conflictとしてfail closed
- post-write revisionが異なる -> conflictとしてfail closed
- attribution refが異なる -> aggregation authority未解決としてfail closed

とする。

last observation wins / majority / newest revision winsを導入しない。

## 8. observation result

test-only normalized observationは:

- WriteAttemptCandidateIdentity
- AcceptedDecisionSinkRef
- IdempotencyIdentity
- WriterObservationRef
- ReportedOutcome
- PostWriteRevisionObservationRef
- PostWriteObservedDecisionRevision
- ObservationIdentity
- AuthorityLimits

までを持てる。

これはaccepted outcome recordではない。

## 9. R3ATでしないこと

R3ATは:

- write実行
- outcome acceptance
- post-write revisionのauthoritative acceptance
- Accepted Decision materialization
- R3Y authority rewrite
- retry判断

を行わない。

## 10. 禁止するshortcut

R3AT shapeは次を持たない。

- AcceptedWriteOutcome
- OutcomeAccepted
- CommitAccepted
- Committed / CommitSucceeded
- AcceptedDecision
- NewDecisionRevision
- CurrentAuthority
- AppliedAuthority
- RuntimeOutcome / AuthorizationResult
- Retry / RetryRequired / Reauthorize
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3at_write_attempt_observation_pressure_test.go`

Pressureするcase:

1. exact R3AS candidateへraw observationをbindingできる
2. wrong attempt identity / sink / idempotency identityはfail closed
3. missing writer/post-revision attributionはfail closed
4. four raw ReportedOutcome classesをobservationとして保持する
5. reported-appliedをAccepted Decision committedへ昇格しない
6. post-write revision変化をoutcome acceptanceへ変換しない
7. exact duplicate normalize
8. conflicting outcome / revision / provenanceはfail closed
9. repeated build / input orderでobservation identityを変えない
10. accepted outcome / authority application / retry shortcutを追加しない

## 12. current conclusion

R3ATで固定するのは:

```text
exact write-attempt candidate
  + attributed writer observation
  + idempotency identity echo
  + attributed post-write revision observation
  -> test-only raw attempt observation

raw attempt observation
  != accepted write outcome
  != Accepted Decision committed
  != current authority applied
```

である。

production writer、transport、signature validation、outcome acceptance engine、decision store mutation、retry schedulerは追加しない。

## 13. 次

次段ではraw observationをaccepted write outcomeへ進める前に:

```text
raw writer observation
  + explicit outcome-acceptance authority
  -> outcome acceptance Gate

Gate qualified
  != Accepted Decision applied
```

をPressureする。
