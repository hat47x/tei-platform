# Second Application Reference — S1 Qualification / S2 Portability

- Status: Source-defined / destination decisions unresolved
- Date: 2026-09-19
- Primary workstream: ISSUE-000040
- Entry record: `plan/SECOND_APPLICATION_GATE_C_ENTRY_2026-09.md`
- Exploration owner: ISSUE-000047
- Portability owner: ISSUE-000053 Phase 5
- Composition owner after S2: ISSUE-000040
- Canonical second Application created: No
- Shared Acceptance attempted: No

## 1. Intent

最初のsecond-Application use caseは次に限定する。

> destination Business contextからCustomerをIDで参照したい。

`repair_intake-test-only`はscenario labelであり、Canonical Application IDではない。

## 2. S1 — candidate qualification

test-only source:

- `tools/reference-business-procedure-check/second_application_s1_s2_actual_pressure_test.go`

current Repository / Source Indexからread-onlyに導出する。

### Customer Semantic

```text
authority = semantic-data:customer.customer
qualification = applicable
reuse candidate = linked-reuse
```

discovery origin:

- `application-pattern:order_entry.customer_reference`

ただしorigin PatternはCustomer meaningのownerではない。

### Customer ID Type

```text
authority = semantic-type:customer.customer_id
qualification = applicable
reuse candidate = linked-reuse
```

current Canonical fact:

```text
customer.customer.customer_id
  -> type customer.customer_id
```

Customer ID Typeの共有はdestination UI/input formを決めない。

### Customer Database Contract

```text
authority = database-contract:master_data.customer
compatible major = 2
required access = read
qualification = applicable
reuse candidate = conditional-linked-reuse
```

これはS1/S2のrequirement probeであり、destination Application sourceでのDatabase requirement Acceptanceではない。

destination datasource aliasはここでは空のままにする。

### Customer Contract Query

```text
authority = database-contract-query:master_data.customer/find_by_id
qualification = applicable
reuse candidate = conditional-linked-reuse

input:
  id : customer.customer_id

result:
  customer.customer

cardinality:
  zero-or-one
```

Query identityを再利用してもorigin Application Patternを共有authorityへ昇格させない。

## 3. S2 — portability relation

### Shared linked authority

- `semantic-data:customer.customer`
- `semantic-type:customer.customer_id`
- `database-contract:master_data.customer`
- `database-contract-query:master_data.customer/find_by_id`

Semantic / Typeはlinked reuse、Contract / Contract Queryはcompatible major等のdestination conditionを伴うconditional linked reuseとして扱う。

### Origin Pattern

```text
origin:
  application-pattern:order_entry.customer_reference

qualification:
  applicable-as-provenance

reuse relation:
  adopted-local-variation

destination candidate:
  application-pattern:<destination>.customer_reference
```

origin Pattern自体をdestination linked dependencyにしない。

originの、

- Relation `sales.order.customer`
- datasource alias `master`

をdestinationへコピーしない。

### Environment

development / productionともorigin Evidenceは、

```text
order_entry.master
  -> master_data instance

reuse relation = rebind-required
qualification = destination-owned
destination binding = unresolved
```

同じDatabase instance / Providerを利用し得ることと、destination bindingを作成済みであることを分離する。

## 4. Unresolved destination Decisions

S1/S2では次を決めない。

- Canonical Application ID
- Business context
- Business Relation
- Business Procedure
- destination Pattern ID
- datasource alias
- Authorization requirement
- development Environment binding
- production Environment binding
- first Web Target surface

`repair_intake-test-only`や`<destination>.customer_reference`をAccepted identityへ昇格させない。

## 5. Gate C carry-forward

Gate Cから受け取った次のunresolvedは、S1/S2のSemantic qualification authorityへ変換しない。

- `actual-external-command-observation-not-acquired`
- `browse-environment-instance-binding-not-exposed-by-current-driver-api`
- `general-target-neutral-presentation-taxonomy-not-yet-defined`
- `target-capability-taxonomy-remains-candidate-until-second-target`

これらは必要なownerへcarry-forwardする。

特にFirst Web TargetのHTTP route / HTML / CSS / synthetic fixture等からsecond Application meaningを逆設計しない。

## 6. Authority boundary

```text
Gate C Ready
  -> second Application entry allowed
  != candidate applicable

candidate applicable
  -> S1 qualification
  != Shared Acceptance

portability relation
  -> S2 reuse meaning
  != destination source

origin Pattern provenance
  != destination Pattern authority

origin Environment binding Evidence
  != destination Environment binding
```

## 7. S1/S2 conclusion

S1/S2で利用可能なshared Customer knowledgeは明確になった。

一方でdestination meaningはまだ未決定である。

次段S3で扱うのは、

```text
Application identity
Business context / Relation
Pattern identity
datasource alias
Authorization requirement
Environment binding policy
first-target presentation scope
```

である。

S1/S2の結果だけからS3 Decisionを自動生成しない。

## 8. Evidence height

現段階で追加したものはSource / Design Evidenceである。

actual `go test` PASSを取得できた場合にのみExecutable Evidenceへ昇格する。

S1/S2 sourceの存在だけで、

- second Application Acceptance
- source reload
- Driver replay
- First Web Target replay
- change propagation pass

を主張しない。
