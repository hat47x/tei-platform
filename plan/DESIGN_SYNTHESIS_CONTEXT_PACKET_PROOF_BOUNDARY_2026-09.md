# Design Synthesis Context Packet / Cross-View Proof Boundary — 2026-09

Status: **Lane D / R3 evaluation / design-only / no production schema adoption**

## 1. この文書の役割

本書は、`DESIGN_SYNTHESIS_CROSS_VIEW_PROOF_DESIGN.md`で定義したcross-view proofと、`DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md`で構想している`DesignContextPacket`の間を接続するR3評価である。

R1 / R2で得た` supported / contradiction / unresolved `を、将来のlocal LLMやhuman reviewへ安全に渡すには何を残し、何を渡してはいけないかを整理する。

本書は次を新設しない。

- Product Requirement
- Gate / PAS
- production `DesignContextPacket` schema
- persisted `ProofObligation` / proof result enum
- Canonical `Claim` / `SemanticGraph`
- graph DB
- AI runtime / provider contract
- automatic Acceptance

Product Gate Cは現在Blockedであり（2026-09-21注記: 起票時点の記述。Gate BはR = `e5231839`、Gate CはT = `0ba87a4e`で2026-09-19にReadyとなり、第一Web Targetも実装・Evidence取得済みである。判定は[Gate C Readiness Audit](GATE_C_READINESS_AUDIT.md)）、Definition-Centered Feedback Loopのproduction AI / Studio implementationはGate C Ready後に進める。したがってR3は**read-onlyな情報境界の評価**で止める。

## 2. 入力となる既存Evidence

### R1 — order-entry P1〜P4

`DESIGN_SYNTHESIS_ORDER_ENTRY_PROOF_REFERENCE_2026-09.md`では、既存`order_entry`から次を確認した。

```text
supported
  explicit Business -> Pattern / Capability ref
  exact Pattern -> Data / Relation trace
  BP-06 -> security.authorize

contradiction
  read-only master datasourceへmutationを要求する候補

unresolved
  no current Data trace
  sales.order.submit -> sales.order.status write attribution
  BP-06 protected targetのBusiness-side authority
```

重要なのは、Database writeやLegacy Authorization policyが存在しても、それぞれCapability mutationやBusiness protected targetを自動authorしなかったことである。

### R2 — manufacturing / inventory P5〜P6

`DESIGN_SYNTHESIS_MANUFACTURING_ACTUALITY_HANDOFF_PROOF_REFERENCE_2026-09.md`では、second-corpus pressureから次を確認した。

```text
contradiction
  Command accepted -> physical completion
  Step adjacency / actor change / Function sequence / Observation -> Handoff

supported
  explicit external Observationが実際に示す範囲のphysical fact

unresolved
  physical fact -> Business acceptance / continuation
  second realistic Handoff existence
  Handoff from / to / condition
```

ここでも、positive ObservationをHandoffへ逆流させなかった。

### test-only proof report

`tools/reference-business-procedure-check/cross_view_proof_report_pressure_test.go`には、R1の代表6問を次のshapeで再構成するtest-only probeがある。

```text
crossViewProofResultProbe
  ID
  Family
  Question
  Result
  EvidencePaths[]
  MissingAuthorities[]
  Reason
```

このprobeは`Canonical`、`Authoritative`、`Accepted`、`Complete`、`Confidence`、`Score`、`WriteBack`、`SourceOfTruth`を持たない。

actual Repository runner greenは未取得であり、本書もこのprobeをproduction contractとして扱わない。

## 3. R3の中心問い

R3では次を問う。

> cross-view proofをLLM / human reviewのbounded contextへ渡すとき、候補縮約に必要な情報を保持しつつ、derived resultをauthorityへ昇格させない最小境界は何か。

この問いを三つへ分ける。

```text
Q1
  candidateを安全に除外するために何が必要か

Q2
  unresolvedを失わず残余判断へ渡すために何が必要か

Q3
  packet / LLM outputがaccepted meaningへ逆流しないために何を禁止するか
```

## 4. DesignContextPacketはtransport / review projectionである

現時点の最も安全な位置づけは次である。

```text
M1 accepted authoritative sources
        ↓ exact resolve
Accepted Semantic Snapshot
        ↓ derive
cross-view proof / constraints / retrieval
        ↓
DesignContextPacket
        ↓
local LLM / human review
        ↓
Proposal / minimum semantic delta
        ↓
Preview
        ↓
Shared Acceptance Boundary
        ↓ Accept only
M1 owner source
```

`DesignContextPacket`はM1 authoritative memoryではない。

また、packet自身を新しいM2 knowledge baseとして永続化する必要もない。入力revisionから再構築できる範囲はdisposable projectionとして扱う。

```text
packet deleted
  -> accepted meaning must remain unchanged

packet rebuilt with same accepted snapshot
  -> semantic inputs should be equivalent

LLM provider changed
  -> accepted meaning must not change until Acceptance
```

## 5. 概念上必要なpacket内容

production JSON schemaではなく、R3で情報損失を確認するための概念形を次とする。

```text
DesignContextPacket
  purpose
  target_scope

  accepted_snapshot
    authority_refs[]
    revision_or_fingerprint_refs[]

  exact_facts[]
    fact
    authority_ref

  candidate_designs[]
    candidate_id
    proposed_meaning
    proposal_origin

  proof_results[]
    proof_id
    family
    question
    result
      supported | contradiction | unresolved
    evidence_refs[]
    missing_authorities[]
    reason

  eliminated_candidates[]
    candidate_id
    proof_refs[]
    reason

  unresolved_dimensions[]
    dimension
    related_candidate_ids[]
    missing_authorities[]

  authority_limits[]
  requested_output_contract
```

すべてを永続fieldにすることは要求しない。

特に`accepted_snapshot`は一つのglobal revision番号を新設する要求ではない。既存source / fingerprint / revisionの組み合わせから、評価対象となったauthority集合を再現できればよい。

## 6. packetへ入れるもの / 入れないもの

### 入れる

1. current taskに必要なexact accepted fact
2. exact identityへ戻れるauthority ref
3. candidateごとのproposal meaning
4. candidateに対するbounded proof result
5. contradictionの根拠
6. unresolvedの不足authority
7. 既に除外したcandidateと除外理由
8. 人またはLLMが判断すべき残余dimension
9. packetのauthority限界

### 原則として入れない

1. Repository全体の無制限dump
2. hidden chain-of-thought相当の探索履歴
3. similarity scoreだけで作ったsemantic fact
4. candidate rankingをaccepted priorityと誤読させる値
5. derived graph edgeをauthorityとして扱う表現
6. past acceptanceをcurrent applicabilityへ変換したfact
7. stale revision上のproofをcurrent factとして扱う値
8. automatic write-back instruction

## 7. candidate contractionの規則

### 7.1 contradictionは条件付きでcandidateを除外できる

`contradiction`なら常にcandidate全体を捨てる、とはしない。

除外できるのは、少なくとも次が揃う場合である。

```text
proof question scope
  candidate claimと一致

accepted authority
  exactに解決可能

snapshot
  current evaluation baseに対応

conflict
  candidate claimとauthorityが両立しない
```

例:

```text
candidate
  customer_referenceからmaster datasourceを更新する

accepted authority
  order_entry.master access = read

result
  contradiction

contraction
  このmutation claimを含むcandidateは除外可能
```

ここではLLM判断を必要としない。

### 7.2 unresolvedは除外理由ではない

```text
unresolved != incompatible
unresolved != false
```

例:

```text
candidate claim
  sales.order.submit mutates sales.order.status

known
  DB Contractにはwrite sales.order.statusがある

missing
  Capability / Pattern -> DB write attribution

result
  unresolved
```

このcandidateを「正しい」とも「誤り」とも扱わない。

残余dimensionとして人または後段LLMへ渡す。

### 7.3 supportedはcandidate全体のAcceptanceではない

```text
P1 supported
  BP-00 -> recent_orders
```

から:

```text
candidate whole design is correct
```

とは言えない。

`supported`は一つのbounded claimがaccepted authorityにより支持されたことだけを意味する。

### 7.4 proof scopeを失わない

同じcandidate内で:

```text
P1 supported
P2 supported
P3 unresolved
P4 contradiction
```

が同時に成立し得る。

したがってcandidateへ単一`valid=true/false`や一つのAI scoreだけを付ける設計は避ける。

## 8. R1をpacketへ写した場合

概念例:

```text
purpose
  order_entryのBusiness / Function / Data整合を確認する

candidate C1
  customer_referenceからmaster customerを更新する

proof P3-02
  result = contradiction
  evidence = application order_entry / datasource master / access read

candidate C2
  sales.order.submitがsales.order.statusを更新する

proof P3-07
  result = unresolved
  evidence = sales_core.order / write sales.order.status
  missing = capability-to-database-write-attribution

candidate C3
  BP-06 protected target = order_entry.submit_order

proof P4-06b
  result = unresolved
  evidence = legacy authorization execute/allow submit_order
  missing = business-protected-target-authority
```

この場合:

```text
C1
  deterministic contractionで除外可能

C2 / C3
  unresolved frontierへ残す
```

となる。

ここでLegacy policyやDatabase writeから不足authorityを自動生成してはいけない。

## 9. R2をpacketへ写した場合

概念例:

```text
candidate M1
  manufacturing command acceptedなら製造完了と扱う

P5
  contradiction

candidate M2
  positive physical Observationがあれば次担当へHandoffする

P6
  contradiction

remaining question
  physical completionが観測されたあと、
  Business上どの責任がどの条件で次へ継続するか

result
  unresolved
  missing = accepted Business Handoff source
```

ここで重要なのは、

```text
physical fact = supported
Handoff = unresolved
```

を同時に保持できることである。

一つのcompletion flagへ畳むと、この境界を失う。

## 10. local LLMへ渡す残余

LLMへ渡すのは、exact factやhard contradictionを再推論させるためではない。

望ましい残余は次である。

```text
- 複数accepted interpretationが残る
- Business intentが不足している
- authorityそのものが未authoring
- trade-offを比較する必要がある
- minimum semantic delta候補を作る必要がある
```

### LLMに要求してよいこと

- unresolvedの説明
- A/B候補の比較
- 必要な追加質問の提案
- unsupported assumptionの列挙
- minimum semantic delta proposal
- counterexampleの提案

### LLMに要求しないこと

- 存在しないCanonical IDの補完
- missing authorityの捏造
- `unresolved`を確率的に`true/false`へ丸めること
- policyからBusiness targetをauthorすること
- DB writeからCapability mutationをauthorすること
- ObservationからHandoffをauthorすること
- Proposalのautomatic Accept

## 11. LLM出力の概念境界

LLM outputもaccepted meaningではない。

概念的には最低限:

```text
DesignProposal
  base_snapshot_ref
  addressed_unresolved_dimensions[]
  proposed_semantic_delta[]
  assumptions[]
  supporting_refs[]
  remaining_unresolved[]
```

程度へ限定する方向が安全である。

ただし、これもproduction schemaとして採用しない。

重要なのは:

```text
LLM output
  -> Proposal
  -> Preview / qualification
  -> Shared Acceptance
  -> accepted source update
```

というauthority方向である。

## 12. stale境界

proof resultもpacketもsnapshot-boundである。

次の変更があれば再評価する。

```text
Business source ref change
Application Pattern change
Semantic Capability change
Database Contract change
Authorization policy change
relevant source add/remove
accepted Handoff source change
```

一方、Business wordingだけが変化し、exact refsや対象meaningが変わらない場合、Data projection等が必ず変化するとは限らない。

したがってpacket freshnessを一つのGit HEADだけへ短絡しない。どのsource集合を基にproofしたか追跡できることを優先する。

## 13. M1 / M2 / M3との関係

Definition-Centered Feedback Loopのmemory separationへ対応させると:

```text
M1
  accepted source / accepted authority
  packetのexact_facts / authority refsの根

M2
  derived graph / cross-view proof / retrieval / ranking
  packetへderived resultとして入る

M3
  Reference evidence / proposal / prior review / experience
  provenance付きcandidate inputになり得る
```

禁止する短絡:

```text
M2 proof result
  -> M1 accepted meaning

M3 past successful proposal
  -> current accepted meaning
```

packetは三memoryを融合した第四のauthorityではない。

## 14. packetの永続化判断

現時点ではproduction persistenceを要求しない。

まずは次の順でよい。

```text
1. accepted sourcesからon-demand生成
2. review / LLM呼び出し中だけ保持
3. Evaluation Evidenceとして必要な場合だけartifact保存
4. revision stalenessを検出して再生成
```

将来、長時間review、複数人review、再現可能なAI評価等でpacket identityが必要になった場合でも、保存理由はBusiness authorityではなくEvaluation / workflow Evidenceである。

保存したpacketを次回のauthority sourceとして読み戻さない。

## 15. Context Packetを小さく保つ基準

bounded contextの価値は、単なるtoken節約ではない。

次をpacket生成側で終えておく。

```text
exact identity resolution
hard contradiction detection
irrelevant branch elimination
duplicate candidate collapse
current scope dependency closure
known authority limits
```

LLMへRepository全体を渡し、同じresolutionを自然言語推論でやり直させない。

一方、局所化しすぎて重要dependencyを落とすことも避ける。

したがって将来のReferenceでは:

```text
whole-repo context
vs
selected semantic neighborhood + proof packet
```

を比較し、見落とし率・unsupported inference・latency・reviewabilityを評価する。

## 16. R3の判断

現時点の結論:

```text
cross-view proof
  -> DesignContextPacketへ渡す価値あり

packet role
  -> disposable / snapshot-bound review projection

contradiction
  -> scopeとauthorityがexactならdeterministic candidate contractionに利用可能

unresolved
  -> candidate除外せずresidual dimensionへ保持

supported
  -> bounded claim supportでありwhole candidate acceptanceではない

LLM
  -> unresolved比較 / explanation / minimum delta proposalへ限定

Shared Acceptance
  -> authorityへ戻す唯一の境界
```

このR3からproduction `DesignContextPacket`型やpersisted proof modelを採用しない。

## 17. 次に実装する条件

現在はProduct Gate CがBlocked、Lane C actual Repository executionも未取得である（2026-09-21注記: 起票時点の記述。Product Gate Cは2026-09-19にReady、Lane C Gate A actual executionは2026-09-20にPASSした。Contract ADRは引き続きNot Ready）。

そのため次のproduction実装へは進まない。

実装をreopenする条件:

1. actual runnerで既存cross-view pressure / proof probeがgreenになる、または同等のactual full checkout Evidenceを得る。
2. Product Gate C Ready後、Definition-Centered Phase 1のone-change closed loopでbounded contextが実際に必要になる。
3. realistic Referenceで、既存local contextだけでは候補比較に情報損失が起きる。

それ以前に行えるのはread-only evaluationだけである。

## 18. 次のread-only pressure

production実装前に、必要なら次を評価する。

```text
A. packetへ全proofを入れる必要があるか
   -> current candidateに関係するproofだけで十分か

B. eliminated candidateをpacket本文へ残す必要があるか
   -> reasonだけで再現可能か

C. unresolved dimensionをproof単位から集約すると
   authority不足を失わないか

D. Reference pattern / associative retrieval candidateを
   exact candidateと同じ欄へ混ぜるとprovenanceを失わないか

E. local LLM出力をminimum deltaへ制約した場合、
   whole-model生成よりreview性が改善するか
```

これらもSchema採用ではなくReference evaluationとして行う。

## 19. 非対象

- Product Gate C前のAI runtime実装
- packet service / packet database
- generic proof engine production化
- graph DB採用
- vector DB採用
- automatic Candidate Acceptance
- LLMによるBusiness authority補完
- cross-view proofをConformance全体の単一scoreへ変換
- Business Procedure Contract ADR blockerの変更
- Gate B / Gate C Ready条件の変更
