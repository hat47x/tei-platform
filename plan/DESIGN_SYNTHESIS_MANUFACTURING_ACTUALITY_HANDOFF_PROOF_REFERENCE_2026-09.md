# Design Synthesis Manufacturing Actuality / Handoff Proof Reference — 2026-09

状態: **Lane D / R2 negative proof evaluation / second realistic Reference未成立（R2評価時点。2026-09-21にISSUE-000050でauthor済み） / 新規Product contract未採用**

## 1. 目的

[`design/DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`](design/DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md)で定義したP5 / P6を、manufacturing / inventory second-corpus pressureへ適用する。

R1では既存`order_entry`を使い、P1〜P4について`accepted authorityから支持できること`と`未解決として残すこと`を確認した。

R2では、まだaccepted second Business sourceが成立していない現状をそのまま使い、次を確認する。

```text
Business instruction
  != Application Command acceptance
  != external / physical actuality
  != Business acceptance / continuation
  != responsibility Handoff
```

本書はsecond realistic Handoff Evidenceそのものではない。

特に次を主張しない。

- RRB-P1達成
- RRB-P2達成
- RRB-P3達成
- second Handoff proven
- Contract ADR blocker B解消
- Gate 15c activation
- production Handoff Canonical kind採用
- production Observation Canonical kind採用

## 2. 評価基線

R2は次の既存文書・pressureを入力とする。

### second-corpus scope / acceptance

- [`issue/000050-second-realistic-handoff-corpus.md`](issue/000050-second-realistic-handoff-corpus.md)
- [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md)
- [`BUSINESS_PROCEDURE_MANUFACTURING_INVENTORY_CORPUS_PRESSURE_OUTLINE_2026-09.md`](BUSINESS_PROCEDURE_MANUFACTURING_INVENTORY_CORPUS_PRESSURE_OUTLINE_2026-09.md)

### actuality / Observation

- [`BUSINESS_PROCEDURE_PHYSICAL_OBSERVATION_PRESSURE_2026-09.md`](BUSINESS_PROCEDURE_PHYSICAL_OBSERVATION_PRESSURE_2026-09.md)

### Handoff inference

- [`BUSINESS_PROCEDURE_SECOND_CORPUS_PRESSURE_2026-09.md`](BUSINESS_PROCEDURE_SECOND_CORPUS_PRESSURE_2026-09.md)（Part 3、旧`SECOND_CORPUS_HANDOFF_INFERENCE_PRESSURE`）

これらが共有する現在地は次である。

```text
manufacturing / inventory continuation
  -> useful independent pressure exists

accepted TEI-native second Business source
  -> not yet established

exact actor / trigger / sequence / Data / realization
  -> unresolved

exact Handoff from / to / condition
  -> unresolved

physical Observation pressure
  -> test-only negative boundary exists

Handoff non-inference pressure
  -> test-only negative boundary exists
```

したがってR2はpositive second-Handoff Referenceではなく、**authorityを越えた短絡を除外できるかのnegative proof Reference**として扱う。

## 3. proof resultの読み方

R1と同じ三状態を使う。

```text
supported
contradiction
unresolved
```

ただしP5 / P6では、次を明確に区別する。

### domain claim

例:

```text
manufacturing physically completed
```

これを証明するEvidenceがなければ`unresolved`である。

### derivation rule claim

例:

```text
Command acceptedならmanufacturing physically completedとみなしてよい
```

既存authorityがこの短絡を明示的に禁止している場合、この**導出規則**は`contradiction`として除外できる。

つまり:

```text
physical completion happened?
  -> unresolved

Command acceptance is sufficient proof of physical completion?
  -> contradiction
```

を同時に保持できる。

これはDesign Synthesisで非常に重要である。

## 4. P5 — System Result / Actuality

### P5-01 — Command acceptanceだけでphysical completionを証明できるか

second-corpus pressureは次を明示する。

```text
Command accepted
  != material physically received
  != manufacturing physically completed
  != finished goods physically present
  != shipment physically completed
```

候補導出規則:

```text
CommandAccepted == true
  -> PhysicalCompletion == true
```

結果: **contradiction**

理由:

既存Driver / Observation境界とphysical Observation pressureが、Command invocation resultとexternal actualityを独立に扱う。

したがってDesign Synthesisは、Command成功だけを根拠にphysical completionを含む候補を採用しない。

### P5-02 — Command accepted後、physical completionは起きていないと言えるか

結果: **unresolved**

P5-01が否定したのは「Command成功からcompletionを証明する導出」であり、物理世界でcompletionが起きなかったことを証明したわけではない。

```text
not proven complete
  != proven incomplete
```

### P5-03 — Observation readが成功すればphysical completionを証明できるか

既存pressureではread成功とResultの意味を分離している。

候補導出規則:

```text
Observation execution success
  -> physical completion
```

結果: **contradiction**

Transport / Query / read execution成功は、観測対象のpredicateがtrueであることを意味しない。

### P5-04 — explicit external Observation result=trueならobserved physical factを支持できるか

scopeを次へ限定する。

```text
external-observed read
  Result = true
  -> this test-local physical fact was observed
```

結果: **supported**

ただしsupportedなのは**observed physical fact**までである。

次へは拡張しない。

```text
observed physical fact
  -X-> Business acceptance
  -X-> responsibility Handoff
  -X-> exact TEI Data authority
  -X-> Canonical Observation identity
```

### P5-05 — Observation read errorからphysical stateを決められるか

結果: **unresolved**

read errorはpositive / negative actualityへ変換しない。

Commandがすでに成功していても同じである。

### P5-06 — positive physical ObservationならBusiness continuation conditionは成立したか

結果: **unresolved**

理由:

Handoff / Business continuation conditionはBusiness predicateであり、その成立を支えるObservationと同一identityではない。

たとえば将来:

```text
condition:
  finished goods have been accepted for warehouse responsibility

evidence:
  physical stock observation = true
```

という関係があり得ても、ObservationだけからBusiness acceptance predicate全体をauthorできるとは限らない。

## 5. P6 — Responsibility Handoff

### P6-01 — Step adjacencyからHandoffを作れるか

既存negative pressure:

```text
ordered steps exist
explicit Handoff source absent
  -> normalized Handoffs = 0
```

候補導出規則:

```text
Step A immediately precedes Step B
  -> Handoff A -> B
```

結果: **contradiction**

これはcurrent H1/R3 authority directionと両立しない。

ただし「AからBへのHandoffが現実には存在しない」というdomain claimを証明するものではない。

Handoff existence自体はaccepted sourceがなければ`unresolved`である。

### P6-02 — actor changeからHandoffを作れるか

second-corpus acceptance boundaryはactor差だけをtransfer根拠にしない。

候補導出規則:

```text
actor(A) != actor(B)
  -> Handoff A -> B
```

結果: **contradiction**

同様にactorが同じことだけを理由にHandoffを否定もしない。

### P6-03 — positive physical ObservationからHandoffを作れるか

既存pressure:

```text
physical fact observed = true
explicit Handoff source absent
  -> normalized Handoffs = 0
```

候補導出規則:

```text
PhysicalCompletionObserved
  -> responsibility Handoff exists
```

結果: **contradiction**

Observationはcontinuationの前提事実になり得るが、`from / to / condition`を所有しない。

### P6-04 — Application Function sequenceからHandoffを作れるか

結果: **contradiction**

Business Procedure / HandoffとRuntime / Application sequenceを分離する既存authorityに反する。

### P6-05 — second corpusにresponsibility Handoffが存在するか

現在のISSUE-000050は明示する。

```text
from / to / condition
  -> still unresolved
```

accepted TEI-native second Business sourceも未成立である。

結果: **unresolved**

ここを「negative pressureが十分だからHandoffあり」と変換しない。

2026-09-21追記: 上の判定はR2評価時点のものであり、ここでは変更しない。その後ISSUE-000050で、独立したBusiness needからsecond Referenceをauthorし、Handoffを一件置いた（[実施記録](issue/000050-second-realistic-handoff-corpus.md)）。

### P6-06 — second corpus Handoffのfrom / toを特定できるか

結果: **unresolved**

Step sequence candidate、複数responsibility context、physical completion boundaryは存在するが、accepted Procedure-scoped endpointsはまだない。

### P6-07 — second corpus Handoff conditionを特定できるか

結果: **unresolved**

次をconditionへ代入しない。

- Observation ID
- runtime event
- Command result
- physical completion factそのもの
- target Step trigger
- 「前Stepが終わったら」という根拠のないtautology

conditionはresponsibility continuationを成立させるBusiness predicateとしてaccepted sourceからlosslessに保持できる必要がある。

## 6. R2 proof matrix

| ID | Family | 結果 | Design Synthesisでの扱い |
| --- | --- | --- | --- |
| P5-01 | Command acceptance proves physical completion | contradiction | この短絡を使う候補を除外 |
| P5-02 | Command accepted means physical completion did not occur | unresolved | negative actualityを発明しない |
| P5-03 | Observation execution success proves completion | contradiction | transport/read successとfactを分離 |
| P5-04 | explicit external Result=true supports observed fact | supported | observed factだけを保持 |
| P5-05 | Observation error determines physical state | unresolved | unknownを保持 |
| P5-06 | positive Observation proves Business continuation condition | unresolved | Business predicateへbackfillしない |
| P6-01 | Step adjacency authors Handoff | contradiction | 候補生成規則として禁止 |
| P6-02 | actor change authors Handoff | contradiction | 候補生成規則として禁止 |
| P6-03 | positive physical Observation authors Handoff | contradiction | Handoff inferenceに使わない |
| P6-04 | Function sequence authors Handoff | contradiction | Runtime sequenceから分離 |
| P6-05 | second corpus has accepted Handoff | unresolved | blockerを解消しない |
| P6-06 | second corpus Handoff endpoints are known | unresolved | from/toを発明しない |
| P6-07 | second corpus Handoff condition is known | unresolved | conditionを発明しない |

## 7. Design Synthesisでのcandidate contraction

### Candidate A — completion commandが成功したので製造完了とする

```text
command result = accepted
  -> manufacturing completed
```

P5-01で**contradiction**。

候補から除外できる。

### Candidate B — completion commandは成功したが製造は未完了だった

P5-02で**unresolved**。

Commandだけではpositive actualityもnegative actualityも決めないため、候補を事実として採用しない。

必要ならexplicit Observation / Evidenceを要求する。

### Candidate C — physical completionが観測されたので倉庫責任へ自動Handoffする

P5-04でphysical fact自体はsupportedになり得る。

しかしP6-03で、そこからHandoffを自動生成する導出規則は**contradiction**。

したがって:

```text
physical fact supported
+
Handoff unresolved
```

を同時に保持する。

### Candidate D — 製造Stepの次に在庫StepがあるのでHandoffを作る

P6-01で**contradiction**。

adjacencyはBusiness responsibility authorityではない。

### Candidate E — Handoffの存在はありそうなのでfrom / to / conditionをLLMに補完させる

P6-05〜07で**unresolved**。

LLMはproposalとして候補を出せても、accepted second sourceなしにFactへ昇格できない。

この場合Design SynthesisがLLMへ渡すべき問いは:

> responsibility transferをaccepted Business meaningとして置く根拠はあるか。ある場合、from / to / conditionをどのsourceで明示できるか。

であり、

> もっともらしいHandoffを書いてください。

ではない。

## 8. bounded Context Packet

R2からLLM前段へ渡せる情報は次の程度まで縮約できる。

```text
purpose:
  evaluate manufacturing / inventory continuation without inventing actuality or handoff

known_pressure:
  instruction / command / physical actuality are distinct concerns
  positive physical observation may establish only the observed fact
  handoff requires explicit accepted Business authority

proof_results:
  CommandAccepted -> PhysicalCompletion
    contradiction

  ReadExecutionSuccess -> PhysicalCompletion
    contradiction

  ExternalObservation(Result=true) -> ObservedPhysicalFact
    supported

  ObservedPhysicalFact -> BusinessAcceptance
    unresolved

  ObservedPhysicalFact -> Handoff
    contradiction as derivation rule

  StepAdjacency -> Handoff
    contradiction as derivation rule

  SecondCorpusHandoffExists
    unresolved

  SecondCorpusHandoffEndpointsCondition
    unresolved

authority_limits:
  do not infer exact actor / trigger / sequence
  do not invent Data identity / Application realization
  do not make Observation a Canonical Business identity
  do not make Runtime sequence a Business Handoff
  do not satisfy ISSUE-000050 by inventing from/to/condition
```

このpacketは「答え」をLLMへ渡すのではなく、**LLMが考えてよい残余の境界**を作る。

## 9. R1 + R2で見えたproof modelの性質

### 9.1 contradictionはdomain falsityだけではない

R1 / R2では二種類のcontradictionが現れる。

```text
A. domain / authority contradiction
   customer master mutation
   vs explicit datasource access read

B. derivation-rule contradiction
   Command success -> physical completion
   Observation -> Handoff
   adjacency -> Handoff
```

将来実装する場合、この二つを診断理由として区別できる方がよい可能性がある。

ただし現時点でproduction enumを増やさない。

### 9.2 unresolvedは設計残差そのもの

R1:

```text
submit capability -> DB write attribution
protected target
actor -> Authorization Role
```

R2:

```text
actual physical completion
Business acceptance
second Handoff existence
from / to / condition
```

がunresolvedとして残った。

Design Synthesisの価値は、この残差を「AIが埋める空欄」ではなく、**本当に意味判断が必要なdimension**として小さく保持することにある。

### 9.3 proof modelはfalse completenessを抑止できる

候補生成器が空欄を埋めるほど見栄えは良くなるが、authorityは弱くなる。

R2ではaccepted second sourceがまだないこと自体が重要なEvidenceである。

```text
insufficient Evidence
  -> unresolved
  != generate plausible source
```

## 10. second Reference maturityとの関係

R2を完了してもRRB-P1にはならない。

RRB-P1には、order_entryと独立したprovenance-safe TEI-native Business sourceが必要である。

R2が提供するのは、そのsourceを将来作るときの**qualification oracle候補**である。

```text
future second source
  ↓
P5 / P6 proof obligations
  ↓
shortcuts rejected
unresolved preserved
  ↓
only then Handoff candidate if naturally evidenced
```

RRB-P2はさらにsource/test-level integrationを要求し、RRB-P3はactual runner greenを要求する。

したがってR2をbreadth Evidenceへ読み替えない。

## 11. Core contractへの含意

R2でも、次のproduction追加はまだ不要である。

- `Observation` Canonical kind
- `PhysicalFact` Canonical kind
- `BusinessAcceptance` Canonical kind
- `ProofObligation` Canonical kind
- `Claim` Canonical kind
- generic Handoff inference engine
- graph database
- persisted proof result enum
- LLM-generated Handoff authority

一方で将来のimplementation probeでは、proof diagnosticに少なくとも次を保持すると有用な可能性がある。

```text
question
result
reason category
supporting authority refs
missing authority / unresolved reason
```

これはまずderived reportで試し、Canonical化しない。

## 12. R2 verdict

```text
P5 System Result / Actuality
  Command -> physical completion short-circuit can be rejected
  read execution -> fact short-circuit can be rejected
  positive explicit Observation can support only observed fact
  Business acceptance remains unresolved without Business authority

P6 Responsibility Handoff
  adjacency / actor change / Function sequence / Observation inference can be rejected
  second-corpus Handoff itself remains unresolved
  from / to / condition remain unresolved

therefore
  proof model works even before positive second Reference exists
  negative authority Evidence is useful for candidate contraction
  ISSUE-000050 blocker must remain open
  RRB-P1/P2/P3 must remain unclaimed
```

## 13. 次の作業

R1 / R2でproof family P1〜P6の最初のReference applicationが一巡した。

次に進む候補は二つある。

### A — proof reportの最小implementation probe

既存Referenceから:

```text
question
result
refs
reason
unresolved
```

だけをderivedに生成し、R1の一部を機械再現する。

これはauthorityを増やさず、DesignContextPacket生成へ直接接続できる。

### B — ISSUE-000050のRRB-P1を待って再評価

accepted second Business sourceが成立した時点でR2を再実行し、P5/P6のpositive proofが増えるかを見る。

現時点ではAを先に行う方が、架空のsecond sourceを作らずDesign Synthesisの実装価値を確認できる。

2026-09-21追記: second Business source（self-reviewのみ）が成立したため、Bの再実行が可能になった。ただし未実施である。
