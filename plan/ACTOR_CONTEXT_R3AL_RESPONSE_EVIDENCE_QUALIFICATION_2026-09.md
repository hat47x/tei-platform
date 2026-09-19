# Actor Context R3AL — Correlated Response Evidence Qualification Boundary

- Status: ISSUE-000053 / Phase 3 response-evidence qualification pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AK_REQUEST_CORRELATION_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AJ_AUTHORITY_INFORMATION_REQUEST_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AKではresponseをcurrent request revisionへcorrelateできるようにした。

R3ALではcorrelated responseが持つEvidenceRefをqualificationの入口へ渡す。

```text
response evidence observed
  != authority evidence qualified
  != Accepted Decision
```

を固定する。

## 2. observed evidence

R3AK correlation resultの:

- RequestIdentity
- RequestRevisionIdentity
- ResponseRef
- ProvidedInformationClasses
- EvidenceRefs

からtest-only evidence assessmentを作る。

EvidenceRefがあることは:

```text
evidence observed
  != evidence qualified
  != authority true
```

である。

## 3. evidence need class overlap

R3AJ requestのEvidenceNeedClassesとresponseのProvidedInformationClassesのintersectionをtest-onlyに記録できる。

```text
information class matches requested evidence need
  != evidence qualified
```

class overlapはscope整合の説明であり、authority decisionではない。

## 4. qualification witness

R3ALではtest-only inputとして明示的なqualification witnessだけを認める。

```text
EvidenceRef
QualificationStatus
QualificationSourceRef
```

current pressureでは:

- no witness -> `qualification-unresolved`
- explicit witness -> `explicit-evidence-qualified`

とする。

ただし:

```text
explicit-evidence-qualified
  != authority accepted
  != asserted authority value accepted
  != stop resolved
```

である。

qualification witness自身のSourceRefが空ならfail closedする。

## 5. exact EvidenceRef binding

qualification witnessはexact EvidenceRefへだけ適用する。

類似locator、same response、same information classだけで別EvidenceRefへqualificationを伝播しない。

current responseに存在しないEvidenceRefへのwitnessはfail closedする。

## 6. request revisionを再確認する

R3ALはR3AK request recordとcorrelation resultの:

- RequestIdentity
- RequestRevisionIdentity

が一致することを要求する。

stale request revisionのqualificationをcurrent requestへ移植しない。

## 7. multiple evidence

複数EvidenceRefは独立assessmentとして保持する。

一つがqualifiedでも他をqualifiedへ昇格させず、winner / primary evidenceを選ばない。

duplicate exact EvidenceRefはnormalized identityとして一つにできるが、conflicting qualification witnessはfail closedする。

## 8. evidenceなしresponse

response correlationが成立してもEvidenceRefsが空ならassessmentは空でよい。

```text
response correlated
  + no evidence ref
  != authority resolved
```

generic success / failureへ丸めない。

## 9. 禁止するshortcut

R3AL assessmentは次を持たない。

- AuthorityValue
- AcceptedAuthorityValue
- AcceptedDecision
- Approved
- Sufficient
- Resolved / ResolutionComplete
- Fresh / Safe / Reusable
- CanExecute / Authorized / Allowed
- Apply / AutoApply
- ToolCall / Command / NextAction
- Retry / Reauthorize
- Confidence / Priority / Severity / Winner

## 10. test-only source

- `tools/reference-business-procedure-check/actor_context_r3al_response_evidence_qualification_pressure_test.go`

Pressureするcase:

1. correlated response EvidenceRefをobserved assessmentへ変換する
2. evidence need class overlapを保持してもqualification-unresolvedのままにする
3. explicit qualification witnessだけがexplicit-evidence-qualifiedへ変える
4. qualification SourceRef欠落はfail closed
5. responseにないEvidenceRefへのwitnessはfail closed
6. request identity / revision mismatchはfail closed
7. multiple evidenceを独立assessmentとして保持する
8. one qualified evidenceを他Evidenceへ伝播しない
9. evidenceなしresponseをresolvedへ変換しない
10. input order / repeated assessmentでidentityを変えない
11. acceptance / execution shortcutを追加しない

## 11. current conclusion

R3ALで固定するのは:

```text
correlated response
  -> observed evidence
  -> explicit evidence qualification

evidence qualification
  != authority acceptance
  != Accepted Decision
  != side effect
```

である。

production evidence registry、trust store、signature validation、authority acceptance workflow、Command executionは追加しない。

## 12. 次

次段ではqualified evidenceがauthority valueをassertする場合でも:

```text
asserted authority value
  != accepted authority value
```

を維持し、assertion provenanceとAccepted Decisionを分離してPressureする。
