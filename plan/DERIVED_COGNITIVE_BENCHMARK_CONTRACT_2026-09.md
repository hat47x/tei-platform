# Derived Cognitive Retrieval Benchmark Contract — 2026-09

## 位置づけ

この文書は、[ISSUE-000051](issue/000051-derived-cognitive-retrieval-evaluation.md)で比較する非LLM認知方式について、同じ問題を同じ条件で評価するためのbenchmark contractを整理します。

状態: **評価待ち / Gate C前の設計準備**

この文書はCognitive Layer、FlyHash、HDC/VSA等の採用を意味しません。Product Gate C前にruntime implementationや新しいCanonical schemaを追加する根拠にもなりません。

benchmarkの目的は、特定アルゴリズムのaccuracyを最大化することではなく、次を分離して観測できる問題集合を用意することです。

```text
candidate retrieval quality
        !=
Semantic qualification
        !=
Acceptance / Authority
```

外部研究のbenchmark datasetをTEIへ持ち込むのではなく、TEI自身のReference資産と意味境界からpressure caseを構成します。

## 現行Referenceをanchorにする

2026-09-10時点の`order_entry`には、最初のbenchmark anchorとして次の意味資産があります。

```text
sales.order
  attributes:
    order_id / customer_id / ordered_at / status / total / external_id
  relations:
    customer -> customer.customer [one, keyed]
    lines    -> sales.order_line  [many]

sales.order_line
  attributes:
    order_line_id / line_number / item_code / quantity / unit_price

validations:
  sales.order_line.quantity_positive
  sales.order.lines.line_number_unique

capability:
  sales.order.submit
```

Application側には少なくとも次があります。

```text
order_entry.order_record       record        -> sales.order
order_entry.order_lines        header-detail -> sales.order.lines
order_entry.customer_reference reference     -> sales.order.customer
                                                + read-only master datasource
                                                + customer contract query
order_entry.recent_orders      browse        -> sales.order + query
order_entry.submit_order       command       -> sales.order.submit
```

この構造は、Data / Relation / Validation / Capability / Application Patternを別candidate familyとして試す最小anchorになります。

ただし現行ReferenceのApplicationは`order_entry`一つです。この規模だけで「構造類似検索が実用的」と結論してはいけません。Gate C後に[ISSUE-000040](issue/000040-semantic-component-composition.md)で第二Applicationがacceptedされた場合、その実資産をextended corpusとして共用します。認知方式のbenchmark専用に第二の業務正本を作りません。

## Benchmark corpusとqueryを分ける

benchmarkでは次を混同しません。

```text
Corpus
  = accepted Canonical / Application meaningから作ったFeatureDocument

Query
  = Canonical subject、editing context、intent等から作る検索入力

Perturbation
  = queryまたは観測条件へ加える再現可能な変形
```

原則としてpressure caseのために偽の`.tei`をauthoritative sourceへ追加しません。

### Corpus

Corpusは固定したReference revisionの`ir.World`から決定論的に生成します。F0〜F3のfeature familyはISSUE-000051の定義を使います。

### Query

Queryは次のいずれかから構成できます。

- 既存Canonical subjectそのもの
- 既存subjectから一部featureを除いたpartial cue
- Studio / CLIで観測できるediting contextの構造化表現
- accepted intentから作った非権威的なfeature request

Query自体をCanonical identityやaccepted meaningとして扱いません。

### Perturbation

同じ問題を各方式へ与えるため、perturbationはアルゴリズム固有処理の前に適用します。

初回候補は次です。

```text
source-move
  Source locatorだけを変え、Canonical identity / meaningは変えない

mask-feature(role)
  指定したrole featureをqueryから除く

mask-group(Fn)
  feature familyを一つqueryから除く

lexical-mask
  symbol / identifier由来のlexical cueを検索入力から除く

context-conflict
  scope / datasource / access / capability条件等へ明示的な不整合を加える

novel-composition
  corpusに存在しないfeature組合せをqueryとして与える
```

`lexical-mask`や`novel-composition`で生成するのはqueryです。新しいCanonical objectを作ったことにはしません。

## Candidate familyを先に限定する

全subjectを一つのglobal similarity spaceへ入れた評価を初回の標準にはしません。

最低限、次のcandidate familyを独立して測ります。

| Family | 初回candidate例 | 主に見るfeature |
| --- | --- | --- |
| semantic-data | `sales.order`, `sales.order_line`, `customer.customer` | F1 / F3 |
| relation | `sales.order.customer`, `sales.order.lines` | F1 / F3 |
| validation | quantity positive / line number unique | F1 / F3 |
| capability | `sales.order.submit` | F1 / F2 / F3 |
| application-pattern | record / header-detail / reference / browse / command | F2 / F3 |

candidate familyの選択自体をLLMの非再現な判断だけに任せません。探索taskが複数familyにまたがる場合は複数fieldを並列に実行し、後からensembleできます。

## Gold truthを三層に分ける

各caseの期待結果を単一の`correct=true/false`へ畳みません。

### Retrieval oracle

候補生成器に対する期待です。

```text
must-include
should-include
allowed-near-match
```

`must-include`がtop-kへ入ることは評価できますが、それだけでSemanticに適用可能とはみなしません。

### Qualification oracle

`ISSUE-000047`側の決定論的照合に対する期待です。

```text
applicable
incompatible(reason)
unknown(reason)
no-applicable-candidate
```

Retrieval oracleとQualification oracleが異なるcaseを意図的に含めます。

### Invariance oracle

source moveやderived index再生成に対して、どの情報が不変であるべきかを定義します。

例:

```text
F0 Canonical reference   same
F1 intrinsic             same
F2 application           same
F3 structural            same
Source locator           changed
```

## Phase A — 現行order_entryだけで設計・実行可能なcase

Gate C前はcase contractの確定までとし、Product benchmarkとしての実行・採用判定は行いません。Gate C後に同じcaseを固定revisionでmaterializeします。

### A-01 source move invariance

**目的:** file pathやSource locatorをmeaning similarityへ混ぜていないことを確認する。

Anchor:

```text
sales.order
```

操作:

```text
source-move(reference/workspace/semantic/sales/order.tei -> benchmark-local alternate locator)
```

期待:

- Canonical IDは同じ。
- F1 / F2 / F3は同一。
- Source locatorだけが変わる。
- derived representationを再生成しても、`sales.order`を別Semantic assetとして扱わない。
- source move後のscore完全一致はalgorithm contractにしないが、候補identityや意味上の近傍が不合理に変わる場合は失敗として記録する。

### A-02 header-detail partial cue

**目的:** 全identifierを知らなくても構造cueから既存Patternを候補へ戻せるかを見る。

Anchor:

```text
order_entry.order_lines
```

Query cue:

```text
pattern.kind = header-detail
relation.cardinality = many
child-side structure exists
```

Perturbation:

- Pattern IDを使わない。
- `sales.order.lines`という完全なqualified IDを使わない。

Retrieval oracle:

```text
must-include: order_entry.order_lines
```

Qualification oracle:

```text
applicable only after relation / scope / ownership等をexactに確認
```

このcaseでは「header-detailという語が一致しただけ」を成功理由にしません。F2だけ、F3だけ、F2+F3でablationします。

### A-03 reference partial cue

**目的:** RelationだけでなくDatasource / Contract Query roleを含む複合文脈を回収できるかを見る。

Anchor:

```text
order_entry.customer_reference
```

Query cue:

```text
pattern.kind = reference
relation.cardinality = one
external/read-only datasource context
contract-query exists
single relation-key-derived input binding
```

Retrieval oracle:

```text
must-include: order_entry.customer_reference
```

`order_entry.order_lines`はRelationを使うという点ではnear-matchになり得ますが、Pattern kindやDatasource / Contract Query contextが異なります。そのためcandidateへ入ること自体を即失敗とせず、rank / feature contributionを記録します。

### A-04 command partial cue

**目的:** Capability signatureとApplication Pattern roleを分けて回収できるかを見る。

Anchor:

```text
order_entry.submit_order
sales.order.submit
```

Query cue:

```text
pattern.kind = command
capability input and output refer to same domain Data
```

期待:

- Pattern familyでは`order_entry.submit_order`をmust-includeとする。
- Capability familyでは`sales.order.submit`をmust-includeとする。
- Pattern identityとCapability identityを一つのcandidate typeへ潰さない。

### A-05 validation role separation

**目的:** Validationを単なる「rule」という同一カテゴリに潰さず、target roleを区別できるかを見る。

Anchors:

```text
sales.order_line.quantity_positive
sales.order.lines.line_number_unique
```

Query 1:

```text
validation.kind = positive
attribute type = decimal-like
```

Query 2:

```text
validation.kind = relation-unique
relation cardinality = many
child attribute participates in uniqueness
```

期待:

- Query 1ではquantity positiveをmust-include。
- Query 2ではline number uniqueをmust-include。
- lexical cueをmaskした状態でも、F1 / F3から区別できるかを記録する。

### A-06 context-conflict / near-match

**目的:** retrievalが近い候補を出しても、qualificationが不適合を落とせることを確認する。

Anchor:

```text
order_entry.customer_reference
```

Queryはreference Patternに近い構造を持たせつつ、accepted Applicationとは矛盾するcontextを明示する。

例:

```text
requires write-capable datasource
but candidate datasource `master` is read-only
```

Retrieval oracle:

```text
customer_reference may be retrieved as a near-match
```

Qualification oracle:

```text
incompatible: datasource / access condition mismatch
```

このcaseでcandidate generatorへ「絶対に返すな」と要求すると、retrievalとqualificationの責務が再び混ざるため避けます。

### A-07 true novel / empty positive set

**目的:** nearest candidateが存在することと、再利用可能な候補が存在することを区別する。

Queryとして、現行corpusに存在しない複合条件を作ります。

例:

```text
pattern.kind = reference
relation.cardinality = many
requires write-capable datasource
contract-query has multiple independently bound inputs
```

Retrieval oracle:

```text
must-include: none
nearest candidates may be returned for inspection
```

Qualification oracle:

```text
no-applicable-candidate
```

評価では、top-1が返ったことをsuccessにしません。false-familiar rateの主要caseとします。

### A-08 stale representation

**目的:** derived cognitive representationをCanonical revisionから切り離して使わないことを確認する。

操作:

1. revision R1からcorpus / cognitive indexを生成する。
2. Canonical sourceを意味上変更したR2を用意する。
3. R1のindexをR2 contextで利用しようとする。

期待:

- revision / fingerprint mismatchを検出するか、R2から再生成する。
- stale indexを使ったrankingをSemantic qualificationへ黙って渡さない。
- R1 / R2の差分そのものは`ISSUE-000039`の意味差分機構が利用可能なら再利用し、認知専用diff engineを作らない。

## Phase B — 第二Application accepted後のcross-application case

Phase Aだけではcandidate数も構造多様性も不足します。Product上の採用判断にはPhase Bを必須とします。

第二Applicationは`ISSUE-000040`のReference成果を共用します。benchmarkのためだけに似たApplicationを作らず、実際に別業務としてacceptedされた意味資産を使います。

### B-01 structurally similar / lexically different

第二Applicationに、語彙は異なるがData / Relation / Pattern構造が`order_entry`の一部と近いaccepted assetが存在する場合に実施します。

例として修理受付domainが採用されたなら、`Order` / `OrderLine`という語を持たないheader-detail構造を比較対象にできます。

評価:

- lexical-maskありでも構造類似候補をtop-kへ回収できるか。
- weighted set / FlyHash系 / role-aware VSAで差が出るか。
- 同型に見えてもCapability / datasource / validation差があればqualificationは別に評価する。

第二Applicationに適切な構造類似が自然に存在しなければ、無理に業務意味を曲げません。その場合はB-01未成立をEvidenceとして残し、別Referenceが必要か判断します。

### B-02 lexical near-match / structurally different

二つのApplication間で名前・語彙が近い一方、Relation / Pattern / Capability条件が異なるassetが自然に存在する場合に実施します。

目的は、lexical重みを高くした方式が誤って上位固定しないかを見ることです。

Qualificationでは、scope / ownership / Contract / Capability等の不一致を具体的なreasonとして残します。

### B-03 cross-application partial cue

Application identityをqueryからmaskし、構造cueだけで両Applicationの関連候補が候補集合へ入るかを見ます。

これは「どちらか一つへhard routeする」評価ではありません。両方の候補を保持でき、その後Application / scope条件でqualificationできることを重視します。

## Benchmark caseの論理記録形

実装schemaではなく、Evidenceとして最低限残す項目を示します。

```text
BenchmarkCase
  id
  baseRevision
  candidateFamily
  candidatePoolRevision

  query
    sourceKind
    featureGroups
    perturbations[]

  retrievalOracle
    mustInclude[]
    shouldInclude[]
    emptyPositiveSet

  qualificationOracle
    expectedClassifications[]
    expectedReasons[]

  invarianceOracle
    sameGroups[]
    changedGroups[]

  observation
    algorithm
    algorithmConfigFingerprint
    seed
    rankedCandidates[]
    latency
    memory
    qualificationResults[]
```

scoreの尺度はalgorithm間で同じとは限らないため、異方式のraw scoreを直接比較しません。比較はRecall@k、candidate set size、false-familiar、latency等の共通指標へ落とします。

## Corpus規模と段階的判定

現行`order_entry`だけを使うPhase Aは**contract / invariance / responsibility boundaryの試験**です。この小規模corpusでFlyHashやHDC/VSAのProduct採用判断をしません。

採用判断を行う最小段階は次とします。

```text
Phase A
  order_entry anchor
  -> feature / perturbation / Evidence contractの検証

Phase B
  second accepted Application
  -> cross-application discrimination

Phase C (必要な場合のみ)
  additional realistic corpus
  -> scale / diversity / operational cost
```

Phase Bで方式差がほとんど出ない場合、複雑な認知方式を入れる理由は弱いと判断できます。exact / graph / weighted setで十分なら、それを肯定的な不採用結果として残します。

## 初回比較のablation

アルゴリズム差だけでなくfeature依存を確認します。

最低限、次を比較します。

```text
F1 only
F2 only
F3 only
F1 + F3
F2 + F3
F1 + F2 + F3
```

F0のID文字列を主要similarity featureとして加えたrunは、必要ならnegative controlとして扱います。F0ありでだけ成績が高い場合、構造認知ではなく名称・identity近接へ依存している可能性を疑います。

## seedと再現性

random projection等を使う方式ではseedをEvidenceへ残します。

一つの都合のよいseedだけで結論しません。少なくとも複数固定seedで結果分散を測定し、seed感度が大きい場合は運用上の不安定性として扱います。

一方、TEIのCanonical feature corpus自体はseedに依存してはいけません。

```text
Canonical World -> FeatureDocument
```

までは決定論的であることを要求します。

## 終了条件

このbenchmark contractのGate C前準備が満ちたとみなす条件は次です。

- 現行ReferenceからPhase Aのanchor identityを一意に指せる。
- A-01〜A-08についてquery perturbationと期待結果が説明できる。
- retrieval / qualification / invariance oracleが分離されている。
- pressure caseのために偽Canonical sourceを追加しなくてよい。
- 第二Applicationをbenchmarkと`ISSUE-000040`で二重管理しない。
- Phase AだけではProduct採用判断しないことが明示されている。
- algorithm固有parameterはbenchmark意味の正本へ入っていない。

Gate C後は、固定revisionに対してこのcontractをfixture / test dataへmaterializeし、[ISSUE-000051](issue/000051-derived-cognitive-retrieval-evaluation.md)のPhase 1 / 2を開始します。

## 非主張範囲

この文書だけから次は主張しません。

- FlyHash / DenseFly / HDC / VSAがTEIに有効であること
- LLMやembeddingが不要であること
- similarityからSemantic applicabilityを推論できること
- novelty scoreから`unknown`を決定できること
- `order_entry`だけで実業務規模のretrieval品質が分かること
- 第二Applicationの業務domainや構造がbenchmark都合で決まること
- Gate B / Gate Cの完了条件が増えたこと

## 関連文書

- [ISSUE-000051](issue/000051-derived-cognitive-retrieval-evaluation.md)
- [ISSUE-000047](issue/000047-knowledge-first-semantic-exploration.md)
- [ISSUE-000040](issue/000040-semantic-component-composition.md)
- [ISSUE-000039](issue/000039-business-change-rehearsal.md)
- [ISSUE-000027](issue/000027-ai-proposal-acceptance-boundary.md)
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/business-system-design.md`
