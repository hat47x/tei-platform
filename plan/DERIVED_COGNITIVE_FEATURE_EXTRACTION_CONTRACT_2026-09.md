# Derived Cognitive Feature Extraction Contract — 2026-09

## 位置づけ

この文書は、[ISSUE-000051](issue/000051-derived-cognitive-retrieval-evaluation.md)のPhase 0で、Canonical Worldから各candidate retrieval方式へ渡す**algorithm-neutralなfeature入力**をどう構成するか整理する評価契約です。

状態: **評価待ち / Gate C前の設計準備**

関連するbenchmark caseは[`DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md`](DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md)を正とします。

この文書は新しいCanonical schema、`internal/cognitive`等のpackage、永続index形式、FlyHash / HDC/VSAの採用を決めません。実装時の型名も固定しません。

目的は、次の比較を可能にすることです。

```text
same Canonical meaning
       ↓
same algorithm-neutral feature source
       ↓
encoder A / encoder B / encoder C
       ↓
retrieval qualityを比較
```

encoderごとに都合のよい意味入力へ作り替えると、アルゴリズム差とfeature engineering差を区別できなくなるためです。

## 原則

### Canonicalから一方向に導出する

feature corpusはaccepted sourceをreloadした`ir.World`と、そこから再生成可能なderived projectionから構成します。

```text
Authoritative Source
      ↓ reload
Canonical / normalized World
      ↓ deterministic extraction
Feature Corpus
      ↓ encoder
Derived Cognitive Representation
```

Cognitive RepresentationからCanonical meaningへ逆書きしません。

### featureはSemantic Factそのものではない

Canonical factからfeatureを導出できますが、feature tokenやvector componentを新しいSemantic primitiveとして扱いません。

```text
feature atom      != Canonical fact identity
feature weight    != business importance
vector distance   != semantic applicability
```

### 抽出と重み付けを分ける

Phase 0 extractorは、可能な限り意味上のroleをlosslessに観測し、アルゴリズム固有のweight、dimension、projection、distance metricを持ちません。

```text
Feature extraction
       !=
Encoder / weighting
```

weighted Jaccard、FlyHash、HDC/VSA等が異なるweightingを使う場合はalgorithm configurationとしてEvidenceへ残します。

## Base documentとContext overlayを分ける

Semantic assetが別Applicationから利用されたことだけで、資産自身の意味表現を変えないようにします。

例えば`sales.order`が`order_entry`に加えて第二Applicationから参照された場合、次のように扱います。

```text
Subject Base
  sales.order
  ├─ intrinsic semantic structure
  └─ canonical structural neighborhood

Context Overlay
  ├─ application: order_entry
  │    └─ usage / Pattern / Datasource context
  └─ application: second_app
       └─ usage / Pattern / Datasource context
```

新しい利用先が一つ増えただけでBaseのfeature digestが変わる設計は避けます。

この分離は、次の二つの検索を区別するためにも必要です。

```text
「構造が似たDataを探す」
   -> Base中心

「このApplication文脈で再利用しやすいDataを探す」
   -> Base + selected Context Overlay
```

## Algorithm-neutralな論理形

実装型ではありませんが、Phase 0では次の論理構造を想定します。

```text
FeatureSubject
  locator
    subjectKind
    canonicalID
    namespace
    parent

  base
    intrinsicAtoms[]
    structuralAtoms[]

  contexts[]
    contextKind
    contextID
    applicationAtoms[]

  lexical
    lexicalAtoms[]

  trace
    worldRevision
    sourceReferences[]
    extractionContractVersion
```

`trace`は再現性とprovenanceのために保持しますが、原則としてsimilarity featureへ入れません。

## Feature atom

各featureを単なる自由文字列だけで持つのではなく、論理上はroleを区別できる形にします。

```text
FeatureAtom
  group
  role
  valueKind
  value
  multiplicity
```

例:

```text
F1 | type.kind                  | enum          | alias
F1 | attribute.type             | canonical-ref | std.money
F1 | relation.cardinality       | enum          | many
F1 | relation.key.count         | integer       | 1
F2 | pattern.kind               | enum          | reference
F2 | datasource.access          | enum          | read
F3 | outgoing.targets.kind      | subject-kind  | semantic-data
F3 | incoming.used-by.kind      | subject-kind  | application-pattern
```

HDC/VSAは`role`と`value`をbindingできます。set / hash方式では同じatomをstable tokenへ変換できます。したがって同じ意味入力から方式を比較できます。

## F0 — Locator / Identity

F0は候補を正本へ戻すための情報であり、標準runではsimilarityの主要入力にしません。

保持するもの:

- subject kind
- Canonical ID
- namespace
- parent Canonical ID
- ownership identity

F0をsimilarityへ入れたrunは、必要に応じて**lexical / identity leakageのnegative control**として扱います。

Canonical IDの文字列が近いだけで高品質に見える方式を、構造認知が優れているとは評価しません。

## FL — Lexical cueを独立させる

ISSUE-000051のF0〜F3とは別に、benchmark実装ではidentifier / symbol由来の語彙cueを**FL (lexical)**として分離することを推奨します。

理由は、語彙情報をF1/F3へ混ぜると、`structurally similar / lexically different`の評価ができなくなるためです。

Phase 0のFL候補:

- Canonical IDのlocal segment
- sourceで保持されるsymbol
- camelCase / snake_case / kebab-caseを決定論的に分割したtoken

初回は自然言語embedding、形態素解析、同義語辞書を入れません。

例:

```text
order_entry.customer_reference
  -> order / entry / customer / reference
```

ただしFLはlocatorではありません。tokenからCanonical IDを復元したり、同名を同一identityとみなしたりしません。

## F1 — Intrinsic semantic structure

F1は対象自身の構造です。

### Type

最低限:

```text
type.kind
underlying.exists
underlying.ref
underlying.kind (解決可能ならderived shapeとして)
```

`underlying.ref`のexact Canonical referenceと、参照先のshapeを分けて抽出できるようにします。

例えば二つのdomain aliasがともに`std.identifier`をunderlyingに持つ場合、名前が異なってもshape類似を観測できます。

### Data

Data全体をattribute名のbagだけにしません。

候補:

```text
attribute.count
attribute.type.ref × N
attribute.type.shape × N
relation.count
relation.cardinality.one.count
relation.cardinality.many.count
relation.keyed.count
```

attributeの順序がCanonical meaningとして定義されていない場合、source記述順を類似特徴へ利用しません。

### Attribute

候補:

```text
attribute.type.ref
attribute.type.shape
parent.kind = data
```

symbol / identifier tokenはFLへ分離します。

### Relation

最低限roleを保持します。

```text
relation.target.ref
relation.target.kind
relation.cardinality
relation.key.count
relation.key.source-attribute.type
relation.key.target-attribute.type
```

key mappingは順序付きCanonical構造として保持されていますが、比較方式によって順序を使うかはencoder側で選べます。extractorは順序情報を捨てない形で出せるようにします。

### Validation

現行kindごとに構造を保持します。

```text
validation.kind
attribute.ref
attribute.type.shape
relation.exists
relation.ref
relation.cardinality
```

`positive`と`relation-unique`を同じ`validation=true`だけへ潰しません。

### Capability

候補:

```text
input.count
input[i].type.ref
input[i].type.shape
output.exists
output.ref / shape
outcome.count
outcome[i].payload.ref / shape
```

Parameter nameは初回のsemantic structureではなく、必要ならFLまたはrole metadataとして別扱いにします。

## exact referenceとshape referenceを分ける

Canonical referenceは強い情報です。これを全部除くと既存の意味関係を不必要に捨てますが、全部そのまま使うとidentity lookupに近づきすぎます。

そこで参照featureは可能なら二系統を保持します。

```text
exact reference
  relation.target.ref = customer.customer

shape reference
  relation.target.kind = data
  target.attribute-type-profile = {...}
```

benchmarkでは次をablationできます。

```text
shape only
exact-ref only
shape + exact-ref
```

これにより「同じ既存資産を再発見しただけ」と「別identityだが構造が似た資産を発見した」を分けて評価できます。

## F2 — Application Context Overlay

F2はBaseへ直接畳み込まず、Application単位のoverlayとします。

### Application use

候補:

```text
semantic-use exists
validation-reference exists
```

### Pattern

Pattern自身を候補にする場合はそのintrinsic application structureとして扱えますが、Semantic Data等へ付与する場合はContext Overlayです。

候補:

```text
pattern.kind
pattern.data-role exists
pattern.relation-role exists
pattern.query-role exists
pattern.capability-role exists
pattern.datasource-role exists
contract-query.exists
contract-query.input.count
```

参照するCanonical IDそのものと参照先shapeを分けます。

### Datasource / Contract context

候補:

```text
datasource.exists
datasource.access
required-contract.exists
required-contract.count
contract-query.exists
contract-query.input-binding.count
```

Application-local alias文字列そのものはFLへ入れる場合を除き、general semantic similarityへ利用しません。

`master`という名前よりも、`read-only datasource used by reference Pattern`というroleを優先します。

## F3 — Structural Neighborhood

F3はCanonical / derived dependencyをrole付き近傍として表します。

初回は無制限graph traversalを行いません。まずone-hopを基準にします。

```text
subject
  --role--> target-kind / target-shape

subject
  <--role-- source-kind / source-shape
```

例:

```text
sales.order.lines
  --targets--> semantic-data

sales.order.submit
  --input--> semantic-data
  --output--> semantic-data
```

必要ならPhase Bでtwo-hop motifを比較しますが、depthを増やすほど高degree nodeが似て見える問題と計算量が増えるため、Evidenceなしに標準化しません。

## multiplicityを失わない

同じroleのfeatureが複数ある場合、setへ早期変換して個数を失わないようにします。

例:

```text
attribute.type = std.money × 2
```

algorithm-neutral corpusではmultiplicityを保持します。

encoder側では、

- binary presence
- count
- capped count
- normalized frequency

等を比較できます。

## high-degree biasへの対策はencoder側で比較する

DataやApplicationはfeature数が多いため、単純なdot product等では大きいsubjectが何にでも似る可能性があります。

extractorで意味を削って調整するのではなく、encoder / similarity側で次を比較します。

- L1 / L2 normalization
- role単位のnormalization
- count cap
- inverse-frequency weighting
- k-WTA等のsparsification

どの方式を使ったかはalgorithm configuration fingerprintへ残します。

## Source / Provenanceをsimilarityから分離する

Source locator、range、file path等はtraceabilityには必要ですが、F1〜F3へ入れません。

```text
trace.sourceReference
       !=
semantic similarity feature
```

`A-01 source move invariance`では、Repository revisionやSource locatorが変わっても、再load後のmeaningが同じならBase feature digestが同じになることを確認します。

ただし、**revisionが違うのに古いindexをそのままcurrentとして扱ってよい**という意味ではありません。

次の二つを分けます。

```text
featureMeaningDigest
  -> 抽出されたBase / selected Contextの意味featureが同じか

source/world revision
  -> どのCanonical Worldから導出したか
```

source moveだけならrebuild後に`featureMeaningDigest`が同じでもよい一方、旧revisionのindexを無検査で再利用してはいけません。

## Index manifest候補

永続schemaは未確定ですが、比較Evidenceには少なくとも次を残します。

```text
CognitiveIndexManifest
  worldRevision
  featureExtractionContractVersion
  corpusMeaningDigest
  encoderKind
  encoderConfigFingerprint
  seed (if applicable)
  builtAt / toolVersion (補助Evidence)
```

`builtAt`やtool versionはSemantic authorityにはしません。

## 決定論的ordering

同じWorldから同じFeatureSubject列を生成できるようにします。

標準化候補:

1. subject kind
2. Canonical ID
3. feature group
4. role
5. value kind
6. value
7. stable occurrence index

でsortしたlogical serializationからdigestを作ります。

Source file iteration order、map iteration order、filesystem enumeration orderへ依存しません。

## Query extraction

Queryも可能な限り同じFeatureAtom vocabularyを使います。

```text
Corpus FeatureAtom vocabulary
          ↑
          │ shared roles
          ↓
Query FeatureAtom vocabulary
```

これにより、`partial cue`はfeature atomを意図的にmaskするだけで作れます。

ただしqueryにはCanonicalでないintent featureが含まれる場合があります。その場合は、

```text
query-only / non-authoritative
```

と明示し、Canonical featureと区別します。

query-only featureをCanonical Worldへ自動追加しません。

## unknown valueを勝手なdefaultへ変えない

QueryでcardinalityやDatasource条件が不明な場合、`one`や`read`等へ推測補完しません。

```text
missing cue
  !=
default semantic fact
```

retrieval encoderがmissingを無視するか、明示的unknown tokenとして扱うかは比較可能ですが、unknown tokenをCanonical factにはしません。

## candidate family filtering

Candidate familyのkind制約はretrieval前の決定論的filterとして適用できます。

例えばApplication Patternを探していることが確定している場合、DataやCapabilityを同じrank listへ混ぜる必要はありません。

```text
allowed subject kinds
      ↓ deterministic filter
candidate corpus
      ↓ approximate retrieval
```

一方、task自体が複数familyにまたがる場合は一つへhard routeせず、複数familyを並列実行します。

## Privacy / Security / Runtime material

次はgeneral cognitive feature corpusへ含めません。

- secret値
- connection string
- credential
- runtime token
- filesystem上のsecret path
- Resource Brokerから供給されるHost material

Provider / Environment / Permission / Resource Binding等がcandidate qualificationに必要な場合も、専用contextとして最小限に扱い、general reusable semantic patternと混ぜません。

## Phase 0で作らないもの

- learned embedding
- online learning weight
- global vector database schema
- LLM generated feature
- natural-language semantic summaryを唯一のfeature sourceにすること
- Source fileの全文tokenization
- SQL本文の汎用semantic embedding
- arbitrary code ASTの全Repository横断表現

必要性がReferenceで出た時点で別に評価します。

## Benchmarkとの対応

[`DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md`](DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md)のcaseに対し、主に次を使います。

| Case | 主feature / control |
| --- | --- |
| A-01 source move | Base digest + trace分離 |
| A-02 header-detail partial cue | F2 / F3 |
| A-03 reference partial cue | F2 / F3 + datasource context |
| A-04 command partial cue | F1 capability + F2 pattern |
| A-05 validation separation | F1 / F3, FL mask |
| A-06 context conflict | retrieval F1〜F3 + qualification context |
| A-07 true novel | query-only composition + empty positive set |
| A-08 stale | worldRevision / manifest |
| B-01 structural analogue | shape ref + F3, FL mask |
| B-02 lexical near-match | FL vs F1/F3 ablation |
| B-03 cross-app partial cue | Base + multiple Context Overlay |

## Phase 0完了時に確認すること

Gate C前の設計準備として、次が説明可能になればfeature extraction contractは十分とします。

- Canonical sourceから一方向に導出する。
- F0 locator、FL lexical、F1 intrinsic、F2 context、F3 structuralを混同しない。
- Semantic Subject BaseとApplication Context Overlayを分離する。
- exact referenceとshape referenceを比較可能にする。
- roleとmultiplicityをencoder前に失わない。
- Source / revision traceとmeaning feature digestを分ける。
- algorithm weight / seed / dimensionをextractorへ埋め込まない。
- query-only featureをCanonical meaningへ昇格させない。
- secret / runtime materialをgeneral feature corpusへ入れない。

## 実装へ進む条件

Product Gate C Ready後、ISSUE-000051で固定Reference revisionを選び、この契約から最小Feature Corpusをmaterializeします。

最初の実装では全featureを網羅する必要はありません。A-01〜A-08を実行するために必要な最小atomを実装し、benchmarkで不足が観測された場合に拡張します。

もしexact / graph / weighted setだけでPhase Bまで十分なretrieval品質が得られた場合、このFeature Corpus自体は有用でもFlyHash / HDC/VSAは不採用になり得ます。

## 非主張範囲

この文書は次を主張しません。

- `FeatureSubject` / `FeatureAtom`という名称を実装APIとして固定すること
- F0 / FL / F1 / F2 / F3をCanonical vocabularyへ追加すること
- exact referenceをsimilarityへ必ず使うこと
- two-hop以上のgraph featureが必要であること
- Application利用頻度がSemantic importanceであること
- featureMeaningDigestがSource revision checkの代替になること
- Cognitive IndexがCanonical / Source Index / Decision historyを置き換えること

## 関連文書

- [`DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md`](DERIVED_COGNITIVE_BENCHMARK_CONTRACT_2026-09.md)
- [ISSUE-000051](issue/000051-derived-cognitive-retrieval-evaluation.md)
- [ISSUE-000047](issue/000047-knowledge-first-semantic-exploration.md)
- [ISSUE-000040](issue/000040-semantic-component-composition.md)
- [ISSUE-000039](issue/000039-business-change-rehearsal.md)
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/architecture.md`
- `docs/business-system-design.md`
