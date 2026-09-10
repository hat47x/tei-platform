# Derived Cognitive Exploration — 根幹趣旨と展開順

## 位置づけ

この文書は、TEIでLLMとは異なる認知方式を検討する理由と、その検討をどの順序で個別課題へ展開するかを示す入口です。

状態: **Research / planning overview — 未採用**

この文書は新しいArchitecture Layer、Canonical schema、runtime subsystemの採用を意味しません。Gate C前に実装を始める根拠でもありません。詳細なalgorithm、feature、benchmarkから読み始めなくても、まずTEI上の問題と判断原則を理解できることを目的とします。

## 根幹の問題

TEIは、業務の意味をCanonical Worldとして明示し、Source、Validation、Capability、Application Pattern、Evidence等へ追跡できることを重視します。

一方、Repositoryに意味資産が蓄積すると、次の問題が生じます。

1. 既存資産があるのに見つけられず、同じ意味を新しく作る。
2. 名前が似ている資産を見つけても、現在の目的へ本当に適用できるとは限らない。
3. 人とAIが別々の探索知識を持つと、Canonical以外の第二の正本が生まれやすい。
4. 全探索をLLMへ依存すると、単純な再発見まで非決定性、外部service、latency、costへ依存する。
5. exact lookupだけでは、語彙の異なる構造類似、partial cue、過去に見た構造の想起を扱いにくい。

したがって必要なのは「LLMを置き換える別のAI」ではありません。

必要なのは、**Canonical Worldをauthorityのまま保ちつつ、次に確認すべき既存資産を軽量・局所的・再生成可能な方法で候補化できるか**を検討することです。

## 目指す認知の役割

この検討でいう認知は、答えを決めることではなく、探索範囲を絞ることです。

```text
Canonical / Authoritative World
          ↓ observation
   candidate cognition
          ↓
   candidate identities
          ↓
 deterministic qualification
  applicable / incompatible / unknown
          ↓
 explicit-reference candidate
          ↓
 Shared Acceptance Boundary
```

認知方式が担うのは主として次です。

- 何を思い出すか
- 何が近そうか
- どの既存構造を先に見るか
- 既知のpatternから外れていそうか

一方、次は担いません。

- Semantic Factの確定
- applicabilityの最終判定
- Evidenceの代替
- Decisionの代替
- write / execution authority

この境界を短く表すと次です。

```text
Activation   != Fact
Similarity   != Applicability
Novelty      != Unknown
Association  != Dependency
Candidate    != Decision
Cognition    != Authority
```

## なぜLLMだけにしないか

LLMには、自然言語intentの解釈、曖昧な要求の整理、説明、仮説生成、新しいcompositionなどの強みがあります。

しかしTEIの探索には、LLMを使わなくても成立し得る処理があります。

- Canonical identityのexact lookup
- graph / dependency traversal
- role付き構造の比較
- partial cueからの候補回収
- 類似構造の局所探索
- novelty / familiarityのattention signal

これらを決定論的または軽量なderived mechanismで処理できれば、LLMを「最初に何でも考える唯一の認知器官」にする必要がありません。

目指すのはLLM中心ではなく、異なる性質の認知方式を組み合わせる構造です。

```text
                 Intent / Context
                       │
       ┌───────────────┼────────────────┐
       ↓               ↓                ↓
 Exact / Graph    Associative       LLM reasoning
 cognition        cognition         / explanation
       │               │                │
       └───────────────┼────────────────┘
                       ↓
                Candidate Space
                       ↓
          Deterministic Semantic Core
                       ↓
             Acceptance / Evidence
```

## ハエ脳の着想をどう扱うか

出発点の一つはショウジョウバエ嗅覚系に着想を得たFlyHash型の計算です。

重要なのは生物学的な再現ではなく、次の計算原理がTEIの探索に有効かという仮説です。

```text
explicit features
      ↓
high-dimensional expansion
      ↓
sparse competition
      ↓
small associative neighborhood
```

ただしFlyHashを採用前提にはしません。

同じ目的に対して、exact / graph、weighted set similarity、SimHash、FlyHash / DenseFly、HDC / VSA、embedding等を比較できるようにします。

したがって根幹の設計対象は特定algorithmではなく、**derived candidate cognitionをCanonical authorityから分離する境界**です。

## TEIで特に価値があり得る場面

最初に価値を確かめる対象はknowledge-first Semantic explorationです。

```text
purpose / editing context
        ↓
existing asset discovery
        ↓
candidate identities
        ↓
exact qualification
        ↓
reuse / reject / unknown
```

ここで有効性が確認できた場合に限り、次の領域へ広げます。

- Studioでの合法候補ranking
- Application Patternの構造類似探索
- cross-viewでの「確認すべき欠落」のattention signal
- Derived Experience Knowledgeの局所的再利用
- 将来の設計書import等でのformat family認識

これらを最初から一つの巨大Cognitive subsystemへ統合しません。

## 根幹から個別課題へ展開する順序

文書と開発課題は、次の順序で深くします。

### Level 0 — Product / Architecture上の趣旨

最初に問うのは「なぜ必要か」です。

- TEIのどのProduct問題を解くか
- Canonical / Evidence / Authorityとの境界は何か
- LLMと何を分担するか
- 採用しなかった場合でもTEI本体が成立するか

本書がこの入口です。

### Level 1 — Product Referenceとして何を成立させるか

次に、algorithmを決める前にknowledge-first探索のProduct動作を定義します。

担当:

- [`issue/000047-knowledge-first-semantic-exploration.md`](issue/000047-knowledge-first-semantic-exploration.md)

ここではHuman / AIが同じauthoritative Source / Canonical / Evidenceへ戻り、`applicable / incompatible / unknown`を区別できることを扱います。

### Level 2 — 非LLM方式を評価する価値があるか

Product動作を壊さず、その候補生成部分だけを差し替え可能なprobeとして評価します。

担当:

- [`issue/000051-derived-cognitive-retrieval-evaluation.md`](issue/000051-derived-cognitive-retrieval-evaluation.md)

ここで初めてFlyHash、HDC/VSA等の比較を扱います。

### Level 3 — 比較問題をどう作るか

algorithmごとに都合のよい問題へ変わらないよう、共通benchmarkを定義します。

担当:

- [`DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md`](DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md)

ここではRetrieval / Qualification / Invariance oracleを分け、`order_entry`と将来の第二Applicationを使うpressure caseを定義します。

### Level 4 — 共通入力をどう抽出するか

最後に、比較方式へ渡すalgorithm-neutralなfeature corpusを具体化します。

担当:

- [`DERIVED_COGNITIVE_FEATURE_EXTRACTION_CONTRACT_2026-09.md`](DERIVED_COGNITIVE_FEATURE_EXTRACTION_CONTRACT_2026-09.md)

ここでF0 / FL / F1 / F2 / F3等の詳細へ降ります。

### Level 5 — 実装方式

Gate C後、上記Referenceとbenchmarkがmaterializeできる時点で初めて、package、API、persistent index、encoder dimension等を実装課題へ落とします。

Reference EvidenceなしにLevel 5から逆算してArchitectureを固定しません。

## 開発順と文書整備順を一致させる

今後は、詳細な実装課題を見つけた場合でも、先に次を確認します。

```text
1. 上位のProduct目的は既に説明されているか
2. どのReference Scenarioで価値を確認するか
3. どのIssueが責務を所有するか
4. 評価契約は必要か
5. その後にfeature / schema / API / algorithmへ降りる
```

上位の説明が不足している場合は、詳細文書を先に増やさず、まず上位文書を補います。

逆に、上位文書へalgorithm detailを大量に持ち込まず、詳細は下位文書へ送ります。

## 採否を決める問い

最終的に問うのは「FlyHashのaccuracyが高かったか」ではありません。

少なくとも次を満たす場合にのみ、非LLM認知方式をProductへ採用する価値があります。

1. exact / graphだけでは見つけにくい有用候補を実際に回収できる。
2. lexical near-matchをSemantic truthへ誤昇格させない。
3. true novel caseで、nearest itemを無理に再利用候補として扱わない。
4. indexを削除・再生成してもCanonical Worldが変わらない。
5. Source revisionとstalenessを追跡できる。
6. LLMを使わないこと自体ではなく、latency / cost / locality / explainability等の実用価値がある。
7. 複雑な新subsystemを維持するコストを上回る価値がある。

この条件を満たさなければ、exact / graph / lexical探索だけで進める判断も正当です。

## Gateとの関係

現在のGate B / Gate C主線は変更しません。

```text
Gate B
  ↓
First Web Target
  ↓
Gate C
  ↓
knowledge-first Product Reference
  ↓
derived cognition comparison
```

Gate C前に行うのは、根幹趣旨、評価課題、feature source、benchmark contract等のread-onlyな設計準備までです。

Cognitive Layerを独立Architecture Layerへ昇格するか、共通interfaceや永続indexを設けるか、learned associationを導入するかは、Reference Evidenceの後に判断します。

## 現時点で固定しないもの

- `Derived Cognitive Layer`という名称を恒久的Architecture用語にすること
- FlyHash / DenseFly / BioHash / HDC / VSAの採用
- embedding / vector databaseの採用または排除
- `internal/cognitive`等のpackage構造
- feature schemaのproduction固定
- online learning
- learned associationの永続化
- similarity / activation scoreの共通尺度

これらは根幹原則から導出される個別課題であり、先に固定しません。

## 関連する上位・隣接文書

- [`README.md`](README.md) — TEI Planning全体の入口
- [`TECH_RESEARCH_INTEGRATION_2026-09.md`](TECH_RESEARCH_INTEGRATION_2026-09.md) — 外部研究をTEIの境界へ翻訳する統合方針
- [`issue/000040-semantic-component-composition.md`](issue/000040-semantic-component-composition.md) — 意味資産の探索・組み立て
- [`issue/000027-ai-proposal-acceptance-boundary.md`](issue/000027-ai-proposal-acceptance-boundary.md) — ProposalとShared Acceptance Boundary

この文書を詳細仕様の第二正本にはしません。根幹の趣旨や展開順が変わった場合はここを更新し、個別algorithmやfixtureの細部は下位文書で更新します。
