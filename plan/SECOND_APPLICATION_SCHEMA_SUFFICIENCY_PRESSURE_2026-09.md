# Second Application Schema Sufficiency Pressure

- Status: Pre-Gate-C / test-only architecture pressure
- Date: 2026-09-19
- Parent: \`plan/SECOND_APPLICATION_REFERENCE_PRE_GATE_CONTRACT_2026-09.md\`
- Related: ISSUE-000040 / ISSUE-000053 Phase 5 / ADR-0017 / ADR-0018
- Production schema impact: None
- Canonical Application impact: None

## 1. 目的

第二Application Referenceを実装するために、新しいKnowledge Package、Portability declaration、inheritance / override構文を追加する必要が本当にあるかをGate C前にPressureする。

現在の仮説は次である。

> 最初のread-only second Applicationは、既存のApplication / Application Pattern / Database Contract requirement / Environment Bindingだけで表現可能である。

この仮説が成立する限り、Knowledge Portabilityはまず、

- discovery / qualification
- authority / provenance
- reuse relation
- freshness / drift

を扱うderived planning contextとして維持し、Canonical grammarへ新しいkindを追加しない。

## 2. test-only shape

scenario label:

\`\`\`text
repair_intake-test-only
\`\`\`

これは採用済みApplication名ではない。

### Application candidate

\`\`\`text
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
\`\`\`

既存grammarだけで次を表現する。

- shared Semantic namespace
- destination-owned datasource alias
- shared Database identity
- read-only access
- compatible Contract major

### Pattern candidate

\`\`\`text
pattern repair_intake.customer_reference reference {
    relation repair_intake.intake.customer
    datasource customer_master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}
\`\`\`

ここで、

- Pattern identityはdestination-owned
- Relation identityはdestination-owned
- Contract / Query / Customer attributeはshared authority
- origin \`sales.order.customer\`をcopyしない
- origin datasource alias \`master\`をcopyしない

ことを示す。

### Environment candidate

\`\`\`text
environment development {
    database-instance master_data.dev {
        definition master_data
        provider oracle
    }

    bind repair_intake.customer_master -> master_data.dev
}
\`\`\`

existing Environment grammarだけでdestination bindingを表現できる。

ただしPre-GateではこのsourceをRepositoryへ追加しない。

## 3. このPressureが確認すること

### A. Application compositionに新kindが不要

shared Semantic / Contractとdestination-owned aliasは既存Application grammarで表現できる。

### B. Pattern portabilityにinheritanceが不要

origin Patternを継承せず、destination-owned Patternがshared Contractを参照できる。

\`\`\`text
origin Pattern provenance
  != Pattern inheritance
\`\`\`

### C. Environment portabilityにcopy構文が不要

destination bindingは既存Environment bindingとして明示できる。

\`\`\`text
origin binding
  != destination binding
\`\`\`

### D. Portability metadataをCanonicalへ混ぜない

少なくともcurrent Referenceでは、

- \`knowledge-package\`
- \`portable\`
- \`inherit\`
- \`override\`
- \`adopt-from\`
- \`sync-from\`

のような専用Application authoring directiveを必要としない。

test-only sourceでは、これらをApplication grammarへ混ぜても受理されないことを確認する。

## 4. 「追加schema不要」の意味

このPressureは、

> TEIに将来いかなるKnowledge Portability schemaも不要

と結論するものではない。

意味するのは、

> 第一のsecond-Application use caseを実装する前に、専用schemaを追加するEvidenceがまだない

ということだけである。

将来、次のような要求が複数Referenceで繰り返し現れた場合は再評価する。

- cross-repository distribution
- reusable package versioning
- explicit compatibility contract
- provenance relationのauthoritative preservation
- organization間distribution
- accepted variation family
- migration / deprecation policy

## 5. current grammarでshared/localを分ける

### shared

existing Canonical identityをそのまま参照する。

\`\`\`text
customer
customer.customer
customer.customer.customer_id
master_data
master_data.customer
find_by_id
\`\`\`

### local

destinationが新しいidentityをauthorする。

\`\`\`text
repair_intake
repair_intake.intake.customer
repair_intake.customer_reference
customer_master
\`\`\`

### environment local

\`\`\`text
repair_intake.customer_master -> master_data.dev
repair_intake.customer_master -> master_data.prod
\`\`\`

この分離だけで初回Portability Referenceを構成できる可能性が高い。

## 6. Source Indexとの関係

destination sourceがGate C後にAcceptanceされた場合、新しいApplication / Patternは通常のSource Index subjectとして扱える。

既存の、

\`\`\`text
application:<id>
application-pattern:<id>
\`\`\`

を再利用する。

shared Semantic / Contractは既存subject identityを参照する。

Portability専用Source Index subjectを最初から追加しない。

## 7. AI Builderとの関係

AI Builderに必要なのは新しいsyntaxより、

- reusable shared subjects
- destination-owned decisions
- unresolved
- origin provenance
- qualification reason
- acceptance boundary

である。

これらはBuilder Contextとしてderivedに渡せる。

AIが最終sourceを生成する場合も、出力は通常のTEI Application / Pattern / Environment sourceになる。

\`\`\`text
AI planning context
  != Canonical portability syntax
\`\`\`

である。

## 8. Human reviewとの関係

Human Reviewでは、

\`\`\`text
shared
  customer.customer
  master_data.customer ^2

new local
  destination Application
  Relation
  Pattern
  alias
  Environment binding
\`\`\`

を明示すればよい。

reviewerにinheritance graphを理解させる必要はまだない。

## 9. Test-only source

\`tools/reference-business-procedure-check/second_application_schema_sufficiency_pressure_test.go\`

次を確認する。

1. existing Application grammarでcandidateをparseできる
2. shared Semantic / Contractを保持できる
3. destination aliasがorigin aliasと異なる
4. existing reference Pattern grammarでcandidateをparseできる
5. destination Relationをorigin Relationと分離できる
6. shared Contract Query / input bindingを保持できる
7. existing Environment grammarでdestination bindingをparseできる
8. binding Application / aliasをoriginと分離できる
9. premature portability directiveをApplication grammarが受け付けない

## 10. Evidence高さ

このtest-only sourceを追加しただけではactual parser execution PASSを主張しない。

\`\`\`text
Design Evidence
  current grammar is expected to be sufficient

Source Evidence
  parser pressure source exists

Executable Evidence
  actual go test PASS

Reference Evidence
  Gate C後のaccepted second Application reloads successfully

Product Evidence
  repeated reuse works without special portability grammar
\`\`\`

各段階を分ける。

## 11. Executable Evidence runner

schema sufficiency pressureはfocused GitHub Actions runnerで実行する。

- `plan/KNOWLEDGE_PORTABILITY_EXECUTABLE_EVIDENCE_2026-09.md`
- `.github/workflows/knowledge-portability-reference-evidence.yml`

successful runで確認できるのはcurrent parser / model / Referenceに対するtest実行までであり、actual second Applicationが既存grammarだけで十分だったというReference Evidenceとは区別する。

## 11. Gate C後の判断

actual second Applicationを実装した結果、

- existing grammarだけで十分
- Source Index / Validationも自然に接続
- change propagationもshared identityで説明可能

なら、専用Portability schema追加を見送る。

逆に、

- provenanceが失われてreview不能
- linked/local distinctionをsourceから復元不能
- package boundaryが必要
- multi-repository distributionでidentityが崩れる

等のEvidenceが出た場合、そこで新schema候補を設計する。

## 12. 停止線

このPressureを理由に、

- second Canonical Application
- global Pattern catalog
- Knowledge Package
- inheritance graph
- override precedence
- auto-sync
- cross-repository registry

を追加しない。
