# Actor Context R3AN — Authority Assertion Acceptance Locus Evidence Gate

- Status: ISSUE-000053 / Phase 3 assertion-acceptance locus Evidence Gate / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AM_QUALIFIED_EVIDENCE_AUTHORITY_ASSERTION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3L_AUTHORIZATION_ACTION_LOCUS_EVIDENCE_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Z_AUTHORIZATION_FRESHNESS_PRECEDENCE_EVIDENCE_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AMでは:

```text
qualified Evidence
  -> provenance-bound authority assertion

authority assertion
  != accepted authority value
  != Accepted Decision
```

を固定した。

次に必要なのは「どのassertionを採用するか」ではない。

先に:

> response-derived authority assertionをacceptできるauthority / decision locusがRepository上に明示されているか。

をEvidence Gateで確認する。

## 2. 一般的Acceptance概念をOperational acceptanceへ転用しない

Phase 0基線では:

```text
Authoritative Source / Canonical Meaning / Accepted Decision
Proposal / Shared Acceptance Boundary
```

が一般原則として言及されている。

しかし:

```text
general Shared Acceptance concept
  != Operational Authorization assertion acceptance authority
```

である。

特定のauthority assertion classについて:

- 誰／何がacceptするか
- どのscopeへ適用するか
- どのAccepted Decision locusへ記録するか
- どのauthoritative sourceへ戻るか

が明示されない限り、一般的Acceptance概念だけでacceptance authorityを補完しない。

## 3. Evidence Gate criteria

test-only gateでproduction acceptance locus候補がqualifyするには、少なくとも次を要求する。

1. explicit acceptance authority kind
2. exact assertion class applicability
3. explicit acceptance scope
4. explicit Accepted Decision / accepted-value sink
5. authority provenance source
6. assertion sourceとは別にacceptance authorityを説明できる
7. Repository-backed Business / Operational need
8. derived R3 plan/test自身をauthority sourceにしていない

```text
assertion provenance
  != acceptance provenance
```

である。

## 4. current inspected Repository state

current R3 lineage / Referenceで確認できるもの:

### Phase 0 projection baseline

- Accepted Decision / Shared Acceptanceを一般概念として言及
- Operational Authorization assertion classごとのacceptance authority / sinkは明示しない

### R3Y

- reevaluation / prior Observation reuse authority axesをPressure
- response-derived assertion acceptance locusではない

### R3AM

- qualified Evidence由来assertionとprovenanceを保持
- derived analysisでありacceptance authorityではない

### Business Procedure / protected-target Reference

- Repository-backed operational need / stop contextを与える
- assertion acceptance owner / locusを与えない

したがってcurrent inspected caseは:

```text
AcceptanceGateStatus = insufficient-evidence
```

とする。

これは「acceptance mechanismが存在しない」というRepository-wide断定ではなく、current inspected evidenceからproduction locusを選べないという意味である。

## 5. evidence roleを分離する

test-only evidence roleは例として:

```text
conceptual-acceptance-boundary
operational-business-need
authority-axis-analysis
derived-assertion
acceptance-authority-candidate
```

を区別できる。

```text
conceptual support
  != qualifying acceptance authority

business need
  != acceptance owner

derived assertion
  != acceptance locus
```

である。

## 6. synthetic pressure

Evidence Gate自体をPressureするためsynthetic candidateを使ってよい。

qualifying synthetic candidateは:

- explicit acceptance authority kind
- exact assertion class
- scope
- accepted-decision sink
- provenance
- Repository-backed need

をすべて持つ。

この場合:

```text
Gate = acceptance-locus-pressure-qualified
```

まで進められる。

ただし:

```text
qualified gate
  != production mechanism selected
  != accepted value selected
  != assertion accepted
```

である。

## 7. multiple candidates

複数candidateがqualifyしても:

- winner
- priority
- precedence
- nearest scope
- last writer
- majority / quorum

を導入しない。

Gateはqualifying candidate identitiesを保持するだけにする。

production precedenceは別Evidenceが必要である。

## 8. incomplete candidate

次のどれかが欠けるcandidateはqualifyさせない。

- authority kind
- assertion class applicability
- scope
- accepted-decision sink
- provenance
- Repository-backed need

特に:

```text
source says "accept"
  != acceptance locus complete
```

とする。

## 9. Gate result

test-only Gate resultは:

- GateStatus
- AssertionClassesUnderPressure
- QualifyingCandidateRefs
- InsufficientEvidenceRefs
- EvidenceRoles
- AuthorityLimits

までを持てる。

次を持たない。

- AcceptedAuthorityValue
- AcceptedDecision
- SelectedAcceptanceLocus
- Winner / PreferredCandidate
- ApprovalStatus
- Apply / AutoApply
- CurrentAuthority
- NextAction
- HandoffOwner
- CanExecute / Authorized / Allowed

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3an_acceptance_locus_evidence_gate_pressure_test.go`

Pressureするcase:

1. current inspected evidenceはinsufficient-evidence
2. Phase 0のgeneral Shared Acceptance概念だけではqualifyしない
3. R3AM assertion provenanceだけではqualifyしない
4. Business Procedure operational needだけではqualifyしない
5. complete synthetic candidateだけがpressure-qualifiedになる
6. missing sink / scope / provenance等はqualifyしない
7. multiple complete candidateをwinnerへ丸めない
8. qualifying candidateがあってもassertion / Accepted Decisionを生成しない
9. input orderでGate identityを変えない
10. acceptance / execution shortcutを追加しない

## 11. current conclusion

R3ANで固定するのは:

```text
assertion provenance available
  != acceptance authority available

general acceptance concept
  != Operational assertion acceptance locus

Evidence Gate qualified
  != assertion accepted
```

である。

current inspected Repository evidenceではproduction acceptance locusを選択しない。

production Accepted Decision schema、approval workflow、authority registry、precedence rule、Command executionは追加しない。

## 12. 次

次段ではR3AN Gateがinsufficient-evidenceのままでも、Agent / Humanへ:

```text
qualified assertion exists
acceptance authority unresolved
```

をlosslessに説明できるoperational stop / bounded projectionへ戻す。

```text
acceptance authority unresolved
  != assertion rejected
  != authorization denied
```

を維持する。
