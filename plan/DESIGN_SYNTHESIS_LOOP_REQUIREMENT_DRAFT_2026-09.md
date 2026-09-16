# Design Synthesis Loop Requirements Draft — ローカル認知で設計空間を縮約する

## 位置づけ

この文書は、TEIにおけるAI / 認知機能の**上位Product Goal**を整理するRequirement Draftです。

状態: **Product direction / requirement draft — 未採用**

Derived Cognition、knowledge-first探索、LLM、Studioはこの目標を実現するための手段です。特定のFlyHash、HDC/VSA、embedding model、local LLM、solver、agent frameworkを採用する文書ではありません。

文書上の位置は次です。

```text
Definition-Centered Feedback Loop
        ↓
本書: Design Synthesis LoopのProduct Goal / Requirements
        ↓
Derived Cognitive Exploration Overview
        ↓
Derived Cognitive Requirement Draft
        ↓
ISSUE-000047 / 000051
        ↓
Benchmark / Feature / implementation details
```

## 1. 長期目標

TEIが目指すのは、生成AIに設計書やコードを大量生成させることではありません。

目標は、**上流工程の曖昧な業務意図から複数の設計候補を作り、既存の業務意味・Reference・他ビュー・constraint・Evidence・runtime能力を使って候補空間を段階的に狭め、最小限の人間判断から実装・テスト・各Target projectionまで再生成できること**です。

理想状態では、人間の主な仕事を次へ寄せます。

```text
人が主に担う
  purpose / business intent
  irreducible meaning
  本当に複数解が残る選択
  exception / responsibility / policy
  acceptance / accountability

機械へ寄せる
  既存資産探索
  類似pattern想起
  dependency / consistency検査
  候補案生成
  不適合案の除外
  cross-view照合
  impact分析
  standard implementation生成
  migration / binding候補
  test derivation
  regression / conformance実行
  feedback整理
```

「手作業による設計・実装・テストがほぼ消える」は、人間のauthorityを消すことではありません。**再生成可能な作業を人間が反復して書く必要を減らし、人間を意味選択と受理へ集中させる**ことを意味します。

## 2. knowledge-first探索は最初の実証スライス

これまでのDerived Cognition検討では、既存Semantic assetを安全に再発見するknowledge-first探索を中心にしてきました。

これは重要ですが、最終目的ではありません。

```text
Phase 1
existing asset recall
        ↓
Phase 2
multiple design candidates
        ↓
Phase 3
constraint / cross-view based contraction
        ↓
Phase 4
accepted semantic design
        ↓
Phase 5
deterministic implementation / test derivation
        ↓
Phase 6
runtime / simulation feedback closure
```

最初にPhase 1を行うのは、候補探索のauthority境界を小さく証明できるからです。価値が確認できた認知部品をPhase 2以降へ広げます。

## 3. Design Synthesis Loop

想定する全体loopは次です。

```text
business purpose / partial requirements / existing system facts
        ↓
normalize intent and known facts
        ↓
LOCAL PREPARATION
  exact Canonical lookup
  graph / dependency traversal
  Reference pattern recall
  constraint propagation
  cross-view consistency checks
  structural / associative retrieval
  novelty / conflict / coverage detection
        ↓
structured design candidate skeletons
  candidate A / B / C / ...
        ↓
LOCAL CONTRACTION
  hard constraint elimination
  qualification
  dependency closure
  capability / target feasibility
  Reference comparison
  simulation / static checks where possible
  unresolved dimension extraction
        ↓
small Pareto-like candidate frontier
        ↓
LOCAL LLM when useful
  compare / explain / combine
  fill only unresolved semantic gaps
  propose minimum semantic deltas
        ↓
structured proposals
  + exact refs
  + assumptions
  + unresolved
  + predicted impact
        ↓
human review / minimal choices / Acceptance
        ↓
authoritative definitions
        ↓
deterministic derivation
  UI / schema / query binding / runtime configuration
  generated code where required
  tests / fixtures / conformance checks
        ↓
execute / simulate / observe
        ↓
feedback routing
        ↺
```

ここで最重要なのは、**LLMをloopの最初に置かない**ことです。

LLMへ渡す前に、Repository全体を自由探索させる代わりに、TEIが知っているexact factとcandidate spaceをローカルで可能な限り整理します。

## 4. LLM呼び出し前にローカルで行う認知

### C0 — exact / symbolic cognition

- Canonical identity lookup
- type / relation / validation / capability解決
- Source / revision / Evidence解決
- dependency / reverse dependency
- authority / ownership boundary

既知の事実をLLMに再推論させません。

### C1 — constraint / rule cognition

候補設計を、自然言語の「よさ」ではなく既知constraintで削ります。

例:

- type compatibility
- relation cardinality
- uniqueness / positivity / validation
- read/write boundary
- required capability
- target capability
- provider / binding feasibility
- authorization / resource boundary
- cross-view required reference

実装候補はrule engine、constraint propagation、CSP / SAT / SMT等を比較できますが、特定solverはProduct contractにしません。

### C2 — pattern / Reference cognition

- Application Pattern
- Reference Model pattern
- accepted design precedent
- past successful composition

をprovenance付き候補として呼び戻します。

patternの存在を正解とはみなしません。

### C3 — associative / structural cognition

- lexical similarity
- structural neighborhood
- sparse high-dimensional association
- role-aware similarity
- partial cue recall
- novelty / familiarity

を用い、exact lookupでは拾えない候補を足します。

### C4 — cross-view / anomaly cognition

Data、Function / Interaction、Business Procedure、Target / Runtime等のviewを照合し、次をattention signalとして出します。

- missing relation / missing action candidate
- unexplained write
- orphan capability
- responsibility gap
- duplicated meaning
- unusual pattern
- incompatible context
- design residual / unresolved

ここでもdiagnostic candidateとFactを区別します。

### C5 — combinatorial design skeleton generation

LLMへ自由生成させる前に、既存部品とvariationから構造的に作れる設計案を列挙・組成します。

例:

```text
record + reference
record + header-detail
browse + record + command
existing semantic component + new validation
existing pattern + target-specific projection
```

組合せ爆発を避けるため、C0〜C4のconstraintとretrievalで早期に枝刈りします。

### C6 — local semantic encoder / reranker

必要ならsmall embedding / cross-encoder / reranker等をローカルで利用し、候補順序を改善します。

これはC0〜C5で得た候補をauthority化せず並べ替える補助手段です。

### C7 — local LLM

ここまでで残った小さい候補空間に対し、LLMの強みを使います。

- intentの曖昧な部分の解釈
- A/B/C案の比較説明
- 異なる案の統合
- unresolved assumptionの抽出
- counterexample / missing concernの仮説
- minimum semantic deltaの作成
- custom implementationが不可避な部分のコード候補

### C8 — optional remote LLM

local LLMでは解けないcaseに限り、policy上許可される場合にremote modelへescalateできます。

remote LLMは必須依存にしません。

## 5. Proposal Space Contraction

本構想で重要なのはcandidate generationよりも**candidate contraction**です。

LLMへ次のような問いを丸投げしません。

> この業務をどうモデル化すればよいですか。

代わりに、ローカル層で次の形へ近づけます。

```text
目的:
  受注明細を持つ受注登録を設計する

既知:
  Customerは既存
  OrderId規則は既存
  master datasourceはread-only
  quantity > 0
  Web Targetではheader-detail対応可能

候補:
  A. Order - OrderLine composition
  B. flat OrderRow
  C. Order + external line capability

除外:
  B: line identity / uniqueness / update semanticsを保持できない
  C: current business requirementに外部ownership根拠なし

残候補:
  A

未解決:
  lineNumberをbusiness identityとするか単なるpresentation orderとするか
```

この状態で初めてLLMに比較・質問生成・proposal deltaを依頼します。

### 単一scoreへ畳まない

候補は一つの「AI score」だけで選びません。

少なくとも次を別軸で保持します。

- business coverage
- exact constraint satisfaction
- reuse / precedent
- semantic simplicity
- cross-view consistency
- implementation derivation leverage
- target / provider feasibility
- unresolved assumption count
- novelty / unfamiliarity
- evidence strength

上位数案をPareto-like frontierとして残し、人間またはLLMがtrade-offを確認できる構造を優先します。

## 6. LLMへ渡すContext Packet

local LLMを実用化するには、大規模Repositoryを毎回promptへ入れるのではなく、前段認知から**bounded context packet**を作ることが重要です。

例:

```text
DesignContextPacket
  purpose
  base_revision
  target_scope
  exact_facts[]
  existing_assets[]
  candidate_patterns[]
  hard_constraints[]
  candidate_designs[]
  eliminated_candidates[]
    + reason
  unresolved_dimensions[]
  evidence_refs[]
  authority_limits
  requested_output_schema
```

local LLMの性能不足を、無制限contextと自由推論で補おうとしません。

**前段で問題を小さく、明示的に、構造化すること自体をlocal LLM実用化戦略の中心**にします。

## 7. Local-completeを第一級の動作形態にする

TEIのDesign Synthesis Loopは、少なくとも一つの実用Referenceについて、外部AI serviceなしで成立できることを目標にします。

```text
local deterministic core
+ local indexes / cognition
+ local encoder if needed
+ local LLM
+ local compiler / generator / test runner
```

remote AIは性能向上または難問処理のoptional escalationです。

### model名をArchitectureへ固定しない

local LLMは急速に変化するため、model familyやparameter countをProduct Requirementへ固定しません。

代わりにtask capabilityで評価します。

```text
L1: classify / extract / normalize
L2: compare bounded design candidates
L3: propose structured semantic delta
L4: reason over cross-view contradictions
L5: generate constrained custom implementation / tests
```

各taskについてlocal modelで十分かをReference Evidenceで判定します。

### local LLMに要求する境界

- structured outputを返せる
- exact refsを勝手に作らない
- unknown / unresolvedを返せる
- current revisionをinputとして持つ
- authority limitsを守る
- proposalとaccepted meaningを区別する
- model差し替えでCanonical schemaを変えない

## 8. 実装とテストをほぼderivedへ寄せる

上流designがacceptedされた後も、LLMによる自由コード生成を標準経路にはしません。

優先順位は次です。

```text
accepted semantic definition
        ↓
existing reusable component / pattern
        ↓
declarative rule / binding
        ↓
deterministic generator / compiler
        ↓
constrained local LLM generation
        ↓
custom handwritten implementation
```

これはRQ-015のknowledge-first順序を、設計から実装まで連続させるものです。

### test derivation

テストも可能な限りaccepted meaningから導出します。

例:

- Validation -> positive / negative cases
- Relation cardinality -> presence / multiplicity cases
- unique constraint -> duplicate rejection
- Capability outcomes -> outcome scenarios
- Authorization -> permitted / denied scenarios
- Application Pattern -> interaction conformance
- Target Capability -> supported / unsupported projection
- migration -> before / after schema expectation

ただし実装コードからだけテストを生成すると、同じ誤りをなぞる可能性があります。

**test oracleは可能な限りaccepted semantics / independent Evidenceへ置き、implementationから独立させます。**

## 9. Human-in-the-loopを「入力作業」から「判断」へ移す

目標はhuman-free developmentではありません。

人間の操作を、次のような大量authoringから減らします。

```text
現状型
requirementsを書く
modelを書く
screenを書く
DB定義を書く
codeを書く
testを書く
レビューする
不整合を直す

目標型
purpose / exceptional meaningを示す
↓
候補案と差分を見る
↓
残った選択肢を選ぶ / 修正する
↓
impactを見る
↓
Accept
↓
生成・実行されたEvidenceを見る
```

人間の入力が少ないこと自体ではなく、**一度入力した意味が多くのartifactへ安全に波及するderivation leverage**を重視します。

## 10. Requirement Draft

以下は正式RQ番号ではなく、議論用の`DSR-*`です。

### DSR-01 Multiple Design Proposal — Must

上流のpurpose / partial factsに対し、一つの即答ではなく、意味の異なる複数のstructured design candidateを保持できること。

候補は自然言語説明だけでなく、参照するCanonical identity、assumption、unresolved、predicted impactへ追跡できること。

### DSR-02 Progressive Proposal-Space Contraction — Must

既存定義、constraint、Reference、cross-view relation、Target / Provider capability等を使い、候補空間を段階的に削減できること。

LLMの主観的rankingだけで候補を除外しないこと。

### DSR-03 Explainable Elimination — Must

除外された候補について、少なくともhard constraint violation、missing Evidence、scope mismatch、unsupported capability等の理由へ戻れること。

「AI scoreが低い」だけを除外理由にしないこと。

### DSR-04 Preserve Real Alternatives — Must

複数の妥当案が残る場合、早すぎるsingle-best化を避け、trade-offと未解決点を保持できること。

### DSR-05 Local Pre-LLM Cognition — Must

LLM呼び出し前に、exact / graph / constraint / pattern / associative / cross-view cognitionをローカルで実行し、bounded candidate spaceとEvidence packetを構成できること。

### DSR-06 Bounded LLM Context — Must

LLMへRepository全体を無制限に渡すことを標準とせず、current taskに必要なexact fact、candidate、constraint、Evidence、unresolvedを構造化して渡せること。

### DSR-07 LLM as Synthesis, not Authority — Must

LLMは候補比較、統合、説明、gap hypothesis、semantic delta提案を担えるが、Canonical Fact、Acceptance、Execution Authorityを所有しないこと。

### DSR-08 Local LLM First-Class Support — Should

少なくとも一つのrealistic Reference Scenarioについて、local LLMを使ったproposal / comparison / semantic delta生成が実用的に成立することを目標にする。

model familyやparameter countではなく、task success、latency、memory、quality、reproducibilityをEvidenceとして判断すること。

### DSR-09 Remote LLM is Optional Escalation — Must

remote LLMは難問時のoptional providerであり、TEIの基本的なDesign Synthesis Loopを成立させる必須authority / dependencyにしないこと。

### DSR-10 Semantic-to-Implementation Derivation — Must

accepted semantic designから、既存component、binding、generator、compilerを優先してimplementation artifactを導出できること。

custom code generation / handwritten codeはderivationできない残余へ限定する方向を目指すこと。

### DSR-11 Semantic-to-Test Derivation — Must

Validation、Relation、Capability Outcome、Authorization、Pattern等のaccepted meaningから、test / fixture / conformance scenarioを導出できること。

implementationそのものだけをtest oracleにしないこと。

### DSR-12 Cross-View Constraint Feedback — Should

Data、Function / Interaction、Business Procedure等の他設計要素を、後から文書整合性チェックするだけでなく、candidate designの生成・除外・質問生成へ利用できること。

### DSR-13 Minimum Irreducible Authoring — Must

人間にauthorさせる内容を、機械的に安全に再生成できない業務意味・選択・例外へ寄せ、standard CRUD、projection、binding、test等の二重入力を減らすこと。

### DSR-14 Structured Unresolved State — Must

判断材料が不足する場合、LLMやsolverが勝手に補完せず、unresolved dimension / unknown / required human choiceをfirst-classに保持できること。

### DSR-15 Feedback Closure — Must

生成されたimplementation / test / runtime / simulationの結果を、visible symptomではなく意味所有層へroutingし、次のcandidate designへ反映できること。

### DSR-16 Manual-Work Reduction Evidence — Should

「AIを使った」ことではなく、Reference Scenarioにおいて次を測定可能にすること。

- human-authored semantic decisions
- human-authored implementation lines / artifacts
- human-authored test cases
- manual correction回数
- proposalからAcceptanceまでのhuman choices
- generated artifactの再現率
- unresolvedを正しく残した割合

固定KPIはReference前に決めないこと。

## 11. Derived Cognitionとの責務分離

本書の`DSR-*`はProduct Goalです。

`DERIVED_COGNITIVE_REQUIREMENT_DRAFT_2026-09.md`の`DCR-*`は、そのうちcandidate retrieval / memory safety / lightweight cognitionを支える下位要件です。

```text
DSR
  design candidate generation
  contraction
  local LLM synthesis
  semantic-to-code/test derivation
        ↓ uses
DCR
  recall
  structural similarity
  novelty
  local cognition
  provenance
  stale safety
```

DCR benchmarkが良くてもDSRの価値が自動的に成立するわけではありません。

逆に、DSRが価値を示してもFlyHash等の特定方式を採用する必要はありません。

## 12. 最初に作るReference Scenario

feature atomやlocal model比較へ進む前に、まずProduct Scenarioを作ります。

ScenarioごとのAcceptance詳細とGate C前後の評価境界は[`DESIGN_SYNTHESIS_SCENARIO_ACCEPTANCE_DRAFT_2026-09.md`](DESIGN_SYNTHESIS_SCENARIO_ACCEPTANCE_DRAFT_2026-09.md)を参照する。

### Scenario S1 — 上流モデリングの複数案提案

入力:

- partial business intent
- existing customer / order系資産
- いくつかのhard business rules

期待:

- 2〜4個の意味的に異なる設計候補
- exact existing asset reuse候補
- assumption / unresolvedの明示
- unsupported案の説明可能な除外

### Scenario S2 — 他設計要素による縮約

S1の候補へ、Business Procedure、Target、Datasource、Validation等の追加情報を順に与えます。

期待:

- 候補空間が再評価される
- unrelatedな設計を全再生成しない
- 除外理由が追跡できる
- 本当に複数解なら複数のまま残る

### Scenario S3 — local LLM synthesis

C0〜C6で候補を縮約したContext Packetをlocal LLMへ渡します。

期待:

- 上位候補を比較できる
- unresolvedだけを質問またはproposalとして返す
- exact refsを捏造しない
- structured semantic deltaを返せる

### Scenario S4 — implementation / test derivation

accepted candidateから、standard UI / persistence / capability binding / testを生成します。

期待:

- 人が同じ意味をコードとtestへ再入力しない
- generated implementationとtest oracleの根拠がaccepted meaningへ戻る
- custom implementation residualが明示される

### Scenario S5 — feedback closure

生成物を実行・Simulationし、矛盾または不足を一つ観測します。

期待:

- feedbackが正しいmeaning-owning layerへ戻る
- next proposalはminimum semantic deltaになる
- derived artifactsを直接authority化しない

## 13. 直近の開発順

このRequirement Draftを起点とするpre-Gate-Cのread-only workは、Acceptance素案、S1 semantic candidate diversity、R1〜R6 cross-view / provenance / freshness pressureまで進んでいる。

完了済みのpre-Gate-C範囲:

```text
S1〜S5 Acceptance draft
  -> S1 candidate meaning / discovery provenance pressure
  -> S1 semantic candidate diversity test-only pressure
  -> S2 R1〜R6
       R1 order-entry proof
       R2 actuality / Handoff negative pressure
       R3 bounded context authority boundary
       R4 candidate-local contraction
       R5 discovery provenance
       R6 snapshot / stale invalidation
```

ここまでを理由に、Gate C前にproduction `DesignCandidate`、`DesignContextPacket`、proof lifecycle、Pattern Memory、LLM provider contract、vector / graph storage等を先行固定しない。

現在の通常の待ち条件は次である。

```text
Product Gate B Ready
  -> Handover Baseline B / revision R
  -> First Web Target build T1
  -> Product Gate C Evidence
```

Gate C後の最初のactual sequence:

```text
1. realistic S1 intentからcandidate frontierを作る
2. S2でincremental cross-view contractionを行う
3. actual information lossを測ってbounded contextを調整する
4. ISSUE-000047のdeterministic qualificationへ接続する
5. 必要ならISSUE-000051でpre-LLM retrieval方式を比較する
6. S3 local LLMでminimum semantic deltaを比較する
7. Shared Acceptance経由でsmall semantic changeを1件Acceptする
8. S4 implementation / test derivationを行う
9. S5 observable feedbackをmeaning ownerへ戻し、closed loopを1周する
10. このactual Evidenceからのみproduction schema / provider / learned cognitionを再検討する
```

GitHub-hosted runnerのpre-run blocker等によりtest-only pressureのactual execution Evidenceが得られていない場合、source-level EvidenceをA2やProduct Acceptanceへ読み替えない。

この順序により、モデル選定やvector index実装から開始しません。

まず「人間の設計・実装・テスト作業をどのようにsemantic decisionへ圧縮するか」をactual Referenceで確認します。
