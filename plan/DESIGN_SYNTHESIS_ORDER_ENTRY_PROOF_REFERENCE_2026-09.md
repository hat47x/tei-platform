# Design Synthesis Order Entry Proof Reference — 2026-09

状態: **Lane D / Reference proof evaluation / 新規Product contract未採用**

## 1. 目的

[`design/DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`](design/DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md)で定義したP1〜P4を、既存の`order_entry` Referenceへ実際に当てる。

このReferenceの目的は、Business / Application / Data / Securityを一つの巨大なmodelへ統合することではない。

確認するのは次である。

1. accepted sourceだけから`Business Action -> Pattern / Capability -> Data / Relation`をどこまでexactに辿れるか
2. 根拠が足りない問いを`false`へ丸めず`unresolved`として残せるか
3. 明示的authorityと両立しない候補だけを`contradiction`として除外できるか
4. Security policyやDatabase write authorityをBusiness meaningへ逆流させないか
5. Design Synthesisの候補縮約に使えるbounded proof packetを作れるか

新しいBusiness meaning、Canonical kind、persisted proof schema、generic Claim model、CRUD field、protected-target fieldは追加しない。

## 2. 評価snapshot

評価基線:

```text
main @ ff9dba8b689b7a46d27ad6e7e2fcd449049c8531
```

このcommit hashはProduct上のglobal semantic revisionではない。本Referenceで「どのsource集合を読んだか」を再現するための評価時点である。

主なauthorityは次である。

### Business

- `reference/business-system-design/order-entry-business-procedure.json`

### Application

- `reference/workspace/applications/order-entry/application.tei`
- `reference/workspace/applications/order-entry/patterns/recent-orders.tei`
- `reference/workspace/applications/order-entry/patterns/order-record.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `reference/workspace/applications/order-entry/patterns/order-lines.tei`
- `reference/workspace/applications/order-entry/patterns/submit-order.tei`

### Semantic

- `reference/workspace/semantic/sales/order.tei`
- `reference/workspace/semantic/security/authorization.tei`

### Database / Security

- `reference/workspace/databases/sales-core/contracts/order.tei`
- `reference/legacy-auth/authorization.json`

補助的に、既存のPressure / Auditを判断境界として利用する。

- `BUSINESS_ACTION_CRUD_2026-09.md`（Part 2、旧`CRUD_OPERATION_GAP_AUDIT`）
- `BUSINESS_PROCEDURE_D4_O1_2026-09.md`（Part 3、旧`O1_AUTHORITY_DIRECTION_PRESSURE`）
- `issue/000042-business-procedure-cross-view-reference.md`

## 3. 判定ルール

本Referenceではproof resultを次の三つに限定する。

```text
supported
  問いに必要なaccepted authorityをexactに辿れ、主張を支持できる

contradiction
  問いの主張が明示的なaccepted authorityと両立しない

unresolved
  必要なidentity / relation / operation attribution / Business-side meaningが不足し、
  肯定も否定もできない
```

ただし、proof queryの作り方自体がauthority ruleへ違反する場合は、domain上の`contradiction`と混同しない。

例:

```text
policy resourceからBusiness protected targetをbackfillする
```

これは「targetが偽」と証明したのではなく、**導出方向が不正**である。

## 4. P1 — Business Action / Realization

### P1-00 — BP-00は`recent_orders`でApplication支援されるか

問い:

```text
BP-00 determine which order requires attention
  -> order_entry.recent_orders ?
```

accepted path:

```text
Business source BP-00
  -> applicationPatterns[order_entry.recent_orders]
  -> Pattern order_entry.recent_orders browse
```

結果: **supported**

理由:

Business source自身がexact Pattern IDを保持し、そのPatternがcurrent workspaceに存在する。

さらにPatternは:

```text
data sales.order
query order_entry.recent_orders
```

を明示する。

### P1-01 — BP-01は`order_record`でApplication支援されるか

accepted path:

```text
BP-01
  -> applicationPatterns[order_entry.order_record]
  -> Pattern order_entry.order_record record
```

結果: **supported**

ここでsupportedなのは「BP-01がこのPatternをApplication支援として参照する」ことだけである。

`record` kindからread / update operationまでは導出しない。

### P1-02 — BP-02は`customer_reference`でApplication支援されるか

accepted path:

```text
BP-02
  -> applicationPatterns[order_entry.customer_reference]
  -> Pattern order_entry.customer_reference reference
```

結果: **supported**

### P1-03 — BP-03は`order_lines`でApplication支援されるか

accepted path:

```text
BP-03
  -> applicationPatterns[order_entry.order_lines]
  -> Pattern order_entry.order_lines header-detail
```

結果: **supported**

### P1-04a — BP-04はcurrent Business sourceでmanual / unresolvedか

Business source:

```text
BP-04
  realization = manual-unresolved
```

結果: **supported**

これは「current Business sourceがmanual-unresolvedと明示している」という問いへの結果である。

### P1-04b — BP-04にはRepository内のどこにもApplication realizationが存在しないか

結果: **unresolved**

理由:

current Business sourceがdirect Application refを持たないことは確認できるが、absenceをRepository全体の否定事実へ拡張しない。

```text
no direct ref in current Business source
  != no possible Application support anywhere
```

### P1-07 — BP-07は`submit_order`を通じてCapabilityを呼ぶか

accepted path:

```text
BP-07
  -> applicationPatterns[order_entry.submit_order]
  -> Pattern order_entry.submit_order command
  -> capability sales.order.submit
  -> realization: custom go
```

結果: **supported**

supportedなのはCapability invocation chainまでであり、persistence mutationはP3で別に問う。

## 5. P2 — Business / Data Trace

P2ではDataへ到達できることと、CRUD operationを持つことを分離する。

### P2-00 — BP-00から`sales.order`へtraceできるか

accepted path:

```text
BP-00
  -> order_entry.recent_orders
  -> data sales.order
```

結果: **supported**

Queryも明示されているためbrowseのread intentは別途Evidence-backedに扱えるが、本proofではData traceだけを主張する。

### P2-01 — BP-01から`sales.order`へtraceできるか

accepted path:

```text
BP-01
  -> order_entry.order_record
  -> data sales.order
```

結果: **supported**

ただし:

```text
Data trace supported
  != read supported
  != update supported
```

### P2-02 — BP-02からCustomer relation / Dataへtraceできるか

accepted path:

```text
BP-02
  -> order_entry.customer_reference
  -> relation sales.order.customer
  -> semantic relation target customer.customer
```

加えてPatternは:

```text
datasource master
contract master_data.customer
query find_by_id
```

を明示する。

結果: **supported**

### P2-03 — BP-03からOrderLineへtraceできるか

accepted path:

```text
BP-03
  -> order_entry.order_lines
  -> relation sales.order.lines
  -> sales.order_line
```

結果: **supported**

### P2-04 — BP-04はData dependencyを持たないか

結果: **unresolved**

BP-04にはcurrent exact Data traceがないが、これは`No Data`を意味しない。

```text
no current Data trace
  != no Data dependency
```

この結果は既存Lane C原則と一致する。

## 6. P3 — Function / Mutation Authority

P3は今回もっとも重要なnegative pressureである。

### P3-01 — `order_record`は`sales.order`を更新できるか

観測できるauthority:

```text
Pattern order_entry.order_record record
  -> data sales.order

Application order_entry
  database sales
  access read-write
```

結果: **unresolved**

理由:

- `record` kindはone-record intentでありupdateを意味しない
- Applicationの`read-write`はdatasource利用可能性でありBusiness Actionのmutation attributionではない
- PatternとDatabase Contract writeを結ぶaccepted operation edgeがない

したがって:

```text
record + read-write datasource
  -X-> U sales.order
```

### P3-02 — `customer_reference`はmaster Customerを変更できるか

観測できるauthority:

```text
Pattern order_entry.customer_reference reference
  datasource master
  contract-query master_data.customer.find_by_id

Application order_entry
  database master
  access read
```

候補主張:

```text
BP-02 / customer_reference may mutate customer master through declared datasource
```

結果: **contradiction**

理由:

current Application authorityは`master`を`access read`として明示する。

この候補は明示的datasource policyと両立しないため、Design Synthesis candidateから理由付きで除外できる。

この判定は「reference Patternだからreadだろう」というkind推測だけに依存しない。

### P3-07 — BP-07は`sales.order.status`を更新するか

accepted chain A:

```text
BP-07
  -> order_entry.submit_order
  -> sales.order.submit
  -> custom go realization
```

accepted chain B:

```text
Database Contract sales_core.order v4.2
  -> write sales.order.status
```

不足するedge:

```text
sales.order.submit
  -> mutates sales.order.status
```

または同等のaccepted realization relation。

結果: **unresolved**

重要なのは、次のすべてが存在してもUを導出しないことである。

- command Pattern
- Capability input / outputが`sales.order`
- custom Go realization
- Application datasourceがread-write
- Database Contractに`write sales.order.status`

これらは別々のauthorityであり、現在はoperation attribution relationが不足する。

## 7. P4 — Business / Security Boundary

### P4-06a — BP-06はAuthorization Capabilityを明示的に参照するか

Business source:

```text
BP-06
  realization = security-boundary
  capabilities = [security.authorize]
```

Semantic source:

```text
capability security.authorize
  input subject
  input action
  input resource
  outcome allowed
  outcome denied
```

結果: **supported**

ここでsupportedなのは、Business ActionがAuthorization supportを必要とし、generic Capabilityをexact参照していることまでである。

### P4-06b — BP-06のprotected targetは`order_entry.submit_order`か

Security-side authorityには次が存在する。

```text
organization = SALES
role         = ORDER_EDITOR
action       = execute
resource     = order_entry.submit_order
effect       = allow
```

一方、Primary Business sourceには`protected target = order_entry.submit_order`を保持するproduction fieldがない。

結果: **unresolved**

Legacy policyはtarget候補をcorroborateできるが、Business targetをauthor / backfillしてはならない。

したがって:

```text
policy says execute order_entry.submit_order
  -X-> persist BP-06 target = order_entry.submit_order
```

### P4-06c — Business actor `sales operator`はAuthorization Role `ORDER_EDITOR`か

Business source:

```text
actor.status = candidate
actor.label  = sales operator
```

Security source:

```text
role_code = ORDER_EDITOR
```

両者を結ぶaccepted mappingはない。

結果: **unresolved**

名称や業務常識だけでmappingしない。

### P4-SEC-01 — Security側で`ORDER_EDITOR`がsubmit実行を許可されるか

問いをSecurity-side factへ限定する。

accepted path:

```text
legacy authorization rule
  SALES / ORDER_EDITOR / execute / order_entry.submit_order / allow
```

結果: **supported**

この結果をBusiness actorやBP-06 targetへ逆流させない。

## 8. R1 proof matrix

| ID | Family | 結果 | Design Synthesisでの扱い |
| --- | --- | --- | --- |
| P1-00 | Business -> recent orders realization | supported | exact reuse候補として保持 |
| P1-01 | Business -> order record realization | supported | exact reuse候補として保持 |
| P1-02 | Business -> customer reference realization | supported | exact reuse候補として保持 |
| P1-03 | Business -> order lines realization | supported | exact reuse候補として保持 |
| P1-04a | BP-04 current source is manual-unresolved | supported | manual residual meaningとして保持 |
| P1-04b | BP-04 has no Application support anywhere | unresolved | 否定事実にしない |
| P1-07 | submit Business -> Pattern -> Capability | supported | exact invocation chainとして保持 |
| P2-00 | BP-00 -> sales.order trace | supported | Data traceとして利用 |
| P2-01 | BP-01 -> sales.order trace | supported | Data traceのみ、CRUDへ拡張しない |
| P2-02 | BP-02 -> customer relation/Data trace | supported | relation / read path候補として利用 |
| P2-03 | BP-03 -> order lines trace | supported | relation traceとして利用 |
| P2-04 | BP-04 has no Data dependency | unresolved | `No Data`へ丸めない |
| P3-01 | order_record updates sales.order | unresolved | human / later operation authorityへ残す |
| P3-02 | customer_reference mutates master | contradiction | candidateを理由付き除外可能 |
| P3-07 | submit mutates sales.order.status | unresolved | operation attribution gapとして保持 |
| P4-06a | BP-06 uses security.authorize | supported | exact support factとして保持 |
| P4-06b | BP-06 protected target is submit_order | unresolved | policyからbackfillしない |
| P4-06c | sales operator == ORDER_EDITOR | unresolved | actor-role mappingを発明しない |
| P4-SEC-01 | ORDER_EDITOR may execute submit_order | supported | Security-side factとしてのみ保持 |

## 9. Candidate contractionの具体例

### Candidate A — Customer確認でmasterを更新する

```text
BP-02
  -> customer_reference
  -> mutate customer master
```

P3-02で**contradiction**。

明示的`access read`と両立しないため除外できる。

### Candidate B — Submitは`sales.order.status`を更新する

直感的にはもっともらしい。

しかしP3-07は**unresolved**。

Database write authorityは存在するが、Capability / Patternとのattribution edgeがないため、自動採用も自動除外もしない。

Design Synthesisへ残す問いは次まで縮約できる。

> `sales.order.submit`のaccepted effectとして`write sales.order.status`を結び付けるauthorityはどこに置くべきか。それともcurrent Referenceでは結び付けないか。

### Candidate C — BP-06 targetをAuthorization policyから自動補完する

これはdomain上のtarget真偽を判定する前に、authority direction違反である。

```text
Security policy
  -> Business target authoring
```

を許可しない。

したがってproposal生成器 / LLMへは、policyをcorroboration Evidenceとして渡しても、Business sourceの欠落fieldを自動補完するauthorityとして渡さない。

### Candidate D — BP-04はDataを使わない

P2-04で**unresolved**。

current traceが無いことをnegative factへ変えないため、この候補は確定できない。

## 10. DesignContextPacketへ渡せる最小形

R1の結果から、LLM前段へ次のようなbounded packetを作れる。

```text
purpose:
  review / synthesize order-entry business design without changing authority

exact_facts:
  BP-00 -> order_entry.recent_orders
  BP-01 -> order_entry.order_record
  BP-02 -> order_entry.customer_reference
  BP-03 -> order_entry.order_lines
  BP-06 -> security.authorize
  BP-07 -> order_entry.submit_order -> sales.order.submit

proof_results:
  P3-02 contradiction:
    master datasource is read-only for order_entry

  P3-07 unresolved:
    Database Contract can write sales.order.status
    but no accepted capability-to-write attribution exists

  P4-06b unresolved:
    security policy corroborates order_entry.submit_order
    but Business source does not persist protected target

  P4-06c unresolved:
    no accepted mapping from sales operator to ORDER_EDITOR

authority_limits:
  do not infer CRUD from Pattern kind
  do not infer Business target from Security policy
  do not infer Authorization Role from Business actor label
  do not turn missing Data trace into no dependency
```

この形なら、local LLMへRepository全体を渡さなくても、「何が既知で、何が本当に未解決か」を小さくできる。

## 11. R1から得られた設計判断

### 11.1 三状態は実用上必要

R1では`supported`だけでなく、`contradiction`と`unresolved`がどちらも実際に現れた。

特に:

```text
customer master mutation
  -> contradiction

submit -> sales.order.status mutation attribution
  -> unresolved
```

は意味が異なる。

これを一つの`invalid`やscoreへ畳むと、Design Synthesisが誤って候補を消す。

### 11.2 P3のunresolvedはBusiness Procedure schema不足ではない

submit mutation gapは、current evidenceでは:

```text
Capability / realization / Database effect boundary
```

側の不足である可能性が高い。

そのためBusiness ProcedureへCRUD fieldを追加しない。

### 11.3 P4のunresolvedもBusiness targetをpolicyから補う理由にならない

Security側Evidenceが豊富でも、Business-side meaningのownershipは別である。

これはcross-viewが「複数sourceを一つへmergeする仕組み」ではなく、「独立authorityを相互牽制させる仕組み」であることを示す。

### 11.4 Derived Meaning Graphはまだ永続化不要

R1の全proofは、既存のexplicit refsと小さい一時的なderivation pathで説明できる。

現時点では次の新規Core要素を正当化するlossは観測されていない。

- persisted `ProofObligation`
- generic `Claim`
- generic `SemanticGraph`
- graph database
- global revision counter
- persisted proof result enum

## 12. R1 verdict

```text
P1 Business / Realization
  -> exact refsで成立

P2 Business / Data Trace
  -> exact refsで部分的に成立
  -> absenceはunresolvedとして保持可能

P3 Mutation Authority
  -> explicit read-only contradictionは検出可能
  -> submit mutation attributionはunresolvedとして残る

P4 Business / Security
  -> generic Authorization supportはexactに成立
  -> protected target / actor-role mappingはauthority境界によりunresolved

therefore
  bounded proof obligationモデルはorder_entryで有効
  new Core proof schemaはまだ不要
```

## 13. 次の作業

R1でP1〜P4の基本挙動は確認できた。

次は[`issue/000050-second-realistic-handoff-corpus.md`](issue/000050-second-realistic-handoff-corpus.md)と既存manufacturing / inventory Pressureを使い、R2としてP5 / P6を検証する。

重点は次である。

```text
Business instruction
  != command acceptance
  != external / physical actuality
  != Business acceptance
  != responsibility Handoff
```

さらに:

```text
positive physical Observation
  -X-> Handoff generation

Step adjacency
  -X-> Handoff generation
```

をactual Reference evidenceで再確認する。

R2でも既存sourceで十分ならCore schemaを増やさない。具体的なlossが反復して初めてdurable proof identity、effect attribution、protected-target role等を再検討する。
