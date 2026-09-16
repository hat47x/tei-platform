# Design Synthesis Context Packet Mixed Frontier Refinement — 2026-09

Status: **Lane D / R3 refinement / read-only evaluation / no production schema adoption**

## 1. この文書の役割

本書は、R3 `DESIGN_SYNTHESIS_CONTEXT_PACKET_PROOF_BOUNDARY_2026-09.md`のconceptual packet shapeを、後続R4 / R5と`DESIGN_SYNTHESIS_ORDER_ENTRY_S1_S2_FRONTIER_PRESSURE_2026-09.md`で得たEvidenceに合わせて補正する。

新しいR番号やproof familyを追加するものではない。

今回見つかったのは、新しいsemantic ruleではなく、**S2後に異なるepistemic statusを持つcandidateが同時に残るfrontierをS3へ渡すときの情報損失リスク**である。

本書は次を採用しない。

- production `DesignContextPacket` schema
- production Candidate lifecycle
- candidate origin enum
- ranking / winner score
- persisted qualification enum
- automatic Acceptance
- local LLM provider contract

## 2. 発見した具体的なgap

R3の初期conceptual shapeには次がある。

```text
candidate_designs[]
  candidate_id
  proposed_meaning
  proposal_origin
```

しかしR5ではすでに:

```text
candidate meaning / target
  != discovery mechanism
  != provenance authority
```

を確認している。

同じcandidateはexact lookup、Reference Pattern、associative retrieval、human proposal等の複数経路で見つかり得るため、単一の`proposal_origin`へ畳むと情報を失う。

さらにS1 / S2 frontier pressureでは:

```text
C-A
  existing order_entry.customer_reference
  current accepted supportあり

C-B
  order-owned customer snapshot proposal
  contradictionなし
  accepted support不足

C-C
  customer master mutation proposal
  accepted read-only authorityとcontradiction
```

となり、S2後に:

```text
C-A  retained / supported existing
C-B  retained / unresolved proposal
C-C  eliminated / contradiction
```

というmixed frontierが残った。

R3 packetが候補を単なる`candidate_id + proposed_meaning + origin`へ畳むと、C-AとC-Bの差をS3 / human reviewへ安全に渡せない。

## 3. R3 conceptual shapeの読み替え

R3の`proposal_origin`はproduction field候補として扱わない。

後続Evidenceを反映したconceptual decompositionは次である。

```text
Candidate Subject
  candidate_id
  semantic_claim / proposed_meaning

Current M1 Relation
  existing target identity, if any
  exact current authority refs, if any

Proposal-only Meaning
  accepted identityをまだ持たないsemantic delta, if any

Discovery Trace(s)
  how / why the candidate was surfaced
  provenance refs
  derived representation base revision, if relevant

Candidate-local Proof / Constraint Results
  supported claims
  contradiction claims
  unresolved claims
  missing authorities

Residual Dimensions
  human / S3が判断すべき未解決
```

これはschema定義ではない。

重要なのは、それぞれを**同じ一軸のcandidate stateへ潰さない**ことである。

## 4. candidate meaningとdiscovery traceを分ける

禁止する形:

```text
candidate.origin = exact | pattern | associative | llm
```

理由:

```text
same semantic candidate
  <- exact lookup
  <- graph neighborhood
  <- Reference Pattern
  <- associative retrieval
```

が同時に成立し得る。

したがってpacket側では:

```text
candidate subject
  1

discovery traces
  0..N
```

として理解する。

Discovery traceの数やscoreはcandidateのauthority strengthではない。

## 5. mixed frontierを平坦化しない

S2後に残ったcandidateは、同じ意味で「valid」なのではない。

```text
C-A
  exact existing identityあり
  read pathへのaccepted supportあり

C-B
  proposal-only
  contradictionはない
  accepted support不足
```

したがって:

```text
survived
  != supported

supported
  != preferred
  != winner
  != accepted
```

をpacket境界でも保持する。

### 単一qualification enumも急いで固定しない

今回のC-A / C-Bだけを見て:

```text
candidate.status = supported-existing | unresolved-proposal | contradicted
```

というproduction enumを作らない。

同じcandidate内で:

```text
P1 supported
P2 supported
P3 unresolved
```

のように複数proof resultが同時に成立し得るためである。

frontierの違いはcandidate-local proof / unresolved traceから再構成できることを優先する。

## 6. S3へ渡すminimum context

今回のmixed frontierから、S3 / human reviewへ少なくとも次を失わず渡す必要がある。

```text
purpose
current accepted snapshot

candidate C-A
  semantic claim
  existing target identity = order_entry.customer_reference
  current supporting proof refs
  unresolved dimensions, if any
  discovery traces

candidate C-B
  semantic claim = order-owned customer snapshot proposal
  existing target identity = none
  accepted support = none for snapshot requirement
  contradiction = none in current scope
  unresolved dimensions
    snapshot necessity
    copy timing
    freshness semantics
    ownership / audit requirement
  discovery traces

eliminated C-C
  contradicted claim
  decisive proof = P3-02
  accepted authority = master datasource access read
```

これによりS3は:

- C-Aがcurrent accepted assetへ戻れること
- C-Bが単なる検索漏れではなくproposal-onlyであること
- C-Bが`false`ではなくunresolvedであること
- C-Cを復活させるにはauthority changeまたは別semantic variantが必要なこと

を区別できる。

## 7. minimum semantic deltaはqualificationではない

TEIはS3でminimum semantic deltaを優先評価する方向だが:

```text
C-Aの方が変更量が小さい
```

ことは:

```text
C-Bがcontradiction
```

であることを意味しない。

したがってpacketへ:

```text
preferred = C-A
winner = C-A
score(C-A) > score(C-B)
```

のような値をS2 proofから生成しない。

minimum semantic deltaは:

- accepted Product / design objectiveが明示されている場合はそのconstraintとして評価する
- そうでなければS3 / human reviewで比較する

という位置づけにする。

## 8. eliminated candidateとの分離

C-Cのようにexact contradictionで落ちたcandidateはfrontierへ残さない。

一方、後段が理由を再現できるようR4のcompact elimination certificate相当を保持する。

```text
candidate = C-C
contradicted claim = master customer mutation
proof = P3-02
authority = order_entry master datasource access read
snapshot = current evaluation base
```

このcertificateはpast universal invalidityを意味しない。

snapshotやcandidate meaningが変われば再評価する。

## 9. test-only pressure

`tools/reference-business-procedure-check/design_synthesis_mixed_frontier_context_pressure_test.go`で、conceptually次をPressureする。

1. surviving frontierにexisting supported candidateとunresolved proposalを同時保持できる
2. proposal-only candidateへ架空のaccepted identityを付けない
3. same candidateへ複数discovery traceを保持できる
4. candidate subjectとdiscovery traceを分離する
5. eliminated contradictionをfrontierから分離しつつreasonを保持する
6. packet shapeに`Origin` / `Winner` / `Preferred` / `Score` / `Accepted`等のauthority・ranking短絡fieldを置かない
7. support数の多さだけでfrontierを1案化しない

actual `go test` greenは別Evidenceであり、本書やtest sourceの存在だけでA2へ進めない。

## 10. R3 / R4 / R5への反映

このrefinementにより、R3の初期conceptual shapeは次の順で読む。

```text
R3
  packetはsnapshot-bound review projection

R4
  proofはcandidate-localに選択し
  unresolved / elimination reasonをtrace-preservingに保持

R5
  candidate meaningとdiscovery trace / provenanceを分離

S1 / S2 frontier pressure
  supported existing + unresolved proposalが
  同じfrontierへ残るcaseを確認

本refinement
  これらをS3へ渡す際にflattenしない
```

したがってR3の`proposal_origin`という初期表現は、**R5以後のcurrent conceptual boundaryでは単一fieldとして採用しない**。

## 11. 現時点の結論

今回のconcrete information-loss gapは、production packet型を追加せずに閉じられる。

必要なのは:

```text
candidate subject
  != discovery trace
  != current qualification / proof

survived
  != supported

supported
  != winner / accepted

minimum semantic delta
  != S2 contradiction proof
```

をbounded contextでも保持することである。

このrefinementからproduction `DesignContextPacket` schemaやcandidate lifecycleを採用しない。

次にread-only workを増やすのは、この境界でもcurrent Referenceの必要情報が失われる具体例が見つかった場合に限る。
