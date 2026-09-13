# Definition-Centered Feedback Loop — P0 Continuation 2026-09

Status: **supporting static audit / P0-3〜P0-4 only / no new Architecture / no actual Product Evidence**

Primary Architecture: `../docs/definition-centered-feedback-loop.md`  
Primary Plan: `DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`  
P0-1 Inventory: `DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md`  
P0-2 Routing: `DEFINITION_FEEDBACK_ROUTING_CATALOG_2026-09.md`  
Issue: GitHub #570

## 1. 位置づけ

この文書は、Definition-centered feedback loopのPhase 0でmain上に未配置だった次の二点だけを補完するsupporting auditです。

- P0-3 authoritative / derived / episodic memory separation
- P0-4 X-TEA / XEAD Reference Pattern provenance handoff

P0-1とP0-2は既にmain上の専用文書を正とするため、本書へ複製しません。

```text
P0-1 Definition / Derived Artifact Inventory
        ↓
P0-2 Feedback Routing Catalog
        ↓
P0-3 Memory Separation   ← this document
P0-4 Pattern Handoff     ← this document
```

本書は新しいArchitecture、Plan、Product Requirement、Gate、PAS、production schemaではありません。

すべてstatic source / artifact reviewです。actual executionやGate readinessを示しません。

## 2. P0-3 — Memory separation audit

Definition-centered loopでは、Repository・Editor・AI・Runtimeが利用する「記憶」を一つのKnowledge Storeへ畳みません。

最低限、次の三つを分けます。

```text
1. authoritative memory
2. derived structural / associative memory
3. episodic / experience memory
```

この三分離はproduction storage topologyを固定するものではなく、authorityを混同しないためのreview modelです。

## 3. Authoritative memory

accepted meaning / contract / realization choiceを、それぞれのowner sourceで保持します。

代表:

- Semantic Definition
- Application intent / Pattern
- Database Contract
- native SQL等のauthoritative asset
- Persistence / Query / Capability realization mapping
- Environment / Resource / Permission choice
- promoted Release snapshot

重要なのは一つの巨大semantic fileへ集約することではありません。

```text
small concern-owned authorities
        +
exact references / mappings
        ↓
accepted system meaning
```

を維持します。

### 3.1 Authorityはconcernごとに異なる

P0-1のA〜F分類でいうAだけがRepository内の唯一のauthorityではありません。

- Semantic / Application meaningではA
- native assetではC
- realization / environment selectionではD
- promoted Releaseではtime-frozen accepted snapshot

が各concernのauthorityになり得ます。

ただしC / D / ReleaseがAのsemantic meaningを複製する第二semantic正本になってはいけません。

## 4. Derived structural / associative memory

代表:

- Source Index
- dependency graph
- Data / Function / Business matrix
- normalized lookup index
- Reference Pattern inventory
- derived feature corpus
- FlyHash / HDC / embedding representation
- retrieval ranking cache

原則:

```text
delete derived memory
  -> authoritative sources remain sufficient
  -> derived memory can be rebuilt from identified inputs
```

### 4.1 Source Indexから継承する性質

現行RepositoryではSource Index自体をReference snapshotとして保存せず、必要時にCanonical sourceから導出します。

将来のcognitive indexも同じ性質を基本とします。

- input revisionへ戻れる
- generation pathを説明できる
- staleを検出できる
-削除・再構築できる
- ranking / similarityをFactへ昇格しない

### 4.2 AIのためにDefinitionを肥大化させない

次をcognitionの都合だけでmandatory authoringにしません。

- embedding専用description
- prompt hint
- synonym list
- search-only tag
- model-specific annotation
- feature vector
- nearest-neighbor label

business meaningとして独立に必要ならauthorできます。

判断基準は「AIに便利か」ではなく、accepted meaningとして必要かです。

## 5. Episodic / experience memory

代表:

- Proposal history
- accepted / rejected candidate trajectory
- Durable Run / execution episode
- quality / evaluation Evidence
- actual correction history
- AI tool / reasoning trajectory when evaluation上必要

過去の成功や失敗はcandidate generation / ranking / explanationへ利用できますが、それだけでcurrent authorityにはなりません。

```text
past success     != current applicability
past acceptance  != current execution authority
trajectory       != Business Procedure
model confidence != semantic evidence
```

### 5.1 accepted changeの残り方

ProposalがAcceptされた後、accepted meaningは正しいowner sourceへ残します。

```text
Proposal / episode
   ↓ Accept
owner source update
   ↓
reload / validation
```

F側のProposal historyだけをaccepted meaningの唯一のdurable recordにしません。

## 6. P0-3 static result

現時点の`order_entry` authoritative sourceへ、FlyHash / embedding / LLM prompt等のcognition専用mandatory metadataが混入している状態は確認していません。

また、Source Index等のderived structureが第二semantic sourceとして保存されている状態もありません。

今後のinvariant:

```text
delete derived / episodic memory
  -> accepted meaning remains reconstructable
```

後続Referenceで確認すべきpressure:

- derived / associative index削除後のcold rebuild
- AI provider無しでのsource reload / validation
- stale indexの検出
- episodic memory削除時のaccepted meaning invariance
- algorithm交換時のsource schema invariance

これらのactual Evidenceは本static auditでは未取得です。

## 7. P0-4 — X-TEA / XEAD Reference Pattern provenance handoff

X-TEA / XEAD系Reference Modelから得た反復構造は、TEIの自動rule、Core primitive、`std.*`へ直接変換しません。

利用目的は、**どのReferenceで何が観測されたかを失わず、比較可能なcandidateとして表面化すること**です。

## 8. Candidate corpusの最低情報

production schemaは固定せず、conceptually少なくとも次へ戻れることを要求します。

```text
pattern identity / label
provenance model / source
observed vs candidate nature
structural signature
representative examples
applicability question
known non-proof / counter-condition
```

### 8.1 provenanceが必要な理由

「よくあるPattern」だけでは次を区別できません。

- どのdomain / Referenceで観測されたか
- 一回の事例か複数事例か
- sourceで明示されていたか分析からderiveしたか
- 現在のrequirementへ適用できるか
- counterexampleがあるか

したがってPattern knowledgeをLLMのpretraining memoryだけへ委ねません。

## 9. Candidate presentation

自然言語intentやlocal model contextに対して、単一の「正解」を返すより複数候補を比較可能にします。

```text
Candidate A / B / C
  ├─ structure
  ├─ provenance / Reference evidence
  ├─ applicability
  ├─ incompatibility / disadvantage
  ├─ unresolved
  └─ example Dataset / Scenario where useful
```

candidate rankingはAcceptanceではありません。

## 10. Function viewのhandoff

X-TEA / XEADから継承するのは、独立`Function` nounを必ず持つことではありません。

現TEIで既存sourceからlosslessに再構成できる範囲は:

```text
View / interaction
+ read / CUD
+ Action / Capability
+ call relation
+ input / output
  ↓
derived Function lens
```

として扱います。

realistic ReferenceでFunction固有meaningが失われることが示された場合だけauthoring surfaceをreopenします。

`Function`という歴史的conceptが存在すること自体は新しいCanonical kindの採用理由になりません。

## 11. Business viewのhandoff

Task / Business ActionはApplication Functionと同一ではありません。

Referenceで次が実在するなら、端的さを理由に消しません。

- manual work
- external organization / system work
- human judgment
- Business acceptance
- responsibility Handoff
- unresolved realization

一方、現在のBusiness Procedure Reference fixtureは`canonical=false`です。

RQ-025 / RQ-026 workstreamのProcedure identity / authority判断を、このPattern handoffから先回りしてCanonicalへ昇格しません。

## 12. Standard adoption guard

Reference knowledgeの採用判断では次を維持します。

```text
Reference frequency != Product Requirement
pattern match        != semantic applicability
similarity           != accepted relation
retrieval rank       != Decision
candidate            != authority
```

Core / `std.*` / authoring syntax追加を検討できるのは、少なくとも:

1. 複数のrealistic domainでdurable semantic needが再現される
2. 現行Definition / Relation / Projection / Bindingではlosslessに表せない
3. derived-only approachではdefectを防げない
4. 追加するauthoring burdenを上回るdeterministic benefitがある
5. AI / Provider / Target固有都合ではなくProduct meaningとして残る

場合です。

## 13. P0-3 / P0-4 status

### P0-3 Memory separation audit

**Static architecture audit complete.**

Cold reconstruction / deletion invarianceのactual execution Evidenceは未取得です。

### P0-4 X-TEA / XEAD Pattern handoff

**Planning-level provenance / authority boundary complete.**

pattern importer、vector index、Pattern Card production schema、特定cognitive algorithmは未実装・未採用です。

## 14. Phase 0の収束

main上のP0成果を次のように一意にします。

```text
P0-1 Definition / artifact inventory
  -> DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md

P0-2 Feedback routing
  -> DEFINITION_FEEDBACK_ROUTING_CATALOG_2026-09.md

P0-3 Memory separation
P0-4 X-TEA / XEAD Pattern handoff
  -> this document
```

同じ内容を別Architecture /別inventoryへ複製しません。

## 15. 次のownership

このP0 completionを理由に、新しいDefinition / AI / cognition workstream branchを増やしません。

Gate C後の実作業は既存ownerへ戻します。

- Studio local authoring: `ISSUE-000046`
- knowledge-first exploration: `ISSUE-000047`
- AI Proposal / Shared Acceptance: `ISSUE-000027`
- derived cognition comparison: `ISSUE-000051`
- Business cross-view: `ISSUE-000042` / RQ-025 / RQ-026

次の順序はPrimary Planを正とします。

```text
Gate B / Gate C mainline
  ↓
Gate C Ready
  ↓
one-change closed loop
  ↓
Reference Pattern candidate source
  ↓
non-LLM cognition comparison
  ↓
actual Observation feedback
```

## 16. Branch convergence

本auditは既存#570 branch上で行い、新branchを作りません。

PR merge後は`CONTRIBUTING.md`のBranch / PR運用に従い、#570 branchを履歴保管場所として放置せず、必要ならcurrent mainへ戻します。

Definition-centered Architecture / Primary Plan / P0-1 / P0-2を本書へ複製しないことも、document-sideのconvergence ruleとします。
