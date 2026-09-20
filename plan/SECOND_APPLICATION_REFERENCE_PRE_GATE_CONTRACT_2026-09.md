# Second Application Reference — Pre-Gate Contract

- Status: Pre-Gate-C contract / test-only
- Date: 2026-09-19
- Related: ISSUE-000040 / ISSUE-000047 / ISSUE-000053 Phase 5 / ISSUE-000039
- Parent: `plan/KNOWLEDGE_PORTABILITY_HANDOFF_2026-09.md`
- Product Gate impact: None
- Canonical Application impact: None
- Production schema impact: None

## 1. 目的

Gate C後に予定する第二Application Referenceを、Gate C前に実装してしまわず、着手時に必要な意味境界と受入条件だけを固定する。

この契約は第二Applicationそのものではない。

```text
Pre-Gate Contract
  != Application source
  != Shared Acceptance
  != Gate C completion
```

最初のscenario labelとして `repair_intake-test-only` を利用するが、これはProduct上のApplication名を採用したことを意味しない。

## 2. 第一use caseの範囲

最初のReferenceはread-only customer lookupに限定する。

目的:

> destination Business contextから既存Customer意味を参照し、既存read-only Contractを利用できる構成を作る。

共有候補:

- `customer.customer`
- `customer.customer_id`
- `master_data.customer ^2`
- `master_data.customer/find_by_id`

destination固有候補:

- Application identity
- Business context / Relation
- Application Pattern
- datasource alias
- Application Database requirement
- Environment binding
- Presentation / Target integration

## 3. 三層を分ける

### 3.1 Shared authority

同じ意味を再利用する場合、destination copyを作らず既存Canonical subjectを参照する。

```text
semantic-data:customer.customer
database-contract:master_data.customer
```

### 3.2 Destination-owned Application meaning

Application固有の意味はdestinationが所有する。

例:

```text
<destination>.customer_reference
datasource alias = customer_master
destination-specific Relation
```

origin `order_entry.customer_reference`をshared Pattern authorityにはしない。

### 3.3 Environment-owned configuration

環境BindingはApplication sourceともshared Semantic authorityとも別である。

```text
development:
  bind <destination>.customer_master -> <selected instance>

production:
  bind <destination>.customer_master -> <selected instance>
```

originの、

```text
bind order_entry.master -> master_data.dev
bind order_entry.master -> master_data.prod
```

をcopyしない。

## 4. Gate C前に確定してよいもの

current Repositoryから次を確認できる。

### Customer Semantic

`customer.customer`はcurrent authoritative Semantic sourceに存在する。

### Customer Contract

`master_data.customer v2.4`は、

- read `customer.customer`
- query `find_by_id`
- input `id type customer.customer_id`
- result `customer.customer zero-or-one`

を提供する。

### Origin Pattern provenance

`order_entry.customer_reference`は、

- reference Pattern
- Relation = `sales.order.customer`
- datasource = `master`
- Contract Query = `master_data.customer/find_by_id`

という既存利用例である。

ただしPatternはApplication-ownedなので、destinationの正本ではない。

### Environment Evidence

development / productionには`master_data` instanceが存在し、current providerはOracleである。

しかし、このEvidenceだけでdestination bindingをauthorしない。

## 5. Gate C前に未決定のまま残すもの

- 第二ApplicationのCanonical ID
- Business Procedure / Action
- destination-specific Relation
- destination Pattern ID
- datasource aliasの最終名称
- Application Database requirementの採用
- development / production binding
- 第一Web Targetでのsurface
- Authorization requirement
- customer maintenance責務
- write Capability
- second Target
- production deployment形態

```text
known reusable knowledge
  != destination design accepted
```

を維持する。

## 6. Pre-Gate Contractの状態

test-only Probeでは必ず次を保持する。

```text
requiredGate = Gate C Ready

canonicalApplicationCreated = false
sourceAuthoringAttempted = false
acceptanceAttempted = false
environmentBindingCreated = false
targetImplementationStarted = false
```

これは「何もしない」ためのfieldではなく、Gate境界をReferenceで破っていないEvidenceである。

## 7. Qualification

### Positive

test-only destination profileが明示的に、

- Customer Semantic = `customer.customer`
- Database = `master_data`
- Contract = `master_data.customer`
- compatible major = 2
- access = read

を選んだ場合:

```text
customer.customer
  -> linked reuse / applicable

master_data.customer
  -> conditional linked reuse / applicable
```

とできる。

### Unknown

Database / Contractが未選択なら、

```text
Customer Semantic
  -> applicable candidate

Contract
  -> unknown
```

を維持する。

origin Applicationの選択を暗黙継承しない。

### Incompatible

Contract major不一致やreadを満たさないaccessならContractはincompatible。

ただし、

```text
Contract incompatible
  != Customer Semantic incompatible
```

である。

## 8. Application Patternの扱い

origin Patternはprovenanceとしてのみ利用する。

```text
order_entry.customer_reference
  -> provenance

<destination>.customer_reference
  -> destination-owned candidate
```

次をcopyしない。

- `sales.order.customer`
- datasource alias `master`
- order_entry ownership
- origin Environment binding

reference Patternという設計経験や、Contract Query input mappingの考え方は再利用候補にできる。

## 9. Change propagation contract

actual change impact engineではなく、第二Application Referenceが成立した後に検証すべき期待値を先に固定する。

| Change | Expected effect |
|---|---|
| `customer.customer` meaning change | linked destination consumerをrequalification |
| `master_data.customer` compatible minor change | compatibility条件に基づき再評価 |
| `master_data.customer` major change | compatible-major requirementを再評価 |
| origin `order_entry.customer_reference` change | destination local variationへprovenance drift |
| origin `order_entry` datasource alias change | destination aliasへ伝播しない |
| origin development / production binding change | destination bindingへ伝播しない |
| destination Pattern change | originへback-propagateしない |
| unrelated order_entry Pattern change | destinationをstale-allにしない |

このmatrixをISSUE-000039の実装範囲へ先回りして組み込まない。

ISSUE-000039がGate C後に提供するimpact解析を利用可能になった場合は、同じexpected effectを実Evidenceで比較する。

## 10. Acceptance後の最小source shape候補

Gate C後に初めてProposalへ進める概念形:

```text
application <accepted-destination-id> {
    uses semantic customer

    database customer_master {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}
```

Pattern:

```text
pattern <accepted-destination-id>.customer_reference reference {
    relation <destination-owned-relation>
    datasource customer_master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}
```

これはsyntax採用案ではなく、current Application / Pattern grammarで表現可能な最小構成を確認するための概念例である。

Gate C前にはsource fileを作らない。

## 11. Environment handoff

Gate C後のApplication ProposalがAcceptanceされた後でも、Environment bindingは別のauthorityとして扱う。

```text
Application accepts database requirement
  != Environment binding exists
```

各Environmentで、

- instance exists
- instance definition matches `master_data`
- provider is supported
- destination Application / alias pair is explicit

を確認してからBinding Proposalへ進む。

## 12. Driver / Targetの境界

ISSUE-000040の完了には最終的にDriverと第一Web Targetでの再現が必要だが、Pre-Gate Contractでは実装しない。

Gate C前:

```text
shared meaning / composition contract only
```

Gate C後:

```text
destination source
  -> reload / validation
  -> Driver
  -> First Web Target
  -> change propagation Evidence
```

とする。

## 13. AI Builderへの利用

この契約はAIに第二Applicationのコードを今生成させるためのpromptではない。

Gate C後のBuilder Contextへ、

- shared authority candidates
- destination-owned decisions
- unresolved
- provenance
- acceptance boundary
- change propagation expectations

を渡すための事前設計である。

AIはGate C Readyを観測していない状態でApplication sourceをauthorしない。

## 14. Test-only Pressure

`tools/reference-business-procedure-check/second_application_reference_pre_gate_pressure_test.go`

で次をPressureする。

1. current Customer Semantic / Contractをexact sourceから解決する
2. shared authority candidateをdestination copyにしない
3. Application Patternをlocal variationとする
4. Environment bindingをrebind-requiredとする
5. Gate C前のCanonical Application / source / Acceptanceをfalseに固定する
6. unresolved destination choicesを保持する
7. expected change propagationを分類する
8. one boolean `Ready` / `Portable` / `AutoCreate`へ畳まない

## 15. Existing schema sufficiency pressure

第二Applicationの最小構成が既存grammarだけで表現できるかは、

- `plan/SECOND_APPLICATION_SCHEMA_SUFFICIENCY_PRESSURE_2026-09.md`
- `tools/reference-business-procedure-check/second_application_schema_sufficiency_pressure_test.go`

でPressureする。

current hypothesis:

```text
existing Application grammar
+ existing reference Pattern grammar
+ existing Environment binding grammar
  -> first read-only second Applicationを表現可能
```

この仮説が崩れるEvidenceが得られるまでは、

- Knowledge Package
- `portable` declaration
- Pattern inheritance
- override DSL
- auto-sync directive

をCanonical grammarへ追加しない。

```text
planning portability relation
  != Canonical portability syntax
```

を維持する。

## 16. Pre-Gate完了条件

- [ ] shared / destination-owned / Environment-ownedを分離した
- [ ] Gate C Readyを実装着手前提として明示した
- [ ] test-only destinationをCanonicalへ昇格させない
- [ ] origin Patternをdestination authorityにしない
- [ ] Contract qualificationをdestination choiceへ依存させる
- [ ] unknown / incompatibleを保持する
- [ ] Environment bindingをcopyしない
- [ ] change propagation期待値を固定した
- [ ] ISSUE-000039の実装を先取りしない
- [ ] Gate C後の最小Proposal shapeを説明できる
- [ ] AI BuilderがGateを越えてauthoringしない境界を示した

## 17. Executable Evidence runner

Pre-Gate pressureの実行経路は、

- `plan/KNOWLEDGE_PORTABILITY_EXECUTABLE_EVIDENCE_2026-09.md`
- `.github/workflows/knowledge-portability-reference-evidence.yml`

へ接続する。

successful focused CI runはPre-Gate contractのExecutable Evidenceにはなるが、Gate C Ready、第二Application Acceptance、Environment Binding、Target Evidenceにはならない。

## 17. Gate C start packetへの移行

Gate C Readyが明示Evidenceで成立した後は、本Pre-Gate契約から、

- `plan/GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md`

へ移行する。

Pre-Gateで使用したsynthetic destination labelを、そのままCanonical Application IDへ自動採用しない。

## 17. Gate C後の着手順

```text
Gate C Ready
  ↓
ISSUE-000047
  candidate qualification
  ↓
ISSUE-000053 Phase 5
  portability relation
  ↓
ISSUE-000040
  destination composition candidate
  ↓
Shared Acceptance
  ↓
destination source
  ↓
reload / validation
  ↓
Environment Binding Proposal
  ↓
Driver / First Web Target
  ↓
change propagation Evidence
```

この順序を崩す必要が実Evidenceから生じた場合だけ見直す。
