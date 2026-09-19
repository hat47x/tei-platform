# Actor Context R3AJ — Authority Information Request / Recipient Non-inference Boundary

- Status: ISSUE-000053 / Phase 3 authority-information request pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AI_READ_ONLY_AGENT_CONSUMPTION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3M_OPERATIONAL_STOP_EXPLANATION_HANDOFF_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3P_PURPOSE_BOUND_STOP_PROJECTION_OMISSION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AIではAgentが未解決状態をread-onlyに理解しても、remediation / invocation / executionを選ばない境界を固定した。

R3AJでは、外部情報が必要な場合でも:

```text
request for missing authority information
  != request to execute remediation

recipient status unresolved
  != guessed owner
```

を維持する。

R3AJはPhase 4 Actionへ進まない。

## 2. 入力authority

R3AJはtest-onlyに次の二つを照合してinformation request projectionを作る。

```text
R3AI read-only reasoning
  + R3P Agent resolution-inspection view
  -> authority-information request projection
```

R3AIから:

- freshness stop reason
- resolution need classes
- reevaluation / reuse authority status
- exact outcome / trace context

を読む。

R3Pから:

- BusinessAddress
- StopStage / StopClass
- MachineReasons
- ResolutionRequested
- HandoffRecipientStatus
- SourceRefs / DrillDownRefs

を読む。

どちらか一方からmissing recipientやmissing authority valueを補完しない。

## 3. current freshness-stop request

current freshness stopでは:

```text
BusinessAddress
  = order-entry.order-handling/BP-06

resolution need
  = authorization-evaluation-freshness-authority

resolution requested
  = true

handoff recipient status
  = unresolved
```

である。

R3AJはこれを:

```text
request class
  = authority-information

information need class
  = authorization-evaluation-freshness-authority

evidence need class
  = authorization-evaluation-freshness-authority

recipient status
  = unresolved
```

としてtest-onlyにmaterializeできる。

ここでinformation need / evidence needはmissing valueそのものではない。

## 4. evidence needとcandidate valueを分離する

R3AJは「何のauthority information / evidenceが不足しているか」は表現する。

しかし:

```text
evidence need class known
  != missing authority value known
  != candidate value known
```

である。

current freshness stopから:

- reevaluate
- reuse
- deny
- allow
- execute
- ORDER_EDITOR

等をrequested valueとして生成しない。

## 5. recipient statusとrecipient identityを分離する

R3M / R3Pのcurrent sourceでは:

```text
handoff recipient status = unresolved
```

である。

R3AJはこれをそのまま保持する。

次を推測しない。

- security team
- application owner
- business owner
- human operator
- administrator
- sales operator

```text
resolution requested
  != recipient known
  != recipient selected
```

である。

## 6. request scope

test-only request scopeは少なくとも:

- BusinessAddress
- StopStage
- StopClass
- MachineReasons
- InformationNeedClasses
- EvidenceNeedClasses
- known SourceRefs
- DrillDownRefs

を保持できる。

これにより外部側は「何について情報が不足しているか」を確認できる。

ただしscopeはrouting hintではない。

```text
request scope
  != routing destination
  != owner mapping
  != workflow step
```

である。

## 7. cross-view agreementを必須にする

R3AIとR3PでResolutionNeedClassesが一致しない場合はfail closedする。

```text
reasoning need classes
  != stop projection need classes
  -> request build error
```

とする。

片方を優先してsilent repairしない。

同様にR3P側で:

- ResolutionRequested = false
- HandoffRecipientStatus = empty
- BusinessAddress = empty

ならinformation requestを作らない。

## 8. deniedはinformation requestへ変換しない

exact current denied ObservationではR3O / R3Pどおり:

```text
runtime outcome = denied
resolution need classes = []
resolution requested = false
```

である。

R3AJはdeniedを:

- ask security team
- request approval
- ask for new policy
- retry authorization

へ変換しない。

deniedはmissing authority information requestではない。

## 9. known evidenceを問い合わせ対象へ昇格させない

SourceRefs / DrillDownRefsはcurrent known contextである。

```text
known source ref
  != missing evidence
```

R3AJは既知sourceを保持するが、その存在だけで:

- authority resolved
- recipient resolved
- candidate value accepted

とはしない。

## 10. read-only shape

R3AJ information request projectionは次を持たない。

- Recipient / RecipientID
- HandoffOwner
- SuggestedOwner
- CandidateValue
- ProposedAuthorityValue
- ToolCall
- InvocationPlan
- Command
- SuggestedAction
- NextAction
- Remediation
- Retry / Reauthorize / Rebuild
- ApprovalRequest
- CanExecute
- Authorized / Allowed
- Continue
- Confidence / Priority / Severity

Request自体をAction commandへ変換しない。

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3aj_authority_information_request_boundary_pressure_test.go`

Pressureするcase:

1. current freshness stopからauthority-information requestを作れる
2. R3AI / R3P ResolutionNeedClassesをcross-checkする
3. information need / evidence need classを保持する
4. HandoffRecipientStatus = unresolvedをそのまま保持する
5. recipient identity / owner / candidate authority valueを生成しない
6. denied Observationからinformation requestを作らない
7. resolution need mismatchはfail closed
8. missing recipient status / BusinessAddress / source traceはfail closed
9. repeated build / input orderでnormalized request identityを変えない
10. request shapeにremediation / execution shortcutを追加しない

## 12. current conclusion

R3AJで固定するのは:

```text
unresolved read-only understanding
  + explicit resolution-request state
  -> scoped authority-information request

authority-information request
  != remediation request
  != action request
  != owner selection
```

である。

production messaging protocol、ticket creation、notification transport、recipient directory、workflow engine、tool invocationは追加しない。

## 13. 次

次段では、recipientがunresolvedのままでもinformation requestを外部へhandoffするために必要な:

```text
request identity
request lifecycle state
response correlation
```

を、delivery / routing / owner selectionと分離してPressureする。

特に:

```text
correlate response
  != approve response
  != accept authority automatically
```

を維持する。
