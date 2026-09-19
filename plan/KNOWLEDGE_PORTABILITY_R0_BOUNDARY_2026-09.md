# Knowledge Portability R0 — Authority / Provenance / Variation Boundary

- Status: ISSUE-000053 Phase 5 / R0 design baseline / Reference-only
- Date: 2026-09-18
- Related: RQ-014 / RQ-015 / RQ-017 / RQ-021 / RQ-029 / ISSUE-000040 / ISSUE-000047 / ADR-0017 / ADR-0018 / ADR-0022
- Product Gate impact: None
- Runtime impact: None
- Production schema impact: None

## 1. 目的

TEIが保持するDomain / Application / Database / Infrastructure knowledgeを別Applicationへ利用するとき、単純copy-pasteではなく、

- どこでそのknowledgeを見つけたか
- 誰がそのmeaningをauthoritativeに所有しているか
- destinationは同じauthorityを直接参照するのか
- destination自身のmeaningとして受け入れ直すのか
- どのassumption / dependency / Variationが適用条件になるか
- originまたはshared authorityが変化したとき何をstaleとみなすか

を区別する。

R0ではKnowledge Package、inheritance graph、override DSL、cross-repository sync protocolを作らない。

current `order_entry` Referenceだけから、portable / conditionally portable / destination-owned / nonportableを区別できるかをPressureする。

## 2. ISSUE-000040 / 000047との責務分担

### ISSUE-000047 — Knowledge-first exploration

担当:

```text
candidate discovery
  -> qualification
     applicable / incompatible / unknown
```

「何が候補か」「現在のintentへ使えるか」を扱う。

### ISSUE-000040 — Semantic component composition

担当:

```text
qualified existing knowledge
  + destination choices
  -> explicit-reference Application candidate
  -> Shared Acceptance
```

同一Repository内で意味資産を別Applicationへ組み立てる。

### ISSUE-000053 Phase 5 — Knowledge portability

担当:

```text
reuse relation itself
  origin observation
  authority owner
  dependency form
  local variation
  provenance
  freshness / drift
```

つまりPhase 5は新しい検索engineやApplication composerを重複実装しない。

000047 / 000040が扱うcandidate / compositionに対し、「再利用後もauthorityとorigin関係を失わないか」をPressureする。

## 3. 最重要の非同一性

### 3.1 discovery provenance != authority owner

例:

```text
order_entry.customer_reference
  -> customer.customer を利用している
```

ここで`customer.customer`をorder_entryから見つけても、そのmeaningのownerはorder_entryではない。

```text
discovered through
  order_entry

authoritative subject
  semantic-data:customer.customer
```

destinationが`customer.customer`を再利用するとき、dependencyは原則としてorder_entryではなく、shared Semantic authorityへ向ける。

### 3.2 provenance != dependency

「この設計はorder_entryを参考にした」というprovenanceを残してもよい。

しかし、

```text
inspired by order_entry
  != depends on order_entry
```

である。

### 3.3 dependency != inheritance

shared Canonical identityを参照するlinked reuseは、origin Applicationの全assumptionを継承することではない。

```text
uses customer.customer
  != inherits order_entry behavior
```

### 3.4 copied shape != shared meaning

Application Patternの構造が似ていても、PatternはApplication-ownedである。

```text
order_entry.customer_reference
  copied to another Application
  != same Canonical Pattern
```

destinationが同様のreference Patternを採用するなら、destination-owned PatternとしてAcceptanceする。

## 4. Reuse relationの四分類

R0ではproduction enumを採用しないが、意味差をPressureするため次の四分類を使う。

### A. linked reuse

destinationが同じauthoritative Canonical subjectへexplicit dependencyを持つ。

例:

- `customer.customer`
- `customer.customer_id`
- 条件を満たした場合の`master_data.customer` Contract

性質:

```text
destination
  -> shared authority subject
```

shared authorityが意味上変更された場合、destinationはrequalification対象になり得る。

### B. conditional linked reuse

同じCanonical subjectを利用できるが、destination側の明示条件が必要。

例:

`master_data.customer`を利用するには、destinationが少なくとも、

- target Database
- Contract major compatibility
- datasource access
- required Query / result semantics

を満たす必要がある。

origin Applicationのdatasource aliasやEnvironment Bindingは条件を代行しない。

### C. adopted local variation

origin knowledgeを参考にするが、destinationが自分のauthorityとして新しいmeaningをAcceptanceする。

例:

`order_entry.customer_reference`の構造を参考に、別Applicationが独自の`<destination>.customer_reference`をauthorする。

性質:

```text
origin pattern
    ↓ provenance
destination-owned pattern
```

origin Pattern更新はdestination authorityを自動変更しない。

### D. nonportable / rebind-required

identityやmeaningがApplication / Environment / Runtime contextへ強く結びついており、そのまま移植してはいけない。

例:

```text
bind order_entry.master -> master_data.dev
bind order_entry.master -> master_data.prod
```

destinationではApplication ID、datasource alias、Environment、instance、Resource等を明示的に再解決する。

## 5. current order_entryからのR0 source map

### 5.1 Semantic Data — linked reuse candidate

Source:

`reference/workspace/semantic/customer/customer.tei`

```text
data Customer {
    attribute customerId type customer.customer_id
    attribute name type std.text
}
```

Canonical:

- `customer.customer`
- `customer.customer.customer_id`
- `customer.customer.name`
- `customer.customer_id`

destinationのBusiness meaningが同じCustomer概念を必要とするなら、同じSemantic identityをexplicitに参照する候補になる。

ここで新しい`repair.customer`等をcopy生成することを標準にはしない。

ただし「Customerという名前が似ている」だけではlinked reuseを確定しない。ISSUE-000047のqualificationを通す。

### 5.2 Database Contract — conditional linked reuse candidate

Source:

`reference/workspace/databases/master-data/contracts/customer.tei`

```text
contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
        input id type customer.customer_id
        result customer.customer zero-or-one
    }
}
```

destinationが同じ`master_data`へ接続し、compatible major 2をrequireし、read useを採用するならlinked reuse候補になる。

ただしorder_entryの、

```text
datasource alias = master
access = read
```

はdestinationへ継承しない。

destinationが自分のApplication sourceでdatasource requirementをauthorする。

### 5.3 Application Pattern — adopted local variation candidate

Source:

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

ADR-0017 / ADR-0018によりPatternはApplication-ownedである。

したがって別ApplicationがこのPatternをそのままCanonical dependencyとして共有することをR0標準にはしない。

再利用できるのは、たとえば次のknowledgeである。

- reference PatternというApplication intent
- Contract Queryを使う構成
- Semantic attributeからQuery inputへのexplicit mapping
- read-only lookupという設計経験

destinationでは、

```text
origin:
  order_entry.customer_reference

provenance:
  observed reference design

destination:
  <destination>.customer_reference

authority:
  destination Application

accepted refs:
  destination-specific Relation
  destination-specific datasource alias
  qualified shared Contract
```

となる。

特に`sales.order.customer`はorder_entry / sales contextに結び付くため、別Business contextへ名前だけで移植しない。

### 5.4 Environment Binding — rebind-required

Source:

`reference/workspace/environments/development/environment.tei`

```text
bind order_entry.master -> master_data.dev
```

production:

```text
bind order_entry.master -> master_data.prod
```

これは、

- Application = order_entry
- local datasource alias = master
- Environment
- Database instance

を含む。

destinationへcopyしてはいけない。

一方、

```text
master_data.dev / master_data.prod
  definition master_data
  provider oracle
```

という事実はdestination infrastructure設計のcandidate Evidenceにはなり得る。

それでもdestination bindingはdestination側のEnvironment authorityで受け入れる。

## 6. 第一destination probe

Gate C前に新しいCanonical Applicationを追加しない。

R0ではtest-only destination profileを使う。

候補scenario:

```text
destination = repair_intake probe
purpose = customerをIDで参照する
customer meaning = existing customer.customerと同一候補
customer maintenance ownership = not requested
required access = read
database target = master_data candidate
contract major = 2 candidate
```

`repair_intake`という名前はtest-only scenario labelであり、Product roadmap上の第二Application採用を意味しない。

### positive case

destination profileが明示的に、

- shared `customer.customer`を利用
- `master_data.customer ^2`をrequire
- read accessを選択

した場合:

- Semantic Data → linked reuse candidate
- Contract → conditional linked reuse qualified
- origin Pattern → destination-owned variation candidate
- origin Environment Binding → nonportable / rebind-required

### unknown case

destinationがCustomer意味は必要とするが、Database target / Contractをまだ選んでいない場合:

- Semantic Data → linked reuse candidate
- Contract → unknown
- Pattern realization → unresolved
- Environment Binding → unresolved / destination-owned

unknownを「order_entryと同じでよい」へ変換しない。

### incompatible case

destinationがorigin Patternをそのまま利用しようとするが、`sales.order.customer` Relationを持たないBusiness contextの場合:

- origin Pattern direct reuse → incompatible
- shared customer Semantic Data →独立にapplicableであり得る

```text
Pattern incompatible
  != every referenced knowledge incompatible
```

を守る。

## 7. linked reuseのfreshness

linked reuseではdestinationがshared authorityを参照する。

そのためfreshnessは、origin Applicationのrevision全体ではなく、実際に依存したsubject / source inputへ結び付ける。

例:

```text
destination
  -> semantic-data:customer.customer
  -> database-contract:master_data.customer
```

これらのauthoritative inputが変わった場合:

```text
dependency changed
  -> requalification required
```

とできる。

一方、order_entryの無関係なUI / Pattern変更だけでdestination reuseをstale-allにしない。

## 8. adopted local variationのdrift

local variationではdestination authorityが独立する。

```text
origin pattern changes
  != destination pattern automatically stale
  != destination authority automatically changes
```

ただしprovenanceを保持しているため、

```text
origin has changed since adoption
```

というdrift signalを出すことはできる。

このsignalは、

- informational
- review candidate
- requalification candidate

にはなり得るが、destination sourceへのauto patchではない。

### provenance drift != dependency staleness

これを明確に分ける。

```text
linked dependency changed
  -> dependency requalification

provenance-only origin changed
  -> provenance drift

provenance drift
  != destination invalid
```

## 9. origin override / destination override

R0ではgeneric override mechanismを作らない。

destinationがorigin candidateと異なる選択をした場合は、まずdestination-owned meaningとして表現する。

例:

```text
origin Pattern
  datasource alias = master

destination Pattern
  datasource alias = customer_master
```

これはdestination variationであり、origin Patternを上書きしたわけではない。

```text
destination difference
  != override of origin authority
```

generic inheritance / override graphが必要かは、複数の実Referenceで差分関係が繰り返し現れてから判断する。

## 10. Infrastructure knowledge portability

Infrastructureでは粒度を分ける。

### 比較的portableなknowledge candidate

- Database definition identity
- Provider compatibility knowledge
- Contract / Provider requirement
- Target-neutral Resource requirement
- validated migration / query realization pattern
- provider-specific caveat with provenance

### destination-owned / Environment-owned

- instance identity
- Application datasource alias
- Environment binding
- secret Resource ID
- network endpoint
- filesystem path
- Permission grant
- Lease
- deployment route
- environment-specific provider availability

```text
provider knowledge reusable
  != environment binding reusable
```

を維持する。

## 11. Portability recordのconceptual decomposition

production schemaではない。

```text
PortabilityCandidate

  discovery
    observed_in
    observed_from_subject
    evidence

  authority
    subject_kind
    subject_id
    source
    fingerprint / version

  candidate_relation
    linked-reuse
    conditional-linked-reuse
    adopted-local-variation
    rebind-required

  assumptions[]
  dependencies[]
  destination_requirements[]

  qualification
    applicable / incompatible / unknown
    reasons[]

  provenance
    origin observation
    adopted-from if local variation

  freshness
    linked dependency inputs
    provenance-only drift inputs

  authority_limits[]
```

field名・enumはproduction contractとして固定しない。

## 12. R0 test-only pressure

test-only sourceではcurrent Repositoryとsynthetic destination profileから、少なくとも次を再現する。

### Case A — shared Semantic linked reuse

`customer.customer`について:

- discovery provenanceはorder_entry / master_data経由でもよい
- authority subjectは`semantic-data:customer.customer`
- linked dependencyはSemantic sourceへ向く
- order_entryをauthority ownerにしない

### Case B — Contract conditional linked reuse

`master_data.customer`について:

- destinationが`master_data.customer ^2`を明示的にrequireする場合のみqualified
- origin datasource alias`master`をcopyしない
- destination aliasはdestination-owned

### Case C — Application Pattern local variation

`order_entry.customer_reference`について:

- direct cross-Application linked reuseを拒否
- origin patternはprovenance candidate
- destination Pattern IDはdestination-owned
- `sales.order.customer`をdestinationへ自動移植しない

### Case D — Environment Binding rebind

`bind order_entry.master -> master_data.dev/prod`について:

- direct copy不可
- destination Application / alias / Environmentを再解決
- provider / database definition knowledgeだけをcandidate Evidenceにできる

### Case E — unknown

destinationがDatabase / Contractを未選択なら`master_data.customer` reuseをunknownとして保持する。

## 13. 禁止するshortcut

test-only Portability modelに次を置かない。

- `AutoApply`
- `AutoSync`
- `Inherited`
- `OverrideWins`
- `OriginOwnsDestination`
- `DestinationUpdatesOrigin`
- `CopyAllDependencies`
- `SameNameMeansSameMeaning`
- `Portable=true`の単一boolean
- `PreferredCandidate`
- `Winner`

portabilityは一値ではなく、subjectごとのauthority / dependency / destination conditionで決まる。

## 14. Test-only implementation source

最初のR0 pressure sourceは次へ置く。

- `tools/reference-business-procedure-check/knowledge_portability_r0_pressure_test.go`

current RepositoryからSource Indexを構築し、test-only destination profileに対して次を確認する。

### Semantic

`customer.customer`を`order_entry.customer_reference`経由で発見しても、

```text
discovery provenance
  = application-pattern:order_entry.customer_reference

authority
  = semantic-data:customer.customer
```

として分離する。

destination dependencyへorigin Patternを混ぜない。

### Contract

`master_data.customer`は、

- destination Database target = `master_data`
- Contract = `master_data.customer`
- compatible major = current major
- read可能なaccess
- destination-owned datasource alias

が明示された場合だけ`conditional-linked-reuse / applicable`とする。

Database / Contract選択がなければ`unknown`を維持する。

### Application Pattern

`order_entry.customer_reference`は、

```text
adopted-local-variation
```

として扱い、test-only destinationには別Pattern identityを要求する。

origin Patternはprovenanceであり、destination dependencyにはしない。

### Environment

developmentの、

```text
bind order_entry.master -> master_data.dev
provider = oracle
```

を読み取るが、destination bindingは作らない。

provider / Database情報はcandidate Evidenceとして保持できても、binding自体は`rebind-required / destination-owned`とする。

### Freshness

test-onlyに、

```text
linked dependency subject change
  -> dependency-requalification

provenance-only origin change
  -> provenance-drift

unrelated origin change
  -> unaffected
```

を分ける。

このsourceはproduction Portability schemaを追加しない。

**sourceの存在だけではactual test PASSを意味しない。**

## 15. R0 success conditions

- [ ] discovery provenanceとauthority ownerを分離できる
- [ ] shared Semantic identityをorigin Applicationのcopyへ変換しない
- [ ] Application-owned Patternを別Applicationのshared authorityへ昇格しない
- [ ] Contract reuseにdestination-specific requirementを要求できる
- [ ] origin datasource alias / Environment Bindingをdestinationへ継承しない
- [ ] linked dependency changeとprovenance-only driftを分けられる
- [ ] incompatible Patternでも内包するshared Semantic candidateまで捨てない
- [ ] unknownをimplicit reuseへ変換しない
- [ ] origin updateをdestinationへauto-applyしない
- [ ] destination variationをoriginへback-propagateしない
- [ ] test-only destinationをCanonical Applicationへ昇格しない

## 16. Executable Evidence runner

focused runnerは次を正本とする。

- `plan/KNOWLEDGE_PORTABILITY_EXECUTABLE_EVIDENCE_2026-09.md`
- `.github/workflows/knowledge-portability-reference-evidence.yml`

successful runがcurrent PR headへ結び付いた場合のみ、R0 pressureをSource EvidenceからExecutable Evidenceへ昇格する。

Workflow未実行 / failure / cancelledではPASSを主張しない。

## 16. R0後

R0で境界が成立したら、ISSUE-000040の第二Application Referenceへ接続する。

順序:

```text
R0 authority/provenance pressure
  ↓
ISSUE-000047 candidate qualification
  ↓
ISSUE-000040 second-Application composition
  ↓
Shared Acceptance
  ↓
linked reuse / local variationのchange propagation test
```

ここで初めて、実際の第二Applicationを使って、

- shared Semantic change
- origin Pattern change
- destination local variation change
- Contract major change
- Environment rebinding

を分離して観測する。

R0だけを理由にKnowledge Packageやsync daemonを実装しない。
