# Design Synthesis Candidate-Local Proof Packet Pressure — 2026-09

Status: **Lane D / R4 read-only evaluation / no production schema adoption**

## 1. この文書の役割

本書は、R3 `DESIGN_SYNTHESIS_CONTEXT_PACKET_PROOF_BOUNDARY_2026-09.md`で残したread-only pressureのうち、次の三点をR1 / R2の具体例へ当てて深める。

```text
A. packetへ全proofを入れず、current candidateに関係するproofだけを選べるか
B. eliminated candidateをどの粒度で残せば、除外理由を再現できるか
C. unresolvedを集約しても、proofごとの不足authorityを失わないか
```

目的は`DesignContextPacket`のproduction schemaを決めることではない。

また、Product Gate C前にAI runtimeを実装するものでもない。

本書が扱うのは、candidate contraction後にlocal LLM / human reviewへ渡す情報を、**authority境界を壊さず十分に小さくできるか**という設計評価である。

## 2. 上位境界

R3で次を固定した。

```text
DesignContextPacket
  = snapshot-bound / rebuildable review projection
  != authoritative source

supported
  = bounded claim support
  != candidate whole acceptance

contradiction
  = exact scopeならdeterministic contractionに利用可能

unresolved
  = candidate除外理由ではない
  = residual dimensionとして保持
```

またDefinition-Centered Feedback Loopでは、Product Gate C Ready前にproduction AI / Studio implementationへ進めない。

したがってR4でも次を採用しない。

- production `DesignContextPacket`型
- production proof engine
- persisted proof enum
- packet database
- generic Candidate lifecycle state machine
- AI provider contract
- automatic Acceptance

## 3. R4の中心仮説

R4ではpacketを候補単位の局所projectionとして考える。

```text
all available proof results
        ↓
exact candidate scope resolution
        ↓
Candidate-Local Proof Set
        ├ supported
        ├ contradiction
        └ unresolved
        ↓
contraction
        ├ eliminated candidate certificate
        └ frontier candidate context
              └ unresolved dimensions
```

重要なのは、Repositoryやproof inventory全体を候補へ流し込まないことである。

一方、単に「proof数を減らす」ことも目的ではない。

```text
small packet
  != incomplete packet
```

必要proofを落とした結果、candidateが安全に見えるようになることは許容しない。

## 4. candidateとproofの関係

production contractではないが、review上は次を区別する必要がある。

```text
Candidate
  candidate_id
  proposed claims[]

Proof Result
  proof_id
  question / scope
  result
  evidence refs
  missing authorities
```

candidateが直接`proof_id[]`をauthorすることは要求しない。

将来の実装では、candidate claimとproof question / subject / dependency closureをexactに照合し、適用可能なproofを導く方が自然である。

R4のtest-only pressureでproof IDを明示する場合も、それは**candidate-local applicabilityの代用表現**であり、production authoring requirementではない。

## 5. A — current candidateに関係するproofだけを選ぶ

### 5.1 proof family単位で全部入れない

同じfamilyだから関連するとは限らない。

たとえばP3 Mutation Authorityには:

```text
P3-02
  customer_reference may mutate read-only master
  -> contradiction

P3-07
  sales.order.submit mutates sales.order.status
  -> unresolved
```

がある。

`submit_order`候補へP3-02を適用してはいけない。

```text
same proof family
  != same candidate scope
```

### 5.2 無関係なcontradictionで候補を落とさない

候補:

```text
C-auth
  BP-06 uses security.authorize
  BP-06 protected target may be order_entry.submit_order
```

Relevant:

```text
P4-06a supported
P4-06b unresolved
```

Irrelevant:

```text
P3-02 read-only master mutation contradiction
```

R4では、全proofから一つでもcontradictionが見つかればcandidate invalid、とする実装をrejectする。

### 5.3 必要proofが見つからない場合は黙って省略しない

candidate claimに対応するproof obligationが必要なのに、current snapshotで解決できない場合:

```text
proof missing
  -> no contradiction
```

として候補を安全扱いしてはいけない。

少なくともreview上は:

```text
candidate proof coverage unresolved
```

として残す。

これは新しいproduction enumを要求しない。

意味上、必要な問い自体が評価されていないことを、`supported`と区別できればよい。

### 5.4 heuristic relationでproof applicabilityを拡張しない

次をproof selection根拠にしない。

- name similarity
- same actor label
- adjacent Business Step
- same Application prefix
- same Data neighborhoodだけ
- embedding similarityだけ
- past candidateで一緒に使われたこと

Derived cognitionは候補proofを**探索**できるが、deterministic contractionへ入るproof applicabilityはexact scopeへ戻れる必要がある。

## 6. candidate-local dependency closure

packetを小さくするには、候補の意味から必要なauthorityまでdependency closureする。

例:

```text
candidate
  customer_referenceからmaster customerを変更する

needed proof scope
  customer_reference realization
  datasource access / mutation authority

not needed for this claim
  BP-06 Authorization protected target
  manufacturing physical Observation
  Handoff condition
```

closureの終端はaccepted authorityである。

```text
candidate claim
  -> exact refs / proof question
  -> accepted authority
```

Derived graphの近接edgeを無制限に辿り、Repository全体へ膨らませない。

## 7. B — eliminated candidateの保持粒度

候補をcontradictionで除外した後、LLMへその候補の全詳細を渡す必要は通常ない。

一方、理由を完全に捨てるとreviewabilityを失う。

そこでR4では、概念的な**elimination certificate**を評価する。

```text
Elimination Certificate
  candidate_id
  contradicted_claim
  decisive_proof_refs[]
  authority / evidence refs[]
  reason
  evaluated snapshot ref
```

これはproduction schemaではない。

### 7.1 full proof logを保持しない

candidateがP1〜P6の多数proofを通過した後、一つの必須claimがP3 contradictionになった場合でも、packet本文へ全proof logを残す必要はない。

最低限:

```text
candidate C1 eliminated
because P3-02 contradiction
against datasource master access=read
```

を再現できれば、後段LLMがC1を再候補化しにくい。

### 7.2 contradictionがcandidateの構成的claimに当たることを確認する

一つのcontradictionがcandidate全体を除外できるのは、そのclaimがcandidateの構成に必須である場合である。

修正可能なoptional proposalだけが矛盾する場合は:

```text
same candidate magically repaired
```

とはせず、矛盾部分を除いた**別candidate variant**として再生成する方が境界が明確である。

これによりproof resultを書き換えずに済む。

### 7.3 eliminated candidateはAcceptance historyではない

除外履歴はM3 / Evaluation Evidenceになり得るが:

```text
past eliminated
  != future universally invalid
```

である。

新snapshotでauthorityが変われば再評価可能である。

## 8. C — unresolved dimensionの集約

LLMへproofごとの生ログを大量に渡すより、意味上の残余dimensionへ集約した方がreviewしやすい。

ただし集約でauthority不足を失ってはいけない。

### 8.1 安全な集約条件

複数unresolvedを一つのdimensionへまとめる場合でも、少なくとも次を保持する。

```text
Unresolved Dimension
  dimension meaning
  member proof refs[]
  related candidate claims[]
  missing authorities by proof
  combined missing authority set
```

つまり:

```text
aggregate
  != flatten
```

である。

### 8.2 同じsubjectに見えても別dimensionなら統合しない

R1のsubmit周辺には:

```text
P3-07
  sales.order.submit -> sales.order.status write attribution
  missing = capability-to-database-write-attribution

P4-06b
  BP-06 protected target = order_entry.submit_order
  missing = business-protected-target-authority
```

がある。

どちらも`submit_order`周辺だが、同じ未解決ではない。

```text
mutation effect authority
  != Business security target authority
```

したがって`submit_order unresolved`一行へ潰さない。

### 8.3 同じdimensionなら不足authorityのunionだけでなくmember traceを残す

たとえば将来Handoffで:

```text
proof A
  from endpoint unresolved
  missing = accepted source for transfer origin

proof B
  condition unresolved
  missing = accepted continuation predicate
```

を`responsibility continuation`へまとめる場合、combined setだけでなくA/B対応も残す。

そうしないと、LLMが一つの追加資料で両方解決したと誤認しやすい。

## 9. R1 pressure

### Candidate R1-C1 — read-only master mutation

```text
claim
  customer_referenceからmaster customerを更新する

relevant proof
  P3-02 contradiction

packet result
  eliminated
```

P4やP5/P6は不要。

Elimination certificateにはP3-02とread-only datasource authorityだけを残せばよい。

### Candidate R1-C2 — submit status mutation

```text
claim
  sales.order.submit mutates sales.order.status

relevant proof
  P3-07 unresolved

known
  Database Contract write sales.order.status exists

missing
  capability-to-database-write-attribution

packet result
  frontier candidate + unresolved mutation-effect dimension
```

DB writeの存在を`effective=true`へ変換しない。

### Candidate R1-C3 — Authorization target

```text
claims
  BP-06 uses security.authorize
  protected target = order_entry.submit_order

relevant proof
  P4-06a supported
  P4-06b unresolved

packet result
  frontier candidate
  unresolved = Business protected target authority
```

Legacy policyによるcorroborationはEvidenceとして残せるが、Business targetをauthorしない。

### Candidate R1-C4 — manual action Data dependency

```text
claim
  BP-04 has no Data dependency

relevant proof
  P2-04 unresolved

packet result
  frontier candidate
  unresolved = accepted Data / Relation trace不足
```

no current traceをnegative factへ変換しない。

## 10. R2 pressure

R2はpositive second Handoff Referenceではないため、negative / boundary pressureとしてのみ使う。

### Candidate R2-M1 — Command acceptance = physical completion

```text
claim
  manufacturing command acceptedならphysical completion

P5
  contradiction

packet result
  eliminated
```

これはdomain factの未確定ではなく、導出規則自体が既存Command / Observation境界に反する。

### Candidate R2-M2 — observed physical fact + responsibility continuation

```text
claims
  finished goods physically present is observed
  therefore next responsibility may continue
```

current pressureでは:

```text
physical fact
  supported at explicit Observation scope

responsibility Handoff
  unresolved
  missing = accepted Business Handoff source / endpoints / condition
```

したがってpacketは:

```text
supported physical fact
+ unresolved responsibility continuation
```

を同時に持つ。

positive ObservationだけでcandidateをHandoff acceptedへ進めない。

## 11. local LLMへ渡す形

R4のpressureから、local LLMへ渡す情報は候補ごとに次へ近づけられる。

```text
candidate frontier item
  candidate summary
  exact accepted facts relevant to this candidate
  selected supported proof
  unresolved dimensions
    member proof refs
    missing authorities
  authority limits
```

必要に応じて:

```text
eliminated alternatives
  candidate summary
  decisive contradiction certificate
```

を少量付ける。

LLMへ全proof inventoryを渡す必要はない。

### LLMの仕事

- unresolved dimension間のtrade-off説明
- 追加で必要なBusiness判断 / sourceの特定
- minimum semantic delta候補
- contradictionを回避する別candidate variantの提案

### LLMの仕事ではないもの

- irrelevant contradictionの適用
- missing proofの自動補完
- missing authorityの捏造
- eliminated candidateを根拠なく復活
- unresolved aggregationから元proof traceを消す
- ProposalのAccept

## 12. stale / missing proof境界

Candidate-local packetを作る際、次はfail closedする。

```text
candidateが要求するproof questionがある
but current snapshot向けproof resultがない
  -> packet completeness unresolved

proof result exists
but evaluated snapshotが古い
  -> contractionへ使わない

proof ID resolves ambiguously
  -> contractionへ使わない
```

ここでも:

```text
missing proof
  != supported
```

を維持する。

## 13. eliminated candidateをどこまでLLMへ見せるか

全除外候補を毎回渡す必要はない。

候補比較の説明に必要な場合だけ、代表的な除外理由を添える。

優先順位:

```text
1. frontier candidate context
2. unresolved dimensions
3. user decisionに近いeliminated alternative
4. その他の除外候補はsummary / on-demand
```

これによりtoken節約だけでなく、LLMが無関係な失敗例へ引っ張られることを避ける。

ただし評価用途ではfull contraction traceを別artifactに保存してよい。

そのartifactもM1 authorityにはしない。

## 14. R4の判断

現時点では次が妥当である。

```text
proof selection
  -> candidate-local / exact scope
  -> same familyだから全件適用しない

irrelevant contradiction
  -> candidate eliminationへ使わない

required proof missing
  -> safe扱いせずunresolved

eliminated candidate
  -> full proof logではなくdecisive contradiction certificateを基本

unresolved aggregation
  -> semantic dimension単位
  -> member proof / per-proof missing authorityを保持

LLM packet
  -> frontier中心
  -> eliminated alternativesは必要最小限
```

これはproduction schema adoptionではなく、R3のbounded-context仮説を具体化したread-only評価である。

## 15. test-only pressureへ進める範囲

Gate C前でも、既存test-only proof report上で次を機械的にPressureする価値がある。

```text
T1
  candidateごとに指定されたproof scopeだけを選び、
  unrelated contradictionを除外理由にしない

T2
  contradiction candidateはdecisive proofを持つ
  compact elimination certificateへ変換できる

T3
  unresolved candidateはmissing authorityを保持する

T4
  二つのunresolvedを同dimensionへ集約しても
  member proofとper-proof missing authorityを失わない

T5
  missing proof refを黙って省略せずfail closedする

T6
  packet shapeへAccepted / Valid / Complete / Score等を入れない
```

これはproduction packet implementationではない。

`cross_view_proof_report_pressure_test.go`のtest-only resultを入力に使い、既存意味を再発明しないことを優先する。

## 16. R4後に残す問い

R3のread-only pressureのうちD/Eは別に残す。

```text
D. Reference pattern / associative retrieval candidateと
   exact candidateを同じcandidate欄へ置くとprovenanceを失わないか

E. local LLM outputをminimum semantic deltaへ制限したとき、
   whole-model generationよりreview性が改善するか
```

DはM2 / M3 provenance境界、EはGate C後のactual review taskに近い。

そのためA〜Cのtest-only pressureより先にproduction実装しない。

## 17. 非対象

- Product Gate C前のAI runtime
- production Context Packet schema
- persisted Candidate / Proof lifecycle
- generic proof registry
- graph DB / vector DB
- LLM ranking scoreによるcandidate elimination
- unresolvedのboolean化
- all-proof global validity score
- automatic semantic repair
- automatic Shared Acceptance
- Business Procedure Contract ADR blocker変更
- Product Gate B / Gate C Evidence条件変更
