# Definition-Centered Feedback Loop

Status: current design direction / no new Canonical kind / no new Gate

## 1. 目的

TEIは、生成AIが大量の設計書やコードを生成すること自体を価値の中心にしません。

中心に置くのは、**人間にも機械にも読める端的な定義体をauthoritative meaningとして保ち、その定義から実行・可視化・検査・探索を豊かに再生成し、現実から得たfeedbackを正しい意味所有層へ戻せること**です。

```text
small authoritative definitions
        ↓ normalize / validate
Canonical meaning
        ↓ derive / compile / project
UI / matrix / graph / runtime / simulator / index
        ↓ use / execute / observe
human review / runtime result / external actuality
        ↓
feedback
        ↓ route to owning concern
candidate change
        ↓ preview / qualify
Shared Acceptance Boundary
        ↓ accepted minimum semantic delta
authoritative source
        ↓ reload / regenerate
        ↺
```

このloopは自己書換え型のAI systemを意味しません。AI、認知index、Studio、Runtimeはauthorityではなく、定義体を理解・検証・改善するための異なる器官です。

## 2. X-TEA / XEADから受け取るもの

TEIはX-TEA / XEADとの互換製品を目指しません。一方、過去のReference Modelと三要素分析法から、現代でも有効な設計原理を抽出します。

### 継承する原理

- 一つの業務システムをData / Function-Interaction / Businessの異なる視点から相互検査する
- Data-firstであってData-onlyにはしない
- 業務行為とApplication Functionを同一視しない
- 人・手作業・外部組織・物理世界を含む社会システムとして業務を扱う
- 仕様を可能な範囲で実行・Simulationし、文書だけで正しさを主張しない
- Reference Modelを単なるfixtureではなく、PlatformとModelを相互に改善する実践コンテンツとして使う
- model上のcross-referenceから矛盾を発見し、元の定義へ戻して直す

### そのまま継承しないもの

- 歴史的なobject種類やfield構造
- 全視点を対称なtop-level objectにすること
- monolithicなDriver
- UI / Runtime / Modelerの密結合
- 旧製品のfile formatをTEIのCanonical形式にすること
- Reference Model中のpatternを自動的な正解として採用すること

TEIでは「外部モデルにfieldがある」ことではなく、**そのfieldや視点がどの設計失敗を防いでいたか**を問い直します。

## 3. Definition-centeredという意味

Definition-centeredは「すべてをTEI DSLへ書き直す」という意味ではありません。

Authoritative sourceは concern ごとに異なり得ます。

```text
TEI-native business / semantic meaning -> .tei等のTEI-native source
native query meaning                  -> SQL
existing implementation meaning       -> Go / existing asset + explicit binding
provider/environment fact              -> provider-owned configuration / observation
```

重要なのは、同じ意味について暗黙の複数正本を作らず、どのsourceが何をauthorしているか説明できることです。

### 3.1 irreducible meaningだけをauthorする

人が手で維持するのは、機械的に安全に再生成できない意味に限定します。

```text
author explicitly
  business identity / relation / constraint / responsibility / exceptional intent
  provider or target choice when it is genuinely a choice
  manual / external work when it is business meaning

prefer derived
  standard CRUD
  ordinary list/detail/form projection
  cross-reference matrix
  dependency graph
  reverse index
  impact view
  search index
  cognitive representation
  generated target code
```

「derivedにできる」ことは、情報を捨てることではありません。元定義と生成規則からいつでも説明・再構成できることを要求します。

### 3.2 progressive disclosure

最初から完全なmodelを要求しません。

```text
Data / Relationで業務事実を表せる
  ↓ 必要な場合だけ
explicit Action / Capability / Validation
  ↓ 実現差が意味を持つ場合だけ
Application Pattern / Binding / Realization
  ↓ 責任・手作業・外部継続が必要な場合だけ
Business Procedure / Handoff
  ↓ domainで必要な場合だけ
time / decision / evidence detail
```

追加する概念は「空欄を埋めるため」ではなく、追加しないとmeaning lossが起きる時だけ増やします。

### 3.3 semantic densityとfalse compressionを区別する

短い定義は目的ですが、短さ自体をKPIにはしません。

```text
high semantic density
  = 少量のauthoringから、多くの正しいderived behaviorを再構成できる

false compression
  = 必要なBusiness meaningを暗黙推論へ追い出し、短く見せる
```

manual work、responsibility、actuality、exception等を削って定義を短くすることは品質向上ではありません。

## 4. 三種類のmemoryを分ける

生成AI・脳科学時代のTEIでは、すべてのknowledgeを一つのKnowledge Graphやmodel weightへ押し込みません。

### 4.1 authoritative semantic memory

長期に保持すべきaccepted meaningです。

- authoritative source
- Canonical identity / relation / constraint
- accepted Business meaning
- explicit Binding / Realization choice
- accepted Decision / policy where required

これは削除するとプロダクトの意味が失われます。

### 4.2 derived associative / structural memory

「何を思い出すか」「何が近いか」を助けます。

- exact index / dependency index
- lexical index
- graph neighborhood
- Reference Model pattern index
- embedding
- sparse high-dimensional representation
- HDC / VSA等のrole-aware representation
- candidate ranking / novelty signal

これは**再生成可能で、revision-bound**でなければなりません。

```text
association != dependency
similarity  != applicability
activation  != fact
novelty     != unknown
```

このmemoryを削除してもCanonical meaningは変わりません。

### 4.3 episodic / experience memory

「何を試し、どう判断し、何が起きたか」を保持します。

- Proposal / rejection / acceptance record
- Semantic Diff / impact preview
- simulator / runtime run
- Observation / Evidence
- review comment / rationale
- past correction trajectory

過去に成功した変更も、そのまま現在のmeaningにはなりません。current revision / Source / Evidenceへ戻って再評価します。

### 4.4 memory間のauthority rule

```text
authoritative semantic memory
        ↑ accepted change only
Shared Acceptance Boundary
        ↑
proposal / experience / cognition

associative memory  -X-> implicit semantic write
episodic memory     -X-> implicit semantic write
model weights       -X-> only durable record of a Decision
```

AIや認知方式を交換・停止しても、accepted systemを再構成できることを不変条件とします。

## 5. 認知機能を一つのAIへ集約しない

TEIの認知は、性質の違う複数方式を協調させます。

```text
Intent / local editing context
        │
        ├─ exact identity / graph traversal
        ├─ Reference pattern recall
        ├─ associative / sparse cognition
        └─ LLM interpretation / hypothesis / explanation
                ↓
           candidate space
                ↓
 deterministic qualification
 applicable / incompatible / unknown
                ↓
       explicit-reference proposal
```

### Exact / graph

既知のidentity、dependency、authorityを確定的に辿ります。既知のことを毎回LLMに再推論させません。

### Reference pattern recall

実践Referenceから観測された構造を、provenanceと適用条件つきの候補として提示します。patternは答えではありません。

### Associative cognition

語彙が違うが構造が近い例、partial cue、局所的なnoveltyを候補化します。FlyHash、HDC/VSA、embedding等はこの役割の比較候補です。

### LLM

自然言語intentの解釈、複数候補の比較、欠落仮説、説明、compositionに強みを使います。LLMにSource / Evidenceを与えても、その出力自体をauthorityにはしません。

## 6. Feedbackを正しい所有層へroutingする

高品質なfeedback loopでは、見つかった問題を何でもSemantic sourceへ返しません。

| Feedback | 主な戻り先 | 戻してはいけない先の例 |
| --- | --- | --- |
| Business fact / relation / constraintの欠落 | Semantic / Business authoring source | UIだけの補修 |
| Business responsibility / manual workの欠落 | Business Procedure source | fake Application Function |
| Target固有のinteraction問題 | Presentation / Target Adapter | domain Dataの歪曲 |
| Provider固有の実現問題 | Binding / Realization / Provider | Canonical business meaning |
| DB actual drift | Observation / Mapping / Migration candidate | Semantic Relationへの自動逆書き |
| Runtime failure | Capability / Execution / Resource境界 | LLM promptだけの調整 |
| 検索取りこぼし / ranking問題 | derived index / cognition | Canonical relationの捏造 |
| Reference patternが合わない | candidate ranking / applicability rule | Reference patternの自動採用 |

原則は次です。

> visible symptomではなく、meaningを所有する最小の層を直す。

これにより「UIを直してDataの誤りを隠す」「AI promptを調整してauthority矛盾を隠す」といった局所最適を防ぎます。

## 7. Minimum Semantic Delta

生成AI時代には、全modelやfileを毎回再生成するより、**現在のaccepted meaningに対する最小の意味差分をproposalすること**を標準にします。

```text
base revision
   + purpose / observed problem
   + exact target identities
   + minimum proposed meaning change
   + provenance / evidence
        ↓
preview
   Semantic Diff
   affected subjects
   derived UI / runtime / persistence impact
   unresolved / contradiction
        ↓
Accept / Reject
```

Minimum Semantic Deltaは固定Patch DSLを意味しません。file-native edit、Studio action、structured AI proposal、refactor等が同じ意味差分へ収束できればよいとします。

### full regenerationを標準にしない理由

- 人間reviewで本質的な変更点が見えにくくなる
- model固有のformatやcommentを不要に書き換えやすい
- stale conflictの範囲が広がる
- AIが未要求の意味まで補完しやすい
- accepted meaningとgenerated wordingが混ざる

ただし新規artifact生成や大規模migrationなど、whole-artifact generationが合理的な作業まで禁止しません。重要なのはsemantic diffを局所化してreviewできることです。

## 8. Previewは生成物ではなく因果を見せる

Authoring UXでは、入力formを増やすより、**一つの定義変更が何へ波及するか**を見せることを重視します。

例:

```text
Add required Relation
  ├─ Canonical: relation-added
  ├─ Function: Action input requirement changes
  ├─ UI: reference selector appears
  ├─ Persistence: migration required
  ├─ Runtime: referential integrity enforced
  ├─ Business: impacted Procedure candidates
  └─ Target: affected projections
```

各edgeはderived ruleまたはexact dependencyへ追跡できる必要があります。LLMの説明だけでimpactを作りません。

## 9. Local-first Editor

大規模modelを常に一枚のgraphへ描くことを要求しません。

利用者が現在扱う定義を中心に、局所近傍を高密度に見せます。

```text
selected definition
  ├─ authored local meaning
  ├─ exact incoming / outgoing refs
  ├─ Data / Function / Business lenses
  ├─ relevant source / evidence
  ├─ affected projections
  ├─ candidate patterns / analogous cases
  └─ unresolved / diagnostics
```

全体graphやmatrixは必要に応じたProjectionです。検索・scope・local neighborhoodにより、数百・数千definitionでも認知負荷を制御します。

AIも同じlocal contextを基本入力とし、Repository全体を毎回自由探索して暗黙知化することを標準にしません。

## 10. Reference Modelを知的社会資本として扱う

Reference Modelには複数の役割があります。

1. **worked example** — 利用者が良い定義の形を学ぶ
2. **executable design content** — Driver / Simulatorで動かして理解する
3. **Conformance pressure** — toy fixtureでは出ない設計欠陥を露出させる
4. **pattern evidence corpus** — AI / cognitionが候補構造を想起する根拠
5. **product feedback source** — Referenceの要求がPlatformの過不足を明らかにする

重要なのは5です。

```text
Platform -> Referenceを実行可能にする
Reference -> Platformの不足・過剰を暴く
```

この双方向性がないReferenceは、単なるdemoやgolden fixtureへ縮退します。

### Reference patternの扱い

Reference群から共通patternを抽出することは有用ですが、patternは次の形で利用します。

```text
intent
  ↓
multiple candidate patterns
  + provenance
  + observed frequency / examples
  + applicability conditions
  + semantic differences
  + optional Simulation Dataset
  ↓
qualification / review
```

「8個のReferenceにあるから採用」ではなく、現在のBusiness requirementへ合うかを別途確認します。

## 11. Observationとlearningを分ける

Runtime / simulator / external worldからのObservationはEvidenceです。Observationそのものをonline learningで即時にsemantic writeへ変換しません。

```text
expected meaning
        ↕ compare
observed actuality
        ↓
contradiction / drift / unknown
        ↓
proposal or routing decision
        ↓
Shared Acceptance Boundary
```

### 学習可能なのはまずderived memory

Referenceや運用から得た経験で自動更新を検討する場合、最初に対象にするのは再生成可能な候補生成・ranking側です。

- task-local association weight
- frequently useful candidate pattern
- false-positive suppression
- local retrieval hint

それでも revision / task / outcome / provenance を持ち、無効化可能にします。

accepted semantic ruleをmodel weightだけへ埋め込むonline learningは標準経路にしません。

## 12. Feedback qualityの観点

現時点で数値KPIへ固定しませんが、設計reviewでは次を観察します。

### semantic density

少量のauthoringでどれだけ正しい意味・Projectionを再構成できるか。ただしfalse compressionを避ける。

### derivation leverage

同じ内容の人手二重管理なしに、UI / Runtime / DB / matrix / docs等へどれだけ正しく派生できるか。

### feedback closure

見つかった矛盾を、その意味を所有するsource / layerへ戻して直せるか。

### reconstructability

index、generated artifact、AI historyを削除してもaccepted systemを再構成できるか。

### locality

局所definitionと近傍だけで安全な変更判断が可能か。

### provenance

candidate / pattern / AI explanation / experienceが、Source / revision / Evidenceへ戻れるか。

これらは比較・review観点であり、達成率を一つのscoreへ畳みません。

## 13. Product invariants

今後のAI・認知機能追加でも次を守ります。

```text
Definition != Projection
Projection != Authority
Observation != Definition
Similarity != Applicability
Proposal != Decision
Autonomy != Authority
Reference Pattern != Requirement
Runtime Success != Business Completion
Generated Artifact != Source of Truth
```

さらに、次のdegradationを許容します。

```text
LLM unavailable
  -> exact / graph / manual authoringでsystemを扱える

associative cognition unavailable
  -> exact / graph / lexical探索へ退化できる

Studio unavailable
  -> authoritative sourceを直接review / reloadできる

Derived index deleted
  -> Canonicalから再生成できる

Target removed
  -> Canonical business meaningは残る
```

このdegradabilityは、AIや特定UIを新しい不可視の正本にしないための設計検査です。

## 14. 現時点で固定しないもの

- permanentな`Cognitive Layer`というArchitecture層
- 一つのuniversal Knowledge Graph
- vector database
- FlyHash / HDC / VSA / embeddingの特定方式
- online learning
- Proposal / Patchのplatform-wide schema
- feedback categoryの永続enum
- AI provider / agent framework
- whole-model graph UI
- Reference patternの自動採用
- Reference Model file format

具体的Referenceで意味が失われる場合だけ固定化をreopenします。

## 15. 関連

- `docs/architecture.md`
- `docs/business-system-design.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000051-derived-cognitive-retrieval-evaluation.md`
- `plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`
