# Second Application Reference — S3 Destination Decision Packet

- Status: Source-defined / Shared Acceptance not attempted
- Date: 2026-09-19
- Primary workstream: ISSUE-000040
- Inputs:
  - `plan/SECOND_APPLICATION_GATE_C_ENTRY_2026-09.md`
  - `plan/SECOND_APPLICATION_S1_S2_QUALIFICATION_PORTABILITY_2026-09.md`
- Next step: S4 composition candidate
- Canonical second Application created: No
- Shared Acceptance attempted: No
- Environment binding created: No
- Target implementation started: No

## 1. 目的

S1/S2でqualifiedになったshared Customer knowledgeを使い、第二Application固有の意味を一つのdestination decision packetへまとめる。

ここで決めるのは、S4のcomposition candidateを一意に組み立てるためのdestination-owned decisionである。

```text
S3 destination decision
  != Shared Acceptance
  != authoritative source
  != Environment binding
  != execution authority
```

Pre-Gateで使った `repair_intake-test-only` はscenario labelのままであり、その文字列を自動的にCanonical identityへ昇格させない。

## 2. Decision summary

### Application identity candidate

```text
application id candidate = repair_intake
decision status          = proposed-for-shared-acceptance
decision basis           = explicit S3 destination decision
```

`repair_intake` はS3で明示的に選ぶdestination candidateであり、`repair_intake-test-only`からの機械的昇格ではない。

Shared Acceptanceまではauthoritative Application identityではない。

### Business context

最初のReferenceのBusiness contextを、修理受付時に既存CustomerをIDで参照する文脈に限定する。

```text
business context = repair intake
```

Customer maintenance、Customer write、修理工程全体のworkflowは含めない。

### Business Relation

```text
relation candidate = repair_intake.intake.customer
```

意味:

> repair intakeが参照するCustomer

originの `sales.order.customer` はコピーしない。

### Application Pattern

```text
pattern candidate = repair_intake.customer_reference
kind              = reference
```

origin `order_entry.customer_reference` はprovenanceとして保持するが、destination Pattern authorityにはしない。

### Datasource alias

```text
datasource alias candidate = customer_master
```

origin alias `master` を継承しない。

aliasはApplication-local nameであり、Database identity `master_data` と同一視しない。

### Database requirement

S1/S2でqualifiedになったshared authorityをS4候補で利用する。

```text
target database = master_data
access          = read
contract        = master_data.customer ^2
query           = master_data.customer/find_by_id
```

これらはshared authorityであり、destination copyを作らない。

## 3. Authorization decision

Customer lookupは、Contractがread-onlyであることだけを理由にpublic actionとは扱わない。

S3では次を決める。

```text
authorization requirement = required-current-authority-check
principal identity         = unresolved
resource identity          = unresolved
permission binding         = unresolved
```

意味:

- effect直前のcurrent authority確認を必要とする
- planner / AI / UIが「読める」と判断したことをauthorizationにしない
- S3でprincipal / Resource / Permission identityを捏造しない
- Shared AcceptanceやApplication sourceだけで将来のread authorityを永久付与しない

exact principal / Resource / Permissionの解決は、そのauthority ownerへ渡す。

```text
authorization required
  != authorization granted
```

を維持する。

## 4. Business Procedure decision

第一Referenceは単一のread-only Customer lookupであり、Business Procedureを新設しない。

```text
business procedure = not-required-for-first-reference
```

これはTEI一般でlookupにProcedureが不要という規則ではない。

今回のReferenceでProcedureを追加しない理由は、意味上必要なstep sequence / handoff / action chainがまだ存在しないためである。

複数stepのrepair workflowを後から追加する場合は別Evidenceで再評価する。

## 5. Environment binding policy

S3ではEnvironmentのbinding値を選ばず、binding policyだけを決める。

development / productionとも:

```text
destination binding = explicit
copy origin binding = forbidden
binding target       = unresolved until S7
provider             = unresolved until S7 verification
```

S7では少なくとも:

- destination Application = `repair_intake`
- destination alias = `customer_master`
- selected instance exists
- selected instance definition = `master_data`
- provider is supported
- Environment authorityで明示的にaccepted

を確認する。

current `order_entry.master -> master_data.*` はEvidenceであってdestination configurationではない。

## 6. First Web Target presentation scope

第一Targetではread-only lookupの意味だけを再現する。

最低surface:

```text
input:
  customer id

action:
  lookup

result:
  zero-or-one customer
  or not-found
```

境界:

- Customer editなし
- Customer create/deleteなし
- repair workflow mutationなし
- datasource / Contract identityをUI上のBusiness meaningへ露出する必要なし
- HTML / route / selector / DOM identityをCanonical meaningへ昇格させない

Target固有のinteraction realizationはS8で決める。

## 7. S1/S2から保持するshared authority

```text
semantic-data:customer.customer
semantic-type:customer.customer_id
database-contract:master_data.customer
database-contract-query:master_data.customer/find_by_id
```

origin provenance:

```text
application-pattern:order_entry.customer_reference
```

保持しないもの:

- origin Relation `sales.order.customer`
- origin alias `master`
- origin Environment binding
- origin Target implementation

## 8. S3で解いたもの / 残すもの

### S3で解いたdestination decision

- Application ID candidate = `repair_intake`
- Business context = repair intake
- Business Relation candidate = `repair_intake.intake.customer`
- Pattern candidate = `repair_intake.customer_reference`
- datasource alias candidate = `customer_master`
- Authorization requirement = current-authority check required
- Business Procedure = first Referenceでは追加しない
- Environment binding policy = destination explicit binding / origin copy禁止
- First Web Target scope = Customer IDによるread-only lookup

### S3後も未解決

- Shared Acceptance result
- authoritative Application / Pattern source
- exact authorization principal
- exact Resource / Permission identity
- development binding target
- production binding target
- provider selection/verification
- Target implementation detail
- actual reload / Validation result
- actual Driver / Target Evidence
- change propagation Evidence

## 9. S4への入力

S4 composition candidateは、S3 decisionを通常の既存grammarへ写す。

概念形:

```text
application repair_intake {
    uses semantic customer

    database customer_master {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}

pattern repair_intake.customer_reference reference {
    relation repair_intake.intake.customer
    datasource customer_master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}
```

S4はProposalであり、まだauthoritative sourceへ書かない。

## 10. Authority boundary

```text
S1/S2 applicable
  -> reusable shared knowledge exists
  != destination decision

S3 decision packet
  -> S4 can be composed deterministically
  != Shared Acceptance

Shared Acceptance
  -> source authoring may proceed
  != Environment binding

Application accepted
  -> binding can be proposed
  != current execution authority
```

## 11. Test-only Source pressure

- `tools/reference-business-procedure-check/second_application_s3_destination_decision_pressure_test.go`

最低限次を固定する。

1. scenario labelからApplication IDを自動昇格しない
2. destination Application / Relation / Pattern / aliasをoriginから分離する
3. shared Customer authorityをcopyしない
4. Authorization requirementとgrantを分離する
5. Environment policyを解いてもbinding targetは選ばない
6. first Target scopeをread-only lookupへ限定する
7. S3でsource authoring / Acceptance / binding / Target実装を開始しない
8. Business Procedureを不要に増やさない

## 12. Evidence height

現段階はSource / Design Evidenceである。

actual focused `go test` PASSが得られた場合のみExecutable Evidenceへ昇格する。

S3 sourceの存在だけで、

- S4 composition Accepted
- Shared Acceptance PASS
- Application source作成済み
- reload / Validation PASS
- Environment binding成立
- Driver / Target replay成立

を主張しない。

## 13. Conclusion

第二Applicationのdestination-owned meaningは、S4を一意に組み立てられる粒度まで絞れた。

一方で、authority boundaryは越えていない。

次は同じISSUE-000040 workstreamでS4 composition candidateを作り、Shared Acceptanceへ渡す。
