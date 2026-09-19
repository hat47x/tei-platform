# Actor Context R3AK — Authority Information Request Identity / Response Correlation Boundary

- Status: ISSUE-000053 / Phase 3 request identity and correlation pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AJ_AUTHORITY_INFORMATION_REQUEST_BOUNDARY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3AI_READ_ONLY_AGENT_CONSUMPTION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3AH_CROSS_AUDIENCE_OUTCOME_DRILLDOWN_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3M_OPERATIONAL_STOP_EXPLANATION_HANDOFF_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AJではmissing authority informationを問い合わせるscopeを作ったが、delivery / routing / owner selectionは導入しなかった。

R3AKでは:

```text
request identity
request revision identity
response correlation
```

だけを追加し、次を分離する。

```text
correlate response
  != approve response
  != accept authority automatically
  != resolve stop
```

## 2. RequestIdentityとRequestRevisionIdentity

test-onlyに二層を分ける。

### RequestIdentity

semantic question identity。

少なくとも:

- RequestClass
- BusinessAddress
- StopStage
- StopClass
- MachineReasons
- InformationNeedClasses
- EvidenceNeedClasses

から正規化して導出する。

次は含めない。

- routing destination
- recipient identity
- source locator order
- delivery metadata
- transient lifecycle state

### RequestRevisionIdentity

同じsemantic requestでも、current context revisionを区別する。

RequestIdentityに加えて:

- KnownSourceRefs
- DrillDownRefs
- HandoffRecipientStatus

を正規化して導出する。

```text
same request identity
  + changed source / recipient context
  -> different request revision
```

とする。

## 3. lifecycleはdelivery stateではない

R3AKのtest-only lifecycleは最小限:

```text
prepared
response-correlated
```

だけをPressureする。

`prepared` はrequest projectionが構築済みという意味であり:

- sent
- delivered
- routed
- assigned

を意味しない。

`response-correlated` も:

- accepted
- approved
- sufficient
- resolved
- applied

を意味しない。

## 4. response correlation input

test-only response witnessは:

- RequestIdentity
- RequestRevisionIdentity
- ResponseRef
- ProvidedInformationClasses
- EvidenceRefs

だけを持てる。

response payloadのauthority valueをR3AKで解釈しない。

```text
response references request
  != response content authoritative
```

である。

## 5. current revisionだけをcorrelateする

responseの:

- RequestIdentity
- RequestRevisionIdentity

がcurrent request recordと両方一致するときだけcorrelateする。

identity mismatch / revision mismatchはfail closedする。

これにより旧request revisionへのresponseをcurrent stateへsilent applyしない。

## 6. information classの一致はacceptance条件ではない

ProvidedInformationClassesはcorrelation witnessとして保持できる。

しかし:

```text
provided information class matches need
  != evidence sufficient
  != authority accepted

provided information class differs
  != response uncorrelated
```

である。

request identity / revisionが一致していればresponse自体はcorrelateできる。

内容のadequacy / authority acceptanceは次段以降の別boundaryである。

## 7. recipient unresolvedでもcorrelationできる

current R3AJではrecipient statusはunresolved。

responseがどのtransport / recipient経由で来たかをR3AKは扱わない。

```text
response correlation
  != recipient validation
  != sender authority validation
```

sender / channel / routing metadataは追加しない。

## 8. source locator change

semantic request scopeが変わらずSourceRefs / DrillDownRefsだけが変わる場合:

```text
RequestIdentity = same
RequestRevisionIdentity = changed
```

とする。

旧revision responseをcurrent requestへcorrelateしない。

## 9. read-only correlation result

test-only correlation resultは:

- RequestIdentity
- RequestRevisionIdentity
- ResponseRef
- CorrelationStatus
- LifecycleState
- ProvidedInformationClasses
- EvidenceRefs
- AuthorityLimits

までを持てる。

```text
CorrelationStatus = matched-current-revision
LifecycleState = response-correlated
```

はauthority decisionではない。

## 10. 禁止するshortcut

R3AK record / response / correlation shapeは次を持たない。

- Destination / RoutingKey / Channel
- Recipient / RecipientID / Sender / SenderAuthority
- TicketID / DeliveryStatus
- Accepted / Approved / Sufficient
- Resolved / ResolutionComplete
- AuthorityValue / AcceptedAuthorityValue
- Apply / AutoApply
- ToolCall / Command / NextAction
- CanExecute / Authorized / Allowed
- Retry / Reauthorize
- Confidence / Priority / Severity

## 11. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ak_request_correlation_boundary_pressure_test.go`

Pressureするcase:

1. R3AJ requestからstable RequestIdentity / RequestRevisionIdentityを作る
2. input orderでidentityを変えない
3. SourceRefsだけ変わればRequestIdentityは同じ、RequestRevisionIdentityは変わる
4. matching identity + revision responseだけcorrelateする
5. stale revision responseはfail closed
6. different ProvidedInformationClassesでもcorrelation自体は可能
7. correlationでrequestをresolved / approved / acceptedへ変換しない
8. recipient unresolvedでもsender / routing identityを捏造しない
9. repeated correlationでauthority strengthを増やさない
10. shapeにdelivery / routing / execution shortcutを追加しない

## 12. current conclusion

R3AKで固定するのは:

```text
semantic request identity
  != request revision

response correlation
  != response acceptance

response-correlated
  != stop resolved
  != authority accepted
```

である。

production messaging、delivery tracking、recipient directory、ticketing、authority acceptance workflow、Command executionは追加しない。

## 13. 次

次段ではcorrelated responseを読む際に:

```text
response evidence observed
  != authority evidence qualified
  != Accepted Decision
```

を維持しながら、response evidence qualificationの入口だけをPressureする。
