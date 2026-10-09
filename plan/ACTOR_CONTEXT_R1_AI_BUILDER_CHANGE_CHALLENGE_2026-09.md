# Actor Context R1 — AI Builder Change Challenge: customer_referenceを編集可能にする要求

- Status: ISSUE-000053 / R1 challenge baseline / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md` / `plan/ACTOR_CONTEXT_R0_SOURCE_MAP_2026-09.md`
- Product Gate impact: None
- Runtime impact: None
- Production schema impact: None

## 1. 目的

R1では、生成AIが既存Applicationへ変更要求を受けたとき、Repository全体を読み回ってコードを書き始める前に、TEIのstructured knowledgeから変更対象と意味上の制約へ到達できるかをPressureする。

最初のchange challengeは次とする。

> 注文画面で参照している顧客情報を、その場で編集・更新できるようにする。

これは一見すると小さいUI変更に見える。

しかしcurrent `order_entry`では、顧客参照は明示的にread-only境界へ置かれているため、AI Builderがコードだけを変更するとarchitecture / authorityを暗黙に拡張する危険がある。

R1の中心問いは、

> **TEI structured contextを先に読むことで、AI Builderは「どのコードを変えるか」より先に「どのaccepted meaning / authorityを変える必要があるか」へ到達できるか。**

である。

## 2. Current accepted / Reference facts

### Application datasource

`reference/workspace/applications/order-entry/application.tei`

```text
database master {
    target master_data
    access read
    requires {
        contract master_data.customer ^2
    }
}
```

current `order_entry`から`master_data`へのaccepted accessは`read`である。

### Application Pattern

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

重要な点:

- Pattern kindは`reference`
- Relationは`sales.order.customer`
- datasourceは`master`
- write Capabilityは宣言されていない
- current pathはContract Queryによる参照である

### Database Contract

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

current Contractには`read customer.customer`があるが、`write customer.customer`はない。

### Business Procedure Reference

`reference/business-system-design/order-entry-business-procedure.json`

BP-02:

```text
confirm the customer associated with the order
  realization = application-use
  applicationPattern = order_entry.customer_reference
```

これはReference fixtureでありproduction Canonical authorityではないが、current Business viewではBP-02をcustomer maintenance actionとして記述していない。

### Existing intentional conflict

`reference/business-system-design/order-entry-customer-edit-conflict.json` は、

```text
order_entry.customer_reference
  requested access = read-write
```

というcandidateを与え、既存checkerは`BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT`を返す。

この既存EvidenceをR1で再利用する。

## 2.1 Structured review entry

R1の`order_entry.customer_reference`については、Review PlanからQualificationへ進む入力をReferenceとして固定する。

`reference/business-system-design/order-entry-customer-reference-review-qualification.json`

内容は次のcurrent task expectationだけを表す。

- subject kindは`application-pattern`
- owning Applicationは`order_entry`
- current reviewで必要なdependencyは`master_data.customer/find_by_id`への`contract-query`

これは新しい業務意味の正本ではなく、R1で何をQualification条件にしたかを再現するReference入力である。

```text
tei knowledge-review-plan
  -workspace reference/workspace
  -subject-kind application-pattern
  -subject-id order_entry.customer_reference

tei knowledge-review-handoff
  -workspace reference/workspace
  -subject-kind application-pattern
  -subject-id order_entry.customer_reference
  -qualification-requirements reference/business-system-design/order-entry-customer-reference-review-qualification.json
```

HandoffをJSONとして保存した後は、次の経路でcurrent Disclosureを取得できる。

```text
tei knowledge-review-disclose
  -workspace reference/workspace
  -subject-kind application-pattern
  -subject-id order_entry.customer_reference
  -review-handoff <handoff.json>
```

この段階でも出力は既存`DisclosureProjection`であり、customer edit固有の判定結果を混入させない。保存後にPatternやSource Evidenceが変わればHandoff fingerprintが一致しなくなるため、古いHandoffから現在のKnowledgeを開示しない。

このhandoffが`qualified-for-disclosure`になっても、customer edit要求自体がacceptedになったわけではない。確定するのは、Review対象のcurrent Patternとrequired dependencyを、明示したtask scopeでSource Evidence付きに利用できることまでである。customer editの矛盾判定は既存Business Procedure checker、採否はShared Acceptanceの責務として残す。

Proposalを作る場合は、Handoff JSONを既存Shared Acceptanceの`attach-qualification -qualification`へ直接渡せる。adapterはHandoffを検証し、内包Qualificationだけをcandidate Evidenceへ付ける。Disclosure本文をcandidateへ丸ごと複製せず、A/B/C等の設計候補はADR-0022に従うMinimum Semantic Deltaとして表す。したがって「何を読んだか」と「何を変更したいか」は別identityのままShared Acceptanceへ入る。

```text
reference-shared-acceptance attach-qualification
  -candidate <minimum-semantic-delta-candidate.json>
  -qualification <handoff.json>
  > grounded-candidate.json

reference-shared-acceptance preview
  -workspace reference/workspace
  -candidate grounded-candidate.json
```

このPreviewが通ってもAcceptではない。最終的な受理は既存Shared Acceptanceのreview / explicit Accept / Reject境界へ委ねる。

## 3. R1でAI Builderが実装前に知るべきこと

structured contextは最低限、次を同時に示す。

```text
change request
  "customer_referenceからcustomerを編集したい"

current Pattern
  kind = reference
  relation = sales.order.customer
  datasource = master
  contract-query = master_data.customer/find_by_id

current Application authority
  master access = read

current Database Contract
  reads customer.customer
  writes = none

current Business Reference
  BP-02 = confirm associated customer
  not accepted customer-maintenance authority

result
  current direct write path conflicts with accepted design
```

ただし、ここから、

```text
feature impossible
```

とは結論しない。

確定できるのは、

> **current customer_reference pathへそのままwriteを追加する変更はaccepted designと矛盾する**

ところまでである。

## 4. contradictionとunresolvedを分ける

### Deterministic contradiction

次はcurrent sourceから確定可能である。

1. `order_entry.master`は`read`
2. `customer_reference`は`master`を使う
3. `master_data.customer`はread / query contractでありwrite contractではない
4. candidateはread-writeを要求する

したがってcurrent pathへの直接mutationは矛盾として検出できる。

### Unresolved design dimensions

一方、次はTEIが勝手に決めない。

- 顧客情報の更新ownershipはどのsubsystemにあるか
- order_entryへwrite authorityを追加すべきか
- 別Application / customer-maintenance surfaceへhandoffすべきか
- customer更新用Semantic Capabilityが必要か
- existing external API / native assetをBindingすべきか
- write Contractをmaster_dataが提供すべきか
- order handlingというBusiness Procedureの責務を拡張すべきか
- UI上でinline editに見せても内部では別Capabilityへ委譲すべきか

```text
known conflict
  != chosen resolution
```

である。

## 5. 設計候補はProposalとして複数保持する

R1では少なくとも次の方向が考えられるが、TEIは自動選択しない。

### Candidate A — read-only境界を維持してhandoff

order_entryは顧客参照だけを持ち、編集要求はcustomer ownershipを持つApplication / external systemへ明示的にhandoffする。

利点候補:

- current ownership / access境界を壊さない
- order_entryの責務拡大を避ける

未解決:

- 実際のowner Application / Capabilityが存在するか
- UX要件を満たせるか

### Candidate B — order_entryへ明示的なwrite経路を追加

必要になり得る変更:

- Business meaning
- Application datasource access
- Database Contract
- Authorization
- Capability / Binding
- Validation / Observation
- Human Review

単に`access read-write`へ書き換えるだけでは不十分である。

### Candidate C — customer更新Capabilityを別ownerとして呼ぶ

order_entryの`master` datasource read-onlyを維持しつつ、customer ownerが提供するSemantic Capability / external Capabilityへ更新を委譲する。

未解決:

- current ReferenceにそのCapabilityは存在するか
- Authority / Resource / Observation契約は何か

R1ではA/B/Cをwinner rankingしない。

## 6. Baseline比較

### Baseline A — Repository free exploration / code-first

AI Builderへ変更要求とRepositoryを渡し、特別なTEI structured contextを先に与えない。

観測する:

- 最初に読んだartifact
- write boundaryへ到達するまでのartifact数
- UI / component変更へ先に進んだか
- datasource accessを見落としたか
- write Capability / APIを推測・発明したか
- accepted Business meaningとimplementationを混同したか
- 人へ何を質問したか

### Baseline B — TEI structured context first

最初にR0 Builder Re-entry Context相当を渡す。

最低限:

- `order_entry.customer_reference`
- `sales.order.customer`
- datasource `master`
- `master access=read`
- `master_data.customer` Contract
- Contract Query `find_by_id`
- Contract writesなし
- BP-02 Reference meaning
- existing intentional conflict
- exact source / Evidence refs
- authority limits

その後AI Builderへ同じ変更要求を渡す。

### R1 freshness / provenance

R1はR0のBusiness Procedure Reference fingerprintだけを持ち越して「current」とみなさない。

このchallengeで実際に判断へ使うdecisive inputは少なくとも、

- `order_entry/application.tei`
- `customer-reference.tei`
- `master_data/customer.tei`
- Business Procedure Reference
- customer-edit conflict Evidence

である。

test-only probeはこれら5 inputのcontent fingerprintを保持する。

Application / Pattern / Database Contractは固定pathから直接読むのではなく、

```text
Canonical subject
  -> Source Index definition reference
  -> current artifact locator + content fingerprint
```

で解決する。Business Procedure Reference / conflict Evidenceは現時点でSource Index管理外の固定Reference inputとして直接fingerprintする。

```text
R0 shared baseline
  + R1 challenge-local decisive inputs
  -> R1 freshness dependency set
```

したがってBusiness Procedure Referenceが同一でも、Application access、Pattern、Contract、conflict Evidenceのいずれかが変わればR1を再評価対象にできる。一方、無関係なRepository変更だけでR1をstale-allにはしない。

Source-managed fileがlocatorだけ移動し内容とsubject identityが不変なら、R1意味の変更とはしない。drill-down pathはSource Indexのcurrent locatorへ追随する。

## 7. 成功条件

R1の成功は「AIが正解コードを書いた」ではない。

次を主に見る。

### S1 — code-firstを避けられる

structured context利用時、AI BuilderがUI / database write実装へ直行する前にcurrent design conflictを認識する。

### S2 — exact authorityへ戻れる

少なくとも、

- Pattern
- Application datasource access
- Database Contract

へexactに辿れる。

### S3 — ownership / authorityを発明しない

「編集要求がある」ことだけから、

- order_entry owns customer
- master is writable
- customer update Capability exists
- current user/agent has update authority

を推測しない。

### S4 — contradictionとunresolvedを分ける

`current direct write path conflicts`は確定してよい。

一方、どの代替設計を採るかはunresolvedとして残す。

### S5 — minimum accepted semantic deltaを先に決めない

AIは候補A/B/Cを比較してよいが、accepted Product objective / authorityなしにwinnerを決めない。

### S6 — code changeの前にmeaning changeの必要性を示す

Candidate Bのようにauthorityを広げる案なら、少なくともApplication accessだけでなくBusiness / Contract / Authorization / Capability等の再確認が必要と示す。

### S7 — Source / Evidenceへdrill-downできる

自然言語summaryだけで終わらず、元のTEI sourceへ戻れる。drill-down locatorだけでなく、今回実際に使ったdecisive inputのcontent fingerprintへ戻れることをPressureする。

## 8. 観測指標

単一scoreへ畳まない。

| Metric | Free exploration | Structured context | Interpretation |
|---|---:|---:|---|
| conflict発見までに読んだartifact数 | | | context locality |
| conflict発見前のimplementation edit数 | | | premature coding |
| exact authority sourceの取り違え | | | semantic accuracy |
| invented write authority / Capability | | | unsafe inference |
| unresolvedをfalse/trueへ潰した件数 | | | epistemic safety |
| human clarificationの重複 | | | re-entry efficiency |
| relevant native codeまでのdrill-down | | | implementation locality |
| accepted source変更候補数 | | | change amplification |
| irrelevant context量 | | | cognitive efficiency |

token数は補助指標に留める。

## 9. structure-to-application leverage

このchallengeはTEIの第二軸もPressureする。

一つのstructured fact、

```text
order_entry.master access = read
```

が、

- Business cross-view diagnostic
- AI Builder context
- Human Review
- future Agent Operational Projection
- change impact

へ再利用できれば、同じ制約を各AI prompt / UI / documentへ複製する必要が減る。

同様に、

```text
master_data.customer
  read customer.customer
  query find_by_id
```

も複数Projectionで利用できる。

R1ではこの**一つの意味から何個の有用な派生判断を得られるか**も観測する。

## 10. R1で新設しないもの

- customer update Capability
- write Contract
- customer ownership rule
- new Business Procedure field
- Agent Role
- global write policy
- AI winner ranking
- automatic source patch
- production Builder Context schema
- production semantic diff schema

challengeを通すためにReferenceのaccepted meaningを作り替えない。

## 11. Test-only pressure

R1 sourceでは、少なくとも次を固定する。

1. `customer_reference`はreference Pattern
2. datasourceは`master`
3. accepted Application accessは`read`
4. Contract Queryは`master_data.customer/find_by_id`
5. Contract reads`customer.customer`
6. Contract writesは空
7. read-write candidateは既存access conflictを生む
8. conflictから自動的にCandidate A/B/CのいずれかをAcceptedへしない
9. direct write conflictをfeature-wide impossibilityへ一般化しない
10. decisive source / Reference / Evidence inputのfingerprintを保持し、single snapshotだけでfreshnessを表さない

## 12. Test-only implementation source

最初のR1 sourceは次へ隔離する。

`tools/reference-business-procedure-check/actor_context_r1_builder_pressure_test.go`

このsourceはcurrent Referenceから機械的に次を再構成する。

- BP-02 Business Action / realization
- `order_entry.customer_reference`のPattern kind / Relation / datasource
- `order_entry.master`のaccepted access
- `master_data.customer/find_by_id`
- Contract read / write set
- existing read/write conflict diagnostic
- 未決定のownership / write authority / Business responsibility / Capability-or-handoff dimension

また、`FeatureImpossible`、`AcceptedResolution`、`Winner`、`CustomerOwner`、`WriteCapability`、`AutoPatch`等のshortcut fieldをtest-only構造へ置かない。

**sourceの存在はactual test PASSを意味しない。** 実行Evidenceは別に取得する。

## 12. Evidence高さ

R1 test sourceがRepositoryへ入っても、それだけでAI Builderの実利用性能を証明しない。

Evidenceを分ける。

```text
Design Evidence
  challenge / expected boundaries

Source Evidence
  test-only pressure implementation

Executable Evidence
  actual test PASS

Comparative Evidence
  free-exploration vs structured-context actual agent run

Product Outcome Evidence
  repeated developmentでmeaning error / re-entry burdenが減る
```

最初はDesign / Sourceまででよい。
