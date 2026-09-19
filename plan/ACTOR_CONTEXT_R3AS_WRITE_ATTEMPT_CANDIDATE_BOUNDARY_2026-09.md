# Actor Context R3AS — Explicit Write-attempt Witness / Idempotency Identity Boundary

- Status: ISSUE-000053 / Phase 3 Accepted Decision write-attempt candidate pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AR_APPLICATION_AUTHORITY_REVISION_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AQ_ACCEPTED_DECISION_COMMIT_CANDIDATE_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AP_ACCEPTED_DECISION_WRITE_AUTHORITY_EVIDENCE_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ARではcommit candidateに対するapplication-authority Evidence GateをPressureした。

R3ASではGateがtest-onlyにsingle-qualifiedでも:

```text
application-authority Gate qualified
  != write attempt created
```

を維持し、explicit write-attempt witnessが存在する場合だけtest-only write-attempt candidateを作る。

```text
write-attempt candidate
  != sink mutation
  != write outcome
  != committed Accepted Decision
```

である。

## 2. current corpus

current Repository-backed R3AR stateは:

```text
accepted-decision-application-authority-unresolved
```

である。

したがってcurrent corpusだけからwrite-attempt candidateを生成しない。

R3ASのcandidate caseはsynthetic single-qualified R3AR application authorityを用いるPressureである。

## 3. explicit write-attempt witness

test-only witnessは最低限:

- CommitCandidateIdentity
- ApplicationAuthorityEvidenceID
- AcceptedDecisionSinkRef
- RevisionObservationRef
- ObservedDecisionRevision
- IdempotencyContractRef
- IdempotencyIdentity
- WriteOutcomeContractRef
- AttemptProvenanceRef

を持つ。

R3AR GateやR3AQ candidateからmissing witness fieldを補完しない。

## 4. exact Gate binding

R3AR Gateは:

```text
accepted-decision-application-authority-evidence-qualified
```

かつQualifiedEvidenceIDsがexactly oneであることを要求する。

write-attempt witnessのApplicationAuthorityEvidenceIDはそのexact IDと一致しなければならない。

```text
application authority unresolved
  -> no attempt candidate

multiple-qualified precedence unresolved
  -> no attempt candidate
```

である。

## 5. exact sink / revision observation

witnessの:

- AcceptedDecisionSinkRef
- ObservedDecisionRevision
- RevisionObservationRef

をR3AR current revision observationへexact bindingする。

```text
R3AQ ExpectedDecisionRevision
  = R3AR ObservedDecisionRevision
  = R3AS witness ObservedDecisionRevision
```

を要求する。

revision mismatch / missing observation provenanceはfail closedする。

ただしこの一致はactual write直前のCASを実行したという意味ではない。

## 6. explicit idempotency identity

R3ARで確認したIdempotencyContractRefへexact bindingし、witnessはIdempotencyIdentityを明示する。

```text
idempotency identity present
  != duplicate suppression executed
  != exactly-once delivery
  != sink dedupe confirmed
```

である。

R3ASでIdempotencyIdentityをhash生成・推測しない。

## 7. write outcome contract

witnessはR3ARでqualifiedしたWriteOutcomeContractRefへexact bindingする。

```text
outcome contract bound
  != outcome observed
  != applied / conflict / unknown decided
```

である。

R3ASでWriteOutcomeを生成しない。

## 8. attempt provenance

AttemptProvenanceRefを必須とする。

```text
R3AQ CommitProvenanceRef
  != R3AS AttemptProvenanceRef

R3AR ApplicationProvenanceRef
  != R3AS AttemptProvenanceRef
```

として、commit intent / application authority / actual attempt intent provenanceを分離する。

## 9. write-attempt candidate

test-only candidateは:

- CommitCandidateIdentity
- ApplicationAuthorityEvidenceID
- AcceptedDecisionSinkRef
- RevisionObservationRef
- ObservedDecisionRevision
- IdempotencyContractRef
- IdempotencyIdentity
- WriteOutcomeContractRef
- AttemptProvenanceRef
- WriteAttemptCandidateIdentity
- AuthorityLimits

までを持てる。

Accepted Decisionの内容はR3AQ commit candidateへ戻ってdrill-downできるため、R3ASで新しいauthority payloadを複製しない。

## 10. duplicate / conflict

同じ:

- CommitCandidateIdentity
- ApplicationAuthorityEvidenceID
- AcceptedDecisionSinkRef
- ObservedDecisionRevision

について:

- exact duplicate witness -> normalize可
- IdempotencyIdentityが異なる -> fail closed
- AttemptProvenanceRefが異なる -> fail closed

とする。

R3ASではretry generation / attempt sequence / latest attempt precedenceを導入しない。

## 11. 禁止するshortcut

R3AS witness / candidate shapeは次を持たない。

- WriteExecuted / Attempted
- WriteOutcome / CommitOutcome
- Applied / NotApplied / Conflict
- Committed / CommitSucceeded
- AcceptedDecision
- NewDecisionRevision
- CurrentAuthority
- ToolCall / Command
- Executor / Endpoint / Transport
- Retry / RetryCount / AttemptNumber
- NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 12. test-only source

- `tools/reference-business-procedure-check/actor_context_r3as_write_attempt_candidate_pressure_test.go`

Pressureするcase:

1. current unresolved R3AR Gateではattempt candidateを作れない
2. single-qualified Gateでもno witnessならauto-attemptしない
3. exact explicit witnessでone attempt candidateを作る
4. wrong CommitCandidateIdentity / ApplicationAuthorityEvidenceID / sinkはfail closed
5. revision / RevisionObservationRef mismatchはfail closed
6. IdempotencyContractRef / WriteOutcomeContractRef mismatchはfail closed
7. missing IdempotencyIdentity / AttemptProvenanceRefはfail closed
8. multiple-qualified R3AR Gateをwitnessで迂回しない
9. exact duplicate normalize
10. conflicting IdempotencyIdentity / AttemptProvenanceRefはfail closed
11. input order / repeated buildでcandidate identityを変えない
12. attempt / outcome / execution shortcutを追加しない

## 13. current conclusion

R3ASで固定するのは:

```text
explicit write-attempt witness
  + exact R3AQ commit candidate
  + exact single R3AR application authority
  + exact revision observation
  + explicit idempotency identity
  + exact outcome contract
  + attempt provenance
  -> test-only write-attempt candidate

write-attempt candidate
  != write execution
  != write outcome
  != Accepted Decision committed
```

である。

production writer、transaction、CAS、transport、retry scheduler、idempotency store、decision store mutationは追加しない。

## 14. 次

次段ではwrite-attempt candidateに対して外部writerから返るtest-only observationを:

```text
write attempt observation
  != authoritative write outcome accepted
  != Accepted Decision applied
```

として扱い、attempt identity / outcome observation / post-write revision observationを分離してPressureする。
