# Design Synthesis Candidate Discovery / Provenance Boundary — 2026-09

Status: **Lane D / R5 read-only evaluation / no production schema adoption**

## 1. この文書の役割

本書は、R3 / R4で残したread-only pressureのうち、次を深める。

```text
Reference pattern / associative retrieval由来のcandidateを
exact existing asset由来のcandidateと同じfrontierへ置いたとき、
provenance / authority / qualificationの差を失わないか。
```

入力となる既存基線:

- `REFERENCE_PATTERN_EVIDENCE_HANDOFF_2026-09.md`
- `ISSUE-000047 knowledge-first semantic exploration`
- `ISSUE-000051 derived cognitive retrieval evaluation`
- `DESIGN_SYNTHESIS_CONTEXT_PACKET_PROOF_BOUNDARY_2026-09.md`
- `DESIGN_SYNTHESIS_CANDIDATE_LOCAL_PROOF_PACKET_PRESSURE_2026-09.md`

本書はProduct Gate C前のread-only evaluationであり、次を採用しない。

- production Candidate schema
- production `DesignContextPacket` schema
- Pattern Memory database
- vector / graph database
- derived cognition algorithm
- AI provider / runtime
- automatic Acceptance

## 2. 中心結論 — candidateの意味と「見つかった経路」は別軸

最も重要な判断は次である。

```text
candidate meaning / target
  != discovery mechanism
  != provenance authority
```

たとえば既存の`customer.customer`というcurrent M1 assetは:

```text
exact Canonical lookup
lexical search
graph neighborhood
Reference patternからの連想
sparse / HDC / embedding retrieval
past episodeからのhint
```

のどの経路でも発見され得る。

したがって:

```text
candidate.origin = exact | pattern | associative
```

という単一の排他的分類をproduction contractへ置くのは不適切である。

同じcandidateが複数のdiscovery traceを持てるからである。

より安全な概念分離は:

```text
Candidate
  what is being considered

Discovery Trace(s)
  why / how it was surfaced

Current Qualification
  whether it is applicable in current M1 context

Proof / Constraint Results
  what current authority says about bounded claims
```

である。

## 3. 四つの情報を混ぜない

### 3.1 Candidate subject / proposal

検討対象そのもの。

例:

```text
existing asset reuse
  customer.customerを参照する

new semantic delta proposal
  Order - OrderLine relationを追加する

structural alternative
  flat rowsではなくheader-detail compositionを採る
```

### 3.2 Discovery trace

候補がfrontierへ現れた経路。

概念例:

```text
exact-lookup
exact-graph-neighborhood
reference-pattern
associative-retrieval
episodic-hint
human-proposed
llm-proposed
```

これはcandidateの正しさではない。

### 3.3 Provenance

そのdiscovery traceが何に由来するか。

```text
exact-lookup
  -> current M1 identity / source / revision

reference-pattern
  -> Pattern Evidence Handoff
     nature / corpus / research artifact / limitation

associative-retrieval
  -> derived representation revision
     feature source revision
     retrieval mechanism / task field
     retrieved underlying subject / pattern provenance

episodic-hint
  -> past proposal / decision / run / outcome context
```

### 3.4 Qualification / proof

current intentに使えるかをcurrent M1へ戻って判定する層。

```text
retrieval rank
  -X-> applicable

pattern frequency
  -X-> applicable

past acceptance
  -X-> applicable

current M1 qualification / cross-view proof
  -> applicable / incompatible / unknown
     supported / contradiction / unresolved
```

## 4. exact existing asset candidate

既存M1 assetを候補にする場合、candidate自身はcurrent identityへexactに戻れる。

```text
candidate
  target = current Canonical / logical identity

provenance
  authoritative source / source ref / current revision
```

ただし:

```text
exists
  != applicable to current intent
```

である。

ISSUE-000047が要求する:

- identity
- dependency
- scope
- constraint
- Source / Evidence
- current revision

を確認して初めて`applicable / incompatible / unknown`へ進める。

### exact discoveryでもAcceptanceではない

exact lookupで見つかったことは候補生成の不確実性を減らすが:

```text
exact existing identity
  != current requirementへの採用
```

である。

## 5. Reference Pattern candidate

Reference PatternはM3 provenance evidenceからcurateされたcandidate knowledgeである。

```text
Reference observation
  -> M3 Pattern Evidence
  -> M2 retrieval representation
  -> candidate
```

Pattern candidateをfrontierへ置く場合、最低限失ってはいけないのは:

```text
pattern key
nature = observed | candidate
research / corpus provenance
structural signature
applicability questions
contraindications / negative evidence
known non-proof
```

である。

### frequencyをcandidate strengthへ短絡しない

```text
10 / 12 source models support
  != high confidence
  != current applicability
  != preferred candidate
```

R5では`model_count`やfrequencyをcross-view proofの`evidence`欄へ混ぜない。

Patternのsupport breadthはPattern provenanceであり、current modelのaccepted authorityではない。

### Pattern keyをcurrent identityへしない

```text
header_detail_transaction
  != Canonical business identity
  != current Application Pattern identity
```

Pattern keyからcurrent sourceへ自動authorしない。

## 6. Associative retrieval candidate

ISSUE-000051で比較予定のsparse / HDC / embedding / weighted-set等はM2 derived mechanismである。

R5での原則:

```text
associative retrieval
  returns attention / candidate
  not semantic fact
```

### retrieval traceが必要

候補を後で説明するには、conceptually少なくとも:

```text
discovery mechanism
retrieval task / field
representation base revision
retrieved subject identity or pattern key
ranking / activation signal if evaluation requires it
```

へ戻れることが望ましい。

ただしscoreをpacketのauthority値へしない。

### underlying provenanceを失わない

associative retrievalが返す対象は少なくとも二種類あり得る。

```text
A. current existing M1 asset
B. Reference Pattern / M3 evidence-derived candidate
```

Aなら:

```text
associative retrieval trace
  + current M1 asset provenance
  -> deterministic qualification
```

Bなら:

```text
associative retrieval trace
  + Pattern Evidence provenance
  -> applicability questions
  -> current M1 qualification
```

となる。

retrieval representation自身をunderlying provenanceの代替にしない。

## 7. 同じcandidateが複数経路で発見された場合

例:

```text
candidate X = existing customer Data reuse

Discovery trace 1
  exact relation neighborhood

Discovery trace 2
  lexical search

Discovery trace 3
  associative structural retrieval
```

candidateを三つへ複製しない。

意味的に同じcurrent identity / proposalであれば、一つのcandidateへ複数discovery traceを関連づけられる。

ただし:

```text
multiple discovery traces
  != stronger semantic authority
```

である。

三経路で見つかったから`confidence=3`のようなauthority化をしない。

### dedup keyはcandidate meaning側

既存assetならcurrent exact identityを使える。

新proposalの場合は、将来のReferenceでproposal meaningをlosslessに比較できる範囲を決める。

retrieval mechanism名をdedup identityにはしない。

## 8. Pattern-inspired new proposal

Reference Patternから得た構造がcurrent M1にそのまま存在しない場合、新しいproposalを作り得る。

例:

```text
current intent
  repeated order linesを持つ取引

Reference Pattern
  header_detail_transaction

proposal
  Order -> OrderLine one-to-many relation
```

このとき:

```text
Pattern Evidence
  = why this structure was considered

current Business intent / exact facts
  = why it may be applicable

new relation proposal
  = M3 Proposal
```

である。

Patternから直接M1 relationを生成しない。

### exact refsを捏造しない

Pattern-inspired proposalに必要なcurrent Data identityがまだ存在しない場合:

```text
new Data / Relation proposal
```

として扱う。

存在しないCanonical IDを「Reference Patternが示したから」という理由でaccepted refとして書かない。

## 9. Cross-view proofとの接続

proofはdiscovery mechanismに対してではなく、candidateのbounded semantic claimへ適用する。

```text
candidate surfaced by embedding
  -> proof question about candidate claim

candidate surfaced by exact lookup
  -> same proof question if claim is same
```

したがって:

```text
retrieval mechanism
  != proof family selection key
```

である。

R4のcandidate-local proof selectionはcandidate meaning / exact scopeを基準にし、`origin=associative`だから別proofを使う、としない。

### Pattern provenanceはproof evidenceではない

例:

```text
Pattern P-01 header_detail_transaction
observed in 8 / 12 historical models
```

は:

```text
current Order -> OrderLine cardinality is correct
```

のproof evidenceではない。

current Source / Semantic authorityから独立に確認する。

## 10. Context Packetへ渡すときの概念形

R5での概念形:

```text
Frontier Candidate
  candidate_id
  candidate meaning / proposal

  discovery_traces[]
    mechanism
    retrieval/provenance refs
    evaluation-only signal?

  current_qualification
    applicable | incompatible | unknown
    authority refs

  proof_results[]
    selected by candidate scope

  unresolved_dimensions[]
```

これはproduction schemaではない。

重要なのは`discovery_traces[]`と`current_qualification`を別にすることである。

### LLMへ何を渡すか

LLMに必要なのは通常:

- candidate meaning
- current accepted facts
- qualification result / reason
- selected proof result
- unresolved dimensions
- Reference Patternを使った場合のapplicability question / contraindication
- provenanceの短いsummary

全retrieval vectorや高次元representationを渡す必要はない。

## 11. score / rankの扱い

retrieval evaluationではscoreが必要な場合がある。

しかしpacket内では:

```text
retrieval_score
  = discovery/evaluation signal
  != confidence
  != evidence strength
  != acceptance probability
```

を固定する。

候補順位をLLMへ渡す場合も、順序がauthorityに見えない表示が必要である。

### Pareto frontierとの関係

Design Synthesis Loopのfrontierは単一retrieval score順ではない。

- exact constraint satisfaction
- cross-view consistency
- implementation feasibility
- unresolved count
- provenance quality
- business coverage

等を別軸として扱う。

retrieval scoreはcandidate generationの一信号に留める。

## 12. Episodic / experience hintとの境界

過去のProposalやaccept/reject履歴から候補を思い出す場合も同じである。

```text
past accepted
  != current applicable

past rejected
  != universally invalid
```

current snapshotで再qualificationする。

R4のelimination certificateも将来M3 Evaluation Evidenceになり得るが、次snapshotへ自動ruleとして持ち越さない。

## 13. stale境界

Candidate discovery provenanceには異なるfreshnessがある。

### current M1 asset

current source / revisionに対してexact resolutionが必要。

### Pattern Evidence

research artifact / corpus / abstraction versionにboundされる。

Pattern Evidenceが古くてもcurrent M1 authorityが変わるわけではないが、candidate generationの研究根拠としてstale / supersededを区別できる方がよい。

### Derived retrieval representation

base M1 / M3 input revisionが変わればrebuild / stale判定する。

```text
index stale
  != M1 stale
```

である。

## 14. R1へ当てる例

### existing exact asset candidate

```text
candidate
  customer.customerをreuse

discovery
  exact Relation sales.order.customer target

qualification
  current M1 identity / datasource / contractを照合
```

もし同じassetがassociative retrievalでも上位に出た場合、candidateは重複させずdiscovery traceだけ増やす。

### unsupported mutation proposal

```text
candidate
  customer_referenceからmaster customerをupdate

discovery
  Reference / associative / humanのどこから出てもよい

P3-02
  contradiction
```

candidate discovery経路に関係なく除外できる。

## 15. Reference Patternへ当てる例

`header_detail_transaction`をcurrent order-entryへ使う場合:

```text
Pattern provenance
  observed / 8 of 12 historical corpus

current exact facts
  sales.order
  sales.order.lines
  sales.order_line

current Pattern
  order_entry.order_lines
```

Pattern provenanceは「この構造を見る価値」を説明できる。

しかしcurrent accepted Relation / Patternのauthorityはcurrent TEI source側にある。

この二つを一つの`evidence strength`へ畳まない。

## 16. R5の判断

現時点の結論:

```text
candidate meaning
  and
candidate discovery provenance
  = orthogonal concerns

single exclusive origin enum
  = avoid

same candidate via multiple retrieval paths
  -> one semantic candidate + multiple discovery traces

exact existing asset
  -> current M1 identityへ戻る

Reference Pattern
  -> M3 provenance / applicability questionを保持

associative retrieval
  -> M2 discovery traceとして保持
  -> underlying M1 / M3 provenanceへ戻る

qualification / proof
  -> discovery mechanismではなくcandidate semantic scopeへ適用

retrieval score / pattern frequency
  -> candidate-generation signal only
  -> authority / confidence / acceptanceへ昇格しない
```

## 17. test-only pressureへ進める範囲

Gate C前でも、production retrieval engineを作らず次だけはsynthetic/test-onlyでPressureできる。

```text
T1
  同一existing candidateがexact + associativeの二経路で見つかっても
  candidateを重複させず複数discovery traceを保持できる

T2
  associative retrievalでexisting assetが見つかっても
  M1 exact qualificationを省略しない

T3
  Reference Pattern candidateはpattern provenance / natureを保持し
  frequencyをcurrent proof evidenceへ変換しない

T4
  same candidateにmultiple discovery traceがあっても
  semantic authority strengthを加算しない

T5
  stale derived retrieval representationをcurrent qualificationへ使わない

T6
  discovery traceを削除してもcandidateのaccepted M1 meaningは変わらない
```

既存algorithm比較やreal retrieval indexを必要としない範囲に限定する。

## 18. R5後に残る論点

R3で残したE:

```text
local LLM outputをminimum semantic deltaへ制約した場合、
whole-model generationよりreview性が改善するか
```

はactual review taskとの比較が必要であり、Product Gate C後のDefinition-Centered Phase 1に近い。

Gate C前に架空のLLM benchmarkを作って完了扱いしない。

一方、R5 T1〜T6はauthority boundaryのtest-only pressureとして先行可能である。

## 19. 非対象

- Product Gate C前のretrieval engine / AI runtime
- candidate origin production enum
- Pattern Memory production schema
- vector DB / graph DB
- FlyHash / HDC / embedding採用
- retrieval scoreのconfidence化
- Pattern frequencyのEvidence strength化
- Pattern keyのCanonical identity化
- discovery traceのM1 authority化
- automatic candidate acceptance
- Business Procedure Contract ADR blocker変更
- Product Gate B / Gate C Evidence条件変更
