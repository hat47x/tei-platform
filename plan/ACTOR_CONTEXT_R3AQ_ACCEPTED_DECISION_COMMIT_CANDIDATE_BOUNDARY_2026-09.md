# Actor Context R3AQ — Accepted Decision Explicit Commit Witness / Candidate Boundary

- Status: ISSUE-000053 / Phase 3 Accepted Decision commit-candidate pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AP_ACCEPTED_DECISION_WRITE_AUTHORITY_EVIDENCE_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AO_EXPLICIT_ACCEPTANCE_WITNESS_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3APでは:

```text
acceptance witness observation
  + repository-backed write authority evidence
  -> Accepted Decision write authority Gate

Gate qualified
  != witness committed
  != Accepted Decision created
```

を固定した。

R3AQではGateがtest-onlyにsingle-qualified write authorityを得た場合でも、explicit commit witnessから作るのは:

```text
commit candidate
```

までに限定する。

```text
commit candidate
  != committed Accepted Decision
  != authoritative sink mutation
  != current authority applied
```

である。

## 2. current corpus

current inspected R3AP stateは:

```text
accepted-decision-write-authority-unresolved
```

である。

したがってcurrent Repository-backed corpusだけからcommit candidateを生成しない。

commit-candidate caseはR3AP GateをPressureするcomplete synthetic write-authority evidenceを用いる。

## 3. explicit commit witness

test-only commit witnessは最低限:

- WitnessIdentity
- WriteAuthorityEvidenceID
- AcceptedDecisionSinkRef
- ExpectedDecisionRevision
- CommitProvenanceRef

を持つ。

R3AO WitnessIdentityはexactでなければならない。

AcceptanceProvenanceRefやAssertionIdentityだけではcommit witnessを補完しない。

## 4. exact write-authority evidence

R3AP Gateが:

```text
accepted-decision-write-authority-evidence-qualified
```

で、QualifiedEvidenceIDsがexactly oneであることを要求する。

commit witnessのWriteAuthorityEvidenceIDはそのexact IDと一致しなければならない。

```text
write authority unresolved
  -> no commit candidate

multiple qualified precedence unresolved
  -> no commit candidate
```

とする。

witnessが一candidateを名指ししてもprecedenceを迂回しない。

## 5. exact authoritative sink

commit witnessのAcceptedDecisionSinkRefはqualified write-authority evidenceのexact sinkと一致しなければならない。

```text
similar source location
  != same Accepted Decision sink
```

である。

DecisionLocusRefやAcceptanceProvenanceRefをsinkへ読み替えない。

## 6. expected revision

commit witnessはExpectedDecisionRevisionを明示する。

qualified write-authority evidenceのCurrentDecisionRevisionと一致するときだけcommit candidateを作れる。

```text
ExpectedDecisionRevision
  != CurrentDecisionRevision
  -> stale commit witness
  -> fail closed
```

とする。

revision文字列の大小やnewer-looking表記をprecedenceへ使わない。

## 7. commit provenance

CommitProvenanceRefを必須にする。

```text
R3AO AcceptanceProvenanceRef
  != R3AQ CommitProvenanceRef

R3AP WriteProvenanceRef
  != R3AQ CommitProvenanceRef
```

であり得る。

decision witness provenance、write-authority provenance、actual commit intent provenanceを分離する。

## 8. commit candidate

test-only candidateは:

- WitnessIdentity
- AssertionIdentity
- AssertionClass
- AssertedValue
- AcceptanceDecision
- AcceptanceProvenanceRef
- WriteAuthorityEvidenceID
- AcceptedDecisionSinkRef
- ExpectedDecisionRevision
- WriteAuthorityRevisionAuthorityRef
- WriteAuthorityProvenanceRef
- CommitProvenanceRef
- CommitCandidateIdentity
- AuthorityLimits

までを持てる。

これはwrite commandではない。

## 9. candidateを作ってもrevisionを進めない

R3AQは:

- compare-and-write
- revision increment
- CAS
- transaction commit
- append
- overwrite

を実行しない。

```text
candidate revision matches current revision
  != write committed
```

である。

## 10. accept / reject

R3AO witnessのDecisionをそのまま保持する。

```text
accept commit candidate
  != authority value applied

reject commit candidate
  != Authorization denied
```

とする。

R3AQでDecisionを変更・推測しない。

## 11. duplicate / conflict

同じWitnessIdentity / WriteAuthorityEvidenceID / sink / expected revisionについて:

- exact duplicate commit witness -> normalize可
- CommitProvenanceRefだけ異なる -> aggregation authority未解決としてfail closed
- ExpectedDecisionRevisionが異なる -> stale / conflictとしてfail closed

とする。

last writer / latest revision / majorityを導入しない。

## 12. 禁止するshortcut

R3AQ shapeは次を持たない。

- Committed
- CommitSucceeded
- AcceptedDecision
- AcceptedAuthorityValue
- NewDecisionRevision
- CurrentAuthority
- ReevaluationAuthorityStatus
- ReuseAuthorityStatus
- RuntimeOutcome
- AuthorizationResult
- Applied / Apply / AutoApply
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Retry / Reauthorize
- Confidence / Priority / Severity / Winner

## 13. test-only source

- `tools/reference-business-procedure-check/actor_context_r3aq_accepted_decision_commit_candidate_pressure_test.go`

Pressureするcase:

1. current unresolved R3AP Gateではcandidateを作れない
2. single-qualified write authority + exact explicit commit witnessでcandidateを作れる
3. no commit witnessではauto-candidateを作らない
4. wrong WitnessIdentityはfail closed
5. wrong WriteAuthorityEvidenceIDはfail closed
6. multiple-qualified precedence unresolvedを迂回しない
7. wrong AcceptedDecisionSinkRefはfail closed
8. stale ExpectedDecisionRevisionはfail closed
9. missing CommitProvenanceRefはfail closed
10. accept / rejectをそのまま保持しAuthorization outcomeへ変換しない
11. exact duplicate normalize / conflicting provenance fail closed
12. input order / repeated buildでcandidate identityを変えない
13. commit / authority application / execution shortcutを追加しない

## 14. current conclusion

R3AQで固定するのは:

```text
explicit commit witness
  + exact R3AO WitnessIdentity
  + exact single write authority
  + exact sink
  + matching expected revision
  + commit provenance
  -> test-only commit candidate

commit candidate
  != authoritative write
  != Accepted Decision committed
  != current authority applied
```

である。

production decision store、transaction protocol、CAS、locking、authority application、Command executionは追加しない。

## 15. 次

次段ではcommit candidateを実際にauthoritative sinkへ適用する前に:

```text
candidate prepared
  != write authorized now
```

を維持し、current sink revision再確認 / application authority / idempotency / write outcome separationをPressureする。
