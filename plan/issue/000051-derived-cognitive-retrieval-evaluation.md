# ISSUE-000051: 非LLM認知方式をSemantic探索の候補生成器として比較する

- 状態: Blocked — Product Gate C Ready待ち
- 提案日: 2026-09-10
- 性格: Architecture probe / Reference evaluation
- 主担当要求: RQ-014 / RQ-015 / RQ-016 / RQ-017 / RQ-021 / RQ-022
- 関連: [ISSUE-000047](000047-knowledge-first-semantic-exploration.md) / [ISSUE-000040](000040-semantic-component-composition.md) / [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)

## このIssueを最初に読むときの要点

このIssueは、FlyHashやHDC/VSAをTEIへ導入するためのIssueではありません。

確かめたいのは、**LLMを呼ぶ前に、既存Canonical構造から「次に確認すべき候補」を軽量・ローカル・再生成可能な方法で十分に絞れるか**です。

```text
Canonical / Applicationの既知構造
        ↓
軽量なcandidate retrieval
        ↓
少数の候補
        ↓
ISSUE-000047でdeterministic qualification
        ↓
必要ならISSUE-000027のProposal / LLMへ
```

したがって評価の中心は「どのアルゴリズムが賢いか」ではありません。

```text
候補を見つける性能
      +
誤った候補をauthorityへ昇格させない境界
      +
CPU / memory / rebuild cost
      +
交換・削除可能性
```

を合わせて見ます。

### このIssueが減らす不確実性

主に次の問いへ答えます。

1. exact / graph / lexicalだけでは拾いにくい構造類似やpartial cueを、非LLM方式で有用に回収できるか。
2. LLMや外部AI serviceを毎回呼ばなくても、候補空間を十分に縮められるか。
3. lexical near-match、true novel、stale representation等で、もっともらしい誤候補をsuccess扱いしない評価ができるか。
4. FlyHash、weighted set、HDC/VSA等を同じdeterministic feature corpusで比較し、方式固有の入力差を性能差と取り違えずに済むか。
5. derived cognitive representationを交換・削除・再生成してもCanonical Worldを変えずに済むか。
6. local LLM等の重い認知へ進む前段として、CPU中心の軽量処理が実用的なlatency / memory / rebuild costに収まるか。

### 完了すると何を判断できるか

このIssueが完了すると、Reference Evidenceに基づいて候補生成方式を**採用 / 限定採用 / 不採用**のいずれかへ判断できます。

採用する場合でも、言えるのは「この方式をderived candidate mechanismとして使う価値がある」までです。

次を意味しません。

- similarityがSemantic applicabilityを証明した
- novelty scoreが`unknown`を決定した
- Cognitive IndexがCanonical source of truthになった
- LLMが不要になった
- 特定algorithm名をCanonical vocabularyへ追加すべきである

適用可能性は[ISSUE-000047](000047-knowledge-first-semantic-exploration.md)、Proposal / Acceptanceは[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)が別に担当します。

## 目的

TEIのknowledge-first探索で、LLMやembeddingだけへ依存せず、Canonical Worldから導出した構造特徴を使う軽量な認知方式が、既存Semantic資産の**候補生成・注意制御・新奇性の示唆**に有効かをReferenceで比較します。

本IssueはFlyHash等の特定アルゴリズムをTEIへ採用するためのIssueではありません。外部研究から得た着想を、TEI自身のReferenceとConformanceで評価し、採用・限定採用・不採用を判断するためのarchitecture probeです。

最も重要な境界は次です。

```text
activation / similarity / novelty
            !=
Semantic applicability / Evidence / Decision / Authority
```

認知方式が返すのは候補です。既存資産が現在のintentへ適用可能かは、`ISSUE-000047`が要求するCanonical identity、dependency、scope、constraint、Source、Evidence、current accepted authority等の決定論的な照合で判定します。

## 背景

### ハエ嗅覚系由来の高次元疎表現

Dasgupta, Stevens, Navlakha (Science, 2017) は、ショウジョウバエ嗅覚回路を類似検索の計算問題として捉え、従来の低次元LSHとは異なる、疎なランダム射影による高次元展開とsparsificationを組み合わせた方式を示しました。

- DOI: <https://doi.org/10.1126/science.aam9868>
- PubMed: <https://pubmed.ncbi.nlm.nih.gov/29123069/>

この研究からTEIへそのまま移植するのは「FlyHash」という名称ではなく、次の計算上の仮説です。

```text
small explicit feature set
        ↓
expanded distributed representation
        ↓
sparse activation
        ↓
small candidate neighborhood
```

### data-dependentな疎表現

Ryali et al. (ICML 2020) のBioHashは、FlyHashのランダム射影を固定前提とせず、局所的な学習則によりdata-dependentな疎高次元表現を得る方式を提示しています。

- PMLR: <https://proceedings.mlr.press/v119/ryali20a.html>

このためTEIでは、random projectionをarchitecture contractにしません。Referenceでは固定seedのdata-independent方式から始め、必要なら学習可能な方式を比較します。

### associative continual learning

Shen, Dasgupta, Navlakha (Neural Computation, 2023) は、疎な高次元表現と、活動した表現から対象outputへの局所的associationだけを更新する構成が、継続学習時の干渉低減に有効であることを理論・実験の両面から検討しています。

- DOI: <https://doi.org/10.1162/neco_a_01615>

TEIで直ちに学習機構を導入する根拠にはしませんが、将来Derived Experience Knowledgeを扱う場合に、全履歴や全表現を書き換えるのではなく、**task-local / outcome-localなassociation更新**を比較する根拠になります。

### HDC / VSA

Hyperdimensional Computing / Vector Symbolic Architecture (HDC/VSA) は、高次元distributed representationに対してbinding、bundling、permutation等を用い、roleや順序を保った構造表現を扱えます。Kleyko et al. のreviewは、graphを含むdata structureのdistributed representationを整理しています。

- Open review article: <https://pmc.ncbi.nlm.nih.gov/articles/PMC10588678/>

2025年のConfiGRはHDCによるgraph representationを構造・taskに応じて構成可能にする必要性を示し、2026年のdirected weighted graph研究ではhard-routing型の階層分類がflat modelより悪化する例も報告されています。

- ConfiGR: <https://doi.org/10.1016/j.artint.2025.104384>
- directed weighted graph study: <https://doi.org/10.1186/s13040-026-00561-1>

TEIにとっての示唆は、Canonical graphをvectorへ置き換えることではありません。**exact graphを正本として保持しつつ、role-awareな曖昧検索や構造類似の候補生成へHDC/VSAを使えるか**を比較することです。

## TEIで固定する仮説と固定しないもの

### 固定する仮説

Referenceでは次を検証します。

1. LLMなしでも、既存Canonical featureから有用な候補集合を高速に作れる可能性がある。
2. lexical similarityだけでは拾いにくい、名前の異なる同型・近接構造を回収できる可能性がある。
3. similarityが高くてもSemanticに不適合なnear-matchを決定論的qualificationで落とせる必要がある。
4. 「既存類例が見つからない」を無理にnearest resultへ変換せず、低activation / low margin等を注意信号として保持できる必要がある。
5. 一つのglobal similarity spaceへ全意味を畳まず、task-specificな特徴集合・candidate fieldを比較できる必要がある。

### 固定しないもの

- FlyHash / DenseFly / BioHash / HDC / VSAの採用
- embeddingやvector databaseの排除
- 一つのembedding / hashをRepository全体で共有すること
- model dimension、sparsity、projection density、distance metric
- learned projectionやonline learning
- `internal/cognitive`等のpackage名
- Cognitive LayerをCanonical architectureの新しい正本層として追加すること
- similarity scoreをconfidence / probability / Evidenceと呼ぶこと
- repository-wideなglobal revision counterを認知indexやM1 authority freshnessの共通正本にすること

## Architecture上の位置

初回Referenceでは、認知indexをSource Index等と同様の**derived / regeneratable / build-input-bound resource**として扱います。

```text
Authoritative Source
        ↓ reload
Canonical World
        │
        ├── exact index / dependency projection
        ├── Source Index
        └── derived cognitive representation
                    │
                    ↓
             candidate retrieval
                    │
                    ↓
      deterministic semantic qualification
        applicable / incompatible / unknown
                    │
                    ↓
         explicit-reference candidate
                    │
                    ↓
        Shared Acceptance Boundary
```

次を不変条件とします。

```text
Sparse Code       != Canonical Identity
Activation        != Semantic Fact
Similarity        != Applicability
Novelty Signal    != Unknown
Association       != Dependency
Candidate Rank    != Recommendation authority
Learned Memory    != Decision history
Cognitive Index   != Canonical source of truth
```

indexを削除してもCanonical Worldは変化してはいけません。index自身のstale判定は、実際にその表現を作ったfeature corpus / extractor / model / seed等のbuild inputへ束縛します。これらの入力が変わった場合はstaleを判定できるか、安全に再生成できる必要があります。

このderived-index freshnessは、R6で扱ったM1 accepted authority / proof freshnessとは別の問題です。

```text
derived index freshness
  -> indexを再生成すべきか
  -> build-input fingerprint / scoped snapshotへ戻る

M1 qualification / proof freshness
  -> current semantic claimを根拠付けられるか
  -> required accepted authorityのsemantic fingerprintへ戻る
```

したがって、認知indexのstale検出を理由にrepository全体のglobal revision counterを導入したり、indexのbase revision一致をSemantic applicabilityのEvidenceへ格上げしたりしません。

## 一つのglobal similarityへ畳まない

TEIはData、Function / Interaction、Businessを同じaccepted meaningへの異なるviewとしてcross-checkします。またSource、Evidence、Capability、Target等も意味上の役割が異なります。

そのため初回probeでは、全subjectを一つの意味距離へ畳む方式だけを評価しません。少なくとも次の二案を比較可能にします。

```text
A. one representation / multiple feature groups

B. task-specific candidate fields
   - reuse-discovery
   - structural-analogy
   - missing-concern
   - interaction-pattern
   - capability-context
```

Bでも最初に一つのfieldへhard routeしません。複数fieldからtop-k候補を保持し、後段のqualificationへ渡します。上流分類の誤りを下流へ固定伝播させないためです。

## Phase 0: deterministic feature corpus

最初に認知アルゴリズムを実装するのではなく、同じReference subjectから比較可能なfeature corpusを作ります。

候補featureは次です。

- subject kind
- qualified logical identityの構成要素。ただしidentityそのものをsimilarityの唯一根拠にはしない
- Type / Attribute role
- Relation source / target / cardinality / traversal role
- Validation / Constraintの種類と対象
- lifecycle / mutability
- Query / Datasource relation
- Capability requirement / Binding presence
- Application Pattern内のrole
- Business Action / Procedureとの参照関係がReferenceで成立している場合のcross-view relation
- Source / Evidenceの存在状態
- Application / scope

file path、生成物の一時的な配置、secret、connection string、runtime credentialはfeatureへ入れません。

feature extractionは、同じfeature-extraction input snapshot / fingerprintから同じ結果になることを必須にします。ここで必要なのはreproducible build inputであり、Repository全体の単一global revision counterではありません。

### 現行コードから抽出できるfeature sourceの棚卸し

2026-09-10時点の実装を見ると、Phase 0の最初のcorpusは新しいCanonical schemaを追加せず構成できます。ただし、既存`index.Build`の出力だけを認知入力の正本にせず、必要なrole情報はnormalized `ir.World`のCanonical objectから抽出します。

現行`internal/semantic/model.go`からは、少なくとも次をlosslessに観測できます。

```text
Namespace
  ├─ Type(kind, underlying)
  ├─ Data
  │   ├─ Attribute(type-ref)
  │   └─ Relation(target, cardinality, ordered relation keys)
  ├─ Validation(kind, attribute, optional relation)
  └─ Capability(inputs, output, named outcomes)
```

現行`internal/application/model.go`からは次を観測できます。

```text
Application
  ├─ semantic uses
  ├─ validation references
  ├─ database requirements(alias, access, required contracts)
  └─ Pattern
       ├─ kind
       ├─ data / relation / query / capability references
       ├─ datasource role
       └─ contract-query binding
```

また`internal/index/index.go`には、既に次の再生成可能なderived projectionがあります。

- Semantic elementのkind / Canonical ID / namespace / parent
- semantic dependencyのfrom / relation / to
- ApplicationやDatabaseが宣言するdependency
- ApplicationとDatabaseのedge / access
- QueryのApplication / Datasource / dialect
- Environment Binding、Capability Binding、Persistence Mapping等のrealization側index

`index.Build`は主要entryを明示的にsortしているため、Phase 0のdeterministic corpus構築では、この既存の安定化方法を再利用候補にできます。ただし`Derived`全体をそのまま一つのfeature vectorへserializeしません。Semantic、Application intent、Realization / Environmentの境界を保ったままfeature familyへ分けます。

### 初回feature contract候補

最初のprobeではfeatureを少なくとも次の四群へ分けます。

#### F0 — locator / identity

候補をCanonicalへ戻すための情報です。similarityの主要特徴にはしません。

- subject kind
- Canonical ID
- namespace
- parent Canonical ID
- Application / Database等のownership identity

`F0`は検索結果のidentityとtraceabilityを保つために必要ですが、ID文字列が似ていることをSemantic similarityの根拠へ格上げしません。

#### F1 — intrinsic semantic structure

対象自身の意味構造です。初回の主要feature familyとします。

- Type kind / underlying relation
- AttributeのTypeRef
- Relationのtarget / cardinality / key role・key数
- Validation kind / target Attribute / Relation有無
- Capabilityのinput signature / output / named Outcome structure

値を単なるbagへ落とす方式と、`relation.target`、`capability.input`のようにroleを保持する方式を比較できる形で抽出します。

#### F2 — application-use context

同じSemantic資産がどのApplication文脈でどう使われるかを表します。

- SemanticUses / Validation reference
- Pattern kind
- PatternからData / Relation / Query / Capabilityへのrole付き参照
- Datasourceの存在とApplication-local role
- Database access mode / required Contract
- Contract Query input binding

Application-local aliasの文字列そのものはdomain meaningと同一視せず、原則として「aliasを介したdatasource roleがある」という構造を優先します。

#### F3 — derived structural neighborhood

既存`SemanticDependencies` / `DeclaredDependencies`等から、対象の近傍構造を再生成します。

```text
subject
  --relation-role--> target-kind
  <--relation-role-- source-kind
```

この群はweighted set比較、FlyHash系の高次元展開、HDC/VSAのrole bindingを同じ入力意味から比較する主要材料になります。

### Realization / Evidenceは初回similarityへ無条件に混ぜない

現行indexにはProvider、Environment Binding、Permission、Resource、Persistence Mapping、physical table、custom adapter / hook等もあります。これらは適用可能性のqualificationや`capability-context`の専用fieldには重要ですが、初回のgeneral semantic similarityへ無条件に混ぜません。

```text
Canonical / Application meaning
          !=
Provider / Environment / physical realization
```

たとえば同じ業務意味がPostgreSQLから別Providerへ移っただけで別Semantic Patternと認識される設計は避けます。一方、「この候補は現在のEnvironmentでは利用できない」という判断にはBindingやCapability条件が必要です。したがってRealization情報は、**候補発見の意味距離**と**候補適用のqualification context**を分けて扱います。

Source locator、file path、physical table名、adapter filename、hook nameも同様です。必要なtraceabilityには保持しますが、名前の偶然な一致をmeaning similarityへ持ち込みません。

### Phase 0の最小出力形

実装時のschemaは未確定ですが、benchmarkを比較可能にする論理出力は次程度で足ります。

```text
FeatureDocument
  subject: Canonical reference
  build_input: feature corpus / source-model fingerprint or scoped snapshot ref
  groups:
    identity:    F0
    intrinsic:   F1
    application: F2
    structural:  F3
```

`build_input`はproduction fieldを先取りする意図ではありません。derived representationを同じ意味入力から再構築したかを比較するためのconceptual traceであり、M1 semantic freshnessやAcceptance revisionの代替ではありません。

各アルゴリズムは同じ`FeatureDocument`から必要な群だけを選んでencoderへ渡します。アルゴリズム固有のpreprocessing結果をfeature sourceの正本にはしません。

この分離により、後からFlyHashをVSAへ替えても、評価対象が「別の意味入力へ変わったため良く見えた」のか「encoderが良かった」のかを区別しやすくします。

## Phase 1: baseline比較

最初の比較では、外部AI serviceやGPUを必須にしません。CPU上で再現可能な方式を使います。

少なくとも次を比較候補にします。

1. exact / inverted index
2. weighted set similarity（Jaccard等）
3. SimHash系の低次元hash
4. FlyHash型のexpand + sparse k-WTA
5. DenseFly型の高次元hash
6. role-awareなHDC/VSA encodingの小さいprobe

BioHash等のlearned representationはPhase 1の必須対象にしません。

各方式は同じcorpus、同じquery case、同じcandidate poolで比較します。方式固有の前処理が必要な場合はEvidenceへ記録します。

## Phase 2: TEI特有のnegative case

一般的なnearest-neighbor benchmarkではなく、TEIで危険な誤りを中心に検査します。

### same meaning / source move

file pathだけ変更しCanonical identityとmeaningが同じ場合、認知結果が意味上の別物として大きく変わらないことを確認します。

### lexical near-match / semantic incompatible

名前や語彙が近いが、scope、Datasource、Capability、Validation、Target条件等が合わない資産を検索上位へ意図的に置きます。認知層が候補へ入れても構いませんが、qualificationは`incompatible`にできなければなりません。

### structurally similar / lexically different

名称が異なってもRelation / Pattern / Constraint構造が近い資産を用意し、set similarity、FlyHash系、VSA系の回収差を比較します。

### partial cue

Featureの一部を欠落させたintent / editing contextから、既存資産をcandidate setへ回収できるかを確認します。

### true novel / no good match

実際に適切な既存資産がないcaseを作り、必ず何かを「最も近い候補」としてsuccess扱いしないことを確認します。low activation、top-1/top-k margin等はheuristic signalとして記録できますが、`unknown`判定の代替にはしません。

### stale representation

認知indexが実際に依存するfeature input / extractor / model等のbuild inputを変更した後に旧indexを読むnegative caseを作り、stale検出または安全な再生成を確認します。無関係なaccepted changeまで全indexをstale化することは要件にしません。

## 評価指標

単一のaccuracyだけで採否を決めません。少なくとも次を記録します。

- Recall@k / relevant candidate inclusion
- candidate set size
- incompatible near-match混入率
- partial cue recovery
- novel caseでのfalse-familiar rate
- seed感度
- feature ablation感度
- index build / incremental update cost
- p50 / p95 retrieval latency
- memory footprint
- deterministic rebuildの可否
- stale detection / build-input traceability
- qualification後に`applicable / incompatible / unknown`がどのように変化したか

最終的なProduct価値は「hash精度」ではなく、既存意味の再利用漏れを減らし、不適合候補をauthorityへ昇格させず、開発者またはAIが調べる範囲を小さくできるかで判断します。

## Phase 3: Derived Experience Knowledgeは別判定

Phase 1/2で静的な候補生成に価値が確認できるまで、online learningを導入しません。

将来評価する場合も、Acceptance結果を単純なrewardへ変換しません。少なくとも次を区別します。

```text
semantic-reuse-success
incompatible
unknown / insufficient evidence
stale-derived-representation
policy-reject
acceptance-failure
ignored / no decision
```

学習対象にするoutcomeと、単に監査へ残すoutcomeを分けます。Decision historyそのものは既存の明示履歴が正であり、association weightへ置き換えません。

局所更新を試す場合は、活動したfeature / candidate fieldに関係するassociationだけを変更し、無関係なfieldやCanonical feature extractorへ副作用を与えないことを最初の安全条件にします。

## LLMとの関係

本IssueはLLM代替を目的にしません。

理想的な探索cascadeの仮説は次です。

```text
exact identity / symbol lookup
          ↓
structural / graph lookup
          ↓
lightweight associative retrieval
          ↓
optional embedding retrieval
          ↓
optional LLM interpretation / composition
```

実際には各段を必ず直列実行する必要はありません。重要なのは、簡単な候補探索まで毎回LLMへ委譲することをarchitecture requirementにせず、LLMもSparse / VSA等も非権威的なcandidate mechanismとして扱えることです。

LLMがintent解釈や説明、新規compositionを行っても、`ISSUE-000027`のShared Acceptance Boundaryを迂回しません。

## ISSUE-000047 / 000040との責務分離

- `ISSUE-000047`はknowledge-first read-only探索と`applicable / incompatible / unknown`へのqualificationのProduct Referenceを担当します。
- `ISSUE-000040`は既存意味資産を第二Applicationへ明示参照で組み立てる根幹機能を担当します。
- 本Issueは、その探索前段に使える**candidate retrieval方式の比較評価**だけを担当します。
- `ISSUE-000027`はProposalからauthoritative sourceへ進むShared Acceptance Boundaryを担当します。

本Issueが不採用になっても000047 / 000040はexact / graph / lexical等の別方式で成立できなければなりません。逆に本Issueのbenchmarkが良くても、000047のqualificationや000027のAcceptanceを省略しません。

## 着手条件

本格着手はProduct Gate C Ready後です。

ただし、Gate C前に次のread-only設計準備は可能です。

- Referenceで利用可能なCanonical feature候補の棚卸し
- benchmark caseの設計
- 外部研究の追跡
- deterministic feature extractionの境界検討

Gate B / Gate Cの完了条件へ本Issueを追加しません。第一Web Targetの実装をこのprobeのために遅らせません。

## 完了条件

- [ ] Gate C Ready後のReference input snapshot / fingerprintを固定した。
- [ ] deterministic feature corpusをCanonical Worldから再生成できる。
- [ ] exact / set similarityを含むbaselineを用意した。
- [ ] FlyHash型のsparse expansive方式を少なくとも一つ比較した。
- [ ] role-awareなstructural encodingを少なくとも一つ比較するか、比較しない理由をEvidenceとして残した。
- [ ] same-meaning source move、lexical near-match、structural near-match、partial cue、true novel、stale representationのnegative / pressure caseを実行した。
- [ ] similarityやactivationをSemantic Evidenceへ昇格させず、000047のqualification結果を別に記録した。
- [ ] index削除・再生成後もCanonical Worldが変わらないことを確認した。
- [ ] latency / memory / rebuild costを含む運用コストを記録した。
- [ ] 採用、限定採用、不採用のいずれかをReference Evidenceから判断した。
- [ ] 採用する場合もalgorithm名をCanonical vocabularyへ追加せず、交換可能なderived mechanismとして扱った。
- [ ] learned associationへ進む場合は別Issue / ADRの必要性を判断した。

## 非目標

- Gate C前の実装
- LLMの置換
- vector databaseの排除
- TEI専用の巨大Knowledge Graph
- 全Repository文書やGit履歴の一括embedding
- similarity scoreによる自動採用
- novelty scoreによる自動`unknown`判定
- online learningの先行実装
- AI / cognitive subsystemへのruntime authority付与
- cognitive representationをCanonical identity、Decision history、Evidenceへ転用すること
- derived indexのstale管理を根拠にrepository-wide global revision counterを導入すること

## 判断をADRへ昇格させる条件

現段階ではADRを作りません。

次のいずれかがReferenceから必要になった場合にだけ、長期判断をADRへ切り出します。

- derived cognitive indexに共通して守るbuild-input / regeneration contractをArchitectureとして固定する必要がある
- 複数のcandidate mechanismを交換可能にする共通interfaceが複数Issueにまたがって必要になる
- learned associationの永続化、更新、忘却、rollbackに長期のauthority境界が必要になる
- Cognitive Layerという独立architecture layerを`docs/architecture.md`へ追加するEvidenceが得られる

それまでは「評価待ちの候補方式」であり、現在のArchitecture baselineを変更しません。

## 関連文書

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/architecture.md`
- `docs/business-system-design.md`
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `plan/DESIGN_SYNTHESIS_SNAPSHOT_INVALIDATION_PRESSURE_2026-09.md`
- `plan/design/README.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000040-semantic-component-composition.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`