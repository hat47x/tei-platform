# Actor Context R3 — Read-only AI Business Actor Challenge

- Status: ISSUE-000053 / Phase 3 first challenge / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md`
- Related: `plan/ACTOR_CONTEXT_R0_SOURCE_MAP_2026-09.md` / ADR-0028 / RQ-018〜RQ-020 / RQ-025 / RQ-030
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3では、AI Business ActorがTEI成果物を読んで業務Actionを理解し、machine-readableなread経路まで到達できるかをPressureする。

最初のActionは`order_entry` Business Procedure ReferenceのBP-02とする。

```text
BP-02
  confirm the customer associated with the order
```

このActionはcurrent Reference上、

```text
BP-02
  -> order_entry.customer_reference
  -> sales.order.customer
  -> datasource master
  -> master_data.customer
  -> find_by_id
```

へ辿れる。

R3ではここからさらに、

- query input mapping
- result Data
- read-only access
- authority limit
- missing runtime context

をAgent向けのread intentへ整理する。

ただしactual query invocationはまだ行わない。

## 2. Current static route

### Business Action

`reference/business-system-design/order-entry-business-procedure.json`

```text
BP-02
  businessAction = confirm the customer associated with the order
  realization = application-use
  applicationPattern = order_entry.customer_reference
```

fixtureは`canonical=false`である。

### Pattern

`reference/workspace/applications/order-entry/patterns/customer-reference.tei`

```text
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}
```

### Application access

`order_entry.master access = read`

### Database Contract

```text
contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
        input id type customer.customer_id
        result customer.customer zero-or-one
    }
}
```

このrouteはsemantic / application / database contract上、read intentとして説明できる。

## 3. Static resolutionとruntime authorityを分ける

R3でAgentへ渡せるstatic knowledge:

- Business Action meaning
- Pattern identity
- Relation identity
- datasource
- Contract
- Query
- input semantic attribute → query input mapping
- expected result Data
- result cardinality
- Application access = read

一方、static sourceだけでは次を確定しない。

- current order instance
- current customer ID value
- AI Agentのprincipal identity
- current user / organization / role
- Authorization decision
- credential / secret availability
- Provider availability
- actual query success
- customer existence
- Business Action completion

```text
route resolvable
  != invocation authorized
  != input available
  != execution successful
  != business action complete
```

## 4. R3のread intent

production schemaではない。

```text
ReadIntentProbe

  business
    procedure
    step
    action

  semantic route
    pattern
    relation
    target semantic attribute

  application route
    datasource
    accepted access

  database contract
    contract
    query
    query input
    input type
    result
    cardinality

  runtime gaps
    principal = unresolved
    input value = unresolved
    authorization = not evaluated
    resource / provider = not evaluated

  execution
    invocation attempted = false
    observation received = false
    business completion = unresolved

  source / evidence refs
  authority limits
```

これをproduction typeへ固定しない。

## 5. input mappingの意味

Patternには、

```text
customer.customer.customer_id -> id
```

というmappingがある。

これは、

> Relation target側のCustomerIdをContract Queryの`id`へ渡す

というApplication intentである。

しかしstatic contextには「現在対象となるorder instanceのcustomer ID値」まではない。

したがってAgentは、

```text
I know which semantic value is required
```

とは言えても、

```text
I have the value and may invoke now
```

とは言えない。

この区別はAgentによる架空input生成を防ぐ。

## 6. Authorizationについて

`order_entry.master access = read`は、Application設計上のdatasource accessである。

これは、

- AI Agent principalのAuthorization
- current userのAuthorization
- secret access
- network permission

を意味しない。

```text
Application access
  != Agent authorization
```

R3ではprincipal / authorizationを`unresolved / not evaluated`として保持する。

将来Agent principalとAuthorizationを接続する場合も、Business Procedure actor labelから暗黙にrole / principalを生成しない。

## 7. read-onlyでもObservationが必要

Contract Queryがread-onlyであっても、実行した結果はObservation / Evidenceである。

```text
query invocation
  -> execution result / error
  -> observed customer or absence
```

そして、

```text
customer record observed
  != BP-02 business action automatically complete
```

である。

たとえば、

- customerが存在しない
- Relationがstale
- Business context上追加確認が必要

可能性がある。

R3 sourceではactual executionを行わないため、completionは常にunresolvedのままにする。

## 8. GUIを経由しないmachine consumer

R3 Agentは`customer_reference`の画面をクリックする必要はない。

Pattern / Contract Queryをmachine-readableに解決できるなら、将来的には明示されたCapability / Query interfaceへ直接接続できる。

```text
Human
  -> Presentation Target
  -> customer reference view

AI Business Actor
  -> Operational Projection
  -> accepted machine interface
```

ただしcurrent R3ではtransport / MCP / API contractを新設しない。

「GUIを使わない」ことと「無権限で直接DBへ接続する」ことも別である。

## 9. R3 success conditions

### S1 — Business Actionからread routeへexactに辿れる

BP-02 → Pattern → Relation → datasource → Contract Query。

### S2 — input mappingを保持する

`customer.customer.customer_id -> id`を保持する。

### S3 — expected resultを保持する

`customer.customer zero-or-one`を保持する。

### S4 — read-only accessを拡張しない

`master access=read`を保持し、writeへ変換しない。

### S5 — missing runtime valuesを発明しない

customer ID value、principal、credential等をstatic sourceから捏造しない。

### S6 — authorizationを推測しない

Application accessからAgent authorization allowを導かない。

### S7 — invokeしない

R3 Source Evidenceではexternal callを行わない。

### S8 — Business completionを推測しない

route resolutionだけでBP-02 completeにしない。

## 10. test-only pressure

test-only typeには概念的に次を持つ。

- Procedure / Step / BusinessAction
- Pattern / Relation
- Datasource / Access
- Contract / Query
- InputSemanticAttribute / QueryInput / QueryInputType
- Result / Cardinality
- RuntimeGaps
- InvocationAttempted
- ObservationReceived
- CompletionUnresolved
- SourceRefs / AuthorityLimits

`InvocationAttempted`と`ObservationReceived`は必ずfalse。

`CompletionUnresolved`はtrue。

次のshortcut fieldは置かない。

- Authorized
- CanExecute
- Principal
- CustomerIDValue
- Credential
- QuerySucceeded
- BusinessCompleted
- AgentTrust
- Confidence

## 11. R3後に必要なEvidence

R3 static sourceが成立しても、AI Agentが実際に業務を遂行できるとは言えない。

次の段階が必要になる。

1. actor / principal context
2. Authorization / Permission評価
3. Resource / Provider availability
4. exact input instance
5. read-only actual invocation
6. Observation / Evidence mapping
7. Business continuation / handoff

これらを一度に実装しない。

## 12. Evidence高さ

```text
Design Evidence
  read-only Agent information boundary

Source Evidence
  static read intent pressure

Executable Evidence
  test PASS

Integration Evidence
  actual read-only invocation under explicit authority

Operational Evidence
  repeated business action with correct observation / handoff
```

本R3ではDesign / Sourceまでを対象とする。
