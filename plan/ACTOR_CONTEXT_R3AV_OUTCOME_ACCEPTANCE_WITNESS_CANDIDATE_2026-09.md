# Actor Context R3AV — Explicit Write Outcome Acceptance Witness / Candidate Boundary

- Status: ISSUE-000053 / Phase 3 explicit outcome-acceptance witness pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AU_WRITE_OUTCOME_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AT_WRITE_ATTEMPT_OBSERVATION_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AS_WRITE_ATTEMPT_CANDIDATE_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AUでは:

```text
raw writer observation
  + repository-backed outcome-acceptance authority evidence
  -> outcome-acceptance Evidence Gate

Gate qualified
  != raw outcome accepted
```

を固定した。

R3AVではsingle-qualified Gateというtest-only pressureの下でも、explicit outcome-acceptance witnessが存在する場合だけcandidateを作る。

```text
explicit outcome-acceptance witness
  + exact ObservationIdentity
  + exact single R3AU authority evidence
  + explicit decision
  + acceptance provenance
  -> test-only outcome-acceptance candidate

outcome-acceptance candidate
  != Accepted Decision applied
  != current authority applied
  != retry authorized
```

## 2. current corpus

current Repository-backed R3AU stateは:

```text
write-outcome-acceptance-authority-unresolved
```

である。

したがってcurrent corpusだけからoutcome-acceptance candidateを生成しない。

positive caseはR3AU complete synthetic outcome-acceptance authorityを使う。

## 3. explicit witness

test-only witnessは最低限:

- ObservationIdentity
- OutcomeAcceptanceAuthorityEvidenceID
- Decision
- AcceptanceProvenanceRef

を持つ。

Decisionはtest-onlyに:

- accept-observed-outcome
- reject-observed-outcome

だけをPressureする。

raw ReportedOutcome、post-write revision、scope specificityからDecisionを推測しない。

## 4. exact observation binding

witness ObservationIdentityはR3AT normalized observationのexact identityと一致しなければならない。

同じWriteAttemptCandidateIdentity / IdempotencyIdentity / ReportedOutcomeだけでは不足する。

stale / other observationはfail closedする。

## 5. exact authority binding

R3AU Gateは:

```text
write-outcome-acceptance-authority-evidence-qualified
```

かつQualifiedEvidenceIDsがexactly oneであることを要求する。

witness OutcomeAcceptanceAuthorityEvidenceIDはそのexact IDと一致しなければならない。

current evidence setにそのIDが存在し、R3AUのexact binding条件を現在も満たすことを再確認する。

multiple-qualified precedence unresolvedをwitness指定で迂回しない。

## 6. explicit decision

Decisionが空、unknown、unsupportedならfail closedする。

```text
writer-reported-applied
  != accept-observed-outcome

writer-reported-conflict
  != reject-observed-outcome
```

である。

R3AVはDecisionをraw outcomeから導出しない。

## 7. acceptance provenance

AcceptanceProvenanceRefを必須にする。

次を分離する。

```text
R3AT WriterObservationRef
  != R3AU OutcomeAcceptanceProvenanceRef
  != R3AV AcceptanceProvenanceRef
```

observation attribution、authority provenance、actual acceptance intent provenanceを統合しない。

## 8. candidate

test-only candidateは:

- ObservationIdentity
- WriteAttemptCandidateIdentity
- RawReportedOutcome
- PostWriteObservedDecisionRevision
- OutcomeAcceptanceAuthorityEvidenceID
- OutcomeDecisionLocusRef
- OutcomeAuthorityProvenanceRef
- Decision
- AcceptanceProvenanceRef
- OutcomeAcceptanceCandidateIdentity
- AuthorityLimits

までを持てる。

これはAcceptedWriteOutcome recordでもAccepted Decision recordでもない。

## 9. accept / rejectの意味を限定する

```text
accept-observed-outcome candidate
  != raw outcome applied to authoritative state
  != Accepted Decision committed

reject-observed-outcome candidate
  != write failed
  != retry required
  != retry eligible
```

とする。

rejectはraw observationをauthoritative outcomeとして受理しない意図であり、操作再試行を意味しない。

## 10. conflict

same ObservationIdentity / authority evidenceについて:

- exact duplicate witness -> normalize可
- accept / reject decisionが異なる -> fail closed
- AcceptanceProvenanceRefが異なる -> aggregation authority未解決としてfail closed

とする。

last writer / majority / authority source proximityでwinnerを選ばない。

## 11. 禁止するshortcut

R3AV shapeは次を持たない。

- AcceptedWriteOutcome
- OutcomeAccepted
- AuthoritativeWriteOutcome
- AcceptedDecision
- Committed / CommitSucceeded
- NewDecisionRevision
- CurrentAuthority
- AppliedAuthority
- RuntimeOutcome / AuthorizationResult
- Retry / RetryEligible / RetryRequired / RetrySafe
- Reauthorize
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Confidence / Priority / Severity / Winner

## 12. test-only source

- `tools/reference-business-procedure-check/actor_context_r3av_outcome_acceptance_witness_candidate_pressure_test.go`

Pressureするcase:

1. current unresolved R3AU Gateではcandidateを作れない
2. single-qualified Gateでもno witnessならauto-acceptしない
3. exact explicit accept witnessでcandidateを作れる
4. explicit reject witnessもdecision dataとして保持できる
5. wrong ObservationIdentity / authority evidence IDはfail closed
6. multiple-qualified R3AU Gateをwitnessで迂回しない
7. missing / unsupported Decisionはfail closed
8. missing AcceptanceProvenanceRefはfail closed
9. raw outcome / revisionからDecisionを推測しない
10. exact duplicate normalize
11. conflicting decision / provenanceはfail closed
12. input order / repeated buildでcandidate identityを変えない
13. Accepted Decision / current authority / retry shortcutを追加しない

## 13. current conclusion

R3AVで固定するのは:

```text
single-qualified R3AU Gate
  + exact raw ObservationIdentity
  + explicit outcome-acceptance witness
  + explicit decision
  + acceptance provenance
  -> test-only outcome-acceptance candidate

candidate
  != authoritative outcome applied
  != Accepted Decision committed
  != retry authorized
```

である。

production outcome acceptance store、authority application、retry scheduler、transaction recovery、decision store mutationは追加しない。

## 14. 次

次段ではaccept-observed-outcome candidateがあっても:

```text
outcome accepted as evidence
  != Accepted Decision state synchronized
```

を維持し、post-write authoritative-state reconciliation / revision acceptance / authority applicationを別Evidence GateとしてPressureする。
