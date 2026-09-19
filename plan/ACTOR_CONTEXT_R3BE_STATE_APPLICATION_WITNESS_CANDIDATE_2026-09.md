# Actor Context R3BE — Explicit Authoritative-state Application Witness / Candidate Boundary

- Status: ISSUE-000053 / Phase 3 explicit state-application witness pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3BD_AUTHORITATIVE_STATE_APPLICATION_AUTHORITY_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3BDではauthoritative-state application authorityをEvidence Gateとして分離した。

R3BEではsingle-qualified Gateでも自動適用せず:

```text
single-qualified R3BD Gate
  + explicit state-application witness
  + exact result candidate
  + explicit apply / reject decision
  + provenance
  -> test-only state-application candidate

candidate
  != state applied
  != revision accepted
  != CurrentAuthority projected
```

を固定する。

## 2. explicit witness

test-only witnessは少なくとも:

- ReconciliationResultAcceptanceCandidateIdentity
- StateApplicationAuthorityEvidenceID
- StateApplicationLocusRef
- RevisionAcceptanceLocusRef
- CurrentAuthorityProjectionLocusRef
- ObservedDecisionRevision
- ObservedStateFingerprint
- Decision
- StateApplicationProvenanceRef

を持つ。

Decisionはtest-onlyに:

- `apply-authoritative-state`
- `reject-authoritative-state-application`

だけをPressureする。

## 3. exact binding

candidate化には:

- R3BD Gate status = single-qualified
- exact qualified EvidenceID
- exact R3BC result candidate identity
- exact application / revision-acceptance / CurrentAuthority-projection loci
- exact observed revision / fingerprint

を要求する。

Gateが保持していないlocusや別candidateへwitnessを差し替える場合はfail closedする。

## 4. Gate qualifiedだけではcandidateを作らない

```text
R3BD Gate qualified
  != state-application intent exists
  != state applied
```

である。

no witnessはcandidateなしを正常系とする。

## 5. Decisionを観測値から推測しない

```text
result candidate = accept
  != apply-authoritative-state

matching revision
  != apply-authoritative-state

state fingerprint present
  != apply-authoritative-state
```

Decisionはexplicit witnessだけから取得する。

## 6. reject candidate

```text
reject-authoritative-state-application
  != rollback
  != compensation
  != retry
  != state failure
```

である。

rejectはapplication intentを進めないことだけを表す。

## 7. candidate

test-only candidateは:

- ReconciliationCandidateIdentity
- ReconciliationResultAcceptanceCandidateIdentity
- StateApplicationAuthorityEvidenceID
- AcceptedDecisionSinkRef
- RevisionAuthorityRef
- StateApplicationLocusRef
- RevisionAcceptanceLocusRef
- CurrentAuthorityProjectionLocusRef
- ObservedDecisionRevision
- ObservedStateFingerprint
- Decision
- StateApplicationProvenanceRef
- StateApplicationCandidateIdentity
- AuthorityLimits

までを持てる。

AppliedState / CurrentAuthorityは持たない。

## 8. multiple-qualified Gateを迂回しない

R3BD Gateがmultiple-qualifiedの場合、witnessが特定EvidenceIDを示してもcandidateを作らない。

```text
witness selected one authority
  != precedence authority exists
```

である。

## 9. duplicate / conflict

same exact result candidate + same authorityについて:

- exact duplicate witness -> normalize可
- Decision conflict -> fail closed
- locus conflict -> fail closed
- provenance conflict -> fail closed

last writer / latest revision / majorityでwinnerを選ばない。

## 10. current Repository Evidence

current Repository-backed R3BD production authorityはunresolvedである。

したがってR3BE candidate caseはsynthetic single-qualified GateによるPressureである。

## 11. 禁止するshortcut

R3BE witness / candidate shapeは次を持たない。

- StateApplied / StateSynchronized / Reconciled
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

- `tools/reference-business-procedure-check/actor_context_r3be_state_application_witness_candidate_pressure_test.go`

Pressureするcase:

1. unresolved R3BD Gateではcandidateを作れない
2. single-qualified Gateでもno witnessではcandidateなし
3. explicit apply / reject witnessをcandidate化できる
4. wrong authority / result candidate / loci / revision / fingerprintはfail closed
5. Decision / provenance missingはfail closed
6. observed stateからDecisionを推測しない
7. multiple-qualified Gateをwitnessで迂回しない
8. exact duplicate normalize、decision/locus/provenance conflictはfail closed
9. repeated buildでcandidate identityを変えない
10. state application / CurrentAuthority / retry shortcutを追加しない

## 13. current conclusion

R3BEで固定するのは:

```text
single-qualified R3BD Gate
  + explicit exact state-application witness
  -> test-only state-application candidate

candidate
  != authoritative state applied
  != revision accepted
  != CurrentAuthority projected
```

である。

production state applicator、authority projection mutation、decision-store mutation、retry / rollback engineは追加しない。

## 14. 次

次段ではapply candidateがあっても実行済みとみなさず:

```text
state-application candidate
  + attributed application execution observation
  + post-application current-authority observation
  -> raw application Evidence

raw Evidence
  != state application accepted
  != CurrentAuthority current
```

をPressureする。
