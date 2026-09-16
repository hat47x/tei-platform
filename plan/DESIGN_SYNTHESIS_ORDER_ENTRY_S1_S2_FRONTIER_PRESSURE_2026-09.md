# Design Synthesis Order Entry S1 / S2 Frontier Pressure — 2026-09

Status: **Lane D / read-only Acceptance pressure / pre-Gate-C / 新規Product contract未採用**

## 1. 目的

`DESIGN_SYNTHESIS_SCENARIO_ACCEPTANCE_DRAFT_2026-09.md`で定義したS1 / S2を、current `order_entry`のaccepted sourceだけをauthorityとして使い、ひとつのsynthetic partial intentへ実際に当てる。

目的はcandidate generatorを実装することでも、最適案を選ぶことでもない。

確認するのは次である。

1. S1で単なる言い換えではないmeaning-different candidateを保持できるか
2. existing accepted assetとproposal-only meaningを同じauthorityへ畳まないか
3. S2でexact contradictionだけを除外し、根拠不足のproposalを`false`へ丸めないか
4. contraction後のfrontierが複数案を含んでも、それを失敗とみなさず残余判断へ渡せるか
5. `supported existing candidate`と`unresolved proposal candidate`が同時に残る場合、proof結果をwinner scoreへ変換しないか

新しいCanonical kind、candidate schema、ranking score、proof lifecycle、AcceptanceSnapshot、LLM contractは追加しない。

## 2. 評価snapshot

評価基線:

```text
main @ e197e997b5e6c7116a85ff64e946e78b6a58054f
```

これはProduct上のglobal semantic revisionではない。本Pressureで読んだsource集合を再現するための評価時点である。

主なaccepted authority:

### Application

`reference/workspace/applications/order-entry/application.tei`

```text
database sales
  access read-write

database master
  target master_data
  access read
  requires master_data.customer ^2
```

### Existing Application Pattern

`reference/workspace/applications/order-entry/patterns/customer-reference.tei`

```text
pattern order_entry.customer_reference reference
  relation sales.order.customer
  datasource master
  contract master_data.customer
  query find_by_id
```

### Semantic

`reference/workspace/semantic/sales/order.tei`

```text
sales.order
  relation customer -> customer.customer one
```

### Existing proof pressure

`DESIGN_SYNTHESIS_ORDER_ENTRY_PROOF_REFERENCE_2026-09.md`

特に次を再利用する。

```text
P2-02
  BP-02 / customer_reference -> customer relation / Data trace
  = supported

P3-02
  customer_reference may mutate customer master
  = contradiction
```

R1のproofを作り直さず、S1 / S2 frontierへ適用する。

## 3. synthetic partial intent

今回の入力はcurrent Business sourceのBP-02そのものではない。

accepted Business fixtureにはすでに`BP-02 -> order_entry.customer_reference`のrealizationがあるため、それをpartial intentとして使うと既存realizationを意図的に隠すことになる。

そこで、Gate C後にあり得る小さい編集要求を模したsynthetic intentを使う。

```text
I-CUSTOMER-CONFIRM

目的:
  orderを準備するとき、operatorがそのorderに関連するcustomerを確認できること。

既知制約:
  customer masterを変更できることは要求しない。

未確定:
  existing customer referenceをそのまま再利用するか
  order側へ別のcustomer snapshotを持つか
  どの表示項目が必要か
```

これはaccepted requirementではなく、S1 / S2 AcceptanceをPressureするための入力である。

## 4. S1 candidate frontier

### C-A — existing `customer_reference`を再利用する

semantic claim:

```text
orderとcustomerのaccepted relationを辿り、
既存order_entry.customer_referenceを使って
master datasourceからcustomerをreadする。
```

classification:

```text
existing candidate
current M1 identity = order_entry.customer_reference
```

current accepted support:

- `sales.order.customer` relationが存在する
- relation targetは`customer.customer`
- `order_entry.customer_reference`がそのrelationを参照する
- Patternは`datasource master`と`master_data.customer.find_by_id`を参照する
- Applicationは`master` datasourceを`access read`として宣言する

このcandidateは「既存資産として現在存在し、customer read pathへexactに戻れる」点でsupportされる。

ただし、この時点で新しいsynthetic intentに対するProduct上の最終採用案になったとは言わない。

### C-B — order-owned customer snapshotを新設する

semantic claim:

```text
customer確認に必要な一部情報を、
order側のsnapshotとして別に保持する。
```

想定されるmeaning difference:

- data ownershipがcustomer masterだけではなくorder側にも生じる
- freshness / copy timing / history semanticsが必要になる
- current customer値とorder時点snapshotを区別する可能性がある

classification:

```text
proposal-only candidate
accepted Canonical identity = none
```

重要:

このPressureのために`customer_snapshot`等の架空Canonical IDを作らない。

current accepted sourceには、このsnapshotを必要とするBusiness requirement、Data ownership、copy timing、freshness ruleはない。

結果:

```text
unresolved proposal
```

これは「不適切」と証明したことを意味しない。

たとえば将来、注文時点のcustomer情報を監査目的で固定するaccepted requirementが追加されれば、同じproposal familyの評価は変わり得る。

### C-C — order handlingからcustomer masterを直接更新する

semantic claim:

```text
customer確認画面からmaster customerを編集・更新できる。
```

classification:

```text
proposal candidate
```

meaningはC-A / C-Bと異なる。

- C-Aはread-only reuse
- C-Bはorder側への新しいcopy semantics
- C-Cはmaster mutation authority

したがって単なるUI表現違いではない。

## 5. S1 Acceptance pressure

この3案はS1のcandidate diversityを次のようにPressureする。

### surface variantではない

```text
C-A = existing read path reuse
C-B = new order-owned snapshot semantics
C-C = master mutation authority
```

三者はownership / freshness / mutation authorityが異なる。

### existing / proposal authorityを分離できる

```text
C-A
  exact existing identityあり

C-B / C-C
  proposal-only
  accepted identityなし
```

C-B / C-Cへ架空のaccepted refを割り当てない。

### candidate countを目的化しない

今回3案を置けるからといって、S1が常に3案以上を生成すべきというRequirementにはしない。

必要なのは、meaning-different alternativeが存在する場合に、言い換えへ潰さず保持できることである。

## 6. S2 contraction

### 6.1 C-A — retain

R1 P2-02で、次のexact traceは`supported`である。

```text
order
  -> sales.order.customer
  -> customer.customer
  -> order_entry.customer_reference
  -> master_data.customer.find_by_id
```

さらにApplication authorityはmaster datasourceを`access read`としている。

したがってC-Aのread-only reuse claimはcurrent accepted sourceと整合する。

扱い:

```text
retain as supported existing candidate
```

ただし:

```text
supported existing candidate
  != whole candidate accepted
  != winner
```

### 6.2 C-C — eliminate

C-Cはcustomer master mutationを要求する。

一方、Application authorityは:

```text
database master
  access read
```

を明示する。

R1 P3-02と同じscopeで、candidate claimはcurrent accepted authorityと両立しない。

結果:

```text
contradiction
```

したがってS2で説明可能に除外できる。

compact elimination reason:

```text
candidate: C-C
claim: order handling may mutate customer master
decisive proof: P3-02
accepted authority: order_entry master datasource access = read
result: eliminate
```

### 6.3 C-B — retain unresolved

C-Bにはcurrent accepted identityも、snapshot requirementも、copy timing / freshness authorityもない。

しかしcurrent sourceに:

```text
order-owned customer snapshot is forbidden
```

というauthorityもない。

したがって:

```text
no accepted support
  != contradiction
```

結果:

```text
retain as unresolved proposal candidate
```

C-Bを「existing assetがあるから不要」とだけ判定してS2で落としてはならない。

既存再利用を優先するProduct / design objectiveがaccepted constraintとして与えられれば、その追加条件を別に評価する。

## 7. S2後のfrontier

contraction result:

```text
C-A  retained  supported existing candidate
C-B  retained  unresolved proposal candidate
C-C  eliminated contradiction
```

ここで重要なのは、S2が一つのwinnerを返していないことである。

これはFailureではない。

S2の責務は:

```text
exact accepted constraintで落とせるclaimを落とす
  +
落とせないunresolvedを残す
```

ことであり、根拠不足のproposalを強制的に除外してfrontierを1件へ縮めることではない。

## 8. このPressureで確認できた追加境界

### 8.1 Survivedは同じ意味ではない

S2後に残るcandidateは同じepistemic statusを持たない。

```text
C-A
  current accepted supportあり

C-B
  contradictionはないがaccepted supportも不足
```

したがって:

```text
survived
  != supported
```

である。

### 8.2 supportedをwinner scoreへ変換しない

C-Aがsupportされているからといって、S2がC-Aを自動採用してはならない。

たとえばsynthetic intentへ次が追加されれば評価は変わる。

```text
「注文確定時点のcustomer表示値を後から再現できること」
```

この場合C-B系proposalへ新しい意味上の必要性が生じる可能性がある。

逆に:

```text
「新しい永続copyは作らずcurrent masterを参照する」
```

がaccepted constraintなら、C-Bをさらに縮約できる。

### 8.3 minimum semantic deltaはS2 proofではない

TEIはS3でminimum semantic deltaを優先評価する方向だが、これはS2のcross-view proofを置き換えない。

```text
existing reuseの方が変更量が小さい
  != accepted authorityがproposalをcontradictした
```

したがってexisting reuse / minimum deltaの比較は:

- explicit accepted objective
- S3 comparison
- human review

へ渡す。

proof resultをPreference scoreへ変換しない。

## 9. Acceptance draftへの反映点

このPressureから、S2 Acceptanceには次を明示すべきである。

1. S2はfrontierを必ず1 candidateへ縮約する必要はない。
2. `supported existing`と`unresolved proposal`が同時に残ることを許容する。
3. `survived`を`validated / recommended / accepted`の同義語にしない。
4. unresolved proposalをwinner選択のためだけに除外しない。
5. existing reuse / minimum semantic delta等のpreferenceは、accepted constraintでない限りcontradictionとして扱わない。

Negative Acceptanceとして:

```text
proof support数でwinnerを決める
unresolved candidateを1案化のために捨てる
existing assetがあるという理由だけでnew proposalをcontradiction扱いする
```

を追加する価値がある。

## 10. 非主張範囲

本Pressureから次は主張しない。

- C-AがProduct上の最善設計である
- C-Bが必要である
- customer snapshotをCanonical kindへ追加すべきである
- S1 generatorがactual user intentから十分なcandidateを生成できる
- S2 engineがactual runtimeでincrementalに動作する
- Gate C後のDesign Synthesis Loopが成立した
- local LLMがC-A / C-Bを適切に比較できる

本書が確認するのは、**S1 / S2 Acceptanceの意味境界をcurrent accepted sourceへ当てたとき、frontierを無理に1案へ潰さず、support / contradiction / unresolvedをcandidate-localに保持できること**までである。
