# Design Synthesis Cross-View Proof Design

状態: **評価待ち（未採用）**

## 1. この文書の役割

この文書は、TEIのDesign SynthesisでData、Function / Interaction、Business Procedure、Security、Runtime / Observation等の異なる視点を使って設計候補を縮約するとき、**何を根拠に「支持される」「矛盾する」「まだ判断できない」と扱うか**を整理する機能設計案です。

この文書自体はProduct Requirement、ADR、Gate、Canonical schemaの正本ではありません。

- Product Requirementの正本: [`../../requirements/TEI_PRODUCT_REQUIREMENTS.md`](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)
- Business設計境界: [`../../docs/business-system-design.md`](../../docs/business-system-design.md)
- RQ-025 / RQ-026解釈: [`../../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](../../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)
- Business Procedure品質モデル: [`../BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)
- Design Synthesis上位目標: [`../DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md`](../DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md)

主にRQ-026の「異なる視点から相互検証できること」と、未採用DSR-02 / DSR-12の候補縮約を接続するための設計仮説です。

## 2. 背景

TEIでは、Data、Function / Interaction、Business Procedureを、同じ業務システムに対する異なるreview viewとして扱います。

```text
                Business
               /        \
              /          \
           Data -------- Function / Interaction
```

重要なのは、三種類の設計書を揃えることではありません。

たとえば「出荷を確定する」というBusiness Actionがある場合、次の問いは互いに別です。

- そのBusiness ActionはApplicationで支援されるのか
- 支援されるなら、どのPattern / Capabilityが直接realizationなのか
- そのinteractionは何を読み、何を変更するのか
- その変更はaccepted operation authorityを持つのか
- 実行者にAuthorizationがあるのか
- system上の成功と、物理的な出荷完了は同じ意味なのか
- 次工程への責任移転は何を条件に成立するのか

一つのviewだけからこれらすべてを推論すると、もっともらしいが根拠のない設計が完成しやすくなります。

Design Synthesisでは、この問題がさらに大きくなります。AIや探索器が複数候補を出した後に、単なるsimilarity scoreやLLM rankingで候補を絞ると、accepted authorityと推測の境界が失われます。

そこで、cross-view照合を単なる「整合性チェック」ではなく、**小さな証明義務（proof obligation）の集合**として扱う案を検討します。

## 3. 防ぎたい失敗

### 3.1 空欄を否定事実へ変える

```text
Data traceが見つからない
  -> Data dependencyは存在しない
```

とは限りません。trace不足はunresolvedであり得ます。

### 3.2 実装可能性をBusiness意味へ逆流させる

```text
DBへUPDATEできる
  -> Business上Updateしてよい
```

とは限りません。

### 3.3 時系列や隣接をsemantic relationへ変える

```text
Step BがStep Aの直後
  -> AからBへHandoffした
```

とは限りません。

### 3.4 Business actorをSecurity authorityへ変える

```text
actor = 倉庫担当者
  -> Authorization Role = warehouse
```

を自動生成してはいけません。

### 3.5 runtime successを外部世界の事実へ変える

```text
Command accepted
  -> physical shipment completed
```

ではありません。

### 3.6 derived graphを第二の正本にする

探索やcross-view照合のために構築したgraphのedgeが、accepted sourceに戻れないままauthorityになることを避けます。

## 4. 中心原則

### 4.1 証明する対象は「問い」であり、巨大な完全modelではない

TEIが一度に「この業務システムは完全に正しい」と証明する必要はありません。

代わりに、個別の問いを明示します。

```text
Q: Business Action BA-12はApplicationで直接realizeされるか
Q: Pattern AP-04はOrderLineへwrite authorityを持つか
Q: Handoff H-02はObservation O-07だけから成立すると言えるか
```

各問いについて、現在のaccepted authorityからどこまで言えるかを評価します。

### 4.2 proof resultは少なくとも三状態を失わない

```text
supported
  必要なaccepted authorityが解決でき、問いを支持する

contradiction
  accepted authority同士が、対象の問いについて両立しない

unresolved
  identity、authority、actuality、Evidence等が不足し、肯定も否定もできない
```

これは現時点で永続enumを要求しません。まずReference / report上の意味分類です。

### 4.3 「証明できない」と「偽」を分ける

```text
not proven != false
```

特にabsenceをnegative factへ変換しません。

### 4.4 derived resultはauthorityへ逆流しない

```text
accepted sources
    -> resolve / derive
    -> proof result / graph / matrix

proof result / graph / matrix
    -X-> accepted sourcesへの暗黙backfill
```

矛盾はauthoritative sourceを**再検討する理由**にはなりますが、修正はShared Acceptance Boundaryを通してsource側で行います。

## 5. Derived Meaning Graph

cross-view照合の実装では、異なるsourceのidentityとrelationを横断するgraphが有用です。ただし、このgraphを新しいsource of truthにはしません。

```text
authoritative file-native sources
        │
        │ parse / normalize / exact resolve
        ▼
Accepted Semantic Snapshot
        │
        │ derive
        ▼
Derived Meaning Graph
        ├─ cross-view proof
        ├─ impact analysis
        ├─ matrix / diagram
        ├─ candidate contraction
        └─ DesignContextPacket
```

要求する性質は次です。

- graphを削除して再構築してもaccepted meaningを失わない
- algorithmやstorageを変更してもaccepted meaningは変わらない
- edgeは可能な限り、その導出元のaccepted identityへ戻れる
- heuristic / similarity edgeとexact semantic edgeを混同しない
- graphにedgeが無いことをrelation不存在とみなさない

したがって、現段階ではgeneric `SemanticGraph` Canonical kind、graph DB、RDF、knowledge graph製品等をProduct contractにしません。

## 6. Proof Obligationの概念形

実装schemaではなく、reviewで必要になる概念を次のように整理します。

```text
Proof Obligation
  question
  subject / scope
  accepted authorities required
  derivation path
  result
    supported | contradiction | unresolved
  supporting / conflicting refs
  unresolved reason
  evaluated snapshot
```

### question

何を確認したいかです。曖昧な「整合しているか」ではなく、Yes / Noまたは限定された意味分類へ落とせる問いにします。

### subject / scope

どのBusiness Action、Pattern、Data、Handoff、Policy等についての問いかを示します。

### accepted authorities required

結論を出すのに必要なauthorityです。必要なauthorityが未解決なら、都合のよい代替sourceで穴埋めしません。

### derivation path

どのexact refやaccepted relationを辿ったかです。単なる説明文ではなく、再評価可能であることを重視します。

### result

`supported / contradiction / unresolved`を意味上区別します。

### evaluated snapshot

どのaccepted source集合に対して評価したかを再現できるようにします。ここは後述するAcceptance Snapshotの考え方と接続します。

これらをすべて永続fieldとして採用する必要はありません。Referenceでlossが起きるものだけをdurable contract候補にします。

## 7. 最初に持つべきproof family

### P1 — Business Action / Realization

**問い:** このBusiness ActionはApplicationでどのように支援されるか。

許される根拠:

- explicit Application Pattern ref
- explicit Capability ref
- accepted realization relation

禁止する短絡:

- Action名とFunction名の類似
- actorから利用画面を推測
- 前後StepのFunctionを流用

Function realizationが無い場合もBusiness Action自体をinvalidにしません。

### P2 — Business / Data Trace

**問い:** このBusiness Actionについて、accepted refからData / Relationへのtraceを導出できるか。

結果例:

```text
supported:
  exact Pattern -> Query -> Data traceが得られる

unresolved:
  Actionは存在するがdirect realization / Data traceが無い
```

重要なのは、

```text
no current Data trace != no Data dependency
```

を維持することです。

### P3 — Function / Mutation Authority

**問い:** このinteractionは対象Dataを変更してよい意味を持つか。

根拠候補:

- accepted operation contract
- Capability / Actionのmutation semantics
- datasource policy

禁止する短絡:

- physical DBがwrite可能
- Data traceが存在する
- Patternがrecord系である
- lifecycle transitionが見える

### P4 — Business / Security Boundary

**問い:** このBusiness文脈から要求されるinteractionについて、accepted Security authorityと矛盾しないか。

```text
Business actor / responsibility
    != authenticated principal
    != Authorization Role / Policy
```

Business actor labelからSecurity objectを自動authorしません。

### P5 — System Result / Actuality

**問い:** system resultから外部・物理世界の事実まで何を言えるか。

```text
Business instruction
  != command acceptance
  != external / physical actuality
  != Business acceptance / completion
```

Observationやexternal Evidenceが無い場合、failureともsuccessとも断定しません。

### P6 — Responsibility Handoff

**問い:** あるBusiness Stepの仕事が別Stepへ継続する責任関係は、accepted Business meaningから証明できるか。

根拠:

- explicit accepted Handoff source
- from / to / conditionのaddressable meaning

証明しないもの:

- Step adjacency
- Function sequence
- actor change
- Lifecycle transition
- Command success
- Observation alone

ObservationはHandoff condition成立のEvidence候補にはなり得ますが、Handoffそのもののauthorではありません。

## 8. Design Synthesisでの利用

Proof Obligationはcandidateをauthority化するためではなく、**候補空間を安全に縮約するため**に使います。

```text
candidate generation
        ↓
proof obligations生成
        ↓
exact authority resolution
        ↓
result
  supported       -> 候補を残す根拠の一部
  contradiction   -> 除外または修正要求
  unresolved      -> 未解決dimensionとして保持
        ↓
small candidate frontier
        ↓
LLM / human review
```

### contradiction

accepted authorityと明確に両立しない候補は、理由付きで除外できます。

### unresolved

未解決は除外理由ではありません。

たとえば、lineNumberがbusiness identityかpresentation orderか未確定なら、そのdimensionだけを残して候補を比較できます。

### supported

supportedであっても「候補全体が正しい」という意味ではありません。一つのproof obligationが満たされたことだけを示します。

## 9. DesignContextPacketとの接続

Design SynthesisがLLMへ渡すbounded contextには、単なるfact一覧だけでなく、proof結果を要約して渡せます。

概念例:

```text
DesignContextPacket
  purpose
  accepted_snapshot
  exact_facts[]
  candidate_designs[]
  proof_results[]
    question
    result
    authority_refs[]
    unresolved_reason
  eliminated_candidates[]
    reason
  unresolved_dimensions[]
  authority_limits
```

LLMへRepository全体を再推論させず、

- 何がaccepted factか
- 何がderived resultか
- 何がproposalか
- 何がunresolvedか

を分離した状態で渡します。

## 10. Acceptance Snapshotと再評価

cross-view proofでは、全artifactへ一つのglobal revision番号を付けるより、**どのaccepted source集合で評価したか**を再現できることが重要です。

概念的には次です。

```text
Acceptance Snapshot
  Business source @ revision/fingerprint
  Application source @ revision/fingerprint
  Data source @ revision/fingerprint
  Security source @ revision/fingerprint
  other required authorities...
```

ここから導出されたproof resultはsnapshot-boundです。

### Business wordingだけが変わった場合

参照identityと意味が変わらなければ、Data trace等のderived resultまで無条件に変更する必要はありません。

### referenced authorityが変わった場合

Business source自体が不変でも、Pattern / Validation / Capability / Policy等のaccepted authorityが変われば、関連proofはstaleになり再評価が必要です。

### source fileが移動した場合

locator変更とsemantic changeを区別します。

この考え方は、global revision modelの採用を要求しません。まずReferenceで、fingerprint / revision / Source Index等の既存機構だけで十分かを確認します。

## 11. Claim / Interpretationを急いでCanonical化しない

proof設計を進めると、汎用的な`Claim`や`Interpretation` objectをCoreへ入れたくなります。

しかし、現段階で必要なのはまずepistemic distinctionです。

```text
source-backed fact
accepted TEI meaning
derived cross-view fact
proposal / inference
unresolved
```

これをreviewで区別できることと、generic Claim modelを持つことは同じではありません。

次のような具体的lossがrealistic Referenceで観測された場合にのみ、durable Claim identityを再検討します。

- 同一Claimを複数sourceから長期間参照する必要がある
- 相反するInterpretationをidentity付きで保持しないとreviewできない
- Evidence更新後もClaim continuityを維持する必要がある

## 12. Referenceで検証する順序

### R1 — order entryでproof familyを通す

既存Referenceを使い、P1〜P4の基本挙動を確認します。

目的は新しいBusiness meaningを発明することではなく、既存accepted refsからproof resultを機械的に再現できるかを見ることです。

### R2 — manufacturing / inventoryでactualityとHandoffを通す

[`../issue/000050-second-realistic-handoff-corpus.md`](../issue/000050-second-realistic-handoff-corpus.md)のsecond realistic corpus候補を使い、P5 / P6を重点的に検証します。

ここでは特に、

```text
Command success
Observation
Business acceptance
Handoff condition
responsibility continuation
```

を一つへ畳まないことを確認します。

### R3 — candidate contractionへ接続する

同じ入力に対して複数候補を作り、proof結果によって、

- contradictionだけを安全に除外できるか
- unresolvedを残せるか
- supportedを過大評価しないか

を確認します。

### R4 — change / stale再評価

accepted sourceを一箇所変更し、影響するproofだけがstaleになるかを確認します。

## 13. 最小評価指標

固定KPIにはしませんが、Referenceでは少なくとも次を観測します。

- proof questionからaccepted authorityへ戻れる割合
- name similarity等のheuristicをauthorityとして使わずに済んだか
- contradictionを理由付きで再現できたか
- insufficient Evidenceをunresolvedとして保持できたか
- source change後に影響proofを再評価できたか
- derived graphを捨てて再構築して同じmeaningを得られたか
- Design Synthesisで人へ残る問いが、単純なtrace作業ではなく意味判断へ縮約されたか

## 14. 実装方式をまだ固定しない

この設計は次を要求しません。

- graph database
- RDF / OWL
- generic rule engine
- SAT / SMT solver
- `ProofObligation` Canonical kind
- `Claim` Canonical kind
- global Step ID
- global revision counter
- persisted `supported / contradiction / unresolved` enum
- LLMによるproof判定

最初のReferenceでは、既存resolverと小さいderived representationだけでも十分です。

必要なのは技術方式ではなく、**authority方向と意味分類を壊さずに問いを再評価できること**です。

## 15. 現時点の判断

現時点では、次を設計仮説として採用候補にします。

```text
cross-view review
  = complete-model validation ではなく
    bounded proof obligations の集合として扱う

proof result
  = supported / contradiction / unresolved を失わない

derived meaning graph
  = disposable projection
  != source of truth

Design Synthesis
  = contradictionを候補縮約へ使い
    unresolvedを残し
    residual meaning choiceだけをLLM / humanへ渡す
```

一方、durable proof object、generic Claim、graph storage、global revision等のCore採用は、Reference Evidenceが具体的なlossを示すまで保留します。

この方針により、三要素分析的なcross-viewの強みを保持しつつ、TEIの小さいkernel、file-native authority、Shared Acceptance Boundary、Derived Cognitionの非authority性を崩さずにDesign Synthesisへ接続できます。
