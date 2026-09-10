# ISSUE-000047: 既存Semantic資産を探索してから変更候補へ進む

- 状態: Blocked — Gate C Ready待ち
- 起点: `ISSUE-000008` Editor / AI follow-up
- 関連: `ISSUE-000027` AI Proposal / Acceptance Boundary
- Product Acceptance Scenario: **PAS-06 primary**
- 関連RQ: RQ-002 / RQ-003 / RQ-014 / RQ-015 / RQ-016 / RQ-017 / RQ-021 / RQ-023

## 目的

Gate C通過後、開発者やAIが新しい実装・定義を作る前に、Repositoryに既に存在するSemantic資産を**read-onlyで探索し、適用可能 / 不適合 / 不明を区別できること**をReferenceで実証します。

AI専用Knowledge Storeや巨大なretrieval基盤を先に作ることが目的ではありません。

最初の価値は、既存の意味を見つけられず重複authoringすること、名前が似ているだけの資産を誤用すること、過去の古い知識を現在の正本として扱うことを減らすことです。

## 基本flow

```text
purpose / intent
    ↓
explore existing Semantic assets
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

## 検索結果とSemantic適用可能性を分ける

検索順位、文字列類似度、embedding similarity、LLMの確信度等は**探索候補の順位**であってSemantic Evidenceではありません。

```text
search / retrieval result
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
- current revisionとの整合

名前が似ている、過去に使われた、AIが推奨した、という理由だけで既存資産を採用しません。

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

- source revision / dependencyへ追跡できる
- staleかどうかを判定または再生成できる
- indexを削除してもCanonical Worldが変わらない
- indexだけに存在する業務meaningを作らない
- similarity scoreをDecisionへ格上げしない

vector databaseやembeddingを初回Referenceの必須要件にはしません。

まずRepositoryのread-only探索とidentity / dependency / Evidence照合を成立させます。

### 非LLMのcandidate retrieval方式を比較する

探索方式はlexical searchやembeddingへ固定しません。Canonical Worldから決定論的に抽出したfeatureを用い、sparse expansive representation、high-dimensional hashing、role-awareなHDC / VSA等がcandidate生成に有効かは、[ISSUE-000051](000051-derived-cognitive-retrieval-evaluation.md)でarchitecture probeとして比較します。

ここで比較対象となるactivation、similarity、novelty等は、すべて探索前段のheuristic signalです。

```text
candidate mechanism
      ↓
candidate identities
      ↓
このIssueのdeterministic qualification
      ↓
applicable / incompatible / unknown
```

したがって000051の方式が高い検索性能を示しても、本Issueのqualificationを置き換えません。逆に000051が不採用でも、本Issueはexact / graph / lexical等の方式で成立できる必要があります。

## AI Knowledgeとの関係

AI向けSkill、examples、trajectory、過去Proposal等を使う場合も、それらは補助的なDerived Knowledgeです。

新しいSemantic資産を提案する前に、現在のRepository上で同等・近接・関連する資産を探索します。

過去trajectoryが「以前これを使った」と示しても、現在revisionで有効とは限りません。current Source / Evidenceへ戻って再確認します。

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

Gate C後の`order_entry`を使い、少なくとも次を比較します。

### 1. existing asset reuse

intentに対して既存Application Pattern / Capability / Relation等を発見し、明示参照candidateへ進めることを確認します。

### 2. incompatible near-match

名前や用途が近いがscope / Capability / Datasource条件が合わない資産を用意し、検索上位でも`incompatible`として拒否できることを確認します。

### 3. unknown

Evidence不足で適用可否を断定できない資産を`unknown`として保持し、推測で採用しないことを確認します。

### 4. stale derived index

source revision更新後に古いindex / cacheを利用しようとした場合、staleを検出して再読込・再生成へ進めることを確認します。

## Evidence

初回Referenceでは少なくとも次を残します。

- user / task intent
- query / exploration input
- source / repository revision
- returned candidate identities
- Source / Evidence references
- applicable / incompatible / unknown分類
- 分類理由
- explicit-reference candidateへ進んだか
- Shared Acceptance結果へのreference
- derived indexを使った場合のrevision / fingerprint
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

## 完了条件

- [ ] Gate C Ready後に着手する
- [ ] Data / Relation / Validation / Capability / Application Pattern等をread-only探索できる
- [ ] Human / AIが同じauthoritative Source / Canonical referencesへ戻れる
- [ ] search ranking / similarityとSemantic適用可能性を分離できる
- [ ] applicable / incompatible / unknownを明示できる
- [ ] unknownを推測でsuccessへ変換しない
- [ ] existing asset reuseをexplicit-reference candidateへ接続できる
- [ ] candidateがShared Acceptance Boundaryを通る
- [ ] stale derived index / cacheを検出または再生成できる
- [ ] index / embedding / trajectoryを削除してもCanonical Worldが変わらない
- [ ] AI Knowledgeを第二のCanonical正本にしない

## 非目標

- Gate C前の本格Knowledge subsystem
- 全文書を統合する巨大Knowledge Graph
- vector databaseの必須化
- embedding similarityによる自動採用
- AI内部記憶をSemantic authorityにすること
- search resultからauthoritative sourceへ直接writeすること
- Derived Experience Knowledgeの完成形

## 関連文書

- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000051-derived-cognitive-retrieval-evaluation.md`
