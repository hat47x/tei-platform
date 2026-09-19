# Actor Context R3AU — Write Outcome Acceptance Authority Evidence Gate

- Status: ISSUE-000053 / Phase 3 write-outcome acceptance-authority pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AT_WRITE_ATTEMPT_OBSERVATION_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AR_APPLICATION_AUTHORITY_REVISION_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AS_WRITE_ATTEMPT_CANDIDATE_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3ATではwriterから返ったtest-only raw observationを:

```text
raw attempt observation
  != accepted write outcome
  != Accepted Decision committed
```

として保持した。

R3AUではraw observationをauthoritative write outcomeとして受理する前に:

```text
raw writer observation
  + explicit outcome-acceptance authority
  -> outcome-acceptance Evidence Gate

Gate qualified
  != authoritative outcome accepted
  != Accepted Decision applied
```

をPressureする。

## 2. current corpus

current Repository-backed Phase 3には:

- R3AR WriteOutcomeContractRef
- R3AT raw writer observation
- attributed post-write revision observation

はある。

しかしcurrent inspected evidenceから、Operational Accepted Decision write outcomeを受理するcomplete production authority locusは確認しない。

したがってcurrent stateは:

```text
write-outcome-acceptance-authority-unresolved
```

を維持する。

これはraw observationがincorrect / rejectedという意味ではない。

## 3. contractはacceptance authorityではない

R3ARのWriteOutcomeContractRefはoutcomeをどのcontractに照らすかのlocusである。

```text
write outcome contract exists
  != outcome acceptance authority exists
```

である。

contract refだけからwriter-reported-applied等をaccepted outcomeへ昇格させない。

## 4. raw observationもauthorityではない

R3AT ObservationIdentity / WriterObservationRef / PostWriteRevisionObservationRefはattributed observationである。

```text
raw observation attributed
  != outcome trusted
  != outcome accepted
```

である。

Derived R3AT observation自身をacceptance authority sourceへしない。

## 5. qualifying evidence

test-only outcome-acceptance authority evidenceは少なくとも:

- EvidenceID
- EvidenceRole
- RepositoryBacked
- AuthorityKind
- AuthorityDeclarationRef
- WriteAttemptCandidateIdentity coverage
- ObservationIdentity coverage
- WriteOutcomeContractRef coverage
- ReportedOutcome coverage
- AcceptedDecisionSinkRef
- BusinessAddress scope
- OutcomeDecisionLocusRef
- OutcomeAcceptanceProvenanceRef
- DerivedOnly

を持つ。

AuthorityKindはtest-onlyに:

```text
operational-write-outcome-acceptance-authority
```

を要求する。

## 6. exact binding

qualifying evidenceはexact:

- R3AS WriteAttemptCandidateIdentity
- R3AT ObservationIdentity
- R3AS WriteOutcomeContractRef
- R3AT ReportedOutcome
- AcceptedDecisionSinkRef
- BusinessAddress scope

へbindingする。

同じIdempotencyIdentity、同じwriter ref family、同じoutcome語彙だけでは補完しない。

## 7. post-write revisionはacceptance authorityを作らない

R3ATのPostWriteObservedDecisionRevisionがpre-write revisionと異なっていても:

```text
revision changed
  != outcome accepted
```

である。

同じでも:

```text
revision unchanged
  != outcome rejected
```

である。

R3AU Gateはrevision changeからqualifying authorityを生成しない。

## 8. raw outcome classはdecisionではない

R3AT raw classes:

- writer-reported-applied
- writer-reported-not-applied
- writer-reported-conflict
- writer-reported-unknown

をそのまま保持する。

```text
writer-reported-applied
  != authoritative applied

writer-reported-not-applied
  != retry authorized

writer-reported-conflict
  != retry authorized

writer-reported-unknown
  != failure
  != retry eligible
```

である。

## 9. multiple-qualified

複数outcome-acceptance authority evidenceがqualifyする場合:

```text
write-outcome-acceptance-authority-multiple-qualified-precedence-unresolved
```

とする。

scope specificity、source location、corroboration count、revision表記、reported outcome内容からwinnerを選ばない。

## 10. Gate result

test-only Gate resultは:

- OutcomeAcceptanceAuthorityStatus
- QualifiedEvidenceIDs
- OutcomeDecisionLocusRefs
- RawReportedOutcome
- PostWriteObservedDecisionRevision
- InsufficientEvidenceIDs
- AuthorityLimits

までを持てる。

これはaccepted outcome recordではない。

## 11. 禁止するshortcut

R3AU Gate / evidence shapeは次を持たない。

- AcceptedWriteOutcome
- OutcomeAccepted
- AcceptedOutcomeDecision
- Committed / CommitSucceeded
- AcceptedDecision
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

- `tools/reference-business-procedure-check/actor_context_r3au_write_outcome_acceptance_authority_gate_pressure_test.go`

Pressureするcase:

1. current inspected evidenceではoutcome acceptance authority unresolved
2. WriteOutcomeContractRefだけではqualifyしない
3. derived raw observationだけではqualifyしない
4. complete synthetic outcome-acceptance authorityはGateだけqualifyできる
5. missing authority / exact binding / scope / decision locus / provenanceはいずれもqualifyしない
6. wrong WriteAttemptCandidateIdentity / ObservationIdentity / contract / raw outcome / sinkはqualifyしない
7. four raw outcome classesをauthority decisionへ変換しない
8. post-write revision changed / unchangedをacceptanceへ変換しない
9. multiple-qualifiedはprecedence unresolved
10. input orderでGate resultを変えない
11. accepted outcome / current authority / retry shortcutを追加しない

## 13. current conclusion

R3AUで固定するのは:

```text
raw writer observation
  + repository-backed outcome-acceptance authority evidence
  -> outcome-acceptance Evidence Gate

Gate qualified
  != raw outcome accepted
  != Accepted Decision committed
  != current authority applied
```

である。

production outcome registry、trust/signature validation、acceptance workflow、retry scheduler、decision store mutationは追加しない。

## 14. 次

次段ではsingle-qualified outcome-acceptance authorityというsynthetic pressureの下でも:

```text
explicit outcome acceptance witness
  + exact ObservationIdentity
  + exact acceptance authority
  + explicit acceptance decision
  + acceptance provenance
  -> test-only accepted-outcome candidate

accepted-outcome candidate
  != Accepted Decision applied
  != retry authorized
```

をPressureする。
