# ISSUE-000047: 既存Semantic資産を探索してから変更候補へ進む

- 状態: In Progress — Gate C Ready後のactual S1 qualificationを実施中
- 起点: `ISSUE-000008` Editor / AI follow-up
- 関連: `ISSUE-000027` AI Proposal / Acceptance Boundary
- Product Acceptance Scenario: **PAS-06 primary**
- 関連RQ: RQ-002 / RQ-003 / RQ-014 / RQ-015 / RQ-016 / RQ-017 / RQ-021 / RQ-023 / RQ-026 / RQ-027

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

### Data設計Knowledgeを生成AIへ渡す

Data / Relationの新規設計や変更では、AIへ単なるtable一覧やfield一覧だけを渡しません。業務上の識別と関数従属性を検討するために必要なcurrent contextを、Source / Evidenceへ戻れるread-only Projectionとして選択的に渡します。

初期のProjectionでは、対象Dataに応じて少なくとも次を候補にします。

```text
Data Design Context
  subject / scope
  business occurrence identifier
  candidate key
  determinant -> dependent
  composite domain / allowed-combination constraint
  Relation topology / cardinality
  temporal / history condition
  Constraint
  Source / provenance / Evidence
  nature: accepted / candidate / unresolved
```

これは新しいData Modelの第二正本ではありません。現在のCanonical World、Source Index、accepted Decision、Reference pattern等から再構成できるderived contextとします。削除・再生成してもaccepted meaningは変わりません。

現在のTEIで関数従属性をfirst-classにauthorしていない範囲では、AIに「従属性なし」とは伝えません。Sourceやaccepted meaningから確定できない場合は`unresolved`とし、項目名、画面配置、一般的なCRUD schema、過去の類似例だけから関数従属性を確定しません。

複数項目の値の組合せを制限するcomposite domain / allowed-combination constraintも同じです。現行Canonicalでfirst-classに表現していないことを「制約なし」と読み替えず、関数従属性や一意性とは別の`unresolved`として保持します。単項の`positive` Validation等、すでにCanonicalで表現できるConstraintまで未解決扱いにはしません。

時間・履歴条件も、日時型Attributeや監査用columnが存在するだけでは確定しません。Data Design Contextでは`temporalHistoryConditions`を独立した`unresolved`として返し、business occurrenceや識別子がいつ有効か、識別子を時間をまたいで再利用できるか、どの状態遷移がidentity / membership / validityを変えるか、どの履歴を保持するか、どのSource / Contractがその意味を定めるかを確認対象にします。

同様に、AIが全tableへUUIDや単独`id`を付けた案を生成しても、そのkeyを業務上の識別条件とみなしません。surrogate keyを実装候補として残す場合でも、元のbusiness identifier、candidate key、Relation topologyとの対応を保持します。

Reference pressureでは少なくとも次を比較します。

- 親occurrenceの内部で繰り返される明細について、親の識別子とlocalな行番等による業務上の識別をcontextから追えるcase
- 独立したrecordが別recordを変更可能に参照するcase
- 独立した二つの識別対象の組合せ自体に意味があるcase
- Source不足のため、関数従属性やcandidate keyを`unresolved`のまま保持するnegative case
- allowed-combinationの根拠がないとき、Relation shapeやunique constraintからcomposite domain constraintを補完しないnegative case
- datetime / timestamp属性やPersistence上の履歴columnがあるだけで、business temporal / history semanticsを確定しないnegative case
- physical surrogate keyを追加しても、acceptedなbusiness identifier / dependencyが消えないcase
- View / Application Pattern / ORM都合の単独`id`から、business identifierを逆算しないnegative case
- physical keyへ正規化されたbusiness keyを置換する場合でも、元candidate keyのuniquenessとMappingを追跡できるcase

このProjectionをprompt専用の文章へ固定しません。CLI、Studio、AIはいずれも同じmachine-readable contextへ戻れることを優先します。actual implementationは、現在のknowledge-context実装laneである`work/issue-000040-second-application-reference`へhandoffし、別branchを増やしません。

### 2026-10-06 Data Design Context — scoped uniqueness / Source Evidence increment

現在のCanonical modelでは、business occurrence identifier、candidate key、functional dependencyをfirst-classにauthorしていない。この状態で単独`id`やRelation keyを業務上の識別条件へ読み替えずにData設計を進めるには、確定している意味と未確定の意味を同じContext上で区別できる必要がある。

そこで`knowledge-context`のData Design Contextを拡張し、`relation-unique` Validationがすでに確定している範囲を`scopedUniqueness`として構造化する。たとえば`sales.order.lines.line_number_unique`から確定できるのは、`sales.order.lines`の各source occurrenceについて、target側の`sales.order_line.line_number`が一意であることまでである。Projectionでは`scope = per-relation-source-occurrence`、`nature = accepted`として表す。

この一意性はbusiness identifierやcandidate keyではない。親子関係が業務上の包含を意味するか、親の識別子とline numberの組合せがbusiness occurrenceを識別するかは、現在のCanonical meaningだけでは確定できない。このため`businessOccurrenceIdentifier`、`candidateKeys`、`functionalDependencies`は引き続き`unresolved`とする。

ただし、設計探索を止めないため、acceptedなscoped uniquenessから検討対象となる`businessIdentityCandidates`は提示する。`sales.order.lines.line_number_unique`では、候補のcomponentsを`relation-source-occurrence: sales.order`と`attribute: sales.order_line.line_number`の2項に分け、target occurrenceを`sales.order_line`とする。これはbusiness identifierやcandidate keyの確定ではない。source occurrenceのbusiness identifier、Relationのbusiness meaning、target membershipの排他性、identifier stability、temporal validityを確認できるまでcandidateのまま保持する。

`businessIdentityCandidates`と`functionalDependencyCandidates`には、候補そのものを指す`fingerprint`と、現在必要な確認項目を指す`evidenceRequirementFingerprint`を分けて持たせる。`fingerprint`はdeterminant / component、scope、target occurrence、basis constraint等の候補意味から決め、`evidenceNeeded`は含めない。そのため、確認項目が増減しても候補意味が同じならcandidate identityは変わらない。`evidenceRequirementFingerprint`はcandidate fingerprintと正規化した`evidenceNeeded`から生成するため、確認契約が変わればこちらだけが変わる。business identity候補とFD候補はtype discriminatorを分け、異なる候補種別を同じidentityとして扱わない。どちらのfingerprintもAcceptance、business identifierとしての確定、Evidence充足を証明しない。

Data Design Contextには、判断材料として使ったData、Attribute、Relation、Relation keyの両端、Validation、関連Dataについて、Source Indexから得たsubject、source path、source fingerprint、range、roleを`evidence`として含める。authoritativeなDefinition / Authoring referenceを取得できないsubjectは黙って省略せず、`evidenceMissing`へ残す。

Context再利用では、Evidenceの揃い方と構造解決の一意性を分ける。`evidenceCoverageStatus=complete|incomplete`は必要subjectのSource Evidenceが揃っているかだけを表し、`resolutionStatus=resolved|ambiguous`はCanonical ID / Persistence Mapping等の構造解決が一意かだけを表す。業務意味がすべて確定したことを`resolved`の意味にはしない。cached Contextのexact fingerprint guardは、`resolutionStatus=resolved`かつEvidence completeの場合だけ通す。Evidenceが揃っていてもduplicate Semantic IDやduplicate Persistence Mappingが残るContextは再利用しない。

構造解決状態を追加した段階ではData Design Projection revisionを`tei.knowledge.data-design/v2`へ上げ、その後、Data全体の`temporalHistoryConditions`とEvidence questionを追加してv3、候補単位のfingerprint追加でv4、候補identityとEvidence確認契約の分離でv5とした。SQL Realizationを明示result mappingからData Design Contextへ接続する現在のrevisionは`tei.knowledge.data-design/v6`とする。Context fingerprintにもprojection revisionが含まれるため、v1〜v5 cacheをv6へ黙って持ち越さない。

Data Design Context全体には`projectionRevision`と`fingerprint`も付ける。fingerprintは、正規化・並べ替え後のData Design Context全体から`fingerprint`自身だけを空にして決定論的に生成する。したがって、関連Source Evidenceだけでなく、ambiguity、候補、Persistence Realization、Evidence coverageなどProjectionの意味が変われば値も変わる。無関係なSourceはProjectionへ入らないため、その追加だけではfingerprintを変えない。この値はData設計の正しさやCanonical meaningの同一性を証明するものではなく、AIやStudioがContextを再取得するためのcache identityとして扱う。

Source Evidenceの揃い方は`evidenceCoverageStatus = complete | incomplete`で別に示す。`complete`が意味するのは、Data Design Contextが必要と判断したsubjectについてDefinition / Authoring EvidenceをSource Indexから取得できたことまでであり、Source Index自体が現在のworkspaceと一致することの証明ではない。Source freshnessはSource Index側のfingerprint検証責務として分離する。`incomplete`なContextにもfingerprintは付くが、完全なEvidenceを持つContextとして再利用しない。

CLIでcached Contextを再利用する場合は、保存していたfingerprintを`tei knowledge-context -expected-data-design-fingerprint <sha256:...>`へ渡せる。再利用が成立するには、current Projectionのfingerprint一致に加えて`resolutionStatus=resolved`と`evidenceCoverageStatus=complete`の両方が必要である。fingerprintが一致しない場合、構造解決がambiguousな場合、Evidence coverageが不完全な場合、対象subjectがData Design Contextを持たない場合はfail closedにする。この比較はstale derived contextの検出に使うものであり、fingerprint一致やEvidence coverage完了をSource freshnessやSemantic適用可能性の証明へ昇格しない。

`scopedUniqueness`はkind名だけから生成しない。Relationが`many`であり、ValidationのAttributeがそのRelation targetに属する場合に限ってProjectionへ出す。壊れた`relation-unique`やSource Evidence不足から、business identityやfunctional dependencyを推測しない。

一方、acceptedなscoped uniquenessからは、次に検証すべき関数従属性の候補を限定的に導ける。`sales.order.lines.line_number_unique`の場合、`line_number`単独をdeterminantとして扱わない。候補のdeterminantは、`relation-source-occurrence: sales.order`と`attribute: sales.order_line.line_number`の2項として保持し、dependent occurrenceを`sales.order_line`、scopeを`per-relation-source-occurrence`とする。これにより、scope注記を読み落としたAIが「line numberだけで明細を決める」と誤解する経路を構造上閉じる。natureは`candidate-from-accepted-scoped-uniqueness`であり、acceptedなfunctional dependencyではない。確定には少なくともsource occurrenceのbusiness identifier、Relationのbusiness meaning、target membershipの排他性、temporal validityが必要であるため、これらを`evidenceNeeded`へ残す。さらに、business identity候補には`history-version-identity`と`identifier-reuse-policy`、functional dependency候補には`history-version-identity`と`temporal-dependency-scope`を候補固有の未解決Evidenceとして保持する。上位の時間・履歴質問を読まなくても、候補単位で「同一occurrenceのversionか」「識別子再利用があるか」「従属性がどの時間scopeで成立するか」を再確認できるようにする。`functionalDependencies`本体は引き続き`unresolved`とする。

Reference testでは、`sales.order_line`についてincoming Relation、`relation-unique`、positive Validation、exact Source Evidenceを同じData Design Contextから取得できることを記述した。また、Relation keyを持つ`sales.order.customer`でも、source / target AttributeのEvidenceを保持しつつbusiness identifierへ昇格しないことを確認対象にした。Relationごとにcardinalityとkey mappingはacceptedな構造情報として示す一方、`businessMeaning`と`temporalSemantics`は`unresolved`で返す。Relationの形だけから変更可能な参照、包含、独立association、current-only、temporal等を決めない。内部negative testでは、不正なRelation shapeから`scopedUniqueness`を生成しないことと、Source Indexが欠けた場合に`evidenceMissing`へ明示することを固定した。

Constraint自体も探索入口にする。`semantic-validation`からは、ValidationのAttribute ownerとRelation targetが一致して一意にDataを決められる場合だけData Design Contextへ接続する。positive ValidationはAttribute ownerへ、`relation-unique`はAttribute ownerとRelation targetの一致を確認したうえでtarget Dataへ降りる。両者が矛盾する場合はDataを推測せず、Data Design Contextを生成しない。

Capabilityも同じread-only探索契約で扱う。`semantic-capability:sales.order.submit`からはCapability定義のSource traceと、input / outputとして参照する`sales.order`へのdependencyを取得できる。Capability IDやruntime invocation identityからData Design identityを作らず、Data Design Contextは生成しない。これによりData / Relation / Validation / Capability / Application Patternを同じContext形式で探索しつつ、各subjectのauthority境界を保つ。

同じCanonical IDを持つData、Attribute、Relation、Validationが複数ある場合も推測しない。Dataを直接参照したContextでは、重複している要素を`ambiguities`へ種類、ID、件数とともに示し、その要素を根拠に`scopedUniqueness`、business identity候補、functional dependency候補を生成しない。Validationを入口にしたContextでは、Attribute ownerやRelation target、対象Data、Validation自身を一意に決められないためData Design Context自体を生成しない。無効なWorkspaceで読み込み順に依存した設計判断を行わないことを、この境界で固定する。

Data Design Contextを生成しない場合も理由を隠さない。Validationのscopeを一意に解決できない場合は`data-design-validation-scope-not-uniquely-resolvable`、Persistence Mappingのsubjectを一意に解決できない場合は`data-design-persistence-subject-not-uniquely-resolvable`を上位Contextの`unresolved`へ残す。利用側は「Data Design Contextがない」ことだけを見て推測を再開せず、元定義の確認へ戻れる。

Application PatternからData設計へ戻る経路もnegative Referenceとして固定した。`order_entry.order_record`のようにPatternが`data sales.order`を明示している場合は、その参照を使って同じ`knowledge-context`内にData Design Contextを提示する。これにより、Patternを設計しているAIがSemantic Data、Constraint、Persistence Realization、未解決のidentityを別呼び出しなしで確認できる。一方、Pattern IDや画面上のrecord identityからbusiness identityを作らず、`businessOccurrenceIdentifier` / `candidateKeys` / `functionalDependencies`のauthorityはSemantic側に残す。`application-pattern -> semantic-data` dependencyも併存させ、参照経路を追跡できるようにする。PatternがDataを明示していない場合は、RelationやQuery等から都合よくDataを推測しない。

first-web-targetにも同じnegative boundaryを追加した。Web TargetはPatternごとに`PresentationCandidate`、Data、Relation、Capability等を投影するが、`item` / `collection` / `relation-selection`等のTargetローカルidentityをbusiness data identityへ読み替えない。`target-or-presentation-identity-is-not-business-data-identity`をTarget Projectionのauthority limitへ加え、record / browseのpresentation candidateが`sales.order`と別物であることをReference testで固定した。ORM / runtime objectについては、current Referenceに該当する独立したauthoritative identity layerが存在するかを確認したうえで扱い、抽象的なORM層を要件充足のためだけに追加しない。

current Repositoryには独立したORM identity layerはない。実在するruntime境界として、Driverの`queryresult.MappedResult`とfirst-web-targetのread scenarioをpressureした。MappedResultはraw columnを明示result mappingでSemantic attributeへ対応付けるが、それはSemantic correspondenceであってbusiness keyの証明ではない。read scenarioのrowは`Attributes`だけを持ち、row indexは選択状態、`order_id` / `order_line_id`等の値はruntimeで観測したAttribute値として扱う。Authority limitへ`runtime-row-index-is-selection-state-not-business-identity`、`runtime-row-or-attribute-value-is-not-business-identity-proof`、`explicit-result-mapping-is-semantic-correspondence-not-business-key-proof`を追加し、runtime convenience identityをbusiness occurrence identityへ昇格しないことをReference testで固定した。

時間・履歴条件についても、Data全体の設計判断とRelation固有の意味を分ける。Relationごとの`temporalSemantics`は、そのRelationがcurrent-onlyか、期間や履歴を持つかを示すための状態である。一方、`temporalHistoryConditions`はbusiness occurrenceや識別子の有効期間・再利用・状態遷移・履歴保持条件を扱う。現行Canonicalではこれらを十分にfirst-class authoringしていないため、Source / Contractが確定するまでは`unresolved`にする。

日時型Attribute、`created_at` / `updated_at`のような名称、Persistence上のhistory / audit columnを、それだけでbusiness temporal semanticsへ読み替えない。生成AIには有効期間、identifier reuse、identity / membershipを変える状態遷移、必要な履歴、根拠Source / ContractをEvidence questionとして渡す。時間意味をfirst-classなCore primitiveへ追加するかは、Referenceでlossが再現された時点で判断する。

関数従属性とは別に、複数項目の値の組合せ自体が許されるかというcomposite domain constraintも独立して扱う。現行Canonical modelには、allowed-combinationや複合的なdomain membershipをfirst-classにauthorするprimitiveがない。このためData Design Contextでは、該当する定義が見つからないことを「複合制約なし」とは扱わず、`compositeDomainConstraints.nature = unresolved`として返す。

生成AIには、許可・禁止される値の組合せ、成立するscopeと時間条件、根拠となるSource / Contract、staticなdomainかreference dataから導かれるdomainかをEvidence questionとして渡す。`relation-unique`、Relation key、Persistenceのprimary key / unique constraintは、それだけではcomposite domain constraintのEvidenceにしない。これにより、関数従属性、識別条件、局所的な一意性、allowed-combinationを同じ概念へまとめずに探索できる。

この変更も新しいCore primitiveの追加ではない。現行primitiveとSource / Evidenceから確定できる範囲を返し、表現不足が現実的なReferenceで再現された時点で、Semantic ExtensionまたはCore表現の追加が必要かを判断する。

Persistence RealizationもData Design Contextへ接続した。`sales.order_line`は`public.order_lines`へ実現され、Persistence Mappingでは`sales.order_line.order_line_id`が単独primary keyである。この事実は`database-owned-realization`として保持するが、business occurrence identifierやcandidate keyへは昇格しない。Database-ownedなunique constraintも同じく`accepted-realization`として保持し、業務上の一意性とは区別する。MappingのSource EvidenceもSource Indexから同じContextへ載せ、取得できない場合は`evidenceMissing`へ残す。これにより、単独primary keyやunique constraintが見えている場合でも、生成AIがそれをbusiness identityとみなす経路を明示的に閉じる。なお、現行Referenceの`order_line_id`がsurrogate keyであるとは断定しない。今回実証するのは、physical / persistence primary keyの存在だけではbusiness keyを確定できないという境界である。

逆方向も同じ境界で扱う。`knowledge-context`へ`persistence:sales_core.order_line`を渡した場合は、primary keyやcolumn名からDataを推測せず、Persistence Mappingが明示する`semantic sales.order_line`だけを使ってData Design Contextへ戻る。逆追跡後もbusiness occurrence identifier、candidate key、functional dependencyは`unresolved`のままである。これにより、SemanticからRealizationを見る経路とRealizationからSemanticへ戻る経路のどちらでも、physical keyをlogical identityへ昇格しない。

同じSemantic Dataに複数のPersistence Mappingがある場合も、各Databaseのprimary key / unique constraintを独立したRealization factとして保持する。たとえば一方が`order_line_id`をprimary keyにし、別のDatabaseが`line_number`をprimary keyにしていても、その差からlogical identityを再定義しない。`businessOccurrenceIdentifier`、`candidateKeys`、`functionalDependencies`のauthorityは変えず、Semantic側のscoped uniquenessから導いたbusiness identity候補もphysical keyで置換しない。このnegative pressureにより、Application↔Database=N:MでもData meaningのidentity境界を維持する。

ただし、異なるMapping IDを持つ複数Databaseの実現と、同じMapping IDの重複定義は分ける。同じMapping IDが複数ある場合はどちらもRealization factとして採用せず、`duplicate-persistence-mapping`を`ambiguities`へ出す。Persistence Mappingを入口にした逆追跡でも、重複IDを読み込み順で解決しない。

このincrementでも`BusinessKey`や`FunctionalDependency`を新しいCore primitiveとして追加しない。まずexisting primitiveからlosslessに説明できる範囲と、なお不足するEvidenceをReferenceで分離する。exact-headのRepository Healthは現在の実行環境では取得できないため、Product Evidence成立や完了条件充足はまだ主張しない。

### 2026-10-09 version / history identity negative pressure

時間・履歴条件については、日時型Attributeがあるだけでは意味を確定しないnegative testを既に持っている。ただし完了条件では、適用開始日、履歴番号、version番号、最新フラグ、履歴行の物理キーが同時に存在する場合でも、それらをbusiness occurrence identifierへ自動昇格しないことを求めている。

この境界を直接固定するため、`sales.order_line`へ`order_line_id`、`version_no`、`valid_from`、`valid_to`、`is_latest`を追加し、Persistence Mappingでは`order_line_id + version_no`を履歴行の複合primary keyとするpressureを追加した。

このcaseでも、Data Design Contextの`businessOccurrenceIdentifier`と`candidateKeys`は`unresolved`のままとし、`temporalHistoryConditions`も`temporal-history-condition-not-first-class-in-current-canonical-model`として未確定を維持する。物理primary keyは`database-owned-realization`として観測できるが、同一business occurrenceのversionを表すのか、別occurrenceを表すのかは決めない。

既存の`sales.order.lines.line_number_unique`から得られるbusiness identity候補も、履歴用columnやversion番号では置換しない。候補componentへ`version_no`、`valid_from`、`valid_to`、`is_latest`を自動追加しないことをtestで固定する。

このpressureが防ぐのは、既存DBやORMを読み取った生成AIが「version番号を含むprimary keyだから、これが業務上の識別子である」「最新フラグがあるから履歴意味は確定している」と短絡することだ。現在のSource / Contractから確認できない限り、同一occurrenceのversionか別occurrenceか、validityの重複可否、最新状態の定義、識別子再利用可否はEvidence questionとして残す。

この変更は新しいTemporal modelやHistory primitiveを導入しない。現行Canonicalで表現できていない意味を、physical schemaや命名から補完しないためのnegative Referenceである。exact-headのfull Repository-healthは未取得なので、該当完了条件を完了扱いにはしない。

### 2026-10-09 SQL Realization identity negative pressure

Target / ORM / runtime objectのidentityからbusiness keyを逆算しない完了条件について、現行Repositoryにはこれらを共通に表すfirst-class subjectはまだない。そこで新しい抽象schemaを先に追加せず、Source Indexへ実在するruntime寄りのsubjectである`contract-query-sql-realization`を使って境界をpressureする。

Contract Query SQL Realizationは、Database / Contract / Query / providerから成るRealization identityを持ち、raw result列をSemantic Attributeへ明示的に対応付ける。Data Design ContextはRealization ID、provider名、raw列名からDataを推測せず、result column mappingがすべて一つのSemantic Dataへ一意に戻れる場合だけ、そのDataへ接続する。

Referenceの`sales_core/sales_core.order_line/find_by_order/sql/postgres`では、result mappingが`sales.order_line`のAttributeだけを指すため、同じData Design Contextへ逆追跡できる。このときもRealization identityやraw result列をbusiness occurrence identifier、candidate key、functional dependencyへ昇格しない。Realization authoring sourceもEvidenceとして同じContextから辿れるようにする。

反対に、一つのSQL Realizationのresult mappingが複数Semantic Dataへまたがるsynthetic caseでは、どれか一つのDataを選ばない。`data-design-contract-query-sql-realization-scope-not-uniquely-resolvable`を`unresolved`へ残し、Data Design Contextを生成しない。

authority limitには、SQL Realization identity、raw result column、provider固有Realizationがbusiness data identityやSemantic authorityではないことを明示する。これにより、物理SQLやruntime registry上のidentityが業務識別へ逆流する経路を閉じる。

このpressureはTarget / ORM / runtime object全般を網羅したものではない。現行モデルで実在するSQL Realizationに対するReferenceであり、TargetやORMのfirst-class subjectが定義された場合は、そのidentity境界を別途pressureする。したがって該当完了条件はまだ完了扱いにしない。

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

#### 2026-10-07 selective disclosure Projection candidate

`tei knowledge-context`のfull Contextは後方互換のため維持し、`-disclose`を指定した場合だけread-onlyなDisclosure Projectionへ切り替えるcandidate実装を追加した。初期sectionは`source`、`definition`、`dependencies`、`data-design`である。

Projectionは`availableSections`、`selectedSections`、`disclosedSections`、`unavailableSections`、`omittedAvailableSections`を別々に持つ。選択したsectionがそのsubjectに存在しない場合は黙って空として扱わず、`status=partial`と`unavailableSections`へ残す。利用可能だが現在のtaskで選択しなかったsectionは`omittedAvailableSections`へ残す。

`subject`、`unresolved`、authority limitは選択対象から外せない安全メタデータとして常に開示する。特に、`omitted-knowledge-is-not-negative-evidence`と`unavailable-requested-section-is-not-empty-meaning`を保持し、開示しなかったことや利用不能だったことを「意味が存在しない」根拠へ使わない。

この`disclose`はsecurity boundary、RBAC、秘密情報のredactionではない。`availableSections`や`omittedAvailableSections`は未開示Knowledgeの存在自体を示し得るため、権限のないKnowledgeを隠す用途には使わない。access controlが必要な場合は、Contextを構築する前段でauthorized source / subjectだけを供給する別責務とする。

Disclosure Projectionには`tei.knowledge.disclosure/v1`と決定論的fingerprintを付ける。section指定はtrim / dedup / sortしてから扱うため、同じ選択集合なら順序が違っても同じProjection identityになる。このfingerprintは「どのKnowledge subsetを実際に開示したか」のidentityであり、Semantic authorityやqualificationの証明ではない。

cached Disclosureを再利用する場合は、`-disclose`と`-expected-disclosure-fingerprint <sha256:...>`を組み合わせる。current Disclosureが`status=complete`で、fingerprintも一致する場合だけ再利用を許す。requested sectionが現在は利用不能で`status=partial`になった場合は、fingerprintが一致していてもfail closedにする。`-expected-disclosure-fingerprint`だけを指定してfull Contextへ暗黙適用することも許さない。

```bash
tei knowledge-context \
  -workspace DIR \
  -subject-kind semantic-data \
  -subject-id sales.order_line \
  -disclose source,data-design

# 後続sessionでexact subsetを再利用する場合
tei knowledge-context \
  -workspace DIR \
  -subject-kind semantic-data \
  -subject-id sales.order_line \
  -disclose source,data-design \
  -expected-disclosure-fingerprint sha256:...
```

この経路はavailable Contextを削除せず、Canonical Worldやqualificationも変更しない。同じsubject / requirementsでDisclosure生成の前後にqualificationを実行し、status、Evidence、dependency Evidenceを含む結果が同一であることをtestへ固定する。selective disclosureはrouting / prompt budget / task-local contextの境界であり、accepted meaningの第二正本にはしない。

Disclosureは元Contextへのview pointerではなく、選択したpointer payloadを独立snapshotとして複製する。返されたData Design候補やApplication PatternのContract Query inputsを呼び出し側が変更しても、元Contextを変更できないことをnegative testへ含める。これによりread-onlyを利用側の慣習ではなく所有境界として扱う。

exact-head full Repository Healthはまだ取得していないため、この段階では完了条件の充足を主張しない。


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


## Gate C後のsecond Application起動

第二Application Referenceの最初の探索は、Gate C Readyを明示Evidenceで確認した後、

- `plan/GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md`

のS1として実行する。

本Issueはcandidate discovery / qualificationを担当し、Application source authoringやEnvironment Bindingを所有しない。

qualification結果は同じISSUE-000040 workstreamへhandoffする。Repositoryの現在のsolo-development flowに従い、remote exploration branchを新設せず、進行状態はIssueとEvidenceで管理する。

### actual S1 status — 2026-09-19

Gate C Ready後のactual second-Application S1を開始した。

- `plan/SECOND_APPLICATION_GATE_C_ENTRY_2026-09.md`
- `plan/SECOND_APPLICATION_S1_S2_QUALIFICATION_PORTABILITY_2026-09.md`
- `tools/reference-business-procedure-check/second_application_s1_s2_actual_pressure_test.go`

current RepositoryからCustomer lookup intentに対し、

- `customer.customer`
- `customer.customer_id`
- `master_data.customer ^2`
- `master_data.customer/find_by_id`

をapplicable candidateとしてSource / fingerprint付きで再導出する。

`order_entry.customer_reference`はshared authorityではなくprovenance candidateとして扱う。

actual S1 sourceはApplication source authoringやEnvironment Bindingを行わない。S1 qualification結果は同じISSUE-000040 primary workstream内でPhase 5 portabilityへhandoffする。

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

## 2026-10-02 Product index increment — Pattern / Validationを機械到達可能にする

Gate C後のknowledge-first baselineを製品コードへ戻して点検した結果、`tei index` / `internal/index.Build`はSemantic Data / Relation / CapabilityやDatabase依存を返す一方、**Semantic ValidationとApplication Pattern自体を索引化していない**ことが分かった。

これはRQ-015 / RQ-027の観点では、AIや人がApplicationへ再入場するときに、業務ruleとPatternだけ別経路で再発見する必要があるGapである。新しいKnowledge Storeを追加せず、既存Canonical Worldから再生成するderived indexを拡張する。

ISSUE-000040 primary workstreamと、ここに列挙する追加変更は、既存branch `work/issue-000040-second-application-reference` に集約する。`research/selective-knowledge-routing-r6` はFirst Web Target R7Bのqualification lineageとして予約されているため、このknowledge workには使用しない。

- Semantic Validationを`semanticElements`へ追加
- Validation → Attribute / Relationを`semanticDependencies`へ追加
- Application Patternを`applicationPatterns`へ追加
- Application → Validation / Patternを`declaredDependencies`へ追加
- Pattern → Data / Relation / Query / Capability / Database Contract / Contract Queryを`declaredDependencies`へ追加
- deterministic orderを保持するtestを追加
- `tei knowledge-context -workspace ... -subject-kind ... -subject-id ...`を追加し、指定subjectのSource / derived detail / outgoing dependency / incoming dependencyだけをscoped projectionとして返す
- 未知subjectは成功へ丸めず、Source / derived detail不足を`unresolved`として保持する

このincrementは**derived machine-readable contextの欠落を埋めるだけ**であり、PatternやValidationの第二正本を作らない。`knowledge-context`も`index.Build`とSource Indexを束ねるread-only Projectionであり、削除してもCanonical Worldは変わらない。Source authorityは既存Source Indexに残る。

また、PatternのDatasource aliasはApplication-localな値としてindex entryに保持するが、alias文字列からDatabase authorityを推測しない。Contract Queryについても、Application Patternが明示したContract / Query identityだけをderived dependencyへ写す。

### 2026-10-02 deterministic qualification increment — 明示要求をcurrent contextへ照合する

`knowledge-context`で候補の局所Source / dependencyを機械取得できるようになったため、次のincrementでは検索順位やAI判断を介さず、**呼び出し側が明示した要求だけ**をcurrent contextへ照合する。

```text
candidate subject
  + expected kind
  + optional Application scope
  + required dependency identities
  ↓
current Source Index / derived dependency context
  ↓
applicable / incompatible / unknown
```

CLI候補は次とする。

```bash
tei knowledge-qualify \
  -workspace reference/workspace \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference \
  -qualification-requirements requirements.json
```

requirementsは少なくとも次を持つ。

```json
{
  "expectedKind": "application-pattern",
  "application": "order_entry",
  "requiredDependencies": [
    {
      "direction": "outgoing",
      "relation": "contract-query",
      "kind": "database-contract-query",
      "id": "master_data.customer/find_by_id"
    }
  ]
}
```

判定境界:

- current Source IndexにDefinition / Authoring referenceが無いcandidateは`applicable`へ昇格させず`unknown`とする。その状態ではderived index上のdependency欠落やApplication scopeも確定的不適合へ昇格させない。
- expected kind、Application-owned Patternのscope、current derived dependencyに既知の矛盾がある場合だけ`incompatible`とする。
- context自体にunresolvedがあり、required dependencyの不存在を確定できない場合は`unknown`を維持する。
- requirementsに書かれていない条件を「満たした」と推測しない。
- `applicable`は明示requirementsへの適合だけを意味し、Acceptance、write authority、Runtime authorityを付与しない。
- retrieval score / similarity / model confidenceは入力にも判定根拠にも使わない。

最初のReference pressureでは、`order_entry.customer_reference`を`order_entry` scope + `master_data.customer/find_by_id`依存で`applicable`とし、同じPatternを`repair_intake`所有のdirect shared authorityとして要求した場合はscope mismatchで`incompatible`へ落とす。存在しないsubjectは推測で近似せず`unknown`を返す。

### 2026-10-03 qualification → Shared Acceptance handoff increment

deterministic qualificationの結果をProposal側へ渡す際に、subjectのIDだけへ縮退させない。`applicable`な結果だけが、subject kind、Source Index identity、authoritative source path、source fingerprint、評価したrequirementsをexact Evidenceとして運ぶ。

requirementsがrequired dependencyを明示している場合は、そのdependencyについてもDefinition / Authoring sourceとfingerprintをscoped Evidenceとして保持する。requirementsに無いdependencyはhandoffへ含めず、必要なKnowledgeだけを渡す。

`reference-shared-acceptance attach-qualification`は、このEvidenceを既存candidateへ付与する。delta自体は変更せず、candidate fingerprintにも含めない。

Shared AcceptanceのPreviewでは添付済みqualificationを信用しない。current Source Indexへ戻ってsubject kind / source / fingerprintを照合し、同じrequirementsでqualificationを再実行する。required dependencyがある場合は、そのauthoritative source / fingerprintの集合もcurrent qualificationと比較する。dependencyのIDが同じままsource内容が変わった場合もstaleとして拒否する。

この鮮度判定はexplicit requirementsに必要なsourceへ限定する。無関係なaccepted changeを理由にqualification全体をstaleへせず、repository-wideなglobal revisionも要求しない。

また、accepted Decisionへ保存したEvidenceもDecision Indexの読込時にcurrent Source Indexへ再照合する。workspace全体のrevision、Decisionが書いたsource、受理理由として使ったEvidenceの現在性を別々に扱い、過去のAcceptanceを現在のapplicabilityとして読み替えない。

`incompatible`と`unknown`はEvidenceとしてcandidateへ付与できない。この接続もqualificationをAcceptanceへ昇格させず、candidateは従来どおりShared Acceptance Boundaryのbase revision、resolve、Validation、architecture policy、Semantic Diff、impact、acceptance policyを通る。

### 2026-10-05 qualification → explicit-reference Proposal candidate

qualificationからShared AcceptanceへEvidenceを渡す経路に続き、探索結果から変更deltaへ進む最小のReference producerを追加した。

`propose-qualified-reference-pattern`は、applicableな`application-pattern` qualification、既存のbase candidate、destination-ownedなreuse requestを受け取る。reuse requestが指定するのはPattern ID、所有Application、Relation、datasource、placementだけである。Contract / Query / input mappingはrequestから受け付けず、current Canonicalのqualified origin Patternから読み直す。

この分離により、`order_entry.customer_reference`をprovenance candidateとして使いながら、origin側の`sales.order.customer`やdatasource alias `master`を`repair_intake`へ持ち込まない。一方、shared authorityとして再利用する`master_data.customer/find_by_id`とinput mappingは、qualificationのrequired dependencyにexact identityで含まれている場合だけProposalへ反映する。

生成前には同じrequirementsでcurrent qualificationを再実行し、subject、Source Evidence、required dependency Evidenceの一致を確認する。過去にapplicableだったという履歴だけからProposalを作らない。生成したcandidateにもcurrent qualification Evidenceを添付するため、Shared Acceptance Preview側の再qualificationを迂回しない。

testには、既知のS4 candidateと同じsemantic deltaになるpositive caseに加え、required Contract Queryをqualificationしていないcase、stale Evidence、reuse requestからContract Queryを注入するcaseをnegative caseとして記述した。CLI自体がworkspaceを変更しないことも対象にしている。

この段階でも探索結果の自動採用やReference Patternの自動適用は行わない。destination-ownedな情報は明示入力のままであり、Proposal生成後もAcceptanceは別である。

### 現在のEvidence境界

このsource changeはまだexact-head full Repository-healthを通していない。したがって、現時点ではProduct Evidence成立や本Issueの完了条件充足を主張しない。

code-changing laneの既存ruleに従い、full `tools/repository-health-check.sh` PASSとclean pre/post stateが得られるまでmainへのintegration authorityは無い。

## 完了条件

- [x] Gate C Ready後に着手する
- [ ] Data / Relation設計時に、AI / Studio / CLIが同じData Design Contextからbusiness occurrence identifier、candidate key、既知・候補のfunctional dependency、composite domain constraint、時間・履歴条件、Relation topology、Source / Evidence、unresolvedへ到達できる
- [ ] 単独`id` / UUID / surrogate keyを持つRealizationでも、acceptedなbusiness identifier / candidate key / uniqueness / Relation meaningを失わず逆追跡できる
- [ ] View / Application Pattern / Target / ORM / runtime objectのidentityからbusiness keyを逆算しないnegative Referenceを持つ
- [ ] functional dependencyをfirst-classにauthorしていない対象で、根拠不足を「従属性なし」へ丸めずunresolvedとしてAIへ返せる
- [ ] composite domain / allowed-combination constraintをfirst-classにauthorしていない対象で、根拠不足を「複合制約なし」へ丸めずunresolvedとしてAIへ返せる
- [ ] temporal / history conditionをfirst-classにauthorしていない対象で、datetime属性や履歴columnを根拠に「時間意味が確定済み」と扱わずunresolvedとしてAIへ返せる
- [ ] 適用開始日、履歴番号、version番号、最新フラグ等をbusiness occurrence identifierへ自動昇格せず、同じoccurrenceのhistory / versionか別occurrenceかをSource / Contract不足時はunresolvedとして保持できる
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