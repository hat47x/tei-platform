# ISSUE-000047: 既存Semantic資産を探索してから変更候補へ進む

- 状態: Blocked — Gate C Ready待ち
- 起点: `ISSUE-000008` Editor / AI follow-up
- 関連: `ISSUE-000027` AI Proposal / Acceptance Boundary
- Product Acceptance Scenario: **PAS-06 primary**
- 関連RQ: RQ-002 / RQ-003 / RQ-014 / RQ-015 / RQ-016 / RQ-017 / RQ-021 / RQ-023

## このIssueを最初に読むときの要点

このIssueの目的は、高性能な検索engineを作ることではありません。

TEIで新しい定義や実装を作ろうとしたとき、まず既存資産を探し、見つかった候補について**「現在の目的へ本当に使えるか」を決定論的に確認する境界**を成立させることが中心です。

```text
既存資産を探す
  ↓
候補が見つかる
  ↓
identity / dependency / scope / Source / Evidence / current authorityを確認
  ↓
applicable / incompatible / unknown
  ↓
明示参照を持つProposal候補へ渡す
```

ここで重要なのは、検索順位や類似度と、Semanticな適用可能性を分けることです。

```text
見つけやすい
  !=
現在の業務文脈で使える
```

### このIssueが減らす不確実性

主に次の問いへ答えます。

1. 既存資産を見つけられず、同じ意味を重複authoringしてしまわないか。
2. 名前や構造が似ているだけのnear-matchを、誤って「使える」と判定しないか。
3. Evidence不足やstaleな候補を`unknown`のまま保持できるか。
4. Human / Studio / AIが別々のKnowledge正本を持たず、同じCanonical / Source / Evidenceへ戻れるか。
5. derived indexや認知方式を削除・交換しても、Canonical Worldが変わらないか。

### 完了すると何を判断できるか

このIssueが完了すると、**既存資産探索から得た候補を、現在のintentへ再利用可能かどうか安全に分類し、明示参照付きの変更候補へ渡してよい**と判断できます。

ただし、ここで`applicable`になったことは、その変更が採用されたことを意味しません。

```text
retrieval
  ↓
qualification        ← このIssue
  ↓
Proposal
  ↓
Shared Acceptance    ← ISSUE-000027 / 000046
  ↓
authoritative source
```

また、どのcandidate retrieval方式が優れているかの比較は[ISSUE-000051](000051-derived-cognitive-retrieval-evaluation.md)が担当します。本Issueは000051の方式が不採用でも成立する必要があります。

## 目的

Gate C通過後、開発者やAIが新しい実装・定義を作る前に、Repositoryに既に存在するSemantic資産を**read-onlyで探索し、適用可能 / 不適合 / 不明を区別できること**をReferenceで実証します。

AI専用Knowledge Storeや巨大なretrieval基盤を先に作ることが目的ではありません。

最初の価値は、既存の意味を見つけられず重複authoringすること、名前が似ているだけの資産を誤用すること、過去の古い知識を現在の正本として扱うことを減らすことです。

Definition-centered feedback loopでは、このIssueは**attention / retrieval後のdeterministic qualification**を所有します。探索対象はcurrent authoritative definitionそのものだけでなく、provenance付きReference patternやderived structural memoryへ広げられますが、どの経路もcurrent Source / Canonical / Evidenceへ戻って確認します。

## 基本flow

```text
purpose / intent / selected local context
    ↓
explore existing Semantic assets
    + exact / graph neighborhood
    + provenance-bound Reference patterns
    + optional derived cognition
    ↓
inspect identity / dependency / source / evidence
    ↓
classify
  ├─ applicable
  ├─ incompatible
  └─ unknown
    ↓
explicit-reference candidate
    ↓
Shared Acceptance Boundary
```

探索結果そのものはCanonical authorityではありません。

## 初回探索対象

v1 Referenceで少なくとも次を横断して検索・参照できることを対象にします。

- Data
- Relation
- Validation
- Capability
- Capability Binding
- Application Pattern
- Query / Datasource
- Source Reference
- Reference Scenario
- Conformance / Evidence

最初から全Repository文書、全履歴、外部Knowledgeを一つの検索体系へ統合しません。

## 三種類のmemoryを区別する

探索Productではknowledge sourceを一つへ畳みません。

```text
1. authoritative semantic memory
   authoritative source / Canonical meaning / accepted Decision

2. derived associative / structural memory
   Source Index / dependency index / lexical / graph / pattern / embedding / sparse code

3. episodic / experience memory
   Proposal / accept-reject / run / Observation / correction trajectory
```

このIssueは主に1を確認し、2から候補を受け取ります。3は「以前この候補が役立った」というrouting hintにはできますが、current applicabilityのEvidenceにはしません。

derived / episodic memoryを削除してもCanonical Worldが変わらないことを要求します。

## 検索結果とSemantic適用可能性を分ける

検索順位、文字列類似度、embedding similarity、Reference pattern frequency、LLMの確信度等は**探索候補の順位**であってSemantic Evidenceではありません。

```text
search / retrieval / pattern match
        ≠
applicable semantic asset
```

適用可能と判断するには、少なくとも対象に応じて次を確認します。

- Canonical / qualified logical identity
- expected subject kind
- Application / scope
- dependency
- Source reference
- Validation / Capability条件
- Evidence / Conformance
- required accepted authority / Evidenceとのcurrent整合

名前が似ている、Referenceに頻出する、過去に使われた、AIが推奨した、という理由だけで既存資産を採用しません。

## applicable / incompatible / unknown

### applicable

現在のintentへ明示的に参照でき、必要なidentity / dependency / constraint / Evidenceを満たすものです。

### incompatible

近い目的でも、scope、Capability、Datasource、Validation、Target条件等が現在のintentと矛盾するものです。

不適合理由を残し、検索上位だからという理由で利用しません。

### unknown

必要なEvidenceや条件が不足し、適用可能とも不適合とも断定できないものです。

`unknown`を暗黙の「たぶん使える」へ変換しません。

必要なら追加確認へ進みます。

## HumanとAIで探索正本を分けない

開発者向けStudio / CLIとAIが、別々のSemantic Knowledgeを人手で二重管理する構造は避けます。

共通する正本はRepository上のauthoritative source / Canonical identity / Evidenceです。

```text
Human exploration ─┐
Studio / CLI -------|--> repository / source index / evidence references
AI exploration -----┘
```

UI、検索index、ranking、要約は異なっても構いません。

## Derived index / cache

高速化や検索品質のために、将来index、embedding、graph、cache等を利用できます。

ただしderived resourceとして扱います。

少なくとも次を要求します。

- 実際に表現生成へ使ったscoped build input / dependencyへ追跡できる
- 実依存build inputが変わった場合にstaleを判定または安全に再生成できる
- 無関係なaccepted changeだけで全derived representationを一律staleにしない
- indexを削除してもCanonical Worldが変わらない
- indexだけに存在する業務meaningを作らない
- similarity scoreをDecisionへ格上げしない

さらに、derived representationのfreshnessとcurrent applicabilityのfreshnessを同一視しません。

```text
derived representation freshness
  != current semantic qualification / proof freshness
```

retrievalに使ったindexがfreshでも、`applicable / incompatible / unknown`の判定はrequired accepted authority / Evidenceへ独立に戻って確認します。cognitive freshnessのためにrepository-wideなglobal revision counterを要求しません。

vector databaseやembeddingを初回Referenceの必須要件にはしません。

まずRepositoryのread-only探索とidentity / dependency / Evidence照合を成立させます。

## Reference Pattern Memory

過去のX-TEA / XEAD Reference ModelやTEI自身のrealistic Referenceから抽出したpatternを、**provenance-bound derived candidate source**として利用する余地を持ちます。

これは「Reference Modelをテンプレートlibraryとして自動適用する」ことではありません。

最小record候補:

```text
pattern id / label
nature: observed / candidate
structural signature
provenance source / model
examples
applicability question
known non-proof / caveat
```

候補利用:

```text
intent / local structure
  ↓
multiple Reference patterns
  ↓
compare applicability / differences
  ↓
applicable / incompatible / unknown
```

### first pattern pressure

既に過去分析で観測・候補化された、たとえば次の構造をReference caseへ使えます。

- header-detail transaction
- conditional subtype / role
- self-referential hierarchy / chain
- plan / actual / consequence / ledger
- derived summary
- composition / BOM-like structure

ただし名称やpattern IDをcurrent modelへそのまま移植しません。TEI-nativeで語彙を変えたcaseに対し、構造candidateとして思い出せるかを確認します。

### negative case

Reference群に頻出するpatternでもcurrent Business requirementには合わないcaseを必須にし、`incompatible`へ落とせることを確認します。

```text
frequency != requirement
pattern match != applicability
```

## 非LLMのcandidate retrieval方式を比較する

探索方式はlexical searchやembeddingへ固定しません。Canonical Worldから決定論的に抽出したfeatureを用い、sparse expansive representation、high-dimensional hashing、role-awareなHDC / VSA等がcandidate生成に有効かは、[ISSUE-000051](000051-derived-cognitive-retrieval-evaluation.md)でarchitecture probeとして比較します。

ここで比較対象となるactivation、similarity、novelty等は、すべて探索前段のheuristic signalです。

```text
candidate mechanism
      ↓
candidate identities / pattern candidates
      ↓
このIssueのdeterministic qualification
      ↓
applicable / incompatible / unknown
```

したがって000051の方式が高い検索性能を示しても、本Issueのqualificationを置き換えません。逆に000051が不採用でも、本Issueはexact / graph / lexical等の方式で成立できる必要があります。

## AI Knowledgeとの関係

AI向けSkill、examples、trajectory、過去Proposal等を使う場合も、それらは補助的なDerived / Experience Knowledgeです。

新しいSemantic資産を提案する前に、現在のRepository上で同等・近接・関連する資産を探索します。

過去trajectoryが「以前これを使った」と示しても、current accepted authority / Evidenceの下で現在も有効とは限りません。current Source / Evidenceへ戻って再確認します。

### experienceからのlocal learning

将来、過去のaccept / reject結果をcandidate rankingへ反映する場合、まず次のような局所的更新を比較します。

- task-local association
- known false-positive suppression
- frequently useful Reference pattern hint
- outcome-local ranking weight

それでもbuild-input provenance / current qualification authority / outcomeへtraceでき、learned stateを無効化できる必要があります。

accepted semantic ruleをlearned weightだけへ保存しません。

### Knowledge routingとselective disclosure

Knowledge資産が利用可能であることと、現在のtaskでAIへ渡すことを同一視しません。

```text
available knowledge
      ↓ routing / shortlist
selected candidates
      ↓ deterministic qualification
applicable / incompatible / unknown
      ↓
必要な範囲だけdisclose
```

routing方式は、exact、lexical、graph、embedding、Derived Cognitive Retrieval、model-native routing等を比較可能にします。特定方式を本Issueの前提にはしません。

routing score、model内部signal、検索順位はcandidate generationのsignalであり、Semantic適用可能性のEvidenceそのものではありません。selected candidateも既存のidentity / dependency / Source / Evidence / current accepted authorityによるqualificationを通します。

全Skill / 全Knowledge metadataを毎回promptへ投入することをarchitecture要件にはしません。必要なKnowledgeだけを選択し、何を選び、何を選ばなかったかを必要に応じてprovenanceとして残せることを優先します。

### Verified Experience Memoryの候補

過去trajectoryを再利用する場合、単なる会話履歴や操作列だけでなく、検証済みの局所関係へ圧縮する余地を残します。

候補例:

- condition
- action / semantic operation
- observed outcome
- failure mode
- boundary condition
- source Target / Driver / build input
- Observation / Evidence reference

このmemoryはDerived Experience Knowledgeであり、accepted Semantic RuleやBusiness Procedureではありません。

過去の成功関係が現在も成立するとは限らないため、current Source / dependency / authority / Target条件へ戻って再評価します。memoryを無効化・削除してもCanonical Worldが変化しないことを維持します。


## Knowledge Portabilityへのhandoff

candidate qualification後、別Applicationで再利用する場合のauthority / provenance / freshnessはISSUE-000053 Phase 5へ渡す。

詳細は `plan/KNOWLEDGE_PORTABILITY_HANDOFF_2026-09.md` を参照する。

本Issueからhandoffするとき、少なくとも次を失わない。

- candidateのCanonical subject kind / identity
- authoritative Source / Evidence
- current qualificationと理由
- required dependency / destination条件
- discovery provenance
- freshnessに必要なactual source input

特に、

```text
discovered through Application A
  != Application A owns the candidate
```

を維持する。

また、

```text
applicable
  != linked reuse
```

である。

たとえばApplication-owned Patternはdirect shared authorityとしては不適切でも、destination-owned Patternを作るためのprovenance candidateとしては利用できる。

そのreuse relationの選択は本Issueでは固定せず、Phase 5へ渡す。

## explicit-reference candidateへ進む

探索後に既存資産を利用する場合、暗黙の名前解決やAI内部記憶だけで変更しません。

候補は既存のqualified logical key / Canonical reference / Source reference等を明示し、`ISSUE-000046` / `ISSUE-000027`と同じShared Acceptance Boundaryへ接続します。

```text
exploration result
   ↓ explicit reference
candidate
   ↓
Shared Acceptance Boundary
   ↓
authoritative source
```

探索はAcceptanceを置き換えません。

## 最初のReference Scenario

Gate C後の`order_entry`を使い、[`../DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`](../DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md)のone-change loopへ参加します。

少なくとも次を比較します。

### 1. existing asset reuse

intentに対して既存Application Pattern / Capability / Relation等を発見し、明示参照candidateへ進めることを確認します。

### 2. incompatible near-match

名前や用途が近いがscope / Capability / Datasource条件が合わない資産を用意し、検索上位でも`incompatible`として拒否できることを確認します。

### 3. Reference pattern candidate

local modelと構造的に近いReference patternをprovenance付きで提示し、直接採用せずdifferences / applicabilityをreviewできることを確認します。

### 4. unknown

Evidence不足で適用可否を断定できない資産を`unknown`として保持し、推測で採用しないことを確認します。

### 5. stale derived index

representation生成に使ったscoped build input更新後に古いindex / cache / pattern-derived featureを利用しようとした場合、staleを検出して再読込・再生成へ進めることを確認します。無関係なaccepted changeだけで全indexをstale扱いにはしません。

## Evidence

初回Referenceでは少なくとも次を残します。

- user / task intent
- selected definition / local context
- query / exploration input
- current qualificationに使ったaccepted authority / Evidence references、および必要な場合のbaseline repository revision
- candidate mechanism（exact / lexical / pattern / cognition / model-native routing / AI等）
- selected / disclosed knowledge identities（利用した場合）
- routing / shortlist理由またはprovenance（利用した場合）
- returned candidate identities / pattern identities
- Source / Evidence / provenance references
- applicable / incompatible / unknown分類
- 分類理由
- explicit-reference candidateへ進んだか
- Shared Acceptance結果へのreference
- derived indexを使った場合のscoped build-input provenance / fingerprints
- stale negative case

AIを使った場合、model名やpromptは補助Evidenceになり得ますが、Semantic適用可能性の根拠そのものにはしません。

## Gate Cとの関係

本IssueはGate C前に本格実装しません。

Gate Cから次を受け取ります。

- Handover Baseline / Application identity
- 第一Web TargetのSource-to-Canonical trace
- Semantic / Functional / Visual Evidence
- Target差 / unresolved

これにより、「何を再利用すべきか」を抽象語彙だけから決めず、実在するReference資産で検証します。

Gate C前にはReference pattern provenanceの整理やbenchmark設計までを行えますが、production index / learned memoryは導入しません。

## 完了条件

- [ ] Gate C Ready後に着手する
- [ ] Data / Relation / Validation / Capability / Application Pattern等をread-only探索できる
- [ ] Human / AIが同じauthoritative Source / Canonical referencesへ戻れる
- [ ] authoritative / derived associative / episodic memoryを混同しない
- [ ] search ranking / similarity / pattern frequency / routing signalとSemantic適用可能性を分離できる
- [ ] available Knowledgeとselected / disclosed Knowledgeを分離できる
- [ ] routing mechanismを交換してもqualification / Canonical meaningが変化しない
- [ ] applicable / incompatible / unknownを明示できる
- [ ] unknownを推測でsuccessへ変換しない
- [ ] provenance付きReference patternをcandidate sourceとして扱える
- [ ] Reference patternのincompatible negative caseを拒否できる
- [ ] existing asset reuseをexplicit-reference candidateへ接続できる
- [ ] candidateがShared Acceptance Boundaryを通る
- [ ] stale derived index / cacheをscoped build inputから検出または再生成できる
- [ ] index / embedding / trajectory / learned ranking / experience memoryを削除してもCanonical Worldが変わらない
- [ ] verified experience candidateを元Execution / Observation / revisionへtraceできる
- [ ] AI KnowledgeやReference pattern memoryを第二のCanonical正本にしない

## 非目標

- Gate C前の本格Knowledge subsystem
- 全文書を統合する巨大Knowledge Graph
- vector databaseの必須化
- Reference patternの自動適用
- pattern frequencyによる自動採用
- embedding similarityによる自動採用
- AI内部記憶をSemantic authorityにすること
- search resultからauthoritative sourceへ直接writeすること
- cognitive freshnessのためのrepository-wide global revision counter
- global online learning
- Derived Experience Knowledgeの完成形

## 関連文書

- `docs/definition-centered-feedback-loop.md`
- `plan/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000051-derived-cognitive-retrieval-evaluation.md`