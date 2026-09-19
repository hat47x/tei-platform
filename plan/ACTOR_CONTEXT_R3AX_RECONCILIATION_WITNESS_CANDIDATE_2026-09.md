# Actor Context R3AX — Explicit Post-write Reconciliation Witness / Candidate Boundary

- Status: ISSUE-000053 / Phase 3 explicit reconciliation witness pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AW_POST_WRITE_RECONCILIATION_AUTHORITY_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AV_OUTCOME_ACCEPTANCE_WITNESS_CANDIDATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AT_WRITE_ATTEMPT_OBSERVATION_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AWではpost-write authoritative-state reconciliation authority Evidence Gateを分離した。

R3AXではsingle-qualified R3AW Gateでも自動reconciliationせず、explicit witnessが存在する場合だけtest-only reconciliation candidateを作る。

```text
single-qualified R3AW Gate
  + exact R3AV outcome-acceptance candidate
  + exact reconciliation authority evidence
  + exact post-write revision
  + explicit reconciliation decision
  + reconciliation provenance
  -> test-only reconciliation candidate

reconciliation candidate
  != authoritative state synchronized
  != revision applied
  != current authority applied
```

## 2. explicit witness

test-only witnessは最低限:

- OutcomeAcceptanceCandidateIdentity
- ReconciliationAuthorityEvidenceID
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- PreWriteDecisionRevision
- PostWriteRevisionObservationRef
- PostWriteDecisionRevision
- Decision
- ReconciliationProvenanceRef

を持つ。

Decisionはtest-onlyに:

- accept-post-write-state-reconciliation
- reject-post-write-state-reconciliation

だけをPressureする。

## 3. exact binding

witnessはcurrent chainへexact bindingする。

- R3AV OutcomeAcceptanceCandidateIdentity
- single-qualified R3AW authority EvidenceID
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- pre-write revision
- post-write revision observation ref
- post-write revision

いずれかがずれればfail closedする。

同じsink / same post-write revisionだけでは不足する。

## 4. no witnessでauto-reconcileしない

single-qualified R3AW Gateでもwitnessがなければcandidateを作らない。

```text
Gate qualified
  != reconciliation intent exists
  != synchronization authorized to execute
```

である。

## 5. decisionをrevision / raw outcomeから推測しない

```text
revision changed
  != accept reconciliation

revision unchanged
  != reject reconciliation

writer-reported-applied
  != accept reconciliation

writer-reported-conflict
  != reject reconciliation
```

Decisionはexplicit witnessだけから得る。

## 6. reject decisionの意味

```text
reject-post-write-state-reconciliation
  != rollback
  != compensation
  != retry
  != write failed
```

rejectはcurrent reconciliation candidateをauthoritative stateへ適用しない意図だけを表す。

## 7. provenance

ReconciliationProvenanceRefを必須にする。

次を統合しない。

```text
R3AV AcceptanceProvenanceRef
  != R3AW ReconciliationProvenanceRef
  != R3AX ReconciliationProvenanceRef
```

authority evidence provenanceとactual reconciliation intent provenanceは別である。

## 8. candidate

test-only candidateは:

- OutcomeAcceptanceCandidateIdentity
- ObservationIdentity
- WriteAttemptCandidateIdentity
- CommitCandidateIdentity
- ReconciliationAuthorityEvidenceID
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- PreWriteDecisionRevision
- PostWriteRevisionObservationRef
- PostWriteDecisionRevision
- ReconciliationLocusRef
- RevisionAcceptanceLocusRef
- AuthorityApplicationLocusRef
- Decision
- ReconciliationProvenanceRef
- ReconciliationCandidateIdentity
- AuthorityLimits

までを持てる。

StateSynchronized / RevisionAccepted / CurrentAuthorityは持たない。

## 9. multiple-qualified Gate

R3AWがmultiple-qualifiedならwitnessでEvidenceIDを指定してもprecedenceを迂回しない。

```text
multiple-qualified Gate
  != witness chooses winner
```

である。

## 10. duplicate / conflict

same outcome candidate / authority evidenceについて:

- exact duplicate witness -> normalize可
- Decisionが異なる -> fail closed
- ReconciliationProvenanceRefが異なる -> fail closed
- post-write revision bindingが異なる -> fail closed

last writer / majority / latest revisionでwinnerを選ばない。

## 11. 禁止するshortcut

R3AX shapeは次を持たない。

- StateSynchronized / Reconciled
- RevisionAccepted / RevisionApplied
- AcceptedWriteOutcome / OutcomeAccepted
- AcceptedDecision / CommittedDecision
- CurrentAuthority / AppliedAuthority
- RuntimeOutcome / AuthorizationResult
- Retry / RetryEligible / RetryRequired
- Rollback / Compensation
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 12. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ax_reconciliation_witness_candidate_pressure_test.go`

Pressureするcase:

1. unresolved R3AW Gateではcandidateを作れない
2. single-qualified Gateでもno witnessならauto-reconcileしない
3. explicit accept / reject reconciliation witnessをcandidate decisionとして保持する
4. wrong outcome candidate / authority EvidenceID / sink / revision authorityはfail closed
5. wrong pre-write / post-write revision / post-write observation refはfail closed
6. changed / unchanged revisionからDecisionを推測しない
7. raw writer outcomeからDecisionを推測しない
8. missing / unsupported Decisionはfail closed
9. missing ReconciliationProvenanceRefはfail closed
10. multiple-qualified Gateをwitnessで迂回しない
11. exact duplicate normalize、decision/provenance conflictはfail closed
12. repeated build / input orderでcandidate identityを変えない
13. synchronized state / current authority / retry shortcutを追加しない

## 13. current conclusion

R3AXで固定するのは:

```text
single-qualified reconciliation authority
  + exact explicit reconciliation witness
  -> test-only reconciliation candidate

candidate
  != state synchronized
  != revision applied
  != current authority applied
```

である。

production reconciliation executor、decision-store mutation、authority projection mutation、rollback / compensation / retry engineは追加しない。

## 14. 次

次段ではreconciliation candidateが存在しても:

```text
reconciliation intent prepared
  != reconciliation execution observed
  != authoritative state current
```

を維持し、actual reconciliation attempt / observation / post-reconciliation current-state evidenceを別boundaryとしてPressureする。
