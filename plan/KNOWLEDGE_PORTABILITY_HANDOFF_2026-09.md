# Knowledge Portability Handoff — Exploration / Portability / Composition

- Status: cross-Issue handoff baseline / pre-Gate-C
- Date: 2026-09-18
- Related: ISSUE-000047 / ISSUE-000053 Phase 5 / ISSUE-000040 / RQ-014 / RQ-015 / RQ-017 / RQ-029
- Production schema impact: None
- Gate impact: None

## 1. 目的

Knowledge Portability R0で整理したauthority / provenance / variation境界を、既存のKnowledge-first explorationとSemantic component compositionへ接続する。

三つのIssueは次の順で責務を分ける。

\`\`\`text
ISSUE-000047
  candidate discovery / qualification
        ↓
ISSUE-000053 Phase 5
  reuse relation / authority / provenance / freshness
        ↓
ISSUE-000040
  destination composition / Proposal / Acceptance
\`\`\`

同じ機能を三箇所で実装しない。

## 2. ISSUE-000047から受け取るもの

000047が返すcandidateは、「検索結果」ではなくdeterministic qualification済みの候補である。

Phase 5が必要とする最小情報:

- candidate subject kind / identity
- authoritative source reference
- source fingerprint / version等のfreshness input
- qualification:
  - applicable
  - incompatible
  - unknown
- qualification reason
- required dependency / constraint
- destinationで不足している条件
- discovery provenance
- candidate mechanismは補助Evidence

重要:

\`\`\`text
candidate discovered in Application A
  != Application A owns candidate meaning
\`\`\`

000047は、discovery provenanceとauthority sourceを可能な限り分離した形でhandoffする。

## 3. Phase 5で追加する解釈

Phase 5はcandidateを別Applicationへどう関係付けるかを扱う。

R0では概念的に次を区別する。

\`\`\`text
linked reuse
conditional linked reuse
adopted local variation
rebind-required
\`\`\`

これは000047のqualificationを置き換えない。

例:

\`\`\`text
candidate = master_data.customer
qualification = applicable
reuse relation = conditional linked reuse
\`\`\`

のように両方を保持する。

### qualification != reuse relation

\`\`\`text
applicable
  != linked reuse

incompatible direct reuse
  != useless knowledge

unknown
  != implicit inheritance
\`\`\`

Application Patternはdirect linked reuseには不適切でも、destination-owned variationのprovenance candidateとして利用価値を持ち得る。

## 4. ISSUE-000040へ渡すもの

000040がdestination compositionを作る際に必要なもの:

### Shared authority references

destinationが同じCanonical subjectへ依存するもの。

例:

- Semantic Data / Type
- compatible Database Contract
- Capability contract

### Destination-owned choices

destinationが自分のauthorityとしてauthorするもの。

例:

- Application Pattern
- datasource alias
- Application database requirement
- Binding
- Environment binding
- destination-specific Validation / Variation

### Provenance-only relationships

origin Application / Pattern / Referenceを設計上の参考にした関係。

これはdestination dependencyへ自動変換しない。

## 5. second-Application compositionでの原則

000040の第二Application Referenceでは次を守る。

### 5.1 shared Semanticはcopyしない

同じmeaningならsame Canonical identityをexplicit referenceする。

\`\`\`text
customer.customer
  -> shared reference
\`\`\`

同名のdestination copyを標準にしない。

### 5.2 Patternはdestinationが所有する

origin Patternが有用でも、

\`\`\`text
order_entry.customer_reference
  != destination Pattern authority
\`\`\`

である。

destinationは自分のPattern identityを持つ。

### 5.3 datasource aliasを引き継がない

originの\`master\`等のlocal aliasはdestinationへ継承しない。

destination Application sourceで明示する。

### 5.4 Environment bindingを引き継がない

\`\`\`text
bind order_entry.master -> master_data.dev
\`\`\`

はdestination bindingではない。

同じDatabase instanceを利用する場合でも、destination用bindingをEnvironment authorityで明示する。

### 5.5 Contract reuseはdestination requirementで成立させる

origin ApplicationがContractを使っていることは、destination利用権の根拠ではない。

destinationがContract / major / accessを明示する。

## 6. 第二Application Referenceのchange propagation matrix

Gate C後にactual second Applicationを導入したら、少なくとも次を分離して検証する。

| Change | Expected destination effect |
|---|---|
| shared Semantic subject meaning change | linked consumerはrequalification対象 |
| shared Contract compatible minor change | compatibility policyに従い再評価 |
| shared Contract major change | compatible-major dependencyを再評価し、必要ならincompatible |
| origin Application Pattern change | local variationにはprovenance drift。destination sourceは自動変更しない |
| destination Pattern change | originへback-propagateしない |
| origin datasource alias rename | destination aliasへ伝播しない |
| origin Environment binding change | destination Environment bindingへ伝播しない |
| shared Database definition/provider constraint change | actual dependencyがあるdestinationだけ再評価 |
| unrelated origin Application change | destinationをstale-allにしない |

## 7. linked dependency freshness

linked reuseでは、destinationが依存したauthoritative inputを追跡する。

\`\`\`text
destination dependency set
  semantic-data:customer.customer
  database-contract:master_data.customer
\`\`\`

このsetの実依存が変化した場合だけ、原則requalification対象とする。

repository-wide revisionだけを理由にstale-allにしない。

## 8. provenance drift

adopted local variationではorigin sourceをprovenanceとして保持できる。

originが変わった場合:

\`\`\`text
origin provenance fingerprint changed
  -> provenance drift
\`\`\`

可能な反応:

- informational notice
- compare candidate
- optional review
- new Proposal

禁止:

- destination auto patch
- destination invalidationの自動断定
- originへのreverse merge

## 9. destination overrideという語を急いで導入しない

R0では、destinationとoriginの差はまず独立authorityとして扱う。

\`\`\`text
origin Pattern uses alias master
destination Pattern uses alias customer_master
\`\`\`

これは「override」と呼ばなくても表現できる。

generic inheritance / override mechanismは、複数の実Referenceで必要性が確認されるまで導入しない。

## 10. Infrastructure knowledgeのhandoff

Phase 5から000040へInfrastructure candidateを渡す場合、次を分ける。

### portable knowledge candidate

- Database definition identity
- Provider compatibility
- Contract requirement
- Target-neutral Resource requirement
- provider-specific caveat / Evidence

### destination-owned configuration

- Database instance
- Environment binding
- secret Resource ID
- network endpoint
- filesystem path
- Permission grant
- Lease policy
- deployment route

\`\`\`text
infrastructure knowledge reusable
  != infrastructure configuration copied
\`\`\`

## 11. Acceptance boundary

Phase 5のreuse relationや000047のqualificationはAcceptanceではない。

\`\`\`text
qualified candidate
  ↓
portability relation
  ↓
destination composition candidate
  ↓
Shared Acceptance
  ↓
destination authoritative source
\`\`\`

AIが組み立てた候補も同じ境界を通す。

## 12. Gate C前に行えること

- current Referenceからcandidate authority / provenanceをPressure
- synthetic destination profileでapplicable / incompatible / unknownをPressure
- linked dependencyとprovenance driftの意味差をPressure
- second-Application change matrixを設計
- required Evidenceを整理

## 13. Gate C前に行わないこと

- second Canonical Applicationの追加
- production package manager
- reusable Pattern global catalogの正本化
- inheritance / override graph
- auto-sync daemon
- destination source auto patch
- cross-repository distribution protocol

## 14. Pre-Gate second Application contract

Gate C後の実Referenceへ進む前の着手条件は、

- `plan/SECOND_APPLICATION_REFERENCE_PRE_GATE_CONTRACT_2026-09.md`

へ固定する。

この契約はsynthetic destinationだけを使い、

- shared authority
- destination-owned Application meaning
- Environment-owned configuration
- unresolved destination choices
- expected change propagation

を分離する。

Canonical Application source、Environment binding、Target implementationは作らない。

## 15. Gate C start packet

Gate C Ready後の実Reference開始順序とbranch収束規則は、

- `plan/GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md`

を正本とする。

Knowledge Portabilityは同じprimary second-Application workstream内の一段階として扱い、独立した長期implementation branchへ分離しない。

### actual S2 status — 2026-09-19

Gate C Ready後の最初の実Referenceについて、S1 qualificationからS2 portability relationまでをSource-definedにした。

- `plan/SECOND_APPLICATION_GATE_C_ENTRY_2026-09.md`
- `plan/SECOND_APPLICATION_S1_S2_QUALIFICATION_PORTABILITY_2026-09.md`
- `tools/reference-business-procedure-check/second_application_s1_s2_actual_pressure_test.go`

current relation:

```text
customer.customer
  -> linked reuse

customer.customer_id
  -> linked reuse

master_data.customer ^2
  -> conditional linked reuse

master_data.customer/find_by_id
  -> conditional linked reuse

order_entry.customer_reference
  -> adopted local variation provenance

order_entry.master Environment binding
  -> rebind-required
```

destination Application identity / Relation / Pattern ID / alias / Authorization / Environment bindingは未決定であり、S2から自動生成しない。

## 15. Gate C後の最初の実Reference

ISSUE-000040で予定されている第二Applicationを使う。

初回は一つのread-only use caseへ限定する。

候補:

\`\`\`text
customer lookup
  shared:
    customer.customer
    customer.customer_id
    master_data.customer ^2

  destination-owned:
    Application identity
    datasource alias
    reference Pattern
    Business context
    Environment binding
\`\`\`

この構成で、shared / local / environment-specificの三層を同時に検証できる。

## 16. Handoff完了条件

- [ ] 000047 candidateがauthority sourceとdiscovery provenanceを混同しない
- [ ] Phase 5がqualificationとreuse relationを別軸で保持する
- [ ] 000040がshared authorityとdestination-owned choiceを分ける
- [ ] Application Patternをcross-Application shared authorityへ誤昇格しない
- [ ] datasource alias / Environment bindingをoriginから暗黙継承しない
- [ ] linked dependency changeとprovenance driftを別扱いする
- [ ] second-Application Referenceでchange propagation matrixを再現する
- [ ] Acceptance前にdestination sourceへ自動適用しない
