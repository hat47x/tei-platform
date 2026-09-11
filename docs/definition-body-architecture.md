# TEI Concise Definition Body Architecture

Status: **Design baseline — concise authority / rich derived intelligence**

関連:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/business-system-design.md`
- `docs/product-quality-architecture.md`
- `plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`
- `plan/CONCISE_DEFINITION_BODY_REFERENCE_PRESSURE_2026-09.md`
- GitHub Issue #570

## 1. 目的

TEIのauthoring sourceを、機械が必要とする中間情報やAI向けmetadataで肥大化させず、**業務上の意味を区別するために不可欠な記述へ集中させる**ためのArchitectureを定めます。

目標は最小LOCや短いDSLではありません。

```text
small authored authority
        +
rich deterministic reconstruction
        +
rich derived projections / intelligence
        =
compact but expressive executable design
```

TEIは、X-TEA / XEAD系Reference Modelが示してきた次の生命を、歴史的schemaをそのまま複製せず継承します。

- Dataを安定した設計核に置く
- FunctionをUI・CRUD・read・call等とのcross-referenceから検査する
- Businessを人・手作業・契機・責任を含む社会システムとして扱う
- 三つの視点を独立した正本へせず相互に牽制する
- 実用的Reference contentからPlatform側の不足を見つける
- 人間が読める仕様と機械実行を近づける

生成AI・derived cognitionは、この簡潔さを壊す追加metadata sourceではなく、**簡潔なauthorityから再構成された知識を探索・比較する外側の知能**として接続します。

## 2. Definition Bodyとは何か

本書でいう**Definition Body**は、authoritative source上で人が明示的に保持する意味記述の本体です。

一つの固定file formatや一種類の`.tei` blockを指しません。

対象に応じて:

- Semantic Data / Relation / Constraint
- Application intent / Pattern / explicit View / Capability use
- Database Contract / native source reference
- Business Procedure上のmanual / external meaning
- policy choice / exception / unresolved

等がDefinition Bodyになり得ます。

一方、次は保存されていてもDefinition Bodyのauthorityにはしません。

- generated CRUD / inspection View
- Function / CRUD matrix
- graph / impact index
- query analysis result
- physical schema observation
- Runtime bundle
- UI component tree
- Evidence artifact
- telemetry
- embedding / FlyHash / HDC representation
- search ranking
- AI trajectory / conversation state

## 3. 最重要原則 — Irreducible Meaning Rule

ある記述をDefinition Bodyへ入れるか迷った場合、まず次を問います。

> **この記述を削除したとき、materially differentなaccepted business meaningを区別できなくなるか、または安全で決定論的な再構成が不可能になるか。**

YESならauthoring候補です。

NOで、accepted sourceと明示されたrule/contextから再構成できるならderived候補です。

### 3.1 例

#### authorする

- `Order`と`OrderLine`の1:N Relation
- customerとのRelationとbusiness key mapping
- read-only datasourceという意図
- `submit order`というdomain-specific Capability
- standard CRUDでは表せないApplication interaction intent
- 人が電話確認するBusiness Action
- external organizationへのHandoff
- domain-specific validation / policy choice
- unresolvedであること自体が重要なknowledge boundary

#### 通常はderiveする

- Dataから一意に作れるinspection list/detail
- Relationから一意に作れるreference selector候補
- Action / View / Queryから計算できるread/CUD cross-reference
- Data / Function / Business cross-view matrix
- source-to-subject impact graph
- cognition feature vector
- search index

#### realization / contextへ置く

- PostgreSQL column type mapping
- Web widget / DOM / CSS layout
- transport / session / Host-specific detail
- Provider connection material

#### observation / Evidenceへ置く

- actual schema snapshot
- latency result
- runtime trace
- migration observation
- Target screenshot

## 4. Explicit Difference / Derived Default

標準的で、一意に再構成できる内容を人が繰り返しauthorしません。

```text
accepted Definition Body
        ↓
standard / deterministic derivation
        ↓
common projection / behavior
```

差があるときだけ明示します。

```text
standard derived behavior
        +
explicit domain difference
        ↓
accepted effective behavior
```

この方針は「Convention over Configuration」と似ていますが、TEIではさらに厳しく、**convention自体が説明可能・versioned・reconstructableであること**を要求します。

暗黙規則が増え、人間にも機械にも「なぜこうなったか」が分からない状態は簡潔ではなくhidden complexityです。

## 5. Semantic Information Conservation

Definition Bodyを短くしても、意味上必要な区別を失ってはいけません。

```text
concise source
  != lossy source
```

次のような圧縮は禁止します。

- `string`へ落とすことでdate / instant / identifier / codeの差を失う
- Relationを単なるID fieldとして扱いcardinality / referential meaningを失う
- manual / external Business Actionを架空のApplication Functionへ変換する
- Business actorをAuthorization Roleへ畳む
- Business ResultとObservationを同一視する
- unknown / unresolvedをdefault値で埋める
- distinct policy choicesを一つの「標準」に押し込む

端的さは、**意味を捨てることではなく、意味から機械的に再構成できる重複を捨てること**です。

## 6. Progressive Disclosure

通常の設計者が、内部Architectureの全概念を毎回知る必要はありません。

```text
simple requirement
  -> small Definition Body

new semantic pressure
  -> only then expose the necessary concept
```

高度な概念は、次のtriggerがあるときだけauthoring surfaceへ現れます。

- standard derivationでは意味が区別できない
- safe executionにexplicit choiceが必要
- external / legacy assetを保持する必要がある
- Target / Provider差ではなくbusiness meaningとして差がある
- Referenceで実際の設計欠陥が再現された

内部実装が複雑であることは、通常authoringを複雑にする理由になりません。

## 7. Five-plane separation

簡潔なDefinition Bodyと豊かな実行・知能を両立するため、責務を概念上5 planeへ分けます。

```text
1. Authoritative Definition Body
      ↓
2. Deterministic Semantic Reconstruction
      ↓
3. Derived Lenses / Knowledge
      ↓
4. Realization / Observation / Evidence
      ↓
5. Cognitive Candidate Layer
```

実装packageを必ず5個に分けるという意味ではありません。

### 7.1 Authoritative Definition Body

人が採用した業務意味と明示選択を保持します。

### 7.2 Deterministic Semantic Reconstruction

parse / normalize / resolve / validate等により、accepted meaningを再構成します。

同じsource + same contract/version/contextから同じmeaningへ戻れることを重視します。

### 7.3 Derived Lenses / Knowledge

Canonical / accepted meaningを別のreview目的へProjectionします。

例:

- Data lens
- Function / Interaction lens
- Business lens
- CRUD/read/call matrix
- graph / flow
- Database Workbench
- impact / diagnostic index
- standard inspection UI
- Pattern candidate inventory

これらは豊かでよいですが第二正本にはしません。

### 7.4 Realization / Observation / Evidence

Provider / Target / Runtime固有実装とactual stateを扱います。

Desired meaningとActual observationを混ぜません。

### 7.5 Cognitive Candidate Layer

exact / graph retrieval、associative cognition、LLM reasoning等が候補を作ります。

```text
candidate
  != fact
  != applicability
  != decision
  != authority
```

## 8. Functionはnounよりlensを優先する

業務システムのFunction視点は必要ですが、独立`Function` objectが常に必要とは限りません。

Functionとして確認したい情報が:

- View / interaction
- Action / Capability
- Data read
- Data CUD
- function / action call
- input / output

から再構成できる場合、別のFunction正本を人に二重管理させません。

```text
Action / View / Query / Capability
          +
read / CUD / call edges
          ↓
derived Function lens
```

Referenceが、このlensでは保持できないFunction固有meaningを示した場合だけ新しいauthoring contractを検討します。

## 9. Viewはinspectionとapplication intentを分ける

Dataから大量のinspection Viewをderiveできても、それをend-user application navigationの意図へ昇格させません。

```text
Data
  -> derived inspection views

explicit user/task intent
  -> authored Application View / Pattern where necessary
```

これにより、設計者は全Dataごとにlist/detail/formを手書きせずに済みますが、実際の利用目的は必要な箇所で明示できます。

UI component tree、DOM、pixel layoutはさらにTarget側へ置きます。

## 10. Business meaningは圧縮で消さない

Data-firstはData-onlyではありません。

次は、実際の業務に存在するならDefinition Body側へ残す価値があります。

- purpose / trigger
- human / organizational responsibility
- manual instruction
- external work
- business judgment
- Handoff / continuation condition
- business acceptance
- exception / unresolved

これらを「Runtimeで実行できないから」という理由で削除しません。

また、すべてを構造化fieldへすることも要求しません。Referenceでcross-check・trace・acceptanceに必要な最小構造だけを採用します。

## 11. Native assetをDefinition Bodyへ複製しない

SQL、Go、external configuration等がauthoritativeなnative assetである場合、TEI利用のためだけに同じ内容を別DSLへ転記しません。

TEI側には必要なidentity、role、contract、binding、traceを持ちます。

```text
TEI accepted meaning
    -> native asset reference / binding
    -> realization
```

native contentの全コピーをDefinition Bodyへ作ることは、簡潔化ではなく二重管理です。

## 12. Derivationは説明可能でなければならない

Definition Bodyから何かをderiveした場合、少なくともconceptually次を説明できる必要があります。

```text
derived item
  <- source subject(s)
  <- derivation contract / rule
  <- relevant version / profile
  <- context if context-dependent
```

このtraceが失われると、derived resultが事実上の第二正本になります。

### 12.1 No convention soup

短いsourceを実現するために、名前の類似、file order、偶然のfirst match、現在のUI位置等からmeaningを推測しません。

critical resolutionはexplicit identity / exact-one / validated relationを優先します。

## 13. Derived defaultのversion変化

標準derivation自体もsoftwareです。

同じDefinition Bodyでもderivation contractが変わればProjectionやgenerated artifactが変わる可能性があります。

したがって重要なEvidenceでは:

- repository revision
- derivation implementation / contract identity
- generated artifact
- Target build

等へ戻れる必要があります。

ただし、このためだけに各sourceへversion fieldを大量追加しません。Repository revisionやpackage/runtime contract等、既存のversioning境界を優先します。

## 14. AIのためにDefinition Bodyを肥大化させない

AIが検索・理解しやすいという理由だけで、次をmandatory authoringにしません。

- embedding-oriented description
- prompt hints
- synonyms list
- generated summaries
- retrieval tags
- feature vectors
- nearest-neighbor labels
- model-specific annotations
- trajectory metadata

business meaningとして本当に必要なdescription / vocabulary / rationaleがあればauthorできます。

判断基準は「AIに便利か」ではなくIrreducible Meaning Ruleです。

## 15. Derived cognitionの正しい位置

脳科学着想のassociative cognitionを導入する場合も、Definition Bodyは入力featureの手作業台帳にはしません。

```text
accepted Definition Body
      ↓ deterministic semantic/index projection
algorithm-neutral features
      ↓
exact / graph / FlyHash / HDC / embedding / other
      ↓
candidate identities
      ↓
deterministic qualification
      ↓
Proposal / exploration
```

feature corpusは可能な限り既存meaningからderiveします。

認知方式を交換・削除してもCanonical Worldとaccepted sourceは不変でなければなりません。

## 16. LLMの役割

LLMに期待するのは、隠されたsemantic defaultの補完ではありません。

価値が高いのは:

- natural-language intentの整理
- existing asset candidateの説明
-複数Pattern候補の比較
- cross-view contradictionの仮説提示
- example Dataset / Scenario候補
- accepted meaningへの変更Proposal
- unresolved questionの表面化

です。

AIは「最適な定義を自由生成する」より、既存Reference / Pattern / Evidenceから比較可能な候補を提示し、Shared Acceptance Boundaryへ返します。

## 17. Reference ModelをKnowledgeとして使う

Referenceはtest fixtureだけではありません。

realistic Referenceから次をderiveできます。

- recurring relation / lifecycle patterns
- Application interaction patterns
- consequence patterns
- Business Handoff patterns
- failure / recovery examples
- Example Dataset / Scenario
- applicability / incompatibility conditions

これらをPattern Card等のderived Knowledgeへ索引化できます。

ただしReference内で頻出することだけを理由に、全domainの標準semantic truthへ昇格しません。

## 18. Candidate提示の形

生成AI・cognition時代の設計支援では、「一つの答え」より比較可能な候補が重要です。

```text
intent / current context
      ↓
existing knowledge retrieval
      ↓
Candidate A / B / C
  - structure
  - source / Reference evidence
  - applicability
  - incompatibility / disadvantage
  - unresolved
  - example scenario
      ↓
Human / policy selection
      ↓
Proposal
      ↓
Shared Acceptance Boundary
```

candidate rankingは採用判断そのものではありません。

## 19. Cold Reconstruction Principle

Derived artifactが第二正本になっていないことを最も強く確認する方法の一つは、derived stateを捨てて再構成することです。

```text
remove:
  generated indexes
  inspection projections
  search caches
  embeddings
  FlyHash / HDC representation
  AI trajectory / summaries

keep:
  authoritative source
  native authoritative assets
  versioned implementation / contracts

reload
  ↓
Canonical / accepted meaning
  ↓
core derived lenses
```

これが成立しないderived mechanismは、authorityを漏らしている可能性があります。

AI serviceが無くてもCanonical meaningをreloadできなければなりません。

## 20. Local Change Stays Local

端的なDefinition Bodyはdiff品質にもつながります。

局所的な意味変更で:

- unrelated filesの全面再serialization
- generated fieldの大量materialization
- comment / orderの不要な移動
- derived metadataのsource書戻し

を起こしません。

AI / Studioによる編集も、人が手書きした場合と同じShared Acceptance / local-diff原則へ収束します。

## 21. Complexity Rent Test

新しいkeyword、top-level kind、mandatory field、authoring file、manual mappingを追加する前に次を問います。

1. realistic Referenceで再現できるdefectは何か。
2. 既存meaning / Relation / Projection / Binding / Evidenceではなぜ表せないか。
3. 追加後、何をdeterministically検証できるようになるか。
4. authoring負担と二重管理はどれだけ増えるか。
5. derived representationではなぜ不足か。
6. 複数Target / Providerでもbusiness meaningとして残るか。
7. その概念を削除した場合に失われるmaterial distinctionは何か。

「将来使うかもしれない」は採用理由になりません。

## 22. Definition Body review classification

設計reviewでは、各情報を次のどこに置くべきか判定できます。

| Class | Authority | 例 |
| --- | --- | --- |
| A — irreducible accepted meaning | authored / native authoritative source | Data, Relation, domain rule, manual work, explicit interaction intent |
| B — deterministic derived meaning/lens | derived | CRUD matrix, inspection View, impact graph |
| C — realization / environment | Binding / Provider / Target | DB mapping, widget, transport |
| D — actual observation / Evidence | observation | schema snapshot, run artifact, latency |
| E — cognitive candidate | non-authoritative derived | similarity, Pattern candidate, AI Proposal |

情報が複数roleを持つ場合も、同じ内容を無条件に複数正本へコピーせずtraceで接続します。

## 23. Product Qualityとの接続

`docs/product-quality-architecture.md`のQuality Envelopeは、Definition Bodyを増やさずquality claimへcontextを付ける仕組みです。

```text
Definition Body
  -> meaning
  -> Target / Provider realization
  -> measurement / Evidence
  -> Quality Envelope
```

performance threshold、accessibility result、runtime telemetry等をbusiness sourceへ逆流させません。

## 24. 新しいRequirementを増やさない理由

このArchitectureは主に既存Requirementの組合せです。

- RQ-001 / 003 — meaningとauthority
- RQ-002 — source trace
- RQ-004 — native asset
- RQ-010 / 011 — Editor / Projection
- RQ-014 — Pattern reuse
- RQ-015 / 016 — AI exploration / Proposal
- RQ-017 — 二重管理最小化
- RQ-021 — Reference / Conformance
- RQ-023 — maintainability
- RQ-025 / 026 — Business meaning / cross-view

Referenceで既存RQでは防げないmaterial defectが確認されるまで、新しい番号付きRequirementは追加しません。

## 25. Gateとの関係

Gate B / Gate Cのscopeは変更しません。

Gate C前に行うのはArchitectureとReference pressureの設計までです。

```text
Gate B actual Evidence
  ↓
Handover / First Target / Gate C
  ↓
Studio / knowledge-first exploration
  ↓
concise-definition pressure / cognition comparison
```

現行`order_entry`等から静的に得られる知見は利用できますが、それをGate C後のactual Product Evidenceへ読み替えません。

## 26. 先に固定しないもの

- 新しい`Definition` Canonical kind
- universal shorthand syntax
- auto-CRUDの永久標準一覧
- complete Function taxonomy
- complete Pattern Card schema
- mandatory authored descriptions / tags
- AI-specific metadata schema
- embedding / FlyHash / HDCの採用
- universal derivation profile field
- source compression score
- authored line-count threshold

Reference / Evidenceが必要性を示す前に固定しません。
