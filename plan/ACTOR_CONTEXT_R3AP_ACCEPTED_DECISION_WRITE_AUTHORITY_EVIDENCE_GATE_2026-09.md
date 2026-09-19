# Actor Context R3AP — Accepted Decision Authoritative Write / Provenance Evidence Gate

- Status: ISSUE-000053 / Phase 3 Accepted Decision write-authority Evidence Gate / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AO_EXPLICIT_ACCEPTANCE_WITNESS_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AN_ACCEPTANCE_AUTHORITY_EVIDENCE_GATE_2026-09.md`
  - `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
  - `plan/BUSINESS_AUTHORING_EXISTING_ACCEPTANCE_IMPLEMENTATION_REUSE_AUDIT_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AOでは:

```text
qualified acceptance authority evidence
  + exact explicit witness
  -> provenance-bound acceptance witness observation

acceptance witness observation
  != Accepted Decision applied
  != current authority
```

を固定した。

R3APでは次に:

> R3AO witness observationをauthoritative Accepted Decisionとしてcommitできるwrite authority / sink / revision boundaryがRepository上に明示されているか。

をEvidence GateでPressureする。

```text
acceptance witness observed
  != Accepted Decision committed
```

である。

## 2. 一般Shared AcceptanceをOperational decision writeへ流用しない

ADR-0022やStudio / AI ProposalのShared Acceptanceは:

```text
candidate
  -> validation / impact / acceptance
  -> authoritative authoring source update
```

というauthoring boundaryを定義する。

しかし:

```text
general authoring Shared Acceptance
  != runtime Authorization assertion Accepted Decision write authority
```

である。

特に:

- authoring candidate
- Canonical source change
- runtime freshness assertion
- Operational Accepted Decision

を同一のwrite authorityへ暗黙統合しない。

既存のstale-check / revision principleをarchitecture precedentとして参照できても、Operational decision sinkを発明しない。

## 3. current inspected Repository state

current Repositoryで確認できるもの:

- R3AO acceptance witness observation
- R3AN acceptance-authority Evidence Gate
- ADR-0022 general Shared Acceptance principle
- Business authoring acceptance implementation reuse audit
- Source Index / revision / stale-check primitives

一方、current inspected evidenceではOperational Authorization assertionについて:

- explicit Accepted Decision write authority declaration
- exact witness-class / assertion-class write applicability
- explicit authoritative Accepted Decision sink
- explicit write scope
- explicit sink revision / freshness authority
- write provenance root

を同時に満たすproduction locusを確認できない。

したがってcurrent statusは:

```text
accepted-decision-write-authority-unresolved
```

とする。

これはR3AO witnessがinvalid / rejectedという意味ではない。

## 4. Evidence Gate criteria

test-only write-authority evidenceがqualifyするには少なくとも次を要求する。

### W1 repository-backed

candidateはrepository-backed Evidenceである。

synthetic witness observationやLLM outputだけでは不可。

### W2 explicit write authority declaration

```text
AuthorityKind = operational-accepted-decision-write-authority
AuthorityDeclarationRef != empty
```

が必要。

R3AN AcceptanceAuthorityEvidenceIDやR3AO AcceptanceProvenanceRefだけからwrite authorityを推定しない。

### W3 exact assertion / witness applicability

candidateは少なくとも:

- exact AssertionClass
- witness Decision semantics

を明示的にcoverする。

same wording / source familyでは補完しない。

### W4 explicit Operational scope

current BusinessAddressをexplicit scopeがcoverする。

test-only scope pressureは:

- exact-business-address
- any-business-address

だけを使う。

scope specificityはqualify条件であってprecedenceではない。

### W5 explicit authoritative Accepted Decision sink

`AcceptedDecisionSinkRef` が必要。

DecisionLocusRefやfile pathをsinkへ読み替えない。

```text
decision locus
  != authoritative decision sink
```

である。

### W6 explicit sink revision / freshness authority

authoritative writeをcurrent sink revisionへbindするため:

- RevisionAuthorityRef
- CurrentDecisionRevision

をtest-only Evidenceとして要求する。

```text
write authority known
  != target revision current
```

である。

具体revision storage / locking protocolは固定しない。

### W7 explicit write provenance root

`WriteProvenanceRef` が必要。

R3AO AcceptanceProvenanceRefはwitness provenanceであり、authoritative write provenanceそのものではない。

```text
witness provenance
  != authoritative write provenance
```

である。

## 5. current general Acceptance evidenceはqualifyしない

ADR-0022 / Shared Acceptanceはconceptual / authoring acceptance precedentである。

Business authoring reuse auditもgeneric production Acceptance APIがcurrent audited surfacesで確立済みとはしていない。

したがってそれらは:

```text
architecture precedent
  != Operational Accepted Decision write authority evidence
```

としてinsufficientのまま保持する。

## 6. synthetic pressure

Gate自体をPressureするためcomplete synthetic candidateを用いてよい。

すべてのW1–W7を満たすcandidateが一つだけある場合:

```text
accepted-decision-write-authority-evidence-qualified
```

まで進める。

ただし:

```text
write-authority evidence qualified
  != witness committed
  != Accepted Decision created
  != current authority rewritten
```

である。

## 7. multiple candidates

複数candidateがqualifyする場合:

```text
accepted-decision-write-authority-multiple-qualified-precedence-unresolved
```

とする。

- exact scope
- newer-looking revision
- closer source path
- same package
- corroboration count

をwinner / precedenceへ使わない。

## 8. stale revision pressure

test-only synthetic write candidateがqualifyしても、そのCurrentDecisionRevisionがwitnessをcommitするときのexpected revisionと一致する必要があるというboundaryを次段へ残す。

R3AP Gate自身はwriteを実行しない。

```text
revision evidence available
  != compare-and-write executed
```

とする。

## 9. Gate result

test-only Gate resultは:

- WriteAuthorityStatus
- QualifiedEvidenceIDs
- AcceptedDecisionSinkRefs
- DecisionRevisions
- UnresolvedReasons
- AuthorityLimits

までを持てる。

次は持たない。

- AcceptedDecision
- AcceptedAuthorityValue
- CommittedDecision
- Applied
- CurrentAuthority
- ReevaluationAuthorityStatus
- ReuseAuthorityStatus
- RuntimeOutcome
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Winner / PreferredAuthority

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ap_accepted_decision_write_authority_evidence_gate_pressure_test.go`

Pressureするcase:

1. current inspected Repository evidence -> write authority unresolved
2. general Shared Acceptance / Business authoring precedentだけではqualifyしない
3. R3AO witness provenanceだけではwrite authorityにならない
4. complete repository-backed synthetic W1–W7 candidateはGateだけqualifyできる
5. W1–W7欠落はいずれもqualifyしない
6. wrong AssertionClass / unsupported Decision / wrong BusinessAddress scopeはqualifyしない
7. multiple qualified candidateはprecedence unresolved
8. scope specificity / source location / revision sizeでwinnerを選ばない
9. input orderでnormalized Gate identityを変えない
10. GateからAccepted Decision / current authority / execution shortcutを生成しない

## 11. current conclusion

R3APで固定するのは:

```text
acceptance witness observation
  + repository-backed write authority evidence
  -> Accepted Decision write authority Gate

Gate qualified
  != witness committed
  != Accepted Decision created
  != current authority applied
```

である。

current inspected Repository evidenceではOperational Accepted Decision write authorityを選択しない。

production Accepted Decision store、transaction protocol、locking、approval UI、authority application、Command executionは追加しない。

## 12. 次

次段ではsingle-qualified write authorityをsyntheticに与えた上で:

```text
explicit commit witness
  + exact R3AO WitnessIdentity
  + exact write-authority evidence
  + expected decision revision
  + write provenance
  -> test-only commit candidate
```

をPressureする。

```text
commit candidate
  != committed Accepted Decision
  != current authority applied
```

を維持する。
