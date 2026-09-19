# Actor Context R3AR — Accepted Decision Application Authority / Current Revision Evidence Gate

- Status: ISSUE-000053 / Phase 3 Accepted Decision pre-write application gate / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AQ_ACCEPTED_DECISION_COMMIT_CANDIDATE_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AP_ACCEPTED_DECISION_WRITE_AUTHORITY_EVIDENCE_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AO_EXPLICIT_ACCEPTANCE_WITNESS_BOUNDARY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AQではexplicit commit witnessからtest-only commit candidateを作った。

しかし:

```text
commit candidate prepared
  != write authorized now
```

である。

R3ARではauthoritative sinkへwrite attemptを作る前に:

- application authority
- current sink revision
- idempotency contract
- write outcome contract

がexplicitに揃っているかをEvidence GateでPressureする。

Gateがqualifiedでもwrite attempt / write outcomeは生成しない。

## 2. static write authorityとapplication authorityを分離する

R3AP write authorityは:

```text
where / under which repository authority
Accepted Decision may be written
```

をPressureした。

R3AR application authorityは:

```text
whether this commit candidate may be advanced
toward an explicit write attempt at the current sink revision
```

を別に扱う。

```text
write-authority evidence qualified
  != application-authority evidence qualified
```

である。

R3AP evidenceやR3AQ CommitProvenanceRefをapplication authorityへ暗黙転用しない。

## 3. current Repository state

current Repository-backed corpusには、Operational Accepted Decision commit candidateに対するcompleteな:

- application-authority declaration
- current sink revision Observation contract
- idempotency contract
- write outcome contract

を一つのproduction locusとして確認していない。

したがってcurrent stateは:

```text
accepted-decision-application-authority-unresolved
```

を維持する。

これはR3AQ commit candidateがinvalidという意味ではない。

## 4. qualifying application-authority evidence

test-only evidence candidateは少なくとも:

- EvidenceID
- RepositoryBacked
- AuthorityKind
- AuthorityDeclarationRef
- BusinessAddress scope
- CommitCandidateIdentity coverage
- WriteAuthorityEvidenceID coverage
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- IdempotencyContractRef
- WriteOutcomeContractRef
- ApplicationProvenanceRef
- DerivedOnly

を持つ。

AuthorityKindはtest-onlyに:

```text
operational-accepted-decision-application-authority
```

を要求する。

## 5. current sink revision observation

application-authority evidenceとは別に、current sink revision observationを要求する。

test-only observation:

- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- ObservedDecisionRevision
- RevisionObservationRef

を持つ。

```text
ExpectedDecisionRevision
  = ObservedDecisionRevision
```

であることを要求する。

R3AQ candidateのExpectedDecisionRevisionと異なる場合:

```text
accepted-decision-current-revision-mismatch
```

としてfail closedする。

newer-looking文字列やrevision大小をprecedenceへ使わない。

## 6. idempotency contract

qualifying evidenceにはIdempotencyContractRefを要求する。

ただし:

```text
idempotency contract exists
  != idempotency key assigned
  != duplicate write prevented
  != write attempted
```

である。

R3ARでIdempotencyKeyを生成しない。

## 7. write outcome contract

qualifying evidenceにはWriteOutcomeContractRefを要求する。

これはwrite attempt後に:

- applied
- not-applied
- conflict
- unknown

等をどう区別するかのcontract locusを示すためのpressureである。

ただしR3ARではactual outcome enumも固定しない。

```text
write outcome contract exists
  != write outcome observed
```

である。

## 8. exact candidate / authority / sink binding

qualifying evidenceはexact:

- CommitCandidateIdentity
- WriteAuthorityEvidenceID
- AcceptedDecisionSinkRef
- RevisionAuthorityRef

へbindingする。

類似sink、同じAssertionClass、同じDecisionだけではqualifyしない。

## 9. multiple-qualified

複数application-authority evidenceがqualifyする場合:

```text
accepted-decision-application-authority-multiple-qualified-precedence-unresolved
```

とする。

candidate指定、scope specificity、revision表記、source location、corroboration countからwinnerを選ばない。

## 10. Gate result

test-only Gateは:

- ApplicationAuthorityStatus
- QualifiedEvidenceIDs
- AcceptedDecisionSinkRefs
- ObservedDecisionRevision
- IdempotencyContractRefs
- WriteOutcomeContractRefs
- InsufficientEvidenceIDs
- AuthorityLimits

までを持てる。

```text
application-authority evidence qualified
  != write attempt created
  != sink mutated
  != Accepted Decision committed
```

である。

## 11. 禁止するshortcut

R3AR Gate / evidence / observation shapeは次を持たない。

- WriteAttempt
- CommitCommand
- IdempotencyKey
- WriteOutcome / CommitOutcome
- Committed / CommitSucceeded
- NewDecisionRevision
- AcceptedDecision
- CurrentAuthority
- Applied / Apply / AutoApply
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Retry / Reauthorize
- Confidence / Priority / Severity / Winner

## 12. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ar_application_authority_revision_gate_pressure_test.go`

Pressureするcase:

1. current inspected corpusではapplication authority unresolved
2. R3AP write authorityだけではR3AR authorityを作らない
3. complete synthetic application authority + exact current revision observationでGate qualified
4. wrong CommitCandidateIdentityはinsufficient
5. wrong WriteAuthorityEvidenceID / sink / RevisionAuthorityRefはinsufficient
6. stale ObservedDecisionRevisionはfail closed
7. missing RevisionObservationRefはfail closed
8. missing IdempotencyContractRef / WriteOutcomeContractRefはinsufficient
9. multiple-qualifiedはprecedence unresolved
10. Gate qualifiedでもattempt / outcome / commitを生成しない
11. input orderでGate resultを変えない
12. write/application/execution shortcutを追加しない

## 13. current conclusion

R3ARで固定するのは:

```text
R3AQ commit candidate
  + explicit repository-backed application authority
  + exact current sink revision observation
  + idempotency contract locus
  + write outcome contract locus
  -> application-authority Evidence Gate

Gate qualified
  != write attempt
  != Accepted Decision committed
```

である。

production transaction protocol、CAS、locking、idempotency implementation、write executor、decision store mutationは追加しない。

## 14. 次

次段ではR3AR Gateがsingle-qualifiedというsynthetic pressureの下で:

```text
explicit write-attempt witness
  + exact current revision
  + explicit idempotency identity
  -> test-only write-attempt candidate

write-attempt candidate
  != write outcome
  != committed decision
```

をPressureする。
